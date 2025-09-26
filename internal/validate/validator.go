package validate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Go module name pattern (simplified)
	moduleNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-._/]*[a-zA-Z0-9])?$`)
	// Project name pattern
	projectNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-_]*$`)
)

// ValidateModuleName validates a Go module name format
func ValidateModuleName(name string) error {
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	if !moduleNamePattern.MatchString(name) {
		return fmt.Errorf("invalid module name format: %s", name)
	}

	// Check for common invalid patterns
	if strings.Contains(name, "..") {
		return fmt.Errorf("module name cannot contain consecutive dots")
	}

	return nil
}

// ValidateProjectName validates a project name
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("invalid project name: must start with letter and contain only letters, numbers, hyphens, and underscores")
	}

	return nil
}

// ValidateOutputDir validates that the output directory is writable
func ValidateOutputDir(dir string) error {
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid output directory path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("output directory does not exist: %s", absDir)
		}
		return fmt.Errorf("cannot access output directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("output path is not a directory: %s", absDir)
	}

	// Test write permissions by creating a temp file
	tempFile := filepath.Join(absDir, ".gogo-write-test")
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("output directory is not writable: %s", absDir)
	}
	_ = f.Close()
	_ = os.Remove(tempFile)

	return nil
}

// ValidateGoVersion validates Go version format
func ValidateGoVersion(version string) error {
	if version == "" {
		return nil // Empty is valid (auto-detect)
	}

	// Simple version pattern check
	versionPattern := regexp.MustCompile(`^1\.\d+(\.\d+)?$`)
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid Go version format: %s (expected format: 1.x or 1.x.y)", version)
	}

	return nil
}

// CheckGoToolchain verifies that Go toolchain is available
func CheckGoToolchain() error {
	_, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go toolchain not found in PATH")
	}

	// Test go version command
	cmd := exec.Command("go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go toolchain is not working properly: %w", err)
	}

	return nil
}

// CheckGitAvailable checks if Git is available (optional)
func CheckGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
