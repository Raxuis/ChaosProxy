package config

import "reflect"

type changeSummary struct {
	added         []string
	removed       []string
	modified      []string
	targetChanged bool
	seedChanged   bool
	corsChanged   bool
}

func summarizeChanges(previous, next *Config) changeSummary {
	summary := changeSummary{
		targetChanged: previous.Target != next.Target,
		seedChanged:   previous.Seed != next.Seed,
		corsChanged:   previous.CORS != next.CORS,
	}

	previousRules := make(map[string]Rule, len(previous.Rules))
	previousIndexes := make(map[string]int, len(previous.Rules))
	for index, rule := range previous.Rules {
		previousRules[rule.Name] = rule
		previousIndexes[rule.Name] = index
	}

	nextRules := make(map[string]struct{}, len(next.Rules))
	for index, rule := range next.Rules {
		nextRules[rule.Name] = struct{}{}
		old, exists := previousRules[rule.Name]
		if !exists {
			summary.added = append(summary.added, rule.Name)
			continue
		}
		if previousIndexes[rule.Name] != index || !sameRule(old, rule) {
			summary.modified = append(summary.modified, rule.Name)
		}
	}

	for _, rule := range previous.Rules {
		if _, exists := nextRules[rule.Name]; !exists {
			summary.removed = append(summary.removed, rule.Name)
		}
	}
	return summary
}

func sameRule(left, right Rule) bool {
	left.source = sourceLocation{}
	right.source = sourceLocation{}
	return reflect.DeepEqual(left, right)
}
