package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"qt1-middleware/config"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	Action    string `json:"action"`
	Level     string `json:"level"`
}

var logFile *os.File

func init() {
	// This will be called when the package is imported
	// We'll initialize the log file when config is loaded
}

func InitLogging() error {
	if !config.AppConfig.Logging.Enabled {
		return nil
	}
	
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	
	// Open log file
	var err error
	logFile, err = os.OpenFile(config.AppConfig.Logging.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	return nil
}

func LogRequest(userID, sessionID, message, action string) {
	if !config.AppConfig.Logging.Enabled {
		return
	}
	
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		UserID:    userID,
		SessionID: sessionID,
		Message:   truncateMessage(message, 500), // Truncate long messages
		Action:    action,
		Level:     "info",
	}
	
	// Log to console
	log.Printf("[%s] User:%s Session:%s Action:%s", entry.Timestamp, entry.UserID, entry.SessionID, entry.Action)
	
	// Log to file if enabled
	if logFile != nil {
		logJSON, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Failed to marshal log entry: %v", err)
			return
		}
		
		if _, err := logFile.WriteString(string(logJSON) + "\n"); err != nil {
			log.Printf("Failed to write to log file: %v", err)
		}
	}
	
	// TODO: Add SQLite logging if enabled
	if config.AppConfig.Logging.UseSQLite {
		// Placeholder for SQLite logging
		log.Printf("SQLite logging not yet implemented")
	}
}

func LogError(userID, sessionID, message, errorMsg string) {
	if !config.AppConfig.Logging.Enabled {
		return
	}
	
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		UserID:    userID,
		SessionID: sessionID,
		Message:   truncateMessage(message, 500),
		Action:    fmt.Sprintf("error: %s", errorMsg),
		Level:     "error",
	}
	
	// Log to console
	log.Printf("[ERROR] [%s] User:%s Session:%s Error:%s", entry.Timestamp, entry.UserID, entry.SessionID, errorMsg)
	
	// Log to file if enabled
	if logFile != nil {
		logJSON, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Failed to marshal error log entry: %v", err)
			return
		}
		
		if _, err := logFile.WriteString(string(logJSON) + "\n"); err != nil {
			log.Printf("Failed to write error to log file: %v", err)
		}
	}
}

func GetRecentLogs(limit int) ([]LogEntry, error) {
	if !config.AppConfig.Logging.Enabled {
		return []LogEntry{}, nil
	}
	
	// For now, return empty logs
	// TODO: Implement log reading from file or SQLite
	return []LogEntry{}, nil
}

func truncateMessage(message string, maxLen int) string {
	if len(message) <= maxLen {
		return message
	}
	return message[:maxLen] + "..."
}

func CloseLogging() {
	if logFile != nil {
		logFile.Close()
	}
}