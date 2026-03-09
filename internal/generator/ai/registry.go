package ai

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	providers = map[string]BusinessLogicProvider{
		"noop":      NoopProvider{},
		"template":  TemplateProvider{},
		"anthropic": AnthropicProvider{},
		"openai":    OpenAIProvider{},
	}
)

// Register adds a provider to the global registry.
// It panics if a provider with the same name has already been registered —
// this mirrors the pattern used by database/sql drivers.
func Register(p BusinessLogicProvider) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := providers[p.Name()]; exists {
		panic(fmt.Sprintf("ai: provider %q already registered", p.Name()))
	}
	providers[p.Name()] = p
}

// Get returns a registered provider by name.
func Get(name string) (BusinessLogicProvider, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("ai: unknown provider %q (did you forget to import it?)", name)
	}
	return p, nil
}
