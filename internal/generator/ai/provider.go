// Package ai defines the interface for pluggable business-logic generation providers.
// The only built-in provider is NoopProvider which emits a TODO stub.
// Real AI-backed providers (Anthropic, OpenAI, Ollama) are registered separately
// and are not implemented yet — this package only establishes the contract.
package ai

import "context"

// Dependency describes a dependency available to a service method
// (e.g. a repository, a cache client, an external HTTP service).
type Dependency struct {
	Name string
	Type string
}

// Example is an optional input/output pair that helps an AI provider
// understand expected behaviour.
type Example struct {
	Input  string
	Output string
}

// FieldInfo describes a single input field passed to MethodRequest.
type FieldInfo struct {
	Name   string
	GoType string
}

// MethodRequest is the context an AI provider receives when asked to
// generate or scaffold the body of a single service method.
type MethodRequest struct {
	MethodName   string
	InputType    string
	OutputType   string
	Operation    string // insert | select | update | delete
	Dependencies []Dependency
	FeatureFlags []FeatureFlag
	Description  string
	Examples     []Example

	// RequiredFields lists input fields marked required: true in the schema.
	// Used by TemplateProvider to generate validation checks.
	RequiredFields []FieldInfo

	// Domain error identifiers for use in generated error wrapping.
	DomainErrValidation string // e.g. "domain.ErrUserValidation"
	DomainErrInternal   string // e.g. "domain.ErrUserInternal"
	DomainErrNotFound   string // e.g. "domain.ErrUserNotFound"
}

// FeatureFlag is a conditional-logic hint that the AI provider can use
// to generate if/switch branches in the service method body.
type FeatureFlag struct {
	Name         string
	Description  string
	DefaultValue bool
}

// BusinessLogicProvider generates (or scaffolds) the body of service methods.
// Implementations must be safe for concurrent use.
type BusinessLogicProvider interface {
	// Name returns the unique identifier of this provider (e.g. "noop", "anthropic").
	Name() string

	// GenerateMethodBody returns Go source code for the body of the method
	// described by req. The returned string must be valid Go — it will be
	// inserted verbatim inside the generated function.
	GenerateMethodBody(ctx context.Context, req MethodRequest) (string, error)
}
