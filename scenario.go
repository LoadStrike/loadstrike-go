package loadstrike

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// scenarioHookContext carries scenario-level metadata for init and clean hooks.
type scenarioHookContext struct {
	ScenarioName          string
	TestSuite             string
	TestName              string
	ScenarioInstanceData  map[string]any
	Partition             scenarioPartitionInfo
	scenarioPartitionInfo scenarioPartitionInfo
	Logger                *LoadStrikeLogger
	nodeInfo              nodeInfo
	testInfo              testInfo
	ScenarioInfo          LoadStrikeScenarioInfo
	CustomSettings        IConfiguration
	GlobalCustomSettings  IConfiguration
	metricRegistry        *metricRegistry
}

// scenarioPartitionInfo identifies the shard assigned to a scenario copy or node.
type scenarioPartitionInfo struct {
	Number int
	Count  int
}

// scenarioDefinition defines a runnable workload scenario.
type scenarioDefinition struct {
	Name                   string
	Init                   func(*scenarioHookContext) error
	Clean                  func(*scenarioHookContext) error
	Weight                 int
	RestartIterationOnFail bool
	MaxFailCount           int
	WarmUpDurationSeconds  float64
	WarmUpDisabled         bool
	LoadSimulations        []LoadSimulation
	Thresholds             []ThresholdSpec
	Tracking               *TrackingConfigurationSpec
	Steps                  []scenarioStep
}

type scenarioLike interface {
	nativeValue() scenarioDefinition
}

func normalizeScenarioLike(value scenarioLike) scenarioDefinition {
	if value == nil {
		panic("scenario must be provided.")
	}
	return value.nativeValue()
}

func (s scenarioDefinition) nativeValue() scenarioDefinition {
	return s
}

func emptyScenario(name string) scenarioDefinition {
	if strings.TrimSpace(name) == "" {
		panic("scenario name must be provided.")
	}
	return scenarioDefinition{Name: name}
}

// EmptyScenario creates a named scenario shell that can be configured fluently.
func EmptyScenario(name string) LoadStrikeScenario {
	return newLoadStrikeScenario(emptyScenario(name))
}

func createScenario(name string, run any) scenarioDefinition {
	if strings.TrimSpace(name) == "" {
		panic("scenario name must be provided.")
	}
	return scenarioDefinition{
		Name: name,
		Steps: []scenarioStep{
			{
				Name: name,
				Run:  normalizeScenarioRun(run),
			},
		},
	}
}

// CreateScenario creates a scenario with one primary runnable step.
func CreateScenario(name string, run LoadStrikeScenarioRunFunc) LoadStrikeScenario {
	return newLoadStrikeScenario(createScenario(name, run))
}

// WithInit sets the scenario init hook.
func (s scenarioDefinition) WithInit(init any) scenarioDefinition {
	s.Init = normalizeScenarioInitHook(init, "scenario init hook must be provided.")
	return s
}

// WithClean sets the scenario cleanup hook.
func (s scenarioDefinition) WithClean(clean any) scenarioDefinition {
	s.Clean = normalizeScenarioInitHook(clean, "scenario clean hook must be provided.")
	return s
}

// WithWeight sets the scenario execution weight.
func (s scenarioDefinition) WithWeight(weight int) scenarioDefinition {
	s.Weight = weight
	return s
}

// WithRestartIterationOnFail toggles failed-iteration restarts.
func (s scenarioDefinition) WithRestartIterationOnFail(enabled bool) scenarioDefinition {
	s.RestartIterationOnFail = enabled
	return s
}

// WithMaxFailCount stops the scenario after the requested number of failures.
func (s scenarioDefinition) WithMaxFailCount(maxFailCount int) scenarioDefinition {
	s.MaxFailCount = maxFailCount
	return s
}

// WithWarmUpDuration sets a scenario warmup period before measured execution begins.
func (s scenarioDefinition) WithWarmUpDuration(seconds float64) scenarioDefinition {
	s.WarmUpDurationSeconds = seconds
	s.WarmUpDisabled = false
	return s
}

// GetScenarioTimerTime returns the elapsed scenario time for init/clean hooks.
func (c *scenarioHookContext) GetScenarioTimerTime() time.Duration {
	if c == nil {
		return 0
	}
	return c.ScenarioInfo.ScenarioDuration
}

func (c *scenarioHookContext) registerMetric(metric IMetric) {
	if c == nil {
		panic("Metric must be provided.")
	}
	if metric == nil {
		panic("Metric must be provided.")
	}
	if c.metricRegistry == nil {
		c.metricRegistry = &metricRegistry{}
	}
	c.metricRegistry.register(metric)
}

// WithoutWarmUp disables scenario warmup.
func (s scenarioDefinition) WithoutWarmUp() scenarioDefinition {
	s.WarmUpDisabled = true
	return s
}

// WithLoadSimulations replaces the configured load simulations.
func (s scenarioDefinition) WithLoadSimulations(loadSimulations ...LoadSimulation) scenarioDefinition {
	if len(loadSimulations) == 0 {
		panic("At least one load simulation should be provided.")
	}
	s.LoadSimulations = append([]LoadSimulation(nil), loadSimulations...)
	return s
}

// WithThresholds replaces the configured thresholds.
func (s scenarioDefinition) WithThresholds(thresholds ...ThresholdSpec) scenarioDefinition {
	if len(thresholds) == 0 {
		panic("At least one threshold should be provided.")
	}
	s.Thresholds = append([]ThresholdSpec(nil), thresholds...)
	return s
}

// WithTrackingConfiguration sets cross-platform tracking configuration.
func (s scenarioDefinition) WithTrackingConfiguration(tracking *TrackingConfigurationSpec) scenarioDefinition {
	s.Tracking = tracking
	return s
}

// WithCrossPlatformTracking is a convenience alias for WithTrackingConfiguration.
func (s scenarioDefinition) WithCrossPlatformTracking(tracking *TrackingConfigurationSpec) scenarioDefinition {
	return s.WithTrackingConfiguration(tracking)
}

// WithSteps replaces the configured scenario steps.
func (s scenarioDefinition) WithSteps(steps ...scenarioStep) scenarioDefinition {
	if len(steps) == 0 {
		panic("At least one step should be provided.")
	}
	for _, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			panic("step name must be provided.")
		}
		if step.Run == nil {
			panic("step run must be provided.")
		}
	}
	s.Steps = append([]scenarioStep(nil), steps...)
	return s
}

func normalizeScenarioRun(run any) func(*stepRuntimeContext) replyResult {
	if callable := reflect.ValueOf(run); callable.IsValid() && callable.Kind() == reflect.Func && callable.IsNil() {
		panic("scenario run must be provided.")
	}
	switch typed := run.(type) {
	case nil:
		panic("scenario run must be provided.")
	case func(*stepRuntimeContext) replyResult:
		return typed
	case func(LoadStrikeScenarioContext) replyResult:
		return func(context *stepRuntimeContext) replyResult {
			return typed(newLoadStrikeScenarioContext(context))
		}
	case func(*stepRuntimeContext) (replyResult, error):
		return func(context *stepRuntimeContext) replyResult {
			reply, err := typed(context)
			if err != nil {
				return failReply("500", err.Error())
			}
			return reply
		}
	case func(LoadStrikeScenarioContext) (replyResult, error):
		return func(context *stepRuntimeContext) replyResult {
			reply, err := typed(newLoadStrikeScenarioContext(context))
			if err != nil {
				return failReply("500", err.Error())
			}
			return reply
		}
	default:
		callable := reflect.ValueOf(run)
		if callable.Kind() != reflect.Func {
			panic(fmt.Sprintf("unsupported scenario run type %T", run))
		}
		return func(context *stepRuntimeContext) replyResult {
			inputs := []reflect.Value{}
			switch callable.Type().NumIn() {
			case 0:
			case 1:
				switch callable.Type().In(0) {
				case reflect.TypeOf((*stepRuntimeContext)(nil)):
					inputs = append(inputs, reflect.ValueOf(context))
				case reflect.TypeOf(LoadStrikeScenarioContext{}):
					inputs = append(inputs, reflect.ValueOf(newLoadStrikeScenarioContext(context)))
				default:
					panic(fmt.Sprintf("unsupported scenario run type %T", run))
				}
			default:
				panic(fmt.Sprintf("unsupported scenario run type %T", run))
			}
			return invokeReplyCallable(callable, inputs, true)
		}
	}
}

func normalizeScenarioInitHook(hook any, nilMessage string) func(*scenarioHookContext) error {
	if callable := reflect.ValueOf(hook); callable.IsValid() && callable.Kind() == reflect.Func && callable.IsNil() {
		panic(nilMessage)
	}
	switch typed := hook.(type) {
	case nil:
		panic(nilMessage)
	case func(*scenarioHookContext) error:
		return typed
	case LoadStrikeScenarioInitFunc:
		return func(context *scenarioHookContext) error {
			return typed(newLoadStrikeScenarioInitContext(context))
		}
	case func(LoadStrikeScenarioInitContext) error:
		return func(context *scenarioHookContext) error {
			return typed(newLoadStrikeScenarioInitContext(context))
		}
	case func(*scenarioHookContext):
		return func(context *scenarioHookContext) error {
			typed(context)
			return nil
		}
	case func(LoadStrikeScenarioInitContext):
		return func(context *scenarioHookContext) error {
			typed(newLoadStrikeScenarioInitContext(context))
			return nil
		}
	case func(*scenarioHookContext) LoadStrikeTask:
		return func(context *scenarioHookContext) error {
			return typed(context).Await()
		}
	case LoadStrikeAsyncScenarioInitFunc:
		return func(context *scenarioHookContext) error {
			return typed(newLoadStrikeScenarioInitContext(context)).Await()
		}
	case func(LoadStrikeScenarioInitContext) LoadStrikeTask:
		return func(context *scenarioHookContext) error {
			return typed(newLoadStrikeScenarioInitContext(context)).Await()
		}
	default:
		panic(fmt.Sprintf("unsupported scenario hook type %T", hook))
	}
}
