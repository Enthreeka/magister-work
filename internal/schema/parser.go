package schema

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and parses a system-gen.yaml file into a Schema.
// It returns an error if the file cannot be read or the YAML is malformed.
func ParseFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schema: read file %q: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals raw YAML bytes into a Schema.
func Parse(data []byte) (*Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("schema: parse yaml: %w", err)
	}
	return &s, nil
}
