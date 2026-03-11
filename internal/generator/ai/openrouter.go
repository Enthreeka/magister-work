package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	openRouterBaseURL    = "https://openrouter.ai/api/v1"
	defaultOpenRouterModel  = "openai/gpt-4o"
	defaultOpenRouterKeyEnv = "OPENROUTER_API_KEY"
)

// OpenRouterProvider генерирует бизнес-логику через OpenRouter (openrouter.ai).
// OpenRouter предоставляет OpenAI-совместимый API, поэтому используется SDK openai-go
// с пользовательским базовым URL.
//
// API-ключ читается из переменной окружения, указанной в ApiKeyEnv
// (по умолчанию: OPENROUTER_API_KEY).
type OpenRouterProvider struct {
	// Model — идентификатор модели, как указано на openrouter.ai, например:
	//   "openai/gpt-4o", "anthropic/claude-opus-4", "meta-llama/llama-3.1-70b-instruct"
	// По умолчанию используется "openai/gpt-4o", если пусто.
	Model string
	// ApiKeyEnv — имя переменной окружения, содержащей API-ключ.
	// По умолчанию OPENROUTER_API_KEY, если пусто.
	ApiKeyEnv string
}

func (p OpenRouterProvider) Name() string { return "openrouter" }

func (p OpenRouterProvider) GenerateMethodBody(ctx context.Context, req MethodRequest) (string, error) {
	envVar := p.ApiKeyEnv
	if envVar == "" {
		envVar = defaultOpenRouterKeyEnv
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return "", fmt.Errorf("openrouter provider: env variable %q is not set", envVar)
	}

	model := p.Model
	if model == "" {
		model = defaultOpenRouterModel
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(openRouterBaseURL),
	)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt()),
			openai.UserMessage(buildUserPrompt(req)),
		},
		MaxTokens: openai.Int(2048),
	})
	if err != nil {
		return "", fmt.Errorf("openrouter provider: API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openrouter provider: empty response from API")
	}

	body := resp.Choices[0].Message.Content
	body = stripCodeFences(body)

	return body, nil
}
