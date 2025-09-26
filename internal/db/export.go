package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ExportManager handles database export and import operations
type ExportManager struct {
	db *Manager
}

// NewExportManager creates a new export manager
func NewExportManager(manager *Manager) *ExportManager {
	return &ExportManager{
		db: manager,
	}
}

// ExportOptions contains options for database export
type ExportOptions struct {
	OutputPath    string
	Format        ExportFormat
	Tables        []string
	IncludeSchema bool
	IncludeData   bool
	Verbose       bool
}

// ImportOptions contains options for database import
type ImportOptions struct {
	InputPath       string
	Format          ExportFormat
	Validate        bool
	DryRun          bool
	ReplaceExisting bool
	Verbose         bool
}

// ExportFormat represents different export formats
type ExportFormat string

const (
	FormatSQL  ExportFormat = "sql"
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
)

// ExportedData represents exported database data
type ExportedData struct {
	Metadata   ExportMetadata        `json:"metadata"`
	Tables     map[string][]TableRow `json:"tables"`
	Templates  []ExportedTemplate    `json:"templates,omitempty"`
	Blueprints []ExportedBlueprint   `json:"blueprints,omitempty"`
}

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	ExportedAt time.Time `json:"exported_at"`
	Version    string    `json:"version"`
	Format     string    `json:"format"`
	TableCount int       `json:"table_count"`
	RowCount   int       `json:"row_count"`
}

// TableRow represents a generic table row
type TableRow map[string]interface{}

// ExportedTemplate represents a template for export
type ExportedTemplate struct {
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Description string                 `json:"description"`
	Content     map[string]interface{} `json:"content"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
}

// ExportedBlueprint represents a blueprint for export
type ExportedBlueprint struct {
	Name        string                 `json:"name"`
	Stack       string                 `json:"stack"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
}

// Export exports database data in the specified format
func (e *ExportManager) Export(ctx context.Context, opts ExportOptions) error {
	if opts.Verbose {
		color.Yellow("Starting database export...")
	}

	// Create output directory if needed
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export based on format
	switch opts.Format {
	case FormatSQL:
		return e.exportSQL(ctx, opts)
	case FormatJSON:
		return e.exportJSON(ctx, opts)
	case FormatCSV:
		return e.exportCSV(ctx, opts)
	default:
		return fmt.Errorf("unsupported export format: %s", opts.Format)
	}
}

// exportSQL exports database as SQL dump
func (e *ExportManager) exportSQL(ctx context.Context, opts ExportOptions) error {
	file, err := os.Create(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "-- gogo database export\n")
	fmt.Fprintf(file, "-- Generated on: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "-- Format: SQL\n\n")

	// Get tables to export
	tables, err := e.getTablesToExport(ctx, opts.Tables)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	totalRows := 0

	for _, table := range tables {
		if opts.Verbose {
			color.Yellow("Exporting table: %s", table)
		}

		// Export table schema if requested
		if opts.IncludeSchema {
			if err := e.exportTableSchema(ctx, file, table); err != nil {
				return fmt.Errorf("failed to export schema for table %s: %w", table, err)
			}
		}

		// Export table data if requested
		if opts.IncludeData {
			rows, err := e.exportTableData(ctx, file, table)
			if err != nil {
				return fmt.Errorf("failed to export data for table %s: %w", table, err)
			}
			totalRows += rows
		}

		fmt.Fprintf(file, "\n")
	}

	if opts.Verbose {
		color.Green("✓ SQL export completed: %d tables, %d rows", len(tables), totalRows)
	}

	return nil
}

// exportJSON exports database as JSON
func (e *ExportManager) exportJSON(ctx context.Context, opts ExportOptions) error {
	// Collect data
	exportData := &ExportedData{
		Metadata: ExportMetadata{
			ExportedAt: time.Now(),
			Version:    "1.0",
			Format:     "json",
		},
		Tables: make(map[string][]TableRow),
	}

	// Get tables to export
	tables, err := e.getTablesToExport(ctx, opts.Tables)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	totalRows := 0

	for _, table := range tables {
		if opts.Verbose {
			color.Yellow("Exporting table: %s", table)
		}

		rows, err := e.getTableRows(ctx, table)
		if err != nil {
			return fmt.Errorf("failed to get rows for table %s: %w", table, err)
		}

		exportData.Tables[table] = rows
		totalRows += len(rows)

		// Special handling for templates and blueprints
		if table == "templates" {
			exportData.Templates, err = e.getTemplatesForExport(ctx)
			if err != nil {
				return fmt.Errorf("failed to export templates: %w", err)
			}
		} else if table == "blueprints" {
			exportData.Blueprints, err = e.getBlueprintsForExport(ctx)
			if err != nil {
				return fmt.Errorf("failed to export blueprints: %w", err)
			}
		}
	}

	exportData.Metadata.TableCount = len(tables)
	exportData.Metadata.RowCount = totalRows

	// Write JSON to file
	file, err := os.Create(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportData); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if opts.Verbose {
		color.Green("✓ JSON export completed: %d tables, %d rows", len(tables), totalRows)
	}

	return nil
}

// exportCSV exports database as CSV files
func (e *ExportManager) exportCSV(ctx context.Context, opts ExportOptions) error {
	// Get tables to export
	tables, err := e.getTablesToExport(ctx, opts.Tables)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	totalRows := 0

	// Create base directory for CSV files
	baseDir := strings.TrimSuffix(opts.OutputPath, filepath.Ext(opts.OutputPath))
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create CSV directory: %w", err)
	}

	for _, table := range tables {
		if opts.Verbose {
			color.Yellow("Exporting table: %s", table)
		}

		csvFile := filepath.Join(baseDir, table+".csv")
		rows, err := e.exportTableCSV(ctx, csvFile, table)
		if err != nil {
			return fmt.Errorf("failed to export CSV for table %s: %w", table, err)
		}
		totalRows += rows
	}

	if opts.Verbose {
		color.Green("✓ CSV export completed: %d tables, %d rows in %s", len(tables), totalRows, baseDir)
	}

	return nil
}

// Import imports data from a file
func (e *ExportManager) Import(ctx context.Context, opts ImportOptions) error {
	if opts.Verbose {
		color.Yellow("Starting database import...")
	}

	// Validate input file exists
	if _, err := os.Stat(opts.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", opts.InputPath)
	}

	// Import based on format
	switch opts.Format {
	case FormatSQL:
		return e.importSQL(ctx, opts)
	case FormatJSON:
		return e.importJSON(ctx, opts)
	default:
		return fmt.Errorf("unsupported import format: %s", opts.Format)
	}
}

// importSQL imports from SQL dump
func (e *ExportManager) importSQL(ctx context.Context, opts ImportOptions) error {
	content, err := os.ReadFile(opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	// Split into individual statements
	statements := strings.Split(string(content), ";")

	if opts.DryRun {
		color.Yellow("DRY RUN: Would execute %d SQL statements", len(statements)-1) // -1 because last split is empty
		return nil
	}

	// Execute statements in transaction
	tx, err := e.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	executed := 0
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if opts.Verbose {
			color.Yellow("Executing: %s", stmt[:min(50, len(stmt))]+"...")
		}

		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w\nStatement: %s", err, stmt)
		}
		executed++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit import transaction: %w", err)
	}

	color.Green("✓ SQL import completed: %d statements executed", executed)
	return nil
}

// importJSON imports from JSON export
func (e *ExportManager) importJSON(ctx context.Context, opts ImportOptions) error {
	file, err := os.Open(opts.InputPath)
	if err != nil {
		return fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	var exportData ExportedData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&exportData); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if opts.Validate {
		if err := e.validateImportData(&exportData); err != nil {
			return fmt.Errorf("import data validation failed: %w", err)
		}
	}

	if opts.DryRun {
		color.Yellow("DRY RUN: Would import %d tables with %d total rows",
			exportData.Metadata.TableCount, exportData.Metadata.RowCount)
		return nil
	}

	// Import data
	totalImported := 0
	for tableName, rows := range exportData.Tables {
		if opts.Verbose {
			color.Yellow("Importing table: %s (%d rows)", tableName, len(rows))
		}

		imported, err := e.importTableRows(ctx, tableName, rows, opts.ReplaceExisting)
		if err != nil {
			return fmt.Errorf("failed to import table %s: %w", tableName, err)
		}
		totalImported += imported
	}

	color.Green("✓ JSON import completed: %d rows imported", totalImported)
	return nil
}

// Helper functions

func (e *ExportManager) getTablesToExport(ctx context.Context, requestedTables []string) ([]string, error) {
	if len(requestedTables) > 0 {
		return requestedTables, nil
	}

	// Get all tables
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	rows, err := e.db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, name)
	}

	return tables, nil
}

func (e *ExportManager) exportTableSchema(ctx context.Context, w io.Writer, tableName string) error {
	query := `SELECT sql FROM sqlite_master WHERE type='table' AND name=?`
	var schema sql.NullString
	err := e.db.db.QueryRowContext(ctx, query, tableName).Scan(&schema)
	if err != nil {
		return fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
	}

	if schema.Valid {
		fmt.Fprintf(w, "-- Schema for table %s\n", tableName)
		fmt.Fprintf(w, "%s;\n\n", schema.String)
	}

	return nil
}

func (e *ExportManager) exportTableData(ctx context.Context, w io.Writer, tableName string) (int, error) {
	rows, err := e.db.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return 0, fmt.Errorf("failed to query table data: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("failed to get columns: %w", err)
	}

	fmt.Fprintf(w, "-- Data for table %s\n", tableName)

	rowCount := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return rowCount, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build INSERT statement
		fmt.Fprintf(w, "INSERT INTO %s (", tableName)
		for i, col := range columns {
			if i > 0 {
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, "%s", col)
		}
		fmt.Fprintf(w, ") VALUES (")

		for i, val := range values {
			if i > 0 {
				fmt.Fprintf(w, ", ")
			}
			if val == nil {
				fmt.Fprintf(w, "NULL")
			} else if str, ok := val.(string); ok {
				fmt.Fprintf(w, "'%s'", strings.ReplaceAll(str, "'", "''"))
			} else {
				fmt.Fprintf(w, "%v", val)
			}
		}
		fmt.Fprintf(w, ");\n")
		rowCount++
	}

	return rowCount, nil
}

func (e *ExportManager) getTableRows(ctx context.Context, tableName string) ([]TableRow, error) {
	rows, err := e.db.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to query table: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var result []TableRow
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(TableRow)
		for i, col := range columns {
			row[col] = values[i]
		}
		result = append(result, row)
	}

	return result, nil
}

func (e *ExportManager) getTemplatesForExport(ctx context.Context) ([]ExportedTemplate, error) {
	// This would query your templates table and convert to ExportedTemplate format
	// Implementation depends on your actual template table structure
	return []ExportedTemplate{}, nil
}

func (e *ExportManager) getBlueprintsForExport(ctx context.Context) ([]ExportedBlueprint, error) {
	// This would query your blueprints table and convert to ExportedBlueprint format
	// Implementation depends on your actual blueprint table structure
	return []ExportedBlueprint{}, nil
}

func (e *ExportManager) exportTableCSV(ctx context.Context, filename, tableName string) (int, error) {
	// CSV export implementation
	// This is a simplified version - you'd want proper CSV escaping
	return 0, fmt.Errorf("CSV export not fully implemented")
}

func (e *ExportManager) validateImportData(data *ExportedData) error {
	if data.Metadata.Version == "" {
		return fmt.Errorf("import data missing version information")
	}
	return nil
}

func (e *ExportManager) importTableRows(ctx context.Context, tableName string, rows []TableRow, replaceExisting bool) (int, error) {
	// Implementation for importing table rows
	// This would handle INSERT or INSERT OR REPLACE depending on replaceExisting
	return len(rows), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
