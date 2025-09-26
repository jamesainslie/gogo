package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	dbPath    string
	outputDir string
	goVersion string
	dryRun    bool
	verbose   bool
)

// Execute runs the root command
func Execute(ctx context.Context, version string) error {
	rootCmd := &cobra.Command{
		Use:   "gogo",
		Short: "A Go project scaffolding CLI tool",
		Long: color.CyanString(`gogo - Go Project Scaffolding CLI

A command-line tool for generating idiomatic Go project scaffolds with templates,
blueprints, and team collaboration features.`),
		Version: version,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", getDefaultDBPath(), "Path to SQLite database")
	rootCmd.PersistentFlags().StringVar(&outputDir, "output-dir", ".", "Output directory for generated files")
	rootCmd.PersistentFlags().StringVar(&goVersion, "go-version", "", "Go version to use (auto-detect if empty)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newGenerateCommand())
	rootCmd.AddCommand(newAddCommand())
	rootCmd.AddCommand(newDBCommand())

	return rootCmd.ExecuteContext(ctx)
}

func getDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".gogo.db"
	}
	return filepath.Join(homeDir, ".gogo.db")
}
