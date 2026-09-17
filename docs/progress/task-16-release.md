# Task 16 — Release and publication

- Status: `in_progress`
- Approval: not requested
- Depends on: every other task

## Goal

Deploy and publish Chaos Proxy once the project is finished. Every step that
pushes, publishes, or touches a third-party service needs explicit user approval.

## Checklist

- [x] Confirm every other task in PROGRESS.md is completed.
- [x] Confirm the npm name `chaosproxy` and the GitHub names are still available.
- [x] Push `main` and make `Raxuis/ChaosProxy` public.
- [ ] Create `Raxuis/homebrew-tap` and the `HOMEBREW_TAP_TOKEN` secret allowed to push to it.
- [ ] Configure npm trusted publishing for `chaosproxy`, after an initial manual publish if npm requires it.
- [ ] Tag the first version and check the release workflow: GitHub release archives, `checksums.txt`, Homebrew cask, npm package with provenance.
- [ ] Test `npx chaosproxy`, `brew install --cask raxuis/tap/chaosproxy`, and `go install github.com/Raxuis/chaosproxy/cmd/chaosproxy@latest` on clean environments and record evidence.
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
