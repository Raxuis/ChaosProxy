package config

import (
	"maps"
	"slices"
)

// Clone returns a deep copy that callers may safely modify before publishing as
// a new immutable runtime configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	cloned := *c
	cloned.source = cloneSource(c.source)
	cloned.CORSOrigins = slices.Clone(c.CORSOrigins)
	cloned.Rules = make([]Rule, len(c.Rules))
	for index, rule := range c.Rules {
		cloned.Rules[index] = cloneRule(rule)
	}
	if c.Scenarios != nil {
		cloned.Scenarios = make([]Scenario, len(c.Scenarios))
		for index, scenario := range c.Scenarios {
			scenario.Steps = slices.Clone(scenario.Steps)
			scenario.source = cloneSource(scenario.source)
			cloned.Scenarios[index] = scenario
		}
	}
	return &cloned
}

func cloneRule(rule Rule) Rule {
	cloned := rule
	cloned.source = cloneSource(rule.source)
	if rule.Latency != nil {
		value := *rule.Latency
		cloned.Latency = &value
	}
	if rule.Status != nil {
		value := *rule.Status
		cloned.Status = &value
	}
	if rule.Hang != nil {
		value := *rule.Hang
		cloned.Hang = &value
	}
	if rule.Truncate != nil {
		value := *rule.Truncate
		cloned.Truncate = &value
	}
	if rule.Reset != nil {
		value := *rule.Reset
		cloned.Reset = &value
	}
	if rule.Bandwidth != nil {
		value := *rule.Bandwidth
		cloned.Bandwidth = &value
	}
	if rule.Headers != nil {
		value := *rule.Headers
		value.Set = maps.Clone(rule.Headers.Set)
		value.Remove = slices.Clone(rule.Headers.Remove)
		cloned.Headers = &value
	}
	if rule.Mutate != nil {
		value := *rule.Mutate
		value.Operations = slices.Clone(rule.Mutate.Operations)
		cloned.Mutate = &value
	}
	if rule.Redirect != nil {
		value := *rule.Redirect
		cloned.Redirect = &value
	}
	return cloned
}

func cloneSource(source sourceLocation) sourceLocation {
	cloned := sourceLocation{line: source.line}
	if source.fields != nil {
		cloned.fields = make(map[string]int, len(source.fields))
		for field, line := range source.fields {
			cloned.fields[field] = line
		}
	}
	return cloned
}
