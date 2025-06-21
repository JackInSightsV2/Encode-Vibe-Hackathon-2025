import React, { useState, useEffect } from 'react'
import {
  ChartBarIcon,
  ChartPieIcon,
  ArrowTrendingUpIcon,
  ArrowTrendingDownIcon,
  ClockIcon,
  UserGroupIcon,
  DocumentTextIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  ArrowPathIcon,
  ArrowDownTrayIcon,
  ServerIcon,
  CalendarIcon,
  DocumentChartBarIcon
} from '@heroicons/react/24/outline'
import { Modal } from '../ui/Modal'
import { Button } from '../ui/Button'

interface AnalyticsData {
  overview: {
    total_changes: number
    unique_users: number
    environments_active: number
    avg_changes_per_day: number
    uptime_percentage: number
    config_versions: number
  }
  trends: {
    daily_changes: Array<{ date: string; count: number }>
    hourly_distribution: Array<{ hour: number; count: number }>
    weekly_pattern: Array<{ day: string; count: number }>
  }
  usage: {
    most_changed_fields: Array<{ field: string; count: number; percentage: number }>
    user_activity: Array<{ user: string; changes: number; last_active: string }>
    environment_usage: Array<{ environment: string; changes: number; deployments: number }>
  }
  errors: {
    validation_failures: number
    failed_deployments: number
    rollback_count: number
    error_rate: number
    common_errors: Array<{ type: string; count: number; last_occurred: string }>
  }
  performance: {
    avg_validation_time: number
    avg_deployment_time: number
    config_size_trend: Array<{ date: string; size_kb: number }>
    api_response_times: Array<{ endpoint: string; avg_ms: number; p95_ms: number }>
  }
}

interface TimeRange {
  label: string
  value: string
  days: number
}

interface ConfigAnalyticsProps {
  onClose: () => void
}

const ConfigAnalytics: React.FC<ConfigAnalyticsProps> = ({ onClose }) => {
  const [analyticsData, setAnalyticsData] = useState<AnalyticsData | null>(null)
  const [loading, setLoading] = useState(true)
  const [timeRange, setTimeRange] = useState<TimeRange>({ label: 'Last 7 days', value: '7d', days: 7 })
  const [activeTab, setActiveTab] = useState<'overview' | 'usage' | 'performance' | 'errors'>('overview')
  const [refreshing, setRefreshing] = useState(false)

  const timeRanges: TimeRange[] = [
    { label: 'Last 24 hours', value: '24h', days: 1 },
    { label: 'Last 7 days', value: '7d', days: 7 },
    { label: 'Last 30 days', value: '30d', days: 30 },
    { label: 'Last 90 days', value: '90d', days: 90 }
  ]

  useEffect(() => {
    fetchAnalyticsData()
  }, [timeRange])

  const fetchAnalyticsData = async () => {
    setLoading(true)
    try {
      const response = await fetch(`/api/config/analytics?range=${timeRange.value}`)
      if (response.ok) {
        const data = await response.json()
        setAnalyticsData(data.data)
      } else {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
    } catch (error) {
      console.error('Failed to fetch analytics:', error)
      setAnalyticsData(null)
    } finally {
      setLoading(false)
    }
  }

  const refreshData = async () => {
    setRefreshing(true)
    await fetchAnalyticsData()
    setRefreshing(false)
  }

  const exportReport = async () => {
    try {
      const response = await fetch('/api/config/analytics/export', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          range: timeRange.value,
          format: 'pdf'
        })
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `config-analytics-${timeRange.value}.pdf`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      }
    } catch (error) {
      console.error('Failed to export report:', error)
    }
  }

  if (loading) {
    return (
      <Modal isOpen onClose={onClose} title="Configuration Analytics" size="xl">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </Modal>
    )
  }

  if (!analyticsData) {
    return (
      <Modal isOpen onClose={onClose} title="Configuration Analytics" size="xl">
        <div className="flex flex-col items-center justify-center h-64 space-y-4">
          <div className="text-muted-foreground text-center">
            <p className="text-lg font-semibold">No Analytics Data Available</p>
            <p className="text-sm">Configuration analytics data is not available</p>
          </div>
          <Button onClick={fetchAnalyticsData} variant="outline">
            Retry
          </Button>
        </div>
      </Modal>
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              Configuration Analytics
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Insights and usage patterns for your configuration
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <select
              value={timeRange.value}
              onChange={(e) => {
                const range = timeRanges.find(r => r.value === e.target.value)
                if (range) setTimeRange(range)
              }}
              className="text-sm border border-gray-300 rounded px-3 py-1"
            >
              {timeRanges.map(range => (
                <option key={range.value} value={range.value}>
                  {range.label}
                </option>
              ))}
            </select>
            <button
              onClick={refreshData}
              disabled={refreshing}
              className="p-2 text-gray-400 hover:text-gray-600 disabled:opacity-50"
            >
              <ArrowPathIcon className={`h-5 w-5 ${refreshing ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={exportReport}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              <ArrowDownTrayIcon className="h-4 w-4 inline-block mr-2" />
              Export
            </button>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
            >
              <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Overview Stats */}
      <div className="bg-gray-50 px-6 py-4 border-b">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <StatCard
            title="Total Changes"
            value={analyticsData.overview.total_changes}
            icon={DocumentTextIcon}
            trend={calculateTrend()}
          />
          <StatCard
            title="Active Users"
            value={analyticsData.overview.unique_users}
            icon={UserGroupIcon}
          />
          <StatCard
            title="Environments"
            value={analyticsData.overview.environments_active}
            icon={ChartBarIcon}
          />
          <StatCard
            title="Avg Changes/Day"
            value={analyticsData.overview.avg_changes_per_day.toFixed(1)}
            icon={ArrowTrendingUpIcon}
          />
          <StatCard
            title="Uptime"
            value={`${analyticsData.overview.uptime_percentage}%`}
            icon={CheckCircleIcon}
            valueColor="text-green-600"
          />
          <StatCard
            title="Versions"
            value={analyticsData.overview.config_versions}
            icon={ClockIcon}
          />
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <nav className="flex space-x-8">
          {(['overview', 'usage', 'performance', 'errors'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {activeTab === 'overview' && (
          <OverviewTab data={analyticsData} />
        )}
        {activeTab === 'usage' && (
          <UsageTab data={analyticsData.usage} />
        )}
        {activeTab === 'performance' && (
          <PerformanceTab data={analyticsData.performance} />
        )}
        {activeTab === 'errors' && (
          <ErrorsTab data={analyticsData.errors} />
        )}
      </div>
    </div>
  )
}

// Tab Components
const OverviewTab: React.FC<{ data: AnalyticsData }> = ({ data }) => {
  return (
    <div className="space-y-6">
      {/* Activity Chart */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Configuration Activity</h3>
        <div className="h-64 flex items-end space-x-2">
          {data.trends.daily_changes.map((day, index) => {
            const maxCount = Math.max(...data.trends.daily_changes.map(d => d.count))
            const height = (day.count / maxCount) * 100
            
            return (
              <div
                key={index}
                className="flex-1 bg-blue-500 hover:bg-blue-600 rounded-t transition-all relative group"
                style={{ height: `${height}%` }}
              >
                <div className="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 bg-gray-800 text-white text-xs rounded px-2 py-1 opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap">
                  {day.date}: {day.count} changes
                </div>
              </div>
            )
          })}
        </div>
        <div className="flex justify-between mt-2 text-xs text-gray-500">
          <span>{data.trends.daily_changes[0]?.date}</span>
          <span>{data.trends.daily_changes[data.trends.daily_changes.length - 1]?.date}</span>
        </div>
      </div>

      {/* Patterns */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Hourly Distribution */}
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Hourly Distribution</h3>
          <div className="space-y-2">
            {data.trends.hourly_distribution.slice(9, 18).map((hour) => {
              const maxCount = Math.max(...data.trends.hourly_distribution.map(h => h.count))
              const percentage = (hour.count / maxCount) * 100
              
              return (
                <div key={hour.hour} className="flex items-center">
                  <span className="text-sm text-gray-600 w-16">
                    {hour.hour}:00
                  </span>
                  <div className="flex-1 bg-gray-200 rounded-full h-4 mr-3">
                    <div
                      className="bg-blue-500 h-4 rounded-full"
                      style={{ width: `${percentage}%` }}
                    />
                  </div>
                  <span className="text-sm text-gray-700 w-12 text-right">
                    {hour.count}
                  </span>
                </div>
              )
            })}
          </div>
        </div>

        {/* Weekly Pattern */}
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Weekly Pattern</h3>
          <div className="space-y-2">
            {data.trends.weekly_pattern.map((day) => {
              const maxCount = Math.max(...data.trends.weekly_pattern.map(d => d.count))
              const percentage = (day.count / maxCount) * 100
              
              return (
                <div key={day.day} className="flex items-center">
                  <span className="text-sm text-gray-600 w-16">
                    {day.day}
                  </span>
                  <div className="flex-1 bg-gray-200 rounded-full h-4 mr-3">
                    <div
                      className="bg-green-500 h-4 rounded-full"
                      style={{ width: `${percentage}%` }}
                    />
                  </div>
                  <span className="text-sm text-gray-700 w-12 text-right">
                    {day.count}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}

const UsageTab: React.FC<{ data: AnalyticsData['usage'] }> = ({ data }) => {
  return (
    <div className="space-y-6">
      {/* Most Changed Fields */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Most Changed Fields</h3>
        <div className="space-y-3">
          {data.most_changed_fields.map((field, index) => (
            <div key={index} className="flex items-center justify-between">
              <div className="flex items-center flex-1">
                <span className="text-sm font-mono text-gray-700 mr-3">
                  {field.field}
                </span>
                <div className="flex-1 bg-gray-200 rounded-full h-2 mr-3 max-w-xs">
                  <div
                    className="bg-blue-500 h-2 rounded-full"
                    style={{ width: `${field.percentage}%` }}
                  />
                </div>
              </div>
              <div className="text-sm text-gray-600">
                {field.count} changes ({field.percentage}%)
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* User Activity */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b">
          <h3 className="text-lg font-medium text-gray-900">User Activity</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  User
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Changes
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Last Active
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Activity
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data.user_activity.map((user, index) => {
                const maxChanges = Math.max(...data.user_activity.map(u => u.changes))
                const activityLevel = (user.changes / maxChanges) * 100
                
                return (
                  <tr key={index}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {user.user}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {user.changes}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(user.last_active).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="w-24 bg-gray-200 rounded-full h-2">
                        <div
                          className={`h-2 rounded-full ${
                            activityLevel > 75 ? 'bg-green-500' :
                            activityLevel > 50 ? 'bg-yellow-500' :
                            activityLevel > 25 ? 'bg-orange-500' :
                            'bg-red-500'
                          }`}
                          style={{ width: `${activityLevel}%` }}
                        />
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Environment Usage */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Environment Usage</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {data.environment_usage.map((env, index) => (
            <div key={index} className="bg-gray-50 rounded-lg p-4">
              <h4 className="text-sm font-medium text-gray-900 mb-2">
                {env.environment}
              </h4>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-600">Changes</span>
                  <span className="font-medium">{env.changes}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-600">Deployments</span>
                  <span className="font-medium">{env.deployments}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

const PerformanceTab: React.FC<{ data: AnalyticsData['performance'] }> = ({ data }) => {
  return (
    <div className="space-y-6">
      {/* Performance Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Avg Validation Time</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">
                {data.avg_validation_time}ms
              </p>
            </div>
            <ClockIcon className="h-8 w-8 text-gray-400" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Avg Deployment Time</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">
                {data.avg_deployment_time}s
              </p>
            </div>
            <ArrowPathIcon className="h-8 w-8 text-gray-400" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Current Config Size</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">
                {data.config_size_trend[data.config_size_trend.length - 1]?.size_kb}KB
              </p>
            </div>
            <DocumentTextIcon className="h-8 w-8 text-gray-400" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">API Response (P95)</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">
                {Math.max(...data.api_response_times.map(a => a.p95_ms))}ms
              </p>
            </div>
            <ChartBarIcon className="h-8 w-8 text-gray-400" />
          </div>
        </div>
      </div>

      {/* Config Size Trend */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Configuration Size Trend</h3>
        <div className="h-64 flex items-end space-x-2">
          {data.config_size_trend.map((point, index) => {
            const maxSize = Math.max(...data.config_size_trend.map(p => p.size_kb))
            const height = (point.size_kb / maxSize) * 100
            
            return (
              <div key={index} className="flex-1 relative group">
                <div
                  className="bg-purple-500 hover:bg-purple-600 rounded-t transition-all"
                  style={{ height: `${height}%` }}
                >
                  <div className="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 bg-gray-800 text-white text-xs rounded px-2 py-1 opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap">
                    {point.date}: {point.size_kb}KB
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* API Performance */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b">
          <h3 className="text-lg font-medium text-gray-900">API Endpoint Performance</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Endpoint
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Avg Response
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  P95 Response
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data.api_response_times.map((api, index) => (
                <tr key={index}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    {api.endpoint}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {api.avg_ms}ms
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {api.p95_ms}ms
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      api.p95_ms < 100 ? 'bg-green-100 text-green-700' :
                      api.p95_ms < 500 ? 'bg-yellow-100 text-yellow-700' :
                      'bg-red-100 text-red-700'
                    }`}>
                      {api.p95_ms < 100 ? 'Good' : api.p95_ms < 500 ? 'Fair' : 'Slow'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

const ErrorsTab: React.FC<{ data: AnalyticsData['errors'] }> = ({ data }) => {
  return (
    <div className="space-y-6">
      {/* Error Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Validation Failures</p>
              <p className="text-2xl font-bold text-red-600 mt-1">
                {data.validation_failures}
              </p>
            </div>
            <ExclamationTriangleIcon className="h-8 w-8 text-red-400" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Failed Deployments</p>
              <p className="text-2xl font-bold text-orange-600 mt-1">
                {data.failed_deployments}
              </p>
            </div>
            <ExclamationTriangleIcon className="h-8 w-8 text-orange-400" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Rollbacks</p>
              <p className="text-2xl font-bold text-yellow-600 mt-1">
                {data.rollback_count}
              </p>
            </div>
            <ArrowPathIcon className="h-8 w-8 text-yellow-400" />
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Error Rate</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">
                {data.error_rate}%
              </p>
            </div>
            <ChartPieIcon className="h-8 w-8 text-gray-400" />
          </div>
        </div>
      </div>

      {/* Common Errors */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b">
          <h3 className="text-lg font-medium text-gray-900">Common Errors</h3>
        </div>
        <div className="p-6">
          <div className="space-y-4">
            {data.common_errors.map((error, index) => (
              <div key={index} className="border-l-4 border-red-500 pl-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-sm font-medium text-gray-900">
                      {error.type}
                    </h4>
                    <p className="text-sm text-gray-600 mt-1">
                      Last occurred: {new Date(error.last_occurred).toLocaleString()}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-2xl font-bold text-red-600">{error.count}</p>
                    <p className="text-xs text-gray-500">occurrences</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Error Prevention Tips */}
      <div className="bg-blue-50 rounded-lg p-6">
        <h3 className="text-lg font-medium text-blue-900 mb-3">
          Error Prevention Tips
        </h3>
        <ul className="space-y-2 text-sm text-blue-700">
          <li className="flex items-start">
            <CheckCircleIcon className="h-5 w-5 mr-2 mt-0.5 flex-shrink-0" />
            Always validate configuration changes before deployment
          </li>
          <li className="flex items-start">
            <CheckCircleIcon className="h-5 w-5 mr-2 mt-0.5 flex-shrink-0" />
            Use configuration templates to avoid common mistakes
          </li>
          <li className="flex items-start">
            <CheckCircleIcon className="h-5 w-5 mr-2 mt-0.5 flex-shrink-0" />
            Test in staging environment before production deployment
          </li>
          <li className="flex items-start">
            <CheckCircleIcon className="h-5 w-5 mr-2 mt-0.5 flex-shrink-0" />
            Enable automatic backups before major changes
          </li>
        </ul>
      </div>
    </div>
  )
}

// Helper Components
const StatCard: React.FC<{
  title: string
  value: string | number
  icon: React.ComponentType<{ className?: string }>
  trend?: number
  valueColor?: string
}> = ({ title, value, icon: Icon, trend, valueColor = 'text-gray-900' }) => {
  return (
    <div className="bg-white rounded-lg shadow p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs text-gray-600">{title}</p>
          <p className={`text-xl font-bold mt-1 ${valueColor}`}>
            {value}
          </p>
          {trend !== undefined && (
            <p className={`text-xs mt-1 flex items-center ${
              trend > 0 ? 'text-green-600' : trend < 0 ? 'text-red-600' : 'text-gray-500'
            }`}>
              {trend > 0 ? (
                <ArrowTrendingUpIcon className="h-3 w-3 mr-1" />
              ) : trend < 0 ? (
                <ArrowTrendingDownIcon className="h-3 w-3 mr-1" />
              ) : null}
              {Math.abs(trend)}%
            </p>
          )}
        </div>
        <Icon className="h-8 w-8 text-gray-400" />
      </div>
    </div>
  )
}

// Helper functions
function calculateTrend(): number {
  // Mock trend calculation
  return Math.floor(Math.random() * 20) - 10
}

export default ConfigAnalytics