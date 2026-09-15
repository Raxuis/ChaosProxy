package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type compiledPath struct {
	segments []compiledSegment
	globstar bool
}

type compiledSegment struct {
	globstar   bool
	anySegment bool
	literal    string
	pattern    *regexp.Regexp
}

func compilePath(pattern string) (compiledPath, error) {
	if pattern == "" || pattern[0] != '/' {
		return compiledPath{}, errors.New("path pattern must start with /")
	}
	if strings.ContainsAny(pattern, "?#") {
		return compiledPath{}, errors.New("path pattern must not contain a query string or fragment")
	}

	parts := splitPath(pattern)
	compiled := compiledPath{segments: make([]compiledSegment, len(parts))}
	for index, part := range parts {
		segment, err := compileSegment(part)
		if err != nil {
			return compiledPath{}, fmt.Errorf("invalid segment %q: %w", part, err)
		}
		compiled.segments[index] = segment
		compiled.globstar = compiled.globstar || segment.globstar
	}
	return compiled, nil
}

func compileSegment(pattern string) (compiledSegment, error) {
	switch {
	case pattern == "**":
		return compiledSegment{globstar: true}, nil
	case strings.Contains(pattern, "**"):
		return compiledSegment{}, errors.New("** must occupy an entire path segment")
	case pattern == "*":
		return compiledSegment{anySegment: true}, nil
	case !strings.Contains(pattern, "*"):
		return compiledSegment{literal: pattern}, nil
	}

	var expression strings.Builder
	expression.WriteByte('^')
	for index, literal := range strings.Split(pattern, "*") {
		if index > 0 {
			expression.WriteString(".*")
		}
		expression.WriteString(regexp.QuoteMeta(literal))
	}
	expression.WriteByte('$')

	compiled, err := regexp.Compile(expression.String())
	if err != nil {
		return compiledSegment{}, err
	}
	return compiledSegment{pattern: compiled}, nil
}

// Only a literal empty segment matches an empty path segment, so "/users/*"
// does not match "/users/".
func (segment compiledSegment) matches(value string) bool {
	switch {
	case segment.pattern != nil:
		return segment.pattern.MatchString(value)
	case segment.anySegment:
		return value != ""
	default:
		return value == segment.literal
	}
}

func (pattern compiledPath) matches(path string) bool {
	if path == "" || path[0] != '/' {
		return false
	}
	if pattern.globstar {
		return pattern.matchesSegments(splitPath(path))
	}

	rest := path[1:]
	for index, segment := range pattern.segments {
		value, next, more := strings.Cut(rest, "/")
		if !segment.matches(value) {
			return false
		}
		if index == len(pattern.segments)-1 {
			return !more
		}
		if !more {
			return false
		}
		rest = next
	}
	return false
}

func (pattern compiledPath) matchesSegments(segments []string) bool {
	columns := len(segments) + 1
	matched := make([]bool, (len(pattern.segments)+1)*columns)
	matched[len(pattern.segments)*columns+len(segments)] = true

	for patternIndex := len(pattern.segments) - 1; patternIndex >= 0; patternIndex-- {
		segment := pattern.segments[patternIndex]
		row := patternIndex * columns
		next := row + columns
		for pathIndex := len(segments); pathIndex >= 0; pathIndex-- {
			remaining := pathIndex < len(segments)
			if segment.globstar {
				matched[row+pathIndex] = matched[next+pathIndex] || (remaining && matched[row+pathIndex+1])
				continue
			}
			matched[row+pathIndex] = remaining && matched[next+pathIndex+1] && segment.matches(segments[pathIndex])
		}
	}
	return matched[0]
}

func splitPath(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}
