package backup

import (
	"context"
	"io"
	"time"
)

// BackupType represents different types of backups
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeDifferential BackupType = "differential"
	BackupTypeConfiguration BackupType = "configuration"
)

// BackupStatus represents the status of a backup operation
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusRunning    BackupStatus = "running"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
	BackupStatusCancelled  BackupStatus = "cancelled"
	BackupStatusCorrupted  BackupStatus = "corrupted"
)

// CompressionType represents different compression algorithms
type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionZlib CompressionType = "zlib"
	CompressionLz4  CompressionType = "lz4"
)

// EncryptionType represents different encryption algorithms
type EncryptionType string

const (
	EncryptionNone   EncryptionType = "none"
	EncryptionAES256 EncryptionType = "aes256"
	EncryptionChaCha EncryptionType = "chacha20"
)

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	ID              string           `json:"id"`
	Type            BackupType       `json:"type"`
	Status          BackupStatus     `json:"status"`
	StartTime       time.Time        `json:"start_time"`
	EndTime         *time.Time       `json:"end_time,omitempty"`
	Duration        time.Duration    `json:"duration"`
	Size            int64            `json:"size"`
	CompressedSize  int64            `json:"compressed_size"`
	CompressionType CompressionType  `json:"compression_type"`
	EncryptionType  EncryptionType   `json:"encryption_type"`
	FilePath        string           `json:"file_path"`
	Checksum        string           `json:"checksum"`
	ParentBackupID  *string          `json:"parent_backup_id,omitempty"`
	DatabaseVersion string           `json:"database_version"`
	AppVersion      string           `json:"app_version"`
	Tags            []string         `json:"tags"`
	Description     string           `json:"description"`
	ErrorMessage    *string          `json:"error_message,omitempty"`
	CreatedBy       string           `json:"created_by"`
	RetentionPolicy *RetentionPolicy `json:"retention_policy,omitempty"`
}

// RetentionPolicy defines how long backups should be kept
type RetentionPolicy struct {
	DailyRetention   int `json:"daily_retention"`   // Days to keep daily backups
	WeeklyRetention  int `json:"weekly_retention"`  // Weeks to keep weekly backups
	MonthlyRetention int `json:"monthly_retention"` // Months to keep monthly backups
	YearlyRetention  int `json:"yearly_retention"`  // Years to keep yearly backups
}

// BackupOptions contains configuration for backup operations
type BackupOptions struct {
	Type            BackupType       `json:"type"`
	CompressionType CompressionType  `json:"compression_type"`
	EncryptionType  EncryptionType   `json:"encryption_type"`
	EncryptionKey   string           `json:"-"` // Not serialized for security
	FilePath        string           `json:"file_path"`
	Tags            []string         `json:"tags"`
	Description     string           `json:"description"`
	IncludeTables   []string         `json:"include_tables,omitempty"`
	ExcludeTables   []string         `json:"exclude_tables,omitempty"`
	ParentBackupID  *string          `json:"parent_backup_id,omitempty"`
	Parallel        bool             `json:"parallel"`
	Verify          bool             `json:"verify"`
	RetentionPolicy *RetentionPolicy `json:"retention_policy,omitempty"`
}

// RestoreOptions contains configuration for restore operations
type RestoreOptions struct {
	BackupID        string            `json:"backup_id"`
	TargetTime      *time.Time        `json:"target_time,omitempty"`
	TargetPath      string            `json:"target_path,omitempty"`
	IncludeTables   []string          `json:"include_tables,omitempty"`
	ExcludeTables   []string          `json:"exclude_tables,omitempty"`
	Overwrite       bool              `json:"overwrite"`
	Parallel        bool              `json:"parallel"`
	Verify          bool              `json:"verify"`
	DryRun          bool              `json:"dry_run"`
	CustomMappings  map[string]string `json:"custom_mappings,omitempty"`
}

// BackupProgress represents the progress of a backup/restore operation
type BackupProgress struct {
	BackupID       string        `json:"backup_id"`
	Status         BackupStatus  `json:"status"`
	PercentComplete float64      `json:"percent_complete"`
	BytesProcessed int64         `json:"bytes_processed"`
	TotalBytes     int64         `json:"total_bytes"`
	CurrentTable   string        `json:"current_table,omitempty"`
	TablesComplete int           `json:"tables_complete"`
	TotalTables    int           `json:"total_tables"`
	EstimatedTime  time.Duration `json:"estimated_time"`
	ErrorMessage   *string       `json:"error_message,omitempty"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// BackupSchedule represents a scheduled backup configuration
type BackupSchedule struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	CronExpression  string           `json:"cron_expression"`
	BackupOptions   BackupOptions    `json:"backup_options"`
	RetentionPolicy RetentionPolicy  `json:"retention_policy"`
	Enabled         bool             `json:"enabled"`
	LastRun         *time.Time       `json:"last_run,omitempty"`
	NextRun         *time.Time       `json:"next_run,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CreatedBy       string           `json:"created_by"`
}

// BackupEngine defines the interface for backup implementations
type BackupEngine interface {
	// CreateBackup creates a new backup with the given options
	CreateBackup(ctx context.Context, options BackupOptions) (*BackupMetadata, error)
	
	// RestoreBackup restores from a backup with the given options
	RestoreBackup(ctx context.Context, options RestoreOptions) error
	
	// GetBackupMetadata retrieves metadata for a specific backup
	GetBackupMetadata(ctx context.Context, backupID string) (*BackupMetadata, error)
	
	// ListBackups returns a list of available backups with optional filtering
	ListBackups(ctx context.Context, filters BackupFilters) ([]*BackupMetadata, error)
	
	// DeleteBackup removes a backup and its associated files
	DeleteBackup(ctx context.Context, backupID string) error
	
	// ValidateBackup verifies the integrity of a backup
	ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error)
	
	// GetProgress returns the current progress of a running operation
	GetProgress(ctx context.Context, backupID string) (*BackupProgress, error)
	
	// CancelOperation cancels a running backup or restore operation
	CancelOperation(ctx context.Context, backupID string) error
}

// BackupRepository defines the interface for backup metadata storage
type BackupRepository interface {
	// CreateBackupRecord saves backup metadata to storage
	CreateBackupRecord(ctx context.Context, metadata *BackupMetadata) error
	
	// UpdateBackupRecord updates existing backup metadata
	UpdateBackupRecord(ctx context.Context, metadata *BackupMetadata) error
	
	// GetBackupRecord retrieves backup metadata by ID
	GetBackupRecord(ctx context.Context, backupID string) (*BackupMetadata, error)
	
	// ListBackupRecords returns backup records with optional filtering
	ListBackupRecords(ctx context.Context, filters BackupFilters) ([]*BackupMetadata, error)
	
	// DeleteBackupRecord removes backup metadata
	DeleteBackupRecord(ctx context.Context, backupID string) error
	
	// UpdateBackupStatus updates the status of a backup operation
	UpdateBackupStatus(ctx context.Context, backupID string, status BackupStatus, errorMsg *string) error
	
	// UpdateBackupProgress updates the progress of a backup operation
	UpdateBackupProgress(ctx context.Context, progress *BackupProgress) error
	
	// GetBackupProgress retrieves the current progress of a backup operation
	GetBackupProgress(ctx context.Context, backupID string) (*BackupProgress, error)
	
	// Schedule management methods
	CreateSchedule(ctx context.Context, schedule *BackupSchedule) error
	UpdateSchedule(ctx context.Context, schedule *BackupSchedule) error
	DeleteSchedule(ctx context.Context, scheduleID string) error
	GetSchedule(ctx context.Context, scheduleID string) (*BackupSchedule, error)
	ListSchedules(ctx context.Context) ([]*BackupSchedule, error)
}

// BackupScheduler defines the interface for backup scheduling
type BackupScheduler interface {
	// CreateSchedule creates a new backup schedule
	CreateSchedule(ctx context.Context, schedule *BackupSchedule) error
	
	// UpdateSchedule updates an existing backup schedule
	UpdateSchedule(ctx context.Context, schedule *BackupSchedule) error
	
	// DeleteSchedule removes a backup schedule
	DeleteSchedule(ctx context.Context, scheduleID string) error
	
	// GetSchedule retrieves a specific backup schedule
	GetSchedule(ctx context.Context, scheduleID string) (*BackupSchedule, error)
	
	// ListSchedules returns all backup schedules
	ListSchedules(ctx context.Context) ([]*BackupSchedule, error)
	
	// EnableSchedule enables a backup schedule
	EnableSchedule(ctx context.Context, scheduleID string) error
	
	// DisableSchedule disables a backup schedule
	DisableSchedule(ctx context.Context, scheduleID string) error
	
	// Start begins the scheduler
	Start(ctx context.Context) error
	
	// Stop shuts down the scheduler
	Stop(ctx context.Context) error
}

// BackupStorage defines the interface for backup file storage
type BackupStorage interface {
	// Store saves backup data to storage and returns the path
	Store(ctx context.Context, backupID string, data io.Reader) (string, error)
	
	// Retrieve gets backup data from storage
	Retrieve(ctx context.Context, backupID string) (io.ReadCloser, error)
	
	// Delete removes backup data from storage
	Delete(ctx context.Context, backupID string) error
	
	// Exists checks if backup data exists in storage
	Exists(ctx context.Context, backupID string) (bool, error)
	
	// GetSize returns the size of backup data in storage
	GetSize(ctx context.Context, backupID string) (int64, error)
	
	// ListBackups returns a list of backup IDs in storage
	ListBackups(ctx context.Context) ([]string, error)
}

// BackupFilters contains options for filtering backup lists
type BackupFilters struct {
	Type       *BackupType   `json:"type,omitempty"`
	Status     *BackupStatus `json:"status,omitempty"`
	StartTime  *time.Time    `json:"start_time,omitempty"`
	EndTime    *time.Time    `json:"end_time,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	CreatedBy  *string       `json:"created_by,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
	SortBy     string        `json:"sort_by,omitempty"`
	SortOrder  string        `json:"sort_order,omitempty"`
}

// ValidationResult contains the result of backup validation
type ValidationResult struct {
	BackupID     string            `json:"backup_id"`
	IsValid      bool              `json:"is_valid"`
	ChecksumMatch bool             `json:"checksum_match"`
	Errors       []string          `json:"errors,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	TableCounts  map[string]int64  `json:"table_counts,omitempty"`
	ValidationTime time.Duration   `json:"validation_time"`
	ValidatedAt    time.Time       `json:"validated_at"`
}

// BackupManager is the main interface for backup management
type BackupManager interface {
	BackupEngine
	
	// Schedule management
	CreateSchedule(ctx context.Context, schedule *BackupSchedule) error
	GetSchedule(ctx context.Context, scheduleID string) (*BackupSchedule, error)
	ListSchedules(ctx context.Context) ([]*BackupSchedule, error)
	UpdateSchedule(ctx context.Context, schedule *BackupSchedule) error
	DeleteSchedule(ctx context.Context, scheduleID string) error
	
	// Retention management
	ApplyRetentionPolicy(ctx context.Context, policy RetentionPolicy) error
	CleanupExpiredBackups(ctx context.Context) error
	
	// Health and monitoring
	GetHealthStatus(ctx context.Context) (*HealthStatus, error)
	GetStatistics(ctx context.Context) (*BackupStatistics, error)
	
	// Configuration
	UpdateConfiguration(ctx context.Context, config *BackupConfiguration) error
	GetConfiguration(ctx context.Context) (*BackupConfiguration, error)
	
	// Start and stop the backup manager
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HealthStatus represents the health of the backup system
type HealthStatus struct {
	IsHealthy          bool              `json:"is_healthy"`
	LastBackupTime     *time.Time        `json:"last_backup_time,omitempty"`
	NextScheduledTime  *time.Time        `json:"next_scheduled_time,omitempty"`
	RunningOperations  int               `json:"running_operations"`
	FailedOperations   int               `json:"failed_operations"`
	StorageUsed        int64             `json:"storage_used"`
	StorageAvailable   int64             `json:"storage_available"`
	Issues             []string          `json:"issues,omitempty"`
	CheckedAt          time.Time         `json:"checked_at"`
}

// BackupStatistics contains statistics about backup operations
type BackupStatistics struct {
	TotalBackups       int64             `json:"total_backups"`
	SuccessfulBackups  int64             `json:"successful_backups"`
	FailedBackups      int64             `json:"failed_backups"`
	TotalStorageUsed   int64             `json:"total_storage_used"`
	AverageBackupSize  int64             `json:"average_backup_size"`
	AverageBackupTime  time.Duration     `json:"average_backup_time"`
	BackupsByType      map[BackupType]int64 `json:"backups_by_type"`
	BackupsByStatus    map[BackupStatus]int64 `json:"backups_by_status"`
	LastPeriodStats    *PeriodStatistics `json:"last_period_stats,omitempty"`
}

// PeriodStatistics contains statistics for a specific time period
type PeriodStatistics struct {
	Period        string        `json:"period"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	BackupCount   int64         `json:"backup_count"`
	SuccessRate   float64       `json:"success_rate"`
	TotalSize     int64         `json:"total_size"`
	AverageTime   time.Duration `json:"average_time"`
}

// BackupConfiguration contains global backup system configuration
type BackupConfiguration struct {
	DefaultCompressionType CompressionType  `json:"default_compression_type"`
	DefaultEncryptionType  EncryptionType   `json:"default_encryption_type"`
	DefaultRetentionPolicy RetentionPolicy  `json:"default_retention_policy"`
	MaxConcurrentBackups   int              `json:"max_concurrent_backups"`
	BackupStoragePath      string           `json:"backup_storage_path"`
	TempStoragePath        string           `json:"temp_storage_path"`
	EnableCompression      bool             `json:"enable_compression"`
	EnableEncryption       bool             `json:"enable_encryption"`
	EnableVerification     bool             `json:"enable_verification"`
	AlertingEnabled        bool             `json:"alerting_enabled"`
	MonitoringEnabled      bool             `json:"monitoring_enabled"`
	UpdatedAt              time.Time        `json:"updated_at"`
	UpdatedBy              string           `json:"updated_by"`
}