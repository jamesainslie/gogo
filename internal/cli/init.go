package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/user/gogo/internal/generator"
	"github.com/user/gogo/internal/prompt"
	"github.com/user/gogo/internal/templates"
)

func newInitCommand() *cobra.Command {
	var (
		template   string
		blueprint  string
		moduleName string
		author     string
		license    string
		gitInit    bool
		force      bool
		wizard     bool
		noWizard   bool
	)

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new Go project",
		Long: color.GreenString(`Initialize a new Go project with the specified template and blueprint.

By default, runs in interactive wizard mode for the best user experience.
Use --no-wizard to disable interactive mode when providing all flags.

Examples:
  gogo init                                          # Interactive wizard (default)
  gogo init myproject --module=github.com/user/myproject --no-wizard
  gogo init myapi --template=api --blueprint=web-stack --no-wizard`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := ""
			if len(args) > 0 {
				projectName = args[0]
			}

			// Set up generator
			engine := templates.NewEngine()
			repo := templates.NewRepository()
			gen := generator.NewProjectGenerator(engine, repo)

			// Build initial options
			opts := generator.InitOptions{
				ProjectName: projectName,
				ModuleName:  moduleName,
				Template:    template,
				Blueprint:   blueprint,
				Author:      author,
				License:     license,
				GoVersion:   goVersion,
				OutputDir:   outputDir,
				Description: fmt.Sprintf("A %s project", template),
				GitInit:     gitInit,
				Force:       force,
				DryRun:      dryRun,
			}

			// Determine if we should run the wizard (default behavior)
			needsWizard := !noWizard

			// Always run wizard if explicitly requested (overrides --no-wizard)
			if wizard {
				needsWizard = true
			}

			// Skip wizard if user provided sufficient flags (unless --wizard is explicit)
			if !wizard && projectName != "" && moduleName != "" {
				needsWizard = false
			}

			if needsWizard {
				color.Cyan("Starting interactive wizard...")
				fmt.Println()

				wizard := prompt.NewWizard()
				wizardOptions, err := wizard.RunInitWizard(cmd.Context(), opts)
				if err != nil {
					return fmt.Errorf("wizard failed: %w", err)
				}

				// Convert wizard options to generator options
				opts = wizardOptions.ConvertToInitOptions()
				opts.DryRun = dryRun // Preserve the dry-run flag from CLI
			}

			// Validate that we have required options
			if opts.ProjectName == "" {
				return fmt.Errorf("project name is required (run without --no-wizard for interactive mode)")
			}
			if opts.ModuleName == "" {
				return fmt.Errorf("module name is required (run without --no-wizard for interactive mode)")
			}

			// Show what we're doing (unless we just showed it in wizard)
			if !needsWizard {
				color.Yellow("Initializing project: %s", opts.ProjectName)
				color.Yellow("Template: %s", opts.Template)
				if opts.Blueprint != "" {
					color.Yellow("Blueprint: %s", opts.Blueprint)
				}
				color.Yellow("Module: %s", opts.ModuleName)
			}

			result, err := gen.InitProject(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("failed to initialize project: %w", err)
			}

			if result.Success {
				color.Green(result.Message)
				if opts.GitInit {
					color.Green("Git repository initialized")
				}
			} else {
				color.Red("Project initialization failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&template, "template", "cli", "Project template (cli, library, api, grpc, microservice)")
	cmd.Flags().StringVar(&blueprint, "blueprint", "", "Stack blueprint name (web-stack, cli-stack, grpc-stack, microservice-stack)")
	cmd.Flags().StringVar(&moduleName, "module", "", "Go module name (e.g., github.com/user/project)")
	cmd.Flags().StringVar(&author, "author", "", "Author name for generated files")
	cmd.Flags().StringVar(&license, "license", "MIT", "License type (MIT, Apache, GPL)")
	cmd.Flags().BoolVar(&gitInit, "git-init", false, "Initialize git repository")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&wizard, "wizard", false, "Force interactive wizard mode (overrides --no-wizard)")
	cmd.Flags().BoolVar(&noWizard, "no-wizard", false, "Disable interactive wizard mode")

	return cmd
}
