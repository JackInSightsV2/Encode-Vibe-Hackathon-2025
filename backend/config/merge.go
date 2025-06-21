package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ConfigMerger handles configuration merging operations
type ConfigMerger struct {
	validator *ConfigValidator
	differ    *ConfigDiffer
}

// NewConfigMerger creates a new configuration merger
func NewConfigMerger(validator *ConfigValidator) *ConfigMerger {
	return &ConfigMerger{
		validator: validator,
		differ:    NewConfigDiffer(),
	}
}

// Merge merges two configurations based on the specified options
func (cm *ConfigMerger) Merge(target, source *Config, options MergeOptions) (*MergeResult, error) {
	result := &MergeResult{
		Success:   true,
		Conflicts: []MergeConflict{},
		Changes:   []ConfigChange{},
	}
	
	// Convert configs to maps for merging
	targetMap, err := cm.configToMap(target)
	if err != nil {
		return nil, fmt.Errorf("failed to convert target config: %w", err)
	}
	
	sourceMap, err := cm.configToMap(source)
	if err != nil {
		return nil, fmt.Errorf("failed to convert source config: %w", err)
	}
	
	// Perform the merge
	mergedMap, conflicts := cm.mergeMaps("", targetMap, sourceMap, options)
	result.Conflicts = conflicts
	
	// Convert merged map back to Config
	mergedConfig, err := cm.mapToConfig(mergedMap)
	if err != nil {
		result.Success = false
		return result, fmt.Errorf("failed to convert merged map to config: %w", err)
	}
	
	result.MergedConfig = mergedConfig
	
	// Calculate changes
	if diff, err := cm.differ.Diff(target, mergedConfig); err == nil {
		result.Changes = diff.Changes
	}
	
	// Validate if requested
	if options.ValidateResult && cm.validator != nil {
		validationResult := cm.validator.Validate(mergedConfig)
		result.ValidationResult = validationResult
		if !validationResult.Valid {
			result.Success = false
		}
	}
	
	return result, nil
}

// MergeWithTemplate merges a configuration with a template
func (cm *ConfigMerger) MergeWithTemplate(config *Config, template *ConfigTemplate, variables map[string]interface{}) (*Config, error) {
	// Apply variables to template
	appliedTemplate, err := cm.applyTemplateVariables(template.Template, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template variables: %w", err)
	}
	
	// Convert template to Config
	templateConfig, err := cm.mapToConfig(appliedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert template to config: %w", err)
	}
	
	// Merge with existing config
	options := MergeOptions{
		Strategy:       MergeStrategyMerge,
		DeepMerge:      true,
		ValidateResult: true,
	}
	
	result, err := cm.Merge(config, templateConfig, options)
	if err != nil {
		return nil, err
	}
	
	if !result.Success {
		return nil, fmt.Errorf("merge failed: validation errors")
	}
	
	return result.MergedConfig, nil
}

// mergeMaps recursively merges two maps
func (cm *ConfigMerger) mergeMaps(path string, target, source map[string]interface{}, options MergeOptions) (map[string]interface{}, []MergeConflict) {
	conflicts := []MergeConflict{}
	result := make(map[string]interface{})
	
	// Copy target values
	for k, v := range target {
		result[k] = v
	}
	
	// Merge source values
	for key, sourceValue := range source {
		fullPath := cm.buildPath(path, key)
		
		// Check if field should be ignored
		if cm.shouldIgnoreField(fullPath, options.IgnoreFields) {
			continue
		}
		
		targetValue, exists := result[key]
		
		if !exists {
			// Field doesn't exist in target, add it
			result[key] = sourceValue
		} else {
			// Field exists, need to merge or resolve conflict
			mergedValue, conflict := cm.mergeValues(fullPath, targetValue, sourceValue, options)
			if conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
			result[key] = mergedValue
		}
	}
	
	return result, conflicts
}

// mergeValues merges two values based on merge options
func (cm *ConfigMerger) mergeValues(path string, targetValue, sourceValue interface{}, options MergeOptions) (interface{}, *MergeConflict) {
	// If values are equal, no merge needed
	if reflect.DeepEqual(targetValue, sourceValue) {
		return targetValue, nil
	}
	
	// Handle different merge strategies
	switch options.Strategy {
	case MergeStrategyOverwrite:
		return sourceValue, nil
		
	case MergeStrategySkip:
		return targetValue, nil
		
	case MergeStrategyAppend:
		// Only applies to arrays
		if targetArray, ok := targetValue.([]interface{}); ok {
			if sourceArray, ok := sourceValue.([]interface{}); ok {
				return cm.appendArrays(targetArray, sourceArray, options.PreserveArrays), nil
			}
		}
		// For non-arrays, fall through to merge
		fallthrough
		
	case MergeStrategyMerge:
		// Deep merge for objects
		if options.DeepMerge {
			if targetMap, ok := targetValue.(map[string]interface{}); ok {
				if sourceMap, ok := sourceValue.(map[string]interface{}); ok {
					merged, conflicts := cm.mergeMaps(path, targetMap, sourceMap, options)
					// Return first conflict if any
					if len(conflicts) > 0 {
						return merged, &conflicts[0]
					}
					return merged, nil
				}
			}
		}
		
		// Handle arrays
		if targetArray, ok := targetValue.([]interface{}); ok {
			if sourceArray, ok := sourceValue.([]interface{}); ok {
				if options.PreserveArrays {
					// Keep target array
					return targetArray, nil
				}
				// Replace with source array
				return sourceArray, nil
			}
		}
		
		// For primitive values, need conflict resolution
		conflict := &MergeConflict{
			Path:        path,
			SourceValue: sourceValue,
			TargetValue: targetValue,
		}
		
		// Apply conflict resolution
		resolvedValue := cm.resolveConflict(conflict, options.ConflictResolver)
		conflict.Resolution = string(options.ConflictResolver)
		conflict.ResolvedValue = resolvedValue
		
		return resolvedValue, conflict
	}
	
	return targetValue, nil
}

// resolveConflict resolves a merge conflict based on the resolver strategy
func (cm *ConfigMerger) resolveConflict(conflict *MergeConflict, resolver ConflictResolver) interface{} {
	switch resolver {
	case ConflictResolverSource:
		return conflict.SourceValue
		
	case ConflictResolverTarget:
		return conflict.TargetValue
		
	case ConflictResolverNewer:
		// Without timestamps, default to source
		return conflict.SourceValue
		
	case ConflictResolverInteractive:
		// In non-interactive mode, default to target
		return conflict.TargetValue
		
	default:
		return conflict.TargetValue
	}
}

// appendArrays appends two arrays
func (cm *ConfigMerger) appendArrays(target, source []interface{}, preserveUnique bool) []interface{} {
	if !preserveUnique {
		return append(target, source...)
	}
	
	// Preserve unique values only
	result := make([]interface{}, len(target))
	copy(result, target)
	
	for _, sourceItem := range source {
		found := false
		for _, targetItem := range target {
			if reflect.DeepEqual(sourceItem, targetItem) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, sourceItem)
		}
	}
	
	return result
}

// shouldIgnoreField checks if a field should be ignored during merge
func (cm *ConfigMerger) shouldIgnoreField(path string, ignoreFields []string) bool {
	for _, ignore := range ignoreFields {
		if path == ignore || strings.HasPrefix(path, ignore+".") {
			return true
		}
	}
	return false
}

// buildPath builds a dot-separated path
func (cm *ConfigMerger) buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// applyTemplateVariables applies variables to a template
func (cm *ConfigMerger) applyTemplateVariables(template map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// Deep copy template
	result := cm.deepCopyMap(template)
	
	// Apply variables recursively
	cm.replaceVariables(result, variables)
	
	return result, nil
}

// replaceVariables recursively replaces template variables
func (cm *ConfigMerger) replaceVariables(data map[string]interface{}, variables map[string]interface{}) {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Check if it's a template variable
			if strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}") {
				varName := strings.TrimSpace(v[2 : len(v)-2])
				if replacement, ok := variables[varName]; ok {
					data[key] = replacement
				}
			}
			
		case map[string]interface{}:
			// Recurse into nested maps
			cm.replaceVariables(v, variables)
			
		case []interface{}:
			// Process array elements
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					cm.replaceVariables(itemMap, variables)
				} else if itemStr, ok := item.(string); ok {
					if strings.HasPrefix(itemStr, "{{") && strings.HasSuffix(itemStr, "}}") {
						varName := strings.TrimSpace(itemStr[2 : len(itemStr)-2])
						if replacement, ok := variables[varName]; ok {
							v[i] = replacement
						}
					}
				}
			}
		}
	}
}

// deepCopyMap creates a deep copy of a map
func (cm *ConfigMerger) deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range m {
		switch v := v.(type) {
		case map[string]interface{}:
			result[k] = cm.deepCopyMap(v)
		case []interface{}:
			result[k] = cm.deepCopySlice(v)
		default:
			result[k] = v
		}
	}
	
	return result
}

// deepCopySlice creates a deep copy of a slice
func (cm *ConfigMerger) deepCopySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	
	for i, v := range s {
		switch v := v.(type) {
		case map[string]interface{}:
			result[i] = cm.deepCopyMap(v)
		case []interface{}:
			result[i] = cm.deepCopySlice(v)
		default:
			result[i] = v
		}
	}
	
	return result
}

// configToMap converts a Config to a map
func (cm *ConfigMerger) configToMap(config *Config) (map[string]interface{}, error) {
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

// mapToConfig converts a map to a Config
func (cm *ConfigMerger) mapToConfig(m map[string]interface{}) (*Config, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// MergeEnvironmentConfig merges environment-specific configuration
func (cm *ConfigMerger) MergeEnvironmentConfig(baseConfig *Config, envConfig map[string]interface{}, environment string) (*Config, error) {
	// Convert base config to map
	baseMap, err := cm.configToMap(baseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert base config: %w", err)
	}
	
	// Apply environment-specific overrides
	options := MergeOptions{
		Strategy:       MergeStrategyMerge,
		DeepMerge:      true,
		ValidateResult: true,
	}
	
	mergedMap, conflicts := cm.mergeMaps("", baseMap, envConfig, options)
	
	// Log conflicts if any
	for _, conflict := range conflicts {
		// In a real implementation, you might want to log these
		_ = conflict
	}
	
	// Convert back to Config
	mergedConfig, err := cm.mapToConfig(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert merged config: %w", err)
	}
	
	return mergedConfig, nil
}

// CreateMergePreview creates a preview of what would be merged
func (cm *ConfigMerger) CreateMergePreview(target, source *Config, options MergeOptions) (*MergeResult, error) {
	// Perform a dry-run merge
	result, err := cm.Merge(target, source, options)
	if err != nil {
		return nil, err
	}
	
	// Don't return the actual merged config, just the preview info
	preview := &MergeResult{
		Success:          result.Success,
		Conflicts:        result.Conflicts,
		Changes:          result.Changes,
		ValidationResult: result.ValidationResult,
		// Don't include MergedConfig in preview
	}
	
	return preview, nil
}

// ApplyDefaults applies default values from schema to a configuration
func (cm *ConfigMerger) ApplyDefaults(config *Config, schema *ConfigSchema) (*Config, error) {
	// Convert config to map
	configMap, err := cm.configToMap(config)
	if err != nil {
		return nil, err
	}
	
	// Apply defaults from schema
	cm.applySchemaDefaults(configMap, schema.Sections, "")
	
	// Convert back to Config
	return cm.mapToConfig(configMap)
}

// applySchemaDefaults recursively applies default values from schema
func (cm *ConfigMerger) applySchemaDefaults(config map[string]interface{}, sections map[string]SectionSchema, parentPath string) {
	for sectionName, section := range sections {
		_ = sectionName // Mark as used
		if parentPath != "" {
			// sectionPath would be used for nested paths
			_ = parentPath + "." + sectionName
		}
		
		// Ensure section exists
		if _, exists := config[sectionName]; !exists {
			config[sectionName] = make(map[string]interface{})
		}
		
		sectionConfig, ok := config[sectionName].(map[string]interface{})
		if !ok {
			continue
		}
		
		// Apply property defaults
		for propName, propSchema := range section.Properties {
			if _, exists := sectionConfig[propName]; !exists && propSchema.Default != nil {
				sectionConfig[propName] = propSchema.Default
			}
			
			// Handle nested properties
			if len(propSchema.Properties) > 0 {
				if propConfig, ok := sectionConfig[propName].(map[string]interface{}); ok {
					cm.applyPropertyDefaults(propConfig, propSchema.Properties)
				}
			}
		}
	}
}

// applyPropertyDefaults applies defaults to nested properties
func (cm *ConfigMerger) applyPropertyDefaults(config map[string]interface{}, properties map[string]PropertySchema) {
	for propName, propSchema := range properties {
		if _, exists := config[propName]; !exists && propSchema.Default != nil {
			config[propName] = propSchema.Default
		}
		
		// Recurse for nested properties
		if len(propSchema.Properties) > 0 {
			if propConfig, ok := config[propName].(map[string]interface{}); ok {
				cm.applyPropertyDefaults(propConfig, propSchema.Properties)
			}
		}
	}
}