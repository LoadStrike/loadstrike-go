package loadstrike

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

type runtimeRegisteredCallback struct {
	ID           string
	ScenarioName string
	StepName     string
	Run          func(*stepRuntimeContext) replyResult
}

type runtimeThresholdRegistration struct {
	Evaluate func(runtimeHTTPThresholdRequest) (bool, error)
}

type runtimeReportingPublishRegistration struct {
	Publish func(topic, payload string) error
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

var (
	errRuntimeScenarioInstanceIdentityInvalid = errors.New("runtime scenario instance identity is invalid")
	errRuntimeScenarioInstanceGone            = errors.New("runtime scenario instance is no longer available")
)

type runtimeScenarioInstanceKey struct {
	RunIdentity        string
	NodeIdentity       string
	PartitionNumber    int
	ScenarioName       string
	ScenarioInstanceID string
}

type runtimeScenarioInstanceState struct {
	mu   sync.Mutex
	data map[string]any
}

type runtimeScenarioBridgeState struct {
	metricsMu         sync.Mutex
	registeredMetrics []IMetric

	mu        sync.Mutex
	active    map[runtimeScenarioInstanceKey]*runtimeScenarioInstanceState
	completed map[runtimeScenarioInstanceKey]struct{}
	closed    bool
	inFlight  sync.WaitGroup
}

func newRuntimeScenarioBridgeState() *runtimeScenarioBridgeState {
	return &runtimeScenarioBridgeState{
		active:    map[runtimeScenarioInstanceKey]*runtimeScenarioInstanceState{},
		completed: map[runtimeScenarioInstanceKey]struct{}{},
	}
}

func (s *runtimeScenarioBridgeState) replaceMetrics(metrics []IMetric) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	s.registeredMetrics = append([]IMetric(nil), metrics...)
}

func (s *runtimeScenarioBridgeState) admitInstance(
	key runtimeScenarioInstanceKey,
) (*runtimeScenarioInstanceState, func(), error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, nil, errRuntimeScenarioInstanceGone
	}
	if _, completed := s.completed[key]; completed {
		s.mu.Unlock()
		return nil, nil, errRuntimeScenarioInstanceGone
	}
	instance := s.active[key]
	if instance == nil {
		instance = &runtimeScenarioInstanceState{data: map[string]any{}}
		s.active[key] = instance
	}
	s.inFlight.Add(1)
	s.mu.Unlock()

	instance.mu.Lock()
	s.mu.Lock()
	_, completed := s.completed[key]
	admitted := !s.closed && !completed && s.active[key] == instance
	s.mu.Unlock()
	if !admitted {
		instance.mu.Unlock()
		s.inFlight.Done()
		return nil, nil, errRuntimeScenarioInstanceGone
	}

	release := func() {
		instance.mu.Unlock()
		s.inFlight.Done()
	}
	return instance, release, nil
}

func (s *runtimeScenarioBridgeState) withInstance(
	key runtimeScenarioInstanceKey,
	invoke func(map[string]any) error,
) error {
	instance, release, err := s.admitInstance(key)
	if err != nil {
		return err
	}
	defer release()
	return invoke(instance.data)
}

func (s *runtimeScenarioBridgeState) cleanInstance(
	key runtimeScenarioInstanceKey,
	invoke func(map[string]any) error,
) error {
	instance, release, err := s.admitInstance(key)
	if err != nil {
		return err
	}
	defer release()

	if err := invoke(instance.data); err != nil {
		return err
	}
	clear(instance.data)

	s.mu.Lock()
	if !s.closed {
		s.completed[key] = struct{}{}
		delete(s.active, key)
	}
	s.mu.Unlock()
	return nil
}

func (s *runtimeScenarioBridgeState) beginClose() map[runtimeScenarioInstanceKey]*runtimeScenarioInstanceState {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	detached := s.active
	s.active = nil
	s.completed = nil
	return detached
}

func (s *runtimeScenarioBridgeState) finishClose(
	detached map[runtimeScenarioInstanceKey]*runtimeScenarioInstanceState,
) {
	if s == nil || detached == nil {
		return
	}

	s.inFlight.Wait()
	for _, instance := range detached {
		instance.mu.Lock()
		clear(instance.data)
		instance.mu.Unlock()
	}
}

func (s *runtimeScenarioBridgeState) isClosed() bool {
	if s == nil {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *runtimeScenarioBridgeState) metricDescriptors() []runtimeScenarioMetricDescriptor {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

	return runtimeMetricDescriptors(s.registeredMetrics)
}

func (s *runtimeScenarioBridgeState) metricSnapshot() runtimeScenarioMetricsSnapshot {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()

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
	mu        sync.RWMutex
	closed    bool
	closeDone chan struct{}

	callbacks                map[string]runtimeRegisteredCallback
	scenarios                map[string]runtimeScenarioRegistration
	plugins                  map[string]LoadStrikeWorkerPlugin
	sinks                    map[string]LoadStrikeReportingSink
	policies                 map[string]LoadStrikeRuntimePolicy
	tracking                 map[string]runtimeTrackingRegistration
	thresholds               map[string]runtimeThresholdRegistration
	publishers               map[string]runtimeReportingPublishRegistration
	observationCancellations map[string]context.Context
}

func newRuntimeCallbackRegistry() *runtimeCallbackRegistry {
	return &runtimeCallbackRegistry{
		closeDone:                make(chan struct{}),
		callbacks:                map[string]runtimeRegisteredCallback{},
		scenarios:                map[string]runtimeScenarioRegistration{},
		plugins:                  map[string]LoadStrikeWorkerPlugin{},
		sinks:                    map[string]LoadStrikeReportingSink{},
		policies:                 map[string]LoadStrikeRuntimePolicy{},
		tracking:                 map[string]runtimeTrackingRegistration{},
		thresholds:               map[string]runtimeThresholdRegistration{},
		publishers:               map[string]runtimeReportingPublishRegistration{},
		observationCancellations: map[string]context.Context{},
	}
}

func (r *runtimeCallbackRegistry) nextCallbackID(prefix string) string {
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		panic(fmt.Sprintf("generate opaque runtime callback ID: %v", err))
	}
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(randomBytes[:])
}

func (r *runtimeCallbackRegistry) registerScenarioStep(scenarioName, stepName string, run func(*stepRuntimeContext) replyResult) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

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
	if r.closed {
		return ""
	}

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

func (r *runtimeCallbackRegistry) registerThreshold(threshold ThresholdSpec) (string, bool) {
	registration, ok := newRuntimeThresholdRegistration(threshold)
	if !ok {
		return "", false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return "", false
	}

	id := r.nextCallbackID("threshold")
	r.thresholds[id] = registration
	return id, true
}

func (r *runtimeCallbackRegistry) registerReportingPublisher(
	publish func(topic, payload string) error,
) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("reporting-publish")
	r.publishers[id] = runtimeReportingPublishRegistration{Publish: publish}
	return id
}

func newRuntimeThresholdRegistration(threshold ThresholdSpec) (runtimeThresholdRegistration, bool) {
	switch {
	case threshold.scenarioPredicate != nil:
		return runtimeThresholdRegistration{Evaluate: func(payload runtimeHTTPThresholdRequest) (bool, error) {
			if err := payload.validateForScope("scenario"); err != nil {
				return false, err
			}
			return invokeRuntimeThresholdPredicate(func() bool {
				return threshold.scenarioPredicate(*payload.ScenarioStats)
			})
		}}, true
	case threshold.stepPredicate != nil:
		return runtimeThresholdRegistration{Evaluate: func(payload runtimeHTTPThresholdRequest) (bool, error) {
			if err := payload.validateForScope("step"); err != nil {
				return false, err
			}
			return invokeRuntimeThresholdPredicate(func() bool {
				return threshold.stepPredicate(*payload.StepStats)
			})
		}}, true
	case threshold.metricPredicate != nil:
		return runtimeThresholdRegistration{Evaluate: func(payload runtimeHTTPThresholdRequest) (bool, error) {
			if err := payload.validateForScope("metric"); err != nil {
				return false, err
			}
			return invokeRuntimeThresholdPredicate(func() bool {
				return threshold.metricPredicate(*payload.MetricStats)
			})
		}}, true
	default:
		return runtimeThresholdRegistration{}, false
	}
}

func invokeRuntimeThresholdPredicate(predicate func() bool) (passed bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			passed = false
			err = fmt.Errorf("threshold predicate callback failed: %v", recovered)
		}
	}()
	return predicate(), nil
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
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("plugin")
	r.plugins[id] = plugin
	return id
}

func (r *runtimeCallbackRegistry) registerReportingSink(sink LoadStrikeReportingSink) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("sink")
	r.sinks[id] = sink
	return id
}

func (r *runtimeCallbackRegistry) registerRuntimePolicy(policy LoadStrikeRuntimePolicy) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("policy")
	r.policies[id] = policy
	return id
}

func (r *runtimeCallbackRegistry) registerTrackingProduce(produce func(context.Context, TrackingPayload) (EndpointProduceResult, error)) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("tracking-produce")
	r.tracking[id] = runtimeTrackingRegistration{Produce: produce}
	return id
}

func (r *runtimeCallbackRegistry) registerTrackingConsume(consume func(context.Context, func(EndpointConsumeEvent) error) error) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("tracking-consume")
	r.tracking[id] = runtimeTrackingRegistration{
		Consume:       consume,
		consumeStream: newRuntimeTrackingConsumeSession(consume),
	}
	return id
}

func (r *runtimeCallbackRegistry) registerObservationCancellation(observationContext context.Context) string {
	if observationContext == nil {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ""
	}

	id := r.nextCallbackID("observation-cancellation")
	r.observationCancellations[id] = observationContext
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

func (r *runtimeCallbackRegistry) lookupThreshold(id string) (runtimeThresholdRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.thresholds[id]
	return value, ok
}

func (r *runtimeCallbackRegistry) lookupReportingPublisher(
	id string,
) (runtimeReportingPublishRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.publishers[id]
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

func (r *runtimeCallbackRegistry) lookupObservationCancellation(id string) (context.Context, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.observationCancellations[id]
	return value, ok
}

// Close releases owned resources. Use this when the current SDK object is no longer needed.
func (r *runtimeCallbackRegistry) Close() {
	type detachedScenarioState struct {
		state     *runtimeScenarioBridgeState
		instances map[runtimeScenarioInstanceKey]*runtimeScenarioInstanceState
	}

	r.mu.Lock()
	if r.closed {
		done := r.closeDone
		r.mu.Unlock()
		if done != nil {
			<-done
		}
		return
	}
	if r.closeDone == nil {
		r.closeDone = make(chan struct{})
	}
	done := r.closeDone
	r.closed = true
	detached := make([]detachedScenarioState, 0, len(r.scenarios))
	for _, registration := range r.scenarios {
		detached = append(detached, detachedScenarioState{
			state:     registration.State,
			instances: registration.State.beginClose(),
		})
	}
	tracking := make([]runtimeTrackingRegistration, 0, len(r.tracking))
	for _, registration := range r.tracking {
		tracking = append(tracking, registration)
	}
	r.observationCancellations = nil
	r.mu.Unlock()

	for _, registration := range tracking {
		if registration.consumeStream != nil {
			registration.consumeStream.Close()
		}
	}
	for _, scenario := range detached {
		scenario.state.finishClose(scenario.instances)
	}
	close(done)
}
