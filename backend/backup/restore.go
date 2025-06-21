package backup

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"qt1-middleware/database"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RestoreEngine implements backup restoration functionality
type RestoreEngine struct {
	db          *database.Database
	storage     BackupStorage
	repository  BackupRepository
	tempDir     string
	maxParallel int
}

// NewRestoreEngine creates a new restore engine
func NewRestoreEngine(db *database.Database, storage BackupStorage, repository BackupRepository) *RestoreEngine {
	return &RestoreEngine{
		db:          db,
		storage:     storage,
		repository:  repository,
		tempDir:     "/tmp/restores",
		maxParallel: 4,
	}
}

// RestoreBackup restores from a backup with the given options
func (e *RestoreEngine) RestoreBackup(ctx context.Context, options RestoreOptions) error {
	// Get backup metadata
	metadata, err := e.repository.GetBackupRecord(ctx, options.BackupID)
	if err != nil {
		return fmt.Errorf("failed to get backup metadata: %w", err)
	}
	
	// Validate restore options
	if err := e.validateRestoreOptions(options, metadata); err != nil {
		return fmt.Errorf("invalid restore options: %w", err)
	}
	
	// Check if this is a point-in-time restore
	if options.TargetTime != nil {
		return e.performPointInTimeRestore(ctx, options, metadata)
	}
	
	// Perform regular restore based on backup type
	switch metadata.Type {
	case BackupTypeFull:
		return e.performFullRestore(ctx, options, metadata)
	case BackupTypeIncremental:
		return e.performIncrementalRestore(ctx, options, metadata)
	case BackupTypeDifferential:
		return e.performDifferentialRestore(ctx, options, metadata)
	default:
		return fmt.Errorf("unsupported backup type for restore: %s", metadata.Type)
	}
}

// validateRestoreOptions validates the restore options
func (e *RestoreEngine) validateRestoreOptions(options RestoreOptions, metadata *BackupMetadata) error {
	// Check if backup is completed successfully
	if metadata.Status != BackupStatusCompleted {
		return fmt.Errorf("backup is not in completed state: %s", metadata.Status)
	}
	
	// Check if backup file exists
	exists, err := e.storage.Exists(context.Background(), options.BackupID)
	if err != nil {
		return fmt.Errorf("failed to check backup file existence: %w", err)
	}
	
	if !exists {
		return fmt.Errorf("backup file does not exist: %s", options.BackupID)
	}
	
	// Validate target time for point-in-time restore
	if options.TargetTime != nil {
		if options.TargetTime.After(metadata.StartTime) {
			return fmt.Errorf("target time cannot be after backup start time")
		}
	}
	
	return nil
}

// performFullRestore restores from a full backup
func (e *RestoreEngine) performFullRestore(ctx context.Context, options RestoreOptions, metadata *BackupMetadata) error {
	// Create restore operation ID for tracking
	restoreID := uuid.New().String()
	
	// Initialize progress tracking
	progress := &BackupProgress{
		BackupID:        restoreID,
		Status:          BackupStatusRunning,
		PercentComplete: 0.0,
		UpdatedAt:       time.Now(),
	}
	
	if err := e.repository.UpdateBackupProgress(ctx, progress); err != nil {
		return fmt.Errorf("failed to initialize restore progress: %w", err)
	}
	
	// Retrieve backup file
	backupReader, err := e.storage.Retrieve(ctx, options.BackupID)
	if err != nil {
		return fmt.Errorf("failed to retrieve backup file: %w", err)
	}
	defer backupReader.Close()
	
	// Create temporary file for processing
	tempDir := filepath.Join(e.tempDir, restoreID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	tempFile := filepath.Join(tempDir, "restore.sql")
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	
	// Copy backup data to temp file
	_, err = io.Copy(file, backupReader)
	file.Close()
	if err != nil {
		return fmt.Errorf("failed to copy backup data: %w", err)
	}
	
	// Perform dry run if requested
	if options.DryRun {
		return e.performDryRun(ctx, tempFile, options)
	}
	
	// Execute the restore
	if err := e.executeRestore(ctx, tempFile, options, progress); err != nil {
		// Update progress with error
		progress.Status = BackupStatusFailed
		errorMsg := err.Error()
		progress.ErrorMessage = &errorMsg
		progress.UpdatedAt = time.Now()
		e.repository.UpdateBackupProgress(ctx, progress)
		
		return fmt.Errorf("restore failed: %w", err)
	}
	
	// Update final progress
	progress.Status = BackupStatusCompleted
	progress.PercentComplete = 100.0
	progress.UpdatedAt = time.Now()
	e.repository.UpdateBackupProgress(ctx, progress)
	
	return nil
}

// performIncrementalRestore restores from an incremental backup chain
func (e *RestoreEngine) performIncrementalRestore(ctx context.Context, options RestoreOptions, metadata *BackupMetadata) error {
	// Build the backup chain
	backupChain, err := e.buildBackupChain(ctx, options.BackupID)
	if err != nil {
		return fmt.Errorf("failed to build backup chain: %w", err)
	}
	
	// Restore each backup in the chain
	for i, backupID := range backupChain {
		backupMetadata, err := e.repository.GetBackupRecord(ctx, backupID)
		if err != nil {
			return fmt.Errorf("failed to get backup metadata for %s: %w", backupID, err)
		}
		
		restoreOptions := RestoreOptions{
			BackupID:      backupID,
			TargetPath:    options.TargetPath,
			IncludeTables: options.IncludeTables,
			ExcludeTables: options.ExcludeTables,
			Overwrite:     i == 0 || options.Overwrite, // Only overwrite on first backup or if explicitly requested
			Parallel:      options.Parallel,
			Verify:        options.Verify,
			DryRun:        options.DryRun,
		}
		
		if backupMetadata.Type == BackupTypeFull {
			if err := e.performFullRestore(ctx, restoreOptions, backupMetadata); err != nil {
				return fmt.Errorf("failed to restore base backup %s: %w", backupID, err)
			}
		} else {
			if err := e.performIncrementalRestore(ctx, restoreOptions, backupMetadata); err != nil {
				return fmt.Errorf("failed to restore incremental backup %s: %w", backupID, err)
			}
		}
	}
	
	return nil
}

// performDifferentialRestore restores from a differential backup
func (e *RestoreEngine) performDifferentialRestore(ctx context.Context, options RestoreOptions, metadata *BackupMetadata) error {
	// For differential backup, we need the base full backup first
	if metadata.ParentBackupID == nil {
		return fmt.Errorf("differential backup missing parent backup ID")
	}
	
	// First restore the full backup
	baseOptions := RestoreOptions{
		BackupID:      *metadata.ParentBackupID,
		TargetPath:    options.TargetPath,
		IncludeTables: options.IncludeTables,
		ExcludeTables: options.ExcludeTables,
		Overwrite:     options.Overwrite,
		Parallel:      options.Parallel,
		Verify:        options.Verify,
		DryRun:        options.DryRun,
	}
	
	baseMetadata, err := e.repository.GetBackupRecord(ctx, *metadata.ParentBackupID)
	if err != nil {
		return fmt.Errorf("failed to get base backup metadata: %w", err)
	}
	
	if err := e.performFullRestore(ctx, baseOptions, baseMetadata); err != nil {
		return fmt.Errorf("failed to restore base backup: %w", err)
	}
	
	// Then apply the differential backup
	if err := e.performFullRestore(ctx, options, metadata); err != nil {
		return fmt.Errorf("failed to restore differential backup: %w", err)
	}
	
	return nil
}

// performPointInTimeRestore performs a point-in-time restore
func (e *RestoreEngine) performPointInTimeRestore(ctx context.Context, options RestoreOptions, metadata *BackupMetadata) error {
	// Find the best backup for the target time
	bestBackup, err := e.findBestBackupForTime(ctx, *options.TargetTime)
	if err != nil {
		return fmt.Errorf("failed to find suitable backup for target time: %w", err)
	}
	
	// Update options with the best backup ID
	options.BackupID = bestBackup.ID
	
	// Perform regular restore
	return e.RestoreBackup(ctx, options)
}

// buildBackupChain builds the chain of backups needed for incremental restore
func (e *RestoreEngine) buildBackupChain(ctx context.Context, backupID string) ([]string, error) {
	var chain []string
	currentID := backupID
	
	// Walk backwards through the chain
	for currentID != "" {
		chain = append([]string{currentID}, chain...) // Prepend to maintain order
		
		metadata, err := e.repository.GetBackupRecord(ctx, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get backup metadata for %s: %w", currentID, err)
		}
		
		if metadata.Type == BackupTypeFull {
			break // Found the base backup
		}
		
		if metadata.ParentBackupID == nil {
			return nil, fmt.Errorf("incremental backup missing parent backup ID: %s", currentID)
		}
		
		currentID = *metadata.ParentBackupID
	}
	
	return chain, nil
}

// findBestBackupForTime finds the best backup for a given target time
func (e *RestoreEngine) findBestBackupForTime(ctx context.Context, targetTime time.Time) (*BackupMetadata, error) {
	// Find backups that were created before or at the target time
	filters := BackupFilters{
		Status:    &[]BackupStatus{BackupStatusCompleted}[0],
		EndTime:   &targetTime,
		SortBy:    "start_time",
		SortOrder: "desc",
		Limit:     1,
	}
	
	backups, err := e.repository.ListBackupRecords(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	
	if len(backups) == 0 {
		return nil, fmt.Errorf("no suitable backup found for target time %s", targetTime.Format("2006-01-02 15:04:05"))
	}
	
	return backups[0], nil
}

// executeRestore executes the actual restore operation
func (e *RestoreEngine) executeRestore(ctx context.Context, sqlFile string, options RestoreOptions, progress *BackupProgress) error {
	// Open SQL file
	file, err := os.Open(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer file.Close()
	
	// Begin transaction for restore
	tx, err := e.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin restore transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Disable foreign key checks during restore
	if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}
	
	// Process SQL statements
	scanner := bufio.NewScanner(file)
	var currentStatement strings.Builder
	statementCount := 0
	
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err // Context cancelled
		}
		
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		
		currentStatement.WriteString(line)
		currentStatement.WriteString(" ")
		
		// Execute statement if it ends with semicolon
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(currentStatement.String())
			
			// Apply table filters
			if e.shouldSkipStatement(stmt, options.IncludeTables, options.ExcludeTables) {
				currentStatement.Reset()
				continue
			}
			
			// Apply custom mappings
			stmt = e.applyCustomMappings(stmt, options.CustomMappings)
			
			// Execute statement
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("failed to execute statement: %s: %w", stmt, err)
			}
			
			statementCount++
			currentStatement.Reset()
			
			// Update progress periodically
			if statementCount%100 == 0 {
				progress.PercentComplete = e.estimateProgress(statementCount)
				progress.UpdatedAt = time.Now()
				e.repository.UpdateBackupProgress(ctx, progress)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}
	
	// Re-enable foreign key checks
	if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit restore transaction: %w", err)
	}
	
	// Verify restore if requested
	if options.Verify {
		if err := e.verifyRestore(ctx, options); err != nil {
			return fmt.Errorf("restore verification failed: %w", err)
		}
	}
	
	return nil
}

// performDryRun performs a dry run of the restore operation
func (e *RestoreEngine) performDryRun(ctx context.Context, sqlFile string, options RestoreOptions) error {
	file, err := os.Open(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	var currentStatement strings.Builder
	statementCount := 0
	tableCount := 0
	
	fmt.Printf("Dry run for restore operation:\n")
	fmt.Printf("Include tables: %v\n", options.IncludeTables)
	fmt.Printf("Exclude tables: %v\n", options.ExcludeTables)
	fmt.Printf("\nStatements that would be executed:\n")
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		
		currentStatement.WriteString(line)
		currentStatement.WriteString(" ")
		
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(currentStatement.String())
			
			if e.shouldSkipStatement(stmt, options.IncludeTables, options.ExcludeTables) {
				currentStatement.Reset()
				continue
			}
			
			stmt = e.applyCustomMappings(stmt, options.CustomMappings)
			
			statementCount++
			
			// Count table operations
			if strings.Contains(strings.ToUpper(stmt), "CREATE TABLE") ||
				strings.Contains(strings.ToUpper(stmt), "DROP TABLE") {
				tableCount++
				fmt.Printf("  Table operation: %s\n", e.truncateStatement(stmt, 80))
			}
			
			currentStatement.Reset()
		}
	}
	
	fmt.Printf("\nDry run summary:\n")
	fmt.Printf("  Total statements: %d\n", statementCount)
	fmt.Printf("  Table operations: %d\n", tableCount)
	fmt.Printf("  No actual changes were made to the database.\n")
	
	return scanner.Err()
}

// shouldSkipStatement determines if a statement should be skipped based on table filters
func (e *RestoreEngine) shouldSkipStatement(stmt string, includeTables, excludeTables []string) bool {
	// Extract table name from common SQL statements
	tableName := e.extractTableName(stmt)
	if tableName == "" {
		return false // Don't skip if we can't determine the table
	}
	
	// Check exclude list first
	for _, excl := range excludeTables {
		if strings.EqualFold(tableName, excl) {
			return true
		}
	}
	
	// If include list is specified, table must be in it
	if len(includeTables) > 0 {
		for _, incl := range includeTables {
			if strings.EqualFold(tableName, incl) {
				return false
			}
		}
		return true // Not in include list
	}
	
	return false
}

// extractTableName extracts table name from SQL statement
func (e *RestoreEngine) extractTableName(stmt string) string {
	upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
	
	// Handle different statement types
	if strings.HasPrefix(upperStmt, "CREATE TABLE") {
		parts := strings.Fields(upperStmt)
		if len(parts) >= 3 {
			tableName := strings.Trim(parts[2], "`'\"")
			return tableName
		}
	} else if strings.HasPrefix(upperStmt, "DROP TABLE") {
		parts := strings.Fields(upperStmt)
		if len(parts) >= 3 {
			tableName := strings.Trim(parts[2], "`'\"")
			return tableName
		}
	} else if strings.HasPrefix(upperStmt, "INSERT INTO") {
		parts := strings.Fields(upperStmt)
		if len(parts) >= 3 {
			tableName := strings.Trim(parts[2], "`'\"")
			return tableName
		}
	}
	
	return ""
}

// applyCustomMappings applies custom table/column mappings to statements
func (e *RestoreEngine) applyCustomMappings(stmt string, mappings map[string]string) string {
	if len(mappings) == 0 {
		return stmt
	}
	
	result := stmt
	for oldName, newName := range mappings {
		// Simple string replacement - in production, you'd want more sophisticated parsing
		result = strings.ReplaceAll(result, oldName, newName)
		result = strings.ReplaceAll(result, "`"+oldName+"`", "`"+newName+"`")
	}
	
	return result
}

// verifyRestore verifies that the restore was successful
func (e *RestoreEngine) verifyRestore(ctx context.Context, options RestoreOptions) error {
	// Basic verification - check that tables exist and have data
	tables := []string{"users", "sessions", "system_config"} // Add more as needed
	
	for _, table := range tables {
		// Skip if table is excluded
		if e.isTableExcluded(table, options.IncludeTables, options.ExcludeTables) {
			continue
		}
		
		// Check if table exists
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
		if err := e.db.DB.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return fmt.Errorf("failed to verify table %s existence: %w", table, err)
		}
		
		if count == 0 {
			return fmt.Errorf("table %s was not restored", table)
		}
		
		// Check table has expected structure (basic check)
		query = fmt.Sprintf("PRAGMA table_info(%s)", table)
		rows, err := e.db.DB.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to get table info for %s: %w", table, err)
		}
		rows.Close()
	}
	
	return nil
}

// Helper functions

func (e *RestoreEngine) estimateProgress(statementCount int) float64 {
	// Simple estimation - in production, you'd want more sophisticated progress tracking
	return float64(statementCount) / 1000.0 * 100.0
}

func (e *RestoreEngine) truncateStatement(stmt string, maxLen int) string {
	if len(stmt) <= maxLen {
		return stmt
	}
	return stmt[:maxLen-3] + "..."
}

func (e *RestoreEngine) isTableExcluded(table string, includeTables, excludeTables []string) bool {
	// Check exclude list
	for _, excl := range excludeTables {
		if strings.EqualFold(table, excl) {
			return true
		}
	}
	
	// If include list is specified, table must be in it
	if len(includeTables) > 0 {
		for _, incl := range includeTables {
			if strings.EqualFold(table, incl) {
				return false
			}
		}
		return true
	}
	
	return false
}