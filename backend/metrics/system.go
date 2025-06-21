package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// SystemMetricsCollector handles collection of system-level metrics
type SystemMetricsCollector struct {
	collector *MetricsCollector
	config    SystemMetricsConfig
	
	// Provider health tracking
	providerHealth   map[string]*ProviderHealthTracker
	providerMutex    sync.RWMutex
	
	// System monitoring
	cpuTracker       *CPUTracker
	memoryTracker    *MemoryTracker
	
	// Collection control
	stopChan         chan struct{}
	isRunning        bool
	mutex            sync.RWMutex
}


// ProviderHealthTracker tracks health metrics for AI providers
type ProviderHealthTracker struct {
	Name            string
	ResponseTimes   []float64
	ErrorCount      int64
	RequestCount    int64
	LastResponse    time.Time
	IsHealthy       bool
	HealthScore     float64
	mutex           sync.RWMutex
}

// CPUTracker monitors CPU usage over time
type CPUTracker struct {
	lastIdle    uint64
	lastTotal   uint64
	currentUsage float64
	mutex       sync.RWMutex
}

// MemoryTracker monitors memory usage
type MemoryTracker struct {
	stats runtime.MemStats
	mutex sync.RWMutex
}

// SystemHealthSnapshot represents a point-in-time system health view
type SystemHealthSnapshot struct {
	Timestamp       time.Time                    `json:"timestamp"`
	CPUUsage        float64                      `json:"cpu_usage"`
	MemoryUsage     float64                      `json:"memory_usage"`
	MemoryAllocated uint64                       `json:"memory_allocated"`
	MemorySystem    uint64                       `json:"memory_system"`
	GoroutineCount  int                          `json:"goroutine_count"`
	GCPauses        []time.Duration              `json:"gc_pauses"`
	ProviderHealth  map[string]ProviderHealth    `json:"provider_health"`
	UptimeSeconds   int64                        `json:"uptime_seconds"`
}

// ProviderHealth represents health status of an AI provider
type ProviderHealth struct {
	Name            string    `json:"name"`
	IsHealthy       bool      `json:"is_healthy"`
	HealthScore     float64   `json:"health_score"`
	AvgResponseTime float64   `json:"avg_response_time"`
	ErrorRate       float64   `json:"error_rate"`
	RequestCount    int64     `json:"request_count"`
	LastResponse    time.Time `json:"last_response"`
}

// NewSystemMetricsCollector creates a new system metrics collector
func NewSystemMetricsCollector(collector *MetricsCollector, config SystemMetricsConfig) *SystemMetricsCollector {
	if config.CollectionInterval == 0 {
		config.CollectionInterval = 30 * time.Second
	}
	
	smc := &SystemMetricsCollector{
		collector:      collector,
		config:         config,
		providerHealth: make(map[string]*ProviderHealthTracker),
		cpuTracker:     &CPUTracker{},
		memoryTracker:  &MemoryTracker{},
		stopChan:       make(chan struct{}),
	}
	
	// Initialize provider health trackers
	for _, provider := range config.ProviderHealth {
		smc.providerHealth[provider] = &ProviderHealthTracker{
			Name:         provider,
			ResponseTimes: make([]float64, 0, 100),
			IsHealthy:    true,
			HealthScore:  1.0,
		}
	}
	
	return smc
}

// Start begins system metrics collection
func (smc *SystemMetricsCollector) Start(ctx context.Context) error {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	
	if smc.isRunning {
		return nil
	}
	
	smc.isRunning = true
	
	// Start collection goroutine
	go smc.collectMetrics(ctx)
	
	return nil
}

// Stop gracefully shuts down system metrics collection
func (smc *SystemMetricsCollector) Stop() error {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	
	if !smc.isRunning {
		return nil
	}
	
	smc.isRunning = false
	close(smc.stopChan)
	
	return nil
}

// collectMetrics runs the main collection loop
func (smc *SystemMetricsCollector) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(smc.config.CollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			smc.collectSystemHealth()
			
		case <-smc.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// collectSystemHealth collects all system health metrics
func (smc *SystemMetricsCollector) collectSystemHealth() {
	if !smc.config.Enabled {
		return
	}
	
	snapshot := smc.GetSystemHealthSnapshot()
	
	// Record CPU metrics
	if smc.config.CPUMonitoring {
		smc.collector.RecordCustomMetric(
			"system_cpu_usage",
			snapshot.CPUUsage,
			MetricTypeGauge,
			map[string]string{"metric": "cpu_usage"},
		)
	}
	
	// Record memory metrics
	if smc.config.MemoryMonitoring {
		smc.collector.RecordCustomMetric(
			"system_memory_usage",
			snapshot.MemoryUsage,
			MetricTypeGauge,
			map[string]string{"metric": "memory_usage"},
		)
		
		smc.collector.RecordCustomMetric(
			"system_memory_allocated",
			float64(snapshot.MemoryAllocated),
			MetricTypeGauge,
			map[string]string{"metric": "memory_allocated_bytes"},
		)
		
		smc.collector.RecordCustomMetric(
			"system_memory_system",
			float64(snapshot.MemorySystem),
			MetricTypeGauge,
			map[string]string{"metric": "memory_system_bytes"},
		)
	}
	
	// Record goroutine count
	smc.collector.RecordCustomMetric(
		"system_goroutines",
		float64(snapshot.GoroutineCount),
		MetricTypeGauge,
		map[string]string{"metric": "goroutine_count"},
	)
	
	// Record provider health metrics
	for provider, health := range snapshot.ProviderHealth {
		smc.collector.RecordCustomMetric(
			"provider_health_score",
			health.HealthScore,
			MetricTypeGauge,
			map[string]string{
				"provider": provider,
				"metric":   "health_score",
			},
		)
		
		smc.collector.RecordCustomMetric(
			"provider_response_time",
			health.AvgResponseTime,
			MetricTypeGauge,
			map[string]string{
				"provider": provider,
				"metric":   "avg_response_time",
			},
		)
		
		smc.collector.RecordCustomMetric(
			"provider_error_rate",
			health.ErrorRate,
			MetricTypeGauge,
			map[string]string{
				"provider": provider,
				"metric":   "error_rate",
			},
		)
	}
}

// GetSystemHealthSnapshot returns current system health snapshot
func (smc *SystemMetricsCollector) GetSystemHealthSnapshot() SystemHealthSnapshot {
	snapshot := SystemHealthSnapshot{
		Timestamp:      time.Now(),
		GoroutineCount: runtime.NumGoroutine(),
		ProviderHealth: make(map[string]ProviderHealth),
	}
	
	// Get CPU usage
	if smc.config.CPUMonitoring {
		snapshot.CPUUsage = smc.getCPUUsage()
	}
	
	// Get memory statistics
	if smc.config.MemoryMonitoring {
		memStats := smc.getMemoryStats()
		snapshot.MemoryUsage = smc.calculateMemoryUsagePercent(memStats)
		snapshot.MemoryAllocated = memStats.Alloc
		snapshot.MemorySystem = memStats.Sys
		snapshot.GCPauses = smc.getRecentGCPauses(memStats)
	}
	
	// Get provider health
	smc.providerMutex.RLock()
	for name, tracker := range smc.providerHealth {
		snapshot.ProviderHealth[name] = tracker.GetHealth()
	}
	smc.providerMutex.RUnlock()
	
	return snapshot
}

// RecordProviderRequest records a request to an AI provider
func (smc *SystemMetricsCollector) RecordProviderRequest(provider string, responseTime time.Duration, isError bool) {
	smc.providerMutex.Lock()
	defer smc.providerMutex.Unlock()
	
	tracker, exists := smc.providerHealth[provider]
	if !exists {
		tracker = &ProviderHealthTracker{
			Name:          provider,
			ResponseTimes: make([]float64, 0, 100),
			IsHealthy:     true,
			HealthScore:   1.0,
		}
		smc.providerHealth[provider] = tracker
	}
	
	tracker.RecordRequest(responseTime, isError)
}

// RecordRequest records a request for the provider
func (pt *ProviderHealthTracker) RecordRequest(responseTime time.Duration, isError bool) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	
	pt.RequestCount++
	pt.LastResponse = time.Now()
	
	if isError {
		pt.ErrorCount++
	}
	
	// Keep last 100 response times
	responseTimeMs := float64(responseTime.Milliseconds())
	if len(pt.ResponseTimes) >= 100 {
		pt.ResponseTimes = pt.ResponseTimes[1:]
	}
	pt.ResponseTimes = append(pt.ResponseTimes, responseTimeMs)
	
	pt.updateHealthScore()
}

// updateHealthScore calculates health score based on recent performance
func (pt *ProviderHealthTracker) updateHealthScore() {
	if pt.RequestCount == 0 {
		pt.HealthScore = 1.0
		pt.IsHealthy = true
		return
	}
	
	errorRate := float64(pt.ErrorCount) / float64(pt.RequestCount)
	
	// Calculate average response time from recent requests
	avgResponseTime := 0.0
	if len(pt.ResponseTimes) > 0 {
		sum := 0.0
		for _, rt := range pt.ResponseTimes {
			sum += rt
		}
		avgResponseTime = sum / float64(len(pt.ResponseTimes))
	}
	
	// Health score based on error rate and response time
	// Lower error rate and faster response time = higher score
	errorPenalty := errorRate * 0.7  // Max penalty of 0.7 for 100% error rate
	
	// Response time penalty (assuming 1000ms as baseline)
	responsePenalty := 0.0
	if avgResponseTime > 1000 {
		responsePenalty = (avgResponseTime - 1000) / 10000 * 0.3 // Max penalty of 0.3
		if responsePenalty > 0.3 {
			responsePenalty = 0.3
		}
	}
	
	pt.HealthScore = 1.0 - errorPenalty - responsePenalty
	if pt.HealthScore < 0 {
		pt.HealthScore = 0
	}
	
	pt.IsHealthy = pt.HealthScore > 0.7 && errorRate < 0.1
}

// GetHealth returns current health status
func (pt *ProviderHealthTracker) GetHealth() ProviderHealth {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()
	
	avgResponseTime := 0.0
	if len(pt.ResponseTimes) > 0 {
		sum := 0.0
		for _, rt := range pt.ResponseTimes {
			sum += rt
		}
		avgResponseTime = sum / float64(len(pt.ResponseTimes))
	}
	
	errorRate := 0.0
	if pt.RequestCount > 0 {
		errorRate = float64(pt.ErrorCount) / float64(pt.RequestCount)
	}
	
	return ProviderHealth{
		Name:            pt.Name,
		IsHealthy:       pt.IsHealthy,
		HealthScore:     pt.HealthScore,
		AvgResponseTime: avgResponseTime,
		ErrorRate:       errorRate,
		RequestCount:    pt.RequestCount,
		LastResponse:    pt.LastResponse,
	}
}

// getCPUUsage returns current CPU usage percentage
func (smc *SystemMetricsCollector) getCPUUsage() float64 {
	// Note: This is a simplified CPU usage calculation
	// In production, you might want to use a more sophisticated approach
	// or integrate with system-specific libraries
	
	// For now, we'll use a placeholder that simulates CPU usage
	// based on goroutine count and GC activity
	numGoroutines := runtime.NumGoroutine()
	
	// Simple heuristic: more goroutines typically means higher CPU usage
	usage := float64(numGoroutines) / 1000.0 * 100
	if usage > 100 {
		usage = 100
	}
	
	return usage
}

// getMemoryStats returns current memory statistics
func (smc *SystemMetricsCollector) getMemoryStats() *runtime.MemStats {
	smc.memoryTracker.mutex.Lock()
	defer smc.memoryTracker.mutex.Unlock()
	
	runtime.ReadMemStats(&smc.memoryTracker.stats)
	return &smc.memoryTracker.stats
}

// calculateMemoryUsagePercent calculates memory usage percentage
func (smc *SystemMetricsCollector) calculateMemoryUsagePercent(stats *runtime.MemStats) float64 {
	// Calculate percentage based on heap in use vs system memory
	if stats.Sys == 0 {
		return 0
	}
	return (float64(stats.Alloc) / float64(stats.Sys)) * 100
}

// getRecentGCPauses returns recent GC pause times
func (smc *SystemMetricsCollector) getRecentGCPauses(stats *runtime.MemStats) []time.Duration {
	pauses := make([]time.Duration, 0, 10)
	
	// Get last 10 GC pauses
	end := int(stats.NumGC)
	start := end - 10
	if start < 0 {
		start = 0
	}
	
	for i := start; i < end; i++ {
		idx := i % len(stats.PauseNs)
		pauses = append(pauses, time.Duration(stats.PauseNs[idx]))
	}
	
	return pauses
}