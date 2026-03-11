// Package ai определяет интерфейс для подключаемых провайдеров генерации бизнес-логики.
// Единственный встроенный провайдер — NoopProvider, генерирующий заглушку с TODO.
// Реальные провайдеры на базе ИИ (Anthropic, OpenAI, Ollama) регистрируются отдельно
// и пока не реализованы — этот пакет только устанавливает контракт.
package ai

import "context"

// Dependency описывает зависимость, доступную методу сервиса
// (например, репозиторий, клиент кэша, внешний HTTP-сервис).
type Dependency struct {
	Name string
	Type string
}

// Example — необязательная пара вход/выход, помогающая провайдеру ИИ
// понять ожидаемое поведение.
type Example struct {
	Input  string
	Output string
}

// FieldInfo описывает одно входное поле, передаваемое в MethodRequest.
type FieldInfo struct {
	Name   string
	GoType string
}

// MethodRequest — контекст, который провайдер ИИ получает при запросе на
// генерацию или создание тела одного метода сервиса.
type MethodRequest struct {
	MethodName   string
	InputType    string
	OutputType   string
	Operation    string // insert | select | update | delete
	Dependencies []Dependency
	FeatureFlags []FeatureFlag
	Description  string
	Examples     []Example

	// RequiredFields перечисляет входные поля, помеченные required: true в схеме.
	// Используется TemplateProvider для генерации проверок валидации.
	RequiredFields []FieldInfo

	// Идентификаторы ошибок домена для использования при оборачивании ошибок в сгенерированном коде.
	DomainErrDomain     string // например, "domain.ErrUserDomain"
	DomainErrValidation string // например, "domain.ErrUserValidation"
	DomainErrInternal   string // например, "domain.ErrUserInternal"
	DomainErrNotFound   string // например, "domain.ErrUserNotFound"
}

// FeatureFlag — подсказка условной логики, которую провайдер ИИ может использовать
// для генерации ветвей if/switch в теле метода сервиса.
type FeatureFlag struct {
	Name         string
	Description  string
	DefaultValue bool
}

// BusinessLogicProvider генерирует (или создаёт заглушку) тела методов сервиса.
// Реализации должны быть безопасны для конкурентного использования.
type BusinessLogicProvider interface {
	// Name возвращает уникальный идентификатор этого провайдера (например, "noop", "anthropic").
	Name() string

	// GenerateMethodBody возвращает исходный код Go для тела метода,
	// описанного в req. Возвращаемая строка должна быть валидным Go — она будет
	// вставлена дословно внутрь сгенерированной функции.
	GenerateMethodBody(ctx context.Context, req MethodRequest) (string, error)
}
