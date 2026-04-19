package loadstrike

import (
	"context"
	"testing"

	"loadstrike.com/sdk/go/runtimeproto"
)

func TestHostServerExecutesRegisteredCallback(t *testing.T) {
	registry := newRuntimeCallbackRegistry()
	callbackID := registry.registerScenarioStep("orders", "orders", func(*stepRuntimeContext) replyResult {
		return LoadStrikeResponse.Ok("200").toNative()
	})

	server := newRuntimeHostServer(registry)
	reply, err := server.InvokeStep(context.Background(), &runtimeproto.InvokeStepRequest{
		CallbackId:   callbackID,
		ScenarioName: "orders",
		StepName:     "orders",
	})
	if err != nil {
		t.Fatalf("InvokeStep() error = %v", err)
	}
	if reply.StatusCode != "200" {
		t.Fatalf("reply.StatusCode = %q, want 200", reply.StatusCode)
	}
}
