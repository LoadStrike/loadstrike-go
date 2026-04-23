package loadstrike

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const defaultLicenseValidationTimeoutSeconds = 10.0
const defaultLicenseValidationBaseURL = "https://licensing.loadstrike.com"
const licenseValidationBaseURLEnv = "LOADSTRIKE_INTERNAL_BLACKBOX_API_BASE_URL"

// ReportFormat identifies a report file type.
type ReportFormat string

const (
	ReportFormatHTML ReportFormat = "html"
	ReportFormatTXT  ReportFormat = "txt"
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatMD   ReportFormat = "md"
)

type reportFailure struct {
	ScenarioName string
	StepName     string
	StatusCode   string
	Message      string
}

type reportTrace struct {
	stepNames       []string
	stepNameSet     map[string]struct{}
	failedResponses []reportFailure
}

func newReportTrace() *reportTrace {
	return &reportTrace{stepNameSet: make(map[string]struct{})}
}

func (t *reportTrace) addStep(stepName string) {
	if t == nil || stepName == "" {
		return
	}
	if _, exists := t.stepNameSet[stepName]; exists {
		return
	}
	t.stepNameSet[stepName] = struct{}{}
	t.stepNames = append(t.stepNames, stepName)
}

func (t *reportTrace) addFailure(scenarioName, stepName string, reply replyResult) {
	if t == nil {
		return
	}
	t.failedResponses = append(t.failedResponses, reportFailure{
		ScenarioName: scenarioName,
		StepName:     stepName,
		StatusCode:   reply.StatusCode,
		Message:      reply.Message,
	})
}

func (t *reportTrace) merge(other *reportTrace) {
	if t == nil || other == nil {
		return
	}
	for _, stepName := range other.stepNames {
		t.addStep(stepName)
	}
	t.failedResponses = append(t.failedResponses, other.failedResponses...)
}

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

// LoadStrikeReportingSinkBase provides default no-op lifecycle behavior.
type LoadStrikeReportingSinkBase struct{}

// Init initializes the current sdk object. Use this when preparing runtime state before execution.
func (LoadStrikeReportingSinkBase) Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask {
	return CompletedTask()
}

// Start starts the current sdk activity. Use this when beginning execution or sink processing.
func (LoadStrikeReportingSinkBase) Start(LoadStrikeSessionStartInfo) LoadStrikeTask {
	return CompletedTask()
}

// SaveRealtimeStats exposes the save realtime stats operation. Use this when interacting with the SDK through this surface.
func (LoadStrikeReportingSinkBase) SaveRealtimeStats([]LoadStrikeScenarioStats) LoadStrikeTask {
	return CompletedTask()
}

// SaveRealtimeMetrics exposes the save realtime metrics operation. Use this when interacting with the SDK through this surface.
func (LoadStrikeReportingSinkBase) SaveRealtimeMetrics(LoadStrikeMetricStats) LoadStrikeTask {
	return CompletedTask()
}

// SaveRunResult exposes the save run result operation. Use this when interacting with the SDK through this surface.
func (LoadStrikeReportingSinkBase) SaveRunResult(LoadStrikeRunResult) LoadStrikeTask {
	return CompletedTask()
}

// Stop stops the current sdk activity. Use this when finishing execution or shutting down a helper.
func (LoadStrikeReportingSinkBase) Stop() LoadStrikeTask {
	return CompletedTask()
}

// Dispose releases owned resources. Use this when the current SDK object is no longer needed.
func (LoadStrikeReportingSinkBase) Dispose() {}

type loadStrikeReportingSinkBase = LoadStrikeReportingSinkBase

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

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (s reportingSinkSpec) SinkName() string {
	return strings.TrimSpace(strings.ToLower(s.Kind))
}

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

type InfluxDbReportingSink struct {
	loadStrikeReportingSinkBase
	Options InfluxDbReportingSinkOptions
}

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (InfluxDbReportingSink) SinkName() string { return "influxdb" }

type TimescaleDbReportingSink struct {
	loadStrikeReportingSinkBase
	Options TimescaleDbReportingSinkOptions
}

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (TimescaleDbReportingSink) SinkName() string { return "timescaledb" }

type GrafanaLokiReportingSink struct {
	loadStrikeReportingSinkBase
	Options GrafanaLokiSinkOptions
}

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (GrafanaLokiReportingSink) SinkName() string { return "grafanaloki" }

type DatadogReportingSink struct {
	loadStrikeReportingSinkBase
	Options DatadogSinkOptions
}

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (DatadogReportingSink) SinkName() string { return "datadog" }

type SplunkReportingSink struct {
	loadStrikeReportingSinkBase
	Options SplunkSinkOptions
}

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (SplunkReportingSink) SinkName() string { return "splunk" }

type OtelCollectorReportingSink struct {
	loadStrikeReportingSinkBase
	Options OtelCollectorReportingSinkOptions
}

// SinkName exposes the sink name operation. Use this when interacting with the SDK through this surface.
func (OtelCollectorReportingSink) SinkName() string { return "otelcollector" }

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

var invalidReportFileCharacters = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizeReportFileName(value string) string {
	return invalidReportFileCharacters.ReplaceAllString(value, "_")
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func currentMachineName() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "unknown"
	}
	return name
}

func resolveLicensingAPIBaseURL() string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv(licenseValidationBaseURLEnv)), "/"); value != "" {
		return value
	}
	return defaultLicenseValidationBaseURL
}

func buildURL(baseURL string, relative string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(relative, "/")
}

func generateSessionID() string {
	return strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), "-", "")
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case bool:
		if typed {
			return 1
		}
		return 0
	default:
		var parsed float64
		if _, err := fmt.Sscan(fmt.Sprint(value), &parsed); err == nil {
			return int(parsed)
		}
		return 0
	}
}

func asDouble(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		var parsed float64
		if _, err := fmt.Sscan(fmt.Sprint(value), &parsed); err == nil {
			return parsed
		}
		return 0
	}
}

func asBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	default:
		text := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		return text == "1" || text == "true" || text == "yes" || text == "on"
	}
}
