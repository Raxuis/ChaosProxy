# Task 09 — Determinism and header override

- Status: `completed`
- Approval: granted on 2026-09-16
- Depends on: task 08

## Goal

Make sequential runs replayable and add an explicitly enabled one-request override mechanism.

## Checklist

- [x] Generate and log a seed when none is supplied.
- [x] Derive independent per-request RNGs from the seed, rule name, and per-rule request index.
- [x] Document the concurrency limitation honestly.
- [x] Strictly parse opt-in `X-Chaos` overrides and return clear 400 errors.
- [x] Return `X-Chaos-Applied` for assertions.
- [x] Test identical-seed decisions and enabled/disabled override behavior.
- [x] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

- Partially implemented during review R1 (`7ea4a75`, `6102aaf`): each rule draws from
  its own sequence, unrelated traffic does not shift it, and `POST /api/reset`
  replays it. The README documents the concurrency limitation.
- Without `seed` in YAML or `--seed`, the CLI generates a random seed, logs
  `generated seed N, replay this run with --seed N`, and reapplies it on hot reload
  unless the file starts defining `seed`. `SeedConfigured` distinguishes an explicit
  `seed: 0` from a missing key.
- Two handlers with seed 42 produce identical decisions; seed 43 differs.
- `--header-overrides` enables `X-Chaos`, for example `latency=800ms; status=503`,
  with `latency`, `status`, `hang`, `reset`, `truncate`, `bandwidth`, and `off`.
  Parsing rejects unknown or duplicated directives, missing or invalid values,
  `off` combined with faults, and more than one of `status`, `hang`, `reset`,
  with a `400` such as `invalid X-Chaos header: unknown fault "teapot"`.
- Overridden requests bypass rule matching, never advance rule decision counters,
  and have the header removed before the upstream. Responses carry
  `X-Chaos-Applied` with the canonical faults, `off`, or `disabled`, exposed to
  browsers in `reflect` CORS mode.
- A binary smoke test returned 418 with `X-Chaos-Applied: status=418`, forwarded
  `off`, added 300 ms for `latency=300ms`, reset the connection (`curl` exit 56),
  rejected `status=503; hang` with an explicit 400, and answered `disabled` without
  the flag.
- `go test -race`, `go vet`, `staticcheck`, and the dashboard tests pass.

## Log

| Date | Note |
|---|---|
| 2026-09-15 | Per-rule RNG derivation and documentation completed during review R1. |
| 2026-09-16 | Seed generation and X-Chaos header overrides implemented and verified; approval requested. |
| 2026-09-16 | User approved task 09. |
