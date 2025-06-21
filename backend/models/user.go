package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username" validate:"required,min=3,max=50,alphanum"`
	Email        string     `json:"email" db:"email" validate:"required,email,max=255"`
	PasswordHash string     `json:"-" db:"password_hash" validate:"required,min=60"`
	Role         string     `json:"role" db:"role" validate:"required,oneof=admin user moderator"`
	APIKey       *string    `json:"api_key,omitempty" db:"api_key" validate:"omitempty,len=32"`
	Active       bool       `json:"active" db:"active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// UserCreateRequest represents the request payload for creating a user
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
	Role     string `json:"role" validate:"required,oneof=admin user moderator"`
}

// UserUpdateRequest represents the request payload for updating a user
type UserUpdateRequest struct {
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=50,alphanum"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=admin user moderator"`
	Active   *bool   `json:"active,omitempty"`
}

// UserLoginRequest represents the request payload for user login
type UserLoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// UserResponse represents the response payload for user data (without sensitive fields)
type UserResponse struct {
	ID          int        `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// ToResponse converts a User to UserResponse (removing sensitive fields)
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Role:        u.Role,
		Active:      u.Active,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

// HasRole checks if the user has the specified role
func (u *User) HasRole(role string) bool {
	return u.Role == role
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

// IsModerator checks if the user is a moderator
func (u *User) IsModerator() bool {
	return u.HasRole("moderator")
}

// CanModerate checks if the user can perform moderation actions
func (u *User) CanModerate() bool {
	return u.IsAdmin() || u.IsModerator()
}

// OAuth2Account represents an OAuth2 account linked to a user
type OAuth2Account struct {
	ID             int       `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	Provider       string    `json:"provider" db:"provider" validate:"required,max=50"`
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id" validate:"required,max=255"`
	Email          string    `json:"email" db:"email" validate:"required,email,max=255"`
	Name           string    `json:"name" db:"name" validate:"max=255"`
	Picture        *string   `json:"picture,omitempty" db:"picture" validate:"omitempty,url,max=500"`
	AccessToken    *string   `json:"-" db:"access_token"`
	RefreshToken   *string   `json:"-" db:"refresh_token"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Scope          *string   `json:"scope,omitempty" db:"scope" validate:"omitempty,max=500"`
	LinkedAt       time.Time `json:"linked_at" db:"linked_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	Active         bool      `json:"active" db:"active"`
}

// OAuth2AccountResponse represents the response payload for OAuth2 account data
type OAuth2AccountResponse struct {
	ID             int        `json:"id"`
	Provider       string     `json:"provider"`
	ProviderUserID string     `json:"provider_user_id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	Picture        *string    `json:"picture,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LinkedAt       time.Time  `json:"linked_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	Active         bool       `json:"active"`
}

// ToResponse converts an OAuth2Account to OAuth2AccountResponse (removing sensitive fields)
func (oa *OAuth2Account) ToResponse() *OAuth2AccountResponse {
	return &OAuth2AccountResponse{
		ID:             oa.ID,
		Provider:       oa.Provider,
		ProviderUserID: oa.ProviderUserID,
		Email:          oa.Email,
		Name:           oa.Name,
		Picture:        oa.Picture,
		ExpiresAt:      oa.ExpiresAt,
		LinkedAt:       oa.LinkedAt,
		LastUsedAt:     oa.LastUsedAt,
		Active:         oa.Active,
	}
}

// IsExpired checks if the OAuth2 access token is expired
func (oa *OAuth2Account) IsExpired() bool {
	return oa.ExpiresAt != nil && time.Now().After(*oa.ExpiresAt)
}

// NeedsRefresh checks if the OAuth2 token needs to be refreshed
func (oa *OAuth2Account) NeedsRefresh() bool {
	if oa.ExpiresAt == nil {
		return false
	}
	// Refresh if token expires within 5 minutes
	return time.Now().Add(5 * time.Minute).After(*oa.ExpiresAt)
}