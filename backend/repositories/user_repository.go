package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"qt1-middleware/models"
	"strings"
	"time"
)

// userRepository implements UserRepository interface
type userRepository struct {
	db *sql.DB
}


// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, api_key, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	
	result, err := r.db.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role,
		user.APIKey, user.Active, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get created user ID: %w", err)
	}
	
	user.ID = int(id)
	return nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE id = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var apiKey sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &apiKey, &user.Active, &user.CreatedAt,
		&user.UpdatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	
	if apiKey.Valid {
		user.APIKey = &apiKey.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	
	return user, nil
}

// GetByUsername retrieves a user by username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE username = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var apiKey sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &apiKey, &user.Active, &user.CreatedAt,
		&user.UpdatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	
	if apiKey.Valid {
		user.APIKey = &apiKey.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE email = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var apiKey sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &apiKey, &user.Active, &user.CreatedAt,
		&user.UpdatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	
	if apiKey.Valid {
		user.APIKey = &apiKey.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	
	return user, nil
}

// GetByAPIKey retrieves a user by API key
func (r *userRepository) GetByAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE api_key = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var userAPIKey sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, apiKey).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &userAPIKey, &user.Active, &user.CreatedAt,
		&user.UpdatedAt, &lastLoginAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by API key: %w", err)
	}
	
	if userAPIKey.Valid {
		user.APIKey = &userAPIKey.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	
	return user, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users 
		SET username = ?, email = ?, password_hash = ?, role = ?, 
		    api_key = ?, active = ?, updated_at = ?
		WHERE id = ?
	`
	
	user.UpdatedAt = time.Now()
	
	_, err := r.db.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role,
		user.APIKey, user.Active, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	return nil
}

// Delete deletes a user by ID
func (r *userRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = ?`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// List retrieves a list of users with pagination
func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users 
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()
	
	return r.scanUsers(rows)
}

// Search searches for users by username or email
func (r *userRepository) Search(ctx context.Context, query string, limit, offset int) ([]*models.User, error) {
	searchQuery := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users 
		WHERE username LIKE ? OR email LIKE ?
		ORDER BY username
		LIMIT ? OFFSET ?
	`
	
	searchTerm := "%" + strings.ToLower(query) + "%"
	
	rows, err := r.db.QueryContext(ctx, searchQuery, searchTerm, searchTerm, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()
	
	return r.scanUsers(rows)
}

// GetActiveUsers retrieves a list of active users
func (r *userRepository) GetActiveUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users 
		WHERE active = true
		ORDER BY last_login_at DESC NULLS LAST
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer rows.Close()
	
	return r.scanUsers(rows)
}

// UpdateLastLogin updates the last login time for a user
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID int) error {
	query := `UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`
	
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	
	return nil
}

// SetActive sets the active status of a user
func (r *userRepository) SetActive(ctx context.Context, userID int, active bool) error {
	query := `UPDATE users SET active = ?, updated_at = ? WHERE id = ?`
	
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, active, now, userID)
	if err != nil {
		return fmt.Errorf("failed to set user active status: %w", err)
	}
	
	return nil
}

// ChangePassword updates the password hash for a user
func (r *userRepository) ChangePassword(ctx context.Context, userID int, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, passwordHash, now, userID)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	
	return nil
}

// Count returns the total number of users
func (r *userRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	
	return count, nil
}

// CountByRole returns the number of users with a specific role
func (r *userRepository) CountByRole(ctx context.Context, role string) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE role = ?`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}
	
	return count, nil
}

// CountActive returns the number of active users
func (r *userRepository) CountActive(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE active = true`
	
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}
	
	return count, nil
}

// scanUsers is a helper function to scan multiple users from rows
func (r *userRepository) scanUsers(rows *sql.Rows) ([]*models.User, error) {
	var users []*models.User
	
	for rows.Next() {
		user := &models.User{}
		var lastLoginAt sql.NullTime
		var apiKey sql.NullString
		
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Role, &apiKey, &user.Active, &user.CreatedAt,
			&user.UpdatedAt, &lastLoginAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		
		if apiKey.Valid {
			user.APIKey = &apiKey.String
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}
		
		users = append(users, user)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}
	
	return users, nil
}