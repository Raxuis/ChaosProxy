package proxy_test

import (
	"errors"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestScenarioPlaysStepsBeforeRules(t *testing.T) {
	t.Parallel()

	handler := newScenarioProxy(t, newRecordingUpstream(t).URL, config.ExhaustPassthrough, "status=503", "status=502", "off")
	if got := statuses(handler, "/checkout", 5); !slices.Equal(got, []int{503, 502, 204, 204, 204}) {
		t.Fatalf("statuses = %v, want the scripted steps then passthrough without the background rule", got)
	}
}

func TestScenarioExhaustionModes(t *testing.T) {
	t.Parallel()

	target := newRecordingUpstream(t).URL
	tests := map[config.ExhaustMode][]int{
		config.ExhaustPassthrough: {204, 503, 204, 204},
		config.ExhaustRepeat:      {204, 503, 204, 503},
		config.ExhaustLast:        {204, 503, 503, 503},
	}
	for mode, want := range tests {
		handler := newScenarioProxy(t, target, mode, "off", "status=503")
		if got := statuses(handler, "/checkout", len(want)); !slices.Equal(got, want) {
			t.Errorf("%s statuses = %v, want %v", mode, got, want)
		}
	}
}

func TestScenarioStateAndReset(t *testing.T) {
	t.Parallel()

	handler := newScenarioProxy(t, newRecordingUpstream(t).URL, config.ExhaustPassthrough, "status=503", "off")
	statuses(handler, "/checkout", 1)
	state := handler.Scenarios()[0]
	if state.Served != 1 || state.Exhausted || state.NextStep == nil || *state.NextStep != 1 || !slices.Equal(state.Steps, []string{"status=503", "off"}) {
		t.Fatalf("state after one request = %+v, want step 1 next", state)
	}

	statuses(handler, "/checkout", 1)
	if state := handler.Scenarios()[0]; state.Served != 2 || !state.Exhausted || state.NextStep != nil {
		t.Fatalf("state after two requests = %+v, want exhausted", state)
	}

	if err := handler.ResetScenario("checkout"); err != nil {
		t.Fatalf("ResetScenario() unexpected error: %v", err)
	}
	if got := statuses(handler, "/checkout", 1); got[0] != http.StatusServiceUnavailable {
		t.Fatalf("status after scenario reset = %d, want the first step again", got[0])
	}
	handler.Reset()
	if state := handler.Scenarios()[0]; state.Served != 0 {
		t.Fatalf("served after Reset = %d, want 0", state.Served)
	}
	if err := handler.ResetScenario("missing"); !errors.Is(err, config.ErrScenarioNotFound) {
		t.Fatalf("ResetScenario(missing) error = %v, want ErrScenarioNotFound", err)
	}
}

func TestScenarioCountersAreConcurrencySafe(t *testing.T) {
	t.Parallel()

	handler := newScenarioProxy(t, newRecordingUpstream(t).URL, config.ExhaustRepeat, "status=500", "status=501", "status=502")
	counts := make(map[int]int)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range 300 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := statusFor(handler, "/checkout")
			mu.Lock()
			counts[status]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if counts[500] != 100 || counts[501] != 100 || counts[502] != 100 {
		t.Fatalf("status counts = %v, want 100 of each step", counts)
	}
	if served := handler.Scenarios()[0].Served; served != 300 {
		t.Fatalf("served = %d, want 300", served)
	}
}

func TestHeaderOverrideDoesNotAdvanceScenario(t *testing.T) {
	t.Parallel()

	handler := newScenarioProxy(t, newRecordingUpstream(t).URL, config.ExhaustPassthrough, "status=503")
	serveWithHeaders(handler, "/checkout", map[string]string{"X-Chaos": "off"})
	if served := handler.Scenarios()[0].Served; served != 0 {
		t.Fatalf("served after an overridden request = %d, want 0", served)
	}
}

func TestUpdateRestartsOnlyChangedScenarios(t *testing.T) {
	t.Parallel()

	target := newRecordingUpstream(t).URL
	scenarios := func(stepsForB string) *config.Config {
		return &config.Config{
			Target: target,
			CORS:   config.CORSPassthrough,
			Scenarios: []config.Scenario{
				{Name: "a", Match: "GET /a", Enabled: true, OnExhausted: config.ExhaustPassthrough, Steps: []string{"status=500"}},
				{Name: "b", Match: "GET /b", Enabled: true, OnExhausted: config.ExhaustPassthrough, Steps: []string{stepsForB}},
			},
		}
	}
	handler, err := proxy.NewHandler(scenarios("status=501"), log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	statuses(handler, "/a", 1)
	statuses(handler, "/b", 1)

	if err := handler.Update(scenarios("status=501")); err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if served := servedByName(handler); served["a"] != 1 || served["b"] != 1 {
		t.Fatalf("served after an identical reload = %v, want both kept", served)
	}

	if err := handler.Update(scenarios("status=502")); err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	if served := servedByName(handler); served["a"] != 1 || served["b"] != 0 {
		t.Fatalf("served after changing b = %v, want a kept and b restarted", served)
	}
}

func newScenarioProxy(t *testing.T, target string, mode config.ExhaustMode, steps ...string) *proxy.Handler {
	t.Helper()

	handler, err := proxy.NewHandler(&config.Config{
		Target: target,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "background",
			Match:   "GET /checkout",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 500, Probability: 1},
		}},
		Scenarios: []config.Scenario{{
			Name:        "checkout",
			Match:       "GET /checkout",
			Enabled:     true,
			OnExhausted: mode,
			Steps:       steps,
		}},
	}, log.New(io.Discard, "", 0), proxy.WithHeaderOverrides())
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	return handler
}

func statuses(handler http.Handler, path string, count int) []int {
	result := make([]int, 0, count)
	for range count {
		result = append(result, statusFor(handler, path))
	}
	return result
}

func servedByName(handler *proxy.Handler) map[string]uint64 {
	served := make(map[string]uint64)
	for _, state := range handler.Scenarios() {
		served[state.Name] = state.Served
	}
	return served
}
