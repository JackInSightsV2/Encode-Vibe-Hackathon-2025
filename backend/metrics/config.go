package metrics

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DefaultMetricsConfig returns the default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled: true,
		Storage: StorageConfig{
			Type:                   "memory",
			RetentionHours:         24,
			CleanupIntervalMinutes: 60,
			MaxMemoryMB:           100,
		},
		Supabase: SupabaseConfig{
			URL:   "",
			Key:   "",
			Table: "qt1_metrics",
		},
		Collection: CollectionConfig{
			HTTPRequests:     true,
			ModerationEvents: true,
			SystemHealth:     true,
			PIIDetection:     true,
		},
		TimeSeries: TimeSeriesConfig{
			Enabled:                   true,
			AggregationIntervalSecs:  60, // 1 minute
			MaxBucketsPerResolution:  1000,
			Resolutions: []ResolutionConfig{
				{Resolution: "1m", RetentionHours: 24},    // 1 day of 1-minute data
				{Resolution: "5m", RetentionHours: 72},    // 3 days of 5-minute data
				{Resolution: "1h", RetentionHours: 720},   // 30 days of 1-hour data
				{Resolution: "1d", RetentionHours: 8760},  // 1 year of 1-day data
			},
		},
		SystemMetrics: SystemMetricsConfig{
			Enabled:            true,
			CollectionInterval: 30 * time.Second,
			CPUMonitoring:      true,
			MemoryMonitoring:   true,
			ProviderHealth:     []string{"openai", "anthropic", "cohere"},
		},
	}
}

// LoadConfigFromEnv loads metrics configuration from environment variables
func LoadConfigFromEnv() MetricsConfig {
	config := DefaultMetricsConfig()
	
	// Override with environment variables
	if enabled := os.Getenv("METRICS_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			config.Enabled = val
		}
	}
	
	if storageType := os.Getenv("METRICS_STORAGE_TYPE"); storageType != "" {
		config.Storage.Type = storageType
	}
	
	if retention := os.Getenv("METRICS_RETENTION_HOURS"); retention != "" {
		if val, err := strconv.Atoi(retention); err == nil {
			config.Storage.RetentionHours = val
		}
	}
	
	if cleanup := os.Getenv("METRICS_CLEANUP_INTERVAL_MINUTES"); cleanup != "" {
		if val, err := strconv.Atoi(cleanup); err == nil {
			config.Storage.CleanupIntervalMinutes = val
		}
	}
	
	if memory := os.Getenv("METRICS_MAX_MEMORY_MB"); memory != "" {
		if val, err := strconv.Atoi(memory); err == nil {
			config.Storage.MaxMemoryMB = val
		}
	}
	
	// Supabase configuration
	if url := os.Getenv("SUPABASE_URL"); url != "" {
		config.Supabase.URL = url
	}
	
	if key := os.Getenv("SUPABASE_ANON_KEY"); key != "" {
		config.Supabase.Key = key
	}
	
	if table := os.Getenv("METRICS_SUPABASE_TABLE"); table != "" {
		config.Supabase.Table = table
	}
	
	// Collection settings
	if httpReq := os.Getenv("METRICS_COLLECT_HTTP"); httpReq != "" {
		if val, err := strconv.ParseBool(httpReq); err == nil {
			config.Collection.HTTPRequests = val
		}
	}
	
	if modEvents := os.Getenv("METRICS_COLLECT_MODERATION"); modEvents != "" {
		if val, err := strconv.ParseBool(modEvents); err == nil {
			config.Collection.ModerationEvents = val
		}
	}
	
	if sysHealth := os.Getenv("METRICS_COLLECT_SYSTEM"); sysHealth != "" {
		if val, err := strconv.ParseBool(sysHealth); err == nil {
			config.Collection.SystemHealth = val
		}
	}
	
	if pii := os.Getenv("METRICS_COLLECT_PII"); pii != "" {
		if val, err := strconv.ParseBool(pii); err == nil {
			config.Collection.PIIDetection = val
		}
	}
	
	return config
}

// ValidateConfig validates the metrics configuration
func ValidateConfig(config MetricsConfig) error {
	if config.Storage.Type != "memory" && config.Storage.Type != "supabase" {
		return fmt.Errorf("unsupported storage type: %s", config.Storage.Type)
	}
	
	if config.Storage.Type == "supabase" {
		if config.Supabase.URL == "" {
			return fmt.Errorf("Supabase URL is required when using supabase storage")
		}
		if config.Supabase.Key == "" {
			return fmt.Errorf("Supabase key is required when using supabase storage")
		}
		if config.Supabase.Table == "" {
			return fmt.Errorf("Supabase table name is required when using supabase storage")
		}
	}
	
	if config.Storage.RetentionHours < 0 {
		return fmt.Errorf("retention hours cannot be negative")
	}
	
	if config.Storage.CleanupIntervalMinutes < 0 {
		return fmt.Errorf("cleanup interval cannot be negative")
	}
	
	if config.Storage.MaxMemoryMB < 0 {
		return fmt.Errorf("max memory MB cannot be negative")
	}
	
	return nil
}

// MergeConfigs merges two configurations, with the second taking precedence
func MergeConfigs(base, override MetricsConfig) MetricsConfig {
	merged := base
	
	// Override enabled setting
	merged.Enabled = override.Enabled
	
	// Override storage settings
	if override.Storage.Type != "" {
		merged.Storage.Type = override.Storage.Type
	}
	if override.Storage.RetentionHours > 0 {
		merged.Storage.RetentionHours = override.Storage.RetentionHours
	}
	if override.Storage.CleanupIntervalMinutes > 0 {
		merged.Storage.CleanupIntervalMinutes = override.Storage.CleanupIntervalMinutes
	}
	if override.Storage.MaxMemoryMB > 0 {
		merged.Storage.MaxMemoryMB = override.Storage.MaxMemoryMB
	}
	
	// Override Supabase settings
	if override.Supabase.URL != "" {
		merged.Supabase.URL = override.Supabase.URL
	}
	if override.Supabase.Key != "" {
		merged.Supabase.Key = override.Supabase.Key
	}
	if override.Supabase.Table != "" {
		merged.Supabase.Table = override.Supabase.Table
	}
	
	// Override collection settings
	merged.Collection = override.Collection
	
	return merged
}