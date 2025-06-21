package auth

import (
	"context"
	"qt1-middleware/models"
	"qt1-middleware/repositories"
	"testing"
	"time"
)

// Mock API Key Repository for testing
type MockAPIKeyRepository struct {
	keys       map[string]*models.APIKey
	permissions map[string][]string
	usage      map[string][]*models.APIKeyUsage
	stats      *models.APIKeyStats
}

func NewMockAPIKeyRepository() *MockAPIKeyRepository {
	return &MockAPIKeyRepository{
		keys:        make(map[string]*models.APIKey),
		permissions: make(map[string][]string),
		usage:       make(map[string][]*models.APIKeyUsage),
		stats: &models.APIKeyStats{
			TotalKeys:    0,
			ActiveKeys:   0,
			ExpiredKeys:  0,
			DisabledKeys: 0,
		},
	}
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	m.keys[apiKey.ID] = apiKey
	if len(apiKey.Permissions) > 0 {
		m.permissions[apiKey.ID] = apiKey.Permissions
	}
	m.stats.TotalKeys++
	if apiKey.Active {
		m.stats.ActiveKeys++
	}
	return nil
}

func (m *MockAPIKeyRepository) GetByID(ctx context.Context, id string) (*models.APIKey, error) {
	if key, exists := m.keys[id]; exists {
		// Load permissions
		if perms, hasPerms := m.permissions[id]; hasPerms {
			key.Permissions = perms
		}
		return key, nil
	}
	return nil, nil
}

func (m *MockAPIKeyRepository) Update(ctx context.Context, apiKey *models.APIKey) error {
	if _, exists := m.keys[apiKey.ID]; !exists {
		return repositories.ErrNotFound
	}
	m.keys[apiKey.ID] = apiKey
	if len(apiKey.Permissions) > 0 {
		m.permissions[apiKey.ID] = apiKey.Permissions
	}
	return nil
}

func (m *MockAPIKeyRepository) Delete(ctx context.Context, id string) error {
	if _, exists := m.keys[id]; !exists {
		return repositories.ErrNotFound
	}
	delete(m.keys, id)
	delete(m.permissions, id)
	delete(m.usage, id)
	m.stats.TotalKeys--
	return nil
}

func (m *MockAPIKeyRepository) GetByUserID(ctx context.Context, userID int, activeOnly bool) ([]*models.APIKey, error) {
	var result []*models.APIKey
	for _, key := range m.keys {
		if key.UserID == userID {
			if !activeOnly || key.Active {
				// Load permissions
				if perms, hasPerms := m.permissions[key.ID]; hasPerms {
					key.Permissions = perms
				}
				result = append(result, key)
			}
		}
	}
	return result, nil
}

func (m *MockAPIKeyRepository) List(ctx context.Context, filters *models.APIKeyFilters) ([]*models.APIKey, error) {
	var result []*models.APIKey
	for _, key := range m.keys {
		if perms, hasPerms := m.permissions[key.ID]; hasPerms {
			key.Permissions = perms
		}
		result = append(result, key)
	}
	return result, nil
}

func (m *MockAPIKeyRepository) UpdateLastUsed(ctx context.Context, keyID string, lastUsed time.Time) error {
	if key, exists := m.keys[keyID]; exists {
		key.LastUsed = &lastUsed
		key.UsageCount++
		return nil
	}
	return repositories.ErrNotFound
}

func (m *MockAPIKeyRepository) DeleteExpired(ctx context.Context, cutoff time.Time) (int, error) {
	count := 0
	for id, key := range m.keys {
		if key.ExpiresAt != nil && key.ExpiresAt.Before(cutoff) {
			delete(m.keys, id)
			delete(m.permissions, id)
			delete(m.usage, id)
			count++
		}
	}
	return count, nil
}

func (m *MockAPIKeyRepository) GetUsage(ctx context.Context, keyID string, startDate time.Time) ([]*models.APIKeyUsage, error) {
	if usage, exists := m.usage[keyID]; exists {
		return usage, nil
	}
	return []*models.APIKeyUsage{}, nil
}

func (m *MockAPIKeyRepository) RecordUsage(ctx context.Context, keyID, ipAddress, userAgent string) error {
	// Simple implementation for testing
	return nil
}

func (m *MockAPIKeyRepository) GetStats(ctx context.Context) (*models.APIKeyStats, error) {
	return m.stats, nil
}

func (m *MockAPIKeyRepository) Count(ctx context.Context) (int, error) {
	return len(m.keys), nil
}

func (m *MockAPIKeyRepository) CountActive(ctx context.Context) (int, error) {
	count := 0
	for _, key := range m.keys {
		if key.Active {
			count++
		}
	}
	return count, nil
}

func (m *MockAPIKeyRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	count := 0
	for _, key := range m.keys {
		if key.UserID == userID {
			count++
		}
	}
	return count, nil
}

// Mock User Repository for testing
type MockUserRepository struct {
	users map[int]*models.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[int]*models.User),
	}
}

func (m *MockUserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, repositories.ErrNotFound
}

// Add other required methods with simple implementations
func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	if _, exists := m.users[user.ID]; !exists {
		return repositories.ErrNotFound
	}
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id int) error {
	if _, exists := m.users[id]; !exists {
		return repositories.ErrNotFound
	}
	delete(m.users, id)
	return nil
}

// Test API Key Manager functionality

func TestAPIKeyManager_GenerateAPIKey(t *testing.T) {
	// Setup
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	config := DefaultAPIKeyConfig()
	manager := NewAPIKeyManager(apiKeyRepo, userRepo, config)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}
	userRepo.Create(context.Background(), user)

	// Test API key generation
	req := &models.APIKeyCreateRequest{
		Name:        "Test Key",
		Description: "A test API key",
		Permissions: []string{"config:read", "logs:read"},
	}

	apiKey, plainKey, err := manager.GenerateAPIKey(context.Background(), user.ID, req)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Verify results
	if apiKey == nil {
		t.Fatal("API key is nil")
	}
	if plainKey == "" {
		t.Fatal("Plain key is empty")
	}
	if apiKey.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, apiKey.Name)
	}
	if apiKey.UserID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, apiKey.UserID)
	}
	if !apiKey.Active {
		t.Error("API key should be active")
	}
	if len(apiKey.Permissions) != len(req.Permissions) {
		t.Errorf("Expected %d permissions, got %d", len(req.Permissions), len(apiKey.Permissions))
	}
}

func TestAPIKeyManager_ValidateAPIKey(t *testing.T) {
	// Setup
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	config := DefaultAPIKeyConfig()
	manager := NewAPIKeyManager(apiKeyRepo, userRepo, config)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}
	userRepo.Create(context.Background(), user)

	// Generate an API key
	req := &models.APIKeyCreateRequest{
		Name:        "Test Key",
		Description: "A test API key",
		Permissions: []string{"config:read"},
	}

	apiKey, plainKey, err := manager.GenerateAPIKey(context.Background(), user.ID, req)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Format the full key
	fullKey := manager.FormatAPIKey(apiKey.ID, plainKey)

	// Test validation
	validatedKey, validatedUser, err := manager.ValidateAPIKey(context.Background(), fullKey)
	if err != nil {
		t.Fatalf("Failed to validate API key: %v", err)
	}

	// Verify results
	if validatedKey.ID != apiKey.ID {
		t.Errorf("Expected key ID %s, got %s", apiKey.ID, validatedKey.ID)
	}
	if validatedUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, validatedUser.ID)
	}

	// Test invalid key
	_, _, err = manager.ValidateAPIKey(context.Background(), "invalid.key")
	if err == nil {
		t.Error("Expected error for invalid API key")
	}
}

func TestAPIKeyManager_ValidateKeyPermissions(t *testing.T) {
	// Setup
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	config := DefaultAPIKeyConfig()
	manager := NewAPIKeyManager(apiKeyRepo, userRepo, config)

	// Create API key with specific permissions
	apiKey := &models.APIKey{
		ID:          "test-key-id",
		UserID:      1,
		Name:        "Test Key",
		Permissions: []string{"config:read", "logs:read"},
		Active:      true,
	}

	// Test valid permissions
	err := manager.ValidateKeyPermissions(apiKey, []string{"config:read"})
	if err != nil {
		t.Errorf("Expected no error for valid permission, got: %v", err)
	}

	// Test multiple valid permissions
	err = manager.ValidateKeyPermissions(apiKey, []string{"config:read", "logs:read"})
	if err != nil {
		t.Errorf("Expected no error for valid permissions, got: %v", err)
	}

	// Test invalid permission
	err = manager.ValidateKeyPermissions(apiKey, []string{"config:write"})
	if err == nil {
		t.Error("Expected error for invalid permission")
	}

	// Test mixed valid/invalid permissions
	err = manager.ValidateKeyPermissions(apiKey, []string{"config:read", "config:write"})
	if err == nil {
		t.Error("Expected error for mixed permissions")
	}
}

func TestAPIKeyManager_UpdateAPIKey(t *testing.T) {
	// Setup
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	config := DefaultAPIKeyConfig()
	manager := NewAPIKeyManager(apiKeyRepo, userRepo, config)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}
	userRepo.Create(context.Background(), user)

	// Generate an API key
	req := &models.APIKeyCreateRequest{
		Name:        "Test Key",
		Description: "Original description",
	}

	apiKey, _, err := manager.GenerateAPIKey(context.Background(), user.ID, req)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Update the API key
	newName := "Updated Test Key"
	newDescription := "Updated description"
	updateReq := &models.APIKeyUpdateRequest{
		Name:        &newName,
		Description: &newDescription,
	}

	updatedKey, err := manager.UpdateAPIKey(context.Background(), apiKey.ID, user.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update API key: %v", err)
	}

	// Verify updates
	if updatedKey.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, updatedKey.Name)
	}
	if updatedKey.Description != newDescription {
		t.Errorf("Expected description %s, got %s", newDescription, updatedKey.Description)
	}
}

func TestAPIKeyManager_DeleteAPIKey(t *testing.T) {
	// Setup
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	config := DefaultAPIKeyConfig()
	manager := NewAPIKeyManager(apiKeyRepo, userRepo, config)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}
	userRepo.Create(context.Background(), user)

	// Generate an API key
	req := &models.APIKeyCreateRequest{
		Name: "Test Key",
	}

	apiKey, _, err := manager.GenerateAPIKey(context.Background(), user.ID, req)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Verify key exists
	_, err = apiKeyRepo.GetByID(context.Background(), apiKey.ID)
	if err != nil {
		t.Fatal("API key should exist before deletion")
	}

	// Delete the API key
	err = manager.DeleteAPIKey(context.Background(), apiKey.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete API key: %v", err)
	}

	// Verify key is deleted
	deletedKey, err := apiKeyRepo.GetByID(context.Background(), apiKey.ID)
	if deletedKey != nil {
		t.Error("API key should be deleted")
	}
}

func TestAPIKeyManager_RotateAPIKey(t *testing.T) {
	// Setup
	apiKeyRepo := NewMockAPIKeyRepository()
	userRepo := NewMockUserRepository()
	config := DefaultAPIKeyConfig()
	manager := NewAPIKeyManager(apiKeyRepo, userRepo, config)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Active:   true,
	}
	userRepo.Create(context.Background(), user)

	// Generate an API key
	req := &models.APIKeyCreateRequest{
		Name:        "Test Key",
		Permissions: []string{"config:read"},
	}

	originalKey, _, err := manager.GenerateAPIKey(context.Background(), user.ID, req)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Rotate the API key
	newKey, plainKey, err := manager.RotateAPIKey(context.Background(), originalKey.ID, user.ID)
	if err != nil {
		t.Fatalf("Failed to rotate API key: %v", err)
	}

	// Verify new key
	if newKey.ID == originalKey.ID {
		t.Error("New key should have different ID")
	}
	if newKey.UserID != originalKey.UserID {
		t.Error("New key should have same user ID")
	}
	if len(newKey.Permissions) != len(originalKey.Permissions) {
		t.Error("New key should have same permissions")
	}
	if plainKey == "" {
		t.Error("Plain key should not be empty")
	}

	// Verify original key is deactivated
	updatedOriginal, err := apiKeyRepo.GetByID(context.Background(), originalKey.ID)
	if err != nil {
		t.Fatal("Original key should still exist")
	}
	if updatedOriginal.Active {
		t.Error("Original key should be deactivated")
	}
}

func TestAPIKeyConfig_Defaults(t *testing.T) {
	config := DefaultAPIKeyConfig()

	if config.KeyLength != 32 {
		t.Errorf("Expected key length 32, got %d", config.KeyLength)
	}
	if config.DefaultExpiry != 365*24*time.Hour {
		t.Errorf("Expected default expiry 1 year, got %v", config.DefaultExpiry)
	}
	if config.MaxKeysPerUser != 10 {
		t.Errorf("Expected max keys per user 10, got %d", config.MaxKeysPerUser)
	}
	if !config.EnableUsageTracking {
		t.Error("Usage tracking should be enabled by default")
	}
}