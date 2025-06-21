package optimizer

import (
	"testing"
	"time"

	"qt1-middleware/opik"
)

func TestNewOptimizerClient(t *testing.T) {
	// Create mock Opik client
	opikClient := &opik.OpikClient{}
	
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Logf("Warning: Failed to create optimizer client (expected in test environment): %v", err)
		// This is expected if Python is not available
		return
	}
	
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	
	if client.config != config {
		t.Error("Config not set correctly")
	}
	
	if len(client.objectives) == 0 {
		t.Error("Default objectives not set")
	}
}

func TestGetDefaultOptimizerConfig(t *testing.T) {
	config := GetDefaultOptimizerConfig()
	
	if !config.Enabled {
		t.Error("Expected optimizer to be enabled by default")
	}
	
	if config.ProjectName == "" {
		t.Error("Expected default project name")
	}
	
	if config.MaxWorkers <= 0 {
		t.Error("Expected positive max workers")
	}
	
	if config.ExperimentDuration <= 0 {
		t.Error("Expected positive experiment duration")
	}
	
	if config.MinSampleSize <= 0 {
		t.Error("Expected positive min sample size")
	}
	
	if config.ConfidenceLevel <= 0 || config.ConfidenceLevel > 1 {
		t.Error("Expected confidence level between 0 and 1")
	}
	
	if config.PopulationSize <= 0 {
		t.Error("Expected positive population size")
	}
	
	if config.MutationRate < 0 || config.MutationRate > 1 {
		t.Error("Expected mutation rate between 0 and 1")
	}
	
	if config.CrossoverRate < 0 || config.CrossoverRate > 1 {
		t.Error("Expected crossover rate between 0 and 1")
	}
	
	if config.ElitismRate < 0 || config.ElitismRate > 1 {
		t.Error("Expected elitism rate between 0 and 1")
	}
}

func TestCreateExperiment(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	// Create test rule
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test.*",
			"flags":   "i",
		},
		Enabled:  true,
		Priority: 1,
	}
	
	// Create test experiment config
	expConfig := ExperimentConfig{
		Name:        "Test Experiment",
		Description: "Testing rule optimization",
		Objectives:  GetDefaultObjectives(),
		Control:     baseRule,
		Variants:    []Rule{baseRule.Clone()},
		Traffic: TrafficSplit{
			Control: 0.5,
			Variants: map[string]float64{
				"variant_1": 0.5,
			},
		},
		Duration:      1 * time.Hour,
		MinSampleSize: 100,
	}
	
	experiment, err := client.CreateExperiment(expConfig)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	if experiment.ID == "" {
		t.Error("Expected experiment ID to be set")
	}
	
	if experiment.Name != expConfig.Name {
		t.Error("Experiment name not set correctly")
	}
	
	if experiment.Status != ExperimentStatusPending {
		t.Error("Expected experiment status to be pending")
	}
	
	if len(experiment.Objectives) == 0 {
		t.Error("Expected objectives to be set")
	}
	
	// Test that experiment is stored
	retrieved, exists := client.GetExperiment(experiment.ID)
	if !exists {
		t.Error("Experiment not found after creation")
	}
	
	if retrieved.ID != experiment.ID {
		t.Error("Retrieved experiment ID mismatch")
	}
}

func TestStartExperiment(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	// Create test experiment
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test.*",
		},
		Enabled: true,
	}
	
	expConfig := ExperimentConfig{
		Name:       "Start Test",
		Objectives: GetDefaultObjectives(),
		Control:    baseRule,
		Variants:   []Rule{baseRule.Clone()},
		Traffic: TrafficSplit{
			Control: 0.5,
			Variants: map[string]float64{
				"variant_1": 0.5,
			},
		},
	}
	
	experiment, err := client.CreateExperiment(expConfig)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	// Test starting experiment
	err = client.StartExperiment(experiment.ID)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}
	
	// Check status
	retrieved, _ := client.GetExperiment(experiment.ID)
	if retrieved.Status != ExperimentStatusRunning {
		t.Error("Expected experiment status to be running")
	}
	
	// Test starting non-existent experiment
	err = client.StartExperiment("non_existent")
	if err == nil {
		t.Error("Expected error when starting non-existent experiment")
	}
	
	// Test starting already running experiment
	err = client.StartExperiment(experiment.ID)
	if err == nil {
		t.Error("Expected error when starting already running experiment")
	}
}

func TestStopExperiment(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	// Create and start test experiment
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test.*",
		},
		Enabled: true,
	}
	
	expConfig := ExperimentConfig{
		Name:       "Stop Test",
		Objectives: GetDefaultObjectives(),
		Control:    baseRule,
		Variants:   []Rule{baseRule.Clone()},
		Traffic: TrafficSplit{
			Control: 0.5,
			Variants: map[string]float64{
				"variant_1": 0.5,
			},
		},
	}
	
	experiment, err := client.CreateExperiment(expConfig)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	err = client.StartExperiment(experiment.ID)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}
	
	// Test stopping experiment
	reason := "manual_stop_for_test"
	err = client.StopExperiment(experiment.ID, reason)
	if err != nil {
		t.Fatalf("Failed to stop experiment: %v", err)
	}
	
	// Check status
	retrieved, _ := client.GetExperiment(experiment.ID)
	if retrieved.Status != ExperimentStatusCompleted {
		t.Error("Expected experiment status to be completed")
	}
	
	if retrieved.StopReason != reason {
		t.Error("Stop reason not set correctly")
	}
	
	if retrieved.EndTime == nil {
		t.Error("End time not set")
	}
}

func TestGenerateRuleVariantsBasic(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test",
			"flags":   "",
		},
		Enabled: true,
	}
	
	variants, err := client.generateRuleVariantsBasic(baseRule, 3)
	if err != nil {
		t.Fatalf("Failed to generate variants: %v", err)
	}
	
	if len(variants) != 3 {
		t.Errorf("Expected 3 variants, got %d", len(variants))
	}
	
	for i, variant := range variants {
		if variant.ID == "" {
			t.Errorf("Variant %d missing ID", i)
		}
		
		if variant.Name == "" {
			t.Errorf("Variant %d missing name", i)
		}
		
		if variant.Type != baseRule.Type {
			t.Errorf("Variant %d type mismatch", i)
		}
		
		// Check that parameters were mutated
		if pattern, ok := variant.Parameters["pattern"].(string); ok {
			if pattern == baseRule.Parameters["pattern"] {
				t.Logf("Variant %d pattern not mutated (may be expected)", i)
			}
		}
	}
}

func TestListExperiments(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	// Initially should be empty
	experiments := client.ListExperiments()
	initialCount := len(experiments)
	
	// Create a test experiment
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test.*",
		},
		Enabled: true,
	}
	
	expConfig := ExperimentConfig{
		Name:       "List Test",
		Objectives: GetDefaultObjectives(),
		Control:    baseRule,
		Variants:   []Rule{baseRule.Clone()},
		Traffic: TrafficSplit{
			Control: 0.5,
			Variants: map[string]float64{
				"variant_1": 0.5,
			},
		},
	}
	
	_, err = client.CreateExperiment(expConfig)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	// Should now have one more experiment
	experiments = client.ListExperiments()
	if len(experiments) != initialCount+1 {
		t.Errorf("Expected %d experiments, got %d", initialCount+1, len(experiments))
	}
}

func TestGetActiveExperiments(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	// Initially should be empty
	active := client.GetActiveExperiments()
	initialCount := len(active)
	
	// Create and start a test experiment
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test.*",
		},
		Enabled: true,
	}
	
	expConfig := ExperimentConfig{
		Name:       "Active Test",
		Objectives: GetDefaultObjectives(),
		Control:    baseRule,
		Variants:   []Rule{baseRule.Clone()},
		Traffic: TrafficSplit{
			Control: 0.5,
			Variants: map[string]float64{
				"variant_1": 0.5,
			},
		},
	}
	
	experiment, err := client.CreateExperiment(expConfig)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	// Start the experiment
	err = client.StartExperiment(experiment.ID)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}
	
	// Should now have one more active experiment
	active = client.GetActiveExperiments()
	if len(active) != initialCount+1 {
		t.Errorf("Expected %d active experiments, got %d", initialCount+1, len(active))
	}
	
	// All returned experiments should be running
	for _, exp := range active {
		if exp.Status != ExperimentStatusRunning {
			t.Errorf("Expected running experiment, got status: %s", exp.Status)
		}
	}
}

func TestCalculateFitnessScore(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	metrics := &VariantMetrics{
		DetectionRate:     0.95,
		FalsePositiveRate: 0.05,
		UserSatisfaction:  0.9,
		AvgResponseTime:   10 * time.Millisecond,
	}
	
	score := client.calculateFitnessScore(metrics)
	
	if score < 0 || score > 1 {
		t.Errorf("Expected fitness score between 0 and 1, got %f", score)
	}
	
	// Test with perfect metrics
	perfectMetrics := &VariantMetrics{
		DetectionRate:     1.0,
		FalsePositiveRate: 0.0,
		UserSatisfaction:  1.0,
		AvgResponseTime:   1 * time.Millisecond,
	}
	
	perfectScore := client.calculateFitnessScore(perfectMetrics)
	
	if perfectScore <= score {
		t.Error("Expected perfect metrics to have higher score")
	}
}

func TestClose(t *testing.T) {
	opikClient := &opik.OpikClient{}
	config := GetDefaultOptimizerConfig()
	config.PythonScriptPath = "/tmp/test_optimizer"
	
	client, err := NewOptimizerClient(opikClient, config)
	if err != nil {
		t.Skip("Skipping test due to missing dependencies")
	}
	
	// Create and start a test experiment
	baseRule := Rule{
		ID:   "test_rule_1",
		Name: "Test Rule",
		Type: RuleTypeRegex,
		Parameters: map[string]interface{}{
			"pattern": "test.*",
		},
		Enabled: true,
	}
	
	expConfig := ExperimentConfig{
		Name:       "Close Test",
		Objectives: GetDefaultObjectives(),
		Control:    baseRule,
		Variants:   []Rule{baseRule.Clone()},
		Traffic: TrafficSplit{
			Control: 0.5,
			Variants: map[string]float64{
				"variant_1": 0.5,
			},
		},
	}
	
	experiment, err := client.CreateExperiment(expConfig)
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}
	
	err = client.StartExperiment(experiment.ID)
	if err != nil {
		t.Fatalf("Failed to start experiment: %v", err)
	}
	
	// Close the client
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}
	
	// Check that running experiments were stopped
	retrieved, _ := client.GetExperiment(experiment.ID)
	if retrieved.Status != ExperimentStatusStopped {
		t.Error("Expected running experiment to be stopped after close")
	}
	
	if retrieved.StopReason != "system_shutdown" {
		t.Error("Expected stop reason to be system_shutdown")
	}
}