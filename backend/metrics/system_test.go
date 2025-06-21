package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewSystemMetricsCollector(t *testing.T) {
	// Create a basic metrics collector for testing
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}

	// Create system metrics config
	systemConfig := SystemMetricsConfig{
		Enabled:            true,
		CollectionInterval: 1 * time.Second,
		CPUMonitoring:      true,
		MemoryMonitoring:   true,
		ProviderHealth:     []string{"openai", "anthropic"},
	}

	// Create system metrics collector
	smc := NewSystemMetricsCollector(collector, systemConfig)

	// Verify initialization
	if smc == nil {
		t.Fatal("System metrics collector should not be nil")
	}

	if smc.config.CollectionInterval != 1*time.Second {
		t.Errorf("Expected collection interval to be 1s, got %v", smc.config.CollectionInterval)
	}

	if !smc.config.Enabled {
		t.Error("Expected system metrics to be enabled")
	}

	if len(smc.providerHealth) != 2 {
		t.Errorf("Expected 2 provider health trackers, got %d", len(smc.providerHealth))
	}

	// Verify provider health trackers were created
	if _, exists := smc.providerHealth["openai"]; !exists {
		t.Error("Expected openai provider health tracker to exist")
	}

	if _, exists := smc.providerHealth["anthropic"]; !exists {
		t.Error("Expected anthropic provider health tracker to exist")
	}
}

func TestSystemMetricsCollector_StartStop(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}

	systemConfig := SystemMetricsConfig{
		Enabled:            true,
		CollectionInterval: 100 * time.Millisecond, // Fast for testing
		CPUMonitoring:      true,
		MemoryMonitoring:   true,
	}

	smc := NewSystemMetricsCollector(collector, systemConfig)

	// Test starting
	ctx := context.Background()
	err = smc.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start system metrics collector: %v", err)
	}

	if !smc.isRunning {
		t.Error("Expected system metrics collector to be running")
	}

	// Test double start (should not error)
	err = smc.Start(ctx)
	if err != nil {
		t.Error("Double start should not return error")
	}

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Test stopping
	err = smc.Stop()
	if err != nil {
		t.Fatalf("Failed to stop system metrics collector: %v", err)
	}

	if smc.isRunning {
		t.Error("Expected system metrics collector to not be running")
	}

	// Test double stop (should not error)
	err = smc.Stop()
	if err != nil {
		t.Error("Double stop should not return error")
	}
}

func TestSystemMetricsCollector_GetSystemHealthSnapshot(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}

	systemConfig := SystemMetricsConfig{
		Enabled:            true,
		CPUMonitoring:      true,
		MemoryMonitoring:   true,
		ProviderHealth:     []string{"test-provider"},
	}

	smc := NewSystemMetricsCollector(collector, systemConfig)

	// Get snapshot
	snapshot := smc.GetSystemHealthSnapshot()

	// Verify snapshot structure
	if snapshot.Timestamp.IsZero() {
		t.Error("Expected snapshot timestamp to be set")
	}

	if snapshot.GoroutineCount <= 0 {
		t.Error("Expected goroutine count to be positive")
	}

	if snapshot.CPUUsage < 0 || snapshot.CPUUsage > 100 {
		t.Errorf("Expected CPU usage to be between 0-100, got %f", snapshot.CPUUsage)
	}

	if snapshot.MemoryUsage < 0 || snapshot.MemoryUsage > 100 {
		t.Errorf("Expected memory usage to be between 0-100, got %f", snapshot.MemoryUsage)
	}

	if snapshot.MemoryAllocated == 0 {
		t.Error("Expected allocated memory to be greater than 0")
	}

	if snapshot.MemorySystem == 0 {
		t.Error("Expected system memory to be greater than 0")
	}

	// Verify provider health is included
	if len(snapshot.ProviderHealth) != 1 {
		t.Errorf("Expected 1 provider in health snapshot, got %d", len(snapshot.ProviderHealth))
	}

	if _, exists := snapshot.ProviderHealth["test-provider"]; !exists {
		t.Error("Expected test-provider to be in health snapshot")
	}
}

func TestProviderHealthTracker_RecordRequest(t *testing.T) {
	tracker := &ProviderHealthTracker{
		Name:          "test-provider",
		ResponseTimes: make([]float64, 0, 100),
		IsHealthy:     true,
		HealthScore:   1.0,
	}

	// Record successful request
	tracker.RecordRequest(100*time.Millisecond, false)

	if tracker.RequestCount != 1 {
		t.Errorf("Expected request count to be 1, got %d", tracker.RequestCount)
	}

	if tracker.ErrorCount != 0 {
		t.Errorf("Expected error count to be 0, got %d", tracker.ErrorCount)
	}

	if len(tracker.ResponseTimes) != 1 {
		t.Errorf("Expected 1 response time recorded, got %d", len(tracker.ResponseTimes))
	}

	if tracker.ResponseTimes[0] != 100 {
		t.Errorf("Expected response time to be 100ms, got %f", tracker.ResponseTimes[0])
	}

	if !tracker.IsHealthy {
		t.Error("Expected provider to be healthy after successful request")
	}

	// Record error request
	tracker.RecordRequest(500*time.Millisecond, true)

	if tracker.RequestCount != 2 {
		t.Errorf("Expected request count to be 2, got %d", tracker.RequestCount)
	}

	if tracker.ErrorCount != 1 {
		t.Errorf("Expected error count to be 1, got %d", tracker.ErrorCount)
	}

	if len(tracker.ResponseTimes) != 2 {
		t.Errorf("Expected 2 response times recorded, got %d", len(tracker.ResponseTimes))
	}
}

func TestProviderHealthTracker_HealthScore(t *testing.T) {
	tracker := &ProviderHealthTracker{
		Name:          "test-provider",
		ResponseTimes: make([]float64, 0, 100),
		IsHealthy:     true,
		HealthScore:   1.0,
	}

	// Record successful requests with good response times
	for i := 0; i < 10; i++ {
		tracker.RecordRequest(50*time.Millisecond, false)
	}

	health := tracker.GetHealth()

	if health.HealthScore < 0.9 {
		t.Errorf("Expected high health score for good requests, got %f", health.HealthScore)
	}

	if !health.IsHealthy {
		t.Error("Expected provider to be healthy with good performance")
	}

	if health.ErrorRate != 0 {
		t.Errorf("Expected error rate to be 0, got %f", health.ErrorRate)
	}

	// Record many errors
	for i := 0; i < 20; i++ {
		tracker.RecordRequest(100*time.Millisecond, true)
	}

	health = tracker.GetHealth()

	if health.HealthScore > 0.6 {
		t.Errorf("Expected low health score with many errors, got %f", health.HealthScore)
	}

	if health.IsHealthy {
		t.Error("Expected provider to be unhealthy with high error rate")
	}

	if health.ErrorRate < 0.5 {
		t.Errorf("Expected high error rate, got %f", health.ErrorRate)
	}
}

func TestProviderHealthTracker_ResponseTimeLimit(t *testing.T) {
	tracker := &ProviderHealthTracker{
		Name:          "test-provider",
		ResponseTimes: make([]float64, 0, 100),
		IsHealthy:     true,
		HealthScore:   1.0,
	}

	// Record more than 100 response times
	for i := 0; i < 150; i++ {
		tracker.RecordRequest(time.Duration(i+1)*time.Millisecond, false)
	}

	// Should only keep last 100
	if len(tracker.ResponseTimes) != 100 {
		t.Errorf("Expected response times to be limited to 100, got %d", len(tracker.ResponseTimes))
	}

	// Should have the most recent times (51-150 ms)
	if tracker.ResponseTimes[0] != 51 {
		t.Errorf("Expected first response time to be 51ms, got %f", tracker.ResponseTimes[0])
	}

	if tracker.ResponseTimes[99] != 150 {
		t.Errorf("Expected last response time to be 150ms, got %f", tracker.ResponseTimes[99])
	}
}

func TestSystemMetricsCollector_RecordProviderRequest(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}

	systemConfig := SystemMetricsConfig{
		Enabled:        true,
		ProviderHealth: []string{"existing-provider"},
	}

	smc := NewSystemMetricsCollector(collector, systemConfig)

	// Record request for existing provider
	smc.RecordProviderRequest("existing-provider", 100*time.Millisecond, false)

	tracker, exists := smc.providerHealth["existing-provider"]
	if !exists {
		t.Fatal("Expected existing provider tracker to exist")
	}

	if tracker.RequestCount != 1 {
		t.Errorf("Expected 1 request recorded, got %d", tracker.RequestCount)
	}

	// Record request for new provider (should be created automatically)
	smc.RecordProviderRequest("new-provider", 200*time.Millisecond, true)

	tracker, exists = smc.providerHealth["new-provider"]
	if !exists {
		t.Fatal("Expected new provider tracker to be created")
	}

	if tracker.RequestCount != 1 {
		t.Errorf("Expected 1 request recorded for new provider, got %d", tracker.RequestCount)
	}

	if tracker.ErrorCount != 1 {
		t.Errorf("Expected 1 error recorded for new provider, got %d", tracker.ErrorCount)
	}
}

func TestSystemMetricsCollector_DisabledCollection(t *testing.T) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}

	// Disable system metrics
	systemConfig := SystemMetricsConfig{
		Enabled: false,
	}

	smc := NewSystemMetricsCollector(collector, systemConfig)

	// Start the collector
	ctx := context.Background()
	err = smc.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start system metrics collector: %v", err)
	}

	// Should still be able to get snapshots, but collection should be disabled
	snapshot := smc.GetSystemHealthSnapshot()
	if snapshot.Timestamp.IsZero() {
		t.Error("Expected snapshot timestamp to be set even when disabled")
	}

	// Stop the collector
	err = smc.Stop()
	if err != nil {
		t.Fatalf("Failed to stop system metrics collector: %v", err)
	}
}

// Benchmark tests for performance
func BenchmarkSystemMetricsCollector_GetSnapshot(b *testing.B) {
	config := DefaultMetricsConfig()
	collector, err := NewMetricsCollector(config)
	if err != nil {
		b.Fatalf("Failed to create metrics collector: %v", err)
	}

	systemConfig := SystemMetricsConfig{
		Enabled:            true,
		CPUMonitoring:      true,
		MemoryMonitoring:   true,
		ProviderHealth:     []string{"provider1", "provider2", "provider3"},
	}

	smc := NewSystemMetricsCollector(collector, systemConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = smc.GetSystemHealthSnapshot()
	}
}

func BenchmarkProviderHealthTracker_RecordRequest(b *testing.B) {
	tracker := &ProviderHealthTracker{
		Name:          "test-provider",
		ResponseTimes: make([]float64, 0, 100),
		IsHealthy:     true,
		HealthScore:   1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.RecordRequest(100*time.Millisecond, false)
	}
}