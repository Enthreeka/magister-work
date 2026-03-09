package layers

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Enthreeka/magister-work/internal/generator"
	gen "github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/golang/tmplsrc"
	"github.com/Enthreeka/magister-work/internal/schema"
	"github.com/Enthreeka/magister-work/pkg/typemap"
)

// DomainLayer generates the domain contracts file (types + interfaces + errors).
type DomainLayer struct{}

func (DomainLayer) Layer() string { return "domain" }

func (DomainLayer) Generate(_ context.Context, data *gen.TemplateData) ([]generator.File, error) {
	s, ok := data.Repository.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("domain layer: expected *schema.Schema in data.Repository")
	}

	content, err := renderDomain(data, s)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(expandDir(data), "domain", "domain.gen.go")

	return []generator.File{
		{Path: path, Content: []byte(content), Protected: true},
	}, nil
}

type domainTmplData struct {
	Header          string
	Domain          string
	DomainTitle     string
	RequestType     string
	ResponseType    string
	OperationMethod string
	NeedsTime       bool
	Input           []schema.Field
	Output          []schema.Field
}

func renderDomain(data *gen.TemplateData, s *schema.Schema) (string, error) {
	allTypes := collectTypes(s)

	td := domainTmplData{
		Header:          gen.Header(data.SourceFile, data.Version),
		Domain:          data.Domain,
		DomainTitle:     data.DomainTitle,
		RequestType:     data.RequestType,
		ResponseType:    data.ResponseType,
		OperationMethod: data.OperationMethod,
		NeedsTime:       typemap.NeedsTimeImport(allTypes),
		Input:           s.Input,
		Output:          s.Output,
	}

	tmpl, err := template.New("domain").
		Funcs(templateFuncs()).
		Parse(tmplsrc.DomainTemplate)
	if err != nil {
		return "", fmt.Errorf("domain layer: parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("domain layer: execute template: %w", err)
	}
	return buf.String(), nil
}

func collectTypes(s *schema.Schema) []string {
	types := make([]string, 0, len(s.Input)+len(s.Output))
	for _, f := range s.Input {
		types = append(types, f.Type)
	}
	for _, f := range s.Output {
		types = append(types, f.Type)
	}
	return types
}

// expandDir returns the per-domain output directory.
// If data.OutputDir is set by the engine, it is used as-is;
// otherwise the domain name is used as a relative fallback.
func expandDir(data *gen.TemplateData) string {
	if data.OutputDir != "" {
		return data.OutputDir
	}
	return strings.ToLower(data.Domain)
}
