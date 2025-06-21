package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	// Common regex patterns for validation
	alphanumPattern = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	emailPattern    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func init() {
	validate = validator.New()
	
	// Register custom validation functions
	validate.RegisterValidation("alphanum", validateAlphanumeric)
	validate.RegisterValidation("username", validateUsername)
}

// ValidateStruct validates a struct using the validator tags
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err != nil {
		return formatValidationError(err)
	}
	return nil
}

// ValidateUserCreate validates a user creation request
func ValidateUserCreate(req *UserCreateRequest) error {
	if err := ValidateStruct(req); err != nil {
		return err
	}
	
	// Additional custom validations
	if err := validateStrongPassword(req.Password); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}
	
	return nil
}

// ValidateUserUpdate validates a user update request
func ValidateUserUpdate(req *UserUpdateRequest) error {
	return ValidateStruct(req)
}

// ValidateUserLogin validates a user login request
func ValidateUserLogin(req *UserLoginRequest) error {
	return ValidateStruct(req)
}

// SanitizeString removes potentially dangerous characters and trims whitespace
func SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Remove control characters except tab, newline, and carriage return
	var result strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\t' || r == '\n' || r == '\r' {
			result.WriteRune(r)
		}
	}
	
	// Trim whitespace
	return strings.TrimSpace(result.String())
}

// SanitizeEmail normalizes an email address
func SanitizeEmail(email string) string {
	email = SanitizeString(email)
	return strings.ToLower(email)
}

// SanitizeUsername normalizes a username
func SanitizeUsername(username string) string {
	username = SanitizeString(username)
	return strings.ToLower(username)
}

// Custom validation functions

// validateAlphanumeric checks if a string contains only alphanumeric characters
func validateAlphanumeric(fl validator.FieldLevel) bool {
	return alphanumPattern.MatchString(fl.Field().String())
}

// validateStrongPassword checks if a password meets security requirements
func validateStrongPassword(password string) error {
	pwd := password
	
	if len(pwd) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	
	if len(pwd) > 72 {
		return fmt.Errorf("password must be no more than 72 characters long")
	}
	
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range pwd {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}
	
	return nil
}

// validateUsername checks if a username is valid
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	
	// Must be alphanumeric
	if !alphanumPattern.MatchString(username) {
		return false
	}
	
	// Must start with a letter
	if len(username) > 0 && !(username[0] >= 'a' && username[0] <= 'z' || username[0] >= 'A' && username[0] <= 'Z') {
		return false
	}
	
	return true
}

// formatValidationError converts validator errors to a readable format
func formatValidationError(err error) error {
	var errors []string
	
	for _, err := range err.(validator.ValidationErrors) {
		var message string
		
		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("%s is required", err.Field())
		case "email":
			message = fmt.Sprintf("%s must be a valid email address", err.Field())
		case "min":
			message = fmt.Sprintf("%s must be at least %s characters long", err.Field(), err.Param())
		case "max":
			message = fmt.Sprintf("%s must be no more than %s characters long", err.Field(), err.Param())
		case "len":
			message = fmt.Sprintf("%s must be exactly %s characters long", err.Field(), err.Param())
		case "alphanum":
			message = fmt.Sprintf("%s must contain only alphanumeric characters", err.Field())
		case "username":
			message = fmt.Sprintf("%s must be a valid username (alphanumeric, starting with a letter)", err.Field())
		case "oneof":
			message = fmt.Sprintf("%s must be one of: %s", err.Field(), err.Param())
		case "gt":
			message = fmt.Sprintf("%s must be greater than %s", err.Field(), err.Param())
		case "gte":
			message = fmt.Sprintf("%s must be greater than or equal to %s", err.Field(), err.Param())
		case "lt":
			message = fmt.Sprintf("%s must be less than %s", err.Field(), err.Param())
		case "lte":
			message = fmt.Sprintf("%s must be less than or equal to %s", err.Field(), err.Param())
		case "ip":
			message = fmt.Sprintf("%s must be a valid IP address", err.Field())
		default:
			message = fmt.Sprintf("%s failed validation (%s)", err.Field(), err.Tag())
		}
		
		errors = append(errors, message)
	}
	
	return fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// GetValidationErrors returns structured validation errors
func GetValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError
	
	if validatorErr, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validatorErr {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Tag:     err.Tag(),
				Message: getValidationMessage(err),
				Value:   fmt.Sprintf("%v", err.Value()),
			})
		}
	}
	
	return validationErrors
}

// getValidationMessage returns a user-friendly validation message
func getValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return fmt.Sprintf("Must be at least %s characters long", err.Param())
	case "max":
		return fmt.Sprintf("Must be no more than %s characters long", err.Param())
	case "alphanum":
		return "Must contain only alphanumeric characters"
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", err.Param())
	default:
		return fmt.Sprintf("Validation failed: %s", err.Tag())
	}
}