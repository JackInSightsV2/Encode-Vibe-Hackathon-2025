package backup

import (
	"bufio"
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
)

// BackupValidator provides backup validation and verification functionality
type BackupValidator struct {
	db         *database.Database
	storage    BackupStorage
	repository BackupRepository
	tempDir    string
}

// NewBackupValidator creates a new backup validator
func NewBackupValidator(db *database.Database, storage BackupStorage, repository BackupRepository) *BackupValidator {
	return &BackupValidator{
		db:         db,
		storage:    storage,
		repository: repository,
		tempDir:    "/tmp/validation",
	}
}

// ValidateBackup verifies the integrity of a backup
func (v *BackupValidator) ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error) {
	startTime := time.Now()
	
	result := &ValidationResult{
		BackupID:       backupID,
		IsValid:        false,
		ChecksumMatch:  false,
		TableCounts:    make(map[string]int64),
		ValidatedAt:    startTime,
	}
	
	// Get backup metadata
	metadata, err := v.repository.GetBackupRecord(ctx, backupID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get backup metadata: %v", err))
		result.ValidationTime = time.Since(startTime)
		return result, nil
	}
	
	// Check if backup file exists
	exists, err := v.storage.Exists(ctx, backupID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to check backup file existence: %v", err))
		result.ValidationTime = time.Since(startTime)
		return result, nil
	}
	
	if !exists {
		result.Errors = append(result.Errors, "Backup file does not exist in storage")
		result.ValidationTime = time.Since(startTime)
		return result, nil
	}
	
	// Validate file size
	size, err := v.storage.GetSize(ctx, backupID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get backup file size: %v", err))
	} else if size != metadata.Size {
		result.Errors = append(result.Errors, fmt.Sprintf("File size mismatch: expected %d, got %d", metadata.Size, size))
	}
	
	// Validate checksum
	if err := v.validateChecksum(ctx, backupID, metadata.Checksum, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Checksum validation failed: %v", err))
	}
	
	// Validate SQL syntax and structure
	if err := v.validateSQLStructure(ctx, backupID, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("SQL structure validation failed: %v", err))
	}
	
	// Validate backup chain for incremental/differential backups
	if metadata.Type == BackupTypeIncremental || metadata.Type == BackupTypeDifferential {
		if err := v.validateBackupChain(ctx, backupID, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Backup chain validation failed: %v", err))
		}
	}
	
	// Perform test restore to temporary database
	if err := v.performTestRestore(ctx, backupID, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Test restore failed: %v", err))
	}
	
	// Determine overall validity
	result.IsValid = len(result.Errors) == 0
	result.ValidationTime = time.Since(startTime)
	
	return result, nil
}

// validateChecksum verifies the backup file checksum
func (v *BackupValidator) validateChecksum(ctx context.Context, backupID, expectedChecksum string, result *ValidationResult) error {
	if expectedChecksum == "" {
		result.Warnings = append(result.Warnings, "No checksum available for verification")
		return nil
	}
	
	// Retrieve backup file
	reader, err := v.storage.Retrieve(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to retrieve backup file: %w", err)
	}
	defer reader.Close()
	
	// Calculate checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	
	if actualChecksum == expectedChecksum {
		result.ChecksumMatch = true
	} else {
		result.ChecksumMatch = false
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	
	return nil
}

// validateSQLStructure validates the SQL syntax and structure
func (v *BackupValidator) validateSQLStructure(ctx context.Context, backupID string, result *ValidationResult) error {
	// Create temporary directory
	tempDir := filepath.Join(v.tempDir, "validation_"+backupID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Retrieve and save backup file
	reader, err := v.storage.Retrieve(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to retrieve backup file: %w", err)
	}
	defer reader.Close()
	
	tempFile := filepath.Join(tempDir, "backup.sql")
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	
	_, err = io.Copy(file, reader)
	file.Close()
	if err != nil {
		return fmt.Errorf("failed to copy backup data: %w", err)
	}
	
	// Parse and validate SQL statements
	return v.parseSQLFile(tempFile, result)
}

// parseSQLFile parses the SQL file and validates its structure
func (v *BackupValidator) parseSQLFile(filePath string, result *ValidationResult) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open SQL file: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var currentStatement strings.Builder
	statementCount := 0
	tableNames := make(map[string]bool)
	
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		
		currentStatement.WriteString(line)
		currentStatement.WriteString(" ")
		
		// Process complete statements
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(currentStatement.String())
			
			// Validate statement syntax
			if err := v.validateSQLStatement(stmt, lineNumber, result); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Line %d: %v", lineNumber, err))
			}
			
			// Extract table information
			if tableName := v.extractTableNameFromStatement(stmt); tableName != "" {
				tableNames[tableName] = true
			}
			
			statementCount++
			currentStatement.Reset()
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}
	
	// Check for incomplete statements
	if currentStatement.Len() > 0 {
		result.Warnings = append(result.Warnings, "Incomplete SQL statement at end of file")
	}
	
	// Validate that we have expected tables
	expectedTables := []string{"users", "sessions", "system_config"} // Add more as needed
	for _, table := range expectedTables {
		if !tableNames[table] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Expected table '%s' not found in backup", table))
		}
	}
	
	return nil
}

// validateSQLStatement validates individual SQL statements
func (v *BackupValidator) validateSQLStatement(stmt string, lineNumber int, result *ValidationResult) error {
	upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
	
	// Check for potentially dangerous statements
	dangerousPatterns := []string{
		"DROP DATABASE",
		"DELETE FROM sqlite_master",
		"PRAGMA user_version",
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(upperStmt, pattern) {
			return fmt.Errorf("potentially dangerous statement: %s", pattern)
		}
	}
	
	// Validate statement structure
	if strings.HasPrefix(upperStmt, "CREATE TABLE") {
		return v.validateCreateTableStatement(stmt)
	} else if strings.HasPrefix(upperStmt, "INSERT INTO") {
		return v.validateInsertStatement(stmt)
	} else if strings.HasPrefix(upperStmt, "DROP TABLE") {
		return v.validateDropTableStatement(stmt)
	}
	
	return nil
}

// validateCreateTableStatement validates CREATE TABLE statements
func (v *BackupValidator) validateCreateTableStatement(stmt string) error {
	// Check for required elements
	if !strings.Contains(strings.ToUpper(stmt), "CREATE TABLE") {
		return fmt.Errorf("invalid CREATE TABLE statement")
	}
	
	// Check for table name
	parts := strings.Fields(stmt)
	if len(parts) < 3 {
		return fmt.Errorf("incomplete CREATE TABLE statement")
	}
	
	// Check for opening parenthesis
	if !strings.Contains(stmt, "(") {
		return fmt.Errorf("CREATE TABLE statement missing column definitions")
	}
	
	return nil
}

// validateInsertStatement validates INSERT statements
func (v *BackupValidator) validateInsertStatement(stmt string) error {
	upperStmt := strings.ToUpper(stmt)
	
	if !strings.Contains(upperStmt, "INSERT INTO") {
		return fmt.Errorf("invalid INSERT statement")
	}
	
	if !strings.Contains(upperStmt, "VALUES") {
		return fmt.Errorf("INSERT statement missing VALUES clause")
	}
	
	return nil
}

// validateDropTableStatement validates DROP TABLE statements
func (v *BackupValidator) validateDropTableStatement(stmt string) error {
	upperStmt := strings.ToUpper(stmt)
	
	if !strings.Contains(upperStmt, "DROP TABLE") {
		return fmt.Errorf("invalid DROP TABLE statement")
	}
	
	// Check for IF EXISTS clause (recommended)
	if !strings.Contains(upperStmt, "IF EXISTS") {
		// This is a warning, not an error
		return nil
	}
	
	return nil
}

// extractTableNameFromStatement extracts table name from SQL statement
func (v *BackupValidator) extractTableNameFromStatement(stmt string) string {
	upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
	
	if strings.HasPrefix(upperStmt, "CREATE TABLE") ||
		strings.HasPrefix(upperStmt, "DROP TABLE") {
		parts := strings.Fields(upperStmt)
		if len(parts) >= 3 {
			tableName := strings.Trim(parts[2], "`'\"")
			return strings.ToLower(tableName)
		}
	} else if strings.HasPrefix(upperStmt, "INSERT INTO") {
		parts := strings.Fields(upperStmt)
		if len(parts) >= 3 {
			tableName := strings.Trim(parts[2], "`'\"")
			return strings.ToLower(tableName)
		}
	}
	
	return ""
}

// validateBackupChain validates the backup chain for incremental/differential backups
func (v *BackupValidator) validateBackupChain(ctx context.Context, backupID string, result *ValidationResult) error {
	metadata, err := v.repository.GetBackupRecord(ctx, backupID)
	if err != nil {
		return fmt.Errorf("failed to get backup metadata: %w", err)
	}
	
	if metadata.ParentBackupID == nil {
		return fmt.Errorf("incremental/differential backup missing parent backup ID")
	}
	
	// Check that parent backup exists and is valid
	parentMetadata, err := v.repository.GetBackupRecord(ctx, *metadata.ParentBackupID)
	if err != nil {
		return fmt.Errorf("failed to get parent backup metadata: %w", err)
	}
	
	if parentMetadata.Status != BackupStatusCompleted {
		return fmt.Errorf("parent backup is not in completed state: %s", parentMetadata.Status)
	}
	
	// Check backup timing
	if metadata.StartTime.Before(parentMetadata.StartTime) {
		return fmt.Errorf("backup timestamp is before parent backup timestamp")
	}
	
	// For differential backups, ensure parent is a full backup
	if metadata.Type == BackupTypeDifferential && parentMetadata.Type != BackupTypeFull {
		return fmt.Errorf("differential backup parent must be a full backup")
	}
	
	return nil
}

// performTestRestore performs a test restore to validate backup integrity
func (v *BackupValidator) performTestRestore(ctx context.Context, backupID string, result *ValidationResult) error {
	// Create temporary database for test restore
	tempDir := filepath.Join(v.tempDir, "test_restore_"+backupID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create temporary database connection
	tempDBPath := filepath.Join(tempDir, "test.db")
	tempDB, err := v.createTempDatabase(tempDBPath)
	if err != nil {
		return fmt.Errorf("failed to create temp database: %w", err)
	}
	defer tempDB.Close()
	
	// Create restore engine with temp database
	restoreEngine := NewRestoreEngine(
		&database.Database{DB: tempDB},
		v.storage,
		v.repository,
	)
	
	// Perform test restore
	restoreOptions := RestoreOptions{
		BackupID:   backupID,
		TargetPath: tempDBPath,
		Overwrite:  true,
		Verify:     false, // Skip verification to avoid recursion
		DryRun:     false,
	}
	
	if err := restoreEngine.RestoreBackup(ctx, restoreOptions); err != nil {
		return fmt.Errorf("test restore failed: %w", err)
	}
	
	// Validate restored data
	if err := v.validateRestoredData(ctx, tempDB, result); err != nil {
		return fmt.Errorf("restored data validation failed: %w", err)
	}
	
	return nil
}

// createTempDatabase creates a temporary database for testing
func (v *BackupValidator) createTempDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp database: %w", err)
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping temp database: %w", err)
	}
	
	return db, nil
}

// validateRestoredData validates the restored data in the temporary database
func (v *BackupValidator) validateRestoredData(ctx context.Context, db *sql.DB, result *ValidationResult) error {
	// Get list of tables
	tables, err := v.getTables(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}
	
	// Count records in each table
	for _, table := range tables {
		count, err := v.getTableRecordCount(ctx, db, table)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to count records in table %s: %v", table, err))
			continue
		}
		
		result.TableCounts[table] = count
		
		// Validate table structure
		if err := v.validateTableStructure(ctx, db, table); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Table structure validation failed for %s: %v", table, err))
		}
	}
	
	return nil
}

// getTables returns list of tables in the database
func (v *BackupValidator) getTables(ctx context.Context, db *sql.DB) ([]string, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	
	rows, err := db.QueryContext(ctx, query)
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

// getTableRecordCount returns the number of records in a table
func (v *BackupValidator) getTableRecordCount(ctx context.Context, db *sql.DB, tableName string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tableName)
	
	var count int64
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	
	return count, nil
}

// validateTableStructure validates the structure of a table
func (v *BackupValidator) validateTableStructure(ctx context.Context, db *sql.DB, tableName string) error {
	// Get table schema
	query := fmt.Sprintf("PRAGMA table_info(`%s`)", tableName)
	
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()
	
	columnCount := 0
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue sql.NullString
		
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		
		columnCount++
		
		// Basic validation - ensure column has a name and type
		if name == "" {
			return fmt.Errorf("column has no name")
		}
		
		if dataType == "" {
			return fmt.Errorf("column %s has no data type", name)
		}
	}
	
	if columnCount == 0 {
		return fmt.Errorf("table has no columns")
	}
	
	return rows.Err()
}

// QuickValidateBackup performs a quick validation without full restore
func (v *BackupValidator) QuickValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error) {
	startTime := time.Now()
	
	result := &ValidationResult{
		BackupID:    backupID,
		IsValid:     false,
		ValidatedAt: startTime,
	}
	
	// Get backup metadata
	metadata, err := v.repository.GetBackupRecord(ctx, backupID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get backup metadata: %v", err))
		result.ValidationTime = time.Since(startTime)
		return result, nil
	}
	
	// Check file existence
	exists, err := v.storage.Exists(ctx, backupID)
	if err != nil || !exists {
		result.Errors = append(result.Errors, "Backup file does not exist")
		result.ValidationTime = time.Since(startTime)
		return result, nil
	}
	
	// Validate checksum only
	if err := v.validateChecksum(ctx, backupID, metadata.Checksum, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Checksum validation failed: %v", err))
	}
	
	result.IsValid = len(result.Errors) == 0
	result.ValidationTime = time.Since(startTime)
	
	return result, nil
}