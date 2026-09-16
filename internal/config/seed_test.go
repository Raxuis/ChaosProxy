package config_test

import (
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestSeedConfiguredReflectsYAML(t *testing.T) {
	t.Parallel()

	withZeroSeed, err := config.Load(writeConfig(t, "target: http://localhost\nseed: 0\n"))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	withoutSeed, err := config.Load(writeConfig(t, "target: http://localhost\n"))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !withZeroSeed.SeedConfigured() || !withZeroSeed.Clone().SeedConfigured() {
		t.Error("SeedConfigured() = false for an explicit seed: 0")
	}
	if withoutSeed.SeedConfigured() || (&config.Config{Seed: 42}).SeedConfigured() {
		t.Error("SeedConfigured() = true without a seed key in YAML")
	}
}
