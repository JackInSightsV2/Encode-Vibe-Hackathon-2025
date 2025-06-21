package providers

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// MockProvider implements the Provider interface for testing purposes
type MockProvider struct {
	config        ProviderConfig
	metrics       ProviderMetrics
	status        ProviderStatus
	mutex         sync.RWMutex
	startTime     time.Time
	lastRequest   time.Time
	
	// Mock configuration
	simulateLatency  time.Duration
	errorRate        float64
	shouldFail       bool
	customResponse   string
	availableModels  []string
}

// MockConfig contains mock-specific configuration
type MockConfig struct {
	SimulateLatency time.Duration `yaml:"simulate_latency"`
	ErrorRate       float64       `yaml:"error_rate"`        // 0.0 to 1.0
	ShouldFail      bool          `yaml:"should_fail"`       // Always fail
	CustomResponse  string        `yaml:"custom_response"`   // Custom response text
	AvailableModels []string      `yaml:"available_models"`
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		status: ProviderStatus{
			State:   StateStopped,
			Healthy: false,
			Version: "1.0.0-mock",
		},
		metrics:         ProviderMetrics{},
		simulateLatency: 100 * time.Millisecond,
		errorRate:       0.0,
		shouldFail:      false,
		customResponse:  "This is a mock response from the test provider.",
		availableModels: []string{"mock-model", "test-model", "fake-gpt-4"},
	}
}

// NewMockProviderWithConfig creates a new mock provider with specific configuration
func NewMockProviderWithConfig(mockConfig MockConfig) *MockProvider {
	p := NewMockProvider()
	
	if mockConfig.SimulateLatency > 0 {
		p.simulateLatency = mockConfig.SimulateLatency
	}
	if mockConfig.ErrorRate >= 0 && mockConfig.ErrorRate <= 1 {
		p.errorRate = mockConfig.ErrorRate
	}
	p.shouldFail = mockConfig.ShouldFail
	if mockConfig.CustomResponse != "" {
		p.customResponse = mockConfig.CustomResponse
	}
	if len(mockConfig.AvailableModels) > 0 {
		p.availableModels = mockConfig.AvailableModels
	}
	
	return p
}

// Name returns the provider name
func (p *MockProvider) Name() string {
	return "mock"
}

// Type returns the provider type
func (p *MockProvider) Type() ProviderType {
	return ProviderTypeMock
}

// Version returns the provider version
func (p *MockProvider) Version() string {
	return "1.0.0-mock"
}

// Configure sets the provider configuration
func (p *MockProvider) Configure(config ProviderConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Parse mock-specific settings
	if config.Settings != nil {
		if latency, ok := config.Settings["simulate_latency"]; ok {
			if duration, err := time.ParseDuration(fmt.Sprintf("%v", latency)); err == nil {
				p.simulateLatency = duration
			}
		}
		
		if errorRate, ok := config.Settings["error_rate"]; ok {
			if rate, ok := errorRate.(float64); ok && rate >= 0 && rate <= 1 {
				p.errorRate = rate
			}
		}
		
		if shouldFail, ok := config.Settings["should_fail"]; ok {
			if fail, ok := shouldFail.(bool); ok {
				p.shouldFail = fail
			}
		}
		
		if customResponse, ok := config.Settings["custom_response"]; ok {
			if response, ok := customResponse.(string); ok && response != "" {
				p.customResponse = response
			}
		}
		
		if models, ok := config.Settings["available_models"]; ok {
			if modelList, ok := models.([]string); ok {
				p.availableModels = modelList
			}
		}
	}

	// Set default values if not provided
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	p.config = config
	return nil
}

// Validate validates the provider configuration
func (p *MockProvider) Validate() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.errorRate < 0 || p.errorRate > 1 {
		return fmt.Errorf("error rate must be between 0 and 1")
	}

	return nil
}

// Start starts the provider
func (p *MockProvider) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status.State == StateRunning {
		return nil // Already running
	}

	p.status.State = StateStarting
	p.startTime = time.Now()

	// Simulate startup time
	select {
	case <-ctx.Done():
		p.status.State = StateError
		p.status.Error = "startup cancelled"
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		// Quick startup
	}

	// Check if we should simulate startup failure
	if p.shouldFail {
		p.status.State = StateError
		p.status.Error = "mock provider configured to fail"
		return fmt.Errorf("mock provider configured to fail")
	}

	p.status.State = StateRunning
	p.status.Healthy = true
	p.status.StartTime = p.startTime
	p.status.Error = ""

	return nil
}

// Stop stops the provider
func (p *MockProvider) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status.State == StateStopped {
		return nil // Already stopped
	}

	p.status.State = StateStopping

	// Simulate shutdown time
	select {
	case <-ctx.Done():
		// Force stop on timeout
	case <-time.After(5 * time.Millisecond):
		// Quick shutdown
	}

	p.status.State = StateStopped
	p.status.Healthy = false

	return nil
}

// HealthCheck performs a health check
func (p *MockProvider) HealthCheck(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.shouldFail {
		return fmt.Errorf("mock provider health check failed")
	}

	// Simulate health check latency
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.simulateLatency / 10):
		// Quick health check
	}

	p.status.LastCheck = time.Now()
	return nil
}

// GetStatus returns the current provider status
func (p *MockProvider) GetStatus() ProviderStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	status := p.status
	if p.status.State == StateRunning {
		status.Uptime = time.Since(p.startTime)
	}

	return status
}

// GetMetrics returns the current provider metrics
func (p *MockProvider) GetMetrics() ProviderMetrics {
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

// SendRequest sends a chat request (mock implementation)
func (p *MockProvider) SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()

	p.mutex.Lock()
	p.lastRequest = startTime
	p.metrics.RequestCount++
	p.mutex.Unlock()

	// Simulate latency
	select {
	case <-ctx.Done():
		p.mutex.Lock()
		p.metrics.ErrorCount++
		p.mutex.Unlock()
		return nil, ctx.Err()
	case <-time.After(p.simulateLatency):
		// Continue
	}

	// Simulate random errors based on error rate
	if p.shouldFail || (p.errorRate > 0 && rand.Float64() < p.errorRate) {
		p.mutex.Lock()
		p.metrics.ErrorCount++
		p.mutex.Unlock()

		errorType := ErrorTypeServer
		if rand.Float64() < 0.3 {
			errorType = ErrorTypeRateLimit
			p.mutex.Lock()
			p.metrics.RateLimitHits++
			p.mutex.Unlock()
		}

		return nil, NewProviderError(p.Name(), errorType, "mock_error",
			"Mock provider error", true, fmt.Errorf("simulated error"))
	}

	// Generate mock response
	latency := time.Since(startTime)
	
	// Simulate token usage
	inputTokens := len(req.Messages) * 10    // Rough estimate
	outputTokens := len(p.customResponse) / 4 // Rough estimate
	totalTokens := inputTokens + outputTokens

	cost := 0.0 // Mock is free
	if p.config.Pricing != nil {
		cost = float64(inputTokens)*p.config.Pricing.InputTokenCost/1000 +
			   float64(outputTokens)*p.config.Pricing.OutputTokenCost/1000 +
			   p.config.Pricing.RequestCost
	}

	response := &ChatResponse{
		ID:       fmt.Sprintf("mock-%d", time.Now().Unix()),
		Provider: p.Name(),
		Model:    req.Model,
		Message: ChatMessage{
			Role:    "assistant",
			Content: p.customResponse,
		},
		FinishReason: "stop",
		Usage: TokenUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
		},
		Latency:   latency,
		Cost:      cost,
		CreatedAt: time.Now(),
	}

	// Update metrics
	p.mutex.Lock()
	p.metrics.SuccessCount++
	p.metrics.TotalLatency += latency
	p.metrics.TokensUsed += int64(totalTokens)
	p.metrics.InputTokens += int64(inputTokens)
	p.metrics.OutputTokens += int64(outputTokens)
	p.metrics.TotalCost += cost
	p.mutex.Unlock()

	return response, nil
}

// GetModels returns available models
func (p *MockProvider) GetModels() []Model {
	models := make([]Model, 0, len(p.availableModels))
	
	for i, modelID := range p.availableModels {
		model := Model{
			ID:          modelID,
			Name:        fmt.Sprintf("Mock Model %s", modelID),
			Description: fmt.Sprintf("Mock model for testing: %s", modelID),
			MaxTokens:   4096,
			InputCost:   0.001, // Mock pricing
			OutputCost:  0.002,
			Capabilities: []string{"text"},
			Provider:    p.Name(),
			Version:     "1.0.0-mock",
		}
		
		// Add some variety to capabilities
		if i%2 == 0 {
			model.Capabilities = append(model.Capabilities, "reasoning")
		}
		if i%3 == 0 {
			model.Capabilities = append(model.Capabilities, "code")
		}
		
		models = append(models, model)
	}

	// Filter by configured models if specified
	if len(p.config.Models) > 0 {
		filtered := make([]Model, 0)
		for _, model := range models {
			for _, configModel := range p.config.Models {
				if model.ID == configModel {
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
func (p *MockProvider) GetCapabilities() Capabilities {
	rateLimits := RateLimit{
		RequestsPerSecond: 100,
		RequestsPerMinute: 6000,
		RequestsPerHour:   100000,
		TokensPerMinute:   1000000,
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
		MaxContextLength:      8192,
		SupportedFormats:      []string{"text", "json", "markdown"},
		RateLimits:           rateLimits,
	}
}

// SetErrorRate dynamically sets the error rate for testing
func (p *MockProvider) SetErrorRate(rate float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if rate >= 0 && rate <= 1 {
		p.errorRate = rate
	}
}

// SetShouldFail dynamically sets whether the provider should always fail
func (p *MockProvider) SetShouldFail(shouldFail bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.shouldFail = shouldFail
}

// SetCustomResponse dynamically sets the custom response text
func (p *MockProvider) SetCustomResponse(response string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if response != "" {
		p.customResponse = response
	}
}

// SetSimulateLatency dynamically sets the simulated latency
func (p *MockProvider) SetSimulateLatency(latency time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if latency >= 0 {
		p.simulateLatency = latency
	}
}

// Reset resets the provider metrics for testing
func (p *MockProvider) Reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.metrics = ProviderMetrics{}
	p.lastRequest = time.Time{}
}

// GetErrorRate returns the current error rate
func (p *MockProvider) GetErrorRate() float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.errorRate
}

// GetShouldFail returns whether the provider should always fail
func (p *MockProvider) GetShouldFail() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.shouldFail
}

// GetCustomResponse returns the custom response text
func (p *MockProvider) GetCustomResponse() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.customResponse
}

// GetSimulateLatency returns the simulated latency
func (p *MockProvider) GetSimulateLatency() time.Duration {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.simulateLatency
}