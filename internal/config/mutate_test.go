package config_test

import (
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
)

func TestLoadMutateDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: broken-user
    match: GET /api/user
    mutate:
      probability: 0.5
      operations:
        - op: nullify
          path: user.email
        - op: inflate
          path: items
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
	mutate := cfg.Rules[0].Mutate
	if mutate.MaxBytes != 1<<20 || mutate.Operations[0].Factor != 0 || mutate.Operations[1].Factor != 10 {
		t.Fatalf("mutate = %+v, want 1 MiB limit and factor 10 for inflate only", mutate)
	}

	cloned := cfg.Clone()
	cloned.Rules[0].Mutate.Operations[0].Path = "changed"
	if mutate.Operations[0].Path != "user.email" {
		t.Error("mutating a cloned operation changed the original")
	}
}

func TestValidateMutate(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(writeConfig(t, `
target: http://localhost:9000
rules:
  - name: broken-user
    match: GET /api/user
    mutate:
      probability: 2
      max_bytes: -1
      operations:
        - op: explode
          path: user
        - op: nullify
          path: user..email
          factor: 3
        - op: stretch
          path: user.name
          factor: 5000
  - name: nothing
    match: GET /nothing
    mutate:
      probability: 1
      operations: []
`))
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want mutate errors")
	}
	for _, expected := range []string{
		"rules[0].mutate.probability must be between 0 and 1",
		"rules[0].mutate.max_bytes must be greater than zero",
		"line 9: rules[0].mutate.operations[0].op must be nullify, empty, inflate, stretch, or drop",
		"rules[0].mutate.operations[1].path must be a dotted path",
		"rules[0].mutate.operations[1].factor only applies to inflate and stretch",
		"rules[0].mutate.operations[2].factor must be between 2 and 1000",
		"rules[1].mutate.operations must contain at least one operation",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Validate() error does not contain %q:\n%s", expected, err)
		}
	}
}
