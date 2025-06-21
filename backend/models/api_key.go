package models

import (
	"time"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name" validate:"required,min=1,max=100"`
	KeyHash   string    `json:"-" db:"key_hash" validate:"required"`
	Active    bool      `json:"active" db:"active"`
	LastUsed  *time.Time `json:"last_used,omitempty" db:"last_used"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	
	// Additional metadata
	Permissions []string `json:"permissions,omitempty" db:"-"` // Will be stored separately if needed
	Description string   `json:"description,omitempty" db:"description" validate:"max=500"`
	UsageCount  int64    `json:"usage_count" db:"usage_count"`
}

// APIKeyCreateRequest represents the request payload for creating an API key
type APIKeyCreateRequest struct {
	Name        string     `json:"name" validate:"required,min=1,max=100"`
	Description string     `json:"description,omitempty" validate:"max=500"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
}

// APIKeyUpdateRequest represents the request payload for updating an API key
type APIKeyUpdateRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=500"`
	Active      *bool      `json:"active,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse represents the response payload for API key data (without sensitive fields)
type APIKeyResponse struct {
	ID          string     `json:"id"`
	UserID      int        `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Active      bool       `json:"active"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	Permissions []string   `json:"permissions,omitempty"`
	
	// Computed fields
	IsExpired bool `json:"is_expired"`
	DaysUntilExpiry *int `json:"days_until_expiry,omitempty"`
}

// ToResponse converts an APIKey to APIKeyResponse (removing sensitive fields)
func (ak *APIKey) ToResponse() *APIKeyResponse {
	response := &APIKeyResponse{
		ID:          ak.ID,
		UserID:      ak.UserID,
		Name:        ak.Name,
		Description: ak.Description,
		Active:      ak.Active,
		LastUsed:    ak.LastUsed,
		CreatedAt:   ak.CreatedAt,
		ExpiresAt:   ak.ExpiresAt,
		UsageCount:  ak.UsageCount,
		Permissions: ak.Permissions,
		IsExpired:   ak.IsExpired(),
	}
	
	// Calculate days until expiry
	if ak.ExpiresAt != nil && !ak.IsExpired() {
		days := int(time.Until(*ak.ExpiresAt).Hours() / 24)
		response.DaysUntilExpiry = &days
	}
	
	return response
}

// IsExpired checks if the API key has expired
func (ak *APIKey) IsExpired() bool {
	return ak.ExpiresAt != nil && time.Now().After(*ak.ExpiresAt)
}

// IsValid checks if the API key is active and not expired
func (ak *APIKey) IsValid() bool {
	return ak.Active && !ak.IsExpired()
}

// MaskKey returns a masked version of the API key for display purposes
func (ak *APIKey) MaskKey(key string) string {
	if len(key) < 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// APIKeyUsage represents usage statistics for an API key
type APIKeyUsage struct {
	APIKeyID    string    `json:"api_key_id" db:"api_key_id"`
	Date        time.Time `json:"date" db:"date"`
	RequestCount int64     `json:"request_count" db:"request_count"`
	LastRequest time.Time `json:"last_request" db:"last_request"`
	IPAddresses []string  `json:"ip_addresses,omitempty" db:"-"`
	UserAgents  []string  `json:"user_agents,omitempty" db:"-"`
}

// APIKeyStats represents aggregated statistics for API keys
type APIKeyStats struct {
	TotalKeys       int64 `json:"total_keys"`
	ActiveKeys      int64 `json:"active_keys"`
	ExpiredKeys     int64 `json:"expired_keys"`
	DisabledKeys    int64 `json:"disabled_keys"`
	KeysUsedToday   int64 `json:"keys_used_today"`
	TotalRequests   int64 `json:"total_requests"`
	RequestsToday   int64 `json:"requests_today"`
}

// APIKeyListResponse represents the response for listing API keys
type APIKeyListResponse struct {
	Keys       []*APIKeyResponse `json:"keys"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// APIKeyCreateResponse represents the response when creating an API key
type APIKeyCreateResponse struct {
	APIKey    *APIKeyResponse `json:"api_key"`
	PlainKey  string          `json:"plain_key"` // Only returned once during creation
	Warning   string          `json:"warning"`
}

// APIKeyFilters represents filters for API key queries
type APIKeyFilters struct {
	UserID     *int       `json:"user_id,omitempty"`
	Active     *bool      `json:"active,omitempty"`
	Expired    *bool      `json:"expired,omitempty"`
	UsedSince  *time.Time `json:"used_since,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	Search     string     `json:"search,omitempty"` // Search in name and description
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	SortBy     string     `json:"sort_by,omitempty"` // name, created_at, last_used, usage_count
	SortOrder  string     `json:"sort_order,omitempty"` // asc, desc
}