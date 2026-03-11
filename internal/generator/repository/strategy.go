// Package repository определяет подключаемый интерфейс стратегии для генерации
// слоя репозитория. Каждая стратегия решает, как создаётся код репозитория:
// напрямую через codegen (native) или с помощью внешнего инструмента (sqlc).
package repository

import (
	"context"

	"github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/schema"
)

// Options содержит параметры времени генерации, переданные от движка.
type Options struct {
	OutputDir    string
	SourceFile   string
	Version      string
	DryRun       bool
	DomainImport string // полный путь импорта Go для пакета domain
}

// RepositoryContract описывает интерфейс, от которого зависят слои сервиса и обработчика.
// Он выводится из схемы независимо от стратегии.
type RepositoryContract struct {
	InterfaceName string
	MethodName    string
	InputType     string
	OutputType    string
}

// Strategy — точка расширения для генерации слоя репозитория.
type Strategy interface {
	// Name возвращает уникальный идентификатор этой стратегии ("native", "sqlc").
	Name() string

	// Prepare выполняет подготовительные шаги перед генерацией (например, запуск sqlc generate).
	// Вызывается один раз перед Files.
	Prepare(ctx context.Context, s *schema.Schema, opts Options) error

	// Contract выводит контракт интерфейса репозитория из схемы.
	// Контракт используется генераторами сервиса и обработчика.
	Contract(s *schema.Schema) (*RepositoryContract, error)

	// Files возвращает набор файлов, которые эта стратегия хочет записать.
	// Nil-срез означает, что стратегия делегирует запись файлов внешнему
	// инструменту (например, sqlc), и codegen не должен сам записывать файлы репозитория.
	Files(s *schema.Schema, opts Options) ([]generator.File, error)
}
