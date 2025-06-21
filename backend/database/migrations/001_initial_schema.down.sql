-- Rollback initial schema
-- This migration removes all tables created in the initial schema

-- Drop tables in reverse order to handle foreign key constraints
DROP TABLE IF EXISTS kill_switch_entries;
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS system_config;
DROP TABLE IF EXISTS moderation_logs;
DROP TABLE IF EXISTS requests;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;