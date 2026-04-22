package loadstrike

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ReportFormat identifies a report file type.
type ReportFormat string

const (
	ReportFormatHTML ReportFormat = "html"
	ReportFormatTXT  ReportFormat = "txt"
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatMD   ReportFormat = "md"
)

type reportFailure struct {
	ScenarioName string
	StepName     string
	StatusCode   string
	Message      string
}

type reportTrace struct {
	stepNames       []string
	stepNameSet     map[string]struct{}
	failedResponses []reportFailure
}

func newReportTrace() *reportTrace {
	return &reportTrace{
		stepNameSet: make(map[string]struct{}),
	}
}

func (t *reportTrace) addStep(stepName string) {
	if t == nil || stepName == "" {
		return
	}

	if _, exists := t.stepNameSet[stepName]; exists {
		return
	}

	t.stepNameSet[stepName] = struct{}{}
	t.stepNames = append(t.stepNames, stepName)
}

func (t *reportTrace) addFailure(scenarioName, stepName string, reply replyResult) {
	if t == nil {
		return
	}

	t.failedResponses = append(t.failedResponses, reportFailure{
		ScenarioName: scenarioName,
		StepName:     stepName,
		StatusCode:   reply.StatusCode,
		Message:      reply.Message,
	})
}

func (t *reportTrace) merge(other *reportTrace) {
	if t == nil || other == nil {
		return
	}

	for _, stepName := range other.stepNames {
		t.addStep(stepName)
	}

	t.failedResponses = append(t.failedResponses, other.failedResponses...)
}

func writeReports(context contextState, result *runResult) error {
	if result == nil {
		return nil
	}

	reportFolder := context.ReportFolder
	if strings.TrimSpace(reportFolder) == "" {
		resolvedFolder, err := defaultArtifactFolder(context)
		if err != nil {
			resolvedFolder = filepath.Join(".", "reports")
		}
		reportFolder = resolvedFolder
	}

	reportFileName := context.ReportFileName
	if strings.TrimSpace(reportFileName) == "" {
		reportFileName = defaultReportBaseName(result.testInfo)
	}
	reportFileName = sanitizeReportFileName(reportFileName)

	if err := os.MkdirAll(reportFolder, 0o755); err != nil {
		return err
	}

	formats := context.ReportFormats
	if len(formats) == 0 {
		formats = []ReportFormat{ReportFormatHTML, ReportFormatTXT, ReportFormatCSV, ReportFormatMD}
	}

	for _, format := range formats {
		reportPath := filepath.Join(reportFolder, reportFileName+"."+string(format))
		var body string

		switch format {
		case ReportFormatHTML:
			body = renderHTMLReport(result)
		case ReportFormatTXT:
			body = renderTextReport(result)
		case ReportFormatCSV:
			body = renderCSVReport(result)
		case ReportFormatMD:
			body = renderMarkdownReport(result)
		default:
			continue
		}

		if err := os.WriteFile(reportPath, []byte(body), 0o600); err != nil {
			return err
		}

		result.ReportFiles = append(result.ReportFiles, reportPath)
	}

	return nil
}

func defaultReportBaseName(testInfo testInfo) string {
	timestamp := testInfo.CreatedUTC
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	return fmt.Sprintf(
		"%s_%s_%s",
		firstNonBlank(testInfo.TestSuite, "default"),
		firstNonBlank(testInfo.TestName, "loadstrike"),
		timestamp.UTC().Format("20060102_150405"),
	)
}

var invalidReportFileCharacters = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizeReportFileName(value string) string {
	return invalidReportFileCharacters.ReplaceAllString(value, "_")
}

func renderHTMLReport(result *runResult) string {
	return renderDotnetHTMLReport(reportStatsFromRunResult(result))
}

func renderTextReport(result *runResult) string {
	return renderDotnetTextReport(reportStatsFromRunResult(result))
}

func renderCSVReport(result *runResult) string {
	return renderDotnetCSVReport(reportStatsFromRunResult(result))
}

func renderMarkdownReport(result *runResult) string {
	return renderDotnetMarkdownReport(reportStatsFromRunResult(result))
}

func renderDotnetTextReport(nodeStats map[string]any) string {
	scenarios := sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats"))
	testInfo := reportObject(nodeStats, "testInfo", "testInfo")
	nodeInfo := reportObject(nodeStats, "nodeInfo", "nodeInfo")

	lines := []string{
		"TestSuite: " + asString(reportValue(testInfo, "testSuite", "TestSuite")),
		"TestName: " + asString(reportValue(testInfo, "testName", "TestName")),
		"SessionId: " + asString(reportValue(testInfo, "sessionId", "SessionId")),
		"NodeType: " + fmt.Sprintf("%d", loadStrikeNodeTypeTag(reportValue(nodeInfo, "nodeType", "NodeType"))),
		"Duration: " + formatDotnetTimeSpan(reportValue(nodeStats, "duration", "Duration", "durationMs", "DurationMs")),
		fmt.Sprintf(
			"Requests: %d  OK: %d  FAIL: %d",
			asInt(reportValue(nodeStats, "allRequestCount", "AllRequestCount")),
			asInt(reportValue(nodeStats, "allOkCount", "AllOkCount")),
			asInt(reportValue(nodeStats, "allFailCount", "AllFailCount")),
		),
		"",
		"Scenarios:",
	}

	for _, scenario := range scenarios {
		lines = append(lines, fmt.Sprintf(
			"- %s: req=%d ok=%d fail=%d duration=%s",
			asString(reportValue(scenario, "scenarioName", "ScenarioName")),
			asInt(reportValue(scenario, "allRequestCount", "AllRequestCount")),
			asInt(reportValue(scenario, "allOkCount", "AllOkCount")),
			asInt(reportValue(scenario, "allFailCount", "AllFailCount")),
			formatDotnetTimeSpan(reportValue(scenario, "duration", "Duration", "durationMs", "DurationMs")),
		))
	}

	hasSteps := false
	for _, scenario := range scenarios {
		if len(reportRecordList(scenario, "stepStats", "stepStats")) > 0 {
			hasSteps = true
			break
		}
	}
	if hasSteps {
		lines = append(lines, "", "Steps:")
		for _, scenario := range scenarios {
			for _, step := range sortBySortIndex(reportRecordList(scenario, "stepStats", "stepStats")) {
				okCount := requestCount(reportObject(step, "ok", "Ok"))
				failCount := requestCount(reportObject(step, "fail", "Fail"))
				lines = append(lines, fmt.Sprintf(
					"- %s.%s: req=%d ok=%d fail=%d",
					asString(reportValue(scenario, "scenarioName", "ScenarioName")),
					asString(reportValue(step, "stepName", "StepName")),
					okCount+failCount,
					okCount,
					failCount,
				))
			}
		}
	}

	thresholds := reportRecordList(nodeStats, "thresholds", "Thresholds")
	if len(thresholds) > 0 {
		lines = append(lines, "", "Thresholds:")
		for _, threshold := range thresholds {
			lines = append(lines, fmt.Sprintf(
				"- scenario=%s step=%s check=%s failed=%s errors=%d message=%s",
				asString(reportValue(threshold, "scenarioName", "ScenarioName")),
				asString(reportValue(threshold, "stepName", "StepName")),
				asString(reportValue(threshold, "checkExpression", "CheckExpression")),
				formatCellValue(asBool(reportValue(threshold, "isFailed", "IsFailed"))),
				asInt(reportValue(threshold, "errorCount", "ErrorCount")),
				asString(reportValue(threshold, "exceptionMessage", "ExceptionMessage", "exceptionMsg", "ExceptionMsg")),
			))
		}
	}

	plugins := reportRecordList(nodeStats, "pluginsData", "PluginsData")
	if len(plugins) > 0 {
		lines = append(lines, "", "Plugin Tables:")
		for _, plugin := range plugins {
			lines = append(lines, "- "+asString(reportValue(plugin, "pluginName", "PluginName")))
			for _, table := range reportRecordList(plugin, "tables", "Tables") {
				lines = append(lines, fmt.Sprintf(
					"  Table: %s rows=%d",
					asString(reportValue(table, "tableName", "TableName")),
					len(reportRecordList(table, "rows", "Rows")),
				))
			}
		}
	}

	return reportLines(lines)
}

func renderDotnetCSVReport(nodeStats map[string]any) string {
	lines := []string{"ScenarioName,Requests,Ok,Fail,DurationSeconds,Rps"}
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		durationSeconds := reportDurationSeconds(reportValue(scenario, "duration", "Duration", "durationMs", "DurationMs"))
		requests := asInt(reportValue(scenario, "allRequestCount", "AllRequestCount"))
		rps := 0.0
		if durationSeconds > 0 {
			rps = float64(requests) / durationSeconds
		}
		lines = append(lines, fmt.Sprintf(
			"%s,%d,%d,%d,%s,%s",
			escapeCSV(reportValue(scenario, "scenarioName", "ScenarioName")),
			requests,
			asInt(reportValue(scenario, "allOkCount", "AllOkCount")),
			asInt(reportValue(scenario, "allFailCount", "AllFailCount")),
			formatReportNumber(durationSeconds),
			formatReportNumber(rps),
		))
	}
	return reportLines(lines)
}

func renderDotnetMarkdownReport(nodeStats map[string]any) string {
	scenarios := sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats"))
	testInfo := reportObject(nodeStats, "testInfo", "testInfo")
	lines := []string{
		"# " + asString(reportValue(testInfo, "testSuite", "TestSuite")) + " / " + asString(reportValue(testInfo, "testName", "TestName")),
		"",
		"- Session: `" + asString(reportValue(testInfo, "sessionId", "SessionId")) + "`",
		"- Duration: `" + formatDotnetTimeSpan(reportValue(nodeStats, "duration", "Duration", "durationMs", "DurationMs")) + "`",
		"- Total Requests: `" + fmt.Sprintf("%d", asInt(reportValue(nodeStats, "allRequestCount", "AllRequestCount"))) + "`",
		"- OK: `" + fmt.Sprintf("%d", asInt(reportValue(nodeStats, "allOkCount", "AllOkCount"))) + "`",
		"- FAIL: `" + fmt.Sprintf("%d", asInt(reportValue(nodeStats, "allFailCount", "AllFailCount"))) + "`",
		"",
		"## Scenarios",
		"",
		"| Scenario | Requests | OK | FAIL | Duration |",
		"|---|---:|---:|---:|---:|",
	}

	for _, scenario := range scenarios {
		lines = append(lines, fmt.Sprintf(
			"| %s | %d | %d | %d | %ss |",
			asString(reportValue(scenario, "scenarioName", "ScenarioName")),
			asInt(reportValue(scenario, "allRequestCount", "AllRequestCount")),
			asInt(reportValue(scenario, "allOkCount", "AllOkCount")),
			asInt(reportValue(scenario, "allFailCount", "AllFailCount")),
			formatReportNumber(reportDurationSeconds(reportValue(scenario, "duration", "Duration", "durationMs", "DurationMs"))),
		))
	}

	hasSteps := false
	for _, scenario := range scenarios {
		if len(reportRecordList(scenario, "stepStats", "stepStats")) > 0 {
			hasSteps = true
			break
		}
	}
	if hasSteps {
		lines = append(lines,
			"",
			"## Steps",
			"",
			"| Scenario | Step | Requests | OK | FAIL |",
			"|---|---|---:|---:|---:|",
		)
		for _, scenario := range scenarios {
			for _, step := range sortBySortIndex(reportRecordList(scenario, "stepStats", "stepStats")) {
				okCount := requestCount(reportObject(step, "ok", "Ok"))
				failCount := requestCount(reportObject(step, "fail", "Fail"))
				lines = append(lines, fmt.Sprintf(
					"| %s | %s | %d | %d | %d |",
					asString(reportValue(scenario, "scenarioName", "ScenarioName")),
					asString(reportValue(step, "stepName", "StepName")),
					okCount+failCount,
					okCount,
					failCount,
				))
			}
		}
	}

	thresholds := reportRecordList(nodeStats, "thresholds", "Thresholds")
	if len(thresholds) > 0 {
		lines = append(lines,
			"",
			"## Thresholds",
			"",
			"| Scenario | Step | Check | Failed | Errors | Exception |",
			"|---|---|---|---:|---:|---|",
		)
		for _, threshold := range thresholds {
			lines = append(lines, fmt.Sprintf(
				"| %s | %s | %s | %s | %d | %s |",
				asString(reportValue(threshold, "scenarioName", "ScenarioName")),
				asString(reportValue(threshold, "stepName", "StepName")),
				asString(reportValue(threshold, "checkExpression", "CheckExpression")),
				formatCellValue(asBool(reportValue(threshold, "isFailed", "IsFailed"))),
				asInt(reportValue(threshold, "errorCount", "ErrorCount")),
				asString(reportValue(threshold, "exceptionMessage", "ExceptionMessage", "exceptionMsg", "ExceptionMsg")),
			))
		}
	}

	return reportLines(lines)
}

func reportStatsFromRunResult(result *runResult) map[string]any {
	if result == nil {
		return map[string]any{}
	}
	body, err := json.Marshal(result)
	if err != nil {
		return map[string]any{}
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return map[string]any{}
	}
	decoded["DurationMs"] = result.DurationMS
	decoded["durationMs"] = result.DurationMS
	if result.testInfo.CreatedUTC.IsZero() {
		decoded["StartedUtc"] = result.StartedUTC.Format(time.RFC3339Nano)
		decoded["CompletedUtc"] = result.CompletedUTC.Format(time.RFC3339Nano)
	}
	if result.nodeInfo.CurrentOperation == "" {
		nodeInfo := reportObject(decoded, "nodeInfo", "nodeInfo")
		nodeInfo["CurrentOperation"] = "Complete"
		decoded["nodeInfo"] = nodeInfo
	}
	return decoded
}

type projectedStatusCode struct {
	Message string
	Count   int
	IsError bool
}

func inferFailureStatusCodes(trace *reportTrace) (map[string]map[string]projectedStatusCode, map[string]map[string]projectedStatusCode) {
	stepFailures := make(map[string]map[string]projectedStatusCode)
	scenarioFailures := make(map[string]map[string]projectedStatusCode)
	if trace == nil {
		return stepFailures, scenarioFailures
	}
	for _, failure := range trace.failedResponses {
		scenarioName := strings.TrimSpace(failure.ScenarioName)
		stepName := strings.TrimSpace(failure.StepName)
		statusCode := strings.TrimSpace(failure.StatusCode)
		if statusCode == "" {
			statusCode = "failed"
		}
		if _, exists := stepFailures[scenarioName+"\x00"+stepName]; !exists {
			stepFailures[scenarioName+"\x00"+stepName] = make(map[string]projectedStatusCode)
		}
		if _, exists := scenarioFailures[scenarioName]; !exists {
			scenarioFailures[scenarioName] = make(map[string]projectedStatusCode)
		}
		stepFailures[scenarioName+"\x00"+stepName][statusCode] = mergeProjectedStatusCode(stepFailures[scenarioName+"\x00"+stepName][statusCode], failure)
		scenarioFailures[scenarioName][statusCode] = mergeProjectedStatusCode(scenarioFailures[scenarioName][statusCode], failure)
	}
	return stepFailures, scenarioFailures
}

func mergeProjectedStatusCode(existing projectedStatusCode, failure reportFailure) projectedStatusCode {
	existing.Count++
	existing.IsError = true
	if existing.Message == "" {
		existing.Message = failure.Message
	}
	return existing
}

func projectDefaultSuccessStatusCodes(count int) []map[string]any {
	if count <= 0 {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"StatusCode": "200",
			"Message":    "OK",
			"Count":      count,
			"Percent":    100,
			"IsError":    false,
		},
	}
}

func projectStatusCodeRecords(codes map[string]projectedStatusCode, totalCount int, defaultError bool) []map[string]any {
	if totalCount <= 0 {
		return []map[string]any{}
	}
	if len(codes) == 0 {
		return []map[string]any{
			{
				"StatusCode": "failed",
				"Message":    "Failure",
				"Count":      totalCount,
				"Percent":    100,
				"IsError":    defaultError,
			},
		}
	}
	keys := make([]string, 0, len(codes))
	for key := range codes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		code := codes[key]
		percent := 0
		if totalCount > 0 {
			percent = int(float64(code.Count) * 100 / float64(totalCount))
		}
		rows = append(rows, map[string]any{
			"StatusCode": key,
			"Message":    code.Message,
			"Count":      code.Count,
			"Percent":    percent,
			"IsError":    code.IsError,
		})
	}
	return rows
}

func reportLines(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}

func reportValue(source any, keys ...string) any {
	record, ok := source.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range keys {
		if value, exists := record[key]; exists {
			return value
		}
	}
	return nil
}

func reportObject(source any, keys ...string) map[string]any {
	value := reportValue(source, keys...)
	record, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return record
}

func reportRecordList(source any, keys ...string) []map[string]any {
	value := reportValue(source, keys...)
	switch typed := value.(type) {
	case []any:
		rows := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if record, ok := item.(map[string]any); ok {
				rows = append(rows, record)
			}
		}
		return rows
	case []map[string]any:
		return append([]map[string]any(nil), typed...)
	default:
		return nil
	}
}

func sortBySortIndex(rows []map[string]any) []map[string]any {
	ordered := append([]map[string]any(nil), rows...)
	sort.SliceStable(ordered, func(left int, right int) bool {
		return asInt(reportValue(ordered[left], "sortIndex", "SortIndex")) < asInt(reportValue(ordered[right], "sortIndex", "SortIndex"))
	})
	return ordered
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case bool:
		if typed {
			return 1
		}
		return 0
	default:
		var parsed float64
		if _, err := fmt.Sscan(fmt.Sprint(value), &parsed); err == nil {
			return int(parsed)
		}
		return 0
	}
}

func asDouble(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		var parsed float64
		if _, err := fmt.Sscan(fmt.Sprint(value), &parsed); err == nil {
			return parsed
		}
		return 0
	}
}

func asBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	default:
		text := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		return text == "1" || text == "true" || text == "yes" || text == "on"
	}
}

func formatCellValue(value any) string {
	if asBool(value) {
		return "True"
	}
	if value == nil {
		return ""
	}
	if text, ok := value.(bool); ok && !text {
		return "False"
	}
	return fmt.Sprint(value)
}

func requestCount(measurement map[string]any) int {
	return asInt(reportValue(reportObject(measurement, "request", "Request"), "count", "Count"))
}

func requestRps(measurement map[string]any) float64 {
	return asDouble(reportValue(reportObject(measurement, "request", "Request"), "rps", "RPS"))
}

func reportDurationSeconds(value any) float64 {
	return coerceReportDurationMs(value) / 1000.0
}

func coerceReportDurationMs(value any) float64 {
	switch typed := value.(type) {
	case string:
		if parsed, ok := parseDotnetTimeSpan(typed); ok {
			return parsed
		}
		return asDouble(typed)
	default:
		return asDouble(value)
	}
}

func formatDotnetTimeSpan(value any) string {
	totalTicks := int64(coerceReportDurationMs(value) * 10000.0)
	sign := ""
	if totalTicks < 0 {
		sign = "-"
		totalTicks = -totalTicks
	}
	ticksPerDay := int64(24 * 60 * 60 * 10000000)
	ticksPerHour := int64(60 * 60 * 10000000)
	ticksPerMinute := int64(60 * 10000000)
	ticksPerSecond := int64(10000000)
	days := totalTicks / ticksPerDay
	totalTicks %= ticksPerDay
	hours := totalTicks / ticksPerHour
	totalTicks %= ticksPerHour
	minutes := totalTicks / ticksPerMinute
	totalTicks %= ticksPerMinute
	seconds := totalTicks / ticksPerSecond
	ticks := totalTicks % ticksPerSecond
	base := fmt.Sprintf("%s", sign)
	if days > 0 {
		base += fmt.Sprintf("%d.", days)
	}
	base += fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	if ticks == 0 {
		return base
	}
	return base + fmt.Sprintf(".%07d", ticks)
}

func parseDotnetTimeSpan(value string) (float64, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0, false
	}
	sign := 1.0
	if strings.HasPrefix(text, "-") {
		sign = -1
		text = strings.TrimPrefix(text, "-")
	}
	days := 0
	if strings.Count(text, ".") > 0 && strings.Index(text, ".") < strings.Index(text, ":") {
		pieces := strings.SplitN(text, ".", 2)
		days = asInt(pieces[0])
		text = pieces[1]
	}
	parts := strings.Split(text, ":")
	if len(parts) != 3 {
		return 0, false
	}
	secondsPart := parts[2]
	fraction := ""
	if strings.Contains(secondsPart, ".") {
		pieces := strings.SplitN(secondsPart, ".", 2)
		secondsPart = pieces[0]
		fraction = pieces[1]
	}
	hours := asInt(parts[0])
	minutes := asInt(parts[1])
	seconds := asInt(secondsPart)
	for len(fraction) < 7 {
		fraction += "0"
	}
	if len(fraction) > 7 {
		fraction = fraction[:7]
	}
	ticks := asInt(fraction)
	milliseconds := ((((float64(days)*24)+float64(hours))*60)+float64(minutes))*60*1000 + float64(seconds)*1000 + float64(ticks)/10000.0
	return sign * milliseconds, true
}

func formatReportNumber(value float64) string {
	text := fmt.Sprintf("%.3f", value)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func escapeCSV(value any) string {
	text := asString(value)
	if strings.ContainsAny(text, ",\"\n\r") {
		return `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
	}
	return text
}

func loadStrikeNodeTypeTag(value any) int {
	if record, ok := value.(map[string]any); ok {
		if tag := asInt(reportValue(record, "tag", "Tag")); tag != 0 {
			return tag
		}
	}
	switch strings.TrimSpace(asString(value)) {
	case "Coordinator":
		return 1
	case "Agent":
		return 2
	default:
		return asInt(value)
	}
}
