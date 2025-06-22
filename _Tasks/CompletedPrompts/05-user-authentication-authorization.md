# User Authentication & Authorization System

## Overview
Implement comprehensive user management with role-based access control, API key management, session handling, and multi-user support for the admin interface.

## Priority: High
**Estimated Effort:** 3-4 days

## Technical Requirements
- [ ] JWT-based authentication
- [ ] Role-based access control (RBAC)
- [ ] API key management
- [ ] Session management
- [ ] Password security and hashing

## Implementation Checklist

### Authentication Backend
- [ ] Install JWT dependencies (`github.com/golang-jwt/jwt/v5`)
- [ ] Create `backend/auth/` directory structure
- [ ] Implement JWT token generation and validation
- [ ] Add password hashing using bcrypt
- [ ] Create authentication middleware
- [ ] Implement login/logout endpoints

### User Management System
- [ ] Create user model with roles:
  ```go
  type User struct {
      ID          int       `json:"id"`
      Username    string    `json:"username"`
      Email       string    `json:"email"`
      PasswordHash string   `json:"-"`
      Role        string    `json:"role"`
      APIKey      string    `json:"api_key,omitempty"`
      Active      bool      `json:"active"`
      CreatedAt   time.Time `json:"created_at"`
      LastLogin   time.Time `json:"last_login"`
  }
  ```
- [ ] Implement user CRUD operations
- [ ] Add user validation and constraints
- [ ] Create default admin user setup

### Role-Based Access Control
- [ ] Define user roles:
  - [ ] `super_admin` - Full system access
  - [ ] `admin` - User and configuration management
  - [ ] `operator` - Monitoring and basic operations
  - [ ] `viewer` - Read-only access
- [ ] Implement permission matrix:
  ```yaml
  permissions:
    super_admin: ["*"]
    admin: ["users:*", "config:*", "logs:read", "killswitch:*"]
    operator: ["config:read", "logs:read", "killswitch:manage"]
    viewer: ["dashboard:read", "logs:read"]
  ```
- [ ] Create authorization middleware
- [ ] Add permission checks to API endpoints

### API Key Management
- [ ] Generate secure API keys for users
- [ ] Implement API key authentication
- [ ] Add API key rotation capability
- [ ] Create usage tracking for API keys
- [ ] Implement rate limiting per API key

### Session Management
- [ ] Create session storage (database-backed)
- [ ] Implement session cleanup and expiration
- [ ] Add concurrent session limits
- [ ] Implement "remember me" functionality
- [ ] Add session activity tracking

### Security Features
- [ ] Implement password complexity requirements
- [ ] Add account lockout after failed attempts
- [ ] Create audit logging for authentication events
- [ ] Add two-factor authentication preparation
- [ ] Implement secure password reset

### Authentication API Endpoints
- [ ] `POST /api/auth/login` - User login
- [ ] `POST /api/auth/logout` - User logout
- [ ] `POST /api/auth/refresh` - Token refresh
- [ ] `GET /api/auth/me` - Current user info
- [ ] `POST /api/auth/change-password` - Password change
- [ ] `POST /api/users` - Create user (admin only)
- [ ] `GET /api/users` - List users (admin only)
- [ ] `PUT /api/users/:id` - Update user (admin only)
- [ ] `DELETE /api/users/:id` - Delete user (super_admin only)
- [ ] `POST /api/users/:id/api-key` - Generate API key

### Frontend Authentication
- [ ] Create authentication context provider
- [ ] Implement login/logout forms
- [ ] Add protected route wrapper
- [ ] Create user management interface
- [ ] Add role-based UI element hiding
- [ ] Implement token refresh logic

### User Management Interface
- [ ] Create `UserManagement.tsx` component
- [ ] Add user creation/editing forms
- [ ] Implement user list with search/filter
- [ ] Add role assignment interface
- [ ] Create API key management UI
- [ ] Add user activity logs display

### Configuration Updates
```yaml
auth:
  jwt:
    secret: "${JWT_SECRET}"
    expiration: "24h"
    refresh_expiration: "7d"
  
  password:
    min_length: 8
    require_special: true
    require_number: true
    require_uppercase: true
  
  session:
    max_concurrent: 5
    cleanup_interval: "1h"
    
  lockout:
    max_attempts: 5
    lockout_duration: "15m"
```

### Database Schema Updates
```sql
-- Users table (extend existing or create new)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    api_key VARCHAR(64) UNIQUE,
    active BOOLEAN DEFAULT true,
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP
);

-- Sessions table
CREATE TABLE sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_activity TIMESTAMP DEFAULT NOW()
);

-- Auth logs table
CREATE TABLE auth_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Testing Requirements
- [ ] Unit tests for authentication functions
- [ ] Integration tests for auth endpoints
- [ ] Frontend tests for auth components
- [ ] Security tests for token validation
- [ ] Role-based access control tests

## Acceptance Criteria
- [ ] Multi-user login/logout works correctly
- [ ] Role-based permissions enforced on all endpoints
- [ ] API keys work for authentication
- [ ] Session management prevents unauthorized access
- [ ] Password security requirements enforced
- [ ] Admin can manage users through the interface
- [ ] Audit trail captures all authentication events
- [ ] Token refresh works seamlessly

## Dependencies
- [ ] Task #04 (Database Integration) for user storage
- [ ] Enhanced middleware for permission checks

## Files to Modify/Create
- `backend/auth/` (new directory)
- `backend/middleware/auth.go` (new)
- `backend/api/auth.go` (new)
- `backend/api/users.go` (new)
- `frontend/src/contexts/AuthContext.tsx` (new)
- `frontend/src/components/auth/` (new directory)
- `frontend/src/components/UserManagement.tsx` (new)
- Database migration files for auth tables