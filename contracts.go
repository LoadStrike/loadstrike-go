package loadstrike

import "encoding/json"

// runRequest is the contract used by the parity harness and Go runtime entrypoints.
type runRequest struct {
	Context   runContext     `json:"Context"`
	Scenarios []scenarioSpec `json:"Scenarios"`
	RunArgs   []string       `json:"RunArgs"`
}

// runContext carries execution-wide settings for a run.
type runContext struct {
	ConsoleMetricsEnabled bool                `json:"ConsoleMetricsEnabled,omitempty"`
	ReportsEnabled        bool                `json:"ReportsEnabled,omitempty"`
	ReportFormats         []string            `json:"ReportFormats,omitempty"`
	ReportingSinks        []reportingSinkSpec `json:"ReportingSinks,omitempty"`
	TestSuite             string              `json:"TestSuite,omitempty"`
	TestName              string              `json:"TestName,omitempty"`
	WorkerPlugins         []workerPluginSpec  `json:"WorkerPlugins,omitempty"`
}

// scenarioSpec defines a scenario in the public contract model.
type scenarioSpec struct {
	Name            string                     `json:"Name"`
	WithoutWarmUp   bool                       `json:"WithoutWarmUp,omitempty"`
	LoadSimulations []loadSimulationSpec       `json:"LoadSimulations,omitempty"`
	Thresholds      []ThresholdSpec            `json:"Thresholds,omitempty"`
	Tracking        *TrackingConfigurationSpec `json:"Tracking,omitempty"`
}

// loadSimulationSpec defines the public load-shape contract.
type loadSimulationSpec struct {
	Kind       string `json:"Kind"`
	Copies     int    `json:"Copies,omitempty"`
	Iterations int    `json:"Iterations,omitempty"`
}

// RawPayload preserves arbitrary JSON payloads used by transport contracts.
type RawPayload = json.RawMessage
