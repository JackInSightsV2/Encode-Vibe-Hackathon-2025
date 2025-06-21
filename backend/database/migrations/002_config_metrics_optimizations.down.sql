-- Rollback Configuration and Metrics Optimizations

-- Remove additional indexes
DROP INDEX IF EXISTS idx_config_key_category;
DROP INDEX IF EXISTS idx_config_is_secret;
DROP INDEX IF EXISTS idx_config_read_only;
DROP INDEX IF EXISTS idx_config_version;

DROP INDEX IF EXISTS idx_metrics_name_timestamp;
DROP INDEX IF EXISTS idx_metrics_source_timestamp;
DROP INDEX IF EXISTS idx_metrics_type_timestamp;
DROP INDEX IF EXISTS idx_metrics_collected_at;
DROP INDEX IF EXISTS idx_metrics_name_type_source;
DROP INDEX IF EXISTS idx_metrics_timestamp_name;

-- Remove added configuration entries
DELETE FROM system_config WHERE key IN (
    'metrics_batch_size',
    'metrics_retention_hours', 
    'config_cache_ttl',
    'config_audit_enabled',
    'metrics_aggregation_interval'
);

-- Note: SQLite doesn't support dropping columns easily
-- In a production environment with PostgreSQL, you would:
-- ALTER TABLE system_config DROP COLUMN id;
-- ALTER TABLE system_config DROP COLUMN value_type;
-- ALTER TABLE system_config DROP COLUMN is_secret;
-- ALTER TABLE system_config DROP COLUMN read_only;
-- ALTER TABLE system_config DROP COLUMN created_at;
-- ALTER TABLE system_config DROP COLUMN version;

-- ALTER TABLE metrics DROP COLUMN name;
-- ALTER TABLE metrics DROP COLUMN source;
-- ALTER TABLE metrics DROP COLUMN collected_at;
-- ALTER TABLE metrics DROP COLUMN labels;