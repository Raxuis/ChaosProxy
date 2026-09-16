package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		wantTarget      string
		wantConfig      string
		wantHost        string
		wantPort        int
		wantControlPort int
		wantSeed        int64
		seedSet         bool
		wantError       string
	}{
		{name: "target only", args: []string{"--target", "http://localhost:9000"}, wantTarget: "http://localhost:9000", wantPort: 7070, wantControlPort: 7071},
		{name: "config only", args: []string{"--config", "chaos.yaml"}, wantConfig: "chaos.yaml", wantPort: 7070, wantControlPort: 7071},
		{name: "all overrides", args: []string{"--config", "chaos.yaml", "--target", "https://api.example.com", "--port", "9090", "--control-port", "9091", "--seed", "0"}, wantConfig: "chaos.yaml", wantTarget: "https://api.example.com", wantPort: 9090, wantControlPort: 9091, wantSeed: 0, seedSet: true},
		{name: "exposed host", args: []string{"--target", "http://localhost:9000", "--host", "0.0.0.0"}, wantTarget: "http://localhost:9000", wantHost: "0.0.0.0", wantPort: 7070, wantControlPort: 7071},
		{name: "empty host", args: []string{"--target", "http://localhost", "--host", ""}, wantError: "--host must not be empty"},
		{name: "missing source", wantError: "either --config or --target is required"},
		{name: "positional argument", args: []string{"--target", "http://localhost", "extra"}, wantError: "unexpected positional arguments"},
		{name: "zero port", args: []string{"--target", "http://localhost", "--port", "0"}, wantError: "--port must be between"},
		{name: "large port", args: []string{"--target", "http://localhost", "--port", "65536"}, wantError: "--port must be between"},
		{name: "zero control port", args: []string{"--target", "http://localhost", "--control-port", "0"}, wantError: "--control-port must be between"},
		{name: "port collision", args: []string{"--target", "http://localhost", "--port", "7070", "--control-port", "7070"}, wantError: "must be different"},
		{name: "invalid seed", args: []string{"--target", "http://localhost", "--seed", "random"}, wantError: "invalid value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseOptions(test.args, io.Discard)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("parseOptions() error = %v, want error containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions() unexpected error: %v", err)
			}
			if got.target != test.wantTarget || got.configPath != test.wantConfig || got.port != test.wantPort || got.controlPort != test.wantControlPort {
				t.Errorf("options = %+v, want target=%q config=%q ports=%d/%d", got, test.wantTarget, test.wantConfig, test.wantPort, test.wantControlPort)
			}
			wantHost := test.wantHost
			if wantHost == "" {
				wantHost = "127.0.0.1"
			}
			if got.host != wantHost {
				t.Errorf("host = %q, want %q", got.host, wantHost)
			}
			if got.seed.value != test.wantSeed || got.seed.set != test.seedSet {
				t.Errorf("seed = %+v, want value=%d set=%t", got.seed, test.wantSeed, test.seedSet)
			}
		})
	}
}

func TestParseOptionsHeaderOverrides(t *testing.T) {
	t.Parallel()

	disabled, err := parseOptions([]string{"--target", "http://localhost"}, io.Discard)
	if err != nil || disabled.headerOverrides {
		t.Fatalf("default headerOverrides = %t, %v; want false", disabled.headerOverrides, err)
	}
	enabled, err := parseOptions([]string{"--target", "http://localhost", "--header-overrides"}, io.Discard)
	if err != nil || !enabled.headerOverrides {
		t.Fatalf("--header-overrides = %t, %v; want true", enabled.headerOverrides, err)
	}
}

func TestResolveConfigSeedSources(t *testing.T) {
	t.Parallel()

	withoutSeed := filepath.Join(t.TempDir(), "chaos.yaml")
	if err := os.WriteFile(withoutSeed, []byte("target: http://configured:9000\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	zeroSeed := filepath.Join(t.TempDir(), "chaos.yaml")
	if err := os.WriteFile(zeroSeed, []byte("target: http://configured:9000\nseed: 0\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	generated := options{configPath: withoutSeed}
	cfg, err := resolveConfig(&generated)
	if err != nil {
		t.Fatalf("resolveConfig() unexpected error: %v", err)
	}
	if cfg.Seed != generated.generatedSeed {
		t.Errorf("seed = %d, want generated seed %d", cfg.Seed, generated.generatedSeed)
	}

	reloaded, err := config.Load(withoutSeed)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	applyOverrides(reloaded, generated)
	if reloaded.Seed != generated.generatedSeed {
		t.Errorf("reloaded seed = %d, want the same generated seed %d", reloaded.Seed, generated.generatedSeed)
	}

	explicitZero := options{configPath: zeroSeed}
	if cfg, err := resolveConfig(&explicitZero); err != nil || cfg.Seed != 0 {
		t.Errorf("explicit seed 0 = %d, %v; want 0", cfg.Seed, err)
	}

	flag := options{configPath: zeroSeed, seed: optionalInt64{value: 7, set: true}}
	if cfg, err := resolveConfig(&flag); err != nil || cfg.Seed != 7 {
		t.Errorf("--seed 7 = %d, %v; want 7", cfg.Seed, err)
	}
}

func TestParseOptionsHelp(t *testing.T) {
	t.Parallel()

	_, err := parseOptions([]string{"--help"}, io.Discard)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("parseOptions() error = %v, want flag.ErrHelp", err)
	}
}

func TestResolveConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "chaos.yaml")
	contents := "target: http://configured:9000\nseed: 12\nrules: []\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts := options{
		configPath: path,
		target:     "http://override:9001",
		seed:       optionalInt64{value: 42, set: true},
	}
	configured, err := resolveConfig(&opts)
	if err != nil {
		t.Fatalf("resolveConfig() unexpected error: %v", err)
	}
	if configured.Target != "http://override:9001" || configured.Seed != 42 || configured.CORS != config.CORSReflect {
		t.Fatalf("resolved config = %+v, want target and seed overrides with reflect CORS", configured)
	}
}

func TestResolveConfigWithoutFileIsPurePassthrough(t *testing.T) {
	t.Parallel()

	configured, err := resolveConfig(&options{target: "http://localhost:9000"})
	if err != nil {
		t.Fatalf("resolveConfig() unexpected error: %v", err)
	}
	if configured.CORS != config.CORSPassthrough || len(configured.Rules) != 0 {
		t.Fatalf("resolved config = %+v, want rule-free passthrough", configured)
	}
}
