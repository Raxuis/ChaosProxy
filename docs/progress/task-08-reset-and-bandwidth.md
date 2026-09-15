# Task 08 — TCP reset and bandwidth faults

- Status: `pending`
- Approval: not requested
- Depends on: task 07

## Goal

Add real TCP resets and streaming-safe response throttling.

## Checklist

- [ ] Implement TCP RST through hijacking, linger zero, and close.
- [ ] Fall back clearly on unsupported protocols such as HTTP/2.
- [ ] Implement cancellation-aware rate-limited body reads without buffering.
- [ ] Register and configure both faults.
- [ ] Test connection failure semantics and progressive throttled delivery.
- [ ] Run `go test ./...`.
- [ ] Manually verify the live UI and SSE reconnection after restart.
- [ ] Receive explicit user approval for phase 2.

## Acceptance evidence

Not started.
