package providers

import (
	"fmt"
)

// ProviderFactory creates provider instances based on configuration
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a provider instance based on the provider type
func (pf *ProviderFactory) CreateProvider(providerType ProviderType) (Provider, error) {
	switch providerType {
	case ProviderTypeOpenAI:
		return NewOpenAIProvider(), nil
	case ProviderTypeAnthropic:
		return NewAnthropicProvider(), nil
	case ProviderTypeLocal:
		return NewLocalProvider(), nil
	case ProviderTypeMock:
		return NewMockProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CreateProviderWithConfig creates and configures a provider instance
func (pf *ProviderFactory) CreateProviderWithConfig(config ProviderConfig) (Provider, error) {
	provider, err := pf.CreateProvider(config.Type)
	if err != nil {
		return nil, err
	}

	if err := provider.Configure(config); err != nil {
		return nil, fmt.Errorf("failed to configure provider: %w", err)
	}

	return provider, nil
}

// GetSupportedTypes returns all supported provider types
func (pf *ProviderFactory) GetSupportedTypes() []ProviderType {
	return []ProviderType{
		ProviderTypeOpenAI,
		ProviderTypeAnthropic,
		ProviderTypeLocal,
		ProviderTypeMock,
	}
}

// IsSupported checks if a provider type is supported
func (pf *ProviderFactory) IsSupported(providerType ProviderType) bool {
	for _, supported := range pf.GetSupportedTypes() {
		if supported == providerType {
			return true
		}
	}
	return false
}

// GetProviderInfo returns information about a provider type
func (pf *ProviderFactory) GetProviderInfo(providerType ProviderType) (ProviderInfo, error) {
	switch providerType {
	case ProviderTypeOpenAI:
		return ProviderInfo{
			Type:         ProviderTypeOpenAI,
			Name:         "OpenAI",
			Description:  "OpenAI GPT models",
			RequiresKey:  true,
			DefaultURL:   "https://api.openai.com/v1",
			SupportedModels: []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		}, nil
	case ProviderTypeAnthropic:
		return ProviderInfo{
			Type:         ProviderTypeAnthropic,
			Name:         "Anthropic",
			Description:  "Anthropic Claude models",
			RequiresKey:  true,
			DefaultURL:   "https://api.anthropic.com",
			SupportedModels: []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		}, nil
	case ProviderTypeLocal:
		return ProviderInfo{
			Type:         ProviderTypeLocal,
			Name:         "Local LLM",
			Description:  "Local LLM servers (Ollama, LM Studio, etc.)",
			RequiresKey:  false,
			DefaultURL:   "http://localhost:11434",
			SupportedModels: []string{"llama2", "mistral", "codellama", "phi"},
		}, nil
	case ProviderTypeMock:
		return ProviderInfo{
			Type:         ProviderTypeMock,
			Name:         "Mock Provider",
			Description:  "Mock provider for testing",
			RequiresKey:  false,
			DefaultURL:   "",
			SupportedModels: []string{"mock-model", "test-model"},
		}, nil
	default:
		return ProviderInfo{}, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// ProviderInfo contains information about a provider type
type ProviderInfo struct {
	Type            ProviderType `json:"type"`
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	RequiresKey     bool         `json:"requires_key"`
	DefaultURL      string       `json:"default_url"`
	SupportedModels []string     `json:"supported_models"`
}

// Global factory instance
var DefaultFactory = NewProviderFactory()