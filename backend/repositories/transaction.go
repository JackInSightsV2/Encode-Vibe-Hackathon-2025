package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Transaction represents a database transaction with repository access
type Transaction interface {
	// Transaction control
	Commit() error
	Rollback() error
	Context() context.Context
	
	// Repository access within transaction
	Users() UserRepository
	Sessions() SessionRepository
	Requests() RequestRepository
	ModerationLogs() ModerationLogRepository
	SystemConfigs() SystemConfigRepository
	Metrics() MetricsRepository
	KillSwitch() KillSwitchRepository
}

// TransactionManager handles transaction creation and management
type TransactionManager interface {
	BeginTransaction(ctx context.Context) (Transaction, error)
	BeginTransactionWithOptions(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
}

// databaseTransaction implements Transaction interface
type databaseTransaction struct {
	tx               *sql.Tx
	ctx              context.Context
	db               *sql.DB
	startTime        time.Time
	committed        bool
	rolledBack       bool
	
	// Repository factory for this transaction
	repoFactory      *repositoryFactory
}

// transactionManager implements TransactionManager interface
type transactionManager struct {
	db *sql.DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *sql.DB) TransactionManager {
	return &transactionManager{db: db}
}

// BeginTransaction starts a new database transaction
func (tm *transactionManager) BeginTransaction(ctx context.Context) (Transaction, error) {
	return tm.BeginTransactionWithOptions(ctx, nil)
}

// BeginTransactionWithOptions starts a new database transaction with specific options
func (tm *transactionManager) BeginTransactionWithOptions(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := tm.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	dbTx := &databaseTransaction{
		tx:          tx,
		ctx:         ctx,
		db:          tm.db,
		startTime:   time.Now(),
		repoFactory: NewRepositoryFactory(tx),
	}
	
	return dbTx, nil
}

// Commit commits the transaction
func (dt *databaseTransaction) Commit() error {
	if dt.committed {
		return fmt.Errorf("transaction already committed")
	}
	if dt.rolledBack {
		return fmt.Errorf("transaction already rolled back")
	}
	
	err := dt.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	dt.committed = true
	
	// Log transaction duration for performance monitoring
	duration := time.Since(dt.startTime)
	logTransactionMetrics("commit", duration, nil)
	
	return nil
}

// Rollback rolls back the transaction
func (dt *databaseTransaction) Rollback() error {
	if dt.committed {
		return fmt.Errorf("transaction already committed")
	}
	if dt.rolledBack {
		return nil // Already rolled back, this is okay
	}
	
	err := dt.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	
	dt.rolledBack = true
	
	// Log transaction duration for performance monitoring
	duration := time.Since(dt.startTime)
	logTransactionMetrics("rollback", duration, nil)
	
	return nil
}

// Context returns the transaction context
func (dt *databaseTransaction) Context() context.Context {
	return dt.ctx
}

// Repository access methods
func (dt *databaseTransaction) Users() UserRepository {
	return dt.repoFactory.Users()
}

func (dt *databaseTransaction) Sessions() SessionRepository {
	return dt.repoFactory.Sessions()
}

func (dt *databaseTransaction) Requests() RequestRepository {
	return dt.repoFactory.Requests()
}

func (dt *databaseTransaction) ModerationLogs() ModerationLogRepository {
	return dt.repoFactory.ModerationLogs()
}

func (dt *databaseTransaction) SystemConfigs() SystemConfigRepository {
	return dt.repoFactory.SystemConfigs()
}

func (dt *databaseTransaction) Metrics() MetricsRepository {
	return dt.repoFactory.Metrics()
}

func (dt *databaseTransaction) KillSwitch() KillSwitchRepository {
	return dt.repoFactory.KillSwitch()
}

// TransactionExecutor provides a convenient way to execute operations within a transaction
type TransactionExecutor struct {
	tm TransactionManager
}

// NewTransactionExecutor creates a new transaction executor
func NewTransactionExecutor(tm TransactionManager) *TransactionExecutor {
	return &TransactionExecutor{tm: tm}
}

// Execute runs a function within a transaction, automatically handling commit/rollback
func (te *TransactionExecutor) Execute(ctx context.Context, fn func(tx Transaction) error) error {
	tx, err := te.tm.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // Re-panic after rollback
		}
	}()
	
	err = fn(tx)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("operation failed: %w, rollback failed: %v", err, rollbackErr)
		}
		return err
	}
	
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}
	
	return nil
}

// ExecuteWithOptions runs a function within a transaction with specific options
func (te *TransactionExecutor) ExecuteWithOptions(ctx context.Context, opts *sql.TxOptions, fn func(tx Transaction) error) error {
	tx, err := te.tm.BeginTransactionWithOptions(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	
	err = fn(tx)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("operation failed: %w, rollback failed: %v", err, rollbackErr)
		}
		return err
	}
	
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}
	
	return nil
}

// Transaction helper functions

// WithTransaction is a convenience function for executing operations within a transaction
func WithTransaction(ctx context.Context, tm TransactionManager, fn func(tx Transaction) error) error {
	executor := NewTransactionExecutor(tm)
	return executor.Execute(ctx, fn)
}

// WithReadOnlyTransaction executes operations within a read-only transaction
func WithReadOnlyTransaction(ctx context.Context, tm TransactionManager, fn func(tx Transaction) error) error {
	executor := NewTransactionExecutor(tm)
	opts := &sql.TxOptions{
		ReadOnly: true,
	}
	return executor.ExecuteWithOptions(ctx, opts, fn)
}

// logTransactionMetrics logs transaction performance metrics
func logTransactionMetrics(action string, duration time.Duration, err error) {
	// In a production system, this would integrate with your metrics collection
	// For now, we'll just log the basic information
	status := "success"
	if err != nil {
		status = "error"
	}
	
	// This could send metrics to your metrics repository or monitoring system
	fmt.Printf("Transaction %s: duration=%v, status=%s\n", action, duration, status)
}

// Transaction isolation levels for different use cases
var (
	// ReadCommittedTx for most operations
	ReadCommittedTx = &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}
	
	// SerializableTx for operations requiring strict consistency
	SerializableTx = &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}
	
	// ReadOnlyTx for read-only operations
	ReadOnlyTx = &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}
)