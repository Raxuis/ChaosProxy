package proxy

import (
	"fmt"

	"github.com/Raxuis/chaosproxy/internal/config"
)

// Update compiles configured completely before atomically publishing it. A
// failed update leaves the current runtime untouched.
func (handler *Handler) Update(configured *config.Config) error {
	runtime, err := compileRuntime(configured)
	if err != nil {
		return err
	}
	handler.current.Store(runtime)
	return nil
}

// CurrentConfig returns a mutable copy of the active configuration.
func (handler *Handler) CurrentConfig() *config.Config {
	return handler.current.Load().configured.Clone()
}

// SetRuleEnabled atomically toggles one rule without modifying its YAML source.
func (handler *Handler) SetRuleEnabled(name string, enabled bool) error {
	for {
		current := handler.current.Load()
		candidate := current.configured.Clone()
		found := false
		for index := range candidate.Rules {
			if candidate.Rules[index].Name == name {
				candidate.Rules[index].Enabled = enabled
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: %q", config.ErrRuleNotFound, name)
		}

		next, err := compileRuntime(candidate)
		if err != nil {
			return err
		}
		if handler.current.CompareAndSwap(current, next) {
			return nil
		}
	}
}

// Reset clears request-scoped deterministic runtime counters.
func (handler *Handler) Reset() {
	handler.requestIndex.Store(0)
}
