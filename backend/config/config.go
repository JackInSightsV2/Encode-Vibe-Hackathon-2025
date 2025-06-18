package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Provider struct {
	Name     string            `yaml:"name" json:"name"`
	APIKey   string            `yaml:"api_key" json:"api_key"`
	BaseURL  string            `yaml:"base_url" json:"base_url"`
	Models   []string          `yaml:"models" json:"models"`
	Headers  map[string]string `yaml:"headers" json:"headers"`
	Timeout  int               `yaml:"timeout" json:"timeout"`
}

type Config struct {
	Server struct {
		Port      int    `yaml:"port" json:"port"`
		Host      string `yaml:"host" json:"host"`
		TargetURL string `yaml:"target_url" json:"target_url"` // Legacy support
	} `yaml:"server" json:"server"`
	
	Providers map[string]Provider `yaml:"providers" json:"providers"`
	
	Routing struct {
		DefaultProvider string            `yaml:"default_provider" json:"default_provider"`
		ModelRouting    map[string]string `yaml:"model_routing" json:"model_routing"`
		FallbackChain   []string          `yaml:"fallback_chain" json:"fallback_chain"`
	} `yaml:"routing" json:"routing"`
	
	Security struct {
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
}

type ModerationLayer struct {
	Type       string            `yaml:"type" json:"type"` // "regex", "llm", "custom"
	Provider   string            `yaml:"provider" json:"provider"`
	Model      string            `yaml:"model" json:"model"`
	Prompt     string            `yaml:"prompt" json:"prompt"`
	Threshold  float64           `yaml:"threshold" json:"threshold"`
	Settings   map[string]interface{} `yaml:"settings" json:"settings"`
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
	
	// Default providers
	AppConfig.Providers = make(map[string]Provider)
	AppConfig.Providers["openai"] = Provider{
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com/v1",
		Models:  []string{"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 30,
	}
	AppConfig.Providers["anthropic"] = Provider{
		Name:    "Anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		Models:  []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		Headers: map[string]string{
			"Content-Type":     "application/json",
			"anthropic-version": "2023-06-01",
		},
		Timeout: 30,
	}
	AppConfig.Providers["local"] = Provider{
		Name:    "Local",
		BaseURL: "http://localhost:11434/v1",
		Models:  []string{"llama2", "mistral", "codellama"},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 60,
	}
	
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