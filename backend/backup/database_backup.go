package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"qt1-middleware/database"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DatabaseBackupEngine implements backup functionality for database
type DatabaseBackupEngine struct {
	db          *database.Database
	storage     BackupStorage
	repository  BackupRepository
	tempDir     string
	maxParallel int
}

// NewDatabaseBackupEngine creates a new database backup engine
func NewDatabaseBackupEngine(db *database.Database, storage BackupStorage, repository BackupRepository) *DatabaseBackupEngine {
	return &DatabaseBackupEngine{
		db:          db,
		storage:     storage,
		repository:  repository,
		tempDir:     "/tmp/backups",
		maxParallel: 4,
	}
}

// CreateBackup creates a new database backup
func (e *DatabaseBackupEngine) CreateBackup(ctx context.Context, options BackupOptions) (*BackupMetadata, error) {
	backupID := uuid.New().String()
	startTime := time.Now()
	
	// Create backup metadata
	metadata := &BackupMetadata{
		ID:              backupID,
		Type:            options.Type,
		Status:          BackupStatusRunning,
		StartTime:       startTime,
		CompressionType: options.CompressionType,
		EncryptionType:  options.EncryptionType,
		DatabaseVersion: e.getDatabaseVersion(),
		AppVersion:      "1.0.0", // Should come from app config
		Tags:            options.Tags,
		Description:     options.Description,
		CreatedBy:       "system", // Should come from context
		RetentionPolicy: options.RetentionPolicy,
	}
	
	// Save initial metadata
	if err := e.repository.CreateBackupRecord(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}
	
	// Initialize progress tracking
	progress := &BackupProgress{
		BackupID:       backupID,
		Status:         BackupStatusRunning,
		PercentComplete: 0.0,
		UpdatedAt:      time.Now(),
	}
	
	if err := e.repository.UpdateBackupProgress(ctx, progress); err != nil {
		return nil, fmt.Errorf("failed to initialize progress: %w", err)
	}
	
	// Perform backup in a goroutine for async operation
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.handleBackupError(ctx, backupID, fmt.Errorf("backup panic: %v", r))
			}
		}()
		
		if err := e.performBackup(ctx, backupID, options); err != nil {
			e.handleBackupError(ctx, backupID, err)
		}
	}()
	
	return metadata, nil
}

// performBackup executes the actual backup process
func (e *DatabaseBackupEngine) performBackup(ctx context.Context, backupID string, options BackupOptions) error {
	// Create temporary directory for backup files
	tempDir := filepath.Join(e.tempDir, backupID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	var backupFunc func(context.Context, string, BackupOptions) error
	
	switch options.Type {
	case BackupTypeFull:
		backupFunc = e.performFullBackup
	case BackupTypeIncremental:
		backupFunc = e.performIncrementalBackup
	case BackupTypeDifferential:
		backupFunc = e.performDifferentialBackup
	default:
		return fmt.Errorf("unsupported backup type: %s", options.Type)
	}
	
	if err := backupFunc(ctx, backupID, options); err != nil {
		return err
	}
	
	// Finalize backup
	return e.finalizeBackup(ctx, backupID, tempDir)
}

// performFullBackup creates a complete database backup
func (e *DatabaseBackupEngine) performFullBackup(ctx context.Context, backupID string, options BackupOptions) error {
	tables, err := e.getDatabaseTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database tables: %w", err)
	}
	
	// Filter tables based on options
	tables = e.filterTables(tables, options.IncludeTables, options.ExcludeTables)
	
	// Update progress with total tables
	progress := &BackupProgress{
		BackupID:    backupID,
		Status:      BackupStatusRunning,
		TotalTables: len(tables),
		UpdatedAt:   time.Now(),
	}
	e.repository.UpdateBackupProgress(ctx, progress)
	
	// Create backup file
	backupFile := filepath.Join(e.tempDir, backupID, "backup.sql")
	file, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()
	
	// Write backup header
	if err := e.writeBackupHeader(file, backupID, options); err != nil {
		return fmt.Errorf("failed to write backup header: %w", err)
	}
	
	// Backup each table
	for i, table := range tables {
		if err := ctx.Err(); err != nil {
			return err // Context cancelled
		}
		
		// Update progress
		progress.CurrentTable = table
		progress.TablesComplete = i
		progress.PercentComplete = float64(i) / float64(len(tables)) * 100
		progress.UpdatedAt = time.Now()
		e.repository.UpdateBackupProgress(ctx, progress)
		
		if err := e.backupTable(ctx, file, table); err != nil {
			return fmt.Errorf("failed to backup table %s: %w", table, err)
		}
	}
	
	// Write backup footer
	if err := e.writeBackupFooter(file, backupID); err != nil {
		return fmt.Errorf("failed to write backup footer: %w", err)
	}
	
	// Update final progress
	progress.TablesComplete = len(tables)
	progress.PercentComplete = 100.0
	progress.UpdatedAt = time.Now()
	e.repository.UpdateBackupProgress(ctx, progress)
	
	return nil
}

// performIncrementalBackup creates an incremental backup
func (e *DatabaseBackupEngine) performIncrementalBackup(ctx context.Context, backupID string, options BackupOptions) error {
	if options.ParentBackupID == nil {
		return fmt.Errorf("incremental backup requires parent backup ID")
	}
	
	// Get parent backup metadata
	parentMetadata, err := e.repository.GetBackupRecord(ctx, *options.ParentBackupID)
	if err != nil {
		return fmt.Errorf("failed to get parent backup: %w", err)
	}
	
	// Get changed records since parent backup
	changedTables, err := e.getChangedTables(ctx, parentMetadata.StartTime)
	if err != nil {
		return fmt.Errorf("failed to get changed tables: %w", err)
	}
	
	// Filter tables
	changedTables = e.filterTables(changedTables, options.IncludeTables, options.ExcludeTables)
	
	// Create incremental backup file
	backupFile := filepath.Join(e.tempDir, backupID, "incremental.sql")
	file, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create incremental backup file: %w", err)
	}
	defer file.Close()
	
	// Write incremental backup header
	if err := e.writeIncrementalHeader(file, backupID, *options.ParentBackupID, parentMetadata.StartTime); err != nil {
		return fmt.Errorf("failed to write incremental header: %w", err)
	}
	
	// Backup changed tables
	for _, table := range changedTables {
		if err := e.backupTableIncremental(ctx, file, table, parentMetadata.StartTime); err != nil {
			return fmt.Errorf("failed to backup changed table %s: %w", table, err)
		}
	}
	
	return nil
}

// performDifferentialBackup creates a differential backup
func (e *DatabaseBackupEngine) performDifferentialBackup(ctx context.Context, backupID string, options BackupOptions) error {
	// Find the latest full backup
	filters := BackupFilters{
		Type:      &[]BackupType{BackupTypeFull}[0],
		Status:    &[]BackupStatus{BackupStatusCompleted}[0],
		SortBy:    "start_time",
		SortOrder: "desc",
		Limit:     1,
	}
	
	fullBackups, err := e.repository.ListBackupRecords(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to find full backup: %w", err)
	}
	
	if len(fullBackups) == 0 {
		return fmt.Errorf("no full backup found for differential backup")
	}
	
	lastFullBackup := fullBackups[0]
	
	// Get changed records since last full backup
	changedTables, err := e.getChangedTables(ctx, lastFullBackup.StartTime)
	if err != nil {
		return fmt.Errorf("failed to get changed tables: %w", err)
	}
	
	// Filter tables
	changedTables = e.filterTables(changedTables, options.IncludeTables, options.ExcludeTables)
	
	// Create differential backup file
	backupFile := filepath.Join(e.tempDir, backupID, "differential.sql")
	file, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create differential backup file: %w", err)
	}
	defer file.Close()
	
	// Write differential backup header
	if err := e.writeDifferentialHeader(file, backupID, lastFullBackup.ID, lastFullBackup.StartTime); err != nil {
		return fmt.Errorf("failed to write differential header: %w", err)
	}
	
	// Backup changed tables
	for _, table := range changedTables {
		if err := e.backupTableIncremental(ctx, file, table, lastFullBackup.StartTime); err != nil {
			return fmt.Errorf("failed to backup changed table %s: %w", table, err)
		}
	}
	
	return nil
}

// backupTable backs up a complete table
func (e *DatabaseBackupEngine) backupTable(ctx context.Context, writer io.Writer, tableName string) error {
	// Write table structure
	if err := e.writeTableStructure(ctx, writer, tableName); err != nil {
		return err
	}
	
	// Write table data
	if err := e.writeTableData(ctx, writer, tableName, nil); err != nil {
		return err
	}
	
	return nil
}

// backupTableIncremental backs up only changed records in a table
func (e *DatabaseBackupEngine) backupTableIncremental(ctx context.Context, writer io.Writer, tableName string, since time.Time) error {
	// Write table structure (for reference)
	if err := e.writeTableStructure(ctx, writer, tableName); err != nil {
		return err
	}
	
	// Write only changed data
	whereClause := e.buildIncrementalWhereClause(tableName, since)
	if err := e.writeTableData(ctx, writer, tableName, &whereClause); err != nil {
		return err
	}
	
	return nil
}

// writeTableStructure writes the CREATE TABLE statement
func (e *DatabaseBackupEngine) writeTableStructure(ctx context.Context, writer io.Writer, tableName string) error {
	// Get table creation SQL
	var createSQL string
	query := `SELECT sql FROM sqlite_master WHERE type='table' AND name=?`
	
	err := e.db.DB.QueryRowContext(ctx, query, tableName).Scan(&createSQL)
	if err != nil {
		return fmt.Errorf("failed to get table structure: %w", err)
	}
	
	// Write DROP TABLE IF EXISTS
	fmt.Fprintf(writer, "-- Table structure for %s\n", tableName)
	fmt.Fprintf(writer, "DROP TABLE IF EXISTS `%s`;\n", tableName)
	fmt.Fprintf(writer, "%s;\n\n", createSQL)
	
	return nil
}

// writeTableData writes table data with optional WHERE clause
func (e *DatabaseBackupEngine) writeTableData(ctx context.Context, writer io.Writer, tableName string, whereClause *string) error {
	// Build query
	query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
	if whereClause != nil && *whereClause != "" {
		query += " WHERE " + *whereClause
	}
	
	rows, err := e.db.DB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query table data: %w", err)
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %w", err)
	}
	
	fmt.Fprintf(writer, "-- Data for table %s\n", tableName)
	
	// Process rows
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		
		if err := e.writeInsertStatement(writer, tableName, columns, values); err != nil {
			return err
		}
	}
	
	fmt.Fprintf(writer, "\n")
	return rows.Err()
}

// writeInsertStatement writes an INSERT statement for a row
func (e *DatabaseBackupEngine) writeInsertStatement(writer io.Writer, tableName string, columns []string, values []interface{}) error {
	fmt.Fprintf(writer, "INSERT INTO `%s` (", tableName)
	
	// Write column names
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(writer, ", ")
		}
		fmt.Fprintf(writer, "`%s`", col)
	}
	
	fmt.Fprint(writer, ") VALUES (")
	
	// Write values
	for i, val := range values {
		if i > 0 {
			fmt.Fprint(writer, ", ")
		}
		
		if val == nil {
			fmt.Fprint(writer, "NULL")
		} else {
			switch v := val.(type) {
			case string:
				fmt.Fprintf(writer, "'%s'", strings.ReplaceAll(v, "'", "''"))
			case []byte:
				fmt.Fprintf(writer, "'%s'", strings.ReplaceAll(string(v), "'", "''"))
			case time.Time:
				fmt.Fprintf(writer, "'%s'", v.Format("2006-01-02 15:04:05"))
			default:
				fmt.Fprintf(writer, "%v", val)
			}
		}
	}
	
	fmt.Fprintf(writer, ");\n")
	return nil
}

// getDatabaseTables returns list of all tables in the database
func (e *DatabaseBackupEngine) getDatabaseTables(ctx context.Context) ([]string, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	
	rows, err := e.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	
	return tables, rows.Err()
}

// getChangedTables returns tables that have been modified since the given time
func (e *DatabaseBackupEngine) getChangedTables(ctx context.Context, since time.Time) ([]string, error) {
	// For SQLite, we'll check all tables with updated_at columns
	// This is a simplified approach - in production, you might use triggers or change tracking
	tables, err := e.getDatabaseTables(ctx)
	if err != nil {
		return nil, err
	}
	
	var changedTables []string
	
	for _, table := range tables {
		// Check if table has updated_at column and recent changes
		hasChanges, err := e.tableHasRecentChanges(ctx, table, since)
		if err != nil {
			// If we can't determine, include the table to be safe
			changedTables = append(changedTables, table)
			continue
		}
		
		if hasChanges {
			changedTables = append(changedTables, table)
		}
	}
	
	return changedTables, nil
}

// tableHasRecentChanges checks if a table has records modified since the given time
func (e *DatabaseBackupEngine) tableHasRecentChanges(ctx context.Context, tableName string, since time.Time) (bool, error) {
	// Check if table has updated_at column
	query := fmt.Sprintf("PRAGMA table_info(`%s`)", tableName)
	rows, err := e.db.DB.QueryContext(ctx, query)
	if err != nil {
		return true, err // Assume changes if we can't check
	}
	defer rows.Close()
	
	hasUpdatedAt := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue sql.NullString
		
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		
		if name == "updated_at" || name == "modified_at" {
			hasUpdatedAt = true
			break
		}
	}
	
	if !hasUpdatedAt {
		return true, nil // Assume changes if no timestamp column
	}
	
	// Check for recent changes
	checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE updated_at > ?", tableName)
	var count int
	
	err = e.db.DB.QueryRowContext(ctx, checkQuery, since).Scan(&count)
	if err != nil {
		return true, nil // Assume changes if query fails
	}
	
	return count > 0, nil
}

// filterTables filters the table list based on include/exclude lists
func (e *DatabaseBackupEngine) filterTables(tables []string, include []string, exclude []string) []string {
	if len(include) == 0 && len(exclude) == 0 {
		return tables
	}
	
	var filtered []string
	
	for _, table := range tables {
		// Check exclude list first
		excluded := false
		for _, excl := range exclude {
			if table == excl {
				excluded = true
				break
			}
		}
		
		if excluded {
			continue
		}
		
		// If include list is specified, table must be in it
		if len(include) > 0 {
			included := false
			for _, incl := range include {
				if table == incl {
					included = true
					break
				}
			}
			
			if !included {
				continue
			}
		}
		
		filtered = append(filtered, table)
	}
	
	return filtered
}

// buildIncrementalWhereClause builds WHERE clause for incremental backups
func (e *DatabaseBackupEngine) buildIncrementalWhereClause(tableName string, since time.Time) string {
	// Try common timestamp column names
	timestampColumns := []string{"updated_at", "modified_at", "created_at"}
	
	for _, col := range timestampColumns {
		// Check if column exists (simplified check)
		return fmt.Sprintf("%s > '%s'", col, since.Format("2006-01-02 15:04:05"))
	}
	
	// If no timestamp column found, return empty (backup all data)
	return ""
}

// writeBackupHeader writes the backup file header
func (e *DatabaseBackupEngine) writeBackupHeader(writer io.Writer, backupID string, options BackupOptions) error {
	fmt.Fprintf(writer, "-- Database Backup\n")
	fmt.Fprintf(writer, "-- Backup ID: %s\n", backupID)
	fmt.Fprintf(writer, "-- Backup Type: %s\n", options.Type)
	fmt.Fprintf(writer, "-- Created: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "-- Description: %s\n", options.Description)
	fmt.Fprintf(writer, "\n")
	fmt.Fprintf(writer, "PRAGMA foreign_keys=OFF;\n")
	fmt.Fprintf(writer, "BEGIN TRANSACTION;\n\n")
	
	return nil
}

// writeBackupFooter writes the backup file footer
func (e *DatabaseBackupEngine) writeBackupFooter(writer io.Writer, backupID string) error {
	fmt.Fprintf(writer, "\n")
	fmt.Fprintf(writer, "COMMIT;\n")
	fmt.Fprintf(writer, "PRAGMA foreign_keys=ON;\n")
	fmt.Fprintf(writer, "-- End of backup %s\n", backupID)
	
	return nil
}

// writeIncrementalHeader writes header for incremental backup
func (e *DatabaseBackupEngine) writeIncrementalHeader(writer io.Writer, backupID, parentID string, since time.Time) error {
	fmt.Fprintf(writer, "-- Incremental Database Backup\n")
	fmt.Fprintf(writer, "-- Backup ID: %s\n", backupID)
	fmt.Fprintf(writer, "-- Parent Backup ID: %s\n", parentID)
	fmt.Fprintf(writer, "-- Changes Since: %s\n", since.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "-- Created: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "\n")
	fmt.Fprintf(writer, "PRAGMA foreign_keys=OFF;\n")
	fmt.Fprintf(writer, "BEGIN TRANSACTION;\n\n")
	
	return nil
}

// writeDifferentialHeader writes header for differential backup
func (e *DatabaseBackupEngine) writeDifferentialHeader(writer io.Writer, backupID, baseID string, since time.Time) error {
	fmt.Fprintf(writer, "-- Differential Database Backup\n")
	fmt.Fprintf(writer, "-- Backup ID: %s\n", backupID)
	fmt.Fprintf(writer, "-- Base Backup ID: %s\n", baseID)
	fmt.Fprintf(writer, "-- Changes Since: %s\n", since.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "-- Created: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "\n")
	fmt.Fprintf(writer, "PRAGMA foreign_keys=OFF;\n")
	fmt.Fprintf(writer, "BEGIN TRANSACTION;\n\n")
	
	return nil
}

// finalizeBackup completes the backup process
func (e *DatabaseBackupEngine) finalizeBackup(ctx context.Context, backupID, tempDir string) error {
	// Get backup file info
	backupFile := filepath.Join(tempDir, "backup.sql")
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		// Try incremental or differential
		if _, err := os.Stat(filepath.Join(tempDir, "incremental.sql")); err == nil {
			backupFile = filepath.Join(tempDir, "incremental.sql")
		} else if _, err := os.Stat(filepath.Join(tempDir, "differential.sql")); err == nil {
			backupFile = filepath.Join(tempDir, "differential.sql")
		} else {
			return fmt.Errorf("backup file not found")
		}
	}
	
	// Calculate file size and checksum
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()
	
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}
	
	// Calculate checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))
	
	// Reset file position for storage
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset file position: %w", err)
	}
	
	// Store backup file
	storagePath, err := e.storage.Store(ctx, backupID, file)
	if err != nil {
		return fmt.Errorf("failed to store backup: %w", err)
	}
	
	// Update metadata
	endTime := time.Now()
	
	metadata, err := e.repository.GetBackupRecord(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup metadata: %w", err)
	}
	
	metadata.Status = BackupStatusCompleted
	metadata.EndTime = &endTime
	metadata.Duration = endTime.Sub(metadata.StartTime)
	metadata.Size = stat.Size()
	metadata.CompressedSize = stat.Size() // Will be different with compression
	metadata.FilePath = storagePath
	metadata.Checksum = checksum
	
	if err := e.repository.UpdateBackupRecord(ctx, metadata); err != nil {
		return fmt.Errorf("failed to update backup metadata: %w", err)
	}
	
	// Update final progress
	progress := &BackupProgress{
		BackupID:        backupID,
		Status:          BackupStatusCompleted,
		PercentComplete: 100.0,
		BytesProcessed:  stat.Size(),
		TotalBytes:      stat.Size(),
		UpdatedAt:       time.Now(),
	}
	
	if err := e.repository.UpdateBackupProgress(ctx, progress); err != nil {
		return fmt.Errorf("failed to update final progress: %w", err)
	}
	
	return nil
}

// handleBackupError handles backup failures
func (e *DatabaseBackupEngine) handleBackupError(ctx context.Context, backupID string, err error) {
	errorMsg := err.Error()
	
	// Update backup status
	e.repository.UpdateBackupStatus(ctx, backupID, BackupStatusFailed, &errorMsg)
	
	// Update progress
	progress := &BackupProgress{
		BackupID:     backupID,
		Status:       BackupStatusFailed,
		ErrorMessage: &errorMsg,
		UpdatedAt:    time.Now(),
	}
	e.repository.UpdateBackupProgress(ctx, progress)
}

// getDatabaseVersion returns the database version
func (e *DatabaseBackupEngine) getDatabaseVersion() string {
	var version string
	e.db.DB.QueryRow("SELECT sqlite_version()").Scan(&version)
	return version
}

// Implement other BackupEngine interface methods...

// GetBackupMetadata retrieves metadata for a specific backup
func (e *DatabaseBackupEngine) GetBackupMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	return e.repository.GetBackupRecord(ctx, backupID)
}

// ListBackups returns a list of available backups with optional filtering
func (e *DatabaseBackupEngine) ListBackups(ctx context.Context, filters BackupFilters) ([]*BackupMetadata, error) {
	return e.repository.ListBackupRecords(ctx, filters)
}

// DeleteBackup removes a backup and its associated files
func (e *DatabaseBackupEngine) DeleteBackup(ctx context.Context, backupID string) error {
	// Get backup metadata
	_, err := e.repository.GetBackupRecord(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup metadata: %w", err)
	}
	
	// Delete from storage
	if err := e.storage.Delete(ctx, backupID); err != nil {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}
	
	// Delete metadata record
	if err := e.repository.DeleteBackupRecord(ctx, backupID); err != nil {
		return fmt.Errorf("failed to delete backup record: %w", err)
	}
	
	return nil
}

// GetProgress returns the current progress of a running operation
func (e *DatabaseBackupEngine) GetProgress(ctx context.Context, backupID string) (*BackupProgress, error) {
	return e.repository.GetBackupProgress(ctx, backupID)
}

// CancelOperation cancels a running backup operation
func (e *DatabaseBackupEngine) CancelOperation(ctx context.Context, backupID string) error {
	// Update status to cancelled
	return e.repository.UpdateBackupStatus(ctx, backupID, BackupStatusCancelled, nil)
}

// RestoreBackup restores from a backup (delegates to restore engine)
func (e *DatabaseBackupEngine) RestoreBackup(ctx context.Context, options RestoreOptions) error {
	// This method is required by the BackupEngine interface but is typically
	// handled by a separate RestoreEngine. For now, return an error indicating
	// that restore should be done through the BackupManager.
	return fmt.Errorf("restore operations should be performed through the BackupManager")
}

// ValidateBackup validates a backup (delegates to validator)
func (e *DatabaseBackupEngine) ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error) {
	// This method is required by the BackupEngine interface but is typically
	// handled by a separate BackupValidator. For now, return an error indicating
	// that validation should be done through the BackupManager.
	return nil, fmt.Errorf("validation operations should be performed through the BackupManager")
}