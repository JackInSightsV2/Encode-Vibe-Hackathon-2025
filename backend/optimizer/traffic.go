package optimizer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"
)

// TrafficRouter handles routing requests to different rule variants in experiments
type TrafficRouter struct {
	experiments   map[string]*TrafficExperiment
	userAllocations map[string]map[string]string // userID -> experimentID -> variantID
	mu            sync.RWMutex
	hashSeed      uint64
}

// TrafficExperiment represents experiment configuration for traffic routing
type TrafficExperiment struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	IsActive      bool                   `json:"is_active"`
	TrafficSplit  TrafficSplit          `json:"traffic_split"`
	Rules         map[string]Rule        `json:"rules"` // variantID -> Rule
	Constraints   *TrafficConstraints    `json:"constraints"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	StickySession bool                   `json:"sticky_session"`
}

// TrafficConstraints define routing constraints
type TrafficConstraints struct {
	AllowedUserTypes    []string               `json:"allowed_user_types"`
	BlockedUserTypes    []string               `json:"blocked_user_types"`
	AllowedCountries    []string               `json:"allowed_countries"`
	BlockedCountries    []string               `json:"blocked_countries"`
	MinAccountAge       *time.Duration         `json:"min_account_age"`
	MaxRequestsPerHour  int                    `json:"max_requests_per_hour"`
	RequiredFeatureFlags []string              `json:"required_feature_flags"`
	CustomFilters       map[string]interface{} `json:"custom_filters"`
}

// RuleAssignment represents the assignment of a request to a rule variant
type RuleAssignment struct {
	RequestID    string                 `json:"request_id"`
	UserID       string                 `json:"user_id"`
	ExperimentID string                 `json:"experiment_id"`
	VariantID    string                 `json:"variant_id"`
	AssignedAt   time.Time              `json:"assigned_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TrafficMetrics tracks traffic distribution metrics
type TrafficMetrics struct {
	ExperimentID      string                    `json:"experiment_id"`
	TotalRequests     int64                     `json:"total_requests"`
	VariantCounts     map[string]int64          `json:"variant_counts"`
	VariantRates      map[string]float64        `json:"variant_rates"`
	LastUpdated       time.Time                 `json:"last_updated"`
	RequestsPerSecond float64                   `json:"requests_per_second"`
	ErrorRate         float64                   `json:"error_rate"`
}

// NewTrafficRouter creates a new traffic router
func NewTrafficRouter() *TrafficRouter {
	return &TrafficRouter{
		experiments:     make(map[string]*TrafficExperiment),
		userAllocations: make(map[string]map[string]string),
		hashSeed:        uint64(time.Now().UnixNano()),
	}
}

// RegisterExperiment registers an experiment for traffic routing
func (tr *TrafficRouter) RegisterExperiment(experiment *Experiment) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	// Convert experiment to traffic experiment
	trafficExp := &TrafficExperiment{
		ID:            experiment.ID,
		Name:          experiment.Name,
		IsActive:      false, // Will be activated when started
		TrafficSplit:  experiment.Traffic,
		Rules:         make(map[string]Rule),
		StartTime:     experiment.StartTime,
		StickySession: true, // Default to sticky sessions
	}
	
	// Add control rule
	trafficExp.Rules["control"] = experiment.Control
	
	// Add variant rules
	for _, variant := range experiment.Variants {
		trafficExp.Rules[variant.ID] = variant
	}
	
	tr.experiments[experiment.ID] = trafficExp
	
	return nil
}

// StartTrafficAllocation starts routing traffic to an experiment
func (tr *TrafficRouter) StartTrafficAllocation(experimentID string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	experiment, exists := tr.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	experiment.IsActive = true
	experiment.StartTime = time.Now()
	
	return nil
}

// StopTrafficAllocation stops routing traffic to an experiment
func (tr *TrafficRouter) StopTrafficAllocation(experimentID string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	experiment, exists := tr.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	experiment.IsActive = false
	endTime := time.Now()
	experiment.EndTime = &endTime
	
	return nil
}

// RouteRequest routes a request to the appropriate rule variant
func (tr *TrafficRouter) RouteRequest(requestID, userID string) (*RuleAssignment, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	// Find active experiments
	activeExperiments := make([]*TrafficExperiment, 0)
	for _, experiment := range tr.experiments {
		if experiment.IsActive {
			activeExperiments = append(activeExperiments, experiment)
		}
	}
	
	if len(activeExperiments) == 0 {
		return nil, fmt.Errorf("no active experiments")
	}
	
	// For simplicity, route to the first active experiment
	// In a production system, you might have more sophisticated logic
	experiment := activeExperiments[0]
	
	// Check if user is already assigned to this experiment (sticky sessions)
	if experiment.StickySession {
		if userExperiments, exists := tr.userAllocations[userID]; exists {
			if variantID, assigned := userExperiments[experiment.ID]; assigned {
				return &RuleAssignment{
					RequestID:    requestID,
					UserID:       userID,
					ExperimentID: experiment.ID,
					VariantID:    variantID,
					AssignedAt:   time.Now(),
					Metadata: map[string]interface{}{
						"sticky_session": true,
						"allocation_method": "existing_assignment",
					},
				}, nil
			}
		}
	}
	
	// Check constraints
	if experiment.Constraints != nil {
		if allowed, reason := tr.checkConstraints(userID, experiment.Constraints); !allowed {
			return nil, fmt.Errorf("user does not meet experiment constraints: %s", reason)
		}
	}
	
	// Assign user to variant based on traffic split
	variantID := tr.assignVariant(userID, experiment.ID, experiment.TrafficSplit)
	
	// Store assignment for sticky sessions
	if experiment.StickySession {
		if tr.userAllocations[userID] == nil {
			tr.userAllocations[userID] = make(map[string]string)
		}
		tr.userAllocations[userID][experiment.ID] = variantID
	}
	
	assignment := &RuleAssignment{
		RequestID:    requestID,
		UserID:       userID,
		ExperimentID: experiment.ID,
		VariantID:    variantID,
		AssignedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"sticky_session": experiment.StickySession,
			"allocation_method": "hash_based",
			"hash_bucket": tr.getUserHashBucket(userID, experiment.ID),
		},
	}
	
	return assignment, nil
}

// assignVariant assigns a user to a variant based on traffic split
func (tr *TrafficRouter) assignVariant(userID, experimentID string, trafficSplit TrafficSplit) string {
	// Get deterministic hash for user in this experiment
	hashValue := tr.getUserHash(userID, experimentID)
	
	// Convert to 0-1 range
	bucket := float64(hashValue) / float64(^uint32(0))
	
	// Assign based on cumulative traffic distribution
	cumulative := 0.0
	
	// Check control first
	cumulative += trafficSplit.Control
	if bucket <= cumulative {
		return "control"
	}
	
	// Check variants
	for variantID, traffic := range trafficSplit.Variants {
		cumulative += traffic
		if bucket <= cumulative {
			return variantID
		}
	}
	
	// Fallback to control (shouldn't happen with proper traffic splits)
	return "control"
}

// getUserHash gets a deterministic hash for a user in an experiment
func (tr *TrafficRouter) getUserHash(userID, experimentID string) uint32 {
	hasher := fnv.New32a()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:%d", userID, experimentID, tr.hashSeed)))
	return hasher.Sum32()
}

// getUserHashBucket returns the hash bucket (0-100) for a user
func (tr *TrafficRouter) getUserHashBucket(userID, experimentID string) int {
	hash := tr.getUserHash(userID, experimentID)
	return int((float64(hash) / float64(^uint32(0))) * 100)
}

// checkConstraints checks if a user meets experiment constraints
func (tr *TrafficRouter) checkConstraints(userID string, constraints *TrafficConstraints) (bool, string) {
	// In a real implementation, these checks would integrate with user service
	// For now, we'll do basic validation
	
	// Check blocked user types (if we had user type info)
	for _, blockedType := range constraints.BlockedUserTypes {
		if blockedType == "test_user" && userID == "test_user" {
			return false, "user type is blocked"
		}
	}
	
	// Check allowed user types
	if len(constraints.AllowedUserTypes) > 0 {
		// If allowlist is specified, user must be in it
		allowed := false
		for _, allowedType := range constraints.AllowedUserTypes {
			if allowedType == "all" || (allowedType == "premium" && userID != "free_user") {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, "user type not in allowlist"
		}
	}
	
	// Additional constraint checks would go here
	
	return true, ""
}

// GetTrafficMetrics returns traffic metrics for an experiment
func (tr *TrafficRouter) GetTrafficMetrics(experimentID string) (*TrafficMetrics, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	experiment, exists := tr.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	// In a real implementation, these metrics would be collected from actual traffic
	// For now, we'll return simulated metrics
	metrics := &TrafficMetrics{
		ExperimentID:  experimentID,
		TotalRequests: tr.calculateTotalRequests(experimentID),
		VariantCounts: tr.calculateVariantCounts(experimentID),
		LastUpdated:   time.Now(),
	}
	
	// Calculate variant rates
	metrics.VariantRates = make(map[string]float64)
	if metrics.TotalRequests > 0 {
		for variantID, count := range metrics.VariantCounts {
			metrics.VariantRates[variantID] = float64(count) / float64(metrics.TotalRequests)
		}
	}
	
	// Calculate requests per second (based on experiment duration)
	if experiment.IsActive {
		duration := time.Since(experiment.StartTime)
		if duration.Seconds() > 0 {
			metrics.RequestsPerSecond = float64(metrics.TotalRequests) / duration.Seconds()
		}
	}
	
	return metrics, nil
}

// calculateTotalRequests calculates total requests for an experiment
func (tr *TrafficRouter) calculateTotalRequests(experimentID string) int64 {
	// In a real implementation, this would query metrics database
	// For simulation, return a reasonable number
	return int64(rand.Intn(10000) + 1000)
}

// calculateVariantCounts calculates request counts per variant
func (tr *TrafficRouter) calculateVariantCounts(experimentID string) map[string]int64 {
	experiment, exists := tr.experiments[experimentID]
	if !exists {
		return make(map[string]int64)
	}
	
	totalRequests := tr.calculateTotalRequests(experimentID)
	counts := make(map[string]int64)
	
	// Simulate counts based on traffic split
	controlCount := int64(float64(totalRequests) * experiment.TrafficSplit.Control)
	counts["control"] = controlCount
	
	remaining := totalRequests - controlCount
	for variantID, traffic := range experiment.TrafficSplit.Variants {
		variantCount := int64(float64(totalRequests) * traffic)
		counts[variantID] = variantCount
		remaining -= variantCount
	}
	
	// Add any remaining to control to ensure total adds up
	if remaining > 0 {
		counts["control"] += remaining
	}
	
	return counts
}

// UpdateTrafficSplit updates the traffic allocation for an experiment
func (tr *TrafficRouter) UpdateTrafficSplit(experimentID string, newSplit TrafficSplit) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	experiment, exists := tr.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	// Validate new traffic split
	total := newSplit.Control
	for _, traffic := range newSplit.Variants {
		total += traffic
	}
	
	if total < 0.99 || total > 1.01 {
		return fmt.Errorf("traffic split must sum to 1.0, got %.3f", total)
	}
	
	experiment.TrafficSplit = newSplit
	
	// Clear user allocations to force re-allocation with new split
	// Only do this if sticky sessions are disabled or if we want to rebalance
	if !experiment.StickySession {
		for userID := range tr.userAllocations {
			if tr.userAllocations[userID] != nil {
				delete(tr.userAllocations[userID], experimentID)
			}
		}
	}
	
	return nil
}

// SetExperimentConstraints sets constraints for an experiment
func (tr *TrafficRouter) SetExperimentConstraints(experimentID string, constraints *TrafficConstraints) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	experiment, exists := tr.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	experiment.Constraints = constraints
	return nil
}

// GetUserAssignments returns all variant assignments for a user
func (tr *TrafficRouter) GetUserAssignments(userID string) map[string]string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	if assignments, exists := tr.userAllocations[userID]; exists {
		// Return a copy to avoid concurrent access issues
		result := make(map[string]string)
		for experimentID, variantID := range assignments {
			result[experimentID] = variantID
		}
		return result
	}
	
	return make(map[string]string)
}

// ClearUserAssignments clears all assignments for a user
func (tr *TrafficRouter) ClearUserAssignments(userID string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	delete(tr.userAllocations, userID)
}

// GetActiveExperiments returns all active experiments
func (tr *TrafficRouter) GetActiveExperiments() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	activeExperiments := make([]string, 0)
	for experimentID, experiment := range tr.experiments {
		if experiment.IsActive {
			activeExperiments = append(activeExperiments, experimentID)
		}
	}
	
	return activeExperiments
}

// ValidateTrafficSplit validates that a traffic split is valid
func ValidateTrafficSplit(split TrafficSplit) error {
	total := split.Control
	for _, traffic := range split.Variants {
		total += traffic
		
		if traffic < 0 || traffic > 1 {
			return fmt.Errorf("variant traffic must be between 0 and 1")
		}
	}
	
	if split.Control < 0 || split.Control > 1 {
		return fmt.Errorf("control traffic must be between 0 and 1")
	}
	
	if total < 0.99 || total > 1.01 {
		return fmt.Errorf("traffic split must sum to 1.0, got %.3f", total)
	}
	
	return nil
}

// GenerateTrafficAllocation generates a balanced traffic allocation
func GenerateTrafficAllocation(variantCount int, controlPercent float64) TrafficSplit {
	if controlPercent < 0 || controlPercent > 1 {
		controlPercent = 0.5 // Default to 50% control
	}
	
	if variantCount <= 0 {
		return TrafficSplit{
			Control:  1.0,
			Variants: make(map[string]float64),
		}
	}
	
	variantPercent := (1.0 - controlPercent) / float64(variantCount)
	
	variants := make(map[string]float64)
	for i := 0; i < variantCount; i++ {
		variants[fmt.Sprintf("variant_%d", i+1)] = variantPercent
	}
	
	return TrafficSplit{
		Control:  controlPercent,
		Variants: variants,
	}
}

// CalculateMinSampleSize calculates minimum sample size for statistical significance
func CalculateMinSampleSize(baseRate, minDetectableEffect, alpha, power float64) int {
	// Simplified sample size calculation for A/B tests
	// This is a basic implementation - in production you'd use more sophisticated formulas
	
	if baseRate <= 0 || baseRate >= 1 {
		baseRate = 0.5 // Default base rate
	}
	
	if minDetectableEffect <= 0 {
		minDetectableEffect = 0.05 // 5% minimum effect
	}
	
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.05 // 5% significance level
	}
	
	if power <= 0 || power >= 1 {
		power = 0.8 // 80% power
	}
	
	// Simplified calculation (normally you'd use statistical libraries)
	variance := baseRate * (1 - baseRate)
	effectSize := minDetectableEffect / baseRate
	
	// Z-scores for alpha and power
	zAlpha := 1.96  // for alpha = 0.05
	zPower := 0.84  // for power = 0.8
	
	sampleSize := (2 * variance * (zAlpha + zPower) * (zAlpha + zPower)) / (effectSize * effectSize * baseRate * baseRate)
	
	return int(sampleSize) + 1 // Add 1 to ensure we round up
}

// HashConsistentAllocation provides consistent allocation using MD5 hash
func HashConsistentAllocation(userID, experimentID string, splits map[string]float64) string {
	// Create deterministic hash
	hasher := md5.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s", userID, experimentID)))
	hash := hex.EncodeToString(hasher.Sum(nil))
	
	// Convert first 8 chars of hash to number
	hashInt := uint64(0)
	for i := 0; i < 8 && i < len(hash); i++ {
		hashInt = hashInt*16 + uint64(hash[i])
		if hash[i] >= '0' && hash[i] <= '9' {
			hashInt += uint64(hash[i] - '0')
		} else if hash[i] >= 'a' && hash[i] <= 'f' {
			hashInt += uint64(hash[i] - 'a' + 10)
		}
	}
	
	// Convert to 0-1 range
	bucket := float64(hashInt%1000000) / 1000000.0
	
	// Assign based on cumulative distribution
	cumulative := 0.0
	for variant, split := range splits {
		cumulative += split
		if bucket <= cumulative {
			return variant
		}
	}
	
	// Fallback to first variant
	for variant := range splits {
		return variant
	}
	
	return "control"
}