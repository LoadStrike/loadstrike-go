package loadstrike

import (
	stdcontext "context"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"time"
)

// LoadStrikeLogger mirrors the .NET public logger concept with a minimal leveled API.
type LoadStrikeLogger struct {
	write func(level string, message string)
}

// LoadStrikeRandom mirrors the .NET public random helper surface.
type LoadStrikeRandom struct {
	inner *rand.Rand
}

func newLoadStrikeLogger(writer func(level string, message string)) *LoadStrikeLogger {
	return &LoadStrikeLogger{write: writer}
}

func (l *LoadStrikeLogger) log(level string, format string, args ...any) {
	if l == nil || l.write == nil {
		return
	}
	message := format
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}
	l.write(level, message)
}

// Verbose exposes the verbose operation. Use this when interacting with the SDK through this surface.
func (l *LoadStrikeLogger) Verbose(format string, args ...any) { l.log("Verbose", format, args...) }
// Debug exposes the debug operation. Use this when interacting with the SDK through this surface.
func (l *LoadStrikeLogger) Debug(format string, args ...any)   { l.log("Debug", format, args...) }
// Information exposes the information operation. Use this when interacting with the SDK through this surface.
func (l *LoadStrikeLogger) Information(format string, args ...any) {
	l.log("Information", format, args...)
}
// Warning exposes the warning operation. Use this when interacting with the SDK through this surface.
func (l *LoadStrikeLogger) Warning(format string, args ...any) { l.log("Warning", format, args...) }
// Error returns the current error text. Use this when you need the readable failure message.
func (l *LoadStrikeLogger) Error(format string, args ...any)   { l.log("Error", format, args...) }
// Fatal exposes the fatal operation. Use this when interacting with the SDK through this surface.
func (l *LoadStrikeLogger) Fatal(format string, args ...any)   { l.log("Fatal", format, args...) }

// LoadStrikeScenarioInfo mirrors the .NET public runtime scenario metadata contract.
type LoadStrikeScenarioInfo struct {
	InstanceID        string                      `json:"InstanceId,omitempty"`
	InstanceId        string                      `json:"-"`
	InstanceNumber    int                         `json:"InstanceNumber,omitempty"`
	ScenarioDuration  time.Duration               `json:"ScenarioDuration,omitempty"`
	ScenarioName      string                      `json:"ScenarioName,omitempty"`
	ScenarioOperation LoadStrikeScenarioOperation `json:"ScenarioOperation,omitempty"`
}

func currentNodeInfo(nodeType NodeType, operation LoadStrikeOperationType) nodeInfo {
	return nodeInfo{
		NodeType:             nodeType,
		MachineName:          currentMachineName(),
		CurrentOperation:     operation.String(),
		CurrentOperationType: operation,
		CoresCount:           runtime.NumCPU(),
		EngineVersion:        Version(),
		OS:                   runtime.GOOS,
		Processor:            runtime.GOARCH,
	}
}

func currentTestInfo(context contextState, created time.Time) testInfo {
	return normalizeTestInfo(testInfo{
		TestSuite:  context.TestSuite,
		TestName:   context.TestName,
		SessionID:  context.SessionID,
		ClusterID:  context.ClusterID,
		CreatedUTC: created,
	})
}

func scenarioRuntimeInfo(name string, instanceNumber int, started time.Time, operation LoadStrikeScenarioOperation) LoadStrikeScenarioInfo {
	return normalizeScenarioInfo(LoadStrikeScenarioInfo{
		InstanceID:        fmt.Sprintf("%s-%d", name, instanceNumber),
		InstanceNumber:    instanceNumber,
		ScenarioDuration:  time.Since(started),
		ScenarioName:      name,
		ScenarioOperation: operation,
	})
}

func newLoadStrikeBaseContext(context *contextState) LoadStrikeBaseContext {
	return loadStrikeBaseContext{
		native: context,
	}
}

func newScenarioCancellationToken() stdcontext.Context {
	return stdcontext.Background()
}

func newScenarioRandom(invocationNumber int) *LoadStrikeRandom {
	source := rand.NewSource(time.Now().UnixNano() + int64(invocationNumber))
	return &LoadStrikeRandom{inner: rand.New(source)}
}

// String returns a string representation. Use this for logs, debugging, or display.
func (o LoadStrikeOperationType) String() string {
	switch o {
	case LoadStrikeOperationTypeInit:
		return "Init"
	case LoadStrikeOperationTypeWarmUp:
		return "WarmUp"
	case LoadStrikeOperationTypeBombing:
		return "Bombing"
	case LoadStrikeOperationTypeStop:
		return "Stop"
	case LoadStrikeOperationTypeComplete:
		return "Complete"
	case LoadStrikeOperationTypeError:
		return "Error"
	default:
		return "None"
	}
}

// Next exposes the next operation. Use this when interacting with the SDK through this surface.
func (r *LoadStrikeRandom) Next(args ...int) int {
	if r == nil || r.inner == nil {
		return 0
	}
	switch len(args) {
	case 0:
		return r.inner.Int()
	case 1:
		if args[0] <= 0 {
			return 0
		}
		return r.inner.Intn(args[0])
	default:
		minimum := args[0]
		maximum := args[1]
		if maximum <= minimum {
			return minimum
		}
		return minimum + r.inner.Intn(maximum-minimum)
	}
}

// NextDouble exposes the next double operation. Use this when interacting with the SDK through this surface.
func (r *LoadStrikeRandom) NextDouble() float64 {
	if r == nil || r.inner == nil {
		return 0
	}
	return r.inner.Float64()
}

// NextBytes exposes the next bytes operation. Use this when interacting with the SDK through this surface.
func (r *LoadStrikeRandom) NextBytes(buffer []byte) {
	if r == nil || r.inner == nil || len(buffer) == 0 {
		return
	}
	_, _ = r.inner.Read(buffer)
}

// Sample exposes the sample operation. Use this when interacting with the SDK through this surface.
func (r *LoadStrikeRandom) Sample(values any) any {
	if r == nil || r.inner == nil || values == nil {
		return nil
	}
	collection := reflect.ValueOf(values)
	if collection.Kind() != reflect.Slice && collection.Kind() != reflect.Array {
		panic("Sample requires a slice or array.")
	}
	if collection.Len() == 0 {
		return nil
	}
	return collection.Index(r.inner.Intn(collection.Len())).Interface()
}
