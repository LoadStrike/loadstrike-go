package loadstrike

import "strings"

// ThresholdSpec defines a public threshold contract.
type ThresholdSpec struct {
	Scope                  string  `json:"Scope"`
	StepName               string  `json:"StepName,omitempty"`
	Field                  string  `json:"Field"`
	Operator               string  `json:"Operator"`
	Value                  float64 `json:"Value,omitempty"`
	AbortWhenErrorCount    int     `json:"AbortWhenErrorCount,omitempty"`
	StartCheckAfterSeconds float64 `json:"StartCheckAfterSeconds,omitempty"`
	scenarioPredicate      func(scenarioStats) bool
	stepPredicate          func(stepStats) bool
	metricPredicate        func(metricStats) bool
}

// ScenarioThreshold creates a scenario-scope threshold.
func ScenarioThreshold(args ...any) ThresholdSpec {
	if len(args) > 0 {
		if predicate, ok := args[0].(func(scenarioStats) bool); ok {
			spec := ThresholdSpec{Scope: "scenario", Field: "<predicate>", Operator: "predicate", scenarioPredicate: predicate}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
		if predicate, ok := args[0].(func(LoadStrikeScenarioStats) bool); ok {
			spec := ThresholdSpec{
				Scope:    "scenario",
				Field:    "<predicate>",
				Operator: "predicate",
				scenarioPredicate: func(stats scenarioStats) bool {
					return predicate(newLoadStrikeScenarioStats(stats))
				},
			}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
		if predicate, ok := args[0].(func(LoadStrikeDetailedScenarioStats) bool); ok {
			spec := ThresholdSpec{
				Scope:    "scenario",
				Field:    "<predicate>",
				Operator: "predicate",
				scenarioPredicate: func(stats scenarioStats) bool {
					return predicate(newLoadStrikeDetailedScenarioStats(stats))
				},
			}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
	}
	spec := ThresholdSpec{Scope: "scenario"}
	if len(args) > 0 {
		spec.Field = asString(args[0])
	}
	if len(args) > 1 {
		spec.Operator = asString(args[1])
	}
	if len(args) > 2 {
		spec.Value = asDouble(args[2])
	}
	return spec
}

// StepThreshold creates a step-scope threshold.
func StepThreshold(stepName string, args ...any) ThresholdSpec {
	if strings.TrimSpace(stepName) == "" {
		panic("step name must be provided.")
	}
	spec := ThresholdSpec{Scope: "step", StepName: stepName}
	if len(args) > 0 {
		if predicate, ok := args[0].(func(stepStats) bool); ok {
			spec.Field = "<predicate>"
			spec.Operator = "predicate"
			spec.stepPredicate = predicate
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
		if predicate, ok := args[0].(func(LoadStrikeStepStats) bool); ok {
			spec.Field = "<predicate>"
			spec.Operator = "predicate"
			spec.stepPredicate = func(stats stepStats) bool {
				return predicate(newLoadStrikeStepStats(stats))
			}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
		if predicate, ok := args[0].(func(LoadStrikeDetailedStepStats) bool); ok {
			spec.Field = "<predicate>"
			spec.Operator = "predicate"
			spec.stepPredicate = func(stats stepStats) bool {
				return predicate(newLoadStrikeDetailedStepStats(stats))
			}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
	}
	if len(args) > 0 {
		spec.Field = asString(args[0])
	}
	if len(args) > 1 {
		spec.Operator = asString(args[1])
	}
	if len(args) > 2 {
		spec.Value = asDouble(args[2])
	}
	return spec
}

// MetricThreshold creates a metric-scope threshold.
func MetricThreshold(args ...any) ThresholdSpec {
	if len(args) > 0 {
		if predicate, ok := args[0].(func(metricStats) bool); ok {
			spec := ThresholdSpec{Scope: "metric", Field: "<predicate>", Operator: "predicate", metricPredicate: predicate}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
		if predicate, ok := args[0].(func(LoadStrikeMetricStats) bool); ok {
			spec := ThresholdSpec{
				Scope:    "metric",
				Field:    "<predicate>",
				Operator: "predicate",
				metricPredicate: func(stats metricStats) bool {
					return predicate(newLoadStrikeMetricStats(stats))
				},
			}
			applyThresholdOptions(&spec, args[1:]...)
			return spec
		}
	}
	spec := ThresholdSpec{Scope: "metric"}
	if len(args) > 0 {
		spec.Field = asString(args[0])
	}
	if len(args) > 1 {
		spec.Operator = asString(args[1])
	}
	if len(args) > 2 {
		spec.Value = asDouble(args[2])
	}
	return spec
}

func applyThresholdOptions(spec *ThresholdSpec, args ...any) {
	if spec == nil {
		return
	}
	if len(args) > 0 {
		switch typed := args[0].(type) {
		case int:
			spec.AbortWhenErrorCount = typed
		case int64:
			spec.AbortWhenErrorCount = int(typed)
		}
	}
	if len(args) > 1 {
		switch typed := args[1].(type) {
		case float64:
			spec.StartCheckAfterSeconds = typed
		case float32:
			spec.StartCheckAfterSeconds = float64(typed)
		}
	}
}
