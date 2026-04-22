package loadstrike

import "strings"

// Create mirrors the .NET static runner constructor.
func Create() LoadStrikeRunner {
	return NewRunner()
}

func requireLoadStrikeContext(context LoadStrikeContext) LoadStrikeContext {
	if context.native == nil {
		panic("context must be provided")
	}
	return context
}

// DisplayConsoleMetrics mirrors the .NET static context helper.
func DisplayConsoleMetrics(context LoadStrikeContext, enable bool) LoadStrikeContext {
	return requireLoadStrikeContext(context).DisplayConsoleMetrics(enable)
}

// EnableLocalDevCluster mirrors the .NET static context helper.
func EnableLocalDevCluster(context LoadStrikeContext, enable bool) LoadStrikeContext {
	return requireLoadStrikeContext(context).EnableLocalDevCluster(enable)
}

// LoadConfig mirrors the .NET static context helper.
func LoadConfig(context LoadStrikeContext, path string) LoadStrikeContext {
	return requireLoadStrikeContext(context).LoadConfig(path)
}

// LoadInfraConfig mirrors the .NET static context helper.
func LoadInfraConfig(context LoadStrikeContext, path string) LoadStrikeContext {
	return requireLoadStrikeContext(context).LoadInfraConfig(path)
}

// Run mirrors the .NET static context runner entrypoint.
func Run(context LoadStrikeContext, args ...string) LoadStrikeRunResult {
	return requireLoadStrikeContext(context).Run(args...)
}

func WithAgentGroup(context LoadStrikeContext, agentGroup string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithAgentGroup(agentGroup)
}

func WithAgentsCount(context LoadStrikeContext, agentsCount int) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithAgentsCount(agentsCount)
}

func WithAgentTargetScenarios(context LoadStrikeContext, scenarioNames ...string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithAgentTargetScenarios(scenarioNames...)
}

func WithClusterId(context LoadStrikeContext, clusterID string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithClusterId(clusterID)
}

func WithCoordinatorTargetScenarios(context LoadStrikeContext, scenarioNames ...string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithCoordinatorTargetScenarios(scenarioNames...)
}

func WithRunnerKey(context LoadStrikeContext, runnerKey string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRunnerKey(runnerKey)
}

func WithLicenseValidationTimeout(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithLicenseValidationTimeout(value)
}

func WithLoggerConfig(context LoadStrikeContext, config LoggerConfigurationFactory) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithLoggerConfig(config)
}

func WithMinimumLogLevel(context LoadStrikeContext, level LogEventLevel) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithMinimumLogLevel(level)
}

func WithNatsServerURL(context LoadStrikeContext, natsURL string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithNatsServerUrl(natsURL)
}

func WithNatsServerUrl(context LoadStrikeContext, natsURL string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithNatsServerUrl(natsURL)
}

func WithNodeType(context LoadStrikeContext, nodeType NodeType) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithNodeType(nodeType)
}

func WithoutReports(context LoadStrikeContext) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithoutReports()
}

func WithReportFileName(context LoadStrikeContext, reportFileName string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportFileName(reportFileName)
}

func WithReportFolder(context LoadStrikeContext, reportFolder string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportFolder(reportFolder)
}

func WithReportFormats(context LoadStrikeContext, reportFormats ...ReportFormat) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportFormats(reportFormats...)
}

func WithReportingInterval(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportingInterval(value)
}

func WithReportingSinks(context LoadStrikeContext, reportingSinks ...LoadStrikeReportingSink) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportingSinks(reportingSinks...)
}

func WithRuntimePolicies(context LoadStrikeContext, runtimePolicies ...LoadStrikeRuntimePolicy) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRuntimePolicies(runtimePolicies...)
}

func WithRuntimePolicyErrorMode(context LoadStrikeContext, mode RuntimePolicyErrorMode) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRuntimePolicyErrorMode(mode)
}

func WithScenarioCompletionTimeout(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithScenarioCompletionTimeout(value)
}

func WithClusterCommandTimeout(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithClusterCommandTimeout(value)
}

func WithRestartIterationMaxAttempts(context LoadStrikeContext, attempts int) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRestartIterationMaxAttempts(attempts)
}

func WithSessionId(context LoadStrikeContext, sessionID string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithSessionId(sessionID)
}

func WithTargetScenarios(context LoadStrikeContext, scenarioNames ...string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithTargetScenarios(scenarioNames...)
}

func WithTestName(context LoadStrikeContext, testName string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithTestName(testName)
}

func WithTestSuite(context LoadStrikeContext, testSuite string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithTestSuite(testSuite)
}

func WithWorkerPlugins(context LoadStrikeContext, plugins ...LoadStrikeWorkerPlugin) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithWorkerPlugins(plugins...)
}

// CreateScenarioThreshold mirrors the .NET predicate threshold helper.
func CreateScenarioThreshold(args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(ScenarioThreshold(args...))
}

// CreateStepThreshold mirrors the .NET predicate threshold helper.
func CreateStepThreshold(stepName string, args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(StepThreshold(stepName, args...))
}

// CreateMetricThreshold mirrors the .NET predicate threshold helper.
func CreateMetricThreshold(args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(MetricThreshold(args...))
}

// CreatePluginData mirrors the .NET plugin-data constructor helper.
func CreatePluginData(pluginName string) LoadStrikePluginData {
	if strings.TrimSpace(pluginName) == "" {
		panic("Plugin name must be provided.")
	}
	return LoadStrikePluginData{
		PluginName: pluginName,
		Hints:      []string{},
		Tables:     []LoadStrikePluginDataTable{},
	}
}

// CreatePluginDataTable mirrors the .NET plugin-table constructor helper.
func CreatePluginDataTable(tableName string) LoadStrikePluginDataTable {
	if strings.TrimSpace(tableName) == "" {
		panic("Table name must be provided.")
	}
	return LoadStrikePluginDataTable{
		TableName: tableName,
		Rows:      []map[string]any{},
	}
}

// OKWith returns a successful typed reply.
func OKWith[T any](value T, args ...any) LoadStrikeReplyWith[T] {
	reply := okReply(args...)
	reply.Payload = value
	return LoadStrikeReplyWith[T]{
		reply:   newLoadStrikeReply(reply),
		payload: value,
	}
}

// FailWith returns a failed typed reply.
func FailWith[T any](value T, args ...any) LoadStrikeReplyWith[T] {
	reply := failReply(args...)
	reply.Payload = value
	return LoadStrikeReplyWith[T]{
		reply:   newLoadStrikeReply(reply),
		payload: value,
	}
}

// RunStep mirrors the .NET helper that executes a named typed step wrapper.
func RunStep[T any](name string, context LoadStrikeScenarioContext, run func(LoadStrikeScenarioContext) LoadStrikeReplyWith[T]) LoadStrikeReplyWith[T] {
	if strings.TrimSpace(name) == "" {
		panic("step name must be provided.")
	}
	if context.native == nil {
		panic("step context must be provided.")
	}
	if run == nil {
		panic("step run must be provided.")
	}
	if context.native.StepName == "" {
		context.native.StepName = name
	}
	return run(context)
}

// CreateScenarioAsync mirrors the async-first .NET scenario builder.
func CreateScenarioAsync(name string, run LoadStrikeAsyncScenarioRunFunc) LoadStrikeScenario {
	return newLoadStrikeScenario(createScenario(
		name,
		func(context LoadStrikeScenarioContext) LoadStrikeReply {
			reply, err := run(context).Await()
			if err != nil {
				return LoadStrikeResponse.Fail("500", err.Error())
			}
			return reply
		},
	))
}
