package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"        // PostgreSQL driver
	_ "modernc.org/sqlite"       // SQLite driver
)

// Database represents a database connection with management features
type Database struct {
	DB          *sql.DB
	config      *DatabaseConfig
	health      *HealthChecker
	migrations  *MigrationRunner
	isConnected bool
	connectedAt time.Time
}

// NewDatabase creates a new database instance with the given configuration
func NewDatabase(config *DatabaseConfig) (*Database, error) {
	if config == nil {
		return nil, fmt.Errorf("database config is required")
	}
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}
	
	db := &Database{
		config: config,
	}
	
	return db, nil
}

// Connect establishes a connection to the database
func (d *Database) Connect() error {
	if d.isConnected {
		return nil
	}
	
	// Create directory for SQLite if needed
	if d.config.Type == "sqlite" {
		dir := filepath.Dir(d.config.SQLiteFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for SQLite file: %w", err)
		}
	}
	
	// Open database connection
	connStr := d.config.GetConnectionString()
	driverName := d.config.GetDriverName()
	
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	
	// Configure connection pool
	if err := d.configureConnectionPool(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to configure connection pool: %w", err)
	}
	
	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Set up database instance
	d.DB = db
	d.isConnected = true
	d.connectedAt = time.Now()
	d.health = NewHealthChecker(db, d.config)
	d.migrations = NewMigrationRunner(d, &d.config.Migrations)
	
	// Perform database-specific initialization
	if err := d.initializeDatabase(); err != nil {
		d.Close()
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Run migrations if enabled
	if d.config.Migrations.Enabled {
		if err := d.migrations.RunMigrations(); err != nil {
			d.Close()
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}
	
	return nil
}

// configureConnectionPool sets up the database connection pool
func (d *Database) configureConnectionPool(db *sql.DB) error {
	// Set maximum number of open connections
	db.SetMaxOpenConns(d.config.MaxConnections)
	
	// Set maximum number of idle connections
	db.SetMaxIdleConns(d.config.MaxIdleConnections)
	
	// Set maximum lifetime of connections
	db.SetConnMaxLifetime(d.config.ConnectionLifetime)
	
	// For SQLite, we typically want only 1 connection to avoid locking issues
	if d.config.Type == "sqlite" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}
	
	return nil
}

// initializeDatabase performs database-specific initialization
func (d *Database) initializeDatabase() error {
	switch d.config.Type {
	case "sqlite":
		return d.initializeSQLite()
	case "postgresql":
		return d.initializePostgreSQL()
	default:
		return fmt.Errorf("unsupported database type: %s", d.config.Type)
	}
}

// initializeSQLite performs SQLite-specific initialization
func (d *Database) initializeSQLite() error {
	// Enable WAL mode if configured
	if d.config.EnableWAL {
		if _, err := d.DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			return fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}
	
	// Enable foreign key constraints if configured
	if d.config.EnableForeignKeys {
		if _, err := d.DB.Exec("PRAGMA foreign_keys=ON"); err != nil {
			return fmt.Errorf("failed to enable foreign key constraints: %w", err)
		}
	}
	
	// Set other SQLite-specific settings for better performance
	settings := []string{
		"PRAGMA synchronous=NORMAL",     // Balance between safety and performance
		"PRAGMA cache_size=-64000",      // Use 64MB cache
		"PRAGMA temp_store=MEMORY",      // Store temporary tables in memory
		"PRAGMA mmap_size=134217728",    // Use 128MB memory-mapped I/O
	}
	
	for _, setting := range settings {
		if _, err := d.DB.Exec(setting); err != nil {
			// Log warning but don't fail - these are optimizations
			fmt.Printf("Warning: Failed to set SQLite setting '%s': %v\n", setting, err)
		}
	}
	
	return nil
}

// initializePostgreSQL performs PostgreSQL-specific initialization
func (d *Database) initializePostgreSQL() error {
	// Test that we can query basic information
	var version string
	err := d.DB.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}
	
	// Set timezone to UTC for consistency
	if _, err := d.DB.Exec("SET timezone = 'UTC'"); err != nil {
		return fmt.Errorf("failed to set timezone: %w", err)
	}
	
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if !d.isConnected || d.DB == nil {
		return nil
	}
	
	err := d.DB.Close()
	d.isConnected = false
	d.DB = nil
	d.health = nil
	
	return err
}

// IsConnected returns true if the database is connected
func (d *Database) IsConnected() bool {
	return d.isConnected && d.DB != nil
}

// Ping tests the database connection
func (d *Database) Ping() error {
	if !d.isConnected {
		return fmt.Errorf("database is not connected")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return d.DB.PingContext(ctx)
}

// HealthCheck returns the current health status of the database
func (d *Database) HealthCheck() *DatabaseHealth {
	if d.health == nil {
		return &DatabaseHealth{
			Status:    "unhealthy",
			Connected: false,
			CheckedAt: time.Now(),
			LastError: "database not initialized",
		}
	}
	
	return d.health.Check()
}

// GetConfig returns the database configuration
func (d *Database) GetConfig() *DatabaseConfig {
	return d.config
}

// GetConnectionStats returns database connection statistics
func (d *Database) GetConnectionStats() sql.DBStats {
	if d.DB == nil {
		return sql.DBStats{}
	}
	return d.DB.Stats()
}

// GetUptime returns how long the database has been connected
func (d *Database) GetUptime() time.Duration {
	if !d.isConnected {
		return 0
	}
	return time.Since(d.connectedAt)
}

// BeginTx starts a new database transaction with options
func (d *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if !d.isConnected {
		return nil, fmt.Errorf("database is not connected")
	}
	
	return d.DB.BeginTx(ctx, opts)
}

// Begin starts a new database transaction
func (d *Database) Begin() (*sql.Tx, error) {
	if !d.isConnected {
		return nil, fmt.Errorf("database is not connected")
	}
	
	return d.DB.Begin()
}

// ExecContext executes a query without returning any rows
func (d *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !d.isConnected {
		return nil, fmt.Errorf("database is not connected")
	}
	
	return d.DB.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (d *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !d.isConnected {
		return nil, fmt.Errorf("database is not connected")
	}
	
	return d.DB.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row
func (d *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if !d.isConnected {
		// Return a row that will return an error when scanned
		return &sql.Row{}
	}
	
	return d.DB.QueryRowContext(ctx, query, args...)
}

// PrepareContext creates a prepared statement for later queries or executions
func (d *Database) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if !d.isConnected {
		return nil, fmt.Errorf("database is not connected")
	}
	
	return d.DB.PrepareContext(ctx, query)
}

// GetDatabaseType returns the type of database (sqlite, postgresql)
func (d *Database) GetDatabaseType() string {
	return d.config.Type
}

// IsReady returns true if the database is connected and ready for queries
func (d *Database) IsReady() bool {
	if !d.isConnected {
		return false
	}
	
	// Perform a quick health check
	return d.health != nil && d.health.IsHealthy()
}

// GetMigrations returns the migration runner
func (d *Database) GetMigrations() *MigrationRunner {
	return d.migrations
}

// GetMigrationStatus returns the current migration status
func (d *Database) GetMigrationStatus() (map[string]interface{}, error) {
	if d.migrations == nil {
		return nil, fmt.Errorf("migrations not initialized")
	}
	return d.migrations.GetMigrationStatus()
}