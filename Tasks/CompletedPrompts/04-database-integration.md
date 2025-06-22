# Database Integration & Data Persistence

## Overview
Implement comprehensive database integration for persistent storage of logs, user data, configuration, metrics, and system state with migration capabilities.

## Priority: Medium
**Estimated Effort:** 3-4 days

## Technical Requirements
- [ ] Database abstraction layer
- [ ] Schema migrations system
- [ ] Connection pooling and optimization
- [ ] Data models and relationships
- [ ] Backup and recovery procedures

## Implementation Checklist

### Database Setup & Configuration
- [ ] Choose database system (PostgreSQL for production, SQLite for development)
- [ ] Add database dependencies (`database/sql`, `github.com/lib/pq` for PostgreSQL)
- [ ] Create database configuration in `config.yaml`
- [ ] Implement connection management with pooling
- [ ] Add database health checks

### Schema Design & Migration System
- [ ] Create `backend/database/migrations/` directory
- [ ] Design database schema:
  - [ ] `users` table (id, username, role, api_key, created_at)
  - [ ] `sessions` table (id, user_id, session_token, expires_at)
  - [ ] `requests` table (id, user_id, session_id, request_data, response_data, timestamp)
  - [ ] `moderation_logs` table (id, request_id, action, reason, severity, timestamp)
  - [ ] `system_config` table (key, value, updated_at, updated_by)
  - [ ] `metrics` table (timestamp, metric_name, value, tags)
  - [ ] `kill_switch_entries` table (id, type, value, reason, created_by, created_at)
- [ ] Implement migration runner
- [ ] Create initial migration files

### Database Access Layer
- [ ] Create `backend/database/db.go` with connection management
- [ ] Implement repository pattern for data access:
  - [ ] `UserRepository` for user management
  - [ ] `SessionRepository` for session handling
  - [ ] `RequestRepository` for request logging
  - [ ] `ConfigRepository` for configuration storage
  - [ ] `MetricsRepository` for metrics persistence
- [ ] Add transaction support for complex operations
- [ ] Implement query optimization and indexing

### Data Models
- [ ] Create `backend/models/` directory with struct definitions:
  ```go
  type User struct {
      ID       int       `json:"id" db:"id"`
      Username string    `json:"username" db:"username"`
      Role     string    `json:"role" db:"role"`
      APIKey   string    `json:"api_key,omitempty" db:"api_key"`
      CreatedAt time.Time `json:"created_at" db:"created_at"`
  }
  ```
- [ ] Add validation tags and methods
- [ ] Implement JSON marshaling/unmarshaling

### Configuration Management
- [ ] Migrate configuration storage to database
- [ ] Implement configuration versioning
- [ ] Add configuration backup/restore functionality
- [ ] Create configuration audit trail

### Logging Integration
- [ ] Enhance logging to use database storage
- [ ] Implement structured logging with searchable fields
- [ ] Add log retention policies
- [ ] Create log analysis and search capabilities

### Performance Optimization
- [ ] Implement database connection pooling
- [ ] Add query performance monitoring
- [ ] Create database indexes for common queries
- [ ] Implement data archival for old records
- [ ] Add read replica support (optional)

### Database Configuration
```yaml
database:
  type: "postgresql"  # or "sqlite"
  host: "localhost"
  port: 5432
  name: "qt1_middleware"
  username: "${DB_USERNAME}"
  password: "${DB_PASSWORD}"
  ssl_mode: "require"
  max_connections: 25
  max_idle_connections: 5
  connection_lifetime: "1h"
  
  # Migration settings
  migrations:
    enabled: true
    directory: "./database/migrations"
    
  # Retention policies
  retention:
    requests: "30d"
    metrics: "7d"
    logs: "90d"
```

### Admin Interface Integration
- [ ] Update frontend to display database-stored data
- [ ] Add user management interface
- [ ] Create configuration management UI
- [ ] Implement data export/import functionality
- [ ] Add database health monitoring to dashboard

## Migration Files Structure
```
backend/database/migrations/
├── 001_initial_schema.up.sql
├── 001_initial_schema.down.sql
├── 002_add_metrics_table.up.sql
├── 002_add_metrics_table.down.sql
└── ...
```

## Testing Requirements
- [ ] Unit tests for repository implementations
- [ ] Integration tests with real database
- [ ] Migration testing (up and down)
- [ ] Performance tests for database operations
- [ ] Data consistency and integrity tests

## Acceptance Criteria
- [ ] All system data persists across restarts
- [ ] Database migrations run automatically on startup
- [ ] Query performance under 100ms for 95% of operations
- [ ] Connection pooling handles 100+ concurrent connections
- [ ] Data integrity maintained under high load
- [ ] Backup and restore procedures work correctly
- [ ] Database health monitoring integrated into dashboard

## Dependencies
- [ ] Task #03 (Metrics Dashboard) for metrics persistence
- [ ] Task #05 (User Management) for user data storage

## Files to Modify/Create
- `backend/database/db.go` (new)
- `backend/database/migrations/` (new directory)
- `backend/models/` (new directory)
- `backend/repositories/` (new directory)
- `backend/config.yaml` (extend)
- `backend/go.mod` (add database dependencies)
- Migration files for schema setup