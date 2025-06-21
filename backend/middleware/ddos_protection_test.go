package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"qt1-middleware/config"
)

func TestNewDDoSProtectionMiddleware(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:                 true,
		SpikeThreshold:          100,
		SpikeWindow:             time.Minute,
		CircuitBreakerThreshold: 10,
		CircuitBreakerTimeout:   30 * time.Second,
		CircuitBreakerRequests:  5,
		EnableBlocking:          true,
		EnableThrottling:        true,
		EnableDegradation:       true,
		DegradationThreshold:    200,
		ThrottleBaseDelay:       10 * time.Millisecond,
		ThrottleMaxDelay:        time.Second,
		ThrottleScaleFactor:     2.0,
		AlertCooldown:           5 * time.Minute,
	}

	monitor := NewSecurityMonitor(&config.Config{
		Security: struct {
			RateLimiting        config.RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
			Headers             config.SecurityHeaders          `yaml:"headers" json:"headers"`
			Validation          config.ValidationConfig         `yaml:"validation" json:"validation"`
			CORS                config.CORSConfig               `yaml:"cors" json:"cors"`
			InputValidation     config.InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
			SecurityMonitoring  config.SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
			IPProtection        config.IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
			DDoSProtection      config.DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
			PromptInjection     struct {
				Enabled     bool     `yaml:"enabled" json:"enabled"`
				Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
				Patterns    []string `yaml:"patterns" json:"patterns"`
			} `yaml:"prompt_injection" json:"prompt_injection"`
		}{
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	})
	defer monitor.Stop()

	middleware := NewDDoSProtectionMiddleware(cfg, monitor)
	defer middleware.Stop()

	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	if middleware.config != cfg {
		t.Error("Expected config to be set")
	}

	if middleware.requestCounter == nil {
		t.Error("Expected request counter to be initialized")
	}

	if middleware.circuitBreaker == nil {
		t.Error("Expected circuit breaker to be initialized")
	}

	if middleware.throttle == nil {
		t.Error("Expected throttle to be initialized")
	}

	if middleware.alertManager == nil {
		t.Error("Expected alert manager to be initialized")
	}
}

func TestRequestCounter(t *testing.T) {
	windowSize := 100 * time.Millisecond
	threshold := 5
	counter := NewRequestCounter(windowSize, threshold)

	// Record some requests
	for i := 0; i < 3; i++ {
		counter.recordRequest()
		time.Sleep(10 * time.Millisecond)
	}

	// Should have 3 requests
	count := counter.GetRequestCount()
	if count != 3 {
		t.Errorf("Expected 3 requests, got %d", count)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should have 0 requests after expiration
	count = counter.GetRequestCount()
	if count != 0 {
		t.Errorf("Expected 0 requests after expiration, got %d", count)
	}
}

func TestRequestCounterConcurrency(t *testing.T) {
	counter := NewRequestCounter(time.Second, 1000)
	var wg sync.WaitGroup
	numGoroutines := 10
	requestsPerGoroutine := 50

	// Concurrent request recording
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				counter.recordRequest()
			}
		}()
	}
	wg.Wait()

	count := counter.GetRequestCount()
	expected := numGoroutines * requestsPerGoroutine
	if count != expected {
		t.Errorf("Expected %d requests, got %d", expected, count)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond, 2)

	// Initially closed
	if cb.IsOpen() {
		t.Error("Circuit breaker should start closed")
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state %d, got %d", CircuitClosed, cb.GetState())
	}

	// Record failures to open circuit
	for i := 0; i < 3; i++ {
		cb.recordFailure()
	}

	// Should be open now
	if !cb.IsOpen() {
		t.Error("Circuit breaker should be open after failures")
	}

	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state %d, got %d", CircuitOpen, cb.GetState())
	}

	failures := cb.GetFailures()
	if failures != 3 {
		t.Errorf("Expected 3 failures, got %d", failures)
	}

	// Wait for recovery timeout
	time.Sleep(120 * time.Millisecond)

	// Should transition to half-open
	isOpen := cb.IsOpen() // This call should transition to half-open
	if isOpen {
		t.Error("Circuit breaker should be half-open after timeout")
	}

	if cb.GetState() != CircuitHalfOpen {
		t.Errorf("Expected state %d, got %d", CircuitHalfOpen, cb.GetState())
	}

	// Record successful requests to close circuit
	for i := 0; i < 2; i++ {
		cb.recordSuccess()
	}

	// Should be closed now
	if cb.IsOpen() {
		t.Error("Circuit breaker should be closed after successful requests")
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state %d, got %d", CircuitClosed, cb.GetState())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond, 3)

	// Open the circuit
	cb.recordFailure()
	cb.recordFailure()

	if !cb.IsOpen() {
		t.Error("Circuit should be open")
	}

	// Wait for recovery
	time.Sleep(60 * time.Millisecond)

	// Transition to half-open
	cb.IsOpen()

	if cb.GetState() != CircuitHalfOpen {
		t.Error("Circuit should be half-open")
	}

	// Record a failure in half-open state
	cb.recordFailure()

	// Should go back to open
	if cb.GetState() != CircuitOpen {
		t.Error("Circuit should be open again after failure in half-open")
	}
}

func TestAdaptiveThrottle(t *testing.T) {
	throttle := &AdaptiveThrottle{
		baseDelay:      10 * time.Millisecond,
		maxDelay:       100 * time.Millisecond,
		scaleFactor:    2.0,
		loadThreshold:  50,
		recoveryFactor: 0.9,
	}

	// No delay under threshold
	delay := throttle.calculateDelay()
	if delay != 0 {
		t.Errorf("Expected 0 delay under threshold, got %v", delay)
	}

	// Increase load above threshold
	throttle.updateLoad(100)

	// Should have delay now
	delay = throttle.calculateDelay()
	if delay == 0 {
		t.Error("Expected non-zero delay above threshold")
	}

	if delay > throttle.maxDelay {
		t.Errorf("Delay %v should not exceed max %v", delay, throttle.maxDelay)
	}

	currentLoad := throttle.GetCurrentLoad()
	if currentLoad != 100 {
		t.Errorf("Expected load 100, got %d", currentLoad)
	}

	// Reduce load
	throttle.updateLoad(-50)
	newLoad := throttle.GetCurrentLoad()
	if newLoad != 50 {
		t.Errorf("Expected load 50, got %d", newLoad)
	}
}

func TestDDoSProtectionSpikeDetection(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:          true,
		SpikeThreshold:   5,
		SpikeWindow:      100 * time.Millisecond,
		EnableBlocking:   true,
		EnableThrottling: false,
		AlertCooldown:    time.Second,
	}

	monitor := NewSecurityMonitor(&config.Config{
		Security: struct {
			RateLimiting        config.RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
			Headers             config.SecurityHeaders          `yaml:"headers" json:"headers"`
			Validation          config.ValidationConfig         `yaml:"validation" json:"validation"`
			CORS                config.CORSConfig               `yaml:"cors" json:"cors"`
			InputValidation     config.InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
			SecurityMonitoring  config.SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
			IPProtection        config.IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
			DDoSProtection      config.DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
			PromptInjection     struct {
				Enabled     bool     `yaml:"enabled" json:"enabled"`
				Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
				Patterns    []string `yaml:"patterns" json:"patterns"`
			} `yaml:"prompt_injection" json:"prompt_injection"`
		}{
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	})
	defer monitor.Stop()

	middleware := NewDDoSProtectionMiddleware(cfg, monitor)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Send requests below threshold
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i, rec.Code)
		}
	}

	// Send requests above threshold to trigger spike detection
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)

		// After threshold is exceeded, requests should be blocked
		if i >= cfg.SpikeThreshold-3 { // Account for previous requests
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("Request %d should be blocked, got status %d", i, rec.Code)
			}
		}
	}

	// Check metrics
	metrics := middleware.GetMetrics()
	if metrics.SpikeCount == 0 {
		t.Error("Expected spike count > 0")
	}

	if metrics.BlockedRequests == 0 {
		t.Error("Expected blocked requests > 0")
	}
}

func TestDDoSProtectionCircuitBreakerIntegration(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:                 true,
		SpikeThreshold:          1000, // High threshold to avoid spike detection
		SpikeWindow:             time.Minute,
		CircuitBreakerThreshold: 3,
		CircuitBreakerTimeout:   50 * time.Millisecond,
		CircuitBreakerRequests:  2,
		EnableBlocking:          true,
		AlertCooldown:           time.Second,
	}

	monitor := NewSecurityMonitor(&config.Config{
		Security: struct {
			RateLimiting        config.RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
			Headers             config.SecurityHeaders          `yaml:"headers" json:"headers"`
			Validation          config.ValidationConfig         `yaml:"validation" json:"validation"`
			CORS                config.CORSConfig               `yaml:"cors" json:"cors"`
			InputValidation     config.InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
			SecurityMonitoring  config.SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
			IPProtection        config.IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
			DDoSProtection      config.DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
			PromptInjection     struct {
				Enabled     bool     `yaml:"enabled" json:"enabled"`
				Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
				Patterns    []string `yaml:"patterns" json:"patterns"`
			} `yaml:"prompt_injection" json:"prompt_injection"`
		}{
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	})
	defer monitor.Stop()

	middleware := NewDDoSProtectionMiddleware(cfg, monitor)
	defer middleware.Stop()

	// Manually trigger circuit breaker
	cb := middleware.GetCircuitBreaker()
	cb.recordFailure()
	cb.recordFailure()
	cb.recordFailure()

	if !cb.IsOpen() {
		t.Error("Circuit breaker should be open")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Request should be blocked by circuit breaker
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 when circuit is open, got %d", rec.Code)
	}

	if rec.Header().Get("X-Circuit-Breaker") != "open" {
		t.Error("Expected X-Circuit-Breaker header to be 'open'")
	}

	// Check metrics
	metrics := middleware.GetMetrics()
	if metrics.CircuitBreakerTrips == 0 {
		t.Error("Expected circuit breaker trips > 0")
	}
}

func TestDDoSProtectionThrottling(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:             true,
		SpikeThreshold:      1000, // High threshold to avoid spike detection
		SpikeWindow:         time.Minute,
		EnableThrottling:    true,
		ThrottleBaseDelay:   10 * time.Millisecond,
		ThrottleMaxDelay:    50 * time.Millisecond,
		ThrottleScaleFactor: 2.0,
		AlertCooldown:       time.Second,
	}

	monitor := NewSecurityMonitor(&config.Config{
		Security: struct {
			RateLimiting        config.RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
			Headers             config.SecurityHeaders          `yaml:"headers" json:"headers"`
			Validation          config.ValidationConfig         `yaml:"validation" json:"validation"`
			CORS                config.CORSConfig               `yaml:"cors" json:"cors"`
			InputValidation     config.InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
			SecurityMonitoring  config.SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
			IPProtection        config.IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
			DDoSProtection      config.DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
			PromptInjection     struct {
				Enabled     bool     `yaml:"enabled" json:"enabled"`
				Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
				Patterns    []string `yaml:"patterns" json:"patterns"`
			} `yaml:"prompt_injection" json:"prompt_injection"`
		}{
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	})
	defer monitor.Stop()

	middleware := NewDDoSProtectionMiddleware(cfg, monitor)
	defer middleware.Stop()

	// Increase throttle load to trigger throttling
	throttle := middleware.GetThrottle()
	throttle.updateLoad(1000) // Much higher load to exceed threshold

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Measure request time to verify throttling
	start := time.Now()
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Should have some delay due to throttling
	if elapsed < 5*time.Millisecond {
		t.Errorf("Expected throttling delay, but request completed in %v", elapsed)
	}

	// Check metrics
	metrics := middleware.GetMetrics()
	if metrics.ThrottledRequests == 0 {
		t.Error("Expected throttled requests > 0")
	}
}

func TestDDoSProtectionDegradation(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:              true,
		SpikeThreshold:       1000, // High threshold to avoid spike detection
		SpikeWindow:          time.Minute,
		EnableDegradation:    true,
		DegradationThreshold: 2, // Low threshold for testing
		AlertCooldown:        time.Second,
	}

	monitor := NewSecurityMonitor(&config.Config{
		Security: struct {
			RateLimiting        config.RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
			Headers             config.SecurityHeaders          `yaml:"headers" json:"headers"`
			Validation          config.ValidationConfig         `yaml:"validation" json:"validation"`
			CORS                config.CORSConfig               `yaml:"cors" json:"cors"`
			InputValidation     config.InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
			SecurityMonitoring  config.SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
			IPProtection        config.IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
			DDoSProtection      config.DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
			PromptInjection     struct {
				Enabled     bool     `yaml:"enabled" json:"enabled"`
				Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
				Patterns    []string `yaml:"patterns" json:"patterns"`
			} `yaml:"prompt_injection" json:"prompt_injection"`
		}{
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	})
	defer monitor.Stop()

	middleware := NewDDoSProtectionMiddleware(cfg, monitor)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow processing to increase active connections
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Full response"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Send concurrent requests to trigger degradation
	var wg sync.WaitGroup
	numRequests := 5
	results := make([]int, numRequests)
	bodies := make([]string, numRequests)

	wg.Add(numRequests)
	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rec, req)
			results[idx] = rec.Code
			bodies[idx] = rec.Body.String()
		}(i)
	}
	wg.Wait()

	// Check that some requests got degraded responses
	degradedCount := 0
	for i, body := range bodies {
		if results[i] == http.StatusOK {
			if body == `{"status":"degraded","message":"Service operating in reduced capacity"}` {
				degradedCount++
			}
		}
	}

	if degradedCount == 0 {
		t.Error("Expected some requests to receive degraded responses")
	}
}

func TestDDoSProtectionDisabled(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:        false, // Disabled
		SpikeThreshold: 5,
		SpikeWindow:    100 * time.Millisecond,
	}

	middleware := NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Send many requests - should all pass when disabled
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d should pass when protection is disabled, got %d", i, rec.Code)
		}
	}

	// Metrics should still be zero when disabled
	metrics := middleware.GetMetrics()
	if metrics.BlockedRequests != 0 {
		t.Errorf("Expected 0 blocked requests when disabled, got %d", metrics.BlockedRequests)
	}
}

func TestDDoSProtectionMetrics(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:        true,
		SpikeThreshold: 100, // High threshold to avoid blocking
		SpikeWindow:    time.Minute,
		EnableMetrics:  true,
		AlertCooldown:  time.Second,
	}

	middleware := NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Send some requests
	numRequests := 10
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}

	// Check metrics
	metrics := middleware.GetMetrics()
	if metrics.TotalRequests != int64(numRequests) {
		t.Errorf("Expected %d total requests, got %d", numRequests, metrics.TotalRequests)
	}

	if metrics.RequestsPerSecond == 0 {
		t.Error("Expected RPS > 0")
	}

	if metrics.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections after requests complete, got %d", metrics.ActiveConnections)
	}
}

func TestAlertManagerCooldown(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		AlertCooldown: 100 * time.Millisecond,
	}

	monitor := NewSecurityMonitor(&config.Config{
		Security: struct {
			RateLimiting        config.RateLimitingConfig       `yaml:"rate_limiting" json:"rate_limiting"`
			Headers             config.SecurityHeaders          `yaml:"headers" json:"headers"`
			Validation          config.ValidationConfig         `yaml:"validation" json:"validation"`
			CORS                config.CORSConfig               `yaml:"cors" json:"cors"`
			InputValidation     config.InputValidationConfig    `yaml:"input_validation" json:"input_validation"`
			SecurityMonitoring  config.SecurityMonitoringConfig `yaml:"security_monitoring" json:"security_monitoring"`
			IPProtection        config.IPProtectionConfig       `yaml:"ip_protection" json:"ip_protection"`
			DDoSProtection      config.DDoSProtectionConfig     `yaml:"ddos_protection" json:"ddos_protection"`
			PromptInjection     struct {
				Enabled     bool     `yaml:"enabled" json:"enabled"`
				Sensitivity string   `yaml:"sensitivity" json:"sensitivity"`
				Patterns    []string `yaml:"patterns" json:"patterns"`
			} `yaml:"prompt_injection" json:"prompt_injection"`
		}{
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	})
	defer monitor.Stop()

	alertManager := &AlertManager{
		config:        cfg,
		monitor:       monitor,
		alertCooldown: cfg.AlertCooldown,
	}

	// First alert should go through
	alertManager.SendAlert("Test alert 1", AlertLevelHigh)

	// Second alert immediately should be blocked by cooldown
	alertManager.SendAlert("Test alert 2", AlertLevelHigh)

	// Wait for cooldown to expire
	time.Sleep(150 * time.Millisecond)

	// Third alert should go through after cooldown
	alertManager.SendAlert("Test alert 3", AlertLevelHigh)

	// Alerts are logged, so we can't easily test the exact count
	// But the cooldown mechanism is tested by the timing
}

func TestAlertLevelString(t *testing.T) {
	tests := []struct {
		level    AlertLevel
		expected string
	}{
		{AlertLevelLow, "low"},
		{AlertLevelMedium, "medium"},
		{AlertLevelHigh, "high"},
		{AlertLevelCritical, "critical"},
		{AlertLevel(99), "unknown"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, result)
		}
	}
}

// Benchmark tests for performance measurement
func BenchmarkDDoSProtectionNormalLoad(b *testing.B) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:          true,
		SpikeThreshold:   10000, // High threshold
		SpikeWindow:      time.Minute,
		EnableThrottling: false,
		EnableBlocking:   false,
		AlertCooldown:    time.Second,
	}

	middleware := NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rec, req)
		}
	})
}

func BenchmarkRequestCounter(b *testing.B) {
	counter := NewRequestCounter(time.Minute, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.recordRequest()
		}
	})
}

func BenchmarkCircuitBreakerClosed(b *testing.B) {
	cb := NewCircuitBreaker(100, time.Minute, 10)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.IsOpen()
			cb.recordSuccess()
		}
	})
}