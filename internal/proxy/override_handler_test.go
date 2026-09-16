package proxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHeaderOverrideForcesFaultsForOneRequest(t *testing.T) {
	t.Parallel()

	upstream := newRecordingUpstream(t)
	handler := newOverrideHandler(t, upstream.URL, 42, proxy.WithHeaderOverrides())

	forced := serveWithHeaders(handler, "/api/orders", map[string]string{"X-Chaos": "status=418", "Origin": "http://localhost:3001"})
	if forced.Code != http.StatusTeapot || forced.Header().Get("X-Chaos-Applied") != "status=418" {
		t.Fatalf("forced response = %d applied %q, want 418 status=418", forced.Code, forced.Header().Get("X-Chaos-Applied"))
	}
	if !strings.Contains(strings.Join(forced.Header().Values("Access-Control-Expose-Headers"), ","), "X-Chaos-Applied") {
		t.Errorf("Access-Control-Expose-Headers = %q, want X-Chaos-Applied", forced.Header().Values("Access-Control-Expose-Headers"))
	}
	if upstream.count() != 0 {
		t.Fatalf("upstream requests = %d, want 0 for a forced status", upstream.count())
	}

	off := serveWithHeaders(handler, "/api/orders", map[string]string{"X-Chaos": "off"})
	if off.Code != http.StatusNoContent || off.Header().Get("X-Chaos-Applied") != "off" {
		t.Fatalf("off response = %d applied %q, want upstream 204 with off", off.Code, off.Header().Get("X-Chaos-Applied"))
	}
	if header := upstream.lastOverrideHeader(); header != "" {
		t.Errorf("upstream received X-Chaos %q, want the header removed", header)
	}

	started := time.Now()
	slow := serveWithHeaders(handler, "/unmatched", map[string]string{"X-Chaos": "latency=40ms"})
	if elapsed := time.Since(started); elapsed < 40*time.Millisecond || slow.Code != http.StatusNoContent {
		t.Fatalf("latency override = %d after %s, want 204 after at least 40ms", slow.Code, elapsed)
	}

	if ruled := serveWithHeaders(handler, "/api/orders", nil); ruled.Code != http.StatusServiceUnavailable || ruled.Header().Get("X-Chaos-Applied") != "" {
		t.Fatalf("request without header = %d applied %q, want rule 503 without X-Chaos-Applied", ruled.Code, ruled.Header().Get("X-Chaos-Applied"))
	}

	invalid := serveWithHeaders(handler, "/api/orders", map[string]string{"X-Chaos": "teapot"})
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "invalid X-Chaos header: unknown fault") {
		t.Fatalf("invalid header = %d %s, want explicit 400", invalid.Code, invalid.Body.String())
	}
}

func TestHeaderOverrideDisabledPassesHeaderThrough(t *testing.T) {
	t.Parallel()

	upstream := newRecordingUpstream(t)
	handler := newOverrideHandler(t, upstream.URL, 42)

	ruled := serveWithHeaders(handler, "/api/orders", map[string]string{"X-Chaos": "status=418"})
	if ruled.Code != http.StatusServiceUnavailable || ruled.Header().Get("X-Chaos-Applied") != "disabled" {
		t.Fatalf("disabled override = %d applied %q, want rule 503 with disabled", ruled.Code, ruled.Header().Get("X-Chaos-Applied"))
	}

	forwarded := serveWithHeaders(handler, "/unmatched", map[string]string{"X-Chaos": "status=418"})
	if forwarded.Code != http.StatusNoContent || upstream.lastOverrideHeader() != "status=418" {
		t.Fatalf("disabled override forwarded = %d with upstream header %q, want 204 and the header kept", forwarded.Code, upstream.lastOverrideHeader())
	}
}

func TestHeaderOverridesDoNotShiftRuleDecisions(t *testing.T) {
	t.Parallel()

	upstream := newRecordingUpstream(t)
	plain := newFlakyHandler(t, upstream.URL, 42)
	want := make([]int, 0, decisionCount)
	for range decisionCount {
		want = append(want, statusFor(plain, "/flaky"))
	}

	overridden := newFlakyHandler(t, upstream.URL, 42, proxy.WithHeaderOverrides())
	got := make([]int, 0, decisionCount)
	for range decisionCount {
		serveWithHeaders(overridden, "/flaky", map[string]string{"X-Chaos": "status=500"})
		got = append(got, statusFor(overridden, "/flaky"))
	}
	if !slices.Equal(got, want) {
		t.Fatalf("decisions with overrides interleaved = %v, want %v", got, want)
	}
}

func TestSameSeedReplaysAcrossHandlers(t *testing.T) {
	t.Parallel()

	upstream := newRecordingUpstream(t)
	decisions := func(seed int64) []int {
		handler := newFlakyHandler(t, upstream.URL, seed)
		statuses := make([]int, 0, decisionCount)
		for range decisionCount {
			statuses = append(statuses, statusFor(handler, "/flaky"))
		}
		return statuses
	}

	first := decisions(42)
	if replay := decisions(42); !slices.Equal(replay, first) {
		t.Fatalf("same seed decisions = %v, want %v", replay, first)
	}
	if other := decisions(43); slices.Equal(other, first) {
		t.Fatalf("seeds 42 and 43 produced the same decisions %v", first)
	}
}

type recordingUpstream struct {
	*httptest.Server
	mu        sync.Mutex
	requests  int
	lastChaos string
}

func newRecordingUpstream(t *testing.T) *recordingUpstream {
	t.Helper()

	upstream := &recordingUpstream{}
	upstream.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.mu.Lock()
		upstream.requests++
		upstream.lastChaos = r.Header.Get("X-Chaos")
		upstream.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)
	return upstream
}

func (u *recordingUpstream) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.requests
}

func (u *recordingUpstream) lastOverrideHeader() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.lastChaos
}

func newOverrideHandler(t *testing.T, target string, seed int64, options ...proxy.Option) http.Handler {
	t.Helper()
	return newTestProxy(t, &config.Config{
		Target: target,
		Seed:   seed,
		CORS:   config.CORSReflect,
		Rules: []config.Rule{{
			Name:    "unavailable",
			Match:   "GET /api/**",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 1},
		}},
	}, options...)
}

func newFlakyHandler(t *testing.T, target string, seed int64, options ...proxy.Option) http.Handler {
	t.Helper()
	return newTestProxy(t, &config.Config{
		Target: target,
		Seed:   seed,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "flaky",
			Match:   "GET /flaky",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 0.5},
		}},
	}, options...)
}

func newTestProxy(t *testing.T, cfg *config.Config, options ...proxy.Option) http.Handler {
	t.Helper()
	handler, err := proxy.NewHandler(cfg, log.New(io.Discard, "", 0), options...)
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	return handler
}

func serveWithHeaders(handler http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "http://proxy.test"+path, nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
