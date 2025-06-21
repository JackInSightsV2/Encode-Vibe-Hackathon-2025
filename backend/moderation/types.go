package moderation

import (
	"time"
)

// ModerationContext provides context information for moderation
type ModerationContext struct {
	UserID        string                 `json:"user_id"`
	SessionID     string                 `json:"session_id"`
	RequestID     string                 `json:"request_id"`
	Timestamp     time.Time              `json:"timestamp"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	UserType      string                 `json:"user_type"`      // admin, user, guest, etc.
	ContentType   string                 `json:"content_type"`   // message, post, comment, etc.
	Channel       string                 `json:"channel"`        // specific channel/room
	ConversationHistory []string         `json:"conversation_history,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ModerationResult represents the result of a moderation check
type ModerationResult struct {
	Score       float64                `json:"score"`        // 0.0 (safe) to 1.0 (unsafe)
	Confidence  float64                `json:"confidence"`   // 0.0 to 1.0
	Blocked     bool                   `json:"blocked"`      // Final decision
	Reason      string                 `json:"reason"`       // Human-readable reason
	Category    string                 `json:"category"`     // e.g., "toxicity", "prompt_injection"
	LayerName   string                 `json:"layer_name"`   // Which layer detected this
	Details     map[string]interface{} `json:"details"`      // Layer-specific details
	ProcessTime time.Duration          `json:"process_time"` // Time taken to process
	
	// Additional fields needed for compatibility
	Metadata       map[string]interface{} `json:"metadata,omitempty"`       // Additional metadata
	ProcessingTime time.Duration          `json:"processing_time"`          // Alternative field for process time
	Categories     map[string]float64     `json:"categories,omitempty"`     // Category scores
	Scores         map[string]float64     `json:"scores,omitempty"`         // Multiple scores
	Flagged        bool                   `json:"flagged"`                  // Alternative to Blocked
	ProcessedAt    time.Time              `json:"processed_at"`             // When processed
}

// AggregatedResult represents the final result from all layers
type AggregatedResult struct {
	FinalScore     float64            `json:"final_score"`
	FinalDecision  bool               `json:"final_decision"`
	Action         string             `json:"action"`           // log, flag, block, block_and_alert
	Severity       string             `json:"severity"`         // low, medium, high, critical
	LayerResults   []ModerationResult `json:"layer_results"`
	ProcessTime    time.Duration      `json:"total_process_time"`
	CacheHit       bool               `json:"cache_hit"`
	RequestContext ModerationContext  `json:"context"`
}

// ModerationLayer interface that all moderation layers must implement
type ModerationLayer interface {
	// Name returns the unique name of this layer
	Name() string
	
	// Moderate performs moderation on the given content
	Moderate(content string, context ModerationContext) ModerationResult
	
	// Weight returns the weight of this layer in the final score (0.0 to 1.0)
	Weight() float64
	
	// Enabled returns whether this layer is currently enabled
	Enabled() bool
	
	// Config returns the configuration for this layer
	Config() LayerConfig
}

// LayerConfig represents configuration for a moderation layer
type LayerConfig struct {
	Name      string                 `yaml:"name" json:"name"`
	Enabled   bool                   `yaml:"enabled" json:"enabled"`
	Weight    float64                `yaml:"weight" json:"weight"`
	Threshold float64                `yaml:"threshold,omitempty" json:"threshold,omitempty"`
	Options   map[string]interface{} `yaml:"options,omitempty" json:"options,omitempty"`
}

// ModerationThresholds defines action thresholds
type ModerationThresholds struct {
	Low      float64 `yaml:"low" json:"low"`           // 0.3
	Medium   float64 `yaml:"medium" json:"medium"`     // 0.6
	High     float64 `yaml:"high" json:"high"`         // 0.8
	Critical float64 `yaml:"critical" json:"critical"` // 0.95
}

// ActionConfig defines what action to take at each threshold
type ActionConfig struct {
	Low      string `yaml:"low" json:"low"`           // "log"
	Medium   string `yaml:"medium" json:"medium"`     // "flag"
	High     string `yaml:"high" json:"high"`         // "block"
	Critical string `yaml:"critical" json:"critical"` // "block_and_alert"
}

// CacheConfig defines caching behavior
type CacheConfig struct {
	Enabled    bool `yaml:"enabled" json:"enabled"`
	TTLMinutes int  `yaml:"ttl_minutes" json:"ttl_minutes"`
	MaxEntries int  `yaml:"max_entries" json:"max_entries"`
}

// AdvancedModerationConfig represents the configuration for advanced moderation
type AdvancedModerationConfig struct {
	Enabled    bool              `yaml:"enabled" json:"enabled"`
	Layers     []LayerConfig     `yaml:"layers" json:"layers"`
	Thresholds ModerationThresholds   `yaml:"thresholds" json:"thresholds"`
	Actions    ActionConfig      `yaml:"actions" json:"actions"`
	Cache      CacheConfig       `yaml:"cache" json:"cache"`
	Analytics  AnalyticsConfig   `yaml:"analytics" json:"analytics"`
}

// AnalyticsConfig defines analytics collection settings
type AnalyticsConfig struct {
	Enabled           bool `yaml:"enabled" json:"enabled"`
	CollectDetails    bool `yaml:"collect_details" json:"collect_details"`
	RetentionDays     int  `yaml:"retention_days" json:"retention_days"`
	EnablePerformance bool `yaml:"enable_performance" json:"enable_performance"`
}

// CacheEntry represents a cached moderation result
type CacheEntry struct {
	Result    AggregatedResult `json:"result"`
	Timestamp time.Time        `json:"timestamp"`
	TTL       time.Time        `json:"ttl"`
}

// ModerationStats tracks performance and effectiveness metrics
type ModerationStats struct {
	TotalRequests     int64         `json:"total_requests"`
	BlockedRequests   int64         `json:"blocked_requests"`
	FlaggedRequests   int64         `json:"flagged_requests"`
	CacheHits         int64         `json:"cache_hits"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	LayerStats        map[string]LayerStats `json:"layer_stats"`
}

// LayerStats tracks per-layer statistics
type LayerStats struct {
	LayerName         string        `json:"layer_name"`
	TotalProcessed    int64         `json:"total_processed"`
	Detections        int64         `json:"detections"`
	AverageScore      float64       `json:"average_score"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	ErrorCount        int64         `json:"error_count"`
}

// Constants for actions and severities
const (
	ActionLog         = "log"
	ActionFlag        = "flag"
	ActionBlock       = "block"
	ActionBlockAlert  = "block_and_alert"
	ActionWarn        = "warn"
	
	SeverityLow       = "low"
	SeverityMedium    = "medium"
	SeverityHigh      = "high"
	SeverityCritical  = "critical"
	
	CategoryToxicity     = "toxicity"
	CategoryPromptInject = "prompt_injection"
	CategoryPII          = "pii"
	CategorySpam         = "spam"
	CategoryHateSpeech   = "hate_speech"
	CategoryViolence     = "violence"
	CategoryCustom       = "custom_rule"
)

// PIIResult represents the result of PII detection
type PIIResult struct {
	PIIFound    bool           `json:"pii_found"`
	PIITypes    []string       `json:"pii_types"`
	PIICount    int            `json:"pii_count"`
	Detections  []PIIDetection `json:"detections"`
	Confidence  float64        `json:"confidence"`
	ProcessedAt time.Time      `json:"processed_at"`
	MaskedText  string         `json:"masked_text,omitempty"`
	
	// For compatibility with ModerationResult
	Blocked        bool                   `json:"blocked"`
	Reason         string                 `json:"reason"`
	Scores         map[string]float64     `json:"scores,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PIIDetection is defined in pii.go to avoid duplication