package proxy_test

import (
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestRuleToggleSurvivesUnrelatedReload(t *testing.T) {
	t.Parallel()

	target := newNoContentUpstream(t).URL
	handler := newToggleHandler(t, toggleConfig(target, true, 503))
	mustSetRuleEnabled(t, handler, false)

	mustUpdate(t, handler, toggleConfig(target, true, 500))
	if got := statusFor(handler, "/api"); got != http.StatusNoContent {
		t.Fatalf("status after reload = %d, want disabled rule forwarding 204", got)
	}
	if active := handler.CurrentConfig().Rules[0]; active.Enabled || active.Status.Code != 500 {
		t.Fatalf("active rule = %+v, want reloaded status code with toggle kept", active)
	}
}

func TestFileEnabledChangeReplacesToggle(t *testing.T) {
	t.Parallel()

	target := newNoContentUpstream(t).URL
	handler := newToggleHandler(t, toggleConfig(target, true, 503))
	mustSetRuleEnabled(t, handler, false)

	mustUpdate(t, handler, toggleConfig(target, false, 503))
	mustUpdate(t, handler, toggleConfig(target, true, 503))
	if got := statusFor(handler, "/api"); got != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want file re-enabling the rule to win", got)
	}
}

func TestRuleToggleDroppedWhenRuleRemoved(t *testing.T) {
	t.Parallel()

	target := newNoContentUpstream(t).URL
	handler := newToggleHandler(t, toggleConfig(target, true, 503))
	mustSetRuleEnabled(t, handler, false)

	mustUpdate(t, handler, &config.Config{Target: target, CORS: config.CORSPassthrough})
	mustUpdate(t, handler, toggleConfig(target, true, 503))
	if got := statusFor(handler, "/api"); got != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want re-added rule to start from the file", got)
	}
}

func TestToggleBackToFileValueClearsOverride(t *testing.T) {
	t.Parallel()

	target := newNoContentUpstream(t).URL
	handler := newToggleHandler(t, toggleConfig(target, true, 503))
	mustSetRuleEnabled(t, handler, false)
	mustSetRuleEnabled(t, handler, true)

	mustUpdate(t, handler, toggleConfig(target, false, 503))
	mustUpdate(t, handler, toggleConfig(target, false, 500))
	if got := statusFor(handler, "/api"); got != http.StatusNoContent {
		t.Fatalf("status = %d, want the file's disabled rule", got)
	}
}

func TestRuleToggleSurvivesConcurrentReloads(t *testing.T) {
	t.Parallel()

	target := newNoContentUpstream(t).URL
	for attempt := range 50 {
		handler := newToggleHandler(t, toggleConfig(target, true, 503))
		reloaded := make(chan struct{})
		go func() {
			defer close(reloaded)
			for code := range 50 {
				if err := handler.Update(toggleConfig(target, true, 500+code)); err != nil {
					t.Errorf("Update() unexpected error: %v", err)
					return
				}
			}
		}()
		mustSetRuleEnabled(t, handler, false)
		<-reloaded

		if handler.CurrentConfig().Rules[0].Enabled {
			t.Fatalf("attempt %d: concurrent reload discarded the toggle", attempt)
		}
	}
}

func newToggleHandler(t *testing.T, configured *config.Config) *proxy.Handler {
	t.Helper()

	handler, err := proxy.NewHandler(configured, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	return handler
}

func toggleConfig(target string, enabled bool, code int) *config.Config {
	return &config.Config{
		Target: target,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "unavailable",
			Match:   "GET /api",
			Enabled: enabled,
			Status:  &config.StatusConfig{Code: code, Probability: 1},
		}},
	}
}

func mustSetRuleEnabled(t *testing.T, handler *proxy.Handler, enabled bool) {
	t.Helper()
	if err := handler.SetRuleEnabled("unavailable", enabled); err != nil {
		t.Fatalf("SetRuleEnabled(%t) unexpected error: %v", enabled, err)
	}
}

func mustUpdate(t *testing.T, handler *proxy.Handler, configured *config.Config) {
	t.Helper()
	if err := handler.Update(configured); err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
}
