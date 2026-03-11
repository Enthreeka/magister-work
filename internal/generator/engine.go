// Package generator — оркестрационное ядро codegen.
// Engine координирует валидацию схемы, проверку совместимости, генерацию слоёв
// и безопасную запись файлов.
package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LayerGenerator создаёт файлы для одного архитектурного слоя.
// Этот интерфейс будет реализовать каждый языковой плагин (Go, Python, Rust).
type LayerGenerator interface {
	// Layer возвращает имя слоя, который обрабатывает данный генератор.
	Layer() string
	// Generate возвращает файлы, которые должны быть записаны для указанных данных схемы.
	Generate(ctx context.Context, data *TemplateData) ([]File, error)
}

// TemplateData — унифицированный контекст, передаваемый каждому LayerGenerator.
// Формируется из разобранной схемы и вычисленных вспомогательных полей.
type TemplateData struct {
	// Метаданные схемы
	SourceFile string
	Version    string
	Module     string
	OutputDir  string // определённый выходной каталог (устанавливается в Engine.Run)

	// Имена, производные от домена (предвычисленные для шаблонов)
	Domain          string // "user"
	DomainTitle     string // "User"
	RequestType     string // "UserRequest"
	ResponseType    string // "UserResponse"
	ServiceType     string // "UserService"
	RepoType        string // "UserRepository"
	HandlerType     string // "UserHandler"
	OperationMethod string // "Create"

	// Секции сырой схемы
	Input      interface{} // []schema.Field — хранится как interface{} во избежание циклического импорта
	Output     interface{}
	Transport  interface{}
	Repository interface{}
	Service    interface{}
}

// Options управляет единичным запуском генерации.
type Options struct {
	SchemaPath    string
	OutputDir     string
	Layers        []string // nil = все слои
	DryRun        bool
	Force         bool // перезаписать пользовательские (несгенерированные) файлы
	ForceBreaking bool
	SourceFile    string
	Version       string
}

// WriteResult фиксирует, что произошло с одним файлом во время запуска.
type WriteResult struct {
	Path   string
	Action string // created | updated | skipped | dry-run
}

// Engine оркестрирует генерацию кода.
type Engine struct {
	generators map[string]LayerGenerator
}

// NewEngine создаёт Engine без зарегистрированных генераторов.
func NewEngine() *Engine {
	return &Engine{generators: make(map[string]LayerGenerator)}
}

// Register добавляет LayerGenerator. Вызывает панику при дублировании имени слоя.
func (e *Engine) Register(g LayerGenerator) {
	if _, exists := e.generators[g.Layer()]; exists {
		panic(fmt.Sprintf("generator: layer %q already registered", g.Layer()))
	}
	e.generators[g.Layer()] = g
}

// Run выполняет полный конвейер генерации и возвращает результаты по каждому файлу.
func (e *Engine) Run(ctx context.Context, data *TemplateData, opts Options) ([]WriteResult, error) {
	layers := opts.Layers
	if len(layers) == 0 {
		layers = []string{"domain", "repository", "service", "handler"}
	}

	// Передаём определённый выходной каталог в данные шаблона, чтобы генераторы слоёв
	// могли вычислять правильные пути к файлам, не зная параметры движка.
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

	// Проверяем, существует ли файл и принадлежит ли он нам
	if _, err := os.Stat(f.Path); err == nil {
		isGen, err := IsGenerated(f.Path)
		if err != nil {
			return WriteResult{}, err
		}

		if !isGen && !f.Protected && !opts.Force {
			// Несгенерированная заглушка, которую пользователь мог отредактировать — пропускаем
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

// ExpandOutputDir раскрывает {{.Domain}} в строке шаблона output_dir.
func ExpandOutputDir(tmpl, domain string) string {
	return strings.ReplaceAll(tmpl, "{{.Domain}}", strings.ToLower(domain))
}
