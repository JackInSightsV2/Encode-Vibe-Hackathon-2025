import React from 'react'

interface DashboardProps {
  systemStatus: any
}

const Dashboard: React.FC<DashboardProps> = ({ systemStatus }) => {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <h1 className="text-3xl font-bold text-slate-900 mb-2">System Dashboard</h1>
        <p className="text-slate-600">Monitor your QT-1 Responsible AI Middleware in real-time</p>
      </div>

      {/* Status Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* System Health */}
        <div className="bg-gradient-to-br from-green-50 to-green-100 border border-green-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">System Health</h3>
            <div className={`w-3 h-3 rounded-full ${systemStatus?.status === 'healthy' ? 'bg-green-500' : 'bg-red-500'} animate-pulse`}></div>
          </div>
          <div className="flex items-center space-x-2">
            <span className={`text-2xl font-bold ${systemStatus?.status === 'healthy' ? 'text-green-700' : 'text-red-700'}`}>
              {systemStatus?.status === 'healthy' ? 'Healthy' : 'Offline'}
            </span>
          </div>
          <p className="text-xs text-slate-500 mt-2">Last checked: just now</p>
        </div>

        {/* Moderation Status */}
        <div className="bg-gradient-to-br from-blue-50 to-blue-100 border border-blue-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Content Filter</h3>
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              systemStatus?.moderation_enabled ? 'bg-blue-600 text-white' : 'bg-slate-300 text-slate-600'
            }`}>
              {systemStatus?.moderation_enabled ? 'ACTIVE' : 'INACTIVE'}
            </span>
          </div>
          <div className="text-2xl font-bold text-blue-700">
            {systemStatus?.moderation_enabled ? 'Protected' : 'Disabled'}
          </div>
          <p className="text-xs text-slate-500 mt-2">Regex + AI moderation</p>
        </div>

        {/* Kill Switch */}
        <div className="bg-gradient-to-br from-red-50 to-red-100 border border-red-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Emergency Control</h3>
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              systemStatus?.kill_switch_enabled ? 'bg-red-600 text-white' : 'bg-slate-300 text-slate-600'
            }`}>
              {systemStatus?.kill_switch_enabled ? 'ARMED' : 'DISABLED'}
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
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Performance</h3>
            <span className="px-2 py-1 bg-purple-600 text-white rounded-full text-xs font-medium">STABLE</span>
          </div>
          <div className="text-2xl font-bold text-purple-700">
            {systemStatus?.version || 'v1.0.0'}
          </div>
          <p className="text-xs text-slate-500 mt-2">Runtime version</p>
        </div>
      </div>

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