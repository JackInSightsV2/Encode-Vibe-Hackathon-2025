package metrics

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// BucketResolution defines time bucket resolutions
type BucketResolution string

const (
	Resolution1Min  BucketResolution = "1m"
	Resolution5Min  BucketResolution = "5m"
	Resolution1Hour BucketResolution = "1h"
	Resolution1Day  BucketResolution = "1d"
)

// ResolutionToDuration converts resolution to time.Duration
func (r BucketResolution) Duration() time.Duration {
	switch r {
	case Resolution1Min:
		return time.Minute
	case Resolution5Min:
		return 5 * time.Minute
	case Resolution1Hour:
		return time.Hour
	case Resolution1Day:
		return 24 * time.Hour
	default:
		return time.Minute
	}
}

// TimeSeriesStorage extends storage with time-series capabilities
type TimeSeriesStorage struct {
	// Existing storage for compatibility
	rawStorage MetricsStorage
	
	// Time-series buckets organized by resolution
	buckets map[BucketResolution]*TimeBucketMap
	
	// Configuration
	config    TimeSeriesConfig
	mutex     sync.RWMutex
	startTime time.Time
	
	// Aggregation control
	aggregator *Aggregator
	stopChan   chan struct{}
}

// TimeBucketMap holds buckets for a specific resolution
type TimeBucketMap struct {
	buckets    map[int64]*TimeBucket // Key is truncated timestamp
	resolution BucketResolution
	retention  time.Duration
	mutex      sync.RWMutex
}

// TimeBucket represents aggregated metrics for a time period
type TimeBucket struct {
	Timestamp time.Time                   `json:"timestamp"`
	Metrics   map[string]*AggregatedValue `json:"metrics"`
	Count     int64                       `json:"count"`
}

// AggregatedValue holds statistical aggregations of metric values
type AggregatedValue struct {
	Sum     float64 `json:"sum"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Count   int64   `json:"count"`
	Average float64 `json:"average"`
	Latest  float64 `json:"latest"`
}


// TimeSeriesQuery represents a query for time-series data
type TimeSeriesQuery struct {
	MetricNames []string         `json:"metric_names,omitempty"`
	Resolution  BucketResolution `json:"resolution"`
	StartTime   time.Time        `json:"start_time"`
	EndTime     time.Time        `json:"end_time"`
	Aggregation string           `json:"aggregation,omitempty"` // avg, min, max, sum, count
}

// TimeSeriesResult represents time-series query results
type TimeSeriesResult struct {
	Resolution BucketResolution         `json:"resolution"`
	TimeRange  TimeRange                `json:"time_range"`
	Series     map[string][]DataPoint   `json:"series"`
	Metadata   TimeSeriesMetadata       `json:"metadata"`
}

// DataPoint represents a single point in time-series data
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TimeSeriesMetadata provides information about the time-series data
type TimeSeriesMetadata struct {
	TotalPoints    int           `json:"total_points"`
	ActualRange    TimeRange     `json:"actual_range"`
	BucketsQueried int           `json:"buckets_queried"`
	QueryDuration  time.Duration `json:"query_duration"`
}

// NewTimeSeriesStorage creates a new time-series storage instance
func NewTimeSeriesStorage(rawStorage MetricsStorage, config TimeSeriesConfig) *TimeSeriesStorage {
	ts := &TimeSeriesStorage{
		rawStorage: rawStorage,
		buckets:    make(map[BucketResolution]*TimeBucketMap),
		config:     config,
		startTime:  time.Now(),
		stopChan:   make(chan struct{}),
	}
	
	// Initialize bucket maps for each resolution
	resolutions := []BucketResolution{Resolution1Min, Resolution5Min, Resolution1Hour, Resolution1Day}
	for _, resolution := range resolutions {
		retention := ts.getRetentionForResolution(resolution)
		ts.buckets[resolution] = &TimeBucketMap{
			buckets:    make(map[int64]*TimeBucket),
			resolution: resolution,
			retention:  retention,
		}
	}
	
	// Start aggregator if enabled
	if config.Enabled {
		ts.aggregator = NewAggregator(ts, config)
		go ts.aggregator.Start()
	}
	
	return ts
}

// Store implements the MetricsStorage interface
func (ts *TimeSeriesStorage) Store(metric Metric) error {
	// Store in raw storage for compatibility
	if err := ts.rawStorage.Store(metric); err != nil {
		return err
	}
	
	// Store in time-series buckets if enabled
	if ts.config.Enabled {
		return ts.storeInTimeSeries(metric)
	}
	
	return nil
}

// Get implements the MetricsStorage interface
func (ts *TimeSeriesStorage) Get(query MetricsQuery) ([]Metric, error) {
	return ts.rawStorage.Get(query)
}

// GetSummary implements the MetricsStorage interface
func (ts *TimeSeriesStorage) GetSummary(timeRange TimeRange) (*MetricsSummary, error) {
	return ts.rawStorage.GetSummary(timeRange)
}

// Cleanup implements the MetricsStorage interface
func (ts *TimeSeriesStorage) Cleanup() error {
	// Cleanup raw storage
	if err := ts.rawStorage.Cleanup(); err != nil {
		return err
	}
	
	// Cleanup time-series buckets
	return ts.cleanupTimeSeries()
}

// Close implements the MetricsStorage interface
func (ts *TimeSeriesStorage) Close() error {
	// Stop aggregator
	if ts.aggregator != nil {
		close(ts.stopChan)
	}
	
	return ts.rawStorage.Close()
}

// GetTimeSeries retrieves time-series data based on query
func (ts *TimeSeriesStorage) GetTimeSeries(query TimeSeriesQuery) (*TimeSeriesResult, error) {
	startTime := time.Now()
	
	ts.mutex.RLock()
	bucketMap, exists := ts.buckets[query.Resolution]
	ts.mutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("resolution %s not supported", query.Resolution)
	}
	
	result := &TimeSeriesResult{
		Resolution: query.Resolution,
		TimeRange:  TimeRange{Start: query.StartTime, End: query.EndTime},
		Series:     make(map[string][]DataPoint),
		Metadata: TimeSeriesMetadata{
			QueryDuration: time.Since(startTime),
		},
	}
	
	// Get buckets in time range
	buckets := ts.getBucketsInRange(bucketMap, query.StartTime, query.EndTime)
	result.Metadata.BucketsQueried = len(buckets)
	
	// Extract time-series data
	for _, bucket := range buckets {
		for metricName, aggValue := range bucket.Metrics {
			// Filter by metric names if specified
			if len(query.MetricNames) > 0 && !contains(query.MetricNames, metricName) {
				continue
			}
			
			// Get value based on aggregation type
			value := ts.getValueFromAggregation(aggValue, query.Aggregation)
			
			// Initialize series if not exists
			if _, exists := result.Series[metricName]; !exists {
				result.Series[metricName] = make([]DataPoint, 0)
			}
			
			// Add data point
			result.Series[metricName] = append(result.Series[metricName], DataPoint{
				Timestamp: bucket.Timestamp,
				Value:     value,
			})
		}
	}
	
	// Sort data points by timestamp
	for metricName := range result.Series {
		sort.Slice(result.Series[metricName], func(i, j int) bool {
			return result.Series[metricName][i].Timestamp.Before(result.Series[metricName][j].Timestamp)
		})
		result.Metadata.TotalPoints += len(result.Series[metricName])
	}
	
	// Update actual range based on data
	if len(buckets) > 0 {
		result.Metadata.ActualRange = TimeRange{
			Start: buckets[0].Timestamp,
			End:   buckets[len(buckets)-1].Timestamp,
		}
	}
	
	result.Metadata.QueryDuration = time.Since(startTime)
	return result, nil
}

// storeInTimeSeries stores a metric in time-series buckets
func (ts *TimeSeriesStorage) storeInTimeSeries(metric Metric) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	// Store in 1-minute buckets immediately
	bucketMap := ts.buckets[Resolution1Min]
	return ts.addToBucket(bucketMap, metric, metric.Timestamp)
}

// addToBucket adds a metric to the appropriate bucket
func (ts *TimeSeriesStorage) addToBucket(bucketMap *TimeBucketMap, metric Metric, timestamp time.Time) error {
	bucketMap.mutex.Lock()
	defer bucketMap.mutex.Unlock()
	
	// Calculate bucket timestamp (truncated to resolution)
	bucketTime := ts.truncateToResolution(timestamp, bucketMap.resolution)
	bucketKey := bucketTime.Unix()
	
	// Get or create bucket
	bucket, exists := bucketMap.buckets[bucketKey]
	if !exists {
		bucket = &TimeBucket{
			Timestamp: bucketTime,
			Metrics:   make(map[string]*AggregatedValue),
			Count:     0,
		}
		bucketMap.buckets[bucketKey] = bucket
	}
	
	// Get or create aggregated value for metric
	aggValue, exists := bucket.Metrics[metric.Name]
	if !exists {
		aggValue = &AggregatedValue{
			Sum:     0,
			Min:     metric.Value,
			Max:     metric.Value,
			Count:   0,
			Average: 0,
			Latest:  metric.Value,
		}
		bucket.Metrics[metric.Name] = aggValue
	}
	
	// Update aggregated value
	aggValue.Sum += metric.Value
	aggValue.Count++
	aggValue.Average = aggValue.Sum / float64(aggValue.Count)
	aggValue.Latest = metric.Value
	
	if metric.Value < aggValue.Min {
		aggValue.Min = metric.Value
	}
	if metric.Value > aggValue.Max {
		aggValue.Max = metric.Value
	}
	
	bucket.Count++
	
	return nil
}

// truncateToResolution truncates timestamp to bucket resolution
func (ts *TimeSeriesStorage) truncateToResolution(timestamp time.Time, resolution BucketResolution) time.Time {
	switch resolution {
	case Resolution1Min:
		return timestamp.Truncate(time.Minute)
	case Resolution5Min:
		return timestamp.Truncate(5 * time.Minute)
	case Resolution1Hour:
		return timestamp.Truncate(time.Hour)
	case Resolution1Day:
		year, month, day := timestamp.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, timestamp.Location())
	default:
		return timestamp.Truncate(time.Minute)
	}
}

// getBucketsInRange retrieves buckets within a time range
func (ts *TimeSeriesStorage) getBucketsInRange(bucketMap *TimeBucketMap, start, end time.Time) []*TimeBucket {
	bucketMap.mutex.RLock()
	defer bucketMap.mutex.RUnlock()
	
	var buckets []*TimeBucket
	
	for _, bucket := range bucketMap.buckets {
		if bucket.Timestamp.After(start) && bucket.Timestamp.Before(end) || 
		   bucket.Timestamp.Equal(start) || bucket.Timestamp.Equal(end) {
			buckets = append(buckets, bucket)
		}
	}
	
	// Sort by timestamp
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Timestamp.Before(buckets[j].Timestamp)
	})
	
	return buckets
}

// getValueFromAggregation extracts value based on aggregation type
func (ts *TimeSeriesStorage) getValueFromAggregation(aggValue *AggregatedValue, aggregation string) float64 {
	switch aggregation {
	case "min":
		return aggValue.Min
	case "max":
		return aggValue.Max
	case "sum":
		return aggValue.Sum
	case "count":
		return float64(aggValue.Count)
	case "latest":
		return aggValue.Latest
	default: // "avg" or empty
		return aggValue.Average
	}
}

// getRetentionForResolution gets retention duration for a resolution
func (ts *TimeSeriesStorage) getRetentionForResolution(resolution BucketResolution) time.Duration {
	// Default retention periods
	defaults := map[BucketResolution]time.Duration{
		Resolution1Min:  6 * time.Hour,
		Resolution5Min:  72 * time.Hour,    // 3 days
		Resolution1Hour: 720 * time.Hour,   // 30 days
		Resolution1Day:  8760 * time.Hour,  // 1 year
	}
	
	// Check configuration for custom retention
	for _, config := range ts.config.Resolutions {
		if BucketResolution(config.Resolution) == resolution {
			return time.Duration(config.RetentionHours) * time.Hour
		}
	}
	
	// Return default
	if retention, exists := defaults[resolution]; exists {
		return retention
	}
	
	return 24 * time.Hour
}

// cleanupTimeSeries removes old buckets based on retention policies
func (ts *TimeSeriesStorage) cleanupTimeSeries() error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	now := time.Now()
	
	for _, bucketMap := range ts.buckets {
		bucketMap.mutex.Lock()
		
		cutoff := now.Add(-bucketMap.retention)
		for key, bucket := range bucketMap.buckets {
			if bucket.Timestamp.Before(cutoff) {
				delete(bucketMap.buckets, key)
			}
		}
		
		bucketMap.mutex.Unlock()
	}
	
	return nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}