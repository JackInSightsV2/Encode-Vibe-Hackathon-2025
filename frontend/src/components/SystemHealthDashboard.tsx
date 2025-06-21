import React, { useState, useEffect } from 'react';
import { metricsService } from '../services/metricsService';
import RealTimeChart from './charts/RealTimeChart';
import { CHART_COLORS } from './charts/MetricsChart';

interface SystemSnapshot {
  timestamp: number;
  cpu_usage: number;
  memory_usage: number;
  memory_allocated: number;
  goroutine_count: number;
  uptime_seconds: number;
  providers: Record<string, {
    is_healthy: boolean;
    health_score: number;
    error_rate: number;
  }>;
}

interface HealthCardProps {
  title: string;
  value: number | string;
  unit?: string;
  status: 'good' | 'warning' | 'error';
  icon?: string;
  trend?: 'up' | 'down' | 'stable';
}

interface ProviderCardProps {
  name: string;
  health: {
    is_healthy: boolean;
    health_score: number;
    error_rate: number;
  };
}

const HealthCard: React.FC<HealthCardProps> = ({
  title,
  value,
  unit = '',
  status,
  icon = '',
  trend = 'stable'
}) => {
  const getStatusColor = () => {
    switch (status) {
      case 'good': return 'text-green-600 bg-green-100 border-green-200';
      case 'warning': return 'text-yellow-600 bg-yellow-100 border-yellow-200';
      case 'error': return 'text-red-600 bg-red-100 border-red-200';
      default: return 'text-gray-600 bg-gray-100 border-gray-200';
    }
  };

  const getTrendIcon = () => {
    switch (trend) {
      case 'up': return '↗';
      case 'down': return '↘';
      default: return '→';
    }
  };

  const formatValue = () => {
    if (typeof value === 'number') {
      if (value >= 1000000) {
        return `${(value / 1000000).toFixed(1)}M`;
      } else if (value >= 1000) {
        return `${(value / 1000).toFixed(1)}K`;
      } else if (value < 1 && value > 0) {
        return value.toFixed(2);
      } else {
        return value.toFixed(0);
      }
    }
    return value;
  };

  return (
    <div className={`bg-white rounded-lg border p-4 ${getStatusColor()}`}>
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2">
            {icon && <span className="text-lg">{icon}</span>}
            <p className="text-sm font-medium text-gray-700">{title}</p>
          </div>
          <div className="flex items-baseline space-x-2 mt-1">
            <p className="text-2xl font-bold text-gray-900">
              {formatValue()}{unit && ` ${unit}`}
            </p>
            <span className="text-sm text-gray-500">{getTrendIcon()}</span>
          </div>
        </div>
        <div className="flex flex-col items-center">
          <div className={`w-4 h-4 rounded-full ${status === 'good' ? 'bg-green-500' : status === 'warning' ? 'bg-yellow-500' : 'bg-red-500'}`}></div>
          <span className="text-xs text-gray-500 mt-1 capitalize">{status}</span>
        </div>
      </div>
    </div>
  );
};

const ProviderCard: React.FC<ProviderCardProps> = ({ name, health }) => {
  const getHealthStatus = (): 'good' | 'warning' | 'error' => {
    if (!health.is_healthy || health.health_score < 0.7) return 'error';
    if (health.health_score < 0.9 || health.error_rate > 0.05) return 'warning';
    return 'good';
  };

  const status = getHealthStatus();

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-medium text-gray-900 capitalize">{name}</h3>
        <div className={`w-3 h-3 rounded-full ${
          status === 'good' ? 'bg-green-500' : 
          status === 'warning' ? 'bg-yellow-500' : 'bg-red-500'
        }`}></div>
      </div>
      
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-gray-600">Health Score</span>
          <span className="font-medium">{(health.health_score * 100).toFixed(1)}%</span>
        </div>
        
        <div className="flex justify-between text-sm">
          <span className="text-gray-600">Error Rate</span>
          <span className="font-medium">{(health.error_rate * 100).toFixed(2)}%</span>
        </div>
        
        <div className="flex justify-between text-sm">
          <span className="text-gray-600">Status</span>
          <span className={`font-medium capitalize ${
            health.is_healthy ? 'text-green-600' : 'text-red-600'
          }`}>
            {health.is_healthy ? 'Healthy' : 'Unhealthy'}
          </span>
        </div>
      </div>
    </div>
  );
};

const SystemHealthDashboard: React.FC = () => {
  const [snapshot, setSnapshot] = useState<SystemSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

  useEffect(() => {
    let cleanup: (() => void) | null = null;

    if (autoRefresh) {
      cleanup = metricsService.startSystemHealthPolling((data) => {
        setSnapshot(data);
        setLoading(false);
        setError(null);
      }, 10000); // Poll every 10 seconds
    } else {
      // Fetch once if auto-refresh is disabled
      fetchSnapshot();
    }

    return () => {
      if (cleanup) cleanup();
    };
  }, [autoRefresh]);

  const fetchSnapshot = async () => {
    try {
      setLoading(true);
      const data = await metricsService.getSystemSnapshot();
      setSnapshot(data);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch system snapshot:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch system health data');
    } finally {
      setLoading(false);
    }
  };

  const getMemoryStatus = (usage: number): 'good' | 'warning' | 'error' => {
    if (usage > 80) return 'error';
    if (usage > 60) return 'warning';
    return 'good';
  };

  const getCPUStatus = (usage: number): 'good' | 'warning' | 'error' => {
    if (usage > 80) return 'error';
    if (usage > 60) return 'warning';
    return 'good';
  };

  const getGoroutineStatus = (count: number): 'good' | 'warning' | 'error' => {
    if (count > 1000) return 'error';
    if (count > 500) return 'warning';
    return 'good';
  };

  const formatUptime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (hours > 24) {
      const days = Math.floor(hours / 24);
      return `${days}d ${hours % 24}h`;
    } else if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else {
      return `${minutes}m`;
    }
  };

  if (loading && !snapshot) {
    return (
      <div className="space-y-6">
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <div className="flex items-center space-x-2">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
            <h1 className="text-2xl font-bold text-gray-900">Loading System Health...</h1>
          </div>
        </div>
      </div>
    );
  }

  if (error && !snapshot) {
    return (
      <div className="space-y-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6">
          <div className="flex items-center">
            <div className="w-5 h-5 rounded-full bg-red-500 mr-3"></div>
            <div>
              <h3 className="text-red-800 font-medium">System Health Error</h3>
              <p className="text-red-600 text-sm mt-1">{error}</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Dashboard Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-gray-900">System Health Dashboard</h1>
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <div className={`w-3 h-3 rounded-full ${autoRefresh ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`}></div>
              <span className="text-sm text-gray-600">
                {autoRefresh ? 'Live Updates' : 'Manual Mode'}
              </span>
            </div>
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`px-3 py-2 text-sm rounded-md border transition-colors ${
                autoRefresh
                  ? 'bg-green-100 border-green-300 text-green-700'
                  : 'bg-gray-100 border-gray-300 text-gray-700'
              }`}
            >
              {autoRefresh ? 'Disable Auto-refresh' : 'Enable Auto-refresh'}
            </button>
            <button
              onClick={fetchSnapshot}
              className="px-3 py-2 text-sm bg-blue-100 border border-blue-300 text-blue-700 rounded-md hover:bg-blue-200 transition-colors"
            >
              Refresh Now
            </button>
          </div>
        </div>
        
        {snapshot && (
          <div className="text-sm text-gray-500">
            Last updated: {new Date(snapshot.timestamp * 1000).toLocaleString()}
          </div>
        )}
      </div>

      {snapshot && (
        <>
          {/* System Health Overview Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <HealthCard
              title="CPU Usage"
              value={snapshot.cpu_usage}
              unit="%"
              status={getCPUStatus(snapshot.cpu_usage)}
              icon="⚡"
            />
            <HealthCard
              title="Memory Usage"
              value={snapshot.memory_usage}
              unit="%"
              status={getMemoryStatus(snapshot.memory_usage)}
              icon="💾"
            />
            <HealthCard
              title="Goroutines"
              value={snapshot.goroutine_count}
              status={getGoroutineStatus(snapshot.goroutine_count)}
              icon="🔧"
            />
            <HealthCard
              title="Uptime"
              value={formatUptime(snapshot.uptime_seconds)}
              status="good"
              icon="⏱️"
            />
          </div>

          {/* Memory Details */}
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Memory Details</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-600">Allocated Memory</span>
                  <span className="font-medium">{(snapshot.memory_allocated / 1024 / 1024).toFixed(1)} MB</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Memory Usage %</span>
                  <span className="font-medium">{snapshot.memory_usage.toFixed(1)}%</span>
                </div>
              </div>
            </div>
          </div>

          {/* Provider Health Status */}
          {snapshot.providers && Object.keys(snapshot.providers).length > 0 && (
            <div className="bg-white rounded-lg border border-gray-200 p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">AI Provider Health</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {Object.entries(snapshot.providers).map(([name, health]) => (
                  <ProviderCard key={name} name={name} health={health} />
                ))}
              </div>
            </div>
          )}

          {/* Real-time System Metrics Charts */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <RealTimeChart
              metricName="system_cpu_usage"
              title="CPU Usage Over Time"
              type="area"
              color={CHART_COLORS.danger}
              height={300}
              refreshInterval={autoRefresh ? 30000 : 0}
              resolution="1m"
              aggregation="avg"
              unit="%"
              timeRange="1h"
            />
            
            <RealTimeChart
              metricName="system_memory_usage"
              title="Memory Usage Over Time"
              type="line"
              color={CHART_COLORS.primary}
              height={300}
              refreshInterval={autoRefresh ? 30000 : 0}
              resolution="1m"
              aggregation="avg"
              unit="%"
              timeRange="1h"
            />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <RealTimeChart
              metricName="system_goroutines"
              title="Goroutine Count"
              type="line"
              color={CHART_COLORS.secondary}
              height={300}
              refreshInterval={autoRefresh ? 30000 : 0}
              resolution="1m"
              aggregation="avg"
              unit=""
              timeRange="1h"
            />
            
            <RealTimeChart
              metricName="system_memory_allocated"
              title="Allocated Memory"
              type="area"
              color={CHART_COLORS.purple}
              height={300}
              refreshInterval={autoRefresh ? 30000 : 0}
              resolution="1m"
              aggregation="avg"
              unit="bytes"
              timeRange="1h"
            />
          </div>

          {/* Provider Health Charts */}
          {snapshot.providers && Object.keys(snapshot.providers).length > 0 && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <RealTimeChart
                metricName="provider_health_score"
                title="Provider Health Scores"
                type="line"
                color={CHART_COLORS.teal}
                height={300}
                refreshInterval={autoRefresh ? 30000 : 0}
                resolution="1m"
                aggregation="avg"
                unit=""
                timeRange="1h"
              />
              
              <RealTimeChart
                metricName="provider_error_rate"
                title="Provider Error Rates"
                type="area"
                color={CHART_COLORS.orange}
                height={300}
                refreshInterval={autoRefresh ? 30000 : 0}
                resolution="1m"
                aggregation="avg"
                unit="%"
                timeRange="1h"
              />
            </div>
          )}
        </>
      )}

      {/* Error Display */}
      {error && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <div className="flex items-center">
            <div className="w-5 h-5 rounded-full bg-yellow-500 mr-3"></div>
            <div>
              <h3 className="text-yellow-800 font-medium">Update Warning</h3>
              <p className="text-yellow-600 text-sm mt-1">{error}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SystemHealthDashboard;