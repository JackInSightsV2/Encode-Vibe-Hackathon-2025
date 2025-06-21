package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"qt1-middleware/api"
	"qt1-middleware/config"
	"qt1-middleware/metrics"
	"qt1-middleware/moderation"
	"qt1-middleware/moderation/layers"
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
	moderationEngine        *moderation.ModerationEngine
}

// GetSDKRouter returns the SDK router instance
func (p *Proxy) GetSDKRouter() *SDKProviderRouter {
	return p.sdkRouter
}

func NewProxy() *Proxy {
	p := &Proxy{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		promptInjectionDetector: NewPromptInjectionDetector(),
		providerRouter:          NewProviderRouter(),
		sdkRouter:               NewSDKProviderRouter(),
	}
	
	// Initialize advanced moderation engine
	p.initModerationEngine()
	
	return p
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

	// 3. Run moderation - use advanced engine if available, fallback to legacy
	log.Printf("DEBUG: About to run moderation for user %s", req.UserID)
	if blocked, reason := p.runModeration(req); blocked {
		response := ChatResponse{
			Success: false,
			Blocked: true,
			Reason:  reason,
		}
		utils.LogRequest(req.UserID, req.SessionID, req.Message, "blocked_moderation")
		p.sendResponse(w, response, http.StatusForbidden)
		return
	}
	log.Printf("DEBUG: Moderation passed for user %s", req.UserID)
	
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
	
	// Check if we should use SDK routing (for OpenAI/Anthropic) or provider routing (for others)
	if (req.Provider == "openai" || req.Provider == "anthropic") ||
	   (req.Provider == "" && (isOpenAIModel(req.Model) || isAnthropicModel(req.Model))) {
		log.Printf("Using SDK routing for provider: %s, model: %s", req.Provider, req.Model)
		resp, err = p.sdkRouter.RouteRequestSDK(req)
	} else {
		log.Printf("Using provider routing for provider: %s, model: %s", req.Provider, req.Model)
		resp, err = p.providerRouter.RouteRequest(req)
	}
	
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

// initModerationEngine initializes the advanced moderation engine
func (p *Proxy) initModerationEngine() {
	// Check if advanced moderation is enabled
	if !config.AppConfig.Moderation.Advanced.Enabled {
		log.Printf("Advanced moderation disabled, using legacy moderation")
		return
	}
	
	// Initialize moderation engine with app config
	moderationConfig := &moderation.AdvancedModerationConfig{
		Enabled: config.AppConfig.Moderation.Advanced.Enabled,
		Layers: convertLayerConfigs(config.AppConfig.Moderation.Advanced.Layers),
		Thresholds: moderation.ModerationThresholds{
			Low:      config.AppConfig.Moderation.Advanced.Thresholds.Low,
			Medium:   config.AppConfig.Moderation.Advanced.Thresholds.Medium,
			High:     config.AppConfig.Moderation.Advanced.Thresholds.High,
			Critical: config.AppConfig.Moderation.Advanced.Thresholds.Critical,
		},
		Actions: moderation.ActionConfig{
			Low:      config.AppConfig.Moderation.Advanced.Actions.Low,
			Medium:   config.AppConfig.Moderation.Advanced.Actions.Medium,
			High:     config.AppConfig.Moderation.Advanced.Actions.High,
			Critical: config.AppConfig.Moderation.Advanced.Actions.Critical,
		},
		Cache: moderation.CacheConfig{
			Enabled:    config.AppConfig.Moderation.Advanced.Cache.Enabled,
			TTLMinutes: config.AppConfig.Moderation.Advanced.Cache.TTLMinutes,
			MaxEntries: config.AppConfig.Moderation.Advanced.Cache.MaxEntries,
		},
		Analytics: moderation.AnalyticsConfig{
			Enabled:           config.AppConfig.Moderation.Advanced.Analytics.Enabled,
			CollectDetails:    config.AppConfig.Moderation.Advanced.Analytics.CollectDetails,
			RetentionDays:     config.AppConfig.Moderation.Advanced.Analytics.RetentionDays,
			EnablePerformance: config.AppConfig.Moderation.Advanced.Analytics.EnablePerformance,
		},
	}
	
	// Create moderation engine with our config
	cache := moderation.NewModerationCache(moderationConfig.Cache)
	configManager := moderation.NewConfigManager("") // Empty path since we're providing config directly
	
	engine := moderation.NewModerationEngineWithConfig(
		make([]moderation.ModerationLayer, 0),
		cache,
		moderationConfig,
		configManager,
	)
	
	// Register enabled layers from config
	for _, layerConfig := range config.AppConfig.Moderation.Advanced.Layers {
		if !layerConfig.Enabled {
			continue
		}
		
		switch layerConfig.Name {
		case "regex":
			regexLayer := layers.NewRegexLayer(moderation.LayerConfig{
				Name:      layerConfig.Name,
				Enabled:   layerConfig.Enabled,
				Weight:    layerConfig.Weight,
				Threshold: layerConfig.Threshold,
				Options:   layerConfig.Options,
			})
			if err := engine.RegisterLayer(regexLayer); err != nil {
				log.Printf("Failed to register regex layer: %v", err)
			} else {
				log.Printf("Registered regex moderation layer")
			}
		case "openai":
			// Create fallback layer (regex) for OpenAI
			fallbackLayer := layers.NewRegexLayer(moderation.LayerConfig{
				Name:      "regex_fallback",
				Enabled:   true,
				Weight:    0.5,
				Threshold: 0.7,
				Options:   make(map[string]interface{}),
			})
			
			openaiLayer := layers.NewOpenAILayer(moderation.LayerConfig{
				Name:      layerConfig.Name,
				Enabled:   layerConfig.Enabled,
				Weight:    layerConfig.Weight,
				Threshold: layerConfig.Threshold,
				Options:   layerConfig.Options,
			}, fallbackLayer)
			
			if err := engine.RegisterLayer(openaiLayer); err != nil {
				log.Printf("Failed to register OpenAI layer: %v", err)
			} else {
				log.Printf("Registered OpenAI moderation layer")
			}
		case "pii":
			piiLayer := layers.NewPIILayer(moderation.LayerConfig{
				Name:      layerConfig.Name,
				Enabled:   layerConfig.Enabled,
				Weight:    layerConfig.Weight,
				Threshold: layerConfig.Threshold,
				Options:   layerConfig.Options,
			})
			if err := engine.RegisterLayer(piiLayer); err != nil {
				log.Printf("Failed to register PII layer: %v", err)
			} else {
				log.Printf("Registered PII moderation layer")
			}
		case "custom_rules", "rules":
			rulesLayer := layers.NewRulesLayer(moderation.LayerConfig{
				Name:      layerConfig.Name,
				Enabled:   layerConfig.Enabled,
				Weight:    layerConfig.Weight,
				Threshold: layerConfig.Threshold,
				Options:   layerConfig.Options,
			})
			if err := engine.RegisterLayer(rulesLayer); err != nil {
				log.Printf("Failed to register Rules layer: %v", err)
			} else {
				log.Printf("Registered Rules moderation layer")
			}
		case "relevancy", "relevance":
			relevancyLayer := layers.NewRelevancyLayer(moderation.LayerConfig{
				Name:      layerConfig.Name,
				Enabled:   layerConfig.Enabled,
				Weight:    layerConfig.Weight,
				Threshold: layerConfig.Threshold,
				Options:   layerConfig.Options,
			})
			if err := engine.RegisterLayer(relevancyLayer); err != nil {
				log.Printf("Failed to register Relevancy layer: %v", err)
			} else {
				log.Printf("Registered Relevancy moderation layer")
			}
		default:
			log.Printf("Layer type '%s' not yet implemented, skipping", layerConfig.Name)
		}
	}
	
	p.moderationEngine = engine
	
	// Set global instance for API handlers
	api.ModerationEngineInstance = engine
	
	log.Printf("Advanced moderation engine initialized with %d layers", len(engine.GetLayerNames()))
}

// runModeration runs content through moderation (advanced or legacy)
func (p *Proxy) runModeration(req ChatRequestExtended) (bool, string) {
	log.Printf("DEBUG: runModeration called - engine nil? %v", p.moderationEngine == nil)
	if p.moderationEngine != nil {
		log.Printf("DEBUG: moderation engine exists, enabled? %v", p.moderationEngine.IsEnabled())
	}
	
	// Try advanced moderation first
	if p.moderationEngine != nil && p.moderationEngine.IsEnabled() {
		context := moderation.ModerationContext{
			UserID:    req.UserID,
			SessionID: req.SessionID,
			Timestamp: time.Now(),
		}
		
		result, err := p.moderationEngine.Moderate(req.Message, context)
		if err != nil {
			log.Printf("Advanced moderation failed: %v", err)
			log.Printf("Falling back to legacy moderation")
		} else {
			// Broadcast moderation events via WebSocket
			p.broadcastModerationEvent(result, req)
			
			// Log advanced moderation result
			if result.FinalDecision {
				log.Printf("Advanced moderation blocked content: score=%.2f, action=%s, severity=%s", 
					result.FinalScore, result.Action, result.Severity)
				return true, fmt.Sprintf("Content blocked by advanced moderation: %s (score: %.2f)", 
					result.Action, result.FinalScore)
			}
			
			log.Printf("Advanced moderation passed: score=%.2f, action=%s", 
				result.FinalScore, result.Action)
			return false, ""
		}
	}
	
	// Fallback to legacy moderation
	if config.AppConfig.Moderation.Enabled {
		return p.moderateContent(req.Message)
	}
	
	return false, ""
}

// ReloadConfiguration reinitializes SDK clients with updated config
func (p *Proxy) ReloadConfiguration() {
	p.sdkRouter.ReinitializeClients()
	
	// Reinitialize moderation engine with new config
	if p.moderationEngine != nil {
		p.moderationEngine.Close()
	}
	p.initModerationEngine()
}

// broadcastModerationEvent sends moderation events via WebSocket
func (p *Proxy) broadcastModerationEvent(result *moderation.AggregatedResult, req ChatRequestExtended) {
	log.Printf("🎯 broadcastModerationEvent CALLED: user=%s, score=%.6f, layers=%d", 
		req.UserID, result.FinalScore, len(result.LayerResults))
	
	if api.WSManager == nil {
		log.Printf("❌ WSManager is nil, cannot broadcast")
		return
	}
	
	// Broadcast general moderation event
	moderationMessage := api.WSMessage{
		Type:      api.MessageTypeModerationEvent,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"user_id":        req.UserID,
			"session_id":     req.SessionID,
			"final_score":    result.FinalScore,
			"final_decision": result.FinalDecision,
			"action":         result.Action,
			"severity":       result.Severity,
			"process_time":   result.ProcessTime.Milliseconds(),
			"cache_hit":      result.CacheHit,
			"layer_count":    len(result.LayerResults),
		},
	}
	
	// Check for PII detections specifically
	piiDetections := make([]map[string]interface{}, 0)
	foundPIILayer := false
	
	for _, layerResult := range result.LayerResults {
		// Track if we found a PII layer (regardless of detection)
		if layerResult.Category == moderation.CategoryPII {
			foundPIILayer = true
		}
		if layerResult.Category == moderation.CategoryPII && layerResult.Score > 0 {
			// Record PII detection metrics
			if api.MetricsCollectorInstance != nil {
				detectedTypes := []string{}
				if types, ok := layerResult.Details["detected_types"].([]string); ok {
					detectedTypes = types
				}
				
				matchCount := 0
				if count, ok := layerResult.Details["pii_matches_count"].(int); ok {
					matchCount = count
				}
				
				maskingEnabled := false
				if masked, ok := layerResult.Details["masking_enabled"].(bool); ok {
					maskingEnabled = masked
				}
				
				// Record the PII detection event
				action := "detected"
				if layerResult.Blocked {
					action = "blocked"
				}
				
				api.MetricsCollectorInstance.RecordPIIDetection(metrics.PIIMetrics{
					DetectedTypes:  detectedTypes,
					MatchCount:     matchCount,
					MaskingEnabled: maskingEnabled,
					Confidence:     layerResult.Confidence,
					Action:         action,
					UserID:         req.UserID,
					ContentLength:  len(req.Message),
				})
			}
			
			// Extract PII matches from layer details
			piiData := map[string]interface{}{
				"layer_name":  layerResult.LayerName,
				"score":       layerResult.Score,
				"confidence":  layerResult.Confidence,
				"blocked":     layerResult.Blocked,
				"reason":      layerResult.Reason,
				"process_time": layerResult.ProcessTime.Milliseconds(),
			}
			
			// Add PII matches if available (without sensitive data if masking is enabled)
			if details, ok := layerResult.Details["matches"]; ok {
				piiData["matches"] = details
			}
			
			if matchCount, ok := layerResult.Details["pii_matches_count"]; ok {
				piiData["match_count"] = matchCount
			}
			
			if detectedTypes, ok := layerResult.Details["detected_types"]; ok {
				piiData["detected_types"] = detectedTypes
			}
			
			if maskingEnabled, ok := layerResult.Details["masking_enabled"]; ok {
				piiData["masking_enabled"] = maskingEnabled
			}
			
			piiDetections = append(piiDetections, piiData)
		}
	}
	
	// Record when PII layer ran but found no PII (for accurate statistics)
	if foundPIILayer && len(piiDetections) == 0 && api.MetricsCollectorInstance != nil {
		api.MetricsCollectorInstance.RecordPIIDetection(metrics.PIIMetrics{
			DetectedTypes:  []string{},
			MatchCount:     0,
			MaskingEnabled: false,
			Confidence:     1.0, // High confidence that no PII was found
			Action:         "clean",
			UserID:         req.UserID,
			ContentLength:  len(req.Message),
		})
	}
	
	// Broadcast PII-specific event if PII was detected
	if len(piiDetections) > 0 {
		piiMessage := api.WSMessage{
			Type:      api.MessageTypePIIDetection,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data: map[string]interface{}{
				"user_id":        req.UserID,
				"session_id":     req.SessionID,
				"pii_detections": piiDetections,
				"total_matches":  len(piiDetections),
				"blocked":        result.FinalDecision,
				"severity":       result.Severity,
			},
		}
		
		api.WSManager.BroadcastMessage(piiMessage)
		log.Printf("Broadcasted PII detection event: %d detections for user %s", 
			len(piiDetections), req.UserID)
	}
	
	// Broadcast the general moderation event
	log.Printf("🔥 Broadcasting moderation event: user=%s, score=%.6f, decision=%v, timestamp=%s", 
		req.UserID, result.FinalScore, result.FinalDecision, moderationMessage.Timestamp)
	api.WSManager.BroadcastMessage(moderationMessage)
}

// convertLayerConfigs converts app config layer configs to moderation layer configs
func convertLayerConfigs(appLayers []config.AdvancedLayerConfig) []moderation.LayerConfig {
	layers := make([]moderation.LayerConfig, len(appLayers))
	for i, appLayer := range appLayers {
		layers[i] = moderation.LayerConfig{
			Name:      appLayer.Name,
			Enabled:   appLayer.Enabled,
			Weight:    appLayer.Weight,
			Threshold: appLayer.Threshold,
			Options:   appLayer.Options,
		}
	}
	return layers
}