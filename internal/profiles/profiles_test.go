package profiles_test

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Raxuis/chaosproxy/internal/profiles"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestEveryProfileCompilesWithATarget(t *testing.T) {
	t.Parallel()

	for _, profile := range profiles.List() {
		cfg, err := profiles.Load(profile.Name)
		if err != nil {
			t.Fatalf("Load(%q) unexpected error: %v", profile.Name, err)
		}
		cfg.Target = "http://127.0.0.1:9000"
		if _, err := proxy.NewHandler(cfg, log.New(io.Discard, "", 0)); err != nil {
			t.Errorf("profile %q does not compile: %v", profile.Name, err)
		}
		if len(cfg.Rules) == 0 || cfg.Rules[0].Match != "/**" || profile.Description == "" {
			t.Errorf("profile %q = %+v, want a described catch-all rule", profile.Name, cfg.Rules)
		}
	}
}

func TestListMatchesEmbeddedFiles(t *testing.T) {
	t.Parallel()

	var files []string
	err := fs.WalkDir(os.DirFS("."), ".", func(path string, entry fs.DirEntry, err error) error {
		if err == nil && filepath.Ext(path) == ".yaml" {
			files = append(files, strings.TrimSuffix(path, ".yaml"))
		}
		return err
	})
	if err != nil {
		t.Fatalf("list profile files: %v", err)
	}
	var listed []string
	for _, profile := range profiles.List() {
		listed = append(listed, profile.Name)
	}
	slices.Sort(files)
	slices.Sort(listed)
	if !slices.Equal(files, listed) {
		t.Fatalf("profile files %v do not match listed profiles %v", files, listed)
	}
}

func TestOutageProfileFailsEveryRequest(t *testing.T) {
	t.Parallel()

	cfg, err := profiles.Load("outage")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	cfg.Target = "http://127.0.0.1:9000"
	handler, err := proxy.NewHandler(cfg, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://proxy.test/any/path", nil))
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Retry-After") != "30" {
		t.Fatalf("outage response = %d Retry-After %q, want 503 and 30", response.Code, response.Header().Get("Retry-After"))
	}
}

func TestLoadRejectsUnknownProfiles(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"missing", "../outage", "outage.yaml"} {
		if _, err := profiles.Load(name); err == nil || !strings.Contains(err.Error(), "use slow-network, flaky-api") {
			t.Errorf("Load(%q) error = %v, want the list of profiles", name, err)
		}
	}
}
