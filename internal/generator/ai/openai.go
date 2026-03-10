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

// OpenAIProvider generates business logic using the OpenAI-compatible API.
// Works with OpenAI, OpenRouter, or any other OpenAI-compatible endpoint.
// API key is read from the env variable specified in ApiKeyEnv (default: OPENAI_API_KEY).
type OpenAIProvider struct {
	// Model overrides the default model.
	// Defaults to gpt-4o if empty.
	Model string
	// ApiKeyEnv is the name of the environment variable holding the API key.
	// Defaults to OPENAI_API_KEY if empty.
	ApiKeyEnv string
	// BaseURL overrides the API base URL.
	// Use https://openrouter.ai/api/v1 for OpenRouter.
	// Defaults to the official OpenAI endpoint if empty.
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
