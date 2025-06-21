package optimizer

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// CombinationEngine handles combining multiple rules into hybrid rules
type CombinationEngine struct {
	rng              *rand.Rand
	maxCombinations  int
	compatibilityMap map[RuleType][]RuleType
}

// CombinationStrategy defines how rules are combined
type CombinationStrategy string

const (
	StrategyAND       CombinationStrategy = "AND"       // All rules must match
	StrategyOR        CombinationStrategy = "OR"        // Any rule must match
	StrategyWeighted  CombinationStrategy = "WEIGHTED"  // Weighted combination
	StrategyThreshold CombinationStrategy = "THRESHOLD" // Score threshold
	StrategyPipeline  CombinationStrategy = "PIPELINE"  // Sequential pipeline
)

// CombinationConfig controls rule combination behavior
type CombinationConfig struct {
	MaxRulesPerCombination int                           `json:"max_rules_per_combination"`
	AllowedStrategies      []CombinationStrategy         `json:"allowed_strategies"`
	StrategyWeights        map[CombinationStrategy]float64 `json:"strategy_weights"`
	CompatibilityMatrix    map[RuleType][]RuleType       `json:"compatibility_matrix"`
	MinCompatibilityScore  float64                       `json:"min_compatibility_score"`
	PreferenceWeights      map[string]float64            `json:"preference_weights"`
}

// CombinationResult represents the result of combining rules
type CombinationResult struct {
	CombinedRule      Rule                `json:"combined_rule"`
	SourceRules       []Rule              `json:"source_rules"`
	Strategy          CombinationStrategy `json:"strategy"`
	CompatibilityScore float64            `json:"compatibility_score"`
	ExpectedPerformance map[string]float64 `json:"expected_performance"`
}

// NewCombinationEngine creates a new rule combination engine
func NewCombinationEngine(maxCombinations int) *CombinationEngine {
	engine := &CombinationEngine{
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		maxCombinations: maxCombinations,
		compatibilityMap: getDefaultCompatibilityMap(),
	}
	
	if maxCombinations == 0 {
		engine.maxCombinations = 5
	}
	
	return engine
}

// GetDefaultCombinationConfig returns default combination configuration
func GetDefaultCombinationConfig() CombinationConfig {
	return CombinationConfig{
		MaxRulesPerCombination: 3,
		AllowedStrategies: []CombinationStrategy{
			StrategyAND, StrategyOR, StrategyWeighted, StrategyThreshold,
		},
		StrategyWeights: map[CombinationStrategy]float64{
			StrategyAND:       0.2,
			StrategyOR:        0.2,
			StrategyWeighted:  0.4,
			StrategyThreshold: 0.2,
		},
		CompatibilityMatrix: getDefaultCompatibilityMap(),
		MinCompatibilityScore: 0.3,
		PreferenceWeights: map[string]float64{
			"detection_rate":       0.4,
			"false_positive_rate":  0.3,
			"response_time":        0.2,
			"complexity":           0.1,
		},
	}
}

// CombineRules combines multiple rules into a single hybrid rule
func (ce *CombinationEngine) CombineRules(rules []Rule, config CombinationConfig) (*CombinationResult, error) {
	if len(rules) < 2 {
		return nil, fmt.Errorf("need at least 2 rules to combine")
	}
	
	if len(rules) > config.MaxRulesPerCombination {
		return nil, fmt.Errorf("too many rules to combine: %d > %d", len(rules), config.MaxRulesPerCombination)
	}
	
	// Check compatibility
	compatibilityScore := ce.calculateCompatibility(rules, config)
	if compatibilityScore < config.MinCompatibilityScore {
		return nil, fmt.Errorf("rules are not compatible: score %f < %f", compatibilityScore, config.MinCompatibilityScore)
	}
	
	// Select combination strategy
	strategy := ce.selectStrategy(rules, config)
	
	// Create combined rule
	combinedRule, err := ce.createCombinedRule(rules, strategy, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create combined rule: %w", err)
	}
	
	// Calculate expected performance
	expectedPerformance := ce.estimatePerformance(rules, strategy, config)
	
	result := &CombinationResult{
		CombinedRule:        combinedRule,
		SourceRules:         rules,
		Strategy:            strategy,
		CompatibilityScore:  compatibilityScore,
		ExpectedPerformance: expectedPerformance,
	}
	
	return result, nil
}

// calculateCompatibility determines how well rules can be combined
func (ce *CombinationEngine) calculateCompatibility(rules []Rule, config CombinationConfig) float64 {
	if len(rules) < 2 {
		return 0.0
	}
	
	totalScore := 0.0
	comparisons := 0
	
	for i := 0; i < len(rules); i++ {
		for j := i + 1; j < len(rules); j++ {
			score := ce.calculatePairCompatibility(rules[i], rules[j], config)
			totalScore += score
			comparisons++
		}
	}
	
	if comparisons == 0 {
		return 0.0
	}
	
	return totalScore / float64(comparisons)
}

// calculatePairCompatibility calculates compatibility between two rules
func (ce *CombinationEngine) calculatePairCompatibility(rule1, rule2 Rule, config CombinationConfig) float64 {
	score := 0.0
	
	// Type compatibility
	compatibleTypes, exists := config.CompatibilityMatrix[rule1.Type]
	if exists {
		for _, compatType := range compatibleTypes {
			if compatType == rule2.Type {
				score += 0.4
				break
			}
		}
	}
	
	// Parameter compatibility
	paramScore := ce.calculateParameterCompatibility(rule1, rule2)
	score += paramScore * 0.3
	
	// Priority compatibility (similar priorities work better together)
	priorityDiff := abs(rule1.Priority - rule2.Priority)
	if priorityDiff <= 1 {
		score += 0.2
	} else if priorityDiff <= 2 {
		score += 0.1
	}
	
	// Tag overlap (rules with similar tags are more compatible)
	tagOverlap := ce.calculateTagOverlap(rule1.Tags, rule2.Tags)
	score += tagOverlap * 0.1
	
	return score
}

// calculateParameterCompatibility checks if rule parameters are compatible
func (ce *CombinationEngine) calculateParameterCompatibility(rule1, rule2 Rule) float64 {
	score := 0.0
	
	// For semantic rules, check if thresholds are in similar ranges
	if rule1.Type == RuleTypeSemantic && rule2.Type == RuleTypeSemantic {
		threshold1, ok1 := rule1.Parameters["threshold"].(float64)
		threshold2, ok2 := rule2.Parameters["threshold"].(float64)
		
		if ok1 && ok2 {
			diff := abs64(threshold1 - threshold2)
			if diff < 0.1 {
				score += 0.5
			} else if diff < 0.2 {
				score += 0.3
			}
		}
	}
	
	// For PII rules, check for overlapping types
	if rule1.Type == RuleTypePII && rule2.Type == RuleTypePII {
		types1, ok1 := rule1.Parameters["pii_types"].([]string)
		types2, ok2 := rule2.Parameters["pii_types"].([]string)
		
		if ok1 && ok2 {
			overlap := ce.calculateStringSliceOverlap(types1, types2)
			score += overlap * 0.3
		}
	}
	
	// For regex rules, check pattern complexity compatibility
	if rule1.Type == RuleTypeRegex && rule2.Type == RuleTypeRegex {
		pattern1, ok1 := rule1.Parameters["pattern"].(string)
		pattern2, ok2 := rule2.Parameters["pattern"].(string)
		
		if ok1 && ok2 {
			complexity1 := ce.calculatePatternComplexity(pattern1)
			complexity2 := ce.calculatePatternComplexity(pattern2)
			
			diff := abs64(complexity1 - complexity2)
			if diff < 0.3 {
				score += 0.4
			} else if diff < 0.5 {
				score += 0.2
			}
		}
	}
	
	return score
}

// selectStrategy chooses the best combination strategy for the given rules
func (ce *CombinationEngine) selectStrategy(rules []Rule, config CombinationConfig) CombinationStrategy {
	// Calculate strategy scores based on rule characteristics
	scores := make(map[CombinationStrategy]float64)
	
	for _, strategy := range config.AllowedStrategies {
		scores[strategy] = ce.calculateStrategyScore(strategy, rules, config)
	}
	
	// Select strategy with highest score
	bestStrategy := config.AllowedStrategies[0]
	bestScore := scores[bestStrategy]
	
	for strategy, score := range scores {
		if score > bestScore {
			bestStrategy = strategy
			bestScore = score
		}
	}
	
	return bestStrategy
}

// calculateStrategyScore calculates how well a strategy fits the given rules
func (ce *CombinationEngine) calculateStrategyScore(strategy CombinationStrategy, rules []Rule, config CombinationConfig) float64 {
	baseScore := config.StrategyWeights[strategy]
	
	switch strategy {
	case StrategyAND:
		// AND works well with complementary rules (different types)
		typeSet := make(map[RuleType]bool)
		for _, rule := range rules {
			typeSet[rule.Type] = true
		}
		if len(typeSet) == len(rules) {
			baseScore += 0.3 // Bonus for diverse types
		}
		
	case StrategyOR:
		// OR works well with similar rules (same type, different parameters)
		if ce.allSameType(rules) {
			baseScore += 0.2
		}
		
	case StrategyWeighted:
		// Weighted works well with rules of different confidence levels
		baseScore += 0.1 // Generally good choice
		
	case StrategyThreshold:
		// Threshold works well when we want to be conservative
		if len(rules) >= 3 {
			baseScore += 0.2
		}
		
	case StrategyPipeline:
		// Pipeline works well with rules that can be ordered by specificity
		baseScore += ce.calculatePipelineScore(rules)
	}
	
	return baseScore
}

// createCombinedRule creates a new hybrid rule from multiple source rules
func (ce *CombinationEngine) createCombinedRule(rules []Rule, strategy CombinationStrategy, config CombinationConfig) (Rule, error) {
	combinedRule := Rule{
		ID:          generateRuleID(),
		Name:        ce.generateCombinedName(rules, strategy),
		Type:        RuleTypeHybrid,
		Parameters:  make(map[string]interface{}),
		Enabled:     true,
		Priority:    ce.calculateCombinedPriority(rules),
		Description: ce.generateCombinedDescription(rules, strategy),
		Tags:        ce.combineTags(rules),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0",
	}
	
	// Set combination strategy
	combinedRule.Parameters["combination_strategy"] = string(strategy)
	combinedRule.Parameters["source_rule_ids"] = ce.extractRuleIDs(rules)
	
	// Add strategy-specific parameters
	switch strategy {
	case StrategyAND:
		combinedRule.Parameters["require_all"] = true
		combinedRule.Parameters["component_rules"] = ce.serializeRules(rules)
		
	case StrategyOR:
		combinedRule.Parameters["require_any"] = true
		combinedRule.Parameters["component_rules"] = ce.serializeRules(rules)
		
	case StrategyWeighted:
		weights := ce.calculateRuleWeights(rules, config)
		combinedRule.Parameters["rule_weights"] = weights
		combinedRule.Parameters["component_rules"] = ce.serializeRules(rules)
		combinedRule.Parameters["threshold"] = 0.5 // Default threshold
		
	case StrategyThreshold:
		combinedRule.Parameters["threshold"] = ce.calculateOptimalThreshold(rules)
		combinedRule.Parameters["min_matches"] = len(rules)/2 + 1
		combinedRule.Parameters["component_rules"] = ce.serializeRules(rules)
		
	case StrategyPipeline:
		orderedRules := ce.orderRulesForPipeline(rules)
		combinedRule.Parameters["pipeline_rules"] = ce.serializeRules(orderedRules)
		combinedRule.Parameters["short_circuit"] = true
	}
	
	return combinedRule, nil
}

// generateCombinedName creates a name for the combined rule
func (ce *CombinationEngine) generateCombinedName(rules []Rule, strategy CombinationStrategy) string {
	if len(rules) == 0 {
		return "empty_combination"
	}
	
	// Extract key terms from rule names
	terms := make([]string, 0)
	for _, rule := range rules {
		// Extract meaningful parts from rule names
		parts := strings.Split(rule.Name, "_")
		for _, part := range parts {
			if len(part) > 2 && part != "rule" && part != "variant" {
				terms = append(terms, part)
			}
		}
	}
	
	// Deduplicate terms
	uniqueTerms := make([]string, 0)
	seen := make(map[string]bool)
	for _, term := range terms {
		if !seen[term] {
			uniqueTerms = append(uniqueTerms, term)
			seen[term] = true
		}
	}
	
	// Limit to first 3 terms
	if len(uniqueTerms) > 3 {
		uniqueTerms = uniqueTerms[:3]
	}
	
	baseName := strings.Join(uniqueTerms, "_")
	if baseName == "" {
		baseName = "combined"
	}
	
	return fmt.Sprintf("%s_%s_%d", baseName, strings.ToLower(string(strategy)), len(rules))
}

// estimatePerformance estimates the performance of the combined rule
func (ce *CombinationEngine) estimatePerformance(rules []Rule, strategy CombinationStrategy, config CombinationConfig) map[string]float64 {
	performance := make(map[string]float64)
	
	// Collect individual rule performances (using fitness as proxy)
	avgDetection := 0.0
	avgFalsePositive := 0.0
	maxResponseTime := 0.0
	
	for _, rule := range rules {
		// Use rule fitness and historical performance if available
		avgDetection += rule.Fitness * 0.95 // Assume fitness correlates with detection
		avgFalsePositive += (1.0 - rule.Fitness) * 0.1 // Inverse for false positives
		maxResponseTime += 5.0 // Base response time per rule
	}
	
	avgDetection /= float64(len(rules))
	avgFalsePositive /= float64(len(rules))
	
	// Adjust based on combination strategy
	switch strategy {
	case StrategyAND:
		// AND reduces false positives but may reduce detection
		performance["detection_rate"] = avgDetection * 0.9
		performance["false_positive_rate"] = avgFalsePositive * 0.5
		performance["response_time_ms"] = maxResponseTime * 1.2
		
	case StrategyOR:
		// OR increases detection but may increase false positives
		performance["detection_rate"] = avgDetection * 1.1
		performance["false_positive_rate"] = avgFalsePositive * 1.3
		performance["response_time_ms"] = maxResponseTime * 0.8
		
	case StrategyWeighted:
		// Weighted is balanced
		performance["detection_rate"] = avgDetection
		performance["false_positive_rate"] = avgFalsePositive
		performance["response_time_ms"] = maxResponseTime
		
	case StrategyThreshold:
		// Threshold is conservative
		performance["detection_rate"] = avgDetection * 0.85
		performance["false_positive_rate"] = avgFalsePositive * 0.3
		performance["response_time_ms"] = maxResponseTime * 1.1
		
	case StrategyPipeline:
		// Pipeline can be efficient
		performance["detection_rate"] = avgDetection * 0.95
		performance["false_positive_rate"] = avgFalsePositive * 0.7
		performance["response_time_ms"] = maxResponseTime * 0.6
	}
	
	// Add user satisfaction estimate
	satisfaction := 0.9 - performance["false_positive_rate"]*0.5 - performance["response_time_ms"]*0.01
	if satisfaction < 0.1 {
		satisfaction = 0.1
	}
	if satisfaction > 1.0 {
		satisfaction = 1.0
	}
	performance["user_satisfaction"] = satisfaction
	
	return performance
}

// Helper functions

func (ce *CombinationEngine) allSameType(rules []Rule) bool {
	if len(rules) == 0 {
		return true
	}
	
	firstType := rules[0].Type
	for _, rule := range rules[1:] {
		if rule.Type != firstType {
			return false
		}
	}
	return true
}

func (ce *CombinationEngine) calculateTagOverlap(tags1, tags2 []string) float64 {
	if len(tags1) == 0 || len(tags2) == 0 {
		return 0.0
	}
	
	set1 := make(map[string]bool)
	for _, tag := range tags1 {
		set1[tag] = true
	}
	
	overlap := 0
	for _, tag := range tags2 {
		if set1[tag] {
			overlap++
		}
	}
	
	total := len(tags1) + len(tags2) - overlap
	if total == 0 {
		return 1.0
	}
	
	return float64(overlap) / float64(total)
}

func (ce *CombinationEngine) calculateStringSliceOverlap(slice1, slice2 []string) float64 {
	if len(slice1) == 0 || len(slice2) == 0 {
		return 0.0
	}
	
	set1 := make(map[string]bool)
	for _, item := range slice1 {
		set1[item] = true
	}
	
	overlap := 0
	for _, item := range slice2 {
		if set1[item] {
			overlap++
		}
	}
	
	return float64(overlap) / float64(len(slice1)+len(slice2)-overlap)
}

func (ce *CombinationEngine) calculatePatternComplexity(pattern string) float64 {
	complexity := 0.0
	
	// Character classes add complexity
	if strings.Contains(pattern, "[") {
		complexity += 0.2
	}
	
	// Quantifiers add complexity
	for _, char := range []string{"*", "+", "?", "{", "}"} {
		if strings.Contains(pattern, char) {
			complexity += 0.1
		}
	}
	
	// Special characters add complexity
	for _, char := range []string{"\\", "^", "$", "|", "(", ")"} {
		if strings.Contains(pattern, char) {
			complexity += 0.1
		}
	}
	
	// Length factor
	complexity += float64(len(pattern)) / 100.0
	
	return complexity
}

func (ce *CombinationEngine) calculateCombinedPriority(rules []Rule) int {
	if len(rules) == 0 {
		return 1
	}
	
	total := 0
	for _, rule := range rules {
		total += rule.Priority
	}
	
	return total / len(rules)
}

func (ce *CombinationEngine) generateCombinedDescription(rules []Rule, strategy CombinationStrategy) string {
	ruleTypes := make(map[RuleType]int)
	for _, rule := range rules {
		ruleTypes[rule.Type]++
	}
	
	typeDesc := make([]string, 0)
	for ruleType, count := range ruleTypes {
		if count == 1 {
			typeDesc = append(typeDesc, string(ruleType))
		} else {
			typeDesc = append(typeDesc, fmt.Sprintf("%dx%s", count, ruleType))
		}
	}
	
	return fmt.Sprintf("Hybrid rule combining %s using %s strategy", 
		strings.Join(typeDesc, ", "), strategy)
}

func (ce *CombinationEngine) combineTags(rules []Rule) []string {
	tagSet := make(map[string]bool)
	for _, rule := range rules {
		for _, tag := range rule.Tags {
			tagSet[tag] = true
		}
	}
	
	// Add combination-specific tags
	tagSet["hybrid"] = true
	tagSet["combined"] = true
	
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	
	sort.Strings(tags)
	return tags
}

func (ce *CombinationEngine) extractRuleIDs(rules []Rule) []string {
	ids := make([]string, len(rules))
	for i, rule := range rules {
		ids[i] = rule.ID
	}
	return ids
}

func (ce *CombinationEngine) serializeRules(rules []Rule) []map[string]interface{} {
	serialized := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		serialized[i] = map[string]interface{}{
			"id":         rule.ID,
			"name":       rule.Name,
			"type":       rule.Type,
			"parameters": rule.Parameters,
			"priority":   rule.Priority,
			"enabled":    rule.Enabled,
		}
	}
	return serialized
}

func (ce *CombinationEngine) calculateRuleWeights(rules []Rule, config CombinationConfig) map[string]float64 {
	weights := make(map[string]float64)
	totalFitness := 0.0
	
	// Calculate total fitness
	for _, rule := range rules {
		totalFitness += rule.Fitness
	}
	
	// Assign weights based on fitness
	if totalFitness > 0 {
		for _, rule := range rules {
			weights[rule.ID] = rule.Fitness / totalFitness
		}
	} else {
		// Equal weights if no fitness data
		weight := 1.0 / float64(len(rules))
		for _, rule := range rules {
			weights[rule.ID] = weight
		}
	}
	
	return weights
}

func (ce *CombinationEngine) calculateOptimalThreshold(rules []Rule) float64 {
	// Conservative threshold based on number of rules
	switch len(rules) {
	case 2:
		return 0.6
	case 3:
		return 0.5
	case 4:
		return 0.4
	default:
		return 0.3
	}
}

func (ce *CombinationEngine) calculatePipelineScore(rules []Rule) float64 {
	// Pipeline works well when rules have different specificities
	if len(rules) < 2 {
		return 0.0
	}
	
	// Check if rules can be ordered by type (regex -> semantic -> PII)
	hasRegex := false
	hasSemantic := false
	hasPII := false
	
	for _, rule := range rules {
		switch rule.Type {
		case RuleTypeRegex:
			hasRegex = true
		case RuleTypeSemantic:
			hasSemantic = true
		case RuleTypePII:
			hasPII = true
		}
	}
	
	// Pipeline works well with this ordering
	if hasRegex && (hasSemantic || hasPII) {
		return 0.3
	}
	
	return 0.1
}

func (ce *CombinationEngine) orderRulesForPipeline(rules []Rule) []Rule {
	// Order rules by type: regex first (fast), then semantic, then PII
	ordered := make([]Rule, len(rules))
	copy(ordered, rules)
	
	sort.Slice(ordered, func(i, j int) bool {
		typeOrder := map[RuleType]int{
			RuleTypeRegex:    1,
			RuleTypeSemantic: 2,
			RuleTypePII:      3,
			RuleTypeHybrid:   4,
			RuleTypeCustom:   5,
		}
		
		return typeOrder[ordered[i].Type] < typeOrder[ordered[j].Type]
	})
	
	return ordered
}

func getDefaultCompatibilityMap() map[RuleType][]RuleType {
	return map[RuleType][]RuleType{
		RuleTypeRegex: {RuleTypeSemantic, RuleTypePII, RuleTypeCustom},
		RuleTypeSemantic: {RuleTypeRegex, RuleTypePII, RuleTypeCustom},
		RuleTypePII: {RuleTypeRegex, RuleTypeSemantic, RuleTypeCustom},
		RuleTypeHybrid: {RuleTypeRegex, RuleTypeSemantic, RuleTypePII, RuleTypeCustom},
		RuleTypeCustom: {RuleTypeRegex, RuleTypeSemantic, RuleTypePII, RuleTypeHybrid},
	}
}

// Utility functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}