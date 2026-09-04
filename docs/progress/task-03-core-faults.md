# Task 03 — Fault interface and core faults

- Status: `pending`
- Approval: not requested
- Depends on: task 02

## Goal

Create the extensible fault contract, registry, and latency, status, hang, and truncate faults.

## Checklist

- [ ] Implement `Fault`, `Context`, `ShortCircuit`, and embeddable no-op base behavior.
- [ ] Implement fixed/lognormal latency with cancellation.
- [ ] Implement probabilistic status and cancellation-safe hang behavior.
- [ ] Implement streaming truncation without reading the full body.
- [ ] Build faults from a rule in deterministic order.
- [ ] Test cancellation, streaming, configuration errors, and goroutine cleanup.
- [ ] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

Not started.
