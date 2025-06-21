package metrics

import (
	"testing"
	"time"
)

func TestAggregator_StartStop(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 1, // 1 second for testing
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	aggregator := ts.aggregator
	if aggregator == nil {
		t.Fatal("Expected aggregator to be created")
	}

	// Test start
	aggregator.Start()
	
	status := aggregator.GetStatus()
	if !status.Running {
		t.Error("Expected aggregator to be running after start")
	}

	// Test stop
	aggregator.Stop()
	
	status = aggregator.GetStatus()
	if status.Running {
		t.Error("Expected aggregator to be stopped after stop")
	}
}

func TestAggregator_ForceAggregation(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 60,
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Start the aggregator first
	ts.aggregator.Start()
	defer ts.aggregator.Stop()

	// Store some 1-minute resolution data
	baseTime := time.Now().Truncate(time.Minute).Add(-10 * time.Minute)
	
	// Add metrics to 1-minute buckets
	for i := 0; i < 5; i++ {
		metric := Metric{
			Name:      "test_metric",
			Value:     float64(i * 10),
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
		}
		
		ts.Store(metric)
	}

	// Force aggregation
	err := ts.aggregator.ForceAggregation()
	if err != nil {
		t.Fatalf("Failed to force aggregation: %v", err)
	}

	// Check if 5-minute buckets were created
	fiveMinBuckets := ts.buckets[Resolution5Min]
	fiveMinBuckets.mutex.RLock()
	bucketCount := len(fiveMinBuckets.buckets)
	fiveMinBuckets.mutex.RUnlock()

	if bucketCount == 0 {
		t.Error("Expected 5-minute buckets to be created after aggregation")
	}

	t.Logf("Created %d five-minute buckets after aggregation", bucketCount)
}

func TestAggregator_GroupBucketsByTargetResolution(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	aggregator := ts.aggregator

	// Create source buckets at 1-minute intervals - use aligned time
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Start at exactly 12:00
	var sourceBuckets []*TimeBucket

	for i := 0; i < 10; i++ {
		bucket := &TimeBucket{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Metrics:   map[string]*AggregatedValue{},
			Count:     1,
		}
		sourceBuckets = append(sourceBuckets, bucket)
	}

	// Group by 5-minute resolution
	groups := aggregator.groupBucketsByTargetResolution(sourceBuckets, Resolution5Min)

	// Should have 2 groups (12:00-12:04 and 12:05-12:09)
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
		for targetTime, buckets := range groups {
			t.Logf("Group %v has %d buckets", targetTime, len(buckets))
		}
	}

	// Verify each group has 5 buckets
	expectedBucketsPerGroup := 5
	totalBuckets := 0
	for targetTime, buckets := range groups {
		totalBuckets += len(buckets)
		if len(buckets) != expectedBucketsPerGroup {
			t.Errorf("Expected %d buckets in group %v, got %d", expectedBucketsPerGroup, targetTime, len(buckets))
		}
	}

	if totalBuckets != 10 {
		t.Errorf("Expected total of 10 buckets across all groups, got %d", totalBuckets)
	}
}

func TestAggregator_UpdateAggregate(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	aggregator := ts.aggregator

	// Create initial aggregate
	target := &AggregatedValue{
		Sum:     10.0,
		Min:     5.0,
		Max:     15.0,
		Count:   2,
		Average: 5.0, // Will be recalculated
		Latest:  15.0,
	}

	// Create source aggregate to merge
	source := &AggregatedValue{
		Sum:     30.0,
		Min:     8.0,
		Max:     22.0,
		Count:   3,
		Average: 10.0, // Will be recalculated
		Latest:  22.0,
	}

	// Update target with source
	aggregator.updateAggregate(target, source)

	// Verify results
	if target.Sum != 40.0 {
		t.Errorf("Expected sum 40.0, got %f", target.Sum)
	}

	if target.Count != 5 {
		t.Errorf("Expected count 5, got %d", target.Count)
	}

	if target.Average != 8.0 {
		t.Errorf("Expected average 8.0, got %f", target.Average)
	}

	if target.Min != 5.0 {
		t.Errorf("Expected min 5.0, got %f", target.Min)
	}

	if target.Max != 22.0 {
		t.Errorf("Expected max 22.0, got %f", target.Max)
	}

	if target.Latest != 22.0 {
		t.Errorf("Expected latest 22.0, got %f", target.Latest)
	}
}

func TestAggregator_MergeAggregates(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	aggregator := ts.aggregator

	// Create target aggregate (existing in bucket)
	target := &AggregatedValue{
		Sum:     20.0,
		Min:     10.0,
		Max:     30.0,
		Count:   2,
		Average: 10.0,
		Latest:  30.0,
	}

	// Create source aggregate (from aggregation)
	source := &AggregatedValue{
		Sum:     60.0,
		Min:     15.0,
		Max:     45.0,
		Count:   4,
		Average: 15.0,
		Latest:  45.0,
	}

	// Merge source into target
	aggregator.mergeAggregates(target, source)

	// Verify results
	if target.Sum != 80.0 {
		t.Errorf("Expected sum 80.0, got %f", target.Sum)
	}

	if target.Count != 6 {
		t.Errorf("Expected count 6, got %d", target.Count)
	}

	expectedAverage := 80.0 / 6.0
	if target.Average != expectedAverage {
		t.Errorf("Expected average %f, got %f", expectedAverage, target.Average)
	}

	if target.Min != 10.0 {
		t.Errorf("Expected min 10.0, got %f", target.Min)
	}

	if target.Max != 45.0 {
		t.Errorf("Expected max 45.0, got %f", target.Max)
	}

	if target.Latest != 45.0 {
		t.Errorf("Expected latest 45.0, got %f", target.Latest)
	}
}

func TestAggregator_GetStatus(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 30,
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	status := ts.aggregator.GetStatus()

	// Check basic status fields
	if status.AggregationInterval != 30*time.Second {
		t.Errorf("Expected interval 30s, got %v", status.AggregationInterval)
	}

	if status.BucketCounts == nil {
		t.Error("Expected bucket counts to be populated")
	}

	// Should have all resolution types
	expectedResolutions := []BucketResolution{
		Resolution1Min, Resolution5Min, Resolution1Hour, Resolution1Day,
	}

	for _, resolution := range expectedResolutions {
		if _, exists := status.BucketCounts[resolution]; !exists {
			t.Errorf("Expected bucket count for resolution %s", resolution)
		}
	}
}

func TestAggregator_EstimateMemoryUsage(t *testing.T) {
	config := TimeSeriesConfig{Enabled: true}
	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Add some test data
	for i := 0; i < 100; i++ {
		metric := Metric{
			Name:      "memory_test",
			Value:     float64(i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
		ts.Store(metric)
	}

	estimate := ts.aggregator.EstimateMemoryUsage()

	// Check that estimate has reasonable values
	if estimate.TotalBuckets == 0 {
		t.Error("Expected some buckets for memory estimation")
	}

	if estimate.EstimatedMB <= 0 {
		t.Error("Expected positive memory estimate")
	}

	if estimate.BucketBreakdown == nil {
		t.Error("Expected bucket breakdown")
	}

	t.Logf("Memory estimate: %.2f MB, %d total buckets", 
		estimate.EstimatedMB, estimate.TotalBuckets)
}

func TestAggregator_ExecuteAggregationJob(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 60,
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Create test data in 1-minute buckets
	baseTime := time.Now().Truncate(time.Minute).Add(-15 * time.Minute)
	
	for i := 0; i < 10; i++ {
		metric := Metric{
			Name:      "job_test",
			Value:     float64((i % 5) * 10), // Varying values for interesting aggregation
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
		}
		ts.Store(metric)
	}

	// Create aggregation job (1min -> 5min)
	job := AggregationJob{
		SourceResolution: Resolution1Min,
		TargetResolution: Resolution5Min,
		StartTime:        baseTime,
		EndTime:          baseTime.Add(10 * time.Minute),
	}

	// Execute the job
	err := ts.aggregator.executeAggregationJob(job)
	if err != nil {
		t.Fatalf("Failed to execute aggregation job: %v", err)
	}

	// Verify 5-minute buckets were created
	fiveMinBuckets := ts.buckets[Resolution5Min]
	fiveMinBuckets.mutex.RLock()
	bucketCount := len(fiveMinBuckets.buckets)
	fiveMinBuckets.mutex.RUnlock()

	if bucketCount == 0 {
		t.Error("Expected 5-minute buckets to be created")
	}

	// Query the aggregated data
	query := TimeSeriesQuery{
		MetricNames: []string{"job_test"},
		Resolution:  Resolution5Min,
		StartTime:   baseTime,
		EndTime:     baseTime.Add(10 * time.Minute),
		Aggregation: "avg",
	}

	result, err := ts.GetTimeSeries(query)
	if err != nil {
		t.Fatalf("Failed to query aggregated data: %v", err)
	}

	series, exists := result.Series["job_test"]
	if !exists {
		t.Error("Expected job_test series in aggregated data")
	}

	if len(series) == 0 {
		t.Error("Expected data points in aggregated series")
	}

	t.Logf("Successfully aggregated to %d data points in 5-minute resolution", len(series))
}

func TestAggregator_ConcurrentAggregation(t *testing.T) {
	config := TimeSeriesConfig{
		Enabled:                 true,
		AggregationIntervalSecs: 1, // Fast aggregation for testing
	}

	rawStorage := NewInMemoryStorage(StorageConfig{})
	ts := NewTimeSeriesStorage(rawStorage, config)
	defer ts.Close()

	// Start aggregator
	ts.aggregator.Start()
	defer ts.aggregator.Stop()

	// Continuously add data while aggregation is running
	done := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			metric := Metric{
				Name:      "concurrent_agg",
				Value:     float64(i),
				Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			}
			ts.Store(metric)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for data insertion to complete
	<-done

	// Give aggregator time to run
	time.Sleep(2 * time.Second)

	// Force final aggregation
	ts.aggregator.ForceAggregation()

	// Verify we can still query data without deadlocks
	query := TimeSeriesQuery{
		MetricNames: []string{"concurrent_agg"},
		Resolution:  Resolution1Min,
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     time.Now(),
		Aggregation: "count",
	}

	result, err := ts.GetTimeSeries(query)
	if err != nil {
		t.Fatalf("Failed to query during concurrent aggregation: %v", err)
	}

	if len(result.Series) == 0 {
		t.Error("Expected data despite concurrent aggregation")
	}

	t.Logf("Successfully handled concurrent aggregation with %d series", len(result.Series))
}