# Task 08 — TCP reset and bandwidth faults

- Status: `completed`
- Approval: granted on 2026-09-16
- Depends on: task 07

## Goal

Add real TCP resets and streaming-safe response throttling.

## Checklist

- [x] Implement TCP RST through hijacking, linger zero, and close.
- [x] Fall back clearly on unsupported protocols such as HTTP/2.
- [x] Implement cancellation-aware rate-limited body reads without buffering.
- [x] Register and configure both faults.
- [x] Test connection failure semantics and progressive throttled delivery.
- [x] Run `go test ./...`.
- [x] Manually verify the live UI and SSE reconnection after restart.
- [x] Receive explicit user approval for phase 2.

## Acceptance evidence

- `reset: { probability }` short-circuits before the upstream. The handler hijacks
  the connection, sets `SO_LINGER` to zero, and closes it, so clients receive a TCP
  RST: `curl` exits with code 56 and a Go client gets `ECONNRESET`.
- When hijacking is unsupported, as over HTTP/2, the handler aborts the stream with
  `http.ErrAbortHandler` and the access log explains the fallback. An HTTP/2 TLS
  test server covers this path.
- `bandwidth: { bytes_per_second }` wraps the upstream body and paces reads in
  tenth-of-a-second chunks without buffering. A 96 KB file at 32 KB/s arrived in
  2.93 s, byte-identical, with `Content-Length` kept and the first byte after 4 ms.
- Throttled reads stop when the client disconnects or shutdown begins: Ctrl+C during
  a throttled download exited with code 0 in 0.03 s.
- Chain order is latency, reset, status, hang, truncate, bandwidth. Configuration,
  validation, cloning, `/api/config`, the dashboard labels, the README, and
  `examples/chaos.yaml` cover both faults.
- The live dashboard and SSE reconnection after a proxy restart were verified in the
  browser during task 07 on 2026-09-16.
- `go test -race`, `go vet`, `staticcheck`, ten repeated reset and bandwidth test
  runs, and the dashboard `node --test` suite pass.

## Log

| Date | Note |
|---|---|
| 2026-09-16 | Reset and bandwidth faults implemented and verified; approval for task 08 and phase 2 requested. |
| 2026-09-16 | User approved task 08 and phase 2. |
