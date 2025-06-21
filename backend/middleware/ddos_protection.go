package middleware

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"qt1-middleware/config"
)

// DDoSProtectionMiddleware provides comprehensive DDoS protection
type DDoSProtectionMiddleware struct {
	requestCounter  *RequestCounter
	circuitBreaker  *CircuitBreaker
	alertManager    *AlertManager
	throttle        *AdaptiveThrottle
	config          *config.DDoSProtectionConfig
	monitor         *SecurityMonitor
	metrics         *DDoSMetrics
	mutex           sync.RWMutex
}

// DDoSMetrics tracks DDoS protection statistics
type DDoSMetrics struct {
	TotalRequests       int64 `json:"total_requests"`
	BlockedRequests     int64 `json:"blocked_requests"`
	ThrottledRequests   int64 `json:"throttled_requests"`
	CircuitBreakerTrips int64 `json:"circuit_breaker_trips"`
	ActiveConnections   int64 `json:"active_connections"`
	RequestsPerSecond   int64 `json:"requests_per_second"`
	LastSpikeTime       time.Time `json:"last_spike_time"`
	SpikeCount          int64 `json:"spike_count"`
}

// RequestCounter tracks requests within a time window for spike detection
type RequestCounter struct {
	requests   []time.Time
	windowSize time.Duration
	maxSize    int
	mutex      sync.RWMutex
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	failureThreshold int
	recoveryTimeout  time.Duration
	requestThreshold int
	state            int32 // 0=Closed, 1=Open, 2=Half-Open
	failures         int64
	lastFailureTime  time.Time
	successCount     int64
	mutex            sync.RWMutex
}

// Circuit breaker states
const (
	CircuitClosed   = 0
	CircuitOpen     = 1
	CircuitHalfOpen = 2
)

// AdaptiveThrottle implements dynamic request throttling
type AdaptiveThrottle struct {
	baseDelay        time.Duration
	maxDelay         time.Duration
	scaleFactor      float64
	currentLoad      int64
	loadThreshold    int64
	recoveryFactor   float64
	lastAdjustment   time.Time
	mutex            sync.RWMutex
}

// AlertManager handles DDoS-related alerts and notifications
type AlertManager struct {
	config   *config.DDoSProtectionConfig
	monitor  *SecurityMonitor
	lastAlert time.Time
	alertCooldown time.Duration
	mutex    sync.RWMutex
}

// AlertLevel defines the severity of security alerts
type AlertLevel int

const (
	AlertLevelLow AlertLevel = iota
	AlertLevelMedium
	AlertLevelHigh
	AlertLevelCritical
)

// String returns the string representation of AlertLevel
func (a AlertLevel) String() string {
	switch a {
	case AlertLevelLow:
		return "low"
	case AlertLevelMedium:
		return "medium"
	case AlertLevelHigh:
		return "high"
	case AlertLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// NewDDoSProtectionMiddleware creates a new DDoS protection middleware
func NewDDoSProtectionMiddleware(cfg *config.DDoSProtectionConfig, monitor *SecurityMonitor) *DDoSProtectionMiddleware {
	ddos := &DDoSProtectionMiddleware{
		config:  cfg,
		monitor: monitor,
		metrics: &DDoSMetrics{},
	}

	// Initialize request counter
	ddos.requestCounter = &RequestCounter{
		requests:   make([]time.Time, 0, cfg.SpikeThreshold*2),
		windowSize: cfg.SpikeWindow,
		maxSize:    cfg.SpikeThreshold * 3, // Buffer for cleanup
	}

	// Initialize circuit breaker
	ddos.circuitBreaker = &CircuitBreaker{
		failureThreshold: cfg.CircuitBreakerThreshold,
		recoveryTimeout:  cfg.CircuitBreakerTimeout,
		requestThreshold: cfg.CircuitBreakerRequests,
	}

	// Initialize adaptive throttle
	ddos.throttle = &AdaptiveThrottle{
		baseDelay:      cfg.ThrottleBaseDelay,
		maxDelay:       cfg.ThrottleMaxDelay,
		scaleFactor:    cfg.ThrottleScaleFactor,
		loadThreshold:  int64(cfg.SpikeThreshold / 2), // Half spike threshold
		recoveryFactor: 0.95, // Gradual recovery
	}

	// Initialize alert manager
	ddos.alertManager = &AlertManager{
		config:        cfg,
		monitor:       monitor,
		alertCooldown: cfg.AlertCooldown,
	}

	// Start background cleanup and monitoring
	go ddos.backgroundWorker()

	log.Printf("DDoS Protection initialized (spike threshold: %d, window: %v, circuit breaker: %d failures)",
		cfg.SpikeThreshold, cfg.SpikeWindow, cfg.CircuitBreakerThreshold)

	return ddos
}

// Handler returns the HTTP middleware handler for DDoS protection
func (ddos *DDoSProtectionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ddos.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		defer func() {
			atomic.AddInt64(&ddos.metrics.TotalRequests, 1)
			atomic.AddInt64(&ddos.metrics.ActiveConnections, -1)
		}()

		atomic.AddInt64(&ddos.metrics.ActiveConnections, 1)

		// Record request for spike detection
		ddos.requestCounter.recordRequest()

		// Check circuit breaker first
		if ddos.circuitBreaker.IsOpen() {
			ddos.handleCircuitOpen(w, r)
			return
		}

		// Detect request spikes
		if ddos.detectSpike() {
			if ddos.config.EnableBlocking {
				ddos.handleDDoSBlocked(w, r)
				return
			}
		}

		// Apply adaptive throttling
		if ddos.config.EnableThrottling {
			if delay := ddos.throttle.calculateDelay(); delay > 0 {
				time.Sleep(delay)
				atomic.AddInt64(&ddos.metrics.ThrottledRequests, 1)
			}
		}

		// Check for graceful degradation
		if ddos.config.EnableDegradation && ddos.shouldDegrade() {
			ddos.handleDegradedResponse(w, r)
			return
		}

		// Process request normally
		next.ServeHTTP(w, r)

		// Record successful request for circuit breaker
		ddos.circuitBreaker.recordSuccess()

		// Update load metrics
		processingTime := time.Since(start)
		ddos.updateLoadMetrics(processingTime)
	})
}

// NewRequestCounter creates a new request counter
func NewRequestCounter(windowSize time.Duration, threshold int) *RequestCounter {
	return &RequestCounter{
		requests:   make([]time.Time, 0, threshold*2),
		windowSize: windowSize,
		maxSize:    threshold * 3,
	}
}

// recordRequest records a new request timestamp
func (rc *RequestCounter) recordRequest() {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	now := time.Now()
	rc.requests = append(rc.requests, now)

	// Clean up old requests if buffer is getting full
	if len(rc.requests) > rc.maxSize {
		rc.cleanup(now)
	}
}

// getRecentRequests returns requests within the time window
func (rc *RequestCounter) getRecentRequests() []time.Time {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rc.windowSize)

	var recent []time.Time
	for _, req := range rc.requests {
		if req.After(cutoff) {
			recent = append(recent, req)
		}
	}

	return recent
}

// cleanup removes requests older than the window
func (rc *RequestCounter) cleanup(now time.Time) {
	cutoff := now.Add(-rc.windowSize)
	validIdx := 0

	for i, req := range rc.requests {
		if req.After(cutoff) {
			validIdx = i
			break
		}
	}

	if validIdx > 0 {
		copy(rc.requests, rc.requests[validIdx:])
		rc.requests = rc.requests[:len(rc.requests)-validIdx]
	}
}

// GetRequestCount returns current request count in window
func (rc *RequestCounter) GetRequestCount() int {
	return len(rc.getRecentRequests())
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, timeout time.Duration, requestThreshold int) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		recoveryTimeout:  timeout,
		requestThreshold: requestThreshold,
		state:           CircuitClosed,
	}
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch atomic.LoadInt32(&cb.state) {
	case CircuitOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
			atomic.StoreInt32(&cb.state, CircuitHalfOpen)
			atomic.StoreInt64(&cb.successCount, 0)
			return false
		}
		return true
	case CircuitHalfOpen:
		return false
	default:
		return false
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	atomic.AddInt64(&cb.failures, 1)
	cb.lastFailureTime = time.Now()

	currentState := atomic.LoadInt32(&cb.state)
	if currentState == CircuitClosed && cb.failures >= int64(cb.failureThreshold) {
		atomic.StoreInt32(&cb.state, CircuitOpen)
		log.Printf("Circuit breaker opened after %d failures", cb.failures)
	} else if currentState == CircuitHalfOpen {
		atomic.StoreInt32(&cb.state, CircuitOpen)
		log.Printf("Circuit breaker re-opened during half-open state")
	}
}

// recordSuccess records a successful request
func (cb *CircuitBreaker) recordSuccess() {
	currentState := atomic.LoadInt32(&cb.state)
	if currentState == CircuitHalfOpen {
		atomic.AddInt64(&cb.successCount, 1)
		if atomic.LoadInt64(&cb.successCount) >= int64(cb.requestThreshold) {
			cb.mutex.Lock()
			atomic.StoreInt32(&cb.state, CircuitClosed)
			atomic.StoreInt64(&cb.failures, 0)
			atomic.StoreInt64(&cb.successCount, 0)
			cb.mutex.Unlock()
			log.Printf("Circuit breaker closed after successful recovery")
		}
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() int32 {
	return atomic.LoadInt32(&cb.state)
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int64 {
	return atomic.LoadInt64(&cb.failures)
}

// calculateDelay calculates throttling delay based on current load
func (at *AdaptiveThrottle) calculateDelay() time.Duration {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	currentLoad := atomic.LoadInt64(&at.currentLoad)
	if currentLoad <= at.loadThreshold {
		return 0
	}

	// Calculate exponential backoff based on load
	loadRatio := float64(currentLoad) / float64(at.loadThreshold)
	delay := time.Duration(float64(at.baseDelay) * loadRatio * at.scaleFactor)

	if delay > at.maxDelay {
		delay = at.maxDelay
	}

	return delay
}

// updateLoad updates the current load metric
func (at *AdaptiveThrottle) updateLoad(delta int64) {
	atomic.AddInt64(&at.currentLoad, delta)
	
	// Ensure load doesn't go negative
	if atomic.LoadInt64(&at.currentLoad) < 0 {
		atomic.StoreInt64(&at.currentLoad, 0)
	}
}

// GetCurrentLoad returns current load value
func (at *AdaptiveThrottle) GetCurrentLoad() int64 {
	return atomic.LoadInt64(&at.currentLoad)
}

// SendAlert sends a DDoS-related security alert
func (am *AlertManager) SendAlert(message string, level AlertLevel) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	now := time.Now()
	if now.Sub(am.lastAlert) < am.alertCooldown {
		return // Skip alert due to cooldown
	}

	am.lastAlert = now

	if am.monitor != nil {
		event := SecurityEvent{
			EventType: "ddos_attack",
			Severity:  level.String(),
			Details:   message,
			Timestamp: now,
			Blocked:   true,
		}
		am.monitor.LogSecurityEvent(event)
	}

	log.Printf("DDoS ALERT [%s]: %s", level.String(), message)
}

// detectSpike detects if there's a request spike indicating potential DDoS
func (ddos *DDoSProtectionMiddleware) detectSpike() bool {
	recentCount := ddos.requestCounter.GetRequestCount()
	
	if recentCount > ddos.config.SpikeThreshold {
		atomic.StoreInt64(&ddos.metrics.SpikeCount, atomic.LoadInt64(&ddos.metrics.SpikeCount)+1)
		ddos.metrics.LastSpikeTime = time.Now()
		
		ddos.alertManager.SendAlert(
			fmt.Sprintf("Request spike detected: %d requests in %v (threshold: %d)",
				recentCount, ddos.config.SpikeWindow, ddos.config.SpikeThreshold),
			AlertLevelHigh)
		
		// Record failure for circuit breaker
		ddos.circuitBreaker.recordFailure()
		
		// Increase throttling load
		ddos.throttle.updateLoad(int64(recentCount - ddos.config.SpikeThreshold))
		
		return true
	}
	
	return false
}

// shouldDegrade determines if the system should enter degraded mode
func (ddos *DDoSProtectionMiddleware) shouldDegrade() bool {
	activeConns := atomic.LoadInt64(&ddos.metrics.ActiveConnections)
	return activeConns > int64(ddos.config.DegradationThreshold)
}

// handleDDoSBlocked handles blocked DDoS requests
func (ddos *DDoSProtectionMiddleware) handleDDoSBlocked(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&ddos.metrics.BlockedRequests, 1)
	
	w.Header().Set("Retry-After", "60")
	w.Header().Set("X-DDoS-Protection", "blocked")
	
	http.Error(w, "Service temporarily unavailable due to high load", http.StatusServiceUnavailable)
}

// handleCircuitOpen handles requests when circuit breaker is open
func (ddos *DDoSProtectionMiddleware) handleCircuitOpen(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&ddos.metrics.BlockedRequests, 1)
	atomic.AddInt64(&ddos.metrics.CircuitBreakerTrips, 1)
	
	w.Header().Set("Retry-After", "30")
	w.Header().Set("X-Circuit-Breaker", "open")
	
	http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
}

// handleDegradedResponse handles requests in degraded mode
func (ddos *DDoSProtectionMiddleware) handleDegradedResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Service-Mode", "degraded")
	w.Header().Set("Cache-Control", "no-cache")
	
	// Return simplified response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"degraded","message":"Service operating in reduced capacity"}`))
}

// updateLoadMetrics updates various load-related metrics
func (ddos *DDoSProtectionMiddleware) updateLoadMetrics(processingTime time.Duration) {
	// Update requests per second (simple moving average)
	recentRequests := ddos.requestCounter.getRecentRequests()
	rps := int64(len(recentRequests))
	atomic.StoreInt64(&ddos.metrics.RequestsPerSecond, rps)
	
	// Gradually reduce throttling load during normal operation
	if processingTime < 100*time.Millisecond { // Fast response indicates normal load
		currentLoad := ddos.throttle.GetCurrentLoad()
		if currentLoad > 0 {
			reduction := int64(float64(currentLoad) * (1.0 - ddos.throttle.recoveryFactor))
			ddos.throttle.updateLoad(-reduction)
		}
	}
}

// backgroundWorker runs background maintenance tasks
func (ddos *DDoSProtectionMiddleware) backgroundWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Cleanup old requests
		ddos.requestCounter.cleanup(time.Now())
		
		// Log metrics periodically
		if ddos.config.EnableMetrics {
			ddos.logMetrics()
		}
	}
}

// logMetrics logs current DDoS protection metrics
func (ddos *DDoSProtectionMiddleware) logMetrics() {
	metricsMap := ddos.GetMetrics().(map[string]interface{})
	
	log.Printf("DDoS Metrics - Total: %v, Blocked: %v, Throttled: %v, Active: %v, RPS: %v, Spikes: %v",
		metricsMap["total_requests"],
		metricsMap["blocked_requests"],
		metricsMap["throttled_requests"],
		metricsMap["active_connections"],
		metricsMap["requests_per_second"],
		metricsMap["spike_count"])
}

// GetMetrics returns current DDoS protection metrics
func (ddos *DDoSProtectionMiddleware) GetMetrics() interface{} {
	ddos.mutex.RLock()
	defer ddos.mutex.RUnlock()
	
	// Return as map to match interface{} requirement and avoid import cycle
	return map[string]interface{}{
		"total_requests":        atomic.LoadInt64(&ddos.metrics.TotalRequests),
		"blocked_requests":      atomic.LoadInt64(&ddos.metrics.BlockedRequests),
		"throttled_requests":    atomic.LoadInt64(&ddos.metrics.ThrottledRequests),
		"circuit_breaker_trips": atomic.LoadInt64(&ddos.metrics.CircuitBreakerTrips),
		"active_connections":    atomic.LoadInt64(&ddos.metrics.ActiveConnections),
		"requests_per_second":   atomic.LoadInt64(&ddos.metrics.RequestsPerSecond),
		"last_spike_time":       ddos.metrics.LastSpikeTime,
		"spike_count":           atomic.LoadInt64(&ddos.metrics.SpikeCount),
	}
}

// GetStatus returns current DDoS protection status
func (ddos *DDoSProtectionMiddleware) GetStatus() interface{} {
	ddos.mutex.RLock()
	defer ddos.mutex.RUnlock()
	
	// Determine circuit breaker state
	cbState := "closed"
	switch ddos.circuitBreaker.GetState() {
	case 1:
		cbState = "open"
	case 2:
		cbState = "half-open"
	}
	
	// Return as map to match interface{} requirement and avoid import cycle
	return map[string]interface{}{
		"protection_enabled":     true,
		"requests_per_second":    atomic.LoadInt64(&ddos.metrics.RequestsPerSecond),
		"blocked_requests":       atomic.LoadInt64(&ddos.metrics.BlockedRequests),
		"spike_count":            atomic.LoadInt64(&ddos.metrics.SpikeCount),
		"circuit_breaker_state":  cbState,
		"failure_count":          ddos.circuitBreaker.GetFailures(),
		"active_connections":     atomic.LoadInt64(&ddos.metrics.ActiveConnections),
		"throttled_requests":     atomic.LoadInt64(&ddos.metrics.ThrottledRequests),
	}
}

// GetRequestCounter returns the request counter for testing
func (ddos *DDoSProtectionMiddleware) GetRequestCounter() *RequestCounter {
	return ddos.requestCounter
}

// GetCircuitBreaker returns the circuit breaker for testing
func (ddos *DDoSProtectionMiddleware) GetCircuitBreaker() *CircuitBreaker {
	return ddos.circuitBreaker
}

// GetThrottle returns the adaptive throttle for testing
func (ddos *DDoSProtectionMiddleware) GetThrottle() *AdaptiveThrottle {
	return ddos.throttle
}

// Stop gracefully stops the DDoS protection middleware
func (ddos *DDoSProtectionMiddleware) Stop() {
	log.Println("DDoS protection middleware stopped")
}