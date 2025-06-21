package middleware

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"qt1-middleware/config"
)

// PromptInjectionDetector handles detection of prompt injection attempts
type PromptInjectionDetector struct {
	patterns []*regexp.Regexp
}

// NewPromptInjectionDetector creates a new detector with configured patterns
func NewPromptInjectionDetector() *PromptInjectionDetector {
	detector := &PromptInjectionDetector{}
	detector.compilePatterns()
	return detector
}

func (p *PromptInjectionDetector) compilePatterns() {
	basePatterns := config.AppConfig.Security.PromptInjection.Patterns
	
	// Add additional sophisticated patterns
	sophisticatedPatterns := []string{
		// Direct instruction override
		`(?i)(ignore|forget|disregard|override)\s+(previous|prior|all|any|the)\s+(instructions?|rules?|commands?|directions?)`,
		`(?i)(new|different|updated|revised)\s+(instructions?|rules?|commands?|system\s+message)`,
		
		// Role manipulation  
		`(?i)(you\s+are\s+now|act\s+as|pretend\s+to\s+be|roleplay\s+as)\s+(?:a\s+)?(?!assistant|ai|helpful)(\w+)`,
		`(?i)(switch\s+to|become\s+a|transform\s+into)\s+(?:a\s+)?(\w+)\s+(mode|character|persona)`,
		
		// System message manipulation
		`(?i)(the\s+)?(system\s+message|initial\s+prompt|base\s+instructions?)\s+(is|are|should\s+be|has\s+been)\s+(changed|modified|updated|replaced)`,
		
		// Jailbreak attempts
		`(?i)(developer|admin|god|root|sudo|debug|test)\s+(mode|access|privileges?)`,
		`(?i)(jailbreak|jail\s*break|break\s+out|escape\s+mode)`,
		`(?i)DAN\s+(mode|\d+|protocol)`,
		
		// Hypothetical scenarios
		`(?i)(in\s+a\s+)?(hypothetical|fictional|imaginary|alternate)\s+(world|universe|scenario|situation)`,
		`(?i)(what\s+if|suppose|imagine\s+if|let's\s+say)\s+(?:that\s+)?(?:you|I|we)`,
		
		// Encoding attempts (basic detection)
		`(?i)(base64|rot\d+|caesar|hex|unicode)\s*(decode|decod|decrypt)`,
		`(?i)(decode|decrypt|decipher)\s+(this|the\s+following)`,
		
		// Boundary testing
		`(?i)(how\s+)?(would\s+you\s+)?(respond|answer|react)\s+if\s+(you\s+were|I\s+told\s+you)`,
		`(?i)(tell\s+me\s+about|explain|describe)\s+(?:your\s+)?(limitations|restrictions|boundaries|rules)`,
		
		// System probing
		`(?i)(what\s+are\s+your|show\s+me\s+your|list\s+your)\s+(instructions|rules|guidelines|parameters)`,
		`(?i)(repeat|echo|output|print)\s+(your\s+)?(system\s+message|instructions|prompt)`,
	}
	
	allPatterns := append(basePatterns, sophisticatedPatterns...)
	
	for _, pattern := range allPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			p.patterns = append(p.patterns, compiled)
		}
	}
}

// DetectPromptInjection checks for prompt injection attempts
func (p *PromptInjectionDetector) DetectPromptInjection(message string) (bool, string) {
	if !config.AppConfig.Security.PromptInjection.Enabled {
		return false, ""
	}
	
	// Normalize the message for analysis
	normalized := p.normalizeMessage(message)
	
	// Check for direct pattern matches
	for _, pattern := range p.patterns {
		if pattern.MatchString(normalized) {
			return true, fmt.Sprintf("Potential prompt injection detected: pattern match")
		}
	}
	
	// Check for encoding attempts
	if p.detectEncodingAttempts(message) {
		return true, "Potential encoding-based injection attempt detected"
	}
	
	// Check for excessive special characters (bypass attempts)
	if p.detectBypassAttempts(message) {
		return true, "Potential bypass attempt detected"
	}
	
	// Check for repetitive injection keywords
	if p.detectRepetitiveKeywords(normalized) {
		return true, "Repetitive injection keywords detected"
	}
	
	return false, ""
}

func (p *PromptInjectionDetector) normalizeMessage(message string) string {
	// Convert to lowercase
	normalized := strings.ToLower(message)
	
	// Remove excessive whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	
	// Normalize common obfuscation attempts
	normalized = strings.ReplaceAll(normalized, "1", "i")
	normalized = strings.ReplaceAll(normalized, "3", "e")
	normalized = strings.ReplaceAll(normalized, "0", "o")
	normalized = strings.ReplaceAll(normalized, "5", "s")
	normalized = strings.ReplaceAll(normalized, "7", "t")
	
	// Remove common separators used in bypass attempts
	normalized = regexp.MustCompile(`[._\-*+]+`).ReplaceAllString(normalized, "")
	
	return strings.TrimSpace(normalized)
}

func (p *PromptInjectionDetector) detectEncodingAttempts(message string) bool {
	// Check for Base64 encoded content
	if p.looksLikeBase64(message) {
		if decoded, err := base64.StdEncoding.DecodeString(message); err == nil {
			decodedStr := string(decoded)
			// Recursively check decoded content
			if contains, _ := p.DetectPromptInjection(decodedStr); contains {
				return true
			}
		}
	}
	
	// Check for hex encoding
	if p.looksLikeHex(message) {
		return true
	}
	
	// Check for excessive unicode or special characters
	if p.hasExcessiveUnicode(message) {
		return true
	}
	
	return false
}

func (p *PromptInjectionDetector) detectBypassAttempts(message string) bool {
	// Count special characters
	specialCharCount := 0
	for _, char := range message {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && !unicode.IsSpace(char) {
			specialCharCount++
		}
	}
	
	// If more than 30% special characters, flag as suspicious
	if len(message) > 0 && float64(specialCharCount)/float64(len(message)) > 0.3 {
		return true
	}
	
	// Check for excessive repetition of bypass characters
	bypassChars := []string{".", "_", "-", "*", "+", "|", "\\", "/"}
	for _, char := range bypassChars {
		if strings.Count(message, char) > 10 {
			return true
		}
	}
	
	return false
}

func (p *PromptInjectionDetector) detectRepetitiveKeywords(message string) bool {
	keywords := []string{"ignore", "forget", "override", "jailbreak", "instructions", "system", "admin", "developer"}
	
	for _, keyword := range keywords {
		if strings.Count(message, keyword) > 2 {
			return true
		}
	}
	
	return false
}

func (p *PromptInjectionDetector) looksLikeBase64(s string) bool {
	// Basic heuristic for Base64 detection
	if len(s) < 10 || len(s)%4 != 0 {
		return false
	}
	
	base64Pattern := regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	return base64Pattern.MatchString(s)
}

func (p *PromptInjectionDetector) looksLikeHex(s string) bool {
	if len(s) < 10 || len(s)%2 != 0 {
		return false
	}
	
	hexPattern := regexp.MustCompile(`^[0-9A-Fa-f]+$`)
	return hexPattern.MatchString(s)
}

func (p *PromptInjectionDetector) hasExcessiveUnicode(s string) bool {
	unicodeCount := 0
	for _, char := range s {
		if char > 127 {
			unicodeCount++
		}
	}
	
	if len(s) > 0 && float64(unicodeCount)/float64(len(s)) > 0.2 {
		return true
	}
	
	return false
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	Severity    string    `json:"severity"`
	ClientIP    string    `json:"client_ip"`
	UserAgent   string    `json:"user_agent"`
	RequestPath string    `json:"request_path"`
	Method      string    `json:"method"`
	Details     string    `json:"details"`
	Blocked     bool      `json:"blocked"`
}

// SecurityMonitor handles security event monitoring and logging
type SecurityMonitor struct {
	config          *config.Config
	suspiciousIPs   map[string]*SuspiciousIPInfo
	mutex           sync.RWMutex
	eventChannel    chan SecurityEvent
	stopChannel     chan struct{}
}

// SuspiciousIPInfo tracks suspicious activity from an IP
type SuspiciousIPInfo struct {
	IP            string    `json:"ip"`
	ViolationCount int      `json:"violation_count"`
	FirstViolation time.Time `json:"first_violation"`
	LastViolation  time.Time `json:"last_violation"`
	BlockedUntil   time.Time `json:"blocked_until"`
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor(cfg *config.Config) *SecurityMonitor {
	sm := &SecurityMonitor{
		config:        cfg,
		suspiciousIPs: make(map[string]*SuspiciousIPInfo),
		eventChannel:  make(chan SecurityEvent, 1000),
		stopChannel:   make(chan struct{}),
	}
	
	// Start event processing goroutine
	go sm.processEvents()
	
	return sm
}

// LogSecurityEvent logs a security event
func (sm *SecurityMonitor) LogSecurityEvent(event SecurityEvent) {
	if !sm.config.Security.SecurityMonitoring.Enabled {
		return
	}
	
	// Add timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// Send to event channel for processing
	select {
	case sm.eventChannel <- event:
		// Event queued successfully
	default:
		// Channel full, log directly
		log.Printf("SECURITY EVENT: %+v", event)
	}
}

// processEvents processes security events in a background goroutine
func (sm *SecurityMonitor) processEvents() {
	for {
		select {
		case event := <-sm.eventChannel:
			sm.handleSecurityEvent(event)
		case <-sm.stopChannel:
			return
		}
	}
}

// handleSecurityEvent processes a single security event
func (sm *SecurityMonitor) handleSecurityEvent(event SecurityEvent) {
	// Log the event if enabled
	if sm.config.Security.SecurityMonitoring.LogSecurityEvents {
		log.Printf("SECURITY: [%s] %s from %s - %s (blocked: %v)", 
			event.Severity, event.EventType, event.ClientIP, event.Details, event.Blocked)
	}
	
	// Track suspicious IPs
	if event.ClientIP != "" && (event.Severity == "high" || event.Severity == "critical") {
		sm.trackSuspiciousIP(event.ClientIP, event)
	}
	
	// Send alerts for critical events
	if event.Severity == "critical" && sm.config.Security.SecurityMonitoring.AlertOnAttacks {
		sm.sendAlert(event)
	}
}

// trackSuspiciousIP tracks suspicious activity from an IP address
func (sm *SecurityMonitor) trackSuspiciousIP(ip string, event SecurityEvent) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	info, exists := sm.suspiciousIPs[ip]
	if !exists {
		info = &SuspiciousIPInfo{
			IP:            ip,
			ViolationCount: 0,
			FirstViolation: event.Timestamp,
		}
		sm.suspiciousIPs[ip] = info
	}
	
	info.ViolationCount++
	info.LastViolation = event.Timestamp
	
	// Check if IP should be blocked
	threshold := sm.config.Security.SecurityMonitoring.SuspiciousRequestThreshold
	if threshold > 0 && info.ViolationCount >= threshold {
		if sm.config.Security.SecurityMonitoring.BlockSuspiciousIPs {
			blockDuration := time.Duration(sm.config.Security.SecurityMonitoring.BlockDurationMinutes) * time.Minute
			info.BlockedUntil = time.Now().Add(blockDuration)
			
			log.Printf("SECURITY: Blocking IP %s for %v due to %d violations", 
				ip, blockDuration, info.ViolationCount)
		}
	}
}

// IsIPBlocked checks if an IP address is currently blocked
func (sm *SecurityMonitor) IsIPBlocked(ip string) bool {
	if !sm.config.Security.SecurityMonitoring.BlockSuspiciousIPs {
		return false
	}
	
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	info, exists := sm.suspiciousIPs[ip]
	if !exists {
		return false
	}
	
	return time.Now().Before(info.BlockedUntil)
}

// sendAlert sends an alert for critical security events
func (sm *SecurityMonitor) sendAlert(event SecurityEvent) {
	// In a production system, this would integrate with alerting systems
	// like email, Slack, PagerDuty, etc.
	log.Printf("SECURITY ALERT: Critical event detected - %s from %s: %s", 
		event.EventType, event.ClientIP, event.Details)
}

// Stop stops the security monitor
func (sm *SecurityMonitor) Stop() {
	close(sm.stopChannel)
}

// SecurityMiddleware provides comprehensive security features
type SecurityMiddleware struct {
	config  *config.Config
	monitor *SecurityMonitor
}

// NewSecurityMiddleware creates a new security middleware instance
func NewSecurityMiddleware(cfg *config.Config) *SecurityMiddleware {
	return &SecurityMiddleware{
		config:  cfg,
		monitor: NewSecurityMonitor(cfg),
	}
}

// Handler returns the HTTP middleware handler for security
func (sm *SecurityMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := sm.getClientIP(r)
		
		// Check if IP is blocked
		if sm.monitor.IsIPBlocked(clientIP) {
			sm.logSecurityEvent(SecurityEvent{
				EventType:   "blocked_ip_access",
				Severity:    "high",
				ClientIP:    clientIP,
				UserAgent:   r.UserAgent(),
				RequestPath: r.URL.Path,
				Method:      r.Method,
				Details:     "Request from blocked IP address",
				Blocked:     true,
			})
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		
		// Apply security headers
		sm.setSecurityHeaders(w)
		
		// Validate request size
		if err := sm.validateRequestSize(r); err != nil {
			sm.logSecurityEvent(SecurityEvent{
				EventType:   "request_size_violation",
				Severity:    "medium",
				ClientIP:    clientIP,
				UserAgent:   r.UserAgent(),
				RequestPath: r.URL.Path,
				Method:      r.Method,
				Details:     err.Error(),
				Blocked:     true,
			})
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		
		// Validate input
		if err := sm.validateRequest(r); err != nil {
			severity := "high"
			if strings.Contains(err.Error(), "SQL injection") || 
			   strings.Contains(err.Error(), "XSS") || 
			   strings.Contains(err.Error(), "command injection") {
				severity = "critical"
			}
			
			sm.logSecurityEvent(SecurityEvent{
				EventType:   "input_validation_failure",
				Severity:    severity,
				ClientIP:    clientIP,
				UserAgent:   r.UserAgent(),
				RequestPath: r.URL.Path,
				Method:      r.Method,
				Details:     err.Error(),
				Blocked:     true,
			})
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		
		// Apply CORS if configured
		if sm.handleCORS(w, r) {
			return
		}
		
		// Apply request timeout if configured
		if sm.config.Security.Validation.RequestTimeout > 0 {
			ctx, cancel := context.WithTimeout(r.Context(), sm.config.Security.Validation.RequestTimeout)
			defer cancel()
			r = r.WithContext(ctx)
		}
		
		next.ServeHTTP(w, r)
	})
}

// setSecurityHeaders applies comprehensive security headers
func (sm *SecurityMiddleware) setSecurityHeaders(w http.ResponseWriter) {
	if !sm.config.Security.Headers.Enabled {
		return
	}
	
	// Basic security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
	w.Header().Set("X-Download-Options", "noopen")
	
	// HSTS header
	if sm.config.Security.Headers.HSTSMaxAge > 0 {
		hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains", sm.config.Security.Headers.HSTSMaxAge)
		w.Header().Set("Strict-Transport-Security", hstsValue)
	}
	
	// Content Security Policy
	if sm.config.Security.Headers.CSPPolicy != "" {
		w.Header().Set("Content-Security-Policy", sm.config.Security.Headers.CSPPolicy)
	}
	
	// Cache control for sensitive endpoints
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// validateRequestSize checks if the request size is within limits
func (sm *SecurityMiddleware) validateRequestSize(r *http.Request) error {
	if sm.config.Security.Validation.MaxRequestSize == "" {
		return nil
	}
	
	maxSize, err := sm.parseSize(sm.config.Security.Validation.MaxRequestSize)
	if err != nil {
		return fmt.Errorf("invalid max request size configuration: %w", err)
	}
	
	if r.ContentLength > maxSize {
		return fmt.Errorf("request size %d exceeds maximum allowed size %d", r.ContentLength, maxSize)
	}
	
	return nil
}

// validateRequest performs comprehensive input validation
func (sm *SecurityMiddleware) validateRequest(r *http.Request) error {
	// Validate headers for suspicious content
	if err := sm.validateHeaders(r); err != nil {
		return fmt.Errorf("header validation failed: %w", err)
	}
	
	// Validate URL for malicious patterns
	if err := sm.validateURL(r); err != nil {
		return fmt.Errorf("URL validation failed: %w", err)
	}
	
	// Validate User-Agent for known bad patterns
	if err := sm.validateUserAgent(r); err != nil {
		return fmt.Errorf("User-Agent validation failed: %w", err)
	}
	
	return nil
}

// validateHeaders checks request headers for suspicious content
func (sm *SecurityMiddleware) validateHeaders(r *http.Request) error {
	suspiciousHeaders := []string{
		"X-Forwarded-Host",
		"X-Original-URL", 
		"X-Rewrite-URL",
		"X-Override-URL",
	}
	
	for _, header := range suspiciousHeaders {
		if value := r.Header.Get(header); value != "" {
			// Check for host header injection
			if strings.Contains(value, "\n") || strings.Contains(value, "\r") {
				return fmt.Errorf("header injection attempt detected in %s", header)
			}
		}
	}
	
	// Check for excessively long headers
	for name, values := range r.Header {
		for _, value := range values {
			if len(value) > 8192 { // 8KB limit per header value
				return fmt.Errorf("header %s exceeds maximum length", name)
			}
		}
	}
	
	return nil
}

// validateURL checks the request URL for malicious patterns
func (sm *SecurityMiddleware) validateURL(r *http.Request) error {
	url := r.URL.String()
	
	// Check for directory traversal attempts
	if strings.Contains(url, "..") || strings.Contains(url, "./") {
		return fmt.Errorf("directory traversal attempt detected")
	}
	
	// Check for SQL injection patterns in URL
	sqlPatterns := []string{
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
		"exec", "execute", "sp_", "xp_", "0x", "@@", "char(", "nchar(",
	}
	
	lowerURL := strings.ToLower(url)
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerURL, pattern) {
			return fmt.Errorf("potential SQL injection pattern detected")
		}
	}
	
	// Check for XSS patterns in URL
	xssPatterns := []string{
		"<script", "javascript:", "vbscript:", "onload=", "onerror=", 
		"onclick=", "onmouseover=", "alert(", "confirm(", "prompt(",
	}
	
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerURL, pattern) {
			return fmt.Errorf("potential XSS pattern detected")
		}
	}
	
	return nil
}

// validateUserAgent checks for known malicious user agents
func (sm *SecurityMiddleware) validateUserAgent(r *http.Request) error {
	userAgent := strings.ToLower(r.UserAgent())
	
	// Known bad user agent patterns
	badPatterns := []string{
		"sqlmap", "nmap", "nikto", "dirb", "dirbuster", "masscan", 
		"nessus", "openvas", "acunetix", "appscan", "burp", "zap",
		"wget", "curl", "python-requests", "go-http-client",
	}
	
	for _, pattern := range badPatterns {
		if strings.Contains(userAgent, pattern) {
			return fmt.Errorf("suspicious user agent detected: %s", pattern)
		}
	}
	
	// Check for empty user agent (common in automated attacks)
	if strings.TrimSpace(userAgent) == "" {
		return fmt.Errorf("empty user agent not allowed")
	}
	
	return nil
}

// handleCORS processes CORS requests with security considerations
func (sm *SecurityMiddleware) handleCORS(w http.ResponseWriter, r *http.Request) bool {
	if !sm.config.Security.CORS.Enabled {
		return false
	}
	
	origin := r.Header.Get("Origin")
	
	// If no origin header, continue with request
	if origin == "" {
		return false
	}
	
	// Check if origin is allowed
	if sm.isOriginAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		
		// Set allowed methods
		methods := sm.config.Security.CORS.AllowedMethods
		if len(methods) == 0 {
			methods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
		}
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		
		// Set allowed headers
		headers := sm.config.Security.CORS.AllowedHeaders
		if len(headers) == 0 {
			headers = []string{"Content-Type", "Authorization", "X-Requested-With"}
		}
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
		
		// Set exposed headers if configured
		if len(sm.config.Security.CORS.ExposedHeaders) > 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(sm.config.Security.CORS.ExposedHeaders, ", "))
		}
		
		// Set credentials
		if sm.config.Security.CORS.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		// Set max age
		maxAge := sm.config.Security.CORS.MaxAge
		if maxAge <= 0 {
			maxAge = 86400 // 24 hours default
		}
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
	}
	
	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	
	return false
}

// isOriginAllowed checks if the origin is in the allowed list
func (sm *SecurityMiddleware) isOriginAllowed(origin string) bool {
	allowedOrigins := sm.config.Security.CORS.AllowedOrigins
	
	// Fallback to default origins if none configured
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:5173", // Vite default
			"https://localhost:3000",
			"https://localhost:5173",
		}
	}
	
	for _, allowed := range allowedOrigins {
		// Exact match
		if origin == allowed {
			return true
		}
		
		// Wildcard support for subdomains
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	
	return false
}

// parseSize converts size strings like "10MB" to bytes
func (sm *SecurityMiddleware) parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))
	
	var multiplier int64 = 1
	var numberStr string
	
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		numberStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		numberStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		numberStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "B") {
		multiplier = 1
		numberStr = strings.TrimSuffix(sizeStr, "B")
	} else {
		// Assume bytes if no suffix
		numberStr = sizeStr
	}
	
	number, err := strconv.ParseInt(strings.TrimSpace(numberStr), 10, 64)
	if err != nil {
		return 0, err
	}
	
	return number * multiplier, nil
}

// logSecurityEvent logs a security event through the monitor
func (sm *SecurityMiddleware) logSecurityEvent(event SecurityEvent) {
	if sm.monitor != nil {
		sm.monitor.LogSecurityEvent(event)
	}
}

// getClientIP extracts the client IP address from the request
func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
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

// Stop stops the security middleware and its monitor
func (sm *SecurityMiddleware) Stop() {
	if sm.monitor != nil {
		sm.monitor.Stop()
	}
}

// InputValidationMiddleware provides comprehensive input validation and sanitization
type InputValidationMiddleware struct {
	config *config.Config
}

// NewInputValidationMiddleware creates a new input validation middleware
func NewInputValidationMiddleware(cfg *config.Config) *InputValidationMiddleware {
	return &InputValidationMiddleware{
		config: cfg,
	}
}

// Handler returns the HTTP middleware handler for input validation
func (ivm *InputValidationMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate and sanitize query parameters
		if err := ivm.validateQueryParameters(r); err != nil {
			http.Error(w, "Invalid query parameters", http.StatusBadRequest)
			return
		}
		
		// Validate form data if present
		if err := ivm.validateFormData(r); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		
		// Validate JSON body if present
		if err := ivm.validateJSONBody(r); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// validateQueryParameters validates and sanitizes URL query parameters
func (ivm *InputValidationMiddleware) validateQueryParameters(r *http.Request) error {
	query := r.URL.Query()
	
	for key, values := range query {
		// Validate parameter name
		if err := ivm.validateParameterName(key); err != nil {
			return fmt.Errorf("invalid parameter name '%s': %w", key, err)
		}
		
		// Validate and sanitize parameter values
		for i, value := range values {
			sanitized, err := ivm.sanitizeInput(value)
			if err != nil {
				return fmt.Errorf("invalid parameter value for '%s': %w", key, err)
			}
			values[i] = sanitized
		}
	}
	
	return nil
}

// validateFormData validates form data
func (ivm *InputValidationMiddleware) validateFormData(r *http.Request) error {
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("failed to parse form data: %w", err)
		}
		
		for key, values := range r.PostForm {
			// Validate field name
			if err := ivm.validateParameterName(key); err != nil {
				return fmt.Errorf("invalid form field name '%s': %w", key, err)
			}
			
			// Validate and sanitize field values
			for i, value := range values {
				sanitized, err := ivm.sanitizeInput(value)
				if err != nil {
					return fmt.Errorf("invalid form field value for '%s': %w", key, err)
				}
				values[i] = sanitized
			}
		}
	}
	
	return nil
}

// validateJSONBody validates JSON request body
func (ivm *InputValidationMiddleware) validateJSONBody(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil
	}
	
	// Basic JSON structure validation (depth and size limits)
	if r.ContentLength > 0 {
		maxDepth := 10
		maxKeys := 100
		
		if err := ivm.validateJSONStructure(r, maxDepth, maxKeys); err != nil {
			return fmt.Errorf("invalid JSON structure: %w", err)
		}
	}
	
	return nil
}

// validateParameterName validates parameter/field names
func (ivm *InputValidationMiddleware) validateParameterName(name string) error {
	// Check length
	if len(name) > 100 {
		return fmt.Errorf("parameter name too long")
	}
	
	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"__", "eval", "exec", "system", "cmd", "shell",
		"../", "./", "\\", "script", "javascript",
	}
	
	lowerName := strings.ToLower(name)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerName, pattern) {
			return fmt.Errorf("suspicious pattern detected in parameter name")
		}
	}
	
	// Must be alphanumeric with limited special characters
	validName := regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("parameter name contains invalid characters")
	}
	
	return nil
}

// sanitizeInput sanitizes input values
func (ivm *InputValidationMiddleware) sanitizeInput(input string) (string, error) {
	// Check length
	if len(input) > 10000 { // 10KB limit for individual values
		return "", fmt.Errorf("input value too long")
	}
	
	// Remove null bytes
	sanitized := strings.ReplaceAll(input, "\x00", "")
	
	// Check for malicious patterns
	if err := ivm.detectMaliciousPatterns(sanitized); err != nil {
		return "", err
	}
	
	// Basic HTML escaping for safety
	sanitized = ivm.escapeHTML(sanitized)
	
	return sanitized, nil
}

// detectMaliciousPatterns detects common attack patterns
func (ivm *InputValidationMiddleware) detectMaliciousPatterns(input string) error {
	lowerInput := strings.ToLower(input)
	
	// SQL injection patterns
	sqlPatterns := []string{
		"' or '1'='1", "' or 1=1", "union select", "drop table",
		"insert into", "update set", "delete from", "create table",
		"alter table", "exec ", "execute ", "sp_", "xp_",
	}
	
	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("potential SQL injection detected")
		}
	}
	
	// XSS patterns
	xssPatterns := []string{
		"<script", "javascript:", "vbscript:", "onload=", "onerror=",
		"onclick=", "onmouseover=", "onmouseout=", "onfocus=", "onblur=",
		"alert(", "confirm(", "prompt(", "document.cookie", "window.location",
	}
	
	for _, pattern := range xssPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("potential XSS detected")
		}
	}
	
	// Command injection patterns
	cmdPatterns := []string{
		"; cat ", "; ls ", "; rm ", "; mv ", "; cp ", "; chmod ",
		"| cat ", "| ls ", "| rm ", "| mv ", "| cp ", "| chmod ",
		"&& cat ", "&& ls ", "&& rm ", "&& mv ", "&& cp ", "&& chmod ",
		"`cat ", "`ls ", "`rm ", "`mv ", "`cp ", "`chmod ",
		"$(cat ", "$(ls ", "$(rm ", "$(mv ", "$(cp ", "$(chmod ",
	}
	
	for _, pattern := range cmdPatterns {
		if strings.Contains(lowerInput, pattern) {
			return fmt.Errorf("potential command injection detected")
		}
	}
	
	// Path traversal patterns
	if strings.Contains(input, "../") || strings.Contains(input, "..\\") {
		return fmt.Errorf("potential path traversal detected")
	}
	
	return nil
}

// escapeHTML performs basic HTML escaping
func (ivm *InputValidationMiddleware) escapeHTML(input string) string {
	replacements := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#39;",
		"&":  "&amp;",
	}
	
	result := input
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	
	return result
}

// validateJSONStructure validates JSON structure limits
func (ivm *InputValidationMiddleware) validateJSONStructure(r *http.Request, maxDepth, maxKeys int) error {
	// This is a simplified implementation
	// In a production system, you'd want to use a proper JSON parser
	// that can enforce depth and key count limits during parsing
	
	body := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(body); err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	
	// Count braces to estimate depth
	depth := 0
	maxDepthFound := 0
	keyCount := 0
	
	for _, char := range string(body) {
		switch char {
		case '{', '[':
			depth++
			if depth > maxDepthFound {
				maxDepthFound = depth
			}
		case '}', ']':
			depth--
		case ':':
			keyCount++
		}
	}
	
	if maxDepthFound > maxDepth {
		return fmt.Errorf("JSON depth %d exceeds maximum allowed depth %d", maxDepthFound, maxDepth)
	}
	
	if keyCount > maxKeys {
		return fmt.Errorf("JSON key count %d exceeds maximum allowed keys %d", keyCount, maxKeys)
	}
	
	return nil
}

// TimeoutMiddleware creates a middleware that enforces request timeouts
func (sm *SecurityMiddleware) TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if timeout <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			r = r.WithContext(ctx)
			
			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()
			
			select {
			case <-done:
				// Request completed normally
				return
			case <-ctx.Done():
				// Request timed out
				if ctx.Err() == context.DeadlineExceeded {
					http.Error(w, "Request timeout", http.StatusRequestTimeout)
				}
				return
			}
		})
	}
}

// BodySizeLimitMiddleware creates a middleware that enforces request body size limits
func (sm *SecurityMiddleware) BodySizeLimitMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxSize <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			
			// Check Content-Length header first
			if r.ContentLength > maxSize {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			
			// Wrap the body with a limited reader
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}
			
			next.ServeHTTP(w, r)
		})
	}
}