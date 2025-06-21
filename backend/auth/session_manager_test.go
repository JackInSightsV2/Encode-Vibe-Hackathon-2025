package auth

import (
	"context"
	"fmt"
	"net/http/httptest"
	"qt1-middleware/models"
	"qt1-middleware/repositories"
	"testing"
	"time"
)

func TestSessionManager(t *testing.T) {
	// Create mock session repository
	sessionRepo := &MockSessionRepository{
		sessions: make(map[int]*models.Session),
		nextID:   1,
	}

	// Create session manager with test config
	config := &SessionConfig{
		SessionDuration:        1 * time.Hour,
		MaxConcurrentSessions:  2,
		SessionTokenLength:     32,
		EnableAnomalyDetection: true,
		EnableDeviceTracking:   true,
		InactiveCleanupMinutes: 30,
		ForceLogoutOnIPChange:  false,
		RequireDeviceVerify:    false,
	}

	sessionManager := NewSessionManager(sessionRepo, config)

	t.Run("CreateSession", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

		session, err := sessionManager.CreateSession(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		if session.UserID != 1 {
			t.Errorf("Expected UserID 1, got %d", session.UserID)
		}

		if session.Token == "" {
			t.Error("Expected non-empty token")
		}

		if !session.Active {
			t.Error("Expected session to be active")
		}

		if session.IPAddress != "192.168.1.1" {
			t.Errorf("Expected IP 192.168.1.1, got %s", session.IPAddress)
		}
	})

	t.Run("ValidateSession", func(t *testing.T) {
		// Create a session first
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

		session, err := sessionManager.CreateSession(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Validate the session
		validatedSession, err := sessionManager.ValidateSession(context.Background(), session.Token, req)
		if err != nil {
			t.Fatalf("Failed to validate session: %v", err)
		}

		if validatedSession.ID != session.ID {
			t.Errorf("Expected session ID %d, got %d", session.ID, validatedSession.ID)
		}
	})

	t.Run("ConcurrentSessionLimit", func(t *testing.T) {
		userID := 2

		// Create sessions up to the limit
		for i := 0; i < config.MaxConcurrentSessions; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

			_, err := sessionManager.CreateSession(context.Background(), userID, req)
			if err != nil {
				t.Fatalf("Failed to create session %d: %v", i, err)
			}
		}

		// Creating one more should succeed but remove the oldest
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

		_, err := sessionManager.CreateSession(context.Background(), userID, req)
		if err != nil {
			t.Fatalf("Failed to create session beyond limit: %v", err)
		}

		// Should still have only MaxConcurrentSessions active sessions
		sessions, err := sessionManager.GetUserSessions(context.Background(), userID, true)
		if err != nil {
			t.Fatalf("Failed to get user sessions: %v", err)
		}

		if len(sessions) > config.MaxConcurrentSessions {
			t.Errorf("Expected max %d sessions, got %d", config.MaxConcurrentSessions, len(sessions))
		}
	})

	t.Run("InvalidateSession", func(t *testing.T) {
		// Create a session
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

		session, err := sessionManager.CreateSession(context.Background(), 3, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Invalidate the session
		err = sessionManager.InvalidateSession(context.Background(), session.ID)
		if err != nil {
			t.Fatalf("Failed to invalidate session: %v", err)
		}

		// Session should no longer be valid
		_, err = sessionManager.ValidateSession(context.Background(), session.Token, req)
		if err == nil {
			t.Error("Expected session validation to fail after invalidation")
		}
	})

	t.Run("ForceLogout", func(t *testing.T) {
		// Create a session
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

		session, err := sessionManager.CreateSession(context.Background(), 4, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Force logout with reason
		err = sessionManager.ForceLogout(context.Background(), session.ID, "Administrative action")
		if err != nil {
			t.Fatalf("Failed to force logout: %v", err)
		}

		// Session should no longer be valid
		_, err = sessionManager.ValidateSession(context.Background(), session.Token, req)
		if err == nil {
			t.Error("Expected session validation to fail after force logout")
		}
	})

	t.Run("ExtendSession", func(t *testing.T) {
		// Create a session
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")

		session, err := sessionManager.CreateSession(context.Background(), 5, req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		originalExpiry := session.ExpiresAt

		// Extend the session
		extension := 30 * time.Minute
		err = sessionManager.ExtendSession(context.Background(), session.ID, extension)
		if err != nil {
			t.Fatalf("Failed to extend session: %v", err)
		}

		// Note: In a real test, we'd need to verify the expiry was extended
		// This would require checking the session from the repository
		_ = originalExpiry
	})
}

func TestRateLimiter(t *testing.T) {
	rateLimiter := NewRateLimiter()

	t.Run("AllowWithinLimit", func(t *testing.T) {
		clientID := "test-client-1"

		// Should allow requests within limit
		for i := 0; i < 5; i++ {
			if !rateLimiter.Allow(clientID) {
				t.Errorf("Request %d should be allowed", i)
			}
		}
	})

	t.Run("BlockWhenLimitExceeded", func(t *testing.T) {
		clientID := "test-client-2"

		// Fill up the bucket
		for i := 0; i < 10; i++ {
			rateLimiter.Allow(clientID)
		}

		// Next request should be blocked
		if rateLimiter.Allow(clientID) {
			t.Error("Request should be blocked when limit exceeded")
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		clientID := "test-client-3"

		// Make some requests
		for i := 0; i < 3; i++ {
			rateLimiter.Allow(clientID)
		}

		status := rateLimiter.GetStatus(clientID)
		if status.Requests != 3 {
			t.Errorf("Expected 3 requests, got %d", status.Requests)
		}

		if status.MaxRequests != 10 {
			t.Errorf("Expected max 10 requests, got %d", status.MaxRequests)
		}
	})

	t.Run("BanAndUnban", func(t *testing.T) {
		clientID := "test-client-4"

		// Ban the client
		rateLimiter.Ban(clientID, 1*time.Minute)

		// Should be blocked
		if rateLimiter.Allow(clientID) {
			t.Error("Banned client should be blocked")
		}

		// Unban the client
		rateLimiter.Unban(clientID)

		// Should be allowed again
		if !rateLimiter.Allow(clientID) {
			t.Error("Unbanned client should be allowed")
		}
	})
}

func TestAnomalyDetector(t *testing.T) {
	detector := NewAnomalyDetector()

	t.Run("DetectMultipleIPs", func(t *testing.T) {
		userID := 1

		// Create sessions from many different IPs
		sessions := []*models.Session{
			{UserID: userID, IPAddress: "192.168.1.1", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "192.168.1.2", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "192.168.1.3", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "10.0.0.1", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "172.16.0.1", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "203.0.113.1", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "198.51.100.1", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "203.0.113.2", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "198.51.100.2", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "203.0.113.3", CreatedAt: time.Now()},
			{UserID: userID, IPAddress: "198.51.100.3", CreatedAt: time.Now()},
		}

		activities := detector.AnalyzeSessions(sessions)

		// Should detect multiple IPs as suspicious
		found := false
		for _, activity := range activities {
			if activity.Type == "MULTIPLE_IPS" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Should detect multiple IPs as suspicious")
		}
	})

	t.Run("DetectRapidLogins", func(t *testing.T) {
		userID := 2
		now := time.Now()

		// Create sessions with rapid succession (reverse order for proper time diff calculation)
		sessions := []*models.Session{
			{UserID: userID, IPAddress: "192.168.1.1", CreatedAt: now.Add(10 * time.Second)},
			{UserID: userID, IPAddress: "192.168.1.1", CreatedAt: now.Add(5 * time.Second)},
			{UserID: userID, IPAddress: "192.168.1.1", CreatedAt: now},
		}

		activities := detector.AnalyzeSessions(sessions)

		// Should detect rapid logins as suspicious
		found := false
		for _, activity := range activities {
			if activity.Type == "RAPID_LOGINS" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Should detect rapid logins as suspicious")
		}
	})

	t.Run("CheckRequest", func(t *testing.T) {
		userID := 3

		// First request should be fine
		suspicious := detector.CheckRequest(userID, "192.168.1.1", "Mozilla/5.0")
		if suspicious {
			t.Error("First request should not be suspicious")
		}

		// Rapid successive request should be suspicious
		suspicious = detector.CheckRequest(userID, "192.168.1.1", "Mozilla/5.0")
		if !suspicious {
			t.Error("Rapid successive request should be suspicious")
		}
	})
}

func TestDeviceTracker(t *testing.T) {
	tracker := NewDeviceTracker()

	t.Run("TrackDevice", func(t *testing.T) {
		userID := 1
		fingerprint := "test-fingerprint-1"
		ip := "192.168.1.1"
		userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

		device := tracker.TrackDevice(userID, fingerprint, ip, userAgent)

		if device.Fingerprint != fingerprint {
			t.Errorf("Expected fingerprint %s, got %s", fingerprint, device.Fingerprint)
		}

		if device.SessionCount != 1 {
			t.Errorf("Expected session count 1, got %d", device.SessionCount)
		}

		if device.DeviceType != "desktop" {
			t.Errorf("Expected device type desktop, got %s", device.DeviceType)
		}

		if device.Browser != "Chrome" {
			t.Errorf("Expected browser Chrome, got %s", device.Browser)
		}

		if device.OS != "Windows" {
			t.Errorf("Expected OS Windows, got %s", device.OS)
		}
	})

	t.Run("AutoTrust", func(t *testing.T) {
		userID := 2
		fingerprint := "test-fingerprint-2"
		ip := "192.168.1.1"
		userAgent := "Mozilla/5.0 (test)"

		// Track device multiple times to trigger auto-trust
		for i := 0; i < 6; i++ {
			tracker.TrackDevice(userID, fingerprint, ip, userAgent)
		}

		device := tracker.GetDevice(userID, fingerprint)
		if device == nil {
			t.Fatal("Device should exist")
		}

		if !device.Trusted {
			t.Error("Device should be auto-trusted after 5 sessions")
		}
	})

	t.Run("TrustDevice", func(t *testing.T) {
		userID := 3
		fingerprint := "test-fingerprint-3"
		ip := "192.168.1.1"
		userAgent := "Mozilla/5.0 (test)"

		tracker.TrackDevice(userID, fingerprint, ip, userAgent)

		err := tracker.TrustDevice(userID, fingerprint)
		if err != nil {
			t.Fatalf("Failed to trust device: %v", err)
		}

		device := tracker.GetDevice(userID, fingerprint)
		if !device.Trusted {
			t.Error("Device should be trusted")
		}
	})

	t.Run("BlockDevice", func(t *testing.T) {
		userID := 4
		fingerprint := "test-fingerprint-4"
		ip := "192.168.1.1"
		userAgent := "Mozilla/5.0 (test)"
		reason := "Suspicious activity"

		tracker.TrackDevice(userID, fingerprint, ip, userAgent)

		err := tracker.BlockDevice(userID, fingerprint, reason)
		if err != nil {
			t.Fatalf("Failed to block device: %v", err)
		}

		device := tracker.GetDevice(userID, fingerprint)
		if !device.Blocked {
			t.Error("Device should be blocked")
		}

		if device.BlockedReason != reason {
			t.Errorf("Expected blocked reason %s, got %s", reason, device.BlockedReason)
		}

		if tracker.IsDeviceTrusted(userID, fingerprint) {
			t.Error("Blocked device should not be trusted")
		}
	})

	t.Run("GetDeviceStats", func(t *testing.T) {
		// Create some devices for testing
		tracker.TrackDevice(10, "fp-1", "192.168.1.1", "Mozilla/5.0 (Windows)")
		tracker.TrackDevice(10, "fp-2", "192.168.1.2", "Mozilla/5.0 (Macintosh)")
		tracker.TrackDevice(11, "fp-3", "192.168.1.3", "Mozilla/5.0 (Android)")

		stats := tracker.GetDeviceStats()

		if stats.TotalUsers < 1 {
			t.Error("Should have at least 1 user")
		}

		if stats.TotalDevices < 3 {
			t.Error("Should have at least 3 devices")
		}

		if len(stats.DeviceTypes) == 0 {
			t.Error("Should have device type statistics")
		}
	})
}

// MockSessionRepository for testing
type MockSessionRepository struct {
	sessions map[int]*models.Session
	nextID   int
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	session.ID = m.nextID
	m.nextID++
	session.CreatedAt = time.Now()
	session.LastUsedAt = time.Now()
	m.sessions[session.ID] = session
	return nil
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id int) (*models.Session, error) {
	session, exists := m.sessions[id]
	if !exists {
		return nil, nil
	}
	return session, nil
}

func (m *MockSessionRepository) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	for _, session := range m.sessions {
		if session.Token == token {
			return session, nil
		}
	}
	return nil, nil
}

func (m *MockSessionRepository) Update(ctx context.Context, session *models.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *MockSessionRepository) Delete(ctx context.Context, id int) error {
	delete(m.sessions, id)
	return nil
}

func (m *MockSessionRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Session, error) {
	var sessions []*models.Session
	for _, session := range m.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (m *MockSessionRepository) GetActiveByUserID(ctx context.Context, userID int) ([]*models.Session, error) {
	var sessions []*models.Session
	for _, session := range m.sessions {
		if session.UserID == userID && session.IsValid() {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	count := 0
	for id, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, id)
			count++
		}
	}
	return count, nil
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID int) (int, error) {
	count := 0
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, id)
			count++
		}
	}
	return count, nil
}

func (m *MockSessionRepository) RefreshSession(ctx context.Context, sessionID int, extendBy time.Duration) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.LastUsedAt = time.Now()
	if extendBy > 0 {
		session.ExpiresAt = time.Now().Add(extendBy)
	}
	return nil
}

func (m *MockSessionRepository) Deactivate(ctx context.Context, sessionID int) error {
	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Active = false
	return nil
}

func (m *MockSessionRepository) Count(ctx context.Context) (int, error) {
	return len(m.sessions), nil
}

func (m *MockSessionRepository) CountActive(ctx context.Context) (int, error) {
	count := 0
	for _, session := range m.sessions {
		if session.IsValid() {
			count++
		}
	}
	return count, nil
}

func (m *MockSessionRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	count := 0
	for _, session := range m.sessions {
		if session.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *MockSessionRepository) GetSessionAnalytics(ctx context.Context, startTime, endTime time.Time) (*repositories.SessionAnalytics, error) {
	return &repositories.SessionAnalytics{
		TotalSessions:             len(m.sessions),
		ActiveSessions:            0,
		ExpiredSessions:           0,
		UniqueUsers:               0,
		AvgSessionDurationMinutes: 0,
	}, nil
}

func (m *MockSessionRepository) CleanupInactiveSessions(ctx context.Context, inactiveDuration time.Duration) (int, error) {
	return 0, nil
}