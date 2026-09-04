package proxy_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerForwardsRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method string
		body   string
	}{
		{method: http.MethodGet},
		{method: http.MethodPost, body: `{"order":42}`},
	}

	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			t.Parallel()

			upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read upstream body: %v", err)
				}
				response := map[string]string{
					"method": request.Method,
					"uri":    request.URL.RequestURI(),
					"body":   string(body),
				}
				writer.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(writer).Encode(response); err != nil {
					t.Errorf("encode upstream response: %v", err)
				}
			}))
			defer upstream.Close()

			proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, 0))
			defer proxyServer.Close()

			request, err := http.NewRequest(
				test.method,
				proxyServer.URL+"/api/orders?expand=user",
				strings.NewReader(test.body),
			)
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
			if got["method"] != test.method {
				t.Errorf("method = %q, want %q", got["method"], test.method)
			}
			if got["uri"] != "/api/orders?expand=user" {
				t.Errorf("URI = %q, want original URI", got["uri"])
			}
			if got["body"] != test.body {
				t.Errorf("body = %q, want %q", got["body"], test.body)
			}
		})
	}
}

func TestHandlerAppliesDelay(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	const delay = 40 * time.Millisecond
	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, delay))
	defer proxyServer.Close()

	started := time.Now()
	response, err := http.Get(proxyServer.URL)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	response.Body.Close()

	if elapsed := time.Since(started); elapsed < delay {
		t.Errorf("elapsed time = %s, want at least %s", elapsed, delay)
	}
}

func TestHandlerStopsBeforeUpstreamWhenCanceled(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamRequests.Add(1)
	}))
	defer upstream.Close()

	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(ctx)

	newTestHandler(t, upstream.URL, time.Second).ServeHTTP(httptest.NewRecorder(), request)
	if got := upstreamRequests.Load(); got != 0 {
		t.Fatalf("upstream requests = %d, want 0", got)
	}
}

func TestHandlerReturnsJSONBadGateway(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	upstreamURL := upstream.URL
	upstream.Close()

	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	response := httptest.NewRecorder()
	newTestHandler(t, upstreamURL, 0).ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadGateway)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", got)
	}
	if got := response.Body.String(); got != "{\"error\":\"upstream unavailable\"}\n" {
		t.Errorf("body = %q, want explicit upstream error", got)
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

	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, 0))
	defer proxyServer.Close()

	type result struct {
		response *http.Response
		err      error
	}
	responseResult := make(chan result, 1)
	go func() {
		response, err := http.Get(proxyServer.URL)
		responseResult <- result{response: response, err: err}
	}()

	var response *http.Response
	select {
	case got := <-responseResult:
		if got.err != nil {
			close(releaseUpstream)
			t.Fatalf("send request: %v", got.err)
		}
		response = got.response
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
	if firstLine != "data: first\n" {
		t.Errorf("first line = %q, want first event", firstLine)
	}
	close(releaseUpstream)
}

func TestHandlerLogsRequestOutcome(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	}))
	defer upstream.Close()

	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}
	handler := proxy.NewHandler(target, 0, logger)
	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "http://proxy.test/orders", nil),
	)

	for _, field := range []string{"method=POST", `path="/orders"`, "upstream_status=201", "status=201", "duration="} {
		if !strings.Contains(logs.String(), field) {
			t.Errorf("log %q does not contain %q", logs.String(), field)
		}
	}
}

func newTestHandler(t *testing.T, upstreamURL string, delay time.Duration) http.Handler {
	t.Helper()

	target, err := url.Parse(upstreamURL)
	if err != nil {
		t.Fatalf("parse upstream URL: %v", err)
	}

	return proxy.NewHandler(target, delay, log.New(io.Discard, "", 0))
}
