package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"qt1-middleware/database"
	"qt1-middleware/models"
	"qt1-middleware/repositories"
	"time"
)

// AuthManager manages the authentication system for the application
type AuthManager struct {
	Service         AuthService
	Handlers        *AuthHandlers
	Middleware      *AuthMiddleware
	Config          *AuthConfig
	SessionManager  *SessionManager
	SessionHandlers *SessionHandlers
	OAuth2Manager   *OAuth2Manager
	OAuth2Handlers  *OAuth2Handlers
	OIDCValidator   *OIDCValidator
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(db *database.Database) (*AuthManager, error) {
	// Create configuration
	config := &AuthConfig{
		JWTSecret:           getEnvOrDefault("JWT_SECRET", "change-this-secret-in-production"),
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
	
	// Create user repository
	userRepo := repositories.NewUserRepository(db.DB)
	
	// Create authentication service
	authService := NewService(config, userRepo)
	
	// Create session repository and manager
	sessionRepo := repositories.NewSessionRepository(db.DB)
	sessionManager := NewSessionManager(sessionRepo, DefaultSessionConfig())
	
	// Create OAuth2 configuration and manager
	oauth2Config := DefaultOAuth2Config()
	oauth2Manager := NewOAuth2Manager(oauth2Config, authService, authService)
	oidcValidator := NewOIDCValidator()
	
	// Create handlers and middleware
	handlers := NewAuthHandlers(authService)
	middleware := NewAuthMiddleware(authService)
	sessionHandlers := NewSessionHandlers(sessionManager, authService)
	oauth2Handlers := NewOAuth2Handlers(oauth2Manager, sessionManager, authService)
	
	manager := &AuthManager{
		Service:         authService,
		Handlers:        handlers,
		Middleware:      middleware,
		Config:          config,
		SessionManager:  sessionManager,
		SessionHandlers: sessionHandlers,
		OAuth2Manager:   oauth2Manager,
		OAuth2Handlers:  oauth2Handlers,
		OIDCValidator:   oidcValidator,
	}
	
	// Create default admin user if needed
	if err := authService.CreateDefaultAdmin(); err != nil {
		return nil, fmt.Errorf("failed to create default admin: %w", err)
	}
	
	return manager, nil
}

// RegisterRoutes registers all authentication routes with the provided mux
func (am *AuthManager) RegisterRoutes(mux *http.ServeMux) {
	// Apply CORS middleware to all auth routes
	corsMiddleware := am.Middleware.CORS
	logMiddleware := am.Middleware.LogAuth
	
	// Combine middlewares
	authMux := http.NewServeMux()
	
	// Public routes (no authentication required)
	authMux.HandleFunc("/api/auth/login", am.Handlers.LoginHandler)
	authMux.HandleFunc("/api/auth/register", am.Handlers.RegisterHandler)
	authMux.HandleFunc("/api/auth/refresh", am.Handlers.RefreshTokenHandler)
	authMux.HandleFunc("/api/auth/validate", am.Handlers.ValidateTokenHandler)
	
	// Protected routes (authentication required)
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/api/auth/logout", am.Handlers.LogoutHandler)
	protectedMux.HandleFunc("/api/auth/me", am.Handlers.MeHandler)
	protectedMux.HandleFunc("/api/auth/change-password", am.Handlers.ChangePasswordHandler)
	protectedMux.HandleFunc("/api/auth/permissions", am.Handlers.GetUserPermissionsHandler)
	protectedMux.HandleFunc("/api/auth/check-permission", am.Handlers.CheckPermissionHandler)
	
	// RBAC management routes (admin only)
	rbacMux := http.NewServeMux()
	rbacMux.HandleFunc("/api/auth/roles", am.Handlers.GetRolesHandler)
	rbacMux.HandleFunc("/api/auth/permissions/all", am.Handlers.GetPermissionsHandler)
	rbacMux.HandleFunc("/api/auth/role-permissions", am.Handlers.GetRolePermissionsHandler)
	rbacMux.HandleFunc("/api/auth/rbac-stats", am.Handlers.GetRBACStatsHandler)
	
	// Session management routes (protected)
	sessionMux := http.NewServeMux()
	sessionMux.HandleFunc("/api/auth/sessions", am.SessionHandlers.GetUserSessions)
	sessionMux.HandleFunc("/api/auth/sessions/invalidate", am.SessionHandlers.InvalidateSession)
	sessionMux.HandleFunc("/api/auth/sessions/invalidate-all", am.SessionHandlers.InvalidateAllSessions)
	sessionMux.HandleFunc("/api/auth/sessions/extend", am.SessionHandlers.ExtendSession)
	sessionMux.HandleFunc("/api/auth/sessions/suspicious-activity", am.SessionHandlers.GetSuspiciousActivity)
	sessionMux.HandleFunc("/api/auth/devices", am.SessionHandlers.GetUserDevices)
	sessionMux.HandleFunc("/api/auth/devices/manage", am.SessionHandlers.ManageDevice)
	sessionMux.HandleFunc("/api/auth/rate-limit-status", am.SessionHandlers.GetRateLimitStatus)
	
	// Admin session management routes (admin only)
	adminSessionMux := http.NewServeMux()
	adminSessionMux.HandleFunc("/api/auth/admin/sessions/analytics", am.SessionHandlers.GetSessionAnalytics)
	adminSessionMux.HandleFunc("/api/auth/admin/sessions/force-logout", am.SessionHandlers.AdminForceLogout)
	adminSessionMux.HandleFunc("/api/auth/admin/security-stats", am.SessionHandlers.GetSecurityStats)
	
	// OAuth2 routes (public and protected)
	oauth2PublicMux := http.NewServeMux()
	oauth2PublicMux.HandleFunc("/api/oauth2/providers", am.OAuth2Handlers.ListProviders)
	oauth2PublicMux.HandleFunc("/api/oauth2/login", am.OAuth2Handlers.InitiateLogin)
	oauth2PublicMux.HandleFunc("/api/oauth2/callback", am.OAuth2Handlers.HandleCallback)
	oauth2PublicMux.HandleFunc("/api/oauth2/status", am.OAuth2Handlers.GetOAuth2Status)
	
	// OAuth2 provider-specific callback routes
	oauth2CallbackMux := http.NewServeMux()
	oauth2CallbackMux.HandleFunc("/api/oauth2/callback/", am.OAuth2Handlers.HandleProviderCallback)
	
	// OAuth2 protected routes (authentication required)
	oauth2ProtectedMux := http.NewServeMux()
	oauth2ProtectedMux.HandleFunc("/api/oauth2/link", am.OAuth2Handlers.LinkAccount)
	oauth2ProtectedMux.HandleFunc("/api/oauth2/unlink", am.OAuth2Handlers.UnlinkAccount)
	oauth2ProtectedMux.HandleFunc("/api/oauth2/linked-accounts", am.OAuth2Handlers.GetLinkedAccounts)
	oauth2ProtectedMux.HandleFunc("/api/oauth2/refresh-token", am.OAuth2Handlers.RefreshToken)
	
	// Apply authentication middleware to protected routes
	protectedHandler := am.Middleware.RequireAuth(protectedMux)
	
	// Apply authentication middleware to session routes
	sessionHandler := am.Middleware.RequireAuth(sessionMux)
	
	// Apply admin permission middleware to RBAC routes
	rbacHandler := am.Middleware.RequirePermission(PermissionUserRoles)(rbacMux)
	
	// Apply admin permission middleware to admin session routes
	adminSessionHandler := am.Middleware.RequirePermission(PermissionSystemAdmin)(adminSessionMux)
	
	// Apply OAuth2 middleware and authentication to OAuth2 protected routes
	oauth2ProtectedHandler := am.Middleware.RequireAuth(oauth2ProtectedMux)
	oauth2PublicHandler := am.OAuth2Handlers.OAuth2Middleware()(oauth2PublicMux)
	oauth2CallbackHandler := am.OAuth2Handlers.OAuth2Middleware()(oauth2CallbackMux)
	
	// Register all routes with the main mux
	mux.Handle("/api/auth/", logMiddleware(corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler based on path
		switch {
		case r.URL.Path == "/api/auth/logout" || 
			 r.URL.Path == "/api/auth/me" || 
			 r.URL.Path == "/api/auth/change-password" ||
			 r.URL.Path == "/api/auth/permissions" ||
			 r.URL.Path == "/api/auth/check-permission":
			protectedHandler.ServeHTTP(w, r)
		case r.URL.Path == "/api/auth/roles" ||
			 r.URL.Path == "/api/auth/permissions/all" ||
			 r.URL.Path == "/api/auth/role-permissions" ||
			 r.URL.Path == "/api/auth/rbac-stats":
			rbacHandler.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/auth/sessions") ||
			 strings.HasPrefix(r.URL.Path, "/api/auth/devices") ||
			 r.URL.Path == "/api/auth/rate-limit-status":
			sessionHandler.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/auth/admin/"):
			adminSessionHandler.ServeHTTP(w, r)
		default:
			authMux.ServeHTTP(w, r)
		}
	}))))
	
	// Register OAuth2 routes separately
	mux.Handle("/api/oauth2/", logMiddleware(corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/oauth2/callback/"):
			oauth2CallbackHandler.ServeHTTP(w, r)
		case r.URL.Path == "/api/oauth2/link" ||
			 r.URL.Path == "/api/oauth2/unlink" ||
			 r.URL.Path == "/api/oauth2/linked-accounts" ||
			 r.URL.Path == "/api/oauth2/refresh-token":
			oauth2ProtectedHandler.ServeHTTP(w, r)
		default:
			oauth2PublicHandler.ServeHTTP(w, r)
		}
	}))))
}

// ProtectEndpoint wraps an HTTP handler with authentication middleware
func (am *AuthManager) ProtectEndpoint(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		am.Middleware.RequireAuth(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// ProtectAdminEndpoint wraps an HTTP handler with admin authentication middleware
func (am *AuthManager) ProtectAdminEndpoint(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		am.Middleware.RequireAdmin(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// ProtectRoleEndpoint wraps an HTTP handler with role-based authentication middleware
func (am *AuthManager) ProtectRoleEndpoint(role string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		am.Middleware.RequireRole(role)(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// OptionalAuth wraps an HTTP handler with optional authentication
func (am *AuthManager) OptionalAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		am.Middleware.OptionalAuth(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}

// GetUserFromRequest extracts the authenticated user from the request context
func (am *AuthManager) GetUserFromRequest(r *http.Request) (*models.User, bool) {
	return GetUserFromRequest(r)
}

// IsAuthenticated checks if the request contains a valid authentication token
func (am *AuthManager) IsAuthenticated(r *http.Request) bool {
	_, ok := am.GetUserFromRequest(r)
	return ok
}

// IsAdmin checks if the authenticated user is an admin
func (am *AuthManager) IsAdmin(r *http.Request) bool {
	return RequireAdmin(r)
}

// HasRole checks if the authenticated user has a specific role
func (am *AuthManager) HasRole(r *http.Request, role string) bool {
	return RequireRole(r, role)
}

// CreateUser creates a new user (admin only operation)
func (am *AuthManager) CreateUser(req *RegisterRequest) (*models.User, error) {
	return am.Service.Register(req)
}

// AuthenticateWebSocket authenticates a WebSocket connection using token from query parameters
func (am *AuthManager) AuthenticateWebSocket(r *http.Request) (*models.User, error) {
	// Extract token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		return nil, &AuthError{
			Code:    "MISSING_TOKEN",
			Message: "Authentication token required in query parameter",
		}
	}
	
	// Get user by token
	return am.Service.GetUserByToken(token)
}

// ValidateAPIKey validates an API key and returns the associated user
func (am *AuthManager) ValidateAPIKey(apiKey string) (*models.User, error) {
	// This would be implemented when API key management is added in future cycles
	return nil, &AuthError{
		Code:    "NOT_IMPLEMENTED",
		Message: "API key authentication not yet implemented",
	}
}

// OAuth2 related methods

// EnableOAuth2 enables OAuth2 authentication
func (am *AuthManager) EnableOAuth2() {
	am.OAuth2Manager.config.Enabled = true
}

// DisableOAuth2 disables OAuth2 authentication
func (am *AuthManager) DisableOAuth2() {
	am.OAuth2Manager.config.Enabled = false
}

// AddOAuth2Provider adds an OAuth2 provider
func (am *AuthManager) AddOAuth2Provider(provider *OAuth2Provider) error {
	return am.OAuth2Manager.AddProvider(provider)
}

// RemoveOAuth2Provider removes an OAuth2 provider
func (am *AuthManager) RemoveOAuth2Provider(providerID string) error {
	delete(am.OAuth2Manager.providers, providerID)
	return nil
}

// GetOAuth2Provider gets an OAuth2 provider
func (am *AuthManager) GetOAuth2Provider(providerID string) (*OAuth2Provider, error) {
	return am.OAuth2Manager.GetProvider(providerID)
}

// ListOAuth2Providers lists all OAuth2 providers
func (am *AuthManager) ListOAuth2Providers() map[string]*OAuth2Provider {
	return am.OAuth2Manager.GetProviders()
}

// SetupCommonOAuth2Providers sets up common OAuth2 providers with environment variables
func (am *AuthManager) SetupCommonOAuth2Providers(baseURL string) error {
	factory := NewProviderFactory(baseURL)
	
	// Google OAuth2
	if clientID := os.Getenv("GOOGLE_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET"); clientSecret != "" {
			provider := factory.CreateGoogle(clientID, clientSecret)
			if err := am.AddOAuth2Provider(provider); err != nil {
				return fmt.Errorf("failed to add Google provider: %w", err)
			}
		}
	}
	
	// Microsoft OAuth2
	if clientID := os.Getenv("MICROSOFT_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("MICROSOFT_CLIENT_SECRET"); clientSecret != "" {
			provider := factory.CreateMicrosoft(clientID, clientSecret)
			if err := am.AddOAuth2Provider(provider); err != nil {
				return fmt.Errorf("failed to add Microsoft provider: %w", err)
			}
		}
	}
	
	// GitHub OAuth2
	if clientID := os.Getenv("GITHUB_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("GITHUB_CLIENT_SECRET"); clientSecret != "" {
			provider := factory.CreateGitHub(clientID, clientSecret)
			if err := am.AddOAuth2Provider(provider); err != nil {
				return fmt.Errorf("failed to add GitHub provider: %w", err)
			}
		}
	}
	
	// GitLab OAuth2
	if clientID := os.Getenv("GITLAB_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("GITLAB_CLIENT_SECRET"); clientSecret != "" {
			gitlabURL := os.Getenv("GITLAB_URL")
			provider := factory.CreateGitLab(clientID, clientSecret, gitlabURL)
			if err := am.AddOAuth2Provider(provider); err != nil {
				return fmt.Errorf("failed to add GitLab provider: %w", err)
			}
		}
	}
	
	// Okta OAuth2
	if clientID := os.Getenv("OKTA_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("OKTA_CLIENT_SECRET"); clientSecret != "" {
			if oktaDomain := os.Getenv("OKTA_DOMAIN"); oktaDomain != "" {
				provider := factory.CreateOkta(clientID, clientSecret, oktaDomain)
				if err := am.AddOAuth2Provider(provider); err != nil {
					return fmt.Errorf("failed to add Okta provider: %w", err)
				}
			}
		}
	}
	
	// Auth0 OAuth2
	if clientID := os.Getenv("AUTH0_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("AUTH0_CLIENT_SECRET"); clientSecret != "" {
			if auth0Domain := os.Getenv("AUTH0_DOMAIN"); auth0Domain != "" {
				provider := factory.CreateAuth0(clientID, clientSecret, auth0Domain)
				if err := am.AddOAuth2Provider(provider); err != nil {
					return fmt.Errorf("failed to add Auth0 provider: %w", err)
				}
			}
		}
	}
	
	// Keycloak OAuth2
	if clientID := os.Getenv("KEYCLOAK_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET"); clientSecret != "" {
			if keycloakURL := os.Getenv("KEYCLOAK_URL"); keycloakURL != "" {
				if realm := os.Getenv("KEYCLOAK_REALM"); realm != "" {
					provider := factory.CreateKeycloak(clientID, clientSecret, keycloakURL, realm)
					if err := am.AddOAuth2Provider(provider); err != nil {
						return fmt.Errorf("failed to add Keycloak provider: %w", err)
					}
				}
			}
		}
	}
	
	// Azure AD OAuth2
	if clientID := os.Getenv("AZURE_CLIENT_ID"); clientID != "" {
		if clientSecret := os.Getenv("AZURE_CLIENT_SECRET"); clientSecret != "" {
			if tenantID := os.Getenv("AZURE_TENANT_ID"); tenantID != "" {
				provider := factory.CreateAzureAD(clientID, clientSecret, tenantID)
				if err := am.AddOAuth2Provider(provider); err != nil {
					return fmt.Errorf("failed to add Azure AD provider: %w", err)
				}
			}
		}
	}
	
	return nil
}

// GetTokenExpiration returns the token expiration duration
func (am *AuthManager) GetTokenExpiration() time.Duration {
	return am.Config.TokenExpiration
}

// GetRefreshTokenExpiration returns the refresh token expiration duration
func (am *AuthManager) GetRefreshTokenExpiration() time.Duration {
	return am.Config.RefreshExpiration
}

// UpdateConfig updates the authentication configuration
func (am *AuthManager) UpdateConfig(config *AuthConfig) error {
	am.Config = config
	
	// Recreate service with new config
	if service, ok := am.Service.(*Service); ok {
		service.config = config
		service.jwtManager = NewJWTManager(config)
		service.passwordManager = NewPasswordManager(config)
	}
	
	return nil
}

// GetStats returns authentication statistics
func (am *AuthManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"token_expiration":         am.Config.TokenExpiration.String(),
		"refresh_token_expiration": am.Config.RefreshExpiration.String(),
		"min_password_length":      am.Config.MinPasswordLength,
		"require_special_char":     am.Config.RequireSpecialChar,
		"require_number":          am.Config.RequireNumber,
		"require_uppercase":       am.Config.RequireUppercase,
		"require_lowercase":       am.Config.RequireLowercase,
		"max_login_attempts":      am.Config.MaxLoginAttempts,
		"lockout_duration":        am.Config.LockoutDuration.String(),
	}
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Example usage functions for integration

// ExampleUsage demonstrates how to use the authentication system
func ExampleUsage() {
	// This is an example of how to integrate the auth system
	// This would typically be in your main.go or server setup
	
	/*
	// 1. Create database connection
	db, err := database.NewDatabase(config.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	// 2. Create authentication manager
	authManager, err := auth.NewAuthManager(db)
	if err != nil {
		log.Fatal("Failed to create auth manager:", err)
	}
	
	// 3. Set up HTTP server
	mux := http.NewServeMux()
	
	// 4. Register authentication routes
	authManager.RegisterRoutes(mux)
	
	// 5. Protect specific endpoints
	mux.HandleFunc("/api/admin/users", authManager.ProtectAdminEndpoint(handleGetUsers))
	mux.HandleFunc("/api/user/profile", authManager.ProtectEndpoint(handleGetProfile))
	mux.HandleFunc("/api/moderator/logs", authManager.ProtectRoleEndpoint("moderator", handleGetLogs))
	
	// 6. Optional authentication for some endpoints
	mux.HandleFunc("/api/public/info", authManager.OptionalAuth(handleGetPublicInfo))
	
	// 7. Start server
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
	*/
}

// Example handler functions

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Admin-only endpoint logic here
	fmt.Fprintf(w, "Admin %s is accessing user list", user.Username)
}

func handleGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// User profile logic here
	fmt.Fprintf(w, "User %s profile", user.Username)
}

func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Moderator logs logic here
	fmt.Fprintf(w, "Moderator %s accessing logs", user.Username)
}

func handleGetPublicInfo(w http.ResponseWriter, r *http.Request) {
	// Optional authentication - check if user is present
	if user, ok := GetUserFromRequest(r); ok {
		fmt.Fprintf(w, "Hello %s! Here's the public info", user.Username)
	} else {
		fmt.Fprintf(w, "Hello anonymous user! Here's the public info")
	}
}