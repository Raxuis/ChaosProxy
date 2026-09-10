// Package config loads and validates Chaos Proxy configuration.
package config

import "time"

// CORSMode controls how the data plane handles cross-origin headers.
type CORSMode string

const (
	CORSReflect     CORSMode = "reflect"
	CORSPassthrough CORSMode = "passthrough"
	CORSOff         CORSMode = "off"
)

// Config is the complete file-backed configuration.
type Config struct {
	Target string   `json:"target" yaml:"target"`
	Seed   int64    `json:"seed" yaml:"seed"`
	CORS   CORSMode `json:"cors" yaml:"cors"`
	Rules  []Rule   `json:"rules" yaml:"rules"`

	source sourceLocation
}

// Rule associates an HTTP route pattern with one or more injected faults.
type Rule struct {
	Name     string          `json:"name" yaml:"name"`
	Match    string          `json:"match" yaml:"match"`
	Enabled  bool            `json:"enabled" yaml:"enabled"`
	Latency  *LatencyConfig  `json:"latency,omitempty" yaml:"latency,omitempty"`
	Status   *StatusConfig   `json:"status,omitempty" yaml:"status,omitempty"`
	Hang     *HangConfig     `json:"hang,omitempty" yaml:"hang,omitempty"`
	Truncate *TruncateConfig `json:"truncate,omitempty" yaml:"truncate,omitempty"`

	source sourceLocation
}

// LatencyConfig configures a fixed or lognormal delay.
type LatencyConfig struct {
	Dist   string        `json:"dist" yaml:"dist"`
	Value  time.Duration `json:"value,omitempty" yaml:"value,omitempty"`
	Jitter time.Duration `json:"jitter,omitempty" yaml:"jitter,omitempty"`
	P50    time.Duration `json:"p50,omitempty" yaml:"p50,omitempty"`
	P99    time.Duration `json:"p99,omitempty" yaml:"p99,omitempty"`
}

// StatusConfig configures an injected HTTP response status.
type StatusConfig struct {
	Code        int     `json:"code" yaml:"code"`
	Probability float64 `json:"probability" yaml:"probability"`
	RetryAfter  int     `json:"retry_after,omitempty" yaml:"retry_after,omitempty"`
}

// HangConfig configures a request that waits until its context is canceled.
type HangConfig struct {
	Probability float64 `json:"probability" yaml:"probability"`
}

// TruncateConfig configures a response body truncation.
type TruncateConfig struct {
	Probability float64 `json:"probability" yaml:"probability"`
	At          float64 `json:"at" yaml:"at"`
}

type rawConfig struct {
	Target string    `yaml:"target"`
	Seed   int64     `yaml:"seed"`
	CORS   CORSMode  `yaml:"cors"`
	Rules  []rawRule `yaml:"rules"`
}

type rawRule struct {
	Name     string          `yaml:"name"`
	Match    string          `yaml:"match"`
	Enabled  *bool           `yaml:"enabled"`
	Latency  *LatencyConfig  `yaml:"latency"`
	Status   *StatusConfig   `yaml:"status"`
	Hang     *HangConfig     `yaml:"hang"`
	Truncate *TruncateConfig `yaml:"truncate"`
}

type sourceLocation struct {
	line   int
	fields map[string]int
}
