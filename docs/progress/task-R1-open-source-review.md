# Task R1 — Code quality and open-source readiness review

- Status: `completed`
- Approval: granted on 2026-09-15
- Depends on: task 06

## Goal

Review the existing codebase before publication and fix correctness, security,
reproducibility, and maintainability issues found along the way.

## Checklist

- [x] Release injected hangs, latencies, and SSE streams on shutdown (`a37789e`).
- [x] Default to loopback listening, protect the control plane, and restrict reflected CORS origins (`8e4a7b3`).
- [x] Draw decisions per rule so unrelated traffic cannot shift them (`7ea4a75`).
- [x] Make truncation abort the connection and support unknown-length bodies (`ed4784b`).
- [x] Encode latency durations as strings in the control-plane JSON (`fa7d07d`).
- [x] Keep rule toggles across hot reloads and publish reloads with compare-and-swap (`660d87f`).
- [x] Make `config.Validate` the single source of validation and CORS defaults (`45df733`).
- [x] Decouple faults from the event model with `faults.Injection` (`8a71f73`).
- [x] Add a pre-push formatting hook (`6184ff7`).
- [x] Speed up the hot path with `math/rand/v2` and an allocation-light matcher (`6102aaf`).
- [x] Adopt idiomatic Go naming and remove per-request nil checks (`2f460b6`).
- [x] Replace the unmaintained `gopkg.in/yaml.v3` with `go.yaml.in/yaml/v3` (`fcadd2b`).
- [x] Add the MIT license, CI, and contributing guide (`584d526`).
- [x] Keep planning records versioned and update them to the reviewed behavior.
- [x] Receive explicit user approval.

## Acceptance evidence

- Ctrl+C with an in-flight hang and a connected SSE client exits with code 0 in
  0.04 s instead of 11 s with code 1.
- Both planes listen on `127.0.0.1` by default; `--host` exposes them with a warning.
  A cross-site `POST /api/reset` and a DNS-rebinding `Host` now return 403.
- Unrelated requests no longer change a rule's decisions for the same seed.
- Truncated responses end with `io.ErrUnexpectedEOF` on the client.
- The handler benchmark dropped from 14.1 µs to 1.76 µs per injected request; matcher
  benchmarks improved 4.5 to 10 times, checked against the previous algorithm.
- CI runs `gofmt`, `go vet`, `staticcheck` v0.8.1, `govulncheck` v1.8.0, and race
  tests on Ubuntu and macOS with Go 1.25 and stable. `actionlint` passes and
  `govulncheck` reports no vulnerabilities.

## Log

| Date | Note |
|---|---|
| 2026-09-15 | Review requested before open-source publication. |
| 2026-09-15 | Twelve review steps implemented, each approved and committed by the user. |
| 2026-09-15 | Planning records versioned again and updated; final approval requested. |
| 2026-09-15 | User approved review R1. |
