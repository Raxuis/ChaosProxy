package proxy_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerPassesThroughRequests(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		_ = json.NewEncoder(writer).Encode(map[string]string{
			"method": request.Method,
			"uri":    request.URL.RequestURI(),
			"body":   string(body),
		})
	}))
	defer upstream.Close()
	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, nil))
	defer proxyServer.Close()

	request, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/orders?expand=user", strings.NewReader(`{"id":42}`))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	var got map[string]string
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["method"] != http.MethodPost || got["uri"] != "/orders?expand=user" || got["body"] != `{"id":42}` {
		t.Fatalf("upstream request = %#v, want unchanged POST", got)
	}
}

func TestHandlerAppliesLatencyFault(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	const delay = 30 * time.Millisecond
	handler := newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
		Name:    "slow",
		Match:   "GET /api",
		Enabled: true,
		Latency: &config.LatencyConfig{Dist: "fixed", Value: delay},
	}})

	started := time.Now()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil))
	if elapsed := time.Since(started); elapsed < delay {
		t.Fatalf("elapsed = %s, want at least %s", elapsed, delay)
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestHandlerAppliesStatusFaultBeforeUpstream(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamRequests.Add(1)
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSReflect, []config.Rule{{
		Name:    "unavailable",
		Match:   "POST /orders",
		Enabled: true,
		Status:  &config.StatusConfig{Code: 503, Probability: 1, RetryAfter: 2},
	}})
	request := httptest.NewRequest(http.MethodPost, "http://proxy.test/orders", nil)
	request.Header.Set("Origin", "http://localhost:3001")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable || upstreamRequests.Load() != 0 {
		t.Fatalf("status/upstream requests = %d/%d, want 503/0", response.Code, upstreamRequests.Load())
	}
	if response.Header().Get("Retry-After") != "2" {
		t.Errorf("Retry-After = %q, want 2", response.Header().Get("Retry-After"))
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3001" {
		t.Errorf("synthetic response did not reflect Origin")
	}
	if !strings.Contains(response.Body.String(), `"rule":"unavailable"`) {
		t.Errorf("body = %q, want applied rule", response.Body.String())
	}
}

func TestHandlerHangStopsOnRequestCancellation(t *testing.T) {
	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamRequests.Add(1)
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
		Name:    "hang",
		Match:   "GET /profile",
		Enabled: true,
		Hang:    &config.HangConfig{Probability: 1},
	}})
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/profile", nil)
	ctx, cancel := context.WithCancel(request.Context())
	request = request.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), request)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hang did not stop after request cancellation")
	}
	if upstreamRequests.Load() != 0 {
		t.Fatalf("upstream requests = %d, want 0", upstreamRequests.Load())
	}
}

func TestHandlerTruncationAbortsConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		upstream   http.HandlerFunc
		wantLength int64
	}{
		{
			name: "declared length",
			upstream: func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Length", "10")
				_, _ = io.WriteString(writer, "0123456789")
			},
			wantLength: 10,
		},
		{
			name: "chunked",
			upstream: func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(writer, "01234")
				_ = http.NewResponseController(writer).Flush()
				_, _ = io.WriteString(writer, "56789")
			},
			wantLength: -1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			upstream := httptest.NewServer(test.upstream)
			defer upstream.Close()
			proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
				Name:     "broken",
				Match:    "GET /payload",
				Enabled:  true,
				Truncate: &config.TruncateConfig{Probability: 1, At: 0.5},
			}}))
			defer proxyServer.Close()

			response, err := http.Get(proxyServer.URL + "/payload")
			if err != nil {
				t.Fatalf("send request: %v", err)
			}
			defer response.Body.Close()
			if response.ContentLength != test.wantLength {
				t.Errorf("ContentLength = %d, want %d", response.ContentLength, test.wantLength)
			}
			body, err := io.ReadAll(response.Body)
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("read error = %v, want io.ErrUnexpectedEOF", err)
			}
			if string(body) != "01234" {
				t.Fatalf("body = %q, want truncated first half", body)
			}
		})
	}
}

func TestHandlerFlushesStreamingResponses(t *testing.T) {
	t.Parallel()

	releaseUpstream := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, "data: first\n\n")
		_ = http.NewResponseController(writer).Flush()
		<-releaseUpstream
		_, _ = io.WriteString(writer, "data: second\n\n")
	}))
	defer upstream.Close()
	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, nil))
	defer proxyServer.Close()

	responseResult := make(chan *http.Response, 1)
	errorResult := make(chan error, 1)
	go func() {
		response, err := http.Get(proxyServer.URL)
		if err != nil {
			errorResult <- err
			return
		}
		responseResult <- response
	}()

	var response *http.Response
	select {
	case response = <-responseResult:
	case err := <-errorResult:
		close(releaseUpstream)
		t.Fatalf("send request: %v", err)
	case <-time.After(time.Second):
		close(releaseUpstream)
		t.Fatal("response headers were buffered")
	}
	defer response.Body.Close()

	firstLine, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil {
		close(releaseUpstream)
		t.Fatalf("read first streamed line: %v", err)
	}
	close(releaseUpstream)
	if firstLine != "data: first\n" {
		t.Fatalf("first line = %q, want first event", firstLine)
	}
}

func TestHandlerLogsRuleFaultAndLatencies(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	}))
	defer upstream.Close()

	var logs bytes.Buffer
	handler, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "slow",
			Match:   "GET /api",
			Enabled: true,
			Latency: &config.LatencyConfig{Dist: "fixed", Value: time.Millisecond},
		}},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil))

	for _, field := range []string{
		`rule="slow"`, `faults="latency"`, "injected_latency_ms=1", "upstream_latency_ms=", "upstream_status=201", "status=201", `error=""`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Errorf("log %q does not contain %q", logs.String(), field)
		}
	}
}

func TestHandlerReturnsBrowserVisibleBadGateway(t *testing.T) {
	t.Parallel()

	unavailable := httptest.NewServer(http.NotFoundHandler())
	target := unavailable.URL
	unavailable.Close()
	handler := newTestHandler(t, target, config.CORSReflect, nil)
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	request.Header.Set("Origin", "http://localhost:3001")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3001" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want reflected origin", got)
	}
	if !strings.Contains(response.Body.String(), "upstream unavailable") {
		t.Fatalf("body = %q, want explicit upstream error", response.Body.String())
	}
}

func TestNewHandlerRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := proxy.NewHandler(&config.Config{Target: "://invalid"}, log.New(io.Discard, "", 0)); err == nil {
		t.Fatal("NewHandler() error = nil for invalid target")
	}
	if _, err := proxy.NewHandler(&config.Config{Target: "http://localhost"}, nil); err == nil {
		t.Fatal("NewHandler() error = nil for nil logger")
	}
}

func newTestHandler(t *testing.T, target string, cors config.CORSMode, configuredRules []config.Rule) http.Handler {
	t.Helper()

	handler, err := proxy.NewHandler(&config.Config{
		Target: target,
		Seed:   42,
		CORS:   cors,
		Rules:  configuredRules,
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	return handler
}
