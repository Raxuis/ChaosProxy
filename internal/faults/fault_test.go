package faults

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type beforeOnlyFault struct {
	BaseFault
}

func (beforeOnlyFault) Name() string {
	return "before-only"
}

func (beforeOnlyFault) Before(*Context) (*ShortCircuit, error) {
	return &ShortCircuit{Status: http.StatusTeapot}, nil
}

func TestBaseFaultSupportsSingleHookImplementations(t *testing.T) {
	var fault Fault = beforeOnlyFault{}

	shortCircuit, err := fault.Before(newFaultContext())
	if err != nil {
		t.Fatalf("Before() unexpected error: %v", err)
	}
	if shortCircuit == nil || shortCircuit.Status != http.StatusTeapot {
		t.Fatalf("Before() = %#v, want teapot short circuit", shortCircuit)
	}
	if err := fault.After(newFaultContext(), &http.Response{}); err != nil {
		t.Fatalf("inherited After() unexpected error: %v", err)
	}
}

func TestBaseFaultBeforeIsNoOp(t *testing.T) {
	shortCircuit, err := (BaseFault{}).Before(newFaultContext())
	if err != nil || shortCircuit != nil {
		t.Fatalf("Before() = (%#v, %v), want no-op", shortCircuit, err)
	}
}

func TestFaultsReportInjections(t *testing.T) {
	var injections []Injection
	ctx := newFaultContext()
	ctx.Emit = func(injection Injection) {
		injections = append(injections, injection)
	}

	if _, err := newLatencyFault(config.LatencyConfig{Dist: "fixed", Value: time.Millisecond}).Before(ctx); err != nil {
		t.Fatalf("latency Before() unexpected error: %v", err)
	}
	if _, err := newStatusFault(config.StatusConfig{Code: 503, Probability: 1}).Before(ctx); err != nil {
		t.Fatalf("status Before() unexpected error: %v", err)
	}
	if _, err := newHangFault(config.HangConfig{Probability: 1}).Before(ctx); err != nil {
		t.Fatalf("hang Before() unexpected error: %v", err)
	}

	want := []Injection{{Fault: "latency", Latency: time.Millisecond}, {Fault: "status"}, {Fault: "hang"}}
	if !reflect.DeepEqual(injections, want) {
		t.Fatalf("injections = %+v, want %+v", injections, want)
	}
}

func newFaultContext() *Context {
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	return &Context{
		Req:  request,
		Rng:  rand.New(rand.NewSource(42)),
		Rule: "test-rule",
	}
}
