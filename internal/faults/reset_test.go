package faults

import (
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestResetBeforeRequestsConnectionReset(t *testing.T) {
	shortCircuit, err := newResetFault(config.ResetConfig{Probability: 1}).Before(newFaultContext())
	if err != nil || shortCircuit == nil || !shortCircuit.Reset {
		t.Fatalf("Before() = (%#v, %v), want reset short circuit", shortCircuit, err)
	}

	shortCircuit, err = newResetFault(config.ResetConfig{Probability: 0}).Before(newFaultContext())
	if err != nil || shortCircuit != nil {
		t.Fatalf("Before() with probability 0 = (%#v, %v), want no short circuit", shortCircuit, err)
	}
}
