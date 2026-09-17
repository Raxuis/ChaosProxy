# Task 14 — Final README

- Status: `completed`
- Approval: granted on 2026-09-17
- Depends on: task 13

## Goal

Write the final English README in the prescribed product-first order with copyable examples.

## Checklist

- [x] Preserve the exact product sentence and add the demo GIF placement.
- [x] Put the three-line npm quickstart first.
- [x] Add the factual comparison, complete fault catalog, and config reference.
- [x] Document profiles, reproducibility, integrations, contributing, and MIT licensing.
- [x] Validate every command and code sample.
- [x] Clean-environment `npx`, Homebrew, and `go install` checks moved to [task 16](task-16-release.md).
- [x] Receive explicit user approval for phase 4.

## Acceptance evidence

- The README opens with the title, the exact product sentence, a placeholder
  comment for the demo GIF recorded in task 15, and a three-line `npx` quickstart.
- It adds a comparison with Toxiproxy, MSW, mitmproxy, and browser DevTools
  limited to documented behavior, an install section (npm, Homebrew cask, `go
  install`, release archives), a flag table with defaults, a fault catalog with
  fields, execution order and client-visible effects, and a configuration
  reference for top-level fields, rules, route expressions, and scenarios.
- Every YAML block parsed and passed `config.Validate` in a temporary test, every
  `X-Chaos` directive parsed, `status=503; hang` was rejected, and every row of
  the route expression table matched as documented; the temporary tests were removed.
- Against a local API, the real binary confirmed: the target path prefix
  (`/users` reached `/v1/users`), the injected JSON body and `Retry-After`, the
  fixed latency with jitter, `cors: off` stripping `Access-Control-*`, the `curl`
  rule toggle, `X-Chaos-Applied`, every control endpoint, SSE, the reset `204`,
  the quoted validation errors and seed log line, `--list-profiles`, and a
  `--report` run where an invalid `X-Chaos` header stopped `--exit-on-error`
  with `stopped_by: error` and exit 1.
- All relative links and anchors resolve. `go install` resolves the module path
  from `Raxuis/ChaosProxy`, but the remote `main` is 27 commits behind.
- Not verifiable before publication: `npx chaosproxy`, `brew install --cask
  raxuis/tap/chaosproxy`, and `go install ...@latest` on the released code. The
  clean-environment `npx` check stays open until the first release.

## Log

| Date | Note |
|---|---|
| 2026-09-17 | Final README written and verified against the binary; npx check waits for the first release; approval requested. |
| 2026-09-17 | User approved task 14; installation checks that need a release moved to task 16. |
