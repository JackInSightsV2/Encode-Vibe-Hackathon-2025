package models

import (
	"time"
)

// KillSwitchEntry represents a kill switch entry for blocking users or sessions
type KillSwitchEntry struct {
	ID          int        `json:"id" db:"id"`
	EntryType   string     `json:"entry_type" db:"entry_type" validate:"required,oneof=user session ip"`
	Value       string     `json:"value" db:"value" validate:"required,max=255"`
	Reason      string     `json:"reason" db:"reason" validate:"required,max=500"`
	CreatedBy   int        `json:"created_by" db:"created_by" validate:"required,gt=0"`
	Active      bool       `json:"active" db:"active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	LastHit     *time.Time `json:"last_hit,omitempty" db:"last_hit"`
	HitCount    int        `json:"hit_count" db:"hit_count" validate:"gte=0"`
	Metadata    *string    `json:"metadata,omitempty" db:"metadata" validate:"omitempty,max=1000"` // JSON string
	ReviewedBy  *int       `json:"reviewed_by,omitempty" db:"reviewed_by" validate:"omitempty,gt=0"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	Notes       *string    `json:"notes,omitempty" db:"notes" validate:"omitempty,max=1000"`
}

// KillSwitchCreatePayload represents the payload for creating a kill switch entry
type KillSwitchCreatePayload struct {
	EntryType string     `json:"entry_type" validate:"required,oneof=user session ip"`
	Value     string     `json:"value" validate:"required,max=255"`
	Reason    string     `json:"reason" validate:"required,max=500"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Metadata  *string    `json:"metadata,omitempty" validate:"omitempty,max=1000"`
	Notes     *string    `json:"notes,omitempty" validate:"omitempty,max=1000"`
}

// KillSwitchUpdatePayload represents the payload for updating a kill switch entry
type KillSwitchUpdatePayload struct {
	Reason     *string    `json:"reason,omitempty" validate:"omitempty,max=500"`
	Active     *bool      `json:"active,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Notes      *string    `json:"notes,omitempty" validate:"omitempty,max=1000"`
	ReviewedBy *int       `json:"reviewed_by,omitempty" validate:"omitempty,gt=0"`
}

// KillSwitchFilters represents filters for querying kill switch entries
type KillSwitchFilters struct {
	EntryType    *string    `json:"entry_type,omitempty" validate:"omitempty,oneof=user session ip"`
	Value        *string    `json:"value,omitempty" validate:"omitempty,max=255"`
	ValuePattern *string    `json:"value_pattern,omitempty" validate:"omitempty,max=255"`
	Active       *bool      `json:"active,omitempty"`
	CreatedBy    *int       `json:"created_by,omitempty" validate:"omitempty,gt=0"`
	ReviewedBy   *int       `json:"reviewed_by,omitempty" validate:"omitempty,gt=0"`
	ExpiresAfter *time.Time `json:"expires_after,omitempty"`
	ExpiresBefore *time.Time `json:"expires_before,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Limit        int        `json:"limit" validate:"gte=1,lte=1000"`
	Offset       int        `json:"offset" validate:"gte=0"`
	OrderBy      string     `json:"order_by" validate:"omitempty,oneof=created_at updated_at last_hit hit_count"`
	OrderDir     string     `json:"order_dir" validate:"omitempty,oneof=asc desc"`
}

// KillSwitchStats represents kill switch statistics
type KillSwitchStats struct {
	TotalEntries    int                    `json:"total_entries"`
	ActiveEntries   int                    `json:"active_entries"`
	ExpiredEntries  int                    `json:"expired_entries"`
	EntriesByType   map[string]int         `json:"entries_by_type"`
	TotalHits       int                    `json:"total_hits"`
	RecentHits      int                    `json:"recent_hits"` // Last 24 hours
	TopBlockedItems []KillSwitchRanking    `json:"top_blocked_items"`
	Period          string                 `json:"period"`
	StartTime       string                 `json:"start_time"`
	EndTime         string                 `json:"end_time"`
}

// KillSwitchRanking represents a kill switch entry with ranking information
type KillSwitchRanking struct {
	EntryType string    `json:"entry_type"`
	Value     string    `json:"value"`
	Reason    string    `json:"reason"`
	HitCount  int       `json:"hit_count"`
	LastHit   *time.Time `json:"last_hit,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// IsActive checks if the kill switch entry is currently active
func (ks *KillSwitchEntry) IsActive() bool {
	if !ks.Active {
		return false
	}
	
	// Check if it has expired
	if ks.ExpiresAt != nil && time.Now().After(*ks.ExpiresAt) {
		return false
	}
	
	return true
}

// IsExpired checks if the kill switch entry has expired
func (ks *KillSwitchEntry) IsExpired() bool {
	return ks.ExpiresAt != nil && time.Now().After(*ks.ExpiresAt)
}

// IsUserType checks if this is a user-type kill switch
func (ks *KillSwitchEntry) IsUserType() bool {
	return ks.EntryType == "user"
}

// IsSessionType checks if this is a session-type kill switch
func (ks *KillSwitchEntry) IsSessionType() bool {
	return ks.EntryType == "session"
}

// IsIPType checks if this is an IP-type kill switch
func (ks *KillSwitchEntry) IsIPType() bool {
	return ks.EntryType == "ip"
}

// RecordHit records a hit on this kill switch entry
func (ks *KillSwitchEntry) RecordHit() {
	now := time.Now()
	ks.LastHit = &now
	ks.HitCount++
	ks.UpdatedAt = now
}

// GetTimeUntilExpiration returns the duration until expiration
func (ks *KillSwitchEntry) GetTimeUntilExpiration() *time.Duration {
	if ks.ExpiresAt == nil {
		return nil
	}
	
	duration := time.Until(*ks.ExpiresAt)
	return &duration
}

// ShouldBlock checks if this entry should block the given value
func (ks *KillSwitchEntry) ShouldBlock(entryType, value string) bool {
	if !ks.IsActive() {
		return false
	}
	
	if ks.EntryType != entryType {
		return false
	}
	
	return ks.Value == value
}

// Deactivate deactivates the kill switch entry
func (ks *KillSwitchEntry) Deactivate() {
	ks.Active = false
	ks.UpdatedAt = time.Now()
}

// Extend extends the expiration time by the given duration
func (ks *KillSwitchEntry) Extend(duration time.Duration) {
	if ks.ExpiresAt == nil {
		// If no expiration is set, set it to now + duration
		expiresAt := time.Now().Add(duration)
		ks.ExpiresAt = &expiresAt
	} else {
		// Extend the existing expiration
		*ks.ExpiresAt = ks.ExpiresAt.Add(duration)
	}
	ks.UpdatedAt = time.Now()
}