package api

import (
	"encoding/json"
	"net/http"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// HandleLogin handles user authentication and token generation
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simple authentication for demo purposes
	// In production, this would check against a database with hashed passwords
	var user User
	switch loginReq.Username {
	case "admin":
		if loginReq.Password != "admin123" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		user = User{
			ID:       "admin-001",
			Username: "admin",
			Email:    "admin@qt1.com",
			Roles:    []string{"admin", "websocket", "health", "logs"},
			IsAdmin:  true,
		}
	case "user":
		if loginReq.Password != "user123" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		user = User{
			ID:       "user-001",
			Username: "user",
			Email:    "user@qt1.com",
			Roles:    []string{"websocket"},
			IsAdmin:  false,
		}
	case "health":
		if loginReq.Password != "health123" {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		user = User{
			ID:       "health-001",
			Username: "health",
			Email:    "health@qt1.com",
			Roles:    []string{"websocket", "health"},
			IsAdmin:  false,
		}
	default:
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := AuthSvc.GenerateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleTokenValidation handles token validation (for testing)
func HandleTokenValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := AuthSvc.AuthenticateRequest(r)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"user":  user,
	})
}