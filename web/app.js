import {
  FEED_CAP,
  RIBBON_CAP,
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

const POLL_INTERVAL_MS = 2000;
const RETRY_DELAY_MS = 3000;
const ANIMATED_BATCH_LIMIT = 8;

const byId = (id) => document.getElementById(id);

const elements = {
  target: byId("target"),
  seed: byId("seed"),
  cors: byId("cors"),
  link: byId("link"),
  linkLabel: byId("link-label"),
  pause: byId("pause"),
  reset: byId("reset"),
  ribbon: byId("ribbon"),
  notice: byId("notice"),
  rules: byId("rules"),
  rulesEmpty: byId("rules-empty"),
  filters: byId("filters"),
  filterText: byId("filter-text"),
  filterStatus: byId("filter-status"),
  filterInjected: byId("filter-injected"),
  buffer: byId("buffer"),
  feed: byId("feed"),
  feedEmpty: byId("feed-empty"),
  requests: byId("count-requests"),
  injected: byId("count-injected"),
  rate: byId("count-rate"),
  serverErrors: byId("count-5xx"),
  aborted: byId("count-aborted"),
  dropped: byId("count-dropped"),
};

const state = {
  link: "connecting",
  paused: false,
  lastId: 0,
  events: [],
  pending: [],
  pendingCount: 0,
  incoming: [],
  frame: 0,
  filter: { text: "", status: "", injectedOnly: false },
  ruleRows: new Map(),
  ruleNames: null,
  busyRules: new Set(),
};

bindControls();
connect();
poll();
setInterval(poll, POLL_INTERVAL_MS);

function connect() {
  const source = new EventSource("api/events");
  source.addEventListener("open", () => {
    clearTraffic();
    setLink("live");
    poll();
  });
  source.addEventListener("request", (message) => {
    prependCapped(state.incoming, JSON.parse(message.data), FEED_CAP + RIBBON_CAP);
    if (!state.frame) {
      state.frame = requestAnimationFrame(flushIncoming);
    }
  });
  source.addEventListener("error", () => {
    setLink("reconnecting");
    if (source.readyState === EventSource.CLOSED) {
      setTimeout(connect, RETRY_DELAY_MS);
    }
  });
}

function flushIncoming() {
  state.frame = 0;
  const batch = state.incoming.reverse().filter((event) => event.id > state.lastId);
  state.incoming = [];
  if (batch.length === 0) {
    return;
  }
  state.lastId = batch[batch.length - 1].id;

  for (const event of batch) {
    addTick(event);
  }
  updateRibbonLabel();

  if (state.paused) {
    for (const event of batch) {
      prependCapped(state.pending, event, FEED_CAP);
    }
    state.pendingCount += batch.length;
    renderBuffer();
    return;
  }

  for (const event of batch) {
    prependCapped(state.events, event, FEED_CAP);
  }
  if (batch.length > ANIMATED_BATCH_LIMIT) {
    renderFeed();
    return;
  }
  for (const event of batch) {
    if (matchesFilter(event, state.filter)) {
      insertRow(event);
    }
  }
  renderFeedEmpty();
}

async function poll() {
  try {
    const [config, stats] = await Promise.all([getJSON("api/config"), getJSON("api/stats")]);
    renderMeta(config);
    renderCounters(stats);
    renderRules(config, stats);
  } catch {
    return;
  }
}

async function getJSON(path) {
  const response = await fetch(path, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await errorMessage(response));
  }
  return response.json();
}

async function errorMessage(response) {
  try {
    return (await response.json()).error ?? response.statusText;
  } catch {
    return response.statusText;
  }
}

function bindControls() {
  elements.pause.addEventListener("click", () => setPaused(!state.paused));
  elements.reset.addEventListener("click", resetRuntime);
  elements.filters.addEventListener("submit", (event) => event.preventDefault());
  elements.filterText.addEventListener("input", updateFilter);
  elements.filterStatus.addEventListener("change", updateFilter);
  elements.filterInjected.addEventListener("change", updateFilter);
  document.addEventListener("keydown", handleShortcut);
}

function handleShortcut(event) {
  if (event.metaKey || event.ctrlKey || event.altKey) {
    return;
  }
  if (event.target === elements.filterText && event.key === "Escape") {
    elements.filterText.value = "";
    updateFilter();
    return;
  }
  if (event.target instanceof Element && event.target.closest("input, select, textarea")) {
    return;
  }
  if (event.key === "/") {
    event.preventDefault();
    elements.filterText.focus();
  } else if (event.key === "p") {
    setPaused(!state.paused);
  }
}

function updateFilter() {
  state.filter = {
    text: elements.filterText.value,
    status: elements.filterStatus.value,
    injectedOnly: elements.filterInjected.checked,
  };
  renderFeed();
}

function setPaused(paused) {
  state.paused = paused;
  elements.pause.setAttribute("aria-pressed", String(paused));
  elements.pause.textContent = paused ? "Resume" : "Pause";
  if (!paused && state.pending.length > 0) {
    state.events = [...state.pending, ...state.events].slice(0, FEED_CAP);
    state.pending = [];
    state.pendingCount = 0;
    renderFeed();
  }
  renderBuffer();
  setLink(state.link);
}

function setLink(link) {
  state.link = link;
  const paused = link === "live" && state.paused;
  elements.link.dataset.state = paused ? "paused" : link;
  elements.linkLabel.textContent = paused ? "live, feed paused" : link;
}

async function resetRuntime() {
  try {
    const response = await fetch("api/reset", { method: "POST" });
    if (!response.ok) {
      throw new Error(await errorMessage(response));
    }
    clearTraffic();
    showNotice("");
    poll();
  } catch (error) {
    showNotice(`Could not reset the proxy: ${error.message}`);
  }
}

async function toggleRule(name) {
  const row = state.ruleRows.get(name);
  if (!row || state.busyRules.has(name)) {
    return;
  }
  const enabled = row.toggle.getAttribute("aria-checked") !== "true";
  state.busyRules.add(name);
  row.toggle.setAttribute("aria-busy", "true");
  setSwitch(row, enabled);
  try {
    const response = await fetch(`api/rules/${encodeURIComponent(name)}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    });
    if (!response.ok) {
      throw new Error(await errorMessage(response));
    }
    showNotice("");
  } catch (error) {
    setSwitch(row, !enabled);
    showNotice(`Could not ${enabled ? "enable" : "disable"} ${name}: ${error.message}`);
  } finally {
    state.busyRules.delete(name);
    row.toggle.removeAttribute("aria-busy");
    poll();
  }
}

function showNotice(message) {
  elements.notice.textContent = message;
  elements.notice.hidden = message === "";
}

function clearTraffic() {
  state.lastId = 0;
  state.events = [];
  state.pending = [];
  state.pendingCount = 0;
  state.incoming = [];
  elements.ribbon.replaceChildren();
  updateRibbonLabel();
  renderFeed();
  renderBuffer();
}

function renderMeta(config) {
  elements.target.textContent = config.target;
  elements.seed.textContent = String(config.seed);
  const origins = config.cors_origins?.length ?? 0;
  elements.cors.textContent = origins > 0 ? `${config.cors} (${origins} origins)` : config.cors;
}

function renderCounters(stats) {
  const statuses = stats.statuses ?? {};
  setCount(elements.requests, stats.published);
  setCount(elements.injected, stats.faulted);
  setCount(elements.serverErrors, statuses["5xx"] ?? 0);
  setCount(elements.aborted, statuses.aborted ?? 0);
  setCount(elements.dropped, stats.dropped);
  elements.rate.textContent = faultRate(stats);
  elements.rate.parentElement.dataset.zero = String(!stats.faulted);
}

function setCount(element, value) {
  element.textContent = formatCount(value);
  element.parentElement.dataset.zero = String(!value);
}

function renderRules(config, stats) {
  const rules = config.rules ?? [];
  elements.rulesEmpty.hidden = rules.length > 0;

  const names = rules.map((rule) => rule.name).join("\n");
  if (names !== state.ruleNames) {
    const focusedRule = document.activeElement?.dataset?.rule;
    state.ruleNames = names;
    state.ruleRows.clear();
    elements.rules.replaceChildren(...rules.map(buildRuleRow));
    if (focusedRule) {
      state.ruleRows.get(focusedRule)?.toggle.focus();
    }
  }

  for (const rule of rules) {
    const row = state.ruleRows.get(rule.name);
    const totals = stats.rules?.[rule.name];
    if (!state.busyRules.has(rule.name)) {
      setSwitch(row, rule.enabled);
    }
    row.match.textContent = rule.match;
    row.faults.replaceChildren(...ruleFaults(rule).map((label) => textElement("span", "tag", label)));
    row.matched.textContent = formatCount(totals?.matched);
    row.injected.textContent = formatCount(totals?.faulted);
    row.injected.classList.toggle("is-hot", (totals?.faulted ?? 0) > 0);
  }
}

function buildRuleRow(rule) {
  const tr = document.createElement("tr");
  const toggle = document.createElement("button");
  toggle.type = "button";
  toggle.className = "switch";
  toggle.dataset.rule = rule.name;
  toggle.setAttribute("role", "switch");
  toggle.setAttribute("aria-label", `Inject faults for ${rule.name}`);
  toggle.addEventListener("click", () => toggleRule(rule.name));

  const toggleCell = document.createElement("td");
  toggleCell.append(toggle);
  const info = textElement("td", "rule-info", "");
  const match = textElement("span", "rule-match", "");
  const faults = textElement("span", "rule-faults", "");
  info.append(textElement("span", "rule-name", rule.name), match, faults);
  const matched = textElement("td", "num", "0");
  const injected = textElement("td", "num injected-count", "0");
  tr.append(toggleCell, info, matched, injected);

  state.ruleRows.set(rule.name, { tr, toggle, match, faults, matched, injected });
  return tr;
}

function setSwitch(row, enabled) {
  row.toggle.setAttribute("aria-checked", String(enabled));
  row.tr.classList.toggle("is-off", !enabled);
}

function renderBuffer() {
  const count = state.pendingCount;
  elements.buffer.hidden = count === 0;
  elements.buffer.textContent = `${formatCount(count)} new ${count === 1 ? "request" : "requests"} held while paused`;
}

function renderFeed() {
  const rows = document.createDocumentFragment();
  for (const event of state.events) {
    if (matchesFilter(event, state.filter)) {
      rows.append(buildRow(event));
    }
  }
  elements.feed.replaceChildren(rows);
  renderFeedEmpty();
}

function insertRow(event) {
  const row = buildRow(event);
  row.classList.add("arrive");
  elements.feed.prepend(row);
  while (elements.feed.childElementCount > FEED_CAP) {
    elements.feed.lastElementChild.remove();
  }
}

function renderFeedEmpty() {
  const empty = elements.feed.childElementCount === 0;
  elements.feedEmpty.hidden = !empty;
  if (empty) {
    elements.feedEmpty.textContent =
      state.events.length === 0 ? "Waiting for requests on the data plane." : "No request matches the filters.";
  }
}

function buildRow(event) {
  const row = document.createElement("tr");
  const kind = statusClass(event.status);
  row.append(
    textElement("td", "time", formatTime(event.timestamp)),
    textElement("td", "method", event.method),
    textElement("td", "path", event.path),
    textElement("td", "rule", event.rule ?? ""),
  );

  const faults = textElement("td", "faults", "");
  for (const fault of event.faults ?? []) {
    faults.append(textElement("span", "badge", fault));
  }

  const latency = textElement("td", "latency num", "");
  if (event.injected_latency_ms > 0) {
    latency.append(textElement("span", "inj", `+${event.injected_latency_ms} `));
  }
  latency.append(`${event.upstream_latency_ms} ms`);
  latency.title = `injected ${event.injected_latency_ms} ms, upstream ${event.upstream_latency_ms} ms`;

  row.append(
    faults,
    latency,
    textElement("td", "bytes num", formatBytes(event.bytes)),
    textElement("td", `status num s-${kind}`, kind === "aborted" ? "abort" : String(event.status)),
  );
  return row;
}

function addTick(event) {
  elements.ribbon.append(textElement("span", `tick ${tickKind(event)}`, ""));
  while (elements.ribbon.childElementCount > RIBBON_CAP) {
    elements.ribbon.firstElementChild.remove();
  }
}

function updateRibbonLabel() {
  const ticks = elements.ribbon.children;
  if (ticks.length === 0) {
    elements.ribbon.setAttribute("aria-label", "No requests yet");
    return;
  }
  let injected = 0;
  let errors = 0;
  for (const tick of ticks) {
    injected += tick.classList.contains("injected") ? 1 : 0;
    errors += tick.classList.contains("error") ? 1 : 0;
  }
  elements.ribbon.setAttribute(
    "aria-label",
    `Last ${ticks.length} requests: ${injected} injected, ${errors} upstream server errors`,
  );
}

function textElement(tag, className, text) {
  const element = document.createElement(tag);
  element.className = className;
  element.textContent = text;
  return element;
}
