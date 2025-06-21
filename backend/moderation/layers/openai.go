package layers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"qt1-middleware/moderation"
)

// OpenAILayer implements AI-powered content moderation using OpenAI's API
type OpenAILayer struct {
	name        string
	weight      float64
	enabled     bool
	config      moderation.LayerConfig
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	fallback    moderation.ModerationLayer
	batchCache  map[string]*BatchRequest
	batchMutex  sync.RWMutex
	stats       *OpenAILayerStats
	rateLimiter *RateLimiter
}

// OpenAILayerStats tracks performance and usage metrics
type OpenAILayerStats struct {
	TotalRequests     int64
	SuccessfulCalls   int64
	FailedCalls       int64
	FallbackUsed      int64
	BatchedRequests   int64
	AverageLatency    time.Duration
	TotalTokensUsed   int64
	APIErrors         map[string]int64
	mutex            sync.RWMutex
}

// BatchRequest represents a batched moderation request
type BatchRequest struct {
	Content   string
	Context   moderation.ModerationContext
	Timestamp time.Time
	Result    chan *moderation.ModerationResult
}

// RateLimiter implements simple rate limiting for API calls
type RateLimiter struct {
	tokens    int
	maxTokens int
	refillRate time.Duration
	lastRefill time.Time
	mutex     sync.Mutex
}

// OpenAI API structures
type OpenAIModerationRequest struct {
	Input string `json:"input"`
	Model string `json:"model,omitempty"`
}

type OpenAIModerationResponse struct {
	ID      string                   `json:"id"`
	Model   string                   `json:"model"`
	Results []OpenAIModerationResult `json:"results"`
}

type OpenAIModerationResult struct {
	Flagged        bool                              `json:"flagged"`
	Categories     OpenAIModerationCategories       `json:"categories"`
	CategoryScores OpenAIModerationCategoryScores   `json:"category_scores"`
}

type OpenAIModerationCategories struct {
	Sexual                bool `json:"sexual"`
	Hate                  bool `json:"hate"`
	Harassment            bool `json:"harassment"`
	SelfHarm              bool `json:"self-harm"`
	SexualMinors          bool `json:"sexual/minors"`
	HateThreatening       bool `json:"hate/threatening"`
	ViolenceGraphic       bool `json:"violence/graphic"`
	SelfHarmIntent        bool `json:"self-harm/intent"`
	SelfHarmInstructions  bool `json:"self-harm/instructions"`
	HarassmentThreatening bool `json:"harassment/threatening"`
	Violence              bool `json:"violence"`
}

type OpenAIModerationCategoryScores struct {
	Sexual                float64 `json:"sexual"`
	Hate                  float64 `json:"hate"`
	Harassment            float64 `json:"harassment"`
	SelfHarm              float64 `json:"self-harm"`
	SexualMinors          float64 `json:"sexual/minors"`
	HateThreatening       float64 `json:"hate/threatening"`
	ViolenceGraphic       float64 `json:"violence/graphic"`
	SelfHarmIntent        float64 `json:"self-harm/intent"`
	SelfHarmInstructions  float64 `json:"self-harm/instructions"`
	HarassmentThreatening float64 `json:"harassment/threatening"`
	Violence              float64 `json:"violence"`
}

// NewOpenAILayer creates a new OpenAI-powered moderation layer
func NewOpenAILayer(config moderation.LayerConfig, fallback moderation.ModerationLayer) *OpenAILayer {
	layer := &OpenAILayer{
		name:       config.Name,
		weight:     config.Weight,
		enabled:    config.Enabled,
		config:     config,
		baseURL:    "https://api.openai.com/v1",
		batchCache: make(map[string]*BatchRequest),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallback: fallback,
		stats: &OpenAILayerStats{
			APIErrors: make(map[string]int64),
		},
		rateLimiter: &RateLimiter{
			tokens:     100,
			maxTokens:  100,
			refillRate: time.Minute,
			lastRefill: time.Now(),
		},
	}

	// Extract API key from config
	if config.Options != nil {
		if apiKey, ok := config.Options["api_key"].(string); ok {
			layer.apiKey = apiKey
		}
		if baseURL, ok := config.Options["base_url"].(string); ok {
			layer.baseURL = baseURL
		}
		if timeout, ok := config.Options["timeout"].(int); ok {
			layer.httpClient.Timeout = time.Duration(timeout) * time.Second
		}
	}

	// Start batch processor
	go layer.processBatchRequests()

	return layer
}

// Name returns the layer name
func (ol *OpenAILayer) Name() string {
	return ol.name
}

// Weight returns the layer weight
func (ol *OpenAILayer) Weight() float64 {
	return ol.weight
}

// Enabled returns whether the layer is enabled
func (ol *OpenAILayer) Enabled() bool {
	return ol.enabled && ol.apiKey != ""
}

// Config returns the layer configuration
func (ol *OpenAILayer) Config() moderation.LayerConfig {
	return ol.config
}

// Moderate performs AI-powered content moderation
func (ol *OpenAILayer) Moderate(content string, context moderation.ModerationContext) moderation.ModerationResult {
	startTime := time.Now()
	ol.updateStats(func(s *OpenAILayerStats) { s.TotalRequests++ })

	// Check if layer is properly configured
	if !ol.Enabled() {
		return ol.fallbackModeration(content, context, "OpenAI layer not enabled or missing API key")
	}

	// Check rate limiting
	if !ol.rateLimiter.Allow() {
		return ol.fallbackModeration(content, context, "Rate limit exceeded")
	}

	// For batch optimization, check if we should batch this request
	if ol.shouldBatch(content, context) {
		return ol.batchModeration(content, context)
	}

	// Perform immediate moderation
	return ol.moderateImmediate(content, context, startTime)
}

// moderateImmediate performs immediate OpenAI moderation
func (ol *OpenAILayer) moderateImmediate(content string, context moderation.ModerationContext, startTime time.Time) moderation.ModerationResult {
	// Call OpenAI Moderation API
	response, err := ol.callOpenAIModerationAPI(content)
	if err != nil {
		ol.updateStats(func(s *OpenAILayerStats) { 
			s.FailedCalls++
			if errType := ol.categorizeError(err); errType != "" {
				s.APIErrors[errType]++
			}
		})
		return ol.fallbackModeration(content, context, fmt.Sprintf("OpenAI API error: %v", err))
	}

	ol.updateStats(func(s *OpenAILayerStats) { 
		s.SuccessfulCalls++
		latency := time.Since(startTime)
		if s.SuccessfulCalls == 1 {
			s.AverageLatency = latency
		} else {
			s.AverageLatency = time.Duration((int64(s.AverageLatency)*(s.SuccessfulCalls-1) + int64(latency)) / s.SuccessfulCalls)
		}
	})

	// Convert OpenAI response to moderation result
	return ol.convertOpenAIResponse(response, content, context, time.Since(startTime))
}

// callOpenAIModerationAPI calls the OpenAI Moderation API
func (ol *OpenAILayer) callOpenAIModerationAPI(content string) (*OpenAIModerationResponse, error) {
	// Prepare request
	request := OpenAIModerationRequest{
		Input: content,
		Model: "text-moderation-latest",
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "POST", ol.baseURL+"/moderations", strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ol.apiKey)

	// Make the request
	resp, err := ol.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response OpenAIModerationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Results) == 0 {
		return nil, fmt.Errorf("no results in API response")
	}

	return &response, nil
}

// convertOpenAIResponse converts OpenAI API response to moderation result
func (ol *OpenAILayer) convertOpenAIResponse(response *OpenAIModerationResponse, content string, context moderation.ModerationContext, processTime time.Duration) moderation.ModerationResult {
	result := response.Results[0]
	
	// Calculate overall score based on category scores
	score := ol.calculateOverallScore(result.CategoryScores)
	
	// Determine primary category
	category := ol.determinePrimaryCategory(result.Categories, result.CategoryScores)
	
	// Build reason
	reason := ol.buildReason(result.Categories, result.CategoryScores)
	
	// Calculate confidence based on score distribution
	confidence := ol.calculateConfidence(result.CategoryScores, score)

	return moderation.ModerationResult{
		Score:       score,
		Confidence:  confidence,
		Blocked:     score >= ol.config.Threshold,
		Reason:      reason,
		Category:    category,
		LayerName:   ol.name,
		ProcessTime: processTime,
		Details: map[string]interface{}{
			"openai_flagged":     result.Flagged,
			"openai_categories":  result.Categories,
			"category_scores":    result.CategoryScores,
			"model":             response.Model,
			"api_id":            response.ID,
			"threshold":         ol.config.Threshold,
		},
	}
}

// calculateOverallScore calculates a weighted overall score from category scores
func (ol *OpenAILayer) calculateOverallScore(scores OpenAIModerationCategoryScores) float64 {
	// Define weights for different categories
	weights := map[string]float64{
		"violence":               1.0,
		"violence/graphic":       0.9,
		"hate":                   0.9,
		"hate/threatening":       1.0,
		"harassment":             0.8,
		"harassment/threatening": 0.9,
		"self-harm":              0.8,
		"self-harm/intent":       0.9,
		"self-harm/instructions": 0.8,
		"sexual":                 0.7,
		"sexual/minors":          1.0,
	}

	categoryScores := map[string]float64{
		"violence":               scores.Violence,
		"violence/graphic":       scores.ViolenceGraphic,
		"hate":                   scores.Hate,
		"hate/threatening":       scores.HateThreatening,
		"harassment":             scores.Harassment,
		"harassment/threatening": scores.HarassmentThreatening,
		"self-harm":              scores.SelfHarm,
		"self-harm/intent":       scores.SelfHarmIntent,
		"self-harm/instructions": scores.SelfHarmInstructions,
		"sexual":                 scores.Sexual,
		"sexual/minors":          scores.SexualMinors,
	}

	// Take the maximum weighted score instead of average to preserve high scores
	maxWeightedScore := 0.0

	for category, score := range categoryScores {
		if weight, exists := weights[category]; exists {
			weightedScore := score * weight
			if weightedScore > maxWeightedScore {
				maxWeightedScore = weightedScore
			}
		}
	}

	return maxWeightedScore
}

// determinePrimaryCategory determines the primary violation category
func (ol *OpenAILayer) determinePrimaryCategory(categories OpenAIModerationCategories, scores OpenAIModerationCategoryScores) string {
	// Find the category with the highest score
	maxScore := 0.0
	primaryCategory := moderation.CategoryCustom

	categoryMap := map[string]float64{
		moderation.CategoryViolence:    scores.Violence,
		moderation.CategoryHateSpeech: scores.Hate,
		moderation.CategoryCustom:     scores.Harassment,
		moderation.CategoryCustom + "_sexual": scores.Sexual,
	}

	for category, score := range categoryMap {
		if score > maxScore {
			maxScore = score
			primaryCategory = category
		}
	}

	// Special handling for high-severity categories
	if categories.Violence || categories.ViolenceGraphic {
		return moderation.CategoryViolence
	}
	if categories.Hate || categories.HateThreatening {
		return moderation.CategoryHateSpeech
	}
	if categories.SexualMinors {
		return moderation.CategoryCustom // High severity
	}

	return primaryCategory
}

// buildReason builds a human-readable reason for the moderation decision
func (ol *OpenAILayer) buildReason(categories OpenAIModerationCategories, scores OpenAIModerationCategoryScores) string {
	reasons := []string{}

	if categories.Violence {
		reasons = append(reasons, "Violence")
	}
	if categories.Hate {
		reasons = append(reasons, "Hate speech")
	}
	if categories.Harassment {
		reasons = append(reasons, "Harassment")
	}
	if categories.SelfHarm {
		reasons = append(reasons, "Self-harm")
	}
	if categories.Sexual {
		reasons = append(reasons, "Sexual content")
	}

	if len(reasons) == 0 {
		return "Content flagged by AI moderation"
	}

	return "OpenAI detected: " + strings.Join(reasons, ", ")
}

// calculateConfidence calculates confidence based on score distribution
func (ol *OpenAILayer) calculateConfidence(scores OpenAIModerationCategoryScores, overallScore float64) float64 {
	// Start with base confidence from overall score
	confidence := overallScore

	// Increase confidence if multiple categories are flagged (lower threshold for flagged)
	flaggedCount := 0
	if scores.Violence > 0.3 { flaggedCount++ }
	if scores.Hate > 0.3 { flaggedCount++ }
	if scores.Harassment > 0.3 { flaggedCount++ }
	if scores.SelfHarm > 0.3 { flaggedCount++ }
	if scores.Sexual > 0.3 { flaggedCount++ }
	if scores.ViolenceGraphic > 0.3 { flaggedCount++ }
	if scores.HateThreatening > 0.3 { flaggedCount++ }
	if scores.HarassmentThreatening > 0.3 { flaggedCount++ }

	// Higher confidence for multiple categories
	if flaggedCount > 1 {
		confidence += 0.15 * float64(flaggedCount-1)
	}

	// Very high confidence for severe categories
	if scores.Violence > 0.8 || scores.HateThreatening > 0.8 || scores.HarassmentThreatening > 0.8 {
		confidence += 0.2
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// Batch processing methods

// shouldBatch determines if a request should be batched
func (ol *OpenAILayer) shouldBatch(content string, context moderation.ModerationContext) bool {
	// Batch shorter content to optimize API usage
	return len(content) < 500 && !ol.isUrgent(context)
}

// isUrgent determines if a request needs immediate processing
func (ol *OpenAILayer) isUrgent(context moderation.ModerationContext) bool {
	// Consider requests urgent if from flagged users or containing certain keywords
	return strings.Contains(context.UserID, "urgent") || 
		   strings.Contains(context.SessionID, "priority")
}

// batchModeration handles batched moderation requests
func (ol *OpenAILayer) batchModeration(content string, context moderation.ModerationContext) moderation.ModerationResult {
	// Create batch request
	batchReq := &BatchRequest{
		Content:   content,
		Context:   context,
		Timestamp: time.Now(),
		Result:    make(chan *moderation.ModerationResult, 1),
	}

	// Add to batch queue
	ol.batchMutex.Lock()
	key := fmt.Sprintf("%s_%d", context.UserID, time.Now().Unix())
	ol.batchCache[key] = batchReq
	ol.batchMutex.Unlock()

	// Wait for result with timeout
	select {
	case result := <-batchReq.Result:
		ol.updateStats(func(s *OpenAILayerStats) { s.BatchedRequests++ })
		return *result
	case <-time.After(5 * time.Second):
		// Timeout - fall back to immediate processing
		return ol.moderateImmediate(content, context, time.Now())
	}
}

// processBatchRequests processes batched requests
func (ol *OpenAILayer) processBatchRequests() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ol.processPendingBatches()
	}
}

// processPendingBatches processes all pending batch requests
func (ol *OpenAILayer) processPendingBatches() {
	ol.batchMutex.Lock()
	batches := make([]*BatchRequest, 0, len(ol.batchCache))
	for key, batch := range ol.batchCache {
		batches = append(batches, batch)
		delete(ol.batchCache, key)
	}
	ol.batchMutex.Unlock()

	// Process batches in parallel
	for _, batch := range batches {
		go func(b *BatchRequest) {
			result := ol.moderateImmediate(b.Content, b.Context, b.Timestamp)
			select {
			case b.Result <- &result:
			default:
				// Channel might be closed
			}
		}(batch)
	}
}

// Error handling and fallback methods

// fallbackModeration uses fallback layer when OpenAI is unavailable
func (ol *OpenAILayer) fallbackModeration(content string, context moderation.ModerationContext, reason string) moderation.ModerationResult {
	ol.updateStats(func(s *OpenAILayerStats) { s.FallbackUsed++ })

	if ol.fallback != nil {
		result := ol.fallback.Moderate(content, context)
		// Add fallback information to details
		if result.Details == nil {
			result.Details = make(map[string]interface{})
		}
		result.Details["openai_fallback"] = true
		result.Details["fallback_reason"] = reason
		return result
	}

	// No fallback available - return safe result
	return moderation.ModerationResult{
		Score:       0.0,
		Confidence:  0.0,
		Blocked:     false,
		Reason:      "OpenAI unavailable, no fallback configured",
		Category:    moderation.CategoryCustom,
		LayerName:   ol.name,
		ProcessTime: 0,
		Details: map[string]interface{}{
			"openai_error":    reason,
			"fallback_used":   false,
		},
	}
}

// categorizeError categorizes API errors for statistics
func (ol *OpenAILayer) categorizeError(err error) string {
	errStr := err.Error()
	if strings.Contains(errStr, "timeout") {
		return "timeout"
	}
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		return "rate_limit"
	}
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		return "auth_error"
	}
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") {
		return "server_error"
	}
	return "other"
}

// Rate limiting methods

// Allow checks if a request should be allowed based on rate limiting
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	
	// Refill tokens if enough time has passed
	if now.Sub(rl.lastRefill) >= rl.refillRate {
		rl.tokens = rl.maxTokens
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Statistics methods

// updateStats safely updates layer statistics
func (ol *OpenAILayer) updateStats(update func(*OpenAILayerStats)) {
	ol.stats.mutex.Lock()
	defer ol.stats.mutex.Unlock()
	update(ol.stats)
}

// GetStats returns comprehensive statistics for this layer
func (ol *OpenAILayer) GetStats() map[string]interface{} {
	ol.stats.mutex.RLock()
	defer ol.stats.mutex.RUnlock()

	successRate := 0.0
	if ol.stats.TotalRequests > 0 {
		successRate = float64(ol.stats.SuccessfulCalls) / float64(ol.stats.TotalRequests)
	}

	fallbackRate := 0.0
	if ol.stats.TotalRequests > 0 {
		fallbackRate = float64(ol.stats.FallbackUsed) / float64(ol.stats.TotalRequests)
	}

	return map[string]interface{}{
		"name":                ol.name,
		"enabled":             ol.enabled,
		"weight":              ol.weight,
		"api_key_configured":  ol.apiKey != "",
		"total_requests":      ol.stats.TotalRequests,
		"successful_calls":    ol.stats.SuccessfulCalls,
		"failed_calls":        ol.stats.FailedCalls,
		"fallback_used":       ol.stats.FallbackUsed,
		"batched_requests":    ol.stats.BatchedRequests,
		"success_rate":        successRate,
		"fallback_rate":       fallbackRate,
		"average_latency":     ol.stats.AverageLatency.String(),
		"total_tokens_used":   ol.stats.TotalTokensUsed,
		"api_errors":          ol.stats.APIErrors,
		"rate_limiter_tokens": ol.rateLimiter.tokens,
	}
}