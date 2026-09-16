package proxy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/faults"
)

const (
	overrideHeader = "X-Chaos"
	appliedHeader  = "X-Chaos-Applied"
	overrideRule   = "x-chaos"
)

type faultSelection struct {
	rule    string
	chain   []faults.Fault
	index   uint64
	applied string
}

func (h *Handler) selectFaults(runtime *runtimeConfig, r *http.Request) (faultSelection, error) {
	values := r.Header.Values(overrideHeader)
	if len(values) > 0 && h.headerOverrides {
		if len(values) > 1 {
			return faultSelection{}, errors.New("invalid X-Chaos header: send a single X-Chaos header")
		}
		rule, applied, err := config.ParseFaults(values[0])
		if err != nil {
			return faultSelection{}, fmt.Errorf("invalid X-Chaos header: %w", err)
		}
		return faultSelection{rule: overrideRule, chain: faults.Build(rule), applied: applied}, nil
	}

	var selection faultSelection
	if len(values) > 0 {
		selection.applied = "disabled"
	}
	if matched := runtime.scenarioMatch.Match(r.Method, r.URL.Path); matched != nil {
		scenario := runtime.scenarios[matched.Name]
		selection.rule = scenario.name
		selection.index = nextIndex(&h.scenarioCounters, scenario.name)
		if step, active := scenario.step(selection.index); active {
			selection.chain = scenario.steps[step]
		}
		return selection, nil
	}
	if matched := runtime.match.Match(r.Method, r.URL.Path); matched != nil {
		selection.rule = matched.Name
		selection.chain = runtime.chains[matched.Name]
		selection.index = nextIndex(&h.ruleCounters, matched.Name)
	}
	return selection, nil
}
