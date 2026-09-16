package proxy

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

type requestMetrics struct {
	rule            string
	faults          []string
	details         []string
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

func (w *statusResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	written, err := w.ResponseWriter.Write(body)
	w.bytes += int64(written)
	return written, err
}

// Unwrap preserves optional net/http capabilities through ResponseController.
func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func logRequest(
	logger *log.Logger,
	r *http.Request,
	metrics *requestMetrics,
	writer *statusResponseWriter,
	started time.Time,
) {
	faultNames := strings.Join(metrics.faults, ",")
	logger.Printf(
		"method=%s path=%q rule=%q faults=%q injected_latency_ms=%d upstream_latency_ms=%d upstream_status=%d status=%d bytes=%d duration=%s details=%q error=%q",
		r.Method,
		r.URL.RequestURI(),
		metrics.rule,
		faultNames,
		metrics.injectedLatency.Milliseconds(),
		metrics.upstreamLatency.Milliseconds(),
		metrics.upstreamStatus,
		writer.status,
		writer.bytes,
		time.Since(started).Round(time.Microsecond),
		strings.Join(metrics.details, "; "),
		requestError(metrics),
	)
}

// Shutdown rejections and client cancellations are expected, so events do not report them.
func eventError(metrics *requestMetrics) string {
	if errors.Is(metrics.pipelineError, errShuttingDown) || errors.Is(metrics.proxyError, context.Canceled) {
		return ""
	}
	return requestError(metrics)
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
