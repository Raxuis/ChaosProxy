package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestReflectCORSHandlesPreflightWithoutUpstream(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamRequests.Add(1)
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSReflect, nil)
	request := httptest.NewRequest(http.MethodOptions, "http://proxy.test/orders", nil)
	request.Header.Set("Origin", "http://localhost:3001")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || upstreamRequests.Load() != 0 {
		t.Fatalf("status/upstream requests = %d/%d, want 204/0", response.Code, upstreamRequests.Load())
	}
	for header, want := range map[string]string{
		"Access-Control-Allow-Origin":      "http://localhost:3001",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "POST",
		"Access-Control-Allow-Headers":     "Content-Type, Authorization",
	} {
		if got := response.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestReflectCORSOverridesUpstreamOrigin(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "https://wrong.example")
		_, _ = io.WriteString(writer, "ok")
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSReflect, nil)
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	request.Header.Set("Origin", "http://localhost:3001")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3001" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want reflected origin", got)
	}
	if !strings.Contains(strings.Join(response.Header().Values("Vary"), ","), "Origin") {
		t.Fatalf("Vary = %q, want Origin", response.Header().Values("Vary"))
	}
}

func TestPassthroughCORSPreservesUpstream(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "https://upstream.example")
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSPassthrough, nil)
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	request.Header.Set("Origin", "http://localhost:3001")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://upstream.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want upstream value", got)
	}
}

func TestOffCORSRemovesUpstreamHeaders(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSOff, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil))

	if response.Header().Get("Access-Control-Allow-Origin") != "" || response.Header().Get("Access-Control-Expose-Headers") != "" {
		t.Fatalf("off mode retained CORS headers: %v", response.Header())
	}
}
