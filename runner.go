package loadstrike

import (
	"errors"
	"strings"
)

var errNoScenarios = errors.New("At least one scenario must be added before building the runner context.")

// Runner composes scenarios and execution settings before a run.
type runnerState struct {
	scenarios []scenarioDefinition
	context   contextState
}

// newRunnerState creates a new internal runner state with default execution settings.
func newRunnerState() *runnerState {
	return &runnerState{
		context: contextState{
			ReportsEnabled:                  true,
			RestartIterationMaxAttempts:     3,
			LicenseValidationTimeoutSeconds: defaultLicenseValidationTimeoutSeconds,
			NodeType:                        NodeTypeSingleNode,
			AgentsCount:                     1,
			RuntimePolicyErrorMode:          RuntimePolicyErrorModeFail,
			ConsoleMetricsEnabled:           true,
			ReportFormats: []ReportFormat{
				ReportFormatHTML,
				ReportFormatTXT,
				ReportFormatCSV,
				ReportFormatMD,
			},
		},
	}
}

// NewRunner creates a new runner with default execution settings.
func NewRunner() LoadStrikeRunner {
	return wrapLoadStrikeRunner(newRunnerState())
}

// WithLicenseValidationTimeout sets the licensing validation timeout.
func (r *runnerState) WithLicenseValidationTimeout(value TimeSpan) *runnerState {
	r.context.WithLicenseValidationTimeout(value)
	return r
}

// RegisterScenarios creates a runner and registers the provided scenarios.
func RegisterScenarios(scenarios ...LoadStrikeScenario) LoadStrikeContext {
	if len(scenarios) == 0 {
		panic("At least one scenario must be provided.")
	}
	runner := newRunnerState()
	for _, scenario := range scenarios {
		runner.addScenarioNative(scenario.nativeValue())
	}
	return wrapLoadStrikeContext(runner.BuildContext())
}

// AddScenario registers one scenario on the runner.
func (r *runnerState) AddScenario(scenario scenarioLike) *runnerState {
	nativeScenario := normalizeScenarioLike(scenario)
	validateScenarioValue(nativeScenario)
	r.scenarios = append(r.scenarios, nativeScenario)
	return r
}

// AddScenarios registers multiple scenarios on the runner.
func (r *runnerState) AddScenarios(scenarios ...scenarioLike) *runnerState {
	if len(scenarios) == 0 {
		panic("At least one scenario must be provided.")
	}
	for _, scenario := range scenarios {
		nativeScenario := normalizeScenarioLike(scenario)
		validateScenarioValue(nativeScenario)
		r.scenarios = append(r.scenarios, nativeScenario)
	}
	return r
}

// addScenarioNative registers one already-normalized native scenario on the runner.
func (r *runnerState) addScenarioNative(scenario scenarioDefinition) *runnerState {
	validateScenarioValue(scenario)
	r.scenarios = append(r.scenarios, scenario)
	return r
}

// WithTestSuite sets the run test suite label.
func (r *runnerState) WithTestSuite(testSuite string) *runnerState {
	r.context.WithTestSuite(testSuite)
	return r
}

// WithTestName sets the run test name label.
func (r *runnerState) WithTestName(testName string) *runnerState {
	r.context.WithTestName(testName)
	return r
}

// WithReportingInterval sets the realtime reporting cadence.
func (r *runnerState) WithReportingInterval(value TimeSpan) *runnerState {
	r.context.WithReportingInterval(value)
	return r
}

// WithMinimumLogLevel records the minimum log level for this run.
func (r *runnerState) WithMinimumLogLevel(level LogEventLevel) *runnerState {
	r.context.WithMinimumLogLevel(level)
	return r
}

// WithLoggerConfig records logger configuration overrides for this run.
func (r *runnerState) WithLoggerConfig(config LoggerConfigurationFactory) *runnerState {
	r.context.WithLoggerConfig(config)
	return r
}

// WithRunnerKey records the runner key used for this run.
func (r *runnerState) WithRunnerKey(runnerKey string) *runnerState {
	r.context.WithRunnerKey(runnerKey)
	return r
}

// WithSessionId records the session identifier used for this run.
func (r *runnerState) WithSessionId(sessionID string) *runnerState {
	r.context.WithSessionId(sessionID)
	return r
}

// WithClusterId records the cluster identifier used for this run.
func (r *runnerState) WithClusterId(clusterID string) *runnerState {
	r.context.WithClusterId(clusterID)
	return r
}

// WithAgentGroup records the agent group used for this run.
func (r *runnerState) WithAgentGroup(agentGroup string) *runnerState {
	r.context.WithAgentGroup(agentGroup)
	return r
}

// WithNatsServerURL records the NATS server URL used for distributed cluster mode.
func (r *runnerState) WithNatsServerURL(serverURL string) *runnerState {
	r.context.WithNatsServerURL(serverURL)
	return r
}

// WithNatsServerUrl mirrors the exact .NET fluent method name.
func (r *runnerState) WithNatsServerUrl(serverURL string) *runnerState {
	return r.WithNatsServerURL(serverURL)
}

// WithNodeType sets the runtime node type.
func (r *runnerState) WithNodeType(nodeType NodeType) *runnerState {
	r.context.WithNodeType(nodeType)
	return r
}

// WithAgentsCount sets the requested agent count for local or distributed runs.
func (r *runnerState) WithAgentsCount(agentsCount int) *runnerState {
	r.context.WithAgentsCount(agentsCount)
	return r
}

// WithTargetScenarios records a shared scenario target list.
func (r *runnerState) WithTargetScenarios(targetScenarios ...string) *runnerState {
	r.context.WithTargetScenarios(targetScenarios...)
	return r
}

// WithAgentTargetScenarios records an agent-only scenario target list.
func (r *runnerState) WithAgentTargetScenarios(targetScenarios ...string) *runnerState {
	r.context.WithAgentTargetScenarios(targetScenarios...)
	return r
}

// WithCoordinatorTargetScenarios records a coordinator-only scenario target list.
func (r *runnerState) WithCoordinatorTargetScenarios(targetScenarios ...string) *runnerState {
	r.context.WithCoordinatorTargetScenarios(targetScenarios...)
	return r
}

// WithScenarioCompletionTimeout records the scenario completion timeout.
func (r *runnerState) WithScenarioCompletionTimeout(value TimeSpan) *runnerState {
	r.context.WithScenarioCompletionTimeout(value)
	return r
}

// WithClusterCommandTimeout records the cluster command timeout.
func (r *runnerState) WithClusterCommandTimeout(value TimeSpan) *runnerState {
	r.context.WithClusterCommandTimeout(value)
	return r
}

// WithDisplayConsoleMetrics toggles realtime console metric display.
func (r *runnerState) WithDisplayConsoleMetrics(enabled bool) *runnerState {
	r.context.WithDisplayConsoleMetrics(enabled)
	return r
}

// DisplayConsoleMetrics mirrors the .NET fluent runner method name.
func (r *runnerState) DisplayConsoleMetrics(enabled bool) *runnerState {
	return r.WithDisplayConsoleMetrics(enabled)
}

// EnableLocalDevCluster toggles local in-process cluster execution mode.
func (r *runnerState) EnableLocalDevCluster(enable bool) *runnerState {
	r.context.EnableLocalDevCluster(enable)
	return r
}

// WithReportingSinks records reporting sinks configured for the run.
func (r *runnerState) WithReportingSinks(reportingSinks ...LoadStrikeReportingSink) *runnerState {
	r.context.WithReportingSinks(reportingSinks...)
	return r
}

// WithWorkerPlugins records worker plugins configured for the run.
func (r *runnerState) WithWorkerPlugins(workerPlugins ...LoadStrikeWorkerPlugin) *runnerState {
	r.context.WithWorkerPlugins(workerPlugins...)
	return r
}

// WithRuntimePolicies records runtime policies configured for the run.
func (r *runnerState) WithRuntimePolicies(runtimePolicies ...LoadStrikeRuntimePolicy) *runnerState {
	r.context.WithRuntimePolicies(runtimePolicies...)
	return r
}

// WithRuntimePolicyErrorMode sets how policy failures should be handled.
func (r *runnerState) WithRuntimePolicyErrorMode(mode RuntimePolicyErrorMode) *runnerState {
	r.context.WithRuntimePolicyErrorMode(mode)
	return r
}

type contextConfigurator func(LoadStrikeContext) LoadStrikeContext

// Configure mutates the runner context using a callback.
func (r *runnerState) Configure(configure contextConfigurator) *runnerState {
	if configure == nil {
		panic("configure callback must be provided")
	}
	next := configure(wrapLoadStrikeContext(&r.context))
	if next.native != nil {
		native := *requireNativeContext(next.nativeValue())
		if len(native.scenarios) > 0 {
			r.scenarios = append([]scenarioDefinition(nil), native.scenarios...)
		}
		native.scenarios = nil
		r.context = native
	}
	return r
}

// ConfigureContext mirrors the .NET runner helper that merges an existing context.
func (r *runnerState) ConfigureContext(context LoadStrikeContext) *runnerState {
	native := *requireNativeContext(context.nativeValue())
	if len(native.scenarios) > 0 {
		r.scenarios = append([]scenarioDefinition(nil), native.scenarios...)
	}
	native.scenarios = nil
	r.context = native
	return r
}

// WithoutReports disables local report generation.
func (r *runnerState) WithoutReports() *runnerState {
	r.context.ReportsEnabled = false
	return r
}

// WithReportFolder sets the output folder for generated report files.
func (r *runnerState) WithReportFolder(reportFolder string) *runnerState {
	r.context.WithReportFolder(reportFolder)
	return r
}

// WithReportFileName sets the base file name for generated report files.
func (r *runnerState) WithReportFileName(reportFileName string) *runnerState {
	r.context.WithReportFileName(reportFileName)
	return r
}

// WithReportFormats sets the explicit report formats to generate.
func (r *runnerState) WithReportFormats(reportFormats ...ReportFormat) *runnerState {
	r.context.WithReportFormats(reportFormats...)
	return r
}

// WithRestartIterationMaxAttempts sets the maximum number of retries for restartable failed iterations.
func (r *runnerState) WithRestartIterationMaxAttempts(attempts int) *runnerState {
	r.context.WithRestartIterationMaxAttempts(attempts)
	return r
}

// BuildContext returns the configured reusable execution context.
func (r *runnerState) BuildContext() *contextState {
	if len(r.scenarios) == 0 {
		panic(errNoScenarios.Error())
	}

	context := r.context
	context.scenarios = append([]scenarioDefinition(nil), r.scenarios...)
	return &context
}

func validateScenarioValue(scenario scenarioDefinition) {
	if strings.TrimSpace(scenario.Name) == "" {
		panic("scenario name must be provided.")
	}
}

// Run executes the registered scenarios.
func (r *runnerState) Run(args ...string) (runResult, error) {
	return r.BuildContext().Run(args...)
}
