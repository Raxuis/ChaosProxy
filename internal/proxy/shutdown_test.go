package proxy_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestBeginShutdownReleasesInjectedWaits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rule config.Rule
	}{
		{
			name: "hang",
			rule: config.Rule{Name: "hang", Match: "GET /wait", Enabled: true, Hang: &config.HangConfig{Probability: 1}},
		},
		{
			name: "latency",
			rule: config.Rule{
				Name:    "slow",
				Match:   "GET /wait",
				Enabled: true,
				Latency: &config.LatencyConfig{Dist: "fixed", Value: time.Hour},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var upstreamRequests atomic.Int64
			upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				upstreamRequests.Add(1)
			}))
			defer upstream.Close()

			proxyServer, entered := newShutdownServer(t, upstream.URL, test.rule)
			defer proxyServer.Close()

			responses := make(chan *http.Response, 1)
			go func() {
				defer close(responses)
				response, err := http.Get(proxyServer.URL + "/wait")
				if err != nil {
					t.Errorf("send request: %v", err)
					return
				}
				responses <- response
			}()
			waitForSignal(t, entered, "request did not reach the proxy")

			shutdownContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := proxyServer.Config.Shutdown(shutdownContext); err != nil {
				t.Fatalf("Shutdown() error = %v, want injected %s released", err, test.name)
			}

			response, ok := <-responses
			if !ok {
				return
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusServiceUnavailable {
				t.Errorf("status = %d, want 503", response.StatusCode)
			}
			if !response.Close {
				t.Error("response does not close the connection")
			}
			if upstreamRequests.Load() != 0 {
				t.Errorf("upstream requests = %d, want 0", upstreamRequests.Load())
			}
		})
	}
}

func TestBeginShutdownLetsForwardedRequestsDrain(t *testing.T) {
	t.Parallel()

	upstreamEntered := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(upstreamEntered)
		time.Sleep(100 * time.Millisecond)
		_, _ = io.WriteString(writer, "drained")
	}))
	defer upstream.Close()

	proxyServer, _ := newShutdownServer(t, upstream.URL, config.Rule{
		Name:    "instant",
		Match:   "GET /slow-upstream",
		Enabled: true,
		Latency: &config.LatencyConfig{Dist: "fixed"},
	})
	defer proxyServer.Close()

	responses := make(chan string, 1)
	go func() {
		defer close(responses)
		response, err := http.Get(proxyServer.URL + "/slow-upstream")
		if err != nil {
			t.Errorf("send request: %v", err)
			return
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Errorf("read response: %v", err)
			return
		}
		responses <- string(body)
	}()
	waitForSignal(t, upstreamEntered, "request did not reach the upstream")

	shutdownContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := proxyServer.Config.Shutdown(shutdownContext); err != nil {
		t.Fatalf("Shutdown() unexpected error: %v", err)
	}
	if body := <-responses; body != "drained" {
		t.Fatalf("body = %q, want forwarded response to drain", body)
	}
}

func newShutdownServer(t *testing.T, target string, rule config.Rule) (*httptest.Server, <-chan struct{}) {
	t.Helper()

	handler, err := proxy.NewHandler(&config.Config{
		Target: target,
		CORS:   config.CORSPassthrough,
		Rules:  []config.Rule{rule},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	entered := make(chan struct{})
	var enteredOnce sync.Once
	proxyServer := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		enteredOnce.Do(func() { close(entered) })
		handler.ServeHTTP(writer, request)
	}))
	proxyServer.Config.RegisterOnShutdown(handler.BeginShutdown)
	proxyServer.Start()
	return proxyServer, entered
}

func waitForSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}
