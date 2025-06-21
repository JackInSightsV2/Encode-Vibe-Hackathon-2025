package optimizer

import (
	"context"
	"fmt"
	"time"
)

// MockRuleManager implements RuleManager interface for testing
type MockRuleManager struct {
	rules map[string]*Rule
}

func NewMockRuleManager() *MockRuleManager {
	return &MockRuleManager{
		rules: make(map[string]*Rule),
	}
}

func (m *MockRuleManager) GetCurrentRule(ruleID string) (*Rule, error) {
	if rule, exists := m.rules[ruleID]; exists {
		return rule, nil
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

func (m *MockRuleManager) UpdateRule(rule Rule) error {
	m.rules[rule.ID] = &rule
	return nil
}

func (m *MockRuleManager) CreateRule(rule Rule) error {
	if _, exists := m.rules[rule.ID]; exists {
		return fmt.Errorf("rule already exists: %s", rule.ID)
	}
	m.rules[rule.ID] = &rule
	return nil
}

func (m *MockRuleManager) ArchiveRule(ruleID string) error {
	delete(m.rules, ruleID)
	return nil
}

func (m *MockRuleManager) GetRuleHistory(ruleID string) ([]Rule, error) {
	if rule, exists := m.rules[ruleID]; exists {
		return []Rule{*rule}, nil
	}
	return []Rule{}, nil
}

// MockDeploymentManager implements DeploymentManager interface for testing
type MockDeploymentManager struct {
	deployments map[string]*Deployment
	rollouts    map[string]*RolloutStatus
}

func NewMockDeploymentManager() *MockDeploymentManager {
	return &MockDeploymentManager{
		deployments: make(map[string]*Deployment),
		rollouts:    make(map[string]*RolloutStatus),
	}
}

func (m *MockDeploymentManager) CreateDeployment(ctx context.Context, request DeploymentRequest) (*Deployment, error) {
	deployment := &Deployment{
		ID:        generateDeploymentID(),
		Status:    "created",
		Strategy:  request.Strategy,
		CreatedAt: time.Now(),
	}
	
	m.deployments[deployment.ID] = deployment
	return deployment, nil
}

func (m *MockDeploymentManager) GetDeployment(deploymentID string) (*Deployment, error) {
	if deployment, exists := m.deployments[deploymentID]; exists {
		return deployment, nil
	}
	
	// Return a default deployment for testing
	return &Deployment{
		ID:        deploymentID,
		Status:    "active",
		Strategy:  RolloutGradual,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}, nil
}

func (m *MockDeploymentManager) StartRollout(ctx context.Context, deploymentID string) error {
	rolloutStatus := &RolloutStatus{
		DeploymentID:   deploymentID,
		Phase:          "initial",
		TrafficPercent: 0,
		Status:         "starting",
		LastUpdated:    time.Now(),
	}
	
	m.rollouts[deploymentID] = rolloutStatus
	return nil
}

func (m *MockDeploymentManager) StopRollout(ctx context.Context, deploymentID string) error {
	if rollout, exists := m.rollouts[deploymentID]; exists {
		rollout.Status = "stopped"
		rollout.LastUpdated = time.Now()
	}
	return nil
}

func (m *MockDeploymentManager) GetRolloutStatus(deploymentID string) (*RolloutStatus, error) {
	if rollout, exists := m.rollouts[deploymentID]; exists {
		return rollout, nil
	}
	return nil, fmt.Errorf("rollout not found: %s", deploymentID)
}

// MockApprovalService implements ApprovalService interface for testing
type MockApprovalService struct {
	workflows map[string]*ApprovalWorkflow
}

func NewMockApprovalService() *MockApprovalService {
	return &MockApprovalService{
		workflows: make(map[string]*ApprovalWorkflow),
	}
}

func (m *MockApprovalService) RequestApproval(ctx context.Context, request ApprovalRequest) (*ApprovalWorkflow, error) {
	workflow := &ApprovalWorkflow{
		ID:        generateWorkflowID(),
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	
	m.workflows[workflow.ID] = workflow
	return workflow, nil
}

func (m *MockApprovalService) GetApprovalStatus(workflowID string) (*ApprovalWorkflow, error) {
	if workflow, exists := m.workflows[workflowID]; exists {
		return workflow, nil
	}
	return nil, fmt.Errorf("workflow not found: %s", workflowID)
}

func (m *MockApprovalService) SubmitApproval(approvalID string, decision ApprovalDecision) error {
	// Mock implementation - in real system this would update the approval
	return nil
}

// Helper functions for generating IDs
func generateDeploymentID() string {
	return fmt.Sprintf("deploy_%d_%s", time.Now().Unix(), generateRandomString(6))
}

func generateWorkflowID() string {
	return fmt.Sprintf("workflow_%d_%s", time.Now().Unix(), generateRandomString(6))
}

func generateExperimentID() string {
	return fmt.Sprintf("exp_%d_%s", time.Now().Unix(), generateRandomString(8))
}

// MockMetricsCollector for testing metrics collection
type MockMetricsCollector struct {
	metrics map[string][]MetricPoint
}

type MetricPoint struct {
	Timestamp time.Time
	Value     float64
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: make(map[string][]MetricPoint),
	}
}

func (m *MockMetricsCollector) RecordMetric(metricName string, value float64) {
	if m.metrics[metricName] == nil {
		m.metrics[metricName] = make([]MetricPoint, 0)
	}
	
	m.metrics[metricName] = append(m.metrics[metricName], MetricPoint{
		Timestamp: time.Now(),
		Value:     value,
	})
}

func (m *MockMetricsCollector) GetMetrics(metricName string, since time.Time) []MetricPoint {
	if points, exists := m.metrics[metricName]; exists {
		filtered := make([]MetricPoint, 0)
		for _, point := range points {
			if point.Timestamp.After(since) {
				filtered = append(filtered, point)
			}
		}
		return filtered
	}
	return []MetricPoint{}
}

// MockNotificationService for testing notifications
type MockNotificationService struct {
	notifications []Notification
}

type Notification struct {
	ID        string
	Type      string
	Message   string
	Recipient string
	SentAt    time.Time
}

func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		notifications: make([]Notification, 0),
	}
}

func (m *MockNotificationService) SendNotification(notificationType, message, recipient string) error {
	notification := Notification{
		ID:        generateNotificationID(),
		Type:      notificationType,
		Message:   message,
		Recipient: recipient,
		SentAt:    time.Now(),
	}
	
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *MockNotificationService) GetNotifications() []Notification {
	return m.notifications
}

func generateNotificationID() string {
	return fmt.Sprintf("notif_%d_%s", time.Now().Unix(), generateRandomString(4))
}

// MockOpikClient for testing Opik integration
type MockOpikClient struct {
	experiments map[string]*OpikExperiment
	traces      map[string]*OpikTrace
}

type OpikExperiment struct {
	ID       string
	Name     string
	Status   string
	Metadata map[string]interface{}
}

type OpikTrace struct {
	ID           string
	ExperimentID string
	Input        map[string]interface{}
	Output       map[string]interface{}
	Metadata     map[string]interface{}
	StartTime    time.Time
	EndTime      time.Time
}

func NewMockOpikClient() *MockOpikClient {
	return &MockOpikClient{
		experiments: make(map[string]*OpikExperiment),
		traces:      make(map[string]*OpikTrace),
	}
}

func (m *MockOpikClient) CreateExperiment(name string, metadata map[string]interface{}) (*OpikExperiment, error) {
	experiment := &OpikExperiment{
		ID:       generateOpikExperimentID(),
		Name:     name,
		Status:   "active",
		Metadata: metadata,
	}
	
	m.experiments[experiment.ID] = experiment
	return experiment, nil
}

func (m *MockOpikClient) LogTrace(experimentID string, input, output, metadata map[string]interface{}) (*OpikTrace, error) {
	trace := &OpikTrace{
		ID:           generateOpikTraceID(),
		ExperimentID: experimentID,
		Input:        input,
		Output:       output,
		Metadata:     metadata,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(10 * time.Millisecond),
	}
	
	m.traces[trace.ID] = trace
	return trace, nil
}

func (m *MockOpikClient) GetExperiment(experimentID string) (*OpikExperiment, error) {
	if experiment, exists := m.experiments[experimentID]; exists {
		return experiment, nil
	}
	return nil, fmt.Errorf("experiment not found: %s", experimentID)
}

func (m *MockOpikClient) GetTraces(experimentID string) ([]*OpikTrace, error) {
	traces := make([]*OpikTrace, 0)
	for _, trace := range m.traces {
		if trace.ExperimentID == experimentID {
			traces = append(traces, trace)
		}
	}
	return traces, nil
}

func generateOpikExperimentID() string {
	return fmt.Sprintf("opik_exp_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateOpikTraceID() string {
	return fmt.Sprintf("opik_trace_%d_%s", time.Now().Unix(), generateRandomString(8))
}

// MockLoadBalancer for testing traffic routing
type MockLoadBalancer struct {
	routes map[string]*TrafficRoute
}

func NewMockLoadBalancer() *MockLoadBalancer {
	return &MockLoadBalancer{
		routes: make(map[string]*TrafficRoute),
	}
}

func (m *MockLoadBalancer) UpdateRoute(ruleID string, trafficPercent float64) error {
	route := &TrafficRoute{
		RolloutID:      "mock_rollout",
		OldRuleID:      "current_rule",
		NewRuleID:      ruleID,
		TrafficPercent: trafficPercent,
		LastUpdated:    time.Now(),
	}
	
	m.routes[ruleID] = route
	return nil
}

func (m *MockLoadBalancer) GetRoute(ruleID string) (*TrafficRoute, error) {
	if route, exists := m.routes[ruleID]; exists {
		return route, nil
	}
	return nil, fmt.Errorf("route not found: %s", ruleID)
}

func (m *MockLoadBalancer) RemoveRoute(ruleID string) error {
	delete(m.routes, ruleID)
	return nil
}

// MockHealthChecker for testing health checks
type MockHealthChecker struct {
	status map[string]bool
}

func NewMockHealthChecker() *MockHealthChecker {
	return &MockHealthChecker{
		status: make(map[string]bool),
	}
}

func (m *MockHealthChecker) CheckHealth(component string) (bool, error) {
	if status, exists := m.status[component]; exists {
		return status, nil
	}
	// Default to healthy
	return true, nil
}

func (m *MockHealthChecker) SetHealthStatus(component string, healthy bool) {
	m.status[component] = healthy
}

func (m *MockHealthChecker) GetAllHealthStatuses() map[string]bool {
	result := make(map[string]bool)
	for component, status := range m.status {
		result[component] = status
	}
	return result
}