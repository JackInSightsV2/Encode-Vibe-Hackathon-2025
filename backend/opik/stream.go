package opik

import (
	"encoding/json"
	"sync"
	"time"
)

// EventType represents different types of events
type EventType string

const (
	EventTypeTrace      EventType = "trace"
	EventTypeSpan       EventType = "span"
	EventTypeThreat     EventType = "threat"
	EventTypeModeration EventType = "moderation"
	EventTypeProvider   EventType = "provider"
	EventTypeMetric     EventType = "metric"
	EventTypeAlert      EventType = "alert"
)

// Event represents a real-time event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Severity  string                 `json:"severity,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// EventStreamManager manages real-time event streaming
type EventStreamManager struct {
	opikClient  *OpikClient
	buffer      *RingBuffer
	subscribers map[string]chan Event
	filters     map[string]EventFilter
	mu          sync.RWMutex
	
	// Aggregation
	aggregator  *EventAggregator
	
	// Metrics
	eventCount  map[EventType]int64
	metricsMu   sync.RWMutex
}

// EventFilter defines criteria for filtering events
type EventFilter struct {
	Types      []EventType
	Severities []string
	Tags       []string
	TraceIDs   []string
}

// NewEventStreamManager creates a new event stream manager
func NewEventStreamManager(opikClient *OpikClient, bufferSize int) *EventStreamManager {
	esm := &EventStreamManager{
		opikClient:  opikClient,
		buffer:      NewRingBuffer(bufferSize),
		subscribers: make(map[string]chan Event),
		filters:     make(map[string]EventFilter),
		eventCount:  make(map[EventType]int64),
		aggregator:  NewEventAggregator(),
	}
	
	// Start aggregation routine
	go esm.aggregator.Start()
	
	return esm
}

// PublishEvent publishes an event to all subscribers
func (esm *EventStreamManager) PublishEvent(event Event) {
	// Add to buffer
	esm.buffer.Push(event)
	
	// Update metrics
	esm.updateMetrics(event)
	
	// Send to aggregator
	esm.aggregator.AddEvent(event)
	
	// Notify subscribers
	esm.mu.RLock()
	defer esm.mu.RUnlock()
	
	for clientID, ch := range esm.subscribers {
		// Apply filters
		if esm.shouldSendEvent(clientID, event) {
			select {
			case ch <- event:
			default:
				// Non-blocking send - skip if channel is full
			}
		}
	}
}

// Subscribe creates a subscription for events
func (esm *EventStreamManager) Subscribe(clientID string, bufferSize int) <-chan Event {
	esm.mu.Lock()
	defer esm.mu.Unlock()
	
	ch := make(chan Event, bufferSize)
	esm.subscribers[clientID] = ch
	
	return ch
}

// Unsubscribe removes a subscription
func (esm *EventStreamManager) Unsubscribe(clientID string) {
	esm.mu.Lock()
	defer esm.mu.Unlock()
	
	if ch, exists := esm.subscribers[clientID]; exists {
		close(ch)
		delete(esm.subscribers, clientID)
		delete(esm.filters, clientID)
	}
}

// SetFilter sets event filter for a client
func (esm *EventStreamManager) SetFilter(clientID string, filter EventFilter) {
	esm.mu.Lock()
	defer esm.mu.Unlock()
	
	esm.filters[clientID] = filter
}

// GetBufferedEvents returns recent events from the buffer
func (esm *EventStreamManager) GetBufferedEvents(count int) []Event {
	items := esm.buffer.GetRecent(count)
	events := make([]Event, 0, len(items))
	
	for _, item := range items {
		if event, ok := item.(Event); ok {
			events = append(events, event)
		}
	}
	
	return events
}

// GetAggregatedData returns aggregated event data
func (esm *EventStreamManager) GetAggregatedData(window time.Duration) map[string]interface{} {
	return esm.aggregator.GetAggregatedData(window)
}

// GetMetrics returns event metrics
func (esm *EventStreamManager) GetMetrics() map[EventType]int64 {
	esm.metricsMu.RLock()
	defer esm.metricsMu.RUnlock()
	
	// Create a copy
	metrics := make(map[EventType]int64)
	for k, v := range esm.eventCount {
		metrics[k] = v
	}
	
	return metrics
}

// shouldSendEvent checks if an event should be sent to a client based on filters
func (esm *EventStreamManager) shouldSendEvent(clientID string, event Event) bool {
	filter, hasFilter := esm.filters[clientID]
	if !hasFilter {
		return true // No filter means send all events
	}
	
	// Check event type filter
	if len(filter.Types) > 0 {
		typeMatch := false
		for _, t := range filter.Types {
			if event.Type == t {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}
	
	// Check severity filter
	if len(filter.Severities) > 0 && event.Severity != "" {
		severityMatch := false
		for _, s := range filter.Severities {
			if event.Severity == s {
				severityMatch = true
				break
			}
		}
		if !severityMatch {
			return false
		}
	}
	
	// Check tag filter
	if len(filter.Tags) > 0 && len(event.Tags) > 0 {
		tagMatch := false
		for _, filterTag := range filter.Tags {
			for _, eventTag := range event.Tags {
				if filterTag == eventTag {
					tagMatch = true
					break
				}
			}
			if tagMatch {
				break
			}
		}
		if !tagMatch {
			return false
		}
	}
	
	// Check trace ID filter
	if len(filter.TraceIDs) > 0 && event.TraceID != "" {
		traceMatch := false
		for _, tid := range filter.TraceIDs {
			if event.TraceID == tid {
				traceMatch = true
				break
			}
		}
		if !traceMatch {
			return false
		}
	}
	
	return true
}

// updateMetrics updates event count metrics
func (esm *EventStreamManager) updateMetrics(event Event) {
	esm.metricsMu.Lock()
	defer esm.metricsMu.Unlock()
	
	esm.eventCount[event.Type]++
}

// CreateThreatEvent creates a threat detection event
func (esm *EventStreamManager) CreateThreatEvent(trace *Trace, threatType string, severity string, confidence float64) Event {
	return Event{
		ID:        generateEventID(),
		Type:      EventTypeThreat,
		Timestamp: time.Now(),
		TraceID:   trace.ID,
		Severity:  severity,
		Data: map[string]interface{}{
			"threat_type": threatType,
			"confidence":  confidence,
			"user_id":     trace.Metadata["user_id"],
			"session_id":  trace.Metadata["session_id"],
		},
		Tags: []string{threatType, severity},
	}
}

// CreateModerationEvent creates a moderation event
func (esm *EventStreamManager) CreateModerationEvent(trace *Trace, action string, score float64, blocked bool) Event {
	return Event{
		ID:        generateEventID(),
		Type:      EventTypeModeration,
		Timestamp: time.Now(),
		TraceID:   trace.ID,
		Data: map[string]interface{}{
			"action":  action,
			"score":   score,
			"blocked": blocked,
		},
		Tags: []string{action},
	}
}

// CreateProviderEvent creates a provider-related event
func (esm *EventStreamManager) CreateProviderEvent(provider string, status string, responseTime int64) Event {
	return Event{
		ID:        generateEventID(),
		Type:      EventTypeProvider,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"provider":      provider,
			"status":        status,
			"response_time": responseTime,
		},
		Tags: []string{provider, status},
	}
}

// CreateMetricEvent creates a metric event
func (esm *EventStreamManager) CreateMetricEvent(metricName string, value interface{}, labels map[string]string) Event {
	return Event{
		ID:        generateEventID(),
		Type:      EventTypeMetric,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"metric": metricName,
			"value":  value,
			"labels": labels,
		},
		Tags: []string{metricName},
	}
}

// CreateAlertEvent creates an alert event
func (esm *EventStreamManager) CreateAlertEvent(alertType string, severity string, message string, details map[string]interface{}) Event {
	return Event{
		ID:        generateEventID(),
		Type:      EventTypeAlert,
		Timestamp: time.Now(),
		Severity:  severity,
		Data: map[string]interface{}{
			"alert_type": alertType,
			"message":    message,
			"details":    details,
		},
		Tags: []string{alertType, severity},
	}
}

// Close shuts down the event stream manager
func (esm *EventStreamManager) Close() {
	esm.mu.Lock()
	defer esm.mu.Unlock()
	
	// Close all subscriber channels
	for _, ch := range esm.subscribers {
		close(ch)
	}
	
	// Clear subscribers
	esm.subscribers = make(map[string]chan Event)
	
	// Stop aggregator
	esm.aggregator.Stop()
}

// Helper function to generate event IDs
func generateEventID() string {
	return "evt_" + time.Now().Format("20060102150405") + "_" + generateRandomString(8)
}

// Helper function to generate random strings
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// RingBuffer is a circular buffer for storing recent events
type RingBuffer struct {
	items    []interface{}
	size     int
	head     int
	count    int
	mu       sync.RWMutex
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		items: make([]interface{}, size),
		size:  size,
	}
}

// Push adds an item to the buffer
func (rb *RingBuffer) Push(item interface{}) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	
	rb.items[rb.head] = item
	rb.head = (rb.head + 1) % rb.size
	
	if rb.count < rb.size {
		rb.count++
	}
}

// GetRecent returns the most recent n items
func (rb *RingBuffer) GetRecent(n int) []interface{} {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	
	if n > rb.count {
		n = rb.count
	}
	
	result := make([]interface{}, n)
	
	// Start from the most recent item
	start := (rb.head - n + rb.size) % rb.size
	
	for i := 0; i < n; i++ {
		idx := (start + i) % rb.size
		result[i] = rb.items[idx]
	}
	
	return result
}

// EventWebSocketMessage represents a WebSocket message for events
type EventWebSocketMessage struct {
	Type    string          `json:"type"`
	Event   *Event          `json:"event,omitempty"`
	Events  []Event         `json:"events,omitempty"`
	Command string          `json:"command,omitempty"`
	Filter  *EventFilter    `json:"filter,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Event
func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Alias
		TimestampUnix int64 `json:"timestamp_unix"`
	}{
		Alias:         Alias(e),
		TimestampUnix: e.Timestamp.Unix(),
	})
}