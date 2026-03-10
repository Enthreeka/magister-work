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

// OpenRouterProvider generates business logic via OpenRouter (openrouter.ai).
// OpenRouter exposes an OpenAI-compatible API, so it reuses the openai-go SDK
// with a custom base URL.
//
// API key is read from the env variable specified in ApiKeyEnv
// (default: OPENROUTER_API_KEY).
type OpenRouterProvider struct {
	// Model is the model identifier as listed on openrouter.ai, e.g.:
	//   "openai/gpt-4o", "anthropic/claude-opus-4", "meta-llama/llama-3.1-70b-instruct"
	// Defaults to "openai/gpt-4o" if empty.
	Model string
	// ApiKeyEnv is the name of the environment variable holding the API key.
	// Defaults to OPENROUTER_API_KEY if empty.
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
