package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Manager handles database operations
type Manager struct {
	db   *sql.DB
	path string
}

// NewManager creates a new database manager
func NewManager() *Manager {
	return &Manager{}
}

// Open opens the database connection
func (m *Manager) Open(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	m.db = db
	m.path = path

	// Test connection
	if err := m.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := m.migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// WithTx executes a function within a transaction
func (m *Manager) WithTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// migrate runs database migrations
func (m *Manager) migrate(ctx context.Context) error {
	migrations := []string{
		createTemplatesTable,
		createBlueprintsTable,
		createConfigsTable,
		createHooksTable,
		createPluginsTable,
		createAuditsTable,
		createIndexes,
	}

	for i, migration := range migrations {
		if _, err := m.db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}
