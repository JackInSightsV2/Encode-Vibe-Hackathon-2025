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

// LocalProvider implements the Provider interface for local LLM servers (Ollama, LM Studio, etc.)
type LocalProvider struct {
	config      ProviderConfig
	client      *http.Client
	metrics     ProviderMetrics
	status      ProviderStatus
	mutex       sync.RWMutex
	startTime   time.Time
	lastRequest time.Time
}

// Local LLM API structures (OpenAI-compatible format)
type localRequest struct {
	Model       string         `json:"model"`
	Messages    []localMessage `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	TopP        *float64       `json:"top_p,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Stop        []string       `json:"stop,omitempty"`
}

type localMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type localResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []localChoice `json:"choices"`
	Usage   localUsage    `json:"usage"`
	Error   *localError   `json:"error,omitempty"`
}

type localChoice struct {
	Index        int          `json:"index"`
	Message      localMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

type localUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type localError struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Code    interface{} `json:"code"`
}

// Ollama-specific structures (for health check)
type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Digest   string `json:"digest"`
	Modified string `json:"modified"`
}

// NewLocalProvider creates a new Local LLM provider
func NewLocalProvider() *LocalProvider {
	return &LocalProvider{
		client: &http.Client{
			Timeout: 120 * time.Second, // Local LLMs can be slower
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
func (p *LocalProvider) Name() string {
	return "local"
}

// Type returns the provider type
func (p *LocalProvider) Type() ProviderType {
	return ProviderTypeLocal
}

// Version returns the provider version
func (p *LocalProvider) Version() string {
	return "1.0.0"
}

// Configure sets the provider configuration
func (p *LocalProvider) Configure(config ProviderConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434" // Default Ollama URL
	}

	if config.Timeout == 0 {
		config.Timeout = 120 * time.Second // Local LLMs need more time
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 2 // Fewer retries for local
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 2 * time.Second
	}

	// Update HTTP client timeout
	p.client.Timeout = config.Timeout

	p.config = config
	return nil
}

// Validate validates the provider configuration
func (p *LocalProvider) Validate() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.config.BaseURL == "" {
		return fmt.Errorf("Local LLM base URL is required")
	}

	return nil
}

// Start starts the provider
func (p *LocalProvider) Start(ctx context.Context) error {
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
func (p *LocalProvider) Stop(ctx context.Context) error {
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
	case <-time.After(10 * time.Second):
		// Allow up to 10 seconds for graceful shutdown (local requests might take longer)
	}

	p.status.State = StateStopped
	p.status.Healthy = false

	return nil
}

// HealthCheck performs a health check
func (p *LocalProvider) HealthCheck(ctx context.Context) error {
	return p.healthCheck(ctx)
}

func (p *LocalProvider) healthCheck(ctx context.Context) error {
	// Try to connect to the local server
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	
	// Try Ollama API first
	if err := p.checkOllamaHealth(ctx, baseURL); err == nil {
		p.mutex.Lock()
		p.status.LastCheck = time.Now()
		p.mutex.Unlock()
		return nil
	}

	// Try OpenAI-compatible endpoint
	if err := p.checkOpenAICompatibleHealth(ctx, baseURL); err == nil {
		p.mutex.Lock()
		p.status.LastCheck = time.Now()
		p.mutex.Unlock()
		return nil
	}

	return fmt.Errorf("local LLM server is not responding at %s", baseURL)
}

func (p *LocalProvider) checkOllamaHealth(ctx context.Context, baseURL string) error {
	url := baseURL + "/api/tags"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "QT1-Middleware/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama health check failed: status %d", resp.StatusCode)
	}

	return nil
}

func (p *LocalProvider) checkOpenAICompatibleHealth(ctx context.Context, baseURL string) error {
	url := baseURL + "/v1/models"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "QT1-Middleware/1.0")
	
	// Add API key if provided (some local servers require it)
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI-compatible health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// GetStatus returns the current provider status
func (p *LocalProvider) GetStatus() ProviderStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	status := p.status
	if p.status.State == StateRunning {
		status.Uptime = time.Since(p.startTime)
	}

	return status
}

// GetMetrics returns the current provider metrics
func (p *LocalProvider) GetMetrics() ProviderMetrics {
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

// SendRequest sends a chat request to the local LLM
func (p *LocalProvider) SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	p.mutex.Lock()
	p.lastRequest = startTime
	p.metrics.RequestCount++
	p.mutex.Unlock()

	// Transform to local LLM format
	localReq := &localRequest{
		Model:    req.Model,
		Messages: make([]localMessage, len(req.Messages)),
		Stream:   req.Stream,
		Stop:     req.Stop,
	}

	// Convert messages
	for i, msg := range req.Messages {
		localReq.Messages[i] = localMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Set optional parameters
	if req.MaxTokens > 0 {
		localReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		localReq.Temperature = &req.Temperature
	}
	if req.TopP > 0 {
		localReq.TopP = &req.TopP
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

		response, lastErr = p.makeRequest(ctx, localReq)
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

func (p *LocalProvider) makeRequest(ctx context.Context, req *localRequest) (*ChatResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "marshal_error",
			"Failed to marshal request", false, err)
	}

	// Create HTTP request
	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := baseURL + "/v1/chat/completions"
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeNetwork, "request_error",
			"Failed to create request", true, err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "QT1-Middleware/1.0")

	// Add API key if provided
	if p.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

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

	// Handle rate limiting (some local servers may implement this)
	if resp.StatusCode == 429 {
		p.mutex.Lock()
		p.metrics.RateLimitHits++
		p.mutex.Unlock()

		return nil, NewProviderError(p.Name(), ErrorTypeRateLimit, "rate_limit",
			"Rate limit exceeded", true, fmt.Errorf("rate limit exceeded"))
	}

	// Handle other HTTP errors
	if resp.StatusCode >= 400 {
		var errorResp localResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != nil {
			retryable := resp.StatusCode >= 500 || resp.StatusCode == 429
			return nil, NewProviderError(p.Name(), ErrorTypeServer, fmt.Sprintf("%d", resp.StatusCode),
				errorResp.Error.Message, retryable, fmt.Errorf("Local LLM API error: %s", errorResp.Error.Message))
		}

		retryable := resp.StatusCode >= 500
		return nil, NewProviderError(p.Name(), ErrorTypeServer, fmt.Sprintf("%d", resp.StatusCode),
			"Unknown server error", retryable, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
	}

	// Parse response
	var localResp localResponse
	if err := json.Unmarshal(respBody, &localResp); err != nil {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "parse_error",
			"Failed to parse response", false, err)
	}

	if len(localResp.Choices) == 0 {
		return nil, NewProviderError(p.Name(), ErrorTypeValidation, "no_choices",
			"No choices in response", false, fmt.Errorf("no choices in response"))
	}

	// Calculate cost (local LLMs are typically free)
	cost := p.calculateCost(localResp.Usage)

	// Convert to standard response
	choice := localResp.Choices[0]
	return &ChatResponse{
		ID:       localResp.ID,
		Provider: p.Name(),
		Model:    localResp.Model,
		Message: ChatMessage{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		},
		FinishReason: choice.FinishReason,
		Usage: TokenUsage{
			InputTokens:  localResp.Usage.PromptTokens,
			OutputTokens: localResp.Usage.CompletionTokens,
			TotalTokens:  localResp.Usage.TotalTokens,
		},
		Cost:      cost,
		CreatedAt: time.Now(),
	}, nil
}

// calculateCost calculates the cost of a request (typically $0 for local LLMs)
func (p *LocalProvider) calculateCost(usage localUsage) float64 {
	if p.config.Pricing == nil {
		return 0 // Local LLMs are typically free
	}

	inputCost := float64(usage.PromptTokens) * p.config.Pricing.InputTokenCost / 1000
	outputCost := float64(usage.CompletionTokens) * p.config.Pricing.OutputTokenCost / 1000
	return inputCost + outputCost + p.config.Pricing.RequestCost
}

// GetModels returns available models
func (p *LocalProvider) GetModels() []Model {
	// Default models for common local LLM setups
	models := []Model{
		{
			ID:           "llama2",
			Name:         "Llama 2",
			Description:  "Meta's Llama 2 model",
			MaxTokens:    4096,
			InputCost:    0,
			OutputCost:   0,
			Capabilities: []string{"text"},
			Provider:     p.Name(),
		},
		{
			ID:           "llama2:13b",
			Name:         "Llama 2 13B",
			Description:  "Larger Llama 2 model with 13B parameters",
			MaxTokens:    4096,
			InputCost:    0,
			OutputCost:   0,
			Capabilities: []string{"text"},
			Provider:     p.Name(),
		},
		{
			ID:           "mistral",
			Name:         "Mistral",
			Description:  "Mistral 7B model",
			MaxTokens:    8192,
			InputCost:    0,
			OutputCost:   0,
			Capabilities: []string{"text"},
			Provider:     p.Name(),
		},
		{
			ID:           "codellama",
			Name:         "Code Llama",
			Description:  "Specialized model for code generation",
			MaxTokens:    4096,
			InputCost:    0,
			OutputCost:   0,
			Capabilities: []string{"text", "code"},
			Provider:     p.Name(),
		},
		{
			ID:           "phi",
			Name:         "Phi",
			Description:  "Microsoft's Phi model",
			MaxTokens:    2048,
			InputCost:    0,
			OutputCost:   0,
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
func (p *LocalProvider) GetCapabilities() Capabilities {
	rateLimits := RateLimit{
		RequestsPerSecond: 1,  // Local LLMs are typically slower
		RequestsPerMinute: 60,
		RequestsPerHour:   3600,
		TokensPerMinute:   10000,
	}

	if p.config.RateLimit != nil {
		rateLimits.RequestsPerSecond = p.config.RateLimit.RequestsPerSecond
		rateLimits.RequestsPerMinute = p.config.RateLimit.RequestsPerMinute
		rateLimits.RequestsPerHour = p.config.RateLimit.RequestsPerHour
		rateLimits.TokensPerMinute = p.config.RateLimit.TokensPerMinute
	}

	return Capabilities{
		SupportsStreaming:     true,
		SupportsVision:        false, // Most local LLMs don't support vision yet
		SupportsFunctionCall:  false, // Limited function calling support
		SupportsEmbeddings:    false, // Embeddings usually require separate models
		MaxContextLength:      8192,  // Varies by model
		SupportedFormats:      []string{"text", "markdown"},
		RateLimits:           rateLimits,
	}
}