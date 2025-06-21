import React, { useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';

interface UserProfileProps {
  isDropdownOpen?: boolean;
  onClose?: () => void;
}

const UserProfile: React.FC<UserProfileProps> = ({ isDropdownOpen = false, onClose }) => {
  const { user, logout, hasRole } = useAuth();
  const [showConfirmLogout, setShowConfirmLogout] = useState(false);

  if (!user) return null;

  const handleLogout = () => {
    if (showConfirmLogout) {
      logout();
      onClose?.();
    } else {
      setShowConfirmLogout(true);
      setTimeout(() => setShowConfirmLogout(false), 3000); // Auto-hide after 3 seconds
    }
  };

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'super_admin':
        return 'bg-purple-100 text-purple-700 border-purple-200';
      case 'admin':
        return 'bg-red-100 text-red-700 border-red-200';
      case 'operator':
        return 'bg-blue-100 text-blue-700 border-blue-200';
      case 'viewer':
        return 'bg-green-100 text-green-700 border-green-200';
      case 'user':
        return 'bg-gray-100 text-gray-700 border-gray-200';
      default:
        return 'bg-gray-100 text-gray-700 border-gray-200';
    }
  };

  const getRoleLabel = (role: string) => {
    switch (role) {
      case 'super_admin':
        return 'Super Admin';
      case 'admin':
        return 'Administrator';
      case 'operator':
        return 'Operator';
      case 'viewer':
        return 'Viewer';
      case 'user':
        return 'User';
      default:
        return role;
    }
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return 'Unknown';
    }
  };

  if (!isDropdownOpen) {
    // Compact header display
    return (
      <div className="flex items-center space-x-3">
        <div className="w-8 h-8 bg-gradient-to-r from-blue-600 to-blue-700 rounded-full flex items-center justify-center">
          <span className="text-white font-bold text-sm">
            {user.username.charAt(0).toUpperCase()}
          </span>
        </div>
        <div className="hidden md:block">
          <div className="text-sm font-medium text-slate-900">{user.username}</div>
          <div className="text-xs text-slate-500">{getRoleLabel(user.role)}</div>
        </div>
      </div>
    );
  }

  // Full dropdown display
  return (
    <div className="w-80 bg-white rounded-lg shadow-lg border border-slate-200 overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-blue-700 p-4 text-white">
        <div className="flex items-center space-x-3">
          <div className="w-12 h-12 bg-white bg-opacity-20 rounded-full flex items-center justify-center">
            <span className="font-bold text-lg">
              {user.username.charAt(0).toUpperCase()}
            </span>
          </div>
          <div>
            <h3 className="font-semibold text-lg">{user.username}</h3>
            <p className="text-blue-100 text-sm">{user.email}</p>
          </div>
        </div>
      </div>

      {/* User Details */}
      <div className="p-4 space-y-4">
        {/* Role Badge */}
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700">Role</span>
          <span className={`px-2 py-1 rounded-full text-xs font-medium border ${getRoleColor(user.role)}`}>
            {getRoleLabel(user.role)}
          </span>
        </div>

        {/* Account Status */}
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700">Status</span>
          <span className={`px-2 py-1 rounded-full text-xs font-medium border ${
            user.active 
              ? 'bg-green-100 text-green-700 border-green-200' 
              : 'bg-red-100 text-red-700 border-red-200'
          }`}>
            {user.active ? 'Active' : 'Inactive'}
          </span>
        </div>

        {/* Member Since */}
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700">Member Since</span>
          <span className="text-sm text-slate-500">
            {formatDate(user.createdAt)}
          </span>
        </div>

        {/* Last Login */}
        {user.lastLogin && (
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-slate-700">Last Login</span>
            <span className="text-sm text-slate-500">
              {formatDate(user.lastLogin)}
            </span>
          </div>
        )}

        {/* Permissions Summary */}
        <div className="pt-3 border-t border-slate-200">
          <div className="text-sm font-medium text-slate-700 mb-2">Permissions</div>
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className={`flex items-center space-x-1 ${hasRole(['admin', 'super_admin']) ? 'text-green-600' : 'text-slate-400'}`}>
              <span>{hasRole(['admin', 'super_admin']) ? '✅' : '❌'}</span>
              <span>User Management</span>
            </div>
            <div className={`flex items-center space-x-1 ${hasRole(['admin', 'super_admin', 'operator']) ? 'text-green-600' : 'text-slate-400'}`}>
              <span>{hasRole(['admin', 'super_admin', 'operator']) ? '✅' : '❌'}</span>
              <span>Configuration</span>
            </div>
            <div className={`flex items-center space-x-1 ${hasRole(['admin', 'super_admin', 'operator']) ? 'text-green-600' : 'text-slate-400'}`}>
              <span>{hasRole(['admin', 'super_admin', 'operator']) ? '✅' : '❌'}</span>
              <span>Kill Switch</span>
            </div>
            <div className="flex items-center space-x-1 text-green-600">
              <span>✅</span>
              <span>View Logs</span>
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="p-4 bg-slate-50 border-t border-slate-200">
        <div className="space-y-2">
          {/* API Keys Link - if user has permission */}
          {hasRole(['admin', 'super_admin', 'operator', 'user']) && (
            <button className="w-full text-left px-3 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg transition-colors">
              🔑 Manage API Keys
            </button>
          )}
          
          {/* User Settings */}
          <button className="w-full text-left px-3 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg transition-colors">
            ⚙️ Account Settings
          </button>

          {/* Logout Button */}
          <button
            onClick={handleLogout}
            className={`w-full text-left px-3 py-2 text-sm rounded-lg transition-colors ${
              showConfirmLogout
                ? 'bg-red-100 text-red-700 hover:bg-red-200'
                : 'text-slate-700 hover:bg-slate-100'
            }`}
          >
            {showConfirmLogout ? '⚠️ Click again to confirm logout' : '🚪 Sign Out'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default UserProfile;