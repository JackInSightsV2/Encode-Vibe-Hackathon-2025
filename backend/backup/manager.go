package backup

import (
	"context"
	"fmt"
	"qt1-middleware/database"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultBackupManager implements the BackupManager interface
type DefaultBackupManager struct {
	db               *database.Database
	storage          BackupStorage
	repository       BackupRepository
	backupEngine     BackupEngine
	restoreEngine    *RestoreEngine
	validator        *BackupValidator
	scheduler        BackupScheduler
	config           *BackupConfiguration
	
	// Runtime state
	isRunning        bool
	runningOps       map[string]context.CancelFunc
	mutex            sync.RWMutex
	
	// Channels for coordination
	stopChan         chan struct{}
	schedulerDone    chan struct{}
}

// NewBackupManager creates a new backup manager
func NewBackupManager(db *database.Database, storagePath string) (*DefaultBackupManager, error) {
	// Initialize storage
	storage, err := NewFileSystemBackupStorage(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	
	// Initialize repository
	repository := NewSQLBackupRepository(db.DB)
	if err := repository.InitializeSchema(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize backup schema: %w", err)
	}
	
	// Initialize components
	backupEngine := NewDatabaseBackupEngine(db, storage, repository)
	restoreEngine := NewRestoreEngine(db, storage, repository)
	validator := NewBackupValidator(db, storage, repository)
	scheduler := NewCronBackupScheduler(repository, backupEngine)
	
	// Default configuration
	config := &BackupConfiguration{
		DefaultCompressionType: CompressionGzip,
		DefaultEncryptionType:  EncryptionNone,
		DefaultRetentionPolicy: RetentionPolicy{
			DailyRetention:   7,
			WeeklyRetention:  4,
			MonthlyRetention: 12,
			YearlyRetention:  5,
		},
		MaxConcurrentBackups: 3,
		BackupStoragePath:    storagePath,
		TempStoragePath:      "/tmp/backups",
		EnableCompression:    true,
		EnableEncryption:     false,
		EnableVerification:   true,
		AlertingEnabled:      false,
		MonitoringEnabled:    true,
		UpdatedAt:           time.Now(),
		UpdatedBy:           "system",
	}
	
	manager := &DefaultBackupManager{
		db:            db,
		storage:       storage,
		repository:    repository,
		backupEngine:  backupEngine,
		restoreEngine: restoreEngine,
		validator:     validator,
		scheduler:     scheduler,
		config:        config,
		runningOps:    make(map[string]context.CancelFunc),
		stopChan:      make(chan struct{}),
		schedulerDone: make(chan struct{}),
	}
	
	return manager, nil
}

// Start starts the backup manager
func (m *DefaultBackupManager) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.isRunning {
		return fmt.Errorf("backup manager is already running")
	}
	
	// Start the scheduler
	if err := m.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backup scheduler: %w", err)
	}
	
	// Start background tasks
	go m.runBackgroundTasks()
	
	m.isRunning = true
	return nil
}

// Stop stops the backup manager
func (m *DefaultBackupManager) Stop(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.isRunning {
		return nil
	}
	
	// Stop the scheduler
	if err := m.scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop backup scheduler: %w", err)
	}
	
	// Cancel all running operations
	for opID, cancel := range m.runningOps {
		cancel()
		delete(m.runningOps, opID)
	}
	
	// Stop background tasks
	close(m.stopChan)
	
	// Wait for scheduler to finish
	select {
	case <-m.schedulerDone:
	case <-time.After(30 * time.Second):
		// Timeout waiting for scheduler
	}
	
	m.isRunning = false
	return nil
}

// runBackgroundTasks runs background maintenance tasks
func (m *DefaultBackupManager) runBackgroundTasks() {
	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Run maintenance tasks
			ctx := context.Background()
			
			// Clean up expired backups
			if err := m.CleanupExpiredBackups(ctx); err != nil {
				// Log error (in production, use proper logging)
				fmt.Printf("Failed to cleanup expired backups: %v\n", err)
			}
			
			// Validate recent backups
			m.validateRecentBackups(ctx)
			
		case <-m.stopChan:
			close(m.schedulerDone)
			return
		}
	}
}

// validateRecentBackups validates backups created in the last 24 hours
func (m *DefaultBackupManager) validateRecentBackups(ctx context.Context) {
	yesterday := time.Now().Add(-24 * time.Hour)
	
	filters := BackupFilters{
		Status:    &[]BackupStatus{BackupStatusCompleted}[0],
		StartTime: &yesterday,
		Limit:     10,
	}
	
	backups, err := m.repository.ListBackupRecords(ctx, filters)
	if err != nil {
		fmt.Printf("Failed to list recent backups for validation: %v\n", err)
		return
	}
	
	for _, backup := range backups {
		// Quick validation only
		result, err := m.validator.QuickValidateBackup(ctx, backup.ID)
		if err != nil {
			fmt.Printf("Failed to validate backup %s: %v\n", backup.ID, err)
			continue
		}
		
		if !result.IsValid {
			fmt.Printf("Backup %s validation failed: %v\n", backup.ID, result.Errors)
		}
	}
}

// BackupEngine interface methods

// CreateBackup creates a new backup with the given options
func (m *DefaultBackupManager) CreateBackup(ctx context.Context, options BackupOptions) (*BackupMetadata, error) {
	// Apply default configuration
	if options.CompressionType == "" {
		options.CompressionType = m.config.DefaultCompressionType
	}
	if options.EncryptionType == "" {
		options.EncryptionType = m.config.DefaultEncryptionType
	}
	if options.RetentionPolicy == nil {
		options.RetentionPolicy = &m.config.DefaultRetentionPolicy
	}
	
	// Check concurrent backup limit
	m.mutex.RLock()
	runningCount := len(m.runningOps)
	m.mutex.RUnlock()
	
	if runningCount >= m.config.MaxConcurrentBackups {
		return nil, fmt.Errorf("maximum concurrent backups (%d) reached", m.config.MaxConcurrentBackups)
	}
	
	// Create cancellable context for this operation
	operationCtx, cancel := context.WithCancel(ctx)
	
	// Generate operation ID
	operationID := uuid.New().String()
	
	// Track running operation
	m.mutex.Lock()
	m.runningOps[operationID] = cancel
	m.mutex.Unlock()
	
	// Ensure cleanup
	defer func() {
		m.mutex.Lock()
		delete(m.runningOps, operationID)
		m.mutex.Unlock()
	}()
	
	// Create backup
	metadata, err := m.backupEngine.CreateBackup(operationCtx, options)
	if err != nil {
		return nil, err
	}
	
	// Validate backup if enabled
	if m.config.EnableVerification && options.Verify {
		go func() {
			// Validate after backup completes
			time.Sleep(5 * time.Second) // Give backup time to complete
			
			result, err := m.validator.ValidateBackup(context.Background(), metadata.ID)
			if err != nil {
				fmt.Printf("Failed to validate backup %s: %v\n", metadata.ID, err)
			} else if !result.IsValid {
				fmt.Printf("Backup %s validation failed: %v\n", metadata.ID, result.Errors)
			}
		}()
	}
	
	return metadata, nil
}

// RestoreBackup restores from a backup with the given options
func (m *DefaultBackupManager) RestoreBackup(ctx context.Context, options RestoreOptions) error {
	return m.restoreEngine.RestoreBackup(ctx, options)
}

// GetBackupMetadata retrieves metadata for a specific backup
func (m *DefaultBackupManager) GetBackupMetadata(ctx context.Context, backupID string) (*BackupMetadata, error) {
	return m.backupEngine.GetBackupMetadata(ctx, backupID)
}

// ListBackups returns a list of available backups with optional filtering
func (m *DefaultBackupManager) ListBackups(ctx context.Context, filters BackupFilters) ([]*BackupMetadata, error) {
	return m.backupEngine.ListBackups(ctx, filters)
}

// DeleteBackup removes a backup and its associated files
func (m *DefaultBackupManager) DeleteBackup(ctx context.Context, backupID string) error {
	return m.backupEngine.DeleteBackup(ctx, backupID)
}

// ValidateBackup verifies the integrity of a backup
func (m *DefaultBackupManager) ValidateBackup(ctx context.Context, backupID string) (*ValidationResult, error) {
	return m.validator.ValidateBackup(ctx, backupID)
}

// GetProgress returns the current progress of a running operation
func (m *DefaultBackupManager) GetProgress(ctx context.Context, backupID string) (*BackupProgress, error) {
	return m.backupEngine.GetProgress(ctx, backupID)
}

// CancelOperation cancels a running backup or restore operation
func (m *DefaultBackupManager) CancelOperation(ctx context.Context, backupID string) error {
	m.mutex.RLock()
	cancel, exists := m.runningOps[backupID]
	m.mutex.RUnlock()
	
	if exists {
		cancel()
	}
	
	return m.backupEngine.CancelOperation(ctx, backupID)
}

// Schedule management methods

// CreateSchedule creates a new backup schedule
func (m *DefaultBackupManager) CreateSchedule(ctx context.Context, schedule *BackupSchedule) error {
	return m.scheduler.CreateSchedule(ctx, schedule)
}

// GetSchedule retrieves a specific backup schedule
func (m *DefaultBackupManager) GetSchedule(ctx context.Context, scheduleID string) (*BackupSchedule, error) {
	return m.scheduler.GetSchedule(ctx, scheduleID)
}

// ListSchedules returns all backup schedules
func (m *DefaultBackupManager) ListSchedules(ctx context.Context) ([]*BackupSchedule, error) {
	return m.scheduler.ListSchedules(ctx)
}

// UpdateSchedule updates an existing backup schedule
func (m *DefaultBackupManager) UpdateSchedule(ctx context.Context, schedule *BackupSchedule) error {
	return m.scheduler.UpdateSchedule(ctx, schedule)
}

// DeleteSchedule removes a backup schedule
func (m *DefaultBackupManager) DeleteSchedule(ctx context.Context, scheduleID string) error {
	return m.scheduler.DeleteSchedule(ctx, scheduleID)
}

// Retention management

// ApplyRetentionPolicy applies retention policy to existing backups
func (m *DefaultBackupManager) ApplyRetentionPolicy(ctx context.Context, policy RetentionPolicy) error {
	// Get all completed backups, sorted by creation time
	filters := BackupFilters{
		Status:    &[]BackupStatus{BackupStatusCompleted}[0],
		SortBy:    "start_time",
		SortOrder: "desc",
	}
	
	backups, err := m.repository.ListBackupRecords(ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}
	
	now := time.Now()
	
	for _, backup := range backups {
		age := now.Sub(backup.StartTime)
		shouldDelete := false
		
		// Apply retention rules
		if age > time.Duration(policy.DailyRetention)*24*time.Hour {
			// Check if it's a weekly backup
			if backup.StartTime.Weekday() == time.Sunday {
				if age > time.Duration(policy.WeeklyRetention)*7*24*time.Hour {
					// Check if it's a monthly backup
					if backup.StartTime.Day() == 1 {
						if age > time.Duration(policy.MonthlyRetention)*30*24*time.Hour {
							// Check if it's a yearly backup
							if backup.StartTime.Month() == time.January && backup.StartTime.Day() == 1 {
								if age > time.Duration(policy.YearlyRetention)*365*24*time.Hour {
									shouldDelete = true
								}
							} else {
								shouldDelete = true
							}
						}
					} else {
						shouldDelete = true
					}
				}
			} else {
				shouldDelete = true
			}
		}
		
		if shouldDelete {
			if err := m.DeleteBackup(ctx, backup.ID); err != nil {
				fmt.Printf("Failed to delete expired backup %s: %v\n", backup.ID, err)
			}
		}
	}
	
	return nil
}

// CleanupExpiredBackups removes backups that have exceeded their retention period
func (m *DefaultBackupManager) CleanupExpiredBackups(ctx context.Context) error {
	return m.ApplyRetentionPolicy(ctx, m.config.DefaultRetentionPolicy)
}

// Health and monitoring

// GetHealthStatus returns the health status of the backup system
func (m *DefaultBackupManager) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		IsHealthy:         true,
		RunningOperations: len(m.runningOps),
		CheckedAt:         time.Now(),
	}
	
	// Check for recent successful backups
	yesterday := time.Now().Add(-24 * time.Hour)
	filters := BackupFilters{
		Status:    &[]BackupStatus{BackupStatusCompleted}[0],
		StartTime: &yesterday,
		Limit:     1,
	}
	
	recentBackups, err := m.repository.ListBackupRecords(ctx, filters)
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to check recent backups: %v", err))
		status.IsHealthy = false
	} else if len(recentBackups) > 0 {
		status.LastBackupTime = &recentBackups[0].StartTime
	} else {
		status.Issues = append(status.Issues, "No successful backups in the last 24 hours")
		status.IsHealthy = false
	}
	
	// Check for failed operations
	failedFilters := BackupFilters{
		Status:    &[]BackupStatus{BackupStatusFailed}[0],
		StartTime: &yesterday,
	}
	
	failedBackups, err := m.repository.ListBackupRecords(ctx, failedFilters)
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to check failed backups: %v", err))
	} else {
		status.FailedOperations = len(failedBackups)
		if status.FailedOperations > 0 {
			status.Issues = append(status.Issues, fmt.Sprintf("%d failed backup operations in the last 24 hours", status.FailedOperations))
		}
	}
	
	// Check next scheduled backup
	schedules, err := m.scheduler.ListSchedules(ctx)
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Failed to check schedules: %v", err))
	} else {
		var nextRun *time.Time
		for _, schedule := range schedules {
			if schedule.Enabled && schedule.NextRun != nil {
				if nextRun == nil || schedule.NextRun.Before(*nextRun) {
					nextRun = schedule.NextRun
				}
			}
		}
		status.NextScheduledTime = nextRun
	}
	
	// TODO: Check storage usage
	
	return status, nil
}

// GetStatistics returns backup statistics
func (m *DefaultBackupManager) GetStatistics(ctx context.Context) (*BackupStatistics, error) {
	stats := &BackupStatistics{
		BackupsByType:   make(map[BackupType]int64),
		BackupsByStatus: make(map[BackupStatus]int64),
	}
	
	// Get all backups
	allBackups, err := m.repository.ListBackupRecords(ctx, BackupFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to get backup statistics: %w", err)
	}
	
	var totalSize int64
	var totalDuration time.Duration
	var completedCount int64
	
	for _, backup := range allBackups {
		stats.TotalBackups++
		stats.BackupsByType[backup.Type]++
		stats.BackupsByStatus[backup.Status]++
		
		if backup.Status == BackupStatusCompleted {
			completedCount++
			totalSize += backup.Size
			totalDuration += backup.Duration
		} else if backup.Status == BackupStatusFailed {
			stats.FailedBackups++
		}
	}
	
	stats.SuccessfulBackups = completedCount
	stats.TotalStorageUsed = totalSize
	
	if completedCount > 0 {
		stats.AverageBackupSize = totalSize / completedCount
		stats.AverageBackupTime = totalDuration / time.Duration(completedCount)
	}
	
	return stats, nil
}

// Configuration management

// UpdateConfiguration updates the backup system configuration
func (m *DefaultBackupManager) UpdateConfiguration(ctx context.Context, config *BackupConfiguration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	config.UpdatedAt = time.Now()
	m.config = config
	
	return nil
}

// GetConfiguration returns the current backup system configuration
func (m *DefaultBackupManager) GetConfiguration(ctx context.Context) (*BackupConfiguration, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	configCopy := *m.config
	return &configCopy, nil
}