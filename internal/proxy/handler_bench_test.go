package proxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func BenchmarkHandlerInjectsStatus(b *testing.B) {
	handler, err := proxy.NewHandler(&config.Config{
		Target: "http://127.0.0.1:1",
		Seed:   42,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{
			{Name: "orders", Match: "POST /api/orders", Enabled: true, Status: &config.StatusConfig{Code: 503, Probability: 1}},
			{Name: "avatar", Match: "GET /api/users/*/avatar", Enabled: true, Status: &config.StatusConfig{Code: 503, Probability: 0.9999}},
		},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		b.Fatalf("NewHandler() unexpected error: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/api/users/42/avatar", nil)

	b.ReportAllocs()
	for b.Loop() {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}
