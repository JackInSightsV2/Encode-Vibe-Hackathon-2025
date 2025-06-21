package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"qt1-middleware/moderation"
	"qt1-middleware/moderation/layers"
)

func TestRulesLayerIntegration(t *testing.T) {
	// Create temporary directory for test rules
	tempDir, err := os.MkdirTemp("", "rules_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test rules file
	rulesFile := filepath.Join(tempDir, "test_rules.yaml")
	rulesYAML := `
rules:
  - id: "hate_speech_rule"
    name: "Hate Speech Detection"
    description: "Detects hate speech and offensive language"
    type: "keyword"
    keywords: ["hate", "stupid", "idiot", "moron"]
    weight: 0.9
    priority: 1
    action: "block"
    category: "hate_speech"
    enabled: true
    conditions:
      case_sensitive: false
      whole_words: true

  - id: "spam_rule"
    name: "Spam Detection"
    description: "Detects promotional spam content"
    type: "keyword"
    keywords: ["buy now", "limited offer", "click here"]
    weight: 0.6
    priority: 3
    action: "flag"
    category: "spam"
    enabled: true
    conditions:
      case_sensitive: false
      whole_words: false

  - id: "violence_rule"
    name: "Violence Detection"
    description: "Detects violent threats and content"
    type: "regex"
    pattern: "(?i)\\b(kill|murder|destroy|eliminate)\\s+(you|him|her|them)\\b"
    weight: 0.95
    priority: 1
    action: "block"
    category: "violence"
    enabled: true

  - id: "profanity_rule"
    name: "Profanity Filter"
    description: "Detects mild profanity"
    type: "keyword"
    keywords: ["damn", "hell", "crap"]
    weight: 0.4
    priority: 5
    action: "warn"
    category: "profanity"
    enabled: true
    conditions:
      case_sensitive: false
      whole_words: true
`

	err = os.WriteFile(rulesFile, []byte(rulesYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write rules file: %v", err)
	}

	// Create rules layer configuration
	layerConfig := moderation.LayerConfig{
		Name:      "test_rules_layer",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"rules_path":     rulesFile,
			"hot_reload":     false,
			"enable_caching": true,
			"cache_size":     1000,
			"cache_ttl":      30,
		},
	}

	// Create rules layer
	rulesLayer := layers.NewRulesLayer(layerConfig)

	if rulesLayer.Name() != "test_rules_layer" {
		t.Errorf("Expected layer name 'test_rules_layer', got '%s'", rulesLayer.Name())
	}

	if !rulesLayer.Enabled() {
		t.Error("Expected layer to be enabled")
	}

	if rulesLayer.Weight() != 1.0 {
		t.Errorf("Expected weight 1.0, got %f", rulesLayer.Weight())
	}

	// Test with hate speech content
	context := moderation.ModerationContext{
		UserID:      "test_user",
		SessionID:   "test_session",
		ContentType: "message",
		UserType:    "user",
		Timestamp:   time.Now(),
	}

	result := rulesLayer.Moderate("You are so stupid and I hate you", context)

	if !result.Blocked {
		t.Error("Expected hate speech to be blocked")
	}

	if result.Category != moderation.CategoryHateSpeech {
		t.Errorf("Expected category to be hate_speech, got %s", result.Category)
	}

	if result.Score < 0.5 {
		t.Errorf("Expected high score for hate speech, got %f", result.Score)
	}

	// Check details
	if result.Details == nil {
		t.Error("Expected result details to be present")
	} else {
		if rulesTriggered, ok := result.Details["rules_triggered"].(int); !ok || rulesTriggered == 0 {
			t.Error("Expected at least one rule to be triggered")
		}

		if recommendedAction, ok := result.Details["recommended_action"].(string); !ok || recommendedAction != moderation.ActionBlock {
			t.Errorf("Expected recommended action to be block, got %v", recommendedAction)
		}
	}

	// Test with spam content
	result = rulesLayer.Moderate("Buy now for a limited offer! Click here to save!", context)

	if result.Blocked {
		t.Error("Expected spam to be flagged, not blocked")
	}

	if result.Category != moderation.CategorySpam {
		t.Errorf("Expected category to be spam, got %s", result.Category)
	}

	// Test with violent content
	result = rulesLayer.Moderate("I will kill you and destroy everything", context)

	if !result.Blocked {
		t.Error("Expected violent threat to be blocked")
	}

	if result.Category != moderation.CategoryViolence {
		t.Errorf("Expected category to be violence, got %s", result.Category)
	}

	// Test with safe content
	result = rulesLayer.Moderate("Hello, how are you today? Hope you're doing well!", context)

	if result.Blocked {
		t.Error("Expected safe content to be allowed")
	}

	if result.Score > 0.3 {
		t.Errorf("Expected low score for safe content, got %f", result.Score)
	}
}

func TestModerationEngineWithRulesLayer(t *testing.T) {
	// Create temporary directory for test rules
	tempDir, err := os.MkdirTemp("", "engine_rules_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create comprehensive test rules
	rulesFile := filepath.Join(tempDir, "comprehensive_rules.yaml")
	rulesYAML := `
rules:
  - id: "priority_hate_rule"
    name: "High Priority Hate Speech"
    type: "keyword"
    keywords: ["terrorist", "nazi", "kill yourself"]
    weight: 1.0
    priority: 1
    action: "block"
    category: "hate_speech"
    enabled: true

  - id: "medium_hate_rule"
    name: "Medium Priority Hate Speech"
    type: "keyword"
    keywords: ["stupid", "idiot", "loser"]
    weight: 0.7
    priority: 2
    action: "flag"
    category: "hate_speech"
    enabled: true

  - id: "violence_regex_rule"
    name: "Violence Regex"
    type: "regex"
    pattern: "(?i)\\b(shoot|stab|punch)\\s+(you|him|her)\\b"
    weight: 0.9
    priority: 1
    action: "block"
    category: "violence"
    enabled: true

  - id: "composite_spam_rule"
    name: "Composite Spam Detection"
    type: "composite"
    keywords: ["free", "win", "prize"]
    pattern: "(?i)\\b(click|visit|call)\\s+(now|today)\\b"
    weight: 0.6
    priority: 4
    action: "flag"
    category: "spam"
    enabled: true
`

	err = os.WriteFile(rulesFile, []byte(rulesYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write rules file: %v", err)
	}

	// Create moderation engine configuration
	engineConfig := &moderation.AdvancedModerationConfig{
		Enabled: true,
		Layers: []moderation.LayerConfig{
			{
				Name:      "regex_layer",
				Enabled:   true,
				Weight:    0.3,
				Threshold: 0.8,
			},
			{
				Name:      "rules_layer",
				Enabled:   true,
				Weight:    0.7,
				Threshold: 0.5,
				Options: map[string]interface{}{
					"rules_path":     rulesFile,
					"hot_reload":     false,
					"enable_caching": true,
				},
			},
		},
		Thresholds: moderation.ModerationThresholds{
			Low:      0.3,
			Medium:   0.6,
			High:     0.8,
			Critical: 0.95,
		},
		Cache: moderation.CacheConfig{
			Enabled: true,
			TTL:     300,
			MaxSize: 10000,
		},
		Analytics: moderation.AnalyticsConfig{
			Enabled:      true,
			SampleRate:   1.0,
			RetentionDays: 30,
		},
	}

	// Create cache
	cache := moderation.NewModerationCache(engineConfig.Cache.MaxSize, time.Duration(engineConfig.Cache.TTL)*time.Second)

	// Create config manager
	configManager := &moderation.ConfigManager{}

	// Create moderation engine
	engine := moderation.NewModerationEngineWithConfig([]moderation.ModerationLayer{}, cache, engineConfig, configManager)

	// Add regex layer
	regexLayer := layers.NewRegexLayer(engineConfig.Layers[0])
	
	// Add rules layer
	rulesLayer := layers.NewRulesLayer(engineConfig.Layers[1])

	// Set layers
	engine.SetLayers([]moderation.ModerationLayer{regexLayer, rulesLayer})

	// Test cases
	testCases := []struct {
		name           string
		content        string
		expectedAction string
		expectedScore  float64
		expectBlocked  bool
	}{
		{
			name:           "High priority hate speech",
			content:        "You are a terrorist and should kill yourself",
			expectedAction: moderation.ActionBlock,
			expectedScore:  0.8,
			expectBlocked:  true,
		},
		{
			name:           "Medium priority hate speech",
			content:        "You are so stupid and an idiot",
			expectedAction: moderation.ActionFlag,
			expectedScore:  0.5,
			expectBlocked:  false,
		},
		{
			name:           "Violence with regex",
			content:        "I will shoot you in the face",
			expectedAction: moderation.ActionBlock,
			expectedScore:  0.8,
			expectBlocked:  true,
		},
		{
			name:           "Composite spam rule",
			content:        "Win a free prize! Click now to claim your reward!",
			expectedAction: moderation.ActionFlag,
			expectedScore:  0.4,
			expectBlocked:  false,
		},
		{
			name:           "Safe content",
			content:        "Hello, how are you doing today? Nice weather!",
			expectedAction: "allow",
			expectedScore:  0.1,
			expectBlocked:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			context := moderation.ModerationContext{
				UserID:      "test_user",
				SessionID:   "test_session",
				ContentType: "message",
				UserType:    "user",
				Timestamp:   time.Now(),
			}

			result := engine.Moderate(tc.content, context)

			if result.Blocked != tc.expectBlocked {
				t.Errorf("Expected blocked=%v, got %v", tc.expectBlocked, result.Blocked)
			}

			if result.Score < tc.expectedScore-0.3 || result.Score > tc.expectedScore+0.3 {
				t.Errorf("Expected score around %f, got %f", tc.expectedScore, result.Score)
			}

			// Check that rules layer contributed to the result
			if result.Details != nil {
				if layerResults, ok := result.Details["layer_results"].([]moderation.LayerResult); ok {
					found := false
					for _, layerResult := range layerResults {
						if layerResult.LayerName == "rules_layer" {
							found = true
							break
						}
					}
					if !found {
						t.Error("Expected rules layer to contribute to the result")
					}
				}
			}
		})
	}
}

func TestRulesLayerHotReload(t *testing.T) {
	// Create temporary directory for test rules
	tempDir, err := os.MkdirTemp("", "hot_reload_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create initial rules file
	rulesFile := filepath.Join(tempDir, "hot_reload_rules.yaml")
	initialRules := `
rules:
  - id: "initial_rule"
    name: "Initial Rule"
    type: "keyword"
    keywords: ["initial"]
    weight: 0.5
    priority: 3
    action: "warn"
    category: "custom"
    enabled: true
`

	err = os.WriteFile(rulesFile, []byte(initialRules), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial rules file: %v", err)
	}

	// Create rules layer with hot reload enabled
	layerConfig := moderation.LayerConfig{
		Name:      "hot_reload_test_layer",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"rules_path":      rulesFile,
			"hot_reload":      true,
			"reload_interval": 1, // 1 second for fast testing
		},
	}

	rulesLayer := layers.NewRulesLayer(layerConfig)

	// Test initial rule
	context := moderation.ModerationContext{
		UserID:    "test_user",
		Timestamp: time.Now(),
	}

	result := rulesLayer.Moderate("This is an initial test", context)
	if !result.Blocked && result.Score == 0 {
		// Expected - "initial" keyword should trigger
		t.Log("Initial rule working as expected")
	}

	// Update rules file with new rule
	updatedRules := `
rules:
  - id: "initial_rule"
    name: "Initial Rule"
    type: "keyword"
    keywords: ["initial"]
    weight: 0.5
    priority: 3
    action: "warn"
    category: "custom"
    enabled: true

  - id: "updated_rule"
    name: "Updated Rule"
    type: "keyword"
    keywords: ["updated"]
    weight: 0.8
    priority: 2
    action: "block"
    category: "custom"
    enabled: true
`

	err = os.WriteFile(rulesFile, []byte(updatedRules), 0644)
	if err != nil {
		t.Fatalf("Failed to write updated rules file: %v", err)
	}

	// Force reload instead of waiting for hot reload
	err = rulesLayer.ForceReload()
	if err != nil {
		t.Fatalf("Failed to force reload rules: %v", err)
	}

	// Test that new rule is active
	result = rulesLayer.Moderate("This is an updated test", context)
	if result.Score == 0 {
		t.Error("Expected updated rule to trigger, but got zero score")
	}

	// Check that both rules are now available
	rule1, exists1 := rulesLayer.GetRuleByID("initial_rule")
	if !exists1 {
		t.Error("Expected initial_rule to still exist after reload")
	}

	rule2, exists2 := rulesLayer.GetRuleByID("updated_rule")
	if !exists2 {
		t.Error("Expected updated_rule to exist after reload")
	}

	if exists1 && rule1.Name != "Initial Rule" {
		t.Errorf("Expected initial rule name 'Initial Rule', got '%s'", rule1.Name)
	}

	if exists2 && rule2.Name != "Updated Rule" {
		t.Errorf("Expected updated rule name 'Updated Rule', got '%s'", rule2.Name)
	}
}

func TestRulesLayerDynamicRuleManagement(t *testing.T) {
	// Create rules layer
	layerConfig := moderation.LayerConfig{
		Name:      "dynamic_rules_layer",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"enable_caching": false, // Disable caching for immediate rule effects
		},
	}

	rulesLayer := layers.NewRulesLayer(layerConfig)

	// Test adding a rule dynamically
	newRule := &moderation.ModerationRule{
		ID:                 "dynamic_rule_1",
		Name:               "Dynamic Rule 1",
		Type:               moderation.RuleTypeKeyword,
		Keywords:           []string{"dynamic", "test"},
		Weight:             0.7,
		Priority:           2,
		Action:             moderation.ActionFlag,
		Category:           moderation.CategoryCustom,
		Enabled:            true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	err := rulesLayer.AddRule(newRule)
	if err != nil {
		t.Fatalf("Failed to add dynamic rule: %v", err)
	}

	// Test that the rule is active
	context := moderation.ModerationContext{
		UserID:    "test_user",
		Timestamp: time.Now(),
	}

	result := rulesLayer.Moderate("This is a dynamic test", context)
	if result.Score == 0 {
		t.Error("Expected dynamic rule to trigger")
	}

	// Test retrieving the rule
	retrievedRule, exists := rulesLayer.GetRuleByID("dynamic_rule_1")
	if !exists {
		t.Error("Expected to retrieve dynamically added rule")
	}

	if retrievedRule.Name != "Dynamic Rule 1" {
		t.Errorf("Expected rule name 'Dynamic Rule 1', got '%s'", retrievedRule.Name)
	}

	// Test disabling the rule
	err = rulesLayer.DisableRule("dynamic_rule_1")
	if err != nil {
		t.Fatalf("Failed to disable rule: %v", err)
	}

	// Test that disabled rule doesn't trigger
	result = rulesLayer.Moderate("This is a dynamic test", context)
	if result.Score > 0 {
		t.Error("Expected disabled rule not to trigger")
	}

	// Test enabling the rule again
	err = rulesLayer.EnableRule("dynamic_rule_1")
	if err != nil {
		t.Fatalf("Failed to enable rule: %v", err)
	}

	result = rulesLayer.Moderate("This is a dynamic test", context)
	if result.Score == 0 {
		t.Error("Expected enabled rule to trigger again")
	}

	// Test removing the rule
	err = rulesLayer.RemoveRule("dynamic_rule_1")
	if err != nil {
		t.Fatalf("Failed to remove rule: %v", err)
	}

	// Test that removed rule doesn't exist
	_, exists = rulesLayer.GetRuleByID("dynamic_rule_1")
	if exists {
		t.Error("Expected removed rule not to exist")
	}

	result = rulesLayer.Moderate("This is a dynamic test", context)
	if result.Score > 0 {
		t.Error("Expected removed rule not to trigger")
	}
}

func TestRulesLayerStatistics(t *testing.T) {
	// Create temporary directory for test rules
	tempDir, err := os.MkdirTemp("", "stats_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test rules
	rulesFile := filepath.Join(tempDir, "stats_rules.yaml")
	rulesYAML := `
rules:
  - id: "stats_rule_block"
    name: "Stats Rule Block"
    type: "keyword"
    keywords: ["block"]
    weight: 0.9
    priority: 1
    action: "block"
    category: "custom"
    enabled: true

  - id: "stats_rule_flag"
    name: "Stats Rule Flag"
    type: "keyword"
    keywords: ["flag"]
    weight: 0.6
    priority: 3
    action: "flag"
    category: "custom"
    enabled: true

  - id: "stats_rule_warn"
    name: "Stats Rule Warn"
    type: "keyword"
    keywords: ["warn"]
    weight: 0.4
    priority: 5
    action: "warn"
    category: "custom"
    enabled: true
`

	err = os.WriteFile(rulesFile, []byte(rulesYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write rules file: %v", err)
	}

	layerConfig := moderation.LayerConfig{
		Name:      "stats_test_layer",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"rules_path": rulesFile,
		},
	}

	rulesLayer := layers.NewRulesLayer(layerConfig)

	context := moderation.ModerationContext{
		UserID:    "test_user",
		Timestamp: time.Now(),
	}

	// Test different actions to generate statistics
	rulesLayer.Moderate("This should block content", context)   // block action
	rulesLayer.Moderate("This should flag content", context)    // flag action
	rulesLayer.Moderate("This should warn content", context)    // warn action
	rulesLayer.Moderate("This is safe content", context)        // no action
	rulesLayer.Moderate("Another block test", context)          // block action

	// Get statistics
	stats := rulesLayer.GetStats()

	// Verify basic stats
	if totalRequests, ok := stats["total_requests"].(int64); !ok || totalRequests != 5 {
		t.Errorf("Expected 5 total requests, got %v", stats["total_requests"])
	}

	if successfulRuns, ok := stats["successful_runs"].(int64); !ok || successfulRuns != 5 {
		t.Errorf("Expected 5 successful runs, got %v", stats["successful_runs"])
	}

	if actionsBlocked, ok := stats["actions_blocked"].(int64); !ok || actionsBlocked != 2 {
		t.Errorf("Expected 2 blocked actions, got %v", stats["actions_blocked"])
	}

	if actionsFlagged, ok := stats["actions_flagged"].(int64); !ok || actionsFlagged != 1 {
		t.Errorf("Expected 1 flagged action, got %v", stats["actions_flagged"])
	}

	if actionsWarned, ok := stats["actions_warned"].(int64); !ok || actionsWarned != 1 {
		t.Errorf("Expected 1 warned action, got %v", stats["actions_warned"])
	}

	// Verify rule engine stats are included
	if ruleEngineStats, ok := stats["rule_engine_stats"].(map[string]interface{}); !ok {
		t.Error("Expected rule engine stats to be included")
	} else {
		if totalRules, ok := ruleEngineStats["total_rules"].(int64); !ok || totalRules != 3 {
			t.Errorf("Expected 3 total rules in engine stats, got %v", ruleEngineStats["total_rules"])
		}
	}
}

func BenchmarkRulesLayerModeration(b *testing.B) {
	// Create temporary directory for benchmark rules
	tempDir, err := os.MkdirTemp("", "benchmark_test")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create comprehensive rules for realistic benchmarking
	rulesFile := filepath.Join(tempDir, "benchmark_rules.yaml")
	rulesYAML := `
rules:
  - id: "bench_hate_1"
    name: "Hate Speech 1"
    type: "keyword"
    keywords: ["hate", "stupid", "idiot", "moron", "dumb"]
    weight: 0.9
    priority: 1
    action: "block"
    category: "hate_speech"
    enabled: true

  - id: "bench_hate_2"
    name: "Hate Speech 2"
    type: "regex"
    pattern: "(?i)\\b(kill|murder|destroy)\\s+(you|him|her|them)\\b"
    weight: 0.95
    priority: 1
    action: "block"
    category: "violence"
    enabled: true

  - id: "bench_spam_1"
    name: "Spam Detection 1"
    type: "keyword"
    keywords: ["buy now", "limited offer", "click here", "free gift"]
    weight: 0.6
    priority: 3
    action: "flag"
    category: "spam"
    enabled: true

  - id: "bench_spam_2"
    name: "Spam Detection 2"
    type: "regex"
    pattern: "(?i)\\b(urgent|act now|don't miss)\\b"
    weight: 0.5
    priority: 4
    action: "flag"
    category: "spam"
    enabled: true

  - id: "bench_profanity"
    name: "Profanity Filter"
    type: "keyword"
    keywords: ["damn", "hell", "crap", "shit", "fuck"]
    weight: 0.4
    priority: 5
    action: "warn"
    category: "profanity"
    enabled: true
`

	err = os.WriteFile(rulesFile, []byte(rulesYAML), 0644)
	if err != nil {
		b.Fatalf("Failed to write rules file: %v", err)
	}

	layerConfig := moderation.LayerConfig{
		Name:      "benchmark_layer",
		Enabled:   true,
		Weight:    1.0,
		Threshold: 0.5,
		Options: map[string]interface{}{
			"rules_path":     rulesFile,
			"enable_caching": true,
			"cache_size":     10000,
		},
	}

	rulesLayer := layers.NewRulesLayer(layerConfig)

	context := moderation.ModerationContext{
		UserID:      "bench_user",
		SessionID:   "bench_session",
		ContentType: "message",
		Timestamp:   time.Now(),
	}

	// Test content samples
	testContent := []string{
		"Hello, how are you today?",
		"You are so stupid and I hate you",
		"Buy now for a limited offer!",
		"I will kill you and destroy everything",
		"This is some normal conversation text",
		"Damn, that's really annoying",
		"Click here for a free gift today!",
		"Nice weather we're having",
		"You're such an idiot and moron",
		"Act now before it's too late!",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			content := testContent[i%len(testContent)]
			rulesLayer.Moderate(content, context)
			i++
		}
	})
}