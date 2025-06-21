package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"qt1-middleware/config"
)

// TestFullSecurityStackIntegration tests the complete security middleware stack
func TestFullSecurityStackIntegration(t *testing.T) {
	// Create a comprehensive security configuration
	cfg := &config.Config{
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
			RateLimiting: config.RateLimitingConfig{
				Enabled: true,
				Global: config.RateLimitRule{
					RequestsPerSecond: 50,
					Burst:             100,
					WindowSize:        time.Minute,
					Algorithm:         "token_bucket",
				},
			},
			Headers: config.SecurityHeaders{
				Enabled:    true,
				HSTSMaxAge: 31536000,
				CSPPolicy:  "default-src 'self'",
			},
			InputValidation: config.InputValidationConfig{
				Enabled:  true,
				DetectXSS: true,
			},
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
			IPProtection: config.IPProtectionConfig{
				Enabled:    true,
				BlockedIPs: []string{"10.0.0.100"},
			},
			DDoSProtection: config.DDoSProtectionConfig{
				Enabled:                 true,
				SpikeThreshold:          10,
				SpikeWindow:             100 * time.Millisecond,
				CircuitBreakerThreshold: 5,
				CircuitBreakerTimeout:   50 * time.Millisecond,
				CircuitBreakerRequests:  3,
				EnableBlocking:          true,
				EnableThrottling:        true,
				EnableDegradation:       false, // Disable for cleaner testing
				ThrottleBaseDelay:       1 * time.Millisecond,
				ThrottleMaxDelay:        10 * time.Millisecond,
				ThrottleScaleFactor:     2.0,
				AlertCooldown:           time.Second,
			},
		},
	}

	// Create the full middleware stack
	securityMonitor := NewSecurityMonitor(cfg)
	defer securityMonitor.Stop()

	rateLimitMiddleware := NewRateLimitMiddleware(&cfg.Security.RateLimiting)
	defer rateLimitMiddleware.Stop()

	securityMiddleware := NewSecurityMiddleware(cfg)
	defer securityMiddleware.Stop()

	ipProtectionMiddleware := NewIPProtectionMiddleware(&cfg.Security.IPProtection, securityMonitor)

	ddosProtectionMiddleware := NewDDoSProtectionMiddleware(&cfg.Security.DDoSProtection, securityMonitor)
	defer ddosProtectionMiddleware.Stop()

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request processed successfully"))
	})

	// Build middleware chain (reverse order)
	var handler http.Handler = testHandler
	handler = rateLimitMiddleware.Handler(handler)
	handler = securityMiddleware.Handler(handler)
	handler = ipProtectionMiddleware.Handler(handler)
	handler = ddosProtectionMiddleware.Handler(handler)

	// Test normal operation
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Normal request should succeed, got status %d", rec.Code)
	}

	// Verify security headers are present
	if hsts := rec.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Error("Expected HSTS header to be set")
	}

	// Test blocked IP
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.100:12345"
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Blocked IP should be denied, got status %d", rec.Code)
	}

	// Test DDoS protection - send burst of requests
	var wg sync.WaitGroup
	numRequests := 20
	var successCount int32
	var blockedCount int32

	wg.Add(numRequests)
	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("User-Agent", "Load-Test-Agent/1.0")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				atomic.AddInt32(&successCount, 1)
			} else if rec.Code == http.StatusServiceUnavailable {
				atomic.AddInt32(&blockedCount, 1)
			}
		}()
	}
	wg.Wait()

	// Some requests should be blocked due to DDoS protection
	if blockedCount == 0 {
		t.Error("Expected some requests to be blocked by DDoS protection")
	}

	if successCount == 0 {
		t.Error("Expected some requests to succeed")
	}

	t.Logf("Load test results: %d successful, %d blocked", successCount, blockedCount)

	// Check DDoS metrics
	metrics := ddosProtectionMiddleware.GetMetrics()
	if metrics.TotalRequests == 0 {
		t.Error("Expected total requests > 0")
	}

	if metrics.SpikeCount == 0 {
		t.Error("Expected spike count > 0")
	}
}

// TestDDoSProtectionAttackSimulation simulates various types of attacks
func TestDDoSProtectionAttackSimulation(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:                 true,
		SpikeThreshold:          20,
		SpikeWindow:             200 * time.Millisecond,
		CircuitBreakerThreshold: 10,
		CircuitBreakerTimeout:   100 * time.Millisecond,
		CircuitBreakerRequests:  5,
		EnableBlocking:          true,
		EnableThrottling:        true,
		EnableDegradation:       true,
		DegradationThreshold:    15,
		ThrottleBaseDelay:       2 * time.Millisecond,
		ThrottleMaxDelay:        20 * time.Millisecond,
		ThrottleScaleFactor:     1.5,
		AlertCooldown:           50 * time.Millisecond,
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
		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	// Simulate flood attack
	t.Run("FloodAttack", func(t *testing.T) {
		var wg sync.WaitGroup
		attackSize := 50
		var responses [50]int

		start := time.Now()
		wg.Add(attackSize)
		for i := 0; i < attackSize; i++ {
			go func(idx int) {
				defer wg.Done()
				req := httptest.NewRequest("GET", "/test", nil)
				req.RemoteAddr = "192.168.1.2:12345"
				rec := httptest.NewRecorder()

				wrappedHandler.ServeHTTP(rec, req)
				responses[idx] = rec.Code
			}(i)
		}
		wg.Wait()
		attackDuration := time.Since(start)

		// Count response types
		okCount := 0
		blockedCount := 0
		degradedCount := 0

		for _, code := range responses {
			switch code {
			case http.StatusOK:
				okCount++
			case http.StatusServiceUnavailable:
				blockedCount++
			}
		}

		t.Logf("Flood attack (%d requests in %v): %d OK, %d blocked, %d degraded",
			attackSize, attackDuration, okCount, blockedCount, degradedCount)

		// Should have blocked some requests
		if blockedCount == 0 {
			t.Error("Expected some requests to be blocked during flood attack")
		}

		// Should have processed some requests (system still functional)
		if okCount == 0 {
			t.Error("Expected some requests to succeed (system should remain partially functional)")
		}
	})

	// Check system recovery
	t.Run("SystemRecovery", func(t *testing.T) {
		// Wait for spike window to expire and system to recover
		time.Sleep(300 * time.Millisecond)

		// Send normal requests
		normalRequests := 5
		successCount := 0

		for i := 0; i < normalRequests; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.3:12345"
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				successCount++
			}

			time.Sleep(50 * time.Millisecond) // Spread requests over time
		}

		t.Logf("Recovery test: %d/%d requests successful", successCount, normalRequests)

		// Most requests should succeed after recovery
		if float64(successCount)/float64(normalRequests) < 0.8 {
			t.Errorf("Expected at least 80%% success rate during recovery, got %d/%d",
				successCount, normalRequests)
		}
	})

	// Check final metrics
	metrics := middleware.GetMetrics()
	t.Logf("Final metrics: Total=%d, Blocked=%d, Throttled=%d, Spikes=%d, CircuitTrips=%d",
		metrics.TotalRequests, metrics.BlockedRequests, metrics.ThrottledRequests,
		metrics.SpikeCount, metrics.CircuitBreakerTrips)

	if metrics.TotalRequests == 0 {
		t.Error("Expected total requests > 0")
	}

	if metrics.SpikeCount == 0 {
		t.Error("Expected spike count > 0")
	}
}

// TestDDoSProtectionPerformanceUnderLoad measures performance under load
func TestDDoSProtectionPerformanceUnderLoad(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:             true,
		SpikeThreshold:      10000, // Very high threshold to avoid interference
		SpikeWindow:         time.Minute,
		EnableThrottling:    false, // Disable for performance measurement
		EnableBlocking:      false,
		EnableDegradation:   false,
		EnableMetrics:       true,
		AlertCooldown:       time.Second,
	}

	middleware := NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Performance test parameters
	numRequests := 1000
	concurrency := 10

	// Warm up with separate middleware instance to avoid metric contamination
	warmupMiddleware := NewDDoSProtectionMiddleware(cfg, nil)
	warmupHandler := warmupMiddleware.Handler(handler)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		warmupHandler.ServeHTTP(rec, req)
	}
	warmupMiddleware.Stop()

	// Measure performance with fresh middleware
	middleware = NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()
	wrappedHandler = middleware.Handler(handler)

	start := time.Now()
	var wg sync.WaitGroup
	requestsPerWorker := numRequests / concurrency

	wg.Add(concurrency)
	for worker := 0; worker < concurrency; worker++ {
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerWorker; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				rec := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(rec, req)
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(numRequests)
	rps := float64(numRequests) / elapsed.Seconds()

	t.Logf("Performance test: %d requests in %v", numRequests, elapsed)
	t.Logf("Average processing time: %v per request", avgTime)
	t.Logf("Throughput: %.1f requests/second", rps)

	// Performance requirements
	if avgTime > 100*time.Microsecond {
		t.Errorf("DDoS protection overhead too high: %v per request (expected < 100µs)", avgTime)
	}

	if rps < 1000 {
		t.Errorf("Throughput too low: %.1f RPS (expected > 1000)", rps)
	}

	// Check metrics
	metrics := middleware.GetMetrics()
	if metrics.TotalRequests != int64(numRequests) {
		t.Errorf("Expected %d total requests, got %d", numRequests, metrics.TotalRequests)
	}
}

// TestCircuitBreakerRecovery tests circuit breaker recovery patterns
func TestCircuitBreakerRecovery(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:                 true,
		SpikeThreshold:          1000, // High threshold to avoid spike detection
		SpikeWindow:             time.Minute,
		CircuitBreakerThreshold: 3,
		CircuitBreakerTimeout:   100 * time.Millisecond,
		CircuitBreakerRequests:  5,
		EnableBlocking:          true,
		AlertCooldown:           time.Second,
	}

	middleware := NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()

	// Simulate failing handler
	failingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate server error
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrappedHandler := middleware.Handler(failingHandler)

	// Trigger circuit breaker by causing failures
	cb := middleware.GetCircuitBreaker()
	for i := 0; i < 3; i++ {
		cb.recordFailure()
	}

	if !cb.IsOpen() {
		t.Error("Circuit breaker should be open after failures")
	}

	// Requests should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 when circuit is open, got %d", rec.Code)
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open
	isOpen := cb.IsOpen()
	if isOpen {
		t.Error("Circuit should be half-open after timeout")
	}

	// Create working handler for recovery
	workingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedWorkingHandler := middleware.Handler(workingHandler)

	// Send successful requests to close circuit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedWorkingHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d should succeed during recovery, got %d", i, rec.Code)
		}
	}

	// Circuit should be closed now
	if cb.IsOpen() {
		t.Error("Circuit should be closed after successful requests")
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected circuit state %d, got %d", CircuitClosed, cb.GetState())
	}

	t.Logf("Circuit breaker successfully recovered: failures=%d, state=%d",
		cb.GetFailures(), cb.GetState())
}

// TestAdaptiveThrottlingBehavior tests adaptive throttling under various loads
func TestAdaptiveThrottlingBehavior(t *testing.T) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:             true,
		SpikeThreshold:      1000, // High threshold
		SpikeWindow:         time.Minute,
		EnableThrottling:    true,
		ThrottleBaseDelay:   5 * time.Millisecond,
		ThrottleMaxDelay:    50 * time.Millisecond,
		ThrottleScaleFactor: 2.0,
		AlertCooldown:       time.Second,
	}

	middleware := NewDDoSProtectionMiddleware(cfg, nil)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)
	throttle := middleware.GetThrottle()

	// Test low load - no throttling
	t.Run("LowLoad", func(t *testing.T) {
		throttle.updateLoad(10) // Low load

		start := time.Now()
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		if elapsed > 10*time.Millisecond {
			t.Errorf("Expected minimal delay under low load, got %v", elapsed)
		}
	})

	// Test high load - throttling should kick in
	t.Run("HighLoad", func(t *testing.T) {
		throttle.updateLoad(1000) // High load

		start := time.Now()
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		if elapsed < cfg.ThrottleBaseDelay {
			t.Errorf("Expected throttling delay under high load, got %v", elapsed)
		}

		if elapsed > cfg.ThrottleMaxDelay*2 {
			t.Errorf("Throttling delay too high: %v (max: %v)", elapsed, cfg.ThrottleMaxDelay)
		}
	})

	// Test load recovery
	t.Run("LoadRecovery", func(t *testing.T) {
		// Gradually reduce load
		for i := 0; i < 10; i++ {
			start := time.Now()
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rec, req)
			elapsed := time.Since(start)

			t.Logf("Request %d: delay=%v, load=%d", i, elapsed, throttle.GetCurrentLoad())
			time.Sleep(10 * time.Millisecond)
		}

		// Load should have decreased significantly from initial 1000
		finalLoad := throttle.GetCurrentLoad()
		if finalLoad >= 1000 {
			t.Errorf("Expected load to decrease during recovery, got %d (started at 1000)", finalLoad)
		}
		
		// The recovery should show a clear downward trend
		if finalLoad > 800 {
			t.Errorf("Expected more significant load reduction, got %d", finalLoad)
		}
	})
}

// Benchmark the complete DDoS protection middleware
func BenchmarkDDoSProtectionComplete(b *testing.B) {
	cfg := &config.DDoSProtectionConfig{
		Enabled:             true,
		SpikeThreshold:      10000, // High threshold
		SpikeWindow:         time.Minute,
		EnableThrottling:    false, // Disable for benchmark
		EnableBlocking:      false,
		EnableDegradation:   false,
		EnableMetrics:       true,
		AlertCooldown:       time.Second,
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