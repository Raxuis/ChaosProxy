package control_test

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/control"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestScenarioEndpoints(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	runtime, err := proxy.NewHandler(&config.Config{
		Target: "http://localhost:9000",
		CORS:   config.CORSPassthrough,
		Scenarios: []config.Scenario{{
			Name:        "checkout",
			Match:       "POST /checkout",
			Enabled:     true,
			OnExhausted: config.ExhaustPassthrough,
			Steps:       []string{"status=503"},
		}},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("proxy.NewHandler() unexpected error: %v", err)
	}
	handler, err := control.NewHandler(bus, runtime)
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	runtime.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "http://proxy.test/checkout", nil))

	states := decodeScenarios(t, serveControl(handler, http.MethodGet, "/api/scenarios", ""))
	if len(states) != 1 || states[0].Served != 1 || !states[0].Exhausted || states[0].NextStep != nil {
		t.Fatalf("scenarios = %+v, want checkout served once and exhausted", states)
	}

	if reset := serveControl(handler, http.MethodPost, "/api/scenarios/checkout/reset", ""); reset.Code != http.StatusNoContent {
		t.Fatalf("reset status = %d, want 204", reset.Code)
	}
	states = decodeScenarios(t, serveControl(handler, http.MethodGet, "/api/scenarios", ""))
	if states[0].Served != 0 || states[0].NextStep == nil || *states[0].NextStep != 0 {
		t.Fatalf("scenario after reset = %+v, want step 0 next", states[0])
	}

	if missing := serveControl(handler, http.MethodPost, "/api/scenarios/missing/reset", ""); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown scenario reset status = %d, want 404", missing.Code)
	}
}

func decodeScenarios(t *testing.T, response *httptest.ResponseRecorder) []proxy.ScenarioState {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/scenarios status = %d, want 200", response.Code)
	}
	var states []proxy.ScenarioState
	if err := json.NewDecoder(response.Body).Decode(&states); err != nil {
		t.Fatalf("decode scenarios: %v", err)
	}
	return states
}
