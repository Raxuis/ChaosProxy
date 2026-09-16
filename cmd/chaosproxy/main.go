package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/Raxuis/chaosproxy/internal/config"
	"github.com/Raxuis/chaosproxy/internal/control"
	"github.com/Raxuis/chaosproxy/internal/events"
	"github.com/Raxuis/chaosproxy/internal/loopback"
	"github.com/Raxuis/chaosproxy/internal/proxy"
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
	eventBus := events.NewBus()
	handlerOptions := []proxy.Option{proxy.WithEventPublisher(eventBus)}
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
	if err := run(serverOptions{
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
	}); err != nil {
		log.Fatalf("chaosproxy stopped: %v", err)
	}
}
