package faults

import (
	"net/http"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type redirectFault struct {
	BaseFault
	code        int
	location    string
	probability float64
}

func newRedirectFault(cfg config.RedirectConfig) *redirectFault {
	return &redirectFault{
		code:        cfg.Code,
		location:    cfg.Location,
		probability: cfg.Probability,
	}
}

func (*redirectFault) Name() string {
	return "redirect"
}

func (f *redirectFault) Before(ctx *Context) (*ShortCircuit, error) {
	if !shouldTrigger(ctx.Rng, f.probability) {
		return nil, nil
	}

	headers := make(http.Header)
	headers.Set("Location", f.location)
	emit(ctx, Injection{Fault: f.Name()})

	return &ShortCircuit{
		Status:  f.code,
		Headers: headers,
	}, nil
}
