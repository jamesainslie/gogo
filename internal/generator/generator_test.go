package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/gogo/internal/templates"
)

func TestProjectGenerator_InitProject(t *testing.T) {
	tempDir := t.TempDir()

	engine := templates.NewEngine()
	repo := templates.NewRepository()
	generator := NewProjectGenerator(engine, repo)
	ctx := context.Background()

	tests := []struct {
		name        string
		opts        InitOptions
		expectFiles []string
		expectError bool
	}{
		{
			name: "CLI project initialization",
			opts: InitOptions{
				ProjectName: "mycli",
				ModuleName:  "github.com/user/mycli",
				Template:    "cli",
				Author:      "Test Author",
				GoVersion:   "1.25.1",
				OutputDir:   filepath.Join(tempDir, "cli-test"),
				Description: "A test CLI application",
			},
			expectFiles: []string{
				"cmd/mycli/main.go",
				"go.mod",
				"README.md",
				".gitignore",
				"Makefile",
			},
			expectError: false,
		},
		{
			name: "library project initialization",
			opts: InitOptions{
				ProjectName: "mylib",
				ModuleName:  "github.com/user/mylib",
				Template:    "library",
				Author:      "Test Author",
				GoVersion:   "1.25.1",
				OutputDir:   filepath.Join(tempDir, "lib-test"),
				Description: "A test library",
			},
			expectFiles: []string{
				"mylib.go",
				"go.mod",
				"README.md",
				".gitignore",
			},
			expectError: false,
		},
		{
			name: "API project initialization",
			opts: InitOptions{
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
				Template:    "api",
				Author:      "Test Author",
				GoVersion:   "1.25.1",
				OutputDir:   filepath.Join(tempDir, "api-test"),
				Description: "A test API",
			},
			expectFiles: []string{
				"cmd/myapi/main.go",
				"go.mod",
				"README.md",
				".gitignore",
				"Makefile",
			},
			expectError: false,
		},
		{
			name: "invalid template",
			opts: InitOptions{
				ProjectName: "test",
				ModuleName:  "github.com/user/test",
				Template:    "nonexistent",
				Author:      "Test Author",
				OutputDir:   filepath.Join(tempDir, "invalid-test"),
			},
			expectFiles: nil,
			expectError: true,
		},
		{
			name: "missing required fields",
			opts: InitOptions{
				Template:  "cli",
				OutputDir: filepath.Join(tempDir, "missing-test"),
			},
			expectFiles: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.InitProject(ctx, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result.ProjectPath)
			assert.True(t, result.Success)
			assert.GreaterOrEqual(t, result.FilesCreated, len(tt.expectFiles))

			// Verify expected files exist
			for _, expectedFile := range tt.expectFiles {
				filePath := filepath.Join(tt.opts.OutputDir, expectedFile)
				_, err := os.Stat(filePath)
				assert.NoError(t, err, "file %s should exist", expectedFile)

				// Verify file has content
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				assert.NotEmpty(t, content, "file %s should not be empty", expectedFile)

				// Verify template variables were substituted
				contentStr := string(content)
				assert.NotContains(t, contentStr, "{{ ProjectName }}", "ProjectName should be substituted")
				assert.NotContains(t, contentStr, "{{ ModuleName }}", "ModuleName should be substituted")
				assert.NotContains(t, contentStr, "{{ Author }}", "Author should be substituted")
			}
		})
	}
}

func TestProjectGenerator_ValidateOptions(t *testing.T) {
	engine := templates.NewEngine()
	repo := templates.NewRepository()
	generator := NewProjectGenerator(engine, repo)
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		opts    InitOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: InitOptions{
				ProjectName: "test",
				ModuleName:  "github.com/user/test",
				Template:    "cli",
				Author:      "Test Author",
				OutputDir:   filepath.Join(tempDir, "test"),
			},
			wantErr: false,
		},
		{
			name: "missing project name",
			opts: InitOptions{
				ModuleName: "github.com/user/test",
				Template:   "cli",
				OutputDir:  filepath.Join(tempDir, "missing-project"),
			},
			wantErr: true,
		},
		{
			name: "missing module name",
			opts: InitOptions{
				ProjectName: "test",
				Template:    "cli",
				OutputDir:   filepath.Join(tempDir, "missing-module"),
			},
			wantErr: true,
		},
		{
			name: "missing template",
			opts: InitOptions{
				ProjectName: "test",
				ModuleName:  "github.com/user/test",
				OutputDir:   "/tmp",
			},
			wantErr: true,
		},
		{
			name: "empty output dir defaults to current",
			opts: InitOptions{
				ProjectName: "test",
				ModuleName:  "github.com/user/test",
				Template:    "cli",
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

func TestProjectGenerator_DryRun(t *testing.T) {
	tempDir := t.TempDir()

	engine := templates.NewEngine()
	repo := templates.NewRepository()
	generator := NewProjectGenerator(engine, repo)
	ctx := context.Background()

	opts := InitOptions{
		ProjectName: "dryruntest",
		ModuleName:  "github.com/user/dryruntest",
		Template:    "cli",
		Author:      "Test Author",
		OutputDir:   filepath.Join(tempDir, "dryrun-test"),
		DryRun:      true,
	}

	result, err := generator.InitProject(ctx, opts)
	require.NoError(t, err)

	// Should return success but not create files
	assert.True(t, result.Success)
	assert.Greater(t, result.FilesCreated, 0) // Should report files that would be created

	// Verify no files were actually created
	_, err = os.Stat(opts.OutputDir)
	assert.True(t, os.IsNotExist(err), "output directory should not exist in dry run")
}
