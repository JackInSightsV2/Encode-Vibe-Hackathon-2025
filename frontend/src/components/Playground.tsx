import React, { useState } from 'react'

interface ChatRequest {
  user_id: string
  session_id: string
  message: string
  target_url?: string
  provider?: string
  model?: string
}

interface ChatResponse {
  success: boolean
  message?: string
  error?: string
  blocked?: boolean
  reason?: string
  model?: string
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
}

const Playground: React.FC = () => {
  const [request, setRequest] = useState<ChatRequest>({
    user_id: 'test_user',
    session_id: 'test_session',
    message: 'Hello, how are you?',
    target_url: 'http://localhost:8081',
    provider: 'openai',
    model: 'gpt-4.1-nano'
  })
  const [response, setResponse] = useState<ChatResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)

  const sendRequest = async () => {
    setLoading(true)
    setResponse(null)
    
    try {
      // If using direct provider (OpenAI/Anthropic), use the test endpoint
      if (request.provider && request.provider !== 'custom') {
        const res = await fetch('/test-openai', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            user_id: request.user_id,
            session_id: request.session_id,
            message: request.message
          }),
        });
        
        const data = await res.json();
        
        // Handle blocked responses
        if (data.blocked) {
          setResponse({
            success: false,
            blocked: true,
            reason: data.reason || 'Content was blocked',
            message: data.message || 'Your request was blocked by the moderation system.'
          });
        } 
        // Handle error responses
        else if (!data.success) {
          setResponse({
            success: false,
            error: data.error || 'An unknown error occurred',
            message: data.message
          });
        } 
        // Handle successful responses
        else {
          setResponse({
            success: true,
            message: data.message,
            model: data.model,
            usage: data.usage,
            ...(data.reason && { reason: data.reason })
          });
        }
      } else {
        // Fall back to regular chat endpoint for custom targets
        const res = await fetch('/chat', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(request),
        });
        
        const data = await res.json();
        setResponse(data);
      }
      
    } catch (error) {
      setResponse({
        success: false,
        error: 'Network error: ' + (error as Error).message
      });
    } finally {
      setLoading(false);
    }
  }

  const testScenarios = [
    {
      name: 'OpenAI API Test',
      description: 'Test direct OpenAI API integration using the test endpoint',
      request: { 
        user_id: 'test_user', 
        session_id: 'test_session', 
        message: 'Tell me a fun fact about space',
        provider: 'openai',
        model: 'gpt-4o-mini'
      }
    },
    {
      name: 'Custom Endpoint Test',
      description: 'Test with a custom AI service endpoint',
      request: { 
        user_id: 'user1', 
        session_id: 'session1', 
        message: 'Hello, how are you today?', 
        target_url: 'http://localhost:8081',
        provider: 'custom'
      }
    },
    {
      name: 'Mock Service Test',
      description: 'Test with the mock AI service (must be running on port 9000)',
      request: { 
        user_id: 'user2', 
        session_id: 'session2', 
        message: 'Hello from test', 
        target_url: 'http://localhost:9000',
        provider: 'custom'
      }
    },
    {
      name: 'API Validation Test',
      description: 'Test API key validation and client initialization',
      request: { 
        user_id: 'user3', 
        session_id: 'session3', 
        message: 'Just testing API connection',
        provider: 'openai'
      }
    }
  ]

  const loadScenario = (scenario: typeof testScenarios[0]) => {
    setRequest(scenario.request)
  }

  return (
    <div>
      <header className="mb-8 border-b border-slate-200 pb-6">
        <h1 className="text-3xl font-bold text-slate-900 mb-2">Interactive Testing Playground</h1>
        <p className="text-slate-600 mb-4">Test your QT-1 middleware with different scenarios to verify moderation and filtering behavior</p>
        
        {/* Important Information Banner */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div className="flex items-start space-x-3">
            <div className="flex-shrink-0">
              <span className="text-blue-600 text-lg">ℹ️</span>
            </div>
            <div className="flex-1">
              <h3 className="text-sm font-semibold text-blue-900 mb-1">Testing Options</h3>
              <div className="text-blue-800 text-sm space-y-1">
                <p><strong>Option 1:</strong> Run the included mock AI service: <code className="bg-blue-100 px-1 rounded">go run mock-ai-service.go</code></p>
                <p><strong>Option 2:</strong> Configure your own AI service endpoint below</p>
                <p><strong>Option 3:</strong> Use direct provider integration (requires API keys in config)</p>
              </div>
            </div>
          </div>
        </div>
      </header>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-bold mb-4">Request Builder</h2>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">User ID</label>
              <input
                type="text"
                value={request.user_id}
                onChange={(e) => setRequest({...request, user_id: e.target.value})}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Session ID</label>
              <input
                type="text"
                value={request.session_id}
                onChange={(e) => setRequest({...request, session_id: e.target.value})}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Message</label>
              <textarea
                value={request.message}
                onChange={(e) => setRequest({...request, message: e.target.value})}
                rows={4}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Enter your test message here..."
              />
            </div>
            
            {/* Advanced Configuration Toggle */}
            <div className="border-t pt-4">
              <button
                type="button"
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="flex items-center text-sm font-medium text-blue-600 hover:text-blue-700"
              >
                <span className={`mr-2 transition-transform ${showAdvanced ? 'rotate-90' : ''}`}>▶</span>
                Advanced Configuration
              </button>
            </div>
            
            {/* Advanced Configuration Panel */}
            {showAdvanced && (
              <div className="space-y-4 bg-gray-50 p-4 rounded-md">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Target URL</label>
                  <input
                    type="text"
                    value={request.target_url || ''}
                    onChange={(e) => setRequest({...request, target_url: e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="http://localhost:8081"
                  />
                  <p className="text-xs text-gray-500 mt-1">Leave empty to use provider routing</p>
                </div>
                
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Provider</label>
                    <select
                      value={request.provider || ''}
                      onChange={(e) => setRequest({...request, provider: e.target.value})}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">Auto (use target_url)</option>
                      <option value="openai">OpenAI</option>
                      <option value="anthropic">Anthropic</option>
                      <option value="custom">Custom</option>
                    </select>
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Model</label>
                    <input
                      type="text"
                      value={request.model || ''}
                      onChange={(e) => setRequest({...request, model: e.target.value})}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="gpt-4o-mini"
                    />
                  </div>
                </div>
              </div>
            )}
            
            <button
              onClick={sendRequest}
              disabled={loading}
              className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? 'Sending...' : 'Send Request'}
            </button>
          </div>
          
          <div className="mt-6">
            <h3 className="text-lg font-semibold mb-3">Test Scenarios</h3>
            <div className="grid grid-cols-1 gap-2">
              {testScenarios.map((scenario, index) => (
                <button
                  key={index}
                  onClick={() => loadScenario(scenario)}
                  className="text-left p-3 border border-gray-200 rounded-md hover:bg-gray-50 transition-colors"
                >
                  <div className="font-medium text-sm">{scenario.name}</div>
                  <div className="text-xs text-gray-600 mb-1">{scenario.description}</div>
                  <div className="text-xs text-gray-500 truncate">
                    {scenario.request.message}
                  </div>
                  {scenario.request.target_url && (
                    <div className="text-xs text-blue-500 mt-1">
                      → {scenario.request.target_url}
                    </div>
                  )}
                  {scenario.request.provider && (
                    <div className="text-xs text-green-500 mt-1">
                      Provider: {scenario.request.provider}
                    </div>
                  )}
                </button>
              ))}
            </div>
          </div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-bold mb-4">Response</h2>
          
          {response ? (
            <div className="space-y-4">
              {/* Status Banner */}
              <div className={`p-4 rounded-lg ${
                response.blocked 
                  ? 'bg-yellow-50 border-2 border-yellow-300' 
                  : response.success 
                    ? 'bg-green-50 border-2 border-green-300'
                    : 'bg-red-50 border-2 border-red-300'
              }`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <span className="text-2xl mr-3">
                      {response.blocked 
                        ? '⚠️' 
                        : response.success 
                          ? '✅' 
                          : '❌'}
                    </span>
                    <h3 className="text-lg font-semibold">
                      {response.blocked 
                        ? 'Content Blocked'
                        : response.success 
                          ? 'Request Successful'
                          : 'Error Occurred'}
                    </h3>
                  </div>
                  {response.model && (
                    <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs font-medium rounded">
                      {response.model}
                    </span>
                  )}
                </div>
                
                {/* Main Message */}
                {response.message && (
                  <div className="mt-4 p-3 bg-white rounded border border-gray-200">
                    <p className="text-gray-800">{response.message}</p>
                  </div>
                )}
                
                {/* Block Reason - More Prominent */}
                {response.blocked && response.reason && (
                  <div className="mt-4 p-3 bg-yellow-50 border-l-4 border-yellow-400">
                    <div className="flex">
                      <div className="flex-shrink-0">
                        <svg className="h-5 w-5 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                        </svg>
                      </div>
                      <div className="ml-3">
                        <h4 className="text-sm font-medium text-yellow-800">Block Reason</h4>
                        <div className="mt-1 text-sm text-yellow-700">
                          <p>{response.reason}</p>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
                
                {/* Error Details - More Prominent */}
                {!response.blocked && response.error && (
                  <div className="mt-4 p-3 bg-red-50 border-l-4 border-red-400">
                    <div className="flex">
                      <div className="flex-shrink-0">
                        <svg className="h-5 w-5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                        </svg>
                      </div>
                      <div className="ml-3">
                        <h4 className="text-sm font-medium text-red-800">Error Details</h4>
                        <div className="mt-1 text-sm text-red-700">
                          <p>{response.error}</p>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
                
                {/* Token Usage - Only show for successful responses */}
                {response.success && response.usage && (
                  <div className="mt-4 pt-3 border-t border-gray-200">
                    <h4 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Token Usage</h4>
                    <div className="grid grid-cols-3 gap-3 text-sm">
                      <div className="bg-gray-50 p-2 rounded">
                        <div className="text-xs text-gray-500">Prompt</div>
                        <div className="font-mono text-gray-900">{response.usage.prompt_tokens}</div>
                      </div>
                      <div className="bg-gray-50 p-2 rounded">
                        <div className="text-xs text-gray-500">Completion</div>
                        <div className="font-mono text-gray-900">{response.usage.completion_tokens}</div>
                      </div>
                      <div className="bg-gray-50 p-2 rounded">
                        <div className="text-xs text-gray-500">Total</div>
                        <div className="font-mono text-gray-900">{response.usage.total_tokens}</div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
              
              <div className="bg-gray-50 p-4 rounded-md">
                <h4 className="font-semibold mb-2">Raw Response:</h4>
                <pre className="text-sm text-gray-700 overflow-x-auto">
                  {JSON.stringify(response, null, 2)}
                </pre>
              </div>
            </div>
          ) : (
            <div className="text-gray-500 text-center py-8">
              Send a request to see the response here
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default Playground