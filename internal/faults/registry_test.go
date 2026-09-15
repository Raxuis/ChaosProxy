package faults

import (
	"reflect"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestBuildUsesDeterministicOrder(t *testing.T) {
	chain := Build(config.Rule{
		Name:     "all-faults",
		Latency:  &config.LatencyConfig{Dist: "fixed", Value: time.Millisecond},
		Status:   &config.StatusConfig{Code: 503, Probability: 0.5},
		Hang:     &config.HangConfig{Probability: 0.5},
		Truncate: &config.TruncateConfig{Probability: 0.5, At: 0.5},
	})

	names := make([]string, len(chain))
	for index, fault := range chain {
		names[index] = fault.Name()
	}
	if want := []string{"latency", "status", "hang", "truncate"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("fault order = %v, want %v", names, want)
	}
}
