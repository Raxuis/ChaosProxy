# Task 01 — Minimal reverse proxy spike

- Status: `completed`
- Approval: granted on 2026-09-05
- Depends on: task 00

## Goal

Deliver the phase 0 stdlib-only proxy with a thin CLI and a focused HTTP module.

## Checklist

- [x] Add required `--target` plus `--port` and `--delay` flags.
- [x] Use `httputil.ReverseProxy.Rewrite` and streaming-safe flushing.
- [x] Make delay cancellation-aware and return explicit JSON on upstream failure.
- [x] Log method, path, upstream status, and total duration.
- [x] Add graceful SIGINT/SIGTERM shutdown.
- [x] Keep CLI and server lifecycle in `cmd/chaosproxy`.
- [x] Encapsulate HTTP behavior behind `proxy.NewHandler` in `internal/proxy`.
- [x] Add table-driven CLI tests and handler-level proxy tests.
- [x] Run formatting, compilation, vet, and local integration checks.
- [x] Manually validate or accept GET/POST proxying, two-second delay, and Ctrl+C.
- [x] Receive explicit user approval.

## Acceptance evidence

- `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check` passed.
- A local upstream received GET and POST requests through the proxy unchanged.
- With `--delay 300ms`, observed totals were 303 ms for GET and 305 ms for POST.
- A client timeout after 100 ms canceled the delay before any upstream request.
- An unavailable upstream returned HTTP 502 and `{"error":"upstream unavailable"}`.
- A streaming response had a 2 ms time to first byte and a 1.002 s total time,
  confirming that the one-second gap between chunks was not buffered.
- Ctrl+C logged `shutdown requested` and exited cleanly.
- Missing and non-HTTP targets were rejected with non-zero exit codes.
- `go test -race ./...` passed after the package split.
- `internal/proxy` has 95.5% statement coverage through its public handler interface.
- Real-frontend behavior and the visible two-second skeleton state await user validation.

## Log

| Date | Note |
|---|---|
| 2026-09-04 | Implementation started after task 00 approval. |
| 2026-09-04 | Automated and local integration checks passed; approval requested. |
| 2026-09-04 | User requested an idiomatic multi-file design; task reopened. |
| 2026-09-04 | Split CLI from HTTP behavior and added tests; approval requested again. |
| 2026-09-05 | User approved continuation; implementation is committed as `b583ff1`. |
