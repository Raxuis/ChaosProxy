# Task 11 — JSON payload mutation

- Status: `completed`
- Approval: granted on 2026-09-16
- Depends on: task 10

## Goal

Mutate bounded JSON payloads with nullify, empty, inflate, stretch, and drop operations.

## Checklist

- [x] Gate mutations by JSON content type, probability, and configurable size limit.
- [x] Support dotted object paths and array indexes.
- [x] Implement every operation with logged no-ops for missing paths.
- [x] Recalculate response length after mutation.
- [x] Add table-driven operation, bound, and content-type tests.
- [x] Run `go test ./...`.
- [x] Exercise the fault against a real frontend and record any bug found.
- [x] Receive explicit user approval.

## Acceptance evidence

- `mutate: { probability, max_bytes, operations }` applies `nullify`, `empty`,
  `inflate`, `stretch`, and `drop` to dotted paths with numeric array indexes and
  `*` wildcards. `max_bytes` defaults to 1 MiB and `factor` to 10 for `inflate`
  and `stretch`; validation rejects unknown operations, empty path segments, and
  factors outside 2 to 1000 or on operations that ignore them.
- Only `application/json` and `*+json` bodies are considered. Compressed bodies,
  declared or streamed bodies larger than `max_bytes`, and invalid JSON pass
  through byte-identical with a note. Operations matching nothing leave the body
  intact and add `mutate <op> <path> matched nothing`.
- Notes travel as `faults.Injection.Detail` into the access log `details` field and
  SSE event `details`, shown as a tooltip on dashboard fault cells.
- Mutated bodies keep exact numbers (`UseNumber`) and unescaped HTML, get a new
  `Content-Length`, and lose their `ETag`. Object keys come out sorted.
- Chain order is latency, reset, status, hang, mutate, truncate, bandwidth.
- A binary smoke test nullified an email, stretched a name threefold, dropped every
  item price through `items.*.price`, sent a matching `Content-Length: 123`, and
  logged the no-op for a missing `user.phone`.
- `go test -race`, `go vet`, `staticcheck`, and the dashboard tests pass.
- Not done: exercising the fault against a real frontend application. This needs a
  real app and remains open.

## Log

| Date | Note |
|---|---|
| 2026-09-16 | JSON mutation implemented and verified against the binary; real-frontend check still open; approval requested. |
| 2026-09-16 | User approved task 11; the real-frontend check stays open. |
| 2026-09-17 | Manual check closed in task 15: one nulled article froze the RealWorld Conduit Vue feed with a TypeError; see `docs/launch/launch-copy.md`. |
