package opik

import (
	"context"
	"testing"
	"time"
)

func TestNewOpikClient(t *testing.T) {
	tests := []struct {
		name    string
		config  OpikConfig
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: OpikConfig{
				Enabled:     true,
				APIKey:      "test-key",
				ProjectName: "test-project",
				BaseURL:     "https://api.opik.com",
				BatchSize:   100,
				FlushInterval: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Disabled client",
			config: OpikConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "Missing API key",
			config: OpikConfig{
				Enabled:     true,
				ProjectName: "test-project",
			},
			wantErr: true,
		},
		{
			name: "Default values applied",
			config: OpikConfig{
				Enabled:     true,
				APIKey:      "test-key",
				ProjectName: "test-project",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpikClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpikClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil && client != nil {
				defer client.Close()
				
				if tt.config.Enabled {
					if client.config.BaseURL == "" {
						t.Error("BaseURL should have default value")
					}
					if client.config.BatchSize == 0 {
						t.Error("BatchSize should have default value")
					}
					if client.config.FlushInterval == 0 {
						t.Error("FlushInterval should have default value")
					}
				}
			}
		})
	}
}

func TestOpikClient_StartTrace(t *testing.T) {
	client, err := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	tests := []struct {
		name    string
		options TraceOptions
		wantErr bool
	}{
		{
			name: "Basic trace",
			options: TraceOptions{
				Input: map[string]interface{}{
					"test": "data",
				},
			},
			wantErr: false,
		},
		{
			name: "Trace with metadata",
			options: TraceOptions{
				Input: map[string]interface{}{
					"request_id": "123",
				},
				Metadata: map[string]interface{}{
					"user_id": "user123",
				},
				Tags: []string{"test", "unit"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trace, err := client.StartTrace(ctx, "test_trace", tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartTrace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err == nil {
				if trace.ID == "" {
					t.Error("Trace should have ID")
				}
				if trace.Name != "test_trace" {
					t.Error("Trace name mismatch")
				}
				if trace.Status != "in_progress" {
					t.Error("Trace should be in progress")
				}
				
				// End the trace
				err = client.EndTrace(trace, map[string]interface{}{
					"result": "success",
				})
				if err != nil {
					t.Errorf("EndTrace() error = %v", err)
				}
			}
		})
	}
}

func TestOpikClient_DisabledClient(t *testing.T) {
	client, err := NewOpikClient(OpikConfig{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("Failed to create disabled client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	// Should return disabled trace without error
	trace, err := client.StartTrace(ctx, "test", TraceOptions{})
	if err != nil {
		t.Errorf("StartTrace() should not error when disabled: %v", err)
	}
	
	if trace.ID != "disabled" {
		t.Error("Disabled client should return disabled trace")
	}
	
	// EndTrace should also work without error
	err = client.EndTrace(trace, nil)
	if err != nil {
		t.Errorf("EndTrace() should not error when disabled: %v", err)
	}
}

func TestOpikClient_RegisterEvaluator(t *testing.T) {
	client, err := NewOpikClient(OpikConfig{
		Enabled:     true,
		APIKey:      "test-key",
		ProjectName: "test-project",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create mock evaluator
	evaluator := &MockEvaluator{name: "test_evaluator"}
	
	err = client.RegisterEvaluator(evaluator)
	if err != nil {
		t.Errorf("RegisterEvaluator() error = %v", err)
	}
	
	if len(client.evaluators) != 1 {
		t.Error("Evaluator should be registered")
	}
}

func TestOpikClient_Batching(t *testing.T) {
	client, err := NewOpikClient(OpikConfig{
		Enabled:       true,
		APIKey:        "test-key",
		ProjectName:   "test-project",
		BatchSize:     2,
		FlushInterval: 1 * time.Hour, // Long interval to test manual batching
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	// Create first trace
	trace1, _ := client.StartTrace(ctx, "trace1", TraceOptions{})
	client.EndTrace(trace1, nil)
	
	if len(client.traceBatch) != 1 {
		t.Error("First trace should be in batch")
	}
	
	// Create second trace (should trigger flush)
	trace2, _ := client.StartTrace(ctx, "trace2", TraceOptions{})
	client.EndTrace(trace2, nil)
	
	// Give some time for flush to complete
	time.Sleep(100 * time.Millisecond)
	
	if len(client.traceBatch) != 0 {
		t.Error("Batch should be flushed after reaching batch size")
	}
}

// Mock evaluator for testing
type MockEvaluator struct {
	name string
}

func (m *MockEvaluator) Name() string {
	return m.name
}

func (m *MockEvaluator) Evaluate(trace *Trace) (float64, map[string]interface{}) {
	return 0.95, map[string]interface{}{
		"test": true,
	}
}