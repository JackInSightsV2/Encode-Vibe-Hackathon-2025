package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"qt1-middleware/utils"
)

// TestRequest represents a test request to the OpenAI API
type TestRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// TestOpenAIHandler handles requests to test the OpenAI client
func (p *Proxy) TestOpenAIHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log request
	utils.LogRequest(req.UserID, req.SessionID, req.Message, "test_received")

	// 1. Check kill switch
	if p.isBlocked(req.UserID, req.SessionID) {
		response := map[string]interface{}{
			"success": false,
			"blocked": true,
			"reason":  "User or session is blocked",
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_killswitch")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 2. Check for prompt injection
	if blocked, reason := p.promptInjectionDetector.DetectPromptInjection(req.Message); blocked {
		response := map[string]interface{}{
			"success": false,
			"blocked": true,
			"reason":  fmt.Sprintf("Prompt injection detected: %s", reason),
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_prompt_injection")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 3. Run moderation
	if blocked, reason := p.moderateContent(req.Message); blocked {
		response := map[string]interface{}{
			"success": false,
			"blocked": true,
			"reason":  reason,
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_moderation")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the SDK router from the proxy
	sdkRouter := p.GetSDKRouter()
	if sdkRouter == nil {
		http.Error(w, "SDK router not initialized", http.StatusInternalServerError)
		return
	}

	// Test the OpenAI client with the provided message
	response, err := sdkRouter.TestOpenAIClient(req.Message)
	if err != nil {
		errorResponse := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "error_openai_api")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Log successful request
	utils.LogRequest(req.UserID, req.SessionID, req.Message, "test_successful")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
