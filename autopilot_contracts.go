package loadstrike

import (
	"fmt"
	"strings"
)

const (
	LoadStrikeAutopilotReady                    = "Ready"
	LoadStrikeAutopilotRequiresSecrets          = "RequiresSecrets"
	LoadStrikeAutopilotRequiresEndpointBinding  = "RequiresEndpointBinding"
	LoadStrikeAutopilotRequiresTargetBinding    = "RequiresTargetBinding"
	LoadStrikeAutopilotRequiresTrackingSelector = "RequiresTrackingSelector"
	LoadStrikeAutopilotRequiresReview           = "RequiresReview"
)

type LoadStrikeAutopilotRequest struct {
	Kind               string                            `json:"Kind"`
	FilePath           string                            `json:"FilePath,omitempty"`
	Content            string                            `json:"Content,omitempty"`
	SourceMessage      *LoadStrikeAutopilotMessageSample `json:"SourceMessage,omitempty"`
	DestinationMessage *LoadStrikeAutopilotMessageSample `json:"DestinationMessage,omitempty"`
	Options            LoadStrikeAutopilotOptions        `json:"Options,omitempty"`
}

type LoadStrikeAutopilotOptions struct {
	ScenarioName         string                               `json:"ScenarioName,omitempty"`
	IncludePreviewReport bool                                 `json:"IncludePreviewReport,omitempty"`
	AllowedReplayHosts   []string                             `json:"AllowedReplayHosts,omitempty"`
	BaseURLRewrite       string                               `json:"BaseUrlRewrite,omitempty"`
	TrackingSelector     string                               `json:"TrackingSelector,omitempty"`
	SecretBindings       []LoadStrikeAutopilotSecretBinding   `json:"SecretBindings,omitempty"`
	EndpointBindings     []LoadStrikeAutopilotEndpointBinding `json:"EndpointBindings,omitempty"`
	RunnerKey            string                               `json:"-"`
}

type LoadStrikeAutopilotSecretBinding struct {
	Location     string `json:"Location"`
	ValueFromEnv string `json:"ValueFromEnv"`
}

type LoadStrikeAutopilotEndpointBinding struct {
	Name   string `json:"Name"`
	Kind   string `json:"Kind,omitempty"`
	Method string `json:"Method,omitempty"`
	URL    string `json:"Url,omitempty"`
}

type LoadStrikeAutopilotMessageSample struct {
	Name              string            `json:"Name,omitempty"`
	ContentType       string            `json:"ContentType,omitempty"`
	Headers           map[string]string `json:"Headers,omitempty"`
	Payload           any               `json:"Payload,omitempty"`
	TransportMetadata map[string]string `json:"TransportMetadata,omitempty"`
}

type LoadStrikeAutopilotPlan struct {
	Version                     string                                          `json:"Version"`
	Scenario                    LoadStrikeAutopilotScenarioPlan                 `json:"Scenario"`
	Endpoints                   []LoadStrikeAutopilotEndpoint                   `json:"Endpoints,omitempty"`
	TrackingSelectorSuggestions []LoadStrikeAutopilotTrackingSelectorSuggestion `json:"TrackingSelectorSuggestions,omitempty"`
	ThresholdSuggestions        []LoadStrikeAutopilotThresholdSuggestion        `json:"ThresholdSuggestions,omitempty"`
	Redactions                  []LoadStrikeAutopilotRedaction                  `json:"Redactions,omitempty"`
	Warnings                    []string                                        `json:"Warnings,omitempty"`
	Confidence                  float64                                         `json:"Confidence,omitempty"`
}

type LoadStrikeAutopilotScenarioPlan struct {
	Name                      string                                        `json:"Name"`
	WithoutWarmUp             bool                                          `json:"WithoutWarmUp"`
	LoadSimulationSuggestions []LoadStrikeAutopilotLoadSimulationSuggestion `json:"LoadSimulationSuggestions,omitempty"`
}

type LoadStrikeAutopilotLoadSimulationSuggestion struct {
	Kind       string `json:"Kind"`
	Copies     int    `json:"Copies"`
	Iterations int    `json:"Iterations"`
}

type LoadStrikeAutopilotThresholdSuggestion struct {
	Kind    string  `json:"Kind"`
	Maximum float64 `json:"Maximum"`
}

type LoadStrikeAutopilotTrackingSelectorSuggestion struct {
	Expression string  `json:"Expression"`
	Confidence float64 `json:"Confidence"`
}

type LoadStrikeAutopilotEndpoint struct {
	Kind   string `json:"Kind"`
	Name   string `json:"Name"`
	Method string `json:"Method,omitempty"`
	URL    string `json:"Url,omitempty"`
}

type LoadStrikeAutopilotRedaction struct {
	Location    string `json:"Location"`
	Reason      string `json:"Reason"`
	Replacement string `json:"Replacement"`
}

type LoadStrikeAutopilotReadinessFailure struct {
	Readiness     string   `json:"Readiness"`
	Reason        string   `json:"Reason"`
	RequiredInput string   `json:"RequiredInput"`
	Locations     []string `json:"Locations,omitempty"`
}

type LoadStrikeAutopilotPreviewReport struct {
	ScenarioName      string                                `json:"ScenarioName"`
	Readiness         string                                `json:"Readiness"`
	Signals           []string                              `json:"Signals,omitempty"`
	Warnings          []string                              `json:"Warnings,omitempty"`
	ReadinessFailures []LoadStrikeAutopilotReadinessFailure `json:"ReadinessFailures,omitempty"`
	Redactions        []LoadStrikeAutopilotRedaction        `json:"Redactions,omitempty"`
}

type LoadStrikeAutopilotHTTPReplay struct {
	Method      string            `json:"Method"`
	URL         string            `json:"Url"`
	Headers     map[string]string `json:"Headers,omitempty"`
	Body        string            `json:"Body,omitempty"`
	ContentType string            `json:"ContentType,omitempty"`
}

type LoadStrikeAutopilotResult struct {
	Readiness         string                                `json:"Readiness"`
	Plan              LoadStrikeAutopilotPlan               `json:"Plan"`
	Endpoints         []LoadStrikeAutopilotEndpoint         `json:"Endpoints,omitempty"`
	Redactions        []LoadStrikeAutopilotRedaction        `json:"Redactions,omitempty"`
	Warnings          []string                              `json:"Warnings,omitempty"`
	ReadinessFailures []LoadStrikeAutopilotReadinessFailure `json:"ReadinessFailures,omitempty"`
	PreviewReport     *LoadStrikeAutopilotPreviewReport     `json:"PreviewReport,omitempty"`
	ObservedLatencyMs *float64                              `json:"ObservedLatencyMs,omitempty"`
	HTTPReplay        *LoadStrikeAutopilotHTTPReplay        `json:"httpReplay,omitempty"`
}

func (r LoadStrikeAutopilotResult) BuildScenario() LoadStrikeScenario {
	if r.Readiness != LoadStrikeAutopilotReady {
		panic(fmt.Sprintf("Autopilot result is not ready to build a scenario. Current readiness: %s.", r.Readiness))
	}
	if r.HTTPReplay == nil {
		panic("Autopilot result does not contain a runnable HTTP request.")
	}

	scenarioName := strings.TrimSpace(r.Plan.Scenario.Name)
	if scenarioName == "" {
		scenarioName = "autopilot-starter"
	}
	thresholds := make([]ThresholdSpec, 0, len(r.Plan.ThresholdSuggestions))
	for _, suggestion := range r.Plan.ThresholdSuggestions {
		thresholds = append(thresholds, ScenarioThreshold(suggestion.Kind, "<=", suggestion.Maximum))
	}
	native := scenarioDefinition{
		Name:            scenarioName,
		WarmUpDisabled:  true,
		LoadSimulations: []LoadSimulation{IterationsForConstant(1, 1)},
		Thresholds:      thresholds,
		AutopilotHTTP:   cloneAutopilotHTTPReplay(r.HTTPReplay),
	}
	return newLoadStrikeScenario(native)
}

func (r LoadStrikeAutopilotResult) buildScenario() LoadStrikeScenario {
	return r.BuildScenario()
}

func cloneAutopilotHTTPReplay(source *LoadStrikeAutopilotHTTPReplay) *LoadStrikeAutopilotHTTPReplay {
	if source == nil {
		return nil
	}
	headers := make(map[string]string, len(source.Headers))
	for key, value := range source.Headers {
		headers[key] = value
	}
	return &LoadStrikeAutopilotHTTPReplay{
		Method:      source.Method,
		URL:         source.URL,
		Headers:     headers,
		Body:        source.Body,
		ContentType: source.ContentType,
	}
}
