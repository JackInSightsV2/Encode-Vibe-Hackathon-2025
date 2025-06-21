package metrics

import (
	"time"
)

// Metric represents a single metric data point
type Metric struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Tags      map[string]string      `json:"tags"`
	Timestamp time.Time              `json:"timestamp"`
	Type      MetricType             `json:"type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

// MetricsQuery represents a query for retrieving metrics
type MetricsQuery struct {
	Names     []string          `json:"names,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
}

// MetricsSummary provides aggregated metrics data
type MetricsSummary struct {
	TotalRequests        int64                  `json:"total_requests"`
	RequestsPerSecond    float64               `json:"requests_per_second"`
	AverageResponseTime  float64               `json:"average_response_time"`
	ErrorRate           float64               `json:"error_rate"`
	ModerationBlocked    int64                 `json:"moderation_blocked"`
	PIIDetections       int64                 `json:"pii_detections"`
	SystemHealth        SystemHealthMetrics   `json:"system_health"`
	TopEndpoints        []EndpointMetrics     `json:"top_endpoints"`
	TimeRange           TimeRange             `json:"time_range"`
}

// SystemHealthMetrics represents system health data
type SystemHealthMetrics struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	GoroutineCount  int     `json:"goroutine_count"`
	DBConnections   int     `json:"db_connections"`
	CacheHitRate    float64 `json:"cache_hit_rate"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
}

// EndpointMetrics represents metrics for a specific endpoint
type EndpointMetrics struct {
	Path         string  `json:"path"`
	Method       string  `json:"method"`
	RequestCount int64   `json:"request_count"`
	AvgDuration  float64 `json:"avg_duration"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
}

// TimeRange represents a time range for metrics queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled       bool                  `yaml:"enabled" json:"enabled"`
	Storage       StorageConfig         `yaml:"storage" json:"storage"`
	Supabase      SupabaseConfig        `yaml:"supabase" json:"supabase"`
	Collection    CollectionConfig      `yaml:"collection" json:"collection"`
	TimeSeries    TimeSeriesConfig      `yaml:"timeseries" json:"timeseries"`
	SystemMetrics SystemMetricsConfig   `yaml:"system_metrics" json:"system_metrics"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type                  string `yaml:"type" json:"type"` // "memory" or "supabase"
	RetentionHours        int    `yaml:"retention_hours" json:"retention_hours"`
	CleanupIntervalMinutes int   `yaml:"cleanup_interval_minutes" json:"cleanup_interval_minutes"`
	MaxMemoryMB           int    `yaml:"max_memory_mb" json:"max_memory_mb"`
}

// SupabaseConfig represents Supabase configuration
type SupabaseConfig struct {
	URL   string `yaml:"url" json:"url"`
	Key   string `yaml:"key" json:"key"`
	Table string `yaml:"table" json:"table"`
}

// CollectionConfig represents what metrics to collect
type CollectionConfig struct {
	HTTPRequests     bool `yaml:"http_requests" json:"http_requests"`
	ModerationEvents bool `yaml:"moderation_events" json:"moderation_events"`
	SystemHealth     bool `yaml:"system_health" json:"system_health"`
	PIIDetection     bool `yaml:"pii_detection" json:"pii_detection"`
}

// HTTPMetrics represents HTTP request metrics
type HTTPMetrics struct {
	Path       string        `json:"path"`
	Method     string        `json:"method"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	UserAgent  string        `json:"user_agent,omitempty"`
	IPAddress  string        `json:"ip_address,omitempty"`
	Error      string        `json:"error,omitempty"`
}

// ModerationMetrics represents moderation event metrics
type ModerationMetrics struct {
	LayerName     string  `json:"layer_name"`
	Score         float64 `json:"score"`
	Confidence    float64 `json:"confidence"`
	Blocked       bool    `json:"blocked"`
	Category      string  `json:"category"`
	Action        string  `json:"action"`
	Duration      time.Duration `json:"duration"`
	UserID        string  `json:"user_id,omitempty"`
	ContentLength int     `json:"content_length"`
}

// PIIMetrics represents PII detection metrics
type PIIMetrics struct {
	DetectedTypes   []string `json:"detected_types"`
	MatchCount      int      `json:"match_count"`
	MaskingEnabled  bool     `json:"masking_enabled"`
	Confidence      float64  `json:"confidence"`
	Action          string   `json:"action"`
	UserID          string   `json:"user_id,omitempty"`
	ContentLength   int      `json:"content_length"`
}

// TimeSeriesConfig represents time-series configuration
type TimeSeriesConfig struct {
	Enabled                   bool                    `yaml:"enabled" json:"enabled"`
	Resolutions              []ResolutionConfig      `yaml:"resolutions" json:"resolutions"`
	AggregationIntervalSecs  int                     `yaml:"aggregation_interval_seconds" json:"aggregation_interval_seconds"`
	MaxBucketsPerResolution  int                     `yaml:"max_buckets_per_resolution" json:"max_buckets_per_resolution"`
}

// ResolutionConfig defines retention for each resolution
type ResolutionConfig struct {
	Resolution     string `yaml:"resolution" json:"resolution"`
	RetentionHours int    `yaml:"retention_hours" json:"retention_hours"`
}

// SystemMetricsConfig represents configuration for system metrics
type SystemMetricsConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	CollectionInterval time.Duration `yaml:"collection_interval" json:"collection_interval"`
	CPUMonitoring      bool          `yaml:"cpu_monitoring" json:"cpu_monitoring"`
	MemoryMonitoring   bool          `yaml:"memory_monitoring" json:"memory_monitoring"`
	ProviderHealth     []string      `yaml:"provider_health" json:"provider_health"`
}

// PIIAnalyticsData represents aggregated PII detection analytics
type PIIAnalyticsData struct {
	TotalScanned        int64                            `json:"total_scanned"`
	PIIDetected         int64                            `json:"pii_detected"`
	DetectionRate       float64                          `json:"detection_rate"`
	FalsePositives      int64                            `json:"false_positives"`
	AccuracyRate        float64                          `json:"accuracy_rate"`
	HourlyTrends        []map[string]interface{}         `json:"hourly_trends"`
	TypeBreakdown       map[string]map[string]interface{} `json:"type_breakdown"`
	HighRiskEvents      int64                            `json:"high_risk_events"`
	MediumRiskEvents    int64                            `json:"medium_risk_events"`
	LowRiskEvents       int64                            `json:"low_risk_events"`
	PreventedExposures  int64                            `json:"prevented_exposures"`
}