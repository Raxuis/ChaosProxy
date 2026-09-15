# Task 03 — Fault interface and core faults

- Status: `completed`
- Approval: granted on 2026-09-10
- Depends on: task 02

## Goal

Create the extensible fault contract, registry, and latency, status, hang, and truncate faults.

## Checklist

- [x] Implement `Fault`, `Context`, `ShortCircuit`, and embeddable no-op base behavior.
- [x] Implement fixed/lognormal latency with cancellation.
- [x] Implement probabilistic status and cancellation-safe hang behavior.
- [x] Implement streaming truncation without reading the full body.
- [x] Build faults from a rule in deterministic order.
- [x] Test cancellation, streaming, configuration errors, and goroutine cleanup.
- [x] Run `go test ./...`.
- [x] Receive explicit user approval.

## Acceptance evidence

- `Fault` exposes only `Name`, `Before`, and `After`; `BaseFault` supplies no-op hooks.
- Each request owns its `*rand.Rand`; fault implementations contain immutable configuration.
- Fixed latency supports uniform positive/negative jitter and clamps at zero.
- Lognormal latency derives mu and sigma from p50/p99 once during `Build`.
- An in-progress latency stops promptly when the request context is canceled.
- Status faults return escaped JSON, `Content-Type`, and optional `Retry-After`.
- Hang returns a marker immediately and starts no goroutine; the pipeline will own the
  cancellation wait in task 1.3.
- Truncate keeps the declared length, sends the `at` fraction, and then fails the body
  read so the connection is aborted. Unknown-length bodies are buffered up to 1 MiB to
  compute the cut.
- Registry order is `latency`, `status`, `hang`, `truncate`.
- Faults report `faults.Injection` values and do not depend on `internal/events`.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, 100 repeated cancellation
  tests, and 10 repeated fault package runs passed.
- `internal/faults` statement coverage is 87.3%.

## Log

| Date | Note |
|---|---|
| 2026-09-05 | Implementation started after task 02 approval. |
| 2026-09-05 | Core faults and deterministic registry verified; approval requested. |
| 2026-09-10 | User approved task 03 by asking to continue. |
| 2026-09-15 | Review R1: truncation aborts connections and buffers unknown lengths (`ed4784b`); constructors trust `config.Validate` (`45df733`); faults emit `Injection` (`8a71f73`); RNG uses `math/rand/v2` PCG (`6102aaf`). Evidence updated to the current behavior. |
