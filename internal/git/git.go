package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// GitManager handles git operations
type GitManager struct {
	workingDir string
}

// NewGitManager creates a new git manager
func NewGitManager(workingDir string) *GitManager {
	return &GitManager{
		workingDir: workingDir,
	}
}

// InitOptions contains options for git initialization
type InitOptions struct {
	ProjectName          string
	Author               string
	Email                string
	InitialCommitMessage string
}

// IsGitInstalled checks if git is available in the system
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsGitRepository checks if the directory is already a git repository
func (g *GitManager) IsGitRepository(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = g.workingDir
	err := cmd.Run()
	return err == nil
}

// Init initializes a git repository
func (g *GitManager) Init(ctx context.Context, opts InitOptions) error {
	if !IsGitInstalled() {
		return fmt.Errorf("git is not installed or not available in PATH")
	}

	// Check if already a git repository
	if g.IsGitRepository(ctx) {
		color.Yellow("Directory is already a git repository, skipping git init")
		return nil
	}

	// Initialize git repository
	if err := g.runGitCommand(ctx, "init"); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Configure user name and email if provided
	if opts.Author != "" {
		if err := g.setGitConfig(ctx, "user.name", opts.Author); err != nil {
			// Don't fail if git config fails, just warn
			color.Yellow("Warning: failed to set git user.name: %v", err)
		}
	}

	if opts.Email != "" {
		if err := g.setGitConfig(ctx, "user.email", opts.Email); err != nil {
			// Don't fail if git config fails, just warn
			color.Yellow("Warning: failed to set git user.email: %v", err)
		}
	}

	// Set default branch to main
	if err := g.setGitConfig(ctx, "init.defaultBranch", "main"); err != nil {
		// Ignore error for older git versions
		color.Yellow("Warning: failed to set default branch to main (git version may be old)")
	}

	return nil
}

// AddAll adds all files to git staging
func (g *GitManager) AddAll(ctx context.Context) error {
	return g.runGitCommand(ctx, "add", ".")
}

// Commit creates an initial commit
func (g *GitManager) Commit(ctx context.Context, message string) error {
	if message == "" {
		message = "Initial commit from gogo"
	}
	return g.runGitCommand(ctx, "commit", "-m", message)
}

// InitialCommit performs git add . && git commit with initial message
func (g *GitManager) InitialCommit(ctx context.Context, opts InitOptions) error {
	// Add all files
	if err := g.AddAll(ctx); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}

	// Create initial commit
	message := opts.InitialCommitMessage
	if message == "" {
		message = fmt.Sprintf("Initial commit: %s project created with gogo", opts.ProjectName)
	}

	if err := g.Commit(ctx, message); err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	return nil
}

// GetUserInfo retrieves git user information
func GetUserInfo(ctx context.Context) (name string, email string) {
	// Try to get user name
	if cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.name"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			name = strings.TrimSpace(string(output))
		}
	}

	// Try to get user email
	if cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.email"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			email = strings.TrimSpace(string(output))
		}
	}

	return name, email
}

// runGitCommand runs a git command in the working directory
func (g *GitManager) runGitCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.workingDir

	// Capture both stdout and stderr for better error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// setGitConfig sets a git configuration value
func (g *GitManager) setGitConfig(ctx context.Context, key, value string) error {
	return g.runGitCommand(ctx, "config", key, value)
}

// ValidateWorkingDir ensures the working directory exists and is writable
func (g *GitManager) ValidateWorkingDir() error {
	// Check if directory exists
	if _, err := os.Stat(g.workingDir); os.IsNotExist(err) {
		// Try to create it
		if err := os.MkdirAll(g.workingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory %s: %w", g.workingDir, err)
		}
	}

	// Check if directory is writable
	testFile := filepath.Join(g.workingDir, ".gogo_write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("working directory %s is not writable: %w", g.workingDir, err)
	}

	// Clean up test file
	_ = os.Remove(testFile)

	return nil
}
