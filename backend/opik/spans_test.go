package opik

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTrace_StartSpan(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})

	tests := []struct {
		name     string
		spanName string
		options  SpanOptions
		wantErr  bool
	}{
		{
			name:     "Basic span",
			spanName: "test_span",
			options: SpanOptions{
				Input:    "test input",
				SpanKind: SpanKindInternal,
			},
			wantErr: false,
		},
		{
			name:     "Span with metadata",
			spanName: "metadata_span",
			options: SpanOptions{
				Input: map[string]interface{}{
					"key": "value",
				},
				Metadata: map[string]interface{}{
					"user": "test",
				},
				SpanKind: SpanKindClient,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := trace.StartSpan(tt.spanName, tt.options)
			
			if span == nil {
				t.Error("StartSpan() should return a span")
				return
			}
			
			if span.Name != tt.spanName {
				t.Errorf("Span name = %v, want %v", span.Name, tt.spanName)
			}
			
			if span.TraceID != trace.ID {
				t.Error("Span should reference parent trace")
			}
			
			if span.Status != "in_progress" {
				t.Error("Span should be in progress")
			}
			
			// End the span
			span.End()
			
			if span.Status != "completed" {
				t.Error("Span should be completed after End()")
			}
			
			if span.Duration == 0 {
				t.Error("Span should have duration after End()")
			}
		})
	}
}

func TestSpan_SetOutput(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})
	span := trace.StartSpan("test_span", SpanOptions{})

	output := map[string]interface{}{
		"result": "success",
		"count":  42,
	}
	
	span.SetOutput(output)
	
	if span.Output == nil {
		t.Error("Output should be set")
	}
}

func TestSpan_SetMetadata(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})
	span := trace.StartSpan("test_span", SpanOptions{})

	metadata := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	
	span.SetMetadata(metadata)
	
	if span.Metadata["key1"] != "value1" {
		t.Error("Metadata should be set")
	}
	
	// Add more metadata
	span.SetMetadata(map[string]interface{}{
		"key3": "value3",
	})
	
	if len(span.Metadata) != 3 {
		t.Error("Metadata should be merged")
	}
}

func TestSpan_SetScore(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})
	span := trace.StartSpan("test_span", SpanOptions{})

	span.SetScore("accuracy", 0.95)
	span.SetScore("performance", 0.87)
	
	if span.Scores["accuracy"] != 0.95 {
		t.Error("Score should be set")
	}
	
	if len(span.Scores) != 2 {
		t.Error("Should have multiple scores")
	}
}

func TestSpan_SetError(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})
	span := trace.StartSpan("test_span", SpanOptions{})

	testErr := errors.New("test error")
	span.SetError(testErr)
	
	if span.Status != "error" {
		t.Error("Span status should be error")
	}
	
	if span.Error != "test error" {
		t.Error("Error message should be set")
	}
}

func TestSpan_AddTag(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})
	span := trace.StartSpan("test_span", SpanOptions{})

	span.AddTag("environment", "test")
	span.AddTag("version", "1.0.0")
	
	tags, ok := span.Metadata["tags"].(map[string]interface{})
	if !ok {
		t.Fatal("Tags should be in metadata")
	}
	
	if tags["environment"] != "test" {
		t.Error("Tag should be set")
	}
	
	if len(tags) != 2 {
		t.Error("Should have multiple tags")
	}
}

func TestSpan_GetDuration(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})
	span := trace.StartSpan("test_span", SpanOptions{})

	// Duration before end
	duration1 := span.GetDuration()
	if duration1 == 0 {
		t.Error("Should calculate duration from start time")
	}
	
	// Sleep a bit
	time.Sleep(10 * time.Millisecond)
	
	// End the span
	span.End()
	
	// Duration after end
	duration2 := span.GetDuration()
	if duration2 == 0 {
		t.Error("Should have duration after end")
	}
	
	// Duration should be consistent after end
	time.Sleep(10 * time.Millisecond)
	duration3 := span.GetDuration()
	
	if duration2 != duration3 {
		t.Error("Duration should be fixed after end")
	}
}

func TestOpikClient_StartSpan(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})

	// Test different span types
	spanTypes := []SpanType{
		SpanTypeRegex,
		SpanTypeLLM,
		SpanTypePII,
		SpanTypeSecurity,
		SpanTypeProvider,
		SpanTypeRateLimit,
		SpanTypeAuth,
		SpanTypeCache,
	}

	for _, st := range spanTypes {
		t.Run(string(st), func(t *testing.T) {
			span := client.StartSpan(trace, st, map[string]interface{}{
				"test": "input",
			})
			
			if span == nil {
				t.Error("StartSpan should return a span")
				return
			}
			
			if span.Name != string(st) {
				t.Errorf("Span name = %v, want %v", span.Name, string(st))
			}
			
			metadata, ok := span.Metadata["span_type"].(string)
			if !ok || metadata != string(st) {
				t.Error("Span type should be in metadata")
			}
		})
	}
}

func TestTrace_GetSpan(t *testing.T) {
	client, _ := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	defer client.Close()

	ctx := context.Background()
	trace, _ := client.StartTrace(ctx, "test_trace", TraceOptions{})

	// Create multiple spans
	_ = trace.StartSpan("span1", SpanOptions{})
	span2 := trace.StartSpan("span2", SpanOptions{})
	_ = trace.StartSpan("span3", SpanOptions{})
	
	// Get existing span
	found := trace.GetSpan("span2")
	if found != span2 {
		t.Error("Should find the correct span")
	}
	
	// Get non-existing span
	notFound := trace.GetSpan("nonexistent")
	if notFound != nil {
		t.Error("Should return nil for non-existing span")
	}
	
	// Verify all spans are tracked
	if len(trace.Spans) != 3 {
		t.Error("Should track all spans")
	}
}