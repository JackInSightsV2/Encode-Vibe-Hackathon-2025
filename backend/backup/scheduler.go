package backup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// CronBackupScheduler implements BackupScheduler using cron
type CronBackupScheduler struct {
	repository   BackupRepository
	backupEngine BackupEngine
	cron         *cron.Cron
	schedules    map[string]*BackupSchedule
	cronEntries  map[string]cron.EntryID
	mutex        sync.RWMutex
	isRunning    bool
}

// NewCronBackupScheduler creates a new cron-based backup scheduler
func NewCronBackupScheduler(repository BackupRepository, backupEngine BackupEngine) *CronBackupScheduler {
	return &CronBackupScheduler{
		repository:  repository,
		backupEngine: backupEngine,
		cron:        cron.New(cron.WithSeconds()), // Support seconds in cron expressions
		schedules:   make(map[string]*BackupSchedule),
		cronEntries: make(map[string]cron.EntryID),
	}
}

// Start begins the scheduler
func (s *CronBackupScheduler) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}
	
	// Load existing schedules from repository
	if err := s.loadSchedules(ctx); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}
	
	// Start the cron scheduler
	s.cron.Start()
	s.isRunning = true
	
	return nil
}

// Stop shuts down the scheduler
func (s *CronBackupScheduler) Stop(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.isRunning {
		return nil
	}
	
	// Stop the cron scheduler
	cronCtx := s.cron.Stop()
	
	// Wait for running jobs to complete (with timeout)
	select {
	case <-cronCtx.Done():
	case <-time.After(30 * time.Second):
		// Timeout - force stop
	}
	
	s.isRunning = false
	return nil
}

// loadSchedules loads all schedules from the repository
func (s *CronBackupScheduler) loadSchedules(ctx context.Context) error {
	schedules, err := s.repository.ListSchedules(ctx)
	if err != nil {
		return err
	}
	
	for _, schedule := range schedules {
		if schedule.Enabled {
			if err := s.addCronJob(schedule); err != nil {
				fmt.Printf("Failed to add cron job for schedule %s: %v\n", schedule.ID, err)
			}
		}
		s.schedules[schedule.ID] = schedule
	}
	
	return nil
}

// addCronJob adds a cron job for a schedule
func (s *CronBackupScheduler) addCronJob(schedule *BackupSchedule) error {
	job := &backupJob{
		schedule:     schedule,
		scheduler:    s,
		backupEngine: s.backupEngine,
		repository:   s.repository,
	}
	
	entryID, err := s.cron.AddJob(schedule.CronExpression, job)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}
	
	s.cronEntries[schedule.ID] = entryID
	
	// Update next run time
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	schedule.NextRun = &nextRun
	
	return nil
}

// removeCronJob removes a cron job for a schedule
func (s *CronBackupScheduler) removeCronJob(scheduleID string) {
	if entryID, exists := s.cronEntries[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.cronEntries, scheduleID)
	}
}

// CreateSchedule creates a new backup schedule
func (s *CronBackupScheduler) CreateSchedule(ctx context.Context, schedule *BackupSchedule) error {
	// Validate cron expression
	if _, err := cron.ParseStandard(schedule.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	
	// Generate ID if not provided
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}
	
	// Set timestamps
	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now
	
	// Save to repository
	if err := s.repository.CreateSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Add to in-memory map
	s.schedules[schedule.ID] = schedule
	
	// Add cron job if enabled and scheduler is running
	if schedule.Enabled && s.isRunning {
		if err := s.addCronJob(schedule); err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}
	}
	
	return nil
}

// UpdateSchedule updates an existing backup schedule
func (s *CronBackupScheduler) UpdateSchedule(ctx context.Context, schedule *BackupSchedule) error {
	// Validate cron expression
	if _, err := cron.ParseStandard(schedule.CronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if schedule exists
	existing, exists := s.schedules[schedule.ID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", schedule.ID)
	}
	
	// Update timestamps
	schedule.UpdatedAt = time.Now()
	schedule.CreatedAt = existing.CreatedAt // Preserve creation time
	
	// Save to repository
	if err := s.repository.UpdateSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}
	
	// Remove existing cron job
	s.removeCronJob(schedule.ID)
	
	// Update in-memory map
	s.schedules[schedule.ID] = schedule
	
	// Add new cron job if enabled and scheduler is running
	if schedule.Enabled && s.isRunning {
		if err := s.addCronJob(schedule); err != nil {
			return fmt.Errorf("failed to add updated cron job: %w", err)
		}
	}
	
	return nil
}

// DeleteSchedule removes a backup schedule
func (s *CronBackupScheduler) DeleteSchedule(ctx context.Context, scheduleID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Remove cron job
	s.removeCronJob(scheduleID)
	
	// Remove from in-memory map
	delete(s.schedules, scheduleID)
	
	// Delete from repository
	if err := s.repository.DeleteSchedule(ctx, scheduleID); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	
	return nil
}

// GetSchedule retrieves a specific backup schedule
func (s *CronBackupScheduler) GetSchedule(ctx context.Context, scheduleID string) (*BackupSchedule, error) {
	s.mutex.RLock()
	schedule, exists := s.schedules[scheduleID]
	s.mutex.RUnlock()
	
	if exists {
		// Return a copy to prevent external modification
		scheduleCopy := *schedule
		return &scheduleCopy, nil
	}
	
	// Try to load from repository
	return s.repository.GetSchedule(ctx, scheduleID)
}

// ListSchedules returns all backup schedules
func (s *CronBackupScheduler) ListSchedules(ctx context.Context) ([]*BackupSchedule, error) {
	return s.repository.ListSchedules(ctx)
}

// EnableSchedule enables a backup schedule
func (s *CronBackupScheduler) EnableSchedule(ctx context.Context, scheduleID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}
	
	if schedule.Enabled {
		return nil // Already enabled
	}
	
	// Update schedule
	schedule.Enabled = true
	schedule.UpdatedAt = time.Now()
	
	// Save to repository
	if err := s.repository.UpdateSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}
	
	// Add cron job if scheduler is running
	if s.isRunning {
		if err := s.addCronJob(schedule); err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}
	}
	
	return nil
}

// DisableSchedule disables a backup schedule
func (s *CronBackupScheduler) DisableSchedule(ctx context.Context, scheduleID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}
	
	if !schedule.Enabled {
		return nil // Already disabled
	}
	
	// Remove cron job
	s.removeCronJob(scheduleID)
	
	// Update schedule
	schedule.Enabled = false
	schedule.UpdatedAt = time.Now()
	schedule.NextRun = nil
	
	// Save to repository
	if err := s.repository.UpdateSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}
	
	return nil
}

// backupJob implements cron.Job interface for scheduled backups
type backupJob struct {
	schedule     *BackupSchedule
	scheduler    *CronBackupScheduler
	backupEngine BackupEngine
	repository   BackupRepository
}

// Run executes the scheduled backup job
func (j *backupJob) Run() {
	ctx := context.Background()
	
	fmt.Printf("Starting scheduled backup: %s (%s)\n", j.schedule.Name, j.schedule.ID)
	
	// Update last run time
	now := time.Now()
	j.schedule.LastRun = &now
	
	// Create backup with schedule options
	metadata, err := j.backupEngine.CreateBackup(ctx, j.schedule.BackupOptions)
	if err != nil {
		fmt.Printf("Scheduled backup failed for %s: %v\n", j.schedule.Name, err)
		return
	}
	
	fmt.Printf("Scheduled backup started: %s (ID: %s)\n", j.schedule.Name, metadata.ID)
	
	// Update next run time
	j.scheduler.mutex.Lock()
	if entryID, exists := j.scheduler.cronEntries[j.schedule.ID]; exists {
		entry := j.scheduler.cron.Entry(entryID)
		nextRun := entry.Next
		j.schedule.NextRun = &nextRun
	}
	j.scheduler.mutex.Unlock()
	
	// Save updated schedule
	if err := j.repository.UpdateSchedule(ctx, j.schedule); err != nil {
		fmt.Printf("Failed to update schedule after backup: %v\n", err)
	}
	
	// Apply retention policy after backup completes
	go j.applyRetentionPolicy(ctx, metadata.ID)
}

// applyRetentionPolicy applies the schedule's retention policy
func (j *backupJob) applyRetentionPolicy(ctx context.Context, backupID string) {
	// Wait for backup to complete
	for i := 0; i < 60; i++ { // Wait up to 1 hour
		progress, err := j.backupEngine.GetProgress(ctx, backupID)
		if err != nil {
			fmt.Printf("Failed to get backup progress: %v\n", err)
			return
		}
		
		if progress.Status == BackupStatusCompleted {
			break
		} else if progress.Status == BackupStatusFailed {
			return // Don't apply retention if backup failed
		}
		
		time.Sleep(1 * time.Minute)
	}
	
	// Get all backups and apply retention policy
	filters := BackupFilters{
		Status:    &[]BackupStatus{BackupStatusCompleted}[0],
		SortBy:    "start_time",
		SortOrder: "desc",
	}
	
	backups, err := j.repository.ListBackupRecords(ctx, filters)
	if err != nil {
		fmt.Printf("Failed to list backups for retention policy: %v\n", err)
		return
	}
	
	now := time.Now()
	policy := j.schedule.RetentionPolicy
	
	for _, backup := range backups {
		age := now.Sub(backup.StartTime)
		shouldDelete := false
		
		// Apply retention rules (simplified version)
		if age > time.Duration(policy.DailyRetention)*24*time.Hour {
			if backup.StartTime.Weekday() == time.Sunday {
				if age > time.Duration(policy.WeeklyRetention)*7*24*time.Hour {
					if backup.StartTime.Day() == 1 {
						if age > time.Duration(policy.MonthlyRetention)*30*24*time.Hour {
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
			if err := j.backupEngine.DeleteBackup(ctx, backup.ID); err != nil {
				fmt.Printf("Failed to delete expired backup %s: %v\n", backup.ID, err)
			} else {
				fmt.Printf("Deleted expired backup: %s\n", backup.ID)
			}
		}
	}
}

// Helper functions for creating common schedules

// CreateDailySchedule creates a schedule for daily backups
func CreateDailySchedule(name, description string, hour, minute int, options BackupOptions) *BackupSchedule {
	return &BackupSchedule{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		CronExpression: fmt.Sprintf("0 %d %d * * *", minute, hour), // Daily at specified time
		BackupOptions:  options,
		RetentionPolicy: RetentionPolicy{
			DailyRetention:   7,
			WeeklyRetention:  4,
			MonthlyRetention: 12,
			YearlyRetention:  5,
		},
		Enabled:   true,
		CreatedBy: "system",
	}
}

// CreateWeeklySchedule creates a schedule for weekly backups
func CreateWeeklySchedule(name, description string, weekday time.Weekday, hour, minute int, options BackupOptions) *BackupSchedule {
	return &BackupSchedule{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		CronExpression: fmt.Sprintf("0 %d %d * * %d", minute, hour, int(weekday)), // Weekly on specified day
		BackupOptions:  options,
		RetentionPolicy: RetentionPolicy{
			DailyRetention:   7,
			WeeklyRetention:  8,
			MonthlyRetention: 12,
			YearlyRetention:  5,
		},
		Enabled:   true,
		CreatedBy: "system",
	}
}

// CreateMonthlySchedule creates a schedule for monthly backups
func CreateMonthlySchedule(name, description string, day, hour, minute int, options BackupOptions) *BackupSchedule {
	return &BackupSchedule{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		CronExpression: fmt.Sprintf("0 %d %d %d * *", minute, hour, day), // Monthly on specified day
		BackupOptions:  options,
		RetentionPolicy: RetentionPolicy{
			DailyRetention:   7,
			WeeklyRetention:  4,
			MonthlyRetention: 24,
			YearlyRetention:  10,
		},
		Enabled:   true,
		CreatedBy: "system",
	}
}