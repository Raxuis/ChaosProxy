package proxy_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerStallPausesBodyMidStream(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("a", 2048) + strings.Repeat("b", 2048)
	var upstreamServed atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamServed.Add(1)
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
		Name:    "mid-stall",
		Match:   "GET /data",
		Enabled: true,
		Stall: &config.StallConfig{
			Probability: 1,
			AfterBytes:  2048,
			Duration:    100 * time.Millisecond,
		},
	}}))
	defer proxyServer.Close()

	started := time.Now()
	resp, err := http.Get(proxyServer.URL + "/data")
	if err != nil {
		t.Fatalf("GET /data: %v", err)
	}
	defer resp.Body.Close()

	if resp.ContentLength != int64(len(payload)) {
		t.Errorf("ContentLength = %d, want %d", resp.ContentLength, len(payload))
	}

	firstChunk := make([]byte, 2048)
	n, err := io.ReadFull(resp.Body, firstChunk)
	if err != nil || n != 2048 {
		t.Fatalf("read first chunk = %d, %v; want 2048, nil", n, err)
	}
	firstChunkDuration := time.Since(started)
	if firstChunkDuration >= 80*time.Millisecond {
		t.Errorf("first chunk arrived after %v, want < 80ms before stall", firstChunkDuration)
	}

	stallStart := time.Now()
	secondChunk, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read second chunk: %v", err)
	}
	stallElapsed := time.Since(stallStart)

	if stallElapsed < 90*time.Millisecond {
		t.Errorf("stall elapsed = %v, want >= 90ms", stallElapsed)
	}
	if string(firstChunk)+string(secondChunk) != payload {
		t.Errorf("received payload does not match original payload")
	}
	if upstreamServed.Load() != 1 {
		t.Errorf("upstream served = %d, want 1", upstreamServed.Load())
	}
}

func TestHandlerStallStopsOnClientDisconnect(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("x", 4096)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()

	proxyServer := httptest.NewServer(newTestHandler(t, upstream.URL, config.CORSPassthrough, []config.Rule{{
		Name:    "disconnect-stall",
		Match:   "GET /stream",
		Enabled: true,
		Stall: &config.StallConfig{
			Probability: 1,
			AfterBytes:  512,
			Duration:    5 * time.Second,
		},
	}}))
	defer proxyServer.Close()

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, proxyServer.URL+"/stream", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do(req): %v", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 512)
	n, err := io.ReadFull(resp.Body, buf)
	if err != nil || n != 512 {
		t.Fatalf("read before stall = %d, %v; want 512, nil", n, err)
	}

	cancel()

	readStart := time.Now()
	_, err = resp.Body.Read(buf)
	readElapsed := time.Since(readStart)

	if readElapsed >= time.Second {
		t.Errorf("Read() after cancel took %v, want abort < 1s", readElapsed)
	}
	if err == nil {
		t.Error("expected error reading after cancellation, got nil")
	}
}

func TestHandlerStallShutdownDuringStall(t *testing.T) {
	t.Parallel()

	payload := strings.Repeat("s", 4096)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()

	handler, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "shutdown-stall",
			Match:   "GET /shutdown-stall",
			Enabled: true,
			Stall: &config.StallConfig{
				Probability: 1,
				AfterBytes:  256,
				Duration:    5 * time.Second,
			},
		}},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	resp, err := http.Get(proxyServer.URL + "/shutdown-stall")
	if err != nil {
		t.Fatalf("GET /shutdown-stall: %v", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 256)
	n, err := io.ReadFull(resp.Body, buf)
	if err != nil || n != 256 {
		t.Fatalf("read 256 bytes = %d, %v; want 256, nil", n, err)
	}

	shutdownDone := make(chan struct{})
	go func() {
		handler.BeginShutdown()
		close(shutdownDone)
	}()

	readStart := time.Now()
	_, readErr := io.ReadAll(resp.Body)
	readElapsed := time.Since(readStart)

	<-shutdownDone

	if readElapsed >= time.Second {
		t.Errorf("ReadAll after BeginShutdown took %v, want < 1s", readElapsed)
	}
	if readErr == nil {
		t.Error("ReadAll after BeginShutdown returned no error, want the stalled body cut short")
	}
}
