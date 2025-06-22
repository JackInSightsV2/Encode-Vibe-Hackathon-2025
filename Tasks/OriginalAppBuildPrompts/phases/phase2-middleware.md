# Phase 2: Core Middleware Pipeline

## Overview

Implement the core middleware pipeline for request processing, including content moderation, logging, validation, and basic filtering capabilities.

## Prerequisites

- Phase 1 completed successfully
- Server running with basic OpenAI integration
- Database schema established

## Architecture

```
Request Flow:
Client → Authentication → Validation → Moderation → Logging → OpenAI → Response → Logging → Client
```

## Project Structure Additions

```
backend/
├── middleware/
│   ├── auth.go            # Request authentication
│   ├── validation.go      # Input validation
│   ├── moderation.go      # Content moderation
│   ├── logging.go         # Request/response logging
│   └── pipeline.go        # Middleware pipeline orchestration
├── filters/
│   ├── regex.go           # Regex-based filtering
│   ├── keywords.go        # Keyword filtering
│   └── content.go         # Content analysis
└── data/
    ├── blocked_words.txt  # List of blocked terms
    └── moderation_rules.yaml # Moderation configuration
```

## Tasks Checklist

### 1. Middleware Architecture
- [ ] Create `middleware/pipeline.go`:
  - `MiddlewareChain` type for chaining middleware
  - `ProcessRequest()` function to run request through pipeline
  - `ProcessResponse()` function for response processing
  - Error handling and recovery mechanisms

- [ ] Define middleware interface:
  ```go
  type Middleware interface {
      ProcessRequest(ctx context.Context, req *Request) (*Request, error)
      ProcessResponse(ctx context.Context, resp *Response) (*Response, error)
      Name() string
      Enabled() bool
  }
  ```

### 2. Request Validation
- [ ] Create `middleware/validation.go`:
  - Validate required fields (user_id, session_id, message)
  - Check message length limits
  - Sanitize input data
  - Rate limiting per user/session
  - Request size validation

- [ ] Add validation rules to config:
  ```yaml
  validation:
    max_message_length: 4000
    max_requests_per_minute: 60
    required_fields: ["user_id", "session_id", "message"]
    allowed_content_types: ["application/json"]
  ```

### 3. Content Moderation System
- [ ] Create `filters/regex.go`:
  - Load regex patterns from configuration
  - Pattern matching for harmful content
  - Category-based filtering (violence, harassment, NSFW)
  - Performance-optimized regex compilation

- [ ] Create `filters/keywords.go`:
  - Load blocked words from file/database
  - Efficient keyword matching (trie or similar)
  - Support for exact match and fuzzy matching
  - Category-based keyword filtering

- [ ] Create `middleware/moderation.go`:
  - Combine multiple filtering approaches
  - OpenAI Moderation API integration
  - Configurable moderation levels (strict, moderate, permissive)
  - Custom response messages for blocked content

### 4. Enhanced Logging System
- [ ] Create `middleware/logging.go`:
  - Pre-request logging (incoming request details)
  - Post-request logging (response and processing time)
  - Structured logging with correlation IDs
  - Performance metrics collection

- [ ] Update database schema for detailed logging:
  ```sql
  ALTER TABLE log_entries ADD COLUMN correlation_id TEXT;
  ALTER TABLE log_entries ADD COLUMN processing_time_ms INTEGER;
  ALTER TABLE log_entries ADD COLUMN moderation_score REAL;
  ALTER TABLE log_entries ADD COLUMN filter_results TEXT; -- JSON
  ```

### 5. Kill Switch Implementation
- [ ] Create `middleware/killswitch.go`:
  - Check blocked users/sessions before processing
  - Support for temporary and permanent blocks
  - IP-based blocking capability
  - Automated blocking based on violation patterns

- [ ] Database operations for kill switch:
  - `IsBlocked(userID, sessionID, ip)` - Check if entity is blocked
  - `BlockEntity(type, value, reason, duration)` - Add block
  - `UnblockEntity(type, value)` - Remove block
  - `GetBlockedEntities()` - List all blocks

### 6. Configuration Management
- [ ] Extend configuration system:
  ```yaml
  moderation:
    enabled: true
    use_openai_api: true
    strict_mode: false
    blocked_categories: ["violence", "harassment", "self-harm"]
    custom_rules_file: "data/moderation_rules.yaml"
    
  filtering:
    regex_enabled: true
    keyword_filtering: true
    blocked_words_file: "data/blocked_words.txt"
    
  rate_limiting:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
  ```

- [ ] Create `data/moderation_rules.yaml`:
  - Custom moderation rules
  - Severity levels
  - Custom response messages
  - Rule priorities

### 7. Response Processing
- [ ] Implement response middleware:
  - Filter OpenAI responses for harmful content
  - Add moderation headers to responses
  - Response size validation
  - Response time tracking

- [ ] Response filtering:
  - Check for leaked sensitive information
  - Ensure responses align with moderation policies
  - Add transparency indicators when content is filtered

### 8. Error Handling & Recovery
- [ ] Comprehensive error handling:
  - Graceful degradation when moderation APIs fail
  - Fallback to local filtering when external services are down
  - Circuit breaker pattern for external API calls
  - Detailed error logging and monitoring

- [ ] Error response standardization:
  ```go
  type ErrorResponse struct {
      Error     string    `json:"error"`
      Code      string    `json:"code"`
      Timestamp time.Time `json:"timestamp"`
      RequestID string    `json:"request_id"`
      Details   string    `json:"details,omitempty"`
  }
  ```

### 9. Performance Optimization
- [ ] Implement caching:
  - Cache moderation results for repeated content
  - Cache blocked entity lookups
  - Cache configuration values
  - TTL-based cache invalidation

- [ ] Connection pooling:
  - Database connection pooling
  - HTTP client connection reuse
  - Resource cleanup and management

### 10. Metrics and Monitoring
- [ ] Add metrics collection:
  - Request processing times
  - Moderation success/failure rates
  - Filter hit rates
  - Error rates by type

- [ ] Health check enhancements:
  - Database connectivity check
  - External API health checks
  - Middleware pipeline status
  - Resource usage metrics

### 11. Testing & Validation
- [ ] Unit tests for each middleware component:
  - Validation middleware tests
  - Moderation filter tests
  - Kill switch functionality tests
  - Pipeline integration tests

- [ ] Integration tests:
  - End-to-end request processing
  - Error scenario handling
  - Performance under load
  - Moderation accuracy tests

- [ ] Test data creation:
  - Sample blocked content for testing
  - Test user/session data
  - Moderation test cases

### 12. Documentation
- [ ] API documentation:
  - Request/response formats
  - Error codes and messages
  - Moderation levels and policies
  - Rate limiting information

- [ ] Configuration documentation:
  - All configuration options explained
  - Example configurations for different use cases
  - Troubleshooting guide

## Success Criteria

- [ ] All requests pass through middleware pipeline
- [ ] Content moderation blocks harmful content effectively
- [ ] Kill switch can block users/sessions immediately
- [ ] Comprehensive logging captures all request details
- [ ] Rate limiting prevents abuse
- [ ] System gracefully handles errors and failures
- [ ] Performance remains acceptable under load
- [ ] All middleware components are individually testable

## Performance Targets

- [ ] Request processing time: < 200ms (95th percentile)
- [ ] Moderation API calls: < 500ms timeout
- [ ] Database operations: < 50ms average
- [ ] Memory usage: < 100MB under normal load
- [ ] CPU usage: < 50% under normal load

## Next Phase

After completion, proceed to Phase 3: AI-Powered Features for implementing advanced relevance filtering using embeddings and enhanced AI capabilities.