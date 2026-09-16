A HTTP-aware chaos proxy for frontend developers — configure latency, errors, truncation and payload mutations per route, in code, reproducibly, in CI.

# Chaos Proxy

Chaos Proxy will sit between a frontend application and its real HTTP API, injecting reproducible failures on a per-route basis.

## Project status

Chaos Proxy is under active development and not released yet. It already
supports route matching, latency, status, hang and truncation faults, hot
reload, reproducible seeded decisions, browser-safe CORS handling, and a control
plane with a live event stream. An embedded web UI, scenario mode and CI
integrations are planned.

See [PROGRESS.md](PROGRESS.md) for the roadmap and the status of each task.

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
--header-overrides  let clients force faults with the X-Chaos header
```

At least one of `--config` or `--target` is required.

Control-plane endpoints:

```text
GET  /
GET  /healthz
GET  /api/stats
GET  /api/events
GET  /api/config
PUT  /api/rules/{name}  body: {"enabled": false}
POST /api/reset
GET  /api/scenarios
POST /api/scenarios/{name}/reset
```

Open `http://localhost:7071/` for the dashboard. It shows the target and seed,
request totals, a full-width ribbon of recent requests, every rule with its toggle
and trigger counts, and a live request feed. Press `/` to filter the feed and
`p` to pause it. The dashboard works offline and loads nothing from the network.

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

## Connection resets and bandwidth

`reset` closes the client connection with a TCP RST before any response, so
clients see `ECONNRESET`. When the connection cannot be taken over, as with
HTTP/2, the proxy aborts the stream instead and logs why.

`bandwidth` limits how fast the upstream body reaches the client without
buffering it, so `Content-Length` and streaming stay intact:

```yaml
rules:
  - name: reset-checkout
    match: POST /api/checkout
    reset:
      probability: 0.2

  - name: slow-assets
    match: GET /static/**
    bandwidth:
      bytes_per_second: 32768
```

## Reproducibility

Each rule draws its decisions from the seed, the rule name, and the number of
requests that rule has matched. Unmatched requests and other rules never shift
a rule's sequence, and `POST /api/reset` restarts every sequence. Concurrent
requests to the same rule can still arrive in a different order between runs.

Without `seed` in the file or `--seed`, Chaos Proxy generates a random seed and
logs it at startup. Pass it back with `--seed` to replay the run.

## Scenarios

A scenario plays one step per matching request, in order, which makes CI runs
exact: the first checkout fails, the second is slow, the third succeeds.

```yaml
scenarios:
  - name: checkout-recovers
    match: POST /api/checkout
    on_exhausted: passthrough
    steps:
      - status=503
      - latency=2s; status=502
      - off
```

Steps use the same directives as the `X-Chaos` header below. `on_exhausted`
decides what happens after the last step: `passthrough` (default) forwards
requests without faults, `repeat` starts again from the first step, and `last`
keeps replaying the final step.

An enabled scenario takes precedence over rules for the requests it matches, and
an `X-Chaos` header takes precedence over both. Steps advance in the order
requests arrive, so run requests to the same scenario sequentially when the
order matters.

`GET /api/scenarios` reports each scenario's steps, how many requests it served,
the next step, and whether it is exhausted. `POST /api/scenarios/{name}/reset`
restarts one scenario, `POST /api/reset` restarts all of them, and a scenario
whose definition changes on reload starts over.

## Header overrides

Start the proxy with `--header-overrides` to let a client force faults for a
single request, which keeps end-to-end tests explicit:

```http
GET /api/orders
X-Chaos: latency=800ms; status=503
```

| Directive | Effect |
|---|---|
| `latency=800ms` | wait before handling the request |
| `status=503` | respond with this status without calling the upstream |
| `hang` | never respond |
| `reset` | reset the TCP connection |
| `truncate=0.5` | cut the response body at this fraction |
| `bandwidth=32768` | deliver the body at this many bytes per second |
| `off` | forward the request without any fault |

The header replaces rule matching for that request, never shifts rule decision
sequences, and is removed before the request reaches the upstream. `status`,
`hang` and `reset` cannot be combined. Responses carry `X-Chaos-Applied` with the
forced faults, `off`, or `disabled` when the proxy runs without
`--header-overrides`. An invalid header gets a `400` response explaining why.

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).
