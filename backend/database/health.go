package database

import (
	"context"
	"database/sql"
	"time"
)

// DatabaseHealth represents the health status of the database
type DatabaseHealth struct {
	Status         string                 `json:"status"`          // "healthy", "degraded", "unhealthy"
	Uptime         time.Duration          `json:"uptime"`          // Time since database connection was established
	Connected      bool                   `json:"connected"`       // Whether the database is connected
	PingLatency    time.Duration          `json:"ping_latency"`    // Latency of last ping
	OpenConnections int                   `json:"open_connections"` // Number of open connections
	IdleConnections int                   `json:"idle_connections"` // Number of idle connections
	Details        map[string]interface{} `json:"details"`         // Additional database-specific details
	LastError      string                 `json:"last_error,omitempty"` // Last error encountered
	CheckedAt      time.Time              `json:"checked_at"`      // When this health check was performed
}

// HealthChecker provides database health monitoring functionality
type HealthChecker struct {
	db        *sql.DB
	config    *DatabaseConfig
	startTime time.Time
	lastError error
}

// NewHealthChecker creates a new database health checker
func NewHealthChecker(db *sql.DB, config *DatabaseConfig) *HealthChecker {
	return &HealthChecker{
		db:        db,
		config:    config,
		startTime: time.Now(),
	}
}

// Check performs a comprehensive health check of the database
func (h *HealthChecker) Check() *DatabaseHealth {
	health := &DatabaseHealth{
		Uptime:    time.Since(h.startTime),
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}
	
	// Check basic connectivity
	connected, pingLatency := h.checkConnectivity()
	health.Connected = connected
	health.PingLatency = pingLatency
	
	// Get connection statistics
	stats := h.db.Stats()
	health.OpenConnections = stats.OpenConnections
	health.IdleConnections = stats.Idle
	
	// Add detailed statistics
	health.Details["max_open_connections"] = stats.MaxOpenConnections
	health.Details["in_use"] = stats.InUse
	health.Details["wait_count"] = stats.WaitCount
	health.Details["wait_duration"] = stats.WaitDuration
	health.Details["max_idle_closed"] = stats.MaxIdleClosed
	health.Details["max_idle_time_closed"] = stats.MaxIdleTimeClosed
	health.Details["max_lifetime_closed"] = stats.MaxLifetimeClosed
	
	// Determine overall status
	health.Status = h.determineStatus(health)
	
	// Add last error if any
	if h.lastError != nil {
		health.LastError = h.lastError.Error()
	}
	
	// Add database-specific details
	h.addDatabaseSpecificDetails(health)
	
	return health
}

// checkConnectivity tests basic database connectivity
func (h *HealthChecker) checkConnectivity() (bool, time.Duration) {
	start := time.Now()
	err := h.db.Ping()
	latency := time.Since(start)
	
	if err != nil {
		h.lastError = err
		return false, latency
	}
	
	h.lastError = nil
	return true, latency
}

// determineStatus determines the overall health status based on various metrics
func (h *HealthChecker) determineStatus(health *DatabaseHealth) string {
	// If not connected, it's unhealthy
	if !health.Connected {
		return "unhealthy"
	}
	
	// Check ping latency (consider degraded if > 100ms, unhealthy if > 1s)
	if health.PingLatency > time.Second {
		return "unhealthy"
	}
	if health.PingLatency > 100*time.Millisecond {
		return "degraded"
	}
	
	// Check connection utilization
	maxConn := h.config.MaxConnections
	if maxConn > 0 {
		utilization := float64(health.OpenConnections) / float64(maxConn)
		if utilization > 0.9 { // More than 90% utilization
			return "degraded"
		}
	}
	
	// Check for excessive wait times
	stats := h.db.Stats()
	if stats.WaitDuration > time.Minute {
		return "degraded"
	}
	
	return "healthy"
}

// addDatabaseSpecificDetails adds database-type specific health information
func (h *HealthChecker) addDatabaseSpecificDetails(health *DatabaseHealth) {
	switch h.config.Type {
	case "sqlite":
		h.addSQLiteDetails(health)
	case "postgresql":
		h.addPostgreSQLDetails(health)
	}
}

// addSQLiteDetails adds SQLite-specific health information
func (h *HealthChecker) addSQLiteDetails(health *DatabaseHealth) {
	health.Details["database_type"] = "sqlite"
	health.Details["database_file"] = h.config.SQLiteFile
	health.Details["wal_enabled"] = h.config.EnableWAL
	health.Details["foreign_keys_enabled"] = h.config.EnableForeignKeys
	
	// Try to get SQLite-specific information
	if health.Connected {
		// Check WAL mode
		var walMode string
		err := h.db.QueryRow("PRAGMA journal_mode").Scan(&walMode)
		if err == nil {
			health.Details["current_journal_mode"] = walMode
		}
		
		// Check foreign keys status
		var foreignKeys int
		err = h.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
		if err == nil {
			health.Details["foreign_keys_active"] = foreignKeys == 1
		}
		
		// Check page count and size
		var pageCount, pageSize int
		err = h.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
		if err == nil {
			health.Details["page_count"] = pageCount
		}
		
		err = h.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
		if err == nil {
			health.Details["page_size"] = pageSize
			if pageCount > 0 {
				health.Details["database_size_bytes"] = pageCount * pageSize
			}
		}
	}
}

// addPostgreSQLDetails adds PostgreSQL-specific health information
func (h *HealthChecker) addPostgreSQLDetails(health *DatabaseHealth) {
	health.Details["database_type"] = "postgresql"
	health.Details["host"] = h.config.Host
	health.Details["port"] = h.config.Port
	health.Details["database_name"] = h.config.Name
	health.Details["ssl_mode"] = h.config.SSLMode
	
	// Try to get PostgreSQL-specific information
	if health.Connected {
		// Get PostgreSQL version
		var version string
		err := h.db.QueryRow("SELECT version()").Scan(&version)
		if err == nil {
			health.Details["server_version"] = version
		}
		
		// Get current database name
		var currentDB string
		err = h.db.QueryRow("SELECT current_database()").Scan(&currentDB)
		if err == nil {
			health.Details["current_database"] = currentDB
		}
		
		// Get database size
		var dbSize int64
		query := "SELECT pg_database_size(current_database())"
		err = h.db.QueryRow(query).Scan(&dbSize)
		if err == nil {
			health.Details["database_size_bytes"] = dbSize
		}
	}
}

// QuickPing performs a quick connectivity check
func (h *HealthChecker) QuickPing() error {
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return h.db.PingContext(ctxWithTimeout)
}

// GetConnectionStats returns current connection statistics
func (h *HealthChecker) GetConnectionStats() sql.DBStats {
	return h.db.Stats()
}

// IsHealthy returns true if the database is considered healthy
func (h *HealthChecker) IsHealthy() bool {
	health := h.Check()
	return health.Status == "healthy"
}

// GetUptimeDuration returns how long the database has been running
func (h *HealthChecker) GetUptimeDuration() time.Duration {
	return time.Since(h.startTime)
}