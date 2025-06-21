package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"qt1-middleware/models"
	"qt1-middleware/repositories"
	"strings"
	"sync"
	"time"
)

// SessionManager manages user sessions with advanced security features
type SessionManager struct {
	sessionRepo     repositories.SessionRepository
	config          *SessionConfig
	rateLimiter     *RateLimiter
	anomalyDetector *AnomalyDetector
	mutex           sync.RWMutex
	deviceTracker   *DeviceTracker
}

// SessionConfig holds configuration for session management
type SessionConfig struct {
	SessionDuration        time.Duration `yaml:"session_duration" json:"session_duration"`
	MaxConcurrentSessions  int           `yaml:"max_concurrent_sessions" json:"max_concurrent_sessions"`
	SessionTokenLength     int           `yaml:"session_token_length" json:"session_token_length"`
	EnableAnomalyDetection bool          `yaml:"enable_anomaly_detection" json:"enable_anomaly_detection"`
	EnableDeviceTracking   bool          `yaml:"enable_device_tracking" json:"enable_device_tracking"`
	InactiveCleanupMinutes int           `yaml:"inactive_cleanup_minutes" json:"inactive_cleanup_minutes"`
	ForceLogoutOnIPChange  bool          `yaml:"force_logout_on_ip_change" json:"force_logout_on_ip_change"`
	RequireDeviceVerify    bool          `yaml:"require_device_verify" json:"require_device_verify"`
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		SessionDuration:        24 * time.Hour,
		MaxConcurrentSessions:  5,
		SessionTokenLength:     64,
		EnableAnomalyDetection: true,
		EnableDeviceTracking:   true,
		InactiveCleanupMinutes: 30,
		ForceLogoutOnIPChange:  false,
		RequireDeviceVerify:    false,
	}
}

// NewSessionManager creates a new session manager
func NewSessionManager(sessionRepo repositories.SessionRepository, config *SessionConfig) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}

	sm := &SessionManager{
		sessionRepo:     sessionRepo,
		config:          config,
		rateLimiter:     NewRateLimiter(),
		anomalyDetector: NewAnomalyDetector(),
		deviceTracker:   NewDeviceTracker(),
	}

	// Start background cleanup process
	go sm.startCleanupWorker()

	return sm
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(ctx context.Context, userID int, r *http.Request) (*models.Session, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check rate limiting
	clientIP := sm.getClientIP(r)
	if !sm.rateLimiter.Allow(clientIP) {
		return nil, &SessionError{
			Code:    "RATE_LIMITED",
			Message: "Too many session creation attempts",
		}
	}

	// Check concurrent session limits
	activeSessions, err := sm.sessionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active sessions: %w", err)
	}

	if len(activeSessions) >= sm.config.MaxConcurrentSessions {
		// Remove oldest session
		if err := sm.removeOldestSession(ctx, activeSessions); err != nil {
			log.Printf("Warning: failed to remove oldest session: %v", err)
		}
	}

	// Generate secure session token
	token, err := sm.generateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	// Extract device information
	userAgent := r.Header.Get("User-Agent")
	deviceFingerprint := sm.generateDeviceFingerprint(r)

	// Create session
	session := &models.Session{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(sm.config.SessionDuration),
		IPAddress: clientIP,
		UserAgent: userAgent,
		Active:    true,
	}

	if err := sm.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Track device if enabled
	if sm.config.EnableDeviceTracking {
		sm.deviceTracker.TrackDevice(userID, deviceFingerprint, clientIP, userAgent)
	}

	// Log session creation
	sm.logSecurityEvent("SESSION_CREATED", userID, clientIP, userAgent, fmt.Sprintf("Session ID: %d", session.ID))

	return session, nil
}

// ValidateSession validates and refreshes a session
func (sm *SessionManager) ValidateSession(ctx context.Context, token string, r *http.Request) (*models.Session, error) {
	session, err := sm.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, &SessionError{
			Code:    "SESSION_NOT_FOUND",
			Message: "Session not found",
		}
	}

	// Check if session is valid
	if !session.IsValid() {
		sm.logSecurityEvent("SESSION_INVALID", session.UserID, session.IPAddress, session.UserAgent, 
			fmt.Sprintf("Expired or inactive session: %d", session.ID))
		return nil, &SessionError{
			Code:    "SESSION_EXPIRED",
			Message: "Session has expired or is inactive",
		}
	}

	// Security checks
	if err := sm.performSecurityChecks(ctx, session, r); err != nil {
		return nil, err
	}

	// Refresh session activity
	if err := sm.sessionRepo.RefreshSession(ctx, session.ID, 0); err != nil {
		log.Printf("Warning: failed to refresh session: %v", err)
	}

	return session, nil
}

// InvalidateSession invalidates a specific session
func (sm *SessionManager) InvalidateSession(ctx context.Context, sessionID int) error {
	session, err := sm.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return &SessionError{
			Code:    "SESSION_NOT_FOUND",
			Message: "Session not found",
		}
	}

	if err := sm.sessionRepo.Deactivate(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to deactivate session: %w", err)
	}

	sm.logSecurityEvent("SESSION_INVALIDATED", session.UserID, session.IPAddress, session.UserAgent,
		fmt.Sprintf("Session ID: %d", sessionID))

	return nil
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (sm *SessionManager) InvalidateAllUserSessions(ctx context.Context, userID int) error {
	activeSessions, err := sm.sessionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get active sessions: %w", err)
	}

	for _, session := range activeSessions {
		if err := sm.sessionRepo.Deactivate(ctx, session.ID); err != nil {
			log.Printf("Warning: failed to deactivate session %d: %v", session.ID, err)
		}
	}

	sm.logSecurityEvent("ALL_SESSIONS_INVALIDATED", userID, "", "", 
		fmt.Sprintf("Invalidated %d sessions", len(activeSessions)))

	return nil
}

// GetUserSessions returns sessions for a user with optional filtering
func (sm *SessionManager) GetUserSessions(ctx context.Context, userID int, activeOnly bool) ([]*models.Session, error) {
	if activeOnly {
		return sm.sessionRepo.GetActiveByUserID(ctx, userID)
	}
	return sm.sessionRepo.GetByUserID(ctx, userID, 50, 0) // Limit to 50 recent sessions
}

// ForceLogout forces logout from a specific session
func (sm *SessionManager) ForceLogout(ctx context.Context, sessionID int, reason string) error {
	session, err := sm.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return &SessionError{
			Code:    "SESSION_NOT_FOUND",
			Message: "Session not found",
		}
	}

	if err := sm.sessionRepo.Deactivate(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to deactivate session: %w", err)
	}

	sm.logSecurityEvent("FORCED_LOGOUT", session.UserID, session.IPAddress, session.UserAgent,
		fmt.Sprintf("Session ID: %d, Reason: %s", sessionID, reason))

	return nil
}

// GetSessionAnalytics returns analytics for sessions
func (sm *SessionManager) GetSessionAnalytics(ctx context.Context, startTime, endTime time.Time) (*repositories.SessionAnalytics, error) {
	return sm.sessionRepo.GetSessionAnalytics(ctx, startTime, endTime)
}

// DetectSuspiciousActivity checks for suspicious session activity
func (sm *SessionManager) DetectSuspiciousActivity(ctx context.Context, userID int) ([]*SuspiciousActivity, error) {
	if !sm.config.EnableAnomalyDetection {
		return nil, nil
	}

	sessions, err := sm.sessionRepo.GetByUserID(ctx, userID, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return sm.anomalyDetector.AnalyzeSessions(sessions), nil
}

// GetDeviceInfo returns device information for a user
func (sm *SessionManager) GetDeviceInfo(userID int) ([]*DeviceInfo, error) {
	if !sm.config.EnableDeviceTracking {
		return nil, fmt.Errorf("device tracking not enabled")
	}

	return sm.deviceTracker.GetUserDevices(userID), nil
}

// ExtendSession extends the expiration time of a session
func (sm *SessionManager) ExtendSession(ctx context.Context, sessionID int, duration time.Duration) error {
	if err := sm.sessionRepo.RefreshSession(ctx, sessionID, duration); err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	session, err := sm.sessionRepo.GetByID(ctx, sessionID)
	if err == nil && session != nil {
		sm.logSecurityEvent("SESSION_EXTENDED", session.UserID, session.IPAddress, session.UserAgent,
			fmt.Sprintf("Extended by %v", duration))
	}

	return nil
}

// Private helper methods

func (sm *SessionManager) generateSessionToken() (string, error) {
	bytes := make([]byte, sm.config.SessionTokenLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (sm *SessionManager) generateDeviceFingerprint(r *http.Request) string {
	// Simple device fingerprinting based on headers
	components := []string{
		r.Header.Get("User-Agent"),
		r.Header.Get("Accept-Language"),
		r.Header.Get("Accept-Encoding"),
		r.Header.Get("Accept"),
	}

	data := strings.Join(components, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (sm *SessionManager) getClientIP(r *http.Request) string {
	// Check for forwarded IP addresses
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for real IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (sm *SessionManager) performSecurityChecks(ctx context.Context, session *models.Session, r *http.Request) error {
	currentIP := sm.getClientIP(r)
	
	// Check for IP address change
	if sm.config.ForceLogoutOnIPChange && session.IPAddress != currentIP {
		sm.logSecurityEvent("IP_CHANGE_DETECTED", session.UserID, currentIP, r.Header.Get("User-Agent"),
			fmt.Sprintf("Session IP: %s, Current IP: %s", session.IPAddress, currentIP))
		
		// Invalidate session due to IP change
		sm.sessionRepo.Deactivate(ctx, session.ID)
		return &SessionError{
			Code:    "IP_CHANGE_DETECTED",
			Message: "Session invalidated due to IP address change",
		}
	}

	// Anomaly detection
	if sm.config.EnableAnomalyDetection {
		if suspicious := sm.anomalyDetector.CheckRequest(session.UserID, currentIP, r.Header.Get("User-Agent")); suspicious {
			sm.logSecurityEvent("SUSPICIOUS_ACTIVITY", session.UserID, currentIP, r.Header.Get("User-Agent"),
				"Anomaly detected in session activity")
		}
	}

	return nil
}

func (sm *SessionManager) removeOldestSession(ctx context.Context, sessions []*models.Session) error {
	if len(sessions) == 0 {
		return nil
	}

	// Find oldest session
	oldest := sessions[0]
	for _, session := range sessions[1:] {
		if session.LastUsedAt.Before(oldest.LastUsedAt) {
			oldest = session
		}
	}

	return sm.sessionRepo.Deactivate(ctx, oldest.ID)
}

func (sm *SessionManager) startCleanupWorker() {
	ticker := time.NewTicker(time.Duration(sm.config.InactiveCleanupMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			
			// Clean up expired sessions
			if deleted, err := sm.sessionRepo.DeleteExpired(ctx); err != nil {
				log.Printf("Error cleaning up expired sessions: %v", err)
			} else if deleted > 0 {
				log.Printf("Cleaned up %d expired sessions", deleted)
			}

			// Clean up inactive sessions
			inactiveDuration := time.Duration(sm.config.InactiveCleanupMinutes) * time.Minute * 2
			if deleted, err := sm.sessionRepo.CleanupInactiveSessions(ctx, inactiveDuration); err != nil {
				log.Printf("Error cleaning up inactive sessions: %v", err)
			} else if deleted > 0 {
				log.Printf("Cleaned up %d inactive sessions", deleted)
			}
		}
	}
}

func (sm *SessionManager) logSecurityEvent(event string, userID int, ip, userAgent, details string) {
	timestamp := time.Now().Format(time.RFC3339)
	log.Printf("SECURITY_EVENT: %s | User: %d | IP: %s | UserAgent: %s | Details: %s | Time: %s",
		event, userID, ip, userAgent, details, timestamp)
}

// SessionError represents a session-related error
type SessionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *SessionError) Error() string {
	return e.Message
}

// SuspiciousActivity represents detected suspicious activity
type SuspiciousActivity struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detected_at"`
	SessionID   int       `json:"session_id"`
	UserID      int       `json:"user_id"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
}

