package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Enthreeka/magister-work/internal/generator/compat"
	"github.com/Enthreeka/magister-work/internal/schema"
)

func newDiffCmd() *cobra.Command {
	var schemaPath string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show schema changes since the last generation (reads .codegen.lock)",
		Example: `  codegen diff
  codegen diff --schema ./user/system-gen.yaml`,
		RunE: func(_ *cobra.Command, _ []string) error {
			s, err := schema.ParseFile(schemaPath)
			if err != nil {
				return err
			}

			changes, err := compat.Check(s)
			if err != nil {
				return fmt.Errorf("diff: %w", err)
			}

			if len(changes) == 0 {
				fmt.Println("No changes since last generation.")
				return nil
			}

			fmt.Printf("%d change(s) detected:\n", len(changes))
			for _, c := range changes {
				fmt.Printf("  %s\n", c.String())
			}

			if compat.HasErrors(changes) {
				fmt.Println("\nThis schema has breaking changes. Use --force-breaking with generate to proceed.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&schemaPath, "schema", "s", "system-gen.yaml", "path to the requirements YAML file")
	return cmd
}
