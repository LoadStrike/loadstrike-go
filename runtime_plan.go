package loadstrike

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

type runtimePlan struct {
	SDKVersion      string                `json:"sdkVersion"`
	ProtocolVersion int                   `json:"protocolVersion"`
	Context         runtimeContextPlan    `json:"context"`
	Scenarios       []runtimeScenarioPlan `json:"scenarios"`
}

type runtimeContextPlan struct {
	ReportsEnabled                   bool                       `json:"reportsEnabled"`
	RestartIterationMaxAttempts      int                        `json:"restartIterationMaxAttempts,omitempty"`
	ReportingIntervalSeconds         float64                    `json:"reportingIntervalSeconds,omitempty"`
	LicenseValidationTimeoutSeconds  float64                    `json:"licenseValidationTimeoutSeconds,omitempty"`
	MinimumLogLevel                  string                     `json:"minimumLogLevel,omitempty"`
	LoggerConfig                     map[string]any             `json:"loggerConfig,omitempty"`
	RunnerKey                        string                     `json:"runnerKey"`
	SessionID                        string                     `json:"sessionId,omitempty"`
	ClusterID                        string                     `json:"clusterId,omitempty"`
	AgentGroup                       string                     `json:"agentGroup,omitempty"`
	NatsServerURL                    string                     `json:"natsServerUrl,omitempty"`
	NodeType                         NodeType                   `json:"nodeType,omitempty"`
	AgentsCount                      int                        `json:"agentsCount,omitempty"`
	LocalDevClusterEnabled           bool                       `json:"localDevClusterEnabled,omitempty"`
	TargetScenarios                  []string                   `json:"targetScenarios,omitempty"`
	AgentTargetScenarios             []string                   `json:"agentTargetScenarios,omitempty"`
	CoordinatorTargetScenarios       []string                   `json:"coordinatorTargetScenarios,omitempty"`
	ScenarioCompletionTimeoutSeconds float64                    `json:"scenarioCompletionTimeoutSeconds,omitempty"`
	ClusterCommandTimeoutSeconds     float64                    `json:"clusterCommandTimeoutSeconds,omitempty"`
	ConsoleMetricsEnabled            bool                       `json:"consoleMetricsEnabled"`
	InfraConfigPath                  string                     `json:"infraConfigPath,omitempty"`
	ReportingSinks                   []runtimeReportingSinkPlan `json:"reportingSinks,omitempty"`
	WorkerPlugins                    []runtimeWorkerPluginPlan  `json:"workerPlugins,omitempty"`
	RuntimePolicies                  []runtimePolicyPlan        `json:"runtimePolicies,omitempty"`
	RuntimePolicyErrorMode           RuntimePolicyErrorMode     `json:"runtimePolicyErrorMode,omitempty"`
	ReportFileName                   string                     `json:"reportFileName,omitempty"`
	ReportFolder                     string                     `json:"reportFolder,omitempty"`
	ReportFormats                    []ReportFormat             `json:"reportFormats,omitempty"`
	TestSuite                        string                     `json:"testSuite,omitempty"`
	TestName                         string                     `json:"testName,omitempty"`
}

type runtimeReportingSinkPlan struct {
	Kind          string                    `json:"kind"`
	Name          string                    `json:"name,omitempty"`
	CallbackURL   string                    `json:"callbackUrl,omitempty"`
	InfluxDB      *InfluxDBSinkOptions      `json:"influxDb,omitempty"`
	TimescaleDB   *TimescaleDBSinkOptions   `json:"timescaleDb,omitempty"`
	GrafanaLoki   *GrafanaLokiSinkOptions   `json:"grafanaLoki,omitempty"`
	Datadog       *DatadogSinkOptions       `json:"datadog,omitempty"`
	Splunk        *SplunkSinkOptions        `json:"splunk,omitempty"`
	OTELCollector *OTELCollectorSinkOptions `json:"otelCollector,omitempty"`
}

type runtimeWorkerPluginPlan struct {
	Name        string `json:"name"`
	CallbackURL string `json:"callbackUrl"`
}

type runtimePolicyPlan struct {
	Name        string `json:"name"`
	CallbackURL string `json:"callbackUrl"`
}

type runtimeScenarioPlan struct {
	Name                   string                     `json:"name"`
	StepCallbackURL        string                     `json:"stepCallbackUrl,omitempty"`
	InitCallbackURL        string                     `json:"initCallbackUrl,omitempty"`
	CleanCallbackURL       string                     `json:"cleanCallbackUrl,omitempty"`
	MetricsCallbackURL     string                     `json:"metricsCallbackUrl,omitempty"`
	Weight                 int                        `json:"weight,omitempty"`
	RestartIterationOnFail bool                       `json:"restartIterationOnFail,omitempty"`
	MaxFailCount           int                        `json:"maxFailCount,omitempty"`
	WarmUpDurationSeconds  float64                    `json:"warmUpDurationSeconds,omitempty"`
	WarmUpDisabled         bool                       `json:"warmUpDisabled,omitempty"`
	LoadSimulations        []LoadSimulation           `json:"loadSimulations,omitempty"`
	Thresholds             []ThresholdSpec            `json:"thresholds,omitempty"`
	Tracking               *TrackingConfigurationSpec `json:"tracking,omitempty"`
}

func newRuntimeContextPlan(context contextState) runtimeContextPlan {
	return runtimeContextPlan{
		ReportsEnabled:                   context.ReportsEnabled,
		RestartIterationMaxAttempts:      context.RestartIterationMaxAttempts,
		ReportingIntervalSeconds:         context.ReportingIntervalSeconds,
		LicenseValidationTimeoutSeconds:  context.LicenseValidationTimeoutSeconds,
		MinimumLogLevel:                  context.MinimumLogLevel,
		LoggerConfig:                     cloneAnyMap(context.LoggerConfig),
		RunnerKey:                        context.RunnerKey,
		SessionID:                        context.SessionID,
		ClusterID:                        context.ClusterID,
		AgentGroup:                       context.AgentGroup,
		NatsServerURL:                    context.NatsServerURL,
		NodeType:                         context.NodeType,
		AgentsCount:                      context.AgentsCount,
		LocalDevClusterEnabled:           context.LocalDevClusterEnabled,
		TargetScenarios:                  append([]string(nil), context.TargetScenarios...),
		AgentTargetScenarios:             append([]string(nil), context.AgentTargetScenarios...),
		CoordinatorTargetScenarios:       append([]string(nil), context.CoordinatorTargetScenarios...),
		ScenarioCompletionTimeoutSeconds: context.ScenarioCompletionTimeoutSeconds,
		ClusterCommandTimeoutSeconds:     context.ClusterCommandTimeoutSeconds,
		ConsoleMetricsEnabled:            context.ConsoleMetricsEnabled,
		InfraConfigPath:                  context.InfraConfigPath,
		RuntimePolicyErrorMode:           context.RuntimePolicyErrorMode,
		ReportFileName:                   context.ReportFileName,
		ReportFolder:                     context.ReportFolder,
		ReportFormats:                    append([]ReportFormat(nil), context.ReportFormats...),
		TestSuite:                        context.TestSuite,
		TestName:                         context.TestName,
	}
}

func buildRuntimePlan(
	context *contextState,
	registry *runtimeCallbackRegistry,
	httpHost *runtimeHTTPHostHandle,
) (runtimePlan, error) {
	if context == nil {
		return runtimePlan{}, errors.New("context must be provided")
	}
	if registry == nil {
		return runtimePlan{}, errors.New("runtime callback registry must be provided")
	}
	if httpHost == nil {
		return runtimePlan{}, errors.New("runtime http host must be provided")
	}

	plan := runtimePlan{
		SDKVersion:      RuntimeArtifactVersion(),
		ProtocolVersion: RuntimeProtocolVersion(),
		Context:         newRuntimeContextPlan(*context),
		Scenarios:       make([]runtimeScenarioPlan, 0, len(context.scenarios)),
	}

	contextPlan, err := buildRuntimeContextExtensions(*context, registry, httpHost)
	if err != nil {
		return runtimePlan{}, err
	}
	plan.Context.ReportingSinks = contextPlan.ReportingSinks
	plan.Context.WorkerPlugins = contextPlan.WorkerPlugins
	plan.Context.RuntimePolicies = contextPlan.RuntimePolicies

	for _, scenario := range context.scenarios {
		scenarioID := registry.registerScenario(scenario)
		item := runtimeScenarioPlan{
			Name:                   scenario.Name,
			Weight:                 scenario.Weight,
			RestartIterationOnFail: scenario.RestartIterationOnFail,
			MaxFailCount:           scenario.MaxFailCount,
			WarmUpDurationSeconds:  scenario.WarmUpDurationSeconds,
			WarmUpDisabled:         scenario.WarmUpDisabled,
			LoadSimulations:        append([]LoadSimulation(nil), scenario.LoadSimulations...),
			Thresholds:             append([]ThresholdSpec(nil), scenario.Thresholds...),
			StepCallbackURL:        httpHost.scenarioCallbackURL(scenarioID, "step"),
			MetricsCallbackURL:     httpHost.scenarioCallbackURL(scenarioID, "metrics"),
		}
		if scenario.Init != nil {
			item.InitCallbackURL = httpHost.scenarioCallbackURL(scenarioID, "init")
		}
		if scenario.Clean != nil {
			item.CleanCallbackURL = httpHost.scenarioCallbackURL(scenarioID, "clean")
		}
		if scenario.Tracking != nil {
			item.Tracking = cloneTrackingConfigurationWithCallbackURLs(scenario.Tracking, registry, httpHost)
		}
		plan.Scenarios = append(plan.Scenarios, item)
	}

	return plan, nil
}

type runtimeContextExtensionPlan struct {
	ReportingSinks  []runtimeReportingSinkPlan
	WorkerPlugins   []runtimeWorkerPluginPlan
	RuntimePolicies []runtimePolicyPlan
}

func buildRuntimeContextExtensions(
	context contextState,
	registry *runtimeCallbackRegistry,
	httpHost *runtimeHTTPHostHandle,
) (runtimeContextExtensionPlan, error) {
	result := runtimeContextExtensionPlan{}

	for _, plugin := range context.WorkerPlugins {
		if plugin == nil {
			continue
		}
		id := registry.registerWorkerPlugin(plugin)
		result.WorkerPlugins = append(result.WorkerPlugins, runtimeWorkerPluginPlan{
			Name:        firstNonBlank(plugin.PluginName(), "worker-plugin"),
			CallbackURL: httpHost.pluginCallbackURL(id),
		})
	}

	for _, policy := range context.RuntimePolicies {
		if policy == nil {
			continue
		}
		id := registry.registerRuntimePolicy(policy)
		result.RuntimePolicies = append(result.RuntimePolicies, runtimePolicyPlan{
			Name:        runtimePolicyName(policy),
			CallbackURL: httpHost.policyCallbackURL(id),
		})
	}

	for _, sink := range context.ReportingSinks {
		if sink == nil {
			continue
		}
		plan, err := buildRuntimeReportingSinkPlan(sink, registry, httpHost)
		if err != nil {
			return runtimeContextExtensionPlan{}, err
		}
		result.ReportingSinks = append(result.ReportingSinks, plan)
	}

	return result, nil
}

func buildRuntimeReportingSinkPlan(
	sink LoadStrikeReportingSink,
	registry *runtimeCallbackRegistry,
	httpHost *runtimeHTTPHostHandle,
) (runtimeReportingSinkPlan, error) {
	switch typed := sink.(type) {
	case InfluxDbReportingSink:
		options := typed.Options
		return runtimeReportingSinkPlan{Kind: "influxdb", Name: typed.SinkName(), InfluxDB: &options}, nil
	case *InfluxDbReportingSink:
		if typed != nil {
			options := typed.Options
			return runtimeReportingSinkPlan{Kind: "influxdb", Name: typed.SinkName(), InfluxDB: &options}, nil
		}
	case TimescaleDbReportingSink:
		options := typed.Options
		return runtimeReportingSinkPlan{Kind: "timescaledb", Name: typed.SinkName(), TimescaleDB: &options}, nil
	case *TimescaleDbReportingSink:
		if typed != nil {
			options := typed.Options
			return runtimeReportingSinkPlan{Kind: "timescaledb", Name: typed.SinkName(), TimescaleDB: &options}, nil
		}
	case GrafanaLokiReportingSink:
		options := typed.Options
		return runtimeReportingSinkPlan{Kind: "grafanaloki", Name: typed.SinkName(), GrafanaLoki: &options}, nil
	case *GrafanaLokiReportingSink:
		if typed != nil {
			options := typed.Options
			return runtimeReportingSinkPlan{Kind: "grafanaloki", Name: typed.SinkName(), GrafanaLoki: &options}, nil
		}
	case DatadogReportingSink:
		options := typed.Options
		return runtimeReportingSinkPlan{Kind: "datadog", Name: typed.SinkName(), Datadog: &options}, nil
	case *DatadogReportingSink:
		if typed != nil {
			options := typed.Options
			return runtimeReportingSinkPlan{Kind: "datadog", Name: typed.SinkName(), Datadog: &options}, nil
		}
	case SplunkReportingSink:
		options := typed.Options
		return runtimeReportingSinkPlan{Kind: "splunk", Name: typed.SinkName(), Splunk: &options}, nil
	case *SplunkReportingSink:
		if typed != nil {
			options := typed.Options
			return runtimeReportingSinkPlan{Kind: "splunk", Name: typed.SinkName(), Splunk: &options}, nil
		}
	case OtelCollectorReportingSink:
		options := typed.Options
		return runtimeReportingSinkPlan{Kind: "otelcollector", Name: typed.SinkName(), OTELCollector: &options}, nil
	case *OtelCollectorReportingSink:
		if typed != nil {
			options := typed.Options
			return runtimeReportingSinkPlan{Kind: "otelcollector", Name: typed.SinkName(), OTELCollector: &options}, nil
		}
	}

	id := registry.registerReportingSink(sink)
	return runtimeReportingSinkPlan{
		Kind:        "callback",
		Name:        firstNonBlank(sink.SinkName(), "reporting-sink"),
		CallbackURL: httpHost.sinkCallbackURL(id),
	}, nil
}

func cloneTrackingConfigurationWithCallbackURLs(
	source *TrackingConfigurationSpec,
	registry *runtimeCallbackRegistry,
	httpHost *runtimeHTTPHostHandle,
) *TrackingConfigurationSpec {
	if source == nil {
		return nil
	}

	encoded, err := json.Marshal(source)
	if err != nil {
		return source
	}

	var cloned TrackingConfigurationSpec
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return source
	}

	cloned.Source = cloneEndpointSpecWithCallbackURLs(source.Source, cloned.Source, registry, httpHost)
	cloned.Destination = cloneEndpointSpecWithCallbackURLs(source.Destination, cloned.Destination, registry, httpHost)
	return &cloned
}

func cloneEndpointSpecWithCallbackURLs(
	source *EndpointSpec,
	cloned *EndpointSpec,
	registry *runtimeCallbackRegistry,
	httpHost *runtimeHTTPHostHandle,
) *EndpointSpec {
	if source == nil || cloned == nil {
		return cloned
	}

	if source.DelegateStream != nil && cloned.DelegateStream != nil {
		if source.DelegateStream.Produce != nil {
			id := registry.registerTrackingProduce(source.DelegateStream.Produce)
			cloned.DelegateStream.ProduceCallbackURL = httpHost.trackingProduceCallbackURL(id)
		}
		if source.DelegateStream.Consume != nil {
			id := registry.registerTrackingConsume(source.DelegateStream.Consume)
			cloned.DelegateStream.ConsumeCallbackURL = httpHost.trackingConsumeCallbackURL(id)
		}
	}

	if source.PushDiffusion != nil && cloned.PushDiffusion != nil {
		if source.PushDiffusion.Publish != nil {
			id := registry.registerTrackingProduce(source.PushDiffusion.Publish)
			cloned.PushDiffusion.PublishCallbackURL = httpHost.trackingProduceCallbackURL(id)
		}
		if source.PushDiffusion.Subscribe != nil {
			id := registry.registerTrackingConsume(source.PushDiffusion.Subscribe)
			cloned.PushDiffusion.SubscribeCallbackURL = httpHost.trackingConsumeCallbackURL(id)
		}
	}

	return cloned
}

func runtimePolicyName(policy any) string {
	if policy == nil {
		return ""
	}
	switch typed := policy.(type) {
	case interface{ GetTypeName() string }:
		return strings.TrimSpace(typed.GetTypeName())
	default:
		policyType := reflect.TypeOf(policy)
		for policyType != nil && policyType.Kind() == reflect.Pointer {
			policyType = policyType.Elem()
		}
		if policyType == nil {
			return "runtime-policy"
		}
		if name := strings.TrimSpace(policyType.Name()); name != "" {
			return name
		}
		return "runtime-policy"
	}
}
