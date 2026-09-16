package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/control"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/loopback"
	"github.com/Raxuis/chaosproxy/internal/proxy"
	"github.com/Raxuis/chaosproxy/internal/report"
)

func main() {
	opts, err := parseOptions(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		log.Fatalf("invalid arguments: %v", err)
	}

	cfg, err := resolveConfig(&opts)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	if !opts.seed.set && !cfg.SeedConfigured() {
		log.Printf("no seed configured; generated seed %d, replay this run with --seed %d", cfg.Seed, cfg.Seed)
	}

	stopRun, stop := context.WithCancelCause(context.Background())
	recorderOptions := report.Options{MaxRequests: uint64(opts.maxRequests)}
	if opts.maxRequests > 0 {
		recorderOptions.OnMaxRequests = func() {
			log.Printf("reached --max-requests %d; shutting down", opts.maxRequests)
			stop(errMaxRequests)
		}
	}
	if opts.exitOnError {
		recorderOptions.OnError = func(event events.Event) {
			log.Printf("--exit-on-error: %s %s failed: %s; shutting down", event.Method, event.Path, event.Error)
			stop(errRequestFailed)
		}
	}
	recorder := report.NewRecorder(recorderOptions)

	eventBus := events.NewBus()
	handlerOptions := []proxy.Option{proxy.WithEventPublisher(publishers{eventBus, recorder})}
	if opts.headerOverrides {
		handlerOptions = append(handlerOptions, proxy.WithHeaderOverrides())
		log.Printf("header overrides enabled: clients can force faults with the X-Chaos header")
	}
	handler, err := proxy.NewHandler(cfg, log.Default(), handlerOptions...)
	if err != nil {
		log.Fatalf("initialize proxy: %v", err)
	}
	controlHandler, err := control.NewHandler(eventBus, handler)
	if err != nil {
		log.Fatalf("initialize control plane: %v", err)
	}

	var watchConfig func(context.Context) error
	if opts.configPath != "" {
		watcher, err := config.NewWatcher(
			opts.configPath,
			cfg,
			func(candidate *config.Config) error {
				applyOverrides(candidate, opts)
				return handler.Update(candidate)
			},
			log.Default(),
		)
		if err != nil {
			log.Fatalf("initialize config watcher: %v", err)
		}
		watchConfig = watcher.Run
	}

	if !loopback.IsHost(opts.host) {
		log.Printf("warning: --host %s exposes the proxy and its unauthenticated control plane to the network", opts.host)
	}
	runErr := run(serverOptions{
		host:                 opts.host,
		dataPort:             opts.port,
		controlPort:          opts.controlPort,
		target:               cfg.Target,
		seed:                 cfg.Seed,
		dataHandler:          handler,
		controlHandler:       controlHandler,
		beginDataShutdown:    handler.BeginShutdown,
		beginControlShutdown: controlHandler.BeginShutdown,
		watchConfig:          watchConfig,
		logger:               log.Default(),
		stop:                 stopRun,
	})

	exitCode := 0
	if runErr != nil {
		log.Printf("chaosproxy stopped: %v", runErr)
		exitCode = 1
	}
	if opts.reportPath != "" {
		current := handler.CurrentConfig()
		runReport := recorder.Report(time.Now(), report.Run{
			Target:    current.Target,
			Seed:      current.Seed,
			StoppedBy: stopReason(stopRun, runErr),
			Scenarios: handler.Scenarios(),
		})
		if err := report.WriteFile(opts.reportPath, runReport); err != nil {
			log.Printf("write report: %v", err)
			exitCode = 1
		} else if opts.reportPath != "-" {
			log.Printf("wrote report to %s", opts.reportPath)
		}
	}
	if failed := recorder.ErrorCount(); opts.exitOnError && failed > 0 {
		log.Printf("--exit-on-error: %d requests failed inside the proxy", failed)
		exitCode = 1
	}
	stop(nil)
	os.Exit(exitCode)
}
