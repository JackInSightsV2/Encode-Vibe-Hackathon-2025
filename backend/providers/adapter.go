package providers

import (
	"qt1-middleware/config"
	"time"
)

// ConfigAdapter provides methods to convert between legacy and enhanced provider configurations
type ConfigAdapter struct{}

// NewConfigAdapter creates a new configuration adapter
func NewConfigAdapter() *ConfigAdapter {
	return &ConfigAdapter{}
}

// LegacyToEnhanced converts a legacy provider configuration to enhanced configuration
func (ca *ConfigAdapter) LegacyToEnhanced(name string, legacy config.LegacyProvider) ProviderConfig {
	// Determine provider type based on base URL or name
	providerType := ca.inferProviderType(legacy)
	
	return ProviderConfig{
		Name:       name,
		Type:       providerType,
		Enabled:    true, // Legacy providers are enabled by default
		Priority:   1,
		APIKey:     legacy.APIKey,
		BaseURL:    legacy.BaseURL,
		Models:     legacy.Models,
		Headers:    legacy.Headers,
		Timeout:    time.Duration(legacy.Timeout) * time.Second,
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		
		// Default health check settings
		HealthCheck: &HealthCheckConfig{
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			MaxFailures: 3,
		},
		
		// Default rate limits based on provider type
		RateLimit: ca.getDefaultRateLimit(providerType),
		
		// Default pricing based on provider type
		Pricing: ca.getDefaultPricing(providerType),
	}
}

// EnhancedToLegacy converts an enhanced provider configuration to legacy configuration (for backward compatibility)
func (ca *ConfigAdapter) EnhancedToLegacy(enhanced ProviderConfig) config.LegacyProvider {
	return config.LegacyProvider{
		Name:    enhanced.Name,
		APIKey:  enhanced.APIKey,
		BaseURL: enhanced.BaseURL,
		Models:  enhanced.Models,
		Headers: enhanced.Headers,
		Timeout: int(enhanced.Timeout.Seconds()),
	}
}

// ConfigToEnhanced converts a config.EnhancedProvider to ProviderConfig
func (ca *ConfigAdapter) ConfigToEnhanced(enhanced config.EnhancedProvider) ProviderConfig {
	providerConfig := ProviderConfig{
		Name:         enhanced.Name,
		Type:         ProviderType(enhanced.Type),
		Enabled:      enhanced.Enabled,
		Version:      enhanced.Version,
		Priority:     enhanced.Priority,
		APIKey:       enhanced.APIKey,
		BaseURL:      enhanced.BaseURL,
		Models:       enhanced.Models,
		Headers:      enhanced.Headers,
		Timeout:      enhanced.Timeout,
		MaxRetries:   enhanced.MaxRetries,
		RetryDelay:   enhanced.RetryDelay,
		Tags:         enhanced.Tags,
		Dependencies: enhanced.Dependencies,
		Settings:     enhanced.Settings,
	}
	
	// Convert health check config
	if enhanced.HealthCheck != nil {
		providerConfig.HealthCheck = &HealthCheckConfig{
			Interval:       enhanced.HealthCheck.Interval,
			Timeout:        enhanced.HealthCheck.Timeout,
			MaxFailures:    enhanced.HealthCheck.MaxFailures,
			CustomEndpoint: enhanced.HealthCheck.CustomEndpoint,
			TestMessage:    enhanced.HealthCheck.TestMessage,
		}
	}
	
	// Convert rate limit config
	if enhanced.RateLimit != nil {
		providerConfig.RateLimit = &RateLimitConfig{
			RequestsPerSecond: enhanced.RateLimit.RequestsPerSecond,
			RequestsPerMinute: enhanced.RateLimit.RequestsPerMinute,
			RequestsPerHour:   enhanced.RateLimit.RequestsPerHour,
			TokensPerMinute:   enhanced.RateLimit.TokensPerMinute,
			BurstSize:         enhanced.RateLimit.BurstSize,
		}
	}
	
	// Convert pricing config
	if enhanced.Pricing != nil {
		providerConfig.Pricing = &PricingConfig{
			InputTokenCost:  enhanced.Pricing.InputTokenCost,
			OutputTokenCost: enhanced.Pricing.OutputTokenCost,
			RequestCost:     enhanced.Pricing.RequestCost,
			Currency:        enhanced.Pricing.Currency,
		}
	}
	
	return providerConfig
}

// EnhancedToConfig converts a ProviderConfig to config.EnhancedProvider
func (ca *ConfigAdapter) EnhancedToConfig(providerConfig ProviderConfig) config.EnhancedProvider {
	enhanced := config.EnhancedProvider{
		Name:         providerConfig.Name,
		Type:         config.ProviderType(providerConfig.Type),
		Enabled:      providerConfig.Enabled,
		Version:      providerConfig.Version,
		Priority:     providerConfig.Priority,
		APIKey:       providerConfig.APIKey,
		BaseURL:      providerConfig.BaseURL,
		Models:       providerConfig.Models,
		Headers:      providerConfig.Headers,
		Timeout:      providerConfig.Timeout,
		MaxRetries:   providerConfig.MaxRetries,
		RetryDelay:   providerConfig.RetryDelay,
		Tags:         providerConfig.Tags,
		Dependencies: providerConfig.Dependencies,
		Settings:     providerConfig.Settings,
	}
	
	// Convert health check config
	if providerConfig.HealthCheck != nil {
		enhanced.HealthCheck = &config.ProviderHealthConfig{
			Interval:       providerConfig.HealthCheck.Interval,
			Timeout:        providerConfig.HealthCheck.Timeout,
			MaxFailures:    providerConfig.HealthCheck.MaxFailures,
			CustomEndpoint: providerConfig.HealthCheck.CustomEndpoint,
			TestMessage:    providerConfig.HealthCheck.TestMessage,
		}
	}
	
	// Convert rate limit config
	if providerConfig.RateLimit != nil {
		enhanced.RateLimit = &config.ProviderRateLimit{
			RequestsPerSecond: providerConfig.RateLimit.RequestsPerSecond,
			RequestsPerMinute: providerConfig.RateLimit.RequestsPerMinute,
			RequestsPerHour:   providerConfig.RateLimit.RequestsPerHour,
			TokensPerMinute:   providerConfig.RateLimit.TokensPerMinute,
			BurstSize:         providerConfig.RateLimit.BurstSize,
		}
	}
	
	// Convert pricing config
	if providerConfig.Pricing != nil {
		enhanced.Pricing = &config.ProviderPricing{
			InputTokenCost:  providerConfig.Pricing.InputTokenCost,
			OutputTokenCost: providerConfig.Pricing.OutputTokenCost,
			RequestCost:     providerConfig.Pricing.RequestCost,
			Currency:        providerConfig.Pricing.Currency,
		}
	}
	
	return enhanced
}

// inferProviderType determines the provider type based on configuration
func (ca *ConfigAdapter) inferProviderType(legacy config.LegacyProvider) ProviderType {
	switch {
	case legacy.Name == "OpenAI" || legacy.BaseURL == "https://api.openai.com/v1":
		return ProviderTypeOpenAI
	case legacy.Name == "Anthropic" || legacy.BaseURL == "https://api.anthropic.com/v1":
		return ProviderTypeAnthropic
	case legacy.Name == "Local" || legacy.BaseURL == "http://localhost:11434/v1":
		return ProviderTypeLocal
	default:
		// Try to infer from base URL
		if legacy.BaseURL != "" {
			if legacy.BaseURL == "https://api.openai.com/v1" {
				return ProviderTypeOpenAI
			}
			if legacy.BaseURL == "https://api.anthropic.com/v1" {
				return ProviderTypeAnthropic
			}
			if legacy.BaseURL == "http://localhost:11434/v1" || legacy.BaseURL == "http://localhost:11434" {
				return ProviderTypeLocal
			}
		}
		// Default to local for unknown providers
		return ProviderTypeLocal
	}
}

// getDefaultRateLimit returns default rate limits for a provider type
func (ca *ConfigAdapter) getDefaultRateLimit(providerType ProviderType) *RateLimitConfig {
	switch providerType {
	case ProviderTypeOpenAI:
		return &RateLimitConfig{
			RequestsPerSecond: 10,
			RequestsPerMinute: 600,
			RequestsPerHour:   10000,
			TokensPerMinute:   90000,
			BurstSize:         20,
		}
	case ProviderTypeAnthropic:
		return &RateLimitConfig{
			RequestsPerSecond: 5,
			RequestsPerMinute: 300,
			RequestsPerHour:   4000,
			TokensPerMinute:   40000,
			BurstSize:         10,
		}
	case ProviderTypeLocal:
		return &RateLimitConfig{
			RequestsPerSecond: 1,
			RequestsPerMinute: 60,
			RequestsPerHour:   3600,
			TokensPerMinute:   10000,
			BurstSize:         5,
		}
	default:
		return &RateLimitConfig{
			RequestsPerSecond: 5,
			RequestsPerMinute: 300,
			RequestsPerHour:   3600,
			TokensPerMinute:   10000,
			BurstSize:         10,
		}
	}
}

// getDefaultPricing returns default pricing for a provider type
func (ca *ConfigAdapter) getDefaultPricing(providerType ProviderType) *PricingConfig {
	switch providerType {
	case ProviderTypeOpenAI:
		return &PricingConfig{
			InputTokenCost:  0.03,
			OutputTokenCost: 0.06,
			RequestCost:     0,
			Currency:        "USD",
		}
	case ProviderTypeAnthropic:
		return &PricingConfig{
			InputTokenCost:  0.015,
			OutputTokenCost: 0.075,
			RequestCost:     0,
			Currency:        "USD",
		}
	case ProviderTypeLocal:
		return &PricingConfig{
			InputTokenCost:  0,
			OutputTokenCost: 0,
			RequestCost:     0,
			Currency:        "USD",
		}
	default:
		return &PricingConfig{
			InputTokenCost:  0,
			OutputTokenCost: 0,
			RequestCost:     0,
			Currency:        "USD",
		}
	}
}

// LoadProviderConfigsFromAppConfig loads provider configurations from the global app config
func (ca *ConfigAdapter) LoadProviderConfigsFromAppConfig() []ProviderConfig {
	var configs []ProviderConfig
	
	// Load enhanced providers first (preferred)
	if config.AppConfig.EnhancedProviders != nil {
		for name, enhanced := range config.AppConfig.EnhancedProviders {
			providerConfig := ca.ConfigToEnhanced(enhanced)
			providerConfig.Name = name // Ensure name is set correctly
			configs = append(configs, providerConfig)
		}
	}
	
	// Load legacy providers for backward compatibility (only if no enhanced providers)
	if len(configs) == 0 && config.AppConfig.Providers != nil {
		for name, legacy := range config.AppConfig.Providers {
			providerConfig := ca.LegacyToEnhanced(name, legacy)
			configs = append(configs, providerConfig)
		}
	}
	
	return configs
}

// Global adapter instance
var DefaultAdapter = NewConfigAdapter()