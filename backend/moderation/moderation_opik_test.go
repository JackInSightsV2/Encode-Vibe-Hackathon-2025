package moderation

import (
	"context"
	"qt1-middleware/config"
	"qt1-middleware/opik"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

// TestRegexModeratorWithOpik tests regex moderation with Opik tracing
func TestRegexModeratorWithOpik(t *testing.T) {
	// Create test Opik client
	opikClient, err := opik.NewOpikClient(opik.OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-moderation",
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer opikClient.Close()

	// Create regex moderator
	config := LayerConfig{
		Name:      "regex",
		Enabled:   true,
		Weight:    0.3,
		Threshold: 0.8,
		Options: map[string]interface{}{
			"blocked_words": []interface{}{"badword", "harmful"},
		},
	}

	moderator, err := NewRegexModerator(config, opikClient)
	if err != nil {
		t.Fatalf("Failed to create regex moderator: %v", err)
	}

	ctx := context.Background()

	// Start trace
	trace, _ := opikClient.StartTrace(ctx, "test_moderation", opik.TraceOptions{})

	// Test cases
	tests := []struct {
		name          string
		content       string
		expectBlocked bool
		expectScore   float64
	}{
		{
			name:          "Clean content",
			content:       "This is a clean message",
			expectBlocked: false,
			expectScore:   0.0,
		},
		{
			name:          "Blocked word",
			content:       "This contains badword in it",
			expectBlocked: true,
			expectScore:   0.8,
		},
		{
			name:          "Harmful content",
			content:       "This is harmful content",
			expectBlocked: true,
			expectScore:   0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := opikClient.StartSpan(trace, opik.SpanTypeRegex, map[string]interface{}{
				"content": tt.content,
			})

			result, err := moderator.ModerateWithTracing(ctx, tt.content, span)
			if err != nil {
				t.Errorf("ModerateWithTracing() error = %v", err)
				return
			}

			if result.Blocked != tt.expectBlocked {
				t.Errorf("Expected blocked = %v, got %v", tt.expectBlocked, result.Blocked)
			}

			if result.Confidence < tt.expectScore-0.1 || result.Confidence > tt.expectScore+0.1 {
				t.Errorf("Expected score ~%v, got %v", tt.expectScore, result.Confidence)
			}
		})
	}

	// End trace
	opikClient.EndTrace(trace, map[string]interface{}{
		"test_completed": true,
	})
}

// TestLLMModeratorWithOpik tests LLM moderation with Opik tracing
func TestLLMModeratorWithOpik(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LLM test in short mode")
	}

	// Create test Opik client
	opikClient, err := opik.NewOpikClient(opik.OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-llm-moderation",
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer opikClient.Close()

	// Mock OpenAI client (you'd use a real one in integration tests)
	openaiClient := openai.NewClient("test-key")

	// Create LLM moderator
	config := LayerConfig{
		Name:      "llm",
		Enabled:   true,
		Weight:    0.4,
		Threshold: 0.7,
		Options: map[string]interface{}{
			"model":     "gpt-3.5-turbo",
			"provider":  "openai",
			"threshold": 0.7,
		},
	}

	moderator, err := NewLLMModerator(config, opikClient, openaiClient)
	if err != nil {
		t.Fatalf("Failed to create LLM moderator: %v", err)
	}

	ctx := context.Background()

	// Start trace
	trace, _ := opikClient.StartTrace(ctx, "test_llm_moderation", opik.TraceOptions{})

	// Create span
	span := opikClient.StartSpan(trace, opik.SpanTypeLLM, map[string]interface{}{
		"content": "Test content for LLM moderation",
	})

	// Note: This would require actual OpenAI API key for real testing
	// For unit tests, you'd mock the OpenAI client
	t.Log("LLM moderator created successfully")

	// Verify moderator properties
	if moderator.Name() != "llm" {
		t.Errorf("Expected name 'llm', got %s", moderator.Name())
	}

	if moderator.Weight() != 0.4 {
		t.Errorf("Expected weight 0.4, got %f", moderator.Weight())
	}

	span.End()
	opikClient.EndTrace(trace, nil)
}

// TestPIIDetectorWithOpik tests PII detection with Opik tracing
func TestPIIDetectorWithOpik(t *testing.T) {
	// Create test Opik client
	opikClient, err := opik.NewOpikClient(opik.OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-pii-detection",
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer opikClient.Close()

	// Create PII detector
	config := LayerConfig{
		Name:      "pii",
		Enabled:   true,
		Weight:    0.2,
		Threshold: 0.6,
		Options: map[string]interface{}{
			"masking_enabled":       true,
			"detect_email":          true,
			"detect_phone":          true,
			"detect_ssn":            true,
			"detect_credit_card":    true,
			"email_confidence":      0.95,
			"phone_confidence":      0.90,
			"ssn_confidence":        0.98,
			"credit_card_confidence": 0.95,
		},
	}

	detector, err := NewPIIDetector(config, opikClient)
	if err != nil {
		t.Fatalf("Failed to create PII detector: %v", err)
	}

	ctx := context.Background()

	// Start trace
	trace, _ := opikClient.StartTrace(ctx, "test_pii_detection", opik.TraceOptions{})

	// Test cases
	tests := []struct {
		name         string
		content      string
		expectPII    bool
		expectTypes  []string
		expectMasked string
	}{
		{
			name:         "No PII",
			content:      "This is a clean message with no personal information",
			expectPII:    false,
			expectTypes:  []string{},
			expectMasked: "",
		},
		{
			name:         "Email detection",
			content:      "Contact me at john.doe@example.com for more info",
			expectPII:    true,
			expectTypes:  []string{"email"},
			expectMasked: "Contact me at ***@*** for more info",
		},
		{
			name:         "Phone detection",
			content:      "Call me at 555-123-4567 anytime",
			expectPII:    true,
			expectTypes:  []string{"phone"},
			expectMasked: "Call me at ***-***-**** anytime",
		},
		{
			name:         "Multiple PII",
			content:      "Email: test@example.com, Phone: 555-987-6543",
			expectPII:    true,
			expectTypes:  []string{"email", "phone"},
			expectMasked: "Email: ***@***, Phone: ***-***-****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := opikClient.StartSpan(trace, opik.SpanTypePII, map[string]interface{}{
				"content": tt.content,
			})

			result, err := detector.DetectWithTracing(ctx, tt.content, span)
			if err != nil {
				t.Errorf("DetectWithTracing() error = %v", err)
				return
			}

			if result.PIIFound != tt.expectPII {
				t.Errorf("Expected PII found = %v, got %v", tt.expectPII, result.PIIFound)
			}

			if tt.expectPII && tt.expectMasked != "" && result.MaskedText != tt.expectMasked {
				t.Errorf("Expected masked text = %v, got %v", tt.expectMasked, result.MaskedText)
			}

			// Check PII types
			if len(result.PIITypes) != len(tt.expectTypes) {
				t.Errorf("Expected %d PII types, got %d", len(tt.expectTypes), len(result.PIITypes))
			}
		})
	}

	// End trace
	opikClient.EndTrace(trace, map[string]interface{}{
		"test_completed": true,
	})
}

// TestEngineWithOpik tests the full moderation engine with Opik tracing
func TestEngineWithOpik(t *testing.T) {
	// Create test Opik client
	opikClient, err := opik.NewOpikClient(opik.OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-engine",
		BatchSize:   10,
		FlushInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Opik client: %v", err)
	}
	defer opikClient.Close()

	// Create test configuration
	cfg := config.AdvancedModerationConfig{
		Enabled: true,
		Layers: []config.AdvancedLayerConfig{
			{
				Name:      "regex",
				Enabled:   true,
				Weight:    0.3,
				Threshold: 0.8,
				Options: map[string]interface{}{
					"blocked_words": []interface{}{"spam", "scam"},
				},
			},
			{
				Name:      "pii",
				Enabled:   true,
				Weight:    0.2,
				Threshold: 0.6,
				Options: map[string]interface{}{
					"masking_enabled": true,
					"detect_email":    true,
					"detect_phone":    true,
				},
			},
		},
		Thresholds: config.ThresholdConfig{
			Low:      0.3,
			Medium:   0.6,
			High:     0.8,
			Critical: 0.95,
		},
		Actions: config.ActionConfig{
			Low:      "log",
			Medium:   "flag",
			High:     "block",
			Critical: "block_and_alert",
		},
		Cache: config.CacheConfig{
			Enabled:    true,
			TTLMinutes: 5,
			MaxEntries: 100,
		},
	}

	// Create engine
	engine, err := NewEngineWithOpik(cfg, opikClient, nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()

	// Test moderation
	tests := []struct {
		name           string
		content        string
		expectBlocked  bool
		expectSeverity string
	}{
		{
			name:           "Clean content",
			content:        "This is a perfectly clean message",
			expectBlocked:  false,
			expectSeverity: "low",
		},
		{
			name:           "Spam content",
			content:        "This is a spam message",
			expectBlocked:  true,
			expectSeverity: "high",
		},
		{
			name:           "PII content",
			content:        "Contact john@example.com for details",
			expectBlocked:  false, // PII alone might not block
			expectSeverity: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			moderationCtx := ModerationContext{
				UserID:    "test-user",
				SessionID: "test-session",
				RequestID: "test-request-" + tt.name,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"endpoint": "/test",
				},
			}

			result, err := engine.ModerateWithOpik(ctx, tt.content, moderationCtx)
			if err != nil {
				t.Errorf("ModerateWithOpik() error = %v", err)
				return
			}

			if result.FinalDecision != tt.expectBlocked {
				t.Errorf("Expected blocked = %v, got %v", tt.expectBlocked, result.FinalDecision)
			}

			if result.Severity != tt.expectSeverity {
				t.Errorf("Expected severity = %v, got %v", tt.expectSeverity, result.Severity)
			}

			// Test cache hit
			cachedResult, err := engine.ModerateWithOpik(ctx, tt.content, moderationCtx)
			if err != nil {
				t.Errorf("Cache lookup error = %v", err)
				return
			}

			if !cachedResult.CacheHit {
				t.Error("Expected cache hit on second call")
			}
		})
	}

	// Give time for batching
	time.Sleep(100 * time.Millisecond)

	// Check statistics
	stats := engine.GetStats()
	if stats.TotalRequests < int64(len(tests)) {
		t.Errorf("Expected at least %d requests, got %d", len(tests), stats.TotalRequests)
	}
}