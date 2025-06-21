package backup

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"qt1-middleware/database"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *database.Database {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			active BOOLEAN NOT NULL DEFAULT true,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);

		CREATE TABLE sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE system_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			value TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
		VALUES 
			('admin', 'admin@example.com', 'hash1', 'admin', datetime('now'), datetime('now')),
			('user1', 'user1@example.com', 'hash2', 'user', datetime('now'), datetime('now')),
			('user2', 'user2@example.com', 'hash3', 'user', datetime('now'), datetime('now'));

		INSERT INTO system_config (key, value, created_at, updated_at)
		VALUES 
			('app_name', 'Test App', datetime('now'), datetime('now')),
			('version', '1.0.0', datetime('now'), datetime('now'));
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return &database.Database{DB: db}
}

func setupTestBackupManager(t *testing.T) (*DefaultBackupManager, string) {
	// Create temporary directory for backups
	tempDir, err := os.MkdirTemp("", "backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Setup test database
	testDB := setupTestDB(t)

	// Create backup manager
	manager, err := NewBackupManager(testDB, tempDir)
	if err != nil {
		t.Fatalf("Failed to create backup manager: %v", err)
	}

	return manager, tempDir
}

func TestBackupManager_CreateFullBackup(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create full backup
	options := BackupOptions{
		Type:            BackupTypeFull,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Test full backup",
		Tags:            []string{"test", "full"},
		Verify:          false, // Skip verification for speed
	}

	metadata, err := manager.CreateBackup(ctx, options)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify metadata
	if metadata.ID == "" {
		t.Error("Backup ID should not be empty")
	}

	if metadata.Type != BackupTypeFull {
		t.Errorf("Expected backup type %s, got %s", BackupTypeFull, metadata.Type)
	}

	if metadata.Status != BackupStatusRunning {
		t.Errorf("Expected initial status %s, got %s", BackupStatusRunning, metadata.Status)
	}

	// Wait for backup to complete
	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		progress, err := manager.GetProgress(ctx, metadata.ID)
		if err != nil {
			t.Fatalf("Failed to get backup progress: %v", err)
		}

		if progress.Status == BackupStatusCompleted {
			break
		} else if progress.Status == BackupStatusFailed {
			t.Fatalf("Backup failed: %v", progress.ErrorMessage)
		}

		time.Sleep(1 * time.Second)
	}

	// Get final metadata
	finalMetadata, err := manager.GetBackupMetadata(ctx, metadata.ID)
	if err != nil {
		t.Fatalf("Failed to get final metadata: %v", err)
	}

	if finalMetadata.Status != BackupStatusCompleted {
		t.Errorf("Expected final status %s, got %s", BackupStatusCompleted, finalMetadata.Status)
	}

	if finalMetadata.Size == 0 {
		t.Error("Backup size should be greater than 0")
	}

	if finalMetadata.Checksum == "" {
		t.Error("Backup checksum should not be empty")
	}
}

func TestBackupManager_RestoreBackup(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create a backup first
	backupOptions := BackupOptions{
		Type:            BackupTypeFull,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Test backup for restore",
		Verify:          false,
	}

	metadata, err := manager.CreateBackup(ctx, backupOptions)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Wait for backup to complete
	for i := 0; i < 30; i++ {
		progress, err := manager.GetProgress(ctx, metadata.ID)
		if err != nil {
			t.Fatalf("Failed to get backup progress: %v", err)
		}

		if progress.Status == BackupStatusCompleted {
			break
		} else if progress.Status == BackupStatusFailed {
			t.Fatalf("Backup failed: %v", progress.ErrorMessage)
		}

		time.Sleep(1 * time.Second)
	}

	// Perform dry run restore first
	restoreOptions := RestoreOptions{
		BackupID:  metadata.ID,
		Overwrite: true,
		Verify:    false,
		DryRun:    true,
	}

	err = manager.RestoreBackup(ctx, restoreOptions)
	if err != nil {
		t.Fatalf("Dry run restore failed: %v", err)
	}

	// Perform actual restore
	restoreOptions.DryRun = false
	err = manager.RestoreBackup(ctx, restoreOptions)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
}

func TestBackupManager_ValidateBackup(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create a backup
	options := BackupOptions{
		Type:            BackupTypeFull,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Test backup for validation",
		Verify:          false,
	}

	metadata, err := manager.CreateBackup(ctx, options)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Wait for backup to complete
	for i := 0; i < 30; i++ {
		progress, err := manager.GetProgress(ctx, metadata.ID)
		if err != nil {
			t.Fatalf("Failed to get backup progress: %v", err)
		}

		if progress.Status == BackupStatusCompleted {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Validate backup
	result, err := manager.ValidateBackup(ctx, metadata.ID)
	if err != nil {
		t.Fatalf("Failed to validate backup: %v", err)
	}

	if !result.IsValid {
		t.Errorf("Backup validation failed: %v", result.Errors)
	}

	if !result.ChecksumMatch {
		t.Error("Backup checksum validation failed")
	}

	if len(result.TableCounts) == 0 {
		t.Error("No table counts in validation result")
	}
}

func TestBackupManager_IncrementalBackup(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create full backup first
	fullOptions := BackupOptions{
		Type:            BackupTypeFull,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Base full backup",
		Verify:          false,
	}

	fullMetadata, err := manager.CreateBackup(ctx, fullOptions)
	if err != nil {
		t.Fatalf("Failed to create full backup: %v", err)
	}

	// Wait for full backup to complete
	for i := 0; i < 30; i++ {
		progress, err := manager.GetProgress(ctx, fullMetadata.ID)
		if err != nil {
			t.Fatalf("Failed to get backup progress: %v", err)
		}

		if progress.Status == BackupStatusCompleted {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Add some new data
	// (In a real test, you'd modify the database here)

	// Create incremental backup
	incOptions := BackupOptions{
		Type:            BackupTypeIncremental,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Incremental backup",
		ParentBackupID:  &fullMetadata.ID,
		Verify:          false,
	}

	incMetadata, err := manager.CreateBackup(ctx, incOptions)
	if err != nil {
		t.Fatalf("Failed to create incremental backup: %v", err)
	}

	// Wait for incremental backup to complete
	for i := 0; i < 30; i++ {
		progress, err := manager.GetProgress(ctx, incMetadata.ID)
		if err != nil {
			t.Fatalf("Failed to get backup progress: %v", err)
		}

		if progress.Status == BackupStatusCompleted {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Verify incremental backup metadata
	finalMetadata, err := manager.GetBackupMetadata(ctx, incMetadata.ID)
	if err != nil {
		t.Fatalf("Failed to get incremental backup metadata: %v", err)
	}

	if finalMetadata.Type != BackupTypeIncremental {
		t.Errorf("Expected backup type %s, got %s", BackupTypeIncremental, finalMetadata.Type)
	}

	if finalMetadata.ParentBackupID == nil || *finalMetadata.ParentBackupID != fullMetadata.ID {
		t.Error("Incremental backup should reference parent backup ID")
	}
}

func TestBackupManager_Scheduling(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create a test schedule
	schedule := &BackupSchedule{
		Name:           "Test Daily Backup",
		Description:    "Daily backup for testing",
		CronExpression: "0 2 * * *", // Daily at 2 AM
		BackupOptions: BackupOptions{
			Type:            BackupTypeFull,
			CompressionType: CompressionNone,
			EncryptionType:  EncryptionNone,
			Description:     "Scheduled backup",
		},
		RetentionPolicy: RetentionPolicy{
			DailyRetention:   7,
			WeeklyRetention:  4,
			MonthlyRetention: 12,
			YearlyRetention:  5,
		},
		Enabled:   true,
		CreatedBy: "test",
	}

	// Create schedule
	err := manager.CreateSchedule(ctx, schedule)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}

	// Verify schedule was created
	retrievedSchedule, err := manager.GetSchedule(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("Failed to get schedule: %v", err)
	}

	if retrievedSchedule.Name != schedule.Name {
		t.Errorf("Expected schedule name %s, got %s", schedule.Name, retrievedSchedule.Name)
	}

	if retrievedSchedule.CronExpression != schedule.CronExpression {
		t.Errorf("Expected cron expression %s, got %s", schedule.CronExpression, retrievedSchedule.CronExpression)
	}

	// List schedules
	schedules, err := manager.ListSchedules(ctx)
	if err != nil {
		t.Fatalf("Failed to list schedules: %v", err)
	}

	if len(schedules) != 1 {
		t.Errorf("Expected 1 schedule, got %d", len(schedules))
	}

	// Update schedule
	schedule.Description = "Updated description"
	err = manager.UpdateSchedule(ctx, schedule)
	if err != nil {
		t.Fatalf("Failed to update schedule: %v", err)
	}

	// Delete schedule
	err = manager.DeleteSchedule(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("Failed to delete schedule: %v", err)
	}

	// Verify schedule was deleted
	_, err = manager.GetSchedule(ctx, schedule.ID)
	if err == nil {
		t.Error("Expected error when getting deleted schedule")
	}
}

func TestBackupManager_ListBackups(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create multiple backups
	for i := 0; i < 3; i++ {
		options := BackupOptions{
			Type:            BackupTypeFull,
			CompressionType: CompressionNone,
			EncryptionType:  EncryptionNone,
			Description:     fmt.Sprintf("Test backup %d", i+1),
			Tags:            []string{"test", fmt.Sprintf("backup%d", i+1)},
			Verify:          false,
		}

		_, err := manager.CreateBackup(ctx, options)
		if err != nil {
			t.Fatalf("Failed to create backup %d: %v", i+1, err)
		}

		// Small delay between backups
		time.Sleep(100 * time.Millisecond)
	}

	// List all backups
	allBackups, err := manager.ListBackups(ctx, BackupFilters{})
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	if len(allBackups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(allBackups))
	}

	// List with filters
	filters := BackupFilters{
		Type:   &[]BackupType{BackupTypeFull}[0],
		Limit:  2,
		Offset: 0,
	}

	filteredBackups, err := manager.ListBackups(ctx, filters)
	if err != nil {
		t.Fatalf("Failed to list filtered backups: %v", err)
	}

	if len(filteredBackups) > 2 {
		t.Errorf("Expected at most 2 backups, got %d", len(filteredBackups))
	}
}

func TestBackupManager_HealthAndStatistics(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Get health status
	health, err := manager.GetHealthStatus(ctx)
	if err != nil {
		t.Fatalf("Failed to get health status: %v", err)
	}

	if health.RunningOperations < 0 {
		t.Error("Running operations count should be non-negative")
	}

	// Get statistics
	stats, err := manager.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if stats.BackupsByType == nil {
		t.Error("Backups by type should not be nil")
	}

	if stats.BackupsByStatus == nil {
		t.Error("Backups by status should not be nil")
	}
}

func TestBackupValidation_SQLStructure(t *testing.T) {
	manager, tempDir := setupTestBackupManager(t)
	defer os.RemoveAll(tempDir)

	// Create a test SQL file with various statements
	testSQL := `
-- Test backup file
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

-- Table structure for users
DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at DATETIME NOT NULL
);

-- Data for users
INSERT INTO users (username, email, created_at) VALUES ('test', 'test@example.com', '2023-01-01 00:00:00');

COMMIT;
PRAGMA foreign_keys=ON;
`

	// Create temporary SQL file
	tempFile := filepath.Join(tempDir, "test.sql")
	err := os.WriteFile(tempFile, []byte(testSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create test SQL file: %v", err)
	}

	// Create a mock validation result
	validator := NewBackupValidator(manager.db, manager.storage, manager.repository)
	result := &ValidationResult{
		BackupID:    "test",
		TableCounts: make(map[string]int64),
	}

	// Test SQL parsing
	err = validator.parseSQLFile(tempFile, result)
	if err != nil {
		t.Fatalf("Failed to parse SQL file: %v", err)
	}

	if len(result.Warnings) > 0 {
		t.Logf("Validation warnings: %v", result.Warnings)
	}
}

func TestFileSystemStorage(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage
	storage, err := NewFileSystemBackupStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()
	backupID := "test-backup-123"
	testData := "This is test backup data"

	// Test Store
	path, err := storage.Store(ctx, backupID, strings.NewReader(testData))
	if err != nil {
		t.Fatalf("Failed to store backup: %v", err)
	}

	if path == "" {
		t.Error("Storage path should not be empty")
	}

	// Test Exists
	exists, err := storage.Exists(ctx, backupID)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}

	if !exists {
		t.Error("Backup should exist after storing")
	}

	// Test GetSize
	size, err := storage.GetSize(ctx, backupID)
	if err != nil {
		t.Fatalf("Failed to get size: %v", err)
	}

	if size != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), size)
	}

	// Test Retrieve
	reader, err := storage.Retrieve(ctx, backupID)
	if err != nil {
		t.Fatalf("Failed to retrieve backup: %v", err)
	}
	defer reader.Close()

	retrievedData, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read retrieved data: %v", err)
	}

	if string(retrievedData) != testData {
		t.Errorf("Retrieved data doesn't match original: expected %s, got %s", testData, string(retrievedData))
	}

	// Test ListBackups
	backupIDs, err := storage.ListBackups(ctx)
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	found := false
	for _, id := range backupIDs {
		if id == backupID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Backup ID should be in the list")
	}

	// Test Delete
	err = storage.Delete(ctx, backupID)
	if err != nil {
		t.Fatalf("Failed to delete backup: %v", err)
	}

	// Verify deletion
	exists, err = storage.Exists(ctx, backupID)
	if err != nil {
		t.Fatalf("Failed to check existence after deletion: %v", err)
	}

	if exists {
		t.Error("Backup should not exist after deletion")
	}
}

// Benchmark tests

func BenchmarkBackupCreation(b *testing.B) {
	manager, tempDir := setupTestBackupManager(&testing.T{})
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	options := BackupOptions{
		Type:            BackupTypeFull,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Benchmark backup",
		Verify:          false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.CreateBackup(ctx, options)
		if err != nil {
			b.Fatalf("Failed to create backup: %v", err)
		}
	}
}

func BenchmarkBackupValidation(b *testing.B) {
	manager, tempDir := setupTestBackupManager(&testing.T{})
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Create a backup first
	options := BackupOptions{
		Type:            BackupTypeFull,
		CompressionType: CompressionNone,
		EncryptionType:  EncryptionNone,
		Description:     "Benchmark validation backup",
		Verify:          false,
	}

	metadata, err := manager.CreateBackup(ctx, options)
	if err != nil {
		b.Fatalf("Failed to create backup: %v", err)
	}

	// Wait for completion
	for i := 0; i < 30; i++ {
		progress, err := manager.GetProgress(ctx, metadata.ID)
		if err != nil {
			b.Fatalf("Failed to get progress: %v", err)
		}
		if progress.Status == BackupStatusCompleted {
			break
		}
		time.Sleep(1 * time.Second)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.ValidateBackup(ctx, metadata.ID)
		if err != nil {
			b.Fatalf("Failed to validate backup: %v", err)
		}
	}
}