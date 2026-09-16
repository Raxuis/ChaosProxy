package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/proxy"
	"github.com/Raxuis/chaosproxy/internal/report"
)

func TestRecorderAggregatesRun(t *testing.T) {
	t.Parallel()

	recorder := report.NewRecorder(report.Options{})
	recorder.Publish(events.Event{Method: "GET", Path: "/search", Rule: "slow", Faults: []string{"latency"}, Status: 200, DurationMs: 810, UpstreamLatencyMs: 10, InjectedLatencyMs: 800})
	recorder.Publish(events.Event{Method: "GET", Path: "/cart", Status: 200, DurationMs: 12, UpstreamLatencyMs: 12})
	recorder.Publish(events.Event{Method: "POST", Path: "/checkout", Rule: "checkout", Faults: []string{"status"}, Status: 503, DurationMs: 1})
	recorder.Publish(events.Event{Method: "GET", Path: "/down", Status: 502, DurationMs: 3, Error: "dial tcp 127.0.0.1:9000: connection refused"})

	finished := time.Now()
	scenarios := []proxy.ScenarioState{{Name: "checkout", Served: 1}}
	got := recorder.Report(finished, report.Run{Target: "http://localhost:9000", Seed: 42, StoppedBy: "signal", Scenarios: scenarios})

	wantTotals := report.Totals{Requests: 4, Injected: 2, Errors: 1, Statuses: map[string]uint64{"2xx": 2, "5xx": 2}}
	if !reflect.DeepEqual(got.Totals, wantTotals) {
		t.Errorf("totals = %+v, want %+v", got.Totals, wantTotals)
	}
	if rule := got.Rules["slow"]; rule.Matched != 1 || rule.Faulted != 1 || rule.Faults["latency"] != 1 {
		t.Errorf("slow rule = %+v, want one latency injection", rule)
	}
	if want := (report.Percentiles{P50: 3, P90: 810, P95: 810, P99: 810, Max: 810}); got.LatencyMs.Total != want {
		t.Errorf("total latency = %+v, want %+v", got.LatencyMs.Total, want)
	}
	if got.LatencyMs.Injected.Max != 800 || got.LatencyMs.Upstream.Max != 12 {
		t.Errorf("latency = %+v, want injected max 800 and upstream max 12", got.LatencyMs)
	}
	if len(got.Errors) != 1 || got.Errors[0].Path != "/down" || got.Errors[0].Time.IsZero() {
		t.Errorf("errors = %+v, want the /down failure with a time", got.Errors)
	}
	if got.Seed != 42 || got.StoppedBy != "signal" || !reflect.DeepEqual(got.Scenarios, scenarios) || got.DurationMs < 0 {
		t.Errorf("run fields = %+v", got)
	}
}

func TestRecorderReportsEmptyCollectionsAsJSONArrays(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(report.NewRecorder(report.Options{}).Report(time.Now(), report.Run{}))
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}
	for _, want := range []string{`"scenarios":[]`, `"errors":[]`, `"p50":0`} {
		if !strings.Contains(string(encoded), want) {
			t.Errorf("report %s does not contain %s", encoded, want)
		}
	}
}

func TestRecorderCallbacksRunOnce(t *testing.T) {
	t.Parallel()

	limits, failures := 0, 0
	recorder := report.NewRecorder(report.Options{
		MaxRequests:   2,
		OnMaxRequests: func() { limits++ },
		OnError:       func(events.Event) { failures++ },
	})
	for index := range 150 {
		event := events.Event{Status: 200}
		if index%2 == 0 {
			event = events.Event{Status: 502, Error: "upstream unavailable"}
		}
		recorder.Publish(event)
	}

	if limits != 1 || failures != 1 {
		t.Fatalf("callbacks limit/error = %d/%d, want 1/1", limits, failures)
	}
	if recorder.ErrorCount() != 75 {
		t.Fatalf("ErrorCount() = %d, want 75", recorder.ErrorCount())
	}
	if got := recorder.Report(time.Now(), report.Run{}); len(got.Errors) != 75 || got.Totals.Errors != 75 {
		t.Fatalf("errors = %d recorded / %d total, want 75/75", len(got.Errors), got.Totals.Errors)
	}
}

func TestRecorderCapsRecordedErrors(t *testing.T) {
	t.Parallel()

	recorder := report.NewRecorder(report.Options{})
	for range 150 {
		recorder.Publish(events.Event{Status: 502, Error: "upstream unavailable"})
	}
	if got := recorder.Report(time.Now(), report.Run{}); len(got.Errors) != 100 || got.Totals.Errors != 150 {
		t.Fatalf("errors = %d recorded / %d total, want 100/150", len(got.Errors), got.Totals.Errors)
	}
}

func TestWriteFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "chaos-report.json")
	written := report.NewRecorder(report.Options{}).Report(time.Now(), report.Run{Target: "http://localhost:9000", StoppedBy: "max-requests"})
	if err := report.WriteFile(path, written); err != nil {
		t.Fatalf("WriteFile() unexpected error: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var decoded report.Report
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.StoppedBy != "max-requests" || decoded.Target != "http://localhost:9000" {
		t.Fatalf("decoded report = %+v, want the written fields", decoded)
	}
}
