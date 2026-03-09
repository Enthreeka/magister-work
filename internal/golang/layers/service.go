package layers

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/Enthreeka/magister-work/internal/generator"
	gen "github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/generator/ai"
	"github.com/Enthreeka/magister-work/internal/golang/tmplsrc"
	"github.com/Enthreeka/magister-work/internal/schema"
)

// ServiceLayer generates two files:
//   - service.gen.go  (Protected=true)  — struct, constructor, compile-time interface check
//   - service.go      (Protected=false) — method stubs; created once, never overwritten
type ServiceLayer struct {
	AIProvider ai.BusinessLogicProvider
}

func (ServiceLayer) Layer() string { return "service" }

func (l ServiceLayer) Generate(ctx context.Context, data *gen.TemplateData) ([]generator.File, error) {
	s, ok := data.Repository.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("service layer: expected *schema.Schema in data.Repository")
	}

	td := buildServiceTmplData(data)

	// 1. Generated scaffold (DO NOT EDIT)
	genContent, err := renderTemplate("service", tmplsrc.ServiceTemplate, td)
	if err != nil {
		return nil, fmt.Errorf("service layer: scaffold: %w", err)
	}

	// 2. User stub (created once, never overwritten)
	provider := l.AIProvider
	if provider == nil {
		provider = ai.TemplateProvider{}
	}
	methodBody, err := provider.GenerateMethodBody(ctx, BuildMethodRequest(data, s))
	if err != nil {
		return nil, fmt.Errorf("service layer: method body: %w", err)
	}
	td.MethodBody = methodBody

	stubContent, err := renderTemplate("service_stub", tmplsrc.ServiceStubTemplate, td)
	if err != nil {
		return nil, fmt.Errorf("service layer: stub: %w", err)
	}

	dir := filepath.Join(expandDir(data), "service")

	return []generator.File{
		{
			Path:      filepath.Join(dir, "service.gen.go"),
			Content:   []byte(genContent),
			Protected: true, // always overwritten on regeneration
		},
		{
			Path:      filepath.Join(dir, "service.go"),
			Content:   []byte(stubContent),
			Protected: false, // written only if file does not exist yet
		},
	}, nil
}

type serviceTmplData struct {
	Header          string
	Module          string
	Domain          string
	DomainTitle     string
	DomainImport    string
	RequestType     string
	ResponseType    string
	RepoType        string
	ServiceType     string
	OperationMethod string
	MethodBody      string // used only by stub template
}

func buildServiceTmplData(data *gen.TemplateData) serviceTmplData {
	return serviceTmplData{
		Header:          gen.Header(data.SourceFile, data.Version),
		Module:          data.Module,
		Domain:          data.Domain,
		DomainTitle:     data.DomainTitle,
		DomainImport:    data.Module + "/" + expandDir(data) + "/domain",
		RequestType:     data.RequestType,
		ResponseType:    data.ResponseType,
		RepoType:        data.RepoType,
		ServiceType:     data.ServiceType,
		OperationMethod: data.OperationMethod,
	}
}

func renderTemplate(name, src string, data any) (string, error) {
	tmpl, err := template.New(name).
		Funcs(templateFuncs()).
		Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
	}
	return buf.String(), nil
}

func serviceDescription(s *schema.Schema) string {
	if s.Service != nil && s.Service.Description != "" {
		return s.Service.Description
	}
	return ""
}

// BuildMethodRequest constructs an ai.MethodRequest from template data and schema.
// Exported so the CLI ai fill command can reuse it without duplicating logic.
func BuildMethodRequest(data *gen.TemplateData, s *schema.Schema) ai.MethodRequest {
	domainTitle := data.DomainTitle

	// Collect required fields with their Go types
	var required []ai.FieldInfo
	for _, f := range s.Input {
		if f.Required {
			required = append(required, ai.FieldInfo{
				Name:   f.Name,
				GoType: toGoType(f.Type),
			})
		}
	}

	return ai.MethodRequest{
		MethodName:          data.OperationMethod,
		InputType:           "domain." + data.RequestType,
		OutputType:          "domain." + data.ResponseType,
		Operation:           s.Repository.Operation,
		Description:         serviceDescription(s),
		RequiredFields:      required,
		DomainErrDomain:     "domain.Err" + domainTitle + "Domain",
		DomainErrValidation: "domain.Err" + domainTitle + "Validation",
		DomainErrInternal:   "domain.Err" + domainTitle + "Internal",
		DomainErrNotFound:   "domain.Err" + domainTitle + "NotFound",
	}
}
