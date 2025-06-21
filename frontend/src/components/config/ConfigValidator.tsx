import React from 'react'
import { 
  CheckCircleIcon, 
  ExclamationTriangleIcon, 
  ExclamationCircleIcon,
  InformationCircleIcon 
} from '@heroicons/react/24/outline'

interface ValidationResult {
  valid: boolean
  errors: ValidationError[]
  warnings: ValidationWarning[]
  summary: string
  validated_at?: string
  duration?: string
}

interface ValidationError {
  field: string
  value?: any
  message: string
  code: string
  severity: string
  suggestion?: string
}

interface ValidationWarning {
  field: string
  value?: any
  message: string
  code: string
  suggestion?: string
}

interface ConfigValidatorProps {
  validationResult: ValidationResult | null
  isValidating: boolean
  compact?: boolean
  onFixError?: (error: ValidationError) => void
}

const ConfigValidator: React.FC<ConfigValidatorProps> = ({
  validationResult,
  isValidating,
  compact = false,
  onFixError
}) => {
  if (isValidating) {
    return (
      <div className={`${compact ? 'p-2' : 'p-4'} bg-blue-50 rounded-md`}>
        <div className="flex items-center">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 mr-2"></div>
          <span className="text-sm text-blue-700">Validating configuration...</span>
        </div>
      </div>
    )
  }

  if (!validationResult) {
    return null
  }

  const { valid, errors = [], warnings = [], summary } = validationResult

  if (compact) {
    return (
      <div className="space-y-2">
        <div className={`flex items-center justify-between p-2 rounded-md ${
          valid ? 'bg-green-50' : 'bg-red-50'
        }`}>
          <div className="flex items-center">
            {valid ? (
              <CheckCircleIcon className="h-4 w-4 text-green-600 mr-2" />
            ) : (
              <ExclamationCircleIcon className="h-4 w-4 text-red-600 mr-2" />
            )}
            <span className={`text-xs font-medium ${
              valid ? 'text-green-700' : 'text-red-700'
            }`}>
              {valid ? 'Valid' : 'Invalid'}
            </span>
          </div>
          <div className="text-xs text-gray-500">
            {errors.length > 0 && <span className="text-red-600">{errors.length} errors</span>}
            {errors.length > 0 && warnings.length > 0 && <span className="mx-1">·</span>}
            {warnings.length > 0 && <span className="text-yellow-600">{warnings.length} warnings</span>}
          </div>
        </div>
        
        {errors.length > 0 && (
          <div className="text-xs space-y-1">
            <div className="font-medium text-gray-700">Top issues:</div>
            {errors.slice(0, 3).map((error, index) => (
              <div key={index} className="text-red-600 truncate">
                • {error.field}: {error.message}
              </div>
            ))}
            {errors.length > 3 && (
              <div className="text-gray-500">...and {errors.length - 3} more</div>
            )}
          </div>
        )}
      </div>
    )
  }

  // Full view
  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className={`p-4 rounded-md border ${
        valid 
          ? 'bg-green-50 border-green-200' 
          : 'bg-red-50 border-red-200'
      }`}>
        <div className="flex items-start">
          {valid ? (
            <CheckCircleIcon className="h-5 w-5 text-green-600 mr-3 mt-0.5" />
          ) : (
            <ExclamationCircleIcon className="h-5 w-5 text-red-600 mr-3 mt-0.5" />
          )}
          <div className="flex-1">
            <h3 className={`text-sm font-medium ${
              valid ? 'text-green-800' : 'text-red-800'
            }`}>
              {summary || (valid ? 'Configuration is valid' : 'Configuration has errors')}
            </h3>
            <div className="mt-1 text-sm text-gray-600">
              {validationResult.validated_at && (
                <span>
                  Validated at {new Date(validationResult.validated_at).toLocaleTimeString()}
                </span>
              )}
              {validationResult.duration && (
                <span className="ml-2">({validationResult.duration})</span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Errors */}
      {errors.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-900 flex items-center">
            <ExclamationCircleIcon className="h-4 w-4 text-red-500 mr-2" />
            Errors ({errors.length})
          </h4>
          <div className="space-y-2">
            {errors.map((error, index) => (
              <ErrorItem key={index} error={error} onFix={onFixError} />
            ))}
          </div>
        </div>
      )}

      {/* Warnings */}
      {warnings.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-900 flex items-center">
            <ExclamationTriangleIcon className="h-4 w-4 text-yellow-500 mr-2" />
            Warnings ({warnings.length})
          </h4>
          <div className="space-y-2">
            {warnings.map((warning, index) => (
              <WarningItem key={index} warning={warning} />
            ))}
          </div>
        </div>
      )}

      {/* Help */}
      {!valid && (
        <div className="p-3 bg-blue-50 rounded-md">
          <div className="flex">
            <InformationCircleIcon className="h-5 w-5 text-blue-400 mr-2" />
            <div className="text-sm text-blue-700">
              <p className="font-medium">Need help?</p>
              <p className="mt-1">
                Fix the errors above to save your configuration. Hover over field labels for more information.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const ErrorItem: React.FC<{
  error: ValidationError
  onFix?: (error: ValidationError) => void
}> = ({ error, onFix }) => {
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-100 border-red-300 text-red-800'
      case 'error':
        return 'bg-red-50 border-red-200 text-red-700'
      default:
        return 'bg-orange-50 border-orange-200 text-orange-700'
    }
  }

  const getSeverityIcon = (severity: string) => {
    return severity === 'critical' 
      ? <ExclamationCircleIcon className="h-4 w-4" />
      : <ExclamationTriangleIcon className="h-4 w-4" />
  }

  return (
    <div className={`p-3 rounded-md border ${getSeverityColor(error.severity)}`}>
      <div className="flex items-start">
        <div className="flex-shrink-0 mr-2 mt-0.5">
          {getSeverityIcon(error.severity)}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <p className="text-sm font-medium">
                {error.field}
              </p>
              <p className="text-sm mt-1">
                {error.message}
              </p>
              {error.suggestion && (
                <p className="text-xs mt-1 opacity-75">
                  💡 {error.suggestion}
                </p>
              )}
            </div>
            {onFix && (
              <button
                onClick={() => onFix(error)}
                className="ml-3 text-xs font-medium text-blue-600 hover:text-blue-800"
              >
                Quick Fix
              </button>
            )}
          </div>
          <div className="mt-2 flex items-center space-x-4 text-xs">
            <span className="font-mono bg-gray-100 px-1 rounded">
              {error.code}
            </span>
            {error.value !== undefined && (
              <span className="text-gray-600">
                Current: <code className="bg-gray-100 px-1 rounded">
                  {JSON.stringify(error.value)}
                </code>
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

const WarningItem: React.FC<{
  warning: ValidationWarning
}> = ({ warning }) => {
  return (
    <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-md">
      <div className="flex items-start">
        <ExclamationTriangleIcon className="h-4 w-4 text-yellow-600 mr-2 mt-0.5" />
        <div className="flex-1">
          <p className="text-sm font-medium text-yellow-800">
            {warning.field}
          </p>
          <p className="text-sm text-yellow-700 mt-1">
            {warning.message}
          </p>
          {warning.suggestion && (
            <p className="text-xs text-yellow-600 mt-1">
              💡 {warning.suggestion}
            </p>
          )}
          <div className="mt-2 text-xs">
            <span className="font-mono text-yellow-700 bg-yellow-100 px-1 rounded">
              {warning.code}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

export default ConfigValidator