# Task 03 — Fault interface and core faults

- Status: `awaiting_validation`
- Approval: requested on 2026-09-05
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
- [ ] Receive explicit user approval.

## Acceptance evidence

- `Fault` exposes only `Name`, `Before`, and `After`; `BaseFault` supplies no-op hooks.
- Each request owns its `*rand.Rand`; fault implementations contain immutable configuration.
- Fixed latency supports uniform positive/negative jitter and clamps at zero.
- Lognormal latency derives mu and sigma from p50/p99 once during `Build`.
- An in-progress latency stops promptly when the request context is canceled.
- Status faults return escaped JSON, `Content-Type`, and optional `Retry-After`.
- Hang returns a marker immediately and starts no goroutine; the pipeline will own the
  cancellation wait in task 1.3.
- Truncate wraps known-length bodies without reading them, delegates `Close`, and removes
  both forms of content length. Unknown-length streams remain untouched because computing
  a fraction would otherwise require buffering.
- Registry order is `latency`, `status`, `hang`, `truncate`.
- The shared event type lives in `internal/events`, preventing a future dependency cycle.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, 100 repeated cancellation
  tests, and 10 repeated fault package runs passed.
- `internal/faults` statement coverage is 87.3%.

## Log

| Date | Note |
|---|---|
| 2026-09-05 | Implementation started after task 02 approval. |
| 2026-09-05 | Core faults and deterministic registry verified; approval requested. |
