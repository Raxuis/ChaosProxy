package rules_test

import (
	"net/http"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/rules"
)

func BenchmarkMatcherMatch(b *testing.B) {
	matcher, err := rules.Compile([]config.Rule{
		{Name: "orders", Match: "POST /api/orders", Enabled: true, Status: statusFault()},
		{Name: "search", Match: "GET /api/search*", Enabled: true, Status: statusFault()},
		{Name: "user", Match: "GET /api/users/*", Enabled: true, Status: statusFault()},
		{Name: "files", Match: "GET /files/**/metadata", Enabled: true, Status: statusFault()},
		{Name: "avatar", Match: "GET /api/users/*/avatar", Enabled: true, Status: statusFault()},
	})
	if err != nil {
		b.Fatalf("Compile() unexpected error: %v", err)
	}

	for name, path := range map[string]string{
		"wildcard": "/api/users/42/avatar",
		"globstar": "/files/a/b/c/metadata",
		"no match": "/static/app.js",
	} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				matcher.Match(http.MethodGet, path)
			}
		})
	}
}
