# Task 10 — Scenario mode

- Status: `completed`
- Approval: granted on 2026-09-16
- Depends on: task 09

## Goal

Add deterministic scripted fault sequences evaluated before probabilistic rules.

## Checklist

- [x] Parse named scenarios with steps and exhaustion behavior.
- [x] Track concurrency-safe counters and support passthrough, repeat, and last.
- [x] Allow any catalog fault in a step.
- [x] Reset counters through the control plane.
- [x] Expose scenario state for test assertions.
- [x] Test exact sequences, exhaustion modes, and reset behavior.
- [x] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

- `scenarios` entries have `name`, `match`, `enabled` (default true),
  `on_exhausted` (`passthrough` by default, `repeat`, or `last`), and `steps`.
  Validation reports duplicate names across rules and scenarios, empty matches,
  unknown exhaustion modes, empty step lists, and invalid steps with their line.
- Steps use the `X-Chaos` directive syntax, so every catalog fault and `off` are
  available. The strict parser moved to `config.ParseFaults` and serves both.
- Selection order is `X-Chaos` header, then scenarios, then rules. An enabled
  scenario owns the requests it matches; overridden requests never advance it.
- Per-scenario atomic counters: 300 concurrent requests on a three-step `repeat`
  scenario produced exactly 100 of each step.
- `GET /api/scenarios` reports steps, served count, next step, and exhaustion.
  `POST /api/scenarios/{name}/reset` restarts one scenario (404 when unknown),
  `POST /api/reset` restarts all, and reloads restart only changed scenarios.
- A binary smoke test returned 503, 502 after 300 ms, then 200 twice; reported the
  exhausted state; and replayed the first step after a single-scenario reset.
- `go test -race`, `go vet`, `staticcheck`, and ten repeated scenario test runs pass.

## Log

| Date | Note |
|---|---|
| 2026-09-16 | Scenario mode implemented and verified; approval requested. |
| 2026-09-16 | User approved task 10. |
