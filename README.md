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

## Public Wrapper Surface

The public module matches the documented LoadStrike builder and context wrapper surface.

Use `Create()` or `NewRunner()` to start a builder, reuse contexts with `BuildContext()` and `ConfigureContext(...)`, load JSON settings with `LoadConfig(...)` and `LoadInfraConfig(...)`, and control local report output with `WithReportFolder(...)`, `WithReportFileName(...)`, `WithReportFormats(...)`, and `WithReportingInterval(...)`.

Targeted execution, realtime console metrics, and validation timing are also available on the public wrapper surface through `WithTargetScenarios(...)`, `WithDisplayConsoleMetrics(...)`, and `WithLicenseValidationTimeout(...)`.

## What You Can Build

- scenario-based load tests with named steps
- HTTP and event-driven transaction workflows
- custom metrics, thresholds, and report generation
- local report output in HTML, TXT, CSV, and Markdown
- clustering and distributed execution
- supported observability sink integrations on eligible plans

Built-in transport coverage includes HTTP, Kafka, RabbitMQ, NATS, Redis Streams, Azure Event Hubs, Push Diffusion, and delegate-based custom streams.

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
		WithRunnerKey("rkl_your_runner_key").
		WithoutReports().
		Run()

	_ = result
}
```

`Run()` returns the full run result, including scenario metrics, generated report files, and sink status information.

## Runner Keys

Runnable workloads require a valid `RunnerKey`.

Supply it with `.WithRunnerKey(...)` or through your application configuration before calling `Run()`. `Run()` validates the key online before execution starts.

## Documentation

- product documentation: https://loadstrike.com/documentation
- repository overview: [../../README.md](../../README.md)
- SDK workspace overview: [../README.md](../README.md)
