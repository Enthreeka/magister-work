package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/generator/ai"
	"github.com/Enthreeka/magister-work/internal/generator/compat"
	repostrategy "github.com/Enthreeka/magister-work/internal/generator/repository"
	gogen "github.com/Enthreeka/magister-work/internal/golang"
	"github.com/Enthreeka/magister-work/internal/schema"
)

type generateFlags struct {
	schemaPath    string
	outputDir     string
	layers        string
	dryRun        bool
	force         bool
	forceBreaking bool
}

func newGenerateCmd() *cobra.Command {
	f := &generateFlags{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Генерировать код из схемы system-gen.yaml",
		Example: `  codegen generate
  codegen generate --schema ./system-gen.yaml --layers domain,service
  codegen generate --dry-run`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGenerate(cmd.Context(), f)
		},
	}

	cmd.Flags().StringVarP(&f.schemaPath, "schema", "s", "system-gen.yaml", "путь к файлу требований YAML")
	cmd.Flags().StringVarP(&f.outputDir, "output", "o", "", "корневой каталог вывода (переопределяет output_dir схемы)")
	cmd.Flags().StringVarP(&f.layers, "layers", "l", "", "список слоёв через запятую для генерации: domain,repository,service,handler")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "вывести файлы, которые были бы записаны, без фактической записи")
	cmd.Flags().BoolVar(&f.force, "force", false, "перезаписать отредактированные пользователем (не сгенерированные) файлы")
	cmd.Flags().BoolVar(&f.forceBreaking, "force-breaking", false, "разрешить ломающие изменения схемы")

	return cmd
}

func runGenerate(ctx context.Context, f *generateFlags) error {
	// 1. Разбор схемы
	s, err := schema.ParseFile(f.schemaPath)
	if err != nil {
		return err
	}

	// 2. Валидация
	if errs := schema.Validate(s); len(errs) > 0 {
		return fmt.Errorf("schema validation failed:\n%s", schema.FormatErrors(errs))
	}

	// 3. Проверка совместимости
	changes, err := compat.Check(s)
	if err != nil {
		return fmt.Errorf("compat check: %w", err)
	}
	if len(changes) > 0 {
		printBreakingChanges(changes)
		if compat.HasErrors(changes) && !f.forceBreaking {
			return fmt.Errorf(
				"\nbreaking changes detected — use --force-breaking to proceed\n" +
					"(review the changes above before forcing)",
			)
		}
	}

	// 4. Определение выходного каталога
	outputDir := f.outputDir
	if outputDir == "" {
		outputDir = generator.ExpandOutputDir(s.Generate.OutputDir, s.Domain)
	}

	// 5. Выбор стратегии репозитория
	repoStrat, err := selectStrategy(s)
	if err != nil {
		return err
	}

	// 6. Создание движка и данных шаблона
	// generate всегда использует TemplateProvider — заполнение AI является отдельным шагом (codegen ai fill)
	engine := gogen.NewEngine(repoStrat, ai.TemplateProvider{})
	data := gogen.BuildTemplateData(s, f.schemaPath, version)

	// 8. Определение слоёв
	var layerList []string
	if f.layers != "" {
		layerList = splitTrim(f.layers, ",")
	} else {
		layerList = expandLayers(s.Generate.Layers)
	}

	// 8. Запуск
	opts := generator.Options{
		SchemaPath:    f.schemaPath,
		OutputDir:     outputDir,
		Layers:        layerList,
		DryRun:        f.dryRun,
		Force:         f.force,
		ForceBreaking: f.forceBreaking,
		SourceFile:    f.schemaPath,
		Version:       version,
	}

	results, err := engine.Run(ctx, data, opts)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// 9. Отчёт
	printResults(results, f.dryRun)

	// 10. Сохранение lock-файла (только при реальном запуске)
	if !f.dryRun {
		if err := compat.SaveLock(s); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update %s: %v\n", compat.LockFile, err)
		}
	}

	return nil
}


func selectStrategy(s *schema.Schema) (repostrategy.Strategy, error) {
	switch strings.ToLower(s.Repository.Strategy) {
	case "native":
		return repostrategy.NativeStrategy{}, nil
	case "sqlc":
		return repostrategy.SqlcStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown repository strategy %q", s.Repository.Strategy)
	}
}

func expandLayers(layers []string) []string {
	for _, l := range layers {
		if l == "all" {
			return []string{"domain", "repository", "service", "handler"}
		}
	}
	return layers
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func printBreakingChanges(changes []compat.BreakingChange) {
	fmt.Println("⚠️  Schema changes detected:")
	for _, c := range changes {
		fmt.Printf("  %s\n", c.String())
	}
}

func printResults(results []generator.WriteResult, dryRun bool) {
	if dryRun {
		fmt.Println("Dry run — no files written:")
	} else {
		fmt.Println("Generation complete:")
	}
	for _, r := range results {
		icon := actionIcon(r.Action)
		fmt.Printf("  %s %s  (%s)\n", icon, r.Path, r.Action)
	}
}

func actionIcon(action string) string {
	switch action {
	case "created":
		return "+"
	case "updated":
		return "~"
	case "skipped":
		return "–"
	case "dry-run":
		return "?"
	default:
		return " "
	}
}
