package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationManager_InitMigrationTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)

	err := migrationManager.InitMigrationTable(context.Background())
	require.NoError(t, err)

	// Verify table was created
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "schema_migrations", name)
}

func TestMigrationManager_RegisterMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)

	migrationManager.RegisterMigration("001_test", "Test migration", "CREATE TABLE test (id INTEGER)", "DROP TABLE test")

	// Check migration was registered
	migration, exists := migrationManager.migrations["001_test"]
	require.True(t, exists)
	assert.Equal(t, "001_test", migration.ID)
	assert.Equal(t, "Test migration", migration.Description)
	assert.Equal(t, "CREATE TABLE test (id INTEGER)", migration.UpSQL)
	assert.Equal(t, "DROP TABLE test", migration.DownSQL)
}

func TestMigrationManager_ApplyMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)
	require.NoError(t, migrationManager.InitMigrationTable(context.Background()))

	migration := &Migration{
		ID:          "001_test_table",
		Description: "Create test table",
		UpSQL:       "CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)",
		DownSQL:     "DROP TABLE test_table",
	}

	err := migrationManager.ApplyMigration(context.Background(), migration)
	require.NoError(t, err)

	// Verify table was created
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "test_table", name)

	// Verify migration was recorded
	var recordedID, recordedDesc string
	var appliedAt time.Time
	err = db.QueryRow("SELECT id, description, applied_at FROM schema_migrations WHERE id=?", "001_test_table").Scan(&recordedID, &recordedDesc, &appliedAt)
	require.NoError(t, err)
	assert.Equal(t, "001_test_table", recordedID)
	assert.Equal(t, "Create test table", recordedDesc)
	assert.WithinDuration(t, time.Now(), appliedAt, time.Minute)
}

func TestMigrationManager_RollbackMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)
	require.NoError(t, migrationManager.InitMigrationTable(context.Background()))

	migration := &Migration{
		ID:          "001_test_rollback",
		Description: "Create rollback test table",
		UpSQL:       "CREATE TABLE rollback_test (id INTEGER PRIMARY KEY)",
		DownSQL:     "DROP TABLE rollback_test",
	}

	// Apply migration first
	err := migrationManager.ApplyMigration(context.Background(), migration)
	require.NoError(t, err)

	// Verify table exists
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='rollback_test'").Scan(&name)
	require.NoError(t, err)

	// Rollback migration
	err = migrationManager.RollbackMigration(context.Background(), migration)
	require.NoError(t, err)

	// Verify table was dropped
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='rollback_test'").Scan(&name)
	assert.Equal(t, sql.ErrNoRows, err)

	// Verify migration record was removed
	var recordedID string
	err = db.QueryRow("SELECT id FROM schema_migrations WHERE id=?", "001_test_rollback").Scan(&recordedID)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestMigrationManager_GetPendingMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)
	require.NoError(t, migrationManager.InitMigrationTable(context.Background()))

	// Register some migrations
	migrationManager.RegisterMigration("001_first", "First migration", "CREATE TABLE first (id INTEGER)", "DROP TABLE first")
	migrationManager.RegisterMigration("002_second", "Second migration", "CREATE TABLE second (id INTEGER)", "DROP TABLE second")
	migrationManager.RegisterMigration("003_third", "Third migration", "CREATE TABLE third (id INTEGER)", "DROP TABLE third")

	// Apply first migration
	migration := &Migration{
		ID:      "001_first",
		UpSQL:   "CREATE TABLE first (id INTEGER)",
		DownSQL: "DROP TABLE first",
	}
	err := migrationManager.ApplyMigration(context.Background(), migration)
	require.NoError(t, err)

	// Get pending migrations
	pending, err := migrationManager.GetPendingMigrations(context.Background())
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	// Should be sorted by ID
	assert.Equal(t, "002_second", pending[0].ID)
	assert.Equal(t, "003_third", pending[1].ID)
}

func TestMigrationManager_ApplyAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)
	require.NoError(t, migrationManager.InitMigrationTable(context.Background()))

	// Register migrations
	migrationManager.RegisterMigration("001_users", "Create users table",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"DROP TABLE users")
	migrationManager.RegisterMigration("002_posts", "Create posts table",
		"CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, title TEXT)",
		"DROP TABLE posts")

	err := migrationManager.ApplyAll(context.Background())
	require.NoError(t, err)

	// Verify both tables were created
	var userTable, postTable string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&userTable)
	require.NoError(t, err)
	assert.Equal(t, "users", userTable)

	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&postTable)
	require.NoError(t, err)
	assert.Equal(t, "posts", postTable)

	// Verify both migrations were recorded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMigrationManager_GetMigrationStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)
	require.NoError(t, migrationManager.InitMigrationTable(context.Background()))

	// Register migrations
	migrationManager.RegisterMigration("001_applied", "Applied migration", "CREATE TABLE applied (id INTEGER)", "DROP TABLE applied")
	migrationManager.RegisterMigration("002_pending", "Pending migration", "CREATE TABLE pending (id INTEGER)", "DROP TABLE pending")

	// Apply first migration
	migration := &Migration{
		ID:      "001_applied",
		UpSQL:   "CREATE TABLE applied (id INTEGER)",
		DownSQL: "DROP TABLE applied",
	}
	err := migrationManager.ApplyMigration(context.Background(), migration)
	require.NoError(t, err)

	// Get status
	status, err := migrationManager.GetMigrationStatus(context.Background())
	require.NoError(t, err)
	assert.Len(t, status, 2)

	// Check status of migrations
	for _, migration := range status {
		switch migration.ID {
		case "001_applied":
			assert.True(t, migration.Applied)
			assert.NotNil(t, migration.AppliedAt)
		case "002_pending":
			assert.False(t, migration.Applied)
			assert.Nil(t, migration.AppliedAt)
		}
	}
}

func TestMigrationManager_RegisterCoreSchemas(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrationManager := NewMigrationManager(db)
	migrationManager.RegisterCoreSchemas()

	// Check that core migrations were registered
	expectedMigrations := []string{
		"001_initial_schema",
		"002_add_indexes",
		"003_add_audit_trail",
		"004_add_metadata_columns",
	}

	for _, expectedID := range expectedMigrations {
		migration, exists := migrationManager.migrations[expectedID]
		assert.True(t, exists, "Migration %s should be registered", expectedID)
		assert.NotEmpty(t, migration.Description, "Migration %s should have a description", expectedID)
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_migrations_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	// Clean up when test finishes
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	// Open database
	db, err := sql.Open("sqlite3", tmpFile.Name()+"?_journal_mode=WAL")
	require.NoError(t, err)

	return db
}
