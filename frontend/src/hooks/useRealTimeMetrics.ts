import { useState, useEffect, useCallback } from 'react';
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext';
import { metricsService, MetricDataPoint } from '../services/metricsService';

interface RealTimeMetricsConfig {
  metricNames: string[];
  resolution?: '1m' | '5m' | '1h' | '1d';
  aggregation?: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest';
  timeRange?: string;
  maxDataPoints?: number;
  fallbackPollingInterval?: number; // milliseconds, if WebSocket is not available
}

interface RealTimeMetricsData {
  [metricName: string]: MetricDataPoint[];
}

interface RealTimeMetricsReturn {
  data: RealTimeMetricsData;
  loading: boolean;
  error: string | null;
  lastUpdate: Date | null;
  connectionMethod: 'websocket' | 'polling' | 'none';
  refresh: () => Promise<void>;
}

export const useRealTimeMetrics = (config: RealTimeMetricsConfig): RealTimeMetricsReturn => {
  const {
    metricNames,
    resolution = '1m',
    aggregation = 'avg',
    timeRange = '1h',
    maxDataPoints = 100,
    fallbackPollingInterval = 30000
  } = config;

  const { isConnected } = useWebSocket();
  const [data, setData] = useState<RealTimeMetricsData>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const [connectionMethod, setConnectionMethod] = useState<'websocket' | 'polling' | 'none'>('none');

  // Handle real-time metric updates via WebSocket
  useWebSocketMessage('metrics_update', useCallback((wsData: any) => {
    try {
      if (wsData && wsData.metrics) {
        setData(prevData => {
          const newData = { ...prevData };
          
          // Process each metric update
          Object.entries(wsData.metrics).forEach(([metricName, metricValue]: [string, any]) => {
            if (metricNames.includes(metricName)) {
              const newPoint: MetricDataPoint = {
                timestamp: wsData.timestamp || new Date().toISOString(),
                value: typeof metricValue === 'number' ? metricValue : metricValue.value || 0
              };
              
              if (!newData[metricName]) {
                newData[metricName] = [];
              }
              
              // Add new point and maintain max data points
              newData[metricName] = [...newData[metricName], newPoint].slice(-maxDataPoints);
            }
          });
          
          return newData;
        });
        
        setLastUpdate(new Date());
        setError(null);
      }
    } catch (err) {
      console.error('Error processing WebSocket metrics update:', err);
    }
  }, [metricNames, maxDataPoints]));

  // Handle real-time health updates via WebSocket
  useWebSocketMessage('health_update', useCallback((wsData: any) => {
    try {
      if (wsData && wsData.system_health) {
        const healthMetrics = wsData.system_health;
        const timestamp = wsData.timestamp || new Date().toISOString();
        
        setData(prevData => {
          const newData = { ...prevData };
          
          // Map health metrics to our metric names
          const healthMapping: Record<string, string> = {
            'system_memory': 'memory_usage',
            'system_goroutines': 'goroutine_count',
            'system_cpu': 'cpu_usage'
          };
          
          Object.entries(healthMapping).forEach(([metricName, healthKey]) => {
            if (metricNames.includes(metricName) && healthMetrics[healthKey] !== undefined) {
              const newPoint: MetricDataPoint = {
                timestamp,
                value: healthMetrics[healthKey]
              };
              
              if (!newData[metricName]) {
                newData[metricName] = [];
              }
              
              newData[metricName] = [...newData[metricName], newPoint].slice(-maxDataPoints);
            }
          });
          
          return newData;
        });
        
        setLastUpdate(new Date());
      }
    } catch (err) {
      console.error('Error processing WebSocket health update:', err);
    }
  }, [metricNames, maxDataPoints]));

  // Fetch initial data and handle polling fallback
  const fetchData = useCallback(async () => {
    try {
      const result = await metricsService.getMultipleMetrics(
        metricNames,
        resolution,
        timeRange,
        aggregation
      );
      
      // Limit data points for each metric
      const limitedData: RealTimeMetricsData = {};
      Object.entries(result).forEach(([metricName, points]) => {
        limitedData[metricName] = points.slice(-maxDataPoints);
      });
      
      setData(limitedData);
      setError(null);
      setLastUpdate(new Date());
    } catch (err) {
      console.error('Error fetching metrics data:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
    } finally {
      setLoading(false);
    }
  }, [metricNames, resolution, timeRange, aggregation, maxDataPoints]);

  // Set up data fetching and connection method
  useEffect(() => {
    // Initial fetch
    fetchData();

    if (isConnected) {
      // Use WebSocket for real-time updates
      setConnectionMethod('websocket');
      
      // Request real-time metrics subscription
      // Note: This would require implementing the subscription logic in the backend WebSocket handler
      // For now, we'll just rely on periodic broadcasts from the server
      
    } else {
      // Fall back to polling
      setConnectionMethod('polling');
      
      const interval = setInterval(fetchData, fallbackPollingInterval);
      return () => clearInterval(interval);
    }
  }, [isConnected, fetchData, fallbackPollingInterval]);

  // Update connection method when WebSocket status changes
  useEffect(() => {
    if (isConnected) {
      setConnectionMethod('websocket');
    } else if (fallbackPollingInterval > 0) {
      setConnectionMethod('polling');
    } else {
      setConnectionMethod('none');
    }
  }, [isConnected, fallbackPollingInterval]);

  return {
    data,
    loading,
    error,
    lastUpdate,
    connectionMethod,
    refresh: fetchData
  };
};

// Hook for a single metric
export const useRealTimeMetric = (
  metricName: string,
  options: Omit<RealTimeMetricsConfig, 'metricNames'> = {}
): RealTimeMetricsReturn & { 
  points: MetricDataPoint[];
  latest: number | null;
  stats: {
    min: number;
    max: number;
    avg: number;
    trend: 'up' | 'down' | 'stable';
  } | null;
} => {
  const result = useRealTimeMetrics({
    ...options,
    metricNames: [metricName]
  });

  const points = result.data[metricName] || [];
  const latest = points.length > 0 ? points[points.length - 1].value : null;
  
  const stats = points.length > 0 ? metricsService.calculateStats(points) : null;

  return {
    ...result,
    points,
    latest,
    stats
  };
};

// Hook for system health metrics
export const useSystemHealthMetrics = (options: Partial<RealTimeMetricsConfig> = {}) => {
  const healthMetrics = [
    'system_memory',
    'system_goroutines', 
    'system_cpu',
    'http_request',
    'moderation_event'
  ];

  return useRealTimeMetrics({
    metricNames: healthMetrics,
    resolution: '1m',
    aggregation: 'avg',
    timeRange: '1h',
    ...options
  });
};

export default useRealTimeMetrics;