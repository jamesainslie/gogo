package db

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthManager_CheckHealth(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	defer manager.Close()

	healthManager := NewHealthManager(manager, dbPath)

	tests := []struct {
		name    string
		verbose bool
	}{
		{
			name:    "health check verbose",
			verbose: true,
		},
		{
			name:    "health check quiet",
			verbose: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := healthManager.CheckHealth(ctx, tt.verbose)
			require.NoError(t, err)

			assert.NotEmpty(t, status.Status)
			assert.NotZero(t, status.CheckedAt)
			assert.Equal(t, dbPath, status.DatabasePath)
			assert.Greater(t, status.DatabaseSize, int64(0))

			// Should have multiple health checks
			assert.Greater(t, len(status.Checks), 5)

			// Check that critical checks exist
			checkNames := make(map[string]bool)
			for _, check := range status.Checks {
				checkNames[check.Name] = true
				assert.NotEmpty(t, check.Status)
				assert.NotEmpty(t, check.Message)
				assert.NotZero(t, check.CheckedAt)
			}

			expectedChecks := []string{
				"Database Connectivity",
				"Database Integrity",
				"SQLite Version",
				"Journal Mode",
				"Table Count",
				"Total Row Count",
				"Free Space",
				"Query Performance",
			}

			for _, expectedCheck := range expectedChecks {
				assert.True(t, checkNames[expectedCheck], "Expected check '%s' not found", expectedCheck)
			}
		})
	}
}

func TestHealthManager_GetDatabaseStats(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))

	// Add some test data
	_, err := manager.GetDB().ExecContext(ctx,
		`INSERT INTO templates (name, description, content) VALUES (?, ?, ?)`,
		"test-template", "Test template", `{"files": []}`)
	require.NoError(t, err)

	manager.Close()

	healthManager := NewHealthManager(manager, dbPath)
	require.NoError(t, manager.Open(ctx, dbPath))
	defer manager.Close()

	stats, err := healthManager.GetDatabaseStats(ctx)
	require.NoError(t, err)

	assert.Greater(t, stats.TotalSize, int64(0))
	assert.Greater(t, stats.PageCount, int64(0))
	assert.Greater(t, stats.PageSize, int64(0))
	assert.NotEmpty(t, stats.JournalMode)

	// Should have at least the core tables
	assert.Greater(t, len(stats.Tables), 0)

	// Check that table stats have reasonable values
	for _, table := range stats.Tables {
		assert.NotEmpty(t, table.Name)
		assert.GreaterOrEqual(t, table.RowCount, int64(0))
	}
}

func TestHealthManager_VacuumDatabase(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))

	// Add and then delete data to create fragmentation
	for i := 0; i < 100; i++ {
		_, err := manager.GetDB().ExecContext(ctx,
			`INSERT INTO templates (name, description, content) VALUES (?, ?, ?)`,
			"temp-template-"+string(rune(i)), "Temporary template", `{"files": []}`)
		require.NoError(t, err)
	}

	// Delete half the data
	_, err := manager.GetDB().ExecContext(ctx,
		`DELETE FROM templates WHERE name LIKE 'temp-template-%' AND rowid % 2 = 0`)
	require.NoError(t, err)

	healthManager := NewHealthManager(manager, dbPath)

	// Get size before vacuum
	statBefore, err := os.Stat(dbPath)
	require.NoError(t, err)

	// Run vacuum
	err = healthManager.VacuumDatabase(ctx, false) // Not verbose to avoid output in tests
	require.NoError(t, err)

	// Verify database still works after vacuum
	var count int
	err = manager.GetDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM templates").Scan(&count)
	require.NoError(t, err)
	assert.Greater(t, count, 0) // Should still have some data

	// Check that vacuum didn't corrupt the database
	var integrityResult string
	err = manager.GetDB().QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrityResult)
	require.NoError(t, err)
	assert.Equal(t, "ok", integrityResult)

	manager.Close()

	// Size may or may not be smaller depending on SQLite version and data
	statAfter, err := os.Stat(dbPath)
	require.NoError(t, err)

	t.Logf("Database size before vacuum: %d bytes", statBefore.Size())
	t.Logf("Database size after vacuum: %d bytes", statAfter.Size())
}

func TestHealthManager_AnalyzeDatabase(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	defer manager.Close()

	// Add some test data
	for i := 0; i < 50; i++ {
		_, err := manager.GetDB().ExecContext(ctx,
			`INSERT INTO templates (name, description, content) VALUES (?, ?, ?)`,
			"analyze-template-"+string(rune(i)), "Template for analysis", `{"files": []}`)
		require.NoError(t, err)
	}

	healthManager := NewHealthManager(manager, dbPath)

	// Run analyze
	err := healthManager.AnalyzeDatabase(ctx, false) // Not verbose to avoid output
	require.NoError(t, err)

	// Verify database still works after analyze
	var count int
	err = manager.GetDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM templates").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 50, count)

	// Check that analyze updated statistics (this is hard to verify directly,
	// but we can at least verify the command didn't break anything)
	var integrityResult string
	err = manager.GetDB().QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrityResult)
	require.NoError(t, err)
	assert.Equal(t, "ok", integrityResult)
}

func TestHealthManager_IndividualChecks(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	defer manager.Close()

	healthManager := NewHealthManager(manager, dbPath)

	// Test connectivity check
	connectCheck := healthManager.checkConnectivity(ctx)
	assert.Equal(t, "Database Connectivity", connectCheck.Name)
	assert.Equal(t, "OK", connectCheck.Status)
	assert.Contains(t, connectCheck.Message, "successful")

	// Test integrity check
	integrityCheck := healthManager.checkIntegrity(ctx)
	assert.Equal(t, "Database Integrity", integrityCheck.Name)
	assert.Equal(t, "OK", integrityCheck.Status)
	assert.Equal(t, "ok", integrityCheck.Value)

	// Test version check
	versionCheck := healthManager.checkVersion(ctx)
	assert.Equal(t, "SQLite Version", versionCheck.Name)
	assert.Equal(t, "OK", versionCheck.Status)
	assert.NotEmpty(t, versionCheck.Value)

	// Test WAL mode check
	walCheck := healthManager.checkWALMode(ctx)
	assert.Equal(t, "Journal Mode", walCheck.Name)
	assert.NotEmpty(t, walCheck.Status)
	assert.NotEmpty(t, walCheck.Value)

	// Test tables check
	tableCheck := healthManager.checkTables(ctx)
	assert.Equal(t, "Table Count", tableCheck.Name)
	assert.Equal(t, "OK", tableCheck.Status)
	assert.NotEmpty(t, tableCheck.Value)

	// Test performance check
	perfCheck := healthManager.checkPerformance(ctx)
	assert.Equal(t, "Query Performance", perfCheck.Name)
	assert.NotEmpty(t, perfCheck.Status)
	assert.NotEmpty(t, perfCheck.Value)
}

func TestHealthManager_GenerateRecommendations(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, manager.Open(ctx, dbPath))
	defer manager.Close()

	healthManager := NewHealthManager(manager, dbPath)

	// Create a status with various conditions that should trigger recommendations
	status := &HealthStatus{
		DatabaseSize: 150 * 1024 * 1024, // 150MB - should trigger large DB recommendation
		TotalRows:    15000,             // High row count - should trigger index recommendation
		WALMode:      false,             // Should trigger WAL recommendation
		Checks: []HealthCheck{
			{
				Name:   "Free Space",
				Status: "WARNING", // Should trigger vacuum recommendation
			},
		},
	}

	recommendations := healthManager.generateRecommendations(status)

	assert.Greater(t, len(recommendations), 0)

	// Check for expected recommendations
	recText := ""
	for _, rec := range recommendations {
		recText += rec + " "
	}

	assert.Contains(t, recText, "WAL mode") // Should recommend WAL mode
	assert.Contains(t, recText, "VACUUM")   // Should recommend vacuum
	assert.Contains(t, recText, "ANALYZE")  // Should recommend analyze for large DB
	assert.Contains(t, recText, "indexes")  // Should recommend indexes for high row count
}

func TestHealthManager_PrintHealthStatus(t *testing.T) {
	manager, dbPath, cleanup := setupTestManager(t)
	defer cleanup()

	healthManager := NewHealthManager(manager, dbPath)

	// Create a test status
	status := &HealthStatus{
		Status:       "OK",
		DatabasePath: dbPath,
		DatabaseSize: 1024 * 1024, // 1MB
		TableCount:   5,
		TotalRows:    100,
		IntegrityOK:  true,
		WALMode:      true,
		Version:      "3.39.0",
		Checks: []HealthCheck{
			{Name: "Test Check", Status: "OK", Message: "All good"},
		},
		Recommendations: []string{"No recommendations"},
	}

	// This mainly tests that the function doesn't panic
	// In a real scenario, you'd capture output to verify formatting
	healthManager.printHealthStatus(status)
}

func TestHealthManager_ColorizeHelpers(t *testing.T) {
	// Test status colorization
	okStatus := colorizeStatus("OK")
	assert.Contains(t, okStatus, "OK")

	warningStatus := colorizeStatus("WARNING")
	assert.Contains(t, warningStatus, "WARNING")

	errorStatus := colorizeStatus("ERROR")
	assert.Contains(t, errorStatus, "ERROR")

	// Test boolean colorization
	trueValue := colorizeBoolean(true)
	assert.Contains(t, trueValue, "Yes")

	falseValue := colorizeBoolean(false)
	assert.Contains(t, falseValue, "No")
}

func TestHealthManager_ParseIntValue(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"-456", -456, false},
		{"abc", 0, true},
		{"", 0, true},
		{"123.45", 123, false}, // Should parse the integer part
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseIntValue(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
