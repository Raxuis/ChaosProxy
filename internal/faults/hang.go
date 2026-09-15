package faults

import "github.com/Raxuis/chaosproxy/internal/config"

type hangFault struct {
	BaseFault
	probability float64
}

func newHangFault(configured config.HangConfig) *hangFault {
	return &hangFault{probability: configured.Probability}
}

func (*hangFault) Name() string {
	return "hang"
}

func (fault *hangFault) Before(ctx *Context) (*ShortCircuit, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}

	triggered, err := shouldTrigger(ctx.Rng, fault.probability)
	if err != nil || !triggered {
		return nil, err
	}
	emit(ctx, Event{Faults: []string{fault.Name()}})
	return &ShortCircuit{Hang: true}, nil
}
