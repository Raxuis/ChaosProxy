# Good first issue drafts

Drafts only. They become GitHub issues in task 16, after approval. Each one adds
a fault and follows the path of existing faults: a config type in
`internal/config/types.go`, validation with line numbers in `validate.go`, a
fault in `internal/faults`, registration in `registry.go`, dashboard labels in
`web/app.js`, and a README catalog row.

## Add a `headers` fault to add, override, or remove response headers

Frontends break when a header they rely on changes: a missing `Content-Type`, a
`Cache-Control` that suddenly caches, a removed `Location`.

```yaml
headers:
  probability: 0.5
  set:
    Cache-Control: max-age=3600
  remove:
    - Content-Type
```

- Apply to the upstream response in `After`, like `mutate`.
- Reject empty or invalid header names, and reject `Content-Length` and
  `Transfer-Encoding`, which the proxy manages.
- Tests: set, override, remove, probability 0 and 1, invalid names with lines.

## Add a `redirect` fault that answers with a 3xx

Expired sessions often surface as a redirect to a login page that a `fetch`
call follows silently.

```yaml
redirect:
  probability: 1
  code: 302
  location: /login
```

- Return a `ShortCircuit` from `Before`, like `status`.
- Accept only 301, 302, 303, 307, and 308, and a non-empty `location`.
- Also add a `redirect=302:/login` directive to `config.ParseFaults` so scenarios
  and `X-Chaos` can use it.
- Tests: status and `Location`, CORS headers on the response, validation errors.

## Add a `stall` fault that pauses the body mid-stream

Real networks stall after the first bytes; spinners that only cover the time to
first byte never show.

```yaml
stall:
  probability: 0.2
  after_bytes: 4096
  duration: 5s
```

- Wrap the response body in `After` like `bandwidth`, without buffering, and keep
  `Content-Length`.
- The wait must stop when the client disconnects or the proxy shuts down.
- Tests: bytes before and after the pause, cancellation during the stall,
  shutdown during the stall with `-race`.
