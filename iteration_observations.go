package loadstrike

import "encoding/json"

// LoadStrikeIterationStepObservation is one compact step outcome nested under
// the scenario attempt that produced it.
type LoadStrikeIterationStepObservation struct {
	StepName            string `json:"stepName"`
	SortIndex           int    `json:"sortIndex"`
	StartedUTCNs        string `json:"startedUtcNs"`
	CompletedUTCNs      string `json:"completedUtcNs"`
	ObservedLatencyUS64 string `json:"observedLatencyUs64"`
	ReportedLatencyUS64 string `json:"reportedLatencyUs64"`
	IsSuccess           bool   `json:"isSuccess"`
	StatusCode          string `json:"statusCode"`
	SizeBytes64         string `json:"sizeBytes64"`
}

// LoadStrikeIterationObservation is the canonical raw scenario-attempt value.
// Messages, payloads, headers, and exception details are deliberately absent.
type LoadStrikeIterationObservation struct {
	SchemaVersion            string                               `json:"schemaVersion"`
	ObservationID            string                               `json:"observationId"`
	RunID                    string                               `json:"runId"`
	SessionID                string                               `json:"sessionId"`
	ResultOwnerID            string                               `json:"resultOwnerId"`
	ProcessGroup             int                                  `json:"processGroup"`
	ScenarioName             string                               `json:"scenarioName"`
	ScenarioIndex            int                                  `json:"scenarioIndex"`
	SimulationIndex          int                                  `json:"simulationIndex"`
	SimulationKind           string                               `json:"simulationKind"`
	Phase                    string                               `json:"phase"`
	GlobalOrdinal64          string                               `json:"globalOrdinal64"`
	GlobalSecondaryOrdinal64 string                               `json:"globalSecondaryOrdinal64"`
	ShardIndex               int                                  `json:"shardIndex"`
	ShardCount               int                                  `json:"shardCount"`
	IterationID              string                               `json:"iterationId"`
	AttemptIndex             int                                  `json:"attemptIndex"`
	IsFinalAttempt           bool                                 `json:"isFinalAttempt"`
	StartedUTCNs             string                               `json:"startedUtcNs"`
	CompletedUTCNs           string                               `json:"completedUtcNs"`
	ObservedLatencyUS64      string                               `json:"observedLatencyUs64"`
	ReportedLatencyUS64      string                               `json:"reportedLatencyUs64"`
	IsSuccess                bool                                 `json:"isSuccess"`
	StatusCode               string                               `json:"statusCode"`
	SizeBytes64              string                               `json:"sizeBytes64"`
	Steps                    []LoadStrikeIterationStepObservation `json:"steps"`
}

// MarshalJSON preserves the canonical wire defaults for observations created
// by older callers that do not set the V2 secondary ordinal explicitly.
func (observation LoadStrikeIterationObservation) MarshalJSON() ([]byte, error) {
	type observationAlias LoadStrikeIterationObservation
	normalized := observationAlias(observation)
	if normalized.GlobalSecondaryOrdinal64 == "" {
		normalized.GlobalSecondaryOrdinal64 = "0"
	}
	if normalized.Steps == nil {
		normalized.Steps = []LoadStrikeIterationStepObservation{}
	}
	return json.Marshal(normalized)
}

// LoadStrikeIterationObservationBatch is the canonical sink delivery unit.
type LoadStrikeIterationObservationBatch struct {
	SchemaVersion         string                           `json:"schemaVersion"`
	RunID                 string                           `json:"runId"`
	SessionID             string                           `json:"sessionId"`
	ResultOwnerID         string                           `json:"resultOwnerId"`
	ProcessGroup          int                              `json:"processGroup"`
	BatchID               string                           `json:"batchId"`
	BatchSequence64       string                           `json:"batchSequence64"`
	CreatedUTCNs          string                           `json:"createdUtcNs"`
	FirstObservationUTCNs string                           `json:"firstObservationUtcNs"`
	LastObservationUTCNs  string                           `json:"lastObservationUtcNs"`
	Observations          []LoadStrikeIterationObservation `json:"observations"`
	CapturedCount64       string                           `json:"capturedCount64"`
	DroppedBeforeBatch64  string                           `json:"droppedBeforeBatch64"`
	Compression           string                           `json:"compression"`
}

// LoadStrikeIterationStreamCompletion seals one result owner's stream before
// the legacy final run result callback.
type LoadStrikeIterationStreamCompletion struct {
	SchemaVersion              string `json:"schemaVersion"`
	RunID                      string `json:"runId"`
	SessionID                  string `json:"sessionId"`
	ResultOwnerID              string `json:"resultOwnerId"`
	ProcessGroup               int    `json:"processGroup"`
	ExpectedResultOwnerCount64 string `json:"expectedResultOwnerCount64"`
	LastBatchSequence64        string `json:"lastBatchSequence64"`
	CapturedCount64            string `json:"capturedCount64"`
	DeliveredCount64           string `json:"deliveredCount64"`
	DroppedBufferCount64       string `json:"droppedBufferCount64"`
	DroppedSinkCount64         string `json:"droppedSinkCount64"`
	ReportingComplete          bool   `json:"reportingComplete"`
	CompletedUTCNs             string `json:"completedUtcNs"`
}

// LoadStrikeIterationBatchSink is the optional secondary reporting capability
// for receiving raw attempt batches. Existing LoadStrikeReportingSink
// implementations remain source compatible without implementing it.
type LoadStrikeIterationBatchSink interface {
	SaveIterationBatch(LoadStrikeIterationObservationBatch) LoadStrikeTask
	CompleteIterationStream(LoadStrikeIterationStreamCompletion) LoadStrikeTask
}

// LoadStrikeGeneratorWarning is a structured generator/reporting warning.
type LoadStrikeGeneratorWarning struct {
	Code               string `json:"code"`
	Message            string `json:"message,omitempty"`
	SinkName           string `json:"sinkName,omitempty"`
	ScenarioName       string `json:"scenarioName,omitempty"`
	ScenarioIndex      int    `json:"scenarioIndex"`
	SimulationIndex    int    `json:"simulationIndex"`
	SimulationKind     string `json:"simulationKind,omitempty"`
	Count64            string `json:"count64"`
	FirstObservedUTCNs string `json:"firstObservedUtcNs"`
	LastObservedUTCNs  string `json:"lastObservedUtcNs"`
}

// LoadStrikeObservationSinkDeliveryStats contains delivery totals for one sink.
type LoadStrikeObservationSinkDeliveryStats struct {
	SinkName         string `json:"sinkName"`
	DeliveredCount64 string `json:"deliveredCount64"`
	DroppedCount64   string `json:"droppedCount64"`
	FailedCount64    string `json:"failedCount64"`
}

// LoadStrikeObservationDeliveryStats contains process-level observation totals.
type LoadStrikeObservationDeliveryStats struct {
	LastBatchSequence64  string                                   `json:"lastBatchSequence64"`
	CapturedCount64      string                                   `json:"capturedCount64"`
	DeliveredCount64     string                                   `json:"deliveredCount64"`
	DroppedBufferCount64 string                                   `json:"droppedBufferCount64"`
	DroppedSinkCount64   string                                   `json:"droppedSinkCount64"`
	Sinks                []LoadStrikeObservationSinkDeliveryStats `json:"sinks,omitempty"`
}

// SinkDroppedCount64 returns one sink's dropped count, or zero when absent.
func (s LoadStrikeObservationDeliveryStats) SinkDroppedCount64(sinkName string) string {
	for _, sink := range s.Sinks {
		if sink.SinkName == sinkName {
			return sink.DroppedCount64
		}
	}
	return "0"
}
