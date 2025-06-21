package optimizer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ExperimentManager manages A/B testing experiments for rules
type ExperimentManager struct {
	experiments      map[string]*Experiment
	trafficRouter    *TrafficRouter
	statisticsEngine *StatisticsEngine
	safetyMonitor    *SafetyMonitor
	mu               sync.RWMutex
	config           *ExperimentManagerConfig
}

// ExperimentManagerConfig configures the experiment manager
type ExperimentManagerConfig struct {
	MaxConcurrentExperiments  int           `json:"max_concurrent_experiments"`
	DefaultTrafficAllocation  float64       `json:"default_traffic_allocation"`
	SafetyCheckInterval       time.Duration `json:"safety_check_interval"`
	MetricsCollectionInterval time.Duration `json:"metrics_collection_interval"`
	AutoStopEnabled           bool          `json:"auto_stop_enabled"`
	MinimumTrafficPerVariant  float64       `json:"minimum_traffic_per_variant"`
	MaximumExperimentDuration time.Duration `json:"maximum_experiment_duration"`
}

// ExperimentPhase represents different phases of an experiment
type ExperimentPhase string

const (
	PhaseRampUp     ExperimentPhase = "ramp_up"
	PhaseFullTraffic ExperimentPhase = "full_traffic"
	PhaseRampDown   ExperimentPhase = "ramp_down"
	PhaseAnalysis   ExperimentPhase = "analysis"
)

// ExperimentEvent represents events that occur during an experiment
type ExperimentEvent struct {
	ID          string                 `json:"id"`
	ExperimentID string                `json:"experiment_id"`
	Type        string                 `json:"type"`
	Phase       ExperimentPhase        `json:"phase"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
}

// NewExperimentManager creates a new experiment manager
func NewExperimentManager(config *ExperimentManagerConfig) *ExperimentManager {
	if config == nil {
		config = GetDefaultExperimentManagerConfig()
	}
	
	manager := &ExperimentManager{
		experiments:      make(map[string]*Experiment),
		trafficRouter:    NewTrafficRouter(),
		statisticsEngine: NewStatisticsEngine(),
		safetyMonitor:    NewSafetyMonitor(),
		config:           config,
	}
	
	return manager
}

// GetDefaultExperimentManagerConfig returns default configuration
func GetDefaultExperimentManagerConfig() *ExperimentManagerConfig {
	return &ExperimentManagerConfig{
		MaxConcurrentExperiments:  5,
		DefaultTrafficAllocation:  0.1,
		SafetyCheckInterval:       1 * time.Minute,
		MetricsCollectionInterval: 30 * time.Second,
		AutoStopEnabled:           true,
		MinimumTrafficPerVariant:  0.05,
		MaximumExperimentDuration: 48 * time.Hour,
	}
}

// CreateExperiment creates and starts a new A/B test experiment
func (em *ExperimentManager) CreateExperiment(ctx context.Context, config ExperimentConfig) (*Experiment, error) {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	// Check if we can create more experiments
	if len(em.experiments) >= em.config.MaxConcurrentExperiments {
		return nil, fmt.Errorf("maximum concurrent experiments reached: %d", em.config.MaxConcurrentExperiments)
	}
	
	// Validate experiment configuration
	if err := em.validateExperimentConfig(config); err != nil {
		return nil, fmt.Errorf("invalid experiment configuration: %w", err)
	}
	
	// Create experiment
	experiment := &Experiment{
		ID:          generateExperimentID(),
		Name:        config.Name,
		Status:      ExperimentStatusPending,
		StartTime:   time.Now(),
		Objectives:  config.Objectives,
		Control:     config.Control,
		Variants:    config.Variants,
		Traffic:     config.Traffic,
		Config:      config,
		Results:     NewExperimentResults(),
	}
	
	// Initialize experiment metrics
	em.initializeExperimentMetrics(experiment)
	
	// Register with traffic router
	if err := em.trafficRouter.RegisterExperiment(experiment); err != nil {
		return nil, fmt.Errorf("failed to register experiment with traffic router: %w", err)
	}
	
	// Start safety monitoring
	em.safetyMonitor.StartMonitoring(experiment)
	
	// Store experiment
	em.experiments[experiment.ID] = experiment
	
	// Log experiment creation event
	em.logExperimentEvent(experiment, "experiment_created", PhaseRampUp, map[string]interface{}{
		"control_rule_id": experiment.Control.ID,
		"variant_count":   len(experiment.Variants),
		"traffic_split":   experiment.Traffic,
	}, "info", "Experiment created and ready to start")
	
	return experiment, nil
}

// StartExperiment begins traffic allocation for an experiment
func (em *ExperimentManager) StartExperiment(ctx context.Context, experimentID string) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	experiment, exists := em.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	if experiment.Status != ExperimentStatusPending {
		return fmt.Errorf("experiment is not in pending status: %s", experiment.Status)
	}
	
	// Start with ramp-up phase
	experiment.Status = ExperimentStatusRunning
	experiment.StartTime = time.Now()
	
	// Begin traffic allocation
	if err := em.trafficRouter.StartTrafficAllocation(experimentID); err != nil {
		return fmt.Errorf("failed to start traffic allocation: %w", err)
	}
	
	// Start metrics collection
	go em.startMetricsCollection(experiment)
	
	// Start safety monitoring
	go em.startSafetyMonitoring(experiment)
	
	// Log experiment start event
	em.logExperimentEvent(experiment, "experiment_started", PhaseRampUp, map[string]interface{}{
		"start_time": experiment.StartTime,
	}, "info", "Experiment started with traffic allocation")
	
	return nil
}

// StopExperiment stops an experiment and collects final results
func (em *ExperimentManager) StopExperiment(ctx context.Context, experimentID string, reason string) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	experiment, exists := em.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	if experiment.Status != ExperimentStatusRunning {
		return fmt.Errorf("experiment is not running: %s", experiment.Status)
	}
	
	// Stop traffic allocation
	if err := em.trafficRouter.StopTrafficAllocation(experimentID); err != nil {
		return fmt.Errorf("failed to stop traffic allocation: %w", err)
	}
	
	// Stop safety monitoring
	em.safetyMonitor.StopMonitoring(experimentID)
	
	// Set final status
	experiment.Status = ExperimentStatusCompleted
	endTime := time.Now()
	experiment.EndTime = &endTime
	experiment.StopReason = reason
	
	// Perform final analysis
	if err := em.performFinalAnalysis(experiment); err != nil {
		return fmt.Errorf("failed to perform final analysis: %w", err)
	}
	
	// Log experiment stop event
	em.logExperimentEvent(experiment, "experiment_stopped", PhaseAnalysis, map[string]interface{}{
		"end_time":    experiment.EndTime,
		"stop_reason": reason,
		"duration":    experiment.GetDuration(),
	}, "info", fmt.Sprintf("Experiment stopped: %s", reason))
	
	return nil
}

// GetExperiment retrieves an experiment by ID
func (em *ExperimentManager) GetExperiment(experimentID string) (*Experiment, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	experiment, exists := em.experiments[experimentID]
	if !exists {
		return nil, fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	return experiment, nil
}

// ListActiveExperiments returns all currently running experiments
func (em *ExperimentManager) ListActiveExperiments() []*Experiment {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	activeExperiments := make([]*Experiment, 0)
	for _, experiment := range em.experiments {
		if experiment.Status == ExperimentStatusRunning {
			activeExperiments = append(activeExperiments, experiment)
		}
	}
	
	return activeExperiments
}

// RouteTraffic routes a request to the appropriate rule variant
func (em *ExperimentManager) RouteTraffic(ctx context.Context, requestID string, userID string) (*Rule, string, error) {
	// Use traffic router to determine which rule to use
	ruleAssignment, err := em.trafficRouter.RouteRequest(requestID, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to route traffic: %w", err)
	}
	
	// Get the experiment and rule
	experiment, exists := em.experiments[ruleAssignment.ExperimentID]
	if !exists {
		return nil, "", fmt.Errorf("experiment not found: %s", ruleAssignment.ExperimentID)
	}
	
	var rule *Rule
	if ruleAssignment.VariantID == "control" {
		rule = &experiment.Control
	} else {
		for _, variant := range experiment.Variants {
			if variant.ID == ruleAssignment.VariantID {
				rule = &variant
				break
			}
		}
	}
	
	if rule == nil {
		return nil, "", fmt.Errorf("rule not found for variant: %s", ruleAssignment.VariantID)
	}
	
	return rule, ruleAssignment.VariantID, nil
}

// RecordResult records the result of applying a rule in an experiment
func (em *ExperimentManager) RecordResult(ctx context.Context, experimentID, variantID string, result ExperimentResult) error {
	em.mu.RLock()
	experiment, exists := em.experiments[experimentID]
	em.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	// Get or create variant metrics
	metrics := experiment.Results.GetVariantMetrics(variantID)
	if metrics == nil {
		metrics = &VariantMetrics{
			VariantID:    variantID,
			FirstSample:  time.Now(),
		}
		experiment.Results.SetVariantMetrics(variantID, metrics)
	}
	
	// Update metrics
	metrics.UpdateMetrics(result.Detection, result.FalsePositive, result.ResponseTime)
	
	// Update user satisfaction if provided
	if result.UserSatisfaction > 0 {
		if metrics.SampleSize == 1 {
			metrics.UserSatisfaction = result.UserSatisfaction
		} else {
			// Rolling average
			metrics.UserSatisfaction = (metrics.UserSatisfaction*float64(metrics.SampleSize-1) + result.UserSatisfaction) / float64(metrics.SampleSize)
		}
	}
	
	// Check for early stopping conditions
	if em.config.AutoStopEnabled {
		if shouldStop, reason := em.checkEarlyStoppingConditions(experiment); shouldStop {
			go func() {
				ctx := context.Background()
				em.StopExperiment(ctx, experimentID, reason)
			}()
		}
	}
	
	return nil
}

// validateExperimentConfig validates experiment configuration
func (em *ExperimentManager) validateExperimentConfig(config ExperimentConfig) error {
	// Check traffic allocation
	totalTraffic := config.Traffic.Control
	for _, variantTraffic := range config.Traffic.Variants {
		totalTraffic += variantTraffic
	}
	
	if totalTraffic < 0.99 || totalTraffic > 1.01 {
		return fmt.Errorf("traffic allocation must sum to 1.0, got %.3f", totalTraffic)
	}
	
	// Check minimum traffic per variant
	if config.Traffic.Control < em.config.MinimumTrafficPerVariant {
		return fmt.Errorf("control traffic %.3f below minimum %.3f", 
			config.Traffic.Control, em.config.MinimumTrafficPerVariant)
	}
	
	for variantID, traffic := range config.Traffic.Variants {
		if traffic < em.config.MinimumTrafficPerVariant {
			return fmt.Errorf("variant %s traffic %.3f below minimum %.3f", 
				variantID, traffic, em.config.MinimumTrafficPerVariant)
		}
	}
	
	// Check experiment duration
	if config.Duration > em.config.MaximumExperimentDuration {
		return fmt.Errorf("experiment duration %v exceeds maximum %v", 
			config.Duration, em.config.MaximumExperimentDuration)
	}
	
	// Validate objectives
	if len(config.Objectives) == 0 {
		return fmt.Errorf("experiment must have at least one objective")
	}
	
	return nil
}

// initializeExperimentMetrics initializes metrics tracking for an experiment
func (em *ExperimentManager) initializeExperimentMetrics(experiment *Experiment) {
	// Initialize control metrics
	controlMetrics := &VariantMetrics{
		VariantID:   "control",
		FirstSample: time.Now(),
	}
	experiment.Results.SetVariantMetrics("control", controlMetrics)
	
	// Initialize variant metrics
	for _, variant := range experiment.Variants {
		variantMetrics := &VariantMetrics{
			VariantID:   variant.ID,
			FirstSample: time.Now(),
		}
		experiment.Results.SetVariantMetrics(variant.ID, variantMetrics)
	}
}

// startMetricsCollection starts collecting metrics for an experiment
func (em *ExperimentManager) startMetricsCollection(experiment *Experiment) {
	ticker := time.NewTicker(em.config.MetricsCollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if experiment.Status != ExperimentStatusRunning {
				return
			}
			
			// Collect and update metrics
			em.collectExperimentMetrics(experiment)
			
		case <-time.After(em.config.MaximumExperimentDuration):
			// Auto-stop experiment if it runs too long
			ctx := context.Background()
			em.StopExperiment(ctx, experiment.ID, "maximum_duration_reached")
			return
		}
	}
}

// startSafetyMonitoring starts safety monitoring for an experiment
func (em *ExperimentManager) startSafetyMonitoring(experiment *Experiment) {
	ticker := time.NewTicker(em.config.SafetyCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if experiment.Status != ExperimentStatusRunning {
				return
			}
			
			// Check safety conditions
			if violation := em.safetyMonitor.CheckSafetyViolation(experiment); violation != nil {
				em.logExperimentEvent(experiment, "safety_violation", PhaseFullTraffic, map[string]interface{}{
					"violation_type": violation.Type,
					"severity":       violation.Severity,
				}, "critical", fmt.Sprintf("Safety violation detected: %s", violation.Message))
				
				// Auto-stop experiment if critical safety violation
				if violation.Severity == "critical" {
					ctx := context.Background()
					em.StopExperiment(ctx, experiment.ID, fmt.Sprintf("safety_violation_%s", violation.Type))
					return
				}
			}
		}
	}
}

// collectExperimentMetrics collects metrics for an experiment
func (em *ExperimentManager) collectExperimentMetrics(experiment *Experiment) {
	// This would integrate with the metrics collection system
	// to gather performance data for each variant
	
	// Update experiment results with current statistics
	em.statisticsEngine.UpdateExperimentStatistics(experiment)
}

// checkEarlyStoppingConditions checks if experiment should be stopped early
func (em *ExperimentManager) checkEarlyStoppingConditions(experiment *Experiment) (bool, string) {
	if !experiment.Config.EarlyStoppingEnabled {
		return false, ""
	}
	
	// Check minimum sample size
	controlMetrics := experiment.Results.GetVariantMetrics("control")
	if controlMetrics == nil || controlMetrics.SampleSize < experiment.Config.MinSampleSize {
		return false, ""
	}
	
	// Check if any variant has significant improvement
	for variantID, variant := range experiment.Results.VariantMetrics {
		if variantID == "control" {
			continue
		}
		
		if variant.SampleSize < experiment.Config.MinSampleSize {
			continue
		}
		
		// Calculate improvement over control
		improvement := (variant.DetectionRate - controlMetrics.DetectionRate) / controlMetrics.DetectionRate
		
		if improvement >= experiment.Config.MinDetectionImprovement {
			// Check if improvement is statistically significant
			if significance := em.statisticsEngine.CalculateSignificance(controlMetrics, variant); significance != nil {
				if significance.Significant && significance.PValue < 0.05 {
					return true, fmt.Sprintf("significant_improvement_variant_%s", variantID)
				}
			}
		}
		
		// Check for degradation beyond tolerance
		degradation := (controlMetrics.DetectionRate - variant.DetectionRate) / controlMetrics.DetectionRate
		if degradation > experiment.Config.MaxDegradationTolerance {
			return true, fmt.Sprintf("degradation_beyond_tolerance_variant_%s", variantID)
		}
	}
	
	return false, ""
}

// performFinalAnalysis performs final statistical analysis of experiment results
func (em *ExperimentManager) performFinalAnalysis(experiment *Experiment) error {
	// Calculate final statistics for all variants
	controlMetrics := experiment.Results.GetVariantMetrics("control")
	if controlMetrics == nil {
		return fmt.Errorf("no control metrics found")
	}
	
	bestVariant := controlMetrics
	bestVariantID := "control"
	
	// Compare each variant to control
	for variantID, variantMetrics := range experiment.Results.VariantMetrics {
		if variantID == "control" {
			continue
		}
		
		// Calculate statistical significance
		significance := em.statisticsEngine.CalculateSignificance(controlMetrics, variantMetrics)
		
		// Store significance results
		experiment.Results.Significance = significance
		
		// Update fitness score
		objectiveSet, _ := NewObjectiveSet(experiment.Objectives)
		if objectiveSet != nil {
			metrics := map[string]float64{
				"detection_rate":       variantMetrics.DetectionRate,
				"false_positive_rate":  variantMetrics.FalsePositiveRate,
				"response_time_ms":     variantMetrics.AvgResponseTime.Seconds() * 1000,
				"user_satisfaction":    variantMetrics.UserSatisfaction,
			}
			variantMetrics.FitnessScore = objectiveSet.CalculateScore(metrics)
		}
		
		// Track best performing variant
		if variantMetrics.FitnessScore > bestVariant.FitnessScore {
			bestVariant = variantMetrics
			bestVariantID = variantID
		}
	}
	
	// Set winner
	experiment.Results.Winner = bestVariant
	
	// Generate summary
	if bestVariantID == "control" {
		experiment.Results.Summary = "Control rule performed best - no improvement found"
		experiment.Results.RecommendedAction = "keep_current_rule"
	} else {
		improvement := (bestVariant.DetectionRate - controlMetrics.DetectionRate) / controlMetrics.DetectionRate
		experiment.Results.Summary = fmt.Sprintf("Variant %s performed best with %.2f%% improvement in detection rate", 
			bestVariantID, improvement*100)
		
		if experiment.Results.Significance != nil && experiment.Results.Significance.Significant {
			experiment.Results.RecommendedAction = "deploy_winning_variant"
		} else {
			experiment.Results.RecommendedAction = "extend_experiment_for_significance"
		}
	}
	
	// Calculate confidence level
	if experiment.Results.Significance != nil {
		experiment.Results.ConfidenceLevel = 1.0 - experiment.Results.Significance.PValue
	}
	
	return nil
}

// logExperimentEvent logs an experiment event
func (em *ExperimentManager) logExperimentEvent(experiment *Experiment, eventType string, phase ExperimentPhase, data map[string]interface{}, severity, message string) {
	event := ExperimentEvent{
		ID:           fmt.Sprintf("event_%d", time.Now().UnixNano()),
		ExperimentID: experiment.ID,
		Type:         eventType,
		Phase:        phase,
		Timestamp:    time.Now(),
		Data:         data,
		Severity:     severity,
		Message:      message,
	}
	
	// In a real implementation, this would be sent to a logging/event system
	// For now, we'll just log to console
	fmt.Printf("[EXPERIMENT_EVENT] %s: %s\n", event.Type, event.Message)
}

// ExperimentResult represents the result of applying a rule
type ExperimentResult struct {
	Detection        bool          `json:"detection"`
	FalsePositive    bool          `json:"false_positive"`
	ResponseTime     time.Duration `json:"response_time"`
	UserSatisfaction float64       `json:"user_satisfaction"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// Cleanup removes completed experiments older than retention period
func (em *ExperimentManager) Cleanup(retentionPeriod time.Duration) error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	cutoff := time.Now().Add(-retentionPeriod)
	toDelete := make([]string, 0)
	
	for experimentID, experiment := range em.experiments {
		if experiment.Status == ExperimentStatusCompleted && 
		   experiment.EndTime != nil && 
		   experiment.EndTime.Before(cutoff) {
			toDelete = append(toDelete, experimentID)
		}
	}
	
	for _, experimentID := range toDelete {
		delete(em.experiments, experimentID)
	}
	
	return nil
}

// GetExperimentStatistics returns statistics for an experiment
func (em *ExperimentManager) GetExperimentStatistics(experimentID string) (map[string]interface{}, error) {
	experiment, err := em.GetExperiment(experimentID)
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]interface{})
	
	// Basic experiment info
	stats["experiment_id"] = experiment.ID
	stats["name"] = experiment.Name
	stats["status"] = experiment.Status
	stats["start_time"] = experiment.StartTime
	stats["duration"] = experiment.GetDuration()
	
	// Variant statistics
	variantStats := make(map[string]interface{})
	for variantID, metrics := range experiment.Results.VariantMetrics {
		variantStats[variantID] = map[string]interface{}{
			"sample_size":         metrics.SampleSize,
			"detection_rate":      metrics.DetectionRate,
			"false_positive_rate": metrics.FalsePositiveRate,
			"avg_response_time":   metrics.AvgResponseTime,
			"user_satisfaction":   metrics.UserSatisfaction,
			"fitness_score":       metrics.FitnessScore,
		}
	}
	stats["variants"] = variantStats
	
	// Results
	if experiment.Results.Winner != nil {
		stats["winner"] = experiment.Results.Winner.VariantID
	}
	if experiment.Results.Significance != nil {
		stats["significance"] = map[string]interface{}{
			"p_value":     experiment.Results.Significance.PValue,
			"significant": experiment.Results.Significance.Significant,
			"improvement": experiment.Results.Significance.Improvement,
		}
	}
	
	return stats, nil
}