package repositories

import (
	"context"
	"database/sql"
	"qt1-middleware/models"
	"testing"

	_ "modernc.org/sqlite"
)

func setupConfigTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create system_config table
	schema := `
		CREATE TABLE system_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key VARCHAR(100) UNIQUE NOT NULL,
			value TEXT NOT NULL,
			value_type VARCHAR(20) NOT NULL DEFAULT 'string',
			category VARCHAR(50) NOT NULL,
			description TEXT,
			is_secret BOOLEAN NOT NULL DEFAULT false,
			read_only BOOLEAN NOT NULL DEFAULT false,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by INTEGER,
			version INTEGER NOT NULL DEFAULT 1
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return db
}

func TestSystemConfigRepository_Create(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	config := &models.SystemConfig{
		Key:         "test_key",
		Value:       "test_value",
		ValueType:   "string",
		Category:    "test",
		Description: stringPtr("Test configuration"),
		IsSecret:    false,
		ReadOnly:    false,
		UpdatedBy:   intPtr(1),
	}

	err := repo.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	if config.ID == 0 {
		t.Error("Expected config ID to be set after creation")
	}

	if config.Version != 1 {
		t.Errorf("Expected version 1, got %d", config.Version)
	}

	if config.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestSystemConfigRepository_GetByKey(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create a config first
	config := &models.SystemConfig{
		Key:       "test_key",
		Value:     "test_value",
		ValueType: "string",
		Category:  "test",
		IsSecret:  false,
		ReadOnly:  false,
	}

	err := repo.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Get the config by key
	retrievedConfig, err := repo.GetByKey(ctx, "test_key")
	if err != nil {
		t.Fatalf("Failed to get config by key: %v", err)
	}

	if retrievedConfig == nil {
		t.Fatal("Expected config to be found")
	}

	if retrievedConfig.Key != config.Key {
		t.Errorf("Expected key %s, got %s", config.Key, retrievedConfig.Key)
	}

	if retrievedConfig.Value != config.Value {
		t.Errorf("Expected value %s, got %s", config.Value, retrievedConfig.Value)
	}
}

func TestSystemConfigRepository_Update(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create a config first
	config := &models.SystemConfig{
		Key:       "test_key",
		Value:     "test_value",
		ValueType: "string",
		Category:  "test",
		IsSecret:  false,
		ReadOnly:  false,
	}

	err := repo.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	originalVersion := config.Version

	// Update the config
	config.Value = "updated_value"
	config.Description = stringPtr("Updated description")
	updatedBy := 2
	config.UpdatedBy = &updatedBy

	err = repo.Update(ctx, config)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Verify the update
	retrievedConfig, err := repo.GetByKey(ctx, config.Key)
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}

	if retrievedConfig.Value != "updated_value" {
		t.Errorf("Expected value %s, got %s", "updated_value", retrievedConfig.Value)
	}

	if retrievedConfig.Version <= originalVersion {
		t.Errorf("Expected version to be incremented, got %d", retrievedConfig.Version)
	}

	if retrievedConfig.UpdatedBy == nil || *retrievedConfig.UpdatedBy != 2 {
		t.Error("Expected UpdatedBy to be set to 2")
	}
}

func TestSystemConfigRepository_GetByCategory(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create configs in different categories
	configs := []*models.SystemConfig{
		{Key: "config1", Value: "value1", ValueType: "string", Category: "category1"},
		{Key: "config2", Value: "value2", ValueType: "string", Category: "category1"},
		{Key: "config3", Value: "value3", ValueType: "string", Category: "category2"},
	}

	for _, config := range configs {
		if err := repo.Create(ctx, config); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
	}

	// Get configs by category
	category1Configs, err := repo.GetByCategory(ctx, "category1")
	if err != nil {
		t.Fatalf("Failed to get configs by category: %v", err)
	}

	if len(category1Configs) != 2 {
		t.Errorf("Expected 2 configs in category1, got %d", len(category1Configs))
	}

	for _, config := range category1Configs {
		if config.Category != "category1" {
			t.Errorf("Expected category category1, got %s", config.Category)
		}
	}
}

func TestSystemConfigRepository_GetSecrets(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create secret and non-secret configs
	configs := []*models.SystemConfig{
		{Key: "secret1", Value: "secret_value1", ValueType: "string", Category: "test", IsSecret: true},
		{Key: "secret2", Value: "secret_value2", ValueType: "string", Category: "test", IsSecret: true},
		{Key: "public1", Value: "public_value1", ValueType: "string", Category: "test", IsSecret: false},
	}

	for _, config := range configs {
		if err := repo.Create(ctx, config); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
	}

	// Get secret configs
	secrets, err := repo.GetSecrets(ctx)
	if err != nil {
		t.Fatalf("Failed to get secret configs: %v", err)
	}

	if len(secrets) != 2 {
		t.Errorf("Expected 2 secret configs, got %d", len(secrets))
	}

	for _, config := range secrets {
		if !config.IsSecret {
			t.Errorf("Expected secret config, got non-secret: %s", config.Key)
		}
	}
}

func TestSystemConfigRepository_SetValue(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Set a new value (should create)
	err := repo.SetValue(ctx, "new_key", "new_value", 1)
	if err != nil {
		t.Fatalf("Failed to set new value: %v", err)
	}

	// Verify it was created
	config, err := repo.GetByKey(ctx, "new_key")
	if err != nil {
		t.Fatalf("Failed to get created config: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be created")
	}

	if config.Value != "new_value" {
		t.Errorf("Expected value new_value, got %s", config.Value)
	}

	// Update existing value
	err = repo.SetValue(ctx, "new_key", "updated_value", 2)
	if err != nil {
		t.Fatalf("Failed to update value: %v", err)
	}

	// Verify it was updated
	config, err = repo.GetByKey(ctx, "new_key")
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}

	if config.Value != "updated_value" {
		t.Errorf("Expected value updated_value, got %s", config.Value)
	}
}

func TestSystemConfigRepository_BulkUpdate(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create multiple configs
	configs := []*models.SystemConfig{
		{Key: "config1", Value: "value1", ValueType: "string", Category: "test"},
		{Key: "config2", Value: "value2", ValueType: "string", Category: "test"},
		{Key: "config3", Value: "value3", ValueType: "string", Category: "test"},
	}

	for _, config := range configs {
		if err := repo.Create(ctx, config); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
	}

	// Update all configs
	for _, config := range configs {
		config.Value = "bulk_updated_" + config.Value
	}

	err := repo.BulkUpdate(ctx, configs, 1)
	if err != nil {
		t.Fatalf("Failed to bulk update: %v", err)
	}

	// Verify all were updated
	for _, config := range configs {
		retrievedConfig, err := repo.GetByKey(ctx, config.Key)
		if err != nil {
			t.Fatalf("Failed to get config after bulk update: %v", err)
		}

		if !contains(retrievedConfig.Value, "bulk_updated_") {
			t.Errorf("Expected value to contain 'bulk_updated_', got %s", retrievedConfig.Value)
		}
	}
}

func TestSystemConfigRepository_Delete(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create a config
	config := &models.SystemConfig{
		Key:       "deletable_key",
		Value:     "deletable_value",
		ValueType: "string",
		Category:  "test",
		ReadOnly:  false,
	}

	err := repo.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Delete the config
	err = repo.Delete(ctx, config.ID)
	if err != nil {
		t.Fatalf("Failed to delete config: %v", err)
	}

	// Verify it was deleted
	retrievedConfig, err := repo.GetByKey(ctx, config.Key)
	if err != nil {
		t.Fatalf("Failed to check deleted config: %v", err)
	}

	if retrievedConfig != nil {
		t.Error("Expected config to be deleted")
	}
}

func TestSystemConfigRepository_ReadOnlyProtection(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create a read-only config
	config := &models.SystemConfig{
		Key:       "readonly_key",
		Value:     "readonly_value",
		ValueType: "string",
		Category:  "test",
		ReadOnly:  true,
	}

	err := repo.Create(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Try to delete read-only config (should fail)
	err = repo.Delete(ctx, config.ID)
	if err == nil {
		t.Error("Expected delete to fail for read-only config")
	}
}

func TestSystemConfigRepository_GetCategories(t *testing.T) {
	db := setupConfigTestDB(t)
	defer db.Close()

	repo := NewSystemConfigRepository(db)
	ctx := context.Background()

	// Create configs in different categories
	configs := []*models.SystemConfig{
		{Key: "config1", Value: "value1", ValueType: "string", Category: "auth"},
		{Key: "config2", Value: "value2", ValueType: "string", Category: "database"},
		{Key: "config3", Value: "value3", ValueType: "string", Category: "auth"},
		{Key: "config4", Value: "value4", ValueType: "string", Category: "logging"},
	}

	for _, config := range configs {
		if err := repo.Create(ctx, config); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}
	}

	// Get categories
	categories, err := repo.GetCategories(ctx)
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	expectedCategories := []string{"auth", "database", "logging"}
	if len(categories) != len(expectedCategories) {
		t.Errorf("Expected %d categories, got %d", len(expectedCategories), len(categories))
	}

	for _, expected := range expectedCategories {
		found := false
		for _, category := range categories {
			if category == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected category %s not found", expected)
		}
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}