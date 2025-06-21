package moderation

import (
	"math"
	"sort"
	"time"
)

// ScoreAggregationStrategy defines how multiple rule scores are combined
type ScoreAggregationStrategy int

const (
	ScoreStrategyMax     ScoreAggregationStrategy = iota // Take maximum score
	ScoreStrategyWeighted                                // Weighted average by priority
	ScoreStrategyAdditive                                // Add scores with diminishing returns
	ScoreStrategyConsensus                              // Require multiple rules to agree
)

// RuleScoreCalculator handles advanced scoring calculations for rule results
type RuleScoreCalculator struct {
	strategy            ScoreAggregationStrategy
	priorityWeights     map[int]float64
	categoryWeights     map[string]float64
	confidenceThreshold float64
	decayFactor         float64
}

// AggregatedRuleResult represents the final result after aggregating multiple rule results
type AggregatedRuleResult struct {
	FinalScore       float64
	FinalConfidence  float64
	RecommendedAction string
	PrimaryCategory   string
	TriggeredRules    []*RuleResult
	ScoreBreakdown    *ScoreBreakdown
	ProcessingTime    time.Duration
}

// ScoreBreakdown provides detailed information about how the final score was calculated
type ScoreBreakdown struct {
	BaseScore         float64                    `json:"base_score"`
	PriorityBonus     float64                    `json:"priority_bonus"`
	CategoryModifier  float64                    `json:"category_modifier"`
	ConfidenceAdjust  float64                    `json:"confidence_adjust"`
	RuleContributions map[string]RuleContribution `json:"rule_contributions"`
	Strategy          string                     `json:"strategy"`
}

// RuleContribution shows how each rule contributed to the final score
type RuleContribution struct {
	RuleID          string  `json:"rule_id"`
	RuleName        string  `json:"rule_name"`
	OriginalScore   float64 `json:"original_score"`
	WeightedScore   float64 `json:"weighted_score"`
	PriorityWeight  float64 `json:"priority_weight"`
	CategoryWeight  float64 `json:"category_weight"`
	ConfidenceBonus float64 `json:"confidence_bonus"`
}

// NewRuleScoreCalculator creates a new score calculator with default settings
func NewRuleScoreCalculator(strategy ScoreAggregationStrategy) *RuleScoreCalculator {
	calculator := &RuleScoreCalculator{
		strategy:            strategy,
		priorityWeights:     getDefaultPriorityWeights(),
		categoryWeights:     getDefaultCategoryWeights(),
		confidenceThreshold: 0.5,
		decayFactor:         0.8,
	}
	
	return calculator
}

// getDefaultPriorityWeights returns default priority weight mappings
func getDefaultPriorityWeights() map[int]float64 {
	return map[int]float64{
		1: 1.5,  // Critical priority rules get 50% boost
		2: 1.2,  // High priority rules get 20% boost
		3: 1.0,  // Normal priority rules (baseline)
		4: 0.9,  // Lower priority rules get 10% reduction
		5: 0.8,  // Low priority rules get 20% reduction
		6: 0.7,  // Very low priority
		7: 0.6,  // Minimal priority
		8: 0.5,  // Advisory only
		9: 0.4,  // Informational
		10: 0.3, // Debug/testing
	}
}

// getDefaultCategoryWeights returns default category weight mappings
func getDefaultCategoryWeights() map[string]float64 {
	return map[string]float64{
		CategoryViolence:   1.3, // Violence gets highest weight
		CategoryHateSpeech: 1.2, // Hate speech gets high weight
		CategorySpam:       0.6, // Spam gets lower weight
		CategoryCustom:     1.0, // Custom categories baseline
		"harassment":      1.1, // Harassment above baseline
		"toxicity":        1.0, // Toxicity baseline
		"profanity":       0.7, // Profanity lower priority
		"advertising":     0.5, // Advertising lowest weight
	}
}

// AggregateRuleResults combines multiple rule results into a final decision
func (rsc *RuleScoreCalculator) AggregateRuleResults(results []*RuleResult) *AggregatedRuleResult {
	startTime := time.Now()
	
	if len(results) == 0 {
		return &AggregatedRuleResult{
			FinalScore:      0.0,
			FinalConfidence: 1.0,
			RecommendedAction: "allow",
			PrimaryCategory: "",
			TriggeredRules:  []*RuleResult{},
			ProcessingTime:  time.Since(startTime),
		}
	}

	// Filter only matched rules
	matchedRules := make([]*RuleResult, 0)
	for _, result := range results {
		if result.Matched {
			matchedRules = append(matchedRules, result)
		}
	}

	if len(matchedRules) == 0 {
		return &AggregatedRuleResult{
			FinalScore:        0.0,
			FinalConfidence:   1.0,
			RecommendedAction: "allow",
			PrimaryCategory:   "",
			TriggeredRules:    []*RuleResult{},
			ProcessingTime:    time.Since(startTime),
		}
	}

	// Calculate aggregated score based on strategy
	var finalScore, finalConfidence float64
	var scoreBreakdown *ScoreBreakdown

	switch rsc.strategy {
	case ScoreStrategyMax:
		finalScore, finalConfidence, scoreBreakdown = rsc.calculateMaxScore(matchedRules)
	case ScoreStrategyWeighted:
		finalScore, finalConfidence, scoreBreakdown = rsc.calculateWeightedScore(matchedRules)
	case ScoreStrategyAdditive:
		finalScore, finalConfidence, scoreBreakdown = rsc.calculateAdditiveScore(matchedRules)
	case ScoreStrategyConsensus:
		finalScore, finalConfidence, scoreBreakdown = rsc.calculateConsensusScore(matchedRules)
	default:
		finalScore, finalConfidence, scoreBreakdown = rsc.calculateMaxScore(matchedRules)
	}

	// Determine recommended action
	recommendedAction := rsc.determineAction(matchedRules, finalScore, finalConfidence)
	
	// Determine primary category
	primaryCategory := rsc.determinePrimaryCategory(matchedRules)

	return &AggregatedRuleResult{
		FinalScore:        finalScore,
		FinalConfidence:   finalConfidence,
		RecommendedAction: recommendedAction,
		PrimaryCategory:   primaryCategory,
		TriggeredRules:    matchedRules,
		ScoreBreakdown:    scoreBreakdown,
		ProcessingTime:    time.Since(startTime),
	}
}

// calculateMaxScore takes the maximum score among all matched rules
func (rsc *RuleScoreCalculator) calculateMaxScore(results []*RuleResult) (float64, float64, *ScoreBreakdown) {
	maxScore := 0.0
	maxConfidence := 0.0
	var bestRule *RuleResult

	ruleContributions := make(map[string]RuleContribution)

	for _, result := range results {
		// Apply priority and category weights
		priorityWeight := rsc.getPriorityWeight(result)
		categoryWeight := rsc.getCategoryWeight(result.Category)
		
		weightedScore := result.Score * priorityWeight * categoryWeight
		confidenceBonus := rsc.calculateConfidenceBonus(result.Confidence)
		finalRuleScore := weightedScore + confidenceBonus

		// Track contribution
		ruleContributions[result.RuleID] = RuleContribution{
			RuleID:          result.RuleID,
			RuleName:        result.RuleName,
			OriginalScore:   result.Score,
			WeightedScore:   weightedScore,
			PriorityWeight:  priorityWeight,
			CategoryWeight:  categoryWeight,
			ConfidenceBonus: confidenceBonus,
		}

		if finalRuleScore > maxScore {
			maxScore = finalRuleScore
			maxConfidence = result.Confidence
			bestRule = result
		}
	}

	// Ensure score doesn't exceed 1.0
	if maxScore > 1.0 {
		maxScore = 1.0
	}

	breakdown := &ScoreBreakdown{
		BaseScore:         bestRule.Score,
		PriorityBonus:     rsc.getPriorityWeight(bestRule) - 1.0,
		CategoryModifier:  rsc.getCategoryWeight(bestRule.Category) - 1.0,
		ConfidenceAdjust:  rsc.calculateConfidenceBonus(bestRule.Confidence),
		RuleContributions: ruleContributions,
		Strategy:          "max",
	}

	return maxScore, maxConfidence, breakdown
}

// calculateWeightedScore calculates a weighted average based on rule priorities
func (rsc *RuleScoreCalculator) calculateWeightedScore(results []*RuleResult) (float64, float64, *ScoreBreakdown) {
	totalWeightedScore := 0.0
	totalWeight := 0.0
	totalConfidence := 0.0

	ruleContributions := make(map[string]RuleContribution)

	for _, result := range results {
		priorityWeight := rsc.getPriorityWeight(result)
		categoryWeight := rsc.getCategoryWeight(result.Category)
		confidenceBonus := rsc.calculateConfidenceBonus(result.Confidence)
		
		// Calculate weighted contribution
		weight := priorityWeight * categoryWeight
		weightedScore := result.Score * weight + confidenceBonus
		
		totalWeightedScore += weightedScore
		totalWeight += weight
		totalConfidence += result.Confidence * weight

		ruleContributions[result.RuleID] = RuleContribution{
			RuleID:          result.RuleID,
			RuleName:        result.RuleName,
			OriginalScore:   result.Score,
			WeightedScore:   weightedScore,
			PriorityWeight:  priorityWeight,
			CategoryWeight:  categoryWeight,
			ConfidenceBonus: confidenceBonus,
		}
	}

	finalScore := 0.0
	finalConfidence := 0.0

	if totalWeight > 0 {
		finalScore = totalWeightedScore / totalWeight
		finalConfidence = totalConfidence / totalWeight
	}

	// Ensure score doesn't exceed 1.0
	if finalScore > 1.0 {
		finalScore = 1.0
	}

	breakdown := &ScoreBreakdown{
		BaseScore:         finalScore,
		PriorityBonus:     0.0, // Already factored into weighted calculation
		CategoryModifier:  0.0, // Already factored into weighted calculation
		ConfidenceAdjust:  0.0, // Already factored into weighted calculation
		RuleContributions: ruleContributions,
		Strategy:          "weighted",
	}

	return finalScore, finalConfidence, breakdown
}

// calculateAdditiveScore adds scores with diminishing returns
func (rsc *RuleScoreCalculator) calculateAdditiveScore(results []*RuleResult) (float64, float64, *ScoreBreakdown) {
	// Sort by priority (lower number = higher priority)
	sortedResults := make([]*RuleResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		// Extract priority from context or assume based on order
		return rsc.getRulePriority(sortedResults[i]) < rsc.getRulePriority(sortedResults[j])
	})

	totalScore := 0.0
	totalConfidence := 0.0
	decayFactor := 1.0

	ruleContributions := make(map[string]RuleContribution)

	for _, result := range sortedResults {
		priorityWeight := rsc.getPriorityWeight(result)
		categoryWeight := rsc.getCategoryWeight(result.Category)
		confidenceBonus := rsc.calculateConfidenceBonus(result.Confidence)
		
		// Apply decay factor for diminishing returns
		contribution := (result.Score * priorityWeight * categoryWeight + confidenceBonus) * decayFactor
		totalScore += contribution
		totalConfidence += result.Confidence * decayFactor

		ruleContributions[result.RuleID] = RuleContribution{
			RuleID:          result.RuleID,
			RuleName:        result.RuleName,
			OriginalScore:   result.Score,
			WeightedScore:   contribution,
			PriorityWeight:  priorityWeight,
			CategoryWeight:  categoryWeight,
			ConfidenceBonus: confidenceBonus,
		}

		// Reduce decay factor for subsequent rules
		decayFactor *= rsc.decayFactor
	}

	// Normalize confidence
	if len(results) > 0 {
		totalConfidence /= float64(len(results))
	}

	// Ensure score doesn't exceed 1.0
	if totalScore > 1.0 {
		totalScore = 1.0
	}

	breakdown := &ScoreBreakdown{
		BaseScore:         totalScore,
		PriorityBonus:     0.0, // Factored into additive calculation
		CategoryModifier:  0.0, // Factored into additive calculation
		ConfidenceAdjust:  0.0, // Factored into additive calculation
		RuleContributions: ruleContributions,
		Strategy:          "additive",
	}

	return totalScore, totalConfidence, breakdown
}

// calculateConsensusScore requires multiple rules to agree for high scores
func (rsc *RuleScoreCalculator) calculateConsensusScore(results []*RuleResult) (float64, float64, *ScoreBreakdown) {
	if len(results) < 2 {
		// Not enough rules for consensus, fall back to max score
		return rsc.calculateMaxScore(results)
	}

	// Group rules by category
	categoryGroups := make(map[string][]*RuleResult)
	for _, result := range results {
		categoryGroups[result.Category] = append(categoryGroups[result.Category], result)
	}

	maxConsensusScore := 0.0
	maxConsensusConfidence := 0.0
	ruleContributions := make(map[string]RuleContribution)

	// Check each category for consensus
	for _, categoryResults := range categoryGroups {
		if len(categoryResults) < 2 {
			// Single rule in category, apply penalty
			for _, result := range categoryResults {
				priorityWeight := rsc.getPriorityWeight(result)
				categoryWeight := rsc.getCategoryWeight(result.Category)
				
				penalizedScore := result.Score * priorityWeight * categoryWeight * 0.7 // 30% penalty for no consensus
				
				ruleContributions[result.RuleID] = RuleContribution{
					RuleID:          result.RuleID,
					RuleName:        result.RuleName,
					OriginalScore:   result.Score,
					WeightedScore:   penalizedScore,
					PriorityWeight:  priorityWeight * 0.7,
					CategoryWeight:  categoryWeight,
					ConfidenceBonus: 0.0,
				}

				if penalizedScore > maxConsensusScore {
					maxConsensusScore = penalizedScore
					maxConsensusConfidence = result.Confidence * 0.7
				}
			}
		} else {
			// Multiple rules in category, calculate consensus
			avgScore := 0.0
			avgConfidence := 0.0
			totalWeight := 0.0

			for _, result := range categoryResults {
				priorityWeight := rsc.getPriorityWeight(result)
				categoryWeight := rsc.getCategoryWeight(result.Category)
				weight := priorityWeight * categoryWeight
				
				avgScore += result.Score * weight
				avgConfidence += result.Confidence * weight
				totalWeight += weight

				ruleContributions[result.RuleID] = RuleContribution{
					RuleID:          result.RuleID,
					RuleName:        result.RuleName,
					OriginalScore:   result.Score,
					WeightedScore:   result.Score * weight,
					PriorityWeight:  priorityWeight,
					CategoryWeight:  categoryWeight,
					ConfidenceBonus: 0.1, // Consensus bonus
				}
			}

			if totalWeight > 0 {
				consensusScore := (avgScore / totalWeight) * 1.1 // 10% consensus bonus
				consensusConfidence := (avgConfidence / totalWeight) * 1.1

				if consensusScore > maxConsensusScore {
					maxConsensusScore = consensusScore
					maxConsensusConfidence = consensusConfidence
				}
			}
		}
	}

	// Ensure score doesn't exceed 1.0
	if maxConsensusScore > 1.0 {
		maxConsensusScore = 1.0
	}
	if maxConsensusConfidence > 1.0 {
		maxConsensusConfidence = 1.0
	}

	breakdown := &ScoreBreakdown{
		BaseScore:         maxConsensusScore,
		PriorityBonus:     0.0, // Factored into consensus calculation
		CategoryModifier:  0.0, // Factored into consensus calculation
		ConfidenceAdjust:  0.1, // Consensus bonus
		RuleContributions: ruleContributions,
		Strategy:          "consensus",
	}

	return maxConsensusScore, maxConsensusConfidence, breakdown
}

// Helper methods

// getPriorityWeight returns the weight multiplier for a rule's priority
func (rsc *RuleScoreCalculator) getPriorityWeight(result *RuleResult) float64 {
	priority := rsc.getRulePriority(result)
	if weight, exists := rsc.priorityWeights[priority]; exists {
		return weight
	}
	return 1.0 // Default weight
}

// getCategoryWeight returns the weight multiplier for a rule's category
func (rsc *RuleScoreCalculator) getCategoryWeight(category string) float64 {
	if weight, exists := rsc.categoryWeights[category]; exists {
		return weight
	}
	return 1.0 // Default weight
}

// getRulePriority extracts priority from rule context or returns default
func (rsc *RuleScoreCalculator) getRulePriority(result *RuleResult) int {
	if result.Context != nil {
		if priority, ok := result.Context["rule_priority"].(int); ok {
			return priority
		}
	}
	return 3 // Default medium priority
}

// calculateConfidenceBonus calculates a score bonus based on confidence level
func (rsc *RuleScoreCalculator) calculateConfidenceBonus(confidence float64) float64 {
	if confidence < rsc.confidenceThreshold {
		return 0.0 // No bonus for low confidence
	}
	
	// Linear bonus: confidence above threshold gets up to 0.1 bonus
	excessConfidence := confidence - rsc.confidenceThreshold
	maxExcess := 1.0 - rsc.confidenceThreshold
	
	if maxExcess > 0 {
		return (excessConfidence / maxExcess) * 0.1
	}
	
	return 0.0
}

// determineAction determines the recommended action based on aggregated results
func (rsc *RuleScoreCalculator) determineAction(results []*RuleResult, finalScore, finalConfidence float64) string {
	// Find the highest priority action among triggered rules
	highestPriorityAction := "allow"
	lowestPriority := 11 // Higher than max priority

	for _, result := range results {
		if result.Matched {
			priority := rsc.getRulePriority(result)
			if priority < lowestPriority {
				lowestPriority = priority
				highestPriorityAction = result.Action
			}
		}
	}

	// Override based on final score and confidence
	if finalScore >= 0.9 && finalConfidence >= 0.8 {
		return ActionBlock // High confidence, high score = block
	} else if finalScore >= 0.7 && finalConfidence >= 0.6 {
		if highestPriorityAction == ActionBlock {
			return ActionBlock
		}
		return ActionFlag // Medium-high score = flag for review
	} else if finalScore >= 0.5 {
		return ActionWarn // Medium score = warn
	}

	return highestPriorityAction
}

// determinePrimaryCategory finds the most significant category among triggered rules
func (rsc *RuleScoreCalculator) determinePrimaryCategory(results []*RuleResult) string {
	categoryScores := make(map[string]float64)
	
	for _, result := range results {
		if result.Matched {
			priorityWeight := rsc.getPriorityWeight(result)
			categoryWeight := rsc.getCategoryWeight(result.Category)
			
			categoryScore := result.Score * priorityWeight * categoryWeight
			if existing, exists := categoryScores[result.Category]; !exists || categoryScore > existing {
				categoryScores[result.Category] = categoryScore
			}
		}
	}

	// Find category with highest weighted score
	maxScore := 0.0
	primaryCategory := ""
	
	for category, score := range categoryScores {
		if score > maxScore {
			maxScore = score
			primaryCategory = category
		}
	}

	return primaryCategory
}

// SetPriorityWeights allows customization of priority weights
func (rsc *RuleScoreCalculator) SetPriorityWeights(weights map[int]float64) {
	rsc.priorityWeights = weights
}

// SetCategoryWeights allows customization of category weights
func (rsc *RuleScoreCalculator) SetCategoryWeights(weights map[string]float64) {
	rsc.categoryWeights = weights
}

// SetConfidenceThreshold sets the minimum confidence for bonuses
func (rsc *RuleScoreCalculator) SetConfidenceThreshold(threshold float64) {
	rsc.confidenceThreshold = math.Max(0.0, math.Min(1.0, threshold))
}

// SetDecayFactor sets the decay factor for additive scoring
func (rsc *RuleScoreCalculator) SetDecayFactor(factor float64) {
	rsc.decayFactor = math.Max(0.1, math.Min(1.0, factor))
}