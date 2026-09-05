package faults

import (
	"runtime"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestHangBeforeReturnsImmediatelyWithoutGoroutine(t *testing.T) {
	fault, err := newHangFault(config.HangConfig{Probability: 1})
	if err != nil {
		t.Fatalf("newHangFault() unexpected error: %v", err)
	}

	runtime.GC()
	before := runtime.NumGoroutine()
	for index := 0; index < 10_000; index++ {
		shortCircuit, err := fault.Before(newFaultContext())
		if err != nil {
			t.Fatalf("Before() unexpected error: %v", err)
		}
		if shortCircuit == nil || !shortCircuit.Hang {
			t.Fatalf("Before() = %#v, want hang short circuit", shortCircuit)
		}
	}
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before {
		t.Fatalf("goroutines before/after = %d/%d, want no leak", before, after)
	}
}

func TestHangProbabilityZeroDoesNotTrigger(t *testing.T) {
	fault, err := newHangFault(config.HangConfig{Probability: 0})
	if err != nil {
		t.Fatalf("newHangFault() unexpected error: %v", err)
	}
	shortCircuit, err := fault.Before(newFaultContext())
	if err != nil || shortCircuit != nil {
		t.Fatalf("Before() = (%#v, %v), want no short circuit", shortCircuit, err)
	}
}

func TestHangRejectsInvalidProbability(t *testing.T) {
	if _, err := newHangFault(config.HangConfig{Probability: 2}); err == nil {
		t.Fatal("newHangFault() error = nil")
	}
}
