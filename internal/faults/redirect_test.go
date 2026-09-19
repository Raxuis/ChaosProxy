package faults

import (
	"net/http"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestRedirectBeforeReturnsSyntheticResponse(t *testing.T) {
	fault := newRedirectFault(config.RedirectConfig{
		Code:        http.StatusFound,
		Location:    "/login",
		Probability: 1,
	})

	var emitted Injection
	ctx := newFaultContext()
	ctx.Emit = func(inj Injection) {
		emitted = inj
	}

	shortCircuit, err := fault.Before(ctx)
	if err != nil {
		t.Fatalf("Before() unexpected error: %v", err)
	}
	if shortCircuit == nil || shortCircuit.Status != http.StatusFound {
		t.Fatalf("Before() = %#v, want 302 short circuit", shortCircuit)
	}
	if got := shortCircuit.Headers.Get("Location"); got != "/login" {
		t.Errorf("Location = %q, want /login", got)
	}
	if emitted.Fault != "redirect" {
		t.Errorf("emitted fault = %q, want redirect", emitted.Fault)
	}
}

func TestRedirectProbabilityZeroDoesNotTrigger(t *testing.T) {
	fault := newRedirectFault(config.RedirectConfig{
		Code:        http.StatusTemporaryRedirect,
		Location:    "/login",
		Probability: 0,
	})
	shortCircuit, err := fault.Before(newFaultContext())
	if err != nil || shortCircuit != nil {
		t.Fatalf("Before() = (%#v, %v), want no short circuit", shortCircuit, err)
	}
}
