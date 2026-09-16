package proxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

type capturingPublisher struct {
	mu     sync.Mutex
	events []events.Event
}

func (p *capturingPublisher) Publish(event events.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
}

func (p *capturingPublisher) last() events.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.events[len(p.events)-1]
}

func TestHandlerPublishesDurationAndProxyErrors(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	unavailable := httptest.NewServer(http.NotFoundHandler())
	unavailableURL := unavailable.URL
	unavailable.Close()

	newHandler := func(target string, publisher *capturingPublisher) http.Handler {
		handler, err := proxy.NewHandler(&config.Config{
			Target: target,
			CORS:   config.CORSPassthrough,
			Rules: []config.Rule{{
				Name:    "slow",
				Match:   "GET /slow",
				Enabled: true,
				Latency: &config.LatencyConfig{Dist: "fixed", Value: 20 * time.Millisecond},
			}},
		}, log.New(io.Discard, "", 0), proxy.WithEventPublisher(publisher))
		if err != nil {
			t.Fatalf("NewHandler() unexpected error: %v", err)
		}
		return handler
	}

	healthy := &capturingPublisher{}
	statusFor(newHandler(upstream.URL, healthy), "/slow")
	if event := healthy.last(); event.DurationMs < 20 || event.Error != "" {
		t.Fatalf("healthy event = %+v, want at least 20ms and no error", event)
	}

	broken := &capturingPublisher{}
	statusFor(newHandler(unavailableURL, broken), "/down")
	if event := broken.last(); event.Status != http.StatusBadGateway || event.Error == "" {
		t.Fatalf("unavailable upstream event = %+v, want 502 with an error", event)
	}
}
