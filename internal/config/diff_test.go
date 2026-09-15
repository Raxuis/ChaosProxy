package config

import (
	"reflect"
	"testing"
	"time"
)

func TestSummarizeChanges(t *testing.T) {
	t.Parallel()

	previous := &Config{
		Target: "http://old.example",
		Seed:   1,
		CORS:   CORSReflect,
		Rules: []Rule{
			{Name: "removed", Match: "GET /removed", Enabled: true, Hang: &HangConfig{Probability: 1}},
			{Name: "changed", Match: "GET /changed", Enabled: true, Latency: &LatencyConfig{Dist: "fixed", Value: time.Second}},
			{Name: "reordered", Match: "GET /same", Enabled: true, Hang: &HangConfig{Probability: 1}},
		},
	}
	next := &Config{
		Target: "http://new.example",
		Seed:   2,
		CORS:   CORSOff,
		Rules: []Rule{
			{Name: "reordered", Match: "GET /same", Enabled: true, Hang: &HangConfig{Probability: 1}},
			{Name: "changed", Match: "GET /changed", Enabled: true, Latency: &LatencyConfig{Dist: "fixed", Value: 2 * time.Second}},
			{Name: "added", Match: "GET /added", Enabled: true, Hang: &HangConfig{Probability: 1}},
		},
	}

	got := summarizeChanges(previous, next)
	if !reflect.DeepEqual(got.added, []string{"added"}) {
		t.Errorf("added = %v, want [added]", got.added)
	}
	if !reflect.DeepEqual(got.removed, []string{"removed"}) {
		t.Errorf("removed = %v, want [removed]", got.removed)
	}
	if !reflect.DeepEqual(got.modified, []string{"reordered", "changed"}) {
		t.Errorf("modified = %v, want [reordered changed]", got.modified)
	}
	if !got.targetChanged || !got.seedChanged || !got.corsChanged {
		t.Errorf("top-level changes = %+v, want all true", got)
	}
}

func TestSummarizeChangesDetectsCORSOrigins(t *testing.T) {
	t.Parallel()

	previous := &Config{CORS: CORSReflect, CORSOrigins: []string{"http://localhost:3001"}}
	next := &Config{CORS: CORSReflect, CORSOrigins: []string{"https://app.test"}}
	if !summarizeChanges(previous, next).corsChanged {
		t.Fatal("corsChanged = false, want true when cors_origins changes")
	}
}

func TestSameRuleIgnoresYAMLSourceLocations(t *testing.T) {
	t.Parallel()

	left := Rule{Name: "same", Match: "GET /api", Enabled: true, Hang: &HangConfig{Probability: 1}}
	left.source = sourceLocation{line: 2, fields: map[string]int{"name": 2}}
	right := left
	right.source = sourceLocation{line: 20, fields: map[string]int{"name": 20}}

	if !sameRule(left, right) {
		t.Fatal("sameRule() treated source line movement as a rule modification")
	}
}
