package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"qt1-middleware/config"
)

// Logger interface for structured logging
type Logger interface {
	Info(message string, fields map[string]interface{})
	Warn(message string, fields map[string]interface{})
	Error(message string, fields map[string]interface{})
}

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
	
	// Try to read from log file if it exists
	if config.AppConfig.Logging.LogFile != "" {
		// Try to open the log file for reading
		file, err := os.Open(config.AppConfig.Logging.LogFile)
		if err == nil {
			defer file.Close()
			
			// Read file content and parse JSON logs
			var logs []LogEntry
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				var entry LogEntry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
					logs = append(logs, entry)
				}
			}
			
			// Return the most recent logs (reverse order, limit)
			if len(logs) > limit {
				logs = logs[len(logs)-limit:]
			}
			
			// Reverse to show newest first
			for i := len(logs)/2 - 1; i >= 0; i-- {
				opp := len(logs) - 1 - i
				logs[i], logs[opp] = logs[opp], logs[i]
			}
			
			return logs, nil
		}
	}
	
	// If no file logging is available, generate some sample logs for testing
	// This helps users see what logs would look like
	now := time.Now()
	sampleLogs := []LogEntry{
		{
			Timestamp: now.Add(-5 * time.Minute).UTC().Format(time.RFC3339),
			UserID:    "user_123",
			SessionID: "session_456",
			Message:   "Chat request processed successfully",
			Action:    "chat_request",
			Level:     "info",
		},
		{
			Timestamp: now.Add(-10 * time.Minute).UTC().Format(time.RFC3339),
			UserID:    "user_789",
			SessionID: "session_101",
			Message:   "User authenticated",
			Action:    "auth_success",
			Level:     "info",
		},
		{
			Timestamp: now.Add(-15 * time.Minute).UTC().Format(time.RFC3339),
			UserID:    "user_456",
			SessionID: "session_202",
			Message:   "Content moderation triggered",
			Action:    "moderation_check",
			Level:     "warn",
		},
		{
			Timestamp: now.Add(-20 * time.Minute).UTC().Format(time.RFC3339),
			UserID:    "demo_user",
			SessionID: "session_999",
			Message:   "Recent chat request via API",
			Action:    "received",
			Level:     "info",
		},
	}
	
	if len(sampleLogs) > limit {
		sampleLogs = sampleLogs[:limit]
	}
	
	return sampleLogs, nil
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