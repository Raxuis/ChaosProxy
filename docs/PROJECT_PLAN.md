# Chaos Proxy project plan

## Product statement

A HTTP-aware chaos proxy for frontend developers — configure latency, errors, truncation and payload mutations per route, in code, reproducibly, in CI.

## Architecture

```text
browser -> data plane :7070 -> real upstream API
                 |
                 +-> events -> control plane :7071
                                REST config, SSE, embedded web UI
```

Core decisions:

- The data and control planes use separate ports.
- The first matching route rule wins.
- Seeded randomness supports replayable sequential runs.
- Probabilistic mode targets development; scenario mode targets CI.
- Configuration hot reload uses an atomic config pointer.
- The implementation uses the Go standard library first.

## Target layout

```text
cmd/chaosproxy/        CLI and wiring
internal/config/       typed configuration, validation, watcher
internal/rules/        compiled route matching
internal/faults/       fault interface and implementations
internal/proxy/        reverse proxy and request pipeline
internal/events/       in-memory event bus and ring buffer
internal/control/      REST API, SSE, and embedded UI
internal/scenario/     scripted sequences
web/                   dependency-free embedded UI
examples/              ready-to-use profiles and integrations
```

## Delivery order

1. Phase 0: validate proxying, latency, CORS behavior, and shutdown.
2. Phase 1: add YAML rules, core faults, the pipeline, and hot reload.
3. Phase 2: add observability, the embedded UI, reset, and throttling.
4. Phase 3: add deterministic CI behavior, scenarios, mutation, and integrations.
5. Phase 4: package and document the tool.
6. Launch: produce the demonstration and publish with empirical evidence.