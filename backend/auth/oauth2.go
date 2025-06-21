package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"qt1-middleware/models"
	"strings"
	"time"
)

// OAuth2Provider represents an OAuth2 identity provider configuration
type OAuth2Provider struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name" json:"name"`
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
	AuthURL      string `yaml:"auth_url" json:"auth_url"`
	TokenURL     string `yaml:"token_url" json:"token_url"`
	UserInfoURL  string `yaml:"user_info_url" json:"user_info_url"`
	Scopes       []string `yaml:"scopes" json:"scopes"`
	RedirectURL  string `yaml:"redirect_url" json:"redirect_url"`
	
	// OpenID Connect specific
	DiscoveryURL    string `yaml:"discovery_url,omitempty" json:"discovery_url,omitempty"`
	JWKSEndpoint    string `yaml:"jwks_endpoint,omitempty" json:"jwks_endpoint,omitempty"`
	Issuer          string `yaml:"issuer,omitempty" json:"issuer,omitempty"`
	
	// Provider-specific settings
	UserMapping     *UserMapping `yaml:"user_mapping,omitempty" json:"user_mapping,omitempty"`
	AutoProvision   bool         `yaml:"auto_provision" json:"auto_provision"`
	DefaultRole     string       `yaml:"default_role" json:"default_role"`
	Enabled         bool         `yaml:"enabled" json:"enabled"`
}

// UserMapping defines how to map OAuth2 user info to internal user fields
type UserMapping struct {
	EmailField    string `yaml:"email_field" json:"email_field"`
	UsernameField string `yaml:"username_field" json:"username_field"`
	NameField     string `yaml:"name_field" json:"name_field"`
	RoleField     string `yaml:"role_field,omitempty" json:"role_field,omitempty"`
	RoleMapping   map[string]string `yaml:"role_mapping,omitempty" json:"role_mapping,omitempty"`
}

// OAuth2Config holds OAuth2/OIDC configuration
type OAuth2Config struct {
	Enabled         bool                        `yaml:"enabled" json:"enabled"`
	Providers       map[string]*OAuth2Provider  `yaml:"providers" json:"providers"`
	StateTimeout    time.Duration               `yaml:"state_timeout" json:"state_timeout"`
	AllowSignup     bool                        `yaml:"allow_signup" json:"allow_signup"`
	RequireVerified bool                        `yaml:"require_verified" json:"require_verified"`
}

// OAuth2Manager manages OAuth2/OIDC authentication flows
type OAuth2Manager struct {
	config       *OAuth2Config
	providers    map[string]*OAuth2Provider
	stateStore   map[string]*OAuth2State
	authService  AuthService
	userService  UserService
	httpClient   *http.Client
}

// OAuth2State stores OAuth2 state information
type OAuth2State struct {
	State       string    `json:"state"`
	Provider    string    `json:"provider"`
	RedirectURL string    `json:"redirect_url"`
	ExpiresAt   time.Time `json:"expires_at"`
	Nonce       string    `json:"nonce,omitempty"`
	UserID      int       `json:"user_id,omitempty"`
}

// OAuth2TokenResponse represents an OAuth2 token response
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuth2UserInfo represents user information from OAuth2 provider
type OAuth2UserInfo struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Username string                 `json:"username"`
	Name     string                 `json:"name"`
	Picture  string                 `json:"picture,omitempty"`
	Verified bool                   `json:"verified"`
	Raw      map[string]interface{} `json:"raw"`
}

// OpenIDDiscovery represents OpenID Connect discovery document
type OpenIDDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSEndpoint          string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
	ResponseTypesSupported []string `json:"response_types_supported"`
}

// DefaultOAuth2Config returns default OAuth2 configuration
func DefaultOAuth2Config() *OAuth2Config {
	return &OAuth2Config{
		Enabled:         false,
		Providers:       make(map[string]*OAuth2Provider),
		StateTimeout:    10 * time.Minute,
		AllowSignup:     true,
		RequireVerified: false,
	}
}

// NewOAuth2Manager creates a new OAuth2 manager
func NewOAuth2Manager(config *OAuth2Config, authService AuthService, userService UserService) *OAuth2Manager {
	if config == nil {
		config = DefaultOAuth2Config()
	}

	return &OAuth2Manager{
		config:      config,
		providers:   config.Providers,
		stateStore:  make(map[string]*OAuth2State),
		authService: authService,
		userService: userService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddProvider adds an OAuth2 provider configuration
func (om *OAuth2Manager) AddProvider(provider *OAuth2Provider) error {
	if provider.ID == "" {
		return fmt.Errorf("provider ID is required")
	}

	// Auto-discover OpenID Connect endpoints if discovery URL is provided
	if provider.DiscoveryURL != "" {
		if err := om.discoverOIDCEndpoints(provider); err != nil {
			return fmt.Errorf("failed to discover OIDC endpoints: %w", err)
		}
	}

	// Set default user mapping if not provided
	if provider.UserMapping == nil {
		provider.UserMapping = &UserMapping{
			EmailField:    "email",
			UsernameField: "preferred_username",
			NameField:     "name",
		}
	}

	om.providers[provider.ID] = provider
	return nil
}

// GetProvider returns a provider by ID
func (om *OAuth2Manager) GetProvider(providerID string) (*OAuth2Provider, error) {
	provider, exists := om.providers[providerID]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	if !provider.Enabled {
		return nil, fmt.Errorf("provider disabled: %s", providerID)
	}

	return provider, nil
}

// GenerateAuthURL generates an OAuth2 authorization URL
func (om *OAuth2Manager) GenerateAuthURL(providerID, redirectURL string) (string, error) {
	provider, err := om.GetProvider(providerID)
	if err != nil {
		return "", err
	}

	// Generate state parameter
	state, err := om.generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state information
	stateInfo := &OAuth2State{
		State:       state,
		Provider:    providerID,
		RedirectURL: redirectURL,
		ExpiresAt:   time.Now().Add(om.config.StateTimeout),
	}

	// Generate nonce for OpenID Connect
	if om.isOIDCProvider(provider) {
		nonce, err := om.generateNonce()
		if err != nil {
			return "", fmt.Errorf("failed to generate nonce: %w", err)
		}
		stateInfo.Nonce = nonce
	}

	om.stateStore[state] = stateInfo

	// Build authorization URL
	authURL, err := url.Parse(provider.AuthURL)
	if err != nil {
		return "", fmt.Errorf("invalid auth URL: %w", err)
	}

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("redirect_uri", provider.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(provider.Scopes, " "))
	params.Set("state", state)

	if stateInfo.Nonce != "" {
		params.Set("nonce", stateInfo.Nonce)
	}

	authURL.RawQuery = params.Encode()
	return authURL.String(), nil
}

// HandleCallback handles OAuth2 callback
func (om *OAuth2Manager) HandleCallback(ctx context.Context, state, code string) (*models.User, *models.Session, error) {
	// Validate state
	stateInfo, exists := om.stateStore[state]
	if !exists {
		return nil, nil, fmt.Errorf("invalid state parameter")
	}

	if time.Now().After(stateInfo.ExpiresAt) {
		delete(om.stateStore, state)
		return nil, nil, fmt.Errorf("state expired")
	}

	// Clean up state
	defer delete(om.stateStore, state)

	provider, err := om.GetProvider(stateInfo.Provider)
	if err != nil {
		return nil, nil, err
	}

	// Exchange code for token
	tokenResponse, err := om.exchangeCodeForToken(provider, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info
	userInfo, err := om.getUserInfo(provider, tokenResponse.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Validate ID token if present (OIDC)
	if tokenResponse.IDToken != "" {
		if err := om.validateIDToken(provider, tokenResponse.IDToken, stateInfo.Nonce); err != nil {
			return nil, nil, fmt.Errorf("ID token validation failed: %w", err)
		}
	}

	// Find or create user
	user, err := om.findOrCreateUser(ctx, provider, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Create session (assuming we have access to session manager)
	// This would typically be passed in or accessed through the auth service
	session := &models.Session{
		UserID:    user.ID,
		Token:     "", // Will be generated
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "", // Would be extracted from request
		UserAgent: "", // Would be extracted from request
		Active:    true,
	}

	return user, session, nil
}

// Private helper methods

func (om *OAuth2Manager) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (om *OAuth2Manager) generateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (om *OAuth2Manager) discoverOIDCEndpoints(provider *OAuth2Provider) error {
	resp, err := om.httpClient.Get(provider.DiscoveryURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery failed with status: %d", resp.StatusCode)
	}

	var discovery OpenIDDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return err
	}

	// Update provider endpoints from discovery
	provider.AuthURL = discovery.AuthorizationEndpoint
	provider.TokenURL = discovery.TokenEndpoint
	provider.UserInfoURL = discovery.UserInfoEndpoint
	provider.JWKSEndpoint = discovery.JWKSEndpoint
	provider.Issuer = discovery.Issuer

	return nil
}

func (om *OAuth2Manager) isOIDCProvider(provider *OAuth2Provider) bool {
	return provider.DiscoveryURL != "" || provider.JWKSEndpoint != ""
}

func (om *OAuth2Manager) exchangeCodeForToken(provider *OAuth2Provider, code string) (*OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", provider.RedirectURL)

	req, err := http.NewRequest(http.MethodPost, provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := om.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %d %s", resp.StatusCode, string(body))
	}

	var tokenResponse OAuth2TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

func (om *OAuth2Manager) getUserInfo(provider *OAuth2Provider, accessToken string) (*OAuth2UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, provider.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := om.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed: %d %s", resp.StatusCode, string(body))
	}

	var rawUserInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawUserInfo); err != nil {
		return nil, err
	}

	return om.mapUserInfo(provider, rawUserInfo), nil
}

func (om *OAuth2Manager) mapUserInfo(provider *OAuth2Provider, raw map[string]interface{}) *OAuth2UserInfo {
	mapping := provider.UserMapping
	
	userInfo := &OAuth2UserInfo{
		Raw: raw,
	}

	// Extract email
	if email, ok := raw[mapping.EmailField].(string); ok {
		userInfo.Email = email
	}

	// Extract username
	if username, ok := raw[mapping.UsernameField].(string); ok {
		userInfo.Username = username
	} else if userInfo.Email != "" {
		// Fallback to email if username not available
		userInfo.Username = userInfo.Email
	}

	// Extract name
	if name, ok := raw[mapping.NameField].(string); ok {
		userInfo.Name = name
	}

	// Extract ID
	if id, ok := raw["sub"].(string); ok {
		userInfo.ID = id
	} else if id, ok := raw["id"].(string); ok {
		userInfo.ID = id
	}

	// Extract picture
	if picture, ok := raw["picture"].(string); ok {
		userInfo.Picture = picture
	}

	// Extract verified status
	if verified, ok := raw["email_verified"].(bool); ok {
		userInfo.Verified = verified
	} else {
		userInfo.Verified = true // Default to verified for OAuth2 providers
	}

	return userInfo
}

func (om *OAuth2Manager) validateIDToken(provider *OAuth2Provider, idToken, nonce string) error {
	// This is a simplified validation - in production, you would:
	// 1. Verify the JWT signature using JWKS
	// 2. Validate issuer, audience, expiration
	// 3. Verify nonce if provided
	// For now, we'll just check that the token exists
	if idToken == "" {
		return fmt.Errorf("empty ID token")
	}
	
	// TODO: Implement full JWT validation
	return nil
}

func (om *OAuth2Manager) findOrCreateUser(ctx context.Context, provider *OAuth2Provider, userInfo *OAuth2UserInfo) (*models.User, error) {
	// Check if OAuth2 signup is allowed
	if !om.config.AllowSignup && !provider.AutoProvision {
		return nil, fmt.Errorf("OAuth2 signup not allowed")
	}

	// Check email verification requirement
	if om.config.RequireVerified && !userInfo.Verified {
		return nil, fmt.Errorf("email verification required")
	}

	// Try to find existing OAuth2 account link first
	if existingUser, err := om.userService.GetByOAuth2Provider(ctx, provider.ID, userInfo.ID); err == nil && existingUser != nil {
		// Update the OAuth2 account with latest info if needed
		om.updateOAuth2Account(ctx, existingUser.ID, provider, userInfo)
		return existingUser, nil
	}

	// Try to find existing user by email
	var existingUser *models.User
	if userInfo.Email != "" {
		if user, err := om.userService.GetByEmail(ctx, userInfo.Email); err == nil && user != nil {
			existingUser = user
		}
	}

	// If no existing user found by email, try by username
	if existingUser == nil && userInfo.Username != "" {
		if user, err := om.userService.GetByUsername(ctx, userInfo.Username); err == nil && user != nil {
			existingUser = user
		}
	}

	// If existing user found, link the OAuth2 account
	if existingUser != nil {
		if err := om.userService.LinkOAuth2Account(ctx, existingUser.ID, provider.ID, userInfo.ID, userInfo.Email); err != nil {
			return nil, fmt.Errorf("failed to link OAuth2 account: %w", err)
		}
		return existingUser, nil
	}

	// Create new user if auto-provisioning is enabled
	if provider.AutoProvision {
		// Ensure username is unique
		username := om.ensureUniqueUsername(ctx, userInfo.Username, userInfo.Email)
		
		// Map role from provider if role mapping is configured
		role := om.mapUserRole(provider, userInfo)
		if role == "" {
			role = provider.DefaultRole
		}

		user := &models.User{
			Username:     username,
			Email:        userInfo.Email,
			PasswordHash: om.generateRandomPassword(), // Generate a random password hash for OAuth2 users
			Role:         role,
			Active:       true,
		}

		if err := om.userService.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Link the OAuth2 account to the new user
		if err := om.userService.LinkOAuth2Account(ctx, user.ID, provider.ID, userInfo.ID, userInfo.Email); err != nil {
			return nil, fmt.Errorf("failed to link OAuth2 account to new user: %w", err)
		}

		return user, nil
	}

	return nil, fmt.Errorf("user not found and auto-provisioning disabled")
}

// GetProviders returns list of enabled providers
func (om *OAuth2Manager) GetProviders() map[string]*OAuth2Provider {
	enabled := make(map[string]*OAuth2Provider)
	for id, provider := range om.providers {
		if provider.Enabled {
			enabled[id] = provider
		}
	}
	return enabled
}

// CleanupExpiredStates removes expired state entries
func (om *OAuth2Manager) CleanupExpiredStates() {
	now := time.Now()
	for state, stateInfo := range om.stateStore {
		if now.After(stateInfo.ExpiresAt) {
			delete(om.stateStore, state)
		}
	}
}

// ensureUniqueUsername ensures the username is unique by appending numbers if necessary
func (om *OAuth2Manager) ensureUniqueUsername(ctx context.Context, preferredUsername, email string) string {
	if preferredUsername == "" {
		// Use email prefix as username if no preferred username
		if email != "" {
			parts := strings.Split(email, "@")
			preferredUsername = parts[0]
		} else {
			preferredUsername = "user"
		}
	}

	// Clean username (remove non-alphanumeric characters)
	username := strings.ToLower(preferredUsername)
	username = strings.ReplaceAll(username, " ", "")
	username = strings.ReplaceAll(username, ".", "")
	username = strings.ReplaceAll(username, "-", "")
	
	// Limit length
	if len(username) > 30 {
		username = username[:30]
	}

	baseUsername := username
	counter := 1

	// Check if username exists and increment counter if needed
	for {
		if _, err := om.userService.GetByUsername(ctx, username); err != nil {
			// Username doesn't exist, we can use it
			break
		}
		
		username = fmt.Sprintf("%s%d", baseUsername, counter)
		counter++
		
		// Prevent infinite loop
		if counter > 9999 {
			username = fmt.Sprintf("%s_%d", baseUsername, time.Now().Unix())
			break
		}
	}

	return username
}

// mapUserRole maps OAuth2 user info to internal role based on provider configuration
func (om *OAuth2Manager) mapUserRole(provider *OAuth2Provider, userInfo *OAuth2UserInfo) string {
	if provider.UserMapping == nil || provider.UserMapping.RoleField == "" {
		return ""
	}

	// Extract role from raw user info
	var roleValue interface{}
	rolePath := strings.Split(provider.UserMapping.RoleField, ".")
	
	current := userInfo.Raw
	for i, part := range rolePath {
		if current == nil {
			return ""
		}
		if i == len(rolePath)-1 {
			// Last part of the path
			roleValue = current[part]
		} else {
			// Intermediate part of the path
			if nextMap, ok := current[part].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return ""
			}
		}
	}

	// Convert role value to string
	var roleStr string
	switch v := roleValue.(type) {
	case string:
		roleStr = v
	case []interface{}:
		// Handle array of roles (take first one)
		if len(v) > 0 {
			if str, ok := v[0].(string); ok {
				roleStr = str
			}
		}
	default:
		return ""
	}

	// Map role using role mapping if configured
	if provider.UserMapping.RoleMapping != nil {
		if mappedRole, exists := provider.UserMapping.RoleMapping[roleStr]; exists {
			return mappedRole
		}
	}

	// Return the role as-is if no mapping found
	return roleStr
}

// generateRandomPassword generates a random password hash for OAuth2 users
func (om *OAuth2Manager) generateRandomPassword() string {
	// Generate a random 32-byte password
	bytes := make([]byte, 32)
	rand.Read(bytes)
	randomPassword := hex.EncodeToString(bytes)
	
	// This should be hashed using the same method as regular passwords
	// For now, we'll return a placeholder hash
	// In a real implementation, you'd use bcrypt or similar
	return "$2a$10$" + randomPassword[:50] + "OAUTH2USER"
}

// updateOAuth2Account updates OAuth2 account information
func (om *OAuth2Manager) updateOAuth2Account(ctx context.Context, userID int, provider *OAuth2Provider, userInfo *OAuth2UserInfo) error {
	// This is a placeholder - you would implement actual OAuth2 account update logic here
	// This might involve updating stored tokens, user info, etc.
	return nil
}