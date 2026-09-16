package proxy

import (
	"errors"

	"github.com/Raxuis/chaosproxy/internal/events"
)

// EventPublisher receives completed data-plane request events.
type EventPublisher interface {
	Publish(events.Event)
}

// Option customizes a Handler without expanding its constructor for optional
// integrations.
type Option func(*Handler) error

// WithEventPublisher publishes one completed event per data-plane request.
func WithEventPublisher(publisher EventPublisher) Option {
	return func(h *Handler) error {
		if publisher == nil {
			return errors.New("event publisher must not be nil")
		}
		h.publisher = publisher
		return nil
	}
}

// WithHeaderOverrides lets clients force faults for one request with the X-Chaos header.
func WithHeaderOverrides() Option {
	return func(h *Handler) error {
		h.headerOverrides = true
		return nil
	}
}
