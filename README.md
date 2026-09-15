A HTTP-aware chaos proxy for frontend developers — configure latency, errors, truncation and payload mutations per route, in code, reproducibly, in CI.

# Chaos Proxy

Chaos Proxy will sit between a frontend application and its real HTTP API, injecting reproducible failures on a per-route basis.

## Project status

Phase 1 is in progress. The proxy already supports route matching, seeded fault
injection, browser-safe CORS handling, and streaming responses.

See [PROGRESS.md](PROGRESS.md) for the roadmap, current task, and validation gates.

## Local development

Requirements:

- Go 1.25 or newer
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
--host HOST    listen address for both planes (default 127.0.0.1)
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

A rule toggled through `PUT /api/rules/{name}` keeps its state across
configuration reloads until the file changes that rule's `enabled` value or
removes the rule.

Press Ctrl+C to stop the server gracefully.

## Truncation

`truncate` sends the first `at` fraction of the body, then aborts the
connection, so clients see a network error such as `unexpected EOF` instead of
a short but valid response. The declared `Content-Length` is kept. Responses
without a length (chunked, compressed or streamed) are buffered up to 1 MiB to
compute the cut, and longer ones are cut at `at` of that first MiB. Avoid
truncate rules on endless streams such as Server-Sent Events: nothing is sent
until 1 MiB has arrived.

## Reproducibility

Each rule draws its decisions from the seed, the rule name, and the number of
requests that rule has matched. Unmatched requests and other rules never shift
a rule's sequence, and `POST /api/reset` restarts every sequence. Concurrent
requests to the same rule can still arrive in a different order between runs.

## Security defaults

Chaos Proxy is a local development tool and its control plane has no
authentication.

- Both planes listen on `127.0.0.1`. Use `--host 0.0.0.0` only on an isolated
  network, such as inside a container.
- The control plane only answers requests whose `Host` is `localhost` or an IP
  address, and rejects cross-origin browser writes.
- With `cors: reflect`, only loopback origins (`localhost`, `*.localhost`,
  `127.0.0.1`, `[::1]`) receive reflected CORS headers. Other origins get the
  upstream CORS headers unchanged. A non-empty `cors_origins` list replaces the
  loopback default:

```yaml
cors: reflect
cors_origins:
  - http://localhost:3001
  - https://app.test
```

`"*"` reflects every origin with credentials; keep it out of shared configurations.

## License

MIT (license file will be finalized during the packaging task).
