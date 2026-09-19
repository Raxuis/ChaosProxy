package config

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Validate returns every semantic configuration error in one error value.
func (c *Config) Validate() error {
	if c == nil {
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

	validateTarget(c, addIssue)
	validateCORS(c, addIssue)
	validateCORSOrigins(c, addIssue)
	validateRules(c, addIssue)
	validateScenarios(c, addIssue)
	return errors.Join(issues...)
}

func validateScenarios(cfg *Config, addIssue func(int, string, string)) {
	used := make(map[string]bool, len(cfg.Rules)+len(cfg.Scenarios))
	for _, rule := range cfg.Rules {
		used[rule.Name] = true
	}

	for index := range cfg.Scenarios {
		scenario := &cfg.Scenarios[index]
		prefix := fmt.Sprintf("scenarios[%d]", index)

		switch {
		case scenario.Name == "":
			addIssue(scenario.source.lineFor("name"), prefix+".name", "must not be empty")
		case used[scenario.Name]:
			addIssue(scenario.source.lineFor("name"), prefix+".name", fmt.Sprintf("must be unique across rules and scenarios; %q is already used", scenario.Name))
		}
		used[scenario.Name] = true

		if strings.TrimSpace(scenario.Match) == "" {
			addIssue(scenario.source.lineFor("match"), prefix+".match", "must not be empty")
		}
		switch scenario.OnExhausted {
		case ExhaustPassthrough, ExhaustRepeat, ExhaustLast:
		default:
			addIssue(scenario.source.lineFor("on_exhausted"), prefix+".on_exhausted", "must be passthrough, repeat, or last")
		}
		if len(scenario.Steps) == 0 {
			addIssue(scenario.source.lineFor("steps"), prefix+".steps", "must contain at least one step")
		}
		for stepIndex, step := range scenario.Steps {
			if _, _, err := ParseFaults(step); err != nil {
				field := fmt.Sprintf("steps[%d]", stepIndex)
				addIssue(scenario.source.lineFor(field), prefix+"."+field, "is invalid: "+err.Error())
			}
		}
	}
}

func validateCORSOrigins(cfg *Config, addIssue func(int, string, string)) {
	line := cfg.source.lineFor("cors_origins")
	for index, origin := range cfg.CORSOrigins {
		if origin == "*" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			addIssue(line, fmt.Sprintf("cors_origins[%d]", index), `must be "*" or an origin such as http://localhost:3000`)
		}
	}
}

func validateCORS(cfg *Config, addIssue func(int, string, string)) {
	switch cfg.CORS {
	case CORSReflect, CORSPassthrough, CORSOff:
		return
	default:
		addIssue(cfg.source.lineFor("cors"), "cors", "must be reflect, passthrough, or off")
	}
}

func validateTarget(cfg *Config, addIssue func(int, string, string)) {
	line := cfg.source.lineFor("target")
	if cfg.Target == "" {
		addIssue(line, "target", "must not be empty")
		return
	}

	target, err := url.Parse(cfg.Target)
	if err != nil {
		addIssue(line, "target", fmt.Sprintf("must be a valid URL: %v", err))
		return
	}
	if (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		addIssue(line, "target", "must be an absolute HTTP or HTTPS URL")
	}
}

func validateRules(cfg *Config, addIssue func(int, string, string)) {
	names := make(map[string]int, len(cfg.Rules))
	for index := range cfg.Rules {
		rule := &cfg.Rules[index]
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
		if rule.Latency == nil && rule.Status == nil && rule.Hang == nil && rule.Truncate == nil &&
			rule.Reset == nil && rule.Bandwidth == nil && rule.Headers == nil && rule.Mutate == nil &&
			rule.Redirect == nil && rule.Stall == nil {
			addIssue(rule.source.line, prefix, "must configure at least one fault")
		}

		validateLatency(rule, prefix, addIssue)
		validateStatus(rule, prefix, addIssue)
		validateRedirect(rule, prefix, addIssue)
		validateProbability(rule, prefix, "hang.probability", probabilityOfHang(rule), addIssue)
		validateTruncate(rule, prefix, addIssue)
		if rule.Reset != nil {
			validateProbability(rule, prefix, "reset.probability", rule.Reset.Probability, addIssue)
		}
		if rule.Bandwidth != nil && rule.Bandwidth.BytesPerSecond <= 0 {
			addIssue(rule.source.lineFor("bandwidth.bytes_per_second"), prefix+".bandwidth.bytes_per_second", "must be greater than zero")
		}
		validateHeaders(cfg, rule, prefix, addIssue)
		validateMutate(rule, prefix, addIssue)
		validateStall(rule, prefix, addIssue)
	}
}

func validateMutate(rule *Rule, prefix string, addIssue func(int, string, string)) {
	mutate := rule.Mutate
	if mutate == nil {
		return
	}

	validateProbability(rule, prefix, "mutate.probability", mutate.Probability, addIssue)
	if mutate.MaxBytes <= 0 {
		addIssue(rule.source.lineFor("mutate.max_bytes"), prefix+".mutate.max_bytes", "must be greater than zero")
	}
	if len(mutate.Operations) == 0 {
		addIssue(rule.source.lineFor("mutate.operations"), prefix+".mutate.operations", "must contain at least one operation")
	}
	for index, operation := range mutate.Operations {
		field := fmt.Sprintf("mutate.operations[%d]", index)
		line := rule.source.lineFor(field)
		switch operation.Op {
		case "nullify", "empty", "drop":
			if operation.Factor != 0 {
				addIssue(line, prefix+"."+field+".factor", "only applies to inflate and stretch")
			}
		case "inflate", "stretch":
			if operation.Factor < 2 || operation.Factor > 1000 {
				addIssue(line, prefix+"."+field+".factor", "must be between 2 and 1000")
			}
		default:
			addIssue(line, prefix+"."+field+".op", "must be nullify, empty, inflate, stretch, or drop")
		}
		if operation.Path == "" || slices.Contains(strings.Split(operation.Path, "."), "") {
			addIssue(line, prefix+"."+field+".path", "must be a dotted path such as user.email or items.*.price")
		}
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

func validateRedirect(rule *Rule, prefix string, addIssue func(int, string, string)) {
	if rule.Redirect == nil {
		return
	}

	if !isRedirectCode(rule.Redirect.Code) {
		addIssue(rule.source.lineFor("redirect.code"), prefix+".redirect.code", "must be 301, 302, 303, 307, or 308")
	}
	validateProbability(rule, prefix, "redirect.probability", rule.Redirect.Probability, addIssue)
	if strings.TrimSpace(rule.Redirect.Location) == "" {
		addIssue(rule.source.lineFor("redirect.location"), prefix+".redirect.location", "must not be empty")
	}
}

func isRedirectCode(code int) bool {
	switch code {
	case 301, 302, 303, 307, 308:
		return true
	default:
		return false
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

func validateStall(rule *Rule, prefix string, addIssue func(int, string, string)) {
	if rule.Stall == nil {
		return
	}

	validateProbability(rule, prefix, "stall.probability", rule.Stall.Probability, addIssue)
	if rule.Stall.AfterBytes < 0 {
		addIssue(rule.source.lineFor("stall.after_bytes"), prefix+".stall.after_bytes", "must not be negative")
	}
	if rule.Stall.Duration <= 0 {
		addIssue(rule.source.lineFor("stall.duration"), prefix+".stall.duration", "must be greater than zero")
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

func (s sourceLocation) lineFor(field string) int {
	if line := s.fields[field]; line > 0 {
		return line
	}
	return s.line
}

func validateHeaders(cfg *Config, rule *Rule, prefix string, addIssue func(int, string, string)) {
	headers := rule.Headers
	if headers == nil {
		return
	}

	validateProbability(rule, prefix, "headers.probability", headers.Probability, addIssue)
	if len(headers.Set) == 0 && len(headers.Remove) == 0 {
		addIssue(rule.source.lineFor("headers"), prefix+".headers", "must specify at least one header to set or remove")
	}

	getSetLine := func(name string) int {
		if name == "" {
			return rule.source.lineFor("headers.set")
		}
		return rule.source.lineFor("headers.set." + name)
	}

	setNames := slices.SortedFunc(maps.Keys(headers.Set), func(a, b string) int {
		lineA := getSetLine(a)
		lineB := getSetLine(b)
		if lineA != lineB {
			return cmp.Compare(lineA, lineB)
		}
		return cmp.Compare(a, b)
	})

	seenSet := make(map[string]string, len(headers.Set))
	for _, name := range setNames {
		field := prefix + ".headers.set." + name
		if name == "" {
			field = prefix + ".headers.set"
		}
		line := getSetLine(name)
		if !validateHeaderName(cfg, addIssue, line, field, name) {
			continue
		}
		lower := strings.ToLower(name)
		if first, exists := seenSet[lower]; exists {
			addIssue(line, field, fmt.Sprintf("header name %q collides with %q (case-insensitive)", name, first))
			continue
		}
		seenSet[lower] = name
	}

	for index, name := range headers.Remove {
		line := rule.source.lineFor(fmt.Sprintf("headers.remove[%d]", index))
		field := fmt.Sprintf("%s.headers.remove[%d]", prefix, index)
		if !validateHeaderName(cfg, addIssue, line, field, name) {
			continue
		}
		lower := strings.ToLower(name)
		if _, inSet := seenSet[lower]; inSet {
			addIssue(line, field, fmt.Sprintf("%q cannot appear in both set and remove", name))
		}
	}
}

func validateHeaderName(cfg *Config, addIssue func(int, string, string), line int, field, name string) bool {
	if name == "" {
		addIssue(line, field, "header name must not be empty")
		return false
	}
	if !isValidHeaderName(name) {
		addIssue(line, field, fmt.Sprintf("invalid header name %q", name))
		return false
	}
	if isManagedHeader(name) {
		addIssue(line, field, fmt.Sprintf("%q is managed by the proxy and cannot be modified", name))
		return false
	}
	if cfg.CORS != CORSPassthrough && strings.HasPrefix(strings.ToLower(name), "access-control-") {
		addIssue(line, field, fmt.Sprintf("%q is managed by CORS reflection; set cors: passthrough to modify CORS headers", name))
		return false
	}
	return true
}

func isManagedHeader(name string) bool {
	return strings.EqualFold(name, "Content-Length") || strings.EqualFold(name, "Transfer-Encoding")
}

func isValidHeaderName(name string) bool {
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !isTokenChar(c) {
			return false
		}
	}
	return true
}

func isTokenChar(c byte) bool {
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}
