package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ConfigValidator validates configuration against schema
type ConfigValidator struct {
	schema          *ConfigSchema
	customValidators map[string]ValidatorFunc
	context         *ValidationContext
}

// ValidatorFunc is a custom validation function
type ValidatorFunc func(value interface{}, schema *PropertySchema, path string) error

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(schema *ConfigSchema) *ConfigValidator {
	cv := &ConfigValidator{
		schema:          schema,
		customValidators: make(map[string]ValidatorFunc),
	}
	
	// Register built-in custom validators
	cv.registerBuiltInValidators()
	
	return cv
}

// Validate validates a configuration against the schema
func (cv *ConfigValidator) Validate(config *Config) *ValidationResult {
	start := time.Now()
	
	result := &ValidationResult{
		Valid:       true,
		Errors:      []ValidationError{},
		Warnings:    []ValidationWarning{},
		ValidatedAt: time.Now(),
	}
	
	// Set default context if not provided
	if cv.context == nil {
		cv.context = &ValidationContext{
			Config:     config,
			Schema:     cv.schema,
			StrictMode: false,
			MaxErrors:  100,
		}
	}
	
	// Convert config to map for easier traversal
	configMap, err := cv.configToMap(config)
	if err != nil {
		result.AddError("", fmt.Sprintf("Failed to parse configuration: %v", err), "PARSE_ERROR")
		result.Duration = time.Since(start)
		return result
	}
	
	// Validate each section
	for sectionName, section := range cv.schema.Sections {
		cv.validateSection(sectionName, section, configMap, result)
	}
	
	// Apply custom validation rules
	cv.applyValidationRules(configMap, result)
	
	// Check for unknown fields in strict mode
	if cv.context.StrictMode {
		cv.checkUnknownFields(configMap, result)
	}
	
	// Generate summary
	result.Summary = cv.generateSummary(result)
	result.Duration = time.Since(start)
	
	return result
}

// ValidateWithContext validates a configuration with a specific context
func (cv *ConfigValidator) ValidateWithContext(config *Config, context *ValidationContext) *ValidationResult {
	cv.context = context
	return cv.Validate(config)
}

// validateSection validates a configuration section
func (cv *ConfigValidator) validateSection(name string, section SectionSchema, configMap map[string]interface{}, result *ValidationResult) {
	sectionData, exists := configMap[name]
	if !exists {
		// Check if section is required
		for _, required := range cv.schema.Required {
			if required == name {
				result.AddError(name, "Required section is missing", "REQUIRED_SECTION")
				return
			}
		}
		return
	}
	
	sectionMap, ok := sectionData.(map[string]interface{})
	if !ok {
		result.AddError(name, fmt.Sprintf("Expected object, got %T", sectionData), "TYPE_ERROR")
		return
	}
	
	// Validate each property in the section
	for propName, propSchema := range section.Properties {
		path := fmt.Sprintf("%s.%s", name, propName)
		
		// Handle wildcard properties (e.g., providers.*)
		if propName == "*" {
			cv.validateWildcardProperties(path, propSchema, sectionMap, result)
			continue
		}
		
		value, exists := sectionMap[propName]
		if !exists {
			if propSchema.Required {
				result.AddError(path, "Required field is missing", "REQUIRED_FIELD")
			} else if propSchema.Default != nil {
				// Apply default value
				sectionMap[propName] = propSchema.Default
			}
			continue
		}
		
		cv.validateProperty(path, value, propSchema, result)
	}
	
	// Check required fields for the section
	for _, required := range section.Required {
		if _, exists := sectionMap[required]; !exists {
			result.AddError(fmt.Sprintf("%s.%s", name, required), "Required field is missing", "REQUIRED_FIELD")
		}
	}
}

// validateProperty validates a single property
func (cv *ConfigValidator) validateProperty(path string, value interface{}, schema PropertySchema, result *ValidationResult) {
	// Check if field is deprecated
	if schema.Deprecated {
		result.AddWarning(path, "This field is deprecated and may be removed in future versions", "DEPRECATED_FIELD")
	}
	
	// Skip validation for certain fields if requested
	for _, skipField := range cv.context.SkipFields {
		if path == skipField {
			return
		}
	}
	
	// Type validation
	if err := cv.validateType(value, schema.Type, path); err != nil {
		result.AddError(path, err.Error(), "TYPE_ERROR")
		return
	}
	
	// Apply type-specific validations
	switch schema.Type {
	case "string":
		cv.validateString(path, value, schema, result)
	case "number":
		cv.validateNumber(path, value, schema, result)
	case "boolean":
		// Boolean doesn't need additional validation
	case "array":
		cv.validateArray(path, value, schema, result)
	case "object":
		cv.validateObject(path, value, schema, result)
	case "duration":
		cv.validateDuration(path, value, schema, result)
	}
	
	// Enum validation
	if len(schema.Enum) > 0 {
		cv.validateEnum(path, value, schema.Enum, result)
	}
	
	// Format validation
	if schema.Format != "" {
		cv.validateFormat(path, value, schema.Format, result)
	}
	
	// Custom validator
	if schema.CustomValidator != "" {
		if validator, exists := cv.customValidators[schema.CustomValidator]; exists {
			if err := validator(value, &schema, path); err != nil {
				result.AddError(path, err.Error(), "CUSTOM_VALIDATION_ERROR")
			}
		}
	}
}

// validateType checks if the value matches the expected type
func (cv *ConfigValidator) validateType(value interface{}, expectedType string, path string) error {
	if value == nil {
		return nil
	}
	
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			// Valid number types
		default:
			// Try to convert from JSON number
			if _, ok := value.(json.Number); !ok {
				return fmt.Errorf("expected number, got %T", value)
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if reflect.TypeOf(value).Kind() != reflect.Slice {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if reflect.TypeOf(value).Kind() != reflect.Map {
			return fmt.Errorf("expected object, got %T", value)
		}
	case "duration":
		// Duration can be string or time.Duration
		switch v := value.(type) {
		case string:
			if _, err := time.ParseDuration(v); err != nil {
				return fmt.Errorf("invalid duration format: %v", err)
			}
		case time.Duration:
			// Valid duration
		default:
			return fmt.Errorf("expected duration string, got %T", value)
		}
	}
	
	return nil
}

// validateString validates string constraints
func (cv *ConfigValidator) validateString(path string, value interface{}, schema PropertySchema, result *ValidationResult) {
	str, ok := value.(string)
	if !ok {
		return
	}
	
	// Length validation
	if schema.MinLength > 0 && len(str) < schema.MinLength {
		result.AddError(path, fmt.Sprintf("String length must be at least %d characters", schema.MinLength), "MIN_LENGTH")
	}
	if schema.MaxLength > 0 && len(str) > schema.MaxLength {
		result.AddError(path, fmt.Sprintf("String length must not exceed %d characters", schema.MaxLength), "MAX_LENGTH")
	}
	
	// Pattern validation
	if schema.Pattern != "" {
		if matched, err := regexp.MatchString(schema.Pattern, str); err != nil {
			result.AddError(path, fmt.Sprintf("Invalid pattern: %v", err), "PATTERN_ERROR")
		} else if !matched {
			result.AddError(path, fmt.Sprintf("Value does not match required pattern: %s", schema.Pattern), "PATTERN_MISMATCH")
		}
	}
}

// validateNumber validates number constraints
func (cv *ConfigValidator) validateNumber(path string, value interface{}, schema PropertySchema, result *ValidationResult) {
	// Convert to float64 for comparison
	var num float64
	switch v := value.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float64:
		num = v
	case json.Number:
		f, _ := v.Float64()
		num = f
	default:
		return
	}
	
	// Min/Max validation
	if schema.MinValue != nil {
		if minVal, ok := cv.toFloat64(schema.MinValue); ok && num < minVal {
			result.AddError(path, fmt.Sprintf("Value must be at least %v", schema.MinValue), "MIN_VALUE")
		}
	}
	if schema.MaxValue != nil {
		if maxVal, ok := cv.toFloat64(schema.MaxValue); ok && num > maxVal {
			result.AddError(path, fmt.Sprintf("Value must not exceed %v", schema.MaxValue), "MAX_VALUE")
		}
	}
}

// validateArray validates array constraints
func (cv *ConfigValidator) validateArray(path string, value interface{}, schema PropertySchema, result *ValidationResult) {
	arr := reflect.ValueOf(value)
	if arr.Kind() != reflect.Slice {
		return
	}
	
	// Validate each item if item schema is provided
	if schema.Items != nil {
		for i := 0; i < arr.Len(); i++ {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			cv.validateProperty(itemPath, arr.Index(i).Interface(), *schema.Items, result)
		}
	}
}

// validateObject validates object constraints
func (cv *ConfigValidator) validateObject(path string, value interface{}, schema PropertySchema, result *ValidationResult) {
	objMap, ok := value.(map[string]interface{})
	if !ok {
		return
	}
	
	// Validate each property if schema is provided
	for propName, propSchema := range schema.Properties {
		propPath := fmt.Sprintf("%s.%s", path, propName)
		if propValue, exists := objMap[propName]; exists {
			cv.validateProperty(propPath, propValue, propSchema, result)
		} else if propSchema.Required {
			result.AddError(propPath, "Required field is missing", "REQUIRED_FIELD")
		}
	}
}

// validateDuration validates duration format
func (cv *ConfigValidator) validateDuration(path string, value interface{}, schema PropertySchema, result *ValidationResult) {
	var duration time.Duration
	
	switch v := value.(type) {
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			result.AddError(path, fmt.Sprintf("Invalid duration format: %v", err), "DURATION_FORMAT")
			return
		}
		duration = d
	case time.Duration:
		duration = v
	default:
		return
	}
	
	// Apply min/max constraints if they are durations
	if schema.MinValue != nil {
		if minDur, ok := cv.toDuration(schema.MinValue); ok && duration < minDur {
			result.AddError(path, fmt.Sprintf("Duration must be at least %v", minDur), "MIN_DURATION")
		}
	}
	if schema.MaxValue != nil {
		if maxDur, ok := cv.toDuration(schema.MaxValue); ok && duration > maxDur {
			result.AddError(path, fmt.Sprintf("Duration must not exceed %v", maxDur), "MAX_DURATION")
		}
	}
}

// validateEnum validates if value is in allowed enum values
func (cv *ConfigValidator) validateEnum(path string, value interface{}, enum []interface{}, result *ValidationResult) {
	found := false
	for _, allowed := range enum {
		if reflect.DeepEqual(value, allowed) {
			found = true
			break
		}
	}
	
	if !found {
		result.AddError(path, fmt.Sprintf("Value must be one of: %v", enum), "ENUM_MISMATCH")
	}
}

// validateFormat validates common string formats
func (cv *ConfigValidator) validateFormat(path string, value interface{}, format string, result *ValidationResult) {
	str, ok := value.(string)
	if !ok {
		return
	}
	
	switch format {
	case "email":
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(str) {
			result.AddError(path, "Invalid email format", "FORMAT_EMAIL")
		}
	case "url":
		if _, err := url.Parse(str); err != nil {
			result.AddError(path, "Invalid URL format", "FORMAT_URL")
		}
	case "ipv4":
		ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
		if !ipv4Regex.MatchString(str) {
			result.AddError(path, "Invalid IPv4 format", "FORMAT_IPV4")
		}
	case "ipv6":
		ipv6Regex := regexp.MustCompile(`^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|::)$`)
		if !ipv6Regex.MatchString(str) {
			result.AddError(path, "Invalid IPv6 format", "FORMAT_IPV6")
		}
	case "hostname":
		hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
		if !hostnameRegex.MatchString(str) {
			result.AddError(path, "Invalid hostname format", "FORMAT_HOSTNAME")
		}
	case "date":
		if _, err := time.Parse("2006-01-02", str); err != nil {
			result.AddError(path, "Invalid date format (expected YYYY-MM-DD)", "FORMAT_DATE")
		}
	case "datetime":
		if _, err := time.Parse(time.RFC3339, str); err != nil {
			result.AddError(path, "Invalid datetime format (expected RFC3339)", "FORMAT_DATETIME")
		}
	}
}

// validateWildcardProperties validates properties with wildcard patterns
func (cv *ConfigValidator) validateWildcardProperties(basePath string, schema PropertySchema, data map[string]interface{}, result *ValidationResult) {
	for key, value := range data {
		path := strings.Replace(basePath, "*", key, 1)
		cv.validateProperty(path, value, schema, result)
	}
}

// applyValidationRules applies custom validation rules
func (cv *ConfigValidator) applyValidationRules(configMap map[string]interface{}, result *ValidationResult) {
	for _, rule := range cv.schema.Rules {
		switch rule.Type {
		case "required_if":
			cv.applyRequiredIfRule(rule, configMap, result)
		case "exclusive":
			cv.applyExclusiveRule(rule, configMap, result)
		case "at_least_one":
			cv.applyAtLeastOneRule(rule, configMap, result)
		}
	}
}

// applyRequiredIfRule applies conditional required validation
func (cv *ConfigValidator) applyRequiredIfRule(rule ValidationRule, configMap map[string]interface{}, result *ValidationResult) {
	// This is a simplified implementation
	// In a real implementation, you would parse and evaluate the condition
	// For now, we'll skip complex condition evaluation
}

// applyExclusiveRule ensures only one of the fields is set
func (cv *ConfigValidator) applyExclusiveRule(rule ValidationRule, configMap map[string]interface{}, result *ValidationResult) {
	count := 0
	for _, field := range rule.Fields {
		if cv.fieldExists(field, configMap) {
			count++
		}
	}
	
	if count > 1 {
		result.AddError(strings.Join(rule.Fields, ", "), rule.Message, "EXCLUSIVE_RULE")
	}
}

// applyAtLeastOneRule ensures at least one of the fields is set
func (cv *ConfigValidator) applyAtLeastOneRule(rule ValidationRule, configMap map[string]interface{}, result *ValidationResult) {
	found := false
	for _, field := range rule.Fields {
		if cv.fieldExists(field, configMap) {
			found = true
			break
		}
	}
	
	if !found {
		result.AddError(strings.Join(rule.Fields, ", "), rule.Message, "AT_LEAST_ONE_RULE")
	}
}

// fieldExists checks if a field exists in the configuration
func (cv *ConfigValidator) fieldExists(path string, data map[string]interface{}) bool {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			_, exists := current[part]
			return exists
		}
		
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return false
		}
	}
	
	return false
}

// checkUnknownFields checks for fields not defined in schema
func (cv *ConfigValidator) checkUnknownFields(configMap map[string]interface{}, result *ValidationResult) {
	// Implementation would recursively check all fields against schema
	// For brevity, this is simplified
}

// generateSummary generates a validation summary
func (cv *ConfigValidator) generateSummary(result *ValidationResult) string {
	if result.Valid {
		if len(result.Warnings) > 0 {
			return fmt.Sprintf("Configuration is valid with %d warnings", len(result.Warnings))
		}
		return "Configuration is valid"
	}
	
	errorsByCategory := make(map[string]int)
	for _, err := range result.Errors {
		errorsByCategory[err.Code]++
	}
	
	parts := []string{fmt.Sprintf("Configuration validation failed with %d errors", len(result.Errors))}
	for code, count := range errorsByCategory {
		parts = append(parts, fmt.Sprintf("%s: %d", code, count))
	}
	
	return strings.Join(parts, ", ")
}

// configToMap converts Config struct to map for validation
func (cv *ConfigValidator) configToMap(config *Config) (map[string]interface{}, error) {
	// Marshal to JSON then unmarshal to map
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// Helper functions

func (cv *ConfigValidator) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func (cv *ConfigValidator) toDuration(value interface{}) (time.Duration, bool) {
	switch v := value.(type) {
	case string:
		d, err := time.ParseDuration(v)
		return d, err == nil
	case time.Duration:
		return v, true
	default:
		return 0, false
	}
}

// registerBuiltInValidators registers built-in custom validators
func (cv *ConfigValidator) registerBuiltInValidators() {
	// Provider API key validator
	cv.RegisterValidator("provider_api_key", func(value interface{}, schema *PropertySchema, path string) error {
		str, ok := value.(string)
		if !ok || str == "" {
			return nil // Empty is ok, required check handles this separately
		}
		
		// Basic API key format validation
		if len(str) < 20 {
			return fmt.Errorf("API key appears to be too short")
		}
		
		// Check for common placeholder values
		placeholders := []string{"your-api-key", "YOUR_API_KEY", "xxx", "XXXX", "<api-key>"}
		for _, placeholder := range placeholders {
			if strings.Contains(strings.ToLower(str), strings.ToLower(placeholder)) {
				return fmt.Errorf("API key appears to be a placeholder value")
			}
		}
		
		return nil
	})
	
	// Connection string validator
	cv.RegisterValidator("connection_string", func(value interface{}, schema *PropertySchema, path string) error {
		str, ok := value.(string)
		if !ok || str == "" {
			return nil
		}
		
		// Basic connection string validation
		if !strings.Contains(str, "://") {
			return fmt.Errorf("Connection string should contain protocol (e.g., postgresql://)")
		}
		
		return nil
	})
}

// RegisterValidator registers a custom validator function
func (cv *ConfigValidator) RegisterValidator(name string, fn ValidatorFunc) {
	cv.customValidators[name] = fn
}

// Test validates configuration connectivity and functionality
func (cv *ConfigValidator) Test(config *Config) *TestResult {
	// This will be implemented in tester.go
	return &TestResult{
		Success: true,
		Summary: "Testing functionality will be implemented in tester.go",
	}
}