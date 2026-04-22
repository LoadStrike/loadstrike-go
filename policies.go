package loadstrike

// LoadStrikeScenarioRuntime mirrors the .NET runtime-policy scenario summary contract.
type LoadStrikeScenarioRuntime struct {
	ScenarioName    string         `json:"ScenarioName,omitempty"`
	AllRequestCount int            `json:"AllRequestCount,omitempty"`
	AllOKCount      int            `json:"AllOkCount,omitempty"`
	AllFailCount    int            `json:"AllFailCount,omitempty"`
	TotalBytes      int64          `json:"TotalBytes,omitempty"`
	TotalLatencyMS  float64        `json:"TotalLatencyMs,omitempty"`
	AvgLatencyMS    float64        `json:"AvgLatencyMs,omitempty"`
	MinLatencyMS    float64        `json:"MinLatencyMs,omitempty"`
	MaxLatencyMS    float64        `json:"MaxLatencyMs,omitempty"`
	StatusCodes     map[string]int `json:"StatusCodes,omitempty"`
}

// LoadStrikeRuntimePolicy mirrors the .NET public runtime-policy contract.
type LoadStrikeRuntimePolicy interface {
	ShouldRunScenario(string) LoadStrikeBoolTask
	BeforeScenario(string) LoadStrikeTask
	AfterScenario(string, LoadStrikeScenarioRuntime) LoadStrikeTask
	BeforeStep(string, string) LoadStrikeTask
	AfterStep(string, string, LoadStrikeReply) LoadStrikeTask
}

// ILoadStrikeRuntimePolicy mirrors the .NET interface name.
type ILoadStrikeRuntimePolicy = LoadStrikeRuntimePolicy

// LoadStrikeRuntimePolicyBase provides default no-op hook behavior,
// mirroring .NET default interface members in the closest valid Go form.
type LoadStrikeRuntimePolicyBase struct{}

func (LoadStrikeRuntimePolicyBase) ShouldRunScenario(string) LoadStrikeBoolTask {
	return TaskFromBool(true)
}

func (LoadStrikeRuntimePolicyBase) BeforeScenario(string) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeRuntimePolicyBase) AfterScenario(string, LoadStrikeScenarioRuntime) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeRuntimePolicyBase) BeforeStep(string, string) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeRuntimePolicyBase) AfterStep(string, string, LoadStrikeReply) LoadStrikeTask {
	return CompletedTask()
}

type loadStrikeRuntimePolicyBase = LoadStrikeRuntimePolicyBase

// runtimePolicy preserves the older native Go hook surface for internal compatibility.
type runtimePolicy interface {
	ShouldRunScenario(*scenarioHookContext) bool
	BeforeScenario(*scenarioHookContext) error
	AfterScenario(*scenarioHookContext, scenarioStats) error
	BeforeStep(*stepRuntimeContext) error
	AfterStep(*stepRuntimeContext, replyResult) error
}

type loadStrikeAsyncRuntimePolicy interface {
	ShouldRunScenarioAsync(string) LoadStrikeBoolTask
	BeforeScenarioAsync(string) LoadStrikeTask
	AfterScenarioAsync(string, LoadStrikeScenarioRuntime) LoadStrikeTask
	BeforeStepAsync(string, string) LoadStrikeTask
	AfterStepAsync(string, string, LoadStrikeReply) LoadStrikeTask
}

type legacyLoadStrikeRuntimePolicy interface {
	ShouldRunScenario(string) bool
	BeforeScenario(string) error
	AfterScenario(string, LoadStrikeScenarioRuntime) error
	BeforeStep(string, string) error
	AfterStep(string, string, LoadStrikeReply) error
}

type loadStrikeTaskRuntimePolicyShouldRun interface {
	ShouldRunScenario(string) LoadStrikeBoolTask
}

type loadStrikeAsyncRuntimePolicyShouldRun interface {
	ShouldRunScenarioAsync(string) LoadStrikeBoolTask
}

type legacyLoadStrikeRuntimePolicyShouldRun interface {
	ShouldRunScenario(string) bool
}

type loadStrikeTaskRuntimePolicyBeforeScenario interface {
	BeforeScenario(string) LoadStrikeTask
}

type loadStrikeAsyncRuntimePolicyBeforeScenario interface {
	BeforeScenarioAsync(string) LoadStrikeTask
}

type legacyLoadStrikeRuntimePolicyBeforeScenario interface {
	BeforeScenario(string) error
}

type loadStrikeTaskRuntimePolicyAfterScenario interface {
	AfterScenario(string, LoadStrikeScenarioRuntime) LoadStrikeTask
}

type loadStrikeAsyncRuntimePolicyAfterScenario interface {
	AfterScenarioAsync(string, LoadStrikeScenarioRuntime) LoadStrikeTask
}

type legacyLoadStrikeRuntimePolicyAfterScenario interface {
	AfterScenario(string, LoadStrikeScenarioRuntime) error
}

type loadStrikeTaskRuntimePolicyBeforeStep interface {
	BeforeStep(string, string) LoadStrikeTask
}

type loadStrikeAsyncRuntimePolicyBeforeStep interface {
	BeforeStepAsync(string, string) LoadStrikeTask
}

type legacyLoadStrikeRuntimePolicyBeforeStep interface {
	BeforeStep(string, string) error
}

type loadStrikeTaskRuntimePolicyAfterStep interface {
	AfterStep(string, string, LoadStrikeReply) LoadStrikeTask
}

type loadStrikeAsyncRuntimePolicyAfterStep interface {
	AfterStepAsync(string, string, LoadStrikeReply) LoadStrikeTask
}

type legacyLoadStrikeRuntimePolicyAfterStep interface {
	AfterStep(string, string, LoadStrikeReply) error
}

// RuntimePolicyError records policy-level execution errors.
type RuntimePolicyError struct {
	PolicyName   string `json:"PolicyName,omitempty"`
	CallbackName string `json:"CallbackName,omitempty"`
	ScenarioName string `json:"ScenarioName,omitempty"`
	StepName     string `json:"StepName,omitempty"`
	Message      string `json:"Message,omitempty"`
}
