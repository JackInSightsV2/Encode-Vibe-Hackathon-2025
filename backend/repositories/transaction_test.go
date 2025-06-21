package repositories

import (
	"context"
	"database/sql"
	"qt1-middleware/database"
	"qt1-middleware/models"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTransactionTestDB(t *testing.T) *database.Database {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create a basic users table for testing
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'user',
			api_key TEXT,
			active BOOLEAN NOT NULL DEFAULT true,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_login_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create system_config table for testing
	_, err = db.Exec(`
		CREATE TABLE system_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			value TEXT NOT NULL,
			value_type TEXT NOT NULL DEFAULT 'string',
			category TEXT NOT NULL DEFAULT 'general',
			description TEXT,
			is_secret BOOLEAN NOT NULL DEFAULT false,
			read_only BOOLEAN NOT NULL DEFAULT false,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			updated_by INTEGER,
			version INTEGER NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create system_config table: %v", err)
	}

	return &database.Database{DB: db}
}

func TestTransactionBasicOperations(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	// Create repository manager
	repoManager := NewRepositoryManager(testDB)
	defer repoManager.Close()

	ctx := context.Background()

	// Test transaction creation
	tx, err := repoManager.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Test accessing repositories through transaction
	userRepo := tx.Users()
	if userRepo == nil {
		t.Fatal("User repository from transaction is nil")
	}

	configRepo := tx.SystemConfigs()
	if configRepo == nil {
		t.Fatal("SystemConfig repository from transaction is nil")
	}

	// Test rollback
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}
}

func TestTransactionUserOperations(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	repoManager := NewRepositoryManager(testDB)
	defer repoManager.Close()

	ctx := context.Background()

	// Test transaction with user operations
	err := repoManager.WithTransaction(ctx, func(tx Transaction) error {
		userRepo := tx.Users()

		// Create a user within transaction
		user := &models.User{
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashed_password",
			Role:         "user",
			Active:       true,
		}

		err := userRepo.Create(ctx, user)
		if err != nil {
			return err
		}

		// Verify user was created and has an ID
		if user.ID == 0 {
			t.Error("User ID should be set after creation")
		}

		// Fetch the user back
		fetchedUser, err := userRepo.GetByID(ctx, user.ID)
		if err != nil {
			return err
		}

		if fetchedUser == nil {
			t.Error("Fetched user should not be nil")
			return nil
		}

		if fetchedUser.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got %s", fetchedUser.Username)
		}

		if fetchedUser.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %s", fetchedUser.Email)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify user exists outside transaction
	normalUserRepo := repoManager.Users()
	users, err := normalUserRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestTransactionRollback(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	repoManager := NewRepositoryManager(testDB)
	defer repoManager.Close()

	ctx := context.Background()

	// Test transaction rollback
	err := repoManager.WithTransaction(ctx, func(tx Transaction) error {
		userRepo := tx.Users()

		// Create a user within transaction
		user := &models.User{
			Username:     "rollback_user",
			Email:        "rollback@example.com",
			PasswordHash: "hashed_password",
			Role:         "user",
			Active:       true,
		}

		err := userRepo.Create(ctx, user)
		if err != nil {
			return err
		}

		// Force a rollback by returning an error
		return sql.ErrTxDone
	})

	if err == nil {
		t.Fatal("Expected transaction to fail and rollback")
	}

	// Verify user was not created due to rollback
	normalUserRepo := repoManager.Users()
	users, err := normalUserRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 0 {
		t.Errorf("Expected 0 users after rollback, got %d", len(users))
	}
}

func TestTransactionSystemConfigOperations(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	repoManager := NewRepositoryManager(testDB)
	defer repoManager.Close()

	ctx := context.Background()

	// Test transaction with system config operations
	err := repoManager.WithTransaction(ctx, func(tx Transaction) error {
		configRepo := tx.SystemConfigs()

		// Create a config within transaction
		config := &models.SystemConfig{
			Key:         "test.setting",
			Value:       "test_value",
			ValueType:   "string",
			Category:    "testing",
			Description: transactionStringPtr("Test configuration setting"),
			IsSecret:    false,
			ReadOnly:    false,
			UpdatedBy:   transactionIntPtr(1),
		}

		err := configRepo.Create(ctx, config)
		if err != nil {
			return err
		}

		// Verify config was created and has an ID
		if config.ID == 0 {
			t.Error("Config ID should be set after creation")
		}

		// Fetch the config back
		fetchedConfig, err := configRepo.GetByKey(ctx, "test.setting")
		if err != nil {
			return err
		}

		if fetchedConfig == nil {
			t.Error("Fetched config should not be nil")
			return nil
		}

		if fetchedConfig.Value != "test_value" {
			t.Errorf("Expected value 'test_value', got %s", fetchedConfig.Value)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}
}

func TestTransactionIsolation(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	repoManager := NewRepositoryManager(testDB)
	defer repoManager.Close()

	ctx := context.Background()

	// Create initial user outside transaction
	normalUserRepo := repoManager.Users()
	initialUser := &models.User{
		Username:     "initial_user",
		Email:        "initial@example.com",
		PasswordHash: "hashed_password",
		Role:         "user",
		Active:       true,
	}
	err := normalUserRepo.Create(ctx, initialUser)
	if err != nil {
		t.Fatalf("Failed to create initial user: %v", err)
	}

	// Start a transaction but don't commit
	tx, err := repoManager.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Create user within transaction
	userRepo := tx.Users()
	txUser := &models.User{
		Username:     "tx_user",
		Email:        "tx@example.com",
		PasswordHash: "hashed_password",
		Role:         "user",
		Active:       true,
	}
	err = userRepo.Create(ctx, txUser)
	if err != nil {
		t.Fatalf("Failed to create user in transaction: %v", err)
	}

	// Outside the transaction, we should only see the initial user
	users, err := normalUserRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user outside transaction, got %d", len(users))
	}

	// Inside the transaction, we should see both users
	txUsers, err := userRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list users in transaction: %v", err)
	}

	if len(txUsers) != 2 {
		t.Errorf("Expected 2 users inside transaction, got %d", len(txUsers))
	}

	// Rollback transaction
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// After rollback, we should still only see the initial user
	finalUsers, err := normalUserRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list users after rollback: %v", err)
	}

	if len(finalUsers) != 1 {
		t.Errorf("Expected 1 user after rollback, got %d", len(finalUsers))
	}
}

// Helper functions for transaction tests
func transactionStringPtr(s string) *string {
	return &s
}

func transactionIntPtr(i int) *int {
	return &i
}