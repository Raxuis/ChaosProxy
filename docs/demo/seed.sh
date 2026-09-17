#!/usr/bin/env bash
set -euo pipefail

api=${API_URL:-http://localhost:9000/api}

register() {
  curl -fsS -X POST "$api/users" -H 'Content-Type: application/json' \
    -d "{\"user\":{\"username\":\"$1\",\"email\":\"$1@example.com\",\"password\":\"password123\"}}" |
    node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>console.log(JSON.parse(s).user.token))'
}

publish() {
  curl -fsS -X POST "$api/articles" -H 'Content-Type: application/json' -H "Authorization: Token $1" \
    -d "{\"article\":{\"title\":\"$2\",\"description\":\"$3\",\"body\":\"$3\",\"tagList\":[\"$4\",\"$5\"]}}" >/dev/null
}

grace=$(register grace)
ada=$(register adalovelace)

publish "$grace" "Loading states nobody tests" "Spinners, skeletons and what happens when the API never answers" frontend ux
publish "$grace" "Chaos in CI" "Reproducible failures make flaky tests boring" testing ci
publish "$ada" "Designing retries that back off" "Why exponential backoff beats hammering a failing API" http resilience
publish "$ada" "Null is a valid JSON value" "Your types say string, the payload says null" typescript api
