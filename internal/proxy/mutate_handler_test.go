package proxy_test

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/proxy"
)

func TestHandlerMutatesJSONResponses(t *testing.T) {
	t.Parallel()

	payload := `{"user":{"name":"Ada","email":"ada@example.com"}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()

	var logs lockedBuffer
	handler, err := proxy.NewHandler(&config.Config{
		Target: upstream.URL,
		CORS:   config.CORSPassthrough,
		Rules: []config.Rule{{
			Name:    "broken-user",
			Match:   "GET /api/user",
			Enabled: true,
			Mutate: &config.MutateConfig{Probability: 1, MaxBytes: 1 << 20, Operations: []config.MutationConfig{
				{Op: "nullify", Path: "user.email"},
				{Op: "drop", Path: "user.phone"},
			}},
		}},
	}, log.New(&logs, "", 0))
	if err != nil {
		t.Fatalf("NewHandler() unexpected error: %v", err)
	}
	proxyServer := httptest.NewServer(handler)
	defer proxyServer.Close()

	response, err := http.Get(proxyServer.URL + "/api/user")
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body with the rewritten Content-Length: %v", err)
	}
	if response.ContentLength != int64(len(body)) {
		t.Errorf("ContentLength = %d, want %d", response.ContentLength, len(body))
	}

	var decoded struct {
		User map[string]any `json:"user"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode body %s: %v", body, err)
	}
	if email, present := decoded.User["email"]; !present || email != nil || decoded.User["name"] != "Ada" {
		t.Fatalf("user = %+v, want a null email and the original name", decoded.User)
	}

	deadline := time.Now().Add(time.Second)
	for !strings.Contains(logs.String(), `details="mutate drop user.phone matched nothing"`) {
		if time.Now().After(deadline) {
			t.Fatalf("log %q does not record the no-op", logs.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !strings.Contains(logs.String(), `faults="mutate"`) {
		t.Errorf("log %q does not record the mutate fault", logs.String())
	}
}
