package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProviderManager manages all providers and provides high-level operations
type ProviderManager struct {
	registry      *ProviderRegistry
	factory       *ProviderFactory
	config        *ManagerConfig
	healthMonitor *HealthMonitor
	routingEngine *RoutingEngine
	circuitBreakers *CircuitBreakerManager
	configManager *ProviderConfigManager
	mutex         sync.RWMutex
	started       bool
}

// ManagerConfig contains configuration for the provider manager
type ManagerConfig struct {
	AutoStart          bool          `yaml:"auto_start" json:"auto_start"`
	StartTimeout       time.Duration `yaml:"start_timeout" json:"start_timeout"`
	StopTimeout        time.Duration `yaml:"stop_timeout" json:"stop_timeout"`
	HealthCheckEnabled bool          `yaml:"health_check_enabled" json:"health_check_enabled"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
}

// NewProviderManager creates a new provider manager
func NewProviderManager(config *ManagerConfig) *ProviderManager {
	if config == nil {
		config = &ManagerConfig{
			AutoStart:           true,
			StartTimeout:        30 * time.Second,
			StopTimeout:         10 * time.Second,
			HealthCheckEnabled:  true,
			HealthCheckInterval: 30 * time.Second,
		}
	}

	manager := &ProviderManager{
		registry: NewProviderRegistry(),
		factory:  NewProviderFactory(),
		config:   config,
		started:  false,
	}

	// Initialize health monitor
	healthConfig := &HealthConfig{
		CheckInterval:    config.HealthCheckInterval,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}
	manager.healthMonitor = NewHealthMonitor(manager.registry, healthConfig)

	// Initialize routing engine
	routingConfig := &RoutingConfig{
		Strategy: "intelligent",
		Factors: map[string]float64{
			"health":      0.4,
			"latency":     0.3,
			"cost":        0.2,
			"availability": 0.1,
		},
	}
	manager.routingEngine = NewRoutingEngine(manager.registry, manager.healthMonitor, routingConfig)

	// Initialize circuit breaker manager
	circuitConfig := &CircuitBreakerConfig{
		FailureThreshold:    5,
		RecoveryTimeout:     30 * time.Second,
		HalfOpenMaxCalls:    3,
		HalfOpenSuccessThreshold: 2,
	}
	manager.circuitBreakers = NewCircuitBreakerManager(circuitConfig)

	// Initialize configuration manager
	manager.configManager = NewProviderConfigManager(manager.registry, manager)

	return manager
}

// Initialize initializes the provider manager with a list of provider configurations
func (pm *ProviderManager) Initialize(providerConfigs []ProviderConfig) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.started {
		return fmt.Errorf("provider manager already started")
	}

	// Create and register providers
	for _, config := range providerConfigs {
		provider, err := pm.factory.CreateProviderWithConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create provider %s: %w", config.Name, err)
		}

		if err := pm.registry.RegisterWithConfig(provider, config); err != nil {
			return fmt.Errorf("failed to register provider %s: %w", config.Name, err)
		}
	}

	return nil
}

// Start starts all enabled providers
func (pm *ProviderManager) Start(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.started {
		return nil // Already started
	}

	// Create timeout context
	startCtx, cancel := context.WithTimeout(ctx, pm.config.StartTimeout)
	defer cancel()

	// Start all providers
	if err := pm.registry.StartAll(startCtx); err != nil {
		return fmt.Errorf("failed to start providers: %w", err)
	}

	pm.started = true

	// Start health monitoring if enabled
	if pm.config.HealthCheckEnabled {
		if err := pm.healthMonitor.Start(startCtx); err != nil {
			return fmt.Errorf("failed to start health monitoring: %w", err)
		}
	}

	return nil
}

// Stop stops all providers
func (pm *ProviderManager) Stop(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.started {
		return nil // Already stopped
	}

	// Create timeout context
	stopCtx, cancel := context.WithTimeout(ctx, pm.config.StopTimeout)
	defer cancel()

	// Stop health monitoring
	if pm.healthMonitor != nil {
		if err := pm.healthMonitor.Stop(stopCtx); err != nil {
			fmt.Printf("Warning: failed to stop health monitoring: %v\n", err)
		}
	}

	// Stop all providers
	if err := pm.registry.StopAll(stopCtx); err != nil {
		return fmt.Errorf("failed to stop providers: %w", err)
	}

	pm.started = false
	return nil
}

// RegisterProvider registers a new provider
func (pm *ProviderManager) RegisterProvider(config ProviderConfig) error {
	provider, err := pm.factory.CreateProviderWithConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	if err := pm.registry.RegisterWithConfig(provider, config); err != nil {
		return fmt.Errorf("failed to register provider: %w", err)
	}

	// Start the provider if the manager is already started
	if pm.started && config.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), pm.config.StartTimeout)
		defer cancel()

		if err := provider.Start(ctx); err != nil {
			return fmt.Errorf("failed to start provider: %w", err)
		}
	}

	return nil
}

// UnregisterProvider unregisters a provider
func (pm *ProviderManager) UnregisterProvider(name string) error {
	return pm.registry.Unregister(name)
}

// GetProvider retrieves a provider by name
func (pm *ProviderManager) GetProvider(name string) (Provider, error) {
	return pm.registry.Get(name)
}

// ListProviders returns all registered providers
func (pm *ProviderManager) ListProviders() []Provider {
	return pm.registry.List()
}

// ListEnabledProviders returns only enabled providers
func (pm *ProviderManager) ListEnabledProviders() []Provider {
	return pm.registry.ListEnabled()
}

// GetProviderStatus returns status for all providers
func (pm *ProviderManager) GetProviderStatus() map[string]ProviderStatus {
	return pm.registry.GetStatus()
}

// GetProviderMetrics returns metrics for all providers
func (pm *ProviderManager) GetProviderMetrics() map[string]ProviderMetrics {
	return pm.registry.GetMetrics()
}

// SendRequest routes a request to an appropriate provider using intelligent routing
func (pm *ProviderManager) SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Create routing request
	routingReq := &RoutingRequest{
		Model:     req.Model,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		Priority:  PriorityNormal, // Default priority
		Constraints: RoutingConstraints{
			MaxLatency: 30 * time.Second, // Default max latency
		},
		Metadata: make(map[string]interface{}),
	}

	// Use routing engine to select provider
	routingResult, err := pm.routingEngine.Route(ctx, routingReq)
	if err != nil {
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Get the selected provider
	provider, err := pm.registry.Get(routingResult.Provider)
	if err != nil {
		return nil, fmt.Errorf("selected provider %s not found: %w", routingResult.Provider, err)
	}

	// Check circuit breaker
	circuitBreaker := pm.circuitBreakers.GetCircuitBreaker(routingResult.Provider)
	if !circuitBreaker.CanExecute() {
		// Try alternatives if circuit breaker is open
		return pm.tryAlternatives(ctx, req, routingResult.Alternatives)
	}

	// Execute request with circuit breaker protection
	var response *ChatResponse
	err = circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		response, execErr = provider.SendRequest(ctx, req)
		return execErr
	})

	if err != nil {
		// If circuit breaker error or provider error, try alternatives
		if IsCircuitBreakerError(err) || pm.isRetryableError(err) {
			return pm.tryAlternatives(ctx, req, routingResult.Alternatives)
		}
		return nil, fmt.Errorf("provider %s failed: %w", routingResult.Provider, err)
	}

	return response, nil
}

// tryAlternatives attempts to use alternative providers
func (pm *ProviderManager) tryAlternatives(ctx context.Context, req *ChatRequest, alternatives []Alternative) (*ChatResponse, error) {
	for _, alt := range alternatives {
		provider, err := pm.registry.Get(alt.Provider)
		if err != nil {
			continue
		}

		// Check circuit breaker for alternative
		circuitBreaker := pm.circuitBreakers.GetCircuitBreaker(alt.Provider)
		if !circuitBreaker.CanExecute() {
			continue
		}

		// Try the alternative provider
		var response *ChatResponse
		err = circuitBreaker.Execute(ctx, func(ctx context.Context) error {
			var execErr error
			response, execErr = provider.SendRequest(ctx, req)
			return execErr
		})

		if err == nil {
			return response, nil
		}

		// If this alternative also fails with non-retryable error, stop trying
		if !pm.isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("all providers failed")
}

// isRetryableError determines if an error is retryable
func (pm *ProviderManager) isRetryableError(err error) bool {
	if providerErr, ok := err.(*ProviderError); ok {
		return providerErr.Retryable
	}
	return true // Default to retryable for unknown errors
}

// GetRoutingEngine returns the routing engine instance
func (pm *ProviderManager) GetRoutingEngine() *RoutingEngine {
	return pm.routingEngine
}

// GetCircuitBreakerManager returns the circuit breaker manager
func (pm *ProviderManager) GetCircuitBreakerManager() *CircuitBreakerManager {
	return pm.circuitBreakers
}

// GetRoutingStats returns routing statistics
func (pm *ProviderManager) GetRoutingStats() map[string]interface{} {
	if pm.routingEngine == nil {
		return map[string]interface{}{
			"error": "routing engine not available",
		}
	}
	return pm.routingEngine.GetMetrics().GetOverallStats()
}

// GetCircuitBreakerStates returns the state of all circuit breakers
func (pm *ProviderManager) GetCircuitBreakerStates() map[string]CircuitBreakerState {
	if pm.circuitBreakers == nil {
		return make(map[string]CircuitBreakerState)
	}
	return pm.circuitBreakers.GetAllStates()
}

// GetConfigManager returns the configuration manager
func (pm *ProviderManager) GetConfigManager() *ProviderConfigManager {
	return pm.configManager
}

// LoadConfigFromFile loads provider configurations from a file
func (pm *ProviderManager) LoadConfigFromFile(filePath string) error {
	if pm.configManager == nil {
		return fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.LoadConfig(filePath)
}

// LoadConfigFromBytes loads provider configurations from byte data
func (pm *ProviderManager) LoadConfigFromBytes(data []byte) error {
	if pm.configManager == nil {
		return fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.LoadConfigFromBytes(data)
}

// EnableHotReload enables hot-reload for configuration files
func (pm *ProviderManager) EnableHotReload() error {
	if pm.configManager == nil {
		return fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.EnableHotReload()
}

// DisableHotReload disables hot-reload
func (pm *ProviderManager) DisableHotReload() error {
	if pm.configManager == nil {
		return fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.DisableHotReload()
}

// UpdateProviderConfig updates a provider's configuration
func (pm *ProviderManager) UpdateProviderConfig(providerName string, config *ProviderConfig) error {
	if pm.configManager == nil {
		return fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.UpdateConfig(providerName, config)
}

// GetProviderConfig returns a provider's configuration
func (pm *ProviderManager) GetProviderConfig(providerName string) (*ProviderConfig, error) {
	if pm.configManager == nil {
		return nil, fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.GetConfig(providerName)
}

// GetAllProviderConfigs returns all provider configurations
func (pm *ProviderManager) GetAllProviderConfigs() map[string]*ProviderConfig {
	if pm.configManager == nil {
		return make(map[string]*ProviderConfig)
	}
	return pm.configManager.GetAllConfigs()
}

// TestProviderConfig tests a provider configuration
func (pm *ProviderManager) TestProviderConfig(providerName string, config *ProviderConfig) (*ProviderTestResult, error) {
	if pm.configManager == nil {
		return nil, fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.TestProvider(providerName, config)
}

// SaveConfig saves the current configuration to a file
func (pm *ProviderManager) SaveConfig(filePath string) error {
	if pm.configManager == nil {
		return fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.SaveConfig(filePath)
}

// ExportConfig exports configuration to bytes
func (pm *ProviderManager) ExportConfig() ([]byte, error) {
	if pm.configManager == nil {
		return nil, fmt.Errorf("configuration manager not available")
	}
	return pm.configManager.ExportConfig()
}

// EnableProvider enables a provider
func (pm *ProviderManager) EnableProvider(name string) error {
	provider, err := pm.registry.Get(name)
	if err != nil {
		return err
	}

	config, err := pm.registry.GetConfig(name)
	if err != nil {
		return err
	}

	if config.Enabled {
		return nil // Already enabled
	}

	// Update config
	config.Enabled = true
	if err := pm.registry.Configure(name, config); err != nil {
		return err
	}

	// Start the provider if manager is started
	if pm.started {
		ctx, cancel := context.WithTimeout(context.Background(), pm.config.StartTimeout)
		defer cancel()

		if err := provider.Start(ctx); err != nil {
			return fmt.Errorf("failed to start provider: %w", err)
		}
	}

	return nil
}

// DisableProvider disables a provider
func (pm *ProviderManager) DisableProvider(name string) error {
	provider, err := pm.registry.Get(name)
	if err != nil {
		return err
	}

	config, err := pm.registry.GetConfig(name)
	if err != nil {
		return err
	}

	if !config.Enabled {
		return nil // Already disabled
	}

	// Stop the provider
	ctx, cancel := context.WithTimeout(context.Background(), pm.config.StopTimeout)
	defer cancel()

	if err := provider.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop provider: %w", err)
	}

	// Update config
	config.Enabled = false
	if err := pm.registry.Configure(name, config); err != nil {
		return err
	}

	return nil
}

// Duplicate UpdateProviderConfig method removed - using the one with pointer parameter

// HealthCheckAll performs health checks on all enabled providers
func (pm *ProviderManager) HealthCheckAll(ctx context.Context) map[string]error {
	providers := pm.registry.ListEnabled()
	results := make(map[string]error)

	for _, provider := range providers {
		err := provider.HealthCheck(ctx)
		results[provider.Name()] = err
	}

	return results
}

// GetSupportedProviderTypes returns all supported provider types
func (pm *ProviderManager) GetSupportedProviderTypes() []ProviderType {
	return pm.factory.GetSupportedTypes()
}

// GetProviderInfo returns information about a provider type
func (pm *ProviderManager) GetProviderInfo(providerType ProviderType) (ProviderInfo, error) {
	return pm.factory.GetProviderInfo(providerType)
}

// IsStarted returns whether the manager is started
func (pm *ProviderManager) IsStarted() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.started
}

// GetRegistry returns the provider registry (for advanced usage)
func (pm *ProviderManager) GetRegistry() *ProviderRegistry {
	return pm.registry
}

// GetHealthMonitor returns the health monitor instance
func (pm *ProviderManager) GetHealthMonitor() *HealthMonitor {
	return pm.healthMonitor
}

// GetProviderHealth returns health information for a specific provider
func (pm *ProviderManager) GetProviderHealth(providerName string) (*ProviderHealth, error) {
	if pm.healthMonitor == nil {
		return nil, fmt.Errorf("health monitoring not enabled")
	}
	return pm.healthMonitor.GetProviderHealth(providerName)
}

// GetAllProviderHealth returns health information for all providers
func (pm *ProviderManager) GetAllProviderHealth() (map[string]*ProviderHealth, error) {
	if pm.healthMonitor == nil {
		return nil, fmt.Errorf("health monitoring not enabled")
	}
	return pm.healthMonitor.GetAllProviderHealth()
}

// GetHealthStats returns overall health monitoring statistics
func (pm *ProviderManager) GetHealthStats() map[string]interface{} {
	if pm.healthMonitor == nil {
		return map[string]interface{}{
			"error": "health monitoring not enabled",
		}
	}
	return pm.healthMonitor.metrics.GetOverallStats()
}

// Global manager instance
var DefaultManager = NewProviderManager(nil)