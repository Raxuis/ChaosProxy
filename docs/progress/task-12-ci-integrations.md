# Task 12 — CI integrations and reports

- Status: `completed`
- Approval: granted on 2026-09-17
- Depends on: task 11

## Goal

Make Chaos Proxy easy to run and assert against in automated test suites.

## Checklist

- [x] Add exit-on-error, max-request, and JSON report CLI behavior.
- [x] Include totals, per-rule fault counts, scenario use, and latency percentiles.
- [x] Provide a typed Playwright fixture and stable retry example.
- [x] Provide a Next.js integration snippet.
- [x] Provide a runnable GitHub Actions example.
- [x] Verify copied examples and deterministic same-seed behavior.
- [x] Run `go test ./...` and receive explicit user approval for phase 3.

## Acceptance evidence

- `--report PATH` (or `-`) writes a JSON report at shutdown with start and finish
  times, `stopped_by` (`signal`, `max-requests`, `error`, `server-error`), target,
  seed, totals (requests, injected, errors, status classes), per-rule fault counts,
  scenario states, p50/p90/p95/p99/max for total, upstream, and injected latency,
  and the first 100 errors.
- `--max-requests N` shuts down gracefully after N requests. `--exit-on-error`
  shuts down on the first request that fails inside the proxy and exits with
  status 1. Injected faults, shutdown rejections, and client cancellations never
  count as errors. Events now carry `duration_ms` and `error`.
- Request aggregation moved to `events.Tally`, shared by the dashboard bus and the
  report recorder.
- Two binary runs with `--seed 7 --max-requests 12` produced the same status
  sequence and identical report totals, rules, and scenarios, then exited 0 with
  `stopped_by: max-requests`. With the upstream down, `--exit-on-error` stopped
  after the first 502, exited 1, and reported the connection error.
- `examples/playwright/chaos.ts`, `checkout.spec.ts`, and `playwright.config.ts`,
  plus `examples/nextjs/next.config.ts` and `api.ts`, type-check with strict
  TypeScript 7.0.2, `@playwright/test` 1.63.0, and Next.js 16.3.5. Two Playwright
  tests using the fixture passed against the binary: scenario sequence and reset,
  automatic reset between tests, rule toggle, stats, and an `X-Chaos` override.
- `examples/github-actions/e2e.yml` and the repository CI pass `actionlint`.
- Not executed: `checkout.spec.ts` and the Next.js files need a real application,
  and the GitHub Actions example needs the module to be public for `go install`.
- `go test -race`, `go vet`, `staticcheck`, and the dashboard tests pass.

## Log

| Date | Note |
|---|---|
| 2026-09-16 | CI flags, run report, and integration examples implemented and verified; approval for task 12 and phase 3 requested. |
| 2026-09-17 | User approved task 12 and phase 3. |
