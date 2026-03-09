package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Enthreeka/magister-work/internal/generator"
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
		Short: "Generate code from a system-gen.yaml schema",
		Example: `  codegen generate
  codegen generate --schema ./system-gen.yaml --layers domain,service
  codegen generate --dry-run`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGenerate(cmd.Context(), f)
		},
	}

	cmd.Flags().StringVarP(&f.schemaPath, "schema", "s", "system-gen.yaml", "path to the requirements YAML file")
	cmd.Flags().StringVarP(&f.outputDir, "output", "o", "", "root output directory (overrides schema output_dir)")
	cmd.Flags().StringVarP(&f.layers, "layers", "l", "", "comma-separated layers to generate: domain,repository,service,handler")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "print files that would be written without writing them")
	cmd.Flags().BoolVar(&f.force, "force", false, "overwrite user-edited (non-generated) files")
	cmd.Flags().BoolVar(&f.forceBreaking, "force-breaking", false, "allow breaking schema changes")

	return cmd
}

func runGenerate(ctx context.Context, f *generateFlags) error {
	// 1. Parse schema
	s, err := schema.ParseFile(f.schemaPath)
	if err != nil {
		return err
	}

	// 2. Validate
	if errs := schema.Validate(s); len(errs) > 0 {
		return fmt.Errorf("schema validation failed:\n%s", schema.FormatErrors(errs))
	}

	// 3. Compat check
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

	// 4. Resolve output directory
	outputDir := f.outputDir
	if outputDir == "" {
		outputDir = generator.ExpandOutputDir(s.Generate.OutputDir, s.Domain)
	}

	// 5. Select repository strategy
	repoStrat, err := selectStrategy(s)
	if err != nil {
		return err
	}

	// 6. Build engine and template data
	engine := gogen.NewEngine(repoStrat, nil) // nil → NoopProvider
	data := gogen.BuildTemplateData(s, f.schemaPath, version)

	// 7. Resolve layers
	var layerList []string
	if f.layers != "" {
		layerList = splitTrim(f.layers, ",")
	} else {
		layerList = expandLayers(s.Generate.Layers)
	}

	// 8. Run
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

	// 9. Report
	printResults(results, f.dryRun)

	// 10. Save lock (only on real run)
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
