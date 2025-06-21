import React, { useState, useEffect } from 'react';
import { SafetyCockpit3D } from './SafetyCockpit3D';
import MetricsDashboard from './MetricsDashboard';
import SystemHealthDashboard from './SystemHealthDashboard';
import AdvancedDashboard from './AdvancedDashboard';
import { getCockpitService, ConnectionState, SafetyEvent, EventType } from '../services/cockpitService';

// Dashboard views
enum DashboardView {
  OVERVIEW = 'overview',
  REALTIME_3D = 'realtime_3d',
  METRICS = 'metrics',
  SYSTEM_HEALTH = 'system_health',
  ADVANCED = 'advanced'
}

// Navigation item interface
interface NavItem {
  id: DashboardView;
  label: string;
  icon: string;
  description: string;
}

// Main AI Safety Cockpit component
export const AISafetyCockpit: React.FC = () => {
  const [currentView, setCurrentView] = useState<DashboardView>(DashboardView.OVERVIEW);
  const [connectionState, setConnectionState] = useState<ConnectionState>(ConnectionState.DISCONNECTED);
  const [realtimeStats, setRealtimeStats] = useState({
    totalEvents: 0,
    activeThreats: 0,
    blockedRequests: 0,
    systemHealth: 'good' as 'good' | 'warning' | 'critical'
  });

  // Navigation items
  const navItems: NavItem[] = [
    {
      id: DashboardView.OVERVIEW,
      label: 'Overview',
      icon: '🏠',
      description: 'System overview and key metrics'
    },
    {
      id: DashboardView.REALTIME_3D,
      label: '3D Visualization',
      icon: '🌐',
      description: 'Real-time 3D threat visualization'
    },
    {
      id: DashboardView.METRICS,
      label: 'Metrics',
      icon: '📊',
      description: 'Performance metrics and analytics'
    },
    {
      id: DashboardView.SYSTEM_HEALTH,
      label: 'System Health',
      icon: '❤️',
      description: 'System health and monitoring'
    },
    {
      id: DashboardView.ADVANCED,
      label: 'Advanced',
      icon: '⚙️',
      description: 'Advanced configuration and tools'
    }
  ];

  // Initialize connection and event listeners
  useEffect(() => {
    const cockpitService = getCockpitService();

    const onStateChange = ({ currentState }: { currentState: ConnectionState }) => {
      setConnectionState(currentState);
    };

    const onEvent = (event: SafetyEvent) => {
      setRealtimeStats(prev => ({
        ...prev,
        totalEvents: prev.totalEvents + 1,
        activeThreats: event.type === EventType.THREAT ? prev.activeThreats + 1 : prev.activeThreats,
        blockedRequests: event.data?.blocked ? prev.blockedRequests + 1 : prev.blockedRequests
      }));
    };

    const onThreat = (event: SafetyEvent) => {
      // Handle high-severity threats
      if (event.severity === 'high') {
        setRealtimeStats(prev => ({
          ...prev,
          systemHealth: 'critical'
        }));
      } else if (event.severity === 'medium') {
        setRealtimeStats(prev => ({
          ...prev,
          systemHealth: prev.systemHealth === 'good' ? 'warning' : prev.systemHealth
        }));
      }
    };

    cockpitService.on('stateChange', onStateChange);
    cockpitService.on('event', onEvent);
    cockpitService.on('threat', onThreat);

    // Connect if not already connected
    if (connectionState === ConnectionState.DISCONNECTED) {
      cockpitService.connect();
    }

    // Cleanup
    return () => {
      cockpitService.off('stateChange', onStateChange);
      cockpitService.off('event', onEvent);
      cockpitService.off('threat', onThreat);
    };
  }, [connectionState]);

  // Reset system health periodically
  useEffect(() => {
    const interval = setInterval(() => {
      setRealtimeStats(prev => ({
        ...prev,
        systemHealth: 'good'
      }));
    }, 60000); // Reset every minute

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* Header */}
      <header className="bg-gray-800 border-b border-gray-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <div className="w-8 h-8 bg-gradient-to-r from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">AI</span>
              </div>
              <h1 className="text-xl font-bold">AI Safety Cockpit</h1>
            </div>
            
            <div className="flex items-center space-x-2">
              <div className={`w-2 h-2 rounded-full ${
                connectionState === ConnectionState.CONNECTED ? 'bg-green-500' :
                connectionState === ConnectionState.CONNECTING ? 'bg-yellow-500' : 'bg-red-500'
              }`} />
              <span className="text-sm text-gray-400 capitalize">{connectionState}</span>
            </div>
          </div>

          {/* Real-time stats */}
          <div className="flex items-center space-x-6">
            <div className="text-center">
              <div className="text-lg font-semibold">{realtimeStats.totalEvents}</div>
              <div className="text-xs text-gray-400">Total Events</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-semibold text-red-400">{realtimeStats.activeThreats}</div>
              <div className="text-xs text-gray-400">Active Threats</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-semibold text-orange-400">{realtimeStats.blockedRequests}</div>
              <div className="text-xs text-gray-400">Blocked</div>
            </div>
            <div className="text-center">
              <div className={`text-lg font-semibold ${
                realtimeStats.systemHealth === 'good' ? 'text-green-400' :
                realtimeStats.systemHealth === 'warning' ? 'text-yellow-400' : 'text-red-400'
              }`}>
                {realtimeStats.systemHealth === 'good' ? '✅' :
                 realtimeStats.systemHealth === 'warning' ? '⚠️' : '🚨'}
              </div>
              <div className="text-xs text-gray-400">Health</div>
            </div>
          </div>
        </div>
      </header>

      <div className="flex h-[calc(100vh-80px)]">
        {/* Sidebar Navigation */}
        <aside className="w-64 bg-gray-800 border-r border-gray-700">
          <nav className="p-4 space-y-2">
            {navItems.map((item) => (
              <button
                key={item.id}
                onClick={() => setCurrentView(item.id)}
                className={`w-full text-left p-3 rounded-lg transition-colors ${
                  currentView === item.id
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                }`}
              >
                <div className="flex items-center space-x-3">
                  <span className="text-lg">{item.icon}</span>
                  <div>
                    <div className="font-medium">{item.label}</div>
                    <div className="text-xs text-gray-400">{item.description}</div>
                  </div>
                </div>
              </button>
            ))}
          </nav>

          {/* Quick Actions */}
          <div className="p-4 border-t border-gray-700">
            <h3 className="text-sm font-semibold text-gray-400 mb-3">Quick Actions</h3>
            <div className="space-y-2">
              <button
                onClick={() => getCockpitService().requestAggregatedData()}
                className="w-full text-left p-2 rounded text-sm bg-gray-700 hover:bg-gray-600 transition-colors"
              >
                🔄 Refresh Data
              </button>
              <button
                onClick={() => getCockpitService().requestMetrics()}
                className="w-full text-left p-2 rounded text-sm bg-gray-700 hover:bg-gray-600 transition-colors"
              >
                📈 Get Metrics
              </button>
              <button
                onClick={() => {
                  getCockpitService().setFilter({});
                }}
                className="w-full text-left p-2 rounded text-sm bg-gray-700 hover:bg-gray-600 transition-colors"
              >
                🔍 Clear Filters
              </button>
            </div>
          </div>

          {/* Status Panel */}
          <div className="p-4 border-t border-gray-700">
            <h3 className="text-sm font-semibold text-gray-400 mb-3">System Status</h3>
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-xs text-gray-400">Connection</span>
                <span className={`text-xs px-2 py-1 rounded ${
                  connectionState === ConnectionState.CONNECTED ? 'bg-green-600' :
                  connectionState === ConnectionState.CONNECTING ? 'bg-yellow-600' : 'bg-red-600'
                }`}>
                  {connectionState}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-xs text-gray-400">Health</span>
                <span className={`text-xs px-2 py-1 rounded capitalize ${
                  realtimeStats.systemHealth === 'good' ? 'bg-green-600' :
                  realtimeStats.systemHealth === 'warning' ? 'bg-yellow-600' : 'bg-red-600'
                }`}>
                  {realtimeStats.systemHealth}
                </span>
              </div>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-hidden">
          {renderCurrentView(currentView)}
        </main>
      </div>
    </div>
  );
};

// Render current view based on selection
function renderCurrentView(view: DashboardView) {
  switch (view) {
    case DashboardView.OVERVIEW:
      return <OverviewDashboard />;
    case DashboardView.REALTIME_3D:
      return <SafetyCockpit3D />;
    case DashboardView.METRICS:
      return <MetricsDashboard />;
    case DashboardView.SYSTEM_HEALTH:
      return <SystemHealthDashboard />;
    case DashboardView.ADVANCED:
      return <AdvancedDashboard systemStatus={{}} />;
    default:
      return <OverviewDashboard />;
  }
}

// Overview Dashboard Component
const OverviewDashboard: React.FC = () => {
  const [stats, setStats] = useState({
    events: { total: 0, threats: 0, moderation: 0, provider: 0 },
    performance: { avgLatency: 0, successRate: 0, throughput: 0 },
    security: { blockedThreats: 0, falsePositives: 0, accuracy: 0 }
  });

  useEffect(() => {
    const cockpitService = getCockpitService();

    const onMetrics = (data: any) => {
      if (data.event_counts) {
        setStats(prev => ({
          ...prev,
          events: {
            total: Object.values(data.event_counts).reduce((sum: number, count: any) => sum + count, 0),
            threats: data.event_counts[EventType.THREAT] || 0,
            moderation: data.event_counts[EventType.MODERATION] || 0,
            provider: data.event_counts[EventType.PROVIDER] || 0
          }
        }));
      }
    };

    const onAggregated = (data: any) => {
      if (data.data && data.data.provider_stats) {
        const providers = Object.values(data.data.provider_stats) as any[];
        const avgLatency = providers.reduce((sum, p) => sum + (p.avg_response_time || 0), 0) / Math.max(providers.length, 1);
        const totalRequests = providers.reduce((sum, p) => sum + (p.request_count || 0), 0);
        const totalSuccess = providers.reduce((sum, p) => sum + (p.success_count || 0), 0);
        const successRate = totalRequests > 0 ? (totalSuccess / totalRequests) * 100 : 0;

        setStats(prev => ({
          ...prev,
          performance: {
            avgLatency: Math.round(avgLatency),
            successRate: Math.round(successRate),
            throughput: Math.round(totalRequests / 60) // requests per minute
          }
        }));
      }
    };

    cockpitService.on('metrics', onMetrics);
    cockpitService.on('aggregated', onAggregated);

    // Request initial data
    cockpitService.requestMetrics();
    cockpitService.requestAggregatedData();

    return () => {
      cockpitService.off('metrics', onMetrics);
      cockpitService.off('aggregated', onAggregated);
    };
  }, []);

  return (
    <div className="p-6 space-y-6 overflow-y-auto">
      <div>
        <h2 className="text-2xl font-bold mb-2">Safety Overview</h2>
        <p className="text-gray-400">Real-time AI safety monitoring and threat detection</p>
      </div>

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="Total Events"
          value={stats.events.total.toLocaleString()}
          trend="+12%"
          trendUp={true}
          icon="📊"
        />
        <MetricCard
          title="Active Threats"
          value={stats.events.threats.toString()}
          trend="-5%"
          trendUp={false}
          icon="🚨"
        />
        <MetricCard
          title="Avg Latency"
          value={`${stats.performance.avgLatency}ms`}
          trend="-8%"
          trendUp={false}
          icon="⚡"
        />
        <MetricCard
          title="Success Rate"
          value={`${stats.performance.successRate}%`}
          trend="+2%"
          trendUp={true}
          icon="✅"
        />
      </div>

      {/* Recent Activity Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4">Event Distribution</h3>
          <div className="space-y-3">
            <EventTypeBar
              label="Threats"
              count={stats.events.threats}
              total={stats.events.total}
              color="red"
            />
            <EventTypeBar
              label="Moderation"
              count={stats.events.moderation}
              total={stats.events.total}
              color="orange"
            />
            <EventTypeBar
              label="Provider"
              count={stats.events.provider}
              total={stats.events.total}
              color="blue"
            />
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4">Performance Metrics</h3>
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-400">Average Latency</span>
              <span className="font-semibold">{stats.performance.avgLatency}ms</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-400">Success Rate</span>
              <span className="font-semibold">{stats.performance.successRate}%</span>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-gray-400">Throughput</span>
              <span className="font-semibold">{stats.performance.throughput} req/min</span>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-gray-800 rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <ActionButton
            label="View 3D Visualization"
            icon="🌐"
            onClick={() => window.location.hash = '#realtime_3d'}
          />
          <ActionButton
            label="Check System Health"
            icon="❤️"
            onClick={() => window.location.hash = '#system_health'}
          />
          <ActionButton
            label="View Metrics"
            icon="📊"
            onClick={() => window.location.hash = '#metrics'}
          />
          <ActionButton
            label="Advanced Settings"
            icon="⚙️"
            onClick={() => window.location.hash = '#advanced'}
          />
        </div>
      </div>
    </div>
  );
};

// Metric Card Component
interface MetricCardProps {
  title: string;
  value: string;
  trend: string;
  trendUp: boolean;
  icon: string;
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, trend, trendUp, icon }) => {

  return (
    <div className="bg-gray-800 rounded-lg p-6">
      <div className="flex items-center justify-between mb-2">
        <span className="text-gray-400 text-sm">{title}</span>
        <span className="text-2xl">{icon}</span>
      </div>
      <div className="text-2xl font-bold mb-1">{value}</div>
      <div className={`text-sm flex items-center ${trendUp ? 'text-green-400' : 'text-red-400'}`}>
        <span className="mr-1">{trendUp ? '↗' : '↘'}</span>
        {trend}
      </div>
    </div>
  );
};

// Event Type Bar Component
interface EventTypeBarProps {
  label: string;
  count: number;
  total: number;
  color: 'red' | 'orange' | 'blue';
}

const EventTypeBar: React.FC<EventTypeBarProps> = ({ label, count, total, color }) => {
  const percentage = total > 0 ? (count / total) * 100 : 0;
  
  const colorClasses = {
    red: 'bg-red-500',
    orange: 'bg-orange-500',
    blue: 'bg-blue-500'
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-1">
        <span className="text-sm text-gray-400">{label}</span>
        <span className="text-sm font-medium">{count}</span>
      </div>
      <div className="w-full bg-gray-700 rounded-full h-2">
        <div
          className={`h-2 rounded-full transition-all duration-300 ${colorClasses[color]}`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
};

// Action Button Component
interface ActionButtonProps {
  label: string;
  icon: string;
  onClick: () => void;
}

const ActionButton: React.FC<ActionButtonProps> = ({ label, icon, onClick }) => {
  return (
    <button
      onClick={onClick}
      className="bg-gray-700 hover:bg-gray-600 p-4 rounded-lg transition-colors text-center"
    >
      <div className="text-2xl mb-2">{icon}</div>
      <div className="text-sm font-medium">{label}</div>
    </button>
  );
};

export default AISafetyCockpit;