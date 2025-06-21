package opik

import (
	"sync"
	"time"
)

// EventAggregator aggregates events for analytics
type EventAggregator struct {
	metrics      map[string]*MetricAggregator
	timeWindows  []time.Duration
	mu           sync.RWMutex
	stopChan     chan struct{}
}

// MetricAggregator aggregates metrics for a specific type
type MetricAggregator struct {
	Name         string
	Type         string
	Count        int64
	Sum          float64
	Min          float64
	Max          float64
	LastValue    float64
	LastUpdate   time.Time
	WindowData   map[time.Duration]*WindowStats
}

// WindowStats represents statistics for a time window
type WindowStats struct {
	StartTime    time.Time
	Count        int64
	Sum          float64
	Min          float64
	Max          float64
	Events       []Event
}

// AggregatedData represents aggregated event data
type AggregatedData struct {
	Timestamp    time.Time                       `json:"timestamp"`
	Window       time.Duration                   `json:"window"`
	EventCounts  map[EventType]int64             `json:"event_counts"`
	ThreatCounts map[string]int64                `json:"threat_counts"`
	ProviderStats map[string]*ProviderStats      `json:"provider_stats"`
	Metrics      map[string]*MetricSummary       `json:"metrics"`
}

// ProviderStats represents provider-specific statistics
type ProviderStats struct {
	Provider         string  `json:"provider"`
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	ErrorCount       int64   `json:"error_count"`
	AvgResponseTime  float64 `json:"avg_response_time"`
	P95ResponseTime  float64 `json:"p95_response_time"`
	P99ResponseTime  float64 `json:"p99_response_time"`
}

// MetricSummary represents a summary of a metric
type MetricSummary struct {
	Name    string  `json:"name"`
	Count   int64   `json:"count"`
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Current float64 `json:"current"`
}

// NewEventAggregator creates a new event aggregator
func NewEventAggregator() *EventAggregator {
	return &EventAggregator{
		metrics:     make(map[string]*MetricAggregator),
		timeWindows: []time.Duration{
			1 * time.Minute,
			5 * time.Minute,
			15 * time.Minute,
			1 * time.Hour,
		},
		stopChan: make(chan struct{}),
	}
}

// Start begins the aggregation routine
func (ea *EventAggregator) Start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ea.cleanupOldData()
		case <-ea.stopChan:
			return
		}
	}
}

// Stop stops the aggregation routine
func (ea *EventAggregator) Stop() {
	close(ea.stopChan)
}

// AddEvent adds an event to the aggregator
func (ea *EventAggregator) AddEvent(event Event) {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	// Process based on event type
	switch event.Type {
	case EventTypeThreat:
		ea.processThreatEvent(event)
	case EventTypeProvider:
		ea.processProviderEvent(event)
	case EventTypeMetric:
		ea.processMetricEvent(event)
	case EventTypeModeration:
		ea.processModerationEvent(event)
	}

	// Update general event counts
	for _, window := range ea.timeWindows {
		ea.updateWindowStats(event, window)
	}
}

// processThreatEvent processes threat detection events
func (ea *EventAggregator) processThreatEvent(event Event) {
	threatType, _ := event.Data["threat_type"].(string)
	confidence, _ := event.Data["confidence"].(float64)

	key := "threat_" + threatType
	aggregator, exists := ea.metrics[key]
	if !exists {
		aggregator = &MetricAggregator{
			Name:       key,
			Type:       "threat",
			Min:        confidence,
			Max:        confidence,
			WindowData: make(map[time.Duration]*WindowStats),
		}
		ea.metrics[key] = aggregator
	}

	aggregator.Count++
	aggregator.Sum += confidence
	aggregator.LastValue = confidence
	aggregator.LastUpdate = event.Timestamp

	if confidence < aggregator.Min {
		aggregator.Min = confidence
	}
	if confidence > aggregator.Max {
		aggregator.Max = confidence
	}
}

// processProviderEvent processes provider-related events
func (ea *EventAggregator) processProviderEvent(event Event) {
	provider, _ := event.Data["provider"].(string)
	status, _ := event.Data["status"].(string)
	responseTime, _ := event.Data["response_time"].(int64)

	key := "provider_" + provider
	aggregator, exists := ea.metrics[key]
	if !exists {
		aggregator = &MetricAggregator{
			Name:       key,
			Type:       "provider",
			Min:        float64(responseTime),
			Max:        float64(responseTime),
			WindowData: make(map[time.Duration]*WindowStats),
		}
		ea.metrics[key] = aggregator
	}

	aggregator.Count++
	rtFloat := float64(responseTime)
	aggregator.Sum += rtFloat
	aggregator.LastValue = rtFloat
	aggregator.LastUpdate = event.Timestamp

	if rtFloat < aggregator.Min {
		aggregator.Min = rtFloat
	}
	if rtFloat > aggregator.Max {
		aggregator.Max = rtFloat
	}

	// Track success/error counts
	if status == "success" {
		successKey := key + "_success"
		if successAgg, exists := ea.metrics[successKey]; exists {
			successAgg.Count++
		} else {
			ea.metrics[successKey] = &MetricAggregator{
				Name:  successKey,
				Type:  "counter",
				Count: 1,
			}
		}
	} else if status == "error" {
		errorKey := key + "_error"
		if errorAgg, exists := ea.metrics[errorKey]; exists {
			errorAgg.Count++
		} else {
			ea.metrics[errorKey] = &MetricAggregator{
				Name:  errorKey,
				Type:  "counter",
				Count: 1,
			}
		}
	}
}

// processMetricEvent processes metric events
func (ea *EventAggregator) processMetricEvent(event Event) {
	metricName, _ := event.Data["metric"].(string)
	value, _ := event.Data["value"].(float64)

	aggregator, exists := ea.metrics[metricName]
	if !exists {
		aggregator = &MetricAggregator{
			Name:       metricName,
			Type:       "metric",
			Min:        value,
			Max:        value,
			WindowData: make(map[time.Duration]*WindowStats),
		}
		ea.metrics[metricName] = aggregator
	}

	aggregator.Count++
	aggregator.Sum += value
	aggregator.LastValue = value
	aggregator.LastUpdate = event.Timestamp

	if value < aggregator.Min {
		aggregator.Min = value
	}
	if value > aggregator.Max {
		aggregator.Max = value
	}
}

// processModerationEvent processes moderation events
func (ea *EventAggregator) processModerationEvent(event Event) {
	action, _ := event.Data["action"].(string)
	score, _ := event.Data["score"].(float64)
	blocked, _ := event.Data["blocked"].(bool)

	key := "moderation_" + action
	aggregator, exists := ea.metrics[key]
	if !exists {
		aggregator = &MetricAggregator{
			Name:       key,
			Type:       "moderation",
			Min:        score,
			Max:        score,
			WindowData: make(map[time.Duration]*WindowStats),
		}
		ea.metrics[key] = aggregator
	}

	aggregator.Count++
	aggregator.Sum += score
	aggregator.LastValue = score
	aggregator.LastUpdate = event.Timestamp

	if score < aggregator.Min {
		aggregator.Min = score
	}
	if score > aggregator.Max {
		aggregator.Max = score
	}

	// Track blocked count
	if blocked {
		blockedKey := "moderation_blocked"
		if blockedAgg, exists := ea.metrics[blockedKey]; exists {
			blockedAgg.Count++
		} else {
			ea.metrics[blockedKey] = &MetricAggregator{
				Name:  blockedKey,
				Type:  "counter",
				Count: 1,
			}
		}
	}
}

// updateWindowStats updates statistics for a specific time window
func (ea *EventAggregator) updateWindowStats(event Event, window time.Duration) {
	now := time.Now()
	windowStart := now.Truncate(window)

	// Update window stats for each relevant metric
	for _, aggregator := range ea.metrics {
		stats, exists := aggregator.WindowData[window]
		if !exists || stats.StartTime != windowStart {
			// New window
			stats = &WindowStats{
				StartTime: windowStart,
				Min:       1e9,
				Max:       -1e9,
				Events:    make([]Event, 0),
			}
			aggregator.WindowData[window] = stats
		}

		// Add event to window
		stats.Events = append(stats.Events, event)
		stats.Count++
	}
}

// GetAggregatedData returns aggregated data for a specific time window
func (ea *EventAggregator) GetAggregatedData(window time.Duration) map[string]interface{} {
	ea.mu.RLock()
	defer ea.mu.RUnlock()

	data := &AggregatedData{
		Timestamp:     time.Now(),
		Window:        window,
		EventCounts:   make(map[EventType]int64),
		ThreatCounts:  make(map[string]int64),
		ProviderStats: make(map[string]*ProviderStats),
		Metrics:       make(map[string]*MetricSummary),
	}

	// Aggregate metrics
	for name, aggregator := range ea.metrics {
		if aggregator.Count == 0 {
			continue
		}

		switch aggregator.Type {
		case "threat":
			threatType := name[7:] // Remove "threat_" prefix
			data.ThreatCounts[threatType] = aggregator.Count

		case "provider":
			if len(name) > 9 && name[:9] == "provider_" {
				providerName := name[9:]
				stats, exists := data.ProviderStats[providerName]
				if !exists {
					stats = &ProviderStats{Provider: providerName}
					data.ProviderStats[providerName] = stats
				}

				stats.RequestCount = aggregator.Count
				if aggregator.Count > 0 {
					stats.AvgResponseTime = aggregator.Sum / float64(aggregator.Count)
				}

				// Get success/error counts
				if successAgg, exists := ea.metrics["provider_"+providerName+"_success"]; exists {
					stats.SuccessCount = successAgg.Count
				}
				if errorAgg, exists := ea.metrics["provider_"+providerName+"_error"]; exists {
					stats.ErrorCount = errorAgg.Count
				}
			}

		default:
			// Generic metrics
			summary := &MetricSummary{
				Name:    name,
				Count:   aggregator.Count,
				Min:     aggregator.Min,
				Max:     aggregator.Max,
				Current: aggregator.LastValue,
			}
			if aggregator.Count > 0 {
				summary.Average = aggregator.Sum / float64(aggregator.Count)
			}
			data.Metrics[name] = summary
		}
	}

	return map[string]interface{}{
		"data": data,
	}
}

// cleanupOldData removes old window data
func (ea *EventAggregator) cleanupOldData() {
	ea.mu.Lock()
	defer ea.mu.Unlock()

	now := time.Now()

	for _, aggregator := range ea.metrics {
		for window, stats := range aggregator.WindowData {
			// Remove windows older than 2x the window duration
			if now.Sub(stats.StartTime) > window*2 {
				delete(aggregator.WindowData, window)
			}
		}
	}
}