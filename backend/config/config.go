package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// LegacyProvider maintains backward compatibility with existing config
type LegacyProvider struct {
	Name     string            `yaml:"name" json:"name"`
	APIKey   string            `yaml:"api_key" json:"api_key"`
	BaseURL  string            `yaml:"base_url" json:"base_url"`
	Models   []string          `yaml:"models" json:"models"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
	Timeout  int               `yaml:"timeout" json:"timeout"`
}

// ProviderType represents the type of provider
type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeLocal      ProviderType = "local"
	ProviderTypeMock       ProviderType = "mock"
)

// EnhancedProvider represents the new provider configuration structure
type EnhancedProvider struct {
	Name         string                 `yaml:"name" json:"name"`
	Type         ProviderType           `yaml:"type" json:"type"`
	Enabled      bool                   `yaml:"enabled" json:"enabled"`
	Version      string                 `yaml:"version" json:"version"`
	Priority     int                    `yaml:"priority" json:"priority"`
	APIKey       string                 `yaml:"api_key" json:"-"` // Hidden in JSON
	BaseURL      string                 `yaml:"base_url" json:"base_url"`
	Models       []string               `yaml:"models" json:"models"`
	Headers      map[string]string      `yaml:"headers" json:"headers"`
	Timeout      time.Duration          `yaml:"timeout" json:"timeout"`
	MaxRetries   int                    `yaml:"max_retries" json:"max_retries"`
	RetryDelay   time.Duration          `yaml:"retry_delay" json:"retry_delay"`
	Tags         []string               `yaml:"tags" json:"tags"`
	Dependencies []string               `yaml:"dependencies" json:"dependencies"`
	
	// Health check configuration
	HealthCheck *ProviderHealthConfig `yaml:"health_check,omitempty" json:"health_check,omitempty"`
	
	// Rate limiting configuration
	RateLimit *ProviderRateLimit `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	
	// Cost configuration
	Pricing *ProviderPricing `yaml:"pricing,omitempty" json:"pricing,omitempty"`
	
	// Provider-specific settings
	Settings map[string]interface{} `yaml:"settings,omitempty" json:"settings,omitempty"`
}

// ProviderHealthConfig contains health check configuration
type ProviderHealthConfig struct {
	Interval       time.Duration `yaml:"interval" json:"interval"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	MaxFailures    int           `yaml:"max_failures" json:"max_failures"`
	CustomEndpoint string        `yaml:"custom_endpoint,omitempty" json:"custom_endpoint,omitempty"`
	TestMessage    string        `yaml:"test_message,omitempty" json:"test_message,omitempty"`
}

// ProviderRateLimit contains rate limiting configuration
type ProviderRateLimit struct {
	RequestsPerSecond int `yaml:"requests_per_second" json:"requests_per_second"`
	RequestsPerMinute int `yaml:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour   int `yaml:"requests_per_hour" json:"requests_per_hour"`
	TokensPerMinute   int `yaml:"tokens_per_minute" json:"tokens_per_minute"`
	BurstSize         int `yaml:"burst_size" json:"burst_size"`
}

// ProviderPricing contains pricing information
type ProviderPricing struct {
	InputTokenCost  float64 `yaml:"input_token_cost" json:"input_token_cost"`
	OutputTokenCost float64 `yaml:"output_token_cost" json:"output_token_cost"`
	RequestCost     float64 `yaml:"request_cost" json:"request_cost"`
	Currency        string  `yaml:"currency" json:"currency"`
}

type Config struct {
	Server struct {
		Port      int    `yaml:"port" json:"port"`
		Host      string `yaml:"host" json:"host"`
		TargetURL string `yaml:"target_url" json:"target_url"` // Legacy support
	} `yaml:"server" json:"server"`
	
	// Legacy provider configuration (maintained for backward compatibility)
	Providers map[string]LegacyProvider `yaml:"providers" json:"providers"`
	
	// Enhanced provider configuration (new system)
	EnhancedProviders map[string]EnhancedProvider `yaml:"enhanced_providers,omitempty" json:"enhanced_providers,omitempty"`
	
	// Provider management configuration
	ProviderManager struct {
		AutoStart           bool          `yaml:"auto_start" json:"auto_start"`
		StartTimeout        time.Duration `yaml:"start_timeout" json:"start_timeout"`
		StopTimeout         time.Duration `yaml:"stop_timeout" json:"stop_timeout"`
		HealthCheckEnabled  bool          `yaml:"health_check_enabled" json:"health_check_enabled"`
		HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	} `yaml:"provider_manager" json:"provider_manager"`
	
	Routing struct {
		DefaultProvider string            `yaml:"default_provider" json:"default_provider"`
		ModelRouting    map[string]string `yaml:"model_routing" json:"model_routing"`
		FallbackChain   []string          `yaml:"fallback_chain" json:"fallback_chain"`
	} `yaml:"routing" json:"routing"`
	
	Security struct {
		RateLimiting        RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
		Headers             SecurityHeaders          `yaml:"headers" json:"headers"`
		Validation          ValidationConfig         `yaml:"validation" json:"validation"`
		CORS                CORSConfig               `yaml:"cors" json:"cors"`
		InputValidation     InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
		SecurityMonitoring  SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
		IPProtection        IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
		DDoSProtection      DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
		PromptInjection struct {
			Enabled     bool     `yaml:"enabled" json:"enabled"`
			Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
			Patterns    []string `yaml:"patterns" json:"patterns"`
		} `yaml:"prompt_injection" json:"prompt_injection"`
	} `yaml:"security" json:"security"`
	
	Moderation struct {
		Enabled        bool     `yaml:"enabled" json:"enabled"`
		Layers         []ModerationLayer `yaml:"layers" json:"layers"`
		BlockedWords   []string `yaml:"blocked_words" json:"blocked_words"`
		Severity       string   `yaml:"severity" json:"severity"`
		// Advanced moderation engine configuration
		Advanced       AdvancedModerationConfig `yaml:"advanced" json:"advanced"`
		// Legacy fields for backward compatibility
		UseOpenAI      bool     `yaml:"use_openai" json:"use_openai"`
		OpenAIAPIKey   string   `yaml:"openai_api_key" json:"-"`
	} `yaml:"moderation" json:"moderation"`
	
	Relevance struct {
		Enabled           bool    `yaml:"enabled" json:"enabled"`
		Threshold         float64 `yaml:"threshold" json:"threshold"`
		DomainEmbedding   string  `yaml:"domain_embedding" json:"domain_embedding"`
		Provider          string  `yaml:"provider" json:"provider"`
		Model             string  `yaml:"model" json:"model"`
		// Legacy field
		UseOpenAI         bool    `yaml:"use_openai" json:"use_openai"`
	} `yaml:"relevance" json:"relevance"`
	
	KillSwitch struct {
		Enabled     bool     `yaml:"enabled" json:"enabled"`
		BlockedUsers []string `yaml:"blocked_users" json:"blocked_users"`
		BlockedSessions []string `yaml:"blocked_sessions" json:"blocked_sessions"`
	} `yaml:"kill_switch" json:"kill_switch"`
	
	Logging struct {
		Enabled    bool   `yaml:"enabled" json:"enabled"`
		LogFile    string `yaml:"log_file" json:"log_file"`
		LogLevel   string `yaml:"log_level" json:"log_level"`
		UseSQLite  bool   `yaml:"use_sqlite" json:"use_sqlite"`
		SQLiteDB   string `yaml:"sqlite_db" json:"sqlite_db"`
	} `yaml:"logging" json:"logging"`
	
	Metrics struct {
		Enabled   bool          `yaml:"enabled" json:"enabled"`
		Storage   StorageConfig `yaml:"storage" json:"storage"`
		Supabase  SupabaseConfig `yaml:"supabase" json:"supabase"`
		Collection CollectionConfig `yaml:"collection" json:"collection"`
	} `yaml:"metrics" json:"metrics"`
	
	Database struct {
		Type                string        `yaml:"type" json:"type"`                                 // "sqlite" or "postgresql"
		ConnectionString    string        `yaml:"connection_string" json:"connection_string"`       // Full connection string (optional)
		Host               string        `yaml:"host" json:"host"`                                 // Database host
		Port               int           `yaml:"port" json:"port"`                                 // Database port
		Name               string        `yaml:"name" json:"name"`                                 // Database name
		Username           string        `yaml:"username" json:"username"`                         // Database username
		Password           string        `yaml:"password" json:"password"`                         // Database password
		SSLMode            string        `yaml:"ssl_mode" json:"ssl_mode"`                         // SSL mode for PostgreSQL
		MaxConnections     int           `yaml:"max_connections" json:"max_connections"`           // Maximum number of connections
		MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections"` // Maximum idle connections
		ConnectionLifetime time.Duration `yaml:"connection_lifetime" json:"connection_lifetime"`   // Connection lifetime
		
		// SQLite specific options
		SQLiteFile     string `yaml:"sqlite_file" json:"sqlite_file"`         // SQLite database file path
		EnableWAL      bool   `yaml:"enable_wal" json:"enable_wal"`           // Enable WAL mode for SQLite
		EnableForeignKeys bool `yaml:"enable_foreign_keys" json:"enable_foreign_keys"` // Enable foreign key constraints
		
		// Migration settings
		Migrations struct {
			Enabled   bool   `yaml:"enabled" json:"enabled"`     // Enable automatic migrations
			Directory string `yaml:"directory" json:"directory"` // Directory containing migration files
			Table     string `yaml:"table" json:"table"`         // Migration tracking table name
		} `yaml:"migrations" json:"migrations"`
		
		// Retention policies
		Retention struct {
			Requests string `yaml:"requests" json:"requests"` // Request data retention (e.g., "30d")
			Metrics  string `yaml:"metrics" json:"metrics"`   // Metrics data retention (e.g., "7d")
			Logs     string `yaml:"logs" json:"logs"`         // Log data retention (e.g., "90d")
		} `yaml:"retention" json:"retention"`
	} `yaml:"database" json:"database"`
	
	// Opik Integration Configuration
	Opik struct {
		Enabled       bool          `yaml:"enabled" json:"enabled"`
		APIKey        string        `yaml:"api_key" json:"-"` // Hidden in JSON
		ProjectName   string        `yaml:"project_name" json:"project_name"`
		BaseURL       string        `yaml:"base_url" json:"base_url"`
		BatchSize     int           `yaml:"batch_size" json:"batch_size"`
		FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval"`
		
		// Tracing configuration
		Tracing struct {
			Enabled         bool    `yaml:"enabled" json:"enabled"`
			SampleRate      float64 `yaml:"sample_rate" json:"sample_rate"`
			TraceModeration bool    `yaml:"trace_moderation" json:"trace_moderation"`
			TraceProviders  bool    `yaml:"trace_providers" json:"trace_providers"`
			TraceSecurity   bool    `yaml:"trace_security" json:"trace_security"`
		} `yaml:"tracing" json:"tracing"`
		
		// Evaluation configuration
		Evaluations struct {
			Enabled    bool          `yaml:"enabled" json:"enabled"`
			RunAsync   bool          `yaml:"run_async" json:"run_async"`
			Timeout    time.Duration `yaml:"timeout" json:"timeout"`
			Evaluators []string      `yaml:"evaluators" json:"evaluators"`
		} `yaml:"evaluations" json:"evaluations"`
	} `yaml:"opik" json:"opik"`
}

type ModerationLayer struct {
	Type       string            `yaml:"type" json:"type"` // "regex", "llm", "custom"
	Provider   string            `yaml:"provider" json:"provider"`
	Model      string            `yaml:"model" json:"model"`
	Prompt     string            `yaml:"prompt" json:"prompt"`
	Threshold  float64           `yaml:"threshold" json:"threshold"`
	Settings   map[string]interface{} `yaml:"settings" json:"settings"`
}

// Advanced moderation configuration types
type AdvancedModerationConfig struct {
	Enabled    bool                   `yaml:"enabled" json:"enabled"`
	Layers     []AdvancedLayerConfig  `yaml:"layers" json:"layers"`
	Thresholds ThresholdConfig        `yaml:"thresholds" json:"thresholds"`
	Actions    ActionConfig           `yaml:"actions" json:"actions"`
	Cache      CacheConfig            `yaml:"cache" json:"cache"`
	Analytics  AnalyticsConfig        `yaml:"analytics" json:"analytics"`
}

type AdvancedLayerConfig struct {
	Name      string                 `yaml:"name" json:"name"`
	Enabled   bool                   `yaml:"enabled" json:"enabled"`
	Weight    float64                `yaml:"weight" json:"weight"`
	Threshold float64                `yaml:"threshold" json:"threshold"`
	Options   map[string]interface{} `yaml:"options" json:"options"`
}

type ThresholdConfig struct {
	Low      float64 `yaml:"low" json:"low"`
	Medium   float64 `yaml:"medium" json:"medium"`
	High     float64 `yaml:"high" json:"high"`
	Critical float64 `yaml:"critical" json:"critical"`
}

type ActionConfig struct {
	Low      string `yaml:"low" json:"low"`
	Medium   string `yaml:"medium" json:"medium"`
	High     string `yaml:"high" json:"high"`
	Critical string `yaml:"critical" json:"critical"`
}

type CacheConfig struct {
	Enabled    bool `yaml:"enabled" json:"enabled"`
	TTLMinutes int  `yaml:"ttl_minutes" json:"ttl_minutes"`
	MaxEntries int  `yaml:"max_entries" json:"max_entries"`
}

type AnalyticsConfig struct {
	Enabled           bool `yaml:"enabled" json:"enabled"`
	CollectDetails    bool `yaml:"collect_details" json:"collect_details"`
	RetentionDays     int  `yaml:"retention_days" json:"retention_days"`
	EnablePerformance bool `yaml:"enable_performance" json:"enable_performance"`
}

// Metrics configuration types
type StorageConfig struct {
	Type                  string `yaml:"type" json:"type"`
	RetentionHours        int    `yaml:"retention_hours" json:"retention_hours"`
	CleanupIntervalMinutes int   `yaml:"cleanup_interval_minutes" json:"cleanup_interval_minutes"`
	MaxMemoryMB           int    `yaml:"max_memory_mb" json:"max_memory_mb"`
}

type SupabaseConfig struct {
	URL   string `yaml:"url" json:"url"`
	Key   string `yaml:"key" json:"key"`
	Table string `yaml:"table" json:"table"`
}

type CollectionConfig struct {
	HTTPRequests     bool `yaml:"http_requests" json:"http_requests"`
	ModerationEvents bool `yaml:"moderation_events" json:"moderation_events"`
	SystemHealth     bool `yaml:"system_health" json:"system_health"`
	PIIDetection     bool `yaml:"pii_detection" json:"pii_detection"`
}

// Rate Limiting Configuration Types (Cycle 6A)
type RateLimitingConfig struct {
	Enabled   bool                        `yaml:"enabled" json:"enabled"`
	Storage   RateLimitStorageConfig      `yaml:"storage" json:"storage"`
	Global    RateLimitRule               `yaml:"global" json:"global"`
	PerIP     RateLimitRule               `yaml:"per_ip" json:"per_ip"`
	PerUser   RateLimitRule               `yaml:"per_user" json:"per_user"`
	Endpoints map[string]RateLimitRule    `yaml:"endpoints" json:"endpoints"`
}

type RateLimitStorageConfig struct {
	Type     string `yaml:"type" json:"type"`         // "memory" or "redis"
	RedisURL string `yaml:"redis_url" json:"redis_url"` // Redis connection URL
	Prefix   string `yaml:"prefix" json:"prefix"`     // Key prefix for Redis
}

type RateLimitRule struct {
	RequestsPerSecond int           `yaml:"requests_per_second" json:"requests_per_second"`
	RequestsPerMinute int           `yaml:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour   int           `yaml:"requests_per_hour" json:"requests_per_hour"`
	Burst             int           `yaml:"burst" json:"burst"`
	WindowSize        time.Duration `yaml:"window_size" json:"window_size"`
	Algorithm         string        `yaml:"algorithm" json:"algorithm"` // "token_bucket" or "sliding_window"
}

// Security Headers Configuration
type SecurityHeaders struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`
	HSTSMaxAge   int    `yaml:"hsts_max_age" json:"hsts_max_age"`
	CSPPolicy    string `yaml:"csp_policy" json:"csp_policy"`
}

// Request Validation Configuration
type ValidationConfig struct {
	MaxRequestSize string        `yaml:"max_request_size" json:"max_request_size"`
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`
}

// CORS Configuration
type CORSConfig struct {
	Enabled         bool     `yaml:"enabled" json:"enabled"`
	AllowedOrigins  []string `yaml:"allowed_origins" json:"allowed_origins"`
	AllowedMethods  []string `yaml:"allowed_methods" json:"allowed_methods"`
	AllowedHeaders  []string `yaml:"allowed_headers" json:"allowed_headers"`
	ExposedHeaders  []string `yaml:"exposed_headers" json:"exposed_headers"`
	AllowCredentials bool    `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge          int      `yaml:"max_age" json:"max_age"`
}

// Input Validation Configuration
type InputValidationConfig struct {
	Enabled               bool `yaml:"enabled" json:"enabled"`
	MaxParameterLength    int  `yaml:"max_parameter_length" json:"max_parameter_length"`
	MaxValueLength        int  `yaml:"max_value_length" json:"max_value_length"`
	MaxJSONDepth          int  `yaml:"max_json_depth" json:"max_json_depth"`
	MaxJSONKeys           int  `yaml:"max_json_keys" json:"max_json_keys"`
	SanitizeHTML          bool `yaml:"sanitize_html" json:"sanitize_html"`
	DetectXSS             bool `yaml:"detect_xss" json:"detect_xss"`
	DetectSQLInjection    bool `yaml:"detect_sql_injection" json:"detect_sql_injection"`
	DetectCommandInjection bool `yaml:"detect_command_injection" json:"detect_command_injection"`
	DetectPathTraversal   bool `yaml:"detect_path_traversal" json:"detect_path_traversal"`
}

// Security Monitoring Configuration
type SecurityMonitoringConfig struct {
	Enabled                    bool `yaml:"enabled" json:"enabled"`
	LogSecurityEvents          bool `yaml:"log_security_events" json:"log_security_events"`
	AlertOnAttacks             bool `yaml:"alert_on_attacks" json:"alert_on_attacks"`
	BlockSuspiciousIPs         bool `yaml:"block_suspicious_ips" json:"block_suspicious_ips"`
	SuspiciousRequestThreshold int  `yaml:"suspicious_request_threshold" json:"suspicious_request_threshold"`
	BlockDurationMinutes       int  `yaml:"block_duration_minutes" json:"block_duration_minutes"`
}

// IP Protection Configuration
type IPProtectionConfig struct {
	Enabled                 bool          `yaml:"enabled" json:"enabled"`
	EnableGeoblocking       bool          `yaml:"enable_geoblocking" json:"enable_geoblocking"`
	EnableReputationCheck   bool          `yaml:"enable_reputation_check" json:"enable_reputation_check"`
	BlockedCountries        []string      `yaml:"blocked_countries" json:"blocked_countries"`
	AllowedCountries        []string      `yaml:"allowed_countries" json:"allowed_countries"`
	BlockedIPs              []string      `yaml:"blocked_ips" json:"blocked_ips"`
	AllowedIPs              []string      `yaml:"allowed_ips" json:"allowed_ips"`
	SuspiciousThreshold     int           `yaml:"suspicious_threshold" json:"suspicious_threshold"`
	AutoBlockDuration       time.Duration `yaml:"auto_block_duration" json:"auto_block_duration"`
	ReputationThreshold     float64       `yaml:"reputation_threshold" json:"reputation_threshold"`
	BlockMaliciousIPs       bool          `yaml:"block_malicious_ips" json:"block_malicious_ips"`
	BlockOnGeoError         bool          `yaml:"block_on_geo_error" json:"block_on_geo_error"`
	BlockOnReputationError  bool          `yaml:"block_on_reputation_error" json:"block_on_reputation_error"`
}

// DDoS Protection Configuration
type DDoSProtectionConfig struct {
	Enabled                 bool          `yaml:"enabled" json:"enabled"`
	SpikeThreshold          int           `yaml:"spike_threshold" json:"spike_threshold"`
	SpikeWindow             time.Duration `yaml:"spike_window" json:"spike_window"`
	CircuitBreakerThreshold int           `yaml:"circuit_breaker_threshold" json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout"`
	CircuitBreakerRequests  int           `yaml:"circuit_breaker_requests" json:"circuit_breaker_requests"`
	EnableBlocking          bool          `yaml:"enable_blocking" json:"enable_blocking"`
	EnableThrottling        bool          `yaml:"enable_throttling" json:"enable_throttling"`
	EnableDegradation       bool          `yaml:"enable_degradation" json:"enable_degradation"`
	EnableMetrics           bool          `yaml:"enable_metrics" json:"enable_metrics"`
	DegradationThreshold    int           `yaml:"degradation_threshold" json:"degradation_threshold"`
	ThrottleBaseDelay       time.Duration `yaml:"throttle_base_delay" json:"throttle_base_delay"`
	ThrottleMaxDelay        time.Duration `yaml:"throttle_max_delay" json:"throttle_max_delay"`
	ThrottleScaleFactor     float64       `yaml:"throttle_scale_factor" json:"throttle_scale_factor"`
	AlertCooldown           time.Duration `yaml:"alert_cooldown" json:"alert_cooldown"`
}

var AppConfig *Config

func LoadConfig(configPath string) error {
	AppConfig = &Config{}
	
	// Set defaults
	setDefaults()
	
	// Load from YAML file if exists
	if configPath != "" {
		if err := loadFromYAML(configPath); err != nil {
			return fmt.Errorf("failed to load config from YAML: %w", err)
		}
	}
	
	// Override with environment variables
	loadFromEnv()
	
	// Validate config
	if err := validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	
	return nil
}

func setDefaults() {
	AppConfig.Server.Port = 8080
	AppConfig.Server.Host = "localhost"
	AppConfig.Server.TargetURL = "http://localhost:8081" // Legacy fallback
	
	// Legacy providers (backward compatibility)
	AppConfig.Providers = make(map[string]LegacyProvider)
	AppConfig.Providers["openai"] = LegacyProvider{
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com/v1",
		Models:  []string{"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 30,
	}
	AppConfig.Providers["anthropic"] = LegacyProvider{
		Name:    "Anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		Models:  []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		Headers: map[string]string{
			"Content-Type":     "application/json",
			"anthropic-version": "2023-06-01",
		},
		Timeout: 30,
	}
	AppConfig.Providers["local"] = LegacyProvider{
		Name:    "Local",
		BaseURL: "http://localhost:11434/v1",
		Models:  []string{"llama2", "mistral", "codellama"},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 60,
	}
	
	// Enhanced providers (new system)
	AppConfig.EnhancedProviders = make(map[string]EnhancedProvider)
	AppConfig.EnhancedProviders["openai"] = EnhancedProvider{
		Name:       "openai",
		Type:       ProviderTypeOpenAI,
		Enabled:    true,
		Priority:   1,
		BaseURL:    "https://api.openai.com/v1",
		Models:     []string{"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"},
		Headers:    map[string]string{"Content-Type": "application/json"},
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		HealthCheck: &ProviderHealthConfig{
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			MaxFailures: 3,
		},
		RateLimit: &ProviderRateLimit{
			RequestsPerSecond: 10,
			RequestsPerMinute: 600,
			RequestsPerHour:   10000,
			TokensPerMinute:   90000,
		},
		Pricing: &ProviderPricing{
			InputTokenCost:  0.03,
			OutputTokenCost: 0.06,
			Currency:        "USD",
		},
	}
	AppConfig.EnhancedProviders["anthropic"] = EnhancedProvider{
		Name:       "anthropic",
		Type:       ProviderTypeAnthropic,
		Enabled:    true,
		Priority:   2,
		BaseURL:    "https://api.anthropic.com",
		Models:     []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		Headers:    map[string]string{"Content-Type": "application/json", "anthropic-version": "2023-06-01"},
		Timeout:    60 * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		HealthCheck: &ProviderHealthConfig{
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			MaxFailures: 3,
		},
		RateLimit: &ProviderRateLimit{
			RequestsPerSecond: 5,
			RequestsPerMinute: 300,
			RequestsPerHour:   4000,
			TokensPerMinute:   40000,
		},
		Pricing: &ProviderPricing{
			InputTokenCost:  0.015,
			OutputTokenCost: 0.075,
			Currency:        "USD",
		},
	}
	AppConfig.EnhancedProviders["local"] = EnhancedProvider{
		Name:       "local",
		Type:       ProviderTypeLocal,
		Enabled:    true,
		Priority:   3,
		BaseURL:    "http://localhost:11434",
		Models:     []string{"llama2", "mistral", "codellama"},
		Headers:    map[string]string{"Content-Type": "application/json"},
		Timeout:    120 * time.Second,
		MaxRetries: 2,
		RetryDelay: 2 * time.Second,
		HealthCheck: &ProviderHealthConfig{
			Interval:    30 * time.Second,
			Timeout:     15 * time.Second,
			MaxFailures: 5,
		},
		RateLimit: &ProviderRateLimit{
			RequestsPerSecond: 1,
			RequestsPerMinute: 60,
			RequestsPerHour:   3600,
			TokensPerMinute:   10000,
		},
		Pricing: &ProviderPricing{
			InputTokenCost:  0,
			OutputTokenCost: 0,
			Currency:        "USD",
		},
	}
	
	// Provider manager defaults
	AppConfig.ProviderManager.AutoStart = true
	AppConfig.ProviderManager.StartTimeout = 30 * time.Second
	AppConfig.ProviderManager.StopTimeout = 10 * time.Second
	AppConfig.ProviderManager.HealthCheckEnabled = true
	AppConfig.ProviderManager.HealthCheckInterval = 30 * time.Second
	
	// Routing defaults
	AppConfig.Routing.DefaultProvider = "local"
	AppConfig.Routing.ModelRouting = map[string]string{
		"gpt-4":     "openai",
		"gpt-3.5":   "openai",
		"claude-3":  "anthropic",
		"llama2":    "local",
		"mistral":   "local",
	}
	AppConfig.Routing.FallbackChain = []string{"openai", "anthropic", "local"}
	
	// Security defaults
	AppConfig.Security.RateLimiting = RateLimitingConfig{
		Enabled: true,
		Storage: RateLimitStorageConfig{
			Type:   "memory",
			Prefix: "qt1_rate_limit:",
		},
		Global: RateLimitRule{
			RequestsPerSecond: 2000,
			Burst:             4000,
			WindowSize:        time.Minute,
			Algorithm:         "token_bucket",
		},
		PerIP: RateLimitRule{
			RequestsPerMinute: 300,
			RequestsPerHour:   9000,
			Burst:             50,
			WindowSize:        time.Minute,
			Algorithm:         "sliding_window",
		},
		PerUser: RateLimitRule{
			RequestsPerMinute: 100,
			RequestsPerHour:   3000,
			Burst:             20,
			WindowSize:        time.Minute,
			Algorithm:         "sliding_window",
		},
		Endpoints: map[string]RateLimitRule{
			"/chat": {
				RequestsPerMinute: 120,
				RequestsPerHour:   3600,
				Burst:             20,
				WindowSize:        time.Minute,
				Algorithm:         "sliding_window",
			},
			"/api/config": {
				RequestsPerMinute: 10,
				RequestsPerHour:   300,
				Burst:             2,
				WindowSize:        time.Minute,
				Algorithm:         "token_bucket",
			},
		},
	}
	
	AppConfig.Security.Headers = SecurityHeaders{
		Enabled:    true,
		HSTSMaxAge: 31536000,
		CSPPolicy:  "default-src 'self'; script-src 'self' 'unsafe-inline'",
	}
	
	AppConfig.Security.Validation = ValidationConfig{
		MaxRequestSize: "10MB",
		RequestTimeout: 30 * time.Second,
	}
	
	AppConfig.Security.PromptInjection.Enabled = true
	AppConfig.Security.PromptInjection.Sensitivity = "high"
	AppConfig.Security.PromptInjection.Patterns = []string{
		"ignore previous instructions",
		"forget everything",
		"new instructions:",
		"you are now",
		"developer mode",
		"jailbreak",
		"DAN mode",
		"act as",
		"pretend you are",
		"roleplay as",
		"hypothetical scenario",
		"in a fictional world",
		"sudo mode",
		"admin mode",
		"override",
		"disregard",
	}
	
	// Enhanced moderation with layers
	AppConfig.Moderation.Enabled = true
	AppConfig.Moderation.Severity = "medium"
	AppConfig.Moderation.BlockedWords = []string{"violence", "hate", "explicit", "harmful", "suicide", "self-harm", "terrorist", "bomb", "weapon"}
	AppConfig.Moderation.Layers = []ModerationLayer{
		{
			Type:      "regex",
			Threshold: 1.0,
		},
		{
			Type:      "llm",
			Provider:  "openai",
			Model:     "gpt-3.5-turbo",
			Prompt:    "Analyze this message for harmful content. Respond with 'SAFE' or 'UNSAFE' followed by a brief reason.",
			Threshold: 0.8,
		},
	}
	
	// Legacy compatibility
	AppConfig.Moderation.UseOpenAI = false
	
	AppConfig.Relevance.Enabled = false
	AppConfig.Relevance.Threshold = 0.7
	AppConfig.Relevance.Provider = "openai"
	AppConfig.Relevance.Model = "text-embedding-3-small"
	AppConfig.Relevance.UseOpenAI = true // Legacy
	
	AppConfig.KillSwitch.Enabled = true
	
	AppConfig.Logging.Enabled = true
	AppConfig.Logging.LogFile = "logs/qt1.log"
	AppConfig.Logging.LogLevel = "info"
	AppConfig.Logging.UseSQLite = false
	AppConfig.Logging.SQLiteDB = "logs/qt1.db"
	
	// Metrics defaults
	AppConfig.Metrics.Enabled = true
	AppConfig.Metrics.Storage = StorageConfig{
		Type:                   "memory",
		RetentionHours:         24,
		CleanupIntervalMinutes: 60,
		MaxMemoryMB:           100,
	}
	AppConfig.Metrics.Supabase = SupabaseConfig{
		URL:   "",
		Key:   "",
		Table: "qt1_metrics",
	}
	AppConfig.Metrics.Collection = CollectionConfig{
		HTTPRequests:     true,
		ModerationEvents: true,
		SystemHealth:     true,
		PIIDetection:     true,
	}
	
	// Database defaults
	AppConfig.Database.Type = "sqlite"
	AppConfig.Database.Host = "localhost"
	AppConfig.Database.Port = 5432
	AppConfig.Database.Name = "qt1_middleware"
	AppConfig.Database.Username = ""
	AppConfig.Database.Password = ""
	AppConfig.Database.SSLMode = "disable"
	AppConfig.Database.MaxConnections = 25
	AppConfig.Database.MaxIdleConnections = 5
	AppConfig.Database.ConnectionLifetime = time.Hour
	AppConfig.Database.SQLiteFile = "storage/qt1.db"
	AppConfig.Database.EnableWAL = true
	AppConfig.Database.EnableForeignKeys = true
	AppConfig.Database.Migrations.Enabled = true
	AppConfig.Database.Migrations.Directory = "./database/migrations"
	AppConfig.Database.Migrations.Table = "schema_migrations"
	AppConfig.Database.Retention.Requests = "30d"
	AppConfig.Database.Retention.Metrics = "7d"
	AppConfig.Database.Retention.Logs = "90d"
	
	// Opik defaults
	AppConfig.Opik.Enabled = false
	AppConfig.Opik.ProjectName = "qt1-safety-cockpit"
	AppConfig.Opik.BaseURL = "https://api.opik.com"
	AppConfig.Opik.BatchSize = 100
	AppConfig.Opik.FlushInterval = 5 * time.Second
	AppConfig.Opik.Tracing.Enabled = true
	AppConfig.Opik.Tracing.SampleRate = 1.0
	AppConfig.Opik.Tracing.TraceModeration = true
	AppConfig.Opik.Tracing.TraceProviders = true
	AppConfig.Opik.Tracing.TraceSecurity = true
	AppConfig.Opik.Evaluations.Enabled = true
	AppConfig.Opik.Evaluations.RunAsync = true
	AppConfig.Opik.Evaluations.Timeout = 10 * time.Second
	AppConfig.Opik.Evaluations.Evaluators = []string{
		"regex_effectiveness",
		"llm_accuracy",
		"pii_coverage",
		"false_positive_rate",
		"response_time",
	}
}

func loadFromYAML(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	
	return yaml.Unmarshal(data, AppConfig)
}

func loadFromEnv() {
	if port := os.Getenv("QT1_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			AppConfig.Server.Port = p
		}
	}
	
	if host := os.Getenv("QT1_HOST"); host != "" {
		AppConfig.Server.Host = host
	}
	
	if targetURL := os.Getenv("QT1_TARGET_URL"); targetURL != "" {
		AppConfig.Server.TargetURL = targetURL
	}
	
	// Provider API keys - check both common env var names
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		openaiKey = os.Getenv("OPENAI_KEY")
	}
	if openaiKey != "" {
		if provider, ok := AppConfig.Providers["openai"]; ok {
			provider.APIKey = openaiKey
			AppConfig.Providers["openai"] = provider
		}
		AppConfig.Moderation.OpenAIAPIKey = openaiKey // Legacy compatibility
	}
	
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		if provider, ok := AppConfig.Providers["anthropic"]; ok {
			provider.APIKey = apiKey
			AppConfig.Providers["anthropic"] = provider
		}
	}
	
	if apiKey := os.Getenv("HUGGINGFACE_API_KEY"); apiKey != "" {
		if provider, ok := AppConfig.Providers["huggingface"]; ok {
			provider.APIKey = apiKey
			AppConfig.Providers["huggingface"] = provider
		}
	}
	
	// Routing configuration
	if defaultProvider := os.Getenv("QT1_DEFAULT_PROVIDER"); defaultProvider != "" {
		AppConfig.Routing.DefaultProvider = defaultProvider
	}
	
	// Security configuration
	if sensitivity := os.Getenv("QT1_PROMPT_INJECTION_SENSITIVITY"); sensitivity != "" {
		AppConfig.Security.PromptInjection.Sensitivity = sensitivity
	}
	
	if logFile := os.Getenv("QT1_LOG_FILE"); logFile != "" {
		AppConfig.Logging.LogFile = logFile
	}
	
	// Database configuration
	if dbType := os.Getenv("QT1_DB_TYPE"); dbType != "" {
		AppConfig.Database.Type = dbType
	}
	
	if dbHost := os.Getenv("QT1_DB_HOST"); dbHost != "" {
		AppConfig.Database.Host = dbHost
	}
	
	if dbPort := os.Getenv("QT1_DB_PORT"); dbPort != "" {
		if p, err := strconv.Atoi(dbPort); err == nil {
			AppConfig.Database.Port = p
		}
	}
	
	if dbName := os.Getenv("QT1_DB_NAME"); dbName != "" {
		AppConfig.Database.Name = dbName
	}
	
	if dbUser := os.Getenv("QT1_DB_USERNAME"); dbUser != "" {
		AppConfig.Database.Username = dbUser
	}
	
	if dbPass := os.Getenv("QT1_DB_PASSWORD"); dbPass != "" {
		AppConfig.Database.Password = dbPass
	}
	
	if sslMode := os.Getenv("QT1_DB_SSL_MODE"); sslMode != "" {
		AppConfig.Database.SSLMode = sslMode
	}
	
	if dbFile := os.Getenv("QT1_DB_SQLITE_FILE"); dbFile != "" {
		AppConfig.Database.SQLiteFile = dbFile
	}
	
	if connStr := os.Getenv("QT1_DB_CONNECTION_STRING"); connStr != "" {
		AppConfig.Database.ConnectionString = connStr
	}
	
	// Opik configuration
	if apiKey := os.Getenv("OPIK_API_KEY"); apiKey != "" {
		AppConfig.Opik.APIKey = apiKey
	}
	
	if projectName := os.Getenv("OPIK_PROJECT_NAME"); projectName != "" {
		AppConfig.Opik.ProjectName = projectName
	}
	
	if baseURL := os.Getenv("OPIK_BASE_URL"); baseURL != "" {
		AppConfig.Opik.BaseURL = baseURL
	}
	
	if enabled := os.Getenv("OPIK_ENABLED"); enabled != "" {
		AppConfig.Opik.Enabled = enabled == "true" || enabled == "1"
	}
}

func validate() error {
	if AppConfig.Server.Port <= 0 || AppConfig.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", AppConfig.Server.Port)
	}
	
	if AppConfig.Server.TargetURL == "" {
		return fmt.Errorf("target_url is required")
	}
	
	if AppConfig.Moderation.UseOpenAI && AppConfig.Moderation.OpenAIAPIKey == "" {
		return fmt.Errorf("OpenAI API key is required when moderation.use_openai is true")
	}
	
	if AppConfig.Relevance.Enabled && AppConfig.Relevance.UseOpenAI && AppConfig.Moderation.OpenAIAPIKey == "" {
		return fmt.Errorf("OpenAI API key is required when relevance is enabled with use_openai")
	}
	
	return nil
}

func SaveConfig(configPath string) error {
	data, err := yaml.Marshal(AppConfig)
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}