import React, { useState, useEffect } from 'react';
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext';
import { MessageTypes } from '../services/websocket';

interface IPReputationData {
  ip: string;
  reputation_score: number; // 0-100, higher is worse
  threat_level: 'low' | 'medium' | 'high' | 'critical';
  country: string;
  city?: string;
  asn?: string;
  is_proxy: boolean;
  is_tor: boolean;
  is_vpn: boolean;
  last_seen: string;
  threat_types: string[];
  sources: string[];
}

interface IPProtectionStatus {
  total_ips_monitored: number;
  blocked_ips: number;
  whitelisted_ips: number;
  suspicious_ips: number;
  geographic_blocks: {
    blocked_countries: string[];
    blocked_regions: string[];
    total_blocked_geoips: number;
  };
  recent_blocks: Array<{
    ip: string;
    reason: string;
    timestamp: string;
    automatic: boolean;
  }>;
  top_threats: IPReputationData[];
}

interface IPProtectionConfig {
  enabled: boolean;
  auto_block_threshold: number; // reputation score threshold for auto-blocking
  geo_blocking: {
    enabled: boolean;
    blocked_countries: string[];
    blocked_regions: string[];
    allow_vpn: boolean;
    allow_proxy: boolean;
    allow_tor: boolean;
  };
  reputation_sources: {
    enabled_sources: string[];
    update_interval: number; // minutes
    cache_duration: number; // hours
  };
  whitelist: string[];
  blacklist: string[];
}

interface GeographicStats {
  country_code: string;
  country_name: string;
  request_count: number;
  threat_count: number;
  blocked_count: number;
}

const IPProtectionManager: React.FC = () => {
  const { isConnected } = useWebSocket();
  const [status, setStatus] = useState<IPProtectionStatus | null>(null);
  const [config, setConfig] = useState<IPProtectionConfig | null>(null);
  const [geoStats, setGeoStats] = useState<GeographicStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'overview' | 'config' | 'reputation' | 'geographic'>('overview');
  const [newIP, setNewIP] = useState('');
  const [bulkIPs, setBulkIPs] = useState('');

  // Listen for real-time IP protection updates
  useWebSocketMessage(MessageTypes.IP_PROTECTION_UPDATE, (data) => {
    if (data?.ip_protection) {
      setStatus(data.ip_protection);
    }
  });

  useEffect(() => {
    fetchIPProtectionData();
    const interval = setInterval(fetchIPProtectionData, 30000); // Update every 30s
    return () => clearInterval(interval);
  }, []);

  const fetchIPProtectionData = async () => {
    try {
      const [statusResponse, configResponse, geoResponse] = await Promise.all([
        fetch('/api/ip-protection/status'),
        fetch('/api/ip-protection/config'),
        fetch('/api/ip-protection/geographic-stats')
      ]);

      if (statusResponse.ok) {
        const statusData = await statusResponse.json();
        setStatus(statusData.data);
      }

      if (configResponse.ok) {
        const configData = await configResponse.json();
        setConfig(configData.data);
      }

      if (geoResponse.ok) {
        const geoData = await geoResponse.json();
        setGeoStats(geoData.data);
      }
    } catch (error) {
      console.error('Failed to fetch IP protection data:', error);
    } finally {
      setLoading(false);
    }
  };

  const updateConfig = async (newConfig: IPProtectionConfig) => {
    try {
      const response = await fetch('/api/ip-protection/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newConfig)
      });

      if (response.ok) {
        setConfig(newConfig);
        await fetchIPProtectionData();
      }
    } catch (error) {
      console.error('Failed to update IP protection config:', error);
    }
  };

  const addToWhitelist = async (ip: string) => {
    if (!config) return;
    try {
      const updatedConfig = {
        ...config,
        whitelist: [...config.whitelist, ip]
      };
      await updateConfig(updatedConfig);
      setNewIP('');
    } catch (error) {
      console.error('Failed to add IP to whitelist:', error);
    }
  };

  const addToBlacklist = async (ip: string) => {
    if (!config) return;
    try {
      const updatedConfig = {
        ...config,
        blacklist: [...config.blacklist, ip]
      };
      await updateConfig(updatedConfig);
      setNewIP('');
    } catch (error) {
      console.error('Failed to add IP to blacklist:', error);
    }
  };

  const removeFromList = async (ip: string, listType: 'whitelist' | 'blacklist') => {
    if (!config) return;
    try {
      const updatedConfig = {
        ...config,
        [listType]: config[listType].filter(item => item !== ip)
      };
      await updateConfig(updatedConfig);
    } catch (error) {
      console.error(`Failed to remove IP from ${listType}:`, error);
    }
  };

  const bulkAddIPs = async (ips: string, listType: 'whitelist' | 'blacklist') => {
    if (!config) return;
    try {
      const ipList = ips.split('\n').map(ip => ip.trim()).filter(ip => ip);
      const updatedConfig = {
        ...config,
        [listType]: [...config[listType], ...ipList]
      };
      await updateConfig(updatedConfig);
      setBulkIPs('');
    } catch (error) {
      console.error(`Failed to bulk add IPs to ${listType}:`, error);
    }
  };

  const blockCountry = async (countryCode: string) => {
    if (!config) return;
    try {
      const updatedConfig = {
        ...config,
        geo_blocking: {
          ...config.geo_blocking,
          blocked_countries: [...config.geo_blocking.blocked_countries, countryCode]
        }
      };
      await updateConfig(updatedConfig);
    } catch (error) {
      console.error('Failed to block country:', error);
    }
  };

  const unblockCountry = async (countryCode: string) => {
    if (!config) return;
    try {
      const updatedConfig = {
        ...config,
        geo_blocking: {
          ...config.geo_blocking,
          blocked_countries: config.geo_blocking.blocked_countries.filter(c => c !== countryCode)
        }
      };
      await updateConfig(updatedConfig);
    } catch (error) {
      console.error('Failed to unblock country:', error);
    }
  };

  const getThreatLevelColor = (level: string) => {
    switch (level) {
      case 'low': return 'text-green-600 bg-green-50';
      case 'medium': return 'text-yellow-600 bg-yellow-50';
      case 'high': return 'text-orange-600 bg-orange-50';
      case 'critical': return 'text-red-600 bg-red-50';
      default: return 'text-slate-600 bg-slate-50';
    }
  };

  const getReputationColor = (score: number) => {
    if (score >= 80) return 'text-red-600';
    if (score >= 60) return 'text-orange-600';
    if (score >= 40) return 'text-yellow-600';
    return 'text-green-600';
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-2 text-slate-600">Loading IP protection data...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900">IP Protection Manager</h1>
            <p className="text-slate-600 mt-1">Monitor and manage IP-based security policies</p>
          </div>
          <div className="flex items-center space-x-3">
            <div className={`text-sm font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? '🟢 Live Updates' : '🔴 No Live Data'}
            </div>
            <div className={`text-sm font-medium ${config?.enabled ? 'text-green-600' : 'text-red-600'}`}>
              {config?.enabled ? '🛡️ Protection Active' : '⚠️ Protection Disabled'}
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
            { id: 'reputation', label: 'Reputation', icon: '🔍' },
            { id: 'geographic', label: 'Geographic', icon: '🌍' }
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
      {activeTab === 'overview' && status && (
        <div className="space-y-6">
          {/* Status Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Total IPs Monitored</p>
                  <p className="text-2xl font-bold text-slate-900">{status.total_ips_monitored}</p>
                </div>
                <div className="text-blue-600">
                  <span className="text-2xl">👁️</span>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Blocked IPs</p>
                  <p className="text-2xl font-bold text-red-600">{status.blocked_ips}</p>
                </div>
                <div className="text-red-600">
                  <span className="text-2xl">🚫</span>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Whitelisted IPs</p>
                  <p className="text-2xl font-bold text-green-600">{status.whitelisted_ips}</p>
                </div>
                <div className="text-green-600">
                  <span className="text-2xl">✅</span>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-600">Suspicious IPs</p>
                  <p className="text-2xl font-bold text-yellow-600">{status.suspicious_ips}</p>
                </div>
                <div className="text-yellow-600">
                  <span className="text-2xl">⚠️</span>
                </div>
              </div>
            </div>
          </div>

          {/* Geographic Blocking Status */}
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
              <span className="mr-2">🌍</span>
              Geographic Blocking
            </h3>
            
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="text-center">
                <div className="text-2xl font-bold text-slate-900">{status.geographic_blocks.blocked_countries.length}</div>
                <div className="text-sm text-slate-600">Blocked Countries</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-slate-900">{status.geographic_blocks.blocked_regions.length}</div>
                <div className="text-sm text-slate-600">Blocked Regions</div>
              </div>
              <div className="text-center">
                <div className="text-2xl font-bold text-red-600">{status.geographic_blocks.total_blocked_geoips}</div>
                <div className="text-sm text-slate-600">Geo-blocked IPs</div>
              </div>
            </div>

            {status.geographic_blocks.blocked_countries.length > 0 && (
              <div className="mt-4 pt-4 border-t border-slate-200">
                <h4 className="text-sm font-medium text-slate-900 mb-2">Blocked Countries</h4>
                <div className="flex flex-wrap gap-2">
                  {status.geographic_blocks.blocked_countries.map((country) => (
                    <span key={country} className="px-2 py-1 bg-red-100 text-red-800 text-xs font-medium rounded">
                      {country}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Recent Blocks */}
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
              <span className="mr-2">🚨</span>
              Recent Blocks
            </h3>
            
            {status.recent_blocks.length > 0 ? (
              <div className="space-y-3">
                {status.recent_blocks.slice(0, 5).map((block, index) => (
                  <div key={index} className="flex items-center justify-between p-3 bg-slate-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <span className="font-mono text-sm text-slate-900">{block.ip}</span>
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                        block.automatic ? 'bg-orange-100 text-orange-800' : 'bg-blue-100 text-blue-800'
                      }`}>
                        {block.automatic ? 'Auto' : 'Manual'}
                      </span>
                    </div>
                    <div className="text-right">
                      <div className="text-sm text-slate-900">{block.reason}</div>
                      <div className="text-xs text-slate-500">{new Date(block.timestamp).toLocaleString()}</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-4 text-slate-500">No recent blocks</div>
            )}
          </div>

          {/* Top Threats */}
          {status.top_threats.length > 0 && (
            <div className="bg-white rounded-lg border border-slate-200 p-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4 flex items-center">
                <span className="mr-2">⚡</span>
                Top Threats
              </h3>
              
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-200">
                  <thead className="bg-slate-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">IP Address</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Country</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Reputation</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Threat Level</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Types</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-slate-200">
                    {status.top_threats.slice(0, 5).map((threat) => (
                      <tr key={threat.ip}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-slate-900">
                          {threat.ip}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                          {threat.country}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          <span className={`font-semibold ${getReputationColor(threat.reputation_score)}`}>
                            {threat.reputation_score}/100
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getThreatLevelColor(threat.threat_level)}`}>
                            {threat.threat_level}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                          {threat.threat_types.slice(0, 2).join(', ')}
                          {threat.threat_types.length > 2 && '...'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          <button
                            onClick={() => addToBlacklist(threat.ip)}
                            className="text-red-600 hover:text-red-900 mr-3"
                          >
                            Block
                          </button>
                          <button
                            onClick={() => addToWhitelist(threat.ip)}
                            className="text-green-600 hover:text-green-900"
                          >
                            Whitelist
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Configuration Tab */}
      {activeTab === 'config' && config && (
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">IP Protection Configuration</h3>
            
            <div className="space-y-6">
              {/* Global Settings */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3">Global Settings</h4>
                <div className="space-y-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={config.enabled}
                      onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
                      className="rounded border-slate-300"
                    />
                    <span className="ml-2 text-sm text-slate-700">Enable IP Protection</span>
                  </label>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      Auto-block Reputation Threshold (0-100)
                    </label>
                    <input
                      type="number"
                      min="0"
                      max="100"
                      value={config.auto_block_threshold}
                      onChange={(e) => setConfig({
                        ...config,
                        auto_block_threshold: parseInt(e.target.value)
                      })}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md"
                    />
                    <p className="text-xs text-slate-500 mt-1">
                      IPs with reputation scores above this threshold will be automatically blocked
                    </p>
                  </div>
                </div>
              </div>

              {/* Geographic Blocking */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3">Geographic Blocking</h4>
                <div className="space-y-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={config.geo_blocking.enabled}
                      onChange={(e) => setConfig({
                        ...config,
                        geo_blocking: { ...config.geo_blocking, enabled: e.target.checked }
                      })}
                      className="rounded border-slate-300"
                    />
                    <span className="ml-2 text-sm text-slate-700">Enable Geographic Blocking</span>
                  </label>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={config.geo_blocking.allow_vpn}
                        onChange={(e) => setConfig({
                          ...config,
                          geo_blocking: { ...config.geo_blocking, allow_vpn: e.target.checked }
                        })}
                        className="rounded border-slate-300"
                      />
                      <span className="ml-2 text-sm text-slate-700">Allow VPN</span>
                    </label>

                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={config.geo_blocking.allow_proxy}
                        onChange={(e) => setConfig({
                          ...config,
                          geo_blocking: { ...config.geo_blocking, allow_proxy: e.target.checked }
                        })}
                        className="rounded border-slate-300"
                      />
                      <span className="ml-2 text-sm text-slate-700">Allow Proxy</span>
                    </label>

                    <label className="flex items-center">
                      <input
                        type="checkbox"
                        checked={config.geo_blocking.allow_tor}
                        onChange={(e) => setConfig({
                          ...config,
                          geo_blocking: { ...config.geo_blocking, allow_tor: e.target.checked }
                        })}
                        className="rounded border-slate-300"
                      />
                      <span className="ml-2 text-sm text-slate-700">Allow Tor</span>
                    </label>
                  </div>
                </div>
              </div>

              {/* IP Lists Management */}
              <div>
                <h4 className="text-md font-medium text-slate-900 mb-3">IP Lists Management</h4>
                
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {/* Whitelist */}
                  <div>
                    <h5 className="text-sm font-medium text-slate-700 mb-2">Whitelist</h5>
                    <div className="space-y-2">
                      <div className="flex space-x-2">
                        <input
                          type="text"
                          placeholder="Enter IP address"
                          value={newIP}
                          onChange={(e) => setNewIP(e.target.value)}
                          className="flex-1 px-3 py-2 border border-slate-300 rounded-md"
                        />
                        <button
                          onClick={() => addToWhitelist(newIP)}
                          className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md"
                        >
                          Add
                        </button>
                      </div>
                      
                      <div className="max-h-32 overflow-y-auto border border-slate-200 rounded-md">
                        {config.whitelist.length > 0 ? (
                          config.whitelist.map((ip, index) => (
                            <div key={index} className="flex items-center justify-between p-2 border-b border-slate-100 last:border-b-0">
                              <span className="font-mono text-sm">{ip}</span>
                              <button
                                onClick={() => removeFromList(ip, 'whitelist')}
                                className="text-red-600 hover:text-red-800 text-xs"
                              >
                                Remove
                              </button>
                            </div>
                          ))
                        ) : (
                          <div className="p-2 text-center text-slate-500 text-sm">No whitelisted IPs</div>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Blacklist */}
                  <div>
                    <h5 className="text-sm font-medium text-slate-700 mb-2">Blacklist</h5>
                    <div className="space-y-2">
                      <div className="flex space-x-2">
                        <input
                          type="text"
                          placeholder="Enter IP address"
                          value={newIP}
                          onChange={(e) => setNewIP(e.target.value)}
                          className="flex-1 px-3 py-2 border border-slate-300 rounded-md"
                        />
                        <button
                          onClick={() => addToBlacklist(newIP)}
                          className="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-md"
                        >
                          Add
                        </button>
                      </div>
                      
                      <div className="max-h-32 overflow-y-auto border border-slate-200 rounded-md">
                        {config.blacklist.length > 0 ? (
                          config.blacklist.map((ip, index) => (
                            <div key={index} className="flex items-center justify-between p-2 border-b border-slate-100 last:border-b-0">
                              <span className="font-mono text-sm">{ip}</span>
                              <button
                                onClick={() => removeFromList(ip, 'blacklist')}
                                className="text-red-600 hover:text-red-800 text-xs"
                              >
                                Remove
                              </button>
                            </div>
                          ))
                        ) : (
                          <div className="p-2 text-center text-slate-500 text-sm">No blacklisted IPs</div>
                        )}
                      </div>
                    </div>
                  </div>
                </div>

                {/* Bulk Import */}
                <div className="mt-4 pt-4 border-t border-slate-200">
                  <h5 className="text-sm font-medium text-slate-700 mb-2">Bulk Import</h5>
                  <div className="space-y-2">
                    <textarea
                      placeholder="Enter IP addresses, one per line"
                      value={bulkIPs}
                      onChange={(e) => setBulkIPs(e.target.value)}
                      rows={4}
                      className="w-full px-3 py-2 border border-slate-300 rounded-md"
                    />
                    <div className="flex space-x-2">
                      <button
                        onClick={() => bulkAddIPs(bulkIPs, 'whitelist')}
                        className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md"
                      >
                        Add to Whitelist
                      </button>
                      <button
                        onClick={() => bulkAddIPs(bulkIPs, 'blacklist')}
                        className="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-md"
                      >
                        Add to Blacklist
                      </button>
                    </div>
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
        </div>
      )}

      {/* Geographic Tab */}
      {activeTab === 'geographic' && (
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">Geographic Analytics</h3>
            
            {geoStats.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-200">
                  <thead className="bg-slate-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Country</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Requests</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Threats</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Blocked</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Threat Rate</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-slate-200">
                    {geoStats.map((stat) => {
                      const threatRate = stat.request_count > 0 ? (stat.threat_count / stat.request_count * 100) : 0;
                      const isBlocked = config?.geo_blocking.blocked_countries.includes(stat.country_code);
                      
                      return (
                        <tr key={stat.country_code}>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                            {stat.country_name} ({stat.country_code})
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                            {stat.request_count.toLocaleString()}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                            {stat.threat_count}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-900">
                            {stat.blocked_count}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm">
                            <span className={`font-semibold ${
                              threatRate > 10 ? 'text-red-600' :
                              threatRate > 5 ? 'text-yellow-600' : 'text-green-600'
                            }`}>
                              {threatRate.toFixed(1)}%
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm">
                            {isBlocked ? (
                              <button
                                onClick={() => unblockCountry(stat.country_code)}
                                className="text-green-600 hover:text-green-900"
                              >
                                Unblock
                              </button>
                            ) : (
                              <button
                                onClick={() => blockCountry(stat.country_code)}
                                className="text-red-600 hover:text-red-900"
                              >
                                Block
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="text-center py-8 text-slate-500">
                No geographic data available.
              </div>
            )}
          </div>
        </div>
      )}

      {/* Reputation Tab */}
      {activeTab === 'reputation' && status && (
        <div className="space-y-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">IP Reputation Analysis</h3>
            
            {status.top_threats.length > 0 ? (
              <div className="space-y-4">
                {status.top_threats.map((threat) => (
                  <div key={threat.ip} className="border border-slate-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center space-x-3">
                        <span className="font-mono text-lg font-semibold">{threat.ip}</span>
                        <span className={`px-2 py-1 text-xs font-semibold rounded-full ${getThreatLevelColor(threat.threat_level)}`}>
                          {threat.threat_level}
                        </span>
                        <span className={`font-semibold ${getReputationColor(threat.reputation_score)}`}>
                          {threat.reputation_score}/100
                        </span>
                      </div>
                      <div className="flex space-x-2">
                        <button
                          onClick={() => addToWhitelist(threat.ip)}
                          className="bg-green-600 hover:bg-green-700 text-white px-3 py-1 rounded text-sm"
                        >
                          Whitelist
                        </button>
                        <button
                          onClick={() => addToBlacklist(threat.ip)}
                          className="bg-red-600 hover:bg-red-700 text-white px-3 py-1 rounded text-sm"
                        >
                          Block
                        </button>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                      <div>
                        <p><span className="font-medium">Location:</span> {threat.city ? `${threat.city}, ` : ''}{threat.country}</p>
                        <p><span className="font-medium">ASN:</span> {threat.asn || 'Unknown'}</p>
                        <p><span className="font-medium">Last Seen:</span> {new Date(threat.last_seen).toLocaleString()}</p>
                      </div>
                      <div>
                        <p><span className="font-medium">Proxy:</span> {threat.is_proxy ? '✅' : '❌'}</p>
                        <p><span className="font-medium">VPN:</span> {threat.is_vpn ? '✅' : '❌'}</p>
                        <p><span className="font-medium">Tor:</span> {threat.is_tor ? '✅' : '❌'}</p>
                      </div>
                    </div>
                    
                    {threat.threat_types.length > 0 && (
                      <div className="mt-3">
                        <p className="font-medium text-sm mb-1">Threat Types:</p>
                        <div className="flex flex-wrap gap-1">
                          {threat.threat_types.map((type, index) => (
                            <span key={index} className="px-2 py-1 bg-red-100 text-red-800 text-xs rounded">
                              {type}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                    
                    {threat.sources.length > 0 && (
                      <div className="mt-3">
                        <p className="font-medium text-sm mb-1">Sources:</p>
                        <div className="flex flex-wrap gap-1">
                          {threat.sources.map((source, index) => (
                            <span key={index} className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
                              {source}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-slate-500">
                No threat intelligence data available.
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default IPProtectionManager;