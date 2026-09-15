package rules

import (
	"regexp"
	"strings"
	"testing"
)

func TestCompiledPathMatchesReferenceImplementation(t *testing.T) {
	t.Parallel()

	patterns := []string{
		"/", "/api", "/api/", "/api/*", "/api/*/avatar", "/api/search*", "/api/*.json", "/*/*",
		"/**", "/api/**", "/**/metadata", "/files/**/metadata", "/a/**/b/**/c", "//x", "/api/**/",
	}
	paths := []string{
		"/", "/api", "/api/", "/api/users", "/api/users/", "/api/users/42/avatar", "/api/search",
		"/api/search-all", "/api/data.json", "/files/metadata", "/files/a/b/metadata", "/files/a/b/metadata/",
		"/a/b/c", "/a/x/b/y/c", "/a/b", "//x", "/x//y", "/api//avatar",
	}
	for _, pattern := range patterns {
		compiled, err := compilePath(pattern)
		if err != nil {
			t.Fatalf("compilePath(%q) unexpected error: %v", pattern, err)
		}
		for _, path := range paths {
			if got, want := compiled.matches(path), referenceMatch(pattern, path); got != want {
				t.Errorf("pattern %q path %q: matches = %t, want %t", pattern, path, got, want)
			}
		}
	}
}

func referenceMatch(pattern, path string) bool {
	patternParts := splitPath(pattern)
	pathParts := splitPath(path)
	var match func(patternIndex, pathIndex int) bool
	match = func(patternIndex, pathIndex int) bool {
		if patternIndex == len(patternParts) {
			return pathIndex == len(pathParts)
		}
		part := patternParts[patternIndex]
		if part == "**" {
			return match(patternIndex+1, pathIndex) || (pathIndex < len(pathParts) && match(patternIndex, pathIndex+1))
		}
		if pathIndex >= len(pathParts) || (pathParts[pathIndex] == "" && part != "") {
			return false
		}
		quoted := make([]string, 0)
		for _, literal := range strings.Split(part, "*") {
			quoted = append(quoted, regexp.QuoteMeta(literal))
		}
		expression := regexp.MustCompile("^" + strings.Join(quoted, ".*") + "$")
		return expression.MatchString(pathParts[pathIndex]) && match(patternIndex+1, pathIndex+1)
	}
	return match(0, 0)
}
