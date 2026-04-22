package loadstrike

import "time"

// localClient executes request-contract payloads directly inside the local process.
type localClient struct{}

// newLocalClient creates a local runtime client.
func newLocalClient() *localClient {
	return &localClient{}
}

// Run executes a request-contract payload and returns a response-contract payload.
func (c *localClient) Run(request runRequest) (runResponse, error) {
	response := runResponse{
		CompletedUTC: time.Now().UTC(),
		Stats: nodeStats{
			testInfo: normalizeTestInfo(testInfo{
				TestSuite: request.Context.TestSuite,
				TestName:  request.Context.TestName,
			}),
			Scenarios:  make([]scenarioStats, 0, len(request.Scenarios)),
			Thresholds: []thresholdResult{},
		},
	}

	for _, scenario := range request.Scenarios {
		requestCount := scenarioSpecIterations(scenario)
		stats := scenarioStats{
			ScenarioName:    scenario.Name,
			AllRequestCount: requestCount,
			AllOKCount:      requestCount,
			AllFailCount:    0,
		}

		response.Stats.AllRequestCount += requestCount
		response.Stats.AllOKCount += requestCount
		response.Stats.Scenarios = append(response.Stats.Scenarios, stats)
	}

	return response, nil
}

func scenarioSpecIterations(scenario scenarioSpec) int {
	if len(scenario.LoadSimulations) == 0 {
		return 1
	}

	total := 0
	for _, simulation := range scenario.LoadSimulations {
		if simulation.Kind == "IterationsForConstant" {
			total += simulation.Iterations
			continue
		}

		copies := simulation.Copies
		if copies <= 0 {
			copies = 1
		}

		iterations := simulation.Iterations
		if iterations <= 0 {
			iterations = 1
		}

		total += copies * iterations
	}

	if total <= 0 {
		return 1
	}

	return total
}
