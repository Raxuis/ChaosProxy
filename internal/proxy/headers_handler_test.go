package proxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerModifiesResponseHeaders(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("ETag", `"12345"`)
		w.Header().Set("Server", "UpstreamServer/1.0")
		w.Header().Set("X-Existing", "original-value")
		_, _ = io.WriteString(w, "hello world")
	}))
	defer upstream.Close()

	var logs lockedBuffer
	handler, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "patch-headers",
			Match:   "GET /api/test",
			Enabled: true,
			Headers: &config.HeadersConfig{
				Probability: 1,
				Set: map[string]string{
					"X-Custom-Header": "custom-value",
					"X-Existing":      "overridden-value",
				},
				Remove: []string{"ETag", "Server"},
			},
		}},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	response, err := http.Get(proxyServer.URL + "/api/test")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "hello world" {
		t.Errorf("body = %q, want %q", string(body), "hello world")
	}

	if val := response.Header.Get("X-Custom-Header"); val != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", val, "custom-value")
	}
	if val := response.Header.Get("X-Existing"); val != "overridden-value" {
		t.Errorf("X-Existing = %q, want %q", val, "overridden-value")
	}
	if _, present := response.Header["Etag"]; present {
		t.Errorf("ETag header was not removed: %v", response.Header["Etag"])
	}
	if _, present := response.Header["Server"]; present {
		t.Errorf("Server header was not removed: %v", response.Header["Server"])
	}
	if val := response.Header.Get("Content-Type"); !strings.HasPrefix(val, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", val)
	}

	deadline := time.Now().Add(time.Second)
	for !strings.Contains(logs.String(), `faults="headers"`) {
		if time.Now().After(deadline) {
			t.Fatalf("log %q does not record the headers fault", logs.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHandlerHeadersProbabilityZeroDoesNotModify(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"original-etag"`)
		w.Header().Set("X-Existing", "untouched")
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()

	var logs lockedBuffer
	handler, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "zero-prob-headers",
			Match:   "GET /api/test",
			Enabled: true,
			Headers: &config.HeadersConfig{
				Probability: 0,
				Set: map[string]string{
					"X-Existing": "changed",
				},
				Remove: []string{"ETag"},
			},
		}},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	response, err := http.Get(proxyServer.URL + "/api/test")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	if val := response.Header.Get("ETag"); val != `"original-etag"` {
		t.Errorf("ETag = %q, want untouched", val)
	}
	if val := response.Header.Get("X-Existing"); val != "untouched" {
		t.Errorf("X-Existing = %q, want untouched", val)
	}
}
