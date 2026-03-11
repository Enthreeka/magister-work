package schema

import (
	"fmt"
	"strings"
)

// ValidationError описывает единственное нарушение схемы.
type ValidationError struct {
	Path    string // например, "input[0].type"
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Validate проверяет схему на согласованность и полноту.
// Все обнаруженные ошибки возвращаются вместе; функция не останавливается на первой.
func Validate(s *Schema) []ValidationError {
	var errs []ValidationError

	errs = append(errs, validateMeta(s)...)
	errs = append(errs, validateTransport(s)...)
	errs = append(errs, validateFields("input", s.Input)...)
	errs = append(errs, validateFields("output", s.Output)...)
	errs = append(errs, validateRepository(s)...)
	errs = append(errs, validateGenerate(s)...)

	return errs
}

func validateMeta(s *Schema) []ValidationError {
	var errs []ValidationError
	if s.Version == "" {
		errs = append(errs, ValidationError{Path: "version", Message: "required"})
	}
	if s.Domain == "" {
		errs = append(errs, ValidationError{Path: "domain", Message: "required"})
	}
	return errs
}

func validateTransport(s *Schema) []ValidationError {
	var errs []ValidationError

	validFrameworks := map[string]bool{"gin": true, "fiber": true, "echo": true}
	if s.Transport.Framework == "" {
		errs = append(errs, ValidationError{Path: "transport.framework", Message: "required"})
	} else if !validFrameworks[strings.ToLower(s.Transport.Framework)] {
		errs = append(errs, ValidationError{
			Path:    "transport.framework",
			Message: fmt.Sprintf("unsupported framework %q; must be one of: gin, fiber, echo", s.Transport.Framework),
		})
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true,
	}
	if s.Transport.Method == "" {
		errs = append(errs, ValidationError{Path: "transport.method", Message: "required"})
	} else if !validMethods[strings.ToUpper(s.Transport.Method)] {
		errs = append(errs, ValidationError{
			Path:    "transport.method",
			Message: fmt.Sprintf("unsupported method %q", s.Transport.Method),
		})
	}

	if s.Transport.URL == "" {
		errs = append(errs, ValidationError{Path: "transport.url", Message: "required"})
	}

	return errs
}

func validateFields(section string, fields []Field) []ValidationError {
	var errs []ValidationError

	validTypes := map[string]bool{
		"string": true, "int": true, "int32": true, "int64": true,
		"float": true, "float32": true, "float64": true,
		"bool": true, "uuid": true, "time": true, "time.Time": true,
		"byte": true, "[]byte": true,
	}
	validSources := map[string]bool{
		"body": true, "query": true, "header": true, "path": true,
	}

	for i, f := range fields {
		prefix := fmt.Sprintf("%s[%d]", section, i)

		if f.Name == "" {
			errs = append(errs, ValidationError{Path: prefix + ".name", Message: "required"})
		}
		if f.Type == "" {
			errs = append(errs, ValidationError{Path: prefix + ".type", Message: "required"})
		} else if !validTypes[f.Type] {
			errs = append(errs, ValidationError{
				Path:    prefix + ".type",
				Message: fmt.Sprintf("unsupported type %q", f.Type),
			})
		}

		// source обязателен только для входных полей
		if section == "input" && f.Source != "" && !validSources[f.Source] {
			errs = append(errs, ValidationError{
				Path:    prefix + ".source",
				Message: fmt.Sprintf("unsupported source %q; must be one of: body, query, header, path", f.Source),
			})
		}
	}

	return errs
}

func validateRepository(s *Schema) []ValidationError {
	var errs []ValidationError

	validStrategies := map[string]bool{"native": true, "sqlc": true}
	if s.Repository.Strategy == "" {
		errs = append(errs, ValidationError{Path: "repository.strategy", Message: "required"})
		return errs
	}
	if !validStrategies[s.Repository.Strategy] {
		errs = append(errs, ValidationError{
			Path:    "repository.strategy",
			Message: fmt.Sprintf("unsupported strategy %q; must be native or sqlc", s.Repository.Strategy),
		})
		return errs
	}

	if s.Repository.Strategy == "native" {
		errs = append(errs, validateNativeRepository(s)...)
	} else {
		errs = append(errs, validateSqlcRepository(s)...)
	}

	return errs
}

func validateNativeRepository(s *Schema) []ValidationError {
	var errs []ValidationError

	validDrivers := map[string]bool{"pgx": true, "sqlx": true}
	if s.Repository.Driver == "" {
		errs = append(errs, ValidationError{Path: "repository.driver", Message: "required for strategy native"})
	} else if !validDrivers[s.Repository.Driver] {
		errs = append(errs, ValidationError{
			Path:    "repository.driver",
			Message: fmt.Sprintf("unsupported driver %q; must be pgx or sqlx", s.Repository.Driver),
		})
	}

	if s.Repository.Table == "" {
		errs = append(errs, ValidationError{Path: "repository.table", Message: "required"})
	}
	if s.Repository.Schema == "" {
		errs = append(errs, ValidationError{Path: "repository.schema", Message: "required"})
	}

	validOps := map[string]bool{"insert": true, "select": true, "update": true, "delete": true}
	if s.Repository.Operation == "" {
		errs = append(errs, ValidationError{Path: "repository.operation", Message: "required"})
	} else if !validOps[s.Repository.Operation] {
		errs = append(errs, ValidationError{
			Path:    "repository.operation",
			Message: fmt.Sprintf("unsupported operation %q; must be insert, select, update or delete", s.Repository.Operation),
		})
	}

	if len(s.Repository.Fields) == 0 {
		errs = append(errs, ValidationError{Path: "repository.fields", Message: "at least one field required"})
	}

	return errs
}

func validateSqlcRepository(s *Schema) []ValidationError {
	var errs []ValidationError

	if s.Repository.Sqlc == nil {
		errs = append(errs, ValidationError{Path: "repository.sqlc", Message: "required for strategy sqlc"})
		return errs
	}

	validModes := map[string]bool{"generate": true, "existing": true}
	if s.Repository.Sqlc.Mode == "" {
		errs = append(errs, ValidationError{Path: "repository.sqlc.mode", Message: "required"})
	} else if !validModes[s.Repository.Sqlc.Mode] {
		errs = append(errs, ValidationError{
			Path:    "repository.sqlc.mode",
			Message: fmt.Sprintf("unsupported mode %q; must be generate or existing", s.Repository.Sqlc.Mode),
		})
	}

	if s.Repository.Sqlc.Mode == "existing" && s.Repository.Sqlc.Config == "" {
		errs = append(errs, ValidationError{
			Path:    "repository.sqlc.config",
			Message: "required when mode is existing",
		})
	}

	// Для режима generate нативные поля также необходимы для построения SQL
	if s.Repository.Sqlc.Mode == "generate" {
		if s.Repository.Table == "" {
			errs = append(errs, ValidationError{Path: "repository.table", Message: "required for sqlc generate mode"})
		}
		if s.Repository.Schema == "" {
			errs = append(errs, ValidationError{Path: "repository.schema", Message: "required for sqlc generate mode"})
		}
		validOps := map[string]bool{"insert": true, "select": true, "update": true, "delete": true}
		if s.Repository.Operation == "" {
			errs = append(errs, ValidationError{Path: "repository.operation", Message: "required for sqlc generate mode"})
		} else if !validOps[s.Repository.Operation] {
			errs = append(errs, ValidationError{
				Path:    "repository.operation",
				Message: fmt.Sprintf("unsupported operation %q", s.Repository.Operation),
			})
		}
		if len(s.Repository.Fields) == 0 {
			errs = append(errs, ValidationError{Path: "repository.fields", Message: "at least one field required for sqlc generate mode"})
		}
	}

	return errs
}

func validateGenerate(s *Schema) []ValidationError {
	var errs []ValidationError

	validLayers := map[string]bool{
		"domain": true, "repository": true, "service": true, "handler": true,
	}

	if len(s.Generate.Layers) == 0 {
		errs = append(errs, ValidationError{Path: "generate.layers", Message: "at least one layer required"})
	}
	for i, l := range s.Generate.Layers {
		if l != "all" && !validLayers[l] {
			errs = append(errs, ValidationError{
				Path:    fmt.Sprintf("generate.layers[%d]", i),
				Message: fmt.Sprintf("unknown layer %q; valid: domain, repository, service, handler", l),
			})
		}
	}

	if s.Generate.OutputDir == "" {
		errs = append(errs, ValidationError{Path: "generate.output_dir", Message: "required"})
	}

	return errs
}

// FormatErrors возвращает все ошибки валидации в виде одной читаемой строки.
func FormatErrors(errs []ValidationError) string {
	if len(errs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation error(s):\n", len(errs)))
	for _, e := range errs {
		sb.WriteString(fmt.Sprintf("  - %s\n", e.Error()))
	}
	return strings.TrimRight(sb.String(), "\n")
}
