package metrics

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeSeriesStorage_Store(t *testing.T) {
	// Create test configuration
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 60,
		MaxBucketsPerResolution: 1000,
		Resolutions: []ResolutionConfig{
			{Resolution: "1m", RetentionHours: 1},
			{Resolution: "5m", RetentionHours: 24},
		},
	}

	// Create in-memory storage and time-series storage
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Test storing a metric
	metric := Metric{
		ID:        "test-1",
		Name:      "test_metric",
		Value:     42.5,
		Timestamp: time.Now(),
		Type:      MetricTypeGauge,
		Tags:      map[string]string{"source": "test"},
	}

	err := ts.Store(metric)
	if err != nil {
		t.Fatalf("Failed to store metric: %v", err)
	}

	// Verify metric was stored in raw storage
	query := MetricsQuery{
		Names:     []string{"test_metric"},
		StartTime: metric.Timestamp.Add(-time.Minute),
		EndTime:   metric.Timestamp.Add(time.Minute),
	}

	metrics, err := ts.Get(query)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Value != 42.5 {
		t.Errorf("Expected value 42.5, got %f", metrics[0].Value)
	}
}

func TestTimeSeriesStorage_GetTimeSeries(t *testing.T) {
	// Create test configuration
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 60,
		MaxBucketsPerResolution: 1000,
		Resolutions: []ResolutionConfig{
			{Resolution: "1m", RetentionHours: 1},
		},
	}

	// Create storage
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Store multiple metrics over time
	baseTime := time.Now().Truncate(time.Minute)
	for i := 0; i < 5; i++ {
		metric := Metric{
			ID:        fmt.Sprintf("test-%d", i),
			Name:      "cpu_usage",
			Value:     float64(10 + i*5), // Values: 10, 15, 20, 25, 30
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Type:      MetricTypeGauge,
		}

		err := ts.Store(metric)
		if err != nil {
			t.Fatalf("Failed to store metric %d: %v", i, err)
		}
	}

	// Query time-series data
	query := TimeSeriesQuery{
		MetricNames: []string{"cpu_usage"},
		Resolution:  Resolution1Min,
		StartTime:   baseTime,
		EndTime:     baseTime.Add(5 * time.Minute),
		Aggregation: "avg",
	}

	result, err := ts.GetTimeSeries(query)
	if err != nil {
		t.Fatalf("Failed to get time-series data: %v", err)
	}

	// Verify results
	if result.Resolution != Resolution1Min {
		t.Errorf("Expected resolution %s, got %s", Resolution1Min, result.Resolution)
	}

	series, exists := result.Series["cpu_usage"]
	if !exists {
		t.Fatal("Expected cpu_usage series not found")
	}

	if len(series) != 5 {
		t.Fatalf("Expected 5 data points, got %d", len(series))
	}

	// Verify data point values
	expectedValues := []float64{10, 15, 20, 25, 30}
	for i, point := range series {
		if point.Value != expectedValues[i] {
			t.Errorf("Expected value %f at index %d, got %f", expectedValues[i], i, point.Value)
		}
	}
}

func TestBucketResolution_Duration(t *testing.T) {
	tests := []struct {
		resolution BucketResolution
		expected   time.Duration
	}{
		{Resolution1Min, time.Minute},
		{Resolution5Min, 5 * time.Minute},
		{Resolution1Hour, time.Hour},
		{Resolution1Day, 24 * time.Hour},
	}

	for _, test := range tests {
		actual := test.resolution.Duration()
		if actual != test.expected {
			t.Errorf("Resolution %s: expected %v, got %v", test.resolution, test.expected, actual)
		}
	}
}

func TestTimeSeriesStorage_TruncateToResolution(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Test timestamp: 2024-01-15 14:35:42
	testTime := time.Date(2024, 1, 15, 14, 35, 42, 0, time.UTC)

	tests := []struct {
		resolution BucketResolution
		expected   time.Time
	}{
		{
			Resolution1Min,
			time.Date(2024, 1, 15, 14, 35, 0, 0, time.UTC), // Truncated to minute
		},
		{
			Resolution5Min,
			time.Date(2024, 1, 15, 14, 35, 0, 0, time.UTC), // Truncated to 5-minute boundary
		},
		{
			Resolution1Hour,
			time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC), // Truncated to hour
		},
		{
			Resolution1Day,
			time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), // Truncated to day
		},
	}

	for _, test := range tests {
		actual := ts.truncateToResolution(testTime, test.resolution)
		if !actual.Equal(test.expected) {
			t.Errorf("Resolution %s: expected %v, got %v", test.resolution, test.expected, actual)
		}
	}
}

func TestAggregatedValue_UpdateLogic(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Create a bucket and add multiple values
	bucketMap := ts.buckets[Resolution1Min]
	
	baseTime := time.Now().Truncate(time.Minute)
	
	// Add first metric
	metric1 := Metric{
		Name:      "test_metric",
		Value:     10.0,
		Timestamp: baseTime,
	}
	err := ts.addToBucket(bucketMap, metric1, baseTime)
	if err != nil {
		t.Fatalf("Failed to add first metric: %v", err)
	}

	// Add second metric to same bucket
	metric2 := Metric{
		Name:      "test_metric",
		Value:     20.0,
		Timestamp: baseTime.Add(30 * time.Second), // Same minute bucket
	}
	err = ts.addToBucket(bucketMap, metric2, baseTime)
	if err != nil {
		t.Fatalf("Failed to add second metric: %v", err)
	}

	// Verify aggregated values
	bucketMap.mutex.RLock()
	bucket := bucketMap.buckets[baseTime.Unix()]
	bucketMap.mutex.RUnlock()

	if bucket == nil {
		t.Fatal("Expected bucket not found")
	}

	aggValue := bucket.Metrics["test_metric"]
	if aggValue == nil {
		t.Fatal("Expected aggregated value not found")
	}

	// Verify aggregation calculations
	if aggValue.Count != 2 {
		t.Errorf("Expected count 2, got %d", aggValue.Count)
	}

	if aggValue.Sum != 30.0 {
		t.Errorf("Expected sum 30.0, got %f", aggValue.Sum)
	}

	if aggValue.Average != 15.0 {
		t.Errorf("Expected average 15.0, got %f", aggValue.Average)
	}

	if aggValue.Min != 10.0 {
		t.Errorf("Expected min 10.0, got %f", aggValue.Min)
	}

	if aggValue.Max != 20.0 {
		t.Errorf("Expected max 20.0, got %f", aggValue.Max)
	}

	if aggValue.Latest != 20.0 {
		t.Errorf("Expected latest 20.0, got %f", aggValue.Latest)
	}
}

func TestTimeSeriesStorage_RetentionPolicy(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled: true,
		Resolutions: []ResolutionConfig{
			{Resolution: "1m", RetentionHours: 1}, // Very short retention for testing
		},
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Store metrics with old timestamps
	oldTime := time.Now().Add(-2 * time.Hour) // Beyond retention
	recentTime := time.Now().Add(-30 * time.Minute) // Within retention

	oldMetric := Metric{
		Name:      "old_metric",
		Value:     100.0,
		Timestamp: oldTime,
	}

	recentMetric := Metric{
		Name:      "recent_metric",
		Value:     200.0,
		Timestamp: recentTime,
	}

	// Store both metrics
	ts.Store(oldMetric)
	ts.Store(recentMetric)

	// Perform cleanup
	err := ts.cleanupTimeSeries()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Query all data to see what remains
	query := TimeSeriesQuery{
		Resolution: Resolution1Min,
		StartTime:  oldTime.Add(-time.Hour),
		EndTime:    time.Now(),
	}

	result, err := ts.GetTimeSeries(query)
	if err != nil {
		t.Fatalf("Failed to query after cleanup: %v", err)
	}

	// Should only have recent data
	if len(result.Series) == 0 {
		t.Error("Expected some series data after cleanup")
	}

	// Old metric should be gone, recent should remain
	if _, hasOld := result.Series["old_metric"]; hasOld {
		t.Error("Expected old metric to be cleaned up")
	}

	if _, hasRecent := result.Series["recent_metric"]; !hasRecent {
		t.Error("Expected recent metric to remain after cleanup")
	}
}

func TestTimeSeriesStorage_GetValueFromAggregation(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	aggValue := &AggregatedValue{
		Sum:     100.0,
		Min:     5.0,
		Max:     25.0,
		Count:   4,
		Average: 25.0,
		Latest:  20.0,
	}

	tests := []struct {
		aggregation string
		expected    float64
	}{
		{"sum", 100.0},
		{"min", 5.0},
		{"max", 25.0},
		{"count", 4.0},
		{"avg", 25.0},
		{"average", 25.0}, // Default case
		{"latest", 20.0},
		{"", 25.0}, // Default case
	}

	for _, test := range tests {
		actual := ts.getValueFromAggregation(aggValue, test.aggregation)
		if actual != test.expected {
			t.Errorf("Aggregation %s: expected %f, got %f", test.aggregation, test.expected, actual)
		}
	}
}

func TestTimeSeriesStorage_ConcurrentAccess(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 60,
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Test concurrent writes
	numGoroutines := 10
	metricsPerGoroutine := 100
	done := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < metricsPerGoroutine; i++ {
				metric := Metric{
					ID:        fmt.Sprintf("metric-%d-%d", goroutineID, i),
					Name:      "concurrent_test",
					Value:     float64(goroutineID*1000 + i),
					Timestamp: time.Now(),
					Type:      MetricTypeCounter,
				}

				err := ts.Store(metric)
				if err != nil {
					t.Errorf("Goroutine %d failed to store metric %d: %v", goroutineID, i, err)
				}
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify we can query the data without errors
	query := TimeSeriesQuery{
		MetricNames: []string{"concurrent_test"},
		Resolution:  Resolution1Min,
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     time.Now(),
	}

	result, err := ts.GetTimeSeries(query)
	if err != nil {
		t.Fatalf("Failed to query after concurrent writes: %v", err)
	}

	if len(result.Series) == 0 {
		t.Error("Expected time-series data after concurrent writes")
	}

	series, exists := result.Series["concurrent_test"]
	if !exists {
		t.Error("Expected concurrent_test series")
	}

	if len(series) == 0 {
		t.Error("Expected data points in concurrent_test series")
	}

	t.Logf("Successfully stored and retrieved %d data points from concurrent writes", len(series))
}