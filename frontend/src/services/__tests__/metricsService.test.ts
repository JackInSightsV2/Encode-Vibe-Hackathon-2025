import MetricsService, { metricsService } from '../metricsService';

// Mock fetch globally
global.fetch = jest.fn();
const mockFetch = fetch as jest.MockedFunction<typeof fetch>;

describe('MetricsService', () => {
  beforeEach(() => {
    mockFetch.mockClear();
  });

  describe('getTimeSeries', () => {
    it('constructs correct query parameters', async () => {
      const mockResponse = {
        success: true,
        data: {
          resolution: '1m',
          time_range: { start: '2024-01-01T10:00:00Z', end: '2024-01-01T11:00:00Z' },
          series: { cpu_usage: [{ timestamp: '2024-01-01T10:00:00Z', value: 50 }] },
          metadata: { total_points: 1, buckets_queried: 1, query_duration: 100 }
        }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const query = {
        metrics: ['cpu_usage', 'memory_usage'],
        resolution: '1m' as const,
        aggregation: 'avg' as const,
        range: '1h'
      };

      await metricsService.getTimeSeries(query);

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/metrics/timeseries?metrics=cpu_usage%2Cmemory_usage&resolution=1m&aggregation=avg&range=1h'
      );
    });

    it('handles custom time range', async () => {
      const mockResponse = {
        success: true,
        data: { series: {}, metadata: {} }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const query = {
        resolution: '5m' as const,
        aggregation: 'max' as const,
        startTime: '2024-01-01T10:00:00Z',
        endTime: '2024-01-01T11:00:00Z'
      };

      await metricsService.getTimeSeries(query);

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/metrics/timeseries?resolution=5m&aggregation=max&start=2024-01-01T10%3A00%3A00Z&end=2024-01-01T11%3A00%3A00Z'
      );
    });

    it('throws error on HTTP failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      } as Response);

      const query = {
        resolution: '1m' as const,
        aggregation: 'avg' as const
      };

      await expect(metricsService.getTimeSeries(query)).rejects.toThrow(
        'HTTP 500: Internal Server Error'
      );
    });

    it('throws error on API failure', async () => {
      const mockResponse = {
        success: false,
        message: 'Invalid metric name'
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const query = {
        resolution: '1m' as const,
        aggregation: 'avg' as const
      };

      await expect(metricsService.getTimeSeries(query)).rejects.toThrow(
        'Invalid metric name'
      );
    });
  });

  describe('getSummary', () => {
    it('fetches summary with time range', async () => {
      const mockResponse = {
        success: true,
        data: {
          total_requests: 1000,
          requests_per_second: 10.5,
          error_rate: 0.05,
          system_health: { memory_usage: 512, cpu_usage: 25 }
        }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      await metricsService.getSummary('24h');

      expect(mockFetch).toHaveBeenCalledWith('/api/metrics/summary?range=24h');
    });

    it('fetches summary without time range', async () => {
      const mockResponse = {
        success: true,
        data: {}
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      await metricsService.getSummary();

      expect(mockFetch).toHaveBeenCalledWith('/api/metrics/summary?');
    });
  });

  describe('getMetric', () => {
    it('fetches single metric with defaults', async () => {
      const mockResponse = {
        success: true,
        data: {
          series: {
            cpu_usage: [
              { timestamp: '2024-01-01T10:00:00Z', value: 50 },
              { timestamp: '2024-01-01T10:01:00Z', value: 55 }
            ]
          }
        }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const result = await metricsService.getMetric('cpu_usage');

      expect(result).toEqual([
        { timestamp: '2024-01-01T10:00:00Z', value: 50 },
        { timestamp: '2024-01-01T10:01:00Z', value: 55 }
      ]);
    });

    it('returns empty array for missing metric', async () => {
      const mockResponse = {
        success: true,
        data: { series: {} }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const result = await metricsService.getMetric('nonexistent_metric');

      expect(result).toEqual([]);
    });
  });

  describe('getMultipleMetrics', () => {
    it('fetches multiple metrics', async () => {
      const mockResponse = {
        success: true,
        data: {
          series: {
            cpu_usage: [{ timestamp: '2024-01-01T10:00:00Z', value: 50 }],
            memory_usage: [{ timestamp: '2024-01-01T10:00:00Z', value: 1024 }]
          }
        }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const result = await metricsService.getMultipleMetrics(
        ['cpu_usage', 'memory_usage'],
        '5m',
        '6h',
        'max'
      );

      expect(result).toEqual({
        cpu_usage: [{ timestamp: '2024-01-01T10:00:00Z', value: 50 }],
        memory_usage: [{ timestamp: '2024-01-01T10:00:00Z', value: 1024 }]
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/metrics/timeseries?metrics=cpu_usage%2Cmemory_usage&resolution=5m&aggregation=max&range=6h'
      );
    });
  });

  describe('forceAggregation', () => {
    it('sends POST request to force aggregation', async () => {
      const mockResponse = {
        success: true,
        message: 'Aggregation triggered'
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      await metricsService.forceAggregation();

      expect(mockFetch).toHaveBeenCalledWith('/api/metrics/aggregator/force', {
        method: 'POST'
      });
    });
  });

  describe('calculateStats', () => {
    it('calculates statistics correctly', () => {
      const data = [
        { timestamp: '2024-01-01T10:00:00Z', value: 10 },
        { timestamp: '2024-01-01T10:01:00Z', value: 20 },
        { timestamp: '2024-01-01T10:02:00Z', value: 30 },
        { timestamp: '2024-01-01T10:03:00Z', value: 25 },
        { timestamp: '2024-01-01T10:04:00Z', value: 35 }
      ];

      const stats = metricsService.calculateStats(data);

      expect(stats.min).toBe(10);
      expect(stats.max).toBe(35);
      expect(stats.avg).toBe(24); // (10 + 20 + 30 + 25 + 35) / 5
      expect(stats.latest).toBe(35);
    });

    it('handles empty data', () => {
      const stats = metricsService.calculateStats([]);

      expect(stats.min).toBe(0);
      expect(stats.max).toBe(0);
      expect(stats.avg).toBe(0);
      expect(stats.latest).toBe(0);
      expect(stats.trend).toBe('stable');
    });

    it('calculates trend correctly', () => {
      // Upward trend data
      const upwardData = Array.from({ length: 20 }, (_, i) => ({
        timestamp: `2024-01-01T10:${i.toString().padStart(2, '0')}:00Z`,
        value: i * 2 // Steadily increasing
      }));

      const upwardStats = metricsService.calculateStats(upwardData);
      expect(upwardStats.trend).toBe('up');

      // Downward trend data
      const downwardData = Array.from({ length: 20 }, (_, i) => ({
        timestamp: `2024-01-01T10:${i.toString().padStart(2, '0')}:00Z`,
        value: 100 - i * 3 // Steadily decreasing
      }));

      const downwardStats = metricsService.calculateStats(downwardData);
      expect(downwardStats.trend).toBe('down');
    });
  });

  describe('startRealTimePolling', () => {
    beforeEach(() => {
      jest.useFakeTimers();
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it('polls metrics at specified interval', async () => {
      const mockResponse = {
        success: true,
        data: { series: { cpu_usage: [] } }
      };

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const callback = jest.fn();
      const cleanup = metricsService.startRealTimePolling(
        ['cpu_usage'],
        callback,
        { interval: 1000 }
      );

      // Initial call
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Advance time and check for polling
      jest.advanceTimersByTime(1000);
      await Promise.resolve(); // Let async operations complete

      expect(mockFetch).toHaveBeenCalledTimes(2);

      // Clean up
      cleanup();
    });

    it('calls callback with data', async () => {
      const mockData = { cpu_usage: [{ timestamp: '2024-01-01T10:00:00Z', value: 50 }] };
      const mockResponse = {
        success: true,
        data: { series: mockData }
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      } as Response);

      const callback = jest.fn();
      const cleanup = metricsService.startRealTimePolling(['cpu_usage'], callback);

      await Promise.resolve(); // Let initial fetch complete

      expect(callback).toHaveBeenCalledWith(mockData);

      cleanup();
    });
  });

  describe('getTimeRanges', () => {
    it('returns predefined time ranges', () => {
      const ranges = metricsService.getTimeRanges();

      expect(ranges).toEqual([
        { label: 'Last 15 minutes', value: '15m', minutes: 15 },
        { label: 'Last hour', value: '1h', minutes: 60 },
        { label: 'Last 6 hours', value: '6h', minutes: 360 },
        { label: 'Last 24 hours', value: '24h', minutes: 1440 },
        { label: 'Last 7 days', value: '7d', minutes: 10080 },
        { label: 'Last 30 days', value: '30d', minutes: 43200 },
      ]);
    });
  });

  describe('formatTimeRange', () => {
    it('formats time range to ISO strings', () => {
      const start = new Date('2024-01-01T10:00:00Z');
      const end = new Date('2024-01-01T11:00:00Z');

      const result = metricsService.formatTimeRange(start, end);

      expect(result).toEqual({
        startTime: '2024-01-01T10:00:00.000Z',
        endTime: '2024-01-01T11:00:00.000Z'
      });
    });
  });
});