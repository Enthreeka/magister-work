package layers

import (
	"strings"

	"github.com/Enthreeka/magister-work/pkg/typemap"
)

// templateFuncs returns the standard set of template helper functions
// used across all Go layer templates.
func templateFuncs() map[string]any {
	return map[string]any{
		"ToCamel":  toCamel,
		"ToSnake":  toSnake,
		"ToTitle":  strings.Title,
		"ToUpper":  strings.ToUpper,
		"ToLower":  strings.ToLower,
		"ToGoType": toGoType,
	}
}

// toCamel converts snake_case or any lower string to CamelCase.
func toCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// toSnake converts CamelCase to snake_case.
func toSnake(s string) string {
	var out []rune
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out = append(out, '_')
			}
			out = append(out, r+32)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// toGoType translates a schema type string to a Go type string.
// Falls back to the original string on unknown types.
func toGoType(schemaType string) string {
	t, err := typemap.GoType(schemaType)
	if err != nil {
		return schemaType
	}
	return t
}
