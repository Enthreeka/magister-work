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
		Short: "Генерация кода с помощью ИИ",
	}
	cmd.AddCommand(newAIFillCmd())
	cmd.AddCommand(newAIListCmd())
	return cmd
}

func newAIListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Показать доступных провайдеров ИИ и их модели по умолчанию",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("Available AI providers:\n")
			fmt.Printf("  %-12s  %-30s  %s\n", "PROVIDER", "DEFAULT MODEL", "ENV VAR")
			fmt.Printf("  %-12s  %-30s  %s\n", "--------", "-------------", "-------")
			fmt.Printf("  %-12s  %-30s  %s\n", "anthropic", "claude-opus-4-6", "ANTHROPIC_API_KEY")
			fmt.Printf("  %-12s  %-30s  %s\n", "openai", "gpt-4o", "OPENAI_API_KEY")
			fmt.Printf("  %-12s  %-30s  %s\n", "openrouter", "openai/gpt-4o", "OPENROUTER_API_KEY")
			fmt.Printf("  %-12s  %-30s  %s\n", "template", "(no API)", "-")
			fmt.Printf("  %-12s  %-30s  %s\n", "noop", "(no API)", "-")
			fmt.Println("\nUsage:")
			fmt.Println("  codegen ai fill --provider anthropic  --schema system-gen.yaml")
			fmt.Println("  codegen ai fill --provider openai     --schema system-gen.yaml")
			fmt.Println("  codegen ai fill --provider openrouter --schema system-gen.yaml")
			fmt.Println("\nOpenRouter example (system-gen.yaml):")
			fmt.Println("  ai_provider:")
			fmt.Println("    name: openrouter")
			fmt.Println("    model: anthropic/claude-opus-4              # any model from openrouter.ai/models")
			fmt.Println("    api_key_env: OPENROUTER_API_KEY")
			fmt.Println("\nOr set any OpenAI-compatible provider:")
			fmt.Println("  ai_provider:")
			fmt.Println("    name: openai")
			fmt.Println("    model: gpt-4o")
			fmt.Println("    api_key_env: MY_OPENAI_KEY")
		},
	}
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
		Short: "Генерировать бизнес-логику для service.go с использованием провайдера ИИ",
		Long: `Читает схему и обращается к провайдеру ИИ для генерации тела
метода сервиса на основе поля service.description.

Результат перезаписывает service.go — проверьте вывод перед коммитом.

Требует переменную окружения ANTHROPIC_API_KEY при использовании провайдера anthropic.`,
		Example: `  codegen ai fill
  codegen ai fill --schema ./user/system-gen.yaml
  codegen ai fill --provider anthropic --model claude-opus-4-6
  codegen ai fill --dry-run`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAIFill(cmd.Context(), f)
		},
	}

	cmd.Flags().StringVarP(&f.schemaPath, "schema", "s", "system-gen.yaml", "путь к файлу требований YAML")
	cmd.Flags().StringVarP(&f.outputDir, "output", "o", "", "корневой каталог вывода (переопределяет output_dir схемы)")
	cmd.Flags().StringVar(&f.provider, "provider", "anthropic", "провайдер ИИ: anthropic | openai | template | noop")
	cmd.Flags().StringVar(&f.model, "model", "", "переопределение модели (например, claude-opus-4-6)")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "вывести сгенерированный код без записи на диск")

	return cmd
}

func runAIFill(ctx context.Context, f *aiFillFlags) error {
	// 1. Разбор и валидация схемы
	s, err := schema.ParseFile(f.schemaPath)
	if err != nil {
		return err
	}
	if errs := schema.Validate(s); len(errs) > 0 {
		return fmt.Errorf("schema validation failed:\n%s", schema.FormatErrors(errs))
	}

	// 2. Определение провайдера — флаг переопределяет схему
	providerName := f.provider
	if s.AIProvider != nil && s.AIProvider.Name != "" && providerName == "anthropic" {
		providerName = s.AIProvider.Name
	}

	provider, err := resolveProvider(providerName, f.model, s)
	if err != nil {
		return err
	}

	// 3. Создание данных шаблона
	data := gogen.BuildTemplateData(s, f.schemaPath, version)

	// 4. Определение выходного каталога
	outputDir := f.outputDir
	if outputDir == "" {
		outputDir = generator.ExpandOutputDir(s.Generate.OutputDir, s.Domain)
	}
	data.OutputDir = outputDir

	// 5. Создание запроса метода с полным контекстом
	methodReq := layers.BuildMethodRequest(data, s)

	fmt.Printf("Calling %s to generate %s.%s...\n",
		providerName, data.ServiceType, data.OperationMethod)

	// 6. Вызов ИИ
	methodBody, err := provider.GenerateMethodBody(ctx, methodReq)
	if err != nil {
		return fmt.Errorf("ai fill: %w", err)
	}

	// 7. Рендеринг service.go с телом, сгенерированным ИИ
	content, err := renderServiceStub(data, methodBody)
	if err != nil {
		return err
	}

	// 8. Вывод
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
	// Объединение конфигурации уровня схемы с переопределениями флагов (флаги имеют приоритет)
	var schemaModel, schemaKeyEnv, schemaBaseURL string
	if s.AIProvider != nil {
		schemaModel = s.AIProvider.Model
		schemaKeyEnv = s.AIProvider.ApiKeyEnv
		schemaBaseURL = s.AIProvider.BaseURL
	}

	model := modelOverride
	if model == "" {
		model = schemaModel
	}

	switch name {
	case "anthropic":
		return ai.AnthropicProvider{Model: model, ApiKeyEnv: schemaKeyEnv}, nil
	case "openai":
		return ai.OpenAIProvider{Model: model, ApiKeyEnv: schemaKeyEnv, BaseURL: schemaBaseURL}, nil
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
		// service.go — пользовательский файл, заголовок DO NOT EDIT не добавляется
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
