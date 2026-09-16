package proxy

import (
	"strings"
	"testing"
)

func TestParseOverride(t *testing.T) {
	t.Parallel()

	valid := map[string]string{
		"status=503":                      "status=503",
		" latency=800ms ;status=429 ":     "latency=800ms; status=429",
		"latency=1.5s":                    "latency=1.5s",
		"bandwidth=1024; truncate=0.5":    "truncate=0.5; bandwidth=1024",
		"reset":                           "reset",
		"hang":                            "hang",
		"off":                             "off",
		"latency=0s; reset; bandwidth=64": "latency=0s; reset; bandwidth=64",
	}
	for header, want := range valid {
		if _, applied, err := parseOverride(header); err != nil || applied != want {
			t.Errorf("parseOverride(%q) = %q, %v; want %q", header, applied, err, want)
		}
	}

	invalid := map[string]string{
		"":                        "empty directive",
		"status=503;":             "empty directive",
		"status":                  "status needs a value, for example status=503",
		"status=abc":              "status must be an HTTP status code",
		"status=700":              "status must be an HTTP status code",
		"latency=-1s":             "latency must be a non-negative duration",
		"latency=soon":            "latency must be a non-negative duration",
		"truncate=1.5":            "truncate must be a fraction between 0 and 1",
		"truncate=NaN":            "truncate must be a fraction between 0 and 1",
		"bandwidth=0":             "bandwidth must be a positive number",
		"hang=1":                  "hang takes no value",
		"teapot":                  `unknown fault "teapot"`,
		"Status=503":              `unknown fault "Status"`,
		"status=503; status=500":  "status is set more than once",
		"off; status=503":         "off cannot be combined with faults",
		"status=503; reset":       "use only one",
		"hang; latency=1s; reset": "use only one",
	}
	for header, want := range invalid {
		if _, _, err := parseOverride(header); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("parseOverride(%q) error = %v, want error containing %q", header, err, want)
		}
	}
}
