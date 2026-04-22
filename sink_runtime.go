package loadstrike

import (
	"bytes"
	stdcontext "context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type reportingSink interface {
	name() string
	init(contextState, map[string]any) error
	start(LoadStrikeSessionStartInfo) error
	saveRealtimeStats([]scenarioStats) error
	saveRealtimeMetrics(metricStats) error
	saveRunResult(runResult) error
	stop() error
	dispose() error
}

type reportingSinkManager struct {
	context     contextState
	session     sinkSessionMetadata
	active      []reportingSink
	disabled    map[string]struct{}
	sinkErrors  []sinkErrorResult
	infraConfig map[string]any
}

func newReportingSinkManager(context contextState) (*reportingSinkManager, error) {
	sinks, err := buildReportingSinks(context.ReportingSinks)
	if err != nil {
		return nil, err
	}
	return &reportingSinkManager{
		context:     context,
		session:     newSinkSessionMetadata(context),
		active:      sinks,
		disabled:    map[string]struct{}{},
		sinkErrors:  []sinkErrorResult{},
		infraConfig: loadInfraConfigMap(context.InfraConfigPath),
	}, nil
}

func (m *reportingSinkManager) init() {
	active := make([]reportingSink, 0, len(m.active))
	for _, sink := range m.active {
		if m.executeStage(sink, "init", true, func() error { return sink.init(m.context, m.infraConfig) }) {
			active = append(active, sink)
		}
	}
	m.active = active
}

func (m *reportingSinkManager) start(sessionInfo LoadStrikeSessionStartInfo) {
	active := make([]reportingSink, 0, len(m.active))
	for _, sink := range m.active {
		if m.executeStage(sink, "start", true, func() error { return sink.start(sessionInfo) }) {
			active = append(active, sink)
		}
	}
	m.active = active
}

func (m *reportingSinkManager) saveRealtime(result runResult) {
	active := make([]reportingSink, 0, len(m.active))
	for _, sink := range m.active {
		statsOK := m.executeStage(sink, "realtime", true, func() error { return sink.saveRealtimeStats(result.scenarioStats) })
		metricsOK := statsOK && m.executeStage(sink, "realtime", true, func() error { return sink.saveRealtimeMetrics(result.metricStats) })
		if statsOK && metricsOK {
			active = append(active, sink)
		}
	}
	m.active = active
}

func (m *reportingSinkManager) saveRunResult(result runResult) {
	for _, sink := range m.active {
		_ = m.executeStage(sink, "run-result", false, func() error { return sink.saveRunResult(result) })
	}
}

func (m *reportingSinkManager) stop() {
	for _, sink := range m.active {
		_ = m.executeStage(sink, "stop", false, func() error { return sink.stop() })
	}
}

func (m *reportingSinkManager) dispose() {
	for _, sink := range m.active {
		_ = m.executeStage(sink, "dispose", false, func() error { return sink.dispose() })
	}
}

func (m *reportingSinkManager) apply(result *runResult) {
	if result == nil {
		return
	}
	if len(m.disabled) > 0 {
		names := make([]string, 0, len(m.disabled))
		for name := range m.disabled {
			names = append(names, name)
		}
		sort.Strings(names)
		result.DisabledSinks = append([]string(nil), names...)
	}
	result.SinkErrors = append([]sinkErrorResult(nil), m.sinkErrors...)
}

func (m *reportingSinkManager) executeStage(sink reportingSink, phase string, disableOnFailure bool, action func() error) bool {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := action(); err == nil {
			return true
		} else {
			lastErr = err
		}
	}

	name := sink.name()
	m.sinkErrors = append(m.sinkErrors, sinkErrorResult{
		SinkName: name,
		Phase:    phase,
		Message:  lastErr.Error(),
		Attempts: maxAttempts,
	})
	if disableOnFailure {
		m.disabled[name] = struct{}{}
	}
	return false
}

type customReportingSinkAdapter struct {
	sink       any
	asyncSink  loadStrikeAsyncReportingSink
	legacySink legacyLoadStrikeReportingSink
}

func (a *customReportingSinkAdapter) name() string {
	if a.asyncSink != nil {
		return firstNonBlank(a.asyncSink.SinkName(), "reporting-sink")
	}
	if a.legacySink != nil {
		return firstNonBlank(a.legacySink.SinkName(), "reporting-sink")
	}
	switch typed := a.sink.(type) {
	case LoadStrikeReportingSink:
		return firstNonBlank(typed.SinkName(), "reporting-sink")
	case loadStrikeTaskReportingSinkCore:
		return firstNonBlank(typed.SinkName(), "reporting-sink")
	default:
		return "reporting-sink"
	}
}
func (a *customReportingSinkAdapter) init(context contextState, infra map[string]any) error {
	infraConfig := newIConfiguration(infra)
	if a.asyncSink != nil {
		return a.asyncSink.InitAsync(newLoadStrikeBaseContext(&context), infraConfig).Await()
	}
	if a.legacySink != nil {
		return a.legacySink.Init(newLoadStrikeBaseContext(&context), infraConfig)
	}
	switch typed := a.sink.(type) {
	case LoadStrikeReportingSink:
		return typed.Init(newLoadStrikeBaseContext(&context), infraConfig).Await()
	default:
		return fmt.Errorf("unsupported reporting sink type %T", a.sink)
	}
}
func (a *customReportingSinkAdapter) start(sessionInfo LoadStrikeSessionStartInfo) error {
	if a.asyncSink != nil {
		return a.asyncSink.StartAsync(sessionInfo).Await()
	}
	if a.legacySink != nil {
		return a.legacySink.Start(sessionInfo)
	}
	switch typed := a.sink.(type) {
	case LoadStrikeReportingSink:
		return typed.Start(sessionInfo).Await()
	default:
		return fmt.Errorf("unsupported reporting sink type %T", a.sink)
	}
}
func (a *customReportingSinkAdapter) saveRealtimeStats(stats []scenarioStats) error {
	publicStats := toLoadStrikeScenarioStats(stats)
	if a.asyncSink != nil {
		return a.asyncSink.SaveRealtimeStatsAsync(publicStats).Await()
	}
	if a.legacySink != nil {
		return a.legacySink.SaveRealtimeStats(publicStats)
	}
	switch typed := a.sink.(type) {
	case LoadStrikeReportingSink:
		return typed.SaveRealtimeStats(publicStats).Await()
	default:
		return fmt.Errorf("unsupported reporting sink type %T", a.sink)
	}
}
func (a *customReportingSinkAdapter) saveRealtimeMetrics(metrics metricStats) error {
	publicMetrics := newLoadStrikeMetricStats(metrics)
	if a.asyncSink != nil {
		return a.asyncSink.SaveRealtimeMetricsAsync(publicMetrics).Await()
	}
	if a.legacySink != nil {
		return a.legacySink.SaveRealtimeMetrics(publicMetrics)
	}
	switch typed := a.sink.(type) {
	case LoadStrikeReportingSink:
		return typed.SaveRealtimeMetrics(publicMetrics).Await()
	default:
		return fmt.Errorf("unsupported reporting sink type %T", a.sink)
	}
}
func (a *customReportingSinkAdapter) saveRunResult(result runResult) error {
	publicResult := newLoadStrikeRunResult(result)
	if a.asyncSink != nil {
		return a.asyncSink.SaveRunResultAsync(publicResult).Await()
	}
	if a.legacySink != nil {
		return a.legacySink.SaveRunResult(publicResult)
	}
	if typed, ok := a.sink.(LoadStrikeReportingSink); ok {
		return typed.SaveRunResult(publicResult).Await()
	}
	return nil
}
func (a *customReportingSinkAdapter) stop() error {
	if a.asyncSink != nil {
		return a.asyncSink.StopAsync().Await()
	}
	if a.legacySink != nil {
		return a.legacySink.Stop()
	}
	switch typed := a.sink.(type) {
	case LoadStrikeReportingSink:
		return typed.Stop().Await()
	default:
		return fmt.Errorf("unsupported reporting sink type %T", a.sink)
	}
}
func (a *customReportingSinkAdapter) dispose() error {
	if a.asyncSink != nil {
		if disposer, ok := a.asyncSink.(interface{ DisposeAsync() LoadStrikeTask }); ok {
			return disposer.DisposeAsync().Await()
		}
		return nil
	}
	if a.legacySink != nil {
		if disposer, ok := a.legacySink.(legacyLoadStrikeReportingSinkDisposer); ok {
			return disposer.Dispose()
		}
		return nil
	}
	if typed, ok := a.sink.(LoadStrikeReportingSink); ok {
		typed.Dispose()
		return nil
	}
	return nil
}

func buildReportingSinks(specs []LoadStrikeReportingSink) ([]reportingSink, error) {
	sinks := make([]reportingSink, 0, len(specs))
	for _, raw := range specs {
		switch spec := raw.(type) {
		case nil:
			continue
		case *reportingSinkSpec:
			if spec == nil {
				continue
			}
			typed := *spec
			switch strings.ToLower(strings.TrimSpace(typed.Kind)) {
			case "influxdb":
				sinks = append(sinks, &influxDBReportingSink{options: firstInfluxOptions(typed.InfluxDB), client: &http.Client{}})
			case "timescaledb":
				sinks = append(sinks, &timescaleDBReportingSink{options: firstTimescaleOptions(typed.TimescaleDB)})
			case "grafanaloki":
				sinks = append(sinks, &grafanaLokiReportingSink{options: firstGrafanaLokiOptions(typed.GrafanaLoki), client: &http.Client{}})
			case "datadog":
				sinks = append(sinks, &datadogReportingSink{options: firstDatadogOptions(typed.Datadog), client: &http.Client{}})
			case "splunk":
				sinks = append(sinks, &splunkReportingSink{options: firstSplunkOptions(typed.Splunk), client: &http.Client{}})
			case "otelcollector":
				sinks = append(sinks, &otelCollectorReportingSink{options: firstOTELCollectorOptions(typed.OTELCollector), client: &http.Client{}})
			case "":
			default:
				return nil, fmt.Errorf("unsupported reporting sink kind %q", typed.Kind)
			}
		case reportingSinkSpec:
			switch strings.ToLower(strings.TrimSpace(spec.Kind)) {
			case "influxdb":
				sinks = append(sinks, &influxDBReportingSink{options: firstInfluxOptions(spec.InfluxDB), client: &http.Client{}})
			case "timescaledb":
				sinks = append(sinks, &timescaleDBReportingSink{options: firstTimescaleOptions(spec.TimescaleDB)})
			case "grafanaloki":
				sinks = append(sinks, &grafanaLokiReportingSink{options: firstGrafanaLokiOptions(spec.GrafanaLoki), client: &http.Client{}})
			case "datadog":
				sinks = append(sinks, &datadogReportingSink{options: firstDatadogOptions(spec.Datadog), client: &http.Client{}})
			case "splunk":
				sinks = append(sinks, &splunkReportingSink{options: firstSplunkOptions(spec.Splunk), client: &http.Client{}})
			case "otelcollector":
				sinks = append(sinks, &otelCollectorReportingSink{options: firstOTELCollectorOptions(spec.OTELCollector), client: &http.Client{}})
			case "":
			default:
				return nil, fmt.Errorf("unsupported reporting sink kind %q", spec.Kind)
			}
		case InfluxDbReportingSink:
			sinks = append(sinks, &influxDBReportingSink{options: firstInfluxOptions(&spec.Options), client: &http.Client{}})
		case *InfluxDbReportingSink:
			if spec != nil {
				sinks = append(sinks, &influxDBReportingSink{options: firstInfluxOptions(&spec.Options), client: &http.Client{}})
			}
		case TimescaleDbReportingSink:
			sinks = append(sinks, &timescaleDBReportingSink{options: firstTimescaleOptions(&spec.Options)})
		case *TimescaleDbReportingSink:
			if spec != nil {
				sinks = append(sinks, &timescaleDBReportingSink{options: firstTimescaleOptions(&spec.Options)})
			}
		case GrafanaLokiReportingSink:
			sinks = append(sinks, &grafanaLokiReportingSink{options: firstGrafanaLokiOptions(&spec.Options), client: &http.Client{}})
		case *GrafanaLokiReportingSink:
			if spec != nil {
				sinks = append(sinks, &grafanaLokiReportingSink{options: firstGrafanaLokiOptions(&spec.Options), client: &http.Client{}})
			}
		case DatadogReportingSink:
			sinks = append(sinks, &datadogReportingSink{options: firstDatadogOptions(&spec.Options), client: &http.Client{}})
		case *DatadogReportingSink:
			if spec != nil {
				sinks = append(sinks, &datadogReportingSink{options: firstDatadogOptions(&spec.Options), client: &http.Client{}})
			}
		case SplunkReportingSink:
			sinks = append(sinks, &splunkReportingSink{options: firstSplunkOptions(&spec.Options), client: &http.Client{}})
		case *SplunkReportingSink:
			if spec != nil {
				sinks = append(sinks, &splunkReportingSink{options: firstSplunkOptions(&spec.Options), client: &http.Client{}})
			}
		case OtelCollectorReportingSink:
			sinks = append(sinks, &otelCollectorReportingSink{options: firstOTELCollectorOptions(&spec.Options), client: &http.Client{}})
		case *OtelCollectorReportingSink:
			if spec != nil {
				sinks = append(sinks, &otelCollectorReportingSink{options: firstOTELCollectorOptions(&spec.Options), client: &http.Client{}})
			}
		default:
			sinks = append(sinks, &customReportingSinkAdapter{sink: raw})
		}
	}
	return sinks, nil
}

func loadInfraConfigMap(path string) map[string]any {
	if strings.TrimSpace(path) == "" {
		return map[string]any{}
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}

func infraSection(infra map[string]any, keys ...string) map[string]any {
	current := infra
	for _, key := range keys {
		value, ok := current[key]
		if !ok {
			found := false
			for existingKey, existingValue := range current {
				if strings.EqualFold(existingKey, key) {
					value = existingValue
					found = true
					break
				}
			}
			if !found {
				return map[string]any{}
			}
		}
		record, ok := value.(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = record
	}
	return current
}

type influxDBReportingSink struct {
	options InfluxDBSinkOptions
	client  *http.Client
	session sinkSessionMetadata
}

func (s *influxDBReportingSink) name() string { return "influxdb" }

func (s *influxDBReportingSink) init(context contextState, infra map[string]any) error {
	s.session = newSinkSessionMetadata(context)
	mergeInfluxOptions(&s.options, infraSection(infra, "LoadStrike", "ReportingSinks", "InfluxDb"))
	if strings.TrimSpace(s.options.BaseURL) == "" {
		return fmt.Errorf("InfluxDbReportingSink requires BaseUrl")
	}
	if strings.TrimSpace(s.options.Organization) == "" {
		return fmt.Errorf("InfluxDbReportingSink requires Organization")
	}
	if strings.TrimSpace(s.options.Bucket) == "" {
		return fmt.Errorf("InfluxDbReportingSink requires Bucket")
	}
	if s.options.TimeoutSeconds > 0 {
		s.client.Timeout = time.Duration(s.options.TimeoutSeconds) * time.Second
	}
	return nil
}

func (s *influxDBReportingSink) start(LoadStrikeSessionStartInfo) error { return nil }

func (s *influxDBReportingSink) saveRealtimeStats(stats []scenarioStats) error {
	return s.persist(sinkEventsFromRealtimeStats(s.session, stats))
}

func (s *influxDBReportingSink) saveRealtimeMetrics(metrics metricStats) error {
	return s.persist(sinkEventsFromRealtimeMetrics(s.session, metrics))
}

func (s *influxDBReportingSink) saveRunResult(result runResult) error {
	return s.persist(sinkEventsFromRunResult(s.session, result))
}

func (s *influxDBReportingSink) stop() error    { return nil }
func (s *influxDBReportingSink) dispose() error { return nil }

func (s *influxDBReportingSink) persist(events []reportingSinkEvent) error {
	if len(events) == 0 {
		return nil
	}
	lines := strings.Builder{}
	for _, event := range events {
		appendInfluxRow(&lines, firstNonBlank(s.options.MeasurementName, "loadstrike"), event.OccurredUTC, mergeStringMaps(map[string]string{"source": "loadstrike", "sink": s.name()}, s.options.StaticTags, event.Tags), event.Fields)
	}
	for _, point := range reportingSinkMetricProjectionFromEvents(s.name(), events, s.options.StaticTags) {
		appendInfluxRow(&lines, firstNonBlank(s.options.MetricsMeasurementName, "loadstrike_metrics"), point.OccurredUTC, point.Tags, map[string]any{
			"metric_name":     point.MetricName,
			"metric_kind":     point.MetricKind,
			"value":           point.Value,
			"unit_of_measure": point.UnitOfMeasure,
		})
	}
	if lines.Len() == 0 {
		return nil
	}
	path := firstNonBlank(s.options.WriteEndpointPath, "/api/v2/write")
	query := fmt.Sprintf("?org=%s&bucket=%s&precision=ns", urlEscape(s.options.Organization), urlEscape(s.options.Bucket))
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(path)+query, strings.NewReader(lines.String()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")
	if strings.TrimSpace(s.options.Token) != "" {
		request.Header.Set("Authorization", "Token "+s.options.Token)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := readAllString(response)
		return fmt.Errorf("InfluxDbReportingSink write failed with status %d: %s", response.StatusCode, body)
	}
	return nil
}

type grafanaLokiReportingSink struct {
	options GrafanaLokiSinkOptions
	client  *http.Client
	session sinkSessionMetadata
}

func (s *grafanaLokiReportingSink) name() string { return "grafana-loki" }

func (s *grafanaLokiReportingSink) init(context contextState, infra map[string]any) error {
	s.session = newSinkSessionMetadata(context)
	mergeGrafanaLokiOptions(&s.options, infraSection(infra, "LoadStrike", "ReportingSinks", "GrafanaLoki"))
	if strings.TrimSpace(s.options.BaseURL) == "" {
		return fmt.Errorf("GrafanaLokiReportingSink requires BaseUrl")
	}
	if s.options.TimeoutSeconds > 0 {
		s.client.Timeout = time.Duration(s.options.TimeoutSeconds) * time.Second
	}
	return nil
}

func (s *grafanaLokiReportingSink) start(LoadStrikeSessionStartInfo) error { return nil }

func (s *grafanaLokiReportingSink) saveRealtimeStats(stats []scenarioStats) error {
	return s.persist(sinkEventsFromRealtimeStats(s.session, stats))
}

func (s *grafanaLokiReportingSink) saveRealtimeMetrics(metrics metricStats) error {
	return s.persist(sinkEventsFromRealtimeMetrics(s.session, metrics))
}

func (s *grafanaLokiReportingSink) saveRunResult(result runResult) error {
	return s.persist(sinkEventsFromRunResult(s.session, result))
}

func (s *grafanaLokiReportingSink) stop() error    { return nil }
func (s *grafanaLokiReportingSink) dispose() error { return nil }

func (s *grafanaLokiReportingSink) persist(events []reportingSinkEvent) error {
	if len(events) == 0 {
		return nil
	}

	grouped := map[string]map[string]any{}
	for _, event := range events {
		labels := map[string]string{
			"source":     "loadstrike",
			"sink":       "grafana_loki",
			"event_type": sanitizeLokiLabelValue(event.EventType),
			"test_suite": sanitizeLokiLabelValue(firstNonBlank(event.TestSuite, "default")),
			"test_name":  sanitizeLokiLabelValue(firstNonBlank(event.TestName, "loadstrike")),
		}
		for key, value := range s.options.StaticLabels {
			if strings.TrimSpace(value) != "" {
				labels[sanitizeLokiLabelKey(key)] = sanitizeLokiLabelValue(value)
			}
		}
		key := labelKey(labels)
		stream, exists := grouped[key]
		if !exists {
			stream = map[string]any{
				"stream": labels,
				"values": [][]string{},
			}
		}
		stream["values"] = append(stream["values"].([][]string), []string{
			fmt.Sprintf("%d", toUnixNanoseconds(event.OccurredUTC)),
			reportingSinkEventJSON(event),
		})
		grouped[key] = stream
	}
	streams := make([]map[string]any, 0, len(grouped))
	for _, stream := range grouped {
		streams = append(streams, stream)
	}
	if err := postJSONWithHeaders(s.client, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(firstNonBlank(s.options.PushEndpointPath, "/loki/api/v1/push")), map[string]any{"streams": streams}, s.grafanaHeaders()); err != nil {
		return err
	}

	points := reportingSinkMetricProjectionFromEvents(s.name(), events, map[string]string{})
	if len(points) == 0 {
		return nil
	}
	metricsPayload := map[string]any{
		"resourceMetrics": []any{
			map[string]any{
				"resource": map[string]any{
					"attributes": []any{
						map[string]any{"key": "service.name", "value": map[string]any{"stringValue": "loadstrike"}},
						map[string]any{"key": "loadstrike.sink", "value": map[string]any{"stringValue": s.name()}},
					},
				},
				"scopeMetrics": []any{
					map[string]any{
						"scope":   map[string]any{"name": "loadstrike.reporting", "version": "1.0.0"},
						"metrics": buildOTELMetrics(points),
					},
				},
			},
		},
	}
	metricsBaseURL := strings.TrimRight(firstNonBlank(s.options.MetricsBaseURL, s.options.BaseURL), "/")
	metricsHeaders := mergeStringMaps(s.grafanaHeaders(), s.options.MetricsHeaders)
	return postJSONWithHeaders(s.client, metricsBaseURL+ensureLeadingSlash(firstNonBlank(s.options.MetricsEndpointPath, "/v1/metrics")), metricsPayload, metricsHeaders)
}

func (s *grafanaLokiReportingSink) grafanaHeaders() map[string]string {
	headers := map[string]string{}
	if strings.TrimSpace(s.options.BearerToken) != "" {
		headers["Authorization"] = "Bearer " + s.options.BearerToken
	} else if strings.TrimSpace(s.options.Username) != "" {
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(s.options.Username+":"+s.options.Password))
	}
	if strings.TrimSpace(s.options.TenantID) != "" {
		headers["X-Scope-OrgID"] = s.options.TenantID
	}
	return headers
}

type datadogReportingSink struct {
	options DatadogSinkOptions
	client  *http.Client
	session sinkSessionMetadata
}

func (s *datadogReportingSink) name() string { return "datadog" }

func (s *datadogReportingSink) init(context contextState, infra map[string]any) error {
	s.session = newSinkSessionMetadata(context)
	mergeDatadogOptions(&s.options, infraSection(infra, "LoadStrike", "ReportingSinks", "Datadog"))
	if strings.TrimSpace(s.options.BaseURL) == "" {
		return fmt.Errorf("DatadogReportingSink requires BaseUrl")
	}
	if strings.TrimSpace(s.options.APIKey) == "" {
		return fmt.Errorf("DatadogReportingSink requires ApiKey")
	}
	if s.options.TimeoutSeconds > 0 {
		s.client.Timeout = time.Duration(s.options.TimeoutSeconds) * time.Second
	}
	return nil
}

func (s *datadogReportingSink) start(LoadStrikeSessionStartInfo) error { return nil }

func (s *datadogReportingSink) saveRealtimeStats(stats []scenarioStats) error {
	return s.persist(sinkEventsFromRealtimeStats(s.session, stats))
}

func (s *datadogReportingSink) saveRealtimeMetrics(metrics metricStats) error {
	return s.persist(sinkEventsFromRealtimeMetrics(s.session, metrics))
}

func (s *datadogReportingSink) saveRunResult(result runResult) error {
	return s.persist(sinkEventsFromRunResult(s.session, result))
}

func (s *datadogReportingSink) stop() error    { return nil }
func (s *datadogReportingSink) dispose() error { return nil }

func (s *datadogReportingSink) persist(events []reportingSinkEvent) error {
	if len(events) == 0 {
		return nil
	}
	logs := make([]map[string]any, 0, len(events))
	for _, event := range events {
		attributes := map[string]any{
			"intake_type":   "log",
			"event_type":    event.EventType,
			"occurred_utc":  event.OccurredUTC,
			"session_id":    event.SessionID,
			"test_suite":    event.TestSuite,
			"test_name":     event.TestName,
			"cluster_id":    event.ClusterID,
			"node_type":     event.NodeType,
			"machine_name":  event.MachineName,
			"scenario_name": event.ScenarioName,
			"step_name":     event.StepName,
			"tags":          event.Tags,
			"fields":        event.Fields,
		}
		for key, value := range s.options.StaticAttributes {
			if strings.TrimSpace(value) != "" {
				attributes[key] = value
			}
		}
		logs = append(logs, map[string]any{
			"message":    reportingSinkEventJSON(event),
			"ddsource":   firstNonBlank(s.options.Source, "loadstrike"),
			"service":    firstNonBlank(s.options.Service, "loadstrike"),
			"hostname":   firstNonBlank(s.options.Host, event.MachineName),
			"status":     "info",
			"ddtags":     datadogTagString(event.Tags, s.options.StaticTags),
			"attributes": attributes,
		})
	}
	headers := s.datadogHeaders()
	if err := postJSONWithHeaders(s.client, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(firstNonBlank(s.options.LogsEndpointPath, "/api/v2/logs")), logs, headers); err != nil {
		return err
	}

	points := reportingSinkMetricProjectionFromEvents(s.name(), events, s.options.StaticTags)
	if len(points) == 0 {
		return nil
	}
	series := make([]map[string]any, 0, len(points))
	for _, point := range points {
		tags := []string{}
		for _, tag := range sortedTags(point.Tags) {
			tags = append(tags, sanitizeTagComponent(tag[0])+":"+sanitizeTagComponent(tag[1]))
		}
		series = append(series, map[string]any{
			"metric": point.MetricName,
			"type":   point.MetricKind,
			"points": [][]any{{toUnixSeconds(point.OccurredUTC), point.Value}},
			"tags":   tags,
		})
	}
	return postJSONWithHeaders(s.client, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(firstNonBlank(s.options.MetricsEndpointPath, "/api/v2/series")), map[string]any{"series": series}, headers)
}

func (s *datadogReportingSink) datadogHeaders() map[string]string {
	headers := map[string]string{"DD-API-KEY": s.options.APIKey}
	if strings.TrimSpace(s.options.ApplicationKey) != "" {
		headers["DD-APPLICATION-KEY"] = s.options.ApplicationKey
	}
	return headers
}

type splunkReportingSink struct {
	options SplunkSinkOptions
	client  *http.Client
	session sinkSessionMetadata
}

func (s *splunkReportingSink) name() string { return "splunk" }

func (s *splunkReportingSink) init(context contextState, infra map[string]any) error {
	s.session = newSinkSessionMetadata(context)
	mergeSplunkOptions(&s.options, infraSection(infra, "LoadStrike", "ReportingSinks", "Splunk"))
	if strings.TrimSpace(s.options.BaseURL) == "" {
		return fmt.Errorf("SplunkReportingSink requires BaseUrl")
	}
	if strings.TrimSpace(s.options.Token) == "" {
		return fmt.Errorf("SplunkReportingSink requires Token")
	}
	if s.options.TimeoutSeconds > 0 {
		s.client.Timeout = time.Duration(s.options.TimeoutSeconds) * time.Second
	}
	return nil
}

func (s *splunkReportingSink) start(LoadStrikeSessionStartInfo) error { return nil }

func (s *splunkReportingSink) saveRealtimeStats(stats []scenarioStats) error {
	return s.persist(sinkEventsFromRealtimeStats(s.session, stats))
}

func (s *splunkReportingSink) saveRealtimeMetrics(metrics metricStats) error {
	return s.persist(sinkEventsFromRealtimeMetrics(s.session, metrics))
}

func (s *splunkReportingSink) saveRunResult(result runResult) error {
	return s.persist(sinkEventsFromRunResult(s.session, result))
}

func (s *splunkReportingSink) stop() error    { return nil }
func (s *splunkReportingSink) dispose() error { return nil }

func (s *splunkReportingSink) persist(events []reportingSinkEvent) error {
	if len(events) == 0 {
		return nil
	}
	envelopes := []string{}
	for _, event := range events {
		fields := map[string]string{"intake_type": "log"}
		for key, value := range s.options.StaticFields {
			if strings.TrimSpace(value) != "" {
				fields[key] = value
			}
		}
		payload, _ := json.Marshal(map[string]any{
			"time":       toUnixSeconds(event.OccurredUTC),
			"host":       firstNonBlank(s.options.Host, event.MachineName),
			"source":     firstNonBlank(s.options.Source, "loadstrike"),
			"sourcetype": firstNonBlank(s.options.Sourcetype, "_json"),
			"index":      strings.TrimSpace(s.options.Index),
			"event": map[string]any{
				"intake_type":   "log",
				"event_type":    event.EventType,
				"occurred_utc":  event.OccurredUTC,
				"session_id":    event.SessionID,
				"test_suite":    event.TestSuite,
				"test_name":     event.TestName,
				"cluster_id":    event.ClusterID,
				"node_type":     event.NodeType,
				"machine_name":  event.MachineName,
				"scenario_name": event.ScenarioName,
				"step_name":     event.StepName,
				"tags":          event.Tags,
				"fields":        event.Fields,
			},
			"fields": fields,
		})
		envelopes = append(envelopes, string(payload))
	}
	for _, point := range reportingSinkMetricProjectionFromEvents(s.name(), events, map[string]string{}) {
		fields := map[string]string{"intake_type": "metric", "metric_name": point.MetricName, "metric_kind": point.MetricKind}
		for key, value := range s.options.StaticFields {
			if strings.TrimSpace(value) != "" {
				fields[key] = value
			}
		}
		payload, _ := json.Marshal(map[string]any{
			"time":       toUnixSeconds(point.OccurredUTC),
			"host":       firstNonBlank(s.options.Host, point.Tags["machine_name"]),
			"source":     firstNonBlank(s.options.Source, "loadstrike"),
			"sourcetype": firstNonBlank(s.options.Sourcetype, "_json"),
			"index":      strings.TrimSpace(s.options.Index),
			"event": map[string]any{
				"intake_type":     "metric",
				"metric_name":     point.MetricName,
				"metric_kind":     point.MetricKind,
				"occurred_utc":    point.OccurredUTC,
				"value":           point.Value,
				"unit_of_measure": point.UnitOfMeasure,
				"tags":            point.Tags,
			},
			"fields": fields,
		})
		envelopes = append(envelopes, string(payload))
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(firstNonBlank(s.options.EventEndpointPath, "/services/collector/event")), strings.NewReader(strings.Join(envelopes, "\n")))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Splunk "+s.options.Token)
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := readAllString(response)
		return fmt.Errorf("SplunkReportingSink write failed with status %d: %s", response.StatusCode, body)
	}
	return nil
}

type otelCollectorReportingSink struct {
	options OTELCollectorSinkOptions
	client  *http.Client
	session sinkSessionMetadata
}

func (s *otelCollectorReportingSink) name() string { return "otel-collector" }

func (s *otelCollectorReportingSink) init(context contextState, infra map[string]any) error {
	s.session = newSinkSessionMetadata(context)
	mergeOTELCollectorOptions(&s.options, infraSection(infra, "LoadStrike", "ReportingSinks", "OtelCollector"))
	if strings.TrimSpace(s.options.BaseURL) == "" {
		return fmt.Errorf("OtelCollectorReportingSink requires BaseUrl")
	}
	if s.options.TimeoutSeconds > 0 {
		s.client.Timeout = time.Duration(s.options.TimeoutSeconds) * time.Second
	}
	return nil
}

func (s *otelCollectorReportingSink) start(LoadStrikeSessionStartInfo) error { return nil }

func (s *otelCollectorReportingSink) saveRealtimeStats(stats []scenarioStats) error {
	return s.persist(sinkEventsFromRealtimeStats(s.session, stats))
}

func (s *otelCollectorReportingSink) saveRealtimeMetrics(metrics metricStats) error {
	return s.persist(sinkEventsFromRealtimeMetrics(s.session, metrics))
}

func (s *otelCollectorReportingSink) saveRunResult(result runResult) error {
	return s.persist(sinkEventsFromRunResult(s.session, result))
}

func (s *otelCollectorReportingSink) stop() error    { return nil }
func (s *otelCollectorReportingSink) dispose() error { return nil }

func (s *otelCollectorReportingSink) persist(events []reportingSinkEvent) error {
	if len(events) == 0 {
		return nil
	}
	logRecords := make([]map[string]any, 0, len(events))
	for _, event := range events {
		logRecords = append(logRecords, map[string]any{
			"timeUnixNano":         fmt.Sprintf("%d", toUnixNanoseconds(event.OccurredUTC)),
			"observedTimeUnixNano": fmt.Sprintf("%d", toUnixNanoseconds(event.OccurredUTC)),
			"severityText":         "INFO",
			"body":                 map[string]any{"stringValue": reportingSinkEventJSON(event)},
			"attributes":           buildOTELAttributes(map[string]any{"intake_type": "log", "event_type": event.EventType, "session_id": event.SessionID, "test_suite": event.TestSuite, "test_name": event.TestName, "cluster_id": event.ClusterID, "node_type": event.NodeType, "machine_name": event.MachineName, "scenario_name": event.ScenarioName, "step_name": event.StepName, "tags": event.Tags, "fields": event.Fields}),
		})
	}
	logPayload := map[string]any{
		"resourceLogs": []any{
			map[string]any{
				"resource": map[string]any{"attributes": s.otelResourceAttributes()},
				"scopeLogs": []any{
					map[string]any{
						"scope":      map[string]any{"name": "loadstrike.reporting", "version": "1.0.0"},
						"logRecords": logRecords,
					},
				},
			},
		},
	}
	if err := postJSONWithHeaders(s.client, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(firstNonBlank(s.options.LogsEndpointPath, "/v1/logs")), logPayload, s.options.Headers); err != nil {
		return err
	}
	points := reportingSinkMetricProjectionFromEvents(s.name(), events, map[string]string{})
	if len(points) == 0 {
		return nil
	}
	metricsPayload := map[string]any{
		"resourceMetrics": []any{
			map[string]any{
				"resource": map[string]any{"attributes": s.otelResourceAttributes()},
				"scopeMetrics": []any{
					map[string]any{
						"scope":   map[string]any{"name": "loadstrike.reporting", "version": "1.0.0"},
						"metrics": buildOTELMetrics(points),
					},
				},
			},
		},
	}
	return postJSONWithHeaders(s.client, strings.TrimRight(s.options.BaseURL, "/")+ensureLeadingSlash(firstNonBlank(s.options.MetricsEndpointPath, "/v1/metrics")), metricsPayload, s.options.Headers)
}

func (s *otelCollectorReportingSink) otelResourceAttributes() []any {
	attributes := map[string]string{"service.name": "loadstrike", "service.namespace": "loadstrike", "loadstrike.sink": s.name()}
	for key, value := range s.options.StaticResourceAttributes {
		if strings.TrimSpace(value) != "" {
			attributes[key] = value
		}
	}
	result := []any{}
	for _, item := range sortedTags(attributes) {
		result = append(result, map[string]any{"key": item[0], "value": map[string]any{"stringValue": item[1]}})
	}
	return result
}

type timescaleDBReportingSink struct {
	options TimescaleDBSinkOptions
	pool    *pgxpool.Pool
	session sinkSessionMetadata
}

var timescaleIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (s *timescaleDBReportingSink) name() string { return "timescaledb" }

func (s *timescaleDBReportingSink) init(context contextState, infra map[string]any) error {
	s.session = newSinkSessionMetadata(context)
	mergeTimescaleOptions(&s.options, infraSection(infra, "LoadStrike", "ReportingSinks", "TimescaleDb"))
	if strings.TrimSpace(s.options.ConnectionString) == "" {
		return fmt.Errorf("TimescaleDbReportingSink requires ConnectionString")
	}
	if strings.TrimSpace(s.options.Schema) == "" {
		s.options.Schema = "public"
	}
	if strings.TrimSpace(s.options.TableName) == "" {
		s.options.TableName = "loadstrike_reporting_events"
	}
	if strings.TrimSpace(s.options.MetricsTableName) == "" {
		s.options.MetricsTableName = "loadstrike_reporting_metrics"
	}
	for _, identifier := range []string{s.options.Schema, s.options.TableName, s.options.MetricsTableName} {
		if !timescaleIdentifier.MatchString(identifier) {
			return fmt.Errorf("invalid TimescaleDb identifier %q", identifier)
		}
	}
	pool, err := pgxpool.New(stdcontext.Background(), s.options.ConnectionString)
	if err != nil {
		return err
	}
	s.pool = pool
	return s.ensureStorage()
}

func (s *timescaleDBReportingSink) start(LoadStrikeSessionStartInfo) error { return nil }

func (s *timescaleDBReportingSink) saveRealtimeStats(stats []scenarioStats) error {
	return s.persist(sinkEventsFromRealtimeStats(s.session, stats))
}

func (s *timescaleDBReportingSink) saveRealtimeMetrics(metrics metricStats) error {
	return s.persist(sinkEventsFromRealtimeMetrics(s.session, metrics))
}

func (s *timescaleDBReportingSink) saveRunResult(result runResult) error {
	return s.persist(sinkEventsFromRunResult(s.session, result))
}

func (s *timescaleDBReportingSink) stop() error { return nil }

func (s *timescaleDBReportingSink) dispose() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

func (s *timescaleDBReportingSink) ensureStorage() error {
	if s.pool == nil {
		return nil
	}
	ctx := stdcontext.Background()
	schema := quoteIdentifier(s.options.Schema)
	table := quoteIdentifier(s.options.TableName)
	metricsTable := quoteIdentifier(s.options.MetricsTableName)
	if s.options.CreateSchemaIfMissing {
		if _, err := s.pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema); err != nil {
			return err
		}
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
id BIGSERIAL PRIMARY KEY,
occurred_utc TIMESTAMPTZ NOT NULL,
session_id TEXT NOT NULL,
test_suite TEXT NULL,
test_name TEXT NULL,
cluster_id TEXT NULL,
node_type TEXT NOT NULL,
machine_name TEXT NOT NULL,
scenario_name TEXT NULL,
step_name TEXT NULL,
event_type TEXT NOT NULL,
tags JSONB NOT NULL,
fields JSONB NOT NULL
)`, schema, table))
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
id BIGSERIAL PRIMARY KEY,
occurred_utc TIMESTAMPTZ NOT NULL,
metric_name TEXT NOT NULL,
metric_kind TEXT NOT NULL,
unit_of_measure TEXT NULL,
value DOUBLE PRECISION NOT NULL,
tags JSONB NOT NULL
)`, schema, metricsTable))
	return err
}

func (s *timescaleDBReportingSink) persist(events []reportingSinkEvent) error {
	if s.pool == nil || len(events) == 0 {
		return nil
	}
	ctx := stdcontext.Background()
	for _, event := range events {
		tags, _ := json.Marshal(mergeStringMaps(map[string]string{"source": "loadstrike", "sink": s.name()}, s.options.StaticTags, event.Tags))
		fields, _ := json.Marshal(event.Fields)
		if _, err := s.pool.Exec(ctx,
			fmt.Sprintf(`INSERT INTO %s.%s (
occurred_utc, session_id, test_suite, test_name, cluster_id, node_type, machine_name, scenario_name, step_name, event_type, tags, fields
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb)`, quoteIdentifier(s.options.Schema), quoteIdentifier(s.options.TableName)),
			event.OccurredUTC, event.SessionID, nullableString(event.TestSuite), nullableString(event.TestName), nullableString(event.ClusterID), event.NodeType, event.MachineName, nullableString(event.ScenarioName), nullableString(event.StepName), event.EventType, string(tags), string(fields),
		); err != nil {
			return err
		}
	}
	for _, point := range reportingSinkMetricProjectionFromEvents(s.name(), events, s.options.StaticTags) {
		tags, _ := json.Marshal(point.Tags)
		if _, err := s.pool.Exec(ctx,
			fmt.Sprintf(`INSERT INTO %s.%s (
occurred_utc, metric_name, metric_kind, unit_of_measure, value, tags
) VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, quoteIdentifier(s.options.Schema), quoteIdentifier(s.options.MetricsTableName)),
			point.OccurredUTC, point.MetricName, point.MetricKind, nullableString(point.UnitOfMeasure), point.Value, string(tags),
		); err != nil {
			return err
		}
	}
	return nil
}

func firstInfluxOptions(options *InfluxDBSinkOptions) InfluxDBSinkOptions {
	if options == nil {
		return InfluxDBSinkOptions{}
	}
	return *options
}

func firstTimescaleOptions(options *TimescaleDBSinkOptions) TimescaleDBSinkOptions {
	if options == nil {
		return TimescaleDBSinkOptions{}
	}
	return *options
}

func firstGrafanaLokiOptions(options *GrafanaLokiSinkOptions) GrafanaLokiSinkOptions {
	if options == nil {
		return GrafanaLokiSinkOptions{}
	}
	return *options
}

func firstDatadogOptions(options *DatadogSinkOptions) DatadogSinkOptions {
	if options == nil {
		return DatadogSinkOptions{}
	}
	return *options
}

func firstSplunkOptions(options *SplunkSinkOptions) SplunkSinkOptions {
	if options == nil {
		return SplunkSinkOptions{}
	}
	return *options
}

func firstOTELCollectorOptions(options *OTELCollectorSinkOptions) OTELCollectorSinkOptions {
	if options == nil {
		return OTELCollectorSinkOptions{}
	}
	return *options
}

func mergeInfluxOptions(target *InfluxDBSinkOptions, section map[string]any) {
	if strings.TrimSpace(target.ConfigurationSectionPath) == "" {
		target.ConfigurationSectionPath = asString(section["ConfigurationSectionPath"])
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		target.BaseURL = asString(section["BaseUrl"])
	}
	if strings.TrimSpace(target.WriteEndpointPath) == "" {
		target.WriteEndpointPath = asString(section["WriteEndpointPath"])
	}
	if strings.TrimSpace(target.Organization) == "" {
		target.Organization = asString(section["Organization"])
	}
	if strings.TrimSpace(target.Bucket) == "" {
		target.Bucket = asString(section["Bucket"])
	}
	if strings.TrimSpace(target.Token) == "" {
		target.Token = asString(section["Token"])
	}
	if strings.TrimSpace(target.MeasurementName) == "" {
		target.MeasurementName = asString(section["MeasurementName"])
	}
	if strings.TrimSpace(target.MetricsMeasurementName) == "" {
		target.MetricsMeasurementName = asString(section["MetricsMeasurementName"])
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = asInt(section["TimeoutSeconds"])
	}
}

func mergeGrafanaLokiOptions(target *GrafanaLokiSinkOptions, section map[string]any) {
	if strings.TrimSpace(target.ConfigurationSectionPath) == "" {
		target.ConfigurationSectionPath = asString(section["ConfigurationSectionPath"])
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		target.BaseURL = asString(section["BaseUrl"])
	}
	if strings.TrimSpace(target.PushEndpointPath) == "" {
		target.PushEndpointPath = asString(section["PushEndpointPath"])
	}
	if strings.TrimSpace(target.BearerToken) == "" {
		target.BearerToken = asString(section["BearerToken"])
	}
	if strings.TrimSpace(target.Username) == "" {
		target.Username = asString(section["Username"])
	}
	if strings.TrimSpace(target.Password) == "" {
		target.Password = asString(section["Password"])
	}
	if strings.TrimSpace(target.TenantID) == "" {
		target.TenantID = asString(section["TenantId"])
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = asInt(section["TimeoutSeconds"])
	}
	if strings.TrimSpace(target.MetricsBaseURL) == "" {
		target.MetricsBaseURL = asString(section["MetricsBaseUrl"])
	}
	if strings.TrimSpace(target.MetricsEndpointPath) == "" {
		target.MetricsEndpointPath = asString(section["MetricsEndpointPath"])
	}
	if len(target.MetricsHeaders) == 0 {
		target.MetricsHeaders = asStringMap(section["MetricsHeaders"])
	}
}

func mergeDatadogOptions(target *DatadogSinkOptions, section map[string]any) {
	if strings.TrimSpace(target.ConfigurationSectionPath) == "" {
		target.ConfigurationSectionPath = asString(section["ConfigurationSectionPath"])
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		target.BaseURL = asString(section["BaseUrl"])
	}
	if strings.TrimSpace(target.LogsEndpointPath) == "" {
		target.LogsEndpointPath = asString(section["LogsEndpointPath"])
	}
	if strings.TrimSpace(target.MetricsEndpointPath) == "" {
		target.MetricsEndpointPath = asString(section["MetricsEndpointPath"])
	}
	if strings.TrimSpace(target.APIKey) == "" {
		target.APIKey = asString(section["ApiKey"])
	}
	if strings.TrimSpace(target.ApplicationKey) == "" {
		target.ApplicationKey = asString(section["ApplicationKey"])
	}
	if strings.TrimSpace(target.Source) == "" {
		target.Source = asString(section["Source"])
	}
	if strings.TrimSpace(target.Service) == "" {
		target.Service = asString(section["Service"])
	}
	if strings.TrimSpace(target.Host) == "" {
		target.Host = asString(section["Host"])
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = asInt(section["TimeoutSeconds"])
	}
}

func mergeSplunkOptions(target *SplunkSinkOptions, section map[string]any) {
	if strings.TrimSpace(target.ConfigurationSectionPath) == "" {
		target.ConfigurationSectionPath = asString(section["ConfigurationSectionPath"])
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		target.BaseURL = asString(section["BaseUrl"])
	}
	if strings.TrimSpace(target.EventEndpointPath) == "" {
		target.EventEndpointPath = asString(section["EventEndpointPath"])
	}
	if strings.TrimSpace(target.Token) == "" {
		target.Token = asString(section["Token"])
	}
	if strings.TrimSpace(target.Source) == "" {
		target.Source = asString(section["Source"])
	}
	if strings.TrimSpace(target.Sourcetype) == "" {
		target.Sourcetype = asString(section["Sourcetype"])
	}
	if strings.TrimSpace(target.Index) == "" {
		target.Index = asString(section["Index"])
	}
	if strings.TrimSpace(target.Host) == "" {
		target.Host = asString(section["Host"])
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = asInt(section["TimeoutSeconds"])
	}
}

func mergeOTELCollectorOptions(target *OTELCollectorSinkOptions, section map[string]any) {
	if strings.TrimSpace(target.ConfigurationSectionPath) == "" {
		target.ConfigurationSectionPath = asString(section["ConfigurationSectionPath"])
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		target.BaseURL = asString(section["BaseUrl"])
	}
	if strings.TrimSpace(target.LogsEndpointPath) == "" {
		target.LogsEndpointPath = asString(section["LogsEndpointPath"])
	}
	if strings.TrimSpace(target.MetricsEndpointPath) == "" {
		target.MetricsEndpointPath = asString(section["MetricsEndpointPath"])
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = asInt(section["TimeoutSeconds"])
	}
}

func mergeTimescaleOptions(target *TimescaleDBSinkOptions, section map[string]any) {
	if strings.TrimSpace(target.ConfigurationSectionPath) == "" {
		target.ConfigurationSectionPath = asString(section["ConfigurationSectionPath"])
	}
	if strings.TrimSpace(target.ConnectionString) == "" {
		target.ConnectionString = asString(section["ConnectionString"])
	}
	if strings.TrimSpace(target.Schema) == "" {
		target.Schema = asString(section["Schema"])
	}
	if strings.TrimSpace(target.TableName) == "" {
		target.TableName = asString(section["TableName"])
	}
	if strings.TrimSpace(target.MetricsTableName) == "" {
		target.MetricsTableName = asString(section["MetricsTableName"])
	}
}

func asStringMap(value any) map[string]string {
	record, ok := value.(map[string]any)
	if !ok {
		if typed, typedOK := value.(map[string]string); typedOK {
			return appendStringMap(typed)
		}
		return map[string]string{}
	}
	result := map[string]string{}
	for key, item := range record {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(asString(item)) != "" {
			result[key] = asString(item)
		}
	}
	return result
}

func appendStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			result[key] = value
		}
	}
	return result
}

func mergeStringMaps(maps ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, source := range maps {
		for key, value := range source {
			if strings.TrimSpace(value) != "" {
				result[key] = value
			}
		}
	}
	return result
}

func appendInfluxRow(builder *strings.Builder, measurement string, occurredUTC time.Time, tags map[string]string, fields map[string]any) {
	fieldParts := []string{}
	keys := make([]string, 0, len(fields))
	for key, value := range fields {
		if value != nil && strings.TrimSpace(asString(value)) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		fieldParts = append(fieldParts, escapeInfluxComponent(key)+"="+formatInfluxField(fields[key]))
	}
	if len(fieldParts) == 0 {
		return
	}
	builder.WriteString(escapeInfluxComponent(measurement))
	for _, tag := range sortedTags(tags) {
		builder.WriteByte(',')
		builder.WriteString(escapeInfluxComponent(tag[0]))
		builder.WriteByte('=')
		builder.WriteString(escapeInfluxComponent(tag[1]))
	}
	builder.WriteByte(' ')
	builder.WriteString(strings.Join(fieldParts, ","))
	builder.WriteByte(' ')
	builder.WriteString(fmt.Sprintf("%d", toUnixNanoseconds(occurredUTC)))
	builder.WriteByte('\n')
}

func formatInfluxField(value any) string {
	switch typed := value.(type) {
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case int, int32, int64:
		return fmt.Sprintf("%di", asInt(typed))
	case float32, float64:
		return fmt.Sprintf("%g", asDouble(typed))
	default:
		return `"` + strings.ReplaceAll(strings.ReplaceAll(asString(value), `\`, `\\`), `"`, `\"`) + `"`
	}
}

func escapeInfluxComponent(value string) string {
	return strings.NewReplacer(",", `\,`, " ", `\ `, "=", `\=`).Replace(value)
}

func sanitizeLokiLabelKey(value string) string {
	builder := strings.Builder{}
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			builder.WriteRune(ch)
		} else {
			builder.WriteRune('_')
		}
	}
	text := builder.String()
	if text == "" || !(text[0] == '_' || (text[0] >= 'A' && text[0] <= 'Z') || (text[0] >= 'a' && text[0] <= 'z')) {
		return "_" + text
	}
	return text
}

func sanitizeLokiLabelValue(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(value))
}

func labelKey(labels map[string]string) string {
	parts := []string{}
	for _, item := range sortedTags(labels) {
		parts = append(parts, item[0]+"="+item[1])
	}
	return strings.Join(parts, "|")
}

func datadogTagString(eventTags map[string]string, staticTags map[string]string) string {
	merged := mergeStringMaps(map[string]string{"source": "loadstrike"}, staticTags, eventTags)
	parts := []string{}
	for _, item := range sortedTags(merged) {
		parts = append(parts, sanitizeTagComponent(item[0])+":"+sanitizeTagComponent(item[1]))
	}
	return strings.Join(parts, ",")
}

func sanitizeTagComponent(value string) string {
	return strings.TrimSpace(strings.NewReplacer(",", "_", ":", "_", " ", "_").Replace(value))
}

func buildOTELMetrics(points []reportingSinkMetricPoint) []any {
	grouped := map[string][]reportingSinkMetricPoint{}
	for _, point := range points {
		key := strings.Join([]string{point.MetricName, point.MetricKind, point.UnitOfMeasure}, "\x00")
		grouped[key] = append(grouped[key], point)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	metrics := []any{}
	for _, key := range keys {
		group := grouped[key]
		first := group[0]
		dataPoints := []any{}
		for _, point := range group {
			attributes := map[string]any{}
			for tagKey, tagValue := range point.Tags {
				attributes[tagKey] = tagValue
			}
			attributes["intake_type"] = "metric"
			attributes["metric_kind"] = point.MetricKind
			dataPoints = append(dataPoints, map[string]any{
				"timeUnixNano": fmt.Sprintf("%d", toUnixNanoseconds(point.OccurredUTC)),
				"asDouble":     point.Value,
				"attributes":   buildOTELAttributes(attributes),
			})
		}
		metric := map[string]any{"name": first.MetricName, "description": "LoadStrike reporting sink metric projection", "unit": first.UnitOfMeasure}
		if first.MetricKind == "count" {
			metric["sum"] = map[string]any{"aggregationTemporality": 2, "isMonotonic": true, "dataPoints": dataPoints}
		} else {
			metric["gauge"] = map[string]any{"dataPoints": dataPoints}
		}
		metrics = append(metrics, metric)
	}
	return metrics
}

func buildOTELAttributes(values map[string]any) []any {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if value != nil {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	attributes := []any{}
	for _, key := range keys {
		attributes = append(attributes, map[string]any{"key": key, "value": otelAnyValue(values[key])})
	}
	return attributes
}

func otelAnyValue(value any) map[string]any {
	switch typed := value.(type) {
	case bool:
		return map[string]any{"boolValue": typed}
	case int, int32, int64:
		return map[string]any{"intValue": fmt.Sprintf("%d", asInt(typed))}
	case float32, float64:
		return map[string]any{"doubleValue": asDouble(typed)}
	default:
		body, err := json.Marshal(value)
		if err == nil && (strings.HasPrefix(string(body), "{") || strings.HasPrefix(string(body), "[")) {
			return map[string]any{"stringValue": string(body)}
		}
		return map[string]any{"stringValue": asString(value)}
	}
}

func postJSONWithHeaders(client *http.Client, url string, payload any, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			request.Header.Set(key, value)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := readAllString(response)
		return fmt.Errorf("HTTP export failed with status %d: %s", response.StatusCode, body)
	}
	return nil
}

func ensureLeadingSlash(value string) string {
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func urlEscape(value string) string {
	return strings.NewReplacer(" ", "%20", "+", "%2B", "/", "%2F", "?", "%3F", "&", "%26", "=", "%3D").Replace(value)
}

func readAllString(response *http.Response) (string, error) {
	body, err := io.ReadAll(response.Body)
	return string(body), err
}

func toUnixNanoseconds(value time.Time) int64 {
	return value.UTC().UnixNano()
}

func toUnixSeconds(value time.Time) float64 {
	return float64(value.UTC().UnixMilli()) / 1000.0
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
