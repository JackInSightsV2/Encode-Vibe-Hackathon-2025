package layers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qt1-middleware/moderation"
)

// MockOpenAIServer creates a test HTTP server that mocks OpenAI API responses
func MockOpenAIServer(responses map[string]OpenAIModerationResponse) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}

		// Parse request
		var req OpenAIModerationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "bad request"}`))
			return
		}

		// Find matching response
		if response, exists := responses[req.Input]; exists {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			// Default safe response
			defaultResponse := OpenAIModerationResponse{
				ID:    "modr-test-" + req.Input[:min(8, len(req.Input))],
				Model: "text-moderation-latest",
				Results: []OpenAIModerationResult{
					{
						Flagged: false,
						Categories: OpenAIModerationCategories{
							Sexual:                false,
							Hate:                  false,
							Harassment:            false,
							SelfHarm:              false,
							SexualMinors:          false,
							HateThreatening:       false,
							ViolenceGraphic:       false,
							SelfHarmIntent:        false,
							SelfHarmInstructions:  false,
							HarassmentThreatening: false,
							Violence:              false,
						},
						CategoryScores: OpenAIModerationCategoryScores{
							Sexual:                0.0,
							Hate:                  0.0,
							Harassment:            0.0,
							SelfHarm:              0.0,
							SexualMinors:          0.0,
							HateThreatening:       0.0,
							ViolenceGraphic:       0.0,
							SelfHarmIntent:        0.0,
							SelfHarmInstructions:  0.0,
							HarassmentThreatening: 0.0,
							Violence:              0.0,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(defaultResponse)
		}
	}))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestOpenAILayerCreation(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "openai_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key": "test-key-123",
			"timeout": 10,
		},
	}

	layer := NewOpenAILayer(config, nil)

	if layer.Name() != "openai_test" {
		t.Errorf("Expected layer name 'openai_test', got %s", layer.Name())
	}

	if !layer.Enabled() {
		t.Error("Layer should be enabled when API key is provided")
	}

	if layer.Weight() != 1.0 {
		t.Errorf("Expected weight 1.0, got %f", layer.Weight())
	}
}

func TestOpenAILayerDisabledWithoutAPIKey(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "openai_disabled",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options:   map[string]interface{}{},
	}

	layer := NewOpenAILayer(config, nil)

	if layer.Enabled() {
		t.Error("Layer should be disabled when no API key is provided")
	}
}

func TestOpenAIModerationSafeContent(t *testing.T) {
	// Setup mock server
	responses := map[string]OpenAIModerationResponse{
		"Hello world, nice weather today!": {
			ID:    "modr-test-safe",
			Model: "text-moderation-latest",
			Results: []OpenAIModerationResult{
				{
					Flagged: false,
					Categories: OpenAIModerationCategories{
						Sexual:                false,
						Hate:                  false,
						Harassment:            false,
						SelfHarm:              false,
						SexualMinors:          false,
						HateThreatening:       false,
						ViolenceGraphic:       false,
						SelfHarmIntent:        false,
						SelfHarmInstructions:  false,
						HarassmentThreatening: false,
						Violence:              false,
					},
					CategoryScores: OpenAIModerationCategoryScores{
						Sexual:                0.001,
						Hate:                  0.002,
						Harassment:            0.001,
						SelfHarm:              0.000,
						SexualMinors:          0.000,
						HateThreatening:       0.000,
						ViolenceGraphic:       0.000,
						SelfHarmIntent:        0.000,
						SelfHarmInstructions:  0.000,
						HarassmentThreatening: 0.001,
						Violence:              0.001,
					},
				},
			},
		},
	}

	server := MockOpenAIServer(responses)
	defer server.Close()

	config := moderation.LayerConfig{
		Name:      "openai_safe_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key":  "test-key-123",
			"base_url": server.URL,
			"timeout":  5,
		},
	}

	layer := NewOpenAILayer(config, nil)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	result := layer.Moderate("Hello world, nice weather today!", context)

	if result.Blocked {
		t.Error("Safe content should not be blocked")
	}

	if result.Score >= 0.5 {
		t.Errorf("Expected low score for safe content, got %f", result.Score)
	}

	if result.LayerName != "openai_safe_test" {
		t.Errorf("Expected layer name 'openai_safe_test', got %s", result.LayerName)
	}

	// Check details
	if details := result.Details; details != nil {
		if flagged, ok := details["openai_flagged"].(bool); !ok || flagged {
			t.Error("Expected content not to be flagged by OpenAI")
		}
	}
}

func TestOpenAIModerationViolentContent(t *testing.T) {
	// Setup mock server with violent content response
	responses := map[string]OpenAIModerationResponse{
		"I will kill you and destroy everything": {
			ID:    "modr-test-violent",
			Model: "text-moderation-latest",
			Results: []OpenAIModerationResult{
				{
					Flagged: true,
					Categories: OpenAIModerationCategories{
						Sexual:                false,
						Hate:                  false,
						Harassment:            true,
						SelfHarm:              false,
						SexualMinors:          false,
						HateThreatening:       false,
						ViolenceGraphic:       false,
						SelfHarmIntent:        false,
						SelfHarmInstructions:  false,
						HarassmentThreatening: true,
						Violence:              true,
					},
					CategoryScores: OpenAIModerationCategoryScores{
						Sexual:                0.001,
						Hate:                  0.102,
						Harassment:            0.851,
						SelfHarm:              0.003,
						SexualMinors:          0.000,
						HateThreatening:       0.203,
						ViolenceGraphic:       0.245,
						SelfHarmIntent:        0.002,
						SelfHarmInstructions:  0.001,
						HarassmentThreatening: 0.923,
						Violence:              0.887,
					},
				},
			},
		},
	}

	server := MockOpenAIServer(responses)
	defer server.Close()

	config := moderation.LayerConfig{
		Name:      "openai_violent_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key":  "test-key-123",
			"base_url": server.URL,
		},
	}

	layer := NewOpenAILayer(config, nil)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	result := layer.Moderate("I will kill you and destroy everything", context)

	if !result.Blocked {
		t.Error("Violent content should be blocked")
	}

	if result.Score < 0.5 {
		t.Errorf("Expected high score for violent content, got %f", result.Score)
	}

	if result.Category != moderation.CategoryViolence {
		t.Errorf("Expected violence category, got %s", result.Category)
	}

	if !strings.Contains(result.Reason, "Violence") {
		t.Errorf("Expected reason to mention violence, got: %s", result.Reason)
	}

	// Check details
	if details := result.Details; details != nil {
		if flagged, ok := details["openai_flagged"].(bool); !ok || !flagged {
			t.Error("Expected content to be flagged by OpenAI")
		}
	}
}

func TestOpenAILayerFallback(t *testing.T) {
	// Create fallback layer
	fallbackConfig := moderation.LayerConfig{
		Name:      "fallback_regex",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
	}
	fallbackLayer := NewRegexLayer(fallbackConfig)

	// Setup OpenAI layer with invalid API endpoint to trigger fallback
	config := moderation.LayerConfig{
		Name:      "openai_fallback_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key":  "test-key-123",
			"base_url": "http://invalid-url-that-does-not-exist.com",
			"timeout":  1, // Short timeout to trigger failure quickly
		},
	}

	layer := NewOpenAILayer(config, fallbackLayer)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	result := layer.Moderate("I hate you with violence", context)

	// Should use fallback layer
	if details := result.Details; details != nil {
		if fallbackUsed, ok := details["openai_fallback"].(bool); !ok || !fallbackUsed {
			t.Error("Expected fallback to be used when OpenAI API fails")
		}
	}

	// Check that stats record fallback usage
	stats := layer.GetStats()
	if fallbackUsed, ok := stats["fallback_used"].(int64); !ok || fallbackUsed == 0 {
		t.Error("Expected fallback usage to be recorded in stats")
	}
}

func TestOpenAILayerRateLimiting(t *testing.T) {
	// Create fallback layer for rate limiting test
	fallbackConfig := moderation.LayerConfig{
		Name:      "fallback_regex",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
	}
	fallbackLayer := NewRegexLayer(fallbackConfig)

	// Create layer with very low rate limit for testing
	config := moderation.LayerConfig{
		Name:      "openai_rate_limit_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key": "test-key-123",
		},
	}

	layer := NewOpenAILayer(config, fallbackLayer)
	
	// Set very low rate limit - start with 0 tokens
	layer.rateLimiter.tokens = 0
	layer.rateLimiter.maxTokens = 1

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// This request should hit rate limit immediately and use fallback
	result := layer.Moderate("test content", context)

	// Check that fallback was used due to rate limiting
	stats := layer.GetStats()
	if fallbackUsed, ok := stats["fallback_used"].(int64); !ok || fallbackUsed == 0 {
		t.Error("Expected rate limiting to trigger fallback")
	}

	// Verify result has fallback details
	if details := result.Details; details != nil {
		if reason, ok := details["fallback_reason"].(string); !ok || !strings.Contains(reason, "Rate limit") {
			t.Errorf("Expected fallback reason to mention rate limit, got: %v", details["fallback_reason"])
		}
	} else {
		t.Error("Expected result to have details about fallback")
	}
}

func TestOpenAIBatchProcessing(t *testing.T) {
	// Setup mock server
	responses := map[string]OpenAIModerationResponse{
		"short content 1": {
			ID:    "modr-batch-1",
			Model: "text-moderation-latest",
			Results: []OpenAIModerationResult{
				{
					Flagged: false,
					Categories: OpenAIModerationCategories{},
					CategoryScores: OpenAIModerationCategoryScores{
						Violence: 0.1,
						Hate:     0.05,
					},
				},
			},
		},
		"short content 2": {
			ID:    "modr-batch-2",
			Model: "text-moderation-latest",
			Results: []OpenAIModerationResult{
				{
					Flagged: false,
					Categories: OpenAIModerationCategories{},
					CategoryScores: OpenAIModerationCategoryScores{
						Violence: 0.15,
						Hate:     0.08,
					},
				},
			},
		},
	}

	server := MockOpenAIServer(responses)
	defer server.Close()

	config := moderation.LayerConfig{
		Name:      "openai_batch_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key":  "test-key-123",
			"base_url": server.URL,
		},
	}

	layer := NewOpenAILayer(config, nil)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Process multiple short content items that should be batched
	result1 := layer.Moderate("short content 1", context)
	result2 := layer.Moderate("short content 2", context)

	// Give batch processor time to work
	time.Sleep(3 * time.Second)

	// Check that requests were processed
	if result1.Score < 0 || result2.Score < 0 {
		t.Error("Batch processing should return valid scores")
	}

	// Check stats for batch processing
	stats := layer.GetStats()
	if batchedRequests, ok := stats["batched_requests"].(int64); ok && batchedRequests > 0 {
		t.Logf("Successfully processed %d batched requests", batchedRequests)
	}
}

func TestOpenAIErrorHandling(t *testing.T) {
	// Setup server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	config := moderation.LayerConfig{
		Name:      "openai_error_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key":  "test-key-123",
			"base_url": server.URL,
			"timeout":  2,
		},
	}

	layer := NewOpenAILayer(config, nil)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	result := layer.Moderate("test content", context)

	// Should handle error gracefully
	if result.LayerName != "openai_error_test" {
		t.Errorf("Expected layer name to be preserved even on error, got %s", result.LayerName)
	}

	// Check error stats
	stats := layer.GetStats()
	if failedCalls, ok := stats["failed_calls"].(int64); !ok || failedCalls == 0 {
		t.Error("Expected failed calls to be recorded in stats")
	}

	if apiErrors, ok := stats["api_errors"].(map[string]int64); ok {
		totalErrors := int64(0)
		for _, count := range apiErrors {
			totalErrors += count
		}
		if totalErrors == 0 {
			t.Error("Expected API errors to be categorized and recorded")
		}
	}
}

func TestOpenAIConfidenceCalculation(t *testing.T) {
	// Setup mock server with varying confidence scenarios
	responses := map[string]OpenAIModerationResponse{
		"high confidence content": {
			ID:    "modr-high-confidence",
			Model: "text-moderation-latest",
			Results: []OpenAIModerationResult{
				{
					Flagged: true,
					Categories: OpenAIModerationCategories{
						Violence:              true,
						Hate:                  true,
						Harassment:            true,
					},
					CategoryScores: OpenAIModerationCategoryScores{
						Violence:   0.95,
						Hate:       0.88,
						Harassment: 0.92,
					},
				},
			},
		},
		"low confidence content": {
			ID:    "modr-low-confidence",
			Model: "text-moderation-latest",
			Results: []OpenAIModerationResult{
				{
					Flagged: false,
					Categories: OpenAIModerationCategories{},
					CategoryScores: OpenAIModerationCategoryScores{
						Violence: 0.05,
						Hate:     0.03,
					},
				},
			},
		},
	}

	server := MockOpenAIServer(responses)
	defer server.Close()

	config := moderation.LayerConfig{
		Name:      "openai_confidence_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key":  "test-key-123",
			"base_url": server.URL,
		},
	}

	layer := NewOpenAILayer(config, nil)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}

	// Test high confidence scenario
	highConfidenceResult := layer.Moderate("high confidence content", context)
	if highConfidenceResult.Confidence < 0.8 {
		t.Errorf("Expected high confidence (>0.8) for multiple category flags, got %f", highConfidenceResult.Confidence)
	}

	// Test low confidence scenario
	lowConfidenceResult := layer.Moderate("low confidence content", context)
	if lowConfidenceResult.Confidence > 0.3 {
		t.Errorf("Expected low confidence (<0.3) for low scores, got %f", lowConfidenceResult.Confidence)
	}
}

func TestOpenAILayerStats(t *testing.T) {
	config := moderation.LayerConfig{
		Name:      "openai_stats_test",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"api_key": "test-key-123",
		},
	}

	layer := NewOpenAILayer(config, nil)

	stats := layer.GetStats()

	// Check that all expected stat fields are present
	expectedFields := []string{
		"name", "enabled", "weight", "api_key_configured",
		"total_requests", "successful_calls", "failed_calls",
		"fallback_used", "batched_requests", "success_rate",
		"fallback_rate", "average_latency", "total_tokens_used",
		"api_errors", "rate_limiter_tokens",
	}

	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected stat field '%s' to exist", field)
		}
	}

	// Verify specific values
	if name, ok := stats["name"].(string); !ok || name != "openai_stats_test" {
		t.Errorf("Expected name 'openai_stats_test', got %v", stats["name"])
	}

	if apiKeyConfigured, ok := stats["api_key_configured"].(bool); !ok || !apiKeyConfigured {
		t.Error("Expected API key to be configured")
	}
}