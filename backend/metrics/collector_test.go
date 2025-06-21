package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}
	
	if collector == nil {
		t.Fatal("Collector is nil")
	}
	
	if collector.config.Enabled != config.Enabled {
		t.Errorf("Expected enabled %v, got %v", config.Enabled, collector.config.Enabled)
	}
}

func TestMetricsCollectorStartStop(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	
	ctx := context.Background()
	
	// Start the collector
	err = collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	
	// Verify it's running
	if !collector.isRunning {
		t.Error("Collector should be running")
	}
	
	// Stop the collector
	err = collector.Stop()
	if err != nil {
		t.Fatalf("Failed to stop collector: %v", err)
	}
	
	// Verify it's stopped
	if collector.isRunning {
		t.Error("Collector should be stopped")
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	
	ctx := context.Background()
	collector.Start(ctx)
	defer collector.Stop()
	
	// Record HTTP request
	httpMetrics := HTTPMetrics{
		Path:       "/api/test",
		Method:     "GET",
		StatusCode: 200,
		Duration:   50 * time.Millisecond,
		UserAgent:  "test-agent",
		IPAddress:  "127.0.0.1",
	}
	
	collector.RecordHTTPRequest(httpMetrics)
	
	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)
	
	// Query metrics
	query := MetricsQuery{
		Names:     []string{"http_request"},
		StartTime: time.Now().Add(-time.Minute),
		EndTime:   time.Now(),
	}
	
	metrics, err := collector.GetMetrics(query)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	
	if len(metrics) == 0 {
		t.Error("Expected at least one metric")
	}
	
	metric := metrics[0]
	if metric.Name != "http_request" {
		t.Errorf("Expected metric name 'http_request', got '%s'", metric.Name)
	}
	
	if metric.Tags["path"] != "/api/test" {
		t.Errorf("Expected path tag '/api/test', got '%s'", metric.Tags["path"])
	}
	
	if metric.Tags["method"] != "GET" {
		t.Errorf("Expected method tag 'GET', got '%s'", metric.Tags["method"])
	}
}

func TestRecordModerationEvent(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	
	ctx := context.Background()
	collector.Start(ctx)
	defer collector.Stop()
	
	// Record moderation event
	modMetrics := ModerationMetrics{
		LayerName:     "regex",
		Score:         0.8,
		Confidence:    0.9,
		Blocked:       true,
		Category:      "harmful",
		Action:        "block",
		Duration:      10 * time.Millisecond,
		UserID:        "test_user",
		ContentLength: 100,
	}
	
	collector.RecordModerationEvent(modMetrics)
	
	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)
	
	// Query metrics
	query := MetricsQuery{
		Names:     []string{"moderation_event"},
		StartTime: time.Now().Add(-time.Minute),
		EndTime:   time.Now(),
	}
	
	metrics, err := collector.GetMetrics(query)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	
	if len(metrics) == 0 {
		t.Error("Expected at least one metric")
	}
	
	metric := metrics[0]
	if metric.Name != "moderation_event" {
		t.Errorf("Expected metric name 'moderation_event', got '%s'", metric.Name)
	}
	
	if metric.Tags["layer"] != "regex" {
		t.Errorf("Expected layer tag 'regex', got '%s'", metric.Tags["layer"])
	}
	
	if metric.Value != 0.8 {
		t.Errorf("Expected metric value 0.8, got %f", metric.Value)
	}
}

func TestRecordPIIDetection(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	
	ctx := context.Background()
	collector.Start(ctx)
	defer collector.Stop()
	
	// Record PII detection
	piiMetrics := PIIMetrics{
		DetectedTypes:  []string{"email", "phone"},
		MatchCount:     2,
		MaskingEnabled: true,
		Confidence:     0.95,
		Action:         "block",
		UserID:         "test_user",
		ContentLength:  200,
	}
	
	collector.RecordPIIDetection(piiMetrics)
	
	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)
	
	// Query metrics
	query := MetricsQuery{
		Names:     []string{"pii_detection"},
		StartTime: time.Now().Add(-time.Minute),
		EndTime:   time.Now(),
	}
	
	metrics, err := collector.GetMetrics(query)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	
	if len(metrics) == 0 {
		t.Error("Expected at least one metric")
	}
	
	metric := metrics[0]
	if metric.Name != "pii_detection" {
		t.Errorf("Expected metric name 'pii_detection', got '%s'", metric.Name)
	}
	
	if metric.Value != 2 {
		t.Errorf("Expected metric value 2, got %f", metric.Value)
	}
}

func TestConcurrentMetricRecording(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	
	ctx := context.Background()
	collector.Start(ctx)
	defer collector.Stop()
	
	// Record metrics concurrently
	const numGoroutines = 10
	const metricsPerGoroutine = 100
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < metricsPerGoroutine; j++ {
				httpMetrics := HTTPMetrics{
					Path:       "/api/test",
					Method:     "GET",
					StatusCode: 200,
					Duration:   time.Duration(j) * time.Millisecond,
				}
				collector.RecordHTTPRequest(httpMetrics)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Give some time for async processing
	time.Sleep(500 * time.Millisecond)
	
	// Verify metrics were recorded
	query := MetricsQuery{
		Names:     []string{"http_request"},
		StartTime: time.Now().Add(-time.Minute),
		EndTime:   time.Now(),
	}
	
	metrics, err := collector.GetMetrics(query)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	
	expectedCount := numGoroutines * metricsPerGoroutine
	if len(metrics) != expectedCount {
		t.Errorf("Expected %d metrics, got %d", expectedCount, len(metrics))
	}
}

func TestGetSummary(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create collector: %v", err)
	}
	
	ctx := context.Background()
	collector.Start(ctx)
	defer collector.Stop()
	
	// Record some test metrics
	for i := 0; i < 10; i++ {
		httpMetrics := HTTPMetrics{
			Path:       "/api/test",
			Method:     "GET",
			StatusCode: 200,
			Duration:   time.Duration(i*10) * time.Millisecond,
		}
		collector.RecordHTTPRequest(httpMetrics)
	}
	
	// Record some errors
	for i := 0; i < 2; i++ {
		httpMetrics := HTTPMetrics{
			Path:       "/api/error",
			Method:     "POST",
			StatusCode: 500,
			Duration:   100 * time.Millisecond,
		}
		collector.RecordHTTPRequest(httpMetrics)
	}
	
	// Give some time for async processing
	time.Sleep(200 * time.Millisecond)
	
	// Get summary
	timeRange := TimeRange{
		Start: time.Now().Add(-time.Hour),
		End:   time.Now(),
	}
	
	summary, err := collector.GetSummary(timeRange)
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}
	
	if summary.TotalRequests != 12 {
		t.Errorf("Expected 12 total requests, got %d", summary.TotalRequests)
	}
	
	expectedErrorRate := float64(2) / float64(12) * 100
	if summary.ErrorRate != expectedErrorRate {
		t.Errorf("Expected error rate %f, got %f", expectedErrorRate, summary.ErrorRate)
	}
	
	if len(summary.TopEndpoints) == 0 {
		t.Error("Expected top endpoints data")
	}
}