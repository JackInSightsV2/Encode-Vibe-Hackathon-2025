package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"qt1-middleware/config"
	"qt1-middleware/providers"
)

// ProviderRouter handles routing requests to different AI providers
// Supports both legacy config-based routing and new provider management system
type ProviderRouter struct {
	client         *http.Client
	providerManager *providers.ProviderManager
	useEnhanced    bool // Whether to use enhanced provider system
	adapter        *providers.ConfigAdapter
}

// NewProviderRouter creates a new provider router
func NewProviderRouter() *ProviderRouter {
	return &ProviderRouter{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		adapter: providers.DefaultAdapter,
	}
}

// NewProviderRouterWithManager creates a new provider router with enhanced provider management
func NewProviderRouterWithManager(manager *providers.ProviderManager) *ProviderRouter {
	return &ProviderRouter{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		providerManager: manager,
		useEnhanced:     true,
		adapter:         providers.DefaultAdapter,
	}
}

// InitializeProviders initializes the provider management system
func (r *ProviderRouter) InitializeProviders() error {
	if r.providerManager != nil {
		return nil // Already initialized
	}

	// Create provider manager
	managerConfig := &providers.ManagerConfig{
		AutoStart:           config.AppConfig.ProviderManager.AutoStart,
		StartTimeout:        config.AppConfig.ProviderManager.StartTimeout,
		StopTimeout:         config.AppConfig.ProviderManager.StopTimeout,
		HealthCheckEnabled:  config.AppConfig.ProviderManager.HealthCheckEnabled,
		HealthCheckInterval: config.AppConfig.ProviderManager.HealthCheckInterval,
	}

	r.providerManager = providers.NewProviderManager(managerConfig)

	// Load provider configurations
	providerConfigs := r.adapter.LoadProviderConfigsFromAppConfig()

	// Initialize with configurations
	if err := r.providerManager.Initialize(providerConfigs); err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Start the provider manager if auto-start is enabled
	if config.AppConfig.ProviderManager.AutoStart {
		ctx, cancel := context.WithTimeout(context.Background(), config.AppConfig.ProviderManager.StartTimeout)
		defer cancel()

		if err := r.providerManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start provider manager: %w", err)
		}
	}

	r.useEnhanced = len(providerConfigs) > 0
	return nil
}

// ChatRequestExtended extends the basic ChatRequest with provider/model selection
type ChatRequestExtended struct {
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message"`
	Provider  string                 `json:"provider,omitempty"`
	Model     string                 `json:"model,omitempty"`
	TargetURL string                 `json:"target_url,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	
	// Additional fields for enhanced provider system
	Messages      []map[string]string `json:"messages,omitempty"` // Simplified for compatibility
	SystemPrompt  string              `json:"system_prompt,omitempty"`
	MaxTokens     int                 `json:"max_tokens,omitempty"`
	Temperature   float64             `json:"temperature,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
}

// RouteRequest determines which provider to use and routes the request
func (r *ProviderRouter) RouteRequest(req ChatRequestExtended) (*ProxyResponse, error) {
	// Check for legacy target_url routing first
	if req.TargetURL != "" {
		return r.routeToTargetURL(req)
	}
	
	provider := r.selectLegacyProvider(req)
	if provider == nil {
		return nil, fmt.Errorf("no suitable provider found")
	}
	
	// Transform request for the specific provider
	transformedReq, err := r.transformRequestForProvider(req, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to transform request: %w", err)
	}
	
	// Make the request with fallback logic
	return r.makeRequestWithFallback(transformedReq, provider, req)
}

func (r *ProviderRouter) selectLegacyProvider(req ChatRequestExtended) *config.LegacyProvider {
	// 1. Use explicitly specified provider
	if req.Provider != "" {
		if provider, ok := config.AppConfig.Providers[req.Provider]; ok {
			return &provider
		}
	}
	
	// 2. Use model-based routing
	if req.Model != "" {
		if providerName, ok := config.AppConfig.Routing.ModelRouting[req.Model]; ok {
			if provider, ok := config.AppConfig.Providers[providerName]; ok {
				return &provider
			}
		}
		
		// Check if any provider supports this model
		for _, provider := range config.AppConfig.Providers {
			for _, model := range provider.Models {
				if strings.Contains(model, req.Model) || strings.Contains(req.Model, model) {
					return &provider
				}
			}
		}
	}
	
	// 3. Use default provider
	if defaultProvider := config.AppConfig.Routing.DefaultProvider; defaultProvider != "" {
		if provider, ok := config.AppConfig.Providers[defaultProvider]; ok {
			return &provider
		}
	}
	
	// 4. Use first available provider
	for _, provider := range config.AppConfig.Providers {
		return &provider
	}
	
	return nil
}

func (r *ProviderRouter) transformRequestForProvider(req ChatRequestExtended, provider *config.LegacyProvider) ([]byte, error) {
	switch {
	case strings.Contains(provider.BaseURL, "openai.com"):
		return r.transformForOpenAI(req)
	case strings.Contains(provider.BaseURL, "anthropic.com"):
		return r.transformForAnthropic(req)
	default:
		// Generic OpenAI-compatible format
		return r.transformForOpenAI(req)
	}
}

func (r *ProviderRouter) transformForOpenAI(req ChatRequestExtended) ([]byte, error) {
	model := req.Model
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	
	// OpenAI API format
	openaiReq := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.Message,
			},
		},
		"max_tokens":  2000,
		"temperature": 0.7,
	}
	
	// Add any custom settings
	for key, value := range req.Settings {
		openaiReq[key] = value
	}
	
	return json.Marshal(openaiReq)
}

func (r *ProviderRouter) transformForAnthropic(req ChatRequestExtended) ([]byte, error) {
	model := req.Model
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}
	
	// Anthropic API format
	anthropicReq := map[string]interface{}{
		"model":      model,
		"max_tokens": 2000,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.Message,
			},
		},
	}
	
	// Add any custom settings
	for key, value := range req.Settings {
		anthropicReq[key] = value
	}
	
	return json.Marshal(anthropicReq)
}

func (r *ProviderRouter) makeRequestWithFallback(reqBody []byte, provider *config.LegacyProvider, originalReq ChatRequestExtended) (*ProxyResponse, error) {
	// Try primary provider first
	resp, err := r.makeRequest(reqBody, provider)
	if err == nil {
		return resp, nil
	}
	
	// Try fallback chain
	for _, fallbackName := range config.AppConfig.Routing.FallbackChain {
		if fallbackName == provider.Name {
			continue // Skip the provider we already tried
		}
		
		if fallbackProvider, ok := config.AppConfig.Providers[fallbackName]; ok {
			// Transform request for fallback provider
			fallbackReq, err := r.transformRequestForProvider(originalReq, &fallbackProvider)
			if err != nil {
				continue
			}
			
			resp, err := r.makeRequest(fallbackReq, &fallbackProvider)
			if err == nil {
				return resp, nil
			}
		}
	}
	
	return nil, fmt.Errorf("all providers failed, last error: %w", err)
}

func (r *ProviderRouter) makeRequest(reqBody []byte, provider *config.LegacyProvider) (*ProxyResponse, error) {
	// Determine endpoint
	endpoint := r.getEndpointForProvider(provider)
	
	// Create HTTP request
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	
	// Set headers
	for key, value := range provider.Headers {
		httpReq.Header.Set(key, value)
	}
	
	// Set authentication
	if provider.APIKey != "" {
		if strings.Contains(provider.BaseURL, "openai.com") {
			httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
		} else if strings.Contains(provider.BaseURL, "anthropic.com") {
			httpReq.Header.Set("x-api-key", provider.APIKey)
		} else {
			// Generic bearer token
			httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
		}
	}
	
	// Set timeout
	if provider.Timeout > 0 {
		r.client.Timeout = time.Duration(provider.Timeout) * time.Second
	}
	
	// Make request
	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}

func (r *ProviderRouter) getEndpointForProvider(provider *config.LegacyProvider) string {
	baseURL := strings.TrimSuffix(provider.BaseURL, "/")
	
	switch {
	case strings.Contains(provider.BaseURL, "openai.com"):
		return baseURL + "/chat/completions"
	case strings.Contains(provider.BaseURL, "anthropic.com"):
		return baseURL + "/messages"
	default:
		// Generic OpenAI-compatible endpoint
		return baseURL + "/chat/completions"
	}
}

// GetAvailableProviders returns list of configured providers with their models
func (r *ProviderRouter) GetAvailableProviders() map[string]interface{} {
	providers := make(map[string]interface{})
	
	for name, provider := range config.AppConfig.Providers {
		providers[name] = map[string]interface{}{
			"name":    provider.Name,
			"models":  provider.Models,
			"baseURL": provider.BaseURL,
			"hasKey":  provider.APIKey != "",
		}
	}
	
	return providers
}

// routeToTargetURL handles legacy target_url routing for backwards compatibility
func (r *ProviderRouter) routeToTargetURL(req ChatRequestExtended) (*ProxyResponse, error) {
	// Create a simple JSON payload for target service
	payload := map[string]interface{}{
		"user_id":    req.UserID,
		"session_id": req.SessionID,
		"message":    req.Message,
	}
	
	// Add any custom settings
	for key, value := range req.Settings {
		payload[key] = value
	}
	
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequest("POST", req.TargetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "QT-1-Middleware/1.0")
	
	// Make request
	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	return &ProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}

// convertToProviderRequest converts ChatRequestExtended to providers.ChatRequest
func (r *ProviderRouter) convertToProviderRequest(req ChatRequestExtended) *providers.ChatRequest {
	chatReq := &providers.ChatRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}

	// Convert messages
	if len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			chatReq.Messages = append(chatReq.Messages, providers.ChatMessage{
				Role:    msg["role"],
				Content: msg["content"],
			})
		}
	} else {
		// Create message from simple text
		chatReq.Messages = []providers.ChatMessage{
			{
				Role:    "user",
				Content: req.Message,
			},
		}
	}

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		systemMsg := providers.ChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		}
		chatReq.Messages = append([]providers.ChatMessage{systemMsg}, chatReq.Messages...)
	}

	// Set defaults if not provided
	if chatReq.Temperature == 0 {
		chatReq.Temperature = 0.7
	}
	if chatReq.MaxTokens == 0 {
		chatReq.MaxTokens = 2000
	}

	return chatReq
}

// convertFromProviderResponse converts providers.ChatResponse to ProxyResponse
func (r *ProviderRouter) convertFromProviderResponse(resp *providers.ChatResponse) (*ProxyResponse, error) {
	// Create response structure compatible with middleware expectations
	responseData := map[string]interface{}{
		"id":      resp.ID,
		"model":   resp.Model,
		"provider": resp.Provider,
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":    resp.Message.Role,
					"content": resp.Message.Content,
				},
				"finish_reason": resp.FinishReason,
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
		"created": resp.CreatedAt.Unix(),
		"metadata": map[string]interface{}{
			"latency_ms": resp.Latency.Milliseconds(),
			"cost":       resp.Cost,
		},
	}

	body, err := json.Marshal(responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &ProxyResponse{
		StatusCode: 200,
		Body:       body,
	}, nil
}

// GetProviderManager returns the provider manager (for external use)
func (r *ProviderRouter) GetProviderManager() *providers.ProviderManager {
	return r.providerManager
}

// IsUsingEnhancedSystem returns whether the router is using the enhanced provider system
func (r *ProviderRouter) IsUsingEnhancedSystem() bool {
	return r.useEnhanced && r.providerManager != nil
}

// GetProviderStatus returns status information for all providers
func (r *ProviderRouter) GetProviderStatus() map[string]interface{} {
	if r.IsUsingEnhancedSystem() {
		// Use enhanced provider system
		statusMap := r.providerManager.GetProviderStatus()
		result := make(map[string]interface{})
		
		for name, status := range statusMap {
			result[name] = map[string]interface{}{
				"state":      string(status.State),
				"healthy":    status.Healthy,
				"uptime":     status.Uptime.String(),
				"last_check": status.LastCheck.Format(time.RFC3339),
				"error":      status.Error,
				"version":    status.Version,
			}
		}
		
		return result
	} else {
		// Use legacy system
		return r.GetAvailableProviders()
	}
}

// GetProviderMetrics returns metrics for all providers
func (r *ProviderRouter) GetProviderMetrics() map[string]interface{} {
	if r.IsUsingEnhancedSystem() {
		metricsMap := r.providerManager.GetProviderMetrics()
		result := make(map[string]interface{})
		
		for name, metrics := range metricsMap {
			result[name] = map[string]interface{}{
				"request_count":       metrics.RequestCount,
				"success_count":       metrics.SuccessCount,
				"error_count":         metrics.ErrorCount,
				"average_latency_ms":  metrics.AverageLatency.Milliseconds(),
				"requests_per_second": metrics.RequestsPerSecond,
				"total_cost":          metrics.TotalCost,
				"tokens_used":         metrics.TokensUsed,
				"rate_limit_hits":     metrics.RateLimitHits,
			}
		}
		
		return result
	} else {
		// Legacy system doesn't have detailed metrics
		return map[string]interface{}{
			"message": "Metrics not available in legacy mode",
		}
	}
}