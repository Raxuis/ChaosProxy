package proxy

import (
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
)

func (h *Handler) finishRequest(
	r *http.Request,
	metrics *requestMetrics,
	writer *statusResponseWriter,
	started time.Time,
) {
	logRequest(h.logger, r, metrics, writer, started)
	if h.publisher == nil {
		return
	}
	h.publisher.Publish(events.Event{
		Method:            r.Method,
		Path:              r.URL.RequestURI(),
		Rule:              metrics.rule,
		Faults:            append([]string(nil), metrics.faults...),
		Details:           append([]string(nil), metrics.details...),
		InjectedLatencyMs: metrics.injectedLatency.Milliseconds(),
		UpstreamLatencyMs: metrics.upstreamLatency.Milliseconds(),
		Status:            writer.status,
		Bytes:             writer.bytes,
		DurationMs:        time.Since(started).Milliseconds(),
		Error:             eventError(metrics),
	})
}
