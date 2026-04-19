package loadstrike

import "errors"

type runtimePlan struct {
	SDKVersion      string                `json:"sdkVersion"`
	ProtocolVersion int                   `json:"protocolVersion"`
	Context         runtimeContextPlan    `json:"context"`
	Scenarios       []runtimeScenarioPlan `json:"scenarios"`
}

type runtimeContextPlan struct {
	RunnerKey      string `json:"runnerKey"`
	ReportsEnabled bool   `json:"reportsEnabled"`
}

type runtimeScenarioPlan struct {
	Name                  string           `json:"name"`
	PrimaryStepCallbackID string           `json:"primaryStepCallbackId"`
	StepNames             []string         `json:"stepNames,omitempty"`
	LoadSimulations       []LoadSimulation `json:"loadSimulations,omitempty"`
}

func newRuntimeContextPlan(context contextState) runtimeContextPlan {
	return runtimeContextPlan{
		RunnerKey:      context.RunnerKey,
		ReportsEnabled: context.ReportsEnabled,
	}
}

func buildRuntimePlan(context *contextState) (runtimePlan, *runtimeCallbackRegistry, error) {
	if context == nil {
		return runtimePlan{}, nil, errors.New("context must be provided")
	}

	registry := newRuntimeCallbackRegistry()
	plan := runtimePlan{
		SDKVersion:      RuntimeArtifactVersion(),
		ProtocolVersion: RuntimeProtocolVersion(),
		Context:         newRuntimeContextPlan(*context),
		Scenarios:       make([]runtimeScenarioPlan, 0, len(context.scenarios)),
	}

	for _, scenario := range context.scenarios {
		item := runtimeScenarioPlan{
			Name:            scenario.Name,
			StepNames:       make([]string, 0, len(scenario.Steps)),
			LoadSimulations: append([]LoadSimulation(nil), scenario.LoadSimulations...),
		}
		for index, step := range scenario.Steps {
			item.StepNames = append(item.StepNames, step.Name)
			if index == 0 {
				item.PrimaryStepCallbackID = registry.registerScenarioStep(scenario.Name, step.Name, step.Run)
			}
		}
		plan.Scenarios = append(plan.Scenarios, item)
	}

	return plan, registry, nil
}
