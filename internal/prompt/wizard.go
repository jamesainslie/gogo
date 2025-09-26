package prompt

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/user/gogo/internal/blueprints"
	"github.com/user/gogo/internal/generator"
	"github.com/user/gogo/internal/templates"
	"github.com/user/gogo/internal/validate"
)

// WizardOptions contains the results from the interactive wizard
type WizardOptions struct {
	ProjectName          string
	ModuleName           string
	Template             string
	Blueprint            string
	Author               string
	Email                string
	License              string
	GoVersion            string
	OutputDir            string
	GitInit              bool
	GenerateCI           bool
	CoverageMin          float64
	InitialCommitMessage string
	Force                bool
}

// Wizard provides interactive prompts for project initialization
type Wizard struct {
	templateRepo  *templates.Repository
	blueprintRepo *blueprints.Repository
}

// NewWizard creates a new wizard instance
func NewWizard() *Wizard {
	return &Wizard{
		templateRepo:  templates.NewRepository(),
		blueprintRepo: blueprints.NewRepository(),
	}
}

// RunInitWizard runs the interactive wizard for project initialization
func (w *Wizard) RunInitWizard(ctx context.Context, initialOptions generator.InitOptions) (*WizardOptions, error) {
	color.Cyan("Welcome to gogo project initialization wizard!")
	fmt.Println()

	options := &WizardOptions{
		ProjectName: initialOptions.ProjectName,
		ModuleName:  initialOptions.ModuleName,
		Template:    initialOptions.Template,
		Blueprint:   initialOptions.Blueprint,
		Author:      initialOptions.Author,
		License:     initialOptions.License,
		GoVersion:   initialOptions.GoVersion,
		OutputDir:   initialOptions.OutputDir,
		GitInit:     initialOptions.GitInit,
		Force:       initialOptions.Force,
	}

	// Project name
	if options.ProjectName == "" {
		if err := w.promptProjectName(options); err != nil {
			return nil, err
		}
	}

	// Module name
	if options.ModuleName == "" {
		if err := w.promptModuleName(options); err != nil {
			return nil, err
		}
	}

	// Template selection
	if options.Template == "" || options.Template == "cli" {
		if err := w.promptTemplate(ctx, options); err != nil {
			return nil, err
		}
	}

	// Blueprint selection (optional, based on template)
	if w.shouldPromptBlueprint(options.Template) {
		if err := w.promptBlueprint(ctx, options); err != nil {
			return nil, err
		}
	}

	// Author name
	if options.Author == "" {
		if err := w.promptAuthor(options); err != nil {
			return nil, err
		}
	}

	// Author email (optional)
	if err := w.promptEmail(options); err != nil {
		return nil, err
	}

	// License
	if options.License == "" || options.License == "MIT" {
		if err := w.promptLicense(options); err != nil {
			return nil, err
		}
	}

	// Go version
	if options.GoVersion == "" {
		if err := w.promptGoVersion(options); err != nil {
			return nil, err
		}
	}

	// Output directory
	if options.OutputDir == "" || options.OutputDir == "." {
		if err := w.promptOutputDir(options); err != nil {
			return nil, err
		}
	}

	// Git initialization
	if err := w.promptGitInit(options); err != nil {
		return nil, err
	}

	// CI/CD configuration (if git is enabled)
	if options.GitInit {
		if err := w.promptCICD(options); err != nil {
			return nil, err
		}

		// Coverage minimum (if CI/CD is enabled)
		if options.GenerateCI {
			if err := w.promptCoverageMin(options); err != nil {
				return nil, err
			}
		}
	}

	// Force overwrite
	if err := w.promptForce(options); err != nil {
		return nil, err
	}

	// Summary
	w.showSummary(options)

	// Confirmation
	if err := w.promptConfirmation(); err != nil {
		return nil, err
	}

	return options, nil
}

func (w *Wizard) promptProjectName(options *WizardOptions) error {
	validate := func(input string) error {
		if input == "" {
			return fmt.Errorf("project name cannot be empty")
		}
		return validate.ValidateProjectName(input)
	}

	prompt := promptui.Prompt{
		Label:    "Project name",
		Validate: validate,
	}

	result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("project name prompt failed: %w", err)
	}

	options.ProjectName = result
	return nil
}

func (w *Wizard) promptModuleName(options *WizardOptions) error {
	// Suggest a module name based on project name
	var defaultModule string
	if options.ProjectName != "" {
		defaultModule = fmt.Sprintf("github.com/user/%s", options.ProjectName)
	}

	validate := func(input string) error {
		if input == "" {
			return fmt.Errorf("module name cannot be empty")
		}
		return validate.ValidateModuleName(input)
	}

	prompt := promptui.Prompt{
		Label:    fmt.Sprintf("Go module name (e.g., %s)", defaultModule),
		Validate: validate,
		Default:  defaultModule,
	}

	result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("module name prompt failed: %w", err)
	}

	options.ModuleName = result
	return nil
}

func (w *Wizard) promptTemplate(ctx context.Context, options *WizardOptions) error {
	templates, err := w.templateRepo.ListPredefinedTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	items := make([]string, len(templates))
	for i, tmpl := range templates {
		items[i] = fmt.Sprintf("%s - %s", tmpl.Name, tmpl.Kind)
	}

	prompt := promptui.Select{
		Label: "Select project template",
		Items: items,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("template selection failed: %w", err)
	}

	options.Template = templates[i].Kind
	return nil
}

func (w *Wizard) shouldPromptBlueprint(template string) bool {
	// Only prompt for blueprints for certain templates that benefit from stacks
	switch template {
	case "api", "grpc", "microservice":
		return true
	default:
		return false
	}
}

func (w *Wizard) promptBlueprint(ctx context.Context, options *WizardOptions) error {
	availableBlueprints, err := w.blueprintRepo.ListBlueprints(ctx)
	if err != nil {
		return fmt.Errorf("failed to list blueprints: %w", err)
	}
	if len(availableBlueprints) == 0 {
		// Skip if no blueprints available
		return nil
	}

	// Filter blueprints suitable for the selected template
	var suitableBlueprints []blueprints.Blueprint
	for _, bp := range availableBlueprints {
		if w.isBlueprintSuitableForTemplate(bp, options.Template) {
			suitableBlueprints = append(suitableBlueprints, bp)
		}
	}

	if len(suitableBlueprints) == 0 {
		return nil
	}

	// Add "None" option
	items := []string{"None (basic template only)"}
	for _, bp := range suitableBlueprints {
		items = append(items, fmt.Sprintf("%s - %s stack", bp.Name, bp.Stack))
	}

	prompt := promptui.Select{
		Label: "Select stack blueprint (optional)",
		Items: items,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("blueprint selection failed: %w", err)
	}

	if i > 0 {
		options.Blueprint = suitableBlueprints[i-1].Name
	}
	return nil
}

func (w *Wizard) isBlueprintSuitableForTemplate(bp blueprints.Blueprint, template string) bool {
	switch template {
	case "api":
		return bp.Stack == "web"
	case "grpc":
		return bp.Stack == "grpc"
	case "microservice":
		return bp.Stack == "microservice" || bp.Stack == "web"
	default:
		return false
	}
}

func (w *Wizard) promptAuthor(options *WizardOptions) error {
	// Try to get default from git config
	defaultAuthor := w.getGitUserName()

	prompt := promptui.Prompt{
		Label:   "Author name",
		Default: defaultAuthor,
	}

	result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("author prompt failed: %w", err)
	}

	options.Author = result
	return nil
}

func (w *Wizard) promptLicense(options *WizardOptions) error {
	licenses := []string{"MIT", "Apache-2.0", "GPL-3.0", "BSD-3-Clause", "ISC", "Other"}

	prompt := promptui.Select{
		Label: "Select license",
		Items: licenses,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("license selection failed: %w", err)
	}

	options.License = result
	return nil
}

func (w *Wizard) promptGoVersion(options *WizardOptions) error {
	versions := []string{"1.25.1", "1.25", "auto-detect", "1.24", "1.23"}

	prompt := promptui.Select{
		Label: "Select Go version",
		Items: versions,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("go version selection failed: %w", err)
	}

	if result == "auto-detect" {
		options.GoVersion = ""
	} else {
		options.GoVersion = result
	}
	return nil
}

func (w *Wizard) promptOutputDir(options *WizardOptions) error {
	defaultDir := options.ProjectName
	if defaultDir == "" {
		defaultDir = "."
	}

	prompt := promptui.Prompt{
		Label:   "Output directory",
		Default: defaultDir,
	}

	result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("output directory prompt failed: %w", err)
	}

	options.OutputDir = result
	return nil
}

func (w *Wizard) promptGitInit(options *WizardOptions) error {
	prompt := promptui.Select{
		Label: "Initialize git repository",
		Items: []string{"Yes", "No"},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("git init prompt failed: %w", err)
	}

	options.GitInit = i == 0 // 0 = "Yes", 1 = "No"
	return nil
}

func (w *Wizard) promptForce(options *WizardOptions) error {
	// Check if output directory exists and is not empty
	if options.OutputDir != "." {
		if _, err := os.Stat(options.OutputDir); err == nil {
			// Directory exists, check if it's empty
			entries, err := os.ReadDir(options.OutputDir)
			if err != nil {
				return fmt.Errorf("failed to read output directory: %w", err)
			}

			if len(entries) > 0 {
				prompt := promptui.Select{
					Label: fmt.Sprintf("Directory '%s' is not empty. Overwrite existing files?", options.OutputDir),
					Items: []string{"No", "Yes"},
				}

				i, _, err := prompt.Run()
				if err != nil {
					return fmt.Errorf("force overwrite prompt failed: %w", err)
				}

				options.Force = i == 1 // 0 = "No", 1 = "Yes"
			}
		}
	}

	return nil
}

func (w *Wizard) showSummary(options *WizardOptions) {
	fmt.Println()
	color.Yellow("Project Configuration Summary:")
	fmt.Printf("  Project Name: %s\n", options.ProjectName)
	fmt.Printf("  Module Name:  %s\n", options.ModuleName)
	fmt.Printf("  Template:     %s\n", options.Template)
	if options.Blueprint != "" {
		fmt.Printf("  Blueprint:    %s\n", options.Blueprint)
	}
	fmt.Printf("  Author:       %s\n", options.Author)
	if options.Email != "" {
		fmt.Printf("  Email:        %s\n", options.Email)
	}
	fmt.Printf("  License:      %s\n", options.License)
	fmt.Printf("  Go Version:   %s\n", w.displayGoVersion(options.GoVersion))
	fmt.Printf("  Output Dir:   %s\n", options.OutputDir)
	fmt.Printf("  Git Init:     %t\n", options.GitInit)
	if options.GitInit && options.GenerateCI {
		fmt.Printf("  Generate CI:  %t\n", options.GenerateCI)
		if options.CoverageMin > 0 {
			fmt.Printf("  Coverage Min: %.0f%%\n", options.CoverageMin*100)
		}
	}
	if options.Force {
		fmt.Printf("  Force:        %t\n", options.Force)
	}
	fmt.Println()
}

func (w *Wizard) displayGoVersion(version string) string {
	if version == "" {
		return "auto-detect"
	}
	return version
}

func (w *Wizard) promptConfirmation() error {
	prompt := promptui.Select{
		Label: "Proceed with project creation",
		Items: []string{"Yes", "No"},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("confirmation prompt failed: %w", err)
	}

	if i != 0 { // 0 = "Yes", 1 = "No"
		return fmt.Errorf("project creation cancelled by user")
	}

	return nil
}

func (w *Wizard) getGitUserName() string {
	// Try to get git user name from git config
	// This is a simple implementation - in a real scenario you might want to exec git config
	if name := os.Getenv("GIT_AUTHOR_NAME"); name != "" {
		return name
	}
	if name := os.Getenv("USER"); name != "" {
		return name
	}
	return ""
}

func (w *Wizard) promptEmail(options *WizardOptions) error {
	// Try to get default from git config
	defaultEmail := w.getGitUserEmail()

	prompt := promptui.Prompt{
		Label:   "Author email (optional)",
		Default: defaultEmail,
	}

	result, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("email prompt failed: %w", err)
	}

	options.Email = result
	return nil
}

func (w *Wizard) promptCICD(options *WizardOptions) error {
	prompt := promptui.Select{
		Label: "Generate CI/CD configurations (.golangci.yml, GitHub Actions, pre-commit hooks)?",
		Items: []string{"Yes", "No"},
	}

	i, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("CI/CD prompt failed: %w", err)
	}

	options.GenerateCI = i == 0 // 0 = "Yes", 1 = "No"
	return nil
}

func (w *Wizard) promptCoverageMin(options *WizardOptions) error {
	coverageOptions := []string{"80%", "75%", "85%", "90%", "Custom"}

	prompt := promptui.Select{
		Label: "Minimum test coverage percentage",
		Items: coverageOptions,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return fmt.Errorf("coverage prompt failed: %w", err)
	}

	switch i {
	case 0: // 80%
		options.CoverageMin = 0.80
	case 1: // 75%
		options.CoverageMin = 0.75
	case 2: // 85%
		options.CoverageMin = 0.85
	case 3: // 90%
		options.CoverageMin = 0.90
	case 4: // Custom
		customPrompt := promptui.Prompt{
			Label:    "Enter coverage percentage (0-100)",
			Default:  "80",
			Validate: w.validateCoveragePercentage,
		}

		customResult, err := customPrompt.Run()
		if err != nil {
			return fmt.Errorf("custom coverage prompt failed: %w", err)
		}

		// Parse percentage
		var percentage float64
		if _, err := fmt.Sscanf(customResult, "%f", &percentage); err != nil {
			return fmt.Errorf("invalid coverage percentage: %w", err)
		}
		options.CoverageMin = percentage / 100.0
	}

	return nil
}

func (w *Wizard) validateCoveragePercentage(input string) error {
	var percentage float64
	if _, err := fmt.Sscanf(input, "%f", &percentage); err != nil {
		return fmt.Errorf("must be a number")
	}
	if percentage < 0 || percentage > 100 {
		return fmt.Errorf("must be between 0 and 100")
	}
	return nil
}

func (w *Wizard) getGitUserEmail() string {
	// Try to get git user email from environment or git config
	if email := os.Getenv("GIT_AUTHOR_EMAIL"); email != "" {
		return email
	}
	return ""
}

// ConvertToInitOptions converts wizard options to generator InitOptions
func (w *WizardOptions) ConvertToInitOptions() generator.InitOptions {
	return generator.InitOptions{
		ProjectName:          w.ProjectName,
		ModuleName:           w.ModuleName,
		Template:             w.Template,
		Blueprint:            w.Blueprint,
		Author:               w.Author,
		Email:                w.Email,
		License:              w.License,
		GoVersion:            w.GoVersion,
		OutputDir:            w.OutputDir,
		Description:          fmt.Sprintf("A %s project", w.Template),
		GitInit:              w.GitInit,
		GenerateCI:           w.GenerateCI,
		CoverageMin:          w.CoverageMin,
		InitialCommitMessage: w.InitialCommitMessage,
		Force:                w.Force,
		DryRun:               false, // Wizard doesn't support dry-run mode
	}
}
