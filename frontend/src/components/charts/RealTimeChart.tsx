import React from 'react';
import MetricsChart, { CHART_COLORS } from './MetricsChart';
import { useRealTimeMetric } from '../../hooks/useRealTimeMetrics';

interface RealTimeChartProps {
  metricName: string;
  title: string;
  type?: 'line' | 'area' | 'bar';
  color?: string;
  height?: number;
  refreshInterval?: number; // milliseconds (for polling fallback)
  maxDataPoints?: number;
  resolution?: '1m' | '5m' | '1h' | '1d';
  aggregation?: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest';
  unit?: string;
  timeRange?: string; // e.g., '1h', '6h', '24h'
  useWebSocket?: boolean; // Enable WebSocket real-time updates
}

const RealTimeChart: React.FC<RealTimeChartProps> = ({
  metricName,
  title,
  type = 'line',
  color = CHART_COLORS.primary,
  height = 300,
  refreshInterval = 30000, // 30 seconds (fallback polling)
  maxDataPoints = 100,
  resolution = '1m',
  aggregation = 'avg',
  unit = '',
  timeRange = '1h',
  useWebSocket = true
}) => {
  // Use the real-time metrics hook
  const {
    points: data,
    loading,
    error,
    lastUpdate,
    connectionMethod,
    refresh
  } = useRealTimeMetric(metricName, {
    resolution,
    aggregation,
    timeRange,
    maxDataPoints,
    fallbackPollingInterval: useWebSocket ? refreshInterval : refreshInterval
  });

  const handleRetry = () => {
    refresh();
  };

  const formatLastUpdate = (): string => {
    if (!lastUpdate) return '';
    
    const now = new Date();
    const diffMs = now.getTime() - lastUpdate.getTime();
    const diffSecs = Math.floor(diffMs / 1000);
    
    if (diffSecs < 60) {
      return `${diffSecs}s ago`;
    } else if (diffSecs < 3600) {
      return `${Math.floor(diffSecs / 60)}m ago`;
    } else {
      return lastUpdate.toLocaleTimeString();
    }
  };

  const getRefreshStatus = (): 'active' | 'paused' | 'error' => {
    if (error) return 'error';
    if (connectionMethod !== 'none') return 'active';
    return 'paused';
  };

  const getConnectionDisplay = (): string => {
    switch (connectionMethod) {
      case 'websocket': return 'Live (WebSocket)';
      case 'polling': return 'Live (Polling)';
      default: return 'Paused';
    }
  };

  return (
    <div className="relative">
      <MetricsChart
        data={data}
        type={type}
        title={title}
        metricName={metricName}
        color={color}
        height={height}
        realTime={true}
        loading={loading}
        error={error || undefined}
        unit={unit}
        aggregation={aggregation}
      />
      
      {/* Real-time status overlay */}
      <div className="absolute top-4 right-4 flex flex-col items-end space-y-2">
        {/* Refresh status indicator */}
        <div className="flex items-center space-x-2 bg-white bg-opacity-90 rounded-lg px-3 py-1 border border-gray-200">
          <div className={`w-2 h-2 rounded-full ${
            getRefreshStatus() === 'active' ? 'bg-green-500 animate-pulse' :
            getRefreshStatus() === 'error' ? 'bg-red-500' :
            'bg-gray-400'
          }`}></div>
          <span className="text-xs text-gray-600">
            {getConnectionDisplay()}
          </span>
        </div>
        
        {/* Last update time */}
        {lastUpdate && !loading && (
          <div className="text-xs text-gray-500 bg-white bg-opacity-90 rounded px-2 py-1">
            Updated {formatLastUpdate()}
          </div>
        )}
        
        {/* Retry button for errors */}
        {error && (
          <button
            onClick={handleRetry}
            className="text-xs bg-red-100 hover:bg-red-200 text-red-700 px-2 py-1 rounded transition-colors"
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
};

export default RealTimeChart;