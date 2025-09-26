package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_OpenAndClose(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	manager := NewManager()
	ctx := context.Background()

	// Test opening database
	err := manager.Open(ctx, dbPath)
	require.NoError(t, err)

	// Test that database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Test closing database
	err = manager.Close()
	assert.NoError(t, err)
}

func TestManager_OpenInvalidPath(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// Test opening database with invalid path
	err := manager.Open(ctx, "/invalid/path/test.db")
	assert.Error(t, err)
}

func TestManager_WithTx(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	manager := NewManager()
	ctx := context.Background()

	err := manager.Open(ctx, dbPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, manager.Close())
	}()

	// Test successful transaction
	err = manager.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO configs (scope, key, value) VALUES (?, ?, ?)", "test", "key1", "value1")
		return err
	})
	assert.NoError(t, err)

	// Test transaction rollback on error
	err = manager.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO configs (scope, key, value) VALUES (?, ?, ?)", "test", "key2", "value2")
		if err != nil {
			return err
		}
		// Force an error to test rollback
		return assert.AnError
	})
	assert.Error(t, err)
}

func TestManager_Migration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	manager := NewManager()
	ctx := context.Background()

	err := manager.Open(ctx, dbPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, manager.Close())
	}()

	// Test that all tables were created
	tables := []string{"templates", "blueprints", "configs", "hooks", "plugins", "audits"}
	
	for _, table := range tables {
		var count int
		err := manager.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should exist", table)
	}
}
