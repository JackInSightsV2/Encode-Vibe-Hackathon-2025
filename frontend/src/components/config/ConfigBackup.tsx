import React, { useState, useEffect } from 'react'
import { 
  CloudArrowUpIcon, 
  CloudArrowDownIcon,
  ArchiveBoxIcon,
  ClockIcon,
  ShieldCheckIcon,
  TrashIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon
} from '@heroicons/react/24/outline'

// Utility functions
const formatSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString()
}

interface Backup {
  id: string
  name: string
  description: string
  created_at: string
  created_by: string
  size: number
  version: string
  encrypted: boolean
  compressed: boolean
  tags: string[]
  retention_days: number
  auto_backup: boolean
}

interface BackupPolicy {
  enabled: boolean
  frequency: string
  retention_days: number
  max_backups: number
  compress: boolean
  encrypt: boolean
  exclude_sensitive: boolean
}

interface ConfigBackupProps {
  config: any
  onClose: () => void
  onBackupCreated: () => void
}

const ConfigBackup: React.FC<ConfigBackupProps> = ({
  config,
  onClose,
  onBackupCreated
}) => {
  const [backups, setBackups] = useState<Backup[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [policy, setPolicy] = useState<BackupPolicy | null>(null)
  const [newBackup, setNewBackup] = useState({
    name: '',
    description: '',
    compress: true,
    encrypt: false,
    tags: [] as string[]
  })

  useEffect(() => {
    fetchBackups()
    fetchBackupPolicy()
  }, [])

  const fetchBackups = async () => {
    try {
      const response = await fetch('/api/config/backups')
      if (response.ok) {
        const data = await response.json()
        setBackups(data.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch backups:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchBackupPolicy = async () => {
    try {
      const response = await fetch('/api/config/backup-policy')
      if (response.ok) {
        const data = await response.json()
        setPolicy(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch backup policy:', error)
    }
  }

  const createBackup = async () => {
    setCreating(true)
    try {
      const response = await fetch('/api/config/backup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...newBackup,
          config: config
        })
      })

      if (response.ok) {
        onBackupCreated()
        setShowCreateForm(false)
        setNewBackup({ name: '', description: '', compress: true, encrypt: false, tags: [] })
        await fetchBackups()
      }
    } catch (error) {
      console.error('Failed to create backup:', error)
    } finally {
      setCreating(false)
    }
  }

  const deleteBackup = async (backup: Backup) => {
    if (!window.confirm(`Are you sure you want to delete backup "${backup.name}"?`)) {
      return
    }

    setDeletingId(backup.id)
    try {
      const response = await fetch(`/api/config/backup/${backup.id}`, {
        method: 'DELETE'
      })

      if (response.ok) {
        await fetchBackups()
      }
    } catch (error) {
      console.error('Failed to delete backup:', error)
    } finally {
      setDeletingId(null)
    }
  }

  const downloadBackup = async (backup: Backup) => {
    try {
      const response = await fetch(`/api/config/backup/${backup.id}/download`)
      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `config-backup-${backup.version}.json`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      }
    } catch (error) {
      console.error('Failed to download backup:', error)
    }
  }


  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading backups...</p>
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
              Configuration Backups
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Create and manage configuration backups
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={() => setShowCreateForm(true)}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
            >
              <CloudArrowUpIcon className="h-4 w-4 inline-block mr-2" />
              Create Backup
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

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {/* Backup Policy */}
        {policy && (
          <div className="p-6 bg-gray-50 border-b">
            <h3 className="text-sm font-medium text-gray-900 mb-3">Automatic Backup Policy</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <div className="text-xs text-gray-500">Status</div>
                <div className="flex items-center mt-1">
                  {policy.enabled ? (
                    <>
                      <CheckCircleIcon className="h-4 w-4 text-green-500 mr-1" />
                      <span className="text-sm font-medium text-green-700">Enabled</span>
                    </>
                  ) : (
                    <>
                      <ExclamationTriangleIcon className="h-4 w-4 text-gray-400 mr-1" />
                      <span className="text-sm font-medium text-gray-600">Disabled</span>
                    </>
                  )}
                </div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Frequency</div>
                <div className="text-sm font-medium text-gray-900 mt-1">{policy.frequency}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Retention</div>
                <div className="text-sm font-medium text-gray-900 mt-1">{policy.retention_days} days</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">Max Backups</div>
                <div className="text-sm font-medium text-gray-900 mt-1">{policy.max_backups}</div>
              </div>
            </div>
            <div className="mt-3 flex items-center space-x-4 text-xs">
              {policy.compress && (
                <span className="flex items-center text-gray-600">
                  <ArchiveBoxIcon className="h-3 w-3 mr-1" />
                  Compression enabled
                </span>
              )}
              {policy.encrypt && (
                <span className="flex items-center text-gray-600">
                  <ShieldCheckIcon className="h-3 w-3 mr-1" />
                  Encryption enabled
                </span>
              )}
            </div>
          </div>
        )}

        {/* Backup List */}
        <div className="p-6">
          {backups.length === 0 ? (
            <div className="text-center py-12">
              <ArchiveBoxIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
              <p className="text-gray-500">No backups found</p>
              <p className="text-sm text-gray-400 mt-2">Create your first backup to get started</p>
            </div>
          ) : (
            <div className="space-y-4">
              {backups.map((backup) => (
                <BackupItem
                  key={backup.id}
                  backup={backup}
                  onDownload={() => downloadBackup(backup)}
                  onDelete={() => deleteBackup(backup)}
                  isDeleting={deletingId === backup.id}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Create Backup Form */}
      {showCreateForm && (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Create New Backup</h3>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Backup Name
                </label>
                <input
                  type="text"
                  value={newBackup.name}
                  onChange={(e) => setNewBackup({ ...newBackup, name: e.target.value })}
                  placeholder="e.g., Pre-deployment backup"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description
                </label>
                <textarea
                  value={newBackup.description}
                  onChange={(e) => setNewBackup({ ...newBackup, description: e.target.value })}
                  placeholder="Optional description..."
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="space-y-2">
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={newBackup.compress}
                    onChange={(e) => setNewBackup({ ...newBackup, compress: e.target.checked })}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <span className="ml-2 text-sm text-gray-700">
                    Compress backup (reduce file size)
                  </span>
                </label>

                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={newBackup.encrypt}
                    onChange={(e) => setNewBackup({ ...newBackup, encrypt: e.target.checked })}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <span className="ml-2 text-sm text-gray-700">
                    Encrypt backup (protect sensitive data)
                  </span>
                </label>
              </div>
            </div>

            <div className="mt-6 flex justify-end space-x-3">
              <button
                onClick={() => setShowCreateForm(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={createBackup}
                disabled={!newBackup.name || creating}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {creating ? 'Creating...' : 'Create Backup'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const BackupItem: React.FC<{
  backup: Backup
  onDownload: () => void
  onDelete: () => void
  isDeleting: boolean
}> = ({ backup, onDownload, onDelete, isDeleting }) => {
  return (
    <div className="bg-white border rounded-lg p-4">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center">
            <h4 className="text-sm font-medium text-gray-900">{backup.name}</h4>
            {backup.auto_backup && (
              <span className="ml-2 px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-800 rounded-full">
                Auto
              </span>
            )}
            {backup.tags.map((tag) => (
              <span
                key={tag}
                className="ml-2 px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-700 rounded-full"
              >
                {tag}
              </span>
            ))}
          </div>
          {backup.description && (
            <p className="text-sm text-gray-600 mt-1">{backup.description}</p>
          )}
          <div className="mt-2 flex items-center space-x-4 text-xs text-gray-500">
            <span className="flex items-center">
              <ClockIcon className="h-3 w-3 mr-1" />
              {formatDate(backup.created_at)}
            </span>
            <span>by {backup.created_by}</span>
            <span>{formatSize(backup.size)}</span>
            <span>v{backup.version}</span>
          </div>
          <div className="mt-2 flex items-center space-x-3 text-xs">
            {backup.compressed && (
              <span className="flex items-center text-gray-600">
                <ArchiveBoxIcon className="h-3 w-3 mr-1" />
                Compressed
              </span>
            )}
            {backup.encrypted && (
              <span className="flex items-center text-gray-600">
                <ShieldCheckIcon className="h-3 w-3 mr-1" />
                Encrypted
              </span>
            )}
            <span className="text-gray-500">
              Expires in {backup.retention_days} days
            </span>
          </div>
        </div>

        <div className="ml-4 flex items-center space-x-2">
          <button
            onClick={onDownload}
            className="p-2 text-gray-400 hover:text-gray-600"
            title="Download backup"
          >
            <CloudArrowDownIcon className="h-5 w-5" />
          </button>
          <button
            onClick={onDelete}
            disabled={isDeleting}
            className="p-2 text-gray-400 hover:text-red-600 disabled:opacity-50"
            title="Delete backup"
          >
            {isDeleting ? (
              <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-gray-600"></div>
            ) : (
              <TrashIcon className="h-5 w-5" />
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConfigBackup