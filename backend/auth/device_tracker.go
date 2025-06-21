package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// DeviceTracker tracks and manages user devices
type DeviceTracker struct {
	userDevices map[int]map[string]*DeviceInfo
	mutex       sync.RWMutex
	config      *DeviceTrackingConfig
}

// DeviceTrackingConfig holds configuration for device tracking
type DeviceTrackingConfig struct {
	MaxDevicesPerUser   int           `yaml:"max_devices_per_user" json:"max_devices_per_user"`
	DeviceRetention     time.Duration `yaml:"device_retention" json:"device_retention"`
	AutoTrustThreshold  int           `yaml:"auto_trust_threshold" json:"auto_trust_threshold"`
	RequireVerification bool          `yaml:"require_verification" json:"require_verification"`
	CleanupInterval     time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// DeviceInfo represents information about a device
type DeviceInfo struct {
	Fingerprint   string    `json:"fingerprint"`
	UserAgent     string    `json:"user_agent"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	IPAddresses   []string  `json:"ip_addresses"`
	SessionCount  int       `json:"session_count"`
	Trusted       bool      `json:"trusted"`
	Verified      bool      `json:"verified"`
	DeviceType    string    `json:"device_type"`
	Browser       string    `json:"browser"`
	OS            string    `json:"os"`
	ThreatScore   int       `json:"threat_score"`
	Blocked       bool      `json:"blocked"`
	BlockedReason string    `json:"blocked_reason,omitempty"`
	mutex         sync.RWMutex
}

// NewDeviceTracker creates a new device tracker
func NewDeviceTracker() *DeviceTracker {
	config := &DeviceTrackingConfig{
		MaxDevicesPerUser:   10,
		DeviceRetention:     90 * 24 * time.Hour, // 90 days
		AutoTrustThreshold:  5,                   // Trust after 5 successful sessions
		RequireVerification: false,
		CleanupInterval:     24 * time.Hour, // Daily cleanup
	}

	dt := &DeviceTracker{
		userDevices: make(map[int]map[string]*DeviceInfo),
		config:      config,
	}

	// Start cleanup goroutine
	go dt.startCleanup()

	return dt
}

// TrackDevice records or updates device information
func (dt *DeviceTracker) TrackDevice(userID int, fingerprint, ip, userAgent string) *DeviceInfo {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	// Initialize user devices map if not exists
	if dt.userDevices[userID] == nil {
		dt.userDevices[userID] = make(map[string]*DeviceInfo)
	}

	userDevices := dt.userDevices[userID]
	now := time.Now()

	if device, exists := userDevices[fingerprint]; exists {
		// Update existing device
		device.mutex.Lock()
		device.LastSeen = now
		device.SessionCount++
		
		// Add IP address if not already tracked
		dt.addIPAddress(device, ip)
		
		// Auto-trust device if threshold reached
		if !device.Trusted && device.SessionCount >= dt.config.AutoTrustThreshold {
			device.Trusted = true
		}
		
		device.mutex.Unlock()
		return device
	}

	// Create new device
	device := &DeviceInfo{
		Fingerprint:  fingerprint,
		UserAgent:    userAgent,
		FirstSeen:    now,
		LastSeen:     now,
		IPAddresses:  []string{ip},
		SessionCount: 1,
		Trusted:      false,
		Verified:     false,
	}

	// Parse device information from user agent
	dt.parseDeviceInfo(device)

	// Check if we need to remove old devices
	if len(userDevices) >= dt.config.MaxDevicesPerUser {
		dt.removeOldestDevice(userDevices)
	}

	userDevices[fingerprint] = device
	return device
}

// GetUserDevices returns all devices for a user
func (dt *DeviceTracker) GetUserDevices(userID int) []*DeviceInfo {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return nil
	}

	devices := make([]*DeviceInfo, 0, len(userDevices))
	for _, device := range userDevices {
		devices = append(devices, dt.copyDeviceInfo(device))
	}

	return devices
}

// GetDevice returns a specific device for a user
func (dt *DeviceTracker) GetDevice(userID int, fingerprint string) *DeviceInfo {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return nil
	}

	if device, exists := userDevices[fingerprint]; exists {
		return dt.copyDeviceInfo(device)
	}

	return nil
}

// TrustDevice marks a device as trusted
func (dt *DeviceTracker) TrustDevice(userID int, fingerprint string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return &DeviceError{
			Code:    "USER_NOT_FOUND",
			Message: "No devices found for user",
		}
	}

	device, exists := userDevices[fingerprint]
	if !exists {
		return &DeviceError{
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		}
	}

	device.mutex.Lock()
	device.Trusted = true
	device.mutex.Unlock()

	return nil
}

// UntrustDevice marks a device as untrusted
func (dt *DeviceTracker) UntrustDevice(userID int, fingerprint string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return &DeviceError{
			Code:    "USER_NOT_FOUND",
			Message: "No devices found for user",
		}
	}

	device, exists := userDevices[fingerprint]
	if !exists {
		return &DeviceError{
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		}
	}

	device.mutex.Lock()
	device.Trusted = false
	device.mutex.Unlock()

	return nil
}

// VerifyDevice marks a device as verified
func (dt *DeviceTracker) VerifyDevice(userID int, fingerprint string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return &DeviceError{
			Code:    "USER_NOT_FOUND",
			Message: "No devices found for user",
		}
	}

	device, exists := userDevices[fingerprint]
	if !exists {
		return &DeviceError{
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		}
	}

	device.mutex.Lock()
	device.Verified = true
	device.Trusted = true // Verified devices are automatically trusted
	device.mutex.Unlock()

	return nil
}

// BlockDevice blocks a device
func (dt *DeviceTracker) BlockDevice(userID int, fingerprint, reason string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return &DeviceError{
			Code:    "USER_NOT_FOUND",
			Message: "No devices found for user",
		}
	}

	device, exists := userDevices[fingerprint]
	if !exists {
		return &DeviceError{
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		}
	}

	device.mutex.Lock()
	device.Blocked = true
	device.BlockedReason = reason
	device.Trusted = false
	device.mutex.Unlock()

	return nil
}

// UnblockDevice unblocks a device
func (dt *DeviceTracker) UnblockDevice(userID int, fingerprint string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return &DeviceError{
			Code:    "USER_NOT_FOUND",
			Message: "No devices found for user",
		}
	}

	device, exists := userDevices[fingerprint]
	if !exists {
		return &DeviceError{
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		}
	}

	device.mutex.Lock()
	device.Blocked = false
	device.BlockedReason = ""
	device.mutex.Unlock()

	return nil
}

// RemoveDevice removes a device for a user
func (dt *DeviceTracker) RemoveDevice(userID int, fingerprint string) error {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	userDevices, exists := dt.userDevices[userID]
	if !exists {
		return &DeviceError{
			Code:    "USER_NOT_FOUND",
			Message: "No devices found for user",
		}
	}

	if _, exists := userDevices[fingerprint]; !exists {
		return &DeviceError{
			Code:    "DEVICE_NOT_FOUND",
			Message: "Device not found",
		}
	}

	delete(userDevices, fingerprint)
	return nil
}

// IsDeviceTrusted checks if a device is trusted
func (dt *DeviceTracker) IsDeviceTrusted(userID int, fingerprint string) bool {
	device := dt.GetDevice(userID, fingerprint)
	return device != nil && device.Trusted && !device.Blocked
}

// IsDeviceBlocked checks if a device is blocked
func (dt *DeviceTracker) IsDeviceBlocked(userID int, fingerprint string) bool {
	device := dt.GetDevice(userID, fingerprint)
	return device != nil && device.Blocked
}

// GetDeviceStats returns statistics about tracked devices
func (dt *DeviceTracker) GetDeviceStats() *DeviceStats {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	stats := &DeviceStats{
		TotalUsers:      len(dt.userDevices),
		TotalDevices:    0,
		TrustedDevices:  0,
		VerifiedDevices: 0,
		BlockedDevices:  0,
		DeviceTypes:     make(map[string]int),
		Browsers:        make(map[string]int),
		OperatingSystems: make(map[string]int),
	}

	for _, userDevices := range dt.userDevices {
		for _, device := range userDevices {
			device.mutex.RLock()
			
			stats.TotalDevices++
			
			if device.Trusted {
				stats.TrustedDevices++
			}
			
			if device.Verified {
				stats.VerifiedDevices++
			}
			
			if device.Blocked {
				stats.BlockedDevices++
			}
			
			if device.DeviceType != "" {
				stats.DeviceTypes[device.DeviceType]++
			}
			
			if device.Browser != "" {
				stats.Browsers[device.Browser]++
			}
			
			if device.OS != "" {
				stats.OperatingSystems[device.OS]++
			}
			
			device.mutex.RUnlock()
		}
	}

	return stats
}

// GetSuspiciousDevices returns devices with high threat scores
func (dt *DeviceTracker) GetSuspiciousDevices(threshold int) []*SuspiciousDevice {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	var suspicious []*SuspiciousDevice

	for userID, userDevices := range dt.userDevices {
		for _, device := range userDevices {
			device.mutex.RLock()
			
			if device.ThreatScore >= threshold {
				suspicious = append(suspicious, &SuspiciousDevice{
					UserID:      userID,
					DeviceInfo:  dt.copyDeviceInfo(device),
					ThreatScore: device.ThreatScore,
				})
			}
			
			device.mutex.RUnlock()
		}
	}

	return suspicious
}

// Private helper methods

func (dt *DeviceTracker) addIPAddress(device *DeviceInfo, ip string) {
	// Check if IP already exists
	for _, existingIP := range device.IPAddresses {
		if existingIP == ip {
			return
		}
	}

	// Add new IP
	device.IPAddresses = append(device.IPAddresses, ip)

	// Keep only recent IPs (limit to 10)
	if len(device.IPAddresses) > 10 {
		device.IPAddresses = device.IPAddresses[1:]
	}
}

func (dt *DeviceTracker) parseDeviceInfo(device *DeviceInfo) {
	userAgent := strings.ToLower(device.UserAgent)

	// Parse device type
	if strings.Contains(userAgent, "mobile") || strings.Contains(userAgent, "android") || strings.Contains(userAgent, "iphone") {
		device.DeviceType = "mobile"
	} else if strings.Contains(userAgent, "tablet") || strings.Contains(userAgent, "ipad") {
		device.DeviceType = "tablet"
	} else {
		device.DeviceType = "desktop"
	}

	// Parse browser
	switch {
	case strings.Contains(userAgent, "chrome"):
		device.Browser = "Chrome"
	case strings.Contains(userAgent, "firefox"):
		device.Browser = "Firefox"
	case strings.Contains(userAgent, "safari"):
		device.Browser = "Safari"
	case strings.Contains(userAgent, "edge"):
		device.Browser = "Edge"
	case strings.Contains(userAgent, "opera"):
		device.Browser = "Opera"
	default:
		device.Browser = "Unknown"
	}

	// Parse OS
	switch {
	case strings.Contains(userAgent, "windows"):
		device.OS = "Windows"
	case strings.Contains(userAgent, "mac os"):
		device.OS = "macOS"
	case strings.Contains(userAgent, "linux"):
		device.OS = "Linux"
	case strings.Contains(userAgent, "android"):
		device.OS = "Android"
	case strings.Contains(userAgent, "ios"):
		device.OS = "iOS"
	default:
		device.OS = "Unknown"
	}

	// Calculate basic threat score
	device.ThreatScore = dt.calculateThreatScore(device)
}

func (dt *DeviceTracker) calculateThreatScore(device *DeviceInfo) int {
	score := 0

	// Unknown browser adds to threat score
	if device.Browser == "Unknown" {
		score += 2
	}

	// Unknown OS adds to threat score
	if device.OS == "Unknown" {
		score += 2
	}

	// Suspicious user agent patterns
	userAgent := strings.ToLower(device.UserAgent)
	suspiciousPatterns := []string{
		"bot", "crawler", "spider", "scraper", "automation",
		"headless", "phantom", "selenium", "wget", "curl",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(userAgent, pattern) {
			score += 5
			break
		}
	}

	return score
}

func (dt *DeviceTracker) removeOldestDevice(userDevices map[string]*DeviceInfo) {
	var oldestFingerprint string
	var oldestTime time.Time

	for fingerprint, device := range userDevices {
		device.mutex.RLock()
		
		// Don't remove trusted or verified devices
		if device.Trusted || device.Verified {
			device.mutex.RUnlock()
			continue
		}

		if oldestFingerprint == "" || device.LastSeen.Before(oldestTime) {
			oldestFingerprint = fingerprint
			oldestTime = device.LastSeen
		}
		
		device.mutex.RUnlock()
	}

	if oldestFingerprint != "" {
		delete(userDevices, oldestFingerprint)
	}
}

func (dt *DeviceTracker) copyDeviceInfo(original *DeviceInfo) *DeviceInfo {
	original.mutex.RLock()
	defer original.mutex.RUnlock()

	return &DeviceInfo{
		Fingerprint:   original.Fingerprint,
		UserAgent:     original.UserAgent,
		FirstSeen:     original.FirstSeen,
		LastSeen:      original.LastSeen,
		IPAddresses:   append([]string(nil), original.IPAddresses...),
		SessionCount:  original.SessionCount,
		Trusted:       original.Trusted,
		Verified:      original.Verified,
		DeviceType:    original.DeviceType,
		Browser:       original.Browser,
		OS:            original.OS,
		ThreatScore:   original.ThreatScore,
		Blocked:       original.Blocked,
		BlockedReason: original.BlockedReason,
	}
}

func (dt *DeviceTracker) startCleanup() {
	ticker := time.NewTicker(dt.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dt.cleanup()
		}
	}
}

func (dt *DeviceTracker) cleanup() {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	cutoff := time.Now().Add(-dt.config.DeviceRetention)

	for userID, userDevices := range dt.userDevices {
		for fingerprint, device := range userDevices {
			device.mutex.RLock()
			
			// Don't remove trusted, verified, or recently used devices
			shouldRemove := !device.Trusted && !device.Verified && device.LastSeen.Before(cutoff)
			
			device.mutex.RUnlock()

			if shouldRemove {
				delete(userDevices, fingerprint)
			}
		}

		// Remove empty user device maps
		if len(userDevices) == 0 {
			delete(dt.userDevices, userID)
		}
	}
}

// GenerateDeviceFingerprint creates a device fingerprint from request headers
func GenerateDeviceFingerprint(userAgent, acceptLanguage, acceptEncoding, accept string) string {
	components := []string{userAgent, acceptLanguage, acceptEncoding, accept}
	data := strings.Join(components, "|")
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// DeviceError represents a device-related error
type DeviceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *DeviceError) Error() string {
	return e.Message
}

// DeviceStats represents device statistics
type DeviceStats struct {
	TotalUsers       int            `json:"total_users"`
	TotalDevices     int            `json:"total_devices"`
	TrustedDevices   int            `json:"trusted_devices"`
	VerifiedDevices  int            `json:"verified_devices"`
	BlockedDevices   int            `json:"blocked_devices"`
	DeviceTypes      map[string]int `json:"device_types"`
	Browsers         map[string]int `json:"browsers"`
	OperatingSystems map[string]int `json:"operating_systems"`
}

// SuspiciousDevice represents a device flagged as suspicious
type SuspiciousDevice struct {
	UserID      int         `json:"user_id"`
	DeviceInfo  *DeviceInfo `json:"device_info"`
	ThreatScore int         `json:"threat_score"`
}