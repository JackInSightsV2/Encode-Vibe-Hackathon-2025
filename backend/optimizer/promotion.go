package optimizer

import (
	"context"
	"fmt"
	"time"
)

// PromotionEngine handles promoting winning rule variants to production
type PromotionEngine struct {
	ruleManager      RuleManager
	deploymentManager DeploymentManager
	rollbackManager  RollbackManager
	approvalService  ApprovalService
	config           *PromotionConfig
}

// PromotionConfig configures the promotion pipeline
type PromotionConfig struct {
	AutoPromotionEnabled      bool          `json:"auto_promotion_enabled"`
	RequireApproval           bool          `json:"require_approval"`
	MinSignificanceLevel      float64       `json:"min_significance_level"`
	MinImprovementThreshold   float64       `json:"min_improvement_threshold"`
	MinSampleSize             int           `json:"min_sample_size"`
	SafetyCheckTimeout        time.Duration `json:"safety_check_timeout"`
	RolloutStrategy           RolloutStrategy `json:"rollout_strategy"`
	MaxConcurrentPromotions   int           `json:"max_concurrent_promotions"`
	PromotionTimeout          time.Duration `json:"promotion_timeout"`
	EnableCanaryDeployment    bool          `json:"enable_canary_deployment"`
	CanaryTrafficPercent      float64       `json:"canary_traffic_percent"`
	CanaryDuration            time.Duration `json:"canary_duration"`
}

// RolloutStrategy defines different rollout strategies
type RolloutStrategy string

const (
	RolloutImmediate   RolloutStrategy = "immediate"
	RolloutGradual     RolloutStrategy = "gradual"
	RolloutCanary      RolloutStrategy = "canary"
	RolloutBlueGreen   RolloutStrategy = "blue_green"
)

// PromotionRequest represents a request to promote a rule variant
type PromotionRequest struct {
	ID               string                 `json:"id"`
	ExperimentID     string                 `json:"experiment_id"`
	WinningVariantID string                 `json:"winning_variant_id"`
	RequestedBy      string                 `json:"requested_by"`
	RequestedAt      time.Time              `json:"requested_at"`
	Justification    string                 `json:"justification"`
	Strategy         RolloutStrategy        `json:"strategy"`
	TargetEnvironment string                `json:"target_environment"`
	Metadata         map[string]interface{} `json:"metadata"`
	
	// Approval workflow
	ApprovalRequired bool               `json:"approval_required"`
	ApprovalStatus   ApprovalStatus     `json:"approval_status"`
	Approvals        []Approval         `json:"approvals"`
	
	// Deployment tracking
	DeploymentID     string             `json:"deployment_id"`
	Status           PromotionStatus    `json:"status"`
	StartedAt        *time.Time         `json:"started_at,omitempty"`
	CompletedAt      *time.Time         `json:"completed_at,omitempty"`
	ErrorMessage     string             `json:"error_message,omitempty"`
}

// PromotionStatus represents the status of a promotion
type PromotionStatus string

const (
	PromotionStatusPending     PromotionStatus = "pending"
	PromotionStatusApproved    PromotionStatus = "approved"
	PromotionStatusRejected    PromotionStatus = "rejected"
	PromotionStatusDeploying   PromotionStatus = "deploying"
	PromotionStatusCanary      PromotionStatus = "canary"
	PromotionStatusRollingOut  PromotionStatus = "rolling_out"
	PromotionStatusCompleted   PromotionStatus = "completed"
	PromotionStatusFailed      PromotionStatus = "failed"
	PromotionStatusRolledBack  PromotionStatus = "rolled_back"
)

// ApprovalStatus represents approval workflow status
type ApprovalStatus string

const (
	ApprovalStatusPending   ApprovalStatus = "pending"
	ApprovalStatusApproved  ApprovalStatus = "approved"
	ApprovalStatusRejected  ApprovalStatus = "rejected"
)

// Approval represents an individual approval
type Approval struct {
	ID          string         `json:"id"`
	ApproverID  string         `json:"approver_id"`
	Status      ApprovalStatus `json:"status"`
	Comments    string         `json:"comments"`
	ApprovedAt  *time.Time     `json:"approved_at,omitempty"`
	RejectedAt  *time.Time     `json:"rejected_at,omitempty"`
}

// RuleManager interface for managing rules in production
type RuleManager interface {
	GetCurrentRule(ruleID string) (*Rule, error)
	UpdateRule(rule Rule) error
	CreateRule(rule Rule) error
	ArchiveRule(ruleID string) error
	GetRuleHistory(ruleID string) ([]Rule, error)
}

// DeploymentManager interface for managing deployments
type DeploymentManager interface {
	CreateDeployment(ctx context.Context, request DeploymentRequest) (*Deployment, error)
	GetDeployment(deploymentID string) (*Deployment, error)
	StartRollout(ctx context.Context, deploymentID string) error
	StopRollout(ctx context.Context, deploymentID string) error
	GetRolloutStatus(deploymentID string) (*RolloutStatus, error)
}

// RollbackManager interface for managing rollbacks
type RollbackManager interface {
	CreateRollbackPlan(deploymentID string) (*RollbackPlan, error)
	ExecuteRollback(ctx context.Context, rollbackID string) error
	ValidateRollback(rollbackID string) error
}

// ApprovalService interface for approval workflows
type ApprovalService interface {
	RequestApproval(ctx context.Context, request ApprovalRequest) (*ApprovalWorkflow, error)
	GetApprovalStatus(workflowID string) (*ApprovalWorkflow, error)
	SubmitApproval(approvalID string, decision ApprovalDecision) error
}

// NewPromotionEngine creates a new promotion engine
func NewPromotionEngine(
	ruleManager RuleManager,
	deploymentManager DeploymentManager,
	rollbackManager RollbackManager,
	approvalService ApprovalService,
	config *PromotionConfig,
) *PromotionEngine {
	if config == nil {
		config = GetDefaultPromotionConfig()
	}
	
	return &PromotionEngine{
		ruleManager:       ruleManager,
		deploymentManager: deploymentManager,
		rollbackManager:   rollbackManager,
		approvalService:   approvalService,
		config:            config,
	}
}

// GetDefaultPromotionConfig returns default promotion configuration
func GetDefaultPromotionConfig() *PromotionConfig {
	return &PromotionConfig{
		AutoPromotionEnabled:      false, // Disabled by default for safety
		RequireApproval:           true,
		MinSignificanceLevel:      0.95,
		MinImprovementThreshold:   0.05, // 5% minimum improvement
		MinSampleSize:             1000,
		SafetyCheckTimeout:        5 * time.Minute,
		RolloutStrategy:           RolloutGradual,
		MaxConcurrentPromotions:   3,
		PromotionTimeout:          30 * time.Minute,
		EnableCanaryDeployment:    true,
		CanaryTrafficPercent:      5.0,
		CanaryDuration:            1 * time.Hour,
	}
}

// EvaluateForPromotion evaluates if an experiment winner should be promoted
func (pe *PromotionEngine) EvaluateForPromotion(ctx context.Context, experiment *Experiment) (*PromotionRecommendation, error) {
	recommendation := &PromotionRecommendation{
		ExperimentID:  experiment.ID,
		EvaluatedAt:   time.Now(),
		ShouldPromote: false,
		Reasons:       make([]string, 0),
	}
	
	// Check if experiment is completed
	if experiment.Status != ExperimentStatusCompleted {
		recommendation.Reasons = append(recommendation.Reasons, "experiment_not_completed")
		return recommendation, nil
	}
	
	// Check if there's a clear winner
	if experiment.Results.Winner == nil {
		recommendation.Reasons = append(recommendation.Reasons, "no_clear_winner")
		return recommendation, nil
	}
	
	winner := experiment.Results.Winner
	control := experiment.Results.GetVariantMetrics("control")
	
	if control == nil {
		recommendation.Reasons = append(recommendation.Reasons, "no_control_metrics")
		return recommendation, nil
	}
	
	// Check sample size requirement
	if winner.SampleSize < pe.config.MinSampleSize {
		recommendation.Reasons = append(recommendation.Reasons, 
			fmt.Sprintf("insufficient_sample_size_%d_required_%d", winner.SampleSize, pe.config.MinSampleSize))
		return recommendation, nil
	}
	
	// Check statistical significance
	if experiment.Results.Significance == nil || !experiment.Results.Significance.Significant {
		recommendation.Reasons = append(recommendation.Reasons, "not_statistically_significant")
		return recommendation, nil
	}
	
	if experiment.Results.ConfidenceLevel < pe.config.MinSignificanceLevel {
		recommendation.Reasons = append(recommendation.Reasons, 
			fmt.Sprintf("confidence_level_too_low_%.3f_required_%.3f", 
				experiment.Results.ConfidenceLevel, pe.config.MinSignificanceLevel))
		return recommendation, nil
	}
	
	// Check improvement threshold
	improvement := (winner.DetectionRate - control.DetectionRate) / control.DetectionRate
	if improvement < pe.config.MinImprovementThreshold {
		recommendation.Reasons = append(recommendation.Reasons, 
			fmt.Sprintf("improvement_too_small_%.3f_required_%.3f", 
				improvement, pe.config.MinImprovementThreshold))
		return recommendation, nil
	}
	
	// Check safety metrics
	if err := pe.validateSafetyMetrics(winner); err != nil {
		recommendation.Reasons = append(recommendation.Reasons, 
			fmt.Sprintf("safety_violation_%s", err.Error()))
		return recommendation, nil
	}
	
	// All checks passed
	recommendation.ShouldPromote = true
	recommendation.WinningVariantID = winner.VariantID
	recommendation.ExpectedImprovement = improvement
	recommendation.ConfidenceLevel = experiment.Results.ConfidenceLevel
	recommendation.RecommendedStrategy = pe.selectRolloutStrategy(experiment)
	
	return recommendation, nil
}

// CreatePromotionRequest creates a new promotion request
func (pe *PromotionEngine) CreatePromotionRequest(ctx context.Context, request CreatePromotionRequest) (*PromotionRequest, error) {
	// Validate the request
	if err := pe.validatePromotionRequest(request); err != nil {
		return nil, fmt.Errorf("invalid promotion request: %w", err)
	}
	
	promotionRequest := &PromotionRequest{
		ID:               generatePromotionID(),
		ExperimentID:     request.ExperimentID,
		WinningVariantID: request.WinningVariantID,
		RequestedBy:      request.RequestedBy,
		RequestedAt:      time.Now(),
		Justification:    request.Justification,
		Strategy:         request.Strategy,
		TargetEnvironment: request.TargetEnvironment,
		Metadata:         request.Metadata,
		ApprovalRequired: pe.config.RequireApproval,
		ApprovalStatus:   ApprovalStatusPending,
		Status:           PromotionStatusPending,
		Approvals:        make([]Approval, 0),
	}
	
	// If approval is required, initiate approval workflow
	if pe.config.RequireApproval {
		if err := pe.initiateApprovalWorkflow(ctx, promotionRequest); err != nil {
			return nil, fmt.Errorf("failed to initiate approval workflow: %w", err)
		}
	} else {
		// Auto-approve if approval not required
		promotionRequest.ApprovalStatus = ApprovalStatusApproved
		promotionRequest.Status = PromotionStatusApproved
	}
	
	return promotionRequest, nil
}

// ProcessPromotion processes an approved promotion request
func (pe *PromotionEngine) ProcessPromotion(ctx context.Context, promotionID string) error {
	// In a real implementation, this would retrieve the promotion request from storage
	// For now, we'll simulate the process
	
	// Update status to deploying
	// promotionRequest.Status = PromotionStatusDeploying
	
	// Create deployment
	deploymentRequest := DeploymentRequest{
		PromotionID:   promotionID,
		Strategy:      pe.config.RolloutStrategy,
		Environment:   "production",
		CreatedBy:     "promotion_engine",
		CreatedAt:     time.Now(),
	}
	
	deployment, err := pe.deploymentManager.CreateDeployment(ctx, deploymentRequest)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	
	// Start rollout based on strategy
	switch pe.config.RolloutStrategy {
	case RolloutImmediate:
		return pe.executeImmediateRollout(ctx, deployment.ID)
	case RolloutGradual:
		return pe.executeGradualRollout(ctx, deployment.ID)
	case RolloutCanary:
		return pe.executeCanaryRollout(ctx, deployment.ID)
	case RolloutBlueGreen:
		return pe.executeBlueGreenRollout(ctx, deployment.ID)
	default:
		return fmt.Errorf("unsupported rollout strategy: %s", pe.config.RolloutStrategy)
	}
}

// executeImmediateRollout executes immediate 100% rollout
func (pe *PromotionEngine) executeImmediateRollout(ctx context.Context, deploymentID string) error {
	// Start immediate rollout
	if err := pe.deploymentManager.StartRollout(ctx, deploymentID); err != nil {
		return fmt.Errorf("failed to start immediate rollout: %w", err)
	}
	
	// Monitor rollout status
	return pe.monitorRollout(ctx, deploymentID)
}

// executeGradualRollout executes gradual percentage-based rollout
func (pe *PromotionEngine) executeGradualRollout(ctx context.Context, deploymentID string) error {
	// Gradual rollout phases: 5% -> 10% -> 25% -> 50% -> 100%
	phases := []float64{5, 10, 25, 50, 100}
	phaseDelay := 10 * time.Minute
	
	for i, percentage := range phases {
		// Update rollout percentage
		if err := pe.updateRolloutPercentage(ctx, deploymentID, percentage); err != nil {
			return fmt.Errorf("failed to update rollout to %.0f%%: %w", percentage, err)
		}
		
		// Monitor this phase
		if err := pe.monitorPhase(ctx, deploymentID, percentage, phaseDelay); err != nil {
			return fmt.Errorf("phase %.0f%% failed: %w", percentage, err)
		}
		
		// Wait before next phase (except for the last phase)
		if i < len(phases)-1 {
			time.Sleep(phaseDelay)
		}
	}
	
	return nil
}

// executeCanaryRollout executes canary deployment
func (pe *PromotionEngine) executeCanaryRollout(ctx context.Context, deploymentID string) error {
	// Start canary with small percentage
	if err := pe.updateRolloutPercentage(ctx, deploymentID, pe.config.CanaryTrafficPercent); err != nil {
		return fmt.Errorf("failed to start canary rollout: %w", err)
	}
	
	// Monitor canary for specified duration
	if err := pe.monitorCanary(ctx, deploymentID, pe.config.CanaryDuration); err != nil {
		return fmt.Errorf("canary monitoring failed: %w", err)
	}
	
	// If canary successful, proceed with full rollout
	return pe.executeGradualRollout(ctx, deploymentID)
}

// executeBlueGreenRollout executes blue-green deployment
func (pe *PromotionEngine) executeBlueGreenRollout(ctx context.Context, deploymentID string) error {
	// In blue-green deployment, we prepare the new environment (green)
	// then switch traffic all at once
	
	// Prepare green environment
	if err := pe.prepareGreenEnvironment(ctx, deploymentID); err != nil {
		return fmt.Errorf("failed to prepare green environment: %w", err)
	}
	
	// Validate green environment
	if err := pe.validateGreenEnvironment(ctx, deploymentID); err != nil {
		return fmt.Errorf("green environment validation failed: %w", err)
	}
	
	// Switch traffic to green
	if err := pe.switchToGreen(ctx, deploymentID); err != nil {
		return fmt.Errorf("failed to switch to green environment: %w", err)
	}
	
	// Monitor the switch
	return pe.monitorRollout(ctx, deploymentID)
}

// validateSafetyMetrics validates that metrics meet safety requirements
func (pe *PromotionEngine) validateSafetyMetrics(metrics *VariantMetrics) error {
	constraints := GetDefaultGlobalConstraints()
	
	if metrics.DetectionRate < constraints.SafetyThresholds.MinDetectionRate {
		return fmt.Errorf("detection_rate_%.3f_below_minimum_%.3f", 
			metrics.DetectionRate, constraints.SafetyThresholds.MinDetectionRate)
	}
	
	if metrics.FalsePositiveRate > constraints.SafetyThresholds.MaxFalsePositiveRate {
		return fmt.Errorf("false_positive_rate_%.3f_above_maximum_%.3f", 
			metrics.FalsePositiveRate, constraints.SafetyThresholds.MaxFalsePositiveRate)
	}
	
	responseTimeMs := metrics.AvgResponseTime.Seconds() * 1000
	if responseTimeMs > constraints.SafetyThresholds.MaxResponseTimeMs {
		return fmt.Errorf("response_time_%.1fms_above_maximum_%.1fms", 
			responseTimeMs, constraints.SafetyThresholds.MaxResponseTimeMs)
	}
	
	if metrics.UserSatisfaction < constraints.SafetyThresholds.MinUserSatisfaction {
		return fmt.Errorf("user_satisfaction_%.3f_below_minimum_%.3f", 
			metrics.UserSatisfaction, constraints.SafetyThresholds.MinUserSatisfaction)
	}
	
	return nil
}

// selectRolloutStrategy selects the appropriate rollout strategy
func (pe *PromotionEngine) selectRolloutStrategy(experiment *Experiment) RolloutStrategy {
	// Default to configured strategy
	strategy := pe.config.RolloutStrategy
	
	// Adjust based on experiment characteristics
	if experiment.Results.Winner != nil {
		improvement := experiment.Results.Significance.Improvement
		
		// Use more conservative strategy for large improvements (might be risky)
		if improvement > 0.5 { // More than 50% improvement
			strategy = RolloutCanary
		} else if improvement > 0.2 { // More than 20% improvement
			strategy = RolloutGradual
		}
		
		// Use immediate rollout for small, safe improvements
		if improvement < 0.1 && experiment.Results.ConfidenceLevel > 0.99 {
			strategy = RolloutImmediate
		}
	}
	
	return strategy
}

// validatePromotionRequest validates a promotion request
func (pe *PromotionEngine) validatePromotionRequest(request CreatePromotionRequest) error {
	if request.ExperimentID == "" {
		return fmt.Errorf("experiment_id is required")
	}
	
	if request.WinningVariantID == "" {
		return fmt.Errorf("winning_variant_id is required")
	}
	
	if request.RequestedBy == "" {
		return fmt.Errorf("requested_by is required")
	}
	
	if request.TargetEnvironment == "" {
		request.TargetEnvironment = "production"
	}
	
	return nil
}

// initiateApprovalWorkflow initiates the approval workflow
func (pe *PromotionEngine) initiateApprovalWorkflow(ctx context.Context, promotionRequest *PromotionRequest) error {
	approvalRequest := ApprovalRequest{
		PromotionID:     promotionRequest.ID,
		Title:           fmt.Sprintf("Promote Rule Variant from Experiment %s", promotionRequest.ExperimentID),
		Description:     promotionRequest.Justification,
		RequestedBy:     promotionRequest.RequestedBy,
		RequiredApprovers: []string{"security_team", "product_owner"}, // Configurable
		Metadata: map[string]interface{}{
			"experiment_id":      promotionRequest.ExperimentID,
			"winning_variant_id": promotionRequest.WinningVariantID,
			"strategy":           promotionRequest.Strategy,
		},
	}
	
	workflow, err := pe.approvalService.RequestApproval(ctx, approvalRequest)
	if err != nil {
		return err
	}
	
	promotionRequest.Metadata["approval_workflow_id"] = workflow.ID
	return nil
}

// Helper methods for rollout phases

func (pe *PromotionEngine) updateRolloutPercentage(ctx context.Context, deploymentID string, percentage float64) error {
	// In a real implementation, this would update the deployment configuration
	// to route the specified percentage of traffic to the new rule
	fmt.Printf("Updating rollout to %.0f%% for deployment %s\n", percentage, deploymentID)
	return nil
}

func (pe *PromotionEngine) monitorPhase(ctx context.Context, deploymentID string, percentage float64, duration time.Duration) error {
	// Monitor metrics during this phase
	fmt.Printf("Monitoring %.0f%% phase for %v for deployment %s\n", percentage, duration, deploymentID)
	
	// Simulate monitoring
	time.Sleep(time.Second) // In real implementation, this would be actual monitoring
	
	return nil
}

func (pe *PromotionEngine) monitorCanary(ctx context.Context, deploymentID string, duration time.Duration) error {
	fmt.Printf("Monitoring canary for %v for deployment %s\n", duration, deploymentID)
	
	// In real implementation, this would:
	// 1. Collect metrics from canary traffic
	// 2. Compare with baseline metrics
	// 3. Alert if significant degradation detected
	// 4. Automatically rollback if critical issues found
	
	return nil
}

func (pe *PromotionEngine) monitorRollout(ctx context.Context, deploymentID string) error {
	fmt.Printf("Monitoring rollout for deployment %s\n", deploymentID)
	
	// Monitor the rollout and ensure everything is working correctly
	return nil
}

func (pe *PromotionEngine) prepareGreenEnvironment(ctx context.Context, deploymentID string) error {
	fmt.Printf("Preparing green environment for deployment %s\n", deploymentID)
	return nil
}

func (pe *PromotionEngine) validateGreenEnvironment(ctx context.Context, deploymentID string) error {
	fmt.Printf("Validating green environment for deployment %s\n", deploymentID)
	return nil
}

func (pe *PromotionEngine) switchToGreen(ctx context.Context, deploymentID string) error {
	fmt.Printf("Switching to green environment for deployment %s\n", deploymentID)
	return nil
}

// Supporting types and interfaces

type PromotionRecommendation struct {
	ExperimentID         string          `json:"experiment_id"`
	ShouldPromote        bool            `json:"should_promote"`
	WinningVariantID     string          `json:"winning_variant_id,omitempty"`
	ExpectedImprovement  float64         `json:"expected_improvement,omitempty"`
	ConfidenceLevel      float64         `json:"confidence_level,omitempty"`
	RecommendedStrategy  RolloutStrategy `json:"recommended_strategy,omitempty"`
	Reasons              []string        `json:"reasons"`
	EvaluatedAt          time.Time       `json:"evaluated_at"`
}

type CreatePromotionRequest struct {
	ExperimentID      string                 `json:"experiment_id"`
	WinningVariantID  string                 `json:"winning_variant_id"`
	RequestedBy       string                 `json:"requested_by"`
	Justification     string                 `json:"justification"`
	Strategy          RolloutStrategy        `json:"strategy"`
	TargetEnvironment string                 `json:"target_environment"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type DeploymentRequest struct {
	PromotionID string          `json:"promotion_id"`
	Strategy    RolloutStrategy `json:"strategy"`
	Environment string          `json:"environment"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Deployment struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	Strategy  RolloutStrategy `json:"strategy"`
	CreatedAt time.Time       `json:"created_at"`
}

type RolloutStatus struct {
	DeploymentID    string    `json:"deployment_id"`
	Phase           string    `json:"phase"`
	TrafficPercent  float64   `json:"traffic_percent"`
	Status          string    `json:"status"`
	LastUpdated     time.Time `json:"last_updated"`
}

type RollbackPlan struct {
	ID           string    `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type ApprovalRequest struct {
	PromotionID       string                 `json:"promotion_id"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	RequestedBy       string                 `json:"requested_by"`
	RequiredApprovers []string               `json:"required_approvers"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type ApprovalWorkflow struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ApprovalDecision struct {
	Approved bool   `json:"approved"`
	Comments string `json:"comments"`
}

// Helper functions

func generatePromotionID() string {
	return fmt.Sprintf("promotion_%d_%s", time.Now().Unix(), generateRandomString(8))
}