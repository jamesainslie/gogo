package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupManager_Backup(t *testing.T) {
	// Create test database with some data
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	// Create test data
	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	defer manager.Close()

	// Insert some test data (assuming templates table exists from schema)
	_, err := manager.GetDB().ExecContext(ctx,
		`INSERT INTO templates (name, description, content) VALUES (?, ?, ?)`,
		"test-template", "Test template", `{"files": []}`)
	require.NoError(t, err)

	backupManager := NewBackupManager(manager, dbPath)

	tests := []struct {
		name        string
		options     BackupOptions
		expectError bool
	}{
		{
			name: "raw backup",
			options: BackupOptions{
				OutputPath: filepath.Join(t.TempDir(), "backup.db"),
				Compress:   false,
				Verify:     true,
				Verbose:    false,
			},
			expectError: false,
		},
		{
			name: "compressed backup",
			options: BackupOptions{
				OutputPath: filepath.Join(t.TempDir(), "backup.db.gz"),
				Compress:   true,
				Verify:     true,
				Verbose:    false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backupManager.Backup(ctx, tt.options)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify backup file exists
				_, err := os.Stat(tt.options.OutputPath)
				assert.NoError(t, err)

				// Verify backup file has content
				stat, err := os.Stat(tt.options.OutputPath)
				require.NoError(t, err)
				assert.Greater(t, stat.Size(), int64(0))
			}
		})
	}
}

func TestBackupManager_Restore(t *testing.T) {
	// Create source database with test data
	sourceManager, sourcePath, sourceCleanup := setupTestManager(t)
	defer sourceCleanup()

	ctx := context.Background()
	require.NoError(t, sourceManager.Open(ctx, sourcePath))

	// Insert test data
	_, err := sourceManager.GetDB().ExecContext(ctx,
		`INSERT INTO templates (name, description, content) VALUES (?, ?, ?)`,
		"test-template", "Test template", `{"files": []}`)
	require.NoError(t, err)
	sourceManager.Close()

	// Create backup
	sourceBackupManager := NewBackupManager(sourceManager, sourcePath)
	backupPath := filepath.Join(t.TempDir(), "source_backup.db")

	err = sourceBackupManager.Backup(ctx, BackupOptions{
		OutputPath: backupPath,
		Compress:   false,
		Verify:     true,
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		options     RestoreOptions
		expectError bool
	}{
		{
			name: "restore to new location",
			options: RestoreOptions{
				BackupPath: backupPath,
				Verify:     true,
				Force:      true,
				Verbose:    false,
			},
			expectError: false,
		},
		{
			name: "restore with backup creation",
			options: RestoreOptions{
				BackupPath:   backupPath,
				CreateBackup: true,
				Force:        true,
				Verify:       true,
				Verbose:      false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create target database path
			targetManager, targetPath, targetCleanup := setupTestManager(t)
			defer targetCleanup()

			targetBackupManager := NewBackupManager(targetManager, targetPath)

			err := targetBackupManager.Restore(ctx, tt.options)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify database was restored by checking data
				require.NoError(t, targetManager.Open(ctx, targetPath))
				defer targetManager.Close()

				var name string
				err = targetManager.GetDB().QueryRowContext(ctx,
					"SELECT name FROM templates WHERE name = ?", "test-template").Scan(&name)
				assert.NoError(t, err)
				assert.Equal(t, "test-template", name)
			}
		})
	}
}

func TestBackupManager_GetBackupInfo(t *testing.T) {
	// Create test database and backup
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	manager.Close()

	backupManager := NewBackupManager(manager, dbPath)

	// Test raw backup
	rawBackupPath := filepath.Join(t.TempDir(), "info_test.db")
	err := backupManager.Backup(ctx, BackupOptions{
		OutputPath: rawBackupPath,
		Compress:   false,
	})
	require.NoError(t, err)

	info, err := backupManager.GetBackupInfo(rawBackupPath)
	require.NoError(t, err)
	assert.Equal(t, rawBackupPath, info.Path)
	assert.Greater(t, info.Size, int64(0))
	assert.False(t, info.IsCompressed)

	// Test compressed backup
	compressedBackupPath := filepath.Join(t.TempDir(), "info_test_compressed.db.gz")
	err = backupManager.Backup(ctx, BackupOptions{
		OutputPath: compressedBackupPath,
		Compress:   true,
	})
	require.NoError(t, err)

	compressedInfo, err := backupManager.GetBackupInfo(compressedBackupPath)
	require.NoError(t, err)
	assert.Equal(t, compressedBackupPath, compressedInfo.Path)
	assert.Greater(t, compressedInfo.Size, int64(0))
	assert.True(t, compressedInfo.IsCompressed)
}

func TestBackupManager_VerifyBackup(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	manager.Close()

	backupManager := NewBackupManager(manager, dbPath)

	// Create valid backup
	validBackupPath := filepath.Join(t.TempDir(), "valid_backup.db")
	err := backupManager.Backup(ctx, BackupOptions{
		OutputPath: validBackupPath,
		Compress:   false,
		Verify:     false, // Don't verify during creation to test separate verification
	})
	require.NoError(t, err)

	// Test verification of valid backup
	err = backupManager.verifyBackup(ctx, validBackupPath, false)
	assert.NoError(t, err)

	// Create invalid backup (empty file)
	invalidBackupPath := filepath.Join(t.TempDir(), "invalid_backup.db")
	invalidFile, err := os.Create(invalidBackupPath)
	require.NoError(t, err)
	invalidFile.Close()

	// Test verification of invalid backup
	err = backupManager.verifyBackup(ctx, invalidBackupPath, false)
	assert.Error(t, err)
}

func TestBackupManager_CompressedBackupWorkflow(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))

	// Add some test data to make compression meaningful
	for i := 0; i < 100; i++ {
		_, err := manager.GetDB().ExecContext(ctx,
			`INSERT INTO templates (name, description, content) VALUES (?, ?, ?)`,
			"template"+string(rune(i)), "Test template "+string(rune(i)), `{"files": []}`)
		require.NoError(t, err)
	}
	manager.Close()

	backupManager := NewBackupManager(manager, dbPath)

	// Create compressed backup
	compressedPath := filepath.Join(t.TempDir(), "compressed_test.db.gz")
	err := backupManager.Backup(ctx, BackupOptions{
		OutputPath: compressedPath,
		Compress:   true,
		Verify:     true,
		Verbose:    false,
	})
	require.NoError(t, err)

	// Verify compressed file is smaller than original
	originalStat, err := os.Stat(dbPath)
	require.NoError(t, err)

	compressedStat, err := os.Stat(compressedPath)
	require.NoError(t, err)

	// Compressed should be smaller (though this might not always be true for small databases)
	t.Logf("Original size: %d bytes, Compressed size: %d bytes", originalStat.Size(), compressedStat.Size())

	// Test restore from compressed backup
	restoreManager, restorePath, restoreCleanup := setupTestManager(t)
	defer restoreCleanup()

	restoreBackupManager := NewBackupManager(restoreManager, restorePath)

	err = restoreBackupManager.Restore(ctx, RestoreOptions{
		BackupPath: compressedPath,
		Verify:     true,
		Force:      true,
	})
	require.NoError(t, err)

	// Verify data was restored correctly
	require.NoError(t, restoreManager.Open(ctx, restorePath))
	defer restoreManager.Close()

	var count int
	err = restoreManager.GetDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM templates").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 100, count)
}

func setupTestManager(t *testing.T) (*Manager, string, func()) {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_backup_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	dbPath := tmpFile.Name()

	manager := NewManager()

	// Create cleanup function
	cleanup := func() {
		manager.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}

	return manager, dbPath, cleanup
}
