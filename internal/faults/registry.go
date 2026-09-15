package faults

import "github.com/Raxuis/chaosproxy/internal/config"

// Build creates the fault chain of a rule that passed config.Validate.
func Build(rule config.Rule) []Fault {
	chain := make([]Fault, 0, 4)
	if rule.Latency != nil {
		chain = append(chain, newLatencyFault(*rule.Latency))
	}
	if rule.Status != nil {
		chain = append(chain, newStatusFault(*rule.Status))
	}
	if rule.Hang != nil {
		chain = append(chain, newHangFault(*rule.Hang))
	}
	if rule.Truncate != nil {
		chain = append(chain, newTruncateFault(*rule.Truncate))
	}
	return chain
}
