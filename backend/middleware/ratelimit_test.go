package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
	"qt1-middleware/config"
)

func TestNewTokenBucketLimiter(t *testing.T) {
	rl := NewTokenBucketLimiter(rate.Limit(10), 5)
	
	if rl == nil {
		t.Fatal("Expected rate limiter to be created")
	}
	
	if rl.limiter == nil {
		t.Fatal("Expected rate limiter to have internal limiter")
	}
	
	// Test that it allows requests initially
	if !rl.Allow() {
		t.Error("Expected rate limiter to allow initial request")
	}
}

func TestTokenBucketLimiterAllow(t *testing.T) {
	// Create a rate limiter that allows 2 requests per second with burst of 2
	rl := NewTokenBucketLimiter(rate.Limit(2), 2)
	
	// Should allow first two requests immediately
	if !rl.Allow() {
		t.Error("Expected first request to be allowed")
	}
	if !rl.Allow() {
		t.Error("Expected second request to be allowed")
	}
	
	// Third request should be denied (burst exhausted)
	if rl.Allow() {
		t.Error("Expected third request to be denied")
	}
}

func TestTokenBucketLimiterConcurrency(t *testing.T) {
	rl := NewTokenBucketLimiter(rate.Limit(100), 10)
	
	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex
	
	// Launch 20 concurrent requests
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow() {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}
	
	wg.Wait()
	
	// Should allow up to burst capacity
	if allowedCount > 10 {
		t.Errorf("Expected at most 10 requests to be allowed, got %d", allowedCount)
	}
}

func TestNewRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.RateLimitingConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name: "disabled config",
			config: &config.RateLimitingConfig{
				Enabled: false,
			},
			expected: false,
		},
		{
			name: "enabled config",
			config: &config.RateLimitingConfig{
				Enabled: true,
				Global: config.RateLimitRule{
					RequestsPerSecond: 10,
					Burst:             5,
				},
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rlm := NewRateLimitMiddleware(tt.config)
			if (rlm != nil) != tt.expected {
				t.Errorf("Expected middleware creation to be %v, got %v", tt.expected, rlm != nil)
			}
		})
	}
}

func TestRateLimitMiddlewareHandler(t *testing.T) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		Global: config.RateLimitRule{
			RequestsPerSecond: 2,
			Burst:             2,
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		t.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	// Create a simple handler that returns 200
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Wrap with rate limiting
	wrappedHandler := rlm.Handler(handler)
	
	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		
		wrappedHandler.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, rec.Code)
		}
		
		// Check rate limit headers
		if rec.Header().Get("X-RateLimit-Limit") == "" {
			t.Errorf("Request %d: missing X-RateLimit-Limit header", i+1)
		}
	}
	
	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rec.Code)
	}
	
	// Check rate limit error response
	var rateLimitErr RateLimitError
	if err := json.Unmarshal(rec.Body.Bytes(), &rateLimitErr); err != nil {
		t.Errorf("Failed to parse rate limit error response: %v", err)
	}
	
	if rateLimitErr.Message == "" {
		t.Error("Expected error message in rate limit response")
	}
	
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Missing Retry-After header")
	}
}

func TestRateLimitMiddlewareIPLimiting(t *testing.T) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		PerIP: config.RateLimitRule{
			RequestsPerMinute: 2,
			Burst:             1,
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		t.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	wrappedHandler := rlm.Handler(handler)
	
	// Test different IPs
	ips := []string{"192.168.1.1", "192.168.1.2"}
	
	for _, ip := range ips {
		// First request should succeed
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip + ":12345"
		rec := httptest.NewRecorder()
		
		wrappedHandler.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("IP %s: expected first request to succeed, got %d", ip, rec.Code)
		}
		
		// Second request should be rate limited (burst=1)
		req = httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = ip + ":12345"
		rec = httptest.NewRecorder()
		
		wrappedHandler.ServeHTTP(rec, req)
		
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("IP %s: expected second request to be rate limited, got %d", ip, rec.Code)
		}
	}
}

func TestRateLimitMiddlewareEndpointLimiting(t *testing.T) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		Endpoints: map[string]config.RateLimitRule{
			"/api/test": {
				RequestsPerMinute: 1,
				Burst:             1,
			},
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		t.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	wrappedHandler := rlm.Handler(handler)
	
	// Test rate limited endpoint
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected first request to succeed, got %d", rec.Code)
	}
	
	// Second request should be rate limited
	req = httptest.NewRequest("GET", "/api/test", nil)
	rec = httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected second request to be rate limited, got %d", rec.Code)
	}
	
	// Test non-rate limited endpoint
	req = httptest.NewRequest("GET", "/other", nil)
	rec = httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected request to non-limited endpoint to succeed, got %d", rec.Code)
	}
}

func TestRateLimitMiddlewareUserLimiting(t *testing.T) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		PerUser: config.RateLimitRule{
			RequestsPerMinute: 2,
			Burst:             1,
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		t.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	wrappedHandler := rlm.Handler(handler)
	
	// Test with user ID in context
	ctx := context.WithValue(context.Background(), "user_id", 123)
	
	// First request should succeed
	req := httptest.NewRequestWithContext(ctx, "GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected first request to succeed, got %d", rec.Code)
	}
	
	// Second request should be rate limited (burst=1)
	req = httptest.NewRequestWithContext(ctx, "GET", "/test", nil)
	rec = httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected second request to be rate limited, got %d", rec.Code)
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name           string
		xForwardedFor  string
		xRealIP        string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name:           "X-Forwarded-For header",
			xForwardedFor:  "192.168.1.100",
			remoteAddr:     "10.0.0.1:12345",
			expectedIP:     "192.168.1.100",
		},
		{
			name:       "X-Real-IP header",
			xRealIP:    "192.168.1.200",
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.200",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "192.168.1.300:12345",
			expectedIP: "192.168.1.300",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.400",
			expectedIP: "192.168.1.400",
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
			
			ip := extractClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name           string
		contextUserID  interface{}
		headerUserID   string
		expectedUserID int
	}{
		{
			name:           "User ID from context",
			contextUserID:  123,
			expectedUserID: 123,
		},
		{
			name:           "User ID from header",
			headerUserID:   "456",
			expectedUserID: 456,
		},
		{
			name:           "No user ID",
			expectedUserID: 0,
		},
		{
			name:           "Invalid header user ID",
			headerUserID:   "invalid",
			expectedUserID: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.contextUserID != nil {
				ctx = context.WithValue(ctx, "user_id", tt.contextUserID)
			}
			
			req := httptest.NewRequestWithContext(ctx, "GET", "/test", nil)
			if tt.headerUserID != "" {
				req.Header.Set("X-User-ID", tt.headerUserID)
			}
			
			userID := getUserID(req)
			if userID != tt.expectedUserID {
				t.Errorf("Expected user ID %d, got %d", tt.expectedUserID, userID)
			}
		})
	}
}

func TestRateLimitCleanup(t *testing.T) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		PerIP: config.RateLimitRule{
			RequestsPerMinute: 60,
			Burst:             10,
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		t.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	// Create some IP limiters
	_ = rlm.getIPLimiter("192.168.1.1")
	_ = rlm.getIPLimiter("192.168.1.2")
	
	// Check that limiters were created
	rlm.mutex.RLock()
	count := len(rlm.ipLimiters)
	rlm.mutex.RUnlock()
	
	if count != 2 {
		t.Errorf("Expected 2 IP limiters, got %d", count)
	}
	
	// Force cleanup by calling the cleanup method directly
	rlm.cleanupExpiredLimiters()
	
	// Limiters should still exist (they're recent)
	rlm.mutex.RLock()
	count = len(rlm.ipLimiters)
	rlm.mutex.RUnlock()
	
	if count != 2 {
		t.Errorf("Expected 2 IP limiters after cleanup, got %d", count)
	}
}

func TestRateLimitMiddlewareIntegration(t *testing.T) {
	// Test the full middleware stack with realistic configuration
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		Global: config.RateLimitRule{
			RequestsPerSecond: 10,
			Burst:             5,
		},
		PerIP: config.RateLimitRule{
			RequestsPerMinute: 30,
			Burst:             3,
		},
		Endpoints: map[string]config.RateLimitRule{
			"/api/chat": {
				RequestsPerMinute: 20,
				Burst:             2,
			},
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		t.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Success")
	})
	
	wrappedHandler := rlm.Handler(handler)
	
	// Test normal operation
	req := httptest.NewRequest("GET", "/api/other", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	
	wrappedHandler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected normal request to succeed, got %d", rec.Code)
	}
	
	// Test endpoint-specific limiting
	for i := 0; i < 3; i++ {
		req = httptest.NewRequest("GET", "/api/chat", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rec = httptest.NewRecorder()
		
		wrappedHandler.ServeHTTP(rec, req)
		
		if i < 2 {
			if rec.Code != http.StatusOK {
				t.Errorf("Chat request %d: expected success, got %d", i+1, rec.Code)
			}
		} else {
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("Chat request %d: expected rate limit, got %d", i+1, rec.Code)
			}
		}
	}
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		Global: config.RateLimitRule{
			RequestsPerSecond: 1000,
			Burst:             100,
		},
	}
	
	rlm := NewRateLimitMiddleware(cfg)
	if rlm == nil {
		b.Fatal("Failed to create rate limit middleware")
	}
	defer rlm.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	wrappedHandler := rlm.Handler(handler)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rec, req)
		}
	})
}

// Cycle 6B Tests

func TestNewSlidingWindowLimiter(t *testing.T) {
	swl := NewSlidingWindowLimiter(5, time.Minute)
	
	if swl == nil {
		t.Fatal("Expected sliding window limiter to be created")
	}
	
	if swl.limit != 5 {
		t.Errorf("Expected limit 5, got %d", swl.limit)
	}
	
	if swl.window != time.Minute {
		t.Errorf("Expected window 1 minute, got %v", swl.window)
	}
}

func TestSlidingWindowLimiterAllow(t *testing.T) {
	swl := NewSlidingWindowLimiter(3, time.Second)
	
	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !swl.Allow() {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}
	
	// Fourth request should be denied
	if swl.Allow() {
		t.Error("Expected fourth request to be denied")
	}
	
	// Wait for window to slide and try again
	time.Sleep(time.Second + 100*time.Millisecond)
	
	if !swl.Allow() {
		t.Error("Expected request after window slide to be allowed")
	}
}

func TestSlidingWindowLimiterGetRequestCount(t *testing.T) {
	swl := NewSlidingWindowLimiter(5, time.Second)
	
	// Add some requests
	for i := 0; i < 3; i++ {
		swl.Allow()
	}
	
	count := swl.GetRequestCount()
	if count != 3 {
		t.Errorf("Expected request count 3, got %d", count)
	}
	
	remaining := swl.GetRemaining()
	if remaining != 2 {
		t.Errorf("Expected remaining 2, got %d", remaining)
	}
}

func TestSlidingWindowLimiterCleanup(t *testing.T) {
	swl := NewSlidingWindowLimiter(100, time.Millisecond*100)
	
	// Add many requests
	for i := 0; i < 50; i++ {
		swl.Allow()
	}
	
	if len(swl.requests) != 50 {
		t.Errorf("Expected 50 requests, got %d", len(swl.requests))
	}
	
	// Wait for requests to expire
	time.Sleep(time.Millisecond * 150)
	
	// Cleanup should remove old requests
	swl.Cleanup()
	
	if len(swl.requests) > 0 {
		t.Errorf("Expected 0 requests after cleanup, got %d", len(swl.requests))
	}
}

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	defer storage.Close()
	
	key := "test_key"
	data := &RateLimitData{
		Count:     5,
		LastReset: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Test Set and Get
	err := storage.Set(key, data, time.Minute)
	if err != nil {
		t.Errorf("Failed to set data: %v", err)
	}
	
	retrieved, err := storage.Get(key)
	if err != nil {
		t.Errorf("Failed to get data: %v", err)
	}
	
	if retrieved == nil {
		t.Fatal("Expected data to be retrieved")
	}
	
	if retrieved.Count != 5 {
		t.Errorf("Expected count 5, got %d", retrieved.Count)
	}
	
	// Test Delete
	err = storage.Delete(key)
	if err != nil {
		t.Errorf("Failed to delete data: %v", err)
	}
	
	retrieved, err = storage.Get(key)
	if err != nil {
		t.Errorf("Failed to get deleted data: %v", err)
	}
	
	if retrieved != nil {
		t.Error("Expected deleted data to be nil")
	}
}

func TestMemoryStorageIncrement(t *testing.T) {
	storage := NewMemoryStorage()
	defer storage.Close()
	
	key := "counter_key"
	window := time.Second
	
	// First increment should return 1
	count, err := storage.Increment(key, window)
	if err != nil {
		t.Errorf("Failed to increment: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	
	// Second increment should return 2
	count, err = storage.Increment(key, window)
	if err != nil {
		t.Errorf("Failed to increment: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	
	// Wait for window to expire and increment again
	time.Sleep(window + 100*time.Millisecond)
	count, err = storage.Increment(key, window)
	if err != nil {
		t.Errorf("Failed to increment after window: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1 after window reset, got %d", count)
	}
}

func TestMemoryStorageRequestTimes(t *testing.T) {
	storage := NewMemoryStorage()
	defer storage.Close()
	
	key := "times_key"
	window := time.Second
	now := time.Now()
	
	// Add some request times
	times := []time.Time{
		now.Add(-500 * time.Millisecond),
		now.Add(-300 * time.Millisecond),
		now,
	}
	
	for _, requestTime := range times {
		err := storage.AddRequestTime(key, requestTime, window)
		if err != nil {
			t.Errorf("Failed to add request time: %v", err)
		}
	}
	
	// Get request times within window
	retrieved, err := storage.GetRequestTimes(key, window)
	if err != nil {
		t.Errorf("Failed to get request times: %v", err)
	}
	
	if len(retrieved) != 3 {
		t.Errorf("Expected 3 request times, got %d", len(retrieved))
	}
	
	// Get request times with smaller window
	retrieved, err = storage.GetRequestTimes(key, 200*time.Millisecond)
	if err != nil {
		t.Errorf("Failed to get request times: %v", err)
	}
	
	// Should only include the most recent request
	if len(retrieved) != 1 {
		t.Errorf("Expected 1 request time in smaller window, got %d", len(retrieved))
	}
}

func TestMultiTierRateLimiter(t *testing.T) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		Storage: config.RateLimitStorageConfig{
			Type: "memory",
		},
		Global: config.RateLimitRule{
			RequestsPerSecond: 10,
			Algorithm:         "token_bucket",
		},
		PerIP: config.RateLimitRule{
			RequestsPerMinute: 5,
			Algorithm:         "sliding_window",
			WindowSize:        time.Minute,
		},
		PerUser: config.RateLimitRule{
			RequestsPerMinute: 8,
			Algorithm:         "sliding_window",
			WindowSize:        time.Minute,
		},
		Endpoints: map[string]config.RateLimitRule{
			"/api/test": {
				RequestsPerMinute: 3,
				Algorithm:         "sliding_window",
				WindowSize:        time.Minute,
			},
		},
	}
	
	mtrl, err := NewMultiTierRateLimiter(cfg)
	if err != nil {
		t.Fatalf("Failed to create multi-tier rate limiter: %v", err)
	}
	defer mtrl.Close()
	
	// Test with different requests
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	
	req2 := httptest.NewRequest("GET", "/api/other", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	
	// First requests should be allowed
	if err := mtrl.CheckRateLimit(req1, 123); err != nil {
		t.Errorf("Expected first request to be allowed: %v", err)
	}
	
	if err := mtrl.CheckRateLimit(req2, 456); err != nil {
		t.Errorf("Expected second request to be allowed: %v", err)
	}
	
	// Test endpoint-specific limiting
	for i := 0; i < 3; i++ {
		err := mtrl.CheckRateLimit(req1, 123)
		if i < 2 && err != nil {
			t.Errorf("Request %d should be allowed: %v", i+1, err)
		}
		if i >= 2 && err == nil {
			t.Errorf("Request %d should be rate limited", i+1)
		}
	}
}

func TestCreateRateLimiterAlgorithms(t *testing.T) {
	// Test sliding window limiter creation
	swRule := config.RateLimitRule{
		RequestsPerMinute: 10,
		Algorithm:         "sliding_window",
		WindowSize:        time.Minute,
	}
	
	swLimiter := createRateLimiter(swRule)
	if _, ok := swLimiter.(*SlidingWindowLimiter); !ok {
		t.Error("Expected sliding window limiter")
	}
	
	// Test token bucket limiter creation
	tbRule := config.RateLimitRule{
		RequestsPerSecond: 5,
		Burst:             10,
		Algorithm:         "token_bucket",
	}
	
	tbLimiter := createRateLimiter(tbRule)
	if _, ok := tbLimiter.(*TokenBucketLimiter); !ok {
		t.Error("Expected token bucket limiter")
	}
}

func BenchmarkSlidingWindowLimiter(b *testing.B) {
	swl := NewSlidingWindowLimiter(1000, time.Minute)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			swl.Allow()
		}
	})
}

func BenchmarkMultiTierRateLimiter(b *testing.B) {
	cfg := &config.RateLimitingConfig{
		Enabled: true,
		Storage: config.RateLimitStorageConfig{
			Type: "memory",
		},
		Global: config.RateLimitRule{
			RequestsPerSecond: 10000,
			Algorithm:         "token_bucket",
		},
		PerIP: config.RateLimitRule{
			RequestsPerMinute: 6000,
			Algorithm:         "sliding_window",
			WindowSize:        time.Minute,
		},
	}
	
	mtrl, err := NewMultiTierRateLimiter(cfg)
	if err != nil {
		b.Fatalf("Failed to create multi-tier rate limiter: %v", err)
	}
	defer mtrl.Close()
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mtrl.CheckRateLimit(req, 123)
		}
	})
}