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
	current      atomic.Pointer[runtimeConfig]
	ruleCounters sync.Map
	proxy        *httputil.ReverseProxy
	logger       *log.Logger
	publisher    EventPublisher
	stopping     context.Context
	stop         context.CancelFunc
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

// NewHandler validates and compiles configured before accepting requests.
func NewHandler(configured *config.Config, logger *log.Logger, options ...Option) (*Handler, error) {
	if logger == nil {
		return nil, errors.New("logger must not be nil")
	}
	runtime, err := compileRuntime(configured, nil)
	if err != nil {
		return nil, err
	}

	handler := &Handler{logger: logger}
	handler.stopping, handler.stop = context.WithCancel(context.Background())
	for _, option := range options {
		if option == nil {
			return nil, errors.New("handler option must not be nil")
		}
		if err := option(handler); err != nil {
			return nil, err
		}
	}
	handler.current.Store(runtime)
	handler.proxy = &httputil.ReverseProxy{
		Rewrite:        handler.rewrite,
		ModifyResponse: handler.modifyResponse,
		ErrorHandler:   handler.handleProxyError,
		ErrorLog:       logger,
		FlushInterval:  -1,
	}
	return handler, nil
}

// ServeHTTP captures one immutable runtime snapshot for the complete request.
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	started := time.Now()
	runtime := handler.current.Load()
	state := &requestState{runtime: runtime}
	request = request.WithContext(context.WithValue(request.Context(), requestStateKey{}, state))
	state.request = request
	statusWriter := &statusResponseWriter{ResponseWriter: writer}
	defer handler.finishRequest(request, &state.metrics, statusWriter, started)

	if handlePreflight(statusWriter, request, runtime.cors) {
		return
	}

	matched := runtime.match.Match(request.Method, request.URL.Path)
	if matched != nil {
		faultRequest, releaseFaultRequest := handler.withShutdownCancel(request)
		defer releaseFaultRequest()

		state.metrics.rule = matched.Name
		state.chain = runtime.chains[matched.Name]
		state.faultContext = &faults.Context{
			Req:  faultRequest,
			Rng:  rand.New(rand.NewPCG(deriveSeed(runtime.seed, matched.Name, handler.nextRuleIndex(matched.Name)), 0)),
			Rule: matched.Name,
			Emit: state.metrics.record,
		}

		shortCircuit, err := runBefore(state.faultContext, state.chain)
		if err != nil {
			if request.Context().Err() != nil {
				return
			}
			if handler.stopping.Err() != nil {
				rejectDuringShutdown(statusWriter, state)
				return
			}
			state.metrics.pipelineError = err
			writeJSONError(statusWriter, request, runtime.cors, http.StatusInternalServerError, "fault injection failed")
			return
		}
		if shortCircuit != nil {
			if shortCircuit.Hang {
				<-faultRequest.Context().Done()
				if request.Context().Err() == nil {
					rejectDuringShutdown(statusWriter, state)
				}
				return
			}
			writeShortCircuit(statusWriter, request, runtime.cors, shortCircuit)
			return
		}
	}

	handler.proxy.ServeHTTP(statusWriter, request)
}

// BeginShutdown interrupts injected hang and latency waits.
func (handler *Handler) BeginShutdown() {
	handler.stop()
}

func (handler *Handler) withShutdownCancel(request *http.Request) (*http.Request, func()) {
	ctx, cancel := context.WithCancel(request.Context())
	stopWatching := context.AfterFunc(handler.stopping, cancel)
	return request.WithContext(ctx), func() {
		stopWatching()
		cancel()
	}
}

func rejectDuringShutdown(writer http.ResponseWriter, state *requestState) {
	state.metrics.pipelineError = errShuttingDown
	writer.Header().Set("Connection", "close")
	writeJSONError(writer, state.request, state.runtime.cors, http.StatusServiceUnavailable, "chaosproxy is shutting down")
}

func (handler *Handler) rewrite(request *httputil.ProxyRequest) {
	state := stateFromRequest(request.In)
	state.metrics.upstreamStarted = time.Now()
	request.SetURL(state.runtime.target)
	request.SetXForwarded()
}

func (handler *Handler) modifyResponse(response *http.Response) error {
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

func (handler *Handler) handleProxyError(writer http.ResponseWriter, request *http.Request, err error) {
	state := stateFromRequest(request)
	state.metrics.proxyError = err
	if !state.metrics.upstreamStarted.IsZero() && state.metrics.upstreamLatency == 0 {
		state.metrics.upstreamLatency = time.Since(state.metrics.upstreamStarted)
	}
	writeJSONError(writer, request, state.runtime.cors, http.StatusBadGateway, "upstream unavailable")
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
	writer http.ResponseWriter,
	request *http.Request,
	cors corsPolicy,
	shortCircuit *faults.ShortCircuit,
) {
	copyHeaders(writer.Header(), shortCircuit.Headers)
	applyResponseCORS(writer.Header(), request, cors)
	status := shortCircuit.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}
	writer.WriteHeader(status)
	if len(shortCircuit.Body) > 0 {
		_, _ = writer.Write(shortCircuit.Body)
	}
}

func writeJSONError(writer http.ResponseWriter, request *http.Request, cors corsPolicy, status int, message string) {
	body, err := json.Marshal(struct {
		Error string `json:"error"`
	}{Error: message})
	if err != nil {
		body = []byte(`{"error":"internal error"}`)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	applyResponseCORS(writer.Header(), request, cors)
	writer.WriteHeader(status)
	_, _ = writer.Write(body)
}

func copyHeaders(destination, source http.Header) {
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}

func stateFromRequest(request *http.Request) *requestState {
	return request.Context().Value(requestStateKey{}).(*requestState)
}

func (metrics *requestMetrics) record(injection faults.Injection) {
	metrics.faults = append(metrics.faults, injection.Fault)
	metrics.injectedLatency += injection.Latency
}

func (handler *Handler) nextRuleIndex(rule string) uint64 {
	counter, found := handler.ruleCounters.Load(rule)
	if !found {
		counter, _ = handler.ruleCounters.LoadOrStore(rule, new(atomic.Uint64))
	}
	return counter.(*atomic.Uint64).Add(1) - 1
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
