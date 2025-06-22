# API Rate Limiting & Security - Refined Implementation Cycles

## Overview
Break down API security into 5 manageable cycles, from basic rate limiting to advanced DDoS protection.

---

## **Cycle 6A: Basic Rate Limiting Implementation**
**Duration:** 4-5 hours | **Priority:** Critical

### Prerequisites
- Basic Go middleware knowledge
- Understanding of rate limiting concepts

### Implementation Tasks
- [ ] Install rate limiting dependencies (`golang.org/x/time/rate`)
- [ ] Create `backend/middleware/ratelimit.go`
- [ ] Implement token bucket rate limiter
- [ ] Add global rate limiting middleware
- [ ] Create rate limit response headers

### Code Deliverables
```go
// backend/middleware/ratelimit.go
type RateLimiter struct {
    limiter *rate.Limiter
    burst   int
    rate    rate.Limit
}

type RateLimitMiddleware struct {
    globalLimiter *RateLimiter
    config        *RateLimitConfig
}

func (rlm *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !rlm.globalLimiter.limiter.Allow() {
            rlm.handleRateLimit(w, r)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func (rlm *RateLimitMiddleware) handleRateLimit(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("X-RateLimit-Limit", "100")
    w.Header().Set("X-RateLimit-Remaining", "0")
    w.Header().Set("Retry-After", "60")
    http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
}
```

### Testing Requirements
- [ ] Unit tests for rate limiter logic
- [ ] Test rate limit enforcement
- [ ] Test rate limit headers
- [ ] Load test with high request volume

### Acceptance Criteria
- [ ] Global rate limiting works correctly
- [ ] Rate limit headers are included in responses
- [ ] 429 status code returned when limit exceeded
- [ ] Rate limiter allows requests within limits
- [ ] Performance impact is minimal (<5ms overhead)

### Risk Mitigation
- Test with realistic traffic patterns
- Monitor performance impact
- Start with conservative limits

---

## **Cycle 6B: Per-IP & Per-User Rate Limiting**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 6A completed and tested
- User authentication system available

### Implementation Tasks
- [ ] Implement per-IP rate limiting
- [ ] Add per-user rate limiting
- [ ] Create sliding window rate limiter
- [ ] Add configurable rate limits per endpoint
- [ ] Implement rate limit storage (in-memory + Redis optional)

### Code Deliverables
```go
// Enhanced rate limiting with multiple strategies
type MultiTierRateLimiter struct {
    ipLimiters   map[string]*rate.Limiter
    userLimiters map[int]*rate.Limiter
    config       *RateLimitConfig
    storage      RateLimitStorage
    mutex        sync.RWMutex
}

type RateLimitConfig struct {
    Global    RateLimitRule `yaml:"global"`
    PerIP     RateLimitRule `yaml:"per_ip"`
    PerUser   RateLimitRule `yaml:"per_user"`
    Endpoints map[string]RateLimitRule `yaml:"endpoints"`
}

type RateLimitRule struct {
    RequestsPerMinute int           `yaml:"requests_per_minute"`
    Burst            int           `yaml:"burst"`
    WindowSize       time.Duration `yaml:"window_size"`
}

func (mtrl *MultiTierRateLimiter) CheckRateLimit(r *http.Request, userID int) error {
    // Check global, IP, and user rate limits
    if err := mtrl.checkGlobalLimit(); err != nil {
        return err
    }
    if err := mtrl.checkIPLimit(getClientIP(r)); err != nil {
        return err
    }
    if userID > 0 {
        if err := mtrl.checkUserLimit(userID); err != nil {
            return err
        }
    }
    return nil
}
```

### Testing Requirements
- [ ] Unit tests for multi-tier rate limiting
- [ ] Test IP-based limiting
- [ ] Test user-based limiting
- [ ] Test endpoint-specific limits

### Acceptance Criteria
- [ ] Per-IP rate limiting works independently
- [ ] Per-user rate limiting enforces user quotas
- [ ] Different endpoints can have different limits
- [ ] Sliding window prevents burst abuse
- [ ] Memory usage stays bounded

### Risk Mitigation
- Monitor memory usage for limiters
- Clean up unused limiters regularly
- Test with realistic user patterns

---

## **Cycle 6C: Security Headers & Input Validation**
**Duration:** 4-5 hours | **Priority:** High

### Prerequisites
- Cycle 6B completed and tested
- Basic security knowledge

### Implementation Tasks
- [ ] Implement comprehensive security headers
- [ ] Add input validation middleware
- [ ] Create request size limiting
- [ ] Add request timeout enforcement
- [ ] Implement CORS security

### Code Deliverables
```go
// backend/middleware/security.go
type SecurityMiddleware struct {
    config *SecurityConfig
}

type SecurityConfig struct {
    MaxRequestSize    int64         `yaml:"max_request_size"`
    RequestTimeout    time.Duration `yaml:"request_timeout"`
    SecurityHeaders   map[string]string `yaml:"security_headers"`
    AllowedOrigins    []string      `yaml:"allowed_origins"`
    ValidationRules   ValidationRules `yaml:"validation"`
}

func (sm *SecurityMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Apply security headers
        sm.setSecurityHeaders(w)
        
        // Validate request size
        if r.ContentLength > sm.config.MaxRequestSize {
            http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
            return
        }
        
        // Validate input
        if err := sm.validateRequest(r); err != nil {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func (sm *SecurityMiddleware) setSecurityHeaders(w http.ResponseWriter) {
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    w.Header().Set("Content-Security-Policy", "default-src 'self'")
    w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}
```

### Testing Requirements
- [ ] Unit tests for security headers
- [ ] Test input validation rules
- [ ] Test request size limits
- [ ] Security scan with tools like OWASP ZAP

### Acceptance Criteria
- [ ] All security headers are properly set
- [ ] Request size limiting prevents large uploads
- [ ] Input validation catches malicious input
- [ ] CORS is properly configured
- [ ] Security scan shows no major vulnerabilities

### Risk Mitigation
- Test with security scanning tools
- Validate against OWASP security guidelines
- Monitor for security events

---

## **Cycle 6D: IP-Based Protection & Geoblocking**
**Duration:** 5-6 hours | **Priority:** Medium

### Prerequisites
- Cycle 6C completed and tested
- Understanding of IP geolocation

### Implementation Tasks
- [ ] Implement IP allowlist/blocklist functionality
- [ ] Add geolocation-based access control
- [ ] Create suspicious IP detection
- [ ] Implement automatic IP blocking for abuse
- [ ] Add IP reputation checking

### Code Deliverables
```go
// backend/middleware/ip_protection.go
type IPProtectionMiddleware struct {
    allowlist     map[string]bool
    blocklist     map[string]bool
    geolocator    GeoLocator
    reputation    IPReputationService
    config        *IPProtectionConfig
    suspiciousIPs map[string]*SuspiciousActivity
    mutex         sync.RWMutex
}

type SuspiciousActivity struct {
    IP            string
    ViolationCount int
    LastViolation time.Time
    Reason        string
}

type IPProtectionConfig struct {
    EnableGeoblocking   bool     `yaml:"enable_geoblocking"`
    BlockedCountries   []string `yaml:"blocked_countries"`
    AllowedCountries   []string `yaml:"allowed_countries"`
    SuspiciousThreshold int      `yaml:"suspicious_threshold"`
    AutoBlockDuration  time.Duration `yaml:"auto_block_duration"`
}

func (ipm *IPProtectionMiddleware) checkIPAccess(ip string) error {
    // Check allowlist/blocklist
    if ipm.isBlocked(ip) {
        return errors.New("IP blocked")
    }
    
    // Check geolocation
    if ipm.config.EnableGeoblocking {
        if err := ipm.checkGeolocation(ip); err != nil {
            return err
        }
    }
    
    // Check reputation
    if err := ipm.checkReputation(ip); err != nil {
        return err
    }
    
    return nil
}
```

### Testing Requirements
- [ ] Unit tests for IP protection logic
- [ ] Test geolocation blocking
- [ ] Test IP reputation checking
- [ ] Test automatic blocking mechanism

### Acceptance Criteria
- [ ] IP blocklist prevents access
- [ ] Geoblocking works for specified countries
- [ ] Suspicious IP detection functions correctly
- [ ] Automatic blocking activates for abuse
- [ ] IP reputation service integration works

### Risk Mitigation
- Test geolocation accuracy
- Have manual override for false positives
- Monitor IP blocking effectiveness

---

## **Cycle 6E: DDoS Protection & Advanced Security**
**Duration:** 6-7 hours | **Priority:** Medium

### Prerequisites
- Cycles 6A-6D completed and tested
- Understanding of DDoS attack patterns

### Implementation Tasks
- [ ] Implement request spike detection
- [ ] Add automatic throttling during high load
- [ ] Create circuit breaker pattern
- [ ] Implement graceful degradation
- [ ] Add security event monitoring and alerting

### Code Deliverables
```go
// backend/middleware/ddos_protection.go
type DDoSProtection struct {
    requestCounter  *RequestCounter
    circuitBreaker  *CircuitBreaker
    alertManager    *AlertManager
    config          *DDoSConfig
}

type DDoSConfig struct {
    SpikeThreshold      int           `yaml:"spike_threshold"`
    SpikeWindow         time.Duration `yaml:"spike_window"`
    CircuitBreakerThreshold int       `yaml:"circuit_breaker_threshold"`
    DegradationMode     bool          `yaml:"degradation_mode"`
}

type RequestCounter struct {
    requests    []time.Time
    windowSize  time.Duration
    mutex       sync.RWMutex
}

func (ddos *DDoSProtection) detectSpike() bool {
    recentRequests := ddos.requestCounter.getRecentRequests()
    if len(recentRequests) > ddos.config.SpikeThreshold {
        ddos.alertManager.SendAlert("DDoS spike detected", AlertLevelHigh)
        return true
    }
    return false
}

func (ddos *DDoSProtection) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ddos.requestCounter.recordRequest()
        
        if ddos.detectSpike() {
            ddos.handleDDoSResponse(w, r)
            return
        }
        
        if ddos.circuitBreaker.IsOpen() {
            ddos.handleCircuitOpen(w, r)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Testing Requirements
- [ ] Unit tests for spike detection
- [ ] Load testing to trigger DDoS protection
- [ ] Test circuit breaker functionality
- [ ] Test graceful degradation

### Acceptance Criteria
- [ ] Spike detection activates under load
- [ ] Circuit breaker prevents overload
- [ ] Graceful degradation maintains service
- [ ] Security alerts are sent for attacks
- [ ] System recovers after attack ends

### Risk Mitigation
- Test with realistic attack simulations
- Ensure legitimate traffic isn't blocked
- Monitor system performance under protection

---

## **Integration Testing**
**Duration:** 3-4 hours

### Comprehensive Security Testing
- [ ] End-to-end security flow testing
- [ ] Load testing with rate limiting
- [ ] DDoS simulation testing
- [ ] Security penetration testing
- [ ] Performance impact measurement

### Success Metrics
- [ ] Rate limiting handles 10,000+ requests/second
- [ ] DDoS protection activates within 10 seconds
- [ ] Security headers pass security scanners
- [ ] System maintains <1% false positive rate
- [ ] Performance overhead <10ms per request

---

## **Security Configuration**
```yaml
security:
  rate_limiting:
    global:
      requests_per_second: 100
      burst: 200
    per_ip:
      requests_per_minute: 60
      burst: 10
    endpoints:
      "/chat":
        requests_per_minute: 30
        
  headers:
    hsts_max_age: 31536000
    csp_policy: "default-src 'self'"
    
  ip_protection:
    enable_geoblocking: false
    suspicious_threshold: 100
    auto_block_duration: "15m"
    
  ddos_protection:
    spike_threshold: 1000
    spike_window: "1m"
    circuit_breaker_threshold: 50
```

---

## **Rollback Plan**
If any cycle fails:
1. Disable specific security features via config
2. Fall back to basic rate limiting
3. Remove problematic middleware temporarily
4. Use feature flags for gradual rollout