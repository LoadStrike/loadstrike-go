package loadstrike

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type runtimeConfigFile struct {
	LoadStrike     runtimeConfigSection `json:"LoadStrike"`
	CustomSettings map[string]any       `json:"-"`
}

type runtimeConfigSection struct {
	TestSuite                   *string                        `json:"TestSuite"`
	TestName                    *string                        `json:"TestName"`
	SessionID                   *string                        `json:"SessionId"`
	ReportFolder                *string                        `json:"ReportFolder"`
	ReportFileName              *string                        `json:"ReportFileName"`
	ReportFormats               json.RawMessage                `json:"ReportFormats"`
	ReportingIntervalMS         *int                           `json:"ReportingIntervalMs"`
	ScenarioCompletionTimeoutMS *int                           `json:"ScenarioCompletionTimeoutMs"`
	ClusterCommandTimeoutMS     *int                           `json:"ClusterCommandTimeoutMs"`
	NodeType                    json.RawMessage                `json:"NodeType"`
	ClusterID                   *string                        `json:"ClusterId"`
	AgentGroup                  *string                        `json:"AgentGroup"`
	AgentID                     *string                        `json:"AgentId"`
	ExpectedAgentIDs            json.RawMessage                `json:"ExpectedAgentIds"`
	AgentsCount                 *int                           `json:"AgentsCount"`
	NatsServerURL               *string                        `json:"NatsServerUrl"`
	TargetScenarios             json.RawMessage                `json:"TargetScenarios"`
	AgentTargetScenarios        json.RawMessage                `json:"AgentTargetScenarios"`
	CoordinatorTargetScenarios  json.RawMessage                `json:"CoordinatorTargetScenarios"`
	MinimumLogLevel             *string                        `json:"MinimumLogLevel"`
	DisplayConsoleMetrics       json.RawMessage                `json:"DisplayConsoleMetrics"`
	EnableLocalDevCluster       json.RawMessage                `json:"EnableLocalDevCluster"`
	RestartIterationMaxAttempts *int                           `json:"RestartIterationMaxAttempts"`
	SinkRetryCount              *int                           `json:"SinkRetryCount"`
	SinkRetryBackoffMs          *int                           `json:"SinkRetryBackoffMs"`
	RunnerKey                   *string                        `json:"RunnerKey"`
	WithoutReports              json.RawMessage                `json:"WithoutReports"`
	UseLoadEngineV2             json.RawMessage                `json:"UseLoadEngineV2"`
	MaxInFlight                 *int                           `json:"MaxInFlight"`
	LicenseValidation           runtimeLicenseValidationConfig `json:"LicenseValidation"`
	parsedReportFormats         []ReportFormat
	parsedNodeType              *NodeType
	parsedExpectedAgentIDs      []string
	parsedTargetScenarios       []string
	parsedAgentTargetScenarios  []string
	parsedCoordinatorScenarios  []string
	parsedMinimumLogLevel       *LogEventLevel
	parsedDisplayConsoleMetrics *bool
	parsedEnableLocalDevCluster *bool
	parsedWithoutReports        *bool
	parsedUseLoadEngineV2       *bool
}

type runtimeLicenseValidationConfig struct {
	TimeoutMS *int `json:"TimeoutMs"`
}

var documentedRuntimeConfigKeys = []string{
	"TestSuite",
	"TestName",
	"SessionId",
	"ReportFolder",
	"ReportFileName",
	"ReportFormats",
	"ReportingIntervalMs",
	"ScenarioCompletionTimeoutMs",
	"ClusterCommandTimeoutMs",
	"NodeType",
	"ClusterId",
	"AgentGroup",
	"AgentId",
	"ExpectedAgentIds",
	"AgentsCount",
	"NatsServerUrl",
	"TargetScenarios",
	"AgentTargetScenarios",
	"CoordinatorTargetScenarios",
	"MinimumLogLevel",
	"DisplayConsoleMetrics",
	"EnableLocalDevCluster",
	"RestartIterationMaxAttempts",
	"SinkRetryCount",
	"SinkRetryBackoffMs",
	"RunnerKey",
	"WithoutReports",
	"UseLoadEngineV2",
	"MaxInFlight",
}

const maximumRuntimeConfigSinkRetryCount = 100

// LoadConfig applies supported settings from a loadstrike JSON config file.
func (r *runnerState) LoadConfig(path string) *runnerState {
	if strings.TrimSpace(path) == "" {
		r.context.recordConfigError(errors.New("config path must be provided"))
		return r
	}
	config, err := loadRuntimeConfig(path)
	if err != nil {
		r.context.recordConfigError(err)
		return r
	}

	applyRuntimeConfig(&r.context, config)
	return r
}

// LoadInfraConfig records the infra-config file path for sinks and plugins.
func (r *runnerState) LoadInfraConfig(path string) *runnerState {
	r.context.LoadInfraConfig(path)
	return r
}

func loadRuntimeConfig(path string) (runtimeConfigFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return runtimeConfigFile{}, err
	}

	if err := validateRuntimeConfigDocument(body); err != nil {
		return runtimeConfigFile{}, err
	}

	var config runtimeConfigFile
	if err := json.Unmarshal(body, &config); err != nil {
		return runtimeConfigFile{}, fmt.Errorf("invalid LoadStrike configuration: %w", err)
	}
	if err := normalizeRuntimeConfigSection(&config.LoadStrike); err != nil {
		return runtimeConfigFile{}, err
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return runtimeConfigFile{}, err
	}
	customSettings, err := cloneJSONCompatibleSettings(root)
	if err != nil {
		return runtimeConfigFile{}, fmt.Errorf("config custom settings: %w", err)
	}
	config.CustomSettings = customSettings

	return config, nil
}

func validateRuntimeConfigDocument(body []byte) error {
	if isJSONNull(body) {
		return errors.New("configuration root must be a JSON object and cannot be null")
	}
	root := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &root); err != nil {
		return err
	}
	if root == nil {
		return errors.New("configuration root must be a JSON object")
	}
	if err := rejectCaseInsensitiveDuplicateConfigKeys("configuration root", root); err != nil {
		return err
	}
	for _, key := range []string{"DisableLicenseEnforcement", "SkipLicenseValidation"} {
		if _, exists := caseInsensitiveRawConfigValue(root, key); exists {
			return fmt.Errorf("configuration.%s is not supported", key)
		}
	}

	loadStrikeRaw, found := caseInsensitiveRawConfigValue(root, "LoadStrike")
	if !found {
		return nil
	}
	if isJSONNull(loadStrikeRaw) {
		return errors.New("LoadStrike cannot be null")
	}
	section := map[string]json.RawMessage{}
	if err := json.Unmarshal(loadStrikeRaw, &section); err != nil {
		return errors.New("LoadStrike must be a JSON object")
	}
	if err := rejectCaseInsensitiveDuplicateConfigKeys("LoadStrike", section); err != nil {
		return err
	}

	for _, key := range []string{"DisableLicenseEnforcement", "SkipLicenseValidation"} {
		if _, exists := caseInsensitiveRawConfigValue(section, key); exists {
			return fmt.Errorf("LoadStrike.%s is not supported", key)
		}
	}
	for _, key := range documentedRuntimeConfigKeys {
		if value, exists := caseInsensitiveRawConfigValue(section, key); exists && isJSONNull(value) {
			return fmt.Errorf("LoadStrike.%s cannot be null", key)
		}
	}

	licenseRaw, hasLicenseValidation := caseInsensitiveRawConfigValue(section, "LicenseValidation")
	if !hasLicenseValidation {
		return nil
	}
	if isJSONNull(licenseRaw) {
		return errors.New("LoadStrike.LicenseValidation cannot be null")
	}
	license := map[string]json.RawMessage{}
	if err := json.Unmarshal(licenseRaw, &license); err != nil {
		return errors.New("LoadStrike.LicenseValidation must be a JSON object")
	}
	if err := rejectCaseInsensitiveDuplicateConfigKeys("LoadStrike.LicenseValidation", license); err != nil {
		return err
	}
	for _, key := range []string{"BaseUrl", "ApiUrl", "Url"} {
		if _, exists := caseInsensitiveRawConfigValue(license, key); exists {
			return fmt.Errorf("LoadStrike.LicenseValidation.%s is not supported", key)
		}
	}
	if timeout, exists := caseInsensitiveRawConfigValue(license, "TimeoutMs"); exists && isJSONNull(timeout) {
		return errors.New("LoadStrike.LicenseValidation.TimeoutMs cannot be null")
	}
	return nil
}

func rejectCaseInsensitiveDuplicateConfigKeys(scope string, values map[string]json.RawMessage) error {
	seen := map[string]string{}
	for key := range values {
		normalized := strings.ToLower(key)
		if first, exists := seen[normalized]; exists {
			return fmt.Errorf("%s.%s has duplicate case-insensitive spellings (%s and %s)", scope, first, first, key)
		}
		seen[normalized] = key
	}
	return nil
}

func caseInsensitiveRawConfigValue(values map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	for candidate, value := range values {
		if strings.EqualFold(candidate, key) {
			return value, true
		}
	}
	return nil, false
}

func isJSONNull(value json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

func normalizeRuntimeConfigSection(config *runtimeConfigSection) error {
	if config == nil {
		return nil
	}

	for _, field := range []struct {
		key   string
		value *string
	}{
		{"TestSuite", config.TestSuite},
		{"TestName", config.TestName},
		{"SessionId", config.SessionID},
		{"ReportFolder", config.ReportFolder},
		{"ReportFileName", config.ReportFileName},
		{"ClusterId", config.ClusterID},
		{"AgentGroup", config.AgentGroup},
		{"AgentId", config.AgentID},
		{"NatsServerUrl", config.NatsServerURL},
		{"RunnerKey", config.RunnerKey},
	} {
		if field.value != nil && strings.TrimSpace(*field.value) == "" {
			return fmt.Errorf("LoadStrike.%s must be provided", field.key)
		}
	}

	var err error
	if config.parsedReportFormats, err = parseRuntimeReportFormats(config.ReportFormats); err != nil {
		return err
	}
	if config.parsedNodeType, err = parseRuntimeNodeType(config.NodeType); err != nil {
		return err
	}
	if config.parsedExpectedAgentIDs, err = parseRuntimeStringList("ExpectedAgentIds", config.ExpectedAgentIDs); err != nil {
		return err
	}
	if config.parsedTargetScenarios, err = parseRuntimeStringList("TargetScenarios", config.TargetScenarios); err != nil {
		return err
	}
	if config.parsedAgentTargetScenarios, err = parseRuntimeStringList("AgentTargetScenarios", config.AgentTargetScenarios); err != nil {
		return err
	}
	if config.parsedCoordinatorScenarios, err = parseRuntimeStringList("CoordinatorTargetScenarios", config.CoordinatorTargetScenarios); err != nil {
		return err
	}
	if config.parsedMinimumLogLevel, err = parseRuntimeMinimumLogLevel(config.MinimumLogLevel); err != nil {
		return err
	}
	if config.parsedDisplayConsoleMetrics, err = parseRuntimeStrictBool("DisplayConsoleMetrics", config.DisplayConsoleMetrics); err != nil {
		return err
	}
	if config.parsedEnableLocalDevCluster, err = parseRuntimeStrictBool("EnableLocalDevCluster", config.EnableLocalDevCluster); err != nil {
		return err
	}
	if config.parsedWithoutReports, err = parseRuntimeStrictBool("WithoutReports", config.WithoutReports); err != nil {
		return err
	}
	if config.parsedUseLoadEngineV2, err = parseRuntimeStrictBool("UseLoadEngineV2", config.UseLoadEngineV2); err != nil {
		return err
	}

	for _, value := range []struct {
		key   string
		value *int
	}{
		{"ReportingIntervalMs", config.ReportingIntervalMS},
		{"ScenarioCompletionTimeoutMs", config.ScenarioCompletionTimeoutMS},
		{"ClusterCommandTimeoutMs", config.ClusterCommandTimeoutMS},
		{"LicenseValidation.TimeoutMs", config.LicenseValidation.TimeoutMS},
	} {
		if value.value != nil {
			if _, err := runtimeConfigMilliseconds(value.key, *value.value); err != nil {
				return err
			}
		}
	}
	if config.AgentsCount != nil && *config.AgentsCount <= 0 {
		return errors.New("LoadStrike.AgentsCount must be greater than zero")
	}
	for _, value := range []struct {
		key   string
		value *int
	}{
		{"RestartIterationMaxAttempts", config.RestartIterationMaxAttempts},
		{"SinkRetryCount", config.SinkRetryCount},
		{"SinkRetryBackoffMs", config.SinkRetryBackoffMs},
	} {
		if value.value != nil && *value.value < 0 {
			return fmt.Errorf("LoadStrike.%s must be zero or greater", value.key)
		}
	}
	if config.MaxInFlight != nil && (*config.MaxInFlight < 1 || *config.MaxInFlight > 1_000_000) {
		return errors.New("LoadStrike.MaxInFlight must be an integer from 1 through 1000000")
	}
	return nil
}

func parseRuntimeStrictBool(key string, raw json.RawMessage) (*bool, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var native bool
	if err := json.Unmarshal(raw, &native); err == nil {
		return &native, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true":
			value := true
			return &value, nil
		case "false":
			value := false
			return &value, nil
		}
	}
	return nil, fmt.Errorf("LoadStrike.%s must be true or false", key)
}

func parseRuntimeStringList(key string, raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []string
	var commaSeparated string
	switch {
	case json.Unmarshal(raw, &commaSeparated) == nil:
		values = strings.Split(commaSeparated, ",")
	case json.Unmarshal(raw, &values) == nil:
	default:
		return nil, fmt.Errorf("LoadStrike.%s must be a comma-delimited string or an array of strings", key)
	}

	normalized := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("LoadStrike.%s cannot contain blank or null values", key)
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("LoadStrike.%s must contain at least one value", key)
	}
	return normalized, nil
}

func parseRuntimeReportFormats(raw json.RawMessage) ([]ReportFormat, error) {
	values, err := parseRuntimeStringList("ReportFormats", raw)
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	formats := make([]ReportFormat, 0, len(values))
	seen := map[ReportFormat]struct{}{}
	for _, value := range values {
		var format ReportFormat
		switch strings.ToLower(value) {
		case "html":
			format = ReportFormatHTML
		case "txt":
			format = ReportFormatTXT
		case "csv":
			format = ReportFormatCSV
		case "md", "markdown":
			format = ReportFormatMD
		default:
			return nil, fmt.Errorf("LoadStrike.ReportFormats contains unsupported format %q", value)
		}
		if _, exists := seen[format]; exists {
			continue
		}
		seen[format] = struct{}{}
		formats = append(formats, format)
	}
	return formats, nil
}

func parseRuntimeNodeType(raw json.RawMessage) (*NodeType, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		if number >= int(NodeTypeSingleNode) && number <= int(NodeTypeAgent) {
			value := NodeType(number)
			return &value, nil
		}
		return nil, errors.New("LoadStrike.NodeType must be SingleNode, Coordinator, Agent, 0, 1, or 2")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		var value NodeType
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "single", "singlenode", "0":
			value = NodeTypeSingleNode
		case "coordinator", "1":
			value = NodeTypeCoordinator
		case "agent", "2":
			value = NodeTypeAgent
		default:
			return nil, errors.New("LoadStrike.NodeType must be SingleNode, Coordinator, Agent, 0, 1, or 2")
		}
		return &value, nil
	}
	return nil, errors.New("LoadStrike.NodeType must be SingleNode, Coordinator, Agent, 0, 1, or 2")
}

func parseRuntimeMinimumLogLevel(raw *string) (*LogEventLevel, error) {
	if raw == nil {
		return nil, nil
	}
	var level LogEventLevel
	switch strings.ToLower(strings.TrimSpace(*raw)) {
	case "verbose":
		level = LogEventLevelVerbose
	case "debug":
		level = LogEventLevelDebug
	case "information":
		level = LogEventLevelInformation
	case "warning":
		level = LogEventLevelWarning
	case "error":
		level = LogEventLevelError
	case "fatal":
		level = LogEventLevelFatal
	default:
		return nil, errors.New("LoadStrike.MinimumLogLevel must be Verbose, Debug, Information, Warning, Error, or Fatal")
	}
	return &level, nil
}

func runtimeConfigMilliseconds(key string, milliseconds int) (time.Duration, error) {
	const maximumDurationMilliseconds = int64(9_223_372_036_854)
	if milliseconds <= 0 || int64(milliseconds) > maximumDurationMilliseconds {
		return 0, fmt.Errorf("LoadStrike.%s must be a positive whole number of milliseconds within the supported duration range", key)
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}

func applyRuntimeConfig(context *contextState, config runtimeConfigFile) {
	if context == nil {
		return
	}

	candidate := *context
	applyRuntimeConfigCandidate(&candidate, config)
	*context = candidate
}

func applyRuntimeConfigCandidate(context *contextState, config runtimeConfigFile) {
	context.CustomSettings = cloneKnownJSONCompatibleSettings(config.CustomSettings)
	section := config.LoadStrike

	if section.TestSuite != nil {
		context.WithTestSuite(*section.TestSuite)
	}
	if section.TestName != nil {
		context.WithTestName(*section.TestName)
	}
	if section.SessionID != nil {
		context.WithSessionId(*section.SessionID)
	}
	if section.ReportFolder != nil {
		context.WithReportFolder(*section.ReportFolder)
	}
	if section.ReportFileName != nil {
		context.WithReportFileName(*section.ReportFileName)
	}
	if len(section.ReportFormats) != 0 {
		context.WithReportFormats(section.parsedReportFormats...)
	}
	if section.ReportingIntervalMS != nil {
		duration, _ := runtimeConfigMilliseconds("ReportingIntervalMs", *section.ReportingIntervalMS)
		context.WithReportingInterval(duration)
	}
	if section.ScenarioCompletionTimeoutMS != nil {
		duration, _ := runtimeConfigMilliseconds("ScenarioCompletionTimeoutMs", *section.ScenarioCompletionTimeoutMS)
		context.WithScenarioCompletionTimeout(duration)
	}
	if section.ClusterCommandTimeoutMS != nil {
		duration, _ := runtimeConfigMilliseconds("ClusterCommandTimeoutMs", *section.ClusterCommandTimeoutMS)
		context.WithClusterCommandTimeout(duration)
	}
	if section.parsedNodeType != nil {
		context.WithNodeType(*section.parsedNodeType)
	}
	if section.ClusterID != nil {
		context.WithClusterId(*section.ClusterID)
	}
	if section.AgentGroup != nil {
		context.WithAgentGroup(*section.AgentGroup)
	}
	if section.AgentID != nil {
		context.WithAgentID(*section.AgentID)
	}
	if len(section.ExpectedAgentIDs) != 0 {
		context.WithExpectedAgentIDs(section.parsedExpectedAgentIDs...)
	}
	if section.AgentsCount != nil {
		context.WithAgentsCount(*section.AgentsCount)
	}
	if section.NatsServerURL != nil {
		context.WithNatsServerUrl(*section.NatsServerURL)
	}
	if len(section.TargetScenarios) != 0 {
		context.WithTargetScenarios(section.parsedTargetScenarios...)
	}
	if len(section.AgentTargetScenarios) != 0 {
		context.WithAgentTargetScenarios(section.parsedAgentTargetScenarios...)
	}
	if len(section.CoordinatorTargetScenarios) != 0 {
		context.WithCoordinatorTargetScenarios(section.parsedCoordinatorScenarios...)
	}
	if section.parsedMinimumLogLevel != nil {
		context.WithMinimumLogLevel(*section.parsedMinimumLogLevel)
	}
	if section.parsedDisplayConsoleMetrics != nil {
		context.DisplayConsoleMetrics(*section.parsedDisplayConsoleMetrics)
	}
	if section.parsedEnableLocalDevCluster != nil {
		context.EnableLocalDevCluster(*section.parsedEnableLocalDevCluster)
	}
	if section.RestartIterationMaxAttempts != nil && *section.RestartIterationMaxAttempts >= 0 {
		context.WithRestartIterationMaxAttempts(*section.RestartIterationMaxAttempts)
	}
	if section.SinkRetryCount != nil && *section.SinkRetryCount >= 0 {
		context.SinkRetryCount = min(*section.SinkRetryCount, maximumRuntimeConfigSinkRetryCount)
	}
	if section.SinkRetryBackoffMs != nil && *section.SinkRetryBackoffMs >= 0 {
		context.SinkRetryBackoffMs = *section.SinkRetryBackoffMs
	}
	if section.RunnerKey != nil {
		context.WithRunnerKey(*section.RunnerKey)
	}
	if section.parsedWithoutReports != nil {
		context.ReportsEnabled = !*section.parsedWithoutReports
	}
	if section.parsedUseLoadEngineV2 != nil {
		if *section.parsedUseLoadEngineV2 {
			context.UseLoadEngineV2()
		} else {
			context.LoadEngineContractVersion = 0
		}
	}
	if section.MaxInFlight != nil {
		if context.LoadEngineContractVersion != 2 {
			context.recordConfigError(errors.New("LoadStrike.MaxInFlight is available only when Load Engine V2 is selected"))
		}
		context.WithMaxInFlight(*section.MaxInFlight)
	}
	if section.LicenseValidation.TimeoutMS != nil {
		duration, _ := runtimeConfigMilliseconds("LicenseValidation.TimeoutMs", *section.LicenseValidation.TimeoutMS)
		context.WithLicenseValidationTimeout(duration)
	}
}

func applyRunArgs(context *contextState, args []string) error {
	if context == nil {
		return nil
	}

	for _, arg := range args {
		name, value, ok := parseRunArg(arg)
		if !ok {
			continue
		}

		switch name {
		case "restartiterationmaxattempts":
			attempts, err := strconv.Atoi(value)
			if err == nil && attempts >= 0 {
				context.RestartIterationMaxAttempts = attempts
			}
		case "sinkretrycount":
			retries, err := strconv.Atoi(value)
			if err == nil && retries >= 0 {
				context.SinkRetryCount = retries
			}
		case "sinkretrybackoffms":
			backoffMs, err := strconv.Atoi(value)
			if err == nil && backoffMs >= 0 {
				context.SinkRetryBackoffMs = backoffMs
			}
		case "disablelicenseenforcement":
			return errors.New("disable license enforcement has been removed and is no longer supported")
		}
	}
	return nil
}

func parseRunArg(arg string) (string, string, bool) {
	if !strings.HasPrefix(arg, "--") {
		return "", "", false
	}

	name, value, found := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
	if !found || name == "" {
		return "", "", false
	}

	return strings.ToLower(strings.TrimSpace(name)), strings.TrimSpace(value), true
}
