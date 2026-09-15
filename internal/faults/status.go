package faults

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Raxuis/chaosproxy/internal/config"
)

type statusFault struct {
	BaseFault
	code        int
	probability float64
	retryAfter  int
}

func newStatusFault(configured config.StatusConfig) *statusFault {
	return &statusFault{
		code:        configured.Code,
		probability: configured.Probability,
		retryAfter:  configured.RetryAfter,
	}
}

func (*statusFault) Name() string {
	return "status"
}

func (fault *statusFault) Before(ctx *Context) (*ShortCircuit, error) {
	if err := validateContext(ctx); err != nil {
		return nil, err
	}

	triggered, err := shouldTrigger(ctx.Rng, fault.probability)
	if err != nil || !triggered {
		return nil, err
	}

	body, err := json.Marshal(struct {
		Error string `json:"error"`
		Rule  string `json:"rule"`
	}{
		Error: "injected by chaosproxy",
		Rule:  ctx.Rule,
	})
	if err != nil {
		return nil, fmt.Errorf("encode status response: %w", err)
	}
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json; charset=utf-8")
	if fault.retryAfter > 0 {
		headers.Set("Retry-After", strconv.Itoa(fault.retryAfter))
	}
	emit(ctx, Injection{Fault: fault.Name()})

	return &ShortCircuit{
		Status:  fault.code,
		Headers: headers,
		Body:    body,
	}, nil
}
