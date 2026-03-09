package schema

// Schema — top-level structure of system-gen.yaml.
type Schema struct {
	Version    string      `yaml:"version"`
	Module     string      `yaml:"module"`
	Domain     string      `yaml:"domain"`
	Transport  Transport   `yaml:"transport"`
	Input      []Field     `yaml:"input"`
	Output     []Field     `yaml:"output"`
	Repository Repository  `yaml:"repository"`
	Generate   Generate    `yaml:"generate"`
	Service    *Service    `yaml:"service,omitempty"`
	AIProvider *AIProvider `yaml:"ai_provider,omitempty"`
}

// Transport describes the HTTP layer of the generated endpoint.
type Transport struct {
	Framework string `yaml:"framework"` // gin | fiber | echo
	Method    string `yaml:"method"`    // GET | POST | PUT | DELETE | PATCH
	URL       string `yaml:"url"`
}

// Field describes a single input or output parameter.
type Field struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Source   string `yaml:"source,omitempty"`   // body | query | header | path
	Required bool   `yaml:"required,omitempty"`
}

// Repository describes the storage layer configuration.
type Repository struct {
	Strategy  string      `yaml:"strategy"`          // native | sqlc
	Driver    string      `yaml:"driver,omitempty"`  // pgx | sqlx
	Table     string      `yaml:"table,omitempty"`
	Schema    string      `yaml:"schema,omitempty"`
	Operation string      `yaml:"operation,omitempty"` // insert | select | update | delete
	Fields    []string    `yaml:"fields,omitempty"`
	Sqlc      *SqlcConfig `yaml:"sqlc,omitempty"`
}

// SqlcConfig holds sqlc-specific options when strategy is "sqlc".
type SqlcConfig struct {
	Mode   string `yaml:"mode"`            // generate | existing
	Config string `yaml:"config,omitempty"` // path to sqlc.yaml (mode: existing)
}

// Generate controls which layers codegen produces and where.
type Generate struct {
	Layers    []string `yaml:"layers"`     // domain | repository | service | handler
	OutputDir string   `yaml:"output_dir"` // supports {{.Domain}} template
}

// Service holds optional hints for business-logic generation.
type Service struct {
	Description  string        `yaml:"description,omitempty"`
	FeatureFlags []FeatureFlag `yaml:"feature_flags,omitempty"`
}

// FeatureFlag is a conditional-logic hint consumed by AI providers (future).
type FeatureFlag struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description,omitempty"`
	DefaultValue bool   `yaml:"default,omitempty"`
}

// AIProvider selects the business-logic generation backend (future).
type AIProvider struct {
	Name  string `yaml:"name"`            // anthropic | openai | ollama | noop
	Model string `yaml:"model,omitempty"` // e.g. claude-opus-4-6
}
