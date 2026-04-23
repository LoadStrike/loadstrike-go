package loadstrike

import (
	stdcontext "context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type runtimeHTTPHostHandle struct {
	baseURL  string
	listener net.Listener
	server   *http.Server
}

type runtimeHTTPScenarioHookRequest struct {
	ScenarioName       string `json:"scenarioName"`
	TestSuite          string `json:"testSuite,omitempty"`
	TestName           string `json:"testName,omitempty"`
	InvocationNumber   int64  `json:"invocationNumber,omitempty"`
	ScenarioStartedUTC string `json:"scenarioStartedUtc,omitempty"`
	AgentIndex         int    `json:"agentIndex,omitempty"`
	AgentCount         int    `json:"agentCount,omitempty"`
}

type runtimeHTTPScenarioHookResponse struct {
	Metrics []runtimeScenarioMetricDescriptor `json:"metrics,omitempty"`
}

type runtimeHTTPStepResponse struct {
	IsSuccess             bool    `json:"isSuccess"`
	StatusCode            string  `json:"statusCode,omitempty"`
	Message               string  `json:"message,omitempty"`
	SizeBytes             int64   `json:"sizeBytes,omitempty"`
	CustomLatencyMS       float64 `json:"customLatencyMs,omitempty"`
	Payload               any     `json:"payload,omitempty"`
	StepName              string  `json:"stepName,omitempty"`
	StopScenario          bool    `json:"stopScenario,omitempty"`
	StopScenarioReason    string  `json:"stopScenarioReason,omitempty"`
	StopCurrentTest       bool    `json:"stopCurrentTest,omitempty"`
	StopCurrentTestReason string  `json:"stopCurrentTestReason,omitempty"`
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
	Stage          string                         `json:"stage"`
	SinkName       string                         `json:"sinkName,omitempty"`
	Context        *runtimeHTTPHostContextPayload `json:"context,omitempty"`
	SessionInfo    *runtimeHTTPSessionInfoPayload `json:"sessionInfo,omitempty"`
	InfraConfig    map[string]any                 `json:"infraConfig,omitempty"`
	RealtimeStats  []LoadStrikeScenarioStats      `json:"realtimeStats,omitempty"`
	RealtimeMetric *LoadStrikeMetricStats         `json:"realtimeMetrics,omitempty"`
	Result         *runResult                     `json:"result,omitempty"`
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

func startRuntimeHTTPHostServer(registry *runtimeCallbackRegistry) (*runtimeHTTPHostHandle, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for runtime http host server: %w", err)
	}

	handle := &runtimeHTTPHostHandle{
		baseURL:  "http://" + listener.Addr().String(),
		listener: listener,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callbacks/scenario/", handle.handleScenarioCallback(registry))
	mux.HandleFunc("/callbacks/plugin/", handle.handlePluginCallback(registry))
	mux.HandleFunc("/callbacks/sink/", handle.handleSinkCallback(registry))
	mux.HandleFunc("/callbacks/policy/", handle.handlePolicyCallback(registry))
	mux.HandleFunc("/callbacks/tracking/", handle.handleTrackingCallback(registry))

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

	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 2*time.Second)
	defer cancel()

	_ = h.server.Shutdown(ctx)
	_ = h.listener.Close()
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
			response, err := runtimeInvokeScenarioStep(registration, payload)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, response)
		case "init":
			response, err := runtimeInvokeScenarioHook(registration, payload, registration.Init)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, response)
		case "clean":
			response, err := runtimeInvokeScenarioHook(registration, payload, registration.Clean)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRuntimeCallbackJSON(writer, response)
		case "metrics":
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
) (runtimeHTTPScenarioHookResponse, error) {
	if hook == nil {
		return runtimeHTTPScenarioHookResponse{}, nil
	}

	state := registration.State
	hookContext := &scenarioHookContext{
		ScenarioName:         registration.ScenarioName,
		TestSuite:            request.TestSuite,
		TestName:             request.TestName,
		ScenarioInstanceData: state.scenarioInstanceDataForRequest(),
		Partition: scenarioPartitionInfo{
			Number: request.AgentIndex,
			Count:  maxInt(request.AgentCount, 1),
		},
		Logger:               newLoadStrikeLogger(nil),
		nodeInfo:             currentNodeInfo(NodeTypeSingleNode, LoadStrikeOperationTypeInit),
		testInfo:             normalizeTestInfo(testInfo{TestSuite: request.TestSuite, TestName: request.TestName}),
		ScenarioInfo:         runtimeScenarioInfoFromRequest(registration.ScenarioName, request, LoadStrikeScenarioOperationInit),
		CustomSettings:       newIConfiguration(map[string]any{}),
		GlobalCustomSettings: newIConfiguration(map[string]any{}),
		metricRegistry:       &metricRegistry{},
	}

	if err := hook(hookContext); err != nil {
		return runtimeHTTPScenarioHookResponse{}, err
	}

	state.replaceMetrics(hookContext.metricRegistry.metrics)
	return runtimeHTTPScenarioHookResponse{
		Metrics: state.metricDescriptors(),
	}, nil
}

func runtimeInvokeScenarioStep(registration runtimeScenarioRegistration, request runtimeHTTPScenarioHookRequest) (runtimeHTTPStepResponse, error) {
	if registration.Run == nil {
		return runtimeHTTPStepResponse{}, fmt.Errorf("scenario %q did not register a runnable callback", registration.ScenarioName)
	}

	started := runtimeParseScenarioStartedUTC(request.ScenarioStartedUTC)
	stepContext := &stepRuntimeContext{
		ScenarioName:              registration.ScenarioName,
		StepName:                  "",
		TestSuite:                 request.TestSuite,
		TestName:                  request.TestName,
		InvocationNumber:          request.InvocationNumber,
		Data:                      map[string]any{},
		ScenarioInstanceData:      registration.State.scenarioInstanceDataForRequest(),
		Logger:                    newLoadStrikeLogger(nil),
		nodeInfo:                  currentNodeInfo(NodeTypeSingleNode, LoadStrikeOperationTypeBombing),
		testInfo:                  normalizeTestInfo(testInfo{TestSuite: request.TestSuite, TestName: request.TestName}),
		ScenarioInfo:              runtimeScenarioInfoFromRequest(registration.ScenarioName, request, LoadStrikeScenarioOperationBombing),
		Random:                    newScenarioRandom(int(request.InvocationNumber)),
		ScenarioCancellationToken: stdcontext.Background(),
		scenarioTimerStarted:      started,
	}

	reply := registration.Run(stepContext)
	return runtimeHTTPStepResponse{
		IsSuccess:             reply.IsSuccess,
		StatusCode:            reply.StatusCode,
		Message:               reply.Message,
		SizeBytes:             reply.SizeBytes,
		CustomLatencyMS:       reply.CustomLatencyMS,
		Payload:               reply.Payload,
		StepName:              stepContext.StepName,
		StopScenario:          stepContext.stopRequested,
		StopCurrentTest:       stepContext.stopCurrentTest,
		StopScenarioReason:    "",
		StopCurrentTestReason: "",
	}, nil
}

func runtimeScenarioInfoFromRequest(
	scenarioName string,
	request runtimeHTTPScenarioHookRequest,
	operation LoadStrikeScenarioOperation,
) LoadStrikeScenarioInfo {
	started := runtimeParseScenarioStartedUTC(request.ScenarioStartedUTC)
	instanceNumber := maxInt(int(request.InvocationNumber), 1)
	if started.IsZero() {
		started = time.Now().UTC()
	}
	return normalizeScenarioInfo(LoadStrikeScenarioInfo{
		InstanceID:        fmt.Sprintf("%s-%d", scenarioName, instanceNumber),
		InstanceNumber:    instanceNumber,
		ScenarioDuration:  time.Since(started),
		ScenarioName:      scenarioName,
		ScenarioOperation: operation,
	})
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
