package faults

import (
	"errors"
	"math/rand/v2"
)

func validateContext(ctx *Context) error {
	if ctx == nil {
		return errors.New("fault context must not be nil")
	}
	if ctx.Req == nil {
		return errors.New("request must not be nil")
	}
	if ctx.Rng == nil {
		return errors.New("request random generator must not be nil")
	}
	return nil
}

func shouldTrigger(rng *rand.Rand, probability float64) (bool, error) {
	if probability <= 0 {
		return false, nil
	}
	if probability >= 1 {
		return true, nil
	}
	if rng == nil {
		return false, errors.New("request random generator must not be nil")
	}
	return rng.Float64() < probability, nil
}

func emit(ctx *Context, injection Injection) {
	if ctx.Emit != nil {
		ctx.Emit(injection)
	}
}
