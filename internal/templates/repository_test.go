package templates

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetPredefinedTemplate(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	tests := []struct {
		name         string
		templateKind string
		expectFound  bool
		expectFields []string // Fields that should exist in the template
	}{
		{
			name:         "CLI template",
			templateKind: "cli",
			expectFound:  true,
			expectFields: []string{"ProjectName", "ModuleName", "Author"},
		},
		{
			name:         "library template",
			templateKind: "library",
			expectFound:  true,
			expectFields: []string{"ProjectName", "ModuleName", "Author"},
		},
		{
			name:         "API template",
			templateKind: "api",
			expectFound:  true,
			expectFields: []string{"ProjectName", "ModuleName", "Author"},
		},
		{
			name:         "gRPC template",
			templateKind: "grpc",
			expectFound:  true,
			expectFields: []string{"ProjectName", "ModuleName", "Author"},
		},
		{
			name:         "microservice template",
			templateKind: "microservice",
			expectFound:  true,
			expectFields: []string{"ProjectName", "ModuleName", "Author"},
		},
		{
			name:         "non-existent template",
			templateKind: "nonexistent",
			expectFound:  false,
			expectFields: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := repo.GetPredefinedTemplate(ctx, tt.templateKind)
			
			if tt.expectFound {
				require.NoError(t, err)
				assert.Equal(t, tt.templateKind, template.Kind)
				assert.NotEmpty(t, template.Name)
				assert.NotEmpty(t, template.Content)
				
				// Verify template contains expected variable placeholders
				for _, field := range tt.expectFields {
					assert.Contains(t, template.Content, "{{ "+field+" }}", 
						"template should contain variable %s", field)
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRepository_ListPredefinedTemplates(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	templates, err := repo.ListPredefinedTemplates(ctx)
	require.NoError(t, err)

	// Should have all 5 predefined templates
	assert.Len(t, templates, 5)

	// Verify all expected kinds are present
	expectedKinds := []string{"cli", "library", "api", "grpc", "microservice"}
	actualKinds := make([]string, len(templates))
	for i, tmpl := range templates {
		actualKinds[i] = tmpl.Kind
	}

	for _, expectedKind := range expectedKinds {
		assert.Contains(t, actualKinds, expectedKind)
	}
}

func TestRepository_GetTemplateFiles(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	tests := []struct {
		name         string
		templateKind string
		expectFiles  []string
	}{
		{
			name:         "CLI template files",
			templateKind: "cli",
			expectFiles:  []string{"main.go", "go.mod", "README.md", ".gitignore", "Makefile"},
		},
		{
			name:         "library template files",
			templateKind: "library",
			expectFiles:  []string{"lib.go", "go.mod", "README.md", ".gitignore"},
		},
		{
			name:         "API template files",
			templateKind: "api",
			expectFiles:  []string{"main.go", "go.mod", "README.md", ".gitignore", "Makefile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := repo.GetTemplateFiles(ctx, tt.templateKind)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(files), len(tt.expectFiles))
			
			// Check that expected files are present
			fileNames := make([]string, len(files))
			for i, file := range files {
				fileNames[i] = file.Name
			}
			
			for _, expectedFile := range tt.expectFiles {
				assert.Contains(t, fileNames, expectedFile)
			}
		})
	}
}
