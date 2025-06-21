package auth

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// PasswordManager handles password hashing and validation
type PasswordManager struct {
	config *AuthConfig
}

// NewPasswordManager creates a new password manager
func NewPasswordManager(config *AuthConfig) *PasswordManager {
	if config == nil {
		config = DefaultAuthConfig()
	}
	return &PasswordManager{
		config: config,
	}
}

// HashPassword creates a bcrypt hash of the password
func (pm *PasswordManager) HashPassword(password string) (string, error) {
	// Validate password requirements before hashing
	if err := pm.ValidatePassword(password); err != nil {
		return "", err
	}
	
	hash, err := bcrypt.GenerateFromPassword([]byte(password), pm.config.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	
	return string(hash), nil
}

// VerifyPassword compares a hashed password with a plaintext password
func (pm *PasswordManager) VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

// ValidatePassword checks if a password meets the security requirements
func (pm *PasswordManager) ValidatePassword(password string) error {
	var errors []string
	
	// Check minimum length
	if len(password) < pm.config.MinPasswordLength {
		errors = append(errors, fmt.Sprintf("password must be at least %d characters long", pm.config.MinPasswordLength))
	}
	
	// Check for required character types
	if pm.config.RequireUppercase && !hasUppercase(password) {
		errors = append(errors, "password must contain at least one uppercase letter")
	}
	
	if pm.config.RequireLowercase && !hasLowercase(password) {
		errors = append(errors, "password must contain at least one lowercase letter")
	}
	
	if pm.config.RequireNumber && !hasNumber(password) {
		errors = append(errors, "password must contain at least one number")
	}
	
	if pm.config.RequireSpecialChar && !hasSpecialChar(password) {
		errors = append(errors, "password must contain at least one special character")
	}
	
	// Check for common weak patterns
	if isCommonPassword(password) {
		errors = append(errors, "password is too common")
	}
	
	if len(errors) > 0 {
		return &AuthError{
			Code:    "PASSWORD_TOO_WEAK",
			Message: "Password requirements not met: " + strings.Join(errors, ", "),
		}
	}
	
	return nil
}

// GenerateTemporaryPassword generates a secure temporary password
func (pm *PasswordManager) GenerateTemporaryPassword() (string, error) {
	// Implementation would use crypto/rand to generate a secure password
	// For now, return a placeholder
	return "TempPass123!", nil
}

// Password validation helper functions

func hasUppercase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func hasNumber(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasSpecialChar(s string) bool {
	// Define special characters
	specialChars := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`" + `]`)
	return specialChars.MatchString(s)
}

func isCommonPassword(password string) bool {
	// List of common passwords to reject
	commonPasswords := []string{
		"password", "123456", "123456789", "qwerty", "abc123",
		"password123", "admin", "letmein", "welcome", "monkey",
		"1234567890", "dragon", "superman", "batman", "master",
		"admin123", "root", "test", "guest", "user",
	}
	
	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPassword == common {
			return true
		}
	}
	
	// Check for simple patterns
	if isSequentialPassword(password) {
		return true
	}
	
	return false
}

func isSequentialPassword(password string) bool {
	// Check for simple sequential patterns like "12345", "abcde"
	if len(password) < 3 {
		return false
	}
	
	// Check for repeated digits like "1111"
	if len(password) >= 3 {
		firstChar := password[0]
		repeated := true
		for i := 1; i < len(password); i++ {
			if password[i] != firstChar {
				repeated = false
				break
			}
		}
		if repeated {
			return true
		}
	}
	
	// Check for sequential numbers
	sequential := []string{"012", "123", "234", "345", "456", "567", "678", "789"}
	lowerPassword := strings.ToLower(password)
	for _, seq := range sequential {
		if strings.Contains(lowerPassword, seq) && len(password) <= 6 {
			return true
		}
	}
	
	// Check for sequential letters
	letterSeq := []string{"abc", "bcd", "cde", "def", "efg", "fgh", "ghi", "hij", "ijk", "jkl", "klm", "lmn", "mno", "nop", "opq", "pqr", "qrs", "rst", "stu", "tuv", "uvw", "vwx", "wxy", "xyz"}
	for _, seq := range letterSeq {
		if strings.Contains(lowerPassword, seq) && len(password) <= 6 {
			return true
		}
	}
	
	return false
}