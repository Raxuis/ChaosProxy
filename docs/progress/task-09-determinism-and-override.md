# Task 09 — Determinism and header override

- Status: `pending`
- Approval: not requested
- Depends on: task 08

## Goal

Make sequential runs replayable and add an explicitly enabled one-request override mechanism.

## Checklist

- [ ] Generate and log a seed when none is supplied.
- [x] Derive independent per-request RNGs from the seed, rule name, and per-rule request index.
- [x] Document the concurrency limitation honestly.
- [ ] Strictly parse opt-in `X-Chaos` overrides and return clear 400 errors.
- [ ] Return `X-Chaos-Applied` for assertions.
- [ ] Test identical-seed decisions and enabled/disabled override behavior.
- [ ] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

- Partially implemented during review R1 (`7ea4a75`, `6102aaf`): each rule draws from
  its own sequence, unrelated traffic does not shift it, and `POST /api/reset`
  replays it. The README documents the concurrency limitation.

## Log

| Date | Note |
|---|---|
| 2026-09-15 | Per-rule RNG derivation and documentation completed during review R1. |
