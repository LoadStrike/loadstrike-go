package loadstrike

import "strings"

type stepRuntimeContext struct {
	ScenarioName string
	StepName     string
}

// LoadStrikeScenarioContext is the public callback context passed to scenario and step functions.
type LoadStrikeScenarioContext struct {
	native *stepRuntimeContext
}

type LoadStrikeScenarioRunFunc func(LoadStrikeScenarioContext) LoadStrikeReply

type scenarioStep struct {
	Name string
	Run  func(*stepRuntimeContext) replyResult
}

type scenarioDefinition struct {
	Name            string
	Steps           []scenarioStep
	LoadSimulations []LoadSimulation
}

// LoadStrikeScenario is the public scenario wrapper used by the runner.
type LoadStrikeScenario struct {
	native scenarioDefinition
}

func (s LoadStrikeScenario) nativeValue() scenarioDefinition {
	return s.native
}

// CreateScenario creates a scenario with one primary step.
func CreateScenario(name string, run LoadStrikeScenarioRunFunc) LoadStrikeScenario {
	if strings.TrimSpace(name) == "" {
		panic("scenario name must be provided")
	}
	if run == nil {
		panic("scenario run must be provided")
	}
	return LoadStrikeScenario{
		native: scenarioDefinition{
			Name: name,
			Steps: []scenarioStep{
				{
					Name: name,
					Run: func(context *stepRuntimeContext) replyResult {
						return run(LoadStrikeScenarioContext{native: context}).toNative()
					},
				},
			},
		},
	}
}

// WithLoadSimulations replaces the scenario load-shape configuration.
func (s LoadStrikeScenario) WithLoadSimulations(loadSimulations ...LoadSimulation) LoadStrikeScenario {
	s.native.LoadSimulations = append([]LoadSimulation(nil), loadSimulations...)
	return s
}

type loadStrikeStepNamespace struct{}

// LoadStrikeStep exposes named-step helpers matching the public SDK style.
var LoadStrikeStep loadStrikeStepNamespace

func (loadStrikeStepNamespace) Run(name string, context LoadStrikeScenarioContext, run LoadStrikeScenarioRunFunc) LoadStrikeReply {
	if context.native == nil {
		panic("step context must be provided")
	}
	if strings.TrimSpace(name) == "" {
		panic("step name must be provided")
	}
	if run == nil {
		panic("step run must be provided")
	}
	if context.native.StepName == "" {
		context.native.StepName = name
	}
	return run(context)
}
