// Package profiles embeds ready-made fault configurations for common failure modes.
package profiles

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"github.com/Raxuis/chaosproxy/internal/config"
)

//go:embed *.yaml
var files embed.FS

// Profile describes one built-in fault configuration.
type Profile struct {
	Name        string
	Description string
}

var profiles = []Profile{
	{Name: "slow-network", Description: "lognormal latency (p50 400ms, p99 3s) and 128 KB/s bandwidth"},
	{Name: "flaky-api", Description: "150ms ±100ms latency and 10% 503 responses with Retry-After: 2"},
	{Name: "connection-drops", Description: "5% TCP resets and 10% of bodies truncated at half"},
	{Name: "broken-payloads", Description: "25% of JSON bodies with nested fields set to null, 5% truncated"},
	{Name: "outage", Description: "every request fails with 503 and Retry-After: 30"},
}

// List returns the built-in profiles in a stable order.
func List() []Profile {
	return slices.Clone(profiles)
}

// Load parses a built-in profile. Profiles have no target, so callers must set one.
func Load(name string) (*config.Config, error) {
	if !slices.ContainsFunc(profiles, func(profile Profile) bool { return profile.Name == name }) {
		names := make([]string, len(profiles))
		for index, profile := range profiles {
			names[index] = profile.Name
		}
		return nil, fmt.Errorf("unknown profile %q; use %s", name, strings.Join(names, ", "))
	}
	contents, err := files.ReadFile(name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("read profile %q: %w", name, err)
	}
	cfg, err := config.Parse(contents)
	if err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", name, err)
	}
	return cfg, nil
}
