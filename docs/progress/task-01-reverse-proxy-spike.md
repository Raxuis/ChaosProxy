# Task 01 — Minimal reverse proxy spike

- Status: `pending`
- Approval: not requested
- Depends on: task 00

## Goal

Deliver the phase 0 stdlib-only proxy in `cmd/chaosproxy/main.go`.

## Checklist

- [ ] Add required `--target` plus `--port` and `--delay` flags.
- [ ] Use `httputil.ReverseProxy.Rewrite` and streaming-safe flushing.
- [ ] Make delay cancellation-aware and return explicit JSON on upstream failure.
- [ ] Log method, path, upstream status, and total duration.
- [ ] Add graceful SIGINT/SIGTERM shutdown.
- [ ] Add automated tests and run `go test ./...`.
- [ ] Manually validate GET/POST proxying, two-second skeleton state, and Ctrl+C.
- [ ] Receive explicit user approval.

## Acceptance evidence

Not started.
