package proxy

import (
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
)

func (handler *Handler) finishRequest(
	request *http.Request,
	metrics *requestMetrics,
	writer *statusResponseWriter,
	started time.Time,
) {
	logRequest(handler.logger, request, metrics, writer, started)
	if handler.publisher == nil {
		return
	}
	handler.publisher.Publish(events.Event{
		Method:            request.Method,
		Path:              request.URL.RequestURI(),
		Rule:              metrics.rule,
		Faults:            append([]string(nil), metrics.faults...),
		InjectedLatencyMs: metrics.injectedLatency.Milliseconds(),
		UpstreamLatencyMs: metrics.upstreamLatency.Milliseconds(),
		Status:            writer.status,
		Bytes:             writer.bytes,
	})
}
