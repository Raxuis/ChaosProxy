package faults

import "github.com/Raxuis/chaosproxy/internal/config"

type hangFault struct {
	BaseFault
	probability float64
}

func newHangFault(cfg config.HangConfig) *hangFault {
	return &hangFault{probability: cfg.Probability}
}

func (*hangFault) Name() string {
	return "hang"
}

func (f *hangFault) Before(ctx *Context) (*ShortCircuit, error) {
	if !shouldTrigger(ctx.Rng, f.probability) {
		return nil, nil
	}
	emit(ctx, Injection{Fault: f.Name()})
	return &ShortCircuit{Hang: true}, nil
}
