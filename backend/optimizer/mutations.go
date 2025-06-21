package optimizer

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// MutationEngine handles rule mutations for genetic algorithms
type MutationEngine struct {
	mutationRate    float64
	maxMutations    int
	rng             *rand.Rand
	patternDatabase []string
}

// MutationType represents different types of mutations
type MutationType string

const (
	MutationTypeAdd    MutationType = "add"
	MutationTypeRemove MutationType = "remove"
	MutationTypeModify MutationType = "modify"
	MutationTypeSwap   MutationType = "swap"
)

// MutationConfig controls mutation behavior
type MutationConfig struct {
	Rate            float64          `json:"rate"`
	MaxMutations    int              `json:"max_mutations"`
	AllowedTypes    []MutationType   `json:"allowed_types"`
	TypeWeights     map[MutationType]float64 `json:"type_weights"`
	RuleConstraints *RuleConstraints `json:"rule_constraints"`
}

// RuleConstraints define limits for rule mutations
type RuleConstraints struct {
	MaxPatternLength     int     `json:"max_pattern_length"`
	MinPatternLength     int     `json:"min_pattern_length"`
	MaxThreshold         float64 `json:"max_threshold"`
	MinThreshold         float64 `json:"min_threshold"`
	AllowedPIITypes      []string `json:"allowed_pii_types"`
	AllowedSemanticModels []string `json:"allowed_semantic_models"`
	MaxCombinedRules     int     `json:"max_combined_rules"`
}

// NewMutationEngine creates a new mutation engine
func NewMutationEngine(config MutationConfig) *MutationEngine {
	if config.Rate == 0 {
		config.Rate = 0.1
	}
	if config.MaxMutations == 0 {
		config.MaxMutations = 3
	}
	
	engine := &MutationEngine{
		mutationRate: config.Rate,
		maxMutations: config.MaxMutations,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
		patternDatabase: getCommonPatterns(),
	}
	
	return engine
}

// GetDefaultMutationConfig returns default mutation configuration
func GetDefaultMutationConfig() MutationConfig {
	return MutationConfig{
		Rate:         0.1,
		MaxMutations: 3,
		AllowedTypes: []MutationType{MutationTypeAdd, MutationTypeRemove, MutationTypeModify, MutationTypeSwap},
		TypeWeights: map[MutationType]float64{
			MutationTypeAdd:    0.3,
			MutationTypeRemove: 0.2,
			MutationTypeModify: 0.4,
			MutationTypeSwap:   0.1,
		},
		RuleConstraints: &RuleConstraints{
			MaxPatternLength:     500,
			MinPatternLength:     3,
			MaxThreshold:         1.0,
			MinThreshold:         0.0,
			AllowedPIITypes:      []string{"email", "phone", "ssn", "credit_card", "address", "name"},
			AllowedSemanticModels: []string{"bert", "roberta", "distilbert", "gpt"},
			MaxCombinedRules:     5,
		},
	}
}

// MutateRule applies mutations to a rule and returns a new variant
func (me *MutationEngine) MutateRule(baseRule Rule, config MutationConfig) (Rule, error) {
	mutated := baseRule.Clone()
	mutated.ID = generateRuleID()
	mutated.Name = fmt.Sprintf("%s_mutated_%d", baseRule.Name, time.Now().Unix())
	mutated.ParentID = baseRule.ID
	
	// Determine number of mutations to apply
	numMutations := 1
	if me.rng.Float64() < config.Rate {
		numMutations = me.rng.Intn(config.MaxMutations) + 1
	}
	
	// Apply mutations based on rule type
	for i := 0; i < numMutations; i++ {
		switch mutated.Type {
		case RuleTypeRegex:
			if err := me.mutateRegexRule(&mutated, config); err != nil {
				return mutated, fmt.Errorf("failed to mutate regex rule: %w", err)
			}
		case RuleTypeSemantic:
			if err := me.mutateSemanticRule(&mutated, config); err != nil {
				return mutated, fmt.Errorf("failed to mutate semantic rule: %w", err)
			}
		case RuleTypePII:
			if err := me.mutatePIIRule(&mutated, config); err != nil {
				return mutated, fmt.Errorf("failed to mutate PII rule: %w", err)
			}
		case RuleTypeHybrid:
			if err := me.mutateHybridRule(&mutated, config); err != nil {
				return mutated, fmt.Errorf("failed to mutate hybrid rule: %w", err)
			}
		}
	}
	
	return mutated, nil
}

// mutateRegexRule applies mutations to regex rules
func (me *MutationEngine) mutateRegexRule(rule *Rule, config MutationConfig) error {
	pattern, ok := rule.Parameters["pattern"].(string)
	if !ok {
		return fmt.Errorf("invalid pattern parameter")
	}
	
	mutationType := me.selectMutationType(config.TypeWeights)
	
	switch mutationType {
	case MutationTypeAdd:
		// Add quantifiers, anchors, or character classes
		newPattern := me.addRegexElements(pattern, config.RuleConstraints)
		rule.Parameters["pattern"] = newPattern
		
	case MutationTypeRemove:
		// Remove optional elements
		newPattern := me.removeRegexElements(pattern)
		rule.Parameters["pattern"] = newPattern
		
	case MutationTypeModify:
		// Modify existing elements
		newPattern := me.modifyRegexElements(pattern, config.RuleConstraints)
		rule.Parameters["pattern"] = newPattern
		
	case MutationTypeSwap:
		// Replace with similar pattern from database
		newPattern := me.swapRegexPattern(pattern)
		rule.Parameters["pattern"] = newPattern
	}
	
	// Mutate flags occasionally
	if me.rng.Float64() < 0.3 {
		me.mutateRegexFlags(rule)
	}
	
	return nil
}

// mutateSemanticRule applies mutations to semantic rules
func (me *MutationEngine) mutateSemanticRule(rule *Rule, config MutationConfig) error {
	// Mutate threshold
	if threshold, ok := rule.Parameters["threshold"].(float64); ok {
		variation := (me.rng.Float64() - 0.5) * 0.2 // ±10% variation
		newThreshold := threshold + variation
		
		// Constrain to valid range
		if newThreshold < config.RuleConstraints.MinThreshold {
			newThreshold = config.RuleConstraints.MinThreshold
		}
		if newThreshold > config.RuleConstraints.MaxThreshold {
			newThreshold = config.RuleConstraints.MaxThreshold
		}
		
		rule.Parameters["threshold"] = newThreshold
	}
	
	// Mutate model occasionally
	if me.rng.Float64() < 0.2 {
		if currentModel, ok := rule.Parameters["model"].(string); ok {
			availableModels := config.RuleConstraints.AllowedSemanticModels
			if len(availableModels) > 1 {
				// Select different model
				newModel := currentModel
				for newModel == currentModel {
					newModel = availableModels[me.rng.Intn(len(availableModels))]
				}
				rule.Parameters["model"] = newModel
			}
		}
	}
	
	// Mutate context window or other semantic parameters
	if contextWindow, ok := rule.Parameters["context_window"].(int); ok {
		variation := me.rng.Intn(21) - 10 // ±10 tokens
		newWindow := contextWindow + variation
		if newWindow < 10 {
			newWindow = 10
		}
		if newWindow > 500 {
			newWindow = 500
		}
		rule.Parameters["context_window"] = newWindow
	}
	
	return nil
}

// mutatePIIRule applies mutations to PII detection rules
func (me *MutationEngine) mutatePIIRule(rule *Rule, config MutationConfig) error {
	// Mutate PII types
	if currentTypes, ok := rule.Parameters["pii_types"].([]string); ok {
		newTypes := make([]string, len(currentTypes))
		copy(newTypes, currentTypes)
		
		mutationType := me.selectMutationType(config.TypeWeights)
		allowedTypes := config.RuleConstraints.AllowedPIITypes
		
		switch mutationType {
		case MutationTypeAdd:
			// Add a new PII type
			for _, t := range allowedTypes {
				found := false
				for _, existing := range newTypes {
					if existing == t {
						found = true
						break
					}
				}
				if !found && me.rng.Float64() < 0.3 {
					newTypes = append(newTypes, t)
					break
				}
			}
			
		case MutationTypeRemove:
			// Remove a PII type (but keep at least one)
			if len(newTypes) > 1 {
				removeIndex := me.rng.Intn(len(newTypes))
				newTypes = append(newTypes[:removeIndex], newTypes[removeIndex+1:]...)
			}
			
		case MutationTypeSwap:
			// Replace one type with another
			if len(allowedTypes) > len(newTypes) {
				replaceIndex := me.rng.Intn(len(newTypes))
				// Find a type not currently in the list
				for _, t := range allowedTypes {
					found := false
					for _, existing := range newTypes {
						if existing == t {
							found = true
							break
						}
					}
					if !found {
						newTypes[replaceIndex] = t
						break
					}
				}
			}
		}
		
		rule.Parameters["pii_types"] = newTypes
	}
	
	// Mutate sensitivity
	if sensitivity, ok := rule.Parameters["sensitivity"].(float64); ok {
		variation := (me.rng.Float64() - 0.5) * 0.3 // ±15% variation
		newSensitivity := sensitivity + variation
		
		if newSensitivity < config.RuleConstraints.MinThreshold {
			newSensitivity = config.RuleConstraints.MinThreshold
		}
		if newSensitivity > config.RuleConstraints.MaxThreshold {
			newSensitivity = config.RuleConstraints.MaxThreshold
		}
		
		rule.Parameters["sensitivity"] = newSensitivity
	}
	
	return nil
}

// mutateHybridRule applies mutations to hybrid rules
func (me *MutationEngine) mutateHybridRule(rule *Rule, config MutationConfig) error {
	// Hybrid rules combine multiple techniques, so mutate each component
	
	// Mutate regex component if present
	if regexPattern, ok := rule.Parameters["regex_pattern"].(string); ok {
		me.mutateRegexFlags(rule)
		newPattern := me.modifyRegexElements(regexPattern, config.RuleConstraints)
		rule.Parameters["regex_pattern"] = newPattern
	}
	
	// Mutate semantic component if present
	if semanticThreshold, ok := rule.Parameters["semantic_threshold"].(float64); ok {
		variation := (me.rng.Float64() - 0.5) * 0.15 // ±7.5% variation
		newThreshold := semanticThreshold + variation
		
		if newThreshold < config.RuleConstraints.MinThreshold {
			newThreshold = config.RuleConstraints.MinThreshold
		}
		if newThreshold > config.RuleConstraints.MaxThreshold {
			newThreshold = config.RuleConstraints.MaxThreshold
		}
		
		rule.Parameters["semantic_threshold"] = newThreshold
	}
	
	// Mutate combination strategy
	if strategy, ok := rule.Parameters["combination_strategy"].(string); ok {
		strategies := []string{"AND", "OR", "WEIGHTED", "THRESHOLD"}
		if me.rng.Float64() < 0.15 {
			newStrategy := strategy
			for newStrategy == strategy {
				newStrategy = strategies[me.rng.Intn(len(strategies))]
			}
			rule.Parameters["combination_strategy"] = newStrategy
		}
	}
	
	return nil
}

// selectMutationType selects a mutation type based on weights
func (me *MutationEngine) selectMutationType(weights map[MutationType]float64) MutationType {
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}
	
	r := me.rng.Float64() * totalWeight
	current := 0.0
	
	for mutType, weight := range weights {
		current += weight
		if r <= current {
			return mutType
		}
	}
	
	// Default fallback
	return MutationTypeModify
}

// addRegexElements adds elements to a regex pattern
func (me *MutationEngine) addRegexElements(pattern string, constraints *RuleConstraints) string {
	if len(pattern) >= constraints.MaxPatternLength {
		return pattern
	}
	
	additions := []string{
		"?",     // Optional
		"+",     // One or more
		"*",     // Zero or more
		"\\b",   // Word boundary
		"(?i)",  // Case insensitive flag
		"\\s*",  // Optional whitespace
	}
	
	// Add random element
	addition := additions[me.rng.Intn(len(additions))]
	
	// Insert at random position (but not breaking existing structure)
	if me.rng.Float64() < 0.7 {
		return pattern + addition
	} else {
		insertPos := me.rng.Intn(len(pattern) + 1)
		return pattern[:insertPos] + addition + pattern[insertPos:]
	}
}

// removeRegexElements removes optional elements from a regex pattern
func (me *MutationEngine) removeRegexElements(pattern string) string {
	// Remove optional quantifiers and some modifiers
	removals := []string{"?", "*", "\\s*", "\\b", "(?i)"}
	
	for _, removal := range removals {
		if strings.Contains(pattern, removal) && me.rng.Float64() < 0.3 {
			pattern = strings.Replace(pattern, removal, "", 1)
			break
		}
	}
	
	return pattern
}

// modifyRegexElements modifies existing elements in a regex pattern
func (me *MutationEngine) modifyRegexElements(pattern string, constraints *RuleConstraints) string {
	// Simple character class substitutions
	substitutions := map[string]string{
		"\\d":     "[0-9]",
		"[0-9]":   "\\d",
		"\\w":     "[a-zA-Z0-9_]",
		"\\s":     "[ \\t\\n\\r]",
		".":       "[^\\n]",
		"[a-z]":   "[a-zA-Z]",
		"[A-Z]":   "[a-zA-Z]",
	}
	
	for old, new := range substitutions {
		if strings.Contains(pattern, old) && me.rng.Float64() < 0.4 {
			pattern = strings.Replace(pattern, old, new, 1)
			break
		}
	}
	
	return pattern
}

// swapRegexPattern replaces pattern with similar one from database
func (me *MutationEngine) swapRegexPattern(pattern string) string {
	if len(me.patternDatabase) == 0 {
		return pattern
	}
	
	// Select random pattern from database
	newPattern := me.patternDatabase[me.rng.Intn(len(me.patternDatabase))]
	
	// Combine with original pattern occasionally
	if me.rng.Float64() < 0.3 {
		if me.rng.Float64() < 0.5 {
			return pattern + "|" + newPattern
		} else {
			return newPattern + "|" + pattern
		}
	}
	
	return newPattern
}

// mutateRegexFlags mutates regex flags
func (me *MutationEngine) mutateRegexFlags(rule *Rule) {
	if flags, ok := rule.Parameters["flags"].(string); ok {
		allFlags := "igm"
		newFlags := flags
		
		// Add or remove flags
		for _, flag := range allFlags {
			flagStr := string(flag)
			if strings.Contains(flags, flagStr) {
				// Remove flag occasionally
				if me.rng.Float64() < 0.2 {
					newFlags = strings.Replace(newFlags, flagStr, "", -1)
				}
			} else {
				// Add flag occasionally
				if me.rng.Float64() < 0.3 {
					newFlags += flagStr
				}
			}
		}
		
		rule.Parameters["flags"] = newFlags
	}
}

// getCommonPatterns returns common regex patterns for mutations
func getCommonPatterns() []string {
	return []string{
		"[a-zA-Z]+",
		"\\d+",
		"\\w+",
		"[a-zA-Z0-9]+",
		"\\b\\w+\\b",
		"[^\\s]+",
		"\\S+",
		".*",
		".+",
		"[a-z]+",
		"[A-Z]+",
		"\\d{1,3}",
		"\\w{3,}",
		"[a-zA-Z]{2,}",
		"\\b[A-Z]\\w*\\b",
		"\\d{4}",
		"[a-zA-Z0-9._-]+",
		"\\w+@\\w+",
		"\\d+-\\d+-\\d+",
		"[a-zA-Z]+\\s+[a-zA-Z]+",
	}
}

// GenerateRuleVariants generates multiple variants of a rule using mutations
func (me *MutationEngine) GenerateRuleVariants(baseRule Rule, count int, config MutationConfig) ([]Rule, error) {
	variants := make([]Rule, 0, count)
	
	for i := 0; i < count; i++ {
		variant, err := me.MutateRule(baseRule, config)
		if err != nil {
			return variants, fmt.Errorf("failed to generate variant %d: %w", i, err)
		}
		
		// Validate the variant
		if err := me.validateRule(variant, config.RuleConstraints); err != nil {
			// Skip invalid variants and try again
			i--
			continue
		}
		
		variants = append(variants, variant)
	}
	
	return variants, nil
}

// validateRule validates that a mutated rule meets constraints
func (me *MutationEngine) validateRule(rule Rule, constraints *RuleConstraints) error {
	switch rule.Type {
	case RuleTypeRegex:
		if pattern, ok := rule.Parameters["pattern"].(string); ok {
			if len(pattern) < constraints.MinPatternLength {
				return fmt.Errorf("pattern too short: %d < %d", len(pattern), constraints.MinPatternLength)
			}
			if len(pattern) > constraints.MaxPatternLength {
				return fmt.Errorf("pattern too long: %d > %d", len(pattern), constraints.MaxPatternLength)
			}
			
			// Validate regex syntax
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("invalid regex pattern: %w", err)
			}
		}
		
	case RuleTypeSemantic:
		if threshold, ok := rule.Parameters["threshold"].(float64); ok {
			if threshold < constraints.MinThreshold || threshold > constraints.MaxThreshold {
				return fmt.Errorf("threshold out of range: %f", threshold)
			}
		}
		
	case RuleTypePII:
		if types, ok := rule.Parameters["pii_types"].([]string); ok {
			if len(types) == 0 {
				return fmt.Errorf("PII rule must have at least one type")
			}
			
			for _, t := range types {
				found := false
				for _, allowed := range constraints.AllowedPIITypes {
					if t == allowed {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("invalid PII type: %s", t)
				}
			}
		}
	}
	
	return nil
}

// MutationResult contains the results of a mutation operation
type MutationResult struct {
	OriginalRule Rule     `json:"original_rule"`
	Variants     []Rule   `json:"variants"`
	Successful   int      `json:"successful"`
	Failed       int      `json:"failed"`
	Errors       []string `json:"errors"`
}

// BatchMutate applies mutations to multiple rules
func (me *MutationEngine) BatchMutate(rules []Rule, variantsPerRule int, config MutationConfig) (*MutationResult, error) {
	result := &MutationResult{
		Variants: make([]Rule, 0),
		Errors:   make([]string, 0),
	}
	
	for _, rule := range rules {
		variants, err := me.GenerateRuleVariants(rule, variantsPerRule, config)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Rule %s: %v", rule.ID, err))
			continue
		}
		
		result.Variants = append(result.Variants, variants...)
		result.Successful++
	}
	
	return result, nil
}