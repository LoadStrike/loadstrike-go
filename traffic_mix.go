package loadstrike

import "strings"

type trafficMixScenarioShare struct {
	Scenario scenarioDefinition
	Weight   int
}

type trafficMixDefinition struct {
	Name        string
	TotalLoad   []LoadSimulation
	ScenarioMix []trafficMixScenarioShare
}

type LoadStrikeScenarioShareDefinition struct {
	native trafficMixScenarioShare
}

type loadStrikeScenarioShareNamespace struct{}

var LoadStrikeScenarioShare loadStrikeScenarioShareNamespace

func (loadStrikeScenarioShareNamespace) Create(scenario LoadStrikeScenario, weight int) LoadStrikeScenarioShareDefinition {
	if scenario.native == nil {
		panic("scenario must be provided.")
	}
	if weight <= 0 {
		panic("Scenario share weight should be greater than zero.")
	}
	return LoadStrikeScenarioShareDefinition{
		native: trafficMixScenarioShare{
			Scenario: scenario.nativeValue(),
			Weight:   weight,
		},
	}
}

func (s LoadStrikeScenarioShareDefinition) Scenario() LoadStrikeScenario {
	return newLoadStrikeScenario(s.native.Scenario)
}

func (s LoadStrikeScenarioShareDefinition) Weight() int {
	return s.native.Weight
}

func (s LoadStrikeScenarioShareDefinition) nativeValue() trafficMixScenarioShare {
	return s.native
}

type LoadStrikeTrafficMixDefinition struct {
	native trafficMixDefinition
}

type loadStrikeTrafficMixNamespace struct{}

var LoadStrikeTrafficMix loadStrikeTrafficMixNamespace

func (loadStrikeTrafficMixNamespace) Create(name string) LoadStrikeTrafficMixDefinition {
	if strings.TrimSpace(name) == "" {
		panic("Traffic mix name must be provided.")
	}
	return LoadStrikeTrafficMixDefinition{native: trafficMixDefinition{Name: name}}
}

func (m LoadStrikeTrafficMixDefinition) WithTotalLoad(totalLoad ...LoadStrikeLoadSimulation) LoadStrikeTrafficMixDefinition {
	if len(totalLoad) == 0 {
		panic("At least one total load simulation should be provided.")
	}
	native := m.nativeValue()
	native.TotalLoad = make([]LoadSimulation, 0, len(totalLoad))
	for _, simulation := range totalLoad {
		native.TotalLoad = append(native.TotalLoad, simulation.nativeValue())
	}
	return LoadStrikeTrafficMixDefinition{native: native}
}

func (m LoadStrikeTrafficMixDefinition) WithScenarioMix(scenarioMix ...LoadStrikeScenarioShareDefinition) LoadStrikeTrafficMixDefinition {
	if len(scenarioMix) == 0 {
		panic("At least one scenario share should be provided.")
	}
	native := m.nativeValue()
	native.ScenarioMix = make([]trafficMixScenarioShare, 0, len(scenarioMix))
	for _, share := range scenarioMix {
		native.ScenarioMix = append(native.ScenarioMix, share.nativeValue())
	}
	return LoadStrikeTrafficMixDefinition{native: native}
}

func (m LoadStrikeTrafficMixDefinition) Name() string {
	return m.native.Name
}

func (m LoadStrikeTrafficMixDefinition) nativeValue() trafficMixDefinition {
	native := m.native
	native.TotalLoad = append([]LoadSimulation(nil), native.TotalLoad...)
	native.ScenarioMix = append([]trafficMixScenarioShare(nil), native.ScenarioMix...)
	return native
}
