package db

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
)

// BackupManager handles database backup and restore operations
type BackupManager struct {
	db   *Manager
	path string
}

// NewBackupManager creates a new backup manager
func NewBackupManager(manager *Manager, dbPath string) *BackupManager {
	return &BackupManager{
		db:   manager,
		path: dbPath,
	}
}

// BackupOptions contains options for database backup
type BackupOptions struct {
	OutputPath string
	Compress   bool
	Verify     bool
	Verbose    bool
}

// RestoreOptions contains options for database restore
type RestoreOptions struct {
	BackupPath   string
	Verify       bool
	CreateBackup bool
	Force        bool
	Verbose      bool
}

// Backup creates a backup of the database
func (b *BackupManager) Backup(ctx context.Context, opts BackupOptions) error {
	if opts.Verbose {
		color.Yellow("Starting database backup...")
	}

	// Validate source database exists
	if _, err := os.Stat(b.path); os.IsNotExist(err) {
		return fmt.Errorf("source database does not exist: %s", b.path)
	}

	// Create output directory if needed
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Perform backup
	if opts.Compress {
		if err := b.backupCompressed(ctx, opts); err != nil {
			return fmt.Errorf("compressed backup failed: %w", err)
		}
	} else {
		if err := b.backupRaw(ctx, opts); err != nil {
			return fmt.Errorf("raw backup failed: %w", err)
		}
	}

	// Verify backup if requested
	if opts.Verify {
		if err := b.verifyBackup(ctx, opts.OutputPath, opts.Verbose); err != nil {
			return fmt.Errorf("backup verification failed: %w", err)
		}
	}

	// Get backup file size
	stat, err := os.Stat(opts.OutputPath)
	if err == nil {
		color.Green("✓ Backup completed: %s (%.2f MB)", opts.OutputPath, float64(stat.Size())/1024/1024)
	} else {
		color.Green("✓ Backup completed: %s", opts.OutputPath)
	}

	return nil
}

// backupRaw performs a raw file copy backup
func (b *BackupManager) backupRaw(ctx context.Context, opts BackupOptions) error {
	// Open source database file
	srcFile, err := os.Open(b.path)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	// Copy database file
	if opts.Verbose {
		color.Yellow("Copying database file...")
	}

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy database: %w", err)
	}

	return dstFile.Sync()
}

// backupCompressed performs a compressed backup
func (b *BackupManager) backupCompressed(ctx context.Context, opts BackupOptions) error {
	// Open source database file
	srcFile, err := os.Open(b.path)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dstFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	// Set gzip metadata
	gzWriter.Name = filepath.Base(b.path)
	gzWriter.ModTime = time.Now()

	if opts.Verbose {
		color.Yellow("Compressing database...")
	}

	// Copy and compress database
	_, err = io.Copy(gzWriter, srcFile)
	if err != nil {
		return fmt.Errorf("failed to compress database: %w", err)
	}

	// Ensure everything is written
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize compression: %w", err)
	}

	return dstFile.Sync()
}

// Restore restores a database from backup
func (b *BackupManager) Restore(ctx context.Context, opts RestoreOptions) error {
	if opts.Verbose {
		color.Yellow("Starting database restore...")
	}

	// Validate backup file exists
	if _, err := os.Stat(opts.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", opts.BackupPath)
	}

	// Check if destination database exists
	destExists := false
	if _, err := os.Stat(b.path); err == nil {
		destExists = true
		if !opts.Force {
			return fmt.Errorf("destination database already exists: %s (use --force to overwrite)", b.path)
		}
	}

	// Create backup of existing database if requested
	if opts.CreateBackup && destExists {
		backupPath := fmt.Sprintf("%s.backup.%d", b.path, time.Now().Unix())
		if opts.Verbose {
			color.Yellow("Creating backup of existing database: %s", backupPath)
		}

		if err := b.backupRaw(ctx, BackupOptions{
			OutputPath: backupPath,
			Verbose:    opts.Verbose,
		}); err != nil {
			return fmt.Errorf("failed to backup existing database: %w", err)
		}
	}

	// Determine if backup is compressed
	isCompressed, err := b.isCompressedFile(opts.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to check backup format: %w", err)
	}

	// Restore from backup
	if isCompressed {
		if err := b.restoreCompressed(ctx, opts); err != nil {
			return fmt.Errorf("compressed restore failed: %w", err)
		}
	} else {
		if err := b.restoreRaw(ctx, opts); err != nil {
			return fmt.Errorf("raw restore failed: %w", err)
		}
	}

	// Verify restored database if requested
	if opts.Verify {
		if err := b.verifyDatabase(ctx, b.path, opts.Verbose); err != nil {
			return fmt.Errorf("restored database verification failed: %w", err)
		}
	}

	color.Green("✓ Database restored successfully from: %s", opts.BackupPath)
	return nil
}

// restoreRaw restores from a raw database file
func (b *BackupManager) restoreRaw(ctx context.Context, opts RestoreOptions) error {
	// Create destination directory if needed
	destDir := filepath.Dir(b.path)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open backup file
	srcFile, err := os.Open(opts.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(b.path)
	if err != nil {
		return fmt.Errorf("failed to create destination database: %w", err)
	}
	defer dstFile.Close()

	if opts.Verbose {
		color.Yellow("Copying backup file...")
	}

	// Copy backup to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy backup: %w", err)
	}

	return dstFile.Sync()
}

// restoreCompressed restores from a compressed backup
func (b *BackupManager) restoreCompressed(ctx context.Context, opts RestoreOptions) error {
	// Create destination directory if needed
	destDir := filepath.Dir(b.path)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open compressed backup file
	srcFile, err := os.Open(opts.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer srcFile.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create destination file
	dstFile, err := os.Create(b.path)
	if err != nil {
		return fmt.Errorf("failed to create destination database: %w", err)
	}
	defer dstFile.Close()

	if opts.Verbose {
		color.Yellow("Decompressing backup...")
	}

	// Decompress and copy
	_, err = io.Copy(dstFile, gzReader)
	if err != nil {
		return fmt.Errorf("failed to decompress backup: %w", err)
	}

	return dstFile.Sync()
}

// verifyBackup verifies the integrity of a backup file
func (b *BackupManager) verifyBackup(ctx context.Context, backupPath string, verbose bool) error {
	if verbose {
		color.Yellow("Verifying backup integrity...")
	}

	isCompressed, err := b.isCompressedFile(backupPath)
	if err != nil {
		return err
	}

	if isCompressed {
		return b.verifyCompressedBackup(backupPath, verbose)
	}

	return b.verifyDatabase(ctx, backupPath, verbose)
}

// verifyCompressedBackup verifies a compressed backup can be read
func (b *BackupManager) verifyCompressedBackup(backupPath string, verbose bool) error {
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("backup file is corrupted (invalid gzip): %w", err)
	}
	defer gzReader.Close()

	// Read and discard content to verify decompression works
	_, err = io.Copy(io.Discard, gzReader)
	if err != nil {
		return fmt.Errorf("backup file is corrupted (decompression failed): %w", err)
	}

	if verbose {
		color.Green("✓ Compressed backup verified successfully")
	}

	return nil
}

// verifyDatabase verifies database integrity
func (b *BackupManager) verifyDatabase(ctx context.Context, dbPath string, verbose bool) error {
	// Create a temporary manager to test the database
	tempManager := NewManager()
	if err := tempManager.Open(ctx, dbPath); err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer tempManager.Close()

	// Run PRAGMA integrity_check
	row := tempManager.db.QueryRowContext(ctx, "PRAGMA integrity_check")
	var result string
	if err := row.Scan(&result); err != nil {
		return fmt.Errorf("failed to check database integrity: %w", err)
	}

	if result != "ok" {
		return fmt.Errorf("database integrity check failed: %s", result)
	}

	if verbose {
		color.Green("✓ Database integrity verified successfully")
	}

	return nil
}

// isCompressedFile checks if a file is gzip compressed
func (b *BackupManager) isCompressedFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to check for gzip magic number
	buffer := make([]byte, 2)
	_, err = file.Read(buffer)
	if err != nil {
		return false, fmt.Errorf("failed to read file header: %w", err)
	}

	// Gzip magic number is 0x1f, 0x8b
	return buffer[0] == 0x1f && buffer[1] == 0x8b, nil
}

// GetBackupInfo returns information about a backup file
func (b *BackupManager) GetBackupInfo(backupPath string) (*BackupInfo, error) {
	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	isCompressed, err := b.isCompressedFile(backupPath)
	if err != nil {
		return nil, err
	}

	return &BackupInfo{
		Path:         backupPath,
		Size:         stat.Size(),
		ModTime:      stat.ModTime(),
		IsCompressed: isCompressed,
	}, nil
}

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Path         string
	Size         int64
	ModTime      time.Time
	IsCompressed bool
}

// String returns a string representation of backup info
func (bi *BackupInfo) String() string {
	compressionStatus := "Raw"
	if bi.IsCompressed {
		compressionStatus = "Compressed"
	}

	return fmt.Sprintf("%s (%.2f MB, %s, %s)",
		bi.Path,
		float64(bi.Size)/1024/1024,
		compressionStatus,
		bi.ModTime.Format("2006-01-02 15:04:05"),
	)
}
