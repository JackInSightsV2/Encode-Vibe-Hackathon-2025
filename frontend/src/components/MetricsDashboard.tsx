import React, { useState, useEffect } from 'react';
import RealTimeChart from './charts/RealTimeChart';
import MultiMetricChart from './charts/MultiMetricChart';
import SystemHealthDashboard from './SystemHealthDashboard';
import { metricsService, MetricsSummary } from '../services/metricsService';
import { CHART_COLORS } from './charts/MetricsChart';
import { performanceMonitor, usePerformanceMonitor } from '../utils/performanceMonitor';

interface DashboardState {
  timeRange: string;
  autoRefresh: boolean;
  refreshInterval: number;
  selectedMetrics: string[];
  resolution: '1m' | '5m' | '1h' | '1d';
  activeTab: 'overview' | 'system-health' | 'alerts' | 'export';
  customTimeRange: {
    start: Date | null;
    end: Date | null;
  };
  metricFilters: {
    search: string;
    categories: string[];
    thresholds: MetricThreshold[];
  };
  layoutMode: 'grid' | 'list';
  alertsEnabled: boolean;
}

interface MetricThreshold {
  metricName: string;
  warningThreshold: number;
  errorThreshold: number;
  enabled: boolean;
}

interface SystemHealthCardProps {
  title: string;
  value: number | string;
  unit?: string;
  status: 'good' | 'warning' | 'error';
}

const SystemHealthCard: React.FC<SystemHealthCardProps> = ({
  title,
  value,
  unit = '',
  status
}) => {
  const getStatusColor = () => {
    switch (status) {
      case 'good': return 'text-green-600 bg-green-100';
      case 'warning': return 'text-yellow-600 bg-yellow-100';
      case 'error': return 'text-red-600 bg-red-100';
      default: return 'text-gray-600 bg-gray-100';
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
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-600">{title}</p>
          <p className="text-2xl font-bold text-gray-900">
            {formatValue()}{unit && ` ${unit}`}
          </p>
        </div>
        <div className={`w-3 h-3 rounded-full ${getStatusColor()}`}></div>
      </div>
    </div>
  );
};

// Alerts and Thresholds Tab Component
interface AlertsTabProps {
  dashboardState: DashboardState;
  updateDashboardState: (updates: Partial<DashboardState>) => void;
  availableMetrics: Array<{ name: string; label: string; unit: string; category: string }>;
  summary: MetricsSummary | null;
}

const AlertsAndThresholdsTab: React.FC<AlertsTabProps> = ({
  dashboardState,
  updateDashboardState,
  availableMetrics,
  summary
}) => {
  const addThreshold = () => {
    const newThreshold: MetricThreshold = {
      metricName: availableMetrics[0]?.name || '',
      warningThreshold: 70,
      errorThreshold: 90,
      enabled: true,
    };
    
    updateDashboardState({
      metricFilters: {
        ...dashboardState.metricFilters,
        thresholds: [...dashboardState.metricFilters.thresholds, newThreshold],
      },
    });
  };

  const updateThreshold = (index: number, updates: Partial<MetricThreshold>) => {
    const newThresholds = [...dashboardState.metricFilters.thresholds];
    newThresholds[index] = { ...newThresholds[index], ...updates };
    
    updateDashboardState({
      metricFilters: {
        ...dashboardState.metricFilters,
        thresholds: newThresholds,
      },
    });
  };

  const removeThreshold = (index: number) => {
    const newThresholds = dashboardState.metricFilters.thresholds.filter((_, i) => i !== index);
    
    updateDashboardState({
      metricFilters: {
        ...dashboardState.metricFilters,
        thresholds: newThresholds,
      },
    });
  };

  const getAlertStatus = (metricName: string, value: number) => {
    const threshold = dashboardState.metricFilters.thresholds.find(t => t.metricName === metricName && t.enabled);
    if (!threshold) return 'good';
    
    if (value >= threshold.errorThreshold) return 'error';
    if (value >= threshold.warningThreshold) return 'warning';
    return 'good';
  };

  return (
    <div className="space-y-6">
      {/* Alert Configuration */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Metric Thresholds</h2>
          <button
            onClick={addThreshold}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
          >
            + Add Threshold
          </button>
        </div>

        <div className="space-y-4">
          {dashboardState.metricFilters.thresholds.map((threshold, index) => (
            <div key={index} className="flex items-center gap-4 p-4 bg-gray-50 rounded-lg">
              <div className="flex-1">
                <select
                  value={threshold.metricName}
                  onChange={(e) => updateThreshold(index, { metricName: e.target.value })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                >
                  {availableMetrics.map(metric => (
                    <option key={metric.name} value={metric.name}>{metric.label}</option>
                  ))}
                </select>
              </div>
              
              <div className="flex-1">
                <label className="block text-xs text-gray-600 mb-1">Warning Threshold</label>
                <input
                  type="number"
                  value={threshold.warningThreshold}
                  onChange={(e) => updateThreshold(index, { warningThreshold: parseFloat(e.target.value) })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
              
              <div className="flex-1">
                <label className="block text-xs text-gray-600 mb-1">Error Threshold</label>
                <input
                  type="number"
                  value={threshold.errorThreshold}
                  onChange={(e) => updateThreshold(index, { errorThreshold: parseFloat(e.target.value) })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
              
              <div className="flex items-center gap-2">
                <button
                  onClick={() => updateThreshold(index, { enabled: !threshold.enabled })}
                  className={`px-3 py-2 text-sm rounded-md border ${
                    threshold.enabled
                      ? 'bg-green-100 border-green-300 text-green-700'
                      : 'bg-gray-100 border-gray-300 text-gray-700'
                  }`}
                >
                  {threshold.enabled ? 'On' : 'Off'}
                </button>
                
                <button
                  onClick={() => removeThreshold(index)}
                  className="px-3 py-2 text-sm bg-red-100 border border-red-300 text-red-700 rounded-md hover:bg-red-200"
                >
                  Remove
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Current Alerts */}
      {summary && (
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Current Alert Status</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {dashboardState.metricFilters.thresholds
              .filter(t => t.enabled)
              .map((threshold, index) => {
                const metric = availableMetrics.find(m => m.name === threshold.metricName);
                const value = summary?.system_health?.memory_usage || 0; // This would be dynamic based on metric
                const status = getAlertStatus(threshold.metricName, value);
                
                return (
                  <div key={index} className={`p-4 rounded-lg border ${
                    status === 'error' ? 'bg-red-50 border-red-200' :
                    status === 'warning' ? 'bg-yellow-50 border-yellow-200' :
                    'bg-green-50 border-green-200'
                  }`}>
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="font-medium text-gray-900">{metric?.label}</h3>
                      <span className={`text-xs px-2 py-1 rounded-full ${
                        status === 'error' ? 'bg-red-100 text-red-800' :
                        status === 'warning' ? 'bg-yellow-100 text-yellow-800' :
                        'bg-green-100 text-green-800'
                      }`}>
                        {status.toUpperCase()}
                      </span>
                    </div>
                    
                    <div className="text-sm text-gray-600">
                      <div>Current: {value.toFixed(1)}{metric?.unit}</div>
                      <div>Warning: {threshold.warningThreshold}{metric?.unit}</div>
                      <div>Error: {threshold.errorThreshold}{metric?.unit}</div>
                    </div>
                  </div>
                );
              })}
          </div>
        </div>
      )}
    </div>
  );
};

// Export and Reports Tab Component
interface ExportTabProps {
  dashboardState: DashboardState;
  summary: MetricsSummary | null;
  availableMetrics: Array<{ name: string; label: string; unit: string; category: string }>;
}

const ExportAndReportsTab: React.FC<ExportTabProps> = ({
  dashboardState,
  summary,
  availableMetrics
}) => {
  const exportToCSV = async () => {
    try {
      const data = await metricsService.getMultipleMetrics(
        dashboardState.selectedMetrics,
        dashboardState.resolution,
        dashboardState.timeRange
      );
      
      const csvContent = convertToCSV(data);
      downloadFile(csvContent, 'metrics-export.csv', 'text/csv');
    } catch (error) {
      console.error('Export failed:', error);
    }
  };

  const exportToJSON = async () => {
    try {
      const data = await metricsService.getMultipleMetrics(
        dashboardState.selectedMetrics,
        dashboardState.resolution,
        dashboardState.timeRange
      );
      
      const jsonContent = JSON.stringify(data, null, 2);
      downloadFile(jsonContent, 'metrics-export.json', 'application/json');
    } catch (error) {
      console.error('Export failed:', error);
    }
  };

  const generateReport = () => {
    const reportData = {
      generatedAt: new Date().toISOString(),
      timeRange: dashboardState.timeRange,
      resolution: dashboardState.resolution,
      selectedMetrics: dashboardState.selectedMetrics,
      summary: summary,
      thresholds: dashboardState.metricFilters.thresholds,
    };
    
    const reportContent = JSON.stringify(reportData, null, 2);
    downloadFile(reportContent, 'metrics-report.json', 'application/json');
  };

  const convertToCSV = (data: Record<string, any[]>): string => {
    const headers = ['timestamp', ...Object.keys(data)];
    const rows: string[][] = [];
    
    // Get all timestamps
    const allTimestamps = new Set<string>();
    Object.values(data).forEach(series => {
      series.forEach(point => allTimestamps.add(point.timestamp));
    });
    
    // Create rows
    Array.from(allTimestamps).sort().forEach(timestamp => {
      const row: string[] = [timestamp];
      Object.keys(data).forEach(metricName => {
        const point = data[metricName].find(p => p.timestamp === timestamp);
        row.push(point?.value?.toString() || '');
      });
      rows.push(row);
    });
    
    return [headers.join(','), ...rows.map(row => row.join(','))].join('\n');
  };

  const downloadFile = (content: string, filename: string, mimeType: string) => {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* Export Options */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Export Data</h2>
        
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <button
            onClick={exportToCSV}
            className="p-4 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <div className="text-center">
              <div className="text-2xl mb-2">📊</div>
              <div className="font-medium text-gray-900">Export to CSV</div>
              <div className="text-sm text-gray-600">Spreadsheet format</div>
            </div>
          </button>
          
          <button
            onClick={exportToJSON}
            className="p-4 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <div className="text-center">
              <div className="text-2xl mb-2">📄</div>
              <div className="font-medium text-gray-900">Export to JSON</div>
              <div className="text-sm text-gray-600">Raw data format</div>
            </div>
          </button>
          
          <button
            onClick={generateReport}
            className="p-4 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <div className="text-center">
              <div className="text-2xl mb-2">📋</div>
              <div className="font-medium text-gray-900">Generate Report</div>
              <div className="text-sm text-gray-600">Full report with metadata</div>
            </div>
          </button>
        </div>
      </div>

      {/* Export Configuration */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Export Configuration</h2>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Selected Metrics</label>
            <div className="flex flex-wrap gap-2">
              {dashboardState.selectedMetrics.map(metricName => {
                const metric = availableMetrics.find(m => m.name === metricName);
                return (
                  <span key={metricName} className="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm">
                    {metric?.label || metricName}
                  </span>
                );
              })}
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Time Range</label>
              <div className="text-sm text-gray-600">{dashboardState.timeRange}</div>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Resolution</label>
              <div className="text-sm text-gray-600">{dashboardState.resolution}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Summary Statistics */}
      {summary && (
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Summary Statistics</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="text-center p-4 bg-gray-50 rounded-lg">
              <div className="text-2xl font-bold text-gray-900">{summary.total_requests.toLocaleString()}</div>
              <div className="text-sm text-gray-600">Total Requests</div>
            </div>
            
            <div className="text-center p-4 bg-gray-50 rounded-lg">
              <div className="text-2xl font-bold text-gray-900">{summary.requests_per_second.toFixed(1)}</div>
              <div className="text-sm text-gray-600">Requests/Second</div>
            </div>
            
            <div className="text-center p-4 bg-gray-50 rounded-lg">
              <div className="text-2xl font-bold text-gray-900">{summary.error_rate.toFixed(2)}%</div>
              <div className="text-sm text-gray-600">Error Rate</div>
            </div>
            
            <div className="text-center p-4 bg-gray-50 rounded-lg">
              <div className="text-2xl font-bold text-gray-900">{(summary.system_health.uptime_seconds / 3600).toFixed(1)}h</div>
              <div className="text-sm text-gray-600">Uptime</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const MetricsDashboard: React.FC = () => {
  const [dashboardState, setDashboardState] = useState<DashboardState>({
    timeRange: '1h',
    autoRefresh: true,
    refreshInterval: 30000, // 30 seconds
    selectedMetrics: ['http_request', 'system_memory', 'system_goroutines'],
    resolution: '1m',
    activeTab: 'overview',
    customTimeRange: {
      start: null,
      end: null,
    },
    metricFilters: {
      search: '',
      categories: [],
      thresholds: [
        { metricName: 'system_memory', warningThreshold: 70, errorThreshold: 90, enabled: true },
        { metricName: 'system_cpu_usage', warningThreshold: 70, errorThreshold: 90, enabled: true },
        { metricName: 'http_request', warningThreshold: 100, errorThreshold: 500, enabled: true },
      ],
    },
    layoutMode: 'grid',
    alertsEnabled: true,
  });

  const [summary, setSummary] = useState<MetricsSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  
  // Performance monitoring
  const { startOperation, endOperation } = usePerformanceMonitor('dashboard_fetch_summary');

  const availableMetrics = [
    { name: 'http_request', label: 'HTTP Requests', unit: 'req/s', category: 'HTTP' },
    { name: 'system_memory', label: 'Memory Usage', unit: 'MB', category: 'System' },
    { name: 'system_goroutines', label: 'Goroutines', unit: '', category: 'System' },
    { name: 'system_cpu_usage', label: 'CPU Usage', unit: '%', category: 'System' },
    { name: 'moderation_event', label: 'Moderation Events', unit: '', category: 'Security' },
    { name: 'pii_detection', label: 'PII Detections', unit: '', category: 'Security' },
    { name: 'provider_health_score', label: 'Provider Health', unit: '', category: 'Providers' },
    { name: 'provider_response_time', label: 'Provider Response Time', unit: 'ms', category: 'Providers' },
  ];

  const timeRangeOptions = [
    { value: '15m', label: 'Last 15 minutes', minutes: 15 },
    { value: '1h', label: 'Last hour', minutes: 60 },
    { value: '6h', label: 'Last 6 hours', minutes: 360 },
    { value: '24h', label: 'Last 24 hours', minutes: 1440 },
    { value: '7d', label: 'Last 7 days', minutes: 10080 },
    { value: '30d', label: 'Last 30 days', minutes: 43200 },
    { value: 'custom', label: 'Custom Range', minutes: 0 },
  ];

  const refreshIntervalOptions = [
    { value: 5000, label: '5 seconds' },
    { value: 10000, label: '10 seconds' },
    { value: 30000, label: '30 seconds' },
    { value: 60000, label: '1 minute' },
    { value: 300000, label: '5 minutes' },
    { value: 0, label: 'Manual only' },
  ];

  // Time ranges are defined in the metrics service
  // const timeRanges = metricsService.getTimeRanges();

  useEffect(() => {
    fetchSummary();
    const interval = setInterval(fetchSummary, 60000); // Update summary every minute
    return () => clearInterval(interval);
  }, [dashboardState.timeRange]);

  const fetchSummary = async () => {
    const timerId = startOperation();
    try {
      setIsLoading(true);
      performanceMonitor.monitorDashboardRefresh(dashboardState.selectedMetrics, dashboardState.timeRange);
      
      const summaryData = await metricsService.getSummary(dashboardState.timeRange);
      setSummary(summaryData);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch summary:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch summary');
    } finally {
      setIsLoading(false);
      endOperation(timerId);
    }
  };

  const updateDashboardState = (updates: Partial<DashboardState>) => {
    setDashboardState(prev => ({ ...prev, ...updates }));
  };

  const toggleMetric = (metricName: string) => {
    const newMetrics = dashboardState.selectedMetrics.includes(metricName)
      ? dashboardState.selectedMetrics.filter(m => m !== metricName)
      : [...dashboardState.selectedMetrics, metricName];
    
    updateDashboardState({ selectedMetrics: newMetrics });
  };

  const getSystemHealthStatus = (value: number, thresholds: { warning: number; error: number }): 'good' | 'warning' | 'error' => {
    if (value >= thresholds.error) return 'error';
    if (value >= thresholds.warning) return 'warning';
    return 'good';
  };

  return (
    <div className="space-y-6">
      {/* Dashboard Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-gray-900">Real-Time Metrics Dashboard</h1>
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <div className={`w-3 h-3 rounded-full ${dashboardState.autoRefresh ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`}></div>
              <span className="text-sm text-gray-600">
                {dashboardState.autoRefresh ? 'Live' : 'Paused'}
              </span>
            </div>
            
            {/* Performance Indicator - Simplified for now */}
            <div className="flex items-center space-x-2">
              <div className="w-3 h-3 rounded-full bg-green-500"></div>
              <span className="text-sm text-gray-600">
                Performance: Good
              </span>
            </div>
          </div>
        </div>
        
        {/* Advanced Dashboard Controls */}
        <div className="grid grid-cols-1 md:grid-cols-6 gap-4">
          {/* Time Range Selector */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Time Range</label>
            <select
              value={dashboardState.timeRange}
              onChange={(e) => updateDashboardState({ timeRange: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              {timeRangeOptions.map(range => (
                <option key={range.value} value={range.value}>{range.label}</option>
              ))}
            </select>
          </div>

          {/* Resolution Selector */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Resolution</label>
            <select
              value={dashboardState.resolution}
              onChange={(e) => updateDashboardState({ resolution: e.target.value as any })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              <option value="1m">1 minute</option>
              <option value="5m">5 minutes</option>
              <option value="1h">1 hour</option>
              <option value="1d">1 day</option>
            </select>
          </div>

          {/* Auto Refresh Toggle */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Auto Refresh</label>
            <button
              onClick={() => updateDashboardState({ autoRefresh: !dashboardState.autoRefresh })}
              className={`w-full px-3 py-2 text-sm rounded-md border ${
                dashboardState.autoRefresh
                  ? 'bg-green-100 border-green-300 text-green-700'
                  : 'bg-gray-100 border-gray-300 text-gray-700'
              }`}
            >
              {dashboardState.autoRefresh ? 'On' : 'Off'}
            </button>
          </div>

          {/* Refresh Interval */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Refresh Rate</label>
            <select
              value={dashboardState.refreshInterval}
              onChange={(e) => updateDashboardState({ refreshInterval: parseInt(e.target.value) })}
              disabled={!dashboardState.autoRefresh}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm disabled:bg-gray-100 disabled:text-gray-400"
            >
              {refreshIntervalOptions.map(option => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>

          {/* Layout Mode */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Layout</label>
            <select
              value={dashboardState.layoutMode}
              onChange={(e) => updateDashboardState({ layoutMode: e.target.value as 'grid' | 'list' })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              <option value="grid">Grid View</option>
              <option value="list">List View</option>
            </select>
          </div>

          {/* Actions */}
          <div className="flex gap-2">
            <button
              onClick={fetchSummary}
              className="flex-1 px-3 py-2 text-sm bg-blue-100 border border-blue-300 text-blue-700 rounded-md hover:bg-blue-200 transition-colors"
              title="Refresh Now"
            >
              🔄
            </button>
            <button
              onClick={() => updateDashboardState({ alertsEnabled: !dashboardState.alertsEnabled })}
              className={`flex-1 px-3 py-2 text-sm rounded-md border transition-colors ${
                dashboardState.alertsEnabled
                  ? 'bg-yellow-100 border-yellow-300 text-yellow-700'
                  : 'bg-gray-100 border-gray-300 text-gray-700'
              }`}
              title="Toggle Alerts"
            >
              🔔
            </button>
          </div>
        </div>

        {/* Custom Time Range Picker */}
        {dashboardState.timeRange === 'custom' && (
          <div className="mt-4 p-4 bg-gray-50 rounded-lg">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Start Date & Time</label>
                <input
                  type="datetime-local"
                  value={dashboardState.customTimeRange.start?.toISOString().slice(0, 16) || ''}
                  onChange={(e) => updateDashboardState({
                    customTimeRange: {
                      ...dashboardState.customTimeRange,
                      start: e.target.value ? new Date(e.target.value) : null
                    }
                  })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">End Date & Time</label>
                <input
                  type="datetime-local"
                  value={dashboardState.customTimeRange.end?.toISOString().slice(0, 16) || ''}
                  onChange={(e) => updateDashboardState({
                    customTimeRange: {
                      ...dashboardState.customTimeRange,
                      end: e.target.value ? new Date(e.target.value) : null
                    }
                  })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Advanced Tab Navigation */}
      <div className="bg-white rounded-lg border border-gray-200">
        <div className="border-b border-gray-200">
          <nav className="flex space-x-8 px-6">
            <button
              onClick={() => updateDashboardState({ activeTab: 'overview' })}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                dashboardState.activeTab === 'overview'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              📈 Metrics Overview
            </button>
            <button
              onClick={() => updateDashboardState({ activeTab: 'system-health' })}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                dashboardState.activeTab === 'system-health'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              🖥️ System Health
            </button>
            <button
              onClick={() => updateDashboardState({ activeTab: 'alerts' })}
              className={`py-4 px-1 border-b-2 font-medium text-sm relative ${
                dashboardState.activeTab === 'alerts'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              🚨 Alerts & Thresholds
              {dashboardState.alertsEnabled && (
                <span className="absolute -top-1 -right-1 w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
              )}
            </button>
            <button
              onClick={() => updateDashboardState({ activeTab: 'export' })}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                dashboardState.activeTab === 'export'
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              💾 Export & Reports
            </button>
          </nav>
        </div>
      </div>

      {/* Tab Content */}
      {dashboardState.activeTab === 'system-health' ? (
        <SystemHealthDashboard />
      ) : dashboardState.activeTab === 'alerts' ? (
        <AlertsAndThresholdsTab 
          dashboardState={dashboardState}
          updateDashboardState={updateDashboardState}
          availableMetrics={availableMetrics}
          summary={summary}
        />
      ) : dashboardState.activeTab === 'export' ? (
        <ExportAndReportsTab
          dashboardState={dashboardState}
          summary={summary}
          availableMetrics={availableMetrics}
        />
      ) : (
        <div className="space-y-6">
          {/* System Health Overview */}
      {isLoading ? (
        <div className="flex justify-center items-center p-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      ) : summary ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <SystemHealthCard
            title="Requests/Second"
            value={summary.requests_per_second}
            unit="req/s"
            status={getSystemHealthStatus(summary.requests_per_second, { warning: 100, error: 500 })}
          />
          <SystemHealthCard
            title="Error Rate"
            value={summary.error_rate}
            unit="%"
            status={getSystemHealthStatus(summary.error_rate, { warning: 5, error: 10 })}
          />
          <SystemHealthCard
            title="Memory Usage"
            value={summary.system_health.memory_usage}
            unit="MB"
            status={getSystemHealthStatus(summary.system_health.memory_usage, { warning: 500, error: 1000 })}
          />
          <SystemHealthCard
            title="Uptime"
            value={Math.floor(summary.system_health.uptime_seconds / 3600)}
            unit="hours"
            status="good"
          />
        </div>
      ) : (
        <div className="text-center p-4 text-gray-500">No data available</div>
      )}

      {/* Advanced Metric Selection */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Select Metrics</h2>
          <div className="flex items-center gap-2">
            <input
              type="text"
              placeholder="Search metrics..."
              value={dashboardState.metricFilters.search}
              onChange={(e) => updateDashboardState({
                metricFilters: {
                  ...dashboardState.metricFilters,
                  search: e.target.value
                }
              })}
              className="px-3 py-2 border border-gray-300 rounded-md text-sm w-48"
            />
            <select
              value={dashboardState.metricFilters.categories[0] || ''}
              onChange={(e) => updateDashboardState({
                metricFilters: {
                  ...dashboardState.metricFilters,
                  categories: e.target.value ? [e.target.value] : []
                }
              })}
              className="px-3 py-2 border border-gray-300 rounded-md text-sm"
            >
              <option value="">All Categories</option>
              <option value="HTTP">HTTP</option>
              <option value="System">System</option>
              <option value="Security">Security</option>
              <option value="Providers">Providers</option>
            </select>
          </div>
        </div>
        
        {/* Category Filters */}
        <div className="mb-4">
          <div className="flex flex-wrap gap-2">
            {['HTTP', 'System', 'Security', 'Providers'].map(category => {
              const isSelected = dashboardState.metricFilters.categories.includes(category);
              return (
                <button
                  key={category}
                  onClick={() => {
                    const newCategories = isSelected
                      ? dashboardState.metricFilters.categories.filter(c => c !== category)
                      : [...dashboardState.metricFilters.categories, category];
                    updateDashboardState({
                      metricFilters: {
                        ...dashboardState.metricFilters,
                        categories: newCategories
                      }
                    });
                  }}
                  className={`px-3 py-1 text-sm rounded-full border transition-colors ${
                    isSelected
                      ? 'bg-purple-100 border-purple-300 text-purple-700'
                      : 'bg-gray-100 border-gray-300 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  {category}
                </button>
              );
            })}
          </div>
        </div>
        
        {/* Filtered Metrics */}
        <div className="space-y-4">
          {(['HTTP', 'System', 'Security', 'Providers'] as const).map(category => {
            const categoryMetrics = availableMetrics.filter(metric => {
              const matchesCategory = dashboardState.metricFilters.categories.length === 0 || 
                                    dashboardState.metricFilters.categories.includes(metric.category);
              const matchesSearch = dashboardState.metricFilters.search === '' ||
                                  metric.label.toLowerCase().includes(dashboardState.metricFilters.search.toLowerCase()) ||
                                  metric.name.toLowerCase().includes(dashboardState.metricFilters.search.toLowerCase());
              return metric.category === category && matchesCategory && matchesSearch;
            });
            
            if (categoryMetrics.length === 0) return null;
            
            return (
              <div key={category}>
                <h3 className="text-sm font-medium text-gray-700 mb-2">{category} Metrics</h3>
                <div className="flex flex-wrap gap-2">
                  {categoryMetrics.map(metric => (
                    <button
                      key={metric.name}
                      onClick={() => toggleMetric(metric.name)}
                      className={`px-3 py-2 text-sm rounded-md border transition-colors ${
                        dashboardState.selectedMetrics.includes(metric.name)
                          ? 'bg-blue-100 border-blue-300 text-blue-700'
                          : 'bg-gray-100 border-gray-300 text-gray-700 hover:bg-gray-200'
                      }`}
                    >
                      {metric.label}
                      <span className="ml-1 text-xs text-gray-500">({metric.unit || 'count'})</span>
                    </button>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
        
        {/* Quick Actions */}
        <div className="mt-4 pt-4 border-t border-gray-200">
          <div className="flex gap-2">
            <button
              onClick={() => updateDashboardState({ selectedMetrics: availableMetrics.map(m => m.name) })}
              className="px-3 py-2 text-sm bg-green-100 border border-green-300 text-green-700 rounded-md hover:bg-green-200 transition-colors"
            >
              Select All
            </button>
            <button
              onClick={() => updateDashboardState({ selectedMetrics: [] })}
              className="px-3 py-2 text-sm bg-red-100 border border-red-300 text-red-700 rounded-md hover:bg-red-200 transition-colors"
            >
              Clear All
            </button>
            <button
              onClick={() => updateDashboardState({ selectedMetrics: ['http_request', 'system_memory', 'system_goroutines'] })}
              className="px-3 py-2 text-sm bg-blue-100 border border-blue-300 text-blue-700 rounded-md hover:bg-blue-200 transition-colors"
            >
              Default Selection
            </button>
          </div>
        </div>
      </div>

      {/* Individual Metric Charts */}
      <div className={`${
        dashboardState.layoutMode === 'grid' 
          ? 'grid grid-cols-1 lg:grid-cols-2 gap-6' 
          : 'space-y-6'
      }`}>
        {dashboardState.selectedMetrics.map((metricName, index) => {
          const metric = availableMetrics.find(m => m.name === metricName);
          if (!metric) return null;

          const colors = Object.values(CHART_COLORS);
          const color = colors[index % colors.length];
          
          // Check if metric has alert thresholds
          const threshold = dashboardState.metricFilters.thresholds.find(t => 
            t.metricName === metricName && t.enabled
          );

          return (
            <div key={metricName} className={`${
              dashboardState.layoutMode === 'list' ? 'w-full' : ''
            }`}>
              {threshold && dashboardState.alertsEnabled && (
                <div className="mb-2 px-4 py-2 bg-yellow-50 border border-yellow-200 rounded-lg text-sm">
                  <span className="font-medium text-yellow-800">
                    Alert thresholds: Warning {threshold.warningThreshold}{metric.unit}, Error {threshold.errorThreshold}{metric.unit}
                  </span>
                </div>
              )}
              <RealTimeChart
                metricName={metricName}
                title={`${metric.label} (${metric.category})`}
                type="line"
                color={color}
                height={dashboardState.layoutMode === 'list' ? 200 : 300}
                refreshInterval={dashboardState.autoRefresh ? dashboardState.refreshInterval : 0}
                resolution={dashboardState.resolution}
                aggregation="avg"
                unit={metric.unit}
                timeRange={dashboardState.timeRange}
              />
            </div>
          );
        })}
      </div>

      {/* Multi-Metric Comparison Chart */}
      {dashboardState.selectedMetrics.length > 1 && (
        <MultiMetricChart
          metrics={dashboardState.selectedMetrics.map((name, index) => {
            const metric = availableMetrics.find(m => m.name === name);
            const colors = Object.values(CHART_COLORS);
            return {
              name,
              color: colors[index % colors.length],
              unit: metric?.unit || '',
              yAxisId: metric?.unit === 'MB' ? 'right' : 'left' // Put memory on right axis
            };
          })}
          title="Metrics Comparison"
          height={400}
          resolution={dashboardState.resolution}
          aggregation="avg"
          timeRange={dashboardState.timeRange}
          refreshInterval={dashboardState.autoRefresh ? dashboardState.refreshInterval : 0}
          realTime={dashboardState.autoRefresh}
        />
      )}

      {/* System Health Detailed Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RealTimeChart
          metricName="http_request"
          title="HTTP Request Rate"
          type="area"
          color={CHART_COLORS.primary}
          height={300}
          refreshInterval={dashboardState.autoRefresh ? dashboardState.refreshInterval : 0}
          resolution={dashboardState.resolution}
          aggregation="count"
          unit="requests"
          timeRange={dashboardState.timeRange}
        />
        
        <RealTimeChart
          metricName="system_memory"
          title="Memory Usage Trend"
          type="line"
          color={CHART_COLORS.secondary}
          height={300}
          refreshInterval={dashboardState.autoRefresh ? dashboardState.refreshInterval : 0}
          resolution={dashboardState.resolution}
          aggregation="avg"
          unit="MB"
          timeRange={dashboardState.timeRange}
        />
      </div>

      {/* Error Handling */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center">
            <div className="w-5 h-5 rounded-full bg-red-500 mr-3"></div>
            <div>
              <h3 className="text-red-800 font-medium">Dashboard Error</h3>
              <p className="text-red-600 text-sm mt-1">{error}</p>
            </div>
          </div>
        </div>
          )}
        </div>
      )}
    </div>
  );
};

export default MetricsDashboard;