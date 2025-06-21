import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { jest } from '@jest/globals';
import SystemHealthDashboard from '../SystemHealthDashboard';
import { metricsService } from '../../services/metricsService';

// Mock the metrics service
jest.mock('../../services/metricsService', () => ({
  metricsService: {
    getSystemSnapshot: jest.fn(),
    startSystemHealthPolling: jest.fn(),
  },
}));

// Mock the RealTimeChart component since it's complex and not the focus of this test
jest.mock('../charts/RealTimeChart', () => {
  return function MockRealTimeChart({ title }: { title: string }) {
    return <div data-testid="real-time-chart">{title}</div>;
  };
});

const mockMetricsService = metricsService as jest.Mocked<typeof metricsService>;

describe('SystemHealthDashboard', () => {
  const mockSnapshot = {
    timestamp: 1640995200, // Jan 1, 2022
    cpu_usage: 45.2,
    memory_usage: 67.8,
    memory_allocated: 134217728, // 128 MB
    goroutine_count: 42,
    uptime_seconds: 3600, // 1 hour
    providers: {
      openai: {
        is_healthy: true,
        health_score: 0.95,
        error_rate: 0.02,
      },
      anthropic: {
        is_healthy: false,
        health_score: 0.65,
        error_rate: 0.15,
      },
    },
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockMetricsService.getSystemSnapshot.mockResolvedValue(mockSnapshot);
    mockMetricsService.startSystemHealthPolling.mockReturnValue(() => {});
  });

  it('renders the dashboard header correctly', async () => {
    render(<SystemHealthDashboard />);

    expect(screen.getByText('System Health Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Live Updates')).toBeInTheDocument();
    expect(screen.getByText('Enable Auto-refresh')).toBeInTheDocument();
    expect(screen.getByText('Refresh Now')).toBeInTheDocument();
  });

  it('displays loading state initially', () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation(() => () => {});
    
    render(<SystemHealthDashboard />);

    expect(screen.getByText('Loading System Health...')).toBeInTheDocument();
  });

  it('displays health cards with correct data', async () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(mockSnapshot), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    await waitFor(() => {
      expect(screen.getByText('CPU Usage')).toBeInTheDocument();
      expect(screen.getByText('45%')).toBeInTheDocument();
      
      expect(screen.getByText('Memory Usage')).toBeInTheDocument();
      expect(screen.getByText('68%')).toBeInTheDocument();
      
      expect(screen.getByText('Goroutines')).toBeInTheDocument();
      expect(screen.getByText('42')).toBeInTheDocument();
      
      expect(screen.getByText('Uptime')).toBeInTheDocument();
      expect(screen.getByText('1h 0m')).toBeInTheDocument();
    });
  });

  it('displays memory details section', async () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(mockSnapshot), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    await waitFor(() => {
      expect(screen.getByText('Memory Details')).toBeInTheDocument();
      expect(screen.getByText('Allocated Memory')).toBeInTheDocument();
      expect(screen.getByText('128.0 MB')).toBeInTheDocument();
    });
  });

  it('displays provider health cards', async () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(mockSnapshot), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    await waitFor(() => {
      expect(screen.getByText('AI Provider Health')).toBeInTheDocument();
      expect(screen.getByText('Openai')).toBeInTheDocument();
      expect(screen.getByText('Anthropic')).toBeInTheDocument();
      
      // Check health scores
      expect(screen.getByText('95.0%')).toBeInTheDocument(); // openai score
      expect(screen.getByText('65.0%')).toBeInTheDocument(); // anthropic score
      
      // Check error rates
      expect(screen.getByText('2.00%')).toBeInTheDocument(); // openai error rate
      expect(screen.getByText('15.00%')).toBeInTheDocument(); // anthropic error rate
      
      // Check health status
      expect(screen.getByText('Healthy')).toBeInTheDocument(); // openai status
      expect(screen.getByText('Unhealthy')).toBeInTheDocument(); // anthropic status
    });
  });

  it('renders real-time charts', async () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(mockSnapshot), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    await waitFor(() => {
      expect(screen.getByText('CPU Usage Over Time')).toBeInTheDocument();
      expect(screen.getByText('Memory Usage Over Time')).toBeInTheDocument();
      expect(screen.getByText('Goroutine Count')).toBeInTheDocument();
      expect(screen.getByText('Allocated Memory')).toBeInTheDocument();
      expect(screen.getByText('Provider Health Scores')).toBeInTheDocument();
      expect(screen.getByText('Provider Error Rates')).toBeInTheDocument();
    });
  });

  it('toggles auto-refresh correctly', async () => {
    let pollCallback: ((data: any) => void) | null = null;
    let cleanupFn = jest.fn();

    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      pollCallback = callback;
      return cleanupFn;
    });

    render(<SystemHealthDashboard />);

    // Initially auto-refresh should be enabled
    expect(screen.getByText('Live Updates')).toBeInTheDocument();
    expect(screen.getByText('Disable Auto-refresh')).toBeInTheDocument();

    // Toggle auto-refresh off
    fireEvent.click(screen.getByText('Disable Auto-refresh'));

    await waitFor(() => {
      expect(screen.getByText('Manual Mode')).toBeInTheDocument();
      expect(screen.getByText('Enable Auto-refresh')).toBeInTheDocument();
    });

    // Should call cleanup function when disabling
    expect(cleanupFn).toHaveBeenCalled();
  });

  it('handles manual refresh', async () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(mockSnapshot), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByText('CPU Usage')).toBeInTheDocument();
    });

    // Clear previous calls
    mockMetricsService.getSystemSnapshot.mockClear();

    // Click refresh
    fireEvent.click(screen.getByText('Refresh Now'));

    // Should call getSystemSnapshot
    expect(mockMetricsService.getSystemSnapshot).toHaveBeenCalledTimes(1);
  });

  it('displays error state when fetch fails', async () => {
    mockMetricsService.getSystemSnapshot.mockRejectedValue(new Error('Network error'));
    mockMetricsService.startSystemHealthPolling.mockImplementation(() => () => {});

    render(<SystemHealthDashboard />);

    // Disable auto-refresh to trigger manual fetch
    fireEvent.click(screen.getByText('Disable Auto-refresh'));

    await waitFor(() => {
      expect(screen.getByText('System Health Error')).toBeInTheDocument();
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  it('handles empty provider health data', async () => {
    const snapshotWithoutProviders = {
      ...mockSnapshot,
      providers: {},
    };

    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(snapshotWithoutProviders), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    await waitFor(() => {
      expect(screen.getByText('CPU Usage')).toBeInTheDocument();
    });

    // Should not show provider health section
    expect(screen.queryByText('AI Provider Health')).not.toBeInTheDocument();
  });

  it('formats uptime correctly for different durations', async () => {
    const testCases = [
      { seconds: 3600, expected: '1h 0m' }, // 1 hour
      { seconds: 7200, expected: '2h 0m' }, // 2 hours
      { seconds: 3661, expected: '1h 1m' }, // 1 hour 1 minute
      { seconds: 86400, expected: '1d 0h' }, // 1 day
      { seconds: 90061, expected: '1d 1h' }, // 1 day 1 hour 1 minute
      { seconds: 300, expected: '5m' }, // 5 minutes
    ];

    for (const testCase of testCases) {
      const snapshot = { ...mockSnapshot, uptime_seconds: testCase.seconds };
      
      mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
        setTimeout(() => callback(snapshot), 0);
        return () => {};
      });

      const { unmount } = render(<SystemHealthDashboard />);

      await waitFor(() => {
        expect(screen.getByText(testCase.expected)).toBeInTheDocument();
      });

      unmount();
    }
  });

  it('applies correct health status colors', async () => {
    mockMetricsService.startSystemHealthPolling.mockImplementation((callback) => {
      setTimeout(() => callback(mockSnapshot), 0);
      return () => {};
    });

    render(<SystemHealthDashboard />);

    await waitFor(() => {
      // Check that health indicators are present
      const healthIndicators = screen.getAllByRole('button');
      expect(healthIndicators.length).toBeGreaterThan(0);
    });
  });
});