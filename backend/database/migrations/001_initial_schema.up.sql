-- Initial database schema for QT-1 Middleware
-- This migration creates the core tables for user management, request logging, and system configuration

-- Users table for authentication and user management
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    api_key VARCHAR(255) UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME
);

-- Sessions table for managing user sessions
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER NOT NULL,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Requests table for logging all API requests
CREATE TABLE requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    session_id VARCHAR(255),
    request_id VARCHAR(255) UNIQUE NOT NULL,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    query_params TEXT,
    headers TEXT, -- JSON
    request_body TEXT,
    response_status INTEGER,
    response_headers TEXT, -- JSON
    response_body TEXT,
    duration_ms INTEGER,
    ip_address VARCHAR(45),
    user_agent TEXT,
    provider VARCHAR(50),
    model VARCHAR(100),
    tokens_used INTEGER DEFAULT 0,
    cost_cents INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE SET NULL
);

-- Moderation logs for tracking content moderation events
CREATE TABLE moderation_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id INTEGER,
    rule_name VARCHAR(100),
    rule_type VARCHAR(50) NOT NULL, -- 'regex', 'llm', 'pii', 'custom'
    action VARCHAR(20) NOT NULL, -- 'allow', 'block', 'flag', 'modify'
    reason TEXT,
    severity VARCHAR(20) DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    confidence REAL DEFAULT 0.0, -- 0.0 to 1.0
    input_text TEXT,
    output_text TEXT,
    metadata TEXT, -- JSON for additional context
    processing_time_ms INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (request_id) REFERENCES requests(id) ON DELETE CASCADE
);

-- System configuration table for dynamic settings
CREATE TABLE system_config (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    category VARCHAR(50) DEFAULT 'general',
    data_type VARCHAR(20) DEFAULT 'string', -- 'string', 'integer', 'boolean', 'json'
    is_sensitive BOOLEAN DEFAULT false,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER,
    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Metrics table for storing performance and usage metrics
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metric_name VARCHAR(100) NOT NULL,
    metric_type VARCHAR(20) NOT NULL, -- 'counter', 'gauge', 'histogram'
    value REAL NOT NULL,
    unit VARCHAR(20),
    tags TEXT, -- JSON for additional dimensions
    metadata TEXT, -- JSON for extra context
    retention_class VARCHAR(20) DEFAULT 'standard' -- for retention policies
);

-- Kill switch entries for blocking users, sessions, or other entities
CREATE TABLE kill_switch_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(20) NOT NULL, -- 'user', 'session', 'ip', 'api_key'
    value VARCHAR(255) NOT NULL,
    reason TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    expires_at DATETIME,
    created_by INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for better query performance
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_api_key ON users(api_key);
CREATE INDEX idx_users_active ON users(active);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(session_token);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE INDEX idx_requests_user_id ON requests(user_id);
CREATE INDEX idx_requests_session_id ON requests(session_id);
CREATE INDEX idx_requests_created_at ON requests(created_at);
CREATE INDEX idx_requests_status ON requests(response_status);
CREATE INDEX idx_requests_provider ON requests(provider);
CREATE INDEX idx_requests_path ON requests(path);

CREATE INDEX idx_moderation_request_id ON moderation_logs(request_id);
CREATE INDEX idx_moderation_action ON moderation_logs(action);
CREATE INDEX idx_moderation_severity ON moderation_logs(severity);
CREATE INDEX idx_moderation_created_at ON moderation_logs(created_at);

CREATE INDEX idx_config_category ON system_config(category);
CREATE INDEX idx_config_updated_at ON system_config(updated_at);

CREATE INDEX idx_metrics_name ON metrics(metric_name);
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX idx_metrics_type ON metrics(metric_type);
CREATE INDEX idx_metrics_retention ON metrics(retention_class);

CREATE INDEX idx_killswitch_type ON kill_switch_entries(type);
CREATE INDEX idx_killswitch_value ON kill_switch_entries(value);
CREATE INDEX idx_killswitch_active ON kill_switch_entries(active);
CREATE INDEX idx_killswitch_expires ON kill_switch_entries(expires_at);

-- Default admin user will be created by the application on startup

-- Insert default system configuration
INSERT INTO system_config (key, value, description, category) VALUES
('app_name', 'QT-1 Middleware', 'Application name', 'general'),
('app_version', '1.0.0', 'Application version', 'general'),
('maintenance_mode', 'false', 'Enable maintenance mode', 'system'),
('max_request_size', '10485760', 'Maximum request size in bytes (10MB)', 'limits'),
('rate_limit_requests', '1000', 'Requests per minute per user', 'limits'),
('session_timeout', '86400', 'Session timeout in seconds (24 hours)', 'auth'),
('log_retention_days', '90', 'Log retention in days', 'logging'),
('metrics_retention_days', '7', 'Metrics retention in days', 'metrics');