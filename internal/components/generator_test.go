package components

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentGenerator_GenerateHandler(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewGenerator()
	ctx := context.Background()

	tests := []struct {
		name        string
		opts        GenerateOptions
		expectFiles []string
		expectError bool
	}{
		{
			name: "generate REST handler",
			opts: GenerateOptions{
				Type:        "handler",
				Name:        "user",
				OutputDir:   tempDir,
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
				Framework:   "gin",
			},
			expectFiles: []string{
				"internal/handlers/user_handler.go",
				"internal/handlers/user_handler_test.go",
			},
			expectError: false,
		},
		{
			name: "generate model",
			opts: GenerateOptions{
				Type:        "model",
				Name:        "user",
				OutputDir:   tempDir,
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
				Database:    "gorm",
			},
			expectFiles: []string{
				"internal/models/user.go",
				"internal/models/user_test.go",
			},
			expectError: false,
		},
		{
			name: "generate service",
			opts: GenerateOptions{
				Type:        "service",
				Name:        "user",
				OutputDir:   tempDir,
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
			},
			expectFiles: []string{
				"internal/services/user_service.go",
				"internal/services/user_service_test.go",
			},
			expectError: false,
		},
		{
			name: "generate migration",
			opts: GenerateOptions{
				Type:        "migration",
				Name:        "create_users_table",
				OutputDir:   tempDir,
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
				Database:    "gorm",
			},
			expectFiles: []string{
				"migrations/001_create_users_table.sql",
			},
			expectError: false,
		},
		{
			name: "invalid component type",
			opts: GenerateOptions{
				Type:      "invalid",
				Name:      "test",
				OutputDir: tempDir,
			},
			expectFiles: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.Generate(ctx, tt.opts)
			
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.GreaterOrEqual(t, result.FilesCreated, len(tt.expectFiles))
			
			// Verify expected files exist and have content
			for _, expectedFile := range tt.expectFiles {
				filePath := filepath.Join(tt.opts.OutputDir, expectedFile)
				_, err := os.Stat(filePath)
				assert.NoError(t, err, "file %s should exist", expectedFile)
				
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.NotEmpty(t, content, "file %s should not be empty", expectedFile)
				
				contentStr := string(content)
				// Verify basic structure (migrations don't have package declarations)
				if tt.opts.Type != "migration" {
					assert.Contains(t, contentStr, "package", "file should have package declaration")
				}
				
				// Verify name substitution
				if tt.opts.Name != "" {
					assert.Contains(t, contentStr, tt.opts.Name, "file should contain component name")
				}
			}
		})
	}
}

func TestComponentGenerator_ValidateOptions(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name    string
		opts    GenerateOptions
		wantErr bool
	}{
		{
			name: "valid handler options",
			opts: GenerateOptions{
				Type:      "handler",
				Name:      "user",
				OutputDir: "/tmp",
			},
			wantErr: false,
		},
		{
			name: "valid model options",
			opts: GenerateOptions{
				Type:      "model",
				Name:      "user",
				OutputDir: "/tmp",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			opts: GenerateOptions{
				Name:      "user",
				OutputDir: "/tmp",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			opts: GenerateOptions{
				Type:      "handler",
				OutputDir: "/tmp",
			},
			wantErr: true,
		},
		{
			name: "invalid component type",
			opts: GenerateOptions{
				Type:      "invalid",
				Name:      "user",
				OutputDir: "/tmp",
			},
			wantErr: true,
		},
		{
			name: "empty output dir defaults to current",
			opts: GenerateOptions{
				Type: "handler",
				Name: "user",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.validateOptions(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComponentGenerator_GetSupportedTypes(t *testing.T) {
	generator := NewGenerator()
	
	types := generator.GetSupportedTypes()
	
	// Should have at least these core types
	expectedTypes := []string{"handler", "model", "service", "migration", "middleware", "test"}
	
	assert.GreaterOrEqual(t, len(types), len(expectedTypes))
	
	for _, expectedType := range expectedTypes {
		assert.Contains(t, types, expectedType, "should support %s component type", expectedType)
	}
}

func TestComponentGenerator_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewGenerator()
	ctx := context.Background()

	opts := GenerateOptions{
		Type:      "handler",
		Name:      "test",
		OutputDir: tempDir,
		DryRun:    true,
	}

	result, err := generator.Generate(ctx, opts)
	require.NoError(t, err)
	
	// Should return success but not create files
	assert.True(t, result.Success)
	assert.Greater(t, result.FilesCreated, 0) // Should report files that would be created
	
	// Verify no files were actually created
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "no files should be created in dry run")
}
