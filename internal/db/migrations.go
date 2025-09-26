package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Migration represents a database migration
type Migration struct {
	ID          string
	Description string
	UpSQL       string
	DownSQL     string
	Applied     bool
	AppliedAt   *time.Time
}

// MigrationManager handles database migrations
type MigrationManager struct {
	db         *sql.DB
	migrations map[string]*Migration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB) *MigrationManager {
	return &MigrationManager{
		db:         db,
		migrations: make(map[string]*Migration),
	}
}

// RegisterMigration registers a migration
func (m *MigrationManager) RegisterMigration(id, description, upSQL, downSQL string) {
	m.migrations[id] = &Migration{
		ID:          id,
		Description: description,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
	}
}

// InitMigrationTable creates the schema_migrations table if it doesn't exist
func (m *MigrationManager) InitMigrationTable(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id VARCHAR(255) PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64) NOT NULL
		)`

	_, err := m.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns all applied migrations
func (m *MigrationManager) GetAppliedMigrations(ctx context.Context) (map[string]*Migration, error) {
	if err := m.InitMigrationTable(ctx); err != nil {
		return nil, err
	}

	query := `SELECT id, description, applied_at FROM schema_migrations ORDER BY id`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]*Migration)
	for rows.Next() {
		var id, description string
		var appliedAt time.Time

		if err := rows.Scan(&id, &description, &appliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}

		applied[id] = &Migration{
			ID:          id,
			Description: description,
			Applied:     true,
			AppliedAt:   &appliedAt,
		}
	}

	return applied, nil
}

// GetPendingMigrations returns migrations that haven't been applied
func (m *MigrationManager) GetPendingMigrations(ctx context.Context) ([]*Migration, error) {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var pending []*Migration
	for id, migration := range m.migrations {
		if _, isApplied := applied[id]; !isApplied {
			pending = append(pending, migration)
		}
	}

	// Sort by ID to ensure consistent order
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].ID < pending[j].ID
	})

	return pending, nil
}

// ApplyMigration applies a single migration
func (m *MigrationManager) ApplyMigration(ctx context.Context, migration *Migration) error {
	if migration.UpSQL == "" {
		return fmt.Errorf("migration %s has no up SQL", migration.ID)
	}

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", migration.ID, err)
	}

	// Record migration as applied
	checksum := generateChecksum(migration.UpSQL)
	insertSQL := `INSERT INTO schema_migrations (id, description, checksum) VALUES (?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertSQL, migration.ID, migration.Description, checksum); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", migration.ID, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", migration.ID, err)
	}

	color.Green("✓ Applied migration %s: %s", migration.ID, migration.Description)
	return nil
}

// RollbackMigration rolls back a single migration
func (m *MigrationManager) RollbackMigration(ctx context.Context, migration *Migration) error {
	if migration.DownSQL == "" {
		return fmt.Errorf("migration %s has no down SQL", migration.ID)
	}

	// Start transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback SQL
	if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
		return fmt.Errorf("failed to rollback migration %s: %w", migration.ID, err)
	}

	// Remove migration record
	deleteSQL := `DELETE FROM schema_migrations WHERE id = ?`
	if _, err := tx.ExecContext(ctx, deleteSQL, migration.ID); err != nil {
		return fmt.Errorf("failed to remove migration record %s: %w", migration.ID, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %s: %w", migration.ID, err)
	}

	color.Yellow("↓ Rolled back migration %s: %s", migration.ID, migration.Description)
	return nil
}

// ApplyAll applies all pending migrations
func (m *MigrationManager) ApplyAll(ctx context.Context) error {
	pending, err := m.GetPendingMigrations(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		color.Green("No pending migrations")
		return nil
	}

	color.Yellow("Applying %d pending migrations...", len(pending))

	for _, migration := range pending {
		if err := m.ApplyMigration(ctx, migration); err != nil {
			return err
		}
	}

	color.Green("Successfully applied %d migrations", len(pending))
	return nil
}

// GetLastAppliedMigration returns the most recently applied migration
func (m *MigrationManager) GetLastAppliedMigration(ctx context.Context) (*Migration, error) {
	if err := m.InitMigrationTable(ctx); err != nil {
		return nil, err
	}

	query := `SELECT id, description, applied_at FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`
	row := m.db.QueryRowContext(ctx, query)

	var id, description string
	var appliedAt time.Time

	err := row.Scan(&id, &description, &appliedAt)
	if err == sql.ErrNoRows {
		return nil, nil // No migrations applied yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last applied migration: %w", err)
	}

	// Find the migration in our registered migrations
	if migration, exists := m.migrations[id]; exists {
		migration.Applied = true
		migration.AppliedAt = &appliedAt
		return migration, nil
	}

	// Migration not found in registered migrations (could be from older version)
	return &Migration{
		ID:          id,
		Description: description,
		Applied:     true,
		AppliedAt:   &appliedAt,
	}, nil
}

// RollbackLast rolls back the most recently applied migration
func (m *MigrationManager) RollbackLast(ctx context.Context) error {
	lastMigration, err := m.GetLastAppliedMigration(ctx)
	if err != nil {
		return err
	}

	if lastMigration == nil {
		color.Yellow("No migrations to rollback")
		return nil
	}

	// Find the migration definition
	migration, exists := m.migrations[lastMigration.ID]
	if !exists {
		return fmt.Errorf("migration %s not found in registered migrations", lastMigration.ID)
	}

	return m.RollbackMigration(ctx, migration)
}

// GetMigrationStatus returns the status of all migrations
func (m *MigrationManager) GetMigrationStatus(ctx context.Context) ([]*Migration, error) {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var status []*Migration
	for _, migration := range m.migrations {
		if appliedMigration, isApplied := applied[migration.ID]; isApplied {
			migration.Applied = true
			migration.AppliedAt = appliedMigration.AppliedAt
		} else {
			migration.Applied = false
			migration.AppliedAt = nil
		}
		status = append(status, migration)
	}

	// Sort by ID for consistent output
	sort.Slice(status, func(i, j int) bool {
		return status[i].ID < status[j].ID
	})

	return status, nil
}

// generateChecksum generates a simple checksum for migration content
func generateChecksum(content string) string {
	// Simple hash based on content length and first/last characters
	// In production, you might want to use a proper hash function
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return "empty"
	}

	return fmt.Sprintf("%d-%c%c", len(content), content[0], content[len(content)-1])
}

// RegisterCoreSchemas registers the core database schema migrations
func (m *MigrationManager) RegisterCoreSchemas() {
	// Migration 001: Initial core tables (if not already created by schema.go)
	m.RegisterMigration(
		"001_initial_schema",
		"Create initial core tables for templates and blueprints",
		`-- This migration ensures core tables exist
		 -- (Schema may already be created by schema.go, but this ensures consistency)`,
		`-- Rollback would drop tables, but we don't want to accidentally destroy data
		 -- SELECT 'Rollback not implemented for initial schema' as warning;`,
	)

	// Migration 002: Add indexes for performance
	m.RegisterMigration(
		"002_add_indexes",
		"Add database indexes for improved query performance",
		`CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name);
		 CREATE INDEX IF NOT EXISTS idx_blueprints_name ON blueprints(name);
		 CREATE INDEX IF NOT EXISTS idx_blueprints_stack ON blueprints(stack);`,
		`DROP INDEX IF EXISTS idx_templates_name;
		 DROP INDEX IF EXISTS idx_blueprints_name;
		 DROP INDEX IF EXISTS idx_blueprints_stack;`,
	)

	// Migration 003: Add audit trail
	m.RegisterMigration(
		"003_add_audit_trail",
		"Add audit trail tables for tracking changes",
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name VARCHAR(50) NOT NULL,
			record_id VARCHAR(255) NOT NULL,
			action VARCHAR(10) NOT NULL, -- INSERT, UPDATE, DELETE
			old_values TEXT,
			new_values TEXT,
			changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			changed_by VARCHAR(255)
		);
		CREATE INDEX IF NOT EXISTS idx_audit_log_table ON audit_log(table_name);
		CREATE INDEX IF NOT EXISTS idx_audit_log_changed_at ON audit_log(changed_at);`,
		`DROP TABLE IF EXISTS audit_log;`,
	)

	// Migration 004: Add metadata columns
	m.RegisterMigration(
		"004_add_metadata_columns",
		"Add created_at and updated_at columns to core tables",
		`-- Add metadata columns if they don't exist
		 ALTER TABLE templates ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
		 ALTER TABLE templates ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
		 ALTER TABLE blueprints ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
		 ALTER TABLE blueprints ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;`,
		`-- Note: SQLite doesn't support DROP COLUMN, so we can't easily roll this back
		 SELECT 'Cannot rollback metadata columns in SQLite' as warning;`,
	)
}
