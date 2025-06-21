package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qt1-middleware/models"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// AuthHandlers contains HTTP handlers for authentication endpoints
type AuthHandlers struct {
	authService AuthService
	rbacManager *RBACManager
	validator   *validator.Validate
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(authService AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		rbacManager: NewRBACManager(authService),
		validator:   validator.New(),
	}
}

// LoginHandler handles user login requests
func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data")
		return
	}
	
	// Attempt login
	result, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		if authErr, ok := err.(*AuthError); ok {
			switch authErr.Code {
			case "INVALID_CREDENTIALS":
				h.writeErrorResponse(w, http.StatusUnauthorized, authErr.Code, authErr.Message)
			case "USER_INACTIVE":
				h.writeErrorResponse(w, http.StatusForbidden, authErr.Code, authErr.Message)
			case "USER_LOCKED":
				h.writeErrorResponse(w, http.StatusForbidden, authErr.Code, authErr.Message)
			default:
				h.writeErrorResponse(w, http.StatusInternalServerError, "LOGIN_FAILED", "Login failed")
			}
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "LOGIN_FAILED", "Login failed")
		}
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"token":         result.Token,
			"refresh_token": result.RefreshToken,
			"expires_at":    result.ExpiresAt,
			"user":          result.User.ToResponse(),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterHandler handles user registration requests
func (h *AuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data")
		return
	}
	
	// Attempt registration
	user, err := h.authService.Register(&req)
	if err != nil {
		if authErr, ok := err.(*AuthError); ok {
			switch authErr.Code {
			case "USERNAME_EXISTS":
				h.writeErrorResponse(w, http.StatusConflict, authErr.Code, authErr.Message)
			case "EMAIL_EXISTS":
				h.writeErrorResponse(w, http.StatusConflict, authErr.Code, authErr.Message)
			case "PASSWORD_TOO_WEAK":
				h.writeErrorResponse(w, http.StatusBadRequest, authErr.Code, authErr.Message)
			default:
				h.writeErrorResponse(w, http.StatusInternalServerError, "REGISTRATION_FAILED", "Registration failed")
			}
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "REGISTRATION_FAILED", "Registration failed")
		}
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"success": true,
		"user":    user.ToResponse(),
		"message": "User registered successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// LogoutHandler handles user logout requests
func (h *AuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	
	// Perform logout
	if err := h.authService.Logout(user.ID); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "LOGOUT_FAILED", "Logout failed")
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RefreshTokenHandler handles token refresh requests
func (h *AuthHandlers) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req TokenRefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data")
		return
	}
	
	// Refresh token
	newToken, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		if authErr, ok := err.(*AuthError); ok {
			switch authErr.Code {
			case "INVALID_REFRESH_TOKEN":
				h.writeErrorResponse(w, http.StatusUnauthorized, authErr.Code, authErr.Message)
			case "USER_NOT_FOUND":
				h.writeErrorResponse(w, http.StatusUnauthorized, authErr.Code, authErr.Message)
			case "USER_INACTIVE":
				h.writeErrorResponse(w, http.StatusForbidden, authErr.Code, authErr.Message)
			default:
				h.writeErrorResponse(w, http.StatusInternalServerError, "REFRESH_FAILED", "Token refresh failed")
			}
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "REFRESH_FAILED", "Token refresh failed")
		}
		return
	}
	
	// Return new token
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"token": newToken,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// MeHandler returns the current user's information
func (h *AuthHandlers) MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	
	// Return user information
	response := map[string]interface{}{
		"success": true,
		"user":    user.ToResponse(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ChangePasswordHandler handles password change requests
func (h *AuthHandlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data")
		return
	}
	
	// Change password
	if err := h.authService.ChangePassword(user.ID, req.OldPassword, req.NewPassword); err != nil {
		if authErr, ok := err.(*AuthError); ok {
			switch authErr.Code {
			case "INVALID_OLD_PASSWORD":
				h.writeErrorResponse(w, http.StatusBadRequest, authErr.Code, authErr.Message)
			case "PASSWORD_TOO_WEAK":
				h.writeErrorResponse(w, http.StatusBadRequest, authErr.Code, authErr.Message)
			default:
				h.writeErrorResponse(w, http.StatusInternalServerError, "PASSWORD_CHANGE_FAILED", "Password change failed")
			}
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "PASSWORD_CHANGE_FAILED", "Password change failed")
		}
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Password changed successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ValidateTokenHandler validates a token and returns user info (for testing)
func (h *AuthHandlers) ValidateTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract token from header
	tokenString := h.extractTokenFromRequest(r)
	if tokenString == "" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "MISSING_TOKEN", "Authentication token required")
		return
	}
	
	// Validate token
	claims, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		if authErr, ok := err.(*AuthError); ok {
			h.writeErrorResponse(w, http.StatusUnauthorized, authErr.Code, authErr.Message)
		} else {
			h.writeErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token")
		}
		return
	}
	
	// Return token info
	response := map[string]interface{}{
		"success": true,
		"valid":   true,
		"claims":  claims,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper methods

func (h *AuthHandlers) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    errorCode,
			"message": message,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandlers) extractTokenFromRequest(r *http.Request) string {
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
	
	return ""
}

// RegisterRoutes registers all authentication routes
func (h *AuthHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/login", h.LoginHandler)
	mux.HandleFunc("/api/auth/register", h.RegisterHandler)
	mux.HandleFunc("/api/auth/logout", h.LogoutHandler)
	mux.HandleFunc("/api/auth/refresh", h.RefreshTokenHandler)
	mux.HandleFunc("/api/auth/me", h.MeHandler)
	mux.HandleFunc("/api/auth/change-password", h.ChangePasswordHandler)
	mux.HandleFunc("/api/auth/validate", h.ValidateTokenHandler)
}

// GetUserFromRequest extracts user from request context
func GetUserFromRequest(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value("user").(*models.User)
	return user, ok
}

// GetUserIDFromRequest extracts user ID from request context
func GetUserIDFromRequest(r *http.Request) (int, bool) {
	user, ok := GetUserFromRequest(r)
	if !ok {
		return 0, false
	}
	return user.ID, true
}

// RequireAdmin checks if the user is an admin
func RequireAdmin(r *http.Request) bool {
	user, ok := GetUserFromRequest(r)
	if !ok {
		return false
	}
	return user.IsAdmin()
}

// RequireRole checks if the user has a specific role
func RequireRole(r *http.Request, role string) bool {
	user, ok := GetUserFromRequest(r)
	if !ok {
		return false
	}
	return user.HasRole(role)
}

// ParseUserIDFromURL extracts user ID from URL path
func ParseUserIDFromURL(urlPath string) (int, error) {
	parts := strings.Split(urlPath, "/")
	if len(parts) < 4 {
		return 0, fmt.Errorf("invalid URL format")
	}
	
	userIDStr := parts[3] // Assuming /api/users/{id} format
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}
	
	return userID, nil
}

// RBAC Management Handlers

// GetRolesHandler returns all available roles in the system
func (h *AuthHandlers) GetRolesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	roles := h.rbacManager.GetAvailableRoles()
	
	// Get detailed info for each role
	roleDetails := make([]map[string]interface{}, len(roles))
	for i, role := range roles {
		roleDetails[i] = h.rbacManager.GetRoleInfo(role)
	}
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"roles":        roles,
			"role_details": roleDetails,
			"total":        len(roles),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetPermissionsHandler returns all available permissions in the system
func (h *AuthHandlers) GetPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	permissions := h.rbacManager.GetAvailablePermissions()
	
	// Get detailed info for each permission
	permissionDetails := make([]map[string]interface{}, len(permissions))
	for i, permission := range permissions {
		permissionDetails[i] = h.rbacManager.GetPermissionInfo(Permission(permission))
	}
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"permissions":        permissions,
			"permission_details": permissionDetails,
			"total":              len(permissions),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetRolePermissionsHandler returns permissions for a specific role
func (h *AuthHandlers) GetRolePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract role from query parameter
	role := r.URL.Query().Get("role")
	if role == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "MISSING_ROLE", "Role parameter is required")
		return
	}
	
	// Validate role
	if !h.rbacManager.IsValidRole(role) {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_ROLE", "Invalid role specified")
		return
	}
	
	permissions := h.rbacManager.GetRolePermissions(role)
	roleInfo := h.rbacManager.GetRoleInfo(role)
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"role":        role,
			"permissions": permissions,
			"role_info":   roleInfo,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetUserPermissionsHandler returns permissions for the current user
func (h *AuthHandlers) GetUserPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get user from request context
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}
	
	permissions := h.rbacManager.GetUserPermissions(user)
	roleInfo := h.rbacManager.GetRoleInfo(user.Role)
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"user":        user.ToResponse(),
			"role":        user.Role,
			"permissions": permissions,
			"role_info":   roleInfo,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CheckPermissionHandler checks if the current user has a specific permission
func (h *AuthHandlers) CheckPermissionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Get user from request context
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}
	
	// Parse request body
	var req struct {
		Permission string `json:"permission" validate:"required"`
		Resource   string `json:"resource"`
		Action     string `json:"action"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	if err := h.validator.Struct(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data")
		return
	}
	
	var permission Permission
	
	// Build permission from resource/action or use direct permission
	if req.Resource != "" && req.Action != "" {
		permission = Permission(fmt.Sprintf("%s:%s", req.Resource, req.Action))
	} else {
		permission = Permission(req.Permission)
	}
	
	hasPermission := h.rbacManager.CheckPermission(user, permission)
	
	// Log the permission check
	h.rbacManager.AuditUserAccess(user, permission, hasPermission, "permission_check_api")
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"user":           user.ToResponse(),
			"permission":     string(permission),
			"has_permission": hasPermission,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetRBACStatsHandler returns RBAC statistics
func (h *AuthHandlers) GetRBACStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	roles := h.rbacManager.GetAvailableRoles()
	permissions := h.rbacManager.GetAvailablePermissions()
	
	// Calculate permission distribution across roles
	permissionDistribution := make(map[string]int)
	for _, role := range roles {
		rolePermissions := h.rbacManager.GetRolePermissions(role)
		permissionDistribution[role] = len(rolePermissions)
	}
	
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"total_roles":             len(roles),
			"total_permissions":       len(permissions),
			"permission_distribution": permissionDistribution,
			"roles":                   roles,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}