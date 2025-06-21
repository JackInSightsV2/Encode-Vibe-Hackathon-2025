-- Configuration and Metrics Performance Optimizations
-- This migration adds additional indexes and constraints for improved query performance

-- Fix the system_config table structure to match our model
ALTER TABLE system_config ADD COLUMN id INTEGER;
ALTER TABLE system_config ADD COLUMN value_type VARCHAR(20) DEFAULT 'string';
ALTER TABLE system_config ADD COLUMN is_secret BOOLEAN DEFAULT false;
ALTER TABLE system_config ADD COLUMN read_only BOOLEAN DEFAULT false;
ALTER TABLE system_config ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE system_config ADD COLUMN version INTEGER DEFAULT 1;

-- Rename columns to match our model
ALTER TABLE system_config RENAME COLUMN data_type TO value_type_old;
ALTER TABLE system_config RENAME COLUMN is_sensitive TO is_secret_old;

-- Update the value_type column
UPDATE system_config SET value_type = COALESCE(value_type_old, 'string');
UPDATE system_config SET is_secret = COALESCE(is_secret_old, false);

-- Fix the metrics table structure to match our model
ALTER TABLE metrics ADD COLUMN name VARCHAR(100);
ALTER TABLE metrics ADD COLUMN source VARCHAR(50) DEFAULT 'system';
ALTER TABLE metrics ADD COLUMN collected_at DATETIME DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE metrics ADD COLUMN labels TEXT;

-- Update the name column from metric_name
UPDATE metrics SET name = metric_name WHERE name IS NULL;

-- Additional indexes for system_config performance
CREATE INDEX IF NOT EXISTS idx_config_key_category ON system_config(key, category);
CREATE INDEX IF NOT EXISTS idx_config_is_secret ON system_config(is_secret);
CREATE INDEX IF NOT EXISTS idx_config_read_only ON system_config(read_only);
CREATE INDEX IF NOT EXISTS idx_config_version ON system_config(version);

-- Additional indexes for metrics performance (time series queries)
CREATE INDEX IF NOT EXISTS idx_metrics_name_timestamp ON metrics(name, timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_source_timestamp ON metrics(source, timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_type_timestamp ON metrics(metric_type, timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_collected_at ON metrics(collected_at);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_metrics_name_type_source ON metrics(name, metric_type, source);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp_name ON metrics(timestamp DESC, name);

-- Partitioning hints for large datasets (commented out for SQLite compatibility)
-- For PostgreSQL, you might want to partition metrics by timestamp
-- CREATE INDEX IF NOT EXISTS idx_metrics_timestamp_month ON metrics(date_trunc('month', timestamp));

-- Add constraints for data integrity
-- Ensure metric values are not null or infinite
-- ALTER TABLE metrics ADD CONSTRAINT chk_metrics_value_valid CHECK (value IS NOT NULL AND value != 'inf' AND value != '-inf');

-- Add default configurations for the new repository features
INSERT OR REPLACE INTO system_config (key, value, value_type, category, description, is_secret, read_only) VALUES
('metrics_batch_size', '1000', 'int', 'metrics', 'Maximum number of metrics to insert in a single batch', false, false),
('metrics_retention_hours', '168', 'int', 'metrics', 'Metrics retention in hours (7 days)', false, false),
('config_cache_ttl', '300', 'int', 'config', 'Configuration cache TTL in seconds', false, false),
('config_audit_enabled', 'true', 'bool', 'config', 'Enable configuration change auditing', false, false),
('metrics_aggregation_interval', '300', 'int', 'metrics', 'Metrics aggregation interval in seconds', false, false);

-- Update existing config values to have proper types
UPDATE system_config SET value_type = 'bool' WHERE key IN ('maintenance_mode', 'config_audit_enabled');
UPDATE system_config SET value_type = 'int' WHERE key IN ('max_request_size', 'rate_limit_requests', 'session_timeout', 'log_retention_days', 'metrics_retention_days', 'metrics_batch_size', 'metrics_retention_hours', 'config_cache_ttl', 'metrics_aggregation_interval');
UPDATE system_config SET value_type = 'string' WHERE key IN ('app_name', 'app_version');

-- Set version for all existing configs
UPDATE system_config SET version = 1 WHERE version IS NULL;