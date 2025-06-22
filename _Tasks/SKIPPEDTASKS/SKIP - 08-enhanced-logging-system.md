# Enhanced Logging System

## Overview
Upgrade the logging system with structured logging, log levels, centralized log management, search capabilities, and integration with monitoring systems.

## Priority: Medium
**Estimated Effort:** 2-3 days

## Technical Requirements
- [ ] Structured logging with JSON format
- [ ] Multiple log levels and filtering
- [ ] Log aggregation and search
- [ ] Log rotation and archival
- [ ] Integration with external log systems

## Implementation Checklist

### Structured Logging Implementation
- [ ] Install logging dependencies (`github.com/sirupsen/logrus` or `go.uber.org/zap`)
- [ ] Create `backend/logging/logger.go` with structured logging
- [ ] Implement log formats:
  ```json
  {
    "timestamp": "2025-06-18T10:30:00Z",
    "level": "info",
    "service": "qt1-middleware",
    "component": "proxy",
    "user_id": "user123",
    "session_id": "session456",
    "request_id": "req-789",
    "message": "Request processed successfully",
    "duration_ms": 150,
    "status_code": 200,
    "tags": ["chat", "openai"],
    "metadata": {
      "provider": "openai",
      "model": "gpt-4",
      "tokens": 256
    }
  }
  ```
- [ ] Add contextual logging with request correlation IDs
- [ ] Implement log sampling for high-volume scenarios

### Log Levels & Categories
- [ ] Implement comprehensive log levels:
  - [ ] `TRACE` - Detailed execution flow
  - [ ] `DEBUG` - Development and troubleshooting
  - [ ] `INFO` - General application flow
  - [ ] `WARN` - Warning conditions
  - [ ] `ERROR` - Error conditions
  - [ ] `FATAL` - Critical failures
- [ ] Create log categories:
  - [ ] `audit` - Security and compliance events
  - [ ] `performance` - Performance metrics
  - [ ] `security` - Security-related events
  - [ ] `business` - Business logic events
  - [ ] `system` - System-level events

### Log Storage & Management
- [ ] Implement multiple log outputs:
  - [ ] File-based logging with rotation
  - [ ] Database logging for structured queries
  - [ ] Stdout/stderr for containerized environments
  - [ ] External log aggregation (ELK stack, Splunk)
- [ ] Add log rotation and compression
- [ ] Implement log retention policies
- [ ] Create log archival system

### Log Search & Analysis
- [ ] Create log search API endpoints:
  - [ ] `GET /api/logs/search` - Search logs with filters
  - [ ] `GET /api/logs/stats` - Log statistics and summaries
  - [ ] `GET /api/logs/export` - Export logs in various formats
- [ ] Implement log filtering by:
  - [ ] Time range
  - [ ] Log level
  - [ ] User ID
  - [ ] Session ID
  - [ ] Request ID
  - [ ] Component/service
  - [ ] Custom tags
- [ ] Add full-text search capabilities

### Enhanced Log Viewer Frontend
- [ ] Upgrade `Logs.tsx` component with advanced features:
  - [ ] Real-time log streaming
  - [ ] Advanced filtering interface
  - [ ] Log level color coding
  - [ ] Expandable log details
  - [ ] Export functionality
  - [ ] Search and highlight
- [ ] Add log analytics dashboard
- [ ] Implement log pattern recognition

### Security & Audit Logging
- [ ] Implement audit trail logging:
  ```go
  type AuditLog struct {
      Timestamp   time.Time `json:"timestamp"`
      UserID      string    `json:"user_id"`
      Action      string    `json:"action"`
      Resource    string    `json:"resource"`
      OldValue    string    `json:"old_value,omitempty"`
      NewValue    string    `json:"new_value,omitempty"`
      IPAddress   string    `json:"ip_address"`
      UserAgent   string    `json:"user_agent"`
      Success     bool      `json:"success"`
      FailReason  string    `json:"fail_reason,omitempty"`
  }
  ```
- [ ] Add PII masking for sensitive data
- [ ] Implement compliance logging (GDPR, SOX, etc.)
- [ ] Add tamper-evident logging

### Performance Logging
- [ ] Implement performance event logging
- [ ] Add request/response timing logs
- [ ] Track resource usage events
- [ ] Log database query performance
- [ ] Monitor external API call performance

### Error Tracking & Alerting
- [ ] Enhanced error logging with stack traces
- [ ] Error categorization and tagging
- [ ] Error rate monitoring and alerting
- [ ] Integration with error tracking services (Sentry)
- [ ] Automatic error escalation

### Log Configuration
```yaml
logging:
  level: "info"
  format: "json"  # json, text, logfmt
  
  outputs:
    - type: "file"
      path: "logs/qt1.log"
      rotation:
        max_size: "100MB"
        max_files: 30
        compress: true
    - type: "database"
      table: "logs"
      batch_size: 100
    - type: "stdout"
      enabled: true
      
  retention:
    file_logs: "30d"
    database_logs: "7d"
    audit_logs: "2y"
    
  sampling:
    enabled: true
    rate: 0.1  # Sample 10% of debug logs
    
  security:
    mask_pii: true
    encrypt_audit: true
    tamper_protection: true
```

### Log Metrics Integration
- [ ] Track logging metrics:
  - [ ] Log volume by level
  - [ ] Error rate trends
  - [ ] Log processing performance
  - [ ] Storage utilization
- [ ] Add logging health monitoring
- [ ] Create log-based alerting rules

### External Integration
- [ ] ELK Stack integration (Elasticsearch, Logstash, Kibana)
- [ ] Fluentd/Fluent Bit integration
- [ ] Prometheus metrics from logs
- [ ] Custom webhook integration for log events
- [ ] Cloud logging service integration (AWS CloudWatch, etc.)

### Log Utilities
- [ ] Create log analysis tools:
  - [ ] Log parser for common patterns
  - [ ] Error trend analysis
  - [ ] Performance bottleneck identification
  - [ ] User behavior analysis
- [ ] Add log replay capabilities for debugging
- [ ] Implement log-based testing tools

## API Endpoints
- [ ] `GET /api/logs` - Get paginated logs
- [ ] `GET /api/logs/search?q=query` - Search logs
- [ ] `GET /api/logs/stats` - Log statistics
- [ ] `GET /api/logs/levels` - Available log levels
- [ ] `POST /api/logs/export` - Export logs
- [ ] `GET /api/logs/audit` - Audit trail logs
- [ ] `DELETE /api/logs/cleanup` - Manual log cleanup

## Testing Requirements
- [ ] Unit tests for logging components
- [ ] Integration tests for log storage
- [ ] Performance tests for high-volume logging
- [ ] Log search functionality tests
- [ ] Log rotation and cleanup tests

## Acceptance Criteria
- [ ] Structured JSON logs with consistent format
- [ ] Real-time log streaming in admin interface
- [ ] Log search returns results within 2 seconds
- [ ] Log rotation works automatically
- [ ] Audit trail captures all security events
- [ ] Error logs include full context and stack traces
- [ ] Log storage handles 10,000+ logs per minute
- [ ] PII masking works correctly for sensitive data

## Dependencies
- [ ] Task #01 (WebSocket Integration) for real-time log streaming
- [ ] Task #04 (Database Integration) for log storage
- [ ] Task #05 (User Authentication) for audit logging

## Files to Modify/Create
- `backend/logging/logger.go` (new)
- `backend/logging/audit.go` (new)
- `backend/api/logs.go` (enhance)
- `backend/middleware/logging.go` (enhance)
- `frontend/src/components/Logs.tsx` (enhance)
- `frontend/src/components/LogViewer.tsx` (new)
- `backend/config.yaml` (extend logging section)