package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

const (
	defaultMutateMaxBytes = 1 << 20
	defaultMutateFactor   = 10
)

// Load parses one YAML document from path. Semantic validation is deliberately
// separate so hot reload callers can distinguish syntax and validation errors.
func Load(path string) (*Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	parsed, err := decode(contents)
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	return parsed, nil
}

// Parse decodes one YAML configuration document without validating it.
func Parse(contents []byte) (*Config, error) {
	return decode(contents)
}

// SeedConfigured reports whether the loaded YAML sets seed explicitly, even to zero.
func (c *Config) SeedConfigured() bool {
	_, found := c.source.fields["seed"]
	return found
}

func decode(contents []byte) (*Config, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)

	var raw rawConfig
	if err := decoder.Decode(&raw); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("configuration is empty")
		}
		return nil, err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("configuration must contain exactly one YAML document")
	}

	var document yaml.Node
	if err := yaml.Unmarshal(contents, &document); err != nil {
		return nil, err
	}

	cfg := &Config{
		Target:      raw.Target,
		Seed:        raw.Seed,
		CORS:        raw.CORS,
		CORSOrigins: raw.CORSOrigins,
		Rules:       make([]Rule, len(raw.Rules)),
	}
	if cfg.CORS == "" {
		cfg.CORS = CORSReflect
	}
	for index, rawRule := range raw.Rules {
		enabled := true
		if rawRule.Enabled != nil {
			enabled = *rawRule.Enabled
		}
		cfg.Rules[index] = Rule{
			Name:      rawRule.Name,
			Match:     rawRule.Match,
			Enabled:   enabled,
			Latency:   rawRule.Latency,
			Status:    rawRule.Status,
			Hang:      rawRule.Hang,
			Truncate:  rawRule.Truncate,
			Reset:     rawRule.Reset,
			Bandwidth: rawRule.Bandwidth,
			Headers:   rawRule.Headers,
			Mutate:    rawRule.Mutate,
		}
		applyMutateDefaults(rawRule.Mutate)
	}

	for _, rawScenario := range raw.Scenarios {
		scenario := Scenario{
			Name:        rawScenario.Name,
			Match:       rawScenario.Match,
			Enabled:     rawScenario.Enabled == nil || *rawScenario.Enabled,
			OnExhausted: rawScenario.OnExhausted,
			Steps:       rawScenario.Steps,
		}
		if scenario.OnExhausted == "" {
			scenario.OnExhausted = ExhaustPassthrough
		}
		cfg.Scenarios = append(cfg.Scenarios, scenario)
	}

	attachSourceLocations(cfg, &document)
	return cfg, nil
}

func applyMutateDefaults(mutate *MutateConfig) {
	if mutate == nil {
		return
	}
	if mutate.MaxBytes == 0 {
		mutate.MaxBytes = defaultMutateMaxBytes
	}
	for index := range mutate.Operations {
		operation := &mutate.Operations[index]
		if operation.Factor == 0 && (operation.Op == "inflate" || operation.Op == "stretch") {
			operation.Factor = defaultMutateFactor
		}
	}
}

func attachSourceLocations(cfg *Config, document *yaml.Node) {
	if len(document.Content) == 0 {
		return
	}

	root := document.Content[0]
	cfg.source = locate(root)
	for index, node := range sequenceItems(root, "rules") {
		if index < len(cfg.Rules) {
			cfg.Rules[index].source = locate(node)
		}
	}
	for index, node := range sequenceItems(root, "scenarios") {
		if index < len(cfg.Scenarios) {
			cfg.Scenarios[index].source = locate(node)
		}
	}
}

func locate(node *yaml.Node) sourceLocation {
	return sourceLocation{line: node.Line, fields: collectFieldLines(node, "")}
}

func sequenceItems(root *yaml.Node, key string) []*yaml.Node {
	node := mappingValue(root, key)
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	return node.Content
}

func collectFieldLines(node *yaml.Node, prefix string) map[string]int {
	lines := make(map[string]int)
	if node == nil || node.Kind != yaml.MappingNode {
		return lines
	}

	for index := 0; index+1 < len(node.Content); index += 2 {
		key := node.Content[index]
		value := node.Content[index+1]
		path := key.Value
		if prefix != "" {
			path = prefix + "." + path
		}
		lines[path] = value.Line
		if value.Kind == yaml.SequenceNode {
			for itemIndex, item := range value.Content {
				lines[fmt.Sprintf("%s[%d]", path, itemIndex)] = item.Line
			}
		}
		for nestedPath, line := range collectFieldLines(value, path) {
			lines[nestedPath] = line
		}
	}

	return lines
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}
