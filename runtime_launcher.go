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
	errNoScenarios       = errors.New("at least one scenario is required before Run()")
	errMissingRunnerKey  = errors.New("runner key is required. call WithRunnerKey(...) before Run()")
	errMissingRuntimeOut = errors.New("runtime did not produce a run result")
)

func (c *contextState) Run(args ...string) (runSummary, error) {
	if c == nil {
		return runSummary{}, errors.New("context must be provided")
	}
	if len(c.scenarios) == 0 {
		return runSummary{}, errNoScenarios
	}
	if strings.TrimSpace(c.RunnerKey) == "" {
		return runSummary{}, errMissingRunnerKey
	}
	plan, registry, err := buildRuntimePlan(c)
	if err != nil {
		return runSummary{}, err
	}

	return runViaPrivateRuntime(c, plan, registry)
}

func runViaPrivateRuntime(contextState *contextState, plan runtimePlan, registry *runtimeCallbackRegistry) (runSummary, error) {
	host, err := startRuntimeHostServer(registry)
	if err != nil {
		return runSummary{}, err
	}
	defer host.Close()

	runtimePath, err := newRuntimeArtifactResolver(runtimeResolverConfig{
		Version: RuntimeArtifactVersion(),
		GOOS:    runtimeGOOS(),
		GOARCH:  runtimeGOARCH(),
	}).resolveRuntimePath(contextState.RunnerKey)
	if err != nil {
		return runSummary{}, err
	}

	tempDir, err := os.MkdirTemp("", "loadstrike-runtime-*")
	if err != nil {
		return runSummary{}, fmt.Errorf("create runtime temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	planPath := filepath.Join(tempDir, "plan.json")
	resultPath := filepath.Join(tempDir, "result.json")
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return runSummary{}, fmt.Errorf("marshal runtime plan: %w", err)
	}
	if err := os.WriteFile(planPath, planBytes, 0o600); err != nil {
		return runSummary{}, fmt.Errorf("write runtime plan: %w", err)
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
			return runSummary{}, fmt.Errorf("loadstrike runtime failed: %w", err)
		}
		return runSummary{}, fmt.Errorf("loadstrike runtime failed: %w: %s", err, message)
	}

	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runSummary{}, errMissingRuntimeOut
		}
		return runSummary{}, fmt.Errorf("read runtime result: %w", err)
	}

	var result runSummary
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return runSummary{}, fmt.Errorf("decode runtime result: %w", err)
	}

	return result, nil
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
