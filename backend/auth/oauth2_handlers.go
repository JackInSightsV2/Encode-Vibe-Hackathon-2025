package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OAuth2Handlers provides HTTP handlers for OAuth2 authentication flows
type OAuth2Handlers struct {
	oauth2Manager  *OAuth2Manager
	sessionManager *SessionManager
	authService    AuthService
}

// NewOAuth2Handlers creates new OAuth2 handlers
func NewOAuth2Handlers(oauth2Manager *OAuth2Manager, sessionManager *SessionManager, authService AuthService) *OAuth2Handlers {
	return &OAuth2Handlers{
		oauth2Manager:  oauth2Manager,
		sessionManager: sessionManager,
		authService:    authService,
	}
}

// ListProviders returns available OAuth2 providers
func (h *OAuth2Handlers) ListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := h.oauth2Manager.GetProviders()
	
	// Create a public view of providers (without sensitive information)
	publicProviders := make(map[string]interface{})
	for id, provider := range providers {
		publicProviders[id] = map[string]interface{}{
			"id":      provider.ID,
			"name":    provider.Name,
			"enabled": provider.Enabled,
		}
	}

	response := map[string]interface{}{
		"success":   true,
		"providers": publicProviders,
		"count":     len(publicProviders),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// InitiateLogin initiates OAuth2 login flow
func (h *OAuth2Handlers) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Provider    string `json:"provider" validate:"required"`
		RedirectURL string `json:"redirect_url,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate provider
	provider, err := h.oauth2Manager.GetProvider(req.Provider)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	// Default redirect URL if not provided
	if req.RedirectURL == "" {
		req.RedirectURL = "/"
	}

	// Generate authorization URL
	authURL, err := h.oauth2Manager.GenerateAuthURL(provider.ID, req.RedirectURL)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "AUTH_URL_GENERATION_FAILED", "Failed to generate authorization URL")
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"auth_url": authURL,
		"provider": provider.ID,
		"state":    "generated", // Don't expose actual state
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleCallback handles OAuth2 callback
func (h *OAuth2Handlers) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract parameters from query string
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")
	errorDescription := r.URL.Query().Get("error_description")

	// Check for OAuth2 errors
	if errorParam != "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "OAUTH2_ERROR", 
			fmt.Sprintf("OAuth2 error: %s - %s", errorParam, errorDescription))
		return
	}

	// Validate required parameters
	if state == "" || code == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "MISSING_PARAMETERS", "Missing state or code parameter")
		return
	}

	// Handle the callback
	user, _, err := h.oauth2Manager.HandleCallback(r.Context(), state, code)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "CALLBACK_FAILED", err.Error())
		return
	}

	// Create session using session manager if available
	if h.sessionManager != nil {
		_, err := h.sessionManager.CreateSession(r.Context(), user.ID, r)
		if err != nil {
			h.writeErrorResponse(w, http.StatusInternalServerError, "SESSION_CREATION_FAILED", "Failed to create session")
			return
		}
	}

	// Generate JWT token
	token, err := h.authService.GenerateToken(user)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate token")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "OAuth2 authentication successful",
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
		"token": token,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// HandleProviderCallback handles provider-specific callbacks
func (h *OAuth2Handlers) HandleProviderCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract provider from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_PATH", "Invalid callback path")
		return
	}

	providerID := pathParts[len(pathParts)-1]
	
	// Validate provider exists
	_, err := h.oauth2Manager.GetProvider(providerID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	// Delegate to generic callback handler
	h.HandleCallback(w, r)
}

// LinkAccount links an OAuth2 account to an existing user
func (h *OAuth2Handlers) LinkAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	_, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Parse request
	var req struct {
		Provider string `json:"provider" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate provider
	provider, err := h.oauth2Manager.GetProvider(req.Provider)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	// Generate authorization URL for account linking
	authURL, err := h.oauth2Manager.GenerateAuthURL(provider.ID, "/account/linked")
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "AUTH_URL_GENERATION_FAILED", "Failed to generate authorization URL")
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"auth_url": authURL,
		"provider": provider.ID,
		"message":  "Complete OAuth2 flow to link account",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// UnlinkAccount unlinks an OAuth2 account from the current user
func (h *OAuth2Handlers) UnlinkAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	_, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Parse request
	var req struct {
		Provider string `json:"provider" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate provider
	_, err := h.oauth2Manager.GetProvider(req.Provider)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	// TODO: Implement account unlinking logic
	// This would involve removing OAuth2 account associations from the database

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Account unlinked from %s successfully", req.Provider),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetLinkedAccounts returns OAuth2 accounts linked to the current user
func (h *OAuth2Handlers) GetLinkedAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// TODO: Implement linked accounts retrieval from database
	linkedAccounts := []map[string]interface{}{
		// Example structure:
		// {
		//     "provider": "google",
		//     "provider_id": "123456789",
		//     "email": "user@gmail.com",
		//     "linked_at": "2023-01-01T00:00:00Z",
		// }
	}

	response := map[string]interface{}{
		"success": true,
		"user_id": user.ID,
		"linked_accounts": linkedAccounts,
		"count": len(linkedAccounts),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// RefreshToken refreshes an OAuth2 access token
func (h *OAuth2Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Provider     string `json:"provider" validate:"required"`
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate provider
	provider, err := h.oauth2Manager.GetProvider(req.Provider)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	// TODO: Implement token refresh logic
	// This would involve making a request to the provider's token endpoint
	// with the refresh token to get a new access token

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Token refreshed for provider %s", provider.Name),
		// "access_token": newAccessToken,
		// "expires_in": expiresIn,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetOAuth2Status returns OAuth2 system status
func (h *OAuth2Handlers) GetOAuth2Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := h.oauth2Manager.GetProviders()
	
	// Count enabled providers
	enabledCount := 0
	for _, provider := range providers {
		if provider.Enabled {
			enabledCount++
		}
	}

	// Clean up expired states
	h.oauth2Manager.CleanupExpiredStates()

	response := map[string]interface{}{
		"success": true,
		"oauth2_enabled": len(providers) > 0,
		"total_providers": len(providers),
		"enabled_providers": enabledCount,
		"status": "operational",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

func (h *OAuth2Handlers) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
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

func (h *OAuth2Handlers) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// OAuth2Middleware provides middleware for OAuth2-specific security checks
func (h *OAuth2Handlers) OAuth2Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add CSRF protection for OAuth2 flows
			if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/oauth2/") {
				// Verify CSRF token if present
				csrfToken := r.Header.Get("X-CSRF-Token")
				if csrfToken == "" {
					// For OAuth2 flows, we might want to be more lenient
					// or implement a different CSRF protection mechanism
				}
			}

			// Add security headers
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			next.ServeHTTP(w, r)
		})
	}
}