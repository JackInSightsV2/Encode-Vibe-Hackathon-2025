package moderation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ExecuteRules executes all applicable rules against content
func (re *RuleExecutor) ExecuteRules(content string, context ModerationContext, rules []*ModerationRule) ([]*RuleResult, error) {
	if len(rules) == 0 {
		return []*RuleResult{}, nil
	}

	results := make([]*RuleResult, 0, len(rules))
	
	// Sort rules by priority (lower number = higher priority)
	sortedRules := make([]*ModerationRule, len(rules))
	copy(sortedRules, rules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority < sortedRules[j].Priority
	})

	// Execute rules in priority order
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		// Check if rule applies to current context
		if !re.ruleAppliestoContext(rule, context) {
			continue
		}

		// Check content-based conditions
		if !re.contentMeetsConditions(rule, content) {
			continue
		}

		// Execute rule based on type
		result, err := re.executeRule(rule, content, context)
		if err != nil {
			return nil, fmt.Errorf("error executing rule %s: %w", rule.ID, err)
		}

		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// executeRule executes a single rule based on its type
func (re *RuleExecutor) executeRule(rule *ModerationRule, content string, context ModerationContext) (*RuleResult, error) {
	startTime := time.Now()
	
	// Check cache first
	if re.cache != nil {
		if cachedResult := re.getCachedResult(rule, content); cachedResult != nil {
			return cachedResult, nil
		}
	}

	var result *RuleResult
	var err error

	switch rule.Type {
	case RuleTypeRegex:
		result, err = re.executeRegexRule(rule, content, context)
	case RuleTypeKeyword:
		result, err = re.executeKeywordRule(rule, content, context)
	case RuleTypeML:
		result, err = re.executeMLRule(rule, content, context)
	case RuleTypeComposite:
		result, err = re.executeCompositeRule(rule, content, context)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.Type)
	}

	if err != nil {
		return nil, err
	}

	// Set execution time
	if result != nil {
		result.Context = map[string]interface{}{
			"execution_time": time.Since(startTime),
			"rule_type":      rule.Type,
			"rule_priority":  rule.Priority,
		}

		// Cache result if caching is enabled
		if re.cache != nil && result.Matched {
			re.cacheResult(rule, content, result)
		}
	}

	return result, nil
}

// executeRegexRule executes a regex-based rule
func (re *RuleExecutor) executeRegexRule(rule *ModerationRule, content string, context ModerationContext) (*RuleResult, error) {
	if rule.compiledPattern == nil {
		return nil, fmt.Errorf("regex pattern not compiled for rule %s", rule.ID)
	}

	// Prepare content for matching
	processedContent := content
	if rule.Conditions == nil || !rule.Conditions.CaseSensitive {
		processedContent = strings.ToLower(processedContent)
	}

	// Find matches
	matches := rule.compiledPattern.FindAllStringSubmatch(processedContent, -1)
	if len(matches) == 0 {
		return &RuleResult{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			Matched:    false,
			Score:      0.0,
			Confidence: 0.0,
			Action:     rule.Action,
			Category:   rule.Category,
			Reason:     "",
		}, nil
	}

	// Calculate score and confidence based on matches
	score := rule.Weight
	confidence := re.calculateRegexConfidence(rule, matches, content)

	// Get first match details
	firstMatch := matches[0]
	matchedText := firstMatch[0]
	position := strings.Index(processedContent, matchedText)
	
	// Extract context around match
	contextText := re.extractContext(content, position, len(matchedText))

	return &RuleResult{
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Matched:    true,
		Score:      score,
		Confidence: confidence,
		Action:     rule.Action,
		Category:   rule.Category,
		Reason:     fmt.Sprintf("Regex pattern matched: %s", rule.Pattern),
		MatchDetails: &MatchDetails{
			MatchedText: matchedText,
			Position:    position,
			Length:      len(matchedText),
			Context:     contextText,
		},
	}, nil
}

// executeKeywordRule executes a keyword-based rule
func (re *RuleExecutor) executeKeywordRule(rule *ModerationRule, content string, context ModerationContext) (*RuleResult, error) {
	if len(rule.normalizedKeywords) == 0 {
		return nil, fmt.Errorf("no keywords compiled for rule %s", rule.ID)
	}

	// Prepare content for matching
	processedContent := content
	if rule.Conditions == nil || !rule.Conditions.CaseSensitive {
		processedContent = strings.ToLower(processedContent)
	}

	matchedKeywords := []string{}
	var firstMatchPos int = -1
	var firstMatchText string

	// Check for keyword matches
	for _, keyword := range rule.normalizedKeywords {
		var found bool
		var pos int

		if rule.Conditions != nil && rule.Conditions.WholeWords {
			// Use word boundary matching
			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
			regex := regexp.MustCompile(pattern)
			if rule.Conditions.CaseSensitive {
				regex = regexp.MustCompile(`(?i)` + pattern)
			}
			
			if match := regex.FindString(processedContent); match != "" {
				found = true
				pos = strings.Index(processedContent, match)
				if firstMatchPos == -1 || pos < firstMatchPos {
					firstMatchPos = pos
					firstMatchText = match
				}
			}
		} else {
			// Simple substring matching
			if pos = strings.Index(processedContent, keyword); pos != -1 {
				found = true
				if firstMatchPos == -1 || pos < firstMatchPos {
					firstMatchPos = pos
					firstMatchText = keyword
				}
			}
		}

		if found {
			matchedKeywords = append(matchedKeywords, keyword)
		}
	}

	// Check if match criteria are met
	matched := false
	if rule.Conditions != nil && rule.Conditions.RequireAll {
		// All keywords must match
		matched = len(matchedKeywords) == len(rule.normalizedKeywords)
	} else {
		// At least one keyword must match
		matched = len(matchedKeywords) > 0
	}

	if !matched {
		return &RuleResult{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			Matched:    false,
			Score:      0.0,
			Confidence: 0.0,
			Action:     rule.Action,
			Category:   rule.Category,
			Reason:     "",
		}, nil
	}

	// Calculate score and confidence
	score := rule.Weight
	confidence := re.calculateKeywordConfidence(rule, matchedKeywords)

	// Extract context around first match
	contextText := ""
	if firstMatchPos != -1 {
		contextText = re.extractContext(content, firstMatchPos, len(firstMatchText))
	}

	return &RuleResult{
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Matched:    true,
		Score:      score,
		Confidence: confidence,
		Action:     rule.Action,
		Category:   rule.Category,
		Reason:     fmt.Sprintf("Keywords matched: %s", strings.Join(matchedKeywords, ", ")),
		MatchDetails: &MatchDetails{
			MatchedText:     firstMatchText,
			MatchedKeywords: matchedKeywords,
			Position:        firstMatchPos,
			Length:          len(firstMatchText),
			Context:         contextText,
		},
	}, nil
}

// executeMLRule executes a machine learning-based rule
func (re *RuleExecutor) executeMLRule(rule *ModerationRule, content string, context ModerationContext) (*RuleResult, error) {
	// For now, implement a stub ML rule that simulates ML processing
	// In a real implementation, this would call actual ML models
	
	modelName := rule.Pattern
	
	// Simulate ML processing based on content characteristics
	score, confidence := re.simulateMLProcessing(content, modelName)
	
	matched := score >= 0.5 // Threshold for ML detection
	
	reason := ""
	if matched {
		reason = fmt.Sprintf("ML model '%s' detected potential violation", modelName)
	}

	return &RuleResult{
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Matched:    matched,
		Score:      score * rule.Weight, // Apply rule weight
		Confidence: confidence,
		Action:     rule.Action,
		Category:   rule.Category,
		Reason:     reason,
		MatchDetails: &MatchDetails{
			MatchedText: content[:min(50, len(content))], // First 50 chars
			Position:    0,
			Length:      len(content),
			Context:     "ML analysis of entire content",
		},
	}, nil
}

// executeCompositeRule executes a composite rule combining multiple techniques
func (re *RuleExecutor) executeCompositeRule(rule *ModerationRule, content string, context ModerationContext) (*RuleResult, error) {
	var regexResult, keywordResult *RuleResult
	var err error

	// Execute regex component if available
	if rule.compiledPattern != nil {
		regexResult, err = re.executeRegexRule(rule, content, context)
		if err != nil {
			return nil, fmt.Errorf("regex component failed: %w", err)
		}
	}

	// Execute keyword component if available
	if len(rule.normalizedKeywords) > 0 {
		keywordResult, err = re.executeKeywordRule(rule, content, context)
		if err != nil {
			return nil, fmt.Errorf("keyword component failed: %w", err)
		}
	}

	// Combine results using weighted scoring
	combinedScore := 0.0
	combinedConfidence := 0.0
	matched := false
	reasons := []string{}
	var firstMatchDetails *MatchDetails

	if regexResult != nil && regexResult.Matched {
		combinedScore += regexResult.Score * 0.6 // 60% weight for regex
		combinedConfidence += regexResult.Confidence * 0.6
		matched = true
		reasons = append(reasons, "Regex: "+regexResult.Reason)
		if firstMatchDetails == nil {
			firstMatchDetails = regexResult.MatchDetails
		}
	}

	if keywordResult != nil && keywordResult.Matched {
		combinedScore += keywordResult.Score * 0.4 // 40% weight for keywords
		combinedConfidence += keywordResult.Confidence * 0.4
		matched = true
		reasons = append(reasons, "Keywords: "+keywordResult.Reason)
		if firstMatchDetails == nil {
			firstMatchDetails = keywordResult.MatchDetails
		}
	}

	// Apply composite rule's own weight
	combinedScore *= rule.Weight

	reason := ""
	if matched {
		reason = strings.Join(reasons, "; ")
	}

	return &RuleResult{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		Matched:      matched,
		Score:        combinedScore,
		Confidence:   combinedConfidence,
		Action:       rule.Action,
		Category:     rule.Category,
		Reason:       reason,
		MatchDetails: firstMatchDetails,
	}, nil
}

// Helper methods

// ruleAppliestoContext checks if a rule applies to the given context
func (re *RuleExecutor) ruleAppliestoContext(rule *ModerationRule, context ModerationContext) bool {
	if rule.Context == nil {
		return true // No context restrictions
	}

	ctx := rule.Context

	// Check user type restrictions
	if len(ctx.UserTypes) > 0 {
		userTypeMatch := false
		for _, allowedType := range ctx.UserTypes {
			if strings.EqualFold(context.UserType, allowedType) {
				userTypeMatch = true
				break
			}
		}
		if !userTypeMatch {
			return false
		}
	}

	// Check content type restrictions
	if len(ctx.ContentTypes) > 0 {
		contentTypeMatch := false
		for _, allowedType := range ctx.ContentTypes {
			if strings.EqualFold(context.ContentType, allowedType) {
				contentTypeMatch = true
				break
			}
		}
		if !contentTypeMatch {
			return false
		}
	}

	// Check channel restrictions
	if len(ctx.Channels) > 0 {
		channelMatch := false
		for _, allowedChannel := range ctx.Channels {
			if strings.EqualFold(context.Channel, allowedChannel) {
				channelMatch = true
				break
			}
		}
		if !channelMatch {
			return false
		}
	}

	// Check time restrictions
	if len(ctx.TimeRanges) > 0 {
		if !re.isWithinTimeRange(ctx.TimeRanges, context.Timestamp) {
			return false
		}
	}

	return true
}

// contentMeetsConditions checks if content meets rule conditions
func (re *RuleExecutor) contentMeetsConditions(rule *ModerationRule, content string) bool {
	if rule.Conditions == nil {
		return true
	}

	conditions := rule.Conditions

	// Check length constraints
	contentLength := len(content)
	if conditions.MinLength != nil && contentLength < *conditions.MinLength {
		return false
	}
	if conditions.MaxLength != nil && contentLength > *conditions.MaxLength {
		return false
	}

	return true
}

// isWithinTimeRange checks if current time falls within any of the specified time ranges
func (re *RuleExecutor) isWithinTimeRange(timeRanges []TimeRange, timestamp time.Time) bool {
	for _, tr := range timeRanges {
		if re.isInTimeRange(tr, timestamp) {
			return true
		}
	}
	return false
}

// isInTimeRange checks if timestamp falls within a specific time range
func (re *RuleExecutor) isInTimeRange(tr TimeRange, timestamp time.Time) bool {
	// Load timezone
	loc := time.UTC
	if tr.Timezone != "" {
		if l, err := time.LoadLocation(tr.Timezone); err == nil {
			loc = l
		}
	}

	// Convert timestamp to specified timezone
	localTime := timestamp.In(loc)
	
	// Check day of week
	if len(tr.Days) > 0 {
		dayMatch := false
		currentDay := strings.ToLower(localTime.Weekday().String())
		for _, day := range tr.Days {
			if strings.ToLower(day) == currentDay {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			return false
		}
	}

	// Parse and check time range
	startTime, err := time.Parse("15:04", tr.Start)
	if err != nil {
		return false
	}
	
	endTime, err := time.Parse("15:04", tr.End)
	if err != nil {
		return false
	}

	// Convert current time to comparable format
	currentTimeOfDay := localTime.Format("15:04")
	currentParsed, err := time.Parse("15:04", currentTimeOfDay)
	if err != nil {
		return false
	}

	// Handle overnight time ranges (e.g., 22:00 to 06:00)
	if endTime.Before(startTime) {
		return currentParsed.After(startTime) || currentParsed.Before(endTime)
	}

	return currentParsed.After(startTime) && currentParsed.Before(endTime)
}

// Confidence calculation methods

// calculateRegexConfidence calculates confidence for regex matches
func (re *RuleExecutor) calculateRegexConfidence(rule *ModerationRule, matches [][]string, content string) float64 {
	// Base confidence from rule weight
	confidence := rule.Weight

	// Increase confidence based on number of matches
	matchCount := len(matches)
	if matchCount > 1 {
		confidence += 0.1 * float64(matchCount-1)
	}

	// Increase confidence based on match length relative to content
	if len(matches) > 0 && len(matches[0]) > 0 {
		matchLength := len(matches[0][0])
		contentLength := len(content)
		if contentLength > 0 {
			matchRatio := float64(matchLength) / float64(contentLength)
			confidence += matchRatio * 0.2
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// calculateKeywordConfidence calculates confidence for keyword matches
func (re *RuleExecutor) calculateKeywordConfidence(rule *ModerationRule, matchedKeywords []string) float64 {
	// Base confidence from rule weight
	confidence := rule.Weight

	// Increase confidence based on match ratio
	totalKeywords := len(rule.normalizedKeywords)
	matchedCount := len(matchedKeywords)
	if totalKeywords > 0 {
		matchRatio := float64(matchedCount) / float64(totalKeywords)
		confidence += matchRatio * 0.3
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// simulateMLProcessing simulates ML model processing
func (re *RuleExecutor) simulateMLProcessing(content string, modelName string) (float64, float64) {
	// This is a placeholder for actual ML processing
	// In a real implementation, this would call external ML services or models
	
	contentLower := strings.ToLower(content)
	
	// Simple heuristic scoring based on content characteristics
	score := 0.0
	confidence := 0.5

	// Check for aggressive language patterns
	aggressivePatterns := []string{
		"hate", "kill", "murder", "die", "threat", "violence",
		"stupid", "idiot", "moron", "disgusting", "awful",
	}
	
	for _, pattern := range aggressivePatterns {
		if strings.Contains(contentLower, pattern) {
			score += 0.3
			confidence += 0.1
		}
	}

	// Check for excessive caps (yelling)
	capsCount := 0
	for _, r := range content {
		if r >= 'A' && r <= 'Z' {
			capsCount++
		}
	}
	if len(content) > 0 {
		capsRatio := float64(capsCount) / float64(len(content))
		if capsRatio > 0.5 {
			score += 0.2
			confidence += 0.05
		}
	}

	// Cap values
	if score > 1.0 {
		score = 1.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return score, confidence
}

// extractContext extracts surrounding text context for a match
func (re *RuleExecutor) extractContext(content string, position int, matchLength int) string {
	contextRadius := 50 // Characters before and after
	
	start := position - contextRadius
	if start < 0 {
		start = 0
	}
	
	end := position + matchLength + contextRadius
	if end > len(content) {
		end = len(content)
	}
	
	context := content[start:end]
	
	// Add ellipsis if truncated
	if start > 0 {
		context = "..." + context
	}
	if end < len(content) {
		context = context + "..."
	}
	
	return context
}

// Cache management methods

// getCachedResult retrieves a cached result for a rule and content
func (re *RuleExecutor) getCachedResult(rule *ModerationRule, content string) *RuleResult {
	if re.cache == nil {
		return nil
	}

	re.cache.mutex.RLock()
	defer re.cache.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", rule.ID, content)
	if entry, exists := re.cache.cache[key]; exists {
		// Check if cache entry is still valid
		if time.Since(entry.Timestamp) <= re.cache.ttl {
			return entry.Result
		}
		// Clean up expired entry
		delete(re.cache.cache, key)
	}

	return nil
}

// cacheResult caches a rule execution result
func (re *RuleExecutor) cacheResult(rule *ModerationRule, content string, result *RuleResult) {
	if re.cache == nil {
		return
	}

	re.cache.mutex.Lock()
	defer re.cache.mutex.Unlock()

	// Clean up old entries if cache is full
	if len(re.cache.cache) >= re.cache.maxSize {
		re.evictOldestEntries()
	}

	key := fmt.Sprintf("%s:%s", rule.ID, content)
	re.cache.cache[key] = &RuleCacheEntry{
		Result:    result,
		Timestamp: time.Now(),
	}
}

// evictOldestEntries removes the oldest cache entries
func (re *RuleExecutor) evictOldestEntries() {
	// Simple LRU eviction - remove 25% of entries
	entriesToRemove := len(re.cache.cache) / 4
	if entriesToRemove == 0 {
		entriesToRemove = 1
	}

	// Collect entries with timestamps
	type entryWithKey struct {
		key       string
		timestamp time.Time
	}
	
	entries := make([]entryWithKey, 0, len(re.cache.cache))
	for key, entry := range re.cache.cache {
		entries = append(entries, entryWithKey{key: key, timestamp: entry.Timestamp})
	}

	// Sort by timestamp (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp.Before(entries[j].timestamp)
	})

	// Remove oldest entries
	for i := 0; i < entriesToRemove && i < len(entries); i++ {
		delete(re.cache.cache, entries[i].key)
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}