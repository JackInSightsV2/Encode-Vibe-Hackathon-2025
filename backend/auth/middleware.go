package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"qt1-middleware/models"
)

// AuthMiddleware provides authentication middleware for HTTP handlers
type AuthMiddleware struct {
	authService AuthService
	rbacManager *RBACManager
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		rbacManager: NewRBACManager(authService),
	}
}

// RequireAuth is middleware that requires authentication
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		tokenString := am.extractTokenFromRequest(r)
		if tokenString == "" {
			am.writeUnauthorizedResponse(w, "MISSING_TOKEN", "Authentication token required")
			return
		}
		
		// Get user from token
		user, err := am.authService.GetUserByToken(tokenString)
		if err != nil {
			if authErr, ok := err.(*AuthError); ok {
				am.writeUnauthorizedResponse(w, authErr.Code, authErr.Message)
			} else {
				am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
			}
			return
		}
		
		// Add user to request context
		ctx := context.WithValue(r.Context(), "user", user)
		r = r.WithContext(ctx)
		
		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// RequireRole is middleware that requires a specific role
func (am *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return am.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := r.Context().Value("user").(*models.User)
			if !ok {
				am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
				return
			}
			
			// Check role
			if !user.HasRole(role) && !user.IsAdmin() {
				am.writeForbiddenResponse(w, "INSUFFICIENT_PERMISSIONS", fmt.Sprintf("Role '%s' required", role))
				return
			}
			
			// Call next handler
			next.ServeHTTP(w, r)
		}))
	}
}

// RequireAdmin is middleware that requires admin role
func (am *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return am.RequireRole("admin")(next)
}

// RequireModerator is middleware that requires moderator or admin role
func (am *AuthMiddleware) RequireModerator(next http.Handler) http.Handler {
	return am.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user, ok := r.Context().Value("user").(*models.User)
		if !ok {
			am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
			return
		}
		
		// Check if user can moderate
		if !user.CanModerate() {
			am.writeForbiddenResponse(w, "INSUFFICIENT_PERMISSIONS", "Moderator privileges required")
			return
		}
		
		// Call next handler
		next.ServeHTTP(w, r)
	}))
}

// OptionalAuth is middleware that extracts user info if token is present, but doesn't require it
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from request
		tokenString := am.extractTokenFromRequest(r)
		if tokenString != "" {
			// Try to get user from token
			user, err := am.authService.GetUserByToken(tokenString)
			if err == nil {
				// Add user to request context
				ctx := context.WithValue(r.Context(), "user", user)
				r = r.WithContext(ctx)
			}
		}
		
		// Call next handler regardless of authentication result
		next.ServeHTTP(w, r)
	})
}

// SelfOrAdmin is middleware that allows access if the user is accessing their own resource or is an admin
func (am *AuthMiddleware) SelfOrAdmin(getUserIDFromPath func(string) (int, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return am.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := r.Context().Value("user").(*models.User)
			if !ok {
				am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
				return
			}
			
			// If user is admin, allow access
			if user.IsAdmin() {
				next.ServeHTTP(w, r)
				return
			}
			
			// Extract target user ID from path
			targetUserID, err := getUserIDFromPath(r.URL.Path)
			if err != nil {
				am.writeForbiddenResponse(w, "INVALID_REQUEST", "Invalid user ID in path")
				return
			}
			
			// Check if user is accessing their own resource
			if user.ID != targetUserID {
				am.writeForbiddenResponse(w, "INSUFFICIENT_PERMISSIONS", "Access denied")
				return
			}
			
			// Call next handler
			next.ServeHTTP(w, r)
		}))
	}
}

// RateLimiter is middleware that implements rate limiting per user
func (am *AuthMiddleware) RateLimiter(requestsPerMinute int) func(http.Handler) http.Handler {
	// This is a placeholder - in production you'd implement proper rate limiting
	// with Redis or in-memory store
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For now, just pass through
			// TODO: Implement rate limiting logic
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware for authentication endpoints
func (am *AuthMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // In production, be more restrictive
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// Logging middleware for authentication events
func (am *AuthMiddleware) LogAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from context if available
		user, _ := r.Context().Value("user").(*models.User)
		
		// Log authentication event
		if user != nil {
			fmt.Printf("Auth: User %s (%d) accessing %s %s\n", user.Username, user.ID, r.Method, r.URL.Path)
		} else {
			fmt.Printf("Auth: Anonymous user accessing %s %s\n", r.Method, r.URL.Path)
		}
		
		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// Helper methods

func (am *AuthMiddleware) extractTokenFromRequest(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	
	// Try query parameter (for WebSocket compatibility)
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}
	
	// Try cookie (if using cookie-based auth)
	if cookie, err := r.Cookie("auth_token"); err == nil {
		return cookie.Value
	}
	
	return ""
}

func (am *AuthMiddleware) writeUnauthorizedResponse(w http.ResponseWriter, code, message string) {
	am.writeErrorResponse(w, http.StatusUnauthorized, code, message)
}

func (am *AuthMiddleware) writeForbiddenResponse(w http.ResponseWriter, code, message string) {
	am.writeErrorResponse(w, http.StatusForbidden, code, message)
}

func (am *AuthMiddleware) writeErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	// Write JSON response
	if err := writeJSON(w, response); err != nil {
		// Fallback to plain text if JSON encoding fails
		http.Error(w, message, statusCode)
	}
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// RequirePermission is middleware that requires a specific permission
func (am *AuthMiddleware) RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return am.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := r.Context().Value("user").(*models.User)
			if !ok {
				am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
				return
			}
			
			// Check permission
			if !am.rbacManager.CheckPermission(user, permission) {
				am.rbacManager.AuditUserAccess(user, permission, false, r.URL.Path)
				am.writeForbiddenResponse(w, "INSUFFICIENT_PERMISSIONS", fmt.Sprintf("Permission '%s' required", permission))
				return
			}
			
			// Log successful access
			am.rbacManager.AuditUserAccess(user, permission, true, r.URL.Path)
			
			// Call next handler
			next.ServeHTTP(w, r)
		}))
	}
}

// RequireAnyPermission is middleware that requires at least one of the specified permissions
func (am *AuthMiddleware) RequireAnyPermission(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return am.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := r.Context().Value("user").(*models.User)
			if !ok {
				am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
				return
			}
			
			// Check if user has any of the required permissions
			hasPermission := false
			var grantedPermission Permission
			
			for _, permission := range permissions {
				if am.rbacManager.CheckPermission(user, permission) {
					hasPermission = true
					grantedPermission = permission
					break
				}
			}
			
			if !hasPermission {
				// Log denied access for the first permission (for audit purposes)
				if len(permissions) > 0 {
					am.rbacManager.AuditUserAccess(user, permissions[0], false, r.URL.Path)
				}
				
				permissionList := make([]string, len(permissions))
				for i, p := range permissions {
					permissionList[i] = string(p)
				}
				am.writeForbiddenResponse(w, "INSUFFICIENT_PERMISSIONS", 
					fmt.Sprintf("One of the following permissions required: %s", strings.Join(permissionList, ", ")))
				return
			}
			
			// Log successful access
			am.rbacManager.AuditUserAccess(user, grantedPermission, true, r.URL.Path)
			
			// Call next handler
			next.ServeHTTP(w, r)
		}))
	}
}

// RequireAllPermissions is middleware that requires all specified permissions
func (am *AuthMiddleware) RequireAllPermissions(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return am.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context
			user, ok := r.Context().Value("user").(*models.User)
			if !ok {
				am.writeUnauthorizedResponse(w, "AUTHENTICATION_FAILED", "Authentication failed")
				return
			}
			
			// Check that user has all required permissions
			missingPermissions := []string{}
			
			for _, permission := range permissions {
				if !am.rbacManager.CheckPermission(user, permission) {
					missingPermissions = append(missingPermissions, string(permission))
				}
			}
			
			if len(missingPermissions) > 0 {
				// Log denied access for the first missing permission
				if len(permissions) > 0 {
					am.rbacManager.AuditUserAccess(user, permissions[0], false, r.URL.Path)
				}
				
				am.writeForbiddenResponse(w, "INSUFFICIENT_PERMISSIONS", 
					fmt.Sprintf("Missing required permissions: %s", strings.Join(missingPermissions, ", ")))
				return
			}
			
			// Log successful access for the first permission (representative)
			if len(permissions) > 0 {
				am.rbacManager.AuditUserAccess(user, permissions[0], true, r.URL.Path)
			}
			
			// Call next handler
			next.ServeHTTP(w, r)
		}))
	}
}

// RequireResourceAccess is middleware that checks access to a specific resource with an action
func (am *AuthMiddleware) RequireResourceAccess(resource, action string) func(http.Handler) http.Handler {
	permission := Permission(fmt.Sprintf("%s:%s", resource, action))
	return am.RequirePermission(permission)
}

// RequireSystemAdmin is middleware that requires system admin permissions
func (am *AuthMiddleware) RequireSystemAdmin(next http.Handler) http.Handler {
	return am.RequirePermission(PermissionSystemAdmin)(next)
}

// RequireUserManagement is middleware that requires user management permissions
func (am *AuthMiddleware) RequireUserManagement(next http.Handler) http.Handler {
	return am.RequireAnyPermission(
		PermissionUserCreate, PermissionUserUpdate, PermissionUserDelete,
	)(next)
}

// RequireConfigAccess is middleware that requires configuration access
func (am *AuthMiddleware) RequireConfigAccess(readOnly bool) func(http.Handler) http.Handler {
	if readOnly {
		return am.RequirePermission(PermissionConfigRead)
	}
	return am.RequirePermission(PermissionConfigWrite)
}

// RequireModerationAccess is middleware that requires moderation permissions
func (am *AuthMiddleware) RequireModerationAccess(next http.Handler) http.Handler {
	return am.RequireAnyPermission(
		PermissionModerateContent, PermissionModerateUsers, PermissionModerateRules,
	)(next)
}

// RequireMetricsAccess is middleware that requires metrics access
func (am *AuthMiddleware) RequireMetricsAccess(writeAccess bool) func(http.Handler) http.Handler {
	if writeAccess {
		return am.RequirePermission(PermissionAnalyticsWrite)
	}
	return am.RequirePermission(PermissionMetricsRead)
}

// RequireLogsAccess is middleware that requires logs access
func (am *AuthMiddleware) RequireLogsAccess(next http.Handler) http.Handler {
	return am.RequirePermission(PermissionLogsRead)(next)
}

// RequireKillSwitchAccess is middleware that requires kill switch permissions
func (am *AuthMiddleware) RequireKillSwitchAccess(writeAccess bool) func(http.Handler) http.Handler {
	if writeAccess {
		return am.RequirePermission(PermissionKillSwitchWrite)
	}
	return am.RequirePermission(PermissionKillSwitchRead)
}

// RequireDatabaseAccess is middleware that requires database access
func (am *AuthMiddleware) RequireDatabaseAccess(adminAccess bool) func(http.Handler) http.Handler {
	if adminAccess {
		return am.RequirePermission(PermissionDatabaseAdmin)
	}
	return am.RequirePermission(PermissionDatabaseRead)
}

// GetRBACManager returns the RBAC manager for advanced permission checks
func (am *AuthMiddleware) GetRBACManager() *RBACManager {
	return am.rbacManager
}

// PermissionCheck is a helper function for checking permissions in handlers
func (am *AuthMiddleware) PermissionCheck(user *models.User, permission Permission) bool {
	return am.rbacManager.CheckPermission(user, permission)
}

// Chain combines multiple middleware functions
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}