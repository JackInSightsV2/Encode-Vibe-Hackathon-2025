package moderation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuleEngineCreation(t *testing.T) {
	config := &RuleEngineConfig{
		RulesPath:       "/tmp/test_rules",
		HotReload:       false,
		MaxRules:        100,
		DefaultWeight:   0.5,
		DefaultPriority: 3,
		EnableCaching:   true,
		CacheSize:       1000,
		CacheTTL:        5 * time.Minute,
	}

	engine := NewRuleEngine(config)

	if engine == nil {
		t.Fatal("Failed to create rule engine")
	}

	if engine.config != config {
		t.Error("Rule engine config not set correctly")
	}

	if engine.parser == nil {
		t.Error("Rule parser not initialized")
	}

	if engine.executor == nil {
		t.Error("Rule executor not initialized")
	}

	if engine.stats == nil {
		t.Error("Rule stats not initialized")
	}
}

func TestRuleEngineSetRules(t *testing.T) {
	config := &RuleEngineConfig{
		MaxRules: 10,
	}
	engine := NewRuleEngine(config)

	rules := []*ModerationRule{
		{
			ID:          "test_rule_1",
			Name:        "Test Rule 1",
			Type:        RuleTypeKeyword,
			Keywords:    []string{"bad", "evil"},
			Weight:      0.8,
			Priority:    1,
			Action:      ActionBlock,
			Category:    CategoryHateSpeech,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "test_rule_2",
			Name:        "Test Rule 2",
			Type:        RuleTypeRegex,
			Pattern:     "(?i)\\b(spam|advertisement)\\b",
			Weight:      0.6,
			Priority:    2,
			Action:      ActionFlag,
			Category:    CategorySpam,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	err := engine.SetRules(rules)
	if err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}

	if len(engine.rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(engine.rules))
	}

	if len(engine.rulesByID) != 2 {
		t.Errorf("Expected 2 rules in ID index, got %d", len(engine.rulesByID))
	}

	// Test rule retrieval by ID
	rule, exists := engine.GetRuleByID("test_rule_1")
	if !exists {
		t.Error("Rule test_rule_1 not found")
	}
	if rule.Name != "Test Rule 1" {
		t.Errorf("Expected rule name 'Test Rule 1', got '%s'", rule.Name)
	}

	// Test rules by type
	keywordRules := engine.GetRulesByType(RuleTypeKeyword)
	if len(keywordRules) != 1 {
		t.Errorf("Expected 1 keyword rule, got %d", len(keywordRules))
	}

	regexRules := engine.GetRulesByType(RuleTypeRegex)
	if len(regexRules) != 1 {
		t.Errorf("Expected 1 regex rule, got %d", len(regexRules))
	}
}

func TestRuleEngineExecute(t *testing.T) {
	config := &RuleEngineConfig{}
	engine := NewRuleEngine(config)

	// Create test rules
	rules := []*ModerationRule{
		{
			ID:               "hate_rule",
			Name:             "Hate Speech Detection",
			Type:             RuleTypeKeyword,
			Keywords:         []string{"hate", "stupid"},
			normalizedKeywords: []string{"hate", "stupid"},
			Weight:           0.9,
			Priority:         1,
			Action:           ActionBlock,
			Category:         CategoryHateSpeech,
			Enabled:          true,
		},
		{
			ID:               "spam_rule",
			Name:             "Spam Detection",
			Type:             RuleTypeKeyword,
			Keywords:         []string{"buy", "discount"},
			normalizedKeywords: []string{"buy", "discount"},
			Weight:           0.6,
			Priority:         3,
			Action:           ActionFlag,
			Category:         CategorySpam,
			Enabled:          true,
		},
	}

	err := engine.SetRules(rules)
	if err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}

	// Test with hate speech content
	context := ModerationContext{
		UserID:      "test_user",
		SessionID:   "test_session",
		ContentType: "message",
		Timestamp:   time.Now(),
	}

	result, err := engine.Execute("You are so stupid and I hate you", context)
	if err != nil {
		t.Fatalf("Failed to execute rules: %v", err)
	}

	if result.RecommendedAction != ActionBlock {
		t.Errorf("Expected action to be block, got %s", result.RecommendedAction)
	}

	if result.PrimaryCategory != CategoryHateSpeech {
		t.Errorf("Expected primary category to be hate_speech, got %s", result.PrimaryCategory)
	}

	if len(result.TriggeredRules) == 0 {
		t.Error("Expected at least one triggered rule")
	}

	// Test with safe content
	result, err = engine.Execute("Hello, how are you today?", context)
	if err != nil {
		t.Fatalf("Failed to execute rules: %v", err)
	}

	if result.RecommendedAction != "allow" {
		t.Errorf("Expected action to be allow, got %s", result.RecommendedAction)
	}

	if len(result.TriggeredRules) != 0 {
		t.Errorf("Expected no triggered rules for safe content, got %d", len(result.TriggeredRules))
	}
}

func TestRuleEngineStats(t *testing.T) {
	config := &RuleEngineConfig{}
	engine := NewRuleEngine(config)

	rules := []*ModerationRule{
		{
			ID:               "test_rule",
			Name:             "Test Rule",
			Type:             RuleTypeKeyword,
			Keywords:         []string{"test"},
			normalizedKeywords: []string{"test"},
			Weight:           0.5,
			Priority:         3,
			Action:           ActionWarn,
			Category:         CategoryCustom,
			Enabled:          true,
		},
	}

	err := engine.SetRules(rules)
	if err != nil {
		t.Fatalf("Failed to set rules: %v", err)
	}

	context := ModerationContext{
		UserID:    "test_user",
		Timestamp: time.Now(),
	}

	// Execute multiple times to generate stats
	for i := 0; i < 5; i++ {
		_, err := engine.Execute("This is a test message", context)
		if err != nil {
			t.Fatalf("Failed to execute rules: %v", err)
		}
	}

	stats := engine.GetStats()

	if totalRules, ok := stats["total_rules"].(int64); !ok || totalRules != 1 {
		t.Errorf("Expected total_rules to be 1, got %v", stats["total_rules"])
	}

	if activeRules, ok := stats["active_rules"].(int64); !ok || activeRules != 1 {
		t.Errorf("Expected active_rules to be 1, got %v", stats["active_rules"])
	}

	if rulesExecuted, ok := stats["rules_executed"].(int64); !ok || rulesExecuted != 5 {
		t.Errorf("Expected rules_executed to be 5, got %v", stats["rules_executed"])
	}
}

func TestRuleParserValidation(t *testing.T) {
	config := &RuleEngineConfig{}
	parser := NewRuleParser(config)

	// Test valid rule
	validRule := &ModerationRule{
		ID:       "valid_rule",
		Name:     "Valid Rule",
		Type:     RuleTypeKeyword,
		Keywords: []string{"test"},
		Weight:   0.5,
		Priority: 3,
		Action:   ActionWarn,
		Category: CategoryCustom,
	}

	err := parser.ValidateRule(validRule)
	if err != nil {
		t.Errorf("Valid rule failed validation: %v", err)
	}

	// Test invalid rule - missing ID
	invalidRule := &ModerationRule{
		Name:     "Invalid Rule",
		Type:     RuleTypeKeyword,
		Keywords: []string{"test"},
		Weight:   0.5,
		Priority: 3,
		Action:   ActionWarn,
		Category: CategoryCustom,
	}

	err = parser.ValidateRule(invalidRule)
	if err == nil {
		t.Error("Invalid rule (missing ID) should have failed validation")
	}

	// Test invalid rule - bad weight
	invalidRule2 := &ModerationRule{
		ID:       "invalid_rule_2",
		Name:     "Invalid Rule 2",
		Type:     RuleTypeKeyword,
		Keywords: []string{"test"},
		Weight:   1.5, // Invalid weight > 1
		Priority: 3,
		Action:   ActionWarn,
		Category: CategoryCustom,
	}

	err = parser.ValidateRule(invalidRule2)
	if err == nil {
		t.Error("Invalid rule (bad weight) should have failed validation")
	}

	// Test invalid rule - bad priority
	invalidRule3 := &ModerationRule{
		ID:       "invalid_rule_3",
		Name:     "Invalid Rule 3",
		Type:     RuleTypeKeyword,
		Keywords: []string{"test"},
		Weight:   0.5,
		Priority: 15, // Invalid priority > 10
		Action:   ActionWarn,
		Category: CategoryCustom,
	}

	err = parser.ValidateRule(invalidRule3)
	if err == nil {
		t.Error("Invalid rule (bad priority) should have failed validation")
	}
}

func TestRuleParserYAMLParsing(t *testing.T) {
	config := &RuleEngineConfig{}
	parser := NewRuleParser(config)

	yamlContent := `
rules:
  - id: "test_yaml_rule"
    name: "Test YAML Rule"
    description: "A test rule from YAML"
    type: "keyword"
    keywords: ["test", "yaml"]
    weight: 0.7
    priority: 2
    action: "flag"
    category: "custom"
    enabled: true
    conditions:
      case_sensitive: false
      whole_words: true
`

	rules, err := parser.ParseRulesFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML rules: %v", err)
	}

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]
	if rule.ID != "test_yaml_rule" {
		t.Errorf("Expected rule ID 'test_yaml_rule', got '%s'", rule.ID)
	}

	if rule.Name != "Test YAML Rule" {
		t.Errorf("Expected rule name 'Test YAML Rule', got '%s'", rule.Name)
	}

	if len(rule.Keywords) != 2 {
		t.Errorf("Expected 2 keywords, got %d", len(rule.Keywords))
	}

	if rule.Weight != 0.7 {
		t.Errorf("Expected weight 0.7, got %f", rule.Weight)
	}

	if rule.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", rule.Priority)
	}

	if rule.Action != ActionFlag {
		t.Errorf("Expected action 'flag', got '%s'", rule.Action)
	}

	if !rule.Enabled {
		t.Error("Expected rule to be enabled")
	}

	if rule.Conditions == nil {
		t.Error("Expected conditions to be set")
	} else {
		if rule.Conditions.CaseSensitive {
			t.Error("Expected case_sensitive to be false")
		}
		if !rule.Conditions.WholeWords {
			t.Error("Expected whole_words to be true")
		}
	}
}

func TestRuleExecutorKeywordRule(t *testing.T) {
	config := &RuleEngineConfig{}
	executor := NewRuleExecutor(config)

	rule := &ModerationRule{
		ID:                 "keyword_test",
		Name:               "Keyword Test",
		Type:               RuleTypeKeyword,
		Keywords:           []string{"bad", "evil"},
		normalizedKeywords: []string{"bad", "evil"},
		Weight:             0.8,
		Priority:           1,
		Action:             ActionBlock,
		Category:           CategoryHateSpeech,
		Enabled:            true,
	}

	context := ModerationContext{
		UserID:    "test_user",
		Timestamp: time.Now(),
	}

	// Test with matching content
	result, err := executor.executeKeywordRule(rule, "This is a bad message", context)
	if err != nil {
		t.Fatalf("Failed to execute keyword rule: %v", err)
	}

	if !result.Matched {
		t.Error("Expected rule to match")
	}

	if result.Score != 0.8 {
		t.Errorf("Expected score 0.8, got %f", result.Score)
	}

	if len(result.MatchDetails.MatchedKeywords) != 1 {
		t.Errorf("Expected 1 matched keyword, got %d", len(result.MatchDetails.MatchedKeywords))
	}

	// Test with non-matching content
	result, err = executor.executeKeywordRule(rule, "This is a good message", context)
	if err != nil {
		t.Fatalf("Failed to execute keyword rule: %v", err)
	}

	if result.Matched {
		t.Error("Expected rule not to match")
	}

	if result.Score != 0.0 {
		t.Errorf("Expected score 0.0, got %f", result.Score)
	}
}

func TestRuleScoreCalculator(t *testing.T) {
	calculator := NewRuleScoreCalculator(ScoreStrategyMax)

	results := []*RuleResult{
		{
			RuleID:     "rule1",
			RuleName:   "Rule 1",
			Matched:    true,
			Score:      0.7,
			Confidence: 0.8,
			Action:     ActionFlag,
			Category:   CategorySpam,
			Context: map[string]interface{}{
				"rule_priority": 2,
			},
		},
		{
			RuleID:     "rule2",
			RuleName:   "Rule 2",
			Matched:    true,
			Score:      0.9,
			Confidence: 0.9,
			Action:     ActionBlock,
			Category:   CategoryViolence,
			Context: map[string]interface{}{
				"rule_priority": 1,
			},
		},
	}

	aggregated := calculator.AggregateRuleResults(results)

	if aggregated.RecommendedAction != ActionBlock {
		t.Errorf("Expected action to be block, got %s", aggregated.RecommendedAction)
	}

	if aggregated.PrimaryCategory != CategoryViolence {
		t.Errorf("Expected primary category to be violence, got %s", aggregated.PrimaryCategory)
	}

	if len(aggregated.TriggeredRules) != 2 {
		t.Errorf("Expected 2 triggered rules, got %d", len(aggregated.TriggeredRules))
	}

	if aggregated.ScoreBreakdown == nil {
		t.Error("Expected score breakdown to be present")
	}
}

func TestRuleManagerValidation(t *testing.T) {
	// Create temporary directory for test rules
	tempDir, err := os.MkdirTemp("", "rule_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid YAML rule file
	ruleFile := filepath.Join(tempDir, "test_rules.yaml")
	yamlContent := `
rules:
  - id: "temp_rule"
    name: "Temporary Rule"
    type: "keyword"
    keywords: ["temp"]
    weight: 0.5
    priority: 3
    action: "warn"
    category: "custom"
    enabled: true
`
	err = os.WriteFile(ruleFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}

	config := &RuleEngineConfig{
		RulesPath: ruleFile,
		HotReload: false,
	}

	engine := NewRuleEngine(config)
	manager := NewRuleManager(engine, config)

	// Test path validation
	err = manager.ValidateRulesPath()
	if err != nil {
		t.Errorf("Valid rules path failed validation: %v", err)
	}

	// Test loading rules
	err = manager.LoadRules()
	if err != nil {
		t.Errorf("Failed to load rules: %v", err)
	}

	// Check that rules were loaded
	rule, exists := engine.GetRuleByID("temp_rule")
	if !exists {
		t.Error("Expected rule 'temp_rule' to be loaded")
	}

	if rule.Name != "Temporary Rule" {
		t.Errorf("Expected rule name 'Temporary Rule', got '%s'", rule.Name)
	}
}

func TestRuleContextMatching(t *testing.T) {
	config := &RuleEngineConfig{}
	executor := NewRuleExecutor(config)

	// Create rule with context restrictions
	rule := &ModerationRule{
		ID:       "context_rule",
		Name:     "Context Restricted Rule",
		Type:     RuleTypeKeyword,
		Keywords: []string{"restricted"},
		normalizedKeywords: []string{"restricted"},
		Weight:   0.8,
		Priority: 1,
		Action:   ActionBlock,
		Category: CategoryCustom,
		Enabled:  true,
		Context: &RuleContext{
			UserTypes:    []string{"user", "guest"},
			ContentTypes: []string{"message"},
		},
	}

	// Test with matching context
	matchingContext := ModerationContext{
		UserID:      "test_user",
		UserType:    "user",
		ContentType: "message",
		Timestamp:   time.Now(),
	}

	applies := executor.ruleAppliestoContext(rule, matchingContext)
	if !applies {
		t.Error("Rule should apply to matching context")
	}

	// Test with non-matching user type
	nonMatchingContext := ModerationContext{
		UserID:      "admin_user",
		UserType:    "admin",
		ContentType: "message",
		Timestamp:   time.Now(),
	}

	applies = executor.ruleAppliestoContext(rule, nonMatchingContext)
	if applies {
		t.Error("Rule should not apply to non-matching user type")
	}

	// Test with non-matching content type
	nonMatchingContext2 := ModerationContext{
		UserID:      "test_user",
		UserType:    "user",
		ContentType: "post",
		Timestamp:   time.Now(),
	}

	applies = executor.ruleAppliestoContext(rule, nonMatchingContext2)
	if applies {
		t.Error("Rule should not apply to non-matching content type")
	}
}