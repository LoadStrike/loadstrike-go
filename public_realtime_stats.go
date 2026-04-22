package loadstrike

import (
	"encoding/json"
	"fmt"
	"time"
)

// LoadStrikeStepStats mirrors the .NET realtime step-stats contract name.
type LoadStrikeStepStats struct {
	Fail      LoadStrikeMeasurementStats `json:"Fail,omitempty"`
	Ok        LoadStrikeMeasurementStats `json:"Ok,omitempty"`
	SortIndex int                        `json:"SortIndex,omitempty"`
	StepName  string                     `json:"StepName,omitempty"`
}

func newLoadStrikeStepStats(native stepStats) LoadStrikeStepStats {
	return LoadStrikeStepStats{
		Fail:      newLoadStrikeMeasurementStats(native.Fail),
		Ok:        newLoadStrikeMeasurementStats(native.Ok),
		SortIndex: native.SortIndex,
		StepName:  native.StepName,
	}
}

func (s LoadStrikeStepStats) toNative() stepStats {
	return stepStats{
		Fail:      s.Fail.toNative(),
		Ok:        s.Ok.toNative(),
		SortIndex: s.SortIndex,
		StepName:  s.StepName,
	}
}

// LoadStrikeLatencyStats mirrors the .NET latency-stats contract name.
type LoadStrikeLatencyStats struct {
	LatencyCount LoadStrikeLatencyCount `json:"LatencyCount,omitempty"`
	MaxMs        float64                `json:"MaxMs,omitempty"`
	MeanMs       float64                `json:"MeanMs,omitempty"`
	MinMs        float64                `json:"MinMs,omitempty"`
	Percent50    float64                `json:"Percent50,omitempty"`
	Percent75    float64                `json:"Percent75,omitempty"`
	Percent95    float64                `json:"Percent95,omitempty"`
	Percent99    float64                `json:"Percent99,omitempty"`
	StdDev       float64                `json:"StdDev,omitempty"`
}

func newLoadStrikeLatencyStats(native latencyStats) LoadStrikeLatencyStats {
	return LoadStrikeLatencyStats{
		LatencyCount: newLoadStrikeLatencyCount(native.LatencyCount),
		MaxMs:        native.MaxMs,
		MeanMs:       native.MeanMs,
		MinMs:        native.MinMs,
		Percent50:    native.Percent50,
		Percent75:    native.Percent75,
		Percent95:    native.Percent95,
		Percent99:    native.Percent99,
		StdDev:       native.StdDev,
	}
}

func (s LoadStrikeLatencyStats) toNative() latencyStats {
	return latencyStats{
		LatencyCount: s.LatencyCount.toNative(),
		MaxMs:        s.MaxMs,
		MeanMs:       s.MeanMs,
		MinMs:        s.MinMs,
		Percent50:    s.Percent50,
		Percent75:    s.Percent75,
		Percent95:    s.Percent95,
		Percent99:    s.Percent99,
		StdDev:       s.StdDev,
	}
}

// LoadStrikeMeasurementStats mirrors the .NET measurement-stats contract name.
type LoadStrikeMeasurementStats struct {
	Request      LoadStrikeRequestStats      `json:"Request,omitempty"`
	DataTransfer LoadStrikeDataTransferStats `json:"DataTransfer,omitempty"`
	Latency      LoadStrikeLatencyStats      `json:"Latency,omitempty"`
	StatusCodes  []LoadStrikeStatusCodeStats `json:"StatusCodes,omitempty"`
}

func newLoadStrikeMeasurementStats(native measurementStats) LoadStrikeMeasurementStats {
	statusCodes := make([]LoadStrikeStatusCodeStats, 0, len(native.StatusCodes))
	for _, statusCode := range native.StatusCodes {
		statusCodes = append(statusCodes, newLoadStrikeStatusCodeStats(statusCode))
	}
	return LoadStrikeMeasurementStats{
		Request:      newLoadStrikeRequestStats(native.Request),
		DataTransfer: newLoadStrikeDataTransferStats(native.DataTransfer),
		Latency:      newLoadStrikeLatencyStats(native.Latency),
		StatusCodes:  statusCodes,
	}
}

func (s LoadStrikeMeasurementStats) toNative() measurementStats {
	statusCodes := make([]statusCodeStats, 0, len(s.StatusCodes))
	for _, statusCode := range s.StatusCodes {
		statusCodes = append(statusCodes, statusCode.toNative())
	}
	return measurementStats{
		Request:      s.Request.toNative(),
		DataTransfer: s.DataTransfer.toNative(),
		Latency:      s.Latency.toNative(),
		StatusCodes:  statusCodes,
	}
}

// LoadStrikeMetricStats mirrors the .NET metric-stats contract name.
type LoadStrikeMetricStats struct {
	Counters []LoadStrikeCounterStats `json:"Counters,omitempty"`
	Duration time.Duration            `json:"Duration,omitempty"`
	Gauges   []LoadStrikeGaugeStats   `json:"Gauges,omitempty"`
}

func newLoadStrikeMetricStats(native metricStats) LoadStrikeMetricStats {
	normalizeMetricStats(&native)
	counters := make([]LoadStrikeCounterStats, 0, len(native.Counters))
	for _, counter := range native.Counters {
		counters = append(counters, newLoadStrikeCounterStats(counter))
	}
	gauges := make([]LoadStrikeGaugeStats, 0, len(native.Gauges))
	for _, gauge := range native.Gauges {
		gauges = append(gauges, newLoadStrikeGaugeStats(gauge))
	}
	return LoadStrikeMetricStats{
		Counters: counters,
		Duration: native.Duration,
		Gauges:   gauges,
	}
}

func (s LoadStrikeMetricStats) toNative() metricStats {
	counters := make([]counterStats, 0, len(s.Counters))
	for _, counter := range s.Counters {
		counters = append(counters, counter.toNative())
	}
	gauges := make([]gaugeStats, 0, len(s.Gauges))
	for _, gauge := range s.Gauges {
		gauges = append(gauges, gauge.toNative())
	}
	return metricStats{
		Duration:   s.Duration,
		DurationMS: float64(s.Duration) / float64(time.Millisecond),
		Counters:   counters,
		Gauges:     gauges,
	}
}

// LoadStrikeScenarioStats mirrors the .NET realtime scenario-stats contract name.
type LoadStrikeScenarioStats struct {
	AllBytes            int64                         `json:"AllBytes,omitempty"`
	AllFailCount        int                           `json:"AllFailCount,omitempty"`
	AllOKCount          int                           `json:"AllOkCount,omitempty"`
	AllRequestCount     int                           `json:"AllRequestCount,omitempty"`
	CurrentOperation    LoadStrikeOperationType       `json:"CurrentOperation,omitempty"`
	Duration            time.Duration                 `json:"Duration,omitempty"`
	Fail                LoadStrikeMeasurementStats    `json:"Fail,omitempty"`
	LoadSimulationStats LoadStrikeLoadSimulationStats `json:"LoadSimulationStats,omitempty"`
	Ok                  LoadStrikeMeasurementStats    `json:"Ok,omitempty"`
	ScenarioName        string                        `json:"ScenarioName,omitempty"`
	SortIndex           int                           `json:"SortIndex,omitempty"`
	stepStats           []LoadStrikeStepStats         `json:"stepStats,omitempty"`
}

func newLoadStrikeScenarioStats(native scenarioStats) LoadStrikeScenarioStats {
	normalizeScenarioStats(&native)
	steps := make([]LoadStrikeStepStats, 0, len(native.stepStats))
	for _, step := range native.stepStats {
		steps = append(steps, newLoadStrikeStepStats(step))
	}
	return LoadStrikeScenarioStats{
		AllBytes:            native.AllBytes,
		AllFailCount:        native.AllFailCount,
		AllOKCount:          native.AllOKCount,
		AllRequestCount:     native.AllRequestCount,
		CurrentOperation:    operationTypeFromScenarioStats(native),
		Duration:            native.Duration,
		Fail:                newLoadStrikeMeasurementStats(native.Fail),
		LoadSimulationStats: newLoadStrikeLoadSimulationStats(native.LoadSimulationStats),
		Ok:                  newLoadStrikeMeasurementStats(native.Ok),
		ScenarioName:        native.ScenarioName,
		SortIndex:           native.SortIndex,
		stepStats:           steps,
	}
}

func (s LoadStrikeScenarioStats) FindStepStats(stepName string) *LoadStrikeStepStats {
	for index := range s.stepStats {
		if s.stepStats[index].StepName == stepName {
			return &s.stepStats[index]
		}
	}
	return nil
}

func (s LoadStrikeScenarioStats) GetStepStats(stepName string) *LoadStrikeStepStats {
	value := s.FindStepStats(stepName)
	if value == nil {
		panic(fmt.Sprintf("step stats not found: %s.", stepName))
	}
	return value
}

func (s LoadStrikeScenarioStats) toNative() scenarioStats {
	steps := make([]stepStats, 0, len(s.stepStats))
	for _, step := range s.stepStats {
		steps = append(steps, step.toNative())
	}
	native := scenarioStats{
		AllBytes:             s.AllBytes,
		AllFailCount:         s.AllFailCount,
		AllOKCount:           s.AllOKCount,
		AllRequestCount:      s.AllRequestCount,
		CurrentOperation:     operationTypeName(s.CurrentOperation),
		CurrentOperationType: s.CurrentOperation,
		Duration:             s.Duration,
		DurationMS:           float64(s.Duration) / float64(time.Millisecond),
		Fail:                 s.Fail.toNative(),
		LoadSimulationStats:  s.LoadSimulationStats.toNative(),
		Ok:                   s.Ok.toNative(),
		ScenarioName:         s.ScenarioName,
		SortIndex:            s.SortIndex,
		stepStats:            steps,
	}
	return native
}

type loadStrikeNodeStats struct {
	AllBytes        int64                       `json:"AllBytes,omitempty"`
	AllFailCount    int                         `json:"AllFailCount,omitempty"`
	AllOKCount      int                         `json:"AllOkCount,omitempty"`
	AllRequestCount int                         `json:"AllRequestCount,omitempty"`
	Duration        time.Duration               `json:"Duration,omitempty"`
	Metrics         LoadStrikeMetricStats       `json:"Metrics,omitempty"`
	nodeInfo        LoadStrikeNodeInfo          `json:"nodeInfo,omitempty"`
	PluginsData     []LoadStrikePluginData      `json:"PluginsData,omitempty"`
	scenarioStats   []LoadStrikeScenarioStats   `json:"scenarioStats,omitempty"`
	testInfo        LoadStrikeTestInfo          `json:"testInfo"`
	Thresholds      []LoadStrikeThresholdResult `json:"Thresholds,omitempty"`
}

func newLoadStrikeNodeStats(native nodeStats) loadStrikeNodeStats {
	normalizeNodeStats(&native)
	scenarios := make([]LoadStrikeScenarioStats, 0, len(native.scenarioStats))
	for _, scenario := range native.scenarioStats {
		scenarios = append(scenarios, newLoadStrikeScenarioStats(scenario))
	}
	return loadStrikeNodeStats{
		AllBytes:        native.AllBytes,
		AllFailCount:    native.AllFailCount,
		AllOKCount:      native.AllOKCount,
		AllRequestCount: native.AllRequestCount,
		Duration:        native.Duration,
		Metrics:         newLoadStrikeMetricStats(native.Metrics),
		nodeInfo:        newLoadStrikeNodeInfo(native.nodeInfo),
		PluginsData:     toLoadStrikePluginDataSlice(native.PluginsData),
		scenarioStats:   scenarios,
		testInfo:        newLoadStrikeTestInfo(native.testInfo),
		Thresholds:      toLoadStrikeThresholdResultSlice(native.Thresholds),
	}
}

func (n loadStrikeNodeStats) FindScenarioStats(scenarioName string) *LoadStrikeScenarioStats {
	for index := range n.scenarioStats {
		if n.scenarioStats[index].ScenarioName == scenarioName {
			return &n.scenarioStats[index]
		}
	}
	return nil
}

func (n loadStrikeNodeStats) GetScenarioStats(scenarioName string) *LoadStrikeScenarioStats {
	value := n.FindScenarioStats(scenarioName)
	if value == nil {
		panic(fmt.Sprintf("scenario stats not found: %s.", scenarioName))
	}
	return value
}

func (n loadStrikeNodeStats) toNative() nodeStats {
	scenarios := make([]scenarioStats, 0, len(n.scenarioStats))
	for _, scenario := range n.scenarioStats {
		scenarios = append(scenarios, scenario.toNative())
	}
	plugins := make([]pluginData, 0, len(n.PluginsData))
	for _, plugin := range n.PluginsData {
		plugins = append(plugins, plugin.toNative())
	}
	thresholds := make([]thresholdResult, 0, len(n.Thresholds))
	for _, threshold := range n.Thresholds {
		thresholds = append(thresholds, threshold.toNative())
	}
	native := nodeStats{
		AllBytes:        n.AllBytes,
		AllRequestCount: n.AllRequestCount,
		AllOKCount:      n.AllOKCount,
		AllFailCount:    n.AllFailCount,
		Duration:        n.Duration,
		DurationMS:      float64(n.Duration) / float64(time.Millisecond),
		Metrics:         n.Metrics.toNative(),
		nodeInfo:        n.nodeInfo.toNative(),
		testInfo:        n.testInfo.toNative(),
		PluginsData:     plugins,
		scenarioStats:   scenarios,
		Thresholds:      thresholds,
	}
	normalizeNodeStats(&native)
	return native
}

func (n loadStrikeNodeStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.toNative())
}

func (n *loadStrikeNodeStats) UnmarshalJSON(data []byte) error {
	var native nodeStats
	if err := json.Unmarshal(data, &native); err != nil {
		return err
	}
	*n = newLoadStrikeNodeStats(native)
	return nil
}

func toLoadStrikeScenarioStats(items []scenarioStats) []LoadStrikeScenarioStats {
	if len(items) == 0 {
		return nil
	}
	result := make([]LoadStrikeScenarioStats, 0, len(items))
	for _, item := range items {
		result = append(result, newLoadStrikeScenarioStats(item))
	}
	return result
}

func operationTypeFromScenarioStats(stats scenarioStats) LoadStrikeOperationType {
	if stats.CurrentOperationType != LoadStrikeOperationTypeNone {
		return stats.CurrentOperationType
	}
	return parseOperationType(stats.CurrentOperation)
}

func operationTypeName(value LoadStrikeOperationType) string {
	switch value {
	case LoadStrikeOperationTypeInit:
		return "Init"
	case LoadStrikeOperationTypeWarmUp:
		return "WarmUp"
	case LoadStrikeOperationTypeBombing:
		return "Bombing"
	case LoadStrikeOperationTypeStop:
		return "Stop"
	case LoadStrikeOperationTypeComplete:
		return "Complete"
	case LoadStrikeOperationTypeError:
		return "Error"
	default:
		return "None"
	}
}
