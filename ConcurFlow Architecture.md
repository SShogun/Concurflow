# ConcurFlow - Guided Build Order

## Why this document exists

This document is here because the hard part is not the idea of ConcurFlow.
The hard part is knowing what to build first, what each file owns, and what not to put in a file yet.

If you are new to this, read it like a checklist, not like a theory essay.

The rules here are simple:

- build one shape at a time
- keep each file small and focused
- do not mix pool logic, pipeline logic, downloader logic, and mux logic in the same place
- if a file starts feeling confusing, stop and ask what that file owns

This is not a code dump.
It is a guided construction plan.

## Where ConcurFlow sits on the ladder

- Phase Alpha = learn one clean goroutine or channel shape at a time
- Phase Beta = keep two shapes clean inside one repo
- Phase Gamma = add real timeout and resource pressure
- Phase 0 / ConcurFlow = combine the earlier shapes into one app

The important mental map is:

- the pool is the fixed worker-count shape
- the pipeline is the source -> transform -> sink shape
- the downloader is the HTTP + timeout shape
- the mux is the fan-in shape
- the app is the coordinator that ties them together

If any part feels blurry, go back to the earlier shape that it reminds you of.

## What is actually new in ConcurFlow

ConcurFlow adds a few things that are easy to mix up if you are new.

### 1. Ownership crosses package boundaries

In a small exercise, one file can own most of the logic.
In ConcurFlow, packages must share work without sharing ownership.

You must always know:

- who creates a channel
- who sends on it
- who closes it
- who reads from it
- who waits for shutdown

### 2. One root context controls the whole run

The root context is the stop signal for the whole app.

That means cancellation must reach:

- the pool workers
- the pipeline stages
- the downloader requests
- the mux forwarding loop
- the app-level orchestration

### 3. The mux is a real subsystem, not a helper

The mux teaches fan-in.
It merges multiple event channels into one.
It must stop cleanly when inputs close or when the root context is canceled.

### 4. Shutdown order matters

Several subsystems will be alive at once.
You need to shut them down in an order that does not break channel ownership.

### 5. Logs are part of the lesson

Logs are not decoration.
They should tell you what started, what blocked, what timed out, what got canceled, and what exited cleanly.

## Final target shape

```text
concurflow/
  go.mod
  cmd/
    concurflow/
      main.go
  internal/
    app/
      app.go
      config.go
    logging/
      logger.go
    pool/
      types.go
      pool.go
      worker.go
    pipeline/
      types.go
      pipeline.go
      source.go
      transform.go
      sink.go
    downloader/
      types.go
      client.go
      downloader.go
    mux/
      types.go
      mux.go
    demo/
      demo.go
      scenarios.go
```

That is the destination.
Do not try to build everything at once.

## File-by-file top-up guide

Use this section as a writing checklist.
It tells you what belongs in each file and what stays out.

### `go.mod`

What to write:

- the module name that matches your imports
- the Go version

What stays out:

- extra dependencies unless a later file really needs them

### `cmd/concurflow/main.go`

What to write:

- create the root context from OS interrupt and terminate signals
- defer cancel immediately
- build the config
- build the logger
- call `app.Run`
- exit non-zero if `app.Run` returns an error

What stays out:

- worker creation
- pipeline logic
- downloader logic
- mux logic
- scenario logic

### `internal/app/config.go`

What to write:

- one `Config` struct for runtime settings
- `DefaultConfig()`

Recommended fields:

- `WorkerCount`
- `QueueDepth`
- `PipelineBufferSize`
- `MaxConcurrentDownloads`
- `PerDownloadTimeout`
- `RunTimeout`

Simple beginner defaults:

- `WorkerCount = 4`
- `QueueDepth = 8`
- `PipelineBufferSize = 8`
- `MaxConcurrentDownloads = 3`
- `PerDownloadTimeout = 2 * time.Second`
- `RunTimeout = 10 * time.Second`

Why these values:

- they are small enough to show backpressure
- they are large enough that the normal case should complete
- they make timeout behavior visible without waiting too long

What stays out:

- goroutines
- channels
- logger setup
- HTTP setup

### `internal/logging/logger.go`

What to write:

- `func New() *slog.Logger`
- a text logger handler
- readable default log level

Useful fields to keep in logs:

- `component`
- `scenario`
- `worker_id`
- `job_id`
- `item_id`
- `url`
- `reason`
- `duration_ms`

What stays out:

- global mutable logger state
- subsystem-specific business logic

### `internal/app/app.go`

What to write:

- `func Run(ctx context.Context, cfg Config, logger *slog.Logger) error`
- derive a run-scoped timeout from the config
- create and wire the subsystems
- start them in a clear order
- wait for them to finish
- return the final error or nil

The app should coordinate, not do the real work.

What stays out:

- worker loops
- HTTP fetch code
- pipeline stage code
- mux forwarding code

### `internal/pool/types.go`

What to write:

- a job type
- a result type

Suggested shape:

- `URLJob` with `ID`, `URL`, `CreatedAt`
- `JobResult` with `JobID`, `Status`, `Err`, `Duration`

What stays out:

- methods
- goroutines
- queue logic

### `internal/pool/pool.go`

What to write:

- the `Pool` struct
- `New`
- `Start`
- `Submit`
- `Results`
- `Shutdown`

What each one should do:

- `New` creates the jobs and results channels
- `Start` launches exactly `WorkerCount` workers
- `Submit` sends jobs into the queue and blocks when the queue is full
- `Results` returns the results channel for reading
- `Shutdown` closes jobs, waits for workers, then closes results

What stays out:

- pipeline logic
- downloader logic
- mux logic

Important rule:

- `Submit` should block naturally when the queue is full
- `Submit` should also stop early if `ctx.Done()` fires

### `internal/pool/worker.go`

What to write:

- the worker loop
- the job processing helper

What the worker should do:

- read jobs from the jobs channel
- process one job at a time
- send a result to the results channel
- stop when the jobs channel closes or the context is canceled

What stays out:

- queue creation
- channel closing
- app-level orchestration

### `internal/pipeline/types.go`

What to write:

- a raw input type
- a normalized output type
- a small reason type or reason constants

Suggested shape:

- `RawURL` with `ID`, `URL`
- `NormalizedURL` with `ID`, `URL`, `Valid`, `Reason`

Reason values to support:

- `ok`
- `empty`
- `missing_scheme`
- `unsupported_scheme`

What stays out:

- stage logic
- channel logic

### `internal/pipeline/source.go`

What to write:

- `Source(ctx, inputs, logger) <-chan RawURL`
- one goroutine that emits raw items
- close the output channel when finished

What the source should do:

- loop over the input slice
- send one item at a time
- stop if the context is canceled

What stays out:

- validation
- normalization
- result collection

### `internal/pipeline/transform.go`

What to write:

- `Transform(ctx, in, logger) <-chan NormalizedURL`
- one goroutine that reads raw items and emits normalized items
- close the output channel when the input is drained

What the transform should do:

- trim whitespace
- check for empty input
- parse the URL
- mark invalid URLs as invalid
- accept only `http` and `https`

What stays out:

- collection
- downloader requests
- pool submission

Important rule:

- every send should still respect `ctx.Done()`

### `internal/pipeline/sink.go`

What to write:

- `Sink(ctx, in, logger) ([]NormalizedURL, error)`
- read until the input channel closes
- stop early on context cancellation
- return the collected slice

What the sink should do:

- append every normalized item it receives
- return the full slice after the channel closes

What stays out:

- source creation
- transform logic
- downloader logic

### `internal/pipeline/pipeline.go`

What to write:

- a small orchestration function
- wire `Source` -> `Transform` -> `Sink`
- return the collected normalized items and any error

What stays out:

- pool logic
- downloader logic
- mux logic

### `internal/downloader/types.go`

What to write:

- a request type
- a result type

Suggested shape:

- `DownloadRequest` with `ID`, `URL`
- `DownloadResult` with `ID`, `URL`, `StatusCode`, `Err`, `Duration`

What stays out:

- HTTP calls
- goroutines
- semaphore logic

### `internal/downloader/client.go`

What to write:

- one function that performs one HTTP request
- use `http.NewRequestWithContext`
- use an HTTP client
- return a structured result

What this file should not do:

- no queueing
- no worker pool logic
- no semaphore logic
- no fan-in logic

This file should answer one question only:

- given one request, how do we fetch it and record the result?

### `internal/downloader/downloader.go`

What to write:

- `Downloader` struct
- constructor
- `Run`

What `Run` should do:

- take many download requests
- use a semaphore to limit concurrency
- launch per-request goroutines
- wait for all requests to finish
- return a slice of results

What stays out:

- pipeline logic
- mux logic
- app coordination logic

### `internal/mux/types.go`

What to write:

- one shared status event type

Suggested shape:

- `StatusEvent` with `Source`, `Kind`, `ItemID`, `Detail`, `Timestamp`

### `internal/mux/mux.go`

What to write:

- `Merge(ctx, logger, inputs ...<-chan StatusEvent) <-chan StatusEvent`
- one goroutine that fan-ins the input channels
- close the output when all inputs are done

What the mux should do:

- forward events from many inputs to one output
- stop when the context is canceled
- stop paying attention to closed inputs

What stays out:

- formatting
- summaries
- business decisions about the events

### `internal/demo/scenarios.go`

What to write:

- a scenario type
- a function that returns all scenarios
- per-scenario config overrides

Good starter scenarios:

- `normal`
- `cancellation`
- `overload`
- `timeout`

### `internal/demo/demo.go`

What to write:

- `Run(ctx, scenario, logger) error`
- choose a scenario
- apply the scenario config changes
- call the app
- print a short summary

What stays out:

- hidden subsystem logic
- extra architecture rules

## Integration rules

These are the rules that keep the whole app sane.

### Pipeline -> Pool

- only valid normalized items become pool jobs
- invalid items are logged and skipped
- the pipeline does not close pool channels

### Pool -> Mux

- the pool emits events and results
- the pool owns the results channel closure
- the mux only forwards events

### Pipeline -> Downloader

- only valid URLs should become download requests
- downloader errors should be item-level, not app-level, by default

### Shutdown ownership

- `main` owns the root signal context
- `app` owns orchestration
- each subsystem closes only the channels it created

## What the logs should tell you

For a normal run, the logs should make this story readable:

1. the app started
2. the pipeline emitted items
3. invalid URLs were marked invalid
4. the pool processed jobs
5. the downloader fetched allowed URLs
6. the mux forwarded events
7. the app shut down cleanly

For cancellation or timeout, the logs should show the reason clearly.

## Done criteria

The phase is not done until these are true:

- normal runs finish cleanly
- overload shows backpressure
- timeout stops slow requests
- cancellation stops the run promptly
- receivers do not close channels they do not own
- race testing stays clean once tests exist

## Build order

Build in this order:

1. config and logging
2. data types
3. worker pool
4. pipeline
5. pipeline to pool connection
6. downloader
7. mux
8. app and main
9. demo scenarios and tests

If you change the order too early, the bugs get harder to understand.

## Phase 0 - skeleton before concurrency

### Goal

Make the repo boring before it becomes concurrent.

### Create now

- `go.mod`
- package folders under `cmd/` and `internal/`
- `internal/app/config.go`
- `internal/logging/logger.go`

### Do not add yet

- goroutines
- channels
- downloader code
- mux code

## Phase 1 - shared shapes before shared behavior

### Goal

Define the data boundaries before writing the concurrency.

### Create now

- `internal/pool/types.go`
- `internal/pipeline/types.go`
- `internal/downloader/types.go`
- `internal/mux/types.go`

### Do not add yet

- loops
- goroutines
- channel creation

## Phase 2 - build the worker pool in isolation

### Goal

Prove that bounded queueing and fixed worker lifetime are stable.

### Create now

- `internal/pool/pool.go`
- `internal/pool/worker.go`

### The pool should:

- start exactly `WorkerCount` workers
- block `Submit` when the queue is full
- unblock `Submit` when `ctx.Done()` fires
- close jobs during shutdown
- close results only after all workers exit

### Do not add yet

- downloader
- mux
- fancy processing logic

## Phase 3 - build the pipeline in isolation

### Goal

Prove that stage ownership stays clean.

### Create now

- `internal/pipeline/source.go`
- `internal/pipeline/transform.go`
- `internal/pipeline/sink.go`
- `internal/pipeline/pipeline.go`

### The pipeline should:

- source items from a slice
- normalize and validate URLs
- collect normalized items in the sink
- close only the channels it created
- respect cancellation in every blocking step

### Do not add yet

- downloader
- mux
- app-wide wiring

## Phase 4 - connect pipeline to pool

### Goal

Connect two familiar shapes without losing clarity.

### What to wire

- pipeline output feeds the pool
- invalid items are skipped or tagged
- backpressure can move upstream

### Do not add yet

- downloader
- mux
- extra app complexity

## Phase 5 - build the downloader in isolation

### Goal

Bring in real HTTP pressure without mixing in the whole app.

### Create now

- `internal/downloader/client.go`
- `internal/downloader/downloader.go`

### The downloader should:

- limit concurrency with a semaphore
- use context-aware HTTP requests
- return structured results
- release permits with `defer`

### Do not add yet

- mux
- app orchestration

## Phase 6 - build the mux in isolation

### Goal

Learn fan-in and closed-channel handling.

### Create now

- `internal/mux/mux.go`

### The mux should:

- merge many event channels into one
- stop when the root context is canceled
- close output when all inputs are done

### Do not add yet

- printing logic
- summary logic
- business rules

## Phase 7 - wire the app and add main

### Goal

Combine the isolated pieces into one run.

### Create or finish now

- `internal/app/app.go`
- `cmd/concurflow/main.go`

### The app should:

- own orchestration
- start subsystems in order
- shut them down in order
- keep `main` thin

## Phase 8 - add scenarios and verification

### Goal

Prove the architecture survives pressure.

### Create now

- `internal/demo/demo.go`
- `internal/demo/scenarios.go`
- tests across packages

### Suggested scenarios

- normal
- cancellation
- overload
- timeout

## Extra thought points

These are the places that usually confuse beginners.

### Worker pool limit vs semaphore limit

- worker pool = fixed number of long-lived workers
- semaphore = fixed number of in-flight actions

### Root cancellation vs per-item timeout

- root cancellation = stop the whole run
- per-item timeout = stop one slow item

### Closed channels in select

- a closed receive is always ready
- if you keep selecting a closed channel, your loop can get stuck

### Ownership across packages

- always know who created the channel
- always know who closes it
- always know who ranges over it

## Things to skip on purpose

Do not add these yet:

- retries and backoff
- fancy CLI options
- metrics backends
- external config systems
- databases
- real internet dependency in tests
- abstractions that hide channel ownership

## Final mindset rule

Do not try to win ConcurFlow by making it clever.

Win it by keeping the shapes understandable.

If something feels vague, it means the file ownership is still too fuzzy.
Go back to the file list and ask: what is this file responsible for, and what is it not responsible for?
