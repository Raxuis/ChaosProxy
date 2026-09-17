# Launch copy drafts

Drafts only. Nothing here is posted before task 16, and each channel needs
explicit approval. Links assume the repository is public and `v0.1.0` is
released; replace them if the version differs.

## The finding every post is built on

RealWorld Conduit's Vue 3 frontend
([`vue-realworld-example-app` at `f7e48c8`](https://github.com/gothinkster/vue-realworld-example-app/tree/f7e48c8))
against its Go/Gin API, behind Chaos Proxy:

- One `null` element in `GET /api/articles` (`mutate` → `nullify articles.1`)
  throws
  `TypeError: Cannot read properties of null (reading 'slug')` in
  `ArticleList.vue`. The API answered `200`; the feed shows "Loading articles..."
  forever and never displays an error. `--profile broken-payloads --seed 7`,
  which nulls every article, reproduces it on the first page load.
- A `503` on the same endpoint ends the same way: `fetchArticles` swallows the
  error and never resets `isLoading`, so there is no message and no retry.
- A `503` on `GET /api/tags` is handled well: the sidebar stays empty and the
  feed still loads.

Keep every claim to what the demo shows. Do not call Conduit badly written: it
is a learning project, and most frontends share these gaps.

## Show HN

**Title:** Show HN: Chaos Proxy – per-route HTTP faults for frontend developers

**Text:**

I built Chaos Proxy because frontends are usually tested against an API that
answers quickly and correctly, while production sends 503s, slow responses, cut
connections, and JSON with unexpected nulls.

It is a single Go binary that sits between the frontend and the real API. Rules
match a method and a path glob and inject latency (fixed or lognormal), status
codes with Retry-After, hangs, TCP resets, truncated bodies, bandwidth limits,
or JSON mutations such as nulling `articles.*.author`. Decisions are seeded, so
a failing CI run can be replayed with `--seed`, and scenarios script exact
sequences like "fail twice, then succeed". A local dashboard shows every request live.

I tried it on RealWorld Conduit's Vue frontend: one null article inside a 200
response leaves the whole feed spinning forever with a TypeError and no error
message. The GIF in the README shows it.

Differences from tools I used before: Toxiproxy works at TCP level, so it cannot
target a route or a status code; MSW mocks responses instead of talking to the
real API; DevTools throttling is manual and per tab.

npx @raxuis/chaosproxy --target http://localhost:9000 --profile flaky-api

MIT, feedback on the fault catalog welcome: https://github.com/Raxuis/ChaosProxy

## Reddit (r/webdev, r/Frontend)

**Title:** I made a proxy that breaks your API on purpose, per route – it found a
stuck loading state in RealWorld Conduit in one reload

**Text:** Same facts as Show HN, shorter: the problem (happy-path API during
development), the Conduit finding with the GIF, the three-line quickstart, the
Playwright fixture for CI, and a direct question: which failure does your
frontend handle worst?

Check each subreddit's self-promotion rules before posting.

## Reddit (r/golang)

**Title:** Chaos Proxy: an HTTP fault-injection reverse proxy in Go (feedback welcome)

**Angle:** implementation rather than product. `httputil.ReverseProxy` with a
per-rule fault chain, atomic runtime snapshots with compare-and-swap for hot
reload, splitmix64-derived PCG streams per rule for reproducibility, TCP resets
through `SO_LINGER=0` on hijacked connections, and an embedded dashboard with
`go:embed` and a strict CSP. Ask for review of the reset and truncation paths.

## X / Bluesky thread

1. Your frontend is tested against an API that never fails. Chaos Proxy puts
   per-route faults between the two: latency, 503s, resets, truncated bodies,
   JSON mutations. Reproducible with a seed. [GIF]
2. On RealWorld Conduit, one null article in a 200 response leaves the feed on
   "Loading articles..." forever. No error, no retry.
3. `npx @raxuis/chaosproxy --target http://localhost:9000 --profile broken-payloads`
4. Built for CI: JSON run report, exit code, Playwright fixture, scenarios like
   "fail twice, then succeed". MIT: https://github.com/Raxuis/ChaosProxy

## dev.to article outline

**Title:** One null in a 200 response: testing the unhappy paths of a real frontend

1. Why development APIs hide failures.
2. Setting up Conduit behind Chaos Proxy in three commands.
3. Findings: null article, 503 on the feed, 503 on tags, each with its config.
4. Making it a CI test: scenario plus Playwright fixture plus `--exit-on-error`.
5. What a resilient fix looks like: filter or validate items, reset loading
   state in `finally`, show an error with a retry button.
