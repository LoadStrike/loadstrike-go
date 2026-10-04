# LoadStrike Go SDK

The LoadStrike Go SDK lets you build transaction-focused load tests directly in Go.

Use it to define scenarios, execute named steps, apply load simulations and thresholds, and review structured results without moving into a separate DSL or runner model.

## Requirements

- Go 1.26.8 or later

## Install

```bash
go get loadstrike.com/sdk/go
```

`loadstrike.com/sdk/go` is the public vanity module path served by the LoadStrike website and backed by the public `loadstrike/loadstrike-go` repository, so `go get` and pkg.go.dev resolve the same package surface.

## v0.2 Migration

The Go SDK v0.2.2 release pair retains protocol 2 and the Go 1.26.8 toolchain floor. It adds separately launched NATS agents and remote multi-process Load Engine V2, plus event-stream consumer, distributed reporting, sink lifecycle, and pause-scheduling corrections. Before running workloads:

1. Install Go 1.26.8 or later.
2. After both the v0.2.2 wrapper and its matching signed runtime are published, run `go get loadstrike.com/sdk/go@v0.2.2`.
3. Configure a valid runner key and run the workload; normal license validation remains required.

The separate-agent and remote Load Engine V2 support described below requires the matching v0.2.2 signed runtime. The immutable published v0.2.1 runtime predates these changes; updating the public wrapper alone does not add them to that runtime. Existing release tags and runtime artifacts remain unchanged.

The immutable v0.2.0 module metadata permits Go 1.26.5; v0.2.1 and v0.2.2 require Go 1.26.8. Following v0.2.0 publication, v0.1.x remains on security-only support for at least 90 days. The v0.2 protocol and capability changes will not be backported to v0.1.x. The retained v0.1.30401 release requires Go 1.26.5 or later.

Import the package in your Go workload code with:

```go
import loadstrike "loadstrike.com/sdk/go"
```

## Execution

The Go SDK preserves the callback-style authoring model shown in the LoadStrike documentation while keeping the installation and execution workflow simple for application teams.

Install the module, configure a valid runner key, and run workloads directly from Go code. Normal license validation remains required. An invalid runner key or incompatible supported setup makes `Run()` fail before scenario callbacks or workload traffic starts. A separately launched NATS agent can wait without its own runner key, using the verified cached runtime described under Separate Agent Processes; it executes work only after validating the coordinator-authorized command.

## Public API Surface

The public module matches the documented LoadStrike builder and context API.

Use `Create()` or `NewRunner()` to start a builder, reuse contexts with `BuildContext()` and `ConfigureContext(...)`, load JSON settings with `LoadConfig(...)` and `LoadInfraConfig(...)`, and control local report output with `WithReportFolder(...)`, `WithReportFileName(...)`, `WithReportFormats(...)`, and `WithReportingInterval(...)`.

Targeted execution and validation timing are available through `WithTargetScenarios(...)` and `WithLicenseValidationTimeout(...)`.

`WithDisplayConsoleMetrics(...)` and `DisplayConsoleMetrics` are accepted for configuration compatibility, but this Go release does not emit live console snapshots. Use reporting sinks for live delivery and the final run result for completed metrics. `WithScenarioCompletionTimeout(...)` and `ScenarioCompletionTimeoutMs` are also accepted and validated against licensed limits, but this Go release does not apply them as a graceful-shutdown deadline.

New load-test templates can explicitly select the versioned scheduler with `UseLoadEngineV2()` and tune its process-wide in-flight ceiling with `WithMaxInFlight(...)`.

## JSON Configuration

`LoadConfig(path)` applies supported settings from the `LoadStrike` object at the point where it appears in the fluent call chain. A later fluent call or later `LoadConfig(...)` call wins for the same setting.

The supported `LoadStrike` keys are `TestSuite`, `TestName`, `SessionId`, `ReportFolder`, `ReportFileName`, `ReportFormats`, `ReportingIntervalMs`, `ScenarioCompletionTimeoutMs`, `ClusterCommandTimeoutMs`, `NodeType`, `ClusterId`, `AgentGroup`, `AgentId`, `ExpectedAgentIds`, `AgentsCount`, `NatsServerUrl`, `TargetScenarios`, `AgentTargetScenarios`, `CoordinatorTargetScenarios`, `MinimumLogLevel`, `DisplayConsoleMetrics`, `EnableLocalDevCluster`, `RestartIterationMaxAttempts`, `SinkRetryCount`, `SinkRetryBackoffMs`, `RunnerKey`, `WithoutReports`, `UseLoadEngineV2`, and `MaxInFlight`. `LicenseValidation` supports the nested `TimeoutMs` key.

Configuration key matching is case-insensitive. List settings accept either a comma-delimited string or a JSON string array. Boolean settings accept JSON booleans or the strings `true` and `false`. `NodeType` accepts `Single`, `SingleNode`, `Coordinator`, `Agent`, or the values `0`, `1`, and `2`; `ReportFormats` accepts `html`, `txt`, `csv`, `md`, and `markdown`. Other JSON values remain available to callbacks through `CustomSettings()`, but they do not configure LoadStrike behavior. License-validation bypass and service-address settings are not supported.

Local child identities come from `ExpectedAgentIds` when supplied or from their generated indexes, rather than the coordinator's `AgentId`. Load Engine V1 applies `TargetScenarios` and `AgentTargetScenarios` but not `CoordinatorTargetScenarios`. Clustered Go V2 supports local child agents or separately launched NATS agents; both role-specific target lists are rejected. Single-node V2 does not require cluster settings. See Separate Agent Processes for the supported source profile and runtime release requirement.

## What You Can Build

- scenario-based load tests with named steps
- trace-to-test Autopilot starter generation from captured HAR, OpenTelemetry trace JSON, browser recordings, or source and destination message pairs
- HTTP and event-driven transaction workflows
- weighted traffic mixes that split one load profile across scenario lanes
- custom metrics, thresholds, and report generation
- local report output in HTML, TXT, CSV, and Markdown
- coordinator-managed local child agents and separately launched NATS agents, including remote multi-process Load Engine V2 in the updated source/runtime profile below
- supported observability sink integrations on Enterprise

Built-in transport coverage includes HTTP, Kafka, RabbitMQ, NATS, Redis Streams, Azure Event Hubs, AWS SQS, Push Diffusion, and delegate-based custom streams.

gRPC endpoints require a matching `Produce` or `Consume` delegate. Without one, initialization fails with `Native gRPC execution is not available in this SDK version. Provide the endpoint Produce/Consume delegate instead.` WebSocket supports native or delegate-backed Produce and Consume; a matching delegate takes precedence over `NativeClient`.

Kafka OAuthBearer authentication accepts either a direct token through `KafkaSASLOAuthBearerOptions.AccessToken`, or `OAuthBearerTokenEndpointURL` together with `ClientId` and `ClientSecret` in `AdditionalSettings`. Endpoint mode obtains tokens through the configured client-credentials flow; optional `Scope`, `Audience`, and `GrantType` settings are included in the token request.

## Cross-Platform Tracking

Cross-platform tracking normally uses an explicit selector such as `header:X-Correlation-Id` or `json:$.trackingId` on each endpoint. When LoadStrike is generating the source traffic and the messages do not already have a business tracking field, set `UseLoadStrikeTraceIDHeader` to `true` and omit `TrackingField`; the runtime uses `LoadStrikeTraceIDTrackingField` (`header:loadstrike-trace-id`) and injects a GUID into `LoadStrikeTraceIDHeader`.

Source endpoints in `Consume` mode and `CorrelateExistingTraffic` runs observe existing traffic only, so they do not inject `loadstrike-trace-id`. In those monitoring flows the header must already exist on the messages for it to be used for matching.

When both sides of the workflow already exist, set `RunMode` to `CorrelateExistingTraffic` and call `ForDuration(...)` on the tracking configuration. `ForDuration` defines the observation window and accepts an optional `context.Context` to stop observation early. Do not add `WithLoadSimulations(...)` to this mode.

Kafka offsets are committed after successful observation; processing or commit failures are reported. RabbitMQ manual acknowledgement, Redis Streams acknowledgement, SQS deletion and Event Hubs partition failures are distinct from an empty poll. Event Hubs observes individual events without waiting for 25 events. Cancelling the observation stops its consumers; final SQS deletion and Event Hubs cleanup have bounded waits. Use dedicated observer groups and queues because consumption can affect application traffic.

```go
observationContext, stopObservation := context.WithCancel(context.Background())
defer stopObservation()

tracking := (&loadstrike.LoadStrikeTrackingConfigurationSpec{
	RunMode: "CorrelateExistingTraffic",
	Source: &loadstrike.EndpointSpec{
		Kind:          "Kafka",
		Name:          "orders-source",
		Mode:          "Consume",
		TrackingField: "json:$.trackingId",
	},
	Destination: &loadstrike.EndpointSpec{
		Kind:          "Kafka",
		Name:          "orders-completed",
		Mode:          "Consume",
		TrackingField: "json:$.trackingId",
	},
}).ForDuration(loadstrike.DurationFromSeconds(600), observationContext)
```

## Quick Start

```go
package main

import (
	loadstrike "loadstrike.com/sdk/go"
)

func main() {
	scenario := loadstrike.CreateScenario("orders", func(ctx loadstrike.LoadStrikeScenarioContext) loadstrike.LoadStrikeReply {
		return loadstrike.LoadStrikeStep.Run("publish-order", ctx, func(loadstrike.LoadStrikeScenarioContext) loadstrike.LoadStrikeReply {
			return loadstrike.LoadStrikeResponse.Ok("200")
		})
	}).WithLoadSimulations(
		loadstrike.LoadStrikeSimulation.IterationsForConstant(1, 10),
	)

	result := loadstrike.Create().
		AddScenario(scenario).
		UseLoadEngineV2().
		WithMaxInFlight(5000).
		WithRunnerKey("rkl_your_runner_key").
		WithoutReports().
		Run()

	_ = result
}
```

`Run()` returns the full run result, including scenario metrics, generated report files, and sink status information.

## Logging

The Go SDK writes runtime logs to one generated text file per runtime node or process by default, using `Information` as the minimum level. A single-node run produces one file; a clustered run can return coordinator and agent-indexed files. Files are created in the configured report folder, or in `./reports` when no report folder is set, and their paths are returned in `LoadStrikeRunResult.LogFiles()`. LoadStrike closes them before `Run()` returns.

Use `WithMinimumLogLevel(...)` with `LogEventLevelVerbose`, `LogEventLevelDebug`, `LogEventLevelInformation`, `LogEventLevelWarning`, `LogEventLevelError`, or `LogEventLevelFatal`. The selected level and all higher-priority events are written. Names are matched case-insensitively, while aliases, numbers, and blank values are rejected.

`WithLoggerConfig(...)` accepts a factory that returns a `LoggerConfiguration` with the exact keys `target`, `format`, `path`, and `minimumLevel`:

- `target` accepts `file`, `stdout`, or `stderr`. An explicit `file` target requires a nonblank `path`; a `path` without `target` remains a supported shorthand for file output. Do not supply `path` with `stdout` or `stderr`.
- `format` accepts `text` or `json` and defaults to `text`. JSON output contains one object per line with exactly `timestampUtc`, `level`, and `message`.
- `minimumLevel` accepts the same six levels. An explicit `WithMinimumLogLevel(...)` call wins over this map entry regardless of call order; otherwise the map entry applies, then the `Information` default.

Use an explicit `path` only for single-node runs. Local clustered nodes currently reuse the same explicit path and can truncate or contend for it; omit `path` in clustered runs so LoadStrike creates node-specific files.

```go
loadstrike.RegisterScenarios(scenario).
	WithLoggerConfig(func() loadstrike.LoggerConfiguration {
		return loadstrike.LoggerConfiguration{
			"target":       "stdout",
			"format":       "json",
			"minimumLevel": "Verbose",
		}
	}).
	WithMinimumLogLevel(loadstrike.LogEventLevelWarning). // Warning wins over Verbose above.
	WithRunnerKey("rkl_your_runner_key").
	Run()
```

Unknown keys, unsupported values, wrong value types, blank paths, directory paths, and missing explicit file paths fail before test traffic starts. File destinations are created or truncated for the run and closed before it returns. `stdout` and `stderr` destinations are not closed by LoadStrike, do not add a `LogFiles()` entry, and receive their configured output after workload execution stops. This includes the logger's final failure record when execution fails. Known runner-key text is redacted from that output. A failed `Run()` also panics with a bounded, sanitized diagnostic that does not duplicate the selected logger stream.

## Callback Outcomes

Runtime-policy callbacks keep their public outcomes. `ShouldRunScenario(...) == false` skips that scenario with zero workload requests. A policy error is recorded in `PolicyErrors()`; continue mode permits the scenario or step to continue, while fail mode ends the run. `BeforeScenario`, `AfterScenario`, `BeforeStep`, and `AfterStep` run at their named lifecycle points.

Scenario init and clean callbacks receive the configured custom and global settings plus the current test, node, scenario, and partition metadata through `LoadStrikeScenarioInitContext`. Step callbacks receive the same metadata through `LoadStrikeScenarioContext`, plus `ScenarioInstanceData()`. That map persists across step callbacks for one scenario instance, remains isolated from other instances, and is released after lifecycle cleanup. Successful callback log records enter the selected logger once and obey its target, format, and minimum level.

`StopScenario(name, reason)` stops the remaining iterations for the named scenario, while `StopCurrentTest(reason)` requests a run-wide stop. Active scenario callbacks can observe cancellation through `ScenarioCancellationToken()`. For existing-traffic correlation, cancelling the `context.Context` supplied to `ForDuration(...)` ends observation early.

`KafkaReportingSinkOptions.Publish(topic, payload)` is an application-provided reporting publisher callback, not a native Kafka client. LoadStrike passes the configured topic and reporting JSON to it under the reporting retry policy. A recovered delivery leaves no final sink error; exhausted delivery is reported against the Kafka sink without changing successful workload request, OK, or failure counts.

## Load Engine V2

Call `UseLoadEngineV2()` explicitly for the versioned smooth-pacing and bounded-work contract. V2 runs registered scenarios concurrently, spreads fixed-rate arrivals across their interval, and uses one process-wide in-flight ceiling shared by those scenarios and colocated logical agents. Results remain ordered by scenario registration. The default ceiling is 10,000; call `WithMaxInFlight(...)` after the V2 opt-in to override it with a value from 1 through 1,000,000.

The requested rate is offered scenario invocations per interval. Compare it with achieved starts, delivery percentage, scheduler lag, and transport throughput. If the generator is late or at capacity, the arrival is dropped and disclosed as a generator warning rather than counted as an application failure. One scenario invocation may contain several requests or Kafka records, while browser journeys require separate host-capacity planning.

Declare statically known step names with `.WithDeclaredSteps("request", "audit")`. V2 creates those report series even when a step receives no observations; an unexpected runtime step is folded into the bounded `<other>` series instead of expanding memory without limit.

Go V2 global sharding supports the local-development cluster and separately launched NATS processes in the updated source/runtime profile below. It validates a shared canonical plan before work, synchronizes each simulation through readiness and clock checks, drains every planned segment, and validates bounded result artifacts before merging them. Each remote agent has its own process-wide in-flight ceiling. Remote V1 retains its existing execution model.

Capacity evidence is hardware-specific. Follow the [Load Simulation guidance](https://loadstrike.com/docs/library-options/load-simulation) to size a safe run, distinguish offered from achieved load, and interpret the repeatable `scheduler-noop/2` profile. A benchmark artifact is evidence for that exact host and configuration, not a general 300,000-RPS claim.

Generated HTML reports are self-contained offline files with responsive SVG charts. They provide exact-value pointer, touch, and keyboard tooltips; outcome legends; zoom, pan, and reset; an accessible expanded view; chart-title search; and compact, comfortable, or spacious grids. Successful and failed latency stay separate, while All appears only when the run has a genuine combined distribution. When temporal history is available, cumulative requests, achieved request rate, bytes, and per-scenario latency include the final partial reporting interval. Correlation charts retain scenario, destination, status, GatherBy selector/value, and all available percentile points without averaging groups.

## Raw Iteration Reporting

Observation-capable reporting sinks receive one compact record for every scenario attempt, including retry attempts and nested steps. Retries share a logical iteration ID while keeping distinct attempt indexes and final-attempt markers. Warm-up and load phases, simulation and shard identity, timestamps, observed and reported latency, outcome, status code, and response size are included; reply messages, payloads, bodies, and headers are not.

If a fail-mode runtime policy callback fails after an attempt begins, the stream receives one final failed observation with status `runtime_policy_error` before the run terminates. The observation does not include the callback error text.

Records are buffered without delaying scenario callbacks and normally flush in compressed chunks every five seconds, bounded by 50,000 observations or 8 MiB. Buffer pressure, a single record that cannot fit a batch, and per-sink queue pressure drop reporting observations with explicit warnings; they do not turn a successful system-under-test response into an application failure. A completion marker is never sent ahead of an active sink write; a drain timeout is disclosed as incomplete reporting without falsely classifying that active write as dropped. Metric-only destinations disclose that they cannot retain arbitrary strings or nested steps.

Set `SinkRetryCount` and `SinkRetryBackoffMs` on the run configuration to control the reporting retry policy. Every reporting-sink callback—including initialization, start, realtime statistics and metrics, final statistics and metrics, raw batches, completion markers, stop, and dispose—uses the same bounded policy. The defaults are three retries after the initial callback, using delays of 250 ms, 500 ms, and 1 second, and the retry count can be set from zero through 100. A recovered callback adds no final sink error or delivery-failed warning. Only an exhausted raw-observation delivery counts as sink observation loss; other exhausted callbacks are reported against that sink without failing the workload. Custom sinks may receive the same delivery more than once and should handle repeats safely. Stop and dispose remain best-effort cleanup, and an exhausted stop callback does not prevent dispose. Sanitized nested error details stay in the local run log rather than generator warnings, portable results, portal payloads, or HTML reports.

`StatsDReportingSink`, `DogStatsDReportingSink`, and `NetdataStatsDReportingSink` emit native UDP measurements for each captured attempt. Configure them with `StatsDReportingSinkOptions`: `Host` defaults to `127.0.0.1`, `Port` defaults to `8125`, and `Prefix` defaults to `loadstrike`; `Tags` adds static DogStatsD tags. Existing `EndpointURL` configuration remains supported as the destination. A successful UDP send confirms only that the local UDP stack accepted the datagram, because the receiver does not acknowledge delivery.

Prometheus Remote Write, CloudWatch, Dynatrace, and New Relic publish their provider-specific metric protocols and apply the shared [metric input limits](../README.md#metric-input-limits). Configure their `HTTPReportingSinkOptions` with `EndpointURL` plus the relevant `BearerToken`, CloudWatch namespace/region/credentials, `APIToken`, or `LicenseKey`; `StaticTags` and `StaticDimensions` add validated attributes. Credential values are redacted from diagnostics.

Elasticsearch and OpenSearch send one JSON document per reporting call to the caller-supplied `EndpointURL`. Supply a complete endpoint suitable for document ingestion, including the intended index and document route. The application owns that routing and any required authorization headers.

`GenericWebhook` intentionally sends generic LoadStrike JSON rather than a vendor metric body. Use it only with a receiver that accepts that contract. When migrating an old generic metrics URL, choose the matching direct vendor sink if the receiver expects a vendor protocol.

Portal reporting calculates cumulative p50, p75, p95, and p99 from all final load-phase outcomes received for each scenario and run. Separate successful and failed percentiles remain available for diagnosis. The SDK does not send SDK-calculated percentile fields as the authoritative portal or observation-capable sink result.

Custom reporting sinks opt in through the secondary `LoadStrikeIterationBatchSink` interface. Observation capture and bounded delivery stay asynchronous; reporting pressure is disclosed through delivery statistics and warnings without reclassifying successful system-under-test responses as failures.

## Traffic Mixes

Use `LoadStrikeTrafficMix` on Pro and Enterprise plans when one total load profile should be distributed across multiple scenario lanes. For example, a 1000 requests-per-second profile with scenario weights of 60, 30, and 10 sends roughly 600 requests per second to the first scenario, 300 to the second, and 100 to the third.

Each lane is still a normal scenario with its own named steps, thresholds, reports, and portal results. Register the mix with `loadstrike.RegisterTrafficMix(...)` or add it to a runner with `.AddTrafficMix(...)`.

## Trace-To-Test Autopilot

Use `GenerateAutopilot(...)` or `LoadStrikeAutopilot.Generate(...)` to infer a starter plan from a captured artifact. Set `LoadStrikeAutopilotOptions.RunnerKey` so generation can validate the Trace-To-Test Autopilot entitlement. Check `result.Readiness` and `result.ReadinessFailures` first; call `result.BuildScenario()` only when it is `LoadStrikeAutopilotReady`, then execute the scenario through the normal runner with a valid `RunnerKey`.

Use `SecretBindings` to map redaction locations such as `header:Authorization` or `body:$.client_secret` to environment variables, `TrackingSelector` when the selector cannot be inferred, and `EndpointBindings`, `AllowedReplayHosts`, or `BaseURLRewrite` when a replay target must be bound. Secret values are resolved when the generated scenario runs; they are not written into the generated plan. Any gate satisfied by user setup is omitted from `ReadinessFailures`.

Provide the normal runner key on the Autopilot request options so entitlement validation can complete before generation. The generated scenario still runs through `WithRunnerKey(...)` and follows the standard runner-key validation rules.

## Runner Keys

Single-node and coordinator workloads require a valid `RunnerKey`.

Supply it with `.WithRunnerKey(...)` or through your application configuration before calling `Run()`. `Run()` validates the key online before execution starts.

A separate NATS agent may omit its runner key only when a compatible publisher-signed, hash-verified runtime is already installed in its protected cache. Keyless startup never resolves or downloads a runtime; a missing or invalid cached artifact fails before the listener starts. Provision the matching updated runtime through the normal licensed installation path or an approved deployment image. The agent still validates the signed coordinator execution token and command before running callbacks.

## Separate Agent Processes

This section requires the v0.2.2 wrapper and matching signed runtime; it is not a capability claim for the immutable published v0.2.1 runtime.

Run the same application and selected scenario definitions in each agent process and in the coordinator. Configure a reachable NATS server, the same explicit `SessionId`, `ClusterId`, and `AgentGroup`, and a unique stable `AgentId` for each agent. The remote V2 coordinator requires `ExpectedAgentIds` with exactly `AgentsCount` distinct IDs. Use `WithAgentID(...)` and `WithExpectedAgentIDs(...)` on a context, or their JSON configuration keys. The context NATS method is `WithNatsServerUrl(...)`.

The following program can be launched once for each agent and once for the coordinator. Set `LOADSTRIKE_ROLE=agent` and `LOADSTRIKE_AGENT_ID=agent-a` or `agent-b` on the agents; set `LOADSTRIKE_ROLE=coordinator` and `LOADSTRIKE_RUNNER_KEY` on the coordinator. Use one new shared `LOADSTRIKE_SESSION_ID` for that run. Agents need the compatible verified runtime cache described above if no runner key is supplied.

```go
package main

import (
	"os"
	loadstrike "loadstrike.com/sdk/go"
)

func main() {
	scenario := loadstrike.CreateScenario("orders", func(ctx loadstrike.LoadStrikeScenarioContext) loadstrike.LoadStrikeReply {
		return loadstrike.LoadStrikeStep.Run("submit-order", ctx, func(loadstrike.LoadStrikeScenarioContext) loadstrike.LoadStrikeReply {
			return loadstrike.LoadStrikeResponse.Ok("200")
		})
	}).WithLoadSimulations(loadstrike.LoadStrikeSimulation.IterationsForConstant(2, 20))

	run := loadstrike.RegisterScenarios(scenario).
		UseLoadEngineV2().
		WithSessionId(os.Getenv("LOADSTRIKE_SESSION_ID")).
		WithClusterId("orders-cluster").
		WithAgentGroup("orders-agents").
		WithNatsServerUrl(os.Getenv("LOADSTRIKE_NATS_URL")).
		WithoutReports()
	if os.Getenv("LOADSTRIKE_ROLE") == "agent" {
		run.WithNodeType(loadstrike.NodeTypeAgent).
			WithAgentID(os.Getenv("LOADSTRIKE_AGENT_ID")).Run()
		return
	}
	run.WithNodeType(loadstrike.NodeTypeCoordinator).
		WithAgentsCount(2).
		WithExpectedAgentIDs("agent-a", "agent-b").
		WithRunnerKey(os.Getenv("LOADSTRIKE_RUNNER_KEY")).Run()
}
```

`IterationsForConstant(2, 20)` uses two workers and 20 global iterations in V2; it does not mean 20 iterations per worker or per agent. The coordinator schedules work on the expected agents and merges their results. A direct agent `Run()` waits for one coordinator-authorized session and returns after that command completes or fails.

The current remote Go V2 profile supports fixed load simulations and traffic mixes. `TargetScenarios` selects the same global scenario set across participants. Positive warm-up, cross-platform tracking/correlation, `AgentTargetScenarios`, and `CoordinatorTargetScenarios` are rejected for this profile. Local cluster and V1 behavior remains available with its existing limits.

Use a private, authenticated NATS deployment with subject permissions appropriate to the cluster. The control channel carries signed execution tokens and run metadata. The SDK checks token signatures, session and process ownership, canonical plan agreement, registration leases, replay protection, all planned drains, and result integrity; normal coordinator entitlement validation remains required.

## Documentation

- product documentation: https://loadstrike.com/docs
- Go package reference: https://pkg.go.dev/loadstrike.com/sdk/go
- public Go repository: https://github.com/loadstrike/loadstrike-go
