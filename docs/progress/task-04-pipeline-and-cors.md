# Task 04 — Proxy pipeline and CORS

- Status: `awaiting_validation`
- Approval: requested on 2026-09-10
- Depends on: task 03

## Goal

Wire rule matching and faults into the reverse proxy while preserving browser-visible failures and streaming.

## Checklist

- [x] Implement the ordered request/fault/response pipeline.
- [x] Load configuration through an atomic pointer and capture it per request.
- [x] Implement `reflect`, `passthrough`, and `off` CORS modes.
- [x] Handle preflights and add CORS headers to synthetic responses.
- [x] Wire `--config`, `--target`, `--port`, and `--seed` into the CLI.
- [x] Add end-to-end `httptest` coverage for passthrough and each fault.
- [ ] Verify injected 503 responses remain visible as 503 in a browser frontend.
- [x] Run `go test ./...`.
- [ ] Receive explicit user approval.

## Acceptance evidence

- Each request captures one immutable compiled runtime snapshot from an
  `atomic.Pointer`; route globs and fault chains are never rebuilt on the hot path.
- A request-scoped RNG is deterministically derived from the global seed and a
  monotonic request index.
- `Before` faults short-circuit in order, `hang` waits only on request cancellation,
  and `After` faults run through `ModifyResponse` without buffering response bodies.
- `ReverseProxy.FlushInterval` is `-1`; an integration test observes the first SSE
  event before the upstream response completes.
- CORS `reflect`, `passthrough`, and `off` modes are covered, including reflected
  preflights and synthetic `503`/`502` responses.
- An HTTP smoke check against `examples/chaos.yaml` returned `503`, the reflected
  `http://localhost:3001` origin, credentials, `Retry-After`, and the injected JSON body.
- Without a config file, `--target` creates a rule-free passthrough configuration.
- The CLI accepts `--config`, `--target`, `--port`, and `--seed`; target and seed
  flags override their YAML values.
- Structured request logs include matched rule, triggered faults, injected and
  upstream latency, upstream and final status, bytes, duration, and errors.
- `go test -count=1 ./...`, `go test -race -count=1 ./...`, and `go vet ./...` pass.

The real-browser check remains intentionally manual and is not marked complete.

## Log

| Date | Note |
|---|---|
| 2026-09-10 | Pipeline, CLI, CORS modes, streaming, and integration tests completed; approval requested. |
