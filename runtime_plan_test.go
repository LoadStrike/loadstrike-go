package loadstrike

import "testing"

func TestBuildRuntimePlanCapturesScenarioMetadataWithoutClosures(t *testing.T) {
	context := Create().
		AddScenario(
			CreateScenario("orders", func(LoadStrikeScenarioContext) LoadStrikeReply {
				return LoadStrikeResponse.Ok("200")
			}).WithLoadSimulations(
				LoadStrikeSimulation.IterationsForConstant(1, 10),
			),
		).
		WithRunnerKey("runner").
		WithoutReports().
		BuildContext()

	plan, registry, err := buildRuntimePlan(requireNativeContext(context.nativeValue()))
	if err != nil {
		t.Fatalf("buildRuntimePlan() error = %v", err)
	}
	if len(plan.Scenarios) != 1 {
		t.Fatalf("len(plan.Scenarios) = %d, want 1", len(plan.Scenarios))
	}
	if plan.Context.RunnerKey != "runner" {
		t.Fatalf("plan.Context.RunnerKey = %q, want runner", plan.Context.RunnerKey)
	}
	if plan.Scenarios[0].PrimaryStepCallbackID == "" {
		t.Fatal("PrimaryStepCallbackID was empty")
	}
	if _, ok := registry.lookup(plan.Scenarios[0].PrimaryStepCallbackID); !ok {
		t.Fatal("callback id was not registered")
	}
	if len(plan.Scenarios[0].LoadSimulations) != 1 {
		t.Fatalf("len(plan.Scenarios[0].LoadSimulations) = %d, want 1", len(plan.Scenarios[0].LoadSimulations))
	}
}
