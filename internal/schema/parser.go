package schema

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseFile читает и разбирает файл system-gen.yaml в структуру Schema.
// Возвращает ошибку, если файл не удалось прочитать или YAML некорректен.
func ParseFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schema: read file %q: %w", path, err)
	}
	return Parse(data)
}

// Parse десериализует сырые байты YAML в структуру Schema.
func Parse(data []byte) (*Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("schema: parse yaml: %w", err)
	}
	return &s, nil
}
