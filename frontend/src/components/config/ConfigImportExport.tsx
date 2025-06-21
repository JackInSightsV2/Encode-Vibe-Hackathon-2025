import React, { useState, useRef } from 'react'
import { 
  ArrowUpTrayIcon,
  ArrowDownTrayIcon,
  DocumentIcon,
  DocumentTextIcon,
  CodeBracketIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  CloudArrowUpIcon,
  CloudArrowDownIcon,
  ShieldCheckIcon,
  ArrowPathIcon
} from '@heroicons/react/24/outline'

interface ExportOptions {
  format: 'json' | 'yaml' | 'toml' | 'env'
  includeSecrets: boolean
  includeDefaults: boolean
  minify: boolean
  encrypt: boolean
  password?: string
  sections?: string[]
}

interface ImportResult {
  success: boolean
  message: string
  warnings: string[]
  errors: string[]
  changes: {
    additions: number
    modifications: number
    deletions: number
  }
  preview?: any
}

interface Template {
  id: string
  name: string
  description: string
  format: string
  size: number
  created_at: string
  downloads: number
  rating: number
  tags: string[]
}

interface ConfigImportExportProps {
  currentConfig: any
  onImport: (config: any) => void
  onClose: () => void
}

const ConfigImportExport: React.FC<ConfigImportExportProps> = ({
  currentConfig,
  onImport,
  onClose
}) => {
  const [activeTab, setActiveTab] = useState<'export' | 'import' | 'templates'>('export')
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'json',
    includeSecrets: false,
    includeDefaults: true,
    minify: false,
    encrypt: false,
    sections: []
  })
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importResult, setImportResult] = useState<ImportResult | null>(null)
  const [validating, setValidating] = useState(false)
  const [templates, setTemplates] = useState<Template[]>([])
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleExport = async () => {
    try {
      const response = await fetch('/api/config/export', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          config: currentConfig,
          options: exportOptions
        })
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `config-export-${Date.now()}.${exportOptions.format}`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
      }
    } catch (error) {
      console.error('Failed to export configuration:', error)
    }
  }

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file) {
      setImportFile(file)
      validateImport(file)
    }
  }

  const validateImport = async (file: File) => {
    setValidating(true)
    setImportResult(null)

    const formData = new FormData()
    formData.append('file', file)

    try {
      const response = await fetch('/api/config/import/validate', {
        method: 'POST',
        body: formData
      })

      if (response.ok) {
        const result = await response.json()
        setImportResult(result.data)
      }
    } catch (error) {
      console.error('Failed to validate import:', error)
      setImportResult({
        success: false,
        message: 'Failed to validate file',
        warnings: [],
        errors: ['Invalid file format or corrupted file'],
        changes: { additions: 0, modifications: 0, deletions: 0 }
      })
    } finally {
      setValidating(false)
    }
  }

  const handleImport = async () => {
    if (!importFile || !importResult?.success) return

    const formData = new FormData()
    formData.append('file', importFile)

    try {
      const response = await fetch('/api/config/import', {
        method: 'POST',
        body: formData
      })

      if (response.ok) {
        const data = await response.json()
        onImport(data.data.config)
        onClose()
      }
    } catch (error) {
      console.error('Failed to import configuration:', error)
    }
  }

  const loadTemplates = async () => {
    try {
      const response = await fetch('/api/config/templates')
      if (response.ok) {
        const data = await response.json()
        setTemplates(data.data || [])
      }
    } catch (error) {
      console.error('Failed to load templates:', error)
    }
  }

  const applyTemplate = async (template: Template) => {
    try {
      const response = await fetch(`/api/config/templates/${template.id}`)
      if (response.ok) {
        const data = await response.json()
        onImport(data.data.config)
      }
    } catch (error) {
      console.error('Failed to apply template:', error)
    }
  }

  React.useEffect(() => {
    if (activeTab === 'templates') {
      loadTemplates()
    }
  }, [activeTab])

  const getFormatIcon = (format: string) => {
    switch (format) {
      case 'json': return <CodeBracketIcon className="h-5 w-5" />
      case 'yaml': return <DocumentTextIcon className="h-5 w-5" />
      case 'toml': return <DocumentIcon className="h-5 w-5" />
      case 'env': return <DocumentTextIcon className="h-5 w-5" />
      default: return <DocumentIcon className="h-5 w-5" />
    }
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              Import & Export Configuration
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Import, export, and apply configuration templates
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

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <nav className="flex space-x-8">
          {['export', 'import', 'templates'].map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab as any)}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {activeTab === 'export' && (
          <ExportTab
            options={exportOptions}
            setOptions={setExportOptions}
            onExport={handleExport}
            currentConfig={currentConfig}
          />
        )}

        {activeTab === 'import' && (
          <ImportTab
            importFile={importFile}
            importResult={importResult}
            validating={validating}
            fileInputRef={fileInputRef}
            onFileSelect={handleFileSelect}
            onImport={handleImport}
            onReset={() => {
              setImportFile(null)
              setImportResult(null)
              if (fileInputRef.current) {
                fileInputRef.current.value = ''
              }
            }}
          />
        )}

        {activeTab === 'templates' && (
          <TemplatesTab
            templates={templates}
            selectedTemplate={selectedTemplate}
            onSelect={setSelectedTemplate}
            onApply={applyTemplate}
          />
        )}
      </div>
    </div>
  )
}

// Sub-components
const ExportTab: React.FC<{
  options: ExportOptions
  setOptions: (options: ExportOptions) => void
  onExport: () => void
  currentConfig: any
}> = ({ options, setOptions, onExport, currentConfig }) => {
  const configSections = Object.keys(currentConfig || {})

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">
          Export Configuration
        </h3>

        <div className="space-y-6">
          {/* Format Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Export Format
            </label>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {(['json', 'yaml', 'toml', 'env'] as const).map((format) => (
                <button
                  key={format}
                  onClick={() => setOptions({ ...options, format })}
                  className={`p-3 border rounded-lg text-center transition-colors ${
                    options.format === format
                      ? 'border-blue-500 bg-blue-50 text-blue-700'
                      : 'border-gray-300 hover:border-gray-400'
                  }`}
                >
                  <div className="flex flex-col items-center">
                    {getFormatIcon(format)}
                    <span className="mt-1 text-sm font-medium uppercase">{format}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Options */}
          <div className="space-y-3">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={options.includeSecrets}
                onChange={(e) => setOptions({ ...options, includeSecrets: e.target.checked })}
                className="h-4 w-4 text-blue-600 rounded border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Include sensitive values</span>
              <ShieldCheckIcon className="h-4 w-4 ml-2 text-gray-400" />
            </label>

            <label className="flex items-center">
              <input
                type="checkbox"
                checked={options.includeDefaults}
                onChange={(e) => setOptions({ ...options, includeDefaults: e.target.checked })}
                className="h-4 w-4 text-blue-600 rounded border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Include default values</span>
            </label>

            <label className="flex items-center">
              <input
                type="checkbox"
                checked={options.minify}
                onChange={(e) => setOptions({ ...options, minify: e.target.checked })}
                className="h-4 w-4 text-blue-600 rounded border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Minify output</span>
            </label>

            <label className="flex items-center">
              <input
                type="checkbox"
                checked={options.encrypt}
                onChange={(e) => setOptions({ ...options, encrypt: e.target.checked })}
                className="h-4 w-4 text-blue-600 rounded border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Encrypt export</span>
            </label>

            {options.encrypt && (
              <div className="ml-6">
                <input
                  type="password"
                  value={options.password || ''}
                  onChange={(e) => setOptions({ ...options, password: e.target.value })}
                  placeholder="Enter encryption password"
                  className="px-3 py-2 border border-gray-300 rounded-md text-sm w-full"
                />
              </div>
            )}
          </div>

          {/* Section Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Export Sections
            </label>
            <div className="space-y-2 max-h-48 overflow-y-auto border rounded-md p-3">
              <label className="flex items-center text-sm">
                <input
                  type="checkbox"
                  checked={options.sections?.length === 0}
                  onChange={(e) => setOptions({ 
                    ...options, 
                    sections: e.target.checked ? [] : configSections 
                  })}
                  className="h-4 w-4 text-blue-600 rounded border-gray-300"
                />
                <span className="ml-2 font-medium">All sections</span>
              </label>
              {configSections.map((section) => (
                <label key={section} className="flex items-center text-sm ml-4">
                  <input
                    type="checkbox"
                    checked={options.sections?.length === 0 || options.sections?.includes(section)}
                    onChange={(e) => {
                      const sections = options.sections || []
                      setOptions({
                        ...options,
                        sections: e.target.checked
                          ? [...sections, section]
                          : sections.filter(s => s !== section)
                      })
                    }}
                    className="h-4 w-4 text-blue-600 rounded border-gray-300"
                  />
                  <span className="ml-2">{section}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Preview */}
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-2">Export Preview</h4>
            <div className="bg-gray-50 rounded-md p-3 text-xs text-gray-600">
              <p>Format: <span className="font-medium">{options.format.toUpperCase()}</span></p>
              <p>Size estimate: ~{estimateSize(currentConfig, options)} KB</p>
              <p>Sections: {options.sections?.length === 0 ? 'All' : options.sections?.join(', ')}</p>
              {options.encrypt && <p className="text-orange-600">⚠️ File will be encrypted</p>}
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end space-x-3 pt-4 border-t">
            <button
              onClick={onExport}
              disabled={options.encrypt && !options.password}
              className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              <ArrowDownTrayIcon className="h-4 w-4 inline-block mr-2" />
              Export Configuration
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

const ImportTab: React.FC<{
  importFile: File | null
  importResult: ImportResult | null
  validating: boolean
  fileInputRef: React.RefObject<HTMLInputElement>
  onFileSelect: (e: React.ChangeEvent<HTMLInputElement>) => void
  onImport: () => void
  onReset: () => void
}> = ({ importFile, importResult, validating, fileInputRef, onFileSelect, onImport, onReset }) => {
  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-6">
          Import Configuration
        </h3>

        {/* File Upload */}
        {!importFile && (
          <div className="border-2 border-dashed border-gray-300 rounded-lg p-12">
            <div className="text-center">
              <CloudArrowUpIcon className="mx-auto h-12 w-12 text-gray-400" />
              <p className="mt-2 text-sm font-medium text-gray-900">
                Upload a configuration file
              </p>
              <p className="mt-1 text-xs text-gray-500">
                JSON, YAML, TOML, or ENV format
              </p>
              <div className="mt-6">
                <label className="cursor-pointer">
                  <span className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700">
                    Choose File
                  </span>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".json,.yaml,.yml,.toml,.env"
                    onChange={onFileSelect}
                    className="hidden"
                  />
                </label>
              </div>
            </div>
          </div>
        )}

        {/* File Selected */}
        {importFile && (
          <div className="space-y-6">
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <DocumentIcon className="h-8 w-8 text-gray-400 mr-3" />
                  <div>
                    <p className="text-sm font-medium text-gray-900">{importFile.name}</p>
                    <p className="text-xs text-gray-500">
                      {(importFile.size / 1024).toFixed(1)} KB
                    </p>
                  </div>
                </div>
                <button
                  onClick={onReset}
                  className="text-sm text-red-600 hover:text-red-800"
                >
                  Remove
                </button>
              </div>
            </div>

            {/* Validation Status */}
            {validating && (
              <div className="flex items-center justify-center py-8">
                <div className="text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                  <p className="mt-3 text-sm text-gray-600">Validating configuration...</p>
                </div>
              </div>
            )}

            {/* Validation Result */}
            {importResult && !validating && (
              <div className="space-y-4">
                {/* Status */}
                <div className={`p-4 rounded-lg ${
                  importResult.success 
                    ? 'bg-green-50 border border-green-200' 
                    : 'bg-red-50 border border-red-200'
                }`}>
                  <div className="flex items-start">
                    {importResult.success ? (
                      <CheckCircleIcon className="h-5 w-5 text-green-600 mt-0.5 mr-3" />
                    ) : (
                      <XCircleIcon className="h-5 w-5 text-red-600 mt-0.5 mr-3" />
                    )}
                    <div className="flex-1">
                      <p className={`text-sm font-medium ${
                        importResult.success ? 'text-green-800' : 'text-red-800'
                      }`}>
                        {importResult.message}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Changes Summary */}
                {importResult.success && (
                  <div className="bg-blue-50 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-blue-900 mb-2">
                      Changes to be Applied
                    </h4>
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div>
                        <p className="text-lg font-bold text-green-600">
                          +{importResult.changes.additions}
                        </p>
                        <p className="text-xs text-gray-600">Additions</p>
                      </div>
                      <div>
                        <p className="text-lg font-bold text-yellow-600">
                          ~{importResult.changes.modifications}
                        </p>
                        <p className="text-xs text-gray-600">Modifications</p>
                      </div>
                      <div>
                        <p className="text-lg font-bold text-red-600">
                          -{importResult.changes.deletions}
                        </p>
                        <p className="text-xs text-gray-600">Deletions</p>
                      </div>
                    </div>
                  </div>
                )}

                {/* Warnings */}
                {importResult.warnings.length > 0 && (
                  <div className="bg-yellow-50 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-yellow-900 mb-2">
                      Warnings ({importResult.warnings.length})
                    </h4>
                    <ul className="space-y-1">
                      {importResult.warnings.map((warning, index) => (
                        <li key={index} className="text-sm text-yellow-700 flex items-start">
                          <ExclamationTriangleIcon className="h-4 w-4 mr-2 mt-0.5 flex-shrink-0" />
                          {warning}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Errors */}
                {importResult.errors.length > 0 && (
                  <div className="bg-red-50 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-red-900 mb-2">
                      Errors ({importResult.errors.length})
                    </h4>
                    <ul className="space-y-1">
                      {importResult.errors.map((error, index) => (
                        <li key={index} className="text-sm text-red-700 flex items-start">
                          <XCircleIcon className="h-4 w-4 mr-2 mt-0.5 flex-shrink-0" />
                          {error}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Actions */}
                <div className="flex justify-end space-x-3 pt-4 border-t">
                  <button
                    onClick={onReset}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={onImport}
                    disabled={!importResult.success}
                    className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <ArrowUpTrayIcon className="h-4 w-4 inline-block mr-2" />
                    Import Configuration
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

const TemplatesTab: React.FC<{
  templates: Template[]
  selectedTemplate: Template | null
  onSelect: (template: Template) => void
  onApply: (template: Template) => void
}> = ({ templates, selectedTemplate, onSelect, onApply }) => {
  const categories = ['All', 'Basic', 'Advanced', 'Security', 'Performance', 'Development']
  const [selectedCategory, setSelectedCategory] = useState('All')

  const filteredTemplates = templates.filter(t => 
    selectedCategory === 'All' || t.tags.includes(selectedCategory.toLowerCase())
  )

  return (
    <div className="p-6">
      {/* Categories */}
      <div className="mb-6">
        <div className="flex items-center space-x-2 overflow-x-auto">
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setSelectedCategory(category)}
              className={`px-4 py-2 text-sm font-medium rounded-full whitespace-nowrap ${
                selectedCategory === category
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      </div>

      {/* Templates Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredTemplates.map((template) => (
          <div
            key={template.id}
            onClick={() => onSelect(template)}
            className={`border rounded-lg p-4 cursor-pointer transition-all hover:shadow-md ${
              selectedTemplate?.id === template.id
                ? 'border-blue-500 bg-blue-50'
                : 'border-gray-300 bg-white'
            }`}
          >
            <div className="flex items-start justify-between mb-2">
              <h4 className="text-sm font-medium text-gray-900">{template.name}</h4>
              <div className="flex items-center text-xs text-gray-500">
                <span className="text-yellow-500">★</span>
                <span className="ml-1">{template.rating.toFixed(1)}</span>
              </div>
            </div>
            
            <p className="text-sm text-gray-600 mb-3 line-clamp-2">
              {template.description}
            </p>

            <div className="flex items-center justify-between text-xs text-gray-500">
              <span>{template.format.toUpperCase()}</span>
              <span>{template.downloads} downloads</span>
            </div>

            <div className="mt-3 flex flex-wrap gap-1">
              {template.tags.map((tag) => (
                <span
                  key={tag}
                  className="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded-full"
                >
                  {tag}
                </span>
              ))}
            </div>

            {selectedTemplate?.id === template.id && (
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onApply(template)
                }}
                className="mt-4 w-full px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
              >
                Apply Template
              </button>
            )}
          </div>
        ))}
      </div>

      {filteredTemplates.length === 0 && (
        <div className="text-center py-12">
          <DocumentIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
          <p className="text-gray-500">No templates found</p>
        </div>
      )}
    </div>
  )
}

// Helper functions
function getFormatIcon(format: string) {
  switch (format) {
    case 'json': return <CodeBracketIcon className="h-5 w-5" />
    case 'yaml': return <DocumentTextIcon className="h-5 w-5" />
    case 'toml': return <DocumentIcon className="h-5 w-5" />
    case 'env': return <DocumentTextIcon className="h-5 w-5" />
    default: return <DocumentIcon className="h-5 w-5" />
  }
}

function estimateSize(config: any, options: ExportOptions): number {
  // Simple size estimation
  let str = JSON.stringify(config)
  if (!options.includeDefaults) str = str.replace(/null|""/g, '')
  if (options.minify) str = str.replace(/\s+/g, '')
  return Math.round(str.length / 1024)
}

export default ConfigImportExport