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

	configured, err := resolveConfig(opts)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	eventBus := events.NewBus()
	handler, err := proxy.NewHandler(configured, log.Default(), proxy.WithEventPublisher(eventBus))
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
			configured,
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

	if err := run(serverOptions{
		dataPort:             opts.port,
		controlPort:          opts.controlPort,
		target:               configured.Target,
		seed:                 configured.Seed,
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
