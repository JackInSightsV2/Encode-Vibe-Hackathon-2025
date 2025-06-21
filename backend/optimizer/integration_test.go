package optimizer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// IntegrationTestSuite runs comprehensive tests for the self-optimizing rules engine
type IntegrationTestSuite struct {
	optimizer       *OptimizerClient
	experimentMgr   *ExperimentManager
	driftDetector   *DriftDetector
	promotionEngine *PromotionEngine
	rolloutManager  *RolloutManager
	rollbackManager *RollbackManager
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite() *IntegrationTestSuite {
	// Initialize all components with test configurations
	optimizer := NewOptimizerClient(GetDefaultOptimizerConfig())
	experimentMgr := NewExperimentManager(GetDefaultExperimentManagerConfig())
	driftDetector := NewDriftDetector(GetDefaultDriftDetectionConfig())
	
	// Mock interfaces for testing
	ruleManager := &MockRuleManager{}
	deploymentManager := &MockDeploymentManager{}
	rollbackManager := NewRollbackManager(nil, deploymentManager, ruleManager, GetDefaultRollbackConfig())
	approvalService := &MockApprovalService{}
	
	promotionEngine := NewPromotionEngine(
		ruleManager,
		deploymentManager,
		rollbackManager,
		approvalService,
		GetDefaultPromotionConfig(),
	)
	
	rolloutManager := NewRolloutManager(GetDefaultRolloutConfig())
	
	return &IntegrationTestSuite{
		optimizer:       optimizer,
		experimentMgr:   experimentMgr,
		driftDetector:   driftDetector,
		promotionEngine: promotionEngine,
		rolloutManager:  rolloutManager,
		rollbackManager: rollbackManager,
	}
}

// RunFullIntegrationTest executes the complete integration test workflow
func (its *IntegrationTestSuite) RunFullIntegrationTest(t *testing.T) {
	ctx := context.Background()
	
	t.Log("Starting comprehensive integration test...")
	
	// Test 1: Rule Generation and Optimization
	t.Run("Rule Generation", func(t *testing.T) {
		its.testRuleGeneration(t, ctx)
	})
	
	// Test 2: A/B Testing Workflow
	t.Run("A/B Testing", func(t *testing.T) {
		its.testABTestingWorkflow(t, ctx)
	})
	
	// Test 3: Drift Detection
	t.Run("Drift Detection", func(t *testing.T) {
		its.testDriftDetection(t, ctx)
	})
	
	// Test 4: Promotion Pipeline
	t.Run("Promotion Pipeline", func(t *testing.T) {
		its.testPromotionPipeline(t, ctx)
	})
	
	// Test 5: Rollout Management
	t.Run("Rollout Management", func(t *testing.T) {
		its.testRolloutManagement(t, ctx)
	})
	
	// Test 6: Rollback Scenarios
	t.Run("Rollback Scenarios", func(t *testing.T) {
		its.testRollbackScenarios(t, ctx)
	})
	
	// Test 7: End-to-End Optimization Cycle
	t.Run("End-to-End Cycle", func(t *testing.T) {
		its.testEndToEndOptimizationCycle(t, ctx)
	})
	
	// Test 8: Performance and Scalability
	t.Run("Performance Tests", func(t *testing.T) {
		its.testPerformanceAndScalability(t, ctx)
	})
	
	// Test 9: Error Handling and Recovery
	t.Run("Error Handling", func(t *testing.T) {
		its.testErrorHandlingAndRecovery(t, ctx)
	})
	
	// Test 10: Concurrent Operations
	t.Run("Concurrent Operations", func(t *testing.T) {
		its.testConcurrentOperations(t, ctx)
	})
}

// testRuleGeneration tests the rule generation and mutation system
func (its *IntegrationTestSuite) testRuleGeneration(t *testing.T, ctx context.Context) {
	// Create base rule
	baseRule := Rule{
		ID:   "base_rule_001",
		Name: "Base Safety Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "(?i)(hate|violence|threat)",
			"flags":   "i",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.85,
	}
	
	// Generate variants
	objectives := GetDefaultObjectives()
	variants, err := its.optimizer.GenerateRuleVariants(ctx, baseRule, 5, objectives)
	if err != nil {
		t.Fatalf("Failed to generate rule variants: %v", err)
	}
	
	if len(variants) == 0 {
		t.Error("No variants generated")
	}
	
	// Test mutation engine
	mutationEngine := NewMutationEngine(GetDefaultMutationConfig())
	mutatedRules, err := mutationEngine.MutateRule(baseRule, 3)
	if err != nil {
		t.Fatalf("Failed to mutate rule: %v", err)
	}
	
	if len(mutatedRules) != 3 {
		t.Errorf("Expected 3 mutated rules, got %d", len(mutatedRules))
	}
	
	// Test combination engine
	combinationEngine := NewCombinationEngine(GetDefaultCombinationConfig())
	hybridRule, err := combinationEngine.CombineRules([]Rule{baseRule, variants[0]}, WeightedCombination)
	if err != nil {
		t.Fatalf("Failed to combine rules: %v", err)
	}
	
	if hybridRule.Type != HybridRule {
		t.Errorf("Expected hybrid rule, got %s", hybridRule.Type)
	}
	
	t.Logf("Successfully generated %d variants and created hybrid rules", len(variants))
}

// testABTestingWorkflow tests the complete A/B testing workflow
func (its *IntegrationTestSuite) testABTestingWorkflow(t *testing.T, ctx context.Context) {
	// Create experiment configuration
	config := ExperimentConfig{
		Name: "Safety Rule Enhancement Test",
		Objectives: []Objective{
			{Name: "detection_rate", Weight: 0.4, Target: 0.95, Type: Maximize},
			{Name: "false_positive_rate", Weight: 0.3, Target: 0.05, Type: Minimize},
		},
		Control: Rule{
			ID:   "control_rule",
			Name: "Current Production Rule",
			Type: RegexRule,
			Parameters: map[string]interface{}{
				"pattern": "baseline_pattern",
			},
			Enabled:  true,
			Priority: 10,
			Fitness:  0.80,
		},
		Variants: []Rule{
			{
				ID:   "variant_rule_001",
				Name: "Enhanced Detection Rule",
				Type: SemanticRule,
				Parameters: map[string]interface{}{
					"model": "enhanced_model",
				},
				Enabled:  true,
				Priority: 10,
				Fitness:  0.85,
			},
		},
		Traffic: TrafficSplit{
			Control:  0.6,
			Variants: map[string]float64{"variant_rule_001": 0.4},
		},
		Duration:               24 * time.Hour,
		MinSampleSize:          1000,
		EarlyStoppingEnabled:   true,
		MinDetectionImprovement: 0.05,
		MaxDegradationTolerance: 0.02,
	}
	
	// Create experiment
	experiment, err := its.experimentMgr.CreateExperiment(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	// Start experiment
	err = its.experimentMgr.StartExperiment(ctx, experiment.ID)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}
	
	// Simulate traffic and results
	its.simulateExperimentTraffic(t, experiment, 2000)
	
	// Check experiment status
	experiment, err = its.experimentMgr.GetExperiment(experiment.ID)
	if err != nil {
		t.Fatalf("Failed to get experiment: %v", err)
	}
	
	if experiment.Status != ExperimentStatusRunning {
		t.Errorf("Expected experiment to be running, got %s", experiment.Status)
	}
	
	// Stop experiment
	err = its.experimentMgr.StopExperiment(ctx, experiment.ID, "test_completion")
	if err != nil {
		t.Fatalf("Failed to stop experiment: %v", err)
	}
	
	// Verify results
	if experiment.Results == nil {
		t.Error("Experiment results not generated")
	}
	
	t.Logf("A/B testing workflow completed successfully for experiment %s", experiment.ID)
}

// testDriftDetection tests the drift detection system
func (its *IntegrationTestSuite) testDriftDetection(t *testing.T, ctx context.Context) {
	// Start drift detection
	err := its.driftDetector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start drift detector: %v", err)
	}
	defer its.driftDetector.Stop()
	
	// Establish baselines
	baselineData := its.generateBaselineData(1000)
	err = its.driftDetector.EstablishBaseline("detection_rate", baselineData, time.Now())
	if err != nil {
		t.Fatalf("Failed to establish baseline: %v", err)
	}
	
	// Test with normal data (no drift)
	normalData := its.generateNormalData(100, 0.95, 0.02)
	results, err := its.driftDetector.DetectDrift("detection_rate", normalData)
	if err != nil {
		t.Fatalf("Failed to detect drift: %v", err)
	}
	
	driftDetected := false
	for _, result := range results {
		if result.DriftDetected {
			driftDetected = true
			break
		}
	}
	
	if driftDetected {
		t.Error("False positive drift detection on normal data")
	}
	
	// Test with drifted data
	driftedData := its.generateDriftedData(100, 0.85, 0.05) // Significant drift
	results, err = its.driftDetector.DetectDrift("detection_rate", driftedData)
	if err != nil {
		t.Fatalf("Failed to detect drift: %v", err)
	}
	
	driftDetected = false
	for _, result := range results {
		if result.DriftDetected && result.Severity >= SeverityMedium {
			driftDetected = true
			break
		}
	}
	
	if !driftDetected {
		t.Error("Failed to detect significant drift")
	}
	
	t.Log("Drift detection system working correctly")
}

// testPromotionPipeline tests the promotion pipeline
func (its *IntegrationTestSuite) testPromotionPipeline(t *testing.T, ctx context.Context) {
	// Create mock experiment with winning variant
	experiment := &Experiment{
		ID:     "test_exp_promotion",
		Name:   "Promotion Test Experiment",
		Status: ExperimentStatusCompleted,
		Results: &ExperimentResults{
			Winner: &VariantMetrics{
				VariantID:         "winning_variant",
				SampleSize:        2000,
				DetectionRate:     0.97,
				FalsePositiveRate: 0.02,
				AvgResponseTime:   15 * time.Millisecond,
				UserSatisfaction:  0.92,
				FitnessScore:      0.94,
			},
			Significance: &SignificanceTest{
				PValue:      0.001,
				Significant: true,
				Improvement: 0.15,
				Method:      "welch_test",
			},
			ConfidenceLevel: 0.999,
		},
	}
	
	// Evaluate for promotion
	recommendation, err := its.promotionEngine.EvaluateForPromotion(ctx, experiment)
	if err != nil {
		t.Fatalf("Failed to evaluate promotion: %v", err)
	}
	
	if !recommendation.ShouldPromote {
		t.Error("Should recommend promotion for winning variant")
	}
	
	// Create promotion request
	promotionRequest := CreatePromotionRequest{
		ExperimentID:      experiment.ID,
		WinningVariantID:  "winning_variant",
		RequestedBy:       "integration_test",
		Justification:     "Significant improvement in detection rate",
		Strategy:          RolloutGradual,
		TargetEnvironment: "production",
	}
	
	promotion, err := its.promotionEngine.CreatePromotionRequest(ctx, promotionRequest)
	if err != nil {
		t.Fatalf("Failed to create promotion request: %v", err)
	}
	
	if promotion.Status != PromotionStatusPending {
		t.Errorf("Expected promotion status pending, got %s", promotion.Status)
	}
	
	t.Logf("Promotion pipeline test completed for experiment %s", experiment.ID)
}

// testRolloutManagement tests rollout management
func (its *IntegrationTestSuite) testRolloutManagement(t *testing.T, ctx context.Context) {
	// Create rollout request
	rolloutRequest := StartRolloutRequest{
		DeploymentID: "test_deployment_001",
		PromotionID:  "test_promotion_001",
		Strategy:     RolloutGradual,
		Metadata: map[string]interface{}{
			"test": true,
		},
	}
	
	// Start rollout
	rollout, err := its.rolloutManager.StartRollout(ctx, rolloutRequest)
	if err != nil {
		t.Fatalf("Failed to start rollout: %v", err)
	}
	
	if rollout.Status != RolloutStatus("starting") {
		t.Errorf("Expected rollout status starting, got %s", rollout.Status)
	}
	
	// Wait for rollout to start
	time.Sleep(100 * time.Millisecond)
	
	// Check rollout progress
	rollout, err = its.rolloutManager.GetActiveRollout(rollout.ID)
	if err != nil {
		t.Fatalf("Failed to get active rollout: %v", err)
	}
	
	expectedPhases := 5 // Gradual rollout has 5 phases
	if len(rollout.Phases) != expectedPhases {
		t.Errorf("Expected %d phases, got %d", expectedPhases, len(rollout.Phases))
	}
	
	t.Logf("Rollout management test completed for deployment %s", rolloutRequest.DeploymentID)
}

// testRollbackScenarios tests various rollback scenarios
func (its *IntegrationTestSuite) testRollbackScenarios(t *testing.T, ctx context.Context) {
	// Create rollback plan
	plan, err := its.rollbackManager.CreateRollbackPlan("test_deployment_rollback")
	if err != nil {
		t.Fatalf("Failed to create rollback plan: %v", err)
	}
	
	if plan.Status != RollbackPlanStatusReady {
		t.Errorf("Expected rollback plan to be ready, got %s", plan.Status)
	}
	
	// Validate rollback plan
	err = its.rollbackManager.ValidateRollback(plan.ID)
	if err != nil {
		t.Fatalf("Rollback validation failed: %v", err)
	}
	
	// Execute rollback (in test mode)
	err = its.rollbackManager.ExecuteRollback(ctx, plan.ID)
	if err != nil {
		t.Fatalf("Failed to execute rollback: %v", err)
	}
	
	// Wait for rollback to complete
	time.Sleep(200 * time.Millisecond)
	
	// Check rollback status
	rollback, err := its.rollbackManager.GetActiveRollback(plan.ID)
	if err == nil { // Rollback might be completed and removed from active list
		if rollback.Status != RollbackStatusRunning && rollback.Status != RollbackStatusCompleted {
			t.Errorf("Unexpected rollback status: %s", rollback.Status)
		}
	}
	
	t.Logf("Rollback scenarios test completed for plan %s", plan.ID)
}

// testEndToEndOptimizationCycle tests the complete optimization cycle
func (its *IntegrationTestSuite) testEndToEndOptimizationCycle(t *testing.T, ctx context.Context) {
	t.Log("Starting end-to-end optimization cycle test...")
	
	// Step 1: Generate initial rule variants
	baseRule := Rule{
		ID:   "e2e_base_rule",
		Name: "E2E Base Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "test_pattern",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.75,
	}
	
	objectives := GetDefaultObjectives()
	variants, err := its.optimizer.GenerateRuleVariants(ctx, baseRule, 2, objectives)
	if err != nil {
		t.Fatalf("Failed to generate variants: %v", err)
	}
	
	// Step 2: Create and run A/B test
	experimentConfig := ExperimentConfig{
		Name:       "E2E Optimization Test",
		Objectives: objectives,
		Control:    baseRule,
		Variants:   variants,
		Traffic: TrafficSplit{
			Control:  0.5,
			Variants: map[string]float64{variants[0].ID: 0.5},
		},
		Duration:               1 * time.Hour,
		MinSampleSize:          500,
		EarlyStoppingEnabled:   true,
		MinDetectionImprovement: 0.03,
		MaxDegradationTolerance: 0.05,
	}
	
	experiment, err := its.experimentMgr.CreateExperiment(ctx, experimentConfig)
	if err != nil {
		t.Fatalf("Failed to create E2E experiment: %v", err)
	}
	
	err = its.experimentMgr.StartExperiment(ctx, experiment.ID)
	if err != nil {
		t.Fatalf("Failed to start E2E experiment: %v", err)
	}
	
	// Step 3: Simulate experiment completion
	its.simulateExperimentTraffic(t, experiment, 1000)
	
	err = its.experimentMgr.StopExperiment(ctx, experiment.ID, "e2e_test_completion")
	if err != nil {
		t.Fatalf("Failed to stop E2E experiment: %v", err)
	}
	
	// Step 4: Evaluate for promotion
	experiment, _ = its.experimentMgr.GetExperiment(experiment.ID)
	recommendation, err := its.promotionEngine.EvaluateForPromotion(ctx, experiment)
	if err != nil {
		t.Fatalf("Failed to evaluate E2E promotion: %v", err)
	}
	
	// Step 5: If recommended, create promotion
	if recommendation.ShouldPromote {
		promotionRequest := CreatePromotionRequest{
			ExperimentID:      experiment.ID,
			WinningVariantID:  recommendation.WinningVariantID,
			RequestedBy:       "e2e_test",
			Justification:     "End-to-end test promotion",
			Strategy:          RolloutCanary,
			TargetEnvironment: "production",
		}
		
		_, err = its.promotionEngine.CreatePromotionRequest(ctx, promotionRequest)
		if err != nil {
			t.Fatalf("Failed to create E2E promotion: %v", err)
		}
	}
	
	t.Log("End-to-end optimization cycle completed successfully")
}

// testPerformanceAndScalability tests system performance under load
func (its *IntegrationTestSuite) testPerformanceAndScalability(t *testing.T, ctx context.Context) {
	t.Log("Starting performance and scalability tests...")
	
	// Test 1: Rule generation performance
	startTime := time.Now()
	baseRule := Rule{
		ID:   "perf_test_rule",
		Name: "Performance Test Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "performance_test",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.80,
	}
	
	objectives := GetDefaultObjectives()
	for i := 0; i < 100; i++ {
		_, err := its.optimizer.GenerateRuleVariants(ctx, baseRule, 5, objectives)
		if err != nil {
			t.Fatalf("Performance test failed at iteration %d: %v", i, err)
		}
	}
	ruleGenDuration := time.Since(startTime)
	t.Logf("Generated 500 rule variants in %v (avg: %v per variant)", 
		ruleGenDuration, ruleGenDuration/500)
	
	// Test 2: Concurrent experiment creation
	startTime = time.Now()
	var wg sync.WaitGroup
	concurrentExperiments := 10
	
	for i := 0; i < concurrentExperiments; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			config := ExperimentConfig{
				Name:       fmt.Sprintf("Concurrent Test %d", id),
				Objectives: objectives,
				Control:    baseRule,
				Variants:   []Rule{baseRule}, // Simplified for test
				Traffic: TrafficSplit{
					Control:  0.5,
					Variants: map[string]float64{"variant": 0.5},
				},
				Duration:      1 * time.Hour,
				MinSampleSize: 100,
			}
			
			experiment, err := its.experimentMgr.CreateExperiment(ctx, config)
			if err != nil {
				t.Errorf("Failed to create concurrent experiment %d: %v", id, err)
				return
			}
			
			// Clean up
			its.experimentMgr.StopExperiment(ctx, experiment.ID, "perf_test_cleanup")
		}(i)
	}
	
	wg.Wait()
	concurrentDuration := time.Since(startTime)
	t.Logf("Created %d concurrent experiments in %v", concurrentExperiments, concurrentDuration)
	
	// Test 3: Drift detection performance
	startTime = time.Now()
	baselineData := its.generateBaselineData(1000)
	err := its.driftDetector.EstablishBaseline("perf_test_metric", baselineData, time.Now())
	if err != nil {
		t.Fatalf("Failed to establish performance test baseline: %v", err)
	}
	
	// Run drift detection 100 times
	for i := 0; i < 100; i++ {
		testData := its.generateNormalData(100, 0.95, 0.02)
		_, err := its.driftDetector.DetectDrift("perf_test_metric", testData)
		if err != nil {
			t.Fatalf("Drift detection performance test failed at iteration %d: %v", i, err)
		}
	}
	driftDetectionDuration := time.Since(startTime)
	t.Logf("Performed 100 drift detections in %v (avg: %v per detection)", 
		driftDetectionDuration, driftDetectionDuration/100)
	
	// Performance benchmarks
	if ruleGenDuration/500 > 10*time.Millisecond {
		t.Errorf("Rule generation too slow: %v per variant (expected < 10ms)", ruleGenDuration/500)
	}
	
	if concurrentDuration > 5*time.Second {
		t.Errorf("Concurrent experiment creation too slow: %v (expected < 5s)", concurrentDuration)
	}
	
	if driftDetectionDuration/100 > 50*time.Millisecond {
		t.Errorf("Drift detection too slow: %v per detection (expected < 50ms)", driftDetectionDuration/100)
	}
	
	t.Log("Performance and scalability tests completed")
}

// testErrorHandlingAndRecovery tests error handling and recovery mechanisms
func (its *IntegrationTestSuite) testErrorHandlingAndRecovery(t *testing.T, ctx context.Context) {
	t.Log("Testing error handling and recovery...")
	
	// Test invalid experiment configuration
	invalidConfig := ExperimentConfig{
		Name:       "", // Invalid: empty name
		Objectives: []Objective{},
		Control:    Rule{},
		Variants:   []Rule{},
		Traffic: TrafficSplit{
			Control:  1.5, // Invalid: > 1.0
			Variants: map[string]float64{},
		},
	}
	
	_, err := its.experimentMgr.CreateExperiment(ctx, invalidConfig)
	if err == nil {
		t.Error("Expected error for invalid experiment configuration")
	}
	
	// Test drift detection with insufficient data
	err = its.driftDetector.EstablishBaseline("insufficient_data", []float64{1.0}, time.Now())
	if err == nil {
		t.Error("Expected error for insufficient baseline data")
	}
	
	// Test rollback with invalid deployment
	_, err = its.rollbackManager.CreateRollbackPlan("nonexistent_deployment")
	if err != nil {
		// This should work in our mock implementation, but the plan might not be ready
		t.Logf("Rollback plan creation handled appropriately: %v", err)
	}
	
	// Test promotion with incomplete experiment
	incompleteExperiment := &Experiment{
		ID:     "incomplete_exp",
		Status: ExperimentStatusRunning, // Still running
	}
	
	recommendation, err := its.promotionEngine.EvaluateForPromotion(ctx, incompleteExperiment)
	if err != nil {
		t.Fatalf("Unexpected error evaluating incomplete experiment: %v", err)
	}
	
	if recommendation.ShouldPromote {
		t.Error("Should not recommend promotion for incomplete experiment")
	}
	
	t.Log("Error handling and recovery tests completed")
}

// testConcurrentOperations tests concurrent operations
func (its *IntegrationTestSuite) testConcurrentOperations(t *testing.T, ctx context.Context) {
	t.Log("Testing concurrent operations...")
	
	var wg sync.WaitGroup
	
	// Test concurrent drift detection
	baselineData := its.generateBaselineData(1000)
	err := its.driftDetector.EstablishBaseline("concurrent_test", baselineData, time.Now())
	if err != nil {
		t.Fatalf("Failed to establish baseline for concurrent test: %v", err)
	}
	
	// Run concurrent drift detections
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			testData := its.generateNormalData(100, 0.95, 0.02)
			_, err := its.driftDetector.DetectDrift("concurrent_test", testData)
			if err != nil {
				t.Errorf("Concurrent drift detection %d failed: %v", id, err)
			}
		}(i)
	}
	
	// Test concurrent rule generation
	baseRule := Rule{
		ID:   "concurrent_rule",
		Name: "Concurrent Test Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "concurrent_test",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.80,
	}
	
	objectives := GetDefaultObjectives()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			_, err := its.optimizer.GenerateRuleVariants(ctx, baseRule, 3, objectives)
			if err != nil {
				t.Errorf("Concurrent rule generation %d failed: %v", id, err)
			}
		}(i)
	}
	
	wg.Wait()
	t.Log("Concurrent operations tests completed")
}

// Helper methods for test data generation

func (its *IntegrationTestSuite) generateBaselineData(size int) []float64 {
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		// Generate data around 0.95 with small variance
		data[i] = 0.95 + (rand.Float64()-0.5)*0.02
	}
	return data
}

func (its *IntegrationTestSuite) generateNormalData(size int, mean, stddev float64) []float64 {
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = mean + (rand.Float64()-0.5)*stddev*2
	}
	return data
}

func (its *IntegrationTestSuite) generateDriftedData(size int, newMean, stddev float64) []float64 {
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = newMean + (rand.Float64()-0.5)*stddev*2
	}
	return data
}

func (its *IntegrationTestSuite) simulateExperimentTraffic(t *testing.T, experiment *Experiment, totalRequests int) {
	controlRequests := int(float64(totalRequests) * experiment.Traffic.Control)
	
	// Simulate control traffic
	for i := 0; i < controlRequests; i++ {
		result := ExperimentResult{
			Detection:        rand.Float64() < 0.95, // 95% detection rate
			FalsePositive:    rand.Float64() < 0.03, // 3% false positive rate
			ResponseTime:     time.Duration(12+rand.Intn(8)) * time.Millisecond,
			UserSatisfaction: 0.85 + rand.Float64()*0.1,
		}
		
		err := its.experimentMgr.RecordResult(context.Background(), experiment.ID, "control", result)
		if err != nil {
			t.Logf("Warning: Failed to record control result: %v", err)
		}
	}
	
	// Simulate variant traffic
	for variantID, trafficSplit := range experiment.Traffic.Variants {
		variantRequests := int(float64(totalRequests) * trafficSplit)
		
		for i := 0; i < variantRequests; i++ {
			result := ExperimentResult{
				Detection:        rand.Float64() < 0.97, // Slightly better detection
				FalsePositive:    rand.Float64() < 0.025, // Slightly fewer false positives
				ResponseTime:     time.Duration(11+rand.Intn(6)) * time.Millisecond,
				UserSatisfaction: 0.88 + rand.Float64()*0.08,
			}
			
			err := its.experimentMgr.RecordResult(context.Background(), experiment.ID, variantID, result)
			if err != nil {
				t.Logf("Warning: Failed to record variant result: %v", err)
			}
		}
	}
}