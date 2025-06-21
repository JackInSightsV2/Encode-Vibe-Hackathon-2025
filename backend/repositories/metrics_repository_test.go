package repositories

import (
	"context"
	"database/sql"
	"qt1-middleware/models"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupMetricsTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create metrics table
	schema := `
		CREATE TABLE metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metric_type VARCHAR(50) NOT NULL,
			name VARCHAR(100) NOT NULL,
			value REAL NOT NULL,
			unit VARCHAR(20),
			tags TEXT,
			labels TEXT,
			source VARCHAR(50) NOT NULL,
			timestamp DATETIME NOT NULL,
			collected_at DATETIME NOT NULL,
			metadata TEXT
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return db
}

func TestMetricsRepository_Create(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	metric := &models.Metric{
		MetricType: "counter",
		Name:       "test_counter",
		Value:      42.5,
		Unit:       stringPtr("requests"),
		Source:     "test_service",
		Timestamp:  time.Now(),
	}

	err := repo.Create(ctx, metric)
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}

	if metric.ID == 0 {
		t.Error("Expected metric ID to be set after creation")
	}

	if metric.CollectedAt.IsZero() {
		t.Error("Expected CollectedAt to be set")
	}
}

func TestMetricsRepository_CreateBatch(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	metrics := []*models.Metric{
		{
			MetricType: "counter",
			Name:       "batch_counter_1",
			Value:      10,
			Source:     "test_service",
		},
		{
			MetricType: "gauge",
			Name:       "batch_gauge_1",
			Value:      85.5,
			Source:     "test_service",
		},
		{
			MetricType: "histogram",
			Name:       "batch_histogram_1",
			Value:      0.250,
			Source:     "test_service",
		},
	}

	err := repo.CreateBatch(ctx, metrics)
	if err != nil {
		t.Fatalf("Failed to create batch metrics: %v", err)
	}

	// Verify all metrics have IDs
	for i, metric := range metrics {
		if metric.ID == 0 {
			t.Errorf("Expected metric %d to have ID set", i)
		}
		if metric.CollectedAt.IsZero() {
			t.Errorf("Expected metric %d to have CollectedAt set", i)
		}
	}
}

func TestMetricsRepository_GetByID(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	// Create a metric first
	metric := &models.Metric{
		MetricType: "gauge",
		Name:       "test_gauge",
		Value:      75.0,
		Unit:       stringPtr("percent"),
		Source:     "test_service",
		Timestamp:  time.Now(),
	}

	err := repo.Create(ctx, metric)
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}

	// Get the metric by ID
	retrievedMetric, err := repo.GetByID(ctx, metric.ID)
	if err != nil {
		t.Fatalf("Failed to get metric by ID: %v", err)
	}

	if retrievedMetric == nil {
		t.Fatal("Expected metric to be found")
	}

	if retrievedMetric.Name != metric.Name {
		t.Errorf("Expected name %s, got %s", metric.Name, retrievedMetric.Name)
	}

	if retrievedMetric.Value != metric.Value {
		t.Errorf("Expected value %f, got %f", metric.Value, retrievedMetric.Value)
	}

	if retrievedMetric.Unit == nil || *retrievedMetric.Unit != *metric.Unit {
		t.Error("Expected unit to match")
	}
}

func TestMetricsRepository_GetByName(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	metricName := "cpu_usage"

	// Create multiple metrics with the same name
	metrics := []*models.Metric{
		{
			MetricType: "gauge",
			Name:       metricName,
			Value:      45.0,
			Source:     "server1",
			Timestamp:  time.Now().Add(-2 * time.Hour),
		},
		{
			MetricType: "gauge",
			Name:       metricName,
			Value:      55.0,
			Source:     "server1",
			Timestamp:  time.Now().Add(-1 * time.Hour),
		},
		{
			MetricType: "gauge",
			Name:       metricName,
			Value:      65.0,
			Source:     "server1",
			Timestamp:  time.Now(),
		},
	}

	for _, metric := range metrics {
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Get metrics by name
	retrievedMetrics, err := repo.GetByName(ctx, metricName, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get metrics by name: %v", err)
	}

	if len(retrievedMetrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(retrievedMetrics))
	}

	// Should be ordered by timestamp DESC (newest first)
	for i := 1; i < len(retrievedMetrics); i++ {
		if retrievedMetrics[i].Timestamp.After(retrievedMetrics[i-1].Timestamp) {
			t.Error("Expected metrics to be ordered by timestamp DESC")
			break
		}
	}
}

func TestMetricsRepository_GetByType(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	// Create metrics of different types
	metrics := []*models.Metric{
		{MetricType: "counter", Name: "requests_total", Value: 1000, Source: "api"},
		{MetricType: "counter", Name: "errors_total", Value: 10, Source: "api"},
		{MetricType: "gauge", Name: "memory_usage", Value: 85.5, Source: "system"},
		{MetricType: "histogram", Name: "response_time", Value: 0.250, Source: "api"},
	}

	for _, metric := range metrics {
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Get counter metrics
	counters, err := repo.GetByType(ctx, "counter", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get counter metrics: %v", err)
	}

	if len(counters) != 2 {
		t.Errorf("Expected 2 counter metrics, got %d", len(counters))
	}

	for _, metric := range counters {
		if metric.MetricType != "counter" {
			t.Errorf("Expected counter metric, got %s", metric.MetricType)
		}
	}
}

func TestMetricsRepository_GetBySource(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	// Create metrics from different sources
	metrics := []*models.Metric{
		{MetricType: "counter", Name: "api_requests", Value: 100, Source: "api_server"},
		{MetricType: "gauge", Name: "cpu_usage", Value: 75, Source: "api_server"},
		{MetricType: "counter", Name: "db_queries", Value: 50, Source: "database"},
		{MetricType: "gauge", Name: "memory_usage", Value: 60, Source: "database"},
	}

	for _, metric := range metrics {
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Get API server metrics
	apiMetrics, err := repo.GetBySource(ctx, "api_server", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get API server metrics: %v", err)
	}

	if len(apiMetrics) != 2 {
		t.Errorf("Expected 2 API server metrics, got %d", len(apiMetrics))
	}

	for _, metric := range apiMetrics {
		if metric.Source != "api_server" {
			t.Errorf("Expected api_server source, got %s", metric.Source)
		}
	}
}

func TestMetricsRepository_List_WithFilters(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Create test metrics
	metrics := []*models.Metric{
		{
			MetricType: "counter",
			Name:       "requests_total",
			Value:      100,
			Source:     "api",
			Timestamp:  now.Add(-2 * time.Hour),
		},
		{
			MetricType: "gauge",
			Name:       "cpu_usage",
			Value:      75,
			Source:     "system",
			Timestamp:  now.Add(-1 * time.Hour),
		},
		{
			MetricType: "counter",
			Name:       "errors_total",
			Value:      5,
			Source:     "api",
			Timestamp:  now,
		},
	}

	for _, metric := range metrics {
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Test filtering by metric type
	filters := &models.MetricFilters{
		MetricType: stringPtr("counter"),
		Limit:      10,
	}

	counters, err := repo.List(ctx, filters)
	if err != nil {
		t.Fatalf("Failed to list counter metrics: %v", err)
	}

	if len(counters) != 2 {
		t.Errorf("Expected 2 counter metrics, got %d", len(counters))
	}

	// Test filtering by source
	filters = &models.MetricFilters{
		Source: stringPtr("api"),
		Limit:  10,
	}

	apiMetrics, err := repo.List(ctx, filters)
	if err != nil {
		t.Fatalf("Failed to list API metrics: %v", err)
	}

	if len(apiMetrics) != 2 {
		t.Errorf("Expected 2 API metrics, got %d", len(apiMetrics))
	}

	// Test filtering by time range
	startTime := now.Add(-90 * time.Minute)
	endTime := now.Add(-30 * time.Minute)

	filters = &models.MetricFilters{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     10,
	}

	timeRangeMetrics, err := repo.List(ctx, filters)
	if err != nil {
		t.Fatalf("Failed to list time range metrics: %v", err)
	}

	if len(timeRangeMetrics) != 1 {
		t.Errorf("Expected 1 metric in time range, got %d", len(timeRangeMetrics))
	}
}

func TestMetricsRepository_DeleteOlderThan(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Create metrics with different timestamps
	metrics := []*models.Metric{
		{
			MetricType: "counter",
			Name:       "old_metric_1",
			Value:      1,
			Source:     "test",
			Timestamp:  now.Add(-7 * 24 * time.Hour), // 7 days ago
		},
		{
			MetricType: "counter",
			Name:       "old_metric_2",
			Value:      2,
			Source:     "test",
			Timestamp:  now.Add(-5 * 24 * time.Hour), // 5 days ago
		},
		{
			MetricType: "counter",
			Name:       "recent_metric",
			Value:      3,
			Source:     "test",
			Timestamp:  now.Add(-1 * time.Hour), // 1 hour ago
		},
	}

	for _, metric := range metrics {
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Delete metrics older than 3 days
	cutoff := now.Add(-3 * 24 * time.Hour)
	deletedCount, err := repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("Failed to delete old metrics: %v", err)
	}

	if deletedCount != 2 {
		t.Errorf("Expected 2 metrics to be deleted, got %d", deletedCount)
	}

	// Verify remaining metrics
	remainingCount, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count remaining metrics: %v", err)
	}

	if remainingCount != 1 {
		t.Errorf("Expected 1 remaining metric, got %d", remainingCount)
	}
}

func TestMetricsRepository_GetTimeSeries(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	metricName := "cpu_usage"
	now := time.Now()

	// Create time series data
	values := []float64{45.0, 50.0, 55.0, 60.0, 65.0}
	for i, value := range values {
		metric := &models.Metric{
			MetricType: "gauge",
			Name:       metricName,
			Value:      value,
			Source:     "server1",
			Timestamp:  now.Add(time.Duration(i-2) * time.Hour), // -2h to +2h
		}
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create time series metric: %v", err)
		}
	}

	// Get time series data
	startTime := now.Add(-3 * time.Hour)
	endTime := now.Add(3 * time.Hour)

	timeSeries, err := repo.GetTimeSeries(ctx, metricName, startTime, endTime, "1h")
	if err != nil {
		t.Fatalf("Failed to get time series: %v", err)
	}

	if timeSeries == nil {
		t.Fatal("Expected time series response")
	}

	if timeSeries.Name != metricName {
		t.Errorf("Expected name %s, got %s", metricName, timeSeries.Name)
	}

	if len(timeSeries.Data) != 5 {
		t.Errorf("Expected 5 data points, got %d", len(timeSeries.Data))
	}

	// Verify data is ordered by timestamp
	for i := 1; i < len(timeSeries.Data); i++ {
		if timeSeries.Data[i].Timestamp.Before(timeSeries.Data[i-1].Timestamp) {
			t.Error("Expected time series data to be ordered by timestamp ASC")
			break
		}
	}
}

func TestMetricsRepository_GetSummary(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Create test metrics
	metrics := []*models.Metric{
		{MetricType: "counter", Name: "requests", Value: 100, Source: "api"},
		{MetricType: "counter", Name: "errors", Value: 5, Source: "api"},
		{MetricType: "gauge", Name: "cpu", Value: 75, Source: "system"},
		{MetricType: "gauge", Name: "memory", Value: 85, Source: "system"},
	}

	for _, metric := range metrics {
		metric.Timestamp = now
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Get summary
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(1 * time.Hour)

	summary, err := repo.GetSummary(ctx, startTime, endTime)
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	if summary.TotalMetrics != 4 {
		t.Errorf("Expected 4 total metrics, got %d", summary.TotalMetrics)
	}

	if summary.MetricTypes["counter"] != 2 {
		t.Errorf("Expected 2 counter metrics, got %d", summary.MetricTypes["counter"])
	}

	if summary.MetricTypes["gauge"] != 2 {
		t.Errorf("Expected 2 gauge metrics, got %d", summary.MetricTypes["gauge"])
	}

	if summary.Sources["api"] != 2 {
		t.Errorf("Expected 2 API metrics, got %d", summary.Sources["api"])
	}

	if summary.Sources["system"] != 2 {
		t.Errorf("Expected 2 system metrics, got %d", summary.Sources["system"])
	}
}

func TestMetricsRepository_Count(t *testing.T) {
	db := setupMetricsTestDB(t)
	defer db.Close()

	repo := NewMetricsRepository(db)
	ctx := context.Background()

	// Initially should be 0
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count metrics: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 metrics, got %d", count)
	}

	// Create some metrics
	metrics := []*models.Metric{
		{MetricType: "counter", Name: "metric1", Value: 1, Source: "test"},
		{MetricType: "gauge", Name: "metric2", Value: 2, Source: "test"},
		{MetricType: "histogram", Name: "metric3", Value: 3, Source: "test"},
	}

	for _, metric := range metrics {
		if err := repo.Create(ctx, metric); err != nil {
			t.Fatalf("Failed to create metric: %v", err)
		}
	}

	// Should now be 3
	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count metrics after creation: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 metrics, got %d", count)
	}
}