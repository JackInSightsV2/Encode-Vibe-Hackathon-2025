package opik

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Tracer handles trace creation and management
type Tracer struct {
	client *OpikClient
	name   string
}

// Trace represents a single trace
type Trace struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	ProjectID string                 `json:"project_id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Input     map[string]interface{} `json:"input"`
	Output    map[string]interface{} `json:"output"`
	Metadata  map[string]interface{} `json:"metadata"`
	Tags      []string               `json:"tags"`
	Status    string                 `json:"status"`
	Spans     []*Span                `json:"spans"`
	Scores    map[string]Score       `json:"scores"`
	
	mu     sync.RWMutex
	client *OpikClient
}

// Score represents an evaluation score
type Score struct {
	Value   float64                `json:"value"`
	Details map[string]interface{} `json:"details"`
}

// NewTracer creates a new tracer
func NewTracer(client *OpikClient) *Tracer {
	return &Tracer{
		client: client,
		name:   "qt1-middleware",
	}
}

// StartTrace begins a new trace
func (t *Tracer) StartTrace(ctx context.Context, name string, options TraceOptions) (*Trace, error) {
	return t.client.StartTrace(ctx, name, options)
}

// TraceModeration creates a trace specifically for moderation requests
func (oc *OpikClient) TraceModeration(ctx context.Context, request ModerationRequest) (*Trace, error) {
	options := TraceOptions{
		Input: map[string]interface{}{
			"request_id":   request.ID,
			"provider":     request.Provider,
			"timestamp":    request.Timestamp,
			"content_size": len(request.Content),
		},
		Metadata: map[string]interface{}{
			"user_id":    request.UserID,
			"session_id": request.SessionID,
			"ip_address": request.IPAddress,
			"endpoint":   request.Endpoint,
		},
		Tags: []string{"moderation", request.Provider},
	}

	return oc.StartTrace(ctx, "moderation_request", options)
}

// StartSpan creates a new span within a trace
func (t *Trace) StartSpan(name string, options SpanOptions) *Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	span := &Span{
		ID:        fmt.Sprintf("%s-%d", t.ID, len(t.Spans)),
		TraceID:   t.ID,
		Name:      name,
		StartTime: time.Now(),
		Input:     options.Input,
		Metadata:  options.Metadata,
		SpanKind:  options.SpanKind,
		Status:    "in_progress",
		trace:     t,
	}

	t.Spans = append(t.Spans, span)
	return span
}

// GetSpan retrieves a span by name
func (t *Trace) GetSpan(name string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, span := range t.Spans {
		if span.Name == name {
			return span
		}
	}
	return nil
}

// AddScore adds an evaluation score to the trace
func (t *Trace) AddScore(name string, value float64, details map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Scores == nil {
		t.Scores = make(map[string]Score)
	}

	t.Scores[name] = Score{
		Value:   value,
		Details: details,
	}
}

// SetError marks the trace as failed with an error
func (t *Trace) SetError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Status = "error"
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata["error"] = err.Error()
	t.Metadata["error_time"] = time.Now()
}

// ModerationRequest represents a request to be traced
type ModerationRequest struct {
	ID        string
	Provider  string
	Timestamp time.Time
	Content   string
	UserID    string
	SessionID string
	IPAddress string
	Endpoint  string
}