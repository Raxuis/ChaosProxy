package config_test

import (
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLoadRedirect(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: session-expired
    match: GET /api/user
    redirect:
      probability: 0.9
      code: 302
      location: /login
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	redirect := cfg.Rules[0].Redirect
	if redirect == nil {
		t.Fatal("redirect is nil, want parsed redirect config")
	}
	if redirect.Probability != 0.9 {
		t.Errorf("redirect.Probability = %v, want 0.9", redirect.Probability)
	}
	if redirect.Code != 302 {
		t.Errorf("redirect.Code = %d, want 302", redirect.Code)
	}
	if redirect.Location != "/login" {
		t.Errorf("redirect.Location = %q, want /login", redirect.Location)
	}

	cloned := cfg.Clone()
	cloned.Rules[0].Redirect.Location = "/other"
	if redirect.Location != "/login" {
		t.Error("mutating cloned redirect location changed original")
	}
}

func TestValidateRedirect(t *testing.T) {
	t.Parallel()

	validCodes := []int{301, 302, 303, 307, 308}
	for _, code := range validCodes {
		cfg := &config.Config{
			Target: "http://localhost:9000",
			CORS:   config.CORSReflect,
			Rules: []config.Rule{{
				Name:    "redirect",
				Match:   "/api",
				Enabled: true,
				Redirect: &config.RedirectConfig{
					Code:        code,
					Location:    "/target",
					Probability: 1,
				},
			}},
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with valid code %d unexpected error: %v", code, err)
		}
	}

	tests := []struct {
		name      string
		yaml      string
		wantError string
	}{
		{
			name: "invalid status code with line number",
			yaml: `
target: http://localhost:9000
rules:
  - name: bad-code
    match: GET /api
    redirect:
      probability: 1
      code: 200
      location: /login
`,
			wantError: "line 7: rules[0].redirect.code must be 301, 302, 303, 307, or 308",
		},
		{
			name: "404 status code with line number",
			yaml: `
target: http://localhost:9000
rules:
  - name: bad-code-404
    match: GET /api
    redirect:
      code: 404
      location: /login
`,
			wantError: "line 6: rules[0].redirect.code must be 301, 302, 303, 307, or 308",
		},
		{
			name: "empty location with line number",
			yaml: `
target: http://localhost:9000
rules:
  - name: empty-loc
    match: GET /api
    redirect:
      code: 302
      location: ""
`,
			wantError: "line 7: rules[0].redirect.location must not be empty",
		},
		{
			name: "whitespace location with line number",
			yaml: `
target: http://localhost:9000
rules:
  - name: whitespace-loc
    match: GET /api
    redirect:
      code: 302
      location: "   "
`,
			wantError: "line 7: rules[0].redirect.location must not be empty",
		},
		{
			name: "probability greater than 1",
			yaml: `
target: http://localhost:9000
rules:
  - name: prob-high
    match: GET /api
    redirect:
      probability: 1.5
      code: 302
      location: /login
`,
			wantError: "line 6: rules[0].redirect.probability must be between 0 and 1",
		},
		{
			name: "probability negative",
			yaml: `
target: http://localhost:9000
rules:
  - name: prob-neg
    match: GET /api
    redirect:
      probability: -0.1
      code: 302
      location: /login
`,
			wantError: "line 6: rules[0].redirect.probability must be between 0 and 1",
		},
		{
			name: "probability NaN",
			yaml: `
target: http://localhost:9000
rules:
  - name: prob-nan
    match: GET /api
    redirect:
      probability: .nan
      code: 302
      location: /login
`,
			wantError: "line 6: rules[0].redirect.probability must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.Load(writeConfig(t, tt.yaml))
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			err = cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Validate() error does not contain %q:\n%s", tt.wantError, err)
			}
		})
	}
}
