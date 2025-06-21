import React, { useState, useEffect } from 'react'
import { 
  DocumentDuplicateIcon,
  CheckCircleIcon,
  XCircleIcon,
  ExclamationTriangleIcon,
  FunnelIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ArrowDownTrayIcon,
  MagnifyingGlassIcon
} from '@heroicons/react/24/outline'

interface ComparisonSource {
  type: 'environment' | 'version' | 'backup' | 'file'
  id: string
  name: string
  config?: any
  timestamp?: string
}

interface DiffResult {
  path: string
  type: 'added' | 'modified' | 'deleted' | 'unchanged'
  left_value: any
  right_value: any
  level: number
  parent?: string
  children?: string[]
}

interface DiffSummary {
  total_differences: number
  additions: number
  modifications: number
  deletions: number
  unchanged: number
  conflict_count: number
  categories: Record<string, number>
}

interface ConfigComparisonProps {
  environments?: Array<{ id: string; name: string; config: any }>
  versions?: Array<{ id: string; version: string; config: any; created_at: string }>
  onClose: () => void
}

const ConfigComparison: React.FC<ConfigComparisonProps> = ({
  environments = [],
  versions = [],
  onClose
}) => {
  const [leftSource, setLeftSource] = useState<ComparisonSource | null>(null)
  const [rightSource, setRightSource] = useState<ComparisonSource | null>(null)
  const [diffResults, setDiffResults] = useState<DiffResult[]>([])
  const [diffSummary, setDiffSummary] = useState<DiffSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [viewMode, setViewMode] = useState<'unified' | 'split' | 'inline'>('split')
  const [filterType, setFilterType] = useState<'all' | 'added' | 'modified' | 'deleted'>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())
  const [showOnlyDifferences, setShowOnlyDifferences] = useState(true)

  useEffect(() => {
    if (leftSource?.config && rightSource?.config) {
      performComparison()
    }
  }, [leftSource, rightSource])

  const performComparison = async () => {
    if (!leftSource?.config || !rightSource?.config) return

    setLoading(true)
    try {
      const response = await fetch('/api/config/compare', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          left: leftSource.config,
          right: rightSource.config,
          options: {
            ignore_order: true,
            case_sensitive: false,
            deep_compare: true
          }
        })
      })

      if (response.ok) {
        const data = await response.json()
        setDiffResults(data.data.differences || [])
        setDiffSummary(data.data.summary)
        
        // Auto-expand first level
        const firstLevel = new Set(
          data.data.differences
            .filter((d: DiffResult) => d.level === 0)
            .map((d: DiffResult) => d.path)
        )
        setExpandedPaths(firstLevel as Set<string>)
      }
    } catch (error) {
      console.error('Failed to compare configurations:', error)
    } finally {
      setLoading(false)
    }
  }

  const toggleExpanded = (path: string) => {
    const newExpanded = new Set(expandedPaths)
    if (newExpanded.has(path)) {
      newExpanded.delete(path)
    } else {
      newExpanded.add(path)
    }
    setExpandedPaths(newExpanded)
  }

  const exportComparison = async () => {
    try {
      const response = await fetch('/api/config/compare/export', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          left_source: leftSource,
          right_source: rightSource,
          differences: diffResults,
          summary: diffSummary,
          format: 'html'
        })
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `config-comparison-${Date.now()}.html`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      }
    } catch (error) {
      console.error('Failed to export comparison:', error)
    }
  }

  const applyChanges = async (direction: 'left-to-right' | 'right-to-left') => {
    const source = direction === 'left-to-right' ? leftSource : rightSource
    const target = direction === 'left-to-right' ? rightSource : leftSource

    if (!source || !target) return

    if (!window.confirm(`Apply changes from ${source.name} to ${target.name}?`)) {
      return
    }

    // TODO: Implement selective change application
    console.log('Applying changes...', direction)
  }

  const getFilteredDiffs = () => {
    let filtered = [...diffResults]

    if (filterType !== 'all') {
      filtered = filtered.filter(d => d.type === filterType)
    }

    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(d => 
        d.path.toLowerCase().includes(query) ||
        JSON.stringify(d.left_value).toLowerCase().includes(query) ||
        JSON.stringify(d.right_value).toLowerCase().includes(query)
      )
    }

    if (showOnlyDifferences) {
      filtered = filtered.filter(d => d.type !== 'unchanged')
    }

    return filtered
  }

  const renderValue = (value: any, type: 'added' | 'modified' | 'deleted' | 'unchanged') => {
    const colorClass = 
      type === 'added' ? 'bg-green-50 text-green-700' :
      type === 'deleted' ? 'bg-red-50 text-red-700' :
      type === 'modified' ? 'bg-yellow-50 text-yellow-700' :
      'bg-gray-50 text-gray-700'

    if (value === null || value === undefined) {
      return <span className="text-gray-400 italic">null</span>
    }

    if (typeof value === 'object') {
      return (
        <pre className={`text-xs p-2 rounded ${colorClass} overflow-x-auto`}>
          {JSON.stringify(value, null, 2)}
        </pre>
      )
    }

    return (
      <span className={`px-2 py-1 rounded text-sm font-mono ${colorClass}`}>
        {String(value)}
      </span>
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              Configuration Comparison
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Compare configurations across environments, versions, or files
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={exportComparison}
              disabled={!diffResults.length}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
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

      {/* Source Selection */}
      <div className="bg-gray-50 border-b px-6 py-4">
        <div className="grid grid-cols-2 gap-6">
          <SourceSelector
            label="Compare From"
            source={leftSource}
            onSelect={setLeftSource}
            environments={environments}
            versions={versions}
            disabled={loading}
          />
          <SourceSelector
            label="Compare To"
            source={rightSource}
            onSelect={setRightSource}
            environments={environments}
            versions={versions}
            disabled={loading}
          />
        </div>
      </div>

      {/* Controls */}
      {diffSummary && (
        <>
          <div className="bg-white border-b px-6 py-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-6 text-sm">
                <span className="text-gray-600">
                  Total: <span className="font-medium">{diffSummary.total_differences}</span>
                </span>
                <span className="text-green-600">
                  Added: <span className="font-medium">+{diffSummary.additions}</span>
                </span>
                <span className="text-yellow-600">
                  Modified: <span className="font-medium">~{diffSummary.modifications}</span>
                </span>
                <span className="text-red-600">
                  Deleted: <span className="font-medium">-{diffSummary.deletions}</span>
                </span>
                {diffSummary.conflict_count > 0 && (
                  <span className="text-orange-600">
                    Conflicts: <span className="font-medium">{diffSummary.conflict_count}</span>
                  </span>
                )}
              </div>
              
              <div className="flex items-center space-x-4">
                {/* View Mode */}
                <div className="flex items-center space-x-2">
                  <span className="text-sm text-gray-500">View:</span>
                  <select
                    value={viewMode}
                    onChange={(e) => setViewMode(e.target.value as any)}
                    className="text-sm border border-gray-300 rounded px-2 py-1"
                  >
                    <option value="split">Split</option>
                    <option value="unified">Unified</option>
                    <option value="inline">Inline</option>
                  </select>
                </div>

                {/* Filter */}
                <div className="flex items-center space-x-2">
                  <FunnelIcon className="h-4 w-4 text-gray-400" />
                  <select
                    value={filterType}
                    onChange={(e) => setFilterType(e.target.value as any)}
                    className="text-sm border border-gray-300 rounded px-2 py-1"
                  >
                    <option value="all">All Changes</option>
                    <option value="added">Added</option>
                    <option value="modified">Modified</option>
                    <option value="deleted">Deleted</option>
                  </select>
                </div>

                {/* Search */}
                <div className="relative">
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Search..."
                    className="pl-8 pr-3 py-1 text-sm border border-gray-300 rounded"
                  />
                  <MagnifyingGlassIcon className="h-4 w-4 absolute left-2 top-1/2 transform -translate-y-1/2 text-gray-400" />
                </div>

                {/* Toggle */}
                <label className="flex items-center text-sm">
                  <input
                    type="checkbox"
                    checked={showOnlyDifferences}
                    onChange={(e) => setShowOnlyDifferences(e.target.checked)}
                    className="h-4 w-4 text-blue-600 rounded border-gray-300"
                  />
                  <span className="ml-2 text-gray-700">Only differences</span>
                </label>
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="bg-blue-50 border-b px-6 py-3">
            <div className="flex items-center justify-between">
              <p className="text-sm text-blue-700">
                Review the differences below and apply changes if needed
              </p>
              <div className="flex items-center space-x-3">
                <button
                  onClick={() => applyChanges('left-to-right')}
                  className="px-3 py-1 text-sm font-medium text-white bg-blue-600 rounded hover:bg-blue-700"
                >
                  Apply {leftSource?.name} → {rightSource?.name}
                </button>
                <button
                  onClick={() => applyChanges('right-to-left')}
                  className="px-3 py-1 text-sm font-medium text-blue-600 bg-white border border-blue-600 rounded hover:bg-blue-50"
                >
                  Apply {rightSource?.name} → {leftSource?.name}
                </button>
              </div>
            </div>
          </div>
        </>
      )}

      {/* Diff View */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
              <p className="mt-4 text-gray-600">Comparing configurations...</p>
            </div>
          </div>
        ) : !leftSource || !rightSource ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center text-gray-500">
              <DocumentDuplicateIcon className="h-12 w-12 mx-auto mb-4 text-gray-300" />
              <p>Select two sources to compare</p>
            </div>
          </div>
        ) : diffResults.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <CheckCircleIcon className="h-12 w-12 mx-auto mb-4 text-green-400" />
              <p className="text-gray-600">Configurations are identical</p>
            </div>
          </div>
        ) : (
          <div className="p-6">
            {viewMode === 'split' && (
              <SplitDiffView
                diffs={getFilteredDiffs()}
                expandedPaths={expandedPaths}
                onToggle={toggleExpanded}
                leftName={leftSource.name}
                rightName={rightSource.name}
              />
            )}
            {viewMode === 'unified' && (
              <UnifiedDiffView
                diffs={getFilteredDiffs()}
                expandedPaths={expandedPaths}
                onToggle={toggleExpanded}
              />
            )}
            {viewMode === 'inline' && (
              <InlineDiffView
                diffs={getFilteredDiffs()}
                expandedPaths={expandedPaths}
                onToggle={toggleExpanded}
              />
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// Sub-components
const SourceSelector: React.FC<{
  label: string
  source: ComparisonSource | null
  onSelect: (source: ComparisonSource) => void
  environments: Array<{ id: string; name: string; config: any }>
  versions: Array<{ id: string; version: string; config: any; created_at: string }>
  disabled: boolean
}> = ({ label, source, onSelect, environments, versions, disabled }) => {
  const [sourceType, setSourceType] = useState<'environment' | 'version'>('environment')

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">
        {label}
      </label>
      <div className="space-y-2">
        <select
          value={sourceType}
          onChange={(e) => setSourceType(e.target.value as any)}
          disabled={disabled}
          className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
        >
          <option value="environment">Environment</option>
          <option value="version">Version</option>
        </select>
        
        {sourceType === 'environment' && (
          <select
            value={source?.id || ''}
            onChange={(e) => {
              const env = environments.find(env => env.id === e.target.value)
              if (env) {
                onSelect({
                  type: 'environment',
                  id: env.id,
                  name: env.name,
                  config: env.config
                })
              }
            }}
            disabled={disabled}
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
          >
            <option value="">Select environment...</option>
            {environments.map(env => (
              <option key={env.id} value={env.id}>{env.name}</option>
            ))}
          </select>
        )}
        
        {sourceType === 'version' && (
          <select
            value={source?.id || ''}
            onChange={(e) => {
              const ver = versions.find(v => v.id === e.target.value)
              if (ver) {
                onSelect({
                  type: 'version',
                  id: ver.id,
                  name: `Version ${ver.version}`,
                  config: ver.config,
                  timestamp: ver.created_at
                })
              }
            }}
            disabled={disabled}
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
          >
            <option value="">Select version...</option>
            {versions.map(ver => (
              <option key={ver.id} value={ver.id}>
                Version {ver.version} ({new Date(ver.created_at).toLocaleDateString()})
              </option>
            ))}
          </select>
        )}
      </div>
      
      {source && (
        <div className="mt-2 p-2 bg-gray-50 rounded text-sm text-gray-600">
          {source.type === 'environment' ? '🌍' : '📦'} {source.name}
          {source.timestamp && (
            <span className="text-xs ml-2">
              ({new Date(source.timestamp).toLocaleString()})
            </span>
          )}
        </div>
      )}
    </div>
  )
}

const SplitDiffView: React.FC<{
  diffs: DiffResult[]
  expandedPaths: Set<string>
  onToggle: (path: string) => void
  leftName: string
  rightName: string
}> = ({ diffs, expandedPaths, onToggle, leftName, rightName }) => {
  // Group diffs by parent path for tree structure
  const rootDiffs = diffs.filter(d => d.level === 0)

  return (
    <div>
      <div className="grid grid-cols-2 gap-4 mb-4">
        <div className="text-sm font-medium text-gray-700 px-4 py-2 bg-gray-100 rounded">
          {leftName}
        </div>
        <div className="text-sm font-medium text-gray-700 px-4 py-2 bg-gray-100 rounded">
          {rightName}
        </div>
      </div>
      
      <div className="space-y-2">
        {rootDiffs.map(diff => (
          <SplitDiffItem
            key={diff.path}
            diff={diff}
            allDiffs={diffs}
            expandedPaths={expandedPaths}
            onToggle={onToggle}
          />
        ))}
      </div>
    </div>
  )
}

const SplitDiffItem: React.FC<{
  diff: DiffResult
  allDiffs: DiffResult[]
  expandedPaths: Set<string>
  onToggle: (path: string) => void
}> = ({ diff, allDiffs, expandedPaths, onToggle }) => {
  const children = allDiffs.filter(d => d.parent === diff.path)
  const hasChildren = children.length > 0
  const isExpanded = expandedPaths.has(diff.path)

  const getIcon = () => {
    switch (diff.type) {
      case 'added': return <CheckCircleIcon className="h-4 w-4 text-green-500" />
      case 'deleted': return <XCircleIcon className="h-4 w-4 text-red-500" />
      case 'modified': return <ExclamationTriangleIcon className="h-4 w-4 text-yellow-500" />
      default: return null
    }
  }

  return (
    <div>
      <div className={`grid grid-cols-2 gap-4 p-2 rounded hover:bg-gray-50 ${
        diff.type !== 'unchanged' ? 'bg-gray-50' : ''
      }`}>
        <div className="flex items-start">
          <div className="flex items-center mr-2" style={{ marginLeft: `${diff.level * 20}px` }}>
            {hasChildren && (
              <button
                onClick={() => onToggle(diff.path)}
                className="p-0.5 hover:bg-gray-200 rounded"
              >
                {isExpanded ? (
                  <ChevronDownIcon className="h-4 w-4 text-gray-500" />
                ) : (
                  <ChevronRightIcon className="h-4 w-4 text-gray-500" />
                )}
              </button>
            )}
            {!hasChildren && <span className="w-5" />}
            {getIcon()}
          </div>
          <div className="flex-1">
            <span className="text-sm font-medium text-gray-700">
              {diff.path.split('.').pop()}
            </span>
            {diff.type !== 'added' && (
              <div className="mt-1">
                {renderValue(diff.left_value, diff.type)}
              </div>
            )}
          </div>
        </div>
        
        <div className="flex items-start">
          <div className="flex-1">
            {diff.type !== 'deleted' && (
              <div>
                {renderValue(diff.right_value, diff.type)}
              </div>
            )}
          </div>
        </div>
      </div>
      
      {isExpanded && hasChildren && (
        <div className="ml-4">
          {children.map(child => (
            <SplitDiffItem
              key={child.path}
              diff={child}
              allDiffs={allDiffs}
              expandedPaths={expandedPaths}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </div>
  )

  function renderValue(value: any, type: string) {
    const colorClass = 
      type === 'added' ? 'bg-green-50 text-green-700 border-green-200' :
      type === 'deleted' ? 'bg-red-50 text-red-700 border-red-200' :
      type === 'modified' ? 'bg-yellow-50 text-yellow-700 border-yellow-200' :
      ''

    if (value === null || value === undefined) {
      return <span className="text-gray-400 italic text-sm">null</span>
    }

    if (typeof value === 'object') {
      return (
        <pre className={`text-xs p-2 rounded border ${colorClass} overflow-x-auto`}>
          {JSON.stringify(value, null, 2)}
        </pre>
      )
    }

    return (
      <span className={`inline-block px-2 py-0.5 rounded text-sm font-mono border ${colorClass}`}>
        {String(value)}
      </span>
    )
  }
}

const UnifiedDiffView: React.FC<{
  diffs: DiffResult[]
  expandedPaths: Set<string>
  onToggle: (path: string) => void
}> = ({ diffs, expandedPaths, onToggle }) => {
  return (
    <div className="space-y-1">
      {diffs.map(diff => (
        <UnifiedDiffItem
          key={diff.path}
          diff={diff}
          expandedPaths={expandedPaths}
          onToggle={onToggle}
        />
      ))}
    </div>
  )
}

const UnifiedDiffItem: React.FC<{
  diff: DiffResult
  expandedPaths: Set<string>
  onToggle: (path: string) => void
}> = ({ diff, expandedPaths, onToggle }) => {
  const getLinePrefix = () => {
    switch (diff.type) {
      case 'added': return '+'
      case 'deleted': return '-'
      case 'modified': return '~'
      default: return ' '
    }
  }

  const getLineClass = () => {
    switch (diff.type) {
      case 'added': return 'bg-green-50 text-green-700'
      case 'deleted': return 'bg-red-50 text-red-700'
      case 'modified': return 'bg-yellow-50 text-yellow-700'
      default: return ''
    }
  }

  return (
    <div className={`px-4 py-2 font-mono text-sm ${getLineClass()}`}>
      <span className="select-none mr-3">{getLinePrefix()}</span>
      <span className="text-gray-600">{diff.path}:</span>
      {diff.type === 'modified' && (
        <>
          <span className="text-red-600 line-through ml-2">
            {JSON.stringify(diff.left_value)}
          </span>
          <span className="text-green-600 ml-2">
            {JSON.stringify(diff.right_value)}
          </span>
        </>
      )}
      {diff.type === 'added' && (
        <span className="text-green-600 ml-2">
          {JSON.stringify(diff.right_value)}
        </span>
      )}
      {diff.type === 'deleted' && (
        <span className="text-red-600 ml-2">
          {JSON.stringify(diff.left_value)}
        </span>
      )}
    </div>
  )
}

const InlineDiffView: React.FC<{
  diffs: DiffResult[]
  expandedPaths: Set<string>
  onToggle: (path: string) => void
}> = ({ diffs, expandedPaths, onToggle }) => {
  return (
    <div className="space-y-3">
      {diffs.map(diff => (
        <div key={diff.path} className="border rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <h4 className="font-medium text-gray-900">{diff.path}</h4>
            <span className={`px-2 py-1 text-xs rounded-full ${
              diff.type === 'added' ? 'bg-green-100 text-green-700' :
              diff.type === 'deleted' ? 'bg-red-100 text-red-700' :
              diff.type === 'modified' ? 'bg-yellow-100 text-yellow-700' :
              'bg-gray-100 text-gray-700'
            }`}>
              {diff.type}
            </span>
          </div>
          
          {diff.type === 'modified' && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-gray-500 mb-1">Before</p>
                <div className="bg-red-50 p-2 rounded">
                  <code className="text-xs text-red-700">
                    {JSON.stringify(diff.left_value, null, 2)}
                  </code>
                </div>
              </div>
              <div>
                <p className="text-xs text-gray-500 mb-1">After</p>
                <div className="bg-green-50 p-2 rounded">
                  <code className="text-xs text-green-700">
                    {JSON.stringify(diff.right_value, null, 2)}
                  </code>
                </div>
              </div>
            </div>
          )}
          
          {diff.type === 'added' && (
            <div className="bg-green-50 p-2 rounded">
              <code className="text-xs text-green-700">
                {JSON.stringify(diff.right_value, null, 2)}
              </code>
            </div>
          )}
          
          {diff.type === 'deleted' && (
            <div className="bg-red-50 p-2 rounded">
              <code className="text-xs text-red-700">
                {JSON.stringify(diff.left_value, null, 2)}
              </code>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

export default ConfigComparison