package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"qt1-middleware/models"
	"strings"
	"time"
)

// DBExecutor is an interface that both sql.DB and sql.Tx implement
// This allows repositories to work with either regular database connections or transactions
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// Ensure that sql.DB and sql.Tx implement DBExecutor
var (
	_ DBExecutor = (*sql.DB)(nil)
	_ DBExecutor = (*sql.Tx)(nil)
)

// TransactionCapable indicates a repository can work within transactions
type TransactionCapable interface {
	WithExecutor(executor DBExecutor) interface{}
}

// repositoryFactory creates repositories that can work with transactions
type repositoryFactory struct {
	executor DBExecutor
}

// NewRepositoryFactory creates a factory for creating transaction-aware repositories
func NewRepositoryFactory(executor DBExecutor) *repositoryFactory {
	return &repositoryFactory{executor: executor}
}

// Users creates a user repository using the configured executor
func (rf *repositoryFactory) Users() UserRepository {
	return &transactionAwareUserRepository{executor: rf.executor}
}

// SystemConfigs creates a system config repository using the configured executor
func (rf *repositoryFactory) SystemConfigs() SystemConfigRepository {
	return &transactionAwareSystemConfigRepository{executor: rf.executor}
}

// Metrics creates a metrics repository using the configured executor
func (rf *repositoryFactory) Metrics() MetricsRepository {
	return &transactionAwareMetricsRepository{executor: rf.executor}
}

// Sessions creates a session repository using the configured executor
func (rf *repositoryFactory) Sessions() SessionRepository {
	return &transactionAwareSessionRepository{executor: rf.executor}
}

// Requests creates a request repository using the configured executor
func (rf *repositoryFactory) Requests() RequestRepository {
	return &transactionAwareRequestRepository{executor: rf.executor}
}

// ModerationLogs creates a moderation log repository using the configured executor
func (rf *repositoryFactory) ModerationLogs() ModerationLogRepository {
	return &transactionAwareModerationLogRepository{executor: rf.executor}
}

// KillSwitch creates a kill switch repository using the configured executor
func (rf *repositoryFactory) KillSwitch() KillSwitchRepository {
	return &transactionAwareKillSwitchRepository{executor: rf.executor}
}

// Transaction-aware repository implementations that use DBExecutor

// For now, implement transaction-aware repositories by wrapping existing ones
// and converting the sql.DB calls to use the transaction

// transactionAwareUserRepository delegates to the regular user repository but uses a transaction
type transactionAwareUserRepository struct {
	executor DBExecutor
	impl     *userRepository
}

func (r *transactionAwareUserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, api_key, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	
	result, err := r.executor.ExecContext(ctx, query,
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

func (r *transactionAwareUserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE id = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var apiKey sql.NullString
	
	err := r.executor.QueryRowContext(ctx, query, id).Scan(
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

func (r *transactionAwareUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE username = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var apiKey sql.NullString
	
	err := r.executor.QueryRowContext(ctx, query, username).Scan(
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

func (r *transactionAwareUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE email = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var apiKey sql.NullString
	
	err := r.executor.QueryRowContext(ctx, query, email).Scan(
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

func (r *transactionAwareUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users WHERE api_key = ?
	`
	
	user := &models.User{}
	var lastLoginAt sql.NullTime
	var userAPIKey sql.NullString
	
	err := r.executor.QueryRowContext(ctx, query, apiKey).Scan(
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

func (r *transactionAwareUserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users 
		SET username = ?, email = ?, password_hash = ?, role = ?, 
		    api_key = ?, active = ?, updated_at = ?
		WHERE id = ?
	`
	
	user.UpdatedAt = time.Now()
	
	_, err := r.executor.ExecContext(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role,
		user.APIKey, user.Active, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	return nil
}

func (r *transactionAwareUserRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = ?`
	
	result, err := r.executor.ExecContext(ctx, query, id)
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

func (r *transactionAwareUserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users 
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.executor.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()
	
	return r.scanUsers(rows)
}

func (r *transactionAwareUserRepository) Search(ctx context.Context, query string, limit, offset int) ([]*models.User, error) {
	searchQuery := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users 
		WHERE username LIKE ? OR email LIKE ?
		ORDER BY username
		LIMIT ? OFFSET ?
	`
	
	searchTerm := "%" + strings.ToLower(query) + "%"
	
	rows, err := r.executor.QueryContext(ctx, searchQuery, searchTerm, searchTerm, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()
	
	return r.scanUsers(rows)
}

func (r *transactionAwareUserRepository) GetActiveUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, api_key, active,
		       created_at, updated_at, last_login_at
		FROM users 
		WHERE active = true
		ORDER BY last_login_at DESC NULLS LAST
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.executor.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer rows.Close()
	
	return r.scanUsers(rows)
}

func (r *transactionAwareUserRepository) UpdateLastLogin(ctx context.Context, userID int) error {
	query := `UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`
	
	now := time.Now()
	_, err := r.executor.ExecContext(ctx, query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	
	return nil
}

func (r *transactionAwareUserRepository) SetActive(ctx context.Context, userID int, active bool) error {
	query := `UPDATE users SET active = ?, updated_at = ? WHERE id = ?`
	
	now := time.Now()
	_, err := r.executor.ExecContext(ctx, query, active, now, userID)
	if err != nil {
		return fmt.Errorf("failed to set user active status: %w", err)
	}
	
	return nil
}

func (r *transactionAwareUserRepository) ChangePassword(ctx context.Context, userID int, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	
	now := time.Now()
	_, err := r.executor.ExecContext(ctx, query, passwordHash, now, userID)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	
	return nil
}

func (r *transactionAwareUserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`
	
	var count int
	err := r.executor.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	
	return count, nil
}

func (r *transactionAwareUserRepository) CountByRole(ctx context.Context, role string) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE role = ?`
	
	var count int
	err := r.executor.QueryRowContext(ctx, query, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}
	
	return count, nil
}

func (r *transactionAwareUserRepository) CountActive(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE active = true`
	
	var count int
	err := r.executor.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}
	
	return count, nil
}

// scanUsers is a helper function to scan multiple users from rows for transaction-aware repository
func (r *transactionAwareUserRepository) scanUsers(rows *sql.Rows) ([]*models.User, error) {
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

// Placeholder implementations for other transaction-aware repositories
// These will return "not implemented" errors for now

type transactionAwareSystemConfigRepository struct {
	executor DBExecutor
}

// SystemConfigRepository methods for transaction-aware implementation
func (r *transactionAwareSystemConfigRepository) Create(ctx context.Context, config *models.SystemConfig) error {
	query := `
		INSERT INTO system_config (key, value, value_type, category, description, is_secret, read_only, created_at, updated_at, updated_by, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.Version = 1
	
	result, err := r.executor.ExecContext(ctx, query,
		config.Key, config.Value, config.ValueType, config.Category,
		config.Description, config.IsSecret, config.ReadOnly,
		config.CreatedAt, config.UpdatedAt, config.UpdatedBy, config.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to create system config: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get created config ID: %w", err)
	}
	
	config.ID = int(id)
	return nil
}

func (r *transactionAwareSystemConfigRepository) GetByID(ctx context.Context, id int) (*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config WHERE id = ?
	`
	
	config := &models.SystemConfig{}
	var description sql.NullString
	var updatedBy sql.NullInt64
	
	err := r.executor.QueryRowContext(ctx, query, id).Scan(
		&config.ID, &config.Key, &config.Value, &config.ValueType,
		&config.Category, &description, &config.IsSecret, &config.ReadOnly,
		&config.CreatedAt, &config.UpdatedAt, &updatedBy, &config.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get config by ID: %w", err)
	}
	
	if description.Valid {
		config.Description = &description.String
	}
	if updatedBy.Valid {
		intVal := int(updatedBy.Int64)
		config.UpdatedBy = &intVal
	}
	
	return config, nil
}

func (r *transactionAwareSystemConfigRepository) GetByKey(ctx context.Context, key string) (*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config WHERE key = ?
	`
	
	config := &models.SystemConfig{}
	var description sql.NullString
	var updatedBy sql.NullInt64
	
	err := r.executor.QueryRowContext(ctx, query, key).Scan(
		&config.ID, &config.Key, &config.Value, &config.ValueType,
		&config.Category, &description, &config.IsSecret, &config.ReadOnly,
		&config.CreatedAt, &config.UpdatedAt, &updatedBy, &config.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get config by key: %w", err)
	}
	
	if description.Valid {
		config.Description = &description.String
	}
	if updatedBy.Valid {
		intVal := int(updatedBy.Int64)
		config.UpdatedBy = &intVal
	}
	
	return config, nil
}

func (r *transactionAwareSystemConfigRepository) Update(ctx context.Context, config *models.SystemConfig) error {
	query := `
		UPDATE system_config 
		SET value = ?, description = ?, is_secret = ?, read_only = ?, 
		    updated_at = ?, updated_by = ?, version = version + 1
		WHERE id = ?
	`
	
	config.UpdatedAt = time.Now()
	config.Version++
	
	result, err := r.executor.ExecContext(ctx, query,
		config.Value, config.Description, config.IsSecret, config.ReadOnly,
		config.UpdatedAt, config.UpdatedBy, config.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update system config: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("config not found")
	}
	
	return nil
}

func (r *transactionAwareSystemConfigRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM system_config WHERE id = ? AND read_only = false`
	
	result, err := r.executor.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete system config: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("config not found or is read-only")
	}
	
	return nil
}

func (r *transactionAwareSystemConfigRepository) List(ctx context.Context, filters *models.SystemConfigFilters) ([]*models.SystemConfig, error) {
	baseQuery := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config
	`
	
	var conditions []string
	var args []interface{}
	
	// Apply filters
	if filters.Category != nil {
		conditions = append(conditions, "category = ?")
		args = append(args, *filters.Category)
	}
	
	if filters.KeyPattern != nil {
		conditions = append(conditions, "key LIKE ?")
		args = append(args, "%"+*filters.KeyPattern+"%")
	}
	
	if filters.ValueType != nil {
		conditions = append(conditions, "value_type = ?")
		args = append(args, *filters.ValueType)
	}
	
	if filters.IsSecret != nil {
		conditions = append(conditions, "is_secret = ?")
		args = append(args, *filters.IsSecret)
	}
	
	if filters.ReadOnly != nil {
		conditions = append(conditions, "read_only = ?")
		args = append(args, *filters.ReadOnly)
	}
	
	if filters.UpdatedBy != nil {
		conditions = append(conditions, "updated_by = ?")
		args = append(args, *filters.UpdatedBy)
	}
	
	// Build query with conditions
	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Add ordering
	orderBy := "key"
	if filters.OrderBy != "" {
		orderBy = filters.OrderBy
	}
	orderDir := "ASC"
	if filters.OrderDir == "desc" {
		orderDir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)
	
	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filters.Offset)
		}
	}
	
	rows, err := r.executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list system configs: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

func (r *transactionAwareSystemConfigRepository) GetByCategory(ctx context.Context, category string) ([]*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config 
		WHERE category = ?
		ORDER BY key
	`
	
	rows, err := r.executor.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get configs by category: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

func (r *transactionAwareSystemConfigRepository) GetSecrets(ctx context.Context) ([]*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config 
		WHERE is_secret = true
		ORDER BY category, key
	`
	
	rows, err := r.executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret configs: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

func (r *transactionAwareSystemConfigRepository) GetNonSecrets(ctx context.Context) ([]*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config 
		WHERE is_secret = false
		ORDER BY category, key
	`
	
	rows, err := r.executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get non-secret configs: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

func (r *transactionAwareSystemConfigRepository) BulkUpdate(ctx context.Context, configs []*models.SystemConfig, updatedBy int) error {
	query := `
		UPDATE system_config 
		SET value = ?, updated_at = ?, updated_by = ?, version = version + 1
		WHERE id = ? AND read_only = false
	`
	
	now := time.Now()
	
	for _, config := range configs {
		config.UpdatedAt = now
		config.UpdatedBy = &updatedBy
		config.Version++
		
		result, err := r.executor.ExecContext(ctx, query, config.Value, config.UpdatedAt, config.UpdatedBy, config.ID)
		if err != nil {
			return fmt.Errorf("failed to update config %s: %w", config.Key, err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected for config %s: %w", config.Key, err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("config %s not found or is read-only", config.Key)
		}
	}
	
	return nil
}

func (r *transactionAwareSystemConfigRepository) GetValue(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM system_config WHERE key = ?`
	
	var value string
	err := r.executor.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("config key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get config value: %w", err)
	}
	
	return value, nil
}

func (r *transactionAwareSystemConfigRepository) SetValue(ctx context.Context, key, value string, updatedBy int) error {
	// Try to update first
	updateQuery := `
		UPDATE system_config 
		SET value = ?, updated_at = ?, updated_by = ?, version = version + 1
		WHERE key = ? AND read_only = false
	`
	
	now := time.Now()
	result, err := r.executor.ExecContext(ctx, updateQuery, value, now, updatedBy, key)
	if err != nil {
		return fmt.Errorf("failed to update config value: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected > 0 {
		return nil // Update successful
	}
	
	// If no rows were updated, try to create new config
	insertQuery := `
		INSERT INTO system_config (key, value, value_type, category, created_at, updated_at, updated_by, version)
		VALUES (?, ?, 'string', 'general', ?, ?, ?, 1)
	`
	
	_, err = r.executor.ExecContext(ctx, insertQuery, key, value, now, now, updatedBy)
	if err != nil {
		return fmt.Errorf("failed to create config value: %w", err)
	}
	
	return nil
}

func (r *transactionAwareSystemConfigRepository) GetCategories(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT category FROM system_config ORDER BY category`
	
	rows, err := r.executor.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()
	
	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}
	
	return categories, nil
}

// scanConfigs helper for transaction-aware SystemConfig repository
func (r *transactionAwareSystemConfigRepository) scanConfigs(rows *sql.Rows) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	
	for rows.Next() {
		config := &models.SystemConfig{}
		var description sql.NullString
		var updatedBy sql.NullInt64
		
		err := rows.Scan(
			&config.ID, &config.Key, &config.Value, &config.ValueType,
			&config.Category, &description, &config.IsSecret, &config.ReadOnly,
			&config.CreatedAt, &config.UpdatedAt, &updatedBy, &config.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config: %w", err)
		}
		
		if description.Valid {
			config.Description = &description.String
		}
		if updatedBy.Valid {
			intVal := int(updatedBy.Int64)
			config.UpdatedBy = &intVal
		}
		
		configs = append(configs, config)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating config rows: %w", err)
	}
	
	return configs, nil
}

type transactionAwareMetricsRepository struct {
	executor DBExecutor
}

// Placeholder implementation - MetricsRepository methods will be implemented later
func (r *transactionAwareMetricsRepository) Create(ctx context.Context, metric *models.Metric) error {
	return fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) CreateBatch(ctx context.Context, metrics []*models.Metric) error {
	return fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetByID(ctx context.Context, id int) (*models.Metric, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) List(ctx context.Context, filters *models.MetricFilters) ([]*models.Metric, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetByName(ctx context.Context, name string, limit, offset int) ([]*models.Metric, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetByType(ctx context.Context, metricType string, limit, offset int) ([]*models.Metric, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetBySource(ctx context.Context, source string, limit, offset int) ([]*models.Metric, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetTimeSeries(ctx context.Context, name string, startTime, endTime time.Time, interval string) (*models.MetricTimeSeriesResponse, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	return 0, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetAggregated(ctx context.Context, filters *models.MetricFilters) ([]*models.MetricAggregation, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) GetSummary(ctx context.Context, startTime, endTime time.Time) (*models.MetricSummary, error) {
	return nil, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) Count(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) CountByType(ctx context.Context, metricType string) (int, error) {
	return 0, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

func (r *transactionAwareMetricsRepository) CountBySource(ctx context.Context, source string) (int, error) {
	return 0, fmt.Errorf("transaction-aware metrics repository not fully implemented yet")
}

type transactionAwareSessionRepository struct {
	executor DBExecutor
}

// SessionRepository methods for transaction-aware implementation
func (r *transactionAwareSessionRepository) Create(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	session.CreatedAt = now
	session.LastUsedAt = now
	
	result, err := r.executor.ExecContext(ctx, query,
		session.UserID, session.Token, session.ExpiresAt, session.CreatedAt,
		session.LastUsedAt, session.IPAddress, session.UserAgent, session.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get created session ID: %w", err)
	}
	
	session.ID = int(id)
	return nil
}

func (r *transactionAwareSessionRepository) GetByID(ctx context.Context, id int) (*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions WHERE id = ?
	`
	
	session := &models.Session{}
	var ipAddress, userAgent sql.NullString
	
	err := r.executor.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}
	
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	
	return session, nil
}

func (r *transactionAwareSessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions WHERE token = ?
	`
	
	session := &models.Session{}
	var ipAddress, userAgent sql.NullString
	
	err := r.executor.QueryRowContext(ctx, query, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}
	
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	
	return session, nil
}

func (r *transactionAwareSessionRepository) Update(ctx context.Context, session *models.Session) error {
	query := `
		UPDATE sessions 
		SET user_id = ?, token = ?, expires_at = ?, last_used_at = ?, 
		    ip_address = ?, user_agent = ?, active = ?
		WHERE id = ?
	`
	
	result, err := r.executor.ExecContext(ctx, query,
		session.UserID, session.Token, session.ExpiresAt, session.LastUsedAt,
		session.IPAddress, session.UserAgent, session.Active, session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

func (r *transactionAwareSessionRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM sessions WHERE id = ?`
	
	result, err := r.executor.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

func (r *transactionAwareSessionRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions 
		WHERE user_id = ?
		ORDER BY last_used_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.executor.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}
	defer rows.Close()
	
	return r.scanSessions(rows)
}

func (r *transactionAwareSessionRepository) GetActiveByUserID(ctx context.Context, userID int) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions 
		WHERE user_id = ? AND active = true AND expires_at > ?
		ORDER BY last_used_at DESC
	`
	
	now := time.Now()
	rows, err := r.executor.QueryContext(ctx, query, userID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions by user ID: %w", err)
	}
	defer rows.Close()
	
	return r.scanSessions(rows)
}

func (r *transactionAwareSessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	query := `DELETE FROM sessions WHERE expires_at <= ?`
	
	now := time.Now()
	result, err := r.executor.ExecContext(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

func (r *transactionAwareSessionRepository) DeleteByUserID(ctx context.Context, userID int) (int, error) {
	query := `DELETE FROM sessions WHERE user_id = ?`
	
	result, err := r.executor.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete sessions by user ID: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

func (r *transactionAwareSessionRepository) RefreshSession(ctx context.Context, sessionID int, extendBy time.Duration) error {
	var query string
	var args []interface{}
	
	now := time.Now()
	
	if extendBy > 0 {
		query = `
			UPDATE sessions 
			SET last_used_at = ?, expires_at = ? 
			WHERE id = ? AND active = true
		`
		newExpiry := now.Add(extendBy)
		args = []interface{}{now, newExpiry, sessionID}
	} else {
		query = `
			UPDATE sessions 
			SET last_used_at = ? 
			WHERE id = ? AND active = true
		`
		args = []interface{}{now, sessionID}
	}
	
	result, err := r.executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found or not active")
	}
	
	return nil
}

func (r *transactionAwareSessionRepository) Deactivate(ctx context.Context, sessionID int) error {
	query := `UPDATE sessions SET active = false WHERE id = ?`
	
	result, err := r.executor.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to deactivate session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

func (r *transactionAwareSessionRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM sessions`
	
	var count int
	err := r.executor.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	
	return count, nil
}

func (r *transactionAwareSessionRepository) CountActive(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE active = true AND expires_at > ?`
	
	now := time.Now()
	var count int
	err := r.executor.QueryRowContext(ctx, query, now).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}
	
	return count, nil
}

func (r *transactionAwareSessionRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE user_id = ?`
	
	var count int
	err := r.executor.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions by user ID: %w", err)
	}
	
	return count, nil
}

func (r *transactionAwareSessionRepository) GetSessionAnalytics(ctx context.Context, startTime, endTime time.Time) (*SessionAnalytics, error) {
	// Delegate to the concrete implementation with proper type conversion
	if db, ok := r.executor.(*sql.DB); ok {
		sessionRepo := &sessionRepository{db: db}
		return sessionRepo.GetSessionAnalytics(ctx, startTime, endTime)
	}
	return nil, fmt.Errorf("analytics not supported in transaction context")
}

func (r *transactionAwareSessionRepository) CleanupInactiveSessions(ctx context.Context, inactiveDuration time.Duration) (int, error) {
	// Delegate to the concrete implementation with proper type conversion
	if db, ok := r.executor.(*sql.DB); ok {
		sessionRepo := &sessionRepository{db: db}
		return sessionRepo.CleanupInactiveSessions(ctx, inactiveDuration)
	}
	return 0, fmt.Errorf("cleanup not supported in transaction context")
}

// scanSessions helper for transaction-aware Session repository
func (r *transactionAwareSessionRepository) scanSessions(rows *sql.Rows) ([]*models.Session, error) {
	var sessions []*models.Session
	
	for rows.Next() {
		session := &models.Session{}
		var ipAddress, userAgent sql.NullString
		
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
			&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		
		if ipAddress.Valid {
			session.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		
		sessions = append(sessions, session)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}
	
	return sessions, nil
}

type transactionAwareRequestRepository struct {
	executor DBExecutor
}

// Placeholder implementation - RequestRepository methods will be implemented later
func (r *transactionAwareRequestRepository) Create(ctx context.Context, request *models.Request) error {
	return fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) GetByID(ctx context.Context, id int) (*models.Request, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) Update(ctx context.Context, request *models.Request) error {
	return fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) List(ctx context.Context, filters *models.RequestFilters) ([]*models.Request, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Request, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) GetBySessionID(ctx context.Context, sessionID int, limit, offset int) ([]*models.Request, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) GetByIPAddress(ctx context.Context, ipAddress string, limit, offset int) ([]*models.Request, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) GetBlocked(ctx context.Context, limit, offset int) ([]*models.Request, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	return 0, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (*models.RequestStats, error) {
	return nil, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) Count(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) CountBlocked(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

func (r *transactionAwareRequestRepository) CountByStatusCode(ctx context.Context, statusCode int) (int, error) {
	return 0, fmt.Errorf("transaction-aware request repository not fully implemented yet")
}

type transactionAwareModerationLogRepository struct {
	executor DBExecutor
}

// Placeholder implementation - ModerationLogRepository methods will be implemented later  
func (r *transactionAwareModerationLogRepository) Create(ctx context.Context, log *models.ModerationLog) error {
	return fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) GetByID(ctx context.Context, id int) (*models.ModerationLog, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) Update(ctx context.Context, log *models.ModerationLog) error {
	return fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) List(ctx context.Context, filters *models.ModerationLogFilters) ([]*models.ModerationLog, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) GetByRequestID(ctx context.Context, requestID int, limit, offset int) ([]*models.ModerationLog, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.ModerationLog, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) GetBySeverity(ctx context.Context, severity string, limit, offset int) ([]*models.ModerationLog, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) GetPendingReview(ctx context.Context, limit, offset int) ([]*models.ModerationLog, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) MarkReviewed(ctx context.Context, logID, reviewerID int, status string) error {
	return fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	return 0, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (*models.ModerationStats, error) {
	return nil, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) Count(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) CountBySeverity(ctx context.Context, severity string) (int, error) {
	return 0, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

func (r *transactionAwareModerationLogRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	return 0, fmt.Errorf("transaction-aware moderation log repository not fully implemented yet")
}

type transactionAwareKillSwitchRepository struct {
	executor DBExecutor
}

// Placeholder implementation - KillSwitchRepository methods will be implemented later
func (r *transactionAwareKillSwitchRepository) Create(ctx context.Context, entry *models.KillSwitchEntry) error {
	return fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) GetByID(ctx context.Context, id int) (*models.KillSwitchEntry, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) Update(ctx context.Context, entry *models.KillSwitchEntry) error {
	return fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) List(ctx context.Context, filters *models.KillSwitchFilters) ([]*models.KillSwitchEntry, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) GetByType(ctx context.Context, entryType string) ([]*models.KillSwitchEntry, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) GetActive(ctx context.Context) ([]*models.KillSwitchEntry, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) GetExpired(ctx context.Context) ([]*models.KillSwitchEntry, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) CheckBlocked(ctx context.Context, entryType, value string) (*models.KillSwitchEntry, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) RecordHit(ctx context.Context, entryID int) error {
	return fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) DeactivateExpired(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) GetStats(ctx context.Context, startTime, endTime time.Time) (*models.KillSwitchStats, error) {
	return nil, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) Count(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) CountActive(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}

func (r *transactionAwareKillSwitchRepository) CountByType(ctx context.Context, entryType string) (int, error) {
	return 0, fmt.Errorf("transaction-aware kill switch repository not fully implemented yet")
}