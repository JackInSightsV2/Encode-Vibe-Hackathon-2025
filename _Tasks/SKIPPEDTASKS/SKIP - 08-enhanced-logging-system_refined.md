# Enhanced Logging System - Refined Implementation Cycles

## Overview
Break down logging enhancement into 5 manageable cycles, from structured logging to advanced search.

---

## **Cycle 8A: Structured Logging Foundation**
**Duration:** 4-5 hours | **Priority:** Critical

### Prerequisites
- Basic Go logging knowledge
- Understanding of structured logging concepts

### Implementation Tasks
- [ ] Install structured logging dependencies (`github.com/sirupsen/logrus` or `go.uber.org/zap`)
- [ ] Create `backend/logging/logger.go` with structured logging
- [ ] Replace existing log statements with structured format
- [ ] Add contextual logging with request correlation IDs
- [ ] Implement log sampling for high-volume scenarios

### Code Deliverables
```go
// backend/logging/logger.go
type StructuredLogger struct {
    logger   *logrus.Logger
    config   *LogConfig
    sampling *LogSampling
}

type LogEntry struct {
    Timestamp   time.Time             `json:"timestamp"`
    Level       string                `json:"level"`
    Service     string                `json:"service"`
    Component   string                `json:"component"`
    RequestID   string                `json:"request_id,omitempty"`
    UserID      string                `json:"user_id,omitempty"`
    SessionID   string                `json:"session_id,omitempty"`
    Message     string                `json:"message"`
    Duration    *time.Duration        `json:"duration_ms,omitempty"`
    StatusCode  *int                  `json:"status_code,omitempty"`
    Error       string                `json:"error,omitempty"`
    Tags        []string              `json:"tags,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (sl *StructuredLogger) WithContext(ctx context.Context) *LogEntryBuilder {
    builder := &LogEntryBuilder{
        logger: sl,
        entry: LogEntry{
            Timestamp: time.Now(),
            Service:   "qt1-middleware",
        },
    }
    
    // Extract context information
    if requestID := ctx.Value("request_id"); requestID != nil {
        builder.entry.RequestID = requestID.(string)
    }
    if userID := ctx.Value("user_id"); userID != nil {
        builder.entry.UserID = userID.(string)
    }
    
    return builder
}

type LogEntryBuilder struct {
    logger *StructuredLogger
    entry  LogEntry
}

func (leb *LogEntryBuilder) WithComponent(component string) *LogEntryBuilder {
    leb.entry.Component = component
    return leb
}

func (leb *LogEntryBuilder) WithDuration(duration time.Duration) *LogEntryBuilder {
    leb.entry.Duration = &duration
    return leb
}

func (leb *LogEntryBuilder) WithError(err error) *LogEntryBuilder {
    if err != nil {
        leb.entry.Error = err.Error()
    }
    return leb
}

func (leb *LogEntryBuilder) Info(message string) {
    leb.entry.Level = "info"
    leb.entry.Message = message
    leb.logger.writeLog(leb.entry)
}
```

### Testing Requirements
- [ ] Unit tests for structured logging
- [ ] Test log format consistency
- [ ] Test contextual information extraction
- [ ] Performance test for logging overhead

### Acceptance Criteria
- [ ] All logs use consistent JSON structure
- [ ] Contextual information is captured correctly
- [ ] Log sampling reduces volume appropriately
- [ ] Logging overhead is <5ms per request
- [ ] Log format is machine-readable

### Risk Mitigation
- Test logging performance impact
- Ensure backward compatibility
- Validate JSON format consistency

---

## **Cycle 8B: Log Levels & Categories**
**Duration:** 4-5 hours | **Priority:** High

### Prerequisites
- Cycle 8A completed and tested
- Understanding of log level hierarchy

### Implementation Tasks
- [ ] Implement comprehensive log levels (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
- [ ] Create log categories (audit, performance, security, business, system)
- [ ] Add dynamic log level configuration
- [ ] Implement category-based filtering
- [ ] Create category-specific loggers

### Code Deliverables
```go
// backend/logging/levels.go
type LogLevel int

const (
    TraceLevel LogLevel = iota
    DebugLevel
    InfoLevel
    WarnLevel
    ErrorLevel
    FatalLevel
)

type LogCategory string

const (
    CategoryAudit       LogCategory = "audit"
    CategoryPerformance LogCategory = "performance"
    CategorySecurity    LogCategory = "security"
    CategoryBusiness    LogCategory = "business"
    CategorySystem      LogCategory = "system"
)

type CategoryLogger struct {
    category LogCategory
    logger   *StructuredLogger
    config   *CategoryConfig
}

type CategoryConfig struct {
    Level     LogLevel `yaml:"level"`
    Enabled   bool     `yaml:"enabled"`
    Sampling  float64  `yaml:"sampling"`
    Output    []string `yaml:"output"` // file, stdout, database
}

func (cl *CategoryLogger) Audit(ctx context.Context, action, resource string, metadata map[string]interface{}) {
    if !cl.shouldLog(InfoLevel) {
        return
    }
    
    entry := LogEntry{
        Timestamp: time.Now(),
        Level:     "info",
        Category:  string(CategoryAudit),
        Message:   fmt.Sprintf("Audit: %s on %s", action, resource),
        Metadata: map[string]interface{}{
            "action":   action,
            "resource": resource,
            "audit":    true,
        },
    }
    
    // Merge additional metadata
    for k, v := range metadata {
        entry.Metadata[k] = v
    }
    
    cl.logger.writeLog(entry)
}

func (cl *CategoryLogger) Security(ctx context.Context, event string, severity string, details map[string]interface{}) {
    entry := LogEntry{
        Timestamp: time.Now(),
        Level:     severity,
        Category:  string(CategorySecurity),
        Message:   fmt.Sprintf("Security: %s", event),
        Tags:      []string{"security", severity},
        Metadata:  details,
    }
    
    cl.logger.writeLog(entry)
}
```

### Testing Requirements
- [ ] Unit tests for log levels and categories
- [ ] Test dynamic level configuration
- [ ] Test category filtering
- [ ] Test sampling by category

### Acceptance Criteria
- [ ] Log levels work correctly in hierarchy
- [ ] Categories filter appropriately
- [ ] Dynamic configuration applies without restart
- [ ] Sampling works per category
- [ ] Category-specific loggers function properly

### Risk Mitigation
- Test level changes don't break existing logs
- Validate category filtering accuracy
- Monitor performance impact of filtering

---

## **Cycle 8C: Multiple Output Destinations**
**Duration:** 5-6 hours | **Priority:** High

### Prerequisites
- Cycle 8B completed and tested
- Database integration available

### Implementation Tasks
- [ ] Implement multiple log outputs (file, database, stdout)
- [ ] Add log rotation and compression for files
- [ ] Create database log storage with proper indexing
- [ ] Implement log retention policies
- [ ] Add external log system integration (optional)

### Code Deliverables
```go
// backend/logging/outputs.go
type LogOutput interface {
    Write(entry LogEntry) error
    Close() error
}

type FileOutput struct {
    file     *os.File
    config   *FileOutputConfig
    rotator  *LogRotator
}

type FileOutputConfig struct {
    Path         string `yaml:"path"`
    MaxSize      int64  `yaml:"max_size_mb"`
    MaxFiles     int    `yaml:"max_files"`
    Compress     bool   `yaml:"compress"`
    RotateDaily  bool   `yaml:"rotate_daily"`
}

type DatabaseOutput struct {
    db     *sql.DB
    stmt   *sql.Stmt
    buffer []LogEntry
    config *DatabaseOutputConfig
}

type DatabaseOutputConfig struct {
    BatchSize     int           `yaml:"batch_size"`
    FlushInterval time.Duration `yaml:"flush_interval"`
    TableName     string        `yaml:"table_name"`
}

func (do *DatabaseOutput) Write(entry LogEntry) error {
    do.buffer = append(do.buffer, entry)
    
    if len(do.buffer) >= do.config.BatchSize {
        return do.flush()
    }
    
    return nil
}

func (do *DatabaseOutput) flush() error {
    if len(do.buffer) == 0 {
        return nil
    }
    
    tx, err := do.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    for _, entry := range do.buffer {
        _, err := tx.Stmt(do.stmt).Exec(
            entry.Timestamp,
            entry.Level,
            entry.Service,
            entry.Component,
            entry.RequestID,
            entry.Message,
            entry.Error,
            entry.MetadataJSON(),
        )
        if err != nil {
            return err
        }
    }
    
    if err := tx.Commit(); err != nil {
        return err
    }
    
    do.buffer = do.buffer[:0] // Clear buffer
    return nil
}

type LogRotator struct {
    config     *FileOutputConfig
    currentFile *os.File
    currentSize int64
}

func (lr *LogRotator) shouldRotate() bool {
    return lr.currentSize > lr.config.MaxSize*1024*1024 ||
           (lr.config.RotateDaily && lr.shouldRotateDaily())
}
```

### Testing Requirements
- [ ] Unit tests for each output type
- [ ] Test log rotation functionality
- [ ] Test database batch writing
- [ ] Integration tests with real storage

### Acceptance Criteria
- [ ] Logs write to multiple outputs simultaneously
- [ ] File rotation works correctly
- [ ] Database storage is efficient and searchable
- [ ] Log retention policies are enforced
- [ ] No data loss during rotation or buffering

### Risk Mitigation
- Test rotation under load
- Ensure database performance
- Validate retention policy accuracy

---

## **Cycle 8D: Log Search & Analysis**
**Duration:** 6-7 hours | **Priority:** Medium

### Prerequisites
- Cycle 8C completed and tested
- Understanding of search algorithms

### Implementation Tasks
- [ ] Create log search API with filtering
- [ ] Implement full-text search capabilities
- [ ] Add log aggregation and statistics
- [ ] Create log analysis tools
- [ ] Add log export functionality

### Code Deliverables
```go
// backend/api/logs.go
type LogSearchRequest struct {
    Query     string    `json:"query"`
    Level     []string  `json:"level,omitempty"`
    Category  []string  `json:"category,omitempty"`
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
    UserID    string    `json:"user_id,omitempty"`
    Component string    `json:"component,omitempty"`
    Tags      []string  `json:"tags,omitempty"`
    Limit     int       `json:"limit"`
    Offset    int       `json:"offset"`
}

type LogSearchResponse struct {
    Logs       []LogEntry     `json:"logs"`
    Total      int            `json:"total"`
    Aggregates LogAggregates  `json:"aggregates"`
    Took       time.Duration  `json:"took"`
}

type LogAggregates struct {
    LevelCounts     map[string]int `json:"level_counts"`
    CategoryCounts  map[string]int `json:"category_counts"`
    ComponentCounts map[string]int `json:"component_counts"`
    ErrorCounts     map[string]int `json:"error_counts"`
    TimeHistogram   []TimeSlice    `json:"time_histogram"`
}

type LogSearchService struct {
    db       *sql.DB
    indexer  *LogIndexer
    analyzer *LogAnalyzer
}

func (lss *LogSearchService) Search(req LogSearchRequest) (*LogSearchResponse, error) {
    start := time.Now()
    
    query := lss.buildQuery(req)
    rows, err := lss.db.Query(query, lss.buildArgs(req)...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var logs []LogEntry
    for rows.Next() {
        var log LogEntry
        err := rows.Scan(
            &log.Timestamp,
            &log.Level,
            &log.Service,
            &log.Component,
            &log.Message,
            &log.Error,
            // ... other fields
        )
        if err != nil {
            return nil, err
        }
        logs = append(logs, log)
    }
    
    aggregates := lss.analyzer.GenerateAggregates(logs, req)
    
    return &LogSearchResponse{
        Logs:       logs,
        Total:      len(logs),
        Aggregates: aggregates,
        Took:       time.Since(start),
    }, nil
}

func (lss *LogSearchService) buildQuery(req LogSearchRequest) string {
    query := "SELECT * FROM logs WHERE timestamp BETWEEN ? AND ?"
    
    if len(req.Level) > 0 {
        query += " AND level IN (" + lss.placeholders(len(req.Level)) + ")"
    }
    
    if req.Query != "" {
        query += " AND (message LIKE ? OR error LIKE ?)"
    }
    
    query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
    return query
}
```

### Testing Requirements
- [ ] Unit tests for search functionality
- [ ] Test search performance with large datasets
- [ ] Test aggregation accuracy
- [ ] Integration tests for various search criteria

### Acceptance Criteria
- [ ] Search returns accurate results within 2 seconds
- [ ] Full-text search works correctly
- [ ] Aggregations provide useful insights
- [ ] Export functionality works for various formats
- [ ] Search handles large result sets efficiently

### Risk Mitigation
- Optimize database queries for search
- Index appropriate columns
- Test with realistic data volumes

---

## **Cycle 8E: Enhanced Frontend Log Viewer**
**Duration:** 6-8 hours | **Priority:** Medium

### Prerequisites
- Cycle 8D completed and tested
- Frontend development environment ready

### Implementation Tasks
- [ ] Upgrade existing Logs component with advanced features
- [ ] Add real-time log streaming via WebSocket
- [ ] Implement advanced filtering interface
- [ ] Create log details expansion view
- [ ] Add log export and sharing functionality

### Code Deliverables
```typescript
// frontend/src/components/logs/LogViewer.tsx
interface LogViewerProps {
    initialFilters?: LogSearchFilters;
    realTime?: boolean;
    height?: string;
}

interface LogSearchFilters {
    query: string;
    levels: string[];
    categories: string[];
    timeRange: TimeRange;
    components: string[];
    tags: string[];
}

const LogViewer: React.FC<LogViewerProps> = ({ 
    initialFilters, 
    realTime = false, 
    height = "600px" 
}) => {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [filters, setFilters] = useState<LogSearchFilters>(initialFilters || defaultFilters);
    const [loading, setLoading] = useState(false);
    const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null);
    
    // Real-time log streaming
    useEffect(() => {
        if (!realTime) return;
        
        const ws = new WebSocket(`ws://localhost:8080/ws/logs`);
        ws.onmessage = (event) => {
            const newLog = JSON.parse(event.data);
            if (matchesFilters(newLog, filters)) {
                setLogs(prev => [newLog, ...prev.slice(0, 999)]); // Keep last 1000
            }
        };
        
        return () => ws.close();
    }, [realTime, filters]);
    
    const searchLogs = async () => {
        setLoading(true);
        try {
            const response = await fetch('/api/logs/search', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(filters),
            });
            const data = await response.json();
            setLogs(data.logs);
        } finally {
            setLoading(false);
        }
    };
    
    return (
        <div className="log-viewer" style={{ height }}>
            <LogFilters 
                filters={filters} 
                onChange={setFilters}
                onSearch={searchLogs}
            />
            <LogTable 
                logs={logs}
                loading={loading}
                onSelectLog={setSelectedLog}
            />
            {selectedLog && (
                <LogDetailModal 
                    log={selectedLog}
                    onClose={() => setSelectedLog(null)}
                />
            )}
        </div>
    );
};

// frontend/src/components/logs/LogTable.tsx
const LogTable: React.FC<LogTableProps> = ({ logs, loading, onSelectLog }) => {
    const virtualization = useVirtualization(logs, { itemHeight: 40 });
    
    return (
        <div className="log-table">
            {loading && <LoadingSpinner />}
            <div className="log-table-header">
                <div className="col-timestamp">Timestamp</div>
                <div className="col-level">Level</div>
                <div className="col-component">Component</div>
                <div className="col-message">Message</div>
            </div>
            <div className="log-table-body" style={{ height: 400, overflow: 'auto' }}>
                {virtualization.virtualItems.map((virtualItem) => {
                    const log = logs[virtualItem.index];
                    return (
                        <LogRow
                            key={virtualItem.key}
                            log={log}
                            style={{
                                position: 'absolute',
                                top: 0,
                                left: 0,
                                width: '100%',
                                height: `${virtualItem.size}px`,
                                transform: `translateY(${virtualItem.start}px)`,
                            }}
                            onClick={() => onSelectLog(log)}
                        />
                    );
                })}
            </div>
        </div>
    );
};
```

### Testing Requirements
- [ ] Unit tests for log viewer components
- [ ] Test real-time log streaming
- [ ] Test search and filtering functionality
- [ ] Test virtualization with large datasets

### Acceptance Criteria
- [ ] Log viewer handles 10,000+ logs smoothly
- [ ] Real-time streaming works without lag
- [ ] Advanced filtering provides accurate results
- [ ] Log details view shows all relevant information
- [ ] Export functionality works correctly

### Risk Mitigation
- Test with large log volumes
- Ensure real-time updates don't overwhelm UI
- Optimize virtualization for performance

---

## **Integration Testing**
**Duration:** 3-4 hours

### Comprehensive Logging Testing
- [ ] End-to-end logging flow testing
- [ ] Performance testing with high log volume
- [ ] Search functionality testing
- [ ] Real-time streaming testing
- [ ] Multi-output reliability testing

### Success Metrics
- [ ] Logging overhead <5ms per request
- [ ] Search completes within 2 seconds for 100k logs
- [ ] Real-time streaming handles 1000+ logs/minute
- [ ] Zero log loss during rotation or restart
- [ ] Search accuracy >99% for structured queries

---

## **Performance Benchmarks**
- **Logging Overhead**: <5ms per structured log entry
- **Search Performance**: <2 seconds for 100k log entries
- **Real-time Latency**: <100ms from log generation to UI
- **Storage Efficiency**: 70% compression for rotated logs
- **Memory Usage**: <100MB for log buffers

---

## **Rollback Plan**
If any cycle fails:
1. Fall back to basic file logging
2. Disable problematic output destinations
3. Use simple grep-based search temporarily
4. Disable real-time features if performance issues