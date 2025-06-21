package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	config      ProviderConfig
	client      *http.Client
	metrics     ProviderMetrics
	status      ProviderStatus
	mutex       sync.RWMutex
	startTime   time.Time
	lastRequest time.Time
}

// OpenAI API specific structures
type openAIRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIMessage     `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
	User        string              `json:"user,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []openAIChoice   `json:"choices"`
	Usage   openAIUsage      `json:"usage"`
	Error   *openAIError     `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIError struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Code    interface{} `json:"code"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		status: ProviderStatus{
			State:     StateStopped,
			Healthy:   false,
			Version:   "1.0.0",
		},
		metrics: ProviderMetrics{},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Type returns the provider type
func (p *OpenAIProvider) Type() ProviderType {
	return ProviderTypeOpenAI
}

// Version returns the provider version
func (p *OpenAIProvider) Version() string {
	return "1.0.0"
}

// Configure sets the provider configuration
func (p *OpenAIProvider) Configure(config ProviderConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Validate OpenAI-specific configuration
	if config.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	// Update HTTP client timeout
	p.client.Timeout = config.Timeout

	p.config = config
	return nil
}

// Validate validates the provider configuration
func (p *OpenAIProvider) Validate() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.config.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}

	if p.config.BaseURL == "" {
		return fmt.Errorf("OpenAI base URL is required")
	}

	return nil
}

// Start starts the provider
func (p *OpenAIProvider) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status.State == StateRunning {
		return nil // Already running
	}

	p.status.State = StateStarting
	p.startTime = time.Now()

	// Perform initial health check
	if err := p.healthCheck(ctx); err != nil {
		p.status.State = StateError
		p.status.Error = err.Error()
		return fmt.Errorf("initial health check failed: %w", err)
	}

	p.status.State = StateRunning
	p.status.Healthy = true
	p.status.StartTime = p.startTime
	p.status.Error = ""

	return nil
}

// Stop stops the provider
func (p *OpenAIProvider) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status.State == StateStopped {
		return nil // Already stopped
	}

	p.status.State = StateStopping

	// Graceful shutdown - wait for ongoing requests to complete
	select {
	case <-ctx.Done():
		// Force stop on timeout
	case <-time.After(5 * time.Second):
		// Allow up to 5 seconds for graceful shutdown
	}

	p.status.State = StateStopped
	p.status.Healthy = false

	return nil
}

// HealthCheck performs a health check
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	return p.healthCheck(ctx)
}

func (p *OpenAIProvider) healthCheck(ctx context.Context) error {
	// Simple health check using models endpoint
	url := strings.TrimSuffix(p.config.BaseURL, "/") + "/models"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("User-Agent", "QT1-Middleware/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Update last check time
	p.mutex.Lock()
	p.status.LastCheck = time.Now()
	p.mutex.Unlock()

	return nil
}

// GetStatus returns the current provider status
func (p *OpenAIProvider) GetStatus() ProviderStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	status := p.status
	if p.status.State == StateRunning {
		status.Uptime = time.Since(p.startTime)
	}

	return status
}

// GetMetrics returns the current provider metrics
func (p *OpenAIProvider) GetMetrics() ProviderMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	metrics := p.metrics

	// Calculate derived metrics
	if metrics.RequestCount > 0 {
		metrics.AverageLatency = metrics.TotalLatency / time.Duration(metrics.RequestCount)
		metrics.ErrorRate = float64(metrics.ErrorCount) / float64(metrics.RequestCount)
		metrics.SuccessRate = float64(metrics.SuccessCount) / float64(metrics.RequestCount)
	}

	// Calculate requests per second
	if !p.lastRequest.IsZero() {
		elapsed := time.Since(p.startTime).Seconds()
		if elapsed > 0 {
			metrics.RequestsPerSecond = float64(metrics.RequestCount) / elapsed
		}
	}

	return metrics
}

// SendRequest sends a chat request to OpenAI
func (p *OpenAIProvider) SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	p.mutex.Lock()
	p.lastRequest = startTime
	p.metrics.RequestCount++
	p.mutex.Unlock()

	// Transform to OpenAI format
	openaiReq := &openAIRequest{
		Model:    req.Model,
		Messages: make([]openAIMessage, len(req.Messages)),
		Stream:   req.Stream,
		Stop:     req.Stop,
		User:     req.UserID,
	}

	// Convert messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Set optional parameters
	if req.MaxTokens > 0 {
		openaiReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		openaiReq.Temperature = &req.Temperature
	}
	if req.TopP > 0 {
		openaiReq.TopP = &req.TopP
	}

	// Make request with retries
	var response *ChatResponse
	var lastErr error

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(p.config.RetryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}

		response, lastErr = p.makeRequest(ctx, openaiReq)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if providerErr, ok := lastErr.(*ProviderError); ok && !providerErr.Retryable {
			break
		}
	}

	// Update metrics
	latency := time.Since(startTime)
	p.mutex.Lock()
	p.metrics.TotalLatency += latency
	if lastErr != nil {
		p.metrics.ErrorCount++
	} else {
		p.metrics.SuccessCount++
		if response != nil {
			p.metrics.TokensUsed += int64(response.Usage.TotalTokens)
			p.metrics.InputTokens += int64(response.Usage.InputTokens)
			p.metrics.OutputTokens += int64(response.Usage.OutputTokens)
			p.metrics.TotalCost += response.Cost
		}
	}
	p.mutex.Unlock()

	if lastErr != nil {
		return nil, lastErr
	}

	response.Latency = latency
	return response, nil
}

func (p *OpenAIProvider) makeRequest(ctx context.Context, req *openAIRequest) (*ChatResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "marshal_error", 
			"Failed to marshal request", false, err)
	}

	// Create HTTP request
	url := strings.TrimSuffix(p.config.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeNetwork, "request_error", 
			"Failed to create request", true, err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	httpReq.Header.Set("User-Agent", "QT1-Middleware/1.0")

	// Add custom headers
	for key, value := range p.config.Headers {
		httpReq.Header.Set(key, value)
	}

	// Make request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeNetwork, "http_error", 
			"HTTP request failed", true, err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeNetwork, "read_error", 
			"Failed to read response", true, err)
	}

	// Handle rate limiting
	if resp.StatusCode == 429 {
		p.mutex.Lock()
		p.metrics.RateLimitHits++
		p.mutex.Unlock()
		
		return nil, NewProviderError(p.Name(), ErrorTypeRateLimit, "rate_limit", 
			"Rate limit exceeded", true, fmt.Errorf("rate limit exceeded"))
	}

	// Handle other HTTP errors
	if resp.StatusCode >= 400 {
		var errorResp openAIResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != nil {
			retryable := resp.StatusCode >= 500 || resp.StatusCode == 429
			return nil, NewProviderError(p.Name(), ErrorTypeServer, fmt.Sprintf("%d", resp.StatusCode), 
				errorResp.Error.Message, retryable, fmt.Errorf("OpenAI API error: %s", errorResp.Error.Message))
		}
		
		retryable := resp.StatusCode >= 500
		return nil, NewProviderError(p.Name(), ErrorTypeServer, fmt.Sprintf("%d", resp.StatusCode), 
			"Unknown server error", retryable, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
	}

	// Parse response
	var openaiResp openAIResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "parse_error", 
			"Failed to parse response", false, err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "no_choices", 
			"No choices in response", false, fmt.Errorf("no choices in response"))
	}

	// Calculate cost
	cost := p.calculateCost(openaiResp.Usage, req.Model)

	// Convert to standard response
	choice := openaiResp.Choices[0]
	return &ChatResponse{
		ID:           openaiResp.ID,
		Provider:     p.Name(),
		Model:        openaiResp.Model,
		Message: ChatMessage{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		},
		FinishReason: choice.FinishReason,
		Usage: TokenUsage{
			InputTokens:  openaiResp.Usage.PromptTokens,
			OutputTokens: openaiResp.Usage.CompletionTokens,
			TotalTokens:  openaiResp.Usage.TotalTokens,
		},
		Cost:      cost,
		CreatedAt: time.Now(),
	}, nil
}

// calculateCost calculates the cost of a request
func (p *OpenAIProvider) calculateCost(usage openAIUsage, model string) float64 {
	if p.config.Pricing == nil {
		// Default pricing for common models
		switch {
		case strings.Contains(model, "gpt-4"):
			return float64(usage.PromptTokens)*0.03/1000 + float64(usage.CompletionTokens)*0.06/1000
		case strings.Contains(model, "gpt-3.5-turbo"):
			return float64(usage.PromptTokens)*0.002/1000 + float64(usage.CompletionTokens)*0.002/1000
		default:
			return 0
		}
	}

	inputCost := float64(usage.PromptTokens) * p.config.Pricing.InputTokenCost / 1000
	outputCost := float64(usage.CompletionTokens) * p.config.Pricing.OutputTokenCost / 1000
	return inputCost + outputCost + p.config.Pricing.RequestCost
}

// GetModels returns available models
func (p *OpenAIProvider) GetModels() []Model {
	models := []Model{
		{
			ID:           "gpt-4",
			Name:         "GPT-4",
			Description:  "Most capable model, best for complex tasks",
			MaxTokens:    8192,
			InputCost:    0.03,
			OutputCost:   0.06,
			Capabilities: []string{"text", "reasoning"},
			Provider:     p.Name(),
		},
		{
			ID:           "gpt-4-turbo",
			Name:         "GPT-4 Turbo",
			Description:  "GPT-4 with improved speed and efficiency",
			MaxTokens:    128000,
			InputCost:    0.01,
			OutputCost:   0.03,
			Capabilities: []string{"text", "reasoning", "vision"},
			Provider:     p.Name(),
		},
		{
			ID:           "gpt-3.5-turbo",
			Name:         "GPT-3.5 Turbo",
			Description:  "Fast and efficient for most tasks",
			MaxTokens:    16384,
			InputCost:    0.002,
			OutputCost:   0.002,
			Capabilities: []string{"text"},
			Provider:     p.Name(),
		},
	}

	// Filter by configured models if specified
	if len(p.config.Models) > 0 {
		filtered := make([]Model, 0)
		for _, model := range models {
			for _, configModel := range p.config.Models {
				if strings.Contains(model.ID, configModel) || strings.Contains(configModel, model.ID) {
					filtered = append(filtered, model)
					break
				}
			}
		}
		return filtered
	}

	return models
}

// GetCapabilities returns provider capabilities
func (p *OpenAIProvider) GetCapabilities() Capabilities {
	rateLimits := RateLimit{
		RequestsPerSecond: 10,
		RequestsPerMinute: 600,
		RequestsPerHour:   10000,
		TokensPerMinute:   90000,
	}

	if p.config.RateLimit != nil {
		rateLimits.RequestsPerSecond = p.config.RateLimit.RequestsPerSecond
		rateLimits.RequestsPerMinute = p.config.RateLimit.RequestsPerMinute
		rateLimits.RequestsPerHour = p.config.RateLimit.RequestsPerHour
		rateLimits.TokensPerMinute = p.config.RateLimit.TokensPerMinute
	}

	return Capabilities{
		SupportsStreaming:     true,
		SupportsVision:        true,
		SupportsFunctionCall:  true,
		SupportsEmbeddings:    true,
		MaxContextLength:      128000,
		SupportedFormats:      []string{"text", "json", "markdown"},
		RateLimits:           rateLimits,
	}
}