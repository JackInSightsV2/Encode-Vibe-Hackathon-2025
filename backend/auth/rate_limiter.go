package auth

import (
	"sync"
	"time"
)

// RateLimiter implements rate limiting for session operations
type RateLimiter struct {
	clients map[string]*ClientBucket
	mutex   sync.RWMutex
	config  *RateLimitConfig
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxRequests     int           `yaml:"max_requests" json:"max_requests"`
	WindowDuration  time.Duration `yaml:"window_duration" json:"window_duration"`
	BanDuration     time.Duration `yaml:"ban_duration" json:"ban_duration"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// ClientBucket tracks requests for a specific client
type ClientBucket struct {
	requests    []time.Time
	bannedUntil time.Time
	violations  int
	mutex       sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	config := &RateLimitConfig{
		MaxRequests:     10,                // 10 requests
		WindowDuration:  1 * time.Minute,  // per minute
		BanDuration:     15 * time.Minute, // ban for 15 minutes
		CleanupInterval: 5 * time.Minute,  // cleanup every 5 minutes
	}

	rl := &RateLimiter{
		clients: make(map[string]*ClientBucket),
		config:  config,
	}

	// Start cleanup goroutine
	go rl.startCleanup()

	return rl
}

// Allow checks if a request from the given client should be allowed
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Get or create client bucket
	bucket, exists := rl.clients[clientID]
	if !exists {
		bucket = &ClientBucket{
			requests: make([]time.Time, 0),
		}
		rl.clients[clientID] = bucket
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	// Check if client is currently banned
	if now.Before(bucket.bannedUntil) {
		return false
	}

	// Remove requests outside the current window
	cutoff := now.Add(-rl.config.WindowDuration)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range bucket.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	bucket.requests = validRequests

	// Check if adding this request would exceed the limit
	if len(bucket.requests) >= rl.config.MaxRequests {
		// Rate limit exceeded
		bucket.violations++
		
		// Ban the client if too many violations
		if bucket.violations >= 3 {
			bucket.bannedUntil = now.Add(rl.config.BanDuration)
		}
		
		return false
	}

	// Add the current request
	bucket.requests = append(bucket.requests, now)
	
	// Reset violations on successful request
	if bucket.violations > 0 {
		bucket.violations--
	}

	return true
}

// GetStatus returns the current status for a client
func (rl *RateLimiter) GetStatus(clientID string) *RateLimitStatus {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.clients[clientID]
	if !exists {
		return &RateLimitStatus{
			Requests:   0,
			MaxRequests: rl.config.MaxRequests,
			WindowEnd:  time.Now().Add(rl.config.WindowDuration),
			Banned:     false,
		}
	}

	bucket.mutex.RLock()
	defer bucket.mutex.RUnlock()

	now := time.Now()
	
	// Count requests in current window
	cutoff := now.Add(-rl.config.WindowDuration)
	currentRequests := 0
	var oldestRequest time.Time
	
	for _, reqTime := range bucket.requests {
		if reqTime.After(cutoff) {
			currentRequests++
			if oldestRequest.IsZero() || reqTime.Before(oldestRequest) {
				oldestRequest = reqTime
			}
		}
	}

	windowEnd := now.Add(rl.config.WindowDuration)
	if !oldestRequest.IsZero() {
		windowEnd = oldestRequest.Add(rl.config.WindowDuration)
	}

	return &RateLimitStatus{
		Requests:    currentRequests,
		MaxRequests: rl.config.MaxRequests,
		WindowEnd:   windowEnd,
		Banned:      now.Before(bucket.bannedUntil),
		BannedUntil: bucket.bannedUntil,
		Violations:  bucket.violations,
	}
}

// Reset removes all tracking for a client
func (rl *RateLimiter) Reset(clientID string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.clients, clientID)
}

// Ban explicitly bans a client for the specified duration
func (rl *RateLimiter) Ban(clientID string, duration time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.clients[clientID]
	if !exists {
		bucket = &ClientBucket{
			requests: make([]time.Time, 0),
		}
		rl.clients[clientID] = bucket
	}

	bucket.mutex.Lock()
	bucket.bannedUntil = time.Now().Add(duration)
	bucket.violations++
	bucket.mutex.Unlock()
}

// Unban removes the ban for a client
func (rl *RateLimiter) Unban(clientID string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.clients[clientID]
	if !exists {
		return
	}

	bucket.mutex.Lock()
	bucket.bannedUntil = time.Time{}
	bucket.violations = 0
	bucket.mutex.Unlock()
}

// GetBannedClients returns a list of currently banned clients
func (rl *RateLimiter) GetBannedClients() []string {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	now := time.Now()
	var banned []string

	for clientID, bucket := range rl.clients {
		bucket.mutex.RLock()
		if now.Before(bucket.bannedUntil) {
			banned = append(banned, clientID)
		}
		bucket.mutex.RUnlock()
	}

	return banned
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() *RateLimitStats {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	now := time.Now()
	stats := &RateLimitStats{
		TotalClients: len(rl.clients),
		Config:       rl.config,
	}

	cutoff := now.Add(-rl.config.WindowDuration)

	for _, bucket := range rl.clients {
		bucket.mutex.RLock()
		
		// Count active requests
		for _, reqTime := range bucket.requests {
			if reqTime.After(cutoff) {
				stats.ActiveRequests++
			}
		}

		// Count banned clients
		if now.Before(bucket.bannedUntil) {
			stats.BannedClients++
		}

		// Count violations
		stats.TotalViolations += bucket.violations
		
		bucket.mutex.RUnlock()
	}

	return stats
}

// startCleanup runs a background goroutine to clean up old entries
func (rl *RateLimiter) startCleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes old entries to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.config.WindowDuration * 2) // Keep some history

	for clientID, bucket := range rl.clients {
		bucket.mutex.Lock()

		// Remove old requests
		validRequests := make([]time.Time, 0)
		for _, reqTime := range bucket.requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}
		bucket.requests = validRequests

		// Check if bucket can be removed
		canRemove := len(bucket.requests) == 0 && 
					now.After(bucket.bannedUntil) && 
					bucket.violations == 0

		bucket.mutex.Unlock()

		if canRemove {
			delete(rl.clients, clientID)
		}
	}
}

// RateLimitStatus represents the current rate limit status for a client
type RateLimitStatus struct {
	Requests    int       `json:"requests"`
	MaxRequests int       `json:"max_requests"`
	WindowEnd   time.Time `json:"window_end"`
	Banned      bool      `json:"banned"`
	BannedUntil time.Time `json:"banned_until,omitempty"`
	Violations  int       `json:"violations"`
}

// RateLimitStats represents overall rate limiter statistics
type RateLimitStats struct {
	TotalClients     int               `json:"total_clients"`
	ActiveRequests   int               `json:"active_requests"`
	BannedClients    int               `json:"banned_clients"`
	TotalViolations  int               `json:"total_violations"`
	Config           *RateLimitConfig  `json:"config"`
}