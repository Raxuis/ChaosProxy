// Package faults implements request and response fault injection.
package faults

import (
	"math/rand"
	"net/http"

	"github.com/Raxuis/chaosproxy/internal/events"
)

// Event aliases the shared observability event without making the events
// module depend on individual fault implementations.
type Event = events.Event

// Fault can intercept a request before forwarding and transform its upstream
// response afterward.
type Fault interface {
	Name() string
	Before(ctx *Context) (*ShortCircuit, error)
	After(ctx *Context, response *http.Response) error
}

// Context contains request-scoped dependencies and observability hooks. Req and
// Rng must be non-nil. A random generator belongs to exactly one request.
type Context struct {
	Req  *http.Request
	Rng  *rand.Rand
	Rule string
	Emit func(Event)
}

// ShortCircuit describes a response that bypasses the upstream.
type ShortCircuit struct {
	Status  int
	Headers http.Header
	Body    []byte
	Hang    bool
}

// BaseFault provides no-op hooks for faults that affect only one pipeline phase.
type BaseFault struct{}

// Before leaves the request unchanged.
func (BaseFault) Before(*Context) (*ShortCircuit, error) {
	return nil, nil
}

// After leaves the response unchanged.
func (BaseFault) After(*Context, *http.Response) error {
	return nil
}
