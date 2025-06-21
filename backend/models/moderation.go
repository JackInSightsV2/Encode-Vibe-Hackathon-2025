package models

import (
	"time"
)

// ModerationLog represents a moderation event log entry
type ModerationLog struct {
	ID          int        `json:"id" db:"id"`
	RequestID   *int       `json:"request_id,omitempty" db:"request_id" validate:"omitempty,gt=0"`
	UserID      *int       `json:"user_id,omitempty" db:"user_id" validate:"omitempty,gt=0"`
	SessionID   *int       `json:"session_id,omitempty" db:"session_id" validate:"omitempty,gt=0"`
	EventType   string     `json:"event_type" db:"event_type" validate:"required,max=50"`
	Severity    string     `json:"severity" db:"severity" validate:"required,oneof=low medium high critical"`
	Action      string     `json:"action" db:"action" validate:"required,oneof=block warn log allow"`
	Rule        string     `json:"rule" db:"rule" validate:"required,max=100"`
	Message     string     `json:"message" db:"message" validate:"required,max=1000"`
	Content     *string    `json:"content,omitempty" db:"content" validate:"omitempty,max=5000"`
	Metadata    *string    `json:"metadata,omitempty" db:"metadata" validate:"omitempty,max=2000"` // JSON string
	IPAddress   *string    `json:"ip_address,omitempty" db:"ip_address" validate:"omitempty,ip"`
	UserAgent   *string    `json:"user_agent,omitempty" db:"user_agent" validate:"omitempty,max=500"`
	Timestamp   time.Time  `json:"timestamp" db:"timestamp"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	ReviewedBy  *int       `json:"reviewed_by,omitempty" db:"reviewed_by" validate:"omitempty,gt=0"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	Status      string     `json:"status" db:"status" validate:"required,oneof=pending reviewed resolved ignored"`
}

// ModerationLogCreatePayload represents the payload for creating a moderation log
type ModerationLogCreatePayload struct {
	RequestID   *int    `json:"request_id,omitempty" validate:"omitempty,gt=0"`
	UserID      *int    `json:"user_id,omitempty" validate:"omitempty,gt=0"`
	SessionID   *int    `json:"session_id,omitempty" validate:"omitempty,gt=0"`
	EventType   string  `json:"event_type" validate:"required,max=50"`
	Severity    string  `json:"severity" validate:"required,oneof=low medium high critical"`
	Action      string  `json:"action" validate:"required,oneof=block warn log allow"`
	Rule        string  `json:"rule" validate:"required,max=100"`
	Message     string  `json:"message" validate:"required,max=1000"`
	Content     *string `json:"content,omitempty" validate:"omitempty,max=5000"`
	Metadata    *string `json:"metadata,omitempty" validate:"omitempty,max=2000"`
	IPAddress   *string `json:"ip_address,omitempty" validate:"omitempty,ip"`
	UserAgent   *string `json:"user_agent,omitempty" validate:"omitempty,max=500"`
}

// ModerationLogUpdatePayload represents the payload for updating a moderation log
type ModerationLogUpdatePayload struct {
	ReviewedBy *int    `json:"reviewed_by,omitempty" validate:"omitempty,gt=0"`
	Status     *string `json:"status,omitempty" validate:"omitempty,oneof=pending reviewed resolved ignored"`
}

// ModerationLogFilters represents filters for querying moderation logs
type ModerationLogFilters struct {
	RequestID   *int       `json:"request_id,omitempty" validate:"omitempty,gt=0"`
	UserID      *int       `json:"user_id,omitempty" validate:"omitempty,gt=0"`
	SessionID   *int       `json:"session_id,omitempty" validate:"omitempty,gt=0"`
	EventType   *string    `json:"event_type,omitempty" validate:"omitempty,max=50"`
	Severity    *string    `json:"severity,omitempty" validate:"omitempty,oneof=low medium high critical"`
	Action      *string    `json:"action,omitempty" validate:"omitempty,oneof=block warn log allow"`
	Rule        *string    `json:"rule,omitempty" validate:"omitempty,max=100"`
	IPAddress   *string    `json:"ip_address,omitempty" validate:"omitempty,ip"`
	Status      *string    `json:"status,omitempty" validate:"omitempty,oneof=pending reviewed resolved ignored"`
	ReviewedBy  *int       `json:"reviewed_by,omitempty" validate:"omitempty,gt=0"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Limit       int        `json:"limit" validate:"gte=1,lte=1000"`
	Offset      int        `json:"offset" validate:"gte=0"`
	OrderBy     string     `json:"order_by" validate:"omitempty,oneof=timestamp severity status"`
	OrderDir    string     `json:"order_dir" validate:"omitempty,oneof=asc desc"`
}

// ModerationStats represents aggregated moderation statistics
type ModerationStats struct {
	TotalEvents      int                    `json:"total_events"`
	EventsBySeverity map[string]int         `json:"events_by_severity"`
	EventsByAction   map[string]int         `json:"events_by_action"`
	EventsByStatus   map[string]int         `json:"events_by_status"`
	EventsByType     map[string]int         `json:"events_by_type"`
	TopRules         []RuleStats            `json:"top_rules"`
	Period           string                 `json:"period"`
	StartTime        string                 `json:"start_time"`
	EndTime          string                 `json:"end_time"`
}

// RuleStats represents statistics for a specific rule
type RuleStats struct {
	Rule  string `json:"rule"`
	Count int    `json:"count"`
}

// IsBlocked checks if the moderation action was to block
func (ml *ModerationLog) IsBlocked() bool {
	return ml.Action == "block"
}

// IsCritical checks if the severity is critical
func (ml *ModerationLog) IsCritical() bool {
	return ml.Severity == "critical"
}

// IsHigh checks if the severity is high
func (ml *ModerationLog) IsHigh() bool {
	return ml.Severity == "high"
}

// RequiresReview checks if the log entry requires manual review
func (ml *ModerationLog) RequiresReview() bool {
	return ml.Status == "pending" && (ml.IsCritical() || ml.IsHigh())
}

// MarkReviewed marks the log as reviewed by a specific user
func (ml *ModerationLog) MarkReviewed(reviewerID int, status string) {
	now := time.Now()
	ml.ReviewedBy = &reviewerID
	ml.ReviewedAt = &now
	ml.Status = status
	ml.ProcessedAt = &now
}