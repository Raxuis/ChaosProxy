# Task 05 — Configuration hot reload

- Status: `completed`
- Approval: granted on 2026-09-10
- Depends on: task 04

## Goal

Reload valid YAML atomically while preserving the last known-good configuration.

## Checklist

- [x] Add `fsnotify` watcher with 200 ms debounce.
- [x] Recover watches after editor rename-and-create saves.
- [x] Reject invalid reloads without crashing or replacing active config.
- [x] Log readable added, removed, and modified rule diffs.
- [x] Prove in-flight requests retain their captured config.
- [x] Add temporary-file integration tests and run `go test ./...`.
- [ ] Manually verify immediate edits and clear invalid-YAML errors.
- [x] Receive explicit user approval for phase 1.

## Acceptance evidence

- `config.Watcher` watches the parent directory and filters events by canonical
  configuration path, so atomic rename-and-create editor saves do not invalidate
  the watch.
- Relevant write/create/rename/remove bursts reset a 200 ms timer and produce one
  reload of the final file contents.
- Candidates are parsed and validated before `Handler.Update` compiles route
  matchers and fault chains. Only a fully compiled runtime is published, through
  compare-and-swap, keeping API toggles whose rule `enabled` value did not change.
- Syntax errors, semantic validation errors, and matcher compilation errors keep
  the last known-good runtime active and produce an explicit rejection log.
- Reload logs report rules added, removed, and modified, including rule-order
  changes, plus target, seed, and CORS changes.
- CLI target and seed overrides are reapplied to every candidate before update.
- A blocked upstream integration test proves an in-flight request completes with
  its original target and truncate chain while the following request uses the new
  runtime.
- Temporary-file tests cover live proxy updates, atomic file replacement, invalid
  candidates, debounce behavior, and watcher cancellation.
- `go test -count=1 ./...`, ten repeated config/proxy runs,
  `go test -race -count=1 ./...`, and `go vet ./...` pass.

The live editor/manual browser check remains intentionally unverified until user
validation.

## Log

| Date | Note |
|---|---|
| 2026-09-10 | Implementation started after task 04 approval. |
| 2026-09-10 | Hot reload and automated verification completed; approval requested. |
| 2026-09-10 | User approved task 05 and phase 1 by asking to continue. |
| 2026-09-15 | Review R1: reloads keep rule toggles and use compare-and-swap (`660d87f`). The manual editor check is still pending. |
