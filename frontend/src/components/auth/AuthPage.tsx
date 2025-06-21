import React, { useState } from 'react';
import LoginForm from './LoginForm';
import RegisterForm from './RegisterForm';

interface AuthPageProps {
  onAuthSuccess?: () => void;
}

const AuthPage: React.FC<AuthPageProps> = ({ onAuthSuccess }) => {
  const [activeTab, setActiveTab] = useState<'login' | 'register'>('login');

  const handleAuthSuccess = () => {
    onAuthSuccess?.();
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Tab Navigation */}
        <div className="bg-white rounded-t-lg shadow-lg border border-slate-200 border-b-0">
          <div className="flex">
            <button
              onClick={() => setActiveTab('login')}
              className={`flex-1 py-3 px-4 text-sm font-medium rounded-tl-lg transition-colors ${
                activeTab === 'login'
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-50 text-slate-600 hover:text-slate-900 hover:bg-slate-100'
              }`}
            >
              Sign In
            </button>
            <button
              onClick={() => setActiveTab('register')}
              className={`flex-1 py-3 px-4 text-sm font-medium rounded-tr-lg transition-colors ${
                activeTab === 'register'
                  ? 'bg-green-600 text-white'
                  : 'bg-slate-50 text-slate-600 hover:text-slate-900 hover:bg-slate-100'
              }`}
            >
              Create Account
            </button>
          </div>
        </div>

        {/* Form Content */}
        <div className="bg-white rounded-b-lg shadow-lg border border-slate-200 border-t-0">
          {activeTab === 'login' ? (
            <div className="p-6">
              <LoginForm
                onSuccess={handleAuthSuccess}
                onShowRegister={() => setActiveTab('register')}
              />
            </div>
          ) : (
            <div className="p-6">
              <RegisterForm
                onSuccess={handleAuthSuccess}
                onShowLogin={() => setActiveTab('login')}
              />
            </div>
          )}
        </div>

        {/* Footer Information */}
        <div className="mt-6 text-center">
          <p className="text-xs text-slate-500">
            By continuing, you agree to QT-1 Middleware's terms of service
          </p>
          <div className="flex items-center justify-center space-x-4 mt-3 text-xs text-slate-400">
            <span>🔒 Secure Authentication</span>
            <span>•</span>
            <span>🛡️ Role-Based Access</span>
            <span>•</span>
            <span>📊 Activity Monitoring</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AuthPage;