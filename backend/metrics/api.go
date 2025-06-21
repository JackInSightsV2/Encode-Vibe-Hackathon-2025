package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// APIHandler provides HTTP endpoints for metrics
type APIHandler struct {
	collector *MetricsCollector
	timeSeries *TimeSeriesStorage
	systemMetrics *SystemMetricsCollector
}

// NewAPIHandler creates a new metrics API handler
func NewAPIHandler(collector *MetricsCollector) *APIHandler {
	return &APIHandler{
		collector: collector,
	}
}

// NewAPIHandlerWithTimeSeries creates a new metrics API handler with time-series support
func NewAPIHandlerWithTimeSeries(collector *MetricsCollector, timeSeries *TimeSeriesStorage) *APIHandler {
	return &APIHandler{
		collector:  collector,
		timeSeries: timeSeries,
	}
}

// NewAPIHandlerWithSystemMetrics creates a new metrics API handler with system metrics support
func NewAPIHandlerWithSystemMetrics(collector *MetricsCollector, timeSeries *TimeSeriesStorage, systemMetrics *SystemMetricsCollector) *APIHandler {
	return &APIHandler{
		collector:     collector,
		timeSeries:    timeSeries,
		systemMetrics: systemMetrics,
	}
}

// RegisterRoutes registers metrics API routes with a mux
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/metrics", h.HandleMetrics)
	mux.HandleFunc("/api/metrics/summary", h.HandleSummary)
	mux.HandleFunc("/api/metrics/health", h.HandleHealth)
	mux.HandleFunc("/api/metrics/stats", h.HandleStats)
	
	// Time-series endpoints (if time-series storage is available)
	if h.timeSeries != nil {
		mux.HandleFunc("/api/metrics/timeseries", h.HandleTimeSeries)
		mux.HandleFunc("/api/metrics/aggregator/status", h.HandleAggregatorStatus)
		mux.HandleFunc("/api/metrics/aggregator/force", h.HandleForceAggregation)
	}
	
	// System health endpoints (if system metrics collector is available)
	if h.systemMetrics != nil {
		mux.HandleFunc("/api/metrics/system", h.HandleSystemMetrics)
		mux.HandleFunc("/api/metrics/system/snapshot", h.HandleSystemSnapshot)
		mux.HandleFunc("/api/metrics/providers", h.HandleProviderHealth)
	}
}

// HandleMetrics returns metrics based on query parameters
func (h *APIHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse query parameters
	query, err := h.parseMetricsQuery(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query parameters: %v", err), http.StatusBadRequest)
		return
	}
	
	// Get metrics from collector
	metrics, err := h.collector.GetMetrics(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve metrics: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    metrics,
		"count":   len(metrics),
		"query":   query,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleSummary returns aggregated metrics summary
func (h *APIHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse time range from query parameters
	timeRange, err := h.parseTimeRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid time range: %v", err), http.StatusBadRequest)
		return
	}
	
	// Get summary from collector
	summary, err := h.collector.GetSummary(timeRange)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate summary: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    summary,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleHealth returns current system health metrics
func (h *APIHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Trigger system health collection
	h.collector.RecordSystemHealth()
	
	// Get recent system health metrics
	query := MetricsQuery{
		Names:     []string{"system_memory", "system_goroutines", "system_uptime"},
		StartTime: time.Now().Add(-5 * time.Minute),
		EndTime:   time.Now(),
		Limit:     10,
	}
	
	metrics, err := h.collector.GetMetrics(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve health metrics: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Organize metrics by type
	health := make(map[string]interface{})
	for _, metric := range metrics {
		switch metric.Name {
		case "system_memory":
			health["memory_usage_mb"] = metric.Value
		case "system_goroutines":
			health["goroutine_count"] = int(metric.Value)
		case "system_uptime":
			health["uptime_seconds"] = int(metric.Value)
		}
	}
	
	// Add collector stats
	stats := h.collector.GetCurrentStats()
	for key, value := range stats {
		health[key] = value
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    health,
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleStats returns collector statistics
func (h *APIHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats := h.collector.GetCurrentStats()
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}
	
	json.NewEncoder(w).Encode(response)
}

// parseMetricsQuery parses query parameters into MetricsQuery
func (h *APIHandler) parseMetricsQuery(r *http.Request) (MetricsQuery, error) {
	query := MetricsQuery{}
	
	// Parse names filter
	if names := r.URL.Query().Get("names"); names != "" {
		// Simple comma-separated parsing
		// In production, you might want more sophisticated parsing
		query.Names = []string{names}
	}
	
	// Parse time range
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if startInt, err := strconv.ParseInt(startStr, 10, 64); err == nil {
			query.StartTime = time.Unix(startInt, 0)
		} else {
			if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
				query.StartTime = startTime
			} else {
				return query, fmt.Errorf("invalid start time format")
			}
		}
	}
	
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if endInt, err := strconv.ParseInt(endStr, 10, 64); err == nil {
			query.EndTime = time.Unix(endInt, 0)
		} else {
			if endTime, err := time.Parse(time.RFC3339, endStr); err == nil {
				query.EndTime = endTime
			} else {
				return query, fmt.Errorf("invalid end time format")
			}
		}
	}
	
	// Set default time range if not specified (last hour)
	if query.StartTime.IsZero() && query.EndTime.IsZero() {
		query.EndTime = time.Now()
		query.StartTime = query.EndTime.Add(-time.Hour)
	}
	
	// Parse limit and offset
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}
	
	// Parse tags (simple key=value format)
	query.Tags = make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 && key != "start" && key != "end" && key != "limit" && key != "offset" && key != "names" {
			query.Tags[key] = values[0]
		}
	}
	
	return query, nil
}

// parseTimeRange parses time range from query parameters
func (h *APIHandler) parseTimeRange(r *http.Request) (TimeRange, error) {
	timeRange := TimeRange{}
	
	// Check for preset ranges
	rangeParam := r.URL.Query().Get("range")
	switch rangeParam {
	case "1h":
		timeRange.End = time.Now()
		timeRange.Start = timeRange.End.Add(-time.Hour)
	case "6h":
		timeRange.End = time.Now()
		timeRange.Start = timeRange.End.Add(-6 * time.Hour)
	case "24h":
		timeRange.End = time.Now()
		timeRange.Start = timeRange.End.Add(-24 * time.Hour)
	case "7d":
		timeRange.End = time.Now()
		timeRange.Start = timeRange.End.Add(-7 * 24 * time.Hour)
	default:
		// Parse custom start and end times
		if startStr := r.URL.Query().Get("start"); startStr != "" {
			if startInt, err := strconv.ParseInt(startStr, 10, 64); err == nil {
				timeRange.Start = time.Unix(startInt, 0)
			} else {
				if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
					timeRange.Start = startTime
				} else {
					return timeRange, fmt.Errorf("invalid start time format")
				}
			}
		}
		
		if endStr := r.URL.Query().Get("end"); endStr != "" {
			if endInt, err := strconv.ParseInt(endStr, 10, 64); err == nil {
				timeRange.End = time.Unix(endInt, 0)
			} else {
				if endTime, err := time.Parse(time.RFC3339, endStr); err == nil {
					timeRange.End = endTime
				} else {
					return timeRange, fmt.Errorf("invalid end time format")
				}
			}
		}
		
		// Default to last hour if no range specified
		if timeRange.Start.IsZero() && timeRange.End.IsZero() {
			timeRange.End = time.Now()
			timeRange.Start = timeRange.End.Add(-time.Hour)
		}
	}
	
	return timeRange, nil
}

// Middleware function to record HTTP metrics
func (h *APIHandler) RecordHTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response recorder to capture status code
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Call the next handler
		next.ServeHTTP(recorder, r)
		
		// Record metrics
		httpMetrics := HTTPMetrics{
			Path:       r.URL.Path,
			Method:     r.Method,
			StatusCode: recorder.statusCode,
			Duration:   time.Since(start),
			UserAgent:  r.UserAgent(),
			IPAddress:  getClientIP(r),
		}
		
		h.collector.RecordHTTPRequest(httpMetrics)
	})
}

// responseRecorder captures the status code from HTTP responses
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (from load balancers/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	
	// Check for X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// HandleTimeSeries returns time-series data based on query parameters
func (h *APIHandler) HandleTimeSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.timeSeries == nil {
		http.Error(w, "Time-series storage not available", http.StatusServiceUnavailable)
		return
	}
	
	// Parse time-series query parameters
	query, err := h.parseTimeSeriesQuery(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query parameters: %v", err), http.StatusBadRequest)
		return
	}
	
	// Get time-series data
	result, err := h.timeSeries.GetTimeSeries(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve time-series data: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    result,
		"query":   query,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleAggregatorStatus returns the current status of the aggregator
func (h *APIHandler) HandleAggregatorStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.timeSeries == nil || h.timeSeries.aggregator == nil {
		http.Error(w, "Aggregator not available", http.StatusServiceUnavailable)
		return
	}
	
	// Get aggregator status
	status := h.timeSeries.aggregator.GetStatus()
	memoryEstimate := h.timeSeries.aggregator.EstimateMemoryUsage()
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success":        true,
		"status":         status,
		"memory_usage":   memoryEstimate,
		"timestamp":      time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleForceAggregation manually triggers aggregation (useful for testing)
func (h *APIHandler) HandleForceAggregation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.timeSeries == nil || h.timeSeries.aggregator == nil {
		http.Error(w, "Aggregator not available", http.StatusServiceUnavailable)
		return
	}
	
	// Force aggregation
	err := h.timeSeries.aggregator.ForceAggregation()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to force aggregation: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success":   true,
		"message":   "Aggregation triggered successfully",
		"timestamp": time.Now().Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// parseTimeSeriesQuery parses query parameters into TimeSeriesQuery
func (h *APIHandler) parseTimeSeriesQuery(r *http.Request) (TimeSeriesQuery, error) {
	query := TimeSeriesQuery{}
	
	// Parse resolution (required)
	resolutionStr := r.URL.Query().Get("resolution")
	if resolutionStr == "" {
		resolutionStr = "1m" // Default to 1-minute resolution
	}
	
	switch resolutionStr {
	case "1m":
		query.Resolution = Resolution1Min
	case "5m":
		query.Resolution = Resolution5Min
	case "1h":
		query.Resolution = Resolution1Hour
	case "1d":
		query.Resolution = Resolution1Day
	default:
		return query, fmt.Errorf("invalid resolution: %s (supported: 1m, 5m, 1h, 1d)", resolutionStr)
	}
	
	// Parse metric names filter
	if names := r.URL.Query().Get("metrics"); names != "" {
		// Simple comma-separated parsing
		query.MetricNames = []string{names}
	}
	
	// Parse aggregation type
	query.Aggregation = r.URL.Query().Get("aggregation")
	if query.Aggregation == "" {
		query.Aggregation = "avg" // Default to average
	}
	
	// Validate aggregation type
	validAggregations := []string{"avg", "min", "max", "sum", "count", "latest"}
	isValid := false
	for _, valid := range validAggregations {
		if query.Aggregation == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return query, fmt.Errorf("invalid aggregation: %s (supported: %v)", query.Aggregation, validAggregations)
	}
	
	// Parse time range
	timeRange, err := h.parseTimeRange(r)
	if err != nil {
		return query, fmt.Errorf("invalid time range: %v", err)
	}
	
	query.StartTime = timeRange.Start
	query.EndTime = timeRange.End
	
	return query, nil
}

// HandleSystemMetrics returns comprehensive system metrics
func (h *APIHandler) HandleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.systemMetrics == nil {
		http.Error(w, "System metrics collector not available", http.StatusServiceUnavailable)
		return
	}
	
	// Get system health snapshot
	snapshot := h.systemMetrics.GetSystemHealthSnapshot()
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    snapshot,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleSystemSnapshot returns current system snapshot (lighter version)
func (h *APIHandler) HandleSystemSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.systemMetrics == nil {
		http.Error(w, "System metrics collector not available", http.StatusServiceUnavailable)
		return
	}
	
	snapshot := h.systemMetrics.GetSystemHealthSnapshot()
	
	// Create a lighter version of the snapshot for frequent polling
	lightSnapshot := map[string]interface{}{
		"timestamp":        snapshot.Timestamp.Unix(),
		"cpu_usage":        snapshot.CPUUsage,
		"memory_usage":     snapshot.MemoryUsage,
		"memory_allocated": snapshot.MemoryAllocated,
		"goroutine_count":  snapshot.GoroutineCount,
		"uptime_seconds":   snapshot.UptimeSeconds,
	}
	
	// Add provider health summary
	providerSummary := make(map[string]interface{})
	for name, health := range snapshot.ProviderHealth {
		providerSummary[name] = map[string]interface{}{
			"is_healthy":    health.IsHealthy,
			"health_score":  health.HealthScore,
			"error_rate":    health.ErrorRate,
		}
	}
	lightSnapshot["providers"] = providerSummary
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    lightSnapshot,
	}
	
	json.NewEncoder(w).Encode(response)
}

// HandleProviderHealth returns detailed provider health information
func (h *APIHandler) HandleProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if h.systemMetrics == nil {
		http.Error(w, "System metrics collector not available", http.StatusServiceUnavailable)
		return
	}
	
	snapshot := h.systemMetrics.GetSystemHealthSnapshot()
	
	// Return just the provider health data
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	response := map[string]interface{}{
		"success": true,
		"data":    snapshot.ProviderHealth,
		"timestamp": snapshot.Timestamp.Unix(),
	}
	
	json.NewEncoder(w).Encode(response)
}