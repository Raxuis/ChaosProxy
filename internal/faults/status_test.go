package faults

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestStatusBeforeReturnsSyntheticResponse(t *testing.T) {
	fault := newStatusFault(config.StatusConfig{
		Code:        http.StatusServiceUnavailable,
		Probability: 1,
		RetryAfter:  2,
	})

	shortCircuit, err := fault.Before(newFaultContext())
	if err != nil {
		t.Fatalf("Before() unexpected error: %v", err)
	}
	if shortCircuit == nil || shortCircuit.Status != http.StatusServiceUnavailable {
		t.Fatalf("Before() = %#v, want 503 short circuit", shortCircuit)
	}
	if got := shortCircuit.Headers.Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After = %q, want 2", got)
	}
	if got := shortCircuit.Headers.Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want JSON", got)
	}

	var body struct {
		Error string `json:"error"`
		Rule  string `json:"rule"`
	}
	if err := json.Unmarshal(shortCircuit.Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "injected by chaosproxy" || body.Rule != "test-rule" {
		t.Errorf("body = %+v, want injected error and rule", body)
	}
}

func TestStatusProbabilityZeroDoesNotTrigger(t *testing.T) {
	fault := newStatusFault(config.StatusConfig{Code: 503, Probability: 0})
	shortCircuit, err := fault.Before(newFaultContext())
	if err != nil || shortCircuit != nil {
		t.Fatalf("Before() = (%#v, %v), want no short circuit", shortCircuit, err)
	}
}
