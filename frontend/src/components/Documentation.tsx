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
                <li>• Content moderation with regex filters</li>
                <li>• Real-time kill switch controls</li>
                <li>• Comprehensive request logging</li>
                <li>• Live configuration management</li>
                <li>• Interactive testing playground</li>
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
          
          <div className="space-y-6">
            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-3">Main Endpoint</h3>
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
            </div>

            <div>
              <h3 className="text-xl font-semibold text-gray-900 mb-3">Admin API Endpoints</h3>
              <div className="grid grid-cols-1 gap-4">
                {[
                  { method: 'GET', endpoint: '/api/status', desc: 'System health and status information' },
                  { method: 'GET', endpoint: '/api/config', desc: 'Current middleware configuration' },
                  { method: 'PUT', endpoint: '/api/config', desc: 'Update middleware configuration' },
                  { method: 'GET', endpoint: '/api/logs', desc: 'Request logs with optional filtering' },
                  { method: 'GET', endpoint: '/api/killswitch', desc: 'View blocked users and sessions' },
                  { method: 'POST', endpoint: '/api/killswitch', desc: 'Block or unblock users/sessions' },
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
                      { var: 'OPENAI_API_KEY', desc: 'OpenAI API key for moderation', default: '(none)' },
                      { var: 'QT1_LOG_FILE', desc: 'Log file path', default: 'logs/qt1.log' },
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

moderation:
  enabled: true
  use_openai: false
  severity: "medium"
  blocked_words:
    - "violence"
    - "hate"
    - "explicit"

relevance:
  enabled: false
  threshold: 0.7
  use_openai: true

kill_switch:
  enabled: true
  blocked_users: []
  blocked_sessions: []

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
        <section className="bg-gray-900 text-white rounded-lg p-8">
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