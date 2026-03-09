// Package generator is the orchestration core of codegen.
// The Engine coordinates schema validation, compat checking, layer generation,
// and safe file writing.
package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LayerGenerator produces files for a single architectural layer.
// Each language plugin (Go, Python, Rust) implements this interface.
type LayerGenerator interface {
	// Layer returns the layer name this generator handles.
	Layer() string
	// Generate returns the files that should be written for the given schema data.
	Generate(ctx context.Context, data *TemplateData) ([]File, error)
}

// TemplateData is the unified context passed to every LayerGenerator.
// It is derived from the parsed Schema plus computed helper fields.
type TemplateData struct {
	// Schema metadata
	SourceFile string
	Version    string
	Module     string
	OutputDir  string // resolved output directory (set by Engine.Run)

	// Domain-derived names (pre-computed for templates)
	Domain          string // "user"
	DomainTitle     string // "User"
	RequestType     string // "UserRequest"
	ResponseType    string // "UserResponse"
	ServiceType     string // "UserService"
	RepoType        string // "UserRepository"
	HandlerType     string // "UserHandler"
	OperationMethod string // "Create"

	// Raw schema sections
	Input      interface{} // []schema.Field — kept as interface{} to avoid import cycle
	Output     interface{}
	Transport  interface{}
	Repository interface{}
	Service    interface{}
}

// Options controls a single generation run.
type Options struct {
	SchemaPath     string
	OutputDir      string
	Layers         []string // nil = all
	DryRun         bool
	Force          bool // overwrite non-generated user files
	ForceBreaking  bool
	SourceFile     string
	Version        string
}

// WriteResult records what happened to a single file during a run.
type WriteResult struct {
	Path    string
	Action  string // created | updated | skipped | dry-run
}

// Engine orchestrates code generation.
type Engine struct {
	generators map[string]LayerGenerator
}

// NewEngine creates an Engine with no registered generators.
func NewEngine() *Engine {
	return &Engine{generators: make(map[string]LayerGenerator)}
}

// Register adds a LayerGenerator. Panics on duplicate layer name.
func (e *Engine) Register(g LayerGenerator) {
	if _, exists := e.generators[g.Layer()]; exists {
		panic(fmt.Sprintf("generator: layer %q already registered", g.Layer()))
	}
	e.generators[g.Layer()] = g
}

// Run executes the full generation pipeline and returns per-file results.
func (e *Engine) Run(ctx context.Context, data *TemplateData, opts Options) ([]WriteResult, error) {
	layers := opts.Layers
	if len(layers) == 0 {
		layers = []string{"domain", "repository", "service", "handler"}
	}

	// Inject resolved output dir into template data so layer generators can
	// compute correct file paths without knowing the engine's options.
	data.OutputDir = opts.OutputDir

	var results []WriteResult
	for _, layer := range layers {
		gen, ok := e.generators[layer]
		if !ok {
			return nil, fmt.Errorf("engine: no generator registered for layer %q", layer)
		}

		files, err := gen.Generate(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("engine: layer %s: %w", layer, err)
		}

		for _, f := range files {
			r, err := e.writeFile(f, opts)
			if err != nil {
				return nil, fmt.Errorf("engine: write %s: %w", f.Path, err)
			}
			results = append(results, r)
		}
	}
	return results, nil
}

func (e *Engine) writeFile(f File, opts Options) (WriteResult, error) {
	if opts.DryRun {
		return WriteResult{Path: f.Path, Action: "dry-run"}, nil
	}

	// Check if file exists and whether we own it
	if _, err := os.Stat(f.Path); err == nil {
		isGen, err := IsGenerated(f.Path)
		if err != nil {
			return WriteResult{}, err
		}

		if !isGen && !f.Protected && !opts.Force {
			// Non-generated stub that the user may have edited — skip it
			return WriteResult{Path: f.Path, Action: "skipped"}, nil
		}
		if !isGen && !opts.Force {
			return WriteResult{Path: f.Path, Action: "skipped"}, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("mkdir %s: %w", filepath.Dir(f.Path), err)
	}

	existed := fileExists(f.Path)
	if err := os.WriteFile(f.Path, f.Content, 0o644); err != nil {
		return WriteResult{}, err
	}

	action := "created"
	if existed {
		action = "updated"
	}
	return WriteResult{Path: f.Path, Action: action}, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ExpandOutputDir resolves {{.Domain}} in the output_dir template string.
func ExpandOutputDir(tmpl, domain string) string {
	return strings.ReplaceAll(tmpl, "{{.Domain}}", strings.ToLower(domain))
}
