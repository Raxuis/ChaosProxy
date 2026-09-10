package main

import (
	"errors"
	"flag"
	"log"
	"os"

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

	if err := run(opts.port, configured.Target, configured.Seed, handler, log.Default()); err != nil {
		log.Fatalf("chaosproxy stopped: %v", err)
	}
}
