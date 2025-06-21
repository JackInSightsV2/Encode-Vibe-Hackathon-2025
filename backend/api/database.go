package api

import (
	"encoding/json"
	"net/http"
	"qt1-middleware/database"
)

// DatabaseAPI handles database-related API endpoints
type DatabaseAPI struct {
	db *database.Database
}

// NewDatabaseAPI creates a new database API handler
func NewDatabaseAPI(db *database.Database) *DatabaseAPI {
	return &DatabaseAPI{db: db}
}

// HandleMigrationStatus returns the current migration status
func (api *DatabaseAPI) HandleMigrationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if api.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}
	
	status, err := api.db.GetMigrationStatus()
	if err != nil {
		http.Error(w, "Failed to get migration status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   status,
	})
}

// HandleDatabaseHealth returns database health information
func (api *DatabaseAPI) HandleDatabaseHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if api.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}
	
	health := api.db.HealthCheck()
	
	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		statusCode = http.StatusPartialContent
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   health,
	})
}

// HandleDatabaseInfo returns general database information
func (api *DatabaseAPI) HandleDatabaseInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if api.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}
	
	config := api.db.GetConfig()
	stats := api.db.GetConnectionStats()
	uptime := api.db.GetUptime()
	
	info := map[string]interface{}{
		"type":              config.Type,
		"connected":         api.db.IsConnected(),
		"ready":             api.db.IsReady(),
		"uptime_seconds":    uptime.Seconds(),
		"connection_stats":  stats,
		"migrations_enabled": config.Migrations.Enabled,
	}
	
	// Add type-specific information
	if config.Type == "sqlite" {
		info["sqlite_file"] = config.SQLiteFile
		info["wal_enabled"] = config.EnableWAL
		info["foreign_keys_enabled"] = config.EnableForeignKeys
	} else if config.Type == "postgresql" {
		info["host"] = config.Host
		info["port"] = config.Port
		info["database_name"] = config.Name
		info["ssl_mode"] = config.SSLMode
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   info,
	})
}