// Package compat detects breaking changes between two versions of a Schema.
// It reads and writes a .codegen.lock file (analogous to go.sum) that stores
// the last successfully generated schema fingerprint.
package compat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Enthreeka/magister-work/internal/schema"
)

const LockFile = ".codegen.lock"

// Severity classifies the impact of a detected change.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// BreakingChange describes a single incompatible schema modification.
type BreakingChange struct {
	Path     string
	Message  string
	Severity Severity
}

func (b BreakingChange) String() string {
	return fmt.Sprintf("[%s] %s: %s", strings.ToUpper(string(b.Severity)), b.Path, b.Message)
}

// lockEntry is the persisted snapshot of a generated schema.
type lockEntry struct {
	Domain    string         `json:"domain"`
	Transport lockTransport  `json:"transport"`
	Input     []lockField    `json:"input"`
	Output    []lockField    `json:"output"`
	Operation string         `json:"operation"`
}

type lockTransport struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

type lockField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// lockFile is the full contents of .codegen.lock, keyed by domain.
type lockFile map[string]lockEntry

// Check compares the incoming schema against the stored lock entry.
// It returns all detected breaking changes. An empty slice means the
// schema is backward-compatible with the previous generation.
func Check(s *schema.Schema) ([]BreakingChange, error) {
	prev, err := loadLock()
	if err != nil {
		return nil, err
	}

	entry, exists := prev[s.Domain]
	if !exists {
		// First time this domain is generated — no previous state to compare.
		return nil, nil
	}

	return compare(entry, s), nil
}

// SaveLock persists the schema as the new lock entry after a successful generation.
func SaveLock(s *schema.Schema) error {
	lock, err := loadLock()
	if err != nil {
		return err
	}
	lock[s.Domain] = toEntry(s)
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("compat: marshal lock: %w", err)
	}
	if err := os.WriteFile(LockFile, data, 0o644); err != nil {
		return fmt.Errorf("compat: write %s: %w", LockFile, err)
	}
	return nil
}

func loadLock() (lockFile, error) {
	data, err := os.ReadFile(LockFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(lockFile), nil
		}
		return nil, fmt.Errorf("compat: read %s: %w", LockFile, err)
	}
	var lf lockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("compat: parse %s: %w", LockFile, err)
	}
	return lf, nil
}

func toEntry(s *schema.Schema) lockEntry {
	input := make([]lockField, len(s.Input))
	for i, f := range s.Input {
		input[i] = lockField{Name: f.Name, Type: f.Type, Required: f.Required}
	}
	output := make([]lockField, len(s.Output))
	for i, f := range s.Output {
		output[i] = lockField{Name: f.Name, Type: f.Type}
	}
	return lockEntry{
		Domain:    s.Domain,
		Transport: lockTransport{Method: s.Transport.Method, URL: s.Transport.URL},
		Input:     input,
		Output:    output,
		Operation: s.Repository.Operation,
	}
}

func compare(prev lockEntry, s *schema.Schema) []BreakingChange {
	var changes []BreakingChange

	// Transport: method change is always breaking
	if !strings.EqualFold(prev.Transport.Method, s.Transport.Method) {
		changes = append(changes, BreakingChange{
			Path:     "transport.method",
			Message:  fmt.Sprintf("changed from %q to %q", prev.Transport.Method, s.Transport.Method),
			Severity: SeverityError,
		})
	}
	// URL change is a warning (clients may still work with redirects)
	if prev.Transport.URL != s.Transport.URL {
		changes = append(changes, BreakingChange{
			Path:     "transport.url",
			Message:  fmt.Sprintf("changed from %q to %q", prev.Transport.URL, s.Transport.URL),
			Severity: SeverityWarning,
		})
	}

	// Repository operation change is always breaking
	if !strings.EqualFold(prev.Operation, s.Repository.Operation) {
		changes = append(changes, BreakingChange{
			Path:     "repository.operation",
			Message:  fmt.Sprintf("changed from %q to %q", prev.Operation, s.Repository.Operation),
			Severity: SeverityError,
		})
	}

	// Input fields: removals and type changes are breaking
	prevInput := indexFields(prev.Input)
	for _, curr := range s.Input {
		old, existed := prevInput[curr.Name]
		if !existed {
			continue // new field — non-breaking
		}
		if old.Type != curr.Type {
			changes = append(changes, BreakingChange{
				Path:     fmt.Sprintf("input[%s].type", curr.Name),
				Message:  fmt.Sprintf("type changed from %q to %q", old.Type, curr.Type),
				Severity: SeverityError,
			})
		}
	}
	currInput := indexFields(toSchemaFields(s.Input))
	for _, old := range prev.Input {
		if old.Required {
			if _, exists := currInput[old.Name]; !exists {
				changes = append(changes, BreakingChange{
					Path:     fmt.Sprintf("input[%s]", old.Name),
					Message:  "required field removed",
					Severity: SeverityError,
				})
			}
		}
	}

	// Output fields: removals are breaking (clients depend on response shape)
	currOutput := indexFields(toSchemaFields(s.Output))
	for _, old := range prev.Output {
		if _, exists := currOutput[old.Name]; !exists {
			changes = append(changes, BreakingChange{
				Path:     fmt.Sprintf("output[%s]", old.Name),
				Message:  "output field removed",
				Severity: SeverityWarning,
			})
		}
	}

	return changes
}

func indexFields(fields []lockField) map[string]lockField {
	m := make(map[string]lockField, len(fields))
	for _, f := range fields {
		m[f.Name] = f
	}
	return m
}

func toSchemaFields(fields []schema.Field) []lockField {
	out := make([]lockField, len(fields))
	for i, f := range fields {
		out[i] = lockField{Name: f.Name, Type: f.Type, Required: f.Required}
	}
	return out
}

// HasErrors returns true if any of the changes have SeverityError.
func HasErrors(changes []BreakingChange) bool {
	for _, c := range changes {
		if c.Severity == SeverityError {
			return true
		}
	}
	return false
}
