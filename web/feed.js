export const FEED_CAP = 200;
export const RIBBON_CAP = 120;

const counts = new Intl.NumberFormat("en-US");

export function formatCount(value) {
  return counts.format(value ?? 0);
}

export function statusClass(status) {
  if (!Number.isInteger(status) || status < 100 || status > 599) {
    return "aborted";
  }
  return `${Math.floor(status / 100)}xx`;
}

export function isInjected(event) {
  return (event.faults?.length ?? 0) > 0;
}

export function tickKind(event) {
  if (isInjected(event)) {
    return "injected";
  }
  const kind = statusClass(event.status);
  if (kind === "5xx") {
    return "error";
  }
  return kind === "aborted" ? "aborted" : "forwarded";
}

export function matchesFilter(event, filter) {
  if (filter.injectedOnly && !isInjected(event)) {
    return false;
  }
  if (filter.status && statusClass(event.status) !== filter.status) {
    return false;
  }
  const text = filter.text.trim().toLowerCase();
  if (!text) {
    return true;
  }
  return [event.path, event.rule ?? "", event.method].some((value) => value.toLowerCase().includes(text));
}

export function prependCapped(list, item, cap) {
  list.unshift(item);
  if (list.length > cap) {
    list.length = cap;
  }
  return list;
}

export function percent(fraction) {
  return `${Number((fraction * 100).toFixed(1))}%`;
}

export function faultRate(stats) {
  if (!stats.published) {
    return "0%";
  }
  return percent(stats.faulted / stats.published);
}

export function formatBytes(bytes) {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${Number((bytes / 1024).toFixed(1))} KB`;
  }
  return `${Number((bytes / (1024 * 1024)).toFixed(1))} MB`;
}

export function formatTime(timestamp) {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const pad = (value, length = 2) => String(value).padStart(length, "0");
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)}`;
}

export function ruleFaults(rule) {
  const labels = [];
  const { latency, status, hang, truncate } = rule;
  if (latency?.dist === "fixed") {
    const value = latency.value ?? "0s";
    labels.push(latency.jitter ? `latency ${value} ±${latency.jitter}` : `latency ${value}`);
  }
  if (latency?.dist === "lognormal") {
    labels.push(`latency p50 ${latency.p50} p99 ${latency.p99}`);
  }
  if (status) {
    labels.push(`${status.code} · ${percent(status.probability)}`);
  }
  if (hang) {
    labels.push(`hang · ${percent(hang.probability)}`);
  }
  if (truncate) {
    labels.push(`truncate ${percent(truncate.at)} · ${percent(truncate.probability)}`);
  }
  return labels;
}
