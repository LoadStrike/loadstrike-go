# LoadStrike Go SDK

The LoadStrike Go SDK lets you build transaction-focused load tests directly in Go.

Use it to define scenarios, execute named steps, apply load simulations and thresholds, and review structured results without moving into a separate DSL or runner model.

## Requirements

- Go 1.24 or later

## Install

```bash
go get loadstrike.com/sdk/go
```

`loadstrike.com/sdk/go` is the public vanity module path served by the LoadStrike website and backed by the public `loadstrike/loadstrike-go` repository, so `go get` and pkg.go.dev resolve the same package surface.

Import the package in your Go workload code with:

```go
import loadstrike "loadstrike.com/sdk/go"
```

## Execution

The Go SDK preserves the callback-style authoring model shown in the LoadStrike documentation while keeping the installation and execution workflow simple for application teams.

Install the module, configure a valid runner key, and run workloads directly from Go code. The public Go module validates the runner key online before execution starts, so denied keys fail fast before the run begins.

The wrapper also requires the current publisher-authenticated runtime manifest before it downloads or reuses its matching execution component. It verifies the LoadStrike publisher signature, exact release identity, download address, byte count, SHA-256 checksum, and validity window before launch. It does not accept an older unsigned format or launch an unverified component. Runtime integrity is separate from licensing: a verified cached component never bypasses the mandatory runner-key validation performed for every run.

## Public API Surface

The public module matches the documented LoadStrike builder and context API.

Use `Create()` or `NewRunner()` to start a builder, reuse contexts with `BuildContext()` and `ConfigureContext(...)`, load JSON settings with `LoadConfig(...)` and `LoadInfraConfig(...)`, and control local report output with `WithReportFolder(...)`, `WithReportFileName(...)`, `WithReportFormats(...)`, and `WithReportingInterval(...)`.

Targeted execution, realtime console metrics, and validation timing are available through `WithTargetScenarios(...)`, `WithDisplayConsoleMetrics(...)`, and `WithLicenseValidationTimeout(...)`.

New load-test templates can explicitly select the versioned scheduler with `UseLoadEngineV2()` and tune its process-wide in-flight ceiling with `WithMaxInFlight(...)`.

## What You Can Build

- scenario-based load tests with named steps
- trace-to-test Autopilot starter generation from captured HAR, OpenTelemetry trace JSON, browser recordings, or source and destination message pairs
- HTTP and event-driven transaction workflows
- weighted traffic mixes that split one load profile across scenario lanes
- custom metrics, thresholds, and report generation
- local report output in HTML, TXT, CSV, and Markdown
- clustering and distributed execution
- supported observability sink integrations on Enterprise

Built-in transport coverage includes HTTP, Kafka, RabbitMQ, NATS, Redis Streams, Azure Event Hubs, Push Diffusion, and delegate-based custom streams.

Kafka OAuthBearer authentication accepts either a direct token through `KafkaSASLOAuthBearerOptions.AccessToken`, or `OAuthBearerTokenEndpointURL` together with `ClientId` and `ClientSecret` in `AdditionalSettings`. Endpoint mode obtains tokens through the configured client-credentials flow; optional `Scope`, `Audience`, and `GrantType` settings are included in the token request.

## Cross-Platform Tracking

Cross-platform tracking normally uses an explicit selector such as `header:X-Correlation-Id` or `json:$.trackingId` on each endpoint. When LoadStrike is generating the source traffic and the messages do not already have a business tracking field, set `UseLoadStrikeTraceIDHeader` to `true` and omit `TrackingField`; the runtime uses `LoadStrikeTraceIDTrackingField` (`header:loadstrike-trace-id`) and injects a GUID into `LoadStrikeTraceIDHeader`.

Source endpoints in `Consume` mode and `CorrelateExistingTraffic` runs observe existing traffic only, so they do not inject `loadstrike-trace-id`. In those monitoring flows the header must already exist on the messages for it to be used for matching.

When both sides of the workflow already exist, set `RunMode` to `CorrelateExistingTraffic` and call `ForDuration(...)` on the tracking configuration. `ForDuration` defines the observation window and accepts an optional `context.Context` to stop observation early. Do not add `WithLoadSimulations(...)` to this mode.

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

LoadStrike automatically obtains and verifies the exact execution component compatible with the installed SDK. Altered, expired, incomplete, unsigned, or incompatible components are never executed; if a publisher-authenticated component cannot be obtained, the operation fails before test traffic starts.

## Load Engine V2

Call `UseLoadEngineV2()` explicitly for the versioned smooth-pacing and bounded-work contract. V2 runs registered scenarios concurrently, spreads fixed-rate arrivals across their interval, and uses one process-wide in-flight ceiling shared by those scenarios and colocated logical agents. Results remain ordered by scenario registration. The default ceiling is 10,000; call `WithMaxInFlight(...)` after the V2 opt-in to override it with a value from 1 through 1,000,000.

The requested rate is offered scenario invocations per interval. Compare it with achieved starts, delivery percentage, scheduler lag, and transport throughput. If the generator is late or at capacity, the arrival is dropped and disclosed as a generator warning rather than counted as an application failure. One scenario invocation may contain several requests or Kafka records, while browser journeys require separate host-capacity planning.

Declare statically known step names with `.WithDeclaredSteps("request", "audit")`. V2 creates those report series even when a step receives no observations; an unexpected runtime step is folded into the bounded `<other>` series instead of expanding memory without limit.

Go V2 global sharding is currently supported for the in-process local-development cluster. Remote multi-process V2 is not supported and is rejected before traffic. Remote V1 behavior is unchanged.

Capacity evidence is hardware-specific. Follow the [Load Simulation guidance](https://loadstrike.com/docs/library-options/load-simulation) to size a safe run, distinguish offered from achieved load, and interpret the repeatable `scheduler-noop/2` profile. A benchmark artifact is evidence for that exact host and configuration, not a general 300,000-RPS claim.

Generated HTML reports are self-contained offline files with responsive SVG charts. They provide exact-value pointer, touch, and keyboard tooltips; outcome legends; zoom, pan, and reset; an accessible expanded view; chart-title search; and compact, comfortable, or spacious grids. Successful and failed latency stay separate, while All appears only when the run has a genuine combined distribution. When temporal history is available, cumulative requests, achieved request rate, bytes, and per-scenario latency include the final partial reporting interval. Correlation charts retain scenario, destination, status, GatherBy selector/value, and all available percentile points without averaging groups.

## Raw Iteration Reporting

Observation-capable reporting sinks receive one compact record for every scenario attempt, including retry attempts and nested steps. Retries share a logical iteration ID while keeping distinct attempt indexes and final-attempt markers. Warm-up and load phases, simulation and shard identity, timestamps, observed and reported latency, outcome, status code, and response size are included; reply messages, payloads, bodies, and headers are not.

If a fail-mode runtime policy callback fails after an attempt begins, the stream receives one final failed observation with status `runtime_policy_error` before the run terminates. The observation does not include the callback error text.

Records are buffered without delaying scenario callbacks and normally flush in compressed chunks every five seconds, bounded by 50,000 observations or 8 MiB. Buffer pressure, a single record that cannot fit a batch, and per-sink queue pressure drop reporting observations with explicit warnings; they do not turn a successful system-under-test response into an application failure. A completion marker is never sent ahead of an active sink write; a drain timeout is disclosed as incomplete reporting without falsely classifying that active write as dropped. Metric-only destinations disclose that they cannot retain arbitrary strings or nested steps.

The public wrapper forwards `SinkRetryCount` and `SinkRetryBackoffMs` with the run plan. Every reporting-sink callback—including initialization, start, realtime statistics and metrics, final statistics and metrics, raw batches, completion markers, stop, and dispose—uses the same bounded policy. The defaults are three retries after the initial callback, using delays of 250 ms, 500 ms, and 1 second, and the retry count can be set from zero through 100. A recovered callback adds no final sink error or delivery-failed warning. Only an exhausted raw-observation delivery counts as sink observation loss; other exhausted callbacks are reported against that sink without failing the workload. Custom sinks have at-least-once delivery semantics and should deduplicate replay by stable batch or observation identity. Stop and dispose remain best-effort cleanup, and an exhausted stop callback does not prevent dispose. Sanitized nested error details stay in the local run log rather than generator warnings, portable results, portal payloads, or HTML reports.

`StatsDReportingSink`, `DogStatsDReportingSink`, and `NetdataStatsDReportingSink` emit native UDP measurements for each captured attempt. Configure them with `StatsDReportingSinkOptions`: `Host` defaults to `127.0.0.1`, `Port` defaults to `8125`, and `Prefix` defaults to `loadstrike`; `Tags` adds static DogStatsD tags. Existing `EndpointURL` configuration remains supported as the destination.

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

Runnable workloads require a valid `RunnerKey`.

Supply it with `.WithRunnerKey(...)` or through your application configuration before calling `Run()`. `Run()` validates the key online before execution starts.

## Documentation

- product documentation: https://loadstrike.com/docs
- Go package reference: https://pkg.go.dev/loadstrike.com/sdk/go
- public Go repository: https://github.com/loadstrike/loadstrike-go
