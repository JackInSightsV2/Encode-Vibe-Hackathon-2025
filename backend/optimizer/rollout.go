package optimizer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RolloutManager manages gradual rollouts of rule changes
type RolloutManager struct {
	rollouts         map[string]*ActiveRollout
	trafficController *TrafficController
	metricsCollector  *MetricsCollector
	alertManager      *AlertManager
	mu                sync.RWMutex
	config            *RolloutConfig
}

// RolloutConfig configures rollout behavior
type RolloutConfig struct {
	DefaultStrategy           RolloutStrategy   `json:"default_strategy"`
	PhaseIntervals           map[string]time.Duration `json:"phase_intervals"`
	MetricsCollectionInterval time.Duration     `json:"metrics_collection_interval"`
	HealthCheckInterval       time.Duration     `json:"health_check_interval"`
	AutoRollbackEnabled       bool              `json:"auto_rollback_enabled"`
	RollbackThresholds        *RollbackThresholds `json:"rollback_thresholds"`
	MaxConcurrentRollouts     int               `json:"max_concurrent_rollouts"`
	RequireHealthChecks       bool              `json:"require_health_checks"`
}

// RollbackThresholds define when to automatically rollback
type RollbackThresholds struct {
	MaxErrorRateIncrease      float64 `json:"max_error_rate_increase"`
	MaxResponseTimeIncrease   float64 `json:"max_response_time_increase"`
	MinDetectionRateDecrease  float64 `json:"min_detection_rate_decrease"`
	MaxFalsePositiveIncrease  float64 `json:"max_false_positive_increase"`
	UserSatisfactionDecrease  float64 `json:"user_satisfaction_decrease"`
}

// ActiveRollout represents an ongoing rollout
type ActiveRollout struct {
	ID                string                 `json:"id"`
	DeploymentID      string                 `json:"deployment_id"`
	PromotionID       string                 `json:"promotion_id"`
	Strategy          RolloutStrategy        `json:"strategy"`
	Status            RolloutStatus          `json:"status"`
	CurrentPhase      *RolloutPhase          `json:"current_phase"`
	Phases            []RolloutPhase         `json:"phases"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	RollbackPlan      *RollbackPlan          `json:"rollback_plan"`
	Metrics           *RolloutMetrics        `json:"metrics"`
	HealthChecks      []HealthCheck          `json:"health_checks"`
	AlertsTriggered   []Alert                `json:"alerts_triggered"`
	Metadata          map[string]interface{} `json:"metadata"`
	
	// Internal state
	cancelFunc        context.CancelFunc     `json:"-"`
	mu                sync.RWMutex           `json:"-"`
}

// RolloutPhase represents a phase in the rollout
type RolloutPhase struct {
	Name              string                 `json:"name"`
	TrafficPercent    float64                `json:"traffic_percent"`
	Duration          time.Duration          `json:"duration"`
	Status            PhaseStatus            `json:"status"`
	StartedAt         *time.Time             `json:"started_at,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	Metrics           *PhaseMetrics          `json:"metrics,omitempty"`
	HealthChecksPassed bool                  `json:"health_checks_passed"`
	Conditions        []PhaseCondition       `json:"conditions"`
}

// PhaseStatus represents the status of a rollout phase
type PhaseStatus string

const (
	PhaseStatusPending    PhaseStatus = "pending"
	PhaseStatusActive     PhaseStatus = "active"
	PhaseStatusCompleted  PhaseStatus = "completed"
	PhaseStatusFailed     PhaseStatus = "failed"
	PhaseStatusRolledBack PhaseStatus = "rolled_back"
)

// PhaseCondition defines conditions that must be met to proceed
type PhaseCondition struct {
	Type        string      `json:"type"`
	Metric      string      `json:"metric"`
	Operator    string      `json:"operator"`
	Threshold   float64     `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Description string      `json:"description"`
	Met         bool        `json:"met"`
	LastChecked time.Time   `json:"last_checked"`
}

// TrafficController manages traffic routing during rollouts
type TrafficController struct {
	activeRoutes map[string]*TrafficRoute
	mu           sync.RWMutex
}

// TrafficRoute represents active traffic routing
type TrafficRoute struct {
	RolloutID        string    `json:"rollout_id"`
	OldRuleID        string    `json:"old_rule_id"`
	NewRuleID        string    `json:"new_rule_id"`
	TrafficPercent   float64   `json:"traffic_percent"`
	LastUpdated      time.Time `json:"last_updated"`
	RequestsRouted   int64     `json:"requests_routed"`
	ErrorCount       int64     `json:"error_count"`
}

// MetricsCollector collects metrics during rollouts
type MetricsCollector struct {
	rolloutMetrics map[string]*RolloutMetrics
	mu             sync.RWMutex
}

// RolloutMetrics contains metrics for a rollout
type RolloutMetrics struct {
	RolloutID           string              `json:"rollout_id"`
	TotalRequests       int64               `json:"total_requests"`
	NewRuleRequests     int64               `json:"new_rule_requests"`
	OldRuleRequests     int64               `json:"old_rule_requests"`
	ErrorRate           float64             `json:"error_rate"`
	AvgResponseTime     time.Duration       `json:"avg_response_time"`
	DetectionRate       float64             `json:"detection_rate"`
	FalsePositiveRate   float64             `json:"false_positive_rate"`
	UserSatisfaction    float64             `json:"user_satisfaction"`
	PhaseMetrics        map[string]*PhaseMetrics `json:"phase_metrics"`
	LastUpdated         time.Time           `json:"last_updated"`
}

// PhaseMetrics contains metrics for a specific phase
type PhaseMetrics struct {
	PhaseName           string        `json:"phase_name"`
	Requests            int64         `json:"requests"`
	Errors              int64         `json:"errors"`
	AvgResponseTime     time.Duration `json:"avg_response_time"`
	DetectionRate       float64       `json:"detection_rate"`
	FalsePositiveRate   float64       `json:"false_positive_rate"`
	UserSatisfaction    float64       `json:"user_satisfaction"`
	StartTime           time.Time     `json:"start_time"`
	EndTime             *time.Time    `json:"end_time,omitempty"`
}

// HealthCheck represents a health check
type HealthCheck struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      HealthCheckStatus      `json:"status"`
	LastRun     time.Time              `json:"last_run"`
	Duration    time.Duration          `json:"duration"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// HealthCheckStatus represents health check status
type HealthCheckStatus string

const (
	HealthCheckStatusPass    HealthCheckStatus = "pass"
	HealthCheckStatusFail    HealthCheckStatus = "fail"
	HealthCheckStatusWarning HealthCheckStatus = "warning"
	HealthCheckStatusUnknown HealthCheckStatus = "unknown"
)

// AlertManager manages alerts during rollouts
type AlertManager struct {
	alerts []Alert
	mu     sync.RWMutex
}

// Alert represents an alert triggered during rollout
type Alert struct {
	ID          string                 `json:"id"`
	RolloutID   string                 `json:"rollout_id"`
	Type        string                 `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	Metric      string                 `json:"metric"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	TriggeredAt time.Time              `json:"triggered_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// NewRolloutManager creates a new rollout manager
func NewRolloutManager(config *RolloutConfig) *RolloutManager {
	if config == nil {
		config = GetDefaultRolloutConfig()
	}
	
	return &RolloutManager{
		rollouts:          make(map[string]*ActiveRollout),
		trafficController: NewTrafficController(),
		metricsCollector:  NewMetricsCollector(),
		alertManager:      NewAlertManager(),
		config:            config,
	}
}

// GetDefaultRolloutConfig returns default rollout configuration
func GetDefaultRolloutConfig() *RolloutConfig {
	return &RolloutConfig{
		DefaultStrategy: RolloutGradual,
		PhaseIntervals: map[string]time.Duration{
			"5%":   10 * time.Minute,
			"10%":  10 * time.Minute,
			"25%":  15 * time.Minute,
			"50%":  20 * time.Minute,
			"100%": 30 * time.Minute,
		},
		MetricsCollectionInterval: 30 * time.Second,
		HealthCheckInterval:       1 * time.Minute,
		AutoRollbackEnabled:       true,
		RollbackThresholds: &RollbackThresholds{
			MaxErrorRateIncrease:     0.05, // 5% increase
			MaxResponseTimeIncrease:  0.5,  // 50% increase
			MinDetectionRateDecrease: 0.02, // 2% decrease
			MaxFalsePositiveIncrease: 0.03, // 3% increase
			UserSatisfactionDecrease: 0.1,  // 10% decrease
		},
		MaxConcurrentRollouts: 3,
		RequireHealthChecks:   true,
	}
}

// StartRollout starts a new rollout
func (rm *RolloutManager) StartRollout(ctx context.Context, request StartRolloutRequest) (*ActiveRollout, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Check concurrent rollout limit
	if len(rm.rollouts) >= rm.config.MaxConcurrentRollouts {
		return nil, fmt.Errorf("maximum concurrent rollouts reached: %d", rm.config.MaxConcurrentRollouts)
	}
	
	// Create rollout context with cancellation
	rolloutCtx, cancelFunc := context.WithCancel(ctx)
	
	// Create active rollout
	rollout := &ActiveRollout{
		ID:           generateRolloutID(),
		DeploymentID: request.DeploymentID,
		PromotionID:  request.PromotionID,
		Strategy:     request.Strategy,
		Status:       RolloutStatus("starting"),
		StartedAt:    time.Now(),
		Phases:       rm.createRolloutPhases(request.Strategy),
		Metadata:     request.Metadata,
		cancelFunc:   cancelFunc,
	}
	
	// Create rollback plan
	rollout.RollbackPlan = &RollbackPlan{
		ID:           generateRollbackID(),
		DeploymentID: request.DeploymentID,
		CreatedAt:    time.Now(),
	}
	
	// Initialize metrics
	rollout.Metrics = &RolloutMetrics{
		RolloutID:    rollout.ID,
		PhaseMetrics: make(map[string]*PhaseMetrics),
		LastUpdated:  time.Now(),
	}
	
	// Store rollout
	rm.rollouts[rollout.ID] = rollout
	
	// Start rollout execution
	go rm.executeRollout(rolloutCtx, rollout)
	
	return rollout, nil
}

// executeRollout executes the rollout phases
func (rm *RolloutManager) executeRollout(ctx context.Context, rollout *ActiveRollout) {
	defer func() {
		rollout.mu.Lock()
		if rollout.Status == RolloutStatus("running") {
			rollout.Status = RolloutStatus("completed")
			completedAt := time.Now()
			rollout.CompletedAt = &completedAt
		}
		rollout.mu.Unlock()
	}()
	
	rollout.mu.Lock()
	rollout.Status = RolloutStatus("running")
	rollout.mu.Unlock()
	
	// Start metrics collection
	go rm.collectMetrics(ctx, rollout)
	
	// Start health monitoring
	if rm.config.RequireHealthChecks {
		go rm.monitorHealth(ctx, rollout)
	}
	
	// Execute phases
	for i := range rollout.Phases {
		phase := &rollout.Phases[i]
		
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		if err := rm.executePhase(ctx, rollout, phase); err != nil {
			rm.handlePhaseError(rollout, phase, err)
			return
		}
		
		// Check if rollback was triggered
		rollout.mu.RLock()
		status := rollout.Status
		rollout.mu.RUnlock()
		
		if status == RolloutStatus("rolling_back") {
			return
		}
	}
}

// executePhase executes a single rollout phase
func (rm *RolloutManager) executePhase(ctx context.Context, rollout *ActiveRollout, phase *RolloutPhase) error {
	// Update phase status
	phase.Status = PhaseStatusActive
	startTime := time.Now()
	phase.StartedAt = &startTime
	
	// Update current phase
	rollout.mu.Lock()
	rollout.CurrentPhase = phase
	rollout.mu.Unlock()
	
	// Update traffic routing
	if err := rm.updateTrafficRouting(rollout, phase.TrafficPercent); err != nil {
		return fmt.Errorf("failed to update traffic routing: %w", err)
	}
	
	// Initialize phase metrics
	rm.initializePhaseMetrics(rollout, phase)
	
	// Monitor phase for specified duration
	phaseCtx, phaseCancel := context.WithTimeout(ctx, phase.Duration)
	defer phaseCancel()
	
	// Monitor phase conditions
	if err := rm.monitorPhaseConditions(phaseCtx, rollout, phase); err != nil {
		return fmt.Errorf("phase conditions not met: %w", err)
	}
	
	// Mark phase as completed
	phase.Status = PhaseStatusCompleted
	completedTime := time.Now()
	phase.CompletedAt = &completedTime
	
	return nil
}

// updateTrafficRouting updates traffic routing for the rollout
func (rm *RolloutManager) updateTrafficRouting(rollout *ActiveRollout, trafficPercent float64) error {
	route := &TrafficRoute{
		RolloutID:      rollout.ID,
		OldRuleID:      "current_rule", // In real implementation, get from rollout metadata
		NewRuleID:      "new_rule",     // In real implementation, get from rollout metadata
		TrafficPercent: trafficPercent,
		LastUpdated:    time.Now(),
	}
	
	return rm.trafficController.UpdateRoute(rollout.ID, route)
}

// monitorPhaseConditions monitors conditions during a phase
func (rm *RolloutManager) monitorPhaseConditions(ctx context.Context, rollout *ActiveRollout, phase *RolloutPhase) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check all phase conditions
			allMet := true
			for i := range phase.Conditions {
				condition := &phase.Conditions[i]
				if !rm.checkPhaseCondition(rollout, condition) {
					allMet = false
				}
			}
			
			// Check rollback thresholds
			if rm.config.AutoRollbackEnabled {
				if shouldRollback, reason := rm.shouldTriggerRollback(rollout); shouldRollback {
					rm.triggerRollback(rollout, reason)
					return fmt.Errorf("rollback triggered: %s", reason)
				}
			}
			
			// If all conditions met and minimum time elapsed, phase can complete
			if allMet && time.Since(*phase.StartedAt) >= time.Minute {
				return nil
			}
		}
	}
}

// checkPhaseCondition checks if a phase condition is met
func (rm *RolloutManager) checkPhaseCondition(rollout *ActiveRollout, condition *PhaseCondition) bool {
	// Get current metric value
	value := rm.getCurrentMetricValue(rollout, condition.Metric)
	
	// Check condition
	var met bool
	switch condition.Operator {
	case "lt":
		met = value < condition.Threshold
	case "lte":
		met = value <= condition.Threshold
	case "gt":
		met = value > condition.Threshold
	case "gte":
		met = value >= condition.Threshold
	case "eq":
		met = value == condition.Threshold
	default:
		met = false
	}
	
	condition.Met = met
	condition.LastChecked = time.Now()
	
	return met
}

// shouldTriggerRollback checks if rollback should be triggered
func (rm *RolloutManager) shouldTriggerRollback(rollout *ActiveRollout) (bool, string) {
	metrics := rollout.Metrics
	thresholds := rm.config.RollbackThresholds
	
	// Check error rate increase
	if metrics.ErrorRate > thresholds.MaxErrorRateIncrease {
		return true, fmt.Sprintf("error_rate_%.3f_exceeds_threshold_%.3f", 
			metrics.ErrorRate, thresholds.MaxErrorRateIncrease)
	}
	
	// Check detection rate decrease
	if metrics.DetectionRate > 0 {
		// Compare with baseline (would come from previous metrics)
		baselineDetectionRate := 0.95 // In real implementation, get from baseline
		decrease := (baselineDetectionRate - metrics.DetectionRate) / baselineDetectionRate
		if decrease > thresholds.MinDetectionRateDecrease {
			return true, fmt.Sprintf("detection_rate_decreased_by_%.3f_exceeds_threshold_%.3f", 
				decrease, thresholds.MinDetectionRateDecrease)
		}
	}
	
	// Check false positive increase
	baselineFPRate := 0.05 // In real implementation, get from baseline
	if metrics.FalsePositiveRate > 0 {
		increase := (metrics.FalsePositiveRate - baselineFPRate) / baselineFPRate
		if increase > thresholds.MaxFalsePositiveIncrease {
			return true, fmt.Sprintf("false_positive_rate_increased_by_%.3f_exceeds_threshold_%.3f", 
				increase, thresholds.MaxFalsePositiveIncrease)
		}
	}
	
	// Check user satisfaction decrease
	if metrics.UserSatisfaction > 0 {
		baselineSatisfaction := 0.85 // In real implementation, get from baseline
		decrease := (baselineSatisfaction - metrics.UserSatisfaction) / baselineSatisfaction
		if decrease > thresholds.UserSatisfactionDecrease {
			return true, fmt.Sprintf("user_satisfaction_decreased_by_%.3f_exceeds_threshold_%.3f", 
				decrease, thresholds.UserSatisfactionDecrease)
		}
	}
	
	return false, ""
}

// triggerRollback triggers an automatic rollback
func (rm *RolloutManager) triggerRollback(rollout *ActiveRollout, reason string) {
	rollout.mu.Lock()
	rollout.Status = RolloutStatus("rolling_back")
	rollout.mu.Unlock()
	
	// Create critical alert
	alert := Alert{
		ID:          generateAlertID(),
		RolloutID:   rollout.ID,
		Type:        "auto_rollback_triggered",
		Severity:    AlertSeverityCritical,
		Message:     fmt.Sprintf("Automatic rollback triggered: %s", reason),
		TriggeredAt: time.Now(),
		Metadata: map[string]interface{}{
			"reason": reason,
			"phase":  rollout.CurrentPhase.Name,
		},
	}
	
	rm.alertManager.TriggerAlert(alert)
	
	// Execute rollback
	go rm.executeRollback(context.Background(), rollout)
}

// executeRollback executes the rollback
func (rm *RolloutManager) executeRollback(ctx context.Context, rollout *ActiveRollout) {
	// Cancel the rollout context
	if rollout.cancelFunc != nil {
		rollout.cancelFunc()
	}
	
	// Revert traffic to 0% (back to old rule)
	rm.updateTrafficRouting(rollout, 0)
	
	// Update rollout status
	rollout.mu.Lock()
	rollout.Status = RolloutStatus("rolled_back")
	completedAt := time.Now()
	rollout.CompletedAt = &completedAt
	rollout.mu.Unlock()
	
	// Create rollback completion alert
	alert := Alert{
		ID:          generateAlertID(),
		RolloutID:   rollout.ID,
		Type:        "rollback_completed",
		Severity:    AlertSeverityInfo,
		Message:     "Rollback completed successfully",
		TriggeredAt: time.Now(),
	}
	
	rm.alertManager.TriggerAlert(alert)
}

// collectMetrics collects metrics during rollout
func (rm *RolloutManager) collectMetrics(ctx context.Context, rollout *ActiveRollout) {
	ticker := time.NewTicker(rm.config.MetricsCollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.updateRolloutMetrics(rollout)
		}
	}
}

// updateRolloutMetrics updates metrics for a rollout
func (rm *RolloutManager) updateRolloutMetrics(rollout *ActiveRollout) {
	// In a real implementation, this would collect actual metrics
	// For simulation, we'll generate realistic values
	
	rollout.mu.Lock()
	defer rollout.mu.Unlock()
	
	metrics := rollout.Metrics
	
	// Simulate metric collection
	metrics.TotalRequests += int64(100) // 100 requests in this interval
	if rollout.CurrentPhase != nil {
		newRulePercent := rollout.CurrentPhase.TrafficPercent / 100.0
		metrics.NewRuleRequests += int64(float64(100) * newRulePercent)
		metrics.OldRuleRequests += int64(float64(100) * (1.0 - newRulePercent))
	}
	
	// Simulate some metrics (in real implementation, get from monitoring system)
	metrics.ErrorRate = 0.01 + (time.Now().UnixNano()%100)/10000.0 // 0.01-0.02
	metrics.DetectionRate = 0.94 + (time.Now().UnixNano()%50)/1000.0 // 0.94-0.99
	metrics.FalsePositiveRate = 0.03 + (time.Now().UnixNano()%30)/1000.0 // 0.03-0.06
	metrics.UserSatisfaction = 0.80 + (time.Now().UnixNano()%200)/1000.0 // 0.80-1.00
	metrics.AvgResponseTime = time.Duration(10+time.Now().UnixNano()%20) * time.Millisecond // 10-30ms
	
	metrics.LastUpdated = time.Now()
}

// monitorHealth monitors health checks during rollout
func (rm *RolloutManager) monitorHealth(ctx context.Context, rollout *ActiveRollout) {
	ticker := time.NewTicker(rm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.runHealthChecks(rollout)
		}
	}
}

// runHealthChecks runs health checks for a rollout
func (rm *RolloutManager) runHealthChecks(rollout *ActiveRollout) {
	healthChecks := []HealthCheck{
		{
			ID:   "response_time_check",
			Name: "Response Time Check",
			Type: "metric",
		},
		{
			ID:   "error_rate_check",
			Name: "Error Rate Check",
			Type: "metric",
		},
		{
			ID:   "detection_rate_check",
			Name: "Detection Rate Check",
			Type: "metric",
		},
	}
	
	for i := range healthChecks {
		check := &healthChecks[i]
		rm.runSingleHealthCheck(rollout, check)
	}
	
	rollout.mu.Lock()
	rollout.HealthChecks = healthChecks
	rollout.mu.Unlock()
}

// runSingleHealthCheck runs a single health check
func (rm *RolloutManager) runSingleHealthCheck(rollout *ActiveRollout, check *HealthCheck) {
	startTime := time.Now()
	defer func() {
		check.Duration = time.Since(startTime)
		check.LastRun = time.Now()
	}()
	
	// Simulate health check logic
	switch check.ID {
	case "response_time_check":
		if rollout.Metrics.AvgResponseTime < 50*time.Millisecond {
			check.Status = HealthCheckStatusPass
			check.Message = "Response time within acceptable range"
		} else {
			check.Status = HealthCheckStatusFail
			check.Message = "Response time too high"
		}
		
	case "error_rate_check":
		if rollout.Metrics.ErrorRate < 0.05 {
			check.Status = HealthCheckStatusPass
			check.Message = "Error rate within acceptable range"
		} else {
			check.Status = HealthCheckStatusFail
			check.Message = "Error rate too high"
		}
		
	case "detection_rate_check":
		if rollout.Metrics.DetectionRate > 0.90 {
			check.Status = HealthCheckStatusPass
			check.Message = "Detection rate within acceptable range"
		} else {
			check.Status = HealthCheckStatusFail
			check.Message = "Detection rate too low"
		}
		
	default:
		check.Status = HealthCheckStatusUnknown
		check.Message = "Unknown health check type"
	}
}

// Helper methods for creating rollout phases
func (rm *RolloutManager) createRolloutPhases(strategy RolloutStrategy) []RolloutPhase {
	switch strategy {
	case RolloutImmediate:
		return []RolloutPhase{
			{
				Name:           "immediate_100",
				TrafficPercent: 100,
				Duration:       5 * time.Minute,
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
		}
		
	case RolloutGradual:
		return []RolloutPhase{
			{
				Name:           "gradual_5",
				TrafficPercent: 5,
				Duration:       rm.config.PhaseIntervals["5%"],
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
			{
				Name:           "gradual_10",
				TrafficPercent: 10,
				Duration:       rm.config.PhaseIntervals["10%"],
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
			{
				Name:           "gradual_25",
				TrafficPercent: 25,
				Duration:       rm.config.PhaseIntervals["25%"],
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
			{
				Name:           "gradual_50",
				TrafficPercent: 50,
				Duration:       rm.config.PhaseIntervals["50%"],
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
			{
				Name:           "gradual_100",
				TrafficPercent: 100,
				Duration:       rm.config.PhaseIntervals["100%"],
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
		}
		
	case RolloutCanary:
		return []RolloutPhase{
			{
				Name:           "canary_5",
				TrafficPercent: 5,
				Duration:       30 * time.Minute,
				Status:         PhaseStatusPending,
				Conditions:     rm.getStrictConditions(),
			},
			{
				Name:           "canary_graduated_25",
				TrafficPercent: 25,
				Duration:       20 * time.Minute,
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
			{
				Name:           "canary_graduated_100",
				TrafficPercent: 100,
				Duration:       15 * time.Minute,
				Status:         PhaseStatusPending,
				Conditions:     rm.getStandardConditions(),
			},
		}
		
	default:
		return rm.createRolloutPhases(RolloutGradual)
	}
}

// getStandardConditions returns standard phase conditions
func (rm *RolloutManager) getStandardConditions() []PhaseCondition {
	return []PhaseCondition{
		{
			Type:        "metric",
			Metric:      "error_rate",
			Operator:    "lt",
			Threshold:   0.05,
			Duration:    2 * time.Minute,
			Description: "Error rate must be below 5%",
		},
		{
			Type:        "metric",
			Metric:      "detection_rate",
			Operator:    "gte",
			Threshold:   0.90,
			Duration:    2 * time.Minute,
			Description: "Detection rate must be at least 90%",
		},
	}
}

// getStrictConditions returns strict phase conditions for canary
func (rm *RolloutManager) getStrictConditions() []PhaseCondition {
	return []PhaseCondition{
		{
			Type:        "metric",
			Metric:      "error_rate",
			Operator:    "lt",
			Threshold:   0.02,
			Duration:    5 * time.Minute,
			Description: "Error rate must be below 2% for canary",
		},
		{
			Type:        "metric",
			Metric:      "detection_rate",
			Operator:    "gte",
			Threshold:   0.93,
			Duration:    5 * time.Minute,
			Description: "Detection rate must be at least 93% for canary",
		},
		{
			Type:        "metric",
			Metric:      "false_positive_rate",
			Operator:    "lt",
			Threshold:   0.04,
			Duration:    5 * time.Minute,
			Description: "False positive rate must be below 4% for canary",
		},
	}
}

// Utility methods and supporting components

func (rm *RolloutManager) initializePhaseMetrics(rollout *ActiveRollout, phase *RolloutPhase) {
	phaseMetrics := &PhaseMetrics{
		PhaseName: phase.Name,
		StartTime: time.Now(),
	}
	
	rollout.Metrics.PhaseMetrics[phase.Name] = phaseMetrics
}

func (rm *RolloutManager) getCurrentMetricValue(rollout *ActiveRollout, metric string) float64 {
	switch metric {
	case "error_rate":
		return rollout.Metrics.ErrorRate
	case "detection_rate":
		return rollout.Metrics.DetectionRate
	case "false_positive_rate":
		return rollout.Metrics.FalsePositiveRate
	case "user_satisfaction":
		return rollout.Metrics.UserSatisfaction
	default:
		return 0.0
	}
}

func (rm *RolloutManager) handlePhaseError(rollout *ActiveRollout, phase *RolloutPhase, err error) {
	phase.Status = PhaseStatusFailed
	
	rollout.mu.Lock()
	rollout.Status = RolloutStatus("failed")
	rollout.mu.Unlock()
	
	// Trigger alert
	alert := Alert{
		ID:          generateAlertID(),
		RolloutID:   rollout.ID,
		Type:        "phase_failed",
		Severity:    AlertSeverityCritical,
		Message:     fmt.Sprintf("Phase %s failed: %s", phase.Name, err.Error()),
		TriggeredAt: time.Now(),
		Metadata: map[string]interface{}{
			"phase": phase.Name,
			"error": err.Error(),
		},
	}
	
	rm.alertManager.TriggerAlert(alert)
	
	// Trigger rollback
	rm.triggerRollback(rollout, fmt.Sprintf("phase_failed_%s", phase.Name))
}

// Supporting component implementations

func NewTrafficController() *TrafficController {
	return &TrafficController{
		activeRoutes: make(map[string]*TrafficRoute),
	}
}

func (tc *TrafficController) UpdateRoute(rolloutID string, route *TrafficRoute) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	tc.activeRoutes[rolloutID] = route
	
	// In real implementation, this would update actual traffic routing
	fmt.Printf("Updated traffic routing for rollout %s to %.1f%%\n", rolloutID, route.TrafficPercent)
	
	return nil
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		rolloutMetrics: make(map[string]*RolloutMetrics),
	}
}

func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts: make([]Alert, 0),
	}
}

func (am *AlertManager) TriggerAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.alerts = append(am.alerts, alert)
	
	// In real implementation, this would send notifications
	fmt.Printf("ALERT [%s]: %s\n", alert.Severity, alert.Message)
}

// Supporting types

type StartRolloutRequest struct {
	DeploymentID string                 `json:"deployment_id"`
	PromotionID  string                 `json:"promotion_id"`
	Strategy     RolloutStrategy        `json:"strategy"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Helper functions

func generateRolloutID() string {
	return fmt.Sprintf("rollout_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateRollbackID() string {
	return fmt.Sprintf("rollback_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateAlertID() string {
	return fmt.Sprintf("alert_%d_%s", time.Now().Unix(), generateRandomString(6))
}