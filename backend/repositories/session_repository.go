package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"qt1-middleware/models"
	"strings"
	"time"
)

// sessionRepository implements SessionRepository interface
type sessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// Create inserts a new session into the database
func (r *sessionRepository) Create(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	session.CreatedAt = now
	session.LastUsedAt = now
	
	result, err := r.db.ExecContext(ctx, query,
		session.UserID, session.Token, session.ExpiresAt, session.CreatedAt,
		session.LastUsedAt, session.IPAddress, session.UserAgent, session.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get created session ID: %w", err)
	}
	
	session.ID = int(id)
	return nil
}

// GetByID retrieves a session by ID
func (r *sessionRepository) GetByID(ctx context.Context, id int) (*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions WHERE id = ?
	`
	
	session := &models.Session{}
	var ipAddress, userAgent sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}
	
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	
	return session, nil
}

// GetByToken retrieves a session by token
func (r *sessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions WHERE token = ?
	`
	
	session := &models.Session{}
	var ipAddress, userAgent sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
		&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}
	
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	
	return session, nil
}

// Update updates an existing session
func (r *sessionRepository) Update(ctx context.Context, session *models.Session) error {
	query := `
		UPDATE sessions 
		SET user_id = ?, token = ?, expires_at = ?, last_used_at = ?, 
		    ip_address = ?, user_agent = ?, active = ?
		WHERE id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query,
		session.UserID, session.Token, session.ExpiresAt, session.LastUsedAt,
		session.IPAddress, session.UserAgent, session.Active, session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

// Delete deletes a session by ID
func (r *sessionRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM sessions WHERE id = ?`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

// GetByUserID retrieves sessions for a specific user with pagination
func (r *sessionRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions 
		WHERE user_id = ?
		ORDER BY last_used_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}
	defer rows.Close()
	
	return r.scanSessions(rows)
}

// GetActiveByUserID retrieves all active sessions for a specific user
func (r *sessionRepository) GetActiveByUserID(ctx context.Context, userID int) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions 
		WHERE user_id = ? AND active = true AND expires_at > ?
		ORDER BY last_used_at DESC
	`
	
	now := time.Now()
	rows, err := r.db.QueryContext(ctx, query, userID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions by user ID: %w", err)
	}
	defer rows.Close()
	
	return r.scanSessions(rows)
}

// DeleteExpired removes expired sessions and returns the count of deleted sessions
func (r *sessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	query := `DELETE FROM sessions WHERE expires_at <= ?`
	
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// DeleteByUserID deletes all sessions for a specific user
func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID int) (int, error) {
	query := `DELETE FROM sessions WHERE user_id = ?`
	
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete sessions by user ID: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// RefreshSession updates the last used time and optionally extends the expiration
func (r *sessionRepository) RefreshSession(ctx context.Context, sessionID int, extendBy time.Duration) error {
	var query string
	var args []interface{}
	
	now := time.Now()
	
	if extendBy > 0 {
		query = `
			UPDATE sessions 
			SET last_used_at = ?, expires_at = ? 
			WHERE id = ? AND active = true
		`
		newExpiry := now.Add(extendBy)
		args = []interface{}{now, newExpiry, sessionID}
	} else {
		query = `
			UPDATE sessions 
			SET last_used_at = ? 
			WHERE id = ? AND active = true
		`
		args = []interface{}{now, sessionID}
	}
	
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found or not active")
	}
	
	return nil
}

// Deactivate marks a session as inactive
func (r *sessionRepository) Deactivate(ctx context.Context, sessionID int) error {
	query := `UPDATE sessions SET active = false WHERE id = ?`
	
	result, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to deactivate session: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}
	
	return nil
}

// Count returns the total number of sessions
func (r *sessionRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM sessions`
	
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	
	return count, nil
}

// CountActive returns the number of active sessions
func (r *sessionRepository) CountActive(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE active = true AND expires_at > ?`
	
	now := time.Now()
	var count int
	err := r.db.QueryRowContext(ctx, query, now).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}
	
	return count, nil
}

// CountByUserID returns the number of sessions for a specific user
func (r *sessionRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE user_id = ?`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions by user ID: %w", err)
	}
	
	return count, nil
}

// Advanced query methods

// GetSessionsWithUserDetails returns sessions with user information using JOIN
func (r *sessionRepository) GetSessionsWithUserDetails(ctx context.Context, limit, offset int) ([]*SessionWithUser, error) {
	query := `
		SELECT 
			s.id, s.user_id, s.token, s.expires_at, s.created_at, s.last_used_at, 
			s.ip_address, s.user_agent, s.active,
			u.username, u.email, u.role
		FROM sessions s
		INNER JOIN users u ON s.user_id = u.id
		WHERE s.active = true AND s.expires_at > ?
		ORDER BY s.last_used_at DESC
		LIMIT ? OFFSET ?
	`
	
	now := time.Now()
	rows, err := r.db.QueryContext(ctx, query, now, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions with user details: %w", err)
	}
	defer rows.Close()
	
	var results []*SessionWithUser
	for rows.Next() {
		session := &models.Session{}
		user := &models.User{}
		var ipAddress, userAgent sql.NullString
		
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
			&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
			&user.Username, &user.Email, &user.Role,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session with user: %w", err)
		}
		
		if ipAddress.Valid {
			session.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		
		user.ID = session.UserID
		
		results = append(results, &SessionWithUser{
			Session: session,
			User:    user,
		})
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session with user rows: %w", err)
	}
	
	return results, nil
}

// GetSessionAnalytics returns session analytics including active sessions, peak times, etc.
func (r *sessionRepository) GetSessionAnalytics(ctx context.Context, startTime, endTime time.Time) (*SessionAnalytics, error) {
	analytics := &SessionAnalytics{}
	
	// Total sessions created in the time period
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sessions 
		WHERE created_at BETWEEN ? AND ?
	`, startTime, endTime).Scan(&analytics.TotalSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total sessions: %w", err)
	}
	
	// Currently active sessions
	now := time.Now()
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sessions 
		WHERE active = true AND expires_at > ?
	`, now).Scan(&analytics.ActiveSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	
	// Expired sessions
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sessions 
		WHERE expires_at <= ?
	`, now).Scan(&analytics.ExpiredSessions)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired sessions: %w", err)
	}
	
	// Unique users with sessions
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM sessions 
		WHERE created_at BETWEEN ? AND ?
	`, startTime, endTime).Scan(&analytics.UniqueUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique users: %w", err)
	}
	
	// Average session duration (for completed sessions)
	var avgDuration sql.NullFloat64
	err = r.db.QueryRowContext(ctx, `
		SELECT AVG(
			CAST((julianday(last_used_at) - julianday(created_at)) * 24 * 60 AS REAL)
		) as avg_minutes
		FROM sessions 
		WHERE created_at BETWEEN ? AND ? AND last_used_at > created_at
	`, startTime, endTime).Scan(&avgDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get average session duration: %w", err)
	}
	
	if avgDuration.Valid {
		analytics.AvgSessionDurationMinutes = avgDuration.Float64
	}
	
	return analytics, nil
}

// GetSessionsByIPAddress finds sessions by IP address with pattern matching
func (r *sessionRepository) GetSessionsByIPAddress(ctx context.Context, ipPattern string, limit, offset int) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions 
		WHERE ip_address LIKE ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	pattern := "%" + ipPattern + "%"
	rows, err := r.db.QueryContext(ctx, query, pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by IP address: %w", err)
	}
	defer rows.Close()
	
	return r.scanSessions(rows)
}

// GetSessionsByUserAgent finds sessions by user agent pattern
func (r *sessionRepository) GetSessionsByUserAgent(ctx context.Context, userAgentPattern string, limit, offset int) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, last_used_at, ip_address, user_agent, active
		FROM sessions 
		WHERE user_agent LIKE ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	pattern := "%" + strings.ToLower(userAgentPattern) + "%"
	rows, err := r.db.QueryContext(ctx, query, pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user agent: %w", err)
	}
	defer rows.Close()
	
	return r.scanSessions(rows)
}

// CleanupInactiveSessions removes sessions that haven't been used for a specified duration
func (r *sessionRepository) CleanupInactiveSessions(ctx context.Context, inactiveDuration time.Duration) (int, error) {
	cutoff := time.Now().Add(-inactiveDuration)
	query := `DELETE FROM sessions WHERE last_used_at < ? OR (active = false AND created_at < ?)`
	
	result, err := r.db.ExecContext(ctx, query, cutoff, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup inactive sessions: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// scanSessions is a helper function to scan multiple sessions from rows
func (r *sessionRepository) scanSessions(rows *sql.Rows) ([]*models.Session, error) {
	var sessions []*models.Session
	
	for rows.Next() {
		session := &models.Session{}
		var ipAddress, userAgent sql.NullString
		
		err := rows.Scan(
			&session.ID, &session.UserID, &session.Token, &session.ExpiresAt,
			&session.CreatedAt, &session.LastUsedAt, &ipAddress, &userAgent, &session.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		
		if ipAddress.Valid {
			session.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		
		sessions = append(sessions, session)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}
	
	return sessions, nil
}

// SessionWithUser represents a session with associated user information
type SessionWithUser struct {
	Session *models.Session `json:"session"`
	User    *models.User    `json:"user"`
}

// SessionAnalytics represents session analytics data
type SessionAnalytics struct {
	TotalSessions              int     `json:"total_sessions"`
	ActiveSessions             int     `json:"active_sessions"`
	ExpiredSessions            int     `json:"expired_sessions"`
	UniqueUsers                int     `json:"unique_users"`
	AvgSessionDurationMinutes  float64 `json:"avg_session_duration_minutes"`
}