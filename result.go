package loadstrike

type runSummary struct {
	AllRequestCount int                  `json:"allRequestCount"`
	AllOKCount      int                  `json:"allOkCount"`
	AllFailCount    int                  `json:"allFailCount"`
	ScenarioStats   []scenarioRunSummary `json:"scenarioStats,omitempty"`
}

type scenarioRunSummary struct {
	ScenarioName    string `json:"scenarioName"`
	AllRequestCount int    `json:"allRequestCount"`
	AllOKCount      int    `json:"allOkCount"`
	AllFailCount    int    `json:"allFailCount"`
}

// LoadStrikeRunResult exposes the primary aggregate counters used by the docs samples.
type LoadStrikeRunResult struct {
	native runSummary
}

func newLoadStrikeRunResult(native runSummary) LoadStrikeRunResult {
	return LoadStrikeRunResult{native: native}
}

func (r LoadStrikeRunResult) AllRequestCount() int {
	return r.native.AllRequestCount
}

func (r LoadStrikeRunResult) AllOKCount() int {
	return r.native.AllOKCount
}

func (r LoadStrikeRunResult) AllFailCount() int {
	return r.native.AllFailCount
}

func (r LoadStrikeRunResult) ScenarioStats() []LoadStrikeScenarioRunStats {
	stats := make([]LoadStrikeScenarioRunStats, 0, len(r.native.ScenarioStats))
	for _, item := range r.native.ScenarioStats {
		stats = append(stats, LoadStrikeScenarioRunStats{native: item})
	}
	return stats
}

type LoadStrikeScenarioRunStats struct {
	native scenarioRunSummary
}

func (s LoadStrikeScenarioRunStats) ScenarioName() string {
	return s.native.ScenarioName
}

func (s LoadStrikeScenarioRunStats) AllRequestCount() int {
	return s.native.AllRequestCount
}

func (s LoadStrikeScenarioRunStats) AllOKCount() int {
	return s.native.AllOKCount
}

func (s LoadStrikeScenarioRunStats) AllFailCount() int {
	return s.native.AllFailCount
}
