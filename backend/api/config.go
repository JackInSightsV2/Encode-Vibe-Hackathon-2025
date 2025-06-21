package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qt1-middleware/config"
	"strings"
	"time"
)

// ConfigAPI handles configuration-related API endpoints
type ConfigAPI struct {
	validator      *config.ConfigValidator
	tester         *config.ConfigTester
	backupManager  *config.BackupManager
	restoreManager *config.RestoreManager
	merger         *config.ConfigMerger
	differ         *config.ConfigDiffer
}

// NewConfigAPI creates a new configuration API handler
func NewConfigAPI(backupDir string) *ConfigAPI {
	schema := config.GetConfigSchema()
	validator := config.NewConfigValidator(schema)
	
	return &ConfigAPI{
		validator:      validator,
		tester:         config.NewConfigTester(config.AppConfig),
		backupManager:  config.NewBackupManager(backupDir),
		restoreManager: config.NewRestoreManager(backupDir, validator),
		merger:         config.NewConfigMerger(validator),
		differ:         config.NewConfigDiffer(),
	}
}

// HandleConfigSchema returns the configuration schema
func (ca *ConfigAPI) HandleConfigSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	schema := config.GetConfigSchema()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    schema,
	})
}

// HandleConfigValidate validates configuration
func (ca *ConfigAPI) HandleConfigValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate the configuration
	result := ca.validator.Validate(&cfg)
	
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if !result.Valid {
		status = http.StatusBadRequest
	}
	
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: result.Valid,
		Data:    result,
	})
}

// HandleConfigTest tests configuration
func (ca *ConfigAPI) HandleConfigTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Check if specific test categories are requested
	categories := r.URL.Query()["category"]
	
	var result *config.TestResult
	if len(categories) > 0 {
		result = ca.tester.TestSpecific(categories)
	} else {
		result = ca.tester.Test()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: result.Success,
		Data:    result,
	})
}

// HandleConfigVersions lists configuration versions (backups)
func (ca *ConfigAPI) HandleConfigVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	backups, err := ca.backupManager.ListBackups()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list versions: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Convert backups to versions format
	versions := make([]config.ConfigVersion, 0, len(backups))
	for i, backup := range backups {
		versions = append(versions, config.ConfigVersion{
			ID:          backup.ID,
			Version:     len(backups) - i, // Reverse order for version numbers
			CreatedBy:   backup.CreatedBy,
			CreatedAt:   backup.CreatedAt,
			Description: backup.Description,
			Active:      false, // Could be determined by comparing with current config
			Tags:        backup.Tags,
			Metadata:    backup.Metadata,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    versions,
	})
}

// HandleConfigCreateVersion creates a new configuration version (backup)
func (ca *ConfigAPI) HandleConfigCreateVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedBy   string `json:"created_by"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Create backup
	backup, err := ca.backupManager.CreateBackup(
		config.AppConfig,
		request.Name,
		request.Description,
		request.CreatedBy,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create version: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"id":         backup.ID,
			"created_at": backup.CreatedAt,
			"message":    "Configuration version created successfully",
		},
	})
}

// HandleConfigRollback rolls back to a specific version
func (ca *ConfigAPI) HandleConfigRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract version ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Version ID required", http.StatusBadRequest)
		return
	}
	versionID := pathParts[len(pathParts)-1]
	
	// Prepare restore options
	options := config.RestoreOptions{
		ValidateBeforeRestore: true,
		CreateBackupFirst:     true,
	}
	
	// Perform rollback
	result, err := ca.restoreManager.RestoreFromBackup(versionID, config.AppConfig, options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to rollback: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Reload configuration if successful
	if result.Success && ProxyInstance != nil {
		ProxyInstance.ReloadConfiguration()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: result.Success,
		Data:    result,
	})
}

// HandleConfigBackup creates a configuration backup
func (ca *ConfigAPI) HandleConfigBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Compress    bool   `json:"compress"`
		Encrypt     bool   `json:"encrypt"`
	}
	
	// Set defaults
	request.Name = fmt.Sprintf("Manual backup %s", time.Now().Format("2006-01-02 15:04"))
	request.Description = "Manual backup created via API"
	request.Compress = true
	
	// Parse request body if provided
	json.NewDecoder(r.Body).Decode(&request)
	
	// Configure backup options
	ca.backupManager.SetCompression(request.Compress)
	
	// Create backup
	backup, err := ca.backupManager.CreateBackup(
		config.AppConfig,
		request.Name,
		request.Description,
		"api",
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create backup: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]interface{}{
			"backup_id":  backup.ID,
			"created_at": backup.CreatedAt,
			"size":       backup.Size,
			"checksum":   backup.Checksum,
		},
	})
}

// HandleConfigBackups lists all backups
func (ca *ConfigAPI) HandleConfigBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	backups, err := ca.backupManager.ListBackups()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list backups: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    backups,
	})
}

// HandleConfigRestore restores from a backup
func (ca *ConfigAPI) HandleConfigRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		BackupID          string                `json:"backup_id"`
		ValidateFirst     bool                  `json:"validate_first"`
		CreateBackupFirst bool                  `json:"create_backup_first"`
		MergeWithCurrent  bool                  `json:"merge_with_current"`
		DryRun            bool                  `json:"dry_run"`
		SelectiveSections []string              `json:"selective_sections"`
		ExcludeSections   []string              `json:"exclude_sections"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if request.BackupID == "" {
		http.Error(w, "Backup ID required", http.StatusBadRequest)
		return
	}
	
	// Prepare restore options
	options := config.RestoreOptions{
		ValidateBeforeRestore: request.ValidateFirst,
		CreateBackupFirst:     request.CreateBackupFirst,
		MergeWithCurrent:      request.MergeWithCurrent,
		SelectiveSections:     request.SelectiveSections,
		ExcludeSections:       request.ExcludeSections,
		DryRun:                request.DryRun,
	}
	
	// Perform restore
	result, err := ca.restoreManager.RestoreFromBackup(request.BackupID, config.AppConfig, options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to restore: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Reload configuration if successful and not dry run
	if result.Success && !request.DryRun && ProxyInstance != nil {
		ProxyInstance.ReloadConfiguration()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: result.Success,
		Data:    result,
	})
}

// HandleConfigExport exports configuration
func (ca *ConfigAPI) HandleConfigExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var options config.ExportOptions
	options.Format = r.URL.Query().Get("format")
	if options.Format == "" {
		options.Format = "json"
	}
	
	// Parse request body for POST
	if r.Method == http.MethodPost {
		json.NewDecoder(r.Body).Decode(&options)
	} else {
		// Set defaults for GET
		options.Pretty = true
		options.IncludeDefaults = true
	}
	
	// Export based on format
	var data []byte
	var err error
	var contentType string
	
	switch options.Format {
	case "json":
		if options.Pretty {
			data, err = json.MarshalIndent(config.AppConfig, "", "  ")
		} else {
			data, err = json.Marshal(config.AppConfig)
		}
		contentType = "application/json"
		
	case "yaml":
		// In real implementation, would use yaml.Marshal
		data, err = json.MarshalIndent(config.AppConfig, "", "  ")
		contentType = "application/x-yaml"
		
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Set headers for file download
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=config.%s", options.Format))
	w.Write(data)
}

// HandleConfigImport imports configuration
func (ca *ConfigAPI) HandleConfigImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse multipart form for file upload
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	file, header, err := r.FormFile("config")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	// Read file content
	data := make([]byte, header.Size)
	_, err = file.Read(data)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	
	// Parse import options
	var options config.ImportOptions
	options.Format = r.FormValue("format")
	if options.Format == "" {
		// Try to detect format from filename
		if strings.HasSuffix(header.Filename, ".yaml") || strings.HasSuffix(header.Filename, ".yml") {
			options.Format = "yaml"
		} else {
			options.Format = "json"
		}
	}
	
	options.ValidateFirst = r.FormValue("validate_first") == "true"
	options.MergeWithExisting = r.FormValue("merge_with_existing") == "true"
	
	// Parse configuration based on format
	var newConfig config.Config
	switch options.Format {
	case "json":
		err = json.Unmarshal(data, &newConfig)
	case "yaml":
		// In real implementation, would use yaml.Unmarshal
		err = json.Unmarshal(data, &newConfig)
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse configuration: %v", err), http.StatusBadRequest)
		return
	}
	
	// Validate if requested
	if options.ValidateFirst {
		validationResult := ca.validator.Validate(&newConfig)
		if !validationResult.Valid {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Data:    validationResult,
				Error:   "Configuration validation failed",
			})
			return
		}
	}
	
	// Apply the configuration
	result := config.ImportResult{
		Success: true,
		Config:  &newConfig,
	}
	
	if options.MergeWithExisting {
		// Merge with current config
		mergeOptions := config.MergeOptions{
			Strategy:       config.MergeStrategyMerge,
			DeepMerge:      true,
			ValidateResult: true,
		}
		
		mergeResult, err := ca.merger.Merge(config.AppConfig, &newConfig, mergeOptions)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Merge failed: %v", err)
		} else {
			result.Config = mergeResult.MergedConfig
			result.Changes = mergeResult.Changes
			result.ValidationResult = mergeResult.ValidationResult
		}
	}
	
	if result.Success {
		// Apply the configuration
		config.AppConfig = result.Config
		
		// Save and reload
		if err := config.SaveConfig("config.yaml"); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to save configuration: %v", err)
		} else if ProxyInstance != nil {
			ProxyInstance.ReloadConfiguration()
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: result.Success,
		Data:    result,
	})
}

// HandleConfigDiff compares two configurations
func (ca *ConfigAPI) HandleConfigDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		OldConfig    *config.Config `json:"old_config"`
		NewConfig    *config.Config `json:"new_config"`
		OldVersionID string         `json:"old_version_id"`
		NewVersionID string         `json:"new_version_id"`
		Format       string         `json:"format"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Load configs from version IDs if provided
	if request.OldVersionID != "" {
		backup, err := ca.backupManager.GetBackup(request.OldVersionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load old version: %v", err), http.StatusBadRequest)
			return
		}
		request.OldConfig = backup.Config
	}
	
	if request.NewVersionID != "" {
		backup, err := ca.backupManager.GetBackup(request.NewVersionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load new version: %v", err), http.StatusBadRequest)
			return
		}
		request.NewConfig = backup.Config
	}
	
	// Use current config as new config if not provided
	if request.NewConfig == nil {
		request.NewConfig = config.AppConfig
	}
	
	if request.OldConfig == nil {
		http.Error(w, "Old configuration required", http.StatusBadRequest)
		return
	}
	
	// Calculate diff
	diff, err := ca.differ.Diff(request.OldConfig, request.NewConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate diff: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Format diff if requested
	if request.Format != "" && request.Format != "json" {
		formatted, err := config.FormatDiff(diff, request.Format)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to format diff: %v", err), http.StatusInternalServerError)
			return
		}
		
		contentType := "text/plain"
		if request.Format == "markdown" {
			contentType = "text/markdown"
		}
		
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(formatted))
		return
	}
	
	// Return JSON diff
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    diff,
	})
}

// HandleConfigMerge merges configurations
func (ca *ConfigAPI) HandleConfigMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		TargetConfig *config.Config       `json:"target_config"`
		SourceConfig *config.Config       `json:"source_config"`
		Options      config.MergeOptions  `json:"options"`
		Preview      bool                 `json:"preview"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Use current config as target if not provided
	if request.TargetConfig == nil {
		request.TargetConfig = config.AppConfig
	}
	
	if request.SourceConfig == nil {
		http.Error(w, "Source configuration required", http.StatusBadRequest)
		return
	}
	
	// Perform merge or preview
	var result *config.MergeResult
	var err error
	
	if request.Preview {
		result, err = ca.merger.CreateMergePreview(request.TargetConfig, request.SourceConfig, request.Options)
	} else {
		result, err = ca.merger.Merge(request.TargetConfig, request.SourceConfig, request.Options)
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to merge: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Apply merged config if not preview and successful
	if !request.Preview && result.Success && result.MergedConfig != nil {
		config.AppConfig = result.MergedConfig
		
		// Save and reload
		if err := config.SaveConfig("config.yaml"); err != nil {
			result.Success = false
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Error:   fmt.Sprintf("Failed to save merged configuration: %v", err),
			})
			return
		}
		
		if ProxyInstance != nil {
			ProxyInstance.ReloadConfiguration()
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: result.Success,
		Data:    result,
	})
}

// RegisterConfigRoutes registers all configuration API routes
func RegisterConfigRoutes(mux *http.ServeMux, configAPI *ConfigAPI) {
	mux.HandleFunc("/api/config/schema", configAPI.HandleConfigSchema)
	mux.HandleFunc("/api/config/validate", configAPI.HandleConfigValidate)
	mux.HandleFunc("/api/config/test", configAPI.HandleConfigTest)
	mux.HandleFunc("/api/config/versions", configAPI.HandleConfigVersions)
	mux.HandleFunc("/api/config/versions/create", configAPI.HandleConfigCreateVersion)
	mux.HandleFunc("/api/config/rollback/", configAPI.HandleConfigRollback)
	mux.HandleFunc("/api/config/backup", configAPI.HandleConfigBackup)
	mux.HandleFunc("/api/config/backups", configAPI.HandleConfigBackups)
	mux.HandleFunc("/api/config/restore", configAPI.HandleConfigRestore)
	mux.HandleFunc("/api/config/export", configAPI.HandleConfigExport)
	mux.HandleFunc("/api/config/import", configAPI.HandleConfigImport)
	mux.HandleFunc("/api/config/diff", configAPI.HandleConfigDiff)
	mux.HandleFunc("/api/config/merge", configAPI.HandleConfigMerge)
}