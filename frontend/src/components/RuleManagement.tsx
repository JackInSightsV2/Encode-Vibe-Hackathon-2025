import React, { useState, useEffect } from 'react'
import { useWebSocketMessage } from '../contexts/WebSocketContext'
import { MessageTypes } from '../services/websocket'

interface Rule {
  id: string
  name: string
  description: string
  type: 'regex' | 'keyword' | 'ml' | 'composite'
  pattern?: string
  keywords?: string[]
  weight: number
  priority: number
  action: 'block' | 'flag' | 'warn'
  category: string
  enabled: boolean
  created_at: string
  updated_at: string
  context?: {
    user_types?: string[]
    content_types?: string[]
    channels?: string[]
  }
  conditions?: {
    min_length?: number
    max_length?: number
    case_sensitive?: boolean
    whole_words?: boolean
  }
}

interface RuleStats {
  rule_id: string
  rule_name: string
  executions: number
  matches: number
  match_rate: number
  avg_exec_time: string
  last_triggered: string
}

const RuleManagement: React.FC = () => {
  const [rules, setRules] = useState<Rule[]>([])
  const [ruleStats, setRuleStats] = useState<RuleStats[]>([])
  const [filter, setFilter] = useState<'all' | 'enabled' | 'disabled'>('all')
  const [sortBy, setSortBy] = useState<'name' | 'priority' | 'weight' | 'created_at'>('priority')
  const [searchTerm, setSearchTerm] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchRules()
    fetchRuleStats()
  }, [])

  // Listen for rule updates
  useWebSocketMessage(MessageTypes.RULE_UPDATED, (data) => {
    if (data?.rule) {
      setRules(prev => prev.map(rule => 
        rule.id === data.rule.id ? data.rule : rule
      ))
    }
  })

  const fetchRules = async () => {
    try {
      setLoading(true)
      setError(null)
      
      const response = await fetch('/api/rules')
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const data = await response.json()
      
      if (data.success && Array.isArray(data.data)) {
        setRules(data.data)
      } else {
        throw new Error(data.error || 'Invalid response format')
      }
    } catch (error) {
      console.error('Error fetching rules:', error)
      setError(error instanceof Error ? error.message : 'Failed to fetch rules')
      setRules([])
    } finally {
      setLoading(false)
    }
  }

  const fetchRuleStats = async () => {
    try {
      const response = await fetch('/api/moderation/rules/stats/detailed')
      if (response.ok) {
        const data = await response.json()
        setRuleStats(data.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch rule stats:', error)
    }
  }

  const toggleRule = async (ruleId: string, enabled: boolean) => {
    try {
      const response = await fetch(`/api/moderation/rules/${ruleId}/${enabled ? 'enable' : 'disable'}`, {
        method: 'POST'
      })
      if (response.ok) {
        setRules(prev => prev.map(rule => 
          rule.id === ruleId ? { ...rule, enabled } : rule
        ))
      }
    } catch (error) {
      console.error('Failed to toggle rule:', error)
    }
  }

  const deleteRule = async (ruleId: string) => {
    if (!confirm('Are you sure you want to delete this rule?')) return
    
    try {
      const response = await fetch(`/api/moderation/rules/${ruleId}`, {
        method: 'DELETE'
      })
      if (response.ok) {
        setRules(prev => prev.filter(rule => rule.id !== ruleId))
      }
    } catch (error) {
      console.error('Failed to delete rule:', error)
    }
  }

  const reloadRules = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/moderation/rules/reload', {
        method: 'POST'
      })
      if (response.ok) {
        await fetchRules()
        alert('Rules reloaded successfully!')
      }
    } catch (error) {
      console.error('Failed to reload rules:', error)
      alert('Failed to reload rules')
    } finally {
      setLoading(false)
    }
  }

  const filteredRules = rules
    .filter(rule => {
      if (filter === 'enabled') return rule.enabled
      if (filter === 'disabled') return !rule.enabled
      return true
    })
    .filter(rule => 
      rule.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      rule.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      rule.category.toLowerCase().includes(searchTerm.toLowerCase())
    )
    .sort((a, b) => {
      switch (sortBy) {
        case 'name': return a.name.localeCompare(b.name)
        case 'priority': return a.priority - b.priority
        case 'weight': return b.weight - a.weight
        case 'created_at': return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        default: return 0
      }
    })

  const getRuleTypeColor = (type: string) => {
    switch (type) {
      case 'regex': return 'bg-blue-100 text-blue-800'
      case 'keyword': return 'bg-green-100 text-green-800'
      case 'ml': return 'bg-purple-100 text-purple-800'
      case 'composite': return 'bg-orange-100 text-orange-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  const getActionColor = (action: string) => {
    switch (action) {
      case 'block': return 'bg-red-100 text-red-800'
      case 'flag': return 'bg-yellow-100 text-yellow-800'
      case 'warn': return 'bg-orange-100 text-orange-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  const getPriorityColor = (priority: number) => {
    if (priority <= 2) return 'text-red-600 font-bold'
    if (priority <= 4) return 'text-orange-600 font-semibold'
    if (priority <= 6) return 'text-yellow-600 font-medium'
    return 'text-gray-600'
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-slate-900 mb-2">Rule Management</h1>
            <p className="text-slate-600">Configure and monitor custom moderation rules</p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={reloadRules}
              disabled={loading}
              className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {loading ? 'Reloading...' : '🔄 Reload Rules'}
            </button>
            <button
              className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition-colors opacity-50 cursor-not-allowed"
              title="Create Rule functionality is not implemented yet"
            >
              ➕ Create Rule
            </button>
          </div>
        </div>
      </div>

      {/* Statistics Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white border border-slate-200 rounded-lg p-4">
          <div className="text-2xl font-bold text-blue-600">{rules.length}</div>
          <div className="text-sm text-slate-600">Total Rules</div>
        </div>
        <div className="bg-white border border-slate-200 rounded-lg p-4">
          <div className="text-2xl font-bold text-green-600">{rules.filter(r => r.enabled).length}</div>
          <div className="text-sm text-slate-600">Active Rules</div>
        </div>
        <div className="bg-white border border-slate-200 rounded-lg p-4">
          <div className="text-2xl font-bold text-red-600">{rules.filter(r => !r.enabled).length}</div>
          <div className="text-sm text-slate-600">Disabled Rules</div>
        </div>
        <div className="bg-white border border-slate-200 rounded-lg p-4">
          <div className="text-2xl font-bold text-purple-600">
            {ruleStats.reduce((sum, stat) => sum + stat.matches, 0).toLocaleString()}
          </div>
          <div className="text-sm text-slate-600">Total Matches</div>
        </div>
      </div>

      {/* Filters and Search */}
      <div className="bg-white border border-slate-200 rounded-xl p-6">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between space-y-4 md:space-y-0">
          <div className="flex items-center space-x-4">
            <div>
              <label className="text-sm font-medium text-slate-700 mb-1 block">Filter</label>
              <select
                value={filter}
                onChange={(e) => setFilter(e.target.value as any)}
                className="border border-slate-300 rounded-lg px-3 py-2 text-sm"
              >
                <option value="all">All Rules</option>
                <option value="enabled">Enabled Only</option>
                <option value="disabled">Disabled Only</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium text-slate-700 mb-1 block">Sort By</label>
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
                className="border border-slate-300 rounded-lg px-3 py-2 text-sm"
              >
                <option value="priority">Priority</option>
                <option value="name">Name</option>
                <option value="weight">Weight</option>
                <option value="created_at">Created Date</option>
              </select>
            </div>
          </div>
          <div>
            <label className="text-sm font-medium text-slate-700 mb-1 block">Search</label>
            <input
              type="text"
              placeholder="Search rules..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="border border-slate-300 rounded-lg px-3 py-2 text-sm w-64"
            />
          </div>
        </div>
      </div>

      {/* Rules List */}
      <div className="bg-white border border-slate-200 rounded-xl overflow-hidden">
        <div className="p-6 border-b border-slate-200">
          <h2 className="text-xl font-bold text-slate-900">Rules ({filteredRules.length})</h2>
        </div>
        
        {filteredRules.length > 0 ? (
          <div className="divide-y divide-slate-200">
            {filteredRules.map((rule) => {
              const stats = ruleStats.find(s => s.rule_id === rule.id)
              
              return (
                <div key={rule.id} className="p-6 hover:bg-slate-50 transition-colors">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-3 mb-2">
                        <h3 className="text-lg font-semibold text-slate-900">{rule.name}</h3>
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${getRuleTypeColor(rule.type)}`}>
                          {rule.type.toUpperCase()}
                        </span>
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(rule.action)}`}>
                          {rule.action.toUpperCase()}
                        </span>
                        <span className={`text-sm ${getPriorityColor(rule.priority)}`}>
                          P{rule.priority}
                        </span>
                        <span className="text-sm text-slate-500">
                          Weight: {rule.weight}
                        </span>
                      </div>
                      
                      <p className="text-slate-600 mb-3">{rule.description}</p>
                      
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                        <div>
                          <span className="font-medium text-slate-700">Pattern/Keywords:</span>
                          <div className="text-slate-600 mt-1">
                            {rule.pattern && (
                              <code className="bg-slate-100 px-2 py-1 rounded text-xs">
                                {rule.pattern.length > 50 ? rule.pattern.substring(0, 50) + '...' : rule.pattern}
                              </code>
                            )}
                            {rule.keywords && (
                              <div className="flex flex-wrap gap-1 mt-1">
                                {rule.keywords.slice(0, 3).map((keyword, i) => (
                                  <span key={i} className="bg-blue-100 text-blue-800 px-2 py-1 rounded text-xs">
                                    {keyword}
                                  </span>
                                ))}
                                {rule.keywords.length > 3 && (
                                  <span className="text-slate-500 text-xs">+{rule.keywords.length - 3} more</span>
                                )}
                              </div>
                            )}
                          </div>
                        </div>
                        
                        <div>
                          <span className="font-medium text-slate-700">Category:</span>
                          <div className="text-slate-600 mt-1">{rule.category}</div>
                        </div>
                        
                        <div>
                          <span className="font-medium text-slate-700">Performance:</span>
                          <div className="text-slate-600 mt-1">
                            {stats ? (
                              <>
                                <div>{stats.executions} executions</div>
                                <div>{stats.matches} matches ({(stats.match_rate * 100).toFixed(1)}%)</div>
                                <div>Avg: {stats.avg_exec_time}</div>
                              </>
                            ) : (
                              <div className="text-slate-400">No stats available</div>
                            )}
                          </div>
                        </div>
                      </div>
                      
                      {(rule.context || rule.conditions) && (
                        <div className="mt-3 text-xs text-slate-500">
                          {rule.context && (
                            <div>
                              Context: {Object.entries(rule.context).map(([key, value]) => 
                                `${key}: ${Array.isArray(value) ? value.join(', ') : value}`
                              ).join(' | ')}
                            </div>
                          )}
                          {rule.conditions && (
                            <div>
                              Conditions: {Object.entries(rule.conditions).map(([key, value]) => 
                                `${key}: ${value}`
                              ).join(' | ')}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                    
                    <div className="flex items-center space-x-2 ml-4">
                      <div className="text-right text-xs text-slate-500 mr-4">
                        <div>Created: {new Date(rule.created_at).toLocaleDateString()}</div>
                        <div>Updated: {new Date(rule.updated_at).toLocaleDateString()}</div>
                      </div>
                      
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={rule.enabled}
                          onChange={(e) => toggleRule(rule.id, e.target.checked)}
                          className="rounded"
                        />
                        <span className="ml-2 text-sm">Enabled</span>
                      </label>
                      
                      <button
                        className="text-blue-600 hover:text-blue-800 opacity-50 cursor-not-allowed"
                        title="Edit functionality is not implemented yet"
                      >
                        Edit
                      </button>
                      
                      <button
                        onClick={() => deleteRule(rule.id)}
                        className="bg-red-100 text-red-800 px-3 py-1 rounded text-sm hover:bg-red-200 transition-colors"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          <div className="text-center py-12 text-slate-500">
            <span className="text-4xl mb-2 block">📝</span>
            No rules found matching your criteria
          </div>
        )}
      </div>
    </div>
  )
}

export default RuleManagement