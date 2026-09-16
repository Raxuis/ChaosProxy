package proxy

import (
	"errors"
	"fmt"
	"maps"

	"github.com/Raxuis/chaosproxy/internal/config"
)

// Update atomically replaces the file configuration and keeps still-relevant rule toggles.
func (h *Handler) Update(base *config.Config) error {
	if base == nil {
		return errors.New("configuration must not be nil")
	}
	for {
		current := h.current.Load()
		next, err := compileRuntime(base, keptOverrides(current, base))
		if err != nil {
			return err
		}
		if h.current.CompareAndSwap(current, next) {
			h.restartChangedScenarios(current, next)
			return nil
		}
	}
}

// CurrentConfig returns a mutable copy of the active configuration, toggles included.
func (h *Handler) CurrentConfig() *config.Config {
	return h.current.Load().effective.Clone()
}

// SetRuleEnabled atomically toggles one rule without modifying its YAML source.
func (h *Handler) SetRuleEnabled(name string, enabled bool) error {
	for {
		current := h.current.Load()
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
		if h.current.CompareAndSwap(current, next) {
			return nil
		}
	}
}

// Reset restarts every rule decision sequence and every scenario.
func (h *Handler) Reset() {
	h.ruleCounters.Clear()
	h.scenarioCounters.Clear()
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

func ruleEnabled(cfg *config.Config, name string) (bool, bool) {
	for _, rule := range cfg.Rules {
		if rule.Name == name {
			return rule.Enabled, true
		}
	}
	return false, false
}
