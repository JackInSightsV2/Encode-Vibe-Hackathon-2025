package providers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// RoutingEngine manages intelligent provider routing
type RoutingEngine struct {
	registry     *ProviderRegistry
	healthMonitor *HealthMonitor
	config       *RoutingConfig
	algorithms   map[string]RoutingAlgorithm
	metrics      *RoutingMetrics
	mutex        sync.RWMutex
}

// RoutingConfig contains routing engine configuration
type RoutingConfig struct {
	Strategy    string                 `yaml:"strategy" json:"strategy"`
	Factors     map[string]float64     `yaml:"factors" json:"factors"`
	Preferences map[string]interface{} `yaml:"preferences" json:"preferences"`
	Fallback    []string              `yaml:"fallback" json:"fallback"`
	Rules       []RoutingRule         `yaml:"rules" json:"rules"`
}

// RoutingRule defines conditional routing logic
type RoutingRule struct {
	Condition string   `yaml:"condition" json:"condition"`
	Providers []string `yaml:"providers" json:"providers"`
	Weight    float64  `yaml:"weight" json:"weight"`
}

// RoutingAlgorithm interface for different routing strategies
type RoutingAlgorithm interface {
	Name() string
	SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error)
	Configure(config map[string]interface{}) error
}

// RoutingRequest contains request information for routing decisions
type RoutingRequest struct {
	Model        string                 `json:"model"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id"`
	MessageType  string                 `json:"message_type"`
	Priority     RoutingPriority        `json:"priority"`
	Constraints  RoutingConstraints     `json:"constraints"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RoutingPriority defines request priority levels
type RoutingPriority string

const (
	PriorityLow    RoutingPriority = "low"
	PriorityNormal RoutingPriority = "normal"
	PriorityHigh   RoutingPriority = "high"
)

// RoutingConstraints defines routing constraints
type RoutingConstraints struct {
	MaxLatency       time.Duration `json:"max_latency"`
	MaxCost          float64       `json:"max_cost"`
	RequiredTags     []string      `json:"required_tags"`
	ExcludeProviders []string      `json:"exclude_providers"`
	PreferProviders  []string      `json:"prefer_providers"`
}

// RoutingResult contains the routing decision
type RoutingResult struct {
	Provider     string                 `json:"provider"`
	Score        float64                `json:"score"`
	Reason       string                 `json:"reason"`
	Alternatives []Alternative          `json:"alternatives"`
	Metadata     map[string]interface{} `json:"metadata"`
	Latency      time.Duration          `json:"latency"`
}

// Alternative represents an alternative provider choice
type Alternative struct {
	Provider string  `json:"provider"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
}

// RoutingMetrics tracks routing performance
type RoutingMetrics struct {
	totalRoutings    int64
	routingLatency   time.Duration
	providerUsage    map[string]int64
	algorithmUsage   map[string]int64
	latencyByProvider map[string]time.Duration
	costByProvider   map[string]float64
	mutex            sync.RWMutex
}

// NewRoutingEngine creates a new routing engine
func NewRoutingEngine(registry *ProviderRegistry, healthMonitor *HealthMonitor, config *RoutingConfig) *RoutingEngine {
	if config == nil {
		config = &RoutingConfig{
			Strategy: "intelligent",
			Factors: map[string]float64{
				"health":      0.4,
				"latency":     0.3,
				"cost":        0.2,
				"availability": 0.1,
			},
			Preferences: make(map[string]interface{}),
			Fallback:    []string{"round_robin"},
			Rules:       []RoutingRule{},
		}
	}

	re := &RoutingEngine{
		registry:      registry,
		healthMonitor: healthMonitor,
		config:        config,
		algorithms:    make(map[string]RoutingAlgorithm),
		metrics:       NewRoutingMetrics(),
	}

	// Register built-in algorithms
	re.algorithms["round_robin"] = NewRoundRobinAlgorithm()
	re.algorithms["weighted_random"] = NewWeightedRandomAlgorithm()
	re.algorithms["health_based"] = NewHealthBasedAlgorithm(healthMonitor)
	re.algorithms["latency_based"] = NewLatencyBasedAlgorithm(healthMonitor)
	re.algorithms["cost_optimized"] = NewCostOptimizedAlgorithm()
	re.algorithms["intelligent"] = NewIntelligentAlgorithm(healthMonitor, re.metrics)

	return re
}

// NewRoutingMetrics creates a new routing metrics tracker
func NewRoutingMetrics() *RoutingMetrics {
	return &RoutingMetrics{
		providerUsage:     make(map[string]int64),
		algorithmUsage:    make(map[string]int64),
		latencyByProvider: make(map[string]time.Duration),
		costByProvider:    make(map[string]float64),
	}
}

// Route selects the best provider for a request
func (re *RoutingEngine) Route(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	start := time.Now()
	defer func() {
		re.metrics.RecordRoutingLatency(time.Since(start))
	}()

	// Apply routing rules first
	filteredProviders, err := re.applyRoutingRules(req)
	if err != nil {
		return nil, err
	}

	// Get routing algorithm
	algorithm, exists := re.algorithms[re.config.Strategy]
	if !exists {
		return nil, fmt.Errorf("routing algorithm %s not found", re.config.Strategy)
	}

	// Create modified request with filtered providers
	routingReq := *req
	if routingReq.Metadata == nil {
		routingReq.Metadata = make(map[string]interface{})
	}
	routingReq.Metadata["filtered_providers"] = filteredProviders

	// Select provider
	result, err := algorithm.SelectProvider(ctx, &routingReq)
	if err != nil {
		// Try fallback chain
		return re.tryFallback(ctx, req)
	}

	// Record metrics
	re.metrics.RecordRouting(req, result)
	result.Latency = time.Since(start)

	return result, nil
}

// applyRoutingRules filters providers based on routing rules
func (re *RoutingEngine) applyRoutingRules(req *RoutingRequest) ([]string, error) {
	// Get all enabled providers
	providers := re.registry.ListEnabled()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no enabled providers available")
	}

	providerNames := make([]string, 0, len(providers))
	for _, provider := range providers {
		providerNames = append(providerNames, provider.Name())
	}

	// Apply exclusion constraints
	if len(req.Constraints.ExcludeProviders) > 0 {
		filtered := make([]string, 0)
		excludeMap := make(map[string]bool)
		for _, exclude := range req.Constraints.ExcludeProviders {
			excludeMap[exclude] = true
		}

		for _, name := range providerNames {
			if !excludeMap[name] {
				filtered = append(filtered, name)
			}
		}
		providerNames = filtered
	}

	// Apply model support filter
	if req.Model != "" {
		filtered := make([]string, 0)
		for _, name := range providerNames {
			provider, err := re.registry.Get(name)
			if err != nil {
				continue
			}

			models := provider.GetModels()
			for _, model := range models {
				if model.ID == req.Model {
					filtered = append(filtered, name)
					break
				}
			}
		}
		providerNames = filtered
	}

	// Apply constraint filters
	if req.Constraints.MaxLatency > 0 || req.Constraints.MaxCost > 0 {
		filtered := make([]string, 0)
		for _, name := range providerNames {
			if re.meetsConstraints(name, req.Constraints) {
				filtered = append(filtered, name)
			}
		}
		providerNames = filtered
	}

	// Apply rule-based filtering
	for _, rule := range re.config.Rules {
		if re.evaluateRuleCondition(rule.Condition, req) {
			// Intersect with rule providers
			ruleMap := make(map[string]bool)
			for _, provider := range rule.Providers {
				ruleMap[provider] = true
			}

			filtered := make([]string, 0)
			for _, name := range providerNames {
				if ruleMap[name] {
					filtered = append(filtered, name)
				}
			}
			providerNames = filtered
			break // Use first matching rule
		}
	}

	if len(providerNames) == 0 {
		return nil, fmt.Errorf("no providers meet the routing criteria")
	}

	return providerNames, nil
}

// meetsConstraints checks if a provider meets the routing constraints
func (re *RoutingEngine) meetsConstraints(providerName string, constraints RoutingConstraints) bool {
	// Check latency constraint
	if constraints.MaxLatency > 0 {
		avgLatency := re.metrics.GetAverageLatency(providerName)
		if avgLatency > constraints.MaxLatency {
			return false
		}
	}

	// Check cost constraint
	if constraints.MaxCost > 0 {
		avgCost := re.metrics.GetAverageCost(providerName)
		if avgCost > constraints.MaxCost {
			return false
		}
	}

	// Check required tags
	if len(constraints.RequiredTags) > 0 {
		config, err := re.registry.GetConfig(providerName)
		if err != nil {
			return false
		}

		configTags := make(map[string]bool)
		for _, tag := range config.Tags {
			configTags[tag] = true
		}

		for _, required := range constraints.RequiredTags {
			if !configTags[required] {
				return false
			}
		}
	}

	return true
}

// evaluateRuleCondition evaluates a rule condition
func (re *RoutingEngine) evaluateRuleCondition(condition string, req *RoutingRequest) bool {
	// Simple condition evaluation - can be extended with a proper expression parser
	switch condition {
	case "high_priority":
		return req.Priority == PriorityHigh
	case "low_priority":
		return req.Priority == PriorityLow
	case "has_model":
		return req.Model != ""
	default:
		return false
	}
}

// tryFallback attempts to route using fallback algorithms
func (re *RoutingEngine) tryFallback(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	for _, fallbackAlg := range re.config.Fallback {
		algorithm, exists := re.algorithms[fallbackAlg]
		if !exists {
			continue
		}

		result, err := algorithm.SelectProvider(ctx, req)
		if err == nil {
			result.Reason = fmt.Sprintf("Fallback to %s: %s", fallbackAlg, result.Reason)
			return result, nil
		}
	}

	return nil, fmt.Errorf("all routing algorithms failed")
}

// GetMetrics returns routing metrics
func (re *RoutingEngine) GetMetrics() *RoutingMetrics {
	return re.metrics
}

// RegisterAlgorithm registers a custom routing algorithm
func (re *RoutingEngine) RegisterAlgorithm(name string, algorithm RoutingAlgorithm) {
	re.mutex.Lock()
	defer re.mutex.Unlock()
	re.algorithms[name] = algorithm
}

// Intelligent Algorithm Implementation
type IntelligentAlgorithm struct {
	healthMonitor *HealthMonitor
	metrics       *RoutingMetrics
	weights       map[string]float64
	mutex         sync.RWMutex
}

// NewIntelligentAlgorithm creates a new intelligent routing algorithm
func NewIntelligentAlgorithm(healthMonitor *HealthMonitor, metrics *RoutingMetrics) *IntelligentAlgorithm {
	return &IntelligentAlgorithm{
		healthMonitor: healthMonitor,
		metrics:       metrics,
		weights: map[string]float64{
			"health":      0.4,
			"latency":     0.3,
			"cost":        0.2,
			"availability": 0.1,
		},
	}
}

func (ia *IntelligentAlgorithm) Name() string {
	return "intelligent"
}

func (ia *IntelligentAlgorithm) Configure(config map[string]interface{}) error {
	ia.mutex.Lock()
	defer ia.mutex.Unlock()

	if weights, ok := config["weights"].(map[string]float64); ok {
		for key, value := range weights {
			ia.weights[key] = value
		}
	}

	return nil
}

func (ia *IntelligentAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	providers, ok := req.Metadata["filtered_providers"].([]string)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	scores := make(map[string]float64)
	alternatives := make([]Alternative, 0)

	for _, providerName := range providers {
		score := ia.calculateProviderScore(ctx, providerName, req)
		scores[providerName] = score

		alternatives = append(alternatives, Alternative{
			Provider: providerName,
			Score:    score,
			Reason:   ia.getScoreReason(providerName, score),
		})
	}

	// Sort alternatives by score
	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Score > alternatives[j].Score
	})

	bestProvider := alternatives[0].Provider

	return &RoutingResult{
		Provider:     bestProvider,
		Score:        scores[bestProvider],
		Reason:       fmt.Sprintf("Best overall score: %.3f", scores[bestProvider]),
		Alternatives: alternatives[1:], // Exclude the selected provider
		Metadata: map[string]interface{}{
			"algorithm": "intelligent",
			"factors":   ia.weights,
		},
	}, nil
}

func (ia *IntelligentAlgorithm) calculateProviderScore(ctx context.Context, providerName string, req *RoutingRequest) float64 {
	var totalScore float64

	// Health score
	if ia.healthMonitor != nil {
		if health, err := ia.healthMonitor.GetProviderHealth(providerName); err == nil {
			totalScore += health.HealthScore * ia.weights["health"]
		}
	}

	// Latency score
	avgLatency := ia.metrics.GetAverageLatency(providerName)
	latencyScore := math.Max(0, 1.0-float64(avgLatency.Milliseconds())/2000.0)
	totalScore += latencyScore * ia.weights["latency"]

	// Cost score (inverse - lower cost = higher score)
	avgCost := ia.metrics.GetAverageCost(providerName)
	costScore := math.Max(0, 1.0-avgCost/10.0) // Normalize assuming max cost of $10
	totalScore += costScore * ia.weights["cost"]

	// Availability score
	availability := ia.metrics.GetAvailability(providerName)
	totalScore += availability * ia.weights["availability"]

	// Apply request-specific adjustments
	totalScore = ia.applyRequestAdjustments(totalScore, providerName, req)

	return math.Max(0, math.Min(1, totalScore))
}

func (ia *IntelligentAlgorithm) applyRequestAdjustments(score float64, providerName string, req *RoutingRequest) float64 {
	// Priority adjustments
	switch req.Priority {
	case PriorityHigh:
		// Prefer providers with better health for high priority
		if ia.healthMonitor != nil {
			if health, err := ia.healthMonitor.GetProviderHealth(providerName); err == nil {
				score += health.HealthScore * 0.1
			}
		}
	case PriorityLow:
		// For low priority, prefer cost-effective providers
		avgCost := ia.metrics.GetAverageCost(providerName)
		if avgCost < 1.0 {
			score += 0.1
		}
	}

	// Model-specific adjustments
	if req.Model != "" {
		modelSupport := ia.metrics.GetModelSupport(providerName, req.Model)
		score *= modelSupport
	}

	// Preferred providers boost
	for _, preferred := range req.Constraints.PreferProviders {
		if preferred == providerName {
			score += 0.15
			break
		}
	}

	return score
}

func (ia *IntelligentAlgorithm) getScoreReason(providerName string, score float64) string {
	if score > 0.8 {
		return "Excellent overall performance"
	} else if score > 0.6 {
		return "Good performance with minor issues"
	} else if score > 0.4 {
		return "Moderate performance"
	} else {
		return "Poor performance, consider alternatives"
	}
}

// Round Robin Algorithm
type RoundRobinAlgorithm struct {
	counter int64
	mutex   sync.Mutex
}

func NewRoundRobinAlgorithm() *RoundRobinAlgorithm {
	return &RoundRobinAlgorithm{}
}

func (rr *RoundRobinAlgorithm) Name() string {
	return "round_robin"
}

func (rr *RoundRobinAlgorithm) Configure(config map[string]interface{}) error {
	return nil // No configuration needed
}

func (rr *RoundRobinAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	providers, ok := req.Metadata["filtered_providers"].([]string)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	rr.mutex.Lock()
	index := int(rr.counter % int64(len(providers)))
	rr.counter++
	rr.mutex.Unlock()

	selectedProvider := providers[index]

	return &RoutingResult{
		Provider: selectedProvider,
		Score:    1.0, // Equal weight for all providers
		Reason:   fmt.Sprintf("Round-robin selection (index %d of %d)", index, len(providers)),
		Metadata: map[string]interface{}{
			"algorithm": "round_robin",
			"index":     index,
		},
	}, nil
}

// Weighted Random Algorithm
type WeightedRandomAlgorithm struct {
	rand *rand.Rand
	mutex sync.Mutex
}

func NewWeightedRandomAlgorithm() *WeightedRandomAlgorithm {
	return &WeightedRandomAlgorithm{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (wr *WeightedRandomAlgorithm) Name() string {
	return "weighted_random"
}

func (wr *WeightedRandomAlgorithm) Configure(config map[string]interface{}) error {
	return nil
}

func (wr *WeightedRandomAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	providers, ok := req.Metadata["filtered_providers"].([]string)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// For simplicity, use equal weights - could be enhanced with actual provider weights
	wr.mutex.Lock()
	index := wr.rand.Intn(len(providers))
	wr.mutex.Unlock()

	selectedProvider := providers[index]

	return &RoutingResult{
		Provider: selectedProvider,
		Score:    1.0,
		Reason:   "Weighted random selection",
		Metadata: map[string]interface{}{
			"algorithm": "weighted_random",
		},
	}, nil
}

// Health-Based Algorithm
type HealthBasedAlgorithm struct {
	healthMonitor *HealthMonitor
}

func NewHealthBasedAlgorithm(healthMonitor *HealthMonitor) *HealthBasedAlgorithm {
	return &HealthBasedAlgorithm{
		healthMonitor: healthMonitor,
	}
}

func (hb *HealthBasedAlgorithm) Name() string {
	return "health_based"
}

func (hb *HealthBasedAlgorithm) Configure(config map[string]interface{}) error {
	return nil
}

func (hb *HealthBasedAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	providers, ok := req.Metadata["filtered_providers"].([]string)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	if hb.healthMonitor == nil {
		return nil, fmt.Errorf("health monitoring not available")
	}

	bestProvider := ""
	bestScore := -1.0

	for _, providerName := range providers {
		if health, err := hb.healthMonitor.GetProviderHealth(providerName); err == nil {
			if health.HealthScore > bestScore {
				bestScore = health.HealthScore
				bestProvider = providerName
			}
		}
	}

	if bestProvider == "" {
		return nil, fmt.Errorf("no healthy providers found")
	}

	return &RoutingResult{
		Provider: bestProvider,
		Score:    bestScore,
		Reason:   fmt.Sprintf("Best health score: %.3f", bestScore),
		Metadata: map[string]interface{}{
			"algorithm": "health_based",
		},
	}, nil
}

// Latency-Based Algorithm
type LatencyBasedAlgorithm struct {
	healthMonitor *HealthMonitor
}

func NewLatencyBasedAlgorithm(healthMonitor *HealthMonitor) *LatencyBasedAlgorithm {
	return &LatencyBasedAlgorithm{
		healthMonitor: healthMonitor,
	}
}

func (lb *LatencyBasedAlgorithm) Name() string {
	return "latency_based"
}

func (lb *LatencyBasedAlgorithm) Configure(config map[string]interface{}) error {
	return nil
}

func (lb *LatencyBasedAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	providers, ok := req.Metadata["filtered_providers"].([]string)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	bestProvider := ""
	bestLatency := time.Duration(math.MaxInt64)

	for _, providerName := range providers {
		if health, err := lb.healthMonitor.GetProviderHealth(providerName); err == nil {
			if len(health.RecentChecks) > 0 {
				avgLatency := time.Duration(0)
				for _, check := range health.RecentChecks {
					avgLatency += check.Latency
				}
				avgLatency /= time.Duration(len(health.RecentChecks))

				if avgLatency < bestLatency {
					bestLatency = avgLatency
					bestProvider = providerName
				}
			}
		}
	}

	if bestProvider == "" {
		return nil, fmt.Errorf("no providers with latency data found")
	}

	score := math.Max(0, 1.0-float64(bestLatency.Milliseconds())/2000.0)

	return &RoutingResult{
		Provider: bestProvider,
		Score:    score,
		Reason:   fmt.Sprintf("Best latency: %v", bestLatency),
		Metadata: map[string]interface{}{
			"algorithm": "latency_based",
			"latency":   bestLatency.Milliseconds(),
		},
	}, nil
}

// Cost-Optimized Algorithm
type CostOptimizedAlgorithm struct{}

func NewCostOptimizedAlgorithm() *CostOptimizedAlgorithm {
	return &CostOptimizedAlgorithm{}
}

func (co *CostOptimizedAlgorithm) Name() string {
	return "cost_optimized"
}

func (co *CostOptimizedAlgorithm) Configure(config map[string]interface{}) error {
	return nil
}

func (co *CostOptimizedAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
	providers, ok := req.Metadata["filtered_providers"].([]string)
	if !ok || len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// For now, prefer local/mock providers (lowest cost)
	// This could be enhanced with actual cost data
	for _, providerName := range providers {
		if providerName == "local" || providerName == "mock" {
			return &RoutingResult{
				Provider: providerName,
				Score:    1.0,
				Reason:   "Lowest cost provider",
				Metadata: map[string]interface{}{
					"algorithm": "cost_optimized",
				},
			}, nil
		}
	}

	// If no local/mock providers, select first available
	return &RoutingResult{
		Provider: providers[0],
		Score:    0.5,
		Reason:   "First available provider (cost data unavailable)",
		Metadata: map[string]interface{}{
			"algorithm": "cost_optimized",
		},
	}, nil
}

// Routing Metrics Methods
func (rm *RoutingMetrics) RecordRouting(req *RoutingRequest, result *RoutingResult) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.totalRoutings++
	rm.providerUsage[result.Provider]++
	
	if algorithm, ok := result.Metadata["algorithm"].(string); ok {
		rm.algorithmUsage[algorithm]++
	}
}

func (rm *RoutingMetrics) RecordRoutingLatency(latency time.Duration) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.routingLatency += latency
}

func (rm *RoutingMetrics) GetAverageLatency(providerName string) time.Duration {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if latency, exists := rm.latencyByProvider[providerName]; exists {
		return latency
	}
	return 100 * time.Millisecond // Default latency
}

func (rm *RoutingMetrics) GetAverageCost(providerName string) float64 {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	if cost, exists := rm.costByProvider[providerName]; exists {
		return cost
	}
	return 0.01 // Default cost
}

func (rm *RoutingMetrics) GetAvailability(providerName string) float64 {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	// Simple availability calculation based on usage
	usage := rm.providerUsage[providerName]
	if usage > 10 {
		return 0.95 // High availability for frequently used providers
	}
	return 0.85 // Default availability
}

func (rm *RoutingMetrics) GetModelSupport(providerName, model string) float64 {
	// For now, assume all providers support all models equally
	// This could be enhanced with actual model compatibility data
	return 1.0
}

func (rm *RoutingMetrics) GetOverallStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	avgRoutingLatency := time.Duration(0)
	if rm.totalRoutings > 0 {
		avgRoutingLatency = rm.routingLatency / time.Duration(rm.totalRoutings)
	}

	return map[string]interface{}{
		"total_routings":          rm.totalRoutings,
		"average_routing_latency": avgRoutingLatency.Milliseconds(),
		"provider_usage":          rm.providerUsage,
		"algorithm_usage":         rm.algorithmUsage,
	}
}