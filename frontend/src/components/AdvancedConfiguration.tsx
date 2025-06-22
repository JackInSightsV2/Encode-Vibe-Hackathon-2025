import React, { useState, useEffect } from 'react'
import { useWebSocket } from '../contexts/WebSocketContext'

interface LayerConfig {
  name: string
  enabled: boolean
  weight: number
  threshold: number
  options: Record<string, any>
}

interface ModerationConfig {
  enabled: boolean
  layers: LayerConfig[]
  thresholds: {
    low: number
    medium: number
    high: number
    critical: number
  }
  cache: {
    enabled: boolean
    ttl: number
    max_size: number
  }
  analytics: {
    enabled: boolean
    sample_rate: number
    retention_days: number
  }
}

interface RuleEngineConfig {
  rules_path: string
  hot_reload: boolean
  reload_interval: number
  max_rules: number
  default_weight: number
  default_priority: number
  enable_caching: boolean
  cache_size: number
  cache_ttl: number
}

const AdvancedConfiguration: React.FC = () => {
  const { isConnected } = useWebSocket()
  const [moderationConfig, setModerationConfig] = useState<ModerationConfig | null>(null)
  const [ruleEngineConfig, setRuleEngineConfig] = useState<RuleEngineConfig | null>(null)
  const [loading, setLoading] = useState(false)
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const [activeTab, setActiveTab] = useState<'moderation' | 'rules' | 'layers' | 'pii' | 'relevancy' | 'regex' | 'performance'>('moderation')
  
  // Relevancy-specific state
  const [relevancyStats, setRelevancyStats] = useState<any>(null)
  const [testContent, setTestContent] = useState('')
  const [testResult, setTestResult] = useState<any>(null)
  const [newKeyword, setNewKeyword] = useState('')
  const [newKeywordScore, setNewKeywordScore] = useState(0.5)
  const [newKeywordCategory, setNewKeywordCategory] = useState('relevant')
  
  // AI Provider state (shared across layers)
  const [aiProviderConfig, setAiProviderConfig] = useState<any>(null)
  const [aiTestContent, setAiTestContent] = useState('')
  const [aiTestResult, setAiTestResult] = useState<any>(null)
  const [aiTestLoading, setAiTestLoading] = useState(false)

  // Regex layer state
  const [regexStats, setRegexStats] = useState<any>(null)
  const [regexTestContent, setRegexTestContent] = useState('')
  const [regexTestResult, setRegexTestResult] = useState<any>(null)
  const [regexTestLoading, setRegexTestLoading] = useState(false)
  const [newRegexPattern, setNewRegexPattern] = useState('')
  const [newRegexDescription, setNewRegexDescription] = useState('')
  const [newRegexSeverity, setNewRegexSeverity] = useState('medium')
  const [regexAiConfig, setRegexAiConfig] = useState<any>(null)

  // PII layer state
  const [piiStats, setPiiStats] = useState<any>(null)
  const [piiTestContent, setPiiTestContent] = useState('')
  const [piiTestResult, setPiiTestResult] = useState<any>(null)
  const [piiTestLoading, setPiiTestLoading] = useState(false)
  const [piiAiConfig, setPiiAiConfig] = useState<any>(null)

  useEffect(() => {
    fetchConfigurations()
  }, [])

  useEffect(() => {
    if (activeTab === 'relevancy' && !aiProviderConfig) {
      fetchAIProviderConfig()
    } else if (activeTab === 'regex' && !regexAiConfig) {
      fetchRegexAIConfig()
    } else if (activeTab === 'pii' && !piiAiConfig) {
      fetchPiiAIConfig()
    }
  }, [activeTab, aiProviderConfig, regexAiConfig, piiAiConfig])

  const fetchConfigurations = async () => {
    setLoading(true)
    try {
      const [moderationResponse, rulesResponse] = await Promise.all([
        fetch('/api/config/moderation'),
        fetch('/api/config/rules')
      ])

      if (moderationResponse.ok) {
        const data = await moderationResponse.json()
        setModerationConfig(data.data)
      }

      if (rulesResponse.ok) {
        const data = await rulesResponse.json()
        setRuleEngineConfig(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch configurations:', error)
    } finally {
      setLoading(false)
    }
  }

  const saveConfiguration = async () => {
    setSaveStatus('saving')
    try {
      const requests = []

      if (moderationConfig) {
        requests.push(
          fetch('/api/config/moderation', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(moderationConfig)
          })
        )
      }

      if (ruleEngineConfig) {
        requests.push(
          fetch('/api/config/rules', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(ruleEngineConfig)
          })
        )
      }

      const responses = await Promise.all(requests)
      const allSuccessful = responses.every(r => r.ok)

      if (allSuccessful) {
        setSaveStatus('saved')
        setTimeout(() => setSaveStatus('idle'), 3000)
      } else {
        setSaveStatus('error')
      }
    } catch (error) {
      console.error('Failed to save configuration:', error)
      setSaveStatus('error')
    }
  }

  const updateModerationConfig = (path: string, value: any) => {
    if (!moderationConfig) return

    const keys = path.split('.')
    const newConfig = { ...moderationConfig }
    let current: any = newConfig

    for (let i = 0; i < keys.length - 1; i++) {
      if (!(keys[i] in current)) current[keys[i]] = {}
      current = current[keys[i]]
    }

    current[keys[keys.length - 1]] = value
    setModerationConfig(newConfig)
  }

  const updateRuleEngineConfig = (key: string, value: any) => {
    if (!ruleEngineConfig) return
    setRuleEngineConfig({ ...ruleEngineConfig, [key]: value })
  }

  const updateLayerConfig = (layerIndex: number, key: string, value: any) => {
    if (!moderationConfig) return
    const newConfig = { ...moderationConfig }
    if (key.includes('.')) {
      const [parentKey, childKey] = key.split('.')
      newConfig.layers[layerIndex] = {
        ...newConfig.layers[layerIndex],
        [parentKey]: {
          ...(newConfig.layers[layerIndex][parentKey as keyof LayerConfig] as object),
          [childKey]: value
        }
      }
    } else {
      newConfig.layers[layerIndex] = {
        ...newConfig.layers[layerIndex],
        [key]: value
      }
    }
    setModerationConfig(newConfig)
  }

  const addLayer = () => {
    if (!moderationConfig) return
    const hasRelevancyLayer = moderationConfig.layers.some(layer => layer.name === 'relevancy')
    
    const newLayer: LayerConfig = hasRelevancyLayer ? {
      name: `custom_layer_${Date.now()}`,
      enabled: false,
      weight: 0.5,
      threshold: 0.5,
      options: {}
    } : {
      name: 'relevancy',
      enabled: true,
      weight: 0.3,
      threshold: 0.3,
      options: {
        relevant_keywords: ['programming', 'coding', 'software', 'ai', 'technology'],
        irrelevant_keywords: ['cooking', 'weather', 'sports', 'celebrity', 'gossip'],
        custom_keywords: {
          'artificial intelligence': 0.95,
          'machine learning': 0.9,
          'deep learning': 0.9,
          'data science': 0.85
        },
        keyword_weight: 0.6,
        pattern_weight: 0.3,
        context_weight: 0.1,
        cache_enabled: true,
        cache_ttl: 300,
        case_sensitive: false
      }
    }
    
    setModerationConfig({
      ...moderationConfig,
      layers: [...moderationConfig.layers, newLayer]
    })
  }

  const removeLayer = (index: number) => {
    if (!moderationConfig) return
    const newLayers = moderationConfig.layers.filter((_, i) => i !== index)
    setModerationConfig({ ...moderationConfig, layers: newLayers })
  }

  // Relevancy-specific functions
  const fetchRelevancyStats = async () => {
    try {
      const response = await fetch('/api/relevancy/stats')
      if (response.ok) {
        const data = await response.json()
        setRelevancyStats(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch relevancy stats:', error)
    }
  }

  const testRelevancy = async () => {
    if (!testContent.trim()) return
    
    try {
      const response = await fetch('/api/relevancy/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: testContent })
      })
      
      if (response.ok) {
        const data = await response.json()
        setTestResult(data.data)
      }
    } catch (error) {
      console.error('Failed to test relevancy:', error)
    }
  }

  const addKeyword = async (relevancyLayerIndex: number) => {
    if (!newKeyword.trim() || !moderationConfig) return
    
    try {
      const relevancyLayer = moderationConfig.layers[relevancyLayerIndex]
      const category = newKeywordCategory === 'relevant' ? 'relevant_keywords' : 'irrelevant_keywords'
      const currentKeywords = relevancyLayer.options[category] || []
      
      const response = await fetch('/api/relevancy/keywords', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          keyword: newKeyword,
          category: newKeywordCategory,
          score: newKeywordScore
        })
      })
      
      if (response.ok) {
        // Update local state
        if (newKeywordCategory === 'custom') {
          updateLayerConfig(relevancyLayerIndex, `options.custom_keywords.${newKeyword}`, newKeywordScore)
        } else {
          updateLayerConfig(relevancyLayerIndex, `options.${category}`, [...currentKeywords, newKeyword])
        }
        
        setNewKeyword('')
        setNewKeywordScore(0.5)
        fetchRelevancyStats()
      }
    } catch (error) {
      console.error('Failed to add keyword:', error)
    }
  }

  const removeKeyword = async (relevancyLayerIndex: number, keyword: string, category: string) => {
    if (!moderationConfig) return
    
    try {
      const response = await fetch(`/api/relevancy/keywords/${encodeURIComponent(keyword)}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ category })
      })
      
      if (response.ok) {
        const relevancyLayer = moderationConfig.layers[relevancyLayerIndex]
        
        if (category === 'custom') {
          const newCustomKeywords = { ...relevancyLayer.options.custom_keywords }
          delete newCustomKeywords[keyword]
          updateLayerConfig(relevancyLayerIndex, 'options.custom_keywords', newCustomKeywords)
        } else {
          const currentKeywords = relevancyLayer.options[category] || []
          const newKeywords = currentKeywords.filter((k: string) => k !== keyword)
          updateLayerConfig(relevancyLayerIndex, `options.${category}`, newKeywords)
        }
        
        fetchRelevancyStats()
      }
    } catch (error) {
      console.error('Failed to remove keyword:', error)
    }
  }

  // AI Provider functions (shared)
  const fetchAIProviderConfig = async () => {
    try {
      const response = await fetch('/api/relevancy-ai-provider')
      if (response.ok) {
        const data = await response.json()
        setAiProviderConfig(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch AI provider config:', error)
    }
  }

  const saveAIProviderConfig = async (config: any) => {
    try {
      const response = await fetch('/api/relevancy-ai-provider', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })
      
      if (response.ok) {
        const data = await response.json()
        setAiProviderConfig(data.data?.config || data.data)
      }
    } catch (error) {
      console.error('Failed to save AI provider config:', error)
    }
  }

  const testAIProvider = async () => {
    if (!aiTestContent.trim() || !aiProviderConfig) return
    
    setAiTestLoading(true)
    try {
      const response = await fetch('/api/relevancy-ai-test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content: aiTestContent,
          provider: aiProviderConfig.provider,
          model: aiProviderConfig.model,
          api_key: aiProviderConfig.api_key
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        setAiTestResult(data.data)
      }
    } catch (error) {
      console.error('Failed to test AI provider:', error)
    } finally {
      setAiTestLoading(false)
    }
  }

  const updateAIProviderConfig = (key: string, value: any) => {
    setAiProviderConfig((prev: any) => ({
      ...prev,
      [key]: value
    }))
  }

  // Regex layer functions
  const fetchRegexAIConfig = async () => {
    try {
      const response = await fetch('/api/regex-ai-provider')
      if (response.ok) {
        const data = await response.json()
        setRegexAiConfig(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch Regex AI provider config:', error)
    }
  }

  const saveRegexAIConfig = async (config: any) => {
    try {
      const response = await fetch('/api/regex-ai-provider', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })
      
      if (response.ok) {
        const data = await response.json()
        setRegexAiConfig(data.data?.config || data.data)
      }
    } catch (error) {
      console.error('Failed to save Regex AI provider config:', error)
    }
  }

  const updateRegexAIConfig = (key: string, value: any) => {
    setRegexAiConfig((prev: any) => ({
      ...prev,
      [key]: value
    }))
  }

  const fetchRegexStats = async () => {
    try {
      const response = await fetch('/api/regex-stats')
      if (response.ok) {
        const data = await response.json()
        setRegexStats(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch regex stats:', error)
    }
  }

  const testRegex = async () => {
    if (!regexTestContent.trim()) return
    
    setRegexTestLoading(true)
    try {
      const response = await fetch('/api/regex-ai-test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content: regexTestContent,
          provider: regexAiConfig?.provider || '',
          model: regexAiConfig?.model || '',
          api_key: regexAiConfig?.api_key || ''
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        setRegexTestResult(data.data)
      }
    } catch (error) {
      console.error('Failed to test regex:', error)
    } finally {
      setRegexTestLoading(false)
    }
  }

  const addRegexPattern = async (regexLayerIndex: number) => {
    if (!newRegexPattern.trim() || !moderationConfig) return
    
    try {
      const response = await fetch('/api/regex/patterns', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          pattern: newRegexPattern,
          description: newRegexDescription,
          severity: newRegexSeverity
        })
      })
      
      if (response.ok) {
        const regexLayer = moderationConfig.layers[regexLayerIndex]
        const currentPatterns = regexLayer.options.patterns || []
        updateLayerConfig(regexLayerIndex, 'options.patterns', [
          ...currentPatterns,
          {
            pattern: newRegexPattern,
            description: newRegexDescription,
            severity: newRegexSeverity,
            enabled: true
          }
        ])
        
        setNewRegexPattern('')
        setNewRegexDescription('')
        setNewRegexSeverity('medium')
        fetchRegexStats()
      }
    } catch (error) {
      console.error('Failed to add regex pattern:', error)
    }
  }

  // PII layer functions
  const fetchPiiAIConfig = async () => {
    try {
      const response = await fetch('/api/pii-ai-provider')
      if (response.ok) {
        const data = await response.json()
        setPiiAiConfig(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch PII AI provider config:', error)
    }
  }

  const savePiiAIConfig = async (config: any) => {
    try {
      const response = await fetch('/api/pii-ai-provider', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(config)
      })
      
      if (response.ok) {
        const data = await response.json()
        setPiiAiConfig(data.data?.config || data.data)
      }
    } catch (error) {
      console.error('Failed to save PII AI provider config:', error)
    }
  }

  const updatePiiAIConfig = (key: string, value: any) => {
    setPiiAiConfig((prev: any) => ({
      ...prev,
      [key]: value
    }))
  }

  const fetchPiiStats = async () => {
    try {
      const response = await fetch('/api/pii-stats')
      if (response.ok) {
        const data = await response.json()
        setPiiStats(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch PII stats:', error)
    }
  }

  const testPii = async () => {
    if (!piiTestContent.trim()) return
    
    setPiiTestLoading(true)
    try {
      const response = await fetch('/api/pii-ai-test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content: piiTestContent,
          provider: piiAiConfig?.provider || '',
          model: piiAiConfig?.model || '',
          api_key: piiAiConfig?.api_key || ''
        })
      })
      
      if (response.ok) {
        const data = await response.json()
        setPiiTestResult(data.data)
      }
    } catch (error) {
      console.error('Failed to test PII:', error)
    } finally {
      setPiiTestLoading(false)
    }
  }

  if (loading && !moderationConfig && !ruleEngineConfig) {
    return (
      <div className="flex items-center justify-center p-12">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        <span className="ml-4 text-slate-600">Loading configuration...</span>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-slate-900 mb-2">Advanced Configuration</h1>
            <p className="text-slate-600">Configure moderation engine, rule engine, and performance settings</p>
          </div>
          <div className="flex items-center space-x-3">
            <div className={`text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
              {isConnected ? '🟢 Live Config' : '🔴 Offline'}
            </div>
            <button
              onClick={saveConfiguration}
              disabled={saveStatus === 'saving'}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                saveStatus === 'saved' 
                  ? 'bg-green-600 text-white' 
                  : saveStatus === 'error'
                  ? 'bg-red-600 text-white'
                  : 'bg-blue-600 text-white hover:bg-blue-700'
              } disabled:opacity-50`}
            >
              {saveStatus === 'saving' ? 'Saving...' : 
               saveStatus === 'saved' ? '✓ Saved' :
               saveStatus === 'error' ? '✗ Error' : 'Save Changes'}
            </button>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200">
        <nav className="flex space-x-8">
          {[
            { key: 'moderation', label: 'Moderation Engine', icon: '🛡️' },
            { key: 'rules', label: 'Rule Engine', icon: '⚙️' },
            { key: 'layers', label: 'Layer Configuration', icon: '🏗️' },
            { key: 'relevancy', label: 'Relevancy Layer', icon: '🎯' },
            { key: 'regex', label: 'Regex Layer', icon: '🔍' },
            { key: 'pii', label: 'PII Detection', icon: '🔒' },
            { key: 'performance', label: 'Performance & Cache', icon: '⚡' }
          ].map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key as any)}
              className={`flex items-center space-x-2 py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.key
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              <span>{tab.icon}</span>
              <span>{tab.label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Moderation Engine Tab */}
      {activeTab === 'moderation' && moderationConfig && (
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">Moderation Engine Settings</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Engine Enabled
                </label>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={moderationConfig.enabled}
                    onChange={(e) => updateModerationConfig('enabled', e.target.checked)}
                    className="rounded"
                  />
                  <span className="ml-2 text-sm text-slate-600">
                    Enable content moderation pipeline
                  </span>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Analytics Enabled
                </label>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={moderationConfig.analytics.enabled}
                    onChange={(e) => updateModerationConfig('analytics.enabled', e.target.checked)}
                    className="rounded"
                  />
                  <span className="ml-2 text-sm text-slate-600">
                    Collect performance analytics
                  </span>
                </div>
              </div>
            </div>

            <div className="mt-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4">Score Thresholds</h3>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {Object.entries(moderationConfig.thresholds).map(([key, value]) => (
                  <div key={key}>
                    <label className="block text-sm font-medium text-slate-700 mb-1 capitalize">
                      {key} Threshold
                    </label>
                    <input
                      type="number"
                      min="0"
                      max="1"
                      step="0.01"
                      value={value}
                      onChange={(e) => updateModerationConfig(`thresholds.${key}`, parseFloat(e.target.value))}
                      className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                    />
                  </div>
                ))}
              </div>
            </div>

            <div className="mt-6">
              <h3 className="text-lg font-semibold text-slate-900 mb-4">Analytics Settings</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Sample Rate (0.0 - 1.0)
                  </label>
                  <input
                    type="number"
                    min="0"
                    max="1"
                    step="0.01"
                    value={moderationConfig.analytics.sample_rate}
                    onChange={(e) => updateModerationConfig('analytics.sample_rate', parseFloat(e.target.value))}
                    className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Retention Days
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="365"
                    value={moderationConfig.analytics.retention_days}
                    onChange={(e) => updateModerationConfig('analytics.retention_days', parseInt(e.target.value))}
                    className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Rule Engine Tab */}
      {activeTab === 'rules' && ruleEngineConfig && (
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">Rule Engine Configuration</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Rules File Path
                </label>
                <input
                  type="text"
                  value={ruleEngineConfig.rules_path}
                  onChange={(e) => updateRuleEngineConfig('rules_path', e.target.value)}
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                  placeholder="/path/to/rules.yaml"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Maximum Rules
                </label>
                <input
                  type="number"
                  min="1"
                  max="10000"
                  value={ruleEngineConfig.max_rules}
                  onChange={(e) => updateRuleEngineConfig('max_rules', parseInt(e.target.value))}
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Hot Reload
                </label>
                <div className="flex items-center">
                  <input
                    type="checkbox"
                    checked={ruleEngineConfig.hot_reload}
                    onChange={(e) => updateRuleEngineConfig('hot_reload', e.target.checked)}
                    className="rounded"
                  />
                  <span className="ml-2 text-sm text-slate-600">
                    Automatically reload rules when files change
                  </span>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Reload Interval (seconds)
                </label>
                <input
                  type="number"
                  min="1"
                  max="3600"
                  value={ruleEngineConfig.reload_interval}
                  onChange={(e) => updateRuleEngineConfig('reload_interval', parseInt(e.target.value))}
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Default Weight (0.0 - 1.0)
                </label>
                <input
                  type="number"
                  min="0"
                  max="1"
                  step="0.01"
                  value={ruleEngineConfig.default_weight}
                  onChange={(e) => updateRuleEngineConfig('default_weight', parseFloat(e.target.value))}
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Default Priority (1-10)
                </label>
                <input
                  type="number"
                  min="1"
                  max="10"
                  value={ruleEngineConfig.default_priority}
                  onChange={(e) => updateRuleEngineConfig('default_priority', parseInt(e.target.value))}
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                />
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Layers Tab */}
      {activeTab === 'layers' && moderationConfig && (
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold text-slate-900">Layer Configuration</h2>
              <button
                onClick={addLayer}
                className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition-colors text-sm"
              >
                {moderationConfig.layers.some(layer => layer.name === 'relevancy') 
                  ? '➕ Add Custom Layer' 
                  : '🎯 Add Relevancy Layer'
                }
              </button>
            </div>
            
            <div className="space-y-4">
              {moderationConfig.layers.map((layer, index) => (
                <div key={index} className="border border-slate-200 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-semibold text-slate-900">{layer.name}</h3>
                    <div className="flex items-center space-x-2">
                      <label className="flex items-center">
                        <input
                          type="checkbox"
                          checked={layer.enabled}
                          onChange={(e) => updateLayerConfig(index, 'enabled', e.target.checked)}
                          className="rounded"
                        />
                        <span className="ml-1 text-sm">Enabled</span>
                      </label>
                      <button
                        onClick={() => removeLayer(index)}
                        className="bg-red-100 text-red-800 px-2 py-1 rounded text-sm hover:bg-red-200"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                  
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Weight (0.0 - 1.0)
                      </label>
                      <input
                        type="number"
                        min="0"
                        max="1"
                        step="0.01"
                        value={layer.weight}
                        onChange={(e) => updateLayerConfig(index, 'weight', parseFloat(e.target.value))}
                        className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                      />
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Threshold (0.0 - 1.0)
                      </label>
                      <input
                        type="number"
                        min="0"
                        max="1"
                        step="0.01"
                        value={layer.threshold}
                        onChange={(e) => updateLayerConfig(index, 'threshold', parseFloat(e.target.value))}
                        className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                      />
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">
                        Layer Name
                      </label>
                      <input
                        type="text"
                        value={layer.name}
                        onChange={(e) => updateLayerConfig(index, 'name', e.target.value)}
                        className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                      />
                    </div>
                  </div>

                  {/* Layer-specific options */}
                  {Object.keys(layer.options).length > 0 && (
                    <div className="mt-4 pt-4 border-t border-slate-200">
                      <div className="flex items-center justify-between mb-2">
                        <h4 className="font-medium text-slate-700">Layer Options</h4>
                        {layer.name === 'relevancy' && (
                          <span className="text-xs text-purple-600 bg-purple-100 px-2 py-1 rounded-full">
                            🎯 See Relevancy tab for advanced configuration
                          </span>
                        )}
                      </div>
                      
                      {layer.name === 'relevancy' ? (
                        // Special UI for relevancy layer
                        <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                            <div>
                              <div className="font-medium text-purple-800 mb-1">Relevant Keywords</div>
                              <div className="text-purple-600">
                                {layer.options.relevant_keywords?.length || 0} configured
                              </div>
                              <div className="flex flex-wrap gap-1 mt-1">
                                {(layer.options.relevant_keywords || []).slice(0, 3).map((kw: string, i: number) => (
                                  <span key={i} className="bg-green-100 text-green-800 px-1 py-0.5 rounded text-xs">
                                    {kw}
                                  </span>
                                ))}
                                {(layer.options.relevant_keywords?.length || 0) > 3 && (
                                  <span className="text-purple-600 text-xs">
                                    +{(layer.options.relevant_keywords?.length || 0) - 3} more
                                  </span>
                                )}
                              </div>
                            </div>
                            
                            <div>
                              <div className="font-medium text-purple-800 mb-1">Irrelevant Keywords</div>
                              <div className="text-purple-600">
                                {layer.options.irrelevant_keywords?.length || 0} configured
                              </div>
                              <div className="flex flex-wrap gap-1 mt-1">
                                {(layer.options.irrelevant_keywords || []).slice(0, 3).map((kw: string, i: number) => (
                                  <span key={i} className="bg-red-100 text-red-800 px-1 py-0.5 rounded text-xs">
                                    {kw}
                                  </span>
                                ))}
                                {(layer.options.irrelevant_keywords?.length || 0) > 3 && (
                                  <span className="text-purple-600 text-xs">
                                    +{(layer.options.irrelevant_keywords?.length || 0) - 3} more
                                  </span>
                                )}
                              </div>
                            </div>
                            
                            <div>
                              <div className="font-medium text-purple-800 mb-1">Custom Keywords</div>
                              <div className="text-purple-600">
                                {Object.keys(layer.options.custom_keywords || {}).length} configured
                              </div>
                            </div>
                            
                            <div>
                              <div className="font-medium text-purple-800 mb-1">Quick Actions</div>
                              <button
                                onClick={() => setActiveTab('relevancy')}
                                className="bg-purple-600 text-white px-3 py-1 rounded text-xs hover:bg-purple-700 transition-colors"
                              >
                                Configure Relevancy →
                              </button>
                            </div>
                          </div>
                        </div>
                      ) : (
                        // Standard UI for other layers
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                          {Object.entries(layer.options).map(([key, value]) => (
                            <div key={key}>
                              <label className="block text-sm text-slate-600 mb-1">{key}</label>
                              <input
                                type="text"
                                value={String(value)}
                                onChange={(e) => {
                                  const newOptions = { ...layer.options, [key]: e.target.value }
                                  updateLayerConfig(index, 'options', newOptions)
                                }}
                                className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                              />
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* PII Detection Tab */}
      {activeTab === 'pii' && moderationConfig && (
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">🔒 PII Detection Configuration</h2>
            
            {/* Find PII Layer */}
            {(() => {
              const piiLayer = moderationConfig.layers.find(layer => layer.name === 'pii')
              if (!piiLayer) {
                return (
                  <div className="text-center py-12 text-slate-500">
                    <span className="text-4xl mb-4 block">🔍</span>
                    <h3 className="text-lg font-medium mb-2">PII Layer Not Found</h3>
                    <p className="text-sm">
                      The PII detection layer is not configured. Please add it to your moderation layers.
                    </p>
                  </div>
                )
              }

              const piiLayerIndex = moderationConfig.layers.findIndex(layer => layer.name === 'pii')
              
              return (
                <div className="space-y-8">
                  {/* PII Layer Status */}
                  <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-lg p-6">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="text-lg font-semibold text-slate-900">PII Layer Status</h3>
                      <div className="flex items-center space-x-4">
                        <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                          piiLayer.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {piiLayer.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                        <button
                          onClick={() => updateLayerConfig(piiLayerIndex, 'enabled', !piiLayer.enabled)}
                          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                            piiLayer.enabled 
                              ? 'bg-red-600 text-white hover:bg-red-700' 
                              : 'bg-green-600 text-white hover:bg-green-700'
                          }`}
                        >
                          {piiLayer.enabled ? 'Disable' : 'Enable'}
                        </button>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                          Weight
                        </label>
                        <input
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={piiLayer.weight}
                          onChange={(e) => updateLayerConfig(piiLayerIndex, 'weight', parseFloat(e.target.value))}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                          Threshold
                        </label>
                        <input
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={piiLayer.threshold}
                          onChange={(e) => updateLayerConfig(piiLayerIndex, 'threshold', parseFloat(e.target.value))}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>
                      <div className="flex items-end">
                        <div className="text-sm text-slate-600">
                          <div>Score threshold for blocking content</div>
                          <div className="text-xs text-slate-500 mt-1">
                            Current: {(piiLayer.threshold * 100).toFixed(0)}%
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* PII Detection Settings */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">Detection Types</h3>
                      <div className="space-y-4">
                        {[
                          { key: 'detect_email', label: 'Email Addresses', icon: '📧', confidence: 'email_confidence' },
                          { key: 'detect_phone', label: 'Phone Numbers', icon: '📞', confidence: 'phone_confidence' },
                          { key: 'detect_ssn', label: 'Social Security Numbers', icon: '🆔', confidence: 'ssn_confidence' },
                          { key: 'detect_credit_card', label: 'Credit Card Numbers', icon: '💳', confidence: 'credit_card_confidence' }
                        ].map((type) => (
                          <div key={type.key} className="border border-slate-200 rounded-lg p-4">
                            <div className="flex items-center justify-between mb-3">
                              <div className="flex items-center space-x-2">
                                <span>{type.icon}</span>
                                <span className="font-medium text-slate-900">{type.label}</span>
                              </div>
                              <input
                                type="checkbox"
                                checked={piiLayer.options?.[type.key] ?? true}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, `options.${type.key}`, e.target.checked)}
                                className="rounded"
                              />
                            </div>
                            <div>
                              <label className="block text-xs font-medium text-slate-600 mb-1">
                                Confidence Threshold
                              </label>
                              <input
                                type="number"
                                min="0"
                                max="1"
                                step="0.01"
                                value={piiLayer.options?.[type.confidence] ?? 0.9}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, `options.${type.confidence}`, parseFloat(e.target.value))}
                                className="w-full border border-slate-300 rounded px-2 py-1 text-xs"
                                disabled={!piiLayer.options?.[type.key]}
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">Privacy & Security</h3>
                      <div className="space-y-4">
                        <div className="border border-slate-200 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-3">
                            <div>
                              <div className="font-medium text-slate-900">PII Masking</div>
                              <div className="text-sm text-slate-600">
                                Automatically mask detected PII in logs and responses
                              </div>
                            </div>
                            <input
                              type="checkbox"
                              checked={piiLayer.options?.masking_enabled ?? true}
                              onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.masking_enabled', e.target.checked)}
                              className="rounded"
                            />
                          </div>
                        </div>

                        <div className="border border-slate-200 rounded-lg p-4">
                          <div className="font-medium text-slate-900 mb-3">Custom Patterns</div>
                          <div className="text-sm text-slate-600 mb-3">
                            Add custom regex patterns for organization-specific PII
                          </div>
                          <textarea
                            placeholder="Enter custom regex patterns (one per line)"
                            value={piiLayer.options?.custom_patterns?.join('\n') ?? ''}
                            onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.custom_patterns', e.target.value.split('\n').filter(p => p.trim()))}
                            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            rows={4}
                          />
                        </div>

                        <div className="border border-slate-200 rounded-lg p-4">
                          <div className="font-medium text-slate-900 mb-3">Alert Settings</div>
                          <div className="space-y-3">
                            <div className="flex items-center justify-between">
                              <span className="text-sm text-slate-700">Real-time alerts for PII detection</span>
                              <input
                                type="checkbox"
                                checked={piiLayer.options?.realtime_alerts ?? true}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.realtime_alerts', e.target.checked)}
                                className="rounded"
                              />
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-sm text-slate-700">Email notifications for high-risk PII</span>
                              <input
                                type="checkbox"
                                checked={piiLayer.options?.email_alerts ?? false}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.email_alerts', e.target.checked)}
                                className="rounded"
                              />
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })()}
          </div>
        </div>
      )}

      {/* Relevancy Tab */}
      {activeTab === 'relevancy' && moderationConfig && (
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">🎯 Relevancy Layer Configuration</h2>
            
            {/* Find Relevancy Layer */}
            {(() => {
              const relevancyLayer = moderationConfig.layers.find(layer => layer.name === 'relevancy')
              if (!relevancyLayer) {
                return (
                  <div className="text-center py-12 text-slate-500">
                    <span className="text-4xl mb-4 block">🎯</span>
                    <h3 className="text-lg font-medium mb-2">Relevancy Layer Not Found</h3>
                    <p className="text-sm mb-4">
                      The relevancy layer is not configured. Add it to start filtering content by relevance.
                    </p>
                    <button
                      onClick={() => {
                        const newLayer: LayerConfig = {
                          name: 'relevancy',
                          enabled: true,
                          weight: 0.3,
                          threshold: 0.3,
                          options: {
                            relevant_keywords: ['programming', 'coding', 'software', 'ai', 'technology'],
                            irrelevant_keywords: ['cooking', 'weather', 'sports', 'celebrity', 'gossip'],
                            custom_keywords: {
                              'artificial intelligence': 0.95,
                              'machine learning': 0.9,
                              'deep learning': 0.9,
                              'data science': 0.85
                            }
                          }
                        }
                        setModerationConfig({
                          ...moderationConfig,
                          layers: [...moderationConfig.layers, newLayer]
                        })
                      }}
                      className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition-colors"
                    >
                      Add Relevancy Layer
                    </button>
                  </div>
                )
              }

              const relevancyLayerIndex = moderationConfig.layers.findIndex(layer => layer.name === 'relevancy')
              
              return (
                <div className="space-y-8">
                  {/* Relevancy Layer Status & Controls */}
                  <div className="bg-gradient-to-r from-purple-50 to-blue-50 border border-purple-200 rounded-lg p-6">
                    <div className="flex items-center justify-between mb-6">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-900">Relevancy Layer Status</h3>
                        <p className="text-sm text-slate-600">Configure content relevance filtering and scoring</p>
                      </div>
                      <div className="flex items-center space-x-4">
                        <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                          relevancyLayer.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {relevancyLayer.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                        <button
                          onClick={() => updateLayerConfig(relevancyLayerIndex, 'enabled', !relevancyLayer.enabled)}
                          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                            relevancyLayer.enabled 
                              ? 'bg-red-600 text-white hover:bg-red-700' 
                              : 'bg-green-600 text-white hover:bg-green-700'
                          }`}
                        >
                          {relevancyLayer.enabled ? 'Disable' : 'Enable'}
                        </button>
                      </div>
                    </div>
                    
                    {/* Core Settings with Sliders */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <label className="block text-sm font-medium text-slate-700">
                            Layer Weight
                          </label>
                          <span className="text-sm text-purple-600 font-medium">
                            {(relevancyLayer.weight * 100).toFixed(0)}%
                          </span>
                        </div>
                        <input
                          type="range"
                          min="0"
                          max="1"
                          step="0.05"
                          value={relevancyLayer.weight}
                          onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'weight', parseFloat(e.target.value))}
                          className="w-full h-2 bg-purple-200 rounded-lg appearance-none cursor-pointer slider"
                        />
                      </div>
                      
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <label className="block text-sm font-medium text-slate-700">
                            Blocking Threshold
                          </label>
                          <span className="text-sm text-purple-600 font-medium">
                            {(relevancyLayer.threshold * 100).toFixed(0)}%
                          </span>
                        </div>
                        <input
                          type="range"
                          min="0"
                          max="1"
                          step="0.05"
                          value={relevancyLayer.threshold}
                          onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'threshold', parseFloat(e.target.value))}
                          className="w-full h-2 bg-purple-200 rounded-lg appearance-none cursor-pointer slider"
                        />
                        <p className="text-xs text-slate-500 mt-1">
                          Content below this score will be blocked
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Real-time Testing */}
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <h3 className="text-lg font-semibold text-slate-900 mb-4">🔬 Real-time Content Testing</h3>
                    <div className="space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                          Test Content
                        </label>
                        <textarea
                          value={testContent}
                          onChange={(e) => setTestContent(e.target.value)}
                          placeholder="Enter content to test relevancy scoring..."
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                          rows={3}
                        />
                      </div>
                      
                      <div className="flex items-center space-x-3">
                        <button
                          onClick={testRelevancy}
                          disabled={!testContent.trim()}
                          className="bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          Test Relevancy
                        </button>
                        
                        {testResult && (
                          <div className="flex items-center space-x-4">
                            <div className={`px-3 py-1 rounded-full text-sm font-medium ${
                              testResult.score >= relevancyLayer.threshold 
                                ? 'bg-green-100 text-green-800' 
                                : 'bg-red-100 text-red-800'
                            }`}>
                              {testResult.score >= relevancyLayer.threshold ? 'Relevant' : 'Blocked'}
                            </div>
                            <div className="text-sm text-slate-600">
                              Score: <span className="font-medium">{(testResult.score * 100).toFixed(1)}%</span>
                            </div>
                          </div>
                        )}
                      </div>

                      {testResult && (
                        <div className="bg-slate-50 rounded-lg p-4 text-sm">
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <div>
                              <div className="font-medium text-slate-700">Keyword Score</div>
                              <div className="text-purple-600">{(testResult.keyword_score * 100).toFixed(1)}%</div>
                            </div>
                            <div>
                              <div className="font-medium text-slate-700">Pattern Score</div>
                              <div className="text-purple-600">{(testResult.pattern_score * 100).toFixed(1)}%</div>
                            </div>
                            <div>
                              <div className="font-medium text-slate-700">Context Score</div>
                              <div className="text-purple-600">{(testResult.context_score * 100).toFixed(1)}%</div>
                            </div>
                          </div>
                          {testResult.matched_keywords && testResult.matched_keywords.length > 0 && (
                            <div className="mt-3">
                              <div className="font-medium text-slate-700 mb-1">Matched Keywords:</div>
                              <div className="flex flex-wrap gap-2">
                                {testResult.matched_keywords.map((keyword: string, i: number) => (
                                  <span key={i} className="bg-purple-100 text-purple-800 px-2 py-1 rounded text-xs">
                                    {keyword}
                                  </span>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Keyword Management */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                    {/* Keyword Categories */}
                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">📝 Keyword Categories</h3>
                      <div className="space-y-4">
                        {/* Relevant Keywords */}
                        <div className="border border-green-200 rounded-lg p-4 bg-green-50">
                          <div className="flex items-center justify-between mb-3">
                            <h4 className="font-medium text-green-800">✅ Relevant Keywords</h4>
                            <span className="text-xs text-green-600">
                              {relevancyLayer.options?.relevant_keywords?.length || 0} keywords
                            </span>
                          </div>
                          <div className="flex flex-wrap gap-2 max-h-24 overflow-y-auto">
                            {(relevancyLayer.options?.relevant_keywords || []).map((keyword: string, i: number) => (
                              <span 
                                key={i} 
                                className="bg-green-100 text-green-800 px-2 py-1 rounded-full text-xs flex items-center space-x-1"
                              >
                                <span>{keyword}</span>
                                <button
                                  onClick={() => removeKeyword(relevancyLayerIndex, keyword, 'relevant')}
                                  className="text-green-600 hover:text-green-800 ml-1"
                                >
                                  ×
                                </button>
                              </span>
                            ))}
                          </div>
                        </div>

                        {/* Irrelevant Keywords */}
                        <div className="border border-red-200 rounded-lg p-4 bg-red-50">
                          <div className="flex items-center justify-between mb-3">
                            <h4 className="font-medium text-red-800">❌ Irrelevant Keywords</h4>
                            <span className="text-xs text-red-600">
                              {relevancyLayer.options?.irrelevant_keywords?.length || 0} keywords
                            </span>
                          </div>
                          <div className="flex flex-wrap gap-2 max-h-24 overflow-y-auto">
                            {(relevancyLayer.options?.irrelevant_keywords || []).map((keyword: string, i: number) => (
                              <span 
                                key={i} 
                                className="bg-red-100 text-red-800 px-2 py-1 rounded-full text-xs flex items-center space-x-1"
                              >
                                <span>{keyword}</span>
                                <button
                                  onClick={() => removeKeyword(relevancyLayerIndex, keyword, 'irrelevant')}
                                  className="text-red-600 hover:text-red-800 ml-1"
                                >
                                  ×
                                </button>
                              </span>
                            ))}
                          </div>
                        </div>

                        {/* Custom Scored Keywords */}
                        <div className="border border-purple-200 rounded-lg p-4 bg-purple-50">
                          <div className="flex items-center justify-between mb-3">
                            <h4 className="font-medium text-purple-800">⚡ Custom Scored Keywords</h4>
                            <span className="text-xs text-purple-600">
                              {Object.keys(relevancyLayer.options?.custom_keywords || {}).length} keywords
                            </span>
                          </div>
                          <div className="space-y-2 max-h-32 overflow-y-auto">
                            {Object.entries(relevancyLayer.options?.custom_keywords || {}).map(([keyword, score], i) => (
                              <div key={i} className="flex items-center justify-between bg-purple-100 rounded px-2 py-1">
                                <span className="text-purple-800 text-sm">{keyword}</span>
                                <div className="flex items-center space-x-2">
                                  <span className="text-xs text-purple-600 font-medium">
                                    {((score as number) * 100).toFixed(0)}%
                                  </span>
                                  <button
                                    onClick={() => removeKeyword(relevancyLayerIndex, keyword, 'custom')}
                                    className="text-purple-600 hover:text-purple-800"
                                  >
                                    ×
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Add New Keywords */}
                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">➕ Add Keywords</h3>
                      <div className="bg-white border border-slate-200 rounded-lg p-4">
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-slate-700 mb-2">
                              Keyword
                            </label>
                            <input
                              type="text"
                              value={newKeyword}
                              onChange={(e) => setNewKeyword(e.target.value)}
                              placeholder="Enter keyword..."
                              className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            />
                          </div>

                          <div>
                            <label className="block text-sm font-medium text-slate-700 mb-2">
                              Category
                            </label>
                            <select
                              value={newKeywordCategory}
                              onChange={(e) => setNewKeywordCategory(e.target.value)}
                              className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            >
                              <option value="relevant">Relevant (Standard)</option>
                              <option value="irrelevant">Irrelevant (Standard)</option>
                              <option value="custom">Custom Score</option>
                            </select>
                          </div>

                          {newKeywordCategory === 'custom' && (
                            <div>
                              <div className="flex items-center justify-between mb-2">
                                <label className="block text-sm font-medium text-slate-700">
                                  Custom Score
                                </label>
                                <span className="text-sm text-purple-600 font-medium">
                                  {(newKeywordScore * 100).toFixed(0)}%
                                </span>
                              </div>
                              <input
                                type="range"
                                min="0"
                                max="1"
                                step="0.05"
                                value={newKeywordScore}
                                onChange={(e) => setNewKeywordScore(parseFloat(e.target.value))}
                                className="w-full h-2 bg-purple-200 rounded-lg appearance-none cursor-pointer slider"
                              />
                            </div>
                          )}

                          <button
                            onClick={() => addKeyword(relevancyLayerIndex)}
                            disabled={!newKeyword.trim()}
                            className="w-full bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            Add Keyword
                          </button>
                        </div>
                      </div>

                      {/* Relevancy Statistics */}
                      <div className="mt-6 bg-slate-50 border border-slate-200 rounded-lg p-4">
                        <div className="flex items-center justify-between mb-3">
                          <h4 className="font-medium text-slate-900">📊 Statistics</h4>
                          <button
                            onClick={fetchRelevancyStats}
                            className="text-sm text-purple-600 hover:text-purple-800"
                          >
                            Refresh
                          </button>
                        </div>
                        {relevancyStats ? (
                          <div className="grid grid-cols-2 gap-4 text-sm">
                            <div>
                              <div className="text-slate-600">Requests Processed</div>
                              <div className="font-medium text-slate-900">{relevancyStats.requests_processed || 0}</div>
                            </div>
                            <div>
                              <div className="text-slate-600">Content Blocked</div>
                              <div className="font-medium text-red-600">{relevancyStats.content_blocked || 0}</div>
                            </div>
                            <div>
                              <div className="text-slate-600">Avg Score</div>
                              <div className="font-medium text-green-600">
                                {relevancyStats.average_score ? (relevancyStats.average_score * 100).toFixed(1) + '%' : 'N/A'}
                              </div>
                            </div>
                            <div>
                              <div className="text-slate-600">Cache Hit Rate</div>
                              <div className="font-medium text-blue-600">
                                {relevancyStats.cache_hit_rate ? (relevancyStats.cache_hit_rate * 100).toFixed(1) + '%' : 'N/A'}
                              </div>
                            </div>
                          </div>
                        ) : (
                          <div className="text-sm text-slate-500">Click refresh to load statistics</div>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Advanced Settings */}
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <h3 className="text-lg font-semibold text-slate-900 mb-4">⚙️ Advanced Settings</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <div>
                        <h4 className="font-medium text-slate-700 mb-3">Scoring Weights</h4>
                        <div className="space-y-3">
                          <div>
                            <div className="flex items-center justify-between mb-1">
                              <label className="text-sm text-slate-600">Keyword Weight</label>
                              <span className="text-sm text-purple-600">
                                {((relevancyLayer.options?.keyword_weight || 0.6) * 100).toFixed(0)}%
                              </span>
                            </div>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.05"
                              value={relevancyLayer.options?.keyword_weight || 0.6}
                              onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'options.keyword_weight', parseFloat(e.target.value))}
                              className="w-full h-2 bg-purple-200 rounded-lg appearance-none cursor-pointer slider"
                            />
                          </div>
                          
                          <div>
                            <div className="flex items-center justify-between mb-1">
                              <label className="text-sm text-slate-600">Pattern Weight</label>
                              <span className="text-sm text-purple-600">
                                {((relevancyLayer.options?.pattern_weight || 0.3) * 100).toFixed(0)}%
                              </span>
                            </div>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.05"
                              value={relevancyLayer.options?.pattern_weight || 0.3}
                              onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'options.pattern_weight', parseFloat(e.target.value))}
                              className="w-full h-2 bg-purple-200 rounded-lg appearance-none cursor-pointer slider"
                            />
                          </div>
                          
                          <div>
                            <div className="flex items-center justify-between mb-1">
                              <label className="text-sm text-slate-600">Context Weight</label>
                              <span className="text-sm text-purple-600">
                                {((relevancyLayer.options?.context_weight || 0.1) * 100).toFixed(0)}%
                              </span>
                            </div>
                            <input
                              type="range"
                              min="0"
                              max="1"
                              step="0.05"
                              value={relevancyLayer.options?.context_weight || 0.1}
                              onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'options.context_weight', parseFloat(e.target.value))}
                              className="w-full h-2 bg-purple-200 rounded-lg appearance-none cursor-pointer slider"
                            />
                          </div>
                        </div>
                      </div>

                      <div>
                        <h4 className="font-medium text-slate-700 mb-3">Performance Settings</h4>
                        <div className="space-y-4">
                          <div className="flex items-center justify-between">
                            <span className="text-sm text-slate-700">Enable Caching</span>
                            <input
                              type="checkbox"
                              checked={relevancyLayer.options?.cache_enabled ?? true}
                              onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'options.cache_enabled', e.target.checked)}
                              className="rounded"
                            />
                          </div>
                          
                          <div>
                            <label className="block text-sm text-slate-700 mb-1">
                              Cache TTL (seconds)
                            </label>
                            <input
                              type="number"
                              min="60"
                              max="3600"
                              value={relevancyLayer.options?.cache_ttl || 300}
                              onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'options.cache_ttl', parseInt(e.target.value))}
                              className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                            />
                          </div>
                          
                          <div className="flex items-center justify-between">
                            <span className="text-sm text-slate-700">Case Sensitive</span>
                            <input
                              type="checkbox"
                              checked={relevancyLayer.options?.case_sensitive ?? false}
                              onChange={(e) => updateLayerConfig(relevancyLayerIndex, 'options.case_sensitive', e.target.checked)}
                              className="rounded"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* AI Provider Configuration */}
                  {aiProviderConfig && (
                    <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-lg p-6">
                      <div className="flex items-center justify-between mb-6">
                        <div>
                          <h3 className="text-lg font-semibold text-slate-900">🤖 AI-Powered Relevancy</h3>
                          <p className="text-sm text-slate-600">Enhance relevancy detection with Large Language Models</p>
                        </div>
                        <div className="flex items-center space-x-4">
                          <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                            aiProviderConfig.ai_enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                          }`}>
                            {aiProviderConfig.ai_enabled ? 'AI Enabled' : 'Local Only'}
                          </span>
                          <button
                            onClick={() => {
                              const newConfig = { ...aiProviderConfig, ai_enabled: !aiProviderConfig.ai_enabled }
                              updateAIProviderConfig('ai_enabled', !aiProviderConfig.ai_enabled)
                              saveAIProviderConfig(newConfig)
                            }}
                            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                              aiProviderConfig.ai_enabled 
                                ? 'bg-gray-600 text-white hover:bg-gray-700' 
                                : 'bg-blue-600 text-white hover:bg-blue-700'
                            }`}
                          >
                            {aiProviderConfig.ai_enabled ? 'Disable AI' : 'Enable AI'}
                          </button>
                        </div>
                      </div>

                      {aiProviderConfig.ai_enabled && (
                        <div className="space-y-6">
                          {/* Provider Configuration */}
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div>
                              <h4 className="font-medium text-slate-700 mb-3">🔧 Provider Settings</h4>
                              <div className="space-y-4">
                                <div>
                                  <label className="block text-sm font-medium text-slate-700 mb-1">
                                    AI Provider
                                  </label>
                                  <select
                                    value={aiProviderConfig.provider}
                                    onChange={(e) => updateAIProviderConfig('provider', e.target.value)}
                                    className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                  >
                                    <option value="">Select Provider</option>
                                    <option value="openai">OpenAI</option>
                                    <option value="anthropic">Anthropic</option>
                                    <option value="local">Local Model</option>
                                    <option value="custom">Custom Endpoint</option>
                                  </select>
                                </div>

                                <div>
                                  <label className="block text-sm font-medium text-slate-700 mb-1">
                                    Model
                                  </label>
                                  <select
                                    value={aiProviderConfig.model}
                                    onChange={(e) => updateAIProviderConfig('model', e.target.value)}
                                    className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                  >
                                    <option value="">Select Model</option>
                                    {aiProviderConfig.provider === 'openai' && (
                                      <>
                                        <option value="gpt-4">GPT-4</option>
                                        <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                                      </>
                                    )}
                                    {aiProviderConfig.provider === 'anthropic' && (
                                      <>
                                        <option value="claude-3-sonnet">Claude 3 Sonnet</option>
                                        <option value="claude-3-haiku">Claude 3 Haiku</option>
                                      </>
                                    )}
                                    {aiProviderConfig.provider === 'local' && (
                                      <>
                                        <option value="llama-2">Llama 2</option>
                                        <option value="mistral">Mistral</option>
                                      </>
                                    )}
                                  </select>
                                </div>

                                <div>
                                  <label className="block text-sm font-medium text-slate-700 mb-1">
                                    API Key
                                  </label>
                                  <input
                                    type="password"
                                    value={aiProviderConfig.api_key || ''}
                                    onChange={(e) => updateAIProviderConfig('api_key', e.target.value)}
                                    placeholder="Enter API key..."
                                    className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                  />
                                  {aiProviderConfig.api_key_set && (
                                    <p className="text-xs text-green-600 mt-1">✓ API key is configured</p>
                                  )}
                                </div>

                                {aiProviderConfig.provider === 'custom' && (
                                  <div>
                                    <label className="block text-sm font-medium text-slate-700 mb-1">
                                      Custom Endpoint
                                    </label>
                                    <input
                                      type="url"
                                      value={aiProviderConfig.endpoint || ''}
                                      onChange={(e) => updateAIProviderConfig('endpoint', e.target.value)}
                                      placeholder="https://api.example.com/v1/chat/completions"
                                      className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                    />
                                  </div>
                                )}
                              </div>
                            </div>

                            <div>
                              <h4 className="font-medium text-slate-700 mb-3">⚡ Model Parameters</h4>
                              <div className="space-y-4">
                                <div>
                                  <div className="flex items-center justify-between mb-1">
                                    <label className="text-sm text-slate-600">Temperature</label>
                                    <span className="text-sm text-blue-600 font-medium">
                                      {aiProviderConfig.temperature?.toFixed(1) || '0.1'}
                                    </span>
                                  </div>
                                  <input
                                    type="range"
                                    min="0"
                                    max="1"
                                    step="0.1"
                                    value={aiProviderConfig.temperature || 0.1}
                                    onChange={(e) => updateAIProviderConfig('temperature', parseFloat(e.target.value))}
                                    className="w-full h-2 bg-blue-200 rounded-lg appearance-none cursor-pointer slider"
                                  />
                                  <p className="text-xs text-slate-500 mt-1">Lower = more focused, Higher = more creative</p>
                                </div>

                                <div>
                                  <label className="block text-sm text-slate-600 mb-1">
                                    Max Tokens
                                  </label>
                                  <input
                                    type="number"
                                    min="50"
                                    max="4096"
                                    value={aiProviderConfig.max_tokens || 2048}
                                    onChange={(e) => updateAIProviderConfig('max_tokens', parseInt(e.target.value))}
                                    className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                                  />
                                </div>

                                <div>
                                  <div className="flex items-center justify-between mb-2">
                                    <span className="text-sm text-slate-700">Hybrid Mode</span>
                                    <input
                                      type="checkbox"
                                      checked={aiProviderConfig.hybrid_mode ?? true}
                                      onChange={(e) => updateAIProviderConfig('hybrid_mode', e.target.checked)}
                                      className="rounded"
                                    />
                                  </div>
                                  <p className="text-xs text-slate-500">Combine AI and local keyword scoring</p>
                                </div>

                                {aiProviderConfig.hybrid_mode && (
                                  <div>
                                    <div className="flex items-center justify-between mb-1">
                                      <label className="text-sm text-slate-600">AI Weight</label>
                                      <span className="text-sm text-blue-600 font-medium">
                                        {((aiProviderConfig.ai_weight || 0.7) * 100).toFixed(0)}%
                                      </span>
                                    </div>
                                    <input
                                      type="range"
                                      min="0"
                                      max="1"
                                      step="0.05"
                                      value={aiProviderConfig.ai_weight || 0.7}
                                      onChange={(e) => updateAIProviderConfig('ai_weight', parseFloat(e.target.value))}
                                      className="w-full h-2 bg-blue-200 rounded-lg appearance-none cursor-pointer slider"
                                    />
                                    <p className="text-xs text-slate-500 mt-1">
                                      AI: {((aiProviderConfig.ai_weight || 0.7) * 100).toFixed(0)}% | 
                                      Local: {((1 - (aiProviderConfig.ai_weight || 0.7)) * 100).toFixed(0)}%
                                    </p>
                                  </div>
                                )}
                              </div>
                            </div>
                          </div>

                          {/* System Prompt */}
                          <div>
                            <h4 className="font-medium text-slate-700 mb-3">💬 System Prompt</h4>
                            <textarea
                              value={aiProviderConfig.system_prompt || ''}
                              onChange={(e) => updateAIProviderConfig('system_prompt', e.target.value)}
                              placeholder="Enter system prompt for the AI model..."
                              className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                              rows={4}
                            />
                            <p className="text-xs text-slate-500 mt-1">
                              Define how the AI should analyze content for relevancy
                            </p>
                          </div>

                          {/* AI Testing */}
                          <div className="bg-white border border-slate-200 rounded-lg p-4">
                            <h4 className="font-medium text-slate-700 mb-3">🧪 Test AI Provider</h4>
                            <div className="space-y-4">
                              <div>
                                <label className="block text-sm font-medium text-slate-700 mb-2">
                                  Test Content
                                </label>
                                <textarea
                                  value={aiTestContent}
                                  onChange={(e) => setAiTestContent(e.target.value)}
                                  placeholder="Enter content to test AI relevancy scoring..."
                                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                                  rows={3}
                                />
                              </div>
                              
                              <div className="flex items-center space-x-3">
                                <button
                                  onClick={testAIProvider}
                                  disabled={!aiTestContent.trim() || aiTestLoading}
                                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                  {aiTestLoading ? 'Testing...' : 'Test AI Provider'}
                                </button>
                                
                                {aiTestResult && (
                                  <div className="flex items-center space-x-4">
                                    <div className={`px-3 py-1 rounded-full text-sm font-medium ${
                                      aiTestResult.ai_score >= 0.5 
                                        ? 'bg-green-100 text-green-800' 
                                        : 'bg-red-100 text-red-800'
                                    }`}>
                                      {aiTestResult.status === 'simulated' ? 'Simulated' : 'Live'} Result
                                    </div>
                                    <div className="text-sm text-slate-600">
                                      Score: <span className="font-medium">{(aiTestResult.ai_score * 100).toFixed(1)}%</span>
                                    </div>
                                    <div className="text-sm text-slate-600">
                                      Time: <span className="font-medium">{aiTestResult.response_time}</span>
                                    </div>
                                  </div>
                                )}
                              </div>

                              {aiTestResult && (
                                <div className="bg-slate-50 rounded-lg p-4 text-sm">
                                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-3">
                                    <div>
                                      <div className="font-medium text-slate-700">AI Score</div>
                                      <div className="text-blue-600">{(aiTestResult.ai_score * 100).toFixed(1)}%</div>
                                    </div>
                                    <div>
                                      <div className="font-medium text-slate-700">Confidence</div>
                                      <div className="text-blue-600">{(aiTestResult.confidence * 100).toFixed(1)}%</div>
                                    </div>
                                    <div>
                                      <div className="font-medium text-slate-700">Provider</div>
                                      <div className="text-slate-600">{aiTestResult.provider || 'Simulated'}</div>
                                    </div>
                                    <div>
                                      <div className="font-medium text-slate-700">Tokens Used</div>
                                      <div className="text-slate-600">{aiTestResult.tokens_used}</div>
                                    </div>
                                  </div>
                                  
                                  <div>
                                    <div className="font-medium text-slate-700 mb-1">AI Reasoning:</div>
                                    <div className="text-slate-600 italic">{aiTestResult.reasoning}</div>
                                  </div>
                                  
                                  {aiTestResult.status === 'simulated' && (
                                    <div className="mt-3 p-2 bg-yellow-100 border border-yellow-200 rounded text-yellow-800 text-xs">
                                      ⚠️ This is a simulated response. Configure a real AI provider for live results.
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          </div>

                          {/* Save Configuration */}
                          <div className="flex justify-end">
                            <button
                              onClick={() => saveAIProviderConfig(aiProviderConfig)}
                              className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition-colors"
                            >
                              Save AI Configuration
                            </button>
                          </div>
                        </div>
                      )}

                      {!aiProviderConfig.ai_enabled && (
                        <div className="text-center py-8">
                          <div className="text-4xl mb-4">🤖</div>
                          <h4 className="text-lg font-medium text-slate-900 mb-2">AI-Enhanced Relevancy</h4>
                          <p className="text-slate-600 mb-4">
                            Enable AI to enhance relevancy detection with Large Language Models.<br/>
                            Get more accurate, context-aware relevancy scoring.
                          </p>
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                            <div className="bg-white rounded-lg p-3 border">
                              <div className="font-medium text-slate-700">🎯 Better Accuracy</div>
                              <div className="text-slate-600">Context-aware analysis beyond keywords</div>
                            </div>
                            <div className="bg-white rounded-lg p-3 border">
                              <div className="font-medium text-slate-700">🧠 Semantic Understanding</div>
                              <div className="text-slate-600">Understands meaning, not just words</div>
                            </div>
                            <div className="bg-white rounded-lg p-3 border">
                              <div className="font-medium text-slate-700">⚡ Hybrid Mode</div>
                              <div className="text-slate-600">Combines AI with local keyword scoring</div>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )
            })()}
          </div>
        </div>
      )}

             {/* Regex Tab */}
       {activeTab === 'regex' && moderationConfig && (
         <div className="space-y-6">
           <div className="bg-white border border-slate-200 rounded-xl p-6">
             <h2 className="text-xl font-bold text-slate-900 mb-6">🔍 Regex Layer Configuration</h2>
             
             {/* Find Regex Layer */}
             {(() => {
               const regexLayer = moderationConfig.layers.find(layer => layer.name === 'regex')
               if (!regexLayer) {
                return (
                  <div className="text-center py-12 text-slate-500">
                    <span className="text-4xl mb-4 block">🔍</span>
                    <h3 className="text-lg font-medium mb-2">Regex Layer Not Found</h3>
                    <p className="text-sm">
                      The regex layer is not configured. Please add it to your moderation layers.
                    </p>
                  </div>
                )
              }

              const regexLayerIndex = moderationConfig.layers.findIndex(layer => layer.name === 'regex')
              
              return (
                <div className="space-y-8">
                  {/* Regex Layer Status */}
                  <div className="bg-gradient-to-r from-green-50 to-teal-50 border border-green-200 rounded-lg p-6">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="text-lg font-semibold text-slate-900">Regex Layer Status</h3>
                      <div className="flex items-center space-x-4">
                        <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                          regexLayer.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {regexLayer.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                        <button
                          onClick={() => updateLayerConfig(regexLayerIndex, 'enabled', !regexLayer.enabled)}
                          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                            regexLayer.enabled 
                              ? 'bg-red-600 text-white hover:bg-red-700' 
                              : 'bg-green-600 text-white hover:bg-green-700'
                          }`}
                        >
                          {regexLayer.enabled ? 'Disable' : 'Enable'}
                        </button>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                          Weight
                        </label>
                        <input
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={regexLayer.weight}
                          onChange={(e) => updateLayerConfig(regexLayerIndex, 'weight', parseFloat(e.target.value))}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                          Threshold
                        </label>
                        <input
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={regexLayer.threshold}
                          onChange={(e) => updateLayerConfig(regexLayerIndex, 'threshold', parseFloat(e.target.value))}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>
                      <div className="flex items-end">
                        <div className="text-sm text-slate-600">
                          <div>Score threshold for blocking content</div>
                          <div className="text-xs text-slate-500 mt-1">
                            Current: {(regexLayer.threshold * 100).toFixed(0)}%
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Regex Settings */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">Regex Patterns</h3>
                      <div className="space-y-4">
                                                 {regexLayer.options.patterns?.map((pattern: any, index: number) => (
                          <div key={index} className="border border-slate-200 rounded-lg p-4">
                            <div className="flex items-center justify-between mb-3">
                              <div className="flex items-center space-x-2">
                                <span>{pattern.pattern}</span>
                                <span className="font-medium text-slate-900">{pattern.description}</span>
                              </div>
                              <input
                                type="checkbox"
                                checked={pattern.enabled}
                                                                 onChange={(e) => updateLayerConfig(regexLayerIndex, 'options.patterns', regexLayer.options.patterns.map((p: any, i: number) => i === index ? { ...p, enabled: e.target.checked } : p))}
                                className="rounded"
                              />
                            </div>
                            <div>
                              <label className="block text-xs font-medium text-slate-600 mb-1">
                                Severity
                              </label>
                              <select
                                value={pattern.severity}
                                                                 onChange={(e) => updateLayerConfig(regexLayerIndex, 'options.patterns', regexLayer.options.patterns.map((p: any, i: number) => i === index ? { ...p, severity: e.target.value } : p))}
                                className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                              >
                                <option value="low">Low</option>
                                <option value="medium">Medium</option>
                                <option value="high">High</option>
                                <option value="critical">Critical</option>
                              </select>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>

                                         <div>
                       <h3 className="text-lg font-semibold text-slate-900 mb-4">🤖 AI Enhancement for Banned Words</h3>
                       <div className="space-y-4">
                         {/* AI Enhancement Toggle */}
                         <div className="bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-lg p-4">
                           <div className="flex items-center justify-between mb-3">
                             <div>
                               <div className="font-medium text-slate-900">AI Word Variation Detection</div>
                               <div className="text-sm text-slate-600">
                                 Use AI to detect variations, misspellings, and synonyms of banned words that regex patterns might miss
                               </div>
                             </div>
                             <input
                               type="checkbox"
                               checked={regexAiConfig?.ai_enabled ?? false}
                               onChange={(e) => updateRegexAIConfig('ai_enabled', e.target.checked)}
                               className="rounded"
                             />
                           </div>
                           
                           {regexAiConfig?.ai_enabled && (
                             <div className="mt-4 bg-white rounded-lg p-3 border border-blue-100">
                               <div className="text-xs text-slate-600 mb-2">
                                 <strong>How it works:</strong> Regex patterns catch exact matches, AI catches variations
                               </div>
                               <div className="flex text-xs text-slate-500">
                                 <div className="flex-1">
                                   <div className="font-medium text-red-600 mb-1">Regex finds:</div>
                                   <div>"violence" → ✓</div>
                                   <div>"violent" → ✗</div>
                                   <div>"violance" → ✗</div>
                                 </div>
                                 <div className="flex-1">
                                   <div className="font-medium text-green-600 mb-1">AI also finds:</div>
                                   <div>"violent" → ✓</div>
                                   <div>"violance" → ✓</div>
                                   <div>"brutality" → ✓</div>
                                 </div>
                               </div>
                             </div>
                           )}
                         </div>

                         {regexAiConfig?.ai_enabled && (
                           <>
                             {/* AI Provider Configuration */}
                             <div className="border border-slate-200 rounded-lg p-4">
                               <div className="font-medium text-slate-900 mb-3">AI Provider Settings</div>
                               <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                 <div>
                                   <label className="block text-sm font-medium text-slate-700 mb-1">
                                     Provider
                                   </label>
                                   <select
                                     value={regexAiConfig?.provider || ''}
                                     onChange={(e) => updateRegexAIConfig('provider', e.target.value)}
                                     className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                                   >
                                     <option value="">Select Provider</option>
                                     <option value="openai">OpenAI</option>
                                     <option value="anthropic">Anthropic</option>
                                     <option value="local">Local Model</option>
                                     <option value="custom">Custom Endpoint</option>
                                   </select>
                                 </div>
                                 <div>
                                   <label className="block text-sm font-medium text-slate-700 mb-1">
                                     Model
                                   </label>
                                   <select
                                     value={regexAiConfig?.model || ''}
                                     onChange={(e) => updateRegexAIConfig('model', e.target.value)}
                                     className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                                   >
                                     <option value="">Select Model</option>
                                     {regexAiConfig?.provider === 'openai' && (
                                       <>
                                         <option value="gpt-4">GPT-4</option>
                                         <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                                       </>
                                     )}
                                     {regexAiConfig?.provider === 'anthropic' && (
                                       <>
                                         <option value="claude-3-sonnet">Claude 3 Sonnet</option>
                                         <option value="claude-3-haiku">Claude 3 Haiku</option>
                                       </>
                                     )}
                                     {regexAiConfig?.provider === 'local' && (
                                       <>
                                         <option value="llama-2">Llama 2</option>
                                         <option value="mistral">Mistral</option>
                                       </>
                                     )}
                                   </select>
                                 </div>
                               </div>
                               
                               <div className="mt-3">
                                 <label className="block text-sm font-medium text-slate-700 mb-1">
                                   API Key
                                 </label>
                                 <input
                                   type="password"
                                   value={regexAiConfig?.api_key || ''}
                                   onChange={(e) => updateRegexAIConfig('api_key', e.target.value)}
                                   placeholder="Enter API key..."
                                   className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                                 />
                               </div>
                             </div>

                             {/* AI Detection Settings */}
                             <div className="border border-slate-200 rounded-lg p-4">
                               <div className="font-medium text-slate-900 mb-3">AI Detection Behavior</div>
                               <div className="space-y-3">
                                 <div className="flex items-center justify-between">
                                   <div>
                                     <div className="text-sm font-medium text-slate-700">Detect Misspellings</div>
                                     <div className="text-xs text-slate-500">Find common typos and intentional misspellings</div>
                                   </div>
                                   <input
                                     type="checkbox"
                                     checked={regexAiConfig?.detect_misspellings ?? true}
                                     onChange={(e) => updateRegexAIConfig('detect_misspellings', e.target.checked)}
                                     className="rounded"
                                   />
                                 </div>
                                 
                                 <div className="flex items-center justify-between">
                                   <div>
                                     <div className="text-sm font-medium text-slate-700">Detect Synonyms</div>
                                     <div className="text-xs text-slate-500">Find words with similar meanings</div>
                                   </div>
                                   <input
                                     type="checkbox"
                                     checked={regexAiConfig?.detect_synonyms ?? true}
                                     onChange={(e) => updateRegexAIConfig('detect_synonyms', e.target.checked)}
                                     className="rounded"
                                   />
                                 </div>
                                 
                                 <div className="flex items-center justify-between">
                                   <div>
                                     <div className="text-sm font-medium text-slate-700">Detect Word Variations</div>
                                     <div className="text-xs text-slate-500">Find different forms (violence → violent)</div>
                                   </div>
                                   <input
                                     type="checkbox"
                                     checked={regexAiConfig?.detect_variations ?? true}
                                     onChange={(e) => updateRegexAIConfig('detect_variations', e.target.checked)}
                                     className="rounded"
                                   />
                                 </div>
                                 
                                 <div className="flex items-center justify-between">
                                   <div>
                                     <div className="text-sm font-medium text-slate-700">Context-Aware Detection</div>
                                     <div className="text-xs text-slate-500">Consider surrounding words for better accuracy</div>
                                   </div>
                                   <input
                                     type="checkbox"
                                     checked={regexAiConfig?.context_aware ?? true}
                                     onChange={(e) => updateRegexAIConfig('context_aware', e.target.checked)}
                                     className="rounded"
                                   />
                                 </div>
                               </div>
                             </div>

                             {/* AI Model Parameters */}
                             <div className="border border-slate-200 rounded-lg p-4">
                               <div className="font-medium text-slate-900 mb-3">Model Parameters</div>
                               <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                 <div>
                                   <label className="block text-sm font-medium text-slate-700 mb-1">
                                     Confidence Threshold
                                   </label>
                                   <input
                                     type="number"
                                     min="0"
                                     max="1"
                                     step="0.05"
                                     value={regexAiConfig?.confidence_threshold || 0.8}
                                     onChange={(e) => updateRegexAIConfig('confidence_threshold', parseFloat(e.target.value))}
                                     className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                                   />
                                   <div className="text-xs text-slate-500 mt-1">
                                     Higher = more strict, Lower = more permissive
                                   </div>
                                 </div>
                                 
                                 <div>
                                   <label className="block text-sm font-medium text-slate-700 mb-1">
                                     Temperature
                                   </label>
                                   <input
                                     type="number"
                                     min="0"
                                     max="1"
                                     step="0.1"
                                     value={regexAiConfig?.temperature || 0.3}
                                     onChange={(e) => updateRegexAIConfig('temperature', parseFloat(e.target.value))}
                                     className="w-full border border-slate-300 rounded px-2 py-1 text-sm"
                                   />
                                   <div className="text-xs text-slate-500 mt-1">
                                     Lower = more focused detection
                                   </div>
                                 </div>
                               </div>
                             </div>

                             {/* Save AI Config Button */}
                             <div className="flex justify-end">
                               <button
                                 onClick={() => saveRegexAIConfig(regexAiConfig)}
                                 className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
                               >
                                 Save AI Configuration
                               </button>
                             </div>
                           </>
                         )}
                       </div>
                     </div>
                  </div>

                  {/* Regex Testing */}
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <h3 className="text-lg font-semibold text-slate-900 mb-4">🔍 Test Regex Patterns</h3>
                    <div className="space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                          Test Content
                        </label>
                        <textarea
                          value={regexTestContent}
                          onChange={(e) => setRegexTestContent(e.target.value)}
                          placeholder="Enter content to test regex patterns..."
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                          rows={3}
                        />
                      </div>
                      
                      <div className="flex items-center space-x-3">
                        <button
                          onClick={testRegex}
                          disabled={!regexTestContent.trim() || regexTestLoading}
                          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          {regexTestLoading ? 'Testing...' : 'Test Regex Patterns'}
                        </button>
                        
                        {regexTestResult && (
                          <div className="flex items-center space-x-4">
                            <div className={`px-3 py-1 rounded-full text-sm font-medium ${
                              regexTestResult.matched ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                            }`}>
                              {regexTestResult.matched ? 'Matched' : 'No Match'}
                            </div>
                            <div className="text-sm text-slate-600">
                              Matches: <span className="font-medium">{regexTestResult.matches.length}</span>
                            </div>
                          </div>
                        )}
                      </div>

                      {regexTestResult && (
                        <div className="bg-slate-50 rounded-lg p-4 text-sm">
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-3">
                            <div>
                              <div className="font-medium text-slate-700">Regex Score</div>
                              <div className="text-green-600">{(regexTestResult.score * 100).toFixed(1)}%</div>
                            </div>
                            <div>
                              <div className="font-medium text-slate-700">Pattern Matches</div>
                              <div className="text-green-600">{regexTestResult.matches.length}</div>
                            </div>
                            <div>
                              <div className="font-medium text-slate-700">Cache Hit Rate</div>
                              <div className="text-blue-600">
                                {regexTestResult.cache_hit_rate ? (regexTestResult.cache_hit_rate * 100).toFixed(1) + '%' : 'N/A'}
                              </div>
                            </div>
                          </div>
                          {regexTestResult.matched && (
                            <div className="mt-3">
                              <div className="font-medium text-slate-700 mb-1">Matched Patterns:</div>
                              <div className="flex flex-wrap gap-2">
                                                                 {regexTestResult.matches.map((match: any, i: number) => (
                                  <span key={i} className="bg-green-100 text-green-800 px-2 py-1 rounded text-xs">
                                    {match}
                                  </span>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Add New Pattern */}
                  <div className="bg-white border border-slate-200 rounded-lg p-6">
                    <h3 className="text-lg font-semibold text-slate-900 mb-4">➕ Add New Pattern</h3>
                    <div className="space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                          Pattern
                        </label>
                        <input
                          type="text"
                          value={newRegexPattern}
                          onChange={(e) => setNewRegexPattern(e.target.value)}
                          placeholder="Enter regex pattern..."
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                          Description
                        </label>
                        <input
                          type="text"
                          value={newRegexDescription}
                          onChange={(e) => setNewRegexDescription(e.target.value)}
                          placeholder="Enter description for the pattern..."
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-2">
                          Severity
                        </label>
                        <select
                          value={newRegexSeverity}
                          onChange={(e) => setNewRegexSeverity(e.target.value)}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        >
                          <option value="low">Low</option>
                          <option value="medium">Medium</option>
                          <option value="high">High</option>
                          <option value="critical">Critical</option>
                        </select>
                      </div>

                      <button
                        onClick={() => addRegexPattern(regexLayerIndex)}
                        disabled={!newRegexPattern.trim()}
                        className="w-full bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        Add Pattern
                      </button>
                    </div>
                  </div>

                  {/* Regex Statistics */}
                  <div className="mt-6 bg-slate-50 border border-slate-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <h4 className="font-medium text-slate-900">📊 Statistics</h4>
                      <button
                        onClick={fetchRegexStats}
                        className="text-sm text-blue-600 hover:text-blue-800"
                      >
                        Refresh
                      </button>
                    </div>
                    {regexStats ? (
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <div className="text-slate-600">Patterns Processed</div>
                          <div className="font-medium text-slate-900">{regexStats.patterns_processed || 0}</div>
                        </div>
                        <div>
                          <div className="text-slate-600">Patterns Blocked</div>
                          <div className="font-medium text-red-600">{regexStats.patterns_blocked || 0}</div>
                        </div>
                        <div>
                          <div className="text-slate-600">Avg Score</div>
                          <div className="font-medium text-green-600">
                            {regexStats.average_score ? (regexStats.average_score * 100).toFixed(1) + '%' : 'N/A'}
                          </div>
                        </div>
                        <div>
                          <div className="text-slate-600">Cache Hit Rate</div>
                          <div className="font-medium text-blue-600">
                            {regexStats.cache_hit_rate ? (regexStats.cache_hit_rate * 100).toFixed(1) + '%' : 'N/A'}
                          </div>
                        </div>
                      </div>
                    ) : (
                      <div className="text-sm text-slate-500">Click refresh to load statistics</div>
                    )}
                  </div>
                </div>
              )
            })()}
          </div>
        </div>
      )}

             {/* PII Tab */}
       {activeTab === 'pii' && moderationConfig && (
         <div className="space-y-6">
           <div className="bg-white border border-slate-200 rounded-xl p-6">
             <h2 className="text-xl font-bold text-slate-900 mb-6">🔒 PII Detection Configuration</h2>
             
             {/* Find PII Layer */}
             {(() => {
               const piiLayer = moderationConfig.layers.find(layer => layer.name === 'pii')
               if (!piiLayer) {
                return (
                  <div className="text-center py-12 text-slate-500">
                    <span className="text-4xl mb-4 block">🔍</span>
                    <h3 className="text-lg font-medium mb-2">PII Layer Not Found</h3>
                    <p className="text-sm">
                      The PII detection layer is not configured. Please add it to your moderation layers.
                    </p>
                  </div>
                )
              }

              const piiLayerIndex = moderationConfig.layers.findIndex(layer => layer.name === 'pii')
              
              return (
                <div className="space-y-8">
                  {/* PII Layer Status */}
                  <div className="bg-gradient-to-r from-purple-50 to-pink-50 border border-purple-200 rounded-lg p-6">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="text-lg font-semibold text-slate-900">🔒 PII Layer Status</h3>
                      <div className="flex items-center space-x-4">
                        <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                          piiLayer.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                        }`}>
                          {piiLayer.enabled ? 'Enabled' : 'Disabled'}
                        </span>
                        <button
                          onClick={() => updateLayerConfig(piiLayerIndex, 'enabled', !piiLayer.enabled)}
                          className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                            piiLayer.enabled 
                              ? 'bg-red-600 text-white hover:bg-red-700' 
                              : 'bg-green-600 text-white hover:bg-green-700'
                          }`}
                        >
                          {piiLayer.enabled ? 'Disable' : 'Enable'}
                        </button>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                          Weight
                        </label>
                        <input
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={piiLayer.weight}
                          onChange={(e) => updateLayerConfig(piiLayerIndex, 'weight', parseFloat(e.target.value))}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                          Threshold
                        </label>
                        <input
                          type="number"
                          min="0"
                          max="1"
                          step="0.1"
                          value={piiLayer.threshold}
                          onChange={(e) => updateLayerConfig(piiLayerIndex, 'threshold', parseFloat(e.target.value))}
                          className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                        />
                      </div>
                      <div className="flex items-end">
                        <div className="text-sm text-slate-600">
                          <div>Score threshold for blocking content</div>
                          <div className="text-xs text-slate-500 mt-1">
                            Current: {(piiLayer.threshold * 100).toFixed(0)}%
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* AI Enhanced PII Detection */}
                  <div className="bg-white border border-slate-200 rounded-lg p-6 mb-6">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="text-lg font-semibold text-slate-900">🤖 AI Enhanced Detection</h3>
                      <div className="flex items-center space-x-2">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                          piiAiConfig?.ai_enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                          {piiAiConfig?.ai_enabled ? 'AI Enabled' : 'AI Disabled'}
                        </span>
                      </div>
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                      {/* AI Provider Configuration */}
                      <div className="space-y-4">
                        <div className="flex items-center space-x-2 mb-3">
                          <input
                            type="checkbox"
                            checked={piiAiConfig?.ai_enabled || false}
                            onChange={(e) => updatePiiAIConfig('ai_enabled', e.target.checked)}
                            className="rounded"
                          />
                          <span className="text-sm font-medium text-slate-700">Enable AI Enhancement</span>
                        </div>

                        <div className="grid grid-cols-2 gap-3">
                          <div>
                            <label className="block text-xs font-medium text-slate-600 mb-1">Provider</label>
                            <select
                              value={piiAiConfig?.provider || ''}
                              onChange={(e) => updatePiiAIConfig('provider', e.target.value)}
                              disabled={!piiAiConfig?.ai_enabled}
                              className="w-full border border-slate-300 rounded px-2 py-1 text-xs disabled:bg-gray-50"
                            >
                              <option value="">Select Provider</option>
                              <option value="openai">OpenAI</option>
                              <option value="anthropic">Anthropic</option>
                              <option value="local">Local LLM</option>
                            </select>
                          </div>
                          <div>
                            <label className="block text-xs font-medium text-slate-600 mb-1">Model</label>
                            <input
                              type="text"
                              value={piiAiConfig?.model || ''}
                              onChange={(e) => updatePiiAIConfig('model', e.target.value)}
                              disabled={!piiAiConfig?.ai_enabled}
                              placeholder="gpt-3.5-turbo"
                              className="w-full border border-slate-300 rounded px-2 py-1 text-xs disabled:bg-gray-50"
                            />
                          </div>
                        </div>

                        <div>
                          <label className="block text-xs font-medium text-slate-600 mb-1">API Key</label>
                          <input
                            type="password"
                            value={piiAiConfig?.api_key || ''}
                            onChange={(e) => updatePiiAIConfig('api_key', e.target.value)}
                            disabled={!piiAiConfig?.ai_enabled}
                            placeholder="Enter API key..."
                            className="w-full border border-slate-300 rounded px-2 py-1 text-xs disabled:bg-gray-50"
                          />
                        </div>

                        <div className="grid grid-cols-2 gap-3">
                          <div>
                            <label className="block text-xs font-medium text-slate-600 mb-1">Max Tokens</label>
                            <input
                              type="number"
                              value={piiAiConfig?.max_tokens || 1024}
                              onChange={(e) => updatePiiAIConfig('max_tokens', parseInt(e.target.value))}
                              disabled={!piiAiConfig?.ai_enabled}
                              className="w-full border border-slate-300 rounded px-2 py-1 text-xs disabled:bg-gray-50"
                            />
                          </div>
                          <div>
                            <label className="block text-xs font-medium text-slate-600 mb-1">Temperature</label>
                            <input
                              type="number"
                              min="0"
                              max="1"
                              step="0.1"
                              value={piiAiConfig?.temperature || 0.2}
                              onChange={(e) => updatePiiAIConfig('temperature', parseFloat(e.target.value))}
                              disabled={!piiAiConfig?.ai_enabled}
                              className="w-full border border-slate-300 rounded px-2 py-1 text-xs disabled:bg-gray-50"
                            />
                          </div>
                        </div>

                        <div className="flex space-x-2">
                          <button
                            onClick={() => savePiiAIConfig(piiAiConfig)}
                            disabled={!piiAiConfig?.ai_enabled}
                            className="bg-blue-600 text-white px-3 py-1 rounded text-xs hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            Save AI Config
                          </button>
                          <button
                            onClick={fetchPiiStats}
                            className="bg-gray-600 text-white px-3 py-1 rounded text-xs hover:bg-gray-700 transition-colors"
                          >
                            Refresh Stats
                          </button>
                        </div>
                      </div>

                      {/* AI Test Interface */}
                      <div className="space-y-4">
                        <div>
                          <label className="block text-xs font-medium text-slate-600 mb-1">Test Content</label>
                          <textarea
                            value={piiTestContent}
                            onChange={(e) => setPiiTestContent(e.target.value)}
                            placeholder="Enter content to test for PII detection..."
                            className="w-full border border-slate-300 rounded px-2 py-1 text-xs"
                            rows={3}
                          />
                        </div>
                        
                        <div className="flex items-center space-x-3">
                          <button
                            onClick={testPii}
                            disabled={!piiTestContent.trim() || piiTestLoading}
                            className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {piiTestLoading ? 'Testing...' : 'Test PII Detection'}
                          </button>
                          
                          {piiTestResult && (
                            <div className="flex items-center space-x-4">
                              <div className={`px-3 py-1 rounded-full text-sm font-medium ${
                                piiTestResult.detected ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'
                              }`}>
                                {piiTestResult.detected ? 'PII Detected' : 'No PII Found'}
                              </div>
                              <div className="text-sm text-slate-600">
                                Score: <span className="font-medium">{(piiTestResult.score * 100).toFixed(1)}%</span>
                              </div>
                            </div>
                          )}
                        </div>

                        {piiTestResult && (
                          <div className="bg-slate-50 rounded-lg p-4 text-sm">
                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-3">
                              <div>
                                <div className="font-medium text-slate-700">Detection Score</div>
                                <div className="text-red-600">{(piiTestResult.score * 100).toFixed(1)}%</div>
                              </div>
                              <div>
                                <div className="font-medium text-slate-700">Risk Level</div>
                                <div className={`${
                                  piiTestResult.risk_level === 'high' ? 'text-red-600' :
                                  piiTestResult.risk_level === 'medium' ? 'text-yellow-600' : 'text-green-600'
                                }`}>
                                  {piiTestResult.risk_level?.toUpperCase() || 'LOW'}
                                </div>
                              </div>
                              <div>
                                <div className="font-medium text-slate-700">Method</div>
                                <div className="text-blue-600">{piiTestResult.method || 'Pattern Match'}</div>
                              </div>
                            </div>
                            {piiTestResult.detected && (
                              <div className="mt-3">
                                <div className="font-medium text-slate-700 mb-1">Detected PII Types:</div>
                                <div className="flex flex-wrap gap-2 mb-3">
                                  {(piiTestResult.types || []).map((type: string, i: number) => (
                                    <span key={i} className="bg-red-100 text-red-800 px-2 py-1 rounded text-xs">
                                      {type}
                                    </span>
                                  ))}
                                </div>
                                {piiTestResult.masked_content && (
                                  <div>
                                    <div className="font-medium text-slate-700 mb-1">Masked Content:</div>
                                    <div className="bg-white border rounded p-2 text-xs font-mono">
                                      {piiTestResult.masked_content}
                                    </div>
                                  </div>
                                )}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* PII Detection Settings */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">Detection Types</h3>
                      <div className="space-y-4">
                        {[
                          { key: 'detect_email', label: 'Email Addresses', icon: '📧', confidence: 'email_confidence' },
                          { key: 'detect_phone', label: 'Phone Numbers', icon: '📞', confidence: 'phone_confidence' },
                          { key: 'detect_ssn', label: 'Social Security Numbers', icon: '🆔', confidence: 'ssn_confidence' },
                          { key: 'detect_credit_card', label: 'Credit Card Numbers', icon: '💳', confidence: 'credit_card_confidence' }
                        ].map((type) => (
                          <div key={type.key} className="border border-slate-200 rounded-lg p-4">
                            <div className="flex items-center justify-between mb-3">
                              <div className="flex items-center space-x-2">
                                <span>{type.icon}</span>
                                <span className="font-medium text-slate-900">{type.label}</span>
                              </div>
                              <input
                                type="checkbox"
                                checked={piiLayer.options?.[type.key] ?? true}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, `options.${type.key}`, e.target.checked)}
                                className="rounded"
                              />
                            </div>
                            <div>
                              <label className="block text-xs font-medium text-slate-600 mb-1">
                                Confidence Threshold
                              </label>
                              <input
                                type="number"
                                min="0"
                                max="1"
                                step="0.01"
                                value={piiLayer.options?.[type.confidence] ?? 0.9}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, `options.${type.confidence}`, parseFloat(e.target.value))}
                                className="w-full border border-slate-300 rounded px-2 py-1 text-xs"
                                disabled={!piiLayer.options?.[type.key]}
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div>
                      <h3 className="text-lg font-semibold text-slate-900 mb-4">Privacy & Security</h3>
                      <div className="space-y-4">
                        <div className="border border-slate-200 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-3">
                            <div>
                              <div className="font-medium text-slate-900">PII Masking</div>
                              <div className="text-sm text-slate-600">
                                Automatically mask detected PII in logs and responses
                              </div>
                            </div>
                            <input
                              type="checkbox"
                              checked={piiLayer.options?.masking_enabled ?? true}
                              onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.masking_enabled', e.target.checked)}
                              className="rounded"
                            />
                          </div>
                        </div>

                        <div className="border border-slate-200 rounded-lg p-4">
                          <div className="font-medium text-slate-900 mb-3">Custom Patterns</div>
                          <div className="text-sm text-slate-600 mb-3">
                            Add custom regex patterns for organization-specific PII
                          </div>
                          <textarea
                            placeholder="Enter custom regex patterns (one per line)"
                            value={piiLayer.options?.custom_patterns?.join('\n') ?? ''}
                            onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.custom_patterns', e.target.value.split('\n').filter(p => p.trim()))}
                            className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                            rows={4}
                          />
                        </div>

                        <div className="border border-slate-200 rounded-lg p-4">
                          <div className="font-medium text-slate-900 mb-3">Alert Settings</div>
                          <div className="space-y-3">
                            <div className="flex items-center justify-between">
                              <span className="text-sm text-slate-700">Real-time alerts for PII detection</span>
                              <input
                                type="checkbox"
                                checked={piiLayer.options?.realtime_alerts ?? true}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.realtime_alerts', e.target.checked)}
                                className="rounded"
                              />
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-sm text-slate-700">Email notifications for high-risk PII</span>
                              <input
                                type="checkbox"
                                checked={piiLayer.options?.email_alerts ?? false}
                                onChange={(e) => updateLayerConfig(piiLayerIndex, 'options.email_alerts', e.target.checked)}
                                className="rounded"
                              />
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })()}
          </div>
        </div>
      )}

      {/* Performance Tab */}
      {activeTab === 'performance' && moderationConfig && ruleEngineConfig && (
        <div className="space-y-6">
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h2 className="text-xl font-bold text-slate-900 mb-6">Performance & Caching</h2>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
              {/* Moderation Cache */}
              <div>
                <h3 className="text-lg font-semibold text-slate-900 mb-4">Moderation Cache</h3>
                <div className="space-y-4">
                  <div className="flex items-center">
                    <input
                      type="checkbox"
                      checked={moderationConfig.cache.enabled}
                      onChange={(e) => updateModerationConfig('cache.enabled', e.target.checked)}
                      className="rounded"
                    />
                    <span className="ml-2 text-sm text-slate-600">Enable caching</span>
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      TTL (seconds)
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="86400"
                      value={moderationConfig.cache.ttl}
                      onChange={(e) => updateModerationConfig('cache.ttl', parseInt(e.target.value))}
                      className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      Max Size (entries)
                    </label>
                    <input
                      type="number"
                      min="100"
                      max="100000"
                      value={moderationConfig.cache.max_size}
                      onChange={(e) => updateModerationConfig('cache.max_size', parseInt(e.target.value))}
                      className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                    />
                  </div>
                </div>
              </div>

              {/* Rule Engine Cache */}
              <div>
                <h3 className="text-lg font-semibold text-slate-900 mb-4">Rule Engine Cache</h3>
                <div className="space-y-4">
                  <div className="flex items-center">
                    <input
                      type="checkbox"
                      checked={ruleEngineConfig.enable_caching}
                      onChange={(e) => updateRuleEngineConfig('enable_caching', e.target.checked)}
                      className="rounded"
                    />
                    <span className="ml-2 text-sm text-slate-600">Enable rule caching</span>
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      Cache Size (entries)
                    </label>
                    <input
                      type="number"
                      min="100"
                      max="50000"
                      value={ruleEngineConfig.cache_size}
                      onChange={(e) => updateRuleEngineConfig('cache_size', parseInt(e.target.value))}
                      className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      Cache TTL (minutes)
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="1440"
                      value={ruleEngineConfig.cache_ttl}
                      onChange={(e) => updateRuleEngineConfig('cache_ttl', parseInt(e.target.value))}
                      className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default AdvancedConfiguration