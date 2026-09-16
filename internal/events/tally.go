package events

import (
	"maps"
	"strconv"
)

// Tally counts requests, injected faults, status classes, and per-rule totals.
// It is not safe for concurrent use.
type Tally struct {
	requests uint64
	faulted  uint64
	statuses map[string]uint64
	rules    map[string]*RuleStats
}

// NewTally creates an empty tally.
func NewTally() *Tally {
	return &Tally{statuses: make(map[string]uint64), rules: make(map[string]*RuleStats)}
}

// Add counts one completed request.
func (t *Tally) Add(event Event) {
	t.requests++
	t.statuses[StatusClass(event.Status)]++
	if len(event.Faults) > 0 {
		t.faulted++
	}
	if event.Rule == "" {
		return
	}

	rule, found := t.rules[event.Rule]
	if !found {
		rule = &RuleStats{Faults: make(map[string]uint64)}
		t.rules[event.Rule] = rule
	}
	rule.Matched++
	if len(event.Faults) > 0 {
		rule.Faulted++
	}
	for _, fault := range event.Faults {
		rule.Faults[fault]++
	}
}

// Requests returns how many requests were counted.
func (t *Tally) Requests() uint64 {
	return t.requests
}

// Faulted returns how many requests received at least one fault.
func (t *Tally) Faulted() uint64 {
	return t.faulted
}

// Statuses returns a copy of the counts per status class.
func (t *Tally) Statuses() map[string]uint64 {
	return maps.Clone(t.statuses)
}

// Rules returns a copy of the per-rule totals.
func (t *Tally) Rules() map[string]RuleStats {
	rules := make(map[string]RuleStats, len(t.rules))
	for name, rule := range t.rules {
		rules[name] = RuleStats{Matched: rule.Matched, Faulted: rule.Faulted, Faults: maps.Clone(rule.Faults)}
	}
	return rules
}

// StatusClass groups a status as 1xx through 5xx, or aborted when no response started.
func StatusClass(status int) string {
	if status < 100 || status > 599 {
		return "aborted"
	}
	return strconv.Itoa(status/100) + "xx"
}
