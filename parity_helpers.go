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

// WithAgentGroup configures agent group. Use this when you want to set agent group on the current SDK object.
func WithAgentGroup(context LoadStrikeContext, agentGroup string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithAgentGroup(agentGroup)
}

// WithAgentsCount configures agents count. Use this when you want to set agents count on the current SDK object.
func WithAgentsCount(context LoadStrikeContext, agentsCount int) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithAgentsCount(agentsCount)
}

// WithAgentTargetScenarios configures agent target scenarios. Use this when you want to set agent target scenarios on the current SDK object.
func WithAgentTargetScenarios(context LoadStrikeContext, scenarioNames ...string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithAgentTargetScenarios(scenarioNames...)
}

// WithClusterId configures cluster id. Use this when you want to set cluster id on the current SDK object.
func WithClusterId(context LoadStrikeContext, clusterID string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithClusterId(clusterID)
}

// WithCoordinatorTargetScenarios configures coordinator target scenarios. Use this when you want to set coordinator target scenarios on the current SDK object.
func WithCoordinatorTargetScenarios(context LoadStrikeContext, scenarioNames ...string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithCoordinatorTargetScenarios(scenarioNames...)
}

// WithRunnerKey configures runner key. Use this when you want to set runner key on the current SDK object.
func WithRunnerKey(context LoadStrikeContext, runnerKey string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRunnerKey(runnerKey)
}

// WithLicenseValidationTimeout configures license validation timeout. Use this when you want to set license validation timeout on the current SDK object.
func WithLicenseValidationTimeout(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithLicenseValidationTimeout(value)
}

// WithLoggerConfig configures logger config. Use this when you want to set logger config on the current SDK object.
func WithLoggerConfig(context LoadStrikeContext, config LoggerConfigurationFactory) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithLoggerConfig(config)
}

// WithMinimumLogLevel configures minimum log level. Use this when you want to set minimum log level on the current SDK object.
func WithMinimumLogLevel(context LoadStrikeContext, level LogEventLevel) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithMinimumLogLevel(level)
}

// WithNatsServerURL configures nats server url. Use this when you want to set nats server url on the current SDK object.
func WithNatsServerURL(context LoadStrikeContext, natsURL string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithNatsServerUrl(natsURL)
}

// WithNatsServerUrl configures nats server url. Use this when you want to set nats server url on the current SDK object.
func WithNatsServerUrl(context LoadStrikeContext, natsURL string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithNatsServerUrl(natsURL)
}

// WithNodeType configures node type. Use this when you want to set node type on the current SDK object.
func WithNodeType(context LoadStrikeContext, nodeType NodeType) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithNodeType(nodeType)
}

// WithoutReports configures out reports. Use this when you want to set out reports on the current SDK object.
func WithoutReports(context LoadStrikeContext) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithoutReports()
}

// WithReportFileName configures report file name. Use this when you want to set report file name on the current SDK object.
func WithReportFileName(context LoadStrikeContext, reportFileName string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportFileName(reportFileName)
}

// WithReportFolder configures report folder. Use this when you want to set report folder on the current SDK object.
func WithReportFolder(context LoadStrikeContext, reportFolder string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportFolder(reportFolder)
}

// WithReportFormats configures report formats. Use this when you want to set report formats on the current SDK object.
func WithReportFormats(context LoadStrikeContext, reportFormats ...ReportFormat) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportFormats(reportFormats...)
}

// WithReportingInterval configures reporting interval. Use this when you want to set reporting interval on the current SDK object.
func WithReportingInterval(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportingInterval(value)
}

// WithReportingSinks configures reporting sinks. Use this when you want to set reporting sinks on the current SDK object.
func WithReportingSinks(context LoadStrikeContext, reportingSinks ...LoadStrikeReportingSink) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithReportingSinks(reportingSinks...)
}

// WithRuntimePolicies configures runtime policies. Use this when you want to set runtime policies on the current SDK object.
func WithRuntimePolicies(context LoadStrikeContext, runtimePolicies ...LoadStrikeRuntimePolicy) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRuntimePolicies(runtimePolicies...)
}

// WithRuntimePolicyErrorMode configures runtime policy error mode. Use this when you want to set runtime policy error mode on the current SDK object.
func WithRuntimePolicyErrorMode(context LoadStrikeContext, mode RuntimePolicyErrorMode) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRuntimePolicyErrorMode(mode)
}

// WithScenarioCompletionTimeout configures scenario completion timeout. Use this when you want to set scenario completion timeout on the current SDK object.
func WithScenarioCompletionTimeout(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithScenarioCompletionTimeout(value)
}

// WithClusterCommandTimeout configures cluster command timeout. Use this when you want to set cluster command timeout on the current SDK object.
func WithClusterCommandTimeout(context LoadStrikeContext, value TimeSpan) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithClusterCommandTimeout(value)
}

// WithRestartIterationMaxAttempts configures restart iteration max attempts. Use this when you want to set restart iteration max attempts on the current SDK object.
func WithRestartIterationMaxAttempts(context LoadStrikeContext, attempts int) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithRestartIterationMaxAttempts(attempts)
}

// WithSessionId configures session id. Use this when you want to set session id on the current SDK object.
func WithSessionId(context LoadStrikeContext, sessionID string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithSessionId(sessionID)
}

// WithTargetScenarios configures target scenarios. Use this when you want to set target scenarios on the current SDK object.
func WithTargetScenarios(context LoadStrikeContext, scenarioNames ...string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithTargetScenarios(scenarioNames...)
}

// WithTestName configures test name. Use this when you want to set test name on the current SDK object.
func WithTestName(context LoadStrikeContext, testName string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithTestName(testName)
}

// WithTestSuite configures test suite. Use this when you want to set test suite on the current SDK object.
func WithTestSuite(context LoadStrikeContext, testSuite string) LoadStrikeContext {
	return requireLoadStrikeContext(context).WithTestSuite(testSuite)
}

// WithWorkerPlugins configures worker plugins. Use this when you want to set worker plugins on the current SDK object.
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
