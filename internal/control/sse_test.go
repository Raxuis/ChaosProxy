package control

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestEventStreamSendsHistoryLiveEventsAndHeartbeats(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	bus.Publish(events.Event{Path: "/history-one"})
	bus.Publish(events.Event{Path: "/history-two"})
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	runtime, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "unavailable",
			Match:   "POST /orders",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 1},
		}},
	}, log.New(io.Discard, "", 0), proxy.WithEventPublisher(bus))
	if err != nil {
		t.Fatalf("proxy.NewHandler() unexpected error: %v", err)
	}
	handler, err := NewHandler(bus, runtime)
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	handler.heartbeat = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "http://control.test/api/events", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()
	waitForSubscribers(t, bus, 1)
	dataResponse := httptest.NewRecorder()
	runtime.ServeHTTP(dataResponse, httptest.NewRequest(http.MethodPost, "http://proxy.test/orders", nil))
	if dataResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("data-plane status = %d, want 503", dataResponse.Code)
	}
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not return after request cancellation")
	}
	waitForSubscribers(t, bus, 0)

	body := response.Body.String()
	first := strings.Index(body, "id: 1")
	second := strings.Index(body, "id: 2")
	live := strings.Index(body, "id: 3")
	if first < 0 || second <= first || live <= second {
		t.Fatalf("SSE event order is incorrect:\n%s", body)
	}
	for _, expected := range []string{
		`"path":"/history-one"`,
		`"path":"/history-two"`,
		`"method":"POST"`,
		`"path":"/orders"`,
		`"rule":"unavailable"`,
		`"faults":["status"]`,
		`"status":503`,
		": heartbeat",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("SSE body does not contain %q:\n%s", expected, body)
		}
	}
}

func waitForSubscribers(t *testing.T, bus *events.Bus, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if bus.Stats().Subscribers == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("subscriber count = %d, want %d", bus.Stats().Subscribers, want)
}
