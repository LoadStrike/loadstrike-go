package loadstrike

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const (
	runtimeCommandDiagnosticBytes = 32 * 1024
	runtimeCommandCopyBufferBytes = 32 * 1024
	runtimeCommandRedaction       = "[REDACTED]"
)

type runtimeCommandOutputOptions struct {
	spoolDirectory    string
	loggerTarget      string
	runnerKey         string
	stdoutDestination io.Writer
	stderrDestination io.Writer
}

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
	if err := plan.validateForLaunch(); err != nil {
		return runResult{}, err
	}

	runtimeExecution, err := newRuntimeArtifactResolver(runtimeResolverConfig{
		Version: RuntimeArtifactVersion(),
		GOOS:    runtimeGOOS(),
		GOARCH:  runtimeGOARCH(),
	}).resolveRuntimeExecution(contextState.RunnerKey, contextState.NodeType == NodeTypeAgent && strings.TrimSpace(contextState.NatsServerURL) != "")
	if err != nil {
		return runResult{}, err
	}
	defer runtimeExecution.Close()
	runtimePath := runtimeExecution.Path

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

	// runtimePath is returned only after authenticated-manifest and SHA-256 verification;
	// exec.Command receives it and the remaining values as argv without invoking a shell.
	cmd := exec.Command(
		runtimePath,
		"--host", host.address,
		"--plan", planPath,
		"--output", resultPath,
		"--sdk-version", RuntimeArtifactVersion(),
		"--protocol", strconv.Itoa(RuntimeProtocolVersion()),
	)
	output, err := executeRuntimeCommand(cmd, runtimeCommandOutputOptions{
		spoolDirectory:    tempDir,
		loggerTarget:      runtimeLoggerTarget(plan.Context.LoggerConfig),
		runnerKey:         contextState.RunnerKey,
		stdoutDestination: os.Stdout,
		stderrDestination: os.Stderr,
	})
	if err != nil {
		return runResult{}, newRuntimeFailure(err, output, contextState.RunnerKey)
	}
	clearAutopilotRunnerKeyBytes(output)

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

func executeRuntimeCommand(cmd *exec.Cmd, options runtimeCommandOutputOptions) ([]byte, error) {
	stdoutSpool, err := os.CreateTemp(options.spoolDirectory, "runtime-stdout-*.log")
	if err != nil {
		return nil, fmt.Errorf("create runtime stdout spool: %w", err)
	}
	stdoutPath := stdoutSpool.Name()
	defer os.Remove(stdoutPath)
	defer stdoutSpool.Close()

	stderrSpool, err := os.CreateTemp(options.spoolDirectory, "runtime-stderr-*.log")
	if err != nil {
		return nil, fmt.Errorf("create runtime stderr spool: %w", err)
	}
	stderrPath := stderrSpool.Name()
	defer os.Remove(stderrPath)
	defer stderrSpool.Close()

	cmd.Stdout = stdoutSpool
	cmd.Stderr = stderrSpool
	executionErr := cmd.Run()
	stdoutCloseErr := stdoutSpool.Close()
	stderrCloseErr := stderrSpool.Close()

	if executionErr != nil {
		if stdoutCloseErr != nil || stderrCloseErr != nil {
			return nil, executionErr
		}
		diagnostic := readRuntimeCommandDiagnostic(options.loggerTarget, stdoutPath, stderrPath, options.runnerKey)
		if forwardErr := forwardRuntimeCommandLoggerOutput(options, stdoutPath, stderrPath); forwardErr != nil {
			executionErr = errors.Join(executionErr, forwardErr)
		}
		return diagnostic, executionErr
	}
	if stdoutCloseErr != nil {
		return nil, fmt.Errorf("close runtime stdout spool: %w", stdoutCloseErr)
	}
	if stderrCloseErr != nil {
		return nil, fmt.Errorf("close runtime stderr spool: %w", stderrCloseErr)
	}

	if err := forwardRuntimeCommandLoggerOutput(options, stdoutPath, stderrPath); err != nil {
		return nil, err
	}
	return nil, nil
}

func forwardRuntimeCommandLoggerOutput(
	options runtimeCommandOutputOptions,
	stdoutPath string,
	stderrPath string,
) error {
	switch strings.ToLower(strings.TrimSpace(options.loggerTarget)) {
	case "stdout":
		if options.stdoutDestination == nil {
			return errors.New("runtime stdout destination must be provided")
		}
		if err := copyRuntimeCommandSpool(stdoutPath, options.stdoutDestination, options.runnerKey); err != nil {
			return fmt.Errorf("forward runtime stdout: %w", err)
		}
	case "stderr":
		if options.stderrDestination == nil {
			return errors.New("runtime stderr destination must be provided")
		}
		if err := copyRuntimeCommandSpool(stderrPath, options.stderrDestination, options.runnerKey); err != nil {
			return fmt.Errorf("forward runtime stderr: %w", err)
		}
	}
	return nil
}

func runtimeLoggerTarget(config map[string]any) string {
	target, _ := config["target"].(string)
	return strings.ToLower(strings.TrimSpace(target))
}

func copyRuntimeCommandSpool(path string, destination io.Writer, runnerKey string) error {
	spool, err := os.Open(path)
	if err != nil {
		return err
	}
	defer spool.Close()
	return copyRuntimeCommandOutput(spool, destination, runnerKey)
}

func copyRuntimeCommandOutput(source io.Reader, destination io.Writer, runnerKey string) error {
	if runnerKey == "" {
		buffer := make([]byte, runtimeCommandCopyBufferBytes)
		_, err := io.CopyBuffer(destination, source, buffer)
		return err
	}

	secret := []byte(runnerKey)
	replacement := []byte(runtimeCommandRedaction)
	buffer := make([]byte, runtimeCommandCopyBufferBytes)
	pending := make([]byte, 0, runtimeCommandCopyBufferBytes+len(secret))
	defer func() {
		clearAutopilotRunnerKeyBytes(secret)
		clearAutopilotRunnerKeyBytes(buffer)
		if cap(pending) > 0 {
			clearAutopilotRunnerKeyBytes(pending[:cap(pending)])
		}
	}()

	for {
		count, readErr := source.Read(buffer)
		if count > 0 {
			pending = append(pending, buffer[:count]...)
			flushBefore := len(pending) - (len(secret) - 1)
			if flushBefore > 0 {
				var writeErr error
				pending, writeErr = writeRuntimeCommandRedactedPrefix(
					destination,
					pending,
					secret,
					replacement,
					flushBefore,
				)
				if writeErr != nil {
					return writeErr
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	_, err := writeRuntimeCommandRedactedPrefix(
		destination,
		pending,
		secret,
		replacement,
		len(pending),
	)
	return err
}

func writeRuntimeCommandRedactedPrefix(
	destination io.Writer,
	pending []byte,
	secret []byte,
	replacement []byte,
	flushBefore int,
) ([]byte, error) {
	consumed := 0
	for consumed < flushBefore {
		relativeMatch := bytes.Index(pending[consumed:], secret)
		if relativeMatch < 0 || consumed+relativeMatch >= flushBefore {
			if err := writeRuntimeCommandBytes(destination, pending[consumed:flushBefore]); err != nil {
				return pending, err
			}
			consumed = flushBefore
			break
		}

		match := consumed + relativeMatch
		if err := writeRuntimeCommandBytes(destination, pending[consumed:match]); err != nil {
			return pending, err
		}
		if err := writeRuntimeCommandBytes(destination, replacement); err != nil {
			return pending, err
		}
		consumed = match + len(secret)
	}

	remaining := append(pending[:0], pending[consumed:]...)
	return remaining, nil
}

func writeRuntimeCommandBytes(destination io.Writer, content []byte) error {
	for len(content) > 0 {
		written, err := destination.Write(content)
		if written > 0 {
			content = content[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func readRuntimeCommandDiagnostic(loggerTarget, stdoutPath, stderrPath, runnerKey string) []byte {
	switch strings.ToLower(strings.TrimSpace(loggerTarget)) {
	case "stdout":
		return readRuntimeCommandSpoolTail(stderrPath, runnerKey)
	case "stderr":
		return readRuntimeCommandSpoolTail(stdoutPath, runnerKey)
	default:
		stdout := readRuntimeCommandSpoolTail(stdoutPath, runnerKey)
		stderr := readRuntimeCommandSpoolTail(stderrPath, runnerKey)
		if len(stdout) == 0 {
			return stderr
		}
		if len(stderr) == 0 {
			return stdout
		}
		return bytes.Join([][]byte{stdout, stderr}, []byte("\n"))
	}
}

func readRuntimeCommandSpoolTail(path string, runnerKey string) []byte {
	spool, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer spool.Close()

	info, err := spool.Stat()
	if err != nil {
		return nil
	}
	overlap := int64(0)
	if runnerKey != "" {
		overlap = int64(len(runnerKey) - 1)
	}
	start := info.Size() - int64(runtimeCommandDiagnosticBytes) - overlap
	if start < 0 {
		start = 0
	}
	if _, err := spool.Seek(start, io.SeekStart); err != nil {
		return nil
	}
	var redacted bytes.Buffer
	if err := copyRuntimeCommandOutput(
		io.LimitReader(spool, int64(runtimeCommandDiagnosticBytes)+overlap),
		&redacted,
		runnerKey,
	); err != nil {
		return nil
	}
	content := redacted.Bytes()
	if len(content) > runtimeCommandDiagnosticBytes {
		content = content[len(content)-runtimeCommandDiagnosticBytes:]
	}
	return append([]byte(nil), content...)
}

func newRuntimeFailure(executionError error, output []byte, secrets ...string) error {
	message := sanitizeRuntimeDiagnostic(string(output), secrets...)
	clearAutopilotRunnerKeyBytes(output)

	executionMessage := "unknown child-process failure"
	if executionError != nil {
		executionMessage = sanitizeRuntimeDiagnostic(executionError.Error(), secrets...)
	}
	if message == "" {
		return fmt.Errorf("loadstrike runtime failed: %s", executionMessage)
	}
	return fmt.Errorf(
		"loadstrike runtime failed: %s: %s",
		executionMessage,
		message,
	)
}

func runAutopilotViaPrivateRuntime(request LoadStrikeAutopilotRequest) (LoadStrikeAutopilotResult, error) {
	runtimeExecution, err := newRuntimeArtifactResolver(runtimeResolverConfig{
		Version: RuntimeArtifactVersion(),
		GOOS:    runtimeGOOS(),
		GOARCH:  runtimeGOARCH(),
	}).resolveRuntimeExecution(request.Options.RunnerKey)
	if err != nil {
		return LoadStrikeAutopilotResult{}, err
	}
	defer runtimeExecution.Close()
	runtimePath := runtimeExecution.Path

	tempDir, err := os.MkdirTemp("", "loadstrike-autopilot-*")
	if err != nil {
		return LoadStrikeAutopilotResult{}, fmt.Errorf("create autopilot temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	requestPath := filepath.Join(tempDir, "autopilot-request.json")
	resultPath := filepath.Join(tempDir, "autopilot-result.json")
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return LoadStrikeAutopilotResult{}, fmt.Errorf("marshal autopilot request: %w", err)
	}
	if err := os.WriteFile(requestPath, requestBytes, 0o600); err != nil {
		return LoadStrikeAutopilotResult{}, fmt.Errorf("write autopilot request: %w", err)
	}

	cmd, runnerKeyBytes := newAutopilotRuntimeCommand(
		runtimePath,
		requestPath,
		resultPath,
		request.Options.RunnerKey,
		request.Options.LicenseValidationTimeoutSeconds,
	)
	defer clearAutopilotRunnerKeyBytes(runnerKeyBytes)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return LoadStrikeAutopilotResult{}, newAutopilotRuntimeFailure(
			err,
			output,
			request.Options.RunnerKey,
		)
	}
	clearAutopilotRunnerKeyBytes(output)

	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		return LoadStrikeAutopilotResult{}, fmt.Errorf("read autopilot result: %w", err)
	}

	var result LoadStrikeAutopilotResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return LoadStrikeAutopilotResult{}, fmt.Errorf("decode autopilot result: %w", err)
	}
	return result, nil
}

func newAutopilotRuntimeCommand(
	runtimePath string,
	requestPath string,
	resultPath string,
	runnerKey string,
	licenseValidationTimeoutSeconds float64,
) (*exec.Cmd, []byte) {
	runnerKeyBytes := []byte(runnerKey)

	// runtimePath is returned only after authenticated-manifest and SHA-256 verification;
	// exec.Command receives it and non-secret values as argv without invoking a shell.
	// The runner key is a bounded one-shot stdin payload and never a process argument.
	cmd := exec.Command(
		runtimePath,
		"--autopilot-input", requestPath,
		"--autopilot-output", resultPath,
		"--autopilot-runner-key-stdin",
		"--autopilot-license-validation-timeout-seconds", strconv.FormatFloat(licenseValidationTimeoutSeconds, 'f', -1, 64),
		"--sdk-version", RuntimeArtifactVersion(),
		"--protocol", strconv.Itoa(RuntimeProtocolVersion()),
	)
	cmd.Stdin = bytes.NewReader(runnerKeyBytes)
	return cmd, runnerKeyBytes
}

func newAutopilotRuntimeFailure(executionError error, output []byte, runnerKey string) error {
	message := sanitizeRuntimeDiagnostic(string(output), runnerKey)
	clearAutopilotRunnerKeyBytes(output)

	executionMessage := "unknown child-process failure"
	if executionError != nil {
		executionMessage = sanitizeRuntimeDiagnostic(
			executionError.Error(),
			runnerKey,
		)
	}
	if message == "" {
		return fmt.Errorf("loadstrike autopilot runtime failed: %s", executionMessage)
	}
	return fmt.Errorf(
		"loadstrike autopilot runtime failed: %s: %s",
		executionMessage,
		message,
	)
}

func clearAutopilotRunnerKeyBytes(content []byte) {
	for index := range content {
		content[index] = 0
	}
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
