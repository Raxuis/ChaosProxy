package rules_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/rules"
)

func TestMatcher(t *testing.T) {
	t.Parallel()

	configured := []config.Rule{
		{Name: "exact-first", Match: "GET /api/users", Enabled: true, Status: statusFault()},
		{Name: "search-prefix", Match: "get /api/search*", Enabled: true, Status: statusFault()},
		{Name: "one-segment", Match: "GET /api/users/*", Enabled: true, Status: statusFault()},
		{Name: "multi-segment", Match: "GET /files/**/metadata", Enabled: true, Status: statusFault()},
		{Name: "methodless", Match: "/public/*", Enabled: true, Status: statusFault()},
		{Name: "trailing", Match: "GET /trailing/", Enabled: true, Status: statusFault()},
		{Name: "disabled", Match: "GET /disabled", Enabled: false, Status: statusFault()},
		{Name: "enabled-after-disabled", Match: "GET /disabled", Enabled: true, Status: statusFault()},
		{Name: "root", Match: "/", Enabled: true, Status: statusFault()},
	}

	matcher, err := rules.Compile(configured)
	if err != nil {
		t.Fatalf("Compile() unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		method   string
		path     string
		wantRule string
	}{
		{name: "exact", method: "GET", path: "/api/users", wantRule: "exact-first"},
		{name: "method is case insensitive", method: "GET", path: "/api/search", wantRule: "search-prefix"},
		{name: "wildcard inside segment", method: "get", path: "/api/search-all", wantRule: "search-prefix"},
		{name: "wrong method", method: "POST", path: "/api/search-all"},
		{name: "one segment", method: "GET", path: "/api/users/42", wantRule: "one-segment"},
		{name: "one segment does not cross slash", method: "GET", path: "/api/users/42/avatar"},
		{name: "one segment is not empty", method: "GET", path: "/api/users/"},
		{name: "globstar zero segments", method: "GET", path: "/files/metadata", wantRule: "multi-segment"},
		{name: "globstar many segments", method: "GET", path: "/files/a/b/metadata", wantRule: "multi-segment"},
		{name: "globstar suffix required", method: "GET", path: "/files/a/b"},
		{name: "method omitted", method: "PATCH", path: "/public/item", wantRule: "methodless"},
		{name: "query ignored", method: "GET", path: "/api/users?expand=profile", wantRule: "exact-first"},
		{name: "trailing slash strict no slash", method: "GET", path: "/trailing"},
		{name: "trailing slash strict with slash", method: "GET", path: "/trailing/", wantRule: "trailing"},
		{name: "disabled rule skipped", method: "GET", path: "/disabled", wantRule: "enabled-after-disabled"},
		{name: "root", method: "OPTIONS", path: "/", wantRule: "root"},
		{name: "path must be absolute", method: "GET", path: "api/users"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			matched := matcher.Match(test.method, test.path)
			if test.wantRule == "" {
				if matched != nil {
					t.Fatalf("Match() = %q, want nil", matched.Name)
				}
				return
			}
			if matched == nil || matched.Name != test.wantRule {
				if matched == nil {
					t.Fatalf("Match() = nil, want %q", test.wantRule)
				}
				t.Fatalf("Match() = %q, want %q", matched.Name, test.wantRule)
			}
		})
	}
}

func TestMatcherUsesFirstMatch(t *testing.T) {
	t.Parallel()

	matcher, err := rules.Compile([]config.Rule{
		{Name: "broad", Match: "GET /api/**", Enabled: true, Status: statusFault()},
		{Name: "specific", Match: "GET /api/users", Enabled: true, Status: statusFault()},
	})
	if err != nil {
		t.Fatalf("Compile() unexpected error: %v", err)
	}

	matched := matcher.Match(httpMethodGet, "/api/users")
	if matched == nil || matched.Name != "broad" {
		t.Fatalf("Match() = %#v, want first rule", matched)
	}
}

func TestCompileReturnsAllPatternErrors(t *testing.T) {
	t.Parallel()

	_, err := rules.Compile([]config.Rule{
		{Name: "missing-path", Match: "GET", Enabled: true},
		{Name: "bad-globstar", Match: "GET /api/us**ers", Enabled: true},
		{Name: "query", Match: "/api?debug=true", Enabled: true},
	})
	if err == nil {
		t.Fatal("Compile() error = nil, want aggregate error")
	}
	for _, expected := range []string{"missing-path", "bad-globstar", "query"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Compile() error does not contain %q: %v", expected, err)
		}
	}
}

func TestCompileRejectsMalformedExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
	}{
		{name: "empty", expression: ""},
		{name: "method without path", expression: "GET"},
		{name: "too many fields", expression: "GET /api extra"},
		{name: "path without leading slash", expression: "GET api"},
		{name: "invalid method", expression: "GE:T /api"},
		{name: "fragment", expression: "/api#fragment"},
		{name: "embedded globstar", expression: "/api/foo**bar"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := rules.Compile([]config.Rule{{Name: test.name, Match: test.expression, Enabled: true}})
			if err == nil {
				t.Fatalf("Compile(%q) error = nil", test.expression)
			}
		})
	}
}

func TestMatcherIsSafeForConcurrentReads(t *testing.T) {
	t.Parallel()

	matcher, err := rules.Compile([]config.Rule{{
		Name:    "all",
		Match:   "/**",
		Enabled: true,
		Status:  statusFault(),
	}})
	if err != nil {
		t.Fatalf("Compile() unexpected error: %v", err)
	}

	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if matched := matcher.Match("GET", "/any/nested/path"); matched == nil {
				t.Error("Match() = nil, want rule")
			}
		}()
	}
	wait.Wait()
}

const httpMethodGet = "GET"

func statusFault() *config.StatusConfig {
	return &config.StatusConfig{Code: 503, Probability: 1}
}
