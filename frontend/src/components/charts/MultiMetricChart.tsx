import React, { useState, useEffect } from 'react'
import { apiGet } from '../../utils/api';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

interface MetricSeries {
  name: string;
  color: string;
  unit?: string;
  yAxisId?: 'left' | 'right';
}

interface MultiMetricDataPoint {
  timestamp: string;
  formattedTime: string;
  [metricName: string]: string | number;
}

interface MultiMetricChartProps {
  metrics: MetricSeries[];
  title: string;
  height?: number;
  resolution?: '1m' | '5m' | '1h' | '1d';
  aggregation?: 'avg' | 'min' | 'max' | 'sum' | 'count' | 'latest';
  timeRange?: string;
  refreshInterval?: number;
  realTime?: boolean;
}

const MultiMetricChart: React.FC<MultiMetricChartProps> = ({
  metrics,
  title,
  height = 400,
  resolution = '1m',
  aggregation = 'avg',
  timeRange = '1h',
  refreshInterval = 30000,
  realTime = false
}) => {
  const [data, setData] = useState<MultiMetricDataPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  useEffect(() => {
    let interval: ReturnType<typeof setInterval> | null = null;
    
    // Initial fetch
    fetchData();
    
    // Set up refresh interval for real-time updates
    if (realTime && refreshInterval > 0) {
      interval = setInterval(fetchData, refreshInterval);
    }
    
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [metrics, resolution, aggregation, timeRange, realTime, refreshInterval]);

  const fetchData = async () => {
    try {
      const metricNames = metrics.map(m => m.name).join(',');
      const params = new URLSearchParams({
        metrics: metricNames,
        resolution,
        aggregation,
        range: timeRange
      });

      const result = await apiGet(`/api/metrics/timeseries?${params}`);
      
      if (result.success && result.data && result.data.series) {
        // Combine multiple metric series into single data points
        const combinedData = combineMetricSeries(result.data.series);
        setData(combinedData);
        setError(null);
        setLastUpdate(new Date());
      } else {
        throw new Error(result.message || 'Invalid response format');
      }
    } catch (err) {
      console.error('Error fetching multi-metric data:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  };

  const combineMetricSeries = (series: Record<string, any[]>): MultiMetricDataPoint[] => {
    const timePointMap = new Map<string, MultiMetricDataPoint>();
    
    // Process each metric series
    Object.entries(series).forEach(([metricName, points]) => {
      points.forEach(point => {
        const timestamp = point.timestamp;
        
        if (!timePointMap.has(timestamp)) {
          timePointMap.set(timestamp, {
            timestamp,
            formattedTime: formatTimestamp(timestamp)
          });
        }
        
        const dataPoint = timePointMap.get(timestamp)!;
        dataPoint[metricName] = Number(point.value.toFixed(2));
      });
    });
    
    // Convert map to array and sort by timestamp
    return Array.from(timePointMap.values()).sort((a, b) => 
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );
  };

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  };

  const formatTooltipValue = (value: number, unit?: string): string => {
    let formatted: string;
    
    if (value >= 1000000) {
      formatted = `${(value / 1000000).toFixed(1)}M`;
    } else if (value >= 1000) {
      formatted = `${(value / 1000).toFixed(1)}K`;
    } else if (value < 1 && value > 0) {
      formatted = value.toFixed(3);
    } else {
      formatted = value.toFixed(1);
    }
    
    return unit ? `${formatted} ${unit}` : formatted;
  };

  const customTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 border border-gray-200 rounded-lg shadow-lg">
          <p className="font-medium text-gray-900 mb-2">{label}</p>
          {payload.map((entry: any, index: number) => {
            const metric = metrics.find(m => m.name === entry.dataKey);
            return (
              <p key={index} className="text-sm" style={{ color: entry.color }}>
                {`${entry.dataKey} (${aggregation}): ${formatTooltipValue(entry.value, metric?.unit)}`}
              </p>
            );
          })}
        </div>
      );
    }
    return null;
  };

  const getRightAxisMetrics = () => metrics.filter(m => m.yAxisId === 'right');
  const hasRightAxis = getRightAxisMetrics().length > 0;

  if (loading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
        </div>
        <div 
          style={{ height }}
          className="flex items-center justify-center bg-gray-50 rounded-lg"
        >
          <div className="text-gray-400">Loading chart data...</div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          <div className="w-4 h-4 rounded-full bg-red-500"></div>
        </div>
        <div 
          style={{ height }}
          className="flex items-center justify-center bg-red-50 rounded-lg border border-red-200"
        >
          <div className="text-center">
            <div className="text-red-600 font-medium">Failed to load chart</div>
            <div className="text-red-500 text-sm mt-1">{error}</div>
          </div>
        </div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          <div className="w-4 h-4 rounded-full bg-gray-400"></div>
        </div>
        <div 
          style={{ height }}
          className="flex items-center justify-center bg-gray-50 rounded-lg"
        >
          <div className="text-gray-400">No data available</div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          <div className="flex items-center space-x-4 mt-1">
            {metrics.map((metric) => (
              <div key={metric.name} className="flex items-center space-x-2">
                <div 
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: metric.color }}
                ></div>
                <span className="text-sm text-gray-600">{metric.name}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="flex items-center space-x-4">
          {realTime && (
            <div className="flex items-center space-x-2">
              <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
              <span className="text-sm text-green-600">Live</span>
            </div>
          )}
          {lastUpdate && (
            <span className="text-sm text-gray-500">
              Updated {lastUpdate.toLocaleTimeString()}
            </span>
          )}
        </div>
      </div>
      
      <div style={{ height }}>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart
            data={data}
            margin={{ top: 5, right: hasRightAxis ? 30 : 5, left: 20, bottom: 5 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis 
              dataKey="formattedTime" 
              stroke="#666"
              fontSize={12}
              tickMargin={10}
            />
            <YAxis 
              yAxisId="left"
              stroke="#666"
              fontSize={12}
              tickFormatter={(value: number) => formatTooltipValue(value)}
            />
            {hasRightAxis && (
              <YAxis 
                yAxisId="right"
                orientation="right"
                stroke="#666"
                fontSize={12}
                tickFormatter={(value: number) => formatTooltipValue(value)}
              />
            )}
            <Tooltip content={customTooltip} />
            <Legend />
            
            {metrics.map((metric) => (
              <Line
                key={metric.name}
                yAxisId={metric.yAxisId || 'left'}
                type="monotone"
                dataKey={metric.name}
                stroke={metric.color}
                strokeWidth={2}
                dot={{ fill: metric.color, strokeWidth: 2, r: 3 }}
                activeDot={{ r: 5, fill: metric.color }}
                connectNulls={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default MultiMetricChart;