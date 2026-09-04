# Task 05 — Configuration hot reload

- Status: `pending`
- Approval: not requested
- Depends on: task 04

## Goal

Reload valid YAML atomically while preserving the last known-good configuration.

## Checklist

- [ ] Add `fsnotify` watcher with 200 ms debounce.
- [ ] Recover watches after editor rename-and-create saves.
- [ ] Reject invalid reloads without crashing or replacing active config.
- [ ] Log readable added, removed, and modified rule diffs.
- [ ] Prove in-flight requests retain their captured config.
- [ ] Add temporary-file integration tests and run `go test ./...`.
- [ ] Manually verify immediate edits and clear invalid-YAML errors.
- [ ] Receive explicit user approval for phase 1.

## Acceptance evidence

Not started.
