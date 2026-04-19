# LoadStrike Go SDK

The public Go wrapper for LoadStrike is installed as `loadstrike.com/sdk/go`.

It preserves the callback-style authoring model shown in the website docs, but it does not expose the private execution engine source.

## Requirements

- Go 1.24 or later

## Install

```bash
go get loadstrike.com/sdk/go
```

```go
import loadstrike "loadstrike.com/sdk/go"
```

## Runtime Delivery

On first run, the public wrapper resolves the exact matching private `loadstrike-runtime` binary for the current platform, verifies it, caches it locally, and runs the workload through that runtime.

The wrapper version and runtime version are locked together so the public API and private engine do not drift.

## Local Development Overrides

- `LOADSTRIKE_RUNTIME_PATH`: use a prebuilt local runtime binary instead of downloading one
- `LOADSTRIKE_RUNTIME_BASE_URL`: override the licensing API base URL used to resolve runtime manifests
- `LOADSTRIKE_RUNTIME_CACHE_DIR`: override the local runtime cache directory
- `LOADSTRIKE_RUNTIME_NO_DOWNLOAD=1`: disable first-run downloads
- `LOADSTRIKE_DEV_USE_LOCAL_CLIENT=1`: use the local in-process stub for host-side tests

## Verification

```bash
go test ./...
```
