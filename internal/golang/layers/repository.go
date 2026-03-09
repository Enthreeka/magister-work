package layers

import (
	"context"
	"fmt"

	"github.com/Enthreeka/magister-work/internal/generator"
	gen "github.com/Enthreeka/magister-work/internal/generator"
	repostrategy "github.com/Enthreeka/magister-work/internal/generator/repository"
	"github.com/Enthreeka/magister-work/internal/schema"
)

// RepositoryLayer generates the repository layer using a pluggable strategy.
type RepositoryLayer struct {
	Strategy repostrategy.Strategy
}

func (RepositoryLayer) Layer() string { return "repository" }

func (l RepositoryLayer) Generate(ctx context.Context, data *gen.TemplateData) ([]generator.File, error) {
	s, ok := data.Repository.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("repository layer: expected *schema.Schema in data.Repository")
	}

	opts := repostrategy.Options{
		OutputDir:    expandDir(data),
		SourceFile:   data.SourceFile,
		Version:      data.Version,
		DomainImport: data.Module + "/" + expandDir(data) + "/domain",
	}

	if err := l.Strategy.Prepare(ctx, s, opts); err != nil {
		return nil, fmt.Errorf("repository layer: prepare: %w", err)
	}

	files, err := l.Strategy.Files(s, opts)
	if err != nil {
		return nil, fmt.Errorf("repository layer: files: %w", err)
	}
	return files, nil
}
