package schema

// Schema — корневая структура файла system-gen.yaml.
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

// Transport описывает HTTP-слой генерируемого эндпоинта.
type Transport struct {
	Framework string `yaml:"framework"` // gin | fiber | echo
	Method    string `yaml:"method"`    // GET | POST | PUT | DELETE | PATCH
	URL       string `yaml:"url"`
}

// Field описывает один входной или выходной параметр.
type Field struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Source   string `yaml:"source,omitempty"`   // body | query | header | path
	Required bool   `yaml:"required,omitempty"`
}

// Repository описывает конфигурацию слоя хранилища.
type Repository struct {
	Strategy  string      `yaml:"strategy"`          // native | sqlc
	Driver    string      `yaml:"driver,omitempty"`  // pgx | sqlx
	Table     string      `yaml:"table,omitempty"`
	Schema    string      `yaml:"schema,omitempty"`
	Operation string      `yaml:"operation,omitempty"` // insert | select | update | delete
	Fields    []string    `yaml:"fields,omitempty"`
	Sqlc      *SqlcConfig `yaml:"sqlc,omitempty"`
}

// SqlcConfig содержит параметры, специфичные для sqlc, когда strategy равна "sqlc".
type SqlcConfig struct {
	Mode   string `yaml:"mode"`            // generate | existing
	Config string `yaml:"config,omitempty"` // путь к sqlc.yaml (mode: existing)
}

// Generate управляет тем, какие слои создаёт codegen и куда.
type Generate struct {
	Layers    []string `yaml:"layers"`     // domain | repository | service | handler
	OutputDir string   `yaml:"output_dir"` // поддерживает шаблон {{.Domain}}
}

// Service содержит необязательные подсказки для генерации бизнес-логики.
type Service struct {
	Description  string        `yaml:"description,omitempty"`
	FeatureFlags []FeatureFlag `yaml:"feature_flags,omitempty"`
}

// FeatureFlag — подсказка условной логики, используемая провайдерами ИИ (в будущем).
type FeatureFlag struct {
	Name         string `yaml:"name"`
	Description  string `yaml:"description,omitempty"`
	DefaultValue bool   `yaml:"default,omitempty"`
}

// AIProvider выбирает бэкенд для генерации бизнес-логики (в будущем).
type AIProvider struct {
	Name      string `yaml:"name"`                  // anthropic | openai | ollama | noop
	Model     string `yaml:"model,omitempty"`       // например, claude-opus-4-6
	ApiKeyEnv string `yaml:"api_key_env,omitempty"` // имя переменной окружения с API-ключом, например MY_ANTHROPIC_KEY
	BaseURL   string `yaml:"base_url,omitempty"`    // переопределение базового URL API (например, https://openrouter.ai/api/v1)
}
