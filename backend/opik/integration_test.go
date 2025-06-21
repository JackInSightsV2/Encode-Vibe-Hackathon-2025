package opik

import (
	"context"
	"testing"
	"time"
)

// MockModerationResult simulates a moderation result
type MockModerationResult struct {
	Blocked      bool
	Confidence   float64
	MatchedRules []string
}

// TestModerationTracing demonstrates how to use Opik for moderation tracing
func TestModerationTracing(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewOpikClient(OpikConfig{
		Enabled:       true,
		APIKey:        "test-key", // Use real key for actual integration
		ProjectName:   "qt1-safety-cockpit-test",
		BatchSize:     5,
		FlushInterval: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Simulate a moderation request
	request := ModerationRequest{
		ID:        "test-123",
		Provider:  "openai",
		Timestamp: time.Now(),
		Content:   "This is a test message for moderation",
		UserID:    "user-456",
		SessionID: "session-789",
		IPAddress: "192.168.1.100",
		Endpoint:  "/api/chat",
	}

	// Start moderation trace
	trace, err := client.TraceModeration(ctx, request)
	if err != nil {
		t.Fatalf("Failed to create moderation trace: %v", err)
	}

	// Simulate regex moderation layer
	regexSpan := client.StartSpan(trace, SpanTypeRegex, map[string]interface{}{
		"content":     request.Content,
		"rules_count": 10,
	})

	// Simulate regex processing
	time.Sleep(5 * time.Millisecond)
	
	regexResult := MockModerationResult{
		Blocked:      false,
		Confidence:   0.2,
		MatchedRules: []string{},
	}

	regexSpan.SetOutput(map[string]interface{}{
		"blocked":       regexResult.Blocked,
		"confidence":    regexResult.Confidence,
		"matched_rules": regexResult.MatchedRules,
	})
	
	regexSpan.SetMetadata(map[string]interface{}{
		"duration_ms": 5,
		"rule_count":  10,
	})
	
	regexSpan.SetScore("effectiveness", 0.8)
	regexSpan.End()

	// Simulate LLM moderation layer
	llmSpan := client.StartSpan(trace, SpanTypeLLM, map[string]interface{}{
		"content": request.Content,
		"model":   "gpt-3.5-turbo",
	})

	// Simulate LLM processing
	time.Sleep(50 * time.Millisecond)
	
	llmSpan.SetOutput(map[string]interface{}{
		"flagged": false,
		"categories": map[string]float64{
			"violence":     0.01,
			"hate":         0.02,
			"harassment":   0.01,
			"self-harm":    0.0,
			"sexual":       0.0,
			"dangerous":    0.03,
		},
	})
	
	llmSpan.SetMetadata(map[string]interface{}{
		"model":       "gpt-3.5-turbo",
		"provider":    "openai",
		"duration_ms": 50,
		"tokens":      25,
	})
	
	llmSpan.SetScore("accuracy", 0.95)
	llmSpan.End()

	// Simulate PII detection
	piiSpan := client.StartSpan(trace, SpanTypePII, map[string]interface{}{
		"content": request.Content,
	})

	time.Sleep(10 * time.Millisecond)
	
	piiSpan.SetOutput(map[string]interface{}{
		"pii_found": false,
		"pii_types": []string{},
		"pii_count": 0,
	})
	
	piiSpan.SetScore("detection_accuracy", 1.0)
	piiSpan.End()

	// End the trace with final result
	err = client.EndTrace(trace, map[string]interface{}{
		"allowed":        true,
		"total_duration": 65,
		"layers_checked": 3,
		"final_score":    0.95,
	})
	
	if err != nil {
		t.Errorf("Failed to end trace: %v", err)
	}

	// Give time for batching
	time.Sleep(100 * time.Millisecond)

	// Verify trace structure
	if len(trace.Spans) != 3 {
		t.Errorf("Expected 3 spans, got %d", len(trace.Spans))
	}

	if trace.Status != "completed" {
		t.Error("Trace should be completed")
	}
}

// TestProviderRequestTracing demonstrates provider request tracing
func TestProviderRequestTracing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "qt1-provider-test",
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Start provider request trace
	trace, err := client.StartTrace(ctx, "provider_request", TraceOptions{
		Input: map[string]interface{}{
			"provider": "openai",
			"model":    "gpt-3.5-turbo",
			"prompt":   "Test prompt",
		},
		Metadata: map[string]interface{}{
			"user_id":    "user-123",
			"session_id": "session-456",
		},
		Tags: []string{"provider", "openai", "chat"},
	})
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	// Rate limiting check
	rateLimitSpan := client.StartSpan(trace, SpanTypeRateLimit, map[string]interface{}{
		"user_id": "user-123",
		"ip":      "192.168.1.100",
	})
	
	rateLimitSpan.SetOutput(map[string]interface{}{
		"allowed":           true,
		"remaining_tokens":  95,
		"reset_in_seconds":  45,
	})
	rateLimitSpan.End()

	// Authentication
	authSpan := client.StartSpan(trace, SpanTypeAuth, map[string]interface{}{
		"method": "api_key",
	})
	
	authSpan.SetOutput(map[string]interface{}{
		"authenticated": true,
		"user_id":       "user-123",
		"scopes":        []string{"chat", "completions"},
	})
	authSpan.End()

	// Provider request
	providerSpan := client.StartSpan(trace, SpanTypeProvider, map[string]interface{}{
		"provider": "openai",
		"model":    "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Test prompt"},
		},
	})
	
	// Simulate provider delay
	time.Sleep(100 * time.Millisecond)
	
	providerSpan.SetOutput(map[string]interface{}{
		"status":         200,
		"tokens_used":    50,
		"response_time":  100,
		"finish_reason":  "stop",
	})
	
	providerSpan.SetMetadata(map[string]interface{}{
		"request_id":    "req-789",
		"cache_hit":     false,
		"stream":        false,
	})
	
	providerSpan.End()

	// End trace
	err = client.EndTrace(trace, map[string]interface{}{
		"success":       true,
		"total_tokens":  50,
		"response_time": 120,
	})
	
	if err != nil {
		t.Errorf("Failed to end trace: %v", err)
	}
}

// TestEvaluatorIntegration tests the evaluator system
func TestEvaluatorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "qt1-evaluator-test",
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer client.Close()

	// Register custom evaluators
	client.RegisterEvaluator(&MockRegexEvaluator{})
	client.RegisterEvaluator(&MockResponseTimeEvaluator{})

	ctx := context.Background()

	// Create trace with known good/bad content
	trace, _ := client.StartTrace(ctx, "evaluation_test", TraceOptions{
		Metadata: map[string]interface{}{
			"known_threat": false,
		},
	})

	// Add some spans
	span := client.StartSpan(trace, SpanTypeRegex, nil)
	span.SetOutput(map[string]interface{}{
		"blocked":    false,
		"confidence": 0.1,
	})
	span.End()

	// End trace - evaluators should run
	client.EndTrace(trace, nil)

	// Give time for async evaluation
	time.Sleep(100 * time.Millisecond)

	// Check if scores were added
	if len(trace.Scores) == 0 {
		t.Error("Evaluators should have added scores")
	}
}

// Mock evaluators for testing
type MockRegexEvaluator struct {
	BaseEvaluator
}

func (e *MockRegexEvaluator) Name() string {
	return "mock_regex_effectiveness"
}

func (e *MockRegexEvaluator) Evaluate(trace *Trace) (float64, map[string]interface{}) {
	// Find regex span
	regexSpan := trace.GetSpan(string(SpanTypeRegex))
	if regexSpan == nil {
		return 0, map[string]interface{}{"error": "no regex span"}
	}

	// Simple evaluation logic
	output, ok := regexSpan.Output.(map[string]interface{})
	if !ok {
		return 0, map[string]interface{}{"error": "invalid output"}
	}

	blocked, _ := output["blocked"].(bool)
	knownThreat, _ := trace.Metadata["known_threat"].(bool)

	score := 1.0
	if knownThreat && !blocked {
		score = 0.0 // False negative
	} else if !knownThreat && blocked {
		score = 0.5 // False positive
	}

	return score, map[string]interface{}{
		"blocked":      blocked,
		"known_threat": knownThreat,
		"correct":      score > 0.5,
	}
}

type MockResponseTimeEvaluator struct {
	BaseEvaluator
}

func (e *MockResponseTimeEvaluator) Name() string {
	return "mock_response_time"
}

func (e *MockResponseTimeEvaluator) Evaluate(trace *Trace) (float64, map[string]interface{}) {
	totalDuration := trace.Duration.Milliseconds()
	
	// Score based on response time (lower is better)
	score := 1.0
	if totalDuration > 1000 {
		score = 0.0
	} else if totalDuration > 500 {
		score = 0.5
	} else if totalDuration > 100 {
		score = 0.8
	}

	return score, map[string]interface{}{
		"duration_ms": totalDuration,
		"threshold":   100,
		"passed":      score > 0.5,
	}
}