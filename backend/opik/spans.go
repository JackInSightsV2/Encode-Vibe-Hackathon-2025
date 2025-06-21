package opik

import (
	"sync"
	"time"
)

// SpanType represents different types of spans
type SpanType string

const (
	SpanTypeRegex     SpanType = "regex_moderation"
	SpanTypeLLM       SpanType = "llm_moderation"
	SpanTypePII       SpanType = "pii_detection"
	SpanTypeSecurity  SpanType = "security_validation"
	SpanTypeProvider  SpanType = "provider_request"
	SpanTypeRateLimit SpanType = "rate_limiting"
	SpanTypeAuth      SpanType = "authentication"
	SpanTypeCache     SpanType = "cache_operation"
)

// SpanKind represents the kind of span
type SpanKind string

const (
	SpanKindInternal SpanKind = "internal"
	SpanKindClient   SpanKind = "client"
	SpanKindServer   SpanKind = "server"
	SpanKindProducer SpanKind = "producer"
	SpanKindConsumer SpanKind = "consumer"
)

// SpanOptions contains options for creating a span
type SpanOptions struct {
	Input    interface{}
	Metadata map[string]interface{}
	SpanKind SpanKind
}

// Span represents a unit of work within a trace
type Span struct {
	ID        string                 `json:"id"`
	TraceID   string                 `json:"trace_id"`
	Name      string                 `json:"name"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Input     interface{}            `json:"input"`
	Output    interface{}            `json:"output"`
	Metadata  map[string]interface{} `json:"metadata"`
	SpanKind  SpanKind               `json:"span_kind"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error,omitempty"`
	Scores    map[string]float64     `json:"scores,omitempty"`
	
	mu    sync.RWMutex
	trace *Trace
}

// StartSpan creates a new span with the specified type
func (oc *OpikClient) StartSpan(trace *Trace, spanType SpanType, input interface{}) *Span {
	options := SpanOptions{
		Input:    input,
		SpanKind: SpanKindInternal,
		Metadata: map[string]interface{}{
			"span_type": string(spanType),
		},
	}

	return trace.StartSpan(string(spanType), options)
}

// End completes the span
func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	s.Status = "completed"

	// Add to client's span batch if needed
	if s.trace != nil && s.trace.client != nil {
		s.trace.client.batchMu.Lock()
		s.trace.client.spanBatch = append(s.trace.client.spanBatch, *s)
		if len(s.trace.client.spanBatch) >= s.trace.client.config.BatchSize {
			s.trace.client.flushSpans()
		}
		s.trace.client.batchMu.Unlock()
	}
}

// SetOutput sets the span's output
func (s *Span) SetOutput(output interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Output = output
}

// SetMetadata adds or updates metadata
func (s *Span) SetMetadata(metadata map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}

	for k, v := range metadata {
		s.Metadata[k] = v
	}
}

// SetScore sets a score for the span
func (s *Span) SetScore(name string, score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Scores == nil {
		s.Scores = make(map[string]float64)
	}
	s.Scores[name] = score
}

// SetError marks the span as failed with an error
func (s *Span) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = "error"
	s.Error = err.Error()
	
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata["error_time"] = time.Now()
}

// AddTag adds a tag to the span metadata
func (s *Span) AddTag(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}

	tags, ok := s.Metadata["tags"].(map[string]interface{})
	if !ok {
		tags = make(map[string]interface{})
		s.Metadata["tags"] = tags
	}
	
	tags[key] = value
}

// GetDuration returns the span duration
func (s *Span) GetDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Duration > 0 {
		return s.Duration
	}

	if !s.EndTime.IsZero() {
		return s.EndTime.Sub(s.StartTime)
	}

	return time.Since(s.StartTime)
}