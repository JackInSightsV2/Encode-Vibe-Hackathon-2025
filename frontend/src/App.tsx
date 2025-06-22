import { useState, useEffect, Suspense, lazy } from 'react'
import AdvancedDashboard from './components/AdvancedDashboard'
import ConnectionStatus from './components/ConnectionStatus'
import AuthStatus from './components/auth/AuthStatus'
import ProtectedRoute from './components/auth/ProtectedRoute'
import ErrorBoundary from './components/ErrorBoundary'

// Lazy load non-critical components for better performance
const AdvancedConfiguration = lazy(() => import('./components/AdvancedConfiguration'))
const RuleManagement = lazy(() => import('./components/RuleManagement'))
const MetricsDashboard = lazy(() => import('./components/MetricsDashboard'))
const UserManagement = lazy(() => import('./components/UserManagement'))
const APIKeyManagement = lazy(() => import('./components/APIKeyManagement'))
const OptimizerDashboard = lazy(() => import('./components/OptimizerDashboard'))
const OpikAnalyticsDashboard = lazy(() => import('./components/OpikAnalyticsDashboard'))
const RateLimitingDashboard = lazy(() => import('./components/RateLimitingDashboard'))
const IPProtectionManager = lazy(() => import('./components/IPProtectionManager'))
const DDoSProtectionDashboard = lazy(() => import('./components/DDoSProtectionDashboard'))
const AdvancedPlayground = lazy(() => import('./components/AdvancedPlayground'))
const ModerationTester = lazy(() => import('./components/ModerationTester'))
const Logs = lazy(() => import('./components/Logs'))
const KillSwitch = lazy(() => import('./components/KillSwitch'))
const Documentation = lazy(() => import('./components/Documentation'))
import { AuthProvider, useAuth, ConditionalRender } from './contexts/AuthContext'
import { WebSocketProvider } from './contexts/WebSocketContext'
import { ThemeProvider } from './contexts/ThemeContext'
import { CompactThemeSelector, AnnouncementsProvider } from './components/ui'

type Page = 'dashboard' | 'metrics' | 'advanced-config' | 'rules' | 'playground' | 'tester' | 'logs' | 'killswitch' | 'docs' | 'users' | 'api-keys' | 'optimizer' | 'opik-analytics' | 'rate-limiting' | 'ip-protection' | 'ddos-protection'

// Main App component wrapped with authentication
function AppContent() {
  const { isAuthenticated, hasPermission } = useAuth();
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

  const LoadingSpinner = () => (
    <div className="flex items-center justify-center p-8">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <span className="ml-2 text-slate-600">Loading...</span>
    </div>
  )

  const renderPage = () => {
    switch (currentPage) {
      case 'dashboard':
        return <AdvancedDashboard systemStatus={systemStatus} />
      case 'metrics':
        return (
          <ProtectedRoute permission="metrics:read">
            <Suspense fallback={<LoadingSpinner />}>
              <MetricsDashboard />
            </Suspense>
          </ProtectedRoute>
        )
      case 'advanced-config':
        return (
          <ProtectedRoute permission="config:write">
            <Suspense fallback={<LoadingSpinner />}>
              <AdvancedConfiguration />
            </Suspense>
          </ProtectedRoute>
        )
      case 'rules':
        return (
          <ProtectedRoute permission="config:write">
            <Suspense fallback={<LoadingSpinner />}>
              <RuleManagement />
            </Suspense>
          </ProtectedRoute>
        )
      case 'playground':
        return (
          <ProtectedRoute permission="config:read">
            <Suspense fallback={<LoadingSpinner />}>
              <AdvancedPlayground />
            </Suspense>
          </ProtectedRoute>
        )
      case 'tester':
        return (
          <ProtectedRoute permission="config:read">
            <Suspense fallback={<LoadingSpinner />}>
              <ModerationTester />
            </Suspense>
          </ProtectedRoute>
        )
      case 'logs':
        return (
          <ProtectedRoute permission="logs:read">
            <Suspense fallback={<LoadingSpinner />}>
              <Logs />
            </Suspense>
          </ProtectedRoute>
        )
      case 'killswitch':
        return (
          <Suspense fallback={<LoadingSpinner />}>
            <KillSwitch />
          </Suspense>
        )
      case 'users':
        return (
          <ProtectedRoute permission="user:manage">
            <Suspense fallback={<LoadingSpinner />}>
              <UserManagement />
            </Suspense>
          </ProtectedRoute>
        )
      case 'api-keys':
        return (
          <ProtectedRoute permission="api_keys:read">
            <Suspense fallback={<LoadingSpinner />}>
              <APIKeyManagement />
            </Suspense>
          </ProtectedRoute>
        )
      case 'docs':
        return (
          <Suspense fallback={<LoadingSpinner />}>
            <Documentation />
          </Suspense>
        )
      case 'optimizer':
        return (
          <ProtectedRoute permission="config:read">
            <Suspense fallback={<LoadingSpinner />}>
              <OptimizerDashboard />
            </Suspense>
          </ProtectedRoute>
        )
      case 'opik-analytics':
        return (
          <ProtectedRoute permission="config:read">
            <Suspense fallback={<LoadingSpinner />}>
              <OpikAnalyticsDashboard />
            </Suspense>
          </ProtectedRoute>
        )
      case 'rate-limiting':
        return (
          <ProtectedRoute permission="security:read">
            <Suspense fallback={<LoadingSpinner />}>
              <RateLimitingDashboard />
            </Suspense>
          </ProtectedRoute>
        )
      case 'ip-protection':
        return (
          <ProtectedRoute permission="security:read">
            <Suspense fallback={<LoadingSpinner />}>
              <IPProtectionManager />
            </Suspense>
          </ProtectedRoute>
        )
      case 'ddos-protection':
        return (
          <ProtectedRoute permission="security:read">
            <Suspense fallback={<LoadingSpinner />}>
              <DDoSProtectionDashboard />
            </Suspense>
          </ProtectedRoute>
        )
      default:
        return <AdvancedDashboard systemStatus={systemStatus} />
    }
  }

  // Define main bar and hamburger menu items
  const getNavigationItems = () => {
    type NavItem = { id: string; label: string; icon: string; permission: string | null };
    
    const mainBarItems: NavItem[] = [
      { id: 'dashboard', label: 'Dashboard', icon: '📊', permission: null },
      { id: 'opik-analytics', label: 'Opik Analytics', icon: '🚀', permission: 'config:read' },
      { id: 'docs', label: 'Documentation', icon: '📚', permission: null },
    ];

    const hamburgerItems: NavItem[] = [
      { id: 'optimizer', label: 'Self-Optimizing Rules', icon: '🤖', permission: 'config:read' },
      { id: 'metrics', label: 'Real-Time Metrics', icon: '📈', permission: 'metrics:read' },
      { id: 'rate-limiting', label: 'Rate Limiting', icon: '🚦', permission: 'security:read' },
      { id: 'ip-protection', label: 'IP Protection', icon: '🌍', permission: 'security:read' },
      { id: 'ddos-protection', label: 'DDoS Protection', icon: '🛡️', permission: 'security:read' },
      { id: 'rules', label: 'Rule Management', icon: '⚙️', permission: 'config:write' },
      { id: 'advanced-config', label: 'Configuration', icon: '🔧', permission: 'config:write' },
      { id: 'playground', label: 'Playground', icon: '🧪', permission: 'config:read' },
      { id: 'tester', label: 'Moderation Tester', icon: '🔬', permission: 'config:read' },
      { id: 'logs', label: 'Logs', icon: '📝', permission: 'logs:read' },
      { id: 'killswitch', label: 'Kill Switch', icon: '🛑', permission: 'killswitch:manage' },
      { id: 'users', label: 'User Management', icon: '👥', permission: 'user:manage' },
      { id: 'api-keys', label: 'API Keys', icon: '🔑', permission: 'api_keys:read' },
    ];

    const filterItems = (items: NavItem[]) => {
      if (!isAuthenticated) {
        return items.filter(item => !item.permission);
      }
      return items.filter(item => !item.permission || hasPermission(item.permission));
    };

    return {
      mainBar: filterItems(mainBarItems),
      hamburger: filterItems(hamburgerItems)
    };
  };

  const navigationItems = getNavigationItems();
  const [hamburgerOpen, setHamburgerOpen] = useState(false);

  return (
    <AnnouncementsProvider>
      <WebSocketProvider autoConnect={true}>
        <div className="min-h-screen bg-theme-background theme-transition">
      {/* Header */}
      <header className="bg-theme-surface shadow-theme-lg border-b border-theme-border-default theme-transition">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo and Status */}
            <div className="flex items-center space-x-4">
              <div className="flex items-center space-x-3">
                <div className="w-10 h-10 bg-gradient-to-r from-blue-600 to-blue-700 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-lg">Q</span>
                </div>
                <div>
                  <h1 className="text-xl font-bold text-theme-text-primary theme-transition">QT-1 Advanced Middleware</h1>
                  <p className="text-xs text-theme-text-secondary theme-transition">Multi-Layer AI Content Moderation</p>
                </div>
              </div>
              
              {/* Status Indicators */}
              <div className="flex items-center space-x-4">
                {/* WebSocket Connection Status */}
                <ConnectionStatus />
              </div>
            </div>

            {/* Theme Selector, Auth Status and Version Info */}
            <div className="hidden md:flex items-center space-x-4">
              <CompactThemeSelector />
              <ConditionalRender requireAuth={true}>
                <AuthStatus />
              </ConditionalRender>
              <div className="text-sm text-slate-600">
                <span>v{systemStatus?.version || '1.0.0'}</span>
                <span className="mx-2">•</span>
                <span>Port {systemStatus?.port || '8080'}</span>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-theme-surface border-b border-theme-border-default sticky top-0 z-40 shadow-theme-sm theme-transition">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between py-2">
            {/* Main Navigation Items */}
            <div className="flex space-x-1">
              {navigationItems.mainBar.map((item) => (
                <button
                  key={item.id}
                  onClick={() => setCurrentPage(item.id as Page)}
                  className={`flex items-center space-x-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                    currentPage === item.id
                      ? 'bg-theme-primary text-theme-text-inverse shadow-theme-md'
                      : 'text-theme-text-secondary hover:text-theme-text-primary hover:bg-theme-surface-alt'
                  }`}
                >
                  <span className="text-base">{item.icon}</span>
                  <span>{item.label}</span>
                </button>
              ))}
            </div>

            {/* Hamburger Menu */}
            <div className="relative">
              <button
                onClick={() => setHamburgerOpen(!hamburgerOpen)}
                className="flex items-center space-x-2 px-4 py-2 rounded-lg text-sm font-medium text-theme-text-secondary hover:text-theme-text-primary hover:bg-theme-surface-alt transition-all duration-200"
              >
                <span className="text-base">☰</span>
                <span>More</span>
              </button>

              {/* Hamburger Dropdown */}
              {hamburgerOpen && (
                <div className="absolute right-0 mt-2 w-64 bg-theme-surface border border-theme-border-default rounded-lg shadow-theme-lg z-50">
                  <div className="py-2">
                    {navigationItems.hamburger.map((item) => (
                      <button
                        key={item.id}
                        onClick={() => {
                          setCurrentPage(item.id as Page);
                          setHamburgerOpen(false);
                        }}
                        className={`w-full flex items-center space-x-3 px-4 py-2 text-left text-sm font-medium transition-all duration-200 ${
                          currentPage === item.id
                            ? 'bg-theme-primary text-theme-text-inverse'
                            : 'text-theme-text-secondary hover:text-theme-text-primary hover:bg-theme-surface-alt'
                        }`}
                      >
                        <span className="text-base">{item.icon}</span>
                        <span>{item.label}</span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {isAuthenticated ? (
          <div className="bg-theme-surface rounded-xl shadow-theme-lg border border-theme-border-default overflow-hidden theme-transition">
            <div className="p-6 lg:p-8">
              <ErrorBoundary>
                {renderPage()}
              </ErrorBoundary>
            </div>
          </div>
        ) : (
          <ProtectedRoute showLoginPrompt={true}>
            <div></div>
          </ProtectedRoute>
        )}
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
    </AnnouncementsProvider>
  )
}

// Main App component with providers
function App() {
  return (
    <ThemeProvider defaultThemeName="light">
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App