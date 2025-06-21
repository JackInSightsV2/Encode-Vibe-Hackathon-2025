package opik

import (
	"context"
	"fmt"
	"log"
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

	trace := &Trace{
		ID:        uuid.New().String(),
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

	// TODO: Implement actual HTTP request to Opik API
	// For now, just log
	log.Printf("Flushing %d traces to Opik", len(oc.traceBatch))
	
	// Clear batch
	oc.traceBatch = make([]Trace, 0, oc.config.BatchSize)
}

// flushSpans sends batched spans to Opik
func (oc *OpikClient) flushSpans() {
	if len(oc.spanBatch) == 0 {
		return
	}

	// TODO: Implement actual HTTP request to Opik API
	// For now, just log
	log.Printf("Flushing %d spans to Opik", len(oc.spanBatch))
	
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

// GetTraceFromContext retrieves the current trace from context
func GetTraceFromContext(ctx context.Context) *Trace {
	if trace, ok := ctx.Value(traceContextKey{}).(*Trace); ok {
		return trace
	}
	return nil
}