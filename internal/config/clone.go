package config

// Clone returns a deep copy that callers may safely modify before publishing as
// a new immutable runtime configuration.
func (configured *Config) Clone() *Config {
	if configured == nil {
		return nil
	}

	cloned := *configured
	cloned.source = cloneSource(configured.source)
	cloned.Rules = make([]Rule, len(configured.Rules))
	for index, rule := range configured.Rules {
		cloned.Rules[index] = cloneRule(rule)
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
