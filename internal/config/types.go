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
	Target      string     `json:"target" yaml:"target"`
	Seed        int64      `json:"seed" yaml:"seed"`
	CORS        CORSMode   `json:"cors" yaml:"cors"`
	CORSOrigins []string   `json:"cors_origins,omitempty" yaml:"cors_origins,omitempty"`
	Rules       []Rule     `json:"rules" yaml:"rules"`
	Scenarios   []Scenario `json:"scenarios,omitempty" yaml:"scenarios,omitempty"`

	source sourceLocation
}

// ExhaustMode chooses what a scenario does after its last step.
type ExhaustMode string

const (
	ExhaustPassthrough ExhaustMode = "passthrough"
	ExhaustRepeat      ExhaustMode = "repeat"
	ExhaustLast        ExhaustMode = "last"
)

// Scenario plays one fault step per matching request, before rules apply.
type Scenario struct {
	Name        string      `json:"name" yaml:"name"`
	Match       string      `json:"match" yaml:"match"`
	Enabled     bool        `json:"enabled" yaml:"enabled"`
	OnExhausted ExhaustMode `json:"on_exhausted" yaml:"on_exhausted"`
	Steps       []string    `json:"steps" yaml:"steps"`

	source sourceLocation
}

// Rule associates an HTTP route pattern with one or more injected faults.
type Rule struct {
	Name      string           `json:"name" yaml:"name"`
	Match     string           `json:"match" yaml:"match"`
	Enabled   bool             `json:"enabled" yaml:"enabled"`
	Latency   *LatencyConfig   `json:"latency,omitempty" yaml:"latency,omitempty"`
	Status    *StatusConfig    `json:"status,omitempty" yaml:"status,omitempty"`
	Hang      *HangConfig      `json:"hang,omitempty" yaml:"hang,omitempty"`
	Truncate  *TruncateConfig  `json:"truncate,omitempty" yaml:"truncate,omitempty"`
	Reset     *ResetConfig     `json:"reset,omitempty" yaml:"reset,omitempty"`
	Bandwidth *BandwidthConfig `json:"bandwidth,omitempty" yaml:"bandwidth,omitempty"`
	Headers   *HeadersConfig   `json:"headers,omitempty" yaml:"headers,omitempty"`
	Mutate    *MutateConfig    `json:"mutate,omitempty" yaml:"mutate,omitempty"`
	Redirect  *RedirectConfig  `json:"redirect,omitempty" yaml:"redirect,omitempty"`
	Stall     *StallConfig     `json:"stall,omitempty" yaml:"stall,omitempty"`

	source sourceLocation
}

// HeadersConfig adds, overrides, or removes upstream response headers.
type HeadersConfig struct {
	Probability float64           `json:"probability" yaml:"probability"`
	Set         map[string]string `json:"set,omitempty" yaml:"set,omitempty"`
	Remove      []string          `json:"remove,omitempty" yaml:"remove,omitempty"`
}

// MutateConfig rewrites JSON response bodies that fit within MaxBytes.
type MutateConfig struct {
	Probability float64          `json:"probability" yaml:"probability"`
	MaxBytes    int64            `json:"max_bytes" yaml:"max_bytes"`
	Operations  []MutationConfig `json:"operations" yaml:"operations"`
}

// MutationConfig applies one operation to every value a dotted path selects.
type MutationConfig struct {
	Op     string `json:"op" yaml:"op"`
	Path   string `json:"path" yaml:"path"`
	Factor int    `json:"factor,omitempty" yaml:"factor,omitempty"`
}

// LatencyConfig configures a fixed or lognormal delay.
type LatencyConfig struct {
	Dist   string        `yaml:"dist"`
	Value  time.Duration `yaml:"value,omitempty"`
	Jitter time.Duration `yaml:"jitter,omitempty"`
	P50    time.Duration `yaml:"p50,omitempty"`
	P99    time.Duration `yaml:"p99,omitempty"`
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

// RedirectConfig configures an injected HTTP redirect.
type RedirectConfig struct {
	Probability float64 `json:"probability" yaml:"probability"`
	Code        int     `json:"code" yaml:"code"`
	Location    string  `json:"location" yaml:"location"`
}

// TruncateConfig configures a response body truncation.
type TruncateConfig struct {
	Probability float64 `json:"probability" yaml:"probability"`
	At          float64 `json:"at" yaml:"at"`
}

// ResetConfig configures a TCP connection reset before any response.
type ResetConfig struct {
	Probability float64 `json:"probability" yaml:"probability"`
}

// BandwidthConfig limits how fast the upstream response body reaches the client.
type BandwidthConfig struct {
	BytesPerSecond int64 `json:"bytes_per_second" yaml:"bytes_per_second"`
}

// StallConfig pauses the response body mid-stream for Duration after AfterBytes.
type StallConfig struct {
	Probability float64       `json:"probability" yaml:"probability"`
	AfterBytes  int64         `json:"after_bytes" yaml:"after_bytes"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
}

type rawConfig struct {
	Target      string        `yaml:"target"`
	Seed        int64         `yaml:"seed"`
	CORS        CORSMode      `yaml:"cors"`
	CORSOrigins []string      `yaml:"cors_origins"`
	Rules       []rawRule     `yaml:"rules"`
	Scenarios   []rawScenario `yaml:"scenarios"`
}

type rawScenario struct {
	Name        string      `yaml:"name"`
	Match       string      `yaml:"match"`
	Enabled     *bool       `yaml:"enabled"`
	OnExhausted ExhaustMode `yaml:"on_exhausted"`
	Steps       []string    `yaml:"steps"`
}

type rawRule struct {
	Name      string           `yaml:"name"`
	Match     string           `yaml:"match"`
	Enabled   *bool            `yaml:"enabled"`
	Latency   *LatencyConfig   `yaml:"latency"`
	Status    *StatusConfig    `yaml:"status"`
	Hang      *HangConfig      `yaml:"hang"`
	Truncate  *TruncateConfig  `yaml:"truncate"`
	Reset     *ResetConfig     `yaml:"reset"`
	Bandwidth *BandwidthConfig `yaml:"bandwidth"`
	Headers   *HeadersConfig   `yaml:"headers"`
	Mutate    *MutateConfig    `yaml:"mutate"`
	Redirect  *RedirectConfig  `yaml:"redirect"`
	Stall     *StallConfig     `yaml:"stall"`
}

type sourceLocation struct {
	line   int
	fields map[string]int
}
