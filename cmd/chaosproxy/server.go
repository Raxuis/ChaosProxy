package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

func run(
	port int,
	target string,
	seed int64,
	handler http.Handler,
	watchConfig func(context.Context) error,
	logger *log.Logger,
) error {
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	applicationContext, cancelApplication := context.WithCancel(context.Background())
	defer cancelApplication()
	signalContext, stopSignals := signal.NotifyContext(
		applicationContext,
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("data plane listening on %s, forwarding to %s, seed=%d", server.Addr, target, seed)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErrors <- err
	}()

	var watcherErrors chan error
	if watchConfig != nil {
		watcherErrors = make(chan error, 1)
		go func() {
			watcherErrors <- watchConfig(signalContext)
		}()
	}

	serverFinished := false
	watcherFinished := watchConfig == nil
	var serveErr error
	var watcherErr error
	select {
	case serveErr = <-serverErrors:
		serverFinished = true
	case watcherErr = <-watcherErrors:
		watcherFinished = true
		if watcherErr == nil && signalContext.Err() == nil {
			watcherErr = errors.New("config watcher stopped unexpectedly")
		}
	case <-signalContext.Done():
		logger.Printf("shutdown requested")
	}
	cancelApplication()

	var shutdownErr error
	if !serverFinished {
		shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
		shutdownErr = server.Shutdown(shutdownContext)
		cancelShutdown()
		if shutdownErr != nil {
			_ = server.Close()
		}
		serveErr = <-serverErrors
	}
	if !watcherFinished {
		watcherErr = <-watcherErrors
	}

	return errors.Join(shutdownErr, serveErr, watcherErr)
}
