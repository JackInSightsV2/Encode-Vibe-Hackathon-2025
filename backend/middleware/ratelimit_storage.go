package middleware

import (
	"fmt"
	"sync"
	"time"
	
	"qt1-middleware/config"
)

// RateLimitStorage defines the interface for rate limit data storage
type RateLimitStorage interface {
	// Get retrieves rate limit data for a key
	Get(key string) (*RateLimitData, error)
	
	// Set stores rate limit data for a key with optional TTL
	Set(key string, data *RateLimitData, ttl time.Duration) error
	
	// Delete removes rate limit data for a key
	Delete(key string) error
	
	// Increment atomically increments a counter and returns the new value
	Increment(key string, window time.Duration) (int, error)
	
	// GetRequestTimes gets request timestamps for sliding window
	GetRequestTimes(key string, window time.Duration) ([]time.Time, error)
	
	// AddRequestTime adds a request timestamp for sliding window
	AddRequestTime(key string, timestamp time.Time, window time.Duration) error
	
	// Cleanup removes expired data
	Cleanup() error
	
	// Close closes the storage connection
	Close() error
}

// RateLimitData represents stored rate limiting information
type RateLimitData struct {
	Count        int         `json:"count"`
	LastReset    time.Time   `json:"last_reset"`
	RequestTimes []time.Time `json:"request_times,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// MemoryStorage implements in-memory rate limit storage
type MemoryStorage struct {
	data  map[string]*RateLimitData
	mutex sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]*RateLimitData),
	}
}

// Get retrieves rate limit data for a key
func (ms *MemoryStorage) Get(key string) (*RateLimitData, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	data, exists := ms.data[key]
	if !exists {
		return nil, nil
	}
	
	// Return a copy to prevent race conditions
	dataCopy := *data
	if data.RequestTimes != nil {
		dataCopy.RequestTimes = make([]time.Time, len(data.RequestTimes))
		copy(dataCopy.RequestTimes, data.RequestTimes)
	}
	
	return &dataCopy, nil
}

// Set stores rate limit data for a key with optional TTL
func (ms *MemoryStorage) Set(key string, data *RateLimitData, ttl time.Duration) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	// Create a copy to store
	dataCopy := *data
	if data.RequestTimes != nil {
		dataCopy.RequestTimes = make([]time.Time, len(data.RequestTimes))
		copy(dataCopy.RequestTimes, data.RequestTimes)
	}
	dataCopy.UpdatedAt = time.Now()
	
	ms.data[key] = &dataCopy
	
	// TTL is handled by cleanup routine for memory storage
	return nil
}

// Delete removes rate limit data for a key
func (ms *MemoryStorage) Delete(key string) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	delete(ms.data, key)
	return nil
}

// Increment atomically increments a counter and returns the new value
func (ms *MemoryStorage) Increment(key string, window time.Duration) (int, error) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	now := time.Now()
	data, exists := ms.data[key]
	
	if !exists || now.Sub(data.LastReset) > window {
		// Create new or reset expired counter
		ms.data[key] = &RateLimitData{
			Count:     1,
			LastReset: now,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return 1, nil
	}
	
	// Increment existing counter
	data.Count++
	data.UpdatedAt = now
	return data.Count, nil
}

// GetRequestTimes gets request timestamps for sliding window
func (ms *MemoryStorage) GetRequestTimes(key string, window time.Duration) ([]time.Time, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	data, exists := ms.data[key]
	if !exists || data.RequestTimes == nil {
		return []time.Time{}, nil
	}
	
	// Filter requests within the window
	now := time.Now()
	cutoff := now.Add(-window)
	
	var validTimes []time.Time
	for _, reqTime := range data.RequestTimes {
		if reqTime.After(cutoff) {
			validTimes = append(validTimes, reqTime)
		}
	}
	
	return validTimes, nil
}

// AddRequestTime adds a request timestamp for sliding window
func (ms *MemoryStorage) AddRequestTime(key string, timestamp time.Time, window time.Duration) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	now := time.Now()
	data, exists := ms.data[key]
	
	if !exists {
		data = &RateLimitData{
			RequestTimes: []time.Time{timestamp},
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		ms.data[key] = data
		return nil
	}
	
	// Add new timestamp
	data.RequestTimes = append(data.RequestTimes, timestamp)
	data.UpdatedAt = now
	
	// Clean up old timestamps to prevent memory leaks
	cutoff := now.Add(-window)
	validTimes := make([]time.Time, 0, len(data.RequestTimes))
	
	for _, reqTime := range data.RequestTimes {
		if reqTime.After(cutoff) {
			validTimes = append(validTimes, reqTime)
		}
	}
	
	data.RequestTimes = validTimes
	return nil
}

// Cleanup removes expired data
func (ms *MemoryStorage) Cleanup() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-time.Hour) // Remove data older than 1 hour
	
	for key, data := range ms.data {
		if data.UpdatedAt.Before(cutoff) {
			delete(ms.data, key)
		}
	}
	
	return nil
}

// Close closes the storage connection (no-op for memory storage)
func (ms *MemoryStorage) Close() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	// Clear all data
	ms.data = make(map[string]*RateLimitData)
	return nil
}

// RedisStorage implements Redis-based rate limit storage
type RedisStorage struct {
	// Note: This is a placeholder implementation
	// In a real implementation, you would use a Redis client like go-redis
	prefix string
	// client redis.Client // Would be the actual Redis client
}

// NewRedisStorage creates a new Redis storage instance
func NewRedisStorage(redisURL, prefix string) (*RedisStorage, error) {
	// Placeholder implementation
	// In a real implementation:
	// 1. Parse redisURL
	// 2. Create Redis client
	// 3. Test connection
	
	if redisURL == "" {
		return nil, fmt.Errorf("Redis URL is required")
	}
	
	return &RedisStorage{
		prefix: prefix,
		// client: redis.NewClient(&redis.Options{Addr: redisURL}),
	}, fmt.Errorf("Redis storage not implemented in this demo - use memory storage instead")
}

// Get retrieves rate limit data for a key
func (rs *RedisStorage) Get(key string) (*RateLimitData, error) {
	// Placeholder implementation
	// In a real implementation:
	// 1. Get data from Redis using rs.client.Get(rs.prefix + key)
	// 2. Unmarshal JSON data
	// 3. Return RateLimitData
	return nil, fmt.Errorf("Redis storage not implemented")
}

// Set stores rate limit data for a key with TTL
func (rs *RedisStorage) Set(key string, data *RateLimitData, ttl time.Duration) error {
	// Placeholder implementation
	// In a real implementation:
	// 1. Marshal data to JSON
	// 2. Store in Redis with TTL using rs.client.SetEX(rs.prefix + key, ttl, jsonData)
	return fmt.Errorf("Redis storage not implemented")
}

// Delete removes rate limit data for a key
func (rs *RedisStorage) Delete(key string) error {
	// Placeholder implementation
	// In a real implementation:
	// rs.client.Del(rs.prefix + key)
	return fmt.Errorf("Redis storage not implemented")
}

// Increment atomically increments a counter
func (rs *RedisStorage) Increment(key string, window time.Duration) (int, error) {
	// Placeholder implementation
	// In a real implementation:
	// 1. Use Redis INCR with expiration
	// 2. Handle window-based reset logic
	return 0, fmt.Errorf("Redis storage not implemented")
}

// GetRequestTimes gets request timestamps for sliding window
func (rs *RedisStorage) GetRequestTimes(key string, window time.Duration) ([]time.Time, error) {
	// Placeholder implementation
	// In a real implementation:
	// 1. Use Redis ZRANGEBYSCORE to get timestamps within window
	// 2. Convert to []time.Time
	return nil, fmt.Errorf("Redis storage not implemented")
}

// AddRequestTime adds a request timestamp for sliding window
func (rs *RedisStorage) AddRequestTime(key string, timestamp time.Time, window time.Duration) error {
	// Placeholder implementation
	// In a real implementation:
	// 1. Use Redis ZADD to add timestamp with score
	// 2. Use ZREMRANGEBYSCORE to remove old timestamps
	// 3. Set TTL on the key
	return fmt.Errorf("Redis storage not implemented")
}

// Cleanup removes expired data
func (rs *RedisStorage) Cleanup() error {
	// Placeholder implementation
	// Redis handles TTL automatically, but we might want to clean up sliding window data
	return fmt.Errorf("Redis storage not implemented")
}

// Close closes the Redis connection
func (rs *RedisStorage) Close() error {
	// Placeholder implementation
	// In a real implementation:
	// return rs.client.Close()
	return fmt.Errorf("Redis storage not implemented")
}

// CreateRateLimitStorage creates a storage instance based on configuration
func CreateRateLimitStorage(storageConfig config.RateLimitStorageConfig) (RateLimitStorage, error) {
	switch storageConfig.Type {
	case "redis":
		return NewRedisStorage(storageConfig.RedisURL, storageConfig.Prefix)
	case "memory":
		fallthrough
	default:
		return NewMemoryStorage(), nil
	}
}