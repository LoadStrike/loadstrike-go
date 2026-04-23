package loadstrike

func wrapLoadStrikeContext(context *contextState) LoadStrikeContext {
	if context == nil {
		return loadStrikeContext{}
	}
	return loadStrikeContext{native: context}
}

func wrapLoadStrikeRunner(runner *runnerState) LoadStrikeRunner {
	if runner == nil {
		return loadStrikeRunner{}
	}
	return loadStrikeRunner{native: runner}
}

func requireNativeContext(context *contextState) *contextState {
	if context == nil {
		panic("context must be provided")
	}
	return context
}

func (c loadStrikeContext) nativeValue() *contextState {
	return c.native
}

func (r loadStrikeRunner) nativeValue() *runnerState {
	return r.native
}

// Create mirrors the .NET type-level runner constructor in the closest valid Go form.
func (loadStrikeRunner) Create() LoadStrikeRunner {
	return NewRunner()
}

// RegisterScenarios mirrors the .NET type-level registration helper in the closest valid Go form.
func (loadStrikeRunner) RegisterScenarios(scenarios ...LoadStrikeScenario) LoadStrikeContext {
	return RegisterScenarios(scenarios...)
}

// Executes the configured workload through the SDK runtime.
// Use this when configuration is complete and you are ready to start the run.
func (c loadStrikeContext) Run(args ...string) LoadStrikeRunResult {
	result, err := requireNativeContext(c.nativeValue()).Run(args...)
	if err != nil {
		panic(err)
	}
	return newLoadStrikeRunResult(result)
}

// Toggles realtime console metric output.
// Use this when a local run or CI log should stream live throughput and latency updates.
func (c loadStrikeContext) DisplayConsoleMetrics(enable bool) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).DisplayConsoleMetrics(enable)
	return c
}

// Toggles local development cluster mode.
// Use this when you want to simulate coordinator and agent behavior on a single machine.
func (c loadStrikeContext) EnableLocalDevCluster(enable bool) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).EnableLocalDevCluster(enable)
	return c
}

// Loads run settings from a JSON config file.
// Use this when configuration should come from a checked-in or generated config document.
func (c loadStrikeContext) LoadConfig(path string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).LoadConfig(path)
	return c
}

// Loads infrastructure settings for sinks, plugins, or transport dependencies.
// Use this when runtime integrations depend on external connection details.
func (c loadStrikeContext) LoadInfraConfig(path string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).LoadInfraConfig(path)
	return c
}

// Sets the test suite label for the run.
// Use this when related tests should be grouped under a suite name.
func (c loadStrikeContext) WithTestSuite(testSuite string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithTestSuite(testSuite)
	return c
}

// Sets the runner key used for licensing validation.
// Use this when the run must authenticate against the licensing API before it starts.
func (c loadStrikeContext) WithRunnerKey(runnerKey string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRunnerKey(runnerKey)
	return c
}

// Sets the test name label for the run.
// Use this when reports and logs should expose a human-readable test identifier.
func (c loadStrikeContext) WithTestName(testName string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithTestName(testName)
	return c
}

// Sets the minimum runtime log level.
// Use this when the run should emit less or more operational detail.
func (c loadStrikeContext) WithMinimumLogLevel(level LogEventLevel) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithMinimumLogLevel(level)
	return c
}

// Supplies a custom logger configuration factory.
// Use this when the default runtime logging output is not enough for your environment.
func (c loadStrikeContext) WithLoggerConfig(config LoggerConfigurationFactory) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithLoggerConfig(config)
	return c
}

// Sets the session identifier for this run.
// Use this when downstream logs, reports, or external systems need a stable session id.
func (c loadStrikeContext) WithSessionId(sessionID string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithSessionId(sessionID)
	return c
}

// Sets the cluster identifier for this run.
// Use this when multiple cluster nodes must join the same distributed execution.
func (c loadStrikeContext) WithClusterId(clusterID string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithClusterId(clusterID)
	return c
}

// Sets the agent group for this node or run.
// Use this when only a subset of agents should pick up a particular workload.
func (c loadStrikeContext) WithAgentGroup(agentGroup string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithAgentGroup(agentGroup)
	return c
}

// Sets the NATS server URL used for distributed cluster coordination.
// Use this when coordinator and agent nodes communicate over NATS.
func (c loadStrikeContext) WithNatsServerUrl(serverURL string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithNatsServerUrl(serverURL)
	return c
}

// Sets the runtime node type.
// Use this when a process should behave as a single node, coordinator, or agent.
func (c loadStrikeContext) WithNodeType(nodeType NodeType) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithNodeType(nodeType)
	return c
}

// Sets the requested agent count.
// Use this when a coordinator should fan work out across a specific number of agents.
func (c loadStrikeContext) WithAgentsCount(agentsCount int) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithAgentsCount(agentsCount)
	return c
}

// Targets a shared set of scenarios.
// Use this when only selected scenarios should execute for the current node or run.
func (c loadStrikeContext) WithTargetScenarios(scenarios ...string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithTargetScenarios(scenarios...)
	return c
}

// Targets scenarios for agent nodes only.
// Use this when agents should execute only a subset of the registered scenarios.
func (c loadStrikeContext) WithAgentTargetScenarios(scenarios ...string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithAgentTargetScenarios(scenarios...)
	return c
}

// Targets scenarios for the coordinator only.
// Use this when orchestration or control-node work should run on the coordinator itself.
func (c loadStrikeContext) WithCoordinatorTargetScenarios(scenarios ...string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithCoordinatorTargetScenarios(scenarios...)
	return c
}

// Sets the output folder for generated reports.
// Use this when report files should land in a specific directory.
func (c loadStrikeContext) WithReportFolder(reportFolder string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportFolder(reportFolder)
	return c
}

// Sets the base file name for generated reports.
// Use this when exported reports should use a predictable naming convention.
func (c loadStrikeContext) WithReportFileName(reportFileName string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportFileName(reportFileName)
	return c
}

// Restricts generated reports to specific formats.
// Use this when you want only selected report artifacts such as HTML, CSV, or Markdown.
func (c loadStrikeContext) WithReportFormats(reportFormats ...ReportFormat) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportFormats(reportFormats...)
	return c
}

// Sets the realtime reporting cadence.
// Use this when sinks or dashboards should receive updates at a controlled interval.
func (c loadStrikeContext) WithReportingInterval(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportingInterval(value)
	return c
}

// Sets how long the SDK waits for licensing validation to complete.
// Use this when control-plane latency or private networking requires a longer validation window.
func (c loadStrikeContext) WithLicenseValidationTimeout(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithLicenseValidationTimeout(value)
	return c
}

// Registers runtime policies for the run.
// Use this when scenario selection or step execution should obey policy callbacks.
func (c loadStrikeContext) WithRuntimePolicies(runtimePolicies ...LoadStrikeRuntimePolicy) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRuntimePolicies(runtimePolicies...)
	return c
}

// Registers one or more reporting sinks.
// Use this when run results must be pushed to external observability or storage systems.
func (c loadStrikeContext) WithReportingSinks(reportingSinks ...LoadStrikeReportingSink) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportingSinks(reportingSinks...)
	return c
}

// Registers worker plugins for the run.
// Use this when custom plugin data should be collected alongside run results.
func (c loadStrikeContext) WithWorkerPlugins(workerPlugins ...LoadStrikeWorkerPlugin) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithWorkerPlugins(workerPlugins...)
	return c
}

// Sets how runtime policy failures are handled.
// Use this when policy errors should either fail fast or be tolerated.
func (c loadStrikeContext) WithRuntimePolicyErrorMode(mode RuntimePolicyErrorMode) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRuntimePolicyErrorMode(mode)
	return c
}

// Sets the timeout for scenario completion.
// Use this when long-running scenarios need an explicit completion window.
func (c loadStrikeContext) WithScenarioCompletionTimeout(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithScenarioCompletionTimeout(value)
	return c
}

// Sets the timeout for cluster command round-trips.
// Use this when distributed control messages need a tighter or looser deadline.
func (c loadStrikeContext) WithClusterCommandTimeout(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithClusterCommandTimeout(value)
	return c
}

// Sets the retry limit for restartable failed iterations.
// Use this when transient failures should be retried a fixed number of times.
func (c loadStrikeContext) WithRestartIterationMaxAttempts(attempts int) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRestartIterationMaxAttempts(attempts)
	return c
}

// Disables local report generation.
// Use this when reports are unnecessary or should be handled exclusively by sinks.
func (c loadStrikeContext) WithoutReports() LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithoutReports()
	return c
}

// Adds a scenario to the current runner.
// Use this when the run should include one more scenario definition.
func (r loadStrikeRunner) AddScenario(scenario LoadStrikeScenario) LoadStrikeRunner {
	requireRunner(r.nativeValue()).addScenarioNative(scenario.nativeValue())
	return r
}

// Adds multiple scenarios to the current runner.
// Use this when the run should execute a batch of scenario definitions together.
func (r loadStrikeRunner) AddScenarios(scenarios ...LoadStrikeScenario) LoadStrikeRunner {
	if len(scenarios) == 0 {
		panic("At least one scenario must be provided.")
	}
	for _, scenario := range scenarios {
		requireRunner(r.nativeValue()).addScenarioNative(scenario.nativeValue())
	}
	return r
}

// Applies a grouped configuration change to the current builder or context.
// Use this when several related settings should be supplied in one step.
func (r loadStrikeRunner) Configure(configure func(LoadStrikeContext) LoadStrikeContext) LoadStrikeRunner {
	if configure == nil {
		panic("configure callback must be provided")
	}
	requireRunner(r.nativeValue()).Configure(configure)
	return r
}

// Copies settings from an existing context snapshot.
// Use this when a runner should inherit configuration that was already assembled elsewhere.
func (r loadStrikeRunner) ConfigureContext(context LoadStrikeContext) LoadStrikeRunner {
	requireRunner(r.nativeValue()).ConfigureContext(context)
	return r
}

// Sets the test suite label for the run.
// Use this when related tests should be grouped under a suite name.
func (r loadStrikeRunner) WithTestSuite(testSuite string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithTestSuite(testSuite)
	return r
}

// Sets the runner key used for licensing validation.
// Use this when the run must authenticate against the licensing API before it starts.
func (r loadStrikeRunner) WithRunnerKey(runnerKey string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRunnerKey(runnerKey)
	return r
}

// Sets the test name label for the run.
// Use this when reports and logs should expose a human-readable test identifier.
func (r loadStrikeRunner) WithTestName(testName string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithTestName(testName)
	return r
}

// Sets the minimum runtime log level.
// Use this when the run should emit less or more operational detail.
func (r loadStrikeRunner) WithMinimumLogLevel(level LogEventLevel) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithMinimumLogLevel(level)
	return r
}

// Supplies a custom logger configuration factory.
// Use this when the default runtime logging output is not enough for your environment.
func (r loadStrikeRunner) WithLoggerConfig(config LoggerConfigurationFactory) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithLoggerConfig(config)
	return r
}

// Sets the session identifier for this run.
// Use this when downstream logs, reports, or external systems need a stable session id.
func (r loadStrikeRunner) WithSessionId(sessionID string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithSessionId(sessionID)
	return r
}

// Sets the cluster identifier for this run.
// Use this when multiple cluster nodes must join the same distributed execution.
func (r loadStrikeRunner) WithClusterId(clusterID string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithClusterId(clusterID)
	return r
}

// Sets the agent group for this node or run.
// Use this when only a subset of agents should pick up a particular workload.
func (r loadStrikeRunner) WithAgentGroup(agentGroup string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithAgentGroup(agentGroup)
	return r
}

// Toggles local development cluster mode.
// Use this when you want to simulate coordinator and agent behavior on a single machine.
func (r loadStrikeRunner) EnableLocalDevCluster(enable bool) LoadStrikeRunner {
	requireRunner(r.nativeValue()).EnableLocalDevCluster(enable)
	return r
}

// Sets the NATS server URL used for distributed cluster coordination.
// Use this when coordinator and agent nodes communicate over NATS.
func (r loadStrikeRunner) WithNatsServerURL(serverURL string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithNatsServerURL(serverURL)
	return r
}

// Sets the NATS server URL used for distributed cluster coordination.
// Use this when coordinator and agent nodes communicate over NATS.
func (r loadStrikeRunner) WithNatsServerUrl(serverURL string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithNatsServerUrl(serverURL)
	return r
}

// Sets the runtime node type.
// Use this when a process should behave as a single node, coordinator, or agent.
func (r loadStrikeRunner) WithNodeType(nodeType NodeType) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithNodeType(nodeType)
	return r
}

// Sets the requested agent count.
// Use this when a coordinator should fan work out across a specific number of agents.
func (r loadStrikeRunner) WithAgentsCount(agentsCount int) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithAgentsCount(agentsCount)
	return r
}

// Targets a shared set of scenarios.
// Use this when only selected scenarios should execute for the current node or run.
func (r loadStrikeRunner) WithTargetScenarios(scenarios ...string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithTargetScenarios(scenarios...)
	return r
}

// Targets scenarios for agent nodes only.
// Use this when agents should execute only a subset of the registered scenarios.
func (r loadStrikeRunner) WithAgentTargetScenarios(scenarios ...string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithAgentTargetScenarios(scenarios...)
	return r
}

// Targets scenarios for the coordinator only.
// Use this when orchestration or control-node work should run on the coordinator itself.
func (r loadStrikeRunner) WithCoordinatorTargetScenarios(scenarios ...string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithCoordinatorTargetScenarios(scenarios...)
	return r
}

// Sets the output folder for generated reports.
// Use this when report files should land in a specific directory.
func (r loadStrikeRunner) WithReportFolder(reportFolder string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportFolder(reportFolder)
	return r
}

// Sets the base file name for generated reports.
// Use this when exported reports should use a predictable naming convention.
func (r loadStrikeRunner) WithReportFileName(reportFileName string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportFileName(reportFileName)
	return r
}

// Restricts generated reports to specific formats.
// Use this when you want only selected report artifacts such as HTML, CSV, or Markdown.
func (r loadStrikeRunner) WithReportFormats(reportFormats ...ReportFormat) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportFormats(reportFormats...)
	return r
}

// Sets the realtime reporting cadence.
// Use this when sinks or dashboards should receive updates at a controlled interval.
func (r loadStrikeRunner) WithReportingInterval(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportingInterval(value)
	return r
}

// Sets how long the SDK waits for licensing validation to complete.
// Use this when control-plane latency or private networking requires a longer validation window.
func (r loadStrikeRunner) WithLicenseValidationTimeout(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithLicenseValidationTimeout(value)
	return r
}

// Registers one or more reporting sinks.
// Use this when run results must be pushed to external observability or storage systems.
func (r loadStrikeRunner) WithReportingSinks(reportingSinks ...LoadStrikeReportingSink) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportingSinks(reportingSinks...)
	return r
}

// Registers worker plugins for the run.
// Use this when custom plugin data should be collected alongside run results.
func (r loadStrikeRunner) WithWorkerPlugins(workerPlugins ...LoadStrikeWorkerPlugin) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithWorkerPlugins(workerPlugins...)
	return r
}

// Registers runtime policies for the run.
// Use this when scenario selection or step execution should obey policy callbacks.
func (r loadStrikeRunner) WithRuntimePolicies(runtimePolicies ...LoadStrikeRuntimePolicy) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRuntimePolicies(runtimePolicies...)
	return r
}

// Sets how runtime policy failures are handled.
// Use this when policy errors should either fail fast or be tolerated.
func (r loadStrikeRunner) WithRuntimePolicyErrorMode(mode RuntimePolicyErrorMode) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRuntimePolicyErrorMode(mode)
	return r
}

// Sets the timeout for cluster command round-trips.
// Use this when distributed control messages need a tighter or looser deadline.
func (r loadStrikeRunner) WithClusterCommandTimeout(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithClusterCommandTimeout(value)
	return r
}

// Sets the timeout for scenario completion.
// Use this when long-running scenarios need an explicit completion window.
func (r loadStrikeRunner) WithScenarioCompletionTimeout(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithScenarioCompletionTimeout(value)
	return r
}

// Configures display console metrics for this SDK object.
// Use this when display console metrics should be set explicitly before the run starts.
func (r loadStrikeRunner) WithDisplayConsoleMetrics(enable bool) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithDisplayConsoleMetrics(enable)
	return r
}

// Sets the retry limit for restartable failed iterations.
// Use this when transient failures should be retried a fixed number of times.
func (r loadStrikeRunner) WithRestartIterationMaxAttempts(attempts int) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRestartIterationMaxAttempts(attempts)
	return r
}

// Loads run settings from a JSON config file.
// Use this when configuration should come from a checked-in or generated config document.
func (r loadStrikeRunner) LoadConfig(path string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).LoadConfig(path)
	return r
}

// Loads infrastructure settings for sinks, plugins, or transport dependencies.
// Use this when runtime integrations depend on external connection details.
func (r loadStrikeRunner) LoadInfraConfig(path string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).LoadInfraConfig(path)
	return r
}

// Disables local report generation.
// Use this when reports are unnecessary or should be handled exclusively by sinks.
func (r loadStrikeRunner) WithoutReports() LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithoutReports()
	return r
}

// Builds the current configuration into a runnable context.
// Use this when you want to inspect or reuse the final run settings before execution.
func (r loadStrikeRunner) BuildContext() LoadStrikeContext {
	return wrapLoadStrikeContext(requireRunner(r.nativeValue()).BuildContext())
}

// Executes the configured workload through the SDK runtime.
// Use this when configuration is complete and you are ready to start the run.
func (r loadStrikeRunner) Run(args ...string) LoadStrikeRunResult {
	result, err := requireRunner(r.nativeValue()).Run(args...)
	if err != nil {
		panic(err)
	}
	return newLoadStrikeRunResult(result)
}

func requireRunner(runner *runnerState) *runnerState {
	if runner == nil {
		panic("runner must be provided")
	}
	return runner
}
