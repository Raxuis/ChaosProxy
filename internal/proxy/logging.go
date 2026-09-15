package proxy

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type requestMetrics struct {
	rule            string
	faults          []string
	injectedLatency time.Duration
	upstreamStarted time.Time
	upstreamLatency time.Duration
	upstreamStatus  int
	pipelineError   error
	proxyError      error
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (writer *statusResponseWriter) WriteHeader(status int) {
	if writer.status != 0 {
		return
	}
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *statusResponseWriter) Write(body []byte) (int, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	written, err := writer.ResponseWriter.Write(body)
	writer.bytes += int64(written)
	return written, err
}

// Unwrap preserves optional net/http capabilities through ResponseController.
func (writer *statusResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func logRequest(
	logger *log.Logger,
	request *http.Request,
	metrics *requestMetrics,
	writer *statusResponseWriter,
	started time.Time,
) {
	faultNames := strings.Join(metrics.faults, ",")
	logger.Printf(
		"method=%s path=%q rule=%q faults=%q injected_latency_ms=%d upstream_latency_ms=%d upstream_status=%d status=%d bytes=%d duration=%s error=%q",
		request.Method,
		request.URL.RequestURI(),
		metrics.rule,
		faultNames,
		metrics.injectedLatency.Milliseconds(),
		metrics.upstreamLatency.Milliseconds(),
		metrics.upstreamStatus,
		writer.status,
		writer.bytes,
		time.Since(started).Round(time.Microsecond),
		requestError(metrics),
	)
}

func requestError(metrics *requestMetrics) string {
	if metrics.pipelineError != nil {
		return metrics.pipelineError.Error()
	}
	if metrics.proxyError != nil {
		return metrics.proxyError.Error()
	}
	return ""
}
