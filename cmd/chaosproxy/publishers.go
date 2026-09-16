package main

import (
	"context"
	"errors"

	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

var (
	errMaxRequests   = errors.New("max requests reached")
	errRequestFailed = errors.New("request failed inside the proxy")
)

type publishers []proxy.EventPublisher

func (p publishers) Publish(event events.Event) {
	for _, publisher := range p {
		publisher.Publish(event)
	}
}

func stopReason(stopRun context.Context, runErr error) string {
	cause := context.Cause(stopRun)
	switch {
	case errors.Is(cause, errMaxRequests):
		return "max-requests"
	case errors.Is(cause, errRequestFailed):
		return "error"
	case runErr != nil:
		return "server-error"
	default:
		return "signal"
	}
}
