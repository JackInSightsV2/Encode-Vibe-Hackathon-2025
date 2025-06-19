package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	IsAdmin  bool     `json:"is_admin"`
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	IsAdmin  bool     `json:"is_admin"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	jwtSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(secret string) *AuthService {
	if secret == "" {
		secret = "default-jwt-secret-change-in-production" // Default for development
	}
	return &AuthService{
		jwtSecret: []byte(secret),
	}
}

// GenerateToken generates a JWT token for a user (for testing purposes)
func (as *AuthService) GenerateToken(user User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
		IsAdmin:  user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "qt1-middleware",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(as.jwtSecret)
}

// ValidateToken validates a JWT token and returns the user
func (as *AuthService) ValidateToken(tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	user := &User{
		ID:       claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
		IsAdmin:  claims.IsAdmin,
	}

	return user, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func (as *AuthService) ExtractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header missing")
	}

	// Check for "Bearer " prefix
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("invalid authorization header format")
	}

	return strings.TrimPrefix(authHeader, "Bearer "), nil
}

// ExtractTokenFromQuery extracts JWT token from query parameter (for WebSocket)
func (as *AuthService) ExtractTokenFromQuery(r *http.Request) (string, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return "", errors.New("token query parameter missing")
	}
	return token, nil
}

// AuthenticateRequest authenticates an HTTP request and returns the user
func (as *AuthService) AuthenticateRequest(r *http.Request) (*User, error) {
	// Try header first, then query parameter
	var tokenString string
	var err error

	tokenString, err = as.ExtractTokenFromHeader(r)
	if err != nil {
		tokenString, err = as.ExtractTokenFromQuery(r)
		if err != nil {
			return nil, errors.New("no valid authentication token found")
		}
	}

	return as.ValidateToken(tokenString)
}

// HasRole checks if a user has a specific role
func (u *User) HasRole(role string) bool {
	if u.IsAdmin {
		return true // Admins have all roles
	}
	
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// CanAccessWebSocket checks if a user can access WebSocket
func (u *User) CanAccessWebSocket() bool {
	return u.HasRole("websocket") || u.HasRole("admin") || u.IsAdmin
}

// CanReceiveHealthUpdates checks if a user can receive health updates
func (u *User) CanReceiveHealthUpdates() bool {
	return u.HasRole("health") || u.HasRole("admin") || u.IsAdmin
}

// CanReceiveLogUpdates checks if a user can receive log updates
func (u *User) CanReceiveLogUpdates() bool {
	return u.HasRole("logs") || u.HasRole("admin") || u.IsAdmin
}

// Rate limiting structures
type RateLimiter struct {
	connections map[string]*ConnectionInfo
	maxConn     int
	timeWindow  time.Duration
}

type ConnectionInfo struct {
	count     int
	firstConn time.Time
	lastConn  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxConnections int, timeWindow time.Duration) *RateLimiter {
	return &RateLimiter{
		connections: make(map[string]*ConnectionInfo),
		maxConn:     maxConnections,
		timeWindow:  timeWindow,
	}
}

// CheckRateLimit checks if a client can make a new connection
func (rl *RateLimiter) CheckRateLimit(clientIP string) bool {
	now := time.Now()
	
	info, exists := rl.connections[clientIP]
	if !exists {
		rl.connections[clientIP] = &ConnectionInfo{
			count:     1,
			firstConn: now,
			lastConn:  now,
		}
		return true
	}

	// Reset counter if time window has passed
	if now.Sub(info.firstConn) > rl.timeWindow {
		info.count = 1
		info.firstConn = now
		info.lastConn = now
		return true
	}

	// Check if within limits
	if info.count >= rl.maxConn {
		return false
	}

	info.count++
	info.lastConn = now
	return true
}

// CleanupOldEntries removes old rate limit entries
func (rl *RateLimiter) CleanupOldEntries() {
	now := time.Now()
	for ip, info := range rl.connections {
		if now.Sub(info.lastConn) > rl.timeWindow*2 {
			delete(rl.connections, ip)
		}
	}
}

// Global auth service instance
var AuthSvc *AuthService
var WSRateLimiter *RateLimiter

// Initialize auth service
func init() {
	// Initialize with default secret (should be configurable in production)
	AuthSvc = NewAuthService("")
	
	// Initialize rate limiter: max 5 connections per minute per IP
	WSRateLimiter = NewRateLimiter(5, time.Minute)
	
	// Start cleanup routine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			WSRateLimiter.CleanupOldEntries()
		}
	}()
}