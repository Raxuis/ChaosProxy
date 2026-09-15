package faults

import "math/rand/v2"

func shouldTrigger(rng *rand.Rand, probability float64) bool {
	switch {
	case probability <= 0:
		return false
	case probability >= 1:
		return true
	default:
		return rng.Float64() < probability
	}
}

func emit(ctx *Context, injection Injection) {
	if ctx.Emit != nil {
		ctx.Emit(injection)
	}
}
