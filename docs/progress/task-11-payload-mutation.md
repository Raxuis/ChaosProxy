# Task 11 — JSON payload mutation

- Status: `pending`
- Approval: not requested
- Depends on: task 10

## Goal

Mutate bounded JSON payloads with nullify, empty, inflate, stretch, and drop operations.

## Checklist

- [ ] Gate mutations by JSON content type, probability, and configurable size limit.
- [ ] Support dotted object paths and array indexes.
- [ ] Implement every operation with logged no-ops for missing paths.
- [ ] Recalculate response length after mutation.
- [ ] Add table-driven operation, bound, and content-type tests.
- [ ] Run `go test ./...`.
- [ ] Exercise the fault against a real frontend and record any bug found.
- [ ] Receive explicit user approval.

## Acceptance evidence

Not started.
