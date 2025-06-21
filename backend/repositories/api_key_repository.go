package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"qt1-middleware/models"
	"strings"
	"time"
)

// apiKeyRepository implements the APIKeyRepository interface
type apiKeyRepository struct {
	db DBExecutor
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db DBExecutor) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

// Create creates a new API key
func (r *apiKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, name, description, key_hash, active, created_at, expires_at, usage_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := r.db.ExecContext(ctx, query,
		apiKey.ID,
		apiKey.UserID,
		apiKey.Name,
		apiKey.Description,
		apiKey.KeyHash,
		apiKey.Active,
		apiKey.CreatedAt,
		apiKey.ExpiresAt,
		apiKey.UsageCount,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	
	// Store permissions separately if provided
	if len(apiKey.Permissions) > 0 {
		if err := r.storePermissions(ctx, apiKey.ID, apiKey.Permissions); err != nil {
			// Try to clean up the API key if permission storage fails
			r.db.ExecContext(ctx, "DELETE FROM api_keys WHERE id = $1", apiKey.ID)
			return fmt.Errorf("failed to store API key permissions: %w", err)
		}
	}
	
	return nil
}

// GetByID retrieves an API key by ID
func (r *apiKeyRepository) GetByID(ctx context.Context, id string) (*models.APIKey, error) {
	query := `
		SELECT id, user_id, name, description, key_hash, active, last_used, created_at, expires_at, usage_count
		FROM api_keys 
		WHERE id = $1`
	
	apiKey := &models.APIKey{}
	var lastUsed sql.NullTime
	var expiresAt sql.NullTime
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&apiKey.ID,
		&apiKey.UserID,
		&apiKey.Name,
		&apiKey.Description,
		&apiKey.KeyHash,
		&apiKey.Active,
		&lastUsed,
		&apiKey.CreatedAt,
		&expiresAt,
		&apiKey.UsageCount,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	
	// Handle nullable fields
	if lastUsed.Valid {
		apiKey.LastUsed = &lastUsed.Time
	}
	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
	}
	
	// Load permissions
	permissions, err := r.getPermissions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key permissions: %w", err)
	}
	apiKey.Permissions = permissions
	
	return apiKey, nil
}

// Update updates an existing API key
func (r *apiKeyRepository) Update(ctx context.Context, apiKey *models.APIKey) error {
	query := `
		UPDATE api_keys 
		SET name = $2, description = $3, active = $4, expires_at = $5, usage_count = $6
		WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query,
		apiKey.ID,
		apiKey.Name,
		apiKey.Description,
		apiKey.Active,
		apiKey.ExpiresAt,
		apiKey.UsageCount,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	
	// Update permissions if they have changed
	if apiKey.Permissions != nil {
		// Delete existing permissions
		if err := r.deletePermissions(ctx, apiKey.ID); err != nil {
			return fmt.Errorf("failed to delete existing permissions: %w", err)
		}
		
		// Store new permissions
		if len(apiKey.Permissions) > 0 {
			if err := r.storePermissions(ctx, apiKey.ID, apiKey.Permissions); err != nil {
				return fmt.Errorf("failed to store updated permissions: %w", err)
			}
		}
	}
	
	return nil
}

// Delete deletes an API key
func (r *apiKeyRepository) Delete(ctx context.Context, id string) error {
	// Delete permissions first (foreign key constraint)
	if err := r.deletePermissions(ctx, id); err != nil {
		return fmt.Errorf("failed to delete API key permissions: %w", err)
	}
	
	// Delete usage records
	_, err := r.db.ExecContext(ctx, "DELETE FROM api_key_usage WHERE api_key_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete API key usage records: %w", err)
	}
	
	// Delete the API key
	query := `DELETE FROM api_keys WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}
	
	return nil
}

// GetByUserID retrieves API keys for a specific user
func (r *apiKeyRepository) GetByUserID(ctx context.Context, userID int, activeOnly bool) ([]*models.APIKey, error) {
	query := `
		SELECT id, user_id, name, description, key_hash, active, last_used, created_at, expires_at, usage_count
		FROM api_keys 
		WHERE user_id = $1`
	
	var args []interface{}
	args = append(args, userID)
	
	if activeOnly {
		query += " AND active = $2"
		args = append(args, true)
	}
	
	query += " ORDER BY created_at DESC"
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys by user ID: %w", err)
	}
	defer rows.Close()
	
	var apiKeys []*models.APIKey
	for rows.Next() {
		apiKey := &models.APIKey{}
		var lastUsed sql.NullTime
		var expiresAt sql.NullTime
		
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.Name,
			&apiKey.Description,
			&apiKey.KeyHash,
			&apiKey.Active,
			&lastUsed,
			&apiKey.CreatedAt,
			&expiresAt,
			&apiKey.UsageCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		
		// Handle nullable fields
		if lastUsed.Valid {
			apiKey.LastUsed = &lastUsed.Time
		}
		if expiresAt.Valid {
			apiKey.ExpiresAt = &expiresAt.Time
		}
		
		// Load permissions for each key
		permissions, err := r.getPermissions(ctx, apiKey.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get permissions for API key %s: %w", apiKey.ID, err)
		}
		apiKey.Permissions = permissions
		
		apiKeys = append(apiKeys, apiKey)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API key rows: %w", err)
	}
	
	return apiKeys, nil
}

// List retrieves API keys based on filters
func (r *apiKeyRepository) List(ctx context.Context, filters *models.APIKeyFilters) ([]*models.APIKey, error) {
	query := `
		SELECT id, user_id, name, description, key_hash, active, last_used, created_at, expires_at, usage_count
		FROM api_keys WHERE 1=1`
	
	var args []interface{}
	argCount := 0
	
	// Apply filters
	if filters != nil {
		if filters.UserID != nil {
			argCount++
			query += fmt.Sprintf(" AND user_id = $%d", argCount)
			args = append(args, *filters.UserID)
		}
		
		if filters.Active != nil {
			argCount++
			query += fmt.Sprintf(" AND active = $%d", argCount)
			args = append(args, *filters.Active)
		}
		
		if filters.Expired != nil {
			if *filters.Expired {
				query += " AND (expires_at IS NOT NULL AND expires_at < NOW())"
			} else {
				query += " AND (expires_at IS NULL OR expires_at >= NOW())"
			}
		}
		
		if filters.UsedSince != nil {
			argCount++
			query += fmt.Sprintf(" AND last_used >= $%d", argCount)
			args = append(args, *filters.UsedSince)
		}
		
		if filters.CreatedAfter != nil {
			argCount++
			query += fmt.Sprintf(" AND created_at >= $%d", argCount)
			args = append(args, *filters.CreatedAfter)
		}
		
		if filters.Search != "" {
			argCount++
			query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
			args = append(args, "%"+filters.Search+"%")
		}
		
		// Sorting
		if filters.SortBy != "" {
			allowedSorts := map[string]bool{
				"name": true, "created_at": true, "last_used": true, "usage_count": true,
			}
			if allowedSorts[filters.SortBy] {
				sortOrder := "ASC"
				if filters.SortOrder == "desc" {
					sortOrder = "DESC"
				}
				query += fmt.Sprintf(" ORDER BY %s %s", filters.SortBy, sortOrder)
			} else {
				query += " ORDER BY created_at DESC"
			}
		} else {
			query += " ORDER BY created_at DESC"
		}
		
		// Pagination
		if filters.Limit > 0 {
			argCount++
			query += fmt.Sprintf(" LIMIT $%d", argCount)
			args = append(args, filters.Limit)
		}
		
		if filters.Offset > 0 {
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, filters.Offset)
		}
	} else {
		query += " ORDER BY created_at DESC"
	}
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()
	
	var apiKeys []*models.APIKey
	for rows.Next() {
		apiKey := &models.APIKey{}
		var lastUsed sql.NullTime
		var expiresAt sql.NullTime
		
		err := rows.Scan(
			&apiKey.ID,
			&apiKey.UserID,
			&apiKey.Name,
			&apiKey.Description,
			&apiKey.KeyHash,
			&apiKey.Active,
			&lastUsed,
			&apiKey.CreatedAt,
			&expiresAt,
			&apiKey.UsageCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		
		// Handle nullable fields
		if lastUsed.Valid {
			apiKey.LastUsed = &lastUsed.Time
		}
		if expiresAt.Valid {
			apiKey.ExpiresAt = &expiresAt.Time
		}
		
		// Load permissions for each key
		permissions, err := r.getPermissions(ctx, apiKey.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get permissions for API key %s: %w", apiKey.ID, err)
		}
		apiKey.Permissions = permissions
		
		apiKeys = append(apiKeys, apiKey)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API key rows: %w", err)
	}
	
	return apiKeys, nil
}

// UpdateLastUsed updates the last used timestamp and increments usage count
func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, keyID string, lastUsed time.Time) error {
	query := `
		UPDATE api_keys 
		SET last_used = $2, usage_count = usage_count + 1
		WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, keyID, lastUsed)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}
	
	return nil
}

// DeleteExpired deletes expired API keys older than the cutoff date
func (r *apiKeyRepository) DeleteExpired(ctx context.Context, cutoff time.Time) (int, error) {
	// First get the IDs of keys to be deleted for cleanup
	selectQuery := `SELECT id FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < $1`
	rows, err := r.db.QueryContext(ctx, selectQuery, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to query expired API keys: %w", err)
	}
	defer rows.Close()
	
	var keyIDs []string
	for rows.Next() {
		var keyID string
		if err := rows.Scan(&keyID); err != nil {
			return 0, fmt.Errorf("failed to scan key ID: %w", err)
		}
		keyIDs = append(keyIDs, keyID)
	}
	
	if len(keyIDs) == 0 {
		return 0, nil
	}
	
	// Delete permissions and usage records for expired keys
	for _, keyID := range keyIDs {
		r.deletePermissions(ctx, keyID)
		r.db.ExecContext(ctx, "DELETE FROM api_key_usage WHERE api_key_id = $1", keyID)
	}
	
	// Delete the expired API keys
	deleteQuery := `DELETE FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < $1`
	result, err := r.db.ExecContext(ctx, deleteQuery, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired API keys: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// GetUsage retrieves usage statistics for an API key
func (r *apiKeyRepository) GetUsage(ctx context.Context, keyID string, startDate time.Time) ([]*models.APIKeyUsage, error) {
	query := `
		SELECT api_key_id, date, request_count, last_request
		FROM api_key_usage 
		WHERE api_key_id = $1 AND date >= $2
		ORDER BY date DESC`
	
	rows, err := r.db.QueryContext(ctx, query, keyID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query API key usage: %w", err)
	}
	defer rows.Close()
	
	var usage []*models.APIKeyUsage
	for rows.Next() {
		u := &models.APIKeyUsage{}
		err := rows.Scan(
			&u.APIKeyID,
			&u.Date,
			&u.RequestCount,
			&u.LastRequest,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key usage: %w", err)
		}
		usage = append(usage, u)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating usage rows: %w", err)
	}
	
	return usage, nil
}

// RecordUsage records a usage event for an API key
func (r *apiKeyRepository) RecordUsage(ctx context.Context, keyID, ipAddress, userAgent string) error {
	today := time.Now().Truncate(24 * time.Hour)
	
	// Insert or update daily usage record
	query := `
		INSERT INTO api_key_usage (api_key_id, date, request_count, last_request)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (api_key_id, date)
		DO UPDATE SET 
			request_count = api_key_usage.request_count + 1,
			last_request = NOW()`
	
	_, err := r.db.ExecContext(ctx, query, keyID, today)
	if err != nil {
		return fmt.Errorf("failed to record API key usage: %w", err)
	}
	
	return nil
}

// GetStats retrieves overall API key statistics
func (r *apiKeyRepository) GetStats(ctx context.Context) (*models.APIKeyStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_keys,
			COUNT(CASE WHEN active = true THEN 1 END) as active_keys,
			COUNT(CASE WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN 1 END) as expired_keys,
			COUNT(CASE WHEN active = false THEN 1 END) as disabled_keys,
			COUNT(CASE WHEN last_used >= CURRENT_DATE THEN 1 END) as keys_used_today,
			COALESCE(SUM(usage_count), 0) as total_requests,
			COALESCE(SUM(CASE WHEN last_used >= CURRENT_DATE THEN usage_count ELSE 0 END), 0) as requests_today
		FROM api_keys`
	
	stats := &models.APIKeyStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalKeys,
		&stats.ActiveKeys,
		&stats.ExpiredKeys,
		&stats.DisabledKeys,
		&stats.KeysUsedToday,
		&stats.TotalRequests,
		&stats.RequestsToday,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get API key stats: %w", err)
	}
	
	return stats, nil
}

// Count returns the total number of API keys
func (r *apiKeyRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_keys").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count API keys: %w", err)
	}
	return count, nil
}

// CountActive returns the number of active API keys
func (r *apiKeyRepository) CountActive(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_keys WHERE active = true").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active API keys: %w", err)
	}
	return count, nil
}

// CountByUserID returns the number of API keys for a specific user
func (r *apiKeyRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_keys WHERE user_id = $1", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count API keys by user ID: %w", err)
	}
	return count, nil
}

// Helper methods for permissions management

func (r *apiKeyRepository) storePermissions(ctx context.Context, keyID string, permissions []string) error {
	if len(permissions) == 0 {
		return nil
	}
	
	// Build bulk insert query
	valueStrings := make([]string, 0, len(permissions))
	valueArgs := make([]interface{}, 0, len(permissions)*2)
	
	for i, permission := range permissions {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		valueArgs = append(valueArgs, keyID, permission)
	}
	
	query := fmt.Sprintf("INSERT INTO api_key_permissions (api_key_id, permission) VALUES %s",
		strings.Join(valueStrings, ","))
	
	_, err := r.db.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to store API key permissions: %w", err)
	}
	
	return nil
}

func (r *apiKeyRepository) getPermissions(ctx context.Context, keyID string) ([]string, error) {
	query := `SELECT permission FROM api_key_permissions WHERE api_key_id = $1 ORDER BY permission`
	
	rows, err := r.db.QueryContext(ctx, query, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API key permissions: %w", err)
	}
	defer rows.Close()
	
	var permissions []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission rows: %w", err)
	}
	
	return permissions, nil
}

func (r *apiKeyRepository) deletePermissions(ctx context.Context, keyID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM api_key_permissions WHERE api_key_id = $1", keyID)
	if err != nil {
		return fmt.Errorf("failed to delete API key permissions: %w", err)
	}
	return nil
}