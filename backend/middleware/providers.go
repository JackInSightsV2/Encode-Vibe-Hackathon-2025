package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"qt1-middleware/config"
)

// ProviderRouter handles routing requests to different AI providers
type ProviderRouter struct {
	client *http.Client
}

// NewProviderRouter creates a new provider router
func NewProviderRouter() *ProviderRouter {
	return &ProviderRouter{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
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
}

// RouteRequest determines which provider to use and routes the request
func (r *ProviderRouter) RouteRequest(req ChatRequestExtended) (*ProxyResponse, error) {
	// Check for legacy target_url routing first
	if req.TargetURL != "" {
		return r.routeToTargetURL(req)
	}
	
	provider := r.selectProvider(req)
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

func (r *ProviderRouter) selectProvider(req ChatRequestExtended) *config.Provider {
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

func (r *ProviderRouter) transformRequestForProvider(req ChatRequestExtended, provider *config.Provider) ([]byte, error) {
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

func (r *ProviderRouter) makeRequestWithFallback(reqBody []byte, provider *config.Provider, originalReq ChatRequestExtended) (*ProxyResponse, error) {
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

func (r *ProviderRouter) makeRequest(reqBody []byte, provider *config.Provider) (*ProxyResponse, error) {
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

func (r *ProviderRouter) getEndpointForProvider(provider *config.Provider) string {
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