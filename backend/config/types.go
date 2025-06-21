package config

import (
	"fmt"
	"time"
)

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid      bool               `json:"valid"`
	Errors     []ValidationError  `json:"errors,omitempty"`
	Warnings   []ValidationWarning `json:"warnings,omitempty"`
	Summary    string             `json:"summary"`
	ValidatedAt time.Time         `json:"validated_at"`
	Duration   time.Duration      `json:"duration"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field       string      `json:"field"`
	Value       interface{} `json:"value,omitempty"`
	Message     string      `json:"message"`
	Code        string      `json:"code"`
	Severity    string      `json:"severity"` // "error", "critical"
	Suggestion  string      `json:"suggestion,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field      string      `json:"field"`
	Value      interface{} `json:"value,omitempty"`
	Message    string      `json:"message"`
	Code       string      `json:"code"`
	Suggestion string      `json:"suggestion,omitempty"`
}

// ConfigChange represents a change between two configurations
type ConfigChange struct {
	Path        string      `json:"path"`
	Operation   string      `json:"operation"` // "create", "update", "delete"
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Description string      `json:"description,omitempty"`
	Impact      string      `json:"impact,omitempty"` // "low", "medium", "high"
}

// ConfigDiff represents the difference between two configurations
type ConfigDiff struct {
	Changes     []ConfigChange `json:"changes"`
	Summary     string         `json:"summary"`
	AddedCount  int            `json:"added_count"`
	UpdatedCount int           `json:"updated_count"`
	DeletedCount int           `json:"deleted_count"`
	ImpactLevel string         `json:"impact_level"` // "none", "low", "medium", "high"
}

// TestResult represents the result of configuration testing
type TestResult struct {
	Success     bool           `json:"success"`
	Tests       []TestCase     `json:"tests"`
	Summary     string         `json:"summary"`
	Duration    time.Duration  `json:"duration"`
	TestedAt    time.Time      `json:"tested_at"`
	PassedCount int            `json:"passed_count"`
	FailedCount int            `json:"failed_count"`
	SkippedCount int           `json:"skipped_count"`
}

// TestCase represents a single test case
type TestCase struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Status      string        `json:"status"` // "passed", "failed", "skipped"
	Message     string        `json:"message,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ConfigBackup represents a configuration backup
type ConfigBackup struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Config      *Config           `json:"config"`
	Checksum    string            `json:"checksum"`
	CreatedAt   time.Time         `json:"created_at"`
	CreatedBy   string            `json:"created_by"`
	Size        int64             `json:"size"`
	Compressed  bool              `json:"compressed"`
	Encrypted   bool              `json:"encrypted"`
	Version     string            `json:"version"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ConfigVersion represents a versioned configuration
type ConfigVersion struct {
	ID          string            `json:"id"`
	Version     int               `json:"version"`
	Config      *Config           `json:"config"`
	CreatedBy   string            `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	Description string            `json:"description"`
	Active      bool              `json:"active"`
	Environment string            `json:"environment"`
	Changes     []ConfigChange    `json:"changes,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// MergeOptions defines options for configuration merging
type MergeOptions struct {
	Strategy          MergeStrategy          `json:"strategy"`
	ConflictResolver  ConflictResolver       `json:"conflict_resolver"`
	PreserveArrays    bool                   `json:"preserve_arrays"`
	DeepMerge         bool                   `json:"deep_merge"`
	IgnoreFields      []string               `json:"ignore_fields"`
	ValidateResult    bool                   `json:"validate_result"`
}

// MergeStrategy defines the merge strategy
type MergeStrategy string

const (
	MergeStrategyOverwrite MergeStrategy = "overwrite"
	MergeStrategyMerge     MergeStrategy = "merge"
	MergeStrategyAppend    MergeStrategy = "append"
	MergeStrategySkip      MergeStrategy = "skip"
)

// ConflictResolver defines how to resolve merge conflicts
type ConflictResolver string

const (
	ConflictResolverSource      ConflictResolver = "source"
	ConflictResolverTarget      ConflictResolver = "target"
	ConflictResolverNewer       ConflictResolver = "newer"
	ConflictResolverInteractive ConflictResolver = "interactive"
)

// MergeResult represents the result of a configuration merge
type MergeResult struct {
	Success       bool           `json:"success"`
	MergedConfig  *Config        `json:"merged_config,omitempty"`
	Conflicts     []MergeConflict `json:"conflicts,omitempty"`
	Changes       []ConfigChange `json:"changes"`
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`
}

// MergeConflict represents a merge conflict
type MergeConflict struct {
	Path         string      `json:"path"`
	SourceValue  interface{} `json:"source_value"`
	TargetValue  interface{} `json:"target_value"`
	Resolution   string      `json:"resolution"`
	ResolvedValue interface{} `json:"resolved_value,omitempty"`
}

// ImportOptions defines options for configuration import
type ImportOptions struct {
	Format           string           `json:"format"` // "json", "yaml", "toml"
	ValidateFirst    bool             `json:"validate_first"`
	MergeWithExisting bool            `json:"merge_with_existing"`
	MergeOptions     *MergeOptions    `json:"merge_options,omitempty"`
	StripSecrets     bool             `json:"strip_secrets"`
	ApplyDefaults    bool             `json:"apply_defaults"`
}

// ImportResult represents the result of configuration import
type ImportResult struct {
	Success      bool              `json:"success"`
	Config       *Config           `json:"config,omitempty"`
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`
	Changes      []ConfigChange    `json:"changes,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// ExportOptions defines options for configuration export
type ExportOptions struct {
	Format          string   `json:"format"` // "json", "yaml", "toml"
	IncludeSecrets  bool     `json:"include_secrets"`
	IncludeDefaults bool     `json:"include_defaults"`
	Pretty          bool     `json:"pretty"`
	Sections        []string `json:"sections,omitempty"` // Export only specific sections
	ExcludeFields   []string `json:"exclude_fields,omitempty"`
	TemplateMode    bool     `json:"template_mode"` // Export as template with placeholders
}

// TemplateVariable represents a variable in a configuration template
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`
	Default      interface{} `json:"default,omitempty"`
	Description  string      `json:"description"`
	Example      interface{} `json:"example,omitempty"`
	Validation   string      `json:"validation,omitempty"`
}

// ConfigTemplate represents a configuration template
type ConfigTemplate struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Category    string                      `json:"category"`
	Version     string                      `json:"version"`
	Template    map[string]interface{}      `json:"template"`
	Variables   []TemplateVariable          `json:"variables"`
	Presets     map[string]map[string]interface{} `json:"presets,omitempty"`
	Tags        []string                    `json:"tags,omitempty"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

// ValidationContext provides context for validation
type ValidationContext struct {
	Config       *Config                `json:"-"`
	Schema       *ConfigSchema          `json:"-"`
	Environment  string                 `json:"environment"`
	StrictMode   bool                   `json:"strict_mode"`
	CustomRules  map[string]interface{} `json:"custom_rules,omitempty"`
	SkipFields   []string               `json:"skip_fields,omitempty"`
	MaxErrors    int                    `json:"max_errors"`
}

// Error implements the error interface for ValidationError
func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// Error implements the error interface for ValidationResult
func (vr ValidationResult) Error() string {
	if vr.Valid {
		return ""
	}
	if len(vr.Errors) == 1 {
		return vr.Errors[0].Error()
	}
	return fmt.Sprintf("validation failed with %d errors", len(vr.Errors))
}

// HasErrors returns true if there are any validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings returns true if there are any validation warnings
func (vr *ValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// AddError adds a validation error
func (vr *ValidationResult) AddError(field, message, code string) {
	vr.Errors = append(vr.Errors, ValidationError{
		Field:    field,
		Message:  message,
		Code:     code,
		Severity: "error",
	})
	vr.Valid = false
}

// AddWarning adds a validation warning
func (vr *ValidationResult) AddWarning(field, message, code string) {
	vr.Warnings = append(vr.Warnings, ValidationWarning{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// HasChanges returns true if there are any changes
func (cd *ConfigDiff) HasChanges() bool {
	return len(cd.Changes) > 0
}

// IsSuccessful returns true if all tests passed
func (tr *TestResult) IsSuccessful() bool {
	return tr.Success && tr.FailedCount == 0
}