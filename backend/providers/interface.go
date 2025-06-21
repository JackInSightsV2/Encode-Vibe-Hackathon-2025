package providers

import (
	"context"
	"time"
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeLocal      ProviderType = "local"
	ProviderTypeMock       ProviderType = "mock"
	ProviderTypeHuggingFace ProviderType = "huggingface"
)

// ProviderState represents the current state of a provider
type ProviderState string

const (
	StateStarting ProviderState = "starting"
	StateRunning  ProviderState = "running"
	StateStopping ProviderState = "stopping"
	StateStopped  ProviderState = "stopped"
	StateError    ProviderState = "error"
)

// Provider defines the interface that all AI providers must implement
type Provider interface {
	// Basic info
	Name() string
	Type() ProviderType
	Version() string
	
	// Health and status
	HealthCheck(ctx context.Context) error
	GetStatus() ProviderStatus
	GetMetrics() ProviderMetrics
	
	// Core functionality
	SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	GetModels() []Model
	GetCapabilities() Capabilities
	
	// Configuration
	Configure(config ProviderConfig) error
	Validate() error
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ProviderStatus contains current status information about a provider
type ProviderStatus struct {
	State       ProviderState `json:"state"`
	Healthy     bool          `json:"healthy"`
	LastCheck   time.Time     `json:"last_check"`
	Error       string        `json:"error,omitempty"`
	Uptime      time.Duration `json:"uptime"`
	Version     string        `json:"version"`
	StartTime   time.Time     `json:"start_time"`
}

// ProviderMetrics contains performance and usage metrics for a provider
type ProviderMetrics struct {
	RequestCount       int64         `json:"request_count"`
	ErrorCount         int64         `json:"error_count"`
	SuccessCount       int64         `json:"success_count"`
	AverageLatency     time.Duration `json:"average_latency"`
	TotalLatency       time.Duration `json:"total_latency"`
	TokensUsed         int64         `json:"tokens_used"`
	InputTokens        int64         `json:"input_tokens"`
	OutputTokens       int64         `json:"output_tokens"`
	TotalCost          float64       `json:"total_cost"`
	RateLimitHits      int64         `json:"rate_limit_hits"`
	LastRequestTime    time.Time     `json:"last_request_time"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	ErrorRate          float64       `json:"error_rate"`
	SuccessRate        float64       `json:"success_rate"`
}

// ChatRequest represents a request to an AI provider
type ChatRequest struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	MessageID   string                 `json:"message_id,omitempty"`
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"`
}

// ChatResponse represents a response from an AI provider
type ChatResponse struct {
	ID            string        `json:"id"`
	Provider      string        `json:"provider"`
	Model         string        `json:"model"`
	Message       ChatMessage   `json:"message"`
	FinishReason  string        `json:"finish_reason"`
	Usage         TokenUsage    `json:"usage"`
	Latency       time.Duration `json:"latency"`
	Cost          float64       `json:"cost"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Model represents an available AI model
type Model struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	MaxTokens    int      `json:"max_tokens"`
	InputCost    float64  `json:"input_cost"`     // Cost per input token
	OutputCost   float64  `json:"output_cost"`    // Cost per output token
	Capabilities []string `json:"capabilities"`   // e.g., ["text", "vision", "function_calling"]
	Provider     string   `json:"provider"`
	Version      string   `json:"version,omitempty"`
}

// Capabilities represents what a provider can do
type Capabilities struct {
	SupportsStreaming     bool     `json:"supports_streaming"`
	SupportsVision        bool     `json:"supports_vision"`
	SupportsFunctionCall  bool     `json:"supports_function_call"`
	SupportsEmbeddings    bool     `json:"supports_embeddings"`
	MaxContextLength      int      `json:"max_context_length"`
	SupportedFormats      []string `json:"supported_formats"`     // e.g., ["text", "json", "markdown"]
	RateLimits           RateLimit `json:"rate_limits"`
}

// RateLimit contains rate limiting information
type RateLimit struct {
	RequestsPerSecond int `json:"requests_per_second"`
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	TokensPerMinute   int `json:"tokens_per_minute"`
}

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	Name         string                 `yaml:"name" json:"name"`
	Type         ProviderType          `yaml:"type" json:"type"`
	Enabled      bool                  `yaml:"enabled" json:"enabled"`
	Version      string                `yaml:"version" json:"version"`
	Priority     int                   `yaml:"priority" json:"priority"`
	APIKey       string                `yaml:"api_key" json:"-"` // Hidden in JSON
	BaseURL      string                `yaml:"base_url" json:"base_url"`
	Models       []string              `yaml:"models" json:"models"`
	Headers      map[string]string     `yaml:"headers" json:"headers"`
	Timeout      time.Duration         `yaml:"timeout" json:"timeout"`
	MaxRetries   int                   `yaml:"max_retries" json:"max_retries"`
	RetryDelay   time.Duration         `yaml:"retry_delay" json:"retry_delay"`
	Tags         []string              `yaml:"tags" json:"tags"`
	Dependencies []string              `yaml:"dependencies" json:"dependencies"`
	
	// Health check configuration
	HealthCheck *HealthCheckConfig `yaml:"health_check,omitempty" json:"health_check,omitempty"`
	
	// Rate limiting configuration
	RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	
	// Cost configuration
	Pricing *PricingConfig `yaml:"pricing,omitempty" json:"pricing,omitempty"`
	
	// Provider-specific settings
	Settings map[string]interface{} `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Interval       time.Duration `yaml:"interval" json:"interval"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	MaxFailures    int           `yaml:"max_failures" json:"max_failures"`
	CustomEndpoint string        `yaml:"custom_endpoint,omitempty" json:"custom_endpoint,omitempty"`
	TestMessage    string        `yaml:"test_message,omitempty" json:"test_message,omitempty"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second" json:"requests_per_second"`
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour   int `yaml:"requests_per_hour" json:"requests_per_hour"`
	TokensPerMinute   int `yaml:"tokens_per_minute" json:"tokens_per_minute"`
	BurstSize         int `yaml:"burst_size" json:"burst_size"`
}

// PricingConfig contains pricing information
type PricingConfig struct {
	InputTokenCost  float64 `yaml:"input_token_cost" json:"input_token_cost"`
	OutputTokenCost float64 `yaml:"output_token_cost" json:"output_token_cost"`
	RequestCost     float64 `yaml:"request_cost" json:"request_cost"`
	Currency        string  `yaml:"currency" json:"currency"`
}

// ProviderError represents an error from a provider
type ProviderError struct {
	Provider string `json:"provider"`
	Type     string `json:"type"`     // "rate_limit", "auth", "network", "server", "unknown"
	Code     string `json:"code"`     // Provider-specific error code
	Message  string `json:"message"`
	Retryable bool  `json:"retryable"`
	Err      error  `json:"-"`
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// Error types
const (
	ErrorTypeRateLimit = "rate_limit"
	ErrorTypeAuth      = "auth"
	ErrorTypeNetwork   = "network"
	ErrorTypeServer    = "server"
	ErrorTypeValidation = "validation"
	ErrorTypeUnknown   = "unknown"
)

// NewProviderError creates a new provider error
func NewProviderError(provider, errorType, code, message string, retryable bool, err error) *ProviderError {
	return &ProviderError{
		Provider:  provider,
		Type:      errorType,
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Err:       err,
	}
}