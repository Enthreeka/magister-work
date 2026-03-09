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
		Short: "Code generation from requirements YAML",
		Long: `codegen generates Clean Architecture Go code from a declarative
requirements file (system-gen.yaml).

Supported layers: domain, repository, service, handler.
Supported repository strategies: native (pgx/sqlx), sqlc (generate/existing).`,
		SilenceUsage: true,
	}

	root.AddCommand(
		newGenerateCmd(),
		newValidateCmd(),
		newDiffCmd(),
		newVersionCmd(),
	)

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the codegen version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("codegen v%s\n", version)
		},
	}
}
