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

func (c loadStrikeContext) Run(args ...string) LoadStrikeRunResult {
	result, err := requireNativeContext(c.nativeValue()).Run(args...)
	if err != nil {
		panic(err)
	}
	return newLoadStrikeRunResult(result)
}

func (c loadStrikeContext) DisplayConsoleMetrics(enable bool) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).DisplayConsoleMetrics(enable)
	return c
}

func (c loadStrikeContext) EnableLocalDevCluster(enable bool) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).EnableLocalDevCluster(enable)
	return c
}

func (c loadStrikeContext) LoadConfig(path string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).LoadConfig(path)
	return c
}

func (c loadStrikeContext) LoadInfraConfig(path string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).LoadInfraConfig(path)
	return c
}

func (c loadStrikeContext) WithTestSuite(testSuite string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithTestSuite(testSuite)
	return c
}

func (c loadStrikeContext) WithRunnerKey(runnerKey string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRunnerKey(runnerKey)
	return c
}

func (c loadStrikeContext) WithTestName(testName string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithTestName(testName)
	return c
}

func (c loadStrikeContext) WithMinimumLogLevel(level LogEventLevel) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithMinimumLogLevel(level)
	return c
}

func (c loadStrikeContext) WithLoggerConfig(config LoggerConfigurationFactory) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithLoggerConfig(config)
	return c
}

func (c loadStrikeContext) WithSessionId(sessionID string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithSessionId(sessionID)
	return c
}

func (c loadStrikeContext) WithClusterId(clusterID string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithClusterId(clusterID)
	return c
}

func (c loadStrikeContext) WithAgentGroup(agentGroup string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithAgentGroup(agentGroup)
	return c
}

func (c loadStrikeContext) WithNatsServerUrl(serverURL string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithNatsServerUrl(serverURL)
	return c
}

func (c loadStrikeContext) WithNodeType(nodeType NodeType) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithNodeType(nodeType)
	return c
}

func (c loadStrikeContext) WithAgentsCount(agentsCount int) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithAgentsCount(agentsCount)
	return c
}

func (c loadStrikeContext) WithTargetScenarios(scenarios ...string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithTargetScenarios(scenarios...)
	return c
}

func (c loadStrikeContext) WithAgentTargetScenarios(scenarios ...string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithAgentTargetScenarios(scenarios...)
	return c
}

func (c loadStrikeContext) WithCoordinatorTargetScenarios(scenarios ...string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithCoordinatorTargetScenarios(scenarios...)
	return c
}

func (c loadStrikeContext) WithReportFolder(reportFolder string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportFolder(reportFolder)
	return c
}

func (c loadStrikeContext) WithReportFileName(reportFileName string) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportFileName(reportFileName)
	return c
}

func (c loadStrikeContext) WithReportFormats(reportFormats ...ReportFormat) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportFormats(reportFormats...)
	return c
}

func (c loadStrikeContext) WithReportingInterval(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportingInterval(value)
	return c
}

func (c loadStrikeContext) WithLicenseValidationTimeout(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithLicenseValidationTimeout(value)
	return c
}

func (c loadStrikeContext) WithRuntimePolicies(runtimePolicies ...LoadStrikeRuntimePolicy) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRuntimePolicies(runtimePolicies...)
	return c
}

func (c loadStrikeContext) WithReportingSinks(reportingSinks ...LoadStrikeReportingSink) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithReportingSinks(reportingSinks...)
	return c
}

func (c loadStrikeContext) WithWorkerPlugins(workerPlugins ...LoadStrikeWorkerPlugin) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithWorkerPlugins(workerPlugins...)
	return c
}

func (c loadStrikeContext) WithRuntimePolicyErrorMode(mode RuntimePolicyErrorMode) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRuntimePolicyErrorMode(mode)
	return c
}

func (c loadStrikeContext) WithScenarioCompletionTimeout(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithScenarioCompletionTimeout(value)
	return c
}

func (c loadStrikeContext) WithClusterCommandTimeout(value TimeSpan) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithClusterCommandTimeout(value)
	return c
}

func (c loadStrikeContext) WithRestartIterationMaxAttempts(attempts int) LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithRestartIterationMaxAttempts(attempts)
	return c
}

func (c loadStrikeContext) WithoutReports() LoadStrikeContext {
	requireNativeContext(c.nativeValue()).WithoutReports()
	return c
}

func (r loadStrikeRunner) AddScenario(scenario LoadStrikeScenario) LoadStrikeRunner {
	requireRunner(r.nativeValue()).addScenarioNative(scenario.nativeValue())
	return r
}

func (r loadStrikeRunner) AddScenarios(scenarios ...LoadStrikeScenario) LoadStrikeRunner {
	if len(scenarios) == 0 {
		panic("At least one scenario must be provided.")
	}
	for _, scenario := range scenarios {
		requireRunner(r.nativeValue()).addScenarioNative(scenario.nativeValue())
	}
	return r
}

func (r loadStrikeRunner) Configure(configure func(LoadStrikeContext) LoadStrikeContext) LoadStrikeRunner {
	if configure == nil {
		panic("configure callback must be provided")
	}
	requireRunner(r.nativeValue()).Configure(configure)
	return r
}

func (r loadStrikeRunner) ConfigureContext(context LoadStrikeContext) LoadStrikeRunner {
	requireRunner(r.nativeValue()).ConfigureContext(context)
	return r
}

func (r loadStrikeRunner) WithTestSuite(testSuite string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithTestSuite(testSuite)
	return r
}

func (r loadStrikeRunner) WithRunnerKey(runnerKey string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRunnerKey(runnerKey)
	return r
}

func (r loadStrikeRunner) WithTestName(testName string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithTestName(testName)
	return r
}

func (r loadStrikeRunner) WithMinimumLogLevel(level LogEventLevel) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithMinimumLogLevel(level)
	return r
}

func (r loadStrikeRunner) WithLoggerConfig(config LoggerConfigurationFactory) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithLoggerConfig(config)
	return r
}

func (r loadStrikeRunner) WithSessionId(sessionID string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithSessionId(sessionID)
	return r
}

func (r loadStrikeRunner) WithClusterId(clusterID string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithClusterId(clusterID)
	return r
}

func (r loadStrikeRunner) WithAgentGroup(agentGroup string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithAgentGroup(agentGroup)
	return r
}

func (r loadStrikeRunner) EnableLocalDevCluster(enable bool) LoadStrikeRunner {
	requireRunner(r.nativeValue()).EnableLocalDevCluster(enable)
	return r
}

func (r loadStrikeRunner) WithNatsServerURL(serverURL string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithNatsServerURL(serverURL)
	return r
}

func (r loadStrikeRunner) WithNatsServerUrl(serverURL string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithNatsServerUrl(serverURL)
	return r
}

func (r loadStrikeRunner) WithNodeType(nodeType NodeType) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithNodeType(nodeType)
	return r
}

func (r loadStrikeRunner) WithAgentsCount(agentsCount int) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithAgentsCount(agentsCount)
	return r
}

func (r loadStrikeRunner) WithTargetScenarios(scenarios ...string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithTargetScenarios(scenarios...)
	return r
}

func (r loadStrikeRunner) WithAgentTargetScenarios(scenarios ...string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithAgentTargetScenarios(scenarios...)
	return r
}

func (r loadStrikeRunner) WithCoordinatorTargetScenarios(scenarios ...string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithCoordinatorTargetScenarios(scenarios...)
	return r
}

func (r loadStrikeRunner) WithReportFolder(reportFolder string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportFolder(reportFolder)
	return r
}

func (r loadStrikeRunner) WithReportFileName(reportFileName string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportFileName(reportFileName)
	return r
}

func (r loadStrikeRunner) WithReportFormats(reportFormats ...ReportFormat) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportFormats(reportFormats...)
	return r
}

func (r loadStrikeRunner) WithReportingInterval(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportingInterval(value)
	return r
}

func (r loadStrikeRunner) WithLicenseValidationTimeout(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithLicenseValidationTimeout(value)
	return r
}

func (r loadStrikeRunner) WithReportingSinks(reportingSinks ...LoadStrikeReportingSink) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithReportingSinks(reportingSinks...)
	return r
}

func (r loadStrikeRunner) WithWorkerPlugins(workerPlugins ...LoadStrikeWorkerPlugin) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithWorkerPlugins(workerPlugins...)
	return r
}

func (r loadStrikeRunner) WithRuntimePolicies(runtimePolicies ...LoadStrikeRuntimePolicy) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRuntimePolicies(runtimePolicies...)
	return r
}

func (r loadStrikeRunner) WithRuntimePolicyErrorMode(mode RuntimePolicyErrorMode) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRuntimePolicyErrorMode(mode)
	return r
}

func (r loadStrikeRunner) WithClusterCommandTimeout(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithClusterCommandTimeout(value)
	return r
}

func (r loadStrikeRunner) WithScenarioCompletionTimeout(value TimeSpan) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithScenarioCompletionTimeout(value)
	return r
}

func (r loadStrikeRunner) WithDisplayConsoleMetrics(enable bool) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithDisplayConsoleMetrics(enable)
	return r
}

func (r loadStrikeRunner) WithRestartIterationMaxAttempts(attempts int) LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithRestartIterationMaxAttempts(attempts)
	return r
}

func (r loadStrikeRunner) LoadConfig(path string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).LoadConfig(path)
	return r
}

func (r loadStrikeRunner) LoadInfraConfig(path string) LoadStrikeRunner {
	requireRunner(r.nativeValue()).LoadInfraConfig(path)
	return r
}

func (r loadStrikeRunner) WithoutReports() LoadStrikeRunner {
	requireRunner(r.nativeValue()).WithoutReports()
	return r
}

func (r loadStrikeRunner) BuildContext() LoadStrikeContext {
	return wrapLoadStrikeContext(requireRunner(r.nativeValue()).BuildContext())
}

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
