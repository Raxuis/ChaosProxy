package proxy

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/faults"
)

// ScenarioState reports how far a configured scenario has progressed.
type ScenarioState struct {
	Name        string   `json:"name"`
	Match       string   `json:"match"`
	Enabled     bool     `json:"enabled"`
	OnExhausted string   `json:"on_exhausted"`
	Steps       []string `json:"steps"`
	Served      uint64   `json:"served"`
	NextStep    *int     `json:"next_step"`
	Exhausted   bool     `json:"exhausted"`
}

type compiledScenario struct {
	name        string
	match       string
	enabled     bool
	onExhausted config.ExhaustMode
	steps       [][]faults.Fault
	labels      []string
}

func compileScenario(scenario config.Scenario) (*compiledScenario, error) {
	compiled := &compiledScenario{
		name:        scenario.Name,
		match:       scenario.Match,
		enabled:     scenario.Enabled,
		onExhausted: scenario.OnExhausted,
		steps:       make([][]faults.Fault, 0, len(scenario.Steps)),
		labels:      make([]string, 0, len(scenario.Steps)),
	}
	for index, step := range scenario.Steps {
		rule, label, err := config.ParseFaults(step)
		if err != nil {
			return nil, fmt.Errorf("scenario %q step %d: %w", scenario.Name, index, err)
		}
		compiled.steps = append(compiled.steps, faults.Build(rule))
		compiled.labels = append(compiled.labels, label)
	}
	return compiled, nil
}

func (s *compiledScenario) step(position uint64) (int, bool) {
	count := uint64(len(s.steps))
	switch {
	case position < count:
		return int(position), true
	case s.onExhausted == config.ExhaustRepeat:
		return int(position % count), true
	case s.onExhausted == config.ExhaustLast:
		return int(count - 1), true
	default:
		return 0, false
	}
}

func (s *compiledScenario) sameDefinition(other *compiledScenario) bool {
	return s.match == other.match && s.onExhausted == other.onExhausted && slices.Equal(s.labels, other.labels)
}

// Scenarios reports the progress of every configured scenario in configuration order.
func (h *Handler) Scenarios() []ScenarioState {
	runtime := h.current.Load()
	states := make([]ScenarioState, 0, len(runtime.scenarioOrder))
	for _, scenario := range runtime.scenarioOrder {
		served := currentIndex(&h.scenarioCounters, scenario.name)
		state := ScenarioState{
			Name:        scenario.name,
			Match:       scenario.match,
			Enabled:     scenario.enabled,
			OnExhausted: string(scenario.onExhausted),
			Steps:       slices.Clone(scenario.labels),
			Served:      served,
			Exhausted:   served >= uint64(len(scenario.steps)),
		}
		if step, active := scenario.step(served); active {
			state.NextStep = &step
		}
		states = append(states, state)
	}
	return states
}

// ResetScenario restarts one scenario from its first step.
func (h *Handler) ResetScenario(name string) error {
	if _, found := h.current.Load().scenarios[name]; !found {
		return fmt.Errorf("%w: %q", config.ErrScenarioNotFound, name)
	}
	h.scenarioCounters.Delete(name)
	return nil
}

func (h *Handler) restartChangedScenarios(previous, next *runtimeConfig) {
	for name, scenario := range previous.scenarios {
		if updated, found := next.scenarios[name]; !found || !scenario.sameDefinition(updated) {
			h.scenarioCounters.Delete(name)
		}
	}
}

func nextIndex(counters *sync.Map, name string) uint64 {
	counter, found := counters.Load(name)
	if !found {
		counter, _ = counters.LoadOrStore(name, new(atomic.Uint64))
	}
	return counter.(*atomic.Uint64).Add(1) - 1
}

func currentIndex(counters *sync.Map, name string) uint64 {
	counter, found := counters.Load(name)
	if !found {
		return 0
	}
	return counter.(*atomic.Uint64).Load()
}
