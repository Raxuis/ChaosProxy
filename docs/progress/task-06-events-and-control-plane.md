# Task 06 — Event bus and control plane

- Status: `awaiting_validation`
- Approval: requested on 2026-09-10
- Depends on: task 05

## Goal

Publish non-blocking request events and expose runtime control on port 7071.

## Checklist

- [x] Implement the event model, subscriber buffers, drop accounting, and 500-event ring.
- [x] Expose SSE history/live events with 15-second heartbeats.
- [x] Expose config, rule toggle, reset, and health endpoints.
- [x] Publish one event per data-plane request without blocking the hot path.
- [x] Test slow subscribers, history ordering, cancellation, and goroutine cleanup.
- [x] Run `go test ./...`.
- [ ] Receive explicit user approval.

## Acceptance evidence

- `events.Bus` assigns IDs, retains the latest 500 events in a circular buffer,
  and fans out through independent 128-event subscriber buffers.
- A saturated subscriber drops its oldest queued event; per-subscription and
  aggregate drop counters are exposed without delaying publishers on channel IO.
- Subscriptions capture ordered history and live delivery atomically, close
  idempotently, and require no bus-owned goroutine.
- The SSE endpoint sends history followed by live events, flushes each frame,
  emits 15-second comment heartbeats, and unregisters on request cancellation.
- The control plane exposes `GET /healthz`, `GET /api/events`, `GET /api/config`,
  `PUT /api/rules/{name}`, and `POST /api/reset` on a separate server.
- Runtime toggles use deep configuration copies plus compare-and-swap, so they
  cannot mutate in-flight snapshots or silently overwrite a concurrent update.
- The proxy publishes exactly one completed event per data-plane request with
  method, path, rule, faults, latencies, final status, and response bytes.
- CLI flag `--control-port` defaults to 7071 and is validated against the data
  port; both HTTP servers and the config watcher share coordinated shutdown.
- A real binary smoke test returned health/config responses on port 17071 and
  streamed a data-plane injected 503 as SSE history and live event.
- `go test -count=1 ./...`, 20 repeated event/control runs, 10 repeated proxy
  runs, `go test -race -count=1 ./...`, and `go vet ./...` pass.
- Statement coverage is 97.9% for events, 83.5% for control, and 77.5% for proxy.

## Log

| Date | Note |
|---|---|
| 2026-09-10 | Implementation started after task 05 approval. |
| 2026-09-10 | Event bus, control plane, SSE, runtime toggles, and dual-server wiring completed; approval requested. |
