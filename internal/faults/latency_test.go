package faults

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLatencyFixedSamples(t *testing.T) {
	fault, err := newLatencyFault(config.LatencyConfig{
		Dist:   "fixed",
		Value:  100 * time.Millisecond,
		Jitter: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("newLatencyFault() unexpected error: %v", err)
	}

	rng := rand.New(rand.NewSource(42))
	for index := 0; index < 1_000; index++ {
		sample, err := fault.sample(rng)
		if err != nil {
			t.Fatalf("sample() unexpected error: %v", err)
		}
		if sample < 75*time.Millisecond || sample > 125*time.Millisecond {
			t.Fatalf("sample() = %s, want value within jitter bounds", sample)
		}
	}
}

func TestLatencyLognormalParametersAndSampling(t *testing.T) {
	const (
		p50 = 400 * time.Millisecond
		p99 = 3 * time.Second
	)
	fault, err := newLatencyFault(config.LatencyConfig{Dist: "lognormal", P50: p50, P99: p99})
	if err != nil {
		t.Fatalf("newLatencyFault() unexpected error: %v", err)
	}

	derivedP50 := time.Duration(math.Exp(fault.mu))
	if difference := derivedP50 - p50; difference < -time.Nanosecond || difference > time.Nanosecond {
		t.Errorf("derived p50 = %s, want %s", derivedP50, p50)
	}
	derivedP99 := time.Duration(math.Exp(fault.mu + standardNormalP99*fault.sigma))
	if difference := derivedP99 - p99; difference < -time.Nanosecond || difference > time.Nanosecond {
		t.Errorf("derived p99 = %s, want %s", derivedP99, p99)
	}

	first, err := fault.sample(rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatalf("sample() unexpected error: %v", err)
	}
	second, err := fault.sample(rand.New(rand.NewSource(7)))
	if err != nil {
		t.Fatalf("sample() unexpected error: %v", err)
	}
	if first != second || first <= 0 {
		t.Fatalf("seeded samples = (%s, %s), want equal positive durations", first, second)
	}
}

func TestLatencyLognormalWithEqualPercentilesIsExact(t *testing.T) {
	const duration = 400 * time.Millisecond
	fault, err := newLatencyFault(config.LatencyConfig{Dist: "lognormal", P50: duration, P99: duration})
	if err != nil {
		t.Fatalf("newLatencyFault() unexpected error: %v", err)
	}

	sample, err := fault.sample(rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("sample() unexpected error: %v", err)
	}
	if sample != duration {
		t.Fatalf("sample() = %s, want exact %s", sample, duration)
	}
}

func TestLatencyBeforeHonorsCancellation(t *testing.T) {
	fault, err := newLatencyFault(config.LatencyConfig{Dist: "fixed", Value: time.Hour})
	if err != nil {
		t.Fatalf("newLatencyFault() unexpected error: %v", err)
	}

	faultContext := newFaultContext()
	canceled, cancel := context.WithCancel(faultContext.Req.Context())
	defer cancel()
	faultContext.Req = faultContext.Req.WithContext(canceled)

	result := make(chan error, 1)
	go func() {
		_, beforeErr := fault.Before(faultContext)
		result <- beforeErr
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Before() error = %v, want context.Canceled", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Before() did not stop promptly after cancellation")
	}
}

func TestLatencyConfigurationErrors(t *testing.T) {
	tests := []struct {
		name       string
		configured config.LatencyConfig
	}{
		{name: "unknown distribution", configured: config.LatencyConfig{Dist: "normal"}},
		{name: "negative value", configured: config.LatencyConfig{Dist: "fixed", Value: -1}},
		{name: "negative jitter", configured: config.LatencyConfig{Dist: "fixed", Jitter: -1}},
		{name: "missing percentiles", configured: config.LatencyConfig{Dist: "lognormal"}},
		{name: "reversed percentiles", configured: config.LatencyConfig{Dist: "lognormal", P50: time.Second, P99: time.Millisecond}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newLatencyFault(test.configured); err == nil {
				t.Fatalf("newLatencyFault(%+v) error = nil", test.configured)
			}
		})
	}
}

func TestLatencyRequiresRequestContext(t *testing.T) {
	fault, err := newLatencyFault(config.LatencyConfig{Dist: "fixed"})
	if err != nil {
		t.Fatalf("newLatencyFault() unexpected error: %v", err)
	}
	if _, err := fault.Before(nil); err == nil {
		t.Fatal("Before(nil) error = nil")
	}
}
