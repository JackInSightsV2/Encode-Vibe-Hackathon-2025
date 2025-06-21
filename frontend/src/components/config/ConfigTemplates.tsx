import React, { useState, useEffect } from 'react'
import { 
  DocumentDuplicateIcon,
  PlusIcon,
  StarIcon,
  TagIcon,
  CloudArrowDownIcon,
  PencilIcon,
  TrashIcon,
  ShareIcon,
  LockClosedIcon,
  LockOpenIcon,
  ClockIcon,
  UserGroupIcon,
  CheckBadgeIcon,
  ArrowTrendingUpIcon
} from '@heroicons/react/24/outline'
import { StarIcon as StarIconSolid } from '@heroicons/react/24/solid'

interface Template {
  id: string
  name: string
  description: string
  category: 'basic' | 'advanced' | 'security' | 'performance' | 'development' | 'custom'
  config: any
  author: string
  author_id: string
  created_at: string
  updated_at: string
  version: string
  downloads: number
  stars: number
  is_starred: boolean
  is_public: boolean
  is_verified: boolean
  tags: string[]
  compatibility: string[]
  preview_image?: string
  documentation_url?: string
}

interface TemplateFilter {
  category: string
  tags: string[]
  author: string
  compatibility: string
  sortBy: 'popular' | 'recent' | 'starred' | 'alphabetical'
  showOnlyPublic: boolean
  showOnlyVerified: boolean
  searchQuery: string
}

interface ConfigTemplatesProps {
  currentConfig?: any
  onApplyTemplate: (config: any) => void
  onClose: () => void
}

const ConfigTemplates: React.FC<ConfigTemplatesProps> = ({
  currentConfig,
  onApplyTemplate,
  onClose
}) => {
  const [templates, setTemplates] = useState<Template[]>([])
  const [myTemplates, setMyTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'browse' | 'my-templates' | 'create'>('browse')
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [filter, setFilter] = useState<TemplateFilter>({
    category: 'all',
    tags: [],
    author: '',
    compatibility: '',
    sortBy: 'popular',
    showOnlyPublic: true,
    showOnlyVerified: false,
    searchQuery: ''
  })
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newTemplate, setNewTemplate] = useState({
    name: '',
    description: '',
    category: 'custom' as const,
    tags: [] as string[],
    is_public: false,
    documentation_url: ''
  })

  useEffect(() => {
    fetchTemplates()
  }, [])

  const fetchTemplates = async () => {
    try {
      const [publicRes, privateRes] = await Promise.all([
        fetch('/api/config/templates/public'),
        fetch('/api/config/templates/my')
      ])

      if (publicRes.ok && privateRes.ok) {
        const publicData = await publicRes.json()
        const privateData = await privateRes.json()
        
        setTemplates(publicData.data || [])
        setMyTemplates(privateData.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch templates:', error)
    } finally {
      setLoading(false)
    }
  }

  const createTemplate = async () => {
    if (!currentConfig) return

    try {
      const response = await fetch('/api/config/templates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...newTemplate,
          config: currentConfig
        })
      })

      if (response.ok) {
        await fetchTemplates()
        setShowCreateForm(false)
        resetNewTemplateForm()
        setActiveTab('my-templates')
      }
    } catch (error) {
      console.error('Failed to create template:', error)
    }
  }

  const updateTemplate = async (template: Template) => {
    try {
      const response = await fetch(`/api/config/templates/${template.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(template)
      })

      if (response.ok) {
        await fetchTemplates()
      }
    } catch (error) {
      console.error('Failed to update template:', error)
    }
  }

  const deleteTemplate = async (templateId: string) => {
    if (!window.confirm('Are you sure you want to delete this template?')) {
      return
    }

    try {
      const response = await fetch(`/api/config/templates/${templateId}`, {
        method: 'DELETE'
      })

      if (response.ok) {
        await fetchTemplates()
        if (selectedTemplate?.id === templateId) {
          setSelectedTemplate(null)
        }
      }
    } catch (error) {
      console.error('Failed to delete template:', error)
    }
  }

  const toggleStar = async (template: Template) => {
    try {
      const response = await fetch(`/api/config/templates/${template.id}/star`, {
        method: template.is_starred ? 'DELETE' : 'POST'
      })

      if (response.ok) {
        // Update local state
        const updatedTemplates = templates.map(t => 
          t.id === template.id 
            ? { ...t, is_starred: !t.is_starred, stars: t.stars + (t.is_starred ? -1 : 1) }
            : t
        )
        setTemplates(updatedTemplates)
      }
    } catch (error) {
      console.error('Failed to toggle star:', error)
    }
  }

  const applyTemplate = (template: Template) => {
    onApplyTemplate(template.config)
    onClose()
  }

  const resetNewTemplateForm = () => {
    setNewTemplate({
      name: '',
      description: '',
      category: 'custom',
      tags: [],
      is_public: false,
      documentation_url: ''
    })
  }

  const getFilteredTemplates = () => {
    let filtered = activeTab === 'browse' ? templates : myTemplates

    // Apply filters
    if (filter.category !== 'all') {
      filtered = filtered.filter(t => t.category === filter.category)
    }
    if (filter.tags.length > 0) {
      filtered = filtered.filter(t => 
        filter.tags.some(tag => t.tags.includes(tag))
      )
    }
    if (filter.author) {
      filtered = filtered.filter(t => 
        t.author.toLowerCase().includes(filter.author.toLowerCase())
      )
    }
    if (filter.showOnlyVerified) {
      filtered = filtered.filter(t => t.is_verified)
    }
    if (filter.searchQuery) {
      const query = filter.searchQuery.toLowerCase()
      filtered = filtered.filter(t => 
        t.name.toLowerCase().includes(query) ||
        t.description.toLowerCase().includes(query) ||
        t.tags.some(tag => tag.toLowerCase().includes(query))
      )
    }

    // Sort
    switch (filter.sortBy) {
      case 'popular':
        filtered.sort((a, b) => b.downloads - a.downloads)
        break
      case 'recent':
        filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
        break
      case 'starred':
        filtered.sort((a, b) => b.stars - a.stars)
        break
      case 'alphabetical':
        filtered.sort((a, b) => a.name.localeCompare(b.name))
        break
    }

    return filtered
  }

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'basic': return 'bg-blue-100 text-blue-700'
      case 'advanced': return 'bg-purple-100 text-purple-700'
      case 'security': return 'bg-red-100 text-red-700'
      case 'performance': return 'bg-green-100 text-green-700'
      case 'development': return 'bg-yellow-100 text-yellow-700'
      case 'custom': return 'bg-gray-100 text-gray-700'
      default: return 'bg-gray-100 text-gray-700'
    }
  }

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading templates...</p>
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
              Configuration Templates
            </h2>
            <p className="text-sm text-gray-600 mt-1">
              Browse and manage configuration templates
            </p>
          </div>
          <div className="flex items-center space-x-3">
            {currentConfig && (
              <button
                onClick={() => setShowCreateForm(true)}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
              >
                <PlusIcon className="h-4 w-4 inline-block mr-2" />
                Create Template
              </button>
            )}
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

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <nav className="flex space-x-8">
          {['browse', 'my-templates'].map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab as any)}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab === 'browse' ? 'Browse Templates' : 'My Templates'}
              {tab === 'my-templates' && myTemplates.length > 0 && (
                <span className="ml-2 px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded-full">
                  {myTemplates.length}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Filters */}
      <FilterBar filter={filter} setFilter={setFilter} />

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {getFilteredTemplates().map((template) => (
            <TemplateCard
              key={template.id}
              template={template}
              isSelected={selectedTemplate?.id === template.id}
              onSelect={() => setSelectedTemplate(template)}
              onApply={() => applyTemplate(template)}
              onToggleStar={() => toggleStar(template)}
              onEdit={() => {
                // TODO: Implement edit functionality
                console.log('Edit template:', template.id)
              }}
              onDelete={() => deleteTemplate(template.id)}
              isOwner={template.author_id === 'current_user'} // TODO: Get actual user ID
            />
          ))}
        </div>

        {getFilteredTemplates().length === 0 && (
          <div className="text-center py-12">
            <DocumentDuplicateIcon className="h-12 w-12 mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">No templates found</p>
            <p className="text-sm text-gray-400 mt-2">
              Try adjusting your filters or create a new template
            </p>
          </div>
        )}
      </div>

      {/* Create Template Modal */}
      {showCreateForm && (
        <CreateTemplateModal
          newTemplate={newTemplate}
          setNewTemplate={setNewTemplate}
          onCreate={createTemplate}
          onClose={() => {
            setShowCreateForm(false)
            resetNewTemplateForm()
          }}
        />
      )}

      {/* Template Details Modal */}
      {selectedTemplate && (
        <TemplateDetailsModal
          template={selectedTemplate}
          onClose={() => setSelectedTemplate(null)}
          onApply={() => applyTemplate(selectedTemplate)}
          onToggleStar={() => toggleStar(selectedTemplate)}
          isOwner={selectedTemplate.author_id === 'current_user'}
        />
      )}
    </div>
  )
}

// Sub-components
const FilterBar: React.FC<{
  filter: TemplateFilter
  setFilter: (filter: TemplateFilter) => void
}> = ({ filter, setFilter }) => {
  const categories = ['all', 'basic', 'advanced', 'security', 'performance', 'development', 'custom']

  return (
    <div className="bg-gray-50 border-b px-6 py-3">
      <div className="flex items-center justify-between space-x-4">
        {/* Categories */}
        <div className="flex items-center space-x-2 overflow-x-auto">
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setFilter({ ...filter, category })}
              className={`px-3 py-1 text-sm rounded-full whitespace-nowrap ${
                filter.category === category
                  ? 'bg-blue-600 text-white'
                  : 'bg-white text-gray-700 hover:bg-gray-100'
              }`}
            >
              {category === 'all' ? 'All' : category.charAt(0).toUpperCase() + category.slice(1)}
            </button>
          ))}
        </div>

        {/* Controls */}
        <div className="flex items-center space-x-3">
          {/* Sort */}
          <select
            value={filter.sortBy}
            onChange={(e) => setFilter({ ...filter, sortBy: e.target.value as any })}
            className="text-sm border border-gray-300 rounded px-3 py-1"
          >
            <option value="popular">Most Popular</option>
            <option value="recent">Most Recent</option>
            <option value="starred">Most Starred</option>
            <option value="alphabetical">Alphabetical</option>
          </select>

          {/* Verified Only */}
          <label className="flex items-center text-sm">
            <input
              type="checkbox"
              checked={filter.showOnlyVerified}
              onChange={(e) => setFilter({ ...filter, showOnlyVerified: e.target.checked })}
              className="h-4 w-4 text-blue-600 rounded border-gray-300"
            />
            <span className="ml-2">Verified only</span>
          </label>

          {/* Search */}
          <input
            type="text"
            value={filter.searchQuery}
            onChange={(e) => setFilter({ ...filter, searchQuery: e.target.value })}
            placeholder="Search templates..."
            className="px-3 py-1 text-sm border border-gray-300 rounded-md"
          />
        </div>
      </div>
    </div>
  )
}

const TemplateCard: React.FC<{
  template: Template
  isSelected: boolean
  onSelect: () => void
  onApply: () => void
  onToggleStar: () => void
  onEdit?: () => void
  onDelete?: () => void
  isOwner: boolean
}> = ({ template, isSelected, onSelect, onApply, onToggleStar, onEdit, onDelete, isOwner }) => {
  return (
    <div
      onClick={onSelect}
      className={`border rounded-lg p-4 cursor-pointer transition-all hover:shadow-md ${
        isSelected ? 'border-blue-500 bg-blue-50' : 'border-gray-300 bg-white'
      }`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <div className="flex items-center">
            <h4 className="text-sm font-medium text-gray-900">{template.name}</h4>
            {template.is_verified && (
              <CheckBadgeIcon className="h-4 w-4 text-blue-500 ml-1" title="Verified" />
            )}
            {!template.is_public && (
              <LockClosedIcon className="h-4 w-4 text-gray-400 ml-1" title="Private" />
            )}
          </div>
          <p className="text-xs text-gray-500 mt-1">by {template.author}</p>
        </div>
        
        <button
          onClick={(e) => {
            e.stopPropagation()
            onToggleStar()
          }}
          className="p-1 hover:bg-gray-100 rounded"
        >
          {template.is_starred ? (
            <StarIconSolid className="h-5 w-5 text-yellow-500" />
          ) : (
            <StarIcon className="h-5 w-5 text-gray-400" />
          )}
        </button>
      </div>

      <p className="text-sm text-gray-600 mb-3 line-clamp-2">
        {template.description}
      </p>

      <div className="space-y-2">
        {/* Category */}
        <div className="flex items-center justify-between">
          <span className={`px-2 py-1 text-xs rounded-full ${getCategoryColor(template.category)}`}>
            {template.category}
          </span>
          <div className="flex items-center space-x-3 text-xs text-gray-500">
            <span className="flex items-center">
              <CloudArrowDownIcon className="h-3 w-3 mr-1" />
              {template.downloads}
            </span>
            <span className="flex items-center">
              <StarIcon className="h-3 w-3 mr-1" />
              {template.stars}
            </span>
          </div>
        </div>

        {/* Tags */}
        {template.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {template.tags.slice(0, 3).map((tag) => (
              <span
                key={tag}
                className="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 rounded"
              >
                {tag}
              </span>
            ))}
            {template.tags.length > 3 && (
              <span className="text-xs text-gray-500">+{template.tags.length - 3}</span>
            )}
          </div>
        )}

        {/* Actions */}
        {isSelected && (
          <div className="flex items-center justify-between pt-3 border-t">
            <div className="flex items-center space-x-2">
              {isOwner && (
                <>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      onEdit?.()
                    }}
                    className="p-1 text-gray-400 hover:text-gray-600"
                  >
                    <PencilIcon className="h-4 w-4" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      onDelete?.()
                    }}
                    className="p-1 text-gray-400 hover:text-red-600"
                  >
                    <TrashIcon className="h-4 w-4" />
                  </button>
                </>
              )}
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation()
                onApply()
              }}
              className="px-3 py-1 text-sm font-medium text-white bg-blue-600 rounded hover:bg-blue-700"
            >
              Apply
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

const CreateTemplateModal: React.FC<{
  newTemplate: any
  setNewTemplate: (template: any) => void
  onCreate: () => void
  onClose: () => void
}> = ({ newTemplate, setNewTemplate, onCreate, onClose }) => {
  const [tagInput, setTagInput] = useState('')

  const addTag = () => {
    if (tagInput && !newTemplate.tags.includes(tagInput)) {
      setNewTemplate({ ...newTemplate, tags: [...newTemplate.tags, tagInput] })
      setTagInput('')
    }
  }

  const removeTag = (tag: string) => {
    setNewTemplate({ 
      ...newTemplate, 
      tags: newTemplate.tags.filter((t: string) => t !== tag) 
    })
  }

  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-md w-full p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">
          Create Configuration Template
        </h3>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Template Name
            </label>
            <input
              type="text"
              value={newTemplate.name}
              onChange={(e) => setNewTemplate({ ...newTemplate, name: e.target.value })}
              placeholder="e.g., Production-ready configuration"
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={newTemplate.description}
              onChange={(e) => setNewTemplate({ ...newTemplate, description: e.target.value })}
              placeholder="Describe what this template is for..."
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Category
            </label>
            <select
              value={newTemplate.category}
              onChange={(e) => setNewTemplate({ ...newTemplate, category: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            >
              <option value="basic">Basic</option>
              <option value="advanced">Advanced</option>
              <option value="security">Security</option>
              <option value="performance">Performance</option>
              <option value="development">Development</option>
              <option value="custom">Custom</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Tags
            </label>
            <div className="flex items-center space-x-2 mb-2">
              <input
                type="text"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && addTag()}
                placeholder="Add a tag..."
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm"
              />
              <button
                onClick={addTag}
                className="px-3 py-2 text-sm text-blue-600 hover:text-blue-800"
              >
                Add
              </button>
            </div>
            <div className="flex flex-wrap gap-1">
              {newTemplate.tags.map((tag: string) => (
                <span
                  key={tag}
                  className="px-2 py-1 text-xs bg-gray-100 text-gray-700 rounded-full flex items-center"
                >
                  {tag}
                  <button
                    onClick={() => removeTag(tag)}
                    className="ml-1 text-gray-400 hover:text-gray-600"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          </div>

          <div>
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={newTemplate.is_public}
                onChange={(e) => setNewTemplate({ ...newTemplate, is_public: e.target.checked })}
                className="h-4 w-4 text-blue-600 rounded border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">
                Make this template public
              </span>
            </label>
            <p className="text-xs text-gray-500 ml-6 mt-1">
              Public templates can be discovered and used by other users
            </p>
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
            disabled={!newTemplate.name || !newTemplate.description}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            Create Template
          </button>
        </div>
      </div>
    </div>
  )
}

const TemplateDetailsModal: React.FC<{
  template: Template
  onClose: () => void
  onApply: () => void
  onToggleStar: () => void
  isOwner: boolean
}> = ({ template, onClose, onApply, onToggleStar, isOwner }) => {
  return (
    <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <h3 className="text-lg font-medium text-gray-900">{template.name}</h3>
              {template.is_verified && (
                <CheckBadgeIcon className="h-5 w-5 text-blue-500 ml-2" />
              )}
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

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-6">
            {/* Meta Info */}
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-4 text-sm text-gray-600">
                <span>by {template.author}</span>
                <span>•</span>
                <span>{new Date(template.created_at).toLocaleDateString()}</span>
                <span>•</span>
                <span>v{template.version}</span>
              </div>
              <div className="flex items-center space-x-4">
                <div className="flex items-center text-sm">
                  <CloudArrowDownIcon className="h-4 w-4 mr-1 text-gray-400" />
                  <span>{template.downloads} downloads</span>
                </div>
                <div className="flex items-center text-sm">
                  <StarIcon className="h-4 w-4 mr-1 text-yellow-500" />
                  <span>{template.stars} stars</span>
                </div>
              </div>
            </div>

            {/* Description */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-2">Description</h4>
              <p className="text-sm text-gray-600">{template.description}</p>
            </div>

            {/* Category & Tags */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-2">Category & Tags</h4>
              <div className="flex items-center flex-wrap gap-2">
                <span className={`px-3 py-1 text-sm rounded-full ${getCategoryColor(template.category)}`}>
                  {template.category}
                </span>
                {template.tags.map((tag) => (
                  <span
                    key={tag}
                    className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-full"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>

            {/* Compatibility */}
            {template.compatibility.length > 0 && (
              <div>
                <h4 className="text-sm font-medium text-gray-900 mb-2">Compatibility</h4>
                <div className="flex flex-wrap gap-2">
                  {template.compatibility.map((compat) => (
                    <span
                      key={compat}
                      className="px-2 py-1 text-xs bg-green-100 text-green-700 rounded"
                    >
                      {compat}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* Configuration Preview */}
            <div>
              <h4 className="text-sm font-medium text-gray-900 mb-2">Configuration Preview</h4>
              <pre className="bg-gray-50 rounded-lg p-4 text-xs overflow-x-auto">
                {JSON.stringify(template.config, null, 2)}
              </pre>
            </div>

            {/* Documentation */}
            {template.documentation_url && (
              <div>
                <h4 className="text-sm font-medium text-gray-900 mb-2">Documentation</h4>
                <a
                  href={template.documentation_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-blue-600 hover:text-blue-800"
                >
                  View documentation →
                </a>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t bg-gray-50">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <button
                onClick={onToggleStar}
                className={`px-4 py-2 text-sm font-medium rounded-md border ${
                  template.is_starred
                    ? 'text-yellow-600 bg-yellow-50 border-yellow-300'
                    : 'text-gray-700 bg-white border-gray-300 hover:bg-gray-50'
                }`}
              >
                {template.is_starred ? (
                  <>
                    <StarIconSolid className="h-4 w-4 inline-block mr-2" />
                    Starred
                  </>
                ) : (
                  <>
                    <StarIcon className="h-4 w-4 inline-block mr-2" />
                    Star
                  </>
                )}
              </button>
              {isOwner && (
                <button className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50">
                  <PencilIcon className="h-4 w-4 inline-block mr-2" />
                  Edit
                </button>
              )}
            </div>
            <div className="flex items-center space-x-3">
              <button
                onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={onApply}
                className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700"
              >
                Apply Template
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// Helper function
function getCategoryColor(category: string) {
  switch (category) {
    case 'basic': return 'bg-blue-100 text-blue-700'
    case 'advanced': return 'bg-purple-100 text-purple-700'
    case 'security': return 'bg-red-100 text-red-700'
    case 'performance': return 'bg-green-100 text-green-700'
    case 'development': return 'bg-yellow-100 text-yellow-700'
    case 'custom': return 'bg-gray-100 text-gray-700'
    default: return 'bg-gray-100 text-gray-700'
  }
}

export default ConfigTemplates