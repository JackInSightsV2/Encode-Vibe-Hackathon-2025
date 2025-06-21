import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

// API Configuration
const API_BASE_URL = 'http://localhost:8080/api';

// Types
interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  active: boolean;
  createdAt: string;
  lastLogin?: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  loading: boolean;
}

interface LoginCredentials {
  username: string;
  password: string;
}

interface RegisterData {
  username: string;
  email: string;
  password: string;
  role?: string;
}

interface AuthContextType extends AuthState {
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  register: (data: RegisterData) => Promise<void>;
  refreshToken: () => Promise<void>;
  hasPermission: (permission: string) => boolean;
  hasRole: (role: string | string[]) => boolean;
  updateUser: (user: Partial<User>) => void;
}

// Permission mapping based on roles
const ROLE_PERMISSIONS: Record<string, string[]> = {
  super_admin: ['*'], // All permissions
  admin: [
    'user:manage',
    'config:read',
    'config:write', 
    'logs:read',
    'logs:write',
    'killswitch:manage',
    'metrics:read',
    'security:read',
    'security:write',
    'api_keys:manage'
  ],
  operator: [
    'config:read',
    'config:write',
    'logs:read',
    'killswitch:manage',
    'metrics:read',
    'security:read',
    'security:write',
    'api_keys:read'
  ],
  viewer: [
    'config:read',
    'logs:read',
    'metrics:read',
    'security:read'
  ],
  user: [
    'config:read',
    'api_keys:read'
  ]
};

// Create context
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Auth provider component
interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [authState, setAuthState] = useState<AuthState>({
    user: null,
    token: null,
    isAuthenticated: false,
    loading: true
  });

  // Initialize auth state from localStorage on mount
  useEffect(() => {
    initializeAuth();
  }, []);

  // Set up token refresh interval
  useEffect(() => {
    if (authState.isAuthenticated && authState.token) {
      const interval = setInterval(() => {
        refreshToken().catch(console.error);
      }, 15 * 60 * 1000); // Refresh every 15 minutes

      return () => clearInterval(interval);
    }
  }, [authState.isAuthenticated, authState.token]);

  const initializeAuth = async () => {
    try {
      const token = localStorage.getItem('authToken');
      const userStr = localStorage.getItem('authUser');

      if (token && userStr) {
        const user = JSON.parse(userStr);
        
        // Validate token with server
        const isValid = await validateToken(token);
        if (isValid) {
          setAuthState({
            user,
            token,
            isAuthenticated: true,
            loading: false
          });
        } else {
          // Token is invalid, clear storage
          clearAuthStorage();
          setAuthState({
            user: null,
            token: null,
            isAuthenticated: false,
            loading: false
          });
        }
      } else {
        setAuthState({
          user: null,
          token: null,
          isAuthenticated: false,
          loading: false
        });
      }
    } catch (error) {
      console.error('Failed to initialize auth:', error);
      clearAuthStorage();
      setAuthState({
        user: null,
        token: null,
        isAuthenticated: false,
        loading: false
      });
    }
  };

  const login = async (credentials: LoginCredentials): Promise<void> => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error?.message || 'Login failed');
      }

      const responseData = await response.json();
      const { user, token } = responseData.data;

      // Store in localStorage
      localStorage.setItem('authToken', token);
      localStorage.setItem('authUser', JSON.stringify(user));

      setAuthState({
        user,
        token,
        isAuthenticated: true,
        loading: false
      });
    } catch (error) {
      console.error('Login error:', error);
      throw error;
    }
  };

  const register = async (data: RegisterData): Promise<void> => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error?.message || 'Registration failed');
      }

      // Registration successful, but user needs to login
      // Some systems auto-login after registration, but we'll require explicit login
    } catch (error) {
      console.error('Registration error:', error);
      throw error;
    }
  };

  const logout = (): void => {
    // Call logout endpoint if needed
    if (authState.token) {
      fetch(`${API_BASE_URL}/auth/logout`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authState.token}`,
        },
      }).catch(console.error); // Don't block logout on API failure
    }

    clearAuthStorage();
    setAuthState({
      user: null,
      token: null,
      isAuthenticated: false,
      loading: false
    });
  };

  const refreshToken = async (): Promise<void> => {
    try {
      if (!authState.token) return;

      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${authState.token}`,
        },
      });

      if (response.ok) {
        const responseData = await response.json();
        const { token, user } = responseData.data;

        localStorage.setItem('authToken', token);
        if (user) {
          localStorage.setItem('authUser', JSON.stringify(user));
        }

        setAuthState(prev => ({
          ...prev,
          token,
          user: user || prev.user
        }));
      } else {
        // Refresh failed, logout user
        logout();
      }
    } catch (error) {
      console.error('Token refresh failed:', error);
      logout();
    }
  };

  const validateToken = async (token: string): Promise<boolean> => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/validate`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
      return response.ok;
    } catch {
      return false;
    }
  };

  const hasPermission = (permission: string): boolean => {
    if (!authState.user) return false;

    const userRole = authState.user.role;
    const rolePermissions = ROLE_PERMISSIONS[userRole] || [];

    // Super admin has all permissions
    if (rolePermissions.includes('*')) return true;

    // Check specific permission
    return rolePermissions.includes(permission);
  };

  const hasRole = (role: string | string[]): boolean => {
    if (!authState.user) return false;

    const userRole = authState.user.role;
    if (Array.isArray(role)) {
      return role.includes(userRole);
    }
    return userRole === role;
  };

  const updateUser = (userUpdate: Partial<User>): void => {
    if (!authState.user) return;

    const updatedUser = { ...authState.user, ...userUpdate };
    localStorage.setItem('authUser', JSON.stringify(updatedUser));
    
    setAuthState(prev => ({
      ...prev,
      user: updatedUser
    }));
  };

  const clearAuthStorage = (): void => {
    localStorage.removeItem('authToken');
    localStorage.removeItem('authUser');
  };

  const contextValue: AuthContextType = {
    ...authState,
    login,
    logout,
    register,
    refreshToken,
    hasPermission,
    hasRole,
    updateUser
  };

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use auth context
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

// Higher-order component for route protection
interface ProtectedRouteProps {
  children: ReactNode;
  permission?: string;
  role?: string | string[];
  fallback?: ReactNode;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  permission,
  role,
  fallback = <div className="text-center text-gray-500">Access denied</div>
}) => {
  const { isAuthenticated, hasPermission, hasRole, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return fallback;
  }

  if (permission && !hasPermission(permission)) {
    return fallback;
  }

  if (role && !hasRole(role)) {
    return fallback;
  }

  return <>{children}</>;
};

// Component to conditionally render based on permissions
interface ConditionalRenderProps {
  children: ReactNode;
  permission?: string;
  role?: string | string[];
  requireAuth?: boolean;
}

export const ConditionalRender: React.FC<ConditionalRenderProps> = ({
  children,
  permission,
  role,
  requireAuth = false
}) => {
  const { isAuthenticated, hasPermission, hasRole } = useAuth();

  if (requireAuth && !isAuthenticated) {
    return null;
  }

  if (permission && !hasPermission(permission)) {
    return null;
  }

  if (role && !hasRole(role)) {
    return null;
  }

  return <>{children}</>;
};

export default AuthContext;