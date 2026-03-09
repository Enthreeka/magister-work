package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultModel = "claude-opus-4-6"

const defaultAPIKeyEnv = "ANTHROPIC_API_KEY"

// AnthropicProvider generates business logic using the Claude API.
// API key is read from the env variable specified in ApiKeyEnv (default: ANTHROPIC_API_KEY).
type AnthropicProvider struct {
	// Model overrides the default Claude model.
	// Defaults to claude-opus-4-6 if empty.
	Model string
	// ApiKeyEnv is the name of the environment variable holding the API key.
	// Defaults to ANTHROPIC_API_KEY if empty.
	ApiKeyEnv string
}

func (p AnthropicProvider) Name() string { return "anthropic" }

func (p AnthropicProvider) GenerateMethodBody(ctx context.Context, req MethodRequest) (string, error) {
	envVar := p.ApiKeyEnv
	if envVar == "" {
		envVar = defaultAPIKeyEnv
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return "", fmt.Errorf("anthropic provider: env variable %q is not set", envVar)
	}

	model := p.Model
	if model == "" {
		model = defaultModel
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(buildUserPrompt(req)),
			),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic provider: API call failed: %w", err)
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("anthropic provider: empty response from API")
	}

	body := resp.Content[0].Text
	body = stripCodeFences(body)

	return body, nil
}

func systemPrompt() string {
	return `You are an expert Go developer generating production-quality business logic.

Rules:
- Return ONLY the function body — no signature, no package declaration, no imports.
- The code must compile as-is when inserted into the function body.
- Use only the dependencies explicitly listed in the prompt.
- Wrap all errors using fmt.Errorf("%w: ...", domainErr) pattern.
- Do not add import statements — they are handled separately.
- Do not repeat input validation that is already described as handled.
- Write idiomatic, concise Go code.`
}

func buildUserPrompt(req MethodRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Generate the body for this Go service method:\n\n"))
	sb.WriteString(fmt.Sprintf("  func (s *service) %s(ctx context.Context, req *%s) (*%s, error)\n\n",
		req.MethodName, req.InputType, req.OutputType))

	sb.WriteString("Available repository method:\n")
	sb.WriteString(fmt.Sprintf("  s.repo.%s(ctx, req) → (*%s, error)\n\n", req.MethodName, req.OutputType))

	sb.WriteString("Domain errors to use for wrapping:\n")
	sb.WriteString(fmt.Sprintf("  %s — business rule violations (conflicts, duplicates)\n", req.DomainErrDomain))
	sb.WriteString(fmt.Sprintf("  %s — input validation failures\n", req.DomainErrValidation))
	sb.WriteString(fmt.Sprintf("  %s — infrastructure / unexpected errors\n", req.DomainErrInternal))
	sb.WriteString(fmt.Sprintf("  %s — resource not found\n\n", req.DomainErrNotFound))

	if len(req.RequiredFields) > 0 {
		sb.WriteString("Already validated (do NOT repeat):\n")
		for _, f := range req.RequiredFields {
			sb.WriteString(fmt.Sprintf("  - %s (%s) is required\n", f.Name, f.GoType))
		}
		sb.WriteString("\n")
	}

	if req.Description != "" {
		sb.WriteString("Business logic to implement:\n")
		sb.WriteString("  " + strings.ReplaceAll(strings.TrimSpace(req.Description), "\n", "\n  "))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Business logic to implement:\n")
		sb.WriteString(fmt.Sprintf("  Perform a %s operation. Call s.repo.%s, wrap errors with domain errors, return result.\n\n",
			req.Operation, req.MethodName))
	}

	if len(req.FeatureFlags) > 0 {
		sb.WriteString("Feature flags (implement as runtime if-checks, default value shown):\n")
		for _, ff := range req.FeatureFlags {
			def := "false"
			if ff.DefaultValue {
				def = "true"
			}
			sb.WriteString(fmt.Sprintf("  - %s (default: %s): %s\n", ff.Name, def, ff.Description))
		}
		sb.WriteString("\nFor each feature flag generate:\n")
		sb.WriteString("  if flags.Get(\"flag_name\") {\n    // feature logic\n  }\n\n")
	}

	sb.WriteString("Return only the raw Go code for the function body, no markdown fences.")

	return sb.String()
}

// stripCodeFences removes ```go ... ``` or ``` ... ``` wrappers if the model
// includes them despite the instructions.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	for _, fence := range []string{"```go", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			s = strings.TrimSuffix(s, "```")
			s = strings.TrimSpace(s)
			break
		}
	}
	return s
}
