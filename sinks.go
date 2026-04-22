package loadstrike

import "strings"

// LoadStrikeReportingSink mirrors the .NET public reporting-sink contract.
type LoadStrikeReportingSink interface {
	SinkName() string
	Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
	Start(LoadStrikeSessionStartInfo) LoadStrikeTask
	SaveRealtimeStats([]LoadStrikeScenarioStats) LoadStrikeTask
	SaveRealtimeMetrics(LoadStrikeMetricStats) LoadStrikeTask
	SaveRunResult(LoadStrikeRunResult) LoadStrikeTask
	Stop() LoadStrikeTask
	Dispose()
}

// LoadStrikeReportingSinkBase provides default no-op lifecycle behavior,
// mirroring .NET default interface members in the closest valid Go form.
type LoadStrikeReportingSinkBase struct{}

func (LoadStrikeReportingSinkBase) Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeReportingSinkBase) Start(LoadStrikeSessionStartInfo) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeReportingSinkBase) SaveRealtimeStats([]LoadStrikeScenarioStats) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeReportingSinkBase) SaveRealtimeMetrics(LoadStrikeMetricStats) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeReportingSinkBase) SaveRunResult(LoadStrikeRunResult) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeReportingSinkBase) Stop() LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeReportingSinkBase) Dispose() {}

type loadStrikeReportingSinkBase = LoadStrikeReportingSinkBase

type loadStrikeTaskReportingSinkCore interface {
	SinkName() string
	Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
	Start(LoadStrikeSessionStartInfo) LoadStrikeTask
	SaveRealtimeStats([]LoadStrikeScenarioStats) LoadStrikeTask
	SaveRealtimeMetrics(LoadStrikeMetricStats) LoadStrikeTask
	Stop() LoadStrikeTask
}

type loadStrikeTaskReportingSinkInitializer interface {
	Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
}

type loadStrikeTaskReportingSinkStarter interface {
	Start(LoadStrikeSessionStartInfo) LoadStrikeTask
}

type loadStrikeTaskReportingSinkRealtimeStatsSaver interface {
	SaveRealtimeStats([]LoadStrikeScenarioStats) LoadStrikeTask
}

type loadStrikeTaskReportingSinkRealtimeMetricsSaver interface {
	SaveRealtimeMetrics(LoadStrikeMetricStats) LoadStrikeTask
}

type loadStrikeTaskReportingSinkStopper interface {
	Stop() LoadStrikeTask
}

type loadStrikeTaskReportingSinkRunResultSaver interface {
	SaveRunResult(LoadStrikeRunResult) LoadStrikeTask
}

type loadStrikeTaskReportingSinkDisposer interface {
	Dispose() LoadStrikeTask
}

type loadStrikeAsyncReportingSink interface {
	SinkName() string
	InitAsync(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
	StartAsync(LoadStrikeSessionStartInfo) LoadStrikeTask
	SaveRealtimeStatsAsync([]LoadStrikeScenarioStats) LoadStrikeTask
	SaveRealtimeMetricsAsync(LoadStrikeMetricStats) LoadStrikeTask
	SaveRunResultAsync(LoadStrikeRunResult) LoadStrikeTask
	StopAsync() LoadStrikeTask
}

type legacyLoadStrikeReportingSink interface {
	SinkName() string
	Init(LoadStrikeBaseContext, IConfiguration) error
	Start(LoadStrikeSessionStartInfo) error
	SaveRealtimeStats([]LoadStrikeScenarioStats) error
	SaveRealtimeMetrics(LoadStrikeMetricStats) error
	SaveRunResult(LoadStrikeRunResult) error
	Stop() error
}

type legacyLoadStrikeReportingSinkDisposer interface {
	Dispose() error
}

// reportingSinkSpec defines an internal built-in reporting sink contract.
type reportingSinkSpec struct {
	loadStrikeReportingSinkBase
	Kind          string                    `json:"Kind"`
	InfluxDB      *InfluxDBSinkOptions      `json:"InfluxDb,omitempty"`
	TimescaleDB   *TimescaleDBSinkOptions   `json:"TimescaleDb,omitempty"`
	GrafanaLoki   *GrafanaLokiSinkOptions   `json:"GrafanaLoki,omitempty"`
	Datadog       *DatadogSinkOptions       `json:"Datadog,omitempty"`
	Splunk        *SplunkSinkOptions        `json:"Splunk,omitempty"`
	OTELCollector *OTELCollectorSinkOptions `json:"OtelCollector,omitempty"`
}

func (s reportingSinkSpec) SinkName() string {
	return strings.TrimSpace(strings.ToLower(s.Kind))
}

// InfluxDBSinkOptions defines InfluxDB sink options.
type InfluxDBSinkOptions struct {
	ConfigurationSectionPath string            `json:"ConfigurationSectionPath,omitempty"`
	BaseURL                  string            `json:"BaseUrl,omitempty"`
	WriteEndpointPath        string            `json:"WriteEndpointPath,omitempty"`
	Organization             string            `json:"Organization,omitempty"`
	Bucket                   string            `json:"Bucket,omitempty"`
	Token                    string            `json:"Token,omitempty"`
	MeasurementName          string            `json:"MeasurementName,omitempty"`
	MetricsMeasurementName   string            `json:"MetricsMeasurementName,omitempty"`
	TimeoutSeconds           int               `json:"TimeoutSeconds,omitempty"`
	StaticTags               map[string]string `json:"StaticTags,omitempty"`
}

// TimescaleDBSinkOptions defines TimescaleDB sink options.
type TimescaleDBSinkOptions struct {
	ConfigurationSectionPath    string            `json:"ConfigurationSectionPath,omitempty"`
	ConnectionString            string            `json:"ConnectionString,omitempty"`
	Schema                      string            `json:"Schema,omitempty"`
	TableName                   string            `json:"TableName,omitempty"`
	MetricsTableName            string            `json:"MetricsTableName,omitempty"`
	CreateSchemaIfMissing       bool              `json:"CreateSchemaIfMissing,omitempty"`
	EnableHypertableIfAvailable bool              `json:"EnableHypertableIfAvailable,omitempty"`
	StaticTags                  map[string]string `json:"StaticTags,omitempty"`
}

// GrafanaLokiSinkOptions defines Grafana Loki sink options.
type GrafanaLokiSinkOptions struct {
	ConfigurationSectionPath string            `json:"ConfigurationSectionPath,omitempty"`
	BaseURL                  string            `json:"BaseUrl,omitempty"`
	PushEndpointPath         string            `json:"PushEndpointPath,omitempty"`
	BearerToken              string            `json:"BearerToken,omitempty"`
	Username                 string            `json:"Username,omitempty"`
	Password                 string            `json:"Password,omitempty"`
	TenantID                 string            `json:"TenantId,omitempty"`
	TimeoutSeconds           int               `json:"TimeoutSeconds,omitempty"`
	StaticLabels             map[string]string `json:"StaticLabels,omitempty"`
	MetricsBaseURL           string            `json:"MetricsBaseUrl,omitempty"`
	MetricsEndpointPath      string            `json:"MetricsEndpointPath,omitempty"`
	MetricsHeaders           map[string]string `json:"MetricsHeaders,omitempty"`
}

// DatadogSinkOptions defines Datadog sink options.
type DatadogSinkOptions struct {
	ConfigurationSectionPath string            `json:"ConfigurationSectionPath,omitempty"`
	BaseURL                  string            `json:"BaseUrl,omitempty"`
	LogsEndpointPath         string            `json:"LogsEndpointPath,omitempty"`
	MetricsEndpointPath      string            `json:"MetricsEndpointPath,omitempty"`
	APIKey                   string            `json:"ApiKey,omitempty"`
	ApplicationKey           string            `json:"ApplicationKey,omitempty"`
	Source                   string            `json:"Source,omitempty"`
	Service                  string            `json:"Service,omitempty"`
	Host                     string            `json:"Host,omitempty"`
	TimeoutSeconds           int               `json:"TimeoutSeconds,omitempty"`
	StaticTags               map[string]string `json:"StaticTags,omitempty"`
	StaticAttributes         map[string]string `json:"StaticAttributes,omitempty"`
}

// SplunkSinkOptions defines Splunk HEC sink options.
type SplunkSinkOptions struct {
	ConfigurationSectionPath string            `json:"ConfigurationSectionPath,omitempty"`
	BaseURL                  string            `json:"BaseUrl,omitempty"`
	EventEndpointPath        string            `json:"EventEndpointPath,omitempty"`
	Token                    string            `json:"Token,omitempty"`
	Source                   string            `json:"Source,omitempty"`
	Sourcetype               string            `json:"Sourcetype,omitempty"`
	Index                    string            `json:"Index,omitempty"`
	Host                     string            `json:"Host,omitempty"`
	TimeoutSeconds           int               `json:"TimeoutSeconds,omitempty"`
	StaticFields             map[string]string `json:"StaticFields,omitempty"`
}

// OTELCollectorSinkOptions defines OTEL collector sink options.
type OTELCollectorSinkOptions struct {
	ConfigurationSectionPath string            `json:"ConfigurationSectionPath,omitempty"`
	BaseURL                  string            `json:"BaseUrl,omitempty"`
	LogsEndpointPath         string            `json:"LogsEndpointPath,omitempty"`
	MetricsEndpointPath      string            `json:"MetricsEndpointPath,omitempty"`
	TimeoutSeconds           int               `json:"TimeoutSeconds,omitempty"`
	Headers                  map[string]string `json:"Headers,omitempty"`
	StaticResourceAttributes map[string]string `json:"StaticResourceAttributes,omitempty"`
}

type InfluxDbReportingSinkOptions = InfluxDBSinkOptions
type TimescaleDbReportingSinkOptions = TimescaleDBSinkOptions
type OtelCollectorReportingSinkOptions = OTELCollectorSinkOptions

// InfluxDbReportingSink mirrors the .NET built-in sink type.
type InfluxDbReportingSink struct {
	loadStrikeReportingSinkBase
	Options InfluxDbReportingSinkOptions
}

func (InfluxDbReportingSink) SinkName() string { return "influxdb" }

// TimescaleDbReportingSink mirrors the .NET built-in sink type.
type TimescaleDbReportingSink struct {
	loadStrikeReportingSinkBase
	Options TimescaleDbReportingSinkOptions
}

func (TimescaleDbReportingSink) SinkName() string { return "timescaledb" }

// GrafanaLokiReportingSink mirrors the .NET built-in sink type.
type GrafanaLokiReportingSink struct {
	loadStrikeReportingSinkBase
	Options GrafanaLokiSinkOptions
}

func (GrafanaLokiReportingSink) SinkName() string { return "grafanaloki" }

// DatadogReportingSink mirrors the .NET built-in sink type.
type DatadogReportingSink struct {
	loadStrikeReportingSinkBase
	Options DatadogSinkOptions
}

func (DatadogReportingSink) SinkName() string { return "datadog" }

// SplunkReportingSink mirrors the .NET built-in sink type.
type SplunkReportingSink struct {
	loadStrikeReportingSinkBase
	Options SplunkSinkOptions
}

func (SplunkReportingSink) SinkName() string { return "splunk" }

// OtelCollectorReportingSink mirrors the .NET built-in sink type.
type OtelCollectorReportingSink struct {
	loadStrikeReportingSinkBase
	Options OtelCollectorReportingSinkOptions
}

func (OtelCollectorReportingSink) SinkName() string { return "otelcollector" }
