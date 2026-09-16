// Package proxy implements the HTTP data plane.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand/v2"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/faults"
)

// Handler applies a request-captured runtime configuration without locking the
// data-plane path.
type Handler struct {
	current          atomic.Pointer[runtimeConfig]
	ruleCounters     sync.Map
	scenarioCounters sync.Map
	proxy            *httputil.ReverseProxy
	logger           *log.Logger
	publisher        EventPublisher
	headerOverrides  bool
	stopping         context.Context
	stop             context.CancelFunc
}

var errShuttingDown = errors.New("proxy shutting down")

type requestState struct {
	runtime      *runtimeConfig
	request      *http.Request
	chain        []faults.Fault
	faultContext *faults.Context
	metrics      requestMetrics
}

type requestStateKey struct{}

// NewHandler validates and compiles cfg before accepting requests.
func NewHandler(cfg *config.Config, logger *log.Logger, options ...Option) (*Handler, error) {
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}
	runtime, err := compileRuntime(cfg, nil)
	if err != nil {
		return nil, err
	}

	h := &Handler{logger: logger}
	h.stopping, h.stop = context.WithCancel(context.Background())
	for _, option := range options {
		if option == nil {
			return nil, errors.New("handler option must not be nil")
		}
		if err := option(h); err != nil {
			return nil, err
		}
	}
	h.current.Store(runtime)
	h.proxy = &httputil.ReverseProxy{
		Rewrite:        h.rewrite,
		ModifyResponse: h.modifyResponse,
		ErrorHandler:   h.handleProxyError,
		ErrorLog:       logger,
		FlushInterval:  -1,
	}
	return h, nil
}

// ServeHTTP captures one immutable runtime snapshot for the complete request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	runtime := h.current.Load()
	state := &requestState{runtime: runtime}
	r = r.WithContext(context.WithValue(r.Context(), requestStateKey{}, state))
	state.request = r
	statusWriter := &statusResponseWriter{ResponseWriter: w}
	defer h.finishRequest(r, &state.metrics, statusWriter, started)

	if handlePreflight(statusWriter, r, runtime.cors) {
		return
	}

	selection, err := h.selectFaults(runtime, r)
	if err != nil {
		state.metrics.pipelineError = err
		writeJSONError(statusWriter, r, runtime.cors, http.StatusBadRequest, err.Error())
		return
	}
	state.metrics.rule = selection.rule
	if selection.applied != "" {
		statusWriter.Header().Set(appliedHeader, selection.applied)
		if runtime.cors.mode == config.CORSReflect {
			statusWriter.Header().Add("Access-Control-Expose-Headers", appliedHeader)
		}
	}

	if len(selection.chain) > 0 {
		faultRequest, releaseFaultRequest := h.withShutdownCancel(r)
		defer releaseFaultRequest()

		state.chain = selection.chain
		state.faultContext = &faults.Context{
			Req:  faultRequest,
			Rng:  rand.New(rand.NewPCG(deriveSeed(runtime.seed, selection.rule, selection.index), 0)),
			Rule: selection.rule,
			Emit: state.metrics.record,
		}

		shortCircuit, err := runBefore(state.faultContext, state.chain)
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			if h.stopping.Err() != nil {
				rejectDuringShutdown(statusWriter, state)
				return
			}
			state.metrics.pipelineError = err
			writeJSONError(statusWriter, r, runtime.cors, http.StatusInternalServerError, "fault injection failed")
			return
		}
		if shortCircuit != nil {
			if shortCircuit.Reset {
				resetConnection(statusWriter, state)
				return
			}
			if shortCircuit.Hang {
				<-faultRequest.Context().Done()
				if r.Context().Err() == nil {
					rejectDuringShutdown(statusWriter, state)
				}
				return
			}
			writeShortCircuit(statusWriter, r, runtime.cors, shortCircuit)
			return
		}
	}

	h.proxy.ServeHTTP(statusWriter, r)
}

// BeginShutdown interrupts injected hang and latency waits.
func (h *Handler) BeginShutdown() {
	h.stop()
}

func (h *Handler) withShutdownCancel(r *http.Request) (*http.Request, func()) {
	ctx, cancel := context.WithCancel(r.Context())
	stopWatching := context.AfterFunc(h.stopping, cancel)
	return r.WithContext(ctx), func() {
		stopWatching()
		cancel()
	}
}

// A zero linger makes Close send a TCP RST instead of a graceful FIN.
func resetConnection(w http.ResponseWriter, state *requestState) {
	conn, _, err := http.NewResponseController(w).Hijack()
	if err != nil {
		state.metrics.pipelineError = fmt.Errorf("reset aborted the stream because the connection cannot be hijacked: %w", err)
		panic(http.ErrAbortHandler)
	}
	if lingerer, ok := conn.(interface{ SetLinger(sec int) error }); ok {
		_ = lingerer.SetLinger(0)
	}
	_ = conn.Close()
}

func rejectDuringShutdown(w http.ResponseWriter, state *requestState) {
	state.metrics.pipelineError = errShuttingDown
	w.Header().Set("Connection", "close")
	writeJSONError(w, state.request, state.runtime.cors, http.StatusServiceUnavailable, "chaosproxy is shutting down")
}

func (h *Handler) rewrite(request *httputil.ProxyRequest) {
	state := stateFromRequest(request.In)
	state.metrics.upstreamStarted = time.Now()
	request.SetURL(state.runtime.target)
	request.SetXForwarded()
	if h.headerOverrides {
		request.Out.Header.Del(overrideHeader)
	}
}

func (h *Handler) modifyResponse(response *http.Response) error {
	state := stateFromRequest(response.Request)
	state.metrics.upstreamStatus = response.StatusCode
	state.metrics.upstreamLatency = time.Since(state.metrics.upstreamStarted)

	for _, fault := range state.chain {
		if err := fault.After(state.faultContext, response); err != nil {
			wrapped := fmt.Errorf("apply %s after hook: %w", fault.Name(), err)
			state.metrics.pipelineError = wrapped
			return wrapped
		}
	}
	applyResponseCORS(response.Header, state.request, state.runtime.cors)
	return nil
}

func (h *Handler) handleProxyError(w http.ResponseWriter, r *http.Request, err error) {
	state := stateFromRequest(r)
	state.metrics.proxyError = err
	if !state.metrics.upstreamStarted.IsZero() && state.metrics.upstreamLatency == 0 {
		state.metrics.upstreamLatency = time.Since(state.metrics.upstreamStarted)
	}
	writeJSONError(w, r, state.runtime.cors, http.StatusBadGateway, "upstream unavailable")
}

func runBefore(ctx *faults.Context, chain []faults.Fault) (*faults.ShortCircuit, error) {
	for _, fault := range chain {
		shortCircuit, err := fault.Before(ctx)
		if err != nil {
			return nil, fmt.Errorf("apply %s before hook: %w", fault.Name(), err)
		}
		if shortCircuit != nil {
			return shortCircuit, nil
		}
	}
	return nil, nil
}

func writeShortCircuit(
	w http.ResponseWriter,
	r *http.Request,
	cors corsPolicy,
	shortCircuit *faults.ShortCircuit,
) {
	copyHeaders(w.Header(), shortCircuit.Headers)
	applyResponseCORS(w.Header(), r, cors)
	status := shortCircuit.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	if len(shortCircuit.Body) > 0 {
		_, _ = w.Write(shortCircuit.Body)
	}
}

func writeJSONError(w http.ResponseWriter, r *http.Request, cors corsPolicy, status int, message string) {
	body, err := json.Marshal(struct {
		Error string `json:"error"`
	}{Error: message})
	if err != nil {
		body = []byte(`{"error":"internal error"}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	applyResponseCORS(w.Header(), r, cors)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func copyHeaders(destination, source http.Header) {
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}

func stateFromRequest(r *http.Request) *requestState {
	return r.Context().Value(requestStateKey{}).(*requestState)
}

func (m *requestMetrics) record(injection faults.Injection) {
	m.faults = append(m.faults, injection.Fault)
	m.injectedLatency += injection.Latency
}

func deriveSeed(seed int64, rule string, index uint64) uint64 {
	ruleHash := fnv.New64a()
	_, _ = ruleHash.Write([]byte(rule))
	return splitMix64(splitMix64(uint64(seed)^ruleHash.Sum64()) + index)
}

func splitMix64(value uint64) uint64 {
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return value ^ (value >> 31)
}
