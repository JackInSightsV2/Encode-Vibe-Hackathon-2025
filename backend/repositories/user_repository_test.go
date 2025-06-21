package repositories

import (
	"context"
	"database/sql"
	"qt1-middleware/models"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create users table
	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'user',
			api_key VARCHAR(255) UNIQUE,
			active BOOLEAN NOT NULL DEFAULT true,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}

	if user.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get the user by ID
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if retrievedUser == nil {
		t.Fatal("Expected user to be found")
	}

	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrievedUser.Username)
	}

	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get the user by username
	retrievedUser, err := repo.GetByUsername(ctx, user.Username)
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if retrievedUser == nil {
		t.Fatal("Expected user to be found")
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected ID %d, got %d", user.ID, retrievedUser.ID)
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get the user by email
	retrievedUser, err := repo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if retrievedUser == nil {
		t.Fatal("Expected user to be found")
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected ID %d, got %d", user.ID, retrievedUser.ID)
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	originalUpdatedAt := user.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(time.Millisecond * 10)

	// Update the user
	user.Email = "newemail@example.com"
	user.Role = "admin"

	err = repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify the update
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if retrievedUser.Email != "newemail@example.com" {
		t.Errorf("Expected email %s, got %s", "newemail@example.com", retrievedUser.Email)
	}

	if retrievedUser.Role != "admin" {
		t.Errorf("Expected role %s, got %s", "admin", retrievedUser.Role)
	}

	if !retrievedUser.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Delete the user
	err = repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify the user is deleted
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to check deleted user: %v", err)
	}

	if retrievedUser != nil {
		t.Error("Expected user to be deleted")
	}
}

func TestUserRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Initially should be 0
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users, got %d", count)
	}

	// Create a user
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err = repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Should now be 1
	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create a user first
	user := &models.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update last login
	err = repo.UpdateLastLogin(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to update last login: %v", err)
	}

	// Verify last login was updated
	retrievedUser, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.LastLoginAt == nil {
		t.Error("Expected LastLoginAt to be set")
	}
}