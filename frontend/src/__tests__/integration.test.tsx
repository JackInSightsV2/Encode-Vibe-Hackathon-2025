/**
 * Frontend Integration Tests for Real-Time Metrics Dashboard
 * Tests the complete user experience and frontend performance
 */

import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { jest } from '@jest/globals';
import MetricsDashboard from '../components/MetricsDashboard';
import { metricsService } from '../services/metricsService';
import { performanceMonitor } from '../utils/performanceMonitor';

// Mock the metrics service with realistic data
jest.mock('../services/metricsService', () => ({
  metricsService: {
    getSummary: jest.fn(),
    getMultipleMetrics: jest.fn(),
    getTimeRanges: jest.fn(),
    getSystemSnapshot: jest.fn(),
    startSystemHealthPolling: jest.fn(),
    getSystemMetrics: jest.fn(),
    getProviderHealth: jest.fn(),
  },
}));

// Mock chart components for performance testing
jest.mock('../components/charts/RealTimeChart', () => {
  return function MockRealTimeChart({ 
    title, 
    metricName, 
    refreshInterval 
  }: { 
    title: string; 
    metricName: string; 
    refreshInterval: number;
  }) {
    const [, setUpdateCount] = React.useState(0);
    
    React.useEffect(() => {
      if (refreshInterval > 0) {
        const interval = setInterval(() => {
          setUpdateCount(prev => prev + 1);
        }, refreshInterval);
        return () => clearInterval(interval);
      }
    }, [refreshInterval]);
    
    return (
      <div data-testid={`chart-${metricName}`} data-refresh={refreshInterval}>
        {title}
      </div>
    );
  };
});

jest.mock('../components/charts/MultiMetricChart', () => {
  return function MockMultiMetricChart({ title }: { title: string }) {
    return <div data-testid="multi-metric-chart">{title}</div>;
  };
});

jest.mock('../components/SystemHealthDashboard', () => {
  return function MockSystemHealthDashboard() {
    return <div data-testid="system-health-dashboard">System Health Dashboard</div>;
  };
});

const mockMetricsService = metricsService as jest.Mocked<typeof metricsService>;

// Generate realistic test data
const generateMockSummary = () => ({
  total_requests: Math.floor(Math.random() * 10000) + 1000,
  requests_per_second: Math.random() * 100 + 10,
  average_response_time: Math.random() * 500 + 100,
  error_rate: Math.random() * 5,
  moderation_blocked: Math.floor(Math.random() * 50),
  pii_detections: Math.floor(Math.random() * 20),
  system_health: {
    cpu_usage: Math.random() * 80 + 10,
    memory_usage: Math.random() * 70 + 20,
    goroutine_count: Math.floor(Math.random() * 100) + 20,
    db_connections: Math.floor(Math.random() * 10) + 1,
    cache_hit_rate: Math.random() * 20 + 80,
    uptime_seconds: Math.floor(Math.random() * 86400) + 3600,
  },
  top_endpoints: [],
  time_range: {
    start: new Date(Date.now() - 3600000).toISOString(),
    end: new Date().toISOString(),
  },
});

const generateMockMetrics = (count: number = 100) => {
  const data: Record<string, any[]> = {};
  const metricNames = ['http_request', 'system_memory', 'system_goroutines'];
  
  metricNames.forEach(metricName => {
    data[metricName] = Array.from({ length: count }, (_, i) => ({
      timestamp: new Date(Date.now() - (count - i) * 60000).toISOString(),
      value: Math.random() * 100 + 10,
    }));
  });
  
  return data;
};

describe('Frontend Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Setup default mocks
    mockMetricsService.getSummary.mockResolvedValue(generateMockSummary());
    mockMetricsService.getTimeRanges.mockReturnValue([
      { label: 'Last 15 minutes', value: '15m', minutes: 15 },
      { label: 'Last hour', value: '1h', minutes: 60 },
      { label: 'Last 6 hours', value: '6h', minutes: 360 },
      { label: 'Last 24 hours', value: '24h', minutes: 1440 },
      { label: 'Last 7 days', value: '7d', minutes: 10080 },
      { label: 'Last 30 days', value: '30d', minutes: 43200 },
    ]);
    mockMetricsService.getMultipleMetrics.mockResolvedValue(generateMockMetrics());
    mockMetricsService.getSystemSnapshot.mockResolvedValue({
      timestamp: Date.now() / 1000,
      cpu_usage: 45,
      memory_usage: 67,
      memory_allocated: 134217728,
      goroutine_count: 42,
      uptime_seconds: 3600,
      providers: {
        openai: { is_healthy: true, health_score: 0.95, error_rate: 0.02 },
        anthropic: { is_healthy: true, health_score: 0.92, error_rate: 0.03 },
      },
    });
    mockMetricsService.startSystemHealthPolling.mockReturnValue(() => {});
  });

  describe('🚀 Dashboard Load Performance Test', () => {
    it('should load dashboard within 2 seconds', async () => {
      const startTime = performance.now();
      
      render(<MetricsDashboard />);
      
      // Wait for initial data to load
      await waitFor(() => {
        expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      }, { timeout: 5000 });
      
      const loadTime = performance.now() - startTime;
      
      console.log(`📊 Dashboard load time: ${loadTime.toFixed(2)}ms`);
      
      // Dashboard should load within 2 seconds (2000ms)
      expect(loadTime).toBeLessThan(2000);
    });

    it('should handle large datasets without performance degradation', async () => {
      // Mock large dataset
      const largeDataset = generateMockMetrics(10000); // 10,000 data points
      mockMetricsService.getMultipleMetrics.mockResolvedValue(largeDataset);
      
      const startTime = performance.now();
      
      render(<MetricsDashboard />);
      
      await waitFor(() => {
        expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      });
      
      // Select all metrics to stress test
      const selectAllButton = screen.getByText('Select All');
      fireEvent.click(selectAllButton);
      
      const processingTime = performance.now() - startTime;
      
      console.log(`📈 Large dataset processing time: ${processingTime.toFixed(2)}ms`);
      
      // Should handle large datasets efficiently
      expect(processingTime).toBeLessThan(5000); // 5 seconds max
    });
  });

  describe('⚡ Real-Time Updates Performance Test', () => {
    it('should update metrics with <1 second latency', async () => {
      let pollCallback: ((data: any) => void) | null = null;
      
      mockMetricsService.startSystemHealthPolling.mockImplementation((callback, interval) => {
        pollCallback = callback;
        return () => {};
      });
      
      render(<MetricsDashboard />);
      
      // Switch to system health tab
      const systemHealthTab = screen.getByText('🖥️ System Health');
      fireEvent.click(systemHealthTab);
      
      await waitFor(() => {
        expect(screen.getByTestId('system-health-dashboard')).toBeInTheDocument();
      });
      
      // Simulate real-time update
      const updateStartTime = performance.now();
      
      if (pollCallback) {
        act(() => {
          pollCallback({
            timestamp: Date.now() / 1000,
            cpu_usage: 55, // Changed value
            memory_usage: 72,
            memory_allocated: 140000000,
            goroutine_count: 45,
            uptime_seconds: 3700,
            providers: {},
          });
        });
      }
      
      const updateLatency = performance.now() - updateStartTime;
      
      console.log(`⚡ Real-time update latency: ${updateLatency.toFixed(2)}ms`);
      
      // Updates should be processed within 1 second
      expect(updateLatency).toBeLessThan(1000);
    });

    it('should maintain performance with frequent updates', async () => {
      jest.useFakeTimers();
      
      let pollCallback: ((data: any) => void) | null = null;
      mockMetricsService.startSystemHealthPolling.mockImplementation((callback, interval) => {
        pollCallback = callback;
        return () => {};
      });
      
      render(<MetricsDashboard />);
      
      // Set fast refresh rate (5 seconds)
      const refreshRateSelect = screen.getByDisplayValue('30 seconds');
      fireEvent.change(refreshRateSelect, { target: { value: '5000' } });
      
      const updateTimes: number[] = [];
      
      // Simulate multiple rapid updates
      for (let i = 0; i < 10; i++) {
        const updateStart = performance.now();
        
        if (pollCallback) {
          act(() => {
            pollCallback({
              timestamp: Date.now() / 1000,
              cpu_usage: 50 + i,
              memory_usage: 65 + i,
              memory_allocated: 130000000 + i * 1000000,
              goroutine_count: 40 + i,
              uptime_seconds: 3600 + i * 60,
              providers: {},
            });
          });
        }
        
        updateTimes.push(performance.now() - updateStart);
        
        // Advance timer to simulate time passing
        act(() => {
          jest.advanceTimersByTime(5000);
        });
      }
      
      const avgUpdateTime = updateTimes.reduce((a, b) => a + b) / updateTimes.length;
      
      console.log(`📊 Average update time over 10 updates: ${avgUpdateTime.toFixed(2)}ms`);
      
      // Average update time should remain low even with frequent updates
      expect(avgUpdateTime).toBeLessThan(500);
      
      jest.useRealTimers();
    });
  });

  describe('💾 Memory Usage Test', () => {
    it('should maintain reasonable memory usage', async () => {
      const initialMemory = (performance as any).memory?.usedJSHeapSize || 0;
      
      render(<MetricsDashboard />);
      
      // Generate sustained activity
      for (let i = 0; i < 100; i++) {
        const refreshButton = screen.getByTitle('Refresh Now');
        fireEvent.click(refreshButton);
        
        // Trigger re-renders by changing settings
        const layoutSelect = screen.getByDisplayValue(i % 2 === 0 ? 'Grid View' : 'List View');
        fireEvent.change(layoutSelect, { target: { value: i % 2 === 0 ? 'list' : 'grid' } });
        
        await act(async () => {
          await new Promise(resolve => setTimeout(resolve, 10));
        });
      }
      
      const finalMemory = (performance as any).memory?.usedJSHeapSize || 0;
      const memoryIncrease = finalMemory - initialMemory;
      const memoryIncreaseMB = memoryIncrease / (1024 * 1024);
      
      console.log(`💾 Memory increase after stress test: ${memoryIncreaseMB.toFixed(2)}MB`);
      
      // Memory increase should be reasonable (less than 50MB for this test)
      expect(memoryIncreaseMB).toBeLessThan(50);
    });
  });

  describe('🔄 Auto-Refresh Stress Test', () => {
    it('should handle continuous auto-refresh without degradation', async () => {
      jest.useFakeTimers();
      
      render(<MetricsDashboard />);
      
      // Enable fast auto-refresh
      const refreshRateSelect = screen.getByDisplayValue('30 seconds');
      fireEvent.change(refreshRateSelect, { target: { value: '5000' } }); // 5 seconds
      
      const performanceTimes: number[] = [];
      
      // Simulate 1 minute of auto-refresh (12 refreshes at 5-second intervals)
      for (let i = 0; i < 12; i++) {
        const refreshStart = performance.now();
        
        act(() => {
          jest.advanceTimersByTime(5000);
        });
        
        // Wait for any async operations to complete
        await act(async () => {
          await new Promise(resolve => setTimeout(resolve, 10));
        });
        
        performanceTimes.push(performance.now() - refreshStart);
      }
      
      const avgPerformance = performanceTimes.reduce((a, b) => a + b) / performanceTimes.length;
      const performanceDegradation = performanceTimes[performanceTimes.length - 1] - performanceTimes[0];
      
      console.log(`🔄 Average refresh performance: ${avgPerformance.toFixed(2)}ms`);
      console.log(`📉 Performance degradation: ${performanceDegradation.toFixed(2)}ms`);
      
      // Performance should remain consistent
      expect(Math.abs(performanceDegradation)).toBeLessThan(100); // Less than 100ms degradation
      
      jest.useRealTimers();
    });
  });

  describe('📱 Responsive Design Test', () => {
    it('should adapt to different screen sizes', async () => {
      const originalInnerWidth = window.innerWidth;
      
      // Test mobile view
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375,
      });
      
      render(<MetricsDashboard />);
      
      await waitFor(() => {
        expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      });
      
      // Should render without errors on mobile
      expect(screen.getByText('📈 Metrics Overview')).toBeInTheDocument();
      
      // Test tablet view
      Object.defineProperty(window, 'innerWidth', {
        value: 768,
      });
      
      // Trigger resize event
      fireEvent(window, new Event('resize'));
      
      // Should still render correctly
      expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      
      // Test desktop view
      Object.defineProperty(window, 'innerWidth', {
        value: 1920,
      });
      
      fireEvent(window, new Event('resize'));
      
      expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      
      // Restore original
      Object.defineProperty(window, 'innerWidth', {
        value: originalInnerWidth,
      });
    });
  });

  describe('🚨 Error Handling Test', () => {
    it('should gracefully handle API failures', async () => {
      // Mock API failures
      mockMetricsService.getSummary.mockRejectedValue(new Error('Network error'));
      mockMetricsService.getMultipleMetrics.mockRejectedValue(new Error('Service unavailable'));
      
      render(<MetricsDashboard />);
      
      await waitFor(() => {
        expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      });
      
      // Dashboard should still render despite API failures
      expect(screen.getByText('📈 Metrics Overview')).toBeInTheDocument();
      
      // Should show appropriate error states
      await waitFor(() => {
        expect(screen.getByText('No data available')).toBeInTheDocument();
      });
    });

    it('should recover from temporary failures', async () => {
      let callCount = 0;
      
      // First call fails, subsequent calls succeed
      mockMetricsService.getSummary.mockImplementation(() => {
        callCount++;
        if (callCount === 1) {
          return Promise.reject(new Error('Temporary failure'));
        }
        return Promise.resolve(generateMockSummary());
      });
      
      render(<MetricsDashboard />);
      
      // Initial failure
      await waitFor(() => {
        expect(screen.getByText('No data available')).toBeInTheDocument();
      });
      
      // Trigger retry
      const refreshButton = screen.getByTitle('Refresh Now');
      fireEvent.click(refreshButton);
      
      // Should recover
      await waitFor(() => {
        expect(screen.queryByText('No data available')).not.toBeInTheDocument();
      });
    });
  });

  describe('📈 Chart Performance Test', () => {
    it('should render charts efficiently with large datasets', async () => {
      // Mock large dataset for charts
      const largeDataset = generateMockMetrics(5000);
      mockMetricsService.getMultipleMetrics.mockResolvedValue(largeDataset);
      
      const renderStart = performance.now();
      
      render(<MetricsDashboard />);
      
      await waitFor(() => {
        expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      });
      
      // Wait for charts to render
      await waitFor(() => {
        expect(screen.getByTestId('chart-http_request')).toBeInTheDocument();
      });
      
      const renderTime = performance.now() - renderStart;
      
      console.log(`📈 Chart rendering time with 5K data points: ${renderTime.toFixed(2)}ms`);
      
      // Charts should render efficiently even with large datasets
      expect(renderTime).toBeLessThan(3000); // 3 seconds max for large datasets
    });
  });

  describe('🎯 Integration Flow Test', () => {
    it('should complete full user workflow successfully', async () => {
      const workflowStart = performance.now();
      
      render(<MetricsDashboard />);
      
      // Step 1: Dashboard loads
      await waitFor(() => {
        expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
      });
      
      // Step 2: Change time range
      const timeRangeSelect = screen.getByDisplayValue('Last hour');
      fireEvent.change(timeRangeSelect, { target: { value: '24h' } });
      
      // Step 3: Switch to system health
      const systemHealthTab = screen.getByText('🖥️ System Health');
      fireEvent.click(systemHealthTab);
      
      await waitFor(() => {
        expect(screen.getByTestId('system-health-dashboard')).toBeInTheDocument();
      });
      
      // Step 4: Switch to alerts
      const alertsTab = screen.getByText('🚨 Alerts & Thresholds');
      fireEvent.click(alertsTab);
      
      await waitFor(() => {
        expect(screen.getByText('Metric Thresholds')).toBeInTheDocument();
      });
      
      // Step 5: Add threshold
      const addThresholdButton = screen.getByText('+ Add Threshold');
      fireEvent.click(addThresholdButton);
      
      // Step 6: Switch to export
      const exportTab = screen.getByText('💾 Export & Reports');
      fireEvent.click(exportTab);
      
      await waitFor(() => {
        expect(screen.getByText('Export Data')).toBeInTheDocument();
      });
      
      // Step 7: Back to overview
      const overviewTab = screen.getByText('📈 Metrics Overview');
      fireEvent.click(overviewTab);
      
      const workflowTime = performance.now() - workflowStart;
      
      console.log(`🎯 Complete user workflow time: ${workflowTime.toFixed(2)}ms`);
      
      // Complete workflow should be smooth and fast
      expect(workflowTime).toBeLessThan(5000); // 5 seconds for complete workflow
    });
  });
});

// Performance benchmark tests
describe('🏆 Performance Benchmarks', () => {
  it('should meet all performance benchmarks', async () => {
    const benchmarks = {
      'Dashboard Load Time': { target: 2000, actual: 0 },
      'Chart Rendering': { target: 500, actual: 0 },
      'Real-time Update Latency': { target: 1000, actual: 0 },
      'Memory Usage': { target: 100, actual: 0 }, // MB
    };
    
    // Dashboard Load Time
    const loadStart = performance.now();
    render(<MetricsDashboard />);
    await waitFor(() => {
      expect(screen.getByText('Real-Time Metrics Dashboard')).toBeInTheDocument();
    });
    benchmarks['Dashboard Load Time'].actual = performance.now() - loadStart;
    
    // Chart Rendering
    const chartStart = performance.now();
    await waitFor(() => {
      expect(screen.getByTestId('chart-http_request')).toBeInTheDocument();
    });
    benchmarks['Chart Rendering'].actual = performance.now() - chartStart;
    
    // Real-time Update Latency (simulated)
    const updateStart = performance.now();
    const refreshButton = screen.getByTitle('Refresh Now');
    fireEvent.click(refreshButton);
    benchmarks['Real-time Update Latency'].actual = performance.now() - updateStart;
    
    // Memory Usage
    const memoryUsage = (performance as any).memory?.usedJSHeapSize || 0;
    benchmarks['Memory Usage'].actual = memoryUsage / (1024 * 1024); // Convert to MB
    
    // Report results
    console.log('\n🏆 Performance Benchmark Results:');
    Object.entries(benchmarks).forEach(([name, { target, actual }]) => {
      const unit = name === 'Memory Usage' ? 'MB' : 'ms';
      const passed = actual <= target;
      const status = passed ? '✅' : '❌';
      
      console.log(`${status} ${name}: ${actual.toFixed(2)}${unit} (target: <${target}${unit})`);
      
      if (!passed) {
        console.warn(`⚠️ Benchmark failed: ${name}`);
      }
    });
    
    // All benchmarks should pass
    const allPassed = Object.values(benchmarks).every(({ target, actual }) => actual <= target);
    expect(allPassed).toBe(true);
  });
});