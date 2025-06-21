package moderation

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// Helper function for floating point comparison
func abs(x float64) float64 {
	return math.Abs(x)
}

// MockLayer implements ModerationLayer for testing
type MockLayer struct {
	name      string
	weight    float64
	enabled   bool
	mockScore float64
	mockReason string
}

func (ml *MockLayer) Name() string {
	return ml.name
}

func (ml *MockLayer) Weight() float64 {
	return ml.weight
}

func (ml *MockLayer) Enabled() bool {
	return ml.enabled
}

func (ml *MockLayer) Config() LayerConfig {
	return LayerConfig{
		Name:    ml.name,
		Enabled: ml.enabled,
		Weight:  ml.weight,
	}
}

func (ml *MockLayer) Moderate(content string, context ModerationContext) ModerationResult {
	return ModerationResult{
		Score:      ml.mockScore,
		Confidence: 0.8,
		Blocked:    ml.mockScore >= 0.8,
		Reason:     ml.mockReason,
		Category:   CategoryCustom,
		LayerName:  ml.name,
		Details:    make(map[string]interface{}),
	}
}

func TestModerationEngineCreation(t *testing.T) {
	// Create a temporary config for testing
	configPath := ""  // Use default config
	
	engine, err := NewModerationEngine(configPath)
	if err != nil {
		t.Fatalf("Failed to create moderation engine: %v", err)
	}
	
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}
	
	if engine.cache == nil {
		t.Fatal("Cache should not be nil")
	}
	
	if engine.config == nil {
		t.Fatal("Config should not be nil")
	}
	
	engine.Close()
}

func TestLayerRegistration(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	// Register a mock layer
	mockLayer := &MockLayer{
		name:    "test_layer",
		weight:  0.5,
		enabled: true,
	}
	
	err = engine.RegisterLayer(mockLayer)
	if err != nil {
		t.Fatalf("Failed to register layer: %v", err)
	}
	
	// Check that layer was registered
	layerNames := engine.GetLayerNames()
	if len(layerNames) != 1 || layerNames[0] != "test_layer" {
		t.Errorf("Expected layer 'test_layer', got: %v", layerNames)
	}
	
	// Try to register the same layer again (should fail)
	err = engine.RegisterLayer(mockLayer)
	if err == nil {
		t.Error("Expected error when registering duplicate layer")
	}
}

func TestModerationPipeline(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	// Enable moderation in config
	engine.config.Enabled = true
	
	// Register test layers
	layer1 := &MockLayer{
		name:       "layer1",
		weight:     0.6,
		enabled:    true,
		mockScore:  0.3,
		mockReason: "Layer 1 detection",
	}
	
	layer2 := &MockLayer{
		name:       "layer2", 
		weight:     0.4,
		enabled:    true,
		mockScore:  0.7,
		mockReason: "Layer 2 detection",
	}
	
	engine.RegisterLayer(layer1)
	engine.RegisterLayer(layer2)
	
	// Create test context
	context := ModerationContext{
		UserID:    "test_user",
		SessionID: "test_session",
		Timestamp: time.Now(),
	}
	
	// Test moderation
	result, err := engine.Moderate("test content", context)
	if err != nil {
		t.Fatalf("Moderation failed: %v", err)
	}
	
	// Check result
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	
	// Calculate expected score: (0.3 * 0.6 + 0.7 * 0.4) / (0.6 + 0.4) = 0.46
	expectedScore := 0.46
	if abs(result.FinalScore - expectedScore) > 0.001 {
		t.Errorf("Expected final score %.2f, got %.2f", expectedScore, result.FinalScore)
	}
	
	// Check that layer results are included
	if len(result.LayerResults) != 2 {
		t.Errorf("Expected 2 layer results, got %d", len(result.LayerResults))
	}
}

func TestModerationDisabled(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	// Ensure moderation is disabled (default)
	engine.config.Enabled = false
	
	context := ModerationContext{
		UserID: "test_user",
	}
	
	result, err := engine.Moderate("test content", context)
	if err != nil {
		t.Fatalf("Moderation failed: %v", err)
	}
	
	// Should return safe result when disabled
	if result.FinalScore != 0.0 {
		t.Errorf("Expected score 0.0 when disabled, got %.2f", result.FinalScore)
	}
	
	if result.FinalDecision {
		t.Error("Expected no blocking when moderation disabled")
	}
}

func TestThresholdActions(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	engine.config.Enabled = true
	
	testCases := []struct {
		score          float64
		expectedAction string
		expectedSeverity string
		shouldBlock    bool
	}{
		{0.2, ActionLog, SeverityLow, false},
		{0.7, ActionFlag, SeverityMedium, false},
		{0.85, ActionBlock, SeverityHigh, true},
		{0.98, ActionBlockAlert, SeverityCritical, true},
	}
	
	for _, tc := range testCases {
		// Register a layer that returns the test score
		layerName := "test_layer"
		layer := &MockLayer{
			name:       layerName,
			weight:     1.0,
			enabled:    true,
			mockScore:  tc.score,
			mockReason: "Test detection",
		}
		
		// Create new engine for each test to avoid layer conflicts
		testEngine, err := NewModerationEngine("")
		if err != nil {
			t.Fatalf("Failed to create test engine: %v", err)
		}
		testEngine.config.Enabled = true
		testEngine.RegisterLayer(layer)
		
		context := ModerationContext{UserID: "test"}
		result, err := testEngine.Moderate("test", context)
		
		if err != nil {
			t.Fatalf("Moderation failed for score %.2f: %v", tc.score, err)
		}
		
		if result.Action != tc.expectedAction {
			t.Errorf("Score %.2f: expected action %s, got %s", tc.score, tc.expectedAction, result.Action)
		}
		
		if result.Severity != tc.expectedSeverity {
			t.Errorf("Score %.2f: expected severity %s, got %s", tc.score, tc.expectedSeverity, result.Severity)
		}
		
		if result.FinalDecision != tc.shouldBlock {
			t.Errorf("Score %.2f: expected blocking %v, got %v", tc.score, tc.shouldBlock, result.FinalDecision)
		}
		
		testEngine.Close()
	}
}

func TestCaching(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	engine.config.Enabled = true
	engine.config.Cache.Enabled = true
	
	// Register a layer
	layer := &MockLayer{
		name:       "cache_test",
		weight:     1.0,
		enabled:    true,
		mockScore:  0.5,
		mockReason: "Cache test",
	}
	engine.RegisterLayer(layer)
	
	context := ModerationContext{
		UserID:    "cache_user",
		SessionID: "cache_session",
	}
	
	content := "test content for caching"
	
	// First call - should not be cached
	result1, err := engine.Moderate(content, context)
	if err != nil {
		t.Fatalf("First moderation failed: %v", err)
	}
	
	if result1.CacheHit {
		t.Error("First call should not be a cache hit")
	}
	
	// Second call - should be cached
	result2, err := engine.Moderate(content, context)
	if err != nil {
		t.Fatalf("Second moderation failed: %v", err)
	}
	
	if !result2.CacheHit {
		t.Error("Second call should be a cache hit")
	}
	
	// Results should be the same
	if result1.FinalScore != result2.FinalScore {
		t.Errorf("Cached result score mismatch: %.2f vs %.2f", result1.FinalScore, result2.FinalScore)
	}
}

func TestStats(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	engine.config.Enabled = true
	engine.config.Cache.Enabled = false // Disable cache for stats testing
	
	// Register layers
	layer1 := &MockLayer{
		name:       "stats_layer1",
		weight:     0.5,
		enabled:    true,
		mockScore:  0.9, // High score
		mockReason: "Stats test 1",
	}
	
	layer2 := &MockLayer{
		name:       "stats_layer2",
		weight:     0.5,
		enabled:    true,
		mockScore:  0.9, // Also high score to ensure blocking
		mockReason: "Stats test 2",
	}
	
	engine.RegisterLayer(layer1)
	engine.RegisterLayer(layer2)
	
	context := ModerationContext{UserID: "stats_user"}
	
	// Process some requests with different content to avoid any potential caching
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("test content %d", i)
		_, err := engine.Moderate(content, context)
		if err != nil {
			t.Fatalf("Moderation %d failed: %v", i, err)
		}
	}
	
	// Check stats
	stats := engine.GetStats()
	
	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 total requests, got %d", stats.TotalRequests)
	}
	
	if stats.BlockedRequests != 5 { // All should be blocked due to high score
		t.Errorf("Expected 5 blocked requests, got %d", stats.BlockedRequests)
	}
	
	// Check layer stats
	if len(stats.LayerStats) != 2 {
		t.Errorf("Expected 2 layer stats, got %d", len(stats.LayerStats))
	}
	
	layer1Stats := stats.LayerStats["stats_layer1"]
	if layer1Stats.TotalProcessed != 5 {
		t.Errorf("Layer 1 should have processed 5 requests, got %d", layer1Stats.TotalProcessed)
	}
}

func TestEnabledLayers(t *testing.T) {
	engine, err := NewModerationEngine("")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()
	
	// Register layers with different enabled states
	enabledLayer := &MockLayer{
		name:    "enabled_layer",
		weight:  0.5,
		enabled: true,
	}
	
	disabledLayer := &MockLayer{
		name:    "disabled_layer",
		weight:  0.5,
		enabled: false,
	}
	
	engine.RegisterLayer(enabledLayer)
	engine.RegisterLayer(disabledLayer)
	
	// Get enabled layers
	enabledLayers := engine.GetEnabledLayers()
	
	if len(enabledLayers) != 1 {
		t.Errorf("Expected 1 enabled layer, got %d", len(enabledLayers))
	}
	
	if enabledLayers[0].Name() != "enabled_layer" {
		t.Errorf("Expected enabled layer name 'enabled_layer', got '%s'", enabledLayers[0].Name())
	}
}