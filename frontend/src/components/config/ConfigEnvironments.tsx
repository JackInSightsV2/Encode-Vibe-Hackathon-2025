import React, { useState, useEffect } from 'react'
import { 
  GlobeAltIcon, 
  ServerIcon, 
  BeakerIcon,
  RocketLaunchIcon,
  DocumentDuplicateIcon,
  ArrowRightIcon,
  ShieldCheckIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  PlusIcon,
  PencilIcon,
  TrashIcon
} from '@heroicons/react/24/outline'

interface Environment {
  id: string
  name: string
  description: string
  type: 'development' | 'staging' | 'production' | 'custom'
  config: any
  variables: Record<string, string>
  protected: boolean
  created_at: string
  updated_at: string
  created_by: string
  active_version: string
  deployment_url?: string
  health_check_url?: string
  status?: 'healthy' | 'degraded' | 'down' | 'unknown'
}

interface EnvironmentVariable {
  key: string
  value: string
  encrypted: boolean
  description?: string
}

interface ConfigEnvironmentsProps {
  currentConfig: any
  onEnvironmentChange: (environment: Environment) => void
  onClose: () => void
}

const ConfigEnvironments: React.FC<ConfigEnvironmentsProps> = ({
  currentConfig,
  onEnvironmentChange,
  onClose
}) => {
  const [environments, setEnvironments] = useState<Environment[]>([])
  const [activeEnvironment, setActiveEnvironment] = useState<string>('development')
  const [loading, setLoading] = useState(true)
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [showVariables, setShowVariables] = useState<string | null>(null)
  const [newEnvironment, setNewEnvironment] = useState({
    name: '',
    description: '',
    type: 'custom' as const,
    protected: false,
    deployment_url: '',
    health_check_url: ''
  })

  useEffect(() => {
    fetchEnvironments()
  }, [])

  const fetchEnvironments = async () => {
    try {
      const response = await fetch('/api/config/environments')
      if (response.ok) {
        const data = await response.json()
        setEnvironments(data.data || [])
        
        // Set active environment from API or default
        const active = data.active || 'development'
        setActiveEnvironment(active)
      }
    } catch (error) {
      console.error('Failed to fetch environments:', error)
    } finally {
      setLoading(false)
    }
  }

  const createEnvironment = async () => {
    try {
      const response = await fetch('/api/config/environments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...newEnvironment,
          config: currentConfig
        })
      })

      if (response.ok) {
        await fetchEnvironments()
        setShowCreateForm(false)
        resetNewEnvironmentForm()
      }
    } catch (error) {
      console.error('Failed to create environment:', error)
    }
  }

  const updateEnvironment = async (env: Environment) => {
    try {
      const response = await fetch(`/api/config/environments/${env.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(env)
      })

      if (response.ok) {
        await fetchEnvironments()
        setEditingId(null)
      }
    } catch (error) {
      console.error('Failed to update environment:', error)
    }
  }

  const deleteEnvironment = async (env: Environment) => {
    if (env.protected) {
      alert('Cannot delete protected environment')
      return
    }

    if (!window.confirm(`Are you sure you want to delete the "${env.name}" environment?`)) {
      return
    }

    try {
      const response = await fetch(`/api/config/environments/${env.id}`, {
        method: 'DELETE'
      })

      if (response.ok) {
        await fetchEnvironments()
      }
    } catch (error) {
      console.error('Failed to delete environment:', error)
    }
  }

  const switchEnvironment = async (envId: string) => {
    const env = environments.find(e => e.id === envId)
    if (!env) return

    try {
      const response = await fetch('/api/config/environments/active', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ environment_id: envId })
      })

      if (response.ok) {
        setActiveEnvironment(envId)
        onEnvironmentChange(env)
      }
    } catch (error) {
      console.error('Failed to switch environment:', error)
    }
  }

  const resetNewEnvironmentForm = () => {
    setNewEnvironment({
      name: '',
      description: '',
      type: 'custom',
      protected: false,
      deployment_url: '',
      health_check_url: ''
    })
  }

  const getEnvironmentIcon = (type: string) => {
    switch (type) {
      case 'development':
        return BeakerIcon
      case 'staging':
        return ServerIcon
      case 'production':
        return RocketLaunchIcon
      default:
        return GlobeAltIcon
    }
  }

  const getEnvironmentColor = (type: string) => {
    switch (type) {
      case 'development':
        return 'text-blue-600 bg-blue-50 border-blue-200'
      case 'staging':
        return 'text-yellow-600 bg-yellow-50 border-yellow-200'
      case 'production':
        return 'text-red-600 bg-red-50 border-red-200'
      default:
        return 'text-gray-600 bg-gray-50 border-gray-200'
    }
  }

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'healthy':
        return 'text-green-600'
      case 'degraded':
        return 'text-yellow-600'
      case 'down':
        return 'text-red-600'
      default:
        return 'text-gray-400'
    }
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading environments...</p>
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
              Environment Management
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Manage configuration across different environments
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={() => setShowCreateForm(true)}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
            >
              <PlusIcon className="h-4 w-4 inline-block mr-2" />
              New Environment
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

      {/* Environment Grid */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {environments.map((env) => {
            const Icon = getEnvironmentIcon(env.type)
            const isActive = env.id === activeEnvironment
            const colorClass = getEnvironmentColor(env.type)

            return (
              <div
                key={env.id}
                className={`border rounded-lg p-4 transition-all ${
                  isActive ? 'ring-2 ring-blue-500' : ''
                } ${editingId === env.id ? 'bg-gray-50' : 'bg-white hover:shadow-md'}`}
              >
                <div className="flex items-start justify-between mb-3">
                  <div className={`p-2 rounded-md ${colorClass}`}>
                    <Icon className="h-6 w-6" />
                  </div>
                  <div className="flex items-center space-x-2">
                    {env.protected && (
                      <ShieldCheckIcon className="h-5 w-5 text-gray-400" title="Protected" />
                    )}
                    {env.status && (
                      <span className={`h-2 w-2 rounded-full ${
                        env.status === 'healthy' ? 'bg-green-500' :
                        env.status === 'degraded' ? 'bg-yellow-500' :
                        env.status === 'down' ? 'bg-red-500' :
                        'bg-gray-300'
                      }`} />
                    )}
                  </div>
                </div>

                {editingId === env.id ? (
                  <EnvironmentEditForm
                    environment={env}
                    onSave={(updated) => updateEnvironment({ ...env, ...updated })}
                    onCancel={() => setEditingId(null)}
                  />
                ) : (
                  <>
                    <h3 className="text-lg font-semibold text-gray-900">
                      {env.name}
                      {isActive && (
                        <span className="ml-2 text-xs font-normal text-blue-600">
                          (Active)
                        </span>
                      )}
                    </h3>
                    <p className="text-sm text-gray-600 mt-1">
                      {env.description || 'No description'}
                    </p>

                    <div className="mt-4 space-y-2 text-xs text-gray-500">
                      <div>Version: {env.active_version}</div>
                      <div>Updated: {new Date(env.updated_at).toLocaleDateString()}</div>
                      {env.deployment_url && (
                        <div className="truncate">
                          URL: <a href={env.deployment_url} className="text-blue-600 hover:underline">
                            {env.deployment_url}
                          </a>
                        </div>
                      )}
                    </div>

                    <div className="mt-4 flex items-center justify-between">
                      <button
                        onClick={() => setShowVariables(env.id)}
                        className="text-sm text-blue-600 hover:text-blue-800"
                      >
                        Variables ({Object.keys(env.variables || {}).length})
                      </button>

                      <div className="flex items-center space-x-2">
                        {!isActive && (
                          <button
                            onClick={() => switchEnvironment(env.id)}
                            className="p-1 text-gray-400 hover:text-gray-600"
                            title="Switch to this environment"
                          >
                            <ArrowRightIcon className="h-4 w-4" />
                          </button>
                        )}
                        <button
                          onClick={() => setEditingId(env.id)}
                          className="p-1 text-gray-400 hover:text-gray-600"
                          title="Edit environment"
                        >
                          <PencilIcon className="h-4 w-4" />
                        </button>
                        {!env.protected && (
                          <button
                            onClick={() => deleteEnvironment(env)}
                            className="p-1 text-gray-400 hover:text-red-600"
                            title="Delete environment"
                          >
                            <TrashIcon className="h-4 w-4" />
                          </button>
                        )}
                      </div>
                    </div>
                  </>
                )}
              </div>
            )
          })}
        </div>

        {/* Quick Actions */}
        <div className="mt-8 bg-blue-50 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">
            Environment Actions
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <button className="flex items-center p-3 bg-white rounded-md hover:shadow-md transition-shadow">
              <DocumentDuplicateIcon className="h-5 w-5 text-blue-600 mr-3" />
              <div className="text-left">
                <div className="font-medium text-sm">Clone Environment</div>
                <div className="text-xs text-gray-500">Duplicate configuration</div>
              </div>
            </button>
            <button className="flex items-center p-3 bg-white rounded-md hover:shadow-md transition-shadow">
              <ArrowRightIcon className="h-5 w-5 text-green-600 mr-3" />
              <div className="text-left">
                <div className="font-medium text-sm">Promote Config</div>
                <div className="text-xs text-gray-500">Move between environments</div>
              </div>
            </button>
            <button className="flex items-center p-3 bg-white rounded-md hover:shadow-md transition-shadow">
              <ShieldCheckIcon className="h-5 w-5 text-purple-600 mr-3" />
              <div className="text-left">
                <div className="font-medium text-sm">Validate All</div>
                <div className="text-xs text-gray-500">Check all environments</div>
              </div>
            </button>
          </div>
        </div>
      </div>

      {/* Create Environment Form */}
      {showCreateForm && (
        <CreateEnvironmentModal
          onClose={() => setShowCreateForm(false)}
          onCreate={createEnvironment}
          newEnvironment={newEnvironment}
          setNewEnvironment={setNewEnvironment}
        />
      )}

      {/* Variables Modal */}
      {showVariables && (
        <EnvironmentVariablesModal
          environment={environments.find(e => e.id === showVariables)!}
          onClose={() => setShowVariables(null)}
          onUpdate={(variables) => {
            const env = environments.find(e => e.id === showVariables)!
            updateEnvironment({ ...env, variables })
          }}
        />
      )}
    </div>
  )
}

// Sub-components
const EnvironmentEditForm: React.FC<{
  environment: Environment
  onSave: (data: any) => void
  onCancel: () => void
}> = ({ environment, onSave, onCancel }) => {
  const [formData, setFormData] = useState({
    name: environment.name,
    description: environment.description,
    deployment_url: environment.deployment_url || '',
    health_check_url: environment.health_check_url || ''
  })

  return (
    <div className="space-y-3">
      <input
        type="text"
        value={formData.name}
        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        className="w-full px-3 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
        placeholder="Environment name"
      />
      <textarea
        value={formData.description}
        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
        className="w-full px-3 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
        placeholder="Description"
        rows={2}
      />
      <div className="flex justify-end space-x-2">
        <button
          onClick={onCancel}
          className="px-3 py-1 text-sm text-gray-600 hover:text-gray-800"
        >
          Cancel
        </button>
        <button
          onClick={() => onSave(formData)}
          className="px-3 py-1 text-sm text-white bg-blue-600 rounded hover:bg-blue-700"
        >
          Save
        </button>
      </div>
    </div>
  )
}

const CreateEnvironmentModal: React.FC<{
  onClose: () => void
  onCreate: () => void
  newEnvironment: any
  setNewEnvironment: (env: any) => void
}> = ({ onClose, onCreate, newEnvironment, setNewEnvironment }) => {
  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-md w-full p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Create New Environment</h3>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Environment Name
            </label>
            <input
              type="text"
              value={newEnvironment.name}
              onChange={(e) => setNewEnvironment({ ...newEnvironment, name: e.target.value })}
              placeholder="e.g., staging-2"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Environment Type
            </label>
            <select
              value={newEnvironment.type}
              onChange={(e) => setNewEnvironment({ ...newEnvironment, type: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="development">Development</option>
              <option value="staging">Staging</option>
              <option value="production">Production</option>
              <option value="custom">Custom</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={newEnvironment.description}
              onChange={(e) => setNewEnvironment({ ...newEnvironment, description: e.target.value })}
              placeholder="Optional description..."
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={newEnvironment.protected}
                onChange={(e) => setNewEnvironment({ ...newEnvironment, protected: e.target.checked })}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <span className="ml-2 text-sm text-gray-700">
                Protected environment (prevent accidental deletion)
              </span>
            </label>
          </div>
        </div>

        <div className="mt-6 flex justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={onCreate}
            disabled={!newEnvironment.name}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Create Environment
          </button>
        </div>
      </div>
    </div>
  )
}

const EnvironmentVariablesModal: React.FC<{
  environment: Environment
  onClose: () => void
  onUpdate: (variables: Record<string, string>) => void
}> = ({ environment, onClose, onUpdate }) => {
  const [variables, setVariables] = useState<EnvironmentVariable[]>(
    Object.entries(environment.variables || {}).map(([key, value]) => ({
      key,
      value: value as string,
      encrypted: false
    }))
  )
  const [newVar, setNewVar] = useState({ key: '', value: '' })

  const addVariable = () => {
    if (newVar.key && newVar.value) {
      setVariables([...variables, { ...newVar, encrypted: false }])
      setNewVar({ key: '', value: '' })
    }
  }

  const removeVariable = (index: number) => {
    setVariables(variables.filter((_, i) => i !== index))
  }

  const saveVariables = () => {
    const varsObject = variables.reduce((acc, v) => ({
      ...acc,
      [v.key]: v.value
    }), {})
    onUpdate(varsObject)
    onClose()
  }

  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-2xl w-full p-6 max-h-[80vh] overflow-hidden flex flex-col">
        <h3 className="text-lg font-medium text-gray-900 mb-4">
          Environment Variables - {environment.name}
        </h3>
        
        <div className="flex-1 overflow-y-auto">
          <div className="space-y-2">
            {variables.map((v, index) => (
              <div key={index} className="flex items-center space-x-2">
                <input
                  type="text"
                  value={v.key}
                  onChange={(e) => {
                    const updated = [...variables]
                    updated[index].key = e.target.value
                    setVariables(updated)
                  }}
                  className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-md"
                  placeholder="Variable name"
                />
                <input
                  type={v.encrypted ? 'password' : 'text'}
                  value={v.value}
                  onChange={(e) => {
                    const updated = [...variables]
                    updated[index].value = e.target.value
                    setVariables(updated)
                  }}
                  className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-md"
                  placeholder="Value"
                />
                <button
                  onClick={() => removeVariable(index)}
                  className="p-2 text-red-600 hover:text-red-800"
                >
                  <TrashIcon className="h-4 w-4" />
                </button>
              </div>
            ))}
            
            <div className="flex items-center space-x-2 pt-2">
              <input
                type="text"
                value={newVar.key}
                onChange={(e) => setNewVar({ ...newVar, key: e.target.value })}
                className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-md"
                placeholder="New variable name"
              />
              <input
                type="text"
                value={newVar.value}
                onChange={(e) => setNewVar({ ...newVar, value: e.target.value })}
                className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-md"
                placeholder="Value"
              />
              <button
                onClick={addVariable}
                disabled={!newVar.key || !newVar.value}
                className="px-4 py-2 text-sm text-blue-600 hover:text-blue-800 disabled:opacity-50"
              >
                Add
              </button>
            </div>
          </div>
        </div>

        <div className="mt-6 flex justify-end space-x-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={saveVariables}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
          >
            Save Variables
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConfigEnvironments