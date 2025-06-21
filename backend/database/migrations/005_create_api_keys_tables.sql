-- Migration: Create API keys tables
-- Description: Creates tables for API key management including keys, permissions, and usage tracking

-- Create api_keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(64) PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    key_hash TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    usage_count BIGINT NOT NULL DEFAULT 0,
    
    CONSTRAINT api_keys_name_user_unique UNIQUE(user_id, name)
);

-- Create api_key_permissions table for storing key permissions
CREATE TABLE IF NOT EXISTS api_key_permissions (
    id SERIAL PRIMARY KEY,
    api_key_id VARCHAR(64) NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    
    CONSTRAINT api_key_permissions_unique UNIQUE(api_key_id, permission)
);

-- Create api_key_usage table for tracking daily usage statistics
CREATE TABLE IF NOT EXISTS api_key_usage (
    id SERIAL PRIMARY KEY,
    api_key_id VARCHAR(64) NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    request_count BIGINT NOT NULL DEFAULT 0,
    last_request TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT api_key_usage_unique UNIQUE(api_key_id, date)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_last_used ON api_keys(last_used);

CREATE INDEX IF NOT EXISTS idx_api_key_permissions_api_key_id ON api_key_permissions(api_key_id);

CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_date ON api_key_usage(date);

-- Add comments for documentation
COMMENT ON TABLE api_keys IS 'Stores API keys for programmatic access to the system';
COMMENT ON COLUMN api_keys.id IS 'Unique identifier for the API key';
COMMENT ON COLUMN api_keys.user_id IS 'ID of the user who owns this API key';
COMMENT ON COLUMN api_keys.name IS 'Human-readable name for the API key';
COMMENT ON COLUMN api_keys.description IS 'Optional description of the API key purpose';
COMMENT ON COLUMN api_keys.key_hash IS 'Hashed version of the API key for secure storage';
COMMENT ON COLUMN api_keys.active IS 'Whether the API key is currently active';
COMMENT ON COLUMN api_keys.last_used IS 'Timestamp of when the API key was last used';
COMMENT ON COLUMN api_keys.expires_at IS 'Optional expiration timestamp for the API key';
COMMENT ON COLUMN api_keys.usage_count IS 'Total number of times this API key has been used';

COMMENT ON TABLE api_key_permissions IS 'Stores permissions associated with API keys';
COMMENT ON COLUMN api_key_permissions.api_key_id IS 'Reference to the API key';
COMMENT ON COLUMN api_key_permissions.permission IS 'Permission string (e.g., config:read, logs:write)';

COMMENT ON TABLE api_key_usage IS 'Tracks daily usage statistics for API keys';
COMMENT ON COLUMN api_key_usage.api_key_id IS 'Reference to the API key';
COMMENT ON COLUMN api_key_usage.date IS 'Date for which usage is tracked';
COMMENT ON COLUMN api_key_usage.request_count IS 'Number of requests made on this date';
COMMENT ON COLUMN api_key_usage.last_request IS 'Timestamp of the last request on this date';