package config_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLatencyJSONUsesDurationStrings(t *testing.T) {
	t.Parallel()

	original := &config.Config{
		Target: "http://localhost:9000",
		CORS:   config.CORSReflect,
		Rules: []config.Rule{
			{
				Name:    "fixed",
				Match:   "GET /search",
				Enabled: true,
				Latency: &config.LatencyConfig{Dist: "fixed", Value: 800 * time.Millisecond, Jitter: 200 * time.Millisecond},
			},
			{
				Name:    "lognormal",
				Match:   "GET /orders",
				Enabled: true,
				Latency: &config.LatencyConfig{Dist: "lognormal", P50: 400 * time.Millisecond, P99: 90 * time.Second},
			},
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}
	for _, expected := range []string{
		`"latency":{"dist":"fixed","value":"800ms","jitter":"200ms"}`,
		`"latency":{"dist":"lognormal","p50":"400ms","p99":"1m30s"}`,
	} {
		if !strings.Contains(string(encoded), expected) {
			t.Errorf("JSON does not contain %s:\n%s", expected, encoded)
		}
	}

	var decoded config.Config
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Rules, original.Rules) {
		t.Fatalf("round trip rules = %+v, want %+v", decoded.Rules, original.Rules)
	}
}

func TestLatencyJSONRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		`{"dist":"fixed","value":"soon"}`:  "latency.value",
		`{"dist":"fixed","value":800}`:     "cannot unmarshal number",
		`{"dist":"fixed","delay":"800ms"}`: "unknown field",
	}
	for input, want := range tests {
		var latency config.LatencyConfig
		err := json.Unmarshal([]byte(input), &latency)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("json.Unmarshal(%s) error = %v, want error containing %q", input, err, want)
		}
	}
}
