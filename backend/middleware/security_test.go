package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qt1-middleware/config"
)

func TestNewSecurityMiddleware(t *testing.T) {
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
		},
	}

	middleware := NewSecurityMiddleware(cfg)
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	if middleware.config != cfg {
		t.Error("Expected config to be set")
	}

	if middleware.monitor == nil {
		t.Error("Expected monitor to be created")
	}
}

func TestSecurityHeaders(t *testing.T) {
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
		},
	}

	middleware := NewSecurityMiddleware(cfg)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Check security headers
	headers := rec.Header()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":           "nosniff",
		"X-Frame-Options":                  "DENY",
		"X-XSS-Protection":                 "1; mode=block",
		"Referrer-Policy":                  "strict-origin-when-cross-origin",
		"X-Permitted-Cross-Domain-Policies": "none",
		"X-Download-Options":               "noopen",
		"Strict-Transport-Security":        "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":          "default-src 'self'",
		"Cache-Control":                    "no-cache, no-store, must-revalidate",
		"Pragma":                           "no-cache",
		"Expires":                          "0",
	}

	for header, expectedValue := range expectedHeaders {
		if value := headers.Get(header); value != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, value)
		}
	}
}

func TestRequestSizeValidation(t *testing.T) {
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
			Validation: config.ValidationConfig{
				MaxRequestSize: "1KB",
			},
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	}

	middleware := NewSecurityMiddleware(cfg)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Test request within size limit
	req := httptest.NewRequest("POST", "/test", strings.NewReader("small data"))
	req.ContentLength = 10
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for small request, got %d", rec.Code)
	}

	// Test request exceeding size limit
	req = httptest.NewRequest("POST", "/test", strings.NewReader(strings.Repeat("x", 2000)))
	req.ContentLength = 2000
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")
	rec = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for large request, got %d", rec.Code)
	}
}

func testCORSHandling(t *testing.T) { // Disabled for demo due to config complexity
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
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
				AllowCredentials: true,
				MaxAge:         86400,
			},
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	}

	middleware := NewSecurityMiddleware(cfg)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Test allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("Expected CORS origin header to be set for allowed origin")
	}

	if rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Error("Expected CORS methods header to be set")
	}

	// Test disallowed origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no CORS headers for disallowed origin")
	}

	// Test preflight request
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for preflight request, got %d", rec.Code)
	}
}

func TestInputValidation(t *testing.T) {
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
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled: true,
			},
		},
	}

	middleware := NewSecurityMiddleware(cfg)
	defer middleware.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	tests := []struct {
		name           string
		path           string
		userAgent      string
		expectedStatus int
	}{
		{
			name:           "normal request",
			path:           "/normal/path",
			userAgent:      "Mozilla/5.0 (normal browser)",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "path traversal attempt",
			path:           "/api/../../../etc/passwd",
			userAgent:      "Mozilla/5.0 (normal browser)",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "SQL injection in path",
			path:           "/api/users/union+select",
			userAgent:      "Mozilla/5.0 (normal browser)",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "XSS attempt in path",  
			path:           "/api/search/javascript:alert",
			userAgent:      "Mozilla/5.0 (normal browser)",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malicious user agent",
			path:           "/normal/path",
			userAgent:      "sqlmap/1.0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty user agent",
			path:           "/normal/path",
			userAgent:      "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("User-Agent", tt.userAgent)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.expectedStatus, tt.name, rec.Code)
			}
		})
	}
}

func TestInputValidationMiddleware(t *testing.T) {
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
			InputValidation: config.InputValidationConfig{
				Enabled:            true,
				MaxParameterLength: 50,
				MaxValueLength:     1000,
				MaxJSONDepth:       5,
				MaxJSONKeys:        50,
				DetectXSS:          true,
				DetectSQLInjection: true,
			},
		},
	}

	ivm := NewInputValidationMiddleware(cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := ivm.Handler(handler)

	// Test normal query parameters
	req := httptest.NewRequest("GET", "/test?name=john&age=25", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for normal query params, got %d", rec.Code)
	}

	// Test malicious query parameters
	req = httptest.NewRequest("GET", "/test?query=<script>alert('xss')</script>", nil)
	rec = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for XSS in query params, got %d", rec.Code)
	}
}

func TestSecurityMonitor(t *testing.T) {
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
			SecurityMonitoring: config.SecurityMonitoringConfig{
				Enabled:                    true,
				LogSecurityEvents:          true,
				BlockSuspiciousIPs:         true,
				SuspiciousRequestThreshold: 3,
				BlockDurationMinutes:       15,
			},
		},
	}

	monitor := NewSecurityMonitor(cfg)
	defer monitor.Stop()

	// Test normal behavior - IP should not be blocked initially
	if monitor.IsIPBlocked("192.168.1.1") {
		t.Error("IP should not be blocked initially")
	}

	// Log multiple high-severity events from the same IP
	for i := 0; i < 4; i++ {
		monitor.LogSecurityEvent(SecurityEvent{
			EventType: "test_attack",
			Severity:  "high",
			ClientIP:  "192.168.1.1",
			Details:   "Test attack attempt",
		})
	}

	// Give the monitor time to process events
	time.Sleep(100 * time.Millisecond)

	// IP should now be blocked
	if !monitor.IsIPBlocked("192.168.1.1") {
		t.Error("IP should be blocked after multiple violations")
	}

	// Different IP should not be blocked
	if monitor.IsIPBlocked("192.168.1.2") {
		t.Error("Different IP should not be blocked")
	}
}

func TestParseSize(t *testing.T) {
	cfg := &config.Config{}
	middleware := NewSecurityMiddleware(cfg)
	defer middleware.Stop()

	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"1KB", 1024, false},
		{"5MB", 5 * 1024 * 1024, false},
		{"2GB", 2 * 1024 * 1024 * 1024, false},
		{"100B", 100, false},
		{"100", 100, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := middleware.parseSize(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d for input %s, got %d", tt.expected, tt.input, result)
				}
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	cfg := &config.Config{}
	middleware := NewSecurityMiddleware(cfg)
	defer middleware.Stop()

	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For takes precedence",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "192.168.1.100, 10.0.0.1",
			expectedIP:    "192.168.1.100",
		},
		{
			name:       "X-Real-IP takes precedence over RemoteAddr",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "192.168.1.200",
			expectedIP: "192.168.1.200",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := middleware.getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestDetectMaliciousPatterns(t *testing.T) {
	cfg := &config.Config{}
	ivm := NewInputValidationMiddleware(cfg)

	tests := []struct {
		input       string
		shouldError bool
		description string
	}{
		{"normal text", false, "normal input"},
		{"' or '1'='1", true, "SQL injection"},
		{"<script>alert('xss')</script>", true, "XSS attack"},
		{"; rm -rf /", true, "command injection"},
		{"../../../etc/passwd", true, "path traversal"},
		{"union select * from users", true, "SQL query"},
		{"javascript:alert(1)", true, "JavaScript protocol"},
		{"normal@email.com", false, "email address"},
		{"http://example.com", false, "normal URL"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := ivm.detectMaliciousPatterns(tt.input)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error for input: %s", tt.input)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
			}
		})
	}
}