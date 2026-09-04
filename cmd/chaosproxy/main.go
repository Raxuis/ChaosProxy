package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Raxuis/chaosproxy/internal/proxy"
)

const (
	defaultPort     = 7070
	shutdownTimeout = 10 * time.Second
)

type options struct {
	target *url.URL
	port   int
	delay  time.Duration
}

func main() {
	opts, err := parseOptions(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		log.Fatalf("invalid arguments: %v", err)
	}

	if err := run(opts); err != nil {
		log.Fatalf("chaosproxy stopped: %v", err)
	}
}

func parseOptions(args []string, output io.Writer) (options, error) {
	flags := flag.NewFlagSet("chaosproxy", flag.ContinueOnError)
	flags.SetOutput(output)

	targetValue := flags.String("target", "", "upstream base URL (required)")
	port := flags.Int("port", defaultPort, "data-plane listen port")
	delay := flags.Duration("delay", 0, "delay applied before forwarding each request")

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if *targetValue == "" {
		return options{}, errors.New("--target is required")
	}
	if *port < 1 || *port > 65535 {
		return options{}, errors.New("--port must be between 1 and 65535")
	}
	if *delay < 0 {
		return options{}, errors.New("--delay must not be negative")
	}

	target, err := url.Parse(*targetValue)
	if err != nil {
		return options{}, fmt.Errorf("parse --target: %w", err)
	}
	if (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return options{}, errors.New("--target must be an absolute HTTP or HTTPS URL")
	}
	if target.User != nil {
		return options{}, errors.New("--target must not contain user information")
	}

	return options{
		target: target,
		port:   *port,
		delay:  *delay,
	}, nil
}

func run(opts options) error {
	logger := log.Default()
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(opts.port),
		Handler:           proxy.NewHandler(opts.target, opts.delay, logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Printf("data plane listening on %s, forwarding to %s", server.Addr, opts.target)
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
