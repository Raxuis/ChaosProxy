# Task 13 — Packaging and distribution

- Status: `pending`
- Approval: not requested
- Depends on: task 12

## Goal

Ship binaries, an npm wrapper, a container image, and embedded failure profiles.

## Checklist

- [ ] Configure cross-platform GoReleaser archives, checksums, version injection, and Homebrew.
- [ ] Build an npm wrapper with platform download and checksum verification.
- [ ] Add tag-driven release and npm publication automation.
- [ ] Define, embed, and document five failure profiles, proposed for approval at task start.
- [ ] Verify packaging locally without publishing external artifacts.
- [ ] Receive explicit user approval before any real publication.

## Deferred on demand

- Multi-stage distroless container image: built only if users ask for it.
  [Issue #1](https://github.com/Raxuis/ChaosProxy/issues/1) collects that demand.

## Acceptance evidence

Not started.

## Log

| Date | Note |
|---|---|
| 2026-09-15 | Container image deferred until users request it; the five profiles were never specified and will be proposed at task start. |
