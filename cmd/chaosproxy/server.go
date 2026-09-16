package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

type serverOptions struct {
	host                 string
	dataPort             int
	controlPort          int
	target               string
	seed                 int64
	dataHandler          http.Handler
	controlHandler       http.Handler
	beginDataShutdown    func()
	beginControlShutdown func()
	watchConfig          func(context.Context) error
	logger               *log.Logger
	stop                 context.Context
}

type namedServer struct {
	name    string
	details string
	server  *http.Server
}

type serverResult struct {
	name string
	err  error
}

func run(options serverOptions) error {
	servers := []namedServer{
		{
			name:    "data plane",
			details: fmt.Sprintf(", forwarding to %s, seed=%d", options.target, options.seed),
			server: &http.Server{
				Addr:              net.JoinHostPort(options.host, strconv.Itoa(options.dataPort)),
				Handler:           options.dataHandler,
				ReadHeaderTimeout: 10 * time.Second,
			},
		},
		{
			name: "control plane",
			server: &http.Server{
				Addr:              net.JoinHostPort(options.host, strconv.Itoa(options.controlPort)),
				Handler:           options.controlHandler,
				ReadHeaderTimeout: 10 * time.Second,
			},
		},
	}
	if options.beginDataShutdown != nil {
		servers[0].server.RegisterOnShutdown(options.beginDataShutdown)
	}
	if options.beginControlShutdown != nil {
		servers[1].server.RegisterOnShutdown(options.beginControlShutdown)
	}

	parent := options.stop
	if parent == nil {
		parent = context.Background()
	}
	applicationContext, cancelApplication := context.WithCancel(parent)
	defer cancelApplication()
	signalContext, stopSignals := signal.NotifyContext(
		applicationContext,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	serverResults := make(chan serverResult, len(servers))
	for _, srv := range servers {
		go serve(srv, options.logger, serverResults)
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
	for _, srv := range servers {
		if err := srv.server.Shutdown(shutdownContext); err != nil {
			errorsSeen = append(errorsSeen, fmt.Errorf("shutdown %s: %w", srv.name, err))
			_ = srv.server.Close()
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

func serve(srv namedServer, logger *log.Logger, results chan<- serverResult) {
	logger.Printf("%s listening on %s%s", srv.name, srv.server.Addr, srv.details)
	err := srv.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	results <- serverResult{name: srv.name, err: err}
}
