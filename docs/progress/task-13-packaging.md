# Task 13 — Packaging and distribution

- Status: `completed`
- Approval: granted on 2026-09-17 for the packaging work; publication not yet approved
- Depends on: task 12

## Goal

Ship binaries, an npm wrapper, a container image, and embedded failure profiles.

## Checklist

- [x] Configure cross-platform GoReleaser archives, checksums, version injection, and Homebrew.
- [x] Build an npm wrapper with platform download and checksum verification.
- [x] Add tag-driven release and npm publication automation.
- [x] Define, embed, and document five failure profiles, proposed for approval at task start.
- [x] Verify packaging locally without publishing external artifacts.
- [x] Real publication moved to [task 16](task-16-release.md).

## Deferred on demand

- Multi-stage distroless container image: built only if users ask for it.
  [Issue #1](https://github.com/Raxuis/ChaosProxy/issues/1) collects that demand.

## Acceptance evidence

- The user approved five profiles on 2026-09-17: `slow-network`, `flaky-api`,
  `connection-drops`, `broken-payloads`, and `outage`. They are embedded YAML
  files in `internal/profiles`, selected with `--profile NAME --target URL`,
  listed with `--list-profiles`, documented in the README, and each compiles into
  a proxy handler in tests. `--profile` rejects `--config` and requires `--target`.
- `--version` prints the GoReleaser-injected version, falling back to Go build
  information for `go install`. Startup logs and run reports include it.
- `.goreleaser.yaml` builds `CGO_ENABLED=0` binaries for Linux, macOS, and Windows
  on amd64 and arm64, packages tar.gz and zip archives with `LICENSE` and
  `README.md`, writes SHA-256 `checksums.txt`, and generates a Homebrew cask for
  `Raxuis/homebrew-tap` that clears the macOS quarantine attribute.
  `goreleaser check` (v2.18.2) validates it.
- A local snapshot built all six archives; `shasum -a 256 -c` verified every
  checksum; the darwin/arm64 binary reported `0.0.0-SNAPSHOT-cc591d5` and listed
  the profiles; the generated cask carried per-platform checksums.
- The `chaosproxy` npm package downloads the matching release archive on first run
  without install scripts, verifies SHA-256, caches it per version, and forwards
  SIGINT and SIGTERM. Node tests cover asset names, checksum parsing, a tampered
  archive, and download plus caching against a local server.
- Packed and installed from its tarball against the snapshot served locally, the
  wrapper shipped five files, downloaded and verified the darwin/arm64 archive,
  reused the cache on the second run, and forwarded Ctrl+C so the proxy exited 0
  and wrote its report.
- `.github/workflows/release.yml` runs tests and GoReleaser on `v*` tags, then
  publishes the npm package with the tag version and provenance. The CI workflow
  now runs the wrapper tests and `goreleaser check`. `actionlint` passes.
- Nothing was published. Publishing needs the repository to be public, a
  `Raxuis/homebrew-tap` repository with a `HOMEBREW_TAP_TOKEN` secret, and npm
  trusted publishing for `chaosproxy`, which may require an initial manual publish.
- `go test -race`, `go vet`, `staticcheck`, and all Node tests pass.

## Log

| Date | Note |
|---|---|
| 2026-09-15 | Container image deferred until users request it; the five profiles were never specified and will be proposed at task start. |
| 2026-09-17 | Profiles, GoReleaser, npm wrapper, and release workflow implemented and verified locally without publishing; approval requested. |
| 2026-09-17 | User approved task 13; real publication still requires separate approval. |
