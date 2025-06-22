package middleware

import (
	"context"
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
	"qt1-middleware/opik"
	"qt1-middleware/utils"

	"github.com/sashabaranov/go-openai"
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

// ModerationEngineInterface defines the interface that both regular and Opik engines must implement
type ModerationEngineInterface interface {
	IsEnabled() bool
	Moderate(content string, context moderation.ModerationContext) (*moderation.AggregatedResult, error)
	GetStats() moderation.ModerationStats
	GetLayerNames() []string
	GetEnabledLayers() []moderation.ModerationLayer
	GetLayerInfo(layerName string) (weight float64, enabled bool, found bool)
	GetConfig() *moderation.AdvancedModerationConfig
	RegisterLayer(layer moderation.ModerationLayer) error
	SetLayers(layers []moderation.ModerationLayer)
	ReloadConfig() error
	Close()
}

// Proxy handles HTTP requests and routes them through moderation layers
type Proxy struct {
	client                  *http.Client
	promptInjectionDetector *PromptInjectionDetector
	providerRouter          *ProviderRouter
	sdkRouter               *SDKProviderRouter
	moderationEngine        ModerationEngineInterface
	moderationEngineOpik    *moderation.EngineWithOpik
	opikClient              *opik.OpikClient
}

// GetSDKRouter returns the SDK router instance
func (p *Proxy) GetSDKRouter() *SDKProviderRouter {
	return p.sdkRouter
}

func NewProxy() *Proxy {
	p := &Proxy{
		client:                  &http.Client{Timeout: time.Second * 30},
		promptInjectionDetector: NewPromptInjectionDetector(),
		providerRouter:          NewProviderRouter(),
		sdkRouter:               NewSDKProviderRouter(),
	}
	
	p.initModerationEngine()
	return p
}

// SetOpikClient sets the Opik client for the proxy
func (p *Proxy) SetOpikClient(client *opik.OpikClient) {
	fmt.Printf("DEBUG: SetOpikClient called - client nil? %t\n", client == nil)
	p.opikClient = client
	if client != nil {
		fmt.Printf("DEBUG: Opik client set successfully for logging moderation results\n")
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
	fmt.Printf("DEBUG: initModerationEngine called - opikClient nil? %t\n", p.opikClient == nil)
	
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
	
	// Convert to config format for EngineWithOpik
	opikModerationConfig := config.AdvancedModerationConfig{
		Enabled: moderationConfig.Enabled,
		Layers: convertToAppLayerConfigs(moderationConfig.Layers),
		Thresholds: config.ThresholdConfig{
			Low:      moderationConfig.Thresholds.Low,
			Medium:   moderationConfig.Thresholds.Medium,
			High:     moderationConfig.Thresholds.High,
			Critical: moderationConfig.Thresholds.Critical,
		},
		Actions: config.ActionConfig{
			Low:      moderationConfig.Actions.Low,
			Medium:   moderationConfig.Actions.Medium,
			High:     moderationConfig.Actions.High,
			Critical: moderationConfig.Actions.Critical,
		},
		Cache: config.CacheConfig{
			Enabled:    moderationConfig.Cache.Enabled,
			TTLMinutes: moderationConfig.Cache.TTLMinutes,
			MaxEntries: moderationConfig.Cache.MaxEntries,
		},
		Analytics: config.AnalyticsConfig{
			Enabled:           moderationConfig.Analytics.Enabled,
			CollectDetails:    moderationConfig.Analytics.CollectDetails,
			RetentionDays:     moderationConfig.Analytics.RetentionDays,
			EnablePerformance: moderationConfig.Analytics.EnablePerformance,
		},
	}
	
	// If Opik client is available, use the Opik-enabled engine
	if p.opikClient != nil {
		fmt.Printf("DEBUG: Using Opik-enabled engine path\n")
		log.Printf("Initializing Opik-enabled moderation engine")
		
		// Initialize OpenAI client for LLM layers if needed
		var openaiClient *openai.Client
		if openaiProvider, exists := config.AppConfig.Providers["openai"]; exists && openaiProvider.APIKey != "" {
			openaiClient = openai.NewClient(openaiProvider.APIKey)
		}
		
		engine, err := moderation.NewEngineWithOpik(opikModerationConfig, p.opikClient, openaiClient)
		if err != nil {
			log.Printf("Failed to create Opik-enabled moderation engine: %v", err)
			// Fall back to regular engine
			p.initRegularModerationEngine(moderationConfig)
			return
		}
		
		p.moderationEngineOpik = engine
		
		// Set global instance for API handlers - we'll use a wrapper to make it compatible
		wrapperEngine := &ModerationEngineWrapper{opikEngine: engine}
		api.ModerationEngineInstance = wrapperEngine
		
		// IMPORTANT: Set the moderationEngine field to the wrapper so runModeration uses it
		p.moderationEngine = wrapperEngine
		
		log.Printf("Opik-enabled moderation engine initialized and set as primary engine")
		return
	}
	
	// Fall back to regular moderation engine
	fmt.Printf("DEBUG: Using regular engine fallback (no Opik client)\n")
	log.Printf("Initializing regular moderation engine (no Opik client)")
	p.initRegularModerationEngine(moderationConfig)
}

// initRegularModerationEngine initializes the regular moderation engine
func (p *Proxy) initRegularModerationEngine(moderationConfig *moderation.AdvancedModerationConfig) {
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
	
	log.Printf("Regular moderation engine initialized with %d layers", len(engine.GetLayerNames()))
}

// runModeration runs content through moderation (advanced or legacy)
func (p *Proxy) runModeration(req ChatRequestExtended) (bool, string) {
	log.Printf("DEBUG: runModeration called - engine nil? %v", p.moderationEngine == nil)
	if p.moderationEngine != nil {
		log.Printf("DEBUG: moderation engine exists, enabled? %v", p.moderationEngine.IsEnabled())
	}
	
	// Try advanced moderation first
	if p.moderationEngine != nil && p.moderationEngine.IsEnabled() {
		// Generate a unique request ID for this moderation request
		requestID := fmt.Sprintf("%s-%s-%d", req.UserID, req.SessionID, time.Now().UnixNano())
		
		context := moderation.ModerationContext{
			UserID:    req.UserID,
			SessionID: req.SessionID,
			RequestID: requestID,
			Timestamp: time.Now(),
			Metadata:  map[string]interface{}{"endpoint": "/chat"},
		}
		
		result, err := p.moderationEngine.Moderate(req.Message, context)
		if err != nil {
			log.Printf("Advanced moderation failed: %v", err)
			log.Printf("Falling back to legacy moderation")
		} else {
			// Add Opik logging step - log the moderation result to Opik
			if p.opikClient != nil {
				go p.logModerationToOpik(req.Message, context, result)
			}
			
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

// convertToAppLayerConfigs converts moderation layer configs to app config layer configs
func convertToAppLayerConfigs(moderationLayers []moderation.LayerConfig) []config.AdvancedLayerConfig {
	layers := make([]config.AdvancedLayerConfig, len(moderationLayers))
	for i, layer := range moderationLayers {
		layers[i] = config.AdvancedLayerConfig{
			Name:      layer.Name,
			Enabled:   layer.Enabled,
			Weight:    layer.Weight,
			Threshold: layer.Threshold,
			Options:   layer.Options,
		}
	}
	return layers
}

// ModerationEngineWrapper wraps the Opik-enabled engine to make it compatible with the regular engine interface
type ModerationEngineWrapper struct {
	opikEngine *moderation.EngineWithOpik
}

func (w *ModerationEngineWrapper) GetStats() moderation.ModerationStats {
	return w.opikEngine.GetStats()
}

func (w *ModerationEngineWrapper) GetLayerNames() []string {
	return w.opikEngine.GetLayerNames()
}

func (w *ModerationEngineWrapper) GetEnabledLayers() []moderation.ModerationLayer {
	return w.opikEngine.GetEnabledLayers()
}

func (w *ModerationEngineWrapper) GetLayerInfo(layerName string) (weight float64, enabled bool, found bool) {
	return w.opikEngine.GetLayerInfo(layerName)
}

func (w *ModerationEngineWrapper) IsEnabled() bool {
	return w.opikEngine.IsEnabled()
}

func (w *ModerationEngineWrapper) GetConfig() *moderation.AdvancedModerationConfig {
	return w.opikEngine.GetConfig()
}

func (w *ModerationEngineWrapper) Moderate(content string, moderationContext moderation.ModerationContext) (*moderation.AggregatedResult, error) {
	fmt.Printf("DEBUG: ModerationEngineWrapper.Moderate called with content: '%s'\n", content)
	fmt.Printf("DEBUG: About to call ModerateWithOpik\n")
	ctx := context.Background()
	result, err := w.opikEngine.ModerateWithOpik(ctx, content, moderationContext)
	if err != nil {
		fmt.Printf("DEBUG: ModerateWithOpik returned error: %v\n", err)
	} else {
		fmt.Printf("DEBUG: ModerateWithOpik completed successfully\n")
	}
	return result, err
}

func (w *ModerationEngineWrapper) Close() {
	w.opikEngine.Close()
}

func (w *ModerationEngineWrapper) RegisterLayer(layer moderation.ModerationLayer) error {
	return w.opikEngine.RegisterLayer(layer)
}

func (w *ModerationEngineWrapper) SetLayers(layers []moderation.ModerationLayer) {
	w.opikEngine.SetLayers(layers)
}

func (w *ModerationEngineWrapper) ReloadConfig() error {
	return w.opikEngine.ReloadConfig()
}

func (p *Proxy) logModerationToOpik(message string, moderationCtx moderation.ModerationContext, result *moderation.AggregatedResult) {
	fmt.Printf("DEBUG: logModerationToOpik called for user %s\n", moderationCtx.UserID)
	
	if p.opikClient == nil {
		fmt.Printf("DEBUG: Opik client is nil, skipping logging\n")
		return
	}
	
	// Create a trace for this moderation request
	request := opik.ModerationRequest{
		ID:        moderationCtx.RequestID,
		Content:   message,
		UserID:    moderationCtx.UserID,
		SessionID: moderationCtx.SessionID,
		IPAddress: moderationCtx.IPAddress,
		Provider:  "qt1-middleware",
		Endpoint:  "/chat",
		Timestamp: moderationCtx.Timestamp,
	}
	
	ctx := context.Background()
	trace, err := p.opikClient.TraceModeration(ctx, request)
	if err != nil {
		fmt.Printf("ERROR: Failed to create Opik trace: %v\n", err)
		return
	}
	
	fmt.Printf("DEBUG: Created Opik trace %s for moderation\n", trace.ID)
	
	// Add spans for each moderation layer that ran
	for _, layerResult := range result.LayerResults {
		span := trace.StartSpan(layerResult.LayerName, opik.SpanOptions{
			Input: map[string]interface{}{
				"content": message,
				"layer":   layerResult.LayerName,
			},
			Metadata: map[string]interface{}{
				"span_type": "moderation_layer",
				"layer_name": layerResult.LayerName,
			},
		})
		
		// Set span output
		span.Output = map[string]interface{}{
			"score":       layerResult.Score,
			"confidence":  layerResult.Confidence,
			"blocked":     layerResult.Blocked,
			"reason":      layerResult.Reason,
			"category":    layerResult.Category,
			"process_time": layerResult.ProcessTime.Milliseconds(),
		}
		
		// End the span
		span.End()
	}
	
	// End the main trace with the final result
	err = p.opikClient.EndTrace(trace, map[string]interface{}{
		"allowed":        !result.FinalDecision,
		"final_score":    result.FinalScore,
		"severity":       result.Severity,
		"action":         result.Action,
		"layers_checked": len(result.LayerResults),
		"cache_hit":      result.CacheHit,
		"total_process_time": result.ProcessTime.Milliseconds(),
	})
	
	if err != nil {
		fmt.Printf("ERROR: Failed to end Opik trace: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Successfully logged moderation to Opik trace %s\n", trace.ID)
	}
}