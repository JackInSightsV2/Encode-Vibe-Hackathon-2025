package auth

import (
	"fmt"
	"time"
	"qt1-middleware/models"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	config *AuthConfig
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config *AuthConfig) *JWTManager {
	if config == nil {
		config = DefaultAuthConfig()
	}
	return &JWTManager{
		config: config,
	}
}

// GenerateToken generates a JWT token for a user
func (jm *JWTManager) GenerateToken(user *models.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}
	
	now := time.Now()
	expiresAt := now.Add(jm.config.TokenExpiration)
	
	claims := &Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}
	
	// Create the token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"email":    claims.Email,
		"role":     claims.Role,
		"iat":      claims.IssuedAt,
		"exp":      claims.ExpiresAt,
		"iss":      "qt1-middleware",
		"sub":      fmt.Sprintf("%d", user.ID),
		"nonce":    now.UnixNano(), // Add nonce to ensure uniqueness
	})
	
	// Sign the token with the secret
	tokenString, err := token.SignedString([]byte(jm.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	
	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (jm *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jm.config.JWTSecret), nil
	})
	
	if err != nil {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: fmt.Sprintf("failed to parse token: %v", err),
		}
	}
	
	// Check if token is valid
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	
	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "invalid token claims",
		}
	}
	
	// Extract and validate required fields
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "missing or invalid user_id in token",
		}
	}
	
	username, ok := claims["username"].(string)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "missing or invalid username in token",
		}
	}
	
	email, ok := claims["email"].(string)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "missing or invalid email in token",
		}
	}
	
	role, ok := claims["role"].(string)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "missing or invalid role in token",
		}
	}
	
	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "missing or invalid iat in token",
		}
	}
	
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, &AuthError{
			Code:    "INVALID_TOKEN",
			Message: "missing or invalid exp in token",
		}
	}
	
	// Check if token is expired
	if time.Now().Unix() > int64(exp) {
		return nil, &AuthError{
			Code:    "TOKEN_EXPIRED",
			Message: "token has expired",
		}
	}
	
	return &Claims{
		UserID:    int(userID),
		Username:  username,
		Email:     email,
		Role:      role,
		IssuedAt:  int64(iat),
		ExpiresAt: int64(exp),
	}, nil
}

// RefreshToken generates a new token if the current token is valid but close to expiration
func (jm *JWTManager) RefreshToken(tokenString string) (string, error) {
	// First validate the current token
	claims, err := jm.ValidateToken(tokenString)
	if err != nil {
		// If token is expired but otherwise valid, we can still refresh it
		// within a grace period (e.g., 1 hour after expiration)
		if authErr, ok := err.(*AuthError); ok && authErr.Code == "TOKEN_EXPIRED" {
			// Parse the expired token without validating expiration
			token, parseErr := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jm.config.JWTSecret), nil
			})
			
			if parseErr != nil {
				return "", err
			}
			
			if tokenClaims, ok := token.Claims.(jwt.MapClaims); ok {
				// Check if token expired within the last hour (grace period)
				if exp, ok := tokenClaims["exp"].(float64); ok {
					expTime := time.Unix(int64(exp), 0)
					if time.Since(expTime) > time.Hour {
						return "", &AuthError{
							Code:    "TOKEN_TOO_OLD",
							Message: "token is too old to refresh",
						}
					}
					
					// Extract claims for refresh
					if userID, ok := tokenClaims["user_id"].(float64); ok {
						if username, ok := tokenClaims["username"].(string); ok {
							if email, ok := tokenClaims["email"].(string); ok {
								if role, ok := tokenClaims["role"].(string); ok {
									claims = &Claims{
										UserID:   int(userID),
										Username: username,
										Email:    email,
										Role:     role,
									}
								}
							}
						}
					}
					
					if claims == nil {
						return "", err
					}
				} else {
					return "", err
				}
			} else {
				return "", err
			}
		} else {
			return "", err
		}
	}
	
	// Create a new user object for token generation
	user := &models.User{
		ID:       claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Role:     claims.Role,
	}
	
	// Generate a new token
	return jm.GenerateToken(user)
}

// GenerateRefreshToken generates a long-lived refresh token
func (jm *JWTManager) GenerateRefreshToken(user *models.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}
	
	now := time.Now()
	expiresAt := now.Add(jm.config.RefreshExpiration)
	
	// Create the refresh token with minimal claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"iat":     now.Unix(),
		"exp":     expiresAt.Unix(),
		"iss":     "qt1-middleware",
		"sub":     fmt.Sprintf("%d", user.ID),
	})
	
	// Sign the token with the secret
	tokenString, err := token.SignedString([]byte(jm.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}
	
	return tokenString, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (jm *JWTManager) ValidateRefreshToken(tokenString string) (int, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jm.config.JWTSecret), nil
	})
	
	if err != nil {
		return 0, &AuthError{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: fmt.Sprintf("failed to parse refresh token: %v", err),
		}
	}
	
	// Check if token is valid
	if !token.Valid {
		return 0, &AuthError{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: "invalid refresh token",
		}
	}
	
	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, &AuthError{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: "invalid refresh token claims",
		}
	}
	
	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return 0, &AuthError{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: "not a refresh token",
		}
	}
	
	// Extract user ID
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, &AuthError{
			Code:    "INVALID_REFRESH_TOKEN",
			Message: "missing or invalid user_id in refresh token",
		}
	}
	
	return int(userID), nil
}

// GetTokenExpiration returns the expiration time for regular tokens
func (jm *JWTManager) GetTokenExpiration() time.Duration {
	return jm.config.TokenExpiration
}

// GetRefreshTokenExpiration returns the expiration time for refresh tokens
func (jm *JWTManager) GetRefreshTokenExpiration() time.Duration {
	return jm.config.RefreshExpiration
}