// Package control implements the HTTP control plane.
package control

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/loopback"
)

const heartbeatInterval = 15 * time.Second

// Runtime is the control seam for reading and changing active proxy state.
type Runtime interface {
	CurrentConfig() *config.Config
	SetRuleEnabled(name string, enabled bool) error
	Reset()
}

// Handler exposes runtime control and observability over HTTP.
type Handler struct {
	bus         *events.Bus
	runtime     Runtime
	heartbeat   time.Duration
	mux         *http.ServeMux
	crossOrigin *http.CrossOriginProtection
	stopping    context.Context
	stop        context.CancelFunc
}

// NewHandler creates a control-plane HTTP handler.
func NewHandler(bus *events.Bus, runtime Runtime) (*Handler, error) {
	if bus == nil {
		return nil, errors.New("event bus must not be nil")
	}
	if runtime == nil {
		return nil, errors.New("runtime must not be nil")
	}

	h := &Handler{
		bus:         bus,
		runtime:     runtime,
		heartbeat:   heartbeatInterval,
		mux:         http.NewServeMux(),
		crossOrigin: http.NewCrossOriginProtection(),
	}
	h.stopping, h.stop = context.WithCancel(context.Background())
	h.routes()
	return h, nil
}

// BeginShutdown ends open event streams.
func (h *Handler) BeginShutdown() {
	h.stop()
}

// ServeHTTP dispatches control-plane requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !allowedHost(r.Host) {
		writeJSONError(w, http.StatusForbidden, "host not allowed")
		return
	}
	if err := h.crossOrigin.Check(r); err != nil {
		writeJSONError(w, http.StatusForbidden, err.Error())
		return
	}
	h.mux.ServeHTTP(w, r)
}

// DNS rebinding attacks need a hostname, so IP literals stay usable from containers.
func allowedHost(hostport string) bool {
	host := hostport
	if split, _, err := net.SplitHostPort(hostport); err == nil {
		host = split
	}
	return loopback.IsHost(host) || net.ParseIP(host) != nil
}

func (h *Handler) routes() {
	h.mux.HandleFunc("GET /healthz", h.health)
	h.mux.HandleFunc("GET /api/events", h.streamEvents)
	h.mux.HandleFunc("GET /api/config", h.getConfig)
	h.mux.HandleFunc("PUT /api/rules/{name}", h.setRuleEnabled)
	h.mux.HandleFunc("POST /api/reset", h.reset)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (h *Handler) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.runtime.CurrentConfig())
}

func (h *Handler) setRuleEnabled(w http.ResponseWriter, r *http.Request) {
	var update struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(r, &update); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if update.Enabled == nil {
		writeJSONError(w, http.StatusBadRequest, "enabled is required")
		return
	}

	name := r.PathValue("name")
	if err := h.runtime.SetRuleEnabled(name, *update.Enabled); err != nil {
		if errors.Is(err, config.ErrRuleNotFound) {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}{Name: name, Enabled: *update.Enabled})
}

func (h *Handler) reset(w http.ResponseWriter, _ *http.Request) {
	h.runtime.Reset()
	h.bus.Reset()
	w.WriteHeader(http.StatusNoContent)
}
