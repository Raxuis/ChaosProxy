// Package report aggregates data-plane events into a JSON run report.
package report

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

const maxRecordedErrors = 100

// Options configures the request limit and the callbacks that end a run.
type Options struct {
	MaxRequests   uint64
	OnMaxRequests func()
	OnError       func(events.Event)
}

// Run describes the proxy state that events do not carry.
type Run struct {
	Version   string
	Target    string
	Seed      int64
	StoppedBy string
	Scenarios []proxy.ScenarioState
}

// Report is the JSON document written at the end of a run.
type Report struct {
	StartedAt  time.Time                   `json:"started_at"`
	FinishedAt time.Time                   `json:"finished_at"`
	DurationMs int64                       `json:"duration_ms"`
	StoppedBy  string                      `json:"stopped_by"`
	Version    string                      `json:"version"`
	Target     string                      `json:"target"`
	Seed       int64                       `json:"seed"`
	Totals     Totals                      `json:"totals"`
	Rules      map[string]events.RuleStats `json:"rules"`
	Scenarios  []proxy.ScenarioState       `json:"scenarios"`
	LatencyMs  Latency                     `json:"latency_ms"`
	Errors     []RequestError              `json:"errors"`
}

// Totals counts every request of the run.
type Totals struct {
	Requests uint64            `json:"requests"`
	Injected uint64            `json:"injected"`
	Errors   uint64            `json:"errors"`
	Statuses map[string]uint64 `json:"statuses"`
}

// Latency summarizes total, upstream, and injected latency distributions.
type Latency struct {
	Total    Percentiles `json:"total"`
	Upstream Percentiles `json:"upstream"`
	Injected Percentiles `json:"injected"`
}

// Percentiles summarizes one latency distribution in milliseconds.
type Percentiles struct {
	P50 int64 `json:"p50"`
	P90 int64 `json:"p90"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
	Max int64 `json:"max"`
}

// RequestError records one request that failed inside the proxy.
type RequestError struct {
	Time   time.Time `json:"time"`
	Method string    `json:"method"`
	Path   string    `json:"path"`
	Rule   string    `json:"rule,omitempty"`
	Status int       `json:"status"`
	Error  string    `json:"error"`
}

// Recorder aggregates every published data-plane event of a run.
type Recorder struct {
	options       Options
	mu            sync.Mutex
	started       time.Time
	tally         *events.Tally
	errors        uint64
	failures      []RequestError
	total         []int64
	upstream      []int64
	injected      []int64
	limitReported bool
	errorReported bool
}

// NewRecorder starts recording a run now.
func NewRecorder(options Options) *Recorder {
	return &Recorder{options: options, started: time.Now().UTC(), tally: events.NewTally()}
}

// Publish records one completed request and runs the limit and error callbacks once.
func (r *Recorder) Publish(event events.Event) {
	limitReached, firstError := r.record(event)
	if limitReached && r.options.OnMaxRequests != nil {
		r.options.OnMaxRequests()
	}
	if firstError && r.options.OnError != nil {
		r.options.OnError(event)
	}
}

// ErrorCount returns how many requests failed inside the proxy.
func (r *Recorder) ErrorCount() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errors
}

// Report summarizes the run up to finished.
func (r *Recorder) Report(finished time.Time, run Run) Report {
	r.mu.Lock()
	defer r.mu.Unlock()

	scenarios := run.Scenarios
	if scenarios == nil {
		scenarios = []proxy.ScenarioState{}
	}
	return Report{
		StartedAt:  r.started,
		FinishedAt: finished.UTC(),
		DurationMs: finished.Sub(r.started).Milliseconds(),
		StoppedBy:  run.StoppedBy,
		Version:    run.Version,
		Target:     run.Target,
		Seed:       run.Seed,
		Totals: Totals{
			Requests: r.tally.Requests(),
			Injected: r.tally.Faulted(),
			Errors:   r.errors,
			Statuses: r.tally.Statuses(),
		},
		Rules:     r.tally.Rules(),
		Scenarios: scenarios,
		LatencyMs: Latency{
			Total:    percentiles(r.total),
			Upstream: percentiles(r.upstream),
			Injected: percentiles(r.injected),
		},
		Errors: append([]RequestError{}, r.failures...),
	}
}

// WriteFile writes report as indented JSON to path, or to standard output when path is "-".
func WriteFile(path string, report Report) error {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	encoded = append(encoded, '\n')
	if path == "-" {
		_, err = os.Stdout.Write(encoded)
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

func (r *Recorder) record(event events.Event) (bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tally.Add(event)
	r.total = append(r.total, event.DurationMs)
	r.upstream = append(r.upstream, event.UpstreamLatencyMs)
	r.injected = append(r.injected, event.InjectedLatencyMs)

	firstError := false
	if event.Error != "" {
		r.errors++
		if len(r.failures) < maxRecordedErrors {
			timestamp := event.Timestamp
			if timestamp.IsZero() {
				timestamp = time.Now().UTC()
			}
			r.failures = append(r.failures, RequestError{
				Time:   timestamp,
				Method: event.Method,
				Path:   event.Path,
				Rule:   event.Rule,
				Status: event.Status,
				Error:  event.Error,
			})
		}
		firstError = !r.errorReported
		r.errorReported = true
	}

	limitReached := false
	if r.options.MaxRequests > 0 && r.tally.Requests() >= r.options.MaxRequests && !r.limitReported {
		r.limitReported = true
		limitReached = true
	}
	return limitReached, firstError
}

func percentiles(samples []int64) Percentiles {
	if len(samples) == 0 {
		return Percentiles{}
	}
	sorted := slices.Clone(samples)
	slices.Sort(sorted)
	rank := func(quantile float64) int64 {
		index := int(math.Ceil(quantile*float64(len(sorted)))) - 1
		return sorted[max(index, 0)]
	}
	return Percentiles{P50: rank(0.5), P90: rank(0.9), P95: rank(0.95), P99: rank(0.99), Max: sorted[len(sorted)-1]}
}
