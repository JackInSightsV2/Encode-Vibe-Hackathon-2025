package moderation

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigManager handles loading and validation of moderation configuration
type ConfigManager struct {
	config     *AdvancedModerationConfig
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig loads the moderation configuration from file
func (cm *ConfigManager) LoadConfig() (*AdvancedModerationConfig, error) {
	// Set default configuration
	config := cm.getDefaultConfig()
	
	// If config file exists, merge with defaults
	if cm.configPath != "" && cm.fileExists(cm.configPath) {
		fileConfig, err := cm.loadFromFile(cm.configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from file: %v", err)
		}
		
		// Merge file config with defaults
		config = cm.mergeConfigs(config, fileConfig)
	}
	
	// Validate configuration
	if err := cm.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}
	
	cm.config = config
	return config, nil
}

// getDefaultConfig returns the default moderation configuration
func (cm *ConfigManager) getDefaultConfig() *AdvancedModerationConfig {
	return &AdvancedModerationConfig{
		Enabled: false, // Disabled by default for safety
		Layers: []LayerConfig{
			{
				Name:      "regex",
				Enabled:   true,
				Weight:    0.3,
				Threshold: 0.8,
				Options:   make(map[string]interface{}),
			},
		},
		Thresholds: ModerationThresholds{
			Low:      0.3,
			Medium:   0.6,
			High:     0.8,
			Critical: 0.95,
		},
		Actions: ActionConfig{
			Low:      ActionLog,
			Medium:   ActionFlag,
			High:     ActionBlock,
			Critical: ActionBlockAlert,
		},
		Cache: CacheConfig{
			Enabled:    true,
			TTLMinutes: 60,
			MaxEntries: 10000,
		},
		Analytics: AnalyticsConfig{
			Enabled:           true,
			CollectDetails:    false,
			RetentionDays:     30,
			EnablePerformance: true,
		},
	}
}

// loadFromFile loads configuration from a YAML file
func (cm *ConfigManager) loadFromFile(path string) (*AdvancedModerationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var config AdvancedModerationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// mergeConfigs merges file configuration with defaults
func (cm *ConfigManager) mergeConfigs(defaults, fileConfig *AdvancedModerationConfig) *AdvancedModerationConfig {
	result := *defaults
	
	// Merge top-level settings
	if fileConfig.Enabled != defaults.Enabled {
		result.Enabled = fileConfig.Enabled
	}
	
	// Merge layers
	if len(fileConfig.Layers) > 0 {
		result.Layers = cm.mergeLayers(defaults.Layers, fileConfig.Layers)
	}
	
	// Merge thresholds
	if fileConfig.Thresholds.Low != 0 {
		result.Thresholds.Low = fileConfig.Thresholds.Low
	}
	if fileConfig.Thresholds.Medium != 0 {
		result.Thresholds.Medium = fileConfig.Thresholds.Medium
	}
	if fileConfig.Thresholds.High != 0 {
		result.Thresholds.High = fileConfig.Thresholds.High
	}
	if fileConfig.Thresholds.Critical != 0 {
		result.Thresholds.Critical = fileConfig.Thresholds.Critical
	}
	
	// Merge actions
	if fileConfig.Actions.Low != "" {
		result.Actions.Low = fileConfig.Actions.Low
	}
	if fileConfig.Actions.Medium != "" {
		result.Actions.Medium = fileConfig.Actions.Medium
	}
	if fileConfig.Actions.High != "" {
		result.Actions.High = fileConfig.Actions.High
	}
	if fileConfig.Actions.Critical != "" {
		result.Actions.Critical = fileConfig.Actions.Critical
	}
	
	// Merge cache config
	if fileConfig.Cache.TTLMinutes != 0 {
		result.Cache.TTLMinutes = fileConfig.Cache.TTLMinutes
	}
	if fileConfig.Cache.MaxEntries != 0 {
		result.Cache.MaxEntries = fileConfig.Cache.MaxEntries
	}
	result.Cache.Enabled = fileConfig.Cache.Enabled
	
	// Merge analytics config
	result.Analytics.Enabled = fileConfig.Analytics.Enabled
	result.Analytics.CollectDetails = fileConfig.Analytics.CollectDetails
	if fileConfig.Analytics.RetentionDays != 0 {
		result.Analytics.RetentionDays = fileConfig.Analytics.RetentionDays
	}
	result.Analytics.EnablePerformance = fileConfig.Analytics.EnablePerformance
	
	return &result
}

// mergeLayers merges layer configurations
func (cm *ConfigManager) mergeLayers(defaults, fileConfig []LayerConfig) []LayerConfig {
	layerMap := make(map[string]LayerConfig)
	
	// Start with defaults
	for _, layer := range defaults {
		layerMap[layer.Name] = layer
	}
	
	// Override with file config
	for _, layer := range fileConfig {
		if existing, exists := layerMap[layer.Name]; exists {
			// Merge with existing
			merged := existing
			merged.Enabled = layer.Enabled
			if layer.Weight != 0 {
				merged.Weight = layer.Weight
			}
			if layer.Threshold != 0 {
				merged.Threshold = layer.Threshold
			}
			if layer.Options != nil {
				merged.Options = layer.Options
			}
			layerMap[layer.Name] = merged
		} else {
			// Add new layer
			layerMap[layer.Name] = layer
		}
	}
	
	// Convert back to slice
	var result []LayerConfig
	for _, layer := range layerMap {
		result = append(result, layer)
	}
	
	return result
}

// validateConfig validates the configuration for consistency and correctness
func (cm *ConfigManager) validateConfig(config *AdvancedModerationConfig) error {
	// Validate thresholds are in ascending order
	thresholds := []float64{
		config.Thresholds.Low,
		config.Thresholds.Medium,
		config.Thresholds.High,
		config.Thresholds.Critical,
	}
	
	for i := 0; i < len(thresholds)-1; i++ {
		if thresholds[i] >= thresholds[i+1] {
			return fmt.Errorf("thresholds must be in ascending order: %v", thresholds)
		}
		if thresholds[i] < 0 || thresholds[i] > 1 {
			return fmt.Errorf("thresholds must be between 0 and 1: %f", thresholds[i])
		}
	}
	
	// Validate layer weights
	totalWeight := 0.0
	enabledLayers := 0
	
	for _, layer := range config.Layers {
		if layer.Enabled {
			enabledLayers++
			if layer.Weight < 0 || layer.Weight > 1 {
				return fmt.Errorf("layer %s weight must be between 0 and 1: %f", layer.Name, layer.Weight)
			}
			totalWeight += layer.Weight
		}
	}
	
	if enabledLayers == 0 {
		return fmt.Errorf("at least one moderation layer must be enabled")
	}
	
	if totalWeight > 1.1 { // Allow slight tolerance for floating point
		return fmt.Errorf("total layer weights cannot exceed 1.0: %f", totalWeight)
	}
	
	// Validate actions
	validActions := map[string]bool{
		ActionLog:        true,
		ActionFlag:       true,
		ActionBlock:      true,
		ActionBlockAlert: true,
	}
	
	actions := []string{
		config.Actions.Low,
		config.Actions.Medium,
		config.Actions.High,
		config.Actions.Critical,
	}
	
	for _, action := range actions {
		if !validActions[action] {
			return fmt.Errorf("invalid action: %s", action)
		}
	}
	
	// Validate cache config
	if config.Cache.Enabled {
		if config.Cache.TTLMinutes <= 0 {
			return fmt.Errorf("cache TTL must be positive: %d", config.Cache.TTLMinutes)
		}
		if config.Cache.MaxEntries <= 0 {
			return fmt.Errorf("cache max entries must be positive: %d", config.Cache.MaxEntries)
		}
	}
	
	// Validate analytics config
	if config.Analytics.Enabled && config.Analytics.RetentionDays <= 0 {
		return fmt.Errorf("analytics retention days must be positive: %d", config.Analytics.RetentionDays)
	}
	
	return nil
}

// SaveConfig saves the current configuration to file
func (cm *ConfigManager) SaveConfig(config *AdvancedModerationConfig) error {
	if cm.configPath == "" {
		return fmt.Errorf("no config path specified")
	}
	
	// Validate before saving
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	cm.config = config
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *AdvancedModerationConfig {
	return cm.config
}

// fileExists checks if a file exists
func (cm *ConfigManager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetLayerConfig returns configuration for a specific layer
func (cm *ConfigManager) GetLayerConfig(layerName string) *LayerConfig {
	if cm.config == nil {
		return nil
	}
	
	for _, layer := range cm.config.Layers {
		if layer.Name == layerName {
			return &layer
		}
	}
	
	return nil
}

// IsLayerEnabled checks if a specific layer is enabled
func (cm *ConfigManager) IsLayerEnabled(layerName string) bool {
	config := cm.GetLayerConfig(layerName)
	return config != nil && config.Enabled
}