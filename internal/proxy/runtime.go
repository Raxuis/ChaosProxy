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
	configured *config.Config
	target     *url.URL
	seed       int64
	cors       corsPolicy
	match      *rules.Matcher
	chains     map[string][]faults.Fault
}

func compileRuntime(configured *config.Config) (*runtimeConfig, error) {
	if configured == nil {
		return nil, errors.New("configuration must not be nil")
	}
	configured = configured.Clone()
	if err := configured.Validate(); err != nil {
		return nil, fmt.Errorf("validate configuration: %w", err)
	}

	target, err := url.Parse(configured.Target)
	if err != nil {
		return nil, fmt.Errorf("parse target: %w", err)
	}
	matcher, err := rules.Compile(configured.Rules)
	if err != nil {
		return nil, fmt.Errorf("compile rules: %w", err)
	}

	chains := make(map[string][]faults.Fault, len(configured.Rules))
	for _, rule := range configured.Rules {
		chain, err := faults.Build(rule)
		if err != nil {
			return nil, err
		}
		chains[rule.Name] = chain
	}

	corsMode := configured.CORS
	if corsMode == "" {
		corsMode = config.CORSReflect
	}
	return &runtimeConfig{
		configured: configured,
		target:     target,
		seed:       configured.Seed,
		cors:       newCORSPolicy(corsMode, configured.CORSOrigins),
		match:      matcher,
		chains:     chains,
	}, nil
}
