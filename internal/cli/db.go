package cli

import (
	"context"
	"fmt"
	"strings"

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
	cmd.AddCommand(newDBRestoreCommand())
	cmd.AddCommand(newDBExportCommand())
	cmd.AddCommand(newDBImportCommand())
	cmd.AddCommand(newDBStatusCommand())
	cmd.AddCommand(newDBVacuumCommand())
	cmd.AddCommand(newDBIntegrityCommand())
	cmd.AddCommand(newDBSizeCommand())

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
	var rollback bool
	var status bool
	var count int

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long: color.GreenString(`Run database migrations to update schema.
		
Use --status to see migration status.
Use --rollback to rollback the last migration.
Use --count=N to rollback N migrations.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			migrationManager := db.NewMigrationManager(manager.GetDB())
			migrationManager.RegisterCoreSchemas()

			if status {
				return showMigrationStatus(ctx, migrationManager)
			}

			if rollback {
				if count > 1 {
					color.Yellow("Rolling back %d migrations...", count)
					for i := 0; i < count; i++ {
						if err := migrationManager.RollbackLast(ctx); err != nil {
							return fmt.Errorf("rollback failed: %w", err)
						}
					}
				} else {
					color.Yellow("Rolling back last migration...")
					if err := migrationManager.RollbackLast(ctx); err != nil {
						return fmt.Errorf("rollback failed: %w", err)
					}
				}
				return nil
			}

			// Apply all pending migrations
			return migrationManager.ApplyAll(ctx)
		},
	}

	cmd.Flags().BoolVar(&rollback, "rollback", false, "Rollback migrations instead of applying")
	cmd.Flags().BoolVar(&status, "status", false, "Show migration status")
	cmd.Flags().IntVar(&count, "count", 1, "Number of migrations to rollback")

	return cmd
}

func newDBBackupCommand() *cobra.Command {
	var outputFile string
	var compress bool
	var verify bool

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup database",
		Long: color.GreenString(`Create a backup of the database.

Use --compress to create a gzip-compressed backup.
Use --verify to verify backup integrity after creation.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			backupManager := db.NewBackupManager(manager, dbPath)

			opts := db.BackupOptions{
				OutputPath: outputFile,
				Compress:   compress,
				Verify:     verify,
				Verbose:    verbose,
			}

			return backupManager.Backup(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "backup.db", "Backup file path")
	cmd.Flags().BoolVar(&compress, "compress", false, "Create compressed backup")
	cmd.Flags().BoolVar(&verify, "verify", false, "Verify backup after creation")
	return cmd
}

func newDBExportCommand() *cobra.Command {
	var outputFile string
	var format string
	var tables []string
	var includeSchema bool
	var includeData bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export database to various formats",
		Long: color.GreenString(`Export database to SQL, JSON, or CSV format.

Formats: sql, json, csv
Use --tables to export specific tables only.
Use --schema-only or --data-only for partial exports.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			exportManager := db.NewExportManager(manager)

			// Determine export format from file extension if not specified
			if format == "" {
				switch {
				case strings.HasSuffix(outputFile, ".sql"):
					format = "sql"
				case strings.HasSuffix(outputFile, ".json"):
					format = "json"
				case strings.HasSuffix(outputFile, ".csv"):
					format = "csv"
				default:
					format = "sql" // Default to SQL
				}
			}

			opts := db.ExportOptions{
				OutputPath:    outputFile,
				Format:        db.ExportFormat(format),
				Tables:        tables,
				IncludeSchema: includeSchema,
				IncludeData:   includeData,
				Verbose:       verbose,
			}

			return exportManager.Export(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "export.sql", "Output file path")
	cmd.Flags().StringVar(&format, "format", "", "Export format (sql, json, csv)")
	cmd.Flags().StringSliceVar(&tables, "tables", nil, "Tables to export (empty = all)")
	cmd.Flags().BoolVar(&includeSchema, "schema", true, "Include table schemas")
	cmd.Flags().BoolVar(&includeData, "data", true, "Include table data")
	return cmd
}

func newDBRestoreCommand() *cobra.Command {
	var backupFile string
	var verify bool
	var createBackup bool
	var force bool

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore database from backup",
		Long: color.GreenString(`Restore database from a backup file.

Use --verify to check backup integrity before restore.
Use --backup to create backup of existing database first.
Use --force to overwrite existing database.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if backupFile == "" && len(args) > 0 {
				backupFile = args[0]
			}
			if backupFile == "" {
				return fmt.Errorf("backup file path is required")
			}

			manager := db.NewManager()
			backupManager := db.NewBackupManager(manager, dbPath)

			opts := db.RestoreOptions{
				BackupPath:   backupFile,
				Verify:       verify,
				CreateBackup: createBackup,
				Force:        force,
				Verbose:      verbose,
			}

			return backupManager.Restore(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&backupFile, "from", "", "Backup file to restore from")
	cmd.Flags().BoolVar(&verify, "verify", false, "Verify backup before restore")
	cmd.Flags().BoolVar(&createBackup, "backup", false, "Backup existing database first")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing database")
	return cmd
}

func newDBImportCommand() *cobra.Command {
	var inputFile string
	var format string
	var validate bool
	var dryRun bool
	var replace bool

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import data into database",
		Long: color.GreenString(`Import data from SQL or JSON files.

Use --dry-run to preview import without making changes.
Use --validate to check data integrity before import.
Use --replace to replace existing data.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if inputFile == "" && len(args) > 0 {
				inputFile = args[0]
			}
			if inputFile == "" {
				return fmt.Errorf("input file path is required")
			}

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			exportManager := db.NewExportManager(manager)

			// Determine format from file extension if not specified
			if format == "" {
				switch {
				case strings.HasSuffix(inputFile, ".sql"):
					format = "sql"
				case strings.HasSuffix(inputFile, ".json"):
					format = "json"
				default:
					format = "sql" // Default to SQL
				}
			}

			opts := db.ImportOptions{
				InputPath:       inputFile,
				Format:          db.ExportFormat(format),
				Validate:        validate,
				DryRun:          dryRun,
				ReplaceExisting: replace,
				Verbose:         verbose,
			}

			return exportManager.Import(ctx, opts)
		},
	}

	cmd.Flags().StringVar(&inputFile, "from", "", "Input file to import from")
	cmd.Flags().StringVar(&format, "format", "", "Import format (sql, json)")
	cmd.Flags().BoolVar(&validate, "validate", true, "Validate data before import")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview import without changes")
	cmd.Flags().BoolVar(&replace, "replace", false, "Replace existing data")
	return cmd
}

func newDBStatusCommand() *cobra.Command {
	var detailed bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show database health status",
		Long: color.GreenString(`Show comprehensive database health information.

Includes connectivity, integrity, performance metrics, and recommendations.
Use --detailed for additional statistics and table information.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			healthManager := db.NewHealthManager(manager, dbPath)

			_, err := healthManager.CheckHealth(ctx, true) // Always verbose for status command
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}

			if detailed {
				stats, err := healthManager.GetDatabaseStats(ctx)
				if err != nil {
					color.Yellow("Warning: could not retrieve detailed stats: %v", err)
				} else {
					fmt.Println()
					color.Yellow("=== Database Statistics ===")
					fmt.Printf("Total Size: %.2f MB\n", float64(stats.TotalSize)/1024/1024)
					fmt.Printf("Page Count: %d\n", stats.PageCount)
					fmt.Printf("Page Size: %d bytes\n", stats.PageSize)
					fmt.Printf("Journal Mode: %s\n", stats.JournalMode)
					if stats.WALSize > 0 {
						fmt.Printf("WAL Size: %.2f MB\n", float64(stats.WALSize)/1024/1024)
					}

					if len(stats.Tables) > 0 {
						fmt.Println()
						color.Yellow("=== Table Statistics ===")
						for _, table := range stats.Tables {
							fmt.Printf("%-20s %d rows\n", table.Name+":", table.RowCount)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed database statistics")
	return cmd
}

func newDBVacuumCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "vacuum",
		Short: "Optimize database (VACUUM)",
		Long: color.GreenString(`Optimize the database by reclaiming unused space.

This command rebuilds the database file, removing fragmentation
and unused pages. May take time for large databases.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			healthManager := db.NewHealthManager(manager, dbPath)
			return healthManager.VacuumDatabase(ctx, verbose)
		},
	}
}

func newDBIntegrityCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "integrity",
		Short: "Check database integrity",
		Long: color.GreenString(`Check the integrity of the database.

Runs SQLite's PRAGMA integrity_check to verify
that the database structure is valid.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			var result string
			err := manager.GetDB().QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result)
			if err != nil {
				return fmt.Errorf("integrity check failed: %w", err)
			}

			if result == "ok" {
				color.Green("✓ Database integrity check passed")
			} else {
				color.Red("✗ Database integrity issues found:")
				fmt.Println(result)
				return fmt.Errorf("database integrity check failed")
			}

			return nil
		},
	}
}

func newDBSizeCommand() *cobra.Command {
	var breakdown bool

	cmd := &cobra.Command{
		Use:   "size",
		Short: "Show database size information",
		Long: color.GreenString(`Show database size and space usage.

Use --breakdown to show size breakdown by table.`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			manager := db.NewManager()
			if err := manager.Open(ctx, dbPath); err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer func() {
				if closeErr := manager.Close(); closeErr != nil {
					color.Red("Warning: failed to close database: %v", closeErr)
				}
			}()

			healthManager := db.NewHealthManager(manager, dbPath)
			stats, err := healthManager.GetDatabaseStats(ctx)
			if err != nil {
				return fmt.Errorf("failed to get database stats: %w", err)
			}

			color.Yellow("=== Database Size ===")
			fmt.Printf("Database File: %.2f MB\n", float64(stats.TotalSize)/1024/1024)
			if stats.WALSize > 0 {
				fmt.Printf("WAL File: %.2f MB\n", float64(stats.WALSize)/1024/1024)
				fmt.Printf("Total Size: %.2f MB\n", float64(stats.TotalSize+stats.WALSize)/1024/1024)
			}
			fmt.Printf("Page Count: %d\n", stats.PageCount)
			fmt.Printf("Page Size: %d bytes\n", stats.PageSize)

			if breakdown && len(stats.Tables) > 0 {
				fmt.Println()
				color.Yellow("=== Size by Table ===")
				for _, table := range stats.Tables {
					fmt.Printf("%-20s %d rows\n", table.Name+":", table.RowCount)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&breakdown, "breakdown", false, "Show size breakdown by table")
	return cmd
}

// Helper functions

func showMigrationStatus(ctx context.Context, migrationManager *db.MigrationManager) error {
	migrations, err := migrationManager.GetMigrationStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	color.Yellow("=== Migration Status ===")

	if len(migrations) == 0 {
		color.Yellow("No migrations registered")
		return nil
	}

	for _, migration := range migrations {
		status := "PENDING"
		timestamp := ""

		if migration.Applied {
			status = color.GreenString("APPLIED")
			if migration.AppliedAt != nil {
				timestamp = migration.AppliedAt.Format("2006-01-02 15:04:05")
			}
		} else {
			status = color.YellowString("PENDING")
		}

		if timestamp != "" {
			fmt.Printf("%-30s %s (%s)\n", migration.ID, status, timestamp)
		} else {
			fmt.Printf("%-30s %s\n", migration.ID, status)
		}

		if migration.Description != "" {
			fmt.Printf("  %s\n", migration.Description)
		}
		fmt.Println()
	}

	return nil
}
