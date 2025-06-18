package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"qt1-middleware/config"
	"qt1-middleware/utils"
)

type ChatRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Blocked bool   `json:"blocked,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type Proxy struct {
	client                  *http.Client
	promptInjectionDetector *PromptInjectionDetector
	providerRouter          *ProviderRouter
	sdkRouter               *SDKProviderRouter
}

// GetSDKRouter returns the SDK router instance
func (p *Proxy) GetSDKRouter() *SDKProviderRouter {
	return p.sdkRouter
}

func NewProxy() *Proxy {
	return &Proxy{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		promptInjectionDetector: NewPromptInjectionDetector(),
		providerRouter:          NewProviderRouter(),
		sdkRouter:               NewSDKProviderRouter(),
	}
}

func (p *Proxy) HandleChat(w http.ResponseWriter, r *http.Request) {
	log.Printf("HANDLECHAT CALLED - METHOD: %s", r.Method)
	fmt.Printf("\n🚀 CHAT REQUEST RECEIVED\n")
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Parse request - support both basic and extended formats
	var req ChatRequestExtended
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	
	fmt.Printf("Request details:\n")
	fmt.Printf("  Provider: %s\n", req.Provider)
	fmt.Printf("  Model: %s\n", req.Model)
	fmt.Printf("  TargetURL: %s\n", req.TargetURL)
	fmt.Printf("  Message: %.50s...\n", req.Message)
	
	// Log request
	utils.LogRequest(req.UserID, req.SessionID, req.Message, "received")
	
	// 1. Check kill switch
	if p.isBlocked(req.UserID, req.SessionID) {
		response := ChatResponse{
			Success: false,
			Blocked: true,
			Reason:  "User or session is blocked",
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_killswitch")
		p.sendResponse(w, response, http.StatusForbidden)
		return
	}
	
	// 2. Check for prompt injection
	if blocked, reason := p.promptInjectionDetector.DetectPromptInjection(req.Message); blocked {
		response := ChatResponse{
			Success: false,
			Blocked: true,
			Reason:  fmt.Sprintf("Prompt injection detected: %s", reason),
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_prompt_injection")
		p.sendResponse(w, response, http.StatusForbidden)
		return
	}

	// 3. Run moderation
	if config.AppConfig.Moderation.Enabled {
		if blocked, reason := p.moderateContent(req.Message); blocked {
			response := ChatResponse{
				Success: false,
				Blocked: true,
				Reason:  reason,
			}
			utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_moderation")
			p.sendResponse(w, response, http.StatusForbidden)
			return
		}
	}
	
	// 4. Run relevance filtering
	if config.AppConfig.Relevance.Enabled {
		if blocked, reason := p.checkRelevance(req.Message); blocked {
			response := ChatResponse{
				Success: false,
				Blocked: true,
				Reason:  reason,
			}
			utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_relevance")
			p.sendResponse(w, response, http.StatusForbidden)
			return
		}
	}
	
	// 5. Route to AI provider - prefer SDK routing for OpenAI/Anthropic
	var resp *ProxyResponse
	var err error
	
	// FORCE OpenAI SDK routing for testing
	log.Printf("FORCING OPENAI SDK ROUTING FOR DEBUGGING")
	resp, err = p.sdkRouter.RouteRequestSDK(req)
	
	if err != nil {
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "error_routing")
		p.sendError(w, fmt.Sprintf("Failed to route request: %v", err), http.StatusBadGateway)
		return
	}
	
	utils.LogRequest(req.UserID, req.SessionID, req.Message, "routed_successfully")
	
	// Return response
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)
}

func (p *Proxy) isBlocked(userID, sessionID string) bool {
	// Check blocked users
	for _, blockedUser := range config.AppConfig.KillSwitch.BlockedUsers {
		if blockedUser == userID {
			return true
		}
	}
	
	// Check blocked sessions
	for _, blockedSession := range config.AppConfig.KillSwitch.BlockedSessions {
		if blockedSession == sessionID {
			return true
		}
	}
	
	return false
}

func (p *Proxy) moderateContent(message string) (bool, string) {
	// Run moderation layers in sequence
	for _, layer := range config.AppConfig.Moderation.Layers {
		blocked, reason := p.runModerationLayer(message, layer)
		if blocked {
			return true, reason
		}
	}
	
	// Legacy support - basic regex-based filtering
	messageLower := strings.ToLower(message)
	for _, word := range config.AppConfig.Moderation.BlockedWords {
		if strings.Contains(messageLower, strings.ToLower(word)) {
			return true, fmt.Sprintf("Content blocked: contains prohibited word '%s'", word)
		}
	}
	
	// Legacy OpenAI moderation support
	if config.AppConfig.Moderation.UseOpenAI {
		log.Printf("Legacy OpenAI moderation not implemented - use moderation layers instead")
	}
	
	return false, ""
}

func (p *Proxy) runModerationLayer(message string, layer config.ModerationLayer) (bool, string) {
	switch layer.Type {
	case "regex":
		return p.regexModeration(message, layer)
	case "llm":
		return p.llmModeration(message, layer)
	case "custom":
		return p.customModeration(message, layer)
	default:
		log.Printf("Unknown moderation layer type: %s", layer.Type)
		return false, ""
	}
}

func (p *Proxy) regexModeration(message string, layer config.ModerationLayer) (bool, string) {
	messageLower := strings.ToLower(message)
	for _, word := range config.AppConfig.Moderation.BlockedWords {
		if strings.Contains(messageLower, strings.ToLower(word)) {
			return true, fmt.Sprintf("Regex moderation: contains prohibited word '%s'", word)
		}
	}
	return false, ""
}

func (p *Proxy) llmModeration(message string, layer config.ModerationLayer) (bool, string) {
	if layer.Provider == "" || layer.Model == "" {
		log.Printf("LLM moderation layer missing provider or model configuration")
		return false, ""
	}
	
	// Create request for LLM moderation
	moderationRequest := ChatRequestExtended{
		UserID:    "system",
		SessionID: "moderation",
		Message:   layer.Prompt + "\n\nMessage to analyze: " + message,
		Provider:  layer.Provider,
		Model:     layer.Model,
		Settings: map[string]interface{}{
			"max_tokens":  100,
			"temperature": 0.1,
		},
	}
	
	// Route to LLM provider
	resp, err := p.providerRouter.RouteRequest(moderationRequest)
	if err != nil {
		log.Printf("LLM moderation failed: %v", err)
		return false, ""
	}
	
	// Parse response - simplified for now
	if resp.StatusCode != 200 {
		log.Printf("LLM moderation returned non-200 status: %d", resp.StatusCode)
		return false, ""
	}
	
	responseText := strings.ToLower(string(resp.Body))
	if strings.Contains(responseText, "unsafe") || strings.Contains(responseText, "harmful") || strings.Contains(responseText, "inappropriate") {
		return true, "LLM moderation: content flagged as potentially harmful"
	}
	
	return false, ""
}

func (p *Proxy) customModeration(message string, layer config.ModerationLayer) (bool, string) {
	// Placeholder for custom moderation logic
	log.Printf("Custom moderation not implemented")
	return false, ""
}

func (p *Proxy) checkRelevance(message string) (bool, string) {
	// TODO: Implement embedding-based relevance checking
	if config.AppConfig.Relevance.UseOpenAI {
		// Placeholder for OpenAI embeddings
		log.Printf("OpenAI relevance checking not yet implemented")
	}
	
	// For now, just return not blocked
	return false, ""
}

type ProxyResponse struct {
	StatusCode int
	Body       []byte
}


func (p *Proxy) sendResponse(w http.ResponseWriter, response ChatResponse, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (p *Proxy) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := ChatResponse{
		Success: false,
		Error:   message,
	}
	p.sendResponse(w, response, statusCode)
}

// Helper functions for model detection
func isOpenAIModel(model string) bool {
	openaiModels := []string{
		"gpt-4o-mini", "gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-4",
	}
	
	modelLower := strings.ToLower(model)
	for _, openaiModel := range openaiModels {
		if strings.Contains(modelLower, strings.ToLower(openaiModel)) {
			return true
		}
	}
	
	return false
}

func isAnthropicModel(model string) bool {
	anthropicModels := []string{
		"claude-3", "claude-2", "claude-instant",
		"claude-3-sonnet", "claude-3-opus", "claude-3-haiku",
	}
	
	modelLower := strings.ToLower(model)
	for _, anthropicModel := range anthropicModels {
		if strings.Contains(modelLower, strings.ToLower(anthropicModel)) {
			return true
		}
	}
	
	return false
}

// ReloadConfiguration reinitializes SDK clients with updated config
func (p *Proxy) ReloadConfiguration() {
	p.sdkRouter.ReinitializeClients()
}