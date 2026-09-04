package main

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantTarget string
		wantPort   int
		wantDelay  time.Duration
		wantError  string
	}{
		{
			name:       "defaults",
			args:       []string{"--target", "http://localhost:9000"},
			wantTarget: "http://localhost:9000",
			wantPort:   7070,
		},
		{
			name:       "all flags",
			args:       []string{"--target", "https://api.example.com/v1", "--port", "9090", "--delay", "250ms"},
			wantTarget: "https://api.example.com/v1",
			wantPort:   9090,
			wantDelay:  250 * time.Millisecond,
		},
		{name: "missing target", wantError: "--target is required"},
		{name: "positional argument", args: []string{"--target", "http://localhost", "extra"}, wantError: "unexpected positional arguments"},
		{name: "zero port", args: []string{"--target", "http://localhost", "--port", "0"}, wantError: "--port must be between"},
		{name: "large port", args: []string{"--target", "http://localhost", "--port", "65536"}, wantError: "--port must be between"},
		{name: "negative delay", args: []string{"--target", "http://localhost", "--delay", "-1s"}, wantError: "--delay must not be negative"},
		{name: "unsupported scheme", args: []string{"--target", "ftp://example.com"}, wantError: "absolute HTTP or HTTPS"},
		{name: "missing host", args: []string{"--target", "http:///api"}, wantError: "absolute HTTP or HTTPS"},
		{name: "credentials", args: []string{"--target", "http://user:pass@example.com"}, wantError: "must not contain user information"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseOptions(test.args, io.Discard)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("parseOptions() error = %v, want error containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOptions() unexpected error: %v", err)
			}

			if got.target.String() != test.wantTarget {
				t.Errorf("target = %q, want %q", got.target, test.wantTarget)
			}
			if got.port != test.wantPort {
				t.Errorf("port = %d, want %d", got.port, test.wantPort)
			}
			if got.delay != test.wantDelay {
				t.Errorf("delay = %s, want %s", got.delay, test.wantDelay)
			}
		})
	}
}

func TestParseOptionsHelp(t *testing.T) {
	t.Parallel()

	_, err := parseOptions([]string{"--help"}, io.Discard)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("parseOptions() error = %v, want flag.ErrHelp", err)
	}
}
