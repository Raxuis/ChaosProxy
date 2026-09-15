package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/Raxuis/chaosproxy/internal/config"
)

const (
	defaultHost        = "127.0.0.1"
	defaultPort        = 7070
	defaultControlPort = 7071
)

type options struct {
	configPath  string
	target      string
	host        string
	port        int
	controlPort int
	seed        optionalInt64
}

type optionalInt64 struct {
	value int64
	set   bool
}

func (value *optionalInt64) Set(raw string) error {
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("parse integer: %w", err)
	}
	value.value = parsed
	value.set = true
	return nil
}

func (value *optionalInt64) String() string {
	if value == nil || !value.set {
		return ""
	}
	return strconv.FormatInt(value.value, 10)
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

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if opts.configPath == "" && opts.target == "" {
		return options{}, errors.New("either --config or --target is required")
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

func resolveConfig(opts options) (*config.Config, error) {
	var configured *config.Config
	if opts.configPath == "" {
		configured = &config.Config{CORS: config.CORSPassthrough}
	} else {
		loaded, err := config.Load(opts.configPath)
		if err != nil {
			return nil, err
		}
		configured = loaded
	}

	applyOverrides(configured, opts)
	return configured, nil
}

func applyOverrides(configured *config.Config, opts options) {
	if opts.target != "" {
		configured.Target = opts.target
	}
	if opts.seed.set {
		configured.Seed = opts.seed.value
	}
}
