package auth

import (
	"net"
	"qt1-middleware/models"
	"strings"
	"sync"
	"time"
)

// AnomalyDetector detects suspicious session activity
type AnomalyDetector struct {
	userProfiles map[int]*UserProfile
	mutex        sync.RWMutex
	config       *AnomalyConfig
}

// AnomalyConfig holds configuration for anomaly detection
type AnomalyConfig struct {
	MaxIPsPerUser           int           `yaml:"max_ips_per_user" json:"max_ips_per_user"`
	MaxDevicesPerUser       int           `yaml:"max_devices_per_user" json:"max_devices_per_user"`
	SuspiciousIPThreshold   int           `yaml:"suspicious_ip_threshold" json:"suspicious_ip_threshold"`
	GeoLocationCheckEnabled bool          `yaml:"geo_location_check_enabled" json:"geo_location_check_enabled"`
	TimezoneJumpThreshold   time.Duration `yaml:"timezone_jump_threshold" json:"timezone_jump_threshold"`
	RapidLoginThreshold     time.Duration `yaml:"rapid_login_threshold" json:"rapid_login_threshold"`
	CleanupInterval         time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// UserProfile tracks user behavior patterns
type UserProfile struct {
	UserID           int                    `json:"user_id"`
	KnownIPs         map[string]*IPInfo     `json:"known_ips"`
	KnownUserAgents  map[string]*DeviceInfo `json:"known_user_agents"`
	LoginPattern     *LoginPattern          `json:"login_pattern"`
	LastActivity     time.Time              `json:"last_activity"`
	SuspiciousEvents []*SuspiciousActivity  `json:"suspicious_events"`
	mutex            sync.RWMutex
}

// IPInfo tracks information about an IP address
type IPInfo struct {
	IPAddress     string    `json:"ip_address"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	LoginCount    int       `json:"login_count"`
	Country       string    `json:"country,omitempty"`
	City          string    `json:"city,omitempty"`
	ISP           string    `json:"isp,omitempty"`
	IsSuspicious  bool      `json:"is_suspicious"`
	ThreatScore   int       `json:"threat_score"`
}

// LoginPattern represents typical login behavior
type LoginPattern struct {
	TypicalHours    []int         `json:"typical_hours"`    // Hours of day (0-23)
	TypicalDays     []int         `json:"typical_days"`     // Days of week (0-6)
	AverageInterval time.Duration `json:"average_interval"` // Average time between logins
	LastLogin       time.Time     `json:"last_login"`
	LoginCount      int           `json:"login_count"`
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	config := &AnomalyConfig{
		MaxIPsPerUser:           10,
		MaxDevicesPerUser:       5,
		SuspiciousIPThreshold:   3,
		GeoLocationCheckEnabled: false, // Disabled for now, requires external service
		TimezoneJumpThreshold:   6 * time.Hour,
		RapidLoginThreshold:     30 * time.Second,
		CleanupInterval:         24 * time.Hour,
	}

	ad := &AnomalyDetector{
		userProfiles: make(map[int]*UserProfile),
		config:       config,
	}

	// Start cleanup goroutine
	go ad.startCleanup()

	return ad
}

// AnalyzeSessions analyzes user sessions for suspicious patterns
func (ad *AnomalyDetector) AnalyzeSessions(sessions []*models.Session) []*SuspiciousActivity {
	if len(sessions) == 0 {
		return nil
	}

	userID := sessions[0].UserID
	profile := ad.getOrCreateProfile(userID)
	
	var suspiciousActivities []*SuspiciousActivity

	// Analyze IP patterns
	suspiciousActivities = append(suspiciousActivities, ad.analyzeIPPatterns(profile, sessions)...)

	// Analyze device patterns
	suspiciousActivities = append(suspiciousActivities, ad.analyzeDevicePatterns(profile, sessions)...)

	// Analyze timing patterns
	suspiciousActivities = append(suspiciousActivities, ad.analyzeTimingPatterns(profile, sessions)...)

	// Analyze geographic patterns (if enabled)
	if ad.config.GeoLocationCheckEnabled {
		suspiciousActivities = append(suspiciousActivities, ad.analyzeGeographicPatterns(profile, sessions)...)
	}

	// Update profile with new data
	ad.updateProfile(profile, sessions)

	return suspiciousActivities
}

// CheckRequest performs real-time anomaly detection on a single request
func (ad *AnomalyDetector) CheckRequest(userID int, ip, userAgent string) bool {
	profile := ad.getOrCreateProfile(userID)
	
	profile.mutex.Lock()
	defer profile.mutex.Unlock()

	now := time.Now()

	// Check for rapid successive logins
	if !profile.LastActivity.IsZero() && now.Sub(profile.LastActivity) < ad.config.RapidLoginThreshold {
		ad.addSuspiciousEvent(profile, "RAPID_LOGIN", "Multiple login attempts in short timeframe", "MEDIUM", ip, userAgent)
		return true
	}

	// Check for unknown IP
	if _, exists := profile.KnownIPs[ip]; !exists && len(profile.KnownIPs) > 0 {
		ad.addSuspiciousEvent(profile, "UNKNOWN_IP", "Login from previously unseen IP address", "LOW", ip, userAgent)
	}

	// Check for unknown device
	if _, exists := profile.KnownUserAgents[userAgent]; !exists && len(profile.KnownUserAgents) > 0 {
		ad.addSuspiciousEvent(profile, "UNKNOWN_DEVICE", "Login from previously unseen device", "LOW", ip, userAgent)
	}

	// Check if IP has high threat score
	if ipInfo, exists := profile.KnownIPs[ip]; exists && ipInfo.ThreatScore > ad.config.SuspiciousIPThreshold {
		ad.addSuspiciousEvent(profile, "HIGH_THREAT_IP", "Login from high-threat IP address", "HIGH", ip, userAgent)
		return true
	}

	profile.LastActivity = now
	return false
}

// GetUserProfile returns the behavior profile for a user
func (ad *AnomalyDetector) GetUserProfile(userID int) *UserProfile {
	ad.mutex.RLock()
	defer ad.mutex.RUnlock()

	if profile, exists := ad.userProfiles[userID]; exists {
		// Return a copy to prevent concurrent modification
		return ad.copyProfile(profile)
	}

	return nil
}

// ResetUserProfile clears the profile for a user
func (ad *AnomalyDetector) ResetUserProfile(userID int) {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	delete(ad.userProfiles, userID)
}

// MarkIPAsSuspicious marks an IP address as suspicious for a user
func (ad *AnomalyDetector) MarkIPAsSuspicious(userID int, ip string, threatScore int) {
	profile := ad.getOrCreateProfile(userID)
	
	profile.mutex.Lock()
	defer profile.mutex.Unlock()

	if ipInfo, exists := profile.KnownIPs[ip]; exists {
		ipInfo.IsSuspicious = true
		ipInfo.ThreatScore = threatScore
	}
}

// GetSuspiciousActivities returns recent suspicious activities for a user
func (ad *AnomalyDetector) GetSuspiciousActivities(userID int, since time.Time) []*SuspiciousActivity {
	profile := ad.getOrCreateProfile(userID)
	
	profile.mutex.RLock()
	defer profile.mutex.RUnlock()

	var recent []*SuspiciousActivity
	for _, activity := range profile.SuspiciousEvents {
		if activity.DetectedAt.After(since) {
			recent = append(recent, activity)
		}
	}

	return recent
}

// Private helper methods

func (ad *AnomalyDetector) getOrCreateProfile(userID int) *UserProfile {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	if profile, exists := ad.userProfiles[userID]; exists {
		return profile
	}

	profile := &UserProfile{
		UserID:          userID,
		KnownIPs:        make(map[string]*IPInfo),
		KnownUserAgents: make(map[string]*DeviceInfo),
		LoginPattern: &LoginPattern{
			TypicalHours: make([]int, 0),
			TypicalDays:  make([]int, 0),
		},
		SuspiciousEvents: make([]*SuspiciousActivity, 0),
	}

	ad.userProfiles[userID] = profile
	return profile
}

func (ad *AnomalyDetector) analyzeIPPatterns(profile *UserProfile, sessions []*models.Session) []*SuspiciousActivity {
	var activities []*SuspiciousActivity

	profile.mutex.RLock()
	defer profile.mutex.RUnlock()

	// Count unique IPs in recent sessions
	recentIPs := make(map[string]bool)
	for _, session := range sessions {
		if session.IPAddress != "" {
			recentIPs[session.IPAddress] = true
		}
	}

	// Check for too many different IPs
	if len(recentIPs) > ad.config.MaxIPsPerUser {
		activity := &SuspiciousActivity{
			Type:        "MULTIPLE_IPS",
			Description: "User accessed from too many different IP addresses",
			Severity:    "MEDIUM",
			DetectedAt:  time.Now(),
			UserID:      profile.UserID,
		}
		activities = append(activities, activity)
	}

	// Check for suspicious IP patterns
	for ip := range recentIPs {
		if ad.isIPSuspicious(ip) {
			activity := &SuspiciousActivity{
				Type:        "SUSPICIOUS_IP",
				Description: "Access from potentially suspicious IP address",
				Severity:    "HIGH",
				DetectedAt:  time.Now(),
				UserID:      profile.UserID,
				IPAddress:   ip,
			}
			activities = append(activities, activity)
		}
	}

	return activities
}

func (ad *AnomalyDetector) analyzeDevicePatterns(profile *UserProfile, sessions []*models.Session) []*SuspiciousActivity {
	var activities []*SuspiciousActivity

	profile.mutex.RLock()
	defer profile.mutex.RUnlock()

	// Count unique user agents
	recentDevices := make(map[string]bool)
	for _, session := range sessions {
		if session.UserAgent != "" {
			recentDevices[session.UserAgent] = true
		}
	}

	// Check for too many different devices
	if len(recentDevices) > ad.config.MaxDevicesPerUser {
		activity := &SuspiciousActivity{
			Type:        "MULTIPLE_DEVICES",
			Description: "User accessed from too many different devices",
			Severity:    "MEDIUM",
			DetectedAt:  time.Now(),
			UserID:      profile.UserID,
		}
		activities = append(activities, activity)
	}

	return activities
}

func (ad *AnomalyDetector) analyzeTimingPatterns(profile *UserProfile, sessions []*models.Session) []*SuspiciousActivity {
	var activities []*SuspiciousActivity

	if len(sessions) < 2 {
		return activities
	}

	// Check for rapid successive logins
	for i := 1; i < len(sessions); i++ {
		timeDiff := sessions[i-1].CreatedAt.Sub(sessions[i].CreatedAt)
		if timeDiff < ad.config.RapidLoginThreshold && timeDiff > 0 {
			activity := &SuspiciousActivity{
				Type:        "RAPID_LOGINS",
				Description: "Rapid successive login attempts detected",
				Severity:    "MEDIUM",
				DetectedAt:  time.Now(),
				UserID:      profile.UserID,
				SessionID:   sessions[i].ID,
				IPAddress:   sessions[i].IPAddress,
				UserAgent:   sessions[i].UserAgent,
			}
			activities = append(activities, activity)
		}
	}

	return activities
}

func (ad *AnomalyDetector) analyzeGeographicPatterns(profile *UserProfile, sessions []*models.Session) []*SuspiciousActivity {
	// This would implement geolocation-based anomaly detection
	// For now, it's a placeholder that would require external geolocation services
	var activities []*SuspiciousActivity
	return activities
}

func (ad *AnomalyDetector) updateProfile(profile *UserProfile, sessions []*models.Session) {
	profile.mutex.Lock()
	defer profile.mutex.Unlock()

	now := time.Now()

	for _, session := range sessions {
		// Update IP information
		if session.IPAddress != "" {
			if ipInfo, exists := profile.KnownIPs[session.IPAddress]; exists {
				ipInfo.LastSeen = session.LastUsedAt
				ipInfo.LoginCount++
			} else {
				profile.KnownIPs[session.IPAddress] = &IPInfo{
					IPAddress:  session.IPAddress,
					FirstSeen:  session.CreatedAt,
					LastSeen:   session.LastUsedAt,
					LoginCount: 1,
				}
			}
		}

		// Update device information
		if session.UserAgent != "" {
			if deviceInfo, exists := profile.KnownUserAgents[session.UserAgent]; exists {
				deviceInfo.LastSeen = session.LastUsedAt
				deviceInfo.SessionCount++
			} else {
				profile.KnownUserAgents[session.UserAgent] = &DeviceInfo{
					UserAgent:    session.UserAgent,
					LastSeen:     session.LastUsedAt,
					SessionCount: 1,
				}
			}
		}

		// Update login patterns
		hour := session.CreatedAt.Hour()
		day := int(session.CreatedAt.Weekday())
		
		profile.LoginPattern.TypicalHours = ad.updateIntSlice(profile.LoginPattern.TypicalHours, hour)
		profile.LoginPattern.TypicalDays = ad.updateIntSlice(profile.LoginPattern.TypicalDays, day)
		profile.LoginPattern.LoginCount++
		profile.LoginPattern.LastLogin = session.CreatedAt
	}

	profile.LastActivity = now
}

func (ad *AnomalyDetector) isIPSuspicious(ip string) bool {
	// Simple heuristics for suspicious IPs
	
	// Check for private/local IPs (generally not suspicious)
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		if parsedIP.IsLoopback() || parsedIP.IsPrivate() {
			return false
		}
	}

	// Check for known suspicious patterns
	suspiciousPatterns := []string{
		"tor-", "proxy-", "vpn-", "anonymizer",
	}

	lowerIP := strings.ToLower(ip)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerIP, pattern) {
			return true
		}
	}

	// Could add more sophisticated checks here:
	// - Threat intelligence feeds
	// - Geolocation checks
	// - Reputation databases

	return false
}

func (ad *AnomalyDetector) updateIntSlice(slice []int, value int) []int {
	// Add value if not already present
	for _, existing := range slice {
		if existing == value {
			return slice
		}
	}
	return append(slice, value)
}

func (ad *AnomalyDetector) addSuspiciousEvent(profile *UserProfile, eventType, description, severity, ip, userAgent string) {
	event := &SuspiciousActivity{
		Type:        eventType,
		Description: description,
		Severity:    severity,
		DetectedAt:  time.Now(),
		UserID:      profile.UserID,
		IPAddress:   ip,
		UserAgent:   userAgent,
	}

	// Keep only recent events (last 100)
	profile.SuspiciousEvents = append(profile.SuspiciousEvents, event)
	if len(profile.SuspiciousEvents) > 100 {
		profile.SuspiciousEvents = profile.SuspiciousEvents[1:]
	}
}

func (ad *AnomalyDetector) copyProfile(original *UserProfile) *UserProfile {
	original.mutex.RLock()
	defer original.mutex.RUnlock()

	// Create a shallow copy for read-only access
	copy := &UserProfile{
		UserID:       original.UserID,
		LastActivity: original.LastActivity,
	}

	// Copy maps
	copy.KnownIPs = make(map[string]*IPInfo)
	for k, v := range original.KnownIPs {
		copy.KnownIPs[k] = v
	}

	copy.KnownUserAgents = make(map[string]*DeviceInfo)
	for k, v := range original.KnownUserAgents {
		copy.KnownUserAgents[k] = v
	}

	// Copy login pattern
	if original.LoginPattern != nil {
		copy.LoginPattern = &LoginPattern{
			TypicalHours:    append([]int(nil), original.LoginPattern.TypicalHours...),
			TypicalDays:     append([]int(nil), original.LoginPattern.TypicalDays...),
			AverageInterval: original.LoginPattern.AverageInterval,
			LastLogin:       original.LoginPattern.LastLogin,
			LoginCount:      original.LoginPattern.LoginCount,
		}
	}

	// Copy suspicious events
	copy.SuspiciousEvents = append([]*SuspiciousActivity(nil), original.SuspiciousEvents...)

	return copy
}

func (ad *AnomalyDetector) startCleanup() {
	ticker := time.NewTicker(ad.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ad.cleanup()
		}
	}
}

func (ad *AnomalyDetector) cleanup() {
	ad.mutex.Lock()
	defer ad.mutex.Unlock()

	cutoff := time.Now().Add(-7 * 24 * time.Hour) // Keep 7 days of data

	for userID, profile := range ad.userProfiles {
		profile.mutex.Lock()

		// Remove old suspicious events
		recentEvents := make([]*SuspiciousActivity, 0)
		for _, event := range profile.SuspiciousEvents {
			if event.DetectedAt.After(cutoff) {
				recentEvents = append(recentEvents, event)
			}
		}
		profile.SuspiciousEvents = recentEvents

		// Remove profiles with no recent activity
		if profile.LastActivity.Before(cutoff) && len(profile.SuspiciousEvents) == 0 {
			profile.mutex.Unlock()
			delete(ad.userProfiles, userID)
			continue
		}

		profile.mutex.Unlock()
	}
}