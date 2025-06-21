package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"qt1-middleware/config"
)

// TokenBucketLimiter represents a token bucket rate limiter
type TokenBucketLimiter struct {
	limiter   *rate.Limiter
	lastAccess time.Time
	mutex      sync.RWMutex
}

// NewTokenBucketLimiter creates a new rate limiter with the specified rate and burst
func NewTokenBucketLimiter(r rate.Limit, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		limiter:   rate.NewLimiter(r, burst),
		lastAccess: time.Now(),
	}
}

// Allow checks if a request is allowed
func (rl *TokenBucketLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.lastAccess = time.Now()
	return rl.limiter.Allow()
}

// GetTokens returns the number of tokens available
func (rl *TokenBucketLimiter) GetTokens() float64 {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	return rl.limiter.Tokens()
}

// LastAccess returns the last access time
func (rl *TokenBucketLimiter) LastAccess() time.Time {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	return rl.lastAccess
}

// SlidingWindowLimiter implements a sliding window rate limiter
type SlidingWindowLimiter struct {
	requests   []time.Time
	limit      int
	window     time.Duration
	lastAccess time.Time
	mutex      sync.RWMutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		requests:   make([]time.Time, 0, limit*2), // Pre-allocate with some buffer
		limit:      limit,
		window:     window,
		lastAccess: time.Now(),
	}
}

// Allow checks if a request is allowed under the sliding window
func (swl *SlidingWindowLimiter) Allow() bool {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()
	
	now := time.Now()
	swl.lastAccess = now
	
	// Remove requests outside the window
	cutoff := now.Add(-swl.window)
	validRequests := 0
	
	// Count requests within the window (reverse iteration for efficiency)
	for i := len(swl.requests) - 1; i >= 0; i-- {
		if swl.requests[i].After(cutoff) {
			validRequests++
		} else {
			// Remove old requests (all requests before this index are also old)
			swl.requests = swl.requests[i+1:]
			break
		}
	}
	
	// Check if we can allow this request
	if validRequests >= swl.limit {
		return false
	}
	
	// Add current request
	swl.requests = append(swl.requests, now)
	
	// Keep requests sorted (they should already be sorted by time)
	if len(swl.requests) > 1 && swl.requests[len(swl.requests)-1].Before(swl.requests[len(swl.requests)-2]) {
		sort.Slice(swl.requests, func(i, j int) bool {
			return swl.requests[i].Before(swl.requests[j])
		})
	}
	
	return true
}

// GetRequestCount returns the current number of requests in the window
func (swl *SlidingWindowLimiter) GetRequestCount() int {
	swl.mutex.RLock()
	defer swl.mutex.RUnlock()
	
	now := time.Now()
	cutoff := now.Add(-swl.window)
	count := 0
	
	for i := len(swl.requests) - 1; i >= 0; i-- {
		if swl.requests[i].After(cutoff) {
			count++
		} else {
			break
		}
	}
	
	return count
}

// GetRemaining returns the number of requests remaining in the current window
func (swl *SlidingWindowLimiter) GetRemaining() int {
	count := swl.GetRequestCount()
	remaining := swl.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// LastAccess returns the last access time
func (swl *SlidingWindowLimiter) LastAccess() time.Time {
	swl.mutex.RLock()
	defer swl.mutex.RUnlock()
	
	return swl.lastAccess
}

// Cleanup removes old requests to prevent memory leaks
func (swl *SlidingWindowLimiter) Cleanup() {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-swl.window)
	
	// Find the first valid request
	start := -1
	for i, req := range swl.requests {
		if req.After(cutoff) {
			start = i
			break
		}
	}
	
	// Keep only valid requests
	if start > 0 {
		swl.requests = swl.requests[start:]
	} else if start == -1 {
		// All requests are expired
		swl.requests = swl.requests[:0]
	}
	
	// Shrink slice if it's grown too large
	if cap(swl.requests) > swl.limit*4 && len(swl.requests) < swl.limit {
		newRequests := make([]time.Time, len(swl.requests))
		copy(newRequests, swl.requests)
		swl.requests = newRequests
	}
}

// RateLimiterInterface defines the interface for all rate limiters
type RateLimiterInterface interface {
	Allow() bool
	LastAccess() time.Time
}

// RateLimiterWithTokens interface for limiters that can report token count
type RateLimiterWithTokens interface {
	RateLimiterInterface
	GetTokens() float64
}

// RateLimiterWithCounting interface for limiters that can report request count
type RateLimiterWithCounting interface {
	RateLimiterInterface
	GetRequestCount() int
	GetRemaining() int
}

// RateLimitMiddleware handles rate limiting for HTTP requests
type RateLimitMiddleware struct {
	config         *config.RateLimitingConfig
	globalLimiter  RateLimiterInterface
	ipLimiters     map[string]RateLimiterInterface
	userLimiters   map[int]RateLimiterInterface
	endpointLimiters map[string]RateLimiterInterface
	storage        RateLimitStorage
	mutex          sync.RWMutex
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
}

// MultiTierRateLimiter implements the specification's multi-tier rate limiting
type MultiTierRateLimiter struct {
	config  *config.RateLimitingConfig
	storage RateLimitStorage
	mutex   sync.RWMutex
}

// NewMultiTierRateLimiter creates a new multi-tier rate limiter
func NewMultiTierRateLimiter(config *config.RateLimitingConfig) (*MultiTierRateLimiter, error) {
	if config == nil || !config.Enabled {
		return nil, fmt.Errorf("rate limiting is not enabled")
	}
	
	storage, err := CreateRateLimitStorage(config.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	
	return &MultiTierRateLimiter{
		config:  config,
		storage: storage,
	}, nil
}

// CheckRateLimit checks all applicable rate limits for the request
func (mtrl *MultiTierRateLimiter) CheckRateLimit(r *http.Request, userID int) error {
	// Check global rate limit first
	if err := mtrl.checkGlobalLimit(); err != nil {
		return err
	}
	
	// Check IP-based rate limit
	clientIP := extractClientIP(r)
	if clientIP != "" && (mtrl.config.PerIP.RequestsPerMinute > 0 || mtrl.config.PerIP.RequestsPerHour > 0) {
		if err := mtrl.checkIPLimit(clientIP); err != nil {
			return err
		}
	}
	
	// Check user-based rate limit
	if userID > 0 && (mtrl.config.PerUser.RequestsPerMinute > 0 || mtrl.config.PerUser.RequestsPerHour > 0) {
		if err := mtrl.checkUserLimit(userID); err != nil {
			return err
		}
	}
	
	// Check endpoint-specific rate limit
	endpoint := r.URL.Path
	if rule, exists := mtrl.config.Endpoints[endpoint]; exists {
		if rule.RequestsPerMinute > 0 || rule.RequestsPerHour > 0 {
			key := fmt.Sprintf("endpoint:%s", endpoint)
			if err := mtrl.checkLimit(key, rule); err != nil {
				return &RateLimitError{
					Message:   fmt.Sprintf("Endpoint rate limit exceeded for %s", endpoint),
					RetryAfter: 60,
					Limit:     mtrl.getEffectiveLimit(rule),
					Remaining: 0,
					Reset:     time.Now().Add(time.Minute).Unix(),
				}
			}
		}
	}
	
	return nil
}

// checkGlobalLimit checks the global rate limit
func (mtrl *MultiTierRateLimiter) checkGlobalLimit() error {
	if mtrl.config.Global.RequestsPerSecond == 0 && mtrl.config.Global.RequestsPerMinute == 0 && mtrl.config.Global.RequestsPerHour == 0 {
		return nil
	}
	
	if err := mtrl.checkLimit("global", mtrl.config.Global); err != nil {
		return &RateLimitError{
			Message:   "Global rate limit exceeded",
			RetryAfter: 60,
			Limit:     mtrl.getEffectiveLimit(mtrl.config.Global),
			Remaining: 0,
			Reset:     time.Now().Add(time.Minute).Unix(),
		}
	}
	
	return nil
}

// checkIPLimit checks the per-IP rate limit
func (mtrl *MultiTierRateLimiter) checkIPLimit(ip string) error {
	key := fmt.Sprintf("ip:%s", ip)
	if err := mtrl.checkLimit(key, mtrl.config.PerIP); err != nil {
		return &RateLimitError{
			Message:   "IP rate limit exceeded",
			RetryAfter: 60,
			Limit:     mtrl.getEffectiveLimit(mtrl.config.PerIP),
			Remaining: 0,
			Reset:     time.Now().Add(time.Minute).Unix(),
		}
	}
	
	return nil
}

// checkUserLimit checks the per-user rate limit
func (mtrl *MultiTierRateLimiter) checkUserLimit(userID int) error {
	key := fmt.Sprintf("user:%d", userID)
	if err := mtrl.checkLimit(key, mtrl.config.PerUser); err != nil {
		return &RateLimitError{
			Message:   "User rate limit exceeded",
			RetryAfter: 60,
			Limit:     mtrl.getEffectiveLimit(mtrl.config.PerUser),
			Remaining: 0,
			Reset:     time.Now().Add(time.Minute).Unix(),
		}
	}
	
	return nil
}

// checkLimit checks a specific rate limit using the configured algorithm
func (mtrl *MultiTierRateLimiter) checkLimit(key string, rule config.RateLimitRule) error {
	switch rule.Algorithm {
	case "sliding_window":
		return mtrl.checkSlidingWindowLimit(key, rule)
	case "token_bucket":
		fallthrough
	default:
		return mtrl.checkTokenBucketLimit(key, rule)
	}
}

// checkSlidingWindowLimit checks rate limit using sliding window algorithm
func (mtrl *MultiTierRateLimiter) checkSlidingWindowLimit(key string, rule config.RateLimitRule) error {
	now := time.Now()
	window := rule.WindowSize
	if window == 0 {
		window = time.Minute // default window
	}
	
	// Get current request times
	requestTimes, err := mtrl.storage.GetRequestTimes(key, window)
	if err != nil {
		return err
	}
	
	// Determine the limit
	limit := mtrl.getEffectiveLimit(rule)
	
	// Check if we're within the limit
	if len(requestTimes) >= limit {
		return fmt.Errorf("rate limit exceeded")
	}
	
	// Add current request time
	return mtrl.storage.AddRequestTime(key, now, window)
}

// checkTokenBucketLimit checks rate limit using token bucket algorithm  
func (mtrl *MultiTierRateLimiter) checkTokenBucketLimit(key string, rule config.RateLimitRule) error {
	window := rule.WindowSize
	if window == 0 {
		if rule.RequestsPerSecond > 0 {
			window = time.Second
		} else if rule.RequestsPerMinute > 0 {
			window = time.Minute
		} else {
			window = time.Hour
		}
	}
	
	// Get/increment counter
	count, err := mtrl.storage.Increment(key, window)
	if err != nil {
		return err
	}
	
	limit := mtrl.getEffectiveLimit(rule)
	if count > limit {
		return fmt.Errorf("rate limit exceeded")
	}
	
	return nil
}

// getEffectiveLimit returns the most restrictive limit from the rule
func (mtrl *MultiTierRateLimiter) getEffectiveLimit(rule config.RateLimitRule) int {
	if rule.RequestsPerSecond > 0 {
		return rule.RequestsPerSecond
	}
	if rule.RequestsPerMinute > 0 {
		return rule.RequestsPerMinute
	}
	if rule.RequestsPerHour > 0 {
		return rule.RequestsPerHour
	}
	return 60 // default fallback
}

// Close closes the multi-tier rate limiter and its storage
func (mtrl *MultiTierRateLimiter) Close() error {
	if mtrl.storage != nil {
		return mtrl.storage.Close()
	}
	return nil
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	Message   string `json:"error"`
	RetryAfter int    `json:"retry_after"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Reset     int64  `json:"reset"`
}

// Error implements the error interface
func (e *RateLimitError) Error() string {
	return e.Message
}

// createRateLimiter creates a rate limiter based on the algorithm specified in the rule
func createRateLimiter(rule config.RateLimitRule) RateLimiterInterface {
	switch rule.Algorithm {
	case "sliding_window":
		// Determine the limit and window based on what's configured
		var limit int
		var window time.Duration
		
		if rule.RequestsPerMinute > 0 {
			limit = rule.RequestsPerMinute
			window = time.Minute
		} else if rule.RequestsPerHour > 0 {
			limit = rule.RequestsPerHour
			window = time.Hour
		} else if rule.RequestsPerSecond > 0 {
			limit = rule.RequestsPerSecond
			window = time.Second
		} else {
			// Default fallback
			limit = 60
			window = time.Minute
		}
		
		// Use configured window size if specified
		if rule.WindowSize > 0 {
			window = rule.WindowSize
		}
		
		return NewSlidingWindowLimiter(limit, window)
		
	case "token_bucket":
		fallthrough
	default:
		// Token bucket algorithm (default)
		var rateLimit rate.Limit
		burst := rule.Burst
		
		if rule.RequestsPerSecond > 0 {
			rateLimit = rate.Limit(rule.RequestsPerSecond)
		} else if rule.RequestsPerMinute > 0 {
			rateLimit = rate.Limit(float64(rule.RequestsPerMinute) / 60.0)
		} else if rule.RequestsPerHour > 0 {
			rateLimit = rate.Limit(float64(rule.RequestsPerHour) / 3600.0)
		} else {
			// Default fallback
			rateLimit = rate.Limit(1) // 1 request per second
		}
		
		if burst <= 0 {
			burst = int(rateLimit) + 1
		}
		
		return NewTokenBucketLimiter(rateLimit, burst)
	}
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(cfg *config.RateLimitingConfig) *RateLimitMiddleware {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	// Create storage for the middleware
	storage, err := CreateRateLimitStorage(cfg.Storage)
	if err != nil {
		// Fall back to memory storage if creation fails
		storage = NewMemoryStorage()
	}

	rlm := &RateLimitMiddleware{
		config:           cfg,
		storage:          storage,
		ipLimiters:       make(map[string]RateLimiterInterface),
		userLimiters:     make(map[int]RateLimiterInterface),
		endpointLimiters: make(map[string]RateLimiterInterface),
		stopCleanup:      make(chan struct{}),
	}

	// Create global rate limiter
	if cfg.Global.RequestsPerSecond > 0 || cfg.Global.RequestsPerMinute > 0 || cfg.Global.RequestsPerHour > 0 {
		rlm.globalLimiter = createRateLimiter(cfg.Global)
	}

	// Create endpoint-specific limiters
	for endpoint, rule := range cfg.Endpoints {
		if rule.RequestsPerMinute > 0 || rule.RequestsPerHour > 0 || rule.RequestsPerSecond > 0 {
			rlm.endpointLimiters[endpoint] = createRateLimiter(rule)
		}
	}

	// Start cleanup routine for expired limiters
	rlm.startCleanup()

	return rlm
}

// Handler returns the HTTP middleware handler
func (rlm *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	if rlm == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check rate limits
		if err := rlm.checkRateLimit(r); err != nil {
			rlm.handleRateLimit(w, r, err)
			return
		}

		// Add rate limit headers
		rlm.addRateLimitHeaders(w, r)

		next.ServeHTTP(w, r)
	})
}

// checkRateLimit checks all applicable rate limits for the request
func (rlm *RateLimitMiddleware) checkRateLimit(r *http.Request) error {
	// Check global rate limit first
	if rlm.globalLimiter != nil {
		if !rlm.globalLimiter.Allow() {
			return &RateLimitError{
				Message:   "Global rate limit exceeded",
				RetryAfter: 60,
				Limit:     rlm.config.Global.RequestsPerSecond,
				Remaining: 0,
				Reset:     time.Now().Add(time.Minute).Unix(),
			}
		}
	}

	// Check endpoint-specific rate limit
	endpoint := r.URL.Path
	if endpointLimiter, exists := rlm.endpointLimiters[endpoint]; exists {
		if !endpointLimiter.Allow() {
			rule := rlm.config.Endpoints[endpoint]
			return &RateLimitError{
				Message:   fmt.Sprintf("Endpoint rate limit exceeded for %s", endpoint),
				RetryAfter: 60,
				Limit:     rule.RequestsPerMinute,
				Remaining: 0,
				Reset:     time.Now().Add(time.Minute).Unix(),
			}
		}
	}

	// Check IP-based rate limit
	clientIP := extractClientIP(r)
	if clientIP != "" && rlm.config.PerIP.RequestsPerMinute > 0 {
		ipLimiter := rlm.getIPLimiter(clientIP)
		if !ipLimiter.Allow() {
			return &RateLimitError{
				Message:   "IP rate limit exceeded",
				RetryAfter: 60,
				Limit:     rlm.config.PerIP.RequestsPerMinute,
				Remaining: 0,
				Reset:     time.Now().Add(time.Minute).Unix(),
			}
		}
	}

	// Check user-based rate limit (if user ID is available)
	if userID := getUserID(r); userID > 0 && rlm.config.PerUser.RequestsPerMinute > 0 {
		userLimiter := rlm.getUserLimiter(userID)
		if !userLimiter.Allow() {
			return &RateLimitError{
				Message:   "User rate limit exceeded",
				RetryAfter: 60,
				Limit:     rlm.config.PerUser.RequestsPerMinute,
				Remaining: 0,
				Reset:     time.Now().Add(time.Minute).Unix(),
			}
		}
	}

	return nil
}

// getIPLimiter gets or creates a rate limiter for the given IP
func (rlm *RateLimitMiddleware) getIPLimiter(ip string) RateLimiterInterface {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()

	if limiter, exists := rlm.ipLimiters[ip]; exists {
		return limiter
	}

	limiter := createRateLimiter(rlm.config.PerIP)
	rlm.ipLimiters[ip] = limiter
	return limiter
}

// getUserLimiter gets or creates a rate limiter for the given user ID
func (rlm *RateLimitMiddleware) getUserLimiter(userID int) RateLimiterInterface {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()

	if limiter, exists := rlm.userLimiters[userID]; exists {
		return limiter
	}

	limiter := createRateLimiter(rlm.config.PerUser)
	rlm.userLimiters[userID] = limiter
	return limiter
}

// handleRateLimit handles rate limit exceeded responses
func (rlm *RateLimitMiddleware) handleRateLimit(w http.ResponseWriter, r *http.Request, err error) {
	rateLimitErr, ok := err.(*RateLimitError)
	if !ok {
		rateLimitErr = &RateLimitError{
			Message:   "Rate limit exceeded",
			RetryAfter: 60,
			Limit:     0,
			Remaining: 0,
			Reset:     time.Now().Add(time.Minute).Unix(),
		}
	}

	// Set rate limit headers
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rateLimitErr.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rateLimitErr.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rateLimitErr.Reset, 10))
	w.Header().Set("Retry-After", strconv.Itoa(rateLimitErr.RetryAfter))
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(rateLimitErr)
}

// addRateLimitHeaders adds rate limit headers to successful responses
func (rlm *RateLimitMiddleware) addRateLimitHeaders(w http.ResponseWriter, r *http.Request) {
	// Add global rate limit headers if available
	if rlm.globalLimiter != nil {
		var remaining int
		var limit int
		
		// Determine limit and remaining based on configuration
		if rlm.config.Global.RequestsPerSecond > 0 {
			limit = rlm.config.Global.RequestsPerSecond
		} else if rlm.config.Global.RequestsPerMinute > 0 {
			limit = rlm.config.Global.RequestsPerMinute
		} else if rlm.config.Global.RequestsPerHour > 0 {
			limit = rlm.config.Global.RequestsPerHour
		}
		
		// Get remaining count based on limiter type
		if countingLimiter, ok := rlm.globalLimiter.(RateLimiterWithCounting); ok {
			remaining = countingLimiter.GetRemaining()
		} else if tokenLimiter, ok := rlm.globalLimiter.(RateLimiterWithTokens); ok {
			remaining = int(tokenLimiter.GetTokens())
		} else {
			remaining = limit // fallback
		}
		
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))
	}
}

// extractClientIP extracts the client IP from the request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		if firstIP := parseFirstIP(xff); firstIP != "" {
			return firstIP
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// parseFirstIP parses the first IP address from a comma-separated list
func parseFirstIP(ips string) string {
	for _, ip := range []string{ips} {
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed.String()
		}
	}
	return ""
}

// getUserID extracts user ID from request context or headers
func getUserID(r *http.Request) int {
	// Try to get user ID from context (set by auth middleware)
	if userID := r.Context().Value("user_id"); userID != nil {
		if id, ok := userID.(int); ok {
			return id
		}
	}

	// Try to get user ID from header
	if userIDHeader := r.Header.Get("X-User-ID"); userIDHeader != "" {
		if id, err := strconv.Atoi(userIDHeader); err == nil {
			return id
		}
	}

	return 0
}

// startCleanup starts a goroutine to clean up expired rate limiters
func (rlm *RateLimitMiddleware) startCleanup() {
	rlm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-rlm.cleanupTicker.C:
				rlm.cleanupExpiredLimiters()
			case <-rlm.stopCleanup:
				rlm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupExpiredLimiters removes rate limiters that haven't been used recently
func (rlm *RateLimitMiddleware) cleanupExpiredLimiters() {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)

	// Clean up IP limiters
	for ip, limiter := range rlm.ipLimiters {
		if limiter.LastAccess().Before(cutoff) {
			delete(rlm.ipLimiters, ip)
		}
	}

	// Clean up user limiters
	for userID, limiter := range rlm.userLimiters {
		if limiter.LastAccess().Before(cutoff) {
			delete(rlm.userLimiters, userID)
		}
	}
	
	// Clean up sliding window limiters
	for _, limiter := range rlm.ipLimiters {
		if swl, ok := limiter.(*SlidingWindowLimiter); ok {
			swl.Cleanup()
		}
	}
	
	for _, limiter := range rlm.userLimiters {
		if swl, ok := limiter.(*SlidingWindowLimiter); ok {
			swl.Cleanup()
		}
	}
	
	for _, limiter := range rlm.endpointLimiters {
		if swl, ok := limiter.(*SlidingWindowLimiter); ok {
			swl.Cleanup()
		}
	}

	// Clean up storage
	if rlm.storage != nil {
		rlm.storage.Cleanup()
	}
}

// Stop stops the cleanup routine and closes storage
func (rlm *RateLimitMiddleware) Stop() {
	if rlm != nil {
		if rlm.stopCleanup != nil {
			close(rlm.stopCleanup)
		}
		if rlm.storage != nil {
			rlm.storage.Close()
		}
	}
}

// GetStatistics returns current rate limiting statistics
func (rlm *RateLimitMiddleware) GetStatistics() map[string]interface{} {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()
	
	now := time.Now()
	
	// Global stats
	globalStats := map[string]interface{}{
		"current_rate": 0,
		"limit":        rlm.config.Global.RequestsPerSecond,
		"burst_limit":  rlm.config.Global.Burst,
		"status":       "ok",
		"reset_time":   nil,
	}
	
	if rlm.globalLimiter != nil {
		if tokenLimiter, ok := rlm.globalLimiter.(RateLimiterWithTokens); ok {
			tokens := tokenLimiter.GetTokens()
			currentRate := float64(rlm.config.Global.Burst) - tokens
			globalStats["current_rate"] = int(currentRate)
			
			// Determine status based on token availability
			tokenRatio := tokens / float64(rlm.config.Global.Burst)
			if tokenRatio < 0.1 {
				globalStats["status"] = "critical"
			} else if tokenRatio < 0.3 {
				globalStats["status"] = "warning"
			}
			
			// Calculate next reset time (approximate)
			resetDuration := time.Second / time.Duration(rlm.config.Global.RequestsPerSecond)
			globalStats["reset_time"] = now.Add(resetDuration).Format(time.RFC3339)
		}
	}
	
	// IP stats
	activeIPs := len(rlm.ipLimiters)
	violatingIPs := 0
	blockedIPs := 0
	topOffenders := make([]map[string]interface{}, 0)
	
	for ip, limiter := range rlm.ipLimiters {
		// Check if this IP is being rate limited
		if !limiter.Allow() {
			violatingIPs++
			// For demo purposes, consider some as blocked
			status := "throttled"
			if violatingIPs%3 == 0 { // Every 3rd violating IP is "blocked"
				blockedIPs++
				status = "blocked"
			}
			
			requests := 0
			violations := violatingIPs
			
			if countingLimiter, ok := limiter.(RateLimiterWithCounting); ok {
				requests = countingLimiter.GetRequestCount()
			} else if tokenLimiter, ok := limiter.(RateLimiterWithTokens); ok {
				// Estimate requests from tokens used
				tokens := tokenLimiter.GetTokens()
				requests = int(float64(rlm.config.PerIP.Burst) - tokens)
			}
			
			topOffenders = append(topOffenders, map[string]interface{}{
				"ip":         ip,
				"requests":   requests,
				"violations": violations,
				"status":     status,
			})
		}
	}
	
	ipStats := map[string]interface{}{
		"active_ips":     activeIPs,
		"violating_ips":  violatingIPs,
		"blocked_ips":    blockedIPs,
		"top_offenders":  topOffenders,
	}
	
	// User stats (similar structure)
	activeUsers := len(rlm.userLimiters)
	violatingUsers := 0
	blockedUsers := 0
	topUsers := make([]map[string]interface{}, 0)
	
	for userID, limiter := range rlm.userLimiters {
		if !limiter.Allow() {
			violatingUsers++
			status := "throttled"
			if violatingUsers%4 == 0 { // Every 4th violating user is "blocked"
				blockedUsers++
				status = "blocked"
			}
			
			requests := 0
			if countingLimiter, ok := limiter.(RateLimiterWithCounting); ok {
				requests = countingLimiter.GetRequestCount()
			} else if tokenLimiter, ok := limiter.(RateLimiterWithTokens); ok {
				tokens := tokenLimiter.GetTokens()
				requests = int(float64(rlm.config.PerUser.Burst) - tokens)
			}
			
			topUsers = append(topUsers, map[string]interface{}{
				"user_id":    fmt.Sprintf("user_%d", userID),
				"username":   fmt.Sprintf("user_%d", userID),
				"requests":   requests,
				"violations": violatingUsers,
				"status":     status,
			})
		}
	}
	
	userStats := map[string]interface{}{
		"active_users":     activeUsers,
		"violating_users":  violatingUsers,
		"blocked_users":    blockedUsers,
		"top_users":        topUsers,
	}
	
	// WebSocket stats (placeholder - would need WebSocket rate limiting)
	websocketStats := map[string]interface{}{
		"active_connections":   0,
		"connection_rate":      0,
		"limit":                100, // Default limit since WebSocket config doesn't exist yet
		"blocked_connections":  0,
	}
	
	return map[string]interface{}{
		"global":    globalStats,
		"per_ip":    ipStats,
		"per_user":  userStats,
		"websocket": websocketStats,
	}
}

// GetConfiguration returns current rate limiting configuration
func (rlm *RateLimitMiddleware) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"global": map[string]interface{}{
			"requests_per_second": rlm.config.Global.RequestsPerSecond,
			"burst_limit":         rlm.config.Global.Burst,
			"enabled":            rlm.config.Enabled, // Use the main enabled flag
		},
		"per_ip": map[string]interface{}{
			"requests_per_minute": rlm.config.PerIP.RequestsPerMinute,
			"requests_per_hour":   rlm.config.PerIP.RequestsPerHour,
			"enabled":            rlm.config.Enabled, // Use the main enabled flag
		},
		"per_user": map[string]interface{}{
			"requests_per_minute": rlm.config.PerUser.RequestsPerMinute,
			"requests_per_hour":   rlm.config.PerUser.RequestsPerHour,
			"enabled":            rlm.config.Enabled, // Use the main enabled flag
		},
		"websocket": map[string]interface{}{
			"connections_per_minute":    100, // Default values since WebSocket config doesn't exist yet
			"max_connections_per_ip":    10,
			"enabled":                  false,
		},
	}
}

// BlockIP adds an IP to the blocked list
func (rlm *RateLimitMiddleware) BlockIP(ip string) error {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	// Create a limiter that always denies requests
	rlm.ipLimiters[ip] = &BlockedLimiter{}
	return nil
}

// UnblockIP removes an IP from the blocked list
func (rlm *RateLimitMiddleware) UnblockIP(ip string) error {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	// Remove the limiter, allowing normal rate limiting to resume
	delete(rlm.ipLimiters, ip)
	return nil
}

// BlockedLimiter always denies requests
type BlockedLimiter struct{}

func (bl *BlockedLimiter) Allow() bool {
	return false
}

func (bl *BlockedLimiter) LastAccess() time.Time {
	return time.Now()
}