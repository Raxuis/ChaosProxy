A HTTP-aware chaos proxy for frontend developers — configure latency, errors, truncation and payload mutations per route, in code, reproducibly, in CI.

# Chaos Proxy

Chaos Proxy will sit between a frontend application and its real HTTP API, injecting reproducible failures on a per-route basis.

## Project status

Phase 1 is in progress. The proxy already supports route matching, seeded fault
injection, browser-safe CORS handling, and streaming responses.

See [PROGRESS.md](PROGRESS.md) for the roadmap, current task, and validation gates.

## Local development

Requirements:

- Go 1.22 or newer
- Git

The `--target` is the real API, not the frontend. For example, if the frontend
runs on port `3001` and its API runs on port `9000`, start Chaos Proxy with:

```sh
go run ./cmd/chaosproxy --target http://localhost:9000
```

The proxy listens on `http://localhost:7070` by default. Configure the frontend's
API base URL as `http://localhost:7070`; keep the frontend itself on port `3001`.
Without `--config`, Chaos Proxy is a transparent passthrough and injects no fault.
Runtime configuration and the SSE event stream are available from the separate
control plane on `http://localhost:7071`.

To inject faults, use the example configuration:

```sh
go run ./cmd/chaosproxy --config ./examples/chaos.yaml
```

The first enabled matching rule wins. Edit the target and route patterns in the
YAML to match the API under test. Valid edits are applied automatically after a
short debounce; invalid edits are logged and the last valid configuration stays
active.

Available flags:

```text
--config PATH  YAML configuration file
--target URL   upstream base URL; overrides the YAML value
--port PORT    data-plane listen port (default 7070)
--control-port PORT  control-plane listen port (default 7071)
--seed N       random seed; overrides the YAML value
```

At least one of `--config` or `--target` is required.

Control-plane endpoints:

```text
GET  /healthz
GET  /api/events
GET  /api/config
PUT  /api/rules/{name}  body: {"enabled": false}
POST /api/reset
```

Press Ctrl+C to stop the server gracefully.

## License

MIT (license file will be finalized during the packaging task).
