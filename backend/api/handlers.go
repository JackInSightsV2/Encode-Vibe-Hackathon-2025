package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"qt1-middleware/config"
	"qt1-middleware/utils"
)

// Package-level variable to store proxy instance for config reloading
var ProxyInstance interface {
	ReloadConfiguration()
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type StatusResponse struct {
	Service     string `json:"service"`
	Status      string `json:"status"`
	Version     string `json:"version"`
	Uptime      string `json:"uptime"`
	Moderation  bool   `json:"moderation_enabled"`
	Relevance   bool   `json:"relevance_enabled"`
	KillSwitch  bool   `json:"kill_switch_enabled"`
}

type KillSwitchRequest struct {
	Action    string `json:"action"`    // "block" or "unblock"
	Type      string `json:"type"`      // "user" or "session"
	ID        string `json:"id"`        // user_id or session_id
}

func HandleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		response := Response{
			Success: true,
			Data:    config.AppConfig,
		}
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPut:
		// Read and parse the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			sendError(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		
		var newConfig config.Config
		if err := json.Unmarshal(body, &newConfig); err != nil {
			sendError(w, "Invalid config format", http.StatusBadRequest)
			return
		}
		
		// Update config (in memory for now)
		*config.AppConfig = newConfig
		
		// Log basic update info without sensitive data
		log.Printf("Config updated in memory")
		
		// Log provider count for debugging without exposing keys
		if newConfig.Providers != nil {
			log.Printf("Loaded %d provider(s)", len(newConfig.Providers))
		}
		
		// Reload SDK clients with updated configuration
		if ProxyInstance != nil {
			log.Printf("Reloading SDK clients...")
			ProxyInstance.ReloadConfiguration()
		}
		
		// TODO: Save to file
		
		response := Response{
			Success: true,
			Data:    config.AppConfig,
		}
		json.NewEncoder(w).Encode(response)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	logs, err := utils.GetRecentLogs(limit)
	if err != nil {
		sendError(w, "Failed to retrieve logs", http.StatusInternalServerError)
		return
	}
	
	response := Response{
		Success: true,
		Data:    logs,
	}
	json.NewEncoder(w).Encode(response)
}

func HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	status := StatusResponse{
		Service:     "qt1-middleware",
		Status:      "healthy",
		Version:     "1.0.0",
		Uptime:      "0m", // TODO: Calculate actual uptime
		Moderation:  config.AppConfig.Moderation.Enabled,
		Relevance:   config.AppConfig.Relevance.Enabled,
		KillSwitch:  config.AppConfig.KillSwitch.Enabled,
	}
	
	response := Response{
		Success: true,
		Data:    status,
	}
	json.NewEncoder(w).Encode(response)
}

func HandleKillSwitch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		data := map[string]interface{}{
			"blocked_users":    config.AppConfig.KillSwitch.BlockedUsers,
			"blocked_sessions": config.AppConfig.KillSwitch.BlockedSessions,
		}
		
		response := Response{
			Success: true,
			Data:    data,
		}
		json.NewEncoder(w).Encode(response)
		
	case http.MethodPost:
		var req KillSwitchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, "Invalid request format", http.StatusBadRequest)
			return
		}
		
		switch req.Action {
		case "block":
			if req.Type == "user" {
				config.AppConfig.KillSwitch.BlockedUsers = append(config.AppConfig.KillSwitch.BlockedUsers, req.ID)
			} else if req.Type == "session" {
				config.AppConfig.KillSwitch.BlockedSessions = append(config.AppConfig.KillSwitch.BlockedSessions, req.ID)
			} else {
				sendError(w, "Invalid type, must be 'user' or 'session'", http.StatusBadRequest)
				return
			}
			
		case "unblock":
			if req.Type == "user" {
				config.AppConfig.KillSwitch.BlockedUsers = removeFromSlice(config.AppConfig.KillSwitch.BlockedUsers, req.ID)
			} else if req.Type == "session" {
				config.AppConfig.KillSwitch.BlockedSessions = removeFromSlice(config.AppConfig.KillSwitch.BlockedSessions, req.ID)
			} else {
				sendError(w, "Invalid type, must be 'user' or 'session'", http.StatusBadRequest)
				return
			}
			
		default:
			sendError(w, "Invalid action, must be 'block' or 'unblock'", http.StatusBadRequest)
			return
		}
		
		response := Response{
			Success: true,
			Data:    map[string]string{"message": "Kill switch updated successfully"},
		}
		json.NewEncoder(w).Encode(response)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := Response{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}

func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}