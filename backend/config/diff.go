package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ConfigDiffer compares configurations and generates diffs
type ConfigDiffer struct {
	ignoreFields    map[string]bool
	sensitiveFields map[string]bool
}

// NewConfigDiffer creates a new configuration differ
func NewConfigDiffer() *ConfigDiffer {
	return &ConfigDiffer{
		ignoreFields: map[string]bool{
			"validated_at": true,
			"created_at":   true,
			"updated_at":   true,
		},
		sensitiveFields: map[string]bool{
			"api_key":        true,
			"password":       true,
			"secret":         true,
			"token":          true,
			"private_key":    true,
			"openai_api_key": true,
		},
	}
}

// Diff compares two configurations and returns the differences
func (cd *ConfigDiffer) Diff(oldConfig, newConfig *Config) (*ConfigDiff, error) {
	// Convert configs to maps for comparison
	oldMap, err := cd.configToMap(oldConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert old config: %w", err)
	}
	
	newMap, err := cd.configToMap(newConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert new config: %w", err)
	}
	
	// Perform the diff
	changes := cd.diffMaps("", oldMap, newMap)
	
	// Calculate summary
	diff := &ConfigDiff{
		Changes: changes,
	}
	
	for _, change := range changes {
		switch change.Operation {
		case "create":
			diff.AddedCount++
		case "update":
			diff.UpdatedCount++
		case "delete":
			diff.DeletedCount++
		}
	}
	
	diff.ImpactLevel = cd.calculateImpactLevel(changes)
	diff.Summary = cd.generateDiffSummary(diff)
	
	return diff, nil
}

// DiffJSON compares two JSON configurations
func (cd *ConfigDiffer) DiffJSON(oldJSON, newJSON []byte) (*ConfigDiff, error) {
	var oldMap, newMap map[string]interface{}
	
	if err := json.Unmarshal(oldJSON, &oldMap); err != nil {
		return nil, fmt.Errorf("failed to parse old JSON: %w", err)
	}
	
	if err := json.Unmarshal(newJSON, &newMap); err != nil {
		return nil, fmt.Errorf("failed to parse new JSON: %w", err)
	}
	
	changes := cd.diffMaps("", oldMap, newMap)
	
	diff := &ConfigDiff{
		Changes: changes,
	}
	
	for _, change := range changes {
		switch change.Operation {
		case "create":
			diff.AddedCount++
		case "update":
			diff.UpdatedCount++
		case "delete":
			diff.DeletedCount++
		}
	}
	
	diff.ImpactLevel = cd.calculateImpactLevel(changes)
	diff.Summary = cd.generateDiffSummary(diff)
	
	return diff, nil
}

// diffMaps recursively compares two maps
func (cd *ConfigDiffer) diffMaps(path string, oldMap, newMap map[string]interface{}) []ConfigChange {
	changes := []ConfigChange{}
	
	// Check for deleted and updated fields
	for key, oldValue := range oldMap {
		fullPath := cd.buildPath(path, key)
		
		if cd.shouldIgnoreField(fullPath) {
			continue
		}
		
		newValue, exists := newMap[key]
		if !exists {
			// Field was deleted
			changes = append(changes, ConfigChange{
				Path:        fullPath,
				Operation:   "delete",
				OldValue:    cd.sanitizeValue(fullPath, oldValue),
				Description: fmt.Sprintf("Removed %s", key),
				Impact:      cd.assessFieldImpact(fullPath, "delete"),
			})
		} else if !cd.valuesEqual(oldValue, newValue) {
			// Field was updated
			change := cd.compareValues(fullPath, oldValue, newValue)
			if change != nil {
				changes = append(changes, *change)
			}
		}
	}
	
	// Check for added fields
	for key, newValue := range newMap {
		fullPath := cd.buildPath(path, key)
		
		if cd.shouldIgnoreField(fullPath) {
			continue
		}
		
		if _, exists := oldMap[key]; !exists {
			// Field was added
			changes = append(changes, ConfigChange{
				Path:        fullPath,
				Operation:   "create",
				NewValue:    cd.sanitizeValue(fullPath, newValue),
				Description: fmt.Sprintf("Added %s", key),
				Impact:      cd.assessFieldImpact(fullPath, "create"),
			})
		}
	}
	
	return changes
}

// compareValues compares two values and generates a change if different
func (cd *ConfigDiffer) compareValues(path string, oldValue, newValue interface{}) *ConfigChange {
	// Handle nested objects
	if oldMap, ok := oldValue.(map[string]interface{}); ok {
		if newMap, ok := newValue.(map[string]interface{}); ok {
			nestedChanges := cd.diffMaps(path, oldMap, newMap)
			if len(nestedChanges) > 0 {
				// For nested changes, we don't create a parent change
				// Instead, we return nil and let the nested changes be added
				return nil
			}
		}
	}
	
	// Handle arrays
	if oldArray, ok := oldValue.([]interface{}); ok {
		if newArray, ok := newValue.([]interface{}); ok {
			if !cd.arraysEqual(oldArray, newArray) {
				return &ConfigChange{
					Path:        path,
					Operation:   "update",
					OldValue:    cd.sanitizeValue(path, oldValue),
					NewValue:    cd.sanitizeValue(path, newValue),
					Description: cd.generateChangeDescription(path, oldValue, newValue),
					Impact:      cd.assessFieldImpact(path, "update"),
				}
			}
			return nil
		}
	}
	
	// Simple value change
	return &ConfigChange{
		Path:        path,
		Operation:   "update",
		OldValue:    cd.sanitizeValue(path, oldValue),
		NewValue:    cd.sanitizeValue(path, newValue),
		Description: cd.generateChangeDescription(path, oldValue, newValue),
		Impact:      cd.assessFieldImpact(path, "update"),
	}
}

// valuesEqual checks if two values are equal
func (cd *ConfigDiffer) valuesEqual(v1, v2 interface{}) bool {
	// Use reflect.DeepEqual for complex types
	return reflect.DeepEqual(v1, v2)
}

// arraysEqual checks if two arrays are equal
func (cd *ConfigDiffer) arraysEqual(a1, a2 []interface{}) bool {
	if len(a1) != len(a2) {
		return false
	}
	
	// For simple arrays, compare elements
	// Note: This doesn't handle order changes well
	for i := range a1 {
		if !cd.valuesEqual(a1[i], a2[i]) {
			return false
		}
	}
	
	return true
}

// buildPath builds a dot-separated path
func (cd *ConfigDiffer) buildPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

// shouldIgnoreField checks if a field should be ignored
func (cd *ConfigDiffer) shouldIgnoreField(path string) bool {
	// Check exact match
	if cd.ignoreFields[path] {
		return true
	}
	
	// Check if any part of the path should be ignored
	parts := strings.Split(path, ".")
	for i := range parts {
		partialPath := strings.Join(parts[:i+1], ".")
		if cd.ignoreFields[partialPath] {
			return true
		}
	}
	
	return false
}

// sanitizeValue sanitizes sensitive values
func (cd *ConfigDiffer) sanitizeValue(path string, value interface{}) interface{} {
	// Check if the field is sensitive
	if cd.isSensitiveField(path) {
		switch v := value.(type) {
		case string:
			if v != "" {
				return "[REDACTED]"
			}
		default:
			return "[REDACTED]"
		}
	}
	
	return value
}

// isSensitiveField checks if a field contains sensitive data
func (cd *ConfigDiffer) isSensitiveField(path string) bool {
	pathLower := strings.ToLower(path)
	
	// Check exact match
	if cd.sensitiveFields[pathLower] {
		return true
	}
	
	// Check if path contains sensitive keywords
	for keyword := range cd.sensitiveFields {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	return false
}

// generateChangeDescription generates a human-readable description
func (cd *ConfigDiffer) generateChangeDescription(path string, oldValue, newValue interface{}) string {
	fieldName := cd.getFieldName(path)
	
	// Special descriptions for common fields
	switch {
	case strings.HasSuffix(path, ".enabled"):
		if oldBool, ok := oldValue.(bool); ok {
			if newBool, ok := newValue.(bool); ok {
				if oldBool && !newBool {
					return fmt.Sprintf("Disabled %s", fieldName)
				} else if !oldBool && newBool {
					return fmt.Sprintf("Enabled %s", fieldName)
				}
			}
		}
		
	case strings.Contains(path, "port"):
		return fmt.Sprintf("Changed %s from %v to %v", fieldName, oldValue, newValue)
		
	case strings.Contains(path, "timeout"):
		return fmt.Sprintf("Changed %s timeout from %v to %v", fieldName, oldValue, newValue)
		
	case cd.isSensitiveField(path):
		return fmt.Sprintf("Updated %s credentials", fieldName)
	}
	
	// Generic description
	return fmt.Sprintf("Changed %s", fieldName)
}

// getFieldName extracts a human-readable field name from path
func (cd *ConfigDiffer) getFieldName(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Convert snake_case to Title Case
		words := strings.Split(lastPart, "_")
		for i, word := range words {
			words[i] = strings.Title(word)
		}
		return strings.Join(words, " ")
	}
	return path
}

// assessFieldImpact assesses the impact of a field change
func (cd *ConfigDiffer) assessFieldImpact(path string, operation string) string {
	// High impact fields
	highImpactPatterns := []string{
		"security",
		"authentication",
		"api_key",
		"database",
		"port",
		"host",
		"enabled",
	}
	
	// Medium impact fields
	mediumImpactPatterns := []string{
		"timeout",
		"retry",
		"cache",
		"logging",
		"metrics",
	}
	
	pathLower := strings.ToLower(path)
	
	// Check for high impact
	for _, pattern := range highImpactPatterns {
		if strings.Contains(pathLower, pattern) {
			return "high"
		}
	}
	
	// Check for medium impact
	for _, pattern := range mediumImpactPatterns {
		if strings.Contains(pathLower, pattern) {
			return "medium"
		}
	}
	
	// Deletions are generally medium impact
	if operation == "delete" {
		return "medium"
	}
	
	return "low"
}

// calculateImpactLevel calculates overall impact level
func (cd *ConfigDiffer) calculateImpactLevel(changes []ConfigChange) string {
	hasHigh := false
	hasMedium := false
	
	for _, change := range changes {
		switch change.Impact {
		case "high":
			hasHigh = true
		case "medium":
			hasMedium = true
		}
	}
	
	if hasHigh {
		return "high"
	} else if hasMedium {
		return "medium"
	} else if len(changes) > 0 {
		return "low"
	}
	
	return "none"
}

// generateDiffSummary generates a summary of the diff
func (cd *ConfigDiffer) generateDiffSummary(diff *ConfigDiff) string {
	if len(diff.Changes) == 0 {
		return "No changes detected"
	}
	
	parts := []string{}
	
	if diff.AddedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d added", diff.AddedCount))
	}
	if diff.UpdatedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", diff.UpdatedCount))
	}
	if diff.DeletedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", diff.DeletedCount))
	}
	
	summary := strings.Join(parts, ", ")
	
	// Add impact level
	if diff.ImpactLevel != "none" {
		summary += fmt.Sprintf(" (%s impact)", diff.ImpactLevel)
	}
	
	return summary
}

// configToMap converts a Config struct to a map
func (cd *ConfigDiffer) configToMap(config *Config) (map[string]interface{}, error) {
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

// SetIgnoreFields sets fields to ignore during diff
func (cd *ConfigDiffer) SetIgnoreFields(fields []string) {
	cd.ignoreFields = make(map[string]bool)
	for _, field := range fields {
		cd.ignoreFields[field] = true
	}
}

// AddIgnoreField adds a field to ignore during diff
func (cd *ConfigDiffer) AddIgnoreField(field string) {
	cd.ignoreFields[field] = true
}

// SetSensitiveFields sets fields to sanitize
func (cd *ConfigDiffer) SetSensitiveFields(fields []string) {
	cd.sensitiveFields = make(map[string]bool)
	for _, field := range fields {
		cd.sensitiveFields[strings.ToLower(field)] = true
	}
}

// AddSensitiveField adds a sensitive field
func (cd *ConfigDiffer) AddSensitiveField(field string) {
	cd.sensitiveFields[strings.ToLower(field)] = true
}

// FormatDiff formats a diff for display
func FormatDiff(diff *ConfigDiff, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(diff, "", "  ")
		return string(data), err
		
	case "text":
		return formatDiffAsText(diff), nil
		
	case "markdown":
		return formatDiffAsMarkdown(diff), nil
		
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func formatDiffAsText(diff *ConfigDiff) string {
	if len(diff.Changes) == 0 {
		return "No changes\n"
	}
	
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("Configuration Changes: %s\n", diff.Summary))
	sb.WriteString(fmt.Sprintf("Impact Level: %s\n\n", diff.ImpactLevel))
	
	// Group changes by operation
	creates := []ConfigChange{}
	updates := []ConfigChange{}
	deletes := []ConfigChange{}
	
	for _, change := range diff.Changes {
		switch change.Operation {
		case "create":
			creates = append(creates, change)
		case "update":
			updates = append(updates, change)
		case "delete":
			deletes = append(deletes, change)
		}
	}
	
	// Format creates
	if len(creates) > 0 {
		sb.WriteString("Added:\n")
		for _, change := range creates {
			sb.WriteString(fmt.Sprintf("  + %s = %v\n", change.Path, change.NewValue))
		}
		sb.WriteString("\n")
	}
	
	// Format updates
	if len(updates) > 0 {
		sb.WriteString("Updated:\n")
		for _, change := range updates {
			sb.WriteString(fmt.Sprintf("  ~ %s: %v → %v\n", 
				change.Path, change.OldValue, change.NewValue))
		}
		sb.WriteString("\n")
	}
	
	// Format deletes
	if len(deletes) > 0 {
		sb.WriteString("Deleted:\n")
		for _, change := range deletes {
			sb.WriteString(fmt.Sprintf("  - %s (was: %v)\n", change.Path, change.OldValue))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

func formatDiffAsMarkdown(diff *ConfigDiff) string {
	if len(diff.Changes) == 0 {
		return "**No changes**\n"
	}
	
	var sb strings.Builder
	
	sb.WriteString("# Configuration Changes\n\n")
	sb.WriteString(fmt.Sprintf("**Summary:** %s\n", diff.Summary))
	sb.WriteString(fmt.Sprintf("**Impact Level:** %s\n\n", diff.ImpactLevel))
	
	// Sort changes by path for consistent output
	changes := make([]ConfigChange, len(diff.Changes))
	copy(changes, diff.Changes)
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})
	
	sb.WriteString("## Changes\n\n")
	sb.WriteString("| Path | Operation | Old Value | New Value | Impact |\n")
	sb.WriteString("|------|-----------|-----------|-----------|--------|\n")
	
	for _, change := range changes {
		oldVal := formatValue(change.OldValue)
		newVal := formatValue(change.NewValue)
		
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
			change.Path,
			change.Operation,
			oldVal,
			newVal,
			change.Impact,
		))
	}
	
	return sb.String()
}

func formatValue(value interface{}) string {
	if value == nil {
		return "-"
	}
	
	switch v := value.(type) {
	case string:
		if v == "" {
			return "(empty)"
		}
		return fmt.Sprintf("`%s`", v)
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	case map[string]interface{}:
		return fmt.Sprintf("{%d fields}", len(v))
	default:
		return fmt.Sprintf("`%v`", v)
	}
}