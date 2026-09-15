package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type latencyJSON struct {
	Dist   string `json:"dist"`
	Value  string `json:"value,omitempty"`
	Jitter string `json:"jitter,omitempty"`
	P50    string `json:"p50,omitempty"`
	P99    string `json:"p99,omitempty"`
}

// MarshalJSON writes durations in the YAML notation, such as "800ms".
func (latency LatencyConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(latencyJSON{
		Dist:   latency.Dist,
		Value:  formatDuration(latency.Value),
		Jitter: formatDuration(latency.Jitter),
		P50:    formatDuration(latency.P50),
		P99:    formatDuration(latency.P99),
	})
}

// UnmarshalJSON reads durations written in the YAML notation.
func (latency *LatencyConfig) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var raw latencyJSON
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	decoded := LatencyConfig{Dist: raw.Dist}
	for _, field := range []struct {
		name   string
		raw    string
		target *time.Duration
	}{
		{name: "value", raw: raw.Value, target: &decoded.Value},
		{name: "jitter", raw: raw.Jitter, target: &decoded.Jitter},
		{name: "p50", raw: raw.P50, target: &decoded.P50},
		{name: "p99", raw: raw.P99, target: &decoded.P99},
	} {
		if field.raw == "" {
			continue
		}
		parsed, err := time.ParseDuration(field.raw)
		if err != nil {
			return fmt.Errorf("latency.%s: %w", field.name, err)
		}
		*field.target = parsed
	}
	*latency = decoded
	return nil
}

func formatDuration(duration time.Duration) string {
	if duration == 0 {
		return ""
	}
	return duration.String()
}
