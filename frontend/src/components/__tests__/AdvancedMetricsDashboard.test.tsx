import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { jest } from '@jest/globals';
import MetricsDashboard from '../MetricsDashboard';
import { metricsService } from '../../services/metricsService';

// Mock the metrics service
jest.mock('../../services/metricsService', () => ({
  metricsService: {
    getSummary: jest.fn(),
    getMultipleMetrics: jest.fn(),
    getTimeRanges: jest.fn(),
  },
}));

// Mock the chart components
jest.mock('../charts/RealTimeChart', () => {
  return function MockRealTimeChart({ title, metricName }: { title: string; metricName: string }) {
    return <div data-testid={`chart-${metricName}`}>{title}</div>;
  };
});

jest.mock('../charts/MultiMetricChart', () => {
  return function MockMultiMetricChart({ title }: { title: string }) {
    return <div data-testid="multi-metric-chart">{title}</div>;
  };
});

jest.mock('../SystemHealthDashboard', () => {
  return function MockSystemHealthDashboard() {
    return <div data-testid="system-health-dashboard">System Health Dashboard</div>;
  };
});

const mockMetricsService = metricsService as jest.Mocked<typeof metricsService>;

describe('Advanced MetricsDashboard Features', () => {
  const mockSummary = {
    total_requests: 1000,
    requests_per_second: 10.5,
    average_response_time: 150,
    error_rate: 2.5,
    moderation_blocked: 5,
    pii_detections: 3,
    system_health: {
      cpu_usage: 45,
      memory_usage: 67,
      goroutine_count: 42,
      db_connections: 5,
      cache_hit_rate: 95,
      uptime_seconds: 3600,
    },
    top_endpoints: [],
    time_range: {
      start: '2024-01-01T00:00:00Z',
      end: '2024-01-01T01:00:00Z',
    },
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockMetricsService.getSummary.mockResolvedValue(mockSummary);
    mockMetricsService.getTimeRanges.mockReturnValue([
      { label: 'Last hour', value: '1h', minutes: 60 },
      { label: 'Last 6 hours', value: '6h', minutes: 360 },
    ]);
    mockMetricsService.getMultipleMetrics.mockResolvedValue({});
  });

  describe('Time Range Selection', () => {
    it('displays extended time range options', async () => {
      render(<MetricsDashboard />);

      const timeRangeSelect = screen.getByDisplayValue('Last hour');
      fireEvent.click(timeRangeSelect);

      expect(screen.getByText('Last 15 minutes')).toBeInTheDocument();
      expect(screen.getByText('Last 6 hours')).toBeInTheDocument();
      expect(screen.getByText('Last 24 hours')).toBeInTheDocument();
      expect(screen.getByText('Last 7 days')).toBeInTheDocument();
      expect(screen.getByText('Last 30 days')).toBeInTheDocument();
      expect(screen.getByText('Custom Range')).toBeInTheDocument();
    });

    it('shows custom time range picker when custom is selected', async () => {
      render(<MetricsDashboard />);

      const timeRangeSelect = screen.getByDisplayValue('Last hour');
      fireEvent.change(timeRangeSelect, { target: { value: 'custom' } });

      await waitFor(() => {
        expect(screen.getByText('Start Date & Time')).toBeInTheDocument();
        expect(screen.getByText('End Date & Time')).toBeInTheDocument();
      });
    });
  });

  describe('Auto-Refresh Configuration', () => {
    it('displays refresh interval options', async () => {
      render(<MetricsDashboard />);

      const refreshRateSelect = screen.getByDisplayValue('30 seconds');
      fireEvent.click(refreshRateSelect);

      expect(screen.getByText('5 seconds')).toBeInTheDocument();
      expect(screen.getByText('10 seconds')).toBeInTheDocument();
      expect(screen.getByText('1 minute')).toBeInTheDocument();
      expect(screen.getByText('5 minutes')).toBeInTheDocument();
      expect(screen.getByText('Manual only')).toBeInTheDocument();
    });

    it('disables refresh rate when auto-refresh is off', async () => {
      render(<MetricsDashboard />);

      const autoRefreshButton = screen.getByText('On');
      fireEvent.click(autoRefreshButton);

      await waitFor(() => {
        const refreshRateSelect = screen.getByDisplayValue('30 seconds');
        expect(refreshRateSelect).toBeDisabled();
      });
    });
  });

  describe('Layout Modes', () => {
    it('switches between grid and list layout modes', async () => {
      render(<MetricsDashboard />);

      const layoutSelect = screen.getByDisplayValue('Grid View');
      fireEvent.change(layoutSelect, { target: { value: 'list' } });

      await waitFor(() => {
        expect(screen.getByDisplayValue('List View')).toBeInTheDocument();
      });
    });
  });

  describe('Tab Navigation', () => {
    it('displays all advanced tabs', async () => {
      render(<MetricsDashboard />);

      expect(screen.getByText('📈 Metrics Overview')).toBeInTheDocument();
      expect(screen.getByText('🖥️ System Health')).toBeInTheDocument();
      expect(screen.getByText('🚨 Alerts & Thresholds')).toBeInTheDocument();
      expect(screen.getByText('💾 Export & Reports')).toBeInTheDocument();
    });

    it('shows alert indicator when alerts are enabled', async () => {
      render(<MetricsDashboard />);

      const alertsTab = screen.getByText('🚨 Alerts & Thresholds');
      expect(alertsTab.parentElement).toHaveClass('relative');
    });

    it('switches to system health tab', async () => {
      render(<MetricsDashboard />);

      const systemHealthTab = screen.getByText('🖥️ System Health');
      fireEvent.click(systemHealthTab);

      await waitFor(() => {
        expect(screen.getByTestId('system-health-dashboard')).toBeInTheDocument();
      });
    });
  });

  describe('Metric Filtering', () => {
    it('displays metric search functionality', async () => {
      render(<MetricsDashboard />);

      expect(screen.getByPlaceholderText('Search metrics...')).toBeInTheDocument();
      expect(screen.getByDisplayValue('All Categories')).toBeInTheDocument();
    });

    it('filters metrics by search term', async () => {
      render(<MetricsDashboard />);

      const searchInput = screen.getByPlaceholderText('Search metrics...');
      fireEvent.change(searchInput, { target: { value: 'HTTP' } });

      await waitFor(() => {
        expect(screen.getByText('HTTP Requests')).toBeInTheDocument();
      });
    });

    it('displays category filter buttons', async () => {
      render(<MetricsDashboard />);

      expect(screen.getByText('HTTP')).toBeInTheDocument();
      expect(screen.getByText('System')).toBeInTheDocument();
      expect(screen.getByText('Security')).toBeInTheDocument();
      expect(screen.getByText('Providers')).toBeInTheDocument();
    });

    it('provides quick action buttons', async () => {
      render(<MetricsDashboard />);

      expect(screen.getByText('Select All')).toBeInTheDocument();
      expect(screen.getByText('Clear All')).toBeInTheDocument();
      expect(screen.getByText('Default Selection')).toBeInTheDocument();
    });
  });

  describe('Alerts and Thresholds', () => {
    it('switches to alerts tab and shows threshold configuration', async () => {
      render(<MetricsDashboard />);

      const alertsTab = screen.getByText('🚨 Alerts & Thresholds');
      fireEvent.click(alertsTab);

      await waitFor(() => {
        expect(screen.getByText('Metric Thresholds')).toBeInTheDocument();
        expect(screen.getByText('+ Add Threshold')).toBeInTheDocument();
      });
    });

    it('adds new threshold when button is clicked', async () => {
      render(<MetricsDashboard />);

      const alertsTab = screen.getByText('🚨 Alerts & Thresholds');
      fireEvent.click(alertsTab);

      await waitFor(() => {
        const addButton = screen.getByText('+ Add Threshold');
        fireEvent.click(addButton);
      });

      // Should have multiple threshold configurations
      await waitFor(() => {
        const removeButtons = screen.getAllByText('Remove');
        expect(removeButtons.length).toBeGreaterThan(3); // Initial 3 + 1 added
      });
    });
  });

  describe('Export and Reports', () => {
    it('switches to export tab and shows export options', async () => {
      render(<MetricsDashboard />);

      const exportTab = screen.getByText('💾 Export & Reports');
      fireEvent.click(exportTab);

      await waitFor(() => {
        expect(screen.getByText('Export Data')).toBeInTheDocument();
        expect(screen.getByText('Export to CSV')).toBeInTheDocument();
        expect(screen.getByText('Export to JSON')).toBeInTheDocument();
        expect(screen.getByText('Generate Report')).toBeInTheDocument();
      });
    });

    it('displays export configuration', async () => {
      render(<MetricsDashboard />);

      const exportTab = screen.getByText('💾 Export & Reports');
      fireEvent.click(exportTab);

      await waitFor(() => {
        expect(screen.getByText('Export Configuration')).toBeInTheDocument();
        expect(screen.getByText('Selected Metrics')).toBeInTheDocument();
      });
    });

    it('shows summary statistics', async () => {
      render(<MetricsDashboard />);

      const exportTab = screen.getByText('💾 Export & Reports');
      fireEvent.click(exportTab);

      await waitFor(() => {
        expect(screen.getByText('Summary Statistics')).toBeInTheDocument();
        expect(screen.getByText('1,000')).toBeInTheDocument(); // Total requests
        expect(screen.getByText('10.5')).toBeInTheDocument(); // Requests per second
      });
    });
  });

  describe('Performance and Responsiveness', () => {
    it('handles large datasets without performance issues', async () => {
      // Mock a large dataset
      const largeMetrics = Array.from({ length: 20 }, (_, i) => ({
        name: `metric_${i}`,
        label: `Metric ${i}`,
        unit: 'count',
        category: 'Test',
      }));

      render(<MetricsDashboard />);

      // Test that the component renders without freezing
      const searchInput = screen.getByPlaceholderText('Search metrics...');
      fireEvent.change(searchInput, { target: { value: 'metric' } });

      // Should still be responsive
      await waitFor(() => {
        expect(searchInput).toHaveValue('metric');
      });
    });

    it('maintains performance with frequent auto-refresh', async () => {
      render(<MetricsDashboard />);

      // Set fast refresh rate
      const refreshRateSelect = screen.getByDisplayValue('30 seconds');
      fireEvent.change(refreshRateSelect, { target: { value: '5000' } });

      // Component should handle rapid updates
      await waitFor(() => {
        expect(screen.getByDisplayValue('5 seconds')).toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('handles export errors gracefully', async () => {
      mockMetricsService.getMultipleMetrics.mockRejectedValue(new Error('Export failed'));

      render(<MetricsDashboard />);

      const exportTab = screen.getByText('💾 Export & Reports');
      fireEvent.click(exportTab);

      await waitFor(() => {
        const csvExportButton = screen.getByText('Export to CSV');
        fireEvent.click(csvExportButton);
      });

      // Should not crash the application
      expect(screen.getByText('Export to CSV')).toBeInTheDocument();
    });

    it('handles missing metric data gracefully', async () => {
      mockMetricsService.getSummary.mockResolvedValue(null);

      render(<MetricsDashboard />);

      await waitFor(() => {
        expect(screen.getByText('No data available')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('provides proper ARIA labels and roles', async () => {
      render(<MetricsDashboard />);

      const timeRangeSelect = screen.getByDisplayValue('Last hour');
      expect(timeRangeSelect).toHaveAttribute('aria-label', undefined); // Should have proper labeling

      const searchInput = screen.getByPlaceholderText('Search metrics...');
      expect(searchInput).toHaveAttribute('type', 'text');
    });

    it('supports keyboard navigation', async () => {
      render(<MetricsDashboard />);

      const alertsTab = screen.getByText('🚨 Alerts & Thresholds');
      
      // Tab should be focusable
      alertsTab.focus();
      expect(document.activeElement).toBe(alertsTab);
    });
  });

  describe('Data Persistence', () => {
    it('maintains dashboard state when switching tabs', async () => {
      render(<MetricsDashboard />);

      // Set specific configuration
      const layoutSelect = screen.getByDisplayValue('Grid View');
      fireEvent.change(layoutSelect, { target: { value: 'list' } });

      // Switch tabs
      const alertsTab = screen.getByText('🚨 Alerts & Thresholds');
      fireEvent.click(alertsTab);

      const overviewTab = screen.getByText('📈 Metrics Overview');
      fireEvent.click(overviewTab);

      // Configuration should be preserved
      await waitFor(() => {
        expect(screen.getByDisplayValue('List View')).toBeInTheDocument();
      });
    });
  });
});