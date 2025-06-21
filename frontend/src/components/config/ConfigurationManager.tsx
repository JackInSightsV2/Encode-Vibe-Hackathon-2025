import React, { useState, useEffect, useCallback } from 'react'
import { 
  Cog6ToothIcon, 
  ServerStackIcon, 
  ShieldCheckIcon, 
  CloudIcon,
  CpuChipIcon,
  DocumentTextIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  InformationCircleIcon
} from '@heroicons/react/24/outline'
import ConfigEditor from './ConfigEditor'

// Wrapper components to bridge interface differences
const ServerConfigSection: React.FC<ConfigSectionProps> = ({ config, onChange }) => (
  <ConfigEditor 
    value={config.server || {}} 
    path={['server']} 
    onChange={onChange} 
    schema={{}}
    errors={[]}
  />
)

const ProvidersConfigSection: React.FC<ConfigSectionProps> = ({ config, onChange }) => (
  <ConfigEditor 
    value={config.providers || {}} 
    path={['providers']} 
    onChange={onChange} 
    schema={{}}
    errors={[]}
  />
)

const SecurityConfigSection: React.FC<ConfigSectionProps> = ({ config, onChange }) => (
  <ConfigEditor 
    value={config.security || {}} 
    path={['security']} 
    onChange={onChange} 
    schema={{}}
    errors={[]}
  />
)

const ModerationConfigSection: React.FC<ConfigSectionProps> = ({ config, onChange }) => (
  <ConfigEditor 
    value={config.moderation || {}} 
    path={['moderation']} 
    onChange={onChange} 
    schema={{}}
    errors={[]}
  />
)

const LoggingConfigSection: React.FC<ConfigSectionProps> = ({ config, onChange }) => (
  <ConfigEditor 
    value={config.logging || {}} 
    path={['logging']} 
    onChange={onChange} 
    schema={{}}
    errors={[]}
  />
)
import ConfigValidator from './ConfigValidator'
import ConfigVersions from './ConfigVersions'
import ConfigBackup from './ConfigBackup'
import ConfigWizard from './ConfigWizard'
import { useConfigValidation } from '../../hooks/useConfigValidation'
import { useAutoSave } from '../../hooks/useAutoSave'

interface ConfigSection {
  key: string
  title: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  component: React.ComponentType<ConfigSectionProps>
  advanced?: boolean
}

interface ConfigSectionProps {
  config: any
  schema: any
  onChange: (path: string[], value: any) => void
  errors: ValidationError[]
  mode?: 'form' | 'json' | 'split'
}

interface ValidationError {
  field: string
  message: string
  code: string
  severity: string
  suggestion?: string
}

interface ConfigurationManagerProps {
  showWizard?: boolean
  onWizardComplete?: () => void
}

const ConfigurationManager: React.FC<ConfigurationManagerProps> = ({ 
  showWizard = false, 
  onWizardComplete 
}) => {
  const [config, setConfig] = useState<any>(null)
  const [schema, setSchema] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [activeSection, setActiveSection] = useState('server')
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [editMode, setEditMode] = useState<'form' | 'json' | 'split'>('form')
  const [message, setMessage] = useState<{type: 'success' | 'error' | 'info', text: string} | null>(null)
  const [showVersions, setShowVersions] = useState(false)
  const [showBackup, setShowBackup] = useState(false)
  const [isWizardOpen, setIsWizardOpen] = useState(showWizard)

  // Custom hooks
  const { validationResult, validate, isValidating } = useConfigValidation()
  const { hasChanges, markSaved } = useAutoSave(config, handleSave, 30000) // Auto-save every 30 seconds

  useEffect(() => {
    fetchConfigAndSchema()
  }, [])

  const fetchConfigAndSchema = async () => {
    try {
      const [configRes, schemaRes] = await Promise.all([
        fetch('/api/config'),
        fetch('/api/config/schema')
      ])

      if (configRes.ok && schemaRes.ok) {
        const configData = await configRes.json()
        const schemaData = await schemaRes.json()
        
        setConfig(configData.data)
        setSchema(schemaData.data)
      } else {
        throw new Error('Failed to fetch configuration')
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Failed to load configuration' })
    } finally {
      setLoading(false)
    }
  }

  const updateConfig = useCallback((path: string[], value: any) => {
    if (!config) return

    const newConfig = JSON.parse(JSON.stringify(config))
    let current: any = newConfig

    for (let i = 0; i < path.length - 1; i++) {
      if (!current[path[i]]) {
        current[path[i]] = {}
      }
      current = current[path[i]]
    }
    current[path[path.length - 1]] = value

    setConfig(newConfig)
    
    // Validate on change
    if (schema) {
      validate(newConfig)
    }
  }, [config, schema, validate])

  async function handleSave() {
    if (!config) return

    setSaving(true)
    setMessage(null)

    try {
      // Validate before saving
      const validation = await fetch('/api/config/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })

      const validationResult = await validation.json()
      
      if (!validationResult.success) {
        setMessage({ type: 'error', text: 'Configuration validation failed. Please fix errors before saving.' })
        setSaving(false)
        return
      }

      // Save configuration
      const response = await fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })

      if (response.ok) {
        setMessage({ type: 'success', text: 'Configuration saved successfully!' })
        markSaved()
        
        // Create automatic backup
        await fetch('/api/config/backup', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: 'Auto-backup after save',
            description: 'Automatic backup created after configuration save'
          })
        })
      } else {
        const error = await response.json()
        throw new Error(error.error || 'Failed to save configuration')
      }
    } catch (error: any) {
      setMessage({ type: 'error', text: error.message || 'Failed to save configuration' })
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    try {
      const response = await fetch('/api/config/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      })

      const result = await response.json()
      
      if (result.success) {
        setMessage({ 
          type: 'success', 
          text: `Configuration test completed: ${result.data.passed_count} passed, ${result.data.failed_count} failed` 
        })
      } else {
        setMessage({ type: 'error', text: 'Configuration test failed' })
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Failed to test configuration' })
    }
  }

  const handleReload = async () => {
    setLoading(true)
    await fetchConfigAndSchema()
    setMessage({ type: 'info', text: 'Configuration reloaded' })
  }

  const sections: ConfigSection[] = [
    {
      key: 'server',
      title: 'Server',
      description: 'Basic server configuration',
      icon: ServerStackIcon,
      component: ServerConfigSection
    },
    {
      key: 'providers',
      title: 'Providers',
      description: 'AI provider configuration',
      icon: CloudIcon,
      component: ProvidersConfigSection
    },
    {
      key: 'security',
      title: 'Security',
      description: 'Security and protection settings',
      icon: ShieldCheckIcon,
      component: SecurityConfigSection
    },
    {
      key: 'moderation',
      title: 'Moderation',
      description: 'Content moderation settings',
      icon: DocumentTextIcon,
      component: ModerationConfigSection
    },
    {
      key: 'routing',
      title: 'Routing',
      description: 'Request routing configuration',
      icon: ArrowPathIcon,
      component: LoggingConfigSection,
      advanced: true
    },
    {
      key: 'database',
      title: 'Database',
      description: 'Database connection settings',
      icon: CpuChipIcon,
      component: LoggingConfigSection,
      advanced: true
    },
    {
      key: 'metrics',
      title: 'Metrics',
      description: 'Metrics and monitoring',
      icon: Cog6ToothIcon,
      component: LoggingConfigSection,
      advanced: true
    }
  ]

  const visibleSections = showAdvanced 
    ? sections 
    : sections.filter(s => !s.advanced)

  const getValidationErrors = (section: string): ValidationError[] => {
    if (!validationResult || !validationResult.errors) return []
    
    return validationResult.errors.filter((error: ValidationError) => 
      error.field.startsWith(section + '.')
    )
  }

  const getSectionStatus = (section: string): 'valid' | 'error' | 'warning' => {
    const errors = getValidationErrors(section)
    if (errors.length === 0) return 'valid'
    
    const hasErrors = errors.some(e => e.severity === 'error' || e.severity === 'critical')
    return hasErrors ? 'error' : 'warning'
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading configuration...</p>
        </div>
      </div>
    )
  }

  if (isWizardOpen) {
    return (
      <ConfigWizard
        schema={schema}
        onComplete={(wizardConfig) => {
          setConfig(wizardConfig)
          setIsWizardOpen(false)
          if (onWizardComplete) onWizardComplete()
        }}
        onSkip={() => {
          setIsWizardOpen(false)
          if (onWizardComplete) onWizardComplete()
        }}
      />
    )
  }

  if (showVersions) {
    return (
      <ConfigVersions
        currentConfig={config}
        onClose={() => setShowVersions(false)}
        onRestore={(restoredConfig) => {
          setConfig(restoredConfig)
          setShowVersions(false)
          setMessage({ type: 'success', text: 'Configuration restored successfully' })
        }}
      />
    )
  }

  if (showBackup) {
    return (
      <ConfigBackup
        config={config}
        onClose={() => setShowBackup(false)}
        onBackupCreated={() => {
          setMessage({ type: 'success', text: 'Backup created successfully' })
        }}
      />
    )
  }

  const activeComponent = sections.find(s => s.key === activeSection)?.component || ConfigEditor

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Configuration Management</h1>
            <p className="text-sm text-gray-600 mt-1">
              Manage QT-1 middleware settings with real-time validation
            </p>
          </div>
          
          <div className="flex items-center space-x-3">
            {hasChanges && (
              <span className="text-sm text-orange-600 flex items-center">
                <ExclamationCircleIcon className="h-4 w-4 mr-1" />
                Unsaved changes
              </span>
            )}
            
            <button
              onClick={handleTest}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Test Config
            </button>
            
            <button
              onClick={() => setShowVersions(true)}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Versions
            </button>
            
            <button
              onClick={() => setShowBackup(true)}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
            >
              Backup
            </button>
            
            <button
              onClick={handleReload}
              className="p-2 text-gray-500 hover:text-gray-700"
              title="Reload configuration"
            >
              <ArrowPathIcon className="h-5 w-5" />
            </button>
            
            <button
              onClick={handleSave}
              disabled={saving || isValidating}
              className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>
      </div>

      {/* Message */}
      {message && (
        <div className={`mx-6 mt-4 p-4 rounded-md flex items-start ${
          message.type === 'success' ? 'bg-green-50 text-green-800 border border-green-200' :
          message.type === 'error' ? 'bg-red-50 text-red-800 border border-red-200' :
          'bg-blue-50 text-blue-800 border border-blue-200'
        }`}>
          {message.type === 'success' && <CheckCircleIcon className="h-5 w-5 mr-2 flex-shrink-0 mt-0.5" />}
          {message.type === 'error' && <ExclamationCircleIcon className="h-5 w-5 mr-2 flex-shrink-0 mt-0.5" />}
          {message.type === 'info' && <InformationCircleIcon className="h-5 w-5 mr-2 flex-shrink-0 mt-0.5" />}
          <div className="flex-1">
            <p className="text-sm">{message.text}</p>
          </div>
          <button
            onClick={() => setMessage(null)}
            className="ml-4 text-gray-400 hover:text-gray-600"
          >
            ×
          </button>
        </div>
      )}

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar */}
        <div className="w-64 bg-gray-50 border-r overflow-y-auto">
          <div className="p-4">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">
                Sections
              </h2>
              <button
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="text-xs text-blue-600 hover:text-blue-800"
              >
                {showAdvanced ? 'Hide' : 'Show'} Advanced
              </button>
            </div>
            
            <nav className="space-y-1">
              {visibleSections.map((section) => {
                const Icon = section.icon
                const status = getSectionStatus(section.key)
                const errorCount = getValidationErrors(section.key).length
                
                return (
                  <button
                    key={section.key}
                    onClick={() => setActiveSection(section.key)}
                    className={`w-full flex items-center px-3 py-2 text-sm font-medium rounded-md transition-colors ${
                      activeSection === section.key
                        ? 'bg-blue-100 text-blue-700'
                        : 'text-gray-700 hover:bg-gray-100'
                    }`}
                  >
                    <Icon className="h-5 w-5 mr-3 flex-shrink-0" />
                    <div className="flex-1 text-left">
                      <div className="flex items-center justify-between">
                        <span>{section.title}</span>
                        {errorCount > 0 && (
                          <span className={`ml-2 px-2 py-0.5 text-xs rounded-full ${
                            status === 'error' 
                              ? 'bg-red-100 text-red-700' 
                              : 'bg-yellow-100 text-yellow-700'
                          }`}>
                            {errorCount}
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-gray-500 mt-0.5">
                        {section.description}
                      </p>
                    </div>
                  </button>
                )
              })}
            </nav>
          </div>
          
          {/* Validation Summary */}
          <div className="border-t p-4">
            <ConfigValidator
              validationResult={validationResult}
              isValidating={isValidating}
              compact
            />
          </div>
        </div>

        {/* Content Area */}
        <div className="flex-1 overflow-hidden">
          <div className="h-full p-6 overflow-y-auto">
            {/* Edit Mode Toggle */}
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">
                {sections.find(s => s.key === activeSection)?.title}
              </h2>
              
              <div className="flex items-center space-x-2">
                <span className="text-sm text-gray-500">View mode:</span>
                <div className="inline-flex rounded-md shadow-sm" role="group">
                  <button
                    onClick={() => setEditMode('form')}
                    className={`px-3 py-1 text-sm font-medium rounded-l-md border ${
                      editMode === 'form'
                        ? 'bg-blue-600 text-white border-blue-600'
                        : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
                    }`}
                  >
                    Form
                  </button>
                  <button
                    onClick={() => setEditMode('json')}
                    className={`px-3 py-1 text-sm font-medium border-t border-b ${
                      editMode === 'json'
                        ? 'bg-blue-600 text-white border-blue-600'
                        : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
                    }`}
                  >
                    JSON
                  </button>
                  <button
                    onClick={() => setEditMode('split')}
                    className={`px-3 py-1 text-sm font-medium rounded-r-md border ${
                      editMode === 'split'
                        ? 'bg-blue-600 text-white border-blue-600'
                        : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
                    }`}
                  >
                    Split
                  </button>
                </div>
              </div>
            </div>

            {/* Section Content */}
            {React.createElement(activeComponent as React.ComponentType<ConfigSectionProps>, {
              config: config?.[activeSection] || {},
              schema: schema?.sections?.[activeSection] || {},
              onChange: (path: string[], value: any) => {
                updateConfig([activeSection, ...path], value)
              },
              errors: getValidationErrors(activeSection),
              mode: editMode
            })}
          </div>
        </div>
      </div>
    </div>
  )
}

export default ConfigurationManager