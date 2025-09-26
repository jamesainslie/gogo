package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/user/gogo/internal/db"
)

func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
		Long: color.GreenString(`Manage the gogo SQLite database.

The database stores templates, blueprints, configurations, and audit logs.`),
	}

	cmd.AddCommand(newDBInitCommand())
	cmd.AddCommand(newDBMigrateCommand())
	cmd.AddCommand(newDBBackupCommand())
	cmd.AddCommand(newDBExportCommand())

	return cmd
}

func newDBInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize database",
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Yellow("Initializing database at: %s", dbPath)

			manager := db.NewManager()
			if err := manager.Open(cmd.Context(), dbPath); err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			color.Green("Database initialized successfully!")
			return nil
		},
	}
}

func newDBMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Yellow("Running migrations on: %s", dbPath)
			return fmt.Errorf("db migrate not implemented yet")
		},
	}
}

func newDBBackupCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup database",
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Yellow("Backing up database to: %s", outputFile)
			return fmt.Errorf("db backup not implemented yet")
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "backup.db", "Backup file path")
	return cmd
}

func newDBExportCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export database to SQL",
		RunE: func(cmd *cobra.Command, args []string) error {
			color.Yellow("Exporting database to: %s", outputFile)
			return fmt.Errorf("db export not implemented yet")
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "backup.sql", "SQL file path")
	return cmd
}
