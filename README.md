A HTTP-aware chaos proxy for frontend developers — configure latency, errors, truncation and payload mutations per route, in code, reproducibly, in CI.

# Chaos Proxy

Chaos Proxy will sit between a frontend application and its real HTTP API, injecting reproducible failures on a per-route basis.

## Project status

The repository is being built incrementally. There is no usable proxy binary yet.

See [PROGRESS.md](PROGRESS.md) for the roadmap, current task, and validation gates.

## Planned local development

Requirements:

- Go 1.22 or newer
- Git

Run the phase 0 spike against a local upstream API:

```sh
go run ./cmd/chaosproxy --target http://localhost:3001 --delay 2s
```

The proxy listens on `http://localhost:7070` by default. Point the frontend's API base URL there, while the proxy forwards requests to the real upstream.

Available phase 0 flags:

```text
--target URL      upstream base URL (required)
--port PORT       data-plane listen port (default 7070)
--delay DURATION  delay before each upstream request (default 0)
```

Press Ctrl+C to stop the server gracefully.

## License

MIT (license file will be finalized during the packaging task).
