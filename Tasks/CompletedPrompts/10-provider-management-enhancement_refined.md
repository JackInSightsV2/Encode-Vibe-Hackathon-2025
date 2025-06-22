# Provider Management Enhancement - Refined Implementation Cycles

## Overview
Break down provider management into 5 manageable cycles, from basic architecture to advanced routing.

---

## **Cycle 10A: Enhanced Provider Architecture**
**Duration:** 5-6 hours | **Priority:** Critical

### Prerequisites
- Basic Go interfaces knowledge
- Understanding of existing provider system

### Implementation Tasks
- [ ] Redesign provider system with plugin architecture
- [ ] Create common provider interface
- [ ] Implement provider registry with dynamic loading
- [ ] Add provider configuration validation
- [ ] Create provider lifecycle management

### Code Deliverables
```go
// backend/providers/interface.go
type Provider interface {
    // Basic info
    Name() string
    Type() ProviderType
    Version() string
    
    // Health and status
    HealthCheck(ctx context.Context) error
    GetStatus() ProviderStatus
    GetMetrics() ProviderMetrics
    
    // Core functionality
    SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    GetModels() []Model
    GetCapabilities() Capabilities
    
    // Configuration
    Configure(config ProviderConfig) error
    Validate() error
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type ProviderType string

const (
    ProviderTypeOpenAI     ProviderType = "openai"
    ProviderTypeAnthropic  ProviderType = "anthropic"
    ProviderTypeLocal      ProviderType = "local"
    ProviderTypeMock       ProviderType = "mock"
)

type ProviderStatus struct {
    State       ProviderState `json:"state"`
    Healthy     bool          `json:"healthy"`
    LastCheck   time.Time     `json:"last_check"`
    Error       string        `json:"error,omitempty"`
    Uptime      time.Duration `json:"uptime"`
    Version     string        `json:"version"`
}

type ProviderState string

const (
    StateStarting ProviderState = "starting"
    StateRunning  ProviderState = "running"
    StateStopping ProviderState = "stopping"
    StateStopped  ProviderState = "stopped"
    StateError    ProviderState = "error"
)

type ProviderMetrics struct {
    RequestCount    int64         `json:"request_count"`
    ErrorCount      int64         `json:"error_count"`
    AverageLatency  time.Duration `json:"average_latency"`
    TokensUsed      int64         `json:"tokens_used"`
    Cost            float64       `json:"cost"`
    RateLimitHits   int64         `json:"rate_limit_hits"`
}

// backend/providers/registry.go
type ProviderRegistry struct {
    providers map[string]Provider
    configs   map[string]ProviderConfig
    mutex     sync.RWMutex
}

func NewProviderRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        providers: make(map[string]Provider),
        configs:   make(map[string]ProviderConfig),
    }
}

func (pr *ProviderRegistry) Register(provider Provider) error {
    pr.mutex.Lock()
    defer pr.mutex.Unlock()
    
    name := provider.Name()
    if _, exists := pr.providers[name]; exists {
        return fmt.Errorf("provider %s already registered", name)
    }
    
    if err := provider.Validate(); err != nil {
        return fmt.Errorf("provider validation failed: %w", err)
    }
    
    pr.providers[name] = provider
    return nil
}

func (pr *ProviderRegistry) Get(name string) (Provider, error) {
    pr.mutex.RLock()
    defer pr.mutex.RUnlock()
    
    provider, exists := pr.providers[name]
    if !exists {
        return nil, fmt.Errorf("provider %s not found", name)
    }
    
    return provider, nil
}

func (pr *ProviderRegistry) List() []Provider {
    pr.mutex.RLock()
    defer pr.mutex.RUnlock()
    
    providers := make([]Provider, 0, len(pr.providers))
    for _, provider := range pr.providers {
        providers = append(providers, provider)
    }
    
    return providers
}

func (pr *ProviderRegistry) Configure(name string, config ProviderConfig) error {
    provider, err := pr.Get(name)
    if err != nil {
        return err
    }
    
    if err := provider.Configure(config); err != nil {
        return err
    }
    
    pr.mutex.Lock()
    pr.configs[name] = config
    pr.mutex.Unlock()
    
    return nil
}
```

### Testing Requirements
- [ ] Unit tests for provider interface
- [ ] Test provider registration and lifecycle
- [ ] Test configuration validation
- [ ] Integration tests with mock providers

### Acceptance Criteria
- [ ] Provider interface is well-defined and extensible
- [ ] Registry manages providers correctly
- [ ] Configuration validation prevents invalid setups
- [ ] Provider lifecycle is properly managed
- [ ] All existing providers work with new architecture

### Risk Mitigation
- Maintain backward compatibility with existing providers
- Test thoroughly with all provider types
- Implement graceful error handling

---

## **Cycle 10B: Health Monitoring System**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 10A completed and tested
- Understanding of health check patterns

### Implementation Tasks
- [ ] Create comprehensive health monitoring system
- [ ] Implement health check strategies
- [ ] Add health score calculation algorithm
- [ ] Create health-based routing decisions
- [ ] Add health monitoring dashboard

### Code Deliverables
```go
// backend/providers/health.go
type HealthMonitor struct {
    registry    *ProviderRegistry
    checkers    map[string]*ProviderChecker
    config      *HealthConfig
    scheduler   *HealthScheduler
    metrics     *HealthMetrics
    mutex       sync.RWMutex
}

type HealthConfig struct {
    CheckInterval    time.Duration `yaml:"check_interval"`
    Timeout          time.Duration `yaml:"timeout"`
    FailureThreshold int           `yaml:"failure_threshold"`
    SuccessThreshold int           `yaml:"success_threshold"`
    Strategies       []HealthStrategy `yaml:"strategies"`
}

type HealthStrategy struct {
    Type    string            `yaml:"type"`
    Weight  float64          `yaml:"weight"`
    Config  map[string]interface{} `yaml:"config"`
}

type ProviderChecker struct {
    provider         Provider
    consecutiveFailures int
    consecutiveSuccesses int
    lastCheck        time.Time
    healthScore      float64
    checks           []HealthCheckResult
    maxHistorySize   int
}

type HealthCheckResult struct {
    Timestamp time.Time     `json:"timestamp"`
    Success   bool          `json:"success"`
    Latency   time.Duration `json:"latency"`
    Error     string        `json:"error,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
}

func NewHealthMonitor(registry *ProviderRegistry, config *HealthConfig) *HealthMonitor {
    hm := &HealthMonitor{
        registry: registry,
        checkers: make(map[string]*ProviderChecker),
        config:   config,
        metrics:  NewHealthMetrics(),
    }
    
    hm.scheduler = NewHealthScheduler(hm, config.CheckInterval)
    return hm
}

func (hm *HealthMonitor) Start(ctx context.Context) error {
    // Initialize checkers for all registered providers
    for _, provider := range hm.registry.List() {
        checker := &ProviderChecker{
            provider:       provider,
            healthScore:    1.0, // Start with perfect health
            checks:         make([]HealthCheckResult, 0),
            maxHistorySize: 100,
        }
        hm.checkers[provider.Name()] = checker
    }
    
    // Start the scheduler
    return hm.scheduler.Start(ctx)
}

func (hm *HealthMonitor) CheckProvider(ctx context.Context, providerName string) *HealthCheckResult {
    provider, err := hm.registry.Get(providerName)
    if err != nil {
        return &HealthCheckResult{
            Timestamp: time.Now(),
            Success:   false,
            Error:     err.Error(),
        }
    }
    
    start := time.Now()
    
    // Perform health check with timeout
    checkCtx, cancel := context.WithTimeout(ctx, hm.config.Timeout)
    defer cancel()
    
    err = provider.HealthCheck(checkCtx)
    latency := time.Since(start)
    
    result := &HealthCheckResult{
        Timestamp: time.Now(),
        Success:   err == nil,
        Latency:   latency,
    }
    
    if err != nil {
        result.Error = err.Error()
    }
    
    // Add additional health check strategies
    hm.runHealthStrategies(checkCtx, provider, result)
    
    // Update checker state
    hm.updateCheckerState(providerName, result)
    
    return result
}

func (hm *HealthMonitor) runHealthStrategies(ctx context.Context, provider Provider, result *HealthCheckResult) {
    for _, strategy := range hm.config.Strategies {
        switch strategy.Type {
        case "response_time":
            hm.checkResponseTime(ctx, provider, result, strategy)
        case "error_rate":
            hm.checkErrorRate(ctx, provider, result, strategy)
        case "rate_limit":
            hm.checkRateLimit(ctx, provider, result, strategy)
        case "model_availability":
            hm.checkModelAvailability(ctx, provider, result, strategy)
        }
    }
}

func (hm *HealthMonitor) checkResponseTime(ctx context.Context, provider Provider, result *HealthCheckResult, strategy HealthStrategy) {
    threshold := time.Duration(strategy.Config["threshold_ms"].(float64)) * time.Millisecond
    
    if result.Latency > threshold {
        result.Success = false
        if result.Error == "" {
            result.Error = fmt.Sprintf("Response time %v exceeds threshold %v", result.Latency, threshold)
        }
    }
    
    if result.Details == nil {
        result.Details = make(map[string]interface{})
    }
    result.Details["response_time_check"] = map[string]interface{}{
        "latency":   result.Latency.Milliseconds(),
        "threshold": threshold.Milliseconds(),
        "passed":    result.Latency <= threshold,
    }
}

func (hm *HealthMonitor) updateCheckerState(providerName string, result *HealthCheckResult) {
    hm.mutex.Lock()
    defer hm.mutex.Unlock()
    
    checker, exists := hm.checkers[providerName]
    if !exists {
        return
    }
    
    checker.lastCheck = result.Timestamp
    
    // Update success/failure counters
    if result.Success {
        checker.consecutiveFailures = 0
        checker.consecutiveSuccesses++
    } else {
        checker.consecutiveSuccesses = 0
        checker.consecutiveFailures++
    }
    
    // Add to history
    checker.checks = append(checker.checks, *result)
    if len(checker.checks) > checker.maxHistorySize {
        checker.checks = checker.checks[1:]
    }
    
    // Calculate health score
    checker.healthScore = hm.calculateHealthScore(checker)
    
    // Update metrics
    hm.metrics.RecordCheck(providerName, result)
}

func (hm *HealthMonitor) calculateHealthScore(checker *ProviderChecker) float64 {
    if len(checker.checks) == 0 {
        return 1.0
    }
    
    // Base score on recent success rate
    recentChecks := 10
    if len(checker.checks) < recentChecks {
        recentChecks = len(checker.checks)
    }
    
    recentResults := checker.checks[len(checker.checks)-recentChecks:]
    successCount := 0
    totalLatency := time.Duration(0)
    
    for _, check := range recentResults {
        if check.Success {
            successCount++
        }
        totalLatency += check.Latency
    }
    
    successRate := float64(successCount) / float64(len(recentResults))
    avgLatency := totalLatency / time.Duration(len(recentResults))
    
    // Score based on success rate (0.7 weight) and latency (0.3 weight)
    latencyScore := math.Max(0, 1.0-float64(avgLatency.Milliseconds())/1000.0)
    
    return (successRate * 0.7) + (latencyScore * 0.3)
}

func (hm *HealthMonitor) GetProviderHealth(providerName string) (*ProviderHealth, error) {
    hm.mutex.RLock()
    defer hm.mutex.RUnlock()
    
    checker, exists := hm.checkers[providerName]
    if !exists {
        return nil, fmt.Errorf("provider %s not found", providerName)
    }
    
    return &ProviderHealth{
        Name:                 providerName,
        HealthScore:          checker.healthScore,
        ConsecutiveFailures:  checker.consecutiveFailures,
        ConsecutiveSuccesses: checker.consecutiveSuccesses,
        LastCheck:           checker.lastCheck,
        RecentChecks:        checker.checks[max(0, len(checker.checks)-10):],
    }, nil
}
```

### Testing Requirements
- [ ] Unit tests for health monitoring logic
- [ ] Test health score calculation accuracy
- [ ] Test health check strategies
- [ ] Integration tests with real providers

### Acceptance Criteria
- [ ] Health monitoring accurately reflects provider status
- [ ] Health scores correlate with actual performance
- [ ] Health check strategies work correctly
- [ ] Monitoring has minimal performance impact
- [ ] Health data is available via API

### Risk Mitigation
- Test health monitoring with various failure scenarios
- Ensure health checks don't overwhelm providers
- Validate health score algorithm accuracy

---

## **Cycle 10C: Intelligent Routing Engine**
**Duration:** 6-7 hours | **Priority:** High

### Prerequisites
- Cycle 10B completed and tested
- Understanding of load balancing algorithms

### Implementation Tasks
- [ ] Enhance routing system with smart algorithms
- [ ] Implement load-based routing
- [ ] Add latency-based routing
- [ ] Create cost-optimized routing
- [ ] Add routing policy configuration

### Code Deliverables
```go
// backend/providers/routing.go
type RoutingEngine struct {
    registry     *ProviderRegistry
    healthMonitor *HealthMonitor
    config       *RoutingConfig
    algorithms   map[string]RoutingAlgorithm
    metrics      *RoutingMetrics
}

type RoutingConfig struct {
    Strategy    string                 `yaml:"strategy"`
    Factors     map[string]float64     `yaml:"factors"`
    Preferences map[string]interface{} `yaml:"preferences"`
    Fallback    []string              `yaml:"fallback"`
    Rules       []RoutingRule         `yaml:"rules"`
}

type RoutingRule struct {
    Condition string   `yaml:"condition"`
    Providers []string `yaml:"providers"`
    Weight    float64  `yaml:"weight"`
}

type RoutingAlgorithm interface {
    Name() string
    SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error)
    Configure(config map[string]interface{}) error
}

type RoutingRequest struct {
    Model        string            `json:"model"`
    UserID       string            `json:"user_id"`
    SessionID    string            `json:"session_id"`
    MessageType  string            `json:"message_type"`
    Priority     RoutingPriority   `json:"priority"`
    Constraints  RoutingConstraints `json:"constraints"`
    Metadata     map[string]interface{} `json:"metadata"`
}

type RoutingPriority string

const (
    PriorityLow    RoutingPriority = "low"
    PriorityNormal RoutingPriority = "normal"
    PriorityHigh   RoutingPriority = "high"
)

type RoutingConstraints struct {
    MaxLatency   time.Duration `json:"max_latency"`
    MaxCost      float64       `json:"max_cost"`
    RequiredTags []string      `json:"required_tags"`
    ExcludeProvider []string   `json:"exclude_provider"`
}

type RoutingResult struct {
    Provider    string        `json:"provider"`
    Score       float64       `json:"score"`
    Reason      string        `json:"reason"`
    Alternatives []Alternative `json:"alternatives"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type Alternative struct {
    Provider string  `json:"provider"`
    Score    float64 `json:"score"`
    Reason   string  `json:"reason"`
}

func NewRoutingEngine(registry *ProviderRegistry, healthMonitor *HealthMonitor, config *RoutingConfig) *RoutingEngine {
    re := &RoutingEngine{
        registry:      registry,
        healthMonitor: healthMonitor,
        config:        config,
        algorithms:    make(map[string]RoutingAlgorithm),
        metrics:       NewRoutingMetrics(),
    }
    
    // Register built-in algorithms
    re.algorithms["round_robin"] = NewRoundRobinAlgorithm()
    re.algorithms["weighted_random"] = NewWeightedRandomAlgorithm()
    re.algorithms["health_based"] = NewHealthBasedAlgorithm(healthMonitor)
    re.algorithms["latency_based"] = NewLatencyBasedAlgorithm(healthMonitor)
    re.algorithms["cost_optimized"] = NewCostOptimizedAlgorithm()
    re.algorithms["intelligent"] = NewIntelligentAlgorithm(healthMonitor, re.metrics)
    
    return re
}

func (re *RoutingEngine) Route(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
    start := time.Now()
    defer func() {
        re.metrics.RecordRoutingLatency(time.Since(start))
    }()
    
    // Apply routing rules first
    filteredProviders, err := re.applyRoutingRules(req)
    if err != nil {
        return nil, err
    }
    
    // Get routing algorithm
    algorithm, exists := re.algorithms[re.config.Strategy]
    if !exists {
        return nil, fmt.Errorf("routing algorithm %s not found", re.config.Strategy)
    }
    
    // Create modified request with filtered providers
    routingReq := *req
    routingReq.Metadata["filtered_providers"] = filteredProviders
    
    // Select provider
    result, err := algorithm.SelectProvider(ctx, &routingReq)
    if err != nil {
        // Try fallback chain
        return re.tryFallback(ctx, req)
    }
    
    // Record metrics
    re.metrics.RecordRouting(req, result)
    
    return result, nil
}

// Intelligent Algorithm Implementation
type IntelligentAlgorithm struct {
    healthMonitor *HealthMonitor
    metrics       *RoutingMetrics
    weights       map[string]float64
}

func NewIntelligentAlgorithm(healthMonitor *HealthMonitor, metrics *RoutingMetrics) *IntelligentAlgorithm {
    return &IntelligentAlgorithm{
        healthMonitor: healthMonitor,
        metrics:       metrics,
        weights: map[string]float64{
            "health":      0.4,
            "latency":     0.3,
            "cost":        0.2,
            "availability": 0.1,
        },
    }
}

func (ia *IntelligentAlgorithm) Name() string {
    return "intelligent"
}

func (ia *IntelligentAlgorithm) SelectProvider(ctx context.Context, req *RoutingRequest) (*RoutingResult, error) {
    providers := req.Metadata["filtered_providers"].([]string)
    if len(providers) == 0 {
        return nil, fmt.Errorf("no providers available")
    }
    
    scores := make(map[string]float64)
    alternatives := make([]Alternative, 0)
    
    for _, providerName := range providers {
        score := ia.calculateProviderScore(ctx, providerName, req)
        scores[providerName] = score
        
        alternatives = append(alternatives, Alternative{
            Provider: providerName,
            Score:    score,
            Reason:   ia.getScoreReason(providerName, score),
        })
    }
    
    // Sort alternatives by score
    sort.Slice(alternatives, func(i, j int) bool {
        return alternatives[i].Score > alternatives[j].Score
    })
    
    bestProvider := alternatives[0].Provider
    
    return &RoutingResult{
        Provider:     bestProvider,
        Score:        scores[bestProvider],
        Reason:       fmt.Sprintf("Best overall score: %.3f", scores[bestProvider]),
        Alternatives: alternatives[1:], // Exclude the selected provider
        Metadata: map[string]interface{}{
            "algorithm": "intelligent",
            "factors":   ia.weights,
        },
    }, nil
}

func (ia *IntelligentAlgorithm) calculateProviderScore(ctx context.Context, providerName string, req *RoutingRequest) float64 {
    var totalScore float64
    
    // Health score
    if health, err := ia.healthMonitor.GetProviderHealth(providerName); err == nil {
        totalScore += health.HealthScore * ia.weights["health"]
    }
    
    // Latency score
    avgLatency := ia.metrics.GetAverageLatency(providerName)
    latencyScore := math.Max(0, 1.0-float64(avgLatency.Milliseconds())/1000.0)
    totalScore += latencyScore * ia.weights["latency"]
    
    // Cost score (inverse - lower cost = higher score)
    avgCost := ia.metrics.GetAverageCost(providerName)
    costScore := math.Max(0, 1.0-avgCost/10.0) // Normalize assuming max cost of $10
    totalScore += costScore * ia.weights["cost"]
    
    // Availability score
    availability := ia.metrics.GetAvailability(providerName)
    totalScore += availability * ia.weights["availability"]
    
    // Apply request-specific adjustments
    totalScore = ia.applyRequestAdjustments(totalScore, providerName, req)
    
    return math.Max(0, math.Min(1, totalScore))
}

func (ia *IntelligentAlgorithm) applyRequestAdjustments(score float64, providerName string, req *RoutingRequest) float64 {
    // Priority adjustments
    switch req.Priority {
    case PriorityHigh:
        // Prefer providers with better health for high priority
        if health, err := ia.healthMonitor.GetProviderHealth(providerName); err == nil {
            score += health.HealthScore * 0.1
        }
    case PriorityLow:
        // For low priority, prefer cost-effective providers
        avgCost := ia.metrics.GetAverageCost(providerName)
        if avgCost < 1.0 {
            score += 0.1
        }
    }
    
    // Model-specific adjustments
    if req.Model != "" {
        modelSupport := ia.metrics.GetModelSupport(providerName, req.Model)
        score *= modelSupport
    }
    
    return score
}

// Circuit Breaker Integration
type ProviderCircuitBreaker struct {
    provider        string
    state          CircuitState
    failureCount   int
    lastFailure    time.Time
    config         *CircuitBreakerConfig
    mutex          sync.RWMutex
}

type CircuitState string

const (
    StateClosed   CircuitState = "closed"
    StateOpen     CircuitState = "open"
    StateHalfOpen CircuitState = "half_open"
)

type CircuitBreakerConfig struct {
    FailureThreshold int           `yaml:"failure_threshold"`
    RecoveryTimeout  time.Duration `yaml:"recovery_timeout"`
    HalfOpenMaxCalls int           `yaml:"half_open_max_calls"`
}

func (cb *ProviderCircuitBreaker) CanExecute() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Since(cb.lastFailure) > cb.config.RecoveryTimeout {
            cb.state = StateHalfOpen
            return true
        }
        return false
    case StateHalfOpen:
        return true
    }
    
    return false
}

func (cb *ProviderCircuitBreaker) RecordSuccess() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failureCount = 0
    cb.state = StateClosed
}

func (cb *ProviderCircuitBreaker) RecordFailure() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failureCount++
    cb.lastFailure = time.Now()
    
    if cb.failureCount >= cb.config.FailureThreshold {
        cb.state = StateOpen
    }
}
```

### Testing Requirements
- [ ] Unit tests for routing algorithms
- [ ] Test intelligent routing accuracy
- [ ] Test circuit breaker functionality
- [ ] Load testing with various routing strategies

### Acceptance Criteria
- [ ] Intelligent routing improves response time by 20%
- [ ] Circuit breakers prevent cascade failures
- [ ] Routing decisions are made within 10ms
- [ ] Fallback mechanisms work correctly
- [ ] Routing metrics are accurate

### Risk Mitigation
- Test routing under various load conditions
- Ensure fallback mechanisms are reliable
- Monitor routing performance impact

---

## **Cycle 10D: Provider Configuration Management**
**Duration:** 4-5 hours | **Priority:** Medium

### Prerequisites
- Cycle 10C completed and tested
- Configuration system understanding

### Implementation Tasks
- [ ] Implement hot-reload for provider configurations
- [ ] Add provider enable/disable functionality
- [ ] Create provider testing and validation tools
- [ ] Add provider versioning support
- [ ] Create provider dependency management

### Code Deliverables
```go
// backend/providers/config.go
type ProviderConfigManager struct {
    registry    *ProviderRegistry
    configs     map[string]*ProviderConfig
    watchers    []ConfigWatcher
    validator   *ConfigValidator
    mutex       sync.RWMutex
}

type ProviderConfig struct {
    Name        string                 `yaml:"name"`
    Type        ProviderType          `yaml:"type"`
    Enabled     bool                  `yaml:"enabled"`
    Version     string                `yaml:"version"`
    Config      map[string]interface{} `yaml:"config"`
    Models      []string              `yaml:"models"`
    Tags        []string              `yaml:"tags"`
    Priority    int                   `yaml:"priority"`
    Dependencies []string             `yaml:"dependencies"`
    
    // Health check configuration
    HealthCheck *ProviderHealthConfig `yaml:"health_check,omitempty"`
    
    // Rate limiting
    RateLimit   *ProviderRateLimit    `yaml:"rate_limit,omitempty"`
    
    // Cost configuration
    Pricing     *ProviderPricing      `yaml:"pricing,omitempty"`
}

type ProviderHealthConfig struct {
    Interval    time.Duration `yaml:"interval"`
    Timeout     time.Duration `yaml:"timeout"`
    MaxFailures int           `yaml:"max_failures"`
    CustomCheck string        `yaml:"custom_check,omitempty"`
}

type ProviderRateLimit struct {
    RequestsPerSecond int `yaml:"requests_per_second"`
    RequestsPerMinute int `yaml:"requests_per_minute"`
    RequestsPerHour   int `yaml:"requests_per_hour"`
    TokensPerMinute   int `yaml:"tokens_per_minute"`
}

type ProviderPricing struct {
    InputTokenCost  float64 `yaml:"input_token_cost"`
    OutputTokenCost float64 `yaml:"output_token_cost"`
    RequestCost     float64 `yaml:"request_cost"`
    Currency        string  `yaml:"currency"`
}

type ConfigWatcher interface {
    OnConfigChanged(providerName string, oldConfig, newConfig *ProviderConfig) error
}

func NewProviderConfigManager(registry *ProviderRegistry) *ProviderConfigManager {
    return &ProviderConfigManager{
        registry:  registry,
        configs:   make(map[string]*ProviderConfig),
        watchers:  make([]ConfigWatcher, 0),
        validator: NewConfigValidator(),
    }
}

func (pcm *ProviderConfigManager) LoadConfig(filePath string) error {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("failed to read config file: %w", err)
    }
    
    var configs map[string]*ProviderConfig
    if err := yaml.Unmarshal(data, &configs); err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }
    
    // Validate all configurations
    for name, config := range configs {
        if err := pcm.validator.Validate(config); err != nil {
            return fmt.Errorf("invalid config for provider %s: %w", name, err)
        }
    }
    
    // Apply configurations
    for name, config := range configs {
        if err := pcm.UpdateConfig(name, config); err != nil {
            return fmt.Errorf("failed to update config for %s: %w", name, err)
        }
    }
    
    return nil
}

func (pcm *ProviderConfigManager) UpdateConfig(providerName string, config *ProviderConfig) error {
    pcm.mutex.Lock()
    defer pcm.mutex.Unlock()
    
    oldConfig := pcm.configs[providerName]
    
    // Validate new configuration
    if err := pcm.validator.Validate(config); err != nil {
        return err
    }
    
    // Check dependencies
    if err := pcm.checkDependencies(config); err != nil {
        return err
    }
    
    // Apply configuration to provider
    provider, err := pcm.registry.Get(providerName)
    if err != nil {
        return err
    }
    
    if err := provider.Configure(*config); err != nil {
        return fmt.Errorf("provider rejected configuration: %w", err)
    }
    
    // Store new configuration
    pcm.configs[providerName] = config
    
    // Notify watchers
    for _, watcher := range pcm.watchers {
        if err := watcher.OnConfigChanged(providerName, oldConfig, config); err != nil {
            log.Printf("Config watcher error: %v", err)
        }
    }
    
    return nil
}

func (pcm *ProviderConfigManager) EnableProvider(providerName string) error {
    pcm.mutex.Lock()
    defer pcm.mutex.Unlock()
    
    config, exists := pcm.configs[providerName]
    if !exists {
        return fmt.Errorf("provider %s not found", providerName)
    }
    
    if config.Enabled {
        return nil // Already enabled
    }
    
    // Check dependencies
    if err := pcm.checkDependencies(config); err != nil {
        return fmt.Errorf("cannot enable provider: %w", err)
    }
    
    provider, err := pcm.registry.Get(providerName)
    if err != nil {
        return err
    }
    
    // Start the provider
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := provider.Start(ctx); err != nil {
        return fmt.Errorf("failed to start provider: %w", err)
    }
    
    config.Enabled = true
    return nil
}

func (pcm *ProviderConfigManager) DisableProvider(providerName string) error {
    pcm.mutex.Lock()
    defer pcm.mutex.Unlock()
    
    config, exists := pcm.configs[providerName]
    if !exists {
        return fmt.Errorf("provider %s not found", providerName)
    }
    
    if !config.Enabled {
        return nil // Already disabled
    }
    
    provider, err := pcm.registry.Get(providerName)
    if err != nil {
        return err
    }
    
    // Stop the provider
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := provider.Stop(ctx); err != nil {
        return fmt.Errorf("failed to stop provider: %w", err)
    }
    
    config.Enabled = false
    return nil
}

func (pcm *ProviderConfigManager) TestProvider(providerName string, testConfig *ProviderConfig) (*ProviderTestResult, error) {
    // Create a temporary provider instance for testing
    provider, err := pcm.createProviderInstance(testConfig)
    if err != nil {
        return nil, err
    }
    defer provider.Stop(context.Background())
    
    result := &ProviderTestResult{
        Provider:  providerName,
        StartTime: time.Now(),
    }
    
    // Test configuration validation
    if err := provider.Validate(); err != nil {
        result.ConfigValidation = &TestResult{
            Success: false,
            Error:   err.Error(),
        }
        return result, nil
    }
    result.ConfigValidation = &TestResult{Success: true}
    
    // Test startup
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := provider.Start(ctx); err != nil {
        result.Startup = &TestResult{
            Success: false,
            Error:   err.Error(),
        }
        return result, nil
    }
    result.Startup = &TestResult{Success: true}
    
    // Test health check
    if err := provider.HealthCheck(ctx); err != nil {
        result.HealthCheck = &TestResult{
            Success: false,
            Error:   err.Error(),
        }
    } else {
        result.HealthCheck = &TestResult{Success: true}
    }
    
    // Test basic functionality
    testReq := &ChatRequest{
        UserID:    "test",
        SessionID: "test",
        Message:   "Test message",
    }
    
    _, err = provider.SendRequest(ctx, testReq)
    if err != nil {
        result.Functionality = &TestResult{
            Success: false,
            Error:   err.Error(),
        }
    } else {
        result.Functionality = &TestResult{Success: true}
    }
    
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    
    return result, nil
}

type ProviderTestResult struct {
    Provider           string        `json:"provider"`
    StartTime          time.Time     `json:"start_time"`
    EndTime            time.Time     `json:"end_time"`
    Duration           time.Duration `json:"duration"`
    ConfigValidation   *TestResult   `json:"config_validation"`
    Startup            *TestResult   `json:"startup"`
    HealthCheck        *TestResult   `json:"health_check"`
    Functionality      *TestResult   `json:"functionality"`
}

type TestResult struct {
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}
```

### Testing Requirements
- [ ] Unit tests for configuration management
- [ ] Test hot-reload functionality
- [ ] Test provider testing tools
- [ ] Integration tests with real providers

### Acceptance Criteria
- [ ] Configuration hot-reload works without restart
- [ ] Provider enable/disable functions correctly
- [ ] Configuration validation prevents invalid setups
- [ ] Provider testing tools work accurately
- [ ] Dependency checking prevents conflicts

### Risk Mitigation
- Validate configurations thoroughly before applying
- Test hot-reload with various scenarios
- Ensure graceful handling of configuration errors

---

## **Cycle 10E: Provider Management UI**
**Duration:** 6-8 hours | **Priority:** Low

### Prerequisites
- Cycles 10A-10D completed and tested
- Frontend development environment ready

### Implementation Tasks
- [ ] Create provider management dashboard
- [ ] Add provider status visualization
- [ ] Implement provider configuration interface
- [ ] Create provider testing tools UI
- [ ] Add provider metrics and analytics

### Code Deliverables
```typescript
// frontend/src/components/providers/ProviderManagement.tsx
interface ProviderManagementProps {
    providers: Provider[];
    onConfigUpdate: (providerName: string, config: ProviderConfig) => void;
    onToggleProvider: (providerName: string, enabled: boolean) => void;
    onTestProvider: (providerName: string) => void;
}

const ProviderManagement: React.FC<ProviderManagementProps> = ({
    providers,
    onConfigUpdate,
    onToggleProvider,
    onTestProvider
}) => {
    const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
    const [testResults, setTestResults] = useState<Map<string, ProviderTestResult>>(new Map());
    
    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <h2 className="text-2xl font-bold text-gray-900">Provider Management</h2>
                <Button onClick={() => window.location.reload()}>
                    <RefreshIcon className="w-4 h-4 mr-2" />
                    Refresh
                </Button>
            </div>
            
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Provider List */}
                <div className="lg:col-span-1">
                    <Card title="Providers" className="h-full">
                        <div className="space-y-2">
                            {providers.map(provider => (
                                <ProviderCard
                                    key={provider.name}
                                    provider={provider}
                                    selected={selectedProvider === provider.name}
                                    onClick={() => setSelectedProvider(provider.name)}
                                    onToggle={(enabled) => onToggleProvider(provider.name, enabled)}
                                    onTest={() => onTestProvider(provider.name)}
                                />
                            ))}
                        </div>
                    </Card>
                </div>
                
                {/* Provider Details */}
                <div className="lg:col-span-2">
                    {selectedProvider ? (
                        <ProviderDetails
                            provider={providers.find(p => p.name === selectedProvider)!}
                            testResult={testResults.get(selectedProvider)}
                            onConfigUpdate={(config) => onConfigUpdate(selectedProvider, config)}
                        />
                    ) : (
                        <Card>
                            <div className="text-center py-12">
                                <ServerIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                                <p className="text-gray-500">Select a provider to view details</p>
                            </div>
                        </Card>
                    )}
                </div>
            </div>
        </div>
    );
};

// frontend/src/components/providers/ProviderCard.tsx
interface ProviderCardProps {
    provider: Provider;
    selected: boolean;
    onClick: () => void;
    onToggle: (enabled: boolean) => void;
    onTest: () => void;
}

const ProviderCard: React.FC<ProviderCardProps> = ({
    provider,
    selected,
    onClick,
    onToggle,
    onTest
}) => {
    const getHealthColor = (health: number) => {
        if (health >= 0.8) return 'text-green-600';
        if (health >= 0.6) return 'text-yellow-600';
        return 'text-red-600';
    };
    
    const getHealthIcon = (health: number) => {
        if (health >= 0.8) return <CheckCircleIcon className="w-5 h-5" />;
        if (health >= 0.6) return <ExclamationTriangleIcon className="w-5 h-5" />;
        return <XCircleIcon className="w-5 h-5" />;
    };
    
    return (
        <div
            className={cn(
                'p-4 border rounded-lg cursor-pointer transition-all',
                selected
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
            )}
            onClick={onClick}
        >
            <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                    <h3 className="font-medium text-gray-900">{provider.name}</h3>
                    <Badge variant={provider.enabled ? 'success' : 'secondary'}>
                        {provider.enabled ? 'Enabled' : 'Disabled'}
                    </Badge>
                </div>
                <div className="flex items-center space-x-1">
                    <Switch
                        checked={provider.enabled}
                        onChange={onToggle}
                        onClick={(e) => e.stopPropagation()}
                    />
                </div>
            </div>
            
            <div className="flex items-center justify-between text-sm">
                <div className={cn('flex items-center space-x-1', getHealthColor(provider.health))}>
                    {getHealthIcon(provider.health)}
                    <span>Health: {(provider.health * 100).toFixed(0)}%</span>
                </div>
                
                <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => {
                        e.stopPropagation();
                        onTest();
                    }}
                >
                    Test
                </Button>
            </div>
            
            <div className="mt-2 text-xs text-gray-500">
                <div>Type: {provider.type}</div>
                <div>Models: {provider.models.length}</div>
                <div>Latency: {provider.averageLatency}ms</div>
            </div>
        </div>
    );
};

// frontend/src/components/providers/ProviderDetails.tsx
const ProviderDetails: React.FC<ProviderDetailsProps> = ({
    provider,
    testResult,
    onConfigUpdate
}) => {
    const [activeTab, setActiveTab] = useState<'overview' | 'config' | 'metrics' | 'test'>('overview');
    
    const tabs = [
        { id: 'overview', label: 'Overview', icon: <InformationCircleIcon className="w-4 h-4" /> },
        { id: 'config', label: 'Configuration', icon: <CogIcon className="w-4 h-4" /> },
        { id: 'metrics', label: 'Metrics', icon: <ChartBarIcon className="w-4 h-4" /> },
        { id: 'test', label: 'Testing', icon: <BeakerIcon className="w-4 h-4" /> },
    ];
    
    return (
        <Card>
            <div className="border-b border-gray-200">
                <nav className="flex space-x-8">
                    {tabs.map(tab => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id as any)}
                            className={cn(
                                'flex items-center space-x-2 py-4 px-1 border-b-2 font-medium text-sm',
                                activeTab === tab.id
                                    ? 'border-blue-500 text-blue-600'
                                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                            )}
                        >
                            {tab.icon}
                            <span>{tab.label}</span>
                        </button>
                    ))}
                </nav>
            </div>
            
            <div className="p-6">
                {activeTab === 'overview' && (
                    <ProviderOverview provider={provider} />
                )}
                {activeTab === 'config' && (
                    <ProviderConfiguration 
                        provider={provider} 
                        onUpdate={onConfigUpdate}
                    />
                )}
                {activeTab === 'metrics' && (
                    <ProviderMetrics provider={provider} />
                )}
                {activeTab === 'test' && (
                    <ProviderTesting 
                        provider={provider}
                        testResult={testResult}
                    />
                )}
            </div>
        </Card>
    );
};

// frontend/src/components/providers/ProviderMetrics.tsx
const ProviderMetrics: React.FC<{ provider: Provider }> = ({ provider }) => {
    const [timeRange, setTimeRange] = useState('1h');
    const [metrics, setMetrics] = useState<ProviderMetrics | null>(null);
    
    useEffect(() => {
        fetchMetrics();
    }, [provider.name, timeRange]);
    
    const fetchMetrics = async () => {
        try {
            const response = await fetch(`/api/providers/${provider.name}/metrics?range=${timeRange}`);
            const data = await response.json();
            setMetrics(data);
        } catch (error) {
            console.error('Failed to fetch metrics:', error);
        }
    };
    
    if (!metrics) {
        return <Skeleton lines={5} />;
    }
    
    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <h3 className="text-lg font-medium">Performance Metrics</h3>
                <select
                    value={timeRange}
                    onChange={(e) => setTimeRange(e.target.value)}
                    className="px-3 py-2 border border-gray-300 rounded-md"
                >
                    <option value="1h">Last Hour</option>
                    <option value="6h">Last 6 Hours</option>
                    <option value="24h">Last 24 Hours</option>
                    <option value="7d">Last 7 Days</option>
                </select>
            </div>
            
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <MetricCard
                    title="Requests"
                    value={metrics.totalRequests.toString()}
                    change={metrics.requestsChange}
                    icon={<ActivityIcon className="w-5 h-5" />}
                />
                <MetricCard
                    title="Avg Latency"
                    value={`${metrics.averageLatency}ms`}
                    change={metrics.latencyChange}
                    icon={<ClockIcon className="w-5 h-5" />}
                />
                <MetricCard
                    title="Success Rate"
                    value={`${(metrics.successRate * 100).toFixed(1)}%`}
                    change={metrics.successRateChange}
                    icon={<CheckCircleIcon className="w-5 h-5" />}
                />
                <MetricCard
                    title="Cost"
                    value={`$${metrics.totalCost.toFixed(2)}`}
                    change={metrics.costChange}
                    icon={<CurrencyDollarIcon className="w-5 h-5" />}
                />
            </div>
            
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <Card title="Response Time">
                    <LineChart
                        data={metrics.latencyHistory}
                        xAxis="timestamp"
                        yAxis="latency"
                        height={200}
                    />
                </Card>
                
                <Card title="Request Volume">
                    <AreaChart
                        data={metrics.requestHistory}
                        xAxis="timestamp"
                        yAxis="count"
                        height={200}
                    />
                </Card>
            </div>
        </div>
    );
};
```

### Testing Requirements
- [ ] Unit tests for provider management components
- [ ] Integration tests with provider API
- [ ] UI/UX testing for provider management
- [ ] Test configuration editing functionality

### Acceptance Criteria
- [ ] Provider status is displayed accurately in real-time
- [ ] Configuration can be edited through the UI
- [ ] Provider testing tools provide clear results
- [ ] Metrics visualization is helpful and informative
- [ ] UI is responsive and user-friendly

### Risk Mitigation
- Test UI with various provider states
- Ensure configuration editing is safe
- Validate all user inputs thoroughly

---

## **Integration Testing**
**Duration:** 4-5 hours

### Comprehensive Provider Testing
- [ ] End-to-end provider management flow
- [ ] Load testing with intelligent routing
- [ ] Failover and circuit breaker testing
- [ ] Configuration hot-reload testing
- [ ] Provider health monitoring accuracy

### Success Metrics
- [ ] Intelligent routing improves performance by 25%
- [ ] Provider failover completes within 5 seconds
- [ ] Health monitoring accuracy >95%
- [ ] Configuration updates apply within 10 seconds
- [ ] Provider management UI loads within 2 seconds

---

## **Rollback Plan**
If any cycle fails:
1. Revert to previous provider system
2. Disable intelligent routing temporarily
3. Use simple round-robin as fallback
4. Disable problematic provider features