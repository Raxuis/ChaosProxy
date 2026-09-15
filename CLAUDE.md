# Chaos Proxy

## Code style

- Keep comments to a minimum. Do not explain what the code does; prefer clear names.
- Only exported identifiers get a doc comment, kept to one short line.
- Comment only non-obvious reasoning that the code cannot express.

## Workflow

- Work one task at a time and stop after each task for explicit user validation.
- Commit only after validation.
- Before handing a task back, run `gofmt -w .`, `go vet ./...` and `go test -race -count=1 ./...`.
- `.githooks/pre-push` rejects pushes with unformatted Go files; enable it with `git config core.hooksPath .githooks`.
