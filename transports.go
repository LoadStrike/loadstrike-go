package loadstrike

import (
	stdcontext "context"
	"encoding/json"
)

// TrackingConfigurationSpec defines public cross-platform tracking options.
type TrackingConfigurationSpec struct {
	Source                          *EndpointSpec         `json:"Source,omitempty"`
	Destination                     *EndpointSpec         `json:"Destination,omitempty"`
	RunMode                         string                `json:"RunMode,omitempty"`
	CorrelationTimeoutSeconds       float64               `json:"CorrelationTimeoutSeconds,omitempty"`
	TimeoutSweepIntervalSeconds     float64               `json:"TimeoutSweepIntervalSeconds,omitempty"`
	TimeoutBatchSize                int                   `json:"TimeoutBatchSize,omitempty"`
	TimeoutCountsAsFailure          bool                  `json:"TimeoutCountsAsFailure,omitempty"`
	TrackingFieldValueCaseSensitive bool                  `json:"TrackingFieldValueCaseSensitive,omitempty"`
	GatherByFieldValueCaseSensitive bool                  `json:"GatherByFieldValueCaseSensitive,omitempty"`
	ExecuteOriginalScenarioRun      bool                  `json:"ExecuteOriginalScenarioRun,omitempty"`
	MetricPrefix                    string                `json:"MetricPrefix,omitempty"`
	CorrelationStore                *CorrelationStoreSpec `json:"CorrelationStore,omitempty"`
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
	DelegateStream                    *DelegateEndpointOptions       `json:"DelegateStream,omitempty"`
	PushDiffusion                     *PushDiffusionEndpointOptions  `json:"PushDiffusion,omitempty"`
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
