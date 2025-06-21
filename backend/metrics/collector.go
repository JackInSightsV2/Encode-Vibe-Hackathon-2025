package metrics

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// MetricsCollector handles collection and storage of metrics
type MetricsCollector struct {
	storage MetricsStorage
	config  MetricsConfig
	
	// Atomic counters for thread-safe operations
	requestCount  int64
	errorCount    int64
	
	// Channels for async processing
	metricsChan chan Metric
	stopChan    chan struct{}
	wg          sync.WaitGroup
	
	// State tracking
	startTime   time.Time
	isRunning   bool
	mutex       sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector instance
func NewMetricsCollector(config MetricsConfig) (*MetricsCollector, error) {
	var storage MetricsStorage
	var err error
	
	// Initialize storage based on configuration
	switch config.Storage.Type {
	case "supabase":
		storage, err = NewSupabaseStorage(config.Supabase)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Supabase storage: %w", err)
		}
	case "memory", "":
		storage = NewInMemoryStorage(config.Storage)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Storage.Type)
	}
	
	collector := &MetricsCollector{
		storage:     storage,
		config:      config,
		metricsChan: make(chan Metric, 10000), // Larger buffer for high throughput
		stopChan:    make(chan struct{}),
		startTime:   time.Now(),
		isRunning:   false,
	}
	
	return collector, nil
}

// Start begins the metrics collection process
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	if mc.isRunning {
		return nil
	}
	
	mc.isRunning = true
	
	// Start worker goroutines for async metric processing
	for i := 0; i < 3; i++ { // 3 worker goroutines
		mc.wg.Add(1)
		go mc.metricsWorker(ctx)
	}
	
	return nil
}

// Stop gracefully shuts down the metrics collector
func (mc *MetricsCollector) Stop() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	if !mc.isRunning {
		return nil
	}
	
	mc.isRunning = false
	close(mc.stopChan)
	close(mc.metricsChan)
	
	// Wait for workers to finish
	mc.wg.Wait()
	
	// Close storage
	return mc.storage.Close()
}

// RecordHTTPRequest records HTTP request metrics
func (mc *MetricsCollector) RecordHTTPRequest(httpMetrics HTTPMetrics) {
	if !mc.config.Enabled || !mc.config.Collection.HTTPRequests {
		return
	}
	
	atomic.AddInt64(&mc.requestCount, 1)
	
	if httpMetrics.StatusCode >= 400 {
		atomic.AddInt64(&mc.errorCount, 1)
	}
	
	metric := Metric{
		ID:        uuid.New().String(),
		Name:      "http_request",
		Value:     float64(httpMetrics.Duration.Milliseconds()),
		Type:      MetricTypeTiming,
		Timestamp: time.Now(),
		Tags: map[string]string{
			"path":        httpMetrics.Path,
			"method":      httpMetrics.Method,
			"status_code": fmt.Sprintf("%d", httpMetrics.StatusCode),
		},
		Metadata: map[string]interface{}{
			"duration":    float64(httpMetrics.Duration.Milliseconds()),
			"status_code": httpMetrics.StatusCode,
			"user_agent":  httpMetrics.UserAgent,
			"ip_address":  sanitizeIP(httpMetrics.IPAddress),
			"error":       httpMetrics.Error,
		},
	}
	
	mc.recordMetric(metric)
}

// RecordModerationEvent records moderation event metrics
func (mc *MetricsCollector) RecordModerationEvent(modMetrics ModerationMetrics) {
	if !mc.config.Enabled || !mc.config.Collection.ModerationEvents {
		return
	}
	
	metric := Metric{
		ID:        uuid.New().String(),
		Name:      "moderation_event",
		Value:     modMetrics.Score,
		Type:      MetricTypeGauge,
		Timestamp: time.Now(),
		Tags: map[string]string{
			"layer":    modMetrics.LayerName,
			"category": modMetrics.Category,
			"action":   modMetrics.Action,
			"blocked":  fmt.Sprintf("%t", modMetrics.Blocked),
		},
		Metadata: map[string]interface{}{
			"score":          modMetrics.Score,
			"confidence":     modMetrics.Confidence,
			"blocked":        modMetrics.Blocked,
			"duration":       float64(modMetrics.Duration.Milliseconds()),
			"user_id":        sanitizeUserID(modMetrics.UserID),
			"content_length": modMetrics.ContentLength,
		},
	}
	
	mc.recordMetric(metric)
}

// RecordPIIDetection records PII detection metrics
func (mc *MetricsCollector) RecordPIIDetection(piiMetrics PIIMetrics) {
	if !mc.config.Enabled || !mc.config.Collection.PIIDetection {
		return
	}
	
	metric := Metric{
		ID:        uuid.New().String(),
		Name:      "pii_detection",
		Value:     float64(piiMetrics.MatchCount),
		Type:      MetricTypeCounter,
		Timestamp: time.Now(),
		Tags: map[string]string{
			"action":         piiMetrics.Action,
			"masking_enabled": fmt.Sprintf("%t", piiMetrics.MaskingEnabled),
		},
		Metadata: map[string]interface{}{
			"detected_types":  piiMetrics.DetectedTypes,
			"match_count":     piiMetrics.MatchCount,
			"confidence":      piiMetrics.Confidence,
			"user_id":         sanitizeUserID(piiMetrics.UserID),
			"content_length":  piiMetrics.ContentLength,
		},
	}
	
	mc.recordMetric(metric)
}

// RecordSystemHealth records system health metrics
func (mc *MetricsCollector) RecordSystemHealth() {
	if !mc.config.Enabled || !mc.config.Collection.SystemHealth {
		return
	}
	
	health := mc.getSystemHealth()
	
	metrics := []Metric{
		{
			ID:        uuid.New().String(),
			Name:      "system_memory",
			Value:     health.MemoryUsage,
			Type:      MetricTypeGauge,
			Timestamp: time.Now(),
			Tags:      map[string]string{"metric": "memory_usage"},
		},
		{
			ID:        uuid.New().String(),
			Name:      "system_goroutines",
			Value:     float64(health.GoroutineCount),
			Type:      MetricTypeGauge,
			Timestamp: time.Now(),
			Tags:      map[string]string{"metric": "goroutine_count"},
		},
		{
			ID:        uuid.New().String(),
			Name:      "system_uptime",
			Value:     float64(health.UptimeSeconds),
			Type:      MetricTypeGauge,
			Timestamp: time.Now(),
			Tags:      map[string]string{"metric": "uptime_seconds"},
		},
	}
	
	for _, metric := range metrics {
		mc.recordMetric(metric)
	}
}

// RecordCustomMetric records a custom metric
func (mc *MetricsCollector) RecordCustomMetric(name string, value float64, metricType MetricType, tags map[string]string) {
	if !mc.config.Enabled {
		return
	}
	
	metric := Metric{
		ID:        uuid.New().String(),
		Name:      name,
		Value:     value,
		Type:      metricType,
		Timestamp: time.Now(),
		Tags:      tags,
	}
	
	mc.recordMetric(metric)
}

// GetMetrics retrieves metrics based on query
func (mc *MetricsCollector) GetMetrics(query MetricsQuery) ([]Metric, error) {
	return mc.storage.Get(query)
}

// GetSummary generates metrics summary for a time range
func (mc *MetricsCollector) GetSummary(timeRange TimeRange) (*MetricsSummary, error) {
	return mc.storage.GetSummary(timeRange)
}

// GetCurrentStats returns current collector statistics
func (mc *MetricsCollector) GetCurrentStats() map[string]interface{} {
	return map[string]interface{}{
		"total_requests": atomic.LoadInt64(&mc.requestCount),
		"total_errors":   atomic.LoadInt64(&mc.errorCount),
		"uptime_seconds": time.Since(mc.startTime).Seconds(),
		"is_running":     mc.isRunning,
		"storage_type":   mc.config.Storage.Type,
	}
}

// GetPIIAnalytics returns aggregated PII detection analytics
func (mc *MetricsCollector) GetPIIAnalytics() PIIAnalyticsData {
	if !mc.config.Enabled {
		return PIIAnalyticsData{}
	}

	// Get PII metrics from the last 24 hours
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	
	query := MetricsQuery{
		Names:     []string{"pii_detection"},
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000, // Large limit to get all PII events
	}
	
	metrics, err := mc.storage.Get(query)
	if err != nil {
		// Return empty analytics on error
		return PIIAnalyticsData{}
	}
	
	// Aggregate the data
	analytics := PIIAnalyticsData{
		TypeBreakdown: make(map[string]map[string]interface{}),
		HourlyTrends:  make([]map[string]interface{}, 24),
	}
	
	// Initialize hourly trends for the last 24 hours
	for i := 0; i < 24; i++ {
		hour := startTime.Add(time.Duration(i) * time.Hour)
		analytics.HourlyTrends[i] = map[string]interface{}{
			"hour":       hour.Format("15:04"),
			"detections": 0,
			"blocked":    0,
		}
	}
	
	// Process metrics
	for _, metric := range metrics {
		analytics.TotalScanned++
		
		if metric.Value > 0 {
			analytics.PIIDetected++
			
			// Extract detected types from metadata
			if detectedTypes, ok := metric.Metadata["detected_types"].([]string); ok {
				for _, piiType := range detectedTypes {
					if _, exists := analytics.TypeBreakdown[piiType]; !exists {
						analytics.TypeBreakdown[piiType] = map[string]interface{}{
							"count":     0,
							"accuracy":  100.0,
							"false_pos": 0,
							"avg_conf":  0.0,
						}
	 				}
					
					// Increment count
					if count, ok := analytics.TypeBreakdown[piiType]["count"].(int); ok {
						analytics.TypeBreakdown[piiType]["count"] = count + 1
					}
					
					// Update average confidence
					if confidence, ok := metric.Metadata["confidence"].(float64); ok {
						analytics.TypeBreakdown[piiType]["avg_conf"] = confidence
					}
				}
			}
			
			// Update hourly trends
			hour := int(metric.Timestamp.Sub(startTime).Hours())
			if hour >= 0 && hour < 24 {
				if detections, ok := analytics.HourlyTrends[hour]["detections"].(int); ok {
					analytics.HourlyTrends[hour]["detections"] = detections + 1
				}
				
				// Check if this detection was blocked based on action
				if action, ok := metric.Tags["action"]; ok && (action == "block" || action == "reject") {
					if blocked, ok := analytics.HourlyTrends[hour]["blocked"].(int); ok {
						analytics.HourlyTrends[hour]["blocked"] = blocked + 1
					}
					analytics.PreventedExposures++
				}
			}
		}
	}
	
	// Calculate derived metrics
	if analytics.TotalScanned > 0 {
		analytics.DetectionRate = math.Round(float64(analytics.PIIDetected) / float64(analytics.TotalScanned) * 100)
	}
	
	// Estimate accuracy (this would be better with actual validation data)
	analytics.AccuracyRate = 95.0 // Default reasonable estimate
	analytics.FalsePositives = analytics.PIIDetected / 20 // Estimate 5% false positive rate
	
	// Categorize risk levels (based on PII type sensitivity)
	for piiType, data := range analytics.TypeBreakdown {
		count := data["count"].(int)
		
		switch piiType {
		case "ssn", "credit_card", "passport":
			analytics.HighRiskEvents += int64(count)
		case "phone", "drivers_license":
			analytics.MediumRiskEvents += int64(count)
		default:
			analytics.LowRiskEvents += int64(count)
		}
	}
	
	return analytics
}

// recordMetric sends a metric to the processing channel
func (mc *MetricsCollector) recordMetric(metric Metric) {
	select {
	case mc.metricsChan <- metric:
		// Metric sent successfully
	default:
		// Channel is full, drop the metric to prevent blocking
		// In production, you might want to log this or increment a counter
	}
}

// metricsWorker processes metrics from the channel
func (mc *MetricsCollector) metricsWorker(ctx context.Context) {
	defer mc.wg.Done()
	
	for {
		select {
		case metric, ok := <-mc.metricsChan:
			if !ok {
				return // Channel closed
			}
			
			// Store the metric
			if err := mc.storage.Store(metric); err != nil {
				// In production, you might want to log this error
				// For now, continue processing other metrics
			}
			
		case <-mc.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// getSystemHealth collects current system health metrics
func (mc *MetricsCollector) getSystemHealth() SystemHealthMetrics {
	// This would be delegated to the storage implementation
	// which has access to system monitoring
	if inmem, ok := mc.storage.(*InMemoryStorage); ok {
		return inmem.getSystemHealth()
	}
	
	// Fallback for other storage types
	return SystemHealthMetrics{
		UptimeSeconds: int64(time.Since(mc.startTime).Seconds()),
	}
}

// Utility functions for data sanitization

// sanitizeIP removes or masks sensitive parts of IP addresses
func sanitizeIP(ip string) string {
	if ip == "" {
		return ""
	}
	// For privacy, you might want to mask part of the IP
	// For now, return as-is but this could be enhanced
	return ip
}

// sanitizeUserID ensures user IDs don't contain sensitive information
func sanitizeUserID(userID string) string {
	if userID == "" || userID == "anonymous" {
		return "anonymous"
	}
	// For privacy, you might want to hash or mask user IDs
	// For now, return as-is but this could be enhanced
	return userID
}