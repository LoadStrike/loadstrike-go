# LoadStrike Go SDK

LoadStrike lets Go teams model transaction-focused load tests directly in code.

Use the Go SDK to define scenarios and named steps, apply load simulations and thresholds, and review structured results using the same language and workflow your team already uses for application development.

## Requirements

- Go 1.24 or later
- A valid LoadStrike runner key for runnable workloads

## Install

```bash
go get loadstrike.com/sdk/go
```

```go
import loadstrike "loadstrike.com/sdk/go"
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
		WithRunnerKey("rkl_your_runner_key").
		WithoutReports().
		Run()

	_ = result
}
```

## What You Can Build

- scenario-based load tests with named steps
- transaction flows across HTTP and event-driven systems
- custom metrics, thresholds, and structured reports
- local report output in HTML, TXT, CSV, and Markdown
- distributed and clustered execution on supported plans
- observability and reporting integrations on supported plans

Built-in transport coverage includes HTTP, Kafka, RabbitMQ, NATS, Redis Streams, Azure Event Hubs, Push Diffusion, and delegate-based custom streams.

## Runner Keys

Runnable workloads require a valid `RunnerKey`.

Supply it with `.WithRunnerKey(...)` or through your application configuration before calling `Run()`.

## Documentation

- Product docs: https://loadstrike.com/documentation
- Sample reference: https://github.com/LoadStrike/loadstrike-sdk-sample-reference
