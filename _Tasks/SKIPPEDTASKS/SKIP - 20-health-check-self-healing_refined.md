# Health Check & Self-Healing System - Refined Implementation Cycles

## Overview
Break down comprehensive health monitoring and self-healing into 4 manageable cycles, each delivering testable functionality with clear outcomes.

---

## **Cycle 20A: Core Health Check Framework**
**Duration:** 5-6 hours | **Priority:** Critical

### Prerequisites
- Go development environment ready
- Understanding of health check patterns and system monitoring
- Knowledge of concurrent programming for health check scheduling

### Implementation Tasks
- [ ] Create health check framework architecture
- [ ] Implement basic system health checks (CPU, memory, disk)
- [ ] Add service health checks (database, Redis, external APIs)
- [ ] Create health check registry and scheduler
- [ ] Implement health status aggregation
- [ ] Add health check API endpoints

### Code Deliverables
```go
// backend/health/types.go
package health

import (
    "context"
    "time"
)

type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
)

type Priority int

const (
    PriorityLow Priority = iota
    PriorityMedium
    PriorityHigh
    PriorityCritical
)

type HealthChecker interface {
    Name() string
    Check(ctx context.Context) HealthCheckResult
    Priority() Priority
    Timeout() time.Duration
    Interval() time.Duration
}

type HealthCheckResult struct {
    Status    HealthStatus           `json:"status"`
    Timestamp time.Time             `json:"timestamp"`
    Duration  time.Duration         `json:"duration"`
    Message   string                 `json:"message,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Error     error                  `json:"error,omitempty"`
}

type SystemHealth struct {
    Overall     HealthStatus        `json:"overall"`
    Timestamp   time.Time          `json:"timestamp"`
    Uptime      time.Duration      `json:"uptime"`
    Version     string             `json:"version"`
    Checks      []HealthCheckResult `json:"checks"`
    Issues      []HealthIssue      `json:"issues,omitempty"`
}

type HealthIssue struct {
    CheckName   string       `json:"check_name"`
    Status      HealthStatus `json:"status"`
    Message     string       `json:"message"`
    Timestamp   time.Time    `json:"timestamp"`
    Priority    Priority     `json:"priority"`
    Actionable  bool         `json:"actionable"`
}
```

```go
// backend/health/registry.go
package health

import (
    "context"
    "sync"
    "time"
)

type HealthRegistry struct {
    checkers     map[string]HealthChecker
    results      map[string]HealthCheckResult
    mutex        sync.RWMutex
    scheduler    *HealthScheduler
    startTime    time.Time
    version      string
}

func NewHealthRegistry(version string) *HealthRegistry {
    return &HealthRegistry{
        checkers:  make(map[string]HealthChecker),
        results:   make(map[string]HealthCheckResult),
        startTime: time.Now(),
        version:   version,
    }
}

func (hr *HealthRegistry) Register(checker HealthChecker) {
    hr.mutex.Lock()
    defer hr.mutex.Unlock()
    
    hr.checkers[checker.Name()] = checker
    
    // Initialize with unknown status
    hr.results[checker.Name()] = HealthCheckResult{
        Status:    HealthStatusUnhealthy,
        Timestamp: time.Now(),
        Message:   "Not yet checked",
    }
}

func (hr *HealthRegistry) Start(ctx context.Context) {
    hr.scheduler = NewHealthScheduler(hr.checkers, hr.updateResult)
    hr.scheduler.Start(ctx)
}

func (hr *HealthRegistry) updateResult(name string, result HealthCheckResult) {
    hr.mutex.Lock()
    defer hr.mutex.Unlock()
    hr.results[name] = result
}

func (hr *HealthRegistry) GetSystemHealth() SystemHealth {
    hr.mutex.RLock()
    defer hr.mutex.RUnlock()
    
    checks := make([]HealthCheckResult, 0, len(hr.results))
    issues := make([]HealthIssue, 0)
    overall := HealthStatusHealthy
    
    for name, result := range hr.results {
        checks = append(checks, result)
        
        // Determine overall status
        switch result.Status {
        case HealthStatusUnhealthy:
            overall = HealthStatusUnhealthy
            issues = append(issues, HealthIssue{
                CheckName: name,
                Status:    result.Status,
                Message:   result.Message,
                Timestamp: result.Timestamp,
                Priority:  hr.checkers[name].Priority(),
                Actionable: true,
            })
        case HealthStatusDegraded:
            if overall == HealthStatusHealthy {
                overall = HealthStatusDegraded
            }
            issues = append(issues, HealthIssue{
                CheckName: name,
                Status:    result.Status,
                Message:   result.Message,
                Timestamp: result.Timestamp,
                Priority:  hr.checkers[name].Priority(),
                Actionable: true,
            })
        }
    }
    
    return SystemHealth{
        Overall:   overall,
        Timestamp: time.Now(),
        Uptime:    time.Since(hr.startTime),
        Version:   hr.version,
        Checks:    checks,
        Issues:    issues,
    }
}
```

```go
// backend/health/checkers/system.go
package checkers

import (
    "context"
    "fmt"
    "runtime"
    "time"
    
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/disk"
    "github.com/qt1-middleware/backend/health"
)

type CPUChecker struct {
    threshold float64
}

func NewCPUChecker(threshold float64) *CPUChecker {
    return &CPUChecker{threshold: threshold}
}

func (c *CPUChecker) Name() string {
    return "cpu"
}

func (c *CPUChecker) Priority() health.Priority {
    return health.PriorityHigh
}

func (c *CPUChecker) Timeout() time.Duration {
    return 5 * time.Second
}

func (c *CPUChecker) Interval() time.Duration {
    return 30 * time.Second
}

func (c *CPUChecker) Check(ctx context.Context) health.HealthCheckResult {
    start := time.Now()
    
    usage, err := cpu.PercentWithContext(ctx, time.Second, false)
    if err != nil {
        return health.HealthCheckResult{
            Status:    health.HealthStatusUnhealthy,
            Timestamp: start,
            Duration:  time.Since(start),
            Message:   "Failed to get CPU usage",
            Error:     err,
        }
    }
    
    cpuUsage := usage[0]
    status := health.HealthStatusHealthy
    message := fmt.Sprintf("CPU usage: %.1f%%", cpuUsage)
    
    if cpuUsage > c.threshold {
        status = health.HealthStatusDegraded
        message = fmt.Sprintf("High CPU usage: %.1f%% (threshold: %.1f%%)", cpuUsage, c.threshold)
    }
    
    if cpuUsage > c.threshold*1.5 {
        status = health.HealthStatusUnhealthy
        message = fmt.Sprintf("Critical CPU usage: %.1f%% (threshold: %.1f%%)", cpuUsage, c.threshold)
    }
    
    return health.HealthCheckResult{
        Status:    status,
        Timestamp: start,
        Duration:  time.Since(start),
        Message:   message,
        Details: map[string]interface{}{
            "cpu_usage":   cpuUsage,
            "threshold":   c.threshold,
            "num_cores":   runtime.NumCPU(),
        },
    }
}

type MemoryChecker struct {
    threshold float64
}

func NewMemoryChecker(threshold float64) *MemoryChecker {
    return &MemoryChecker{threshold: threshold}
}

func (m *MemoryChecker) Name() string {
    return "memory"
}

func (m *MemoryChecker) Priority() health.Priority {
    return health.PriorityHigh
}

func (m *MemoryChecker) Timeout() time.Duration {
    return 5 * time.Second
}

func (m *MemoryChecker) Interval() time.Duration {
    return 30 * time.Second
}

func (m *MemoryChecker) Check(ctx context.Context) health.HealthCheckResult {
    start := time.Now()
    
    memInfo, err := mem.VirtualMemoryWithContext(ctx)
    if err != nil {
        return health.HealthCheckResult{
            Status:    health.HealthStatusUnhealthy,
            Timestamp: start,
            Duration:  time.Since(start),
            Message:   "Failed to get memory usage",
            Error:     err,
        }
    }
    
    memUsage := memInfo.UsedPercent
    status := health.HealthStatusHealthy
    message := fmt.Sprintf("Memory usage: %.1f%%", memUsage)
    
    if memUsage > m.threshold {
        status = health.HealthStatusDegraded
        message = fmt.Sprintf("High memory usage: %.1f%% (threshold: %.1f%%)", memUsage, m.threshold)
    }
    
    if memUsage > m.threshold*1.2 {
        status = health.HealthStatusUnhealthy
        message = fmt.Sprintf("Critical memory usage: %.1f%% (threshold: %.1f%%)", memUsage, m.threshold)
    }
    
    return health.HealthCheckResult{
        Status:    status,
        Timestamp: start,
        Duration:  time.Since(start),
        Message:   message,
        Details: map[string]interface{}{
            "memory_usage":    memUsage,
            "threshold":       m.threshold,
            "total_memory":    memInfo.Total,
            "available_memory": memInfo.Available,
            "used_memory":     memInfo.Used,
        },
    }
}

type DatabaseChecker struct {
    db        *sql.DB
    timeout   time.Duration
}

func NewDatabaseChecker(db *sql.DB) *DatabaseChecker {
    return &DatabaseChecker{
        db:      db,
        timeout: 5 * time.Second,
    }
}

func (d *DatabaseChecker) Name() string {
    return "database"
}

func (d *DatabaseChecker) Priority() health.Priority {
    return health.PriorityCritical
}

func (d *DatabaseChecker) Timeout() time.Duration {
    return d.timeout
}

func (d *DatabaseChecker) Interval() time.Duration {
    return 30 * time.Second
}

func (d *DatabaseChecker) Check(ctx context.Context) health.HealthCheckResult {
    start := time.Now()
    
    // Check connection pool stats
    stats := d.db.Stats()
    if stats.OpenConnections == 0 {
        return health.HealthCheckResult{
            Status:    health.HealthStatusUnhealthy,
            Timestamp: start,
            Duration:  time.Since(start),
            Message:   "No database connections available",
        }
    }
    
    // Test query execution
    var result int
    err := d.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
    if err != nil {
        return health.HealthCheckResult{
            Status:    health.HealthStatusUnhealthy,
            Timestamp: start,
            Duration:  time.Since(start),
            Message:   "Database query failed",
            Error:     err,
        }
    }
    
    // Check connection pool health
    status := health.HealthStatusHealthy
    message := "Database connection healthy"
    
    if stats.InUse > stats.MaxOpenConnections*0.8 {
        status = health.HealthStatusDegraded
        message = "High database connection usage"
    }
    
    return health.HealthCheckResult{
        Status:    status,
        Timestamp: start,
        Duration:  time.Since(start),
        Message:   message,
        Details: map[string]interface{}{
            "open_connections":    stats.OpenConnections,
            "in_use":             stats.InUse,
            "idle":               stats.Idle,
            "max_open_connections": stats.MaxOpenConnections,
            "query_duration_ms":   time.Since(start).Milliseconds(),
        },
    }
}
```

### Testing Requirements
- [ ] Test health check registration and execution
- [ ] Test health status aggregation logic
- [ ] Test health check timeouts and error handling
- [ ] Test concurrent health check execution
- [ ] Test health API endpoints

### Acceptance Criteria
- [ ] Health checks execute on schedule without blocking
- [ ] Health status accurately reflects system state
- [ ] Health API responds within 2 seconds
- [ ] Failed health checks don't crash the system
- [ ] Health check results are properly aggregated
- [ ] Health registry handles concurrent access safely

### Risk Mitigation
- Use context timeouts to prevent hanging health checks
- Implement proper error handling and logging
- Test health checks under various system conditions

---

## **Cycle 20B: Self-Healing Mechanisms**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 20A completed and tested
- Understanding of common system failure patterns
- Knowledge of recovery and healing strategies

### Implementation Tasks
- [ ] Create self-healing framework
- [ ] Implement connection pool healing
- [ ] Add memory and resource leak detection/cleanup
- [ ] Create circuit breaker patterns
- [ ] Add automatic service restart capabilities
- [ ] Implement healing action tracking and logging

### Code Deliverables
```go
// backend/health/healing.go
package health

import (
    "context"
    "database/sql"
    "log"
    "sync"
    "time"
)

type SelfHealer interface {
    CanHeal(issue HealthIssue) bool
    Heal(ctx context.Context, issue HealthIssue) HealingResult
    Priority() Priority
}

type HealingResult struct {
    Success   bool          `json:"success"`
    Action    string        `json:"action"`
    Message   string        `json:"message"`
    Duration  time.Duration `json:"duration"`
    Timestamp time.Time     `json:"timestamp"`
    Error     error         `json:"error,omitempty"`
}

type HealingEngine struct {
    healers     map[string]SelfHealer
    healingLog  []HealingResult
    mutex       sync.RWMutex
    maxAttempts int
    cooldown    time.Duration
    lastHealing map[string]time.Time
}

func NewHealingEngine(maxAttempts int, cooldown time.Duration) *HealingEngine {
    return &HealingEngine{
        healers:     make(map[string]SelfHealer),
        healingLog:  make([]HealingResult, 0),
        maxAttempts: maxAttempts,
        cooldown:    cooldown,
        lastHealing: make(map[string]time.Time),
    }
}

func (he *HealingEngine) RegisterHealer(name string, healer SelfHealer) {
    he.mutex.Lock()
    defer he.mutex.Unlock()
    he.healers[name] = healer
}

func (he *HealingEngine) TriggerHealing(ctx context.Context, issues []HealthIssue) []HealingResult {
    he.mutex.Lock()
    defer he.mutex.Unlock()
    
    var results []HealingResult
    
    for _, issue := range issues {
        // Check cooldown period
        if lastHeal, exists := he.lastHealing[issue.CheckName]; exists {
            if time.Since(lastHeal) < he.cooldown {
                continue
            }
        }
        
        // Find suitable healer
        for name, healer := range he.healers {
            if healer.CanHeal(issue) {
                log.Printf("Attempting to heal issue: %s with healer: %s", issue.CheckName, name)
                
                result := healer.Heal(ctx, issue)
                result.Timestamp = time.Now()
                
                results = append(results, result)
                he.healingLog = append(he.healingLog, result)
                he.lastHealing[issue.CheckName] = time.Now()
                
                if result.Success {
                    log.Printf("Successfully healed issue: %s", issue.CheckName)
                    break
                } else {
                    log.Printf("Failed to heal issue: %s - %s", issue.CheckName, result.Message)
                }
            }
        }
    }
    
    return results
}

func (he *HealingEngine) GetHealingHistory() []HealingResult {
    he.mutex.RLock()
    defer he.mutex.RUnlock()
    
    // Return copy of healing log
    history := make([]HealingResult, len(he.healingLog))
    copy(history, he.healingLog)
    return history
}
```

```go
// backend/health/healers/connection.go
package healers

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/qt1-middleware/backend/health"
)

type DatabaseConnectionHealer struct {
    db     *sql.DB
    logger *log.Logger
}

func NewDatabaseConnectionHealer(db *sql.DB, logger *log.Logger) *DatabaseConnectionHealer {
    return &DatabaseConnectionHealer{
        db:     db,
        logger: logger,
    }
}

func (dch *DatabaseConnectionHealer) CanHeal(issue health.HealthIssue) bool {
    return issue.CheckName == "database" && issue.Status == health.HealthStatusUnhealthy
}

func (dch *DatabaseConnectionHealer) Priority() health.Priority {
    return health.PriorityCritical
}

func (dch *DatabaseConnectionHealer) Heal(ctx context.Context, issue health.HealthIssue) health.HealingResult {
    start := time.Now()
    
    dch.logger.Info("Attempting to heal database connection issues")
    
    // Try to reset connection pool
    stats := dch.db.Stats()
    originalMaxOpen := stats.MaxOpenConnections
    
    // Temporarily reduce connections to force reset
    dch.db.SetMaxOpenConns(1)
    time.Sleep(100 * time.Millisecond)
    
    // Restore original settings with optimizations
    dch.db.SetMaxOpenConns(originalMaxOpen)
    dch.db.SetMaxIdleConns(originalMaxOpen / 4)
    dch.db.SetConnMaxLifetime(time.Hour)
    dch.db.SetConnMaxIdleTime(15 * time.Minute)
    
    // Test connection
    err := dch.db.PingContext(ctx)
    success := err == nil
    
    message := "Database connection pool reset"
    if !success {
        message = fmt.Sprintf("Database connection healing failed: %v", err)
    }
    
    return health.HealingResult{
        Success:   success,
        Action:    "reset_database_connections",
        Message:   message,
        Duration:  time.Since(start),
        Error:     err,
    }
}

type RedisConnectionHealer struct {
    redis  *redis.Client
    logger *log.Logger
}

func NewRedisConnectionHealer(redis *redis.Client, logger *log.Logger) *RedisConnectionHealer {
    return &RedisConnectionHealer{
        redis:  redis,
        logger: logger,
    }
}

func (rch *RedisConnectionHealer) CanHeal(issue health.HealthIssue) bool {
    return issue.CheckName == "redis" && issue.Status != health.HealthStatusHealthy
}

func (rch *RedisConnectionHealer) Priority() health.Priority {
    return health.PriorityHigh
}

func (rch *RedisConnectionHealer) Heal(ctx context.Context, issue health.HealthIssue) health.HealingResult {
    start := time.Now()
    
    rch.logger.Info("Attempting to heal Redis connection issues")
    
    // Force reconnection
    err := rch.redis.Conn().Close()
    if err != nil {
        rch.logger.Printf("Error closing Redis connection: %v", err)
    }
    
    // Test new connection
    _, err = rch.redis.Ping(ctx).Result()
    success := err == nil
    
    message := "Redis connection reset"
    if !success {
        message = fmt.Sprintf("Redis connection healing failed: %v", err)
    }
    
    return health.HealingResult{
        Success:   success,
        Action:    "reset_redis_connection",
        Message:   message,
        Duration:  time.Since(start),
        Error:     err,
    }
}

type MemoryHealer struct {
    threshold float64
    logger    *log.Logger
}

func NewMemoryHealer(threshold float64, logger *log.Logger) *MemoryHealer {
    return &MemoryHealer{
        threshold: threshold,
        logger:    logger,
    }
}

func (mh *MemoryHealer) CanHeal(issue health.HealthIssue) bool {
    return issue.CheckName == "memory" && issue.Status == health.HealthStatusDegraded
}

func (mh *MemoryHealer) Priority() health.Priority {
    return health.PriorityMedium
}

func (mh *MemoryHealer) Heal(ctx context.Context, issue health.HealthIssue) health.HealingResult {
    start := time.Now()
    
    mh.logger.Info("Attempting to free memory")
    
    // Get memory stats before
    var memBefore runtime.MemStats
    runtime.ReadMemStats(&memBefore)
    
    // Force garbage collection
    runtime.GC()
    runtime.GC() // Double GC for better cleanup
    
    // Return unused memory to OS
    debug.FreeOSMemory()
    
    // Get memory stats after
    var memAfter runtime.MemStats
    runtime.ReadMemStats(&memAfter)
    
    freedMB := float64(memBefore.Sys-memAfter.Sys) / 1024 / 1024
    success := freedMB > 0
    
    message := fmt.Sprintf("Memory cleanup completed. Freed: %.1f MB", freedMB)
    if !success {
        message = "Memory cleanup did not free significant memory"
    }
    
    return health.HealingResult{
        Success:  success,
        Action:   "garbage_collection",
        Message:  message,
        Duration: time.Since(start),
    }
}

type CircuitBreakerHealer struct {
    breakers map[string]*CircuitBreaker
    logger   *log.Logger
}

func NewCircuitBreakerHealer(breakers map[string]*CircuitBreaker, logger *log.Logger) *CircuitBreakerHealer {
    return &CircuitBreakerHealer{
        breakers: breakers,
        logger:   logger,
    }
}

func (cbh *CircuitBreakerHealer) CanHeal(issue health.HealthIssue) bool {
    _, exists := cbh.breakers[issue.CheckName]
    return exists && issue.Status == health.HealthStatusUnhealthy
}

func (cbh *CircuitBreakerHealer) Priority() health.Priority {
    return health.PriorityHigh
}

func (cbh *CircuitBreakerHealer) Heal(ctx context.Context, issue health.HealthIssue) health.HealingResult {
    start := time.Now()
    
    breaker, exists := cbh.breakers[issue.CheckName]
    if !exists {
        return health.HealingResult{
            Success:  false,
            Action:   "circuit_breaker_reset",
            Message:  "Circuit breaker not found",
            Duration: time.Since(start),
        }
    }
    
    cbh.logger.Printf("Resetting circuit breaker for: %s", issue.CheckName)
    
    // Reset circuit breaker
    breaker.Reset()
    
    return health.HealingResult{
        Success:  true,
        Action:   "circuit_breaker_reset",
        Message:  fmt.Sprintf("Circuit breaker reset for %s", issue.CheckName),
        Duration: time.Since(start),
    }
}
```

### Testing Requirements
- [ ] Test self-healing triggers correctly
- [ ] Test healing actions work as expected
- [ ] Test healing cooldown periods
- [ ] Test healing action logging
- [ ] Test failed healing scenarios

### Acceptance Criteria
- [ ] Self-healing triggers automatically on health issues
- [ ] Healing actions complete within 30 seconds
- [ ] Healing success rate >80% for common issues
- [ ] Healing actions don't cause additional problems
- [ ] Healing history is tracked and accessible
- [ ] Cooldown prevents healing loops

### Risk Mitigation
- Implement conservative healing actions initially
- Add extensive logging and monitoring
- Test healing actions thoroughly in non-production

---

## **Cycle 20C: Health Dashboard and Monitoring Integration**
**Duration:** 4-5 hours | **Priority:** Medium

### Prerequisites
- Cycles 20A and 20B completed
- React dashboard development environment
- Understanding of real-time data visualization

### Implementation Tasks
- [ ] Create health monitoring dashboard
- [ ] Add real-time health status visualization
- [ ] Implement health trends and history
- [ ] Create healing action management interface
- [ ] Add health check configuration UI
- [ ] Integrate with alerting system

### Code Deliverables
```typescript
// frontend/src/components/health/HealthDashboard.tsx
import React, { useState, useEffect } from 'react';
import { AlertCircle, CheckCircle, Clock, Settings, RefreshCw } from 'lucide-react';
import { useWebSocket } from '../../hooks/useWebSocket';

interface HealthDashboardProps {
  onTriggerHealing?: (checkName: string) => Promise<void>;
  onConfigureCheck?: (checkName: string) => void;
}

interface SystemHealth {
  overall: 'healthy' | 'degraded' | 'unhealthy';
  timestamp: string;
  uptime: string;
  version: string;
  checks: HealthCheck[];
  issues: HealthIssue[];
}

interface HealthCheck {
  name: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  message: string;
  duration: number;
  timestamp: string;
  details?: Record<string, any>;
}

interface HealthIssue {
  check_name: string;
  status: 'degraded' | 'unhealthy';
  message: string;
  timestamp: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  actionable: boolean;
}

export const HealthDashboard: React.FC<HealthDashboardProps> = ({
  onTriggerHealing,
  onConfigureCheck
}) => {
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null);
  const [healingHistory, setHealingHistory] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  // WebSocket connection for real-time updates
  const { lastMessage, sendMessage } = useWebSocket('/ws/health');

  useEffect(() => {
    fetchSystemHealth();
    const interval = setInterval(fetchSystemHealth, 30000); // Update every 30 seconds
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (lastMessage) {
      const data = JSON.parse(lastMessage.data);
      if (data.type === 'health_update') {
        setSystemHealth(data.payload);
        setLastUpdate(new Date());
      }
    }
  }, [lastMessage]);

  const fetchSystemHealth = async () => {
    try {
      setIsLoading(true);
      const response = await fetch('/api/health/detailed');
      const data = await response.json();
      setSystemHealth(data);
      setLastUpdate(new Date());
    } catch (error) {
      console.error('Failed to fetch system health:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchHealingHistory = async () => {
    try {
      const response = await fetch('/api/health/healing/history');
      const data = await response.json();
      setHealingHistory(data);
    } catch (error) {
      console.error('Failed to fetch healing history:', error);
    }
  };

  const triggerManualHealing = async (checkName: string) => {
    if (onTriggerHealing) {
      await onTriggerHealing(checkName);
      await fetchHealingHistory();
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="w-5 h-5 text-green-500" />;
      case 'degraded':
        return <AlertCircle className="w-5 h-5 text-yellow-500" />;
      case 'unhealthy':
        return <AlertCircle className="w-5 h-5 text-red-500" />;
      default:
        return <Clock className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'bg-green-50 border-green-200';
      case 'degraded': return 'bg-yellow-50 border-yellow-200';
      case 'unhealthy': return 'bg-red-50 border-red-200';
      default: return 'bg-gray-50 border-gray-200';
    }
  };

  if (isLoading && !systemHealth) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
        <span className="ml-2">Loading health status...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Overall System Status */}
      <div className={`rounded-lg border p-6 ${getStatusColor(systemHealth?.overall || 'unhealthy')}`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            {getStatusIcon(systemHealth?.overall || 'unhealthy')}
            <div>
              <h2 className="text-xl font-semibold">
                System {systemHealth?.overall === 'healthy' ? 'Online' : 'Issues Detected'}
              </h2>
              <p className="text-sm text-gray-600">
                Uptime: {systemHealth?.uptime} | Version: {systemHealth?.version}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-sm text-gray-500">Last updated</p>
            <p className="text-sm font-medium">{lastUpdate.toLocaleTimeString()}</p>
          </div>
        </div>
      </div>

      {/* Health Checks Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {systemHealth?.checks.map((check) => (
          <div
            key={check.name}
            className={`rounded-lg border p-4 ${getStatusColor(check.status)}`}
          >
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center space-x-2">
                {getStatusIcon(check.status)}
                <h3 className="font-medium capitalize">{check.name}</h3>
              </div>
              <div className="flex space-x-1">
                <button
                  onClick={() => onConfigureCheck?.(check.name)}
                  className="p-1 hover:bg-gray-200 rounded"
                  title="Configure check"
                >
                  <Settings className="w-4 h-4 text-gray-500" />
                </button>
                {check.status !== 'healthy' && (
                  <button
                    onClick={() => triggerManualHealing(check.name)}
                    className="p-1 hover:bg-gray-200 rounded"
                    title="Trigger healing"
                  >
                    <RefreshCw className="w-4 h-4 text-blue-500" />
                  </button>
                )}
              </div>
            </div>
            
            <p className="text-sm text-gray-700 mb-2">{check.message}</p>
            
            <div className="flex justify-between text-xs text-gray-500">
              <span>Response: {check.duration}ms</span>
              <span>{new Date(check.timestamp).toLocaleTimeString()}</span>
            </div>
            
            {check.details && (
              <details className="mt-2">
                <summary className="text-xs text-gray-500 cursor-pointer">Details</summary>
                <pre className="text-xs mt-1 bg-gray-100 p-2 rounded overflow-x-auto">
                  {JSON.stringify(check.details, null, 2)}
                </pre>
              </details>
            )}
          </div>
        ))}
      </div>

      {/* Active Issues */}
      {systemHealth?.issues && systemHealth.issues.length > 0 && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <h3 className="font-medium text-red-800 mb-3">Active Issues</h3>
          <div className="space-y-2">
            {systemHealth.issues.map((issue, index) => (
              <div key={index} className="flex items-center justify-between bg-white p-3 rounded border">
                <div className="flex items-center space-x-3">
                  {getStatusIcon(issue.status)}
                  <div>
                    <p className="font-medium">{issue.check_name}</p>
                    <p className="text-sm text-gray-600">{issue.message}</p>
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <span className={`px-2 py-1 text-xs rounded ${
                    issue.priority === 'critical' ? 'bg-red-100 text-red-800' :
                    issue.priority === 'high' ? 'bg-orange-100 text-orange-800' :
                    issue.priority === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {issue.priority}
                  </span>
                  {issue.actionable && (
                    <button
                      onClick={() => triggerManualHealing(issue.check_name)}
                      className="px-3 py-1 text-xs bg-blue-100 text-blue-800 rounded hover:bg-blue-200"
                    >
                      Heal
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Healing History */}
      <div className="bg-white border rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-medium">Recent Healing Actions</h3>
          <button
            onClick={fetchHealingHistory}
            className="text-sm text-blue-600 hover:text-blue-800"
          >
            Refresh
          </button>
        </div>
        
        <div className="space-y-2 max-h-60 overflow-y-auto">
          {healingHistory.map((action, index) => (
            <div key={index} className="flex items-center justify-between py-2 border-b">
              <div className="flex items-center space-x-3">
                {action.success ? 
                  <CheckCircle className="w-4 h-4 text-green-500" /> :
                  <AlertCircle className="w-4 h-4 text-red-500" />
                }
                <div>
                  <p className="text-sm font-medium">{action.action}</p>
                  <p className="text-xs text-gray-500">{action.message}</p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-xs text-gray-500">{action.duration}ms</p>
                <p className="text-xs text-gray-400">
                  {new Date(action.timestamp).toLocaleTimeString()}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
```

```go
// backend/api/health.go
package api

import (
    "encoding/json"
    "net/http"
    "strconv"
    
    "github.com/gorilla/mux"
    "github.com/qt1-middleware/backend/health"
)

type HealthHandler struct {
    registry *health.HealthRegistry
    healer   *health.HealingEngine
}

func NewHealthHandler(registry *health.HealthRegistry, healer *health.HealingEngine) *HealthHandler {
    return &HealthHandler{
        registry: registry,
        healer:   healer,
    }
}

func (h *HealthHandler) RegisterRoutes(r *mux.Router) {
    r.HandleFunc("/health", h.GetBasicHealth).Methods("GET")
    r.HandleFunc("/health/detailed", h.GetDetailedHealth).Methods("GET")
    r.HandleFunc("/health/heal", h.TriggerHealing).Methods("POST")
    r.HandleFunc("/health/healing/history", h.GetHealingHistory).Methods("GET")
}

func (h *HealthHandler) GetBasicHealth(w http.ResponseWriter, r *http.Request) {
    systemHealth := h.registry.GetSystemHealth()
    
    status := http.StatusOK
    if systemHealth.Overall == health.HealthStatusUnhealthy {
        status = http.StatusServiceUnavailable
    } else if systemHealth.Overall == health.HealthStatusDegraded {
        status = http.StatusOK // Still accepting requests
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    
    basicHealth := map[string]interface{}{
        "status":    systemHealth.Overall,
        "timestamp": systemHealth.Timestamp,
        "uptime":    systemHealth.Uptime.String(),
        "version":   systemHealth.Version,
    }
    
    json.NewEncoder(w).Encode(basicHealth)
}

func (h *HealthHandler) GetDetailedHealth(w http.ResponseWriter, r *http.Request) {
    systemHealth := h.registry.GetSystemHealth()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(systemHealth)
}

func (h *HealthHandler) TriggerHealing(w http.ResponseWriter, r *http.Request) {
    systemHealth := h.registry.GetSystemHealth()
    
    if len(systemHealth.Issues) == 0 {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "message": "No issues to heal",
        })
        return
    }
    
    results := h.healer.TriggerHealing(r.Context(), systemHealth.Issues)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "healing_results": results,
        "total_attempts":  len(results),
    })
}

func (h *HealthHandler) GetHealingHistory(w http.ResponseWriter, r *http.Request) {
    limitStr := r.URL.Query().Get("limit")
    limit := 50 // default
    
    if limitStr != "" {
        if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
            limit = parsedLimit
        }
    }
    
    history := h.healer.GetHealingHistory()
    
    // Return most recent results up to limit
    start := 0
    if len(history) > limit {
        start = len(history) - limit
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(history[start:])
}
```

### Testing Requirements
- [ ] Test health dashboard rendering
- [ ] Test real-time health updates
- [ ] Test manual healing triggers
- [ ] Test health configuration interface
- [ ] Test responsive design

### Acceptance Criteria
- [ ] Dashboard displays real-time health status
- [ ] Health checks update automatically every 30 seconds
- [ ] Manual healing triggers work correctly
- [ ] Health trends and history are visualized
- [ ] Dashboard is responsive on mobile devices
- [ ] WebSocket updates work reliably

### Risk Mitigation
- Implement fallback polling if WebSocket fails
- Add proper error handling for API calls
- Test dashboard performance with many health checks

---

## **Cycle 20D: Advanced Diagnostics and Proactive Maintenance**
**Duration:** 4-5 hours | **Priority:** Low

### Prerequisites
- Cycles 20A, 20B, and 20C completed
- Understanding of predictive maintenance concepts
- Knowledge of system performance analysis

### Implementation Tasks
- [ ] Create advanced diagnostic tools
- [ ] Implement predictive health monitoring
- [ ] Add automated maintenance tasks
- [ ] Create health trend analysis
- [ ] Add capacity planning insights
- [ ] Implement health benchmarking

### Code Deliverables
```go
// backend/health/diagnostics.go
package health

import (
    "context"
    "fmt"
    "math"
    "sync"
    "time"
)

type DiagnosticEngine struct {
    registry         *health.HealthRegistry
    metrics          *MetricsCollector
    benchmarks       map[string]HealthBenchmark
    trends           map[string]*HealthTrend
    maintenance      *MaintenanceScheduler
    mutex            sync.RWMutex
    predictionModel  *HealthPredictor
}

type HealthBenchmark struct {
    CheckName       string    `json:"check_name"`
    BaselineValue   float64   `json:"baseline_value"`
    Threshold       float64   `json:"threshold"`
    LastUpdated     time.Time `json:"last_updated"`
    SampleCount     int       `json:"sample_count"`
}

type HealthTrend struct {
    CheckName       string            `json:"check_name"`
    DataPoints      []TrendDataPoint  `json:"data_points"`
    Trend           string            `json:"trend"` // improving, degrading, stable
    Confidence      float64           `json:"confidence"`
    PredictedIssue  *PredictedIssue   `json:"predicted_issue,omitempty"`
}

type TrendDataPoint struct {
    Timestamp time.Time `json:"timestamp"`
    Value     float64   `json:"value"`
    Status    string    `json:"status"`
}

type PredictedIssue struct {
    CheckName       string        `json:"check_name"`
    PredictedTime   time.Time     `json:"predicted_time"`
    Confidence      float64       `json:"confidence"`
    Severity        string        `json:"severity"`
    Recommendation  string        `json:"recommendation"`
}

type DiagnosticReport struct {
    Timestamp       time.Time           `json:"timestamp"`
    SystemHealth    SystemHealth        `json:"system_health"`
    Trends          map[string]*HealthTrend `json:"trends"`
    Predictions     []PredictedIssue    `json:"predictions"`
    Recommendations []Recommendation    `json:"recommendations"`
    Benchmarks      map[string]HealthBenchmark `json:"benchmarks"`
}

type Recommendation struct {
    Priority    string    `json:"priority"`
    Category    string    `json:"category"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Action      string    `json:"action"`
    Impact      string    `json:"impact"`
}

func NewDiagnosticEngine(registry *health.HealthRegistry) *DiagnosticEngine {
    return &DiagnosticEngine{
        registry:        registry,
        benchmarks:      make(map[string]HealthBenchmark),
        trends:          make(map[string]*HealthTrend),
        maintenance:     NewMaintenanceScheduler(),
        predictionModel: NewHealthPredictor(),
    }
}

func (de *DiagnosticEngine) RunDiagnostics(ctx context.Context) DiagnosticReport {
    de.mutex.Lock()
    defer de.mutex.Unlock()
    
    systemHealth := de.registry.GetSystemHealth()
    
    // Update trends
    de.updateTrends(systemHealth)
    
    // Generate predictions
    predictions := de.generatePredictions()
    
    // Generate recommendations
    recommendations := de.generateRecommendations(systemHealth)
    
    return DiagnosticReport{
        Timestamp:       time.Now(),
        SystemHealth:    systemHealth,
        Trends:          de.trends,
        Predictions:     predictions,
        Recommendations: recommendations,
        Benchmarks:      de.benchmarks,
    }
}

func (de *DiagnosticEngine) updateTrends(systemHealth SystemHealth) {
    for _, check := range systemHealth.Checks {
        trend, exists := de.trends[check.Name]
        if !exists {
            trend = &HealthTrend{
                CheckName:  check.Name,
                DataPoints: make([]TrendDataPoint, 0),
            }
            de.trends[check.Name] = trend
        }
        
        // Add new data point
        value := de.extractNumericValue(check)
        dataPoint := TrendDataPoint{
            Timestamp: time.Now(),
            Value:     value,
            Status:    string(check.Status),
        }
        
        trend.DataPoints = append(trend.DataPoints, dataPoint)
        
        // Keep only last 100 data points
        if len(trend.DataPoints) > 100 {
            trend.DataPoints = trend.DataPoints[1:]
        }
        
        // Analyze trend
        de.analyzeTrend(trend)
    }
}

func (de *DiagnosticEngine) extractNumericValue(check HealthCheckResult) float64 {
    // Extract meaningful numeric values from health check details
    if check.Details == nil {
        return 0
    }
    
    switch check.Name {
    case "cpu":
        if usage, ok := check.Details["cpu_usage"].(float64); ok {
            return usage
        }
    case "memory":
        if usage, ok := check.Details["memory_usage"].(float64); ok {
            return usage
        }
    case "database":
        if responseTime, ok := check.Details["query_duration_ms"].(int64); ok {
            return float64(responseTime)
        }
    }
    
    return 0
}

func (de *DiagnosticEngine) analyzeTrend(trend *HealthTrend) {
    if len(trend.DataPoints) < 5 {
        trend.Trend = "insufficient_data"
        trend.Confidence = 0
        return
    }
    
    // Simple linear regression for trend analysis
    n := len(trend.DataPoints)
    recent := trend.DataPoints[max(0, n-20):] // Last 20 points
    
    if len(recent) < 3 {
        return
    }
    
    // Calculate slope
    sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
    for i, point := range recent {
        x := float64(i)
        y := point.Value
        sumX += x
        sumY += y
        sumXY += x * y
        sumX2 += x * x
    }
    
    n64 := float64(len(recent))
    slope := (n64*sumXY - sumX*sumY) / (n64*sumX2 - sumX*sumX)
    
    // Determine trend direction
    if math.Abs(slope) < 0.1 {
        trend.Trend = "stable"
    } else if slope > 0 {
        trend.Trend = "degrading" // For metrics like CPU, memory usage
    } else {
        trend.Trend = "improving"
    }
    
    // Calculate confidence based on variance
    variance := de.calculateVariance(recent)
    trend.Confidence = math.Max(0, math.Min(1, 1-variance/100))
}

func (de *DiagnosticEngine) generatePredictions() []PredictedIssue {
    var predictions []PredictedIssue
    
    for _, trend := range de.trends {
        if trend.Trend == "degrading" && trend.Confidence > 0.7 {
            prediction := de.predictionModel.PredictIssue(trend)
            if prediction != nil {
                predictions = append(predictions, *prediction)
            }
        }
    }
    
    return predictions
}

func (de *DiagnosticEngine) generateRecommendations(systemHealth SystemHealth) []Recommendation {
    var recommendations []Recommendation
    
    // Analyze system state and generate recommendations
    for _, check := range systemHealth.Checks {
        switch check.Name {
        case "cpu":
            if check.Status == health.HealthStatusDegraded {
                recommendations = append(recommendations, Recommendation{
                    Priority:    "medium",
                    Category:    "performance",
                    Title:       "High CPU Usage Detected",
                    Description: "CPU usage is above normal levels",
                    Action:      "investigate_cpu_usage",
                    Impact:      "May affect system responsiveness",
                })
            }
        case "memory":
            if check.Status == health.HealthStatusDegraded {
                recommendations = append(recommendations, Recommendation{
                    Priority:    "medium",
                    Category:    "performance",
                    Title:       "High Memory Usage Detected",
                    Description: "Memory usage is above recommended levels",
                    Action:      "memory_optimization",
                    Impact:      "Risk of out-of-memory errors",
                })
            }
        case "database":
            if check.Status != health.HealthStatusHealthy {
                recommendations = append(recommendations, Recommendation{
                    Priority:    "high",
                    Category:    "reliability",
                    Title:       "Database Performance Issues",
                    Description: "Database health check indicates problems",
                    Action:      "database_optimization",
                    Impact:      "May affect application functionality",
                })
            }
        }
    }
    
    return recommendations
}

type MaintenanceScheduler struct {
    tasks []MaintenanceTask
    mutex sync.RWMutex
}

type MaintenanceTask struct {
    ID          string        `json:"id"`
    Name        string        `json:"name"`
    Schedule    string        `json:"schedule"` // Cron expression
    LastRun     time.Time     `json:"last_run"`
    NextRun     time.Time     `json:"next_run"`
    Duration    time.Duration `json:"duration"`
    Status      string        `json:"status"`
    Description string        `json:"description"`
    Handler     func(context.Context) error `json:"-"`
}

func NewMaintenanceScheduler() *MaintenanceScheduler {
    return &MaintenanceScheduler{
        tasks: make([]MaintenanceTask, 0),
    }
}

func (ms *MaintenanceScheduler) AddTask(task MaintenanceTask) {
    ms.mutex.Lock()
    defer ms.mutex.Unlock()
    
    // Calculate next run time
    task.NextRun = ms.calculateNextRun(task.Schedule)
    ms.tasks = append(ms.tasks, task)
}

func (ms *MaintenanceScheduler) RunDueTasks(ctx context.Context) {
    ms.mutex.Lock()
    defer ms.mutex.Unlock()
    
    now := time.Now()
    for i := range ms.tasks {
        task := &ms.tasks[i]
        if now.After(task.NextRun) && task.Status != "running" {
            go ms.executeTask(ctx, task)
        }
    }
}

func (ms *MaintenanceScheduler) executeTask(ctx context.Context, task *MaintenanceTask) {
    task.Status = "running"
    task.LastRun = time.Now()
    
    start := time.Now()
    err := task.Handler(ctx)
    task.Duration = time.Since(start)
    
    if err != nil {
        task.Status = fmt.Sprintf("failed: %v", err)
    } else {
        task.Status = "completed"
    }
    
    // Calculate next run
    task.NextRun = ms.calculateNextRun(task.Schedule)
}

// Built-in maintenance tasks
func (de *DiagnosticEngine) RegisterMaintenanceTasks() {
    // Log cleanup task
    de.maintenance.AddTask(MaintenanceTask{
        ID:          "log_cleanup",
        Name:        "Log File Cleanup",
        Schedule:    "0 2 * * *", // Daily at 2 AM
        Description: "Clean up old log files to free disk space",
        Handler: func(ctx context.Context) error {
            return de.cleanupLogs()
        },
    })
    
    // Memory optimization task
    de.maintenance.AddTask(MaintenanceTask{
        ID:          "memory_optimization",
        Name:        "Memory Optimization",
        Schedule:    "0 */4 * * *", // Every 4 hours
        Description: "Force garbage collection and memory optimization",
        Handler: func(ctx context.Context) error {
            return de.optimizeMemory()
        },
    })
    
    // Health benchmark update
    de.maintenance.AddTask(MaintenanceTask{
        ID:          "benchmark_update",
        Name:        "Update Health Benchmarks",
        Schedule:    "0 1 * * 0", // Weekly on Sunday at 1 AM
        Description: "Update baseline health benchmarks",
        Handler: func(ctx context.Context) error {
            return de.updateBenchmarks()
        },
    })
}
```

### Testing Requirements
- [ ] Test diagnostic report generation
- [ ] Test trend analysis accuracy
- [ ] Test predictive maintenance alerts
- [ ] Test maintenance task scheduling
- [ ] Test recommendation generation

### Acceptance Criteria
- [ ] Diagnostic reports provide actionable insights
- [ ] Trend analysis detects performance degradation
- [ ] Predictive alerts give 24-48 hour advance warning
- [ ] Maintenance tasks run on schedule automatically
- [ ] Recommendations are relevant and prioritized correctly
- [ ] System capacity planning data is accurate

### Risk Mitigation
- Validate prediction algorithms with historical data
- Start with conservative maintenance schedules
- Monitor impact of automated maintenance tasks

---

## **Integration Testing Checklist**
After all cycles complete:
- [ ] Full health monitoring system integration test
- [ ] Self-healing trigger and execution test under load
- [ ] Dashboard real-time updates with WebSocket failover
- [ ] Predictive maintenance accuracy validation
- [ ] Health system performance impact assessment

## **Success Metrics**
- Health checks execute reliably with <5s response time
- Self-healing success rate >80% for common issues  
- Health dashboard loads and updates in real-time
- Predictive alerts achieve >70% accuracy
- System uptime improved by 25% through proactive healing
- Health monitoring overhead <2% of system resources