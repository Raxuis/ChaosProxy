# Task 06 — Event bus and control plane

- Status: `pending`
- Approval: not requested
- Depends on: task 05

## Goal

Publish non-blocking request events and expose runtime control on port 7071.

## Checklist

- [ ] Implement the event model, subscriber buffers, drop accounting, and 500-event ring.
- [ ] Expose SSE history/live events with 15-second heartbeats.
- [ ] Expose config, rule toggle, reset, and health endpoints.
- [ ] Publish one event per data-plane request without blocking the hot path.
- [ ] Test slow subscribers, history ordering, cancellation, and goroutine cleanup.
- [ ] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

Not started.
