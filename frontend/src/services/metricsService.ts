import { apiGet, apiPost } from '../utils/api';

export interface TimeSeriesQuery {
  metrics?: string[];
  resolution: '1m' | '5m' | '1h' | '1d';
  aggregation: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest';
  range?: string; // e.g., '1h', '6h', '24h', '7d'
  startTime?: string;
  endTime?: string;
}

export interface MetricDataPoint {
  timestamp: string;
  value: number;
}

export interface TimeSeriesResult {
  resolution: string;
  time_range: {
    start: string;
    end: string;
  };
  series: Record<string, MetricDataPoint[]>;
  metadata: {
    total_points: number;
    actual_range: {
      start: string;
      end: string;
    };
    buckets_queried: number;
    query_duration: number;
  };
}

export interface MetricsSummary {
  total_requests: number;
  requests_per_second: number;
  average_response_time: number;
  error_rate: number;
  moderation_blocked: number;
  pii_detections: number;
  system_health: {
    cpu_usage: number;
    memory_usage: number;
    goroutine_count: number;
    db_connections: number;
    cache_hit_rate: number;
    uptime_seconds: number;
  };
  top_endpoints: Array<{
    path: string;
    method: string;
    request_count: number;
    avg_duration: number;
    error_count: number;
    error_rate: number;
  }>;
  time_range: {
    start: string;
    end: string;
  };
}

export interface AggregatorStatus {
  running: boolean;
  aggregation_interval: number;
  bucket_counts: Record<string, number>;
}

export interface MemoryUsageEstimate {
  total_buckets: number;
  estimated_mb: number;
  bucket_breakdown: Record<string, number>;
}

export interface SystemHealthSnapshot {
  timestamp: number;
  cpu_usage: number;
  memory_usage: number;
  memory_allocated: number;
  memory_system: number;
  goroutine_count: number;
  gc_pauses: number[];
  provider_health: Record<string, ProviderHealth>;
  uptime_seconds: number;
}

export interface ProviderHealth {
  name: string;
  is_healthy: boolean;
  health_score: number;
  avg_response_time: number;
  error_rate: number;
  request_count: number;
  last_response: string;
}

class MetricsService {
  private baseUrl: string;

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl;
  }

  /**
   * Fetch time-series data for metrics
   */
  async getTimeSeries(query: TimeSeriesQuery): Promise<TimeSeriesResult> {
    const params = new URLSearchParams();
    
    if (query.metrics && query.metrics.length > 0) {
      params.append('metrics', query.metrics.join(','));
    }
    
    params.append('resolution', query.resolution);
    params.append('aggregation', query.aggregation);
    
    if (query.range) {
      params.append('range', query.range);
    }
    
    if (query.startTime && query.endTime) {
      params.append('start', query.startTime);
      params.append('end', query.endTime);
    }

    const result = await apiGet(`${this.baseUrl}/api/metrics/timeseries?${params}`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch time-series data');
    }
    
    return result.data;
  }

  /**
   * Fetch metrics summary
   */
  async getSummary(timeRange?: string): Promise<MetricsSummary> {
    const params = new URLSearchParams();
    
    if (timeRange) {
      params.append('range', timeRange);
    }

    const result = await apiGet(`${this.baseUrl}/api/metrics/summary?${params}`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch metrics summary');
    }
    
    return result.data;
  }

  /**
   * Fetch system health metrics
   */
  async getHealthMetrics(): Promise<any> {
    const result = await apiGet(`${this.baseUrl}/api/metrics/health`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch health metrics');
    }
    
    return result.data;
  }

  /**
   * Get aggregator status
   */
  async getAggregatorStatus(): Promise<{
    status: AggregatorStatus;
    memory_usage: MemoryUsageEstimate;
  }> {
    const result = await apiGet(`${this.baseUrl}/api/metrics/aggregator/status`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch aggregator status');
    }
    
    return {
      status: result.status,
      memory_usage: result.memory_usage
    };
  }

  /**
   * Force aggregation (for testing)
   */
  async forceAggregation(): Promise<void> {
    const result = await apiPost(`${this.baseUrl}/api/metrics/aggregator/force`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to force aggregation');
    }
  }

  /**
   * Get specific metric data with caching
   */
  async getMetric(
    metricName: string,
    resolution: '1m' | '5m' | '1h' | '1d' = '1m',
    timeRange: string = '1h',
    aggregation: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest' = 'avg'
  ): Promise<MetricDataPoint[]> {
    const result = await this.getTimeSeries({
      metrics: [metricName],
      resolution,
      aggregation,
      range: timeRange
    });
    
    return result.series[metricName] || [];
  }

  /**
   * Get multiple metrics in a single request
   */
  async getMultipleMetrics(
    metricNames: string[],
    resolution: '1m' | '5m' | '1h' | '1d' = '1m',
    timeRange: string = '1h',
    aggregation: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest' = 'avg'
  ): Promise<Record<string, MetricDataPoint[]>> {
    const result = await this.getTimeSeries({
      metrics: metricNames,
      resolution,
      aggregation,
      range: timeRange
    });
    
    return result.series;
  }

  /**
   * Stream real-time metrics using a polling approach
   */
  startRealTimePolling(
    metricNames: string[],
    callback: (data: Record<string, MetricDataPoint[]>) => void,
    options: {
      resolution?: '1m' | '5m' | '1h' | '1d';
      aggregation?: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest';
      interval?: number; // milliseconds
      timeRange?: string;
    } = {}
  ): () => void {
    const {
      resolution = '1m',
      aggregation = 'avg',
      interval = 30000, // 30 seconds
      timeRange = '1h'
    } = options;

    let intervalId: ReturnType<typeof setInterval>;
    let isActive = true;

    const fetchData = async () => {
      if (!isActive) return;
      
      try {
        const data = await this.getMultipleMetrics(
          metricNames,
          resolution,
          timeRange,
          aggregation
        );
        
        if (isActive) {
          callback(data);
        }
      } catch (error) {
        console.error('Real-time polling error:', error);
      }
    };

    // Initial fetch
    fetchData();

    // Set up polling
    intervalId = setInterval(fetchData, interval);

    // Return cleanup function
    return () => {
      isActive = false;
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }

  /**
   * Calculate metric statistics
   */
  calculateStats(data: MetricDataPoint[]): {
    min: number;
    max: number;
    avg: number;
    latest: number;
    trend: 'up' | 'down' | 'stable';
  } {
    if (data.length === 0) {
      return { min: 0, max: 0, avg: 0, latest: 0, trend: 'stable' };
    }

    const values = data.map(d => d.value);
    const min = Math.min(...values);
    const max = Math.max(...values);
    const avg = values.reduce((sum, val) => sum + val, 0) / values.length;
    const latest = values[values.length - 1];

    // Simple trend calculation (last 20% vs previous 20%)
    let trend: 'up' | 'down' | 'stable' = 'stable';
    if (data.length >= 10) {
      const recentCount = Math.floor(data.length * 0.2);
      const recentAvg = values.slice(-recentCount).reduce((sum, val) => sum + val, 0) / recentCount;
      const previousAvg = values.slice(-(recentCount * 2), -recentCount).reduce((sum, val) => sum + val, 0) / recentCount;
      
      const changePercent = ((recentAvg - previousAvg) / previousAvg) * 100;
      
      if (changePercent > 5) trend = 'up';
      else if (changePercent < -5) trend = 'down';
    }

    return { min, max, avg, latest, trend };
  }

  /**
   * Format time range for API queries
   */
  formatTimeRange(start: Date, end: Date): { startTime: string; endTime: string } {
    return {
      startTime: start.toISOString(),
      endTime: end.toISOString()
    };
  }

  /**
   * Get comprehensive system health metrics
   */
  async getSystemMetrics(): Promise<SystemHealthSnapshot> {
    const result = await apiGet(`${this.baseUrl}/api/metrics/system`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch system metrics');
    }
    
    return result.data;
  }

  /**
   * Get lightweight system snapshot for frequent polling
   */
  async getSystemSnapshot(): Promise<any> {
    const result = await apiGet(`${this.baseUrl}/api/metrics/system/snapshot`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch system snapshot');
    }
    
    return result.data;
  }

  /**
   * Get provider health information
   */
  async getProviderHealth(): Promise<Record<string, ProviderHealth>> {
    const result = await apiGet(`${this.baseUrl}/api/metrics/providers`);
    
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch provider health');
    }
    
    return result.data;
  }

  /**
   * Start real-time system health polling
   */
  startSystemHealthPolling(
    callback: (snapshot: any) => void,
    interval: number = 10000 // 10 seconds
  ): () => void {
    let intervalId: ReturnType<typeof setInterval>;
    let isActive = true;

    const fetchSnapshot = async () => {
      if (!isActive) return;
      
      try {
        const snapshot = await this.getSystemSnapshot();
        
        if (isActive) {
          callback(snapshot);
        }
      } catch (error) {
        console.error('System health polling error:', error);
      }
    };

    // Initial fetch
    fetchSnapshot();

    // Set up polling
    intervalId = setInterval(fetchSnapshot, interval);

    // Return cleanup function
    return () => {
      isActive = false;
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }

  /**
   * Get common time ranges
   */
  getTimeRanges(): Array<{ label: string; value: string; minutes: number }> {
    return [
      { label: 'Last 15 minutes', value: '15m', minutes: 15 },
      { label: 'Last hour', value: '1h', minutes: 60 },
      { label: 'Last 6 hours', value: '6h', minutes: 360 },
      { label: 'Last 24 hours', value: '24h', minutes: 1440 },
      { label: 'Last 7 days', value: '7d', minutes: 10080 },
      { label: 'Last 30 days', value: '30d', minutes: 43200 },
    ];
  }
}

// Create and export singleton instance
export const metricsService = new MetricsService();
export default MetricsService;