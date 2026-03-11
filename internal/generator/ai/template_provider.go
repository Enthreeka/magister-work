package ai

import (
	"context"
	"fmt"
	"strings"
)

// TemplateProvider генерирует реально работающую бизнес-логику на основе схемы.
// Покрывает типовой случай: валидация обязательных полей → вызов репозитория → оборачивание ошибок.
// Является провайдером по умолчанию и не требует внешних зависимостей.
//
// Для пользовательских бизнес-правил (проверки уникальности, внешние вызовы и т.д.):
//   - отредактируйте service.go вручную после генерации, или
//   - переключитесь на провайдера на базе ИИ через ai_provider в схеме.
type TemplateProvider struct{}

func (TemplateProvider) Name() string { return "template" }

func (TemplateProvider) GenerateMethodBody(_ context.Context, req MethodRequest) (string, error) {
	var sb strings.Builder

	// 1. Блок валидации обязательных полей
	validations := buildValidations(req)
	if len(validations) > 0 {
		for _, v := range validations {
			sb.WriteString(v)
		}
		sb.WriteString("\n")
	}

	// 2. Вызов репозитория + обработка ошибок
	repoCall := buildRepoCall(req)
	sb.WriteString(repoCall)

	return sb.String(), nil
}

// buildValidations генерирует проверки nil/пустоты для обязательных полей.
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
		// Указатели, срезы и т.д.
		return fmt.Sprintf(
			"\tif req.%s == nil {\n\t\treturn nil, fmt.Errorf(\"%%w: %s is required\", %s)\n\t}\n",
			camel, fieldName, domainErrValidation,
		)
	}
}

// buildRepoCall генерирует вызов репозитория с оборачиванием ошибок.
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
		// Запасной вариант — не должен произойти при валидированной схеме
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
