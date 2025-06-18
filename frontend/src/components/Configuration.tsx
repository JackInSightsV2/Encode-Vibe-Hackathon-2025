import React, { useState, useEffect } from 'react'

interface Provider {
  name: string
  api_key: string
  base_url: string
  models: string[]
  headers: Record<string, string>
  timeout: number
}

interface ModerationLayer {
  type: string
  provider?: string
  model?: string
  prompt?: string
  threshold: number
  settings?: Record<string, any>
}

interface Config {
  server: {
    port: number
    host: string
    target_url: string
  }
  providers: Record<string, Provider>
  routing: {
    default_provider: string
    model_routing: Record<string, string>
    fallback_chain: string[]
  }
  security: {
    prompt_injection: {
      enabled: boolean
      sensitivity: string
      patterns: string[]
    }
  }
  moderation: {
    enabled: boolean
    severity: string
    blocked_words: string[]
    layers: ModerationLayer[]
    use_openai: boolean
  }
  relevance: {
    enabled: boolean
    threshold: number
    provider: string
    model: string
    use_openai: boolean
  }
  kill_switch: {
    enabled: boolean
    blocked_users: string[]
    blocked_sessions: string[]
  }
  logging: {
    enabled: boolean
    log_file: string
    log_level: string
    use_sqlite: boolean
    sqlite_db: string
  }
}

const Configuration: React.FC = () => {
  const [config, setConfig] = useState<Config | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{type: 'success' | 'error', text: string} | null>(null)
  const [activeTab, setActiveTab] = useState('providers')

  useEffect(() => {
    fetchConfig()
  }, [])

  const fetchConfig = async () => {
    try {
      const response = await fetch('/api/config')
      if (response.ok) {
        const data = await response.json()
        setConfig(data.data)
      } else {
        throw new Error('Failed to fetch config')
      }
    } catch (error) {
      setMessage({type: 'error', text: 'Failed to load configuration'})
    } finally {
      setLoading(false)
    }
  }

  const saveConfig = async () => {
    if (!config) return
    
    setSaving(true)
    setMessage(null)
    
    try {
      const response = await fetch('/api/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(config),
      })
      
      if (response.ok) {
        setMessage({type: 'success', text: 'Configuration saved successfully!'})
      } else {
        throw new Error('Failed to save config')
      }
    } catch (error) {
      setMessage({type: 'error', text: 'Failed to save configuration'})
    } finally {
      setSaving(false)
    }
  }

  const updateConfig = (path: string[], value: any) => {
    if (!config) return
    
    const newConfig = JSON.parse(JSON.stringify(config))
    let current: any = newConfig
    
    for (let i = 0; i < path.length - 1; i++) {
      current = current[path[i]]
    }
    current[path[path.length - 1]] = value
    
    setConfig(newConfig)
  }

  const updateProvider = (providerKey: string, field: string, value: any) => {
    if (!config) return
    const newConfig = { ...config }
    newConfig.providers[providerKey] = {
      ...newConfig.providers[providerKey],
      [field]: value
    }
    setConfig(newConfig)
  }

  const tabs = [
    { id: 'endpoints', label: 'Endpoints', icon: '🔌' },
    { id: 'routing', label: 'Routing', icon: '🔀' },
    { id: 'security', label: 'Security', icon: '🛡️' },
    { id: 'moderation', label: 'Moderation', icon: '🔍' },
    { id: 'server', label: 'Server', icon: '⚙️' },
    { id: 'logging', label: 'Logging', icon: '📋' }
  ]

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-lg">Loading configuration...</div>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="text-center py-8">
        <div className="text-red-600">Failed to load configuration</div>
        <button onClick={fetchConfig} className="mt-2 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
          Retry
        </button>
      </div>
    )
  }

  const renderEndpointsTab = () => {
    if (!config) return null;
    
    // Reorder providers to put local first, mock last
    const providerEntries = Object.entries(config.providers);
    const localProvider = providerEntries.find(([key]) => key === 'local');
    const mockProvider = providerEntries.find(([key]) => key === 'mock');
    const otherProviders = providerEntries.filter(
      ([key]) => key !== 'local' && key !== 'mock' && key !== 'google'
    );
    
    // Define Google provider
    const googleProvider: [string, Provider] = [
      'google', 
      config.providers.google || { 
        name: 'Google', 
        api_key: '', 
        base_url: '', 
        models: ['gemini-pro'], 
        headers: {}, 
        timeout: 30 
      }
    ];
    
    const orderedProviders: [string, Provider][] = [];
    
    // Add local provider if it exists
    if (localProvider) orderedProviders.push(localProvider);
    
    // Add other providers
    orderedProviders.push(...otherProviders);
    
    // Add Google provider
    orderedProviders.push(googleProvider);
    
    // Add mock provider if it exists
    if (mockProvider) orderedProviders.push(mockProvider);

    return (
      <div className="space-y-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold mb-4">Endpoint Configuration</h3>
          <p className="text-gray-600 mb-6">Configure your AI endpoints and API credentials</p>
          
          {orderedProviders.map(([key, provider]) => (
            <div key={key} className="border rounded-lg p-4 mb-4">
              <div className="flex justify-between items-center mb-3">
                <h4 className="font-medium text-lg capitalize">{provider.name}</h4>
                <div className="flex items-center">
                  <span className="text-sm text-gray-500 mr-2">Enabled</span>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input 
                      type="checkbox" 
                      className="sr-only peer"
                      defaultChecked={key !== 'mock'}
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                  </label>
                </div>
              </div>
              
              {key === 'local' && (
                <div className="mb-3 p-3 bg-blue-50 text-blue-700 text-sm rounded-md">
                  ℹ️ Use this to route to your backend endpoints or local LLMs.
                </div>
              )}
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">API Key</label>
                  <div className="relative">
                    <input
                      type="password"
                      placeholder={key === 'mock' ? 'sk-Mock' : "Set via environment variable"}
                      value={key === 'mock' ? 'sk-Mock' : provider.api_key}
                      disabled={key === 'mock'}
                      onChange={(e) => updateProvider(key, 'api_key', e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
                    />
                    {key === 'mock' && (
                      <div className="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none">
                        <span className="text-gray-500 text-xs">Auto-filled</span>
                      </div>
                    )}
                  </div>
                  <p className="text-xs text-gray-500 mt-1">
                    {key === 'mock' ? 'Mock API key for testing' : 'Recommend using env vars for security'}
                  </p>
                </div>
                
                {key !== 'openai' && key !== 'anthropic' && key !== 'google' && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Base URL
                      {key === 'local' && (
                        <span className="text-xs text-gray-500 ml-1">(e.g., http://localhost:11434)</span>
                      )}
                    </label>
                    <input
                      type="text"
                      value={provider.base_url}
                      onChange={(e) => updateProvider(key, 'base_url', e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder={key === 'local' ? 'http://localhost:11434' : 'https://api.example.com'}
                    />
                  </div>
                )}
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Timeout (seconds)
                  </label>
                  <div className="relative">
                    <input
                      type="number"
                      value={provider.timeout}
                      onChange={(e) => updateProvider(key, 'timeout', parseInt(e.target.value))}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      min="1"
                    />
                  </div>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Models
                    <span className="text-xs font-normal text-gray-500 ml-1">(comma separated)</span>
                  </label>
                  <input
                    type="text"
                    value={provider.models.join(', ')}
                    onChange={(e) => {
                      const models = e.target.value.split(',').map(m => m.trim()).filter(Boolean);
                      updateProvider(key, 'models', models);
                    }}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="model1, model2, model3"
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  const renderSecurityTab = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Prompt Injection Protection</h3>
        <div className="space-y-4">
          <div className="flex items-center space-x-3">
            <input
              type="checkbox"
              id="prompt-injection-enabled"
              checked={config.security.prompt_injection.enabled}
              onChange={(e) => updateConfig(['security', 'prompt_injection', 'enabled'], e.target.checked)}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            />
            <label htmlFor="prompt-injection-enabled" className="text-sm font-medium text-gray-700">
              Enable Prompt Injection Detection
            </label>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Sensitivity Level</label>
            <select
              value={config.security.prompt_injection.sensitivity}
              onChange={(e) => updateConfig(['security', 'prompt_injection', 'sensitivity'], e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </div>
          
          <div>
            <label className="text-sm font-medium text-gray-700">Detection Patterns</label>
            <div className="mt-2 flex flex-wrap gap-2">
              {config.security.prompt_injection.patterns.map((pattern, index) => (
                <span
                  key={index}
                  className="inline-flex items-center px-3 py-1 rounded-full text-sm bg-yellow-100 text-yellow-800"
                >
                  {pattern}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
      
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Kill Switch</h3>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-gray-700">Kill Switch</h4>
              <p className="text-xs text-gray-500">Block specific users or sessions from accessing the API</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input 
                type="checkbox" 
                className="sr-only peer"
                checked={config.kill_switch.enabled}
                onChange={(e) => updateConfig(['kill_switch', 'enabled'], e.target.checked)}
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>
          
          {config.kill_switch.enabled && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Blocked User IDs</label>
                <div className="flex flex-wrap gap-2">
                  {config.kill_switch.blocked_users.map((userId, index) => (
                    <div key={index} className="flex items-center bg-red-50 text-red-700 px-2 py-1 rounded text-sm">
                      {userId}
                      <button 
                        onClick={() => {
                          const updated = [...config.kill_switch.blocked_users];
                          updated.splice(index, 1);
                          updateConfig(['kill_switch', 'blocked_users'], updated);
                        }}
                        className="ml-2 text-red-500 hover:text-red-700"
                      >
                        ×
                      </button>
                    </div>
                  ))}
                  <button 
                    onClick={() => {
                      const userId = prompt('Enter user ID to block:');
                      if (userId) {
                        updateConfig(
                          ['kill_switch', 'blocked_users'], 
                          [...new Set([...config.kill_switch.blocked_users, userId])]
                        );
                      }
                    }}
                    className="text-sm text-blue-600 hover:text-blue-800"
                  >
                    + Add User ID
                  </button>
                </div>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Blocked Session IDs</label>
                <div className="flex flex-wrap gap-2">
                  {config.kill_switch.blocked_sessions.map((sessionId, index) => (
                    <div key={index} className="flex items-center bg-red-50 text-red-700 px-2 py-1 rounded text-sm">
                      {sessionId}
                      <button 
                        onClick={() => {
                          const updated = [...config.kill_switch.blocked_sessions];
                          updated.splice(index, 1);
                          updateConfig(['kill_switch', 'blocked_sessions'], updated);
                        }}
                        className="ml-2 text-red-500 hover:text-red-700"
                      >
                        ×
                      </button>
                    </div>
                  ))}
                  <button 
                    onClick={() => {
                      const sessionId = prompt('Enter session ID to block:');
                      if (sessionId) {
                        updateConfig(
                          ['kill_switch', 'blocked_sessions'], 
                          [...new Set([...config.kill_switch.blocked_sessions, sessionId])]
                        );
                      }
                    }}
                    className="text-sm text-blue-600 hover:text-blue-800"
                  >
                    + Add Session ID
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )

  const renderModerationTab = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Content Moderation</h3>
        <div className="space-y-6">
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h4 className="text-sm font-medium text-gray-700">Word Filter</h4>
                <p className="text-xs text-gray-500">Block messages containing specific words or phrases</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input 
                  type="checkbox" 
                  className="sr-only peer"
                  checked={config.moderation.enabled}
                  onChange={(e) => updateConfig(['moderation', 'enabled'], e.target.checked)}
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>
            
            {config.moderation.enabled && (
              <div className="space-y-4">
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-sm font-medium text-gray-700">Blocked Words</label>
                    <div className="text-xs text-gray-500">
                      {config.moderation.blocked_words.length} words
                    </div>
                  </div>
                  
                  <div className="flex flex-wrap gap-2 p-3 bg-gray-50 rounded-md border border-gray-200 min-h-[60px]">
                    {config.moderation.blocked_words.length > 0 ? (
                      config.moderation.blocked_words.map((word, index) => (
                        <div key={index} className="flex items-center bg-white border border-gray-200 rounded-full px-3 py-1 text-sm">
                          {word}
                          <button 
                            onClick={() => {
                              const updated = [...config.moderation.blocked_words];
                              updated.splice(index, 1);
                              updateConfig(['moderation', 'blocked_words'], updated);
                            }}
                            className="ml-1.5 text-gray-400 hover:text-red-500"
                          >
                            ×
                          </button>
                        </div>
                      ))
                    ) : (
                      <p className="text-sm text-gray-400 italic">No blocked words added yet</p>
                    )}
                  </div>
                  
                  <div className="mt-2 flex">
                    <input
                      type="text"
                      id="add-word"
                      placeholder="Add a word or phrase"
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-l-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && e.currentTarget.value.trim()) {
                          const word = e.currentTarget.value.trim().toLowerCase();
                          if (!config.moderation.blocked_words.includes(word)) {
                            updateConfig(
                              ['moderation', 'blocked_words'], 
                              [...config.moderation.blocked_words, word]
                            );
                          }
                          e.currentTarget.value = '';
                        }
                      }}
                    />
                    <button
                      onClick={() => {
                        const input = document.getElementById('add-word') as HTMLInputElement;
                        if (input && input.value.trim()) {
                          const word = input.value.trim().toLowerCase();
                          if (!config.moderation.blocked_words.includes(word)) {
                            updateConfig(
                              ['moderation', 'blocked_words'], 
                              [...config.moderation.blocked_words, word]
                            );
                          }
                          input.value = '';
                        }
                      }}
                      className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-r-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      Add
                    </button>
                  </div>
                  
                  <div className="mt-2">
                    <button
                      onClick={() => {
                        const words = prompt('Enter multiple words separated by commas:');
                        if (words) {
                          const newWords = words
                            .split(',')
                            .map(w => w.trim().toLowerCase())
                            .filter(w => w.length > 0 && !config.moderation.blocked_words.includes(w));
                          
                          if (newWords.length > 0) {
                            updateConfig(
                              ['moderation', 'blocked_words'], 
                              [...config.moderation.blocked_words, ...newWords]
                            );
                          }
                        }
                      }}
                      className="text-xs text-blue-600 hover:text-blue-800"
                    >
                      + Add multiple words at once
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
          
          <div className="border-t border-gray-200 pt-6">
            <h4 className="text-sm font-medium text-gray-700 mb-4">LLM Content Moderation</h4>
            <div className="space-y-4">
              <div className="flex items-start">
                <div className="flex items-center h-5">
                  <input
                    id="llm-moderation"
                    type="checkbox"
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                </div>
                <div className="ml-3 text-sm">
                  <label htmlFor="llm-moderation" className="font-medium text-gray-700">
                    Enable AI-powered content moderation
                  </label>
                  <p className="text-gray-500">
                    Uses an LLM to detect and block inappropriate content
                  </p>
                </div>
              </div>
              
              <div className="ml-7 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Provider
                  </label>
                  <select
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                  >
                    <option>OpenAI</option>
                    <option>Anthropic</option>
                    <option>Google</option>
                  </select>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Model
                  </label>
                  <select
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                  >
                    <option>gpt-4</option>
                    <option>claude-3-opus</option>
                    <option>gemini-pro</option>
                  </select>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Moderation Categories
                  </label>
                  <div className="space-y-2">
                    {['Hate', 'Harassment', 'Self-harm', 'Violence', 'Adult content'].map((category) => (
                      <div key={category} className="flex items-center">
                        <input
                          id={`category-${category.toLowerCase()}`}
                          type="checkbox"
                          className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                          defaultChecked
                        />
                        <label htmlFor={`category-${category.toLowerCase()}`} className="ml-2 text-sm text-gray-700">
                          {category}
                        </label>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )

  const renderRoutingTab = () => {
    const providerOptions = Object.keys(config.providers).map(provider => ({
      value: provider,
      label: config.providers[provider].name || provider
    }));

    const updateFallbackProvider = (index: number, provider: string) => {
      const updated = [...(config.routing.fallback_chain || [])];
      updated[index] = provider;
      updateConfig(['routing', 'fallback_chain'], updated);
    };

    return (
      <div className="space-y-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold mb-6">Provider Routing</h3>
          
          <div className="space-y-6 max-w-2xl">
            <div>
              <h4 className="text-sm font-medium text-gray-700 mb-3">Fallback Chain</h4>
              <p className="text-sm text-gray-500 mb-4">
                Select up to 4 providers in order of preference. The system will try each provider in sequence if the previous one fails.
              </p>
              
              <div className="space-y-4">
                {[0, 1, 2, 3].map((index) => (
                  <div key={index} className="flex items-center">
                    <span className="w-8 text-sm font-medium text-gray-500">{index + 1}.</span>
                    <select
                      value={config.routing.fallback_chain[index] || ''}
                      onChange={(e) => updateFallbackProvider(index, e.target.value)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
                    >
                      <option value="">Select a provider...</option>
                      {providerOptions.map((option) => (
                        <option 
                          key={option.value} 
                          value={option.value}
                          disabled={config.routing.fallback_chain.includes(option.value) && 
                                   config.routing.fallback_chain[index] !== option.value}
                        >
                          {option.label}
                        </option>
                      ))}
                    </select>
                    
                    {index > 0 && (
                      <button
                        onClick={() => {
                          const updated = [...config.routing.fallback_chain];
                          updated.splice(index, 1);
                          updateConfig(['routing', 'fallback_chain'], updated);
                        }}
                        className="ml-2 p-1 text-gray-400 hover:text-red-500"
                        title="Remove provider"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                          <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
                        </svg>
                      </button>
                    )}
                  </div>
                ))}
                
                {config.routing.fallback_chain.length < 4 && (
                  <div className="pt-2">
                    <button
                      onClick={() => {
                        const availableProviders = Object.keys(config.providers).filter(
                          p => !config.routing.fallback_chain.includes(p)
                        );
                        
                        if (availableProviders.length > 0) {
                          updateConfig(
                            ['routing', 'fallback_chain'],
                            [...config.routing.fallback_chain, availableProviders[0]]
                          );
                        }
                      }}
                      className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 mr-1" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clipRule="evenodd" />
                      </svg>
                      Add another provider
                    </button>
                  </div>
                )}
              </div>
            </div>
            
            <div className="pt-4 border-t border-gray-200">
              <h4 className="text-sm font-medium text-gray-700 mb-2">Current Fallback Order</h4>
              {config.routing.fallback_chain.length > 0 ? (
                <ol className="list-decimal list-inside space-y-1 text-sm text-gray-600">
                  {config.routing.fallback_chain.map((provider, index) => (
                    <li key={index} className="flex items-center">
                      <span className="ml-1">
                        {config.providers[provider]?.name || provider}
                        <span className="text-xs text-gray-400 ml-2">({provider})</span>
                      </span>
                    </li>
                  ))}
                </ol>
              ) : (
                <p className="text-sm text-gray-500 italic">No providers in the fallback chain</p>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }

  const renderServerTab = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Server Configuration</h3>
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Server Port</label>
              <input
                type="number"
                value={config.server.port}
                onChange={(e) => updateConfig(['server', 'port'], parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                min="1"
                max="65535"
              />
              <p className="mt-1 text-xs text-gray-500">Port number to run the server on</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Bind Address</label>
              <input
                type="text"
                value={config.server.host}
                onChange={(e) => updateConfig(['server', 'host'], e.target.value)}
                placeholder="0.0.0.0"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="mt-1 text-xs text-gray-500">Leave as 0.0.0.0 to accept connections from any network</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )

  const renderLoggingTab = () => (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Logging Configuration</h3>
        <div className="space-y-4">
          <div className="flex items-center space-x-3">
            <input
              type="checkbox"
              id="logging-enabled"
              checked={config.logging.enabled}
              onChange={(e) => updateConfig(['logging', 'enabled'], e.target.checked)}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            />
            <label htmlFor="logging-enabled" className="text-sm font-medium text-gray-700">
              Enable Logging
            </label>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Log Level</label>
            <select
              value={config.logging.log_level}
              onChange={(e) => updateConfig(['logging', 'log_level'], e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="debug">Debug</option>
              <option value="info">Info</option>
              <option value="warn">Warning</option>
              <option value="error">Error</option>
            </select>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Log File Path</label>
            <input
              type="text"
              value={config.logging.log_file}
              onChange={(e) => updateConfig(['logging', 'log_file'], e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          
          <div className="flex items-center space-x-3">
            <input
              type="checkbox"
              id="sqlite-enabled"
              checked={config.logging.use_sqlite}
              onChange={(e) => updateConfig(['logging', 'use_sqlite'], e.target.checked)}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            />
            <label htmlFor="sqlite-enabled" className="text-sm font-medium text-gray-700">
              Use SQLite Database
            </label>
          </div>
        </div>
      </div>
    </div>
  )

  const renderTabContent = () => {
    if (!config) return null;
    
    switch (activeTab) {
      case 'endpoints':
        return renderEndpointsTab();
      case 'routing':
        return renderRoutingTab();
      case 'security':
        return renderSecurityTab();
      case 'moderation':
        return renderModerationTab();
      case 'server':
        return renderServerTab();
      case 'logging':
        return renderLoggingTab();
      default:
        return null;
    }
  };

  return (
    <div>
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Configuration</h1>
          <p className="text-gray-600">Manage QT-1 middleware settings and AI provider configuration</p>
        </div>
        <button
          onClick={saveConfig}
          disabled={saving}
          className="bg-green-600 text-white px-6 py-2 rounded-md hover:bg-green-700 disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving...' : 'Save Configuration'}
        </button>
      </header>

      {message && (
        <div className={`mb-6 p-4 rounded-md ${
          message.type === 'success' ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'
        }`}>
          {message.text}
        </div>
      )}

      {/* Tab Navigation */}
      <div className="mb-6 border-b border-gray-200">
        <nav className="-mb-px flex space-x-8 overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`py-2 px-1 border-b-2 font-medium text-sm whitespace-nowrap ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      {renderTabContent()}
    </div>
  )
}

export default Configuration