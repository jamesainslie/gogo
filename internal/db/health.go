package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

// HealthManager handles database health monitoring and maintenance
type HealthManager struct {
	db   *Manager
	path string
}

// NewHealthManager creates a new health manager
func NewHealthManager(manager *Manager, dbPath string) *HealthManager {
	return &HealthManager{
		db:   manager,
		path: dbPath,
	}
}

// HealthStatus represents the overall health of the database
type HealthStatus struct {
	Status          string        `json:"status"`
	CheckedAt       time.Time     `json:"checked_at"`
	DatabasePath    string        `json:"database_path"`
	DatabaseSize    int64         `json:"database_size_bytes"`
	TableCount      int           `json:"table_count"`
	TotalRows       int           `json:"total_rows"`
	IntegrityOK     bool          `json:"integrity_ok"`
	WALMode         bool          `json:"wal_mode"`
	Version         string        `json:"sqlite_version"`
	Checks          []HealthCheck `json:"checks"`
	Recommendations []string      `json:"recommendations,omitempty"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // OK, WARNING, ERROR
	Message   string    `json:"message"`
	Value     string    `json:"value,omitempty"`
	Duration  string    `json:"duration"`
	CheckedAt time.Time `json:"checked_at"`
}

// DatabaseStats contains detailed database statistics
type DatabaseStats struct {
	TotalSize   int64        `json:"total_size_bytes"`
	DataSize    int64        `json:"data_size_bytes"`
	IndexSize   int64        `json:"index_size_bytes"`
	FreeSpace   int64        `json:"free_space_bytes"`
	PageCount   int64        `json:"page_count"`
	PageSize    int64        `json:"page_size"`
	Tables      []TableStats `json:"tables"`
	WALSize     int64        `json:"wal_size_bytes,omitempty"`
	JournalMode string       `json:"journal_mode"`
	CacheSize   int64        `json:"cache_size"`
	TempStore   string       `json:"temp_store"`
}

// TableStats contains statistics for individual tables
type TableStats struct {
	Name      string `json:"name"`
	RowCount  int64  `json:"row_count"`
	DataSize  int64  `json:"data_size_bytes"`
	IndexSize int64  `json:"index_size_bytes"`
}

// CheckHealth performs a comprehensive health check of the database
func (h *HealthManager) CheckHealth(ctx context.Context, verbose bool) (*HealthStatus, error) {
	if verbose {
		color.Yellow("Performing database health check...")
	}

	status := &HealthStatus{
		CheckedAt:    time.Now(),
		DatabasePath: h.path,
		Status:       "OK",
	}

	// Get database file size
	if stat, err := os.Stat(h.path); err == nil {
		status.DatabaseSize = stat.Size()
	}

	var checks []HealthCheck

	// Check 1: Database connectivity
	connectCheck := h.checkConnectivity(ctx)
	checks = append(checks, connectCheck)
	if connectCheck.Status == "ERROR" {
		status.Status = "ERROR"
		status.Checks = checks
		return status, nil
	}

	// Check 2: Database integrity
	integrityCheck := h.checkIntegrity(ctx)
	checks = append(checks, integrityCheck)
	status.IntegrityOK = integrityCheck.Status == "OK"
	if integrityCheck.Status == "ERROR" {
		status.Status = "ERROR"
	}

	// Check 3: SQLite version
	versionCheck := h.checkVersion(ctx)
	checks = append(checks, versionCheck)
	status.Version = versionCheck.Value

	// Check 4: Journal mode (WAL)
	walCheck := h.checkWALMode(ctx)
	checks = append(checks, walCheck)
	status.WALMode = walCheck.Value == "wal"

	// Check 5: Table counts
	tableCheck := h.checkTables(ctx)
	checks = append(checks, tableCheck)
	if count, err := parseIntValue(tableCheck.Value); err == nil {
		status.TableCount = int(count)
	}

	// Check 6: Row counts
	rowCheck := h.checkRowCounts(ctx)
	checks = append(checks, rowCheck)
	if count, err := parseIntValue(rowCheck.Value); err == nil {
		status.TotalRows = int(count)
	}

	// Check 7: Free space
	freeSpaceCheck := h.checkFreeSpace(ctx)
	checks = append(checks, freeSpaceCheck)

	// Check 8: Performance metrics
	perfCheck := h.checkPerformance(ctx)
	checks = append(checks, perfCheck)

	status.Checks = checks

	// Generate recommendations
	status.Recommendations = h.generateRecommendations(status)

	// Determine overall status
	for _, check := range checks {
		if check.Status == "ERROR" {
			status.Status = "ERROR"
			break
		} else if check.Status == "WARNING" && status.Status == "OK" {
			status.Status = "WARNING"
		}
	}

	if verbose {
		h.printHealthStatus(status)
	}

	return status, nil
}

// GetDatabaseStats returns detailed database statistics
func (h *HealthManager) GetDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Get basic database info
	if stat, err := os.Stat(h.path); err == nil {
		stats.TotalSize = stat.Size()
	}

	// Get SQLite-specific stats
	pragmas := map[string]*string{
		"page_count":   nil,
		"page_size":    nil,
		"journal_mode": &stats.JournalMode,
		"cache_size":   nil,
		"temp_store":   &stats.TempStore,
	}

	for pragma, target := range pragmas {
		var value string
		err := h.db.db.QueryRowContext(ctx, fmt.Sprintf("PRAGMA %s", pragma)).Scan(&value)
		if err != nil {
			continue
		}

		switch pragma {
		case "page_count":
			if count, err := parseIntValue(value); err == nil {
				stats.PageCount = count
			}
		case "page_size":
			if size, err := parseIntValue(value); err == nil {
				stats.PageSize = size
			}
		case "cache_size":
			if size, err := parseIntValue(value); err == nil {
				stats.CacheSize = size
			}
		case "journal_mode", "temp_store":
			if target != nil {
				*target = value
			}
		}
	}

	// Calculate derived stats
	if stats.PageCount > 0 && stats.PageSize > 0 {
		stats.DataSize = stats.PageCount * stats.PageSize
	}

	// Get table statistics
	tableStats, err := h.getTableStats(ctx)
	if err == nil {
		stats.Tables = tableStats
	}

	// Check for WAL file
	walPath := h.path + "-wal"
	if walStat, err := os.Stat(walPath); err == nil {
		stats.WALSize = walStat.Size()
	}

	return stats, nil
}

// VacuumDatabase performs database optimization
func (h *HealthManager) VacuumDatabase(ctx context.Context, verbose bool) error {
	if verbose {
		color.Yellow("Starting database vacuum...")
	}

	start := time.Now()

	// Get size before vacuum
	sizeBefore := int64(0)
	if stat, err := os.Stat(h.path); err == nil {
		sizeBefore = stat.Size()
	}

	// Perform vacuum
	if _, err := h.db.db.ExecContext(ctx, "VACUUM"); err != nil {
		return fmt.Errorf("vacuum failed: %w", err)
	}

	duration := time.Since(start)

	// Get size after vacuum
	sizeAfter := int64(0)
	if stat, err := os.Stat(h.path); err == nil {
		sizeAfter = stat.Size()
	}

	spaceReclaimed := sizeBefore - sizeAfter

	if verbose {
		color.Green("✓ Database vacuum completed in %v", duration)
		if spaceReclaimed > 0 {
			color.Green("✓ Reclaimed %.2f MB of space", float64(spaceReclaimed)/1024/1024)
		} else {
			color.Yellow("No space was reclaimed")
		}
	}

	return nil
}

// AnalyzeDatabase updates database statistics
func (h *HealthManager) AnalyzeDatabase(ctx context.Context, verbose bool) error {
	if verbose {
		color.Yellow("Analyzing database statistics...")
	}

	start := time.Now()

	if _, err := h.db.db.ExecContext(ctx, "ANALYZE"); err != nil {
		return fmt.Errorf("analyze failed: %w", err)
	}

	duration := time.Since(start)

	if verbose {
		color.Green("✓ Database analysis completed in %v", duration)
	}

	return nil
}

// Individual health check functions

func (h *HealthManager) checkConnectivity(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Database Connectivity",
		CheckedAt: start,
	}

	if err := h.db.db.PingContext(ctx); err != nil {
		check.Status = "ERROR"
		check.Message = fmt.Sprintf("Failed to connect to database: %v", err)
	} else {
		check.Status = "OK"
		check.Message = "Database connection successful"
	}

	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkIntegrity(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Database Integrity",
		CheckedAt: start,
	}

	var result string
	err := h.db.db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result)
	if err != nil {
		check.Status = "ERROR"
		check.Message = fmt.Sprintf("Integrity check failed: %v", err)
	} else if result == "ok" {
		check.Status = "OK"
		check.Message = "Database integrity verified"
		check.Value = "ok"
	} else {
		check.Status = "ERROR"
		check.Message = fmt.Sprintf("Integrity issues found: %s", result)
		check.Value = result
	}

	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkVersion(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "SQLite Version",
		CheckedAt: start,
	}

	var version string
	err := h.db.db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version)
	if err != nil {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Could not retrieve SQLite version: %v", err)
	} else {
		check.Status = "OK"
		check.Message = fmt.Sprintf("SQLite version: %s", version)
		check.Value = version
	}

	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkWALMode(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Journal Mode",
		CheckedAt: start,
	}

	var mode string
	err := h.db.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Could not retrieve journal mode: %v", err)
	} else {
		check.Value = mode
		if mode == "wal" {
			check.Status = "OK"
			check.Message = "WAL mode enabled (optimal for concurrency)"
		} else {
			check.Status = "WARNING"
			check.Message = fmt.Sprintf("Journal mode is %s (consider enabling WAL)", mode)
		}
	}

	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkTables(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Table Count",
		CheckedAt: start,
	}

	var count int
	err := h.db.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count)
	if err != nil {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Could not count tables: %v", err)
	} else {
		check.Status = "OK"
		check.Message = fmt.Sprintf("Database contains %d tables", count)
		check.Value = fmt.Sprintf("%d", count)
	}

	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkRowCounts(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Total Row Count",
		CheckedAt: start,
	}

	// Get all table names
	rows, err := h.db.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Could not retrieve table names: %v", err)
		check.Duration = time.Since(start).String()
		return check
	}
	defer rows.Close()

	totalRows := 0
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		var rowCount int
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := h.db.db.QueryRowContext(ctx, countQuery).Scan(&rowCount); err == nil {
			totalRows += rowCount
		}
	}

	check.Status = "OK"
	check.Message = fmt.Sprintf("Database contains %d total rows", totalRows)
	check.Value = fmt.Sprintf("%d", totalRows)
	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkFreeSpace(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Free Space",
		CheckedAt: start,
	}

	var freePages int
	err := h.db.db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&freePages)
	if err != nil {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Could not check free space: %v", err)
	} else {
		check.Status = "OK"
		if freePages > 100 {
			check.Status = "WARNING"
			check.Message = fmt.Sprintf("Database has %d free pages (consider VACUUM)", freePages)
		} else {
			check.Message = fmt.Sprintf("Database has %d free pages", freePages)
		}
		check.Value = fmt.Sprintf("%d", freePages)
	}

	check.Duration = time.Since(start).String()
	return check
}

func (h *HealthManager) checkPerformance(ctx context.Context) HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:      "Query Performance",
		CheckedAt: start,
	}

	// Simple performance test
	testStart := time.Now()
	var result int
	err := h.db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master").Scan(&result)
	queryDuration := time.Since(testStart)

	if err != nil {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Performance test failed: %v", err)
	} else if queryDuration > 100*time.Millisecond {
		check.Status = "WARNING"
		check.Message = fmt.Sprintf("Slow query performance: %v", queryDuration)
	} else {
		check.Status = "OK"
		check.Message = fmt.Sprintf("Query performance: %v", queryDuration)
	}
	check.Value = queryDuration.String()

	check.Duration = time.Since(start).String()
	return check
}

// Helper functions

func (h *HealthManager) getTableStats(ctx context.Context) ([]TableStats, error) {
	rows, err := h.db.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TableStats
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		tableStat := TableStats{Name: tableName}

		// Get row count
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := h.db.db.QueryRowContext(ctx, countQuery).Scan(&tableStat.RowCount); err != nil {
			tableStat.RowCount = -1 // Indicate error
		}

		stats = append(stats, tableStat)
	}

	return stats, nil
}

func (h *HealthManager) generateRecommendations(status *HealthStatus) []string {
	var recommendations []string

	if !status.WALMode {
		recommendations = append(recommendations, "Enable WAL mode for better concurrency: PRAGMA journal_mode=WAL")
	}

	// Check for free space issues
	for _, check := range status.Checks {
		if check.Name == "Free Space" && check.Status == "WARNING" {
			recommendations = append(recommendations, "Run VACUUM to reclaim free space and optimize database")
		}
	}

	if status.DatabaseSize > 100*1024*1024 { // > 100MB
		recommendations = append(recommendations, "Large database detected - consider regular ANALYZE for query optimization")
	}

	if status.TotalRows > 10000 {
		recommendations = append(recommendations, "High row count - ensure proper indexes are in place for frequently queried columns")
	}

	return recommendations
}

func (h *HealthManager) printHealthStatus(status *HealthStatus) {
	color.Yellow("=== Database Health Report ===")
	fmt.Printf("Status: %s\n", colorizeStatus(status.Status))
	fmt.Printf("Database: %s\n", status.DatabasePath)
	fmt.Printf("Size: %.2f MB\n", float64(status.DatabaseSize)/1024/1024)
	fmt.Printf("Tables: %d\n", status.TableCount)
	fmt.Printf("Total Rows: %d\n", status.TotalRows)
	fmt.Printf("Integrity: %s\n", colorizeBoolean(status.IntegrityOK))
	fmt.Printf("WAL Mode: %s\n", colorizeBoolean(status.WALMode))
	fmt.Printf("SQLite Version: %s\n", status.Version)
	fmt.Println()

	color.Yellow("=== Health Checks ===")
	for _, check := range status.Checks {
		fmt.Printf("%-20s %s %s\n", check.Name+":", colorizeStatus(check.Status), check.Message)
	}

	if len(status.Recommendations) > 0 {
		fmt.Println()
		color.Yellow("=== Recommendations ===")
		for _, rec := range status.Recommendations {
			fmt.Printf("• %s\n", rec)
		}
	}
}

func colorizeStatus(status string) string {
	switch status {
	case "OK":
		return color.GreenString("✓ %s", status)
	case "WARNING":
		return color.YellowString("⚠ %s", status)
	case "ERROR":
		return color.RedString("✗ %s", status)
	default:
		return status
	}
}

func colorizeBoolean(value bool) string {
	if value {
		return color.GreenString("✓ Yes")
	}
	return color.RedString("✗ No")
}

func parseIntValue(value string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err
}
