package middleware

import (
	"context"
	"net/http"
	"qt1-middleware/auth"
	"qt1-middleware/models"
	"qt1-middleware/utils"
	"strings"
)

// APIKeyAuthMiddleware provides API key authentication middleware
type APIKeyAuthMiddleware struct {
	apiKeyManager *auth.APIKeyManager
	logger        utils.Logger
}

// NewAPIKeyAuthMiddleware creates a new API key authentication middleware
func NewAPIKeyAuthMiddleware(apiKeyManager *auth.APIKeyManager, logger utils.Logger) *APIKeyAuthMiddleware {
	return &APIKeyAuthMiddleware{
		apiKeyManager: apiKeyManager,
		logger:        logger,
	}
}

// APIKeyAuth middleware validates API key authentication
func (m *APIKeyAuthMiddleware) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from various possible headers
		apiKey := m.extractAPIKey(r)
		if apiKey == "" {
			m.writeErrorResponse(w, http.StatusUnauthorized, "API key required", "MISSING_API_KEY")
			return
		}

		// Validate the API key
		keyRecord, user, err := m.apiKeyManager.ValidateAPIKey(r.Context(), apiKey)
		if err != nil {
			m.logger.Warn("API key validation failed", map[string]interface{}{
				"error":      err.Error(),
				"ip_address": getClientIP(r),
				"user_agent": r.UserAgent(),
			})
			m.writeErrorResponse(w, http.StatusUnauthorized, "Invalid API key", "INVALID_API_KEY")
			return
		}

		// Add API key and user to request context
		ctx := context.WithValue(r.Context(), "api_key", keyRecord)
		ctx = context.WithValue(ctx, "user", user)
		ctx = context.WithValue(ctx, "auth_method", "api_key")

		// Log successful authentication
		m.logger.Info("API key authentication successful", map[string]interface{}{
			"user_id":    user.ID,
			"username":   user.Username,
			"api_key_id": keyRecord.ID,
			"ip_address": getClientIP(r),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAPIKeyPermissions middleware checks if the authenticated API key has required permissions
func (m *APIKeyAuthMiddleware) RequireAPIKeyPermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from context
			apiKey, ok := r.Context().Value("api_key").(*models.APIKey)
			if !ok {
				m.writeErrorResponse(w, http.StatusUnauthorized, "API key authentication required", "API_KEY_REQUIRED")
				return
			}

			// Check permissions
			if err := m.apiKeyManager.ValidateKeyPermissions(apiKey, permissions); err != nil {
				m.logger.Warn("API key permission check failed", map[string]interface{}{
					"error":              err.Error(),
					"api_key_id":         apiKey.ID,
					"required_permissions": permissions,
					"key_permissions":    apiKey.Permissions,
					"user_id":           apiKey.UserID,
				})
				m.writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions", "INSUFFICIENT_PERMISSIONS")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyOrJWTAuth middleware accepts either API key or JWT authentication
func (m *APIKeyAuthMiddleware) APIKeyOrJWTAuth(jwtMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for API key first
			apiKey := m.extractAPIKey(r)
			if apiKey != "" {
				// Use API key authentication
				m.APIKeyAuth(next).ServeHTTP(w, r)
				return
			}

			// Fall back to JWT authentication
			jwtMiddleware(next).ServeHTTP(w, r)
		})
	}
}

// RateLimitByAPIKey applies rate limiting based on API key
func (m *APIKeyAuthMiddleware) RateLimitByAPIKey(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from context
			apiKey, ok := r.Context().Value("api_key").(*models.APIKey)
			if !ok {
				// No API key in context, skip rate limiting
				next.ServeHTTP(w, r)
				return
			}

			// Use API key ID as the rate limiting key
			rateLimitKey := "api_key:" + apiKey.ID

			// Check rate limit
			if !limiter.Allow(rateLimitKey) {
				m.logger.Warn("API key rate limit exceeded", map[string]interface{}{
					"api_key_id": apiKey.ID,
					"user_id":    apiKey.UserID,
					"ip_address": getClientIP(r),
				})
				m.writeErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded", "RATE_LIMIT_EXCEEDED")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractAPIKey extracts API key from various header formats
func (m *APIKeyAuthMiddleware) extractAPIKey(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		apiKey := m.apiKeyManager.ExtractKeyFromHeader(authHeader)
		if apiKey != "" {
			return apiKey
		}
	}

	// Try X-API-Key header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		return strings.TrimSpace(apiKey)
	}

	// Try X-API-TOKEN header
	apiKey = r.Header.Get("X-API-TOKEN")
	if apiKey != "" {
		return strings.TrimSpace(apiKey)
	}

	// Try query parameter as fallback (less secure)
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return strings.TrimSpace(apiKey)
	}

	return ""
}

// writeErrorResponse writes a standardized error response
func (m *APIKeyAuthMiddleware) writeErrorResponse(w http.ResponseWriter, statusCode int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	
	// Use JSON encoding if available
	if encoder, ok := w.(interface{ EncodeJSON(interface{}) error }); ok {
		encoder.EncodeJSON(response)
	} else {
		// Fallback to simple JSON string
		w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + message + `"}}`))
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		return ip[:colon]
	}
	return ip
}

// RateLimiter interface for rate limiting functionality
type RateLimiter interface {
	Allow(key string) bool
}



// APIKeyInfo provides information about an authenticated API key
type APIKeyInfo struct {
	ID          string    `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	Active      bool      `json:"active"`
}

// GetAPIKeyFromContext extracts API key information from request context
func GetAPIKeyFromContext(ctx context.Context) (*models.APIKey, bool) {
	if apiKey, ok := ctx.Value("api_key").(*models.APIKey); ok {
		return apiKey, true
	}
	return nil, false
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	if user, ok := ctx.Value("user").(*models.User); ok {
		return user, true
	}
	return nil, false
}

// GetAuthMethodFromContext extracts authentication method from request context
func GetAuthMethodFromContext(ctx context.Context) (string, bool) {
	if method, ok := ctx.Value("auth_method").(string); ok {
		return method, true
	}
	return "", false
}

// APIKeyStatsMiddleware tracks API key usage statistics
func (m *APIKeyAuthMiddleware) APIKeyStatsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from context
		apiKey, hasAPIKey := GetAPIKeyFromContext(r.Context())
		
		// Execute the request
		next.ServeHTTP(w, r)
		
		// Record usage if API key was used
		if hasAPIKey {
			go func() {
				// Record usage in background to avoid blocking the response
				// This would typically be done through the repository
				clientIP := getClientIP(r)
				userAgent := r.UserAgent()
				
				// Log the usage
				m.logger.Info("API key usage recorded", map[string]interface{}{
					"api_key_id": apiKey.ID,
					"user_id":    apiKey.UserID,
					"endpoint":   r.URL.Path,
					"method":     r.Method,
					"ip_address": clientIP,
					"user_agent": userAgent,
				})
			}()
		}
	})
}