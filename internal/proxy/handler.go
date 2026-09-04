// Package proxy implements the HTTP data plane.
package proxy

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type requestMetrics struct {
	upstreamStatus int
	proxyError     error
}

type metricsContextKey struct{}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
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

	return w.ResponseWriter.Write(body)
}

// Unwrap lets net/http retain optional response capabilities such as flushing
// and connection hijacking when the writer is wrapped for status logging.
func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// NewHandler returns a streaming reverse proxy that applies delay before each
// upstream request. Target and logger must remain valid for the handler lifetime.
func NewHandler(target *url.URL, delay time.Duration, logger *log.Logger) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.SetXForwarded()
		},
		ModifyResponse: func(response *http.Response) error {
			metricsFromRequest(response.Request).upstreamStatus = response.StatusCode
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			metrics := metricsFromRequest(request)
			metrics.proxyError = err

			writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			writer.Header().Set("Cache-Control", "no-store")
			writer.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(writer, "{\"error\":\"upstream unavailable\"}\n")
		},
		FlushInterval: -1,
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		metrics := &requestMetrics{}
		request = request.WithContext(context.WithValue(request.Context(), metricsContextKey{}, metrics))
		statusWriter := &statusResponseWriter{ResponseWriter: writer}

		defer logRequest(logger, request, metrics, statusWriter, started)

		if !waitForDelay(request.Context(), delay) {
			return
		}

		proxy.ServeHTTP(statusWriter, request)
	})
}

func waitForDelay(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func logRequest(
	logger *log.Logger,
	request *http.Request,
	metrics *requestMetrics,
	writer *statusResponseWriter,
	started time.Time,
) {
	duration := time.Since(started).Round(time.Microsecond)
	if metrics.proxyError != nil {
		logger.Printf(
			"method=%s path=%q upstream_status=%d status=%d duration=%s error=%q",
			request.Method,
			request.URL.RequestURI(),
			metrics.upstreamStatus,
			writer.status,
			duration,
			metrics.proxyError,
		)
		return
	}

	logger.Printf(
		"method=%s path=%q upstream_status=%d status=%d duration=%s",
		request.Method,
		request.URL.RequestURI(),
		metrics.upstreamStatus,
		writer.status,
		duration,
	)
}

func metricsFromRequest(request *http.Request) *requestMetrics {
	metrics, ok := request.Context().Value(metricsContextKey{}).(*requestMetrics)
	if !ok {
		return &requestMetrics{}
	}

	return metrics
}
