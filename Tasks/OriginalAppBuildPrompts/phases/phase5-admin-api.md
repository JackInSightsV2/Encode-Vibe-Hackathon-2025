# Phase 5: Admin API & Management

## Overview

Build comprehensive admin API endpoints and management interfaces for configuring, monitoring, and controlling QT-1. Includes authentication, real-time monitoring, configuration management, and administrative controls.

## Prerequisites

- Phase 1-4 completed successfully
- All providers working with unified interface
- Database schema established
- Core middleware pipeline operational

## Architecture

```
Admin API Architecture:
React Frontend → Admin API → Authentication → Business Logic → Database/Providers
                      ↓
              WebSocket Connections (Real-time updates)
```

## Project Structure Additions

```
backend/
├── api/
│   ├── server.go          # API server setup
│   ├── routes.go          # Route definitions
│   ├── middleware.go      # API middleware
│   └── handlers/
│       ├── auth.go        # Authentication endpoints
│       ├── config.go      # Configuration management
│       ├── logs.go        # Log retrieval and filtering
│       ├── providers.go   # Provider management
│       ├── killswitch.go  # Kill switch management
│       ├── health.go      # Health and monitoring
│       ├── users.go       # User management
│       └── websocket.go   # Real-time updates
├── auth/
│   ├── jwt.go             # JWT token management
│   ├── sessions.go        # Session management
│   ├── middleware.go      # Authentication middleware
│   └── permissions.go     # Role-based permissions
├── admin/
│   ├── dashboard.go       # Dashboard data aggregation
│   ├── monitoring.go      # System monitoring
│   └── notifications.go   # Alert system
└── websockets/
    ├── hub.go             # WebSocket connection hub
    ├── client.go          # WebSocket client management
    └── handlers.go        # WebSocket message handlers
```

## Tasks Checklist

### 1. Authentication System
- [ ] Create `auth/jwt.go`:
  - JWT token generation and validation
  - Token refresh mechanism
  - Secure token storage and rotation
  - Role-based access control (RBAC)

- [ ] Create `auth/sessions.go`:
  - Session management with SQLite storage
  - Session timeout and cleanup
  - Concurrent session limits
  - Session activity tracking

- [ ] Database schema for authentication:
  ```sql
  CREATE TABLE admin_users (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      username TEXT UNIQUE NOT NULL,
      email TEXT UNIQUE NOT NULL,
      password_hash TEXT NOT NULL,
      role TEXT NOT NULL DEFAULT 'admin',
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      last_login DATETIME,
      active BOOLEAN DEFAULT TRUE
  );

  CREATE TABLE admin_sessions (
      id TEXT PRIMARY KEY,
      user_id INTEGER NOT NULL,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      expires_at DATETIME NOT NULL,
      ip_address TEXT,
      user_agent TEXT,
      active BOOLEAN DEFAULT TRUE,
      FOREIGN KEY (user_id) REFERENCES admin_users (id)
  );
  ```

### 2. API Server Setup
- [ ] Create `api/server.go`:
  - HTTP server configuration
  - CORS handling for frontend
  - Request logging and monitoring
  - Graceful shutdown handling

- [ ] Create `api/routes.go`:
  - RESTful API route definitions
  - Route grouping and versioning
  - Middleware application per route
  - API documentation integration

- [ ] Create `api/middleware.go`:
  - Authentication middleware
  - Request rate limiting
  - Request validation
  - Error handling and recovery

### 3. Configuration Management API
- [ ] Create `api/handlers/config.go`:
  ```go
  // GET /api/config - Get current configuration
  // PUT /api/config - Update configuration
  // POST /api/config/validate - Validate configuration
  // GET /api/config/schema - Get configuration schema
  // POST /api/config/reload - Reload configuration
  ```

- [ ] Configuration endpoints:
  - Global settings management
  - Provider-specific configurations
  - Middleware settings
  - Feature toggles
  - Environment variable management

- [ ] Configuration validation:
  - Schema validation
  - Dependency checking
  - Provider availability validation
  - Security policy enforcement

### 4. Logging and Monitoring API
- [ ] Create `api/handlers/logs.go`:
  ```go
  // GET /api/logs - Get filtered logs
  // GET /api/logs/stats - Get log statistics
  // GET /api/logs/export - Export logs
  // DELETE /api/logs - Clear old logs
  // GET /api/logs/stream - Real-time log streaming
  ```

- [ ] Log filtering capabilities:
  - Date range filtering
  - User/session filtering
  - Provider/model filtering
  - Severity level filtering
  - Full-text search in log messages

- [ ] Log analytics:
  - Request volume metrics
  - Error rate analysis
  - Provider performance stats
  - User behavior patterns
  - Cost analysis per user/session

### 5. Provider Management API
- [ ] Create `api/handlers/providers.go`:
  ```go
  // GET /api/providers - List all providers
  // GET /api/providers/{name} - Get provider details
  // PUT /api/providers/{name} - Update provider config
  // POST /api/providers/{name}/test - Test provider connection
  // GET /api/providers/{name}/models - List available models
  // GET /api/providers/{name}/stats - Get provider statistics
  ```

- [ ] Provider health monitoring:
  - Real-time status checking
  - Response time monitoring
  - Error rate tracking
  - API quota monitoring
  - Cost tracking per provider

### 6. Kill Switch Management API
- [ ] Create `api/handlers/killswitch.go`:
  ```go
  // GET /api/killswitch - List all blocked entities
  // POST /api/killswitch/block - Block user/session/IP
  // DELETE /api/killswitch/unblock/{id} - Unblock entity
  // GET /api/killswitch/history - Get blocking history
  // POST /api/killswitch/bulk - Bulk block operations
  ```

- [ ] Kill switch features:
  - Immediate blocking capabilities
  - Temporary vs permanent blocks
  - Bulk operations
  - Block reason tracking
  - Automatic expiry handling

### 7. User Management API
- [ ] Create `api/handlers/users.go`:
  ```go
  // GET /api/users - List admin users
  // POST /api/users - Create new admin user
  // PUT /api/users/{id} - Update user
  // DELETE /api/users/{id} - Delete user
  // POST /api/users/{id}/reset-password - Reset password
  ```

- [ ] User management features:
  - Role-based permissions
  - Password policy enforcement
  - User activity tracking
  - Account lockout mechanisms
  - Audit trail for user actions

### 8. Health and Monitoring API
- [ ] Create `api/handlers/health.go`:
  ```go
  // GET /api/health - Basic health check
  // GET /api/health/detailed - Detailed system status
  // GET /api/health/providers - Provider health status
  // GET /api/health/database - Database connection status
  // GET /api/health/metrics - System metrics
  ```

- [ ] System monitoring:
  - CPU and memory usage
  - Database performance metrics
  - Request processing times
  - Error rates and patterns
  - Provider availability status

### 9. Real-time Updates with WebSockets
- [ ] Create `websockets/hub.go`:
  - WebSocket connection management
  - Message broadcasting
  - Connection cleanup
  - Authentication for WebSocket connections

- [ ] Create `websockets/client.go`:
  - Individual client management
  - Message queuing
  - Connection state tracking
  - Heartbeat and ping/pong handling

- [ ] Real-time features:
  - Live log streaming
  - Real-time metrics updates
  - Provider status changes
  - Configuration change notifications
  - Kill switch activation alerts

### 10. Dashboard Data Aggregation
- [ ] Create `admin/dashboard.go`:
  - Real-time metrics aggregation
  - Historical data analysis
  - Performance trend calculation
  - Alert threshold monitoring

- [ ] Dashboard metrics:
  - Requests per minute/hour/day
  - Success/failure rates
  - Average response times
  - Cost per time period
  - User activity levels
  - Provider distribution

### 11. Alert and Notification System
- [ ] Create `admin/notifications.go`:
  - Alert rule definition
  - Threshold-based alerting
  - Email/webhook notifications
  - Alert escalation policies

- [ ] Alert types:
  - High error rates
  - Provider failures
  - Cost threshold exceeded
  - Unusual usage patterns
  - System resource alerts

### 12. API Documentation
- [ ] OpenAPI/Swagger integration:
  - Automatic API documentation generation
  - Interactive API explorer
  - Request/response examples
  - Authentication flow documentation

- [ ] Documentation endpoints:
  ```go
  // GET /api/docs - API documentation
  // GET /api/docs/swagger.json - OpenAPI specification
  // GET /api/docs/postman - Postman collection
  ```

### 13. Security and Validation
- [ ] Input validation:
  - Request payload validation
  - SQL injection prevention
  - XSS protection
  - CSRF protection

- [ ] API security:
  - Rate limiting per endpoint
  - IP whitelisting for admin endpoints
  - Audit logging for all admin actions
  - Secure headers (HSTS, CSP, etc.)

### 14. Database Migrations for Admin Features
- [ ] Additional tables for admin functionality:
  ```sql
  CREATE TABLE admin_audit_log (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      user_id INTEGER NOT NULL,
      action TEXT NOT NULL,
      resource TEXT NOT NULL,
      resource_id TEXT,
      old_value TEXT,
      new_value TEXT,
      ip_address TEXT,
      user_agent TEXT,
      timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (user_id) REFERENCES admin_users (id)
  );

  CREATE TABLE system_alerts (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      alert_type TEXT NOT NULL,
      severity TEXT NOT NULL,
      title TEXT NOT NULL,
      message TEXT NOT NULL,
      metadata TEXT, -- JSON
      acknowledged BOOLEAN DEFAULT FALSE,
      acknowledged_by INTEGER,
      acknowledged_at DATETIME,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      resolved_at DATETIME
  );

  CREATE TABLE api_keys (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL,
      key_hash TEXT UNIQUE NOT NULL,
      user_id INTEGER NOT NULL,
      permissions TEXT NOT NULL, -- JSON
      last_used DATETIME,
      expires_at DATETIME,
      active BOOLEAN DEFAULT TRUE,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (user_id) REFERENCES admin_users (id)
  );
  ```

### 15. Testing and Validation
- [ ] API endpoint tests:
  - Authentication flow testing
  - CRUD operation tests
  - Permission-based access tests
  - Error handling validation

- [ ] Integration tests:
  - Frontend-backend integration
  - WebSocket functionality
  - Real-time update testing
  - Performance under load

- [ ] Security tests:
  - Authentication bypass attempts
  - Authorization testing
  - Input validation testing
  - Rate limiting validation

### 16. Configuration Templates
- [ ] Create default admin configuration:
  ```yaml
  admin_api:
    enabled: true
    port: 8081
    host: "localhost"
    cors_origins: ["http://localhost:3000"]
    jwt_secret_env: "JWT_SECRET"
    session_timeout: "24h"
    rate_limit:
      requests_per_minute: 100
      burst_size: 20
    
  authentication:
    password_policy:
      min_length: 8
      require_uppercase: true
      require_lowercase: true
      require_numbers: true
      require_symbols: true
    session_limits:
      max_concurrent_sessions: 5
      session_timeout: "24h"
    
  monitoring:
    metrics_retention_days: 30
    log_retention_days: 90
    alert_cooldown: "15m"
    
  websocket:
    enabled: true
    ping_interval: "30s"
    pong_timeout: "10s"
    max_connections: 100
  ```

## Success Criteria

- [ ] Complete admin API with all CRUD operations
- [ ] Secure authentication and authorization system
- [ ] Real-time monitoring and alerting capabilities
- [ ] WebSocket integration for live updates
- [ ] Comprehensive logging and audit trails
- [ ] Provider management and health monitoring
- [ ] Kill switch management interface
- [ ] Performance metrics and analytics
- [ ] API documentation and testing tools
- [ ] Security measures against common attacks

## API Endpoints Summary

### Authentication
- `POST /api/auth/login` - Admin login
- `POST /api/auth/logout` - Admin logout
- `POST /api/auth/refresh` - Refresh JWT token
- `GET /api/auth/me` - Get current user info

### Configuration
- `GET /api/config` - Get configuration
- `PUT /api/config` - Update configuration
- `POST /api/config/validate` - Validate config
- `POST /api/config/reload` - Reload configuration

### Logs & Monitoring
- `GET /api/logs` - Get filtered logs
- `GET /api/logs/stats` - Log statistics
- `GET /api/health` - System health
- `GET /api/metrics` - System metrics

### Provider Management
- `GET /api/providers` - List providers
- `PUT /api/providers/{name}` - Update provider
- `POST /api/providers/{name}/test` - Test provider

### Kill Switch
- `GET /api/killswitch` - List blocks
- `POST /api/killswitch/block` - Block entity
- `DELETE /api/killswitch/unblock/{id}` - Unblock

### User Management
- `GET /api/users` - List admin users
- `POST /api/users` - Create user
- `PUT /api/users/{id}` - Update user

## Performance Targets

- [ ] API response time: < 100ms (95th percentile)
- [ ] WebSocket connection handling: > 100 concurrent connections
- [ ] Database query performance: < 50ms average
- [ ] Real-time update latency: < 500ms
- [ ] Authentication: < 50ms per request

## Next Phase

After completion, proceed to Phase 6: Testing & Production Readiness for comprehensive testing, optimization, and deployment preparation.