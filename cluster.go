package loadstrike

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type clusterRunCommand struct {
	CommandID       string   `json:"commandId"`
	SessionID       string   `json:"sessionId"`
	TestSuite       string   `json:"testSuite"`
	TestName        string   `json:"testName"`
	AgentIndex      int      `json:"agentIndex"`
	AgentCount      int      `json:"agentCount"`
	TargetScenarios []string `json:"targetScenarios"`
	ReplySubject    string   `json:"replySubject"`
}

type clusterRunResult struct {
	CommandID    string    `json:"commandId"`
	AgentID      string    `json:"agentId"`
	IsSuccess    bool      `json:"isSuccess"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Result       runResult `json:"result"`
}

func executeClustered(context contextState, scenarios []scenarioDefinition) (runResult, error) {
	nc, err := nats.Connect(strings.TrimSpace(context.NatsServerURL), nats.Name("loadstrike-go-cluster-coordinator"))
	if err != nil {
		return runResult{}, err
	}
	defer nc.Close()

	agentCount := maxInt(context.AgentsCount, 1)
	runSubject := clusterRunSubject(context)
	replySubject := clusterReplySubject(context)

	var localAgentsWG sync.WaitGroup
	agentReady := make([]chan struct{}, agentCount)
	if context.LocalDevClusterEnabled {
		for agentIndex := 0; agentIndex < agentCount; agentIndex++ {
			agentReady[agentIndex] = make(chan struct{})
			localAgentsWG.Add(1)
			go func(index int, ready chan struct{}) {
				defer localAgentsWG.Done()
				agentRunSubject := runSubject
				if context.LocalDevClusterEnabled {
					agentRunSubject = clusterRunSubjectForAgent(context, index)
				}
				_ = runLocalClusterAgent(context, scenarios, agentRunSubject, index, agentCount, ready)
			}(agentIndex, agentReady[agentIndex])
		}
		for _, ready := range agentReady {
			if ready == nil {
				continue
			}
			select {
			case <-ready:
			case <-time.After(clusterCommandTimeout(context)):
				return runResult{}, fmt.Errorf("cluster agent subscription readiness timed out")
			}
		}
	}

	subscription, err := nc.SubscribeSync(replySubject)
	if err != nil {
		return runResult{}, err
	}
	defer func() {
		_ = subscription.Unsubscribe()
	}()
	if err := nc.Flush(); err != nil {
		return runResult{}, err
	}

	assignments := planAgentScenarioAssignments(scenarios, agentCount, context)
	for agentIndex, scenarioNames := range assignments {
		targetRunSubject := runSubject
		if context.LocalDevClusterEnabled {
			targetRunSubject = clusterRunSubjectForAgent(context, agentIndex)
		}
		command := clusterRunCommand{
			CommandID:       fmt.Sprintf("cmd-%d-%d", time.Now().UnixNano(), agentIndex),
			SessionID:       firstNonBlank(context.SessionID, generateSessionID()),
			TestSuite:       context.TestSuite,
			TestName:        context.TestName,
			AgentIndex:      agentIndex,
			AgentCount:      agentCount,
			TargetScenarios: scenarioNames,
			ReplySubject:    replySubject,
		}
		payload, err := json.Marshal(command)
		if err != nil {
			return runResult{}, err
		}
		if err := nc.Publish(targetRunSubject, payload); err != nil {
			return runResult{}, err
		}
	}
	if err := nc.Flush(); err != nil {
		return runResult{}, err
	}

	timeout := clusterCommandTimeout(context)
	results := make([]clusterRunResult, 0, agentCount)
	for len(results) < agentCount {
		message, err := subscription.NextMsg(timeout)
		if err != nil {
			return runResult{}, err
		}
		var result clusterRunResult
		if err := json.Unmarshal(message.Data, &result); err != nil {
			return runResult{}, err
		}
		results = append(results, result)
	}

	localAgentsWG.Wait()
	return mergeClusterResults(results), nil
}

func runLocalClusterAgent(baseContext contextState, scenarios []scenarioDefinition, runSubject string, agentIndex int, agentCount int, ready chan<- struct{}) error {
	nc, err := nats.Connect(strings.TrimSpace(baseContext.NatsServerURL), nats.Name(fmt.Sprintf("loadstrike-go-agent-%d", agentIndex)))
	if err != nil {
		return err
	}
	defer nc.Close()

	queueGroup := clusterQueueGroup(baseContext)
	subscription, err := nc.QueueSubscribeSync(runSubject, queueGroup)
	if err != nil {
		return err
	}
	defer func() {
		_ = subscription.Unsubscribe()
	}()
	if err := nc.Flush(); err != nil {
		return err
	}
	if ready != nil {
		close(ready)
	}

	message, err := subscription.NextMsg(clusterCommandTimeout(baseContext))
	if err != nil {
		return err
	}

	var command clusterRunCommand
	if err := json.Unmarshal(message.Data, &command); err != nil {
		return err
	}

	agentContext := baseContext
	agentContext.NodeType = NodeTypeAgent
	agentContext.agentIndex = agentIndex
	agentContext.agentCount = agentCount
	agentContext.TargetScenarios = append([]string(nil), command.TargetScenarios...)
	agentContext.ReportsEnabled = false

	filtered := []scenarioDefinition{}
	if len(command.TargetScenarios) > 0 {
		filtered = filterScenariosByName(scenarios, command.TargetScenarios)
	}
	result, executeErr := execute(agentContext, filtered)
	runResult := clusterRunResult{
		CommandID: command.CommandID,
		AgentID:   fmt.Sprintf("agent-%d", agentIndex),
		IsSuccess: executeErr == nil,
		Result:    result,
	}
	if executeErr != nil {
		runResult.ErrorMessage = executeErr.Error()
	}

	payload, err := json.Marshal(runResult)
	if err != nil {
		return err
	}
	return nc.Publish(command.ReplySubject, payload)
}

func mergeClusterResults(results []clusterRunResult) runResult {
	merged := runResult{
		StartedUTC:            time.Time{},
		CompletedUTC:          time.Time{},
		scenarioStats:         []scenarioStats{},
		stepStats:             []stepStats{},
		ScenarioDurationsMS:   map[string]float64{},
		DisabledSinks:         []string{},
		SinkErrors:            []sinkErrorResult{},
		Metrics:               []metricResult{},
		metricStats:           metricStats{},
		Thresholds:            []thresholdResult{},
		ThresholdResults:      []thresholdResult{},
		CorrelationRows:       []correlationRow{},
		FailedCorrelationRows: []correlationRow{},
		PluginsData:           []pluginData{},
		LogFiles:              []string{},
		PolicyErrors:          []RuntimePolicyError{},
		reportTrace:           newReportTrace(),
	}
	nodeResults := make([]runResult, 0, len(results))

	for _, nodeResult := range results {
		if !nodeResult.IsSuccess {
			merged.AllFailCount += 1
			continue
		}
		result := nodeResult.Result
		nodeResults = append(nodeResults, result)
		if merged.StartedUTC.IsZero() || (!result.StartedUTC.IsZero() && result.StartedUTC.Before(merged.StartedUTC)) {
			merged.StartedUTC = result.StartedUTC
		}
		if merged.CompletedUTC.IsZero() || result.CompletedUTC.After(merged.CompletedUTC) {
			merged.CompletedUTC = result.CompletedUTC
		}
		if merged.testInfo.TestSuite == "" {
			merged.testInfo = result.testInfo
		}
		if merged.nodeInfo.MachineName == "" {
			merged.nodeInfo = result.nodeInfo
		}
		merged.AllRequestCount += result.AllRequestCount
		merged.AllOKCount += result.AllOKCount
		merged.AllFailCount += result.AllFailCount
		merged.AllBytes += result.AllBytes
		merged.DisabledSinks = append(merged.DisabledSinks, result.DisabledSinks...)
		merged.SinkErrors = append(merged.SinkErrors, result.SinkErrors...)
		merged.Metrics = append(merged.Metrics, result.Metrics...)
		merged.CorrelationRows = append(merged.CorrelationRows, result.CorrelationRows...)
		merged.FailedCorrelationRows = append(merged.FailedCorrelationRows, result.FailedCorrelationRows...)
		merged.PolicyErrors = append(merged.PolicyErrors, result.PolicyErrors...)
		merged.LogFiles = append(merged.LogFiles, result.LogFiles...)
		merged.reportTrace.merge(result.reportTrace)
	}

	if len(nodeResults) > 0 {
		merged.scenarioStats = aggregateClusterScenarioStats(nodeResults)
		merged.stepStats = flattenClusterStepStats(merged.scenarioStats)
		merged.Thresholds = aggregateClusterThresholds(nodeResults)
		merged.ThresholdResults = append([]thresholdResult(nil), merged.Thresholds...)
		merged.FailedThresholds = 0
		for _, threshold := range merged.Thresholds {
			if threshold.IsFailed {
				merged.FailedThresholds++
			}
		}
		merged.metricStats = aggregateClusterMetricStats(nodeResults)
		merged.PluginsData = aggregateClusterPlugins(nodeResults)
		for _, scenario := range merged.scenarioStats {
			merged.ScenarioDurationsMS[scenario.ScenarioName] = scenario.DurationMS
		}
	}
	merged.LogFiles = uniqueSortedStrings(merged.LogFiles)
	merged.DisabledSinks = uniqueSortedStrings(merged.DisabledSinks)
	merged.NodeType = NodeTypeCoordinator
	merged.nodeInfo.NodeType = NodeTypeCoordinator
	merged.DurationMS = merged.CompletedUTC.Sub(merged.StartedUTC).Seconds() * 1000
	if merged.nodeInfo.CurrentOperation == "" {
		merged.nodeInfo.CurrentOperation = "Complete"
	}

	return merged
}

func filterScenariosByName(scenarios []scenarioDefinition, names []string) []scenarioDefinition {
	if len(names) == 0 {
		return append([]scenarioDefinition(nil), scenarios...)
	}
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			allowed[strings.ToLower(trimmed)] = struct{}{}
		}
	}
	filtered := []scenarioDefinition{}
	for _, scenario := range scenarios {
		if _, ok := allowed[strings.ToLower(strings.TrimSpace(scenario.Name))]; ok {
			filtered = append(filtered, scenario)
		}
	}
	return filtered
}

func planAgentScenarioAssignments[S scenarioLike](scenarios []S, agentCount int, context contextState) [][]string {
	assignments := make([][]string, maxInt(agentCount, 1))
	selected := make([]scenarioDefinition, 0, len(scenarios))
	for _, scenario := range scenarios {
		selected = append(selected, normalizeScenarioLike(scenario))
	}
	if len(context.TargetScenarios) > 0 {
		selected = filterScenariosByName(selected, context.TargetScenarios)
	}
	if len(context.AgentTargetScenarios) > 0 {
		allowed := map[string]string{}
		for _, scenario := range selected {
			allowed[strings.ToLower(strings.TrimSpace(scenario.Name))] = scenario.Name
		}
		names := make([]string, 0, len(context.AgentTargetScenarios))
		for _, scenarioName := range context.AgentTargetScenarios {
			if name, ok := allowed[strings.ToLower(strings.TrimSpace(scenarioName))]; ok {
				names = append(names, name)
			}
		}
		for index := range assignments {
			assignments[index] = append([]string(nil), names...)
		}
		return assignments
	}
	weightedPool := make([]string, 0, len(selected))
	for _, scenario := range selected {
		weight := scenario.Weight
		if weight <= 0 {
			weight = 1
		}
		for copyIndex := 0; copyIndex < weight; copyIndex++ {
			weightedPool = append(weightedPool, scenario.Name)
		}
	}
	for index, scenarioName := range weightedPool {
		agentIndex := index % len(assignments)
		assignments[agentIndex] = append(assignments[agentIndex], scenarioName)
	}
	return assignments
}

func aggregateClusterScenarioStats(results []runResult) []scenarioStats {
	byScenario := map[string][]scenarioStats{}
	for _, result := range results {
		for _, scenario := range result.scenarioStats {
			byScenario[scenario.ScenarioName] = append(byScenario[scenario.ScenarioName], scenario)
		}
	}
	names := make([]string, 0, len(byScenario))
	for name := range byScenario {
		names = append(names, name)
	}
	sort.SliceStable(names, func(left int, right int) bool {
		return minScenarioSortIndex(byScenario[names[left]]) < minScenarioSortIndex(byScenario[names[right]])
	})

	aggregated := make([]scenarioStats, 0, len(names))
	for _, name := range names {
		items := byScenario[name]
		totalRequests := 0
		totalOK := 0
		totalFail := 0
		totalBytes := int64(0)
		maxDuration := 0.0
		operation := ""
		loadStats := loadSimulationStats{}
		okMeasurements := make([]measurementStats, 0, len(items))
		failMeasurements := make([]measurementStats, 0, len(items))
		for _, item := range items {
			totalRequests += item.AllRequestCount
			totalOK += item.AllOKCount
			totalFail += item.AllFailCount
			totalBytes += item.AllBytes
			if item.DurationMS > maxDuration {
				maxDuration = item.DurationMS
			}
			if operation == "" {
				operation = item.CurrentOperation
			}
			if loadStats.SimulationName == "" && item.LoadSimulationStats.SimulationName != "" {
				loadStats = item.LoadSimulationStats
			}
			okMeasurements = append(okMeasurements, item.Ok)
			failMeasurements = append(failMeasurements, item.Fail)
		}
		stepStats := aggregateClusterStepStats(items, totalRequests, maxDuration)
		scenario := scenarioStats{
			ScenarioName:        name,
			AllBytes:            totalBytes,
			AllRequestCount:     totalRequests,
			AllOKCount:          totalOK,
			AllFailCount:        totalFail,
			CurrentOperation:    operation,
			DurationMS:          maxDuration,
			Ok:                  aggregateClusterMeasurementStats(okMeasurements, totalRequests, maxDuration),
			Fail:                aggregateClusterMeasurementStats(failMeasurements, totalRequests, maxDuration),
			LoadSimulationStats: loadStats,
			SortIndex:           minScenarioSortIndex(items),
			stepStats:           stepStats,
		}
		enrichDetailedScenarioStats(&scenario)
		aggregated = append(aggregated, scenario)
	}
	return aggregated
}

func aggregateClusterStepStats(scenarios []scenarioStats, totalScenarioRequests int, scenarioDurationMS float64) []stepStats {
	byStep := map[string][]stepStats{}
	for _, scenario := range scenarios {
		for _, step := range scenario.stepStats {
			byStep[step.StepName] = append(byStep[step.StepName], step)
		}
	}
	names := make([]string, 0, len(byStep))
	for name := range byStep {
		names = append(names, name)
	}
	sort.SliceStable(names, func(left int, right int) bool {
		return minStepSortIndex(byStep[names[left]]) < minStepSortIndex(byStep[names[right]])
	})

	aggregated := make([]stepStats, 0, len(names))
	for _, name := range names {
		items := byStep[name]
		okMeasurements := make([]measurementStats, 0, len(items))
		failMeasurements := make([]measurementStats, 0, len(items))
		totalBytes := int64(0)
		totalOK := 0
		totalFail := 0
		for _, item := range items {
			okMeasurements = append(okMeasurements, item.Ok)
			failMeasurements = append(failMeasurements, item.Fail)
			totalBytes += item.AllBytes
			totalOK += item.AllOKCount
			totalFail += item.AllFailCount
		}
		okMeasurement := aggregateClusterMeasurementStats(okMeasurements, totalScenarioRequests, scenarioDurationMS)
		failMeasurement := aggregateClusterMeasurementStats(failMeasurements, totalScenarioRequests, scenarioDurationMS)
		step := stepStats{
			ScenarioName:    firstNonBlank(firstStepScenarioName(items), ""),
			StepName:        name,
			AllBytes:        totalBytes,
			AllRequestCount: totalOK + totalFail,
			AllOKCount:      totalOK,
			AllFailCount:    totalFail,
			Ok:              okMeasurement,
			Fail:            failMeasurement,
			SortIndex:       minStepSortIndex(items),
		}
		enrichDetailedStepStats(&step)
		aggregated = append(aggregated, step)
	}
	return aggregated
}

func aggregateClusterMeasurementStats(measurements []measurementStats, allRequestCount int, durationMS float64) measurementStats {
	if len(measurements) == 0 {
		return measurementStats{}
	}
	totalCount := 0
	totalBytes := int64(0)
	minBytes := int64(0)
	maxBytes := int64(0)
	weightedMeanBytes := 0.0
	weightedP50Bytes := 0.0
	weightedP75Bytes := 0.0
	weightedP95Bytes := 0.0
	weightedP99Bytes := 0.0
	weightedByteStdDev := 0.0
	minLatency := 0.0
	maxLatency := 0.0
	weightedMeanLatency := 0.0
	weightedP50Latency := 0.0
	weightedP75Latency := 0.0
	weightedP95Latency := 0.0
	weightedP99Latency := 0.0
	weightedLatencyStdDev := 0.0
	latencyCount := latencyCount{}
	statusCodes := map[string]statusCodeStats{}

	for _, measurement := range measurements {
		weight := measurement.Request.Count
		totalCount += weight
		totalBytes += measurement.DataTransfer.AllBytes
		minBytes = minInt64NonZero(minBytes, measurement.DataTransfer.MinBytes)
		maxBytes = maxInt64(maxBytes, measurement.DataTransfer.MaxBytes)
		weightedMeanBytes += float64(weight) * float64(measurement.DataTransfer.MeanBytes)
		weightedP50Bytes += float64(weight) * float64(measurement.DataTransfer.Percent50)
		weightedP75Bytes += float64(weight) * float64(measurement.DataTransfer.Percent75)
		weightedP95Bytes += float64(weight) * float64(measurement.DataTransfer.Percent95)
		weightedP99Bytes += float64(weight) * float64(measurement.DataTransfer.Percent99)
		weightedByteStdDev += float64(weight) * measurement.DataTransfer.StdDev
		minLatency = minPositiveFloat(minLatency, measurement.Latency.MinMs)
		maxLatency = thresholdMaxFloat(maxLatency, measurement.Latency.MaxMs)
		weightedMeanLatency += float64(weight) * measurement.Latency.MeanMs
		weightedP50Latency += float64(weight) * latencyPercentile(measurement.Latency, 0.50)
		weightedP75Latency += float64(weight) * latencyPercentile(measurement.Latency, 0.75)
		weightedP95Latency += float64(weight) * latencyPercentile(measurement.Latency, 0.95)
		weightedP99Latency += float64(weight) * latencyPercentile(measurement.Latency, 0.99)
		weightedLatencyStdDev += float64(weight) * measurement.Latency.StdDev
		latencyCount = mergeLatencyCount(latencyCount, measurement.Latency.LatencyCount)
		for _, status := range measurement.StatusCodes {
			key := strings.ToLower(strings.TrimSpace(status.StatusCode + "\x00" + status.Message + "\x00" + fmt.Sprintf("%t", status.IsError)))
			current := statusCodes[key]
			if current.StatusCode == "" {
				current = statusCodeStats{
					StatusCode: status.StatusCode,
					Message:    status.Message,
					IsError:    status.IsError,
				}
			}
			current.Count += status.Count
			statusCodes[key] = current
		}
	}

	statusRows := make([]statusCodeStats, 0, len(statusCodes))
	for _, row := range statusCodes {
		row.Percent = percentageInt(row.Count, totalCount)
		statusRows = append(statusRows, row)
	}
	sort.SliceStable(statusRows, func(left int, right int) bool {
		if statusRows[left].Count == statusRows[right].Count {
			return statusRows[left].StatusCode < statusRows[right].StatusCode
		}
		return statusRows[left].Count > statusRows[right].Count
	})

	request := requestStats{Count: totalCount, Percent: percentageInt(totalCount, allRequestCount)}
	if durationMS > 0 {
		request.RPS = float64(totalCount) / (durationMS / 1000.0)
	}

	return measurementStats{
		Request: request,
		DataTransfer: dataTransferStats{
			AllBytes:  totalBytes,
			MinBytes:  minBytes,
			MaxBytes:  maxBytes,
			MeanBytes: weightedInt64(weightedMeanBytes, totalCount),
			Percent50: weightedInt64(weightedP50Bytes, totalCount),
			Percent75: weightedInt64(weightedP75Bytes, totalCount),
			Percent95: weightedInt64(weightedP95Bytes, totalCount),
			Percent99: weightedInt64(weightedP99Bytes, totalCount),
			StdDev:    weightedFloat(weightedByteStdDev, totalCount),
		},
		Latency: latencyStats{
			LatencyCount: latencyCount,
			MinMs:        minLatency,
			MaxMs:        maxLatency,
			MeanMs:       weightedFloat(weightedMeanLatency, totalCount),
			Percent50:    weightedFloat(weightedP50Latency, totalCount),
			Percent75:    weightedFloat(weightedP75Latency, totalCount),
			Percent95:    weightedFloat(weightedP95Latency, totalCount),
			Percent99:    weightedFloat(weightedP99Latency, totalCount),
			StdDev:       weightedFloat(weightedLatencyStdDev, totalCount),
			P50Ms:        weightedFloat(weightedP50Latency, totalCount),
			P75Ms:        weightedFloat(weightedP75Latency, totalCount),
			P95Ms:        weightedFloat(weightedP95Latency, totalCount),
			P99Ms:        weightedFloat(weightedP99Latency, totalCount),
			Samples:      totalCount,
		},
		StatusCodes: statusRows,
	}
}

func aggregateClusterMetricStats(results []runResult) metricStats {
	counters := map[string]counterStats{}
	gaugeValues := map[string][]float64{}
	gaugeMeta := map[string]gaugeStats{}
	maxDuration := 0.0
	for _, result := range results {
		if result.metricStats.DurationMS > maxDuration {
			maxDuration = result.metricStats.DurationMS
		}
		for _, counter := range result.metricStats.Counters {
			key := strings.ToLower(strings.TrimSpace(counter.ScenarioName + "\x00" + counter.MetricName + "\x00" + counter.UnitOfMeasure))
			current := counters[key]
			if current.MetricName == "" {
				current = counterStats{
					ScenarioName:  counter.ScenarioName,
					MetricName:    counter.MetricName,
					UnitOfMeasure: counter.UnitOfMeasure,
				}
			}
			current.Value += counter.Value
			counters[key] = current
		}
		for _, gauge := range result.metricStats.Gauges {
			key := strings.ToLower(strings.TrimSpace(gauge.ScenarioName + "\x00" + gauge.MetricName + "\x00" + gauge.UnitOfMeasure))
			gaugeValues[key] = append(gaugeValues[key], gauge.Value)
			if _, ok := gaugeMeta[key]; !ok {
				gaugeMeta[key] = gaugeStats{
					ScenarioName:  gauge.ScenarioName,
					MetricName:    gauge.MetricName,
					UnitOfMeasure: gauge.UnitOfMeasure,
				}
			}
		}
	}
	counterRows := make([]counterStats, 0, len(counters))
	for _, row := range counters {
		counterRows = append(counterRows, row)
	}
	sort.SliceStable(counterRows, func(left int, right int) bool {
		return counterRows[left].MetricName < counterRows[right].MetricName
	})
	gaugeRows := make([]gaugeStats, 0, len(gaugeValues))
	for key, values := range gaugeValues {
		total := 0.0
		for _, value := range values {
			total += value
		}
		meta := gaugeMeta[key]
		gaugeRows = append(gaugeRows, gaugeStats{
			ScenarioName:  meta.ScenarioName,
			MetricName:    meta.MetricName,
			UnitOfMeasure: meta.UnitOfMeasure,
			Value:         total / float64(len(values)),
		})
	}
	sort.SliceStable(gaugeRows, func(left int, right int) bool {
		return gaugeRows[left].MetricName < gaugeRows[right].MetricName
	})
	return metricStats{
		DurationMS: maxDuration,
		Duration:   time.Duration(maxDuration * float64(time.Millisecond)),
		Counters:   counterRows,
		Gauges:     gaugeRows,
	}
}

func aggregateClusterThresholds(results []runResult) []thresholdResult {
	index := map[string]thresholdResult{}
	for _, result := range results {
		for _, threshold := range result.Thresholds {
			key := strings.ToLower(strings.TrimSpace(threshold.ScenarioName + "\x00" + threshold.StepName + "\x00" + threshold.CheckExpression))
			current := index[key]
			if current.CheckExpression == "" {
				current = thresholdResult{
					ScenarioName:    threshold.ScenarioName,
					StepName:        threshold.StepName,
					CheckExpression: threshold.CheckExpression,
				}
			}
			current.ErrorCount += threshold.ErrorCount
			current.IsFailed = current.IsFailed || threshold.IsFailed
			current.ExceptionMessage = appendDistinctMessage(current.ExceptionMessage, threshold.ExceptionMessage)
			index[key] = current
		}
	}
	rows := make([]thresholdResult, 0, len(index))
	for _, row := range index {
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(left int, right int) bool {
		return rows[left].CheckExpression < rows[right].CheckExpression
	})
	return rows
}

func aggregateClusterPlugins(results []runResult) []pluginData {
	index := map[string]*pluginData{}
	order := []string{}
	for _, result := range results {
		for _, plugin := range result.PluginsData {
			key := strings.TrimSpace(plugin.PluginName)
			entry := index[key]
			if entry == nil {
				entry = &pluginData{PluginName: plugin.PluginName, Hints: []string{}, Tables: []pluginDataTable{}}
				index[key] = entry
				order = append(order, key)
			}
			entry.Hints = uniqueSortedStrings(append(entry.Hints, plugin.Hints...))
			for _, table := range plugin.Tables {
				tableIndex := -1
				for idx := range entry.Tables {
					if entry.Tables[idx].TableName == table.TableName {
						tableIndex = idx
						break
					}
				}
				if tableIndex < 0 {
					entry.Tables = append(entry.Tables, pluginDataTable{TableName: table.TableName, Rows: []map[string]any{}})
					tableIndex = len(entry.Tables) - 1
				}
				entry.Tables[tableIndex].Rows = append(entry.Tables[tableIndex].Rows, table.Rows...)
			}
		}
	}
	plugins := make([]pluginData, 0, len(order))
	for _, key := range order {
		plugins = append(plugins, *index[key])
	}
	return plugins
}

func flattenClusterStepStats(scenarios []scenarioStats) []stepStats {
	rows := []stepStats{}
	for _, scenario := range scenarios {
		rows = append(rows, scenario.stepStats...)
	}
	return rows
}

func minScenarioSortIndex(items []scenarioStats) int {
	if len(items) == 0 {
		return 0
	}
	minimum := items[0].SortIndex
	for _, item := range items[1:] {
		if item.SortIndex < minimum {
			minimum = item.SortIndex
		}
	}
	return minimum
}

func minStepSortIndex(items []stepStats) int {
	if len(items) == 0 {
		return 0
	}
	minimum := items[0].SortIndex
	for _, item := range items[1:] {
		if item.SortIndex < minimum {
			minimum = item.SortIndex
		}
	}
	return minimum
}

func firstStepScenarioName(items []stepStats) string {
	for _, item := range items {
		if strings.TrimSpace(item.ScenarioName) != "" {
			return item.ScenarioName
		}
	}
	return ""
}

func weightedInt64(weightedSum float64, totalCount int) int64 {
	if totalCount <= 0 {
		return 0
	}
	return int64(weightedSum / float64(totalCount))
}

func weightedFloat(weightedSum float64, totalCount int) float64 {
	if totalCount <= 0 {
		return 0
	}
	return weightedSum / float64(totalCount)
}

func uniqueSortedStrings(values []string) []string {
	index := map[string]string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		index[strings.ToLower(trimmed)] = trimmed
	}
	keys := make([]string, 0, len(index))
	for _, value := range index {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

func appendDistinctMessage(existing string, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return strings.TrimSpace(existing)
	}
	if strings.TrimSpace(existing) == "" {
		return next
	}
	parts := strings.Split(existing, " | ")
	for _, part := range parts {
		if strings.TrimSpace(part) == next {
			return existing
		}
	}
	return existing + " | " + next
}

func clusterCommandTimeout(context contextState) time.Duration {
	seconds := context.ClusterCommandTimeoutSeconds
	if seconds <= 0 {
		seconds = 120
	}
	return time.Duration(seconds * float64(time.Second))
}

func clusterRunSubject(context contextState) string {
	clusterID := sanitizeClusterToken(firstNonBlank(context.ClusterID, "default"))
	agentGroup := sanitizeClusterToken(firstNonBlank(context.AgentGroup, "default"))
	return fmt.Sprintf("loadstrike.%s.%s.run", clusterID, agentGroup)
}

func clusterRunSubjectForAgent(context contextState, agentIndex int) string {
	return fmt.Sprintf("%s.%d", clusterRunSubject(context), agentIndex)
}

func clusterReplySubject(context contextState) string {
	clusterID := sanitizeClusterToken(firstNonBlank(context.ClusterID, "default"))
	sessionID := sanitizeClusterToken(firstNonBlank(context.SessionID, generateSessionID()))
	return fmt.Sprintf("loadstrike.%s.%s.reply.%d", clusterID, sessionID, time.Now().UnixNano())
}

func clusterQueueGroup(context contextState) string {
	clusterID := sanitizeClusterToken(firstNonBlank(context.ClusterID, "default"))
	agentGroup := sanitizeClusterToken(firstNonBlank(context.AgentGroup, "default"))
	return fmt.Sprintf("loadstrike.%s.%s.agents", clusterID, agentGroup)
}

func sanitizeClusterToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "default"
	}
	builder := strings.Builder{}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
			builder.WriteRune(character)
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
		default:
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}
