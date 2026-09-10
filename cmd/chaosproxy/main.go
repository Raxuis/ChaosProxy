package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"

	"github.com/Raxuis/chaosproxy/internal/config"
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
	handler, err := proxy.NewHandler(configured, log.Default())
	if err != nil {
		log.Fatalf("initialize proxy: %v", err)
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

	if err := run(opts.port, configured.Target, configured.Seed, handler, watchConfig, log.Default()); err != nil {
		log.Fatalf("chaosproxy stopped: %v", err)
	}
}
