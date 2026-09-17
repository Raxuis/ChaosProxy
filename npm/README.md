# chaosproxy

An HTTP-aware chaos proxy for frontend developers: configure latency, errors,
truncation and payload mutations per route, reproducibly, in CI.

```sh
npx chaosproxy --target http://localhost:9000 --profile flaky-api
```

Point your frontend at `http://localhost:7070` and open the dashboard on
`http://localhost:7071`.

This package downloads the prebuilt binary for your platform from the matching
[GitHub release](https://github.com/Raxuis/ChaosProxy/releases) on first run,
verifies its SHA-256 checksum, and caches it. Set `CHAOSPROXY_CACHE_DIR` to
change the cache location or `CHAOSPROXY_DOWNLOAD_BASE_URL` to use a mirror.

Documentation: https://github.com/Raxuis/ChaosProxy#readme
