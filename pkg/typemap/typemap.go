// Package typemap translates schema type names to language-specific types.
package typemap

import "fmt"

// GoType returns the Go type string for a given schema type.
// Returns an error for unknown types so callers can surface it clearly.
func GoType(schemaType string) (string, error) {
	m := map[string]string{
		"string":    "string",
		"int":       "int64",
		"int32":     "int32",
		"int64":     "int64",
		"float":     "float64",
		"float32":   "float32",
		"float64":   "float64",
		"bool":      "bool",
		"uuid":      "string",
		"time":      "time.Time",
		"time.Time": "time.Time",
		"byte":      "byte",
		"[]byte":    "[]byte",
	}
	if t, ok := m[schemaType]; ok {
		return t, nil
	}
	return "", fmt.Errorf("typemap: unknown type %q", schemaType)
}

// GoZeroValue returns the zero-value literal for a given schema type.
func GoZeroValue(schemaType string) string {
	m := map[string]string{
		"string":    `""`,
		"int":       "0",
		"int32":     "0",
		"int64":     "0",
		"float":     "0.0",
		"float32":   "0.0",
		"float64":   "0.0",
		"bool":      "false",
		"uuid":      `""`,
		"time":      "time.Time{}",
		"time.Time": "time.Time{}",
		"byte":      "0",
		"[]byte":    "nil",
	}
	if z, ok := m[schemaType]; ok {
		return z
	}
	return "nil"
}

// NeedsTimeImport returns true if any field requires "time" import.
func NeedsTimeImport(types []string) bool {
	for _, t := range types {
		if t == "time" || t == "time.Time" {
			return true
		}
	}
	return false
}
