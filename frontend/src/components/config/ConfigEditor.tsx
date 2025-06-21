import React, { useState, useEffect, useRef } from 'react'
// @ts-ignore
import { Controlled as CodeMirror } from 'react-codemirror2'
import 'codemirror/lib/codemirror.css'
import 'codemirror/theme/material.css'
import 'codemirror/mode/javascript/javascript'
import 'codemirror/addon/lint/lint'
import 'codemirror/addon/lint/json-lint'
import 'codemirror/addon/edit/matchbrackets'
import 'codemirror/addon/edit/closebrackets'
import 'codemirror/addon/fold/foldcode'
import 'codemirror/addon/fold/foldgutter'
import 'codemirror/addon/fold/brace-fold'
import 'codemirror/addon/fold/foldgutter.css'
import { ExclamationCircleIcon } from '@heroicons/react/24/outline'

interface ConfigEditorProps {
  value: any
  schema: any
  path: string[]
  onChange: (value: any, path: string[]) => void
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

interface FieldProps {
  name: string
  value: any
  schema: any
  path: string[]
  onChange: (value: any, path: string[]) => void
  error?: ValidationError
}

const ConfigEditor: React.FC<ConfigEditorProps> = ({
  value,
  schema,
  path = [],
  onChange,
  errors = [],
  mode = 'form'
}) => {
  const [jsonValue, setJsonValue] = useState(JSON.stringify(value, null, 2))
  const [jsonError, setJsonError] = useState<string | null>(null)
  const editorRef = useRef<any>(null)

  useEffect(() => {
    setJsonValue(JSON.stringify(value, null, 2))
  }, [value])

  const handleJsonChange = (newValue: string) => {
    setJsonValue(newValue)
    try {
      const parsed = JSON.parse(newValue)
      setJsonError(null)
      onChange(parsed, path)
    } catch (e: any) {
      setJsonError(e.message)
    }
  }

  const getFieldError = (fieldPath: string[]): ValidationError | undefined => {
    const fullPath = [...path, ...fieldPath].join('.')
    return errors.find(e => e.field === fullPath)
  }

  const renderField = (fieldProps: FieldProps) => {
    const { name, value, schema, path, onChange, error } = fieldProps
    const fieldId = [...path, name].join('.')

    // Determine field type
    const fieldType = schema.type || (typeof value)

    switch (fieldType) {
      case 'string':
        if (schema.enum && schema.enum.length > 0) {
          return (
            <div key={fieldId}>
              <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700 mb-1">
                {schema.title || name}
                {schema.required && <span className="text-red-500 ml-1">*</span>}
              </label>
              <select
                id={fieldId}
                value={value || ''}
                onChange={(e) => onChange(e.target.value, [...path, name])}
                className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 ${
                  error 
                    ? 'border-red-300 focus:ring-red-500' 
                    : 'border-gray-300 focus:ring-blue-500'
                }`}
              >
                <option value="">Select {schema.title || name}</option>
                {schema.enum.map((option: any) => (
                  <option key={option} value={option}>
                    {option}
                  </option>
                ))}
              </select>
              {renderFieldHelp(schema, error)}
            </div>
          )
        }

        if (schema.ui_component === 'password' || schema.sensitive) {
          return (
            <div key={fieldId}>
              <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700 mb-1">
                {schema.title || name}
                {schema.required && <span className="text-red-500 ml-1">*</span>}
              </label>
              <input
                id={fieldId}
                type="password"
                value={value || ''}
                onChange={(e) => onChange(e.target.value, [...path, name])}
                placeholder={schema.example || schema.description}
                className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 ${
                  error 
                    ? 'border-red-300 focus:ring-red-500' 
                    : 'border-gray-300 focus:ring-blue-500'
                }`}
              />
              {renderFieldHelp(schema, error)}
            </div>
          )
        }

        if (schema.format === 'textarea' || (value && value.length > 50)) {
          return (
            <div key={fieldId}>
              <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700 mb-1">
                {schema.title || name}
                {schema.required && <span className="text-red-500 ml-1">*</span>}
              </label>
              <textarea
                id={fieldId}
                value={value || ''}
                onChange={(e) => onChange(e.target.value, [...path, name])}
                placeholder={schema.example || schema.description}
                rows={3}
                className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 ${
                  error 
                    ? 'border-red-300 focus:ring-red-500' 
                    : 'border-gray-300 focus:ring-blue-500'
                }`}
              />
              {renderFieldHelp(schema, error)}
            </div>
          )
        }

        return (
          <div key={fieldId}>
            <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700 mb-1">
              {schema.title || name}
              {schema.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <input
              id={fieldId}
              type="text"
              value={value || ''}
              onChange={(e) => onChange(e.target.value, [...path, name])}
              placeholder={schema.example || schema.description}
              className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 ${
                error 
                  ? 'border-red-300 focus:ring-red-500' 
                  : 'border-gray-300 focus:ring-blue-500'
              }`}
            />
            {renderFieldHelp(schema, error)}
          </div>
        )

      case 'number':
        return (
          <div key={fieldId}>
            <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700 mb-1">
              {schema.title || name}
              {schema.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <input
              id={fieldId}
              type="number"
              value={value || 0}
              onChange={(e) => onChange(parseFloat(e.target.value) || 0, [...path, name])}
              min={schema.min_value}
              max={schema.max_value}
              className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 ${
                error 
                  ? 'border-red-300 focus:ring-red-500' 
                  : 'border-gray-300 focus:ring-blue-500'
              }`}
            />
            {renderFieldHelp(schema, error)}
          </div>
        )

      case 'boolean':
        return (
          <div key={fieldId} className="flex items-center">
            <input
              id={fieldId}
              type="checkbox"
              checked={value || false}
              onChange={(e) => onChange(e.target.checked, [...path, name])}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            />
            <label htmlFor={fieldId} className="ml-2 block text-sm text-gray-700">
              {schema.title || name}
              {schema.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            {error && (
              <div className="ml-4 text-sm text-red-600">{error.message}</div>
            )}
          </div>
        )

      case 'array':
        return (
          <div key={fieldId}>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {schema.title || name}
              {schema.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <ArrayField
              value={value || []}
              schema={schema}
              path={[...path, name]}
              onChange={onChange}
              error={error}
            />
            {renderFieldHelp(schema, error)}
          </div>
        )

      case 'object':
        return (
          <div key={fieldId} className="space-y-4">
            <h4 className="text-sm font-medium text-gray-900">
              {schema.title || name}
              {schema.required && <span className="text-red-500 ml-1">*</span>}
            </h4>
            {schema.description && (
              <p className="text-sm text-gray-500">{schema.description}</p>
            )}
            <div className="ml-4 space-y-4 border-l-2 border-gray-200 pl-4">
              {renderObjectFields(value || {}, schema.properties || {}, [...path, name])}
            </div>
          </div>
        )

      case 'duration':
        return (
          <div key={fieldId}>
            <label htmlFor={fieldId} className="block text-sm font-medium text-gray-700 mb-1">
              {schema.title || name}
              {schema.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <input
              id={fieldId}
              type="text"
              value={value || ''}
              onChange={(e) => onChange(e.target.value, [...path, name])}
              placeholder="e.g., 30s, 5m, 1h"
              pattern="^\d+[smh]$"
              className={`w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 ${
                error 
                  ? 'border-red-300 focus:ring-red-500' 
                  : 'border-gray-300 focus:ring-blue-500'
              }`}
            />
            {renderFieldHelp(schema, error)}
          </div>
        )

      default:
        return null
    }
  }

  const renderFieldHelp = (schema: any, error?: ValidationError) => {
    return (
      <div className="mt-1">
        {error ? (
          <div className="flex items-start text-sm text-red-600">
            <ExclamationCircleIcon className="h-4 w-4 mr-1 mt-0.5 flex-shrink-0" />
            <div>
              <p>{error.message}</p>
              {error.suggestion && (
                <p className="text-xs mt-1">{error.suggestion}</p>
              )}
            </div>
          </div>
        ) : schema.description ? (
          <p className="text-sm text-gray-500">{schema.description}</p>
        ) : null}
      </div>
    )
  }

  const renderObjectFields = (obj: any, properties: any, currentPath: string[]) => {
    if (!properties || Object.keys(properties).length === 0) {
      return <p className="text-sm text-gray-500 italic">No properties defined</p>
    }

    return Object.entries(properties).map(([key, propSchema]: [string, any]) => {
      const fieldError = getFieldError([...currentPath, key])
      return renderField({
        name: key,
        value: obj[key],
        schema: propSchema,
        path: currentPath,
        onChange,
        error: fieldError
      })
    })
  }

  const renderFormView = () => {
    if (!schema.properties || Object.keys(schema.properties).length === 0) {
      return (
        <div className="text-center py-8 text-gray-500">
          <p>No configuration properties available for this section.</p>
        </div>
      )
    }

    return (
      <div className="space-y-6">
        {renderObjectFields(value, schema.properties, path)}
      </div>
    )
  }

  const renderJsonView = () => {
    return (
      <div className="h-full">
        {jsonError && (
          <div className="mb-2 p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            JSON Error: {jsonError}
          </div>
        )}
        <div className="border rounded-md overflow-hidden h-[calc(100%-3rem)]">
          <CodeMirror
            value={jsonValue}
            options={{
              mode: 'application/json',
              theme: 'material',
              lineNumbers: true,
              lineWrapping: true,
              matchBrackets: true,
              autoCloseBrackets: true,
              foldGutter: true,
              gutters: ['CodeMirror-linenumbers', 'CodeMirror-foldgutter'],
              extraKeys: {
                'Ctrl-Space': 'autocomplete'
              }
            }}
            onBeforeChange={(_editor, _data, value) => {
              handleJsonChange(value)
            }}
            editorDidMount={(editor) => {
              editorRef.current = editor
              editor.setSize(null, '100%')
            }}
          />
        </div>
      </div>
    )
  }

  const renderSplitView = () => {
    return (
      <div className="grid grid-cols-2 gap-4 h-full">
        <div className="overflow-y-auto pr-2">
          <h3 className="text-sm font-medium text-gray-700 mb-4">Form View</h3>
          {renderFormView()}
        </div>
        <div className="overflow-hidden">
          <h3 className="text-sm font-medium text-gray-700 mb-4">JSON View</h3>
          {renderJsonView()}
        </div>
      </div>
    )
  }

  switch (mode) {
    case 'json':
      return renderJsonView()
    case 'split':
      return renderSplitView()
    case 'form':
    default:
      return renderFormView()
  }
}

// ArrayField component for handling array inputs
const ArrayField: React.FC<{
  value: any[]
  schema: any
  path: string[]
  onChange: (value: any, path: string[]) => void
  error?: ValidationError
}> = ({ value, path, onChange }) => {
  const handleAdd = () => {
    const newValue = [...value, '']
    onChange(newValue, path)
  }

  const handleRemove = (index: number) => {
    const newValue = value.filter((_, i) => i !== index)
    onChange(newValue, path)
  }

  const handleItemChange = (index: number, itemValue: any) => {
    const newValue = [...value]
    newValue[index] = itemValue
    onChange(newValue, path)
  }

  return (
    <div className="space-y-2">
      {value.map((item, index) => (
        <div key={index} className="flex items-center space-x-2">
          <input
            type="text"
            value={item}
            onChange={(e) => handleItemChange(index, e.target.value)}
            className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            onClick={() => handleRemove(index)}
            className="p-2 text-red-600 hover:text-red-800"
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      ))}
      <button
        onClick={handleAdd}
        className="inline-flex items-center px-3 py-1 text-sm text-blue-600 hover:text-blue-800"
      >
        <svg className="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        Add Item
      </button>
    </div>
  )
}

export default ConfigEditor