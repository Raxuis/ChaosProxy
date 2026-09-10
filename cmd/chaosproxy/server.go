package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

type serverOptions struct {
	dataPort       int
	controlPort    int
	target         string
	seed           int64
	dataHandler    http.Handler
	controlHandler http.Handler
	watchConfig    func(context.Context) error
	logger         *log.Logger
}

type namedServer struct {
	name   string
	server *http.Server
}

type serverResult struct {
	name string
	err  error
}

func run(options serverOptions) error {
	servers := []namedServer{
		{
			name: "data plane",
			server: &http.Server{
				Addr:              ":" + strconv.Itoa(options.dataPort),
				Handler:           options.dataHandler,
				ReadHeaderTimeout: 10 * time.Second,
			},
		},
		{
			name: "control plane",
			server: &http.Server{
				Addr:              ":" + strconv.Itoa(options.controlPort),
				Handler:           options.controlHandler,
				ReadHeaderTimeout: 10 * time.Second,
			},
		},
	}

	applicationContext, cancelApplication := context.WithCancel(context.Background())
	defer cancelApplication()
	signalContext, stopSignals := signal.NotifyContext(
		applicationContext,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	serverResults := make(chan serverResult, len(servers))
	for _, configured := range servers {
		go serve(configured, options, serverResults)
	}

	var watcherResults chan error
	if options.watchConfig != nil {
		watcherResults = make(chan error, 1)
		go func() {
			watcherResults <- options.watchConfig(signalContext)
		}()
	}

	resultsRead := 0
	watcherRead := options.watchConfig == nil
	errorsSeen := make([]error, 0, len(servers)+2)
	select {
	case result := <-serverResults:
		resultsRead++
		if result.err != nil {
			errorsSeen = append(errorsSeen, fmt.Errorf("%s: %w", result.name, result.err))
		}
	case err := <-watcherResults:
		watcherRead = true
		if err == nil && signalContext.Err() == nil {
			err = errors.New("config watcher stopped unexpectedly")
		}
		if err != nil {
			errorsSeen = append(errorsSeen, err)
		}
	case <-signalContext.Done():
		options.logger.Printf("shutdown requested")
	}
	cancelApplication()

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	for _, configured := range servers {
		if err := configured.server.Shutdown(shutdownContext); err != nil {
			errorsSeen = append(errorsSeen, fmt.Errorf("shutdown %s: %w", configured.name, err))
			_ = configured.server.Close()
		}
	}
	cancelShutdown()

	for resultsRead < len(servers) {
		result := <-serverResults
		resultsRead++
		if result.err != nil {
			errorsSeen = append(errorsSeen, fmt.Errorf("%s: %w", result.name, result.err))
		}
	}
	if !watcherRead {
		if err := <-watcherResults; err != nil {
			errorsSeen = append(errorsSeen, err)
		}
	}
	return errors.Join(errorsSeen...)
}

func serve(configured namedServer, options serverOptions, results chan<- serverResult) {
	if configured.name == "data plane" {
		options.logger.Printf(
			"data plane listening on %s, forwarding to %s, seed=%d",
			configured.server.Addr,
			options.target,
			options.seed,
		)
	} else {
		options.logger.Printf("control plane listening on %s", configured.server.Addr)
	}
	err := configured.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	results <- serverResult{name: configured.name, err: err}
}
