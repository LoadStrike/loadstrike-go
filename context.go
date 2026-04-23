package loadstrike

import (
	"fmt"
	"strings"
	"time"
)

// contextState stores execution-wide settings for a configured runner.
type contextState struct {
	ReportsEnabled                   bool
	RestartIterationMaxAttempts      int
	ReportingIntervalSeconds         float64
	LicenseValidationTimeoutSeconds  float64
	MinimumLogLevel                  string
	LoggerConfig                     LoggerConfiguration
	RunnerKey                        string
	SessionID                        string
	ClusterID                        string
	AgentGroup                       string
	NatsServerURL                    string
	NodeType                         NodeType
	AgentsCount                      int
	LocalDevClusterEnabled           bool
	TargetScenarios                  []string
	AgentTargetScenarios             []string
	CoordinatorTargetScenarios       []string
	ScenarioCompletionTimeoutSeconds float64
	ClusterCommandTimeoutSeconds     float64
	ConsoleMetricsEnabled            bool
	InfraConfigPath                  string
	ReportingSinks                   []LoadStrikeReportingSink
	WorkerPlugins                    []LoadStrikeWorkerPlugin
	RuntimePolicies                  []LoadStrikeRuntimePolicy
	RuntimePolicyErrorMode           RuntimePolicyErrorMode
	ReportFileName                   string
	ReportFolder                     string
	ReportFormats                    []ReportFormat
	TestSuite                        string
	TestName                         string
	testInfo                         testInfo
	nodeInfo                         nodeInfo
	Logger                           *LoadStrikeLogger
	reportConfigured                 bool
	scenarios                        []scenarioDefinition
	configError                      error
	agentIndex                       int
	agentCount                       int
}

// WithRunnerKey records the runner key used for this run.
func (c *contextState) WithRunnerKey(runnerKey string) *contextState {
	if c != nil {
		c.requireNonBlank("runner key", runnerKey)
		c.RunnerKey = runnerKey
	}
	return c
}

// WithTestSuite sets the run test suite label.
func (c *contextState) WithTestSuite(testSuite string) *contextState {
	if c != nil {
		c.requireNonBlank("test suite", testSuite)
		c.TestSuite = testSuite
	}
	return c
}

// WithTestName sets the run test name label.
func (c *contextState) WithTestName(testName string) *contextState {
	if c != nil {
		c.requireNonBlank("test name", testName)
		c.TestName = testName
	}
	return c
}

// WithReportingInterval sets the realtime reporting cadence.
func (c *contextState) WithReportingInterval(value TimeSpan) *contextState {
	if c != nil {
		c.ReportingIntervalSeconds = durationArgumentSeconds("reporting interval", value)
	}
	return c
}

// WithLicenseValidationTimeout sets the licensing validation timeout.
func (c *contextState) WithLicenseValidationTimeout(value TimeSpan) *contextState {
	if c != nil {
		c.LicenseValidationTimeoutSeconds = durationArgumentSeconds("license validation timeout", value)
	}
	return c
}

// WithMinimumLogLevel records the minimum log level for this run.
func (c *contextState) WithMinimumLogLevel(level LogEventLevel) *contextState {
	if c != nil {
		c.MinimumLogLevel = normalizeLogEventLevel(level)
	}
	return c
}

// WithLoggerConfig records logger configuration overrides for this run.
func (c *contextState) WithLoggerConfig(config LoggerConfigurationFactory) *contextState {
	if c != nil {
		c.LoggerConfig = normalizeLoggerConfiguration(config)
	}
	return c
}

// WithSessionId records the session identifier used for this run.
func (c *contextState) WithSessionId(sessionID string) *contextState {
	if c != nil {
		c.requireNonBlank("session id", sessionID)
		c.SessionID = sessionID
	}
	return c
}

// WithClusterId records the cluster identifier used for this run.
func (c *contextState) WithClusterId(clusterID string) *contextState {
	if c != nil {
		c.requireNonBlank("cluster id", clusterID)
		c.ClusterID = clusterID
	}
	return c
}

// WithAgentGroup records the agent group used for this run.
func (c *contextState) WithAgentGroup(agentGroup string) *contextState {
	if c != nil {
		c.requireNonBlank("agent group", agentGroup)
		c.AgentGroup = agentGroup
	}
	return c
}

// WithNatsServerURL records the NATS server URL used for distributed cluster mode.
func (c *contextState) WithNatsServerURL(serverURL string) *contextState {
	if c != nil {
		c.requireNonBlank("nats server url", serverURL)
		c.NatsServerURL = serverURL
	}
	return c
}

// WithNatsServerUrl mirrors the exact .NET fluent method name.
func (c *contextState) WithNatsServerUrl(serverURL string) *contextState {
	return c.WithNatsServerURL(serverURL)
}

// WithNodeType sets the runtime node type.
func (c *contextState) WithNodeType(nodeType NodeType) *contextState {
	if c != nil {
		c.NodeType = nodeType
	}
	return c
}

// EnableLocalDevCluster toggles local in-process cluster execution mode.
func (c *contextState) EnableLocalDevCluster(enable bool) *contextState {
	if c != nil {
		c.LocalDevClusterEnabled = enable
	}
	return c
}

// WithAgentsCount sets the requested agent count.
func (c *contextState) WithAgentsCount(agentsCount int) *contextState {
	if c != nil {
		c.requirePositiveInt("agents count", agentsCount)
		c.AgentsCount = agentsCount
	}
	return c
}

// WithTargetScenarios records a shared scenario target list.
func (c *contextState) WithTargetScenarios(targetScenarios ...string) *contextState {
	if c != nil {
		c.validateNames("target scenarios", targetScenarios)
		c.TargetScenarios = append([]string(nil), targetScenarios...)
	}
	return c
}

// WithAgentTargetScenarios records an agent-only scenario target list.
func (c *contextState) WithAgentTargetScenarios(targetScenarios ...string) *contextState {
	if c != nil {
		c.validateNames("agent target scenarios", targetScenarios)
		c.AgentTargetScenarios = append([]string(nil), targetScenarios...)
	}
	return c
}

// WithCoordinatorTargetScenarios records a coordinator-only scenario target list.
func (c *contextState) WithCoordinatorTargetScenarios(targetScenarios ...string) *contextState {
	if c != nil {
		c.validateNames("coordinator target scenarios", targetScenarios)
		c.CoordinatorTargetScenarios = append([]string(nil), targetScenarios...)
	}
	return c
}

// WithScenarioCompletionTimeout records the scenario completion timeout.
func (c *contextState) WithScenarioCompletionTimeout(value TimeSpan) *contextState {
	if c != nil {
		c.ScenarioCompletionTimeoutSeconds = durationArgumentSeconds("scenario completion timeout", value)
	}
	return c
}

// WithClusterCommandTimeout records the cluster command timeout.
func (c *contextState) WithClusterCommandTimeout(value TimeSpan) *contextState {
	if c != nil {
		c.ClusterCommandTimeoutSeconds = durationArgumentSeconds("cluster command timeout", value)
	}
	return c
}

// WithDisplayConsoleMetrics toggles realtime console metric display.
func (c *contextState) WithDisplayConsoleMetrics(enabled bool) *contextState {
	if c != nil {
		c.ConsoleMetricsEnabled = enabled
	}
	return c
}

// DisplayConsoleMetrics mirrors the .NET fluent context method name.
func (c *contextState) DisplayConsoleMetrics(enabled bool) *contextState {
	return c.WithDisplayConsoleMetrics(enabled)
}

// WithReportingSinks records reporting sinks configured for the run.
func (c *contextState) WithReportingSinks(reportingSinks ...LoadStrikeReportingSink) *contextState {
	if c != nil {
		if len(reportingSinks) == 0 {
			c.recordConfigError(fmt.Errorf("at least one reporting sink should be provided"))
		}
		for _, sink := range reportingSinks {
			if sink == nil {
				c.recordConfigError(fmt.Errorf("reporting sink collection cannot contain nil values"))
			}
		}
		c.ReportingSinks = append([]LoadStrikeReportingSink(nil), reportingSinks...)
	}
	return c
}

// WithWorkerPlugins records worker plugins configured for the run.
func (c *contextState) WithWorkerPlugins(workerPlugins ...LoadStrikeWorkerPlugin) *contextState {
	if c != nil {
		if len(workerPlugins) == 0 {
			c.recordConfigError(fmt.Errorf("at least one worker plugin should be provided"))
		}
		for _, plugin := range workerPlugins {
			if plugin == nil {
				c.recordConfigError(fmt.Errorf("worker plugin collection cannot contain nil values"))
			}
		}
		c.WorkerPlugins = append([]LoadStrikeWorkerPlugin(nil), workerPlugins...)
	}
	return c
}

// WithRuntimePolicies records runtime policies configured for the run.
func (c *contextState) WithRuntimePolicies(runtimePolicies ...LoadStrikeRuntimePolicy) *contextState {
	if c != nil {
		if len(runtimePolicies) == 0 {
			c.recordConfigError(fmt.Errorf("at least one runtime policy should be provided"))
		}
		for _, policy := range runtimePolicies {
			if policy == nil {
				c.recordConfigError(fmt.Errorf("runtime policy collection cannot contain nil values"))
			}
		}
		c.RuntimePolicies = append([]LoadStrikeRuntimePolicy(nil), runtimePolicies...)
	}
	return c
}

// WithRuntimePolicyErrorMode sets how policy failures should be handled.
func (c *contextState) WithRuntimePolicyErrorMode(mode RuntimePolicyErrorMode) *contextState {
	if c != nil {
		c.RuntimePolicyErrorMode = mode
	}
	return c
}

// configure mutates the reusable context using a callback.
func (c *contextState) configure(configure func(*contextState) *contextState) *contextState {
	if configure == nil {
		panic("configure callback must be provided")
	}
	if c == nil {
		return c
	}
	if next := configure(c); next != nil {
		*c = *next
	}
	return c
}

// GetNodeInfo returns the populated runtime node info when available.
func (c contextState) GetNodeInfo() nodeInfo {
	return c.nodeInfo
}

// WithReportFolder sets the output folder for generated report files.
func (c *contextState) WithReportFolder(reportFolder string) *contextState {
	if c != nil {
		c.requireNonBlank("report folder", reportFolder)
		c.ReportFolder = reportFolder
		c.reportConfigured = true
	}
	return c
}

// WithReportFileName sets the base file name for generated report files.
func (c *contextState) WithReportFileName(reportFileName string) *contextState {
	if c != nil {
		c.requireNonBlank("report file name", reportFileName)
		c.ReportFileName = reportFileName
		c.reportConfigured = true
	}
	return c
}

// WithReportFormats sets the explicit report formats to generate.
func (c *contextState) WithReportFormats(reportFormats ...ReportFormat) *contextState {
	if c != nil {
		if len(reportFormats) == 0 {
			c.recordConfigError(fmt.Errorf("at least one report format should be provided"))
		}
		for _, format := range reportFormats {
			if strings.TrimSpace(string(format)) == "" {
				c.recordConfigError(fmt.Errorf("report format collection cannot contain empty values"))
			}
		}
		c.ReportFormats = append([]ReportFormat(nil), reportFormats...)
		c.reportConfigured = true
	}
	return c
}

// WithoutReports disables local report generation.
func (c *contextState) WithoutReports() *contextState {
	if c != nil {
		c.ReportsEnabled = false
	}
	return c
}

// WithRestartIterationMaxAttempts sets the maximum number of retries for restartable failed iterations.
func (c *contextState) WithRestartIterationMaxAttempts(attempts int) *contextState {
	if c != nil {
		c.requireNonNegativeInt("restart iteration max attempts", attempts)
		c.RestartIterationMaxAttempts = attempts
	}
	return c
}

// LoadConfig applies supported settings from a loadstrike JSON config file.
func (c *contextState) LoadConfig(path string) *contextState {
	if c == nil {
		return c
	}
	c.requireNonBlank("config path", path)

	config, err := loadRuntimeConfig(path)
	if err != nil {
		c.recordConfigError(err)
	}

	applyRuntimeConfig(c, config)
	return c
}

// LoadInfraConfig records the infra-config file path for sinks and plugins.
func (c *contextState) LoadInfraConfig(path string) *contextState {
	if c != nil {
		c.requireNonBlank("infra config path", path)
		c.InfraConfigPath = path
	}
	return c
}

// Run executes the scenarios registered on this reusable context.
func (c *contextState) Run(args ...string) (runResult, error) {
	if c == nil {
		return runResult{}, fmt.Errorf("context must be provided")
	}
	if len(c.scenarios) == 0 {
		return runResult{}, errNoScenarios
	}
	if err := applyRunArgs(c, args); err != nil {
		return runResult{}, err
	}
	if strings.TrimSpace(c.RunnerKey) == "" {
		return runResult{}, fmt.Errorf("Runner key is required. Call WithRunnerKey(...) before Run().")
	}

	registry := newRuntimeCallbackRegistry()
	defer registry.Close()
	return runViaPrivateRuntime(c, registry)
}

func (c *contextState) recordConfigError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func (c *contextState) requireNonBlank(name string, value string) {
	if strings.TrimSpace(value) == "" {
		c.recordConfigError(fmt.Errorf("%s must be provided", name))
	}
}

func (c *contextState) requirePositiveFloat(name string, value float64) {
	if value <= 0 {
		c.recordConfigError(fmt.Errorf("%s must be greater than zero", name))
	}
}

func (c *contextState) requirePositiveInt(name string, value int) {
	if value <= 0 {
		c.recordConfigError(fmt.Errorf("%s must be greater than zero", name))
	}
}

func (c *contextState) requireNonNegativeInt(name string, value int) {
	if value < 0 {
		c.recordConfigError(fmt.Errorf("%s must be zero or greater", name))
	}
}

func (c *contextState) validateNames(name string, values []string) {
	if len(values) == 0 {
		c.recordConfigError(fmt.Errorf("%s must contain at least one value", name))
		return
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			c.recordConfigError(fmt.Errorf("%s cannot contain blank values", name))
			return
		}
	}
}

func durationArgumentSeconds(name string, value any) float64 {
	switch typed := value.(type) {
	case nil:
		panic(fmt.Sprintf("%s must be provided", name))
	case float64:
		if typed <= 0 {
			panic(fmt.Sprintf("%s must be greater than zero", name))
		}
		return typed
	case float32:
		if typed <= 0 {
			panic(fmt.Sprintf("%s must be greater than zero", name))
		}
		return float64(typed)
	case int:
		if typed <= 0 {
			panic(fmt.Sprintf("%s must be greater than zero", name))
		}
		return float64(typed)
	case int64:
		if typed <= 0 {
			panic(fmt.Sprintf("%s must be greater than zero", name))
		}
		return float64(typed)
	case time.Duration:
		if typed <= 0 {
			panic(fmt.Sprintf("%s must be greater than zero", name))
		}
		return typed.Seconds()
	default:
		panic(fmt.Sprintf("%s must be provided", name))
	}
}

func normalizeLogEventLevel(level any) string {
	switch typed := level.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case LogEventLevel:
		return strings.TrimSpace(string(typed))
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func normalizeLoggerConfiguration(config LoggerConfigurationFactory) LoggerConfiguration {
	if config == nil {
		panic("logger config must be provided")
	}
	typed := config()
	cloned := make(LoggerConfiguration, len(typed))
	for key, value := range typed {
		cloned[key] = value
	}
	return cloned
}

func isWorkerPluginLike(value any) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case LoadStrikeWorkerPlugin, loadStrikeAsyncWorkerPlugin, legacyLoadStrikeWorkerPlugin, workerPlugin, workerPluginSpec, *workerPluginSpec:
		return true
	default:
		return false
	}
}

func isReportingSinkLike(value any) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case LoadStrikeReportingSink, loadStrikeAsyncReportingSink, legacyLoadStrikeReportingSink, reportingSinkSpec, *reportingSinkSpec,
		InfluxDbReportingSink, *InfluxDbReportingSink,
		TimescaleDbReportingSink, *TimescaleDbReportingSink,
		GrafanaLokiReportingSink, *GrafanaLokiReportingSink,
		DatadogReportingSink, *DatadogReportingSink,
		SplunkReportingSink, *SplunkReportingSink,
		OtelCollectorReportingSink, *OtelCollectorReportingSink:
		return true
	default:
		return false
	}
}

func isRuntimePolicyLike(value any) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case LoadStrikeRuntimePolicy,
		loadStrikeAsyncRuntimePolicy,
		legacyLoadStrikeRuntimePolicy,
		runtimePolicy,
		loadStrikeTaskRuntimePolicyShouldRun,
		loadStrikeTaskRuntimePolicyBeforeScenario,
		loadStrikeTaskRuntimePolicyAfterScenario,
		loadStrikeTaskRuntimePolicyBeforeStep,
		loadStrikeTaskRuntimePolicyAfterStep:
		return true
	default:
		return false
	}
}
