// Package compat обнаруживает ломающие изменения между двумя версиями Schema.
// Пакет читает и записывает файл .codegen.lock (аналог go.sum), хранящий
// отпечаток последней успешно сгенерированной схемы.
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

// Severity классифицирует влияние обнаруженного изменения.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// BreakingChange описывает единственное несовместимое изменение схемы.
type BreakingChange struct {
	Path     string
	Message  string
	Severity Severity
}

func (b BreakingChange) String() string {
	return fmt.Sprintf("[%s] %s: %s", strings.ToUpper(string(b.Severity)), b.Path, b.Message)
}

// lockEntry — сохранённый снимок сгенерированной схемы.
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

// lockFile — полное содержимое .codegen.lock, индексированное по домену.
type lockFile map[string]lockEntry

// Check сравнивает входящую схему с сохранённой записью lock.
// Возвращает все обнаруженные ломающие изменения. Пустой срез означает,
// что схема обратно совместима с предыдущей генерацией.
func Check(s *schema.Schema) ([]BreakingChange, error) {
	prev, err := loadLock()
	if err != nil {
		return nil, err
	}

	entry, exists := prev[s.Domain]
	if !exists {
		// Домен генерируется впервые — нет предыдущего состояния для сравнения.
		return nil, nil
	}

	return compare(entry, s), nil
}

// SaveLock сохраняет схему как новую запись lock после успешной генерации.
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

	// Transport: изменение метода всегда является ломающим
	if !strings.EqualFold(prev.Transport.Method, s.Transport.Method) {
		changes = append(changes, BreakingChange{
			Path:     "transport.method",
			Message:  fmt.Sprintf("changed from %q to %q", prev.Transport.Method, s.Transport.Method),
			Severity: SeverityError,
		})
	}
	// Изменение URL является предупреждением (клиенты могут продолжать работать с перенаправлениями)
	if prev.Transport.URL != s.Transport.URL {
		changes = append(changes, BreakingChange{
			Path:     "transport.url",
			Message:  fmt.Sprintf("changed from %q to %q", prev.Transport.URL, s.Transport.URL),
			Severity: SeverityWarning,
		})
	}

	// Изменение операции репозитория всегда является ломающим
	if !strings.EqualFold(prev.Operation, s.Repository.Operation) {
		changes = append(changes, BreakingChange{
			Path:     "repository.operation",
			Message:  fmt.Sprintf("changed from %q to %q", prev.Operation, s.Repository.Operation),
			Severity: SeverityError,
		})
	}

	// Входные поля: удаления и изменения типов являются ломающими
	prevInput := indexFields(prev.Input)
	for _, curr := range s.Input {
		old, existed := prevInput[curr.Name]
		if !existed {
			continue // новое поле — не является ломающим
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

	// Выходные поля: удаления являются ломающими (клиенты зависят от формы ответа)
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

// HasErrors возвращает true, если хотя бы одно из изменений имеет уровень SeverityError.
func HasErrors(changes []BreakingChange) bool {
	for _, c := range changes {
		if c.Severity == SeverityError {
			return true
		}
	}
	return false
}
