package faults

import "github.com/Raxuis/chaosproxy/internal/config"

type resetFault struct {
	BaseFault
	probability float64
}

func newResetFault(cfg config.ResetConfig) *resetFault {
	return &resetFault{probability: cfg.Probability}
}

func (*resetFault) Name() string {
	return "reset"
}

func (f *resetFault) Before(ctx *Context) (*ShortCircuit, error) {
	if !shouldTrigger(ctx.Rng, f.probability) {
		return nil, nil
	}
	emit(ctx, Injection{Fault: f.Name()})
	return &ShortCircuit{Reset: true}, nil
}
