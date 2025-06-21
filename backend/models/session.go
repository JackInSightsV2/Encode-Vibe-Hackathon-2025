package models

import (
	"time"
)

// Session represents a user session in the system
type Session struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id" validate:"required,gt=0"`
	Token      string    `json:"-" db:"token" validate:"required,len=64"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at" validate:"required"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	LastUsedAt time.Time `json:"last_used_at" db:"last_used_at"`
	IPAddress  string    `json:"ip_address" db:"ip_address" validate:"omitempty,ip"`
	UserAgent  string    `json:"user_agent" db:"user_agent" validate:"omitempty,max=500"`
	Active     bool      `json:"active" db:"active"`
}

// SessionCreateRequest represents the request payload for creating a session
type SessionCreateRequest struct {
	UserID    int    `json:"user_id" validate:"required,gt=0"`
	IPAddress string `json:"ip_address" validate:"omitempty,ip"`
	UserAgent string `json:"user_agent" validate:"omitempty,max=500"`
}

// SessionResponse represents the response payload for session data
type SessionResponse struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Active     bool      `json:"active"`
}

// ToResponse converts a Session to SessionResponse (removing sensitive fields)
func (s *Session) ToResponse() *SessionResponse {
	return &SessionResponse{
		ID:         s.ID,
		UserID:     s.UserID,
		ExpiresAt:  s.ExpiresAt,
		CreatedAt:  s.CreatedAt,
		LastUsedAt: s.LastUsedAt,
		IPAddress:  s.IPAddress,
		UserAgent:  s.UserAgent,
		Active:     s.Active,
	}
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid (active and not expired)
func (s *Session) IsValid() bool {
	return s.Active && !s.IsExpired()
}

// Refresh updates the last used time and optionally extends the expiration
func (s *Session) Refresh(extendBy time.Duration) {
	s.LastUsedAt = time.Now()
	if extendBy > 0 {
		s.ExpiresAt = time.Now().Add(extendBy)
	}
}