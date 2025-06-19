import { useState, useEffect } from 'react'
import Dashboard from './components/Dashboard'
import Configuration from './components/Configuration'
import Playground from './components/Playground'
import Logs from './components/Logs'
import KillSwitch from './components/KillSwitch'
import Documentation from './components/Documentation'
import ConnectionStatus from './components/ConnectionStatus'
import { WebSocketProvider } from './contexts/WebSocketContext'

type Page = 'dashboard' | 'config' | 'playground' | 'logs' | 'killswitch' | 'docs'

function App() {
  const [currentPage, setCurrentPage] = useState<Page>('dashboard')
  const [systemStatus, setSystemStatus] = useState<any>(null)

  useEffect(() => {
    fetchSystemStatus()
    const interval = setInterval(fetchSystemStatus, 10000) // Update every 10s
    return () => clearInterval(interval)
  }, [])

  const fetchSystemStatus = async () => {
    try {
      const response = await fetch('/api/status')
      if (response.ok) {
        const data = await response.json()
        setSystemStatus(data.data)
      }
    } catch (error) {
      console.error('Failed to fetch system status:', error)
    }
  }

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard':
        return <Dashboard systemStatus={systemStatus} />
      case 'config':
        return <Configuration />
      case 'playground':
        return <Playground />
      case 'logs':
        return <Logs />
      case 'killswitch':
        return <KillSwitch />
      case 'docs':
        return <Documentation />
      default:
        return <Dashboard systemStatus={systemStatus} />
    }
  }

  const navigationItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '📊' },
    { id: 'playground', label: 'Playground', icon: '🧪' },
    { id: 'config', label: 'Configuration', icon: '⚙️' },
    { id: 'logs', label: 'Logs', icon: '📝' },
    { id: 'killswitch', label: 'Kill Switch', icon: '🛑' },
    { id: 'docs', label: 'Documentation', icon: '📚' },
  ]

  return (
    <WebSocketProvider autoConnect={true}>
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100">
      {/* Header */}
      <header className="bg-white shadow-lg border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo and Status */}
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-3">
                <div className="w-10 h-10 bg-gradient-to-r from-blue-600 to-blue-700 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-lg">Q</span>
                </div>
                <div>
                  <h1 className="text-xl font-bold text-slate-900">QT-1 Middleware</h1>
                  <p className="text-xs text-slate-500">Responsible AI Framework</p>
                </div>
              </div>
              
              {/* Status Indicators */}
              <div className="flex items-center space-x-4">
                {/* System Status */}
                <div className="flex items-center space-x-2">
                  <div className={`w-2 h-2 rounded-full ${
                    systemStatus?.status === 'healthy' ? 'bg-green-500' : 'bg-red-500'
                  } animate-pulse`}></div>
                  <span className={`text-sm font-medium ${
                    systemStatus?.status === 'healthy' ? 'text-green-700' : 'text-red-700'
                  }`}>
                    {systemStatus?.status === 'healthy' ? 'System Online' : 'System Offline'}
                  </span>
                </div>
                
                {/* WebSocket Connection Status */}
                <ConnectionStatus />
              </div>
            </div>

            {/* Version Info */}
            <div className="hidden md:flex items-center space-x-4 text-sm text-slate-600">
              <span>v{systemStatus?.version || '1.0.0'}</span>
              <span>•</span>
              <span>Port {systemStatus?.port || '8080'}</span>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b border-slate-200 sticky top-0 z-40 shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex space-x-1 overflow-x-auto py-2">
            {navigationItems.map((item) => (
              <button
                key={item.id}
                onClick={() => setCurrentPage(item.id as Page)}
                className={`flex items-center space-x-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 whitespace-nowrap ${
                  currentPage === item.id
                    ? 'bg-blue-600 text-white shadow-md'
                    : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
                }`}
              >
                <span className="text-base">{item.icon}</span>
                <span>{item.label}</span>
              </button>
            ))}
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="bg-white rounded-xl shadow-lg border border-slate-200 overflow-hidden">
          <div className="p-6 lg:p-8">
            {renderPage()}
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="bg-slate-900 text-slate-300 py-8 mt-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center">
            <div className="mb-4 md:mb-0">
              <p className="text-sm">© 2025 QT-1 Responsible AI Middleware</p>
              <p className="text-xs text-slate-400">Built for Encode Vibe Hackathon 2025</p>
            </div>
            <div className="flex items-center space-x-6 text-sm">
              <span>Framework Laws: 7</span>
              <span>•</span>
              <span>Runtime Enforced</span>
              <span>•</span>
              <span>Production Ready</span>
            </div>
          </div>
        </div>
      </footer>
      </div>
    </WebSocketProvider>
  )
}

export default App