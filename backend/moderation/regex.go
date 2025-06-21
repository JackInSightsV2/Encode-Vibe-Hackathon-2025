package moderation

import (
	"context"
	"fmt"
	"qt1-middleware/opik"
	"regexp"
	"strings"
	"sync"
	"time"
)

// RegexRule represents a single regex pattern rule
type RegexRule struct {
	Pattern     string  `json:"pattern"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Severity    string  `json:"severity"`
	compiled    *regexp.Regexp
}

// RegexModerator implements regex-based content moderation
type RegexModerator struct {
	rules        []RegexRule
	config       LayerConfig
	mu           sync.RWMutex
	opikClient   *opik.OpikClient
	defaultRules []RegexRule
}

// NewRegexModerator creates a new regex moderator
func NewRegexModerator(config LayerConfig, opikClient *opik.OpikClient) (*RegexModerator, error) {
	rm := &RegexModerator{
		config:     config,
		opikClient: opikClient,
		rules:      make([]RegexRule, 0),
	}

	// Initialize default rules
	rm.initializeDefaultRules()

	// Load custom rules from config
	if err := rm.loadCustomRules(); err != nil {
		return nil, err
	}

	// Compile all regex patterns
	if err := rm.compilePatterns(); err != nil {
		return nil, err
	}

	return rm, nil
}

// Name returns the name of this moderation layer
func (rm *RegexModerator) Name() string {
	return "regex"
}

// Weight returns the weight of this layer
func (rm *RegexModerator) Weight() float64 {
	return rm.config.Weight
}

// Enabled returns whether this layer is enabled
func (rm *RegexModerator) Enabled() bool {
	return rm.config.Enabled
}

// Config returns the layer configuration
func (rm *RegexModerator) Config() LayerConfig {
	return rm.config
}

// Moderate performs regex-based moderation
func (rm *RegexModerator) Moderate(content string, context ModerationContext) ModerationResult {
	startTime := time.Now()

	result := ModerationResult{
		Score:      0.0,
		Confidence: 1.0, // Regex is deterministic
		Blocked:    false,
		LayerName:  rm.Name(),
		Details:    make(map[string]interface{}),
	}

	// Normalize content for better matching
	normalizedContent := rm.normalizeContent(content)

	rm.mu.RLock()
	matchedRules := make([]string, 0)
	maxScore := 0.0
	var worstCategory string

	for _, rule := range rm.rules {
		if rule.compiled.MatchString(normalizedContent) {
			matchedRules = append(matchedRules, rule.Description)
			if rule.Weight > maxScore {
				maxScore = rule.Weight
				worstCategory = rule.Category
				result.Reason = fmt.Sprintf("Matched rule: %s", rule.Description)
			}
		}
	}
	rm.mu.RUnlock()

	result.Score = maxScore
	result.Category = worstCategory
	result.Details["matched_rules"] = matchedRules
	result.Details["rule_count"] = len(rm.rules)
	result.ProcessTime = time.Since(startTime)

	// Determine if content should be blocked based on threshold
	threshold := rm.config.Threshold
	if threshold == 0 {
		threshold = 0.8 // Default threshold
	}
	result.Blocked = result.Score >= threshold

	return result
}

// ModerateWithTracing performs moderation with Opik tracing
func (rm *RegexModerator) ModerateWithTracing(ctx context.Context, content string, span *opik.Span) (*ModerationResult, error) {
	defer span.End()

	startTime := time.Now()
	
	// Create moderation context from Opik trace context
	moderationCtx := ModerationContext{
		RequestID: span.TraceID,
		Timestamp: time.Now(),
	}

	// Perform moderation
	result := rm.Moderate(content, moderationCtx)

	// Set span output
	span.SetOutput(map[string]interface{}{
		"blocked":       result.Blocked,
		"score":         result.Score,
		"matched_rules": result.Details["matched_rules"],
		"confidence":    result.Confidence,
		"category":      result.Category,
	})

	// Set span metadata
	span.SetMetadata(map[string]interface{}{
		"duration_ms": time.Since(startTime).Milliseconds(),
		"rule_count":  len(rm.rules),
		"content_length": len(content),
	})

	// Calculate and set effectiveness score
	score := rm.calculateEffectivenessScore(&result)
	span.SetScore("effectiveness", score)

	// Convert to pointer type
	moderationResult := &ModerationResult{
		Blocked:        result.Blocked,
		Reason:         result.Reason,
		Confidence:     result.Confidence,
		Categories:     map[string]float64{result.Category: result.Score},
		Scores:         map[string]float64{"regex": result.Score},
		Flagged:        result.Score > 0.5,
		ProcessedAt:    time.Now(),
		ProcessingTime: result.ProcessTime,
		Metadata:       result.Details,
	}

	return moderationResult, nil
}

// calculateEffectivenessScore calculates how effective the regex moderation was
func (rm *RegexModerator) calculateEffectivenessScore(result *ModerationResult) float64 {
	// Base score on confidence and detection
	score := result.Confidence

	// Adjust based on number of matched rules
	if matchedRules, ok := result.Details["matched_rules"].([]string); ok {
		if len(matchedRules) > 1 {
			score *= 1.2 // Bonus for multiple rule matches
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// initializeDefaultRules sets up default regex patterns
func (rm *RegexModerator) initializeDefaultRules() {
	rm.defaultRules = []RegexRule{
		// Violence patterns
		{
			Pattern:     `(?i)\b(kill|murder|destroy|eliminate)\s+(yourself|myself|everyone|all)\b`,
			Category:    CategoryViolence,
			Description: "Violent language with self/others targeting",
			Weight:      0.9,
			Severity:    SeverityHigh,
		},
		{
			Pattern:     `(?i)\b(suicide|self[\s-]?harm|cut\s+myself|end\s+my\s+life)\b`,
			Category:    CategoryViolence,
			Description: "Self-harm or suicide references",
			Weight:      0.95,
			Severity:    SeverityCritical,
		},
		// Hate speech patterns
		{
			Pattern:     `(?i)\b(hate|despise)\s+(all\s+)?(blacks?|whites?|asians?|latinos?|hispanics?|jews?|muslims?|christians?|gays?|lesbians?|trans)\b`,
			Category:    CategoryHateSpeech,
			Description: "Hate speech targeting specific groups",
			Weight:      0.9,
			Severity:    SeverityHigh,
		},
		// Prompt injection patterns
		{
			Pattern:     `(?i)(ignore|forget|disregard)\s+(all\s+)?(previous|prior|above)\s+(instructions?|commands?|rules?)`,
			Category:    CategoryPromptInject,
			Description: "Prompt injection attempt - ignore instructions",
			Weight:      0.8,
			Severity:    SeverityMedium,
		},
		{
			Pattern:     `(?i)you\s+are\s+now\s+(a|an|in)\s+(new\s+)?(mode|role|character)`,
			Category:    CategoryPromptInject,
			Description: "Prompt injection attempt - role change",
			Weight:      0.7,
			Severity:    SeverityMedium,
		},
		// Spam patterns
		{
			Pattern:     `(?i)(click\s+here|buy\s+now|limited\s+offer|act\s+now)\s*!{2,}`,
			Category:    CategorySpam,
			Description: "Spam with excessive exclamation marks",
			Weight:      0.6,
			Severity:    SeverityLow,
		},
		{
			Pattern:     `(?i)\b(viagra|cialis|pharmacy|pills|medication)\s+(online|cheap|discount|free)`,
			Category:    CategorySpam,
			Description: "Pharmaceutical spam",
			Weight:      0.7,
			Severity:    SeverityMedium,
		},
	}
}

// loadCustomRules loads custom rules from configuration
func (rm *RegexModerator) loadCustomRules() error {
	// Start with default rules
	rm.rules = append(rm.rules, rm.defaultRules...)

	// Load blocked words from config if available
	if blockedWords, ok := rm.config.Options["blocked_words"].([]interface{}); ok {
		for _, word := range blockedWords {
			if wordStr, ok := word.(string); ok {
				rule := RegexRule{
					Pattern:     fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(wordStr)),
					Category:    CategoryToxicity,
					Description: fmt.Sprintf("Blocked word: %s", wordStr),
					Weight:      0.8,
					Severity:    SeverityMedium,
				}
				rm.rules = append(rm.rules, rule)
			}
		}
	}

	// Load custom patterns from config
	if customPatterns, ok := rm.config.Options["custom_patterns"].([]interface{}); ok {
		for _, pattern := range customPatterns {
			if patternMap, ok := pattern.(map[string]interface{}); ok {
				rule := RegexRule{
					Pattern:     patternMap["pattern"].(string),
					Category:    CategoryCustom,
					Description: patternMap["reason"].(string),
					Weight:      patternMap["weight"].(float64),
					Severity:    SeverityMedium,
				}
				rm.rules = append(rm.rules, rule)
			}
		}
	}

	return nil
}

// compilePatterns compiles all regex patterns
func (rm *RegexModerator) compilePatterns() error {
	for i := range rm.rules {
		compiled, err := regexp.Compile(rm.rules[i].Pattern)
		if err != nil {
			return fmt.Errorf("failed to compile pattern '%s': %w", rm.rules[i].Pattern, err)
		}
		rm.rules[i].compiled = compiled
	}
	return nil
}

// normalizeContent normalizes content for better pattern matching
func (rm *RegexModerator) normalizeContent(content string) string {
	// Convert to lowercase for case-insensitive matching
	normalized := strings.ToLower(content)
	
	// Replace common obfuscation techniques
	replacements := map[string]string{
		"@": "a",
		"3": "e",
		"1": "i",
		"0": "o",
		"5": "s",
		"$": "s",
		"7": "t",
	}
	
	for old, new := range replacements {
		normalized = strings.ReplaceAll(normalized, old, new)
	}
	
	// Remove extra whitespace
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	return normalized
}

// UpdateRules updates the regex rules dynamically
func (rm *RegexModerator) UpdateRules(rules []RegexRule) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Compile new rules
	for i := range rules {
		compiled, err := regexp.Compile(rules[i].Pattern)
		if err != nil {
			return fmt.Errorf("failed to compile pattern '%s': %w", rules[i].Pattern, err)
		}
		rules[i].compiled = compiled
	}

	// Replace rules
	rm.rules = rules
	return nil
}

// GetStats returns statistics about the regex moderator
func (rm *RegexModerator) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return map[string]interface{}{
		"rule_count":    len(rm.rules),
		"enabled":       rm.Enabled(),
		"weight":        rm.Weight(),
		"threshold":     rm.config.Threshold,
	}
}