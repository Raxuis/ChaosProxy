package control_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/control"
	"github.com/Raxuis/chaosproxy/internal/events"
)

func TestDashboardServesEmbeddedAssets(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	handler, err := control.NewHandler(bus, newRuntime(t, "http://localhost:9000", bus))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/", contentType: "text/html", contains: "<title>chaosproxy control</title>"},
		{path: "/app.js", contentType: "text/javascript", contains: `new EventSource("api/events")`},
		{path: "/feed.js", contentType: "text/javascript", contains: "export function matchesFilter"},
		{path: "/style.css", contentType: "text/css", contains: "--canvas"},
		{path: "/favicon.svg", contentType: "image/svg+xml", contains: "<svg"},
	}
	for _, test := range tests {
		response := serveControl(handler, http.MethodGet, test.path, "")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", test.path, response.Code)
		}
		if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, test.contentType) {
			t.Errorf("GET %s Content-Type = %q, want %s", test.path, got, test.contentType)
		}
		if !strings.Contains(response.Body.String(), test.contains) {
			t.Errorf("GET %s body does not contain %q", test.path, test.contains)
		}
		if policy := response.Header().Get("Content-Security-Policy"); !strings.Contains(policy, "script-src 'self'") {
			t.Errorf("GET %s Content-Security-Policy = %q, want self-only scripts", test.path, policy)
		}
	}

	for _, path := range []string{"/feed.test.js", "/package.json", "/embed.go"} {
		if response := serveControl(handler, http.MethodGet, path, ""); response.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404 for a file outside the dashboard", path, response.Code)
		}
	}
}

func TestStatsEndpointReportsTotals(t *testing.T) {
	t.Parallel()

	bus := events.NewBus()
	handler, err := control.NewHandler(bus, newRuntime(t, "http://localhost:9000", bus))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	bus.Publish(events.Event{Rule: "injected", Faults: []string{"status"}, Status: 503})
	bus.Publish(events.Event{Status: 200})

	response := serveControl(handler, http.MethodGet, "/api/stats", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET /api/stats status = %d, want 200", response.Code)
	}
	var stats events.Stats
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Published != 2 || stats.Faulted != 1 || stats.Statuses["5xx"] != 1 || stats.Rules["injected"].Faulted != 1 {
		t.Fatalf("stats = %+v, want 2 published and 1 injected 503", stats)
	}
}
