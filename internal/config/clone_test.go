package config_test

import (
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestConfigCloneOwnsNestedValues(t *testing.T) {
	t.Parallel()

	original := &config.Config{
		Target:      "http://localhost",
		CORSOrigins: []string{"http://localhost:3001"},
		Rules: []config.Rule{{
			Name:      "fault",
			Match:     "GET /api",
			Enabled:   true,
			Status:    &config.StatusConfig{Code: 503, Probability: 1},
			Reset:     &config.ResetConfig{Probability: 0.5},
			Bandwidth: &config.BandwidthConfig{BytesPerSecond: 1024},
		}},
	}
	cloned := original.Clone()
	cloned.Rules[0].Reset.Probability = 1
	cloned.Rules[0].Bandwidth.BytesPerSecond = 1
	if original.Rules[0].Reset.Probability != 0.5 || original.Rules[0].Bandwidth.BytesPerSecond != 1024 {
		t.Fatalf("mutating clone changed original reset or bandwidth: %+v", original.Rules[0])
	}
	cloned.Target = "http://other"
	cloned.CORSOrigins[0] = "https://changed.test"
	cloned.Rules[0].Name = "changed"
	cloned.Rules[0].Status.Code = 418

	if original.Target != "http://localhost" || original.CORSOrigins[0] != "http://localhost:3001" ||
		original.Rules[0].Name != "fault" || original.Rules[0].Status.Code != 503 {
		t.Fatalf("mutating clone changed original: %+v", original)
	}
}
