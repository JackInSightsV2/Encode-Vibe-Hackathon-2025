package models

import (
	"testing"
	"time"
)

func TestUserToResponse(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:           1,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	response := user.ToResponse()

	// Check that sensitive fields are not included
	if response.ID != user.ID {
		t.Errorf("Expected ID %d, got %d", user.ID, response.ID)
	}
	if response.Username != user.Username {
		t.Errorf("Expected Username %s, got %s", user.Username, response.Username)
	}
	if response.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, response.Email)
	}
	if response.Role != user.Role {
		t.Errorf("Expected Role %s, got %s", user.Role, response.Role)
	}
	if response.Active != user.Active {
		t.Errorf("Expected Active %v, got %v", user.Active, response.Active)
	}
}

func TestUserHasRole(t *testing.T) {
	user := &User{Role: "admin"}

	if !user.HasRole("admin") {
		t.Error("Expected user to have admin role")
	}
	if user.HasRole("user") {
		t.Error("Expected user to not have user role")
	}
}

func TestUserIsAdmin(t *testing.T) {
	adminUser := &User{Role: "admin"}
	normalUser := &User{Role: "user"}

	if !adminUser.IsAdmin() {
		t.Error("Expected admin user to be admin")
	}
	if normalUser.IsAdmin() {
		t.Error("Expected normal user to not be admin")
	}
}

func TestUserIsModerator(t *testing.T) {
	moderatorUser := &User{Role: "moderator"}
	normalUser := &User{Role: "user"}

	if !moderatorUser.IsModerator() {
		t.Error("Expected moderator user to be moderator")
	}
	if normalUser.IsModerator() {
		t.Error("Expected normal user to not be moderator")
	}
}

func TestUserCanModerate(t *testing.T) {
	adminUser := &User{Role: "admin"}
	moderatorUser := &User{Role: "moderator"}
	normalUser := &User{Role: "user"}

	if !adminUser.CanModerate() {
		t.Error("Expected admin user to be able to moderate")
	}
	if !moderatorUser.CanModerate() {
		t.Error("Expected moderator user to be able to moderate")
	}
	if normalUser.CanModerate() {
		t.Error("Expected normal user to not be able to moderate")
	}
}

func TestValidateUserCreate(t *testing.T) {
	tests := []struct {
		name    string
		req     UserCreateRequest
		wantErr bool
	}{
		{
			name: "valid user",
			req: UserCreateRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "StrongPass123!",
				Role:     "user",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			req: UserCreateRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "StrongPass123!",
				Role:     "user",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			req: UserCreateRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "StrongPass123!",
				Role:     "user",
			},
			wantErr: true,
		},
		{
			name: "weak password",
			req: UserCreateRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "weak",
				Role:     "user",
			},
			wantErr: true,
		},
		{
			name: "invalid role",
			req: UserCreateRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "StrongPass123!",
				Role:     "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserCreate(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello world  ", "hello world"},
		{"hello\x00world", "helloworld"},
		{"hello\tworld\n", "hello\tworld"},
		{"hello\x01world", "helloworld"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		result := SanitizeString(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Test@Example.COM  ", "test@example.com"},
		{"USER@DOMAIN.ORG", "user@domain.org"},
		{"", ""},
	}

	for _, tt := range tests {
		result := SanitizeEmail(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeEmail(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  TestUser  ", "testuser"},
		{"USERNAME", "username"},
		{"", ""},
	}

	for _, tt := range tests {
		result := SanitizeUsername(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeUsername(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}