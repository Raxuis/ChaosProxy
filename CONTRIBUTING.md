# Contributing to Chaos Proxy

Bug reports, fixes and focused features are welcome.

## Before you start

Open an issue before a large change so the approach can be agreed on first.
Small fixes can go straight to a pull request. Planned work and its status are
tracked in [PROGRESS.md](PROGRESS.md).

## Development setup

Requirements: Go 1.25 or newer, Git, and Node.js 20 or newer to test the
dashboard logic. The dashboard itself has no build step and no dependencies.

```sh
git clone https://github.com/Raxuis/ChaosProxy.git
cd ChaosProxy
git config core.hooksPath .githooks
go test ./...
```

The `pre-push` hook rejects pushes that contain unformatted Go files.

## Checks

CI runs these checks on every pull request. Run them locally before pushing:

```sh
gofmt -l .
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
go test -race -count=1 ./...
node --test web/feed.test.js
```

## Code guidelines

- Prefer the standard library, and discuss any new dependency in an issue first.
- Follow idiomatic Go naming: short receivers, `w` and `r` in HTTP handlers.
- Keep comments rare: one-line doc comments on exported identifiers, and
  explanations only for reasoning the code cannot express.
- Every behavior change comes with a test. Timing-sensitive tests must pass
  repeatedly with `-race`.
- Configuration errors belong in `config.Validate`, which reports YAML line numbers.

## Commits and pull requests

- Use [Conventional Commits](https://www.conventionalcommits.org), for example
  `fix(proxy): keep rule toggles across reloads`.
- Keep each pull request focused on one change and describe how you tested it.
- Update the README when a flag, configuration field or endpoint changes.

## License

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE).
