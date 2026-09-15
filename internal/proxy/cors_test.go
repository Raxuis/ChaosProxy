package proxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestReflectCORSOriginPolicy(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	tests := []struct {
		name    string
		origins []string
		origin  string
		allowed bool
	}{
		{name: "default localhost", origin: "http://localhost:3001", allowed: true},
		{name: "default localhost subdomain", origin: "http://app.localhost:3001", allowed: true},
		{name: "default IPv4 loopback", origin: "http://127.0.0.1:5173", allowed: true},
		{name: "default IPv6 loopback", origin: "http://[::1]:5173", allowed: true},
		{name: "default remote", origin: "https://evil.example", allowed: false},
		{name: "default lookalike", origin: "http://localhost.evil.example", allowed: false},
		{name: "default opaque origin", origin: "null", allowed: false},
		{name: "listed", origins: []string{"https://app.test"}, origin: "https://app.test", allowed: true},
		{name: "list replaces loopback default", origins: []string{"https://app.test"}, origin: "http://localhost:3001", allowed: false},
		{name: "wildcard", origins: []string{"*"}, origin: "https://evil.example", allowed: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()
			newCORSHandler(t, upstream.URL, test.origins).ServeHTTP(response, request)

			got := response.Header().Get("Access-Control-Allow-Origin")
			if (test.allowed && got != test.origin) || (!test.allowed && got != "") {
				t.Fatalf("Access-Control-Allow-Origin = %q, want allowed=%t", got, test.allowed)
			}
			if !strings.Contains(strings.Join(response.Header().Values("Vary"), ","), "Origin") {
				t.Fatalf("Vary = %q, want Origin", response.Header().Values("Vary"))
			}
		})
	}
}

func TestReflectCORSForwardsPreflightFromUnallowedOrigin(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		upstreamRequests.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	request := httptest.NewRequest(http.MethodOptions, "http://proxy.test/orders", nil)
	request.Header.Set("Origin", "https://evil.example")
	request.Header.Set("Access-Control-Request-Method", "POST")
	response := httptest.NewRecorder()
	newCORSHandler(t, upstream.URL, nil).ServeHTTP(response, request)

	if upstreamRequests.Load() != 1 {
		t.Fatalf("upstream requests = %d, want preflight forwarded", upstreamRequests.Load())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want none", got)
	}
}

func newCORSHandler(t *testing.T, target string, origins []string) http.Handler {
	t.Helper()

	handler, err := proxy.NewHandler(&config.Config{
		Target:      target,
		CORS:        config.CORSReflect,
		CORSOrigins: origins,
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	return handler
}

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

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
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

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
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

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
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
