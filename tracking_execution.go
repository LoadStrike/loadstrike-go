package loadstrike

import (
	stdcontext "context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	trackingRunModeGenerateAndCorrelate     = "generateandcorrelate"
	trackingRunModeCorrelateExistingTraffic = "correlateexistingtraffic"
)

type trackingOutcome struct {
	Status                 string
	TrackingID             string
	EventID                string
	GatherByValue          string
	Message                string
	SourceTimestampMS      int64
	DestinationTimestampMS int64
	LatencyMS              int64
	IsSuccess              bool
	IsFailure              bool
}

type trackingScenarioExecutor struct {
	context         contextState
	scenario        scenarioDefinition
	config          *TrackingConfigurationSpec
	runMode         string
	sourceOnly      bool
	source          endpointAdapter
	destination     endpointAdapter
	runtime         *crossPlatformTrackingRuntime
	cancel          stdcontext.CancelFunc
	waiters         map[string]chan trackingOutcome
	waitersMu       sync.Mutex
	pendingOutcomes map[string]trackingOutcome
	outcomes        chan trackingOutcome
	rowsMu          sync.Mutex
	correlation     []correlationRow
	failed          []correlationRow
	wg              sync.WaitGroup
	closeOnce       sync.Once
}

type readyAwareEndpointAdapter interface {
	WaitReady(stdcontext.Context) error
}

func newTrackingScenarioExecutor(context contextState, scenario scenarioDefinition) (*trackingScenarioExecutor, error) {
	if scenario.Tracking == nil {
		return nil, nil
	}
	if scenario.Tracking.Source == nil {
		return nil, nil
	}
	if err := validateTrackingConfiguration(scenario.Tracking); err != nil {
		return nil, err
	}

	sourceAdapter, err := newEndpointAdapter(scenario.Tracking.Source)
	if err != nil {
		return nil, err
	}

	var destinationAdapter endpointAdapter
	if scenario.Tracking.Destination != nil {
		destinationAdapter, err = newEndpointAdapter(scenario.Tracking.Destination)
		if err != nil {
			_ = sourceAdapter.Close()
			return nil, err
		}
	}

	store, err := newTrackingCorrelationStore(scenario.Tracking, context, scenario)
	if err != nil {
		_ = sourceAdapter.Close()
		if destinationAdapter != nil {
			_ = destinationAdapter.Close()
		}
		return nil, err
	}

	runMode := strings.ToLower(strings.TrimSpace(scenario.Tracking.RunMode))
	runtime := newCrossPlatformTrackingRuntime(crossPlatformTrackingRuntimeOptions{
		SourceTrackingField:             scenario.Tracking.Source.TrackingField,
		DestinationTrackingField:        destinationTrackingField(scenario.Tracking),
		DestinationGatherByField:        destinationGatherByField(scenario.Tracking),
		TrackingFieldValueCaseSensitive: scenario.Tracking.TrackingFieldValueCaseSensitive,
		GatherByFieldValueCaseSensitive: scenario.Tracking.GatherByFieldValueCaseSensitive,
		CorrelationTimeoutMS:            timeoutDuration(scenario.Tracking.CorrelationTimeoutSeconds, time.Second).Milliseconds(),
		TimeoutCountsAsFailure:          scenario.Tracking.TimeoutCountsAsFailure,
		Store:                           store,
	})

	loopCtx, cancel := stdcontext.WithCancel(stdcontext.Background())
	executor := &trackingScenarioExecutor{
		context:         context,
		scenario:        scenario,
		config:          scenario.Tracking,
		runMode:         runMode,
		sourceOnly:      scenario.Tracking.Destination == nil,
		source:          sourceAdapter,
		destination:     destinationAdapter,
		runtime:         runtime,
		cancel:          cancel,
		waiters:         map[string]chan trackingOutcome{},
		pendingOutcomes: map[string]trackingOutcome{},
		outcomes:        make(chan trackingOutcome, 128),
		correlation:     []correlationRow{},
		failed:          []correlationRow{},
	}

	if destinationAdapter != nil {
		executor.wg.Add(1)
		go func() {
			defer executor.wg.Done()
			_ = destinationAdapter.Consume(loopCtx, executor.processDestinationEvent)
		}()
		if err := waitForEndpointAdapterReady(destinationAdapter, scenario.Tracking); err != nil {
			_ = executor.Close()
			return nil, err
		}
	}

	if runMode == trackingRunModeCorrelateExistingTraffic {
		executor.wg.Add(1)
		go func() {
			defer executor.wg.Done()
			_ = sourceAdapter.Consume(loopCtx, executor.processSourceEvent)
		}()
	}

	if destinationAdapter != nil {
		executor.wg.Add(1)
		go func() {
			defer executor.wg.Done()
			executor.runTimeoutSweepLoop(loopCtx)
		}()
	}

	return executor, nil
}

func waitForEndpointAdapterReady(adapter endpointAdapter, config *TrackingConfigurationSpec) error {
	readyAdapter, ok := adapter.(readyAwareEndpointAdapter)
	if !ok {
		return nil
	}
	waitSeconds := 5.0
	if config != nil && config.CorrelationTimeoutSeconds > 0 {
		waitSeconds = maxFloat(waitSeconds, config.CorrelationTimeoutSeconds)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), time.Duration(waitSeconds*float64(time.Second)))
	defer cancel()
	return readyAdapter.WaitReady(ctx)
}

func maxFloat(left, right float64) float64 {
	if left >= right {
		return left
	}
	return right
}

func (e *trackingScenarioExecutor) ExecuteIteration(ctx stdcontext.Context) (replyResult, error) {
	if e == nil {
		return okReply(), nil
	}

	if e.sourceOnly {
		return e.executeSourceOnly(ctx)
	}

	switch e.runMode {
	case trackingRunModeGenerateAndCorrelate:
		return e.executeGenerateAndCorrelate(ctx)
	case trackingRunModeCorrelateExistingTraffic:
		select {
		case <-ctx.Done():
			return failReply("cancelled", ctx.Err().Error()), nil
		case outcome := <-e.outcomes:
			return e.replyFromOutcome(outcome), nil
		}
	default:
		return failReply("config_error", "Unsupported tracking run mode."), nil
	}
}

func (e *trackingScenarioExecutor) Close() error {
	if e == nil {
		return nil
	}

	e.closeOnce.Do(func() {
		if e.cancel != nil {
			e.cancel()
		}
		e.wg.Wait()
		if e.destination != nil {
			_ = e.destination.Close()
		}
		if e.source != nil {
			_ = e.source.Close()
		}
		if e.runtime != nil && e.runtime.store != nil {
			_ = e.runtime.store.Close()
		}
	})
	return nil
}

func (e *trackingScenarioExecutor) CorrelationRows() []correlationRow {
	e.rowsMu.Lock()
	defer e.rowsMu.Unlock()
	return append([]correlationRow(nil), e.correlation...)
}

func (e *trackingScenarioExecutor) FailedCorrelationRows() []correlationRow {
	e.rowsMu.Lock()
	defer e.rowsMu.Unlock()
	return append([]correlationRow(nil), e.failed...)
}

func (e *trackingScenarioExecutor) executeSourceOnly(ctx stdcontext.Context) (replyResult, error) {
	sourceResult, err := e.source.Produce(ctx)
	if err != nil {
		return replyResult{}, err
	}
	if !sourceResult.IsSuccess {
		outcome := trackingOutcome{
			Status:     "source_error",
			TrackingID: sourceResult.TrackingID,
			Message:    firstNonBlank(sourceResult.ErrorMessage, "Source endpoint did not produce a successful response."),
			IsSuccess:  false,
			IsFailure:  true,
		}
		e.recordOutcome(outcome)
		return e.replyFromOutcome(outcome), nil
	}

	sourcePayload := ensurePayloadTrackingID(e.config.Source, sourceResult.Payload, sourceResult.TrackingID)
	source, ok, err := e.runtime.OnSourceProduced(sourcePayload, nonZeroTimestamp(sourceResult.TimestampMS))
	if err != nil {
		return replyResult{}, err
	}
	trackingID := sourceResult.TrackingID
	if ok && source.TrackingID != "" {
		trackingID = source.TrackingID
	}

	outcome := trackingOutcome{
		Status:            "source_success",
		TrackingID:        trackingID,
		EventID:           source.EventID,
		SourceTimestampMS: source.SourceTimestampMS,
		IsSuccess:         true,
		IsFailure:         false,
	}
	e.recordOutcome(outcome)
	return e.replyFromOutcome(outcome), nil
}

func (e *trackingScenarioExecutor) executeGenerateAndCorrelate(ctx stdcontext.Context) (replyResult, error) {
	sourceResult, err := e.source.Produce(ctx)
	if err != nil {
		return replyResult{}, err
	}
	if !sourceResult.IsSuccess {
		outcome := trackingOutcome{
			Status:     "source_error",
			TrackingID: sourceResult.TrackingID,
			Message:    firstNonBlank(sourceResult.ErrorMessage, "Source endpoint did not produce a valid tracking id."),
			IsSuccess:  false,
			IsFailure:  true,
		}
		e.recordOutcome(outcome)
		return e.replyFromOutcome(outcome), nil
	}

	sourcePayload := ensurePayloadTrackingID(e.config.Source, sourceResult.Payload, sourceResult.TrackingID)
	source, ok, err := e.runtime.OnSourceProduced(sourcePayload, nonZeroTimestamp(sourceResult.TimestampMS))
	if err != nil {
		return replyResult{}, err
	}
	if !ok || strings.TrimSpace(source.EventID) == "" {
		outcome := trackingOutcome{
			Status:     "source_error",
			TrackingID: sourceResult.TrackingID,
			Message:    "Source endpoint did not produce a valid tracking id.",
			IsSuccess:  false,
			IsFailure:  true,
		}
		e.recordOutcome(outcome)
		return e.replyFromOutcome(outcome), nil
	}

	waiter := e.registerWaiter(source.EventID)
	defer e.unregisterWaiter(source.EventID)

	timeout := timeoutDuration(e.config.CorrelationTimeoutSeconds, time.Second)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return failReply("cancelled", ctx.Err().Error()), nil
	case outcome := <-waiter:
		return e.replyFromOutcome(outcome), nil
	case <-timer.C:
		expired, err := e.runtime.SweepTimeoutEntries(time.Now().UTC().UnixMilli(), maxInt(e.config.TimeoutBatchSize, 1))
		if err != nil {
			return replyResult{}, err
		}
		for _, entry := range expired {
			e.resolveOutcome(e.timeoutOutcome(entry))
		}
		select {
		case outcome := <-waiter:
			return e.replyFromOutcome(outcome), nil
		default:
			outcome := trackingOutcome{
				Status:            "timeout",
				TrackingID:        source.TrackingID,
				EventID:           source.EventID,
				Message:           fmt.Sprintf("Timeout waiting for destination event for trackingId `%s`.", source.TrackingID),
				SourceTimestampMS: source.SourceTimestampMS,
				IsSuccess:         false,
				IsFailure:         e.config.TimeoutCountsAsFailure,
			}
			e.recordOutcome(outcome)
			return e.replyFromOutcome(outcome), nil
		}
	}
}

func (e *trackingScenarioExecutor) processSourceEvent(event endpointConsumeEvent) error {
	if e.runtime == nil {
		return nil
	}
	_, _, err := e.runtime.OnSourceProduced(event.Payload, nonZeroTimestamp(event.TimestampMS))
	return err
}

func (e *trackingScenarioExecutor) processDestinationEvent(event endpointConsumeEvent) error {
	if e.runtime == nil {
		return nil
	}

	match, err := e.runtime.OnDestinationConsumed(event.Payload, nonZeroTimestamp(event.TimestampMS))
	if err != nil {
		return err
	}
	if shouldRetryUnmatchedDestination(match) {
		match, err = e.retryDestinationMatch(event, match)
		if err != nil {
			return err
		}
	}
	if match.TrackingID == "" {
		return nil
	}

	outcome := trackingOutcome{
		Status:                 "unmatched_destination",
		TrackingID:             match.TrackingID,
		EventID:                match.EventID,
		GatherByValue:          normalizeGatherValue(match.GatherKey),
		Message:                firstNonBlank(match.Message, fmt.Sprintf("Destination event `%s` had no matching source event.", match.TrackingID)),
		SourceTimestampMS:      match.SourceTimestampUTCMS,
		DestinationTimestampMS: match.DestinationTimestampMS,
		LatencyMS:              match.LatencyMS,
		IsSuccess:              match.Matched,
		IsFailure:              !match.Matched,
	}
	if match.Matched {
		outcome.Status = "matched"
		outcome.Message = ""
		outcome.IsFailure = false
	}

	e.resolveOutcome(outcome)
	return nil
}

func (e *trackingScenarioExecutor) retryDestinationMatch(event endpointConsumeEvent, initial destinationConsumeResult) (destinationConsumeResult, error) {
	match := initial
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		retried, err := e.runtime.RetryDestinationMatch(event.Payload, nonZeroTimestamp(event.TimestampMS))
		if err != nil {
			return match, err
		}
		if retried.Matched {
			return retried, nil
		}
		if retried.TrackingID == "" {
			return retried, nil
		}
		match = retried
	}
	return match, nil
}

func shouldRetryUnmatchedDestination(match destinationConsumeResult) bool {
	return strings.TrimSpace(match.TrackingID) != "" && !match.Matched && match.Message == "Destination tracking id had no source match."
}

func (e *trackingScenarioExecutor) runTimeoutSweepLoop(ctx stdcontext.Context) {
	ticker := time.NewTicker(timeoutDuration(e.config.TimeoutSweepIntervalSeconds, time.Second))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := e.runtime.SweepTimeoutEntries(time.Now().UTC().UnixMilli(), maxInt(e.config.TimeoutBatchSize, 1))
			if err != nil {
				continue
			}
			for _, entry := range expired {
				e.resolveOutcome(e.timeoutOutcome(entry))
			}
		}
	}
}

func (e *trackingScenarioExecutor) timeoutOutcome(entry correlationEntry) trackingOutcome {
	trackingID := entry.TrackingID
	if e.runtime != nil {
		trackingID = e.runtime.resolveTrackingIDDisplay(entry.EventID, entry.TrackingID, false)
	}
	return trackingOutcome{
		Status:            "timeout",
		TrackingID:        trackingID,
		EventID:           entry.EventID,
		Message:           fmt.Sprintf("TrackingId `%s` timed out.", trackingID),
		SourceTimestampMS: entry.SourceTimestampUTCMS,
		IsSuccess:         false,
		IsFailure:         e.config.TimeoutCountsAsFailure,
	}
}

func (e *trackingScenarioExecutor) resolveOutcome(outcome trackingOutcome) {
	e.recordOutcome(outcome)

	if strings.TrimSpace(outcome.EventID) != "" {
		e.waitersMu.Lock()
		waiter := e.waiters[outcome.EventID]
		if waiter == nil && e.runMode != trackingRunModeCorrelateExistingTraffic {
			if e.pendingOutcomes == nil {
				e.pendingOutcomes = map[string]trackingOutcome{}
			}
			e.pendingOutcomes[outcome.EventID] = outcome
		}
		e.waitersMu.Unlock()
		if waiter != nil {
			select {
			case waiter <- outcome:
			default:
			}
			return
		}
	}

	if e.runMode == trackingRunModeCorrelateExistingTraffic {
		select {
		case e.outcomes <- outcome:
		default:
		}
	}
}

func (e *trackingScenarioExecutor) recordOutcome(outcome trackingOutcome) {
	row := correlationRow{
		OccurredUTC:             time.Now().UTC().Format(time.RFC3339Nano),
		ScenarioName:            e.scenario.Name,
		Source:                  e.config.Source.Name,
		Destination:             destinationName(e.config),
		RunMode:                 normalizedRunModeName(e.config.RunMode),
		Status:                  outcome.Status,
		TrackingID:              outcome.TrackingID,
		EventID:                 outcome.EventID,
		SourceTimestampUTC:      formatTimestampMS(outcome.SourceTimestampMS),
		DestinationTimestampUTC: formatTimestampMS(outcome.DestinationTimestampMS),
		LatencyMS:               formatLatencyMS(outcome.LatencyMS),
		Message:                 outcome.Message,
		IsSuccess:               outcome.IsSuccess,
		IsFailure:               outcome.IsFailure,
		GatherByField:           destinationGatherByField(e.config),
		GatherByValue:           outcome.GatherByValue,
	}

	e.rowsMu.Lock()
	defer e.rowsMu.Unlock()

	e.correlation = append(e.correlation, row)
	if outcome.IsFailure {
		failed := row
		e.failed = append(e.failed, failed)
	}
}

func (e *trackingScenarioExecutor) registerWaiter(eventID string) chan trackingOutcome {
	waiter := make(chan trackingOutcome, 1)
	e.waitersMu.Lock()
	if e.waiters == nil {
		e.waiters = map[string]chan trackingOutcome{}
	}
	e.waiters[eventID] = waiter
	if outcome, exists := e.pendingOutcomes[eventID]; exists {
		delete(e.pendingOutcomes, eventID)
		waiter <- outcome
	}
	e.waitersMu.Unlock()
	return waiter
}

func (e *trackingScenarioExecutor) unregisterWaiter(eventID string) {
	e.waitersMu.Lock()
	delete(e.waiters, eventID)
	delete(e.pendingOutcomes, eventID)
	e.waitersMu.Unlock()
}

func (e *trackingScenarioExecutor) replyFromOutcome(outcome trackingOutcome) replyResult {
	switch outcome.Status {
	case "matched":
		return normalizeReplyValue(replyResult{
			IsSuccess:       true,
			StatusCode:      "matched",
			CustomLatencyMS: float64(outcome.LatencyMS),
		})
	case "source_success":
		return normalizeReplyValue(replyResult{
			IsSuccess:  true,
			StatusCode: "source_success",
		})
	default:
		return normalizeReplyValue(replyResult{
			IsSuccess:  false,
			StatusCode: outcome.Status,
			Message:    outcome.Message,
		})
	}
}

func validateTrackingConfiguration(config *TrackingConfigurationSpec) error {
	if config == nil {
		return fmt.Errorf("tracking configuration is required")
	}
	if config.Source == nil {
		return fmt.Errorf("tracking source endpoint is required")
	}
	if config.CorrelationStore == nil {
		return fmt.Errorf("correlation store is required")
	}
	if strings.TrimSpace(config.MetricPrefix) == "" {
		return fmt.Errorf("MetricPrefix must be provided")
	}
	if err := validateEndpointSpec(config.Source); err != nil {
		return err
	}
	if config.Destination != nil {
		if err := validateEndpointSpec(config.Destination); err != nil {
			return err
		}
	}
	if err := validateCorrelationStoreSpec(config.CorrelationStore); err != nil {
		return err
	}
	if strings.TrimSpace(config.Source.GatherByField) != "" {
		return fmt.Errorf("GatherByField is supported on destination endpoint only")
	}
	if config.CorrelationTimeoutSeconds <= 0 {
		return fmt.Errorf("CorrelationTimeoutSeconds must be greater than zero")
	}
	if config.TimeoutSweepIntervalSeconds <= 0 {
		return fmt.Errorf("TimeoutSweepIntervalSeconds must be greater than zero")
	}
	if config.TimeoutBatchSize <= 0 {
		return fmt.Errorf("TimeoutBatchSize must be greater than zero")
	}
	runMode := strings.ToLower(strings.TrimSpace(config.RunMode))
	switch runMode {
	case trackingRunModeGenerateAndCorrelate:
		if !strings.EqualFold(config.Source.Mode, "Produce") {
			return fmt.Errorf("Source endpoint mode must be Produce for GenerateAndCorrelate mode")
		}
		if config.Destination != nil && !strings.EqualFold(config.Destination.Mode, "Consume") {
			return fmt.Errorf("Destination endpoint mode must be Consume for GenerateAndCorrelate mode")
		}
	case trackingRunModeCorrelateExistingTraffic:
		if config.Destination == nil {
			return fmt.Errorf("Destination endpoint must be provided for CorrelateExistingTraffic mode")
		}
		if !strings.EqualFold(config.Source.Mode, "Consume") {
			return fmt.Errorf("Source endpoint mode must be Consume for CorrelateExistingTraffic mode")
		}
		if !strings.EqualFold(config.Destination.Mode, "Consume") {
			return fmt.Errorf("Destination endpoint mode must be Consume for CorrelateExistingTraffic mode")
		}
	default:
		return fmt.Errorf("Unsupported tracking run mode %q", config.RunMode)
	}
	return nil
}

func newTrackingCorrelationStore(config *TrackingConfigurationSpec, context contextState, scenario scenarioDefinition) (correlationStore, error) {
	if config == nil || config.CorrelationStore == nil {
		return nil, fmt.Errorf("correlation store is required")
	}
	if strings.TrimSpace(config.CorrelationStore.Kind) == "" || strings.EqualFold(config.CorrelationStore.Kind, "InMemory") {
		return newInMemoryCorrelationStore(), nil
	}
	if !strings.EqualFold(config.CorrelationStore.Kind, "Redis") {
		return nil, fmt.Errorf("unsupported correlation store kind %q", config.CorrelationStore.Kind)
	}
	if config.CorrelationStore.Redis == nil {
		return nil, fmt.Errorf("redis correlation store requires Redis options")
	}

	runNamespace := sanitizeClusterToken(firstNonBlank(context.SessionID, generateSessionID()) + "-" + scenario.Name)
	return newRedisCorrelationStore(*config.CorrelationStore.Redis, runNamespace)
}

func validateCorrelationStoreSpec(store *CorrelationStoreSpec) error {
	if store == nil {
		return fmt.Errorf("correlation store is required")
	}
	switch strings.ToLower(strings.TrimSpace(store.Kind)) {
	case "", "inmemory":
		return nil
	case "redis":
		if store.Redis == nil {
			return fmt.Errorf("redis correlation store requires Redis options")
		}
		if strings.TrimSpace(store.Redis.ConnectionString) == "" {
			return fmt.Errorf("Redis ConnectionString must be provided")
		}
		if strings.TrimSpace(store.Redis.KeyPrefix) == "" {
			return fmt.Errorf("Redis KeyPrefix must be provided")
		}
		if store.Redis.EntryTTLSeconds <= 0 {
			return fmt.Errorf("Redis EntryTtlSeconds must be greater than zero")
		}
		return nil
	default:
		return fmt.Errorf("unsupported correlation store kind %q", store.Kind)
	}
}

func validateEndpointSpec(endpoint *EndpointSpec) error {
	if endpoint == nil {
		return fmt.Errorf("endpoint definition is required")
	}
	if strings.TrimSpace(endpoint.Name) == "" {
		return fmt.Errorf("Endpoint name must be provided")
	}
	if strings.TrimSpace(endpoint.TrackingField) == "" {
		return fmt.Errorf("TrackingField must be provided")
	}
	if _, err := parseTrackingFieldSelector(endpoint.TrackingField); err != nil {
		return err
	}
	if strings.TrimSpace(endpoint.GatherByField) != "" && strings.TrimSpace(strings.TrimPrefix(endpoint.GatherByField, "header:")) == "" && strings.TrimSpace(strings.TrimPrefix(endpoint.GatherByField, "json:")) == "" {
		return fmt.Errorf("GatherByField path must be provided when GatherByField is configured")
	}
	if strings.TrimSpace(endpoint.GatherByField) != "" {
		if _, err := parseTrackingFieldSelector(endpoint.GatherByField); err != nil {
			return err
		}
	}
	if endpoint.PollIntervalSeconds < 0 {
		return fmt.Errorf("PollInterval must be greater than zero")
	}

	switch strings.ToLower(strings.TrimSpace(endpoint.Kind)) {
	case "http":
		if endpoint.HTTP == nil {
			return fmt.Errorf("http options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.HTTP.URL) == "" {
			return fmt.Errorf("Url must be provided for HTTP endpoint")
		}
		if _, err := url.ParseRequestURI(endpoint.HTTP.URL); err != nil {
			return fmt.Errorf("Url must be an absolute URI for HTTP endpoint")
		}
		if endpoint.HTTP.RequestTimeoutSeconds < 0 {
			return fmt.Errorf("RequestTimeoutSeconds must be greater than zero")
		}
		if err := validateHTTPAuthOptions(endpoint.HTTP.Auth); err != nil {
			return err
		}
	case "kafka":
		if endpoint.Kafka == nil {
			return fmt.Errorf("kafka options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.Kafka.BootstrapServers) == "" {
			return fmt.Errorf("BootstrapServers must be provided for Kafka endpoint")
		}
		if strings.TrimSpace(endpoint.Kafka.Topic) == "" {
			return fmt.Errorf("Topic must be provided for Kafka endpoint")
		}
		if strings.EqualFold(endpoint.Mode, "Consume") && strings.TrimSpace(endpoint.Kafka.ConsumerGroupID) == "" {
			return fmt.Errorf("ConsumerGroupId must be provided when Kafka endpoint mode is Consume")
		}
		if protocol := strings.ToLower(strings.TrimSpace(endpoint.Kafka.SecurityProtocol)); protocol == "sasl_plaintext" || protocol == "saslplaintext" || protocol == "sasl_ssl" || protocol == "saslssl" {
			if endpoint.Kafka.SASL == nil {
				return fmt.Errorf("Sasl options must be provided for SASL Kafka security protocol")
			}
			if strings.TrimSpace(endpoint.Kafka.SASL.Mechanism) == "gssapi" && endpoint.Kafka.SASL.GSSAPI == nil {
				return fmt.Errorf("Kafka GSSAPI options must be provided when using the GSSAPI mechanism")
			}
		}
	case "nats":
		if endpoint.NATS == nil {
			return fmt.Errorf("nats options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.NATS.ServerURL) == "" {
			return fmt.Errorf("ServerUrl must be provided for NATS endpoint")
		}
		if strings.TrimSpace(endpoint.NATS.Subject) == "" {
			return fmt.Errorf("Subject must be provided for NATS endpoint")
		}
		if strings.TrimSpace(endpoint.NATS.UserName) != "" && endpoint.NATS.Password == "" {
			return fmt.Errorf("Password must be provided when UserName is configured for NATS endpoint")
		}
		if endpoint.NATS.MaxReconnectAttempts < 0 {
			return fmt.Errorf("MaxReconnectAttempts must be zero or greater")
		}
	case "rabbitmq":
		if endpoint.RabbitMQ == nil {
			return fmt.Errorf("rabbitmq options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.RabbitMQ.HostName) == "" {
			return fmt.Errorf("HostName must be provided for RabbitMQ endpoint")
		}
		if endpoint.RabbitMQ.Port <= 0 {
			return fmt.Errorf("Port must be greater than zero")
		}
		if strings.TrimSpace(endpoint.RabbitMQ.UserName) == "" {
			return fmt.Errorf("UserName must be provided for RabbitMQ endpoint")
		}
		if strings.EqualFold(endpoint.Mode, "Produce") && strings.TrimSpace(endpoint.RabbitMQ.RoutingKey) == "" && strings.TrimSpace(endpoint.RabbitMQ.QueueName) == "" {
			return fmt.Errorf("Either RoutingKey or QueueName must be provided for RabbitMQ producer endpoint")
		}
		if strings.EqualFold(endpoint.Mode, "Consume") && strings.TrimSpace(endpoint.RabbitMQ.QueueName) == "" {
			return fmt.Errorf("QueueName must be provided for RabbitMQ consumer endpoint")
		}
	case "redisstreams":
		if endpoint.RedisStreams == nil {
			return fmt.Errorf("redis streams options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.RedisStreams.ConnectionString) == "" {
			return fmt.Errorf("ConnectionString must be provided for Redis Streams endpoint")
		}
		if strings.TrimSpace(endpoint.RedisStreams.StreamKey) == "" {
			return fmt.Errorf("StreamKey must be provided for Redis Streams endpoint")
		}
		if strings.EqualFold(endpoint.Mode, "Consume") && strings.TrimSpace(endpoint.RedisStreams.ConsumerGroup) == "" {
			return fmt.Errorf("ConsumerGroup must be provided when Redis Streams endpoint mode is Consume")
		}
		if strings.EqualFold(endpoint.Mode, "Consume") && strings.TrimSpace(endpoint.RedisStreams.ConsumerName) == "" {
			return fmt.Errorf("ConsumerName must be provided when Redis Streams endpoint mode is Consume")
		}
		if endpoint.RedisStreams.ReadCount <= 0 {
			return fmt.Errorf("ReadCount must be greater than zero")
		}
		if endpoint.RedisStreams.MaxLength < 0 {
			return fmt.Errorf("MaxLength must be greater than zero when configured")
		}
	case "azureeventhubs":
		if endpoint.AzureEventHubs == nil {
			return fmt.Errorf("azure event hubs options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.AzureEventHubs.ConnectionString) == "" {
			return fmt.Errorf("ConnectionString must be provided for Azure Event Hubs endpoint")
		}
		if strings.TrimSpace(endpoint.AzureEventHubs.EventHubName) == "" {
			return fmt.Errorf("EventHubName must be provided for Azure Event Hubs endpoint")
		}
		if endpoint.AzureEventHubs.PartitionCount < 0 {
			return fmt.Errorf("PartitionCount must be zero or greater when configured")
		}
	case "delegatestream":
		if endpoint.DelegateStream == nil {
			return fmt.Errorf("delegate stream options are required for endpoint %q", endpoint.Name)
		}
		if strings.EqualFold(endpoint.Mode, "Produce") && endpoint.DelegateStream.Produce == nil && strings.TrimSpace(endpoint.DelegateStream.ProduceCallbackURL) == "" {
			return fmt.Errorf("ProduceAsync delegate must be provided when endpoint mode is Produce")
		}
		if strings.EqualFold(endpoint.Mode, "Consume") && endpoint.DelegateStream.Consume == nil && strings.TrimSpace(endpoint.DelegateStream.ConsumeCallbackURL) == "" {
			return fmt.Errorf("ConsumeAsync delegate must be provided when endpoint mode is Consume")
		}
	case "pushdiffusion":
		if endpoint.PushDiffusion == nil {
			return fmt.Errorf("push diffusion options are required for endpoint %q", endpoint.Name)
		}
		if strings.TrimSpace(endpoint.PushDiffusion.ServerURL) == "" {
			return fmt.Errorf("ServerUrl must be provided for Push Diffusion endpoint")
		}
		if strings.TrimSpace(endpoint.PushDiffusion.TopicPath) == "" {
			return fmt.Errorf("TopicPath must be provided for Push Diffusion endpoint")
		}
		if strings.EqualFold(endpoint.Mode, "Produce") && endpoint.PushDiffusion.Publish == nil && strings.TrimSpace(endpoint.PushDiffusion.PublishCallbackURL) == "" {
			return fmt.Errorf("PublishAsync delegate must be provided for Push Diffusion producer endpoint")
		}
		if strings.EqualFold(endpoint.Mode, "Consume") && endpoint.PushDiffusion.Subscribe == nil && strings.TrimSpace(endpoint.PushDiffusion.SubscribeCallbackURL) == "" {
			return fmt.Errorf("SubscribeAsync delegate must be provided for Push Diffusion consumer endpoint")
		}
	}
	return nil
}

func validateHTTPAuthOptions(auth *HTTPAuthOptions) error {
	if auth == nil {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(auth.Type)) {
	case "", "none":
		return nil
	case "basic":
		if strings.TrimSpace(auth.Username) == "" {
			return fmt.Errorf("Username must be provided for basic auth")
		}
		return nil
	case "bearer":
		if strings.TrimSpace(auth.BearerToken) == "" {
			return fmt.Errorf("BearerToken must be provided for bearer auth")
		}
		return nil
	case "oauth2clientcredentials":
		tokenEndpoint := strings.TrimSpace(auth.TokenURL)
		clientID := strings.TrimSpace(auth.ClientID)
		clientSecret := strings.TrimSpace(auth.ClientSecret)
		if auth.OAuth2ClientCredentials != nil {
			tokenEndpoint = firstNonBlank(tokenEndpoint, auth.OAuth2ClientCredentials.TokenEndpoint)
			clientID = firstNonBlank(clientID, auth.OAuth2ClientCredentials.ClientID)
			clientSecret = firstNonBlank(clientSecret, auth.OAuth2ClientCredentials.ClientSecret)
		}
		if strings.TrimSpace(tokenEndpoint) == "" {
			return fmt.Errorf("TokenEndpoint must be provided for OAuth2 client credentials flow")
		}
		if strings.TrimSpace(clientID) == "" {
			return fmt.Errorf("ClientId must be provided for OAuth2 client credentials flow")
		}
		if strings.TrimSpace(clientSecret) == "" {
			return fmt.Errorf("ClientSecret must be provided for OAuth2 client credentials flow")
		}
		return nil
	default:
		return fmt.Errorf("unsupported HTTP auth type %q", auth.Type)
	}
}

func normalizedRunModeName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case trackingRunModeGenerateAndCorrelate:
		return "GenerateAndCorrelate"
	case trackingRunModeCorrelateExistingTraffic:
		return "CorrelateExistingTraffic"
	default:
		return strings.TrimSpace(value)
	}
}

func destinationName(config *TrackingConfigurationSpec) string {
	if config == nil || config.Destination == nil {
		return "<source-only>"
	}
	return config.Destination.Name
}

func destinationGatherByField(config *TrackingConfigurationSpec) string {
	if config == nil || config.Destination == nil {
		return ""
	}
	return strings.TrimSpace(config.Destination.GatherByField)
}

func destinationTrackingField(config *TrackingConfigurationSpec) string {
	if config == nil || config.Destination == nil || strings.TrimSpace(config.Destination.TrackingField) == "" {
		if config != nil && config.Source != nil {
			return config.Source.TrackingField
		}
		return ""
	}
	return config.Destination.TrackingField
}

func normalizeGatherValue(value string) string {
	switch strings.TrimSpace(value) {
	case "", "__ungrouped__":
		return ""
	case "__unknown__":
		return ""
	default:
		return value
	}
}

func formatTimestampMS(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.UnixMilli(value).UTC().Format(time.RFC3339Nano)
}

func formatLatencyMS(value int64) string {
	if value <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}

func timeoutDuration(value float64, unit time.Duration) time.Duration {
	if value <= 0 {
		if unit == time.Second {
			value = 30
		} else {
			value = 1
		}
	}
	return time.Duration(value * float64(unit))
}

func nonZeroTimestamp(value int64) int64 {
	if value > 0 {
		return value
	}
	return time.Now().UTC().UnixMilli()
}

func ensurePayloadTrackingID(endpoint *EndpointSpec, payload trackingPayload, trackingID string) trackingPayload {
	if endpoint == nil || strings.TrimSpace(trackingID) == "" {
		return payload
	}
	selector, err := parseTrackingFieldSelector(endpoint.TrackingField)
	if err != nil {
		return payload
	}
	if selector.Extract(payload) != "" {
		return payload
	}
	selector.Set(&payload, trackingID)
	return payload
}

func executeTrackingScenarioIteration(
	context contextState,
	scenario scenarioDefinition,
	scenarioInstanceData map[string]any,
	invocationNumber int,
	runTrace *reportTrace,
	trackingExecutor *trackingScenarioExecutor,
) (int, int, int, []stepStats, []RuntimePolicyError, bool, error) {
	attemptsLeft := 0
	if scenario.RestartIterationOnFail {
		attemptsLeft = context.RestartIterationMaxAttempts
	}

	for {
		requestCount := 0
		okCount := 0
		failCount := 0
		shouldRestart := false
		stopScenario := false
		attemptTrace := newReportTrace()
		stepStatsRows := []stepStats{}
		policyErrors := []RuntimePolicyError{}
		stepData := map[string]any{}

		if scenario.Tracking != nil && scenario.Tracking.ExecuteOriginalScenarioRun && len(scenario.Steps) > 0 {
			for _, step := range scenario.Steps {
				if step.Run == nil {
					continue
				}

				stepContext := &stepRuntimeContext{
					ScenarioName:         scenario.Name,
					StepName:             step.Name,
					TestSuite:            context.TestSuite,
					TestName:             context.TestName,
					InvocationNumber:     int64(invocationNumber),
					Data:                 stepData,
					ScenarioInstanceData: scenarioInstanceData,
				}

				if err := runBeforeStepPolicies(context, stepContext, &policyErrors); err != nil && context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
					return 0, 0, 0, nil, nil, false, err
				}

				reply := step.Run(stepContext)
				requestCount++
				attemptTrace.addStep(step.Name)

				stepStat := stepStats{
					ScenarioName:    scenario.Name,
					StepName:        step.Name,
					AllRequestCount: 1,
					SortIndex:       len(stepStatsRows),
				}
				if reply.IsSuccess {
					okCount++
					stepStat.AllOKCount = 1
				} else {
					failCount++
					stepStat.AllFailCount = 1
					attemptTrace.addFailure(scenario.Name, step.Name, reply)
					shouldRestart = scenario.RestartIterationOnFail && attemptsLeft > 0
				}
				stepStatsRows = append(stepStatsRows, stepStat)

				if err := runAfterStepPolicies(context, stepContext, reply, &policyErrors); err != nil && context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
					return 0, 0, 0, nil, nil, false, err
				}
				if stepContext.stopRequested {
					stopScenario = true
				}
				if !reply.IsSuccess || stopScenario {
					break
				}
			}
		}

		if failCount == 0 && !stopScenario {
			stepName := firstNonBlank(scenario.Tracking.Source.Name, scenario.Name)
			stepContext := &stepRuntimeContext{
				ScenarioName:         scenario.Name,
				StepName:             stepName,
				TestSuite:            context.TestSuite,
				TestName:             context.TestName,
				InvocationNumber:     int64(invocationNumber),
				Data:                 stepData,
				ScenarioInstanceData: scenarioInstanceData,
			}

			if err := runBeforeStepPolicies(context, stepContext, &policyErrors); err != nil && context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
				return 0, 0, 0, nil, nil, false, err
			}

			reply, err := trackingExecutor.ExecuteIteration(stdcontext.Background())
			if err != nil {
				return 0, 0, 0, nil, nil, false, err
			}

			requestCount++
			attemptTrace.addStep(stepName)
			stepStat := stepStats{
				ScenarioName:    scenario.Name,
				StepName:        stepName,
				AllRequestCount: 1,
				SortIndex:       len(stepStatsRows),
			}
			if reply.IsSuccess {
				okCount++
				stepStat.AllOKCount = 1
			} else {
				failCount++
				stepStat.AllFailCount = 1
				attemptTrace.addFailure(scenario.Name, stepName, reply)
				shouldRestart = scenario.RestartIterationOnFail && attemptsLeft > 0
			}
			stepStatsRows = append(stepStatsRows, stepStat)

			if err := runAfterStepPolicies(context, stepContext, reply, &policyErrors); err != nil && context.RuntimePolicyErrorMode == RuntimePolicyErrorModeFail {
				return 0, 0, 0, nil, nil, false, err
			}
			if stepContext.stopRequested {
				stopScenario = true
			}
		}

		if !shouldRestart {
			runTrace.merge(attemptTrace)
			return requestCount, okCount, failCount, stepStatsRows, policyErrors, stopScenario, nil
		}

		attemptsLeft--
	}
}
