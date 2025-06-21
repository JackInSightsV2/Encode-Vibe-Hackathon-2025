package opik

import (
	"time"
)

// DefaultConfig returns the default Opik configuration
func DefaultConfig() OpikConfig {
	return OpikConfig{
		Enabled:       false,
		BaseURL:       "https://api.opik.com",
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}
}

// ValidateConfig validates the Opik configuration
func ValidateConfig(config OpikConfig) error {
	if config.Enabled && config.APIKey == "" {
		return ErrMissingAPIKey
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}

	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	return nil
}