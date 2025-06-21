package models

import (
	"time"
)

// SystemConfig represents a system configuration entry
type SystemConfig struct {
	ID          int        `json:"id" db:"id"`
	Key         string     `json:"key" db:"key" validate:"required,max=100"`
	Value       string     `json:"value" db:"value" validate:"required,max=5000"`
	ValueType   string     `json:"value_type" db:"value_type" validate:"required,oneof=string int float bool json"`
	Category    string     `json:"category" db:"category" validate:"required,max=50"`
	Description *string    `json:"description,omitempty" db:"description" validate:"omitempty,max=500"`
	IsSecret    bool       `json:"is_secret" db:"is_secret"`
	ReadOnly    bool       `json:"read_only" db:"read_only"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	UpdatedBy   *int       `json:"updated_by,omitempty" db:"updated_by" validate:"omitempty,gt=0"`
	Version     int        `json:"version" db:"version" validate:"gte=1"`
}

// SystemConfigCreatePayload represents the payload for creating a system config
type SystemConfigCreatePayload struct {
	Key         string  `json:"key" validate:"required,max=100"`
	Value       string  `json:"value" validate:"required,max=5000"`
	ValueType   string  `json:"value_type" validate:"required,oneof=string int float bool json"`
	Category    string  `json:"category" validate:"required,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	IsSecret    bool    `json:"is_secret"`
	ReadOnly    bool    `json:"read_only"`
}

// SystemConfigUpdatePayload represents the payload for updating a system config
type SystemConfigUpdatePayload struct {
	Value       *string `json:"value,omitempty" validate:"omitempty,max=5000"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	IsSecret    *bool   `json:"is_secret,omitempty"`
	ReadOnly    *bool   `json:"read_only,omitempty"`
}

// SystemConfigResponse represents the response payload for system config (hiding secrets)
type SystemConfigResponse struct {
	ID          int        `json:"id"`
	Key         string     `json:"key"`
	Value       string     `json:"value,omitempty"` // Hidden if IsSecret
	ValueType   string     `json:"value_type"`
	Category    string     `json:"category"`
	Description *string    `json:"description,omitempty"`
	IsSecret    bool       `json:"is_secret"`
	ReadOnly    bool       `json:"read_only"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UpdatedBy   *int       `json:"updated_by,omitempty"`
	Version     int        `json:"version"`
}

// SystemConfigFilters represents filters for querying system configs
type SystemConfigFilters struct {
	Category    *string `json:"category,omitempty" validate:"omitempty,max=50"`
	KeyPattern  *string `json:"key_pattern,omitempty" validate:"omitempty,max=100"`
	ValueType   *string `json:"value_type,omitempty" validate:"omitempty,oneof=string int float bool json"`
	IsSecret    *bool   `json:"is_secret,omitempty"`
	ReadOnly    *bool   `json:"read_only,omitempty"`
	UpdatedBy   *int    `json:"updated_by,omitempty" validate:"omitempty,gt=0"`
	Limit       int     `json:"limit" validate:"gte=1,lte=1000"`
	Offset      int     `json:"offset" validate:"gte=0"`
	OrderBy     string  `json:"order_by" validate:"omitempty,oneof=key category updated_at version"`
	OrderDir    string  `json:"order_dir" validate:"omitempty,oneof=asc desc"`
}

// ConfigCategory represents a configuration category with its settings
type ConfigCategory struct {
	Category    string                     `json:"category"`
	Description string                     `json:"description,omitempty"`
	Settings    []SystemConfigResponse     `json:"settings"`
}

// ToResponse converts a SystemConfig to SystemConfigResponse (hiding secrets if needed)
func (sc *SystemConfig) ToResponse(hideSecrets bool) *SystemConfigResponse {
	resp := &SystemConfigResponse{
		ID:          sc.ID,
		Key:         sc.Key,
		Value:       sc.Value,
		ValueType:   sc.ValueType,
		Category:    sc.Category,
		Description: sc.Description,
		IsSecret:    sc.IsSecret,
		ReadOnly:    sc.ReadOnly,
		CreatedAt:   sc.CreatedAt,
		UpdatedAt:   sc.UpdatedAt,
		UpdatedBy:   sc.UpdatedBy,
		Version:     sc.Version,
	}

	// Hide secret values if requested
	if hideSecrets && sc.IsSecret {
		resp.Value = "***HIDDEN***"
	}

	return resp
}

// IsEditable checks if the config can be edited
func (sc *SystemConfig) IsEditable() bool {
	return !sc.ReadOnly
}

// GetTypedValue returns the value parsed according to its type
func (sc *SystemConfig) GetTypedValue() interface{} {
	switch sc.ValueType {
	case "int":
		// In a real implementation, you'd parse the string to int
		return sc.Value
	case "float":
		// In a real implementation, you'd parse the string to float
		return sc.Value
	case "bool":
		return sc.Value == "true"
	case "json":
		// In a real implementation, you'd parse the JSON
		return sc.Value
	default:
		return sc.Value
	}
}