package database

import (
	"fmt"
	"time"
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Type                string        `yaml:"type" json:"type"`                                 // "sqlite" or "postgresql"
	ConnectionString    string        `yaml:"connection_string" json:"connection_string"`       // Full connection string (optional)
	Host               string        `yaml:"host" json:"host"`                                 // Database host
	Port               int           `yaml:"port" json:"port"`                                 // Database port
	Name               string        `yaml:"name" json:"name"`                                 // Database name
	Username           string        `yaml:"username" json:"username"`                         // Database username
	Password           string        `yaml:"password" json:"password"`                         // Database password
	SSLMode            string        `yaml:"ssl_mode" json:"ssl_mode"`                         // SSL mode for PostgreSQL
	MaxConnections     int           `yaml:"max_connections" json:"max_connections"`           // Maximum number of connections
	MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections"` // Maximum idle connections
	ConnectionLifetime time.Duration `yaml:"connection_lifetime" json:"connection_lifetime"`   // Connection lifetime
	
	// SQLite specific options
	SQLiteFile     string `yaml:"sqlite_file" json:"sqlite_file"`         // SQLite database file path
	EnableWAL      bool   `yaml:"enable_wal" json:"enable_wal"`           // Enable WAL mode for SQLite
	EnableForeignKeys bool `yaml:"enable_foreign_keys" json:"enable_foreign_keys"` // Enable foreign key constraints
	
	// Migration settings
	Migrations MigrationConfig `yaml:"migrations" json:"migrations"`
	
	// Retention policies
	Retention RetentionConfig `yaml:"retention" json:"retention"`
}

// MigrationConfig holds migration-specific configuration
type MigrationConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`     // Enable automatic migrations
	Directory string `yaml:"directory" json:"directory"` // Directory containing migration files
	Table     string `yaml:"table" json:"table"`         // Migration tracking table name
}

// RetentionConfig holds data retention policies
type RetentionConfig struct {
	Requests string `yaml:"requests" json:"requests"` // Request data retention (e.g., "30d")
	Metrics  string `yaml:"metrics" json:"metrics"`   // Metrics data retention (e.g., "7d")
	Logs     string `yaml:"logs" json:"logs"`         // Log data retention (e.g., "90d")
}

// DefaultDatabaseConfig returns a default database configuration
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Type:                "sqlite",
		Host:                "localhost",
		Port:                5432,
		Name:                "qt1_middleware",
		Username:            "",
		Password:            "",
		SSLMode:             "disable",
		MaxConnections:      25,
		MaxIdleConnections:  5,
		ConnectionLifetime:  time.Hour,
		SQLiteFile:          "storage/qt1.db",
		EnableWAL:           true,
		EnableForeignKeys:   true,
		Migrations: MigrationConfig{
			Enabled:   true,
			Directory: "./database/migrations",
			Table:     "schema_migrations",
		},
		Retention: RetentionConfig{
			Requests: "30d",
			Metrics:  "7d",
			Logs:     "90d",
		},
	}
}

// Validate checks if the database configuration is valid
func (c *DatabaseConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("database type is required")
	}
	
	if c.Type != "sqlite" && c.Type != "postgresql" {
		return fmt.Errorf("unsupported database type: %s", c.Type)
	}
	
	if c.Type == "sqlite" {
		if c.SQLiteFile == "" {
			return fmt.Errorf("sqlite_file is required for SQLite database")
		}
	}
	
	if c.Type == "postgresql" {
		if c.ConnectionString == "" {
			if c.Host == "" || c.Name == "" {
				return fmt.Errorf("host and name are required for PostgreSQL database")
			}
		}
	}
	
	if c.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be greater than 0")
	}
	
	if c.MaxIdleConnections < 0 {
		return fmt.Errorf("max_idle_connections cannot be negative")
	}
	
	if c.MaxIdleConnections > c.MaxConnections {
		return fmt.Errorf("max_idle_connections cannot be greater than max_connections")
	}
	
	return nil
}

// GetConnectionString builds a connection string based on the configuration
func (c *DatabaseConfig) GetConnectionString() string {
	if c.ConnectionString != "" {
		return c.ConnectionString
	}
	
	switch c.Type {
	case "sqlite":
		connStr := c.SQLiteFile
		if c.EnableWAL {
			connStr += "?_journal_mode=WAL"
		}
		if c.EnableForeignKeys {
			if c.EnableWAL {
				connStr += "&_foreign_keys=1"
			} else {
				connStr += "?_foreign_keys=1"
			}
		}
		return connStr
		
	case "postgresql":
		if c.Port == 0 {
			c.Port = 5432
		}
		
		connStr := fmt.Sprintf("host=%s port=%d dbname=%s", c.Host, c.Port, c.Name)
		
		if c.Username != "" {
			connStr += fmt.Sprintf(" user=%s", c.Username)
		}
		
		if c.Password != "" {
			connStr += fmt.Sprintf(" password=%s", c.Password)
		}
		
		if c.SSLMode != "" {
			connStr += fmt.Sprintf(" sslmode=%s", c.SSLMode)
		}
		
		return connStr
		
	default:
		return ""
	}
}

// GetDriverName returns the appropriate database driver name
func (c *DatabaseConfig) GetDriverName() string {
	switch c.Type {
	case "sqlite":
		return "sqlite"
	case "postgresql":
		return "postgres"
	default:
		return ""
	}
}