package config_test

import (
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestValidateCORSOrigins(t *testing.T) {
	t.Parallel()

	valid, err := config.Load(writeConfig(t, `
target: http://localhost:9000
cors_origins:
  - "*"
  - http://localhost:3001
  - https://app.test
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if len(valid.CORSOrigins) != 3 {
		t.Fatalf("CORSOrigins = %v, want 3 entries", valid.CORSOrigins)
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	invalid, err := config.Load(writeConfig(t, `
target: http://localhost:9000
cors_origins:
  - localhost:3001
  - http://localhost:3001/
  - ftp://files.test
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	err = invalid.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid origins")
	}
	for _, expected := range []string{"cors_origins[0] must be", "cors_origins[1] must be", "cors_origins[2] must be"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate() error does not contain %q:\n%s", expected, err)
		}
	}
}

func TestValidateReturnsAllErrorsWithSourceLines(t *testing.T) {
	t.Parallel()

	configured, err := config.Load(writeConfig(t, `
target: ://invalid
rules:
  - name: duplicate
    match: ""
    latency:
      dist: lognormal
      p50: 3s
      p99: 1s
    status:
      code: 700
      probability: 1.5
      retry_after: -1
    hang:
      probability: -0.1
    truncate:
      probability: 2
      at: 1.5
  - name: duplicate
    match: GET /api/orders
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	err = configured.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want aggregate error")
	}
	message := err.Error()
	for _, expected := range []string{
		"line 1: target",
		"rules[0].match must not be empty",
		"rules[0].latency must satisfy p50 <= p99",
		"rules[0].status.code must be between 100 and 599",
		"rules[0].status.probability must be between 0 and 1",
		"rules[0].status.retry_after must not be negative",
		"rules[0].hang.probability must be between 0 and 1",
		"rules[0].truncate.probability must be between 0 and 1",
		"rules[0].truncate.at must be between 0 and 1",
		"rules[1].name must be unique",
		"rules[1] must configure at least one fault",
	} {
		if !strings.Contains(message, expected) {
			t.Errorf("Validate() error does not contain %q:\n%s", expected, message)
		}
	}
}

func TestValidateRequiredFieldsAndLatencyModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured *config.Config
		wantError  string
	}{
		{
			name:       "nil config",
			configured: nil,
			wantError:  "must not be nil",
		},
		{
			name:       "missing fields and fault",
			configured: &config.Config{Rules: []config.Rule{{}}},
			wantError:  "target must not be empty",
		},
		{
			name: "unknown latency distribution",
			configured: &config.Config{
				Target: "http://localhost",
				Rules:  []config.Rule{{Name: "slow", Match: "/api", Enabled: true, Latency: &config.LatencyConfig{Dist: "normal"}}},
			},
			wantError: "must be either fixed or lognormal",
		},
		{
			name: "negative fixed duration",
			configured: &config.Config{
				Target: "http://localhost",
				Rules:  []config.Rule{{Name: "slow", Match: "/api", Enabled: true, Latency: &config.LatencyConfig{Dist: "fixed", Value: -1}}},
			},
			wantError: "latency.value must not be negative",
		},
		{
			name: "missing lognormal percentiles",
			configured: &config.Config{
				Target: "http://localhost",
				Rules:  []config.Rule{{Name: "slow", Match: "/api", Enabled: true, Latency: &config.LatencyConfig{Dist: "lognormal"}}},
			},
			wantError: "latency.p50 must be greater than zero",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.configured.Validate()
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Validate() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestValidateAllowsProbabilityBoundaries(t *testing.T) {
	t.Parallel()

	configured := &config.Config{
		Target: "https://api.example.com",
		Rules: []config.Rule{
			{
				Name:    "never",
				Match:   "/never",
				Enabled: true,
				Status:  &config.StatusConfig{Code: 503, Probability: 0},
			},
			{
				Name:     "always",
				Match:    "/always",
				Enabled:  true,
				Hang:     &config.HangConfig{Probability: 1},
				Truncate: &config.TruncateConfig{Probability: 1, At: 0},
			},
		},
	}

	if err := configured.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}

func TestValidateRejectsUnknownCORSMode(t *testing.T) {
	t.Parallel()

	configured := &config.Config{Target: "http://localhost", CORS: "mirror"}
	if err := configured.Validate(); err == nil || !strings.Contains(err.Error(), "cors must be") {
		t.Fatalf("Validate() error = %v, want CORS mode error", err)
	}
}

func TestValidateRejectsNaNProbabilitiesAndFractions(t *testing.T) {
	t.Parallel()

	configured, err := config.Load(writeConfig(t, `
target: http://localhost
rules:
  - name: invalid-numbers
    match: /api
    truncate:
      probability: .nan
      at: .nan
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	err = configured.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid numeric values")
	}
	for _, field := range []string{"truncate.probability", "truncate.at"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Validate() error does not contain %q: %v", field, err)
		}
	}
}
