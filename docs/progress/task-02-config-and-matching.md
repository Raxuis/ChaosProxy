# Task 02 — YAML configuration and route matching

- Status: `completed`
- Approval: granted on 2026-09-05
- Depends on: task 01

## Goal

Add typed YAML configuration, aggregate validation, and precompiled first-match route rules.

## Checklist

- [x] Implement `internal/config` types, duration decoding, loading, and validation.
- [x] Report all validation errors with source lines where possible.
- [x] Implement optional-method matching with `*` and `**` path semantics.
- [x] Ignore query strings and define method case/trailing-slash behavior in tests.
- [x] Add only `gopkg.in/yaml.v3` as a direct dependency.
- [x] Run matcher and validation tests plus `go test ./...`.
- [x] Receive explicit user approval.

## Acceptance evidence

- `Load` strictly rejects unknown fields, empty files, malformed durations, and
  multiple YAML documents.
- Omitted `enabled` values default to `true`; explicit `false` is preserved.
- Validation aggregates target, unique-name, required-fault, status, duration,
  probability, retry, and truncation errors with YAML source lines where available.
- `NaN`, infinities, and out-of-range probabilities/fractions are rejected.
- `*` matches within one non-empty segment; `**` occupies a complete segment and
  matches zero or more segments.
- Methods are optional and case-insensitive; query strings are ignored.
- Trailing slashes are intentionally significant unless covered by a wildcard.
- Disabled rules are skipped and the first enabled match wins.
- `go mod verify`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and
  ten repeated config/rules test runs passed.
- Coverage: `internal/config` 94.5%, `internal/rules` 95.0%.

## Log

| Date | Note |
|---|---|
| 2026-09-05 | Implementation started after task 01 approval. |
| 2026-09-05 | Configuration and matching modules verified; approval requested. |
| 2026-09-05 | User approved continuation; implementation is committed as `6f9e3e7`. |
| 2026-09-15 | Review R1: YAML dependency replaced by `go.yaml.in/yaml/v3` (`fcadd2b`); `cors_origins` added (`8e4a7b3`); validation is the single source of truth and CORS mode must be explicit outside `Load` (`45df733`); the matcher avoids regexps for literal and `*` segments (`6102aaf`). |
