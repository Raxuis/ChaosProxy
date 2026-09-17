package faults

import "github.com/Raxuis/chaosproxy/internal/config"

// Build creates the fault chain of a rule that passed config.Validate.
func Build(rule config.Rule) []Fault {
	chain := make([]Fault, 0, 9)
	if rule.Latency != nil {
		chain = append(chain, newLatencyFault(*rule.Latency))
	}
	if rule.Reset != nil {
		chain = append(chain, newResetFault(*rule.Reset))
	}
	if rule.Status != nil {
		chain = append(chain, newStatusFault(*rule.Status))
	}
	if rule.Hang != nil {
		chain = append(chain, newHangFault(*rule.Hang))
	}
	if rule.Mutate != nil {
		chain = append(chain, newMutateFault(*rule.Mutate))
	}
	if rule.Headers != nil {
		chain = append(chain, newHeadersFault(*rule.Headers))
	}
	if rule.Truncate != nil {
		chain = append(chain, newTruncateFault(*rule.Truncate))
	}
	if rule.Stall != nil {
		chain = append(chain, newStallFault(*rule.Stall))
	}
	if rule.Bandwidth != nil {
		chain = append(chain, newBandwidthFault(*rule.Bandwidth))
	}
	return chain
}
