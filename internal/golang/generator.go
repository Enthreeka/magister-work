// Package golang is the Go language plugin for codegen.
// It wires together the domain, repository, service, and handler layer generators
// and exposes a single NewEngine factory that returns a configured generator.Engine.
package golang

import (
	"strings"

	"github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/generator/ai"
	repostrategy "github.com/Enthreeka/magister-work/internal/generator/repository"
	"github.com/Enthreeka/magister-work/internal/golang/layers"
	"github.com/Enthreeka/magister-work/internal/schema"
)

// NewEngine creates a generator.Engine pre-loaded with all Go layer generators.
// strategy is "native" or "sqlc"; aiProvider defaults to noop if empty.
func NewEngine(repoStrategy repostrategy.Strategy, aiProvider ai.BusinessLogicProvider) *generator.Engine {
	e := generator.NewEngine()

	e.Register(layers.DomainLayer{})
	e.Register(layers.RepositoryLayer{Strategy: repoStrategy})
	e.Register(layers.ServiceLayer{AIProvider: aiProvider})
	e.Register(layers.HandlerLayer{})

	return e
}

// BuildTemplateData constructs a TemplateData from a parsed Schema.
// outputDir is the resolved ({{.Domain}}-expanded) output directory.
func BuildTemplateData(s *schema.Schema, sourceFile, version string) *generator.TemplateData {
	domain := strings.ToLower(s.Domain)
	domainTitle := snakeToCamel(domain) // user_create → UserCreate
	method := operationToMethod(s.Repository.Operation)

	return &generator.TemplateData{
		SourceFile:      sourceFile,
		Version:         version,
		Module:          s.Module,
		Domain:          domain,
		DomainTitle:     domainTitle,
		RequestType:     domainTitle + "Request",
		ResponseType:    domainTitle + "Response",
		ServiceType:     domainTitle + "Service",
		RepoType:        domainTitle + "Repository",
		HandlerType:     domainTitle + "Handler",
		OperationMethod: method,
		Input:           s.Input,
		Output:          s.Output,
		Transport:       s.Transport,
		Repository:      s, // layers receive the full schema
		Service:         s.Service,
	}
}

// snakeToCamel converts snake_case to CamelCase: user_create → UserCreate.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func operationToMethod(op string) string {
	m := map[string]string{
		"insert": "Create",
		"select": "Get",
		"update": "Update",
		"delete": "Delete",
	}
	if name, ok := m[strings.ToLower(op)]; ok {
		return name
	}
	return strings.Title(op)
}
