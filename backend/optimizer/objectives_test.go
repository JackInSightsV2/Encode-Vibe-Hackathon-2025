package optimizer

import (
	"testing"
)

func TestGetDefaultObjectives(t *testing.T) {
	objectives := GetDefaultObjectives()
	
	if len(objectives) == 0 {
		t.Fatal("Expected default objectives to be non-empty")
	}
	
	totalWeight := 0.0
	for _, obj := range objectives {
		if obj.Name == "" {
			t.Error("Objective name cannot be empty")
		}
		
		if obj.Weight <= 0 {
			t.Errorf("Objective %s weight must be positive, got %f", obj.Name, obj.Weight)
		}
		
		if obj.Target < 0 {
			t.Errorf("Objective %s target must be non-negative, got %f", obj.Name, obj.Target)
		}
		
		if obj.Type != Minimize && obj.Type != Maximize {
			t.Errorf("Objective %s has invalid type: %s", obj.Name, obj.Type)
		}
		
		totalWeight += obj.Weight
	}
	
	// Check that weights sum to approximately 1.0
	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("Objective weights should sum to 1.0, got %f", totalWeight)
	}
}

func TestGetSecurityFocusedObjectives(t *testing.T) {
	objectives := GetSecurityFocusedObjectives()
	
	if len(objectives) == 0 {
		t.Fatal("Expected security focused objectives to be non-empty")
	}
	
	// Security focused should prioritize detection rate
	detectionFound := false
	maxWeight := 0.0
	
	for _, obj := range objectives {
		if obj.Name == "detection_rate" {
			detectionFound = true
			if obj.Weight < 0.4 { // Should be at least 40% for security focus
				t.Errorf("Detection rate weight too low for security focus: %f", obj.Weight)
			}
		}
		if obj.Weight > maxWeight {
			maxWeight = obj.Weight
		}
	}
	
	if !detectionFound {
		t.Error("Security focused objectives should include detection_rate")
	}
}

func TestGetPerformanceFocusedObjectives(t *testing.T) {
	objectives := GetPerformanceFocusedObjectives()
	
	if len(objectives) == 0 {
		t.Fatal("Expected performance focused objectives to be non-empty")
	}
	
	// Performance focused should prioritize response time
	responseTimeFound := false
	
	for _, obj := range objectives {
		if obj.Name == "response_time_ms" {
			responseTimeFound = true
			if obj.Weight < 0.3 { // Should be significant for performance focus
				t.Errorf("Response time weight too low for performance focus: %f", obj.Weight)
			}
		}
	}
	
	if !responseTimeFound {
		t.Error("Performance focused objectives should include response_time_ms")
	}
}

func TestGetBalancedObjectives(t *testing.T) {
	objectives := GetBalancedObjectives()
	
	if len(objectives) == 0 {
		t.Fatal("Expected balanced objectives to be non-empty")
	}
	
	// Balanced should have relatively equal weights
	for _, obj := range objectives {
		if obj.Weight < 0.15 || obj.Weight > 0.35 {
			t.Errorf("Balanced objective %s weight should be between 0.15-0.35, got %f", obj.Name, obj.Weight)
		}
	}
}

func TestNewObjectiveSet(t *testing.T) {
	// Test valid objective set
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create valid objective set: %v", err)
	}
	
	if len(objSet.Objectives) != len(objectives) {
		t.Error("Objective set length mismatch")
	}
	
	// Test empty objectives
	_, err = NewObjectiveSet([]Objective{})
	if err == nil {
		t.Error("Expected error for empty objectives")
	}
	
	// Test duplicate names
	duplicateObjs := []Objective{
		{Name: "test", Weight: 0.5, Target: 0.9, Type: Maximize},
		{Name: "test", Weight: 0.5, Target: 0.9, Type: Maximize},
	}
	_, err = NewObjectiveSet(duplicateObjs)
	if err == nil {
		t.Error("Expected error for duplicate objective names")
	}
	
	// Test invalid weights
	invalidWeights := []Objective{
		{Name: "test1", Weight: 0.7, Target: 0.9, Type: Maximize},
		{Name: "test2", Weight: 0.5, Target: 0.9, Type: Maximize}, // Sum > 1.0
	}
	_, err = NewObjectiveSet(invalidWeights)
	if err == nil {
		t.Error("Expected error for weights not summing to 1.0")
	}
	
	// Test negative weight
	negativeWeight := []Objective{
		{Name: "test", Weight: -0.1, Target: 0.9, Type: Maximize},
	}
	_, err = NewObjectiveSet(negativeWeight)
	if err == nil {
		t.Error("Expected error for negative weight")
	}
}

func TestValidateObjectiveTarget(t *testing.T) {
	// Test valid targets
	validObjectives := []Objective{
		{Name: "detection_rate", Target: 0.95, Type: Maximize},
		{Name: "false_positive_rate", Target: 0.05, Type: Minimize},
		{Name: "response_time_ms", Target: 10.0, Type: Minimize},
		{Name: "user_satisfaction", Target: 0.9, Type: Maximize},
		{Name: "throughput", Target: 1000.0, Type: Maximize},
	}
	
	for _, obj := range validObjectives {
		if err := validateObjectiveTarget(obj); err != nil {
			t.Errorf("Valid objective %s failed validation: %v", obj.Name, err)
		}
	}
	
	// Test invalid targets
	invalidObjectives := []Objective{
		{Name: "detection_rate", Target: 1.5, Type: Maximize},       // > 1.0
		{Name: "false_positive_rate", Target: -0.1, Type: Minimize}, // < 0
		{Name: "response_time_ms", Target: -5.0, Type: Minimize},    // < 0
		{Name: "response_time_ms", Target: 2000.0, Type: Minimize},  // > 1000
		{Name: "throughput", Target: -100.0, Type: Maximize},        // < 0
	}
	
	for _, obj := range invalidObjectives {
		if err := validateObjectiveTarget(obj); err == nil {
			t.Errorf("Invalid objective %s passed validation", obj.Name)
		}
	}
}

func TestCalculateScore(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	// Test with perfect metrics
	perfectMetrics := map[string]float64{
		"detection_rate":       1.0,
		"false_positive_rate":  0.0,
		"response_time_ms":     5.0,
		"user_satisfaction":    1.0,
	}
	
	score := objSet.CalculateScore(perfectMetrics)
	if score <= 0.8 { // Should be high for perfect metrics
		t.Errorf("Expected high score for perfect metrics, got %f", score)
	}
	
	// Test with poor metrics
	poorMetrics := map[string]float64{
		"detection_rate":       0.5,
		"false_positive_rate":  0.2,
		"response_time_ms":     50.0,
		"user_satisfaction":    0.3,
	}
	
	poorScore := objSet.CalculateScore(poorMetrics)
	if poorScore >= score {
		t.Error("Poor metrics should have lower score than perfect metrics")
	}
	
	// Test with missing metrics
	partialMetrics := map[string]float64{
		"detection_rate": 0.9,
	}
	
	partialScore := objSet.CalculateScore(partialMetrics)
	if partialScore < 0 || partialScore > 1 {
		t.Errorf("Score should be between 0 and 1, got %f", partialScore)
	}
}

func TestNormalizeValue(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	// Test detection rate normalization (already 0-1)
	detectionObj := Objective{Name: "detection_rate", Target: 0.95}
	normalized := objSet.normalizeValue(detectionObj, 0.9)
	if normalized != 0.9 {
		t.Errorf("Detection rate should not be modified, got %f", normalized)
	}
	
	// Test response time normalization
	responseObj := Objective{Name: "response_time_ms", Target: 10.0}
	normalized = objSet.normalizeValue(responseObj, 5.0)
	expected := 5.0 / 10.0
	if normalized != expected {
		t.Errorf("Expected %f, got %f", expected, normalized)
	}
	
	// Test capping at 1.0
	normalized = objSet.normalizeValue(responseObj, 20.0)
	if normalized != 1.0 {
		t.Errorf("Expected capping at 1.0, got %f", normalized)
	}
	
	// Test throughput normalization
	throughputObj := Objective{Name: "throughput", Target: 1000.0}
	normalized = objSet.normalizeValue(throughputObj, 500.0)
	expected = 500.0 / 1000.0
	if normalized != expected {
		t.Errorf("Expected %f, got %f", expected, normalized)
	}
}

func TestGetObjectiveByName(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	// Test existing objective
	obj, found := objSet.GetObjectiveByName("detection_rate")
	if !found {
		t.Error("Expected to find detection_rate objective")
	}
	if obj.Name != "detection_rate" {
		t.Error("Retrieved wrong objective")
	}
	
	// Test non-existing objective
	_, found = objSet.GetObjectiveByName("non_existent")
	if found {
		t.Error("Should not find non-existent objective")
	}
}

func TestGetPrimaryObjective(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	primary := objSet.GetPrimaryObjective()
	if primary == nil {
		t.Fatal("Expected primary objective")
	}
	
	// Find max weight manually
	maxWeight := 0.0
	for _, obj := range objectives {
		if obj.Weight > maxWeight {
			maxWeight = obj.Weight
		}
	}
	
	if primary.Weight != maxWeight {
		t.Errorf("Primary objective weight %f should be max weight %f", primary.Weight, maxWeight)
	}
	
	// Test empty objectives
	emptyObjSet := &ObjectiveSet{Objectives: []Objective{}}
	primary = emptyObjSet.GetPrimaryObjective()
	if primary != nil {
		t.Error("Expected nil primary objective for empty set")
	}
}

func TestValidateMetrics(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	// Test metrics that meet targets
	goodMetrics := map[string]float64{
		"detection_rate":       0.96, // Target: 0.95 (maximize)
		"false_positive_rate":  0.04, // Target: 0.05 (minimize)
		"response_time_ms":     8.0,  // Target: 10.0 (minimize)
		"user_satisfaction":    0.92, // Target: 0.9 (maximize)
	}
	
	failures := objSet.ValidateMetrics(goodMetrics)
	if len(failures) > 0 {
		t.Errorf("Good metrics should not have failures: %v", failures)
	}
	
	// Test metrics that don't meet targets
	badMetrics := map[string]float64{
		"detection_rate":       0.90, // Below target
		"false_positive_rate":  0.10, // Above target
		"response_time_ms":     15.0, // Above target
		"user_satisfaction":    0.80, // Below target
	}
	
	failures = objSet.ValidateMetrics(badMetrics)
	if len(failures) != 4 {
		t.Errorf("Expected 4 failures, got %d", len(failures))
	}
	
	// Test missing metrics
	missingMetrics := map[string]float64{
		"detection_rate": 0.95,
		// Missing other metrics
	}
	
	failures = objSet.ValidateMetrics(missingMetrics)
	expectedMissing := len(objectives) - 1
	if len(failures) != expectedMissing {
		t.Errorf("Expected %d missing metric failures, got %d", expectedMissing, len(failures))
	}
}

func TestGenerateReport(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	metrics := map[string]float64{
		"detection_rate":       0.94,
		"false_positive_rate":  0.06,
		"response_time_ms":     12.0,
		"user_satisfaction":    0.88,
	}
	
	report := objSet.GenerateReport(metrics)
	
	if report == nil {
		t.Fatal("Expected non-nil report")
	}
	
	if report.TotalScore < 0 || report.TotalScore > 1 {
		t.Errorf("Total score should be between 0 and 1, got %f", report.TotalScore)
	}
	
	if len(report.ObjectiveScores) != len(objectives) {
		t.Errorf("Expected %d objective scores, got %d", len(objectives), len(report.ObjectiveScores))
	}
	
	if len(report.MetricValues) != len(metrics) {
		t.Errorf("Expected %d metric values, got %d", len(metrics), len(report.MetricValues))
	}
	
	if len(report.TargetComparison) != len(metrics) {
		t.Errorf("Expected %d target comparisons, got %d", len(metrics), len(report.TargetComparison))
	}
	
	// Check target comparisons
	for name, comparison := range report.TargetComparison {
		if comparison.Actual != metrics[name] {
			t.Errorf("Actual value mismatch for %s", name)
		}
		
		obj, _ := objSet.GetObjectiveByName(name)
		if comparison.Target != obj.Target {
			t.Errorf("Target value mismatch for %s", name)
		}
		
		expectedDiff := metrics[name] - obj.Target
		if comparison.Difference != expectedDiff {
			t.Errorf("Difference calculation wrong for %s", name)
		}
	}
	
	if report.GeneratedAt.IsZero() {
		t.Error("Report timestamp not set")
	}
}

func TestUpdateWeights(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	originalWeights := make(map[string]float64)
	for _, obj := range objSet.Objectives {
		originalWeights[obj.Name] = obj.Weight
	}
	
	// Test valid weight update (must sum to 1.0)
	newWeights := map[string]float64{
		"detection_rate": 0.5,
		"false_positive_rate": 0.3,
		"response_time_ms": 0.1,
		"user_satisfaction": 0.1,
	}
	
	err = objSet.UpdateWeights(newWeights)
	if err != nil {
		t.Fatalf("Failed to update weights: %v", err)
	}
	
	// Check that specified weights were updated
	for _, obj := range objSet.Objectives {
		if newWeight, exists := newWeights[obj.Name]; exists {
			if obj.Weight != newWeight {
				t.Errorf("Weight for %s not updated: expected %f, got %f", obj.Name, newWeight, obj.Weight)
			}
		}
	}
	
	// Test invalid weights (don't sum to 1.0)
	invalidWeights := map[string]float64{
		"detection_rate": 0.8,
		"false_positive_rate": 0.8,
	}
	
	err = objSet.UpdateWeights(invalidWeights)
	if err == nil {
		t.Error("Expected error for weights not summing to 1.0")
	}
	
	// Test negative weight
	negativeWeights := map[string]float64{
		"detection_rate": -0.1,
	}
	
	err = objSet.UpdateWeights(negativeWeights)
	if err == nil {
		t.Error("Expected error for negative weight")
	}
}

func TestClone(t *testing.T) {
	objectives := GetDefaultObjectives()
	objSet, err := NewObjectiveSet(objectives)
	if err != nil {
		t.Fatalf("Failed to create objective set: %v", err)
	}
	
	clone := objSet.Clone()
	
	if clone == nil {
		t.Fatal("Expected non-nil clone")
	}
	
	if len(clone.Objectives) != len(objSet.Objectives) {
		t.Error("Clone length mismatch")
	}
	
	// Verify deep copy
	for i, obj := range clone.Objectives {
		if obj.Name != objSet.Objectives[i].Name {
			t.Error("Clone name mismatch")
		}
		if obj.Weight != objSet.Objectives[i].Weight {
			t.Error("Clone weight mismatch")
		}
	}
	
	// Test that modifying clone doesn't affect original
	clone.Objectives[0].Weight = 0.999
	if objSet.Objectives[0].Weight == 0.999 {
		t.Error("Clone modification affected original")
	}
}