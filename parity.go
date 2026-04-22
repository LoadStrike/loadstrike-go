package loadstrike

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// RunParityFiles executes a parity request file and writes the normalized result JSON.
func RunParityFiles(requestPath string, outputPath string) error {
	requestBytes, err := os.ReadFile(requestPath)
	if err != nil {
		return err
	}

	var request runRequest
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		return err
	}

	response, err := newLocalClient().Run(request)
	if err != nil {
		return err
	}

	normalized := NormalizeParityResponse(response)
	outputBytes, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(outputPath, append(outputBytes, '\n'), 0o644)
}

// NormalizeParityResponse returns the shared parity comparison shape for a raw
// parity response or a public final run result.
func NormalizeParityResponse(response any) map[string]any {
	var stats nodeStats
	switch typed := response.(type) {
	case runResponse:
		stats = typed.Stats
	case LoadStrikeRunResult:
		native := typed.toNative()
		stats = nodeStats{
			AllBytes:        native.AllBytes,
			AllRequestCount: native.AllRequestCount,
			AllOKCount:      native.AllOKCount,
			AllFailCount:    native.AllFailCount,
			DurationMS:      native.DurationMS,
			Duration:        native.Duration,
			Metrics:         native.metricStats,
			nodeInfo:        native.nodeInfo,
			testInfo:        native.testInfo,
			PluginsData:     append([]pluginData(nil), native.PluginsData...),
			scenarioStats:   append([]scenarioStats(nil), native.scenarioStats...),
			Scenarios:       append([]scenarioStats(nil), native.scenarioStats...),
			Thresholds:      append([]thresholdResult(nil), native.Thresholds...),
		}
	case *LoadStrikeRunResult:
		if typed != nil {
			native := typed.toNative()
			stats = nodeStats{
				AllBytes:        native.AllBytes,
				AllRequestCount: native.AllRequestCount,
				AllOKCount:      native.AllOKCount,
				AllFailCount:    native.AllFailCount,
				DurationMS:      native.DurationMS,
				Duration:        native.Duration,
				Metrics:         native.metricStats,
				nodeInfo:        native.nodeInfo,
				testInfo:        native.testInfo,
				PluginsData:     append([]pluginData(nil), native.PluginsData...),
				scenarioStats:   append([]scenarioStats(nil), native.scenarioStats...),
				Scenarios:       append([]scenarioStats(nil), native.scenarioStats...),
				Thresholds:      append([]thresholdResult(nil), native.Thresholds...),
			}
		}
	default:
		return map[string]any{}
	}

	scenarioCounts := make(map[string]any, len(stats.Scenarios))
	for _, scenario := range stats.Scenarios {
		scenarioCounts[scenario.ScenarioName] = map[string]any{
			"allRequestCount": scenario.AllRequestCount,
			"allOkCount":      scenario.AllOKCount,
			"allFailCount":    scenario.AllFailCount,
		}
	}

	failedThresholds := 0
	for _, threshold := range stats.Thresholds {
		if threshold.IsFailed {
			failedThresholds++
		}
	}

	return map[string]any{
		"allRequestCount":  stats.AllRequestCount,
		"allOkCount":       stats.AllOKCount,
		"allFailCount":     stats.AllFailCount,
		"scenarioCounts":   scenarioCounts,
		"failedThresholds": failedThresholds,
	}
}
