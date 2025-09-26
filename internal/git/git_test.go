package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitInstalled(t *testing.T) {
	// This test assumes git is installed on the development machine
	// If git is not available, the test will pass but note it
	installed := IsGitInstalled()
	if !installed {
		t.Log("Git is not installed, skipping git-dependent tests")
	}
	// Don't fail if git is not installed, as this might be expected in some environments
}

func TestGitManager_ValidateWorkingDir(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func(t *testing.T) string
		expectError bool
	}{
		{
			name: "valid existing directory",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: false,
		},
		{
			name: "non-existent directory that can be created",
			setupDir: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "newdir")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir := tt.setupDir(t)
			manager := NewGitManager(workingDir)

			err := manager.ValidateWorkingDir()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify directory exists and is writable
				stat, err := os.Stat(workingDir)
				require.NoError(t, err)
				assert.True(t, stat.IsDir())
			}
		})
	}
}

func TestGitManager_Init(t *testing.T) {
	if !IsGitInstalled() {
		t.Skip("Git is not installed, skipping git init test")
	}

	tests := []struct {
		name    string
		opts    InitOptions
		wantErr bool
	}{
		{
			name: "successful init with basic options",
			opts: InitOptions{
				ProjectName:          "testproject",
				Author:               "Test Author",
				Email:                "test@example.com",
				InitialCommitMessage: "",
			},
			wantErr: false,
		},
		{
			name: "successful init with custom commit message",
			opts: InitOptions{
				ProjectName:          "testproject2",
				Author:               "Test Author 2",
				Email:                "test2@example.com",
				InitialCommitMessage: "Custom initial commit",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			manager := NewGitManager(tmpDir)

			err := manager.Init(context.Background(), tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify git repository was created
				gitDir := filepath.Join(tmpDir, ".git")
				stat, err := os.Stat(gitDir)
				require.NoError(t, err)
				assert.True(t, stat.IsDir())

				// Verify it's a git repository
				assert.True(t, manager.IsGitRepository(context.Background()))
			}
		})
	}
}

func TestGitManager_InitialCommit(t *testing.T) {
	if !IsGitInstalled() {
		t.Skip("Git is not installed, skipping git commit test")
	}

	tmpDir := t.TempDir()
	manager := NewGitManager(tmpDir)

	// Initialize git repository first
	err := manager.Init(context.Background(), InitOptions{
		ProjectName: "testproject",
		Author:      "Test Author",
		Email:       "test@example.com",
	})
	require.NoError(t, err)

	// Create some test files
	testFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Project"), 0644)
	require.NoError(t, err)

	// Test initial commit
	opts := InitOptions{
		ProjectName:          "testproject",
		InitialCommitMessage: "Initial test commit",
	}

	err = manager.InitialCommit(context.Background(), opts)
	assert.NoError(t, err)
}

func TestGitManager_IsGitRepository(t *testing.T) {
	tests := []struct {
		name     string
		setupDir func(t *testing.T) string
		expected bool
	}{
		{
			name: "non-git directory",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
		{
			name: "git directory",
			setupDir: func(t *testing.T) string {
				if !IsGitInstalled() {
					t.Skip("Git is not installed")
				}
				tmpDir := t.TempDir()
				manager := NewGitManager(tmpDir)
				err := manager.Init(context.Background(), InitOptions{
					ProjectName: "test",
				})
				require.NoError(t, err)
				return tmpDir
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workingDir := tt.setupDir(t)
			manager := NewGitManager(workingDir)

			result := manager.IsGitRepository(context.Background())
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserInfo(t *testing.T) {
	if !IsGitInstalled() {
		t.Skip("Git is not installed, skipping user info test")
	}

	// This test will return whatever git config is set, or empty strings
	name, email := GetUserInfo(context.Background())

	// Just verify the function doesn't panic and returns strings
	assert.IsType(t, "", name)
	assert.IsType(t, "", email)
	t.Logf("Git user info - Name: %s, Email: %s", name, email)
}
