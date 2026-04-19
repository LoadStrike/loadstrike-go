package loadstrike

import (
	"os"
	"testing"
)

func TestRunLaunchesExactVersionRuntimeForSingleNodeExecution(t *testing.T) {
	t.Setenv(envAllowLocalStub, "1")

	result := NewRunner().
		AddScenario(CreateScenario("orders", func(LoadStrikeScenarioContext) LoadStrikeReply {
			return LoadStrikeResponse.Ok("200")
		})).
		WithRunnerKey("runner").
		WithoutReports().
		Run()

	if result.AllRequestCount() != 1 {
		t.Fatalf("AllRequestCount() = %d, want 1", result.AllRequestCount())
	}
	if result.AllOKCount() != 1 {
		t.Fatalf("AllOKCount() = %d, want 1", result.AllOKCount())
	}
}

func TestRunUsesExternalRuntimeBinaryWhenConfigured(t *testing.T) {
	if os.Getenv(envRuntimePath) == "" {
		t.Skip("set LOADSTRIKE_RUNTIME_PATH to run the external runtime integration test")
	}

	result := NewRunner().
		AddScenario(CreateScenario("orders", func(LoadStrikeScenarioContext) LoadStrikeReply {
			return LoadStrikeResponse.Ok("200")
		})).
		WithRunnerKey("runner").
		Run()

	if result.AllRequestCount() != 1 {
		t.Fatalf("AllRequestCount() = %d, want 1", result.AllRequestCount())
	}
	if result.AllOKCount() != 1 {
		t.Fatalf("AllOKCount() = %d, want 1", result.AllOKCount())
	}
}
