package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrationRunner_Initialize(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()
	
	config := &MigrationConfig{
		Enabled:   true,
		Directory: "./test_migrations",
		Table:     "test_migrations",
	}
	
	runner := NewMigrationRunner(db, config)
	
	err := runner.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize migration runner: %v", err)
	}
	
	if !runner.initialized {
		t.Error("Migration runner should be initialized")
	}
	
	// Verify migration table was created
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", config.Table).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration table: %v", err)
	}
	
	if count != 1 {
		t.Error("Migration table should be created")
	}
}

func TestMigrationRunner_LoadMigrations(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()
	
	// Create temporary migration directory
	tempDir := "/tmp/test_migrations"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)
	
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	// Create test migration files
	upSQL := "CREATE TABLE test_table (id INTEGER PRIMARY KEY);"
	downSQL := "DROP TABLE test_table;"
	
	err = os.WriteFile(filepath.Join(tempDir, "001_test_migration.up.sql"), []byte(upSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create up migration: %v", err)
	}
	
	err = os.WriteFile(filepath.Join(tempDir, "001_test_migration.down.sql"), []byte(downSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create down migration: %v", err)
	}
	
	config := &MigrationConfig{
		Enabled:   true,
		Directory: tempDir,
		Table:     "test_migrations",
	}
	
	runner := NewMigrationRunner(db, config)
	err = runner.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize migration runner: %v", err)
	}
	
	err = runner.LoadMigrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}
	
	if len(runner.migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(runner.migrations))
	}
	
	migration := runner.migrations[0]
	if migration.Version != 1 {
		t.Errorf("Expected version 1, got %d", migration.Version)
	}
	
	if migration.Name != "test migration" {
		t.Errorf("Expected name 'test migration', got '%s'", migration.Name)
	}
	
	if migration.UpSQL != upSQL {
		t.Error("Up SQL doesn't match")
	}
	
	if migration.DownSQL != downSQL {
		t.Error("Down SQL doesn't match")
	}
}

func TestMigrationRunner_ApplyMigration(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()
	
	// Create temporary migration directory
	tempDir := "/tmp/test_migrations_apply"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)
	
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	// Create test migration files
	upSQL := "CREATE TABLE test_apply (id INTEGER PRIMARY KEY, name TEXT);"
	downSQL := "DROP TABLE test_apply;"
	
	err = os.WriteFile(filepath.Join(tempDir, "001_test_apply.up.sql"), []byte(upSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create up migration: %v", err)
	}
	
	err = os.WriteFile(filepath.Join(tempDir, "001_test_apply.down.sql"), []byte(downSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create down migration: %v", err)
	}
	
	config := &MigrationConfig{
		Enabled:   true,
		Directory: tempDir,
		Table:     "test_migrations",
	}
	
	runner := NewMigrationRunner(db, config)
	err = runner.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize migration runner: %v", err)
	}
	
	err = runner.LoadMigrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}
	
	// Apply migration
	migration := runner.migrations[0]
	err = runner.ApplyMigration(migration)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}
	
	// Verify table was created
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_apply'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check table: %v", err)
	}
	
	if count != 1 {
		t.Error("Test table should be created")
	}
	
	// Verify migration was recorded
	err = db.DB.QueryRow("SELECT COUNT(*) FROM test_migrations WHERE version = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration record: %v", err)
	}
	
	if count != 1 {
		t.Error("Migration should be recorded")
	}
}

func TestMigrationRunner_RollbackMigration(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()
	
	// Create temporary migration directory
	tempDir := "/tmp/test_migrations_rollback"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)
	
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	// Create test migration files
	upSQL := "CREATE TABLE test_rollback (id INTEGER PRIMARY KEY, data TEXT);"
	downSQL := "DROP TABLE test_rollback;"
	
	err = os.WriteFile(filepath.Join(tempDir, "001_test_rollback.up.sql"), []byte(upSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create up migration: %v", err)
	}
	
	err = os.WriteFile(filepath.Join(tempDir, "001_test_rollback.down.sql"), []byte(downSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to create down migration: %v", err)
	}
	
	config := &MigrationConfig{
		Enabled:   true,
		Directory: tempDir,
		Table:     "test_migrations",
	}
	
	runner := NewMigrationRunner(db, config)
	err = runner.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize migration runner: %v", err)
	}
	
	err = runner.LoadMigrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}
	
	// Apply migration first
	migration := runner.migrations[0]
	err = runner.ApplyMigration(migration)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}
	
	// Reload migrations to update applied status
	err = runner.LoadMigrations()
	if err != nil {
		t.Fatalf("Failed to reload migrations: %v", err)
	}
	
	// Verify table exists
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_rollback'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check table: %v", err)
	}
	
	if count != 1 {
		t.Error("Test table should exist before rollback")
	}
	
	// Rollback migration
	err = runner.RollbackMigration(1)
	if err != nil {
		t.Fatalf("Failed to rollback migration: %v", err)
	}
	
	// Verify table was dropped
	err = db.DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_rollback'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check table after rollback: %v", err)
	}
	
	if count != 0 {
		t.Error("Test table should be dropped after rollback")
	}
	
	// Verify migration record was removed
	err = db.DB.QueryRow("SELECT COUNT(*) FROM test_migrations WHERE version = 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check migration record after rollback: %v", err)
	}
	
	if count != 0 {
		t.Error("Migration record should be removed after rollback")
	}
}

func TestMigrationRunner_GetMigrationStatus(t *testing.T) {
	db := createTestDatabase(t)
	defer db.Close()
	
	// Create temporary migration directory
	tempDir := "/tmp/test_migrations_status"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)
	
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	// Create multiple test migration files
	migrations := []struct {
		version int
		name    string
		upSQL   string
		downSQL string
	}{
		{1, "create_users", "CREATE TABLE users (id INTEGER PRIMARY KEY);", "DROP TABLE users;"},
		{2, "add_email", "ALTER TABLE users ADD COLUMN email TEXT;", "ALTER TABLE users DROP COLUMN email;"},
	}
	
	for _, mig := range migrations {
		upFile := filepath.Join(tempDir, fmt.Sprintf("%03d_%s.up.sql", mig.version, mig.name))
		downFile := filepath.Join(tempDir, fmt.Sprintf("%03d_%s.down.sql", mig.version, mig.name))
		
		err = os.WriteFile(upFile, []byte(mig.upSQL), 0644)
		if err != nil {
			t.Fatalf("Failed to create up migration %d: %v", mig.version, err)
		}
		
		err = os.WriteFile(downFile, []byte(mig.downSQL), 0644)
		if err != nil {
			t.Fatalf("Failed to create down migration %d: %v", mig.version, err)
		}
	}
	
	config := &MigrationConfig{
		Enabled:   true,
		Directory: tempDir,
		Table:     "test_migrations",
	}
	
	runner := NewMigrationRunner(db, config)
	err = runner.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize migration runner: %v", err)
	}
	
	// Get status before applying any migrations
	status, err := runner.GetMigrationStatus()
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}
	
	if status["total_migrations"] != 2 {
		t.Errorf("Expected 2 total migrations, got %v", status["total_migrations"])
	}
	
	if status["applied_migrations"] != 0 {
		t.Errorf("Expected 0 applied migrations, got %v", status["applied_migrations"])
	}
	
	if status["pending_migrations"] != 2 {
		t.Errorf("Expected 2 pending migrations, got %v", status["pending_migrations"])
	}
	
	if status["up_to_date"] != false {
		t.Error("Should not be up to date with pending migrations")
	}
	
	// Apply first migration
	err = runner.LoadMigrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}
	
	err = runner.ApplyMigration(runner.migrations[0])
	if err != nil {
		t.Fatalf("Failed to apply first migration: %v", err)
	}
	
	// Get status after applying one migration
	status, err = runner.GetMigrationStatus()
	if err != nil {
		t.Fatalf("Failed to get migration status after apply: %v", err)
	}
	
	if status["applied_migrations"] != 1 {
		t.Errorf("Expected 1 applied migration, got %v", status["applied_migrations"])
	}
	
	if status["pending_migrations"] != 1 {
		t.Errorf("Expected 1 pending migration, got %v", status["pending_migrations"])
	}
	
	if status["up_to_date"] != false {
		t.Error("Should not be up to date with one pending migration")
	}
}

// Helper function to create a test database
func createTestDatabase(t *testing.T) *Database {
	config := &DatabaseConfig{
		Type:       "sqlite",
		SQLiteFile: ":memory:",
		EnableWAL:  false,
		EnableForeignKeys: true,
		MaxConnections:     1,
		MaxIdleConnections: 1,
		ConnectionLifetime: time.Hour,
		Migrations: MigrationConfig{
			Enabled:   true,
			Directory: "./migrations",
			Table:     "schema_migrations",
		},
	}
	
	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Don't run migrations automatically for tests
	config.Migrations.Enabled = false
	
	err = db.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	
	return db
}