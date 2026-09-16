package proxy

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/faults"
)

const (
	overrideHeader = "X-Chaos"
	appliedHeader  = "X-Chaos-Applied"
	overrideRule   = "x-chaos"
)

var overrideExamples = map[string]string{
	"latency":   "latency=800ms",
	"status":    "status=503",
	"truncate":  "truncate=0.5",
	"bandwidth": "bandwidth=32768",
}

type faultSelection struct {
	rule    string
	chain   []faults.Fault
	index   uint64
	applied string
}

func (h *Handler) selectFaults(runtime *runtimeConfig, r *http.Request) (faultSelection, error) {
	values := r.Header.Values(overrideHeader)
	if len(values) == 0 || !h.headerOverrides {
		var selection faultSelection
		if len(values) > 0 {
			selection.applied = "disabled"
		}
		if matched := runtime.match.Match(r.Method, r.URL.Path); matched != nil {
			selection.rule = matched.Name
			selection.chain = runtime.chains[matched.Name]
			selection.index = h.nextRuleIndex(matched.Name)
		}
		return selection, nil
	}

	if len(values) > 1 {
		return faultSelection{}, errors.New("invalid X-Chaos header: send a single X-Chaos header")
	}
	rule, applied, err := parseOverride(values[0])
	if err != nil {
		return faultSelection{}, fmt.Errorf("invalid X-Chaos header: %w", err)
	}
	return faultSelection{rule: overrideRule, chain: faults.Build(rule), applied: applied}, nil
}

func parseOverride(value string) (config.Rule, string, error) {
	var rule config.Rule
	seen := make(map[string]bool)
	for _, directive := range strings.Split(value, ";") {
		name, argument, hasArgument := strings.Cut(strings.TrimSpace(directive), "=")
		name = strings.TrimSpace(name)
		argument = strings.TrimSpace(argument)
		if name == "" {
			return config.Rule{}, "", errors.New("empty directive; separate faults with a single ';'")
		}
		if seen[name] {
			return config.Rule{}, "", fmt.Errorf("%s is set more than once", name)
		}
		seen[name] = true

		switch name {
		case "off", "hang", "reset":
			if hasArgument {
				return config.Rule{}, "", fmt.Errorf("%s takes no value", name)
			}
		case "latency", "status", "truncate", "bandwidth":
			if argument == "" {
				return config.Rule{}, "", fmt.Errorf("%s needs a value, for example %s", name, overrideExamples[name])
			}
		default:
			return config.Rule{}, "", fmt.Errorf("unknown fault %q; use latency, status, hang, reset, truncate, bandwidth, or off", name)
		}

		switch name {
		case "hang":
			rule.Hang = &config.HangConfig{Probability: 1}
		case "reset":
			rule.Reset = &config.ResetConfig{Probability: 1}
		case "latency":
			delay, err := time.ParseDuration(argument)
			if err != nil || delay < 0 {
				return config.Rule{}, "", fmt.Errorf("latency must be a non-negative duration such as 800ms, got %q", argument)
			}
			rule.Latency = &config.LatencyConfig{Dist: "fixed", Value: delay}
		case "status":
			code, err := strconv.Atoi(argument)
			if err != nil || code < 100 || code > 599 {
				return config.Rule{}, "", fmt.Errorf("status must be an HTTP status code between 100 and 599, got %q", argument)
			}
			rule.Status = &config.StatusConfig{Code: code, Probability: 1}
		case "truncate":
			at, err := strconv.ParseFloat(argument, 64)
			if err != nil || math.IsNaN(at) || at < 0 || at > 1 {
				return config.Rule{}, "", fmt.Errorf("truncate must be a fraction between 0 and 1, got %q", argument)
			}
			rule.Truncate = &config.TruncateConfig{Probability: 1, At: at}
		case "bandwidth":
			bytesPerSecond, err := strconv.ParseInt(argument, 10, 64)
			if err != nil || bytesPerSecond <= 0 {
				return config.Rule{}, "", fmt.Errorf("bandwidth must be a positive number of bytes per second, got %q", argument)
			}
			rule.Bandwidth = &config.BandwidthConfig{BytesPerSecond: bytesPerSecond}
		}
	}

	if seen["off"] {
		if len(seen) > 1 {
			return config.Rule{}, "", errors.New("off cannot be combined with faults")
		}
		return config.Rule{}, "off", nil
	}
	if terminal := countTrue(seen["status"], seen["hang"], seen["reset"]); terminal > 1 {
		return config.Rule{}, "", errors.New("status, hang, and reset each end the request; use only one")
	}
	return rule, describeOverride(rule), nil
}

func describeOverride(rule config.Rule) string {
	parts := make([]string, 0, 6)
	if rule.Latency != nil {
		parts = append(parts, "latency="+rule.Latency.Value.String())
	}
	if rule.Reset != nil {
		parts = append(parts, "reset")
	}
	if rule.Status != nil {
		parts = append(parts, "status="+strconv.Itoa(rule.Status.Code))
	}
	if rule.Hang != nil {
		parts = append(parts, "hang")
	}
	if rule.Truncate != nil {
		parts = append(parts, "truncate="+strconv.FormatFloat(rule.Truncate.At, 'f', -1, 64))
	}
	if rule.Bandwidth != nil {
		parts = append(parts, "bandwidth="+strconv.FormatInt(rule.Bandwidth.BytesPerSecond, 10))
	}
	return strings.Join(parts, "; ")
}

func countTrue(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}
