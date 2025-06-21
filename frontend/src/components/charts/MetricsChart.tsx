import React, { useState, useEffect } from 'react';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

export interface MetricDataPoint {
  timestamp: string;
  value: number;
  formattedTime?: string;
}

export interface MetricsChartProps {
  data: MetricDataPoint[];
  type: 'line' | 'area' | 'bar';
  title: string;
  metricName: string;
  color?: string;
  height?: number;
  realTime?: boolean;
  loading?: boolean;
  error?: string;
  unit?: string;
  aggregation?: string;
}

const CHART_COLORS = {
  primary: '#3B82F6',
  secondary: '#10B981', 
  accent: '#F59E0B',
  danger: '#EF4444',
  purple: '#8B5CF6',
  pink: '#EC4899',
  teal: '#14B8A6',
  orange: '#F97316',
};

const MetricsChart: React.FC<MetricsChartProps> = ({
  data,
  type,
  title,
  metricName,
  color = CHART_COLORS.primary,
  height = 300,
  realTime = false,
  loading = false,
  error = null,
  unit = '',
  aggregation = 'avg'
}) => {
  const [processedData, setProcessedData] = useState<MetricDataPoint[]>([]);

  useEffect(() => {
    if (data && data.length > 0) {
      const processed = data.map(point => ({
        ...point,
        formattedTime: formatTimestamp(point.timestamp),
        value: Number(point.value.toFixed(2))
      }));
      setProcessedData(processed);
    }
  }, [data]);

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMins / 60);

    if (diffMins < 60) {
      return date.toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit' 
      });
    } else if (diffHours < 24) {
      return date.toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit' 
      });
    } else {
      return date.toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    }
  };

  const formatTooltipValue = (value: number): string => {
    if (unit) {
      return `${value} ${unit}`;
    }
    
    // Auto-format based on value magnitude
    if (value >= 1000000) {
      return `${(value / 1000000).toFixed(1)}M`;
    } else if (value >= 1000) {
      return `${(value / 1000).toFixed(1)}K`;
    } else if (value < 1 && value > 0) {
      return value.toFixed(3);
    } else {
      return value.toFixed(1);
    }
  };

  const customTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      const value = payload[0].value;
      return (
        <div className="bg-white p-3 border border-gray-200 rounded-lg shadow-lg">
          <p className="font-medium text-gray-900">{label}</p>
          <p className="text-sm" style={{ color: payload[0].color }}>
            {`${metricName} (${aggregation}): ${formatTooltipValue(value)}`}
          </p>
        </div>
      );
    }
    return null;
  };

  const renderChart = (): React.ReactElement => {
    // Early returns for loading/error states
    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;
    if (!processedData || processedData.length === 0) return <div>No data available</div>;

    const commonProps = {
      data: processedData,
      margin: { top: 5, right: 30, left: 20, bottom: 5 },
    };

    switch (type) {
      case 'line':
        return (
          <LineChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis 
              dataKey="formattedTime" 
              stroke="#666"
              fontSize={12}
              tickMargin={10}
            />
            <YAxis 
              stroke="#666"
              fontSize={12}
              tickFormatter={formatTooltipValue}
            />
            <Tooltip content={customTooltip} />
            <Line
              type="monotone"
              dataKey="value"
              stroke={color}
              strokeWidth={2}
              dot={{ fill: color, strokeWidth: 2, r: realTime ? 4 : 3 }}
              activeDot={{ r: 6, fill: color }}
              connectNulls={false}
            />
          </LineChart>
        );

      case 'area':
        return (
          <AreaChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis 
              dataKey="formattedTime" 
              stroke="#666"
              fontSize={12}
              tickMargin={10}
            />
            <YAxis 
              stroke="#666"
              fontSize={12}
              tickFormatter={formatTooltipValue}
            />
            <Tooltip content={customTooltip} />
            <Area
              type="monotone"
              dataKey="value"
              stroke={color}
              fill={color}
              fillOpacity={0.3}
              strokeWidth={2}
            />
          </AreaChart>
        );

      case 'bar':
        return (
          <BarChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis 
              dataKey="formattedTime" 
              stroke="#666"
              fontSize={12}
              tickMargin={10}
            />
            <YAxis 
              stroke="#666"
              fontSize={12}
              tickFormatter={formatTooltipValue}
            />
            <Tooltip content={customTooltip} />
            <Bar
              dataKey="value"
              fill={color}
              opacity={0.8}
            />
          </BarChart>
        );

      default:
        return <div>Unsupported chart type</div>;
    }
  };





  const latestValue = processedData[processedData.length - 1]?.value;
  const dataCount = processedData.length;

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          {latestValue !== undefined && (
            <div className="flex items-center space-x-2 mt-1">
              <span className="text-2xl font-bold" style={{ color }}>
                {formatTooltipValue(latestValue)}
              </span>
              <span className="text-sm text-gray-500">
                current ({aggregation})
              </span>
            </div>
          )}
        </div>
        <div className="flex items-center space-x-4">
          {realTime && (
            <div className="flex items-center space-x-2">
              <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
              <span className="text-sm text-green-600">Live</span>
            </div>
          )}
          <span className="text-sm text-gray-500">
            {dataCount} points
          </span>
        </div>
      </div>
      
      <div style={{ height }}>
        <ResponsiveContainer width="100%" height="100%">
          {renderChart() || <div>No chart data available</div>}
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export default MetricsChart;
export { CHART_COLORS };