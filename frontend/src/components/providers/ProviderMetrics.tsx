import React, { useState, useEffect } from 'react';
import { 
  ClockIcon, 
  CheckCircleIcon, 
  CurrencyDollarIcon,
  ArrowTrendingUpIcon as TrendingUpIcon,
  ArrowTrendingDownIcon as TrendingDownIcon
} from '@heroicons/react/24/outline';
import { Card } from '../ui/Card';
import { Button } from '../ui/Button';
import { providerService } from '../../services/providerService';

interface Provider {
  name: string;
  type: string;
  enabled: boolean;
  health: number;
  averageLatency: number;
  models: string[];
  version: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  config?: any;
}

interface ProviderMetricsProps {
  provider: Provider;
}

interface MetricCardProps {
  title: string;
  value: string;
  change?: number;
  icon: React.ReactNode;
  trend?: 'up' | 'down' | 'stable';
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, change, icon, trend }) => {
  const getTrendColor = () => {
    if (!trend || trend === 'stable') return 'text-gray-500 dark:text-gray-400';
    return trend === 'up' ? 'text-green-500' : 'text-red-500';
  };

  const getTrendIcon = () => {
    if (!trend || trend === 'stable') return null;
    return trend === 'up' ? 
      <TrendingUpIcon className="w-4 h-4" /> : 
      <TrendingDownIcon className="w-4 h-4" />;
  };

  return (
    <Card className="p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400">{title}</p>
          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">{value}</p>
          {change !== undefined && (
            <div className={`flex items-center space-x-1 mt-1 ${getTrendColor()}`}>
              {getTrendIcon()}
              <span className="text-sm">
                {change > 0 ? '+' : ''}{change}%
              </span>
            </div>
          )}
        </div>
        <div className="p-3 bg-gray-100 dark:bg-gray-700 rounded-lg">
          {icon}
        </div>
      </div>
    </Card>
  );
};

export const ProviderMetrics: React.FC<ProviderMetricsProps> = ({ provider }) => {
  const [timeRange, setTimeRange] = useState('1h');
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMetrics = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await providerService.getProviderMetrics(provider.name, timeRange);
      setMetrics(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
      console.error('Failed to fetch metrics:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
  }, [provider.name, timeRange]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-red-600 dark:text-red-400">{error}</p>
        <Button onClick={fetchMetrics} variant="outline" className="mt-4">
          Retry
        </Button>
      </div>
    );
  }

  if (!metrics) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">No metrics data available</p>
        <Button onClick={fetchMetrics} variant="outline" className="mt-4">
          Refresh
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Performance Metrics
        </h3>
        <select
          value={timeRange}
          onChange={(e) => setTimeRange(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
        >
          <option value="1h">Last Hour</option>
          <option value="6h">Last 6 Hours</option>
          <option value="24h">Last 24 Hours</option>
          <option value="7d">Last 7 Days</option>
        </select>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          title="Requests"
          value={metrics.totalRequests?.toString() || '0'}
          change={metrics.requestsChange}
          icon={<div className="w-6 h-6 text-blue-600">📊</div>}
          trend={metrics.requestsChange > 0 ? 'up' : metrics.requestsChange < 0 ? 'down' : 'stable'}
        />
        <MetricCard
          title="Avg Latency"
          value={`${metrics.averageLatency || 0}ms`}
          change={metrics.latencyChange}
          icon={<ClockIcon className="w-6 h-6 text-yellow-600" />}
          trend={metrics.latencyChange < 0 ? 'up' : metrics.latencyChange > 0 ? 'down' : 'stable'}
        />
        <MetricCard
          title="Success Rate"
          value={`${((metrics.successRate || 0) * 100).toFixed(1)}%`}
          change={metrics.successRateChange}
          icon={<CheckCircleIcon className="w-6 h-6 text-green-600" />}
          trend={metrics.successRateChange > 0 ? 'up' : metrics.successRateChange < 0 ? 'down' : 'stable'}
        />
        <MetricCard
          title="Cost"
          value={`$${(metrics.totalCost || 0).toFixed(2)}`}
          change={metrics.costChange}
          icon={<CurrencyDollarIcon className="w-6 h-6 text-purple-600" />}
          trend={metrics.costChange > 0 ? 'down' : metrics.costChange < 0 ? 'up' : 'stable'}
        />
      </div>

      {/* Health Score Visualization */}
      <Card title="Health Score">
        <div className="flex items-center space-x-4">
          <div className="flex-1">
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4">
              <div 
                className={`h-4 rounded-full ${
                  provider.health >= 0.8 ? 'bg-green-500' :
                  provider.health >= 0.6 ? 'bg-yellow-500' :
                  'bg-red-500'
                }`}
                style={{ width: `${provider.health * 100}%` }}
              />
            </div>
          </div>
          <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {(provider.health * 100).toFixed(0)}%
          </div>
        </div>
        <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
          <div className="text-center">
            <div className="text-green-600 dark:text-green-400 font-medium">Excellent</div>
            <div className="text-gray-500 dark:text-gray-400">80-100%</div>
          </div>
          <div className="text-center">
            <div className="text-yellow-600 dark:text-yellow-400 font-medium">Good</div>
            <div className="text-gray-500 dark:text-gray-400">60-79%</div>
          </div>
          <div className="text-center">
            <div className="text-red-600 dark:text-red-400 font-medium">Poor</div>
            <div className="text-gray-500 dark:text-gray-400">0-59%</div>
          </div>
        </div>
      </Card>

      {/* Status History */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card title="Request Volume Trend">
          <div className="h-48 flex items-center justify-center text-gray-500 dark:text-gray-400">
            <div className="text-center">
              <div className="w-12 h-12 mx-auto mb-2 opacity-50 text-4xl">📊</div>
              <p>Chart visualization would be here</p>
              <p className="text-sm">Request volume over {timeRange}</p>
            </div>
          </div>
        </Card>

        <Card title="Response Time Trend">
          <div className="h-48 flex items-center justify-center text-gray-500 dark:text-gray-400">
            <div className="text-center">
              <ClockIcon className="w-12 h-12 mx-auto mb-2 opacity-50" />
              <p>Chart visualization would be here</p>
              <p className="text-sm">Response time over {timeRange}</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Detailed Stats */}
      <Card title="Detailed Statistics">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {metrics.totalRequests || 0}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Total Requests</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {Math.round((metrics.totalRequests || 0) * (metrics.successRate || 0))}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Successful</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {Math.round((metrics.totalRequests || 0) * (1 - (metrics.successRate || 1)))}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Failed</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {((metrics.totalRequests || 0) / (timeRange === '1h' ? 3600 : timeRange === '6h' ? 21600 : timeRange === '24h' ? 86400 : 604800)).toFixed(2)}
            </div>
            <div className="text-sm text-gray-600 dark:text-gray-400">Req/sec avg</div>
          </div>
        </div>
      </Card>
    </div>
  );
};