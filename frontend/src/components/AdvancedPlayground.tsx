import React, { useState, useEffect } from 'react'
import { useWebSocket, useWebSocketMessage } from '../contexts/WebSocketContext'
import { MessageTypes } from '../services/websocket'

interface TestResult {
  content: string
  final_score: number
  final_confidence: number
  recommended_action: string
  primary_category: string
  triggered_rules: any[]
  layer_results: any[]
  process_time: string
  score_breakdown: any
  context: any
  timestamp: string
}

interface LayerInfo {
  name: string
  enabled: boolean
  weight: number
}

const AdvancedPlayground: React.FC = () => {
  const { isConnected } = useWebSocket()
  const [testContent, setTestContent] = useState('')
  const [testResults, setTestResults] = useState<TestResult[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedLayers, setSelectedLayers] = useState<string[]>([])
  const [availableLayers, setAvailableLayers] = useState<LayerInfo[]>([])
  const [testContext, setTestContext] = useState({
    user_id: 'test_user',
    user_type: 'user',
    content_type: 'message',
    channel: 'general'
  })
  const [batchTest, setBatchTest] = useState(false)
  const [batchContent, setBatchContent] = useState('')

  useEffect(() => {
    fetchAvailableLayers()
  }, [])

  // Listen for real-time test results
  useWebSocketMessage(MessageTypes.TEST_RESULT, (data) => {
    if (data?.result) {
      setTestResults(prev => [data.result, ...prev.slice(0, 19)])
      setLoading(false)
    }
  })

  const fetchAvailableLayers = async () => {
    try {
      const response = await fetch('/api/moderation/layers')
      if (response.ok) {
        const data = await response.json()
        setAvailableLayers(data.data || [])
        setSelectedLayers(data.data?.filter((l: LayerInfo) => l.enabled).map((l: LayerInfo) => l.name) || [])
      }
    } catch (error) {
      console.error('Failed to fetch layers:', error)
    }
  }

  const runSingleTest = async () => {
    if (!testContent.trim()) return
    
    setLoading(true)
    try {
      const response = await fetch('/api/moderation/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content: testContent,
          context: testContext,
          layers: selectedLayers.length > 0 ? selectedLayers : undefined
        })
      })

      if (response.ok) {
        const data = await response.json()
        setTestResults(prev => [{ ...data.data, timestamp: new Date().toISOString() }, ...prev.slice(0, 19)])
      }
    } catch (error) {
      console.error('Test failed:', error)
    } finally {
      setLoading(false)
    }
  }

  const runBatchTest = async () => {
    if (!batchContent.trim()) return
    
    const lines = batchContent.split('\n').filter(line => line.trim())
    if (lines.length === 0) return

    setLoading(true)
    try {
      const response = await fetch('/api/moderation/test/batch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content_list: lines,
          context: testContext,
          layers: selectedLayers.length > 0 ? selectedLayers : undefined
        })
      })

      if (response.ok) {
        const data = await response.json()
        const results = data.data.map((result: any) => ({
          ...result,
          timestamp: new Date().toISOString()
        }))
        setTestResults(prev => [...results, ...prev.slice(0, 20 - results.length)])
      }
    } catch (error) {
      console.error('Batch test failed:', error)
    } finally {
      setLoading(false)
    }
  }

  const predefinedTests = [
    { label: 'Hate Speech', content: 'You are so stupid and I hate you' },
    { label: 'Violence Threat', content: 'I will kill you and destroy everything' },
    { label: 'Spam Content', content: 'Buy now for a limited offer! Click here to save 90%!' },
    { label: 'Profanity', content: 'This is damn annoying and really pisses me off' },
    { label: 'Safe Content', content: 'Hello, how are you today? Hope you are doing well!' },
    { label: 'Mild Toxicity', content: 'That idea is really dumb and makes no sense' },
    { label: 'Sexual Content', content: 'Let me tell you about my sexual fantasies and desires' },
    { label: 'Self Harm', content: 'I want to hurt myself and end it all tonight' }
  ]

  const getScoreColor = (score: number) => {
    if (score >= 0.9) return 'text-red-600 bg-red-50 border-red-200'
    if (score >= 0.7) return 'text-orange-600 bg-orange-50 border-orange-200'
    if (score >= 0.5) return 'text-yellow-600 bg-yellow-50 border-yellow-200'
    return 'text-green-600 bg-green-50 border-green-200'
  }

  const getActionColor = (action: string) => {
    if (!action) return 'bg-gray-100 text-gray-800'
    switch (action.toLowerCase()) {
      case 'block': return 'bg-red-100 text-red-800'
      case 'flag': return 'bg-orange-100 text-orange-800'
      case 'warn': return 'bg-yellow-100 text-yellow-800'
      default: return 'bg-green-100 text-green-800'
    }
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-slate-200 pb-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-slate-900 mb-2">Advanced Testing Playground</h1>
            <p className="text-slate-600">Test content against the multi-layer moderation system with detailed analysis</p>
          </div>
          <div className={`text-sm font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
            {isConnected ? '🟢 Live Testing' : '🔴 Offline Mode'}
          </div>
        </div>
      </div>

      {/* Test Configuration */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Input Section */}
        <div className="lg:col-span-2 space-y-6">
          {/* Test Mode Toggle */}
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <div className="flex items-center space-x-4 mb-4">
              <button
                onClick={() => setBatchTest(false)}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  !batchTest ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                }`}
              >
                Single Test
              </button>
              <button
                onClick={() => setBatchTest(true)}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  batchTest ? 'bg-blue-600 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                }`}
              >
                Batch Test
              </button>
            </div>

            {!batchTest ? (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Test Content
                </label>
                <textarea
                  value={testContent}
                  onChange={(e) => setTestContent(e.target.value)}
                  placeholder="Enter content to test..."
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm h-32 resize-none"
                />
                <div className="flex items-center justify-between mt-4">
                  <div className="text-xs text-slate-500">
                    {testContent.length} characters
                  </div>
                  <button
                    onClick={runSingleTest}
                    disabled={loading || !testContent.trim()}
                    className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    {loading ? 'Testing...' : '🧪 Run Test'}
                  </button>
                </div>
              </div>
            ) : (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">
                  Batch Content (one per line)
                </label>
                <textarea
                  value={batchContent}
                  onChange={(e) => setBatchContent(e.target.value)}
                  placeholder="Enter multiple lines of content to test..."
                  className="w-full border border-slate-300 rounded-lg px-3 py-2 text-sm h-32 resize-none"
                />
                <div className="flex items-center justify-between mt-4">
                  <div className="text-xs text-slate-500">
                    {batchContent.split('\n').filter(line => line.trim()).length} test cases
                  </div>
                  <button
                    onClick={runBatchTest}
                    disabled={loading || !batchContent.trim()}
                    className="bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50"
                  >
                    {loading ? 'Testing...' : '📊 Run Batch Test'}
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Predefined Tests */}
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">Quick Tests</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {predefinedTests.map((test, index) => (
                <button
                  key={index}
                  onClick={() => setTestContent(test.content)}
                  className="bg-slate-100 hover:bg-slate-200 text-slate-700 px-3 py-2 rounded-lg text-sm transition-colors text-left"
                >
                  {test.label}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Configuration Panel */}
        <div className="space-y-6">
          {/* Layer Selection */}
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">Active Layers</h3>
            <div className="space-y-2">
              {availableLayers.map((layer) => (
                <label key={layer.name} className="flex items-center">
                  <input
                    type="checkbox"
                    checked={selectedLayers.includes(layer.name)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setSelectedLayers(prev => [...prev, layer.name])
                      } else {
                        setSelectedLayers(prev => prev.filter(l => l !== layer.name))
                      }
                    }}
                    className="rounded"
                  />
                  <span className="ml-2 text-sm">
                    {layer.name}
                    <span className="text-slate-500 text-xs ml-1">
                      (w:{layer.weight})
                    </span>
                  </span>
                  <span className={`ml-auto px-2 py-1 rounded text-xs ${
                    layer.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  }`}>
                    {layer.enabled ? 'Active' : 'Disabled'}
                  </span>
                </label>
              ))}
            </div>
          </div>

          {/* Test Context */}
          <div className="bg-white border border-slate-200 rounded-xl p-6">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">Test Context</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">User ID</label>
                <input
                  type="text"
                  value={testContext.user_id}
                  onChange={(e) => setTestContext(prev => ({ ...prev, user_id: e.target.value }))}
                  className="w-full border border-slate-300 rounded px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">User Type</label>
                <select
                  value={testContext.user_type}
                  onChange={(e) => setTestContext(prev => ({ ...prev, user_type: e.target.value }))}
                  className="w-full border border-slate-300 rounded px-3 py-2 text-sm"
                >
                  <option value="user">User</option>
                  <option value="admin">Admin</option>
                  <option value="guest">Guest</option>
                  <option value="moderator">Moderator</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Content Type</label>
                <select
                  value={testContext.content_type}
                  onChange={(e) => setTestContext(prev => ({ ...prev, content_type: e.target.value }))}
                  className="w-full border border-slate-300 rounded px-3 py-2 text-sm"
                >
                  <option value="message">Message</option>
                  <option value="post">Post</option>
                  <option value="comment">Comment</option>
                  <option value="chat">Chat</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Channel</label>
                <input
                  type="text"
                  value={testContext.channel}
                  onChange={(e) => setTestContext(prev => ({ ...prev, channel: e.target.value }))}
                  className="w-full border border-slate-300 rounded px-3 py-2 text-sm"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Test Results */}
      <div className="bg-white border border-slate-200 rounded-xl p-6">
        <h2 className="text-xl font-bold text-slate-900 mb-6">Test Results ({testResults.length})</h2>
        
        {testResults.length > 0 ? (
          <div className="space-y-4 max-h-96 overflow-y-auto">
            {testResults.map((result, index) => (
              <div key={index} className="border border-slate-200 rounded-lg p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1">
                    <div className="flex items-center space-x-3 mb-2">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getActionColor(result.recommended_action)}`}>
                        {result.recommended_action?.toUpperCase() || 'UNKNOWN'}
                      </span>
                      <span className={`px-2 py-1 rounded border text-xs font-medium ${getScoreColor(result.final_score || 0)}`}>
                        Score: {(result.final_score || 0).toFixed(3)}
                      </span>
                      <span className={`px-2 py-1 rounded border text-xs font-medium ${getScoreColor(result.final_confidence || 0)}`}>
                        Confidence: {(result.final_confidence || 0).toFixed(3)}
                      </span>
                      <span className="text-xs text-slate-500">
                        {new Date(result.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                    
                    <div className="text-sm text-slate-700 mb-2">
                      <strong>Content:</strong> {result.content || 'No content'}
                    </div>
                    
                    <div className="text-xs text-slate-500 mb-3">
                      <strong>Category:</strong> {result.primary_category || 'None'} | 
                      <strong> Process Time:</strong> {result.process_time || 'N/A'} |
                      <strong> Rules Triggered:</strong> {result.triggered_rules?.length || 0}
                    </div>

                    {/* Layer Results */}
                    {result.layer_results && result.layer_results.length > 0 && (
                      <div className="mt-3">
                        <h4 className="text-xs font-medium text-slate-700 mb-2">Layer Results:</h4>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                          {result.layer_results.map((layer, i) => (
                            <div key={i} className="bg-slate-50 rounded px-2 py-1 text-xs">
                              <div className="font-medium">{layer.layer_name || 'Unknown Layer'}</div>
                              <div className="text-slate-600">
                                Score: {layer.score?.toFixed(3) || 'N/A'} | 
                                {layer.blocked ? ' Blocked' : ' Allowed'}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Triggered Rules */}
                    {result.triggered_rules && result.triggered_rules.length > 0 && (
                      <div className="mt-3">
                        <h4 className="text-xs font-medium text-slate-700 mb-2">Triggered Rules:</h4>
                        <div className="space-y-1">
                          {result.triggered_rules.map((rule, i) => (
                            <div key={i} className="bg-yellow-50 border border-yellow-200 rounded px-2 py-1 text-xs">
                              <span className="font-medium">{rule.rule_name || 'Unknown Rule'}</span>
                              <span className="text-slate-600 ml-2">
                                ({rule.rule_id || 'N/A'}) - Score: {rule.score?.toFixed(3) || 'N/A'}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12 text-slate-500">
            <span className="text-4xl mb-2 block">🧪</span>
            No test results yet
            <div className="text-sm mt-2">Run a test to see detailed moderation analysis</div>
          </div>
        )}
      </div>
    </div>
  )
}

export default AdvancedPlayground