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

func run(port int, target string, seed int64, handler http.Handler, logger *log.Logger) error {
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("data plane listening on %s, forwarding to %s, seed=%d", server.Addr, target, seed)
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErrors <- err
	}()

	signalContext, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	select {
	case err := <-serverErrors:
		return err
	case <-signalContext.Done():
		logger.Printf("shutdown requested")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	shutdownErr := server.Shutdown(shutdownContext)
	serveErr := <-serverErrors
	if shutdownErr != nil {
		_ = server.Close()
	}
	return errors.Join(shutdownErr, serveErr)
}
