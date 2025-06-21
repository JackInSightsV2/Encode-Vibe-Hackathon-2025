package auth

import (
	"time"
	"qt1-middleware/models"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Authentication
	Login(username, password string) (*AuthResult, error)
	Logout(userID int) error
	Register(req *RegisterRequest) (*models.User, error)
	
	// Token management
	GenerateToken(user *models.User) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
	RefreshToken(tokenString string) (string, error)
	GetUserByToken(tokenString string) (*models.User, error)
	
	// Password management
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) error
	ChangePassword(userID int, oldPassword, newPassword string) error
	
	// UserService interface for OAuth2 compatibility
	UserService
}

// Claims represents JWT token claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IssuedAt int64  `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

// AuthResult represents the result of a successful authentication
type AuthResult struct {
	User         *models.User `json:"user"`
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role,omitempty" validate:"omitempty,oneof=admin user moderator"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// TokenRefreshRequest represents a token refresh request
type TokenRefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret         string        `yaml:"jwt_secret" json:"jwt_secret"`
	TokenExpiration   time.Duration `yaml:"token_expiration" json:"token_expiration"`
	RefreshExpiration time.Duration `yaml:"refresh_expiration" json:"refresh_expiration"`
	
	// Password requirements
	MinPasswordLength    int  `yaml:"min_password_length" json:"min_password_length"`
	RequireSpecialChar   bool `yaml:"require_special_char" json:"require_special_char"`
	RequireNumber        bool `yaml:"require_number" json:"require_number"`
	RequireUppercase     bool `yaml:"require_uppercase" json:"require_uppercase"`
	RequireLowercase     bool `yaml:"require_lowercase" json:"require_lowercase"`
	
	// Security settings
	MaxLoginAttempts     int           `yaml:"max_login_attempts" json:"max_login_attempts"`
	LockoutDuration      time.Duration `yaml:"lockout_duration" json:"lockout_duration"`
	BcryptCost          int           `yaml:"bcrypt_cost" json:"bcrypt_cost"`
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		JWTSecret:           "change-this-secret-in-production",
		TokenExpiration:     24 * time.Hour,
		RefreshExpiration:   7 * 24 * time.Hour,
		MinPasswordLength:   8,
		RequireSpecialChar:  true,
		RequireNumber:       true,
		RequireUppercase:    true,
		RequireLowercase:    true,
		MaxLoginAttempts:    5,
		LockoutDuration:     15 * time.Minute,
		BcryptCost:         12,
	}
}

// AuthError represents an authentication error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common authentication errors
var (
	ErrInvalidCredentials = &AuthError{
		Code:    "INVALID_CREDENTIALS",
		Message: "Invalid username or password",
	}
	ErrUserNotFound = &AuthError{
		Code:    "USER_NOT_FOUND",
		Message: "User not found",
	}
	ErrUserInactive = &AuthError{
		Code:    "USER_INACTIVE",
		Message: "User account is inactive",
	}
	ErrUserLocked = &AuthError{
		Code:    "USER_LOCKED",
		Message: "User account is locked due to too many failed login attempts",
	}
	ErrInvalidToken = &AuthError{
		Code:    "INVALID_TOKEN",
		Message: "Invalid or expired token",
	}
	ErrPasswordTooWeak = &AuthError{
		Code:    "PASSWORD_TOO_WEAK",
		Message: "Password does not meet security requirements",
	}
	ErrUsernameExists = &AuthError{
		Code:    "USERNAME_EXISTS",
		Message: "Username already exists",
	}
	ErrEmailExists = &AuthError{
		Code:    "EMAIL_EXISTS",
		Message: "Email address already exists",
	}
)