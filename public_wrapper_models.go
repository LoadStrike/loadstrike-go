package loadstrike

// LoadStrikeRequestStats mirrors the .NET request-stats wrapper contract.
type LoadStrikeRequestStats struct {
	Count   int     `json:"Count,omitempty"`
	Percent int     `json:"Percent,omitempty"`
	RPS     float64 `json:"RPS,omitempty"`
}

func newLoadStrikeRequestStats(native requestStats) LoadStrikeRequestStats {
	return LoadStrikeRequestStats{
		Count:   native.Count,
		Percent: native.Percent,
		RPS:     native.RPS,
	}
}

func (s LoadStrikeRequestStats) toNative() requestStats {
	return requestStats{
		Count:   s.Count,
		Percent: s.Percent,
		RPS:     s.RPS,
	}
}

// LoadStrikeDataTransferStats mirrors the .NET data-transfer-stats wrapper contract.
type LoadStrikeDataTransferStats struct {
	AllBytes  int64   `json:"AllBytes,omitempty"`
	MinBytes  int64   `json:"MinBytes,omitempty"`
	MaxBytes  int64   `json:"MaxBytes,omitempty"`
	MeanBytes int64   `json:"MeanBytes,omitempty"`
	Percent50 int64   `json:"Percent50,omitempty"`
	Percent75 int64   `json:"Percent75,omitempty"`
	Percent95 int64   `json:"Percent95,omitempty"`
	Percent99 int64   `json:"Percent99,omitempty"`
	StdDev    float64 `json:"StdDev,omitempty"`
}

func newLoadStrikeDataTransferStats(native dataTransferStats) LoadStrikeDataTransferStats {
	return LoadStrikeDataTransferStats{
		AllBytes:  native.AllBytes,
		MinBytes:  native.MinBytes,
		MaxBytes:  native.MaxBytes,
		MeanBytes: native.MeanBytes,
		Percent50: native.Percent50,
		Percent75: native.Percent75,
		Percent95: native.Percent95,
		Percent99: native.Percent99,
		StdDev:    native.StdDev,
	}
}

func (s LoadStrikeDataTransferStats) toNative() dataTransferStats {
	return dataTransferStats{
		AllBytes:  s.AllBytes,
		MinBytes:  s.MinBytes,
		MaxBytes:  s.MaxBytes,
		MeanBytes: s.MeanBytes,
		Percent50: s.Percent50,
		Percent75: s.Percent75,
		Percent95: s.Percent95,
		Percent99: s.Percent99,
		StdDev:    s.StdDev,
	}
}

// LoadStrikeStatusCodeStats mirrors the .NET status-code-stats wrapper contract.
type LoadStrikeStatusCodeStats struct {
	StatusCode string `json:"StatusCode,omitempty"`
	Message    string `json:"Message,omitempty"`
	IsError    bool   `json:"IsError,omitempty"`
	Count      int    `json:"Count,omitempty"`
	Percent    int    `json:"Percent,omitempty"`
}

func newLoadStrikeStatusCodeStats(native statusCodeStats) LoadStrikeStatusCodeStats {
	return LoadStrikeStatusCodeStats{
		StatusCode: native.StatusCode,
		Message:    native.Message,
		IsError:    native.IsError,
		Count:      native.Count,
		Percent:    native.Percent,
	}
}

func (s LoadStrikeStatusCodeStats) toNative() statusCodeStats {
	return statusCodeStats{
		StatusCode: s.StatusCode,
		Message:    s.Message,
		IsError:    s.IsError,
		Count:      s.Count,
		Percent:    s.Percent,
	}
}

// LoadStrikeLoadSimulationStats mirrors the .NET load-simulation-stats wrapper contract.
type LoadStrikeLoadSimulationStats struct {
	SimulationName string `json:"SimulationName,omitempty"`
	Value          int    `json:"Value,omitempty"`
}

func newLoadStrikeLoadSimulationStats(native loadSimulationStats) LoadStrikeLoadSimulationStats {
	return LoadStrikeLoadSimulationStats{
		SimulationName: native.SimulationName,
		Value:          native.Value,
	}
}

func (s LoadStrikeLoadSimulationStats) toNative() loadSimulationStats {
	return loadSimulationStats{
		SimulationName: s.SimulationName,
		Value:          s.Value,
	}
}

// LoadStrikeThresholdResult mirrors the .NET threshold-result wrapper contract.
type LoadStrikeThresholdResult struct {
	ScenarioName     string `json:"ScenarioName,omitempty"`
	StepName         string `json:"StepName,omitempty"`
	CheckExpression  string `json:"CheckExpression,omitempty"`
	ErrorCount       int    `json:"ErrorCount,omitempty"`
	IsFailed         bool   `json:"IsFailed,omitempty"`
	ExceptionMessage string `json:"ExceptionMessage,omitempty"`
}

func newLoadStrikeThresholdResult(native thresholdResult) LoadStrikeThresholdResult {
	return LoadStrikeThresholdResult{
		ScenarioName:     native.ScenarioName,
		StepName:         native.StepName,
		CheckExpression:  native.CheckExpression,
		ErrorCount:       native.ErrorCount,
		IsFailed:         native.IsFailed,
		ExceptionMessage: native.ExceptionMessage,
	}
}

func (r LoadStrikeThresholdResult) toNative() thresholdResult {
	return thresholdResult{
		ScenarioName:     r.ScenarioName,
		StepName:         r.StepName,
		CheckExpression:  r.CheckExpression,
		ErrorCount:       r.ErrorCount,
		IsFailed:         r.IsFailed,
		ExceptionMessage: r.ExceptionMessage,
	}
}

// LoadStrikeRuntimePolicyError mirrors the .NET runtime-policy-error wrapper contract.
type LoadStrikeRuntimePolicyError struct {
	PolicyName   string `json:"PolicyName,omitempty"`
	CallbackName string `json:"CallbackName,omitempty"`
	ScenarioName string `json:"ScenarioName,omitempty"`
	StepName     string `json:"StepName,omitempty"`
	Message      string `json:"Message,omitempty"`
}

func newLoadStrikeRuntimePolicyError(native RuntimePolicyError) LoadStrikeRuntimePolicyError {
	return LoadStrikeRuntimePolicyError{
		PolicyName:   native.PolicyName,
		CallbackName: native.CallbackName,
		ScenarioName: native.ScenarioName,
		StepName:     native.StepName,
		Message:      native.Message,
	}
}

func (e LoadStrikeRuntimePolicyError) toNative() RuntimePolicyError {
	return RuntimePolicyError{
		PolicyName:   e.PolicyName,
		CallbackName: e.CallbackName,
		ScenarioName: e.ScenarioName,
		StepName:     e.StepName,
		Message:      e.Message,
	}
}

// LoadStrikeSinkError mirrors the .NET sink-error wrapper contract.
type LoadStrikeSinkError struct {
	SinkName string `json:"SinkName,omitempty"`
	Phase    string `json:"Phase,omitempty"`
	Message  string `json:"Message,omitempty"`
	Attempts int    `json:"Attempts,omitempty"`
}

func newLoadStrikeSinkError(native sinkErrorResult) LoadStrikeSinkError {
	return LoadStrikeSinkError{
		SinkName: native.SinkName,
		Phase:    native.Phase,
		Message:  native.Message,
		Attempts: native.Attempts,
	}
}

func (e LoadStrikeSinkError) toNative() sinkErrorResult {
	return sinkErrorResult{
		SinkName: e.SinkName,
		Phase:    e.Phase,
		Message:  e.Message,
		Attempts: e.Attempts,
	}
}

// LoadStrikePluginDataTable mirrors the .NET plugin-table wrapper contract.
type LoadStrikePluginDataTable struct {
	TableName string           `json:"TableName"`
	Rows      []map[string]any `json:"Rows,omitempty"`
}

func newLoadStrikePluginDataTable(native pluginDataTable) LoadStrikePluginDataTable {
	rows := make([]map[string]any, 0, len(native.Rows))
	for _, row := range native.Rows {
		rows = append(rows, cloneAnyMap(row))
	}
	return LoadStrikePluginDataTable{
		TableName: native.TableName,
		Rows:      rows,
	}
}

func (t LoadStrikePluginDataTable) toNative() pluginDataTable {
	rows := make([]map[string]any, 0, len(t.Rows))
	for _, row := range t.Rows {
		rows = append(rows, cloneAnyMap(row))
	}
	return pluginDataTable{
		TableName: t.TableName,
		Rows:      rows,
	}
}

// LoadStrikePluginData mirrors the .NET plugin-data wrapper contract.
type LoadStrikePluginData struct {
	PluginName string                      `json:"PluginName"`
	Hints      []string                    `json:"Hints,omitempty"`
	Tables     []LoadStrikePluginDataTable `json:"Tables,omitempty"`
}

func newLoadStrikePluginData(native pluginData) LoadStrikePluginData {
	tables := make([]LoadStrikePluginDataTable, 0, len(native.Tables))
	for _, table := range native.Tables {
		tables = append(tables, newLoadStrikePluginDataTable(table))
	}
	return LoadStrikePluginData{
		PluginName: native.PluginName,
		Hints:      append([]string(nil), native.Hints...),
		Tables:     tables,
	}
}

func (d LoadStrikePluginData) toNative() pluginData {
	tables := make([]pluginDataTable, 0, len(d.Tables))
	for _, table := range d.Tables {
		tables = append(tables, table.toNative())
	}
	return pluginData{
		PluginName: d.PluginName,
		Hints:      append([]string(nil), d.Hints...),
		Tables:     tables,
	}
}

func toLoadStrikePluginDataSlice(items []pluginData) []LoadStrikePluginData {
	if len(items) == 0 {
		return nil
	}
	result := make([]LoadStrikePluginData, 0, len(items))
	for _, item := range items {
		result = append(result, newLoadStrikePluginData(item))
	}
	return result
}

func toLoadStrikeThresholdResultSlice(items []thresholdResult) []LoadStrikeThresholdResult {
	if len(items) == 0 {
		return nil
	}
	result := make([]LoadStrikeThresholdResult, 0, len(items))
	for _, item := range items {
		result = append(result, newLoadStrikeThresholdResult(item))
	}
	return result
}

func cloneAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
