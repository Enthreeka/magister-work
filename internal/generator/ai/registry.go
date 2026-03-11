package ai

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	providers = map[string]BusinessLogicProvider{
		"noop":       NoopProvider{},
		"template":   TemplateProvider{},
		"anthropic":  AnthropicProvider{},
		"openai":     OpenAIProvider{},
		"openrouter": OpenRouterProvider{},
	}
)

// Register добавляет провайдера в глобальный реестр.
// Вызывает панику, если провайдер с таким же именем уже зарегистрирован —
// аналогично паттерну, используемому драйверами database/sql.
func Register(p BusinessLogicProvider) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := providers[p.Name()]; exists {
		panic(fmt.Sprintf("ai: provider %q already registered", p.Name()))
	}
	providers[p.Name()] = p
}

// Get возвращает зарегистрированного провайдера по имени.
func Get(name string) (BusinessLogicProvider, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("ai: unknown provider %q (did you forget to import it?)", name)
	}
	return p, nil
}
