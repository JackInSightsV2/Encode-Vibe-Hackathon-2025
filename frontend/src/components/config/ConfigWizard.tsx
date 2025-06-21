import React, { useState } from 'react'
import { 
  CheckIcon,
  ChevronRightIcon,
  ChevronLeftIcon,
  ServerStackIcon,
  CloudIcon,
  ShieldCheckIcon,
  DocumentTextIcon,
  CpuChipIcon,
  RocketLaunchIcon
} from '@heroicons/react/24/outline'

interface WizardStep {
  id: string
  title: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  fields: string[]
}

interface ConfigWizardProps {
  schema: any
  onComplete: (config: any) => void
  onSkip: () => void
}

const ConfigWizard: React.FC<ConfigWizardProps> = ({
  schema,
  onComplete,
  onSkip
}) => {
  const [currentStep, setCurrentStep] = useState(0)
  const [config, setConfig] = useState<any>({
    server: {
      port: 8080,
      host: 'localhost'
    },
    providers: {},
    security: {
      prompt_injection: {
        enabled: false,
        sensitivity: 'medium',
        patterns: []
      }
    },
    moderation: {
      enabled: false,
      severity: 'medium',
      blocked_words: [],
      layers: []
    },
    database: {
      enabled: false,
      type: 'sqlite',
      connection_string: ''
    }
  })

  const steps: WizardStep[] = [
    {
      id: 'welcome',
      title: 'Welcome to QT-1 Middleware',
      description: 'Let\'s set up your configuration in a few simple steps',
      icon: RocketLaunchIcon,
      fields: []
    },
    {
      id: 'server',
      title: 'Server Configuration',
      description: 'Configure basic server settings',
      icon: ServerStackIcon,
      fields: ['port', 'host', 'target_url']
    },
    {
      id: 'providers',
      title: 'AI Providers',
      description: 'Add at least one AI provider',
      icon: CloudIcon,
      fields: ['provider_name', 'api_key', 'base_url']
    },
    {
      id: 'security',
      title: 'Security Settings',
      description: 'Configure security and protection features',
      icon: ShieldCheckIcon,
      fields: ['enable_security', 'prompt_injection', 'rate_limiting']
    },
    {
      id: 'moderation',
      title: 'Content Moderation',
      description: 'Set up content moderation (optional)',
      icon: DocumentTextIcon,
      fields: ['enable_moderation', 'severity', 'blocked_words']
    },
    {
      id: 'database',
      title: 'Database Setup',
      description: 'Configure database for persistence (optional)',
      icon: CpuChipIcon,
      fields: ['enable_database', 'database_type', 'connection']
    },
    {
      id: 'review',
      title: 'Review Configuration',
      description: 'Review and confirm your settings',
      icon: CheckIcon,
      fields: []
    }
  ]

  const handleNext = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      onComplete(config)
    }
  }

  const handlePrevious = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1)
    }
  }

  const updateConfig = (section: string, field: string, value: any) => {
    setConfig((prev: any) => ({
      ...prev,
      [section]: {
        ...prev[section],
        [field]: value
      }
    }))
  }

  const renderStepContent = () => {
    const step = steps[currentStep]

    switch (step.id) {
      case 'welcome':
        return (
          <div className="text-center py-12">
            <RocketLaunchIcon className="h-24 w-24 mx-auto text-blue-500 mb-6" />
            <h2 className="text-2xl font-bold text-gray-900 mb-4">
              Welcome to QT-1 Middleware Configuration
            </h2>
            <p className="text-gray-600 max-w-2xl mx-auto mb-8">
              This wizard will guide you through the initial configuration of your QT-1 middleware.
              You can always modify these settings later from the configuration page.
            </p>
            <div className="text-sm text-gray-500">
              This process will take approximately 3-5 minutes
            </div>
          </div>
        )

      case 'server':
        return (
          <div className="space-y-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Server Port
              </label>
              <input
                type="number"
                value={config.server.port}
                onChange={(e) => updateConfig('server', 'port', parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-sm text-gray-500 mt-1">
                The port your middleware will listen on
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Host Address
              </label>
              <input
                type="text"
                value={config.server.host}
                onChange={(e) => updateConfig('server', 'host', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-sm text-gray-500 mt-1">
                Use 'localhost' for local development or '0.0.0.0' for all interfaces
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Target URL (Optional)
              </label>
              <input
                type="text"
                value={config.server.target_url || ''}
                onChange={(e) => updateConfig('server', 'target_url', e.target.value)}
                placeholder="https://api.example.com"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-sm text-gray-500 mt-1">
                Default target URL for proxying requests
              </p>
            </div>
          </div>
        )

      case 'providers':
        return (
          <div className="space-y-6">
            <div className="bg-blue-50 p-4 rounded-md">
              <p className="text-sm text-blue-700">
                Add your first AI provider. You can add more providers later.
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Provider Name
              </label>
              <select
                value={config.providers.name || 'openai'}
                onChange={(e) => updateConfig('providers', 'name', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
                <option value="bedrock">AWS Bedrock</option>
                <option value="vertex">Google Vertex AI</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                API Key
              </label>
              <input
                type="password"
                value={config.providers.api_key || ''}
                onChange={(e) => updateConfig('providers', 'api_key', e.target.value)}
                placeholder="sk-..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-sm text-gray-500 mt-1">
                Your provider API key (stored securely)
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Base URL (Optional)
              </label>
              <input
                type="text"
                value={config.providers.base_url || ''}
                onChange={(e) => updateConfig('providers', 'base_url', e.target.value)}
                placeholder="https://api.openai.com/v1"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-sm text-gray-500 mt-1">
                Leave empty to use the default URL
              </p>
            </div>
          </div>
        )

      case 'security':
        return (
          <div className="space-y-6">
            <div>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={config.security.prompt_injection.enabled}
                  onChange={(e) => setConfig({
                    ...config,
                    security: {
                      ...config.security,
                      prompt_injection: {
                        ...config.security.prompt_injection,
                        enabled: e.target.checked
                      }
                    }
                  })}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <span className="ml-2 text-sm font-medium text-gray-700">
                  Enable Prompt Injection Protection
                </span>
              </label>
              <p className="text-sm text-gray-500 mt-1 ml-6">
                Detect and block potential prompt injection attacks
              </p>
            </div>

            {config.security.prompt_injection.enabled && (
              <div className="ml-6">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Sensitivity Level
                </label>
                <select
                  value={config.security.prompt_injection.sensitivity}
                  onChange={(e) => setConfig({
                    ...config,
                    security: {
                      ...config.security,
                      prompt_injection: {
                        ...config.security.prompt_injection,
                        sensitivity: e.target.value
                      }
                    }
                  })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="low">Low - Minimal false positives</option>
                  <option value="medium">Medium - Balanced detection</option>
                  <option value="high">High - Maximum security</option>
                </select>
              </div>
            )}

            <div>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={config.security.rate_limiting?.enabled || false}
                  onChange={(e) => setConfig({
                    ...config,
                    security: {
                      ...config.security,
                      rate_limiting: {
                        enabled: e.target.checked,
                        requests_per_minute: 60
                      }
                    }
                  })}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <span className="ml-2 text-sm font-medium text-gray-700">
                  Enable Rate Limiting
                </span>
              </label>
              <p className="text-sm text-gray-500 mt-1 ml-6">
                Limit the number of requests per user
              </p>
            </div>
          </div>
        )

      case 'moderation':
        return (
          <div className="space-y-6">
            <div>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={config.moderation.enabled}
                  onChange={(e) => updateConfig('moderation', 'enabled', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <span className="ml-2 text-sm font-medium text-gray-700">
                  Enable Content Moderation
                </span>
              </label>
              <p className="text-sm text-gray-500 mt-1 ml-6">
                Filter inappropriate content and enforce content policies
              </p>
            </div>

            {config.moderation.enabled && (
              <>
                <div className="ml-6">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Moderation Severity
                  </label>
                  <select
                    value={config.moderation.severity}
                    onChange={(e) => updateConfig('moderation', 'severity', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="low">Low - Allow most content</option>
                    <option value="medium">Medium - Balanced filtering</option>
                    <option value="high">High - Strict filtering</option>
                  </select>
                </div>

                <div className="ml-6">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Blocked Words (Optional)
                  </label>
                  <textarea
                    value={config.moderation.blocked_words.join('\n')}
                    onChange={(e) => updateConfig('moderation', 'blocked_words', 
                      e.target.value.split('\n').filter(w => w.trim())
                    )}
                    placeholder="Enter words to block, one per line"
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </>
            )}
          </div>
        )

      case 'database':
        return (
          <div className="space-y-6">
            <div>
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={config.database.enabled}
                  onChange={(e) => updateConfig('database', 'enabled', e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <span className="ml-2 text-sm font-medium text-gray-700">
                  Enable Database
                </span>
              </label>
              <p className="text-sm text-gray-500 mt-1 ml-6">
                Store logs, metrics, and configuration history
              </p>
            </div>

            {config.database.enabled && (
              <>
                <div className="ml-6">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Database Type
                  </label>
                  <select
                    value={config.database.type}
                    onChange={(e) => updateConfig('database', 'type', e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="sqlite">SQLite (Recommended for development)</option>
                    <option value="postgres">PostgreSQL</option>
                    <option value="mysql">MySQL</option>
                  </select>
                </div>

                {config.database.type !== 'sqlite' && (
                  <div className="ml-6">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Connection String
                    </label>
                    <input
                      type="text"
                      value={config.database.connection_string}
                      onChange={(e) => updateConfig('database', 'connection_string', e.target.value)}
                      placeholder={
                        config.database.type === 'postgres'
                          ? 'postgres://user:pass@localhost/dbname'
                          : 'mysql://user:pass@localhost/dbname'
                      }
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                )}
              </>
            )}
          </div>
        )

      case 'review':
        return (
          <div className="space-y-6">
            <div className="bg-green-50 p-4 rounded-md">
              <p className="text-sm text-green-700">
                Great! Your configuration is ready. Review the settings below before completing the setup.
              </p>
            </div>

            <div className="space-y-4">
              <ConfigSummarySection
                title="Server"
                items={[
                  { label: 'Port', value: config.server.port },
                  { label: 'Host', value: config.server.host },
                  { label: 'Target URL', value: config.server.target_url || 'Not configured' }
                ]}
              />

              <ConfigSummarySection
                title="Provider"
                items={[
                  { label: 'Provider', value: config.providers.name || 'Not configured' },
                  { label: 'API Key', value: config.providers.api_key ? '••••••••' : 'Not set' }
                ]}
              />

              <ConfigSummarySection
                title="Security"
                items={[
                  { 
                    label: 'Prompt Injection Protection', 
                    value: config.security.prompt_injection.enabled ? 'Enabled' : 'Disabled' 
                  },
                  { 
                    label: 'Rate Limiting', 
                    value: config.security.rate_limiting?.enabled ? 'Enabled' : 'Disabled' 
                  }
                ]}
              />

              <ConfigSummarySection
                title="Moderation"
                items={[
                  { label: 'Status', value: config.moderation.enabled ? 'Enabled' : 'Disabled' },
                  { 
                    label: 'Severity', 
                    value: config.moderation.enabled ? config.moderation.severity : 'N/A' 
                  }
                ]}
              />

              <ConfigSummarySection
                title="Database"
                items={[
                  { label: 'Status', value: config.database.enabled ? 'Enabled' : 'Disabled' },
                  { 
                    label: 'Type', 
                    value: config.database.enabled ? config.database.type : 'N/A' 
                  }
                ]}
              />
            </div>
          </div>
        )

      default:
        return null
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full">
        {/* Progress */}
        <div className="px-8 pt-8">
          <div className="flex items-center justify-between mb-8">
            {steps.map((step, index) => {
              const Icon = step.icon
              return (
                <div
                  key={step.id}
                  className={`flex items-center ${
                    index < steps.length - 1 ? 'flex-1' : ''
                  }`}
                >
                  <div
                    className={`flex items-center justify-center w-10 h-10 rounded-full border-2 ${
                      index < currentStep
                        ? 'bg-blue-600 border-blue-600 text-white'
                        : index === currentStep
                        ? 'border-blue-600 text-blue-600'
                        : 'border-gray-300 text-gray-400'
                    }`}
                  >
                    {index < currentStep ? (
                      <CheckIcon className="h-5 w-5" />
                    ) : (
                      <Icon className="h-5 w-5" />
                    )}
                  </div>
                  {index < steps.length - 1 && (
                    <div
                      className={`flex-1 h-0.5 mx-2 ${
                        index < currentStep ? 'bg-blue-600' : 'bg-gray-300'
                      }`}
                    />
                  )}
                </div>
              )
            })}
          </div>
        </div>

        {/* Content */}
        <div className="px-8 pb-8">
          <div className="mb-8">
            <h2 className="text-2xl font-bold text-gray-900">
              {steps[currentStep].title}
            </h2>
            <p className="text-gray-600 mt-2">
              {steps[currentStep].description}
            </p>
          </div>

          <div className="min-h-[300px]">
            {renderStepContent()}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between mt-8">
            <button
              onClick={onSkip}
              className="text-sm text-gray-500 hover:text-gray-700"
            >
              Skip Setup
            </button>

            <div className="flex items-center space-x-3">
              <button
                onClick={handlePrevious}
                disabled={currentStep === 0}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <ChevronLeftIcon className="h-4 w-4 inline-block mr-1" />
                Previous
              </button>
              <button
                onClick={handleNext}
                className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
              >
                {currentStep === steps.length - 1 ? 'Complete Setup' : 'Next'}
                {currentStep < steps.length - 1 && (
                  <ChevronRightIcon className="h-4 w-4 inline-block ml-1" />
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

const ConfigSummarySection: React.FC<{
  title: string
  items: Array<{ label: string; value: any }>
}> = ({ title, items }) => {
  return (
    <div className="bg-gray-50 rounded-md p-4">
      <h4 className="text-sm font-medium text-gray-900 mb-2">{title}</h4>
      <div className="space-y-1">
        {items.map((item, index) => (
          <div key={index} className="flex justify-between text-sm">
            <span className="text-gray-600">{item.label}:</span>
            <span className="font-medium text-gray-900">{item.value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default ConfigWizard