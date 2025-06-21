package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SQLBackupRepository implements BackupRepository using SQL database
type SQLBackupRepository struct {
	db *sql.DB
}

// NewSQLBackupRepository creates a new SQL backup repository
func NewSQLBackupRepository(db *sql.DB) *SQLBackupRepository {
	return &SQLBackupRepository{db: db}
}

// InitializeSchema creates the necessary tables for backup metadata
func (r *SQLBackupRepository) InitializeSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS backup_metadata (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration INTEGER,
			size INTEGER NOT NULL DEFAULT 0,
			compressed_size INTEGER NOT NULL DEFAULT 0,
			compression_type TEXT NOT NULL DEFAULT 'none',
			encryption_type TEXT NOT NULL DEFAULT 'none',
			file_path TEXT,
			checksum TEXT,
			parent_backup_id TEXT,
			database_version TEXT,
			app_version TEXT,
			tags TEXT, -- JSON array
			description TEXT,
			error_message TEXT,
			created_by TEXT NOT NULL,
			retention_policy TEXT, -- JSON object
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_backup_id) REFERENCES backup_metadata(id)
		)`,
		
		`CREATE TABLE IF NOT EXISTS backup_progress (
			backup_id TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			percent_complete REAL NOT NULL DEFAULT 0.0,
			bytes_processed INTEGER NOT NULL DEFAULT 0,
			total_bytes INTEGER NOT NULL DEFAULT 0,
			current_table TEXT,
			tables_complete INTEGER NOT NULL DEFAULT 0,
			total_tables INTEGER NOT NULL DEFAULT 0,
			estimated_time INTEGER, -- Duration in nanoseconds
			error_message TEXT,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (backup_id) REFERENCES backup_metadata(id) ON DELETE CASCADE
		)`,
		
		`CREATE TABLE IF NOT EXISTS backup_schedules (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			cron_expression TEXT NOT NULL,
			backup_options TEXT NOT NULL, -- JSON object
			retention_policy TEXT NOT NULL, -- JSON object
			enabled BOOLEAN NOT NULL DEFAULT true,
			last_run DATETIME,
			next_run DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_by TEXT NOT NULL
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_backup_metadata_type ON backup_metadata(type)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_metadata_status ON backup_metadata(status)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_metadata_start_time ON backup_metadata(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_metadata_created_by ON backup_metadata(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_metadata_parent ON backup_metadata(parent_backup_id)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_schedules_enabled ON backup_schedules(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_backup_schedules_next_run ON backup_schedules(next_run)`,
	}
	
	for _, query := range queries {
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}
	
	return nil
}

// CreateBackupRecord saves backup metadata to storage
func (r *SQLBackupRepository) CreateBackupRecord(ctx context.Context, metadata *BackupMetadata) error {
	query := `
		INSERT INTO backup_metadata (
			id, type, status, start_time, end_time, duration, size, compressed_size,
			compression_type, encryption_type, file_path, checksum, parent_backup_id,
			database_version, app_version, tags, description, error_message,
			created_by, retention_policy
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// Serialize JSON fields
	tagsJSON, _ := json.Marshal(metadata.Tags)
	var retentionJSON []byte
	if metadata.RetentionPolicy != nil {
		retentionJSON, _ = json.Marshal(metadata.RetentionPolicy)
	}
	
	var duration *int64
	if metadata.Duration > 0 {
		d := int64(metadata.Duration)
		duration = &d
	}
	
	_, err := r.db.ExecContext(ctx, query,
		metadata.ID, metadata.Type, metadata.Status, metadata.StartTime,
		metadata.EndTime, duration, metadata.Size, metadata.CompressedSize,
		metadata.CompressionType, metadata.EncryptionType, metadata.FilePath,
		metadata.Checksum, metadata.ParentBackupID, metadata.DatabaseVersion,
		metadata.AppVersion, string(tagsJSON), metadata.Description,
		metadata.ErrorMessage, metadata.CreatedBy, string(retentionJSON),
	)
	
	if err != nil {
		return fmt.Errorf("failed to create backup record: %w", err)
	}
	
	return nil
}

// UpdateBackupRecord updates existing backup metadata
func (r *SQLBackupRepository) UpdateBackupRecord(ctx context.Context, metadata *BackupMetadata) error {
	query := `
		UPDATE backup_metadata SET
			type = ?, status = ?, start_time = ?, end_time = ?, duration = ?,
			size = ?, compressed_size = ?, compression_type = ?, encryption_type = ?,
			file_path = ?, checksum = ?, parent_backup_id = ?, database_version = ?,
			app_version = ?, tags = ?, description = ?, error_message = ?,
			created_by = ?, retention_policy = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	// Serialize JSON fields
	tagsJSON, _ := json.Marshal(metadata.Tags)
	var retentionJSON []byte
	if metadata.RetentionPolicy != nil {
		retentionJSON, _ = json.Marshal(metadata.RetentionPolicy)
	}
	
	var duration *int64
	if metadata.Duration > 0 {
		d := int64(metadata.Duration)
		duration = &d
	}
	
	result, err := r.db.ExecContext(ctx, query,
		metadata.Type, metadata.Status, metadata.StartTime, metadata.EndTime,
		duration, metadata.Size, metadata.CompressedSize, metadata.CompressionType,
		metadata.EncryptionType, metadata.FilePath, metadata.Checksum,
		metadata.ParentBackupID, metadata.DatabaseVersion, metadata.AppVersion,
		string(tagsJSON), metadata.Description, metadata.ErrorMessage,
		metadata.CreatedBy, string(retentionJSON), metadata.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update backup record: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("backup record not found: %s", metadata.ID)
	}
	
	return nil
}

// GetBackupRecord retrieves backup metadata by ID
func (r *SQLBackupRepository) GetBackupRecord(ctx context.Context, backupID string) (*BackupMetadata, error) {
	query := `
		SELECT id, type, status, start_time, end_time, duration, size, compressed_size,
			   compression_type, encryption_type, file_path, checksum, parent_backup_id,
			   database_version, app_version, tags, description, error_message,
			   created_by, retention_policy
		FROM backup_metadata
		WHERE id = ?
	`
	
	metadata := &BackupMetadata{}
	var endTime sql.NullTime
	var duration sql.NullInt64
	var filePath, checksum, parentBackupID, databaseVersion, appVersion sql.NullString
	var tags, description, errorMessage, retentionPolicyJSON sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, backupID).Scan(
		&metadata.ID, &metadata.Type, &metadata.Status, &metadata.StartTime,
		&endTime, &duration, &metadata.Size, &metadata.CompressedSize,
		&metadata.CompressionType, &metadata.EncryptionType, &filePath,
		&checksum, &parentBackupID, &databaseVersion, &appVersion,
		&tags, &description, &errorMessage, &metadata.CreatedBy, &retentionPolicyJSON,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup not found: %s", backupID)
		}
		return nil, fmt.Errorf("failed to get backup record: %w", err)
	}
	
	// Handle nullable fields
	if endTime.Valid {
		metadata.EndTime = &endTime.Time
	}
	if duration.Valid {
		metadata.Duration = time.Duration(duration.Int64)
	}
	if filePath.Valid {
		metadata.FilePath = filePath.String
	}
	if checksum.Valid {
		metadata.Checksum = checksum.String
	}
	if parentBackupID.Valid {
		metadata.ParentBackupID = &parentBackupID.String
	}
	if databaseVersion.Valid {
		metadata.DatabaseVersion = databaseVersion.String
	}
	if appVersion.Valid {
		metadata.AppVersion = appVersion.String
	}
	if description.Valid {
		metadata.Description = description.String
	}
	if errorMessage.Valid {
		metadata.ErrorMessage = &errorMessage.String
	}
	
	// Deserialize JSON fields
	if tags.Valid && tags.String != "" {
		json.Unmarshal([]byte(tags.String), &metadata.Tags)
	}
	if retentionPolicyJSON.Valid && retentionPolicyJSON.String != "" {
		var policy RetentionPolicy
		if err := json.Unmarshal([]byte(retentionPolicyJSON.String), &policy); err == nil {
			metadata.RetentionPolicy = &policy
		}
	}
	
	return metadata, nil
}

// ListBackupRecords returns backup records with optional filtering
func (r *SQLBackupRepository) ListBackupRecords(ctx context.Context, filters BackupFilters) ([]*BackupMetadata, error) {
	query := `
		SELECT id, type, status, start_time, end_time, duration, size, compressed_size,
			   compression_type, encryption_type, file_path, checksum, parent_backup_id,
			   database_version, app_version, tags, description, error_message,
			   created_by, retention_policy
		FROM backup_metadata
	`
	
	var conditions []string
	var args []interface{}
	
	// Apply filters
	if filters.Type != nil {
		conditions = append(conditions, "type = ?")
		args = append(args, *filters.Type)
	}
	
	if filters.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filters.Status)
	}
	
	if filters.StartTime != nil {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, *filters.StartTime)
	}
	
	if filters.EndTime != nil {
		conditions = append(conditions, "start_time <= ?")
		args = append(args, *filters.EndTime)
	}
	
	if filters.CreatedBy != nil {
		conditions = append(conditions, "created_by = ?")
		args = append(args, *filters.CreatedBy)
	}
	
	// Build WHERE clause
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Add sorting
	sortBy := "start_time"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	
	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = strings.ToUpper(filters.SortOrder)
	}
	
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)
	
	// Add pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
		
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup records: %w", err)
	}
	defer rows.Close()
	
	var backups []*BackupMetadata
	
	for rows.Next() {
		metadata := &BackupMetadata{}
		var endTime sql.NullTime
		var duration sql.NullInt64
		var filePath, checksum, parentBackupID, databaseVersion, appVersion sql.NullString
		var tags, description, errorMessage, retentionPolicyJSON sql.NullString
		
		err := rows.Scan(
			&metadata.ID, &metadata.Type, &metadata.Status, &metadata.StartTime,
			&endTime, &duration, &metadata.Size, &metadata.CompressedSize,
			&metadata.CompressionType, &metadata.EncryptionType, &filePath,
			&checksum, &parentBackupID, &databaseVersion, &appVersion,
			&tags, &description, &errorMessage, &metadata.CreatedBy, &retentionPolicyJSON,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup record: %w", err)
		}
		
		// Handle nullable fields
		if endTime.Valid {
			metadata.EndTime = &endTime.Time
		}
		if duration.Valid {
			metadata.Duration = time.Duration(duration.Int64)
		}
		if filePath.Valid {
			metadata.FilePath = filePath.String
		}
		if checksum.Valid {
			metadata.Checksum = checksum.String
		}
		if parentBackupID.Valid {
			metadata.ParentBackupID = &parentBackupID.String
		}
		if databaseVersion.Valid {
			metadata.DatabaseVersion = databaseVersion.String
		}
		if appVersion.Valid {
			metadata.AppVersion = appVersion.String
		}
		if description.Valid {
			metadata.Description = description.String
		}
		if errorMessage.Valid {
			metadata.ErrorMessage = &errorMessage.String
		}
		
		// Deserialize JSON fields
		if tags.Valid && tags.String != "" {
			json.Unmarshal([]byte(tags.String), &metadata.Tags)
		}
		if retentionPolicyJSON.Valid && retentionPolicyJSON.String != "" {
			var policy RetentionPolicy
			if err := json.Unmarshal([]byte(retentionPolicyJSON.String), &policy); err == nil {
				metadata.RetentionPolicy = &policy
			}
		}
		
		backups = append(backups, metadata)
	}
	
	return backups, rows.Err()
}

// DeleteBackupRecord removes backup metadata
func (r *SQLBackupRepository) DeleteBackupRecord(ctx context.Context, backupID string) error {
	query := `DELETE FROM backup_metadata WHERE id = ?`
	
	result, err := r.db.ExecContext(ctx, query, backupID)
	if err != nil {
		return fmt.Errorf("failed to delete backup record: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("backup record not found: %s", backupID)
	}
	
	return nil
}

// UpdateBackupStatus updates the status of a backup operation
func (r *SQLBackupRepository) UpdateBackupStatus(ctx context.Context, backupID string, status BackupStatus, errorMsg *string) error {
	query := `
		UPDATE backup_metadata 
		SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query, status, errorMsg, backupID)
	if err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("backup record not found: %s", backupID)
	}
	
	return nil
}

// UpdateBackupProgress updates the progress of a backup operation
func (r *SQLBackupRepository) UpdateBackupProgress(ctx context.Context, progress *BackupProgress) error {
	query := `
		INSERT OR REPLACE INTO backup_progress (
			backup_id, status, percent_complete, bytes_processed, total_bytes,
			current_table, tables_complete, total_tables, estimated_time,
			error_message, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	var estimatedTime *int64
	if progress.EstimatedTime > 0 {
		et := int64(progress.EstimatedTime)
		estimatedTime = &et
	}
	
	_, err := r.db.ExecContext(ctx, query,
		progress.BackupID, progress.Status, progress.PercentComplete,
		progress.BytesProcessed, progress.TotalBytes, progress.CurrentTable,
		progress.TablesComplete, progress.TotalTables, estimatedTime,
		progress.ErrorMessage, progress.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update backup progress: %w", err)
	}
	
	return nil
}

// GetBackupProgress retrieves the current progress of a backup operation
func (r *SQLBackupRepository) GetBackupProgress(ctx context.Context, backupID string) (*BackupProgress, error) {
	query := `
		SELECT backup_id, status, percent_complete, bytes_processed, total_bytes,
			   current_table, tables_complete, total_tables, estimated_time,
			   error_message, updated_at
		FROM backup_progress
		WHERE backup_id = ?
	`
	
	progress := &BackupProgress{}
	var currentTable sql.NullString
	var estimatedTime sql.NullInt64
	var errorMessage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, backupID).Scan(
		&progress.BackupID, &progress.Status, &progress.PercentComplete,
		&progress.BytesProcessed, &progress.TotalBytes, &currentTable,
		&progress.TablesComplete, &progress.TotalTables, &estimatedTime,
		&errorMessage, &progress.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup progress not found: %s", backupID)
		}
		return nil, fmt.Errorf("failed to get backup progress: %w", err)
	}
	
	// Handle nullable fields
	if currentTable.Valid {
		progress.CurrentTable = currentTable.String
	}
	if estimatedTime.Valid {
		progress.EstimatedTime = time.Duration(estimatedTime.Int64)
	}
	if errorMessage.Valid {
		progress.ErrorMessage = &errorMessage.String
	}
	
	return progress, nil
}

// CreateSchedule creates a new backup schedule
func (r *SQLBackupRepository) CreateSchedule(ctx context.Context, schedule *BackupSchedule) error {
	query := `
		INSERT INTO backup_schedules (
			id, name, description, cron_expression, backup_options,
			retention_policy, enabled, last_run, next_run, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	// Serialize JSON fields
	optionsJSON, err := json.Marshal(schedule.BackupOptions)
	if err != nil {
		return fmt.Errorf("failed to serialize backup options: %w", err)
	}
	
	retentionJSON, err := json.Marshal(schedule.RetentionPolicy)
	if err != nil {
		return fmt.Errorf("failed to serialize retention policy: %w", err)
	}
	
	_, err = r.db.ExecContext(ctx, query,
		schedule.ID, schedule.Name, schedule.Description, schedule.CronExpression,
		string(optionsJSON), string(retentionJSON), schedule.Enabled,
		schedule.LastRun, schedule.NextRun, schedule.CreatedBy,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create backup schedule: %w", err)
	}
	
	return nil
}

// UpdateSchedule updates an existing backup schedule
func (r *SQLBackupRepository) UpdateSchedule(ctx context.Context, schedule *BackupSchedule) error {
	query := `
		UPDATE backup_schedules SET
			name = ?, description = ?, cron_expression = ?, backup_options = ?,
			retention_policy = ?, enabled = ?, last_run = ?, next_run = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	// Serialize JSON fields
	optionsJSON, err := json.Marshal(schedule.BackupOptions)
	if err != nil {
		return fmt.Errorf("failed to serialize backup options: %w", err)
	}
	
	retentionJSON, err := json.Marshal(schedule.RetentionPolicy)
	if err != nil {
		return fmt.Errorf("failed to serialize retention policy: %w", err)
	}
	
	result, err := r.db.ExecContext(ctx, query,
		schedule.Name, schedule.Description, schedule.CronExpression,
		string(optionsJSON), string(retentionJSON), schedule.Enabled,
		schedule.LastRun, schedule.NextRun, schedule.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update backup schedule: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("backup schedule not found: %s", schedule.ID)
	}
	
	return nil
}

// GetSchedule retrieves a specific backup schedule
func (r *SQLBackupRepository) GetSchedule(ctx context.Context, scheduleID string) (*BackupSchedule, error) {
	query := `
		SELECT id, name, description, cron_expression, backup_options,
			   retention_policy, enabled, last_run, next_run,
			   created_at, updated_at, created_by
		FROM backup_schedules
		WHERE id = ?
	`
	
	schedule := &BackupSchedule{}
	var description sql.NullString
	var lastRun, nextRun sql.NullTime
	var optionsJSON, retentionJSON string
	
	err := r.db.QueryRowContext(ctx, query, scheduleID).Scan(
		&schedule.ID, &schedule.Name, &description, &schedule.CronExpression,
		&optionsJSON, &retentionJSON, &schedule.Enabled, &lastRun, &nextRun,
		&schedule.CreatedAt, &schedule.UpdatedAt, &schedule.CreatedBy,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup schedule not found: %s", scheduleID)
		}
		return nil, fmt.Errorf("failed to get backup schedule: %w", err)
	}
	
	// Handle nullable fields
	if description.Valid {
		schedule.Description = description.String
	}
	if lastRun.Valid {
		schedule.LastRun = &lastRun.Time
	}
	if nextRun.Valid {
		schedule.NextRun = &nextRun.Time
	}
	
	// Deserialize JSON fields
	if err := json.Unmarshal([]byte(optionsJSON), &schedule.BackupOptions); err != nil {
		return nil, fmt.Errorf("failed to deserialize backup options: %w", err)
	}
	
	if err := json.Unmarshal([]byte(retentionJSON), &schedule.RetentionPolicy); err != nil {
		return nil, fmt.Errorf("failed to deserialize retention policy: %w", err)
	}
	
	return schedule, nil
}

// ListSchedules returns all backup schedules
func (r *SQLBackupRepository) ListSchedules(ctx context.Context) ([]*BackupSchedule, error) {
	query := `
		SELECT id, name, description, cron_expression, backup_options,
			   retention_policy, enabled, last_run, next_run,
			   created_at, updated_at, created_by
		FROM backup_schedules
		ORDER BY name
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup schedules: %w", err)
	}
	defer rows.Close()
	
	var schedules []*BackupSchedule
	
	for rows.Next() {
		schedule := &BackupSchedule{}
		var description sql.NullString
		var lastRun, nextRun sql.NullTime
		var optionsJSON, retentionJSON string
		
		err := rows.Scan(
			&schedule.ID, &schedule.Name, &description, &schedule.CronExpression,
			&optionsJSON, &retentionJSON, &schedule.Enabled, &lastRun, &nextRun,
			&schedule.CreatedAt, &schedule.UpdatedAt, &schedule.CreatedBy,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup schedule: %w", err)
		}
		
		// Handle nullable fields
		if description.Valid {
			schedule.Description = description.String
		}
		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}
		if nextRun.Valid {
			schedule.NextRun = &nextRun.Time
		}
		
		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(optionsJSON), &schedule.BackupOptions); err != nil {
			return nil, fmt.Errorf("failed to deserialize backup options: %w", err)
		}
		
		if err := json.Unmarshal([]byte(retentionJSON), &schedule.RetentionPolicy); err != nil {
			return nil, fmt.Errorf("failed to deserialize retention policy: %w", err)
		}
		
		schedules = append(schedules, schedule)
	}
	
	return schedules, rows.Err()
}

// DeleteSchedule removes a backup schedule
func (r *SQLBackupRepository) DeleteSchedule(ctx context.Context, scheduleID string) error {
	query := `DELETE FROM backup_schedules WHERE id = ?`
	
	result, err := r.db.ExecContext(ctx, query, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to delete backup schedule: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("backup schedule not found: %s", scheduleID)
	}
	
	return nil
}