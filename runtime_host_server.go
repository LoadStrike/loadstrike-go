package loadstrike

import (
	"context"
	"fmt"

	"loadstrike.com/sdk/go/runtimeproto"
)

type runtimeHostServer struct {
	runtimeproto.UnimplementedHostRuntimeServer
	registry *runtimeCallbackRegistry
}

func newRuntimeHostServer(registry *runtimeCallbackRegistry) *runtimeHostServer {
	return &runtimeHostServer{registry: registry}
}

// Handshake exposes the handshake operation. Use this when interacting with the SDK through this surface.
func (s *runtimeHostServer) Handshake(_ context.Context, request *runtimeproto.HandshakeRequest) (*runtimeproto.HandshakeResponse, error) {
	if request.GetSdkVersion() != RuntimeArtifactVersion() {
		return nil, RuntimeMismatchError{
			ExpectedVersion: RuntimeArtifactVersion(),
			ActualVersion:   request.GetSdkVersion(),
		}
	}
	if int(request.GetProtocolVersion()) != RuntimeProtocolVersion() {
		return nil, fmt.Errorf("loadstrike runtime protocol mismatch: expected %d, got %d", RuntimeProtocolVersion(), request.GetProtocolVersion())
	}
	return &runtimeproto.HandshakeResponse{
		SdkVersion:      RuntimeArtifactVersion(),
		ProtocolVersion: int32(RuntimeProtocolVersion()),
	}, nil
}

// InvokeStep exposes the invoke step operation. Use this when interacting with the SDK through this surface.
func (s *runtimeHostServer) InvokeStep(_ context.Context, request *runtimeproto.InvokeStepRequest) (*runtimeproto.InvokeStepResponse, error) {
	callback, ok := s.registry.lookup(request.GetCallbackId())
	if !ok {
		return nil, fmt.Errorf("loadstrike callback %q was not registered", request.GetCallbackId())
	}

	reply := callback.Run(&stepRuntimeContext{
		ScenarioName: request.GetScenarioName(),
		StepName:     request.GetStepName(),
	})

	return &runtimeproto.InvokeStepResponse{
		IsSuccess:  reply.IsSuccess,
		StatusCode: reply.StatusCode,
		Message:    reply.Message,
	}, nil
}
