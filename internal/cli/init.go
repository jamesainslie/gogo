package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/user/gogo/internal/generator"
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
	)

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new Go project",
		Long: color.GreenString(`Initialize a new Go project with the specified template and blueprint.

Examples:
  gogo init myproject --template=cli
  gogo init myapi --template=api --blueprint=web-stack
  gogo init --wizard  # Interactive mode`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := ""
			if len(args) > 0 {
				projectName = args[0]
			}

			if wizard || (projectName == "" && moduleName == "") {
				return fmt.Errorf("wizard mode not yet implemented")
			}

			// Set up generator
			engine := templates.NewEngine()
			repo := templates.NewRepository()
			gen := generator.NewProjectGenerator(engine, repo)

			// Build options
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

			color.Yellow("Initializing project: %s", projectName)
			color.Yellow("Template: %s", template)
			if blueprint != "" {
				color.Yellow("Blueprint: %s", blueprint)
			}
			color.Yellow("Module: %s", moduleName)

			result, err := gen.InitProject(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("failed to initialize project: %w", err)
			}

			if result.Success {
				color.Green(result.Message)
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
	cmd.Flags().BoolVar(&wizard, "wizard", false, "Interactive wizard mode")

	return cmd
}
