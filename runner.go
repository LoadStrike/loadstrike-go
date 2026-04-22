package loadstrike

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"
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

func execute(context contextState, scenarios []scenarioDefinition) (result runResult, err error) {
	startedUTC := time.Now().UTC()
	nodeInfo := currentNodeInfo(context.NodeType, LoadStrikeOperationTypeComplete)
	testInfo := currentTestInfo(context, startedUTC)
	result = runResult{
		StartedUTC:            startedUTC,
		NodeType:              context.NodeType,
		nodeInfo:              nodeInfo,
		testInfo:              testInfo,
		scenarioStats:         make([]scenarioStats, 0, len(scenarios)),
		stepStats:             []stepStats{},
		Thresholds:            []thresholdResult{},
		ScenarioDurationsMS:   make(map[string]float64, len(scenarios)),
		DisabledSinks:         []string{},
		SinkErrors:            []sinkErrorResult{},
		Metrics:               []metricResult{},
		CorrelationRows:       []correlationRow{},
		FailedCorrelationRows: []correlationRow{},
		PluginsData:           []pluginData{},
		PolicyErrors:          []RuntimePolicyError{},
		reportTrace:           newReportTrace(),
	}
	logger, err := startRunLogger(context, result.testInfo, result.nodeInfo)
	if err != nil {
		return runResult{}, err
	}
	publicLogger := newLoadStrikeLogger(nil)
	if logger != nil {
		publicLogger = logger.publicLogger()
		defer func() {
			logger.closeWithResult(&result, err)
		}()
		if path := logger.pathValue(); path != "" {
			result.LogFiles = append(result.LogFiles, path)
		}
	}
	control := newExecutionControl()

	for scenarioIndex, scenario := range scenarios {
		if control.shouldStopTest() {
			break
		}
		if control.shouldStopScenario(scenario.Name) {
			result.ScenarioDurationsMS[scenario.Name] = 0
			result.scenarioStats = append(result.scenarioStats, scenarioStats{
				ScenarioName:     scenario.Name,
				CurrentOperation: "Stop",
				SortIndex:        scenarioIndex,
				stepStats:        []stepStats{},
			})
			continue
		}
		scenarioStarted := time.Now()
		scenarioStats := scenarioStats{
			ScenarioName:     scenario.Name,
			CurrentOperation: "Complete",
			SortIndex:        scenarioIndex,
			stepStats:        []stepStats{},
		}
		aggregatedStepStats := map[string]*stepStats{}
		scenarioInstanceData := map[string]any{}

		scenarioContext := &scenarioHookContext{
			ScenarioName:         scenario.Name,
			TestSuite:            context.TestSuite,
			TestName:             context.TestName,
			ScenarioInstanceData: scenarioInstanceData,
			metricRegistry:       &metricRegistry{},
			Logger:               publicLogger,
			nodeInfo:             result.nodeInfo,
			testInfo:             result.testInfo,
			CustomSettings:       newIConfiguration(map[string]any{}),
			GlobalCustomSettings: newIConfiguration(map[string]any{}),
			Partition: scenarioPartitionInfo{
				Number: context.agentIndex,
				Count:  maxInt(maxInt(context.agentCount, context.AgentsCount), 1),
			},
		}
		scenarioContext.scenarioPartitionInfo = scenarioContext.Partition
		scenarioContext.ScenarioInfo = scenarioRuntimeInfo(scenario.Name, maxInt(context.agentIndex, 0), scenarioStarted, LoadStrikeScenarioOperationInit)

		shouldRunScenario, policyErrors, err := evaluateScenarioPolicies(context, scenarioContext)
		if err != nil && context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
			return runResult{}, err
		}

		result.PolicyErrors = append(result.PolicyErrors, policyErrors...)
		if !shouldRunScenario {
			result.ScenarioDurationsMS[scenario.Name] = 0
			result.scenarioStats = append(result.scenarioStats, scenarioStats)
			continue
		}

		if scenario.Init != nil {
			if err := scenario.Init(scenarioContext); err != nil {
				return runResult{}, err
			}
		}

		if !scenario.WarmUpDisabled && scenario.WarmUpDurationSeconds > 0 {
			time.Sleep(time.Duration(scenario.WarmUpDurationSeconds * float64(time.Second)))
		}

		var trackingExecutor *trackingScenarioExecutor
		if scenario.Tracking != nil {
			trackingExecutor, err = newTrackingScenarioExecutor(context, scenario)
			if err != nil {
				return runResult{}, err
			}
		}

		iterations := scenarioIterations(scenario)
		stopCurrentTest := false
		for iteration := 0; iteration < iterations; iteration++ {
			requestCount, okCount, failCount, stepStats, policyErrors, stopScenario, stopTest, err := executeScenarioIteration(
				context,
				scenario,
				scenarioInstanceData,
				iteration+1,
				control,
				result.reportTrace,
				trackingExecutor,
				publicLogger,
				result.nodeInfo,
				result.testInfo,
				scenarioStarted,
			)
			if err != nil {
				if trackingExecutor != nil {
					_ = trackingExecutor.Close()
				}
				return runResult{}, err
			}
			scenarioStats.AllRequestCount += requestCount
			scenarioStats.AllOKCount += okCount
			scenarioStats.AllFailCount += failCount
			result.AllRequestCount += requestCount
			result.AllOKCount += okCount
			result.AllFailCount += failCount
			result.PolicyErrors = append(result.PolicyErrors, policyErrors...)
			for _, stepStat := range stepStats {
				mergeStepStats(aggregatedStepStats, stepStat)
				scenarioStats.AllBytes += stepStat.AllBytes
			}

			if scenario.MaxFailCount > 0 && scenarioStats.AllFailCount >= scenario.MaxFailCount {
				break
			}

			if stopScenario || control.shouldStopScenario(scenario.Name) {
				break
			}
			if stopTest {
				control.stopTest("requested by scenario step")
				stopCurrentTest = true
				break
			}
		}

		if scenario.Clean != nil {
			scenarioContext.ScenarioInfo = scenarioRuntimeInfo(scenario.Name, maxInt(context.agentIndex, 0), scenarioStarted, LoadStrikeScenarioOperationClean)
			if err := scenario.Clean(scenarioContext); err != nil {
				if trackingExecutor != nil {
					_ = trackingExecutor.Close()
				}
				return runResult{}, err
			}
		}

		if trackingExecutor != nil {
			if err := trackingExecutor.Close(); err != nil {
				return runResult{}, err
			}
			result.CorrelationRows = append(result.CorrelationRows, trackingExecutor.CorrelationRows()...)
			result.FailedCorrelationRows = append(result.FailedCorrelationRows, trackingExecutor.FailedCorrelationRows()...)
		}

		result.ScenarioDurationsMS[scenario.Name] = time.Since(scenarioStarted).Seconds() * 1000
		scenarioStats.DurationMS = result.ScenarioDurationsMS[scenario.Name]
		scenarioStats.Duration = time.Duration(scenarioStats.DurationMS * float64(time.Millisecond))
		scenarioStats.CurrentOperationType = LoadStrikeOperationTypeComplete
		scenarioStats.stepStats = orderedMergedStepStats(aggregatedStepStats)
		populateScenarioMeasurementStats(&scenarioStats)
		mergeScenarioMetrics(&result, scenario.Name, scenarioContext.metricRegistry, scenarioStats.Duration)
		if err := runAfterScenarioPolicies(context, scenarioContext, scenarioStats, &result.PolicyErrors); err != nil && context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
			return runResult{}, err
		}
		result.AllBytes += scenarioStats.AllBytes
		result.stepStats = append(result.stepStats, scenarioStats.stepStats...)
		result.scenarioStats = append(result.scenarioStats, scenarioStats)
		if stopCurrentTest {
			break
		}
	}

	result.Thresholds = evaluateThresholds(scenarios, result)
	result.ThresholdResults = append([]thresholdResult(nil), result.Thresholds...)
	for _, threshold := range result.Thresholds {
		if threshold.IsFailed {
			result.FailedThresholds++
		}
	}
	result.CompletedUTC = time.Now().UTC()
	result.DurationMS = result.CompletedUTC.Sub(result.StartedUTC).Seconds() * 1000
	result.Duration = result.CompletedUTC.Sub(result.StartedUTC)
	normalizeRunResult(&result)
	return result, nil
}

func evaluateScenarioPolicies(context contextState, scenarioContext *scenarioHookContext) (bool, []RuntimePolicyError, error) {
	policyErrors := []RuntimePolicyError{}
	for _, runtimePolicy := range context.RuntimePolicies {
		if runtimePolicy == nil {
			continue
		}

		shouldRun, err := runtimePolicyShouldRunScenario(runtimePolicy, scenarioContext)
		if err != nil {
			policyErrors = append(policyErrors, RuntimePolicyError{
				PolicyName:   runtimePolicyName(runtimePolicy),
				CallbackName: "shouldRunScenario",
				ScenarioName: scenarioContext.ScenarioName,
				Message:      err.Error(),
			})
			return true, policyErrors, err
		}
		if !shouldRun {
			return false, policyErrors, nil
		}
		if err := runtimePolicyBeforeScenario(runtimePolicy, scenarioContext); err != nil {
			policyErrors = append(policyErrors, RuntimePolicyError{
				PolicyName:   runtimePolicyName(runtimePolicy),
				CallbackName: "beforeScenario",
				ScenarioName: scenarioContext.ScenarioName,
				Message:      err.Error(),
			})
			return true, policyErrors, err
		}
	}

	return true, policyErrors, nil
}

func runAfterScenarioPolicies(context contextState, scenarioContext *scenarioHookContext, scenarioStats scenarioStats, policyErrors *[]RuntimePolicyError) error {
	for _, runtimePolicy := range context.RuntimePolicies {
		if runtimePolicy == nil {
			continue
		}

		if err := runtimePolicyAfterScenario(runtimePolicy, scenarioContext, scenarioStats); err != nil {
			*policyErrors = append(*policyErrors, RuntimePolicyError{
				PolicyName:   runtimePolicyName(runtimePolicy),
				CallbackName: "afterScenario",
				ScenarioName: scenarioContext.ScenarioName,
				Message:      err.Error(),
			})
			return err
		}
	}

	return nil
}

func executeScenarioIteration(
	context contextState,
	scenario scenarioDefinition,
	scenarioInstanceData map[string]any,
	invocationNumber int,
	control *executionControl,
	runTrace *reportTrace,
	trackingExecutor *trackingScenarioExecutor,
	logger *LoadStrikeLogger,
	nodeInfo nodeInfo,
	testInfo testInfo,
	scenarioStarted time.Time,
) (int, int, int, []stepStats, []RuntimePolicyError, bool, bool, error) {
	if trackingExecutor != nil {
		requestCount, okCount, failCount, stepStats, policyErrors, stopScenario, err := executeTrackingScenarioIteration(context, scenario, scenarioInstanceData, invocationNumber, runTrace, trackingExecutor)
		return requestCount, okCount, failCount, stepStats, policyErrors, stopScenario, false, err
	}

	attemptsLeft := 0
	if scenario.RestartIterationOnFail {
		attemptsLeft = context.RestartIterationMaxAttempts
	}

	for {
		requestCount := 0
		okCount := 0
		failCount := 0
		shouldRestart := false
		stopScenario := false
		attemptTrace := newReportTrace()
		stepStatsRows := make([]stepStats, 0, len(scenario.Steps))
		policyErrors := []RuntimePolicyError{}
		stepData := map[string]any{}

		for _, step := range scenario.Steps {
			if step.Run == nil {
				continue
			}

			scenarioToken, scenarioCancel := control.scenarioContext(scenario.Name)
			stepContext := &stepRuntimeContext{
				ScenarioName:              scenario.Name,
				StepName:                  step.Name,
				TestSuite:                 context.TestSuite,
				TestName:                  context.TestName,
				InvocationNumber:          int64(invocationNumber),
				Data:                      stepData,
				ScenarioInstanceData:      scenarioInstanceData,
				Logger:                    logger,
				nodeInfo:                  nodeInfo,
				testInfo:                  testInfo,
				ScenarioInfo:              scenarioRuntimeInfo(scenario.Name, invocationNumber, scenarioStarted, LoadStrikeScenarioOperationBombing),
				Random:                    newScenarioRandom(invocationNumber),
				ScenarioCancellationToken: scenarioToken,
				scenarioCancel:            scenarioCancel,
				executionControl:          control,
				scenarioTimerStarted:      scenarioStarted,
			}

			if err := runBeforeStepPolicies(context, stepContext, &policyErrors); err != nil {
				if context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
					return 0, 0, 0, nil, nil, false, false, err
				}
			}

			stepStarted := time.Now()
			reply := step.Run(stepContext)
			latencyMS := reply.CustomLatencyMS
			if latencyMS <= 0 {
				latencyMS = float64(time.Since(stepStarted)) / float64(time.Millisecond)
			}

			requestCount++
			attemptTrace.addStep(step.Name)
			stepStat := stepStats{
				ScenarioName:    scenario.Name,
				StepName:        step.Name,
				AllBytes:        reply.SizeBytes,
				AllRequestCount: 1,
				SortIndex:       len(stepStatsRows),
			}

			if reply.IsSuccess {
				okCount++
				stepStat.AllOKCount = 1
				stepStat.Ok = measurementStatsForReply(reply, latencyMS, true)
			} else {
				failCount++
				stepStat.AllFailCount = 1
				stepStat.Fail = measurementStatsForReply(reply, latencyMS, false)
				attemptTrace.addFailure(scenario.Name, step.Name, reply)
				shouldRestart = scenario.RestartIterationOnFail && attemptsLeft > 0
			}

			stepStatsRows = append(stepStatsRows, stepStat)

			if err := runAfterStepPolicies(context, stepContext, reply, &policyErrors); err != nil {
				if context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
					return 0, 0, 0, nil, nil, false, false, err
				}
			}

			if stepContext.stopRequested {
				stopScenario = true
			}
			if stepContext.stopCurrentTest {
				return requestCount, okCount, failCount, stepStatsRows, policyErrors, stopScenario, true, nil
			}
			if scenarioCancel != nil {
				scenarioCancel()
			}

			if !reply.IsSuccess {
				break
			}

			if stopScenario {
				break
			}
		}

		if !shouldRestart {
			runTrace.merge(attemptTrace)
			return requestCount, okCount, failCount, stepStatsRows, policyErrors, stopScenario, false, nil
		}

		attemptsLeft--
	}
}

func runBeforeStepPolicies(context contextState, stepContext *stepRuntimeContext, policyErrors *[]RuntimePolicyError) error {
	for _, runtimePolicy := range context.RuntimePolicies {
		if runtimePolicy == nil {
			continue
		}

		if err := runtimePolicyBeforeStep(runtimePolicy, stepContext); err != nil {
			*policyErrors = append(*policyErrors, RuntimePolicyError{
				PolicyName:   runtimePolicyName(runtimePolicy),
				CallbackName: "beforeStep",
				ScenarioName: stepContext.ScenarioName,
				StepName:     stepContext.StepName,
				Message:      err.Error(),
			})
			return err
		}
	}

	return nil
}

func runAfterStepPolicies(context contextState, stepContext *stepRuntimeContext, reply replyResult, policyErrors *[]RuntimePolicyError) error {
	for _, runtimePolicy := range context.RuntimePolicies {
		if runtimePolicy == nil {
			continue
		}

		if err := runtimePolicyAfterStep(runtimePolicy, stepContext, reply); err != nil {
			*policyErrors = append(*policyErrors, RuntimePolicyError{
				PolicyName:   runtimePolicyName(runtimePolicy),
				CallbackName: "afterStep",
				ScenarioName: stepContext.ScenarioName,
				StepName:     stepContext.StepName,
				Message:      err.Error(),
			})
			return err
		}
	}

	return nil
}

func scenarioIterations(scenario scenarioDefinition) int {
	if len(scenario.LoadSimulations) == 0 {
		return 1
	}

	total := 0
	for _, simulation := range scenario.LoadSimulations {
		if simulation.Kind == "IterationsForConstant" {
			total += applyScenarioWeight(simulation.Iterations, scenario.Weight)
			continue
		}
		if simulation.Kind == "IterationsForInject" {
			total += applyScenarioWeight(maxInt(simulation.Iterations, 1), scenario.Weight)
			continue
		}

		copies := simulation.Copies
		if copies <= 0 {
			copies = 1
		}

		iterations := simulation.Iterations
		if iterations <= 0 {
			iterations = 1
		}

		total += copies * iterations
	}

	if total <= 0 {
		return 1
	}

	return total
}

func applyScenarioWeight(value int, weight int) int {
	if value <= 0 {
		return value
	}

	if weight <= 0 {
		weight = 1
	}

	maxInt := int(^uint(0) >> 1)
	if value > maxInt/weight {
		return maxInt
	}

	return value * weight
}

func currentMachineName() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "unknown"
	}
	return name
}

func measurementStatsForReply(reply replyResult, latencyMS float64, success bool) measurementStats {
	statusCodes := []statusCodeStats{}
	if strings.TrimSpace(reply.StatusCode) != "" {
		statusCodes = append(statusCodes, statusCodeStats{
			StatusCode: reply.StatusCode,
			Message:    reply.Message,
			IsError:    !success,
			Count:      1,
			Percent:    100,
		})
	}
	request := requestStats{Count: 1, Percent: 100}
	if latencyMS > 0 {
		request.RPS = 1000.0 / latencyMS
	}
	return measurementStats{
		Request: request,
		DataTransfer: dataTransferStats{
			AllBytes:  reply.SizeBytes,
			MinBytes:  reply.SizeBytes,
			MaxBytes:  reply.SizeBytes,
			MeanBytes: reply.SizeBytes,
			Percent50: reply.SizeBytes,
			Percent75: reply.SizeBytes,
			Percent95: reply.SizeBytes,
			Percent99: reply.SizeBytes,
		},
		Latency: latencyStats{
			LatencyCount: classifyLatency(latencyMS, 1),
			MinMs:        latencyMS,
			MaxMs:        latencyMS,
			MeanMs:       latencyMS,
			Percent50:    latencyMS,
			Percent75:    latencyMS,
			Percent95:    latencyMS,
			Percent99:    latencyMS,
			P50Ms:        latencyMS,
			P75Ms:        latencyMS,
			P80Ms:        latencyMS,
			P85Ms:        latencyMS,
			P90Ms:        latencyMS,
			P95Ms:        latencyMS,
			P99Ms:        latencyMS,
			Samples:      1,
		},
		StatusCodes: statusCodes,
	}
}

func mergeStepStats(target map[string]*stepStats, source stepStats) {
	enrichDetailedStepStats(&source)
	key := strings.TrimSpace(source.StepName)
	if existing, ok := target[key]; ok {
		existing.AllBytes += source.AllBytes
		existing.AllRequestCount += source.AllRequestCount
		existing.AllOKCount += source.AllOKCount
		existing.AllFailCount += source.AllFailCount
		existing.Ok = mergeMeasurementStats(existing.Ok, source.Ok)
		existing.Fail = mergeMeasurementStats(existing.Fail, source.Fail)
		enrichDetailedStepStats(existing)
		return
	}
	copyValue := source
	target[key] = &copyValue
}

func orderedMergedStepStats(aggregated map[string]*stepStats) []stepStats {
	result := make([]stepStats, 0, len(aggregated))
	for _, stepStat := range aggregated {
		result = append(result, *stepStat)
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].SortIndex == result[right].SortIndex {
			return result[left].StepName < result[right].StepName
		}
		return result[left].SortIndex < result[right].SortIndex
	})
	return result
}

func mergeMeasurementStats(left measurementStats, right measurementStats) measurementStats {
	totalCount := left.Request.Count + right.Request.Count
	statusCodeCounts := map[string]int{}
	for _, item := range left.StatusCodes {
		statusCodeCounts[item.StatusCode] += item.Count
	}
	for _, item := range right.StatusCodes {
		statusCodeCounts[item.StatusCode] += item.Count
	}
	statusCodes := make([]statusCodeStats, 0, len(statusCodeCounts))
	for code, count := range statusCodeCounts {
		statusCodes = append(statusCodes, statusCodeStats{
			StatusCode: code,
			Count:      count,
			Percent:    percentageInt(count, totalCount),
			IsError:    isErrorStatusCode(code),
			Message:    statusCodeMessage(left.StatusCodes, right.StatusCodes, code),
		})
	}
	sort.Slice(statusCodes, func(leftIndex int, rightIndex int) bool {
		return statusCodes[leftIndex].StatusCode < statusCodes[rightIndex].StatusCode
	})
	totalBytes := left.DataTransfer.AllBytes + right.DataTransfer.AllBytes
	totalLatencySamples := left.Latency.Samples + right.Latency.Samples
	totalLatencyValue := left.Latency.MeanMs*float64(left.Latency.Samples) + right.Latency.MeanMs*float64(right.Latency.Samples)
	request := requestStats{Count: totalCount}
	if totalCount > 0 {
		request.Percent = 100
	}
	latencyMean := 0.0
	if totalLatencySamples > 0 {
		latencyMean = totalLatencyValue / float64(totalLatencySamples)
	}
	return measurementStats{
		Request: request,
		DataTransfer: dataTransferStats{
			AllBytes:  totalBytes,
			MinBytes:  minInt64NonZero(left.DataTransfer.MinBytes, right.DataTransfer.MinBytes),
			MaxBytes:  maxInt64(left.DataTransfer.MaxBytes, right.DataTransfer.MaxBytes),
			MeanBytes: meanInt64(totalBytes, totalCount),
			Percent50: maxInt64(left.DataTransfer.Percent50, right.DataTransfer.Percent50),
			Percent75: maxInt64(left.DataTransfer.Percent75, right.DataTransfer.Percent75),
			Percent95: maxInt64(left.DataTransfer.Percent95, right.DataTransfer.Percent95),
			Percent99: maxInt64(left.DataTransfer.Percent99, right.DataTransfer.Percent99),
		},
		Latency: latencyStats{
			LatencyCount: mergeLatencyCount(left.Latency.LatencyCount, right.Latency.LatencyCount),
			MinMs:        minPositiveFloat(left.Latency.MinMs, right.Latency.MinMs),
			MaxMs:        thresholdMaxFloat(left.Latency.MaxMs, right.Latency.MaxMs),
			MeanMs:       latencyMean,
			Percent50:    thresholdMaxFloat(latencyPercentile(left.Latency, 0.50), latencyPercentile(right.Latency, 0.50)),
			Percent75:    thresholdMaxFloat(latencyPercentile(left.Latency, 0.75), latencyPercentile(right.Latency, 0.75)),
			Percent95:    thresholdMaxFloat(latencyPercentile(left.Latency, 0.95), latencyPercentile(right.Latency, 0.95)),
			Percent99:    thresholdMaxFloat(latencyPercentile(left.Latency, 0.99), latencyPercentile(right.Latency, 0.99)),
			P50Ms:        thresholdMaxFloat(left.Latency.P50Ms, right.Latency.P50Ms),
			P75Ms:        thresholdMaxFloat(left.Latency.P75Ms, right.Latency.P75Ms),
			P80Ms:        thresholdMaxFloat(left.Latency.P80Ms, right.Latency.P80Ms),
			P85Ms:        thresholdMaxFloat(left.Latency.P85Ms, right.Latency.P85Ms),
			P90Ms:        thresholdMaxFloat(left.Latency.P90Ms, right.Latency.P90Ms),
			P95Ms:        thresholdMaxFloat(left.Latency.P95Ms, right.Latency.P95Ms),
			P99Ms:        thresholdMaxFloat(left.Latency.P99Ms, right.Latency.P99Ms),
			Samples:      totalLatencySamples,
		},
		StatusCodes: statusCodes,
	}
}

func populateScenarioMeasurementStats(stats *scenarioStats) {
	if stats == nil {
		return
	}
	stats.Ok = measurementStats{}
	stats.Fail = measurementStats{}
	for _, stepStat := range stats.stepStats {
		stats.Ok = mergeMeasurementStats(stats.Ok, stepStat.Ok)
		stats.Fail = mergeMeasurementStats(stats.Fail, stepStat.Fail)
	}
	stats.LoadSimulationStats = loadSimulationStats{
		SimulationName: "",
		Value:          stats.AllRequestCount,
	}
	enrichDetailedScenarioStats(stats)
}

func mergeScenarioMetrics(result *runResult, scenarioName string, registry *metricRegistry, duration time.Duration) {
	if result == nil || registry == nil {
		return
	}
	metricStats, projected := registry.snapshot(scenarioName)
	if duration > 0 {
		metricStats.Duration = duration
		metricStats.DurationMS = float64(duration) / float64(time.Millisecond)
	}
	if len(metricStats.Counters) > 0 {
		result.metricStats.Counters = append(result.metricStats.Counters, metricStats.Counters...)
	}
	if len(metricStats.Gauges) > 0 {
		result.metricStats.Gauges = append(result.metricStats.Gauges, metricStats.Gauges...)
	}
	if metricStats.DurationMS > result.metricStats.DurationMS {
		result.metricStats.DurationMS = metricStats.DurationMS
		result.metricStats.Duration = metricStats.Duration
	}
	if len(projected) > 0 {
		result.Metrics = append(result.Metrics, projected...)
	}
}

func evaluateThresholds(scenarios []scenarioDefinition, result runResult) []thresholdResult {
	byScenarioName := map[string]scenarioStats{}
	for _, scenarioStats := range result.scenarioStats {
		byScenarioName[scenarioStats.ScenarioName] = scenarioStats
	}
	thresholds := make([]thresholdResult, 0)
	for _, scenario := range scenarios {
		scenarioStats, ok := byScenarioName[scenario.Name]
		if !ok {
			continue
		}
		for _, threshold := range scenario.Thresholds {
			thresholdResult := thresholdResult{
				ScenarioName:    scenario.Name,
				StepName:        threshold.StepName,
				CheckExpression: buildThresholdExpression(threshold),
			}
			if threshold.StartCheckAfterSeconds > 0 && scenarioStats.DurationMS < threshold.StartCheckAfterSeconds*1000.0 {
				thresholds = append(thresholds, thresholdResult)
				continue
			}
			failed, err := evaluateThresholdSpec(threshold, scenarioStats, result.metricStats)
			if err != nil {
				thresholdResult.ErrorCount = 1
				thresholdResult.IsFailed = true
				thresholdResult.ExceptionMessage = err.Error()
			} else if failed {
				thresholdResult.ErrorCount = 1
				thresholdResult.IsFailed = true
			}
			if threshold.AbortWhenErrorCount > 0 && thresholdResult.ErrorCount >= threshold.AbortWhenErrorCount {
				thresholdResult.IsFailed = true
			}
			thresholds = append(thresholds, thresholdResult)
		}
	}
	return thresholds
}

func evaluateThresholdSpec(threshold ThresholdSpec, scenarioStats scenarioStats, metricStats metricStats) (bool, error) {
	if threshold.scenarioPredicate != nil {
		return !threshold.scenarioPredicate(scenarioStats), nil
	}
	if threshold.stepPredicate != nil {
		stepStats := scenarioStats.FindStepStats(threshold.StepName)
		if stepStats == nil {
			return true, fmt.Errorf("step stats not found for %q", threshold.StepName)
		}
		return !threshold.stepPredicate(*stepStats), nil
	}
	if threshold.metricPredicate != nil {
		return !threshold.metricPredicate(metricStats), nil
	}

	var actual float64
	var err error
	switch strings.ToLower(strings.TrimSpace(threshold.Scope)) {
	case "scenario":
		actual, err = thresholdScenarioValue(scenarioStats, threshold.Field)
	case "step":
		stepStats := scenarioStats.FindStepStats(threshold.StepName)
		if stepStats == nil {
			return true, fmt.Errorf("step stats not found for %q", threshold.StepName)
		}
		actual, err = thresholdStepValue(*stepStats, threshold.Field)
	case "metric":
		actual, err = thresholdMetricValue(metricStats, threshold.Field)
	default:
		return true, fmt.Errorf("unsupported threshold scope %q", threshold.Scope)
	}
	if err != nil {
		return true, err
	}
	return !compareThresholdValues(actual, threshold.Operator, threshold.Value), nil
}

func buildThresholdExpression(threshold ThresholdSpec) string {
	if threshold.scenarioPredicate != nil {
		return "scenario predicate"
	}
	if threshold.stepPredicate != nil {
		return fmt.Sprintf("step[%s] predicate", threshold.StepName)
	}
	if threshold.metricPredicate != nil {
		return "metric predicate"
	}
	return fmt.Sprintf("%s %s %s", threshold.Field, threshold.Operator, formatThresholdValue(threshold.Value))
}

func formatThresholdValue(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%g", value)
}

func thresholdScenarioValue(stats scenarioStats, field string) (float64, error) {
	return thresholdMeasurementValue(field, stats.AllRequestCount, stats.AllOKCount, stats.AllFailCount, stats.AllBytes, stats.DurationMS, stats.Ok, stats.Fail)
}

func thresholdStepValue(stats stepStats, field string) (float64, error) {
	return thresholdMeasurementValue(field, stats.AllRequestCount, stats.AllOKCount, stats.AllFailCount, stats.AllBytes, 0, stats.Ok, stats.Fail)
}

func thresholdMeasurementValue(field string, requestCount int, okCount int, failCount int, allBytes int64, durationMS float64, ok measurementStats, fail measurementStats) (float64, error) {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "allrequestcount", "requestcount":
		return float64(requestCount), nil
	case "allokcount", "okcount":
		return float64(okCount), nil
	case "allfailcount", "failcount":
		return float64(failCount), nil
	case "allbytes", "bytes":
		return float64(allBytes), nil
	case "durationms":
		return durationMS, nil
	case "latencyp95ms":
		return thresholdMaxFloat(latencyPercentile(ok.Latency, 0.95), latencyPercentile(fail.Latency, 0.95)), nil
	case "latencyp99ms":
		return thresholdMaxFloat(latencyPercentile(ok.Latency, 0.99), latencyPercentile(fail.Latency, 0.99)), nil
	case "latencymeanms":
		return thresholdMaxFloat(ok.Latency.MeanMs, fail.Latency.MeanMs), nil
	default:
		return 0, fmt.Errorf("unsupported threshold field %q", field)
	}
}

func thresholdMetricValue(stats metricStats, field string) (float64, error) {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "countercount":
		return float64(len(stats.Counters)), nil
	case "gaugecount":
		return float64(len(stats.Gauges)), nil
	case "durationms":
		return stats.DurationMS, nil
	}
	total := 0.0
	matched := false
	for _, counter := range stats.Counters {
		if strings.EqualFold(counter.MetricName, field) {
			total += float64(counter.Value)
			matched = true
		}
	}
	for _, gauge := range stats.Gauges {
		if strings.EqualFold(gauge.MetricName, field) {
			total += gauge.Value
			matched = true
		}
	}
	if matched {
		return total, nil
	}
	return 0, fmt.Errorf("unsupported metric threshold field %q", field)
}

func compareThresholdValues(actual float64, operator string, expected float64) bool {
	switch strings.TrimSpace(operator) {
	case ">", "gt":
		return actual > expected
	case ">=", "gte":
		return actual >= expected
	case "<", "lt":
		return actual < expected
	case "<=", "lte":
		return actual <= expected
	case "==", "=":
		return actual == expected
	case "!=", "<>":
		return actual != expected
	default:
		return false
	}
}

func minInt64NonZero(left int64, right int64) int64 {
	switch {
	case left == 0:
		return right
	case right == 0:
		return left
	case left < right:
		return left
	default:
		return right
	}
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func meanInt64(total int64, count int) int64 {
	if count <= 0 {
		return 0
	}
	return total / int64(count)
}

func minPositiveFloat(left float64, right float64) float64 {
	switch {
	case left <= 0:
		return right
	case right <= 0:
		return left
	case left < right:
		return left
	default:
		return right
	}
}

func thresholdMaxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func latencyPercentile(stats latencyStats, percentile float64) float64 {
	switch percentile {
	case 0.50:
		if stats.Percent50 > 0 {
			return stats.Percent50
		}
		return stats.P50Ms
	case 0.75:
		if stats.Percent75 > 0 {
			return stats.Percent75
		}
		return stats.P75Ms
	case 0.95:
		if stats.Percent95 > 0 {
			return stats.Percent95
		}
		return stats.P95Ms
	case 0.99:
		if stats.Percent99 > 0 {
			return stats.Percent99
		}
		return stats.P99Ms
	default:
		return 0
	}
}

func mergeLatencyCount(left latencyCount, right latencyCount) latencyCount {
	return latencyCount{
		LessOrEq800:     left.LessOrEq800 + right.LessOrEq800,
		More800Less1200: left.More800Less1200 + right.More800Less1200,
		MoreOrEq1200:    left.MoreOrEq1200 + right.MoreOrEq1200,
	}
}

func classifyLatency(latencyMS float64, count int) latencyCount {
	switch {
	case latencyMS >= 1200:
		return latencyCount{MoreOrEq1200: count}
	case latencyMS > 800:
		return latencyCount{More800Less1200: count}
	default:
		return latencyCount{LessOrEq800: count}
	}
}

func percentageInt(value int, total int) int {
	if total <= 0 {
		return 0
	}
	return int(float64(value)*100.0/float64(total) + 0.5)
}

func enrichDetailedScenarioStats(stats *scenarioStats) {
	if stats == nil {
		return
	}
	stats.TotalBytes = stats.AllBytes
	stats.TotalLatencyMS = measurementTotalLatency(stats.Ok) + measurementTotalLatency(stats.Fail)
	if stats.AllRequestCount > 0 {
		stats.AvgLatencyMS = stats.TotalLatencyMS / float64(stats.AllRequestCount)
	}
	stats.MinLatencyMS = measurementMinLatency(stats.Ok, stats.Fail)
	stats.MaxLatencyMS = thresholdMaxFloat(stats.Ok.Latency.MaxMs, stats.Fail.Latency.MaxMs)
	stats.StatusCodes = mergeStatusCodeCountMaps(statusCodeCountMap(stats.Ok.StatusCodes), statusCodeCountMap(stats.Fail.StatusCodes))
}

func enrichDetailedStepStats(stats *stepStats) {
	if stats == nil {
		return
	}
	stats.TotalBytes = stats.AllBytes
	stats.TotalLatencyMS = measurementTotalLatency(stats.Ok) + measurementTotalLatency(stats.Fail)
	if stats.AllRequestCount > 0 {
		stats.AvgLatencyMS = stats.TotalLatencyMS / float64(stats.AllRequestCount)
	}
	stats.MinLatencyMS = measurementMinLatency(stats.Ok, stats.Fail)
	stats.MaxLatencyMS = thresholdMaxFloat(stats.Ok.Latency.MaxMs, stats.Fail.Latency.MaxMs)
	stats.StatusCodes = mergeStatusCodeCountMaps(statusCodeCountMap(stats.Ok.StatusCodes), statusCodeCountMap(stats.Fail.StatusCodes))
}

func measurementTotalLatency(stats measurementStats) float64 {
	return stats.Latency.MeanMs * float64(stats.Request.Count)
}

func measurementMinLatency(values ...measurementStats) float64 {
	minimum := 0.0
	for _, value := range values {
		if value.Request.Count <= 0 {
			continue
		}
		minimum = minPositiveFloat(minimum, value.Latency.MinMs)
	}
	return minimum
}

func statusCodeCountMap(rows []statusCodeStats) map[string]int {
	if len(rows) == 0 {
		return nil
	}
	result := make(map[string]int, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.StatusCode) == "" {
			continue
		}
		result[row.StatusCode] += row.Count
	}
	return result
}

func mergeStatusCodeCountMaps(maps ...map[string]int) map[string]int {
	merged := map[string]int{}
	for _, source := range maps {
		for key, value := range source {
			if strings.TrimSpace(key) != "" {
				merged[key] += value
			}
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func isErrorStatusCode(code string) bool {
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return false
	}
	if code == "failed" {
		return true
	}
	return strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") || strings.Contains(code, "error")
}

func statusCodeMessage(left []statusCodeStats, right []statusCodeStats, code string) string {
	for _, candidate := range append(append([]statusCodeStats(nil), left...), right...) {
		if strings.EqualFold(candidate.StatusCode, code) && strings.TrimSpace(candidate.Message) != "" {
			return candidate.Message
		}
	}
	return ""
}

func runtimePolicyName(policy any) string {
	if policy == nil {
		return ""
	}
	typed := reflect.TypeOf(policy)
	for typed.Kind() == reflect.Pointer {
		typed = typed.Elem()
	}
	return typed.Name()
}

func runtimePolicyShouldRunScenario(policy any, scenarioContext *scenarioHookContext) (bool, error) {
	switch typed := policy.(type) {
	case LoadStrikeRuntimePolicy:
		return typed.ShouldRunScenario(scenarioContext.ScenarioName).Await()
	case loadStrikeAsyncRuntimePolicy:
		return typed.ShouldRunScenarioAsync(scenarioContext.ScenarioName).Await()
	case legacyLoadStrikeRuntimePolicy:
		return typed.ShouldRunScenario(scenarioContext.ScenarioName), nil
	case runtimePolicy:
		return typed.ShouldRunScenario(scenarioContext), nil
	case loadStrikeTaskRuntimePolicyShouldRun:
		return typed.ShouldRunScenario(scenarioContext.ScenarioName).Await()
	case loadStrikeAsyncRuntimePolicyShouldRun:
		return typed.ShouldRunScenarioAsync(scenarioContext.ScenarioName).Await()
	case legacyLoadStrikeRuntimePolicyShouldRun:
		return typed.ShouldRunScenario(scenarioContext.ScenarioName), nil
	default:
		if isRuntimePolicyLike(policy) {
			return true, nil
		}
		return true, fmt.Errorf("unsupported runtime policy type %T", policy)
	}
}

func runtimePolicyBeforeScenario(policy any, scenarioContext *scenarioHookContext) error {
	switch typed := policy.(type) {
	case LoadStrikeRuntimePolicy:
		return typed.BeforeScenario(scenarioContext.ScenarioName).Await()
	case loadStrikeAsyncRuntimePolicy:
		return typed.BeforeScenarioAsync(scenarioContext.ScenarioName).Await()
	case legacyLoadStrikeRuntimePolicy:
		return typed.BeforeScenario(scenarioContext.ScenarioName)
	case runtimePolicy:
		return typed.BeforeScenario(scenarioContext)
	case loadStrikeTaskRuntimePolicyBeforeScenario:
		return typed.BeforeScenario(scenarioContext.ScenarioName).Await()
	case loadStrikeAsyncRuntimePolicyBeforeScenario:
		return typed.BeforeScenarioAsync(scenarioContext.ScenarioName).Await()
	case legacyLoadStrikeRuntimePolicyBeforeScenario:
		return typed.BeforeScenario(scenarioContext.ScenarioName)
	default:
		if isRuntimePolicyLike(policy) {
			return nil
		}
		return fmt.Errorf("unsupported runtime policy type %T", policy)
	}
}

func runtimePolicyAfterScenario(policy any, scenarioContext *scenarioHookContext, scenarioStats scenarioStats) error {
	switch typed := policy.(type) {
	case LoadStrikeRuntimePolicy:
		return typed.AfterScenario(scenarioContext.ScenarioName, buildScenarioRuntime(scenarioStats)).Await()
	case loadStrikeAsyncRuntimePolicy:
		return typed.AfterScenarioAsync(scenarioContext.ScenarioName, buildScenarioRuntime(scenarioStats)).Await()
	case legacyLoadStrikeRuntimePolicy:
		return typed.AfterScenario(scenarioContext.ScenarioName, buildScenarioRuntime(scenarioStats))
	case runtimePolicy:
		return typed.AfterScenario(scenarioContext, scenarioStats)
	case loadStrikeTaskRuntimePolicyAfterScenario:
		return typed.AfterScenario(scenarioContext.ScenarioName, buildScenarioRuntime(scenarioStats)).Await()
	case loadStrikeAsyncRuntimePolicyAfterScenario:
		return typed.AfterScenarioAsync(scenarioContext.ScenarioName, buildScenarioRuntime(scenarioStats)).Await()
	case legacyLoadStrikeRuntimePolicyAfterScenario:
		return typed.AfterScenario(scenarioContext.ScenarioName, buildScenarioRuntime(scenarioStats))
	default:
		if isRuntimePolicyLike(policy) {
			return nil
		}
		return fmt.Errorf("unsupported runtime policy type %T", policy)
	}
}

func runtimePolicyBeforeStep(policy any, stepContext *stepRuntimeContext) error {
	switch typed := policy.(type) {
	case LoadStrikeRuntimePolicy:
		return typed.BeforeStep(stepContext.ScenarioName, stepContext.StepName).Await()
	case loadStrikeAsyncRuntimePolicy:
		return typed.BeforeStepAsync(stepContext.ScenarioName, stepContext.StepName).Await()
	case legacyLoadStrikeRuntimePolicy:
		return typed.BeforeStep(stepContext.ScenarioName, stepContext.StepName)
	case runtimePolicy:
		return typed.BeforeStep(stepContext)
	case loadStrikeTaskRuntimePolicyBeforeStep:
		return typed.BeforeStep(stepContext.ScenarioName, stepContext.StepName).Await()
	case loadStrikeAsyncRuntimePolicyBeforeStep:
		return typed.BeforeStepAsync(stepContext.ScenarioName, stepContext.StepName).Await()
	case legacyLoadStrikeRuntimePolicyBeforeStep:
		return typed.BeforeStep(stepContext.ScenarioName, stepContext.StepName)
	default:
		if isRuntimePolicyLike(policy) {
			return nil
		}
		return fmt.Errorf("unsupported runtime policy type %T", policy)
	}
}

func runtimePolicyAfterStep(policy any, stepContext *stepRuntimeContext, reply replyResult) error {
	switch typed := policy.(type) {
	case LoadStrikeRuntimePolicy:
		return typed.AfterStep(stepContext.ScenarioName, stepContext.StepName, newLoadStrikeReply(reply)).Await()
	case loadStrikeAsyncRuntimePolicy:
		return typed.AfterStepAsync(stepContext.ScenarioName, stepContext.StepName, newLoadStrikeReply(reply)).Await()
	case legacyLoadStrikeRuntimePolicy:
		return typed.AfterStep(stepContext.ScenarioName, stepContext.StepName, newLoadStrikeReply(reply))
	case runtimePolicy:
		return typed.AfterStep(stepContext, reply)
	case loadStrikeTaskRuntimePolicyAfterStep:
		return typed.AfterStep(stepContext.ScenarioName, stepContext.StepName, newLoadStrikeReply(reply)).Await()
	case loadStrikeAsyncRuntimePolicyAfterStep:
		return typed.AfterStepAsync(stepContext.ScenarioName, stepContext.StepName, newLoadStrikeReply(reply)).Await()
	case legacyLoadStrikeRuntimePolicyAfterStep:
		return typed.AfterStep(stepContext.ScenarioName, stepContext.StepName, newLoadStrikeReply(reply))
	default:
		if isRuntimePolicyLike(policy) {
			return nil
		}
		return fmt.Errorf("unsupported runtime policy type %T", policy)
	}
}

func buildScenarioRuntime(stats scenarioStats) LoadStrikeScenarioRuntime {
	return LoadStrikeScenarioRuntime{
		ScenarioName:    stats.ScenarioName,
		AllRequestCount: stats.AllRequestCount,
		AllOKCount:      stats.AllOKCount,
		AllFailCount:    stats.AllFailCount,
		TotalBytes:      stats.TotalBytes,
		TotalLatencyMS:  stats.TotalLatencyMS,
		AvgLatencyMS:    stats.AvgLatencyMS,
		MinLatencyMS:    stats.MinLatencyMS,
		MaxLatencyMS:    stats.MaxLatencyMS,
		StatusCodes:     cloneStatusCodeCounts(stats.StatusCodes),
	}
}
