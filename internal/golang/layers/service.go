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

// ServiceLayer generates the service (business-logic) layer.
type ServiceLayer struct {
	// AIProvider is used to generate method bodies.
	// Defaults to NoopProvider if nil.
	AIProvider ai.BusinessLogicProvider
}

func (ServiceLayer) Layer() string { return "service" }

func (l ServiceLayer) Generate(ctx context.Context, data *gen.TemplateData) ([]generator.File, error) {
	s, ok := data.Repository.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("service layer: expected *schema.Schema in data.Repository")
	}

	provider := l.AIProvider
	if provider == nil {
		provider = ai.NoopProvider{}
	}

	body, err := provider.GenerateMethodBody(ctx, ai.MethodRequest{
		MethodName:  data.OperationMethod,
		InputType:   "domain." + data.RequestType,
		OutputType:  "domain." + data.ResponseType,
		Description: serviceDescription(s),
	})
	if err != nil {
		return nil, fmt.Errorf("service layer: generate method body: %w", err)
	}

	content, err := renderService(data, body)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(expandDir(data), "gen", "service.gen.go")

	return []generator.File{
		{Path: path, Content: []byte(content), Protected: true},
	}, nil
}

type serviceTmplData struct {
	Header          string
	Module          string
	Domain          string
	DomainTitle     string
	RequestType     string
	ResponseType    string
	RepoType        string
	ServiceType     string
	OperationMethod string
	MethodBody      string
}

func renderService(data *gen.TemplateData, methodBody string) (string, error) {
	td := serviceTmplData{
		Header:          gen.Header(data.SourceFile, data.Version),
		Module:          data.Module,
		Domain:          data.Domain,
		DomainTitle:     data.DomainTitle,
		RequestType:     data.RequestType,
		ResponseType:    data.ResponseType,
		RepoType:        data.RepoType,
		ServiceType:     data.ServiceType,
		OperationMethod: data.OperationMethod,
		MethodBody:      methodBody,
	}

	tmpl, err := template.New("service").
		Funcs(templateFuncs()).
		Parse(tmplsrc.ServiceTemplate)
	if err != nil {
		return "", fmt.Errorf("service layer: parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("service layer: execute template: %w", err)
	}
	return buf.String(), nil
}

func serviceDescription(s *schema.Schema) string {
	if s.Service != nil && s.Service.Description != "" {
		return s.Service.Description
	}
	return ""
}
