package config

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

// Validate returns every semantic configuration error in one error value.
func (configured *Config) Validate() error {
	if configured == nil {
		return errors.New("configuration must not be nil")
	}

	issues := make([]error, 0)
	addIssue := func(line int, field, message string) {
		if line > 0 {
			issues = append(issues, fmt.Errorf("line %d: %s %s", line, field, message))
			return
		}
		issues = append(issues, fmt.Errorf("%s %s", field, message))
	}

	validateTarget(configured, addIssue)
	validateCORS(configured, addIssue)
	validateRules(configured, addIssue)
	return errors.Join(issues...)
}

func validateCORS(configured *Config, addIssue func(int, string, string)) {
	switch configured.CORS {
	case "", CORSReflect, CORSPassthrough, CORSOff:
		return
	default:
		addIssue(configured.source.lineFor("cors"), "cors", "must be reflect, passthrough, or off")
	}
}

func validateTarget(configured *Config, addIssue func(int, string, string)) {
	line := configured.source.lineFor("target")
	if configured.Target == "" {
		addIssue(line, "target", "must not be empty")
		return
	}

	target, err := url.Parse(configured.Target)
	if err != nil {
		addIssue(line, "target", fmt.Sprintf("must be a valid URL: %v", err))
		return
	}
	if (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		addIssue(line, "target", "must be an absolute HTTP or HTTPS URL")
	}
}

func validateRules(configured *Config, addIssue func(int, string, string)) {
	names := make(map[string]int, len(configured.Rules))
	for index := range configured.Rules {
		rule := &configured.Rules[index]
		prefix := fmt.Sprintf("rules[%d]", index)

		if rule.Name == "" {
			addIssue(rule.source.lineFor("name"), prefix+".name", "must not be empty")
		} else if firstLine, exists := names[rule.Name]; exists {
			addIssue(
				rule.source.lineFor("name"),
				prefix+".name",
				fmt.Sprintf("must be unique; %q was first declared on line %d", rule.Name, firstLine),
			)
		} else {
			names[rule.Name] = rule.source.lineFor("name")
		}

		if strings.TrimSpace(rule.Match) == "" {
			addIssue(rule.source.lineFor("match"), prefix+".match", "must not be empty")
		}
		if rule.Latency == nil && rule.Status == nil && rule.Hang == nil && rule.Truncate == nil {
			addIssue(rule.source.line, prefix, "must configure at least one fault")
		}

		validateLatency(rule, prefix, addIssue)
		validateStatus(rule, prefix, addIssue)
		validateProbability(rule, prefix, "hang.probability", probabilityOfHang(rule), addIssue)
		validateTruncate(rule, prefix, addIssue)
	}
}

func validateLatency(rule *Rule, prefix string, addIssue func(int, string, string)) {
	if rule.Latency == nil {
		return
	}

	latency := rule.Latency
	switch latency.Dist {
	case "fixed":
		validateNonNegativeDuration(rule, prefix, "latency.value", latency.Value, addIssue)
		validateNonNegativeDuration(rule, prefix, "latency.jitter", latency.Jitter, addIssue)
	case "lognormal":
		if latency.P50 <= 0 {
			addIssue(rule.source.lineFor("latency.p50"), prefix+".latency.p50", "must be greater than zero")
		}
		if latency.P99 <= 0 {
			addIssue(rule.source.lineFor("latency.p99"), prefix+".latency.p99", "must be greater than zero")
		}
		if latency.P50 > latency.P99 {
			addIssue(rule.source.lineFor("latency.p99"), prefix+".latency", "must satisfy p50 <= p99")
		}
	default:
		addIssue(rule.source.lineFor("latency.dist"), prefix+".latency.dist", "must be either fixed or lognormal")
	}
}

func validateStatus(rule *Rule, prefix string, addIssue func(int, string, string)) {
	if rule.Status == nil {
		return
	}

	if rule.Status.Code < 100 || rule.Status.Code > 599 {
		addIssue(rule.source.lineFor("status.code"), prefix+".status.code", "must be between 100 and 599")
	}
	validateProbability(rule, prefix, "status.probability", rule.Status.Probability, addIssue)
	if rule.Status.RetryAfter < 0 {
		addIssue(rule.source.lineFor("status.retry_after"), prefix+".status.retry_after", "must not be negative")
	}
}

func validateTruncate(rule *Rule, prefix string, addIssue func(int, string, string)) {
	if rule.Truncate == nil {
		return
	}

	validateProbability(rule, prefix, "truncate.probability", rule.Truncate.Probability, addIssue)
	if math.IsNaN(rule.Truncate.At) || rule.Truncate.At < 0 || rule.Truncate.At > 1 {
		addIssue(rule.source.lineFor("truncate.at"), prefix+".truncate.at", "must be between 0 and 1")
	}
}

func validateProbability(
	rule *Rule,
	prefix string,
	field string,
	probability float64,
	addIssue func(int, string, string),
) {
	if math.IsNaN(probability) || probability < 0 || probability > 1 {
		addIssue(rule.source.lineFor(field), prefix+"."+field, "must be between 0 and 1")
	}
}

func probabilityOfHang(rule *Rule) float64 {
	if rule.Hang == nil {
		return 0
	}
	return rule.Hang.Probability
}

func validateNonNegativeDuration(
	rule *Rule,
	prefix string,
	field string,
	duration time.Duration,
	addIssue func(int, string, string),
) {
	if duration < 0 {
		addIssue(rule.source.lineFor(field), prefix+"."+field, "must not be negative")
	}
}

func (source sourceLocation) lineFor(field string) int {
	if line := source.fields[field]; line > 0 {
		return line
	}
	return source.line
}
