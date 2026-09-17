package faults

import (
	"reflect"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestBuildUsesDeterministicOrder(t *testing.T) {
	chain := Build(config.Rule{
		Name:      "all-faults",
		Latency:   &config.LatencyConfig{Dist: "fixed", Value: time.Millisecond},
		Status:    &config.StatusConfig{Code: 503, Probability: 0.5},
		Hang:      &config.HangConfig{Probability: 0.5},
		Headers:   &config.HeadersConfig{Probability: 0.5, Set: map[string]string{"Cache-Control": "no-cache"}},
		Truncate:  &config.TruncateConfig{Probability: 0.5, At: 0.5},
		Reset:     &config.ResetConfig{Probability: 0.5},
		Bandwidth: &config.BandwidthConfig{BytesPerSecond: 1024},
		Mutate:    &config.MutateConfig{Probability: 1, MaxBytes: 1024, Operations: []config.MutationConfig{{Op: "drop", Path: "id"}}},
	})

	names := make([]string, len(chain))
	for index, fault := range chain {
		names[index] = fault.Name()
	}
	if want := []string{"latency", "reset", "status", "hang", "mutate", "headers", "truncate", "bandwidth"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("fault order = %v, want %v", names, want)
	}
}
