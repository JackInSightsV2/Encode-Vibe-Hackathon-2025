package opik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OpikConfig holds configuration for Opik integration
type OpikConfig struct {
	APIKey      string
	ProjectName string
	BaseURL     string
	BatchSize   int
	FlushInterval time.Duration
	Enabled     bool
}

// Client represents the Opik client interface
type Client interface {
	StartTrace(ctx context.Context, name string, options TraceOptions) (*Trace, error)
	EndTrace(trace *Trace, output map[string]interface{}) error
	RegisterEvaluator(evaluator Evaluator) error
	Close() error
}

// OpikClient handles communication with Opik service
type OpikClient struct {
	config    OpikConfig
	projectID string
	tracer    *Tracer
	evaluators []Evaluator
	mu        sync.RWMutex
	closed    bool
	httpClient *http.Client
	
	// Batching
	traceBatch    []Trace
	spanBatch     []Span
	batchMu       sync.Mutex
	flushTicker   *time.Ticker
	shutdownChan  chan struct{}
}

// TraceOptions contains options for creating a trace
type TraceOptions struct {
	Input    map[string]interface{}
	Metadata map[string]interface{}
	Tags     []string
}

// OpikTracePayload represents the payload sent to Opik API for traces
type OpikTracePayload struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	StartTime  string                 `json:"start_time"`
	EndTime    string                 `json:"end_time,omitempty"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	ProjectID  string                 `json:"project_id"`
}

// OpikSpanPayload represents the payload sent to Opik API for spans
type OpikSpanPayload struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"trace_id"`
	ParentID  string                 `json:"parent_id,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	StartTime string                 `json:"start_time"`
	EndTime   string                 `json:"end_time,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// NewOpikClient creates a new Opik client
func NewOpikClient(config OpikConfig) (*OpikClient, error) {
	if !config.Enabled {
		log.Println("Opik integration is disabled")
		return &OpikClient{config: config}, nil
	}

	if config.APIKey == "" {
		// Try to get from environment
		config.APIKey = os.Getenv("OPIK_API_KEY")
		if config.APIKey == "" {
			return nil, fmt.Errorf("OPIK_API_KEY is required")
		}
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.opik.com"
	}

	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}

	client := &OpikClient{
		config:       config,
		projectID:    config.ProjectName,
		traceBatch:   make([]Trace, 0, config.BatchSize),
		spanBatch:    make([]Span, 0, config.BatchSize),
		shutdownChan: make(chan struct{}),
		evaluators:   make([]Evaluator, 0),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}

	// Create tracer
	client.tracer = NewTracer(client)

	// Start background flush routine
	client.startFlushRoutine()

	log.Printf("Opik client initialized for project: %s", config.ProjectName)
	return client, nil
}

// StartTrace begins a new trace
func (oc *OpikClient) StartTrace(ctx context.Context, name string, options TraceOptions) (*Trace, error) {
	if !oc.config.Enabled {
		return &Trace{ID: "disabled"}, nil
	}

	oc.mu.RLock()
	if oc.closed {
		oc.mu.RUnlock()
		return nil, fmt.Errorf("client is closed")
	}
	oc.mu.RUnlock()

	// Generate UUID v7 for trace ID
	traceUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
	}

	trace := &Trace{
		ID:        traceUUID.String(),
		Name:      name,
		StartTime: time.Now(),
		Input:     options.Input,
		Metadata:  options.Metadata,
		Tags:      options.Tags,
		ProjectID: oc.projectID,
		Status:    "in_progress",
		Spans:     make([]*Span, 0),
		client:    oc,
	}

	// Store trace context
	ctx = context.WithValue(ctx, traceContextKey{}, trace)

	return trace, nil
}

// EndTrace completes a trace
func (oc *OpikClient) EndTrace(trace *Trace, output map[string]interface{}) error {
	if !oc.config.Enabled || trace.ID == "disabled" {
		return nil
	}

	trace.mu.Lock()
	trace.EndTime = time.Now()
	trace.Duration = trace.EndTime.Sub(trace.StartTime)
	trace.Output = output
	trace.Status = "completed"
	trace.mu.Unlock()

	// Run evaluators
	if len(oc.evaluators) > 0 {
		oc.runEvaluators(trace)
	}

	// Add to batch
	oc.batchMu.Lock()
	oc.traceBatch = append(oc.traceBatch, *trace)
	
	// Check if batch is full
	if len(oc.traceBatch) >= oc.config.BatchSize {
		oc.flushTraces()
	}
	oc.batchMu.Unlock()

	return nil
}

// RegisterEvaluator adds a new evaluator
func (oc *OpikClient) RegisterEvaluator(evaluator Evaluator) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	oc.evaluators = append(oc.evaluators, evaluator)
	log.Printf("Registered evaluator: %s", evaluator.Name())
	
	return nil
}

// runEvaluators executes all registered evaluators on a trace
func (oc *OpikClient) runEvaluators(trace *Trace) {
	for _, evaluator := range oc.evaluators {
		go func(e Evaluator) {
			score, details := e.Evaluate(trace)
			trace.AddScore(e.Name(), score, details)
		}(evaluator)
	}
}

// startFlushRoutine starts the background flush routine
func (oc *OpikClient) startFlushRoutine() {
	if !oc.config.Enabled {
		return
	}
	
	oc.flushTicker = time.NewTicker(oc.config.FlushInterval)
	
	go func() {
		for {
			select {
			case <-oc.flushTicker.C:
				oc.flush()
			case <-oc.shutdownChan:
				return
			}
		}
	}()
}

// flush sends all batched data
func (oc *OpikClient) flush() {
	oc.batchMu.Lock()
	defer oc.batchMu.Unlock()

	if len(oc.traceBatch) > 0 {
		oc.flushTraces()
	}

	if len(oc.spanBatch) > 0 {
		oc.flushSpans()
	}
}

// flushTraces sends batched traces to Opik
func (oc *OpikClient) flushTraces() {
	if len(oc.traceBatch) == 0 {
		return
	}

	// Convert traces to Opik API format
	payloads := make([]OpikTracePayload, len(oc.traceBatch))
	for i, trace := range oc.traceBatch {
		payloads[i] = OpikTracePayload{
			ID:          trace.ID,
			Name:        trace.Name,
			StartTime:   trace.StartTime.Format(time.RFC3339),
			EndTime:     trace.EndTime.Format(time.RFC3339),
			Input:       trace.Input,
			Output:      trace.Output,
			Metadata:    trace.Metadata,
			Tags:        trace.Tags,
			ProjectID:   "01979287-3120-7763-a3d5-0cb8eda04a54", // QT-1 Middleware project ID
		}
	}

	// Send traces to Opik API individually
	for _, payload := range payloads {
		if err := oc.sendSingleTraceToOpik(payload); err != nil {
			log.Printf("Failed to send trace to Opik: %v", err)
			continue // Continue with other traces even if one fails
		} else {
			log.Printf("Successfully sent trace %s to Opik", payload.ID)
		}
	}

	log.Printf("Successfully flushed %d traces to Opik", len(oc.traceBatch))
	
	// Clear batch
	oc.traceBatch = make([]Trace, 0, oc.config.BatchSize)
}

// flushSpans sends batched spans to Opik
func (oc *OpikClient) flushSpans() {
	if len(oc.spanBatch) == 0 {
		return
	}

	// Convert spans to Opik API format
	payloads := make([]OpikSpanPayload, len(oc.spanBatch))
	for i, span := range oc.spanBatch {
		// Convert input to map[string]interface{} if possible
		var input map[string]interface{}
		if inputMap, ok := span.Input.(map[string]interface{}); ok {
			input = inputMap
		} else if span.Input != nil {
			input = map[string]interface{}{"data": span.Input}
		}

		// Convert output to map[string]interface{} if possible
		var output map[string]interface{}
		if outputMap, ok := span.Output.(map[string]interface{}); ok {
			output = outputMap
		} else if span.Output != nil {
			output = map[string]interface{}{"data": span.Output}
		}

		// Extract span type from metadata
		spanType := "unknown"
		if span.Metadata != nil {
			if typeValue, ok := span.Metadata["span_type"].(string); ok {
				spanType = typeValue
			}
		}

		// Extract tags from metadata
		var tags []string
		if span.Metadata != nil {
			if tagsMap, ok := span.Metadata["tags"].(map[string]interface{}); ok {
				for key := range tagsMap {
					tags = append(tags, key)
				}
			}
		}

		payloads[i] = OpikSpanPayload{
			ID:        span.ID,
			TraceID:   span.TraceID,
			ParentID:  "", // No parent ID in current implementation
			Name:      span.Name,
			Type:      spanType,
			StartTime: span.StartTime.Format(time.RFC3339),
			EndTime:   span.EndTime.Format(time.RFC3339),
			Input:     input,
			Output:    output,
			Metadata:  span.Metadata,
			Tags:      tags,
		}
	}

	// Send to Opik API
	if err := oc.sendSpansToOpik(payloads); err != nil {
		log.Printf("Failed to send spans to Opik: %v", err)
		return
	}

	log.Printf("Successfully flushed %d spans to Opik", len(oc.spanBatch))
	
	// Clear batch
	oc.spanBatch = make([]Span, 0, oc.config.BatchSize)
}

// Close gracefully shuts down the client
func (oc *OpikClient) Close() error {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if oc.closed || !oc.config.Enabled {
		return nil
	}

	oc.closed = true
	
	// Stop flush routine
	if oc.flushTicker != nil {
		oc.flushTicker.Stop()
	}
	
	// Only close channel if it was initialized
	if oc.shutdownChan != nil {
		close(oc.shutdownChan)
	}

	// Final flush
	oc.flush()

	log.Println("Opik client closed")
	return nil
}

// Context key for trace storage
type traceContextKey struct{}

// sendSingleTraceToOpik sends a single trace to the Opik API
func (oc *OpikClient) sendSingleTraceToOpik(trace OpikTracePayload) error {
	if !oc.config.Enabled {
		return nil
	}

	jsonData, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %w", err)
	}

	log.Printf("Sending trace to Opik: %s", string(jsonData))

	req, err := http.NewRequest("POST", oc.config.BaseURL+"/v1/private/traces", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Comet-Workspace", "jisencodevibehackathon2025")
	req.Header.Set("authorization", oc.config.APIKey)

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Opik traces API error response: %s", string(body))
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// sendSpansToOpik sends spans to the Opik API
func (oc *OpikClient) sendSpansToOpik(spans []OpikSpanPayload) error {
	if !oc.config.Enabled {
		return nil
	}

	jsonData, err := json.Marshal(spans)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	req, err := http.NewRequest("POST", oc.config.BaseURL+"/v1/private/spans", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Comet-Workspace", "jisencodevibehackathon2025")
	req.Header.Set("authorization", oc.config.APIKey)

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetTraceFromContext retrieves the current trace from context
func GetTraceFromContext(ctx context.Context) *Trace {
	if trace, ok := ctx.Value(traceContextKey{}).(*Trace); ok {
		return trace
	}
	return nil
}