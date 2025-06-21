package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qt1-middleware/config"
)

func TestNewIPProtectionMiddleware(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:             true,
		EnableGeoblocking:   false,
		EnableReputationCheck: true,
		BlockedIPs:          []string{"192.168.1.100"},
		AllowedIPs:          []string{"127.0.0.1"},
		SuspiciousThreshold: 5,
		AutoBlockDuration:   15 * time.Minute,
		ReputationThreshold: 0.7,
		BlockMaliciousIPs:   true,
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

	middleware := NewIPProtectionMiddleware(cfg, monitor)
	
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	if middleware.config != cfg {
		t.Error("Expected config to be set")
	}

	if middleware.geolocator == nil {
		t.Error("Expected geolocator to be initialized")
	}

	if middleware.reputation == nil {
		t.Error("Expected reputation service to be initialized")
	}
}

func TestIPAllowlistBlocklist(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:    true,
		BlockedIPs: []string{"192.168.1.100", "10.0.0.0/8"},
		AllowedIPs: []string{"127.0.0.1", "192.168.1.0/24"},
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	tests := []struct {
		name     string
		ip       string
		allowed  bool
		blocked  bool
	}{
		{
			name:    "allowed IP",
			ip:      "127.0.0.1",
			allowed: true,
			blocked: false,
		},
		{
			name:    "IP in allowed CIDR",
			ip:      "192.168.1.50",
			allowed: true,
			blocked: false,
		},
		{
			name:    "blocked IP (but allowlisted by CIDR)",
			ip:      "192.168.1.100",
			allowed: true,  // This IP is in the allowed CIDR range 192.168.1.0/24
			blocked: true,  // Also in blocked list, but allowlist takes precedence
		},
		{
			name:    "IP in blocked CIDR",
			ip:      "10.0.0.5",
			allowed: false,
			blocked: true,
		},
		{
			name:    "unknown IP",
			ip:      "8.8.8.8",
			allowed: false,
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := middleware.isAllowlisted(tt.ip)
			blocked := middleware.isBlocked(tt.ip)

			if allowed != tt.allowed {
				t.Errorf("Expected allowlisted %v for IP %s, got %v", tt.allowed, tt.ip, allowed)
			}

			if blocked != tt.blocked {
				t.Errorf("Expected blocked %v for IP %s, got %v", tt.blocked, tt.ip, blocked)
			}
		})
	}
}

func TestGeoblocking(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:           true,
		EnableGeoblocking: true,
		BlockedCountries:  []string{"RU", "CN"},
		AllowedCountries:  []string{}, // Empty means all allowed except blocked
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	tests := []struct {
		name        string
		ip          string
		shouldBlock bool
	}{
		{
			name:        "US IP should be allowed",
			ip:          "1.1.1.1",
			shouldBlock: false,
		},
		{
			name:        "Russian IP should be blocked",
			ip:          "185.220.100.1",
			shouldBlock: true,
		},
		{
			name:        "Chinese IP should be blocked",
			ip:          "103.224.182.1",
			shouldBlock: true,
		},
		{
			name:        "French IP should be allowed",
			ip:          "37.187.96.1",
			shouldBlock: false,
		},
		{
			name:        "Local IP should be allowed",
			ip:          "192.168.1.1",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := middleware.checkGeolocation(tt.ip)
			hasError := err != nil

			if hasError != tt.shouldBlock {
				t.Errorf("Expected block %v for IP %s, got error: %v", tt.shouldBlock, tt.ip, err)
			}
		})
	}
}

func TestGeoblockingAllowedCountriesOnly(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:           true,
		EnableGeoblocking: true,
		BlockedCountries:  []string{},
		AllowedCountries:  []string{"US", "FR"}, // Only US and France allowed
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	tests := []struct {
		name        string
		ip          string
		shouldBlock bool
	}{
		{
			name:        "US IP should be allowed",
			ip:          "1.1.1.1",
			shouldBlock: false,
		},
		{
			name:        "French IP should be allowed",
			ip:          "37.187.96.1",
			shouldBlock: false,
		},
		{
			name:        "Russian IP should be blocked",
			ip:          "185.220.100.1",
			shouldBlock: true,
		},
		{
			name:        "Chinese IP should be blocked",
			ip:          "103.224.182.1",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := middleware.checkGeolocation(tt.ip)
			hasError := err != nil

			if hasError != tt.shouldBlock {
				t.Errorf("Expected block %v for IP %s, got error: %v", tt.shouldBlock, tt.ip, err)
			}
		})
	}
}

func TestIPReputationChecking(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:               true,
		EnableReputationCheck: true,
		ReputationThreshold:   0.7,
		BlockMaliciousIPs:     true,
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	tests := []struct {
		name        string
		ip          string
		shouldBlock bool
	}{
		{
			name:        "clean IP should be allowed",
			ip:          "8.8.8.8",
			shouldBlock: false,
		},
		{
			name:        "malicious IP should be blocked",
			ip:          "1.2.3.4",
			shouldBlock: true,
		},
		{
			name:        "high reputation score IP should be blocked",
			ip:          "185.220.100.1",
			shouldBlock: true,
		},
		{
			name:        "medium reputation score IP should be blocked (score = threshold)",
			ip:          "192.168.1.100",
			shouldBlock: true,  // Score 0.7 >= threshold 0.7, so blocked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := middleware.checkReputation(tt.ip)
			hasError := err != nil

			if hasError != tt.shouldBlock {
				t.Errorf("Expected block %v for IP %s, got error: %v", tt.shouldBlock, tt.ip, err)
			}
		})
	}
}

func TestSuspiciousActivityTracking(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:           true,
		SuspiciousThreshold: 3,
		AutoBlockDuration: 15 * time.Minute,
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	ip := "192.168.1.50"

	// Record suspicious activity below threshold
	for i := 0; i < 2; i++ {
		middleware.RecordSuspiciousActivity(ip, "test violation")
	}

	// Should not be blocked yet
	if middleware.isSuspiciouslyBlocked(ip) {
		t.Error("IP should not be blocked before reaching threshold")
	}

	// Record one more to reach threshold
	middleware.RecordSuspiciousActivity(ip, "test violation")

	// Should now be blocked
	if !middleware.isSuspiciouslyBlocked(ip) {
		t.Error("IP should be blocked after reaching threshold")
	}

	// Check suspicious IP data
	suspiciousIPs := middleware.GetSuspiciousIPs()
	if activity, exists := suspiciousIPs[ip]; exists {
		if activity.ViolationCount != 3 {
			t.Errorf("Expected 3 violations, got %d", activity.ViolationCount)
		}
		if activity.Reason != "test violation" {
			t.Errorf("Expected reason 'test violation', got '%s'", activity.Reason)
		}
	} else {
		t.Error("Expected IP to be in suspicious IPs list")
	}
}

func TestIPProtectionMiddlewareHandler(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:    true,
		BlockedIPs: []string{"192.168.1.100"},
		AllowedIPs: []string{"127.0.0.1"},
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrappedHandler := middleware.Handler(handler)

	tests := []struct {
		name           string
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "allowed IP should pass",
			clientIP:       "127.0.0.1:12345",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blocked IP should be denied",
			clientIP:       "192.168.1.100:12345",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "unknown IP should pass",
			clientIP:       "8.8.8.8:12345",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.clientIP
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d for IP %s, got %d", tt.expectedStatus, tt.clientIP, rec.Code)
			}
		})
	}
}

func TestIPProtectionDisabled(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:    false, // Disabled
		BlockedIPs: []string{"192.168.1.100"},
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Handler(handler)

	// Even blocked IP should pass when protection is disabled
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 when protection is disabled, got %d", rec.Code)
	}
}

func TestAddRemoveFromBlocklist(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled: true,
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	ip := "1.2.3.4"

	// Should not be blocked initially
	if middleware.isBlocked(ip) {
		t.Error("IP should not be blocked initially")
	}

	// Add to blocklist
	middleware.AddToBlocklist(ip, "test block")

	// Should now be blocked
	if !middleware.isBlocked(ip) {
		t.Error("IP should be blocked after adding to blocklist")
	}

	// Remove from blocklist
	middleware.RemoveFromBlocklist(ip)

	// Should not be blocked anymore
	if middleware.isBlocked(ip) {
		t.Error("IP should not be blocked after removing from blocklist")
	}
}

func TestAddToAllowlist(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled: true,
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	ip := "1.2.3.4"

	// Should not be allowlisted initially
	if middleware.isAllowlisted(ip) {
		t.Error("IP should not be allowlisted initially")
	}

	// Add to allowlist
	middleware.AddToAllowlist(ip)

	// Should now be allowlisted
	if !middleware.isAllowlisted(ip) {
		t.Error("IP should be allowlisted after adding to allowlist")
	}
}

func TestCIDRRangeMatching(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled: true,
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	tests := []struct {
		name     string
		ip       string
		cidr     string
		expected bool
	}{
		{
			name:     "IP in CIDR range",
			ip:       "192.168.1.50",
			cidr:     "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "IP not in CIDR range",
			ip:       "192.168.2.50",
			cidr:     "192.168.1.0/24",
			expected: false,
		},
		{
			name:     "exact IP match",
			ip:       "192.168.1.1",
			cidr:     "192.168.1.1",
			expected: true,
		},
		{
			name:     "IP in larger CIDR",
			ip:       "10.1.1.1",
			cidr:     "10.0.0.0/8",
			expected: true,
		},
		{
			name:     "IP not in larger CIDR",
			ip:       "172.16.1.1",
			cidr:     "10.0.0.0/8",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.isIPInCIDR(tt.ip, tt.cidr)
			if result != tt.expected {
				t.Errorf("Expected %v for IP %s in CIDR %s, got %v", tt.expected, tt.ip, tt.cidr, result)
			}
		})
	}
}

func TestIPProtectionGetClientIP(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled: true,
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

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

func TestMockGeoLocator(t *testing.T) {
	geolocator := NewMockGeoLocator()

	tests := []struct {
		name            string
		ip              string
		expectedCountry string
		isValid         bool
	}{
		{
			name:            "US IP",
			ip:              "1.1.1.1",
			expectedCountry: "United States",
			isValid:         true,
		},
		{
			name:            "Russian IP",
			ip:              "185.220.100.1",
			expectedCountry: "Russia",
			isValid:         true,
		},
		{
			name:            "Private IP",
			ip:              "192.168.1.1",
			expectedCountry: "Local",
			isValid:         false,
		},
		{
			name:            "Unknown IP",
			ip:              "5.5.5.5",
			expectedCountry: "Unknown",
			isValid:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, err := geolocator.GetLocation(tt.ip)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if location.Country != tt.expectedCountry {
				t.Errorf("Expected country %s, got %s", tt.expectedCountry, location.Country)
			}

			isValid := geolocator.IsValidIP(tt.ip)
			if isValid != tt.isValid {
				t.Errorf("Expected IsValidIP %v for IP %s, got %v", tt.isValid, tt.ip, isValid)
			}
		})
	}
}

func TestMockIPReputationService(t *testing.T) {
	reputation := NewMockIPReputationService()

	tests := []struct {
		name         string
		ip           string
		expectedMalicious bool
		expectedScore     float64
	}{
		{
			name:              "clean IP",
			ip:                "8.8.8.8",
			expectedMalicious: false,
			expectedScore:     0.1,
		},
		{
			name:              "malicious IP",
			ip:                "1.2.3.4",
			expectedMalicious: true,
			expectedScore:     0.8,
		},
		{
			name:              "tor exit node",
			ip:                "185.220.100.1",
			expectedMalicious: true,
			expectedScore:     0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep, err := reputation.CheckReputation(tt.ip)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if rep.IsMalicious != tt.expectedMalicious {
				t.Errorf("Expected malicious %v for IP %s, got %v", tt.expectedMalicious, tt.ip, rep.IsMalicious)
			}

			if rep.Score != tt.expectedScore {
				t.Errorf("Expected score %f for IP %s, got %f", tt.expectedScore, tt.ip, rep.Score)
			}

			isMalicious := reputation.IsKnownMalicious(tt.ip)
			if isMalicious != tt.expectedMalicious {
				t.Errorf("Expected IsKnownMalicious %v for IP %s, got %v", tt.expectedMalicious, tt.ip, isMalicious)
			}
		})
	}
}

func TestCleanup(t *testing.T) {
	cfg := &config.IPProtectionConfig{
		Enabled:           true,
		SuspiciousThreshold: 5,
		AutoBlockDuration: 1 * time.Millisecond, // Very short for testing
	}

	middleware := NewIPProtectionMiddleware(cfg, nil)

	// Add suspicious activity
	middleware.RecordSuspiciousActivity("192.168.1.1", "test")

	// Verify it exists
	suspiciousIPs := middleware.GetSuspiciousIPs()
	if len(suspiciousIPs) != 1 {
		t.Errorf("Expected 1 suspicious IP, got %d", len(suspiciousIPs))
	}

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	// Run cleanup
	middleware.Cleanup()

	// Should be cleaned up now
	suspiciousIPs = middleware.GetSuspiciousIPs()
	if len(suspiciousIPs) != 0 {
		t.Errorf("Expected 0 suspicious IPs after cleanup, got %d", len(suspiciousIPs))
	}
}