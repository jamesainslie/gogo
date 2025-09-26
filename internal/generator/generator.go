package generator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/user/gogo/internal/blueprints"
	"github.com/user/gogo/internal/cicd"
	"github.com/user/gogo/internal/git"
	"github.com/user/gogo/internal/templates"
	"github.com/user/gogo/internal/validate"
)

// InitOptions contains options for project initialization
type InitOptions struct {
	ProjectName          string
	ModuleName           string
	Template             string
	Blueprint            string // Blueprint name for enhanced stack support
	Author               string
	Email                string // Author email for git configuration
	License              string
	GoVersion            string
	OutputDir            string
	Description          string
	GitInit              bool
	GenerateCI           bool    // Generate CI/CD configurations
	CoverageMin          float64 // Minimum test coverage percentage
	InitialCommitMessage string  // Custom initial commit message
	Force                bool
	DryRun               bool
}

// Result contains the result of a generation operation
type Result struct {
	Success      bool
	ProjectPath  string
	FilesCreated int
	Message      string
}

// ProjectGenerator interface for generating projects
type ProjectGenerator interface {
	InitProject(ctx context.Context, opts InitOptions) (Result, error)
}

// Generator implements ProjectGenerator
type Generator struct {
	templateEngine      templates.TemplateRenderer
	templateRepository  *templates.Repository
	blueprintRepository *blueprints.Repository
	blueprintResolver   blueprints.BlueprintResolver
}

// NewProjectGenerator creates a new project generator
func NewProjectGenerator(engine templates.TemplateRenderer, repo *templates.Repository) *Generator {
	return &Generator{
		templateEngine:      engine,
		templateRepository:  repo,
		blueprintRepository: blueprints.NewRepository(),
		blueprintResolver:   blueprints.NewResolver(),
	}
}

// InitProject initializes a new Go project
func (g *Generator) InitProject(ctx context.Context, opts InitOptions) (Result, error) {
	// Validate options
	if err := g.validateOptions(opts); err != nil {
		return Result{}, fmt.Errorf("invalid options: %w", err)
	}

	// Set defaults
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if opts.GoVersion == "" {
		opts.GoVersion = "1.25.1"
	}
	if opts.License == "" {
		opts.License = "MIT"
	}
	if opts.Description == "" {
		opts.Description = fmt.Sprintf("A %s project", opts.Template)
	}

	// Prepare base template variables
	variables := map[string]any{
		"ProjectName": opts.ProjectName,
		"ModuleName":  opts.ModuleName,
		"Author":      opts.Author,
		"License":     opts.License,
		"GoVersion":   opts.GoVersion,
		"Description": opts.Description,
	}

	var templateFiles []templates.TemplateFile

	// Use blueprint if specified
	if opts.Blueprint != "" {
		blueprint, err := g.blueprintRepository.GetBlueprint(ctx, opts.Blueprint)
		if err != nil {
			return Result{}, fmt.Errorf("failed to get blueprint: %w", err)
		}

		// Resolve blueprint variables
		resolvedVars, err := g.blueprintResolver.Resolve(ctx, blueprint, variables)
		if err != nil {
			return Result{}, fmt.Errorf("failed to resolve blueprint variables: %w", err)
		}
		variables = resolvedVars

		// Get blueprint-specific template files
		blueprintTemplates := templates.GetBlueprintTemplates()
		if stackTemplates, exists := blueprintTemplates[blueprint.Stack]; exists {
			// Convert BlueprintTemplateFile to TemplateFile
			templateFiles = make([]templates.TemplateFile, len(stackTemplates))
			for i, bt := range stackTemplates {
				templateFiles[i] = templates.TemplateFile{
					Name:    bt.Name,
					Path:    bt.Path,
					Content: bt.Content,
				}
			}
		} else {
			// Fallback to regular template files
			files, err := g.templateRepository.GetTemplateFiles(ctx, opts.Template)
			if err != nil {
				return Result{}, fmt.Errorf("failed to get template files: %w", err)
			}
			templateFiles = files
		}
	} else {
		// Get regular template files
		files, err := g.templateRepository.GetTemplateFiles(ctx, opts.Template)
		if err != nil {
			return Result{}, fmt.Errorf("failed to get template files: %w", err)
		}
		templateFiles = files
	}

	result := Result{
		ProjectPath:  opts.OutputDir,
		FilesCreated: len(templateFiles),
		Success:      true,
	}

	// Dry run - just validate and return
	if opts.DryRun {
		result.Message = fmt.Sprintf("Would create %d files in %s", len(templateFiles), opts.OutputDir)
		return result, nil
	}

	// Render and write each template file
	for _, templateFile := range templateFiles {
		// Render the file path template
		renderedPath, err := g.templateEngine.RenderString(ctx, templateFile.Path, variables)
		if err != nil {
			return Result{}, fmt.Errorf("failed to render path template for %s: %w", templateFile.Name, err)
		}

		outputPath := filepath.Join(opts.OutputDir, renderedPath)

		// Render the file content
		err = g.templateEngine.RenderToFile(ctx, templateFile.Content, variables, outputPath)
		if err != nil {
			return Result{}, fmt.Errorf("failed to render file %s: %w", templateFile.Name, err)
		}
	}

	// Generate CI/CD configurations if requested
	if opts.GenerateCI {
		if err := g.generateCICD(ctx, opts, variables); err != nil {
			return Result{}, fmt.Errorf("failed to generate CI/CD configurations: %w", err)
		}
		result.FilesCreated += 3 // .golangci.yml, ci.yml, .pre-commit-config.yaml
	}

	// Initialize git repository if requested
	if opts.GitInit {
		if err := g.initializeGit(ctx, opts); err != nil {
			return Result{}, fmt.Errorf("failed to initialize git repository: %w", err)
		}
	}

	result.Message = g.buildResultMessage(opts, len(templateFiles))
	return result, nil
}

// validateOptions validates the initialization options
func (g *Generator) validateOptions(opts InitOptions) error {
	if opts.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}

	if opts.ModuleName == "" {
		return fmt.Errorf("module name is required")
	}

	if opts.Template == "" {
		return fmt.Errorf("template is required")
	}

	// Validate project name format
	if err := validate.ValidateProjectName(opts.ProjectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	// Validate module name format
	if err := validate.ValidateModuleName(opts.ModuleName); err != nil {
		return fmt.Errorf("invalid module name: %w", err)
	}

	// Validate Go version if provided
	if opts.GoVersion != "" {
		if err := validate.ValidateGoVersion(opts.GoVersion); err != nil {
			return fmt.Errorf("invalid Go version: %w", err)
		}
	}

	// Validate output directory if provided
	if opts.OutputDir != "" {
		// For validation, we only check if the parent directory exists and is writable
		// The actual output directory will be created during generation
		parentDir := filepath.Dir(opts.OutputDir)
		if parentDir != "." && parentDir != opts.OutputDir {
			if err := validate.ValidateOutputDir(parentDir); err != nil {
				return fmt.Errorf("invalid output directory: %w", err)
			}
		}
	}

	return nil
}

// generateCICD generates CI/CD configuration files
func (g *Generator) generateCICD(ctx context.Context, opts InitOptions, variables map[string]any) error {
	// Set defaults for CI/CD generation
	generateCI := opts.GenerateCI
	if !generateCI && opts.GitInit {
		// If GitInit is enabled, auto-enable CI generation
		generateCI = true
	}

	if !generateCI {
		return nil
	}

	// Determine if project has database based on blueprint
	hasDatabase := false
	databaseType := ""
	if opts.Blueprint != "" {
		blueprint, err := g.blueprintRepository.GetBlueprint(ctx, opts.Blueprint)
		if err == nil {
			if len(blueprint.Config.Database) > 0 {
				hasDatabase = true
				if dbType, ok := blueprint.Config.Database["type"].(string); ok {
					databaseType = dbType
				}
			}
		}
	}

	// Set default coverage minimum
	coverageMin := opts.CoverageMin
	if coverageMin == 0 {
		coverageMin = 0.80 // Default to 80%
	}

	// Create CI/CD configuration
	cicdConfig := cicd.Config{
		ProjectName:   opts.ProjectName,
		GoVersion:     opts.GoVersion,
		CoverageMin:   coverageMin,
		TestFramework: "testify", // Default framework
		HasDatabase:   hasDatabase,
		DatabaseType:  databaseType,
		HasDocker:     false, // TODO: Determine from blueprint in future
		LintTimeout:   "5m",
		BuildTargets:  []string{"linux", "darwin", "windows"},
	}

	// Generate CI/CD files
	cicdGenerator := cicd.NewGenerator()
	return cicdGenerator.GenerateAll(ctx, opts.OutputDir, cicdConfig)
}

// initializeGit initializes a git repository with initial commit
func (g *Generator) initializeGit(ctx context.Context, opts InitOptions) error {
	if !git.IsGitInstalled() {
		return fmt.Errorf("git is not installed or not available in PATH")
	}

	gitManager := git.NewGitManager(opts.OutputDir)

	// Validate working directory
	if err := gitManager.ValidateWorkingDir(); err != nil {
		return fmt.Errorf("git working directory validation failed: %w", err)
	}

	// Get email if not provided
	email := opts.Email
	if email == "" {
		// Try to get email from git config
		_, gitEmail := git.GetUserInfo(ctx)
		if gitEmail != "" {
			email = gitEmail
		}
	}

	// Initialize git repository
	gitOpts := git.InitOptions{
		ProjectName:          opts.ProjectName,
		Author:               opts.Author,
		Email:                email,
		InitialCommitMessage: opts.InitialCommitMessage,
	}

	if err := gitManager.Init(ctx, gitOpts); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Create initial commit
	if err := gitManager.InitialCommit(ctx, gitOpts); err != nil {
		return fmt.Errorf("initial commit failed: %w", err)
	}

	return nil
}

// buildResultMessage builds the result message based on what was generated
func (g *Generator) buildResultMessage(opts InitOptions, templateFilesCount int) string {
	message := fmt.Sprintf("Created %d files in %s", templateFilesCount, opts.OutputDir)

	if opts.GenerateCI {
		message += "\nGenerated CI/CD configurations (.golangci.yml, GitHub Actions, pre-commit hooks)"
	}

	if opts.GitInit {
		message += "\nInitialized git repository with initial commit"
	}

	return message
}
