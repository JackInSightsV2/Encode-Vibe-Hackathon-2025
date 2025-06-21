package database

import (
	"os"
	"testing"
	"time"
)

func TestNewDatabase(t *testing.T) {
	config := &DatabaseConfig{
		Type:       "sqlite",
		SQLiteFile: ":memory:",
		EnableWAL:  false, // WAL doesn't work with in-memory databases
		EnableForeignKeys: true,
		MaxConnections:     1,
		MaxIdleConnections: 1,
		ConnectionLifetime: time.Hour,
	}
	
	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	
	if db == nil {
		t.Fatal("Database instance is nil")
	}
	
	if db.config != config {
		t.Error("Database config not set correctly")
	}
}

func TestDatabaseConnect(t *testing.T) {
	config := &DatabaseConfig{
		Type:       "sqlite",
		SQLiteFile: ":memory:",
		EnableWAL:  false,
		EnableForeignKeys: true,
		MaxConnections:     1,
		MaxIdleConnections: 1,
		ConnectionLifetime: time.Hour,
	}
	
	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	
	// Test connection
	err = db.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	
	if !db.IsConnected() {
		t.Error("Database should be connected")
	}
	
	// Test ping
	err = db.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
	
	// Test close
	err = db.Close()
	if err != nil {
		t.Errorf("Failed to close database: %v", err)
	}
	
	if db.IsConnected() {
		t.Error("Database should be disconnected after close")
	}
}

func TestDatabaseHealthCheck(t *testing.T) {
	config := &DatabaseConfig{
		Type:       "sqlite",
		SQLiteFile: ":memory:",
		EnableWAL:  false,
		EnableForeignKeys: true,
		MaxConnections:     1,
		MaxIdleConnections: 1,
		ConnectionLifetime: time.Hour,
	}
	
	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	
	// Test health check before connection
	health := db.HealthCheck()
	if health.Connected {
		t.Error("Database should not be connected initially")
	}
	if health.Status != "unhealthy" {
		t.Error("Database should be unhealthy when not connected")
	}
	
	// Connect and test health check
	err = db.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	health = db.HealthCheck()
	if !health.Connected {
		t.Error("Database should be connected")
	}
	if health.Status == "unhealthy" {
		t.Errorf("Database should not be unhealthy, got: %s", health.Status)
	}
	// Accept either "healthy" or "degraded" as the ping latency can vary in tests
	
	// Check that health details are populated
	if health.Details == nil {
		t.Error("Health details should not be nil")
	}
	
	if health.CheckedAt.IsZero() {
		t.Error("Health check timestamp should be set")
	}
}

func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid sqlite config",
			config: &DatabaseConfig{
				Type:       "sqlite",
				SQLiteFile: "test.db",
				MaxConnections: 1,
			},
			wantErr: false,
		},
		{
			name: "valid postgresql config",
			config: &DatabaseConfig{
				Type: "postgresql",
				Host: "localhost",
				Name: "testdb",
				MaxConnections: 10,
			},
			wantErr: false,
		},
		{
			name: "missing type",
			config: &DatabaseConfig{
				SQLiteFile: "test.db",
			},
			wantErr: true,
		},
		{
			name: "unsupported type",
			config: &DatabaseConfig{
				Type: "mysql",
				MaxConnections: 10,
			},
			wantErr: true,
		},
		{
			name: "sqlite missing file",
			config: &DatabaseConfig{
				Type: "sqlite",
				MaxConnections: 1,
			},
			wantErr: true,
		},
		{
			name: "postgresql missing host",
			config: &DatabaseConfig{
				Type: "postgresql",
				MaxConnections: 10,
			},
			wantErr: true,
		},
		{
			name: "invalid max connections",
			config: &DatabaseConfig{
				Type:           "sqlite",
				SQLiteFile:     "test.db",
				MaxConnections: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid idle connections",
			config: &DatabaseConfig{
				Type:               "sqlite",
				SQLiteFile:         "test.db",
				MaxConnections:     10,
				MaxIdleConnections: 15,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabaseConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   *DatabaseConfig
		expected string
	}{
		{
			name: "sqlite basic",
			config: &DatabaseConfig{
				Type:       "sqlite",
				SQLiteFile: "test.db",
			},
			expected: "test.db",
		},
		{
			name: "sqlite with WAL",
			config: &DatabaseConfig{
				Type:       "sqlite",
				SQLiteFile: "test.db",
				EnableWAL:  true,
			},
			expected: "test.db?_journal_mode=WAL",
		},
		{
			name: "sqlite with foreign keys",
			config: &DatabaseConfig{
				Type:              "sqlite",
				SQLiteFile:        "test.db",
				EnableForeignKeys: true,
			},
			expected: "test.db?_foreign_keys=1",
		},
		{
			name: "sqlite with both WAL and foreign keys",
			config: &DatabaseConfig{
				Type:              "sqlite",
				SQLiteFile:        "test.db",
				EnableWAL:         true,
				EnableForeignKeys: true,
			},
			expected: "test.db?_journal_mode=WAL&_foreign_keys=1",
		},
		{
			name: "postgresql basic",
			config: &DatabaseConfig{
				Type: "postgresql",
				Host: "localhost",
				Port: 5432,
				Name: "testdb",
			},
			expected: "host=localhost port=5432 dbname=testdb",
		},
		{
			name: "postgresql with credentials",
			config: &DatabaseConfig{
				Type:     "postgresql",
				Host:     "localhost",
				Port:     5432,
				Name:     "testdb",
				Username: "user",
				Password: "pass",
				SSLMode:  "require",
			},
			expected: "host=localhost port=5432 dbname=testdb user=user password=pass sslmode=require",
		},
		{
			name: "custom connection string",
			config: &DatabaseConfig{
				Type:             "postgresql",
				ConnectionString: "postgres://user:pass@localhost/testdb",
			},
			expected: "postgres://user:pass@localhost/testdb",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetConnectionString()
			if result != tt.expected {
				t.Errorf("GetConnectionString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDatabaseDriverName(t *testing.T) {
	tests := []struct {
		dbType   string
		expected string
	}{
		{"sqlite", "sqlite"},
		{"postgresql", "postgres"},
		{"unknown", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			config := &DatabaseConfig{Type: tt.dbType}
			result := config.GetDriverName()
			if result != tt.expected {
				t.Errorf("GetDriverName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSQLiteFileCreation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := "/tmp/qt1-test"
	sqliteFile := tempDir + "/test.db"
	
	// Clean up before and after test
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)
	
	config := &DatabaseConfig{
		Type:              "sqlite",
		SQLiteFile:        sqliteFile,
		EnableWAL:         true,
		EnableForeignKeys: true,
		MaxConnections:    1,
		MaxIdleConnections: 1,
		ConnectionLifetime: time.Hour,
	}
	
	db, err := NewDatabase(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	
	// Connect should create the directory and file
	err = db.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Check that file was created
	if _, err := os.Stat(sqliteFile); os.IsNotExist(err) {
		t.Error("SQLite file was not created")
	}
	
	// Check that directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("SQLite directory was not created")
	}
}