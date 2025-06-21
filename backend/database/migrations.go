package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a single database migration
type Migration struct {
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	UpSQL     string    `json:"-"`
	DownSQL   string    `json:"-"`
	Timestamp time.Time `json:"timestamp"`
	Applied   bool      `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db          *Database
	config      *MigrationConfig
	migrations  []Migration
	initialized bool
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *Database, config *MigrationConfig) *MigrationRunner {
	if config == nil {
		config = &MigrationConfig{
			Enabled:   true,
			Directory: "./database/migrations",
			Table:     "schema_migrations",
		}
	}
	
	return &MigrationRunner{
		db:     db,
		config: config,
	}
}

// Initialize sets up the migration tracking table
func (mr *MigrationRunner) Initialize() error {
	if mr.initialized {
		return nil
	}
	
	if !mr.db.IsConnected() {
		return fmt.Errorf("database is not connected")
	}
	
	// Create migration tracking table
	createTableSQL := mr.getCreateMigrationTableSQL()
	
	_, err := mr.db.DB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}
	
	mr.initialized = true
	return nil
}

// LoadMigrations discovers and loads migration files from the migrations directory
func (mr *MigrationRunner) LoadMigrations() error {
	mr.migrations = nil
	
	// Check if migrations directory exists
	dbType := mr.db.GetDatabaseType()
	
	// First try database-specific migrations, then fall back to generic ones
	migrationPatterns := []string{
		filepath.Join(mr.config.Directory, fmt.Sprintf("*_%s.*.sql", dbType)), // e.g., 001_init_sqlite.up.sql
		filepath.Join(mr.config.Directory, "*.sql"),                           // e.g., 001_init.up.sql
	}
	
	var migrationFiles []string
	for _, pattern := range migrationPatterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		migrationFiles = append(migrationFiles, files...)
	}
	
	if len(migrationFiles) == 0 {
		return fmt.Errorf("no migration files found in %s", mr.config.Directory)
	}
	
	// Group migration files by version (up/down pairs)
	migrationMap := make(map[int]*Migration)
	
	for _, file := range migrationFiles {
		migration, isUp, err := mr.parseMigrationFile(file)
		if err != nil {
			continue // Skip invalid files
		}
		
		if existing, exists := migrationMap[migration.Version]; exists {
			if isUp {
				existing.UpSQL = migration.UpSQL
				existing.Name = migration.Name
			} else {
				existing.DownSQL = migration.DownSQL
			}
		} else {
			migrationMap[migration.Version] = migration
		}
	}
	
	// Convert map to sorted slice
	versions := make([]int, 0, len(migrationMap))
	for version := range migrationMap {
		versions = append(versions, version)
	}
	sort.Ints(versions)
	
	for _, version := range versions {
		mr.migrations = append(mr.migrations, *migrationMap[version])
	}
	
	// Load applied migration status from database
	if err := mr.loadAppliedMigrations(); err != nil {
		return fmt.Errorf("failed to load applied migrations: %w", err)
	}
	
	return nil
}

// RunMigrations executes all pending migrations
func (mr *MigrationRunner) RunMigrations() error {
	if !mr.config.Enabled {
		return nil
	}
	
	if err := mr.Initialize(); err != nil {
		return err
	}
	
	if err := mr.LoadMigrations(); err != nil {
		return err
	}
	
	pendingMigrations := mr.GetPendingMigrations()
	if len(pendingMigrations) == 0 {
		return nil
	}
	
	fmt.Printf("Running %d pending migrations...\n", len(pendingMigrations))
	
	for _, migration := range pendingMigrations {
		if err := mr.ApplyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Name)
	}
	
	fmt.Printf("Successfully applied %d migrations\n", len(pendingMigrations))
	return nil
}

// ApplyMigration executes a single migration
func (mr *MigrationRunner) ApplyMigration(migration Migration) error {
	if migration.UpSQL == "" {
		return fmt.Errorf("migration %d has no up SQL", migration.Version)
	}
	
	// Start transaction
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Execute migration SQL
	_, err = tx.Exec(migration.UpSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}
	
	// Record migration as applied  
	now := time.Now()
	var insertSQL string
	if mr.db.GetDatabaseType() == "postgresql" {
		insertSQL = fmt.Sprintf("INSERT INTO %s (version, name, applied_at) VALUES ($1, $2, $3)", mr.config.Table)
	} else {
		insertSQL = fmt.Sprintf("INSERT INTO %s (version, name, applied_at) VALUES (?, ?, ?)", mr.config.Table)
	}
	
	_, err = tx.Exec(insertSQL, migration.Version, migration.Name, now)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}
	
	return nil
}

// RollbackMigration rolls back a single migration
func (mr *MigrationRunner) RollbackMigration(version int) error {
	migration := mr.findMigration(version)
	if migration == nil {
		return fmt.Errorf("migration %d not found", version)
	}
	
	if !migration.Applied {
		return fmt.Errorf("migration %d is not applied", version)
	}
	
	if migration.DownSQL == "" {
		return fmt.Errorf("migration %d has no down SQL", version)
	}
	
	// Start transaction
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Execute rollback SQL
	_, err = tx.Exec(migration.DownSQL)
	if err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}
	
	// Remove migration record
	var deleteSQL string
	if mr.db.GetDatabaseType() == "postgresql" {
		deleteSQL = fmt.Sprintf("DELETE FROM %s WHERE version = $1", mr.config.Table)
	} else {
		deleteSQL = fmt.Sprintf("DELETE FROM %s WHERE version = ?", mr.config.Table)
	}
	
	_, err = tx.Exec(deleteSQL, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}
	
	fmt.Printf("Rolled back migration %d: %s\n", migration.Version, migration.Name)
	return nil
}

// GetPendingMigrations returns migrations that haven't been applied
func (mr *MigrationRunner) GetPendingMigrations() []Migration {
	var pending []Migration
	for _, migration := range mr.migrations {
		if !migration.Applied {
			pending = append(pending, migration)
		}
	}
	return pending
}

// GetAppliedMigrations returns migrations that have been applied
func (mr *MigrationRunner) GetAppliedMigrations() []Migration {
	var applied []Migration
	for _, migration := range mr.migrations {
		if migration.Applied {
			applied = append(applied, migration)
		}
	}
	return applied
}

// GetMigrationStatus returns the current migration status
func (mr *MigrationRunner) GetMigrationStatus() (map[string]interface{}, error) {
	if err := mr.LoadMigrations(); err != nil {
		return nil, err
	}
	
	pending := mr.GetPendingMigrations()
	applied := mr.GetAppliedMigrations()
	
	return map[string]interface{}{
		"total_migrations":   len(mr.migrations),
		"applied_migrations": len(applied),
		"pending_migrations": len(pending),
		"migrations":         mr.migrations,
		"up_to_date":        len(pending) == 0,
	}, nil
}

// parseMigrationFile extracts migration information from a filename and content
func (mr *MigrationRunner) parseMigrationFile(filepath string) (*Migration, bool, error) {
	filename := filepath[strings.LastIndex(filepath, "/")+1:]
	
	// Expected format: 001_migration_name.up.sql or 001_migration_name.down.sql
	re := regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return nil, false, fmt.Errorf("invalid migration filename format: %s", filename)
	}
	
	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, false, fmt.Errorf("invalid version number: %s", matches[1])
	}
	
	name := strings.ReplaceAll(matches[2], "_", " ")
	isUp := matches[3] == "up"
	
	// Read file content
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read migration file: %w", err)
	}
	
	migration := &Migration{
		Version:   version,
		Name:      name,
		Timestamp: time.Now(), // This will be updated when we read file info
	}
	
	if isUp {
		migration.UpSQL = string(content)
	} else {
		migration.DownSQL = string(content)
	}
	
	return migration, isUp, nil
}

// loadAppliedMigrations loads the status of applied migrations from the database
func (mr *MigrationRunner) loadAppliedMigrations() error {
	if !mr.initialized {
		return nil
	}
	
	rows, err := mr.db.DB.Query(
		fmt.Sprintf("SELECT version, name, applied_at FROM %s ORDER BY version", mr.config.Table),
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	appliedMap := make(map[int]*time.Time)
	for rows.Next() {
		var version int
		var name string
		var appliedAt time.Time
		
		if err := rows.Scan(&version, &name, &appliedAt); err != nil {
			return err
		}
		
		appliedMap[version] = &appliedAt
	}
	
	// Update migration status
	for i := range mr.migrations {
		if appliedAt, exists := appliedMap[mr.migrations[i].Version]; exists {
			mr.migrations[i].Applied = true
			mr.migrations[i].AppliedAt = appliedAt
		}
	}
	
	return nil
}

// findMigration finds a migration by version
func (mr *MigrationRunner) findMigration(version int) *Migration {
	for i := range mr.migrations {
		if mr.migrations[i].Version == version {
			return &mr.migrations[i]
		}
	}
	return nil
}

// getCreateMigrationTableSQL returns the SQL to create the migration tracking table
func (mr *MigrationRunner) getCreateMigrationTableSQL() string {
	switch mr.db.GetDatabaseType() {
	case "sqlite":
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`, mr.config.Table)
	case "postgresql":
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT NOW()
			)
		`, mr.config.Table)
	default:
		return ""
	}
}