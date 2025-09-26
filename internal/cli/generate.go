package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/user/gogo/internal/components"
)

func newGenerateCommand() *cobra.Command {
	var (
		componentType string
		name          string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate project components",
		Long: color.GreenString(`Generate components for an existing Go project.

Examples:
  gogo generate --type=handler --name=Health
  gogo generate --type=model --name=User
  gogo generate --type=test --name=service`),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set up component generator
			generator := components.NewGenerator()

			// Build options
			opts := components.GenerateOptions{
				Type:      componentType,
				Name:      name,
				OutputDir: ".",
				DryRun:    false, // Will be handled by global flag
			}

			color.Yellow("Generating component: %s", componentType)
			color.Yellow("Name: %s", name)

			result, err := generator.Generate(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("failed to generate component: %w", err)
			}

			if result.Success {
				color.Green(result.Message)
				if len(result.Files) > 0 {
					color.Cyan("Generated files:")
					for _, file := range result.Files {
						color.Cyan("  - %s", file)
					}
				}
			} else {
				color.Red("Component generation failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&componentType, "type", "", "Component type (handler, model, service, migration, middleware, test)")
	cmd.Flags().StringVar(&name, "name", "", "Component name")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
