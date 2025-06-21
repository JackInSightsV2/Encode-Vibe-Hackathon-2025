package repositories

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RetentionPolicy defines data retention rules
type RetentionPolicy struct {
	TableName    string
	RetentionKey string        // Column name for date comparison
	MaxAge       time.Duration // How long to keep data
	BatchSize    int           // Number of records to delete per batch
}

// RetentionService handles data cleanup according to retention policies
type RetentionService struct {
	manager  RepositoryManager
	policies []RetentionPolicy
}

// NewRetentionService creates a new retention service
func NewRetentionService(manager RepositoryManager) *RetentionService {
	return &RetentionService{
		manager: manager,
		policies: []RetentionPolicy{
			{
				TableName:    "metrics",
				RetentionKey: "timestamp",
				MaxAge:       7 * 24 * time.Hour, // 7 days
				BatchSize:    1000,
			},
			{
				TableName:    "requests",
				RetentionKey: "created_at",
				MaxAge:       90 * 24 * time.Hour, // 90 days
				BatchSize:    500,
			},
			{
				TableName:    "moderation_logs",
				RetentionKey: "created_at",
				MaxAge:       30 * 24 * time.Hour, // 30 days
				BatchSize:    500,
			},
			{
				TableName:    "sessions",
				RetentionKey: "expires_at",
				MaxAge:       0, // Clean up immediately after expiration
				BatchSize:    100,
			},
		},
	}
}

// RunCleanup executes all retention policies
func (rs *RetentionService) RunCleanup(ctx context.Context) error {
	log.Printf("Starting data retention cleanup...")
	
	var totalCleaned int
	for _, policy := range rs.policies {
		cleaned, err := rs.executePolicy(ctx, policy)
		if err != nil {
			log.Printf("Error executing retention policy for %s: %v", policy.TableName, err)
			continue
		}
		
		if cleaned > 0 {
			log.Printf("Cleaned %d records from %s", cleaned, policy.TableName)
		}
		totalCleaned += cleaned
	}
	
	log.Printf("Data retention cleanup completed. Total records cleaned: %d", totalCleaned)
	return nil
}

// executePolicy executes a single retention policy
func (rs *RetentionService) executePolicy(ctx context.Context, policy RetentionPolicy) (int, error) {
	cutoffTime := time.Now().Add(-policy.MaxAge)
	
	// Special handling for sessions - clean up expired sessions
	if policy.TableName == "sessions" {
		cutoffTime = time.Now() // Clean up sessions that have already expired
	}
	
	switch policy.TableName {
	case "metrics":
		return rs.manager.Metrics().DeleteOlderThan(ctx, cutoffTime)
	case "requests":
		return rs.manager.Requests().DeleteOlderThan(ctx, cutoffTime)
	case "moderation_logs":
		return rs.manager.ModerationLogs().DeleteOlderThan(ctx, cutoffTime)
	case "sessions":
		return rs.manager.Sessions().DeleteExpired(ctx)
	default:
		return 0, fmt.Errorf("unknown table for retention policy: %s", policy.TableName)
	}
}

// GetRetentionPolicies returns the current retention policies
func (rs *RetentionService) GetRetentionPolicies() []RetentionPolicy {
	return rs.policies
}

// UpdateRetentionPolicy updates a specific retention policy
func (rs *RetentionService) UpdateRetentionPolicy(tableName string, maxAge time.Duration) error {
	for i, policy := range rs.policies {
		if policy.TableName == tableName {
			rs.policies[i].MaxAge = maxAge
			return nil
		}
	}
	return fmt.Errorf("retention policy not found for table: %s", tableName)
}

// AddRetentionPolicy adds a new retention policy
func (rs *RetentionService) AddRetentionPolicy(policy RetentionPolicy) {
	rs.policies = append(rs.policies, policy)
}

// RemoveRetentionPolicy removes a retention policy
func (rs *RetentionService) RemoveRetentionPolicy(tableName string) error {
	for i, policy := range rs.policies {
		if policy.TableName == tableName {
			rs.policies = append(rs.policies[:i], rs.policies[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("retention policy not found for table: %s", tableName)
}

// GetRetentionStats returns statistics about data retention
func (rs *RetentionService) GetRetentionStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get total counts for each table
	metricsCount, err := rs.manager.Metrics().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count metrics: %w", err)
	}
	stats["total_metrics"] = metricsCount
	
	requestsCount, err := rs.manager.Requests().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count requests: %w", err)
	}
	stats["total_requests"] = requestsCount
	
	moderationCount, err := rs.manager.ModerationLogs().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count moderation logs: %w", err)
	}
	stats["total_moderation_logs"] = moderationCount
	
	sessionsCount, err := rs.manager.Sessions().Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}
	stats["total_sessions"] = sessionsCount
	
	activeSessions, err := rs.manager.Sessions().CountActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active sessions: %w", err)
	}
	stats["active_sessions"] = activeSessions
	
	// Add policy information
	policies := make([]map[string]interface{}, len(rs.policies))
	for i, policy := range rs.policies {
		policies[i] = map[string]interface{}{
			"table":       policy.TableName,
			"max_age":     policy.MaxAge.String(),
			"batch_size":  policy.BatchSize,
		}
	}
	stats["retention_policies"] = policies
	
	return stats, nil
}

// DryRunCleanup simulates cleanup without actually deleting data
func (rs *RetentionService) DryRunCleanup(ctx context.Context) (map[string]int, error) {
	results := make(map[string]int)
	
	for _, policy := range rs.policies {
		// This would require additional methods in repositories to count records that would be deleted
		// For now, we'll return estimated counts based on policy
		
		// This would require additional methods in repositories to count records that would be deleted
		// For now, we'll return estimated counts based on policy
		switch policy.TableName {
		case "metrics":
			// Estimate based on retention period
			results["metrics_estimated"] = 0 // Would need COUNT query with date filter
		case "requests":
			results["requests_estimated"] = 0
		case "moderation_logs":
			results["moderation_logs_estimated"] = 0
		case "sessions":
			results["sessions_estimated"] = 0
		}
	}
	
	return results, nil
}

// ScheduleCleanup can be used to schedule periodic cleanup
// This would typically be called from a cron job or scheduler
func (rs *RetentionService) ScheduleCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("Retention cleanup scheduler stopped")
			return
		case <-ticker.C:
			if err := rs.RunCleanup(ctx); err != nil {
				log.Printf("Scheduled cleanup failed: %v", err)
			}
		}
	}
}

// ValidateRetentionConfig validates retention configuration from system config
func (rs *RetentionService) ValidateRetentionConfig(ctx context.Context) error {
	// Get retention settings from system config
	configRepo := rs.manager.SystemConfigs()
	
	// Check if metrics retention is configured
	metricsRetention, err := configRepo.GetByKey(ctx, "metrics_retention_hours")
	if err != nil {
		return fmt.Errorf("metrics retention not configured: %w", err)
	}
	
	if metricsRetention == nil {
		return fmt.Errorf("metrics retention configuration not found")
	}
	
	// Additional validation could be added here
	log.Printf("Retention configuration validated: metrics_retention=%s", metricsRetention.Value)
	
	return nil
}

// UpdatePoliciesFromConfig updates retention policies from system configuration
func (rs *RetentionService) UpdatePoliciesFromConfig(ctx context.Context) error {
	configRepo := rs.manager.SystemConfigs()
	
	// Update metrics retention
	if config, err := configRepo.GetByKey(ctx, "metrics_retention_hours"); err == nil && config != nil {
		if hours, err := time.ParseDuration(config.Value + "h"); err == nil {
			rs.UpdateRetentionPolicy("metrics", hours)
		}
	}
	
	// Update request logs retention
	if config, err := configRepo.GetByKey(ctx, "log_retention_days"); err == nil && config != nil {
		if days, err := time.ParseDuration(config.Value + "h"); err == nil {
			rs.UpdateRetentionPolicy("requests", days*24)
		}
	}
	
	log.Printf("Retention policies updated from configuration")
	return nil
}