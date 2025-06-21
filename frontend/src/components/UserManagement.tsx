import React, { useState, useEffect } from 'react';
import { useAuth, ConditionalRender } from '../contexts/AuthContext';

interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  active: boolean;
  createdAt: string;
  lastLogin?: string;
}

interface UserFilters {
  role?: string;
  active?: boolean;
  search?: string;
}

const UserManagement: React.FC = () => {
  const { hasPermission } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<UserFilters>({});
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [showUserModal, setShowUserModal] = useState<boolean>(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  useEffect(() => {
    fetchUsers();
  }, [filters]);

  const fetchUsers = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch('/api/users');
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const data = await response.json();
      
      if (data.success && Array.isArray(data.data)) {
        setUsers(data.data);
      } else {
        throw new Error(data.error || 'Invalid response format');
      }
    } catch (error) {
      console.error('Error fetching users:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch users');
      setUsers([]);
    } finally {
      setLoading(false);
    }
  };

  const handleUserAction = async (userId: number, action: 'activate' | 'deactivate' | 'delete', newRole?: string) => {
    setActionLoading(`${action}-${userId}`);
    
    try {
      let endpoint = `/api/users/${userId}`;
      let method = 'PATCH';
      let body: any = {};

      switch (action) {
        case 'activate':
          body = { active: true };
          break;
        case 'deactivate':
          body = { active: false };
          break;
        case 'delete':
          method = 'DELETE';
          body = undefined;
          break;
      }

      if (newRole) {
        body.role = newRole;
      }

      const response = await fetch(endpoint, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
        body: body ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        throw new Error(`Failed to ${action} user`);
      }

      // Refresh the user list
      await fetchUsers();
      setShowUserModal(false);
      setSelectedUser(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} user`);
    } finally {
      setActionLoading(null);
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

  const handleSaveUser = async () => {
    if (!selectedUser) return;
    
    try {
      setActionLoading('saving-user');
      const url = selectedUser.id 
        ? `/api/users/${selectedUser.id}`
        : '/api/users';
      
      const method = selectedUser.id ? 'PUT' : 'POST';
      
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
        body: JSON.stringify(selectedUser)
      });
      
      if (!response.ok) {
        throw new Error('Failed to save user');
      }
      
      await fetchUsers();
      setShowUserModal(false);
      setSelectedUser(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save user');
    } finally {
      setActionLoading(null);
    }
  };

  const UserEditModal = () => (
    <div className={`fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 ${!showUserModal ? 'hidden' : ''}`}>
      <div className="bg-white rounded-lg w-full max-w-md p-6">
        <h2 className="text-xl font-bold mb-4">
          {selectedUser?.id ? 'Edit User' : 'Add New User'}
        </h2>
        
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Username</label>
            <input
              type="text"
              value={selectedUser?.username || ''}
              onChange={(e) => selectedUser && setSelectedUser({...selectedUser, username: e.target.value})}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg"
              disabled={!selectedUser}
            />
          </div>
          
          <div className="flex justify-end space-x-3 pt-4">
            <button
              onClick={() => setShowUserModal(false)}
              className="px-4 py-2 border border-slate-300 rounded-lg"
              disabled={!!actionLoading}
            >
              Cancel
            </button>
            <button
              onClick={handleSaveUser}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg disabled:opacity-50"
              disabled={!selectedUser?.username || !!actionLoading}
            >
              {actionLoading === 'saving-user' ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );

  // Check if user has permission to access user management
  if (!hasPermission('user:manage')) {
    return (
      <div className="text-center py-12">
        <div className="w-16 h-16 bg-yellow-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <span className="text-yellow-600 text-2xl">⚠️</span>
        </div>
        <h2 className="text-xl font-bold text-slate-900 mb-2">Access Denied</h2>
        <p className="text-slate-600">You don't have permission to manage users.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">User Management</h1>
          <p className="text-slate-600 mt-1">Manage system users and their permissions</p>
        </div>
        
        <ConditionalRender permission="user:manage">
          <button
            onClick={() => {
              setSelectedUser(null);
              setShowUserModal(true);
            }}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
          >
            Add User
          </button>
        </ConditionalRender>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg border border-slate-200 p-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* Search */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Search</label>
            <input
              type="text"
              placeholder="Username or email..."
              value={filters.search || ''}
              onChange={(e) => setFilters(prev => ({ ...prev, search: e.target.value }))}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {/* Role Filter */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Role</label>
            <select
              value={filters.role || ''}
              onChange={(e) => setFilters(prev => ({ ...prev, role: e.target.value || undefined }))}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">All Roles</option>
              <option value="super_admin">Super Admin</option>
              <option value="admin">Admin</option>
              <option value="operator">Operator</option>
              <option value="viewer">Viewer</option>
              <option value="user">User</option>
            </select>
          </div>

          {/* Status Filter */}
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Status</label>
            <select
              value={filters.active === undefined ? '' : filters.active.toString()}
              onChange={(e) => setFilters(prev => ({ 
                ...prev, 
                active: e.target.value === '' ? undefined : e.target.value === 'true' 
              }))}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">All Users</option>
              <option value="true">Active</option>
              <option value="false">Inactive</option>
            </select>
          </div>

          {/* Clear Filters */}
          <div className="flex items-end">
            <button
              onClick={() => setFilters({})}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-600 hover:bg-slate-50 transition-colors"
            >
              Clear Filters
            </button>
          </div>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center">
            <span className="text-red-600 mr-2">⚠️</span>
            <span className="text-red-700">{error}</span>
            <button
              onClick={() => setError(null)}
              className="ml-auto text-red-600 hover:text-red-800"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {/* Users Table */}
      <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-slate-600">Loading users...</p>
          </div>
        ) : users.length === 0 ? (
          <div className="p-8 text-center">
            <div className="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <span className="text-slate-400 text-2xl">👥</span>
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">No Users Found</h3>
            <p className="text-slate-600">No users match your current filters.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-200">
              <thead className="bg-slate-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                    User
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                    Role
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                    Last Login
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-slate-200">
                {users.map((user) => (
                  <tr key={user.id} className="hover:bg-slate-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className="w-10 h-10 bg-gradient-to-r from-blue-600 to-blue-700 rounded-full flex items-center justify-center">
                          <span className="text-white font-bold text-sm">
                            {user.username.charAt(0).toUpperCase()}
                          </span>
                        </div>
                        <div className="ml-4">
                          <div className="text-sm font-medium text-slate-900">{user.username}</div>
                          <div className="text-sm text-slate-500">{user.email}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full border ${getRoleColor(user.role)}`}>
                        {user.role.replace('_', ' ').toUpperCase()}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full border ${
                        user.active 
                          ? 'bg-green-100 text-green-700 border-green-200' 
                          : 'bg-red-100 text-red-700 border-red-200'
                      }`}>
                        {user.active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500">
                      {formatDate(user.createdAt)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500">
                      {user.lastLogin ? formatDate(user.lastLogin) : 'Never'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button
                        onClick={() => {
                          setSelectedUser(user);
                          setShowUserModal(true);
                        }}
                        className="text-blue-600 hover:text-blue-900 mr-3"
                        disabled={actionLoading !== null}
                      >
                        Edit
                      </button>
                      {user.active ? (
                        <button
                          onClick={() => handleUserAction(user.id, 'deactivate')}
                          className="text-orange-600 hover:text-orange-900 mr-3"
                          disabled={actionLoading === `deactivate-${user.id}`}
                        >
                          {actionLoading === `deactivate-${user.id}` ? '...' : 'Deactivate'}
                        </button>
                      ) : (
                        <button
                          onClick={() => handleUserAction(user.id, 'activate')}
                          className="text-green-600 hover:text-green-900 mr-3"
                          disabled={actionLoading === `activate-${user.id}`}
                        >
                          {actionLoading === `activate-${user.id}` ? '...' : 'Activate'}
                        </button>
                      )}
                      <ConditionalRender role={['super_admin']}>
                        <button
                          onClick={() => {
                            if (window.confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
                              handleUserAction(user.id, 'delete');
                            }
                          }}
                          className="text-red-600 hover:text-red-900"
                          disabled={actionLoading === `delete-${user.id}`}
                        >
                          {actionLoading === `delete-${user.id}` ? '...' : 'Delete'}
                        </button>
                      </ConditionalRender>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* User Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white rounded-lg border border-slate-200 p-6">
          <div className="text-2xl font-bold text-slate-900">{users.length}</div>
          <div className="text-sm text-slate-600">Total Users</div>
        </div>
        <div className="bg-white rounded-lg border border-slate-200 p-6">
          <div className="text-2xl font-bold text-green-600">{users.filter(u => u.active).length}</div>
          <div className="text-sm text-slate-600">Active Users</div>
        </div>
        <div className="bg-white rounded-lg border border-slate-200 p-6">
          <div className="text-2xl font-bold text-blue-600">{users.filter(u => u.role === 'admin' || u.role === 'super_admin').length}</div>
          <div className="text-sm text-slate-600">Administrators</div>
        </div>
        <div className="bg-white rounded-lg border border-slate-200 p-6">
          <div className="text-2xl font-bold text-orange-600">{users.filter(u => u.lastLogin && new Date(u.lastLogin) > new Date(Date.now() - 24 * 60 * 60 * 1000)).length}</div>
          <div className="text-sm text-slate-600">Active Today</div>
        </div>
      </div>

      {UserEditModal()}
    </div>
  );
};

export default UserManagement;