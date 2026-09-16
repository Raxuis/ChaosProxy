import assert from "node:assert/strict";
import { test } from "node:test";

import {
  faultRate,
  formatBytes,
  formatCount,
  formatTime,
  matchesFilter,
  prependCapped,
  ruleFaults,
  statusClass,
  tickKind,
} from "./feed.js";

const noFilter = { text: "", status: "", injectedOnly: false };

test("statusClass groups statuses and treats missing responses as aborted", () => {
  assert.equal(statusClass(204), "2xx");
  assert.equal(statusClass(503), "5xx");
  assert.equal(statusClass(0), "aborted");
  assert.equal(statusClass(undefined), "aborted");
  assert.equal(statusClass(700), "aborted");
});

test("tickKind separates injected faults from real upstream errors", () => {
  assert.equal(tickKind({ status: 503, faults: ["status"] }), "injected");
  assert.equal(tickKind({ status: 502 }), "error");
  assert.equal(tickKind({ status: 0 }), "aborted");
  assert.equal(tickKind({ status: 200, faults: [] }), "forwarded");
});

test("matchesFilter combines text, status and injected filters", () => {
  const injected = { method: "POST", path: "/api/orders", rule: "unavailable", status: 503, faults: ["status"] };
  const forwarded = { method: "GET", path: "/static/app.js", status: 200 };

  assert.equal(matchesFilter(forwarded, noFilter), true);
  assert.equal(matchesFilter(injected, { ...noFilter, text: "  ORDERS " }), true);
  assert.equal(matchesFilter(injected, { ...noFilter, text: "unavail" }), true);
  assert.equal(matchesFilter(forwarded, { ...noFilter, text: "post" }), false);
  assert.equal(matchesFilter(forwarded, { ...noFilter, status: "5xx" }), false);
  assert.equal(matchesFilter(forwarded, { ...noFilter, injectedOnly: true }), false);
  assert.equal(matchesFilter(injected, { text: "orders", status: "5xx", injectedOnly: true }), true);
});

test("prependCapped keeps the newest items first within the cap", () => {
  const list = [2, 1];
  prependCapped(list, 3, 2);
  assert.deepEqual(list, [3, 2]);
});

test("formatting helpers produce compact labels", () => {
  assert.equal(formatCount(1204), "1,204");
  assert.equal(faultRate({ published: 0, faulted: 0 }), "0%");
  assert.equal(faultRate({ published: 1204, faulted: 37 }), "3.1%");
  assert.equal(faultRate({ published: 8, faulted: 4 }), "50%");
  assert.equal(formatBytes(512), "512 B");
  assert.equal(formatBytes(2048), "2 KB");
  assert.equal(formatBytes(1536 * 1024), "1.5 MB");
  assert.match(formatTime("2026-09-15T14:02:11.042Z"), /^\d{2}:\d{2}:\d{2}\.042$/);
  assert.equal(formatTime("not a date"), "");
});

test("ruleFaults describes every configured fault", () => {
  assert.deepEqual(
    ruleFaults({
      latency: { dist: "fixed", value: "800ms", jitter: "200ms" },
      status: { code: 503, probability: 0.3 },
      hang: { probability: 0.1 },
      truncate: { probability: 0.125, at: 0.5 },
    }),
    ["latency 800ms ±200ms", "503 · 30%", "hang · 10%", "truncate 50% · 12.5%"],
  );
  assert.deepEqual(ruleFaults({ latency: { dist: "lognormal", p50: "400ms", p99: "3s" } }), ["latency p50 400ms p99 3s"]);
  assert.deepEqual(
    ruleFaults({ reset: { probability: 0.2 }, bandwidth: { bytes_per_second: 32768 } }),
    ["reset · 20%", "bandwidth 32 KB/s"],
  );
  assert.deepEqual(
    ruleFaults({ mutate: { probability: 0.5, operations: [{ op: "nullify", path: "user.email" }, { op: "drop", path: "id" }] } }),
    ["mutate 2 ops · 50%"],
  );
});
