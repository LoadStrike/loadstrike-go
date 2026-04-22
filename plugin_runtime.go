package loadstrike

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type workerPluginRuntime interface {
	name() string
	init(contextState, map[string]any) error
	start(LoadStrikeSessionStartInfo) error
	getData(runResult) (pluginData, error)
	stop() error
	dispose() error
}

type workerPluginManager struct {
	plugins []workerPluginRuntime
}

func newWorkerPluginManager(context contextState) *workerPluginManager {
	plugins := []workerPluginRuntime{
		&failedResponseBuiltinWorkerPlugin{},
		&correlationBuiltinWorkerPlugin{},
	}

	for _, configuredPlugin := range context.WorkerPlugins {
		switch typed := configuredPlugin.(type) {
		case nil:
			continue
		case workerPluginSpec:
			plugins = appendConfiguredWorkerPlugin(plugins, typed)
		case *workerPluginSpec:
			if typed != nil {
				plugins = appendConfiguredWorkerPlugin(plugins, *typed)
			}
		case LoadStrikeWorkerPlugin:
			plugins = append(plugins, &inProcessWorkerPluginAdapter{plugin: typed})
		}
	}

	return &workerPluginManager{plugins: plugins}
}

func appendConfiguredWorkerPlugin(plugins []workerPluginRuntime, spec workerPluginSpec) []workerPluginRuntime {
	if strings.TrimSpace(spec.CallbackURL) != "" {
		return append(plugins, &httpCallbackWorkerPlugin{
			pluginName:  firstNonBlank(spec.PluginName(), "worker-plugin"),
			callbackURL: spec.CallbackURL,
			httpClient:  &http.Client{Timeout: 30 * time.Second},
		})
	}
	return append(plugins, &inProcessWorkerPluginAdapter{spec: &spec})
}

func (m *workerPluginManager) init(context contextState, infraConfig map[string]any) error {
	for _, plugin := range m.plugins {
		if err := plugin.init(context, infraConfig); err != nil {
			return err
		}
	}
	return nil
}

func (m *workerPluginManager) start(sessionInfo LoadStrikeSessionStartInfo) error {
	for _, plugin := range m.plugins {
		if err := plugin.start(sessionInfo); err != nil {
			return err
		}
	}
	return nil
}

func (m *workerPluginManager) collect(result runResult) ([]pluginData, error) {
	data := make([]pluginData, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		value, err := plugin.getData(result)
		if err != nil {
			return nil, err
		}
		data = append(data, value)
	}
	return data, nil
}

func (m *workerPluginManager) stop() error {
	var firstErr error
	for _, plugin := range m.plugins {
		if err := plugin.stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *workerPluginManager) dispose() error {
	var firstErr error
	for _, plugin := range m.plugins {
		if err := plugin.dispose(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type httpCallbackWorkerPlugin struct {
	pluginName  string
	callbackURL string
	httpClient  *http.Client
}

type inProcessWorkerPluginAdapter struct {
	legacyPlugin      workerPlugin
	legacyTypedPlugin legacyLoadStrikeWorkerPlugin
	asyncPlugin       loadStrikeAsyncWorkerPlugin
	plugin            LoadStrikeWorkerPlugin
	spec              *workerPluginSpec
	startContext      contextState
}

func (p *inProcessWorkerPluginAdapter) name() string {
	if p == nil {
		return "worker-plugin"
	}
	if p.asyncPlugin != nil {
		return firstNonBlank(p.asyncPlugin.PluginName(), "worker-plugin")
	}
	if p.plugin != nil {
		return firstNonBlank(p.plugin.PluginName(), "worker-plugin")
	}
	if p.legacyTypedPlugin != nil {
		return firstNonBlank(p.legacyTypedPlugin.PluginName(), "worker-plugin")
	}
	if p.spec != nil {
		return firstNonBlank(p.spec.PluginName(), "worker-plugin")
	}
	if p.legacyPlugin != nil {
		return firstNonBlank(p.legacyPlugin.Name(), "worker-plugin")
	}
	return "worker-plugin"
}

func (c loadStrikeBaseContext) nativeValue() *contextState {
	return c.native
}

func (p *inProcessWorkerPluginAdapter) init(context contextState, infraConfig map[string]any) error {
	if p == nil {
		return nil
	}
	baseContext := newLoadStrikeBaseContext(&context)
	infra := newIConfiguration(infraConfig)
	if p.asyncPlugin != nil {
		return p.asyncPlugin.InitAsync(baseContext, infra).Await()
	}
	if p.plugin != nil {
		return p.plugin.Init(baseContext, infra).Await()
	}
	if p.legacyTypedPlugin != nil {
		return p.legacyTypedPlugin.Init(baseContext, infra)
	}
	if p.spec != nil {
		p.startContext = context
		return p.spec.Init(baseContext, infra).Await()
	}
	if p.legacyPlugin != nil {
		p.startContext = context
		return p.legacyPlugin.Init(baseContext, infra)
	}
	return nil
}

func (p *inProcessWorkerPluginAdapter) start(sessionInfo LoadStrikeSessionStartInfo) error {
	if p == nil {
		return nil
	}
	if p.asyncPlugin != nil {
		return p.asyncPlugin.StartAsync(sessionInfo).Await()
	}
	if p.plugin != nil {
		return p.plugin.Start(sessionInfo).Await()
	}
	if p.legacyTypedPlugin != nil {
		return p.legacyTypedPlugin.Start(sessionInfo)
	}
	if p.spec != nil {
		if err := p.spec.Start(sessionInfo).Await(); err != nil {
			return err
		}
		if p.spec.StartFunc != nil {
			return p.spec.StartFunc(newLoadStrikeBaseContext(&p.startContext))
		}
		return nil
	}
	if p.legacyPlugin != nil {
		return p.legacyPlugin.Start(newLoadStrikeBaseContext(&p.startContext))
	}
	return nil
}

func (p *inProcessWorkerPluginAdapter) getData(result runResult) (pluginData, error) {
	if p == nil {
		return pluginData{}, nil
	}
	if p.asyncPlugin != nil {
		value, err := p.asyncPlugin.GetDataAsync(newLoadStrikeRunResult(result)).Await()
		if err != nil {
			return pluginData{}, err
		}
		return value.toNative(), nil
	}
	if p.plugin != nil {
		value, err := p.plugin.GetData(newLoadStrikeRunResult(result)).Await()
		if err != nil {
			return pluginData{}, err
		}
		return value.toNative(), nil
	}
	if p.legacyTypedPlugin != nil {
		value, err := p.legacyTypedPlugin.GetData(newLoadStrikeRunResult(result))
		if err != nil {
			return pluginData{}, err
		}
		return value.toNative(), nil
	}
	if p.spec != nil {
		value, err := p.spec.GetData(newLoadStrikeRunResult(result)).Await()
		if err != nil {
			return pluginData{}, err
		}
		return value.toNative(), nil
	}
	if p.legacyPlugin != nil {
		return p.legacyPlugin.GetData(result)
	}
	return pluginData{}, nil
}

func (p *inProcessWorkerPluginAdapter) stop() error {
	if p == nil {
		return nil
	}
	if p.asyncPlugin != nil {
		return p.asyncPlugin.StopAsync().Await()
	}
	if p.plugin != nil {
		return p.plugin.Stop().Await()
	}
	if p.legacyTypedPlugin != nil {
		return p.legacyTypedPlugin.Stop()
	}
	if p.spec != nil {
		return p.spec.Stop().Await()
	}
	if p.legacyPlugin != nil {
		return p.legacyPlugin.Stop()
	}
	return nil
}

func (p *inProcessWorkerPluginAdapter) dispose() error {
	if p == nil {
		return nil
	}
	if p.asyncPlugin != nil {
		if disposer, ok := p.asyncPlugin.(loadStrikeAsyncWorkerPluginDisposer); ok {
			return disposer.DisposeAsync().Await()
		}
		return nil
	}
	if p.plugin != nil {
		return p.plugin.Dispose().Await()
	}
	if p.legacyTypedPlugin != nil {
		return p.legacyTypedPlugin.Dispose()
	}
	if p.spec != nil {
		return p.spec.Dispose().Await()
	}
	if p.legacyPlugin != nil {
		return p.legacyPlugin.Dispose()
	}
	return nil
}

func (p *httpCallbackWorkerPlugin) name() string {
	return p.pluginName
}

func (p *httpCallbackWorkerPlugin) init(context contextState, infraConfig map[string]any) error {
	return p.invoke("init", context, nil, infraConfig)
}

func (p *httpCallbackWorkerPlugin) start(sessionInfo LoadStrikeSessionStartInfo) error {
	payload := map[string]any{
		"stage":       "start",
		"pluginName":  p.pluginName,
		"sessionInfo": sessionInfo,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, p.callbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := p.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("worker plugin %q stage %q failed with status %d", p.pluginName, "start", response.StatusCode)
	}
	return nil
}

func (p *httpCallbackWorkerPlugin) getData(result runResult) (pluginData, error) {
	payload := map[string]any{}
	if err := p.invokeWithResponse("getData", contextState{}, &result, nil, &payload); err != nil {
		return pluginData{}, err
	}

	data := pluginData{
		PluginName: firstNonBlank(asString(payload["PluginName"]), p.pluginName),
		Hints:      normalizeStringList(payload["Hints"]),
		Tables:     []pluginDataTable{},
	}

	for _, table := range reportRecordList(payload, "Tables", "tables") {
		rows := []map[string]any{}
		for _, row := range reportRecordList(table, "Rows", "rows") {
			rows = append(rows, row)
		}
		if rawRows, ok := reportValue(table, "Rows", "rows").([]any); ok {
			rows = rows[:0]
			for _, item := range rawRows {
				if record, ok := item.(map[string]any); ok {
					rows = append(rows, record)
				}
			}
		}
		data.Tables = append(data.Tables, pluginDataTable{
			TableName: asString(reportValue(table, "TableName", "tableName")),
			Rows:      rows,
		})
	}

	return data, nil
}

func (p *httpCallbackWorkerPlugin) stop() error {
	return p.invoke("stop", contextState{}, nil, nil)
}

func (p *httpCallbackWorkerPlugin) dispose() error {
	return p.invoke("dispose", contextState{}, nil, nil)
}

func (p *httpCallbackWorkerPlugin) invoke(stage string, context contextState, result *runResult, infraConfig map[string]any) error {
	return p.invokeWithResponse(stage, context, result, infraConfig, nil)
}

func (p *httpCallbackWorkerPlugin) invokeWithResponse(stage string, context contextState, result *runResult, infraConfig map[string]any, responseTarget any) error {
	payload := map[string]any{
		"stage":      stage,
		"pluginName": p.pluginName,
	}

	if stage == "init" || stage == "start" {
		payload["context"] = map[string]any{
			"TestSuite":   context.TestSuite,
			"TestName":    context.TestName,
			"SessionId":   context.SessionID,
			"ClusterId":   context.ClusterID,
			"AgentGroup":  context.AgentGroup,
			"NodeType":    context.NodeType,
			"AgentsCount": context.AgentsCount,
		}
	}
	if infraConfig != nil {
		payload["infraConfig"] = infraConfig
	}
	if result != nil {
		payload["result"] = result
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, p.callbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("worker plugin %q stage %q failed with status %d", p.pluginName, stage, response.StatusCode)
	}
	if responseTarget == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(responseTarget)
}

type failedResponseBuiltinWorkerPlugin struct{}

func (p *failedResponseBuiltinWorkerPlugin) name() string                            { return "LoadStrike Failed Responses" }
func (p *failedResponseBuiltinWorkerPlugin) init(contextState, map[string]any) error { return nil }
func (p *failedResponseBuiltinWorkerPlugin) start(LoadStrikeSessionStartInfo) error  { return nil }
func (p *failedResponseBuiltinWorkerPlugin) stop() error                             { return nil }
func (p *failedResponseBuiltinWorkerPlugin) dispose() error                          { return nil }

func (p *failedResponseBuiltinWorkerPlugin) getData(result runResult) (pluginData, error) {
	data := pluginData{
		PluginName: p.name(),
		Hints:      []string{},
		Tables:     []pluginDataTable{},
	}

	rows := append([]correlationRow(nil), result.FailedCorrelationRows...)
	if len(rows) == 0 {
		data.Hints = append(data.Hints, "No failed or timed out cross-platform rows were captured.")
		return data, nil
	}

	data.Hints = append(data.Hints, fmt.Sprintf("Captured rows: %d", len(rows)))
	data.Hints = append(data.Hints, "Includes timeout, source_error, and unmatched_destination outcomes.")

	table := pluginDataTable{
		TableName: "Failed and Timed Out Rows",
		Rows:      make([]map[string]any, 0, len(rows)),
	}
	for _, row := range rows {
		table.Rows = append(table.Rows, map[string]any{
			"OccurredUtc":             row.OccurredUTC,
			"scenario":                row.ScenarioName,
			"Source":                  row.Source,
			"Destination":             row.Destination,
			"RunMode":                 row.RunMode,
			"StatusCode":              row.Status,
			"TrackingId":              row.TrackingID,
			"EventId":                 row.EventID,
			"SourceTimestampUtc":      row.SourceTimestampUTC,
			"DestinationTimestampUtc": row.DestinationTimestampUTC,
			"LatencyMs":               row.LatencyMS,
			"Message":                 row.Message,
		})
	}
	data.Tables = append(data.Tables, table)
	return data, nil
}

type correlationBuiltinWorkerPlugin struct{}

func (p *correlationBuiltinWorkerPlugin) name() string                            { return "LoadStrike Correlation" }
func (p *correlationBuiltinWorkerPlugin) init(contextState, map[string]any) error { return nil }
func (p *correlationBuiltinWorkerPlugin) start(LoadStrikeSessionStartInfo) error  { return nil }
func (p *correlationBuiltinWorkerPlugin) stop() error                             { return nil }
func (p *correlationBuiltinWorkerPlugin) dispose() error                          { return nil }

func (p *correlationBuiltinWorkerPlugin) getData(result runResult) (pluginData, error) {
	data := pluginData{
		PluginName: p.name(),
		Hints:      []string{},
		Tables:     []pluginDataTable{},
	}

	rows := append([]correlationRow(nil), result.CorrelationRows...)
	sort.SliceStable(rows, func(left int, right int) bool {
		return rows[left].OccurredUTC > rows[right].OccurredUTC
	})
	if len(rows) == 0 {
		data.Hints = append(data.Hints, "No cross-platform correlation rows were captured.")
		return data, nil
	}

	data.Hints = append(data.Hints, fmt.Sprintf("Captured correlation rows: %d", len(rows)))
	ungrouped := pluginDataTable{
		TableName: "Ungrouped Correlation Rows",
		Rows:      make([]map[string]any, 0, len(rows)),
	}
	for _, row := range rows {
		ungrouped.Rows = append(ungrouped.Rows, map[string]any{
			"OccurredUtc":             row.OccurredUTC,
			"scenario":                row.ScenarioName,
			"Source":                  row.Source,
			"Destination":             row.Destination,
			"RunMode":                 row.RunMode,
			"StatusCode":              row.Status,
			"IsSuccess":               row.IsSuccess,
			"IsFailure":               row.IsFailure,
			"GatherByField":           row.GatherByField,
			"GatherByValue":           row.GatherByValue,
			"TrackingId":              row.TrackingID,
			"EventId":                 row.EventID,
			"SourceTimestampUtc":      row.SourceTimestampUTC,
			"DestinationTimestampUtc": row.DestinationTimestampUTC,
			"LatencyMs":               row.LatencyMS,
			"Message":                 row.Message,
		})
	}
	data.Tables = append(data.Tables, ungrouped)

	groupedRows := buildGroupedCorrelationRows(rows)
	grouped := pluginDataTable{
		TableName: "Grouped Correlation Summary (GatherByField)",
		Rows:      groupedRows,
	}
	if len(groupedRows) == 0 {
		data.Hints = append(data.Hints, "No GatherByField grouping data was captured. Configure Destination.GatherByField to enable grouped summaries.")
	} else {
		data.Hints = append(data.Hints, fmt.Sprintf("GatherBy groups: %d", len(groupedRows)))
	}
	data.Tables = append(data.Tables, grouped)

	return data, nil
}

func buildGroupedCorrelationRows(rows []correlationRow) []map[string]any {
	type groupedSummary struct {
		ScenarioName string
		Destination  string
		Field        string
		Value        string
		Rows         []correlationRow
	}

	index := map[string]*groupedSummary{}
	order := []string{}
	for _, row := range rows {
		if strings.TrimSpace(row.GatherByField) == "" {
			continue
		}
		value := strings.TrimSpace(row.GatherByValue)
		if value == "" {
			value = "<missing>"
		}
		key := strings.Join([]string{row.ScenarioName, row.Destination, row.GatherByField, value}, "\x00")
		if _, exists := index[key]; !exists {
			index[key] = &groupedSummary{
				ScenarioName: row.ScenarioName,
				Destination:  row.Destination,
				Field:        row.GatherByField,
				Value:        value,
				Rows:         []correlationRow{},
			}
			order = append(order, key)
		}
		index[key].Rows = append(index[key].Rows, row)
	}

	result := make([]map[string]any, 0, len(order))
	for _, key := range order {
		group := index[key]
		latencySamples := []float64{}
		matched := 0
		timeouts := 0
		unmatched := 0
		sourceErrors := 0
		sourceSuccess := 0
		success := 0
		failure := 0

		for _, row := range group.Rows {
			if row.IsSuccess {
				success++
			}
			if row.IsFailure {
				failure++
			}
			switch strings.ToLower(strings.TrimSpace(row.Status)) {
			case "matched":
				matched++
			case "timeout":
				timeouts++
			case "unmatched_destination":
				unmatched++
			case "source_error":
				sourceErrors++
			case "source_success":
				sourceSuccess++
			}
			if value := strings.TrimSpace(row.LatencyMS); value != "" {
				latencySamples = append(latencySamples, asDouble(value))
			}
		}

		sort.Float64s(latencySamples)
		total := len(group.Rows)
		knownStatuses := matched + timeouts + unmatched + sourceErrors + sourceSuccess
		row := map[string]any{
			"scenario":             group.ScenarioName,
			"Destination":          group.Destination,
			"GatherByField":        group.Field,
			"GatherByValue":        group.Value,
			"Total":                total,
			"Success":              success,
			"Failure":              failure,
			"SuccessRatePercent":   percentageString(success, total),
			"FailureRatePercent":   percentageString(failure, total),
			"Matched":              matched,
			"Timeout":              timeouts,
			"UnmatchedDestination": unmatched,
			"SourceError":          sourceErrors,
			"SourceSuccess":        sourceSuccess,
			"OtherStatus":          maxInt(total-knownStatuses, 0),
			"LatencySamples":       len(latencySamples),
			"LatencyMinMs":         percentileString(latencySamples, 0.0),
			"LatencyMeanMs":        meanString(latencySamples),
			"LatencyP50Ms":         percentileString(latencySamples, 0.50),
			"LatencyP80Ms":         percentileString(latencySamples, 0.80),
			"LatencyP85Ms":         percentileString(latencySamples, 0.85),
			"LatencyP90Ms":         percentileString(latencySamples, 0.90),
			"LatencyP95Ms":         percentileString(latencySamples, 0.95),
			"LatencyP99Ms":         percentileString(latencySamples, 0.99),
			"LatencyMaxMs":         percentileString(latencySamples, 1.0),
		}
		result = append(result, row)
	}

	sort.SliceStable(result, func(left int, right int) bool {
		return fmt.Sprint(result[left]["scenario"], result[left]["Destination"], result[left]["GatherByField"], result[left]["GatherByValue"]) <
			fmt.Sprint(result[right]["scenario"], result[right]["Destination"], result[right]["GatherByField"], result[right]["GatherByValue"])
	})
	return result
}

func normalizeStringList(value any) []string {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		text := strings.TrimSpace(asString(item))
		if text != "" {
			result = append(result, text)
		}
	}
	return result
}

func percentageString(value int, total int) string {
	if total <= 0 {
		return "0"
	}
	return formatReportNumber((float64(value) * 100.0) / float64(total))
}

func percentileString(samples []float64, percentile float64) string {
	if len(samples) == 0 {
		return ""
	}
	index := int(percentile * float64(len(samples)))
	if percentile > 0 {
		index--
	}
	if index < 0 {
		index = 0
	}
	if index >= len(samples) {
		index = len(samples) - 1
	}
	return formatReportNumber(samples[index])
}

func meanString(samples []float64) string {
	if len(samples) == 0 {
		return ""
	}
	total := 0.0
	for _, sample := range samples {
		total += sample
	}
	return formatReportNumber(total / float64(len(samples)))
}
