package loadstrike

import (
	stdcontext "context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// TimeSpan mirrors the .NET public duration contract using Go's native duration type.
type TimeSpan = time.Duration

// LoggerConfiguration mirrors the .NET public logger-configuration concept.
type LoggerConfiguration map[string]any

// LoggerConfigurationFactory mirrors the .NET deferred logger configuration contract.
type LoggerConfigurationFactory func() LoggerConfiguration

// ILogger mirrors the .NET logger contract name in the closest valid Go form.
type ILogger interface {
	Verbose(format string, args ...any)
	Debug(format string, args ...any)
	Information(format string, args ...any)
	Warning(format string, args ...any)
	Error(format string, args ...any)
	Fatal(format string, args ...any)
}

// IConfiguration mirrors the .NET infra configuration concept using a named wrapper.
type IConfiguration struct {
	values map[string]any
}

func newIConfiguration(values map[string]any) IConfiguration {
	return IConfiguration{values: values}
}

func (c IConfiguration) Get(key string) any {
	if c.values == nil {
		return nil
	}
	return c.values[key]
}

func (c IConfiguration) Lookup(key string) (any, bool) {
	if c.values == nil {
		return nil, false
	}
	value, ok := c.values[key]
	return value, ok
}

func (c IConfiguration) Len() int {
	return len(c.values)
}

func (c IConfiguration) Values() map[string]any {
	return cloneAnyMap(c.values)
}

func (c IConfiguration) nativeValue() map[string]any {
	return c.values
}

// CancellationToken mirrors the .NET cancellation-token contract name using a named wrapper.
type CancellationToken struct {
	native stdcontext.Context
}

var _ stdcontext.Context = CancellationToken{}

func newCancellationToken(context stdcontext.Context) CancellationToken {
	return CancellationToken{native: context}
}

func (t CancellationToken) Deadline() (time.Time, bool) {
	if t.native == nil {
		return time.Time{}, false
	}
	return t.native.Deadline()
}

func (t CancellationToken) Done() <-chan struct{} {
	if t.native == nil {
		return nil
	}
	return t.native.Done()
}

func (t CancellationToken) Err() error {
	if t.native == nil {
		return nil
	}
	return t.native.Err()
}

func (t CancellationToken) Value(key any) any {
	if t.native == nil {
		return nil
	}
	return t.native.Value(key)
}

func (t CancellationToken) Context() stdcontext.Context {
	return t.native
}

// LoadStrikeBaseContext mirrors the .NET plugin and sink init contract.
type LoadStrikeBaseContext struct {
	native *contextState
}

type loadStrikeBaseContext = LoadStrikeBaseContext

// Logger returns the runtime logger when available.
func (c loadStrikeBaseContext) Logger() ILogger {
	if c.native == nil {
		return nil
	}
	return c.native.Logger
}

// NodeInfo returns runtime node information.
func (c loadStrikeBaseContext) NodeInfo() LoadStrikeNodeInfo {
	if c.native == nil {
		return LoadStrikeNodeInfo{}
	}
	return newLoadStrikeNodeInfo(c.native.nodeInfo)
}

// TestInfo returns runtime test information.
func (c loadStrikeBaseContext) TestInfo() LoadStrikeTestInfo {
	if c.native == nil {
		return LoadStrikeTestInfo{}
	}
	return newLoadStrikeTestInfo(c.native.testInfo)
}

// GetNodeInfo preserves the older Go accessor name on the wrapper surface.
func (c loadStrikeBaseContext) GetNodeInfo() LoadStrikeNodeInfo {
	if c.native == nil {
		return LoadStrikeNodeInfo{}
	}
	return newLoadStrikeNodeInfo(c.native.GetNodeInfo())
}

// LoadStrikeContext mirrors the .NET public context contract name.
type LoadStrikeContext struct {
	native *contextState
}

type loadStrikeContext = LoadStrikeContext

// LoadStrikeRunner mirrors the .NET public runner contract name.
type LoadStrikeRunner struct {
	native *runnerState
}

type loadStrikeRunner = LoadStrikeRunner

// LoadStrikeScenario mirrors the .NET public scenario contract name.
type LoadStrikeScenario struct {
	native *scenarioDefinition
}

type loadStrikeScenario = LoadStrikeScenario

// LoadStrikeNodeType mirrors the .NET public node-type contract name.
type LoadStrikeNodeType = NodeType

// LoadStrikeReportFormat mirrors the .NET public report-format contract name.
type LoadStrikeReportFormat = ReportFormat

// LoadStrikeRuntimePolicyErrorMode mirrors the .NET public runtime-policy error-mode contract name.
type LoadStrikeRuntimePolicyErrorMode = RuntimePolicyErrorMode

// LoadStrikeLatencyCount mirrors the .NET latency-count contract name.
type LoadStrikeLatencyCount struct {
	LessOrEq800     int `json:"LessOrEq800,omitempty"`
	More800Less1200 int `json:"More800Less1200,omitempty"`
	MoreOrEq1200    int `json:"MoreOrEq1200,omitempty"`
}

// LoadStrikeCounterStats mirrors the .NET counter-stats contract name.
type LoadStrikeCounterStats struct {
	MetricName    string `json:"MetricName,omitempty"`
	ScenarioName  string `json:"ScenarioName,omitempty"`
	UnitOfMeasure string `json:"UnitOfMeasure,omitempty"`
	Value         int64  `json:"Value,omitempty"`
}

// LoadStrikeGaugeStats mirrors the .NET gauge-stats contract name.
type LoadStrikeGaugeStats struct {
	MetricName    string  `json:"MetricName,omitempty"`
	ScenarioName  string  `json:"ScenarioName,omitempty"`
	UnitOfMeasure string  `json:"UnitOfMeasure,omitempty"`
	Value         float64 `json:"Value,omitempty"`
}

// LoadStrikeScenarioPartition mirrors the .NET scenario-partition contract name.
type LoadStrikeScenarioPartition struct {
	Number int `json:"Number,omitempty"`
	Count  int `json:"Count,omitempty"`
}

// LoadStrikeReply mirrors the .NET public reply contract name.
type LoadStrikeReply struct {
	native *replyResult
}

// LoadStrikeReplyWith mirrors the typed .NET reply helper shape using Go generics.
type LoadStrikeReplyWith[T any] struct {
	reply   LoadStrikeReply
	payload T
}

// LoadStrikeScenarioRunFunc mirrors the .NET sync scenario callback shape in the closest valid Go form.
type LoadStrikeScenarioRunFunc func(LoadStrikeScenarioContext) LoadStrikeReply

// LoadStrikeAsyncScenarioRunFunc mirrors the .NET async scenario callback shape in the closest valid Go form.
type LoadStrikeAsyncScenarioRunFunc func(LoadStrikeScenarioContext) LoadStrikeValueTask[LoadStrikeReply]

// LoadStrikeScenarioInitFunc mirrors the .NET sync init/clean callback shape in the closest valid Go form.
type LoadStrikeScenarioInitFunc func(LoadStrikeScenarioInitContext) error

// LoadStrikeAsyncScenarioInitFunc mirrors the .NET async init/clean callback shape in the closest valid Go form.
type LoadStrikeAsyncScenarioInitFunc func(LoadStrikeScenarioInitContext) LoadStrikeTask

// LoadStrikeStepRunFunc mirrors the .NET step callback shape in the closest valid Go form.
type LoadStrikeStepRunFunc func(LoadStrikeScenarioContext) LoadStrikeReply

// LoadStrikeAsyncStepRunFunc mirrors the .NET async step callback shape in the closest valid Go form.
type LoadStrikeAsyncStepRunFunc func(LoadStrikeScenarioContext) LoadStrikeValueTask[LoadStrikeReply]

// LoadStrikeLoadSimulation mirrors the .NET public load-simulation wrapper.
type LoadStrikeLoadSimulation struct {
	native LoadSimulation
}

// LoadStrikeThreshold mirrors the .NET public threshold wrapper.
type LoadStrikeThreshold struct {
	native ThresholdSpec
}

// AsReply returns the untyped reply projection.
func (r LoadStrikeReplyWith[T]) AsReply() LoadStrikeReply {
	return r.reply
}

func (r LoadStrikeReplyWith[T]) Payload() T {
	return r.payload
}

func newLoadStrikeLoadSimulation(native LoadSimulation) LoadStrikeLoadSimulation {
	return LoadStrikeLoadSimulation{native: native}
}

func (s LoadStrikeLoadSimulation) nativeValue() LoadSimulation {
	return s.native
}

func (s LoadStrikeLoadSimulation) IsInject() bool {
	return s.native.Kind == "Inject"
}

func (s LoadStrikeLoadSimulation) IsInjectRandom() bool {
	return s.native.Kind == "InjectRandom"
}

func (s LoadStrikeLoadSimulation) IsIterationsForConstant() bool {
	return s.native.Kind == "IterationsForConstant"
}

func (s LoadStrikeLoadSimulation) IsIterationsForInject() bool {
	return s.native.Kind == "IterationsForInject"
}

func (s LoadStrikeLoadSimulation) IsKeepConstant() bool {
	return s.native.Kind == "KeepConstant"
}

func (s LoadStrikeLoadSimulation) IsPause() bool {
	return s.native.Kind == "Pause"
}

func (s LoadStrikeLoadSimulation) IsRampingConstant() bool {
	return s.native.Kind == "RampingConstant"
}

func (s LoadStrikeLoadSimulation) IsRampingInject() bool {
	return s.native.Kind == "RampingInject"
}

func (s LoadStrikeLoadSimulation) String() string {
	return s.native.Kind
}

func newLoadStrikeThreshold(native ThresholdSpec) LoadStrikeThreshold {
	return LoadStrikeThreshold{native: native}
}

func (t LoadStrikeThreshold) nativeValue() ThresholdSpec {
	return t.native
}

func newLoadStrikeLatencyCount(native latencyCount) LoadStrikeLatencyCount {
	return LoadStrikeLatencyCount{
		LessOrEq800:     native.LessOrEq800,
		More800Less1200: native.More800Less1200,
		MoreOrEq1200:    native.MoreOrEq1200,
	}
}

func (c LoadStrikeLatencyCount) toNative() latencyCount {
	return latencyCount{
		LessOrEq800:     c.LessOrEq800,
		More800Less1200: c.More800Less1200,
		MoreOrEq1200:    c.MoreOrEq1200,
	}
}

func newLoadStrikeCounterStats(native counterStats) LoadStrikeCounterStats {
	return LoadStrikeCounterStats{
		MetricName:    native.MetricName,
		ScenarioName:  native.ScenarioName,
		UnitOfMeasure: native.UnitOfMeasure,
		Value:         native.Value,
	}
}

func (s LoadStrikeCounterStats) toNative() counterStats {
	return counterStats{
		MetricName:    s.MetricName,
		ScenarioName:  s.ScenarioName,
		UnitOfMeasure: s.UnitOfMeasure,
		Value:         s.Value,
	}
}

func newLoadStrikeGaugeStats(native gaugeStats) LoadStrikeGaugeStats {
	return LoadStrikeGaugeStats{
		MetricName:    native.MetricName,
		ScenarioName:  native.ScenarioName,
		UnitOfMeasure: native.UnitOfMeasure,
		Value:         native.Value,
	}
}

func (s LoadStrikeGaugeStats) toNative() gaugeStats {
	return gaugeStats{
		MetricName:    s.MetricName,
		ScenarioName:  s.ScenarioName,
		UnitOfMeasure: s.UnitOfMeasure,
		Value:         s.Value,
	}
}

func newLoadStrikeScenarioPartition(native scenarioPartitionInfo) LoadStrikeScenarioPartition {
	return LoadStrikeScenarioPartition{
		Number: native.Number,
		Count:  native.Count,
	}
}

func (p LoadStrikeScenarioPartition) toNative() scenarioPartitionInfo {
	return scenarioPartitionInfo{
		Number: p.Number,
		Count:  p.Count,
	}
}

func newLoadStrikeScenario(native scenarioDefinition) LoadStrikeScenario {
	value := native
	return loadStrikeScenario{
		native: &value,
	}
}

func (s loadStrikeScenario) Name() string {
	if s.native == nil {
		return ""
	}
	return s.native.Name
}

// Create mirrors the .NET scenario factory in the closest valid Go form.
func (loadStrikeScenario) Create(name string, run LoadStrikeScenarioRunFunc) LoadStrikeScenario {
	return CreateScenario(name, run)
}

// CreateAsync mirrors the .NET async scenario factory in the closest valid Go form.
func (loadStrikeScenario) CreateAsync(name string, run LoadStrikeAsyncScenarioRunFunc) LoadStrikeScenario {
	return CreateScenarioAsync(name, run)
}

// Empty mirrors the .NET empty scenario factory in the closest valid Go form.
func (loadStrikeScenario) Empty(name string) LoadStrikeScenario {
	return EmptyScenario(name)
}

func (s loadStrikeScenario) nativeValue() scenarioDefinition {
	if s.native == nil {
		return scenarioDefinition{}
	}
	return *s.native
}

func (s loadStrikeScenario) WithInit(init LoadStrikeScenarioInitFunc) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithInit(init))
}

func (s loadStrikeScenario) WithInitAsync(init LoadStrikeAsyncScenarioInitFunc) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithInit(init))
}

func (s loadStrikeScenario) WithClean(clean LoadStrikeScenarioInitFunc) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithClean(clean))
}

func (s loadStrikeScenario) WithCleanAsync(clean LoadStrikeAsyncScenarioInitFunc) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithClean(clean))
}

func (s loadStrikeScenario) WithWeight(weight int) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithWeight(weight))
}

func (s loadStrikeScenario) WithRestartIterationOnFail(enabled bool) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithRestartIterationOnFail(enabled))
}

func (s loadStrikeScenario) WithMaxFailCount(maxFailCount int) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithMaxFailCount(maxFailCount))
}

func (s loadStrikeScenario) WithWarmUpDuration(seconds float64) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithWarmUpDuration(seconds))
}

func (s loadStrikeScenario) WithoutWarmUp() LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithoutWarmUp())
}

func (s loadStrikeScenario) WithLoadSimulations(loadSimulations ...LoadStrikeLoadSimulation) LoadStrikeScenario {
	native := make([]LoadSimulation, 0, len(loadSimulations))
	for _, simulation := range loadSimulations {
		native = append(native, simulation.nativeValue())
	}
	return newLoadStrikeScenario(s.nativeValue().WithLoadSimulations(native...))
}

func (s loadStrikeScenario) WithThresholds(thresholds ...LoadStrikeThreshold) LoadStrikeScenario {
	native := make([]ThresholdSpec, 0, len(thresholds))
	for _, threshold := range thresholds {
		native = append(native, threshold.nativeValue())
	}
	return newLoadStrikeScenario(s.nativeValue().WithThresholds(native...))
}

func (s loadStrikeScenario) WithTrackingConfiguration(tracking *TrackingConfigurationSpec) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithTrackingConfiguration(tracking))
}

func (s loadStrikeScenario) WithCrossPlatformTracking(tracking *TrackingConfigurationSpec) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithCrossPlatformTracking(tracking))
}

func (s loadStrikeScenario) withSteps(steps ...any) LoadStrikeScenario {
	return newLoadStrikeScenario(s.nativeValue().WithSteps(normalizePublicSteps(steps...)...))
}

func newLoadStrikeReply(native replyResult) LoadStrikeReply {
	value := normalizeReplyValue(native)
	return LoadStrikeReply{
		native: &value,
	}
}

func (r LoadStrikeReply) toNative() replyResult {
	if r.native == nil {
		return replyResult{}
	}
	return normalizeReplyValue(*r.native)
}

func (r LoadStrikeReply) IsError() bool {
	return r.toNative().IsError
}

func (r LoadStrikeReply) StatusCode() string {
	return r.toNative().StatusCode
}

func (r LoadStrikeReply) Message() string {
	return r.toNative().Message
}

func (r LoadStrikeReply) SizeBytes() int64 {
	return r.toNative().SizeBytes
}

func (r LoadStrikeReply) CustomLatencyMS() float64 {
	return r.toNative().CustomLatencyMS
}

// AsReply returns the untyped reply projection.
func (r LoadStrikeReply) AsReply() LoadStrikeReply {
	return newLoadStrikeReply(r.toNative())
}

func (LoadStrikeThreshold) CreateScenario(args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(ScenarioThreshold(args...))
}

func (LoadStrikeThreshold) CreateStep(stepName string, args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(StepThreshold(stepName, args...))
}

func (LoadStrikeThreshold) CreateMetric(args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(MetricThreshold(args...))
}

func (LoadStrikeThreshold) ScenarioPredicate(args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(ScenarioThreshold(args...))
}

func (LoadStrikeThreshold) StepPredicate(stepName string, args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(StepThreshold(stepName, args...))
}

func (LoadStrikeThreshold) MetricPredicate(args ...any) LoadStrikeThreshold {
	return newLoadStrikeThreshold(MetricThreshold(args...))
}

// LoadStrikeScenarioContext mirrors the .NET public step-helper context name.
type LoadStrikeScenarioContext struct {
	native *stepRuntimeContext
}

type loadStrikeScenarioContext = LoadStrikeScenarioContext

func newLoadStrikeScenarioContext(context *stepRuntimeContext) LoadStrikeScenarioContext {
	if context == nil {
		return loadStrikeScenarioContext{}
	}
	return loadStrikeScenarioContext{
		native: context,
	}
}

func (c loadStrikeScenarioContext) Data() map[string]any {
	if c.native == nil {
		return nil
	}
	return c.native.Data
}

func (c loadStrikeScenarioContext) InvocationNumber() int64 {
	if c.native == nil {
		return 0
	}
	return c.native.InvocationNumber
}

func (c loadStrikeScenarioContext) Logger() ILogger {
	if c.native == nil {
		return nil
	}
	return c.native.Logger
}

func (c loadStrikeScenarioContext) NodeInfo() LoadStrikeNodeInfo {
	if c.native == nil {
		return LoadStrikeNodeInfo{}
	}
	return newLoadStrikeNodeInfo(c.native.nodeInfo)
}

func (c loadStrikeScenarioContext) Random() *LoadStrikeRandom {
	if c.native == nil {
		return nil
	}
	return c.native.Random
}

func (c loadStrikeScenarioContext) ScenarioCancellationToken() CancellationToken {
	if c.native == nil {
		return CancellationToken{}
	}
	return newCancellationToken(c.native.ScenarioCancellationToken)
}

func (c loadStrikeScenarioContext) ScenarioInfo() LoadStrikeScenarioInfo {
	if c.native == nil {
		return LoadStrikeScenarioInfo{}
	}
	return c.native.ScenarioInfo
}

func (c loadStrikeScenarioContext) ScenarioInstanceData() map[string]any {
	if c.native == nil {
		return nil
	}
	return c.native.ScenarioInstanceData
}

func (c loadStrikeScenarioContext) TestInfo() LoadStrikeTestInfo {
	if c.native == nil {
		return LoadStrikeTestInfo{}
	}
	return newLoadStrikeTestInfo(c.native.testInfo)
}

// GetScenarioTimerTime mirrors the .NET runtime scenario timer helper.
func (c loadStrikeScenarioContext) GetScenarioTimerTime() time.Duration {
	if c.native == nil {
		return 0
	}
	return c.native.GetScenarioTimerTime()
}

// StopCurrentTest mirrors the .NET runtime test-stop helper.
func (c loadStrikeScenarioContext) StopCurrentTest(reason string) {
	if c.native == nil {
		return
	}
	c.native.StopCurrentTest(reason)
}

// StopScenario mirrors the .NET runtime scenario-stop helper.
func (c loadStrikeScenarioContext) StopScenario(scenarioName string, reason string) {
	if c.native == nil {
		return
	}
	c.native.StopScenario(scenarioName, reason)
}

// LoadStrikeScenarioInitContext mirrors the .NET public init-hook context name.
type LoadStrikeScenarioInitContext struct {
	native *scenarioHookContext
}

type loadStrikeScenarioInitContext = LoadStrikeScenarioInitContext

func newLoadStrikeScenarioInitContext(context *scenarioHookContext) LoadStrikeScenarioInitContext {
	if context == nil {
		return loadStrikeScenarioInitContext{}
	}
	context.scenarioPartitionInfo = context.Partition
	return loadStrikeScenarioInitContext{
		native: context,
	}
}

func (c loadStrikeScenarioInitContext) CustomSettings() IConfiguration {
	if c.native == nil {
		return IConfiguration{}
	}
	return c.native.CustomSettings
}

func (c loadStrikeScenarioInitContext) GlobalCustomSettings() IConfiguration {
	if c.native == nil {
		return IConfiguration{}
	}
	return c.native.GlobalCustomSettings
}

func (c loadStrikeScenarioInitContext) Logger() ILogger {
	if c.native == nil {
		return nil
	}
	return c.native.Logger
}

func (c loadStrikeScenarioInitContext) NodeInfo() LoadStrikeNodeInfo {
	if c.native == nil {
		return LoadStrikeNodeInfo{}
	}
	return newLoadStrikeNodeInfo(c.native.nodeInfo)
}

func (c loadStrikeScenarioInitContext) ScenarioInfo() LoadStrikeScenarioInfo {
	if c.native == nil {
		return LoadStrikeScenarioInfo{}
	}
	return c.native.ScenarioInfo
}

func (c loadStrikeScenarioInitContext) ScenarioPartition() LoadStrikeScenarioPartition {
	if c.native == nil {
		return LoadStrikeScenarioPartition{}
	}
	return newLoadStrikeScenarioPartition(c.native.scenarioPartitionInfo)
}

func (c loadStrikeScenarioInitContext) TestInfo() LoadStrikeTestInfo {
	if c.native == nil {
		return LoadStrikeTestInfo{}
	}
	return newLoadStrikeTestInfo(c.native.testInfo)
}

// RegisterMetric mirrors the .NET init-context metric registration contract.
func (c loadStrikeScenarioInitContext) RegisterMetric(metric IMetric) {
	if c.native == nil {
		panic("Metric must be provided.")
	}
	c.native.registerMetric(metric)
}

// LogEventLevel mirrors the .NET public log-level contract.
type LogEventLevel string

const (
	LogEventLevelVerbose     LogEventLevel = "Verbose"
	LogEventLevelDebug       LogEventLevel = "Debug"
	LogEventLevelInformation LogEventLevel = "Information"
	LogEventLevelWarning     LogEventLevel = "Warning"
	LogEventLevelError       LogEventLevel = "Error"
	LogEventLevelFatal       LogEventLevel = "Fatal"
)

// DurationFromSeconds creates a TimeSpan-style duration from seconds.
func DurationFromSeconds(seconds float64) TimeSpan {
	return time.Duration(seconds * float64(time.Second))
}

// DurationFromMilliseconds creates a TimeSpan-style duration from milliseconds.
func DurationFromMilliseconds(milliseconds float64) TimeSpan {
	return time.Duration(milliseconds * float64(time.Millisecond))
}

// Empty mirrors the .NET Empty scenario constructor.
func Empty(name string) LoadStrikeScenario {
	return EmptyScenario(name)
}

type loadStrikeSimulationNamespace struct{}

// LoadStrikeSimulation mirrors the .NET public load-simulation helper namespace.
var LoadStrikeSimulation loadStrikeSimulationNamespace

func (loadStrikeSimulationNamespace) Inject(rate int, interval TimeSpan, during TimeSpan) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(Inject(rate, interval.Seconds(), during.Seconds()))
}

func (loadStrikeSimulationNamespace) InjectRandom(minRate int, maxRate int, interval TimeSpan, during TimeSpan) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(InjectRandom(minRate, maxRate, interval.Seconds(), during.Seconds()))
}

func (loadStrikeSimulationNamespace) IterationsForConstant(copies int, iterations int) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(IterationsForConstant(copies, iterations))
}

func (loadStrikeSimulationNamespace) IterationsForInject(rate int, interval TimeSpan, iterations int) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(IterationsForInject(rate, interval.Seconds(), iterations))
}

func (loadStrikeSimulationNamespace) KeepConstant(copies int, during TimeSpan) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(KeepConstant(copies, during.Seconds()))
}

func (loadStrikeSimulationNamespace) Pause(during TimeSpan) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(Pause(during.Seconds()))
}

func (loadStrikeSimulationNamespace) RampingConstant(copies int, during TimeSpan) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(RampingConstant(copies, during.Seconds()))
}

func (loadStrikeSimulationNamespace) RampingInject(rate int, interval TimeSpan, during TimeSpan) LoadStrikeLoadSimulation {
	return newLoadStrikeLoadSimulation(RampingInject(rate, interval.Seconds(), during.Seconds()))
}

type loadStrikeResponseNamespace struct{}

// LoadStrikeResponse mirrors the .NET public response helper namespace.
var LoadStrikeResponse loadStrikeResponseNamespace

func (loadStrikeResponseNamespace) Ok(args ...any) LoadStrikeReply {
	return newLoadStrikeReply(dotnetStyleReply(true, args...))
}

func (loadStrikeResponseNamespace) Fail(args ...any) LoadStrikeReply {
	return newLoadStrikeReply(dotnetStyleReply(false, args...))
}

func (loadStrikeResponseNamespace) OkWith(payload any, args ...any) LoadStrikeReplyWith[any] {
	reply := dotnetStyleReply(true, append([]any{payload}, args...)...)
	return normalizeLoadStrikeReplyWith(reply, payload)
}

func (loadStrikeResponseNamespace) FailWith(payload any, args ...any) LoadStrikeReplyWith[any] {
	reply := dotnetStyleReply(false, append([]any{payload}, args...)...)
	return normalizeLoadStrikeReplyWith(reply, payload)
}

type loadStrikeStepNamespace struct{}

// LoadStrikeStep mirrors the .NET public step helper namespace.
var LoadStrikeStep loadStrikeStepNamespace

func (loadStrikeStepNamespace) Run(name string, context LoadStrikeScenarioContext, run LoadStrikeStepRunFunc) LoadStrikeReply {
	if strings.TrimSpace(name) == "" {
		panic("step name must be provided.")
	}
	if run == nil {
		panic("step run must be provided.")
	}
	scenarioContext := requireLoadStrikeScenarioContext(context)
	if scenarioContext.native.StepName == "" {
		scenarioContext.native.StepName = name
	}
	return run(scenarioContext)
}

func (loadStrikeStepNamespace) RunAsync(name string, context LoadStrikeScenarioContext, run LoadStrikeAsyncStepRunFunc) LoadStrikeReply {
	if strings.TrimSpace(name) == "" {
		panic("step name must be provided.")
	}
	if run == nil {
		panic("step run must be provided.")
	}
	return LoadStrikeStep.Run(name, context, func(context LoadStrikeScenarioContext) LoadStrikeReply {
		reply, err := run(context).Await()
		if err != nil {
			return newLoadStrikeReply(failReply("step_exception", err.Error()))
		}
		return reply
	})
}

func requireLoadStrikeScenarioContext(context LoadStrikeScenarioContext) LoadStrikeScenarioContext {
	if context.native == nil {
		panic("step context must be provided.")
	}
	return context
}

func dotnetStyleReply(success bool, args ...any) replyResult {
	if len(args) == 0 {
		if success {
			return okReply()
		}
		return failReply()
	}

	replyArgs := args
	var payload any
	if shouldTreatAsPayload(args[0]) {
		payload = args[0]
		replyArgs = args[1:]
	}

	var reply replyResult
	if success {
		reply = okReply(replyArgs...)
	} else {
		reply = failReply(replyArgs...)
	}
	if payload != nil {
		reply.Payload = payload
	}
	return reply
}

var replyStatusCodePattern = regexp.MustCompile(`^\d{3}$`)

func shouldTreatAsPayload(value any) bool {
	if value == nil {
		return false
	}

	if text, ok := value.(string); ok {
		trimmed := strings.TrimSpace(text)
		return !replyStatusCodePattern.MatchString(trimmed)
	}

	return true
}

func invokeReplyCallable(callable reflect.Value, inputs []reflect.Value, allowErrorTuple bool) replyResult {
	results := callable.Call(inputs)
	switch len(results) {
	case 1:
		return unwrapReplyValue(results[0])
	case 2:
		if !allowErrorTuple {
			panic("Step run must return a reply.")
		}
		err := unwrapErrorValue(results[1])
		if err != nil {
			return failReply("500", err.Error())
		}
		return unwrapReplyValue(results[0])
	default:
		panic("Step run must return a reply.")
	}
}

func unwrapReplyValue(value reflect.Value) replyResult {
	if !value.IsValid() {
		panic("Step run must return a reply.")
	}

	if await := value.MethodByName("Await"); await.IsValid() && await.Type().NumIn() == 0 {
		results := await.Call(nil)
		switch len(results) {
		case 1:
			if err := unwrapErrorValue(results[0]); err != nil {
				panic(err.Error())
			}
			panic("Step run must return a reply.")
		case 2:
			if err := unwrapErrorValue(results[1]); err != nil {
				panic(err.Error())
			}
			return unwrapReplyValue(results[0])
		}
	}

	replyType := reflect.TypeOf(replyResult{})
	loadStrikeReplyType := reflect.TypeOf(LoadStrikeReply{})
	if value.Type() == replyType {
		return normalizeReplyValue(value.Interface().(replyResult))
	}
	if value.Type() == loadStrikeReplyType {
		return value.Interface().(LoadStrikeReply).toNative()
	}

	if value.Kind() == reflect.Struct {
		if asReply := value.MethodByName("AsReply"); asReply.IsValid() && asReply.Type().NumIn() == 0 && asReply.Type().NumOut() == 1 {
			results := asReply.Call(nil)
			if len(results) == 1 {
				switch results[0].Type() {
				case replyType:
					return normalizeReplyValue(results[0].Interface().(replyResult))
				case loadStrikeReplyType:
					return results[0].Interface().(LoadStrikeReply).toNative()
				}
			}
		}
		field := value.FieldByName("Reply")
		if field.IsValid() && field.Type() == replyType {
			return normalizeReplyValue(field.Interface().(replyResult))
		}
		if field.IsValid() && field.Type() == loadStrikeReplyType {
			return field.Interface().(LoadStrikeReply).toNative()
		}
	}

	panic(fmt.Sprintf("Step run returned unsupported reply type %s", value.Type()))
}

func unwrapErrorValue(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !value.Type().Implements(errorType) {
		return nil
	}
	if value.IsNil() {
		return nil
	}
	err, _ := value.Interface().(error)
	return err
}

func normalizeReplyValue(reply replyResult) replyResult {
	reply.IsError = !reply.IsSuccess
	return reply
}

func normalizeLoadStrikeReplyWith[T any](reply replyResult, payload T) LoadStrikeReplyWith[T] {
	reply = normalizeReplyValue(reply)
	return LoadStrikeReplyWith[T]{
		reply:   newLoadStrikeReply(reply),
		payload: payload,
	}
}
