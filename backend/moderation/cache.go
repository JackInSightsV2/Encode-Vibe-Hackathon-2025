package moderation

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// ModerationCache provides in-memory caching for moderation results
type ModerationCache struct {
	entries   map[string]*CacheEntry
	mutex     sync.RWMutex
	config    CacheConfig
	stats     CacheStats
	cleanupCh chan struct{}
}

// CacheStats tracks cache performance
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Entries     int   `json:"entries"`
	Evictions   int64 `json:"evictions"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// NewModerationCache creates a new cache instance
func NewModerationCache(config CacheConfig) *ModerationCache {
	cache := &ModerationCache{
		entries:   make(map[string]*CacheEntry),
		config:    config,
		cleanupCh: make(chan struct{}),
	}
	
	if config.Enabled {
		// Start cleanup goroutine
		go cache.startCleanupRoutine()
	}
	
	return cache
}

// generateCacheKey creates a deterministic cache key from content and context
func (mc *ModerationCache) generateCacheKey(content string, context ModerationContext) string {
	// Hash the content for consistent keys
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	
	// Include relevant context that affects moderation results
	contextKey := fmt.Sprintf("%s:%s", context.UserID, context.SessionID)
	
	// Combine for final key
	return fmt.Sprintf("mod:%s:%s", contentHash, contextKey)
}

// Get retrieves a cached result if available and not expired
func (mc *ModerationCache) Get(content string, context ModerationContext) (*AggregatedResult, bool) {
	if !mc.config.Enabled {
		return nil, false
	}
	
	key := mc.generateCacheKey(content, context)
	
	mc.mutex.RLock()
	entry, exists := mc.entries[key]
	mc.mutex.RUnlock()
	
	if !exists {
		mc.incrementMisses()
		return nil, false
	}
	
	// Check if entry has expired
	if time.Now().After(entry.TTL) {
		mc.mutex.Lock()
		delete(mc.entries, key)
		mc.mutex.Unlock()
		mc.incrementMisses()
		return nil, false
	}
	
	mc.incrementHits()
	
	// Mark as cache hit in the result
	result := entry.Result
	result.CacheHit = true
	
	return &result, true
}

// Set stores a result in the cache
func (mc *ModerationCache) Set(content string, context ModerationContext, result AggregatedResult) {
	if !mc.config.Enabled {
		return
	}
	
	key := mc.generateCacheKey(content, context)
	ttl := time.Now().Add(time.Duration(mc.config.TTLMinutes) * time.Minute)
	
	entry := &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
	
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// Check if we need to evict entries to stay under max limit
	if len(mc.entries) >= mc.config.MaxEntries {
		mc.evictOldestEntries(1)
	}
	
	mc.entries[key] = entry
}

// evictOldestEntries removes the oldest entries from cache
func (mc *ModerationCache) evictOldestEntries(count int) {
	if len(mc.entries) == 0 {
		return
	}
	
	// Find oldest entries
	type keyTime struct {
		key       string
		timestamp time.Time
	}
	
	var entries []keyTime
	for key, entry := range mc.entries {
		entries = append(entries, keyTime{key: key, timestamp: entry.Timestamp})
	}
	
	// Sort by timestamp (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].timestamp.After(entries[j].timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Remove oldest entries
	evicted := 0
	for i := 0; i < len(entries) && evicted < count; i++ {
		delete(mc.entries, entries[i].key)
		evicted++
	}
	
	mc.stats.Evictions += int64(evicted)
}

// startCleanupRoutine periodically removes expired entries
func (mc *ModerationCache) startCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mc.cleanup()
		case <-mc.cleanupCh:
			return
		}
	}
}

// cleanup removes expired entries
func (mc *ModerationCache) cleanup() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	now := time.Now()
	var expiredKeys []string
	
	for key, entry := range mc.entries {
		if now.After(entry.TTL) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	for _, key := range expiredKeys {
		delete(mc.entries, key)
	}
	
	mc.stats.LastCleanup = now
	mc.stats.Evictions += int64(len(expiredKeys))
}

// Clear removes all entries from the cache
func (mc *ModerationCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	mc.entries = make(map[string]*CacheEntry)
	mc.stats = CacheStats{}
}

// GetStats returns current cache statistics
func (mc *ModerationCache) GetStats() CacheStats {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	stats := mc.stats
	stats.Entries = len(mc.entries)
	return stats
}

// Close shuts down the cache and cleanup routines
func (mc *ModerationCache) Close() {
	if mc.config.Enabled {
		close(mc.cleanupCh)
	}
}

// incrementHits safely increments the hit counter
func (mc *ModerationCache) incrementHits() {
	mc.mutex.Lock()
	mc.stats.Hits++
	mc.mutex.Unlock()
}

// incrementMisses safely increments the miss counter
func (mc *ModerationCache) incrementMisses() {
	mc.mutex.Lock()
	mc.stats.Misses++
	mc.mutex.Unlock()
}

// GetHitRate returns the cache hit rate as a percentage
func (mc *ModerationCache) GetHitRate() float64 {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	total := mc.stats.Hits + mc.stats.Misses
	if total == 0 {
		return 0.0
	}
	
	return float64(mc.stats.Hits) / float64(total) * 100.0
}

// IsEnabled returns whether caching is enabled
func (mc *ModerationCache) IsEnabled() bool {
	return mc.config.Enabled
}