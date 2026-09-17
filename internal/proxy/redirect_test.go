package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerInjectsRedirect(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamRequests.Add(1)
	}))
	defer upstream.Close()

	handler := newTestHandler(t, upstream.URL, config.CORSReflect, []config.Rule{{
		Name:    "expired-session",
		Match:   "GET /session",
		Enabled: true,
		Redirect: &config.RedirectConfig{
			Code:        http.StatusFound,
			Location:    "/login",
			Probability: 1,
		},
	}})

	request := httptest.NewRequest(http.MethodGet, "http://proxy.test/session", nil)
	request.Header.Set("Origin", "http://localhost:3001")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusFound || upstreamRequests.Load() != 0 {
		t.Fatalf("status/upstream requests = %d/%d, want 302/0", response.Code, upstreamRequests.Load())
	}
	if got := response.Header().Get("Location"); got != "/login" {
		t.Errorf("Location = %q, want /login", got)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3001" {
		t.Errorf("synthetic response did not reflect Origin: %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	exposeHeaders := strings.Join(response.Header().Values("Access-Control-Expose-Headers"), ",")
	if !strings.Contains(exposeHeaders, "Location") {
		t.Errorf("Access-Control-Expose-Headers = %q, want Location", exposeHeaders)
	}
}

func TestHandlerInjectsRedirectHeaderOverride(t *testing.T) {
	t.Parallel()

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	handler := newOverrideHandler(t, upstream.URL, 42, proxy.WithHeaderOverrides())

	response := serveWithHeaders(handler, "/items", map[string]string{
		"X-Chaos": "redirect=307:https://auth.example.com/login",
	})

	if response.Code != http.StatusTemporaryRedirect || upstreamRequests.Load() != 0 {
		t.Fatalf("status/upstream requests = %d/%d, want 307/0", response.Code, upstreamRequests.Load())
	}
	if got := response.Header().Get("Location"); got != "https://auth.example.com/login" {
		t.Errorf("Location = %q, want https://auth.example.com/login", got)
	}
	applied := response.Header().Get("X-Chaos-Applied")
	if applied != "redirect=307:https://auth.example.com/login" {
		t.Errorf("X-Chaos-Applied = %q, want redirect=307:https://auth.example.com/login", applied)
	}
}
