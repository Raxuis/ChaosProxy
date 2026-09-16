// Package events defines runtime observability events.
package events

import "time"

// Event describes the outcome of one proxied request.
type Event struct {
	ID                uint64    `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	Method            string    `json:"method"`
	Path              string    `json:"path"`
	Rule              string    `json:"rule,omitempty"`
	Faults            []string  `json:"faults,omitempty"`
	Details           []string  `json:"details,omitempty"`
	InjectedLatencyMs int64     `json:"injected_latency_ms"`
	UpstreamLatencyMs int64     `json:"upstream_latency_ms"`
	Status            int       `json:"status"`
	Bytes             int64     `json:"bytes"`
	DurationMs        int64     `json:"duration_ms"`
	Error             string    `json:"error,omitempty"`
}
