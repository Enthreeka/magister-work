package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/generator/ai"
	gogen "github.com/Enthreeka/magister-work/internal/golang"
	"github.com/Enthreeka/magister-work/internal/golang/layers"
	"github.com/Enthreeka/magister-work/internal/golang/tmplsrc"
	"github.com/Enthreeka/magister-work/internal/schema"
)

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-assisted code generation",
	}
	cmd.AddCommand(newAIFillCmd())
	return cmd
}

type aiFillFlags struct {
	schemaPath string
	model      string
	provider   string
	outputDir  string
	dryRun     bool
}

func newAIFillCmd() *cobra.Command {
	f := &aiFillFlags{}

	cmd := &cobra.Command{
		Use:   "fill",
		Short: "Generate business logic for service.go using an AI provider",
		Long: `Reads the schema and calls an AI provider to generate the body of
the service method based on the service.description field.

The result overwrites service.go — review the output before committing.

Requires ANTHROPIC_API_KEY env variable when using the anthropic provider.`,
		Example: `  codegen ai fill
  codegen ai fill --schema ./user/system-gen.yaml
  codegen ai fill --provider anthropic --model claude-opus-4-6
  codegen ai fill --dry-run`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAIFill(cmd.Context(), f)
		},
	}

	cmd.Flags().StringVarP(&f.schemaPath, "schema", "s", "system-gen.yaml", "path to the requirements YAML file")
	cmd.Flags().StringVarP(&f.outputDir, "output", "o", "", "root output directory (overrides schema output_dir)")
	cmd.Flags().StringVar(&f.provider, "provider", "anthropic", "AI provider: anthropic | openai | ollama")
	cmd.Flags().StringVar(&f.model, "model", "", "model override (e.g. claude-opus-4-6)")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "print generated code without writing to disk")

	return cmd
}

func runAIFill(ctx context.Context, f *aiFillFlags) error {
	// 1. Parse and validate schema
	s, err := schema.ParseFile(f.schemaPath)
	if err != nil {
		return err
	}
	if errs := schema.Validate(s); len(errs) > 0 {
		return fmt.Errorf("schema validation failed:\n%s", schema.FormatErrors(errs))
	}

	// 2. Resolve provider — flag overrides schema
	providerName := f.provider
	if s.AIProvider != nil && s.AIProvider.Name != "" && providerName == "anthropic" {
		providerName = s.AIProvider.Name
	}

	provider, err := resolveProvider(providerName, f.model, s)
	if err != nil {
		return err
	}

	// 3. Build template data
	data := gogen.BuildTemplateData(s, f.schemaPath, version)

	// 4. Resolve output dir
	outputDir := f.outputDir
	if outputDir == "" {
		outputDir = generator.ExpandOutputDir(s.Generate.OutputDir, s.Domain)
	}
	data.OutputDir = outputDir

	// 5. Build method request with full context
	methodReq := layers.BuildMethodRequest(data, s)

	fmt.Printf("Calling %s to generate %s.%s...\n",
		providerName, data.ServiceType, data.OperationMethod)

	// 6. Call AI
	methodBody, err := provider.GenerateMethodBody(ctx, methodReq)
	if err != nil {
		return fmt.Errorf("ai fill: %w", err)
	}

	// 7. Render service.go with AI-generated body
	content, err := renderServiceStub(data, methodBody)
	if err != nil {
		return err
	}

	// 8. Output
	if f.dryRun {
		fmt.Println("\n--- Generated service.go ---")
		fmt.Println(content)
		return nil
	}

	servicePath := filepath.Join(outputDir, "service", "service.go")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		return fmt.Errorf("ai fill: mkdir: %w", err)
	}
	if err := os.WriteFile(servicePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("ai fill: write %s: %w", servicePath, err)
	}

	fmt.Printf("~ %s  (ai-filled)\n", servicePath)
	fmt.Println("\nReview the generated code before committing.")
	return nil
}

func resolveProvider(name, modelOverride string, s *schema.Schema) (ai.BusinessLogicProvider, error) {
	switch name {
	case "anthropic":
		model := modelOverride
		apiKeyEnv := ""
		if s.AIProvider != nil {
			if model == "" {
				model = s.AIProvider.Model
			}
			apiKeyEnv = s.AIProvider.ApiKeyEnv
		}
		return ai.AnthropicProvider{Model: model, ApiKeyEnv: apiKeyEnv}, nil
	default:
		return ai.Get(name)
	}
}

func renderServiceStub(data *generator.TemplateData, methodBody string) (string, error) {
	td := struct {
		Header          string
		DomainImport    string
		ServiceType     string
		DomainTitle     string
		OperationMethod string
		RequestType     string
		ResponseType    string
		MethodBody      string
	}{
		// service.go is a user file — no DO NOT EDIT header
		DomainImport:    data.Module + "/" + strings.TrimRight(data.OutputDir, "/") + "/domain",
		ServiceType:     data.ServiceType,
		DomainTitle:     data.DomainTitle,
		OperationMethod: data.OperationMethod,
		RequestType:     data.RequestType,
		ResponseType:    data.ResponseType,
		MethodBody:      methodBody,
	}

	tmpl, err := template.New("service_stub").Parse(tmplsrc.ServiceStubTemplate)
	if err != nil {
		return "", fmt.Errorf("render service stub: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("render service stub: %w", err)
	}
	return buf.String(), nil
}
