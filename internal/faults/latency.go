package faults

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

const standardNormalP99 = 2.3263478740408408

type latencyFault struct {
	BaseFault
	distribution string
	value        time.Duration
	jitter       time.Duration
	mu           float64
	sigma        float64
}

func newLatencyFault(configured config.LatencyConfig) (*latencyFault, error) {
	fault := &latencyFault{
		distribution: configured.Dist,
		value:        configured.Value,
		jitter:       configured.Jitter,
	}

	switch configured.Dist {
	case "fixed":
		if configured.Value < 0 {
			return nil, errors.New("value must not be negative")
		}
		if configured.Jitter < 0 {
			return nil, errors.New("jitter must not be negative")
		}
	case "lognormal":
		if configured.P50 <= 0 || configured.P99 <= 0 {
			return nil, errors.New("p50 and p99 must be greater than zero")
		}
		if configured.P50 > configured.P99 {
			return nil, errors.New("p50 must not exceed p99")
		}
		fault.mu = math.Log(float64(configured.P50))
		fault.sigma = math.Log(float64(configured.P99)/float64(configured.P50)) / standardNormalP99
		fault.value = configured.P50
	default:
		return nil, errors.New("distribution must be either fixed or lognormal")
	}

	return fault, nil
}

func (*latencyFault) Name() string {
	return "latency"
}

func (fault *latencyFault) Before(ctx *Context) (*ShortCircuit, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}

	delay, err := fault.sample(ctx.Rng)
	if err != nil {
		return nil, err
	}
	if err := wait(ctx.Req.Context(), delay); err != nil {
		return nil, err
	}
	return nil, nil
}

func (fault *latencyFault) sample(rng *rand.Rand) (time.Duration, error) {
	switch fault.distribution {
	case "fixed":
		if fault.jitter == 0 {
			return fault.value, nil
		}
		if rng == nil {
			return 0, errors.New("request random generator must not be nil")
		}
		offset := (rng.Float64()*2 - 1) * float64(fault.jitter)
		return boundedDuration(float64(fault.value) + offset), nil
	case "lognormal":
		if fault.sigma == 0 {
			return fault.value, nil
		}
		if rng == nil {
			return 0, errors.New("request random generator must not be nil")
		}
		return boundedDuration(math.Exp(fault.mu + fault.sigma*rng.NormFloat64())), nil
	default:
		return 0, errors.New("unsupported latency distribution")
	}
}

func wait(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func boundedDuration(value float64) time.Duration {
	if math.IsNaN(value) || value <= 0 {
		return 0
	}
	if math.IsInf(value, 1) || value >= float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(value)
}
