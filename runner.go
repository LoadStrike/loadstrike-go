package loadstrike

type runnerState struct {
	context contextState
}

// LoadStrikeRunner composes scenarios and execution settings before a run.
type LoadStrikeRunner struct {
	native *runnerState
}

// NewRunner creates a runner with default public settings.
func NewRunner() LoadStrikeRunner {
	return LoadStrikeRunner{
		native: &runnerState{
			context: contextState{
				ReportsEnabled: true,
				scenarios:      []scenarioDefinition{},
			},
		},
	}
}

// Create mirrors the current static runner constructor.
func Create() LoadStrikeRunner {
	return NewRunner()
}

func requireRunner(runner *runnerState) *runnerState {
	if runner == nil {
		panic("runner must be provided")
	}
	return runner
}

// AddScenario registers a scenario on the runner.
func (r LoadStrikeRunner) AddScenario(scenario LoadStrikeScenario) LoadStrikeRunner {
	requireRunner(r.native).context.scenarios = append(requireRunner(r.native).context.scenarios, scenario.nativeValue())
	return r
}

// WithRunnerKey records the runner key used for runtime resolution and licensing.
func (r LoadStrikeRunner) WithRunnerKey(runnerKey string) LoadStrikeRunner {
	requireRunner(r.native).context.RunnerKey = runnerKey
	return r
}

// WithoutReports disables local report generation for the run.
func (r LoadStrikeRunner) WithoutReports() LoadStrikeRunner {
	requireRunner(r.native).context.ReportsEnabled = false
	return r
}

// BuildContext returns a reusable execution context snapshot.
func (r LoadStrikeRunner) BuildContext() LoadStrikeContext {
	state := requireRunner(r.native).context
	state.scenarios = append([]scenarioDefinition(nil), state.scenarios...)
	return wrapLoadStrikeContext(&state)
}

// Run builds a context snapshot and executes it through the private runtime.
func (r LoadStrikeRunner) Run(args ...string) LoadStrikeRunResult {
	return r.BuildContext().Run(args...)
}
