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

Once the phase 0 spike is approved, the development command will be:

```sh
go run ./cmd/chaosproxy --target http://localhost:3001
```

## License

MIT (license file will be finalized during the packaging task).
