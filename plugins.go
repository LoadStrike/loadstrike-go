package loadstrike

// LoadStrikeScenarioStartInfo mirrors the .NET public session start scenario contract.
type LoadStrikeScenarioStartInfo struct {
	native *scenarioStartInfo
}

type loadStrikeScenarioStartInfo = LoadStrikeScenarioStartInfo

// LoadStrikeSessionStartInfo mirrors the .NET public session start contract.
type LoadStrikeSessionStartInfo struct {
	native *sessionStartInfo
}

type loadStrikeSessionStartInfo = LoadStrikeSessionStartInfo

func newLoadStrikeScenarioStartInfo(native *scenarioStartInfo) LoadStrikeScenarioStartInfo {
	return loadStrikeScenarioStartInfo{native: native}
}

func newLoadStrikeSessionStartInfo(native *sessionStartInfo) LoadStrikeSessionStartInfo {
	return loadStrikeSessionStartInfo{native: native}
}

func (i loadStrikeScenarioStartInfo) ScenarioName() string {
	if i.native == nil {
		return ""
	}
	return i.native.ScenarioName
}

func (i loadStrikeScenarioStartInfo) SortIndex() int {
	if i.native == nil {
		return 0
	}
	return i.native.SortIndex
}

func (i loadStrikeSessionStartInfo) Scenarios() []LoadStrikeScenarioStartInfo {
	if i.native == nil || len(i.native.Scenarios) == 0 {
		return nil
	}
	scenarios := make([]LoadStrikeScenarioStartInfo, 0, len(i.native.Scenarios))
	for index := range i.native.Scenarios {
		scenarios = append(scenarios, newLoadStrikeScenarioStartInfo(&i.native.Scenarios[index]))
	}
	return scenarios
}

// LoadStrikeWorkerPlugin mirrors the .NET public worker-plugin contract.
type LoadStrikeWorkerPlugin interface {
	PluginName() string
	Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
	Start(LoadStrikeSessionStartInfo) LoadStrikeTask
	GetData(LoadStrikeRunResult) LoadStrikeValueTask[LoadStrikePluginData]
	Stop() LoadStrikeTask
	Dispose() LoadStrikeTask
}

// ILoadStrikeWorkerPlugin mirrors the .NET interface name.
type ILoadStrikeWorkerPlugin = LoadStrikeWorkerPlugin

// LoadStrikeWorkerPluginBase provides default no-op lifecycle behavior,
// mirroring .NET default interface members in the closest valid Go form.
type LoadStrikeWorkerPluginBase struct{}

func (LoadStrikeWorkerPluginBase) Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeWorkerPluginBase) Start(LoadStrikeSessionStartInfo) LoadStrikeTask {
	return CompletedTask()
}

func (LoadStrikeWorkerPluginBase) GetData(LoadStrikeRunResult) LoadStrikeValueTask[LoadStrikePluginData] {
	return TaskFromResult(LoadStrikePluginData{})
}

func (LoadStrikeWorkerPluginBase) Stop() LoadStrikeTask { return CompletedTask() }
func (LoadStrikeWorkerPluginBase) Dispose() LoadStrikeTask {
	return CompletedTask()
}

type loadStrikeWorkerPluginBase = LoadStrikeWorkerPluginBase

type loadStrikeAsyncWorkerPlugin interface {
	PluginName() string
	InitAsync(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
	StartAsync(LoadStrikeSessionStartInfo) LoadStrikeTask
	GetDataAsync(LoadStrikeRunResult) LoadStrikeValueTask[LoadStrikePluginData]
	StopAsync() LoadStrikeTask
}

type loadStrikeTaskWorkerPluginInitializer interface {
	Init(LoadStrikeBaseContext, IConfiguration) LoadStrikeTask
}

type loadStrikeTaskWorkerPluginStarter interface {
	Start(LoadStrikeSessionStartInfo) LoadStrikeTask
}

type loadStrikeTaskWorkerPluginDataProvider interface {
	GetData(LoadStrikeRunResult) LoadStrikeValueTask[LoadStrikePluginData]
}

type loadStrikeTaskWorkerPluginStopper interface {
	Stop() LoadStrikeTask
}

type legacyLoadStrikeWorkerPlugin interface {
	PluginName() string
	Init(LoadStrikeBaseContext, IConfiguration) error
	Start(LoadStrikeSessionStartInfo) error
	GetData(LoadStrikeRunResult) (LoadStrikePluginData, error)
	Stop() error
	Dispose() error
}

type workerPlugin interface {
	Name() string
	Init(LoadStrikeBaseContext, IConfiguration) error
	Start(LoadStrikeBaseContext) error
	GetData(runResult) (pluginData, error)
	Stop() error
	Dispose() error
}

type loadStrikeWorkerPluginDisposer interface {
	Dispose() LoadStrikeTask
}

type loadStrikeAsyncWorkerPluginDisposer interface {
	DisposeAsync() LoadStrikeTask
}

// workerPluginSpec defines an internal callback-backed or inline worker plugin contract.
type workerPluginSpec struct {
	loadStrikeWorkerPluginBase
	Name             string                                            `json:"PluginName"`
	CallbackURL      string                                            `json:"CallbackUrl"`
	InitFunc         func(LoadStrikeBaseContext, IConfiguration) error `json:"-"`
	StartFunc        func(LoadStrikeBaseContext) error                 `json:"-"`
	StartSessionFunc func(LoadStrikeSessionStartInfo) error            `json:"-"`
	GetDataFunc      func(runResult) (pluginData, error)               `json:"-"`
	StopFunc         func() error                                      `json:"-"`
	DisposeFunc      func() error                                      `json:"-"`
}

func (w workerPluginSpec) PluginName() string {
	return firstNonBlank(w.Name, "worker-plugin")
}

func (w workerPluginSpec) Init(context LoadStrikeBaseContext, infra IConfiguration) LoadStrikeTask {
	if w.InitFunc != nil {
		return TaskFromError(w.InitFunc(context, infra))
	}
	return CompletedTask()
}

func (w workerPluginSpec) Start(sessionInfo LoadStrikeSessionStartInfo) LoadStrikeTask {
	if w.StartSessionFunc != nil {
		return TaskFromError(w.StartSessionFunc(sessionInfo))
	}
	return CompletedTask()
}

func (w workerPluginSpec) GetData(result LoadStrikeRunResult) LoadStrikeValueTask[LoadStrikePluginData] {
	if w.GetDataFunc != nil {
		value, err := w.GetDataFunc(result.toNative())
		if err != nil {
			return TaskFromResultError[LoadStrikePluginData](err)
		}
		return TaskFromResult(newLoadStrikePluginData(value))
	}
	return TaskFromResult(LoadStrikePluginData{PluginName: w.PluginName()})
}

func (w workerPluginSpec) Stop() LoadStrikeTask {
	if w.StopFunc != nil {
		return TaskFromError(w.StopFunc())
	}
	return CompletedTask()
}

func (w workerPluginSpec) Dispose() LoadStrikeTask {
	if w.DisposeFunc != nil {
		return TaskFromError(w.DisposeFunc())
	}
	return CompletedTask()
}

func buildSessionStartInfo(context contextState, scenarios []scenarioDefinition) LoadStrikeSessionStartInfo {
	native := sessionStartInfo{
		Scenarios: make([]scenarioStartInfo, 0, len(scenarios)),
	}
	for index, scenario := range scenarios {
		native.Scenarios = append(native.Scenarios, scenarioStartInfo{
			ScenarioName: scenario.Name,
			SortIndex:    index,
		})
	}
	return newLoadStrikeSessionStartInfo(&native)
}

type sessionStartInfo struct {
	Scenarios []scenarioStartInfo
}

type scenarioStartInfo struct {
	ScenarioName string
	SortIndex    int
}
