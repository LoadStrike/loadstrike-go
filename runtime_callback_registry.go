package loadstrike

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

type runtimeRegisteredCallback struct {
	ID           string
	ScenarioName string
	StepName     string
	Run          func(*stepRuntimeContext) replyResult
}

type runtimeScenarioMetricDescriptor struct {
	Kind          string `json:"kind"`
	MetricName    string `json:"metricName"`
	UnitOfMeasure string `json:"unitOfMeasure"`
}

type runtimeScenarioMetricsSnapshot struct {
	Counters []LoadStrikeCounterStats `json:"counters,omitempty"`
	Gauges   []LoadStrikeGaugeStats   `json:"gauges,omitempty"`
}

type runtimeScenarioBridgeState struct {
	mu                   sync.Mutex
	registeredMetrics    []IMetric
	scenarioInstanceData map[string]any
}

func newRuntimeScenarioBridgeState() *runtimeScenarioBridgeState {
	return &runtimeScenarioBridgeState{
		scenarioInstanceData: map[string]any{},
	}
}

func (s *runtimeScenarioBridgeState) replaceMetrics(metrics []IMetric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.registeredMetrics = append([]IMetric(nil), metrics...)
}

func (s *runtimeScenarioBridgeState) scenarioInstanceDataForRequest() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scenarioInstanceData == nil {
		s.scenarioInstanceData = map[string]any{}
	}
	return s.scenarioInstanceData
}

func (s *runtimeScenarioBridgeState) metricDescriptors() []runtimeScenarioMetricDescriptor {
	s.mu.Lock()
	defer s.mu.Unlock()

	return runtimeMetricDescriptors(s.registeredMetrics)
}

func (s *runtimeScenarioBridgeState) metricSnapshot() runtimeScenarioMetricsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := runtimeScenarioMetricsSnapshot{}
	for _, metric := range s.registeredMetrics {
		switch typed := metric.(type) {
		case ICounter:
			snapshot.Counters = append(snapshot.Counters, LoadStrikeCounterStats{
				MetricName:    typed.MetricName(),
				UnitOfMeasure: typed.UnitOfMeasure(),
				Value:         typed.Value(),
			})
		case IGauge:
			snapshot.Gauges = append(snapshot.Gauges, LoadStrikeGaugeStats{
				MetricName:    typed.MetricName(),
				UnitOfMeasure: typed.UnitOfMeasure(),
				Value:         typed.Value(),
			})
		}
	}
	return snapshot
}

func runtimeMetricDescriptors(metrics []IMetric) []runtimeScenarioMetricDescriptor {
	if len(metrics) == 0 {
		return nil
	}

	descriptors := make([]runtimeScenarioMetricDescriptor, 0, len(metrics))
	for _, metric := range metrics {
		switch typed := metric.(type) {
		case ICounter:
			descriptors = append(descriptors, runtimeScenarioMetricDescriptor{
				Kind:          "counter",
				MetricName:    typed.MetricName(),
				UnitOfMeasure: typed.UnitOfMeasure(),
			})
		case IGauge:
			descriptors = append(descriptors, runtimeScenarioMetricDescriptor{
				Kind:          "gauge",
				MetricName:    typed.MetricName(),
				UnitOfMeasure: typed.UnitOfMeasure(),
			})
		}
	}

	return descriptors
}

type runtimeScenarioRegistration struct {
	ScenarioName string
	Run          func(*stepRuntimeContext) replyResult
	Init         func(*scenarioHookContext) error
	Clean        func(*scenarioHookContext) error
	State        *runtimeScenarioBridgeState
}

type runtimeTrackingRegistration struct {
	Produce       func(context.Context, TrackingPayload) (EndpointProduceResult, error)
	Consume       func(context.Context, func(EndpointConsumeEvent) error) error
	consumeStream *runtimeTrackingConsumeSession
}

type runtimeTrackingConsumeSession struct {
	consume func(context.Context, func(EndpointConsumeEvent) error) error

	startOnce sync.Once

	mu        sync.Mutex
	messages  []EndpointConsumeEvent
	completed bool
	err       error
	closed    bool
	cancel    context.CancelFunc
	wake      chan struct{}
}

func newRuntimeTrackingConsumeSession(
	consume func(context.Context, func(EndpointConsumeEvent) error) error,
) *runtimeTrackingConsumeSession {
	return &runtimeTrackingConsumeSession{
		consume: consume,
		wake:    make(chan struct{}, 1),
	}
}

func (s *runtimeTrackingConsumeSession) poll(ctx context.Context) ([]EndpointConsumeEvent, bool, error) {
	if s == nil {
		return nil, true, nil
	}

	s.start()
	timer := time.NewTimer(20 * time.Millisecond)
	defer timer.Stop()

	for {
		messages, completed, err := s.snapshot()
		if len(messages) > 0 || completed || err != nil {
			return messages, completed, err
		}

		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-s.wake:
		case <-timer.C:
			return s.snapshot()
		}
	}
}

// Close releases owned resources. Use this when the current SDK object is no longer needed.
func (s *runtimeTrackingConsumeSession) Close() {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.closed = true
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (s *runtimeTrackingConsumeSession) start() {
	if s == nil {
		return
	}

	s.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		s.mu.Lock()
		s.cancel = cancel
		s.mu.Unlock()

		go func() {
			err := s.consume(ctx, func(event EndpointConsumeEvent) error {
				s.mu.Lock()
				s.messages = append(s.messages, event)
				s.mu.Unlock()
				s.notify()
				return nil
			})

			s.mu.Lock()
			if !(s.closed && errors.Is(err, context.Canceled)) {
				s.err = err
			}
			s.completed = true
			s.mu.Unlock()
			s.notify()
		}()
	})
}

func (s *runtimeTrackingConsumeSession) snapshot() ([]EndpointConsumeEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages := append([]EndpointConsumeEvent(nil), s.messages...)
	s.messages = nil
	return messages, s.completed, s.err
}

func (s *runtimeTrackingConsumeSession) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

type runtimeCallbackRegistry struct {
	mu sync.RWMutex

	nextID int

	callbacks map[string]runtimeRegisteredCallback
	scenarios map[string]runtimeScenarioRegistration
	plugins   map[string]LoadStrikeWorkerPlugin
	sinks     map[string]LoadStrikeReportingSink
	policies  map[string]LoadStrikeRuntimePolicy
	tracking  map[string]runtimeTrackingRegistration
}

func newRuntimeCallbackRegistry() *runtimeCallbackRegistry {
	return &runtimeCallbackRegistry{
		callbacks: map[string]runtimeRegisteredCallback{},
		scenarios: map[string]runtimeScenarioRegistration{},
		plugins:   map[string]LoadStrikeWorkerPlugin{},
		sinks:     map[string]LoadStrikeReportingSink{},
		policies:  map[string]LoadStrikeRuntimePolicy{},
		tracking:  map[string]runtimeTrackingRegistration{},
	}
}

func (r *runtimeCallbackRegistry) nextCallbackID(prefix string) string {
	r.nextID++
	return prefix + "-" + strconv.Itoa(r.nextID)
}

func (r *runtimeCallbackRegistry) registerScenarioStep(scenarioName, stepName string, run func(*stepRuntimeContext) replyResult) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("step")
	r.callbacks[id] = runtimeRegisteredCallback{
		ID:           id,
		ScenarioName: scenarioName,
		StepName:     stepName,
		Run:          run,
	}
	return id
}

func (r *runtimeCallbackRegistry) registerScenario(scenario scenarioDefinition) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("scenario")
	r.scenarios[id] = runtimeScenarioRegistration{
		ScenarioName: scenario.Name,
		Run:          primaryScenarioRun(scenario),
		Init:         scenario.Init,
		Clean:        scenario.Clean,
		State:        newRuntimeScenarioBridgeState(),
	}
	return id
}

func primaryScenarioRun(scenario scenarioDefinition) func(*stepRuntimeContext) replyResult {
	if len(scenario.Steps) == 0 || scenario.Steps[0].Run == nil {
		return nil
	}
	return scenario.Steps[0].Run
}

func (r *runtimeCallbackRegistry) registerWorkerPlugin(plugin LoadStrikeWorkerPlugin) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("plugin")
	r.plugins[id] = plugin
	return id
}

func (r *runtimeCallbackRegistry) registerReportingSink(sink LoadStrikeReportingSink) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("sink")
	r.sinks[id] = sink
	return id
}

func (r *runtimeCallbackRegistry) registerRuntimePolicy(policy LoadStrikeRuntimePolicy) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("policy")
	r.policies[id] = policy
	return id
}

func (r *runtimeCallbackRegistry) registerTrackingProduce(produce func(context.Context, TrackingPayload) (EndpointProduceResult, error)) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("tracking-produce")
	r.tracking[id] = runtimeTrackingRegistration{Produce: produce}
	return id
}

func (r *runtimeCallbackRegistry) registerTrackingConsume(consume func(context.Context, func(EndpointConsumeEvent) error) error) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextCallbackID("tracking-consume")
	r.tracking[id] = runtimeTrackingRegistration{
		Consume:       consume,
		consumeStream: newRuntimeTrackingConsumeSession(consume),
	}
	return id
}

func (r *runtimeCallbackRegistry) lookup(id string) (runtimeRegisteredCallback, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.callbacks[id]
	return value, ok
}

func (r *runtimeCallbackRegistry) lookupScenario(id string) (runtimeScenarioRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.scenarios[id]
	return value, ok
}

func (r *runtimeCallbackRegistry) lookupWorkerPlugin(id string) (LoadStrikeWorkerPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.plugins[id]
	return value, ok
}

func (r *runtimeCallbackRegistry) lookupReportingSink(id string) (LoadStrikeReportingSink, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.sinks[id]
	return value, ok
}

func (r *runtimeCallbackRegistry) lookupRuntimePolicy(id string) (LoadStrikeRuntimePolicy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.policies[id]
	return value, ok
}

func (r *runtimeCallbackRegistry) lookupTracking(id string) (runtimeTrackingRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.tracking[id]
	return value, ok
}

// Close releases owned resources. Use this when the current SDK object is no longer needed.
func (r *runtimeCallbackRegistry) Close() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, registration := range r.tracking {
		if registration.consumeStream != nil {
			registration.consumeStream.Close()
		}
	}
}
