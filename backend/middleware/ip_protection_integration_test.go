package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qt1-middleware/config"
)

// TestIPProtectionIntegration tests the complete IP protection middleware integration
func TestIPProtectionIntegration(t *testing.T) {
	// Create a complete config similar to production
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
			Headers: config.SecurityHeaders{
				Enabled:    true,
				HSTSMaxAge: 31536000,
				CSPPolicy:  "default-src 'self'",
			},
			InputValidation: config.InputValidationConfig{
				Enabled:            true,
				DetectXSS:         true,
				MaxParameterLength: 100,
				MaxValueLength:     10000,
			},
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
			IPProtection: config.IPProtectionConfig{
				Enabled:               true,
				EnableGeoblocking:     true,
				EnableReputationCheck: true,
				BlockedCountries:      []string{"CN", "RU"},
				AllowedCountries:      []string{},
				BlockedIPs:           []string{"10.0.0.100"},  // Use IP outside allowed range
				AllowedIPs:           []string{"127.0.0.1", "192.168.1.0/24"},
				SuspiciousThreshold:   3,
				AutoBlockDuration:     15 * time.Minute,
				ReputationThreshold:   0.7,
				BlockMaliciousIPs:     true,
			},
		},
	}

	// Create the complete middleware chain
	securityMonitor := NewSecurityMonitor(cfg)
	defer securityMonitor.Stop()
	
	securityMiddleware := NewSecurityMiddleware(cfg)
	defer securityMiddleware.Stop()
	
	ipProtectionMiddleware := NewIPProtectionMiddleware(&cfg.Security.IPProtection, securityMonitor)

	// Create a simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Request processed successfully"))
	})

	// Create the middleware chain (reverse order)
	var handler http.Handler = testHandler
	handler = securityMiddleware.Handler(handler)
	handler = ipProtectionMiddleware.Handler(handler)

	// Test cases for integration
	tests := []struct {
		name           string
		clientIP       string
		userAgent      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "allowed IP should pass all middleware",
			clientIP:       "127.0.0.1:12345",
			userAgent:      "Go-Test-Agent/1.0",
			expectedStatus: http.StatusOK,
			expectedBody:   "Request processed successfully",
		},
		{
			name:           "blocked IP should be denied by IP protection",
			clientIP:       "10.0.0.100:12345",  // IP that is blocked and not in allowlist
			userAgent:      "Go-Test-Agent/1.0",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied",
		},
		{
			name:           "Chinese IP should be blocked by geolocation",
			clientIP:       "103.224.182.1:12345",
			userAgent:      "Go-Test-Agent/1.0",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied",
		},
		{
			name:           "malicious IP should be blocked by reputation",
			clientIP:       "1.2.3.4:12345",
			userAgent:      "Go-Test-Agent/1.0",
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Access denied",
		},
		{
			name:           "normal IP with clean user agent should pass",
			clientIP:       "8.8.8.8:12345",
			userAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedStatus: http.StatusOK,
			expectedBody:   "Request processed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.clientIP
			req.Header.Set("User-Agent", tt.userAgent)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			body := strings.TrimSpace(rec.Body.String())
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, body)
			}
		})
	}
}

// TestIPProtectionPerformance measures the performance impact of IP protection
func TestIPProtectionPerformance(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:               true,
		EnableGeoblocking:     true,
		EnableReputationCheck: true,
		BlockedIPs:           []string{"192.168.1.100"},
		AllowedIPs:           []string{"127.0.0.1"},
		SuspiciousThreshold:   5,
		AutoBlockDuration:     15 * time.Minute,
		ReputationThreshold:   0.7,
		BlockMaliciousIPs:     true,
	}

	securityMonitor := NewSecurityMonitor(&config.Config{
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
	defer securityMonitor.Stop()

	middleware := NewIPProtectionMiddleware(cfg, securityMonitor)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(testHandler)

	// Performance test parameters
	numRequests := 1000
	allowedIP := "127.0.0.1:12345"

	// Measure performance
	start := time.Now()
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = allowedIP
		req.Header.Set("User-Agent", "Performance-Test-Agent/1.0")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d failed with status %d", i, rec.Code)
		}
	}
	elapsed := time.Since(start)

	// Calculate average processing time per request
	avgTime := elapsed / time.Duration(numRequests)
	
	t.Logf("Processed %d requests in %v", numRequests, elapsed)
	t.Logf("Average processing time per request: %v", avgTime)

	// Performance threshold: should be less than 1ms per request for IP protection
	if avgTime > time.Millisecond {
		t.Errorf("IP Protection performance too slow: %v per request (expected < 1ms)", avgTime)
	}
}

// TestSuspiciousActivityIntegration tests the suspicious activity detection and auto-blocking
func TestSuspiciousActivityIntegration(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:             true,
		SuspiciousThreshold: 3,
		AutoBlockDuration:   100 * time.Millisecond, // Short duration for testing
	}

	securityMonitor := NewSecurityMonitor(&config.Config{
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
	defer securityMonitor.Stop()

	middleware := NewIPProtectionMiddleware(cfg, securityMonitor)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Handler(testHandler)
	testIP := "192.168.1.50:12345"

	// Record suspicious activity to trigger auto-blocking
	middleware.RecordSuspiciousActivity("192.168.1.50", "test violation 1")
	middleware.RecordSuspiciousActivity("192.168.1.50", "test violation 2")
	middleware.RecordSuspiciousActivity("192.168.1.50", "test violation 3")

	// First request should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = testIP
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected request to be blocked with status 403, got %d", rec.Code)
	}

	// Wait for auto-block to expire
	time.Sleep(150 * time.Millisecond)

	// Request should now be allowed
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = testIP
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected request to be allowed after auto-block expiry, got status %d", rec.Code)
	}

	// Check suspicious IPs data
	suspiciousIPs := middleware.GetSuspiciousIPs()
	if activity, exists := suspiciousIPs["192.168.1.50"]; exists {
		if activity.ViolationCount != 3 {
			t.Errorf("Expected 3 violations, got %d", activity.ViolationCount)
		}
	} else {
		t.Error("Expected IP to be in suspicious IPs list")
	}
}

// TestMiddlewareOrder tests that middleware is applied in the correct order
func TestMiddlewareOrder(t *testing.T) {
	// This test verifies that IP Protection (outermost) -> Security Headers (middle) -> App (innermost)
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
			Headers: config.SecurityHeaders{
				Enabled:    true,
				HSTSMaxAge: 31536000,
				CSPPolicy:  "default-src 'self'",
			},
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
			IPProtection: config.IPProtectionConfig{
				Enabled:    true,
				BlockedIPs: []string{"192.168.1.100"},
			},
		},
	}

	// Create middleware chain
	securityMonitor := NewSecurityMonitor(cfg)
	defer securityMonitor.Stop()
	
	securityMiddleware := NewSecurityMiddleware(cfg)
	defer securityMiddleware.Stop()
	
	ipProtectionMiddleware := NewIPProtectionMiddleware(&cfg.Security.IPProtection, securityMonitor)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply in correct order: IP Protection (outer) -> Security (inner) -> Handler
	var handler http.Handler = testHandler
	handler = securityMiddleware.Handler(handler)
	handler = ipProtectionMiddleware.Handler(handler)

	// Test with allowed IP - should get security headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check that security headers are present
	if hsts := rec.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Error("Expected HSTS header to be set")
	}

	// Test with blocked IP - should be denied before reaching security middleware
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for blocked IP, got %d", rec.Code)
	}

	// Should still have some basic headers but not reach the inner security middleware
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, "Access denied") {
		t.Errorf("Expected IP protection error message, got: %s", body)
	}
}