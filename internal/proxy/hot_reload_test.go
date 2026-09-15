package proxy_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestWatcherUpdatesProxyWithoutRestart(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	path := filepath.Join(t.TempDir(), "chaos.yaml")
	writeHotConfig(t, path, upstream.URL, http.StatusServiceUnavailable)
	configured, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	handler, err := proxy.NewHandler(configured, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	applied := make(chan struct{}, 1)
	watcher, err := config.NewWatcher(path, configured, func(candidate *config.Config) error {
		if err := handler.Update(candidate); err != nil {
			return err
		}
		applied <- struct{}{}
		return nil
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewWatcher() unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = watcher.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- watcher.Run(ctx) }()

	if got := serveStatus(handler); got != http.StatusServiceUnavailable {
		t.Fatalf("initial status = %d, want 503", got)
	}
	writeHotConfig(t, path, upstream.URL, http.StatusTeapot)
	select {
	case <-applied:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for proxy configuration update")
	}
	if got := serveStatus(handler); got != http.StatusTeapot {
		t.Fatalf("reloaded status = %d, want 418", got)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("watcher Run() error after cancellation: %v", err)
	}
}

func TestHandlerUpdatePreservesInFlightSnapshot(t *testing.T) {
	t.Parallel()

	requestReachedOldUpstream := make(chan struct{})
	releaseOldUpstream := make(chan struct{})
	oldUpstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(requestReachedOldUpstream)
		<-releaseOldUpstream
		writer.Header().Set("Content-Length", "10")
		_, _ = io.WriteString(writer, "0123456789")
	}))
	defer oldUpstream.Close()
	newUpstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "new")
	}))
	defer newUpstream.Close()

	handler, err := proxy.NewHandler(&config.Config{
		Target: oldUpstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:     "truncate-old",
			Match:    "GET /payload",
			Enabled:  true,
			Truncate: &config.TruncateConfig{Probability: 1, At: 0.5},
		}},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	oldResponse := httptest.NewRecorder()
	requestDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(oldResponse, httptest.NewRequest(http.MethodGet, "http://proxy.test/payload", nil))
		close(requestDone)
	}()
	select {
	case <-requestReachedOldUpstream:
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request did not reach old upstream")
	}

	if err := handler.Update(&config.Config{Target: newUpstream.URL, CORS: config.CORSPassthrough}); err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}
	close(releaseOldUpstream)
	select {
	case <-requestDone:
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request did not finish")
	}
	if got := oldResponse.Body.String(); got != "01234" {
		t.Fatalf("in-flight body = %q, want old truncate rule result", got)
	}

	newResponse := httptest.NewRecorder()
	handler.ServeHTTP(newResponse, httptest.NewRequest(http.MethodGet, "http://proxy.test/payload", nil))
	if got := newResponse.Body.String(); got != "new" {
		t.Fatalf("new request body = %q, want new runtime response", got)
	}
}

func TestHandlerUpdateKeepsLastRuntimeOnCompileError(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()
	handler, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "active",
			Match:   "GET /api",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 503, Probability: 1},
		}},
	}, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}

	err = handler.Update(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "broken",
			Match:   "not-a-path",
			Enabled: true,
			Status:  &config.StatusConfig{Code: 418, Probability: 1},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "compile rules") {
		t.Fatalf("Update() error = %v, want matcher compilation error", err)
	}
	if got := serveStatus(handler); got != http.StatusServiceUnavailable {
		t.Fatalf("status after rejected update = %d, want retained 503", got)
	}
}

func serveStatus(handler http.Handler) int {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://proxy.test/api", nil))
	return response.Code
}

func writeHotConfig(t *testing.T, path, target string, status int) {
	t.Helper()
	contents := "target: " + target + "\n" +
		"rules:\n" +
		"  - name: injected\n" +
		"    match: GET /api\n" +
		"    status:\n" +
		fmt.Sprintf("      code: %d\n", status) +
		"      probability: 1\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
