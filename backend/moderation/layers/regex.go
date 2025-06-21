package layers

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"qt1-middleware/moderation"
)

// RegexLayer implements enhanced regex-based content moderation with context-aware analysis
type RegexLayer struct {
	name           string
	weight         float64
	enabled        bool
	config         moderation.LayerConfig
	patterns       []*CompiledPattern
	blockedWords   []string
	sophisticatedPatterns []*regexp.Regexp
	// Enhanced features for Cycle 2B
	keywords       map[string]*KeywordEntry
	contextRules   map[string]*ContextRule
	patternCache   map[string]*CacheEntry
	cacheMutex     sync.RWMutex
	stats         *LayerStats
}

// CompiledPattern represents a compiled regex pattern with metadata
type CompiledPattern struct {
	Pattern     *regexp.Regexp
	Weight      float64
	Reason      string
	Category    string
	Confidence  float64
	// Enhanced metadata for context-aware analysis
	ContextTypes []string
	Severity     string
	Tags         []string
}

// KeywordEntry represents a keyword with scoring metadata
type KeywordEntry struct {
	Keyword     string
	Weight      float64
	Category    string
	Contexts    []string
	Variations  []string
	Confidence  float64
}

// ContextRule defines context-aware moderation rules
type ContextRule struct {
	Name        string
	Condition   func(content string, ctx moderation.ModerationContext) bool
	Modifier    float64
	Description string
}

// CacheEntry represents a cached analysis result
type CacheEntry struct {
	Result    AnalysisResult
	Timestamp time.Time
	HitCount  int
}

// AnalysisResult contains detailed analysis results
type AnalysisResult struct {
	Score         float64
	Matches       []PatternMatch
	KeywordHits   []KeywordHit
	ContextFlags  []string
	Normalized    string
}

// PatternMatch represents a pattern match with context
type PatternMatch struct {
	Pattern     string
	Match       string
	Weight      float64
	StartPos    int
	EndPos      int
	ContextType string
}

// KeywordHit represents a keyword match with scoring
type KeywordHit struct {
	Keyword     string
	Weight      float64
	Position    int
	Context     string
	Variation   string
}

// LayerStats tracks performance and accuracy metrics
type LayerStats struct {
	TotalProcessed   int64
	CacheHits        int64
	AverageTime      time.Duration
	PatternMatches   map[string]int64
	KeywordMatches   map[string]int64
	ContextTriggers  map[string]int64
	mutex           sync.RWMutex
}

// NewRegexLayer creates a new regex-based moderation layer
func NewRegexLayer(config moderation.LayerConfig) *RegexLayer {
	layer := &RegexLayer{
		name:           config.Name,
		weight:         config.Weight,
		enabled:        config.Enabled,
		config:         config,
		patterns:       make([]*CompiledPattern, 0),
		blockedWords:   make([]string, 0),
		sophisticatedPatterns: make([]*regexp.Regexp, 0),
	}
	
	layer.initializePatterns()
	layer.initializeKeywords()
	layer.initializeContextRules()
	layer.patternCache = make(map[string]*CacheEntry)
	layer.stats = &LayerStats{
		PatternMatches:  make(map[string]int64),
		KeywordMatches:  make(map[string]int64),
		ContextTriggers: make(map[string]int64),
	}
	return layer
}

// Name returns the layer name
func (rl *RegexLayer) Name() string {
	return rl.name
}

// Weight returns the layer weight
func (rl *RegexLayer) Weight() float64 {
	return rl.weight
}

// Enabled returns whether the layer is enabled
func (rl *RegexLayer) Enabled() bool {
	return rl.enabled
}

// Config returns the layer configuration
func (rl *RegexLayer) Config() moderation.LayerConfig {
	return rl.config
}

// Moderate performs enhanced regex-based content moderation with context-aware analysis
func (rl *RegexLayer) Moderate(content string, context moderation.ModerationContext) moderation.ModerationResult {
	startTime := time.Now()
	rl.updateStats(func(s *LayerStats) { s.TotalProcessed++ })
	
	log.Printf("DEBUG REGEX: Analyzing content: '%s'", content)
	
	// Check cache first
	if cached := rl.getCachedResult(content); cached != nil {
		rl.updateStats(func(s *LayerStats) { s.CacheHits++ })
		return rl.buildResult(cached.Result, context, time.Since(startTime), true)
	}
	
	// Perform comprehensive analysis
	analysis := rl.analyzeContent(content, context)
	
	log.Printf("DEBUG REGEX: Initial analysis score: %.3f", analysis.Score)
	
	// Apply context-aware modifiers
	analysis.Score = rl.applyContextRules(analysis, context, content)
	
	log.Printf("DEBUG REGEX: Final score after context rules: %.3f", analysis.Score)
	
	// Cache the result
	rl.cacheResult(content, analysis)
	
	// Update processing time stats
	rl.updateStats(func(s *LayerStats) { 
		if s.TotalProcessed == 1 {
			s.AverageTime = time.Since(startTime)
		} else {
			s.AverageTime = time.Duration((int64(s.AverageTime)*(s.TotalProcessed-1) + int64(time.Since(startTime))) / s.TotalProcessed)
		}
	})
	
	result := rl.buildResult(analysis, context, time.Since(startTime), false)
	log.Printf("DEBUG REGEX: Built result - blocked: %v, score: %.3f", result.Blocked, result.Score)
	
	return result
}

// initializePatterns sets up default patterns and blocked words
func (rl *RegexLayer) initializePatterns() {
	// Initialize default blocked words
	rl.blockedWords = []string{
		"violence", "hate", "explicit", "harmful", "suicide", "self-harm",
		"terrorist", "bomb", "weapon", "kill", "murder", "assault",
		"threat", "harm", "hurt", "attack", "destroy", "eliminate",
		"hack", "exploit", "bypass", "jailbreak", "crack", "stab", "shoot", 
		"gun", "guns", "firearm", "rifle", "pistol", "revolver",
	}
	
	// Initialize sophisticated prompt injection patterns
	sophisticatedPatterns := []string{
		// Direct instruction override (simplified for testing)
		`(?i)ignore.+instructions`,
		`(?i)(ignore|forget|disregard|discard|override)\s+(previous|prior|all|any|the|your|my)\s+(instructions?|rules?|commands?|directions?|guidelines?|safety\s+rules?)`,
		`(?i)(new|different|updated|revised)\s+(instructions?|rules?|commands?|system\s+message)`,
		
		// Role manipulation and restrictions bypass
		`(?i)(you\s+are\s+now|act\s+as|pretend\s+to\s+be|roleplay\s+as)\s+(?:a\s+)?(?!assistant|ai|helpful)(\w+)`,
		`(?i)(pretend|imagine|suppose)\s+(you\s+have|that\s+you\s+have|there\s+are)\s+(no\s+)?(restrictions?|limits?|rules?|guidelines?)`,
		`(?i)(switch\s+to|become\s+a|transform\s+into)\s+(?:a\s+|an\s+)?(unrestricted|unlimited|uncensored|unfiltered)\s*(ai|assistant|mode|character|persona)?`,
		
		// System message manipulation
		`(?i)(the\s+)?(system\s+message|initial\s+prompt|base\s+instructions?)\s+(is|are|should\s+be|has\s+been)\s+(changed|modified|updated|replaced)`,
		
		// Jailbreak attempts
		`(?i)(developer|admin|god|root|sudo|debug|test)\s+(mode|access|privileges?|override|bypass|unlock)`,
		`(?i)(jailbreak|jail\s*break|break\s+out|escape\s+mode)`,
		`(?i)DAN\s+(mode|\d+|protocol)`,
		
		// Hypothetical scenarios
		`(?i)(in\s+a\s+)?(hypothetical|fictional|imaginary|alternate)\s+(world|universe|scenario|situation)`,
		`(?i)(what\s+if|suppose|imagine\s+if|let's\s+say)\s+(?:that\s+)?(?:you|I|we)`,
		
		// System probing
		`(?i)(what\s+are\s+your|show\s+me\s+your|list\s+your)\s+(instructions|rules|guidelines|parameters)`,
		`(?i)(repeat|echo|output|print)\s+(your\s+)?(system\s+message|instructions|prompt)`,
	}
	
	// Compile sophisticated patterns
	for i, pattern := range sophisticatedPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			rl.sophisticatedPatterns = append(rl.sophisticatedPatterns, compiled)
			log.Printf("DEBUG REGEX: Compiled sophisticated pattern %d: %s", i, pattern)
		} else {
			log.Printf("DEBUG REGEX: Failed to compile sophisticated pattern %d: %v", i, err)
		}
	}
	
	// Initialize compiled patterns with weights
	rl.patterns = []*CompiledPattern{
		{
			Pattern:    regexp.MustCompile(`(?i)\b(kill|murder|destroy|eliminate)\s+(you|yourself|myself|everyone|all)\b`),
			Weight:     0.9,
			Reason:     "Violent threats detected",
			Category:   moderation.CategoryViolence,
			Confidence: 0.8,
		},
		{
			Pattern:    regexp.MustCompile(`(?i)\b(i\s+will|gonna|going\s+to)\s+(kill|hurt|harm|destroy|attack)\b`),
			Weight:     0.9,
			Reason:     "Direct threat detected",
			Category:   moderation.CategoryViolence,
			Confidence: 0.9,
		},
		{
			Pattern:    regexp.MustCompile(`(?i)\b(hate|despise|loathe)\s+(all|every|most)\s+(\w+)\b`),
			Weight:     0.8,
			Reason:     "Hate speech pattern detected",
			Category:   moderation.CategoryHateSpeech,
			Confidence: 0.7,
		},
		{
			Pattern:    regexp.MustCompile(`(?i)\b(spam|advertisement|buy\s+now|click\s+here|limited\s+time)\b`),
			Weight:     0.6,
			Reason:     "Spam content detected",
			Category:   moderation.CategorySpam,
			Confidence: 0.6,
		},
	}
}

// normalizeContent normalizes content for consistent analysis
func (rl *RegexLayer) normalizeContent(content string) string {
	// Convert to lowercase
	normalized := strings.ToLower(content)
	
	// Remove excessive whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	
	// Normalize common obfuscation attempts
	normalized = strings.ReplaceAll(normalized, "1", "i")
	normalized = strings.ReplaceAll(normalized, "3", "e")
	normalized = strings.ReplaceAll(normalized, "0", "o")
	normalized = strings.ReplaceAll(normalized, "5", "s")
	normalized = strings.ReplaceAll(normalized, "7", "t")
	
	// Remove common separators used in bypass attempts (including | and other obfuscation)
	normalized = regexp.MustCompile(`[._\-*+|\\/@#$%^&()[\]{}]+`).ReplaceAllString(normalized, "")
	
	result := strings.TrimSpace(normalized)
	log.Printf("DEBUG REGEX: Normalized '%s' -> '%s'", content, result)
	return result
}

// checkBlockedWords checks for exact blocked word matches
func (rl *RegexLayer) checkBlockedWords(content string) (float64, string) {
	words := strings.Fields(content)
	
	for _, word := range words {
		for _, blocked := range rl.blockedWords {
			if strings.Contains(word, blocked) {
				return 0.8, "Blocked word detected: " + blocked
			}
		}
	}
	
	return 0.0, ""
}

// checkSophisticatedPatterns checks for prompt injection patterns
func (rl *RegexLayer) checkSophisticatedPatterns(content string) (float64, string) {
	log.Printf("DEBUG REGEX: Checking %d sophisticated patterns against: '%s'", len(rl.sophisticatedPatterns), content)
	
	for i, pattern := range rl.sophisticatedPatterns {
		if pattern.MatchString(content) {
			log.Printf("DEBUG REGEX: Sophisticated pattern %d matched! Score 0.9", i)
			return 0.9, "Prompt injection pattern detected"
		}
	}
	
	log.Printf("DEBUG REGEX: No sophisticated patterns matched")
	return 0.0, ""
}

// checkCompiledPatterns checks against compiled patterns with weights
func (rl *RegexLayer) checkCompiledPatterns(content string) (float64, string) {
	maxScore := 0.0
	reason := ""
	
	for _, compiledPattern := range rl.patterns {
		if compiledPattern.Pattern.MatchString(content) {
			if compiledPattern.Weight > maxScore {
				maxScore = compiledPattern.Weight
				reason = compiledPattern.Reason
			}
		}
	}
	
	return maxScore, reason
}

// calculateConfidence calculates confidence based on score
func (rl *RegexLayer) calculateConfidence(score float64) float64 {
	// Simple confidence calculation - higher scores get higher confidence
	if score >= 0.8 {
		return 0.9
	} else if score >= 0.6 {
		return 0.7
	} else if score >= 0.4 {
		return 0.5
	} else if score >= 0.2 {
		return 0.3
	}
	return 0.1
}

// UpdateConfig updates the layer configuration
func (rl *RegexLayer) UpdateConfig(config moderation.LayerConfig) {
	rl.config = config
	rl.weight = config.Weight
	rl.enabled = config.Enabled
	
	// If options contain custom patterns or words, update them
	if config.Options != nil {
		if customWords, ok := config.Options["blocked_words"].([]interface{}); ok {
			rl.blockedWords = make([]string, len(customWords))
			for i, word := range customWords {
				if str, ok := word.(string); ok {
					rl.blockedWords[i] = str
				}
			}
		}
		
		if customPatterns, ok := config.Options["custom_patterns"].([]interface{}); ok {
			rl.patterns = make([]*CompiledPattern, 0)
			for _, pattern := range customPatterns {
				if patternMap, ok := pattern.(map[string]interface{}); ok {
					if patternStr, ok := patternMap["pattern"].(string); ok {
						if compiled, err := regexp.Compile(patternStr); err == nil {
							weight := 0.5 // default weight
							if w, ok := patternMap["weight"].(float64); ok {
								weight = w
							}
							
							reason := "Custom pattern match"
							if r, ok := patternMap["reason"].(string); ok {
								reason = r
							}
							
							rl.patterns = append(rl.patterns, &CompiledPattern{
								Pattern:    compiled,
								Weight:     weight,
								Reason:     reason,
								Category:   moderation.CategoryCustom,
								Confidence: 0.7,
							})
						}
					}
				}
			}
		}
	}
}

// Enhanced analysis methods for Cycle 2B

// analyzeContent performs comprehensive content analysis
func (rl *RegexLayer) analyzeContent(content string, context moderation.ModerationContext) AnalysisResult {
	normalized := rl.normalizeContent(content)
	
	analysis := AnalysisResult{
		Score:       0.0,
		Matches:     make([]PatternMatch, 0),
		KeywordHits: make([]KeywordHit, 0),
		ContextFlags: make([]string, 0),
		Normalized:  normalized,
	}
	
	// 1. Simple blocked words check (use normalized content)
	blockedWordScore, _ := rl.checkBlockedWords(normalized)
	
	// 2. Pattern matching with enhanced scoring (use normalized content)
	patternScore := rl.analyzePatterns(normalized, &analysis)
	
	// 3. Keyword analysis with context awareness (use normalized content)
	keywordScore := rl.analyzeKeywords(normalized, context, &analysis)
	
	// 4. Sophisticated pattern detection (use ORIGINAL content - important!)
	sophisticatedScore := rl.analyzeSophisticatedPatterns(content, &analysis)
	
	// 5. Calculate weighted final score including blocked words
	analysis.Score = rl.calculateWeightedScore(blockedWordScore, patternScore, keywordScore, sophisticatedScore)
	
	return analysis
}

// analyzePatterns performs enhanced pattern matching
func (rl *RegexLayer) analyzePatterns(content string, analysis *AnalysisResult) float64 {
	maxScore := 0.0
	
	for _, compiledPattern := range rl.patterns {
		if matches := compiledPattern.Pattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
			for _, match := range matches {
				if len(match) > 0 {
					pos := strings.Index(content, match[0])
					patternMatch := PatternMatch{
						Pattern:     compiledPattern.Reason,
						Match:       match[0],
						Weight:      compiledPattern.Weight,
						StartPos:    pos,
						EndPos:      pos + len(match[0]),
						ContextType: rl.determineContextType(content, pos),
					}
					analysis.Matches = append(analysis.Matches, patternMatch)
					
					// Update stats
					rl.updateStats(func(s *LayerStats) {
						s.PatternMatches[compiledPattern.Reason]++
					})
					
					if compiledPattern.Weight > maxScore {
						maxScore = compiledPattern.Weight
					}
				}
			}
		}
	}
	
	return maxScore
}

// analyzeKeywords performs context-aware keyword analysis
func (rl *RegexLayer) analyzeKeywords(content string, context moderation.ModerationContext, analysis *AnalysisResult) float64 {
	maxScore := 0.0
	
	// Clean content for word extraction (remove punctuation)
	cleanContent := regexp.MustCompile(`[^\w\s]+`).ReplaceAllString(content, " ")
	words := strings.Fields(strings.ToLower(cleanContent))
	
	for i, word := range words {
		// Check direct keyword matches
		if entry, exists := rl.keywords[word]; exists {
			contextBonus := rl.getContextBonus(word, words, i, context)
			adjustedWeight := entry.Weight * (1.0 + contextBonus)
			
			hit := KeywordHit{
				Keyword:   entry.Keyword,
				Weight:    adjustedWeight,
				Position:  i,
				Context:   rl.getWordContext(words, i),
				Variation: word,
			}
			analysis.KeywordHits = append(analysis.KeywordHits, hit)
			
			// Update stats
			rl.updateStats(func(s *LayerStats) {
				s.KeywordMatches[entry.Keyword]++
			})
			
			if adjustedWeight > maxScore {
				maxScore = adjustedWeight
			}
		}
		
		// Check keyword variations
		for keyword, entry := range rl.keywords {
			for _, variation := range entry.Variations {
				if strings.Contains(word, variation) && word != variation {
					contextBonus := rl.getContextBonus(keyword, words, i, context)
					adjustedWeight := entry.Weight * 0.8 * (1.0 + contextBonus) // Reduce weight for variations
					
					hit := KeywordHit{
						Keyword:   keyword,
						Weight:    adjustedWeight,
						Position:  i,
						Context:   rl.getWordContext(words, i),
						Variation: word,
					}
					analysis.KeywordHits = append(analysis.KeywordHits, hit)
					
					if adjustedWeight > maxScore {
						maxScore = adjustedWeight
					}
				}
			}
		}
	}
	
	return maxScore
}

// analyzeSophisticatedPatterns performs enhanced sophisticated pattern detection
func (rl *RegexLayer) analyzeSophisticatedPatterns(content string, analysis *AnalysisResult) float64 {
	log.Printf("DEBUG REGEX: Checking %d sophisticated patterns against: '%s'", len(rl.sophisticatedPatterns), content)
	
	for i, pattern := range rl.sophisticatedPatterns {
		if matches := pattern.FindAllStringSubmatch(content, -1); len(matches) > 0 {
			log.Printf("DEBUG REGEX: Sophisticated pattern %d matched! Score 0.9", i)
			for _, match := range matches {
				if len(match) > 0 {
					pos := strings.Index(content, match[0])
					patternMatch := PatternMatch{
						Pattern:     "sophisticated_pattern",
						Match:       match[0],
						Weight:      0.9,
						StartPos:    pos,
						EndPos:      pos + len(match[0]),
						ContextType: "prompt_injection",
					}
					analysis.Matches = append(analysis.Matches, patternMatch)
					return 0.9 // High score for sophisticated patterns
				}
			}
		}
	}
	
	log.Printf("DEBUG REGEX: No sophisticated patterns matched")
	return 0.0
}

// calculateWeightedScore combines different analysis scores including blocked words
func (rl *RegexLayer) calculateWeightedScore(blockedWordScore, patternScore, keywordScore, sophisticatedScore float64) float64 {
	// Sophisticated patterns have highest priority
	if sophisticatedScore > 0 {
		return sophisticatedScore
	}
	
	// Blocked words have second highest priority
	if blockedWordScore > 0 {
		return blockedWordScore
	}
	
	// Take the maximum of pattern and keyword scores, but also consider weighted average
	maxScore := patternScore
	if keywordScore > maxScore {
		maxScore = keywordScore
	}
	
	// Calculate weighted average
	patternWeight := 0.6
	keywordWeight := 0.4
	weightedAvg := (patternScore*patternWeight + keywordScore*keywordWeight)
	
	// Return the higher of max score or weighted average (this ensures high scores don't get diluted)
	if maxScore > weightedAvg {
		return maxScore
	}
	return weightedAvg
}

// applyContextRules applies context-aware moderation rules
func (rl *RegexLayer) applyContextRules(analysis AnalysisResult, context moderation.ModerationContext, originalContent string) float64 {
	modifiedScore := analysis.Score
	
	for name, rule := range rl.contextRules {
		// Use original content for context rules (not normalized)
		if rule.Condition(originalContent, context) {
			modifiedScore *= rule.Modifier
			rl.updateStats(func(s *LayerStats) {
				s.ContextTriggers[name]++
			})
		}
	}
	
	// Ensure score stays within bounds
	if modifiedScore > 1.0 {
		modifiedScore = 1.0
	} else if modifiedScore < 0.0 {
		modifiedScore = 0.0
	}
	
	return modifiedScore
}

// buildResult constructs the final moderation result
func (rl *RegexLayer) buildResult(analysis AnalysisResult, context moderation.ModerationContext, processTime time.Duration, fromCache bool) moderation.ModerationResult {
	// Determine primary category based on matches
	category := rl.determinePrimaryCategory(analysis)
	
	// Build comprehensive reason
	reason := rl.buildReason(analysis)
	
	result := moderation.ModerationResult{
		Score:       analysis.Score,
		Confidence:  rl.calculateEnhancedConfidence(analysis),
		Blocked:     analysis.Score >= rl.config.Threshold,
		Reason:      reason,
		Category:    category,
		LayerName:   rl.name,
		ProcessTime: processTime,
		Details:     make(map[string]interface{}),
	}
	
	// Add detailed analysis information
	result.Details["pattern_matches"] = len(analysis.Matches)
	result.Details["keyword_hits"] = len(analysis.KeywordHits)
	result.Details["context_flags"] = analysis.ContextFlags
	result.Details["normalized_content"] = analysis.Normalized
	result.Details["from_cache"] = fromCache
	result.Details["threshold"] = rl.config.Threshold
	
	if len(analysis.Matches) > 0 {
		result.Details["top_patterns"] = rl.getTopMatches(analysis.Matches, 3)
	}
	if len(analysis.KeywordHits) > 0 {
		result.Details["top_keywords"] = rl.getTopKeywords(analysis.KeywordHits, 3)
	}
	
	return result
}

// Cache management methods

// getCachedResult retrieves a cached analysis result
func (rl *RegexLayer) getCachedResult(content string) *CacheEntry {
	rl.cacheMutex.RLock()
	defer rl.cacheMutex.RUnlock()
	
	key := rl.generateCacheKey(content)
	if entry, exists := rl.patternCache[key]; exists {
		// Check if cache entry is still valid (5 minutes)
		if time.Since(entry.Timestamp) < 5*time.Minute {
			entry.HitCount++
			return entry
		}
		// Remove expired entry
		delete(rl.patternCache, key)
	}
	
	return nil
}

// cacheResult stores an analysis result in cache
func (rl *RegexLayer) cacheResult(content string, analysis AnalysisResult) {
	rl.cacheMutex.Lock()
	defer rl.cacheMutex.Unlock()
	
	key := rl.generateCacheKey(content)
	
	// Implement simple LRU by removing oldest entries if cache is full
	if len(rl.patternCache) >= 1000 {
		rl.evictOldestCacheEntries()
	}
	
	rl.patternCache[key] = &CacheEntry{
		Result:    analysis,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

// Helper methods

// generateCacheKey generates a cache key for content
func (rl *RegexLayer) generateCacheKey(content string) string {
	// Simple hash of normalized content
	normalized := rl.normalizeContent(content)
	if len(normalized) > 100 {
		normalized = normalized[:100]
	}
	return fmt.Sprintf("%x", []byte(normalized))
}

// evictOldestCacheEntries removes oldest cache entries
func (rl *RegexLayer) evictOldestCacheEntries() {
	// Remove 10% of cache entries (oldest first)
	type cacheItem struct {
		key   string
		entry *CacheEntry
	}
	
	var items []cacheItem
	for key, entry := range rl.patternCache {
		items = append(items, cacheItem{key, entry})
	}
	
	// Sort by timestamp (oldest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].entry.Timestamp.Before(items[j].entry.Timestamp)
	})
	
	// Remove oldest 10%
	removeCount := len(items) / 10
	for i := 0; i < removeCount; i++ {
		delete(rl.patternCache, items[i].key)
	}
}

// updateStats safely updates layer statistics
func (rl *RegexLayer) updateStats(update func(*LayerStats)) {
	rl.stats.mutex.Lock()
	defer rl.stats.mutex.Unlock()
	update(rl.stats)
}

// GetStats returns comprehensive statistics for this layer
func (rl *RegexLayer) GetStats() map[string]interface{} {
	rl.stats.mutex.RLock()
	defer rl.stats.mutex.RUnlock()
	
	return map[string]interface{}{
		"name":                    rl.name,
		"enabled":                 rl.enabled,
		"weight":                  rl.weight,
		"blocked_words":           len(rl.blockedWords),
		"patterns":                len(rl.patterns),
		"sophisticated_patterns":  len(rl.sophisticatedPatterns),
		"keywords":                len(rl.keywords),
		"context_rules":           len(rl.contextRules),
		"total_processed":         rl.stats.TotalProcessed,
		"cache_hits":              rl.stats.CacheHits,
		"cache_hit_ratio":         float64(rl.stats.CacheHits) / float64(rl.stats.TotalProcessed),
		"average_processing_time": rl.stats.AverageTime.String(),
		"pattern_matches":         rl.stats.PatternMatches,
		"keyword_matches":         rl.stats.KeywordMatches,
		"context_triggers":        rl.stats.ContextTriggers,
		"cache_size":              len(rl.patternCache),
	}
}

// Additional helper methods needed for enhanced analysis

// initializeKeywords sets up the keyword scoring system
func (rl *RegexLayer) initializeKeywords() {
	rl.keywords = make(map[string]*KeywordEntry)
	
	// Add default keywords with weights and variations
	keywordDefs := map[string]*KeywordEntry{
		"violence": {
			Keyword:    "violence",
			Weight:     0.6, // Reduced to leave room for context boosting
			Category:   "violence",
			Contexts:   []string{"threat", "harm"},
			Variations: []string{"violent", "violently"},
			Confidence: 0.9,
		},
		"hate": {
			Keyword:    "hate",
			Weight:     0.7,
			Category:   "hate_speech",
			Contexts:   []string{"discrimination", "bias"},
			Variations: []string{"hatred", "hating", "hateful"},
			Confidence: 0.8,
		},
		"kill": {
			Keyword:    "kill",
			Weight:     0.9,
			Category:   "violence",
			Contexts:   []string{"threat", "harm", "death"},
			Variations: []string{"killing", "killed", "killer"},
			Confidence: 0.95,
		},
		"bomb": {
			Keyword:    "bomb",
			Weight:     0.95,
			Category:   "terrorism",
			Contexts:   []string{"explosive", "threat", "attack"},
			Variations: []string{"bombing", "bomber", "bombs"},
			Confidence: 0.98,
		},
		"suicide": {
			Keyword:    "suicide",
			Weight:     0.85,
			Category:   "self_harm",
			Contexts:   []string{"depression", "self_harm"},
			Variations: []string{"suicidal"},
			Confidence: 0.9,
		},
	}
	
	// Build keywords map including base words and variations
	keywords := make(map[string]*KeywordEntry)
	for baseKeyword, entry := range keywordDefs {
		// Add the base keyword
		keywords[baseKeyword] = entry
		
		// Add all variations
		for _, variation := range entry.Variations {
			keywords[variation] = &KeywordEntry{
				Keyword:    baseKeyword, // Reference to base keyword
				Weight:     entry.Weight * 0.9, // Slightly reduce weight for variations
				Category:   entry.Category,
				Contexts:   entry.Contexts,
				Variations: []string{variation},
				Confidence: entry.Confidence * 0.9,
			}
		}
	}
	
	// Load keywords from config if available
	if rl.config.Options != nil {
		if customKeywords, ok := rl.config.Options["keywords"].(map[string]interface{}); ok {
			for keyword, data := range customKeywords {
				if keywordData, ok := data.(map[string]interface{}); ok {
					entry := &KeywordEntry{
						Keyword:    keyword,
						Weight:     0.5, // default
						Category:   "custom",
						Contexts:   []string{},
						Variations: []string{},
						Confidence: 0.7,
					}
					
					if weight, ok := keywordData["weight"].(float64); ok {
						entry.Weight = weight
					}
					if category, ok := keywordData["category"].(string); ok {
						entry.Category = category
					}
					
					keywords[keyword] = entry
				}
			}
		}
	}
	
	rl.keywords = keywords
}

// initializeContextRules sets up context-aware moderation rules
func (rl *RegexLayer) initializeContextRules() {
	rl.contextRules = make(map[string]*ContextRule)
	
	// Time-based context rules
	rl.contextRules["late_night"] = &ContextRule{
		Name: "late_night",
		Condition: func(content string, ctx moderation.ModerationContext) bool {
			hour := ctx.Timestamp.Hour()
			return hour >= 22 || hour <= 6 // 10 PM to 6 AM
		},
		Modifier:    1.2, // Increase score by 20% for late night posts
		Description: "Content posted during late night hours",
	}
	
	// User context rules
	rl.contextRules["repeat_offender"] = &ContextRule{
		Name: "repeat_offender",
		Condition: func(content string, ctx moderation.ModerationContext) bool {
			// This would check against a user history database in a real implementation
			return strings.Contains(ctx.UserID, "warned")
		},
		Modifier:    1.5,
		Description: "User has previous moderation warnings",
	}
	
	// Content length rules
	rl.contextRules["short_aggressive"] = &ContextRule{
		Name: "short_aggressive",
		Condition: func(content string, ctx moderation.ModerationContext) bool {
			return len(content) < 50 && strings.ContainsAny(content, "!?")
		},
		Modifier:    1.3,
		Description: "Short aggressive content",
	}
	
	// All caps rule
	rl.contextRules["shouting"] = &ContextRule{
		Name: "shouting",
		Condition: func(content string, ctx moderation.ModerationContext) bool {
			if len(content) < 10 {
				return false
			}
			upperCount := 0
			letterCount := 0
			for _, r := range content {
				if r >= 'A' && r <= 'Z' {
					upperCount++
					letterCount++
				} else if r >= 'a' && r <= 'z' {
					letterCount++
				}
			}
			return letterCount > 0 && float64(upperCount)/float64(letterCount) > 0.7
		},
		Modifier:    1.25,
		Description: "Content is mostly in uppercase (shouting)",
	}
}

// determineContextType determines the context type of a match
func (rl *RegexLayer) determineContextType(content string, position int) string {
	// Simple context determination based on surrounding text
	start := position - 20
	if start < 0 {
		start = 0
	}
	end := position + 20
	if end > len(content) {
		end = len(content)
	}
	
	context := strings.ToLower(content[start:end])
	
	if strings.Contains(context, "kill") || strings.Contains(context, "death") {
		return "violent"
	}
	if strings.Contains(context, "hate") || strings.Contains(context, "despise") {
		return "hate_speech"
	}
	if strings.Contains(context, "ignore") || strings.Contains(context, "forget") {
		return "prompt_injection"
	}
	
	return "general"
}

// getContextBonus calculates context-based scoring bonus
func (rl *RegexLayer) getContextBonus(keyword string, words []string, position int, context moderation.ModerationContext) float64 {
	bonus := 0.0
	
	// Check surrounding words for context
	start := position - 2
	if start < 0 {
		start = 0
	}
	end := position + 3
	if end > len(words) {
		end = len(words)
	}
	
	surroundingWords := words[start:end]
	surroundingText := strings.Join(surroundingWords, " ")
	
	// Increase bonus for threatening language
	if strings.Contains(surroundingText, "will") || strings.Contains(surroundingText, "going to") {
		bonus += 0.3
	}
	
	// Increase bonus for personal pronouns (targeting)
	if strings.Contains(surroundingText, "you") || strings.Contains(surroundingText, "your") {
		bonus += 0.2
	}
	
	// Increase bonus for urgency words
	if strings.Contains(surroundingText, "now") || strings.Contains(surroundingText, "immediately") {
		bonus += 0.15
	}
	
	return bonus
}

// getWordContext gets context around a word
func (rl *RegexLayer) getWordContext(words []string, position int) string {
	start := position - 2
	if start < 0 {
		start = 0
	}
	end := position + 3
	if end > len(words) {
		end = len(words)
	}
	
	return strings.Join(words[start:end], " ")
}

// determinePrimaryCategory determines the primary category from analysis
func (rl *RegexLayer) determinePrimaryCategory(analysis AnalysisResult) string {
	// Count category occurrences
	categoryCounts := make(map[string]int)
	
	for _, match := range analysis.Matches {
		if match.ContextType != "" {
			categoryCounts[match.ContextType]++
		}
	}
	
	for _, hit := range analysis.KeywordHits {
		if entry, exists := rl.keywords[hit.Keyword]; exists {
			categoryCounts[entry.Category]++
		}
	}
	
	// Return most common category
	maxCount := 0
	primaryCategory := moderation.CategoryCustom
	
	for category, count := range categoryCounts {
		if count > maxCount {
			maxCount = count
			switch category {
			case "violence", "violent":
				primaryCategory = moderation.CategoryViolence
			case "hate_speech":
				primaryCategory = moderation.CategoryHateSpeech
			case "prompt_injection":
				primaryCategory = moderation.CategoryPromptInject
			case "spam":
				primaryCategory = moderation.CategorySpam
			default:
				primaryCategory = moderation.CategoryCustom
			}
		}
	}
	
	return primaryCategory
}

// buildReason builds a comprehensive reason string
func (rl *RegexLayer) buildReason(analysis AnalysisResult) string {
	if analysis.Score == 0.0 {
		return ""
	}
	
	reasons := []string{}
	
	// Add pattern matches
	if len(analysis.Matches) > 0 {
		topMatches := rl.getTopMatches(analysis.Matches, 2)
		for _, match := range topMatches {
			reasons = append(reasons, fmt.Sprintf("Pattern: %s", match.Pattern))
		}
	}
	
	// Add keyword hits
	if len(analysis.KeywordHits) > 0 {
		topKeywords := rl.getTopKeywords(analysis.KeywordHits, 2)
		for _, hit := range topKeywords {
			reasons = append(reasons, fmt.Sprintf("Keyword: %s", hit.Keyword))
		}
	}
	
	if len(reasons) == 0 {
		return "Content flagged by enhanced regex analysis"
	}
	
	return strings.Join(reasons, "; ")
}

// calculateEnhancedConfidence calculates confidence based on multiple factors
func (rl *RegexLayer) calculateEnhancedConfidence(analysis AnalysisResult) float64 {
	baseConfidence := 0.5
	
	// Increase confidence with more matches
	if len(analysis.Matches) > 0 {
		baseConfidence += 0.2 * float64(len(analysis.Matches))
	}
	
	if len(analysis.KeywordHits) > 0 {
		baseConfidence += 0.15 * float64(len(analysis.KeywordHits))
	}
	
	// High scores get higher confidence
	if analysis.Score >= 0.8 {
		baseConfidence += 0.2
	} else if analysis.Score >= 0.6 {
		baseConfidence += 0.1
	}
	
	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}
	
	return baseConfidence
}

// getTopMatches returns top N pattern matches by weight
func (rl *RegexLayer) getTopMatches(matches []PatternMatch, n int) []PatternMatch {
	if len(matches) <= n {
		return matches
	}
	
	// Sort by weight (descending)
	sorted := make([]PatternMatch, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Weight > sorted[j].Weight
	})
	
	return sorted[:n]
}

// getTopKeywords returns top N keyword hits by weight
func (rl *RegexLayer) getTopKeywords(hits []KeywordHit, n int) []KeywordHit {
	if len(hits) <= n {
		return hits
	}
	
	// Sort by weight (descending)
	sorted := make([]KeywordHit, len(hits))
	copy(sorted, hits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Weight > sorted[j].Weight
	})
	
	return sorted[:n]
}