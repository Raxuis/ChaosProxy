package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var directiveExamples = map[string]string{
	"latency":   "latency=800ms",
	"status":    "status=503",
	"truncate":  "truncate=0.5",
	"bandwidth": "bandwidth=32768",
	"redirect":  "redirect=302:/login",
}

// ParseFaults parses directives such as "latency=800ms; status=503" into faults that
// always trigger and returns their canonical text. The "off" directive yields no fault.
func ParseFaults(value string) (Rule, string, error) {
	var rule Rule
	seen := make(map[string]bool)
	for _, directive := range strings.Split(value, ";") {
		name, argument, hasArgument := strings.Cut(strings.TrimSpace(directive), "=")
		name = strings.TrimSpace(name)
		argument = strings.TrimSpace(argument)
		if name == "" {
			return Rule{}, "", errors.New("empty directive; separate faults with a single ';'")
		}
		if seen[name] {
			return Rule{}, "", fmt.Errorf("%s is set more than once", name)
		}
		seen[name] = true

		switch name {
		case "off", "hang", "reset":
			if hasArgument {
				return Rule{}, "", fmt.Errorf("%s takes no value", name)
			}
		case "latency", "status", "truncate", "bandwidth", "redirect":
			if argument == "" {
				return Rule{}, "", fmt.Errorf("%s needs a value, for example %s", name, directiveExamples[name])
			}
		default:
			return Rule{}, "", fmt.Errorf("unknown fault %q; use latency, status, hang, reset, truncate, bandwidth, redirect, or off", name)
		}

		switch name {
		case "hang":
			rule.Hang = &HangConfig{Probability: 1}
		case "reset":
			rule.Reset = &ResetConfig{Probability: 1}
		case "latency":
			delay, err := time.ParseDuration(argument)
			if err != nil || delay < 0 {
				return Rule{}, "", fmt.Errorf("latency must be a non-negative duration such as 800ms, got %q", argument)
			}
			rule.Latency = &LatencyConfig{Dist: "fixed", Value: delay}
		case "status":
			code, err := strconv.Atoi(argument)
			if err != nil || code < 100 || code > 599 {
				return Rule{}, "", fmt.Errorf("status must be an HTTP status code between 100 and 599, got %q", argument)
			}
			rule.Status = &StatusConfig{Code: code, Probability: 1}
		case "redirect":
			rawCode, location, hasLocation := strings.Cut(argument, ":")
			if !hasLocation || strings.TrimSpace(location) == "" {
				return Rule{}, "", fmt.Errorf("redirect must be <code>:<location> such as 302:/login, got %q", argument)
			}
			code, err := strconv.Atoi(rawCode)
			if err != nil || !isRedirectCode(code) {
				return Rule{}, "", fmt.Errorf("redirect code must be 301, 302, 303, 307, or 308, got %q", rawCode)
			}
			rule.Redirect = &RedirectConfig{Code: code, Location: location, Probability: 1}
		case "truncate":
			at, err := strconv.ParseFloat(argument, 64)
			if err != nil || math.IsNaN(at) || at < 0 || at > 1 {
				return Rule{}, "", fmt.Errorf("truncate must be a fraction between 0 and 1, got %q", argument)
			}
			rule.Truncate = &TruncateConfig{Probability: 1, At: at}
		case "bandwidth":
			bytesPerSecond, err := strconv.ParseInt(argument, 10, 64)
			if err != nil || bytesPerSecond <= 0 {
				return Rule{}, "", fmt.Errorf("bandwidth must be a positive number of bytes per second, got %q", argument)
			}
			rule.Bandwidth = &BandwidthConfig{BytesPerSecond: bytesPerSecond}
		}
	}

	if seen["off"] {
		if len(seen) > 1 {
			return Rule{}, "", errors.New("off cannot be combined with faults")
		}
		return Rule{}, "off", nil
	}
	if countTrue(seen["status"], seen["hang"], seen["reset"], seen["redirect"]) > 1 {
		return Rule{}, "", errors.New("status, hang, reset, and redirect each end the request; use only one")
	}
	return rule, describeFaults(rule), nil
}

func describeFaults(rule Rule) string {
	parts := make([]string, 0, 7)
	if rule.Latency != nil {
		parts = append(parts, "latency="+rule.Latency.Value.String())
	}
	if rule.Reset != nil {
		parts = append(parts, "reset")
	}
	if rule.Status != nil {
		parts = append(parts, "status="+strconv.Itoa(rule.Status.Code))
	}
	if rule.Redirect != nil {
		parts = append(parts, "redirect="+strconv.Itoa(rule.Redirect.Code)+":"+rule.Redirect.Location)
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
