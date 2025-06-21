package repositories

import (
	"context"
	"errors"
	"qt1-middleware/models"
	"time"
)

// Common repository errors
var (
	ErrNotFound = errors.New("record not found")
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*models.User, error)
	GetActiveUsers(ctx context.Context, limit, offset int) ([]*models.User, error)
	UpdateLastLogin(ctx context.Context, userID int) error
	SetActive(ctx context.Context, userID int, active bool) error
	ChangePassword(ctx context.Context, userID int, passwordHash string) error
	
	// Statistics
	Count(ctx context.Context) (int, error)
	CountByRole(ctx context.Context, role string) (int, error)
	CountActive(ctx context.Context) (int, error)
}

// SessionRepository defines the interface for session data access
type SessionRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id int) (*models.Session, error)
	GetByToken(ctx context.Context, token string) (*models.Session, error)
	Update(ctx context.Context, session *models.Session) error
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Session, error)
	GetActiveByUserID(ctx context.Context, userID int) ([]*models.Session, error)
	DeleteExpired(ctx context.Context) (int, error)
	DeleteByUserID(ctx context.Context, userID int) (int, error)
	RefreshSession(ctx context.Context, sessionID int, extendBy time.Duration) error
	Deactivate(ctx context.Context, sessionID int) error
	
	// Advanced analytics and cleanup
	GetSessionAnalytics(ctx context.Context, startTime, endTime time.Time) (*SessionAnalytics, error)
	CleanupInactiveSessions(ctx context.Context, inactiveDuration time.Duration) (int, error)
	
	// Statistics
	Count(ctx context.Context) (int, error)
	CountActive(ctx context.Context) (int, error)
	CountByUserID(ctx context.Context, userID int) (int, error)
}

// RequestRepository defines the interface for request log data access
type RequestRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, request *models.Request) error
	GetByID(ctx context.Context, id int) (*models.Request, error)
	Update(ctx context.Context, request *models.Request) error
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	List(ctx context.Context, filters *models.RequestFilters) ([]*models.Request, error)
	GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Request, error)
	GetBySessionID(ctx context.Context, sessionID int, limit, offset int) ([]*models.Request, error)
	GetByIPAddress(ctx context.Context, ipAddress string, limit, offset int) ([]*models.Request, error)
	GetBlocked(ctx context.Context, limit, offset int) ([]*models.Request, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
	
	// Statistics
	GetStats(ctx context.Context, startTime, endTime time.Time) (*models.RequestStats, error)
	Count(ctx context.Context) (int, error)
	CountBlocked(ctx context.Context) (int, error)
	CountByStatusCode(ctx context.Context, statusCode int) (int, error)
}

// ModerationLogRepository defines the interface for moderation log data access
type ModerationLogRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, log *models.ModerationLog) error
	GetByID(ctx context.Context, id int) (*models.ModerationLog, error)
	Update(ctx context.Context, log *models.ModerationLog) error
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	List(ctx context.Context, filters *models.ModerationLogFilters) ([]*models.ModerationLog, error)
	GetByRequestID(ctx context.Context, requestID int, limit, offset int) ([]*models.ModerationLog, error)
	GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.ModerationLog, error)
	GetBySeverity(ctx context.Context, severity string, limit, offset int) ([]*models.ModerationLog, error)
	GetPendingReview(ctx context.Context, limit, offset int) ([]*models.ModerationLog, error)
	MarkReviewed(ctx context.Context, logID, reviewerID int, status string) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
	
	// Statistics
	GetStats(ctx context.Context, startTime, endTime time.Time) (*models.ModerationStats, error)
	Count(ctx context.Context) (int, error)
	CountBySeverity(ctx context.Context, severity string) (int, error)
	CountByStatus(ctx context.Context, status string) (int, error)
}

// SystemConfigRepository defines the interface for system configuration data access
type SystemConfigRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, config *models.SystemConfig) error
	GetByID(ctx context.Context, id int) (*models.SystemConfig, error)
	GetByKey(ctx context.Context, key string) (*models.SystemConfig, error)
	Update(ctx context.Context, config *models.SystemConfig) error
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	List(ctx context.Context, filters *models.SystemConfigFilters) ([]*models.SystemConfig, error)
	GetByCategory(ctx context.Context, category string) ([]*models.SystemConfig, error)
	GetSecrets(ctx context.Context) ([]*models.SystemConfig, error)
	GetNonSecrets(ctx context.Context) ([]*models.SystemConfig, error)
	BulkUpdate(ctx context.Context, configs []*models.SystemConfig, updatedBy int) error
	
	// Utility methods
	GetValue(ctx context.Context, key string) (string, error)
	SetValue(ctx context.Context, key, value string, updatedBy int) error
	GetCategories(ctx context.Context) ([]string, error)
}

// MetricsRepository defines the interface for metrics data access
type MetricsRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, metric *models.Metric) error
	CreateBatch(ctx context.Context, metrics []*models.Metric) error
	GetByID(ctx context.Context, id int) (*models.Metric, error)
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	List(ctx context.Context, filters *models.MetricFilters) ([]*models.Metric, error)
	GetByName(ctx context.Context, name string, limit, offset int) ([]*models.Metric, error)
	GetByType(ctx context.Context, metricType string, limit, offset int) ([]*models.Metric, error)
	GetBySource(ctx context.Context, source string, limit, offset int) ([]*models.Metric, error)
	GetTimeSeries(ctx context.Context, name string, startTime, endTime time.Time, interval string) (*models.MetricTimeSeriesResponse, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
	
	// Aggregations
	GetAggregated(ctx context.Context, filters *models.MetricFilters) ([]*models.MetricAggregation, error)
	GetSummary(ctx context.Context, startTime, endTime time.Time) (*models.MetricSummary, error)
	
	// Statistics
	Count(ctx context.Context) (int, error)
	CountByType(ctx context.Context, metricType string) (int, error)
	CountBySource(ctx context.Context, source string) (int, error)
}

// KillSwitchRepository defines the interface for kill switch data access
type KillSwitchRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, entry *models.KillSwitchEntry) error
	GetByID(ctx context.Context, id int) (*models.KillSwitchEntry, error)
	Update(ctx context.Context, entry *models.KillSwitchEntry) error
	Delete(ctx context.Context, id int) error
	
	// Advanced operations
	List(ctx context.Context, filters *models.KillSwitchFilters) ([]*models.KillSwitchEntry, error)
	GetByType(ctx context.Context, entryType string) ([]*models.KillSwitchEntry, error)
	GetActive(ctx context.Context) ([]*models.KillSwitchEntry, error)
	GetExpired(ctx context.Context) ([]*models.KillSwitchEntry, error)
	CheckBlocked(ctx context.Context, entryType, value string) (*models.KillSwitchEntry, error)
	RecordHit(ctx context.Context, entryID int) error
	DeactivateExpired(ctx context.Context) (int, error)
	
	// Statistics
	GetStats(ctx context.Context, startTime, endTime time.Time) (*models.KillSwitchStats, error)
	Count(ctx context.Context) (int, error)
	CountActive(ctx context.Context) (int, error)
	CountByType(ctx context.Context, entryType string) (int, error)
}

// APIKeyRepository defines the interface for API key data access
type APIKeyRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, apiKey *models.APIKey) error
	GetByID(ctx context.Context, id string) (*models.APIKey, error)
	Update(ctx context.Context, apiKey *models.APIKey) error
	Delete(ctx context.Context, id string) error
	
	// Advanced operations
	GetByUserID(ctx context.Context, userID int, activeOnly bool) ([]*models.APIKey, error)
	List(ctx context.Context, filters *models.APIKeyFilters) ([]*models.APIKey, error)
	UpdateLastUsed(ctx context.Context, keyID string, lastUsed time.Time) error
	DeleteExpired(ctx context.Context, cutoff time.Time) (int, error)
	
	// Usage tracking
	GetUsage(ctx context.Context, keyID string, startDate time.Time) ([]*models.APIKeyUsage, error)
	RecordUsage(ctx context.Context, keyID, ipAddress, userAgent string) error
	
	// Statistics
	GetStats(ctx context.Context) (*models.APIKeyStats, error)
	Count(ctx context.Context) (int, error)
	CountActive(ctx context.Context) (int, error)
	CountByUserID(ctx context.Context, userID int) (int, error)
}

// RepositoryManager defines the interface for managing all repositories
type RepositoryManager interface {
	Users() UserRepository
	Sessions() SessionRepository
	Requests() RequestRepository
	ModerationLogs() ModerationLogRepository
	SystemConfigs() SystemConfigRepository
	Metrics() MetricsRepository
	KillSwitch() KillSwitchRepository
	APIKeys() APIKeyRepository
	Close() error
	
	// Transaction support
	BeginTransaction(ctx context.Context) (Transaction, error)
	WithTransaction(ctx context.Context, fn func(tx Transaction) error) error
}