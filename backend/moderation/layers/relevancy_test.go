package layers

import (
	"testing"
	"time"

	"qt1-middleware/moderation"
)

func TestRelevancyLayer_NewRelevancyLayer(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)

	if layer.Name() != "relevancy" {
		t.Errorf("Expected name 'relevancy', got '%s'", layer.Name())
	}

	if layer.Weight() != 0.3 {
		t.Errorf("Expected weight 0.3, got %f", layer.Weight())
	}

	if !layer.Enabled() {
		t.Error("Expected layer to be enabled")
	}

	// Check that default keywords are loaded
	keywords := layer.GetKeywords()
	if len(keywords) == 0 {
		t.Error("Expected default keywords to be loaded")
	}

	// Check for some expected default keywords
	if _, exists := keywords["ai"]; !exists {
		t.Error("Expected 'ai' keyword to be in default keywords")
	}

	if _, exists := keywords["programming"]; !exists {
		t.Error("Expected 'programming' keyword to be in default keywords")
	}
}

func TestRelevancyLayer_ModerateRelevantContent(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Test relevant content
	relevantContent := "How do I implement a machine learning algorithm in Python?"
	result := layer.Moderate(relevantContent, context)

	if result.Blocked {
		t.Error("Relevant content should not be blocked")
	}

	if result.Score < 0.5 {
		t.Errorf("Expected high relevancy score for relevant content, got %f", result.Score)
	}

	if result.LayerName != "relevancy" {
		t.Errorf("Expected layer name 'relevancy', got '%s'", result.LayerName)
	}

	// Check details
	if details, ok := result.Details["relevant"].(bool); !ok || !details {
		t.Error("Expected content to be marked as relevant in details")
	}
}

func TestRelevancyLayer_ModerateIrrelevantContent(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Test irrelevant content
	irrelevantContent := "What's the weather like today? I want to go on vacation and cook some food."
	result := layer.Moderate(irrelevantContent, context)

	if !result.Blocked {
		t.Error("Irrelevant content should be blocked")
	}

	if result.Score > 0.5 {
		t.Errorf("Expected low relevancy score for irrelevant content, got %f", result.Score)
	}

	// Check details
	if details, ok := result.Details["relevant"].(bool); !ok || details {
		t.Error("Expected content to be marked as irrelevant in details")
	}
}

func TestRelevancyLayer_ModerateNeutralContent(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Test neutral content (no specific keywords)
	neutralContent := "Hello, can you help me with something?"
	result := layer.Moderate(neutralContent, context)

	// Neutral content should not be blocked (score should be around 0.5)
	if result.Blocked {
		t.Error("Neutral content should not be blocked")
	}

	if result.Score < 0.3 || result.Score > 0.7 {
		t.Errorf("Expected neutral score (0.3-0.7) for neutral content, got %f", result.Score)
	}
}

func TestRelevancyLayer_CustomKeywords(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options: map[string]interface{}{
			"relevant_keywords": []interface{}{"blockchain", "cryptocurrency"},
			"irrelevant_keywords": []interface{}{"music", "dance"},
			"custom_keywords": map[string]interface{}{
				"bitcoin": 0.9,
				"ethereum": 0.9,
				"party": 0.1,
			},
		},
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Test content with custom relevant keywords
	relevantContent := "I want to learn about blockchain and bitcoin technology."
	result := layer.Moderate(relevantContent, context)

	if result.Blocked {
		t.Error("Content with custom relevant keywords should not be blocked")
	}

	if result.Score < 0.6 {
		t.Errorf("Expected high score for custom relevant content, got %f", result.Score)
	}

	// Test content with custom irrelevant keywords
	irrelevantContent := "I love music and dancing at parties."
	result2 := layer.Moderate(irrelevantContent, context)

	if !result2.Blocked {
		t.Error("Content with custom irrelevant keywords should be blocked")
	}

	if result2.Score > 0.4 {
		t.Errorf("Expected low score for custom irrelevant content, got %f", result2.Score)
	}
}

func TestRelevancyLayer_DisabledLayer(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   false, // Disabled
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	result := layer.Moderate("Any content", context)

	if result.Blocked {
		t.Error("Disabled layer should not block content")
	}

	if result.Score != 0.0 {
		t.Errorf("Expected score 0.0 for disabled layer, got %f", result.Score)
	}

	if result.Reason != "Relevancy layer disabled" {
		t.Errorf("Expected disabled reason, got '%s'", result.Reason)
	}
}

func TestRelevancyLayer_Cache(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	content := "How do I build an AI application?"

	// First call should not be from cache
	result1 := layer.Moderate(content, context)
	if fromCache, ok := result1.Details["from_cache"].(bool); ok && fromCache {
		t.Error("First call should not be from cache")
	}

	// Second call should be from cache
	result2 := layer.Moderate(content, context)
	if fromCache, ok := result2.Details["from_cache"].(bool); !ok || !fromCache {
		t.Error("Second call should be from cache")
	}

	// Scores should be similar
	if abs(result1.Score-result2.Score) > 0.01 {
		t.Errorf("Cached result score should match original: %f vs %f", result1.Score, result2.Score)
	}
}

func TestRelevancyLayer_AddRemoveKeywords(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)

	// Add a custom keyword
	layer.AddKeyword("customtech", 0.8)
	
	keywords := layer.GetKeywords()
	if score, exists := keywords["customtech"]; !exists || score != 0.8 {
		t.Errorf("Expected custom keyword 'customtech' with score 0.8, got %f", score)
	}

	// Remove the keyword
	layer.RemoveKeyword("customtech")
	
	keywords = layer.GetKeywords()
	if _, exists := keywords["customtech"]; exists {
		t.Error("Expected custom keyword to be removed")
	}
}

func TestRelevancyLayer_Stats(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "relevancy",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.3,
		Options:   make(map[string]interface{}),
	}

	layer := NewRelevancyLayer(config)
	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Process some content
	layer.Moderate("How do I code in Python?", context)
	layer.Moderate("What's the weather today?", context)

	stats := layer.GetStats()

	if totalProcessed, ok := stats["total_processed"].(int64); !ok || totalProcessed != 2 {
		t.Errorf("Expected 2 total processed, got %v", stats["total_processed"])
	}

	if keywordCount, ok := stats["keyword_count"].(int); !ok || keywordCount == 0 {
		t.Errorf("Expected positive keyword count, got %v", stats["keyword_count"])
	}

	if patternCount, ok := stats["pattern_count"].(int); !ok || patternCount == 0 {
		t.Errorf("Expected positive pattern count, got %v", stats["pattern_count"])
	}
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
} 