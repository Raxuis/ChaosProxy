# Task 15 — Demonstration and launch preparation

- Status: `completed`
- Approval: granted on 2026-09-17
- Depends on: task 14

## Goal

Prepare evidence-led launch material locally. Nothing is published, pushed, or
created on a third-party service in this task; that happens in task 16.

## Checklist

- [x] Close the remaining manual checks from tasks 04 (browser sees the injected 503) and 05 (reload after an editor save).
- [x] Exercise `broken-payloads` against a suitable open-source frontend and document a real finding, closing the task 11 manual check.
- [x] Record the 15-second demonstration GIF and place it in the README.
- [x] Draft channel-specific launch copy grounded in the demonstration, kept as a local draft.
- [x] Draft three good-first-issue fault ideas, kept as local drafts.
- [x] Receive explicit user approval.

## Acceptance evidence

- Test bed: RealWorld Conduit, Vue 3 frontend `vue-realworld-example-app` at
  `f7e48c8` on Vite, Go/Gin API `golang-gin-realworld-example-app` on SQLite,
  seeded with two users and four articles, all behind the local binary. The API
  has no CORS support; `cors: reflect` made the frontend work through the proxy.
- Task 04 check: Chrome read an injected `503` and its JSON body cross-origin,
  and the home page degraded cleanly (empty tag sidebar, articles still shown).
  The same check found a product bug: `Retry-After` was invisible to browser
  code. Injected responses now add it to `Access-Control-Expose-Headers` when
  the origin is reflected; a test covers it, failing before the fix, and Chrome
  then read `Retry-After: 5`.
- Task 05 check: in-place writes, atomic rename saves (JetBrains style),
  `sed -i`, and Vim saves each reloaded within a second and changed the served
  `Retry-After`. An invalid status code was logged as
  `config reload rejected: line 8: rules[0].status.code must be between 100 and 599`
  while the previous rule kept answering `503`.
- Task 11 check and finding: one `null` element in `GET /api/articles` throws
  `TypeError: Cannot read properties of null (reading 'slug')` in
  `ArticleList.vue` and leaves "Loading articles..." forever with no error
  message. `--profile broken-payloads --seed 7` reproduces it on the first load.
  A `503` on the same endpoint also spins forever because `fetchArticles`
  swallows the error without resetting `isLoading`. A `null` author is handled.
- The demo exposed dashboard layout problems around 1100–1280 px: the status
  column was scrolled out of view, paths broke mid-word, and latency read as
  `+307 1 ms`. The feed now shows status right after time, a total `duration`
  highlighted when latency was injected (breakdown in the tooltip), hides bytes
  up to 1280 px, and the rules column is narrower. Dashboard tests pass.
- `docs/assets/demo.gif`: 14 seconds, 960 px, 231 KB, six captioned steps
  recorded with headless Chrome through Playwright. `docs/demo` holds the
  configuration, `seed.sh`, `record.mjs`, and `compose.sh`; `compose.sh`
  rebuilt the GIF byte for byte, and `seed.sh` was run against a fresh API. The
  README shows the GIF under the product sentence. `node_modules/` is ignored.
- `docs/launch/launch-copy.md` drafts Show HN, Reddit, X/Bluesky, and dev.to
  copy limited to the demonstrated findings. `docs/launch/good-first-issues.md`
  drafts `headers`, `redirect`, and `stall` faults with config, scope, and tests.
- Nothing was published, pushed, or reported upstream. Offering the findings to
  the Conduit maintainers is listed in task 16.

## Log

| Date | Note |
|---|---|
| 2026-09-17 | Scope limited to local preparation; publication, name checks, and external issues moved to task 16. |
| 2026-09-17 | Manual checks closed, Conduit finding, Retry-After and dashboard fixes, demo GIF, and drafts done; approval requested. |
| 2026-09-17 | User approved task 15. |
