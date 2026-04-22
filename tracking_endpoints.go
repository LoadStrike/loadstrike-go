package loadstrike

import (
	"bytes"
	stdcontext "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type endpointProduceResult struct {
	IsSuccess    bool
	TrackingID   string
	TimestampMS  int64
	Payload      trackingPayload
	ErrorMessage string
}

type endpointConsumeEvent struct {
	TrackingID  string
	TimestampMS int64
	Payload     trackingPayload
}

type endpointAdapter interface {
	Produce(stdcontext.Context) (endpointProduceResult, error)
	Consume(stdcontext.Context, func(endpointConsumeEvent) error) error
	Close() error
}

func newEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint definition is required")
	}

	switch strings.ToLower(strings.TrimSpace(endpoint.Kind)) {
	case "delegatestream":
		return newDelegateStreamEndpointAdapter(endpoint)
	case "pushdiffusion":
		return newPushDiffusionEndpointAdapter(endpoint)
	case "http":
		return newHTTPEndpointAdapter(endpoint)
	case "nats":
		return newNATSEndpointAdapter(endpoint)
	case "redisstreams":
		return newRedisStreamsEndpointAdapter(endpoint)
	case "kafka":
		return newKafkaEndpointAdapter(endpoint)
	case "rabbitmq":
		return newRabbitMQEndpointAdapter(endpoint)
	case "azureeventhubs":
		return newAzureEventHubsEndpointAdapter(endpoint)
	default:
		return nil, fmt.Errorf("endpoint kind %q is not supported", endpoint.Kind)
	}
}

func newTrackingPayload(endpoint *EndpointSpec) trackingPayload {
	payload := trackingPayload{
		Headers:            cloneStringMap(endpoint.MessageHeaders),
		ContentType:        strings.TrimSpace(endpoint.ContentType),
		MessagePayloadType: strings.TrimSpace(endpoint.MessagePayloadType),
	}

	if len(endpoint.MessagePayload) > 0 {
		var decoded any
		if json.Unmarshal(endpoint.MessagePayload, &decoded) == nil {
			payload.Body = decoded
		} else {
			payload.Body = []byte(endpoint.MessagePayload)
		}
		if payload.ContentType == "" {
			payload.ContentType = "application/json"
		}
	}

	return payload
}

func resolveOrInjectTrackingID(endpoint *EndpointSpec, payload *trackingPayload) (string, error) {
	if endpoint == nil {
		return "", fmt.Errorf("endpoint definition is required")
	}
	if payload == nil {
		return "", fmt.Errorf("payload is required")
	}

	selector, err := parseTrackingFieldSelector(endpoint.TrackingField)
	if err != nil {
		return "", err
	}

	if trackingID := selector.Extract(*payload); trackingID != "" {
		return trackingID, nil
	}

	trackingID := generateTrackingID()
	selector.Set(payload, trackingID)
	if selector.Extract(*payload) == "" {
		return "", fmt.Errorf(
			"tracking id could not be injected using selector %q for endpoint %q",
			endpoint.TrackingField,
			endpoint.Name,
		)
	}

	return trackingID, nil
}

func payloadBodyBytes(payload trackingPayload) ([]byte, error) {
	switch body := payload.Body.(type) {
	case nil:
		return nil, nil
	case []byte:
		return append([]byte(nil), body...), nil
	case string:
		return []byte(body), nil
	default:
		return json.Marshal(body)
	}
}

func payloadBodyText(payload trackingPayload) string {
	body, err := payloadBodyBytes(payload)
	if err != nil {
		return ""
	}
	return string(body)
}

func payloadFromResponse(response *http.Response, body []byte) trackingPayload {
	headers := map[string]string{}
	if response != nil {
		for key, values := range response.Header {
			headers[key] = strings.Join(values, ",")
		}
	}

	contentType := ""
	if response != nil {
		contentType = response.Header.Get("Content-Type")
	}

	var decoded any
	if len(body) > 0 && json.Unmarshal(body, &decoded) == nil {
		return trackingPayload{
			Headers:     headers,
			Body:        decoded,
			ContentType: contentType,
		}
	}

	return trackingPayload{
		Headers:     headers,
		Body:        append([]byte(nil), body...),
		ContentType: contentType,
	}
}

func payloadFromMap(record map[string]any) trackingPayload {
	payload := trackingPayload{
		Headers: map[string]string{},
	}

	if headers, ok := record["headers"].(map[string]any); ok {
		for key, value := range headers {
			payload.Headers[key] = asString(value)
		}
	}
	if headers, ok := record["headers"].(map[string]string); ok {
		for key, value := range headers {
			payload.Headers[key] = value
		}
	}

	if body, ok := record["body"]; ok {
		payload.Body = body
	}
	if contentType := strings.TrimSpace(asString(record["contentType"])); contentType != "" {
		payload.ContentType = contentType
	}
	if messagePayloadType := strings.TrimSpace(asString(record["messagePayloadType"])); messagePayloadType != "" {
		payload.MessagePayloadType = messagePayloadType
	}
	if producedUTC := strings.TrimSpace(asString(record["producedUtc"])); producedUTC != "" {
		payload.ProducedUTC = producedUTC
	}
	if consumedUTC := strings.TrimSpace(asString(record["consumedUtc"])); consumedUTC != "" {
		payload.ConsumedUTC = consumedUTC
	}

	return payload
}

func parseTimestampMS(value any, fallback int64) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed
		}
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return fallback
		}
		if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
			return parsed.UnixMilli()
		}
		var numeric int64
		if _, err := fmt.Sscan(trimmed, &numeric); err == nil {
			return numeric
		}
	}
	return fallback
}

func trackingPollInterval(endpoint *EndpointSpec) time.Duration {
	interval := endpoint.PollIntervalSeconds
	if interval <= 0 {
		interval = 0.25
	}
	return time.Duration(interval * float64(time.Second))
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

type delegateStreamEndpointAdapter struct {
	endpoint *EndpointSpec
	client   *http.Client
}

func newDelegateStreamEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.DelegateStream == nil {
		return nil, fmt.Errorf("delegate stream options are required for endpoint %q", endpoint.Name)
	}
	return &delegateStreamEndpointAdapter{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (a *delegateStreamEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if !strings.EqualFold(a.endpoint.Mode, "Produce") {
		return endpointProduceResult{ErrorMessage: "Delegate stream endpoint is not configured in Produce mode."}, nil
	}

	payload := newTrackingPayload(a.endpoint)
	trackingID, err := resolveOrInjectTrackingID(a.endpoint, &payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}
	if a.endpoint.DelegateStream.Produce != nil {
		result, callbackErr := a.endpoint.DelegateStream.Produce(ctx, newTrackingPayloadWrapper(payload))
		if callbackErr != nil {
			return endpointProduceResult{ErrorMessage: callbackErr.Error()}, nil
		}
		nativeResult := result.toNative()
		if strings.TrimSpace(nativeResult.TrackingID) == "" {
			nativeResult.TrackingID = trackingID
		}
		if nativeResult.TimestampMS == 0 {
			nativeResult.TimestampMS = time.Now().UnixMilli()
		}
		if nativeResult.Payload.Headers == nil && nativeResult.Payload.Body == nil {
			nativeResult.Payload = payload
		}
		return nativeResult, nil
	}
	callbackURL := strings.TrimSpace(a.endpoint.DelegateStream.ProduceCallbackURL)
	if callbackURL == "" {
		return endpointProduceResult{ErrorMessage: "Delegate stream produce callback URL is required."}, nil
	}

	requestBody := map[string]any{
		"endpoint":   a.endpoint,
		"payload":    payload,
		"trackingId": trackingID,
	}
	requestPayload, err := json.Marshal(requestBody)
	if err != nil {
		return endpointProduceResult{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(requestPayload))
	if err != nil {
		return endpointProduceResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := a.client.Do(request)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return endpointProduceResult{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return endpointProduceResult{
			ErrorMessage: fmt.Sprintf("delegate stream produce failed with status %d", response.StatusCode),
		}, nil
	}

	result := endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UnixMilli(),
		Payload:     payloadFromResponse(response, body),
	}

	if len(body) > 0 {
		var decoded map[string]any
		if json.Unmarshal(body, &decoded) == nil {
			if success, exists := decoded["isSuccess"]; exists && !asBool(success) {
				result.IsSuccess = false
				result.ErrorMessage = firstNonBlank(asString(decoded["errorMessage"]), "delegate stream producer reported a failure")
			}
			if timestamp := parseTimestampMS(decoded["timestampUtc"], 0); timestamp > 0 {
				result.TimestampMS = timestamp
			}
			if callbackTrackingID := strings.TrimSpace(asString(decoded["trackingId"])); callbackTrackingID != "" {
				result.TrackingID = callbackTrackingID
			}
			if payloadRecord, ok := decoded["payload"].(map[string]any); ok {
				result.Payload = payloadFromMap(payloadRecord)
			}
		}
	}

	return result, nil
}

func (a *delegateStreamEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if !strings.EqualFold(a.endpoint.Mode, "Consume") {
		return fmt.Errorf("delegate stream endpoint is not configured in Consume mode")
	}
	if a.endpoint.DelegateStream.Consume != nil {
		return a.endpoint.DelegateStream.Consume(ctx, func(event EndpointConsumeEvent) error {
			return onMessage(event.toNative())
		})
	}
	callbackURL := strings.TrimSpace(a.endpoint.DelegateStream.ConsumeCallbackURL)
	if callbackURL == "" {
		return fmt.Errorf("delegate stream consume callback URL is required")
	}

	ticker := time.NewTicker(trackingPollInterval(a.endpoint))
	defer ticker.Stop()

	for {
		if err := a.consumeOnce(ctx, callbackURL, onMessage); err != nil && ctx.Err() == nil {
			// Delegate callbacks are polled and intentionally resilient to transient errors.
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (a *delegateStreamEndpointAdapter) consumeOnce(ctx stdcontext.Context, callbackURL string, onMessage func(endpointConsumeEvent) error) error {
	requestBody, _ := json.Marshal(map[string]any{
		"endpoint": a.endpoint,
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("delegate stream consume failed with status %d", response.StatusCode)
	}
	if len(body) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return err
	}

	records := []map[string]any{}
	switch typed := decoded.(type) {
	case map[string]any:
		if messages, ok := typed["messages"].([]any); ok {
			for _, message := range messages {
				if record, ok := message.(map[string]any); ok {
					records = append(records, record)
				}
			}
		} else {
			records = append(records, typed)
		}
	case []any:
		for _, message := range typed {
			if record, ok := message.(map[string]any); ok {
				records = append(records, record)
			}
		}
	}

	for _, record := range records {
		payload := payloadFromResponse(response, body)
		if payloadRecord, ok := record["payload"].(map[string]any); ok {
			payload = payloadFromMap(payloadRecord)
		}
		trackingID := strings.TrimSpace(asString(record["trackingId"]))
		if trackingID == "" {
			selector, err := parseTrackingFieldSelector(a.endpoint.TrackingField)
			if err != nil {
				return err
			}
			trackingID = selector.Extract(payload)
		}
		if trackingID == "" {
			continue
		}

		event := endpointConsumeEvent{
			TrackingID:  trackingID,
			TimestampMS: parseTimestampMS(record["timestampUtc"], time.Now().UnixMilli()),
			Payload:     payload,
		}
		if err := onMessage(event); err != nil {
			return err
		}
	}

	return nil
}

func (a *delegateStreamEndpointAdapter) Close() error {
	return nil
}

type pushDiffusionEndpointAdapter struct {
	*delegateStreamEndpointAdapter
}

func newPushDiffusionEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	callbacks := &DelegateEndpointOptions{}
	if endpoint.PushDiffusion != nil {
		callbacks.ProduceCallbackURL = endpoint.PushDiffusion.PublishCallbackURL
		callbacks.ConsumeCallbackURL = endpoint.PushDiffusion.SubscribeCallbackURL
		callbacks.ConnectionMetadata = endpoint.PushDiffusion.ConnectionProperties
		callbacks.Produce = endpoint.PushDiffusion.Publish
		callbacks.Consume = endpoint.PushDiffusion.Subscribe
	}

	cloned := *endpoint
	cloned.DelegateStream = callbacks
	return &pushDiffusionEndpointAdapter{delegateStreamEndpointAdapter: &delegateStreamEndpointAdapter{
		endpoint: &cloned,
		client:   &http.Client{Timeout: 30 * time.Second},
	}}, nil
}

type httpEndpointAdapter struct {
	endpoint *EndpointSpec
	client   *http.Client
}

func newHTTPEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.HTTP == nil {
		return nil, fmt.Errorf("http options are required for endpoint %q", endpoint.Name)
	}

	timeout := endpoint.HTTP.RequestTimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}

	return &httpEndpointAdapter{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: time.Duration(timeout * float64(time.Second)),
		},
	}, nil
}

func (a *httpEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if !strings.EqualFold(a.endpoint.Mode, "Produce") {
		return endpointProduceResult{ErrorMessage: "HTTP endpoint is not configured in Produce mode."}, nil
	}

	payload := newTrackingPayload(a.endpoint)
	trackingID, err := resolveOrInjectTrackingID(a.endpoint, &payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	responsePayload, response, err := a.send(ctx, payload)
	if err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return endpointProduceResult{
			ErrorMessage: fmt.Sprintf("HTTP source call failed with status %d", response.StatusCode),
		}, nil
	}

	if strings.EqualFold(strings.TrimSpace(a.endpoint.HTTP.TrackingPayloadSource), "Response") {
		if selector, selectorErr := parseTrackingFieldSelector(a.endpoint.TrackingField); selectorErr == nil {
			if responseTrackingID := selector.Extract(responsePayload); responseTrackingID != "" {
				trackingID = responseTrackingID
			}
		}
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UnixMilli(),
		Payload:     responsePayload,
	}, nil
}

func (a *httpEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if !strings.EqualFold(a.endpoint.Mode, "Consume") {
		return fmt.Errorf("HTTP endpoint is not configured in Consume mode")
	}

	ticker := time.NewTicker(trackingPollInterval(a.endpoint))
	defer ticker.Stop()

	for {
		if err := a.consumeOnce(ctx, onMessage); err != nil && ctx.Err() == nil {
			// HTTP polling is resilient to transient errors.
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (a *httpEndpointAdapter) consumeOnce(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	payload := newTrackingPayload(a.endpoint)
	responsePayload, response, err := a.send(ctx, payload)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil
	}

	records := []trackingPayload{responsePayload}
	arrayPath := ""
	if a.endpoint.HTTP != nil {
		arrayPath = strings.TrimSpace(a.endpoint.HTTP.ConsumeArrayPath)
		if arrayPath == "" && a.endpoint.HTTP.ConsumeJSONArrayResponse {
			arrayPath = "$"
		}
	}

	if arrayPath != "" {
		items := extractPayloadArray(responsePayload.Body, arrayPath)
		records = make([]trackingPayload, 0, len(items))
		for _, item := range items {
			records = append(records, trackingPayload{
				Headers:     cloneStringMap(responsePayload.Headers),
				Body:        item,
				ContentType: responsePayload.ContentType,
			})
		}
	}

	for _, record := range records {
		selector, err := parseTrackingFieldSelector(a.endpoint.TrackingField)
		if err != nil {
			return err
		}
		trackingID := selector.Extract(record)
		if trackingID == "" {
			continue
		}
		if err := onMessage(endpointConsumeEvent{
			TrackingID:  trackingID,
			TimestampMS: time.Now().UnixMilli(),
			Payload:     record,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (a *httpEndpointAdapter) send(ctx stdcontext.Context, payload trackingPayload) (trackingPayload, *http.Response, error) {
	body, err := payloadBodyBytes(payload)
	if err != nil {
		return trackingPayload{}, nil, err
	}

	method := http.MethodGet
	if a.endpoint.HTTP != nil && strings.TrimSpace(a.endpoint.HTTP.Method) != "" {
		method = strings.ToUpper(strings.TrimSpace(a.endpoint.HTTP.Method))
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimSpace(a.endpoint.HTTP.URL), bytes.NewReader(body))
	if err != nil {
		return trackingPayload{}, nil, err
	}

	for key, value := range payload.Headers {
		request.Header.Set(key, value)
	}
	if payload.ContentType != "" {
		request.Header.Set("Content-Type", payload.ContentType)
	}
	if err := applyHTTPAuth(request, a.endpoint.HTTP); err != nil {
		return trackingPayload{}, nil, err
	}

	response, err := a.client.Do(request)
	if err != nil {
		return trackingPayload{}, nil, err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		response.Body.Close()
		return trackingPayload{}, nil, err
	}
	_ = response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(responseBody))

	return payloadFromResponse(response, responseBody), response, nil
}

func (a *httpEndpointAdapter) Close() error {
	return nil
}

func applyHTTPAuth(request *http.Request, options *HTTPEndpointOptions) error {
	if request == nil || options == nil || options.Auth == nil {
		return nil
	}

	auth := options.Auth
	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "", "none":
		return nil
	case "basic":
		raw := auth.Username + ":" + auth.Password
		request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
		return nil
	case "bearer":
		request.Header.Set("Authorization", "Bearer "+auth.BearerToken)
		return nil
	case "oauth2clientcredentials":
		tokenURL := strings.TrimSpace(auth.TokenURL)
		if auth.OAuth2ClientCredentials != nil && strings.TrimSpace(auth.OAuth2ClientCredentials.TokenEndpoint) != "" {
			tokenURL = strings.TrimSpace(auth.OAuth2ClientCredentials.TokenEndpoint)
			auth.ClientID = firstNonBlank(auth.ClientID, auth.OAuth2ClientCredentials.ClientID)
			auth.ClientSecret = firstNonBlank(auth.ClientSecret, auth.OAuth2ClientCredentials.ClientSecret)
			if len(auth.Scopes) == 0 {
				auth.Scopes = append([]string(nil), auth.OAuth2ClientCredentials.Scopes...)
			}
			if len(auth.AdditionalFormFields) == 0 {
				auth.AdditionalFormFields = cloneStringMap(auth.OAuth2ClientCredentials.AdditionalFormFields)
			}
		}
		if tokenURL == "" {
			return fmt.Errorf("oauth2 client credentials token endpoint is required")
		}

		form := url.Values{}
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", auth.ClientID)
		form.Set("client_secret", auth.ClientSecret)
		if auth.Scope != "" {
			form.Set("scope", auth.Scope)
		} else if len(auth.Scopes) > 0 {
			form.Set("scope", strings.Join(auth.Scopes, " "))
		}
		if auth.Audience != "" {
			form.Set("audience", auth.Audience)
		}
		for key, value := range auth.AdditionalFormFields {
			form.Set(key, value)
		}

		tokenRequest, err := http.NewRequestWithContext(request.Context(), http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for key, value := range options.TokenRequestHeaders {
			tokenRequest.Header.Set(key, value)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		response, err := client.Do(tokenRequest)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("oauth token request failed with status %d", response.StatusCode)
		}

		var tokenPayload map[string]any
		if err := json.Unmarshal(body, &tokenPayload); err != nil {
			return err
		}
		token := strings.TrimSpace(asString(tokenPayload["access_token"]))
		if token == "" {
			return fmt.Errorf("oauth token response did not include access_token")
		}

		headerName := firstNonBlank(auth.TokenHeaderName, "Authorization")
		request.Header.Set(headerName, "Bearer "+token)
		return nil
	default:
		return fmt.Errorf("unsupported http auth type %q", auth.Type)
	}
}

func extractPayloadArray(body any, path string) []any {
	decoded := normalizeJSONBody(body)
	if path == "$" {
		if items, ok := decoded.([]any); ok {
			return append([]any(nil), items...)
		}
		return nil
	}

	record, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}

	current := any(record)
	trimmed := strings.TrimPrefix(strings.TrimSpace(path), "$.")
	for _, segment := range strings.Split(trimmed, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		next, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = next[segment]
	}
	items, ok := current.([]any)
	if !ok {
		return nil
	}
	return append([]any(nil), items...)
}

type natsEndpointAdapter struct {
	endpoint     *EndpointSpec
	connection   *nats.Conn
	subscription *nats.Subscription
	messageCh    chan *nats.Msg
	ready        chan struct{}
	readyOnce    sync.Once
}

func newNATSEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.NATS == nil {
		return nil, fmt.Errorf("nats options are required for endpoint %q", endpoint.Name)
	}

	options := []nats.Option{
		nats.Name(firstNonBlank(endpoint.NATS.ConnectionName, endpoint.Name)),
	}
	if endpoint.NATS.MaxReconnectAttempts > 0 {
		options = append(options, nats.MaxReconnects(endpoint.NATS.MaxReconnectAttempts))
	}
	if endpoint.NATS.Token != "" {
		options = append(options, nats.Token(endpoint.NATS.Token))
	}
	if endpoint.NATS.UserName != "" {
		options = append(options, nats.UserInfo(endpoint.NATS.UserName, endpoint.NATS.Password))
	}

	connection, err := nats.Connect(strings.TrimSpace(endpoint.NATS.ServerURL), options...)
	if err != nil {
		return nil, err
	}

	adapter := &natsEndpointAdapter{
		endpoint:   endpoint,
		connection: connection,
		ready:      make(chan struct{}),
	}
	if strings.EqualFold(endpoint.Mode, "Consume") {
		adapter.messageCh = make(chan *nats.Msg, 256)
		if strings.TrimSpace(endpoint.NATS.QueueGroup) != "" {
			adapter.subscription, err = connection.ChanQueueSubscribe(endpoint.NATS.Subject, endpoint.NATS.QueueGroup, adapter.messageCh)
		} else {
			adapter.subscription, err = connection.ChanSubscribe(endpoint.NATS.Subject, adapter.messageCh)
		}
		if err != nil {
			connection.Close()
			return nil, err
		}
		if err := connection.Flush(); err != nil {
			_ = adapter.subscription.Unsubscribe()
			connection.Close()
			return nil, err
		}
		adapter.signalReady()
	}
	return adapter, nil
}

func (a *natsEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if !strings.EqualFold(a.endpoint.Mode, "Produce") {
		return endpointProduceResult{ErrorMessage: "NATS endpoint is not configured in Produce mode."}, nil
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
	message := nats.NewMsg(a.endpoint.NATS.Subject)
	message.Data = body
	message.Header = nats.Header{}
	for key, value := range payload.Headers {
		message.Header.Set(key, value)
	}
	if payload.ContentType != "" {
		message.Header.Set("content-type", payload.ContentType)
	}

	if err := a.connection.PublishMsg(message); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}
	flushContext := ctx
	var cancel stdcontext.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		flushContext, cancel = stdcontext.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	if err := a.connection.FlushWithContext(flushContext); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UnixMilli(),
		Payload:     payload,
	}, nil
}

func (a *natsEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if !strings.EqualFold(a.endpoint.Mode, "Consume") {
		return fmt.Errorf("NATS endpoint is not configured in Consume mode")
	}
	if a.messageCh == nil {
		return fmt.Errorf("NATS endpoint consumer subscription was not initialized")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case message := <-a.messageCh:
			if message == nil {
				continue
			}
			headers := map[string]string{}
			for key, values := range message.Header {
				headers[key] = strings.Join(values, ",")
			}
			payload := trackingPayload{
				Headers:     headers,
				Body:        append([]byte(nil), message.Data...),
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
				TimestampMS: time.Now().UnixMilli(),
				Payload:     payload,
			}); err != nil {
				return err
			}
		}
	}
}

func (a *natsEndpointAdapter) Close() error {
	if a.subscription != nil {
		_ = a.subscription.Unsubscribe()
	}
	if a.connection != nil {
		a.connection.Close()
	}
	return nil
}

func (a *natsEndpointAdapter) WaitReady(ctx stdcontext.Context) error {
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

func (a *natsEndpointAdapter) signalReady() {
	a.readyOnce.Do(func() {
		if a.ready != nil {
			close(a.ready)
		}
	})
}

type redisStreamsEndpointAdapter struct {
	endpoint *EndpointSpec
	client   redis.UniversalClient
}

func newRedisStreamsEndpointAdapter(endpoint *EndpointSpec) (endpointAdapter, error) {
	if endpoint.RedisStreams == nil {
		return nil, fmt.Errorf("redis streams options are required for endpoint %q", endpoint.Name)
	}

	options, err := redis.ParseURL(strings.TrimSpace(endpoint.RedisStreams.ConnectionString))
	if err != nil {
		return nil, err
	}

	return &redisStreamsEndpointAdapter{
		endpoint: endpoint,
		client:   redis.NewClient(options),
	}, nil
}

func (a *redisStreamsEndpointAdapter) Produce(ctx stdcontext.Context) (endpointProduceResult, error) {
	if !strings.EqualFold(a.endpoint.Mode, "Produce") {
		return endpointProduceResult{ErrorMessage: "Redis Streams endpoint is not configured in Produce mode."}, nil
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
	values := map[string]any{
		"body": body,
	}
	if payload.ContentType != "" {
		values["content-type"] = payload.ContentType
	}
	for key, value := range payload.Headers {
		values["header:"+key] = value
	}

	args := &redis.XAddArgs{
		Stream: a.endpoint.RedisStreams.StreamKey,
		Values: values,
	}
	if a.endpoint.RedisStreams.MaxLength > 0 {
		args.MaxLen = int64(a.endpoint.RedisStreams.MaxLength)
		args.Approx = true
	}

	if _, err := a.client.XAdd(ctx, args).Result(); err != nil {
		return endpointProduceResult{ErrorMessage: err.Error()}, nil
	}

	return endpointProduceResult{
		IsSuccess:   true,
		TrackingID:  trackingID,
		TimestampMS: time.Now().UnixMilli(),
		Payload:     payload,
	}, nil
}

func (a *redisStreamsEndpointAdapter) Consume(ctx stdcontext.Context, onMessage func(endpointConsumeEvent) error) error {
	if !strings.EqualFold(a.endpoint.Mode, "Consume") {
		return fmt.Errorf("Redis Streams endpoint is not configured in Consume mode")
	}

	group := firstNonBlank(a.endpoint.RedisStreams.ConsumerGroup, "loadstrike")
	consumer := firstNonBlank(a.endpoint.RedisStreams.ConsumerName, "loadstrike-go")
	start := "$"
	if a.endpoint.RedisStreams.StartFromEarliest {
		start = "0-0"
	}
	if err := a.client.XGroupCreateMkStream(ctx, a.endpoint.RedisStreams.StreamKey, group, start).Err(); err != nil && !strings.Contains(strings.ToUpper(err.Error()), "BUSYGROUP") {
		return err
	}

	readCount := int64(a.endpoint.RedisStreams.ReadCount)
	if readCount <= 0 {
		readCount = 10
	}
	block := trackingPollInterval(a.endpoint)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		streams, err := a.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{a.endpoint.RedisStreams.StreamKey, ">"},
			Count:    readCount,
			Block:    block,
		}).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				headers := map[string]string{}
				var body any
				contentType := ""
				for key, value := range message.Values {
					switch {
					case key == "body":
						body = value
					case key == "content-type":
						contentType = asString(value)
					case strings.HasPrefix(key, "header:"):
						headers[strings.TrimPrefix(key, "header:")] = asString(value)
					}
				}

				payload := trackingPayload{
					Headers:     headers,
					Body:        body,
					ContentType: contentType,
				}
				selector, err := parseTrackingFieldSelector(a.endpoint.TrackingField)
				if err != nil {
					return err
				}
				trackingID := selector.Extract(payload)
				if trackingID != "" {
					if err := onMessage(endpointConsumeEvent{
						TrackingID:  trackingID,
						TimestampMS: parseTimestampMS(message.ID, time.Now().UnixMilli()),
						Payload:     payload,
					}); err != nil {
						return err
					}
				}
				_, _ = a.client.XAck(ctx, a.endpoint.RedisStreams.StreamKey, group, message.ID).Result()
			}
		}
	}
}

func (a *redisStreamsEndpointAdapter) Close() error {
	return a.client.Close()
}
