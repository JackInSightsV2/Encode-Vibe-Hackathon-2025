package api

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"qt1-middleware/config"
)

// HealthData represents the system health information
type HealthData struct {
	Timestamp    string                 `json:"timestamp"`
	System       SystemHealth           `json:"system"`
	Service      ServiceHealth          `json:"service"`
	WebSocket    WebSocketHealth        `json:"websocket"`
	Providers    map[string]interface{} `json:"providers"`
	RequestStats RequestStats           `json:"request_stats"`
}

type SystemHealth struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	Goroutines  int     `json:"goroutines"`
	Uptime      string  `json:"uptime"`
}

type ServiceHealth struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	Port        int    `json:"port"`
	Moderation  bool   `json:"moderation_enabled"`
	Relevance   bool   `json:"relevance_enabled"`
	KillSwitch  bool   `json:"kill_switch_active"`
}

type WebSocketHealth struct {
	ActiveConnections int    `json:"active_connections"`
	TotalConnections  int64  `json:"total_connections"`
	Status           string `json:"status"`
}

type RequestStats struct {
	TotalRequests    int64   `json:"total_requests"`
	SuccessfulReqs   int64   `json:"successful_requests"`
	FailedRequests   int64   `json:"failed_requests"`
	AverageResponse  float64 `json:"average_response_ms"`
	RequestsPerMin   float64 `json:"requests_per_minute"`
}

// HealthManager manages health data collection and broadcasting
type HealthManager struct {
	wsManager      *WebSocketManager
	startTime      time.Time
	ctx            context.Context
	cancel         context.CancelFunc
	ticker         *time.Ticker
	
	// Metrics counters
	totalConnections   int64
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	
	// Response time tracking
	responseTimes      []float64
	responseTimeIndex  int
	responseTimeSum    float64
}

// NewHealthManager creates a new health manager
func NewHealthManager(wsManager *WebSocketManager) *HealthManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	hm := &HealthManager{
		wsManager:     wsManager,
		startTime:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
		ticker:        time.NewTicker(10 * time.Second), // Update every 10 seconds
		responseTimes: make([]float64, 100), // Keep last 100 response times
	}
	
	// Start the health broadcasting goroutine
	go hm.startHealthBroadcasting()
	
	return hm
}

// GetMessageType implements MessageHandler interface
func (hm *HealthManager) GetMessageType() string {
	return MessageTypeHealthUpdate
}

// HandleMessage implements MessageHandler interface
func (hm *HealthManager) HandleMessage(conn *websocket.Conn, msg WSMessage) error {
	// Health manager doesn't handle incoming messages, only broadcasts
	log.Printf("Health manager received unexpected message: %v", msg.Type)
	return nil
}

// startHealthBroadcasting starts the periodic health broadcasting
func (hm *HealthManager) startHealthBroadcasting() {
	defer hm.ticker.Stop()
	
	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-hm.ticker.C:
			hm.broadcastHealthUpdate()
		}
	}
}

// broadcastHealthUpdate collects and broadcasts current health data
func (hm *HealthManager) broadcastHealthUpdate() {
	healthData := hm.collectSystemHealth()
	
	message := WSMessage{
		Type:      MessageTypeHealthUpdate,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"health": healthData,
		},
	}
	
	hm.wsManager.BroadcastMessage(message)
	log.Printf("Broadcasted health update to %d connections", hm.wsManager.GetConnectionCount())
}

// collectSystemHealth gathers current system health metrics
func (hm *HealthManager) collectSystemHealth() HealthData {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Calculate uptime
	uptime := time.Since(hm.startTime)
	
	// Calculate memory usage percentage (rough estimate)
	memoryUsage := float64(m.Alloc) / (1024 * 1024) // MB
	
	// Calculate average response time
	avgResponseTime := hm.getAverageResponseTime()
	
	// Calculate requests per minute
	requestsPerMin := hm.getRequestsPerMinute()
	
	return HealthData{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		System: SystemHealth{
			CPUUsage:    hm.getCPUUsage(), // Simplified CPU usage
			MemoryUsage: memoryUsage,
			Goroutines:  runtime.NumGoroutine(),
			Uptime:      hm.formatUptime(uptime),
		},
		Service: ServiceHealth{
			Status:     "healthy",
			Version:    "1.0.0",
			Port:       config.AppConfig.Server.Port,
			Moderation: config.AppConfig.Moderation.Enabled,
			Relevance:  config.AppConfig.Relevance.Enabled,
			KillSwitch: false, // TODO: Implement kill switch status
		},
		WebSocket: WebSocketHealth{
			ActiveConnections: hm.wsManager.GetConnectionCount(),
			TotalConnections:  atomic.LoadInt64(&hm.totalConnections),
			Status:           "active",
		},
		Providers: map[string]interface{}{
			"openai": map[string]interface{}{
				"status": "available",
				"model":  "gpt-3.5-turbo",
			},
		},
		RequestStats: RequestStats{
			TotalRequests:    atomic.LoadInt64(&hm.totalRequests),
			SuccessfulReqs:   atomic.LoadInt64(&hm.successfulRequests),
			FailedRequests:   atomic.LoadInt64(&hm.failedRequests),
			AverageResponse:  avgResponseTime,
			RequestsPerMin:   requestsPerMin,
		},
	}
}

// RecordConnection increments the total connection counter
func (hm *HealthManager) RecordConnection() {
	atomic.AddInt64(&hm.totalConnections, 1)
}

// RecordRequest records a new request with its response time
func (hm *HealthManager) RecordRequest(responseTime float64, success bool) {
	atomic.AddInt64(&hm.totalRequests, 1)
	
	if success {
		atomic.AddInt64(&hm.successfulRequests, 1)
	} else {
		atomic.AddInt64(&hm.failedRequests, 1)
	}
	
	// Record response time (thread-safe circular buffer)
	hm.responseTimes[hm.responseTimeIndex] = responseTime
	hm.responseTimeIndex = (hm.responseTimeIndex + 1) % len(hm.responseTimes)
	hm.responseTimeSum += responseTime
}

// getCPUUsage returns a simplified CPU usage estimate
func (hm *HealthManager) getCPUUsage() float64 {
	// Simplified CPU usage based on goroutine count
	// In production, you'd use a proper CPU monitoring library
	goroutines := float64(runtime.NumGoroutine())
	baselineGoroutines := 10.0 // Expected baseline
	
	if goroutines <= baselineGoroutines {
		return 0.1 // 10% baseline
	}
	
	// Scale CPU usage based on goroutine count (very rough estimate)
	usage := (goroutines - baselineGoroutines) / 100.0
	if usage > 0.9 {
		usage = 0.9 // Cap at 90%
	}
	
	return usage
}

// getAverageResponseTime calculates the average response time
func (hm *HealthManager) getAverageResponseTime() float64 {
	if atomic.LoadInt64(&hm.totalRequests) == 0 {
		return 0.0
	}
	
	sum := 0.0
	count := 0
	
	for _, responseTime := range hm.responseTimes {
		if responseTime > 0 {
			sum += responseTime
			count++
		}
	}
	
	if count == 0 {
		return 0.0
	}
	
	return sum / float64(count)
}

// getRequestsPerMinute calculates requests per minute
func (hm *HealthManager) getRequestsPerMinute() float64 {
	totalRequests := atomic.LoadInt64(&hm.totalRequests)
	uptime := time.Since(hm.startTime).Minutes()
	
	if uptime == 0 {
		return 0.0
	}
	
	return float64(totalRequests) / uptime
}

// formatUptime formats the uptime duration into a human-readable string
func (hm *HealthManager) formatUptime(uptime time.Duration) string {
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	
	return fmt.Sprintf("%ds", seconds)
}

// Close gracefully shuts down the health manager
func (hm *HealthManager) Close() {
	hm.cancel()
}

// Global health manager instance
var HealthMgr *HealthManager