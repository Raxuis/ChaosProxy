package faults

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
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

func newFaultContext() *Context {
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil)
	return &Context{
		Req:  request,
		Rng:  rand.New(rand.NewSource(42)),
		Rule: "test-rule",
	}
}
