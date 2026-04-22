package loadstrike

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
)

type sinkSessionMetadata struct {
	SessionID   string
	TestSuite   string
	TestName    string
	ClusterID   string
	NodeType    string
	MachineName string
}

type reportingSinkEvent struct {
	EventType    string
	OccurredUTC  time.Time
	SessionID    string
	TestSuite    string
	TestName     string
	ClusterID    string
	NodeType     string
	MachineName  string
	ScenarioName string
	StepName     string
	Tags         map[string]string
	Fields       map[string]any
}

type reportingSinkMetricPoint struct {
	MetricName    string
	MetricKind    string
	OccurredUTC   time.Time
	Value         float64
	UnitOfMeasure string
	Tags          map[string]string
}

func newSinkSessionMetadata(context contextState) sinkSessionMetadata {
	machineName, _ := os.Hostname()
	if strings.TrimSpace(machineName) == "" {
		machineName = "local-machine"
	}
	return sinkSessionMetadata{
		SessionID:   firstNonBlank(context.SessionID, generateSessionID()),
		TestSuite:   firstNonBlank(context.TestSuite, "default"),
		TestName:    firstNonBlank(context.TestName, "loadstrike"),
		ClusterID:   firstNonBlank(context.ClusterID, "default"),
		NodeType:    nodeTypeName(context.NodeType),
		MachineName: machineName,
	}
}

func sinkEventsFromRealtimeStats(session sinkSessionMetadata, scenarios []scenarioStats) []reportingSinkEvent {
	now := time.Now().UTC()
	events := make([]reportingSinkEvent, 0, len(scenarios))
	for _, scenario := range scenarios {
		events = append(events, reportingSinkEvent{
			EventType:    "scenario.realtime",
			OccurredUTC:  now,
			SessionID:    session.SessionID,
			TestSuite:    session.TestSuite,
			TestName:     session.TestName,
			ClusterID:    session.ClusterID,
			NodeType:     session.NodeType,
			MachineName:  session.MachineName,
			ScenarioName: scenario.ScenarioName,
			Tags: map[string]string{
				"phase":  "realtime",
				"entity": "scenario",
			},
			Fields: map[string]any{
				"all_request_count": scenario.AllRequestCount,
				"all_ok_count":      scenario.AllOKCount,
				"all_fail_count":    scenario.AllFailCount,
				"duration_ms":       scenario.DurationMS,
				"sort_index":        scenario.SortIndex,
			},
		})
		for _, step := range scenario.stepStats {
			events = append(events, reportingSinkEvent{
				EventType:    "step.realtime",
				OccurredUTC:  now,
				SessionID:    session.SessionID,
				TestSuite:    session.TestSuite,
				TestName:     session.TestName,
				ClusterID:    session.ClusterID,
				NodeType:     session.NodeType,
				MachineName:  session.MachineName,
				ScenarioName: scenario.ScenarioName,
				StepName:     step.StepName,
				Tags: map[string]string{
					"phase":  "realtime",
					"entity": "step",
				},
				Fields: map[string]any{
					"all_request_count": step.AllRequestCount,
					"all_ok_count":      step.AllOKCount,
					"all_fail_count":    step.AllFailCount,
					"sort_index":        step.SortIndex,
				},
			})
		}
	}
	return events
}

func sinkEventsFromRealtimeMetrics(session sinkSessionMetadata, metrics metricStats) []reportingSinkEvent {
	now := time.Now().UTC()
	events := make([]reportingSinkEvent, 0, len(metrics.Counters)+len(metrics.Gauges))
	for _, metric := range metrics.Counters {
		events = append(events, reportingSinkEvent{
			EventType:   "metric.counter.realtime",
			OccurredUTC: now,
			SessionID:   session.SessionID,
			TestSuite:   session.TestSuite,
			TestName:    session.TestName,
			ClusterID:   session.ClusterID,
			NodeType:    session.NodeType,
			MachineName: session.MachineName,
			Tags: map[string]string{
				"phase":         "realtime",
				"entity":        "metric",
				"metric_name":   metric.MetricName,
				"scenario_name": metric.ScenarioName,
			},
			Fields: map[string]any{
				"value":           metric.Value,
				"unit_of_measure": metric.UnitOfMeasure,
				"scenario_name":   metric.ScenarioName,
			},
		})
	}
	for _, metric := range metrics.Gauges {
		events = append(events, reportingSinkEvent{
			EventType:   "metric.gauge.realtime",
			OccurredUTC: now,
			SessionID:   session.SessionID,
			TestSuite:   session.TestSuite,
			TestName:    session.TestName,
			ClusterID:   session.ClusterID,
			NodeType:    session.NodeType,
			MachineName: session.MachineName,
			Tags: map[string]string{
				"phase":         "realtime",
				"entity":        "metric",
				"metric_name":   metric.MetricName,
				"scenario_name": metric.ScenarioName,
			},
			Fields: map[string]any{
				"value":           metric.Value,
				"unit_of_measure": metric.UnitOfMeasure,
				"scenario_name":   metric.ScenarioName,
			},
		})
	}
	return events
}

func sinkEventsFromRunResult(session sinkSessionMetadata, result runResult) []reportingSinkEvent {
	now := time.Now().UTC()
	events := []reportingSinkEvent{
		{
			EventType:   "test.final",
			OccurredUTC: now,
			SessionID:   session.SessionID,
			TestSuite:   session.TestSuite,
			TestName:    session.TestName,
			ClusterID:   session.ClusterID,
			NodeType:    session.NodeType,
			MachineName: session.MachineName,
			Tags: map[string]string{
				"phase":  "final",
				"entity": "test",
			},
			Fields: map[string]any{
				"all_request_count": result.AllRequestCount,
				"all_ok_count":      result.AllOKCount,
				"all_fail_count":    result.AllFailCount,
				"duration_ms":       result.CompletedUTC.Sub(result.StartedUTC).Seconds() * 1000,
				"scenario_count":    len(result.scenarioStats),
				"threshold_count":   len(result.Thresholds),
			},
		},
	}

	for _, scenario := range result.scenarioStats {
		events = append(events, reportingSinkEvent{
			EventType:    "scenario.final",
			OccurredUTC:  now,
			SessionID:    session.SessionID,
			TestSuite:    session.TestSuite,
			TestName:     session.TestName,
			ClusterID:    session.ClusterID,
			NodeType:     session.NodeType,
			MachineName:  session.MachineName,
			ScenarioName: scenario.ScenarioName,
			Tags: map[string]string{
				"phase":  "final",
				"entity": "scenario",
			},
			Fields: map[string]any{
				"all_request_count": scenario.AllRequestCount,
				"all_ok_count":      scenario.AllOKCount,
				"all_fail_count":    scenario.AllFailCount,
				"duration_ms":       scenario.DurationMS,
				"sort_index":        scenario.SortIndex,
			},
		})
		for _, step := range scenario.stepStats {
			events = append(events, reportingSinkEvent{
				EventType:    "step.final",
				OccurredUTC:  now,
				SessionID:    session.SessionID,
				TestSuite:    session.TestSuite,
				TestName:     session.TestName,
				ClusterID:    session.ClusterID,
				NodeType:     session.NodeType,
				MachineName:  session.MachineName,
				ScenarioName: scenario.ScenarioName,
				StepName:     step.StepName,
				Tags: map[string]string{
					"phase":  "final",
					"entity": "step",
				},
				Fields: map[string]any{
					"all_request_count": step.AllRequestCount,
					"all_ok_count":      step.AllOKCount,
					"all_fail_count":    step.AllFailCount,
					"sort_index":        step.SortIndex,
				},
			})
		}
	}

	for _, threshold := range result.Thresholds {
		events = append(events, reportingSinkEvent{
			EventType:    "threshold.final",
			OccurredUTC:  now,
			SessionID:    session.SessionID,
			TestSuite:    session.TestSuite,
			TestName:     session.TestName,
			ClusterID:    session.ClusterID,
			NodeType:     session.NodeType,
			MachineName:  session.MachineName,
			ScenarioName: threshold.ScenarioName,
			StepName:     threshold.StepName,
			Tags: map[string]string{
				"phase":  "final",
				"entity": "threshold",
			},
			Fields: map[string]any{
				"check_expression": threshold.CheckExpression,
				"error_count":      threshold.ErrorCount,
				"is_failed":        threshold.IsFailed,
				"exception_msg":    threshold.ExceptionMessage,
			},
		})
	}

	for _, plugin := range result.PluginsData {
		events = append(events, reportingSinkEvent{
			EventType:   "plugin.final",
			OccurredUTC: now,
			SessionID:   session.SessionID,
			TestSuite:   session.TestSuite,
			TestName:    session.TestName,
			ClusterID:   session.ClusterID,
			NodeType:    session.NodeType,
			MachineName: session.MachineName,
			Tags: map[string]string{
				"phase":       "final",
				"entity":      "plugin",
				"plugin_name": plugin.PluginName,
			},
			Fields: map[string]any{
				"hint_count":  len(plugin.Hints),
				"table_count": len(plugin.Tables),
			},
		})
	}

	for _, reportFile := range result.ReportFiles {
		events = append(events, reportingSinkEvent{
			EventType:   "run.report.final",
			OccurredUTC: now,
			SessionID:   session.SessionID,
			TestSuite:   session.TestSuite,
			TestName:    session.TestName,
			ClusterID:   session.ClusterID,
			NodeType:    session.NodeType,
			MachineName: session.MachineName,
			Tags: map[string]string{
				"phase":  "final",
				"entity": "run-report",
			},
			Fields: map[string]any{
				"path": reportFile,
			},
		})
	}

	for _, sinkError := range result.SinkErrors {
		events = append(events, reportingSinkEvent{
			EventType:   "run.sink-error.final",
			OccurredUTC: now,
			SessionID:   session.SessionID,
			TestSuite:   session.TestSuite,
			TestName:    session.TestName,
			ClusterID:   session.ClusterID,
			NodeType:    session.NodeType,
			MachineName: session.MachineName,
			Tags: map[string]string{
				"phase":      "final",
				"entity":     "sink-error",
				"sink_name":  sinkError.SinkName,
				"sink_phase": sinkError.Phase,
			},
			Fields: map[string]any{
				"sink_name": sinkError.SinkName,
				"phase":     sinkError.Phase,
				"message":   sinkError.Message,
				"attempts":  sinkError.Attempts,
			},
		})
	}

	for _, metricEvent := range sinkEventsFromRealtimeMetrics(session, result.metricStats) {
		metricEvent.EventType = strings.Replace(metricEvent.EventType, ".realtime", ".final", 1)
		metricEvent.OccurredUTC = now
		events = append(events, metricEvent)
	}

	return events
}

func reportingSinkMetricProjectionFromEvents(sinkName string, events []reportingSinkEvent, staticTags map[string]string) []reportingSinkMetricPoint {
	points := []reportingSinkMetricPoint{}
	for _, event := range events {
		metricTags := mergeMetricTags(sinkName, event, staticTags)
		isCounterEvent := strings.HasPrefix(strings.ToLower(event.EventType), "metric.counter")
		isGaugeEvent := strings.HasPrefix(strings.ToLower(event.EventType), "metric.gauge")
		metricNameTag := metricTags["metric_name"]
		unit := asString(event.Fields["unit_of_measure"])

		if (isCounterEvent || isGaugeEvent) && event.Fields["value"] != nil {
			points = append(points, reportingSinkMetricPoint{
				MetricName:    buildCustomMetricName(metricNameTag),
				MetricKind:    metricKind(isCounterEvent),
				OccurredUTC:   event.OccurredUTC,
				Value:         asDouble(event.Fields["value"]),
				UnitOfMeasure: unit,
				Tags:          metricTags,
			})
		}

		for fieldName, fieldValue := range event.Fields {
			if fieldName == "unit_of_measure" || ((isCounterEvent || isGaugeEvent) && strings.EqualFold(fieldName, "value")) {
				continue
			}
			switch fieldValue.(type) {
			case int, int32, int64, float32, float64, json.Number:
				points = append(points, reportingSinkMetricPoint{
					MetricName:    buildEventMetricName(event.EventType, fieldName),
					MetricKind:    "gauge",
					OccurredUTC:   event.OccurredUTC,
					Value:         asDouble(fieldValue),
					UnitOfMeasure: inferUnitOfMeasure(fieldName),
					Tags:          metricTags,
				})
			}
		}
	}
	return points
}

func mergeMetricTags(sinkName string, event reportingSinkEvent, staticTags map[string]string) map[string]string {
	merged := map[string]string{
		"source":       "loadstrike",
		"sink":         sinkName,
		"event_type":   event.EventType,
		"session_id":   firstNonBlank(event.SessionID, "default"),
		"test_suite":   firstNonBlank(event.TestSuite, "default"),
		"test_name":    firstNonBlank(event.TestName, "loadstrike"),
		"cluster_id":   firstNonBlank(event.ClusterID, "default"),
		"node_type":    firstNonBlank(event.NodeType, "SingleNode"),
		"machine_name": firstNonBlank(event.MachineName, "local-machine"),
	}
	if strings.TrimSpace(event.ScenarioName) != "" {
		merged["scenario_name"] = event.ScenarioName
	}
	if strings.TrimSpace(event.StepName) != "" {
		merged["step_name"] = event.StepName
	}
	for key, value := range staticTags {
		if strings.TrimSpace(value) != "" {
			merged[key] = value
		}
	}
	for key, value := range event.Tags {
		if strings.TrimSpace(value) != "" {
			merged[key] = value
		}
	}
	return merged
}

func buildCustomMetricName(metricName string) string {
	normalized := normalizeMetricPath(metricName)
	if normalized == "" {
		return "loadstrike.metric.value"
	}
	return "loadstrike.metric." + normalized
}

func buildEventMetricName(eventType string, fieldName string) string {
	return "loadstrike." + normalizeMetricPath(eventType) + "." + normalizeMetricPath(fieldName)
}

func normalizeMetricPath(value string) string {
	if strings.TrimSpace(value) == "" {
		return "value"
	}
	segments := strings.Split(value, ".")
	normalized := []string{}
	for _, segment := range segments {
		builder := strings.Builder{}
		lastUnderscore := false
		for _, ch := range segment {
			switch {
			case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9'):
				builder.WriteRune([]rune(strings.ToLower(string(ch)))[0])
				lastUnderscore = false
			case !lastUnderscore:
				builder.WriteRune('_')
				lastUnderscore = true
			}
		}
		part := strings.Trim(builder.String(), "_")
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	if len(normalized) == 0 {
		return "value"
	}
	return strings.Join(normalized, ".")
}

func inferUnitOfMeasure(fieldName string) string {
	lower := strings.ToLower(strings.TrimSpace(fieldName))
	switch {
	case strings.HasSuffix(lower, "_ms"):
		return "ms"
	case strings.HasSuffix(lower, "_bytes"), strings.HasSuffix(lower, "_byte"):
		return "bytes"
	default:
		return ""
	}
}

func metricKind(counter bool) string {
	if counter {
		return "count"
	}
	return "gauge"
}

func reportingSinkEventJSON(event reportingSinkEvent) string {
	body, err := json.Marshal(map[string]any{
		"EventType":    event.EventType,
		"occurredUtc":  event.OccurredUTC,
		"SessionId":    event.SessionID,
		"TestSuite":    event.TestSuite,
		"TestName":     event.TestName,
		"ClusterId":    event.ClusterID,
		"NodeType":     event.NodeType,
		"MachineName":  event.MachineName,
		"ScenarioName": event.ScenarioName,
		"StepName":     event.StepName,
		"Tags":         event.Tags,
		"Fields":       event.Fields,
	})
	if err != nil {
		return "{}"
	}
	return string(body)
}

func sortedTags(tags map[string]string) [][2]string {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([][2]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, [2]string{key, tags[key]})
	}
	return result
}
