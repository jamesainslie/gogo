package templates

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flosch/pongo2/v6"
)

// Template represents a template with metadata
type Template struct {
	ID          int
	Name        string
	Kind        string
	Content     string
	MetadataJSON string
}

// TemplateRenderer interface for rendering templates
type TemplateRenderer interface {
	RenderString(ctx context.Context, template string, variables map[string]any) (string, error)
	RenderToFile(ctx context.Context, template string, variables map[string]any, outputPath string) error
	RenderTemplate(ctx context.Context, template Template, variables map[string]any, outputPath string) error
}

// Engine implements the TemplateRenderer interface using pongo2
type Engine struct{}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	return &Engine{}
}

// RenderString renders a template string with variables
func (e *Engine) RenderString(ctx context.Context, template string, variables map[string]any) (string, error) {
	tpl, err := pongo2.FromString(template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	result, err := tpl.Execute(variables)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result, nil
}

// RenderToFile renders a template string to a file
func (e *Engine) RenderToFile(ctx context.Context, template string, variables map[string]any, outputPath string) error {
	result, err := e.RenderString(ctx, template, variables)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

// RenderTemplate renders a Template struct to a file
func (e *Engine) RenderTemplate(ctx context.Context, template Template, variables map[string]any, outputPath string) error {
	return e.RenderToFile(ctx, template.Content, variables, outputPath)
}
