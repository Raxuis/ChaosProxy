package config_test

import (
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestConfigCloneOwnsNestedValues(t *testing.T) {
	t.Parallel()

	original := &config.Config{
		Target: "http://localhost",
		Rules: []config.Rule{{
			Name:    "fault",
			Match:   "GET /api",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 1},
		}},
	}
	cloned := original.Clone()
	cloned.Target = "http://other"
	cloned.Rules[0].Name = "changed"
	cloned.Rules[0].Status.Code = 418

	if original.Target != "http://localhost" || original.Rules[0].Name != "fault" || original.Rules[0].Status.Code != 503 {
		t.Fatalf("mutating clone changed original: %+v", original)
	}
}
