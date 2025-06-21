package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"qt1-middleware/config"
)

// IPProtectionMiddleware provides comprehensive IP-based protection
type IPProtectionMiddleware struct {
	allowlist     map[string]bool
	blocklist     map[string]bool
	geolocator    GeoLocator
	reputation    IPReputationService
	config        *config.IPProtectionConfig
	suspiciousIPs map[string]*SuspiciousActivity
	mutex         sync.RWMutex
	monitor       *SecurityMonitor
}

// SuspiciousActivity tracks suspicious activity from an IP
type SuspiciousActivity struct {
	IP             string    `json:"ip"`
	ViolationCount int       `json:"violation_count"`
	FirstViolation time.Time `json:"first_violation"`
	LastViolation  time.Time `json:"last_violation"`
	Reason         string    `json:"reason"`
	BlockedUntil   time.Time `json:"blocked_until"`
	Country        string    `json:"country,omitempty"`
	ReputationScore float64  `json:"reputation_score,omitempty"`
}

// GeoLocation represents geographical location data
type GeoLocation struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}

// GeoLocator interface for IP geolocation services
type GeoLocator interface {
	GetLocation(ip string) (*GeoLocation, error)
	IsValidIP(ip string) bool
}

// IPReputationService interface for IP reputation checking
type IPReputationService interface {
	CheckReputation(ip string) (*IPReputation, error)
	IsKnownMalicious(ip string) bool
}

// IPReputation represents IP reputation data
type IPReputation struct {
	IP            string    `json:"ip"`
	Score         float64   `json:"score"`         // 0.0 (safe) to 1.0 (malicious)
	Categories    []string  `json:"categories"`    // malware, spam, phishing, etc.
	LastSeen      time.Time `json:"last_seen"`
	FirstSeen     time.Time `json:"first_seen"`
	Confidence    float64   `json:"confidence"`    // 0.0 to 1.0
	Sources       []string  `json:"sources"`       // reputation data sources
	IsMalicious   bool      `json:"is_malicious"`
	Description   string    `json:"description"`
}

// MockGeoLocator provides a simple mock implementation for testing
type MockGeoLocator struct {
	locations map[string]*GeoLocation
}

// NewMockGeoLocator creates a new mock geolocator with sample data
func NewMockGeoLocator() *MockGeoLocator {
	return &MockGeoLocator{
		locations: map[string]*GeoLocation{
			"1.1.1.1":       {IP: "1.1.1.1", Country: "United States", CountryCode: "US"},
			"8.8.8.8":       {IP: "8.8.8.8", Country: "United States", CountryCode: "US"},
			"185.220.100.1": {IP: "185.220.100.1", Country: "Russia", CountryCode: "RU"},
			"103.224.182.1": {IP: "103.224.182.1", Country: "China", CountryCode: "CN"},
			"37.187.96.1":   {IP: "37.187.96.1", Country: "France", CountryCode: "FR"},
			"192.168.1.1":   {IP: "192.168.1.1", Country: "Local", CountryCode: "LOCAL"},
			"127.0.0.1":     {IP: "127.0.0.1", Country: "Local", CountryCode: "LOCAL"},
		},
	}
}

// GetLocation returns location data for an IP
func (m *MockGeoLocator) GetLocation(ip string) (*GeoLocation, error) {
	if location, exists := m.locations[ip]; exists {
		return location, nil
	}
	
	// Return default for unknown IPs
	return &GeoLocation{
		IP:          ip,
		Country:     "Unknown",
		CountryCode: "XX",
	}, nil
}

// IsValidIP checks if an IP is valid for geolocation
func (m *MockGeoLocator) IsValidIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	
	// Don't geolocate private IPs
	return !parsedIP.IsPrivate() && !parsedIP.IsLoopback()
}

// MockIPReputationService provides a simple mock implementation
type MockIPReputationService struct {
	reputations map[string]*IPReputation
}

// NewMockIPReputationService creates a new mock reputation service
func NewMockIPReputationService() *MockIPReputationService {
	return &MockIPReputationService{
		reputations: map[string]*IPReputation{
			"185.220.100.1": {
				IP:          "185.220.100.1",
				Score:       0.9,
				Categories:  []string{"tor", "proxy"},
				IsMalicious: true,
				Confidence:  0.8,
				Description: "Known Tor exit node",
			},
			"1.2.3.4": {
				IP:          "1.2.3.4",
				Score:       0.8,
				Categories:  []string{"malware", "botnet"},
				IsMalicious: true,
				Confidence:  0.9,
				Description: "Known malware C&C server",
			},
			"192.168.1.100": {
				IP:          "192.168.1.100",
				Score:       0.7,
				Categories:  []string{"scanning"},
				IsMalicious: false,
				Confidence:  0.6,
				Description: "Suspicious scanning activity",
			},
		},
	}
}

// CheckReputation returns reputation data for an IP
func (m *MockIPReputationService) CheckReputation(ip string) (*IPReputation, error) {
	if rep, exists := m.reputations[ip]; exists {
		return rep, nil
	}
	
	// Return clean reputation for unknown IPs
	return &IPReputation{
		IP:          ip,
		Score:       0.1,
		Categories:  []string{},
		IsMalicious: false,
		Confidence:  0.5,
		Description: "No reputation data available",
	}, nil
}

// IsKnownMalicious checks if an IP is known to be malicious
func (m *MockIPReputationService) IsKnownMalicious(ip string) bool {
	if rep, exists := m.reputations[ip]; exists {
		return rep.IsMalicious
	}
	return false
}

// NewIPProtectionMiddleware creates a new IP protection middleware
func NewIPProtectionMiddleware(cfg *config.IPProtectionConfig, monitor *SecurityMonitor) *IPProtectionMiddleware {
	ipm := &IPProtectionMiddleware{
		allowlist:     make(map[string]bool),
		blocklist:     make(map[string]bool),
		config:        cfg,
		suspiciousIPs: make(map[string]*SuspiciousActivity),
		monitor:       monitor,
		geolocator:    NewMockGeoLocator(),    // In production, use real service
		reputation:    NewMockIPReputationService(), // In production, use real service
	}
	
	// Initialize allowlist and blocklist from config
	ipm.initializeLists()
	
	return ipm
}

// initializeLists initializes allowlist and blocklist from configuration
func (ipm *IPProtectionMiddleware) initializeLists() {
	if ipm.config == nil {
		return
	}
	
	// Initialize allowlist
	for _, ip := range ipm.config.AllowedIPs {
		ipm.allowlist[ip] = true
	}
	
	// Initialize blocklist
	for _, ip := range ipm.config.BlockedIPs {
		ipm.blocklist[ip] = true
	}
}

// Handler returns the HTTP middleware handler for IP protection
func (ipm *IPProtectionMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ipm.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		
		clientIP := ipm.getClientIP(r)
		
		// Check IP access permissions
		if err := ipm.checkIPAccess(clientIP, r); err != nil {
			ipm.logSecurityEvent(SecurityEvent{
				EventType:   "ip_access_denied",
				Severity:    "high",
				ClientIP:    clientIP,
				UserAgent:   r.UserAgent(),
				RequestPath: r.URL.Path,
				Method:      r.Method,
				Details:     err.Error(),
				Blocked:     true,
			})
			
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// checkIPAccess performs comprehensive IP access checking
func (ipm *IPProtectionMiddleware) checkIPAccess(ip string, r *http.Request) error {
	// Check allowlist first (allowlist overrides everything)
	if ipm.isAllowlisted(ip) {
		return nil
	}
	
	// Check blocklist
	if ipm.isBlocked(ip) {
		return fmt.Errorf("IP %s is explicitly blocked", ip)
	}
	
	// Check if IP is currently blocked for suspicious activity
	if ipm.isSuspiciouslyBlocked(ip) {
		return fmt.Errorf("IP %s is temporarily blocked due to suspicious activity", ip)
	}
	
	// Check geolocation if enabled
	if ipm.config.EnableGeoblocking {
		if err := ipm.checkGeolocation(ip); err != nil {
			return fmt.Errorf("geolocation check failed: %w", err)
		}
	}
	
	// Check IP reputation if enabled
	if ipm.config.EnableReputationCheck {
		if err := ipm.checkReputation(ip); err != nil {
			return fmt.Errorf("reputation check failed: %w", err)
		}
	}
	
	return nil
}

// isAllowlisted checks if an IP is in the allowlist
func (ipm *IPProtectionMiddleware) isAllowlisted(ip string) bool {
	ipm.mutex.RLock()
	defer ipm.mutex.RUnlock()
	
	// Check exact match
	if ipm.allowlist[ip] {
		return true
	}
	
	// Check CIDR ranges in allowlist
	for allowedIP := range ipm.allowlist {
		if ipm.isIPInCIDR(ip, allowedIP) {
			return true
		}
	}
	
	return false
}

// isBlocked checks if an IP is in the blocklist
func (ipm *IPProtectionMiddleware) isBlocked(ip string) bool {
	ipm.mutex.RLock()
	defer ipm.mutex.RUnlock()
	
	// Check exact match
	if ipm.blocklist[ip] {
		return true
	}
	
	// Check CIDR ranges in blocklist
	for blockedIP := range ipm.blocklist {
		if ipm.isIPInCIDR(ip, blockedIP) {
			return true
		}
	}
	
	return false
}

// isSuspiciouslyBlocked checks if an IP is temporarily blocked for suspicious activity
func (ipm *IPProtectionMiddleware) isSuspiciouslyBlocked(ip string) bool {
	ipm.mutex.RLock()
	defer ipm.mutex.RUnlock()
	
	if activity, exists := ipm.suspiciousIPs[ip]; exists {
		return time.Now().Before(activity.BlockedUntil)
	}
	
	return false
}

// checkGeolocation performs geolocation-based access control
func (ipm *IPProtectionMiddleware) checkGeolocation(ip string) error {
	if !ipm.geolocator.IsValidIP(ip) {
		return nil // Skip geolocation for private/local IPs
	}
	
	location, err := ipm.geolocator.GetLocation(ip)
	if err != nil {
		log.Printf("Geolocation lookup failed for IP %s: %v", ip, err)
		if ipm.config.BlockOnGeoError {
			return fmt.Errorf("geolocation lookup failed")
		}
		return nil // Allow if geolocation fails and BlockOnGeoError is false
	}
	
	// Check blocked countries
	for _, blockedCountry := range ipm.config.BlockedCountries {
		if strings.EqualFold(location.CountryCode, blockedCountry) ||
		   strings.EqualFold(location.Country, blockedCountry) {
			return fmt.Errorf("access denied from blocked country: %s", location.Country)
		}
	}
	
	// Check allowed countries (if specified, only these are allowed)
	if len(ipm.config.AllowedCountries) > 0 {
		allowed := false
		for _, allowedCountry := range ipm.config.AllowedCountries {
			if strings.EqualFold(location.CountryCode, allowedCountry) ||
			   strings.EqualFold(location.Country, allowedCountry) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("access denied from non-allowed country: %s", location.Country)
		}
	}
	
	return nil
}

// checkReputation performs IP reputation checking
func (ipm *IPProtectionMiddleware) checkReputation(ip string) error {
	reputation, err := ipm.reputation.CheckReputation(ip)
	if err != nil {
		log.Printf("Reputation check failed for IP %s: %v", ip, err)
		if ipm.config.BlockOnReputationError {
			return fmt.Errorf("reputation check failed")
		}
		return nil
	}
	
	// Block if reputation score exceeds threshold
	if reputation.Score >= ipm.config.ReputationThreshold {
		return fmt.Errorf("IP has poor reputation (score: %.2f, threshold: %.2f): %s", 
			reputation.Score, ipm.config.ReputationThreshold, reputation.Description)
	}
	
	// Block known malicious IPs regardless of score
	if reputation.IsMalicious && ipm.config.BlockMaliciousIPs {
		return fmt.Errorf("IP is known malicious: %s", reputation.Description)
	}
	
	return nil
}

// AddToBlocklist adds an IP to the blocklist
func (ipm *IPProtectionMiddleware) AddToBlocklist(ip string, reason string) {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	ipm.blocklist[ip] = true
	
	if ipm.monitor != nil {
		ipm.monitor.LogSecurityEvent(SecurityEvent{
			EventType: "ip_added_to_blocklist",
			Severity:  "medium",
			ClientIP:  ip,
			Details:   fmt.Sprintf("IP added to blocklist: %s", reason),
			Blocked:   false,
		})
	}
	
	log.Printf("IP %s added to blocklist: %s", ip, reason)
}

// RemoveFromBlocklist removes an IP from the blocklist
func (ipm *IPProtectionMiddleware) RemoveFromBlocklist(ip string) {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	delete(ipm.blocklist, ip)
	log.Printf("IP %s removed from blocklist", ip)
}

// AddToAllowlist adds an IP to the allowlist
func (ipm *IPProtectionMiddleware) AddToAllowlist(ip string) {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	ipm.allowlist[ip] = true
	log.Printf("IP %s added to allowlist", ip)
}

// RecordSuspiciousActivity records suspicious activity from an IP
func (ipm *IPProtectionMiddleware) RecordSuspiciousActivity(ip string, reason string) {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	now := time.Now()
	
	activity, exists := ipm.suspiciousIPs[ip]
	if !exists {
		activity = &SuspiciousActivity{
			IP:             ip,
			ViolationCount: 0,
			FirstViolation: now,
		}
		ipm.suspiciousIPs[ip] = activity
	}
	
	activity.ViolationCount++
	activity.LastViolation = now
	activity.Reason = reason
	
	// Auto-block if threshold exceeded
	if activity.ViolationCount >= ipm.config.SuspiciousThreshold {
		if ipm.config.AutoBlockDuration > 0 {
			activity.BlockedUntil = now.Add(ipm.config.AutoBlockDuration)
			
			if ipm.monitor != nil {
				ipm.monitor.LogSecurityEvent(SecurityEvent{
					EventType: "ip_auto_blocked",
					Severity:  "high",
					ClientIP:  ip,
					Details:   fmt.Sprintf("IP auto-blocked after %d violations: %s", activity.ViolationCount, reason),
					Blocked:   false,
				})
			}
			
			log.Printf("IP %s auto-blocked for %v after %d violations: %s", 
				ip, ipm.config.AutoBlockDuration, activity.ViolationCount, reason)
		}
	}
}

// GetSuspiciousIPs returns a copy of current suspicious IPs
func (ipm *IPProtectionMiddleware) GetSuspiciousIPs() map[string]*SuspiciousActivity {
	ipm.mutex.RLock()
	defer ipm.mutex.RUnlock()
	
	result := make(map[string]*SuspiciousActivity)
	for ip, activity := range ipm.suspiciousIPs {
		// Create a copy
		result[ip] = &SuspiciousActivity{
			IP:             activity.IP,
			ViolationCount: activity.ViolationCount,
			FirstViolation: activity.FirstViolation,
			LastViolation:  activity.LastViolation,
			Reason:         activity.Reason,
			BlockedUntil:   activity.BlockedUntil,
			Country:        activity.Country,
			ReputationScore: activity.ReputationScore,
		}
	}
	
	return result
}

// isIPInCIDR checks if an IP is within a CIDR range
func (ipm *IPProtectionMiddleware) isIPInCIDR(ip, cidr string) bool {
	// If CIDR doesn't contain /, it's a single IP
	if !strings.Contains(cidr, "/") {
		return ip == cidr
	}
	
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}
	
	return network.Contains(ipAddr)
}

// getClientIP extracts the client IP from the request
func (ipm *IPProtectionMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		if firstIP := strings.Split(xff, ",")[0]; firstIP != "" {
			return strings.TrimSpace(firstIP)
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// logSecurityEvent logs a security event if monitor is available
func (ipm *IPProtectionMiddleware) logSecurityEvent(event SecurityEvent) {
	if ipm.monitor != nil {
		ipm.monitor.LogSecurityEvent(event)
	}
}

// Cleanup removes expired suspicious IP entries
func (ipm *IPProtectionMiddleware) Cleanup() {
	ipm.mutex.Lock()
	defer ipm.mutex.Unlock()
	
	now := time.Now()
	expiration := ipm.config.AutoBlockDuration * 2 // Keep data for twice the block duration
	
	for ip, activity := range ipm.suspiciousIPs {
		if now.Sub(activity.LastViolation) > expiration {
			delete(ipm.suspiciousIPs, ip)
		}
	}
}