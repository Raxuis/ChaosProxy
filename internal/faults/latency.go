package faults

import (
	"context"
	"math"
	"math/rand/v2"
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

func newLatencyFault(cfg config.LatencyConfig) *latencyFault {
	fault := &latencyFault{
		distribution: cfg.Dist,
		value:        cfg.Value,
		jitter:       cfg.Jitter,
	}
	if cfg.Dist == "lognormal" {
		fault.mu = math.Log(float64(cfg.P50))
		fault.sigma = math.Log(float64(cfg.P99)/float64(cfg.P50)) / standardNormalP99
		fault.value = cfg.P50
	}
	return fault
}

func (*latencyFault) Name() string {
	return "latency"
}

func (f *latencyFault) Before(ctx *Context) (*ShortCircuit, error) {
	delay := f.sample(ctx.Rng)
	if err := wait(ctx.Req.Context(), delay); err != nil {
		return nil, err
	}
	emit(ctx, Injection{Fault: f.Name(), Latency: delay})
	return nil, nil
}

func (f *latencyFault) sample(rng *rand.Rand) time.Duration {
	switch {
	case f.distribution == "lognormal" && f.sigma != 0:
		return boundedDuration(math.Exp(f.mu + f.sigma*rng.NormFloat64()))
	case f.distribution == "fixed" && f.jitter != 0:
		offset := (rng.Float64()*2 - 1) * float64(f.jitter)
		return boundedDuration(float64(f.value) + offset)
	default:
		return f.value
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
