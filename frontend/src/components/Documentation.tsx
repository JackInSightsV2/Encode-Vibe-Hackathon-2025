import React from 'react'

const Documentation: React.FC = () => {
  return (
    <div className="max-w-4xl mx-auto">
      <header className="mb-12">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">QT-1 Documentation</h1>
        <p className="text-xl text-gray-600">Complete guide to the QT-1 Responsible AI Middleware</p>
      </header>

      <div className="space-y-12">
        {/* Overview */}
        <section className="bg-white rounded-lg shadow-lg p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Overview</h2>
          <p className="text-gray-700 mb-4">
            QT-1 is a high-performance AI middleware that acts as a responsible proxy between users and AI services. 
            It implements the QT-1 Framework's seven laws to ensure ethical AI interactions through real-time content 
            moderation, relevance filtering, and comprehensive monitoring.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6">
            <div className="bg-blue-50 p-4 rounded-lg">
              <h3 className="font-semibold text-blue-900 mb-2">Key Features</h3>
              <ul className="text-blue-800 space-y-1">
                <li>• Multi-layer content moderation (regex, PII, AI-powered)</li>
                <li>• Real-time security protection (DDoS, rate limiting, IP blocking)</li>
                <li>• Advanced analytics and metrics collection</li>
                <li>• AI provider management and health monitoring</li>
                <li>• Kill switch controls for emergency blocking</li>
                <li>• WebSocket support for real-time monitoring</li>
                <li>• Comprehensive request logging and audit trails</li>
                <li>• Opik integration for AI observability</li>
              </ul>
            </div>
            
            <div className="bg-green-50 p-4 rounded-lg">
              <h3 className="font-semibold text-green-900 mb-2">QT-1 Framework Laws</h3>
              <ul className="text-green-800 space-y-1">
                <li>• Do No Harm</li>
                <li>• Respect Human Autonomy</li>
                <li>• Be Transparent & Honest</li>
                <li>• Accept Oversight</li>
                <li>• Stay Within Scope</li>
                <li>• Protect Privacy</li>
                <li>• Ensure Accountability</li>
              </ul>
            </div>
          </div>
        </section>

        {/* Quick Start */}
        <section className="bg-white rounded-lg shadow-lg p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Quick Start Guide</h2>
          
          <div className="space-y-6">
            <div className="border-l-4 border-blue-500 pl-4">
              <h3 className="font-semibold text-gray-900 mb-2">1. Configure Target Service</h3>
              <p className="text-gray-700 mb-2">
                Set your AI service endpoint in the Configuration tab or update the target_url in config.yaml:
              </p>
              <pre className="bg-gray-100 p-3 rounded text-sm overflow-x-auto">
                <code>{`server:
  target_url: "http://your-ai-service:8081/chat"`}</code>
              </pre>
            </div>

            <div className="border-l-4 border-green-500 pl-4">
              <h3 className="font-semibold text-gray-900 mb-2">2. Test in Playground</h3>
              <p className="text-gray-700">
                Use the Playground tab to send test requests and verify moderation rules are working correctly.
              </p>
            </div>

            <div className="border-l-4 border-purple-500 pl-4">
              <h3 className="font-semibold text-gray-900 mb-2">3. Monitor Logs</h3>
              <p className="text-gray-700">
                View real-time request logs in the Logs tab to monitor system activity and blocked requests.
              </p>
            </div>

            <div className="border-l-4 border-red-500 pl-4">
              <h3 className="font-semibold text-gray-900 mb-2">4. Emergency Controls</h3>
              <p className="text-gray-700">
                Use the Kill Switch tab to immediately block specific users or sessions if needed.
              </p>
            </div>
          </div>
        </section>

        {/* API Reference */}
        <section className="bg-white rounded-lg shadow-lg p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">API Reference</h2>
          
          <div className="space-y-8">
            {/* Core Endpoints */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Core Endpoints</h3>
              <div className="space-y-4">
                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="flex items-center mb-2">
                    <span className="bg-blue-600 text-white px-2 py-1 rounded text-sm font-mono mr-3">POST</span>
                    <code className="text-lg">/chat</code>
                  </div>
                  <p className="text-gray-700 mb-3">Main proxy endpoint for AI interactions</p>
                  
                  <h4 className="font-semibold mb-2">Request Body:</h4>
                  <pre className="bg-gray-100 p-3 rounded text-sm overflow-x-auto mb-3">
                    <code>{`{
  "user_id": "string",
  "session_id": "string", 
  "message": "string"
}`}</code>
                  </pre>

                  <h4 className="font-semibold mb-2">Example:</h4>
                  <pre className="bg-gray-100 p-3 rounded text-sm overflow-x-auto">
                    <code>{`curl -X POST http://localhost:8080/chat \\
  -H "Content-Type: application/json" \\
  -d '{
    "user_id": "user123",
    "session_id": "session456", 
    "message": "Hello, how can you help me?"
  }'`}</code>
                  </pre>
                </div>

                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="flex items-center mb-2">
                    <span className="bg-green-600 text-white px-2 py-1 rounded text-sm font-mono mr-3">GET</span>
                    <code className="text-lg">/health</code>
                  </div>
                  <p className="text-gray-700">Health check endpoint</p>
                </div>

                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="flex items-center mb-2">
                    <span className="bg-green-600 text-white px-2 py-1 rounded text-sm font-mono mr-3">GET</span>
                    <code className="text-lg">/ws</code>
                  </div>
                  <p className="text-gray-700">WebSocket endpoint for real-time updates</p>
                </div>
              </div>
            </div>

            {/* System Management */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">System Management</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/status', desc: 'System health and status information' },
                  { method: 'GET', endpoint: '/api/config', desc: 'Current middleware configuration' },
                  { method: 'PUT', endpoint: '/api/config', desc: 'Update middleware configuration' },
                  { method: 'GET', endpoint: '/api/logs', desc: 'Request logs with optional filtering' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method === 'GET' ? 'bg-green-600' : 
                      api.method === 'POST' ? 'bg-blue-600' : 'bg-purple-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Authentication */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Authentication</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'POST', endpoint: '/api/auth/login', desc: 'User authentication' },
                  { method: 'POST', endpoint: '/api/auth/register', desc: 'User registration' },
                  { method: 'POST', endpoint: '/api/auth/logout', desc: 'User logout' },
                  { method: 'POST', endpoint: '/api/auth/refresh', desc: 'Refresh authentication token' },
                  { method: 'GET', endpoint: '/api/auth/me', desc: 'Get current user information' },
                  { method: 'POST', endpoint: '/api/auth/change-password', desc: 'Change user password' },
                  { method: 'GET', endpoint: '/api/auth/validate', desc: 'Validate authentication token' },
                  { method: 'GET', endpoint: '/api/auth/sessions', desc: 'Get user sessions' },
                  { method: 'POST', endpoint: '/api/auth/sessions/invalidate', desc: 'Invalidate a session' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method === 'GET' ? 'bg-green-600' : 'bg-blue-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Security & Protection */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Security & Protection</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/ip-protection/status', desc: 'IP protection status and metrics' },
                  { method: 'GET/PUT', endpoint: '/api/ip-protection/config', desc: 'IP protection configuration' },
                  { method: 'GET', endpoint: '/api/ip-protection/geographic-stats', desc: 'Geographic IP statistics' },
                  { method: 'GET', endpoint: '/api/ddos-protection/status', desc: 'DDoS protection status' },
                  { method: 'GET/PUT', endpoint: '/api/ddos-protection/config', desc: 'DDoS protection configuration' },
                  { method: 'GET', endpoint: '/api/ddos-protection/metrics', desc: 'DDoS protection metrics' },
                  { method: 'GET', endpoint: '/api/ddos-protection/threat-analysis', desc: 'DDoS threat analysis' },
                  { method: 'POST', endpoint: '/api/ddos-protection/emergency-mode', desc: 'Toggle emergency mode' },
                  { method: 'POST', endpoint: '/api/ddos-protection/circuit-breaker/reset', desc: 'Reset circuit breaker' },
                  { method: 'GET', endpoint: '/api/rate-limits/status', desc: 'Rate limiting status' },
                  { method: 'GET/PUT', endpoint: '/api/rate-limits/config', desc: 'Rate limiting configuration' },
                  { method: 'POST', endpoint: '/api/rate-limits/block-ip', desc: 'Block an IP address' },
                  { method: 'POST', endpoint: '/api/rate-limits/unblock-ip', desc: 'Unblock an IP address' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method.includes('GET') ? 'bg-green-600' : 
                      api.method === 'POST' ? 'bg-blue-600' : 'bg-purple-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Content Moderation */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Content Moderation</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/moderation/stats', desc: 'Moderation statistics and analytics' },
                  { method: 'GET', endpoint: '/api/moderation/pii-analytics', desc: 'PII detection analytics' },
                  { method: 'GET', endpoint: '/api/moderation/rules/stats', desc: 'Moderation rules statistics' },
                  { method: 'GET', endpoint: '/api/moderation/layers', desc: 'Available moderation layers' },
                  { method: 'POST', endpoint: '/api/moderation/test', desc: 'Test moderation engine' },
                  { method: 'GET/PUT', endpoint: '/api/config/moderation', desc: 'Moderation configuration' },
                  { method: 'GET/PUT', endpoint: '/api/config/rules', desc: 'Rules engine configuration' },
                  { method: 'GET/PUT', endpoint: '/api/config/pii', desc: 'PII detection configuration' },
                  { method: 'GET', endpoint: '/api/rules', desc: 'Get moderation rules' },
                  { method: 'POST', endpoint: '/api/rules/create', desc: 'Create new moderation rule' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method.includes('GET') ? 'bg-green-600' : 
                      api.method === 'POST' ? 'bg-blue-600' : 'bg-purple-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* AI & Relevancy */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">AI & Relevancy</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET/PUT', endpoint: '/api/relevancy/config', desc: 'Relevancy filtering configuration' },
                  { method: 'GET/PUT', endpoint: '/api/relevancy/keywords', desc: 'Relevancy keywords management' },
                  { method: 'POST', endpoint: '/api/relevancy/test', desc: 'Test relevancy filtering' },
                  { method: 'GET/POST', endpoint: '/api/relevancy/ai-provider', desc: 'AI provider configuration for relevancy' },
                  { method: 'POST', endpoint: '/api/relevancy/ai-test', desc: 'Test AI provider connectivity' },
                  { method: 'GET/POST', endpoint: '/api/regex/ai-provider', desc: 'AI provider for regex enhancement' },
                  { method: 'POST', endpoint: '/api/regex/ai-test', desc: 'Test regex AI capabilities' },
                  { method: 'GET', endpoint: '/api/regex/stats', desc: 'Regex processing statistics' },
                  { method: 'GET/POST', endpoint: '/api/pii/ai-provider', desc: 'AI provider for PII detection' },
                  { method: 'POST', endpoint: '/api/pii/ai-test', desc: 'Test PII AI detection' },
                  { method: 'GET', endpoint: '/api/pii/stats', desc: 'PII detection statistics' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method.includes('GET') ? 'bg-green-600' : 
                      api.method === 'POST' ? 'bg-blue-600' : 'bg-purple-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Metrics & Analytics */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Metrics & Analytics</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/metrics', desc: 'System metrics and performance data' },
                  { method: 'GET', endpoint: '/api/metrics/summary', desc: 'Metrics summary dashboard' },
                  { method: 'GET', endpoint: '/api/metrics/health', desc: 'Metrics system health' },
                  { method: 'GET', endpoint: '/api/metrics/stats', desc: 'Detailed metrics statistics' },
                  { method: 'GET', endpoint: '/api/metrics/system/snapshot', desc: 'System performance snapshot' },
                  { method: 'GET', endpoint: '/api/metrics/providers', desc: 'Provider health metrics' },
                  { method: 'GET', endpoint: '/api/metrics/timeseries', desc: 'Time-series metrics data' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method === 'GET' ? 'bg-green-600' : 'bg-blue-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* AI Optimization */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">AI Optimization</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/optimizer/status', desc: 'Optimizer system status' },
                  { method: 'GET', endpoint: '/api/optimizer/experiments', desc: 'Active optimization experiments' },
                  { method: 'GET', endpoint: '/api/optimizer/drift/alerts', desc: 'Model drift detection alerts' },
                  { method: 'GET', endpoint: '/api/optimizer/rollouts', desc: 'Model rollout status' },
                  { method: 'GET', endpoint: '/api/optimizer/metrics/historical', desc: 'Historical optimization metrics' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className="bg-green-600 text-white px-2 py-1 rounded text-xs font-mono mr-3">
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Provider Management */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Provider Management</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/providers', desc: 'Available AI providers and their status' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className="bg-green-600 text-white px-2 py-1 rounded text-xs font-mono mr-3">
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Kill Switch & Emergency Controls */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Kill Switch & Emergency Controls</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/killswitch/status', desc: 'Kill switch status and blocked entities' },
                  { method: 'GET/POST', endpoint: '/api/killswitch', desc: 'Manage user/session blocking' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method.includes('GET') ? 'bg-green-600' : 'bg-red-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Opik Integration */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">Opik Integration (AI Observability)</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/opik/status', desc: 'Opik integration status' },
                  { method: 'GET', endpoint: '/api/opik/traces', desc: 'AI interaction traces' },
                  { method: 'GET', endpoint: '/api/opik/evaluations', desc: 'AI model evaluations' },
                  { method: 'GET/PUT', endpoint: '/api/opik/config', desc: 'Opik configuration' },
                  { method: 'POST', endpoint: '/api/opik/test-connection', desc: 'Test Opik connectivity' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method.includes('GET') ? 'bg-green-600' : 
                      api.method === 'POST' ? 'bg-blue-600' : 'bg-purple-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* User Management */}
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-4">User Management</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/users', desc: 'Get all users' },
                  { method: 'POST', endpoint: '/api/users/create', desc: 'Create new user' },
                ].map((api, index) => (
                  <div key={index} className="bg-gray-50 rounded p-3 flex items-center">
                    <span className={`px-2 py-1 rounded text-xs font-mono mr-3 text-white ${
                      api.method === 'GET' ? 'bg-green-600' : 'bg-blue-600'
                    }`}>
                      {api.method}
                    </span>
                    <code className="font-mono mr-4">{api.endpoint}</code>
                    <span className="text-gray-600">{api.desc}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* Configuration */}
        <section className="bg-white rounded-lg shadow-lg p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Configuration Guide</h2>
          
          <div className="space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-3">Environment Variables</h3>
              <div className="overflow-x-auto">
                <table className="min-w-full border-collapse border border-gray-300">
                  <thead>
                    <tr className="bg-gray-50">
                      <th className="border border-gray-300 px-4 py-2 text-left">Variable</th>
                      <th className="border border-gray-300 px-4 py-2 text-left">Description</th>
                      <th className="border border-gray-300 px-4 py-2 text-left">Default</th>
                    </tr>
                  </thead>
                  <tbody>
                    {[
                      { var: 'QT1_PORT', desc: 'Server port', default: '8080' },
                      { var: 'QT1_HOST', desc: 'Server host', default: 'localhost' },
                      { var: 'QT1_TARGET_URL', desc: 'Downstream AI service URL', default: 'http://localhost:8081' },
                      { var: 'OPENAI_API_KEY', desc: 'OpenAI API key for AI-enhanced moderation', default: '(none)' },
                      { var: 'ANTHROPIC_API_KEY', desc: 'Anthropic API key for Claude models', default: '(none)' },
                      { var: 'QT1_LOG_FILE', desc: 'Log file path', default: 'logs/qt1.log' },
                      { var: 'QT1_DB_TYPE', desc: 'Database type (sqlite, postgres)', default: 'sqlite' },
                      { var: 'QT1_DB_PATH', desc: 'Database file path (SQLite)', default: 'storage/qt1.db' },
                      { var: 'OPIK_API_KEY', desc: 'Opik API key for observability', default: '(none)' },
                      { var: 'OPIK_PROJECT_NAME', desc: 'Opik project name', default: 'qt1-middleware' },
                    ].map((env, index) => (
                      <tr key={index}>
                        <td className="border border-gray-300 px-4 py-2 font-mono">{env.var}</td>
                        <td className="border border-gray-300 px-4 py-2">{env.desc}</td>
                        <td className="border border-gray-300 px-4 py-2 font-mono text-sm">{env.default}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold text-gray-900 mb-3">Sample Configuration File</h3>
              <pre className="bg-gray-100 p-4 rounded text-sm overflow-x-auto">
                <code>{`server:
  port: 8080
  host: "localhost"
  target_url: "http://localhost:8081"

database:
  type: "sqlite"
  sqlite_file: "storage/qt1.db"
  # For PostgreSQL:
  # postgres_host: "localhost"
  # postgres_port: 5432
  # postgres_user: "qt1_user"
  # postgres_password: "secure_password"
  # postgres_database: "qt1_db"

moderation:
  enabled: true
  use_openai: false
  severity: "medium"
  blocked_words:
    - "violence"
    - "hate"
    - "explicit"
  pii_detection:
    enabled: true
    ai_enhanced: false

relevance:
  enabled: false
  threshold: 0.7
  use_openai: true

security:
  rate_limiting:
    enabled: true
    global:
      requests_per_second: 100
      burst: 200
  ip_protection:
    enabled: true
    enable_geoblocking: false
    enable_reputation_check: true
  ddos_protection:
    enabled: true
    spike_threshold: 1000
    enable_throttling: true

kill_switch:
  enabled: true
  blocked_users: []
  blocked_sessions: []

metrics:
  enabled: true
  storage:
    type: "memory"  # or "supabase"

opik:
  enabled: false
  api_key: ""
  project_name: "qt1-middleware"
  base_url: "https://cloud.opik.io"

logging:
  enabled: true
  log_file: "logs/qt1.log"
  log_level: "info"`}</code>
              </pre>
            </div>
          </div>
        </section>

        {/* Troubleshooting */}
        <section className="bg-white rounded-lg shadow-lg p-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">Troubleshooting</h2>
          
          <div className="space-y-4">
            {[
              {
                problem: "Connection refused to target service",
                solution: "Ensure your AI service is running on the configured target_url port, or update the target_url in configuration."
              },
              {
                problem: "Frontend shows blank screen",
                solution: "Check browser console for MIME type errors. Rebuild frontend with 'npm run build' and restart middleware."
              },
              {
                problem: "Requests being blocked unexpectedly", 
                solution: "Check the blocked words list in Configuration. Review logs to see which moderation rule triggered."
              },
              {
                problem: "Kill switch not working",
                solution: "Verify kill_switch is enabled in configuration and the user_id/session_id match exactly."
              },
              {
                problem: "Database connection errors",
                solution: "Check database configuration and ensure the database file/server is accessible. For SQLite, verify the storage directory exists."
              },
              {
                problem: "High memory usage",
                solution: "Check metrics configuration and consider using external storage like Supabase for large-scale deployments."
              },
              {
                problem: "Authentication not working",
                solution: "Ensure database is properly configured and auth endpoints are enabled. Check JWT secret configuration."
              },
              {
                problem: "Rate limiting too aggressive",
                solution: "Adjust rate limiting thresholds in security configuration. Use /api/rate-limits/config to fine-tune settings."
              }
            ].map((item, index) => (
              <div key={index} className="border-l-4 border-yellow-500 bg-yellow-50 p-4">
                <h3 className="font-semibold text-yellow-900 mb-2">Problem: {item.problem}</h3>
                <p className="text-yellow-800">Solution: {item.solution}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Support */}
        <section className="bg-gray-900 text-white rounded-lg shadow-lg p-8">
          <h2 className="text-2xl font-bold mb-4">Support & Resources</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="font-semibold mb-2">Framework Information</h3>
              <p className="text-gray-300 mb-2">
                QT-1 implements a comprehensive responsible AI framework designed for production deployment.
              </p>
              <p className="text-gray-300">
                Built for the Encode Vibe Hackathon 2025 as a demonstration of runtime-enforceable AI ethics.
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-2">System Status</h3>
              <p className="text-gray-300">
                Monitor system health in real-time through the Dashboard tab, and view detailed logs 
                in the Logs section for troubleshooting and audit purposes.
              </p>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}

export default Documentation