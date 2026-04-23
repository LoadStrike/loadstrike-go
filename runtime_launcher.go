package loadstrike

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"

	"loadstrike.com/sdk/go/runtimeproto"
)

var (
	errMissingRuntimeOut = errors.New("runtime did not produce a run result")
)

func runViaPrivateRuntime(contextState *contextState, registry *runtimeCallbackRegistry) (runResult, error) {
	host, err := startRuntimeHostServer(registry)
	if err != nil {
		return runResult{}, err
	}
	defer host.Close()

	httpHost, err := startRuntimeHTTPHostServer(registry)
	if err != nil {
		return runResult{}, err
	}
	defer httpHost.Close()

	plan, err := buildRuntimePlan(contextState, registry, httpHost)
	if err != nil {
		return runResult{}, err
	}

	runtimePath, err := newRuntimeArtifactResolver(runtimeResolverConfig{
		Version: RuntimeArtifactVersion(),
		GOOS:    runtimeGOOS(),
		GOARCH:  runtimeGOARCH(),
	}).resolveRuntimePath(contextState.RunnerKey)
	if err != nil {
		return runResult{}, err
	}

	tempDir, err := os.MkdirTemp("", "loadstrike-runtime-*")
	if err != nil {
		return runResult{}, fmt.Errorf("create runtime temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	planPath := filepath.Join(tempDir, "plan.json")
	resultPath := filepath.Join(tempDir, "result.json")
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return runResult{}, fmt.Errorf("marshal runtime plan: %w", err)
	}
	if err := os.WriteFile(planPath, planBytes, 0o600); err != nil {
		return runResult{}, fmt.Errorf("write runtime plan: %w", err)
	}

	cmd := exec.Command(
		runtimePath,
		"--host", host.address,
		"--plan", planPath,
		"--output", resultPath,
		"--sdk-version", RuntimeArtifactVersion(),
		"--protocol", strconv.Itoa(RuntimeProtocolVersion()),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return runResult{}, fmt.Errorf("loadstrike runtime failed: %w", err)
		}
		return runResult{}, fmt.Errorf("loadstrike runtime failed: %w: %s", err, message)
	}

	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runResult{}, errMissingRuntimeOut
		}
		return runResult{}, fmt.Errorf("read runtime result: %w", err)
	}

	var result LoadStrikeRunResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return runResult{}, fmt.Errorf("decode runtime result: %w", err)
	}

	return result.toNative(), nil
}

type runtimeHostHandle struct {
	address  string
	listener net.Listener
	server   *grpc.Server
}

func startRuntimeHostServer(registry *runtimeCallbackRegistry) (*runtimeHostHandle, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for runtime host server: %w", err)
	}

	server := grpc.NewServer()
	runtimeproto.RegisterHostRuntimeServer(server, newRuntimeHostServer(registry))

	go func() {
		_ = server.Serve(listener)
	}()

	return &runtimeHostHandle{
		address:  listener.Addr().String(),
		listener: listener,
		server:   server,
	}, nil
}

// Close releases owned resources. Use this when the current SDK object is no longer needed.
func (h *runtimeHostHandle) Close() {
	if h == nil {
		return
	}

	stopped := make(chan struct{})
	go func() {
		h.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		h.server.Stop()
	}

	_ = h.listener.Close()
}

func runtimeDialContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
