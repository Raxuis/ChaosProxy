package proxy_test

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerResetSendsTCPReset(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamRequests.Add(1)
	}))
	defer upstream.Close()
	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
		Name:    "reset",
		Match:   "GET /api",
		Enabled: true,
		Reset:   &config.ResetConfig{Probability: 1},
	}}))
	defer proxyServer.Close()

	response, err := http.Get(proxyServer.URL + "/api")
	if err == nil {
		response.Body.Close()
		t.Fatalf("GET status = %d, want a connection reset", response.StatusCode)
	}
	if !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("GET error = %v, want ECONNRESET", err)
	}
	if upstreamRequests.Load() != 0 {
		t.Fatalf("upstream requests = %d, want 0", upstreamRequests.Load())
	}
}

func TestHandlerResetAbortsHTTP2Stream(t *testing.T) {
	t.Parallel()

	var logs lockedBuffer
	handler, err := proxy.NewHandler(&config.Config{
		Target: "http://127.0.0.1:1",
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "reset",
			Match:   "GET /api",
			Enabled: true,
			Reset:   &config.ResetConfig{Probability: 1},
		}},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	proxyServer := httptest.NewUnstartedServer(handler)
	proxyServer.EnableHTTP2 = true
	proxyServer.StartTLS()
	defer proxyServer.Close()

	response, err := proxyServer.Client().Get(proxyServer.URL + "/api")
	if err == nil {
		response.Body.Close()
		t.Fatalf("GET status = %d over %s, want an aborted HTTP/2 stream", response.StatusCode, response.Proto)
	}

	deadline := time.Now().Add(time.Second)
	for !strings.Contains(logs.String(), "reset aborted the stream") {
		if time.Now().After(deadline) {
			t.Fatalf("log %q does not explain the HTTP/2 fallback", logs.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHandlerBandwidthDeliversBodyProgressively(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("x", 4096)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()
	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
		Name:      "slow",
		Match:     "GET /asset",
		Enabled:   true,
		Bandwidth: &config.BandwidthConfig{BytesPerSecond: 8192},
	}}))
	defer proxyServer.Close()

	started := time.Now()
	response, err := http.Get(proxyServer.URL + "/asset")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()
	if response.ContentLength != int64(len(payload)) {
		t.Errorf("ContentLength = %d, want %d", response.ContentLength, len(payload))
	}

	first := make([]byte, 1)
	if _, err := io.ReadFull(response.Body, first); err != nil {
		t.Fatalf("read first byte: %v", err)
	}
	firstByte := time.Since(started)
	rest, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	elapsed := time.Since(started)

	if len(rest)+1 != len(payload) {
		t.Fatalf("body length = %d, want %d", len(rest)+1, len(payload))
	}
	if firstByte > 300*time.Millisecond {
		t.Errorf("first byte after %s, want progressive delivery", firstByte)
	}
	if elapsed < 400*time.Millisecond {
		t.Errorf("4096 bytes at 8192 B/s arrived in %s, want about 500ms", elapsed)
	}
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
