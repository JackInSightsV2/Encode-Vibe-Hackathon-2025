import React, { useState, useEffect } from 'react'
import { 
  ClockIcon,
  UserIcon,
  DocumentTextIcon,
  FunnelIcon,
  MagnifyingGlassIcon,
  ArrowDownTrayIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  TagIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  XCircleIcon,
  InformationCircleIcon
} from '@heroicons/react/24/outline'

interface AuditEntry {
  id: string
  timestamp: string
  user: string
  user_id: string
  action: 'create' | 'update' | 'delete' | 'restore' | 'promote' | 'rollback'
  resource_type: 'configuration' | 'environment' | 'backup' | 'promotion'
  resource_id: string
  resource_name: string
  environment?: string
  changes: ChangeDetail[]
  metadata: {
    ip_address?: string
    user_agent?: string
    session_id?: string
    request_id?: string
    duration_ms?: number
  }
  risk_level: 'low' | 'medium' | 'high' | 'critical'
  status: 'success' | 'failed' | 'partial'
  error_message?: string
}

interface ChangeDetail {
  field: string
  old_value: any
  new_value: any
  change_type: 'added' | 'modified' | 'deleted'
}

interface AuditFilter {
  dateRange: { start: Date | null; end: Date | null }
  users: string[]
  actions: string[]
  resources: string[]
  environments: string[]
  riskLevels: string[]
  searchQuery: string
}

interface AuditStats {
  total_changes: number
  changes_by_user: Record<string, number>
  changes_by_action: Record<string, number>
  changes_by_risk: Record<string, number>
  recent_high_risk: number
  failed_attempts: number
}

interface ConfigAuditProps {
  onClose: () => void
}

const ConfigAudit: React.FC<ConfigAuditProps> = ({ onClose }) => {
  const [auditEntries, setAuditEntries] = useState<AuditEntry[]>([])
  const [filteredEntries, setFilteredEntries] = useState<AuditEntry[]>([])
  const [stats, setStats] = useState<AuditStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedEntry, setSelectedEntry] = useState<AuditEntry | null>(null)
  const [expandedEntries, setExpandedEntries] = useState<Set<string>>(new Set())
  const [filter, setFilter] = useState<AuditFilter>({
    dateRange: { 
      start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // Last 7 days
      end: new Date() 
    },
    users: [],
    actions: [],
    resources: [],
    environments: [],
    riskLevels: [],
    searchQuery: ''
  })
  const [showFilters, setShowFilters] = useState(false)

  useEffect(() => {
    fetchAuditData()
  }, [])

  useEffect(() => {
    applyFilters()
  }, [auditEntries, filter])

  const fetchAuditData = async () => {
    try {
      const [auditRes, statsRes] = await Promise.all([
        fetch('/api/config/audit'),
        fetch('/api/config/audit/stats')
      ])

      if (auditRes.ok && statsRes.ok) {
        const auditData = await auditRes.json()
        const statsData = await statsRes.json()
        
        setAuditEntries(auditData.data || [])
        setStats(statsData.data)
      }
    } catch (error) {
      console.error('Failed to fetch audit data:', error)
    } finally {
      setLoading(false)
    }
  }

  const applyFilters = () => {
    let filtered = [...auditEntries]

    // Date range filter
    if (filter.dateRange.start) {
      filtered = filtered.filter(entry => 
        new Date(entry.timestamp) >= filter.dateRange.start!
      )
    }
    if (filter.dateRange.end) {
      filtered = filtered.filter(entry => 
        new Date(entry.timestamp) <= filter.dateRange.end!
      )
    }

    // Other filters
    if (filter.users.length > 0) {
      filtered = filtered.filter(entry => filter.users.includes(entry.user))
    }
    if (filter.actions.length > 0) {
      filtered = filtered.filter(entry => filter.actions.includes(entry.action))
    }
    if (filter.resources.length > 0) {
      filtered = filtered.filter(entry => filter.resources.includes(entry.resource_type))
    }
    if (filter.environments.length > 0) {
      filtered = filtered.filter(entry => entry.environment && filter.environments.includes(entry.environment))
    }
    if (filter.riskLevels.length > 0) {
      filtered = filtered.filter(entry => filter.riskLevels.includes(entry.risk_level))
    }

    // Search query
    if (filter.searchQuery) {
      const query = filter.searchQuery.toLowerCase()
      filtered = filtered.filter(entry => 
        entry.user.toLowerCase().includes(query) ||
        entry.resource_name.toLowerCase().includes(query) ||
        entry.changes.some(change => 
          change.field.toLowerCase().includes(query) ||
          JSON.stringify(change.old_value).toLowerCase().includes(query) ||
          JSON.stringify(change.new_value).toLowerCase().includes(query)
        )
      )
    }

    setFilteredEntries(filtered)
  }

  const exportAuditLog = async () => {
    try {
      const response = await fetch('/api/config/audit/export', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          filters: filter,
          format: 'csv'
        })
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `audit-log-${new Date().toISOString().split('T')[0]}.csv`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      }
    } catch (error) {
      console.error('Failed to export audit log:', error)
    }
  }

  const toggleExpanded = (entryId: string) => {
    const newExpanded = new Set(expandedEntries)
    if (newExpanded.has(entryId)) {
      newExpanded.delete(entryId)
    } else {
      newExpanded.add(entryId)
    }
    setExpandedEntries(newExpanded)
  }

  const getActionIcon = (action: string) => {
    switch (action) {
      case 'create': return <CheckCircleIcon className="h-5 w-5 text-green-500" />
      case 'update': return <DocumentTextIcon className="h-5 w-5 text-blue-500" />
      case 'delete': return <XCircleIcon className="h-5 w-5 text-red-500" />
      case 'restore': return <ArrowDownTrayIcon className="h-5 w-5 text-purple-500" />
      case 'promote': return <ChevronRightIcon className="h-5 w-5 text-indigo-500" />
      case 'rollback': return <ExclamationTriangleIcon className="h-5 w-5 text-orange-500" />
      default: return <InformationCircleIcon className="h-5 w-5 text-gray-500" />
    }
  }

  const getRiskLevelColor = (level: string) => {
    switch (level) {
      case 'low': return 'text-green-600 bg-green-50'
      case 'medium': return 'text-yellow-600 bg-yellow-50'
      case 'high': return 'text-orange-600 bg-orange-50'
      case 'critical': return 'text-red-600 bg-red-50'
      default: return 'text-gray-600 bg-gray-50'
    }
  }

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins} minutes ago`
    if (diffHours < 24) return `${diffHours} hours ago`
    if (diffDays < 7) return `${diffDays} days ago`
    
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading audit log...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              Configuration Audit Log
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Track all configuration changes and access history
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`px-4 py-2 text-sm font-medium border rounded-md transition-colors ${
                showFilters 
                  ? 'text-blue-600 bg-blue-50 border-blue-300' 
                  : 'text-gray-700 bg-white border-gray-300 hover:bg-gray-50'
              }`}
            >
              <FunnelIcon className="h-4 w-4 inline-block mr-2" />
              Filters
              {Object.values(filter).some(v => 
                (Array.isArray(v) && v.length > 0) || 
                (typeof v === 'string' && v.length > 0)
              ) && (
                <span className="ml-2 px-2 py-0.5 text-xs bg-blue-600 text-white rounded-full">
                  Active
                </span>
              )}
            </button>
            <button
              onClick={exportAuditLog}
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

      {/* Stats Summary */}
      {stats && (
        <div className="bg-gray-50 border-b px-6 py-4">
          <div className="grid grid-cols-2 md:grid-cols-6 gap-4 text-center">
            <div>
              <p className="text-2xl font-bold text-gray-900">{stats.total_changes}</p>
              <p className="text-xs text-gray-600">Total Changes</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-blue-600">
                {Object.keys(stats.changes_by_user).length}
              </p>
              <p className="text-xs text-gray-600">Active Users</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-orange-600">{stats.recent_high_risk}</p>
              <p className="text-xs text-gray-600">High Risk (24h)</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-red-600">{stats.failed_attempts}</p>
              <p className="text-xs text-gray-600">Failed Attempts</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-green-600">
                {stats.changes_by_action.update || 0}
              </p>
              <p className="text-xs text-gray-600">Updates</p>
            </div>
            <div>
              <p className="text-2xl font-bold text-purple-600">
                {stats.changes_by_action.promote || 0}
              </p>
              <p className="text-xs text-gray-600">Promotions</p>
            </div>
          </div>
        </div>
      )}

      {/* Filters Panel */}
      {showFilters && (
        <FilterPanel
          filter={filter}
          setFilter={setFilter}
          availableUsers={[...new Set(auditEntries.map(e => e.user))]}
          availableEnvironments={[...new Set(auditEntries.map(e => e.environment).filter(Boolean))] as string[]}
        />
      )}

      {/* Audit Entries */}
      <div className="flex-1 overflow-y-auto">
        {filteredEntries.length === 0 ? (
          <div className="text-center py-12">
            <ClockIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">No audit entries found</p>
            {filter.searchQuery && (
              <p className="text-sm text-gray-400 mt-2">
                Try adjusting your search or filters
              </p>
            )}
          </div>
        ) : (
          <div className="divide-y">
            {filteredEntries.map((entry) => (
              <AuditEntryItem
                key={entry.id}
                entry={entry}
                isExpanded={expandedEntries.has(entry.id)}
                onToggle={() => toggleExpanded(entry.id)}
                onSelect={() => setSelectedEntry(entry)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Entry Details Modal */}
      {selectedEntry && (
        <AuditDetailsModal
          entry={selectedEntry}
          onClose={() => setSelectedEntry(null)}
        />
      )}
    </div>
  )
}

// Sub-components
const FilterPanel: React.FC<{
  filter: AuditFilter
  setFilter: (filter: AuditFilter) => void
  availableUsers: string[]
  availableEnvironments: string[]
}> = ({ filter, setFilter, availableUsers, availableEnvironments }) => {
  return (
    <div className="bg-white border-b px-6 py-4">
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4">
        {/* Date Range */}
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Start Date
          </label>
          <input
            type="date"
            value={filter.dateRange.start?.toISOString().split('T')[0] || ''}
            onChange={(e) => setFilter({
              ...filter,
              dateRange: {
                ...filter.dateRange,
                start: e.target.value ? new Date(e.target.value) : null
              }
            })}
            className="w-full px-3 py-1 text-sm border border-gray-300 rounded-md"
          />
        </div>

        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            End Date
          </label>
          <input
            type="date"
            value={filter.dateRange.end?.toISOString().split('T')[0] || ''}
            onChange={(e) => setFilter({
              ...filter,
              dateRange: {
                ...filter.dateRange,
                end: e.target.value ? new Date(e.target.value) : null
              }
            })}
            className="w-full px-3 py-1 text-sm border border-gray-300 rounded-md"
          />
        </div>

        {/* Users */}
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Users
          </label>
          <select
            multiple
            value={filter.users}
            onChange={(e) => setFilter({
              ...filter,
              users: Array.from(e.target.selectedOptions, option => option.value)
            })}
            className="w-full px-3 py-1 text-sm border border-gray-300 rounded-md"
            size={3}
          >
            {availableUsers.map(user => (
              <option key={user} value={user}>{user}</option>
            ))}
          </select>
        </div>

        {/* Actions */}
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Actions
          </label>
          <select
            multiple
            value={filter.actions}
            onChange={(e) => setFilter({
              ...filter,
              actions: Array.from(e.target.selectedOptions, option => option.value)
            })}
            className="w-full px-3 py-1 text-sm border border-gray-300 rounded-md"
            size={3}
          >
            <option value="create">Create</option>
            <option value="update">Update</option>
            <option value="delete">Delete</option>
            <option value="restore">Restore</option>
            <option value="promote">Promote</option>
            <option value="rollback">Rollback</option>
          </select>
        </div>

        {/* Risk Levels */}
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Risk Level
          </label>
          <select
            multiple
            value={filter.riskLevels}
            onChange={(e) => setFilter({
              ...filter,
              riskLevels: Array.from(e.target.selectedOptions, option => option.value)
            })}
            className="w-full px-3 py-1 text-sm border border-gray-300 rounded-md"
            size={3}
          >
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="critical">Critical</option>
          </select>
        </div>

        {/* Search */}
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1">
            Search
          </label>
          <div className="relative">
            <input
              type="text"
              value={filter.searchQuery}
              onChange={(e) => setFilter({ ...filter, searchQuery: e.target.value })}
              placeholder="Search..."
              className="w-full pl-8 pr-3 py-1 text-sm border border-gray-300 rounded-md"
            />
            <MagnifyingGlassIcon className="h-4 w-4 absolute left-2 top-1/2 transform -translate-y-1/2 text-gray-400" />
          </div>
        </div>
      </div>

      <div className="mt-3 flex justify-end">
        <button
          onClick={() => setFilter({
            dateRange: { start: null, end: null },
            users: [],
            actions: [],
            resources: [],
            environments: [],
            riskLevels: [],
            searchQuery: ''
          })}
          className="text-sm text-blue-600 hover:text-blue-800"
        >
          Clear Filters
        </button>
      </div>
    </div>
  )
}

const AuditEntryItem: React.FC<{
  entry: AuditEntry
  isExpanded: boolean
  onToggle: () => void
  onSelect: () => void
}> = ({ entry, isExpanded, onToggle, onSelect }) => {
  return (
    <div className="hover:bg-gray-50">
      <div className="px-6 py-4">
        <div className="flex items-start">
          {/* Expand/Collapse */}
          <button
            onClick={onToggle}
            className="mr-3 mt-0.5 text-gray-400 hover:text-gray-600"
          >
            {isExpanded ? (
              <ChevronDownIcon className="h-5 w-5" />
            ) : (
              <ChevronRightIcon className="h-5 w-5" />
            )}
          </button>

          {/* Action Icon */}
          <div className="mr-3">
            {getActionIcon(entry.action)}
          </div>

          {/* Content */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <span className="text-sm font-medium text-gray-900">
                  {entry.user}
                </span>
                <span className="text-sm text-gray-600">
                  {entry.action} {entry.resource_type}
                </span>
                <span className="text-sm text-gray-500">
                  "{entry.resource_name}"
                </span>
                {entry.environment && (
                  <span className="px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-700 rounded-full">
                    {entry.environment}
                  </span>
                )}
                <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${getRiskLevelColor(entry.risk_level)}`}>
                  {entry.risk_level}
                </span>
                {entry.status !== 'success' && (
                  <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                    entry.status === 'failed' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'
                  }`}>
                    {entry.status}
                  </span>
                )}
              </div>
              <div className="flex items-center space-x-3">
                <span className="text-sm text-gray-500">
                  {formatTimestamp(entry.timestamp)}
                </span>
                <button
                  onClick={onSelect}
                  className="text-sm text-blue-600 hover:text-blue-800"
                >
                  Details
                </button>
              </div>
            </div>

            {/* Error Message */}
            {entry.error_message && (
              <p className="mt-1 text-sm text-red-600">
                Error: {entry.error_message}
              </p>
            )}

            {/* Expanded Changes */}
            {isExpanded && (
              <div className="mt-3 pl-8 space-y-2">
                {entry.changes.slice(0, 5).map((change, index) => (
                  <ChangeDetailItem key={index} change={change} />
                ))}
                {entry.changes.length > 5 && (
                  <p className="text-sm text-gray-500">
                    ...and {entry.changes.length - 5} more changes
                  </p>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

const ChangeDetailItem: React.FC<{ change: ChangeDetail }> = ({ change }) => {
  const getChangeIcon = () => {
    switch (change.change_type) {
      case 'added': return '+'
      case 'modified': return '~'
      case 'deleted': return '-'
    }
  }

  const getChangeColor = () => {
    switch (change.change_type) {
      case 'added': return 'text-green-600'
      case 'modified': return 'text-yellow-600'
      case 'deleted': return 'text-red-600'
    }
  }

  return (
    <div className="flex items-start space-x-2 text-sm">
      <span className={`font-mono font-bold ${getChangeColor()}`}>
        {getChangeIcon()}
      </span>
      <div className="flex-1">
        <span className="font-mono text-gray-700">{change.field}</span>
        {change.change_type === 'modified' && (
          <div className="mt-1 grid grid-cols-2 gap-2 text-xs">
            <div className="bg-red-50 p-2 rounded">
              <span className="text-gray-500">From: </span>
              <code className="text-red-600">{JSON.stringify(change.old_value)}</code>
            </div>
            <div className="bg-green-50 p-2 rounded">
              <span className="text-gray-500">To: </span>
              <code className="text-green-600">{JSON.stringify(change.new_value)}</code>
            </div>
          </div>
        )}
        {change.change_type === 'added' && (
          <div className="mt-1 bg-green-50 p-2 rounded text-xs">
            <code className="text-green-600">{JSON.stringify(change.new_value)}</code>
          </div>
        )}
        {change.change_type === 'deleted' && (
          <div className="mt-1 bg-red-50 p-2 rounded text-xs">
            <code className="text-red-600">{JSON.stringify(change.old_value)}</code>
          </div>
        )}
      </div>
    </div>
  )
}

const AuditDetailsModal: React.FC<{
  entry: AuditEntry
  onClose: () => void
}> = ({ entry, onClose }) => {
  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-medium text-gray-900">
              Audit Entry Details
            </h3>
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

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-6">
            {/* Basic Info */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-500">User</p>
                <p className="mt-1 text-sm font-medium">{entry.user}</p>
                <p className="text-xs text-gray-400">ID: {entry.user_id}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Timestamp</p>
                <p className="mt-1 text-sm font-medium">
                  {new Date(entry.timestamp).toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Action</p>
                <p className="mt-1 text-sm font-medium capitalize">{entry.action}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Resource</p>
                <p className="mt-1 text-sm font-medium">{entry.resource_type}</p>
                <p className="text-xs text-gray-400">{entry.resource_name}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Risk Level</p>
                <p className={`mt-1 text-sm font-medium px-2 py-1 rounded-full inline-block ${
                  getRiskLevelColor(entry.risk_level)
                }`}>
                  {entry.risk_level}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Status</p>
                <p className={`mt-1 text-sm font-medium ${
                  entry.status === 'success' ? 'text-green-600' :
                  entry.status === 'failed' ? 'text-red-600' :
                  'text-yellow-600'
                }`}>
                  {entry.status}
                </p>
              </div>
            </div>

            {/* Metadata */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">Session Information</h4>
              <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
                {entry.metadata.ip_address && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">IP Address:</span>
                    <span className="font-mono">{entry.metadata.ip_address}</span>
                  </div>
                )}
                {entry.metadata.session_id && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Session ID:</span>
                    <span className="font-mono text-xs">{entry.metadata.session_id}</span>
                  </div>
                )}
                {entry.metadata.request_id && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Request ID:</span>
                    <span className="font-mono text-xs">{entry.metadata.request_id}</span>
                  </div>
                )}
                {entry.metadata.duration_ms && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Duration:</span>
                    <span>{entry.metadata.duration_ms}ms</span>
                  </div>
                )}
              </div>
            </div>

            {/* Changes */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-3">
                Changes ({entry.changes.length})
              </h4>
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {entry.changes.map((change, index) => (
                  <div key={index} className="bg-gray-50 rounded-lg p-4">
                    <ChangeDetailItem change={change} />
                  </div>
                ))}
              </div>
            </div>

            {/* Error Details */}
            {entry.error_message && (
              <div className="bg-red-50 rounded-lg p-4">
                <h4 className="text-sm font-medium text-red-900 mb-2">Error Details</h4>
                <p className="text-sm text-red-700">{entry.error_message}</p>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t bg-gray-50">
          <div className="flex justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// Helper function
function getActionIcon(action: string) {
  switch (action) {
    case 'create': return <CheckCircleIcon className="h-5 w-5 text-green-500" />
    case 'update': return <DocumentTextIcon className="h-5 w-5 text-blue-500" />
    case 'delete': return <XCircleIcon className="h-5 w-5 text-red-500" />
    case 'restore': return <ArrowDownTrayIcon className="h-5 w-5 text-purple-500" />
    case 'promote': return <ChevronRightIcon className="h-5 w-5 text-indigo-500" />
    case 'rollback': return <ExclamationTriangleIcon className="h-5 w-5 text-orange-500" />
    default: return <InformationCircleIcon className="h-5 w-5 text-gray-500" />
  }
}

function getRiskLevelColor(level: string) {
  switch (level) {
    case 'low': return 'text-green-600 bg-green-50'
    case 'medium': return 'text-yellow-600 bg-yellow-50'
    case 'high': return 'text-orange-600 bg-orange-50'
    case 'critical': return 'text-red-600 bg-red-50'
    default: return 'text-gray-600 bg-gray-50'
  }
}

function formatTimestamp(timestamp: string) {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} minutes ago`
  if (diffHours < 24) return `${diffHours} hours ago`
  if (diffDays < 7) return `${diffDays} days ago`
  
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
}

export default ConfigAudit