package faults

import (
	"errors"
	"fmt"

	"github.com/Raxuis/chaosproxy/internal/config"
)

// Build creates a rule's fault chain in deterministic pipeline order.
func Build(rule config.Rule) ([]Fault, error) {
	chain := make([]Fault, 0, 4)

	if rule.Latency != nil {
		fault, err := newLatencyFault(*rule.Latency)
		if err != nil {
			return nil, fmt.Errorf("build latency fault for rule %q: %w", rule.Name, err)
		}
		chain = append(chain, fault)
	}
	if rule.Status != nil {
		fault, err := newStatusFault(*rule.Status)
		if err != nil {
			return nil, fmt.Errorf("build status fault for rule %q: %w", rule.Name, err)
		}
		chain = append(chain, fault)
	}
	if rule.Hang != nil {
		fault, err := newHangFault(*rule.Hang)
		if err != nil {
			return nil, fmt.Errorf("build hang fault for rule %q: %w", rule.Name, err)
		}
		chain = append(chain, fault)
	}
	if rule.Truncate != nil {
		fault, err := newTruncateFault(*rule.Truncate)
		if err != nil {
			return nil, fmt.Errorf("build truncate fault for rule %q: %w", rule.Name, err)
		}
		chain = append(chain, fault)
	}

	if len(chain) == 0 {
		return nil, errors.New("rule must configure at least one fault")
	}
	return chain, nil
}
