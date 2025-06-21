package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"qt1-middleware/models"
	"testing"
)

func TestRBACPermissionMatrix(t *testing.T) {
	pm := GetDefaultPermissionMatrix()

	// Test super admin has all permissions
	superAdminPerms := pm.GetPermissions(RoleSuperAdmin)
	if len(superAdminPerms) < 30 {
		t.Errorf("Super admin should have at least 30 permissions, got %d", len(superAdminPerms))
	}

	// Test specific role permissions
	testCases := []struct {
		role       Role
		permission Permission
		expected   bool
	}{
		{RoleSuperAdmin, PermissionSystemAdmin, true},
		{RoleAdmin, PermissionUserCreate, true},
		{RoleAdmin, PermissionSystemShutdown, false}, // Admin can't shutdown system
		{RoleModerator, PermissionModerateContent, true},
		{RoleModerator, PermissionUserDelete, false}, // Moderator can't delete users
		{RoleOperator, PermissionMetricsRead, true},
		{RoleOperator, PermissionSystemRestart, false}, // Operator can't restart system
		{RoleAnalyst, PermissionAnalyticsRead, true},
		{RoleAnalyst, PermissionConfigWrite, false}, // Analyst can't write config
		{RoleViewer, PermissionSystemHealth, true},
		{RoleViewer, PermissionUserCreate, false}, // Viewer can't create users
		{RoleUser, PermissionWebSocketConnect, true},
		{RoleUser, PermissionLogsRead, false}, // User can't read logs
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.role, tc.permission), func(t *testing.T) {
			result := pm.HasPermission(tc.role, tc.permission)
			if result != tc.expected {
				t.Errorf("Role %s permission %s: expected %v, got %v", tc.role, tc.permission, tc.expected, result)
			}
		})
	}
}

func TestRBACManager(t *testing.T) {
	// Create test auth service
	authService := &MockAuthService{}
	rbacManager := NewRBACManager(authService)

	// Test users with different roles
	users := []*models.User{
		{ID: 1, Username: "superadmin", Role: "super_admin"},
		{ID: 2, Username: "admin", Role: "admin"},
		{ID: 3, Username: "moderator", Role: "moderator"},
		{ID: 4, Username: "operator", Role: "operator"},
		{ID: 5, Username: "analyst", Role: "analyst"},
		{ID: 6, Username: "viewer", Role: "viewer"},
		{ID: 7, Username: "user", Role: "user"},
	}

	// Test permission checks
	testCases := []struct {
		userIndex  int
		permission Permission
		expected   bool
	}{
		{0, PermissionSystemAdmin, true},     // Super admin
		{1, PermissionUserCreate, true},      // Admin
		{1, PermissionSystemShutdown, false}, // Admin can't shutdown
		{2, PermissionModerateContent, true}, // Moderator
		{2, PermissionUserDelete, false},     // Moderator can't delete users
		{3, PermissionMetricsRead, true},     // Operator
		{3, PermissionSystemRestart, false},  // Operator can't restart
		{4, PermissionAnalyticsRead, true},   // Analyst
		{4, PermissionConfigWrite, false},    // Analyst can't write config
		{5, PermissionSystemHealth, true},    // Viewer
		{5, PermissionUserCreate, false},     // Viewer can't create users
		{6, PermissionWebSocketConnect, true}, // User
		{6, PermissionLogsRead, false},       // User can't read logs
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("user_%d_%s", tc.userIndex, tc.permission), func(t *testing.T) {
			user := users[tc.userIndex]
			result := rbacManager.CheckPermission(user, tc.permission)
			if result != tc.expected {
				t.Errorf("User %s (role: %s) permission %s: expected %v, got %v", 
					user.Username, user.Role, tc.permission, tc.expected, result)
			}
		})
	}
}

func TestRBACManagerHelperFunctions(t *testing.T) {
	authService := &MockAuthService{}
	rbacManager := NewRBACManager(authService)

	superAdmin := &models.User{ID: 1, Username: "superadmin", Role: "super_admin"}
	admin := &models.User{ID: 2, Username: "admin", Role: "admin"}
	moderator := &models.User{ID: 3, Username: "moderator", Role: "moderator"}
	user := &models.User{ID: 7, Username: "user", Role: "user"}

	// Test helper functions
	testCases := []struct {
		name     string
		user     *models.User
		function func(*models.User) bool
		expected bool
	}{
		{"SuperAdmin_IsSystemAdmin", superAdmin, rbacManager.IsSystemAdmin, true},
		{"Admin_IsSystemAdmin", admin, rbacManager.IsSystemAdmin, false},
		{"Admin_CanManageUsers", admin, rbacManager.CanManageUsers, true},
		{"Moderator_CanManageUsers", moderator, rbacManager.CanManageUsers, true},
		{"Moderator_CanModerateContent", moderator, rbacManager.CanModerateContent, true},
		{"User_CanModerateContent", user, rbacManager.CanModerateContent, false},
		{"Admin_CanAccessLogs", admin, rbacManager.CanAccessLogs, true},
		{"User_CanAccessLogs", user, rbacManager.CanAccessLogs, false},
		{"Admin_CanManageConfig", admin, rbacManager.CanManageConfig, true},
		{"Moderator_CanManageConfig", moderator, rbacManager.CanManageConfig, false},
		{"Admin_CanAccessMetrics", admin, rbacManager.CanAccessMetrics, true},
		{"User_CanAccessMetrics", user, rbacManager.CanAccessMetrics, false},
		{"Admin_CanManageKillSwitch", admin, rbacManager.CanManageKillSwitch, true},
		{"Moderator_CanManageKillSwitch", moderator, rbacManager.CanManageKillSwitch, true},
		{"User_CanManageKillSwitch", user, rbacManager.CanManageKillSwitch, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.function(tc.user)
			if result != tc.expected {
				t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}

func TestRBACMiddleware(t *testing.T) {
	// Create test auth service and middleware
	authService := &MockAuthService{
		users: map[string]*models.User{
			"admin-token": {ID: 1, Username: "admin", Role: "admin"},
			"user-token":  {ID: 2, Username: "user", Role: "user"},
		},
	}
	middleware := NewAuthMiddleware(authService)

	// Test handler that requires admin permission
	adminHandler := middleware.RequirePermission(PermissionUserCreate)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("admin access granted"))
	}))

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "AdminToken_Success",
			token:          "admin-token",
			expectedStatus: http.StatusOK,
			expectedBody:   "admin access granted",
		},
		{
			name:           "UserToken_Forbidden",
			token:          "user-token",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "NoToken_Unauthorized",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			
			rr := httptest.NewRecorder()
			adminHandler.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			if tc.expectedBody != "" && !bytes.Contains(rr.Body.Bytes(), []byte(tc.expectedBody)) {
				t.Errorf("Expected body to contain '%s', got '%s'", tc.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestRBACHandlers(t *testing.T) {
	// Create test auth service and handlers
	authService := &MockAuthService{
		users: map[string]*models.User{
			"admin-token": {ID: 1, Username: "admin", Role: "admin"},
		},
	}
	handlers := NewAuthHandlers(authService)

	t.Run("GetRoles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/roles", nil)
		rr := httptest.NewRecorder()

		handlers.GetRolesHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		roles, ok := data["roles"].([]interface{})
		if !ok {
			t.Fatal("Roles is not an array")
		}

		if len(roles) != 7 {
			t.Errorf("Expected 7 roles, got %d", len(roles))
		}
	})

	t.Run("GetPermissions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/permissions/all", nil)
		rr := httptest.NewRecorder()

		handlers.GetPermissionsHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		permissions, ok := data["permissions"].([]interface{})
		if !ok {
			t.Fatal("Permissions is not an array")
		}

		if len(permissions) < 30 {
			t.Errorf("Expected at least 30 permissions, got %d", len(permissions))
		}
	})

	t.Run("GetRolePermissions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/role-permissions?role=admin", nil)
		rr := httptest.NewRecorder()

		handlers.GetRolePermissionsHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		role, ok := data["role"].(string)
		if !ok || role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", role)
		}

		permissions, ok := data["permissions"].([]interface{})
		if !ok {
			t.Fatal("Permissions is not an array")
		}

		if len(permissions) < 10 {
			t.Errorf("Expected at least 10 permissions for admin role, got %d", len(permissions))
		}
	})

	t.Run("GetRBACStats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/auth/rbac-stats", nil)
		rr := httptest.NewRecorder()

		handlers.GetRBACStatsHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		totalRoles, ok := data["total_roles"].(float64)
		if !ok || totalRoles != 7 {
			t.Errorf("Expected 7 total roles, got %v", totalRoles)
		}

		totalPermissions, ok := data["total_permissions"].(float64)
		if !ok || totalPermissions < 30 {
			t.Errorf("Expected at least 30 total permissions, got %v", totalPermissions)
		}
	})
}

func TestResourceActionPermissions(t *testing.T) {
	// Test that permissions follow resource:action pattern
	testCases := []struct {
		permission Permission
		resource   string
		action     string
	}{
		{PermissionSystemAdmin, "system", "admin"},
		{PermissionUserCreate, "user", "create"},
		{PermissionConfigRead, "config", "read"},
		{PermissionLogsDownload, "logs", "download"},
		{PermissionModerateContent, "moderate", "content"},
		{PermissionKillSwitchWrite, "killswitch", "write"},
		{PermissionMetricsRead, "metrics", "read"},
		{PermissionDatabaseAdmin, "database", "admin"},
		{PermissionWebSocketConnect, "websocket", "connect"},
		{PermissionAPIAccess, "api", "access"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.permission), func(t *testing.T) {
			expected := fmt.Sprintf("%s:%s", tc.resource, tc.action)
			if string(tc.permission) != expected {
				t.Errorf("Permission format mismatch: expected %s, got %s", expected, tc.permission)
			}
		})
	}
}

func TestRBACCanAccessResource(t *testing.T) {
	authService := &MockAuthService{}
	rbacManager := NewRBACManager(authService)

	admin := &models.User{ID: 1, Username: "admin", Role: "admin"}
	user := &models.User{ID: 2, Username: "user", Role: "user"}

	testCases := []struct {
		name     string
		user     *models.User
		resource string
		action   string
		expected bool
	}{
		{"Admin_UserCreate", admin, "user", "create", true},
		{"Admin_SystemRestart", admin, "system", "restart", false},
		{"User_UserCreate", user, "user", "create", false},
		{"User_WebSocketConnect", user, "websocket", "connect", true},
		{"Admin_ConfigRead", admin, "config", "read", true},
		{"User_ConfigRead", user, "config", "read", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rbacManager.CanAccessResource(tc.user, tc.resource, tc.action)
			if result != tc.expected {
				t.Errorf("User %s accessing %s:%s: expected %v, got %v", 
					tc.user.Username, tc.resource, tc.action, tc.expected, result)
			}
		})
	}
}

// MockAuthService for testing
type MockAuthService struct {
	users map[string]*models.User
}

func (m *MockAuthService) Login(username, password string) (*AuthResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockAuthService) Register(req *RegisterRequest) (*models.User, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockAuthService) Logout(userID int) error {
	return fmt.Errorf("not implemented")
}

func (m *MockAuthService) GenerateToken(user *models.User) (string, error) {
	return fmt.Sprintf("token-for-%s", user.Username), nil
}

func (m *MockAuthService) ValidateToken(token string) (*Claims, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockAuthService) RefreshToken(refreshToken string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *MockAuthService) GetUserByToken(token string) (*models.User, error) {
	if m.users == nil {
		return nil, &AuthError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}
	
	user, exists := m.users[token]
	if !exists {
		return nil, &AuthError{Code: "USER_NOT_FOUND", Message: "User not found"}
	}
	
	return user, nil
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	return "hashed-" + password, nil
}

func (m *MockAuthService) VerifyPassword(hashedPassword, password string) error {
	if hashedPassword == "hashed-"+password {
		return nil
	}
	return fmt.Errorf("password mismatch")
}

func (m *MockAuthService) ChangePassword(userID int, oldPassword, newPassword string) error {
	return nil
}

// UserService interface methods
func (m *MockAuthService) Create(ctx context.Context, user *models.User) error {
	return nil
}

func (m *MockAuthService) GetByID(ctx context.Context, id int) (*models.User, error) {
	return &models.User{ID: id, Username: "testuser"}, nil
}

func (m *MockAuthService) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return &models.User{Username: username}, nil
}

func (m *MockAuthService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return &models.User{Email: email}, nil
}

func (m *MockAuthService) Update(ctx context.Context, user *models.User) error {
	return nil
}

func (m *MockAuthService) Delete(ctx context.Context, id int) error {
	return nil
}

func (m *MockAuthService) GetByOAuth2Provider(ctx context.Context, provider, providerUserID string) (*models.User, error) {
	return nil, fmt.Errorf("OAuth2 account not found")
}

func (m *MockAuthService) LinkOAuth2Account(ctx context.Context, userID int, provider, providerUserID, email string) error {
	return nil
}

func (m *MockAuthService) UnlinkOAuth2Account(ctx context.Context, userID int, provider string) error {
	return nil
}

func (m *MockAuthService) GetLinkedOAuth2Accounts(ctx context.Context, userID int) ([]*models.OAuth2Account, error) {
	return []*models.OAuth2Account{}, nil
}