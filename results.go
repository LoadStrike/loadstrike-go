package loadstrike

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type statsLookupFlavor uint8

const (
	statsLookupFlavorDetailed statsLookupFlavor = iota
	statsLookupFlavorRealtime
)

// runResult is the public runtime result returned by runner execution.
type runResult struct {
	StartedUTC            time.Time            `json:"StartedUtc"`
	CompletedUTC          time.Time            `json:"CompletedUtc"`
	DurationMS            float64              `json:"DurationMs,omitempty"`
	Duration              time.Duration        `json:"Duration,omitempty"`
	AllBytes              int64                `json:"AllBytes,omitempty"`
	AllRequestCount       int                  `json:"AllRequestCount"`
	AllOKCount            int                  `json:"AllOkCount"`
	AllFailCount          int                  `json:"AllFailCount"`
	FailedThresholds      int                  `json:"FailedThresholds,omitempty"`
	NodeType              NodeType             `json:"NodeType,omitempty"`
	nodeInfo              nodeInfo             `json:"nodeInfo,omitempty"`
	testInfo              testInfo             `json:"testInfo"`
	Thresholds            []thresholdResult    `json:"Thresholds,omitempty"`
	ThresholdResults      []thresholdResult    `json:"ThresholdResults,omitempty"`
	metricStats           metricStats          `json:"metricStats,omitempty"`
	Metrics               []metricResult       `json:"Metrics,omitempty"`
	scenarioStats         []scenarioStats      `json:"scenarioStats"`
	stepStats             []stepStats          `json:"stepStats,omitempty"`
	ScenarioDurationsMS   map[string]float64   `json:"ScenarioDurationsMs,omitempty"`
	PluginsData           []pluginData         `json:"PluginsData,omitempty"`
	DisabledSinks         []string             `json:"DisabledSinks,omitempty"`
	SinkErrors            []sinkErrorResult    `json:"SinkErrors,omitempty"`
	ReportFiles           []string             `json:"ReportFiles,omitempty"`
	LogFiles              []string             `json:"LogFiles,omitempty"`
	PolicyErrors          []RuntimePolicyError `json:"PolicyErrors,omitempty"`
	CorrelationRows       []correlationRow     `json:"CorrelationRows,omitempty"`
	FailedCorrelationRows []correlationRow     `json:"FailedCorrelationRows,omitempty"`
	reportTrace           *reportTrace
}

// MarshalJSON serializes the current value to json. Use this when bridging SDK models to JSON payloads.
func (r runResult) MarshalJSON() ([]byte, error) {
	normalized := r
	normalizeRunResult(&normalized)
	type payload struct {
		StartedUTC            time.Time            `json:"StartedUtc"`
		CompletedUTC          time.Time            `json:"CompletedUtc"`
		DurationMS            float64              `json:"DurationMs,omitempty"`
		Duration              time.Duration        `json:"Duration,omitempty"`
		AllBytes              int64                `json:"AllBytes,omitempty"`
		AllRequestCount       int                  `json:"AllRequestCount"`
		AllOKCount            int                  `json:"AllOkCount"`
		AllFailCount          int                  `json:"AllFailCount"`
		FailedThresholds      int                  `json:"FailedThresholds,omitempty"`
		NodeType              NodeType             `json:"NodeType,omitempty"`
		NodeInfo              nodeInfo             `json:"nodeInfo,omitempty"`
		TestInfo              testInfo             `json:"testInfo"`
		Thresholds            []thresholdResult    `json:"Thresholds,omitempty"`
		ThresholdResults      []thresholdResult    `json:"ThresholdResults,omitempty"`
		MetricStats           metricStats          `json:"metricStats,omitempty"`
		Metrics               []metricResult       `json:"Metrics,omitempty"`
		ScenarioStats         []scenarioStats      `json:"scenarioStats"`
		StepStats             []stepStats          `json:"stepStats,omitempty"`
		ScenarioDurationsMS   map[string]float64   `json:"ScenarioDurationsMs,omitempty"`
		PluginsData           []pluginData         `json:"PluginsData,omitempty"`
		DisabledSinks         []string             `json:"DisabledSinks,omitempty"`
		SinkErrors            []sinkErrorResult    `json:"SinkErrors,omitempty"`
		ReportFiles           []string             `json:"ReportFiles,omitempty"`
		LogFiles              []string             `json:"LogFiles,omitempty"`
		PolicyErrors          []RuntimePolicyError `json:"PolicyErrors,omitempty"`
		CorrelationRows       []correlationRow     `json:"CorrelationRows,omitempty"`
		FailedCorrelationRows []correlationRow     `json:"FailedCorrelationRows,omitempty"`
	}
	return json.Marshal(payload{
		StartedUTC:            normalized.StartedUTC,
		CompletedUTC:          normalized.CompletedUTC,
		DurationMS:            normalized.DurationMS,
		Duration:              normalized.Duration,
		AllBytes:              normalized.AllBytes,
		AllRequestCount:       normalized.AllRequestCount,
		AllOKCount:            normalized.AllOKCount,
		AllFailCount:          normalized.AllFailCount,
		FailedThresholds:      normalized.FailedThresholds,
		NodeType:              normalized.NodeType,
		NodeInfo:              normalized.nodeInfo,
		TestInfo:              normalized.testInfo,
		Thresholds:            append([]thresholdResult(nil), normalized.Thresholds...),
		ThresholdResults:      append([]thresholdResult(nil), normalized.ThresholdResults...),
		MetricStats:           normalized.metricStats,
		Metrics:               append([]metricResult(nil), normalized.Metrics...),
		ScenarioStats:         append([]scenarioStats(nil), normalized.scenarioStats...),
		StepStats:             append([]stepStats(nil), normalized.stepStats...),
		ScenarioDurationsMS:   cloneFloatMap(normalized.ScenarioDurationsMS),
		PluginsData:           append([]pluginData(nil), normalized.PluginsData...),
		DisabledSinks:         append([]string(nil), normalized.DisabledSinks...),
		SinkErrors:            append([]sinkErrorResult(nil), normalized.SinkErrors...),
		ReportFiles:           append([]string(nil), normalized.ReportFiles...),
		LogFiles:              append([]string(nil), normalized.LogFiles...),
		PolicyErrors:          append([]RuntimePolicyError(nil), normalized.PolicyErrors...),
		CorrelationRows:       append([]correlationRow(nil), normalized.CorrelationRows...),
		FailedCorrelationRows: append([]correlationRow(nil), normalized.FailedCorrelationRows...),
	})
}

// UnmarshalJSON populates the current value from json. Use this when rehydrating SDK models from JSON payloads.
func (r *runResult) UnmarshalJSON(data []byte) error {
	type payload struct {
		StartedUTC            time.Time            `json:"StartedUtc"`
		CompletedUTC          time.Time            `json:"CompletedUtc"`
		DurationMS            float64              `json:"DurationMs,omitempty"`
		Duration              time.Duration        `json:"Duration,omitempty"`
		AllBytes              int64                `json:"AllBytes,omitempty"`
		AllRequestCount       int                  `json:"AllRequestCount"`
		AllOKCount            int                  `json:"AllOkCount"`
		AllFailCount          int                  `json:"AllFailCount"`
		FailedThresholds      int                  `json:"FailedThresholds,omitempty"`
		NodeType              NodeType             `json:"NodeType,omitempty"`
		NodeInfo              nodeInfo             `json:"nodeInfo,omitempty"`
		TestInfo              testInfo             `json:"testInfo"`
		Thresholds            []thresholdResult    `json:"Thresholds,omitempty"`
		ThresholdResults      []thresholdResult    `json:"ThresholdResults,omitempty"`
		MetricStats           metricStats          `json:"metricStats,omitempty"`
		Metrics               []metricResult       `json:"Metrics,omitempty"`
		ScenarioStats         []scenarioStats      `json:"scenarioStats"`
		StepStats             []stepStats          `json:"stepStats,omitempty"`
		ScenarioDurationsMS   map[string]float64   `json:"ScenarioDurationsMs,omitempty"`
		PluginsData           []pluginData         `json:"PluginsData,omitempty"`
		DisabledSinks         []string             `json:"DisabledSinks,omitempty"`
		SinkErrors            []sinkErrorResult    `json:"SinkErrors,omitempty"`
		ReportFiles           []string             `json:"ReportFiles,omitempty"`
		LogFiles              []string             `json:"LogFiles,omitempty"`
		PolicyErrors          []RuntimePolicyError `json:"PolicyErrors,omitempty"`
		CorrelationRows       []correlationRow     `json:"CorrelationRows,omitempty"`
		FailedCorrelationRows []correlationRow     `json:"FailedCorrelationRows,omitempty"`
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*r = runResult{
		StartedUTC:            decoded.StartedUTC,
		CompletedUTC:          decoded.CompletedUTC,
		DurationMS:            decoded.DurationMS,
		Duration:              decoded.Duration,
		AllBytes:              decoded.AllBytes,
		AllRequestCount:       decoded.AllRequestCount,
		AllOKCount:            decoded.AllOKCount,
		AllFailCount:          decoded.AllFailCount,
		FailedThresholds:      decoded.FailedThresholds,
		NodeType:              decoded.NodeType,
		nodeInfo:              decoded.NodeInfo,
		testInfo:              decoded.TestInfo,
		Thresholds:            append([]thresholdResult(nil), decoded.Thresholds...),
		ThresholdResults:      append([]thresholdResult(nil), decoded.ThresholdResults...),
		metricStats:           decoded.MetricStats,
		Metrics:               append([]metricResult(nil), decoded.Metrics...),
		scenarioStats:         append([]scenarioStats(nil), decoded.ScenarioStats...),
		stepStats:             append([]stepStats(nil), decoded.StepStats...),
		ScenarioDurationsMS:   cloneFloatMap(decoded.ScenarioDurationsMS),
		PluginsData:           append([]pluginData(nil), decoded.PluginsData...),
		DisabledSinks:         append([]string(nil), decoded.DisabledSinks...),
		SinkErrors:            append([]sinkErrorResult(nil), decoded.SinkErrors...),
		ReportFiles:           append([]string(nil), decoded.ReportFiles...),
		LogFiles:              append([]string(nil), decoded.LogFiles...),
		PolicyErrors:          append([]RuntimePolicyError(nil), decoded.PolicyErrors...),
		CorrelationRows:       append([]correlationRow(nil), decoded.CorrelationRows...),
		FailedCorrelationRows: append([]correlationRow(nil), decoded.FailedCorrelationRows...),
	}
	normalizeRunResult(r)
	return nil
}

// FindScenarioStats returns the named scenario stats when present.
func (r runResult) FindScenarioStats(scenarioName string) *scenarioStats {
	validateScenarioStatsLookupName(scenarioName)
	for index := range r.scenarioStats {
		if r.scenarioStats[index].ScenarioName == scenarioName {
			normalizeScenarioStats(&r.scenarioStats[index])
			return &r.scenarioStats[index]
		}
	}
	return nil
}

// GetScenarioStats returns the named scenario stats or panics when missing.
func (r runResult) GetScenarioStats(scenarioName string) *scenarioStats {
	value := r.FindScenarioStats(scenarioName)
	if value == nil {
		panic(fmt.Sprintf("Scenario '%s' was not found.", scenarioName))
	}
	return value
}

// scenarioStats captures top-level per-scenario request counters.
type scenarioStats struct {
	ScenarioName         string                  `json:"ScenarioName"`
	AllBytes             int64                   `json:"AllBytes,omitempty"`
	AllRequestCount      int                     `json:"AllRequestCount"`
	AllOKCount           int                     `json:"AllOkCount"`
	AllFailCount         int                     `json:"AllFailCount"`
	TotalBytes           int64                   `json:"TotalBytes,omitempty"`
	TotalLatencyMS       float64                 `json:"TotalLatencyMs,omitempty"`
	AvgLatencyMS         float64                 `json:"AvgLatencyMs,omitempty"`
	MinLatencyMS         float64                 `json:"MinLatencyMs,omitempty"`
	MaxLatencyMS         float64                 `json:"MaxLatencyMs,omitempty"`
	StatusCodes          map[string]int          `json:"StatusCodes,omitempty"`
	CurrentOperation     string                  `json:"CurrentOperation,omitempty"`
	CurrentOperationType LoadStrikeOperationType `json:"CurrentOperationType,omitempty"`
	DurationMS           float64                 `json:"DurationMs,omitempty"`
	Duration             time.Duration           `json:"Duration,omitempty"`
	Ok                   measurementStats        `json:"Ok,omitempty"`
	Fail                 measurementStats        `json:"Fail,omitempty"`
	LoadSimulationStats  loadSimulationStats     `json:"LoadSimulationStats,omitempty"`
	SortIndex            int                     `json:"SortIndex,omitempty"`
	stepStats            []stepStats             `json:"stepStats,omitempty"`
	lookupFlavor         statsLookupFlavor       `json:"-"`
}

// MarshalJSON serializes the current value to json. Use this when bridging SDK models to JSON payloads.
func (s scenarioStats) MarshalJSON() ([]byte, error) {
	normalized := s
	normalizeScenarioStats(&normalized)
	type payload struct {
		ScenarioName         string                  `json:"ScenarioName"`
		AllBytes             int64                   `json:"AllBytes,omitempty"`
		AllRequestCount      int                     `json:"AllRequestCount"`
		AllOKCount           int                     `json:"AllOkCount"`
		AllFailCount         int                     `json:"AllFailCount"`
		TotalBytes           int64                   `json:"TotalBytes,omitempty"`
		TotalLatencyMS       float64                 `json:"TotalLatencyMs,omitempty"`
		AvgLatencyMS         float64                 `json:"AvgLatencyMs,omitempty"`
		MinLatencyMS         float64                 `json:"MinLatencyMs,omitempty"`
		MaxLatencyMS         float64                 `json:"MaxLatencyMs,omitempty"`
		StatusCodes          map[string]int          `json:"StatusCodes,omitempty"`
		CurrentOperation     string                  `json:"CurrentOperation,omitempty"`
		CurrentOperationType LoadStrikeOperationType `json:"CurrentOperationType,omitempty"`
		DurationMS           float64                 `json:"DurationMs,omitempty"`
		Duration             time.Duration           `json:"Duration,omitempty"`
		Ok                   measurementStats        `json:"Ok,omitempty"`
		Fail                 measurementStats        `json:"Fail,omitempty"`
		LoadSimulationStats  loadSimulationStats     `json:"LoadSimulationStats,omitempty"`
		SortIndex            int                     `json:"SortIndex,omitempty"`
		StepStats            []stepStats             `json:"stepStats,omitempty"`
	}
	return json.Marshal(payload{
		ScenarioName:         normalized.ScenarioName,
		AllBytes:             normalized.AllBytes,
		AllRequestCount:      normalized.AllRequestCount,
		AllOKCount:           normalized.AllOKCount,
		AllFailCount:         normalized.AllFailCount,
		TotalBytes:           normalized.TotalBytes,
		TotalLatencyMS:       normalized.TotalLatencyMS,
		AvgLatencyMS:         normalized.AvgLatencyMS,
		MinLatencyMS:         normalized.MinLatencyMS,
		MaxLatencyMS:         normalized.MaxLatencyMS,
		StatusCodes:          cloneStatusCodeCounts(normalized.StatusCodes),
		CurrentOperation:     normalized.CurrentOperation,
		CurrentOperationType: normalized.CurrentOperationType,
		DurationMS:           normalized.DurationMS,
		Duration:             normalized.Duration,
		Ok:                   normalized.Ok,
		Fail:                 normalized.Fail,
		LoadSimulationStats:  normalized.LoadSimulationStats,
		SortIndex:            normalized.SortIndex,
		StepStats:            append([]stepStats(nil), normalized.stepStats...),
	})
}

// UnmarshalJSON populates the current value from json. Use this when rehydrating SDK models from JSON payloads.
func (s *scenarioStats) UnmarshalJSON(data []byte) error {
	type payload struct {
		ScenarioName         string                  `json:"ScenarioName"`
		AllBytes             int64                   `json:"AllBytes,omitempty"`
		AllRequestCount      int                     `json:"AllRequestCount"`
		AllOKCount           int                     `json:"AllOkCount"`
		AllFailCount         int                     `json:"AllFailCount"`
		TotalBytes           int64                   `json:"TotalBytes,omitempty"`
		TotalLatencyMS       float64                 `json:"TotalLatencyMs,omitempty"`
		AvgLatencyMS         float64                 `json:"AvgLatencyMs,omitempty"`
		MinLatencyMS         float64                 `json:"MinLatencyMs,omitempty"`
		MaxLatencyMS         float64                 `json:"MaxLatencyMs,omitempty"`
		StatusCodes          map[string]int          `json:"StatusCodes,omitempty"`
		CurrentOperation     string                  `json:"CurrentOperation,omitempty"`
		CurrentOperationType LoadStrikeOperationType `json:"CurrentOperationType,omitempty"`
		DurationMS           float64                 `json:"DurationMs,omitempty"`
		Duration             time.Duration           `json:"Duration,omitempty"`
		Ok                   measurementStats        `json:"Ok,omitempty"`
		Fail                 measurementStats        `json:"Fail,omitempty"`
		LoadSimulationStats  loadSimulationStats     `json:"LoadSimulationStats,omitempty"`
		SortIndex            int                     `json:"SortIndex,omitempty"`
		StepStats            []stepStats             `json:"stepStats,omitempty"`
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*s = scenarioStats{
		ScenarioName:         decoded.ScenarioName,
		AllBytes:             decoded.AllBytes,
		AllRequestCount:      decoded.AllRequestCount,
		AllOKCount:           decoded.AllOKCount,
		AllFailCount:         decoded.AllFailCount,
		TotalBytes:           decoded.TotalBytes,
		TotalLatencyMS:       decoded.TotalLatencyMS,
		AvgLatencyMS:         decoded.AvgLatencyMS,
		MinLatencyMS:         decoded.MinLatencyMS,
		MaxLatencyMS:         decoded.MaxLatencyMS,
		StatusCodes:          cloneStatusCodeCounts(decoded.StatusCodes),
		CurrentOperation:     decoded.CurrentOperation,
		CurrentOperationType: decoded.CurrentOperationType,
		DurationMS:           decoded.DurationMS,
		Duration:             decoded.Duration,
		Ok:                   decoded.Ok,
		Fail:                 decoded.Fail,
		LoadSimulationStats:  decoded.LoadSimulationStats,
		SortIndex:            decoded.SortIndex,
		stepStats:            append([]stepStats(nil), decoded.StepStats...),
	}
	normalizeScenarioStats(s)
	return nil
}

// FindStepStats returns the named step stats when present.
func (s scenarioStats) FindStepStats(stepName string) *stepStats {
	if s.lookupFlavor != statsLookupFlavorRealtime {
		validateStepStatsLookupName(stepName)
	}
	for index := range s.stepStats {
		if s.stepStats[index].StepName == stepName {
			s.stepStats[index].lookupFlavor = s.lookupFlavor
			return &s.stepStats[index]
		}
	}
	return nil
}

// GetStepStats returns the named step stats or panics when missing.
func (s scenarioStats) GetStepStats(stepName string) *stepStats {
	value := s.FindStepStats(stepName)
	if value == nil {
		if s.lookupFlavor == statsLookupFlavorRealtime {
			panic(fmt.Sprintf("Step stats not found: %s.", stepName))
		}
		panic(fmt.Sprintf("Step '%s' was not found for scenario '%s'.", stepName, s.ScenarioName))
	}
	return value
}

// stepStats captures flattened step counters.
type stepStats struct {
	ScenarioName    string            `json:"ScenarioName,omitempty"`
	StepName        string            `json:"StepName,omitempty"`
	AllBytes        int64             `json:"AllBytes,omitempty"`
	AllRequestCount int               `json:"AllRequestCount,omitempty"`
	AllOKCount      int               `json:"AllOkCount,omitempty"`
	AllFailCount    int               `json:"AllFailCount,omitempty"`
	TotalBytes      int64             `json:"TotalBytes,omitempty"`
	TotalLatencyMS  float64           `json:"TotalLatencyMs,omitempty"`
	AvgLatencyMS    float64           `json:"AvgLatencyMs,omitempty"`
	MinLatencyMS    float64           `json:"MinLatencyMs,omitempty"`
	MaxLatencyMS    float64           `json:"MaxLatencyMs,omitempty"`
	StatusCodes     map[string]int    `json:"StatusCodes,omitempty"`
	Ok              measurementStats  `json:"Ok,omitempty"`
	Fail            measurementStats  `json:"Fail,omitempty"`
	SortIndex       int               `json:"SortIndex,omitempty"`
	lookupFlavor    statsLookupFlavor `json:"-"`
}

// FindStepStats mirrors the .NET detailed-step self lookup behavior.
func (s stepStats) FindStepStats(stepName string) *stepStats {
	if s.lookupFlavor != statsLookupFlavorRealtime {
		validateStepStatsLookupName(stepName)
	}
	if s.StepName == stepName {
		return &s
	}
	return nil
}

// GetStepStats mirrors the .NET detailed-step self lookup behavior.
func (s stepStats) GetStepStats(stepName string) *stepStats {
	value := s.FindStepStats(stepName)
	if value == nil {
		if s.lookupFlavor == statsLookupFlavorRealtime {
			panic(fmt.Sprintf("Step stats not found: %s.", stepName))
		}
		panic(fmt.Sprintf("Step '%s' was not found.", stepName))
	}
	return value
}

// measurementStats captures request, transfer, latency, and status-code projections.
type measurementStats struct {
	Request      requestStats      `json:"Request,omitempty"`
	DataTransfer dataTransferStats `json:"DataTransfer,omitempty"`
	Latency      latencyStats      `json:"Latency,omitempty"`
	StatusCodes  []statusCodeStats `json:"StatusCodes,omitempty"`
}

// requestStats captures request count and rate.
type requestStats struct {
	Count   int     `json:"Count,omitempty"`
	Percent int     `json:"Percent,omitempty"`
	RPS     float64 `json:"RPS,omitempty"`
}

// dataTransferStats captures bytes distribution data.
type dataTransferStats struct {
	AllBytes  int64   `json:"AllBytes,omitempty"`
	MinBytes  int64   `json:"MinBytes,omitempty"`
	MaxBytes  int64   `json:"MaxBytes,omitempty"`
	MeanBytes int64   `json:"MeanBytes,omitempty"`
	Percent50 int64   `json:"Percent50,omitempty"`
	Percent75 int64   `json:"Percent75,omitempty"`
	Percent95 int64   `json:"Percent95,omitempty"`
	Percent99 int64   `json:"Percent99,omitempty"`
	StdDev    float64 `json:"StdDev,omitempty"`
}

// latencyStats captures latency distribution data in milliseconds.
type latencyCount struct {
	LessOrEq800     int `json:"LessOrEq800,omitempty"`
	More800Less1200 int `json:"More800Less1200,omitempty"`
	MoreOrEq1200    int `json:"MoreOrEq1200,omitempty"`
}

// latencyStats captures latency distribution data in milliseconds.
type latencyStats struct {
	LatencyCount latencyCount `json:"LatencyCount,omitempty"`
	MinMs        float64      `json:"MinMs,omitempty"`
	MaxMs        float64      `json:"MaxMs,omitempty"`
	MeanMs       float64      `json:"MeanMs,omitempty"`
	Percent50    float64      `json:"Percent50,omitempty"`
	Percent75    float64      `json:"Percent75,omitempty"`
	Percent95    float64      `json:"Percent95,omitempty"`
	Percent99    float64      `json:"Percent99,omitempty"`
	StdDev       float64      `json:"StdDev,omitempty"`
	P50Ms        float64      `json:"P50Ms,omitempty"`
	P75Ms        float64      `json:"P75Ms,omitempty"`
	P80Ms        float64      `json:"P80Ms,omitempty"`
	P85Ms        float64      `json:"P85Ms,omitempty"`
	P90Ms        float64      `json:"P90Ms,omitempty"`
	P95Ms        float64      `json:"P95Ms,omitempty"`
	P99Ms        float64      `json:"P99Ms,omitempty"`
	Samples      int          `json:"Samples,omitempty"`
}

// statusCodeStats captures grouped request counts by status code.
type statusCodeStats struct {
	StatusCode string `json:"StatusCode,omitempty"`
	Message    string `json:"Message,omitempty"`
	IsError    bool   `json:"IsError,omitempty"`
	Count      int    `json:"Count,omitempty"`
	Percent    int    `json:"Percent,omitempty"`
}

// loadSimulationStats captures the last active simulation projection.
type loadSimulationStats struct {
	SimulationName string `json:"SimulationName,omitempty"`
	Value          int    `json:"Value,omitempty"`
}

// metricStats captures counters and gauges registered during the run.
type metricStats struct {
	DurationMS float64        `json:"DurationMs,omitempty"`
	Duration   time.Duration  `json:"Duration,omitempty"`
	Counters   []counterStats `json:"Counters,omitempty"`
	Gauges     []gaugeStats   `json:"Gauges,omitempty"`
}

// counterStats captures counter metric values.
type counterStats struct {
	MetricName    string `json:"MetricName,omitempty"`
	ScenarioName  string `json:"ScenarioName,omitempty"`
	UnitOfMeasure string `json:"UnitOfMeasure,omitempty"`
	Value         int64  `json:"Value,omitempty"`
}

// gaugeStats captures gauge metric values.
type gaugeStats struct {
	MetricName    string  `json:"MetricName,omitempty"`
	ScenarioName  string  `json:"ScenarioName,omitempty"`
	UnitOfMeasure string  `json:"UnitOfMeasure,omitempty"`
	Value         float64 `json:"Value,omitempty"`
}

// runResponse is the public response contract produced by the parity runner.
type runResponse struct {
	CompletedUTC time.Time `json:"CompletedUtc"`
	Stats        nodeStats `json:"Stats"`
}

// nodeInfo captures node metadata surfaced in reports and run results.
type nodeInfo struct {
	NodeType             NodeType                `json:"NodeType,omitempty"`
	MachineName          string                  `json:"MachineName,omitempty"`
	CurrentOperation     string                  `json:"CurrentOperation,omitempty"`
	CurrentOperationType LoadStrikeOperationType `json:"CurrentOperationType,omitempty"`
	CoresCount           int                     `json:"CoresCount,omitempty"`
	DotNetVersion        string                  `json:"DotNetVersion,omitempty"`
	EngineVersion        string                  `json:"EngineVersion,omitempty"`
	OS                   string                  `json:"OS,omitempty"`
	Processor            string                  `json:"Processor,omitempty"`
}

// nodeStats captures top-level aggregated request counters.
type nodeStats struct {
	AllBytes        int64             `json:"AllBytes,omitempty"`
	AllRequestCount int               `json:"AllRequestCount"`
	AllOKCount      int               `json:"AllOkCount"`
	AllFailCount    int               `json:"AllFailCount"`
	DurationMS      float64           `json:"DurationMs,omitempty"`
	Duration        time.Duration     `json:"Duration,omitempty"`
	Metrics         metricStats       `json:"Metrics,omitempty"`
	nodeInfo        nodeInfo          `json:"nodeInfo,omitempty"`
	testInfo        testInfo          `json:"testInfo"`
	PluginsData     []pluginData      `json:"PluginsData,omitempty"`
	scenarioStats   []scenarioStats   `json:"scenarioStats,omitempty"`
	Scenarios       []scenarioStats   `json:"-"`
	Thresholds      []thresholdResult `json:"Thresholds,omitempty"`
}

// MarshalJSON serializes the current value to json. Use this when bridging SDK models to JSON payloads.
func (n nodeStats) MarshalJSON() ([]byte, error) {
	normalized := n
	normalizeNodeStats(&normalized)
	type payload nodeStats
	return json.Marshal(payload(normalized))
}

// UnmarshalJSON populates the current value from json. Use this when rehydrating SDK models from JSON payloads.
func (n *nodeStats) UnmarshalJSON(data []byte) error {
	type payload struct {
		AllBytes        int64             `json:"AllBytes,omitempty"`
		AllRequestCount int               `json:"AllRequestCount"`
		AllOKCount      int               `json:"AllOkCount"`
		AllFailCount    int               `json:"AllFailCount"`
		DurationMS      float64           `json:"DurationMs,omitempty"`
		Duration        time.Duration     `json:"Duration,omitempty"`
		Metrics         metricStats       `json:"Metrics,omitempty"`
		nodeInfo        nodeInfo          `json:"nodeInfo,omitempty"`
		testInfo        testInfo          `json:"testInfo"`
		PluginsData     []pluginData      `json:"PluginsData,omitempty"`
		scenarioStats   []scenarioStats   `json:"scenarioStats,omitempty"`
		Scenarios       []scenarioStats   `json:"Scenarios,omitempty"`
		Thresholds      []thresholdResult `json:"Thresholds,omitempty"`
	}
	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	n.AllBytes = decoded.AllBytes
	n.AllRequestCount = decoded.AllRequestCount
	n.AllOKCount = decoded.AllOKCount
	n.AllFailCount = decoded.AllFailCount
	n.DurationMS = decoded.DurationMS
	n.Duration = decoded.Duration
	n.Metrics = decoded.Metrics
	n.nodeInfo = decoded.nodeInfo
	n.testInfo = decoded.testInfo
	n.PluginsData = append([]pluginData(nil), decoded.PluginsData...)
	n.scenarioStats = append([]scenarioStats(nil), decoded.scenarioStats...)
	if len(n.scenarioStats) == 0 {
		n.scenarioStats = append([]scenarioStats(nil), decoded.Scenarios...)
	}
	n.Scenarios = append([]scenarioStats(nil), n.scenarioStats...)
	n.Thresholds = append([]thresholdResult(nil), decoded.Thresholds...)
	normalizeNodeStats(n)
	return nil
}

// FindScenarioStats returns the named scenario stats when present.
func (n nodeStats) FindScenarioStats(scenarioName string) *scenarioStats {
	scenarios := normalizedNodeScenarioStats(n)
	for index := range scenarios {
		if scenarios[index].ScenarioName == scenarioName {
			scenarios[index].lookupFlavor = statsLookupFlavorRealtime
			for stepIndex := range scenarios[index].stepStats {
				scenarios[index].stepStats[stepIndex].lookupFlavor = statsLookupFlavorRealtime
			}
			normalizeScenarioStats(&scenarios[index])
			return &scenarios[index]
		}
	}
	return nil
}

// GetScenarioStats returns the named scenario stats or panics when missing.
func (n nodeStats) GetScenarioStats(scenarioName string) *scenarioStats {
	value := n.FindScenarioStats(scenarioName)
	if value == nil {
		panic(fmt.Sprintf("Scenario stats not found: %s.", scenarioName))
	}
	return value
}

// testInfo captures run labels used in reports and parity payloads.
type testInfo struct {
	TestSuite  string    `json:"TestSuite,omitempty"`
	TestName   string    `json:"TestName,omitempty"`
	SessionID  string    `json:"SessionId,omitempty"`
	SessionId  string    `json:"-"`
	ClusterID  string    `json:"ClusterId,omitempty"`
	ClusterId  string    `json:"-"`
	CreatedUTC time.Time `json:"CreatedUtc,omitempty"`
	Created    time.Time `json:"Created,omitempty"`
}

// thresholdResult captures the public threshold outcome shape used by parity normalization.
type thresholdResult struct {
	ScenarioName     string `json:"ScenarioName,omitempty"`
	StepName         string `json:"StepName,omitempty"`
	CheckExpression  string `json:"CheckExpression,omitempty"`
	ErrorCount       int    `json:"ErrorCount,omitempty"`
	IsFailed         bool   `json:"IsFailed,omitempty"`
	ExceptionMessage string `json:"ExceptionMessage,omitempty"`
}

// sinkErrorResult captures sink-level failures in the final run result.
type sinkErrorResult struct {
	SinkName string `json:"SinkName,omitempty"`
	Phase    string `json:"Phase,omitempty"`
	Message  string `json:"Message,omitempty"`
	Attempts int    `json:"Attempts,omitempty"`
}

// metricResult captures projected metric data included in final results.
type metricResult struct {
	Name          string  `json:"Name,omitempty"`
	ScenarioName  string  `json:"ScenarioName,omitempty"`
	Type          string  `json:"Type,omitempty"`
	Value         float64 `json:"Value,omitempty"`
	Unit          string  `json:"Unit,omitempty"`
	UnitOfMeasure string  `json:"UnitOfMeasure,omitempty"`
}

// correlationRow captures correlated or failed-correlated transaction rows.
type correlationRow struct {
	OccurredUTC             string `json:"OccurredUtc,omitempty"`
	ScenarioName            string `json:"ScenarioName,omitempty"`
	Source                  string `json:"Source,omitempty"`
	Destination             string `json:"Destination,omitempty"`
	RunMode                 string `json:"RunMode,omitempty"`
	Status                  string `json:"Status,omitempty"`
	TrackingID              string `json:"TrackingId,omitempty"`
	EventID                 string `json:"EventId,omitempty"`
	SourceTimestampUTC      string `json:"SourceTimestampUtc,omitempty"`
	DestinationTimestampUTC string `json:"DestinationTimestampUtc,omitempty"`
	LatencyMS               string `json:"LatencyMs,omitempty"`
	Message                 string `json:"Message,omitempty"`
	IsSuccess               bool   `json:"IsSuccess,omitempty"`
	IsFailure               bool   `json:"IsFailure,omitempty"`
	GatherByField           string `json:"GatherByField,omitempty"`
	GatherByValue           string `json:"GatherByValue,omitempty"`
}

func (r correlationRow) toMap() map[string]any {
	return map[string]any{
		"OccurredUtc":             r.OccurredUTC,
		"ScenarioName":            r.ScenarioName,
		"Source":                  r.Source,
		"Destination":             r.Destination,
		"RunMode":                 r.RunMode,
		"Status":                  r.Status,
		"TrackingId":              r.TrackingID,
		"EventId":                 r.EventID,
		"SourceTimestampUtc":      r.SourceTimestampUTC,
		"DestinationTimestampUtc": r.DestinationTimestampUTC,
		"LatencyMs":               r.LatencyMS,
		"Message":                 r.Message,
		"IsSuccess":               r.IsSuccess,
		"IsFailure":               r.IsFailure,
		"GatherByField":           r.GatherByField,
		"GatherByValue":           r.GatherByValue,
	}
}

func newCorrelationRowFromMap(row map[string]any) correlationRow {
	return correlationRow{
		OccurredUTC:             stringFromAny(row["OccurredUtc"]),
		ScenarioName:            stringFromAny(row["ScenarioName"]),
		Source:                  stringFromAny(row["Source"]),
		Destination:             stringFromAny(row["Destination"]),
		RunMode:                 stringFromAny(row["RunMode"]),
		Status:                  stringFromAny(row["Status"]),
		TrackingID:              stringFromAny(row["TrackingId"]),
		EventID:                 stringFromAny(row["EventId"]),
		SourceTimestampUTC:      stringFromAny(row["SourceTimestampUtc"]),
		DestinationTimestampUTC: stringFromAny(row["DestinationTimestampUtc"]),
		LatencyMS:               stringFromAny(row["LatencyMs"]),
		Message:                 stringFromAny(row["Message"]),
		IsSuccess:               boolFromAny(row["IsSuccess"]),
		IsFailure:               boolFromAny(row["IsFailure"]),
		GatherByField:           stringFromAny(row["GatherByField"]),
		GatherByValue:           stringFromAny(row["GatherByValue"]),
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.TrimSpace(strings.ToLower(typed)) {
		case "true", "1", "yes":
			return true
		default:
			return false
		}
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	default:
		return false
	}
}

// pluginData captures worker-plugin table output surfaced in reports.
type pluginData struct {
	PluginName string            `json:"PluginName"`
	Hints      []string          `json:"Hints,omitempty"`
	Tables     []pluginDataTable `json:"Tables,omitempty"`
}

// pluginDataTable captures a single worker-plugin table.
type pluginDataTable struct {
	TableName string           `json:"TableName"`
	Rows      []map[string]any `json:"Rows,omitempty"`
}

type distributionAccumulator struct {
	totalFloat float64
	values     []float64
}

func cloneStatusCodeCounts(source map[string]int) map[string]int {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneFloatMap(source map[string]float64) map[string]float64 {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]float64, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func normalizeScenarioStats(stats *scenarioStats) {
	if stats == nil {
		return
	}
	if stats.Duration == 0 && stats.DurationMS > 0 {
		stats.Duration = time.Duration(stats.DurationMS * float64(time.Millisecond))
	}
	if stats.CurrentOperationType == LoadStrikeOperationTypeNone && strings.TrimSpace(stats.CurrentOperation) != "" {
		stats.CurrentOperationType = parseOperationType(stats.CurrentOperation)
	}
	for index := range stats.stepStats {
		if stats.lookupFlavor == statsLookupFlavorRealtime {
			stats.stepStats[index].lookupFlavor = statsLookupFlavorRealtime
		}
	}
}

func normalizeRunResult(result *runResult) {
	if result == nil {
		return
	}
	if result.Duration == 0 && result.DurationMS > 0 {
		result.Duration = time.Duration(result.DurationMS * float64(time.Millisecond))
	}
	result.testInfo = normalizeTestInfo(result.testInfo)
	normalizeMetricStats(&result.metricStats)
}

func normalizeMetricStats(stats *metricStats) {
	if stats == nil {
		return
	}
	if stats.Duration == 0 && stats.DurationMS > 0 {
		stats.Duration = time.Duration(stats.DurationMS * float64(time.Millisecond))
	}
}

func normalizeNodeStats(stats *nodeStats) {
	if stats == nil {
		return
	}
	if stats.Duration == 0 && stats.DurationMS > 0 {
		stats.Duration = time.Duration(stats.DurationMS * float64(time.Millisecond))
	}
	if len(stats.scenarioStats) == 0 && len(stats.Scenarios) > 0 {
		stats.scenarioStats = append([]scenarioStats(nil), stats.Scenarios...)
	}
	if len(stats.Scenarios) == 0 && len(stats.scenarioStats) > 0 {
		stats.Scenarios = append([]scenarioStats(nil), stats.scenarioStats...)
	}
	stats.testInfo = normalizeTestInfo(stats.testInfo)
	for index := range stats.scenarioStats {
		normalizeScenarioStats(&stats.scenarioStats[index])
	}
	if len(stats.scenarioStats) > 0 {
		stats.Scenarios = append(stats.Scenarios[:0], stats.scenarioStats...)
	}
	normalizeMetricStats(&stats.Metrics)
}

func normalizedNodeScenarioStats(stats nodeStats) []scenarioStats {
	normalizeNodeStats(&stats)
	return stats.scenarioStats
}

func normalizeTestInfo(info testInfo) testInfo {
	if strings.TrimSpace(info.SessionID) == "" {
		info.SessionID = info.SessionId
	}
	if strings.TrimSpace(info.SessionId) == "" {
		info.SessionId = info.SessionID
	}
	if strings.TrimSpace(info.ClusterID) == "" {
		info.ClusterID = info.ClusterId
	}
	if strings.TrimSpace(info.ClusterId) == "" {
		info.ClusterId = info.ClusterID
	}
	if info.CreatedUTC.IsZero() {
		info.CreatedUTC = info.Created
	}
	if info.Created.IsZero() {
		info.Created = info.CreatedUTC
	}
	return info
}

func normalizeScenarioInfo(info LoadStrikeScenarioInfo) LoadStrikeScenarioInfo {
	if strings.TrimSpace(info.InstanceID) == "" {
		info.InstanceID = info.InstanceId
	}
	if strings.TrimSpace(info.InstanceId) == "" {
		info.InstanceId = info.InstanceID
	}
	return info
}

func validateScenarioStatsLookupName(scenarioName string) {
	if strings.TrimSpace(scenarioName) == "" {
		panic("scenario name must be provided")
	}
}

func validateStepStatsLookupName(stepName string) {
	if strings.TrimSpace(stepName) == "" {
		panic("step name must be provided")
	}
}

func parseOperationType(value string) LoadStrikeOperationType {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "init":
		return LoadStrikeOperationTypeInit
	case "warmup":
		return LoadStrikeOperationTypeWarmUp
	case "bombing":
		return LoadStrikeOperationTypeBombing
	case "stop":
		return LoadStrikeOperationTypeStop
	case "complete":
		return LoadStrikeOperationTypeComplete
	case "error":
		return LoadStrikeOperationTypeError
	default:
		return LoadStrikeOperationTypeNone
	}
}

func (d *distributionAccumulator) add(value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	d.totalFloat += value
	d.values = append(d.values, value)
}
