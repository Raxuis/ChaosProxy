package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
target: http://localhost:9000
seed: 42
rules:
  - name: slow-search
    match: GET /api/search*
    latency:
      dist: lognormal
      p50: 400ms
      p99: 3s
  - name: flaky-orders
    match: POST /api/orders
    enabled: false
    status:
      code: 503
      probability: 0.3
      retry_after: 2
  - name: hang-profile
    match: GET /api/profile
    hang:
      probability: 0.1
  - name: broken-json
    match: GET /api/user/*
    truncate:
      probability: 0.1
      at: 0.5
`)

	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if got.Target != "http://localhost:9000" {
		t.Errorf("Target = %q, want http://localhost:9000", got.Target)
	}
	if got.Seed != 42 {
		t.Errorf("Seed = %d, want 42", got.Seed)
	}
	if got.CORS != config.CORSReflect {
		t.Errorf("CORS = %q, want reflect default", got.CORS)
	}
	if len(got.Rules) != 4 {
		t.Fatalf("len(Rules) = %d, want 4", len(got.Rules))
	}
	if !got.Rules[0].Enabled {
		t.Error("omitted enabled field should default to true")
	}
	if got.Rules[1].Enabled {
		t.Error("explicit enabled: false should be preserved")
	}
	if got.Rules[0].Latency.P50 != 400*time.Millisecond || got.Rules[0].Latency.P99 != 3*time.Second {
		t.Errorf("latency durations = (%s, %s), want (400ms, 3s)", got.Rules[0].Latency.P50, got.Rules[0].Latency.P99)
	}
	if got.Rules[1].Status.RetryAfter != 2 {
		t.Errorf("RetryAfter = %d, want 2", got.Rules[1].Status.RetryAfter)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}

func TestLoadRejectsMalformedDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		contents  string
		wantError string
	}{
		{name: "empty", wantError: "configuration is empty"},
		{name: "unknown field", contents: "target: http://localhost\nunknown: true\n", wantError: "field unknown not found"},
		{name: "invalid duration", contents: "target: http://localhost\nrules:\n  - name: slow\n    match: /api\n    latency:\n      dist: fixed\n      value: eventually\n", wantError: "cannot unmarshal"},
		{name: "multiple documents", contents: "target: http://one\n---\ntarget: http://two\n", wantError: "exactly one YAML document"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.Load(writeConfig(t, test.contents))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Load() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestLoadWrapsFileErrors(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.yaml")
	_, err := config.Load(path)
	if err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("Load() error = %v, want wrapped read error", err)
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "chaos.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
