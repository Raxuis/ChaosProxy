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

	handler := &Handler{
		bus:         bus,
		runtime:     runtime,
		heartbeat:   heartbeatInterval,
		mux:         http.NewServeMux(),
		crossOrigin: http.NewCrossOriginProtection(),
	}
	handler.stopping, handler.stop = context.WithCancel(context.Background())
	handler.routes()
	return handler, nil
}

// BeginShutdown ends open event streams.
func (handler *Handler) BeginShutdown() {
	handler.stop()
}

// ServeHTTP dispatches control-plane requests.
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !allowedHost(request.Host) {
		writeJSONError(writer, http.StatusForbidden, "host not allowed")
		return
	}
	if err := handler.crossOrigin.Check(request); err != nil {
		writeJSONError(writer, http.StatusForbidden, err.Error())
		return
	}
	handler.mux.ServeHTTP(writer, request)
}

// DNS rebinding attacks need a hostname, so IP literals stay usable from containers.
func allowedHost(hostport string) bool {
	host := hostport
	if split, _, err := net.SplitHostPort(hostport); err == nil {
		host = split
	}
	return loopback.IsHost(host) || net.ParseIP(host) != nil
}

func (handler *Handler) routes() {
	handler.mux.HandleFunc("GET /healthz", handler.health)
	handler.mux.HandleFunc("GET /api/events", handler.streamEvents)
	handler.mux.HandleFunc("GET /api/config", handler.getConfig)
	handler.mux.HandleFunc("PUT /api/rules/{name}", handler.setRuleEnabled)
	handler.mux.HandleFunc("POST /api/reset", handler.reset)
}

func (handler *Handler) health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok\n"))
}

func (handler *Handler) getConfig(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, handler.runtime.CurrentConfig())
}

func (handler *Handler) setRuleEnabled(writer http.ResponseWriter, request *http.Request) {
	var update struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(request, &update); err != nil {
		writeJSONError(writer, http.StatusBadRequest, err.Error())
		return
	}
	if update.Enabled == nil {
		writeJSONError(writer, http.StatusBadRequest, "enabled is required")
		return
	}

	name := request.PathValue("name")
	if err := handler.runtime.SetRuleEnabled(name, *update.Enabled); err != nil {
		if errors.Is(err, config.ErrRuleNotFound) {
			writeJSONError(writer, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(writer, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}{Name: name, Enabled: *update.Enabled})
}

func (handler *Handler) reset(writer http.ResponseWriter, _ *http.Request) {
	handler.runtime.Reset()
	handler.bus.Reset()
	writer.WriteHeader(http.StatusNoContent)
}
