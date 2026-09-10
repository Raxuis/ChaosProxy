package control_test

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/control"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestControlPlaneReadsConfigAndTogglesRule(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	bus := events.NewBus()
	runtime := newRuntime(t, upstream.URL, bus)
	handler, err := control.NewHandler(bus, runtime)
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	configResponse := serveControl(handler, http.MethodGet, "/api/config", "")
	if configResponse.Code != http.StatusOK {
		t.Fatalf("GET /api/config status = %d, want 200", configResponse.Code)
	}
	var active config.Config
	if err := json.NewDecoder(configResponse.Body).Decode(&active); err != nil {
		t.Fatalf("decode config response: %v", err)
	}
	if len(active.Rules) != 1 || !active.Rules[0].Enabled {
		t.Fatalf("active rules = %+v, want enabled injected rule", active.Rules)
	}

	toggleResponse := serveControl(handler, http.MethodPut, "/api/rules/injected", `{"enabled":false}`)
	if toggleResponse.Code != http.StatusOK {
		t.Fatalf("PUT rule status = %d body=%s, want 200", toggleResponse.Code, toggleResponse.Body.String())
	}
	if got := serveData(runtime); got != http.StatusNoContent {
		t.Fatalf("data status after disabling rule = %d, want upstream 204", got)
	}
	if runtime.CurrentConfig().Rules[0].Enabled {
		t.Fatal("runtime config still reports rule enabled")
	}

	missingResponse := serveControl(handler, http.MethodPut, "/api/rules/missing", `{"enabled":true}`)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing rule status = %d, want 404", missingResponse.Code)
	}
	badResponse := serveControl(handler, http.MethodPut, "/api/rules/injected", `{"enabled":false,"extra":true}`)
	if badResponse.Code != http.StatusBadRequest {
		t.Fatalf("unknown JSON field status = %d, want 400", badResponse.Code)
	}
}

func TestControlPlaneHealthAndReset(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	runtime := newRuntime(t, "http://localhost:9000", bus)
	handler, err := control.NewHandler(bus, runtime)
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	health := serveControl(handler, http.MethodGet, "/healthz", "")
	if health.Code != http.StatusOK || health.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q, want 200 ok", health.Code, health.Body.String())
	}
	bus.Publish(events.Event{Path: "/before-reset"})
	reset := serveControl(handler, http.MethodPost, "/api/reset", "")
	if reset.Code != http.StatusNoContent {
		t.Fatalf("reset status = %d, want 204", reset.Code)
	}
	if stats := bus.Stats(); stats.Published != 0 || stats.History != 0 {
		t.Fatalf("bus stats after reset = %+v", stats)
	}
}

func TestControlPlaneRejectsInvalidDependencies(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	runtime := newRuntime(t, "http://localhost:9000", bus)
	if _, err := control.NewHandler(nil, runtime); err == nil {
		t.Fatal("NewHandler() error = nil for nil bus")
	}
	if _, err := control.NewHandler(bus, nil); err == nil {
		t.Fatal("NewHandler() error = nil for nil runtime")
	}
}

func newRuntime(t *testing.T, target string, bus *events.Bus) *proxy.Handler {
	t.Helper()
	runtime, err := proxy.NewHandler(&config.Config{
		Target: target,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "injected",
			Match:   "GET /api",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 1},
		}},
	}, log.New(io.Discard, "", 0), proxy.WithEventPublisher(bus))
	if err != nil {
		t.Fatalf("proxy.NewHandler() unexpected error: %v", err)
	}
	return runtime
}

func serveControl(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, "http://control.test"+path, strings.NewReader(body)))
	return response
}

func serveData(handler http.Handler) int {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil))
	return response.Code
}
