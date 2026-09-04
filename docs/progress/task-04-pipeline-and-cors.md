# Task 04 — Proxy pipeline and CORS

- Status: `pending`
- Approval: not requested
- Depends on: task 03

## Goal

Wire rule matching and faults into the reverse proxy while preserving browser-visible failures and streaming.

## Checklist

- [ ] Implement the ordered request/fault/response pipeline.
- [ ] Load configuration through an atomic pointer and capture it per request.
- [ ] Implement `reflect`, `passthrough`, and `off` CORS modes.
- [ ] Handle preflights and add CORS headers to synthetic responses.
- [ ] Wire `--config`, `--target`, `--port`, and `--seed` into the CLI.
- [ ] Add end-to-end `httptest` coverage for passthrough and each fault.
- [ ] Verify injected 503 responses remain visible as 503 in a browser frontend.
- [ ] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

Not started.
