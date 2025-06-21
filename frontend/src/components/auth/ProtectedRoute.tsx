import React, { ReactNode } from 'react';
import { useAuth, ProtectedRoute as AuthProtectedRoute } from '../../contexts/AuthContext';
import AuthPage from './AuthPage';

interface ProtectedRouteProps {
  children: ReactNode;
  permission?: string;
  role?: string | string[];
  showLoginPrompt?: boolean;
  fallbackMessage?: string;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  permission,
  role,
  showLoginPrompt = true,
  fallbackMessage
}) => {
  const { isAuthenticated, hasPermission, hasRole, loading } = useAuth();

  // Show loading spinner while auth state is being determined
  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-slate-600">Loading...</p>
        </div>
      </div>
    );
  }

  // If not authenticated, show login page or access denied
  if (!isAuthenticated) {
    if (showLoginPrompt) {
      return <AuthPage />;
    }
    
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-lg border border-slate-200 p-8 max-w-md w-full text-center">
          <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <span className="text-red-600 text-2xl">🔒</span>
          </div>
          <h2 className="text-xl font-bold text-slate-900 mb-2">Authentication Required</h2>
          <p className="text-slate-600 mb-6">
            {fallbackMessage || 'You must be signed in to access this page.'}
          </p>
          <button
            onClick={() => window.location.reload()}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
          >
            Sign In
          </button>
        </div>
      </div>
    );
  }

  // Check permissions
  if (permission && !hasPermission(permission)) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-lg border border-slate-200 p-8 max-w-md w-full text-center">
          <div className="w-16 h-16 bg-yellow-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <span className="text-yellow-600 text-2xl">⚠️</span>
          </div>
          <h2 className="text-xl font-bold text-slate-900 mb-2">Access Denied</h2>
          <p className="text-slate-600 mb-4">
            You don't have permission to access this page.
          </p>
          <div className="text-sm text-slate-500 mb-6">
            Required permission: <code className="bg-slate-100 px-2 py-1 rounded">{permission}</code>
          </div>
          <button
            onClick={() => window.history.back()}
            className="bg-slate-600 hover:bg-slate-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
          >
            Go Back
          </button>
        </div>
      </div>
    );
  }

  // Check roles
  if (role && !hasRole(role)) {
    const roleText = Array.isArray(role) ? role.join(' or ') : role;
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-lg border border-slate-200 p-8 max-w-md w-full text-center">
          <div className="w-16 h-16 bg-orange-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <span className="text-orange-600 text-2xl">👤</span>
          </div>
          <h2 className="text-xl font-bold text-slate-900 mb-2">Insufficient Role</h2>
          <p className="text-slate-600 mb-4">
            Your account role doesn't have access to this page.
          </p>
          <div className="text-sm text-slate-500 mb-6">
            Required role: <code className="bg-slate-100 px-2 py-1 rounded">{roleText}</code>
          </div>
          <button
            onClick={() => window.history.back()}
            className="bg-slate-600 hover:bg-slate-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
          >
            Go Back
          </button>
        </div>
      </div>
    );
  }

  // All checks passed, render the protected content
  return <>{children}</>;
};

// Wrapper component that uses the AuthContext ProtectedRoute
export const AuthenticatedRoute: React.FC<ProtectedRouteProps> = (props) => {
  return (
    <AuthProtectedRoute
      permission={props.permission}
      role={props.role}
      fallback={
        <ProtectedRoute {...props} showLoginPrompt={false} />
      }
    >
      {props.children}
    </AuthProtectedRoute>
  );
};

export default ProtectedRoute;