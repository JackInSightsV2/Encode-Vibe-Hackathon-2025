import React, { useState, useEffect } from 'react';
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext';
import { MessageTypes } from '../services/websocket';

interface RateLimitStatus {
  global: {
    current_rate: number;
    limit: number;
    burst_limit: number;
    status: 'ok' | 'warning' | 'critical';
    reset_time?: string;
  };
  per_ip: {
    active_ips: number;
    violating_ips: number;
    blocked_ips: number;
    top_offenders: Array<{
      ip: string;
      requests: number;
      violations: number;
      status: 'blocked' | 'throttled' | 'warning';
    }>;
  };
  per_user: {
    active_users: number;
    violating_users: number;
    blocked_users: number;
    top_users: Array<{
      user_id: string;
      username?: string;
      requests: number;
      violations: number;
      status: 'blocked' | 'throttled' | 'warning';
    }>;
  };
  websocket: {
    active_connections: number;
    connection_rate: number;
    limit: number;
    blocked_connections: number;
  };
}

interface RateLimitConfig {
  global: {
    requests_per_second: number;
    burst_limit: number;
    enabled: boolean;
  };
  per_ip: {
    requests_per_minute: number;
    requests_per_hour: number;
    enabled: boolean;
  };
  per_user: {
    requests_per_minute: number;
    requests_per_hour: number;
    enabled: boolean;
  };
  websocket: {
    connections_per_minute: number;
    max_connections_per_ip: number;
    enabled: boolean;
  };
}

const RateLimitingDashboard: React.FC = () => {
  const { isConnected } = useWebSocket();
  const [status, setStatus] = useState<RateLimitStatus | null>(null);
  const [config, setConfig] = useState<RateLimitConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'overview' | 'config' | 'violations'>('overview');

  // Listen for real-time rate limiting updates
  useWebSocketMessage(MessageTypes.RATE_LIMIT_UPDATE, (data) => {
    if (data?.rate_limits) {
      setStatus(data.rate_limits);
    }
  });

  useEffect(() => {
    fetchRateLimitData();
    const interval = setInterval(fetchRateLimitData, 10000); // Update every 10s
    return () => clearInterval(interval);
  }, []);

  const fetchRateLimitData = async () => {
    try {
      const [statusResponse, configResponse] = await Promise.all([
        fetch('/api/rate-limits/status'),
        fetch('/api/rate-limits/config')
      ]);

      if (statusResponse.ok) {
        const statusData = await statusResponse.json();
        setStatus(statusData.data);
      }

      if (configResponse.ok) {
        const configData = await configResponse.json();
        setConfig(configData.data);
      }
    } catch (error) {
      console.error('Failed to fetch rate limit data:', error);
    } finally {
      setLoading(false);
    }
  };

  const updateConfig = async (newConfig: RateLimitConfig) => {
    try {
      const response = await fetch('/api/rate-limits/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newConfig)
      });

      if (response.ok) {
        setConfig(newConfig);
        await fetchRateLimitData(); // Refresh status
      }
    } catch (error) {
      console.error('Failed to update rate limit config:', error);
    }
  };

  const blockIP = async (ip: string) => {
    try {
      await fetch('/api/rate-limits/block-ip', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ip })
      });
      await fetchRateLimitData();
    } catch (error) {
      console.error('Failed to block IP:', error);
    }
  };

  const unblockIP = async (ip: string) => {
    try {
      await fetch('/api/rate-limits/unblock-ip', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ip })
      });
      await fetchRateLimitData();
    } catch (error) {
      console.error('Failed to unblock IP:', error);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ok': return 'text-green-600 bg-green-50';
      case 'warning': return 'text-yellow-600 bg-yellow-50';
      case 'critical': return 'text-red-600 bg-red-50';
      case 'blocked': return 'text-red-600 bg-red-50';
      case 'throttled': return 'text-yellow-600 bg-yellow-50';
      default: return 'text-slate-600 bg-slate-50';
    }
  };

  const getRatePercentage = (current: number, limit: number) => {
    return Math.min((current / limit) * 100, 100);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-2 text-slate-600">Loading rate limiting data...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">Rate Limiting Dashboard</h1>
            <p className="text-slate-600 mt-1">Monitor and configure API rate limits in real-time</p>
          </div>
          <div className="flex items-center space-x-3">
            <div className={`text-sm font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? '🟢 Live Updates' : '🔴 No Live Data'}
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200">
        <nav className="-mb-px flex space-x-8">
          {[
            { id: 'overview', label: 'Overview', icon: '📊' },
            { id: 'config', label: 'Configuration', icon: '⚙️' },
            { id: 'violations', label: 'Violations', icon: '🚨' }
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center space-x-2 py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              <span>{tab.icon}</span>
              <span>{tab.label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && (
        <div className="space-y-6">
          {!status ? (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="text-center text-slate-500">
                <div className="text-lg font-medium mb-2">No Rate Limit Data Available</div>
                <div className="text-sm">Waiting for data from the server...</div>
              </div>
            </div>
          ) : (
            <>
              {/* Global Rate Limit Status */}
              <div className="bg-white rounded-lg border border-slate-200 p-6">
                <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
                  <span className="mr-2">🌐</span>
                  Global Rate Limits
                </h3>
                
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-slate-900">{status.global?.current_rate ?? 0}</div>
                    <div className="text-sm text-slate-600">Requests/Second</div>
                    <div className="mt-2">
                      <div className="bg-slate-200 rounded-full h-2">
                        <div 
                          className={`h-2 rounded-full ${
                            status.global?.status === 'critical' ? 'bg-red-500' :
                            status.global?.status === 'warning' ? 'bg-yellow-500' : 'bg-green-500'
                          }`}
                          style={{ width: `${getRatePercentage(status.global?.current_rate ?? 0, status.global?.limit ?? 1)}%` }}
                        ></div>
                      </div>
                      <div className="text-xs text-slate-500 mt-1">
                        {status.global?.limit ?? 0} req/s limit
                      </div>
                    </div>
                  </div>

                  <div className="text-center">
                    <div className="text-2xl font-bold text-slate-900">{status.global?.burst_limit ?? 0}</div>
                    <div className="text-sm text-slate-600">Burst Limit</div>
                    <div className={`mt-2 inline-flex px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(status.global?.status ?? 'ok')}`}>
                      {(status.global?.status ?? 'ok').toUpperCase()}
                    </div>
                  </div>

                  <div className="text-center">
                    <div className="text-2xl font-bold text-slate-900">
                      {status.global?.reset_time ? new Date(status.global.reset_time).toLocaleTimeString() : 'N/A'}
                    </div>
                    <div className="text-sm text-slate-600">Next Reset</div>
                  </div>
                </div>
              </div>

          {/* IP and User Stats */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* IP Rate Limits */}
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
                <span className="mr-2">🌍</span>
                IP Rate Limits
              </h3>
              
              <div className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="text-sm text-slate-600">Active IPs</span>
                  <span className="font-semibold">{status.per_ip?.active_ips ?? 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-slate-600">Violating IPs</span>
                  <span className="font-semibold text-yellow-600">{status.per_ip?.violating_ips ?? 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-slate-600">Blocked IPs</span>
                  <span className="font-semibold text-red-600">{status.per_ip?.blocked_ips ?? 0}</span>
                </div>
              </div>

              {status.per_ip?.top_offenders && status.per_ip.top_offenders.length > 0 && (
                <div className="mt-4 pt-4 border-t border-slate-200">
                  <h4 className="text-sm font-medium text-slate-900 mb-2">Top Offenders</h4>
                  <div className="space-y-2">
                    {status.per_ip.top_offenders.slice(0, 3).map((offender) => (
                      <div key={offender.ip} className="flex items-center justify-between text-sm">
                        <div className="flex items-center space-x-2">
                          <span className="font-mono">{offender.ip}</span>
                          <span className={`px-2 py-1 rounded-full text-xs ${getStatusColor(offender.status)}`}>
                            {offender.status}
                          </span>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className="text-slate-600">{offender.requests} reqs</span>
                          {offender.status !== 'blocked' && (
                            <button
                              onClick={() => blockIP(offender.ip)}
                              className="text-red-600 hover:text-red-800 text-xs"
                            >
                              Block
                            </button>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* User Rate Limits */}
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
                <span className="mr-2">👥</span>
                User Rate Limits
              </h3>
              
              <div className="space-y-4">
                <div className="flex justify-between items-center">
                  <span className="text-sm text-slate-600">Active Users</span>
                  <span className="font-semibold">{status.per_user?.active_users ?? 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-slate-600">Violating Users</span>
                  <span className="font-semibold text-yellow-600">{status.per_user?.violating_users ?? 0}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-slate-600">Blocked Users</span>
                  <span className="font-semibold text-red-600">{status.per_user?.blocked_users ?? 0}</span>
                </div>
              </div>

              {status.per_user?.top_users && status.per_user.top_users.length > 0 && (
                <div className="mt-4 pt-4 border-t border-slate-200">
                  <h4 className="text-sm font-medium text-slate-900 mb-2">Top Users</h4>
                  <div className="space-y-2">
                    {status.per_user.top_users.slice(0, 3).map((user) => (
                      <div key={user.user_id} className="flex items-center justify-between text-sm">
                        <div className="flex items-center space-x-2">
                          <span className="font-mono">{user.username || user.user_id}</span>
                          <span className={`px-2 py-1 rounded-full text-xs ${getStatusColor(user.status)}`}>
                            {user.status}
                          </span>
                        </div>
                        <span className="text-slate-600">{user.requests} reqs</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* WebSocket Connections */}
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
              <span className="mr-2">🔌</span>
              WebSocket Rate Limits
            </h3>
            
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div className="text-center">
                <div className="text-2xl font-bold text-slate-900">{status.websocket?.active_connections ?? 0}</div>
                <div className="text-sm text-slate-600">Active Connections</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-slate-900">{status.websocket?.connection_rate ?? 0}</div>
                <div className="text-sm text-slate-600">Conn/Min</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-slate-900">{status.websocket?.limit ?? 0}</div>
                <div className="text-sm text-slate-600">Rate Limit</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-red-600">{status.websocket?.blocked_connections ?? 0}</div>
                <div className="text-sm text-slate-600">Blocked</div>
              </div>
            </div>
          </div>
            </>
          )}
        </div>
      )}

      {/* Configuration Tab */}
      {activeTab === 'config' && (
        <div className="space-y-6">
          {!config ? (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="text-center text-slate-500">
                <div className="text-lg font-medium mb-2">No Configuration Data Available</div>
                <div className="text-sm">Waiting for configuration data from the server...</div>
              </div>
            </div>
          ) : (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4">Rate Limit Configuration</h3>
              
              <div className="space-y-6">
                {/* Global Config */}
                <div>
                  <h4 className="text-md font-medium text-slate-900 mb-3">Global Rate Limits</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Requests per Second
                      </label>
                      <input
                        type="number"
                        value={config.global?.requests_per_second ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          global: { ...config.global, requests_per_second: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Burst Limit
                      </label>
                      <input
                        type="number"
                        value={config.global?.burst_limit ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          global: { ...config.global, burst_limit: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div className="flex items-center">
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={config.global?.enabled ?? false}
                          onChange={(e) => setConfig({
                            ...config,
                            global: { ...config.global, enabled: e.target.checked }
                          })}
                          className="rounded border-slate-300"
                        />
                        <span className="ml-2 text-sm text-slate-700">Enabled</span>
                      </label>
                    </div>
                  </div>
                </div>

                {/* IP Config */}
                <div>
                  <h4 className="text-md font-medium text-slate-900 mb-3">Per-IP Rate Limits</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Requests per Minute
                      </label>
                      <input
                        type="number"
                        value={config.per_ip?.requests_per_minute ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          per_ip: { ...config.per_ip, requests_per_minute: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Requests per Hour
                      </label>
                      <input
                        type="number"
                        value={config.per_ip?.requests_per_hour ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          per_ip: { ...config.per_ip, requests_per_hour: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div className="flex items-center">
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={config.per_ip?.enabled ?? false}
                          onChange={(e) => setConfig({
                            ...config,
                            per_ip: { ...config.per_ip, enabled: e.target.checked }
                          })}
                          className="rounded border-slate-300"
                        />
                        <span className="ml-2 text-sm text-slate-700">Enabled</span>
                      </label>
                    </div>
                  </div>
                </div>

                {/* User Config */}
                <div>
                  <h4 className="text-md font-medium text-slate-900 mb-3">Per-User Rate Limits</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Requests per Minute
                      </label>
                      <input
                        type="number"
                        value={config.per_user?.requests_per_minute ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          per_user: { ...config.per_user, requests_per_minute: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Requests per Hour
                      </label>
                      <input
                        type="number"
                        value={config.per_user?.requests_per_hour ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          per_user: { ...config.per_user, requests_per_hour: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div className="flex items-center">
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={config.per_user?.enabled ?? false}
                          onChange={(e) => setConfig({
                            ...config,
                            per_user: { ...config.per_user, enabled: e.target.checked }
                          })}
                          className="rounded border-slate-300"
                        />
                        <span className="ml-2 text-sm text-slate-700">Enabled</span>
                      </label>
                    </div>
                  </div>
                </div>

                {/* WebSocket Config */}
                <div>
                  <h4 className="text-md font-medium text-slate-900 mb-3">WebSocket Rate Limits</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Connections per Minute
                      </label>
                      <input
                        type="number"
                        value={config.websocket?.connections_per_minute ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          websocket: { ...config.websocket, connections_per_minute: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Max Connections per IP
                      </label>
                      <input
                        type="number"
                        value={config.websocket?.max_connections_per_ip ?? 0}
                        onChange={(e) => setConfig({
                          ...config,
                          websocket: { ...config.websocket, max_connections_per_ip: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                      />
                    </div>
                    <div className="flex items-center">
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={config.websocket?.enabled ?? false}
                          onChange={(e) => setConfig({
                            ...config,
                            websocket: { ...config.websocket, enabled: e.target.checked }
                          })}
                          className="rounded border-slate-300"
                        />
                        <span className="ml-2 text-sm text-slate-700">Enabled</span>
                      </label>
                    </div>
                  </div>
                </div>

                <div className="pt-4 border-t border-slate-200">
                  <button
                    onClick={() => updateConfig(config)}
                    className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                  >
                    Save Configuration
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Violations Tab */}
      {activeTab === 'violations' && (
        <div className="space-y-6">
          {!status ? (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="text-center text-slate-500">
                <div className="text-lg font-medium mb-2">No Violations Data Available</div>
                <div className="text-sm">Waiting for data from the server...</div>
              </div>
            </div>
          ) : (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4">Rate Limit Violations</h3>
              
              {/* All Offenders */}
              {status.per_ip?.top_offenders && status.per_ip.top_offenders.length > 0 && (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-200">
                  <thead className="bg-slate-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                        IP Address
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                        Requests
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                        Violations
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                        Status
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-slate-200">
                    {status.per_ip.top_offenders.map((offender) => (
                      <tr key={offender.ip}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-slate-900">
                          {offender.ip}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                          {offender.requests}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                          {offender.violations}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(offender.status)}`}>
                            {offender.status}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                          {offender.status === 'blocked' ? (
                            <button
                              onClick={() => unblockIP(offender.ip)}
                              className="text-green-600 hover:text-green-900"
                            >
                              Unblock
                            </button>
                          ) : (
                            <button
                              onClick={() => blockIP(offender.ip)}
                              className="text-red-600 hover:text-red-900"
                            >
                              Block
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

              {status.per_ip?.top_offenders && status.per_ip.top_offenders.length === 0 && (
                <div className="text-center py-8 text-slate-500">
                  No rate limit violations detected.
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default RateLimitingDashboard;