package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// ValidationRuleSet contains all validation rules for the configuration
type ValidationRuleSet struct {
	rules map[string][]RuleFunc
}

// RuleFunc is a function that validates a specific aspect of the configuration
type RuleFunc func(config map[string]interface{}, result *ValidationResult)

// NewValidationRuleSet creates a new validation rule set
func NewValidationRuleSet() *ValidationRuleSet {
	vrs := &ValidationRuleSet{
		rules: make(map[string][]RuleFunc),
	}
	
	// Register all validation rules
	vrs.registerServerRules()
	vrs.registerProviderRules()
	vrs.registerSecurityRules()
	vrs.registerModerationRules()
	vrs.registerDatabaseRules()
	vrs.registerRoutingRules()
	vrs.registerMetricsRules()
	
	return vrs
}

// Apply applies all validation rules to the configuration
func (vrs *ValidationRuleSet) Apply(config map[string]interface{}, result *ValidationResult) {
	for _, rules := range vrs.rules {
		for _, rule := range rules {
			rule(config, result)
		}
	}
}

// registerServerRules registers validation rules for server configuration
func (vrs *ValidationRuleSet) registerServerRules() {
	vrs.rules["server"] = []RuleFunc{
		// Validate port range
		func(config map[string]interface{}, result *ValidationResult) {
			if server, ok := config["server"].(map[string]interface{}); ok {
				if port, ok := server["port"].(float64); ok {
					if port < 1024 {
						result.AddWarning("server.port", 
							"Port below 1024 requires root privileges", 
							"PRIVILEGED_PORT")
					}
					if port == 80 || port == 443 {
						result.AddWarning("server.port",
							"Using standard HTTP/HTTPS ports may conflict with other services",
							"STANDARD_PORT")
					}
				}
			}
		},
		
		// Validate host format
		func(config map[string]interface{}, result *ValidationResult) {
			if server, ok := config["server"].(map[string]interface{}); ok {
				if host, ok := server["host"].(string); ok {
					// Check if it's a valid hostname or IP
					if net.ParseIP(host) == nil {
						// Not an IP, check if valid hostname
						if strings.Contains(host, " ") || strings.Contains(host, "/") {
							result.AddError("server.host",
								"Invalid host format",
								"INVALID_HOST")
						}
					}
					
					// Warn about listening on all interfaces
					if host == "0.0.0.0" || host == "::" {
						result.AddWarning("server.host",
							"Listening on all interfaces may expose the service externally",
							"EXPOSE_ALL_INTERFACES")
					}
				}
			}
		},
		
		// Validate target URL if present
		func(config map[string]interface{}, result *ValidationResult) {
			if server, ok := config["server"].(map[string]interface{}); ok {
				if targetURL, ok := server["target_url"].(string); ok && targetURL != "" {
					if _, err := url.Parse(targetURL); err != nil {
						result.AddError("server.target_url",
							fmt.Sprintf("Invalid URL format: %v", err),
							"INVALID_URL")
					}
				}
			}
		},
	}
}

// registerProviderRules registers validation rules for provider configuration
func (vrs *ValidationRuleSet) registerProviderRules() {
	vrs.rules["providers"] = []RuleFunc{
		// Validate provider configurations
		func(config map[string]interface{}, result *ValidationResult) {
			providers, ok := config["providers"].(map[string]interface{})
			if !ok {
				return
			}
			
			enabledCount := 0
			for name, providerData := range providers {
				provider, ok := providerData.(map[string]interface{})
				if !ok {
					continue
				}
				
				path := fmt.Sprintf("providers.%s", name)
				
				// Check if enabled
				if enabled, ok := provider["enabled"].(bool); ok && enabled {
					enabledCount++
					
					// Validate API key for cloud providers
					providerType, _ := provider["type"].(string)
					if providerType == "openai" || providerType == "anthropic" {
						apiKey, _ := provider["api_key"].(string)
						if apiKey == "" {
							result.AddError(path+".api_key",
								"API key is required for cloud providers",
								"MISSING_API_KEY")
						}
					}
				}
				
				// Validate base URL
				if baseURL, ok := provider["base_url"].(string); ok {
					if u, err := url.Parse(baseURL); err != nil {
						result.AddError(path+".base_url",
							"Invalid base URL format",
							"INVALID_BASE_URL")
					} else if u.Scheme != "http" && u.Scheme != "https" {
						result.AddError(path+".base_url",
							"Base URL must use HTTP or HTTPS protocol",
							"INVALID_PROTOCOL")
					}
				}
				
				// Validate timeout
				if timeout, ok := provider["timeout"].(string); ok {
					if _, err := time.ParseDuration(timeout); err != nil {
						result.AddError(path+".timeout",
							"Invalid timeout duration format",
							"INVALID_DURATION")
					}
				}
				
				// Validate retry configuration
				if maxRetries, ok := provider["max_retries"].(float64); ok {
					if maxRetries > 10 {
						result.AddWarning(path+".max_retries",
							"High retry count may cause long delays",
							"HIGH_RETRY_COUNT")
					}
				}
			}
			
			// Warn if no providers are enabled
			if enabledCount == 0 {
				result.AddWarning("providers",
					"No providers are enabled",
					"NO_ENABLED_PROVIDERS")
			}
		},
		
		// Validate enhanced providers if present
		func(config map[string]interface{}, result *ValidationResult) {
			enhancedProviders, ok := config["enhanced_providers"].(map[string]interface{})
			if !ok {
				return
			}
			
			for name, providerData := range enhancedProviders {
				provider, ok := providerData.(map[string]interface{})
				if !ok {
					continue
				}
				
				path := fmt.Sprintf("enhanced_providers.%s", name)
				
				// Validate health check configuration
				if healthCheck, ok := provider["health_check"].(map[string]interface{}); ok {
					if interval, ok := healthCheck["interval"].(string); ok {
						if d, err := time.ParseDuration(interval); err != nil {
							result.AddError(path+".health_check.interval",
								"Invalid interval duration",
								"INVALID_DURATION")
						} else if d < 10*time.Second {
							result.AddWarning(path+".health_check.interval",
								"Very frequent health checks may impact performance",
								"FREQUENT_HEALTH_CHECK")
						}
					}
				}
				
				// Validate rate limiting
				if rateLimit, ok := provider["rate_limit"].(map[string]interface{}); ok {
					validateProviderRateLimit(path+".rate_limit", rateLimit, result)
				}
			}
		},
	}
}

// registerSecurityRules registers validation rules for security configuration
func (vrs *ValidationRuleSet) registerSecurityRules() {
	vrs.rules["security"] = []RuleFunc{
		// Validate rate limiting configuration
		func(config map[string]interface{}, result *ValidationResult) {
			security, ok := config["security"].(map[string]interface{})
			if !ok {
				return
			}
			
			if rateLimiting, ok := security["rate_limiting"].(map[string]interface{}); ok {
				if enabled, _ := rateLimiting["enabled"].(bool); enabled {
					// Validate storage configuration
					if storage, ok := rateLimiting["storage"].(map[string]interface{}); ok {
						storageType, _ := storage["type"].(string)
						if storageType == "redis" {
							redisURL, _ := storage["redis_url"].(string)
							if redisURL == "" {
								result.AddError("security.rate_limiting.storage.redis_url",
									"Redis URL is required when storage type is redis",
									"MISSING_REDIS_URL")
							}
						}
					}
					
					// Validate rate limit rules
					validateRateLimitRule("security.rate_limiting.global", rateLimiting["global"], result)
					validateRateLimitRule("security.rate_limiting.per_ip", rateLimiting["per_ip"], result)
					validateRateLimitRule("security.rate_limiting.per_user", rateLimiting["per_user"], result)
				}
			}
		},
		
		// Validate CORS configuration
		func(config map[string]interface{}, result *ValidationResult) {
			security, ok := config["security"].(map[string]interface{})
			if !ok {
				return
			}
			
			if cors, ok := security["cors"].(map[string]interface{}); ok {
				if enabled, _ := cors["enabled"].(bool); enabled {
					// Check allowed origins
					if origins, ok := cors["allowed_origins"].([]interface{}); ok {
						hasWildcard := false
						hasSpecific := false
						
						for _, origin := range origins {
							if originStr, ok := origin.(string); ok {
								if originStr == "*" {
									hasWildcard = true
								} else {
									hasSpecific = true
									// Validate origin format
									if !strings.HasPrefix(originStr, "http://") && 
									   !strings.HasPrefix(originStr, "https://") {
										result.AddError("security.cors.allowed_origins",
											fmt.Sprintf("Origin must include protocol: %s", originStr),
											"INVALID_ORIGIN_FORMAT")
									}
								}
							}
						}
						
						if hasWildcard && hasSpecific {
							result.AddWarning("security.cors.allowed_origins",
								"Using wildcard (*) with specific origins may be confusing",
								"MIXED_ORIGIN_CONFIG")
						}
						
						if hasWildcard {
							result.AddWarning("security.cors.allowed_origins",
								"Allowing all origins (*) may pose security risks",
								"WILDCARD_ORIGIN")
						}
					}
				}
			}
		},
		
		// Validate input validation settings
		func(config map[string]interface{}, result *ValidationResult) {
			security, ok := config["security"].(map[string]interface{})
			if !ok {
				return
			}
			
			if inputVal, ok := security["input_validation"].(map[string]interface{}); ok {
				if enabled, _ := inputVal["enabled"].(bool); enabled {
					// Check if all detection methods are disabled
					detections := []string{
						"detect_xss",
						"detect_sql_injection",
						"detect_command_injection",
						"detect_path_traversal",
					}
					
					allDisabled := true
					for _, detection := range detections {
						if val, _ := inputVal[detection].(bool); val {
							allDisabled = false
							break
						}
					}
					
					if allDisabled {
						result.AddWarning("security.input_validation",
							"All detection methods are disabled",
							"NO_DETECTION_ENABLED")
					}
				}
			}
		},
		
		// Validate DDoS protection settings
		func(config map[string]interface{}, result *ValidationResult) {
			security, ok := config["security"].(map[string]interface{})
			if !ok {
				return
			}
			
			if ddos, ok := security["ddos_protection"].(map[string]interface{}); ok {
				if enabled, _ := ddos["enabled"].(bool); enabled {
					// Validate spike threshold
					if threshold, ok := ddos["spike_threshold"].(float64); ok {
						if threshold < 10 {
							result.AddWarning("security.ddos_protection.spike_threshold",
								"Very low spike threshold may cause false positives",
								"LOW_SPIKE_THRESHOLD")
						}
					}
					
					// Validate circuit breaker settings
					if cbThreshold, ok := ddos["circuit_breaker_threshold"].(float64); ok {
						if cbThreshold < 5 {
							result.AddWarning("security.ddos_protection.circuit_breaker_threshold",
								"Low circuit breaker threshold may interrupt service",
								"LOW_CB_THRESHOLD")
						}
					}
				}
			}
		},
	}
}

// registerModerationRules registers validation rules for moderation configuration
func (vrs *ValidationRuleSet) registerModerationRules() {
	vrs.rules["moderation"] = []RuleFunc{
		// Validate moderation layers
		func(config map[string]interface{}, result *ValidationResult) {
			moderation, ok := config["moderation"].(map[string]interface{})
			if !ok {
				return
			}
			
			if enabled, _ := moderation["enabled"].(bool); enabled {
				// Check if any layers are configured
				layers, _ := moderation["layers"].([]interface{})
				if len(layers) == 0 {
					result.AddWarning("moderation.layers",
						"No moderation layers configured",
						"NO_MODERATION_LAYERS")
				}
				
				// Validate each layer
				for i, layerData := range layers {
					if layer, ok := layerData.(map[string]interface{}); ok {
						path := fmt.Sprintf("moderation.layers[%d]", i)
						
						layerType, _ := layer["type"].(string)
						if layerType == "llm" {
							// Check if provider and model are specified
							provider, _ := layer["provider"].(string)
							model, _ := layer["model"].(string)
							
							if provider == "" {
								result.AddError(path+".provider",
									"Provider is required for LLM moderation layer",
									"MISSING_PROVIDER")
							}
							if model == "" {
								result.AddError(path+".model",
									"Model is required for LLM moderation layer",
									"MISSING_MODEL")
							}
						}
						
						// Validate threshold
						if threshold, ok := layer["threshold"].(float64); ok {
							if threshold < 0 || threshold > 1 {
								result.AddError(path+".threshold",
									"Threshold must be between 0 and 1",
									"INVALID_THRESHOLD")
							}
						}
					}
				}
				
				// Validate severity levels
				severity, _ := moderation["severity"].(string)
				validSeverities := []string{"low", "medium", "high", "strict"}
				validSeverity := false
				for _, valid := range validSeverities {
					if severity == valid {
						validSeverity = true
						break
					}
				}
				if !validSeverity && severity != "" {
					result.AddError("moderation.severity",
						fmt.Sprintf("Invalid severity level. Must be one of: %v", validSeverities),
						"INVALID_SEVERITY")
				}
			}
		},
		
		// Validate advanced moderation if present
		func(config map[string]interface{}, result *ValidationResult) {
			moderation, ok := config["moderation"].(map[string]interface{})
			if !ok {
				return
			}
			
			if advanced, ok := moderation["advanced"].(map[string]interface{}); ok {
				if enabled, _ := advanced["enabled"].(bool); enabled {
					// Validate thresholds
					if thresholds, ok := advanced["thresholds"].(map[string]interface{}); ok {
						validateModerationThresholds("moderation.advanced.thresholds", thresholds, result)
					}
					
					// Validate cache settings
					if cache, ok := advanced["cache"].(map[string]interface{}); ok {
						if cacheEnabled, _ := cache["enabled"].(bool); cacheEnabled {
							if ttl, ok := cache["ttl_minutes"].(float64); ok && ttl < 1 {
								result.AddError("moderation.advanced.cache.ttl_minutes",
									"Cache TTL must be at least 1 minute",
									"INVALID_CACHE_TTL")
							}
						}
					}
				}
			}
		},
	}
}

// registerDatabaseRules registers validation rules for database configuration
func (vrs *ValidationRuleSet) registerDatabaseRules() {
	vrs.rules["database"] = []RuleFunc{
		// Validate database connection settings
		func(config map[string]interface{}, result *ValidationResult) {
			database, ok := config["database"].(map[string]interface{})
			if !ok {
				return
			}
			
			dbType, _ := database["type"].(string)
			connectionString, _ := database["connection_string"].(string)
			
			// If connection string is provided, other settings are optional
			if connectionString != "" {
				return
			}
			
			// Validate based on database type
			switch dbType {
			case "postgresql":
				// PostgreSQL requires connection details
				requiredFields := map[string]string{
					"host":     "Database host",
					"port":     "Database port",
					"name":     "Database name",
					"username": "Database username",
				}
				
				for field, desc := range requiredFields {
					if value, _ := database[field].(string); value == "" {
						if field == "port" {
							// Port might be a number
							if port, _ := database[field].(float64); port == 0 {
								result.AddError(fmt.Sprintf("database.%s", field),
									fmt.Sprintf("%s is required for PostgreSQL", desc),
									"REQUIRED_FIELD")
							}
						} else {
							result.AddError(fmt.Sprintf("database.%s", field),
								fmt.Sprintf("%s is required for PostgreSQL", desc),
								"REQUIRED_FIELD")
						}
					}
				}
				
			case "sqlite":
				// SQLite requires file path
				if sqliteFile, _ := database["sqlite_file"].(string); sqliteFile == "" {
					result.AddError("database.sqlite_file",
						"SQLite file path is required",
						"REQUIRED_FIELD")
				}
			}
			
			// Validate connection pool settings
			if maxConn, ok := database["max_connections"].(float64); ok {
				if maxConn < 1 {
					result.AddError("database.max_connections",
						"Maximum connections must be at least 1",
						"INVALID_CONNECTION_LIMIT")
				} else if maxConn > 1000 {
					result.AddWarning("database.max_connections",
						"Very high connection limit may exhaust resources",
						"HIGH_CONNECTION_LIMIT")
				}
			}
			
			// Validate retention policies
			if retention, ok := database["retention"].(map[string]interface{}); ok {
				validateRetentionPolicy("database.retention", retention, result)
			}
		},
	}
}

// registerRoutingRules registers validation rules for routing configuration
func (vrs *ValidationRuleSet) registerRoutingRules() {
	vrs.rules["routing"] = []RuleFunc{
		// Validate routing configuration
		func(config map[string]interface{}, result *ValidationResult) {
			routing, ok := config["routing"].(map[string]interface{})
			if !ok {
				return
			}
			
			defaultProvider, _ := routing["default_provider"].(string)
			providers, _ := config["providers"].(map[string]interface{})
			enhancedProviders, _ := config["enhanced_providers"].(map[string]interface{})
			
			// Check if default provider exists
			if defaultProvider != "" {
				found := false
				if providers != nil {
					if _, ok := providers[defaultProvider]; ok {
						found = true
					}
				}
				if !found && enhancedProviders != nil {
					if _, ok := enhancedProviders[defaultProvider]; ok {
						found = true
					}
				}
				
				if !found {
					result.AddError("routing.default_provider",
						fmt.Sprintf("Default provider '%s' not found in providers", defaultProvider),
						"PROVIDER_NOT_FOUND")
				}
			}
			
			// Validate model routing
			if modelRouting, ok := routing["model_routing"].(map[string]interface{}); ok {
				for model, providerName := range modelRouting {
					if provider, ok := providerName.(string); ok {
						// Check if routed provider exists
						found := false
						if providers != nil {
							if _, ok := providers[provider]; ok {
								found = true
							}
						}
						if !found && enhancedProviders != nil {
							if _, ok := enhancedProviders[provider]; ok {
								found = true
							}
						}
						
						if !found {
							result.AddError(fmt.Sprintf("routing.model_routing.%s", model),
								fmt.Sprintf("Provider '%s' not found", provider),
								"PROVIDER_NOT_FOUND")
						}
					}
				}
			}
			
			// Validate fallback chain
			if fallbackChain, ok := routing["fallback_chain"].([]interface{}); ok {
				seen := make(map[string]bool)
				for i, providerName := range fallbackChain {
					if provider, ok := providerName.(string); ok {
						// Check for duplicates
						if seen[provider] {
							result.AddWarning(fmt.Sprintf("routing.fallback_chain[%d]", i),
								fmt.Sprintf("Duplicate provider '%s' in fallback chain", provider),
								"DUPLICATE_FALLBACK")
						}
						seen[provider] = true
						
						// Check if provider exists
						found := false
						if providers != nil {
							if _, ok := providers[provider]; ok {
								found = true
							}
						}
						if !found && enhancedProviders != nil {
							if _, ok := enhancedProviders[provider]; ok {
								found = true
							}
						}
						
						if !found {
							result.AddError(fmt.Sprintf("routing.fallback_chain[%d]", i),
								fmt.Sprintf("Provider '%s' not found", provider),
								"PROVIDER_NOT_FOUND")
						}
					}
				}
			}
		},
	}
}

// registerMetricsRules registers validation rules for metrics configuration
func (vrs *ValidationRuleSet) registerMetricsRules() {
	vrs.rules["metrics"] = []RuleFunc{
		// Validate metrics configuration
		func(config map[string]interface{}, result *ValidationResult) {
			metrics, ok := config["metrics"].(map[string]interface{})
			if !ok {
				return
			}
			
			if enabled, _ := metrics["enabled"].(bool); enabled {
				// Validate storage configuration
				if storage, ok := metrics["storage"].(map[string]interface{}); ok {
					storageType, _ := storage["type"].(string)
					
					if storageType == "" {
						result.AddError("metrics.storage.type",
							"Storage type is required when metrics are enabled",
							"REQUIRED_FIELD")
					}
					
					// Validate retention
					if retention, ok := storage["retention_hours"].(float64); ok {
						if retention < 1 {
							result.AddError("metrics.storage.retention_hours",
								"Retention must be at least 1 hour",
								"INVALID_RETENTION")
						} else if retention > 8760 { // 1 year
							result.AddWarning("metrics.storage.retention_hours",
								"Very long retention period may consume significant storage",
								"HIGH_RETENTION")
						}
					}
					
					// Validate memory limits
					if maxMemory, ok := storage["max_memory_mb"].(float64); ok {
						if maxMemory < 10 {
							result.AddWarning("metrics.storage.max_memory_mb",
								"Very low memory limit may cause frequent evictions",
								"LOW_MEMORY_LIMIT")
						}
					}
				}
				
				// Validate Supabase configuration if present
				if supabase, ok := metrics["supabase"].(map[string]interface{}); ok {
					url, _ := supabase["url"].(string)
					key, _ := supabase["key"].(string)
					
					if url != "" && key == "" {
						result.AddError("metrics.supabase.key",
							"Supabase key is required when URL is provided",
							"MISSING_SUPABASE_KEY")
					}
					if key != "" && url == "" {
						result.AddError("metrics.supabase.url",
							"Supabase URL is required when key is provided",
							"MISSING_SUPABASE_URL")
					}
				}
			}
		},
	}
}

// Helper functions

func validateRateLimitRule(path string, ruleData interface{}, result *ValidationResult) {
	rule, ok := ruleData.(map[string]interface{})
	if !ok {
		return
	}
	
	// Check if at least one limit is set
	hasLimit := false
	limitFields := []string{"requests_per_second", "requests_per_minute", "requests_per_hour"}
	
	for _, field := range limitFields {
		if val, ok := rule[field].(float64); ok && val > 0 {
			hasLimit = true
			
			// Validate reasonable limits
			switch field {
			case "requests_per_second":
				if val > 1000 {
					result.AddWarning(path+"."+field,
						"Very high rate limit may not provide effective protection",
						"HIGH_RATE_LIMIT")
				}
			case "requests_per_minute":
				if val > 60000 {
					result.AddWarning(path+"."+field,
						"Very high rate limit may not provide effective protection",
						"HIGH_RATE_LIMIT")
				}
			}
		}
	}
	
	if !hasLimit {
		result.AddWarning(path,
			"No rate limits configured",
			"NO_RATE_LIMITS")
	}
	
	// Validate window size if present
	if windowSize, ok := rule["window_size"].(string); ok {
		if _, err := time.ParseDuration(windowSize); err != nil {
			result.AddError(path+".window_size",
				"Invalid window size duration",
				"INVALID_DURATION")
		}
	}
	
	// Validate algorithm
	if algorithm, ok := rule["algorithm"].(string); ok {
		validAlgorithms := []string{"token_bucket", "sliding_window"}
		valid := false
		for _, validAlgo := range validAlgorithms {
			if algorithm == validAlgo {
				valid = true
				break
			}
		}
		if !valid {
			result.AddError(path+".algorithm",
				fmt.Sprintf("Invalid algorithm. Must be one of: %v", validAlgorithms),
				"INVALID_ALGORITHM")
		}
	}
}

func validateProviderRateLimit(path string, rateLimit map[string]interface{}, result *ValidationResult) {
	// Validate requests per second
	if rps, ok := rateLimit["requests_per_second"].(float64); ok {
		if rps <= 0 {
			result.AddError(path+".requests_per_second",
				"Requests per second must be positive",
				"INVALID_RATE_LIMIT")
		}
	}
	
	// Validate token limits
	if tpm, ok := rateLimit["tokens_per_minute"].(float64); ok {
		if tpm <= 0 {
			result.AddError(path+".tokens_per_minute",
				"Tokens per minute must be positive",
				"INVALID_TOKEN_LIMIT")
		}
	}
}

func validateModerationThresholds(path string, thresholds map[string]interface{}, result *ValidationResult) {
	levels := []string{"low", "medium", "high", "critical"}
	values := make(map[string]float64)
	
	// Extract threshold values
	for _, level := range levels {
		if val, ok := thresholds[level].(float64); ok {
			if val < 0 || val > 1 {
				result.AddError(fmt.Sprintf("%s.%s", path, level),
					"Threshold must be between 0 and 1",
					"INVALID_THRESHOLD")
			}
			values[level] = val
		}
	}
	
	// Ensure thresholds are in ascending order
	if len(values) > 1 {
		if low, ok := values["low"]; ok {
			if medium, ok := values["medium"]; ok && medium < low {
				result.AddError(path,
					"Medium threshold should be higher than low threshold",
					"THRESHOLD_ORDER")
			}
			if high, ok := values["high"]; ok && high < low {
				result.AddError(path,
					"High threshold should be higher than low threshold",
					"THRESHOLD_ORDER")
			}
		}
		if medium, ok := values["medium"]; ok {
			if high, ok := values["high"]; ok && high < medium {
				result.AddError(path,
					"High threshold should be higher than medium threshold",
					"THRESHOLD_ORDER")
			}
		}
	}
}

func validateRetentionPolicy(path string, retention map[string]interface{}, result *ValidationResult) {
	retentionFields := map[string]string{
		"requests": "Request retention",
		"metrics":  "Metrics retention",
		"logs":     "Logs retention",
	}
	
	for field, desc := range retentionFields {
		if retentionStr, ok := retention[field].(string); ok {
			// Validate duration format (e.g., "30d", "7d", "90d")
			if !isValidRetentionFormat(retentionStr) {
				result.AddError(fmt.Sprintf("%s.%s", path, field),
					fmt.Sprintf("%s must be in format like '30d' or '7d'", desc),
					"INVALID_RETENTION_FORMAT")
			}
		}
	}
}

func isValidRetentionFormat(retention string) bool {
	// Simple validation for retention format
	if len(retention) < 2 {
		return false
	}
	
	// Check if it ends with a valid unit
	validUnits := []string{"d", "w", "m", "y"}
	unit := retention[len(retention)-1:]
	validUnit := false
	for _, valid := range validUnits {
		if unit == valid {
			validUnit = true
			break
		}
	}
	
	if !validUnit {
		return false
	}
	
	// Check if the number part is valid
	numberPart := retention[:len(retention)-1]
	for _, char := range numberPart {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}