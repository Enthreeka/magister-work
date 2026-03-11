package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "codegen",
		Short: "Генерация кода из YAML требований",
		Long: `codegen генерирует Go-код по архитектуре Clean Architecture из декларативного
файла требований (system-gen.yaml).

Поддерживаемые слои: domain, repository, service, handler.
Поддерживаемые стратегии репозитория: native (pgx/sqlx), sqlc (generate/existing).`,
		SilenceUsage: true,
	}

	root.AddCommand(
		newGenerateCmd(),
		newValidateCmd(),
		newDiffCmd(),
		newAICmd(),
		newVersionCmd(),
	)

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Вывести версию codegen",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("codegen v%s\n", version)
		},
	}
}
