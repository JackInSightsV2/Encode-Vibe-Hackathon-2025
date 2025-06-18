package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ChatRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Model   string `json:"model"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Mock AI response
	response := ChatResponse{
		Success: true,
		Message: fmt.Sprintf("Hello %s! You said: %s. This is a mock AI response.", req.UserID, req.Message),
		Model:   "mock-ai-v1",
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/chat", handleChat)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"mock-ai"}`))
	})
	
	log.Printf("Mock AI service starting on localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}