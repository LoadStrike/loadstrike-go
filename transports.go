package loadstrike

import (
	stdcontext "context"
	"encoding/json"
)

// TrackingConfigurationSpec defines public cross-platform tracking options.
type TrackingConfigurationSpec struct {
	Source                             *EndpointSpec         `json:"Source,omitempty"`
	Destination                        *EndpointSpec         `json:"Destination,omitempty"`
	RunMode                            string                `json:"RunMode,omitempty"`
	ObservationDurationSeconds         float64               `json:"ObservationDurationSeconds,omitempty"`
	CorrelationTimeoutSeconds          float64               `json:"CorrelationTimeoutSeconds,omitempty"`
	TimeoutSweepIntervalSeconds        float64               `json:"TimeoutSweepIntervalSeconds,omitempty"`
	TimeoutBatchSize                   int                   `json:"TimeoutBatchSize,omitempty"`
	TimeoutCountsAsFailure             bool                  `json:"TimeoutCountsAsFailure,omitempty"`
	TrackingFieldValueCaseSensitive    bool                  `json:"TrackingFieldValueCaseSensitive"`
	GatherByFieldValueCaseSensitive    bool                  `json:"GatherByFieldValueCaseSensitive"`
	ExecuteOriginalScenarioRun         bool                  `json:"ExecuteOriginalScenarioRun,omitempty"`
	UseLoadStrikeTraceIDHeader         bool                  `json:"UseLoadStrikeTraceIdHeader,omitempty"`
	MetricPrefix                       string                `json:"MetricPrefix,omitempty"`
	CorrelationStore                   *CorrelationStoreSpec `json:"CorrelationStore,omitempty"`
	ObservationCancellationContext     stdcontext.Context    `json:"-"`
	ObservationCancellationCallbackURL string                `json:"ObservationCancellationCallbackUrl,omitempty"`
}

// LoadStrikeTrackingConfigurationSpec exposes the shared runtime-contract tracking name.
type LoadStrikeTrackingConfigurationSpec = TrackingConfigurationSpec

// ForDuration configures the observation window used by CorrelateExistingTraffic.
func (c *TrackingConfigurationSpec) ForDuration(duration TimeSpan, cancellationContexts ...stdcontext.Context) *TrackingConfigurationSpec {
	if c == nil {
		return nil
	}
	c.ObservationDurationSeconds = duration.Seconds()
	if len(cancellationContexts) > 0 && cancellationContexts[0] != nil {
		c.ObservationCancellationContext = cancellationContexts[0]
	}
	return c
}

// CorrelationStoreSpec defines correlation-store configuration.
type CorrelationStoreSpec struct {
	Kind  string                     `json:"Kind,omitempty"`
	Redis *RedisCorrelationStoreSpec `json:"Redis,omitempty"`
}

// RedisCorrelationStoreSpec defines Redis-backed correlation storage options.
type RedisCorrelationStoreSpec struct {
	ConnectionString string  `json:"ConnectionString,omitempty"`
	Database         int     `json:"Database,omitempty"`
	KeyPrefix        string  `json:"KeyPrefix,omitempty"`
	EntryTTLSeconds  float64 `json:"EntryTtlSeconds,omitempty"`
}

// EndpointSpec defines a public source or destination endpoint.
type EndpointSpec struct {
	Kind                              string                         `json:"Kind"`
	Name                              string                         `json:"Name"`
	Mode                              string                         `json:"Mode"`
	TrackingField                     string                         `json:"TrackingField"`
	GatherByField                     string                         `json:"GatherByField,omitempty"`
	AutoGenerateTrackingIDWhenMissing bool                           `json:"AutoGenerateTrackingIdWhenMissing,omitempty"`
	PollIntervalSeconds               float64                        `json:"PollIntervalSeconds,omitempty"`
	MessageHeaders                    map[string]string              `json:"MessageHeaders,omitempty"`
	MessagePayload                    RawPayload                     `json:"MessagePayload,omitempty"`
	MessagePayloadType                string                         `json:"MessagePayloadType,omitempty"`
	JsonSettings                      map[string]any                 `json:"JsonSettings,omitempty"`
	JsonConvertSettings               map[string]any                 `json:"JsonConvertSettings,omitempty"`
	ContentType                       string                         `json:"ContentType,omitempty"`
	HTTP                              *HTTPEndpointOptions           `json:"Http,omitempty"`
	Kafka                             *KafkaEndpointOptions          `json:"Kafka,omitempty"`
	RabbitMQ                          *RabbitMQEndpointOptions       `json:"RabbitMq,omitempty"`
	NATS                              *NATSEndpointOptions           `json:"Nats,omitempty"`
	RedisStreams                      *RedisStreamsEndpointOptions   `json:"RedisStreams,omitempty"`
	AzureEventHubs                    *AzureEventHubsEndpointOptions `json:"AzureEventHubs,omitempty"`
	SQS                               *SQSEndpointOptions            `json:"Sqs,omitempty"`
	DelegateStream                    *DelegateEndpointOptions       `json:"DelegateStream,omitempty"`
	PushDiffusion                     *PushDiffusionEndpointOptions  `json:"PushDiffusion,omitempty"`
	Grpc                              *GrpcEndpointOptions           `json:"Grpc,omitempty"`
	WebSocket                         *WebSocketEndpointOptions      `json:"WebSocket,omitempty"`
}

// HTTPEndpointOptions defines HTTP endpoint behavior.
type HTTPEndpointOptions struct {
	URL                      string            `json:"Url,omitempty"`
	Method                   string            `json:"Method,omitempty"`
	BodyType                 string            `json:"BodyType,omitempty"`
	RequestTimeoutSeconds    float64           `json:"RequestTimeoutSeconds,omitempty"`
	ConsumePoll              bool              `json:"ConsumePoll,omitempty"`
	ResponseSource           string            `json:"ResponseSource,omitempty"`
	TrackingPayloadSource    string            `json:"TrackingPayloadSource,omitempty"`
	ConsumeArrayPath         string            `json:"ConsumeArrayPath,omitempty"`
	ConsumeJSONArrayResponse bool              `json:"ConsumeJsonArrayResponse,omitempty"`
	TokenRequestHeaders      map[string]string `json:"TokenRequestHeaders,omitempty"`
	Auth                     *HTTPAuthOptions  `json:"Auth,omitempty"`
}

// HTTPAuthOptions defines HTTP authentication behavior.
type HTTPAuthOptions struct {
	Type                    string                              `json:"Type,omitempty"`
	Username                string                              `json:"Username,omitempty"`
	Password                string                              `json:"Password,omitempty"`
	BearerToken             string                              `json:"BearerToken,omitempty"`
	TokenURL                string                              `json:"TokenUrl,omitempty"`
	ClientID                string                              `json:"ClientId,omitempty"`
	ClientSecret            string                              `json:"ClientSecret,omitempty"`
	Scope                   string                              `json:"Scope,omitempty"`
	Scopes                  []string                            `json:"Scopes,omitempty"`
	Audience                string                              `json:"Audience,omitempty"`
	AdditionalFormFields    map[string]string                   `json:"AdditionalFormFields,omitempty"`
	TokenHeaderName         string                              `json:"TokenHeaderName,omitempty"`
	OAuth2ClientCredentials *HTTPOAuth2ClientCredentialsOptions `json:"OAuth2ClientCredentials,omitempty"`
}

// HTTPOAuth2ClientCredentialsOptions defines OAuth2 client credentials options.
type HTTPOAuth2ClientCredentialsOptions struct {
	TokenEndpoint        string            `json:"TokenEndpoint,omitempty"`
	ClientID             string            `json:"ClientId,omitempty"`
	ClientSecret         string            `json:"ClientSecret,omitempty"`
	Scopes               []string          `json:"Scopes,omitempty"`
	AdditionalFormFields map[string]string `json:"AdditionalFormFields,omitempty"`
}

// KafkaEndpointOptions defines Kafka endpoint options.
type KafkaEndpointOptions struct {
	BootstrapServers  string            `json:"BootstrapServers,omitempty"`
	Topic             string            `json:"Topic,omitempty"`
	ConsumerGroupID   string            `json:"ConsumerGroupId,omitempty"`
	SecurityProtocol  string            `json:"SecurityProtocol,omitempty"`
	SASL              *KafkaSASLOptions `json:"Sasl,omitempty"`
	ConfluentSettings map[string]string `json:"ConfluentSettings,omitempty"`
	StartFromEarliest bool              `json:"StartFromEarliest,omitempty"`
}

// KafkaSASLOptions defines Kafka SASL configuration.
type KafkaSASLOptions struct {
	Mechanism                   string                       `json:"Mechanism,omitempty"`
	Username                    string                       `json:"Username,omitempty"`
	Password                    string                       `json:"Password,omitempty"`
	OAuthBearerTokenEndpointURL string                       `json:"OAuthBearerTokenEndpointUrl,omitempty"`
	AdditionalSettings          map[string]string            `json:"AdditionalSettings,omitempty"`
	GSSAPI                      *KafkaSASLGSSAPIOptions      `json:"Gssapi,omitempty"`
	OAuthBearer                 *KafkaSASLOAuthBearerOptions `json:"OAuthBearer,omitempty"`
}

// KafkaSASLGSSAPIOptions defines Kerberos-backed SASL options.
type KafkaSASLGSSAPIOptions struct {
	ServiceName string `json:"ServiceName,omitempty"`
	Realm       string `json:"Realm,omitempty"`
	Username    string `json:"Username,omitempty"`
	Password    string `json:"Password,omitempty"`
}

// KafkaSASLOAuthBearerOptions defines OAuthBearer SASL options.
type KafkaSASLOAuthBearerOptions struct {
	AccessToken string            `json:"AccessToken,omitempty"`
	Extensions  map[string]string `json:"Extensions,omitempty"`
}

// RabbitMQEndpointOptions defines RabbitMQ endpoint options.
type RabbitMQEndpointOptions struct {
	HostName         string            `json:"HostName,omitempty"`
	Port             int               `json:"Port,omitempty"`
	VirtualHost      string            `json:"VirtualHost,omitempty"`
	UserName         string            `json:"UserName,omitempty"`
	Password         string            `json:"Password,omitempty"`
	Exchange         string            `json:"Exchange,omitempty"`
	RoutingKey       string            `json:"RoutingKey,omitempty"`
	QueueName        string            `json:"QueueName,omitempty"`
	Durable          bool              `json:"Durable,omitempty"`
	AutoAck          bool              `json:"AutoAck,omitempty"`
	UseSSL           bool              `json:"UseSsl,omitempty"`
	ClientProperties map[string]string `json:"ClientProperties,omitempty"`
}

// NATSEndpointOptions defines NATS endpoint options.
type NATSEndpointOptions struct {
	ServerURL            string `json:"ServerUrl,omitempty"`
	Subject              string `json:"Subject,omitempty"`
	QueueGroup           string `json:"QueueGroup,omitempty"`
	UserName             string `json:"UserName,omitempty"`
	Password             string `json:"Password,omitempty"`
	Token                string `json:"Token,omitempty"`
	ConnectionName       string `json:"ConnectionName,omitempty"`
	MaxReconnectAttempts int    `json:"MaxReconnectAttempts,omitempty"`
}

// RedisStreamsEndpointOptions defines Redis Streams endpoint options.
type RedisStreamsEndpointOptions struct {
	ConnectionString  string `json:"ConnectionString,omitempty"`
	StreamKey         string `json:"StreamKey,omitempty"`
	ConsumerGroup     string `json:"ConsumerGroup,omitempty"`
	ConsumerName      string `json:"ConsumerName,omitempty"`
	StartFromEarliest bool   `json:"StartFromEarliest,omitempty"`
	ReadCount         int    `json:"ReadCount,omitempty"`
	MaxLength         int    `json:"MaxLength,omitempty"`
}

// AzureEventHubsEndpointOptions defines Azure Event Hubs endpoint options.
type AzureEventHubsEndpointOptions struct {
	ConnectionString  string `json:"ConnectionString,omitempty"`
	EventHubName      string `json:"EventHubName,omitempty"`
	ConsumerGroup     string `json:"ConsumerGroup,omitempty"`
	StartFromEarliest bool   `json:"StartFromEarliest,omitempty"`
	PartitionID       string `json:"PartitionId,omitempty"`
	PartitionKey      string `json:"PartitionKey,omitempty"`
	PartitionCount    int    `json:"PartitionCount,omitempty"`
}

// SQSEndpointOptions defines AWS SQS endpoint options.
type SQSEndpointOptions struct {
	QueueURL                 string `json:"QueueUrl,omitempty"`
	Region                   string `json:"Region,omitempty"`
	ServiceURL               string `json:"ServiceUrl,omitempty"`
	AccessKeyID              string `json:"AccessKeyId,omitempty"`
	SecretAccessKey          string `json:"SecretAccessKey,omitempty"`
	SessionToken             string `json:"SessionToken,omitempty"`
	WaitTimeSeconds          int    `json:"WaitTimeSeconds,omitempty"`
	MaxNumberOfMessages      int    `json:"MaxNumberOfMessages,omitempty"`
	VisibilityTimeoutSeconds int    `json:"VisibilityTimeoutSeconds,omitempty"`
	DeleteAfterConsume       *bool  `json:"DeleteAfterConsume,omitempty"`
}

// DelegateEndpointOptions defines delegate-stream callback options.
type DelegateEndpointOptions struct {
	ProduceCallbackURL string                                                                   `json:"ProduceCallbackUrl,omitempty"`
	ConsumeCallbackURL string                                                                   `json:"ConsumeCallbackUrl,omitempty"`
	ConnectionMetadata map[string]string                                                        `json:"ConnectionMetadata,omitempty"`
	Produce            func(stdcontext.Context, TrackingPayload) (EndpointProduceResult, error) `json:"-"`
	Consume            func(stdcontext.Context, func(EndpointConsumeEvent) error) error         `json:"-"`
}

// PushDiffusionEndpointOptions defines Push Diffusion endpoint options.
type PushDiffusionEndpointOptions struct {
	ServerURL            string                                                                   `json:"ServerUrl,omitempty"`
	TopicPath            string                                                                   `json:"TopicPath,omitempty"`
	Principal            string                                                                   `json:"Principal,omitempty"`
	Password             string                                                                   `json:"Password,omitempty"`
	ConnectionProperties map[string]string                                                        `json:"ConnectionProperties,omitempty"`
	PublishCallbackURL   string                                                                   `json:"PublishCallbackUrl,omitempty"`
	SubscribeCallbackURL string                                                                   `json:"SubscribeCallbackUrl,omitempty"`
	Publish              func(stdcontext.Context, TrackingPayload) (EndpointProduceResult, error) `json:"-"`
	Subscribe            func(stdcontext.Context, func(EndpointConsumeEvent) error) error         `json:"-"`
}

// GrpcEndpointOptions defines delegate-backed gRPC endpoint options.
type GrpcEndpointOptions struct {
	Target             string                                                                   `json:"Target,omitempty"`
	ServiceName        string                                                                   `json:"ServiceName,omitempty"`
	MethodName         string                                                                   `json:"MethodName,omitempty"`
	MethodType         string                                                                   `json:"MethodType,omitempty"`
	DeadlineSeconds    float64                                                                  `json:"DeadlineSeconds,omitempty"`
	Metadata           map[string]string                                                        `json:"Metadata,omitempty"`
	ConnectionMetadata map[string]string                                                        `json:"ConnectionMetadata,omitempty"`
	NativeClient       *GrpcNativeClientOptions                                                 `json:"NativeClient,omitempty"`
	ProduceCallbackURL string                                                                   `json:"ProduceCallbackUrl,omitempty"`
	ConsumeCallbackURL string                                                                   `json:"ConsumeCallbackUrl,omitempty"`
	Produce            func(stdcontext.Context, TrackingPayload) (EndpointProduceResult, error) `json:"-"`
	Consume            func(stdcontext.Context, func(EndpointConsumeEvent) error) error         `json:"-"`
}

type GrpcNativeClientOptions struct {
	ProtoFilePath                string            `json:"ProtoFilePath,omitempty"`
	DescriptorSetPath            string            `json:"DescriptorSetPath,omitempty"`
	UseReflection                bool              `json:"UseReflection,omitempty"`
	UseTLS                       bool              `json:"UseTls,omitempty"`
	AllowUntrustedCertificates   bool              `json:"AllowUntrustedCertificates,omitempty"`
	DeadlineSeconds              float64           `json:"DeadlineSeconds,omitempty"`
	Metadata                     map[string]string `json:"Metadata,omitempty"`
	RequestPayloadJSON           string            `json:"RequestPayloadJson,omitempty"`
	RequestPayloadJSONStream     []string          `json:"RequestPayloadJsonStream,omitempty"`
	TrackStatusCodes             bool              `json:"TrackStatusCodes,omitempty"`
	TrackStreamingMessageLatency bool              `json:"TrackStreamingMessageLatency,omitempty"`
}

type GrpcStatusMapping struct {
	StatusCode int    `json:"StatusCode"`
	StatusName string `json:"StatusName"`
	IsSuccess  bool   `json:"IsSuccess"`
}

func MapGrpcStatusCode(statusCode int) GrpcStatusMapping {
	names := map[int]string{
		0: "OK", 1: "CANCELLED", 2: "UNKNOWN", 3: "INVALID_ARGUMENT",
		4: "DEADLINE_EXCEEDED", 5: "NOT_FOUND", 6: "ALREADY_EXISTS",
		7: "PERMISSION_DENIED", 8: "RESOURCE_EXHAUSTED", 9: "FAILED_PRECONDITION",
		10: "ABORTED", 11: "OUT_OF_RANGE", 12: "UNIMPLEMENTED", 13: "INTERNAL",
		14: "UNAVAILABLE", 15: "DATA_LOSS", 16: "UNAUTHENTICATED",
	}
	name := names[statusCode]
	if name == "" {
		name = "UNKNOWN"
	}
	return GrpcStatusMapping{StatusCode: statusCode, StatusName: name, IsSuccess: statusCode == 0}
}

type ProtocolMetricSnapshot struct {
	Protocol         string  `json:"Protocol,omitempty"`
	Requests         int     `json:"Requests,omitempty"`
	MessagesSent     int     `json:"MessagesSent,omitempty"`
	MessagesReceived int     `json:"MessagesReceived,omitempty"`
	LatencyMS        float64 `json:"LatencyMs,omitempty"`
	StreamDurationMS float64 `json:"StreamDurationMs,omitempty"`
	BytesSent        int64   `json:"BytesSent,omitempty"`
	BytesReceived    int64   `json:"BytesReceived,omitempty"`
	Reconnects       int     `json:"Reconnects,omitempty"`
	Errors           int     `json:"Errors,omitempty"`
	Status           string  `json:"Status,omitempty"`
}

// WebSocketEndpointOptions defines delegate-backed WebSocket endpoint options.
type WebSocketEndpointOptions struct {
	URL                   string                                                                   `json:"Url,omitempty"`
	Subprotocols          []string                                                                 `json:"Subprotocols,omitempty"`
	ConnectTimeoutSeconds float64                                                                  `json:"ConnectTimeoutSeconds,omitempty"`
	CloseTimeoutSeconds   float64                                                                  `json:"CloseTimeoutSeconds,omitempty"`
	ConnectionMetadata    map[string]string                                                        `json:"ConnectionMetadata,omitempty"`
	NativeClient          *WebSocketNativeClientOptions                                            `json:"NativeClient,omitempty"`
	ProduceCallbackURL    string                                                                   `json:"ProduceCallbackUrl,omitempty"`
	ConsumeCallbackURL    string                                                                   `json:"ConsumeCallbackUrl,omitempty"`
	Produce               func(stdcontext.Context, TrackingPayload) (EndpointProduceResult, error) `json:"-"`
	Consume               func(stdcontext.Context, func(EndpointConsumeEvent) error) error         `json:"-"`
}

type WebSocketNativeClientOptions struct {
	Headers               map[string]string          `json:"Headers,omitempty"`
	Messages              []WebSocketMessageSpec     `json:"Messages,omitempty"`
	ExpectedMessages      []WebSocketExpectedMessage `json:"ExpectedMessages,omitempty"`
	ReconnectPolicy       *WebSocketReconnectPolicy  `json:"ReconnectPolicy,omitempty"`
	BinaryMessages        bool                       `json:"BinaryMessages,omitempty"`
	TrackPingPong         bool                       `json:"TrackPingPong,omitempty"`
	TrackCloseCodes       bool                       `json:"TrackCloseCodes,omitempty"`
	TrackMessageLatency   bool                       `json:"TrackMessageLatency,omitempty"`
	ReceiveTimeoutSeconds float64                    `json:"ReceiveTimeoutSeconds,omitempty"`
}

type WebSocketReconnectPolicy struct {
	Enabled        bool    `json:"Enabled,omitempty"`
	MaxAttempts    int     `json:"MaxAttempts,omitempty"`
	BackoffSeconds float64 `json:"BackoffSeconds,omitempty"`
}

type WebSocketMessageSpec struct {
	Type          string  `json:"Type,omitempty"`
	Payload       string  `json:"Payload,omitempty"`
	BinaryPayload []byte  `json:"BinaryPayload,omitempty"`
	DelaySeconds  float64 `json:"DelaySeconds,omitempty"`
}

func TextWebSocketMessage(payload string) WebSocketMessageSpec {
	return WebSocketMessageSpec{Type: "Text", Payload: payload}
}

func BinaryWebSocketMessage(payload []byte) WebSocketMessageSpec {
	return WebSocketMessageSpec{Type: "Binary", BinaryPayload: append([]byte(nil), payload...)}
}

type WebSocketExpectedMessage struct {
	MatchText      string  `json:"MatchText,omitempty"`
	MatchJSONPath  string  `json:"MatchJsonPath,omitempty"`
	TimeoutSeconds float64 `json:"TimeoutSeconds,omitempty"`
}

// RawPayloadFromAny marshals an arbitrary payload into raw JSON.
func RawPayloadFromAny(value any) (RawPayload, error) {
	if value == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return RawPayload(encoded), nil
}
