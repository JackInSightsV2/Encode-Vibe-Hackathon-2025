package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// RestoreManager handles configuration restoration from backups
type RestoreManager struct {
	backupManager *BackupManager
	validator     *ConfigValidator
	merger        *ConfigMerger
	differ        *ConfigDiffer
}

// NewRestoreManager creates a new restore manager
func NewRestoreManager(backupDir string, validator *ConfigValidator) *RestoreManager {
	return &RestoreManager{
		backupManager: NewBackupManager(backupDir),
		validator:     validator,
		merger:        NewConfigMerger(validator),
		differ:        NewConfigDiffer(),
	}
}

// RestoreOptions defines options for restoration
type RestoreOptions struct {
	ValidateBeforeRestore bool                   `json:"validate_before_restore"`
	CreateBackupFirst     bool                   `json:"create_backup_first"`
	MergeWithCurrent      bool                   `json:"merge_with_current"`
	MergeOptions          *MergeOptions          `json:"merge_options,omitempty"`
	SelectiveSections     []string               `json:"selective_sections,omitempty"`
	ExcludeSections       []string               `json:"exclude_sections,omitempty"`
	DryRun                bool                   `json:"dry_run"`
	PreRestoreHooks       []func() error         `json:"-"`
	PostRestoreHooks      []func(*Config) error  `json:"-"`
}

// RestoreResult represents the result of a restore operation
type RestoreResult struct {
	Success          bool              `json:"success"`
	RestoredConfig   *Config           `json:"restored_config,omitempty"`
	BackupUsed       *ConfigBackup     `json:"backup_used"`
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`
	Changes          []ConfigChange    `json:"changes,omitempty"`
	PreBackupID      string            `json:"pre_backup_id,omitempty"`
	Error            string            `json:"error,omitempty"`
	Duration         time.Duration     `json:"duration"`
}

// RestoreFromBackup restores configuration from a backup
func (rm *RestoreManager) RestoreFromBackup(backupID string, currentConfig *Config, options RestoreOptions) (*RestoreResult, error) {
	startTime := time.Now()
	
	result := &RestoreResult{
		Success: false,
	}
	
	// Load the backup
	backup, err := rm.backupManager.GetBackup(backupID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to load backup: %v", err)
		return result, err
	}
	
	result.BackupUsed = backup
	
	// Run pre-restore hooks
	for _, hook := range options.PreRestoreHooks {
		if err := hook(); err != nil {
			result.Error = fmt.Sprintf("Pre-restore hook failed: %v", err)
			return result, err
		}
	}
	
	// Create a backup of current config if requested
	if options.CreateBackupFirst && currentConfig != nil && !options.DryRun {
		preBackup, err := rm.backupManager.CreateBackup(
			currentConfig,
			"Pre-restore backup",
			fmt.Sprintf("Automatic backup before restoring from %s", backupID),
			"system",
		)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to create pre-restore backup: %v", err)
			return result, err
		}
		result.PreBackupID = preBackup.ID
	}
	
	// Prepare the config to restore
	configToRestore := backup.Config
	
	// Apply selective restoration if specified
	if len(options.SelectiveSections) > 0 {
		configToRestore, err = rm.extractSections(backup.Config, options.SelectiveSections)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to extract sections: %v", err)
			return result, err
		}
	}
	
	// Exclude sections if specified
	if len(options.ExcludeSections) > 0 {
		configToRestore, err = rm.excludeSections(configToRestore, options.ExcludeSections)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to exclude sections: %v", err)
			return result, err
		}
	}
	
	// Handle merge if requested
	if options.MergeWithCurrent && currentConfig != nil {
		mergeOptions := options.MergeOptions
		if mergeOptions == nil {
			mergeOptions = &MergeOptions{
				Strategy:       MergeStrategyMerge,
				DeepMerge:      true,
				ValidateResult: options.ValidateBeforeRestore,
			}
		}
		
		mergeResult, err := rm.merger.Merge(currentConfig, configToRestore, *mergeOptions)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to merge configurations: %v", err)
			return result, err
		}
		
		if !mergeResult.Success {
			result.Error = "Merge failed due to conflicts or validation errors"
			result.ValidationResult = mergeResult.ValidationResult
			return result, fmt.Errorf("merge failed")
		}
		
		configToRestore = mergeResult.MergedConfig
		result.Changes = mergeResult.Changes
	} else {
		// Calculate changes for full restore
		if currentConfig != nil {
			if diff, err := rm.differ.Diff(currentConfig, configToRestore); err == nil {
				result.Changes = diff.Changes
			}
		}
	}
	
	// Validate before restore if requested
	if options.ValidateBeforeRestore && rm.validator != nil {
		validationResult := rm.validator.Validate(configToRestore)
		result.ValidationResult = validationResult
		
		if !validationResult.Valid {
			result.Error = "Validation failed"
			return result, fmt.Errorf("restored configuration is invalid: %s", validationResult.Summary)
		}
	}
	
	// If dry run, stop here
	if options.DryRun {
		result.Success = true
		result.RestoredConfig = configToRestore
		result.Duration = time.Since(startTime)
		return result, nil
	}
	
	// Apply the configuration
	if err := rm.applyConfiguration(configToRestore); err != nil {
		result.Error = fmt.Sprintf("Failed to apply configuration: %v", err)
		return result, err
	}
	
	// Run post-restore hooks
	for _, hook := range options.PostRestoreHooks {
		if err := hook(configToRestore); err != nil {
			// Log but don't fail the restore
			fmt.Printf("Warning: Post-restore hook failed: %v\n", err)
		}
	}
	
	result.Success = true
	result.RestoredConfig = configToRestore
	result.Duration = time.Since(startTime)
	
	return result, nil
}

// RestoreFromFile restores configuration from an exported backup file
func (rm *RestoreManager) RestoreFromFile(filePath string, currentConfig *Config, options RestoreOptions) (*RestoreResult, error) {
	// Import the backup first
	backup, err := rm.backupManager.ImportBackup(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to import backup: %w", err)
	}
	
	// Now restore from the imported backup
	return rm.RestoreFromBackup(backup.ID, currentConfig, options)
}

// RestoreToVersion restores to a specific configuration version
func (rm *RestoreManager) RestoreToVersion(version int, currentConfig *Config, options RestoreOptions) (*RestoreResult, error) {
	// Find backup with the specified version
	backups, err := rm.backupManager.ListBackups()
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	
	var targetBackup *ConfigBackup
	for _, backup := range backups {
		if backup.Metadata != nil {
			if v, ok := backup.Metadata["config_version"]; ok && v == fmt.Sprintf("%d", version) {
				targetBackup = backup
				break
			}
		}
	}
	
	if targetBackup == nil {
		return nil, fmt.Errorf("no backup found for version %d", version)
	}
	
	return rm.RestoreFromBackup(targetBackup.ID, currentConfig, options)
}

// PreviewRestore previews what would be restored without applying changes
func (rm *RestoreManager) PreviewRestore(backupID string, currentConfig *Config, options RestoreOptions) (*RestoreResult, error) {
	// Set dry run to true for preview
	options.DryRun = true
	return rm.RestoreFromBackup(backupID, currentConfig, options)
}

// extractSections extracts only specified sections from a configuration
func (rm *RestoreManager) extractSections(config *Config, sections []string) (*Config, error) {
	// Convert to map
	configMap, err := rm.configToMap(config)
	if err != nil {
		return nil, err
	}
	
	// Create new map with only selected sections
	extractedMap := make(map[string]interface{})
	for _, section := range sections {
		if value, exists := configMap[section]; exists {
			extractedMap[section] = value
		}
	}
	
	// Convert back to Config
	return rm.mapToConfig(extractedMap)
}

// excludeSections removes specified sections from a configuration
func (rm *RestoreManager) excludeSections(config *Config, sections []string) (*Config, error) {
	// Convert to map
	configMap, err := rm.configToMap(config)
	if err != nil {
		return nil, err
	}
	
	// Remove excluded sections
	for _, section := range sections {
		delete(configMap, section)
	}
	
	// Convert back to Config
	return rm.mapToConfig(configMap)
}

// applyConfiguration applies the restored configuration
func (rm *RestoreManager) applyConfiguration(config *Config) error {
	// Update the global AppConfig
	AppConfig = config
	
	// Save to default config file
	configPath := os.Getenv("QT1_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	
	return SaveConfig(configPath)
}

// configToMap converts Config to map
func (rm *RestoreManager) configToMap(config *Config) (map[string]interface{}, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// mapToConfig converts map to Config
func (rm *RestoreManager) mapToConfig(m map[string]interface{}) (*Config, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// QuickRestore performs a quick restore with minimal options
func (rm *RestoreManager) QuickRestore(backupID string) error {
	options := RestoreOptions{
		ValidateBeforeRestore: true,
		CreateBackupFirst:     true,
	}
	
	result, err := rm.RestoreFromBackup(backupID, AppConfig, options)
	if err != nil {
		return err
	}
	
	if !result.Success {
		return fmt.Errorf("restore failed: %s", result.Error)
	}
	
	return nil
}

// RollbackToPreviousBackup rolls back to the most recent backup
func (rm *RestoreManager) RollbackToPreviousBackup() error {
	backups, err := rm.backupManager.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}
	
	if len(backups) == 0 {
		return fmt.Errorf("no backups available for rollback")
	}
	
	// Backups are sorted by date (newest first)
	return rm.QuickRestore(backups[0].ID)
}

// RestorePoint represents a named restore point
type RestorePoint struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BackupID    string    `json:"backup_id"`
	CreatedAt   time.Time `json:"created_at"`
	Tags        []string  `json:"tags"`
}

// CreateRestorePoint creates a named restore point
func (rm *RestoreManager) CreateRestorePoint(name, description string) (*RestorePoint, error) {
	// Create a backup
	backup, err := rm.backupManager.CreateBackup(
		AppConfig,
		fmt.Sprintf("Restore Point: %s", name),
		description,
		"system",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create restore point: %w", err)
	}
	
	// Tag it as a restore point
	backup.Tags = append(backup.Tags, "restore-point")
	backup.Metadata["restore_point_name"] = name
	
	// Update backup metadata
	metaFile := filepath.Join(rm.backupManager.backupDir, fmt.Sprintf("%s.meta", backup.ID))
	metaData, _ := json.MarshalIndent(backup, "", "  ")
	ioutil.WriteFile(metaFile, metaData, 0644)
	
	return &RestorePoint{
		Name:        name,
		Description: description,
		BackupID:    backup.ID,
		CreatedAt:   backup.CreatedAt,
		Tags:        backup.Tags,
	}, nil
}

// ListRestorePoints lists all available restore points
func (rm *RestoreManager) ListRestorePoints() ([]*RestorePoint, error) {
	backups, err := rm.backupManager.ListBackups()
	if err != nil {
		return nil, err
	}
	
	restorePoints := []*RestorePoint{}
	
	for _, backup := range backups {
		// Check if it's a restore point
		isRestorePoint := false
		for _, tag := range backup.Tags {
			if tag == "restore-point" {
				isRestorePoint = true
				break
			}
		}
		
		if isRestorePoint {
			name := backup.Name
			if rpName, ok := backup.Metadata["restore_point_name"]; ok {
				name = rpName
			}
			
			restorePoints = append(restorePoints, &RestorePoint{
				Name:        name,
				Description: backup.Description,
				BackupID:    backup.ID,
				CreatedAt:   backup.CreatedAt,
				Tags:        backup.Tags,
			})
		}
	}
	
	return restorePoints, nil
}

// RestoreToPoint restores configuration to a named restore point
func (rm *RestoreManager) RestoreToPoint(pointName string) error {
	points, err := rm.ListRestorePoints()
	if err != nil {
		return err
	}
	
	for _, point := range points {
		if point.Name == pointName {
			return rm.QuickRestore(point.BackupID)
		}
	}
	
	return fmt.Errorf("restore point '%s' not found", pointName)
}

// VerifyRestore verifies that a restore operation would succeed
func (rm *RestoreManager) VerifyRestore(backupID string) error {
	// Load the backup
	backup, err := rm.backupManager.GetBackup(backupID)
	if err != nil {
		return fmt.Errorf("failed to load backup: %w", err)
	}
	
	// Validate the configuration
	if rm.validator != nil {
		result := rm.validator.Validate(backup.Config)
		if !result.Valid {
			return fmt.Errorf("backup configuration is invalid: %s", result.Summary)
		}
	}
	
	// Test the configuration
	tester := NewConfigTester(backup.Config)
	testResult := tester.Test()
	if !testResult.Success {
		return fmt.Errorf("configuration tests failed: %s", testResult.Summary)
	}
	
	return nil
}