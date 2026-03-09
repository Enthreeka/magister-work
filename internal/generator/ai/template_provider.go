package ai

import (
	"context"
	"fmt"
	"strings"
)

// TemplateProvider generates real working business logic based on the schema.
// It covers the common case: validate required fields → call repo → wrap errors.
// This is the default provider and requires no external dependencies.
//
// For custom business rules (uniqueness checks, external calls, etc.) either:
//   - edit service.go manually after generation, or
//   - switch to an AI-backed provider via ai_provider in the schema.
type TemplateProvider struct{}

func (TemplateProvider) Name() string { return "template" }

func (TemplateProvider) GenerateMethodBody(_ context.Context, req MethodRequest) (string, error) {
	var sb strings.Builder

	// 1. Validation block for required fields
	validations := buildValidations(req)
	if len(validations) > 0 {
		for _, v := range validations {
			sb.WriteString(v)
		}
		sb.WriteString("\n")
	}

	// 2. Repository call + error handling
	repoCall := buildRepoCall(req)
	sb.WriteString(repoCall)

	return sb.String(), nil
}

// buildValidations generates nil/empty checks for required fields.
func buildValidations(req MethodRequest) []string {
	var lines []string

	for _, f := range req.RequiredFields {
		check := buildFieldCheck(f.Name, f.GoType, req.DomainErrValidation)
		if check != "" {
			lines = append(lines, check)
		}
	}

	return lines
}

func buildFieldCheck(fieldName, goType, domainErrValidation string) string {
	camel := toCamelCase(fieldName)

	switch goType {
	case "string", "uuid":
		return fmt.Sprintf(
			"\tif req.%s == \"\" {\n\t\treturn nil, fmt.Errorf(\"%%w: %s is required\", %s)\n\t}\n",
			camel, fieldName, domainErrValidation,
		)
	case "int64", "int32", "int", "float64", "float32":
		return fmt.Sprintf(
			"\tif req.%s == 0 {\n\t\treturn nil, fmt.Errorf(\"%%w: %s is required\", %s)\n\t}\n",
			camel, fieldName, domainErrValidation,
		)
	default:
		// Pointer types, slices, etc.
		return fmt.Sprintf(
			"\tif req.%s == nil {\n\t\treturn nil, fmt.Errorf(\"%%w: %s is required\", %s)\n\t}\n",
			camel, fieldName, domainErrValidation,
		)
	}
}

// buildRepoCall generates the repository invocation with error wrapping.
func buildRepoCall(req MethodRequest) string {
	var sb strings.Builder

	switch strings.ToLower(req.Operation) {
	case "insert", "select":
		sb.WriteString(fmt.Sprintf("\tresp, err := s.repo.%s(ctx, req)\n", req.MethodName))
		sb.WriteString("\tif err != nil {\n")
		if req.Operation == "select" {
			sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%%w: %%v\", %s, err)\n", req.DomainErrNotFound))
		} else {
			sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%%w: %%v\", %s, err)\n", req.DomainErrInternal))
		}
		sb.WriteString("\t}\n\n")
		sb.WriteString("\treturn resp, nil\n")

	case "update":
		sb.WriteString(fmt.Sprintf("\t_, err := s.repo.%s(ctx, req)\n", req.MethodName))
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%%w: %%v\", %s, err)\n", req.DomainErrInternal))
		sb.WriteString("\t}\n\n")
		sb.WriteString("\treturn nil, nil\n")

	case "delete":
		sb.WriteString(fmt.Sprintf("\tif err := s.repo.%s(ctx, req); err != nil {\n", req.MethodName))
		sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%%w: %%v\", %s, err)\n", req.DomainErrInternal))
		sb.WriteString("\t}\n\n")
		sb.WriteString("\treturn nil, nil\n")

	default:
		// Fallback — should not happen with validated schema
		sb.WriteString(fmt.Sprintf("\treturn s.repo.%s(ctx, req)\n", req.MethodName))
	}

	return sb.String()
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
