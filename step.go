package loadstrike

import (
	stdcontext "context"
	"strings"
	"time"
)

// replyResult is the public step result contract.
type replyResult struct {
	IsSuccess       bool
	IsError         bool
	StatusCode      string
	Message         string
	SizeBytes       int64
	CustomLatencyMS float64
	Payload         any
}

// stepRuntimeContext carries the currently executing scenario or step metadata.
type stepRuntimeContext struct {
	ScenarioName              string
	StepName                  string
	TestSuite                 string
	TestName                  string
	InvocationNumber          int64
	Data                      map[string]any
	ScenarioInstanceData      map[string]any
	Logger                    *LoadStrikeLogger
	nodeInfo                  nodeInfo
	testInfo                  testInfo
	ScenarioInfo              LoadStrikeScenarioInfo
	Random                    *LoadStrikeRandom
	ScenarioCancellationToken stdcontext.Context
	scenarioCancel            stdcontext.CancelFunc
	executionControl          *executionControl
	scenarioTimerStarted      time.Time
	stopRequested             bool
	stopCurrentTest           bool
}

// scenarioStep is a single runnable unit inside a scenario.
type scenarioStep struct {
	Name string
	Run  func(*stepRuntimeContext) replyResult
}

func okReply(args ...any) replyResult {
	statusCode, sizeBytes, message, customLatencyMS := parseReplyOptions(true, args...)
	return replyResult{
		IsSuccess:       true,
		IsError:         false,
		StatusCode:      statusCode,
		SizeBytes:       sizeBytes,
		Message:         message,
		CustomLatencyMS: customLatencyMS,
	}
}

func failReply(args ...any) replyResult {
	statusCode, sizeBytes, message, customLatencyMS := parseReplyOptions(false, args...)
	return replyResult{
		IsSuccess:       false,
		IsError:         true,
		StatusCode:      statusCode,
		SizeBytes:       sizeBytes,
		Message:         message,
		CustomLatencyMS: customLatencyMS,
	}
}

// OK returns a successful default reply on the public wrapper surface.
func OK(args ...any) LoadStrikeReply {
	return newLoadStrikeReply(okReply(args...))
}

// Fail returns a failed reply with the provided status code and message on the public wrapper surface.
func Fail(args ...any) LoadStrikeReply {
	return newLoadStrikeReply(failReply(args...))
}

// StopScenario requests that no further iterations are executed for the current scenario.
func (c *stepRuntimeContext) StopScenario(args ...string) {
	if c != nil {
		scenarioName := c.ScenarioName
		reason := ""
		if len(args) == 1 {
			reason = args[0]
		}
		if len(args) >= 2 {
			scenarioName = args[0]
			reason = args[1]
		}
		c.stopRequested = true
		if c.executionControl != nil {
			c.executionControl.stopScenario(firstNonBlank(strings.TrimSpace(scenarioName), c.ScenarioName), reason)
		}
		if c.scenarioCancel != nil && strings.EqualFold(strings.TrimSpace(firstNonBlank(scenarioName, c.ScenarioName)), c.ScenarioName) {
			c.scenarioCancel()
		}
	}
}

// StopCurrentTest requests that the current test run is stopped.
func (c *stepRuntimeContext) StopCurrentTest(reason string) {
	if c != nil {
		c.stopCurrentTest = true
		c.stopRequested = true
		if c.executionControl != nil {
			c.executionControl.stopTest(reason)
		}
		if c.scenarioCancel != nil {
			c.scenarioCancel()
		}
	}
}

// GetScenarioTimerTime returns the elapsed time for the current scenario iteration.
func (c *stepRuntimeContext) GetScenarioTimerTime() time.Duration {
	if c == nil || c.scenarioTimerStarted.IsZero() {
		return 0
	}
	return time.Since(c.scenarioTimerStarted)
}

func parseReplyOptions(success bool, args ...any) (string, int64, string, float64) {
	statusCode := "200"
	message := ""
	sizeBytes := int64(0)
	customLatencyMS := float64(-1)
	if !success {
		statusCode = "500"
	}
	if len(args) > 0 {
		if value, ok := args[0].(string); ok && strings.TrimSpace(value) != "" {
			statusCode = value
		}
	}
	if len(args) > 1 {
		switch typed := args[1].(type) {
		case string:
			message = typed
		case int:
			sizeBytes = int64(typed)
		case int64:
			sizeBytes = typed
		case int32:
			sizeBytes = int64(typed)
		case float64:
			sizeBytes = int64(typed)
		}
	}
	if len(args) > 2 {
		switch typed := args[2].(type) {
		case string:
			message = typed
		case int:
			sizeBytes = int64(typed)
		case int64:
			sizeBytes = typed
		case float64:
			sizeBytes = int64(typed)
		}
	}
	if len(args) > 3 {
		switch typed := args[3].(type) {
		case float64:
			customLatencyMS = typed
		case float32:
			customLatencyMS = float64(typed)
		case int:
			customLatencyMS = float64(typed)
		case int64:
			customLatencyMS = float64(typed)
		case time.Duration:
			customLatencyMS = typed.Seconds() * 1000
		}
	}
	return statusCode, sizeBytes, message, customLatencyMS
}
