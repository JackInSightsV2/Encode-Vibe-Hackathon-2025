package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"qt1-middleware/models"
	"strings"
	"testing"
	"time"
)

// Mock implementations for testing

type MockUserService struct {
	users           map[int]*models.User
	usersByEmail    map[string]*models.User
	usersByUsername map[string]*models.User
	oauth2Accounts  map[string]*models.User // provider:providerUserID -> User
	nextUserID      int
}

func NewMockUserService() *MockUserService {
	return &MockUserService{
		users:           make(map[int]*models.User),
		usersByEmail:    make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
		oauth2Accounts:  make(map[string]*models.User),
		nextUserID:      1,
	}
}

func (m *MockUserService) Create(ctx context.Context, user *models.User) error {
	user.ID = m.nextUserID
	m.nextUserID++
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	m.usersByUsername[user.Username] = user
	
	return nil
}

func (m *MockUserService) GetByID(ctx context.Context, id int) (*models.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockUserService) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	if user, exists := m.usersByUsername[username]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockUserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if user, exists := m.usersByEmail[email]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockUserService) Update(ctx context.Context, user *models.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *MockUserService) Delete(ctx context.Context, id int) error {
	if user, exists := m.users[id]; exists {
		delete(m.users, id)
		delete(m.usersByEmail, user.Email)
		delete(m.usersByUsername, user.Username)
		return nil
	}
	return fmt.Errorf("user not found")
}

func (m *MockUserService) GetByOAuth2Provider(ctx context.Context, provider, providerUserID string) (*models.User, error) {
	key := provider + ":" + providerUserID
	if user, exists := m.oauth2Accounts[key]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("OAuth2 account not found")
}

func (m *MockUserService) LinkOAuth2Account(ctx context.Context, userID int, provider, providerUserID, email string) error {
	if user, exists := m.users[userID]; exists {
		key := provider + ":" + providerUserID
		m.oauth2Accounts[key] = user
		return nil
	}
	return fmt.Errorf("user not found")
}

func (m *MockUserService) UnlinkOAuth2Account(ctx context.Context, userID int, provider string) error {
	// Find and remove the OAuth2 account link
	for key := range m.oauth2Accounts {
		if strings.HasPrefix(key, provider+":") {
			if m.oauth2Accounts[key].ID == userID {
				delete(m.oauth2Accounts, key)
				return nil
			}
		}
	}
	return fmt.Errorf("OAuth2 account not found")
}

func (m *MockUserService) GetLinkedOAuth2Accounts(ctx context.Context, userID int) ([]*models.OAuth2Account, error) {
	var accounts []*models.OAuth2Account
	// This is simplified for testing
	return accounts, nil
}

type MockOAuth2AuthService struct {
	mockUserService *MockUserService
}

func NewMockOAuth2AuthService() *MockOAuth2AuthService {
	return &MockOAuth2AuthService{
		mockUserService: NewMockUserService(),
	}
}

func (m *MockOAuth2AuthService) GenerateToken(user *models.User) (string, error) {
	return fmt.Sprintf("token_%d_%d", user.ID, time.Now().Unix()), nil
}

func (m *MockOAuth2AuthService) ValidateToken(tokenString string) (*Claims, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockOAuth2AuthService) RefreshToken(tokenString string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *MockOAuth2AuthService) Login(username, password string) (*AuthResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockOAuth2AuthService) Logout(userID int) error {
	return fmt.Errorf("not implemented")
}

func (m *MockOAuth2AuthService) Register(req *RegisterRequest) (*models.User, error) {
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Role:     req.Role,
		Active:   true,
	}
	err := m.mockUserService.Create(context.Background(), user)
	return user, err
}

func (m *MockOAuth2AuthService) GetUserByToken(tokenString string) (*models.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockOAuth2AuthService) HashPassword(password string) (string, error) {
	return "$2a$10$" + password, nil
}

func (m *MockOAuth2AuthService) VerifyPassword(hashedPassword, password string) error {
	return nil
}

func (m *MockOAuth2AuthService) ChangePassword(userID int, oldPassword, newPassword string) error {
	return nil
}

// Implement UserService interface
func (m *MockOAuth2AuthService) Create(ctx context.Context, user *models.User) error {
	return m.mockUserService.Create(ctx, user)
}

func (m *MockOAuth2AuthService) GetByID(ctx context.Context, id int) (*models.User, error) {
	return m.mockUserService.GetByID(ctx, id)
}

func (m *MockOAuth2AuthService) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return m.mockUserService.GetByUsername(ctx, username)
}

func (m *MockOAuth2AuthService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.mockUserService.GetByEmail(ctx, email)
}

func (m *MockOAuth2AuthService) Update(ctx context.Context, user *models.User) error {
	return m.mockUserService.Update(ctx, user)
}

func (m *MockOAuth2AuthService) Delete(ctx context.Context, id int) error {
	return m.mockUserService.Delete(ctx, id)
}

func (m *MockOAuth2AuthService) GetByOAuth2Provider(ctx context.Context, provider, providerUserID string) (*models.User, error) {
	return m.mockUserService.GetByOAuth2Provider(ctx, provider, providerUserID)
}

func (m *MockOAuth2AuthService) LinkOAuth2Account(ctx context.Context, userID int, provider, providerUserID, email string) error {
	return m.mockUserService.LinkOAuth2Account(ctx, userID, provider, providerUserID, email)
}

func (m *MockOAuth2AuthService) UnlinkOAuth2Account(ctx context.Context, userID int, provider string) error {
	return m.mockUserService.UnlinkOAuth2Account(ctx, userID, provider)
}

func (m *MockOAuth2AuthService) GetLinkedOAuth2Accounts(ctx context.Context, userID int) ([]*models.OAuth2Account, error) {
	return m.mockUserService.GetLinkedOAuth2Accounts(ctx, userID)
}

// Test OAuth2Manager

func TestOAuth2Manager_AddProvider(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := GoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback")
	// Disable OIDC discovery for testing
	provider.DiscoveryURL = ""

	err := manager.AddProvider(provider)
	if err != nil {
		t.Fatalf("Failed to add provider: %v", err)
	}

	retrievedProvider, err := manager.GetProvider("google")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if retrievedProvider.ID != "google" {
		t.Errorf("Expected provider ID 'google', got %s", retrievedProvider.ID)
	}
}

func TestOAuth2Manager_GenerateAuthURL(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := GoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback")
	// Disable OIDC discovery for testing
	provider.DiscoveryURL = ""
	manager.AddProvider(provider)

	authURL, err := manager.GenerateAuthURL("google", "/dashboard")
	if err != nil {
		t.Fatalf("Failed to generate auth URL: %v", err)
	}

	if !strings.Contains(authURL, "accounts.google.com") {
		t.Errorf("Auth URL should contain Google's auth endpoint")
	}

	if !strings.Contains(authURL, "client_id=test-client-id") {
		t.Errorf("Auth URL should contain client ID")
	}

	if !strings.Contains(authURL, "state=") {
		t.Errorf("Auth URL should contain state parameter")
	}
}

func TestOAuth2Manager_FindOrCreateUser(t *testing.T) {
	config := DefaultOAuth2Config()
	config.AllowSignup = true
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := GoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback")
	provider.AutoProvision = true

	userInfo := &OAuth2UserInfo{
		ID:       "google-123",
		Email:    "test@example.com",
		Username: "testuser",
		Name:     "Test User",
		Verified: true,
		Raw:      make(map[string]interface{}),
	}

	// Test user creation
	user, err := manager.findOrCreateUser(context.Background(), provider, userInfo)
	if err != nil {
		t.Fatalf("Failed to find or create user: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	if user.Role != "user" {
		t.Errorf("Expected role 'user', got %s", user.Role)
	}

	// Test finding existing user
	user2, err := manager.findOrCreateUser(context.Background(), provider, userInfo)
	if err != nil {
		t.Fatalf("Failed to find existing user: %v", err)
	}

	if user2.ID != user.ID {
		t.Errorf("Should find the same user, got different IDs: %d vs %d", user.ID, user2.ID)
	}
}

func TestOAuth2Manager_EnsureUniqueUsername(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	// Create a user with username "testuser"
	existingUser := &models.User{
		Username: "testuser",
		Email:    "existing@example.com",
		Role:     "user",
		Active:   true,
	}
	authService.Create(context.Background(), existingUser)

	// Test unique username generation
	username := manager.ensureUniqueUsername(context.Background(), "testuser", "new@example.com")
	if username == "testuser" {
		t.Errorf("Should generate unique username, got existing username")
	}

	if !strings.HasPrefix(username, "testuser") {
		t.Errorf("New username should be based on preferred username")
	}
}

func TestOAuth2Manager_MapUserRole(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := &OAuth2Provider{
		UserMapping: &UserMapping{
			RoleField: "roles",
			RoleMapping: map[string]string{
				"admin": "admin",
				"user":  "user",
			},
		},
	}

	userInfo := &OAuth2UserInfo{
		Raw: map[string]interface{}{
			"roles": "admin",
		},
	}

	role := manager.mapUserRole(provider, userInfo)
	if role != "admin" {
		t.Errorf("Expected role 'admin', got %s", role)
	}

	// Test array of roles
	userInfo.Raw["roles"] = []interface{}{"user", "moderator"}
	role = manager.mapUserRole(provider, userInfo)
	if role != "user" {
		t.Errorf("Expected role 'user' (first in array), got %s", role)
	}
}

// Test OAuth2Handlers

func TestOAuth2Handlers_ListProviders(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)
	handlers := NewOAuth2Handlers(manager, nil, authService)

	// Add test provider
	provider := GoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback")
	// Disable OIDC discovery for testing
	provider.DiscoveryURL = ""
	manager.AddProvider(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/oauth2/providers", nil)
	w := httptest.NewRecorder()

	handlers.ListProviders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}

	providers := response["providers"].(map[string]interface{})
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if _, exists := providers["google"]; !exists {
		t.Errorf("Expected Google provider to be listed")
	}
}

func TestOAuth2Handlers_InitiateLogin(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)
	handlers := NewOAuth2Handlers(manager, nil, authService)

	// Add test provider
	provider := GoogleProvider("test-client-id", "test-client-secret", "http://localhost/callback")
	// Disable OIDC discovery for testing
	provider.DiscoveryURL = ""
	manager.AddProvider(provider)

	requestBody := `{"provider": "google", "redirect_url": "/dashboard"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth2/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.InitiateLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response["success"].(bool) {
		t.Errorf("Expected success to be true")
	}

	authURL := response["auth_url"].(string)
	if !strings.Contains(authURL, "accounts.google.com") {
		t.Errorf("Auth URL should contain Google's endpoint")
	}
}

// Test provider configurations

func TestGoogleProvider(t *testing.T) {
	provider := GoogleProvider("test-id", "test-secret", "http://localhost/callback")

	if provider.ID != "google" {
		t.Errorf("Expected ID 'google', got %s", provider.ID)
	}

	if provider.Name != "Google" {
		t.Errorf("Expected name 'Google', got %s", provider.Name)
	}

	if provider.ClientID != "test-id" {
		t.Errorf("Expected client ID 'test-id', got %s", provider.ClientID)
	}

	if !strings.Contains(provider.AuthURL, "accounts.google.com") {
		t.Errorf("Auth URL should contain Google's domain")
	}

	if !contains(provider.Scopes, "openid") {
		t.Errorf("Scopes should include 'openid'")
	}
}

func TestMicrosoftProvider(t *testing.T) {
	provider := MicrosoftProvider("test-id", "test-secret", "http://localhost/callback")

	if provider.ID != "microsoft" {
		t.Errorf("Expected ID 'microsoft', got %s", provider.ID)
	}

	if !strings.Contains(provider.AuthURL, "login.microsoftonline.com") {
		t.Errorf("Auth URL should contain Microsoft's domain")
	}

	if !contains(provider.Scopes, "User.Read") {
		t.Errorf("Scopes should include 'User.Read'")
	}
}

func TestGitHubProvider(t *testing.T) {
	provider := GitHubProvider("test-id", "test-secret", "http://localhost/callback")

	if provider.ID != "github" {
		t.Errorf("Expected ID 'github', got %s", provider.ID)
	}

	if !strings.Contains(provider.AuthURL, "github.com") {
		t.Errorf("Auth URL should contain GitHub's domain")
	}

	if !contains(provider.Scopes, "user:email") {
		t.Errorf("Scopes should include 'user:email'")
	}
}

// Test OIDC validator

func TestOIDCValidator_ValidateAudience(t *testing.T) {
	validator := NewOIDCValidator()

	// Test string audience
	err := validator.validateAudience("test-client-id", "test-client-id")
	if err != nil {
		t.Errorf("Should validate correct string audience: %v", err)
	}

	err = validator.validateAudience("wrong-client-id", "test-client-id")
	if err == nil {
		t.Errorf("Should reject wrong string audience")
	}

	// Test array audience
	audience := []interface{}{"test-client-id", "other-client-id"}
	err = validator.validateAudience(audience, "test-client-id")
	if err != nil {
		t.Errorf("Should validate correct array audience: %v", err)
	}

	audience = []interface{}{"wrong-client-id", "other-client-id"}
	err = validator.validateAudience(audience, "test-client-id")
	if err == nil {
		t.Errorf("Should reject wrong array audience")
	}
}

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory("http://localhost:8080")

	// Test Google provider creation
	provider := factory.CreateGoogle("test-id", "test-secret")
	expectedRedirect := "http://localhost:8080/oauth2/callback/google"
	if provider.RedirectURL != expectedRedirect {
		t.Errorf("Expected redirect URL %s, got %s", expectedRedirect, provider.RedirectURL)
	}

	// Test Microsoft provider creation
	provider = factory.CreateMicrosoft("test-id", "test-secret")
	expectedRedirect = "http://localhost:8080/oauth2/callback/microsoft"
	if provider.RedirectURL != expectedRedirect {
		t.Errorf("Expected redirect URL %s, got %s", expectedRedirect, provider.RedirectURL)
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Test OAuth2 configuration

func TestOAuth2Config_Defaults(t *testing.T) {
	config := DefaultOAuth2Config()

	if config.Enabled {
		t.Errorf("OAuth2 should be disabled by default")
	}

	if config.StateTimeout != 10*time.Minute {
		t.Errorf("Expected state timeout 10 minutes, got %v", config.StateTimeout)
	}

	if !config.AllowSignup {
		t.Errorf("Allow signup should be true by default")
	}

	if config.RequireVerified {
		t.Errorf("Require verified should be false by default")
	}
}

// Test error cases

func TestOAuth2Manager_GetProvider_NotFound(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	_, err := manager.GetProvider("nonexistent")
	if err == nil {
		t.Errorf("Should return error for nonexistent provider")
	}
}

func TestOAuth2Manager_GetProvider_Disabled(t *testing.T) {
	config := DefaultOAuth2Config()
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := GoogleProvider("test-id", "test-secret", "http://localhost/callback")
	provider.Enabled = false
	manager.AddProvider(provider)

	_, err := manager.GetProvider("google")
	if err == nil {
		t.Errorf("Should return error for disabled provider")
	}
}

func TestOAuth2Manager_FindOrCreateUser_SignupDisabled(t *testing.T) {
	config := DefaultOAuth2Config()
	config.AllowSignup = false
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := GoogleProvider("test-id", "test-secret", "http://localhost/callback")
	provider.AutoProvision = false

	userInfo := &OAuth2UserInfo{
		ID:       "google-123",
		Email:    "test@example.com",
		Username: "testuser",
		Verified: true,
		Raw:      make(map[string]interface{}),
	}

	_, err := manager.findOrCreateUser(context.Background(), provider, userInfo)
	if err == nil {
		t.Errorf("Should return error when signup is disabled")
	}
}

func TestOAuth2Manager_FindOrCreateUser_EmailNotVerified(t *testing.T) {
	config := DefaultOAuth2Config()
	config.RequireVerified = true
	authService := NewMockOAuth2AuthService()
	manager := NewOAuth2Manager(config, authService, authService)

	provider := GoogleProvider("test-id", "test-secret", "http://localhost/callback")

	userInfo := &OAuth2UserInfo{
		ID:       "google-123",
		Email:    "test@example.com",
		Username: "testuser",
		Verified: false, // Not verified
		Raw:      make(map[string]interface{}),
	}

	_, err := manager.findOrCreateUser(context.Background(), provider, userInfo)
	if err == nil {
		t.Errorf("Should return error when email is not verified and verification is required")
	}
}