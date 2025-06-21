package providers

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// HealthMonitor manages health checking for all providers
type HealthMonitor struct {
	registry    *ProviderRegistry
	checkers    map[string]*ProviderChecker
	config      *HealthConfig
	scheduler   *HealthScheduler
	metrics     *HealthMetrics
	mutex       sync.RWMutex
	running     bool
}

// HealthConfig contains configuration for health monitoring
type HealthConfig struct {
	CheckInterval    time.Duration    `yaml:"check_interval" json:"check_interval"`
	Timeout          time.Duration    `yaml:"timeout" json:"timeout"`
	FailureThreshold int              `yaml:"failure_threshold" json:"failure_threshold"`
	SuccessThreshold int              `yaml:"success_threshold" json:"success_threshold"`
	Strategies       []HealthStrategy `yaml:"strategies" json:"strategies"`
}

// HealthStrategy defines different health check strategies
type HealthStrategy struct {
	Type   string                 `yaml:"type" json:"type"`
	Weight float64                `yaml:"weight" json:"weight"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// ProviderChecker tracks health status for a single provider
type ProviderChecker struct {
	provider             Provider
	consecutiveFailures  int
	consecutiveSuccesses int
	lastCheck           time.Time
	healthScore         float64
	checks              []HealthCheckResult
	maxHistorySize      int
	mutex               sync.RWMutex
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Timestamp time.Time              `json:"timestamp"`
	Success   bool                   `json:"success"`
	Latency   time.Duration          `json:"latency"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ProviderHealth contains comprehensive health information for a provider
type ProviderHealth struct {
	Name                 string               `json:"name"`
	HealthScore          float64              `json:"health_score"`
	ConsecutiveFailures  int                  `json:"consecutive_failures"`
	ConsecutiveSuccesses int                  `json:"consecutive_successes"`
	LastCheck           time.Time            `json:"last_check"`
	RecentChecks        []HealthCheckResult  `json:"recent_checks"`
	Status              string               `json:"status"`
	Trends              HealthTrends         `json:"trends"`
}

// HealthTrends contains trend analysis for provider health
type HealthTrends struct {
	LatencyTrend   string  `json:"latency_trend"`   // "improving", "stable", "degrading"
	SuccessTrend   string  `json:"success_trend"`   // "improving", "stable", "degrading"
	OverallTrend   string  `json:"overall_trend"`   // "improving", "stable", "degrading"
	TrendScore     float64 `json:"trend_score"`     // -1.0 to 1.0
}

// HealthMetrics tracks aggregated health metrics
type HealthMetrics struct {
	totalChecks    int64
	totalFailures  int64
	totalLatency   time.Duration
	providerStats  map[string]*ProviderStats
	mutex          sync.RWMutex
}

// ProviderStats tracks statistics for a specific provider
type ProviderStats struct {
	CheckCount    int64         `json:"check_count"`
	FailureCount  int64         `json:"failure_count"`
	TotalLatency  time.Duration `json:"total_latency"`
	LastUpdated   time.Time     `json:"last_updated"`
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(registry *ProviderRegistry, config *HealthConfig) *HealthMonitor {
	if config == nil {
		config = &HealthConfig{
			CheckInterval:    30 * time.Second,
			Timeout:          10 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 2,
			Strategies: []HealthStrategy{
				{
					Type:   "response_time",
					Weight: 0.3,
					Config: map[string]interface{}{
						"threshold_ms": 1000.0,
					},
				},
				{
					Type:   "error_rate",
					Weight: 0.4,
					Config: map[string]interface{}{
						"threshold": 0.1,
					},
				},
				{
					Type:   "rate_limit",
					Weight: 0.2,
					Config: map[string]interface{}{
						"threshold": 10.0,
					},
				},
				{
					Type:   "model_availability",
					Weight: 0.1,
					Config: map[string]interface{}{
						"required_models": []string{},
					},
				},
			},
		}
	}

	hm := &HealthMonitor{
		registry: registry,
		checkers: make(map[string]*ProviderChecker),
		config:   config,
		metrics:  NewHealthMetrics(),
		running:  false,
	}

	hm.scheduler = NewHealthScheduler(hm, config.CheckInterval)
	return hm
}

// NewHealthMetrics creates a new health metrics tracker
func NewHealthMetrics() *HealthMetrics {
	return &HealthMetrics{
		providerStats: make(map[string]*ProviderStats),
	}
}

// Start begins health monitoring
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if hm.running {
		return nil // Already running
	}

	// Initialize checkers for all registered providers
	providers := hm.registry.List()
	for _, provider := range providers {
		checker := &ProviderChecker{
			provider:       provider,
			healthScore:    1.0, // Start with perfect health
			checks:         make([]HealthCheckResult, 0),
			maxHistorySize: 100,
		}
		hm.checkers[provider.Name()] = checker
	}

	// Start the scheduler
	if err := hm.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health scheduler: %w", err)
	}

	hm.running = true
	return nil
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop(ctx context.Context) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if !hm.running {
		return nil // Already stopped
	}

	if err := hm.scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop health scheduler: %w", err)
	}

	hm.running = false
	return nil
}

// CheckProvider performs a comprehensive health check on a provider
func (hm *HealthMonitor) CheckProvider(ctx context.Context, providerName string) *HealthCheckResult {
	provider, err := hm.registry.Get(providerName)
	if err != nil {
		return &HealthCheckResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		}
	}

	start := time.Now()

	// Perform health check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, hm.config.Timeout)
	defer cancel()

	err = provider.HealthCheck(checkCtx)
	latency := time.Since(start)

	result := &HealthCheckResult{
		Timestamp: time.Now(),
		Success:   err == nil,
		Latency:   latency,
		Details:   make(map[string]interface{}),
	}

	if err != nil {
		result.Error = err.Error()
	}

	// Add additional health check strategies
	hm.runHealthStrategies(checkCtx, provider, result)

	// Update checker state
	hm.updateCheckerState(providerName, result)

	// Record metrics
	hm.metrics.RecordCheck(providerName, result)

	return result
}

// runHealthStrategies executes all configured health check strategies
func (hm *HealthMonitor) runHealthStrategies(ctx context.Context, provider Provider, result *HealthCheckResult) {
	for _, strategy := range hm.config.Strategies {
		switch strategy.Type {
		case "response_time":
			hm.checkResponseTime(ctx, provider, result, strategy)
		case "error_rate":
			hm.checkErrorRate(ctx, provider, result, strategy)
		case "rate_limit":
			hm.checkRateLimit(ctx, provider, result, strategy)
		case "model_availability":
			hm.checkModelAvailability(ctx, provider, result, strategy)
		}
	}
}

// checkResponseTime validates response time against threshold
func (hm *HealthMonitor) checkResponseTime(ctx context.Context, provider Provider, result *HealthCheckResult, strategy HealthStrategy) {
	threshold := time.Duration(strategy.Config["threshold_ms"].(float64)) * time.Millisecond

	passed := result.Latency <= threshold
	if !passed {
		result.Success = false
		if result.Error == "" {
			result.Error = fmt.Sprintf("Response time %v exceeds threshold %v", result.Latency, threshold)
		}
	}

	result.Details["response_time_check"] = map[string]interface{}{
		"latency":   result.Latency.Milliseconds(),
		"threshold": threshold.Milliseconds(),
		"passed":    passed,
		"weight":    strategy.Weight,
	}
}

// checkErrorRate validates error rate against threshold
func (hm *HealthMonitor) checkErrorRate(ctx context.Context, provider Provider, result *HealthCheckResult, strategy HealthStrategy) {
	threshold := strategy.Config["threshold"].(float64)
	
	// Get recent error rate from provider metrics
	metrics := provider.GetMetrics()
	errorRate := metrics.ErrorRate

	passed := errorRate <= threshold
	if !passed {
		result.Success = false
		if result.Error == "" {
			result.Error = fmt.Sprintf("Error rate %.3f exceeds threshold %.3f", errorRate, threshold)
		}
	}

	result.Details["error_rate_check"] = map[string]interface{}{
		"error_rate": errorRate,
		"threshold":  threshold,
		"passed":     passed,
		"weight":     strategy.Weight,
	}
}

// checkRateLimit validates rate limit status
func (hm *HealthMonitor) checkRateLimit(ctx context.Context, provider Provider, result *HealthCheckResult, strategy HealthStrategy) {
	threshold := strategy.Config["threshold"].(float64)
	
	// Get recent rate limit hits from provider metrics
	metrics := provider.GetMetrics()
	rateLimitHits := float64(metrics.RateLimitHits)

	passed := rateLimitHits <= threshold
	if !passed {
		result.Success = false
		if result.Error == "" {
			result.Error = fmt.Sprintf("Rate limit hits %.0f exceeds threshold %.0f", rateLimitHits, threshold)
		}
	}

	result.Details["rate_limit_check"] = map[string]interface{}{
		"rate_limit_hits": rateLimitHits,
		"threshold":       threshold,
		"passed":          passed,
		"weight":          strategy.Weight,
	}
}

// checkModelAvailability validates that required models are available
func (hm *HealthMonitor) checkModelAvailability(ctx context.Context, provider Provider, result *HealthCheckResult, strategy HealthStrategy) {
	requiredModels, ok := strategy.Config["required_models"].([]string)
	if !ok || len(requiredModels) == 0 {
		// No required models specified, always pass
		result.Details["model_availability_check"] = map[string]interface{}{
			"passed": true,
			"weight": strategy.Weight,
		}
		return
	}

	availableModels := provider.GetModels()
	availableModelMap := make(map[string]bool)
	for _, model := range availableModels {
		availableModelMap[model.ID] = true
	}

	missingModels := make([]string, 0)
	for _, required := range requiredModels {
		if !availableModelMap[required] {
			missingModels = append(missingModels, required)
		}
	}

	passed := len(missingModels) == 0
	if !passed {
		result.Success = false
		if result.Error == "" {
			result.Error = fmt.Sprintf("Required models not available: %v", missingModels)
		}
	}

	result.Details["model_availability_check"] = map[string]interface{}{
		"required_models":  requiredModels,
		"available_models": len(availableModels),
		"missing_models":   missingModels,
		"passed":           passed,
		"weight":           strategy.Weight,
	}
}

// updateCheckerState updates the internal state of a provider checker
func (hm *HealthMonitor) updateCheckerState(providerName string, result *HealthCheckResult) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	checker, exists := hm.checkers[providerName]
	if !exists {
		return
	}

	checker.mutex.Lock()
	defer checker.mutex.Unlock()

	checker.lastCheck = result.Timestamp

	// Update success/failure counters
	if result.Success {
		checker.consecutiveFailures = 0
		checker.consecutiveSuccesses++
	} else {
		checker.consecutiveSuccesses = 0
		checker.consecutiveFailures++
	}

	// Add to history
	checker.checks = append(checker.checks, *result)
	if len(checker.checks) > checker.maxHistorySize {
		checker.checks = checker.checks[1:]
	}

	// Calculate health score
	checker.healthScore = hm.calculateHealthScore(checker)
}

// calculateHealthScore computes a comprehensive health score (0.0 to 1.0)
func (hm *HealthMonitor) calculateHealthScore(checker *ProviderChecker) float64 {
	if len(checker.checks) == 0 {
		return 1.0
	}

	// Base score on recent success rate
	recentChecks := 10
	if len(checker.checks) < recentChecks {
		recentChecks = len(checker.checks)
	}

	recentResults := checker.checks[len(checker.checks)-recentChecks:]
	successCount := 0
	totalLatency := time.Duration(0)
	strategyScores := make(map[string]float64)

	for _, check := range recentResults {
		if check.Success {
			successCount++
		}
		totalLatency += check.Latency

		// Calculate strategy-specific scores
		for strategy, details := range check.Details {
			if detailsMap, ok := details.(map[string]interface{}); ok {
				if passed, ok := detailsMap["passed"].(bool); ok {
					if weight, ok := detailsMap["weight"].(float64); ok {
						if passed {
							strategyScores[strategy] += weight / float64(recentChecks)
						}
					}
				}
			}
		}
	}

	successRate := float64(successCount) / float64(len(recentResults))
	avgLatency := totalLatency / time.Duration(len(recentResults))

	// Base score on success rate (0.5 weight)
	baseScore := successRate * 0.5

	// Latency score (0.2 weight)
	latencyScore := math.Max(0, 1.0-float64(avgLatency.Milliseconds())/2000.0)
	baseScore += latencyScore * 0.2

	// Strategy scores (0.3 weight total)
	strategyWeight := 0.3
	for _, score := range strategyScores {
		baseScore += score * strategyWeight
	}

	// Apply penalties for consecutive failures
	if checker.consecutiveFailures > 0 {
		penalty := math.Min(0.5, float64(checker.consecutiveFailures)*0.1)
		baseScore -= penalty
	}

	// Apply bonuses for consecutive successes
	if checker.consecutiveSuccesses > 5 {
		bonus := math.Min(0.1, float64(checker.consecutiveSuccesses-5)*0.02)
		baseScore += bonus
	}

	return math.Max(0, math.Min(1, baseScore))
}

// GetProviderHealth returns comprehensive health information for a provider
func (hm *HealthMonitor) GetProviderHealth(providerName string) (*ProviderHealth, error) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	checker, exists := hm.checkers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	checker.mutex.RLock()
	defer checker.mutex.RUnlock()

	// Get recent checks (last 10)
	recentChecks := checker.checks
	if len(recentChecks) > 10 {
		recentChecks = recentChecks[len(recentChecks)-10:]
	}

	// Calculate trends
	trends := hm.calculateHealthTrends(checker)

	// Determine status
	status := "healthy"
	if checker.healthScore < 0.3 {
		status = "unhealthy"
	} else if checker.healthScore < 0.7 {
		status = "degraded"
	}

	return &ProviderHealth{
		Name:                 providerName,
		HealthScore:          checker.healthScore,
		ConsecutiveFailures:  checker.consecutiveFailures,
		ConsecutiveSuccesses: checker.consecutiveSuccesses,
		LastCheck:           checker.lastCheck,
		RecentChecks:        recentChecks,
		Status:              status,
		Trends:              trends,
	}, nil
}

// calculateHealthTrends analyzes trends in provider health
func (hm *HealthMonitor) calculateHealthTrends(checker *ProviderChecker) HealthTrends {
	if len(checker.checks) < 5 {
		return HealthTrends{
			LatencyTrend: "stable",
			SuccessTrend: "stable",
			OverallTrend: "stable",
			TrendScore:   0.0,
		}
	}

	// Analyze recent vs older checks
	mid := len(checker.checks) / 2
	olderChecks := checker.checks[:mid]
	recentChecks := checker.checks[mid:]

	// Calculate latency trend
	olderAvgLatency := hm.calculateAverageLatency(olderChecks)
	recentAvgLatency := hm.calculateAverageLatency(recentChecks)
	latencyChange := float64(recentAvgLatency-olderAvgLatency) / float64(olderAvgLatency)

	latencyTrend := "stable"
	if latencyChange > 0.1 {
		latencyTrend = "degrading"
	} else if latencyChange < -0.1 {
		latencyTrend = "improving"
	}

	// Calculate success rate trend
	olderSuccessRate := hm.calculateSuccessRate(olderChecks)
	recentSuccessRate := hm.calculateSuccessRate(recentChecks)
	successChange := recentSuccessRate - olderSuccessRate

	successTrend := "stable"
	if successChange > 0.1 {
		successTrend = "improving"
	} else if successChange < -0.1 {
		successTrend = "degrading"
	}

	// Calculate overall trend
	overallTrend := "stable"
	trendScore := 0.0

	if latencyTrend == "improving" && successTrend == "improving" {
		overallTrend = "improving"
		trendScore = 0.5
	} else if latencyTrend == "degrading" && successTrend == "degrading" {
		overallTrend = "degrading"
		trendScore = -0.5
	} else if latencyTrend == "improving" || successTrend == "improving" {
		overallTrend = "improving"
		trendScore = 0.25
	} else if latencyTrend == "degrading" || successTrend == "degrading" {
		overallTrend = "degrading"
		trendScore = -0.25
	}

	return HealthTrends{
		LatencyTrend: latencyTrend,
		SuccessTrend: successTrend,
		OverallTrend: overallTrend,
		TrendScore:   trendScore,
	}
}

// calculateAverageLatency calculates average latency for a set of checks
func (hm *HealthMonitor) calculateAverageLatency(checks []HealthCheckResult) time.Duration {
	if len(checks) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, check := range checks {
		total += check.Latency
	}

	return total / time.Duration(len(checks))
}

// calculateSuccessRate calculates success rate for a set of checks
func (hm *HealthMonitor) calculateSuccessRate(checks []HealthCheckResult) float64 {
	if len(checks) == 0 {
		return 1.0
	}

	successCount := 0
	for _, check := range checks {
		if check.Success {
			successCount++
		}
	}

	return float64(successCount) / float64(len(checks))
}

// GetAllProviderHealth returns health information for all providers
func (hm *HealthMonitor) GetAllProviderHealth() (map[string]*ProviderHealth, error) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[string]*ProviderHealth)

	for providerName := range hm.checkers {
		health, err := hm.GetProviderHealth(providerName)
		if err != nil {
			continue // Skip providers with errors
		}
		result[providerName] = health
	}

	return result, nil
}

// IsHealthy returns whether a provider is considered healthy
func (hm *HealthMonitor) IsHealthy(providerName string) bool {
	health, err := hm.GetProviderHealth(providerName)
	if err != nil {
		return false
	}

	return health.HealthScore >= 0.7 && health.ConsecutiveFailures < hm.config.FailureThreshold
}

// RecordCheck records a health check result in metrics
func (hm *HealthMetrics) RecordCheck(providerName string, result *HealthCheckResult) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.totalChecks++
	hm.totalLatency += result.Latency

	if !result.Success {
		hm.totalFailures++
	}

	// Update provider-specific stats
	stats, exists := hm.providerStats[providerName]
	if !exists {
		stats = &ProviderStats{}
		hm.providerStats[providerName] = stats
	}

	stats.CheckCount++
	stats.TotalLatency += result.Latency
	stats.LastUpdated = time.Now()

	if !result.Success {
		stats.FailureCount++
	}
}

// GetOverallStats returns overall health monitoring statistics
func (hm *HealthMetrics) GetOverallStats() map[string]interface{} {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	var avgLatency time.Duration
	if hm.totalChecks > 0 {
		avgLatency = hm.totalLatency / time.Duration(hm.totalChecks)
	}

	successRate := float64(hm.totalChecks-hm.totalFailures) / float64(hm.totalChecks)
	if hm.totalChecks == 0 {
		successRate = 1.0
	}

	return map[string]interface{}{
		"total_checks":         hm.totalChecks,
		"total_failures":       hm.totalFailures,
		"average_latency_ms":   avgLatency.Milliseconds(),
		"overall_success_rate": successRate,
		"provider_count":       len(hm.providerStats),
	}
}

// GetProviderStats returns statistics for a specific provider
func (hm *HealthMetrics) GetProviderStats(providerName string) (*ProviderStats, error) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	stats, exists := hm.providerStats[providerName]
	if !exists {
		return nil, fmt.Errorf("no stats found for provider %s", providerName)
	}

	// Return a copy to avoid data races
	return &ProviderStats{
		CheckCount:   stats.CheckCount,
		FailureCount: stats.FailureCount,
		TotalLatency: stats.TotalLatency,
		LastUpdated:  stats.LastUpdated,
	}, nil
}