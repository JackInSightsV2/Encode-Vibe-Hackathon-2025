package providers

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ProviderRegistry manages all registered providers with thread-safe operations
type ProviderRegistry struct {
	providers    map[string]Provider
	configs      map[string]ProviderConfig
	dependencies map[string][]string // provider -> dependencies
	watchers     []RegistryWatcher
	mutex        sync.RWMutex
	started      bool
}

// RegistryWatcher is notified when provider registry changes occur
type RegistryWatcher interface {
	OnProviderRegistered(name string, provider Provider) error
	OnProviderUnregistered(name string) error
	OnProviderConfigured(name string, config ProviderConfig) error
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers:    make(map[string]Provider),
		configs:      make(map[string]ProviderConfig),
		dependencies: make(map[string][]string),
		watchers:     make([]RegistryWatcher, 0),
		started:      false,
	}
}

// Register adds a new provider to the registry
func (pr *ProviderRegistry) Register(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	// Check if provider already exists
	if _, exists := pr.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	// Validate the provider
	if err := provider.Validate(); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	// Add to registry
	pr.providers[name] = provider

	// Notify watchers
	for _, watcher := range pr.watchers {
		if err := watcher.OnProviderRegistered(name, provider); err != nil {
			// Log error but don't fail registration
			fmt.Printf("Registry watcher error on register: %v\n", err)
		}
	}

	return nil
}

// RegisterWithConfig registers a provider with its configuration
func (pr *ProviderRegistry) RegisterWithConfig(provider Provider, config ProviderConfig) error {
	if err := pr.Register(provider); err != nil {
		return err
	}

	return pr.Configure(provider.Name(), config)
}

// Unregister removes a provider from the registry
func (pr *ProviderRegistry) Unregister(name string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	provider, exists := pr.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Check for dependents
	dependents := pr.getDependents(name)
	if len(dependents) > 0 {
		return fmt.Errorf("cannot unregister provider %s: has dependents: %v", name, dependents)
	}

	// Stop the provider if it's running
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.Stop(ctx); err != nil {
		fmt.Printf("Warning: failed to stop provider %s during unregistration: %v\n", name, err)
	}

	// Remove from registry
	delete(pr.providers, name)
	delete(pr.configs, name)
	delete(pr.dependencies, name)

	// Notify watchers
	for _, watcher := range pr.watchers {
		if err := watcher.OnProviderUnregistered(name); err != nil {
			fmt.Printf("Registry watcher error on unregister: %v\n", err)
		}
	}

	return nil
}

// Get retrieves a provider by name
func (pr *ProviderRegistry) Get(name string) (Provider, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	provider, exists := pr.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GetConfig retrieves a provider's configuration
func (pr *ProviderRegistry) GetConfig(name string) (ProviderConfig, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	config, exists := pr.configs[name]
	if !exists {
		return ProviderConfig{}, fmt.Errorf("configuration for provider %s not found", name)
	}

	return config, nil
}

// List returns all registered providers
func (pr *ProviderRegistry) List() []Provider {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	providers := make([]Provider, 0, len(pr.providers))
	for _, provider := range pr.providers {
		providers = append(providers, provider)
	}

	return providers
}

// ListNames returns names of all registered providers
func (pr *ProviderRegistry) ListNames() []string {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	names := make([]string, 0, len(pr.providers))
	for name := range pr.providers {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// ListByType returns providers of a specific type
func (pr *ProviderRegistry) ListByType(providerType ProviderType) []Provider {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	providers := make([]Provider, 0)
	for _, provider := range pr.providers {
		if provider.Type() == providerType {
			providers = append(providers, provider)
		}
	}

	return providers
}

// ListEnabled returns only enabled providers
func (pr *ProviderRegistry) ListEnabled() []Provider {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	providers := make([]Provider, 0)
	for name, provider := range pr.providers {
		if config, exists := pr.configs[name]; exists && config.Enabled {
			providers = append(providers, provider)
		}
	}

	return providers
}

// Configure sets the configuration for a provider
func (pr *ProviderRegistry) Configure(name string, config ProviderConfig) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	provider, exists := pr.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Validate configuration
	if err := pr.validateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check dependencies
	if err := pr.checkDependencies(config); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	// Apply configuration to provider
	if err := provider.Configure(config); err != nil {
		return fmt.Errorf("provider rejected configuration: %w", err)
	}

	// Store configuration
	pr.configs[name] = config
	pr.dependencies[name] = config.Dependencies

	// Notify watchers
	for _, watcher := range pr.watchers {
		if err := watcher.OnProviderConfigured(name, config); err != nil {
			fmt.Printf("Registry watcher error on configure: %v\n", err)
		}
	}

	return nil
}

// StartAll starts all enabled providers in dependency order
func (pr *ProviderRegistry) StartAll(ctx context.Context) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if pr.started {
		return fmt.Errorf("registry already started")
	}

	// Get enabled providers in dependency order
	startOrder, err := pr.getStartOrder()
	if err != nil {
		return fmt.Errorf("failed to determine start order: %w", err)
	}

	// Start providers
	for _, name := range startOrder {
		provider := pr.providers[name]
		config := pr.configs[name]

		if !config.Enabled {
			continue
		}

		fmt.Printf("Starting provider: %s\n", name)
		if err := provider.Start(ctx); err != nil {
			return fmt.Errorf("failed to start provider %s: %w", name, err)
		}
	}

	pr.started = true
	return nil
}

// StopAll stops all providers in reverse dependency order
func (pr *ProviderRegistry) StopAll(ctx context.Context) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if !pr.started {
		return nil // Already stopped
	}

	// Get start order and reverse it for stop order
	startOrder, err := pr.getStartOrder()
	if err != nil {
		return fmt.Errorf("failed to determine stop order: %w", err)
	}

	// Reverse the order for stopping
	for i := len(startOrder) - 1; i >= 0; i-- {
		name := startOrder[i]
		provider := pr.providers[name]

		fmt.Printf("Stopping provider: %s\n", name)
		if err := provider.Stop(ctx); err != nil {
			fmt.Printf("Warning: failed to stop provider %s: %v\n", name, err)
		}
	}

	pr.started = false
	return nil
}

// AddWatcher adds a registry watcher
func (pr *ProviderRegistry) AddWatcher(watcher RegistryWatcher) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	pr.watchers = append(pr.watchers, watcher)
}

// RemoveWatcher removes a registry watcher
func (pr *ProviderRegistry) RemoveWatcher(watcher RegistryWatcher) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for i, w := range pr.watchers {
		if w == watcher {
			pr.watchers = append(pr.watchers[:i], pr.watchers[i+1:]...)
			break
		}
	}
}

// GetStatus returns the status of all providers
func (pr *ProviderRegistry) GetStatus() map[string]ProviderStatus {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	status := make(map[string]ProviderStatus)
	for name, provider := range pr.providers {
		status[name] = provider.GetStatus()
	}

	return status
}

// GetMetrics returns the metrics of all providers
func (pr *ProviderRegistry) GetMetrics() map[string]ProviderMetrics {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	metrics := make(map[string]ProviderMetrics)
	for name, provider := range pr.providers {
		metrics[name] = provider.GetMetrics()
	}

	return metrics
}

// validateConfig validates a provider configuration
func (pr *ProviderRegistry) validateConfig(config ProviderConfig) error {
	if config.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if config.Type == "" {
		return fmt.Errorf("provider type cannot be empty")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if config.Priority < 0 {
		return fmt.Errorf("priority cannot be negative")
	}

	return nil
}

// checkDependencies validates that all dependencies exist and are enabled
func (pr *ProviderRegistry) checkDependencies(config ProviderConfig) error {
	for _, dep := range config.Dependencies {
		if _, exists := pr.providers[dep]; !exists {
			return fmt.Errorf("dependency %s not found", dep)
		}

		if depConfig, exists := pr.configs[dep]; exists && !depConfig.Enabled {
			return fmt.Errorf("dependency %s is not enabled", dep)
		}
	}

	return nil
}

// getDependents returns providers that depend on the given provider
func (pr *ProviderRegistry) getDependents(name string) []string {
	dependents := make([]string, 0)
	for providerName, deps := range pr.dependencies {
		for _, dep := range deps {
			if dep == name {
				dependents = append(dependents, providerName)
				break
			}
		}
	}
	return dependents
}

// getStartOrder determines the order to start providers based on dependencies
func (pr *ProviderRegistry) getStartOrder() ([]string, error) {
	// Topological sort to handle dependencies
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	order := make([]string, 0)

	var visit func(name string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving %s", name)
		}
		if visited[name] {
			return nil
		}

		visiting[name] = true

		// Visit dependencies first
		if deps, exists := pr.dependencies[name]; exists {
			for _, dep := range deps {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		visiting[name] = false
		visited[name] = true
		order = append(order, name)

		return nil
	}

	// Visit all providers
	for name := range pr.providers {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// GetProvidersByPriority returns providers sorted by priority (highest first)
func (pr *ProviderRegistry) GetProvidersByPriority() []Provider {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	type providerWithPriority struct {
		provider Provider
		priority int
	}

	providers := make([]providerWithPriority, 0)
	for name, provider := range pr.providers {
		priority := 0
		if config, exists := pr.configs[name]; exists {
			priority = config.Priority
		}
		providers = append(providers, providerWithPriority{
			provider: provider,
			priority: priority,
		})
	}

	// Sort by priority (highest first)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].priority > providers[j].priority
	})

	result := make([]Provider, len(providers))
	for i, p := range providers {
		result[i] = p.provider
	}

	return result
}

// IsStarted returns whether the registry has been started
func (pr *ProviderRegistry) IsStarted() bool {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return pr.started
}

// Count returns the number of registered providers
func (pr *ProviderRegistry) Count() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return len(pr.providers)
}

// CountEnabled returns the number of enabled providers
func (pr *ProviderRegistry) CountEnabled() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	count := 0
	for name := range pr.providers {
		if config, exists := pr.configs[name]; exists && config.Enabled {
			count++
		}
	}
	return count
}