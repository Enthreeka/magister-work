package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Enthreeka/magister-work/internal/schema"
)

func newValidateCmd() *cobra.Command {
	var schemaPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Проверить system-gen.yaml без генерации каких-либо файлов",
		Example: `  codegen validate
  codegen validate --schema ./user/system-gen.yaml`,
		RunE: func(_ *cobra.Command, _ []string) error {
			s, err := schema.ParseFile(schemaPath)
			if err != nil {
				return err
			}
			errs := schema.Validate(s)
			if len(errs) > 0 {
				return fmt.Errorf("%s", schema.FormatErrors(errs))
			}
			fmt.Printf("✓ %s is valid\n", schemaPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&schemaPath, "schema", "s", "system-gen.yaml", "путь к файлу требований YAML")
	return cmd
}
