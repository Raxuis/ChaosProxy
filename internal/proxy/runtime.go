package proxy

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/faults"
	"github.com/Raxuis/chaosproxy/internal/rules"
)

type runtimeConfig struct {
	base      *config.Config
	overrides map[string]bool
	effective *config.Config
	target    *url.URL
	seed      int64
	cors      corsPolicy
	match     *rules.Matcher
	chains    map[string][]faults.Fault

	scenarioMatch *rules.Matcher
	scenarios     map[string]*compiledScenario
	scenarioOrder []*compiledScenario
}

func compileRuntime(base *config.Config, overrides map[string]bool) (*runtimeConfig, error) {
	if base == nil {
		return nil, errors.New("configuration must not be nil")
	}
	base = base.Clone()
	cfg := base.Clone()
	for index := range cfg.Rules {
		if enabled, overridden := overrides[cfg.Rules[index].Name]; overridden {
			cfg.Rules[index].Enabled = enabled
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate configuration: %w", err)
	}

	target, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("parse target: %w", err)
	}
	matcher, err := rules.Compile(cfg.Rules)
	if err != nil {
		return nil, fmt.Errorf("compile rules: %w", err)
	}

	chains := make(map[string][]faults.Fault, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		chains[rule.Name] = faults.Build(rule)
	}

	scenarioRules := make([]config.Rule, len(cfg.Scenarios))
	scenarios := make(map[string]*compiledScenario, len(cfg.Scenarios))
	scenarioOrder := make([]*compiledScenario, 0, len(cfg.Scenarios))
	for index, scenario := range cfg.Scenarios {
		compiled, err := compileScenario(scenario)
		if err != nil {
			return nil, fmt.Errorf("compile scenarios: %w", err)
		}
		scenarioRules[index] = config.Rule{Name: scenario.Name, Match: scenario.Match, Enabled: scenario.Enabled}
		scenarios[scenario.Name] = compiled
		scenarioOrder = append(scenarioOrder, compiled)
	}
	scenarioMatch, err := rules.Compile(scenarioRules)
	if err != nil {
		return nil, fmt.Errorf("compile scenarios: %w", err)
	}

	return &runtimeConfig{
		base:          base,
		overrides:     overrides,
		effective:     cfg,
		target:        target,
		seed:          cfg.Seed,
		cors:          newCORSPolicy(cfg.CORS, cfg.CORSOrigins),
		match:         matcher,
		chains:        chains,
		scenarioMatch: scenarioMatch,
		scenarios:     scenarios,
		scenarioOrder: scenarioOrder,
	}, nil
}
