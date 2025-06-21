package auth

import (
	"database/sql"
	"qt1-middleware/database"
	"qt1-middleware/models"
	"qt1-middleware/repositories"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// Test helpers

func setupTestDB(t *testing.T) *database.Database {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create test schema
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

	return &database.Database{DB: db}
}

func setupTestAuthService(t *testing.T) *Service {
	db := setupTestDB(t)
	userRepo := repositories.NewUserRepository(db.DB)
	config := DefaultAuthConfig()
	config.JWTSecret = "test-secret-key"
	config.BcryptCost = 4 // Lower cost for faster tests
	
	return NewService(config, userRepo)
}

func createTestUser(t *testing.T, authService *Service) *models.User {
	req := &RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "TestPassword123!",
		Role:     "user",
	}
	
	user, err := authService.Register(req)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	
	return user
}

// JWT Manager Tests

func TestJWTManager_GenerateToken(t *testing.T) {
	config := DefaultAuthConfig()
	config.JWTSecret = "test-secret"
	jwtManager := NewJWTManager(config)
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	token, err := jwtManager.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	if token == "" {
		t.Error("Generated token is empty")
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	config := DefaultAuthConfig()
	config.JWTSecret = "test-secret"
	jwtManager := NewJWTManager(config)
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	// Generate token
	token, err := jwtManager.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	// Validate token
	claims, err := jwtManager.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	
	if claims.UserID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, claims.UserID)
	}
	
	if claims.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, claims.Username)
	}
	
	if claims.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, claims.Email)
	}
	
	if claims.Role != user.Role {
		t.Errorf("Expected role %s, got %s", user.Role, claims.Role)
	}
}

func TestJWTManager_ValidateToken_Invalid(t *testing.T) {
	config := DefaultAuthConfig()
	config.JWTSecret = "test-secret"
	jwtManager := NewJWTManager(config)
	
	// Test with invalid token
	_, err := jwtManager.ValidateToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
	
	// Test with empty token
	_, err = jwtManager.ValidateToken("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestJWTManager_RefreshToken(t *testing.T) {
	config := DefaultAuthConfig()
	config.JWTSecret = "test-secret"
	config.TokenExpiration = 30 * time.Minute // Longer expiration for testing valid refresh
	jwtManager := NewJWTManager(config)
	
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	// Generate token
	token, err := jwtManager.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	// Add a small delay to ensure timestamps are different
	time.Sleep(100 * time.Millisecond)
	
	// Refresh the valid token
	newToken, err := jwtManager.RefreshToken(token)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}
	
	if newToken == "" {
		t.Error("Refreshed token is empty")
	}
	
	if newToken == token {
		t.Error("Refreshed token should be different from original")
	}
}

// Password Manager Tests

func TestPasswordManager_HashPassword(t *testing.T) {
	config := DefaultAuthConfig()
	config.BcryptCost = 4 // Lower cost for faster tests
	passwordManager := NewPasswordManager(config)
	
	password := "TestPassword123!"
	hash, err := passwordManager.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	if hash == "" {
		t.Error("Password hash is empty")
	}
	
	if hash == password {
		t.Error("Hash should not equal original password")
	}
}

func TestPasswordManager_VerifyPassword(t *testing.T) {
	config := DefaultAuthConfig()
	config.BcryptCost = 4 // Lower cost for faster tests
	passwordManager := NewPasswordManager(config)
	
	password := "TestPassword123!"
	hash, err := passwordManager.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	
	// Test correct password
	err = passwordManager.VerifyPassword(hash, password)
	if err != nil {
		t.Errorf("Failed to verify correct password: %v", err)
	}
	
	// Test incorrect password
	err = passwordManager.VerifyPassword(hash, "wrongpassword")
	if err == nil {
		t.Error("Expected error for incorrect password")
	}
}

func TestPasswordManager_ValidatePassword(t *testing.T) {
	config := DefaultAuthConfig()
	passwordManager := NewPasswordManager(config)
	
	tests := []struct {
		password string
		valid    bool
		desc     string
	}{
		{"TestPassword123!", true, "valid strong password"},
		{"test123!", false, "missing uppercase"},
		{"TEST123!", false, "missing lowercase"},
		{"TestPassword!", false, "missing number"},
		{"TestPassword123", false, "missing special character"},
		{"Test1!", false, "too short"},
		{"password", false, "common password"},
		{"123456", false, "numeric sequence"},
	}
	
	for _, test := range tests {
		err := passwordManager.ValidatePassword(test.password)
		if test.valid && err != nil {
			t.Errorf("Expected password '%s' to be valid (%s), but got error: %v", test.password, test.desc, err)
		}
		if !test.valid && err == nil {
			t.Errorf("Expected password '%s' to be invalid (%s), but no error was returned", test.password, test.desc)
		}
	}
}

// Authentication Service Tests

func TestAuthService_Register(t *testing.T) {
	authService := setupTestAuthService(t)
	
	req := &RegisterRequest{
		Username: "newuser",
		Email:    "newuser@example.com",
		Password: "NewPassword123!",
		Role:     "user",
	}
	
	user, err := authService.Register(req)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	
	if user.Username != req.Username {
		t.Errorf("Expected username %s, got %s", req.Username, user.Username)
	}
	
	if user.Email != req.Email {
		t.Errorf("Expected email %s, got %s", req.Email, user.Email)
	}
	
	if user.Role != req.Role {
		t.Errorf("Expected role %s, got %s", req.Role, user.Role)
	}
	
	if !user.Active {
		t.Error("Expected user to be active")
	}
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create first user
	req1 := &RegisterRequest{
		Username: "testuser",
		Email:    "test1@example.com",
		Password: "TestPassword123!",
		Role:     "user",
	}
	
	_, err := authService.Register(req1)
	if err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}
	
	// Try to create user with same username
	req2 := &RegisterRequest{
		Username: "testuser", // Same username
		Email:    "test2@example.com",
		Password: "TestPassword123!",
		Role:     "user",
	}
	
	_, err = authService.Register(req2)
	if err == nil {
		t.Error("Expected error for duplicate username")
	}
	
	if authErr, ok := err.(*AuthError); ok {
		if authErr.Code != "USERNAME_EXISTS" {
			t.Errorf("Expected error code USERNAME_EXISTS, got %s", authErr.Code)
		}
	} else {
		t.Error("Expected AuthError for duplicate username")
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create first user
	req1 := &RegisterRequest{
		Username: "testuser1",
		Email:    "test@example.com",
		Password: "TestPassword123!",
		Role:     "user",
	}
	
	_, err := authService.Register(req1)
	if err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}
	
	// Try to create user with same email
	req2 := &RegisterRequest{
		Username: "testuser2",
		Email:    "test@example.com", // Same email
		Password: "TestPassword123!",
		Role:     "user",
	}
	
	_, err = authService.Register(req2)
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
	
	if authErr, ok := err.(*AuthError); ok {
		if authErr.Code != "EMAIL_EXISTS" {
			t.Errorf("Expected error code EMAIL_EXISTS, got %s", authErr.Code)
		}
	} else {
		t.Error("Expected AuthError for duplicate email")
	}
}

func TestAuthService_Login(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create test user
	user := createTestUser(t, authService)
	
	// Attempt login
	result, err := authService.Login(user.Username, "TestPassword123!")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	
	if result.User.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, result.User.ID)
	}
	
	if result.Token == "" {
		t.Error("Expected non-empty token")
	}
	
	if result.RefreshToken == "" {
		t.Error("Expected non-empty refresh token")
	}
	
	if result.ExpiresAt.IsZero() {
		t.Error("Expected non-zero expiration time")
	}
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create test user
	user := createTestUser(t, authService)
	
	// Attempt login with wrong password
	_, err := authService.Login(user.Username, "WrongPassword")
	if err == nil {
		t.Error("Expected error for invalid credentials")
	}
	
	if authErr, ok := err.(*AuthError); ok {
		if authErr.Code != "INVALID_CREDENTIALS" {
			t.Errorf("Expected error code INVALID_CREDENTIALS, got %s", authErr.Code)
		}
	} else {
		t.Error("Expected AuthError for invalid credentials")
	}
	
	// Attempt login with non-existent user
	_, err = authService.Login("nonexistent", "password")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create test user
	user := createTestUser(t, authService)
	
	// Change password
	err := authService.ChangePassword(user.ID, "TestPassword123!", "NewPassword456!")
	if err != nil {
		t.Fatalf("Failed to change password: %v", err)
	}
	
	// Try to login with new password
	_, err = authService.Login(user.Username, "NewPassword456!")
	if err != nil {
		t.Errorf("Failed to login with new password: %v", err)
	}
	
	// Try to login with old password (should fail)
	_, err = authService.Login(user.Username, "TestPassword123!")
	if err == nil {
		t.Error("Expected error when logging in with old password")
	}
}

func TestAuthService_ChangePassword_InvalidOldPassword(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create test user
	user := createTestUser(t, authService)
	
	// Try to change password with wrong old password
	err := authService.ChangePassword(user.ID, "WrongPassword", "NewPassword456!")
	if err == nil {
		t.Error("Expected error for invalid old password")
	}
	
	if authErr, ok := err.(*AuthError); ok {
		if authErr.Code != "INVALID_OLD_PASSWORD" {
			t.Errorf("Expected error code INVALID_OLD_PASSWORD, got %s", authErr.Code)
		}
	} else {
		t.Error("Expected AuthError for invalid old password")
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create test user
	user := createTestUser(t, authService)
	
	// Login to get refresh token
	result, err := authService.Login(user.Username, "TestPassword123!")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	
	// Add a small delay to ensure timestamps are different
	time.Sleep(100 * time.Millisecond)
	
	// Use refresh token to get new access token
	newToken, err := authService.RefreshToken(result.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}
	
	if newToken == "" {
		t.Error("Expected non-empty new token")
	}
	
	if newToken == result.Token {
		t.Errorf("New token should be different from original\nOriginal: %s\nNew: %s", result.Token, newToken)
	}
}

func TestAuthService_GetUserByToken(t *testing.T) {
	authService := setupTestAuthService(t)
	
	// Create test user
	user := createTestUser(t, authService)
	
	// Generate token
	token, err := authService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	// Get user by token
	retrievedUser, err := authService.GetUserByToken(token)
	if err != nil {
		t.Fatalf("Failed to get user by token: %v", err)
	}
	
	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, retrievedUser.ID)
	}
	
	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrievedUser.Username)
	}
}

// Benchmark tests

func BenchmarkPasswordManager_HashPassword(b *testing.B) {
	config := DefaultAuthConfig()
	passwordManager := NewPasswordManager(config)
	password := "TestPassword123!"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := passwordManager.HashPassword(password)
		if err != nil {
			b.Fatalf("Failed to hash password: %v", err)
		}
	}
}

func BenchmarkPasswordManager_VerifyPassword(b *testing.B) {
	config := DefaultAuthConfig()
	passwordManager := NewPasswordManager(config)
	password := "TestPassword123!"
	hash, _ := passwordManager.HashPassword(password)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := passwordManager.VerifyPassword(hash, password)
		if err != nil {
			b.Fatalf("Failed to verify password: %v", err)
		}
	}
}

func BenchmarkJWTManager_GenerateToken(b *testing.B) {
	config := DefaultAuthConfig()
	jwtManager := NewJWTManager(config)
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jwtManager.GenerateToken(user)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}
	}
}

func BenchmarkJWTManager_ValidateToken(b *testing.B) {
	config := DefaultAuthConfig()
	jwtManager := NewJWTManager(config)
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	token, _ := jwtManager.GenerateToken(user)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jwtManager.ValidateToken(token)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}