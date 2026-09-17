# Task 16 — Release and publication

- Status: `in_progress`
- Approval: not requested
- Depends on: every other task

## Goal

Deploy and publish Chaos Proxy once the project is finished. Every step that
pushes, publishes, or touches a third-party service needs explicit user approval.

## Checklist

- [x] Confirm every other task in PROGRESS.md is completed.
- [x] Confirm the npm and GitHub names are still available.
- [x] Push `main` and make `Raxuis/ChaosProxy` public.
- [x] Create `Raxuis/homebrew-tap` and the `HOMEBREW_TAP_TOKEN` secret allowed to push to it.
- [x] Configure npm trusted publishing for `@raxuis/chaosproxy`, after an initial manual publish if npm requires it.
- [ ] Tag the first version and check the release workflow: GitHub release archives, `checksums.txt`, Homebrew cask, npm package with provenance. (Tag, archives, checksums, and cask verified; npm pending.)
- [x] Test `npx @raxuis/chaosproxy`, `brew install --cask raxuis/tap/chaosproxy`, and `go install github.com/Raxuis/chaosproxy/cmd/chaosproxy@latest` on clean environments and record evidence. Windows archives are verified by checksum only.
- [ ] Offer the Conduit findings to the `vue-realworld-example-app` maintainers.
- [ ] Create the good-first-issues drafted in task 15.
- [ ] Publish the launch copy drafted in task 15, channel by channel.
- [ ] Receive final user approval.

## Deferred on demand

- Container image: [issue #1](https://github.com/Raxuis/ChaosProxy/issues/1).

## Acceptance evidence

- Preflight on 2026-09-17: tasks 00–15 and R1 completed with no open checkbox;
  `npm view chaosproxy` returned 404; `Raxuis/ChaosProxy` is private and owned;
  `Raxuis/homebrew-tap` does not exist yet; no repository secrets.
- History check before going public: `gitleaks git` found no leaks in 31
  commits, no key, `.env`, or credential file was ever committed, and the largest
  blob is the 200 KB demo GIF. The user chose to keep the commit email as is and
  to release `v0.1.0`.
- `release.yml` now upgrades npm to 11.5.1 or later, required by trusted
  publishing, and skips `npm publish` when the tag version already exists, so the
  job can be re-run after a first manual publish. `actionlint` passes.
- With approval, `main` was pushed (`b583ff1..dd5ccf0`) and CI passed: lint job
  and tests on ubuntu and macOS with Go 1.25.x and stable.
- The user made the repository public and set its description and nine topics
  (the auto mode classifier blocked Claude from changing visibility).

## Log

| Date | Note |
|---|---|
| 2026-09-17 | Created to group every deployment and publication step at the very end of the roadmap. |
| 2026-09-17 | Preflight done, release workflow made re-runnable, `main` pushed with approval. |
| 2026-09-17 | CI green on `main`; repository made public by the user with description and topics. |
| 2026-09-17 | Release workflow change pushed in `9a55a8f` (CI green); `Raxuis/homebrew-tap` created public and initialized with a README on `main`. |
| 2026-09-17 | User added a fine-grained `HOMEBREW_TAP_TOKEN`; `gh secret list` shows it. |
| 2026-09-17 | With approval, `v0.1.0` tagged on `9a55a8f` and pushed. Release run `35210498613`: `binaries` succeeded, `npm` failed as expected with `404 PUT chaosproxy` (no trusted publisher yet). |
| 2026-09-17 | Verified the public release: six archives match `checksums.txt`, the darwin/arm64 binary reports `0.1.0`, `Casks/chaosproxy.rb` in the tap carries per-platform checksums, and `go install github.com/Raxuis/chaosproxy/cmd/chaosproxy@latest` installed `0.1.0`. |
| 2026-09-17 | npm package `0.1.0` prepared from the tag outside the repository: tests pass, 5 files, 3.3 kB; installed from its tarball, it downloaded the real darwin/arm64 release asset, verified it, and ran `0.1.0`. |
| 2026-09-17 | npm rejected the manual publish of `chaosproxy` with `403 Package name too similar to existing package chaos-proxy`; nothing was published. `chaos-proxy` is an active Express-based chaos proxy CLI configured in YAML. The user chose `@raxuis/chaosproxy`; the package name, release workflow, READMEs, CONTRIBUTING, and launch drafts were updated, while the installed command stays `chaosproxy`. |
| 2026-09-17 | README comparison gained a `chaos-proxy` (npm) column limited to its documented features: Koa-based Node.js CLI and library, route middleware, every-nth failures, rate limiting, throttling, and custom middleware. |
| 2026-09-17 | The user published `@raxuis/chaosproxy@0.1.0` manually; the registry reports it as `latest` with repository `git+https://github.com/Raxuis/ChaosProxy.git`. The user created a GitHub Actions trusted publisher (`Raxuis/ChaosProxy`, `release.yml`, no environment, `npm publish` allowed) and pushed the rename in `a630421`. The `v0.1.0` npm job stays failed because its workflow predates the rename; automated publishing is first exercised by the next tag. |
| 2026-09-17 | Clean macOS check: with an empty `HOME` and npm cache, `npx -y @raxuis/chaosproxy --version` downloaded the darwin/arm64 archive and printed `0.1.0` in 3.1 s; `--profile outage` then answered `503` with `Retry-After: 30` in front of a local server. |
| 2026-09-17 | Homebrew 7.0.1 check: `brew install --cask raxuis/tap/chaosproxy` installed `0.1.0` (exit 0) but printed four `Calling postflight is deprecated! Use postflight_steps instead` warnings; GoReleaser emits that stanza from `hooks` (upstream issue goreleaser#6870, PR #6873 open). Without any hook the binary kept `com.apple.quarantine` and macOS Gatekeeper blocked it, so the hook is required. |
| 2026-09-17 | `.goreleaser.yaml` now emits `postflight_steps` with `run "/usr/bin/xattr"` on `{{staged_path}}` through `custom_block`; `goreleaser check` passes and the snapshot cask renders the token literally. The same block in the live `0.1.0` cask installed with no deprecation warning, no quarantine attribute, and a working binary, then uninstalled cleanly. The fix ships with the next tag; the published tap still has the old stanza. Homebrew was restored: tap, cask, trust entry, and binary removed. |
| 2026-09-17 | Clean Linux check in `node:24-slim` containers (Node 24.21.0): on linux/arm64 and linux/amd64, `npx -y @raxuis/chaosproxy` downloaded the matching archive, printed `0.1.0`, and `--profile outage` answered `503` with `Retry-After: 30`. No Windows machine was available. Docker was restored afterwards. |
