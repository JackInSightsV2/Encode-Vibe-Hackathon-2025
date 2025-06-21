package optimizer

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ConstraintEngine validates and enforces constraints on rules and experiments
type ConstraintEngine struct {
	globalConstraints *GlobalConstraints
	ruleConstraints   map[RuleType]*RuleTypeConstraints
}

// GlobalConstraints define system-wide limits
type GlobalConstraints struct {
	MaxActiveExperiments      int           `json:"max_active_experiments"`
	MaxRulesPerExperiment     int           `json:"max_rules_per_experiment"`
	MaxExperimentDuration     time.Duration `json:"max_experiment_duration"`
	MinExperimentDuration     time.Duration `json:"min_experiment_duration"`
	MaxConcurrentMutations    int           `json:"max_concurrent_mutations"`
	MaxRuleComplexity         float64       `json:"max_rule_complexity"`
	MinSampleSize             int           `json:"min_sample_size"`
	MaxResourceUsage          float64       `json:"max_resource_usage"`
	RequiredConfidenceLevel   float64       `json:"required_confidence_level"`
	SafetyThresholds          *SafetyThresholds `json:"safety_thresholds"`
}

// SafetyThresholds define safety limits that cannot be violated
type SafetyThresholds struct {
	MinDetectionRate          float64 `json:"min_detection_rate"`
	MaxFalsePositiveRate      float64 `json:"max_false_positive_rate"`
	MaxResponseTimeMs         float64 `json:"max_response_time_ms"`
	MinUserSatisfaction       float64 `json:"min_user_satisfaction"`
	MaxDegradationTolerance   float64 `json:"max_degradation_tolerance"`
	CriticalSecurityPatterns  []string `json:"critical_security_patterns"`
}

// RuleTypeConstraints define constraints specific to each rule type
type RuleTypeConstraints struct {
	MaxParameterCount    int                    `json:"max_parameter_count"`
	RequiredParameters   []string               `json:"required_parameters"`
	ParameterConstraints map[string]*ParameterConstraint `json:"parameter_constraints"`
	MaxExecutionTime     time.Duration          `json:"max_execution_time"`
	AllowedOperations    []string               `json:"allowed_operations"`
	SecurityRestrictions *SecurityRestrictions  `json:"security_restrictions"`
}

// ParameterConstraint defines constraints for individual parameters
type ParameterConstraint struct {
	Type         ParameterType   `json:"type"`
	Required     bool            `json:"required"`
	MinValue     interface{}     `json:"min_value,omitempty"`
	MaxValue     interface{}     `json:"max_value,omitempty"`
	AllowedValues []interface{}  `json:"allowed_values,omitempty"`
	Pattern      string          `json:"pattern,omitempty"`
	MaxLength    int             `json:"max_length,omitempty"`
	MinLength    int             `json:"min_length,omitempty"`
	Validator    func(interface{}) error `json:"-"`
}

// SecurityRestrictions define security-related constraints
type SecurityRestrictions struct {
	ForbiddenPatterns     []string `json:"forbidden_patterns"`
	RequiredSafetyChecks  []string `json:"required_safety_checks"`
	MaxPrivilegeLevel     int      `json:"max_privilege_level"`
	RequireApproval       bool     `json:"require_approval"`
	AuditRequired         bool     `json:"audit_required"`
}

// ConstraintViolation represents a constraint violation
type ConstraintViolation struct {
	Type        string      `json:"type"`
	Rule        string      `json:"rule"`
	Parameter   string      `json:"parameter,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Constraint  string      `json:"constraint"`
	Severity    string      `json:"severity"`
	Message     string      `json:"message"`
	Suggestion  string      `json:"suggestion,omitempty"`
}

// ValidationResult contains the result of constraint validation
type ValidationResult struct {
	Valid       bool                   `json:"valid"`
	Violations  []ConstraintViolation  `json:"violations"`
	Warnings    []ConstraintViolation  `json:"warnings"`
	Score       float64                `json:"score"`
	Suggestions []string               `json:"suggestions"`
}

// NewConstraintEngine creates a new constraint engine
func NewConstraintEngine() *ConstraintEngine {
	return &ConstraintEngine{
		globalConstraints: GetDefaultGlobalConstraints(),
		ruleConstraints:   GetDefaultRuleConstraints(),
	}
}

// GetDefaultGlobalConstraints returns default global constraints
func GetDefaultGlobalConstraints() *GlobalConstraints {
	return &GlobalConstraints{
		MaxActiveExperiments:      10,
		MaxRulesPerExperiment:     50,
		MaxExperimentDuration:     24 * time.Hour,
		MinExperimentDuration:     30 * time.Minute,
		MaxConcurrentMutations:    100,
		MaxRuleComplexity:         10.0,
		MinSampleSize:             1000,
		MaxResourceUsage:          0.8,
		RequiredConfidenceLevel:   0.95,
		SafetyThresholds: &SafetyThresholds{
			MinDetectionRate:         0.90,
			MaxFalsePositiveRate:     0.10,
			MaxResponseTimeMs:        100.0,
			MinUserSatisfaction:      0.7,
			MaxDegradationTolerance:  0.05,
			CriticalSecurityPatterns: []string{
				".*password.*",
				".*secret.*",
				".*key.*",
				".*token.*",
				".*credit.*card.*",
				".*ssn.*",
			},
		},
	}
}

// GetDefaultRuleConstraints returns default constraints for each rule type
func GetDefaultRuleConstraints() map[RuleType]*RuleTypeConstraints {
	return map[RuleType]*RuleTypeConstraints{
		RuleTypeRegex: {
			MaxParameterCount:    5,
			RequiredParameters:   []string{"pattern"},
			MaxExecutionTime:     100 * time.Millisecond,
			AllowedOperations:    []string{"match", "find", "replace"},
			ParameterConstraints: map[string]*ParameterConstraint{
				"pattern": {
					Type:      ParameterTypeString,
					Required:  true,
					MinLength: 1,
					MaxLength: 1000,
					Validator: validateRegexPattern,
				},
				"flags": {
					Type:          ParameterTypeString,
					Required:      false,
					MaxLength:     10,
					AllowedValues: []interface{}{"i", "g", "m", "s", "u", "ig", "im", "gm", "igm"},
				},
				"timeout_ms": {
					Type:     ParameterTypeInt,
					Required: false,
					MinValue: 1,
					MaxValue: 1000,
				},
			},
			SecurityRestrictions: &SecurityRestrictions{
				ForbiddenPatterns: []string{
					".*\\(\\?\\{.*", // No code execution
					".*\\\\e.*",     // No escape sequences
					".*eval.*",      // No eval patterns
				},
				RequiredSafetyChecks:  []string{"catastrophic_backtracking", "memory_usage"},
				MaxPrivilegeLevel:     1,
				RequireApproval:       false,
				AuditRequired:         true,
			},
		},
		RuleTypeSemantic: {
			MaxParameterCount:  8,
			RequiredParameters: []string{"threshold"},
			MaxExecutionTime:   500 * time.Millisecond,
			AllowedOperations:  []string{"classify", "score", "analyze"},
			ParameterConstraints: map[string]*ParameterConstraint{
				"threshold": {
					Type:     ParameterTypeFloat,
					Required: true,
					MinValue: 0.0,
					MaxValue: 1.0,
				},
				"model": {
					Type:          ParameterTypeString,
					Required:      false,
					AllowedValues: []interface{}{"bert", "roberta", "distilbert", "gpt", "claude"},
				},
				"context_window": {
					Type:     ParameterTypeInt,
					Required: false,
					MinValue: 10,
					MaxValue: 1000,
				},
				"temperature": {
					Type:     ParameterTypeFloat,
					Required: false,
					MinValue: 0.0,
					MaxValue: 2.0,
				},
			},
			SecurityRestrictions: &SecurityRestrictions{
				ForbiddenPatterns:     []string{".*system.*prompt.*", ".*jailbreak.*"},
				RequiredSafetyChecks:  []string{"prompt_injection", "model_safety"},
				MaxPrivilegeLevel:     2,
				RequireApproval:       true,
				AuditRequired:         true,
			},
		},
		RuleTypePII: {
			MaxParameterCount:  6,
			RequiredParameters: []string{"pii_types"},
			MaxExecutionTime:   200 * time.Millisecond,
			AllowedOperations:  []string{"detect", "mask", "classify"},
			ParameterConstraints: map[string]*ParameterConstraint{
				"pii_types": {
					Type:          ParameterTypeStringArray,
					Required:      true,
					AllowedValues: []interface{}{"email", "phone", "ssn", "credit_card", "address", "name", "ip_address"},
				},
				"sensitivity": {
					Type:     ParameterTypeFloat,
					Required: false,
					MinValue: 0.0,
					MaxValue: 1.0,
				},
				"mask_char": {
					Type:      ParameterTypeString,
					Required:  false,
					MaxLength: 1,
				},
			},
			SecurityRestrictions: &SecurityRestrictions{
				ForbiddenPatterns:    []string{},
				RequiredSafetyChecks: []string{"data_leakage", "privacy_compliance"},
				MaxPrivilegeLevel:    2,
				RequireApproval:      false,
				AuditRequired:        true,
			},
		},
		RuleTypeHybrid: {
			MaxParameterCount:  15,
			RequiredParameters: []string{"combination_strategy", "component_rules"},
			MaxExecutionTime:   1 * time.Second,
			AllowedOperations:  []string{"combine", "evaluate", "score"},
			ParameterConstraints: map[string]*ParameterConstraint{
				"combination_strategy": {
					Type:          ParameterTypeString,
					Required:      true,
					AllowedValues: []interface{}{"AND", "OR", "WEIGHTED", "THRESHOLD", "PIPELINE"},
				},
				"threshold": {
					Type:     ParameterTypeFloat,
					Required: false,
					MinValue: 0.0,
					MaxValue: 1.0,
				},
				"max_components": {
					Type:     ParameterTypeInt,
					Required: false,
					MinValue: 2,
					MaxValue: 5,
				},
			},
			SecurityRestrictions: &SecurityRestrictions{
				ForbiddenPatterns:    []string{},
				RequiredSafetyChecks: []string{"component_validation", "complexity_check"},
				MaxPrivilegeLevel:    3,
				RequireApproval:      true,
				AuditRequired:        true,
			},
		},
		RuleTypeCustom: {
			MaxParameterCount:  10,
			RequiredParameters: []string{"implementation"},
			MaxExecutionTime:   300 * time.Millisecond,
			AllowedOperations:  []string{"execute", "validate"},
			ParameterConstraints: map[string]*ParameterConstraint{
				"implementation": {
					Type:     ParameterTypeString,
					Required: true,
					Validator: validateCustomImplementation,
				},
			},
			SecurityRestrictions: &SecurityRestrictions{
				ForbiddenPatterns: []string{
					".*exec.*",
					".*eval.*",
					".*system.*",
					".*import.*os.*",
					".*subprocess.*",
				},
				RequiredSafetyChecks: []string{"code_injection", "privilege_escalation", "resource_limits"},
				MaxPrivilegeLevel:    1,
				RequireApproval:      true,
				AuditRequired:        true,
			},
		},
	}
}

// ValidateRule validates a rule against all applicable constraints
func (ce *ConstraintEngine) ValidateRule(rule Rule) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Violations:  make([]ConstraintViolation, 0),
		Warnings:    make([]ConstraintViolation, 0),
		Suggestions: make([]string, 0),
	}
	
	// Validate against rule type constraints
	if constraints, exists := ce.ruleConstraints[rule.Type]; exists {
		ce.validateRuleTypeConstraints(rule, constraints, result)
	}
	
	// Validate against global constraints
	ce.validateGlobalConstraints(rule, result)
	
	// Calculate validation score
	result.Score = ce.calculateValidationScore(result)
	
	// Set overall validity
	result.Valid = len(result.Violations) == 0
	
	return result
}

// ValidateExperiment validates an experiment configuration
func (ce *ConstraintEngine) ValidateExperiment(config ExperimentConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Violations:  make([]ConstraintViolation, 0),
		Warnings:    make([]ConstraintViolation, 0),
		Suggestions: make([]string, 0),
	}
	
	// Validate experiment duration
	if config.Duration > ce.globalConstraints.MaxExperimentDuration {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "experiment",
			Rule:       "duration",
			Value:      config.Duration,
			Constraint: "max_experiment_duration",
			Severity:   "error",
			Message:    fmt.Sprintf("Experiment duration %v exceeds maximum %v", config.Duration, ce.globalConstraints.MaxExperimentDuration),
			Suggestion: fmt.Sprintf("Reduce duration to %v or less", ce.globalConstraints.MaxExperimentDuration),
		})
	}
	
	if config.Duration < ce.globalConstraints.MinExperimentDuration {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "experiment",
			Rule:       "duration",
			Value:      config.Duration,
			Constraint: "min_experiment_duration",
			Severity:   "error",
			Message:    fmt.Sprintf("Experiment duration %v is below minimum %v", config.Duration, ce.globalConstraints.MinExperimentDuration),
			Suggestion: fmt.Sprintf("Increase duration to at least %v", ce.globalConstraints.MinExperimentDuration),
		})
	}
	
	// Validate sample size
	if config.MinSampleSize < ce.globalConstraints.MinSampleSize {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "experiment",
			Rule:       "sample_size",
			Value:      config.MinSampleSize,
			Constraint: "min_sample_size",
			Severity:   "error",
			Message:    fmt.Sprintf("Sample size %d is below minimum %d", config.MinSampleSize, ce.globalConstraints.MinSampleSize),
			Suggestion: fmt.Sprintf("Increase sample size to at least %d", ce.globalConstraints.MinSampleSize),
		})
	}
	
	// Validate rules count
	totalRules := 1 + len(config.Variants) // control + variants
	if totalRules > ce.globalConstraints.MaxRulesPerExperiment {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "experiment",
			Rule:       "rules_count",
			Value:      totalRules,
			Constraint: "max_rules_per_experiment",
			Severity:   "error",
			Message:    fmt.Sprintf("Total rules %d exceeds maximum %d", totalRules, ce.globalConstraints.MaxRulesPerExperiment),
			Suggestion: fmt.Sprintf("Reduce number of variants to stay under %d total rules", ce.globalConstraints.MaxRulesPerExperiment),
		})
	}
	
	// Validate control rule
	controlValidation := ce.ValidateRule(config.Control)
	if !controlValidation.Valid {
		for _, violation := range controlValidation.Violations {
			violation.Rule = "control_" + violation.Rule
			result.Violations = append(result.Violations, violation)
		}
	}
	
	// Validate variant rules
	for i, variant := range config.Variants {
		variantValidation := ce.ValidateRule(variant)
		if !variantValidation.Valid {
			for _, violation := range variantValidation.Violations {
				violation.Rule = fmt.Sprintf("variant_%d_%s", i, violation.Rule)
				result.Violations = append(result.Violations, violation)
			}
		}
	}
	
	// Validate traffic distribution
	ce.validateTrafficSplit(config.Traffic, result)
	
	// Validate safety thresholds
	ce.validateSafetyThresholds(config, result)
	
	result.Score = ce.calculateValidationScore(result)
	result.Valid = len(result.Violations) == 0
	
	return result
}

// validateRuleTypeConstraints validates rule against type-specific constraints
func (ce *ConstraintEngine) validateRuleTypeConstraints(rule Rule, constraints *RuleTypeConstraints, result *ValidationResult) {
	// Validate parameter count
	if len(rule.Parameters) > constraints.MaxParameterCount {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "rule",
			Rule:       rule.ID,
			Constraint: "max_parameter_count",
			Severity:   "error",
			Message:    fmt.Sprintf("Rule has %d parameters, maximum allowed is %d", len(rule.Parameters), constraints.MaxParameterCount),
		})
	}
	
	// Validate required parameters
	for _, required := range constraints.RequiredParameters {
		if _, exists := rule.Parameters[required]; !exists {
			result.Violations = append(result.Violations, ConstraintViolation{
				Type:       "rule",
				Rule:       rule.ID,
				Parameter:  required,
				Constraint: "required_parameter",
				Severity:   "error",
				Message:    fmt.Sprintf("Required parameter '%s' is missing", required),
			})
		}
	}
	
	// Validate individual parameters
	for paramName, paramValue := range rule.Parameters {
		if paramConstraint, exists := constraints.ParameterConstraints[paramName]; exists {
			ce.validateParameter(rule.ID, paramName, paramValue, paramConstraint, result)
		}
	}
	
	// Validate security restrictions
	if constraints.SecurityRestrictions != nil {
		ce.validateSecurityRestrictions(rule, constraints.SecurityRestrictions, result)
	}
}

// validateParameter validates a single parameter against its constraints
func (ce *ConstraintEngine) validateParameter(ruleID, paramName string, value interface{}, constraint *ParameterConstraint, result *ValidationResult) {
	// Type validation would go here based on constraint.Type
	
	// Custom validator
	if constraint.Validator != nil {
		if err := constraint.Validator(value); err != nil {
			result.Violations = append(result.Violations, ConstraintViolation{
				Type:       "parameter",
				Rule:       ruleID,
				Parameter:  paramName,
				Value:      value,
				Constraint: "custom_validator",
				Severity:   "error",
				Message:    fmt.Sprintf("Parameter validation failed: %v", err),
			})
		}
	}
	
	// String-specific validations
	if strValue, ok := value.(string); ok {
		if constraint.MinLength > 0 && len(strValue) < constraint.MinLength {
			result.Violations = append(result.Violations, ConstraintViolation{
				Type:       "parameter",
				Rule:       ruleID,
				Parameter:  paramName,
				Value:      value,
				Constraint: "min_length",
				Severity:   "error",
				Message:    fmt.Sprintf("String length %d is below minimum %d", len(strValue), constraint.MinLength),
			})
		}
		
		if constraint.MaxLength > 0 && len(strValue) > constraint.MaxLength {
			result.Violations = append(result.Violations, ConstraintViolation{
				Type:       "parameter",
				Rule:       ruleID,
				Parameter:  paramName,
				Value:      value,
				Constraint: "max_length",
				Severity:   "error",
				Message:    fmt.Sprintf("String length %d exceeds maximum %d", len(strValue), constraint.MaxLength),
			})
		}
		
		if constraint.Pattern != "" {
			if matched, _ := regexp.MatchString(constraint.Pattern, strValue); !matched {
				result.Violations = append(result.Violations, ConstraintViolation{
					Type:       "parameter",
					Rule:       ruleID,
					Parameter:  paramName,
					Value:      value,
					Constraint: "pattern",
					Severity:   "error",
					Message:    fmt.Sprintf("String does not match required pattern: %s", constraint.Pattern),
				})
			}
		}
	}
	
	// Allowed values validation
	if len(constraint.AllowedValues) > 0 {
		allowed := false
		for _, allowedValue := range constraint.AllowedValues {
			if value == allowedValue {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Violations = append(result.Violations, ConstraintViolation{
				Type:       "parameter",
				Rule:       ruleID,
				Parameter:  paramName,
				Value:      value,
				Constraint: "allowed_values",
				Severity:   "error",
				Message:    fmt.Sprintf("Value is not in allowed list: %v", constraint.AllowedValues),
			})
		}
	}
}

// validateSecurityRestrictions validates security-related constraints
func (ce *ConstraintEngine) validateSecurityRestrictions(rule Rule, restrictions *SecurityRestrictions, result *ValidationResult) {
	// Check forbidden patterns
	for _, forbiddenPattern := range restrictions.ForbiddenPatterns {
		for paramName, paramValue := range rule.Parameters {
			if strValue, ok := paramValue.(string); ok {
				if matched, _ := regexp.MatchString(forbiddenPattern, strValue); matched {
					result.Violations = append(result.Violations, ConstraintViolation{
						Type:       "security",
						Rule:       rule.ID,
						Parameter:  paramName,
						Value:      paramValue,
						Constraint: "forbidden_pattern",
						Severity:   "critical",
						Message:    fmt.Sprintf("Parameter contains forbidden pattern: %s", forbiddenPattern),
					})
				}
			}
		}
	}
}

// validateGlobalConstraints validates rule against global constraints
func (ce *ConstraintEngine) validateGlobalConstraints(rule Rule, result *ValidationResult) {
	// Calculate rule complexity
	complexity := ce.calculateRuleComplexity(rule)
	if complexity > ce.globalConstraints.MaxRuleComplexity {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "rule",
			Rule:       rule.ID,
			Constraint: "max_rule_complexity",
			Severity:   "warning",
			Message:    fmt.Sprintf("Rule complexity %.2f exceeds recommended maximum %.2f", complexity, ce.globalConstraints.MaxRuleComplexity),
			Suggestion: "Consider simplifying the rule or breaking it into smaller components",
		})
	}
	
	// Check critical security patterns
	for _, pattern := range ce.globalConstraints.SafetyThresholds.CriticalSecurityPatterns {
		for paramName, paramValue := range rule.Parameters {
			if strValue, ok := paramValue.(string); ok {
				if matched, _ := regexp.MatchString(pattern, strValue); matched {
					result.Warnings = append(result.Warnings, ConstraintViolation{
						Type:       "security",
						Rule:       rule.ID,
						Parameter:  paramName,
						Constraint: "critical_security_pattern",
						Severity:   "warning",
						Message:    fmt.Sprintf("Parameter may contain sensitive data pattern: %s", pattern),
						Suggestion: "Review parameter to ensure no sensitive data is exposed",
					})
				}
			}
		}
	}
}

// validateTrafficSplit validates traffic distribution
func (ce *ConstraintEngine) validateTrafficSplit(traffic TrafficSplit, result *ValidationResult) {
	total := traffic.Control
	for _, weight := range traffic.Variants {
		total += weight
	}
	
	if total < 0.99 || total > 1.01 {
		result.Violations = append(result.Violations, ConstraintViolation{
			Type:       "traffic",
			Constraint: "traffic_sum",
			Severity:   "error",
			Message:    fmt.Sprintf("Traffic weights must sum to 1.0, got %.3f", total),
			Suggestion: "Adjust traffic weights to sum to exactly 1.0",
		})
	}
	
	if traffic.Control < 0.1 {
		result.Warnings = append(result.Warnings, ConstraintViolation{
			Type:       "traffic",
			Constraint: "min_control_traffic",
			Severity:   "warning",
			Message:    fmt.Sprintf("Control group has very low traffic: %.1f%%", traffic.Control*100),
			Suggestion: "Consider allocating at least 10% traffic to control group",
		})
	}
}

// validateSafetyThresholds validates experiment against safety thresholds
func (ce *ConstraintEngine) validateSafetyThresholds(config ExperimentConfig, result *ValidationResult) {
	// Check if experiment has safety monitoring enabled
	if config.EarlyStoppingEnabled {
		if config.MinDetectionImprovement < 0.01 {
			result.Warnings = append(result.Warnings, ConstraintViolation{
				Type:       "safety",
				Constraint: "min_detection_improvement",
				Severity:   "warning",
				Message:    "Very low minimum detection improvement may lead to false positives in early stopping",
				Suggestion: "Consider setting minimum improvement to at least 1%",
			})
		}
		
		if config.MaxDegradationTolerance > ce.globalConstraints.SafetyThresholds.MaxDegradationTolerance {
			result.Violations = append(result.Violations, ConstraintViolation{
				Type:       "safety",
				Constraint: "max_degradation_tolerance",
				Severity:   "error",
				Message:    fmt.Sprintf("Degradation tolerance %.2f%% exceeds safety limit %.2f%%", 
					config.MaxDegradationTolerance*100, ce.globalConstraints.SafetyThresholds.MaxDegradationTolerance*100),
				Suggestion: fmt.Sprintf("Reduce degradation tolerance to %.2f%% or less", 
					ce.globalConstraints.SafetyThresholds.MaxDegradationTolerance*100),
			})
		}
	} else {
		result.Warnings = append(result.Warnings, ConstraintViolation{
			Type:       "safety",
			Constraint: "early_stopping_disabled",
			Severity:   "warning",
			Message:    "Early stopping is disabled - experiment may continue even with poor performance",
			Suggestion: "Consider enabling early stopping for safety",
		})
	}
}

// calculateRuleComplexity calculates a complexity score for a rule
func (ce *ConstraintEngine) calculateRuleComplexity(rule Rule) float64 {
	complexity := 0.0
	
	// Base complexity from rule type
	switch rule.Type {
	case RuleTypeRegex:
		complexity += 1.0
	case RuleTypeSemantic:
		complexity += 2.0
	case RuleTypePII:
		complexity += 1.5
	case RuleTypeHybrid:
		complexity += 3.0
	case RuleTypeCustom:
		complexity += 4.0
	}
	
	// Add complexity from parameters
	complexity += float64(len(rule.Parameters)) * 0.2
	
	// Type-specific complexity
	switch rule.Type {
	case RuleTypeRegex:
		if pattern, ok := rule.Parameters["pattern"].(string); ok {
			complexity += ce.calculatePatternComplexity(pattern)
		}
	case RuleTypeHybrid:
		if components, ok := rule.Parameters["component_rules"].([]interface{}); ok {
			complexity += float64(len(components)) * 0.5
		}
	}
	
	return complexity
}

// calculatePatternComplexity calculates complexity of a regex pattern
func (ce *ConstraintEngine) calculatePatternComplexity(pattern string) float64 {
	complexity := 0.0
	
	// Length factor
	complexity += float64(len(pattern)) / 100.0
	
	// Special character count
	specialChars := []string{"*", "+", "?", "{", "}", "[", "]", "(", ")", "|", "\\", "^", "$"}
	for _, char := range specialChars {
		complexity += float64(strings.Count(pattern, char)) * 0.1
	}
	
	// Nested groups
	complexity += float64(strings.Count(pattern, "(")) * 0.2
	
	// Quantifiers
	if strings.Contains(pattern, "{") {
		complexity += 0.3
	}
	
	// Lookaheads/lookbehinds
	if strings.Contains(pattern, "?=") || strings.Contains(pattern, "?!") {
		complexity += 0.5
	}
	
	return complexity
}

// calculateValidationScore calculates an overall validation score
func (ce *ConstraintEngine) calculateValidationScore(result *ValidationResult) float64 {
	score := 1.0
	
	// Deduct for violations
	for _, violation := range result.Violations {
		switch violation.Severity {
		case "critical":
			score -= 0.5
		case "error":
			score -= 0.2
		case "warning":
			score -= 0.05
		}
	}
	
	// Deduct for warnings
	for range result.Warnings {
		score -= 0.02
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// Custom validation functions

func validateRegexPattern(value interface{}) error {
	pattern, ok := value.(string)
	if !ok {
		return fmt.Errorf("pattern must be a string")
	}
	
	// Compile to check syntax
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	// Check for catastrophic backtracking patterns
	if strings.Contains(pattern, "(.*)+") || strings.Contains(pattern, "(a+)+") {
		return fmt.Errorf("pattern may cause catastrophic backtracking")
	}
	
	return nil
}

func validateCustomImplementation(value interface{}) error {
	implementation, ok := value.(string)
	if !ok {
		return fmt.Errorf("implementation must be a string")
	}
	
	// Check for dangerous functions
	forbiddenPatterns := []string{
		"exec", "eval", "system", "subprocess", "__import__",
		"open", "file", "input", "raw_input",
	}
	
	lowerImpl := strings.ToLower(implementation)
	for _, forbidden := range forbiddenPatterns {
		if strings.Contains(lowerImpl, forbidden) {
			return fmt.Errorf("implementation contains forbidden function: %s", forbidden)
		}
	}
	
	return nil
}