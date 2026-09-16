# Task 07 — Embedded web UI

- Status: `awaiting_validation`
- Approval: requested on 2026-09-16
- Depends on: task 06

## Goal

Add an offline-capable, dependency-free traffic dashboard suitable for a README demonstration.

## Checklist

- [x] Embed vanilla HTML, CSS, and JavaScript without a build step or CDN.
- [x] Show target, seed, counters, rules, toggles, and per-rule trigger totals.
- [x] Show a capped, newest-first live request feed with useful status/fault detail.
- [x] Add filters, pause, and automatic SSE reconnection feedback.
- [x] Verify keyboard usability, responsive layout, and dark default styling.
- [x] Test static serving and core UI behavior where practical.
- [ ] Receive explicit user visual approval.

## Acceptance evidence

- Package `web` embeds `index.html`, `style.css`, `app.js`, `feed.js`, and
  `favicon.svg` (31 KB). The control plane serves them at `/` with a strict
  Content-Security-Policy (`script-src 'self'`, no inline code, no framing).
- The event bus aggregates totals, faulted requests, status classes, and per-rule
  matched and injected counts, exposed at `GET /api/stats` and cleared by reset.
- The dashboard shows target, seed, CORS mode, six counters, a full-width ribbon of
  up to 600 recent requests (injected faults in amber, real 5xx in red), rules with
  accessible switches and trigger totals, and a feed capped at 200 rows.
- Browser checks against a live demo proxy: text, status, and injected filters;
  Escape clears the text filter; `/` focuses it; `p` pauses and the held-request
  counter increments; resume merges held requests; toggles reach the server and
  keep keyboard focus across refreshes; reset clears counters and ribbon; stopping
  the proxy shows "reconnecting" and restarting it returns to "live".
- A request path containing `<img onerror>` and `<script>` renders as text only.
- Batching uses a 100 ms timer instead of `requestAnimationFrame`, so hidden tabs
  keep the ribbon and paused counters accurate.
- Verified at 1268 px and 363 px wide with no horizontal overflow; the mobile feed
  switches to compact two-line rows. Focus rings are visible and animations stop
  under `prefers-reduced-motion`.
- `go test -race`, `go vet`, `staticcheck`, six `node --test` cases, and
  `actionlint` pass. CI runs the dashboard tests.

## Log

| Date | Note |
|---|---|
| 2026-09-16 | Direction chosen with the user: instrumented console, traffic ribbon, arrival motion; vanilla CSS kept over React, htmx, and Tailwind. |
| 2026-09-16 | Dashboard implemented and verified in the browser; ribbon widened after user feedback; approval requested. |
