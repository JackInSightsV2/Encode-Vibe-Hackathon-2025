import React, { useState, useEffect } from 'react'
import { apiGet } from '../utils/api'
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext'
import { MessageTypes } from '../services/websocket'

interface AdvancedDashboardProps {
  systemStatus: any
}

interface ModerationStats {
  total_requests: number
  blocked_requests: number  
  flagged_requests: number
  warned_requests: number
  average_score: number
  average_latency: string
  layer_count: number
  layer_stats: LayerStats[]
  cache_hit_rate: number
  cache_size: number
}

interface LayerStats {
  name: string
  enabled: boolean
  weight: number
  total_requests: number
  successful_runs: number
  failed_runs: number
  success_rate: number
  average_latency: string
  rules_triggered?: number
  actions_blocked?: number
  actions_flagged?: number
  actions_warned?: number
}

interface RuleEngineStats {
  total_rules: number
  active_rules: number
  rules_executed: number
  rules_matched: number
  average_exec_time: string
  last_reload: string
  reload_count: number
  error_count: number
  rule_type_stats: Record<string, any>
}

const AdvancedDashboard: React.FC<AdvancedDashboardProps> = () => {
  const { isConnected } = useWebSocket()
  const [moderationStats, setModerationStats] = useState<ModerationStats | null>(null)
  const [ruleEngineStats, setRuleEngineStats] = useState<RuleEngineStats | null>(null)
  // Initialize recent moderations from localStorage if available
  const [recentModerations, setRecentModerations] = useState<any[]>(() => {
    try {
      const saved = localStorage.getItem('recentModerations')
      return saved ? JSON.parse(saved) : []
    } catch {
      return []
    }
  })
  const [alertsCount, setAlertsCount] = useState(0)
  const [piiAnalytics, setPIIAnalytics] = useState<any>(null)
  const [recentPIIEvents, setRecentPIIEvents] = useState<any[]>(() => {
    try {
      const saved = localStorage.getItem('recentPIIEvents')
      return saved ? JSON.parse(saved) : []
    } catch {
      return []
    }
  })

  useEffect(() => {
    fetchModerationStats()
    fetchRuleEngineStats()
    fetchPIIAnalytics()
    const interval = setInterval(() => {
      fetchModerationStats()
      fetchRuleEngineStats()
      fetchPIIAnalytics()
    }, 5000)
    return () => clearInterval(interval)
  }, [])

  // Save recent moderations to localStorage whenever they change
  useEffect(() => {
    try {
      localStorage.setItem('recentModerations', JSON.stringify(recentModerations))
    } catch (error) {
      console.error('Failed to save recent moderations to localStorage:', error)
    }
  }, [recentModerations])

  // Save recent PII events to localStorage whenever they change
  useEffect(() => {
    try {
      localStorage.setItem('recentPIIEvents', JSON.stringify(recentPIIEvents))
    } catch (error) {
      console.error('Failed to save recent PII events to localStorage:', error)
    }
  }, [recentPIIEvents])

  // Debug effect to track recentModerations changes
  useEffect(() => {
    console.log('🔍 recentModerations state changed:', recentModerations.length, 'events')
    recentModerations.forEach((event, index) => {
      console.log(`  ${index + 1}. ${event.action} - ${event.final_score} - ${event.timestamp}`)
    })
  }, [recentModerations])

  // Listen for real-time moderation events
  useWebSocketMessage(MessageTypes.MODERATION_EVENT, (data, message) => {
    console.log('🔥 MODERATION EVENT RECEIVED:', {
      messageId: message?.id,
      timestamp: message?.timestamp,
      userId: data?.user_id,
      score: data?.final_score,
      action: data?.action
    })
    
    if (data) {
      // Check if the event is recent (within last 5 minutes) to prevent old replayed messages
      const eventTime = new Date(message?.timestamp || new Date())
      const now = new Date()
      const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000)
      
      if (eventTime < fiveMinutesAgo) {
        console.log('⏰ Skipping old moderation event:', message?.timestamp)
        return
      }
      
      // Create a unique ID for this event to prevent duplicates
      const eventId = `${message?.timestamp || Date.now()}-${data.user_id || 'unknown'}-${data.final_score || 0}`
      
      // Create a moderation event from the WebSocket data
      const moderationEvent = {
        id: eventId,
        timestamp: message?.timestamp || new Date().toISOString(), // Use backend timestamp
        final_score: data.final_score || 0,
        action: data.action || 'unknown',
        blocked: data.final_decision || false,
        content: `Content moderated (score: ${data.final_score?.toFixed(3) || '0.000'}) - User: ${data.user_id || 'unknown'}`,
        primary_category: 'automated',
        triggered_rules: [],
        context: { user_id: data.user_id || 'unknown' },
        process_time: `${data.process_time || 0}ms`
      }
      
      console.log('📝 Adding moderation event with ID:', eventId)
      setRecentModerations(prev => {
        // Check if this event already exists to prevent duplicates
        const existingEvent = prev.find(event => event.id === eventId)
        if (existingEvent) {
          console.log('⚠️ Duplicate event detected, skipping:', eventId)
          return prev
        }
        
        const newList = [moderationEvent, ...prev.slice(0, 19)]
        console.log('📋 Updated moderation events list length:', newList.length)
        return newList
      })
      
      if (data.final_decision) {
        setAlertsCount(prev => prev + 1)
        console.log('🚨 Alert count increased for blocked content')
      }
      
      // Also refresh stats to get updated counts
      fetchModerationStats()
    } else {
      console.log('⚠️ Received moderation event with no data')
    }
  })

  // Listen for real-time PII detection events
  useWebSocketMessage(MessageTypes.PII_DETECTION, (data, message) => {
    if (data?.pii_detections) {
      // Check if the event is recent (within last 5 minutes) to prevent old replayed messages
      const eventTime = new Date(message?.timestamp || new Date())
      const now = new Date()
      const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000)
      
      if (eventTime < fiveMinutesAgo) {
        console.log('⏰ Skipping old PII event:', message?.timestamp)
        return
      }
      
      // Add timestamp to the PII event
      const piiEventWithTimestamp = {
        ...data,
        timestamp: message?.timestamp || new Date().toISOString()
      }
      setRecentPIIEvents(prev => [piiEventWithTimestamp, ...prev.slice(0, 9)])
      if (data.blocked) {
        setAlertsCount(prev => prev + 1)
      }
    }
  })

  const fetchModerationStats = async () => {
    try {
      console.log('Fetching moderation stats...')
      const data = await apiGet('/api/moderation/stats')
      console.log('Moderation stats response:', data)
      setModerationStats(data.data)
    } catch (error) {
      console.error('Failed to fetch moderation stats:', error)
    }
  }

  const fetchRuleEngineStats = async () => {
    try {
      console.log('Fetching rule engine stats...')
      const data = await apiGet('/api/moderation/rules/stats')
      console.log('Rule engine stats response:', data)
      setRuleEngineStats(data.data)
    } catch (error) {
      console.error('Failed to fetch rule engine stats:', error)
    }
  }

  const fetchPIIAnalytics = async () => {
    try {
      console.log('Fetching PII analytics...')
      const data = await apiGet('/api/moderation/pii-analytics')
      console.log('PII analytics response:', data)
      setPIIAnalytics(data.data)
    } catch (error) {
      console.error('Failed to fetch PII analytics:', error)
    }
  }

  const getSeverityColor = (score: number) => {
    if (score >= 0.9) return 'text-red-600 bg-red-50 border-red-200'
    if (score >= 0.7) return 'text-orange-600 bg-orange-50 border-orange-200'
    if (score >= 0.5) return 'text-yellow-600 bg-yellow-50 border-yellow-200'
    return 'text-green-600 bg-green-50 border-green-200'
  }

  const getActionColor = (action: string) => {
    switch (action.toLowerCase()) {
      case 'block': return 'bg-red-100 text-red-800'
      case 'flag': return 'bg-orange-100 text-orange-800'
      case 'warn': return 'bg-yellow-100 text-yellow-800'
      default: return 'bg-green-100 text-green-800'
    }
  }

  return (
    <div className="space-y-8">
      {/* Header with Alerts */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-slate-900 mb-2">Advanced Moderation Dashboard</h1>
            <p className="text-slate-600">Real-time monitoring of multi-layer content moderation system</p>
          </div>
          <div className="flex items-center space-x-4">
            {alertsCount > 0 && (
              <div className="bg-red-100 border border-red-200 rounded-lg px-3 py-2">
                <div className="flex items-center space-x-2">
                  <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></div>
                  <span className="text-red-700 font-medium text-sm">{alertsCount} New Alerts</span>
                </div>
              </div>
            )}
            <button
              onClick={() => {
                setRecentModerations([])
                setRecentPIIEvents([])
                setAlertsCount(0)
                localStorage.removeItem('recentModerations')
                localStorage.removeItem('recentPIIEvents')
                console.log('🗑️ Cleared all recent events')
              }}
              className="px-3 py-1 text-xs bg-gray-100 hover:bg-gray-200 text-gray-600 rounded-md transition-colors"
            >
              Clear Events
            </button>
            <div className={`text-sm font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? '🟢 Live Monitoring' : '🔴 Offline'}
            </div>
          </div>
        </div>
      </div>

      {/* Key Performance Indicators */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Total Requests */}
        <div className="bg-gradient-to-br from-blue-50 to-blue-100 border border-blue-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Total Requests</h3>
            <span className="text-2xl">📊</span>
          </div>
          <div className="text-3xl font-bold text-blue-700">
            {moderationStats?.total_requests?.toLocaleString() || '0'}
          </div>
          <p className="text-xs text-slate-500 mt-2">
            Avg: {moderationStats?.average_latency || '0ms'} response time
          </p>
        </div>

        {/* Blocked Content */}
        <div className="bg-gradient-to-br from-red-50 to-red-100 border border-red-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Blocked Content</h3>
            <span className="text-2xl">🛡️</span>
          </div>
          <div className="text-3xl font-bold text-red-700">
            {moderationStats?.blocked_requests?.toLocaleString() || '0'}
          </div>
          <p className="text-xs text-slate-500 mt-2">
            {moderationStats?.total_requests ? 
              ((moderationStats.blocked_requests / moderationStats.total_requests) * 100).toFixed(1) : '0'}% blocked
          </p>
        </div>

        {/* Active Rules */}
        <div className="bg-gradient-to-br from-green-50 to-green-100 border border-green-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Active Rules</h3>
            <span className="text-2xl">⚙️</span>
          </div>
          <div className="text-3xl font-bold text-green-700">
            {ruleEngineStats?.active_rules || '0'}
          </div>
          <p className="text-xs text-slate-500 mt-2">
            {ruleEngineStats?.total_rules || '0'} total rules configured
          </p>
        </div>

        {/* Cache Performance */}
        <div className="bg-gradient-to-br from-purple-50 to-purple-100 border border-purple-200 rounded-xl p-6">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Cache Hit Rate</h3>
            <span className="text-2xl">⚡</span>
          </div>
          <div className="text-3xl font-bold text-purple-700">
            {moderationStats?.cache_hit_rate?.toFixed(1) || '0'}%
          </div>
          <p className="text-xs text-slate-500 mt-2">
            {moderationStats?.cache_size || '0'} cached entries
          </p>
        </div>
      </div>

      {/* Moderation Layers Overview */}
      <div className="bg-white border border-slate-200 rounded-xl p-6">
        <h2 className="text-xl font-bold text-slate-900 mb-6">Moderation Layers Performance</h2>
        
        {moderationStats?.layer_stats ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {moderationStats.layer_stats
              .sort((a, b) => {
                // Define the desired order: regex, pii, custom rules (everything else)
                const getOrder = (name: string) => {
                  if (name.toLowerCase().includes('regex')) return 1;
                  if (name.toLowerCase().includes('pii')) return 2;
                  return 3; // custom rules and other layers
                };
                return getOrder(a.name) - getOrder(b.name);
              })
              .map((layer, index) => (
              <div key={index} className="border border-slate-200 rounded-lg p-4">
                <div className="flex items-center justify-between mb-3">
                  <h3 className="font-semibold text-slate-900">{layer.name}</h3>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                    layer.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  }`}>
                    {layer.enabled ? 'Active' : 'Disabled'}
                  </span>
                </div>
                
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-600">Weight:</span>
                    <span className="font-medium">{layer.weight}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-600">Requests:</span>
                    <span className="font-medium">{layer.total_requests?.toLocaleString()}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-600">Success Rate:</span>
                    <span className={`font-medium ${layer.success_rate > 0.95 ? 'text-green-600' : 'text-yellow-600'}`}>
                      {(layer.success_rate * 100).toFixed(1)}%
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-600">Avg Latency:</span>
                    <span className="font-medium">{layer.average_latency}</span>
                  </div>
                  
                  {/* Layer-specific stats */}
                  {layer.actions_blocked && (
                    <div className="pt-2 border-t border-slate-100">
                      <div className="flex justify-between text-xs">
                        <span className="text-slate-500">Blocked:</span>
                        <span className="text-red-600 font-medium">{layer.actions_blocked}</span>
                      </div>
                      <div className="flex justify-between text-xs">
                        <span className="text-slate-500">Flagged:</span>
                        <span className="text-orange-600 font-medium">{layer.actions_flagged}</span>
                      </div>
                      <div className="flex justify-between text-xs">
                        <span className="text-slate-500">Warned:</span>
                        <span className="text-yellow-600 font-medium">{layer.actions_warned}</span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-slate-500">
            <span className="text-4xl mb-2 block">📊</span>
            No layer statistics available
          </div>
        )}
      </div>

      {/* PII Detection Analytics */}
      {piiAnalytics && (
        <div className="bg-white border border-slate-200 rounded-xl p-6">
          <h2 className="text-xl font-bold text-slate-900 mb-6">🔒 PII Detection Analytics</h2>
          
          {/* PII Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-gradient-to-br from-indigo-50 to-indigo-100 border border-indigo-200 rounded-lg p-4">
              <div className="text-sm font-medium text-slate-700 mb-1">Total Scanned</div>
              <div className="text-2xl font-bold text-indigo-700">
                {piiAnalytics.summary?.total_scanned?.toLocaleString() || '0'}
              </div>
            </div>
            <div className="bg-gradient-to-br from-amber-50 to-amber-100 border border-amber-200 rounded-lg p-4">
              <div className="text-sm font-medium text-slate-700 mb-1">PII Detected</div>
              <div className="text-2xl font-bold text-amber-700">
                {piiAnalytics.summary?.pii_detected?.toLocaleString() || '0'}
              </div>
              <div className="text-xs text-slate-500 mt-1">
                {piiAnalytics.summary?.detection_rate || '0'}% detection rate
              </div>
            </div>
            <div className="bg-gradient-to-br from-emerald-50 to-emerald-100 border border-emerald-200 rounded-lg p-4">
              <div className="text-sm font-medium text-slate-700 mb-1">Accuracy Rate</div>
              <div className="text-2xl font-bold text-emerald-700">
                {piiAnalytics.summary?.accuracy_rate || '0'}%
              </div>
            </div>
            <div className="bg-gradient-to-br from-rose-50 to-rose-100 border border-rose-200 rounded-lg p-4">
              <div className="text-sm font-medium text-slate-700 mb-1">Prevented Exposures</div>
              <div className="text-2xl font-bold text-rose-700">
                {piiAnalytics.risk_assessment?.prevented_exposures?.toLocaleString() || '0'}
              </div>
            </div>
          </div>

          {/* PII Types Breakdown */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div>
              <h3 className="font-semibold text-slate-900 mb-4">PII Types Performance</h3>
              <div className="space-y-3">
                {piiAnalytics.pii_types && Object.entries(piiAnalytics.pii_types).map(([type, stats]: [string, any]) => (
                  <div key={type} className="border border-slate-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="font-medium capitalize text-slate-900">{type.replace('_', ' ')}</span>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                        stats.accuracy > 95 ? 'bg-green-100 text-green-800' : 
                        stats.accuracy > 90 ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'
                      }`}>
                        {stats.accuracy}% accurate
                      </span>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <div className="text-slate-600">Detections</div>
                        <div className="font-medium">{stats.count}</div>
                      </div>
                      <div>
                        <div className="text-slate-600">Confidence</div>
                        <div className="font-medium">{(stats.avg_conf * 100).toFixed(1)}%</div>
                      </div>
                      <div>
                        <div className="text-slate-600">False Pos</div>
                        <div className={`font-medium ${stats.false_pos === 0 ? 'text-green-600' : 'text-red-600'}`}>
                          {stats.false_pos}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="font-semibold text-slate-900 mb-4">Risk Assessment</h3>
              <div className="space-y-3">
                <div className="border border-slate-200 rounded-lg p-4">
                  <div className="text-sm text-slate-600 mb-2">Risk Distribution</div>
                  <div className="space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-red-600">High Risk</span>
                      <span className="font-medium text-red-700">
                        {piiAnalytics.risk_assessment?.high_risk_events || 0}
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-orange-600">Medium Risk</span>
                      <span className="font-medium text-orange-700">
                        {piiAnalytics.risk_assessment?.medium_risk_events || 0}
                      </span>
                    </div>
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-yellow-600">Low Risk</span>
                      <span className="font-medium text-yellow-700">
                        {piiAnalytics.risk_assessment?.low_risk_events || 0}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Recent PII Events */}
                <div className="border border-slate-200 rounded-lg p-4">
                  <div className="text-sm text-slate-600 mb-3">Recent PII Events</div>
                  <div className="space-y-2 max-h-40 overflow-y-auto">
                    {recentPIIEvents.length > 0 ? (
                      recentPIIEvents.map((event, index) => (
                        <div key={index} className="text-xs p-2 bg-slate-50 rounded border-l-2 border-l-amber-400">
                          <div className="flex justify-between items-center mb-1">
                            <span className="font-medium">
                              {event.pii_detections?.length || 0} PII types detected
                            </span>
                            <span className="text-slate-500">
                              {new Date(event.timestamp || Date.now()).toLocaleTimeString()}
                            </span>
                          </div>
                          <div className="text-slate-600">
                            User: {event.user_id || 'Anonymous'} | 
                            Severity: {event.severity || 'Unknown'}
                            {event.blocked && ' | BLOCKED'}
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="text-xs text-slate-500 text-center py-2">
                        No recent PII events
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Rule Engine Statistics */}
      {ruleEngineStats && (
        <div className="bg-white border border-slate-200 rounded-xl p-6">
          <h2 className="text-xl font-bold text-slate-900 mb-6">Rule Engine Performance</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">{ruleEngineStats.rules_executed.toLocaleString()}</div>
              <div className="text-sm text-slate-600">Rules Executed</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">{ruleEngineStats.rules_matched.toLocaleString()}</div>
              <div className="text-sm text-slate-600">Rules Matched</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-600">{ruleEngineStats.average_exec_time}</div>
              <div className="text-sm text-slate-600">Avg Execution Time</div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="font-semibold text-slate-900 mb-3">Rule Types Distribution</h3>
              <div className="space-y-2">
                {Object.entries(ruleEngineStats.rule_type_stats).map(([type, stats]: [string, any]) => (
                  <div key={type} className="flex items-center justify-between py-2 border-b border-slate-100">
                    <span className="capitalize text-slate-700">{type}</span>
                    <div className="text-right">
                      <div className="font-medium">{stats.count} rules</div>
                      <div className="text-xs text-slate-500">{stats.executions} executions</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="font-semibold text-slate-900 mb-3">Engine Health</h3>
              <div className="space-y-2">
                <div className="flex justify-between py-2 border-b border-slate-100">
                  <span className="text-slate-700">Last Reload:</span>
                  <span className="font-medium text-sm">{new Date(ruleEngineStats.last_reload).toLocaleString()}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-slate-100">
                  <span className="text-slate-700">Reload Count:</span>
                  <span className="font-medium">{ruleEngineStats.reload_count}</span>
                </div>
                <div className="flex justify-between py-2">
                  <span className="text-slate-700">Error Count:</span>
                  <span className={`font-medium ${ruleEngineStats.error_count > 0 ? 'text-red-600' : 'text-green-600'}`}>
                    {ruleEngineStats.error_count}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Recent Moderation Events */}
      <div className="bg-white border border-slate-200 rounded-xl p-6">
        <h2 className="text-xl font-bold text-slate-900 mb-6">
          Recent Moderation Events 
          <span className="text-sm font-normal text-slate-500 ml-2">
            ({recentModerations.length} events)
          </span>
        </h2>
        
        {recentModerations.length > 0 ? (
          <div className="space-y-3 max-h-96 overflow-y-auto">
            {recentModerations.map((event, index) => (
              <div key={index} className="border border-slate-200 rounded-lg p-4 hover:bg-slate-50 transition-colors">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-3 mb-2">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(event.action)}`}>
                        {event.action.toUpperCase()}
                      </span>
                      <span className={`px-2 py-1 rounded border text-xs font-medium ${getSeverityColor(event.final_score)}`}>
                        Score: {event.final_score.toFixed(3)}
                      </span>
                      <span className="text-xs text-slate-500">
                        {new Date(event.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                    <div className="text-sm text-slate-700 mb-2">
                      <strong>Content:</strong> {event.content?.substring(0, 100)}
                      {event.content?.length > 100 && '...'}
                    </div>
                    <div className="text-xs text-slate-500">
                      <strong>Category:</strong> {event.primary_category || 'Unknown'} | 
                      <strong> Rules Triggered:</strong> {event.triggered_rules?.length || 0} |
                      <strong> User:</strong> {event.context?.user_id || 'Anonymous'}
                    </div>
                  </div>
                  <div className="text-xs text-slate-400">
                    {event.process_time}
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-8 text-slate-500">
            <span className="text-4xl mb-2 block">📝</span>
            No recent moderation events
            <div className="text-sm mt-2">Events will appear here as content is moderated</div>
          </div>
        )}
      </div>
    </div>
  )
}

export default AdvancedDashboard