package loadstrike

type step = scenarioStep

type compatStep struct {
	Name string
	Run  any
}

func (s compatStep) toNative() scenarioStep {
	return scenarioStep{
		Name: s.Name,
		Run:  normalizeScenarioRun(s.Run),
	}
}

type compatScenario struct {
	Name                   string
	Init                   any
	Clean                  any
	Weight                 int
	RestartIterationOnFail bool
	MaxFailCount           int
	WarmUpDurationSeconds  float64
	WarmUpDisabled         bool
	LoadSimulations        []LoadSimulation
	Thresholds             []ThresholdSpec
	Tracking               *TrackingConfigurationSpec
	Steps                  []compatStep
}

func (s compatScenario) nativeValue() scenarioDefinition {
	native := scenarioDefinition{
		Name:                   s.Name,
		Weight:                 s.Weight,
		RestartIterationOnFail: s.RestartIterationOnFail,
		MaxFailCount:           s.MaxFailCount,
		WarmUpDurationSeconds:  s.WarmUpDurationSeconds,
		WarmUpDisabled:         s.WarmUpDisabled,
		LoadSimulations:        append([]LoadSimulation(nil), s.LoadSimulations...),
		Thresholds:             append([]ThresholdSpec(nil), s.Thresholds...),
		Tracking:               s.Tracking,
	}
	if s.Init != nil {
		native.Init = normalizeScenarioInitHook(s.Init, "scenario init hook must be provided.")
	}
	if s.Clean != nil {
		native.Clean = normalizeScenarioInitHook(s.Clean, "scenario clean hook must be provided.")
	}
	if len(s.Steps) > 0 {
		nativeSteps := make([]scenarioStep, 0, len(s.Steps))
		for _, step := range s.Steps {
			nativeSteps = append(nativeSteps, step.toNative())
		}
		native.Steps = nativeSteps
	}
	return native
}

func newCompatScenario(native scenarioDefinition) compatScenario {
	steps := make([]compatStep, 0, len(native.Steps))
	for _, step := range native.Steps {
		steps = append(steps, compatStep{Name: step.Name, Run: step.Run})
	}
	return compatScenario{
		Name:                   native.Name,
		Weight:                 native.Weight,
		RestartIterationOnFail: native.RestartIterationOnFail,
		MaxFailCount:           native.MaxFailCount,
		WarmUpDurationSeconds:  native.WarmUpDurationSeconds,
		WarmUpDisabled:         native.WarmUpDisabled,
		LoadSimulations:        append([]LoadSimulation(nil), native.LoadSimulations...),
		Thresholds:             append([]ThresholdSpec(nil), native.Thresholds...),
		Tracking:               native.Tracking,
		Steps:                  steps,
	}
}

func (s compatScenario) WithInit(init any) compatScenario {
	return newCompatScenario(s.nativeValue().WithInit(init))
}

func (s compatScenario) WithClean(clean any) compatScenario {
	return newCompatScenario(s.nativeValue().WithClean(clean))
}

func (s compatScenario) WithWeight(weight int) compatScenario {
	return newCompatScenario(s.nativeValue().WithWeight(weight))
}

func (s compatScenario) WithRestartIterationOnFail(enabled bool) compatScenario {
	return newCompatScenario(s.nativeValue().WithRestartIterationOnFail(enabled))
}

func (s compatScenario) WithMaxFailCount(maxFailCount int) compatScenario {
	return newCompatScenario(s.nativeValue().WithMaxFailCount(maxFailCount))
}

func (s compatScenario) WithWarmUpDuration(seconds float64) compatScenario {
	return newCompatScenario(s.nativeValue().WithWarmUpDuration(seconds))
}

func (s compatScenario) WithoutWarmUp() compatScenario {
	return newCompatScenario(s.nativeValue().WithoutWarmUp())
}

func (s compatScenario) WithLoadSimulations(loadSimulations ...LoadSimulation) compatScenario {
	return newCompatScenario(s.nativeValue().WithLoadSimulations(loadSimulations...))
}

func (s compatScenario) WithThresholds(thresholds ...ThresholdSpec) compatScenario {
	return newCompatScenario(s.nativeValue().WithThresholds(thresholds...))
}

func (s compatScenario) WithTrackingConfiguration(tracking *TrackingConfigurationSpec) compatScenario {
	return newCompatScenario(s.nativeValue().WithTrackingConfiguration(tracking))
}

func (s compatScenario) WithCrossPlatformTracking(tracking *TrackingConfigurationSpec) compatScenario {
	return newCompatScenario(s.nativeValue().WithCrossPlatformTracking(tracking))
}

func (s compatScenario) WithSteps(steps ...compatStep) compatScenario {
	args := make([]any, 0, len(steps))
	for _, step := range steps {
		args = append(args, step)
	}
	return newCompatScenario(s.nativeValue().WithSteps(normalizePublicSteps(args...)...))
}

func normalizePublicSteps(values ...any) []scenarioStep {
	if len(values) == 0 {
		return nil
	}
	steps := make([]scenarioStep, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case scenarioStep:
			steps = append(steps, typed)
		case compatStep:
			steps = append(steps, typed.toNative())
		default:
			panic("At least one step should be provided.")
		}
	}
	return steps
}
