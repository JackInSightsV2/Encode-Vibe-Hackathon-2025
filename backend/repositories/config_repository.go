package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"qt1-middleware/models"
	"strings"
	"time"
)

// systemConfigRepository implements SystemConfigRepository interface
type systemConfigRepository struct {
	db *sql.DB
}

// NewSystemConfigRepository creates a new system config repository
func NewSystemConfigRepository(db *sql.DB) SystemConfigRepository {
	return &systemConfigRepository{db: db}
}

// Create inserts a new system config entry
func (r *systemConfigRepository) Create(ctx context.Context, config *models.SystemConfig) error {
	query := `
		INSERT INTO system_config (key, value, value_type, category, description, is_secret, read_only, created_at, updated_at, updated_by, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.Version = 1
	
	result, err := r.db.ExecContext(ctx, query,
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

// GetByID retrieves a system config by ID
func (r *systemConfigRepository) GetByID(ctx context.Context, id int) (*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config WHERE id = ?
	`
	
	config := &models.SystemConfig{}
	var description sql.NullString
	var updatedBy sql.NullInt64
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
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

// GetByKey retrieves a system config by key
func (r *systemConfigRepository) GetByKey(ctx context.Context, key string) (*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config WHERE key = ?
	`
	
	config := &models.SystemConfig{}
	var description sql.NullString
	var updatedBy sql.NullInt64
	
	err := r.db.QueryRowContext(ctx, query, key).Scan(
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

// Update updates an existing system config
func (r *systemConfigRepository) Update(ctx context.Context, config *models.SystemConfig) error {
	query := `
		UPDATE system_config 
		SET value = ?, description = ?, is_secret = ?, read_only = ?, 
		    updated_at = ?, updated_by = ?, version = version + 1
		WHERE id = ?
	`
	
	config.UpdatedAt = time.Now()
	config.Version++
	
	result, err := r.db.ExecContext(ctx, query,
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

// Delete deletes a system config by ID
func (r *systemConfigRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM system_config WHERE id = ? AND read_only = false`
	
	result, err := r.db.ExecContext(ctx, query, id)
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

// List retrieves system configs with filtering
func (r *systemConfigRepository) List(ctx context.Context, filters *models.SystemConfigFilters) ([]*models.SystemConfig, error) {
	baseQuery := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config
	`
	
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	// Apply filters
	if filters.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = ?"))
		args = append(args, *filters.Category)
		argIndex++
	}
	
	if filters.KeyPattern != nil {
		conditions = append(conditions, fmt.Sprintf("key LIKE ?"))
		args = append(args, "%"+*filters.KeyPattern+"%")
		argIndex++
	}
	
	if filters.ValueType != nil {
		conditions = append(conditions, fmt.Sprintf("value_type = ?"))
		args = append(args, *filters.ValueType)
		argIndex++
	}
	
	if filters.IsSecret != nil {
		conditions = append(conditions, fmt.Sprintf("is_secret = ?"))
		args = append(args, *filters.IsSecret)
		argIndex++
	}
	
	if filters.ReadOnly != nil {
		conditions = append(conditions, fmt.Sprintf("read_only = ?"))
		args = append(args, *filters.ReadOnly)
		argIndex++
	}
	
	if filters.UpdatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("updated_by = ?"))
		args = append(args, *filters.UpdatedBy)
		argIndex++
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
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list system configs: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

// GetByCategory retrieves all configs in a specific category
func (r *systemConfigRepository) GetByCategory(ctx context.Context, category string) ([]*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config 
		WHERE category = ?
		ORDER BY key
	`
	
	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get configs by category: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

// GetSecrets retrieves all secret configurations
func (r *systemConfigRepository) GetSecrets(ctx context.Context) ([]*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config 
		WHERE is_secret = true
		ORDER BY category, key
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret configs: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

// GetNonSecrets retrieves all non-secret configurations
func (r *systemConfigRepository) GetNonSecrets(ctx context.Context) ([]*models.SystemConfig, error) {
	query := `
		SELECT id, key, value, value_type, category, description, is_secret, read_only,
		       created_at, updated_at, updated_by, version
		FROM system_config 
		WHERE is_secret = false
		ORDER BY category, key
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get non-secret configs: %w", err)
	}
	defer rows.Close()
	
	return r.scanConfigs(rows)
}

// BulkUpdate updates multiple configurations in a transaction
func (r *systemConfigRepository) BulkUpdate(ctx context.Context, configs []*models.SystemConfig, updatedBy int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
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
		
		result, err := tx.ExecContext(ctx, query, config.Value, config.UpdatedAt, config.UpdatedBy, config.ID)
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
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit bulk update: %w", err)
	}
	
	return nil
}

// GetValue retrieves a config value by key
func (r *systemConfigRepository) GetValue(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM system_config WHERE key = ?`
	
	var value string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("config key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get config value: %w", err)
	}
	
	return value, nil
}

// SetValue sets a config value by key, creating if not exists
func (r *systemConfigRepository) SetValue(ctx context.Context, key, value string, updatedBy int) error {
	// Try to update first
	updateQuery := `
		UPDATE system_config 
		SET value = ?, updated_at = ?, updated_by = ?, version = version + 1
		WHERE key = ? AND read_only = false
	`
	
	now := time.Now()
	result, err := r.db.ExecContext(ctx, updateQuery, value, now, updatedBy, key)
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
	
	_, err = r.db.ExecContext(ctx, insertQuery, key, value, now, now, updatedBy)
	if err != nil {
		return fmt.Errorf("failed to create config value: %w", err)
	}
	
	return nil
}

// GetCategories retrieves all distinct categories
func (r *systemConfigRepository) GetCategories(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT category FROM system_config ORDER BY category`
	
	rows, err := r.db.QueryContext(ctx, query)
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

// scanConfigs is a helper function to scan multiple configs from rows
func (r *systemConfigRepository) scanConfigs(rows *sql.Rows) ([]*models.SystemConfig, error) {
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