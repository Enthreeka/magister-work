// Package repository defines the pluggable strategy interface for the
// repository layer generation. Each strategy decides how repository code
// is produced: directly by codegen (native) or via an external tool (sqlc).
package repository

import (
	"context"

	"github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/schema"
)

// Options carries generation-time settings forwarded from the engine.
type Options struct {
	OutputDir  string
	SourceFile string
	Version    string
	DryRun     bool
}

// RepositoryContract describes the interface that the service and handler
// layers will depend on. It is derived from the schema regardless of strategy.
type RepositoryContract struct {
	InterfaceName string
	MethodName    string
	InputType     string
	OutputType    string
}

// Strategy is the extension point for repository-layer generation.
// Implementations must be safe for concurrent use.
type Strategy interface {
	// Name returns the unique identifier of this strategy ("native", "sqlc").
	Name() string

	// Prepare performs any pre-generation steps (e.g. running sqlc generate).
	// It is called once before Files.
	Prepare(ctx context.Context, s *schema.Schema, opts Options) error

	// Contract derives the repository interface contract from the schema.
	// The contract is used by service and handler generators.
	Contract(s *schema.Schema) (*RepositoryContract, error)

	// Files returns the set of files this strategy wants to write.
	// A nil slice means the strategy delegates file writing to an external
	// tool (e.g. sqlc) and codegen should not write repository files itself.
	Files(s *schema.Schema, opts Options) ([]generator.File, error)
}
