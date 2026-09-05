package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type compiledPath []compiledSegment

type compiledSegment struct {
	globstar     bool
	pattern      *regexp.Regexp
	matchesEmpty bool
}

func compilePath(pattern string) (compiledPath, error) {
	if pattern == "" || pattern[0] != '/' {
		return nil, errors.New("path pattern must start with /")
	}
	if strings.ContainsAny(pattern, "?#") {
		return nil, errors.New("path pattern must not contain a query string or fragment")
	}

	parts := splitPath(pattern)
	compiled := make(compiledPath, len(parts))
	for index, part := range parts {
		segment, err := compileSegment(part)
		if err != nil {
			return nil, fmt.Errorf("invalid segment %q: %w", part, err)
		}
		compiled[index] = segment
	}
	return compiled, nil
}

func compileSegment(pattern string) (compiledSegment, error) {
	if pattern == "**" {
		return compiledSegment{globstar: true}, nil
	}
	if strings.Contains(pattern, "**") {
		return compiledSegment{}, errors.New("** must occupy an entire path segment")
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
	return compiledSegment{pattern: compiled, matchesEmpty: pattern == ""}, nil
}

func (pattern compiledPath) matches(path string) bool {
	if path == "" || path[0] != '/' {
		return false
	}

	segments := splitPath(path)
	type position struct {
		pattern int
		path    int
	}
	memo := make(map[position]bool)
	visited := make(map[position]bool)

	var match func(int, int) bool
	match = func(patternIndex, pathIndex int) bool {
		current := position{pattern: patternIndex, path: pathIndex}
		if visited[current] {
			return memo[current]
		}
		visited[current] = true

		if patternIndex == len(pattern) {
			memo[current] = pathIndex == len(segments)
			return memo[current]
		}

		segment := pattern[patternIndex]
		if segment.globstar {
			memo[current] = match(patternIndex+1, pathIndex) ||
				pathIndex < len(segments) && match(patternIndex, pathIndex+1)
			return memo[current]
		}
		if pathIndex >= len(segments) || (segments[pathIndex] == "" && !segment.matchesEmpty) {
			return false
		}

		memo[current] = segment.pattern.MatchString(segments[pathIndex]) && match(patternIndex+1, pathIndex+1)
		return memo[current]
	}

	return match(0, 0)
}

func splitPath(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}
