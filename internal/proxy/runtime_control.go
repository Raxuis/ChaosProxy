package proxy

import (
	"errors"
	"fmt"
	"maps"

	"github.com/Raxuis/chaosproxy/internal/config"
)

// Update atomically replaces the file configuration and keeps still-relevant rule toggles.
func (handler *Handler) Update(base *config.Config) error {
	if base == nil {
		return errors.New("configuration must not be nil")
	}
	for {
		current := handler.current.Load()
		next, err := compileRuntime(base, keptOverrides(current, base))
		if err != nil {
			return err
		}
		if handler.current.CompareAndSwap(current, next) {
			return nil
		}
	}
}

// CurrentConfig returns a mutable copy of the active configuration, toggles included.
func (handler *Handler) CurrentConfig() *config.Config {
	return handler.current.Load().configured.Clone()
}

// SetRuleEnabled atomically toggles one rule without modifying its YAML source.
func (handler *Handler) SetRuleEnabled(name string, enabled bool) error {
	for {
		current := handler.current.Load()
		fileEnabled, found := ruleEnabled(current.base, name)
		if !found {
			return fmt.Errorf("%w: %q", config.ErrRuleNotFound, name)
		}

		overrides := make(map[string]bool, len(current.overrides)+1)
		maps.Copy(overrides, current.overrides)
		if enabled == fileEnabled {
			delete(overrides, name)
		} else {
			overrides[name] = enabled
		}

		next, err := compileRuntime(current.base, overrides)
		if err != nil {
			return err
		}
		if handler.current.CompareAndSwap(current, next) {
			return nil
		}
	}
}

// Reset restarts every rule's deterministic decision sequence.
func (handler *Handler) Reset() {
	handler.ruleCounters.Clear()
}

// A toggle is dropped once the file changes that rule's enabled value, so the
// most recent intent wins.
func keptOverrides(current *runtimeConfig, base *config.Config) map[string]bool {
	kept := make(map[string]bool, len(current.overrides))
	for name, enabled := range current.overrides {
		previous, _ := ruleEnabled(current.base, name)
		if next, exists := ruleEnabled(base, name); exists && next == previous {
			kept[name] = enabled
		}
	}
	return kept
}

func ruleEnabled(configured *config.Config, name string) (bool, bool) {
	for _, rule := range configured.Rules {
		if rule.Name == name {
			return rule.Enabled, true
		}
	}
	return false, false
}
