package moderation

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RuleFileFormat represents the structure of a YAML rules file
type RuleFileFormat struct {
	Rules []*ModerationRule `yaml:"rules"`
}

// ValidationError represents a rule validation error
type ValidationError struct {
	RuleID  string
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Rule %s: %s - %s", e.RuleID, e.Field, e.Message)
}

// ParseRulesFromFile parses rules from a YAML file
func (rp *RuleParser) ParseRulesFromFile(filePath string) ([]*ModerationRule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	return rp.ParseRulesFromYAML(data)
}

// ParseRulesFromYAML parses rules from YAML data
func (rp *RuleParser) ParseRulesFromYAML(data []byte) ([]*ModerationRule, error) {
	var ruleFile RuleFileFormat
	
	if err := yaml.Unmarshal(data, &ruleFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate and process rules
	validRules := make([]*ModerationRule, 0, len(ruleFile.Rules))
	
	for _, rule := range ruleFile.Rules {
		if err := rp.ValidateRule(rule); err != nil {
			return nil, fmt.Errorf("rule validation failed: %w", err)
		}

		// Compile and normalize rule
		if err := rp.CompileRule(rule); err != nil {
			return nil, fmt.Errorf("rule compilation failed: %w", err)
		}

		// Set timestamps if not provided
		if rule.CreatedAt.IsZero() {
			rule.CreatedAt = time.Now()
		}
		rule.UpdatedAt = time.Now()

		validRules = append(validRules, rule)
	}

	return validRules, nil
}

// ValidateRule validates a single rule for correctness
func (rp *RuleParser) ValidateRule(rule *ModerationRule) error {
	if rule == nil {
		return &ValidationError{"", "rule", "rule cannot be nil"}
	}

	// Validate required fields
	if rule.ID == "" {
		return &ValidationError{rule.ID, "id", "rule ID is required"}
	}

	if rule.Name == "" {
		return &ValidationError{rule.ID, "name", "rule name is required"}
	}

	if rule.Type == "" {
		return &ValidationError{rule.ID, "type", "rule type is required"}
	}

	// Validate rule type
	validTypes := map[string]bool{
		RuleTypeRegex:     true,
		RuleTypeKeyword:   true,
		RuleTypeML:        true,
		RuleTypeComposite: true,
	}
	if !validTypes[rule.Type] {
		return &ValidationError{rule.ID, "type", fmt.Sprintf("invalid rule type '%s'", rule.Type)}
	}

	// Validate weight
	if rule.Weight < 0 || rule.Weight > 1 {
		return &ValidationError{rule.ID, "weight", "weight must be between 0 and 1"}
	}

	// Validate priority
	if rule.Priority < 1 || rule.Priority > 10 {
		return &ValidationError{rule.ID, "priority", "priority must be between 1 and 10"}
	}

	// Validate action
	validActions := map[string]bool{
		ActionBlock: true,
		ActionFlag:  true,
		ActionWarn:  true,
	}
	if !validActions[rule.Action] {
		return &ValidationError{rule.ID, "action", fmt.Sprintf("invalid action '%s'", rule.Action)}
	}

	// Validate category
	if rule.Category == "" {
		return &ValidationError{rule.ID, "category", "category is required"}
	}

	// Type-specific validation
	switch rule.Type {
	case RuleTypeRegex:
		if err := rp.validateRegexRule(rule); err != nil {
			return err
		}
	case RuleTypeKeyword:
		if err := rp.validateKeywordRule(rule); err != nil {
			return err
		}
	case RuleTypeML:
		if err := rp.validateMLRule(rule); err != nil {
			return err
		}
	case RuleTypeComposite:
		if err := rp.validateCompositeRule(rule); err != nil {
			return err
		}
	}

	// Validate context if provided
	if rule.Context != nil {
		if err := rp.validateRuleContext(rule); err != nil {
			return err
		}
	}

	// Validate conditions if provided
	if rule.Conditions != nil {
		if err := rp.validateRuleConditions(rule); err != nil {
			return err
		}
	}

	return nil
}

// validateRegexRule validates regex-specific fields
func (rp *RuleParser) validateRegexRule(rule *ModerationRule) error {
	if rule.Pattern == "" {
		return &ValidationError{rule.ID, "pattern", "regex pattern is required for regex rules"}
	}

	// Test regex compilation
	_, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return &ValidationError{rule.ID, "pattern", fmt.Sprintf("invalid regex pattern: %v", err)}
	}

	return nil
}

// validateKeywordRule validates keyword-specific fields
func (rp *RuleParser) validateKeywordRule(rule *ModerationRule) error {
	if len(rule.Keywords) == 0 {
		return &ValidationError{rule.ID, "keywords", "keywords list is required for keyword rules"}
	}

	// Validate individual keywords
	for i, keyword := range rule.Keywords {
		if strings.TrimSpace(keyword) == "" {
			return &ValidationError{rule.ID, "keywords", fmt.Sprintf("keyword at index %d is empty", i)}
		}
	}

	return nil
}

// validateMLRule validates ML-specific fields
func (rp *RuleParser) validateMLRule(rule *ModerationRule) error {
	if rule.Pattern == "" {
		return &ValidationError{rule.ID, "pattern", "model name/pattern is required for ML rules"}
	}

	// Additional ML-specific validation could go here
	// For example, checking if the model exists or is valid

	return nil
}

// validateCompositeRule validates composite rule fields
func (rp *RuleParser) validateCompositeRule(rule *ModerationRule) error {
	// Composite rules might combine multiple patterns/keywords
	// Validation logic depends on how composite rules are structured
	
	// For now, just ensure we have some criteria
	if rule.Pattern == "" && len(rule.Keywords) == 0 {
		return &ValidationError{rule.ID, "pattern/keywords", "composite rules must have either pattern or keywords"}
	}

	return nil
}

// validateRuleContext validates rule context configuration
func (rp *RuleParser) validateRuleContext(rule *ModerationRule) error {
	context := rule.Context

	// Validate user types
	validUserTypes := map[string]bool{
		"admin": true,
		"user":  true,
		"guest": true,
		"moderator": true,
	}
	for _, userType := range context.UserTypes {
		if !validUserTypes[userType] {
			return &ValidationError{rule.ID, "context.user_types", fmt.Sprintf("invalid user type '%s'", userType)}
		}
	}

	// Validate content types
	validContentTypes := map[string]bool{
		"message": true,
		"post":    true,
		"comment": true,
		"chat":    true,
	}
	for _, contentType := range context.ContentTypes {
		if !validContentTypes[contentType] {
			return &ValidationError{rule.ID, "context.content_types", fmt.Sprintf("invalid content type '%s'", contentType)}
		}
	}

	// Validate time ranges
	for i, timeRange := range context.TimeRanges {
		if err := rp.validateTimeRange(rule.ID, timeRange, i); err != nil {
			return err
		}
	}

	return nil
}

// validateTimeRange validates a time range configuration
func (rp *RuleParser) validateTimeRange(ruleID string, tr TimeRange, index int) error {
	// Validate time format (HH:MM)
	timePattern := regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)
	
	if !timePattern.MatchString(tr.Start) {
		return &ValidationError{ruleID, fmt.Sprintf("context.time_ranges[%d].start", index), "invalid time format, use HH:MM"}
	}
	
	if !timePattern.MatchString(tr.End) {
		return &ValidationError{ruleID, fmt.Sprintf("context.time_ranges[%d].end", index), "invalid time format, use HH:MM"}
	}

	// Validate timezone
	if tr.Timezone != "" {
		_, err := time.LoadLocation(tr.Timezone)
		if err != nil {
			return &ValidationError{ruleID, fmt.Sprintf("context.time_ranges[%d].timezone", index), fmt.Sprintf("invalid timezone: %v", err)}
		}
	}

	// Validate days
	validDays := map[string]bool{
		"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
		"friday": true, "saturday": true, "sunday": true,
	}
	for _, day := range tr.Days {
		if !validDays[strings.ToLower(day)] {
			return &ValidationError{ruleID, fmt.Sprintf("context.time_ranges[%d].days", index), fmt.Sprintf("invalid day '%s'", day)}
		}
	}

	return nil
}

// validateRuleConditions validates rule conditions
func (rp *RuleParser) validateRuleConditions(rule *ModerationRule) error {
	conditions := rule.Conditions

	// Validate length constraints
	if conditions.MinLength != nil && *conditions.MinLength < 0 {
		return &ValidationError{rule.ID, "conditions.min_length", "min_length cannot be negative"}
	}

	if conditions.MaxLength != nil && *conditions.MaxLength < 0 {
		return &ValidationError{rule.ID, "conditions.max_length", "max_length cannot be negative"}
	}

	if conditions.MinLength != nil && conditions.MaxLength != nil && *conditions.MinLength > *conditions.MaxLength {
		return &ValidationError{rule.ID, "conditions", "min_length cannot be greater than max_length"}
	}

	return nil
}

// CompileRule compiles and prepares a rule for execution
func (rp *RuleParser) CompileRule(rule *ModerationRule) error {
	switch rule.Type {
	case RuleTypeRegex:
		return rp.compileRegexRule(rule)
	case RuleTypeKeyword:
		return rp.compileKeywordRule(rule)
	case RuleTypeML:
		return rp.compileMLRule(rule)
	case RuleTypeComposite:
		return rp.compileCompositeRule(rule)
	}

	return nil
}

// compileRegexRule compiles regex patterns for a rule
func (rp *RuleParser) compileRegexRule(rule *ModerationRule) error {
	if rule.Pattern == "" {
		return nil
	}

	compiled, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("failed to compile regex pattern for rule %s: %w", rule.ID, err)
	}

	rule.compiledPattern = compiled
	return nil
}

// compileKeywordRule prepares keywords for efficient matching
func (rp *RuleParser) compileKeywordRule(rule *ModerationRule) error {
	if len(rule.Keywords) == 0 {
		return nil
	}

	// Normalize keywords based on conditions
	normalized := make([]string, 0, len(rule.Keywords))
	
	for _, keyword := range rule.Keywords {
		processed := strings.TrimSpace(keyword)
		
		if rule.Conditions == nil || !rule.Conditions.CaseSensitive {
			processed = strings.ToLower(processed)
		}
		
		if processed != "" {
			normalized = append(normalized, processed)
		}
	}

	rule.normalizedKeywords = normalized
	return nil
}

// compileMLRule prepares ML model configuration
func (rp *RuleParser) compileMLRule(rule *ModerationRule) error {
	// ML rule compilation might involve model loading, validation, etc.
	// For now, just validate the pattern (model name)
	if rule.Pattern == "" {
		return fmt.Errorf("ML rule %s requires a model name in pattern field", rule.ID)
	}

	return nil
}

// compileCompositeRule compiles composite rule components
func (rp *RuleParser) compileCompositeRule(rule *ModerationRule) error {
	// Compile regex pattern if present
	if rule.Pattern != "" {
		if err := rp.compileRegexRule(rule); err != nil {
			return err
		}
	}

	// Compile keywords if present
	if len(rule.Keywords) > 0 {
		if err := rp.compileKeywordRule(rule); err != nil {
			return err
		}
	}

	return nil
}

// ValidateRulesFile validates an entire rules file without parsing
func (rp *RuleParser) ValidateRulesFile(filePath string) error {
	_, err := rp.ParseRulesFromFile(filePath)
	return err
}

// GetRuleStatistics returns statistics about parsed rules
func (rp *RuleParser) GetRuleStatistics(rules []*ModerationRule) map[string]interface{} {
	stats := map[string]interface{}{
		"total_rules":    len(rules),
		"enabled_rules":  0,
		"disabled_rules": 0,
		"rule_types":     make(map[string]int),
		"actions":        make(map[string]int),
		"categories":     make(map[string]int),
		"priorities":     make(map[int]int),
	}

	for _, rule := range rules {
		if rule.Enabled {
			stats["enabled_rules"] = stats["enabled_rules"].(int) + 1
		} else {
			stats["disabled_rules"] = stats["disabled_rules"].(int) + 1
		}

		// Type statistics
		ruleTypes := stats["rule_types"].(map[string]int)
		ruleTypes[rule.Type]++

		// Action statistics
		actions := stats["actions"].(map[string]int)
		actions[rule.Action]++

		// Category statistics
		categories := stats["categories"].(map[string]int)
		categories[rule.Category]++

		// Priority statistics
		priorities := stats["priorities"].(map[int]int)
		priorities[rule.Priority]++
	}

	return stats
}