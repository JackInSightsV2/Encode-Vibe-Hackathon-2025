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

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	config      ProviderConfig
	client      *http.Client
	metrics     ProviderMetrics
	status      ProviderStatus
	mutex       sync.RWMutex
	startTime   time.Time
	lastRequest time.Time
}

// Anthropic API specific structures
type anthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []anthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	System        string             `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        anthropicUsage     `json:"usage"`
	Error        *anthropicError    `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		status: ProviderStatus{
			State:   StateStopped,
			Healthy: false,
			Version: "1.0.0",
		},
		metrics: ProviderMetrics{},
	}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Type returns the provider type
func (p *AnthropicProvider) Type() ProviderType {
	return ProviderTypeAnthropic
}

// Version returns the provider version
func (p *AnthropicProvider) Version() string {
	return "1.0.0"
}

// Configure sets the provider configuration
func (p *AnthropicProvider) Configure(config ProviderConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Validate Anthropic-specific configuration
	if config.APIKey == "" {
		return fmt.Errorf("Anthropic API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
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
func (p *AnthropicProvider) Validate() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.config.APIKey == "" {
		return fmt.Errorf("Anthropic API key is required")
	}

	if p.config.BaseURL == "" {
		return fmt.Errorf("Anthropic base URL is required")
	}

	return nil
}

// Start starts the provider
func (p *AnthropicProvider) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status.State == StateRunning {
		return nil // Already running
	}

	p.status.State = StateStarting
	p.startTime = time.Now()

	// Anthropic doesn't have a simple health check endpoint like OpenAI
	// We'll just validate the configuration and mark as healthy
	if err := p.Validate(); err != nil {
		p.status.State = StateError
		p.status.Error = err.Error()
		return fmt.Errorf("validation failed: %w", err)
	}

	p.status.State = StateRunning
	p.status.Healthy = true
	p.status.StartTime = p.startTime
	p.status.Error = ""

	return nil
}

// Stop stops the provider
func (p *AnthropicProvider) Stop(ctx context.Context) error {
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
func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	// Anthropic doesn't have a dedicated health check endpoint
	// We'll perform a minimal test request
	testReq := &ChatRequest{
		UserID:    "health-check",
		SessionID: "health-check",
		Model:     "claude-3-haiku-20240307", // Use fastest model for health check
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 10,
	}

	// Create a short timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := p.SendRequest(healthCtx, testReq)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}

	// Update last check time
	p.mutex.Lock()
	p.status.LastCheck = time.Now()
	p.mutex.Unlock()

	return nil
}

// GetStatus returns the current provider status
func (p *AnthropicProvider) GetStatus() ProviderStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	status := p.status
	if p.status.State == StateRunning {
		status.Uptime = time.Since(p.startTime)
	}

	return status
}

// GetMetrics returns the current provider metrics
func (p *AnthropicProvider) GetMetrics() ProviderMetrics {
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

// SendRequest sends a chat request to Anthropic
func (p *AnthropicProvider) SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	p.mutex.Lock()
	p.lastRequest = startTime
	p.metrics.RequestCount++
	p.mutex.Unlock()

	// Transform to Anthropic format
	anthropicReq := &anthropicRequest{
		Model:         req.Model,
		Messages:      make([]anthropicMessage, 0),
		MaxTokens:     req.MaxTokens,
		Stream:        req.Stream,
		StopSequences: req.Stop,
	}

	// Default max tokens if not specified
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 1000
	}

	// Set optional parameters
	if req.Temperature > 0 {
		anthropicReq.Temperature = &req.Temperature
	}
	if req.TopP > 0 {
		anthropicReq.TopP = &req.TopP
	}

	// Convert messages, handling system message specially
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
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

		response, lastErr = p.makeRequest(ctx, anthropicReq)
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

func (p *AnthropicProvider) makeRequest(ctx context.Context, req *anthropicRequest) (*ChatResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "marshal_error",
			"Failed to marshal request", false, err)
	}

	// Create HTTP request
	url := strings.TrimSuffix(p.config.BaseURL, "/") + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeNetwork, "request_error",
			"Failed to create request", true, err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
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
		var errorResp anthropicResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != nil {
			retryable := resp.StatusCode >= 500 || resp.StatusCode == 429
			return nil, NewProviderError(p.Name(), ErrorTypeServer, fmt.Sprintf("%d", resp.StatusCode),
				errorResp.Error.Message, retryable, fmt.Errorf("Anthropic API error: %s", errorResp.Error.Message))
		}

		retryable := resp.StatusCode >= 500
		return nil, NewProviderError(p.Name(), ErrorTypeServer, fmt.Sprintf("%d", resp.StatusCode),
			"Unknown server error", retryable, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
	}

	// Parse response
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "parse_error",
			"Failed to parse response", false, err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "no_content",
			"No content in response", false, fmt.Errorf("no content in response"))
	}

	// Calculate cost
	cost := p.calculateCost(anthropicResp.Usage, req.Model)

	// Convert to standard response
	content := anthropicResp.Content[0]
	return &ChatResponse{
		ID:       anthropicResp.ID,
		Provider: p.Name(),
		Model:    anthropicResp.Model,
		Message: ChatMessage{
			Role:    anthropicResp.Role,
			Content: content.Text,
		},
		FinishReason: anthropicResp.StopReason,
		Usage: TokenUsage{
			InputTokens:  anthropicResp.Usage.InputTokens,
			OutputTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:  anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
		Cost:      cost,
		CreatedAt: time.Now(),
	}, nil
}

// calculateCost calculates the cost of a request
func (p *AnthropicProvider) calculateCost(usage anthropicUsage, model string) float64 {
	if p.config.Pricing == nil {
		// Default pricing for Claude models
		switch {
		case strings.Contains(model, "claude-3-opus"):
			return float64(usage.InputTokens)*0.015/1000 + float64(usage.OutputTokens)*0.075/1000
		case strings.Contains(model, "claude-3-sonnet"):
			return float64(usage.InputTokens)*0.003/1000 + float64(usage.OutputTokens)*0.015/1000
		case strings.Contains(model, "claude-3-haiku"):
			return float64(usage.InputTokens)*0.00025/1000 + float64(usage.OutputTokens)*0.00125/1000
		default:
			return 0
		}
	}

	inputCost := float64(usage.InputTokens) * p.config.Pricing.InputTokenCost / 1000
	outputCost := float64(usage.OutputTokens) * p.config.Pricing.OutputTokenCost / 1000
	return inputCost + outputCost + p.config.Pricing.RequestCost
}

// GetModels returns available models
func (p *AnthropicProvider) GetModels() []Model {
	models := []Model{
		{
			ID:           "claude-3-opus-20240229",
			Name:         "Claude 3 Opus",
			Description:  "Most capable model for complex tasks",
			MaxTokens:    200000,
			InputCost:    0.015,
			OutputCost:   0.075,
			Capabilities: []string{"text", "reasoning", "analysis"},
			Provider:     p.Name(),
		},
		{
			ID:           "claude-3-sonnet-20240229",
			Name:         "Claude 3 Sonnet",
			Description:  "Balanced performance and speed",
			MaxTokens:    200000,
			InputCost:    0.003,
			OutputCost:   0.015,
			Capabilities: []string{"text", "reasoning"},
			Provider:     p.Name(),
		},
		{
			ID:           "claude-3-haiku-20240307",
			Name:         "Claude 3 Haiku",
			Description:  "Fastest model for simple tasks",
			MaxTokens:    200000,
			InputCost:    0.00025,
			OutputCost:   0.00125,
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
func (p *AnthropicProvider) GetCapabilities() Capabilities {
	rateLimits := RateLimit{
		RequestsPerSecond: 5,
		RequestsPerMinute: 300,
		RequestsPerHour:   4000,
		TokensPerMinute:   40000,
	}

	if p.config.RateLimit != nil {
		rateLimits.RequestsPerSecond = p.config.RateLimit.RequestsPerSecond
		rateLimits.RequestsPerMinute = p.config.RateLimit.RequestsPerMinute
		rateLimits.RequestsPerHour = p.config.RateLimit.RequestsPerHour
		rateLimits.TokensPerMinute = p.config.RateLimit.TokensPerMinute
	}

	return Capabilities{
		SupportsStreaming:     true,
		SupportsVision:        false, // Claude 3 supports vision but not implemented yet
		SupportsFunctionCall:  false, // Not yet supported
		SupportsEmbeddings:    false, // Anthropic doesn't provide embeddings
		MaxContextLength:      200000,
		SupportedFormats:      []string{"text", "markdown"},
		RateLimits:           rateLimits,
	}
}