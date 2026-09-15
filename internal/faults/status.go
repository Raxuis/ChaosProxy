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

func newStatusFault(cfg config.StatusConfig) *statusFault {
	return &statusFault{
		code:        cfg.Code,
		probability: cfg.Probability,
		retryAfter:  cfg.RetryAfter,
	}
}

func (*statusFault) Name() string {
	return "status"
}

func (f *statusFault) Before(ctx *Context) (*ShortCircuit, error) {
	if !shouldTrigger(ctx.Rng, f.probability) {
		return nil, nil
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
	if f.retryAfter > 0 {
		headers.Set("Retry-After", strconv.Itoa(f.retryAfter))
	}
	emit(ctx, Injection{Fault: f.Name()})

	return &ShortCircuit{
		Status:  f.code,
		Headers: headers,
		Body:    body,
	}, nil
}
