package layers

import (
	"log"
	"time"

	"qt1-middleware/moderation"
)

// RulesLayer integrates the custom rule engine as a moderation layer
type RulesLayer struct {
	name         string
	weight       float64
	enabled      bool
	config       moderation.LayerConfig
	ruleEngine   *moderation.RuleEngine
	ruleManager  *moderation.RuleManager
	stats        *RulesLayerStats
}

// RulesLayerStats tracks performance metrics for the rules layer
type RulesLayerStats struct {
	TotalRequests     int64
	SuccessfulRuns    int64
	FailedRuns        int64
	AverageLatency    time.Duration
	RulesTriggered    int64
	ActionsBlocked    int64
	ActionsFlagged    int64
	ActionsWarned     int64
}

// NewRulesLayer creates a new rules-based moderation layer
func NewRulesLayer(config moderation.LayerConfig) *RulesLayer {
	// Create rule engine configuration from layer options
	ruleConfig := &moderation.RuleEngineConfig{
		RulesPath:        "/etc/moderation/rules", // Default path
		HotReload:        false,
		ReloadInterval:   5 * time.Minute,
		MaxRules:         1000,
		DefaultWeight:    0.5,
		DefaultPriority:  3,
		EnableCaching:    true,
		CacheSize:        10000,
		CacheTTL:         30 * time.Minute,
	}

	// Override with layer-specific options
	if config.Options != nil {
		if rulesPath, ok := config.Options["rules_path"].(string); ok {
			ruleConfig.RulesPath = rulesPath
		}
		if hotReload, ok := config.Options["hot_reload"].(bool); ok {
			ruleConfig.HotReload = hotReload
		}
		if reloadInterval, ok := config.Options["reload_interval"].(int); ok {
			ruleConfig.ReloadInterval = time.Duration(reloadInterval) * time.Second
		}
		if maxRules, ok := config.Options["max_rules"].(int); ok {
			ruleConfig.MaxRules = maxRules
		}
		if enableCaching, ok := config.Options["enable_caching"].(bool); ok {
			ruleConfig.EnableCaching = enableCaching
		}
		if cacheSize, ok := config.Options["cache_size"].(int); ok {
			ruleConfig.CacheSize = cacheSize
		}
		if cacheTTL, ok := config.Options["cache_ttl"].(int); ok {
			ruleConfig.CacheTTL = time.Duration(cacheTTL) * time.Minute
		}
	}

	// Create rule engine and manager
	ruleEngine := moderation.NewRuleEngine(ruleConfig)
	ruleManager := moderation.NewRuleManager(ruleEngine, ruleConfig)

	layer := &RulesLayer{
		name:        config.Name,
		weight:      config.Weight,
		enabled:     config.Enabled,
		config:      config,
		ruleEngine:  ruleEngine,
		ruleManager: ruleManager,
		stats:       &RulesLayerStats{},
	}

	// Load rules if rules path is configured
	if ruleConfig.RulesPath != "" {
		err := ruleManager.LoadRules()
		if err != nil {
			// Log error but don't fail layer creation
			// In production, you'd want proper logging here
		}

		// Start hot reload if enabled
		if ruleConfig.HotReload {
			err = ruleManager.StartHotReload()
			if err != nil {
				// Log error but continue
			}
		}
	}

	return layer
}

// Name returns the layer name
func (rl *RulesLayer) Name() string {
	return rl.name
}

// Weight returns the layer weight
func (rl *RulesLayer) Weight() float64 {
	return rl.weight
}

// Enabled returns whether the layer is enabled
func (rl *RulesLayer) Enabled() bool {
	return rl.enabled
}

// Config returns the layer configuration
func (rl *RulesLayer) Config() moderation.LayerConfig {
	return rl.config
}

// Moderate performs content moderation using the custom rule engine
func (rl *RulesLayer) Moderate(content string, context moderation.ModerationContext) moderation.ModerationResult {
	startTime := time.Now()
	rl.stats.TotalRequests++

	if !rl.enabled {
		return moderation.ModerationResult{
			Score:       0.0,
			Confidence:  0.0,
			Blocked:     false,
			Reason:      "Rules layer disabled",
			Category:    moderation.CategoryCustom,
			LayerName:   rl.name,
			ProcessTime: time.Since(startTime),
			Details: map[string]interface{}{
				"layer_enabled": false,
			},
		}
	}

	// Execute rules using the rule engine
	log.Printf("DEBUG RULES: About to execute rules for content: '%s'", content)
	aggregatedResult, err := rl.ruleEngine.Execute(content, context)
	if err != nil {
		log.Printf("DEBUG RULES: Rule execution failed: %v", err)
		rl.stats.FailedRuns++
		return moderation.ModerationResult{
			Score:       0.0,
			Confidence:  0.0,
			Blocked:     false,
			Reason:      "Rule execution failed: " + err.Error(),
			Category:    moderation.CategoryCustom,
			LayerName:   rl.name,
			ProcessTime: time.Since(startTime),
			Details: map[string]interface{}{
				"error":      err.Error(),
				"layer_name": rl.name,
			},
		}
	}

	rl.stats.SuccessfulRuns++
	log.Printf("DEBUG RULES: Rule execution completed. Triggered rules: %d, Action: %s, Score: %.3f", 
		len(aggregatedResult.TriggeredRules), aggregatedResult.RecommendedAction, aggregatedResult.FinalScore)

	// Update latency statistics
	latency := time.Since(startTime)
	if rl.stats.SuccessfulRuns == 1 {
		rl.stats.AverageLatency = latency
	} else {
		rl.stats.AverageLatency = time.Duration((int64(rl.stats.AverageLatency)*(rl.stats.SuccessfulRuns-1) + int64(latency)) / rl.stats.SuccessfulRuns)
	}

	// Update action statistics
	if len(aggregatedResult.TriggeredRules) > 0 {
		rl.stats.RulesTriggered += int64(len(aggregatedResult.TriggeredRules))
		
		switch aggregatedResult.RecommendedAction {
		case moderation.ActionBlock:
			rl.stats.ActionsBlocked++
		case moderation.ActionFlag:
			rl.stats.ActionsFlagged++
		case moderation.ActionWarn:
			rl.stats.ActionsWarned++
		}
	}

	// Convert aggregated result to moderation result
	return rl.convertToModerationResult(aggregatedResult, content, context, latency)
}

// convertToModerationResult converts an AggregatedRuleResult to a ModerationResult
func (rl *RulesLayer) convertToModerationResult(aggregated *moderation.AggregatedRuleResult, content string, context moderation.ModerationContext, processTime time.Duration) moderation.ModerationResult {
	// Determine if content should be blocked
	blocked := aggregated.RecommendedAction == moderation.ActionBlock

	// Build reason string
	reason := "No rules triggered"
	if len(aggregated.TriggeredRules) > 0 {
		if aggregated.RecommendedAction == "allow" {
			reason = "Rules triggered but content allowed"
		} else {
			reason = "Rules triggered: " + aggregated.RecommendedAction
		}
	}

	// Determine category
	category := aggregated.PrimaryCategory
	if category == "" {
		category = moderation.CategoryCustom
	}

	// Build details with comprehensive information
	details := map[string]interface{}{
		"layer_name":              rl.name,
		"rules_triggered":         len(aggregated.TriggeredRules),
		"recommended_action":      aggregated.RecommendedAction,
		"primary_category":        aggregated.PrimaryCategory,
		"processing_time":         aggregated.ProcessingTime.String(),
		"triggered_rule_ids":      rl.extractTriggeredRuleIDs(aggregated.TriggeredRules),
		"score_breakdown":         aggregated.ScoreBreakdown,
		"rule_engine_stats":       rl.ruleEngine.GetStats(),
	}

	// Add individual rule results for debugging
	if len(aggregated.TriggeredRules) > 0 {
		ruleResults := make([]map[string]interface{}, 0, len(aggregated.TriggeredRules))
		for _, ruleResult := range aggregated.TriggeredRules {
			ruleResults = append(ruleResults, map[string]interface{}{
				"rule_id":      ruleResult.RuleID,
				"rule_name":    ruleResult.RuleName,
				"score":        ruleResult.Score,
				"confidence":   ruleResult.Confidence,
				"action":       ruleResult.Action,
				"category":     ruleResult.Category,
				"reason":       ruleResult.Reason,
				"match_details": ruleResult.MatchDetails,
			})
		}
		details["rule_results"] = ruleResults
	}

	return moderation.ModerationResult{
		Score:       aggregated.FinalScore,
		Confidence:  aggregated.FinalConfidence,
		Blocked:     blocked,
		Reason:      reason,
		Category:    category,
		LayerName:   rl.name,
		ProcessTime: processTime,
		Details:     details,
	}
}

// extractTriggeredRuleIDs extracts rule IDs from triggered rules
func (rl *RulesLayer) extractTriggeredRuleIDs(triggeredRules []*moderation.RuleResult) []string {
	ids := make([]string, 0, len(triggeredRules))
	for _, rule := range triggeredRules {
		ids = append(ids, rule.RuleID)
	}
	return ids
}

// GetStats returns comprehensive statistics for this layer
func (rl *RulesLayer) GetStats() map[string]interface{} {
	successRate := 0.0
	if rl.stats.TotalRequests > 0 {
		successRate = float64(rl.stats.SuccessfulRuns) / float64(rl.stats.TotalRequests)
	}

	ruleEngineStats := rl.ruleEngine.GetStats()
	ruleManagerStats := rl.ruleManager.GetReloadStats()

	return map[string]interface{}{
		"name":               rl.name,
		"enabled":            rl.enabled,
		"weight":             rl.weight,
		"total_requests":     rl.stats.TotalRequests,
		"successful_runs":    rl.stats.SuccessfulRuns,
		"failed_runs":        rl.stats.FailedRuns,
		"success_rate":       successRate,
		"average_latency":    rl.stats.AverageLatency.String(),
		"rules_triggered":    rl.stats.RulesTriggered,
		"actions_blocked":    rl.stats.ActionsBlocked,
		"actions_flagged":    rl.stats.ActionsFlagged,
		"actions_warned":     rl.stats.ActionsWarned,
		"rule_engine_stats":  ruleEngineStats,
		"rule_manager_stats": ruleManagerStats,
	}
}

// LoadRules forces a reload of rules
func (rl *RulesLayer) LoadRules() error {
	return rl.ruleManager.LoadRules()
}

// ForceReload forces an immediate rule reload
func (rl *RulesLayer) ForceReload() error {
	return rl.ruleManager.ForceReload()
}

// GetRuleByID retrieves a rule by its ID
func (rl *RulesLayer) GetRuleByID(id string) (*moderation.ModerationRule, bool) {
	return rl.ruleEngine.GetRuleByID(id)
}

// GetRulesByType retrieves all rules of a specific type
func (rl *RulesLayer) GetRulesByType(ruleType string) []*moderation.ModerationRule {
	return rl.ruleEngine.GetRulesByType(ruleType)
}

// AddRule dynamically adds a new rule to the engine
func (rl *RulesLayer) AddRule(rule *moderation.ModerationRule) error {
	// Validate the rule
	err := rl.ruleEngine.GetParser().ValidateRule(rule)
	if err != nil {
		return err
	}

	// Compile the rule
	err = rl.ruleEngine.GetParser().CompileRule(rule)
	if err != nil {
		return err
	}

	// Get current rules and add the new one
	currentRules := rl.ruleEngine.GetRules()
	newRules := make([]*moderation.ModerationRule, len(currentRules)+1)
	copy(newRules, currentRules)
	newRules[len(currentRules)] = rule

	// Set the updated rules
	return rl.ruleEngine.SetRules(newRules)
}

// RemoveRule removes a rule by ID
func (rl *RulesLayer) RemoveRule(ruleID string) error {
	currentRules := rl.ruleEngine.GetRules()
	newRules := make([]*moderation.ModerationRule, 0, len(currentRules))
	
	found := false
	for _, rule := range currentRules {
		if rule.ID != ruleID {
			newRules = append(newRules, rule)
		} else {
			found = true
		}
	}

	if !found {
		return &moderation.ValidationError{
			RuleID:  ruleID,
			Field:   "id",
			Message: "rule not found",
		}
	}

	return rl.ruleEngine.SetRules(newRules)
}

// EnableRule enables a rule by ID
func (rl *RulesLayer) EnableRule(ruleID string) error {
	rule, exists := rl.ruleEngine.GetRuleByID(ruleID)
	if !exists {
		return &moderation.ValidationError{
			RuleID:  ruleID,
			Field:   "id",
			Message: "rule not found",
		}
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	return nil
}

// DisableRule disables a rule by ID
func (rl *RulesLayer) DisableRule(ruleID string) error {
	rule, exists := rl.ruleEngine.GetRuleByID(ruleID)
	if !exists {
		return &moderation.ValidationError{
			RuleID:  ruleID,
			Field:   "id",
			Message: "rule not found",
		}
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	return nil
}

