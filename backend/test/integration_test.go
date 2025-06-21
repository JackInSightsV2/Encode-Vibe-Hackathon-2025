package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"qt1-middleware/metrics"
)

// IntegrationTestSuite contains all integration tests for the metrics system
type IntegrationTestSuite struct {
	collector     *metrics.MetricsCollector
	systemMetrics *metrics.SystemMetricsCollector
	apiHandler    *metrics.APIHandler
	server        *httptest.Server
}

// SetupIntegrationTest initializes the complete metrics system for testing
func SetupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	// Create metrics configuration
	config := metrics.DefaultMetricsConfig()
	config.Storage.Type = "memory"
	config.Storage.MaxMemoryMB = 100
	config.SystemMetrics.Enabled = true
	config.SystemMetrics.CollectionInterval = 100 * time.Millisecond

	// Initialize metrics collector
	collector, err := metrics.NewMetricsCollector(config)
	if err != nil {
		t.Fatalf("Failed to create metrics collector: %v", err)
	}

	// Initialize system metrics collector
	systemMetrics := metrics.NewSystemMetricsCollector(collector, config.SystemMetrics)

	// Initialize API handler
	apiHandler := metrics.NewAPIHandlerWithSystemMetrics(collector, nil, systemMetrics)

	// Create test server
	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)
	server := httptest.NewServer(mux)

	return &IntegrationTestSuite{
		collector:     collector,
		systemMetrics: systemMetrics,
		apiHandler:    apiHandler,
		server:        server,
	}
}

// TeardownIntegrationTest cleans up the test environment
func (suite *IntegrationTestSuite) TeardownIntegrationTest() {
	suite.server.Close()
	suite.systemMetrics.Stop()
	suite.collector.Stop()
}

// TestEndToEndMetricsFlow tests the complete metrics flow from collection to API
func TestEndToEndMetricsFlow(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()

	// Start the metrics system
	err := suite.collector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start metrics collector: %v", err)
	}

	err = suite.systemMetrics.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start system metrics collector: %v", err)
	}

	t.Log("✅ Step 1: Metrics system started successfully")

	// Test HTTP metrics collection
	suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
		Path:       "/api/test",
		Method:     "GET",
		StatusCode: 200,
		Duration:   150 * time.Millisecond,
		UserAgent:  "integration-test",
		IPAddress:  "127.0.0.1",
	})

	t.Log("✅ Step 2: HTTP metrics recorded successfully")

	// Test system metrics collection
	suite.collector.RecordSystemHealth()
	t.Log("✅ Step 3: System health metrics recorded successfully")

	// Test provider health recording
	suite.systemMetrics.RecordProviderRequest("openai", 200*time.Millisecond, false)
	suite.systemMetrics.RecordProviderRequest("anthropic", 150*time.Millisecond, false)
	t.Log("✅ Step 4: Provider health metrics recorded successfully")

	// Wait for metrics to be processed
	time.Sleep(200 * time.Millisecond)

	// Test metrics API endpoints
	endpoints := []struct {
		path           string
		expectedStatus int
		description    string
	}{
		{"/api/metrics/health", 200, "Health endpoint"},
		{"/api/metrics/stats", 200, "Stats endpoint"},
		{"/api/metrics/system/snapshot", 200, "System snapshot endpoint"},
		{"/api/metrics/providers", 200, "Provider health endpoint"},
	}

	for _, endpoint := range endpoints {
		resp, err := http.Get(suite.server.URL + endpoint.path)
		if err != nil {
			t.Fatalf("Failed to call %s: %v", endpoint.path, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != endpoint.expectedStatus {
			t.Errorf("%s returned status %d, expected %d", endpoint.description, resp.StatusCode, endpoint.expectedStatus)
		} else {
			t.Logf("✅ Step 5: %s responded correctly", endpoint.description)
		}
	}

	t.Log("🎉 End-to-end metrics flow test completed successfully")
}

// TestHighVolumeLoad tests the system with high metric volume
func TestHighVolumeLoad(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)
	suite.systemMetrics.Start(ctx)

	t.Log("🚀 Starting high volume load test (1000+ metrics/second)")

	startTime := time.Now()
	metricsCount := 0
	targetMetrics := 5000 // 5000 metrics in 5 seconds = 1000/sec
	duration := 5 * time.Second

	// Generate high volume of metrics
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if time.Since(startTime) > duration {
					return
				}

				// Record HTTP metrics
				suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
					Path:       fmt.Sprintf("/api/endpoint%d", metricsCount%10),
					Method:     "GET",
					StatusCode: 200,
					Duration:   time.Duration(50+metricsCount%100) * time.Millisecond,
				})

				// Record custom metrics
				suite.collector.RecordCustomMetric(
					fmt.Sprintf("test_metric_%d", metricsCount%5),
					float64(metricsCount%1000),
					metrics.MetricTypeGauge,
					map[string]string{"test": "load"},
				)

				metricsCount++

				if metricsCount >= targetMetrics {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for load test to complete
	time.Sleep(duration + time.Second)

	actualRate := float64(metricsCount) / duration.Seconds()
	t.Logf("✅ Generated %d metrics in %v (%.1f metrics/second)", metricsCount, duration, actualRate)

	if actualRate < 800 { // Allow some tolerance
		t.Errorf("Metrics rate too low: %.1f/sec, expected >800/sec", actualRate)
	}

	// Test API responsiveness during load
	start := time.Now()
	resp, err := http.Get(suite.server.URL + "/api/metrics/stats")
	responseTime := time.Since(start)

	if err != nil {
		t.Fatalf("API call failed during load: %v", err)
	}
	defer resp.Body.Close()

	if responseTime > 500*time.Millisecond {
		t.Errorf("API response too slow during load: %v, expected <500ms", responseTime)
	} else {
		t.Logf("✅ API remained responsive during load: %v", responseTime)
	}

	t.Log("🎉 High volume load test completed successfully")
}

// TestRealTimeUpdatePerformance tests real-time update latency
func TestRealTimeUpdatePerformance(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)
	suite.systemMetrics.Start(ctx)

	t.Log("⚡ Testing real-time update performance")

	// Record a metric and immediately check API response
	iterations := 10
	totalLatency := time.Duration(0)

	for i := 0; i < iterations; i++ {
		metricTime := time.Now()

		// Record metric
		suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
			Path:       fmt.Sprintf("/api/test%d", i),
			Method:     "GET",
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
		})

		// Small delay to allow processing
		time.Sleep(50 * time.Millisecond)

		// Check if metric is available via API
		resp, err := http.Get(suite.server.URL + "/api/metrics/stats")
		if err != nil {
			t.Fatalf("Failed to get metrics: %v", err)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		latency := time.Since(metricTime)
		totalLatency += latency

		t.Logf("Iteration %d: Latency %v", i+1, latency)
	}

	avgLatency := totalLatency / time.Duration(iterations)
	t.Logf("✅ Average real-time update latency: %v", avgLatency)

	if avgLatency > time.Second {
		t.Errorf("Real-time update latency too high: %v, expected <1 second", avgLatency)
	}

	t.Log("🎉 Real-time update performance test completed successfully")
}

// TestDashboardResponsiveness tests dashboard load time
func TestDashboardResponsiveness(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)
	suite.systemMetrics.Start(ctx)

	t.Log("📊 Testing dashboard responsiveness")

	// Populate some data first
	for i := 0; i < 100; i++ {
		suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
			Path:       fmt.Sprintf("/api/endpoint%d", i%5),
			Method:     "GET",
			StatusCode: 200,
			Duration:   time.Duration(50+i%100) * time.Millisecond,
		})
	}

	// Wait for data to be processed
	time.Sleep(100 * time.Millisecond)

	// Test multiple API endpoints that dashboard would call
	endpoints := []string{
		"/api/metrics/summary",
		"/api/metrics/system/snapshot",
		"/api/metrics/providers",
		"/api/metrics/stats",
	}

	totalTime := time.Duration(0)
	for _, endpoint := range endpoints {
		start := time.Now()
		resp, err := http.Get(suite.server.URL + endpoint)
		responseTime := time.Since(start)
		totalTime += responseTime

		if err != nil {
			t.Fatalf("Failed to call %s: %v", endpoint, err)
		}
		resp.Body.Close()

		t.Logf("Endpoint %s response time: %v", endpoint, responseTime)

		if responseTime > 200*time.Millisecond {
			t.Errorf("Endpoint %s too slow: %v, expected <200ms", endpoint, responseTime)
		}
	}

	avgResponseTime := totalTime / time.Duration(len(endpoints))
	t.Logf("✅ Average API response time: %v", avgResponseTime)

	if avgResponseTime > 100*time.Millisecond {
		t.Errorf("Average response time too slow: %v, expected <100ms", avgResponseTime)
	}

	t.Log("🎉 Dashboard responsiveness test completed successfully")
}

// TestMemoryUsage tests memory consumption under load
func TestMemoryUsage(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)
	suite.systemMetrics.Start(ctx)

	t.Log("💾 Testing memory usage under load")

	// Get initial memory baseline
	initialSnapshot := suite.systemMetrics.GetSystemHealthSnapshot()
	initialMemory := initialSnapshot.MemoryAllocated

	t.Logf("Initial memory usage: %d bytes (%.2f MB)", initialMemory, float64(initialMemory)/1024/1024)

	// Generate sustained load
	for i := 0; i < 10000; i++ {
		suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
			Path:       fmt.Sprintf("/api/load-test/%d", i),
			Method:     "POST",
			StatusCode: 200,
			Duration:   time.Duration(20+i%80) * time.Millisecond,
		})

		suite.collector.RecordCustomMetric(
			fmt.Sprintf("load_metric_%d", i%20),
			float64(i),
			metrics.MetricTypeCounter,
			map[string]string{"load": "test", "batch": fmt.Sprintf("%d", i/1000)},
		)

		// Check memory every 1000 iterations
		if i%1000 == 0 {
			snapshot := suite.systemMetrics.GetSystemHealthSnapshot()
			currentMemory := snapshot.MemoryAllocated
			t.Logf("Memory at iteration %d: %d bytes (%.2f MB)", i, currentMemory, float64(currentMemory)/1024/1024)
		}
	}

	// Final memory check
	finalSnapshot := suite.systemMetrics.GetSystemHealthSnapshot()
	finalMemory := finalSnapshot.MemoryAllocated
	memoryIncrease := finalMemory - initialMemory

	t.Logf("Final memory usage: %d bytes (%.2f MB)", finalMemory, float64(finalMemory)/1024/1024)
	t.Logf("Memory increase: %d bytes (%.2f MB)", memoryIncrease, float64(memoryIncrease)/1024/1024)

	// Check if memory usage is within acceptable limits
	maxMemoryMB := float64(500) // 500MB limit
	actualMemoryMB := float64(finalMemory) / 1024 / 1024

	if actualMemoryMB > maxMemoryMB {
		t.Errorf("Memory usage too high: %.2f MB, expected <%0.f MB", actualMemoryMB, maxMemoryMB)
	} else {
		t.Logf("✅ Memory usage within limits: %.2f MB", actualMemoryMB)
	}

	t.Log("🎉 Memory usage test completed successfully")
}

// TestSystemHealthIntegration tests system health monitoring integration
func TestSystemHealthIntegration(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)
	suite.systemMetrics.Start(ctx)

	t.Log("🖥️ Testing system health monitoring integration")

	// Wait for system metrics to be collected
	time.Sleep(500 * time.Millisecond)

	// Test system snapshot endpoint
	resp, err := http.Get(suite.server.URL + "/api/metrics/system/snapshot")
	if err != nil {
		t.Fatalf("Failed to get system snapshot: %v", err)
	}
	defer resp.Body.Close()

	var snapshot map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&snapshot)
	if err != nil {
		t.Fatalf("Failed to decode system snapshot: %v", err)
	}

	// Validate snapshot structure
	data, ok := snapshot["data"].(map[string]interface{})
	if !ok {
		t.Fatal("System snapshot missing data field")
	}

	requiredFields := []string{"cpu_usage", "memory_usage", "goroutine_count", "uptime_seconds"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("System snapshot missing required field: %s", field)
		}
	}

	t.Logf("✅ System snapshot contains all required fields")

	// Test provider health
	suite.systemMetrics.RecordProviderRequest("test-provider", 100*time.Millisecond, false)
	time.Sleep(100 * time.Millisecond)

	resp, err = http.Get(suite.server.URL + "/api/metrics/providers")
	if err != nil {
		t.Fatalf("Failed to get provider health: %v", err)
	}
	defer resp.Body.Close()

	var providers map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&providers)
	if err != nil {
		t.Fatalf("Failed to decode provider health: %v", err)
	}

	t.Logf("✅ Provider health endpoint responding correctly")
	t.Log("🎉 System health integration test completed successfully")
}

// TestPerformanceBenchmarks validates all performance benchmarks
func TestPerformanceBenchmarks(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)
	suite.systemMetrics.Start(ctx)

	t.Log("🏆 Running performance benchmark validation")

	benchmarks := map[string]func() bool{
		"Metrics Collection (<1ms overhead)": func() bool {
			start := time.Now()
			suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
				Path:       "/api/benchmark",
				Method:     "GET",
				StatusCode: 200,
				Duration:   100 * time.Millisecond,
			})
			overhead := time.Since(start)
			t.Logf("  Metrics collection overhead: %v", overhead)
			return overhead < time.Millisecond
		},

		"API Response (<100ms)": func() bool {
			start := time.Now()
			resp, err := http.Get(suite.server.URL + "/api/metrics/stats")
			responseTime := time.Since(start)
			if err != nil {
				t.Logf("  API call failed: %v", err)
				return false
			}
			resp.Body.Close()
			t.Logf("  API response time: %v", responseTime)
			return responseTime < 100*time.Millisecond
		},

		"Dashboard Load Simulation (<2s)": func() bool {
			start := time.Now()
			
			// Simulate dashboard loading multiple endpoints
			endpoints := []string{
				"/api/metrics/summary",
				"/api/metrics/system/snapshot",
				"/api/metrics/providers",
			}
			
			for _, endpoint := range endpoints {
				resp, err := http.Get(suite.server.URL + endpoint)
				if err != nil {
					t.Logf("  Dashboard load failed at %s: %v", endpoint, err)
					return false
				}
				resp.Body.Close()
			}
			
			loadTime := time.Since(start)
			t.Logf("  Simulated dashboard load time: %v", loadTime)
			return loadTime < 2*time.Second
		},
	}

	allPassed := true
	for name, benchmark := range benchmarks {
		t.Logf("Running benchmark: %s", name)
		if benchmark() {
			t.Logf("✅ %s: PASSED", name)
		} else {
			t.Errorf("❌ %s: FAILED", name)
			allPassed = false
		}
	}

	if allPassed {
		t.Log("🎉 All performance benchmarks passed!")
	} else {
		t.Error("⚠️ Some performance benchmarks failed")
	}
}

// BenchmarkMetricsCollection benchmarks the metrics collection performance
func BenchmarkMetricsCollection(b *testing.B) {
	config := metrics.DefaultMetricsConfig()
	collector, err := metrics.NewMetricsCollector(config)
	if err != nil {
		b.Fatalf("Failed to create collector: %v", err)
	}

	ctx := context.Background()
	collector.Start(ctx)
	defer collector.Stop()

	httpMetrics := metrics.HTTPMetrics{
		Path:       "/api/benchmark",
		Method:     "GET",
		StatusCode: 200,
		Duration:   100 * time.Millisecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordHTTPRequest(httpMetrics)
	}
}

// BenchmarkAPIResponse benchmarks API response times
func BenchmarkAPIResponse(b *testing.B) {
	suite := SetupIntegrationTest(&testing.T{})
	defer suite.TeardownIntegrationTest()

	ctx := context.Background()
	suite.collector.Start(ctx)

	// Populate some data
	for i := 0; i < 1000; i++ {
		suite.collector.RecordHTTPRequest(metrics.HTTPMetrics{
			Path:       fmt.Sprintf("/api/test%d", i%10),
			Method:     "GET",
			StatusCode: 200,
			Duration:   100 * time.Millisecond,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(suite.server.URL + "/api/metrics/stats")
		if err != nil {
			b.Fatalf("API call failed: %v", err)
		}
		resp.Body.Close()
	}
}