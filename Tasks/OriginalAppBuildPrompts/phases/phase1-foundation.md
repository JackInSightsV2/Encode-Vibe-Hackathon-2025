# Phase 1: Foundation & Core Setup

## Overview

Establish the foundational architecture for QT-1 Go backend with SQLite database, configuration system, and basic OpenAI integration.

## Prerequisites

- Go 1.21+ installed
- OpenAI API key available in .env file
- SQLite3 support

## Project Structure

```
backend/
├── main.go                 # Entry point
├── go.mod                  # Go module file
├── go.sum                  # Go dependencies
├── config/
│   ├── config.go          # Configuration loading
│   └── types.go           # Configuration types
├── storage/
│   ├── sqlite.go          # SQLite database operations
│   ├── migrations.go      # Database schema setup
│   └── models.go          # Data models
├── utils/
│   └── logger.go          # Logging utilities
└── .env                   # Environment variables
```

## Tasks Checklist

### 1. Project Initialization
- [ ] Create `backend/` directory
- [ ] Initialize Go module: `go mod init qt1-backend`
- [ ] Create basic directory structure
- [ ] Set up .gitignore for Go project

### 2. Dependencies Setup
- [ ] Add required Go dependencies:
  - `github.com/gorilla/mux` - HTTP router
  - `github.com/mattn/go-sqlite3` - SQLite driver
  - `github.com/joho/godotenv` - Environment variables
  - `gopkg.in/yaml.v3` - YAML configuration
  - HTTP client libraries for API calls

### 3. Configuration System
- [ ] Create `config/types.go` with configuration structs:
  - ServerConfig (port, host, timeouts)
  - DatabaseConfig (file path, connection settings)
  - OpenAIConfig (API key, base URL, model settings)
  - ModerationConfig (enabled filters, thresholds)
  - LoggingConfig (level, file paths, formats)

- [ ] Create `config/config.go` with functions:
  - `LoadConfig()` - Load from YAML and .env
  - `ValidateConfig()` - Validate configuration values
  - `GetConfig()` - Global config access

- [ ] Create `config.yaml` template with default settings

### 4. Database Setup
- [ ] Create `storage/models.go` with data structures:
  ```go
  type LogEntry struct {
      ID          int64     `json:"id"`
      Timestamp   time.Time `json:"timestamp"`
      UserID      string    `json:"user_id"`
      SessionID   string    `json:"session_id"`
      RequestData string    `json:"request_data"`
      Response    string    `json:"response"`
      Action      string    `json:"action"` // allowed, blocked, filtered
      Reason      string    `json:"reason"`
      ModelUsed   string    `json:"model_used"`
  }

  type BlockedEntity struct {
      ID        int64     `json:"id"`
      Type      string    `json:"type"` // user, session, ip
      Value     string    `json:"value"`
      Reason    string    `json:"reason"`
      CreatedAt time.Time `json:"created_at"`
      ExpiresAt *time.Time `json:"expires_at"`
      Active    bool      `json:"active"`
  }

  type ConfigEntry struct {
      Key       string    `json:"key"`
      Value     string    `json:"value"`
      UpdatedAt time.Time `json:"updated_at"`
  }
  ```

- [ ] Create `storage/migrations.go`:
  - `CreateTables()` - Create all required tables
  - `MigrateDatabase()` - Handle schema updates
  - SQL statements for table creation

- [ ] Create `storage/sqlite.go`:
  - `NewDatabase()` - Initialize database connection
  - `Close()` - Close database connection
  - Basic CRUD operations for each model

### 5. Logging System
- [ ] Create `utils/logger.go`:
  - Structured logging with levels (DEBUG, INFO, WARN, ERROR)
  - File rotation support
  - JSON formatting for structured logs
  - Context-aware logging

### 6. Basic HTTP Server
- [ ] Create `main.go`:
  - Load configuration
  - Initialize database
  - Set up HTTP server with Gorilla Mux
  - Basic health check endpoint (`/health`)
  - Graceful shutdown handling

- [ ] Basic route structure:
  ```go
  r := mux.NewRouter()
  r.HandleFunc("/health", healthHandler).Methods("GET")
  r.HandleFunc("/chat", chatHandler).Methods("POST")
  
  api := r.PathPrefix("/api").Subrouter()
  api.HandleFunc("/status", statusHandler).Methods("GET")
  ```

### 7. OpenAI Integration Foundation
- [ ] Create `providers/` directory
- [ ] Create `providers/openai.go`:
  - OpenAI client initialization
  - Basic chat completion function
  - Error handling for API calls
  - Rate limiting considerations

- [ ] Create `providers/types.go`:
  - Common interfaces for model providers
  - Request/response structures
  - Error types

### 8. Basic Request Flow
- [ ] Implement basic `/chat` endpoint:
  - Accept JSON: `{"user_id": "string", "session_id": "string", "message": "string"}`
  - Log incoming request
  - Forward to OpenAI (basic implementation)
  - Log response
  - Return response to client

### 9. Environment Configuration
- [ ] Update `.env` file with required variables:
  ```
  OPENAI_KEY=your_key_here
  SERVER_PORT=8080
  SERVER_HOST=localhost
  DATABASE_PATH=./qt1.db
  LOG_LEVEL=INFO
  LOG_FILE=./logs/qt1.log
  ```

### 10. Testing & Validation
- [ ] Create basic unit tests for:
  - Configuration loading
  - Database operations
  - OpenAI client functionality

- [ ] Manual testing:
  - Server starts successfully
  - Health endpoint responds
  - Database tables are created
  - Basic chat request works with OpenAI

## Success Criteria

- [ ] Go server starts on configured port
- [ ] SQLite database is created with proper schema
- [ ] Configuration loads from YAML and .env files
- [ ] Health check endpoint returns 200 OK
- [ ] Basic chat endpoint forwards requests to OpenAI
- [ ] All requests and responses are logged to database
- [ ] No external dependencies except essential Go packages

## Next Phase

After completion, proceed to Phase 2: Core Middleware Pipeline for implementing moderation, filtering, and kill switch functionality.