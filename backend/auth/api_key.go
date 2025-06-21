package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"qt1-middleware/models"
	"qt1-middleware/repositories"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// APIKeyManager manages API key generation, validation, and lifecycle
type APIKeyManager struct {
	apiKeyRepo repositories.APIKeyRepository
	userRepo   repositories.UserRepository
	config     *APIKeyConfig
}

// APIKeyConfig holds configuration for API key management
type APIKeyConfig struct {
	KeyLength       int           `yaml:"key_length" json:"key_length"`
	DefaultExpiry   time.Duration `yaml:"default_expiry" json:"default_expiry"`
	MaxExpiry       time.Duration `yaml:"max_expiry" json:"max_expiry"`
	BcryptCost      int           `yaml:"bcrypt_cost" json:"bcrypt_cost"`
	MaxKeysPerUser  int           `yaml:"max_keys_per_user" json:"max_keys_per_user"`
	PurgeInactiveAfter time.Duration `yaml:"purge_inactive_after" json:"purge_inactive_after"`
	EnableUsageTracking bool        `yaml:"enable_usage_tracking" json:"enable_usage_tracking"`
}

// DefaultAPIKeyConfig returns default API key configuration
func DefaultAPIKeyConfig() *APIKeyConfig {
	return &APIKeyConfig{
		KeyLength:          32, // 32 bytes = 256 bits
		DefaultExpiry:      365 * 24 * time.Hour, // 1 year
		MaxExpiry:          2 * 365 * 24 * time.Hour, // 2 years
		BcryptCost:         12,
		MaxKeysPerUser:     10,
		PurgeInactiveAfter: 2 * 365 * 24 * time.Hour, // 2 years
		EnableUsageTracking: true,
	}
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(apiKeyRepo repositories.APIKeyRepository, userRepo repositories.UserRepository, config *APIKeyConfig) *APIKeyManager {
	if config == nil {
		config = DefaultAPIKeyConfig()
	}

	return &APIKeyManager{
		apiKeyRepo: apiKeyRepo,
		userRepo:   userRepo,
		config:     config,
	}
}

// GenerateAPIKey generates a new API key for a user
func (akm *APIKeyManager) GenerateAPIKey(ctx context.Context, userID int, req *models.APIKeyCreateRequest) (*models.APIKey, string, error) {
	// Validate user exists
	user, err := akm.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, "", fmt.Errorf("user not found")
	}

	// Check user's existing API key count
	existingKeys, err := akm.apiKeyRepo.GetByUserID(ctx, userID, true) // active only
	if err != nil {
		return nil, "", fmt.Errorf("failed to check existing keys: %w", err)
	}

	if len(existingKeys) >= akm.config.MaxKeysPerUser {
		return nil, "", fmt.Errorf("maximum number of API keys (%d) exceeded", akm.config.MaxKeysPerUser)
	}

	// Generate secure API key
	plainKey, err := akm.generateSecureKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Hash the key for storage
	keyHash, err := akm.hashKey(plainKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash key: %w", err)
	}

	// Generate unique ID
	keyID, err := akm.generateKeyID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key ID: %w", err)
	}

	// Set expiry
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		// Validate requested expiry
		if time.Until(*req.ExpiresAt) > akm.config.MaxExpiry {
			return nil, "", fmt.Errorf("expiry date cannot exceed maximum allowed period")
		}
		expiresAt = req.ExpiresAt
	} else {
		// Use default expiry
		defaultExpiry := time.Now().Add(akm.config.DefaultExpiry)
		expiresAt = &defaultExpiry
	}

	// Create API key record
	apiKey := &models.APIKey{
		ID:          keyID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		Active:      true,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		UsageCount:  0,
		Permissions: req.Permissions,
	}

	// Store in database
	if err := akm.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("failed to store API key: %w", err)
	}

	return apiKey, plainKey, nil
}

// ValidateAPIKey validates an API key and returns the associated key and user
func (akm *APIKeyManager) ValidateAPIKey(ctx context.Context, keyString string) (*models.APIKey, *models.User, error) {
	if keyString == "" {
		return nil, nil, fmt.Errorf("empty API key")
	}

	// Extract key ID from the key string (first part before the actual key)
	parts := strings.Split(keyString, ".")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid API key format")
	}

	keyID := parts[0]
	actualKey := parts[1]

	// Get API key by ID
	apiKey, err := akm.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return nil, nil, fmt.Errorf("API key not found")
	}

	// Check if key is valid (active and not expired)
	if !apiKey.IsValid() {
		return nil, nil, fmt.Errorf("API key is inactive or expired")
	}

	// Verify key hash
	if err := akm.verifyKey(actualKey, apiKey.KeyHash); err != nil {
		return nil, nil, fmt.Errorf("invalid API key")
	}

	// Get associated user
	user, err := akm.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || !user.Active {
		return nil, nil, fmt.Errorf("user not found or inactive")
	}

	// Update usage statistics
	if akm.config.EnableUsageTracking {
		go akm.updateUsageStats(context.Background(), apiKey.ID)
	}

	return apiKey, user, nil
}

// UpdateAPIKey updates an existing API key
func (akm *APIKeyManager) UpdateAPIKey(ctx context.Context, keyID string, userID int, req *models.APIKeyUpdateRequest) (*models.APIKey, error) {
	// Get existing API key
	apiKey, err := akm.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return nil, fmt.Errorf("API key not found")
	}

	// Check ownership
	if apiKey.UserID != userID {
		return nil, fmt.Errorf("API key not owned by user")
	}

	// Update fields
	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Description != nil {
		apiKey.Description = *req.Description
	}
	if req.Active != nil {
		apiKey.Active = *req.Active
	}
	if req.ExpiresAt != nil {
		// Validate new expiry
		if time.Until(*req.ExpiresAt) > akm.config.MaxExpiry {
			return nil, fmt.Errorf("expiry date cannot exceed maximum allowed period")
		}
		apiKey.ExpiresAt = req.ExpiresAt
	}

	// Save changes
	if err := akm.apiKeyRepo.Update(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	return apiKey, nil
}

// DeleteAPIKey deletes an API key
func (akm *APIKeyManager) DeleteAPIKey(ctx context.Context, keyID string, userID int) error {
	// Get existing API key to verify ownership
	apiKey, err := akm.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return fmt.Errorf("API key not found")
	}

	// Check ownership
	if apiKey.UserID != userID {
		return fmt.Errorf("API key not owned by user")
	}

	// Delete the key
	if err := akm.apiKeyRepo.Delete(ctx, keyID); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// GetUserAPIKeys returns all API keys for a user
func (akm *APIKeyManager) GetUserAPIKeys(ctx context.Context, userID int, activeOnly bool) ([]*models.APIKey, error) {
	return akm.apiKeyRepo.GetByUserID(ctx, userID, activeOnly)
}

// GetAPIKeyStats returns statistics about API keys
func (akm *APIKeyManager) GetAPIKeyStats(ctx context.Context) (*models.APIKeyStats, error) {
	return akm.apiKeyRepo.GetStats(ctx)
}

// CleanupExpiredKeys removes expired and inactive keys
func (akm *APIKeyManager) CleanupExpiredKeys(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-akm.config.PurgeInactiveAfter)
	return akm.apiKeyRepo.DeleteExpired(ctx, cutoff)
}

// Private helper methods

func (akm *APIKeyManager) generateSecureKey() (string, error) {
	bytes := make([]byte, akm.config.KeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (akm *APIKeyManager) generateKeyID() (string, error) {
	bytes := make([]byte, 8) // 8 bytes for key ID
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate key ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (akm *APIKeyManager) hashKey(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), akm.config.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash key: %w", err)
	}
	return string(hash), nil
}

func (akm *APIKeyManager) verifyKey(key, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
}

func (akm *APIKeyManager) updateUsageStats(ctx context.Context, keyID string) {
	// Update last used timestamp and usage count
	now := time.Now()
	if err := akm.apiKeyRepo.UpdateLastUsed(ctx, keyID, now); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to update API key usage stats: %v\n", err)
	}
}

// FormatAPIKey formats a key ID and key into the final API key string
func (akm *APIKeyManager) FormatAPIKey(keyID, key string) string {
	return fmt.Sprintf("%s.%s", keyID, key)
}

// ExtractKeyFromHeader extracts API key from various header formats
func (akm *APIKeyManager) ExtractKeyFromHeader(authHeader string) string {
	// Support multiple formats:
	// - Bearer <key>
	// - ApiKey <key>
	// - <key> (direct)
	
	if authHeader == "" {
		return ""
	}
	
	// Remove common prefixes
	prefixes := []string{"Bearer ", "ApiKey ", "API-Key "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(authHeader, prefix) {
			return strings.TrimSpace(authHeader[len(prefix):])
		}
	}
	
	// Return as-is if no prefix found
	return strings.TrimSpace(authHeader)
}

// ValidateKeyPermissions checks if an API key has specific permissions
func (akm *APIKeyManager) ValidateKeyPermissions(apiKey *models.APIKey, requiredPermissions []string) error {
	if len(requiredPermissions) == 0 {
		return nil // No specific permissions required
	}

	if len(apiKey.Permissions) == 0 {
		return fmt.Errorf("API key has no permissions assigned")
	}

	// Check if key has all required permissions
	keyPermissions := make(map[string]bool)
	for _, perm := range apiKey.Permissions {
		keyPermissions[perm] = true
	}

	var missingPermissions []string
	for _, required := range requiredPermissions {
		if !keyPermissions[required] {
			missingPermissions = append(missingPermissions, required)
		}
	}

	if len(missingPermissions) > 0 {
		return fmt.Errorf("API key missing required permissions: %v", missingPermissions)
	}

	return nil
}

// RotateAPIKey creates a new API key to replace an existing one
func (akm *APIKeyManager) RotateAPIKey(ctx context.Context, keyID string, userID int) (*models.APIKey, string, error) {
	// Get existing key
	oldKey, err := akm.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get API key: %w", err)
	}
	if oldKey == nil {
		return nil, "", fmt.Errorf("API key not found")
	}

	// Check ownership
	if oldKey.UserID != userID {
		return nil, "", fmt.Errorf("API key not owned by user")
	}

	// Create new key with same properties
	req := &models.APIKeyCreateRequest{
		Name:        oldKey.Name + " (rotated)",
		Description: oldKey.Description,
		ExpiresAt:   oldKey.ExpiresAt,
		Permissions: oldKey.Permissions,
	}

	newKey, plainKey, err := akm.GenerateAPIKey(ctx, userID, req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate new key: %w", err)
	}

	// Deactivate old key
	oldKey.Active = false
	if err := akm.apiKeyRepo.Update(ctx, oldKey); err != nil {
		// Log error but don't fail - new key was created successfully
		fmt.Printf("Warning: failed to deactivate old key %s: %v\n", keyID, err)
	}

	return newKey, plainKey, nil
}

// GetAPIKeyUsage returns usage statistics for an API key
func (akm *APIKeyManager) GetAPIKeyUsage(ctx context.Context, keyID string, userID int, days int) ([]*models.APIKeyUsage, error) {
	// Verify ownership
	apiKey, err := akm.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	if apiKey == nil {
		return nil, fmt.Errorf("API key not found")
	}
	if apiKey.UserID != userID {
		return nil, fmt.Errorf("API key not owned by user")
	}

	// Get usage statistics
	startDate := time.Now().AddDate(0, 0, -days)
	return akm.apiKeyRepo.GetUsage(ctx, keyID, startDate)
}