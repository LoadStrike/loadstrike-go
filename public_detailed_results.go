package loadstrike

import (
	"encoding/json"
	"fmt"
	"time"
)

// LoadStrikeDetailedStepStats mirrors the .NET public detailed step-stats wrapper.
type LoadStrikeDetailedStepStats struct {
	scenarioName    string
	stepName        string
	allBytes        int64
	allRequestCount int
	allOKCount      int
	allFailCount    int
	totalBytes      int64
	totalLatencyMS  float64
	avgLatencyMS    float64
	minLatencyMS    float64
	maxLatencyMS    float64
	statusCodes     map[string]int
	allMeasurement  *LoadStrikeMeasurementStats
	ok              LoadStrikeMeasurementStats
	fail            LoadStrikeMeasurementStats
	sortIndex       int
	lookupFlavor    statsLookupFlavor `json:"-"`
}

func newLoadStrikeDetailedStepStats(native stepStats) LoadStrikeDetailedStepStats {
	result := LoadStrikeDetailedStepStats{
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
	if native.AllMeasurement != nil {
		all := newLoadStrikeMeasurementStats(*native.AllMeasurement)
		result.allMeasurement = &all
	}
	return result
}

func (s LoadStrikeDetailedStepStats) toNative() stepStats {
	native := stepStats{
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
	if s.allMeasurement != nil {
		all := s.allMeasurement.toNative()
		native.AllMeasurement = &all
	}
	return native
}

// MarshalJSON serializes the detailed step projection without exposing mutable fields.
func (s LoadStrikeDetailedStepStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toNative())
}

// UnmarshalJSON rehydrates the detailed step projection from its public wire shape.
func (s *LoadStrikeDetailedStepStats) UnmarshalJSON(data []byte) error {
	var native stepStats
	if err := json.Unmarshal(data, &native); err != nil {
		return err
	}
	*s = newLoadStrikeDetailedStepStats(native)
	return nil
}

// ScenarioName exposes the scenario name operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) ScenarioName() string { return s.scenarioName }

// StepName exposes the step name operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) StepName() string { return s.stepName }

// AllRequestCount exposes the all request count operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) AllRequestCount() int { return s.allRequestCount }

// OkCount exposes the ok count operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) OkCount() int { return s.allOKCount }

// FailCount exposes the fail count operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) FailCount() int { return s.allFailCount }

// TotalBytes exposes the total bytes operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) TotalBytes() int64 { return s.totalBytes }

// TotalLatencyMS exposes the total latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) TotalLatencyMS() float64 {
	return s.totalLatencyMS
}

// AvgLatencyMS exposes the avg latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) AvgLatencyMS() float64 { return s.avgLatencyMS }

// MinLatencyMS exposes the min latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) MinLatencyMS() float64 { return s.minLatencyMS }

// MaxLatencyMS exposes the max latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) MaxLatencyMS() float64 { return s.maxLatencyMS }

// StatusCodes exposes the status codes operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) StatusCodes() map[string]int {
	return cloneStatusCodeCounts(s.statusCodes)
}

// Ok exposes the ok operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) Ok() LoadStrikeMeasurementStats { return s.ok }

// Fail exposes the fail operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) Fail() LoadStrikeMeasurementStats { return s.fail }

// AllMeasurement returns the cumulative V2 measurement across successful and failed outcomes.
func (s LoadStrikeDetailedStepStats) AllMeasurement() *LoadStrikeMeasurementStats {
	if s.allMeasurement == nil {
		return nil
	}
	value := *s.allMeasurement
	return &value
}

// SortIndex exposes the sort index operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedStepStats) SortIndex() int { return s.sortIndex }

// FindStepStats finds step stats. Use this when you want an optional lookup from SDK state.
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

// GetStepStats returns step stats. Use this when you need to inspect SDK state.
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
	scenarioName         string
	allBytes             int64
	allRequestCount      int
	allOKCount           int
	allFailCount         int
	totalBytes           int64
	totalLatencyMS       float64
	avgLatencyMS         float64
	minLatencyMS         float64
	maxLatencyMS         float64
	statusCodes          map[string]int
	allMeasurement       *LoadStrikeMeasurementStats
	currentOperation     string
	currentOperationType LoadStrikeOperationType
	durationMS           float64
	duration             time.Duration
	ok                   LoadStrikeMeasurementStats
	fail                 LoadStrikeMeasurementStats
	loadSimulationStats  LoadStrikeLoadSimulationStats
	sortIndex            int
	stepStats            []LoadStrikeDetailedStepStats
	lookupFlavor         statsLookupFlavor `json:"-"`
}

func newLoadStrikeDetailedScenarioStats(native scenarioStats) LoadStrikeDetailedScenarioStats {
	normalizeScenarioStats(&native)
	steps := make([]LoadStrikeDetailedStepStats, 0, len(native.stepStats))
	for _, step := range native.stepStats {
		steps = append(steps, newLoadStrikeDetailedStepStats(step))
	}
	result := LoadStrikeDetailedScenarioStats{
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
	if native.AllMeasurement != nil {
		all := newLoadStrikeMeasurementStats(*native.AllMeasurement)
		result.allMeasurement = &all
	}
	return result
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
	if s.allMeasurement != nil {
		all := s.allMeasurement.toNative()
		native.AllMeasurement = &all
	}
	normalizeScenarioStats(&native)
	return native
}

// MarshalJSON serializes the detailed scenario projection without exposing mutable fields.
func (s LoadStrikeDetailedScenarioStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toNative())
}

// UnmarshalJSON rehydrates the detailed scenario projection from its public wire shape.
func (s *LoadStrikeDetailedScenarioStats) UnmarshalJSON(data []byte) error {
	var native scenarioStats
	if err := json.Unmarshal(data, &native); err != nil {
		return err
	}
	*s = newLoadStrikeDetailedScenarioStats(native)
	return nil
}

// ScenarioName exposes the scenario name operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) ScenarioName() string { return s.scenarioName }

// AllBytes exposes the all bytes operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) AllBytes() int64 { return s.allBytes }

// AllRequestCount exposes the all request count operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) AllRequestCount() int { return s.allRequestCount }

// AllOKCount exposes the all ok count operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) AllOKCount() int { return s.allOKCount }

// AllFailCount exposes the all fail count operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) AllFailCount() int { return s.allFailCount }

// TotalBytes exposes the total bytes operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) TotalBytes() int64 { return s.totalBytes }

// TotalLatencyMS exposes the total latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) TotalLatencyMS() float64 {
	return s.totalLatencyMS
}

// AvgLatencyMS exposes the avg latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) AvgLatencyMS() float64 { return s.avgLatencyMS }

// MinLatencyMS exposes the min latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) MinLatencyMS() float64 { return s.minLatencyMS }

// MaxLatencyMS exposes the max latency ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) MaxLatencyMS() float64 { return s.maxLatencyMS }

// StatusCodes exposes the status codes operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) StatusCodes() map[string]int {
	return cloneStatusCodeCounts(s.statusCodes)
}

// CurrentOperation exposes the current operation operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) CurrentOperation() string {
	return s.currentOperation
}

// CurrentOperationType exposes the current operation type operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) CurrentOperationType() LoadStrikeOperationType {
	return s.currentOperationType
}

// DurationMS exposes the duration ms operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) DurationMS() float64 { return s.durationMS }

// Ok exposes the ok operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) Ok() LoadStrikeMeasurementStats { return s.ok }

// Fail exposes the fail operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) Fail() LoadStrikeMeasurementStats {
	return s.fail
}

// AllMeasurement returns the cumulative V2 measurement across successful and failed outcomes.
func (s LoadStrikeDetailedScenarioStats) AllMeasurement() *LoadStrikeMeasurementStats {
	if s.allMeasurement == nil {
		return nil
	}
	value := *s.allMeasurement
	return &value
}

// LoadSimulationStats loads simulation stats. Use this when configuration or runtime state should be read from external input.
func (s LoadStrikeDetailedScenarioStats) LoadSimulationStats() LoadStrikeLoadSimulationStats {
	return s.loadSimulationStats
}

// SortIndex exposes the sort index operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) SortIndex() int { return s.sortIndex }

// FindStepStats finds step stats. Use this when you want an optional lookup from SDK state.
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

// GetStepStats returns step stats. Use this when you need to inspect SDK state.
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

// StepStats exposes the step stats operation. Use this when interacting with the SDK through this surface.
func (s LoadStrikeDetailedScenarioStats) StepStats() []LoadStrikeDetailedStepStats {
	return append([]LoadStrikeDetailedStepStats(nil), s.stepStats...)
}

// LoadStrikeRunResult mirrors the .NET public final run-result wrapper.
type LoadStrikeRunResult struct {
	startedUTC               time.Time
	completedUTC             time.Time
	durationMS               float64
	duration                 time.Duration
	allBytes                 int64
	allRequestCount          int
	allOKCount               int
	allFailCount             int
	failedThresholds         int
	nodeType                 LoadStrikeNodeType
	nodeInfo                 LoadStrikeNodeInfo
	testInfo                 LoadStrikeTestInfo
	thresholds               []LoadStrikeThresholdResult
	thresholdResults         []LoadStrikeThresholdResult
	metricStats              LoadStrikeMetricStats
	metrics                  []LoadStrikeMetricValue
	scenarioStats            []LoadStrikeDetailedScenarioStats
	stepStats                []LoadStrikeDetailedStepStats
	scenarioDurationsMS      map[string]int64
	pluginsData              []LoadStrikePluginData
	disabledSinks            []string
	sinkErrors               []LoadStrikeSinkError
	reportFiles              []string
	logFiles                 []string
	policyErrors             []LoadStrikeRuntimePolicyError
	generatorWarnings        []LoadStrikeGeneratorWarning
	observationDeliveryStats LoadStrikeObservationDeliveryStats
	reportingComplete        bool
	correlationRows          []map[string]any
	failedCorrelationRows    []map[string]any
	reportTrace              *reportTrace `json:"-"`
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
		startedUTC:               native.StartedUTC,
		completedUTC:             native.CompletedUTC,
		durationMS:               native.DurationMS,
		duration:                 native.Duration,
		allBytes:                 native.AllBytes,
		allRequestCount:          native.AllRequestCount,
		allOKCount:               native.AllOKCount,
		allFailCount:             native.AllFailCount,
		failedThresholds:         native.FailedThresholds,
		nodeType:                 native.NodeType,
		nodeInfo:                 newLoadStrikeNodeInfo(native.nodeInfo),
		testInfo:                 newLoadStrikeTestInfo(native.testInfo),
		thresholds:               toLoadStrikeThresholdResultSlice(native.Thresholds),
		thresholdResults:         toLoadStrikeThresholdResultSlice(native.ThresholdResults),
		metricStats:              newLoadStrikeMetricStats(native.metricStats),
		metrics:                  metrics,
		scenarioStats:            scenarios,
		stepStats:                steps,
		scenarioDurationsMS:      cloneInt64MapFromFloatMap(native.ScenarioDurationsMS),
		pluginsData:              toLoadStrikePluginDataSlice(native.PluginsData),
		disabledSinks:            append([]string(nil), native.DisabledSinks...),
		sinkErrors:               sinkErrors,
		reportFiles:              append([]string(nil), native.ReportFiles...),
		logFiles:                 append([]string(nil), native.LogFiles...),
		policyErrors:             policyErrors,
		generatorWarnings:        append([]LoadStrikeGeneratorWarning(nil), native.GeneratorWarnings...),
		observationDeliveryStats: native.ObservationDeliveryStats,
		reportingComplete:        native.ReportingComplete,
		correlationRows:          newPublicCorrelationRows(native.CorrelationRows),
		failedCorrelationRows:    newPublicCorrelationRows(native.FailedCorrelationRows),
		reportTrace:              native.reportTrace,
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
		StartedUTC:               r.startedUTC,
		CompletedUTC:             r.completedUTC,
		DurationMS:               r.durationMS,
		Duration:                 r.duration,
		AllBytes:                 r.allBytes,
		AllRequestCount:          r.allRequestCount,
		AllOKCount:               r.allOKCount,
		AllFailCount:             r.allFailCount,
		FailedThresholds:         r.failedThresholds,
		NodeType:                 r.nodeType,
		nodeInfo:                 r.nodeInfo.toNative(),
		testInfo:                 r.testInfo.toNative(),
		Thresholds:               toNativeThresholdResults(r.thresholds),
		ThresholdResults:         toNativeThresholdResults(r.thresholdResults),
		metricStats:              r.metricStats.toNative(),
		Metrics:                  metrics,
		scenarioStats:            scenarios,
		stepStats:                steps,
		ScenarioDurationsMS:      cloneFloatMapFromInt64Map(r.scenarioDurationsMS),
		PluginsData:              plugins,
		DisabledSinks:            append([]string(nil), r.disabledSinks...),
		SinkErrors:               sinkErrors,
		ReportFiles:              append([]string(nil), r.reportFiles...),
		LogFiles:                 append([]string(nil), r.logFiles...),
		PolicyErrors:             toNativeRuntimePolicyErrors(r.policyErrors),
		GeneratorWarnings:        append([]LoadStrikeGeneratorWarning(nil), r.generatorWarnings...),
		ObservationDeliveryStats: r.observationDeliveryStats,
		ReportingComplete:        r.reportingComplete,
		CorrelationRows:          toNativeCorrelationRows(r.correlationRows),
		FailedCorrelationRows:    toNativeCorrelationRows(r.failedCorrelationRows),
		reportTrace:              r.reportTrace,
	}
	normalizeRunResult(&native)
	return native
}

// StartedUtc exposes the started utc operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) StartedUtc() string {
	if r.startedUTC.IsZero() {
		return ""
	}
	return r.startedUTC.Format(time.RFC3339Nano)
}

// CompletedUtc exposes the completed utc operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) CompletedUtc() string {
	if r.completedUTC.IsZero() {
		return ""
	}
	return r.completedUTC.Format(time.RFC3339Nano)
}

// DurationMS exposes the duration ms operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) DurationMS() float64 { return r.durationMS }

// AllBytes exposes the all bytes operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) AllBytes() int64 { return r.allBytes }

// AllRequestCount exposes the all request count operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) AllRequestCount() int { return r.allRequestCount }

// AllOKCount exposes the all ok count operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) AllOKCount() int { return r.allOKCount }

// AllFailCount exposes the all fail count operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) AllFailCount() int { return r.allFailCount }

// FailedThresholds exposes the failed thresholds operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) FailedThresholds() int { return r.failedThresholds }

// Thresholds exposes the thresholds operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) Thresholds() []LoadStrikeThresholdResult {
	return append([]LoadStrikeThresholdResult(nil), r.thresholds...)
}

// ThresholdResults exposes the threshold results operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) ThresholdResults() []LoadStrikeThresholdResult {
	return append([]LoadStrikeThresholdResult(nil), r.thresholdResults...)
}

// Metrics exposes the metrics operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) Metrics() []LoadStrikeMetricValue {
	return append([]LoadStrikeMetricValue(nil), r.metrics...)
}

// ScenarioDurationsMS exposes the scenario durations ms operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) ScenarioDurationsMS() map[string]int64 {
	return cloneInt64Map(r.scenarioDurationsMS)
}

// PluginsData exposes the plugins data operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) PluginsData() []LoadStrikePluginData {
	return append([]LoadStrikePluginData(nil), r.pluginsData...)
}

// DisabledSinks exposes the disabled sinks operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) DisabledSinks() []string {
	return append([]string(nil), r.disabledSinks...)
}

// SinkErrors exposes the sink errors operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) SinkErrors() []LoadStrikeSinkError {
	return append([]LoadStrikeSinkError(nil), r.sinkErrors...)
}

// ReportFiles exposes the report files operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) ReportFiles() []string {
	return append([]string(nil), r.reportFiles...)
}

// LogFiles exposes the log files operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) LogFiles() []string {
	return append([]string(nil), r.logFiles...)
}

// PolicyErrors exposes the policy errors operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) PolicyErrors() []LoadStrikeRuntimePolicyError {
	return append([]LoadStrikeRuntimePolicyError(nil), r.policyErrors...)
}

// GeneratorWarnings exposes structured generator and reporting warnings.
func (r LoadStrikeRunResult) GeneratorWarnings() []LoadStrikeGeneratorWarning {
	return append([]LoadStrikeGeneratorWarning(nil), r.generatorWarnings...)
}

// ObservationDeliveryStats exposes raw observation capture and delivery totals.
func (r LoadStrikeRunResult) ObservationDeliveryStats() LoadStrikeObservationDeliveryStats {
	return r.observationDeliveryStats
}

// ReportingComplete reports whether every configured raw observation stream
// completed without buffer, queue, or delivery loss.
func (r LoadStrikeRunResult) ReportingComplete() bool {
	return r.reportingComplete
}

// CorrelationRows exposes the correlation rows operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) CorrelationRows() []map[string]any {
	rows := make([]map[string]any, 0, len(r.correlationRows))
	for _, row := range r.correlationRows {
		rows = append(rows, cloneAnyMap(row))
	}
	return rows
}

// FailedCorrelationRows exposes the failed correlation rows operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) FailedCorrelationRows() []map[string]any {
	rows := make([]map[string]any, 0, len(r.failedCorrelationRows))
	for _, row := range r.failedCorrelationRows {
		rows = append(rows, cloneAnyMap(row))
	}
	return rows
}

// FindScenarioStats finds scenario stats. Use this when you want an optional lookup from SDK state.
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

// GetScenarioStats returns scenario stats. Use this when you need to inspect SDK state.
func (r LoadStrikeRunResult) GetScenarioStats(scenarioName string) *LoadStrikeDetailedScenarioStats {
	value := r.FindScenarioStats(scenarioName)
	if value == nil {
		panic(fmt.Sprintf("Scenario '%s' was not found.", scenarioName))
	}
	return value
}

// ScenarioStats exposes the scenario stats operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) ScenarioStats() []LoadStrikeDetailedScenarioStats {
	return append([]LoadStrikeDetailedScenarioStats(nil), r.scenarioStats...)
}

// StepStats exposes the step stats operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) StepStats() []LoadStrikeDetailedStepStats {
	return append([]LoadStrikeDetailedStepStats(nil), r.stepStats...)
}

// MetricStats exposes the metric stats operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) MetricStats() LoadStrikeMetricStats {
	return r.metricStats
}

// NodeInfo exposes the node info operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) NodeInfo() LoadStrikeNodeInfo {
	return r.nodeInfo
}

// TestInfo exposes the test info operation. Use this when interacting with the SDK through this surface.
func (r LoadStrikeRunResult) TestInfo() LoadStrikeTestInfo {
	return r.testInfo
}

// MarshalJSON serializes the current value to json. Use this when bridging SDK models to JSON payloads.
func (r LoadStrikeRunResult) MarshalJSON() ([]byte, error) {
	body, err := json.Marshal(r.toNative())
	if err != nil {
		return nil, err
	}
	decoded := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	if r.startedUTC.IsZero() {
		decoded["StartedUtc"] = json.RawMessage("null")
	}
	if r.completedUTC.IsZero() {
		decoded["CompletedUtc"] = json.RawMessage("null")
	}
	delete(decoded, "Duration")
	delete(decoded, "NodeType")
	scenarioDurationsMS, err := json.Marshal(r.ScenarioDurationsMS())
	if err != nil {
		return nil, err
	}
	decoded["ScenarioDurationsMs"] = scenarioDurationsMS
	return json.Marshal(decoded)
}

// UnmarshalJSON populates the current value from json. Use this when rehydrating SDK models from JSON payloads.
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
