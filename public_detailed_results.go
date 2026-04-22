package loadstrike

import (
	"encoding/json"
	"fmt"
	"time"
)

// LoadStrikeDetailedStepStats mirrors the .NET public detailed step-stats wrapper.
type LoadStrikeDetailedStepStats struct {
	scenarioName    string                     `json:"ScenarioName,omitempty"`
	stepName        string                     `json:"StepName,omitempty"`
	allBytes        int64                      `json:"AllBytes,omitempty"`
	allRequestCount int                        `json:"AllRequestCount,omitempty"`
	allOKCount      int                        `json:"AllOkCount,omitempty"`
	allFailCount    int                        `json:"AllFailCount,omitempty"`
	totalBytes      int64                      `json:"TotalBytes,omitempty"`
	totalLatencyMS  float64                    `json:"TotalLatencyMs,omitempty"`
	avgLatencyMS    float64                    `json:"AvgLatencyMs,omitempty"`
	minLatencyMS    float64                    `json:"MinLatencyMs,omitempty"`
	maxLatencyMS    float64                    `json:"MaxLatencyMs,omitempty"`
	statusCodes     map[string]int             `json:"StatusCodes,omitempty"`
	ok              LoadStrikeMeasurementStats `json:"Ok,omitempty"`
	fail            LoadStrikeMeasurementStats `json:"Fail,omitempty"`
	sortIndex       int                        `json:"SortIndex,omitempty"`
	lookupFlavor    statsLookupFlavor          `json:"-"`
}

func newLoadStrikeDetailedStepStats(native stepStats) LoadStrikeDetailedStepStats {
	return LoadStrikeDetailedStepStats{
		scenarioName:    native.ScenarioName,
		stepName:        native.StepName,
		allBytes:        native.AllBytes,
		allRequestCount: native.AllRequestCount,
		allOKCount:      native.AllOKCount,
		allFailCount:    native.AllFailCount,
		totalBytes:      native.TotalBytes,
		totalLatencyMS:  native.TotalLatencyMS,
		avgLatencyMS:    native.AvgLatencyMS,
		minLatencyMS:    native.MinLatencyMS,
		maxLatencyMS:    native.MaxLatencyMS,
		statusCodes:     cloneStatusCodeCounts(native.StatusCodes),
		ok:              newLoadStrikeMeasurementStats(native.Ok),
		fail:            newLoadStrikeMeasurementStats(native.Fail),
		sortIndex:       native.SortIndex,
		lookupFlavor:    native.lookupFlavor,
	}
}

func (s LoadStrikeDetailedStepStats) toNative() stepStats {
	return stepStats{
		ScenarioName:    s.scenarioName,
		StepName:        s.stepName,
		AllBytes:        s.allBytes,
		AllRequestCount: s.allRequestCount,
		AllOKCount:      s.allOKCount,
		AllFailCount:    s.allFailCount,
		TotalBytes:      s.totalBytes,
		TotalLatencyMS:  s.totalLatencyMS,
		AvgLatencyMS:    s.avgLatencyMS,
		MinLatencyMS:    s.minLatencyMS,
		MaxLatencyMS:    s.maxLatencyMS,
		StatusCodes:     cloneStatusCodeCounts(s.statusCodes),
		Ok:              s.ok.toNative(),
		Fail:            s.fail.toNative(),
		SortIndex:       s.sortIndex,
		lookupFlavor:    s.lookupFlavor,
	}
}

func (s LoadStrikeDetailedStepStats) ScenarioName() string { return s.scenarioName }
func (s LoadStrikeDetailedStepStats) StepName() string     { return s.stepName }
func (s LoadStrikeDetailedStepStats) AllRequestCount() int { return s.allRequestCount }
func (s LoadStrikeDetailedStepStats) OkCount() int         { return s.allOKCount }
func (s LoadStrikeDetailedStepStats) FailCount() int       { return s.allFailCount }
func (s LoadStrikeDetailedStepStats) TotalBytes() int64    { return s.totalBytes }
func (s LoadStrikeDetailedStepStats) TotalLatencyMS() float64 {
	return s.totalLatencyMS
}
func (s LoadStrikeDetailedStepStats) AvgLatencyMS() float64 { return s.avgLatencyMS }
func (s LoadStrikeDetailedStepStats) MinLatencyMS() float64 { return s.minLatencyMS }
func (s LoadStrikeDetailedStepStats) MaxLatencyMS() float64 { return s.maxLatencyMS }
func (s LoadStrikeDetailedStepStats) StatusCodes() map[string]int {
	return cloneStatusCodeCounts(s.statusCodes)
}
func (s LoadStrikeDetailedStepStats) Ok() LoadStrikeMeasurementStats   { return s.ok }
func (s LoadStrikeDetailedStepStats) Fail() LoadStrikeMeasurementStats { return s.fail }
func (s LoadStrikeDetailedStepStats) SortIndex() int                   { return s.sortIndex }

func (s LoadStrikeDetailedStepStats) FindStepStats(stepName string) *LoadStrikeDetailedStepStats {
	if s.lookupFlavor != statsLookupFlavorRealtime {
		validateStepStatsLookupName(stepName)
	}
	if s.stepName == stepName {
		value := s
		return &value
	}
	return nil
}

func (s LoadStrikeDetailedStepStats) GetStepStats(stepName string) *LoadStrikeDetailedStepStats {
	value := s.FindStepStats(stepName)
	if value == nil {
		if s.lookupFlavor == statsLookupFlavorRealtime {
			panic(fmt.Sprintf("Step stats not found: %s.", stepName))
		}
		panic(fmt.Sprintf("Step '%s' was not found.", stepName))
	}
	return value
}

// LoadStrikeDetailedScenarioStats mirrors the .NET public detailed scenario-stats wrapper.
type LoadStrikeDetailedScenarioStats struct {
	scenarioName         string                        `json:"ScenarioName,omitempty"`
	allBytes             int64                         `json:"AllBytes,omitempty"`
	allRequestCount      int                           `json:"AllRequestCount,omitempty"`
	allOKCount           int                           `json:"AllOkCount,omitempty"`
	allFailCount         int                           `json:"AllFailCount,omitempty"`
	totalBytes           int64                         `json:"TotalBytes,omitempty"`
	totalLatencyMS       float64                       `json:"TotalLatencyMs,omitempty"`
	avgLatencyMS         float64                       `json:"AvgLatencyMs,omitempty"`
	minLatencyMS         float64                       `json:"MinLatencyMs,omitempty"`
	maxLatencyMS         float64                       `json:"MaxLatencyMs,omitempty"`
	statusCodes          map[string]int                `json:"StatusCodes,omitempty"`
	currentOperation     string                        `json:"CurrentOperation,omitempty"`
	currentOperationType LoadStrikeOperationType       `json:"CurrentOperationType,omitempty"`
	durationMS           float64                       `json:"DurationMs,omitempty"`
	duration             time.Duration                 `json:"Duration,omitempty"`
	ok                   LoadStrikeMeasurementStats    `json:"Ok,omitempty"`
	fail                 LoadStrikeMeasurementStats    `json:"Fail,omitempty"`
	loadSimulationStats  LoadStrikeLoadSimulationStats `json:"LoadSimulationStats,omitempty"`
	sortIndex            int                           `json:"SortIndex,omitempty"`
	stepStats            []LoadStrikeDetailedStepStats `json:"stepStats,omitempty"`
	lookupFlavor         statsLookupFlavor             `json:"-"`
}

func newLoadStrikeDetailedScenarioStats(native scenarioStats) LoadStrikeDetailedScenarioStats {
	normalizeScenarioStats(&native)
	steps := make([]LoadStrikeDetailedStepStats, 0, len(native.stepStats))
	for _, step := range native.stepStats {
		steps = append(steps, newLoadStrikeDetailedStepStats(step))
	}
	return LoadStrikeDetailedScenarioStats{
		scenarioName:         native.ScenarioName,
		allBytes:             native.AllBytes,
		allRequestCount:      native.AllRequestCount,
		allOKCount:           native.AllOKCount,
		allFailCount:         native.AllFailCount,
		totalBytes:           native.TotalBytes,
		totalLatencyMS:       native.TotalLatencyMS,
		avgLatencyMS:         native.AvgLatencyMS,
		minLatencyMS:         native.MinLatencyMS,
		maxLatencyMS:         native.MaxLatencyMS,
		statusCodes:          cloneStatusCodeCounts(native.StatusCodes),
		currentOperation:     native.CurrentOperation,
		currentOperationType: native.CurrentOperationType,
		durationMS:           native.DurationMS,
		duration:             native.Duration,
		ok:                   newLoadStrikeMeasurementStats(native.Ok),
		fail:                 newLoadStrikeMeasurementStats(native.Fail),
		loadSimulationStats:  newLoadStrikeLoadSimulationStats(native.LoadSimulationStats),
		sortIndex:            native.SortIndex,
		stepStats:            steps,
		lookupFlavor:         native.lookupFlavor,
	}
}

func (s LoadStrikeDetailedScenarioStats) toNative() scenarioStats {
	steps := make([]stepStats, 0, len(s.stepStats))
	for _, step := range s.stepStats {
		steps = append(steps, step.toNative())
	}
	native := scenarioStats{
		ScenarioName:         s.scenarioName,
		AllBytes:             s.allBytes,
		AllRequestCount:      s.allRequestCount,
		AllOKCount:           s.allOKCount,
		AllFailCount:         s.allFailCount,
		TotalBytes:           s.totalBytes,
		TotalLatencyMS:       s.totalLatencyMS,
		AvgLatencyMS:         s.avgLatencyMS,
		MinLatencyMS:         s.minLatencyMS,
		MaxLatencyMS:         s.maxLatencyMS,
		StatusCodes:          cloneStatusCodeCounts(s.statusCodes),
		CurrentOperation:     s.currentOperation,
		CurrentOperationType: s.currentOperationType,
		DurationMS:           s.durationMS,
		Duration:             s.duration,
		Ok:                   s.ok.toNative(),
		Fail:                 s.fail.toNative(),
		LoadSimulationStats:  s.loadSimulationStats.toNative(),
		SortIndex:            s.sortIndex,
		stepStats:            steps,
		lookupFlavor:         s.lookupFlavor,
	}
	normalizeScenarioStats(&native)
	return native
}

func (s LoadStrikeDetailedScenarioStats) ScenarioName() string { return s.scenarioName }
func (s LoadStrikeDetailedScenarioStats) AllBytes() int64      { return s.allBytes }
func (s LoadStrikeDetailedScenarioStats) AllRequestCount() int { return s.allRequestCount }
func (s LoadStrikeDetailedScenarioStats) AllOKCount() int      { return s.allOKCount }
func (s LoadStrikeDetailedScenarioStats) AllFailCount() int    { return s.allFailCount }
func (s LoadStrikeDetailedScenarioStats) TotalBytes() int64    { return s.totalBytes }
func (s LoadStrikeDetailedScenarioStats) TotalLatencyMS() float64 {
	return s.totalLatencyMS
}
func (s LoadStrikeDetailedScenarioStats) AvgLatencyMS() float64 { return s.avgLatencyMS }
func (s LoadStrikeDetailedScenarioStats) MinLatencyMS() float64 { return s.minLatencyMS }
func (s LoadStrikeDetailedScenarioStats) MaxLatencyMS() float64 { return s.maxLatencyMS }
func (s LoadStrikeDetailedScenarioStats) StatusCodes() map[string]int {
	return cloneStatusCodeCounts(s.statusCodes)
}
func (s LoadStrikeDetailedScenarioStats) CurrentOperation() string {
	return s.currentOperation
}
func (s LoadStrikeDetailedScenarioStats) CurrentOperationType() LoadStrikeOperationType {
	return s.currentOperationType
}
func (s LoadStrikeDetailedScenarioStats) DurationMS() float64            { return s.durationMS }
func (s LoadStrikeDetailedScenarioStats) Ok() LoadStrikeMeasurementStats { return s.ok }
func (s LoadStrikeDetailedScenarioStats) Fail() LoadStrikeMeasurementStats {
	return s.fail
}
func (s LoadStrikeDetailedScenarioStats) LoadSimulationStats() LoadStrikeLoadSimulationStats {
	return s.loadSimulationStats
}
func (s LoadStrikeDetailedScenarioStats) SortIndex() int { return s.sortIndex }

func (s LoadStrikeDetailedScenarioStats) FindStepStats(stepName string) *LoadStrikeDetailedStepStats {
	if s.lookupFlavor != statsLookupFlavorRealtime {
		validateStepStatsLookupName(stepName)
	}
	for index := range s.stepStats {
		if s.stepStats[index].stepName == stepName {
			value := s.stepStats[index]
			value.lookupFlavor = s.lookupFlavor
			return &value
		}
	}
	return nil
}

func (s LoadStrikeDetailedScenarioStats) GetStepStats(stepName string) *LoadStrikeDetailedStepStats {
	value := s.FindStepStats(stepName)
	if value == nil {
		if s.lookupFlavor == statsLookupFlavorRealtime {
			panic(fmt.Sprintf("Step stats not found: %s.", stepName))
		}
		panic(fmt.Sprintf("Step '%s' was not found for scenario '%s'.", stepName, s.scenarioName))
	}
	return value
}

func (s LoadStrikeDetailedScenarioStats) StepStats() []LoadStrikeDetailedStepStats {
	return append([]LoadStrikeDetailedStepStats(nil), s.stepStats...)
}

// LoadStrikeRunResult mirrors the .NET public final run-result wrapper.
type LoadStrikeRunResult struct {
	startedUTC            time.Time                         `json:"StartedUtc"`
	completedUTC          time.Time                         `json:"CompletedUtc"`
	durationMS            float64                           `json:"DurationMs,omitempty"`
	duration              time.Duration                     `json:"Duration,omitempty"`
	allBytes              int64                             `json:"AllBytes,omitempty"`
	allRequestCount       int                               `json:"AllRequestCount"`
	allOKCount            int                               `json:"AllOkCount"`
	allFailCount          int                               `json:"AllFailCount"`
	failedThresholds      int                               `json:"FailedThresholds,omitempty"`
	nodeType              LoadStrikeNodeType                `json:"NodeType,omitempty"`
	nodeInfo              LoadStrikeNodeInfo                `json:"nodeInfo,omitempty"`
	testInfo              LoadStrikeTestInfo                `json:"testInfo"`
	thresholds            []LoadStrikeThresholdResult       `json:"Thresholds,omitempty"`
	thresholdResults      []LoadStrikeThresholdResult       `json:"ThresholdResults,omitempty"`
	metricStats           LoadStrikeMetricStats             `json:"metricStats,omitempty"`
	metrics               []LoadStrikeMetricValue           `json:"Metrics,omitempty"`
	scenarioStats         []LoadStrikeDetailedScenarioStats `json:"scenarioStats"`
	stepStats             []LoadStrikeDetailedStepStats     `json:"stepStats,omitempty"`
	scenarioDurationsMS   map[string]int64                  `json:"ScenarioDurationsMs,omitempty"`
	pluginsData           []LoadStrikePluginData            `json:"PluginsData,omitempty"`
	disabledSinks         []string                          `json:"DisabledSinks,omitempty"`
	sinkErrors            []LoadStrikeSinkError             `json:"SinkErrors,omitempty"`
	reportFiles           []string                          `json:"ReportFiles,omitempty"`
	logFiles              []string                          `json:"LogFiles,omitempty"`
	policyErrors          []LoadStrikeRuntimePolicyError    `json:"PolicyErrors,omitempty"`
	correlationRows       []map[string]any                  `json:"CorrelationRows,omitempty"`
	failedCorrelationRows []map[string]any                  `json:"FailedCorrelationRows,omitempty"`
	reportTrace           *reportTrace                      `json:"-"`
}

func newLoadStrikeRunResult(native runResult) LoadStrikeRunResult {
	normalizeRunResult(&native)
	scenarios := make([]LoadStrikeDetailedScenarioStats, 0, len(native.scenarioStats))
	for _, scenario := range native.scenarioStats {
		scenarios = append(scenarios, newLoadStrikeDetailedScenarioStats(scenario))
	}
	steps := make([]LoadStrikeDetailedStepStats, 0, len(native.stepStats))
	for _, step := range native.stepStats {
		steps = append(steps, newLoadStrikeDetailedStepStats(step))
	}
	metrics := make([]LoadStrikeMetricValue, 0, len(native.Metrics))
	for _, metric := range native.Metrics {
		metrics = append(metrics, newLoadStrikeMetricValue(metric))
	}
	sinkErrors := make([]LoadStrikeSinkError, 0, len(native.SinkErrors))
	for _, sinkError := range native.SinkErrors {
		sinkErrors = append(sinkErrors, newLoadStrikeSinkError(sinkError))
	}
	policyErrors := make([]LoadStrikeRuntimePolicyError, 0, len(native.PolicyErrors))
	for _, policyError := range native.PolicyErrors {
		policyErrors = append(policyErrors, newLoadStrikeRuntimePolicyError(policyError))
	}
	return LoadStrikeRunResult{
		startedUTC:            native.StartedUTC,
		completedUTC:          native.CompletedUTC,
		durationMS:            native.DurationMS,
		duration:              native.Duration,
		allBytes:              native.AllBytes,
		allRequestCount:       native.AllRequestCount,
		allOKCount:            native.AllOKCount,
		allFailCount:          native.AllFailCount,
		failedThresholds:      native.FailedThresholds,
		nodeType:              native.NodeType,
		nodeInfo:              newLoadStrikeNodeInfo(native.nodeInfo),
		testInfo:              newLoadStrikeTestInfo(native.testInfo),
		thresholds:            toLoadStrikeThresholdResultSlice(native.Thresholds),
		thresholdResults:      toLoadStrikeThresholdResultSlice(native.ThresholdResults),
		metricStats:           newLoadStrikeMetricStats(native.metricStats),
		metrics:               metrics,
		scenarioStats:         scenarios,
		stepStats:             steps,
		scenarioDurationsMS:   cloneInt64MapFromFloatMap(native.ScenarioDurationsMS),
		pluginsData:           toLoadStrikePluginDataSlice(native.PluginsData),
		disabledSinks:         append([]string(nil), native.DisabledSinks...),
		sinkErrors:            sinkErrors,
		reportFiles:           append([]string(nil), native.ReportFiles...),
		logFiles:              append([]string(nil), native.LogFiles...),
		policyErrors:          policyErrors,
		correlationRows:       newPublicCorrelationRows(native.CorrelationRows),
		failedCorrelationRows: newPublicCorrelationRows(native.FailedCorrelationRows),
		reportTrace:           native.reportTrace,
	}
}

func (r LoadStrikeRunResult) toNative() runResult {
	scenarios := make([]scenarioStats, 0, len(r.scenarioStats))
	for _, scenario := range r.scenarioStats {
		scenarios = append(scenarios, scenario.toNative())
	}
	steps := make([]stepStats, 0, len(r.stepStats))
	for _, step := range r.stepStats {
		steps = append(steps, step.toNative())
	}
	metrics := make([]metricResult, 0, len(r.metrics))
	for _, metric := range r.metrics {
		metrics = append(metrics, metric.toNative())
	}
	plugins := make([]pluginData, 0, len(r.pluginsData))
	for _, plugin := range r.pluginsData {
		plugins = append(plugins, plugin.toNative())
	}
	sinkErrors := make([]sinkErrorResult, 0, len(r.sinkErrors))
	for _, sinkError := range r.sinkErrors {
		sinkErrors = append(sinkErrors, sinkError.toNative())
	}
	native := runResult{
		StartedUTC:            r.startedUTC,
		CompletedUTC:          r.completedUTC,
		DurationMS:            r.durationMS,
		Duration:              r.duration,
		AllBytes:              r.allBytes,
		AllRequestCount:       r.allRequestCount,
		AllOKCount:            r.allOKCount,
		AllFailCount:          r.allFailCount,
		FailedThresholds:      r.failedThresholds,
		NodeType:              r.nodeType,
		nodeInfo:              r.nodeInfo.toNative(),
		testInfo:              r.testInfo.toNative(),
		Thresholds:            toNativeThresholdResults(r.thresholds),
		ThresholdResults:      toNativeThresholdResults(r.thresholdResults),
		metricStats:           r.metricStats.toNative(),
		Metrics:               metrics,
		scenarioStats:         scenarios,
		stepStats:             steps,
		ScenarioDurationsMS:   cloneFloatMapFromInt64Map(r.scenarioDurationsMS),
		PluginsData:           plugins,
		DisabledSinks:         append([]string(nil), r.disabledSinks...),
		SinkErrors:            sinkErrors,
		ReportFiles:           append([]string(nil), r.reportFiles...),
		LogFiles:              append([]string(nil), r.logFiles...),
		PolicyErrors:          toNativeRuntimePolicyErrors(r.policyErrors),
		CorrelationRows:       toNativeCorrelationRows(r.correlationRows),
		FailedCorrelationRows: toNativeCorrelationRows(r.failedCorrelationRows),
		reportTrace:           r.reportTrace,
	}
	normalizeRunResult(&native)
	return native
}

func (r LoadStrikeRunResult) StartedUtc() string {
	if r.startedUTC.IsZero() {
		return ""
	}
	return r.startedUTC.Format(time.RFC3339Nano)
}
func (r LoadStrikeRunResult) CompletedUtc() string {
	if r.completedUTC.IsZero() {
		return ""
	}
	return r.completedUTC.Format(time.RFC3339Nano)
}
func (r LoadStrikeRunResult) DurationMS() float64   { return r.durationMS }
func (r LoadStrikeRunResult) AllBytes() int64       { return r.allBytes }
func (r LoadStrikeRunResult) AllRequestCount() int  { return r.allRequestCount }
func (r LoadStrikeRunResult) AllOKCount() int       { return r.allOKCount }
func (r LoadStrikeRunResult) AllFailCount() int     { return r.allFailCount }
func (r LoadStrikeRunResult) FailedThresholds() int { return r.failedThresholds }
func (r LoadStrikeRunResult) Thresholds() []LoadStrikeThresholdResult {
	return append([]LoadStrikeThresholdResult(nil), r.thresholds...)
}
func (r LoadStrikeRunResult) ThresholdResults() []LoadStrikeThresholdResult {
	return append([]LoadStrikeThresholdResult(nil), r.thresholdResults...)
}
func (r LoadStrikeRunResult) Metrics() []LoadStrikeMetricValue {
	return append([]LoadStrikeMetricValue(nil), r.metrics...)
}
func (r LoadStrikeRunResult) ScenarioDurationsMS() map[string]int64 {
	return cloneInt64Map(r.scenarioDurationsMS)
}
func (r LoadStrikeRunResult) PluginsData() []LoadStrikePluginData {
	return append([]LoadStrikePluginData(nil), r.pluginsData...)
}
func (r LoadStrikeRunResult) DisabledSinks() []string {
	return append([]string(nil), r.disabledSinks...)
}
func (r LoadStrikeRunResult) SinkErrors() []LoadStrikeSinkError {
	return append([]LoadStrikeSinkError(nil), r.sinkErrors...)
}
func (r LoadStrikeRunResult) ReportFiles() []string {
	return append([]string(nil), r.reportFiles...)
}
func (r LoadStrikeRunResult) LogFiles() []string {
	return append([]string(nil), r.logFiles...)
}
func (r LoadStrikeRunResult) PolicyErrors() []LoadStrikeRuntimePolicyError {
	return append([]LoadStrikeRuntimePolicyError(nil), r.policyErrors...)
}
func (r LoadStrikeRunResult) CorrelationRows() []map[string]any {
	rows := make([]map[string]any, 0, len(r.correlationRows))
	for _, row := range r.correlationRows {
		rows = append(rows, cloneAnyMap(row))
	}
	return rows
}
func (r LoadStrikeRunResult) FailedCorrelationRows() []map[string]any {
	rows := make([]map[string]any, 0, len(r.failedCorrelationRows))
	for _, row := range r.failedCorrelationRows {
		rows = append(rows, cloneAnyMap(row))
	}
	return rows
}

func (r LoadStrikeRunResult) FindScenarioStats(scenarioName string) *LoadStrikeDetailedScenarioStats {
	validateScenarioStatsLookupName(scenarioName)
	for index := range r.scenarioStats {
		if r.scenarioStats[index].scenarioName == scenarioName {
			value := r.scenarioStats[index]
			return &value
		}
	}
	return nil
}

func (r LoadStrikeRunResult) GetScenarioStats(scenarioName string) *LoadStrikeDetailedScenarioStats {
	value := r.FindScenarioStats(scenarioName)
	if value == nil {
		panic(fmt.Sprintf("Scenario '%s' was not found.", scenarioName))
	}
	return value
}

func (r LoadStrikeRunResult) ScenarioStats() []LoadStrikeDetailedScenarioStats {
	return append([]LoadStrikeDetailedScenarioStats(nil), r.scenarioStats...)
}

func (r LoadStrikeRunResult) StepStats() []LoadStrikeDetailedStepStats {
	return append([]LoadStrikeDetailedStepStats(nil), r.stepStats...)
}

func (r LoadStrikeRunResult) MetricStats() LoadStrikeMetricStats {
	return r.metricStats
}

func (r LoadStrikeRunResult) NodeInfo() LoadStrikeNodeInfo {
	return r.nodeInfo
}

func (r LoadStrikeRunResult) TestInfo() LoadStrikeTestInfo {
	return r.testInfo
}

func (r LoadStrikeRunResult) MarshalJSON() ([]byte, error) {
	body, err := json.Marshal(r.toNative())
	if err != nil {
		return nil, err
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	decoded["StartedUtc"] = r.StartedUtc()
	decoded["CompletedUtc"] = r.CompletedUtc()
	delete(decoded, "Duration")
	delete(decoded, "NodeType")
	decoded["ScenarioDurationsMs"] = r.ScenarioDurationsMS()
	return json.Marshal(decoded)
}

func (r *LoadStrikeRunResult) UnmarshalJSON(data []byte) error {
	var native runResult
	if err := json.Unmarshal(data, &native); err != nil {
		return err
	}
	*r = newLoadStrikeRunResult(native)
	return nil
}

func toNativeThresholdResults(items []LoadStrikeThresholdResult) []thresholdResult {
	if len(items) == 0 {
		return nil
	}
	result := make([]thresholdResult, 0, len(items))
	for _, item := range items {
		result = append(result, item.toNative())
	}
	return result
}

func cloneStatusCodeCountsFloat(source map[string]float64) map[string]float64 {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]float64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneInt64Map(source map[string]int64) map[string]int64 {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]int64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneInt64MapFromFloatMap(source map[string]float64) map[string]int64 {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]int64, len(source))
	for key, value := range source {
		result[key] = int64(value + 0.5)
	}
	return result
}

func cloneFloatMapFromInt64Map(source map[string]int64) map[string]float64 {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]float64, len(source))
	for key, value := range source {
		result[key] = float64(value)
	}
	return result
}

func toNativeRuntimePolicyErrors(items []LoadStrikeRuntimePolicyError) []RuntimePolicyError {
	if len(items) == 0 {
		return nil
	}
	result := make([]RuntimePolicyError, 0, len(items))
	for _, item := range items {
		result = append(result, item.toNative())
	}
	return result
}

func newPublicCorrelationRows(rows []correlationRow) []map[string]any {
	if len(rows) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.toMap())
	}
	return result
}

func toNativeCorrelationRows(rows []map[string]any) []correlationRow {
	if len(rows) == 0 {
		return nil
	}
	result := make([]correlationRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, newCorrelationRowFromMap(row))
	}
	return result
}
