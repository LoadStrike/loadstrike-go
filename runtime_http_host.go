package loadstrike

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type runtimeHTTPHostHandle struct {
	baseURL   string
	listener  net.Listener
	server    *http.Server
	registry  *runtimeCallbackRegistry
	closeOnce sync.Once
}

type runtimeHTTPScenarioHookRequest struct {
	RunIdentity          string                        `json:"runIdentity"`
	NodeIdentity         string                        `json:"nodeIdentity"`
	PartitionNumber      *int                          `json:"partitionNumber"`
	ScenarioName         string                        `json:"scenarioName"`
	ScenarioInstanceID   string                        `json:"scenarioInstanceId"`
	TestSuite            string                        `json:"testSuite,omitempty"`
	TestName             string                        `json:"testName,omitempty"`
	InvocationNumber     int64                         `json:"invocationNumber,omitempty"`
	ScenarioStartedUTC   string                        `json:"scenarioStartedUtc,omitempty"`
	AgentIndex           int                           `json:"agentIndex,omitempty"`
	AgentCount           int                           `json:"agentCount,omitempty"`
	TestInfo             *runtimeHTTPTestInfo          `json:"testInfo,omitempty"`
	NodeInfo             *runtimeHTTPNodeInfo          `json:"nodeInfo,omitempty"`
	ScenarioInfo         *runtimeHTTPScenarioInfo      `json:"scenarioInfo,omitempty"`
	Partition            *runtimeHTTPScenarioPartition `json:"partition,omitempty"`
	CustomSettings       map[string]any                `json:"customSettings,omitempty"`
	GlobalCustomSettings map[string]any                `json:"globalCustomSettings,omitempty"`
}

type runtimeHTTPTestInfo struct {
	TestSuite  string `json:"testSuite"`
	TestName   string `json:"testName"`
	SessionID  string `json:"sessionId"`
	ClusterID  string `json:"clusterId"`
	CreatedUTC string `json:"createdUtc"`
}

type runtimeHTTPNodeInfo struct {
	NodeType             NodeType                `json:"nodeType"`
	MachineName          string                  `json:"machineName"`
	CurrentOperation     string                  `json:"currentOperation"`
	CurrentOperationType LoadStrikeOperationType `json:"currentOperationType"`
	CoresCount           int                     `json:"coresCount"`
	DotNetVersion        string                  `json:"dotNetVersion"`
	EngineVersion        string                  `json:"engineVersion"`
	OS                   string                  `json:"os"`
	Processor            string                  `json:"processor"`
}

type runtimeHTTPScenarioInfo struct {
	InstanceID                  string                      `json:"instanceId"`
	InstanceNumber              int                         `json:"instanceNumber"`
	ScenarioDurationNanoseconds int64                       `json:"scenarioDurationNanoseconds"`
	ScenarioName                string                      `json:"scenarioName"`
	ScenarioOperation           LoadStrikeScenarioOperation `json:"scenarioOperation"`
}

type runtimeHTTPScenarioPartition struct {
	Number int `json:"number"`
	Count  int `json:"count"`
}

type runtimeScenarioCallbackMetadata struct {
	TestInfo             testInfo
	NodeInfo             nodeInfo
	ScenarioInfo         LoadStrikeScenarioInfo
	Partition            scenarioPartitionInfo
	CustomSettings       map[string]any
	GlobalCustomSettings map[string]any
}

func (r runtimeHTTPScenarioHookRequest) scenarioInstanceKey() (runtimeScenarioInstanceKey, error) {
	fields := []struct {
		name  string
		value string
	}{
		{name: "runIdentity", value: r.RunIdentity},
		{name: "nodeIdentity", value: r.NodeIdentity},
		{name: "scenarioName", value: r.ScenarioName},
		{name: "scenarioInstanceId", value: r.ScenarioInstanceID},
	}
	for _, field := range fields {
		if !utf8.ValidString(field.value) || strings.TrimSpace(field.value) == "" || len([]byte(field.value)) > 512 {
			return runtimeScenarioInstanceKey{}, fmt.Errorf(
				"%w: %s must contain between 1 and 512 UTF-8 bytes",
				errRuntimeScenarioInstanceIdentityInvalid,
				field.name,
			)
		}
	}
	if r.PartitionNumber == nil || *r.PartitionNumber < 0 {
		return runtimeScenarioInstanceKey{}, fmt.Errorf(
			"%w: partitionNumber must be a nonnegative integer",
			errRuntimeScenarioInstanceIdentityInvalid,
		)
	}

	return runtimeScenarioInstanceKey{
		RunIdentity:        r.RunIdentity,
		NodeIdentity:       r.NodeIdentity,
		PartitionNumber:    *r.PartitionNumber,
		ScenarioName:       r.ScenarioName,
		ScenarioInstanceID: r.ScenarioInstanceID,
	}, nil
}

func (r runtimeHTTPScenarioHookRequest) scenarioInstanceKeyForRegistration(
	registeredScenarioName string,
) (runtimeScenarioInstanceKey, error) {
	key, err := r.scenarioInstanceKey()
	if err != nil {
		return runtimeScenarioInstanceKey{}, err
	}
	if r.ScenarioName != registeredScenarioName {
		return runtimeScenarioInstanceKey{}, fmt.Errorf(
			"%w: scenarioName does not match the registered callback",
			errRuntimeScenarioInstanceIdentityInvalid,
		)
	}
	if r.ScenarioInfo != nil {
		if r.ScenarioInfo.ScenarioName != r.ScenarioName {
			return runtimeScenarioInstanceKey{}, fmt.Errorf(
				"%w: scenarioInfo.scenarioName must match scenarioName",
				errRuntimeScenarioInstanceIdentityInvalid,
			)
		}
		if r.ScenarioInfo.InstanceID != r.ScenarioInstanceID {
			return runtimeScenarioInstanceKey{}, fmt.Errorf(
				"%w: scenarioInfo.instanceId must match scenarioInstanceId",
				errRuntimeScenarioInstanceIdentityInvalid,
			)
		}
		if r.ScenarioInfo.InstanceNumber < 0 || r.ScenarioInfo.ScenarioDurationNanoseconds < 0 {
			return runtimeScenarioInstanceKey{}, fmt.Errorf(
				"%w: scenarioInfo instance number and duration must be nonnegative",
				errRuntimeScenarioInstanceIdentityInvalid,
			)
		}
	}
	if r.Partition != nil {
		if r.Partition.Number != key.PartitionNumber {
			return runtimeScenarioInstanceKey{}, fmt.Errorf(
				"%w: partition.number must match partitionNumber",
				errRuntimeScenarioInstanceIdentityInvalid,
			)
		}
		if r.Partition.Count < 1 {
			return runtimeScenarioInstanceKey{}, fmt.Errorf(
				"%w: partition.count must be a positive integer",
				errRuntimeScenarioInstanceIdentityInvalid,
			)
		}
	}
	return key, nil
}

func (r runtimeHTTPScenarioHookRequest) callbackMetadata() (runtimeScenarioCallbackMetadata, error) {
	if r.TestInfo == nil || r.NodeInfo == nil || r.ScenarioInfo == nil || r.Partition == nil {
		return runtimeScenarioCallbackMetadata{}, fmt.Errorf(
			"%w: testInfo, nodeInfo, scenarioInfo, and partition are required",
			errRuntimeScenarioInstanceIdentityInvalid,
		)
	}
	customSettings, err := cloneJSONCompatibleSettings(r.CustomSettings)
	if err != nil {
		return runtimeScenarioCallbackMetadata{}, fmt.Errorf(
			"%w: customSettings: %v",
			errRuntimeScenarioInstanceIdentityInvalid,
			err,
		)
	}
	globalCustomSettings, err := cloneJSONCompatibleSettings(r.GlobalCustomSettings)
	if err != nil {
		return runtimeScenarioCallbackMetadata{}, fmt.Errorf(
			"%w: globalCustomSettings: %v",
			errRuntimeScenarioInstanceIdentityInvalid,
			err,
		)
	}
	metadata := runtimeScenarioCallbackMetadata{
		CustomSettings:       customSettings,
		GlobalCustomSettings: globalCustomSettings,
	}

	created, err := time.Parse(time.RFC3339Nano, r.TestInfo.CreatedUTC)
	if err != nil {
		return runtimeScenarioCallbackMetadata{}, fmt.Errorf(
			"%w: testInfo.createdUtc must be an RFC3339Nano timestamp",
			errRuntimeScenarioInstanceIdentityInvalid,
		)
	}
	metadata.TestInfo = normalizeTestInfo(testInfo{
		TestSuite:  r.TestInfo.TestSuite,
		TestName:   r.TestInfo.TestName,
		SessionID:  r.TestInfo.SessionID,
		ClusterID:  r.TestInfo.ClusterID,
		CreatedUTC: created.UTC(),
	})
	metadata.NodeInfo = nodeInfo{
		NodeType:             r.NodeInfo.NodeType,
		MachineName:          r.NodeInfo.MachineName,
		CurrentOperation:     r.NodeInfo.CurrentOperation,
		CurrentOperationType: r.NodeInfo.CurrentOperationType,
		CoresCount:           r.NodeInfo.CoresCount,
		DotNetVersion:        r.NodeInfo.DotNetVersion,
		EngineVersion:        r.NodeInfo.EngineVersion,
		OS:                   r.NodeInfo.OS,
		Processor:            r.NodeInfo.Processor,
	}
	metadata.ScenarioInfo = normalizeScenarioInfo(LoadStrikeScenarioInfo{
		InstanceID:        r.ScenarioInfo.InstanceID,
		InstanceNumber:    r.ScenarioInfo.InstanceNumber,
		ScenarioDuration:  time.Duration(r.ScenarioInfo.ScenarioDurationNanoseconds),
		ScenarioName:      r.ScenarioInfo.ScenarioName,
		ScenarioOperation: r.ScenarioInfo.ScenarioOperation,
	})
	metadata.Partition = scenarioPartitionInfo{
		Number: r.Partition.Number,
		Count:  r.Partition.Count,
	}
	return metadata, nil
}

func writeRuntimeScenarioCallbackError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errRuntimeScenarioInstanceIdentityInvalid):
		http.Error(writer, err.Error(), http.StatusBadRequest)
	case errors.Is(err, errRuntimeScenarioInstanceGone):
		http.Error(writer, err.Error(), http.StatusGone)
	default:
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

type runtimeHTTPScenarioHookResponse struct {
	Metrics []runtimeScenarioMetricDescriptor `json:"metrics,omitempty"`
	Logs    []runtimeCallbackLog              `json:"logs,omitempty"`
}

type runtimeHTTPStepResponse struct {
	IsSuccess             bool                 `json:"isSuccess"`
	StatusCode            string               `json:"statusCode,omitempty"`
	Message               string               `json:"message,omitempty"`
	SizeBytes             int64                `json:"sizeBytes,omitempty"`
	CustomLatencyMS       float64              `json:"customLatencyMs,omitempty"`
	Payload               any                  `json:"payload,omitempty"`
	StepName              string               `json:"stepName,omitempty"`
	StopScenario          bool                 `json:"stopScenario,omitempty"`
	StopScenarioName      string               `json:"stopScenarioName,omitempty"`
	StopScenarioReason    string               `json:"stopScenarioReason,omitempty"`
	StopCurrentTest       bool                 `json:"stopCurrentTest,omitempty"`
	StopCurrentTestReason string               `json:"stopCurrentTestReason,omitempty"`
	Logs                  []runtimeCallbackLog `json:"logs,omitempty"`
}

const (
	runtimeCallbackMaximumLogCount = 1000
	runtimeCallbackMaximumLogBytes = 256 * 1024
)

var (
	errRuntimeCallbackLogLimitExceeded = errors.New("runtime callback log limit exceeded")
	errRuntimeCallbackLogInvalidUTF8   = errors.New("runtime callback log contains invalid UTF-8")
)

type runtimeCallbackLog struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type runtimeCallbackLogBuffer struct {
	mu             sync.Mutex
	logs           []runtimeCallbackLog
	aggregateBytes int
	err            error
}

func newRuntimeCallbackLogBuffer() *runtimeCallbackLogBuffer {
	return &runtimeCallbackLogBuffer{logs: make([]runtimeCallbackLog, 0)}
}

func (b *runtimeCallbackLogBuffer) logger() *LoadStrikeLogger {
	return newLoadStrikeLogger(func(level string, message string) {
		b.append(level, message)
	})
}

func (b *runtimeCallbackLogBuffer) append(level string, message string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return
	}
	if !utf8.ValidString(level) || !utf8.ValidString(message) {
		b.err = errRuntimeCallbackLogInvalidUTF8
		return
	}
	entryBytes := len(level) + len(message)
	if len(b.logs) >= runtimeCallbackMaximumLogCount || b.aggregateBytes+entryBytes > runtimeCallbackMaximumLogBytes {
		b.err = errRuntimeCallbackLogLimitExceeded
		return
	}
	b.logs = append(b.logs, runtimeCallbackLog{Level: level, Message: message})
	b.aggregateBytes += entryBytes
}

func (b *runtimeCallbackLogBuffer) result() ([]runtimeCallbackLog, error) {
	if b == nil {
		return nil, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return nil, b.err
	}
	return append([]runtimeCallbackLog(nil), b.logs...), nil
}

type runtimeHTTPHostContextPayload struct {
	TestSuite   string   `json:"testSuite,omitempty"`
	TestName    string   `json:"testName,omitempty"`
	SessionID   string   `json:"sessionId,omitempty"`
	ClusterID   string   `json:"clusterId,omitempty"`
	AgentGroup  string   `json:"agentGroup,omitempty"`
	NodeType    NodeType `json:"nodeType,omitempty"`
	AgentsCount int      `json:"agentsCount,omitempty"`
}

type runtimeHTTPSessionScenarioPayload struct {
	ScenarioName string `json:"scenarioName"`
	SortIndex    int    `json:"sortIndex"`
}

type runtimeHTTPSessionInfoPayload struct {
	Scenarios []runtimeHTTPSessionScenarioPayload `json:"scenarios,omitempty"`
}

type runtimeHTTPPluginRequest struct {
	Stage       string                         `json:"stage"`
	PluginName  string                         `json:"pluginName,omitempty"`
	Context     *runtimeHTTPHostContextPayload `json:"context,omitempty"`
	SessionInfo *runtimeHTTPSessionInfoPayload `json:"sessionInfo,omitempty"`
	InfraConfig map[string]any                 `json:"infraConfig,omitempty"`
	Result      *runResult                     `json:"result,omitempty"`
}

type runtimeHTTPSinkRequest struct {
	Stage               string                               `json:"stage"`
	SinkName            string                               `json:"sinkName,omitempty"`
	Context             *runtimeHTTPHostContextPayload       `json:"context,omitempty"`
	SessionInfo         *runtimeHTTPSessionInfoPayload       `json:"sessionInfo,omitempty"`
	InfraConfig         map[string]any                       `json:"infraConfig,omitempty"`
	RealtimeStats       []LoadStrikeScenarioStats            `json:"realtimeStats,omitempty"`
	RealtimeMetric      *LoadStrikeMetricStats               `json:"realtimeMetrics,omitempty"`
	Result              *runResult                           `json:"result,omitempty"`
	IterationBatch      *LoadStrikeIterationObservationBatch `json:"iterationBatch,omitempty"`
	IterationCompletion *LoadStrikeIterationStreamCompletion `json:"iterationCompletion,omitempty"`
}

type runtimeHTTPPolicyRequest struct {
	Stage        string                     `json:"stage"`
	ScenarioName string                     `json:"scenarioName,omitempty"`
	StepName     string                     `json:"stepName,omitempty"`
	Stats        *LoadStrikeScenarioRuntime `json:"stats,omitempty"`
	Reply        *runtimeHTTPStepResponse   `json:"reply,omitempty"`
}

type runtimeHTTPPolicyResponse struct {
	ShouldRun bool `json:"shouldRun"`
}

type runtimeHTTPThresholdRequest struct {
	ScenarioStats *scenarioStats `json:"scenarioStats,omitempty"`
	StepStats     *stepStats     `json:"stepStats,omitempty"`
	MetricStats   *metricStats   `json:"metricStats,omitempty"`
}

func (r runtimeHTTPThresholdRequest) validateForScope(scope string) error {
	payloadCount := 0
	if r.ScenarioStats != nil {
		payloadCount++
	}
	if r.StepStats != nil {
		payloadCount++
	}
	if r.MetricStats != nil {
		payloadCount++
	}
	if payloadCount != 1 {
		return fmt.Errorf("threshold predicate callback requires exactly one stats payload")
	}

	switch scope {
	case "scenario":
		if r.ScenarioStats == nil {
			return fmt.Errorf("threshold predicate callback requires scenarioStats")
		}
	case "step":
		if r.StepStats == nil {
			return fmt.Errorf("threshold predicate callback requires stepStats")
		}
	case "metric":
		if r.MetricStats == nil {
			return fmt.Errorf("threshold predicate callback requires metricStats")
		}
	default:
		return fmt.Errorf("unsupported threshold predicate scope %q", scope)
	}
	return nil
}

type runtimeHTTPThresholdResponse struct {
	Passed bool `json:"passed"`
}

type runtimeHTTPReportingPublishRequest struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

type runtimeHTTPObservationCancellationResponse struct {
	Cancelled bool `json:"cancelled"`
}

func startRuntimeHTTPHostServer(registry *runtimeCallbackRegistry) (*runtimeHTTPHostHandle, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for runtime http host server: %w", err)
	}

	handle := &runtimeHTTPHostHandle{
		baseURL:  "http://" + listener.Addr().String(),
		listener: listener,
		registry: registry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callbacks/scenario/", handle.handleScenarioCallback(registry))
	mux.HandleFunc("/callbacks/plugin/", handle.handlePluginCallback(registry))
	mux.HandleFunc("/callbacks/sink/", handle.handleSinkCallback(registry))
	mux.HandleFunc("/callbacks/policy/", handle.handlePolicyCallback(registry))
	mux.HandleFunc("/callbacks/tracking/", handle.handleTrackingCallback(registry))
	mux.HandleFunc("/callbacks/threshold/", handle.handleThresholdCallback(registry))
	mux.HandleFunc("/callbacks/reporting-publish/", handle.handleReportingPublishCallback(registry))
	mux.HandleFunc("/callbacks/observation-cancellation/", handle.handleObservationCancellationCallback(registry))

	handle.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = handle.server.Serve(listener)
	}()

	return handle, nil
}

// Close releases owned resources. Use this when the current SDK object is no longer needed.
func (h *runtimeHTTPHostHandle) Close() {
	if h == nil || h.server == nil {
		return
	}

	h.closeOnce.Do(func() {
		if h.registry != nil {
			h.registry.Close()
		}
		ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 2*time.Second)
		defer cancel()

		_ = h.server.Shutdown(ctx)
		_ = h.listener.Close()
	})
}

func (h *runtimeHTTPHostHandle) scenarioCallbackURL(id string, stage string) string {
	return h.baseURL + "/callbacks/scenario/" + id + "/" + stage
}

func (h *runtimeHTTPHostHandle) pluginCallbackURL(id string) string {
	return h.baseURL + "/callbacks/plugin/" + id
}

func (h *runtimeHTTPHostHandle) sinkCallbackURL(id string) string {
	return h.baseURL + "/callbacks/sink/" + id
}

func (h *runtimeHTTPHostHandle) policyCallbackURL(id string) string {
	return h.baseURL + "/callbacks/policy/" + id
}

func (h *runtimeHTTPHostHandle) trackingProduceCallbackURL(id string) string {
	return h.baseURL + "/callbacks/tracking/" + id + "/produce"
}

func (h *runtimeHTTPHostHandle) trackingConsumeCallbackURL(id string) string {
	return h.baseURL + "/callbacks/tracking/" + id + "/consume"
}

func (h *runtimeHTTPHostHandle) thresholdCallbackURL(id string) string {
	return h.baseURL + "/callbacks/threshold/" + id
}

func (h *runtimeHTTPHostHandle) reportingPublishCallbackURL(id string) string {
	return h.baseURL + "/callbacks/reporting-publish/" + id
}

func (h *runtimeHTTPHostHandle) observationCancellationCallbackURL(id string) string {
	return h.baseURL + "/callbacks/observation-cancellation/" + id
}

func (h *runtimeHTTPHostHandle) handleObservationCancellationCallback(
	registry *runtimeCallbackRegistry,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		const pathPrefix = "/callbacks/observation-cancellation/"
		id := strings.TrimPrefix(request.URL.Path, pathPrefix)
		if id == request.URL.Path || id == "" || id != strings.TrimSpace(id) || strings.ContainsAny(id, `/\\`) {
			http.NotFound(writer, request)
			return
		}
		observationContext, ok := registry.lookupObservationCancellation(id)
		if !ok || observationContext == nil {
			http.NotFound(writer, request)
			return
		}

		cancelled := false
		select {
		case <-observationContext.Done():
			cancelled = true
		default:
		}
		writeRuntimeCallbackJSON(writer, runtimeHTTPObservationCancellationResponse{Cancelled: cancelled})
	}
}

func (h *runtimeHTTPHostHandle) handleReportingPublishCallback(
	registry *runtimeCallbackRegistry,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.Trim(strings.TrimPrefix(request.URL.Path, "/callbacks/reporting-publish/"), "/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(writer, request)
			return
		}
		registration, ok := registry.lookupReportingPublisher(id)
		if !ok || registration.Publish == nil {
			http.NotFound(writer, request)
			return
		}

		decoder := json.NewDecoder(request.Body)
		var payload runtimeHTTPReportingPublishRequest
		if err := decoder.Decode(&payload); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			http.Error(writer, "reporting publisher callback requires exactly one JSON payload", http.StatusBadRequest)
			return
		}

		if err := registration.Publish(payload.Topic, payload.Payload); err != nil {
			http.Error(writer, sanitizeRuntimeDiagnostic(err.Error()), http.StatusInternalServerError)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}
}

func (h *runtimeHTTPHostHandle) handleThresholdCallback(registry *runtimeCallbackRegistry) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.Trim(strings.TrimPrefix(request.URL.Path, "/callbacks/threshold/"), "/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(writer, request)
			return
		}
		registration, ok := registry.lookupThreshold(id)
		if !ok || registration.Evaluate == nil {
			http.NotFound(writer, request)
			return
		}

		decoder := json.NewDecoder(request.Body)
		var payload runtimeHTTPThresholdRequest
		if err := decoder.Decode(&payload); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			http.Error(writer, "threshold predicate callback requires exactly one JSON payload", http.StatusBadRequest)
			return
		}

		passed, err := registration.Evaluate(payload)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writeRuntimeCallbackJSON(writer, runtimeHTTPThresholdResponse{Passed: passed})
	}
}

func (h *runtimeHTTPHostHandle) handleScenarioCallback(registry *runtimeCallbackRegistry) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, stage, ok := runtimeCallbackPathParts(request.URL.Path, "/callbacks/scenario/")
		if !ok {
			http.NotFound(writer, request)
			return
		}

		registration, exists := registry.lookupScenario(id)
		if !exists {
			http.NotFound(writer, request)
			return
		}

		var payload runtimeHTTPScenarioHookRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		switch stage {
		case "step":
			response, err := runtimeInvokeScenarioStep(request.Context(), registration, payload)
			if err != nil {
				writeRuntimeScenarioCallbackError(writer, err)
				return
			}
			writeRuntimeCallbackJSON(writer, response)
		case "init":
			response, err := runtimeInvokeScenarioHook(registration, payload, registration.Init, false)
			if err != nil {
				writeRuntimeScenarioCallbackError(writer, err)
				return
			}
			writeRuntimeCallbackJSON(writer, response)
		case "clean":
			response, err := runtimeInvokeScenarioHook(registration, payload, registration.Clean, true)
			if err != nil {
				writeRuntimeScenarioCallbackError(writer, err)
				return
			}
			writeRuntimeCallbackJSON(writer, response)
		case "metrics":
			if registration.State.isClosed() {
				writeRuntimeScenarioCallbackError(writer, errRuntimeScenarioInstanceGone)
				return
			}
			writeRuntimeCallbackJSON(writer, registration.State.metricSnapshot())
		default:
			http.NotFound(writer, request)
		}
	}
}

func runtimeInvokeScenarioHook(
	registration runtimeScenarioRegistration,
	request runtimeHTTPScenarioHookRequest,
	hook func(*scenarioHookContext) error,
	complete bool,
) (runtimeHTTPScenarioHookResponse, error) {
	key, err := request.scenarioInstanceKeyForRegistration(registration.ScenarioName)
	if err != nil {
		return runtimeHTTPScenarioHookResponse{}, err
	}
	metadata, err := request.callbackMetadata()
	if err != nil {
		return runtimeHTTPScenarioHookResponse{}, err
	}

	state := registration.State
	var callbackMetrics []IMetric
	callbackLogs := newRuntimeCallbackLogBuffer()
	invoke := func(instanceData map[string]any) error {
		if hook == nil {
			return nil
		}
		hookContext := &scenarioHookContext{
			ScenarioName:         metadata.ScenarioInfo.ScenarioName,
			TestSuite:            metadata.TestInfo.TestSuite,
			TestName:             metadata.TestInfo.TestName,
			ScenarioInstanceData: instanceData,
			Partition:            metadata.Partition,
			Logger:               callbackLogs.logger(),
			nodeInfo:             metadata.NodeInfo,
			testInfo:             metadata.TestInfo,
			ScenarioInfo:         metadata.ScenarioInfo,
			CustomSettings:       newIConfiguration(metadata.CustomSettings),
			GlobalCustomSettings: newIConfiguration(metadata.GlobalCustomSettings),
			metricRegistry:       &metricRegistry{},
		}
		if err := hook(hookContext); err != nil {
			return err
		}
		if _, err := callbackLogs.result(); err != nil {
			return err
		}
		callbackMetrics = append([]IMetric(nil), hookContext.metricRegistry.metrics...)
		return nil
	}

	if complete {
		err = state.cleanInstance(key, invoke)
	} else {
		err = state.withInstance(key, invoke)
	}
	if err != nil {
		return runtimeHTTPScenarioHookResponse{}, err
	}

	if hook != nil {
		state.replaceMetrics(callbackMetrics)
	}
	logs, err := callbackLogs.result()
	if err != nil {
		return runtimeHTTPScenarioHookResponse{}, err
	}
	return runtimeHTTPScenarioHookResponse{
		Metrics: state.metricDescriptors(),
		Logs:    logs,
	}, nil
}

func runtimeInvokeScenarioStep(
	callbackContext stdcontext.Context,
	registration runtimeScenarioRegistration,
	request runtimeHTTPScenarioHookRequest,
) (runtimeHTTPStepResponse, error) {
	key, err := request.scenarioInstanceKeyForRegistration(registration.ScenarioName)
	if err != nil {
		return runtimeHTTPStepResponse{}, err
	}
	if registration.Run == nil {
		return runtimeHTTPStepResponse{}, fmt.Errorf("scenario %q did not register a runnable callback", registration.ScenarioName)
	}
	metadata, err := request.callbackMetadata()
	if err != nil {
		return runtimeHTTPStepResponse{}, err
	}

	var response runtimeHTTPStepResponse
	callbackLogs := newRuntimeCallbackLogBuffer()
	err = registration.State.withInstance(key, func(instanceData map[string]any) error {
		started := runtimeParseScenarioStartedUTC(request.ScenarioStartedUTC)
		stepContext := &stepRuntimeContext{
			ScenarioName:              metadata.ScenarioInfo.ScenarioName,
			StepName:                  "",
			TestSuite:                 metadata.TestInfo.TestSuite,
			TestName:                  metadata.TestInfo.TestName,
			InvocationNumber:          request.InvocationNumber,
			Data:                      map[string]any{},
			ScenarioInstanceData:      instanceData,
			Logger:                    callbackLogs.logger(),
			nodeInfo:                  metadata.NodeInfo,
			testInfo:                  metadata.TestInfo,
			ScenarioInfo:              metadata.ScenarioInfo,
			Partition:                 metadata.Partition,
			CustomSettings:            newIConfiguration(metadata.CustomSettings),
			GlobalCustomSettings:      newIConfiguration(metadata.GlobalCustomSettings),
			Random:                    newScenarioRandom(runtimeInvocationNumberToInt(request.InvocationNumber)),
			ScenarioCancellationToken: callbackContext,
			scenarioTimerStarted:      started,
		}

		reply := registration.Run(stepContext)
		logs, err := callbackLogs.result()
		if err != nil {
			return err
		}
		response = runtimeHTTPStepResponse{
			IsSuccess:             reply.IsSuccess,
			StatusCode:            reply.StatusCode,
			Message:               reply.Message,
			SizeBytes:             reply.SizeBytes,
			CustomLatencyMS:       reply.CustomLatencyMS,
			Payload:               reply.Payload,
			StepName:              stepContext.StepName,
			StopScenario:          stepContext.stopScenarioName != "",
			StopScenarioName:      stepContext.stopScenarioName,
			StopCurrentTest:       stepContext.stopCurrentTest,
			StopScenarioReason:    stepContext.stopScenarioReason,
			StopCurrentTestReason: stepContext.stopCurrentTestReason,
			Logs:                  logs,
		}
		return nil
	})
	if err != nil {
		return runtimeHTTPStepResponse{}, err
	}
	return response, nil
}

func runtimeInvocationNumberToInt(value int64) int {
	maxInt := int64(^uint(0) >> 1)
	minInt := -maxInt - 1
	if value > maxInt {
		return int(maxInt)
	}
	if value < minInt {
		return int(minInt)
	}
	return int(value)
}

func runtimeParseScenarioStartedUTC(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func (h *runtimeHTTPHostHandle) handlePluginCallback(registry *runtimeCallbackRegistry) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(request.URL.Path, "/callbacks/plugin/")
		plugin, ok := registry.lookupWorkerPlugin(id)
		if !ok {
			http.NotFound(writer, request)
			return
		}

		var payload runtimeHTTPPluginRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		switch strings.ToLower(strings.TrimSpace(payload.Stage)) {
		case "init":
			err := plugin.Init(runtimeBaseContextFromPayload(payload.Context), newIConfiguration(payload.InfraConfig)).Await()
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "start":
			err := plugin.Start(runtimeSessionInfoFromPayload(payload.SessionInfo)).Await()
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "getdata":
			var result LoadStrikePluginData
			var err error
			if payload.Result != nil {
				result, err = plugin.GetData(newLoadStrikeRunResult(*payload.Result)).Await()
			} else {
				result, err = plugin.GetData(LoadStrikeRunResult{}).Await()
			}
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, result)
		case "stop":
			if err := plugin.Stop().Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "dispose":
			if err := plugin.Dispose().Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}
}

func (h *runtimeHTTPHostHandle) handleSinkCallback(registry *runtimeCallbackRegistry) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(request.URL.Path, "/callbacks/sink/")
		sink, ok := registry.lookupReportingSink(id)
		if !ok {
			http.NotFound(writer, request)
			return
		}

		var payload runtimeHTTPSinkRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		switch strings.ToLower(strings.TrimSpace(payload.Stage)) {
		case "init":
			if err := sink.Init(runtimeBaseContextFromPayload(payload.Context), newIConfiguration(payload.InfraConfig)).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "start":
			if err := sink.Start(runtimeSessionInfoFromPayload(payload.SessionInfo)).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "saverealtimestats":
			if err := sink.SaveRealtimeStats(payload.RealtimeStats).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "saverealtimemetrics":
			metrics := LoadStrikeMetricStats{}
			if payload.RealtimeMetric != nil {
				metrics = *payload.RealtimeMetric
			}
			if err := sink.SaveRealtimeMetrics(metrics).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "saverunresult":
			result := LoadStrikeRunResult{}
			if payload.Result != nil {
				result = newLoadStrikeRunResult(*payload.Result)
			}
			if err := sink.SaveRunResult(result).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "saveiterationbatch":
			iterationSink, ok := sink.(LoadStrikeIterationBatchSink)
			if !ok {
				http.Error(writer, "reporting sink does not support raw iteration batches", http.StatusNotImplemented)
				return
			}
			batch := LoadStrikeIterationObservationBatch{}
			if payload.IterationBatch != nil {
				batch = *payload.IterationBatch
			}
			if err := iterationSink.SaveIterationBatch(batch).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "completeiterationstream":
			iterationSink, ok := sink.(LoadStrikeIterationBatchSink)
			if !ok {
				http.Error(writer, "reporting sink does not support raw iteration batches", http.StatusNotImplemented)
				return
			}
			completion := LoadStrikeIterationStreamCompletion{}
			if payload.IterationCompletion != nil {
				completion = *payload.IterationCompletion
			}
			if err := iterationSink.CompleteIterationStream(completion).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "stop":
			if err := sink.Stop().Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "dispose":
			sink.Dispose()
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}
}

func (h *runtimeHTTPHostHandle) handlePolicyCallback(registry *runtimeCallbackRegistry) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(request.URL.Path, "/callbacks/policy/")
		policy, ok := registry.lookupRuntimePolicy(id)
		if !ok {
			http.NotFound(writer, request)
			return
		}

		var payload runtimeHTTPPolicyRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		switch strings.ToLower(strings.TrimSpace(payload.Stage)) {
		case "shouldrunscenario":
			value, err := policy.ShouldRunScenario(payload.ScenarioName).Await()
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, runtimeHTTPPolicyResponse{ShouldRun: value})
		case "beforescenario":
			if err := policy.BeforeScenario(payload.ScenarioName).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "afterscenario":
			stats := LoadStrikeScenarioRuntime{}
			if payload.Stats != nil {
				stats = *payload.Stats
			}
			if err := policy.AfterScenario(payload.ScenarioName, stats).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "beforestep":
			if err := policy.BeforeStep(payload.ScenarioName, payload.StepName).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		case "afterstep":
			reply := LoadStrikeReply{}
			if payload.Reply != nil {
				reply = runtimeReplyPayloadToPublicReply(*payload.Reply)
			}
			if err := policy.AfterStep(payload.ScenarioName, payload.StepName, reply).Await(); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}
}

func (h *runtimeHTTPHostHandle) handleTrackingCallback(registry *runtimeCallbackRegistry) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, stage, ok := runtimeCallbackPathParts(request.URL.Path, "/callbacks/tracking/")
		if !ok {
			http.NotFound(writer, request)
			return
		}

		registration, exists := registry.lookupTracking(id)
		if !exists {
			http.NotFound(writer, request)
			return
		}

		switch stage {
		case "produce":
			if registration.Produce == nil {
				http.NotFound(writer, request)
				return
			}
			var payload struct {
				Payload TrackingPayload `json:"payload"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
			result, err := registration.Produce(request.Context(), payload.Payload)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, result)
		case "consume":
			if registration.consumeStream == nil {
				http.NotFound(writer, request)
				return
			}
			messages, completed, err := registration.consumeStream.poll(request.Context())
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, map[string]any{
				"messages":  messages,
				"completed": completed,
			})
		default:
			http.NotFound(writer, request)
		}
	}
}

func runtimeBaseContextFromPayload(payload *runtimeHTTPHostContextPayload) LoadStrikeBaseContext {
	context := &contextState{}
	if payload != nil {
		context.TestSuite = payload.TestSuite
		context.TestName = payload.TestName
		context.SessionID = payload.SessionID
		context.ClusterID = payload.ClusterID
		context.AgentGroup = payload.AgentGroup
		context.NodeType = payload.NodeType
		context.AgentsCount = payload.AgentsCount
	}
	context.Logger = newLoadStrikeLogger(nil)
	context.nodeInfo = currentNodeInfo(context.NodeType, LoadStrikeOperationTypeComplete)
	context.testInfo = normalizeTestInfo(testInfo{
		TestSuite:  context.TestSuite,
		TestName:   context.TestName,
		SessionID:  context.SessionID,
		ClusterID:  context.ClusterID,
		CreatedUTC: time.Now().UTC(),
	})
	return newLoadStrikeBaseContext(context)
}

func runtimeSessionInfoFromPayload(payload *runtimeHTTPSessionInfoPayload) LoadStrikeSessionStartInfo {
	if payload == nil {
		return newLoadStrikeSessionStartInfo(&sessionStartInfo{})
	}
	native := sessionStartInfo{
		Scenarios: make([]scenarioStartInfo, 0, len(payload.Scenarios)),
	}
	for _, scenario := range payload.Scenarios {
		native.Scenarios = append(native.Scenarios, scenarioStartInfo{
			ScenarioName: scenario.ScenarioName,
			SortIndex:    scenario.SortIndex,
		})
	}
	return newLoadStrikeSessionStartInfo(&native)
}

func runtimeReplyPayloadToPublicReply(payload runtimeHTTPStepResponse) LoadStrikeReply {
	if payload.IsSuccess {
		return LoadStrikeResponse.OkWith(payload.Payload, payload.StatusCode, payload.Message, payload.SizeBytes, payload.CustomLatencyMS).AsReply()
	}
	return LoadStrikeResponse.FailWith(payload.Payload, payload.StatusCode, payload.Message, payload.SizeBytes, payload.CustomLatencyMS).AsReply()
}

func runtimeCallbackPathParts(path string, prefix string) (string, string, bool) {
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func writeRuntimeCallbackJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(value)
}
