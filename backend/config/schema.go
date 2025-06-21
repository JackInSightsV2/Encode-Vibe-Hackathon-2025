package config

import (
	"encoding/json"
	"time"
)

// ConfigSchema represents the complete configuration schema
type ConfigSchema struct {
	Version     string                        `json:"version"`
	Description string                        `json:"description"`
	Sections    map[string]SectionSchema      `json:"sections"`
	Properties  map[string]PropertySchema     `json:"properties"`
	Required    []string                      `json:"required"`
	Rules       []ValidationRule              `json:"rules,omitempty"`
}

// SectionSchema represents a configuration section schema
type SectionSchema struct {
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Properties  map[string]PropertySchema `json:"properties"`
	Required    []string                  `json:"required"`
	Order       int                       `json:"order"`
	Icon        string                    `json:"icon,omitempty"`
	Advanced    bool                      `json:"advanced,omitempty"`
}

// PropertySchema defines the schema for a single configuration property
type PropertySchema struct {
	Type         string                 `json:"type"`                    // string, number, boolean, array, object, duration
	Title        string                 `json:"title"`                   // Display name
	Description  string                 `json:"description"`             // Help text
	Required     bool                   `json:"required"`                // Is this field required?
	Default      interface{}            `json:"default,omitempty"`       // Default value
	Example      interface{}            `json:"example,omitempty"`       // Example value
	Sensitive    bool                   `json:"sensitive,omitempty"`     // Is this sensitive data?
	Deprecated   bool                   `json:"deprecated,omitempty"`    // Is this field deprecated?
	MinValue     interface{}            `json:"min_value,omitempty"`     // Minimum value for numbers
	MaxValue     interface{}            `json:"max_value,omitempty"`     // Maximum value for numbers
	MinLength    int                    `json:"min_length,omitempty"`    // Minimum length for strings
	MaxLength    int                    `json:"max_length,omitempty"`    // Maximum length for strings
	Pattern      string                 `json:"pattern,omitempty"`       // Regex pattern for validation
	Enum         []interface{}          `json:"enum,omitempty"`          // Allowed values
	Items        *PropertySchema        `json:"items,omitempty"`         // Schema for array items
	Properties   map[string]PropertySchema `json:"properties,omitempty"` // Schema for object properties
	Format       string                 `json:"format,omitempty"`        // Format hint (email, url, date, etc.)
	DependsOn    []string               `json:"depends_on,omitempty"`    // Fields this depends on
	CustomValidator string              `json:"custom_validator,omitempty"` // Name of custom validation function
	UIComponent  string                 `json:"ui_component,omitempty"`  // UI component hint
	UIOptions    map[string]interface{} `json:"ui_options,omitempty"`    // UI-specific options
}

// ValidationRule represents a custom validation rule
type ValidationRule struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Fields      []string `json:"fields"`
	Type        string   `json:"type"` // "required_if", "exclusive", "at_least_one", "custom"
	Condition   string   `json:"condition,omitempty"`
	Message     string   `json:"message"`
}

// GetConfigSchema returns the complete configuration schema
func GetConfigSchema() *ConfigSchema {
	return &ConfigSchema{
		Version:     "1.0.0",
		Description: "QT1 Middleware Configuration Schema",
		Sections: map[string]SectionSchema{
			"server": {
				Title:       "Server Configuration",
				Description: "Basic server settings",
				Order:       1,
				Icon:        "server",
				Properties: map[string]PropertySchema{
					"port": {
						Type:        "number",
						Title:       "Port",
						Description: "Server port number",
						Required:    true,
						Default:     8080,
						MinValue:    1024,
						MaxValue:    65535,
						Example:     8080,
					},
					"host": {
						Type:        "string",
						Title:       "Host",
						Description: "Server host address",
						Required:    true,
						Default:     "localhost",
						Pattern:     `^[a-zA-Z0-9.-]+$`,
						Example:     "localhost",
					},
					"target_url": {
						Type:        "string",
						Title:       "Target URL",
						Description: "Legacy target URL for backward compatibility",
						Required:    false,
						Format:      "url",
						Example:     "http://localhost:8081",
						Deprecated:  true,
					},
				},
				Required: []string{"port", "host"},
			},
			"providers": {
				Title:       "Provider Configuration",
				Description: "AI provider settings",
				Order:       2,
				Icon:        "cloud",
				Properties: map[string]PropertySchema{
					"*": { // Wildcard for dynamic provider names
						Type:  "object",
						Title: "Provider",
						Properties: map[string]PropertySchema{
							"name": {
								Type:        "string",
								Title:       "Provider Name",
								Description: "Display name for the provider",
								Required:    true,
							},
							"type": {
								Type:        "string",
								Title:       "Provider Type",
								Description: "Type of provider",
								Required:    true,
								Enum:        []interface{}{"openai", "anthropic", "local", "mock"},
							},
							"enabled": {
								Type:        "boolean",
								Title:       "Enabled",
								Description: "Whether this provider is enabled",
								Default:     true,
							},
							"api_key": {
								Type:        "string",
								Title:       "API Key",
								Description: "API key for authentication",
								Sensitive:   true,
								UIComponent: "password",
							},
							"base_url": {
								Type:        "string",
								Title:       "Base URL",
								Description: "Base URL for the provider API",
								Required:    true,
								Format:      "url",
							},
							"models": {
								Type:        "array",
								Title:       "Supported Models",
								Description: "List of models supported by this provider",
								Items: &PropertySchema{
									Type: "string",
								},
							},
							"timeout": {
								Type:        "duration",
								Title:       "Timeout",
								Description: "Request timeout duration",
								Default:     "30s",
								Example:     "30s",
							},
							"max_retries": {
								Type:        "number",
								Title:       "Max Retries",
								Description: "Maximum number of retry attempts",
								Default:     3,
								MinValue:    0,
								MaxValue:    10,
							},
							"priority": {
								Type:        "number",
								Title:       "Priority",
								Description: "Provider priority for routing (lower is higher priority)",
								Default:     10,
								MinValue:    1,
								MaxValue:    100,
							},
						},
					},
				},
			},
			"security": {
				Title:       "Security Configuration",
				Description: "Security and protection settings",
				Order:       3,
				Icon:        "shield",
				Properties: map[string]PropertySchema{
					"rate_limiting": {
						Type:  "object",
						Title: "Rate Limiting",
						Properties: map[string]PropertySchema{
							"enabled": {
								Type:        "boolean",
								Title:       "Enable Rate Limiting",
								Description: "Enable rate limiting protection",
								Default:     true,
							},
							"storage": {
								Type:  "object",
								Title: "Storage Configuration",
								Properties: map[string]PropertySchema{
									"type": {
										Type:        "string",
										Title:       "Storage Type",
										Description: "Rate limit storage backend",
										Default:     "memory",
										Enum:        []interface{}{"memory", "redis"},
									},
									"redis_url": {
										Type:        "string",
										Title:       "Redis URL",
										Description: "Redis connection URL (required if type is redis)",
										Format:      "url",
										DependsOn:   []string{"type"},
									},
								},
							},
							"global": {
								Type:        "object",
								Title:       "Global Rate Limit",
								Description: "Global rate limiting rules",
								Properties:  getRateLimitRuleSchema(),
							},
							"per_ip": {
								Type:        "object",
								Title:       "Per-IP Rate Limit",
								Description: "Rate limiting rules per IP address",
								Properties:  getRateLimitRuleSchema(),
							},
							"per_user": {
								Type:        "object",
								Title:       "Per-User Rate Limit",
								Description: "Rate limiting rules per authenticated user",
								Properties:  getRateLimitRuleSchema(),
							},
						},
					},
					"headers": {
						Type:  "object",
						Title: "Security Headers",
						Properties: map[string]PropertySchema{
							"enabled": {
								Type:        "boolean",
								Title:       "Enable Security Headers",
								Default:     true,
							},
							"hsts_max_age": {
								Type:        "number",
								Title:       "HSTS Max Age",
								Description: "HTTP Strict Transport Security max age in seconds",
								Default:     31536000,
								MinValue:    0,
							},
							"csp_policy": {
								Type:        "string",
								Title:       "Content Security Policy",
								Description: "Content Security Policy header value",
								Default:     "default-src 'self'",
							},
						},
					},
					"cors": {
						Type:  "object",
						Title: "CORS Configuration",
						Properties: map[string]PropertySchema{
							"enabled": {
								Type:    "boolean",
								Title:   "Enable CORS",
								Default: true,
							},
							"allowed_origins": {
								Type:  "array",
								Title: "Allowed Origins",
								Items: &PropertySchema{
									Type:    "string",
									Pattern: `^https?://`,
								},
								Default: []interface{}{"*"},
							},
							"allowed_methods": {
								Type:  "array",
								Title: "Allowed Methods",
								Items: &PropertySchema{
									Type: "string",
									Enum: []interface{}{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
								},
								Default: []interface{}{"GET", "POST", "OPTIONS"},
							},
							"max_age": {
								Type:        "number",
								Title:       "Max Age",
								Description: "CORS preflight cache duration in seconds",
								Default:     3600,
								MinValue:    0,
							},
						},
					},
				},
			},
			"moderation": {
				Title:       "Content Moderation",
				Description: "Content moderation and filtering settings",
				Order:       4,
				Icon:        "filter",
				Properties: map[string]PropertySchema{
					"enabled": {
						Type:        "boolean",
						Title:       "Enable Moderation",
						Description: "Enable content moderation",
						Default:     true,
					},
					"severity": {
						Type:        "string",
						Title:       "Severity Level",
						Description: "Moderation severity level",
						Default:     "medium",
						Enum:        []interface{}{"low", "medium", "high", "strict"},
					},
					"blocked_words": {
						Type:        "array",
						Title:       "Blocked Words",
						Description: "List of words to block",
						Items: &PropertySchema{
							Type: "string",
						},
						Sensitive: true,
					},
					"layers": {
						Type:  "array",
						Title: "Moderation Layers",
						Items: &PropertySchema{
							Type: "object",
							Properties: map[string]PropertySchema{
								"type": {
									Type:     "string",
									Title:    "Layer Type",
									Required: true,
									Enum:     []interface{}{"regex", "llm", "custom"},
								},
								"provider": {
									Type:      "string",
									Title:     "Provider",
									DependsOn: []string{"type"},
								},
								"model": {
									Type:      "string",
									Title:     "Model",
									DependsOn: []string{"type", "provider"},
								},
								"threshold": {
									Type:     "number",
									Title:    "Threshold",
									Default:  0.8,
									MinValue: 0,
									MaxValue: 1,
								},
							},
						},
					},
				},
			},
			"database": {
				Title:       "Database Configuration",
				Description: "Database connection and storage settings",
				Order:       5,
				Icon:        "database",
				Advanced:    true,
				Properties: map[string]PropertySchema{
					"type": {
						Type:        "string",
						Title:       "Database Type",
						Description: "Type of database to use",
						Default:     "sqlite",
						Enum:        []interface{}{"sqlite", "postgresql"},
					},
					"connection_string": {
						Type:        "string",
						Title:       "Connection String",
						Description: "Full database connection string (overrides individual settings)",
						Sensitive:   true,
						UIComponent: "password",
					},
					"host": {
						Type:        "string",
						Title:       "Host",
						Description: "Database host",
						Default:     "localhost",
						DependsOn:   []string{"type"},
					},
					"port": {
						Type:        "number",
						Title:       "Port",
						Description: "Database port",
						Default:     5432,
						MinValue:    1,
						MaxValue:    65535,
						DependsOn:   []string{"type"},
					},
					"name": {
						Type:        "string",
						Title:       "Database Name",
						Description: "Name of the database",
						Default:     "qt1_middleware",
						DependsOn:   []string{"type"},
					},
					"username": {
						Type:        "string",
						Title:       "Username",
						Description: "Database username",
						DependsOn:   []string{"type"},
					},
					"password": {
						Type:        "string",
						Title:       "Password",
						Description: "Database password",
						Sensitive:   true,
						UIComponent: "password",
						DependsOn:   []string{"type"},
					},
					"sqlite_file": {
						Type:        "string",
						Title:       "SQLite File",
						Description: "Path to SQLite database file",
						Default:     "storage/qt1.db",
						DependsOn:   []string{"type"},
					},
					"max_connections": {
						Type:        "number",
						Title:       "Max Connections",
						Description: "Maximum number of database connections",
						Default:     25,
						MinValue:    1,
						MaxValue:    1000,
					},
				},
			},
		},
		Rules: []ValidationRule{
			{
				Name:        "redis_url_required",
				Description: "Redis URL is required when storage type is redis",
				Fields:      []string{"security.rate_limiting.storage.type", "security.rate_limiting.storage.redis_url"},
				Type:        "required_if",
				Condition:   "security.rate_limiting.storage.type == 'redis'",
				Message:     "Redis URL is required when storage type is set to redis",
			},
			{
				Name:        "database_connection_config",
				Description: "Database connection details required based on type",
				Fields:      []string{"database.type", "database.host", "database.port", "database.username"},
				Type:        "required_if",
				Condition:   "database.type == 'postgresql' && !database.connection_string",
				Message:     "PostgreSQL connection details are required when connection string is not provided",
			},
			{
				Name:        "provider_api_key",
				Description: "API key required for cloud providers",
				Fields:      []string{"providers.*.type", "providers.*.api_key"},
				Type:        "required_if",
				Condition:   "providers.*.type in ['openai', 'anthropic'] && providers.*.enabled",
				Message:     "API key is required for enabled cloud providers",
			},
		},
	}
}

// getRateLimitRuleSchema returns the schema for rate limit rules
func getRateLimitRuleSchema() map[string]PropertySchema {
	return map[string]PropertySchema{
		"requests_per_second": {
			Type:        "number",
			Title:       "Requests Per Second",
			Description: "Maximum requests per second",
			MinValue:    0,
		},
		"requests_per_minute": {
			Type:        "number",
			Title:       "Requests Per Minute",
			Description: "Maximum requests per minute",
			MinValue:    0,
		},
		"requests_per_hour": {
			Type:        "number",
			Title:       "Requests Per Hour",
			Description: "Maximum requests per hour",
			MinValue:    0,
		},
		"burst": {
			Type:        "number",
			Title:       "Burst Size",
			Description: "Maximum burst size for token bucket",
			Default:     10,
			MinValue:    0,
		},
		"window_size": {
			Type:        "duration",
			Title:       "Window Size",
			Description: "Time window for rate limiting",
			Default:     "1m",
		},
		"algorithm": {
			Type:        "string",
			Title:       "Algorithm",
			Description: "Rate limiting algorithm",
			Default:     "sliding_window",
			Enum:        []interface{}{"token_bucket", "sliding_window"},
		},
	}
}

// MarshalJSON implements custom JSON marshaling for ConfigSchema
func (cs *ConfigSchema) MarshalJSON() ([]byte, error) {
	type Alias ConfigSchema
	return json.Marshal(&struct {
		*Alias
		Generated time.Time `json:"generated"`
	}{
		Alias:     (*Alias)(cs),
		Generated: time.Now(),
	})
}

// ValidateValue validates a value against a PropertySchema
func (ps *PropertySchema) ValidateValue(value interface{}) error {
	// This will be implemented in validator.go
	return nil
}