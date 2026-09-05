package faults

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestBuildUsesDeterministicOrder(t *testing.T) {
	chain, err := Build(config.Rule{
		Name:     "all-faults",
		Latency:  &config.LatencyConfig{Dist: "fixed", Value: time.Millisecond},
		Status:   &config.StatusConfig{Code: 503, Probability: 0.5},
		Hang:     &config.HangConfig{Probability: 0.5},
		Truncate: &config.TruncateConfig{Probability: 0.5, At: 0.5},
	})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	names := make([]string, len(chain))
	for index, fault := range chain {
		names[index] = fault.Name()
	}
	if want := []string{"latency", "status", "hang", "truncate"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("fault order = %v, want %v", names, want)
	}
}

func TestBuildRejectsInvalidRules(t *testing.T) {
	tests := []struct {
		name string
		rule config.Rule
	}{
		{name: "no faults", rule: config.Rule{Name: "empty"}},
		{name: "latency", rule: config.Rule{Name: "invalid", Latency: &config.LatencyConfig{Dist: "unknown"}}},
		{name: "status", rule: config.Rule{Name: "invalid", Status: &config.StatusConfig{Code: 700, Probability: 1}}},
		{name: "hang", rule: config.Rule{Name: "invalid", Hang: &config.HangConfig{Probability: -1}}},
		{name: "truncate", rule: config.Rule{Name: "invalid", Truncate: &config.TruncateConfig{Probability: 1, At: math.NaN()}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Build(test.rule); err == nil {
				t.Fatalf("Build(%+v) error = nil", test.rule)
			}
		})
	}
}
