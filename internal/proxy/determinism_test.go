package proxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

const decisionCount = 32

func TestRuleDecisionsIgnoreUnrelatedTraffic(t *testing.T) {
	t.Parallel()

	upstream := newNoContentUpstream(t)
	isolated := newDeterminismHandler(t, upstream.URL)
	want := make([]int, 0, decisionCount)
	for range decisionCount {
		want = append(want, statusFor(isolated, "/flaky"))
	}
	if !slices.Contains(want, http.StatusServiceUnavailable) || !slices.Contains(want, http.StatusNoContent) {
		t.Fatalf("decisions = %v, want a mix of injected and forwarded responses", want)
	}

	interleaved := newDeterminismHandler(t, upstream.URL)
	got := make([]int, 0, decisionCount)
	for range decisionCount {
		statusFor(interleaved, "/unmatched")
		statusFor(interleaved, "/twin")
		got = append(got, statusFor(interleaved, "/flaky"))
	}
	if !slices.Equal(got, want) {
		t.Fatalf("decisions with unrelated traffic = %v, want %v", got, want)
	}
}

func TestRulesWithIdenticalFaultsDrawIndependently(t *testing.T) {
	t.Parallel()

	handler := newDeterminismHandler(t, newNoContentUpstream(t).URL)
	flaky := make([]int, 0, decisionCount)
	twin := make([]int, 0, decisionCount)
	for range decisionCount {
		flaky = append(flaky, statusFor(handler, "/flaky"))
		twin = append(twin, statusFor(handler, "/twin"))
	}
	if slices.Equal(flaky, twin) {
		t.Fatalf("identical rules produced the same decisions %v", flaky)
	}
}

func TestResetReplaysRuleDecisions(t *testing.T) {
	t.Parallel()

	handler := newDeterminismHandler(t, newNoContentUpstream(t).URL)
	first := make([]int, 0, decisionCount)
	for range decisionCount {
		first = append(first, statusFor(handler, "/flaky"))
	}
	handler.Reset()
	replayed := make([]int, 0, decisionCount)
	for range decisionCount {
		replayed = append(replayed, statusFor(handler, "/flaky"))
	}
	if !slices.Equal(replayed, first) {
		t.Fatalf("decisions after reset = %v, want %v", replayed, first)
	}
}

func newDeterminismHandler(t *testing.T, target string) *proxy.Handler {
	t.Helper()

	handler, err := proxy.NewHandler(&config.Config{
		Target: target,
		Seed:   42,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{
			{Name: "flaky", Match: "GET /flaky", Enabled: true, Status: &config.StatusConfig{Code: 503, Probability: 0.5}},
			{Name: "twin", Match: "GET /twin", Enabled: true, Status: &config.StatusConfig{Code: 503, Probability: 0.5}},
		},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	return handler
}

func newNoContentUpstream(t *testing.T) *httptest.Server {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)
	return upstream
}

func statusFor(handler http.Handler, path string) int {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://proxy.test"+path, nil))
	return response.Code
}
