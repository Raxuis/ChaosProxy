package config_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLoadStall(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: stall-download
    match: GET /download/*
    stall:
      probability: 0.2
      after_bytes: 4096
      duration: 5s
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
	stall := cfg.Rules[0].Stall
	if stall == nil {
		t.Fatal("stall is nil, want parsed stall config")
	}
	if stall.Probability != 0.2 || stall.AfterBytes != 4096 || stall.Duration != 5*time.Second {
		t.Fatalf("stall = %+v, want probability=0.2, after_bytes=4096, duration=5s", stall)
	}

	cloned := cfg.Clone()
	cloned.Rules[0].Stall.Probability = 0.8
	cloned.Rules[0].Stall.AfterBytes = 1024
	cloned.Rules[0].Stall.Duration = time.Second
	if stall.Probability != 0.2 || stall.AfterBytes != 4096 || stall.Duration != 5*time.Second {
		t.Errorf("mutating clone changed original stall: %+v", stall)
	}
}

func TestStallJSON(t *testing.T) {
	t.Parallel()

	original := &config.StallConfig{
		Probability: 0.5,
		AfterBytes:  2048,
		Duration:    3 * time.Second,
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}
	if !strings.Contains(string(encoded), `"duration":"3s"`) {
		t.Errorf("JSON does not contain expected duration notation: %s", encoded)
	}

	var decoded config.StallConfig
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() unexpected error: %v", err)
	}
	if !reflect.DeepEqual(decoded, *original) {
		t.Fatalf("decoded = %+v, want %+v", decoded, *original)
	}

	for _, invalid := range []string{
		`{"probability":0.5,"after_bytes":100,"duration":"invalid"}`,
		`{"probability":0.5,"after_bytes":100,"unknown":123}`,
	} {
		var stall config.StallConfig
		if err := json.Unmarshal([]byte(invalid), &stall); err == nil {
			t.Errorf("json.Unmarshal(%s) expected error, got nil", invalid)
		}
	}
}

func TestValidateStall(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: invalid-stall
    match: GET /stream
    stall:
      probability: 1.5
      after_bytes: -10
      duration: 0s
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want stall errors")
	}
	for _, expected := range []string{
		"rules[0].stall.probability must be between 0 and 1",
		"rules[0].stall.after_bytes must not be negative",
		"rules[0].stall.duration must be greater than zero",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate() error does not contain %q:\n%s", expected, err)
		}
	}
}
