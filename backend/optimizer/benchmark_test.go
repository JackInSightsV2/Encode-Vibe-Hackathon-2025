package optimizer

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// BenchmarkRuleGeneration benchmarks rule generation performance
func BenchmarkRuleGeneration(b *testing.B) {
	optimizer := NewOptimizerClient(GetDefaultOptimizerConfig())
	objectives := GetDefaultObjectives()
	
	baseRule := Rule{
		ID:   "benchmark_rule",
		Name: "Benchmark Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "(?i)(benchmark|test|performance)",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.80,
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.GenerateRuleVariants(ctx, baseRule, 3, objectives)
		if err != nil {
			b.Fatalf("Failed to generate rule variants: %v", err)
		}
	}
}

// BenchmarkMutationEngine benchmarks the mutation engine
func BenchmarkMutationEngine(b *testing.B) {
	mutationEngine := NewMutationEngine(GetDefaultMutationConfig())
	
	baseRule := Rule{
		ID:   "mutation_benchmark",
		Name: "Mutation Benchmark Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "(?i)(mutation|benchmark)",
			"flags":   "i",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.85,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mutationEngine.MutateRule(baseRule, 5)
		if err != nil {
			b.Fatalf("Failed to mutate rule: %v", err)
		}
	}
}

// BenchmarkExperimentCreation benchmarks experiment creation
func BenchmarkExperimentCreation(b *testing.B) {
	experimentMgr := NewExperimentManager(GetDefaultExperimentManagerConfig())
	
	baseRule := Rule{
		ID:   "exp_benchmark_control",
		Name: "Experiment Benchmark Control",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "benchmark_control",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.80,
	}
	
	variantRule := Rule{
		ID:   "exp_benchmark_variant",
		Name: "Experiment Benchmark Variant",
		Type: SemanticRule,
		Parameters: map[string]interface{}{
			"model": "benchmark_model",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.85,
	}
	
	config := ExperimentConfig{
		Name:       "Benchmark Experiment",
		Objectives: GetDefaultObjectives(),
		Control:    baseRule,
		Variants:   []Rule{variantRule},
		Traffic: TrafficSplit{
			Control:  0.5,
			Variants: map[string]float64{variantRule.ID: 0.5},
		},
		Duration:      24 * time.Hour,
		MinSampleSize: 1000,
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		experiment, err := experimentMgr.CreateExperiment(ctx, config)
		if err != nil {
			b.Fatalf("Failed to create experiment: %v", err)
		}
		
		// Clean up
		experimentMgr.StopExperiment(ctx, experiment.ID, "benchmark_cleanup")
	}
}

// BenchmarkDriftDetection benchmarks drift detection algorithms
func BenchmarkDriftDetection(b *testing.B) {
	driftDetector := NewDriftDetector(GetDefaultDriftDetectionConfig())
	
	// Establish baseline
	baselineData := make([]float64, 1000)
	for i := 0; i < 1000; i++ {
		baselineData[i] = 0.95 + (rand.Float64()-0.5)*0.02
	}
	
	err := driftDetector.EstablishBaseline("benchmark_metric", baselineData, time.Now())
	if err != nil {
		b.Fatalf("Failed to establish baseline: %v", err)
	}
	
	// Test data
	testData := make([]float64, 100)
	for i := 0; i < 100; i++ {
		testData[i] = 0.95 + (rand.Float64()-0.5)*0.02
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := driftDetector.DetectDrift("benchmark_metric", testData)
		if err != nil {
			b.Fatalf("Failed to detect drift: %v", err)
		}
	}
}

// BenchmarkStatisticalTests benchmarks statistical significance tests
func BenchmarkStatisticalTests(b *testing.B) {
	statsEngine := NewStatisticsEngine()
	
	controlMetrics := &VariantMetrics{
		VariantID:         "control",
		SampleSize:        1000,
		DetectionRate:     0.95,
		FalsePositiveRate: 0.03,
		AvgResponseTime:   15 * time.Millisecond,
		UserSatisfaction:  0.85,
		FitnessScore:      0.88,
	}
	
	variantMetrics := &VariantMetrics{
		VariantID:         "variant",
		SampleSize:        1000,
		DetectionRate:     0.97,
		FalsePositiveRate: 0.025,
		AvgResponseTime:   14 * time.Millisecond,
		UserSatisfaction:  0.88,
		FitnessScore:      0.92,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = statsEngine.CalculateSignificance(controlMetrics, variantMetrics)
	}
}

// BenchmarkTrafficRouting benchmarks traffic routing decisions
func BenchmarkTrafficRouting(b *testing.B) {
	trafficRouter := NewTrafficRouter()
	
	// Create a mock experiment
	experiment := &Experiment{
		ID:   "routing_benchmark",
		Name: "Routing Benchmark",
		Control: Rule{
			ID:   "control_rule",
			Name: "Control Rule",
			Type: RegexRule,
		},
		Variants: []Rule{
			{
				ID:   "variant_rule",
				Name: "Variant Rule",
				Type: SemanticRule,
			},
		},
		Traffic: TrafficSplit{
			Control:  0.6,
			Variants: map[string]float64{"variant_rule": 0.4},
		},
	}
	
	err := trafficRouter.RegisterExperiment(experiment)
	if err != nil {
		b.Fatalf("Failed to register experiment: %v", err)
	}
	
	err = trafficRouter.StartTrafficAllocation(experiment.ID)
	if err != nil {
		b.Fatalf("Failed to start traffic allocation: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := fmt.Sprintf("user_%d", i%1000) // Simulate 1000 unique users
		requestID := fmt.Sprintf("req_%d", i)
		
		_, err := trafficRouter.RouteRequest(requestID, userID)
		if err != nil {
			b.Fatalf("Failed to route request: %v", err)
		}
	}
}

// BenchmarkRolloutExecution benchmarks rollout execution
func BenchmarkRolloutExecution(b *testing.B) {
	rolloutManager := NewRolloutManager(GetDefaultRolloutConfig())
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := StartRolloutRequest{
			DeploymentID: fmt.Sprintf("benchmark_deployment_%d", i),
			PromotionID:  fmt.Sprintf("benchmark_promotion_%d", i),
			Strategy:     RolloutGradual,
		}
		
		rollout, err := rolloutManager.StartRollout(ctx, request)
		if err != nil {
			b.Fatalf("Failed to start rollout: %v", err)
		}
		
		// Simulate quick completion for benchmark
		time.Sleep(1 * time.Millisecond)
		
		// Clean up
		rolloutManager.CancelRollback(rollout.ID)
	}
}

// BenchmarkFitnessCalculation benchmarks fitness score calculation
func BenchmarkFitnessCalculation(b *testing.B) {
	objectives := GetDefaultObjectives()
	objectiveSet, err := NewObjectiveSet(objectives)
	if err != nil {
		b.Fatalf("Failed to create objective set: %v", err)
	}
	
	metrics := map[string]float64{
		"detection_rate":       0.95,
		"false_positive_rate":  0.03,
		"response_time_ms":     15.0,
		"user_satisfaction":    0.85,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = objectiveSet.CalculateScore(metrics)
	}
}

// BenchmarkCombinationEngine benchmarks rule combination
func BenchmarkCombinationEngine(b *testing.B) {
	combinationEngine := NewCombinationEngine(GetDefaultCombinationConfig())
	
	rule1 := Rule{
		ID:   "combo_rule_1",
		Name: "Combination Rule 1",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "pattern1",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.85,
	}
	
	rule2 := Rule{
		ID:   "combo_rule_2",
		Name: "Combination Rule 2",
		Type: SemanticRule,
		Parameters: map[string]interface{}{
			"model": "model2",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.88,
	}
	
	rules := []Rule{rule1, rule2}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := combinationEngine.CombineRules(rules, WeightedCombination)
		if err != nil {
			b.Fatalf("Failed to combine rules: %v", err)
		}
	}
}

// BenchmarkMemoryUsage benchmarks memory usage of the optimizer
func BenchmarkMemoryUsage(b *testing.B) {
	var m1, m2 runtime.MemStats
	
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	optimizer := NewOptimizerClient(GetDefaultOptimizerConfig())
	experimentMgr := NewExperimentManager(GetDefaultExperimentManagerConfig())
	driftDetector := NewDriftDetector(GetDefaultDriftDetectionConfig())
	
	// Create multiple experiments and rules
	for i := 0; i < 100; i++ {
		baseRule := Rule{
			ID:   fmt.Sprintf("memory_test_rule_%d", i),
			Name: fmt.Sprintf("Memory Test Rule %d", i),
			Type: RegexRule,
			Parameters: map[string]interface{}{
				"pattern": fmt.Sprintf("pattern_%d", i),
			},
			Enabled:  true,
			Priority: 10,
			Fitness:  0.80 + rand.Float64()*0.2,
		}
		
		config := ExperimentConfig{
			Name:       fmt.Sprintf("Memory Test Experiment %d", i),
			Objectives: GetDefaultObjectives(),
			Control:    baseRule,
			Variants:   []Rule{baseRule},
			Traffic: TrafficSplit{
				Control:  0.5,
				Variants: map[string]float64{"variant": 0.5},
			},
			Duration:      1 * time.Hour,
			MinSampleSize: 100,
		}
		
		_, err := experimentMgr.CreateExperiment(context.Background(), config)
		if err != nil {
			b.Fatalf("Failed to create memory test experiment: %v", err)
		}
		
		// Generate some rule variants
		_, err = optimizer.GenerateRuleVariants(context.Background(), baseRule, 3, GetDefaultObjectives())
		if err != nil {
			b.Fatalf("Failed to generate variants for memory test: %v", err)
		}
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	memoryUsed := m2.TotalAlloc - m1.TotalAlloc
	b.Logf("Memory used for 100 experiments and 300 rule variants: %d bytes", memoryUsed)
	
	// Set a reasonable memory usage threshold (e.g., 50MB)
	const maxMemoryUsage = 50 * 1024 * 1024 // 50MB
	if memoryUsed > maxMemoryUsage {
		b.Errorf("Memory usage too high: %d bytes (expected < %d bytes)", memoryUsed, maxMemoryUsage)
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	optimizer := NewOptimizerClient(GetDefaultOptimizerConfig())
	objectives := GetDefaultObjectives()
	
	baseRule := Rule{
		ID:   "concurrent_benchmark",
		Name: "Concurrent Benchmark Rule",
		Type: RegexRule,
		Parameters: map[string]interface{}{
			"pattern": "concurrent_test",
		},
		Enabled:  true,
		Priority: 10,
		Fitness:  0.80,
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := optimizer.GenerateRuleVariants(ctx, baseRule, 2, objectives)
			if err != nil {
				b.Fatalf("Failed to generate rule variants in concurrent benchmark: %v", err)
			}
		}
	})
}

// TestIntegrationSuite runs the complete integration test suite
func TestIntegrationSuite(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite := NewIntegrationTestSuite()
	suite.RunFullIntegrationTest(t)
}

// Helper function to generate random strings for tests
func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// BenchmarkEndToEndWorkflow benchmarks the complete end-to-end workflow
func BenchmarkEndToEndWorkflow(b *testing.B) {
	suite := NewIntegrationTestSuite()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate rule variants
		baseRule := Rule{
			ID:   fmt.Sprintf("e2e_rule_%d", i),
			Name: fmt.Sprintf("E2E Rule %d", i),
			Type: RegexRule,
			Parameters: map[string]interface{}{
				"pattern": fmt.Sprintf("e2e_pattern_%d", i),
			},
			Enabled:  true,
			Priority: 10,
			Fitness:  0.80,
		}
		
		objectives := GetDefaultObjectives()
		variants, err := suite.optimizer.GenerateRuleVariants(ctx, baseRule, 2, objectives)
		if err != nil {
			b.Fatalf("Failed to generate variants in E2E benchmark: %v", err)
		}
		
		// Create experiment
		config := ExperimentConfig{
			Name:       fmt.Sprintf("E2E Benchmark %d", i),
			Objectives: objectives,
			Control:    baseRule,
			Variants:   variants,
			Traffic: TrafficSplit{
				Control:  0.5,
				Variants: map[string]float64{variants[0].ID: 0.5},
			},
			Duration:      1 * time.Hour,
			MinSampleSize: 100,
		}
		
		experiment, err := suite.experimentMgr.CreateExperiment(ctx, config)
		if err != nil {
			b.Fatalf("Failed to create experiment in E2E benchmark: %v", err)
		}
		
		// Start experiment
		err = suite.experimentMgr.StartExperiment(ctx, experiment.ID)
		if err != nil {
			b.Fatalf("Failed to start experiment in E2E benchmark: %v", err)
		}
		
		// Stop experiment (simulate completion)
		err = suite.experimentMgr.StopExperiment(ctx, experiment.ID, "benchmark_completion")
		if err != nil {
			b.Fatalf("Failed to stop experiment in E2E benchmark: %v", err)
		}
		
		// Evaluate for promotion
		_, err = suite.promotionEngine.EvaluateForPromotion(ctx, experiment)
		if err != nil {
			b.Fatalf("Failed to evaluate promotion in E2E benchmark: %v", err)
		}
	}
}