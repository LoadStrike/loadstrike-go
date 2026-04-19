package loadstrike

// LoadSimulation stores public load-shape configuration for a scenario.
type LoadSimulation struct {
	Kind       string
	Copies     int
	Iterations int
}

type loadStrikeSimulationNamespace struct{}

// LoadStrikeSimulation exposes load-shape constructors matching the current SDK style.
var LoadStrikeSimulation loadStrikeSimulationNamespace

func (loadStrikeSimulationNamespace) IterationsForConstant(copies, iterations int) LoadSimulation {
	if copies <= 0 {
		copies = 1
	}
	if iterations <= 0 {
		iterations = 1
	}
	return LoadSimulation{
		Kind:       "IterationsForConstant",
		Copies:     copies,
		Iterations: iterations,
	}
}
