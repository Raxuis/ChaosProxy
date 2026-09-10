package faults

import (
	"errors"
	"io"
	"math"
	"net/http"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type truncateFault struct {
	BaseFault
	probability float64
	at          float64
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}

func newTruncateFault(configured config.TruncateConfig) (*truncateFault, error) {
	if err := validateProbability(configured.Probability); err != nil {
		return nil, err
	}
	if math.IsNaN(configured.At) || configured.At < 0 || configured.At > 1 {
		return nil, errors.New("at must be between 0 and 1")
	}
	return &truncateFault{probability: configured.Probability, at: configured.At}, nil
}

func (*truncateFault) Name() string {
	return "truncate"
}

func (fault *truncateFault) After(ctx *Context, response *http.Response) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if response == nil || response.Body == nil {
		return errors.New("upstream response and body must not be nil")
	}

	triggered, err := shouldTrigger(ctx.Rng, fault.probability)
	if err != nil || !triggered {
		return err
	}
	// A fractional cutoff cannot be derived for an unknown-length stream without
	// buffering it. Preserve such responses rather than breaking streaming.
	if response.ContentLength < 0 || fault.at >= 1 {
		return nil
	}

	limit := int64(math.Floor(float64(response.ContentLength) * fault.at))
	response.Body = &limitedReadCloser{
		Reader: io.LimitReader(response.Body, limit),
		Closer: response.Body,
	}
	response.ContentLength = -1
	response.Header.Del("Content-Length")
	emit(ctx, Event{Faults: []string{fault.Name()}})
	return nil
}
