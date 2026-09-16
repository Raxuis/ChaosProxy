package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"

	"github.com/Raxuis/chaosproxy/internal/config"
)

const (
	defaultHost        = "127.0.0.1"
	defaultPort        = 7070
	defaultControlPort = 7071
)

type options struct {
	configPath      string
	target          string
	host            string
	port            int
	controlPort     int
	seed            optionalInt64
	generatedSeed   int64
	headerOverrides bool
	reportPath      string
	maxRequests     int
	exitOnError     bool
}

type optionalInt64 struct {
	value int64
	set   bool
}

func (o *optionalInt64) Set(raw string) error {
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("parse integer: %w", err)
	}
	o.value = parsed
	o.set = true
	return nil
}

func (o *optionalInt64) String() string {
	if o == nil || !o.set {
		return ""
	}
	return strconv.FormatInt(o.value, 10)
}

func parseOptions(args []string, output io.Writer) (options, error) {
	flags := flag.NewFlagSet("chaosproxy", flag.ContinueOnError)
	flags.SetOutput(output)

	var opts options
	flags.StringVar(&opts.configPath, "config", "", "path to a chaos YAML configuration")
	flags.StringVar(&opts.target, "target", "", "upstream base URL (overrides config)")
	flags.StringVar(&opts.host, "host", defaultHost, "listen address for both planes")
	flags.IntVar(&opts.port, "port", defaultPort, "data-plane listen port")
	flags.IntVar(&opts.controlPort, "control-port", defaultControlPort, "control-plane listen port")
	flags.Var(&opts.seed, "seed", "random seed (overrides config)")
	flags.BoolVar(&opts.headerOverrides, "header-overrides", false, "let clients force faults with the X-Chaos request header")
	flags.StringVar(&opts.reportPath, "report", "", "write a JSON run report to PATH on shutdown, or - for standard output")
	flags.IntVar(&opts.maxRequests, "max-requests", 0, "shut down after this many data-plane requests (0 means no limit)")
	flags.BoolVar(&opts.exitOnError, "exit-on-error", false, "shut down and exit with status 1 when a request fails inside the proxy")

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if opts.configPath == "" && opts.target == "" {
		return options{}, errors.New("either --config or --target is required")
	}
	if opts.maxRequests < 0 {
		return options{}, errors.New("--max-requests must not be negative")
	}
	if opts.host == "" {
		return options{}, errors.New("--host must not be empty")
	}
	if opts.port < 1 || opts.port > 65535 {
		return options{}, errors.New("--port must be between 1 and 65535")
	}
	if opts.controlPort < 1 || opts.controlPort > 65535 {
		return options{}, errors.New("--control-port must be between 1 and 65535")
	}
	if opts.controlPort == opts.port {
		return options{}, errors.New("--port and --control-port must be different")
	}
	return opts, nil
}

func resolveConfig(opts *options) (*config.Config, error) {
	var cfg *config.Config
	if opts.configPath == "" {
		cfg = &config.Config{CORS: config.CORSPassthrough}
	} else {
		loaded, err := config.Load(opts.configPath)
		if err != nil {
			return nil, err
		}
		cfg = loaded
	}

	opts.generatedSeed = rand.Int64()
	applyOverrides(cfg, *opts)
	return cfg, nil
}

func applyOverrides(cfg *config.Config, opts options) {
	if opts.target != "" {
		cfg.Target = opts.target
	}
	switch {
	case opts.seed.set:
		cfg.Seed = opts.seed.value
	case !cfg.SeedConfigured():
		cfg.Seed = opts.generatedSeed
	}
}
