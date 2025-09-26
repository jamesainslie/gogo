package cicd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	generator := NewGenerator()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.templateEngine)
}

func TestGenerator_GenerateGolangCIConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "basic configuration",
			config: Config{
				ProjectName:   "testproject",
				GoVersion:     "1.25.1",
				CoverageMin:   0.80,
				TestFramework: "testify",
				LintTimeout:   "5m",
			},
		},
		{
			name: "configuration with custom timeout",
			config: Config{
				ProjectName: "testproject2",
				GoVersion:   "1.24",
				LintTimeout: "10m",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator()
			tmpDir := t.TempDir()

			err := generator.GenerateGolangCIConfig(context.Background(), tmpDir, tt.config)
			require.NoError(t, err)

			// Verify file was created
			configFile := filepath.Join(tmpDir, ".golangci.yml")
			assert.FileExists(t, configFile)

			// Verify content contains expected elements
			content, err := os.ReadFile(configFile)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, "timeout: "+tt.config.LintTimeout)
			assert.Contains(t, contentStr, "enable:")
			assert.Contains(t, contentStr, "- errcheck")
			assert.Contains(t, contentStr, "- gosimple")
			assert.Contains(t, contentStr, "- staticcheck")
		})
	}
}

func TestGenerator_GenerateGitHubActions(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "basic project",
			config: Config{
				ProjectName:  "testproject",
				GoVersion:    "1.25.1",
				CoverageMin:  0.80,
				HasDatabase:  false,
				HasDocker:    false,
				BuildTargets: []string{"linux", "darwin", "windows"},
			},
		},
		{
			name: "project with database",
			config: Config{
				ProjectName:  "dbproject",
				GoVersion:    "1.25.1",
				CoverageMin:  0.85,
				HasDatabase:  true,
				DatabaseType: "postgres",
				BuildTargets: []string{"linux", "darwin"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator()
			tmpDir := t.TempDir()

			err := generator.GenerateGitHubActions(context.Background(), tmpDir, tt.config)
			require.NoError(t, err)

			// Verify file was created
			workflowFile := filepath.Join(tmpDir, ".github", "workflows", "ci.yml")
			assert.FileExists(t, workflowFile)

			// Verify content contains expected elements
			content, err := os.ReadFile(workflowFile)
			require.NoError(t, err)

			contentStr := string(content)
			assert.Contains(t, contentStr, "name: CI")
			assert.Contains(t, contentStr, tt.config.GoVersion)
			assert.Contains(t, contentStr, "go-version: \""+tt.config.GoVersion+"\"")

			if tt.config.HasDatabase {
				assert.Contains(t, contentStr, "services:")
				assert.Contains(t, contentStr, "postgres:")
			}

			// Check build targets
			for _, target := range tt.config.BuildTargets {
				assert.Contains(t, contentStr, "\""+target+"\"")
			}
		})
	}
}

func TestGenerator_GeneratePreCommitConfig(t *testing.T) {
	generator := NewGenerator()
	tmpDir := t.TempDir()

	config := Config{
		ProjectName: "testproject",
		GoVersion:   "1.25.1",
	}

	err := generator.GeneratePreCommitConfig(context.Background(), tmpDir, config)
	require.NoError(t, err)

	// Verify file was created
	configFile := filepath.Join(tmpDir, ".pre-commit-config.yaml")
	assert.FileExists(t, configFile)

	// Verify content contains expected elements
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "repos:")
	assert.Contains(t, contentStr, "golangci-lint")
	assert.Contains(t, contentStr, "go fmt")
	assert.Contains(t, contentStr, "go mod tidy")
	assert.Contains(t, contentStr, "go test")
	assert.Contains(t, contentStr, "go build")
}

func TestGenerator_GenerateAll(t *testing.T) {
	generator := NewGenerator()
	tmpDir := t.TempDir()

	config := Config{
		ProjectName:   "fullproject",
		GoVersion:     "1.25.1",
		CoverageMin:   0.75,
		TestFramework: "testify",
		HasDatabase:   true,
		DatabaseType:  "postgres",
		HasDocker:     true,
		LintTimeout:   "5m",
		BuildTargets:  []string{"linux", "darwin", "windows"},
	}

	err := generator.GenerateAll(context.Background(), tmpDir, config)
	require.NoError(t, err)

	// Verify all expected files were created
	expectedFiles := []string{
		".golangci.yml",
		".github/workflows/ci.yml",
		".pre-commit-config.yaml",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(tmpDir, file)
		assert.FileExists(t, fullPath, "Expected file %s to exist", file)

		// Verify file has content
		stat, err := os.Stat(fullPath)
		require.NoError(t, err)
		assert.Greater(t, stat.Size(), int64(0), "Expected file %s to have content", file)
	}
}

func TestConfig_Defaults(t *testing.T) {
	generator := NewGenerator()
	tmpDir := t.TempDir()

	// Test with minimal config to verify defaults are applied
	config := Config{
		ProjectName: "defaultstest",
	}

	err := generator.GenerateAll(context.Background(), tmpDir, config)
	require.NoError(t, err)

	// Check that defaults were applied by examining the generated content
	ciFile := filepath.Join(tmpDir, ".github", "workflows", "ci.yml")
	content, err := os.ReadFile(ciFile)
	require.NoError(t, err)

	contentStr := string(content)
	// Default Go version should be 1.25.1
	assert.Contains(t, contentStr, "1.25.1")
	// Default build targets should include linux, darwin, windows
	assert.Contains(t, contentStr, "linux")
	assert.Contains(t, contentStr, "darwin")
	assert.Contains(t, contentStr, "windows")
}

func TestGenerator_GenerateGitHubActions_NoDatabaseService(t *testing.T) {
	generator := NewGenerator()
	tmpDir := t.TempDir()

	config := Config{
		ProjectName: "nodbproject",
		GoVersion:   "1.25.1",
		HasDatabase: false,
	}

	err := generator.GenerateGitHubActions(context.Background(), tmpDir, config)
	require.NoError(t, err)

	workflowFile := filepath.Join(tmpDir, ".github", "workflows", "ci.yml")
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err)

	contentStr := string(content)
	// Should not contain database service configuration
	assert.NotContains(t, contentStr, "services:")
	assert.NotContains(t, contentStr, "postgres:")
	assert.NotContains(t, contentStr, "DATABASE_URL")
}
