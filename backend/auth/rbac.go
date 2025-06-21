package auth

import (
	"fmt"
	"log"
	"qt1-middleware/models"
	"strings"
)

// Role represents a user role in the system
type Role string

const (
	RoleSuperAdmin Role = "super_admin" // Full system access, can manage other admins
	RoleAdmin      Role = "admin"       // System administration, user management
	RoleModerator  Role = "moderator"   // Content moderation, limited admin
	RoleOperator   Role = "operator"    // System operations, monitoring
	RoleAnalyst    Role = "analyst"     // Read-only access to analytics and metrics
	RoleViewer     Role = "viewer"      // Read-only access to basic functionality
	RoleUser       Role = "user"        // Basic authenticated user access
)

// Permission represents a specific permission in the system
type Permission string

// System Administration Permissions
const (
	PermissionSystemAdmin     Permission = "system:admin"         // Full system administration
	PermissionSystemRestart   Permission = "system:restart"       // Restart system services
	PermissionSystemShutdown  Permission = "system:shutdown"      // Shutdown system
	PermissionSystemConfig    Permission = "system:config"        // Manage system configuration
	PermissionSystemHealth    Permission = "system:health"        // View system health
	PermissionSystemMetrics   Permission = "system:metrics"       // View system metrics
)

// User Management Permissions
const (
	PermissionUserCreate Permission = "user:create" // Create new users
	PermissionUserRead   Permission = "user:read"   // View user information
	PermissionUserUpdate Permission = "user:update" // Update user information
	PermissionUserDelete Permission = "user:delete" // Delete users
	PermissionUserList   Permission = "user:list"   // List all users
	PermissionUserRoles  Permission = "user:roles"  // Manage user roles
)

// Configuration Management Permissions
const (
	PermissionConfigRead   Permission = "config:read"   // Read configuration
	PermissionConfigWrite  Permission = "config:write"  // Write configuration
	PermissionConfigDeploy Permission = "config:deploy" // Deploy configuration changes
	PermissionConfigBackup Permission = "config:backup" // Backup/restore configuration
)

// Logging and Monitoring Permissions
const (
	PermissionLogsRead     Permission = "logs:read"     // Read log files
	PermissionLogsDownload Permission = "logs:download" // Download log files
	PermissionLogsDelete   Permission = "logs:delete"   // Delete log files
	PermissionLogsConfig   Permission = "logs:config"   // Configure logging settings
)

// Moderation Permissions
const (
	PermissionModerateContent Permission = "moderate:content" // Moderate user content
	PermissionModerateUsers   Permission = "moderate:users"   // Moderate users (ban/unban)
	PermissionModerateRules   Permission = "moderate:rules"   // Manage moderation rules
	PermissionModerateAudit   Permission = "moderate:audit"   // View moderation audit logs
)

// Kill Switch and Security Permissions
const (
	PermissionKillSwitchRead   Permission = "killswitch:read"   // View kill switch status
	PermissionKillSwitchWrite  Permission = "killswitch:write"  // Manage kill switches
	PermissionKillSwitchDeploy Permission = "killswitch:deploy" // Deploy kill switch changes
	PermissionSecurityManage   Permission = "security:manage"   // Manage security settings
)

// Analytics and Metrics Permissions
const (
	PermissionMetricsRead     Permission = "metrics:read"     // Read metrics data
	PermissionMetricsAnalyze  Permission = "metrics:analyze"  // Advanced metrics analysis
	PermissionMetricsExport   Permission = "metrics:export"   // Export metrics data
	PermissionAnalyticsRead   Permission = "analytics:read"   // Read analytics data
	PermissionAnalyticsWrite  Permission = "analytics:write"  // Write analytics data
	PermissionAnalyticsManage Permission = "analytics:manage" // Manage analytics settings
)

// Database Permissions
const (
	PermissionDatabaseRead   Permission = "database:read"   // Read database information
	PermissionDatabaseWrite  Permission = "database:write"  // Write to database
	PermissionDatabaseAdmin  Permission = "database:admin"  // Database administration
	PermissionDatabaseBackup Permission = "database:backup" // Database backup/restore
)

// WebSocket and API Permissions
const (
	PermissionWebSocketConnect Permission = "websocket:connect" // Connect to WebSocket
	PermissionAPIAccess        Permission = "api:access"        // General API access
	PermissionAPIAdmin         Permission = "api:admin"         // Administrative API access
)

// PermissionMatrix defines the permissions for each role
type PermissionMatrix map[Role][]Permission

// GetDefaultPermissionMatrix returns the default permission matrix for the system
func GetDefaultPermissionMatrix() PermissionMatrix {
	return PermissionMatrix{
		RoleSuperAdmin: {
			// Super Admin has ALL permissions
			PermissionSystemAdmin, PermissionSystemRestart, PermissionSystemShutdown,
			PermissionSystemConfig, PermissionSystemHealth, PermissionSystemMetrics,
			PermissionUserCreate, PermissionUserRead, PermissionUserUpdate,
			PermissionUserDelete, PermissionUserList, PermissionUserRoles,
			PermissionConfigRead, PermissionConfigWrite, PermissionConfigDeploy, PermissionConfigBackup,
			PermissionLogsRead, PermissionLogsDownload, PermissionLogsDelete, PermissionLogsConfig,
			PermissionModerateContent, PermissionModerateUsers, PermissionModerateRules, PermissionModerateAudit,
			PermissionKillSwitchRead, PermissionKillSwitchWrite, PermissionKillSwitchDeploy, PermissionSecurityManage,
			PermissionMetricsRead, PermissionMetricsAnalyze, PermissionMetricsExport,
			PermissionAnalyticsRead, PermissionAnalyticsWrite, PermissionAnalyticsManage,
			PermissionDatabaseRead, PermissionDatabaseWrite, PermissionDatabaseAdmin, PermissionDatabaseBackup,
			PermissionWebSocketConnect, PermissionAPIAccess, PermissionAPIAdmin,
		},
		RoleAdmin: {
			// Admin has most permissions except super admin functions
			PermissionSystemConfig, PermissionSystemHealth, PermissionSystemMetrics,
			PermissionUserCreate, PermissionUserRead, PermissionUserUpdate,
			PermissionUserDelete, PermissionUserList, PermissionUserRoles,
			PermissionConfigRead, PermissionConfigWrite, PermissionConfigDeploy, PermissionConfigBackup,
			PermissionLogsRead, PermissionLogsDownload, PermissionLogsConfig,
			PermissionModerateContent, PermissionModerateUsers, PermissionModerateRules, PermissionModerateAudit,
			PermissionKillSwitchRead, PermissionKillSwitchWrite, PermissionKillSwitchDeploy, PermissionSecurityManage,
			PermissionMetricsRead, PermissionMetricsAnalyze, PermissionMetricsExport,
			PermissionAnalyticsRead, PermissionAnalyticsWrite, PermissionAnalyticsManage,
			PermissionDatabaseRead, PermissionDatabaseAdmin, PermissionDatabaseBackup,
			PermissionWebSocketConnect, PermissionAPIAccess, PermissionAPIAdmin,
		},
		RoleModerator: {
			// Moderator focused on content and user moderation
			PermissionSystemHealth, PermissionSystemMetrics,
			PermissionUserRead, PermissionUserUpdate, PermissionUserList,
			PermissionConfigRead,
			PermissionLogsRead,
			PermissionModerateContent, PermissionModerateUsers, PermissionModerateRules, PermissionModerateAudit,
			PermissionKillSwitchRead, PermissionKillSwitchWrite,
			PermissionMetricsRead, PermissionAnalyticsRead,
			PermissionDatabaseRead,
			PermissionWebSocketConnect, PermissionAPIAccess,
		},
		RoleOperator: {
			// Operator focused on system operations and monitoring
			PermissionSystemHealth, PermissionSystemMetrics, PermissionSystemConfig,
			PermissionUserRead, PermissionUserList,
			PermissionConfigRead, PermissionConfigWrite,
			PermissionLogsRead, PermissionLogsDownload,
			PermissionModerateAudit,
			PermissionKillSwitchRead,
			PermissionMetricsRead, PermissionMetricsAnalyze, PermissionMetricsExport,
			PermissionAnalyticsRead, PermissionAnalyticsWrite,
			PermissionDatabaseRead,
			PermissionWebSocketConnect, PermissionAPIAccess,
		},
		RoleAnalyst: {
			// Analyst focused on data analysis and reporting
			PermissionSystemHealth, PermissionSystemMetrics,
			PermissionUserRead, PermissionUserList,
			PermissionConfigRead,
			PermissionLogsRead,
			PermissionModerateAudit,
			PermissionMetricsRead, PermissionMetricsAnalyze, PermissionMetricsExport,
			PermissionAnalyticsRead, PermissionAnalyticsWrite, PermissionAnalyticsManage,
			PermissionDatabaseRead,
			PermissionWebSocketConnect, PermissionAPIAccess,
		},
		RoleViewer: {
			// Viewer has read-only access to most resources
			PermissionSystemHealth, PermissionSystemMetrics,
			PermissionUserRead,
			PermissionConfigRead,
			PermissionLogsRead,
			PermissionMetricsRead, PermissionAnalyticsRead,
			PermissionDatabaseRead,
			PermissionWebSocketConnect, PermissionAPIAccess,
		},
		RoleUser: {
			// User has basic access for normal operations
			PermissionSystemHealth,
			PermissionWebSocketConnect, PermissionAPIAccess,
		},
	}
}

// HasPermission checks if a role has a specific permission
func (pm PermissionMatrix) HasPermission(role Role, permission Permission) bool {
	permissions, exists := pm[role]
	if !exists {
		log.Printf("RBAC: Role '%s' not found in permission matrix", role)
		return false
	}
	
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	
	return false
}

// GetPermissions returns all permissions for a given role
func (pm PermissionMatrix) GetPermissions(role Role) []Permission {
	permissions, exists := pm[role]
	if !exists {
		return []Permission{}
	}
	return permissions
}

// ValidateRole checks if a role is valid
func (pm PermissionMatrix) ValidateRole(role Role) bool {
	_, exists := pm[role]
	return exists
}

// GetAllRoles returns all available roles
func (pm PermissionMatrix) GetAllRoles() []Role {
	roles := make([]Role, 0, len(pm))
	for role := range pm {
		roles = append(roles, role)
	}
	return roles
}

// GetAllPermissions returns all available permissions
func (pm PermissionMatrix) GetAllPermissions() []Permission {
	permissionSet := make(map[Permission]bool)
	for _, permissions := range pm {
		for _, permission := range permissions {
			permissionSet[permission] = true
		}
	}
	
	permissions := make([]Permission, 0, len(permissionSet))
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	return permissions
}

// RBACManager manages role-based access control
type RBACManager struct {
	permissionMatrix PermissionMatrix
	authService      AuthService
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(authService AuthService) *RBACManager {
	return &RBACManager{
		permissionMatrix: GetDefaultPermissionMatrix(),
		authService:      authService,
	}
}

// CheckPermission checks if a user has a specific permission
func (rm *RBACManager) CheckPermission(user *models.User, permission Permission) bool {
	if user == nil {
		log.Printf("RBAC: Null user cannot have permission '%s'", permission)
		return false
	}
	
	role := Role(user.Role)
	hasPermission := rm.permissionMatrix.HasPermission(role, permission)
	
	if !hasPermission {
		log.Printf("RBAC: User '%s' (role: %s) denied permission '%s'", user.Username, user.Role, permission)
	}
	
	return hasPermission
}

// CheckRolePermission checks if a role has a specific permission (without user context)
func (rm *RBACManager) CheckRolePermission(role string, permission Permission) bool {
	return rm.permissionMatrix.HasPermission(Role(role), permission)
}

// GetUserPermissions returns all permissions for a user
func (rm *RBACManager) GetUserPermissions(user *models.User) []Permission {
	if user == nil {
		return []Permission{}
	}
	
	role := Role(user.Role)
	return rm.permissionMatrix.GetPermissions(role)
}

// GetRolePermissions returns all permissions for a specific role
func (rm *RBACManager) GetRolePermissions(role string) []Permission {
	return rm.permissionMatrix.GetPermissions(Role(role))
}

// ValidateUserRole checks if a user's role is valid
func (rm *RBACManager) ValidateUserRole(user *models.User) bool {
	if user == nil {
		return false
	}
	
	role := Role(user.Role)
	return rm.permissionMatrix.ValidateRole(role)
}

// IsValidRole checks if a role string is valid
func (rm *RBACManager) IsValidRole(role string) bool {
	return rm.permissionMatrix.ValidateRole(Role(role))
}

// GetAvailableRoles returns all available roles in the system
func (rm *RBACManager) GetAvailableRoles() []string {
	roles := rm.permissionMatrix.GetAllRoles()
	stringRoles := make([]string, len(roles))
	for i, role := range roles {
		stringRoles[i] = string(role)
	}
	return stringRoles
}

// GetAvailablePermissions returns all available permissions in the system
func (rm *RBACManager) GetAvailablePermissions() []string {
	permissions := rm.permissionMatrix.GetAllPermissions()
	stringPermissions := make([]string, len(permissions))
	for i, permission := range permissions {
		stringPermissions[i] = string(permission)
	}
	return stringPermissions
}

// CanAccessResource checks if a user can access a specific resource
func (rm *RBACManager) CanAccessResource(user *models.User, resource string, action string) bool {
	if user == nil {
		return false
	}
	
	// Build permission string from resource and action
	permission := Permission(fmt.Sprintf("%s:%s", resource, action))
	return rm.CheckPermission(user, permission)
}

// GetPermissionInfo returns detailed information about a permission
func (rm *RBACManager) GetPermissionInfo(permission Permission) map[string]interface{} {
	parts := strings.Split(string(permission), ":")
	resource := ""
	action := ""
	
	if len(parts) >= 2 {
		resource = parts[0]
		action = parts[1]
	}
	
	// Count how many roles have this permission
	roleCount := 0
	rolesWithPermission := []string{}
	
	for role, permissions := range rm.permissionMatrix {
		for _, p := range permissions {
			if p == permission {
				roleCount++
				rolesWithPermission = append(rolesWithPermission, string(role))
				break
			}
		}
	}
	
	return map[string]interface{}{
		"permission": string(permission),
		"resource":   resource,
		"action":     action,
		"roles":      rolesWithPermission,
		"role_count": roleCount,
	}
}

// GetRoleInfo returns detailed information about a role
func (rm *RBACManager) GetRoleInfo(role string) map[string]interface{} {
	roleType := Role(role)
	permissions := rm.permissionMatrix.GetPermissions(roleType)
	
	permissionStrings := make([]string, len(permissions))
	for i, permission := range permissions {
		permissionStrings[i] = string(permission)
	}
	
	return map[string]interface{}{
		"role":             role,
		"permissions":      permissionStrings,
		"permission_count": len(permissions),
		"valid":           rm.permissionMatrix.ValidateRole(roleType),
	}
}

// AuditUserAccess logs user access attempts for security auditing
func (rm *RBACManager) AuditUserAccess(user *models.User, permission Permission, granted bool, context string) {
	username := "anonymous"
	role := "none"
	
	if user != nil {
		username = user.Username
		role = user.Role
	}
	
	status := "DENIED"
	if granted {
		status = "GRANTED"
	}
	
	log.Printf("RBAC AUDIT: user='%s' role='%s' permission='%s' status='%s' context='%s'",
		username, role, permission, status, context)
}

// Helper functions for common permission checks

// IsSystemAdmin checks if user has system admin permissions
func (rm *RBACManager) IsSystemAdmin(user *models.User) bool {
	return rm.CheckPermission(user, PermissionSystemAdmin)
}

// CanManageUsers checks if user can manage other users
func (rm *RBACManager) CanManageUsers(user *models.User) bool {
	return rm.CheckPermission(user, PermissionUserCreate) ||
		rm.CheckPermission(user, PermissionUserUpdate) ||
		rm.CheckPermission(user, PermissionUserDelete)
}

// CanModerateContent checks if user can moderate content
func (rm *RBACManager) CanModerateContent(user *models.User) bool {
	return rm.CheckPermission(user, PermissionModerateContent)
}

// CanAccessLogs checks if user can access log files
func (rm *RBACManager) CanAccessLogs(user *models.User) bool {
	return rm.CheckPermission(user, PermissionLogsRead)
}

// CanManageConfig checks if user can manage configuration
func (rm *RBACManager) CanManageConfig(user *models.User) bool {
	return rm.CheckPermission(user, PermissionConfigWrite)
}

// CanAccessMetrics checks if user can access metrics
func (rm *RBACManager) CanAccessMetrics(user *models.User) bool {
	return rm.CheckPermission(user, PermissionMetricsRead)
}

// CanManageKillSwitch checks if user can manage kill switches
func (rm *RBACManager) CanManageKillSwitch(user *models.User) bool {
	return rm.CheckPermission(user, PermissionKillSwitchWrite)
}