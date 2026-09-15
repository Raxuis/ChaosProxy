package faults

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLatencyFixedSamples(t *testing.T) {
	fault := newLatencyFault(config.LatencyConfig{
		Dist:   "fixed",
		Value:  100 * time.Millisecond,
		Jitter: 25 * time.Millisecond,
	})

	rng := rand.New(rand.NewPCG(42, 0))
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
	fault := newLatencyFault(config.LatencyConfig{Dist: "lognormal", P50: p50, P99: p99})

	derivedP50 := time.Duration(math.Exp(fault.mu))
	if difference := derivedP50 - p50; difference < -time.Nanosecond || difference > time.Nanosecond {
		t.Errorf("derived p50 = %s, want %s", derivedP50, p50)
	}
	derivedP99 := time.Duration(math.Exp(fault.mu + standardNormalP99*fault.sigma))
	if difference := derivedP99 - p99; difference < -time.Nanosecond || difference > time.Nanosecond {
		t.Errorf("derived p99 = %s, want %s", derivedP99, p99)
	}

	first, err := fault.sample(rand.New(rand.NewPCG(7, 0)))
	if err != nil {
		t.Fatalf("sample() unexpected error: %v", err)
	}
	second, err := fault.sample(rand.New(rand.NewPCG(7, 0)))
	if err != nil {
		t.Fatalf("sample() unexpected error: %v", err)
	}
	if first != second || first <= 0 {
		t.Fatalf("seeded samples = (%s, %s), want equal positive durations", first, second)
	}
}

func TestLatencyLognormalWithEqualPercentilesIsExact(t *testing.T) {
	const duration = 400 * time.Millisecond
	fault := newLatencyFault(config.LatencyConfig{Dist: "lognormal", P50: duration, P99: duration})

	sample, err := fault.sample(rand.New(rand.NewPCG(42, 0)))
	if err != nil {
		t.Fatalf("sample() unexpected error: %v", err)
	}
	if sample != duration {
		t.Fatalf("sample() = %s, want exact %s", sample, duration)
	}
}

func TestLatencyBeforeHonorsCancellation(t *testing.T) {
	fault := newLatencyFault(config.LatencyConfig{Dist: "fixed", Value: time.Hour})

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

func TestLatencyRequiresRequestContext(t *testing.T) {
	fault := newLatencyFault(config.LatencyConfig{Dist: "fixed"})
	if _, err := fault.Before(nil); err == nil {
		t.Fatal("Before(nil) error = nil")
	}
}
