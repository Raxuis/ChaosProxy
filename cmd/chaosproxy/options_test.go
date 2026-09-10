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
		name       string
		args       []string
		wantTarget string
		wantConfig string
		wantPort   int
		wantSeed   int64
		seedSet    bool
		wantError  string
	}{
		{name: "target only", args: []string{"--target", "http://localhost:9000"}, wantTarget: "http://localhost:9000", wantPort: 7070},
		{name: "config only", args: []string{"--config", "chaos.yaml"}, wantConfig: "chaos.yaml", wantPort: 7070},
		{name: "all overrides", args: []string{"--config", "chaos.yaml", "--target", "https://api.example.com", "--port", "9090", "--seed", "0"}, wantConfig: "chaos.yaml", wantTarget: "https://api.example.com", wantPort: 9090, wantSeed: 0, seedSet: true},
		{name: "missing source", wantError: "either --config or --target is required"},
		{name: "positional argument", args: []string{"--target", "http://localhost", "extra"}, wantError: "unexpected positional arguments"},
		{name: "zero port", args: []string{"--target", "http://localhost", "--port", "0"}, wantError: "--port must be between"},
		{name: "large port", args: []string{"--target", "http://localhost", "--port", "65536"}, wantError: "--port must be between"},
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
			if got.target != test.wantTarget || got.configPath != test.wantConfig || got.port != test.wantPort {
				t.Errorf("options = %+v, want target=%q config=%q port=%d", got, test.wantTarget, test.wantConfig, test.wantPort)
			}
			if got.seed.value != test.wantSeed || got.seed.set != test.seedSet {
				t.Errorf("seed = %+v, want value=%d set=%t", got.seed, test.wantSeed, test.seedSet)
			}
		})
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

	configured, err := resolveConfig(options{
		configPath: path,
		target:     "http://override:9001",
		seed:       optionalInt64{value: 42, set: true},
	})
	if err != nil {
		t.Fatalf("resolveConfig() unexpected error: %v", err)
	}
	if configured.Target != "http://override:9001" || configured.Seed != 42 || configured.CORS != config.CORSReflect {
		t.Fatalf("resolved config = %+v, want target and seed overrides with reflect CORS", configured)
	}
}

func TestResolveConfigWithoutFileIsPurePassthrough(t *testing.T) {
	t.Parallel()

	configured, err := resolveConfig(options{target: "http://localhost:9000"})
	if err != nil {
		t.Fatalf("resolveConfig() unexpected error: %v", err)
	}
	if configured.CORS != config.CORSPassthrough || len(configured.Rules) != 0 {
		t.Fatalf("resolved config = %+v, want rule-free passthrough", configured)
	}
}
