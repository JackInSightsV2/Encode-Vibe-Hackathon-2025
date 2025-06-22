# User Authentication & Authorization - Refined Implementation Cycles

## Overview
Break down authentication system into 6 manageable cycles, from basic auth to advanced RBAC.

---

## **Cycle 5A: Basic JWT Authentication Setup**
**Duration:** 4-6 hours | **Priority:** Critical

### Prerequisites
- Database integration (Task 4A-4C) completed
- Basic understanding of JWT tokens

### Implementation Tasks
- [ ] Install JWT dependencies (`github.com/golang-jwt/jwt/v5`)
- [ ] Create `backend/auth/` directory structure
- [ ] Implement JWT token generation and validation
- [ ] Add password hashing using bcrypt
- [ ] Create basic login endpoint

### Code Deliverables
```go
// backend/auth/jwt.go
type JWTManager struct {
    secretKey string
    tokenTTL  time.Duration
}

type Claims struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

func (jm *JWTManager) GenerateToken(user *models.User) (string, error) {
    // Generate JWT token with user claims
}

func (jm *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    // Validate and parse JWT token
}
```

### Testing Requirements
- [ ] Unit tests for JWT generation
- [ ] Unit tests for token validation
- [ ] Test token expiration
- [ ] Test password hashing

### Acceptance Criteria
- [ ] JWT tokens are generated correctly
- [ ] Token validation works with valid tokens
- [ ] Expired tokens are rejected
- [ ] Password hashing is secure (bcrypt)
- [ ] Login endpoint returns valid JWT

### Risk Mitigation
- Use strong secret key for JWT signing
- Test token validation thoroughly
- Implement proper error handling

---

## **Cycle 5B: Authentication Middleware & User Model**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 5A completed and tested
- User repository from database integration

### Implementation Tasks
- [ ] Enhance user model with authentication fields
- [ ] Create authentication middleware
- [ ] Implement login/logout endpoints
- [ ] Add password validation
- [ ] Create user registration endpoint

### Code Deliverables
```go
// Enhanced user model
type User struct {
    ID           int       `json:"id" db:"id"`
    Username     string    `json:"username" db:"username"`
    Email        string    `json:"email" db:"email"`
    PasswordHash string    `json:"-" db:"password_hash"`
    Role         string    `json:"role" db:"role"`
    Active       bool      `json:"active" db:"active"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    LastLogin    time.Time `json:"last_login" db:"last_login"`
}

// backend/auth/middleware.go
func AuthMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract and validate JWT token
            // Add user to request context
            next.ServeHTTP(w, r)
        })
    }
}
```

### Testing Requirements
- [ ] Unit tests for authentication middleware
- [ ] Test login endpoint with valid/invalid credentials
- [ ] Test user registration
- [ ] Integration tests with database

### Acceptance Criteria
- [ ] Middleware validates tokens correctly
- [ ] Login works with correct credentials
- [ ] Login fails with incorrect credentials
- [ ] User registration creates new users
- [ ] Password requirements are enforced

### Risk Mitigation
- Validate all user input
- Use secure password requirements
- Test edge cases thoroughly

---

## **Cycle 5C: Role-Based Access Control (RBAC)**
**Duration:** 6-7 hours | **Priority:** High

### Prerequisites
- Cycle 5B completed and tested
- Understanding of RBAC concepts

### Implementation Tasks
- [ ] Define user roles and permissions
- [ ] Create permission matrix
- [ ] Implement authorization middleware
- [ ] Add role-based endpoint protection
- [ ] Create role management functions

### Code Deliverables
```go
// backend/auth/rbac.go
type Role string

const (
    RoleSuperAdmin Role = "super_admin"
    RoleAdmin      Role = "admin"
    RoleOperator   Role = "operator"
    RoleViewer     Role = "viewer"
)

type Permission string

const (
    PermissionUserManage   Permission = "user:manage"
    PermissionConfigRead   Permission = "config:read"
    PermissionConfigWrite  Permission = "config:write"
    PermissionLogsRead     Permission = "logs:read"
    PermissionKillSwitch   Permission = "killswitch:manage"
)

type PermissionMatrix map[Role][]Permission

func (pm PermissionMatrix) HasPermission(role Role, permission Permission) bool {
    // Check if role has specific permission
}

func RequirePermission(permission Permission) func(http.Handler) http.Handler {
    // Middleware to check specific permission
}
```

### Testing Requirements
- [ ] Unit tests for permission checking
- [ ] Test role-based access control
- [ ] Test unauthorized access attempts
- [ ] Integration tests with various roles

### Acceptance Criteria
- [ ] Roles are properly defined and enforced
- [ ] Permission matrix works correctly
- [ ] Unauthorized users are rejected
- [ ] Different roles have appropriate access
- [ ] Permission middleware protects endpoints

### Risk Mitigation
- Test all permission combinations
- Use deny-by-default security model
- Log all authorization failures

---

## **Cycle 5D: Session Management & Security**
**Duration:** 5-6 hours | **Priority:** Medium

### Prerequisites
- Cycle 5C completed and tested
- Database tables for sessions

### Implementation Tasks
- [ ] Create session management system
- [ ] Implement session storage in database
- [ ] Add session cleanup and expiration
- [ ] Implement concurrent session limits
- [ ] Add security features (account lockout, etc.)

### Code Deliverables
```go
// backend/auth/session.go
type Session struct {
    ID        string    `json:"id" db:"id"`
    UserID    int       `json:"user_id" db:"user_id"`
    Token     string    `json:"token" db:"token"`
    ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    IPAddress string    `json:"ip_address" db:"ip_address"`
    UserAgent string    `json:"user_agent" db:"user_agent"`
}

type SessionManager struct {
    db           *sql.DB
    maxSessions  int
    sessionTTL   time.Duration
    cleanupTimer *time.Timer
}

func (sm *SessionManager) CreateSession(userID int, ipAddress, userAgent string) (*Session, error) {
    // Create new session with cleanup of old sessions
}
```

### Testing Requirements
- [ ] Unit tests for session management
- [ ] Test session expiration
- [ ] Test concurrent session limits
- [ ] Test session cleanup

### Acceptance Criteria
- [ ] Sessions are created and validated correctly
- [ ] Old sessions are cleaned up automatically
- [ ] Concurrent session limits are enforced
- [ ] Session security features work
- [ ] Session data is properly stored

### Risk Mitigation
- Use secure session ID generation
- Clean up expired sessions regularly
- Monitor session table size

---

## **Cycle 5E: API Key Management**
**Duration:** 4-5 hours | **Priority:** Medium

### Prerequisites
- Cycle 5D completed and tested
- Understanding of API key authentication

### Implementation Tasks
- [ ] Create API key model and storage
- [ ] Implement API key generation
- [ ] Add API key authentication
- [ ] Create API key management endpoints
- [ ] Add API key usage tracking

### Code Deliverables
```go
// backend/models/api_key.go
type APIKey struct {
    ID        string    `json:"id" db:"id"`
    UserID    int       `json:"user_id" db:"user_id"`
    Name      string    `json:"name" db:"name"`
    KeyHash   string    `json:"-" db:"key_hash"`
    Active    bool      `json:"active" db:"active"`
    LastUsed  time.Time `json:"last_used" db:"last_used"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// backend/auth/api_key.go
func GenerateAPIKey(userID int, name string) (*APIKey, string, error) {
    // Generate secure API key
}

func ValidateAPIKey(keyString string) (*APIKey, error) {
    // Validate API key and update last used
}
```

### Testing Requirements
- [ ] Unit tests for API key generation
- [ ] Test API key validation
- [ ] Test API key authentication
- [ ] Test key management endpoints

### Acceptance Criteria
- [ ] API keys are generated securely
- [ ] API key authentication works
- [ ] Key usage is tracked correctly
- [ ] Key management endpoints function
- [ ] Expired keys are handled properly

### Risk Mitigation
- Use cryptographically secure key generation
- Hash API keys in storage
- Implement proper key rotation

---

## **Cycle 5F: Frontend Authentication UI**
**Duration:** 6-8 hours | **Priority:** Medium

### Prerequisites
- Cycles 5A-5E completed and tested
- Frontend development environment ready

### Implementation Tasks
- [ ] Create authentication context provider
- [ ] Implement login/logout forms
- [ ] Add protected route wrapper
- [ ] Create user management interface
- [ ] Add role-based UI element hiding

### Code Deliverables
```typescript
// frontend/src/contexts/AuthContext.tsx
interface AuthContextType {
    user: User | null;
    login: (username: string, password: string) => Promise<void>;
    logout: () => void;
    hasPermission: (permission: string) => boolean;
    loading: boolean;
}

// frontend/src/components/auth/LoginForm.tsx
const LoginForm: React.FC = () => {
    const [credentials, setCredentials] = useState({ username: '', password: '' });
    const { login } = useAuth();
    
    const handleSubmit = async (e: React.FormEvent) => {
        // Handle login form submission
    };
};

// frontend/src/components/auth/ProtectedRoute.tsx
const ProtectedRoute: React.FC<{ children: React.ReactNode; permission?: string }> = ({
    children,
    permission
}) => {
    // Protect routes based on authentication and permissions
};
```

### Testing Requirements
- [ ] Unit tests for auth components
- [ ] Test login/logout functionality
- [ ] Test protected routes
- [ ] Integration tests with backend

### Acceptance Criteria
- [ ] Login form works correctly
- [ ] Authentication state is managed properly
- [ ] Protected routes require authentication
- [ ] Role-based UI elements work
- [ ] Logout clears authentication state

### Risk Mitigation
- Validate all form inputs
- Handle authentication errors gracefully
- Test with various user roles

---

## **Integration Testing**
**Duration:** 3-4 hours

### Comprehensive Testing
- [ ] End-to-end authentication flow
- [ ] Role-based access control testing
- [ ] Session management testing
- [ ] API key authentication testing
- [ ] Security penetration testing

### Success Metrics
- [ ] Authentication completes in <2 seconds
- [ ] Authorization checks complete in <50ms
- [ ] Session management handles 1000+ concurrent users
- [ ] All security tests pass
- [ ] UI provides smooth user experience

---

## **Security Checklist**
- [ ] Passwords are properly hashed (bcrypt)
- [ ] JWT tokens use strong secrets
- [ ] API keys are generated securely
- [ ] Sessions are properly managed
- [ ] Authorization is deny-by-default
- [ ] All inputs are validated
- [ ] Security events are logged

---

## **Rollback Plan**
If any cycle fails:
1. Disable authentication temporarily (admin-only mode)
2. Revert to previous user management system
3. Use basic HTTP auth as fallback
4. Disable role-based features if RBAC fails