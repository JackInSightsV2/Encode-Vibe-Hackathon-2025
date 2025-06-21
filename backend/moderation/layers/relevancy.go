package layers

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"qt1-middleware/moderation"
)

// RelevancyLayer implements local keyword-based relevancy checking
type RelevancyLayer struct {
	name           string
	weight         float64
	enabled        bool
	config         moderation.LayerConfig
	domainKeywords map[string]float64  // keyword -> relevance score
	patterns       []*RelevancyPattern
	stats          *RelevancyStats
	cache          map[string]*RelevancyCacheEntry
	cacheMutex     sync.RWMutex
}

// RelevancyPattern represents a pattern with relevancy scoring
type RelevancyPattern struct {
	Pattern     *regexp.Regexp
	Score       float64  // positive for relevant, negative for irrelevant
	Description string
	Category    string
}

// RelevancyStats tracks layer performance
type RelevancyStats struct {
	TotalProcessed   int64
	RelevantCount    int64
	IrrelevantCount  int64
	CacheHits        int64
	AverageScore     float64
	AverageTime      time.Duration
	mutex           sync.RWMutex
}

// RelevancyCacheEntry caches relevancy analysis
type RelevancyCacheEntry struct {
	Score         float64
	Relevant      bool
	Keywords      []string
	Timestamp     time.Time
	HitCount      int
}

// RelevancyAnalysis contains the analysis results
type RelevancyAnalysis struct {
	Score           float64
	Relevant        bool
	MatchedKeywords []string
	MatchedPatterns []string
	Confidence      float64
	Reason          string
}

// NewRelevancyLayer creates a new local relevancy layer
func NewRelevancyLayer(config moderation.LayerConfig) *RelevancyLayer {
	layer := &RelevancyLayer{
		name:           config.Name,
		weight:         config.Weight,
		enabled:        config.Enabled,
		config:         config,
		domainKeywords: make(map[string]float64),
		patterns:       make([]*RelevancyPattern, 0),
		cache:          make(map[string]*RelevancyCacheEntry),
		stats:          &RelevancyStats{},
	}
	
	layer.initializeDefaultKeywords()
	layer.initializePatterns()
	layer.loadConfigKeywords()
	
	return layer
}

// Name returns the layer name
func (rl *RelevancyLayer) Name() string {
	return rl.name
}

// Weight returns the layer weight
func (rl *RelevancyLayer) Weight() float64 {
	return rl.weight
}

// Enabled returns whether the layer is enabled
func (rl *RelevancyLayer) Enabled() bool {
	return rl.enabled
}

// Config returns the layer configuration
func (rl *RelevancyLayer) Config() moderation.LayerConfig {
	return rl.config
}

// Moderate performs relevancy checking on content
func (rl *RelevancyLayer) Moderate(content string, context moderation.ModerationContext) moderation.ModerationResult {
	startTime := time.Now()
	rl.updateStats(func(s *RelevancyStats) { s.TotalProcessed++ })

	if !rl.enabled {
		return moderation.ModerationResult{
			Score:       0.0,
			Confidence:  0.0,
			Blocked:     false,
			Reason:      "Relevancy layer disabled",
			Category:    moderation.CategoryCustom,
			LayerName:   rl.name,
			ProcessTime: time.Since(startTime),
			Details: map[string]interface{}{
				"layer_enabled": false,
			},
		}
	}

	log.Printf("DEBUG RELEVANCY: Analyzing content: '%s'", content)

	// Check cache first
	if cached := rl.getCachedResult(content); cached != nil {
		rl.updateStats(func(s *RelevancyStats) { s.CacheHits++ })
		return rl.buildResultFromCache(cached, time.Since(startTime))
	}

	// Perform relevancy analysis
	analysis := rl.analyzeRelevancy(content, context)
	
	// Cache result
	rl.cacheResult(content, analysis)
	
	// Update stats
	rl.updateStats(func(s *RelevancyStats) {
		if analysis.Score > 0.5 {
			s.RelevantCount++
		} else {
			s.IrrelevantCount++
		}
		
		// Update average score
		if s.TotalProcessed == 1 {
			s.AverageScore = analysis.Score
		} else {
			s.AverageScore = (s.AverageScore*float64(s.TotalProcessed-1) + analysis.Score) / float64(s.TotalProcessed)
		}
		
		// Update average time
		processTime := time.Since(startTime)
		if s.TotalProcessed == 1 {
			s.AverageTime = processTime
		} else {
			avgNanos := (s.AverageTime.Nanoseconds()*int64(s.TotalProcessed-1) + processTime.Nanoseconds()) / int64(s.TotalProcessed)
			s.AverageTime = time.Duration(avgNanos)
		}
	})

	result := rl.buildResult(analysis, time.Since(startTime))
	log.Printf("DEBUG RELEVANCY: Final score: %.3f, relevant: %v", result.Score, result.Score > 0.5)
	
	return result
}

// analyzeRelevancy performs the actual relevancy analysis
func (rl *RelevancyLayer) analyzeRelevancy(content string, context moderation.ModerationContext) *RelevancyAnalysis {
	normalizedContent := strings.ToLower(strings.TrimSpace(content))
	
	analysis := &RelevancyAnalysis{
		Score:           0.5, // Start neutral
		MatchedKeywords: make([]string, 0),
		MatchedPatterns: make([]string, 0),
	}

	// 1. Keyword-based analysis
	keywordScore := rl.analyzeKeywords(normalizedContent, analysis)
	
	// 2. Pattern-based analysis
	patternScore := rl.analyzePatterns(normalizedContent, analysis)
	
	// 3. Context-based adjustments
	contextScore := rl.analyzeContext(normalizedContent, context, analysis)
	
	// 4. Calculate final score (weighted combination)
	analysis.Score = (keywordScore * 0.6) + (patternScore * 0.3) + (contextScore * 0.1)
	
	// Ensure score is in valid range
	if analysis.Score > 1.0 {
		analysis.Score = 1.0
	} else if analysis.Score < 0.0 {
		analysis.Score = 0.0
	}
	
	analysis.Relevant = analysis.Score > 0.5
	analysis.Confidence = rl.calculateConfidence(analysis)
	analysis.Reason = rl.buildReason(analysis)
	
	return analysis
}

// analyzeKeywords performs keyword-based relevancy scoring
func (rl *RelevancyLayer) analyzeKeywords(content string, analysis *RelevancyAnalysis) float64 {
	words := strings.Fields(content)
	if len(words) == 0 {
		return 0.5 // Neutral for empty content
	}
	
	totalScore := 0.0
	matchedCount := 0
	
	for _, word := range words {
		// Clean word (remove punctuation)
		cleanWord := regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		if len(cleanWord) < 2 {
			continue
		}
		
		// Check exact match
		if score, exists := rl.domainKeywords[cleanWord]; exists {
			totalScore += score
			matchedCount++
			analysis.MatchedKeywords = append(analysis.MatchedKeywords, cleanWord)
		}
		
		// Check partial matches for longer words
		if len(cleanWord) > 4 {
			for keyword, score := range rl.domainKeywords {
				if len(keyword) > 3 && strings.Contains(cleanWord, keyword) {
					totalScore += score * 0.7 // Reduce score for partial matches
					matchedCount++
					analysis.MatchedKeywords = append(analysis.MatchedKeywords, keyword + " (partial)")
				}
			}
		}
	}
	
	if matchedCount == 0 {
		return 0.5 // Neutral if no keywords matched
	}
	
	// Normalize score based on content length and matches
	avgScore := totalScore / float64(matchedCount)
	lengthFactor := float64(matchedCount) / float64(len(words))
	
	// Combine average score with match density
	finalScore := (avgScore + lengthFactor) / 2.0
	
	// Ensure it's in valid range and bias toward neutral
	if finalScore > 1.0 {
		finalScore = 1.0
	} else if finalScore < 0.0 {
		finalScore = 0.0
	}
	
	return finalScore
}

// analyzePatterns performs pattern-based relevancy scoring
func (rl *RelevancyLayer) analyzePatterns(content string, analysis *RelevancyAnalysis) float64 {
	totalScore := 0.0
	matchCount := 0
	
	for _, pattern := range rl.patterns {
		if pattern.Pattern.MatchString(content) {
			totalScore += pattern.Score
			matchCount++
			analysis.MatchedPatterns = append(analysis.MatchedPatterns, pattern.Description)
		}
	}
	
	if matchCount == 0 {
		return 0.5 // Neutral if no patterns matched
	}
	
	// Average the pattern scores and normalize
	avgScore := totalScore / float64(matchCount)
	
	// Convert to 0-1 scale (patterns can have negative scores)
	normalizedScore := (avgScore + 1.0) / 2.0
	
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	} else if normalizedScore < 0.0 {
		normalizedScore = 0.0
	}
	
	return normalizedScore
}

// analyzeContext performs context-based relevancy adjustments
func (rl *RelevancyLayer) analyzeContext(content string, context moderation.ModerationContext, analysis *RelevancyAnalysis) float64 {
	baseScore := 0.5
	
	// Content length analysis
	wordCount := len(strings.Fields(content))
	
	if wordCount < 3 {
		// Very short content is likely not very relevant
		baseScore -= 0.2
	} else if wordCount > 50 {
		// Very long content might be more relevant (detailed questions)
		baseScore += 0.1
	}
	
	// Time-based factors (optional)
	hour := context.Timestamp.Hour()
	if hour >= 9 && hour <= 17 {
		// Business hours - might be more work-related
		baseScore += 0.05
	}
	
	// User type considerations
	if context.UserType == "admin" || context.UserType == "moderator" {
		// Trust admin users more
		baseScore += 0.1
	}
	
	// Ensure valid range
	if baseScore > 1.0 {
		baseScore = 1.0
	} else if baseScore < 0.0 {
		baseScore = 0.0
	}
	
	return baseScore
}

// initializeDefaultKeywords sets up default relevancy keywords
func (rl *RelevancyLayer) initializeDefaultKeywords() {
	// Highly relevant AI/tech keywords (positive scores)
	relevant := map[string]float64{
		"ai":           0.9,
		"artificial":   0.8,
		"intelligence": 0.8,
		"machine":      0.7,
		"learning":     0.7,
		"model":        0.8,
		"neural":       0.8,
		"network":      0.7,
		"algorithm":    0.8,
		"data":         0.7,
		"api":          0.8,
		"code":         0.7,
		"programming":  0.8,
		"software":     0.7,
		"technology":   0.7,
		"computer":     0.7,
		"system":       0.6,
		"platform":     0.6,
		"application":  0.6,
		"development":  0.7,
		"framework":    0.7,
		"language":     0.6,
		"python":       0.8,
		"javascript":   0.8,
		"golang":       0.8,
		"database":     0.7,
		"server":       0.7,
		"client":       0.6,
		"web":          0.6,
		"mobile":       0.6,
		"cloud":        0.7,
		"security":     0.7,
		"automation":   0.7,
		"optimization": 0.7,
	}
	
	// Irrelevant topics (negative scores)
	irrelevant := map[string]float64{
		"weather":     0.2,
		"cooking":     0.2,
		"recipe":      0.2,
		"food":        0.3,
		"sports":      0.2,
		"football":    0.1,
		"basketball":  0.1,
		"movie":       0.2,
		"film":        0.2,
		"celebrity":   0.1,
		"gossip":      0.1,
		"fashion":     0.2,
		"music":       0.3, // Could be tech-related
		"game":        0.4, // Could be tech-related
		"travel":      0.2,
		"vacation":    0.1,
		"politics":    0.2,
		"religion":    0.2,
		"personal":    0.3,
		"family":      0.2,
		"relationship": 0.1,
	}
	
	// Combine all keywords
	for word, score := range relevant {
		rl.domainKeywords[word] = score
	}
	
	for word, score := range irrelevant {
		rl.domainKeywords[word] = score
	}
}

// initializePatterns sets up relevancy patterns
func (rl *RelevancyLayer) initializePatterns() {
	patterns := []struct {
		pattern     string
		score       float64
		description string
		category    string
	}{
		// Highly relevant patterns (positive scores)
		{`\b(how\s+to|help\s+me|explain|implement|create|build)\b.*\b(ai|api|code|program|system)\b`, 0.8, "Technical how-to question", "tech_question"},
		{`\b(debug|error|exception|bug|issue)\b`, 0.7, "Technical problem", "tech_problem"},
		{`\b(install|setup|configure|deploy)\b`, 0.7, "Technical setup", "tech_setup"},
		{`\b(best\s+practice|recommend|suggest)\b.*\b(library|framework|tool|language)\b`, 0.8, "Technical recommendation", "tech_advice"},
		{`\b(database|sql|query|table|schema)\b`, 0.8, "Database question", "database"},
		{`\b(frontend|backend|fullstack|api|endpoint)\b`, 0.8, "Web development", "webdev"},
		
		// Irrelevant patterns (negative scores)
		{`\b(what.*weather|weather.*today|temperature.*outside)\b`, -0.8, "Weather inquiry", "weather"},
		{`\b(recipe\s+for|how.*cook|bake.*cake)\b`, -0.7, "Cooking question", "cooking"},
		{`\b(movie.*recommend|watch.*film|cinema)\b`, -0.6, "Entertainment question", "entertainment"},
		{`\b(personal.*life|relationship.*advice|family.*problem)\b`, -0.7, "Personal question", "personal"},
		{`\b(sports.*score|game.*result|who.*won)\b`, -0.7, "Sports question", "sports"},
		{`\b(tell.*joke|funny.*story|make.*laugh)\b`, -0.5, "Entertainment request", "humor"},
		
		// Question patterns (generally relevant)
		{`^(what|how|why|when|where|can|could|would|should)\b`, 0.6, "Question format", "question"},
		{`\?$`, 0.5, "Ends with question mark", "question"},
	}
	
	for _, p := range patterns {
		compiled, err := regexp.Compile(`(?i)` + p.pattern)
		if err != nil {
			log.Printf("Failed to compile relevancy pattern %s: %v", p.pattern, err)
			continue
		}
		
		rl.patterns = append(rl.patterns, &RelevancyPattern{
			Pattern:     compiled,
			Score:       p.score,
			Description: p.description,
			Category:    p.category,
		})
	}
}

// loadConfigKeywords loads custom keywords from configuration
func (rl *RelevancyLayer) loadConfigKeywords() {
	if rl.config.Options == nil {
		return
	}
	
	// Load custom relevant keywords
	if relevantWords, ok := rl.config.Options["relevant_keywords"].([]interface{}); ok {
		for _, word := range relevantWords {
			if str, ok := word.(string); ok {
				rl.domainKeywords[strings.ToLower(str)] = 0.8
			}
		}
	}
	
	// Load custom irrelevant keywords
	if irrelevantWords, ok := rl.config.Options["irrelevant_keywords"].([]interface{}); ok {
		for _, word := range irrelevantWords {
			if str, ok := word.(string); ok {
				rl.domainKeywords[strings.ToLower(str)] = 0.2
			}
		}
	}
	
	// Load custom keyword scores
	if customKeywords, ok := rl.config.Options["custom_keywords"].(map[string]interface{}); ok {
		for keyword, scoreInterface := range customKeywords {
			if score, ok := scoreInterface.(float64); ok {
				rl.domainKeywords[strings.ToLower(keyword)] = score
			}
		}
	}
}

// calculateConfidence calculates confidence in the relevancy score
func (rl *RelevancyLayer) calculateConfidence(analysis *RelevancyAnalysis) float64 {
	confidence := 0.5
	
	// Higher confidence with more keyword matches
	keywordFactor := float64(len(analysis.MatchedKeywords)) * 0.1
	if keywordFactor > 0.4 {
		keywordFactor = 0.4
	}
	confidence += keywordFactor
	
	// Higher confidence with pattern matches
	patternFactor := float64(len(analysis.MatchedPatterns)) * 0.15
	if patternFactor > 0.3 {
		patternFactor = 0.3
	}
	confidence += patternFactor
	
	// Higher confidence for extreme scores
	if analysis.Score > 0.8 || analysis.Score < 0.2 {
		confidence += 0.2
	}
	
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// buildReason creates a human-readable reason for the relevancy score
func (rl *RelevancyLayer) buildReason(analysis *RelevancyAnalysis) string {
	if analysis.Score > 0.7 {
		return fmt.Sprintf("Highly relevant content (keywords: %d, patterns: %d)", 
			len(analysis.MatchedKeywords), len(analysis.MatchedPatterns))
	} else if analysis.Score > 0.5 {
		return fmt.Sprintf("Relevant content (keywords: %d, patterns: %d)", 
			len(analysis.MatchedKeywords), len(analysis.MatchedPatterns))
	} else if analysis.Score > 0.3 {
		return fmt.Sprintf("Possibly irrelevant content (keywords: %d, patterns: %d)", 
			len(analysis.MatchedKeywords), len(analysis.MatchedPatterns))
	} else {
		return fmt.Sprintf("Likely irrelevant content (keywords: %d, patterns: %d)", 
			len(analysis.MatchedKeywords), len(analysis.MatchedPatterns))
	}
}

// buildResult constructs the final moderation result
func (rl *RelevancyLayer) buildResult(analysis *RelevancyAnalysis, processTime time.Duration) moderation.ModerationResult {
	// For relevancy, we want to block IRRELEVANT content
	// So we block when score is LOW (irrelevant)
	threshold := rl.config.Threshold
	if threshold == 0 {
		threshold = 0.3 // Default threshold - block if score below 0.3
	}
	
	blocked := analysis.Score < threshold
	
	// Determine category
	category := moderation.CategoryCustom
	if !analysis.Relevant {
		category = moderation.CategorySpam // Treat irrelevant as spam-like
	}
	
	return moderation.ModerationResult{
		Score:       analysis.Score,
		Confidence:  analysis.Confidence,
		Blocked:     blocked,
		Reason:      analysis.Reason,
		Category:    category,
		LayerName:   rl.name,
		ProcessTime: processTime,
		Details: map[string]interface{}{
			"relevant":         analysis.Relevant,
			"matched_keywords": analysis.MatchedKeywords,
			"matched_patterns": analysis.MatchedPatterns,
			"threshold":        threshold,
			"keyword_count":    len(analysis.MatchedKeywords),
			"pattern_count":    len(analysis.MatchedPatterns),
		},
	}
}

// Cache management methods

func (rl *RelevancyLayer) getCachedResult(content string) *RelevancyCacheEntry {
	rl.cacheMutex.RLock()
	defer rl.cacheMutex.RUnlock()
	
	key := rl.generateCacheKey(content)
	if entry, exists := rl.cache[key]; exists {
		// Check if cache entry is still valid (10 minutes)
		if time.Since(entry.Timestamp) < 10*time.Minute {
			entry.HitCount++
			return entry
		}
		// Remove expired entry
		delete(rl.cache, key)
	}
	
	return nil
}

func (rl *RelevancyLayer) cacheResult(content string, analysis *RelevancyAnalysis) {
	rl.cacheMutex.Lock()
	defer rl.cacheMutex.Unlock()
	
	key := rl.generateCacheKey(content)
	
	rl.cache[key] = &RelevancyCacheEntry{
		Score:     analysis.Score,
		Relevant:  analysis.Relevant,
		Keywords:  analysis.MatchedKeywords,
		Timestamp: time.Now(),
		HitCount:  0,
	}
	
	// Simple cache cleanup - remove old entries if cache gets too large
	if len(rl.cache) > 1000 {
		oldestKey := ""
		oldestTime := time.Now()
		
		for k, v := range rl.cache {
			if v.Timestamp.Before(oldestTime) {
				oldestTime = v.Timestamp
				oldestKey = k
			}
		}
		
		if oldestKey != "" {
			delete(rl.cache, oldestKey)
		}
	}
}

func (rl *RelevancyLayer) generateCacheKey(content string) string {
	// Simple cache key based on content length and first/last words
	words := strings.Fields(strings.ToLower(content))
	if len(words) == 0 {
		return "empty"
	}
	
	key := fmt.Sprintf("%d_%s", len(words), words[0])
	if len(words) > 1 {
		key += "_" + words[len(words)-1]
	}
	
	return key
}

func (rl *RelevancyLayer) buildResultFromCache(cached *RelevancyCacheEntry, processTime time.Duration) moderation.ModerationResult {
	threshold := rl.config.Threshold
	if threshold == 0 {
		threshold = 0.3
	}
	
	blocked := cached.Score < threshold
	
	return moderation.ModerationResult{
		Score:       cached.Score,
		Confidence:  0.8, // Slightly lower confidence for cached results
		Blocked:     blocked,
		Reason:      fmt.Sprintf("Cached relevancy result (score: %.2f)", cached.Score),
		Category:    moderation.CategoryCustom,
		LayerName:   rl.name,
		ProcessTime: processTime,
		Details: map[string]interface{}{
			"relevant":         cached.Relevant,
			"matched_keywords": cached.Keywords,
			"from_cache":       true,
			"cache_hits":       cached.HitCount,
			"threshold":        threshold,
		},
	}
}

// Stats and utility methods

func (rl *RelevancyLayer) updateStats(fn func(*RelevancyStats)) {
	rl.stats.mutex.Lock()
	defer rl.stats.mutex.Unlock()
	fn(rl.stats)
}

// GetStats returns current layer statistics
func (rl *RelevancyLayer) GetStats() map[string]interface{} {
	rl.stats.mutex.RLock()
	defer rl.stats.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_processed":   rl.stats.TotalProcessed,
		"relevant_count":    rl.stats.RelevantCount,
		"irrelevant_count":  rl.stats.IrrelevantCount,
		"cache_hits":        rl.stats.CacheHits,
		"average_score":     rl.stats.AverageScore,
		"average_time":      rl.stats.AverageTime.String(),
		"keyword_count":     len(rl.domainKeywords),
		"pattern_count":     len(rl.patterns),
		"cache_size":        len(rl.cache),
	}
}

// UpdateConfig updates the layer configuration
func (rl *RelevancyLayer) UpdateConfig(config moderation.LayerConfig) {
	rl.config = config
	rl.weight = config.Weight
	rl.enabled = config.Enabled
	
	// Reload custom keywords
	rl.loadConfigKeywords()
}

// AddKeyword adds a custom keyword with relevancy score
func (rl *RelevancyLayer) AddKeyword(keyword string, score float64) {
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}
	
	rl.domainKeywords[strings.ToLower(keyword)] = score
}

// RemoveKeyword removes a keyword from the relevancy list
func (rl *RelevancyLayer) RemoveKeyword(keyword string) {
	delete(rl.domainKeywords, strings.ToLower(keyword))
}

// GetKeywords returns all keywords and their scores
func (rl *RelevancyLayer) GetKeywords() map[string]float64 {
	result := make(map[string]float64)
	for k, v := range rl.domainKeywords {
		result[k] = v
	}
	return result
}

// SetAIProvider allows setting an external AI provider for enhanced relevancy checking
// This is the extension point for users who want to add their own AI models
func (rl *RelevancyLayer) SetAIProvider(provider RelevancyAIProvider) {
	// This would be implemented if users want to add their own AI
	// For now, it's just a placeholder to show the extension point
	log.Printf("AI provider integration not yet implemented - staying local for now")
}

// RelevancyAIProvider interface for custom AI integration
type RelevancyAIProvider interface {
	CheckRelevancy(content string, context moderation.ModerationContext) (score float64, confidence float64, err error)
	GetEmbedding(content string) ([]float64, error)
	CompareSimilarity(content1, content2 string) (float64, error)
} 