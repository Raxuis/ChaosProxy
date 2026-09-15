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
			Name:    "fault",
			Match:   "GET /api",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 1},
		}},
	}
	cloned := original.Clone()
	cloned.Target = "http://other"
	cloned.CORSOrigins[0] = "https://changed.test"
	cloned.Rules[0].Name = "changed"
	cloned.Rules[0].Status.Code = 418

	if original.Target != "http://localhost" || original.CORSOrigins[0] != "http://localhost:3001" ||
		original.Rules[0].Name != "fault" || original.Rules[0].Status.Code != 503 {
		t.Fatalf("mutating clone changed original: %+v", original)
	}
}
