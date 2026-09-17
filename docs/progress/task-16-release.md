# Task 16 — Release and publication

- Status: `pending`
- Approval: not requested
- Depends on: every other task

## Goal

Deploy and publish Chaos Proxy once the project is finished. Every step that
pushes, publishes, or touches a third-party service needs explicit user approval.

## Checklist

- [ ] Confirm every other task in PROGRESS.md is completed.
- [ ] Confirm the npm name `chaosproxy` and the GitHub names are still available.
- [ ] Push `main` and make `Raxuis/ChaosProxy` public.
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

Not started.

## Log

| Date | Note |
|---|---|
| 2026-09-17 | Created to group every deployment and publication step at the very end of the roadmap. |
