import React, { useState, useEffect } from 'react';
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext';
import { MessageTypes } from '../services/websocket';

interface DDoSAttackPattern {
  type: 'volumetric' | 'protocol' | 'application';
  severity: 'low' | 'medium' | 'high' | 'critical';
  start_time: string;
  end_time?: string;
  peak_rps: number;
  total_requests: number;
  source_ips: number;
  blocked_requests: number;
  target_endpoints: string[];
}

interface CircuitBreakerStatus {
  name: string;
  state: 'closed' | 'open' | 'half-open';
  failure_count: number;
  failure_threshold: number;
  timeout: number;
  next_attempt?: string;
  success_count?: number;
  last_failure?: string;
}

interface DDoSMetrics {
  current_rps: number;
  average_rps_1m: number;
  average_rps_5m: number;
  peak_rps_today: number;
  total_requests_today: number;
  blocked_requests_today: number;
  active_connections: number;
  response_time_p50: number;
  response_time_p95: number;
  response_time_p99: number;
  error_rate: number;
  circuit_breakers: CircuitBreakerStatus[];
  recent_attacks: DDoSAttackPattern[];
}

interface DDoSConfig {
  enabled: boolean;
  detection: {
    rps_threshold: number;
    spike_multiplier: number;
    analysis_window_seconds: number;
    minimum_requests: number;
  };
  mitigation: {
    rate_limiting: {
      enabled: boolean;
      max_rps: number;
      burst_allowance: number;
    };
    circuit_breaker: {
      enabled: boolean;
      failure_threshold: number;
      timeout_seconds: number;
      recovery_threshold: number;
    };
    adaptive_throttling: {
      enabled: boolean;
      target_response_time: number;
      adjustment_factor: number;
    };
  };
  emergency_mode: {
    enabled: boolean;
    auto_trigger: boolean;
    trigger_threshold: number;
    max_concurrent_connections: number;
    challenge_mode: boolean;
  };
}

interface ThreatAnalysis {
  threat_score: number; // 0-100
  threat_indicators: string[];
  geographic_distribution: Array<{
    country: string;
    percentage: number;
    is_suspicious: boolean;
  }>;
  user_agent_analysis: Array<{
    user_agent: string;
    count: number;
    is_bot: boolean;
  }>;
  request_pattern_analysis: {
    entropy_score: number;
    repetition_rate: number;
    timing_pattern: 'uniform' | 'burst' | 'random';
  };
}

const DDoSProtectionDashboard: React.FC = () => {
  const { isConnected } = useWebSocket();
  const [metrics, setMetrics] = useState<DDoSMetrics | null>(null);
  const [config, setConfig] = useState<DDoSConfig | null>(null);
  const [threatAnalysis, setThreatAnalysis] = useState<ThreatAnalysis | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'overview' | 'detection' | 'mitigation' | 'analysis'>('overview');
  const [emergencyMode, setEmergencyMode] = useState(false);

  // Listen for real-time DDoS updates
  useWebSocketMessage(MessageTypes.DDOS_UPDATE, (data) => {
    if (data?.ddos_metrics) {
      setMetrics(data.ddos_metrics);
    }
    if (data?.threat_analysis) {
      setThreatAnalysis(data.threat_analysis);
    }
  });

  useEffect(() => {
    fetchDDoSData();
    const interval = setInterval(fetchDDoSData, 5000); // Update every 5s for real-time monitoring
    return () => clearInterval(interval);
  }, []);

  const fetchDDoSData = async () => {
    try {
      const [metricsResponse, configResponse, threatResponse] = await Promise.all([
        fetch('/api/ddos-protection/metrics'),
        fetch('/api/ddos-protection/config'),
        fetch('/api/ddos-protection/threat-analysis')
      ]);

      if (metricsResponse.ok) {
        const metricsData = await metricsResponse.json();
        setMetrics(metricsData.data);
      }

      if (configResponse.ok) {
        const configData = await configResponse.json();
        setConfig(configData.data);
      }

      if (threatResponse.ok) {
        const threatData = await threatResponse.json();
        setThreatAnalysis(threatData.data);
      }
    } catch (error) {
      console.error('Failed to fetch DDoS protection data:', error);
    } finally {
      setLoading(false);
    }
  };

  const updateConfig = async (newConfig: DDoSConfig) => {
    try {
      const response = await fetch('/api/ddos-protection/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newConfig)
      });

      if (response.ok) {
        setConfig(newConfig);
        await fetchDDoSData();
      }
    } catch (error) {
      console.error('Failed to update DDoS protection config:', error);
    }
  };

  const toggleEmergencyMode = async () => {
    try {
      const response = await fetch('/api/ddos-protection/emergency-mode', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !emergencyMode })
      });

      if (response.ok) {
        setEmergencyMode(!emergencyMode);
        await fetchDDoSData();
      }
    } catch (error) {
      console.error('Failed to toggle emergency mode:', error);
    }
  };

  const resetCircuitBreaker = async (name: string) => {
    try {
      await fetch('/api/ddos-protection/circuit-breaker/reset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      });
      await fetchDDoSData();
    } catch (error) {
      console.error('Failed to reset circuit breaker:', error);
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'low': return 'text-green-600 bg-green-50';
      case 'medium': return 'text-yellow-600 bg-yellow-50';
      case 'high': return 'text-orange-600 bg-orange-50';
      case 'critical': return 'text-red-600 bg-red-50';
      default: return 'text-slate-600 bg-slate-50';
    }
  };

  const getCircuitBreakerColor = (state: string) => {
    switch (state) {
      case 'closed': return 'text-green-600 bg-green-50';
      case 'open': return 'text-red-600 bg-red-50';
      case 'half-open': return 'text-yellow-600 bg-yellow-50';
      default: return 'text-slate-600 bg-slate-50';
    }
  };

  const getThreatScoreColor = (score: number) => {
    if (score >= 80) return 'text-red-600';
    if (score >= 60) return 'text-orange-600';
    if (score >= 40) return 'text-yellow-600';
    return 'text-green-600';
  };

  const formatDuration = (startTime: string, endTime?: string) => {
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const duration = end.getTime() - start.getTime();
    const minutes = Math.floor(duration / 60000);
    const seconds = Math.floor((duration % 60000) / 1000);
    return `${minutes}m ${seconds}s`;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-2 text-slate-600">Loading DDoS protection data...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">DDoS Protection Dashboard</h1>
            <p className="text-slate-600 mt-1">Real-time DDoS attack detection and mitigation</p>
          </div>
          <div className="flex items-center space-x-3">
            <div className={`text-sm font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? '🟢 Live Updates' : '🔴 No Live Data'}
            </div>
            <div className={`text-sm font-medium ${config?.enabled ? 'text-green-600' : 'text-red-600'}`}>
              {config?.enabled ? '🛡️ Protection Active' : '⚠️ Protection Disabled'}
            </div>
            <button
              onClick={toggleEmergencyMode}
              className={`px-3 py-1 rounded-md text-sm font-medium ${
                emergencyMode 
                  ? 'bg-red-600 text-white hover:bg-red-700' 
                  : 'bg-orange-600 text-white hover:bg-orange-700'
              }`}
            >
              {emergencyMode ? '🚨 Emergency Mode ON' : '⚡ Emergency Mode'}
            </button>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200">
        <nav className="-mb-px flex space-x-8">
          {[
            { id: 'overview', label: 'Overview', icon: '📊' },
            { id: 'detection', label: 'Detection', icon: '🔍' },
            { id: 'mitigation', label: 'Mitigation', icon: '🛡️' },
            { id: 'analysis', label: 'Threat Analysis', icon: '🧬' }
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
      {activeTab === 'overview' && metrics && (
        <div className="space-y-6">
          {/* Real-time Metrics */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Current RPS</p>
                  <p className="text-2xl font-bold text-slate-900">{metrics.current_rps}</p>
                </div>
                <div className="text-blue-600">
                  <span className="text-2xl">⚡</span>
                </div>
              </div>
              <div className="mt-2">
                <div className="flex items-center text-xs text-slate-500">
                  <span>1m avg: {metrics.average_rps_1m}</span>
                  <span className="mx-2">•</span>
                  <span>5m avg: {metrics.average_rps_5m}</span>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Blocked Today</p>
                  <p className="text-2xl font-bold text-red-600">{metrics.blocked_requests_today.toLocaleString()}</p>
                </div>
                <div className="text-red-600">
                  <span className="text-2xl">🚫</span>
                </div>
              </div>
              <div className="mt-2 text-xs text-slate-500">
                {((metrics.blocked_requests_today / metrics.total_requests_today) * 100).toFixed(1)}% of total
              </div>
            </div>

            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Response Time</p>
                  <p className="text-2xl font-bold text-slate-900">{metrics.response_time_p50}ms</p>
                </div>
                <div className="text-green-600">
                  <span className="text-2xl">⏱️</span>
                </div>
              </div>
              <div className="mt-2 text-xs text-slate-500">
                P95: {metrics.response_time_p95}ms | P99: {metrics.response_time_p99}ms
              </div>
            </div>

            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Error Rate</p>
                  <p className={`text-2xl font-bold ${
                    metrics.error_rate > 5 ? 'text-red-600' : 
                    metrics.error_rate > 2 ? 'text-yellow-600' : 'text-green-600'
                  }`}>
                    {metrics.error_rate.toFixed(1)}%
                  </p>
                </div>
                <div className="text-orange-600">
                  <span className="text-2xl">⚠️</span>
                </div>
              </div>
              <div className="mt-2 text-xs text-slate-500">
                {metrics.active_connections} active connections
              </div>
            </div>
          </div>

          {/* Circuit Breakers Status */}
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
              <span className="mr-2">🔌</span>
              Circuit Breakers
            </h3>
            
            {metrics.circuit_breakers.length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {metrics.circuit_breakers.map((breaker) => (
                  <div key={breaker.name} className="border border-slate-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-slate-900">{breaker.name}</h4>
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${getCircuitBreakerColor(breaker.state)}`}>
                        {breaker.state.toUpperCase()}
                      </span>
                    </div>
                    
                    <div className="space-y-1 text-sm text-slate-600">
                      <div className="flex justify-between">
                        <span>Failures:</span>
                        <span className={breaker.failure_count >= breaker.failure_threshold ? 'text-red-600 font-semibold' : ''}>
                          {breaker.failure_count}/{breaker.failure_threshold}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span>Timeout:</span>
                        <span>{breaker.timeout}s</span>
                      </div>
                      {breaker.next_attempt && (
                        <div className="flex justify-between">
                          <span>Next Attempt:</span>
                          <span>{new Date(breaker.next_attempt).toLocaleTimeString()}</span>
                        </div>
                      )}
                    </div>
                    
                    {breaker.state === 'open' && (
                      <button
                        onClick={() => resetCircuitBreaker(breaker.name)}
                        className="mt-2 w-full bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
                      >
                        Reset
                      </button>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-4 text-slate-500">No circuit breakers configured</div>
            )}
          </div>

          {/* Recent Attacks */}
          {metrics.recent_attacks.length > 0 && (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
                <span className="mr-2">⚔️</span>
                Recent Attacks
              </h3>
              
              <div className="space-y-4">
                {metrics.recent_attacks.slice(0, 5).map((attack, index) => (
                  <div key={index} className="border border-slate-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        <span className={`px-2 py-1 text-xs font-semibold rounded-full ${getSeverityColor(attack.severity)}`}>
                          {attack.severity.toUpperCase()}
                        </span>
                        <span className="font-medium">{attack.type} attack</span>
                        <span className="text-sm text-slate-500">
                          {formatDuration(attack.start_time, attack.end_time)}
                        </span>
                      </div>
                      <div className="text-sm text-slate-500">
                        {new Date(attack.start_time).toLocaleString()}
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="font-medium">Peak RPS:</span>
                        <div className="text-lg font-bold text-red-600">{attack.peak_rps.toLocaleString()}</div>
                      </div>
                      <div>
                        <span className="font-medium">Total Requests:</span>
                        <div className="text-lg font-bold">{attack.total_requests.toLocaleString()}</div>
                      </div>
                      <div>
                        <span className="font-medium">Source IPs:</span>
                        <div className="text-lg font-bold">{attack.source_ips}</div>
                      </div>
                      <div>
                        <span className="font-medium">Blocked:</span>
                        <div className="text-lg font-bold text-green-600">{attack.blocked_requests.toLocaleString()}</div>
                      </div>
                    </div>
                    
                    {attack.target_endpoints.length > 0 && (
                      <div className="mt-3">
                        <span className="font-medium text-sm">Target Endpoints:</span>
                        <div className="mt-1 flex flex-wrap gap-1">
                          {attack.target_endpoints.slice(0, 5).map((endpoint, i) => (
                            <span key={i} className="px-2 py-1 bg-slate-100 text-slate-700 text-xs rounded">
                              {endpoint}
                            </span>
                          ))}
                          {attack.target_endpoints.length > 5 && (
                            <span className="px-2 py-1 bg-slate-100 text-slate-700 text-xs rounded">
                              +{attack.target_endpoints.length - 5} more
                            </span>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Detection Tab */}
      {activeTab === 'detection' && config && (
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">Attack Detection Configuration</h3>
            
            <div className="space-y-6">
              <div>
                <label className="flex items-center mb-4">
                  <input
                    type="checkbox"
                    checked={config.enabled}
                    onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
                    className="rounded border-slate-300"
                  />
                  <span className="ml-2 text-sm text-slate-700">Enable DDoS Protection</span>
                </label>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    RPS Threshold
                  </label>
                  <input
                    type="number"
                    value={config.detection.rps_threshold}
                    onChange={(e) => setConfig({
                      ...config,
                      detection: { ...config.detection, rps_threshold: parseInt(e.target.value) }
                    })}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md"
                  />
                  <p className="text-xs text-slate-500 mt-1">Requests per second that triggers detection</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Spike Multiplier
                  </label>
                  <input
                    type="number"
                    step="0.1"
                    value={config.detection.spike_multiplier}
                    onChange={(e) => setConfig({
                      ...config,
                      detection: { ...config.detection, spike_multiplier: parseFloat(e.target.value) }
                    })}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md"
                  />
                  <p className="text-xs text-slate-500 mt-1">Multiplier for baseline traffic to detect spikes</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Analysis Window (seconds)
                  </label>
                  <input
                    type="number"
                    value={config.detection.analysis_window_seconds}
                    onChange={(e) => setConfig({
                      ...config,
                      detection: { ...config.detection, analysis_window_seconds: parseInt(e.target.value) }
                    })}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md"
                  />
                  <p className="text-xs text-slate-500 mt-1">Time window for analyzing traffic patterns</p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Minimum Requests
                  </label>
                  <input
                    type="number"
                    value={config.detection.minimum_requests}
                    onChange={(e) => setConfig({
                      ...config,
                      detection: { ...config.detection, minimum_requests: parseInt(e.target.value) }
                    })}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md"
                  />
                  <p className="text-xs text-slate-500 mt-1">Minimum requests in window to trigger analysis</p>
                </div>
              </div>

              <div className="pt-4 border-t border-slate-200">
                <button
                  onClick={() => updateConfig(config)}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                >
                  Save Detection Settings
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Mitigation Tab */}
      {activeTab === 'mitigation' && config && (
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">Mitigation Configuration</h3>
            
            <div className="space-y-8">
              {/* Rate Limiting */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3 flex items-center">
                  <span className="mr-2">🚦</span>
                  Rate Limiting
                </h4>
                <div className="space-y-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={config.mitigation.rate_limiting.enabled}
                      onChange={(e) => setConfig({
                        ...config,
                        mitigation: {
                          ...config.mitigation,
                          rate_limiting: { ...config.mitigation.rate_limiting, enabled: e.target.checked }
                        }
                      })}
                      className="rounded border-slate-300"
                    />
                    <span className="ml-2 text-sm text-slate-700">Enable Rate Limiting</span>
                  </label>
                  
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Max RPS
                      </label>
                      <input
                        type="number"
                        value={config.mitigation.rate_limiting.max_rps}
                        onChange={(e) => setConfig({
                          ...config,
                          mitigation: {
                            ...config.mitigation,
                            rate_limiting: { ...config.mitigation.rate_limiting, max_rps: parseInt(e.target.value) }
                          }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.mitigation.rate_limiting.enabled}
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Burst Allowance
                      </label>
                      <input
                        type="number"
                        value={config.mitigation.rate_limiting.burst_allowance}
                        onChange={(e) => setConfig({
                          ...config,
                          mitigation: {
                            ...config.mitigation,
                            rate_limiting: { ...config.mitigation.rate_limiting, burst_allowance: parseInt(e.target.value) }
                          }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.mitigation.rate_limiting.enabled}
                      />
                    </div>
                  </div>
                </div>
              </div>

              {/* Circuit Breaker */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3 flex items-center">
                  <span className="mr-2">🔌</span>
                  Circuit Breaker
                </h4>
                <div className="space-y-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={config.mitigation.circuit_breaker.enabled}
                      onChange={(e) => setConfig({
                        ...config,
                        mitigation: {
                          ...config.mitigation,
                          circuit_breaker: { ...config.mitigation.circuit_breaker, enabled: e.target.checked }
                        }
                      })}
                      className="rounded border-slate-300"
                    />
                    <span className="ml-2 text-sm text-slate-700">Enable Circuit Breaker</span>
                  </label>
                  
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Failure Threshold
                      </label>
                      <input
                        type="number"
                        value={config.mitigation.circuit_breaker.failure_threshold}
                        onChange={(e) => setConfig({
                          ...config,
                          mitigation: {
                            ...config.mitigation,
                            circuit_breaker: { ...config.mitigation.circuit_breaker, failure_threshold: parseInt(e.target.value) }
                          }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.mitigation.circuit_breaker.enabled}
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Timeout (seconds)
                      </label>
                      <input
                        type="number"
                        value={config.mitigation.circuit_breaker.timeout_seconds}
                        onChange={(e) => setConfig({
                          ...config,
                          mitigation: {
                            ...config.mitigation,
                            circuit_breaker: { ...config.mitigation.circuit_breaker, timeout_seconds: parseInt(e.target.value) }
                          }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.mitigation.circuit_breaker.enabled}
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Recovery Threshold
                      </label>
                      <input
                        type="number"
                        value={config.mitigation.circuit_breaker.recovery_threshold}
                        onChange={(e) => setConfig({
                          ...config,
                          mitigation: {
                            ...config.mitigation,
                            circuit_breaker: { ...config.mitigation.circuit_breaker, recovery_threshold: parseInt(e.target.value) }
                          }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.mitigation.circuit_breaker.enabled}
                      />
                    </div>
                  </div>
                </div>
              </div>

              {/* Emergency Mode */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3 flex items-center">
                  <span className="mr-2">🚨</span>
                  Emergency Mode
                </h4>
                <div className="space-y-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={config.emergency_mode.enabled}
                      onChange={(e) => setConfig({
                        ...config,
                        emergency_mode: { ...config.emergency_mode, enabled: e.target.checked }
                      })}
                      className="rounded border-slate-300"
                    />
                    <span className="ml-2 text-sm text-slate-700">Enable Emergency Mode</span>
                  </label>

                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={config.emergency_mode.auto_trigger}
                      onChange={(e) => setConfig({
                        ...config,
                        emergency_mode: { ...config.emergency_mode, auto_trigger: e.target.checked }
                      })}
                      className="rounded border-slate-300"
                      disabled={!config.emergency_mode.enabled}
                    />
                    <span className="ml-2 text-sm text-slate-700">Auto-trigger Emergency Mode</span>
                  </label>
                  
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Trigger Threshold
                      </label>
                      <input
                        type="number"
                        value={config.emergency_mode.trigger_threshold}
                        onChange={(e) => setConfig({
                          ...config,
                          emergency_mode: { ...config.emergency_mode, trigger_threshold: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.emergency_mode.enabled || !config.emergency_mode.auto_trigger}
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Max Concurrent Connections
                      </label>
                      <input
                        type="number"
                        value={config.emergency_mode.max_concurrent_connections}
                        onChange={(e) => setConfig({
                          ...config,
                          emergency_mode: { ...config.emergency_mode, max_concurrent_connections: parseInt(e.target.value) }
                        })}
                        className="w-full px-3 py-2 border border-slate-300 rounded-md"
                        disabled={!config.emergency_mode.enabled}
                      />
                    </div>
                    <div className="flex items-center">
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={config.emergency_mode.challenge_mode}
                          onChange={(e) => setConfig({
                            ...config,
                            emergency_mode: { ...config.emergency_mode, challenge_mode: e.target.checked }
                          })}
                          className="rounded border-slate-300"
                          disabled={!config.emergency_mode.enabled}
                        />
                        <span className="ml-2 text-sm text-slate-700">Challenge Mode</span>
                      </label>
                    </div>
                  </div>
                </div>
              </div>

              <div className="pt-4 border-t border-slate-200">
                <button
                  onClick={() => updateConfig(config)}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium"
                >
                  Save Mitigation Settings
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Threat Analysis Tab */}
      {activeTab === 'analysis' && threatAnalysis && (
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
              <span className="mr-2">🧬</span>
              Threat Analysis
            </h3>
            
            <div className="space-y-6">
              {/* Threat Score */}
              <div className="text-center">
                <div className={`text-6xl font-bold ${getThreatScoreColor(threatAnalysis.threat_score)}`}>
                  {threatAnalysis.threat_score}
                </div>
                <div className="text-lg text-slate-600">Threat Score (0-100)</div>
                {threatAnalysis.threat_indicators.length > 0 && (
                  <div className="mt-4">
                    <h4 className="text-sm font-medium text-slate-900 mb-2">Active Threat Indicators</h4>
                    <div className="flex flex-wrap gap-2 justify-center">
                      {threatAnalysis.threat_indicators.map((indicator, index) => (
                        <span key={index} className="px-2 py-1 bg-red-100 text-red-800 text-xs font-medium rounded">
                          {indicator}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Geographic Distribution */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3">Geographic Distribution</h4>
                <div className="space-y-2">
                  {threatAnalysis.geographic_distribution.map((geo, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-slate-50 rounded-lg">
                      <div className="flex items-center space-x-3">
                        <span className="font-medium">{geo.country}</span>
                        {geo.is_suspicious && (
                          <span className="px-2 py-1 bg-red-100 text-red-800 text-xs font-medium rounded">
                            Suspicious
                          </span>
                        )}
                      </div>
                      <div className="text-right">
                        <div className="font-bold">{geo.percentage.toFixed(1)}%</div>
                        <div className="w-16 bg-slate-200 rounded-full h-2">
                          <div 
                            className={`h-2 rounded-full ${geo.is_suspicious ? 'bg-red-500' : 'bg-blue-500'}`}
                            style={{ width: `${geo.percentage}%` }}
                          ></div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* User Agent Analysis */}
              {threatAnalysis.user_agent_analysis.length > 0 && (
                <div>
                  <h4 className="text-md font-medium text-slate-900 mb-3">User Agent Analysis</h4>
                  <div className="space-y-2">
                    {threatAnalysis.user_agent_analysis.slice(0, 5).map((ua, index) => (
                      <div key={index} className="flex items-center justify-between p-3 bg-slate-50 rounded-lg">
                        <div className="flex items-center space-x-3 flex-1 min-w-0">
                          <span className="font-mono text-sm truncate">{ua.user_agent}</span>
                          {ua.is_bot && (
                            <span className="px-2 py-1 bg-orange-100 text-orange-800 text-xs font-medium rounded">
                              Bot
                            </span>
                          )}
                        </div>
                        <div className="text-right">
                          <div className="font-bold">{ua.count}</div>
                          <div className="text-xs text-slate-500">requests</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Request Pattern Analysis */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3">Request Pattern Analysis</h4>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="text-center p-4 bg-slate-50 rounded-lg">
                    <div className="text-2xl font-bold text-slate-900">{threatAnalysis.request_pattern_analysis.entropy_score.toFixed(2)}</div>
                    <div className="text-sm text-slate-600">Entropy Score</div>
                    <div className="text-xs text-slate-500">Higher = more random</div>
                  </div>
                  <div className="text-center p-4 bg-slate-50 rounded-lg">
                    <div className="text-2xl font-bold text-slate-900">{(threatAnalysis.request_pattern_analysis.repetition_rate * 100).toFixed(1)}%</div>
                    <div className="text-sm text-slate-600">Repetition Rate</div>
                    <div className="text-xs text-slate-500">Higher = more repetitive</div>
                  </div>
                  <div className="text-center p-4 bg-slate-50 rounded-lg">
                    <div className="text-2xl font-bold text-slate-900 capitalize">{threatAnalysis.request_pattern_analysis.timing_pattern}</div>
                    <div className="text-sm text-slate-600">Timing Pattern</div>
                    <div className="text-xs text-slate-500">Request timing analysis</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DDoSProtectionDashboard;