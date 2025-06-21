package moderation

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// RuleEngine manages and executes custom moderation rules
type RuleEngine struct {
	rules         []*ModerationRule
	parser        *RuleParser
	executor      *RuleExecutor
	config        *RuleEngineConfig
	rulesByID     map[string]*ModerationRule
	rulesByType   map[string][]*ModerationRule
	mutex         sync.RWMutex
	lastReload    time.Time
	stats         *RuleEngineStats
}

// ModerationRule represents a single moderation rule
type ModerationRule struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Type        string                 `yaml:"type"` // regex, keyword, ml, composite
	Pattern     string                 `yaml:"pattern"`
	Keywords    []string               `yaml:"keywords,omitempty"`
	Weight      float64               `yaml:"weight"`
	Priority    int                   `yaml:"priority"`
	Action      string                 `yaml:"action"` // block, flag, warn
	Category    string                 `yaml:"category"`
	Enabled     bool                  `yaml:"enabled"`
	Context     *RuleContext          `yaml:"context,omitempty"`
	Conditions  *RuleConditions       `yaml:"conditions,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty"`
	CreatedAt   time.Time             `yaml:"created_at"`
	UpdatedAt   time.Time             `yaml:"updated_at"`
	// Compiled fields (not in YAML)
	compiledPattern *regexp.Regexp `yaml:"-"`
	normalizedKeywords []string    `yaml:"-"`
}

// RuleContext defines when and where a rule applies
type RuleContext struct {
	UserTypes     []string          `yaml:"user_types,omitempty"`     // admin, user, guest
	ContentTypes  []string          `yaml:"content_types,omitempty"`  // message, post, comment
	Channels      []string          `yaml:"channels,omitempty"`       // specific channels
	TimeRanges    []TimeRange       `yaml:"time_ranges,omitempty"`    // time-based rules
	UserAttributes map[string]string `yaml:"user_attributes,omitempty"` // custom user attributes
}

// TimeRange defines time-based rule activation
type TimeRange struct {
	Start    string `yaml:"start"`    // "09:00"
	End      string `yaml:"end"`      // "17:00"
	Timezone string `yaml:"timezone"` // "UTC", "America/New_York"
	Days     []string `yaml:"days"`   // ["monday", "tuesday"]
}

// RuleConditions defines logical conditions for rule execution
type RuleConditions struct {
	MinLength    *int     `yaml:"min_length,omitempty"`
	MaxLength    *int     `yaml:"max_length,omitempty"`
	RequireAll   bool     `yaml:"require_all,omitempty"`   // AND vs OR for keywords
	CaseSensitive bool    `yaml:"case_sensitive,omitempty"`
	WholeWords   bool     `yaml:"whole_words,omitempty"`
	ExcludeUsers []string `yaml:"exclude_users,omitempty"`
	IncludeUsers []string `yaml:"include_users,omitempty"`
}

// RuleResult represents the result of rule execution
type RuleResult struct {
	RuleID      string
	RuleName    string
	Matched     bool
	Score       float64
	Confidence  float64
	Action      string
	Category    string
	Reason      string
	Context     map[string]interface{}
	MatchDetails *MatchDetails
}

// MatchDetails provides detailed information about what matched
type MatchDetails struct {
	MatchedText     string   `json:"matched_text"`
	MatchedKeywords []string `json:"matched_keywords,omitempty"`
	Position        int      `json:"position"`
	Length          int      `json:"length"`
	Context         string   `json:"context"` // surrounding text
}

// RuleEngineConfig holds configuration for the rule engine
type RuleEngineConfig struct {
	RulesPath        string        `yaml:"rules_path"`
	HotReload        bool          `yaml:"hot_reload"`
	ReloadInterval   time.Duration `yaml:"reload_interval"`
	MaxRules         int           `yaml:"max_rules"`
	DefaultWeight    float64       `yaml:"default_weight"`
	DefaultPriority  int           `yaml:"default_priority"`
	EnableCaching    bool          `yaml:"enable_caching"`
	CacheSize        int           `yaml:"cache_size"`
	CacheTTL         time.Duration `yaml:"cache_ttl"`
}

// RuleEngineStats tracks rule engine performance
type RuleEngineStats struct {
	TotalRules       int64
	ActiveRules      int64
	RulesExecuted    int64
	RulesMatched     int64
	AverageExecTime  time.Duration
	LastReload       time.Time
	ReloadCount      int64
	ErrorCount       int64
	RuleTypeStats    map[string]*RuleTypeStats
	mutex           sync.RWMutex
}

// RuleTypeStats tracks statistics per rule type
type RuleTypeStats struct {
	Count       int64
	Executions  int64
	Matches     int64
	AvgExecTime time.Duration
	ErrorCount  int64
}

// RuleParser handles parsing and validation of YAML rules
type RuleParser struct {
	config *RuleEngineConfig
}

// RuleExecutor handles rule execution logic
type RuleExecutor struct {
	config *RuleEngineConfig
	cache  *RuleCache
}

// RuleCache provides caching for rule execution results
type RuleCache struct {
	cache     map[string]*RuleCacheEntry
	maxSize   int
	ttl       time.Duration
	mutex     sync.RWMutex
}

// RuleCacheEntry represents a cached rule execution result
type RuleCacheEntry struct {
	Result    *RuleResult
	Timestamp time.Time
}

// Rule type constants
const (
	RuleTypeRegex     = "regex"
	RuleTypeKeyword   = "keyword"
	RuleTypeML        = "ml"
	RuleTypeComposite = "composite"
)

// Action constants for rules (different from main moderation actions)
const (
	RuleActionBlock = "block"
	RuleActionFlag  = "flag"
	RuleActionWarn  = "warn"
)

// NewRuleEngine creates a new rule engine instance
func NewRuleEngine(config *RuleEngineConfig) *RuleEngine {
	engine := &RuleEngine{
		rules:       make([]*ModerationRule, 0),
		config:      config,
		rulesByID:   make(map[string]*ModerationRule),
		rulesByType: make(map[string][]*ModerationRule),
		parser:      NewRuleParser(config),
		executor:    NewRuleExecutor(config),
		stats: &RuleEngineStats{
			RuleTypeStats: make(map[string]*RuleTypeStats),
		},
	}

	// Initialize rule type stats
	for _, ruleType := range []string{RuleTypeRegex, RuleTypeKeyword, RuleTypeML, RuleTypeComposite} {
		engine.stats.RuleTypeStats[ruleType] = &RuleTypeStats{}
	}

	return engine
}

// Execute runs all applicable rules against content and returns aggregated results
func (re *RuleEngine) Execute(content string, context ModerationContext) (*AggregatedRuleResult, error) {
	startTime := time.Now()
	
	re.mutex.RLock()
	activeRules := make([]*ModerationRule, 0)
	for _, rule := range re.rules {
		if rule.Enabled {
			activeRules = append(activeRules, rule)
		}
	}
	re.mutex.RUnlock()

	// Execute rules
	results, err := re.executor.ExecuteRules(content, context, activeRules)
	if err != nil {
		re.updateStats(func(s *RuleEngineStats) { s.ErrorCount++ })
		return nil, fmt.Errorf("rule execution failed: %w", err)
	}

	// Update execution stats
	re.updateStats(func(s *RuleEngineStats) {
		s.RulesExecuted += int64(len(activeRules))
		matchedCount := int64(0)
		for _, result := range results {
			if result.Matched {
				matchedCount++
			}
		}
		s.RulesMatched += matchedCount
		
		execTime := time.Since(startTime)
		if s.RulesExecuted == 1 {
			s.AverageExecTime = execTime
		} else {
			s.AverageExecTime = time.Duration((int64(s.AverageExecTime)*(s.RulesExecuted-1) + int64(execTime)) / s.RulesExecuted)
		}
	})

	// Create score calculator and aggregate results
	calculator := NewRuleScoreCalculator(ScoreStrategyWeighted)
	aggregated := calculator.AggregateRuleResults(results)
	
	return aggregated, nil
}

// GetParser returns the rule parser instance
func (re *RuleEngine) GetParser() *RuleParser {
	return re.parser
}

// GetRules returns all loaded rules
func (re *RuleEngine) GetRules() []*ModerationRule {
	re.mutex.RLock()
	defer re.mutex.RUnlock()
	
	rules := make([]*ModerationRule, len(re.rules))
	copy(rules, re.rules)
	return rules
}

// GetRuleByID retrieves a rule by its ID
func (re *RuleEngine) GetRuleByID(id string) (*ModerationRule, bool) {
	re.mutex.RLock()
	defer re.mutex.RUnlock()
	
	rule, exists := re.rulesByID[id]
	return rule, exists
}

// GetRulesByType retrieves all rules of a specific type
func (re *RuleEngine) GetRulesByType(ruleType string) []*ModerationRule {
	re.mutex.RLock()
	defer re.mutex.RUnlock()
	
	rules, exists := re.rulesByType[ruleType]
	if !exists {
		return []*ModerationRule{}
	}
	
	// Return a copy to prevent external modification
	result := make([]*ModerationRule, len(rules))
	copy(result, rules)
	return result
}

// GetStats returns comprehensive statistics about the rule engine
func (re *RuleEngine) GetStats() map[string]interface{} {
	re.stats.mutex.RLock()
	defer re.stats.mutex.RUnlock()

	typeStats := make(map[string]interface{})
	for ruleType, stats := range re.stats.RuleTypeStats {
		typeStats[ruleType] = map[string]interface{}{
			"count":           stats.Count,
			"executions":      stats.Executions,
			"matches":         stats.Matches,
			"avg_exec_time":   stats.AvgExecTime.String(),
			"error_count":     stats.ErrorCount,
		}
	}

	return map[string]interface{}{
		"total_rules":       re.stats.TotalRules,
		"active_rules":      re.stats.ActiveRules,
		"rules_executed":    re.stats.RulesExecuted,
		"rules_matched":     re.stats.RulesMatched,
		"average_exec_time": re.stats.AverageExecTime.String(),
		"last_reload":       re.stats.LastReload,
		"reload_count":      re.stats.ReloadCount,
		"error_count":       re.stats.ErrorCount,
		"rule_type_stats":   typeStats,
	}
}

// updateStats safely updates rule engine statistics
func (re *RuleEngine) updateStats(update func(*RuleEngineStats)) {
	re.stats.mutex.Lock()
	defer re.stats.mutex.Unlock()
	update(re.stats)
}

// NewRuleParser creates a new rule parser
func NewRuleParser(config *RuleEngineConfig) *RuleParser {
	return &RuleParser{
		config: config,
	}
}

// NewRuleExecutor creates a new rule executor
func NewRuleExecutor(config *RuleEngineConfig) *RuleExecutor {
	var cache *RuleCache
	if config.EnableCaching {
		cache = &RuleCache{
			cache:   make(map[string]*RuleCacheEntry),
			maxSize: config.CacheSize,
			ttl:     config.CacheTTL,
		}
	}

	return &RuleExecutor{
		config: config,
		cache:  cache,
	}
}

// Example YAML rule definitions that this engine supports
const ExampleRulesYAML = `
# Violence Detection Rules
rules:
  - id: "violence_threats_001"
    name: "Direct Violence Threats"
    description: "Detects direct threats of violence against individuals"
    type: "regex"
    pattern: "(?i)\\b(kill|murder|destroy|eliminate)\\s+(you|him|her|them)\\b"
    weight: 0.9
    priority: 1
    action: "block"
    category: "violence"
    enabled: true
    context:
      content_types: ["message", "comment"]
      user_types: ["user", "guest"]
    conditions:
      case_sensitive: false
      whole_words: true
    metadata:
      severity: "high"
      review_required: true

  - id: "hate_speech_001"
    name: "Racial Slurs"
    description: "Detects racial slurs and hate speech"
    type: "keyword"
    keywords: ["slur1", "slur2", "hate_term"]
    weight: 0.95
    priority: 1
    action: "block"
    category: "hate_speech"
    enabled: true
    conditions:
      case_sensitive: false
      whole_words: true
      exclude_users: ["admin", "moderator"]

  - id: "spam_detection_001"
    name: "Promotional Spam"
    description: "Detects promotional spam content"
    type: "composite"
    weight: 0.6
    priority: 3
    action: "flag"
    category: "spam"
    enabled: true
    conditions:
      min_length: 20
    metadata:
      requires_review: true
      auto_remove: false

  - id: "ml_toxicity_001"
    name: "ML Toxicity Detection"
    description: "Machine learning based toxicity detection"
    type: "ml"
    pattern: "toxicity_model_v1"
    weight: 0.8
    priority: 2
    action: "flag"
    category: "toxicity"
    enabled: true
    context:
      time_ranges:
        - start: "00:00"
          end: "23:59"
          timezone: "UTC"
          days: ["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"]

# Context-Aware Rules
  - id: "work_hours_relaxed"
    name: "Relaxed Moderation During Work Hours"
    description: "More permissive rules during business hours"
    type: "keyword"
    keywords: ["mild_profanity"]
    weight: 0.3
    priority: 5
    action: "warn"
    category: "profanity"
    enabled: true
    context:
      time_ranges:
        - start: "09:00"
          end: "17:00"
          timezone: "America/New_York"
          days: ["monday", "tuesday", "wednesday", "thursday", "friday"]
`