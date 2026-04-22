package loadstrike

import "time"

// LoadStrikeTestInfo mirrors the .NET test-info contract name.
type LoadStrikeTestInfo struct {
	ClusterId string    `json:"ClusterId,omitempty"`
	Created   time.Time `json:"Created,omitempty"`
	SessionId string    `json:"SessionId,omitempty"`
	TestName  string    `json:"TestName,omitempty"`
	TestSuite string    `json:"TestSuite,omitempty"`
}

func newLoadStrikeTestInfo(native testInfo) LoadStrikeTestInfo {
	native = normalizeTestInfo(native)
	return LoadStrikeTestInfo{
		ClusterId: native.ClusterId,
		Created:   native.Created,
		SessionId: native.SessionId,
		TestName:  native.TestName,
		TestSuite: native.TestSuite,
	}
}

func (i LoadStrikeTestInfo) toNative() testInfo {
	return normalizeTestInfo(testInfo{
		ClusterId: i.ClusterId,
		Created:   i.Created,
		SessionId: i.SessionId,
		TestName:  i.TestName,
		TestSuite: i.TestSuite,
	})
}

// LoadStrikeNodeInfo mirrors the .NET node-info contract name.
type LoadStrikeNodeInfo struct {
	CoresCount       int                     `json:"CoresCount,omitempty"`
	CurrentOperation LoadStrikeOperationType `json:"CurrentOperation,omitempty"`
	DotNetVersion    string                  `json:"DotNetVersion,omitempty"`
	MachineName      string                  `json:"MachineName,omitempty"`
	EngineVersion    string                  `json:"EngineVersion,omitempty"`
	NodeType         LoadStrikeNodeType      `json:"NodeType,omitempty"`
	OS               string                  `json:"OS,omitempty"`
	Processor        string                  `json:"Processor,omitempty"`
}

func newLoadStrikeNodeInfo(native nodeInfo) LoadStrikeNodeInfo {
	return LoadStrikeNodeInfo{
		CoresCount:       native.CoresCount,
		CurrentOperation: nodeOperationType(native),
		DotNetVersion:    native.DotNetVersion,
		MachineName:      native.MachineName,
		EngineVersion:    native.EngineVersion,
		NodeType:         native.NodeType,
		OS:               native.OS,
		Processor:        native.Processor,
	}
}

func (i LoadStrikeNodeInfo) toNative() nodeInfo {
	return nodeInfo{
		CoresCount:           i.CoresCount,
		CurrentOperation:     operationTypeName(i.CurrentOperation),
		CurrentOperationType: i.CurrentOperation,
		DotNetVersion:        i.DotNetVersion,
		MachineName:          i.MachineName,
		EngineVersion:        i.EngineVersion,
		NodeType:             i.NodeType,
		OS:                   i.OS,
		Processor:            i.Processor,
	}
}

// LoadStrikeMetricValue mirrors the .NET metric-value contract name.
type LoadStrikeMetricValue struct {
	Name          string  `json:"Name,omitempty"`
	ScenarioName  string  `json:"ScenarioName,omitempty"`
	Type          string  `json:"Type,omitempty"`
	UnitOfMeasure string  `json:"UnitOfMeasure,omitempty"`
	Value         float64 `json:"Value,omitempty"`
}

func newLoadStrikeMetricValue(native metricResult) LoadStrikeMetricValue {
	return LoadStrikeMetricValue{
		Name:          native.Name,
		ScenarioName:  native.ScenarioName,
		Type:          native.Type,
		UnitOfMeasure: native.UnitOfMeasure,
		Value:         native.Value,
	}
}

func (m LoadStrikeMetricValue) toNative() metricResult {
	return metricResult{
		Name:          m.Name,
		ScenarioName:  m.ScenarioName,
		Type:          m.Type,
		UnitOfMeasure: m.UnitOfMeasure,
		Value:         m.Value,
	}
}

func nodeOperationType(native nodeInfo) LoadStrikeOperationType {
	if native.CurrentOperationType != LoadStrikeOperationTypeNone {
		return native.CurrentOperationType
	}
	return parseOperationType(native.CurrentOperation)
}
