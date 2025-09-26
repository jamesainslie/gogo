package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add components to existing project",
		Long: color.GreenString(`Add components to an existing Go project.

This is an alias for the generate command for convenience.

Examples:
  gogo add --type=handler --name=Health
  gogo add --type=model --name=User`),
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Yellow("Add command is an alias for generate")
			return fmt.Errorf("add command not fully implemented yet")
		},
	}

	return cmd
}
