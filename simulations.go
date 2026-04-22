package loadstrike

// LoadSimulation defines a public load-shape input for the runner.
type LoadSimulation struct {
	Kind            string
	Copies          int
	Iterations      int
	Rate            int
	MinRate         int
	MaxRate         int
	IntervalSeconds float64
	DuringSeconds   float64
	PauseSeconds    float64
}

// Inject creates a rate-based simulation descriptor.
func Inject(rate int, intervalSeconds float64, duringSeconds float64) LoadSimulation {
	return LoadSimulation{
		Kind:            "Inject",
		Copies:          maxInt(rate, 1),
		Iterations:      maxInt(estimateIterations(intervalSeconds, duringSeconds), 1),
		Rate:            rate,
		MinRate:         maxInt(rate, 1),
		MaxRate:         maxInt(rate, 1),
		IntervalSeconds: intervalSeconds,
		DuringSeconds:   duringSeconds,
	}
}

// RampingInject creates a ramping rate simulation descriptor.
func RampingInject(rate int, intervalSeconds float64, duringSeconds float64) LoadSimulation {
	simulation := Inject(rate, intervalSeconds, duringSeconds)
	simulation.Kind = "RampingInject"
	return simulation
}

// KeepConstant creates a constant concurrency simulation descriptor.
func KeepConstant(copies int, duringSeconds float64) LoadSimulation {
	return LoadSimulation{
		Kind:          "KeepConstant",
		Copies:        maxInt(copies, 1),
		Iterations:    maxInt(int(duringSeconds), 1),
		DuringSeconds: duringSeconds,
	}
}

// RampingConstant creates a ramping concurrency simulation descriptor.
func RampingConstant(copies int, duringSeconds float64) LoadSimulation {
	simulation := KeepConstant(copies, duringSeconds)
	simulation.Kind = "RampingConstant"
	return simulation
}

// InjectRandom creates a randomized rate simulation descriptor.
func InjectRandom(minRate int, maxRate int, intervalSeconds float64, duringSeconds float64) LoadSimulation {
	if maxRate < minRate {
		minRate, maxRate = maxRate, minRate
	}
	simulation := Inject(maxInt(maxRate, 1), intervalSeconds, duringSeconds)
	simulation.Kind = "InjectRandom"
	simulation.MinRate = maxInt(minRate, 1)
	simulation.MaxRate = maxInt(maxRate, simulation.MinRate)
	return simulation
}

// Pause creates a pause simulation descriptor.
func Pause(seconds float64) LoadSimulation {
	return LoadSimulation{
		Kind:         "Pause",
		Copies:       1,
		Iterations:   1,
		PauseSeconds: seconds,
	}
}

// IterationsForConstant creates a fixed-iteration simulation descriptor.
func IterationsForConstant(copies int, iterations int) LoadSimulation {
	return LoadSimulation{
		Kind:       "IterationsForConstant",
		Copies:     maxInt(copies, 1),
		Iterations: maxInt(iterations, 1),
	}
}

// IterationsForInject creates a fixed-iteration inject simulation descriptor.
func IterationsForInject(rate int, intervalSeconds float64, iterations int) LoadSimulation {
	return LoadSimulation{
		Kind:            "IterationsForInject",
		Copies:          maxInt(rate, 1),
		Iterations:      maxInt(iterations, 1),
		Rate:            maxInt(rate, 1),
		MinRate:         maxInt(rate, 1),
		MaxRate:         maxInt(rate, 1),
		IntervalSeconds: intervalSeconds,
	}
}

func estimateIterations(intervalSeconds float64, duringSeconds float64) int {
	if intervalSeconds <= 0 || duringSeconds <= 0 {
		return 1
	}

	estimated := int(duringSeconds / intervalSeconds)
	return maxInt(estimated, 1)
}

func maxInt(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
}
