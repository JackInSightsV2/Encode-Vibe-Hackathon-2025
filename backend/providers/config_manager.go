package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ProviderConfigManager manages provider configurations with hot-reload capabilities
type ProviderConfigManager struct {
	registry     *ProviderRegistry
	manager      *ProviderManager
	configs      map[string]*ProviderConfig
	watchers     []ConfigWatcher
	validator    *ConfigValidator
	fileWatcher  *FileWatcher
	mutex        sync.RWMutex
	configPath   string
}

// ConfigWatcher interface for configuration change notifications
type ConfigWatcher interface {
	OnConfigChanged(providerName string, oldConfig, newConfig *ProviderConfig) error
	OnProviderAdded(providerName string, config *ProviderConfig) error
	OnProviderRemoved(providerName string) error
}

// ConfigValidator validates provider configurations
type ConfigValidator struct{}

// NewProviderConfigManager creates a new provider configuration manager
func NewProviderConfigManager(registry *ProviderRegistry, manager *ProviderManager) *ProviderConfigManager {
	return &ProviderConfigManager{
		registry:  registry,
		manager:   manager,
		configs:   make(map[string]*ProviderConfig),
		watchers:  make([]ConfigWatcher, 0),
		validator: NewConfigValidator(),
	}
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// LoadConfig loads provider configurations from a file
func (pcm *ProviderConfigManager) LoadConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configs map[string]*ProviderConfig
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate all configurations
	for name, config := range configs {
		config.Name = name // Ensure name is set
		if err := pcm.validator.Validate(config); err != nil {
			return fmt.Errorf("invalid config for provider %s: %w", name, err)
		}
	}

	// Apply configurations
	for name, config := range configs {
		if err := pcm.UpdateConfig(name, config); err != nil {
			return fmt.Errorf("failed to update config for %s: %w", name, err)
		}
	}

	// Store config path for hot-reload
	pcm.configPath = filePath

	return nil
}

// LoadConfigFromBytes loads provider configurations from byte data
func (pcm *ProviderConfigManager) LoadConfigFromBytes(data []byte) error {
	var configs map[string]*ProviderConfig
	if err := yaml.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate all configurations
	for name, config := range configs {
		config.Name = name // Ensure name is set
		if err := pcm.validator.Validate(config); err != nil {
			return fmt.Errorf("invalid config for provider %s: %w", name, err)
		}
	}

	// Apply configurations
	for name, config := range configs {
		if err := pcm.UpdateConfig(name, config); err != nil {
			return fmt.Errorf("failed to update config for %s: %w", name, err)
		}
	}

	return nil
}

// EnableHotReload enables hot-reload for the configuration file
func (pcm *ProviderConfigManager) EnableHotReload() error {
	if pcm.configPath == "" {
		return fmt.Errorf("no config file path set")
	}

	if pcm.fileWatcher != nil {
		return fmt.Errorf("hot-reload already enabled")
	}

	var err error
	pcm.fileWatcher, err = NewFileWatcher(pcm.configPath, pcm.onConfigFileChanged)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	return pcm.fileWatcher.Start()
}

// DisableHotReload disables hot-reload
func (pcm *ProviderConfigManager) DisableHotReload() error {
	if pcm.fileWatcher == nil {
		return nil // Already disabled
	}

	err := pcm.fileWatcher.Stop()
	pcm.fileWatcher = nil
	return err
}

// onConfigFileChanged handles configuration file changes
func (pcm *ProviderConfigManager) onConfigFileChanged() {
	fmt.Printf("Configuration file changed, reloading...\n")
	
	if err := pcm.LoadConfig(pcm.configPath); err != nil {
		fmt.Printf("Failed to reload configuration: %v\n", err)
	} else {
		fmt.Printf("Configuration reloaded successfully\n")
	}
}

// UpdateConfig updates a provider's configuration
func (pcm *ProviderConfigManager) UpdateConfig(providerName string, config *ProviderConfig) error {
	pcm.mutex.Lock()
	defer pcm.mutex.Unlock()

	oldConfig := pcm.configs[providerName]

	// Validate new configuration
	if err := pcm.validator.Validate(config); err != nil {
		return err
	}

	// Check dependencies
	if err := pcm.checkDependencies(config); err != nil {
		return err
	}

	// Check if provider exists in registry
	provider, err := pcm.registry.Get(providerName)
	if err != nil {
		// Provider doesn't exist, create it
		if err := pcm.createProvider(providerName, config); err != nil {
			return fmt.Errorf("failed to create provider: %w", err)
		}
	} else {
		// Provider exists, update configuration
		if err := provider.Configure(*config); err != nil {
			return fmt.Errorf("provider rejected configuration: %w", err)
		}

		// If enabled state changed, start/stop provider
		if oldConfig != nil && oldConfig.Enabled != config.Enabled {
			if config.Enabled {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := provider.Start(ctx); err != nil {
					return fmt.Errorf("failed to start provider: %w", err)
				}
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := provider.Stop(ctx); err != nil {
					return fmt.Errorf("failed to stop provider: %w", err)
				}
			}
		}
	}

	// Store new configuration
	pcm.configs[providerName] = config

	// Notify watchers
	for _, watcher := range pcm.watchers {
		if oldConfig == nil {
			if err := watcher.OnProviderAdded(providerName, config); err != nil {
				fmt.Printf("Config watcher error on provider added: %v\n", err)
			}
		} else {
			if err := watcher.OnConfigChanged(providerName, oldConfig, config); err != nil {
				fmt.Printf("Config watcher error on config changed: %v\n", err)
			}
		}
	}

	return nil
}

// createProvider creates a new provider with the given configuration
func (pcm *ProviderConfigManager) createProvider(providerName string, config *ProviderConfig) error {
	if pcm.manager == nil {
		return fmt.Errorf("provider manager not available")
	}

	// Use the provider manager to register the new provider
	return pcm.manager.RegisterProvider(*config)
}

// RemoveConfig removes a provider configuration
func (pcm *ProviderConfigManager) RemoveConfig(providerName string) error {
	pcm.mutex.Lock()
	defer pcm.mutex.Unlock()

	config, exists := pcm.configs[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	// Stop and unregister provider
	if pcm.manager != nil {
		if err := pcm.manager.UnregisterProvider(providerName); err != nil {
			return fmt.Errorf("failed to unregister provider: %w", err)
		}
	}

	// Remove from configs
	delete(pcm.configs, providerName)

	// Notify watchers
	for _, watcher := range pcm.watchers {
		if err := watcher.OnProviderRemoved(providerName); err != nil {
			fmt.Printf("Config watcher error on provider removed: %v\n", err)
		}
	}

	// If this was the only reference to the config, clean up
	_ = config

	return nil
}

// GetConfig returns a provider's configuration
func (pcm *ProviderConfigManager) GetConfig(providerName string) (*ProviderConfig, error) {
	pcm.mutex.RLock()
	defer pcm.mutex.RUnlock()

	config, exists := pcm.configs[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Return a copy to prevent external modification
	configCopy := *config
	return &configCopy, nil
}

// GetAllConfigs returns all provider configurations
func (pcm *ProviderConfigManager) GetAllConfigs() map[string]*ProviderConfig {
	pcm.mutex.RLock()
	defer pcm.mutex.RUnlock()

	result := make(map[string]*ProviderConfig)
	for name, config := range pcm.configs {
		configCopy := *config
		result[name] = &configCopy
	}

	return result
}

// EnableProvider enables a provider
func (pcm *ProviderConfigManager) EnableProvider(providerName string) error {
	pcm.mutex.Lock()
	defer pcm.mutex.Unlock()

	config, exists := pcm.configs[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if config.Enabled {
		return nil // Already enabled
	}

	// Check dependencies
	if err := pcm.checkDependencies(config); err != nil {
		return fmt.Errorf("cannot enable provider: %w", err)
	}

	provider, err := pcm.registry.Get(providerName)
	if err != nil {
		return err
	}

	// Start the provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.Start(ctx); err != nil {
		return fmt.Errorf("failed to start provider: %w", err)
	}

	// Update configuration
	oldConfig := *config
	config.Enabled = true

	// Notify watchers
	for _, watcher := range pcm.watchers {
		if err := watcher.OnConfigChanged(providerName, &oldConfig, config); err != nil {
			fmt.Printf("Config watcher error: %v\n", err)
		}
	}

	return nil
}

// DisableProvider disables a provider
func (pcm *ProviderConfigManager) DisableProvider(providerName string) error {
	pcm.mutex.Lock()
	defer pcm.mutex.Unlock()

	config, exists := pcm.configs[providerName]
	if !exists {
		return fmt.Errorf("provider %s not found", providerName)
	}

	if !config.Enabled {
		return nil // Already disabled
	}

	provider, err := pcm.registry.Get(providerName)
	if err != nil {
		return err
	}

	// Stop the provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop provider: %w", err)
	}

	// Update configuration
	oldConfig := *config
	config.Enabled = false

	// Notify watchers
	for _, watcher := range pcm.watchers {
		if err := watcher.OnConfigChanged(providerName, &oldConfig, config); err != nil {
			fmt.Printf("Config watcher error: %v\n", err)
		}
	}

	return nil
}

// TestProvider tests a provider configuration
func (pcm *ProviderConfigManager) TestProvider(providerName string, testConfig *ProviderConfig) (*ProviderTestResult, error) {
	// Create a temporary provider instance for testing
	provider, err := pcm.createProviderInstance(testConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		provider.Stop(ctx)
	}()

	result := &ProviderTestResult{
		Provider:  providerName,
		StartTime: time.Now(),
	}

	// Test configuration validation
	if err := pcm.validator.Validate(testConfig); err != nil {
		result.ConfigValidation = &TestResult{
			Success: false,
			Error:   err.Error(),
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, nil
	}
	result.ConfigValidation = &TestResult{Success: true}

	// Test provider validation
	if err := provider.Validate(); err != nil {
		result.ConfigValidation = &TestResult{
			Success: false,
			Error:   err.Error(),
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, nil
	}

	// Test startup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.Start(ctx); err != nil {
		result.Startup = &TestResult{
			Success: false,
			Error:   err.Error(),
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, nil
	}
	result.Startup = &TestResult{Success: true}

	// Test health check
	if err := provider.HealthCheck(ctx); err != nil {
		result.HealthCheck = &TestResult{
			Success: false,
			Error:   err.Error(),
		}
	} else {
		result.HealthCheck = &TestResult{Success: true}
	}

	// Test basic functionality
	testReq := &ChatRequest{
		UserID:    "test",
		SessionID: "test",
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: "Test message",
			},
		},
	}

	_, err = provider.SendRequest(ctx, testReq)
	if err != nil {
		result.Functionality = &TestResult{
			Success: false,
			Error:   err.Error(),
		}
	} else {
		result.Functionality = &TestResult{Success: true}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// createProviderInstance creates a temporary provider instance for testing
func (pcm *ProviderConfigManager) createProviderInstance(config *ProviderConfig) (Provider, error) {
	if pcm.manager == nil {
		return nil, fmt.Errorf("provider manager not available")
	}

	factory := pcm.manager.factory
	return factory.CreateProviderWithConfig(*config)
}

// checkDependencies checks if provider dependencies are satisfied
func (pcm *ProviderConfigManager) checkDependencies(config *ProviderConfig) error {
	for _, dependency := range config.Dependencies {
		depConfig, exists := pcm.configs[dependency]
		if !exists {
			return fmt.Errorf("dependency %s not found", dependency)
		}

		if !depConfig.Enabled {
			return fmt.Errorf("dependency %s is not enabled", dependency)
		}
	}

	return nil
}

// AddWatcher adds a configuration watcher
func (pcm *ProviderConfigManager) AddWatcher(watcher ConfigWatcher) {
	pcm.mutex.Lock()
	defer pcm.mutex.Unlock()
	pcm.watchers = append(pcm.watchers, watcher)
}

// RemoveWatcher removes a configuration watcher
func (pcm *ProviderConfigManager) RemoveWatcher(watcher ConfigWatcher) {
	pcm.mutex.Lock()
	defer pcm.mutex.Unlock()

	for i, w := range pcm.watchers {
		if w == watcher {
			pcm.watchers = append(pcm.watchers[:i], pcm.watchers[i+1:]...)
			break
		}
	}
}

// SaveConfig saves the current configuration to file
func (pcm *ProviderConfigManager) SaveConfig(filePath string) error {
	pcm.mutex.RLock()
	defer pcm.mutex.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(pcm.configs)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// ExportConfig exports configuration to bytes
func (pcm *ProviderConfigManager) ExportConfig() ([]byte, error) {
	pcm.mutex.RLock()
	defer pcm.mutex.RUnlock()

	return yaml.Marshal(pcm.configs)
}

// Configuration validator implementation
func (cv *ConfigValidator) Validate(config *ProviderConfig) error {
	if config.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if config.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	// Validate provider type
	validTypes := map[ProviderType]bool{
		ProviderTypeOpenAI:    true,
		ProviderTypeAnthropic: true,
		ProviderTypeLocal:     true,
		ProviderTypeMock:      true,
	}

	if !validTypes[config.Type] {
		return fmt.Errorf("invalid provider type: %s", config.Type)
	}

	// Validate API key for cloud providers
	if (config.Type == ProviderTypeOpenAI || config.Type == ProviderTypeAnthropic) && config.APIKey == "" {
		return fmt.Errorf("API key is required for provider type %s", config.Type)
	}

	// Validate base URL
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	// Validate timeout
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Validate priority
	if config.Priority < 0 {
		return fmt.Errorf("priority must be non-negative")
	}

	// Validate health check configuration
	if config.HealthCheck != nil {
		if config.HealthCheck.Interval <= 0 {
			return fmt.Errorf("health check interval must be positive")
		}
		if config.HealthCheck.Timeout <= 0 {
			return fmt.Errorf("health check timeout must be positive")
		}
		if config.HealthCheck.MaxFailures <= 0 {
			return fmt.Errorf("health check max failures must be positive")
		}
	}

	// Validate rate limit configuration
	if config.RateLimit != nil {
		if config.RateLimit.RequestsPerSecond < 0 {
			return fmt.Errorf("requests per second must be non-negative")
		}
		if config.RateLimit.RequestsPerMinute < 0 {
			return fmt.Errorf("requests per minute must be non-negative")
		}
		if config.RateLimit.RequestsPerHour < 0 {
			return fmt.Errorf("requests per hour must be non-negative")
		}
		if config.RateLimit.TokensPerMinute < 0 {
			return fmt.Errorf("tokens per minute must be non-negative")
		}
		if config.RateLimit.BurstSize < 0 {
			return fmt.Errorf("burst size must be non-negative")
		}
	}

	// Validate pricing configuration
	if config.Pricing != nil {
		if config.Pricing.InputTokenCost < 0 {
			return fmt.Errorf("input token cost must be non-negative")
		}
		if config.Pricing.OutputTokenCost < 0 {
			return fmt.Errorf("output token cost must be non-negative")
		}
		if config.Pricing.RequestCost < 0 {
			return fmt.Errorf("request cost must be non-negative")
		}
		if config.Pricing.Currency == "" {
			config.Pricing.Currency = "USD" // Default currency
		}
	}

	return nil
}

// ProviderTestResult contains the results of provider testing
type ProviderTestResult struct {
	Provider         string        `json:"provider"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	ConfigValidation *TestResult   `json:"config_validation"`
	Startup          *TestResult   `json:"startup"`
	HealthCheck      *TestResult   `json:"health_check"`
	Functionality    *TestResult   `json:"functionality"`
}

// TestResult represents the result of a single test
type TestResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// IsSuccessful returns true if all tests passed
func (ptr *ProviderTestResult) IsSuccessful() bool {
	return ptr.ConfigValidation != nil && ptr.ConfigValidation.Success &&
		ptr.Startup != nil && ptr.Startup.Success &&
		ptr.HealthCheck != nil && ptr.HealthCheck.Success &&
		ptr.Functionality != nil && ptr.Functionality.Success
}

// GetSummary returns a summary of the test results
func (ptr *ProviderTestResult) GetSummary() string {
	if ptr.IsSuccessful() {
		return fmt.Sprintf("All tests passed in %v", ptr.Duration)
	}

	failedTests := make([]string, 0)
	if ptr.ConfigValidation != nil && !ptr.ConfigValidation.Success {
		failedTests = append(failedTests, "config validation")
	}
	if ptr.Startup != nil && !ptr.Startup.Success {
		failedTests = append(failedTests, "startup")
	}
	if ptr.HealthCheck != nil && !ptr.HealthCheck.Success {
		failedTests = append(failedTests, "health check")
	}
	if ptr.Functionality != nil && !ptr.Functionality.Success {
		failedTests = append(failedTests, "functionality")
	}

	if len(failedTests) > 0 {
		return fmt.Sprintf("Failed tests: %v (duration: %v)", failedTests, ptr.Duration)
	}

	return fmt.Sprintf("Some tests not completed (duration: %v)", ptr.Duration)
}