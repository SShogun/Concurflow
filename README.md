# ConcurFlow

ConcurFlow is a Go concurrency reference project that processes URLs through a clear pipeline, rejects invalid inputs early, and downloads valid URLs with bounded parallelism and graceful cancellation.

It is intentionally compact, but it demonstrates patterns that matter in production systems: channel-based pipelines, semaphore-backed backpressure, context propagation, structured logging, and clean separation between validation and I/O.

## Project Highlights

- URL normalization and validation through a source -> transform -> sink pipeline.
- Controlled HTTP concurrency with per-request timeouts.
- Graceful shutdown via signal-aware contexts.
- Structured logging across each stage of execution.
- Reusable concurrency primitives in the worker pool and event mux packages.

## What This Project Demonstrates

This repository is useful as a resume project because it shows more than a basic fetcher. It highlights practical engineering decisions around concurrency, safety, and observability:

- Input validation happens before network work begins.
- Backpressure limits the number of in-flight downloads.
- Cancellation is propagated through the full execution path.
- Each major component has a narrow responsibility.
- Error states are explicit and traceable.

## Architecture

1. The application starts from a signal-aware root context.
2. Raw URL strings are wrapped into pipeline records.
3. The source stage emits records into the pipeline.
4. The transform stage trims, parses, and classifies each URL.
5. Invalid URLs are marked with reasons such as empty, missing scheme, or unsupported scheme.
6. Valid URLs are filtered into download requests.
7. The downloader limits concurrency with a semaphore.
8. Each request uses its own timeout.
9. Results are collected and summarized in logs.

## Package Overview

| Package | Responsibility |
| --- | --- |
| `cmd/concurflow` | Program entrypoint and signal-aware shutdown |
| `internal/app` | Orchestrates pipeline execution, filtering, downloading, and summary |
| `internal/pipeline` | Source, transform, and sink stages for URL normalization |
| `internal/downloader` | HTTP fetching with bounded concurrency and timeouts |
| `internal/pool` | Worker pool for bounded job processing |
| `internal/mux` | Fan-in utility for merging event channels |
| `internal/config` | Central configuration defaults |
| `internal/logging` | Structured logger setup |
| `internal/demo` | Testable scenarios that exercise the system |

## Pipeline Behavior

The pipeline is intentionally simple and explicit:

- Source emits raw URL records into a channel.
- Transform trims input, parses URLs, and classifies invalid values.
- Sink collects normalized URLs into a final slice.

Validation reasons are encoded in the output so failures are understandable, not just rejected:

- `empty`
- `missing_scheme`
- `unsupported_scheme`

## Downloader Behavior

The downloader applies backpressure before it applies network pressure.

- `MaxConcurrentDownloads` sets the number of in-flight requests.
- `PerDownloadTimeout` bounds each HTTP request.
- Context cancellation stops work cleanly.
- Non-2xx responses are returned as structured failures.

This is the main design idea behind ConcurFlow: control the fan-out instead of letting it grow without limits.

## Supporting Components

The repository also includes reusable concurrency building blocks:

- `pool` provides bounded job processing with a worker pool.
- `mux` merges multiple event streams into one output channel.

These packages are useful on their own, even though the main demo focuses on URL validation and downloading.

## Configuration Defaults

| Setting | Default | Meaning |
| --- | --- | --- |
| `WorkerCount` | `5` | Number of worker goroutines in the pool |
| `QueueDepth` | `100` | Buffered queue depth for submitted jobs |
| `PipelineBufferSize` | `10` | Buffer depth for pipeline stages |
| `MaxConcurrentDownloads` | `3` | Maximum concurrent HTTP requests |
| `PerDownloadTimeout` | `10s` | Timeout for each individual download |
| `RunTimeout` | `1m` | Overall operation timeout |

## Requirements

- Go 1.25 or newer
- Standard library only

## Quick Start

### Build

```bash
go build ./cmd/concurflow
```

### Run

```bash
go run ./cmd/concurflow
```

### Test

```bash
go test ./...
```

The default run processes a sample mix of valid and invalid URLs so you can see validation, filtering, rate limiting, and cancellation behavior in one place.

## Demo Scenarios

The demo package contains scenarios that are useful for walkthroughs and testing:

- Basic flow
- Cancellation handling
- Backpressure and rate limiting
- Invalid URL handling
- Mixed valid and invalid traffic

## Error Handling

ConcurFlow handles failures explicitly instead of hiding them:

- Empty URLs are rejected during transformation.
- URLs without a supported scheme are flagged before download.
- Download timeouts are enforced per request.
- Non-2xx HTTP responses are returned as structured errors.
- Context cancellation is respected across all stages.

## Resume Value

This project is a strong resume piece because it shows practical system design, not just syntax. It demonstrates an understanding of concurrency control, cancellation, observability, and modular Go code structure.

## Limitations

This repository is intentionally scoped as a reference implementation. It does not include:

- Persistent storage
- Retry logic
- Authentication
- Distributed processing
- Advanced crawling beyond scheme validation

## License

Educational example. Use freely.
