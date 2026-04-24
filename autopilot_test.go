package loadstrike

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func ensureAutopilotRuntimeArtifact(t *testing.T) {
	t.Helper()

	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("LOCALAPPDATA", cacheRoot)

	runtimePath := newRuntimeArtifactResolver(runtimeResolverConfig{
		Version: RuntimeArtifactVersion(),
		GOOS:    runtimeGOOS(),
		GOARCH:  runtimeGOARCH(),
	}).expectedRuntimePath()
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0o755); err != nil {
		t.Fatalf("create runtime cache dir: %v", err)
	}

	sourcePath := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(sourcePath, []byte(fakeAutopilotRuntimeSource), 0o600); err != nil {
		t.Fatalf("write fake runtime source: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", runtimePath, sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build fake runtime: %v\n%s", err, string(output))
	}
}

const fakeAutopilotRuntimeSource = `package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var inputPath string
	var outputPath string
	for index := 1; index < len(os.Args); index++ {
		switch os.Args[index] {
		case "--autopilot-input":
			index++
			if index < len(os.Args) {
				inputPath = os.Args[index]
			}
		case "--autopilot-output":
			index++
			if index < len(os.Args) {
				outputPath = os.Args[index]
			}
		}
	}
	if outputPath == "" {
		_, _ = fmt.Fprintln(os.Stderr, "missing autopilot output")
		os.Exit(2)
	}

	requestBytes, err := os.ReadFile(inputPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var request struct {
		Kind string
	}
	_ = json.Unmarshal(requestBytes, &request)

	result := map[string]any{
		"Readiness": "Ready",
		"Plan": map[string]any{
			"Version": "1",
			"Scenario": map[string]any{
				"Name":          "orders-har",
				"WithoutWarmUp": true,
			},
			"ThresholdSuggestions": []map[string]any{
				{"Kind": "AllFailCount", "Maximum": 0},
			},
		},
		"Endpoints": []map[string]any{
			{"Kind": "Http", "Name": "source-http", "Method": "GET", "Url": "http://127.0.0.1:1/orders"},
		},
		"PreviewReport": map[string]any{
			"ScenarioName": "orders-har",
			"Readiness":    "Ready",
		},
		"httpReplay": map[string]any{
			"Method": "GET",
			"Url":    "http://127.0.0.1:1/orders",
		},
	}
	if request.Kind == "MessagePair" {
		result = map[string]any{
			"Readiness": "RequiresEndpointBinding",
			"Plan": map[string]any{
				"Version": "1",
				"Scenario": map[string]any{
					"Name":          "autopilot-starter",
					"WithoutWarmUp": true,
				},
			},
			"Endpoints": []map[string]any{
				{"Kind": "DelegateStream", "Name": "source"},
				{"Kind": "DelegateStream", "Name": "destination"},
			},
			"Warnings": []string{"Endpoint bindings are required."},
		}
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.WriteFile(outputPath, resultBytes, 0o600); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
`

func TestGoAutopilotPublicWrapperGeneratesReadyHAR(t *testing.T) {
	ensureAutopilotRuntimeArtifact(t)

	result := LoadStrikeAutopilot.Generate(LoadStrikeAutopilotRequest{
		Kind:    "Har",
		Content: `{"log":{"entries":[]}}`,
		Options: LoadStrikeAutopilotOptions{
			ScenarioName:         "orders-har",
			IncludePreviewReport: true,
			AllowedReplayHosts:   []string{"api.loadstrike.test"},
		},
	})

	if result.Readiness != LoadStrikeAutopilotReady {
		t.Fatalf("Readiness = %q", result.Readiness)
	}
	if result.HTTPReplay == nil {
		t.Fatal("expected private runtime replay plan")
	}
	if result.PreviewReport == nil {
		t.Fatal("expected preview report")
	}
	if scenario := result.BuildScenario(); !scenario.nativeValue().WarmUpDisabled {
		t.Fatal("expected generated scenario to disable warmup")
	}
}

func TestGoAutopilotPublicWrapperMessagePairRequiresEndpointBinding(t *testing.T) {
	ensureAutopilotRuntimeArtifact(t)

	source := LoadStrikeAutopilotMessageSample{
		Name:    "orders-source",
		Headers: map[string]string{"x-correlation-id": "ord-1004"},
		Payload: map[string]any{"orderId": "ord-1004"},
	}
	destination := LoadStrikeAutopilotMessageSample{
		Name:    "orders-destination",
		Headers: map[string]string{"x-correlation-id": "ord-1004"},
		Payload: map[string]any{"orderId": "ord-1004", "status": "processed"},
	}

	result := LoadStrikeAutopilot.Generate(LoadStrikeAutopilotRequest{
		Kind:               "MessagePair",
		SourceMessage:      &source,
		DestinationMessage: &destination,
	})

	if result.Readiness != LoadStrikeAutopilotRequiresEndpointBinding {
		t.Fatalf("Readiness = %q", result.Readiness)
	}
	if len(result.Endpoints) != 2 || result.Endpoints[0].Kind != "DelegateStream" {
		t.Fatalf("endpoints = %#v", result.Endpoints)
	}
}
