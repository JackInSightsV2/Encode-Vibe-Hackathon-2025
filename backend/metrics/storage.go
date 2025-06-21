package metrics

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
)

// MetricsStorage defines the interface for metric storage backends
type MetricsStorage interface {
	Store(metric Metric) error
	Get(query MetricsQuery) ([]Metric, error)
	GetSummary(timeRange TimeRange) (*MetricsSummary, error)
	Cleanup() error
	Close() error
}

// InMemoryStorage provides in-memory storage for metrics
type InMemoryStorage struct {
	metrics   []Metric
	mutex     sync.RWMutex
	config    StorageConfig
	startTime time.Time
	lastCleanup time.Time
}

// NewInMemoryStorage creates a new in-memory storage instance
func NewInMemoryStorage(config StorageConfig) *InMemoryStorage {
	storage := &InMemoryStorage{
		metrics:     make([]Metric, 0),
		config:      config,
		startTime:   time.Now(),
		lastCleanup: time.Now(),
	}
	
	// Start cleanup routine
	go storage.cleanupRoutine()
	
	return storage
}

// Store adds a metric to in-memory storage
func (s *InMemoryStorage) Store(metric Metric) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check memory usage before adding
	if err := s.checkMemoryUsage(); err != nil {
		return err
	}
	
	s.metrics = append(s.metrics, metric)
	return nil
}

// Get retrieves metrics based on query parameters
func (s *InMemoryStorage) Get(query MetricsQuery) ([]Metric, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	var results []Metric
	
	for _, metric := range s.metrics {
		if s.matchesQuery(metric, query) {
			results = append(results, metric)
		}
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	
	// Apply limit and offset
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}
	
	return results, nil
}

// GetSummary generates aggregated metrics summary
func (s *InMemoryStorage) GetSummary(timeRange TimeRange) (*MetricsSummary, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	summary := &MetricsSummary{
		TimeRange: timeRange,
	}
	
	var (
		totalRequests     int64
		totalResponseTime float64
		errorCount        int64
		moderationBlocked int64
		piiDetections     int64
		endpointMap       = make(map[string]*EndpointMetrics)
	)
	
	for _, metric := range s.metrics {
		if metric.Timestamp.Before(timeRange.Start) || metric.Timestamp.After(timeRange.End) {
			continue
		}
		
		switch metric.Name {
		case "http_request":
			totalRequests++
			if duration, ok := metric.Metadata["duration"].(float64); ok {
				totalResponseTime += duration
			}
			if statusCode, ok := metric.Metadata["status_code"].(int); ok && statusCode >= 400 {
				errorCount++
			}
			
			// Track endpoint metrics
			if path, ok := metric.Tags["path"]; ok {
				if method, ok := metric.Tags["method"]; ok {
					key := fmt.Sprintf("%s %s", method, path)
					if ep, exists := endpointMap[key]; exists {
						ep.RequestCount++
						if duration, ok := metric.Metadata["duration"].(float64); ok {
							ep.AvgDuration = (ep.AvgDuration*float64(ep.RequestCount-1) + duration) / float64(ep.RequestCount)
						}
						if statusCode, ok := metric.Metadata["status_code"].(int); ok && statusCode >= 400 {
							ep.ErrorCount++
						}
					} else {
						ep := &EndpointMetrics{
							Path:         path,
							Method:       method,
							RequestCount: 1,
							ErrorCount:   0,
						}
						if duration, ok := metric.Metadata["duration"].(float64); ok {
							ep.AvgDuration = duration
						}
						if statusCode, ok := metric.Metadata["status_code"].(int); ok && statusCode >= 400 {
							ep.ErrorCount++
						}
						endpointMap[key] = ep
					}
				}
			}
			
		case "moderation_event":
			if blocked, ok := metric.Metadata["blocked"].(bool); ok && blocked {
				moderationBlocked++
			}
			
		case "pii_detection":
			piiDetections++
		}
	}
	
	// Calculate summary statistics
	if totalRequests > 0 {
		summary.TotalRequests = totalRequests
		summary.AverageResponseTime = totalResponseTime / float64(totalRequests)
		summary.ErrorRate = float64(errorCount) / float64(totalRequests) * 100
		
		duration := timeRange.End.Sub(timeRange.Start).Seconds()
		if duration > 0 {
			summary.RequestsPerSecond = float64(totalRequests) / duration
		}
	}
	
	summary.ModerationBlocked = moderationBlocked
	summary.PIIDetections = piiDetections
	
	// Get system health
	summary.SystemHealth = s.getSystemHealth()
	
	// Get top endpoints (limit to top 10)
	var endpoints []EndpointMetrics
	for _, ep := range endpointMap {
		if ep.RequestCount > 0 {
			ep.ErrorRate = float64(ep.ErrorCount) / float64(ep.RequestCount) * 100
		}
		endpoints = append(endpoints, *ep)
	}
	
	// Sort by request count
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].RequestCount > endpoints[j].RequestCount
	})
	
	if len(endpoints) > 10 {
		endpoints = endpoints[:10]
	}
	summary.TopEndpoints = endpoints
	
	return summary, nil
}

// Cleanup removes old metrics based on retention policy
func (s *InMemoryStorage) Cleanup() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.config.RetentionHours <= 0 {
		return nil
	}
	
	cutoff := time.Now().Add(-time.Duration(s.config.RetentionHours) * time.Hour)
	
	// Filter out old metrics
	filtered := make([]Metric, 0, len(s.metrics))
	for _, metric := range s.metrics {
		if metric.Timestamp.After(cutoff) {
			filtered = append(filtered, metric)
		}
	}
	
	s.metrics = filtered
	s.lastCleanup = time.Now()
	
	return nil
}

// Close cleans up resources
func (s *InMemoryStorage) Close() error {
	return nil
}

// matchesQuery checks if a metric matches the query criteria
func (s *InMemoryStorage) matchesQuery(metric Metric, query MetricsQuery) bool {
	// Check timestamp range
	if !metric.Timestamp.IsZero() {
		if !query.StartTime.IsZero() && metric.Timestamp.Before(query.StartTime) {
			return false
		}
		if !query.EndTime.IsZero() && metric.Timestamp.After(query.EndTime) {
			return false
		}
	}
	
	// Check names filter
	if len(query.Names) > 0 {
		found := false
		for _, name := range query.Names {
			if metric.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check tags filter
	if len(query.Tags) > 0 {
		for key, value := range query.Tags {
			if metricValue, exists := metric.Tags[key]; !exists || metricValue != value {
				return false
			}
		}
	}
	
	return true
}

// checkMemoryUsage ensures we don't exceed memory limits
func (s *InMemoryStorage) checkMemoryUsage() error {
	if s.config.MaxMemoryMB <= 0 {
		return nil
	}
	
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	
	currentMB := int(m.Alloc / 1024 / 1024)
	if currentMB > s.config.MaxMemoryMB {
		// Force cleanup and check again
		s.Cleanup()
		runtime.GC()
		runtime.ReadMemStats(&m)
		currentMB = int(m.Alloc / 1024 / 1024)
		
		if currentMB > s.config.MaxMemoryMB {
			return fmt.Errorf("memory usage (%dMB) exceeds limit (%dMB)", currentMB, s.config.MaxMemoryMB)
		}
	}
	
	return nil
}

// getSystemHealth collects current system health metrics
func (s *InMemoryStorage) getSystemHealth() SystemHealthMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return SystemHealthMetrics{
		MemoryUsage:    float64(m.Alloc) / 1024 / 1024, // MB
		GoroutineCount: runtime.NumGoroutine(),
		UptimeSeconds:  int64(time.Since(s.startTime).Seconds()),
		// CPU usage would require additional system monitoring
		CPUUsage: 0, // TODO: Implement CPU monitoring
	}
}

// cleanupRoutine runs periodic cleanup
func (s *InMemoryStorage) cleanupRoutine() {
	if s.config.CleanupIntervalMinutes <= 0 {
		return
	}
	
	ticker := time.NewTicker(time.Duration(s.config.CleanupIntervalMinutes) * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.Cleanup()
	}
}

// SupabaseStorage provides Supabase database storage for metrics
type SupabaseStorage struct {
	config SupabaseConfig
	client *SupabaseClient
}

// SupabaseClient represents a Supabase client (placeholder for actual implementation)
type SupabaseClient struct {
	URL string
	Key string
}

// NewSupabaseStorage creates a new Supabase storage instance
func NewSupabaseStorage(config SupabaseConfig) (*SupabaseStorage, error) {
	if config.URL == "" || config.Key == "" {
		return nil, fmt.Errorf("supabase URL and key are required")
	}
	
	client := &SupabaseClient{
		URL: config.URL,
		Key: config.Key,
	}
	
	return &SupabaseStorage{
		config: config,
		client: client,
	}, nil
}

// Store saves a metric to Supabase (placeholder implementation)
func (s *SupabaseStorage) Store(metric Metric) error {
	// TODO: Implement actual Supabase storage
	// This would use the Supabase Go client to insert the metric
	
	// For now, return success (this would be implemented with actual Supabase SDK)
	data, _ := json.Marshal(metric)
	_ = data // Placeholder to avoid unused variable error
	
	return nil
}

// Get retrieves metrics from Supabase (placeholder implementation)
func (s *SupabaseStorage) Get(query MetricsQuery) ([]Metric, error) {
	// TODO: Implement actual Supabase query
	return []Metric{}, nil
}

// GetSummary generates summary from Supabase data (placeholder implementation)
func (s *SupabaseStorage) GetSummary(timeRange TimeRange) (*MetricsSummary, error) {
	// TODO: Implement actual Supabase aggregation
	return &MetricsSummary{
		TimeRange: timeRange,
	}, nil
}

// Cleanup removes old metrics from Supabase (placeholder implementation)
func (s *SupabaseStorage) Cleanup() error {
	// TODO: Implement actual Supabase cleanup
	return nil
}

// Close closes the Supabase connection
func (s *SupabaseStorage) Close() error {
	// TODO: Implement actual connection cleanup
	return nil
}