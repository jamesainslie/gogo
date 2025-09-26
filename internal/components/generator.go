package components

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/gogo/internal/templates"
	"github.com/user/gogo/internal/validate"
)

// GenerateOptions contains options for component generation
type GenerateOptions struct {
	Type        string // handler, model, service, migration, middleware, test
	Name        string
	OutputDir   string
	ProjectName string
	ModuleName  string
	Framework   string // gin, echo, chi
	Database    string // gorm, sqlx, pgx
	DryRun      bool
	Force       bool
}

// GenerateResult contains the result of a component generation
type GenerateResult struct {
	Success      bool
	FilesCreated int
	Message      string
	Files        []string
}

// ComponentGenerator interface for generating components
type ComponentGenerator interface {
	Generate(ctx context.Context, opts GenerateOptions) (GenerateResult, error)
	GetSupportedTypes() []string
}

// Generator implements ComponentGenerator
type Generator struct {
	templateEngine templates.TemplateRenderer
}

// NewGenerator creates a new component generator
func NewGenerator() *Generator {
	return &Generator{
		templateEngine: templates.NewEngine(),
	}
}

// Generate generates a component based on the options
func (g *Generator) Generate(ctx context.Context, opts GenerateOptions) (GenerateResult, error) {
	// Validate options
	if err := g.validateOptions(opts); err != nil {
		return GenerateResult{}, fmt.Errorf("invalid options: %w", err)
	}

	// Set defaults
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if opts.Framework == "" {
		opts.Framework = "gin"
	}
	if opts.Database == "" {
		opts.Database = "gorm"
	}

	// Get component templates
	componentTemplates, err := g.getComponentTemplates(opts.Type)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("failed to get component templates: %w", err)
	}

	// Prepare template variables
	variables := g.prepareVariables(opts)

	result := GenerateResult{
		Success:      true,
		FilesCreated: len(componentTemplates),
		Files:        make([]string, len(componentTemplates)),
	}

	// Dry run - just validate and return
	if opts.DryRun {
		for i, template := range componentTemplates {
			renderedPath, err := g.templateEngine.RenderString(ctx, template.Path, variables)
			if err != nil {
				return GenerateResult{}, fmt.Errorf("failed to render path template: %w", err)
			}
			result.Files[i] = renderedPath
		}
		result.Message = fmt.Sprintf("Would create %d files", len(componentTemplates))
		return result, nil
	}

	// Generate each component file
	for i, template := range componentTemplates {
		// Render the file path
		renderedPath, err := g.templateEngine.RenderString(ctx, template.Path, variables)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("failed to render path template: %w", err)
		}

		outputPath := filepath.Join(opts.OutputDir, renderedPath)
		result.Files[i] = renderedPath

		// Render and write the file
		err = g.templateEngine.RenderToFile(ctx, template.Content, variables, outputPath)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("failed to render component file %s: %w", template.Name, err)
		}
	}

	result.Message = fmt.Sprintf("Created %d files", len(componentTemplates))
	return result, nil
}

// GetSupportedTypes returns the list of supported component types
func (g *Generator) GetSupportedTypes() []string {
	return []string{
		"handler",
		"model", 
		"service",
		"migration",
		"middleware",
		"test",
	}
}

// validateOptions validates the generation options
func (g *Generator) validateOptions(opts GenerateOptions) error {
	if opts.Type == "" {
		return fmt.Errorf("component type is required")
	}

	if opts.Name == "" {
		return fmt.Errorf("component name is required")
	}

	// Validate component type
	supportedTypes := g.GetSupportedTypes()
	validType := false
	for _, t := range supportedTypes {
		if t == opts.Type {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("unsupported component type '%s', supported types: %s", opts.Type, strings.Join(supportedTypes, ", "))
	}

	// Validate component name
	if err := validate.ValidateProjectName(opts.Name); err != nil {
		return fmt.Errorf("invalid component name: %w", err)
	}

	return nil
}

// prepareVariables prepares template variables for rendering
func (g *Generator) prepareVariables(opts GenerateOptions) map[string]any {
	// Convert name to different cases
	name := opts.Name
	titleName := toTitleCase(name)
	camelName := toCamelCase(name)
	snakeName := toSnakeCase(name)
	kebabName := toKebabCase(name)

	variables := map[string]any{
		"Name":        name,
		"TitleName":   titleName,
		"CamelName":   camelName,
		"SnakeName":   snakeName,
		"KebabName":   kebabName,
		"ProjectName": opts.ProjectName,
		"ModuleName":  opts.ModuleName,
		"Framework":   opts.Framework,
		"Database":    opts.Database,
		"Timestamp":   time.Now().Format("20060102150405"),
		"Year":        time.Now().Year(),
	}

	// Add framework-specific variables
	variables["IsGin"] = opts.Framework == "gin"
	variables["IsEcho"] = opts.Framework == "echo"
	variables["IsChi"] = opts.Framework == "chi"

	// Add database-specific variables
	variables["IsGorm"] = opts.Database == "gorm"
	variables["IsSqlx"] = opts.Database == "sqlx"
	variables["IsPgx"] = opts.Database == "pgx"

	return variables
}

// getComponentTemplates returns templates for a specific component type
func (g *Generator) getComponentTemplates(componentType string) ([]ComponentTemplate, error) {
	templates := getComponentTemplates()
	
	if componentTemplates, exists := templates[componentType]; exists {
		return componentTemplates, nil
	}
	
	return nil, fmt.Errorf("no templates found for component type '%s'", componentType)
}

// Helper functions for name conversion
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	// Simple title case: capitalize first letter of each word
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "")
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		parts = strings.Split(s, "-")
	}
	
	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return result
}

func toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "-", "_"), " ", "_"))
}

func toKebabCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, "_", "-"), " ", "-"))
}
