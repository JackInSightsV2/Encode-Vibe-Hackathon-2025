package repositories

import (
	"context"
	"database/sql"
	"qt1-middleware/database"
)

// repositoryManager implements RepositoryManager interface
type repositoryManager struct {
	db                  *database.Database
	transactionManager  TransactionManager
	userRepo            UserRepository
	sessionRepo         SessionRepository
	requestRepo         RequestRepository
	moderationLogRepo   ModerationLogRepository
	systemConfigRepo    SystemConfigRepository
	metricsRepo         MetricsRepository
	killSwitchRepo      KillSwitchRepository
	apiKeyRepo          APIKeyRepository
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(db *database.Database) RepositoryManager {
	return &repositoryManager{
		db:                 db,
		transactionManager: NewTransactionManager(db.DB),
		userRepo:           NewUserRepository(db.DB),
		sessionRepo:        NewSessionRepository(db.DB),
		requestRepo:        NewRequestRepository(db.DB),
		moderationLogRepo:  NewModerationLogRepository(db.DB),
		systemConfigRepo:   NewSystemConfigRepository(db.DB),
		metricsRepo:        NewMetricsRepository(db.DB),
		killSwitchRepo:     NewKillSwitchRepository(db.DB),
		apiKeyRepo:         NewAPIKeyRepository(db.DB),
	}
}

// Users returns the user repository
func (rm *repositoryManager) Users() UserRepository {
	return rm.userRepo
}

// Sessions returns the session repository
func (rm *repositoryManager) Sessions() SessionRepository {
	return rm.sessionRepo
}

// Requests returns the request repository
func (rm *repositoryManager) Requests() RequestRepository {
	return rm.requestRepo
}

// ModerationLogs returns the moderation log repository
func (rm *repositoryManager) ModerationLogs() ModerationLogRepository {
	return rm.moderationLogRepo
}

// SystemConfigs returns the system config repository
func (rm *repositoryManager) SystemConfigs() SystemConfigRepository {
	return rm.systemConfigRepo
}

// Metrics returns the metrics repository
func (rm *repositoryManager) Metrics() MetricsRepository {
	return rm.metricsRepo
}

// KillSwitch returns the kill switch repository
func (rm *repositoryManager) KillSwitch() KillSwitchRepository {
	return rm.killSwitchRepo
}

// APIKeys returns the API key repository
func (rm *repositoryManager) APIKeys() APIKeyRepository {
	return rm.apiKeyRepo
}

// Close closes all repository connections
func (rm *repositoryManager) Close() error {
	if rm.db != nil {
		return rm.db.Close()
	}
	return nil
}

// BeginTransaction starts a new database transaction
func (rm *repositoryManager) BeginTransaction(ctx context.Context) (Transaction, error) {
	return rm.transactionManager.BeginTransaction(ctx)
}

// WithTransaction executes a function within a transaction
func (rm *repositoryManager) WithTransaction(ctx context.Context, fn func(tx Transaction) error) error {
	return WithTransaction(ctx, rm.transactionManager, fn)
}

// Placeholder implementations for other repositories
// These would be implemented similarly to the UserRepository

// NewSessionRepository is now implemented in session_repository.go

func NewRequestRepository(db *sql.DB) RequestRepository {
	return &requestRepository{db: db}
}

func NewModerationLogRepository(db *sql.DB) ModerationLogRepository {
	return &moderationLogRepository{db: db}
}

// Implementations are now in their respective files

func NewKillSwitchRepository(db *sql.DB) KillSwitchRepository {
	return &killSwitchRepository{db: db}
}

// sessionRepository struct is now implemented in session_repository.go
type requestRepository struct{ db *sql.DB }
type moderationLogRepository struct{ db *sql.DB }
type killSwitchRepository struct{ db *sql.DB }

// TODO: Implement all repository methods for each repository type
// For now, we'll implement placeholder methods that return not implemented errors
// This allows the code to compile while we incrementally implement each repository