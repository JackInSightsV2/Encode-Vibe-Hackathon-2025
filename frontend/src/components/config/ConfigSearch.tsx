import React, { useState, useEffect, useRef } from 'react'
import { 
  MagnifyingGlassIcon,
  DocumentTextIcon,
  QuestionMarkCircleIcon,
  BookOpenIcon,
  CodeBracketIcon,
  CommandLineIcon,
  LightBulbIcon,
  ArrowRightIcon,
  TagIcon,
  ClockIcon,
  SparklesIcon,
  AcademicCapIcon
} from '@heroicons/react/24/outline'

interface SearchResult {
  type: 'config' | 'documentation' | 'example' | 'api' | 'tutorial'
  title: string
  description: string
  path?: string
  url?: string
  relevance: number
  tags: string[]
  snippet?: string
  context?: string
}

interface ConfigField {
  path: string
  type: string
  title: string
  description: string
  required: boolean
  default?: any
  example?: any
  validations?: string[]
  related_fields?: string[]
  documentation_url?: string
}

interface SearchSuggestion {
  text: string
  category: string
  icon: React.ComponentType<{ className?: string }>
}

interface ConfigSearchProps {
  schema?: any
  onNavigate?: (path: string) => void
  onClose: () => void
}

const ConfigSearch: React.FC<ConfigSearchProps> = ({
  schema,
  onNavigate,
  onClose
}) => {
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<SearchResult[]>([])
  const [suggestions, setSuggestions] = useState<SearchSuggestion[]>([])
  const [recentSearches, setRecentSearches] = useState<string[]>([])
  const [selectedResult, setSelectedResult] = useState<SearchResult | null>(null)
  const [selectedField, setSelectedField] = useState<ConfigField | null>(null)
  const [loading, setLoading] = useState(false)
  const [showHelp, setShowHelp] = useState(false)
  const searchInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    searchInputRef.current?.focus()
    loadRecentSearches()
  }, [])

  useEffect(() => {
    if (searchQuery.length > 2) {
      performSearch()
    } else if (searchQuery.length === 0) {
      loadSuggestions()
    }
  }, [searchQuery])

  const loadRecentSearches = () => {
    // Load from localStorage
    const recent = localStorage.getItem('config_recent_searches')
    if (recent) {
      setRecentSearches(JSON.parse(recent).slice(0, 5))
    }
  }

  const saveRecentSearch = (query: string) => {
    const updated = [query, ...recentSearches.filter(s => s !== query)].slice(0, 5)
    setRecentSearches(updated)
    localStorage.setItem('config_recent_searches', JSON.stringify(updated))
  }

  const loadSuggestions = async () => {
    // Load popular searches and suggestions
    setSuggestions([
      { text: 'How to configure providers?', category: 'Tutorial', icon: AcademicCapIcon },
      { text: 'Security best practices', category: 'Documentation', icon: BookOpenIcon },
      { text: 'Rate limiting configuration', category: 'Example', icon: CodeBracketIcon },
      { text: 'Environment variables', category: 'Configuration', icon: CommandLineIcon },
      { text: 'API authentication setup', category: 'Tutorial', icon: LightBulbIcon }
    ])
  }

  const performSearch = async () => {
    setLoading(true)
    saveRecentSearch(searchQuery)

    try {
      const response = await fetch('/api/config/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          query: searchQuery,
          include_docs: true,
          include_examples: true
        })
      })

      if (response.ok) {
        const data = await response.json()
        setSearchResults(data.data || [])
      } else {
        // Fallback to local search
        performLocalSearch()
      }
    } catch (error) {
      console.error('Search failed:', error)
      performLocalSearch()
    } finally {
      setLoading(false)
    }
  }

  const performLocalSearch = () => {
    // Search through schema if available
    if (!schema) return

    const results: SearchResult[] = []
    const query = searchQuery.toLowerCase()

    // Search configuration fields
    searchSchemaRecursive(schema, '', (field) => {
      const titleMatch = field.title?.toLowerCase().includes(query)
      const descMatch = field.description?.toLowerCase().includes(query)
      const pathMatch = field.path.toLowerCase().includes(query)

      if (titleMatch || descMatch || pathMatch) {
        results.push({
          type: 'config',
          title: field.title || field.path,
          description: field.description || 'Configuration field',
          path: field.path,
          relevance: titleMatch ? 1 : descMatch ? 0.8 : 0.6,
          tags: ['configuration', ...field.path.split('.').slice(0, -1)],
          snippet: field.example ? `Example: ${JSON.stringify(field.example)}` : undefined
        })
      }
    })

    // Add documentation results (mock data)
    if ('security'.includes(query) || 'protection'.includes(query)) {
      results.push({
        type: 'documentation',
        title: 'Security Configuration Guide',
        description: 'Learn how to configure security features including rate limiting, IP protection, and authentication',
        url: '/docs/security',
        relevance: 0.9,
        tags: ['security', 'protection', 'guide']
      })
    }

    // Sort by relevance
    results.sort((a, b) => b.relevance - a.relevance)
    setSearchResults(results)
  }

  const searchSchemaRecursive = (
    obj: any, 
    path: string, 
    callback: (field: any) => void
  ) => {
    if (!obj || typeof obj !== 'object') return

    if (obj.properties) {
      Object.entries(obj.properties).forEach(([key, value]: [string, any]) => {
        const fieldPath = path ? `${path}.${key}` : key
        callback({
          path: fieldPath,
          type: value.type,
          title: value.title,
          description: value.description,
          required: value.required,
          default: value.default,
          example: value.example
        })
        searchSchemaRecursive(value, fieldPath, callback)
      })
    }
  }

  const handleResultClick = async (result: SearchResult) => {
    setSelectedResult(result)

    if (result.type === 'config' && result.path) {
      // Load field details
      try {
        const response = await fetch(`/api/config/field/${result.path}`)
        if (response.ok) {
          const data = await response.json()
          setSelectedField(data.data)
        }
      } catch (error) {
        console.error('Failed to load field details:', error)
      }

      // Navigate to field if callback provided
      if (onNavigate) {
        onNavigate(result.path)
      }
    } else if (result.url) {
      // Open documentation URL
      window.open(result.url, '_blank')
    }
  }

  const getResultIcon = (type: string) => {
    switch (type) {
      case 'config': return <CodeBracketIcon className="h-5 w-5" />
      case 'documentation': return <BookOpenIcon className="h-5 w-5" />
      case 'example': return <DocumentTextIcon className="h-5 w-5" />
      case 'api': return <CommandLineIcon className="h-5 w-5" />
      case 'tutorial': return <AcademicCapIcon className="h-5 w-5" />
      default: return <DocumentTextIcon className="h-5 w-5" />
    }
  }

  const getResultColor = (type: string) => {
    switch (type) {
      case 'config': return 'text-blue-600 bg-blue-50'
      case 'documentation': return 'text-green-600 bg-green-50'
      case 'example': return 'text-purple-600 bg-purple-50'
      case 'api': return 'text-orange-600 bg-orange-50'
      case 'tutorial': return 'text-pink-600 bg-pink-50'
      default: return 'text-gray-600 bg-gray-50'
    }
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              Configuration Search & Documentation
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Search configuration fields, documentation, and examples
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <button
              onClick={() => setShowHelp(true)}
              className="p-2 text-gray-400 hover:text-gray-600"
            >
              <QuestionMarkCircleIcon className="h-5 w-5" />
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

      {/* Search Bar */}
      <div className="bg-white border-b px-6 py-4">
        <div className="relative">
          <MagnifyingGlassIcon className="h-5 w-5 absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search configuration fields, docs, examples..."
            className="w-full pl-10 pr-4 py-3 text-lg border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            onKeyPress={(e) => {
              if (e.key === 'Enter' && searchQuery) {
                performSearch()
              }
            }}
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              ×
            </button>
          )}
        </div>

        {/* Quick Filters */}
        <div className="mt-3 flex items-center space-x-2">
          <span className="text-sm text-gray-500">Filter:</span>
          {['All', 'Config', 'Docs', 'Examples', 'API'].map((filter) => (
            <button
              key={filter}
              className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-full hover:bg-gray-200"
            >
              {filter}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Results */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                <p className="mt-3 text-sm text-gray-600">Searching...</p>
              </div>
            </div>
          ) : searchResults.length > 0 ? (
            <div className="space-y-3">
              {searchResults.map((result, index) => (
                <SearchResultItem
                  key={index}
                  result={result}
                  isSelected={selectedResult === result}
                  onClick={() => handleResultClick(result)}
                />
              ))}
            </div>
          ) : searchQuery ? (
            <div className="text-center py-12">
              <MagnifyingGlassIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
              <p className="text-gray-500">No results found for "{searchQuery}"</p>
              <p className="text-sm text-gray-400 mt-2">
                Try different keywords or check the documentation
              </p>
            </div>
          ) : (
            <div className="space-y-6">
              {/* Recent Searches */}
              {recentSearches.length > 0 && (
                <div>
                  <h3 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                    <ClockIcon className="h-4 w-4 mr-2" />
                    Recent Searches
                  </h3>
                  <div className="space-y-2">
                    {recentSearches.map((search, index) => (
                      <button
                        key={index}
                        onClick={() => setSearchQuery(search)}
                        className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded"
                      >
                        {search}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Suggestions */}
              <div>
                <h3 className="text-sm font-medium text-gray-900 mb-3 flex items-center">
                  <SparklesIcon className="h-4 w-4 mr-2" />
                  Popular Searches
                </h3>
                <div className="space-y-2">
                  {suggestions.map((suggestion, index) => (
                    <button
                      key={index}
                      onClick={() => setSearchQuery(suggestion.text)}
                      className="block w-full text-left px-3 py-2 hover:bg-gray-50 rounded"
                    >
                      <div className="flex items-center">
                        <suggestion.icon className="h-4 w-4 text-gray-400 mr-3" />
                        <div className="flex-1">
                          <p className="text-sm text-gray-700">{suggestion.text}</p>
                          <p className="text-xs text-gray-500">{suggestion.category}</p>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Details Panel */}
        {selectedResult && (
          <div className="w-96 border-l bg-gray-50 overflow-y-auto">
            <DetailsPanel
              result={selectedResult}
              field={selectedField}
              onClose={() => {
                setSelectedResult(null)
                setSelectedField(null)
              }}
            />
          </div>
        )}
      </div>

      {/* Help Modal */}
      {showHelp && (
        <HelpModal onClose={() => setShowHelp(false)} />
      )}
    </div>
  )
}

// Sub-components
const SearchResultItem: React.FC<{
  result: SearchResult
  isSelected: boolean
  onClick: () => void
}> = ({ result, isSelected, onClick }) => {
  return (
    <div
      onClick={onClick}
      className={`p-4 rounded-lg border cursor-pointer transition-all ${
        isSelected 
          ? 'border-blue-500 bg-blue-50' 
          : 'border-gray-200 bg-white hover:border-gray-300 hover:shadow-sm'
      }`}
    >
      <div className="flex items-start">
        <div className={`p-2 rounded-md mr-3 ${getResultColor(result.type)}`}>
          {getResultIcon(result.type)}
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-medium text-gray-900 mb-1">
            {result.title}
          </h4>
          <p className="text-sm text-gray-600 line-clamp-2">
            {result.description}
          </p>
          {result.snippet && (
            <p className="text-xs text-gray-500 mt-2 font-mono bg-gray-100 p-2 rounded">
              {result.snippet}
            </p>
          )}
          <div className="mt-2 flex items-center space-x-3">
            <span className="text-xs text-gray-500 capitalize">
              {result.type}
            </span>
            {result.path && (
              <span className="text-xs text-gray-400 font-mono">
                {result.path}
              </span>
            )}
            {result.tags.length > 0 && (
              <div className="flex items-center space-x-1">
                {result.tags.slice(0, 3).map((tag, index) => (
                  <span
                    key={index}
                    className="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
        <ArrowRightIcon className="h-4 w-4 text-gray-400 ml-3" />
      </div>
    </div>
  )

  function getResultIcon(type: string) {
    switch (type) {
      case 'config': return <CodeBracketIcon className="h-5 w-5" />
      case 'documentation': return <BookOpenIcon className="h-5 w-5" />
      case 'example': return <DocumentTextIcon className="h-5 w-5" />
      case 'api': return <CommandLineIcon className="h-5 w-5" />
      case 'tutorial': return <AcademicCapIcon className="h-5 w-5" />
      default: return <DocumentTextIcon className="h-5 w-5" />
    }
  }

  function getResultColor(type: string) {
    switch (type) {
      case 'config': return 'text-blue-600 bg-blue-50'
      case 'documentation': return 'text-green-600 bg-green-50'
      case 'example': return 'text-purple-600 bg-purple-50'
      case 'api': return 'text-orange-600 bg-orange-50'
      case 'tutorial': return 'text-pink-600 bg-pink-50'
      default: return 'text-gray-600 bg-gray-50'
    }
  }
}

const DetailsPanel: React.FC<{
  result: SearchResult
  field: ConfigField | null
  onClose: () => void
}> = ({ result, field, onClose }) => {
  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b bg-white">
        <div className="flex items-start justify-between">
          <h3 className="text-lg font-medium text-gray-900">
            {result.title}
          </h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            ×
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        {result.type === 'config' && field ? (
          <ConfigFieldDetails field={field} />
        ) : (
          <div className="space-y-6">
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-2">Description</h4>
              <p className="text-sm text-gray-600">{result.description}</p>
            </div>

            {result.context && (
              <div>
                <h4 className="text-sm font-medium text-gray-900 mb-2">Context</h4>
                <p className="text-sm text-gray-600">{result.context}</p>
              </div>
            )}

            {result.url && (
              <div>
                <a
                  href={result.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
                >
                  View full documentation
                  <ArrowRightIcon className="h-3 w-3 ml-1" />
                </a>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

const ConfigFieldDetails: React.FC<{ field: ConfigField }> = ({ field }) => {
  return (
    <div className="space-y-6">
      <div>
        <h4 className="text-sm font-medium text-gray-900 mb-2">Field Information</h4>
        <dl className="space-y-2">
          <div>
            <dt className="text-xs text-gray-500">Path</dt>
            <dd className="text-sm font-mono text-gray-900">{field.path}</dd>
          </div>
          <div>
            <dt className="text-xs text-gray-500">Type</dt>
            <dd className="text-sm text-gray-900">{field.type}</dd>
          </div>
          <div>
            <dt className="text-xs text-gray-500">Required</dt>
            <dd className="text-sm text-gray-900">{field.required ? 'Yes' : 'No'}</dd>
          </div>
        </dl>
      </div>

      {field.default !== undefined && (
        <div>
          <h4 className="text-sm font-medium text-gray-900 mb-2">Default Value</h4>
          <pre className="text-sm bg-gray-100 p-2 rounded">
            {JSON.stringify(field.default, null, 2)}
          </pre>
        </div>
      )}

      {field.example !== undefined && (
        <div>
          <h4 className="text-sm font-medium text-gray-900 mb-2">Example</h4>
          <pre className="text-sm bg-blue-50 p-2 rounded text-blue-700">
            {JSON.stringify(field.example, null, 2)}
          </pre>
        </div>
      )}

      {field.validations && field.validations.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-gray-900 mb-2">Validations</h4>
          <ul className="space-y-1">
            {field.validations.map((validation, index) => (
              <li key={index} className="text-sm text-gray-600 flex items-start">
                <span className="text-gray-400 mr-2">•</span>
                {validation}
              </li>
            ))}
          </ul>
        </div>
      )}

      {field.related_fields && field.related_fields.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-gray-900 mb-2">Related Fields</h4>
          <div className="space-y-1">
            {field.related_fields.map((related, index) => (
              <button
                key={index}
                className="text-sm text-blue-600 hover:text-blue-800 flex items-center"
              >
                {related}
                <ArrowRightIcon className="h-3 w-3 ml-1" />
              </button>
            ))}
          </div>
        </div>
      )}

      {field.documentation_url && (
        <div>
          <a
            href={field.documentation_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
          >
            <BookOpenIcon className="h-4 w-4 mr-2" />
            View documentation
          </a>
        </div>
      )}
    </div>
  )
}

const HelpModal: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-md w-full p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-900">Search Help</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            ×
          </button>
        </div>

        <div className="space-y-4 text-sm">
          <div>
            <h4 className="font-medium text-gray-900 mb-2">Search Tips</h4>
            <ul className="space-y-1 text-gray-600">
              <li>• Use keywords from field names or descriptions</li>
              <li>• Search for configuration sections like "security" or "providers"</li>
              <li>• Look for examples with "example:" prefix</li>
              <li>• Find tutorials with "how to" queries</li>
            </ul>
          </div>

          <div>
            <h4 className="font-medium text-gray-900 mb-2">Keyboard Shortcuts</h4>
            <ul className="space-y-1 text-gray-600">
              <li><kbd className="px-2 py-1 bg-gray-100 rounded">Enter</kbd> - Search</li>
              <li><kbd className="px-2 py-1 bg-gray-100 rounded">Esc</kbd> - Clear search</li>
              <li><kbd className="px-2 py-1 bg-gray-100 rounded">↑ ↓</kbd> - Navigate results</li>
            </ul>
          </div>

          <div>
            <h4 className="font-medium text-gray-900 mb-2">Available Resources</h4>
            <ul className="space-y-1 text-gray-600">
              <li>• Configuration field reference</li>
              <li>• Setup guides and tutorials</li>
              <li>• API documentation</li>
              <li>• Example configurations</li>
              <li>• Best practices guides</li>
            </ul>
          </div>
        </div>

        <div className="mt-6">
          <button
            onClick={onClose}
            className="w-full px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
          >
            Got it
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConfigSearch