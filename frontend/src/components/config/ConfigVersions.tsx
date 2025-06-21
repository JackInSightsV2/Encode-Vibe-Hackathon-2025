import React, { useState, useEffect } from 'react'
import { 
  ClockIcon, 
  ArrowPathIcon, 
  DocumentDuplicateIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  ChevronRightIcon,
  ArrowLeftIcon
} from '@heroicons/react/24/outline'

// Utility functions
const formatSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} minutes ago`
  if (diffHours < 24) return `${diffHours} hours ago`
  if (diffDays < 7) return `${diffDays} days ago`
  
  return date.toLocaleDateString()
}

interface ConfigVersion {
  id: string
  version: string
  created_at: string
  created_by: string
  description: string
  size: number
  changes_count: number
  is_current: boolean
  tags: string[]
}

interface ConfigDiff {
  field: string
  old_value: any
  new_value: any
  change_type: 'added' | 'modified' | 'deleted'
}

interface ConfigVersionsProps {
  currentConfig: any
  onClose: () => void
  onRestore: (config: any) => void
}

const ConfigVersions: React.FC<ConfigVersionsProps> = ({
  currentConfig,
  onClose,
  onRestore
}) => {
  const [versions, setVersions] = useState<ConfigVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedVersion, setSelectedVersion] = useState<ConfigVersion | null>(null)
  const [comparing, setComparing] = useState(false)
  const [compareVersion, setCompareVersion] = useState<ConfigVersion | null>(null)
  const [diff, setDiff] = useState<ConfigDiff[]>([])
  const [restoringId, setRestoringId] = useState<string | null>(null)

  useEffect(() => {
    fetchVersions()
  }, [])

  const fetchVersions = async () => {
    try {
      const response = await fetch('/api/config/versions')
      if (response.ok) {
        const data = await response.json()
        setVersions(data.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch versions:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCompare = async (version1: ConfigVersion, version2: ConfigVersion) => {
    try {
      const response = await fetch('/api/config/diff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          version1: version1.version,
          version2: version2.version
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        setDiff(data.data || [])
        setComparing(true)
      }
    } catch (error) {
      console.error('Failed to compare versions:', error)
    }
  }

  const handleRestore = async (version: ConfigVersion) => {
    if (!window.confirm(`Are you sure you want to restore configuration to version ${version.version}?`)) {
      return
    }

    setRestoringId(version.id)
    try {
      const response = await fetch(`/api/config/restore/${version.version}`, {
        method: 'POST'
      })
      
      if (response.ok) {
        const data = await response.json()
        onRestore(data.data)
      }
    } catch (error) {
      console.error('Failed to restore version:', error)
    } finally {
      setRestoringId(null)
    }
  }


  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading version history...</p>
        </div>
      </div>
    )
  }

  if (comparing && selectedVersion && compareVersion) {
    return (
      <div className="h-full flex flex-col">
        {/* Header */}
        <div className="bg-white border-b px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <button
                onClick={() => {
                  setComparing(false)
                  setCompareVersion(null)
                  setDiff([])
                }}
                className="mr-4 p-2 text-gray-500 hover:text-gray-700"
              >
                <ArrowLeftIcon className="h-5 w-5" />
              </button>
              <h2 className="text-xl font-semibold text-gray-900">
                Configuration Comparison
              </h2>
            </div>
          </div>
          <div className="mt-2 flex items-center text-sm text-gray-600">
            <span className="font-medium">{selectedVersion.version}</span>
            <ChevronRightIcon className="h-4 w-4 mx-2" />
            <span className="font-medium">{compareVersion.version}</span>
          </div>
        </div>

        {/* Diff View */}
        <div className="flex-1 overflow-y-auto p-6">
          {diff.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <CheckCircleIcon className="h-12 w-12 mx-auto mb-4 text-green-400" />
              <p>No differences found between these versions</p>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="text-sm text-gray-600 mb-4">
                Found {diff.length} difference{diff.length !== 1 ? 's' : ''}
              </div>
              {diff.map((change, index) => (
                <DiffItem key={index} diff={change} />
              ))}
            </div>
          )}
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
              Configuration Version History
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              View and restore previous configuration versions
            </p>
          </div>
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

      {/* Version List */}
      <div className="flex-1 overflow-y-auto">
        {versions.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <ClockIcon className="h-12 w-12 mx-auto mb-4 text-gray-300" />
            <p>No version history available</p>
          </div>
        ) : (
          <div className="divide-y">
            {versions.map((version) => (
              <VersionItem
                key={version.id}
                version={version}
                isSelected={selectedVersion?.id === version.id}
                onSelect={() => setSelectedVersion(version)}
                onCompare={() => {
                  if (selectedVersion && selectedVersion.id !== version.id) {
                    setCompareVersion(version)
                    handleCompare(selectedVersion, version)
                  }
                }}
                onRestore={() => handleRestore(version)}
                isRestoring={restoringId === version.id}
                canCompare={!!selectedVersion && selectedVersion.id !== version.id}
              />
            ))}
          </div>
        )}
      </div>

      {/* Actions */}
      {selectedVersion && (
        <div className="border-t p-4 bg-gray-50">
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600">
              Selected: Version {selectedVersion.version}
            </div>
            <div className="flex items-center space-x-3">
              <button
                onClick={() => setSelectedVersion(null)}
                className="px-3 py-1 text-sm text-gray-600 hover:text-gray-800"
              >
                Clear Selection
              </button>
              <button
                onClick={() => handleRestore(selectedVersion)}
                disabled={selectedVersion.is_current}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Restore This Version
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const VersionItem: React.FC<{
  version: ConfigVersion
  isSelected: boolean
  onSelect: () => void
  onCompare: () => void
  onRestore: () => void
  isRestoring: boolean
  canCompare: boolean
}> = ({ version, isSelected, onSelect, onCompare, onRestore, isRestoring, canCompare }) => {
  return (
    <div
      className={`p-4 hover:bg-gray-50 cursor-pointer transition-colors ${
        isSelected ? 'bg-blue-50' : ''
      }`}
      onClick={onSelect}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center">
            <h3 className="text-sm font-medium text-gray-900">
              Version {version.version}
            </h3>
            {version.is_current && (
              <span className="ml-2 px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                Current
              </span>
            )}
            {version.tags.map((tag) => (
              <span
                key={tag}
                className="ml-2 px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-700 rounded-full"
              >
                {tag}
              </span>
            ))}
          </div>
          <p className="text-sm text-gray-600 mt-1">
            {version.description || 'No description'}
          </p>
          <div className="mt-2 flex items-center space-x-4 text-xs text-gray-500">
            <span className="flex items-center">
              <ClockIcon className="h-3 w-3 mr-1" />
              {formatDate(version.created_at)}
            </span>
            <span>by {version.created_by}</span>
            <span>{formatSize(version.size)}</span>
            <span>{version.changes_count} changes</span>
          </div>
        </div>
        
        <div className="ml-4 flex items-center space-x-2">
          {canCompare && (
            <button
              onClick={(e) => {
                e.stopPropagation()
                onCompare()
              }}
              className="p-2 text-gray-400 hover:text-gray-600"
              title="Compare with selected"
            >
              <DocumentDuplicateIcon className="h-5 w-5" />
            </button>
          )}
          {!version.is_current && (
            <button
              onClick={(e) => {
                e.stopPropagation()
                onRestore()
              }}
              disabled={isRestoring}
              className="p-2 text-gray-400 hover:text-gray-600 disabled:opacity-50"
              title="Restore this version"
            >
              {isRestoring ? (
                <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-gray-600"></div>
              ) : (
                <ArrowPathIcon className="h-5 w-5" />
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

const DiffItem: React.FC<{ diff: ConfigDiff }> = ({ diff }) => {
  const getChangeColor = () => {
    switch (diff.change_type) {
      case 'added': return 'bg-green-50 border-green-200'
      case 'deleted': return 'bg-red-50 border-red-200'
      case 'modified': return 'bg-yellow-50 border-yellow-200'
    }
  }

  const getChangeIcon = () => {
    switch (diff.change_type) {
      case 'added': return '+'
      case 'deleted': return '-'
      case 'modified': return '~'
    }
  }

  return (
    <div className={`p-4 rounded-md border ${getChangeColor()}`}>
      <div className="flex items-start">
        <span className={`text-lg font-bold mr-3 ${
          diff.change_type === 'added' ? 'text-green-600' :
          diff.change_type === 'deleted' ? 'text-red-600' :
          'text-yellow-600'
        }`}>
          {getChangeIcon()}
        </span>
        <div className="flex-1">
          <div className="font-mono text-sm text-gray-700">
            {diff.field}
          </div>
          <div className="mt-2 space-y-1">
            {diff.change_type !== 'added' && (
              <div className="text-sm">
                <span className="text-gray-500">Old:</span>
                <code className="ml-2 px-2 py-1 bg-gray-100 rounded text-red-600">
                  {JSON.stringify(diff.old_value)}
                </code>
              </div>
            )}
            {diff.change_type !== 'deleted' && (
              <div className="text-sm">
                <span className="text-gray-500">New:</span>
                <code className="ml-2 px-2 py-1 bg-gray-100 rounded text-green-600">
                  {JSON.stringify(diff.new_value)}
                </code>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default ConfigVersions