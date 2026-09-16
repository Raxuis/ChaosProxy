package config_test

import (
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLoadScenariosWithDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
scenarios:
  - name: checkout-recovers
    match: POST /api/checkout
    steps:
      - status=503
      - latency=2s; status=502
      - off
  - name: flaky-login
    match: POST /api/login
    enabled: false
    on_exhausted: repeat
    steps: [status=401]
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	first, second := cfg.Scenarios[0], cfg.Scenarios[1]
	if !first.Enabled || first.OnExhausted != config.ExhaustPassthrough || len(first.Steps) != 3 || first.Steps[1] != "latency=2s; status=502" {
		t.Errorf("first scenario = %+v, want enabled passthrough with three steps", first)
	}
	if second.Enabled || second.OnExhausted != config.ExhaustRepeat || len(second.Steps) != 1 {
		t.Errorf("second scenario = %+v, want disabled repeat with one step", second)
	}

	cloned := cfg.Clone()
	cloned.Scenarios[0].Steps[0] = "off"
	if cfg.Scenarios[0].Steps[0] != "status=503" {
		t.Error("mutating a cloned scenario changed the original steps")
	}
}

func TestValidateScenarios(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: taken
    match: GET /api
    hang:
      probability: 0.1
scenarios:
  - name: taken
    match: ""
    on_exhausted: forever
    steps:
      - status=503
      - teapot
  - name: empty
    match: GET /empty
    steps: []
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want scenario errors")
	}
	for _, expected := range []string{
		`scenarios[0].name must be unique across rules and scenarios; "taken" is already used`,
		"scenarios[0].match must not be empty",
		"scenarios[0].on_exhausted must be passthrough, repeat, or last",
		`line 13: scenarios[0].steps[1] is invalid: unknown fault "teapot"`,
		"scenarios[1].steps must contain at least one step",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate() error does not contain %q:\n%s", expected, err)
		}
	}
}
