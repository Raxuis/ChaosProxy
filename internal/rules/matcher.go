// Package rules compiles and evaluates ordered route rules.
package rules

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Raxuis/chaosproxy/internal/config"
)

// Matcher is an immutable, concurrency-safe ordered rule matcher.
type Matcher struct {
	rules []compiledRule
}

type compiledRule struct {
	rule   config.Rule
	method string
	path   compiledPath
}

// Compile parses every rule expression once. It returns all expression errors
// together and never leaves a partially usable matcher.
func Compile(configured []config.Rule) (*Matcher, error) {
	compiled := make([]compiledRule, 0, len(configured))
	issues := make([]error, 0)

	for index := range configured {
		method, pathPattern, err := parseExpression(configured[index].Match)
		if err != nil {
			issues = append(issues, fmt.Errorf("rules[%d] %q: %w", index, configured[index].Name, err))
			continue
		}
		path, err := compilePath(pathPattern)
		if err != nil {
			issues = append(issues, fmt.Errorf("rules[%d] %q: %w", index, configured[index].Name, err))
			continue
		}

		compiled = append(compiled, compiledRule{
			rule:   configured[index],
			method: method,
			path:   path,
		})
	}

	if err := errors.Join(issues...); err != nil {
		return nil, err
	}
	return &Matcher{rules: compiled}, nil
}

// Match returns the first enabled rule matching method and path. Query strings
// are ignored, HTTP methods are compared case-insensitively, and trailing
// slashes remain significant unless the pattern explicitly uses a wildcard.
func (matcher *Matcher) Match(method, path string) *config.Rule {
	if matcher == nil {
		return nil
	}

	path, _, _ = strings.Cut(path, "?")
	for index := range matcher.rules {
		candidate := &matcher.rules[index]
		if !candidate.rule.Enabled {
			continue
		}
		if candidate.method != "" && !strings.EqualFold(candidate.method, method) {
			continue
		}
		if candidate.path.matches(path) {
			return &candidate.rule
		}
	}

	return nil
}

func parseExpression(expression string) (string, string, error) {
	parts := strings.Fields(expression)
	switch len(parts) {
	case 1:
		if !strings.HasPrefix(parts[0], "/") {
			return "", "", errors.New("a path-only expression must start with /")
		}
		return "", parts[0], nil
	case 2:
		if !validMethod(parts[0]) {
			return "", "", fmt.Errorf("invalid HTTP method %q", parts[0])
		}
		if !strings.HasPrefix(parts[1], "/") {
			return "", "", errors.New("path pattern must start with /")
		}
		return strings.ToUpper(parts[0]), parts[1], nil
	default:
		return "", "", errors.New("expression must be either /path/glob or METHOD /path/glob")
	}
}

func validMethod(method string) bool {
	if method == "" {
		return false
	}
	for index := 0; index < len(method); index++ {
		character := method[index]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') {
			continue
		}
		switch character {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return true
}
