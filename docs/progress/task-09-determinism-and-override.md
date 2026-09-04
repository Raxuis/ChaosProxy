# Task 09 — Determinism and header override

- Status: `pending`
- Approval: not requested
- Depends on: task 08

## Goal

Make sequential runs replayable and add an explicitly enabled one-request override mechanism.

## Checklist

- [ ] Generate and log a seed when none is supplied.
- [ ] Derive independent per-request RNGs from seed and request index.
- [ ] Document the concurrency limitation honestly.
- [ ] Strictly parse opt-in `X-Chaos` overrides and return clear 400 errors.
- [ ] Return `X-Chaos-Applied` for assertions.
- [ ] Test identical-seed decisions and enabled/disabled override behavior.
- [ ] Run `go test ./...` and receive explicit user approval.

## Acceptance evidence

Not started.
