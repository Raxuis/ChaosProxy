package config_test

import (
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLoadHeaders(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: modify-headers
    match: GET /api/data
    headers:
      probability: 0.8
      set:
        X-Custom-Header: custom-value
        Cache-Control: no-store
      remove:
        - ETag
        - X-Powered-By
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	headers := cfg.Rules[0].Headers
	if headers == nil {
		t.Fatal("headers is nil, want parsed headers config")
	}
	if headers.Probability != 0.8 {
		t.Errorf("headers.Probability = %v, want 0.8", headers.Probability)
	}
	if len(headers.Set) != 2 || headers.Set["X-Custom-Header"] != "custom-value" || headers.Set["Cache-Control"] != "no-store" {
		t.Errorf("headers.Set = %+v, want configured set map", headers.Set)
	}
	if len(headers.Remove) != 2 || headers.Remove[0] != "ETag" || headers.Remove[1] != "X-Powered-By" {
		t.Errorf("headers.Remove = %+v, want configured remove slice", headers.Remove)
	}

	cloned := cfg.Clone()
	cloned.Rules[0].Headers.Set["X-Custom-Header"] = "modified"
	cloned.Rules[0].Headers.Remove[0] = "Modified"
	if headers.Set["X-Custom-Header"] != "custom-value" {
		t.Error("mutating cloned headers set map changed original")
	}
	if headers.Remove[0] != "ETag" {
		t.Error("mutating cloned headers remove slice changed original")
	}
}

func TestValidateHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yaml      string
		wantError string
	}{
		{
			name: "empty set and remove",
			yaml: `
target: http://localhost:9000
rules:
  - name: empty-headers
    match: GET /api
    headers:
      probability: 0.5
`,
			wantError: "rules[0].headers must specify at least one header to set or remove",
		},
		{
			name: "probability greater than 1",
			yaml: `
target: http://localhost:9000
rules:
  - name: prob-high
    match: GET /api
    headers:
      probability: 1.5
      remove:
        - X-Debug
`,
			wantError: "rules[0].headers.probability must be between 0 and 1",
		},
		{
			name: "probability negative",
			yaml: `
target: http://localhost:9000
rules:
  - name: prob-neg
    match: GET /api
    headers:
      probability: -0.1
      remove:
        - X-Debug
`,
			wantError: "rules[0].headers.probability must be between 0 and 1",
		},
		{
			name: "probability NaN",
			yaml: `
target: http://localhost:9000
rules:
  - name: prob-nan
    match: GET /api
    headers:
      probability: .nan
      remove:
        - X-Debug
`,
			wantError: "rules[0].headers.probability must be between 0 and 1",
		},
		{
			name: "empty header name in remove",
			yaml: `
target: http://localhost:9000
rules:
  - name: empty-remove
    match: GET /api
    headers:
      probability: 1
      remove:
        - ""
`,
			wantError: "rules[0].headers.remove[0] header name must not be empty",
		},
		{
			name: "invalid header name characters in set",
			yaml: `
target: http://localhost:9000
rules:
  - name: invalid-set
    match: GET /api
    headers:
      probability: 1
      set:
        "bad header": value
`,
			wantError: `rules[0].headers.set.bad header invalid header name "bad header"`,
		},
		{
			name: "invalid header name characters in remove",
			yaml: `
target: http://localhost:9000
rules:
  - name: invalid-remove
    match: GET /api
    headers:
      probability: 1
      remove:
        - "header:colon"
`,
			wantError: `rules[0].headers.remove[0] invalid header name "header:colon"`,
		},
		{
			name: "managed header Content-Length in set",
			yaml: `
target: http://localhost:9000
rules:
  - name: managed-set
    match: GET /api
    headers:
      probability: 1
      set:
        Content-Length: "100"
`,
			wantError: `"Content-Length" is managed by the proxy and cannot be modified`,
		},
		{
			name: "managed header Transfer-Encoding in remove (case-insensitive)",
			yaml: `
target: http://localhost:9000
rules:
  - name: managed-remove
    match: GET /api
    headers:
      probability: 1
      remove:
        - transfer-encoding
`,
			wantError: `"transfer-encoding" is managed by the proxy and cannot be modified`,
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
				t.Fatalf("Validate() expected error containing %q, got nil", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Validate() error %q does not contain %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestValidateHeadersValidVariants(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: only-set
    match: GET /set
    headers:
      probability: 1
      set:
        X-Test: "1"
  - name: only-remove
    match: GET /remove
    headers:
      probability: 0
      remove:
        - ETag
  - name: both
    match: GET /both
    headers:
      probability: 0.5
      set:
        Cache-Control: "no-cache"
      remove:
        - Server
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}
