import React, { useState } from 'react'
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext'
import { MessageTypes } from '../services/websocket'

interface DashboardProps {
  systemStatus: any
}

interface HealthData {
  timestamp: string;
  system: {
    cpu_usage: number;
    memory_usage: number;
    goroutines: number;
    uptime: string;
  };
  service: {
    status: string;
    version: string;
    port: number;
    moderation_enabled: boolean;
    relevance_enabled: boolean;
    kill_switch_active: boolean;
  };
  websocket: {
    active_connections: number;
    total_connections: number;
    status: string;
  };
  providers: Record<string, any>;
  request_stats: {
    total_requests: number;
    successful_requests: number;
    failed_requests: number;
    average_response_ms: number;
    requests_per_minute: number;
  };
}

const Dashboard: React.FC<DashboardProps> = ({ systemStatus }) => {
  const { isConnected } = useWebSocket();
  const [healthData, setHealthData] = useState<HealthData | null>(null);
  const [lastUpdateTime, setLastUpdateTime] = useState<string>('');

  // Listen for health updates
  useWebSocketMessage(MessageTypes.HEALTH_UPDATE, (data) => {
    if (data?.health) {
      setHealthData(data.health);
      setLastUpdateTime(new Date().toLocaleTimeString());
    }
  });

  // Fallback to API status if no WebSocket health data
  const currentStatus = healthData?.service || systemStatus;
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-slate-900 mb-2">System Dashboard</h1>
            <p className="text-slate-600">Monitor your QT-1 Responsible AI Middleware in real-time</p>
          </div>
          <div className="text-right">
            <div className={`text-sm font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? '🟢 Live Updates' : '🔴 No Live Data'}
            </div>
            {lastUpdateTime && (
              <div className="text-xs text-slate-500">
                Last update: {lastUpdateTime}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* System Health */}
        <div className="bg-gradient-to-br from-green-50 to-green-100 border border-green-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">System Health</h3>
            <div className={`w-3 h-3 rounded-full ${currentStatus?.status === 'healthy' ? 'bg-green-500' : 'bg-red-500'} animate-pulse`}></div>
          </div>
          <div className="flex items-center space-x-2">
            <span className={`text-2xl font-bold ${currentStatus?.status === 'healthy' ? 'text-green-700' : 'text-red-700'}`}>
              {currentStatus?.status === 'healthy' ? 'Healthy' : 'Offline'}
            </span>
          </div>
          {healthData?.system && (
            <div className="text-xs text-slate-600 mt-2 space-y-1">
              <div>CPU: {(healthData.system.cpu_usage * 100).toFixed(1)}%</div>
              <div>RAM: {healthData.system.memory_usage.toFixed(1)}MB</div>
              <div>Goroutines: {healthData.system.goroutines}</div>
            </div>
          )}
          <p className="text-xs text-slate-500 mt-2">
            {isConnected ? 'Live updates' : 'Last checked: just now'}
          </p>
        </div>

        {/* Moderation Status */}
        <div className="bg-gradient-to-br from-blue-50 to-blue-100 border border-blue-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Content Filter</h3>
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              currentStatus?.moderation_enabled ? 'bg-blue-600 text-white' : 'bg-slate-300 text-slate-600'
            }`}>
              {currentStatus?.moderation_enabled ? 'ACTIVE' : 'INACTIVE'}
            </span>
          </div>
          <div className="text-2xl font-bold text-blue-700">
            {currentStatus?.moderation_enabled ? 'Protected' : 'Disabled'}
          </div>
          {healthData?.request_stats && (
            <div className="text-xs text-slate-600 mt-2">
              {healthData.request_stats.total_requests} total requests
            </div>
          )}
          <p className="text-xs text-slate-500 mt-1">Regex + AI moderation</p>
        </div>

        {/* Kill Switch */}
        <div className="bg-gradient-to-br from-red-50 to-red-100 border border-red-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Emergency Control</h3>
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              currentStatus?.kill_switch_active ? 'bg-red-600 text-white' : 'bg-slate-300 text-slate-600'
            }`}>
              {currentStatus?.kill_switch_active ? 'ARMED' : 'DISABLED'}
            </span>
          </div>
          <div className="text-2xl font-bold text-red-700">
            Kill Switch
          </div>
          <p className="text-xs text-slate-500 mt-2">User/session blocking</p>
        </div>

        {/* Performance */}
        <div className="bg-gradient-to-br from-purple-50 to-purple-100 border border-purple-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">WebSocket</h3>
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              isConnected ? 'bg-purple-600 text-white' : 'bg-slate-300 text-slate-600'
            }`}>
              {isConnected ? 'LIVE' : 'OFFLINE'}
            </span>
          </div>
          <div className="text-2xl font-bold text-purple-700">
            {healthData?.websocket?.active_connections || 0} Active
          </div>
          {healthData?.websocket && (
            <div className="text-xs text-slate-600 mt-2">
              {healthData.websocket.total_connections} total connections
            </div>
          )}
          <p className="text-xs text-slate-500 mt-1">Real-time connections</p>
        </div>
      </div>

      {/* Real-time Metrics Section */}
      {healthData && (
        <div className="bg-white border border-slate-200 rounded-xl p-6">
          <h2 className="text-xl font-bold text-slate-900 mb-6">Real-time Performance Metrics</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Request Stats */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Request Statistics</h3>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-slate-600">Total Requests:</span>
                  <span className="font-semibold">{healthData.request_stats.total_requests}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Success Rate:</span>
                  <span className="font-semibold text-green-600">
                    {healthData.request_stats.total_requests > 0 
                      ? ((healthData.request_stats.successful_requests / healthData.request_stats.total_requests) * 100).toFixed(1)
                      : 0}%
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Avg Response:</span>
                  <span className="font-semibold">{healthData.request_stats.average_response_ms.toFixed(1)}ms</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Requests/min:</span>
                  <span className="font-semibold">{healthData.request_stats.requests_per_minute.toFixed(1)}</span>
                </div>
              </div>
            </div>

            {/* System Resources */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">System Resources</h3>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-slate-600">CPU Usage:</span>
                  <span className="font-semibold">{(healthData.system.cpu_usage * 100).toFixed(1)}%</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Memory:</span>
                  <span className="font-semibold">{healthData.system.memory_usage.toFixed(1)} MB</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Goroutines:</span>
                  <span className="font-semibold">{healthData.system.goroutines}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Uptime:</span>
                  <span className="font-semibold">{healthData.system.uptime}</span>
                </div>
              </div>
            </div>

            {/* Service Status */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Service Configuration</h3>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-slate-600">Version:</span>
                  <span className="font-semibold">{healthData.service.version}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Port:</span>
                  <span className="font-semibold">{healthData.service.port}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Moderation:</span>
                  <span className={`font-semibold ${healthData.service.moderation_enabled ? 'text-green-600' : 'text-red-600'}`}>
                    {healthData.service.moderation_enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-600">Relevance:</span>
                  <span className={`font-semibold ${healthData.service.relevance_enabled ? 'text-green-600' : 'text-red-600'}`}>
                    {healthData.service.relevance_enabled ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* System Information */}
        <div className="lg:col-span-2">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">System Configuration</h2>
            
            <div className="space-y-4">
              <div className="flex items-center justify-between py-3 border-b border-slate-100">
                <div className="flex items-center space-x-3">
                  <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                  <span className="font-medium text-slate-700">Service Name</span>
                </div>
                <span className="text-slate-600 font-mono text-sm">{systemStatus?.service || 'qt1-middleware'}</span>
              </div>

              <div className="flex items-center justify-between py-3 border-b border-slate-100">
                <div className="flex items-center space-x-3">
                  <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                  <span className="font-medium text-slate-700">Uptime</span>
                </div>
                <span className="text-slate-600 font-mono text-sm">{systemStatus?.uptime || '< 1m'}</span>
              </div>

              <div className="flex items-center justify-between py-3 border-b border-slate-100">
                <div className="flex items-center space-x-3">
                  <div className={`w-2 h-2 rounded-full ${systemStatus?.relevance_enabled ? 'bg-yellow-500' : 'bg-slate-300'}`}></div>
                  <span className="font-medium text-slate-700">Relevance Filter</span>
                </div>
                <span className={`text-sm font-medium ${systemStatus?.relevance_enabled ? 'text-yellow-600' : 'text-slate-400'}`}>
                  {systemStatus?.relevance_enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>

              <div className="flex items-center justify-between py-3">
                <div className="flex items-center space-x-3">
                  <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
                  <span className="font-medium text-slate-700">Logging</span>
                </div>
                <span className="text-green-600 text-sm font-medium">Active</span>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">Quick Actions</h2>
            
            <div className="space-y-3">
              <button className="w-full bg-gradient-to-r from-blue-600 to-blue-700 text-white px-4 py-3 rounded-lg hover:from-blue-700 hover:to-blue-800 transition-all duration-200 text-left group">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold">Test Playground</div>
                    <div className="text-sm text-blue-100">Interactive testing</div>
                  </div>
                  <span className="text-blue-200 group-hover:text-white transition-colors">🧪</span>
                </div>
              </button>

              <button className="w-full bg-gradient-to-r from-green-600 to-green-700 text-white px-4 py-3 rounded-lg hover:from-green-700 hover:to-green-800 transition-all duration-200 text-left group">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold">Configuration</div>
                    <div className="text-sm text-green-100">Manage settings</div>
                  </div>
                  <span className="text-green-200 group-hover:text-white transition-colors">⚙️</span>
                </div>
              </button>

              <button className="w-full bg-gradient-to-r from-purple-600 to-purple-700 text-white px-4 py-3 rounded-lg hover:from-purple-700 hover:to-purple-800 transition-all duration-200 text-left group">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold">View Logs</div>
                    <div className="text-sm text-purple-100">Monitor activity</div>
                  </div>
                  <span className="text-purple-200 group-hover:text-white transition-colors">📝</span>
                </div>
              </button>

              <button className="w-full bg-gradient-to-r from-red-600 to-red-700 text-white px-4 py-3 rounded-lg hover:from-red-700 hover:to-red-800 transition-all duration-200 text-left group">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold">Emergency Stop</div>
                    <div className="text-sm text-red-100">Kill switch controls</div>
                  </div>
                  <span className="text-red-200 group-hover:text-white transition-colors">🛑</span>
                </div>
              </button>
            </div>
          </div>

          {/* Framework Info */}
          <div className="bg-gradient-to-br from-slate-50 to-slate-100 border border-slate-200 rounded-xl p-6">
            <h3 className="font-bold text-slate-900 mb-3">QT-1 Framework</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-slate-600">Laws Enforced:</span>
                <span className="font-semibold text-slate-900">7/7</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-600">Runtime Mode:</span>
                <span className="font-semibold text-green-600">Active</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-600">Compliance:</span>
                <span className="font-semibold text-green-600">100%</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Dashboard