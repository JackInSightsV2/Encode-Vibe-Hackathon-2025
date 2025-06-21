package optimizer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RollbackManager manages rollbacks of failed deployments
type RollbackManager struct {
	rollbackPlans    map[string]*RollbackPlan
	activeRollbacks  map[string]*ActiveRollback
	rolloutManager   *RolloutManager
	deploymentManager DeploymentManager
	ruleManager      RuleManager
	mu               sync.RWMutex
	config           *RollbackConfig
}

// RollbackConfig configures rollback behavior
type RollbackConfig struct {
	AutoRollbackEnabled       bool              `json:"auto_rollback_enabled"`
	RollbackTimeout          time.Duration     `json:"rollback_timeout"`
	ValidationTimeout        time.Duration     `json:"validation_timeout"`
	MaxRollbackAttempts      int               `json:"max_rollback_attempts"`
	RollbackStrategy         RollbackStrategy  `json:"rollback_strategy"`
	RequireApproval          bool              `json:"require_approval"`
	HealthCheckTimeout       time.Duration     `json:"health_check_timeout"`
	MetricsValidationPeriod  time.Duration     `json:"metrics_validation_period"`
	NotificationChannels     []string          `json:"notification_channels"`
}

// RollbackStrategy defines different rollback strategies
type RollbackStrategy string

const (
	RollbackStrategyImmediate RollbackStrategy = "immediate"
	RollbackStrategyGradual   RollbackStrategy = "gradual"
	RollbackStrategyValidated RollbackStrategy = "validated"
)

// RollbackPlan contains the plan for rolling back a deployment
type RollbackPlan struct {
	ID               string                 `json:"id"`
	DeploymentID     string                 `json:"deployment_id"`
	PromotionID      string                 `json:"promotion_id"`
	RollbackStrategy RollbackStrategy       `json:"rollback_strategy"`
	TargetVersion    string                 `json:"target_version"`
	PreviousRuleID   string                 `json:"previous_rule_id"`
	CreatedAt        time.Time              `json:"created_at"`
	CreatedBy        string                 `json:"created_by"`
	Status           RollbackPlanStatus     `json:"status"`
	Steps            []RollbackStep         `json:"steps"`
	Prerequisites    []RollbackPrerequisite `json:"prerequisites"`
	ValidationChecks []ValidationCheck      `json:"validation_checks"`
	EstimatedDuration time.Duration         `json:"estimated_duration"`
	RiskAssessment   *RiskAssessment        `json:"risk_assessment"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// RollbackPlanStatus represents the status of a rollback plan
type RollbackPlanStatus string

const (
	RollbackPlanStatusDraft     RollbackPlanStatus = "draft"
	RollbackPlanStatusReady     RollbackPlanStatus = "ready"
	RollbackPlanStatusExecuting RollbackPlanStatus = "executing"
	RollbackPlanStatusCompleted RollbackPlanStatus = "completed"
	RollbackPlanStatusFailed    RollbackPlanStatus = "failed"
	RollbackPlanStatusCancelled RollbackPlanStatus = "cancelled"
)

// ActiveRollback represents an ongoing rollback
type ActiveRollback struct {
	ID              string                 `json:"id"`
	PlanID          string                 `json:"plan_id"`
	DeploymentID    string                 `json:"deployment_id"`
	Status          RollbackStatus         `json:"status"`
	CurrentStep     int                    `json:"current_step"`
	Steps           []RollbackStepExecution `json:"steps"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	HealthChecks    []HealthCheckResult    `json:"health_checks"`
	Metrics         *RollbackMetrics       `json:"metrics"`
	Notifications   []RollbackNotification `json:"notifications"`
	Metadata        map[string]interface{} `json:"metadata"`
	
	// Internal state
	cancelFunc      context.CancelFunc     `json:"-"`
	mu              sync.RWMutex           `json:"-"`
}

// RollbackStatus represents the status of a rollback
type RollbackStatus string

const (
	RollbackStatusPending    RollbackStatus = "pending"
	RollbackStatusRunning    RollbackStatus = "running"
	RollbackStatusCompleted  RollbackStatus = "completed"
	RollbackStatusFailed     RollbackStatus = "failed"
	RollbackStatusCancelled  RollbackStatus = "cancelled"
	RollbackStatusValidating RollbackStatus = "validating"
)

// RollbackStep represents a step in the rollback plan
type RollbackStep struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            RollbackStepType       `json:"type"`
	Description     string                 `json:"description"`
	Order           int                    `json:"order"`
	Parameters      map[string]interface{} `json:"parameters"`
	Dependencies    []string               `json:"dependencies"`
	Timeout         time.Duration          `json:"timeout"`
	RetryPolicy     *RetryPolicy           `json:"retry_policy"`
	ValidationRule  string                 `json:"validation_rule"`
	CriticalStep    bool                   `json:"critical_step"`
}

// RollbackStepType defines different types of rollback steps
type RollbackStepType string

const (
	StepTypeTrafficRevert    RollbackStepType = "traffic_revert"
	StepTypeRuleRevert       RollbackStepType = "rule_revert"
	StepTypeHealthCheck      RollbackStepType = "health_check"
	StepTypeMetricsValidation RollbackStepType = "metrics_validation"
	StepTypeNotification     RollbackStepType = "notification"
	StepTypeCleanup          RollbackStepType = "cleanup"
	StepTypeRolloutStop      RollbackStepType = "rollout_stop"
)

// RollbackStepExecution represents the execution of a rollback step
type RollbackStepExecution struct {
	StepID      string                 `json:"step_id"`
	Status      StepExecutionStatus    `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Result      map[string]interface{} `json:"result"`
	Error       string                 `json:"error,omitempty"`
	Attempts    int                    `json:"attempts"`
	Output      string                 `json:"output"`
}

// StepExecutionStatus represents the status of step execution
type StepExecutionStatus string

const (
	StepStatusPending   StepExecutionStatus = "pending"
	StepStatusRunning   StepExecutionStatus = "running"
	StepStatusCompleted StepExecutionStatus = "completed"
	StepStatusFailed    StepExecutionStatus = "failed"
	StepStatusSkipped   StepExecutionStatus = "skipped"
)

// RollbackPrerequisite represents a prerequisite for rollback
type RollbackPrerequisite struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Condition   string    `json:"condition"`
	Required    bool      `json:"required"`
	CheckedAt   time.Time `json:"checked_at"`
	Met         bool      `json:"met"`
	Message     string    `json:"message"`
}

// ValidationCheck represents a validation check during rollback
type ValidationCheck struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Metric       string                 `json:"metric"`
	Operator     string                 `json:"operator"`
	Threshold    float64                `json:"threshold"`
	Duration     time.Duration          `json:"duration"`
	Passed       bool                   `json:"passed"`
	LastChecked  time.Time              `json:"last_checked"`
	Value        float64                `json:"value"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// RiskAssessment represents risk assessment for rollback
type RiskAssessment struct {
	OverallRisk     string               `json:"overall_risk"`
	RiskFactors     []RiskFactor         `json:"risk_factors"`
	Mitigations     []RiskMitigation     `json:"mitigations"`
	ApprovalNeeded  bool                 `json:"approval_needed"`
	AssessedAt      time.Time            `json:"assessed_at"`
	AssessedBy      string               `json:"assessed_by"`
}

// RiskFactor represents a risk factor
type RiskFactor struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Likelihood  string  `json:"likelihood"`
	Impact      string  `json:"impact"`
	Score       float64 `json:"score"`
}

// RiskMitigation represents a risk mitigation
type RiskMitigation struct {
	RiskFactorID string `json:"risk_factor_id"`
	Description  string `json:"description"`
	Action       string `json:"action"`
	Implemented  bool   `json:"implemented"`
}

// RetryPolicy defines retry behavior for rollback steps
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     string        `json:"backoff"` // linear, exponential
	MaxDelay    time.Duration `json:"max_delay"`
}

// HealthCheckResult represents health check result during rollback
type HealthCheckResult struct {
	CheckID     string                 `json:"check_id"`
	Name        string                 `json:"name"`
	Status      HealthCheckStatus      `json:"status"`
	ExecutedAt  time.Time              `json:"executed_at"`
	Duration    time.Duration          `json:"duration"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
}

// RollbackMetrics contains metrics for rollback execution
type RollbackMetrics struct {
	RollbackID          string            `json:"rollback_id"`
	TotalSteps          int               `json:"total_steps"`
	CompletedSteps      int               `json:"completed_steps"`
	FailedSteps         int               `json:"failed_steps"`
	SkippedSteps        int               `json:"skipped_steps"`
	TotalDuration       time.Duration     `json:"total_duration"`
	EstimatedRemaining  time.Duration     `json:"estimated_remaining"`
	SuccessRate         float64           `json:"success_rate"`
	AverageStepDuration time.Duration     `json:"average_step_duration"`
	HealthChecksPassed  int               `json:"health_checks_passed"`
	HealthChecksFailed  int               `json:"health_checks_failed"`
	LastUpdated         time.Time         `json:"last_updated"`
}

// RollbackNotification represents a notification sent during rollback
type RollbackNotification struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Channel     string                 `json:"channel"`
	Recipient   string                 `json:"recipient"`
	Message     string                 `json:"message"`
	SentAt      time.Time              `json:"sent_at"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(
	rolloutManager *RolloutManager,
	deploymentManager DeploymentManager,
	ruleManager RuleManager,
	config *RollbackConfig,
) *RollbackManager {
	if config == nil {
		config = GetDefaultRollbackConfig()
	}
	
	return &RollbackManager{
		rollbackPlans:     make(map[string]*RollbackPlan),
		activeRollbacks:   make(map[string]*ActiveRollback),
		rolloutManager:    rolloutManager,
		deploymentManager: deploymentManager,
		ruleManager:       ruleManager,
		config:            config,
	}
}

// GetDefaultRollbackConfig returns default rollback configuration
func GetDefaultRollbackConfig() *RollbackConfig {
	return &RollbackConfig{
		AutoRollbackEnabled:      true,
		RollbackTimeout:          30 * time.Minute,
		ValidationTimeout:        5 * time.Minute,
		MaxRollbackAttempts:      3,
		RollbackStrategy:         RollbackStrategyValidated,
		RequireApproval:          false,
		HealthCheckTimeout:       2 * time.Minute,
		MetricsValidationPeriod:  5 * time.Minute,
		NotificationChannels:     []string{"email", "slack"},
	}
}

// CreateRollbackPlan creates a rollback plan for a deployment
func (rm *RollbackManager) CreateRollbackPlan(deploymentID string) (*RollbackPlan, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Get deployment information
	deployment, err := rm.deploymentManager.GetDeployment(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	
	// Create rollback plan
	plan := &RollbackPlan{
		ID:               generateRollbackPlanID(),
		DeploymentID:     deploymentID,
		RollbackStrategy: rm.config.RollbackStrategy,
		CreatedAt:        time.Now(),
		CreatedBy:        "rollback_manager",
		Status:           RollbackPlanStatusDraft,
		EstimatedDuration: rm.estimateRollbackDuration(deployment),
		Metadata:         make(map[string]interface{}),
	}
	
	// Generate rollback steps
	plan.Steps = rm.generateRollbackSteps(deployment)
	
	// Generate prerequisites
	plan.Prerequisites = rm.generatePrerequisites(deployment)
	
	// Generate validation checks
	plan.ValidationChecks = rm.generateValidationChecks(deployment)
	
	// Perform risk assessment
	plan.RiskAssessment = rm.assessRollbackRisk(deployment)
	
	// Mark as ready if all prerequisites are met
	if rm.checkPrerequisites(plan) {
		plan.Status = RollbackPlanStatusReady
	}
	
	// Store plan
	rm.rollbackPlans[plan.ID] = plan
	
	return plan, nil
}

// ExecuteRollback executes a rollback plan
func (rm *RollbackManager) ExecuteRollback(ctx context.Context, rollbackID string) error {
	rm.mu.Lock()
	plan, exists := rm.rollbackPlans[rollbackID]
	if !exists {
		rm.mu.Unlock()
		return fmt.Errorf("rollback plan not found: %s", rollbackID)
	}
	
	// Check if plan is ready for execution
	if plan.Status != RollbackPlanStatusReady {
		rm.mu.Unlock()
		return fmt.Errorf("rollback plan is not ready for execution: %s", plan.Status)
	}
	
	// Create rollback context with timeout
	rollbackCtx, cancelFunc := context.WithTimeout(ctx, rm.config.RollbackTimeout)
	
	// Create active rollback
	activeRollback := &ActiveRollback{
		ID:           generateActiveRollbackID(),
		PlanID:       rollbackID,
		DeploymentID: plan.DeploymentID,
		Status:       RollbackStatusPending,
		CurrentStep:  0,
		Steps:        rm.initializeStepExecutions(plan.Steps),
		StartedAt:    time.Now(),
		Metrics:      rm.initializeRollbackMetrics(plan),
		Metadata:     make(map[string]interface{}),
		cancelFunc:   cancelFunc,
	}
	
	// Store active rollback
	rm.activeRollbacks[activeRollback.ID] = activeRollback
	plan.Status = RollbackPlanStatusExecuting
	rm.mu.Unlock()
	
	// Send start notification
	rm.sendNotification(activeRollback, "rollback_started", "Rollback execution started")
	
	// Execute rollback steps
	go rm.executeRollbackSteps(rollbackCtx, activeRollback)
	
	return nil
}

// ValidateRollback validates a rollback plan
func (rm *RollbackManager) ValidateRollback(rollbackID string) error {
	rm.mu.RLock()
	plan, exists := rm.rollbackPlans[rollbackID]
	rm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("rollback plan not found: %s", rollbackID)
	}
	
	// Check prerequisites
	if !rm.checkPrerequisites(plan) {
		return fmt.Errorf("rollback prerequisites not met")
	}
	
	// Validate rollback steps
	for _, step := range plan.Steps {
		if err := rm.validateRollbackStep(step); err != nil {
			return fmt.Errorf("step validation failed for %s: %w", step.Name, err)
		}
	}
	
	// Check risk assessment
	if plan.RiskAssessment.OverallRisk == "critical" && rm.config.RequireApproval {
		return fmt.Errorf("rollback requires approval due to critical risk level")
	}
	
	return nil
}

// executeRollbackSteps executes all rollback steps
func (rm *RollbackManager) executeRollbackSteps(ctx context.Context, rollback *ActiveRollback) {
	defer func() {
		rollback.mu.Lock()
		if rollback.Status == RollbackStatusRunning {
			rollback.Status = RollbackStatusCompleted
			completedAt := time.Now()
			rollback.CompletedAt = &completedAt
		}
		rollback.mu.Unlock()
		
		// Send completion notification
		rm.sendNotification(rollback, "rollback_completed", "Rollback execution completed")
	}()
	
	rollback.mu.Lock()
	rollback.Status = RollbackStatusRunning
	rollback.mu.Unlock()
	
	// Execute steps in order
	for i, stepExecution := range rollback.Steps {
		select {
		case <-ctx.Done():
			rm.handleRollbackCancellation(rollback, fmt.Errorf("rollback cancelled: %w", ctx.Err()))
			return
		default:
		}
		
		rollback.mu.Lock()
		rollback.CurrentStep = i
		rollback.mu.Unlock()
		
		if err := rm.executeRollbackStep(ctx, rollback, &stepExecution, i); err != nil {
			rm.handleRollbackError(rollback, i, err)
			return
		}
		
		// Update metrics
		rm.updateRollbackMetrics(rollback)
		
		// Run health checks if specified
		if stepExecution.StepID == "health_check" {
			if err := rm.runRollbackHealthChecks(rollback); err != nil {
				rm.handleRollbackError(rollback, i, fmt.Errorf("health check failed: %w", err))
				return
			}
		}
	}
}

// executeRollbackStep executes a single rollback step
func (rm *RollbackManager) executeRollbackStep(ctx context.Context, rollback *ActiveRollback, stepExecution *RollbackStepExecution, stepIndex int) error {
	stepExecution.Status = StepStatusRunning
	stepExecution.StartedAt = time.Now()
	stepExecution.Attempts++
	
	plan := rm.rollbackPlans[rollback.PlanID]
	step := plan.Steps[stepIndex]
	
	// Create step context with timeout
	stepCtx, stepCancel := context.WithTimeout(ctx, step.Timeout)
	defer stepCancel()
	
	var err error
	switch step.Type {
	case StepTypeTrafficRevert:
		err = rm.executeTrafficRevert(stepCtx, rollback, step)
	case StepTypeRuleRevert:
		err = rm.executeRuleRevert(stepCtx, rollback, step)
	case StepTypeHealthCheck:
		err = rm.executeHealthCheck(stepCtx, rollback, step)
	case StepTypeMetricsValidation:
		err = rm.executeMetricsValidation(stepCtx, rollback, step)
	case StepTypeNotification:
		err = rm.executeNotification(stepCtx, rollback, step)
	case StepTypeCleanup:
		err = rm.executeCleanup(stepCtx, rollback, step)
	case StepTypeRolloutStop:
		err = rm.executeRolloutStop(stepCtx, rollback, step)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Type)
	}
	
	stepExecution.CompletedAt = &[]time.Time{time.Now()}[0]
	stepExecution.Duration = time.Since(stepExecution.StartedAt)
	
	if err != nil {
		stepExecution.Status = StepStatusFailed
		stepExecution.Error = err.Error()
		
		// Retry if policy allows
		if step.RetryPolicy != nil && stepExecution.Attempts < step.RetryPolicy.MaxAttempts {
			time.Sleep(step.RetryPolicy.Delay)
			return rm.executeRollbackStep(ctx, rollback, stepExecution, stepIndex)
		}
		
		return err
	}
	
	stepExecution.Status = StepStatusCompleted
	return nil
}

// executeTrafficRevert reverts traffic routing
func (rm *RollbackManager) executeTrafficRevert(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	// Revert traffic to 0% for new rule (back to old rule)
	if rm.rolloutManager != nil {
		return rm.rolloutManager.updateTrafficRouting(&ActiveRollout{
			ID:           rollback.DeploymentID,
			DeploymentID: rollback.DeploymentID,
		}, 0)
	}
	
	fmt.Printf("Reverting traffic for deployment %s\n", rollback.DeploymentID)
	return nil
}

// executeRuleRevert reverts rule changes
func (rm *RollbackManager) executeRuleRevert(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	plan := rm.rollbackPlans[rollback.PlanID]
	
	if plan.PreviousRuleID != "" {
		// In a real implementation, this would revert to the previous rule
		fmt.Printf("Reverting to previous rule %s for deployment %s\n", plan.PreviousRuleID, rollback.DeploymentID)
	}
	
	return nil
}

// executeHealthCheck runs health checks
func (rm *RollbackManager) executeHealthCheck(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	return rm.runRollbackHealthChecks(rollback)
}

// executeMetricsValidation validates metrics
func (rm *RollbackManager) executeMetricsValidation(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	plan := rm.rollbackPlans[rollback.PlanID]
	
	for i := range plan.ValidationChecks {
		check := &plan.ValidationChecks[i]
		
		// Simulate metrics validation
		check.LastChecked = time.Now()
		check.Value = 0.01 // Simulate good metrics
		check.Passed = check.Value <= check.Threshold
		
		if !check.Passed {
			return fmt.Errorf("validation check %s failed: value %.3f exceeds threshold %.3f", 
				check.Name, check.Value, check.Threshold)
		}
	}
	
	return nil
}

// executeNotification sends notifications
func (rm *RollbackManager) executeNotification(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	message := fmt.Sprintf("Rollback step %s completed for deployment %s", step.Name, rollback.DeploymentID)
	rm.sendNotification(rollback, "step_completed", message)
	return nil
}

// executeCleanup performs cleanup tasks
func (rm *RollbackManager) executeCleanup(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	// Cleanup any temporary resources, logs, etc.
	fmt.Printf("Performing cleanup for deployment %s\n", rollback.DeploymentID)
	return nil
}

// executeRolloutStop stops any active rollouts
func (rm *RollbackManager) executeRolloutStop(ctx context.Context, rollback *ActiveRollback, step RollbackStep) error {
	// Stop any active rollouts for this deployment
	fmt.Printf("Stopping rollouts for deployment %s\n", rollback.DeploymentID)
	return nil
}

// generateRollbackSteps generates rollback steps for a deployment
func (rm *RollbackManager) generateRollbackSteps(deployment *Deployment) []RollbackStep {
	steps := []RollbackStep{
		{
			ID:          "stop_rollout",
			Name:        "Stop Active Rollout",
			Type:        StepTypeRolloutStop,
			Description: "Stop any active rollout for this deployment",
			Order:       1,
			Timeout:     2 * time.Minute,
			RetryPolicy: &RetryPolicy{MaxAttempts: 3, Delay: 10 * time.Second, Backoff: "linear"},
			CriticalStep: true,
		},
		{
			ID:          "revert_traffic",
			Name:        "Revert Traffic Routing",
			Type:        StepTypeTrafficRevert,
			Description: "Revert traffic routing to previous configuration",
			Order:       2,
			Timeout:     5 * time.Minute,
			RetryPolicy: &RetryPolicy{MaxAttempts: 3, Delay: 10 * time.Second, Backoff: "exponential"},
			CriticalStep: true,
		},
		{
			ID:          "revert_rules",
			Name:        "Revert Rule Changes",
			Type:        StepTypeRuleRevert,
			Description: "Revert to previous rule configuration",
			Order:       3,
			Timeout:     3 * time.Minute,
			RetryPolicy: &RetryPolicy{MaxAttempts: 2, Delay: 5 * time.Second, Backoff: "linear"},
			CriticalStep: true,
		},
		{
			ID:          "health_check",
			Name:        "Run Health Checks",
			Type:        StepTypeHealthCheck,
			Description: "Validate system health after rollback",
			Order:       4,
			Timeout:     rm.config.HealthCheckTimeout,
			RetryPolicy: &RetryPolicy{MaxAttempts: 3, Delay: 30 * time.Second, Backoff: "linear"},
			CriticalStep: false,
		},
		{
			ID:          "validate_metrics",
			Name:        "Validate Metrics",
			Type:        StepTypeMetricsValidation,
			Description: "Validate system metrics after rollback",
			Order:       5,
			Timeout:     rm.config.MetricsValidationPeriod,
			RetryPolicy: &RetryPolicy{MaxAttempts: 2, Delay: 1 * time.Minute, Backoff: "linear"},
			CriticalStep: false,
		},
		{
			ID:          "notify_completion",
			Name:        "Send Completion Notification",
			Type:        StepTypeNotification,
			Description: "Notify stakeholders of rollback completion",
			Order:       6,
			Timeout:     1 * time.Minute,
			RetryPolicy: &RetryPolicy{MaxAttempts: 3, Delay: 10 * time.Second, Backoff: "linear"},
			CriticalStep: false,
		},
		{
			ID:          "cleanup",
			Name:        "Cleanup Resources",
			Type:        StepTypeCleanup,
			Description: "Clean up temporary resources and logs",
			Order:       7,
			Timeout:     2 * time.Minute,
			RetryPolicy: &RetryPolicy{MaxAttempts: 1, Delay: 0, Backoff: "none"},
			CriticalStep: false,
		},
	}
	
	return steps
}

// generatePrerequisites generates prerequisites for rollback
func (rm *RollbackManager) generatePrerequisites(deployment *Deployment) []RollbackPrerequisite {
	return []RollbackPrerequisite{
		{
			ID:        "deployment_exists",
			Name:      "Deployment Exists",
			Type:      "existence_check",
			Condition: "deployment must exist",
			Required:  true,
			CheckedAt: time.Now(),
			Met:       deployment != nil,
			Message:   "Deployment exists and is accessible",
		},
		{
			ID:        "previous_version_available",
			Name:      "Previous Version Available",
			Type:      "version_check",
			Condition: "previous version must be available for rollback",
			Required:  true,
			CheckedAt: time.Now(),
			Met:       true, // Simplified for this implementation
			Message:   "Previous version is available",
		},
	}
}

// generateValidationChecks generates validation checks for rollback
func (rm *RollbackManager) generateValidationChecks(deployment *Deployment) []ValidationCheck {
	return []ValidationCheck{
		{
			ID:        "error_rate_check",
			Name:      "Error Rate Validation",
			Type:      "metric",
			Metric:    "error_rate",
			Operator:  "lt",
			Threshold: 0.05,
			Duration:  2 * time.Minute,
		},
		{
			ID:        "response_time_check",
			Name:      "Response Time Validation",
			Type:      "metric",
			Metric:    "response_time_ms",
			Operator:  "lt",
			Threshold: 100,
			Duration:  2 * time.Minute,
		},
		{
			ID:        "detection_rate_check",
			Name:      "Detection Rate Validation",
			Type:      "metric",
			Metric:    "detection_rate",
			Operator:  "gt",
			Threshold: 0.90,
			Duration:  3 * time.Minute,
		},
	}
}

// assessRollbackRisk performs risk assessment for rollback
func (rm *RollbackManager) assessRollbackRisk(deployment *Deployment) *RiskAssessment {
	riskFactors := []RiskFactor{
		{
			Type:        "deployment_age",
			Description: "How long the deployment has been running",
			Severity:    "low",
			Likelihood:  "low",
			Impact:      "low",
			Score:       0.2,
		},
		{
			Type:        "traffic_volume",
			Description: "Current traffic volume",
			Severity:    "medium",
			Likelihood:  "medium",
			Impact:      "medium",
			Score:       0.5,
		},
	}
	
	// Calculate overall risk
	totalScore := 0.0
	for _, factor := range riskFactors {
		totalScore += factor.Score
	}
	averageScore := totalScore / float64(len(riskFactors))
	
	var overallRisk string
	if averageScore < 0.3 {
		overallRisk = "low"
	} else if averageScore < 0.7 {
		overallRisk = "medium"
	} else {
		overallRisk = "high"
	}
	
	return &RiskAssessment{
		OverallRisk:    overallRisk,
		RiskFactors:    riskFactors,
		Mitigations:    []RiskMitigation{},
		ApprovalNeeded: overallRisk == "high",
		AssessedAt:     time.Now(),
		AssessedBy:     "rollback_manager",
	}
}

// Helper methods

func (rm *RollbackManager) estimateRollbackDuration(deployment *Deployment) time.Duration {
	// Simple estimation based on deployment strategy
	baseTime := 10 * time.Minute
	
	switch deployment.Strategy {
	case RolloutGradual:
		return baseTime + 15*time.Minute
	case RolloutCanary:
		return baseTime + 10*time.Minute
	default:
		return baseTime
	}
}

func (rm *RollbackManager) checkPrerequisites(plan *RollbackPlan) bool {
	for _, prereq := range plan.Prerequisites {
		if prereq.Required && !prereq.Met {
			return false
		}
	}
	return true
}

func (rm *RollbackManager) validateRollbackStep(step RollbackStep) error {
	if step.Timeout <= 0 {
		return fmt.Errorf("step timeout must be positive")
	}
	
	if step.Order < 0 {
		return fmt.Errorf("step order must be non-negative")
	}
	
	return nil
}

func (rm *RollbackManager) initializeStepExecutions(steps []RollbackStep) []RollbackStepExecution {
	executions := make([]RollbackStepExecution, len(steps))
	
	for i, step := range steps {
		executions[i] = RollbackStepExecution{
			StepID:    step.ID,
			Status:    StepStatusPending,
			Result:    make(map[string]interface{}),
			Attempts:  0,
		}
	}
	
	return executions
}

func (rm *RollbackManager) initializeRollbackMetrics(plan *RollbackPlan) *RollbackMetrics {
	return &RollbackMetrics{
		RollbackID:          plan.ID,
		TotalSteps:          len(plan.Steps),
		CompletedSteps:      0,
		FailedSteps:         0,
		SkippedSteps:        0,
		TotalDuration:       0,
		EstimatedRemaining:  plan.EstimatedDuration,
		SuccessRate:         0,
		AverageStepDuration: 0,
		HealthChecksPassed:  0,
		HealthChecksFailed:  0,
		LastUpdated:         time.Now(),
	}
}

func (rm *RollbackManager) updateRollbackMetrics(rollback *ActiveRollback) {
	rollback.mu.Lock()
	defer rollback.mu.Unlock()
	
	metrics := rollback.Metrics
	
	completed := 0
	failed := 0
	totalDuration := time.Duration(0)
	
	for _, step := range rollback.Steps {
		switch step.Status {
		case StepStatusCompleted:
			completed++
			totalDuration += step.Duration
		case StepStatusFailed:
			failed++
			totalDuration += step.Duration
		}
	}
	
	metrics.CompletedSteps = completed
	metrics.FailedSteps = failed
	metrics.TotalDuration = totalDuration
	
	if completed > 0 {
		metrics.AverageStepDuration = totalDuration / time.Duration(completed)
		metrics.SuccessRate = float64(completed) / float64(completed+failed)
	}
	
	metrics.LastUpdated = time.Now()
}

func (rm *RollbackManager) runRollbackHealthChecks(rollback *ActiveRollback) error {
	healthChecks := []HealthCheckResult{
		{
			CheckID:    "system_health",
			Name:       "System Health Check",
			ExecutedAt: time.Now(),
		},
		{
			CheckID:    "database_connectivity",
			Name:       "Database Connectivity Check",
			ExecutedAt: time.Now(),
		},
	}
	
	for i := range healthChecks {
		check := &healthChecks[i]
		startTime := time.Now()
		
		// Simulate health check
		check.Status = HealthCheckStatusPass
		check.Message = "Health check passed"
		check.Duration = time.Since(startTime)
		
		rollback.Metrics.HealthChecksPassed++
	}
	
	rollback.HealthChecks = append(rollback.HealthChecks, healthChecks...)
	
	return nil
}

func (rm *RollbackManager) sendNotification(rollback *ActiveRollback, notificationType, message string) {
	notification := RollbackNotification{
		ID:        generateNotificationID(),
		Type:      notificationType,
		Channel:   "system",
		Recipient: "rollback_manager",
		Message:   message,
		SentAt:    time.Now(),
		Status:    "sent",
		Metadata:  map[string]interface{}{
			"rollback_id":   rollback.ID,
			"deployment_id": rollback.DeploymentID,
		},
	}
	
	rollback.Notifications = append(rollback.Notifications, notification)
	
	// In real implementation, send to actual notification channels
	fmt.Printf("ROLLBACK NOTIFICATION [%s]: %s\n", notificationType, message)
}

func (rm *RollbackManager) handleRollbackError(rollback *ActiveRollback, stepIndex int, err error) {
	rollback.mu.Lock()
	rollback.Status = RollbackStatusFailed
	rollback.ErrorMessage = err.Error()
	completedAt := time.Now()
	rollback.CompletedAt = &completedAt
	rollback.mu.Unlock()
	
	rm.sendNotification(rollback, "rollback_failed", fmt.Sprintf("Rollback failed at step %d: %s", stepIndex+1, err.Error()))
}

func (rm *RollbackManager) handleRollbackCancellation(rollback *ActiveRollback, err error) {
	rollback.mu.Lock()
	rollback.Status = RollbackStatusCancelled
	rollback.ErrorMessage = err.Error()
	completedAt := time.Now()
	rollback.CompletedAt = &completedAt
	rollback.mu.Unlock()
	
	rm.sendNotification(rollback, "rollback_cancelled", fmt.Sprintf("Rollback cancelled: %s", err.Error()))
}

// Public API methods

// GetRollbackPlan retrieves a rollback plan by ID
func (rm *RollbackManager) GetRollbackPlan(planID string) (*RollbackPlan, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	plan, exists := rm.rollbackPlans[planID]
	if !exists {
		return nil, fmt.Errorf("rollback plan not found: %s", planID)
	}
	
	return plan, nil
}

// GetActiveRollback retrieves an active rollback by ID
func (rm *RollbackManager) GetActiveRollback(rollbackID string) (*ActiveRollback, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	rollback, exists := rm.activeRollbacks[rollbackID]
	if !exists {
		return nil, fmt.Errorf("active rollback not found: %s", rollbackID)
	}
	
	return rollback, nil
}

// ListActiveRollbacks returns all currently active rollbacks
func (rm *RollbackManager) ListActiveRollbacks() []*ActiveRollback {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	rollbacks := make([]*ActiveRollback, 0, len(rm.activeRollbacks))
	for _, rollback := range rm.activeRollbacks {
		if rollback.Status == RollbackStatusRunning || rollback.Status == RollbackStatusPending {
			rollbacks = append(rollbacks, rollback)
		}
	}
	
	return rollbacks
}

// CancelRollback cancels an active rollback
func (rm *RollbackManager) CancelRollback(rollbackID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rollback, exists := rm.activeRollbacks[rollbackID]
	if !exists {
		return fmt.Errorf("active rollback not found: %s", rollbackID)
	}
	
	if rollback.Status != RollbackStatusRunning && rollback.Status != RollbackStatusPending {
		return fmt.Errorf("rollback is not active: %s", rollback.Status)
	}
	
	// Cancel the context
	if rollback.cancelFunc != nil {
		rollback.cancelFunc()
	}
	
	rollback.Status = RollbackStatusCancelled
	completedAt := time.Now()
	rollback.CompletedAt = &completedAt
	
	rm.sendNotification(rollback, "rollback_cancelled", "Rollback manually cancelled")
	
	return nil
}

// Helper functions

func generateRollbackPlanID() string {
	return fmt.Sprintf("rollback_plan_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateActiveRollbackID() string {
	return fmt.Sprintf("rollback_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateNotificationID() string {
	return fmt.Sprintf("notification_%d_%s", time.Now().Unix(), generateRandomString(6))
}