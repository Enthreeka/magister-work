package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const defaultOpenAIModel  = "gpt-4o"
const defaultOpenAIKeyEnv = "OPENAI_API_KEY"

// OpenAIProvider генерирует бизнес-логику с использованием OpenAI-совместимого API.
// Работает с OpenAI, OpenRouter или любым другим OpenAI-совместимым эндпоинтом.
// API-ключ читается из переменной окружения, указанной в ApiKeyEnv (по умолчанию: OPENAI_API_KEY).
type OpenAIProvider struct {
	// Model переопределяет модель по умолчанию.
	// По умолчанию используется gpt-4o, если пусто.
	Model string
	// ApiKeyEnv — имя переменной окружения, содержащей API-ключ.
	// По умолчанию OPENAI_API_KEY, если пусто.
	ApiKeyEnv string
	// BaseURL переопределяет базовый URL API.
	// Используйте https://openrouter.ai/api/v1 для OpenRouter.
	// По умолчанию используется официальный эндпоинт OpenAI, если пусто.
	BaseURL string
}

func (p OpenAIProvider) Name() string { return "openai" }

func (p OpenAIProvider) GenerateMethodBody(ctx context.Context, req MethodRequest) (string, error) {
	envVar := p.ApiKeyEnv
	if envVar == "" {
		envVar = defaultOpenAIKeyEnv
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return "", fmt.Errorf("openai provider: env variable %q is not set", envVar)
	}

	model := p.Model
	if model == "" {
		model = defaultOpenAIModel
	}

	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if p.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(p.BaseURL))
	}
	client := openai.NewClient(opts...)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt()),
			openai.UserMessage(buildUserPrompt(req)),
		},
		MaxTokens: openai.Int(2048),
	})
	if err != nil {
		return "", fmt.Errorf("openai provider: API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai provider: empty response from API")
	}

	body := resp.Choices[0].Message.Content
	body = stripCodeFences(body)

	return body, nil
}
