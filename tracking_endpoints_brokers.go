package loadstrike

import (
	stdcontext "context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	sarama "github.com/IBM/sarama"
	amqp "github.com/rabbitmq/amqp091-go"
	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	kafkascram "github.com/segmentio/kafka-go/sasl/scram"
)

type kafkaEndpointAdapter struct {
	endpoint  *EndpointSpec
	writer    *kafka.Writer
	reader    *kafka.Reader
	brokers   []string
	dialer    *kafka.Dialer
	ready     chan struct{}
	readyOnce sync.Once
}

func newKafkaEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.Kafka == nil {
		return nil, fmt.Errorf("kafka options are required for endpoint %q", endpoint.Name)
	}
	if usesKafkaGSSAPI(endpoint.Kafka) {
		return newSaramaKafkaEndpointAdapter(endpoint)
	}

	dialer, err := newKafkaDialer(endpoint.Kafka)
	if err != nil {
		return nil, err
	}
	brokers := parseBrokerList(endpoint.Kafka.BootstrapServers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka bootstrap servers are required for endpoint %q", endpoint.Name)
	}

	adapter := &kafkaEndpointAdapter{endpoint: endpoint, brokers: brokers, dialer: dialer, ready: make(chan struct{})}
	if strings.EqualFold(endpoint.Mode, "Produce") {
		adapter.writer = kafka.NewWriter(kafka.WriterConfig{
			Brokers:      brokers,
			Topic:        endpoint.Kafka.Topic,
			Dialer:       dialer,
			Balancer:     &kafka.RoundRobin{},
			RequiredAcks: -1,
		})
		return adapter, nil
	}

	startOffset := kafka.LastOffset
	if endpoint.Kafka.StartFromEarliest {
		startOffset = kafka.FirstOffset
	}
	groupID := firstNonBlank(endpoint.Kafka.ConsumerGroupID, "loadstrike-go")
	adapter.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       endpoint.Kafka.Topic,
		GroupID:     groupID,
		Dialer:      dialer,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: startOffset,
		MaxWait:     trackingPollInterval(endpoint),
	})
	return adapter, nil
}

func usesKafkaGSSAPI(options *KafkaEndpointOptions) bool {
	return options != nil &&
		options.SASL != nil &&
		strings.EqualFold(strings.TrimSpace(options.SASL.Mechanism), "gssapi")
}

func (a *kafkaEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if a.writer == nil {
		return endpointProduceResult{ErrorMessage: "Kafka endpoint is not configured in Produce mode."}, nil
	}

	payload := newTrackingPayload(a.endpoint)
	trackingID, err := resolveOrInjectTrackingID(a.endpoint, &payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	body, err := payloadBodyBytes(payload)
	if err != nil {
		return endpointProduceResult{}, err
	}
	headers := make([]kafka.Header, 0, len(payload.Headers)+1)
	for key, value := range payload.Headers {
		headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
	}
	if payload.ContentType != "" {
		headers = append(headers, kafka.Header{Key: "content-type", Value: []byte(payload.ContentType)})
	}
	if err := a.ensureTopic(ctx); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	message := kafka.Message{
		Key:     []byte(trackingID),
		Value:   body,
		Headers: headers,
		Time:    time.Now().UTC(),
	}
	var writeErr error
	for attempt := 0; attempt < 10; attempt++ {
		if err := a.writer.WriteMessages(ctx, message); err == nil {
			writeErr = nil
			break
		} else {
			writeErr = err
			if !isRetriableKafkaProduceError(err) {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	if writeErr != nil {
		return endpointProduceResult{ErrorMessage: writeErr.Error()}, nil
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UTC().UnixMilli(),
		Payload:     payload,
	}, nil
}

func (a *kafkaEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if a.reader == nil {
		return fmt.Errorf("kafka endpoint is not configured in Consume mode")
	}
	go func() {
		time.Sleep(2 * time.Second)
		a.signalReady()
	}()

	for {
		message, err := a.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		headers := map[string]string{}
		for _, header := range message.Headers {
			headers[header.Key] = string(header.Value)
		}
		payload := trackingPayload{
			Headers:     headers,
			Body:        append([]byte(nil), message.Value...),
			ContentType: headers["content-type"],
		}
		selector, err := parseTrackingFieldSelector(a.endpoint.TrackingField)
		if err != nil {
			return err
		}
		trackingID := selector.Extract(payload)
		if trackingID == "" {
			continue
		}

		if err := onMessage(endpointConsumeEvent{
			TrackingID:  trackingID,
			TimestampMS: message.Time.UTC().UnixMilli(),
			Payload:     payload,
		}); err != nil {
			return err
		}
	}
}

func (a *kafkaEndpointAdapter) Close() error {
	if a.reader != nil {
		_ = a.reader.Close()
	}
	if a.writer != nil {
		return a.writer.Close()
	}
	return nil
}

func (a *kafkaEndpointAdapter) WaitReady(ctx stdcontext.Context) error {
	if a.ready == nil {
		return nil
	}
	select {
	case <-a.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *kafkaEndpointAdapter) signalReady() {
	a.readyOnce.Do(func() {
		if a.ready != nil {
			close(a.ready)
		}
	})
}

func newKafkaDialer(options *KafkaEndpointOptions) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{
		Timeout:   30 * time.Second,
		DualStack: true,
	}

	protocol := strings.ToLower(strings.TrimSpace(options.SecurityProtocol))
	switch protocol {
	case "", "plaintext":
	case "ssl":
		dialer.TLS = &tls.Config{}
	case "sasl_plaintext", "saslplaintext", "sasl_ssl", "saslssl":
		if strings.Contains(protocol, "ssl") {
			dialer.TLS = &tls.Config{}
		}
		if options.SASL == nil {
			return nil, fmt.Errorf("kafka SASL settings are required when security protocol uses SASL")
		}
		mechanism, err := newKafkaSASLMechanism(options.SASL)
		if err != nil {
			return nil, err
		}
		dialer.SASLMechanism = mechanism
	default:
		return nil, fmt.Errorf("unsupported kafka security protocol %q", options.SecurityProtocol)
	}

	return dialer, nil
}

func newSaramaKafkaConfig(options *KafkaEndpointOptions, consumeMode bool) (*sarama.Config, error) {
	if options == nil {
		return nil, fmt.Errorf("kafka options are required")
	}
	if options.SASL == nil || options.SASL.GSSAPI == nil {
		return nil, fmt.Errorf("kafka gssapi settings are required when using the gssapi mechanism")
	}

	config := sarama.NewConfig()
	config.Net.DialTimeout = 30 * time.Second
	config.Net.ReadTimeout = 30 * time.Second
	config.Net.WriteTimeout = 30 * time.Second
	config.Net.ResolveCanonicalBootstrapServers = true
	config.Net.SASL.Enable = true
	config.Net.SASL.Handshake = true
	config.Net.SASL.Version = sarama.SASLHandshakeV1
	config.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
	config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
	config.Net.SASL.GSSAPI.ServiceName = firstNonBlank(options.SASL.GSSAPI.ServiceName, "kafka")
	config.Net.SASL.GSSAPI.Username = strings.TrimSpace(options.SASL.GSSAPI.Username)
	config.Net.SASL.GSSAPI.Password = options.SASL.GSSAPI.Password
	config.Net.SASL.GSSAPI.Realm = strings.TrimSpace(options.SASL.GSSAPI.Realm)
	config.Net.SASL.GSSAPI.KerberosConfigPath = resolveKerberosConfigPath()

	switch protocol := strings.ToLower(strings.TrimSpace(options.SecurityProtocol)); protocol {
	case "", "plaintext":
	case "ssl":
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{}
	case "sasl_plaintext", "saslplaintext":
	case "sasl_ssl", "saslssl":
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{}
	default:
		return nil, fmt.Errorf("unsupported kafka security protocol %q", options.SecurityProtocol)
	}

	if consumeMode {
		config.Consumer.Return.Errors = true
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
		if options.StartFromEarliest {
			config.Consumer.Offsets.Initial = sarama.OffsetOldest
		}
		config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	} else {
		config.Producer.Return.Successes = true
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Producer.Retry.Max = 10
		config.Producer.Retry.Backoff = 500 * time.Millisecond
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func resolveKerberosConfigPath() string {
	if value := strings.TrimSpace(os.Getenv("KRB5_CONFIG")); value != "" {
		return value
	}
	candidates := []string{
		filepath.Join(string(filepath.Separator), "etc", "krb5.conf"),
		filepath.Join(os.Getenv("ProgramData"), "MIT", "Kerberos5", "krb5.ini"),
		filepath.Join(os.Getenv("WINDIR"), "krb5.ini"),
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func newKafkaSASLMechanism(options *KafkaSASLOptions) (sasl.Mechanism, error) {
	switch strings.ToLower(strings.TrimSpace(options.Mechanism)) {
	case "", "plain":
		return plain.Mechanism{
			Username: options.Username,
			Password: options.Password,
		}, nil
	case "scramsha256", "scram_sha_256", "scram-sha-256":
		return kafkascram.Mechanism(kafkascram.SHA256, options.Username, options.Password)
	case "scramsha512", "scram_sha_512", "scram-sha-512":
		return kafkascram.Mechanism(kafkascram.SHA512, options.Username, options.Password)
	case "oauthbearer", "oauth_bearer", "oauth-bearer":
		token := strings.TrimSpace(options.OAuthBearerTokenEndpointURL)
		extensions := map[string]string{}
		if options.OAuthBearer != nil && strings.TrimSpace(options.OAuthBearer.AccessToken) != "" {
			token = strings.TrimSpace(options.OAuthBearer.AccessToken)
			extensions = cloneStringMap(options.OAuthBearer.Extensions)
		}
		if token == "" {
			token = strings.TrimSpace(options.AdditionalSettings["access_token"])
		}
		if token == "" {
			return nil, fmt.Errorf("kafka oauth bearer requires an access token")
		}
		return kafkaOAuthBearerMechanism{token: token, extensions: extensions}, nil
	case "gssapi":
		if options.GSSAPI == nil {
			return nil, fmt.Errorf("kafka gssapi settings are required when using the gssapi mechanism")
		}
		return nil, fmt.Errorf("kafka gssapi is handled by the sarama-backed adapter and should not be resolved through kafka-go")
	default:
		return nil, fmt.Errorf("unsupported kafka sasl mechanism %q", options.Mechanism)
	}
}

type saramaKafkaEndpointAdapter struct {
	endpoint      *EndpointSpec
	brokers       []string
	config        *sarama.Config
	producer      sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup
	ready         chan struct{}
	readyOnce     sync.Once
}

func newSaramaKafkaEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint == nil || endpoint.Kafka == nil {
		return nil, fmt.Errorf("kafka options are required for endpoint %q", endpoint.Name)
	}
	brokers := parseBrokerList(endpoint.Kafka.BootstrapServers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka bootstrap servers are required for endpoint %q", endpoint.Name)
	}

	consumeMode := strings.EqualFold(endpoint.Mode, "Consume")
	config, err := newSaramaKafkaConfig(endpoint.Kafka, consumeMode)
	if err != nil {
		return nil, err
	}

	adapter := &saramaKafkaEndpointAdapter{
		endpoint:  endpoint,
		brokers:   brokers,
		config:    config,
		ready:     make(chan struct{}),
		readyOnce: sync.Once{},
	}

	if consumeMode {
		groupID := firstNonBlank(endpoint.Kafka.ConsumerGroupID, "loadstrike-go")
		consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
		if err != nil {
			return nil, err
		}
		adapter.consumerGroup = consumerGroup
		return adapter, nil
	}

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	adapter.producer = producer
	adapter.signalReady()
	return adapter, nil
}

func (a *saramaKafkaEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if a == nil || a.producer == nil {
		return endpointProduceResult{ErrorMessage: "Kafka endpoint is not configured in Produce mode."}, nil
	}

	payload := newTrackingPayload(a.endpoint)
	trackingID, err := resolveOrInjectTrackingID(a.endpoint, &payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}
	body, err := payloadBodyBytes(payload)
	if err != nil {
		return endpointProduceResult{}, err
	}

	headers := make([]sarama.RecordHeader, 0, len(payload.Headers)+1)
	for key, value := range payload.Headers {
		headers = append(headers, sarama.RecordHeader{Key: []byte(key), Value: []byte(value)})
	}
	if payload.ContentType != "" {
		headers = append(headers, sarama.RecordHeader{Key: []byte("content-type"), Value: []byte(payload.ContentType)})
	}

	message := &sarama.ProducerMessage{
		Topic:   a.endpoint.Kafka.Topic,
		Key:     sarama.StringEncoder(trackingID),
		Value:   sarama.ByteEncoder(body),
		Headers: headers,
	}
	if _, _, err := a.producer.SendMessage(message); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UTC().UnixMilli(),
		Payload:     payload,
	}, nil
}

func (a *saramaKafkaEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if a == nil || a.consumerGroup == nil {
		return fmt.Errorf("kafka endpoint is not configured in Consume mode")
	}

	handler := &saramaTrackingConsumerGroupHandler{
		endpoint:    a.endpoint,
		onMessage:   onMessage,
		signalReady: a.signalReady,
	}
	for {
		if err := a.consumerGroup.Consume(ctx, []string{a.endpoint.Kafka.Topic}, handler); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

func (a *saramaKafkaEndpointAdapter) Close() error {
	if a == nil {
		return nil
	}
	var firstErr error
	if a.consumerGroup != nil {
		if err := a.consumerGroup.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if a.producer != nil {
		if err := a.producer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (a *saramaKafkaEndpointAdapter) WaitReady(ctx stdcontext.Context) error {
	if a == nil || a.ready == nil {
		return nil
	}
	select {
	case <-a.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *saramaKafkaEndpointAdapter) signalReady() {
	if a == nil {
		return
	}
	a.readyOnce.Do(func() {
		if a.ready != nil {
			close(a.ready)
		}
	})
}

type saramaTrackingConsumerGroupHandler struct {
	endpoint    *EndpointSpec
	onMessage   func(endpointConsumeEvent) error
	signalReady func()
}

func (h *saramaTrackingConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	if h.signalReady != nil {
		h.signalReady()
	}
	return nil
}

func (h *saramaTrackingConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *saramaTrackingConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	if h == nil || h.endpoint == nil {
		return nil
	}
	selector, err := parseTrackingFieldSelector(h.endpoint.TrackingField)
	if err != nil {
		return err
	}
	for message := range claim.Messages() {
		headers := map[string]string{}
		for _, header := range message.Headers {
			headers[string(header.Key)] = string(header.Value)
		}
		payload := trackingPayload{
			Headers:     headers,
			Body:        append([]byte(nil), message.Value...),
			ContentType: headers["content-type"],
		}
		trackingID := selector.Extract(payload)
		if trackingID == "" {
			session.MarkMessage(message, "")
			continue
		}
		if err := h.onMessage(endpointConsumeEvent{
			TrackingID:  trackingID,
			TimestampMS: message.Timestamp.UTC().UnixMilli(),
			Payload:     payload,
		}); err != nil {
			return err
		}
		session.MarkMessage(message, "")
	}
	return nil
}

type kafkaOAuthBearerMechanism struct {
	token      string
	extensions map[string]string
}

func (m kafkaOAuthBearerMechanism) Name() string {
	return "OAUTHBEARER"
}

func (m kafkaOAuthBearerMechanism) Start(_ stdcontext.Context) (sasl.StateMachine, []byte, error) {
	builder := strings.Builder{}
	builder.WriteString("n,,\x01auth=Bearer ")
	builder.WriteString(m.token)
	builder.WriteByte('\x01')
	for key, value := range m.extensions {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			builder.WriteString(key)
			builder.WriteByte('=')
			builder.WriteString(value)
			builder.WriteByte('\x01')
		}
	}
	builder.WriteByte('\x01')
	return kafkaOAuthBearerState{}, []byte(builder.String()), nil
}

type kafkaOAuthBearerState struct{}

func (kafkaOAuthBearerState) Next(_ stdcontext.Context, _ []byte) (bool, []byte, error) {
	return true, nil, nil
}

func parseBrokerList(value string) []string {
	parts := strings.Split(value, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			brokers = append(brokers, trimmed)
		}
	}
	return brokers
}

func (a *kafkaEndpointAdapter) ensureTopic(ctx stdcontext.Context) error {
	if a == nil || a.endpoint == nil || a.endpoint.Kafka == nil || len(a.brokers) == 0 || a.dialer == nil {
		return nil
	}
	topic := strings.TrimSpace(a.endpoint.Kafka.Topic)
	if topic == "" {
		return nil
	}
	conn, err := a.dialer.DialContext(ctx, "tcp", a.brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	controllerAddress := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := a.dialer.DialContext(ctx, "tcp", controllerAddress)
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	if err := controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil && !strings.Contains(strings.ToLower(err.Error()), "topic with this name already exists") {
		return err
	}
	return nil
}

func isRetriableKafkaProduceError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "unknown topic") ||
		strings.Contains(message, "leader not available") ||
		strings.Contains(message, "context deadline exceeded")
}

type rabbitMQEndpointAdapter struct {
	endpoint *EndpointSpec
	conn     *amqp.Connection
	channel  *amqp.Channel
}

func newRabbitMQEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.RabbitMQ == nil {
		return nil, fmt.Errorf("rabbitmq options are required for endpoint %q", endpoint.Name)
	}

	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		connection, err := amqp.DialConfig(buildRabbitMQURL(endpoint.RabbitMQ), amqp.Config{})
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		channel, err := connection.Channel()
		if err != nil {
			lastErr = err
			_ = connection.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		if queue := strings.TrimSpace(endpoint.RabbitMQ.QueueName); queue != "" {
			if _, err := channel.QueueDeclare(queue, endpoint.RabbitMQ.Durable, false, false, false, nil); err != nil {
				lastErr = err
				_ = channel.Close()
				_ = connection.Close()
				time.Sleep(1 * time.Second)
				continue
			}
		}

		return &rabbitMQEndpointAdapter{
			endpoint: endpoint,
			conn:     connection,
			channel:  channel,
		}, nil
	}

	return nil, lastErr
}

func (a *rabbitMQEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if !strings.EqualFold(a.endpoint.Mode, "Produce") {
		return endpointProduceResult{ErrorMessage: "RabbitMQ endpoint is not configured in Produce mode."}, nil
	}

	payload := newTrackingPayload(a.endpoint)
	trackingID, err := resolveOrInjectTrackingID(a.endpoint, &payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	body, err := payloadBodyBytes(payload)
	if err != nil {
		return endpointProduceResult{}, err
	}
	headers := amqp.Table{}
	for key, value := range payload.Headers {
		headers[key] = value
	}

	publishing := amqp.Publishing{
		Headers:     headers,
		ContentType: payload.ContentType,
		Body:        body,
		Timestamp:   time.Now().UTC(),
	}
	routingKey := firstNonBlank(a.endpoint.RabbitMQ.RoutingKey, a.endpoint.RabbitMQ.QueueName)
	if err := a.channel.PublishWithContext(ctx, firstNonBlank(a.endpoint.RabbitMQ.Exchange, ""), routingKey, false, false, publishing); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UTC().UnixMilli(),
		Payload:     payload,
	}, nil
}

func (a *rabbitMQEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if !strings.EqualFold(a.endpoint.Mode, "Consume") {
		return fmt.Errorf("rabbitmq endpoint is not configured in Consume mode")
	}

	deliveries, err := a.channel.Consume(
		a.endpoint.RabbitMQ.QueueName,
		"",
		a.endpoint.RabbitMQ.AutoAck,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}

			headers := map[string]string{}
			for key, value := range delivery.Headers {
				headers[key] = asString(value)
			}
			payload := trackingPayload{
				Headers:     headers,
				Body:        append([]byte(nil), delivery.Body...),
				ContentType: delivery.ContentType,
			}
			selector, err := parseTrackingFieldSelector(a.endpoint.TrackingField)
			if err != nil {
				return err
			}
			trackingID := selector.Extract(payload)
			if trackingID == "" {
				if !a.endpoint.RabbitMQ.AutoAck {
					_ = delivery.Ack(false)
				}
				continue
			}

			if err := onMessage(endpointConsumeEvent{
				TrackingID:  trackingID,
				TimestampMS: time.Now().UTC().UnixMilli(),
				Payload:     payload,
			}); err != nil {
				return err
			}
			if !a.endpoint.RabbitMQ.AutoAck {
				_ = delivery.Ack(false)
			}
		}
	}
}

func (a *rabbitMQEndpointAdapter) Close() error {
	if a.channel != nil {
		_ = a.channel.Close()
	}
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func buildRabbitMQURL(options *RabbitMQEndpointOptions) string {
	scheme := "amqp"
	if options.UseSSL {
		scheme = "amqps"
	}
	host := strings.TrimSpace(options.HostName)
	if host == "" {
		host = "127.0.0.1"
	}
	port := options.Port
	if port <= 0 {
		port = 5672
	}
	user := url.QueryEscape(firstNonBlank(options.UserName, "guest"))
	password := url.QueryEscape(firstNonBlank(options.Password, "guest"))
	virtualHost := strings.TrimPrefix(firstNonBlank(options.VirtualHost, "/"), "/")
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s", scheme, user, password, host, port, virtualHost)
}

type azureEventHubsEndpointAdapter struct {
	endpoint *EndpointSpec
	producer *azeventhubs.ProducerClient
	consumer *azeventhubs.ConsumerClient
}

func newAzureEventHubsEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.AzureEventHubs == nil {
		return nil, fmt.Errorf("azure event hubs options are required for endpoint %q", endpoint.Name)
	}

	if strings.EqualFold(endpoint.Mode, "Produce") {
		producer, err := azeventhubs.NewProducerClientFromConnectionString(
			endpoint.AzureEventHubs.ConnectionString,
			endpoint.AzureEventHubs.EventHubName,
			nil,
		)
		if err != nil {
			return nil, err
		}
		return &azureEventHubsEndpointAdapter{
			endpoint: endpoint,
			producer: producer,
		}, nil
	}

	consumer, err := azeventhubs.NewConsumerClientFromConnectionString(
		endpoint.AzureEventHubs.ConnectionString,
		endpoint.AzureEventHubs.EventHubName,
		firstNonBlank(endpoint.AzureEventHubs.ConsumerGroup, azeventhubs.DefaultConsumerGroup),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &azureEventHubsEndpointAdapter{
		endpoint: endpoint,
		consumer: consumer,
	}, nil
}

func (a *azureEventHubsEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if a.producer == nil {
		return endpointProduceResult{ErrorMessage: "Azure Event Hubs endpoint is not configured in Produce mode."}, nil
	}

	payload := newTrackingPayload(a.endpoint)
	trackingID, err := resolveOrInjectTrackingID(a.endpoint, &payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	body, err := payloadBodyBytes(payload)
	if err != nil {
		return endpointProduceResult{}, err
	}
	event := &azeventhubs.EventData{
		Body:       body,
		Properties: map[string]any{},
	}
	if payload.ContentType != "" {
		event.ContentType = &payload.ContentType
	}
	for key, value := range payload.Headers {
		event.Properties[key] = value
	}

	var batch *azeventhubs.EventDataBatch
	if partitionID := resolveEventHubPartitionID(a.endpoint.AzureEventHubs); partitionID != "" {
		batch, err = a.producer.NewEventDataBatch(ctx, &azeventhubs.EventDataBatchOptions{PartitionID: &partitionID})
	} else {
		batch, err = a.producer.NewEventDataBatch(ctx, nil)
	}
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}
	if err := batch.AddEventData(event, nil); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}
	if err := a.producer.SendEventDataBatch(ctx, batch, nil); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UTC().UnixMilli(),
		Payload:     payload,
	}, nil
}

func (a *azureEventHubsEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if a.consumer == nil {
		return fmt.Errorf("azure event hubs endpoint is not configured in Consume mode")
	}

	partitionIDs, err := a.resolvePartitionIDs(ctx)
	if err != nil {
		return err
	}
	startPosition := azeventhubs.StartPosition{}
	if a.endpoint.AzureEventHubs.StartFromEarliest {
		startPosition.Earliest = boolPtr(true)
	} else {
		startPosition.Latest = boolPtr(true)
	}

	for _, partitionID := range partitionIDs {
		partitionClient, err := a.consumer.NewPartitionClient(partitionID, &azeventhubs.PartitionClientOptions{
			StartPosition: startPosition,
		})
		if err != nil {
			return err
		}

		go func(client *azeventhubs.PartitionClient) {
			defer client.Close(stdcontext.Background())
			for {
				events, receiveErr := client.ReceiveEvents(ctx, 25, nil)
				if receiveErr != nil {
					if ctx.Err() != nil {
						return
					}
					time.Sleep(trackingPollInterval(a.endpoint))
					continue
				}
				for _, event := range events {
					if event == nil {
						continue
					}
					headers := map[string]string{}
					for key, value := range event.Properties {
						headers[key] = asString(value)
					}
					if event.ContentType != nil {
						headers["content-type"] = *event.ContentType
					}
					payload := trackingPayload{
						Headers:     headers,
						Body:        append([]byte(nil), event.Body...),
						ContentType: derefString(event.ContentType),
					}
					selector, selectorErr := parseTrackingFieldSelector(a.endpoint.TrackingField)
					if selectorErr != nil {
						return
					}
					trackingID := selector.Extract(payload)
					if trackingID == "" {
						continue
					}
					timestamp := time.Now().UTC().UnixMilli()
					if event.EnqueuedTime != nil {
						timestamp = event.EnqueuedTime.UTC().UnixMilli()
					}
					if callbackErr := onMessage(endpointConsumeEvent{
						TrackingID:  trackingID,
						TimestampMS: timestamp,
						Payload:     payload,
					}); callbackErr != nil {
						return
					}
				}
			}
		}(partitionClient)
	}

	<-ctx.Done()
	return nil
}

func (a *azureEventHubsEndpointAdapter) Close() error {
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 10*time.Second)
	defer cancel()
	if a.consumer != nil {
		_ = a.consumer.Close(ctx)
	}
	if a.producer != nil {
		return a.producer.Close(ctx)
	}
	return nil
}

func (a *azureEventHubsEndpointAdapter) resolvePartitionIDs(ctx stdcontext.Context) ([]string, error) {
	if partitionID := strings.TrimSpace(a.endpoint.AzureEventHubs.PartitionID); partitionID != "" {
		return []string{partitionID}, nil
	}
	properties, err := a.consumer.GetEventHubProperties(ctx, nil)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), properties.PartitionIDs...), nil
}

func resolveEventHubPartitionID(options *AzureEventHubsEndpointOptions) string {
	if options == nil {
		return ""
	}
	if strings.TrimSpace(options.PartitionID) != "" {
		return strings.TrimSpace(options.PartitionID)
	}
	if strings.TrimSpace(options.PartitionKey) == "" || options.PartitionCount <= 0 {
		return ""
	}

	hash := 0
	for _, character := range options.PartitionKey {
		hash = ((hash << 5) - hash) + int(character)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("%d", hash%options.PartitionCount)
}

func boolPtr(value bool) *bool {
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
