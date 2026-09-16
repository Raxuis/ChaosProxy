package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
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
		}
	}

	attachSourceLocations(cfg, &document)
	return cfg, nil
}

func attachSourceLocations(cfg *Config, document *yaml.Node) {
	if len(document.Content) == 0 {
		return
	}

	root := document.Content[0]
	cfg.source = sourceLocation{
		line:   root.Line,
		fields: collectFieldLines(root, ""),
	}

	rulesNode := mappingValue(root, "rules")
	if rulesNode == nil || rulesNode.Kind != yaml.SequenceNode {
		return
	}
	for index, ruleNode := range rulesNode.Content {
		if index >= len(cfg.Rules) {
			break
		}
		cfg.Rules[index].source = sourceLocation{
			line:   ruleNode.Line,
			fields: collectFieldLines(ruleNode, ""),
		}
	}
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
