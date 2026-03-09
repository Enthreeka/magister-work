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
)

// HandlerLayer generates the transport (HTTP handler) layer for Gin.
type HandlerLayer struct{}

func (HandlerLayer) Layer() string { return "handler" }

func (HandlerLayer) Generate(_ context.Context, data *gen.TemplateData) ([]generator.File, error) {
	s, ok := data.Repository.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("handler layer: expected *schema.Schema in data.Repository")
	}

	content, err := renderHandler(data, s)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(expandDir(data), "gen", "handler.gen.go")

	return []generator.File{
		{Path: path, Content: []byte(content), Protected: true},
	}, nil
}

type handlerTmplData struct {
	Header          string
	Module          string
	Domain          string
	DomainTitle     string
	RequestType     string
	ResponseType    string
	ServiceType     string
	HandlerType     string
	OperationMethod string
	Transport       schema.Transport
	BindingBlock    string
}

func renderHandler(data *gen.TemplateData, s *schema.Schema) (string, error) {
	binding, err := buildBindingBlock(s)
	if err != nil {
		return "", err
	}

	td := handlerTmplData{
		Header:          gen.Header(data.SourceFile, data.Version),
		Module:          data.Module,
		Domain:          data.Domain,
		DomainTitle:     data.DomainTitle,
		RequestType:     data.RequestType,
		ResponseType:    data.ResponseType,
		ServiceType:     data.ServiceType,
		HandlerType:     data.HandlerType,
		OperationMethod: data.OperationMethod,
		Transport:       s.Transport,
		BindingBlock:    binding,
	}

	tmpl, err := template.New("handler_gin").
		Funcs(templateFuncs()).
		Parse(tmplsrc.HandlerGinTemplate)
	if err != nil {
		return "", fmt.Errorf("handler layer: parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("handler layer: execute template: %w", err)
	}
	return buf.String(), nil
}

// buildBindingBlock generates the request-binding code for Gin based on
// the input field sources (body, query, header, path).
func buildBindingBlock(s *schema.Schema) (string, error) {
	// Categorise fields by source
	var bodyFields, queryFields, headerFields, pathFields []schema.Field
	for _, f := range s.Input {
		switch strings.ToLower(f.Source) {
		case "body", "":
			bodyFields = append(bodyFields, f)
		case "query":
			queryFields = append(queryFields, f)
		case "header":
			headerFields = append(headerFields, f)
		case "path":
			pathFields = append(pathFields, f)
		}
	}

	var sb strings.Builder
	reqType := strings.Title(strings.ToLower(s.Domain)) + "Request"

	sb.WriteString(fmt.Sprintf("\tvar req domain.%s\n", reqType))

	// Body binding
	if len(bodyFields) > 0 {
		sb.WriteString("\tif err := c.ShouldBindJSON(&req); err != nil {\n")
		sb.WriteString("\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": err.Error()})\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n")
	}

	// Query binding
	if len(queryFields) > 0 {
		sb.WriteString("\tif err := c.ShouldBindQuery(&req); err != nil {\n")
		sb.WriteString("\t\tc.JSON(http.StatusBadRequest, gin.H{\"error\": err.Error()})\n")
		sb.WriteString("\t\treturn\n")
		sb.WriteString("\t}\n")
	}

	// Header fields (manual extraction)
	for _, f := range headerFields {
		fieldName := toCamel(f.Name)
		sb.WriteString(fmt.Sprintf("\treq.%s = c.GetHeader(%q)\n", fieldName, f.Name))
	}

	// Path params (manual extraction)
	for _, f := range pathFields {
		fieldName := toCamel(f.Name)
		sb.WriteString(fmt.Sprintf("\treq.%s = c.Param(%q)\n", fieldName, f.Name))
	}

	return sb.String(), nil
}
