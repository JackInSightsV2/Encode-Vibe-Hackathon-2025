import React, { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';

interface APIKey {
  id: string;
  name: string;
  description?: string;
  active: boolean;
  lastUsed?: string;
  createdAt: string;
  expiresAt?: string;
  usageCount: number;
  permissions: string[];
  isExpired: boolean;
  daysUntilExpiry?: number;
}

interface CreateAPIKeyRequest {
  name: string;
  description?: string;
  expiresAt?: string;
  permissions: string[];
}

const APIKeyManagement: React.FC = () => {
  const { hasPermission } = useAuth();
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState<CreateAPIKeyRequest>({
    name: '',
    description: '',
    permissions: []
  });
  const [createdKey, setCreatedKey] = useState<{ key: APIKey; plainKey: string } | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  useEffect(() => {
    fetchAPIKeys();
  }, []);

  const fetchAPIKeys = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/api-keys', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch API keys');
      }

      const data = await response.json();
      setApiKeys(data.data?.keys || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch API keys');
    } finally {
      setLoading(false);
    }
  };

  const createAPIKey = async () => {
    if (!createForm.name.trim()) return;

    setActionLoading('create');
    try {
      const response = await fetch('/api/api-keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
        body: JSON.stringify(createForm),
      });

      if (!response.ok) {
        throw new Error('Failed to create API key');
      }

      const data = await response.json();
      setCreatedKey({
        key: data.data.api_key,
        plainKey: data.data.plain_key
      });
      
      // Reset form
      setCreateForm({
        name: '',
        description: '',
        permissions: []
      });
      
      // Refresh list
      await fetchAPIKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create API key');
    } finally {
      setActionLoading(null);
    }
  };

  const toggleAPIKey = async (keyId: string, active: boolean) => {
    setActionLoading(`toggle-${keyId}`);
    try {
      const response = await fetch(`/api/api-keys/${keyId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
        body: JSON.stringify({ active }),
      });

      if (!response.ok) {
        throw new Error(`Failed to ${active ? 'activate' : 'deactivate'} API key`);
      }

      await fetchAPIKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update API key');
    } finally {
      setActionLoading(null);
    }
  };

  const deleteAPIKey = async (keyId: string) => {
    if (!window.confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return;
    }

    setActionLoading(`delete-${keyId}`);
    try {
      const response = await fetch(`/api/api-keys/${keyId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to delete API key');
      }

      await fetchAPIKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete API key');
    } finally {
      setActionLoading(null);
    }
  };

  const rotateAPIKey = async (keyId: string) => {
    if (!window.confirm('This will create a new API key and deactivate the old one. Continue?')) {
      return;
    }

    setActionLoading(`rotate-${keyId}`);
    try {
      const response = await fetch(`/api/api-keys/${keyId}/rotate`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('authToken')}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to rotate API key');
      }

      const data = await response.json();
      setCreatedKey({
        key: data.data.api_key,
        plainKey: data.data.plain_key
      });
      
      await fetchAPIKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rotate API key');
    } finally {
      setActionLoading(null);
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

  const getStatusColor = (key: APIKey) => {
    if (!key.active) return 'bg-gray-100 text-gray-700 border-gray-200';
    if (key.isExpired) return 'bg-red-100 text-red-700 border-red-200';
    if (key.daysUntilExpiry && key.daysUntilExpiry <= 7) return 'bg-yellow-100 text-yellow-700 border-yellow-200';
    return 'bg-green-100 text-green-700 border-green-200';
  };

  const getStatusText = (key: APIKey) => {
    if (!key.active) return 'Inactive';
    if (key.isExpired) return 'Expired';
    if (key.daysUntilExpiry && key.daysUntilExpiry <= 7) return `Expires in ${key.daysUntilExpiry} days`;
    return 'Active';
  };

  const availablePermissions = [
    'config:read',
    'config:write',
    'logs:read',
    'metrics:read',
    'api_keys:read',
    'api_keys:manage'
  ];

  // Check if user has permission to manage API keys
  if (!hasPermission('api_keys:read')) {
    return (
      <div className="text-center py-12">
        <div className="w-16 h-16 bg-yellow-100 rounded-full flex items-center justify-center mx-auto mb-4">
          <span className="text-yellow-600 text-2xl">🔑</span>
        </div>
        <h2 className="text-xl font-bold text-slate-900 mb-2">Access Denied</h2>
        <p className="text-slate-600">You don't have permission to view API keys.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">API Key Management</h1>
          <p className="text-slate-600 mt-1">Create and manage your API keys for programmatic access</p>
        </div>
        
        {hasPermission('api_keys:manage') && (
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
          >
            Create API Key
          </button>
        )}
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

      {/* Created Key Modal */}
      {createdKey && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <div className="text-center mb-6">
              <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <span className="text-green-600 text-2xl">🔑</span>
              </div>
              <h3 className="text-xl font-bold text-slate-900">API Key Created</h3>
              <p className="text-slate-600 mt-1">Save this key securely - it won't be shown again</p>
            </div>

            <div className="bg-slate-100 p-4 rounded-lg mb-4">
              <div className="text-sm font-medium text-slate-700 mb-2">API Key:</div>
              <div className="font-mono text-sm bg-white p-2 rounded border break-all">
                {createdKey.plainKey}
              </div>
            </div>

            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3 mb-4">
              <div className="flex items-start">
                <span className="text-yellow-600 mr-2 mt-0.5">⚠️</span>
                <div className="text-yellow-700 text-sm">
                  This is the only time the full API key will be displayed. Make sure to copy and store it securely.
                </div>
              </div>
            </div>

            <button
              onClick={() => setCreatedKey(null)}
              className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg font-medium transition-colors"
            >
              I've Saved the Key
            </button>
          </div>
        </div>
      )}

      {/* Create Key Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h3 className="text-xl font-bold text-slate-900 mb-4">Create New API Key</h3>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Name *</label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, name: e.target.value }))}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., Production API Key"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Description</label>
                <textarea
                  value={createForm.description}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, description: e.target.value }))}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  rows={3}
                  placeholder="Optional description..."
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-2">Permissions</label>
                <div className="space-y-2 max-h-32 overflow-y-auto">
                  {availablePermissions.map(permission => (
                    <label key={permission} className="flex items-center">
                      <input
                        type="checkbox"
                        checked={createForm.permissions.includes(permission)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setCreateForm(prev => ({
                              ...prev,
                              permissions: [...prev.permissions, permission]
                            }));
                          } else {
                            setCreateForm(prev => ({
                              ...prev,
                              permissions: prev.permissions.filter(p => p !== permission)
                            }));
                          }
                        }}
                        className="mr-2"
                      />
                      <span className="text-sm text-slate-700">{permission}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div className="flex space-x-3 mt-6">
              <button
                onClick={() => setShowCreateModal(false)}
                className="flex-1 px-4 py-2 border border-slate-300 rounded-lg text-slate-700 hover:bg-slate-50 transition-colors"
                disabled={actionLoading === 'create'}
              >
                Cancel
              </button>
              <button
                onClick={createAPIKey}
                disabled={!createForm.name.trim() || actionLoading === 'create'}
                className={`flex-1 px-4 py-2 rounded-lg font-medium transition-colors ${
                  createForm.name.trim() && actionLoading !== 'create'
                    ? 'bg-blue-600 hover:bg-blue-700 text-white'
                    : 'bg-slate-300 text-slate-500 cursor-not-allowed'
                }`}
              >
                {actionLoading === 'create' ? 'Creating...' : 'Create Key'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* API Keys List */}
      <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-slate-600">Loading API keys...</p>
          </div>
        ) : apiKeys.length === 0 ? (
          <div className="p-8 text-center">
            <div className="w-16 h-16 bg-slate-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <span className="text-slate-400 text-2xl">🔑</span>
            </div>
            <h3 className="text-lg font-medium text-slate-900 mb-2">No API Keys</h3>
            <p className="text-slate-600 mb-4">You haven't created any API keys yet.</p>
            {hasPermission('api_keys:manage') && (
              <button
                onClick={() => setShowCreateModal(true)}
                className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
              >
                Create Your First API Key
              </button>
            )}
          </div>
        ) : (
          <div className="divide-y divide-slate-200">
            {apiKeys.map((key) => (
              <div key={key.id} className="p-6">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-3 mb-2">
                      <h3 className="text-lg font-medium text-slate-900">{key.name}</h3>
                      <span className={`px-2 py-1 text-xs font-medium rounded-full border ${getStatusColor(key)}`}>
                        {getStatusText(key)}
                      </span>
                    </div>
                    
                    {key.description && (
                      <p className="text-slate-600 mb-3">{key.description}</p>
                    )}

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                      <div>
                        <span className="text-slate-500">Created:</span>
                        <div className="font-medium">{formatDate(key.createdAt)}</div>
                      </div>
                      <div>
                        <span className="text-slate-500">Last Used:</span>
                        <div className="font-medium">{key.lastUsed ? formatDate(key.lastUsed) : 'Never'}</div>
                      </div>
                      <div>
                        <span className="text-slate-500">Usage Count:</span>
                        <div className="font-medium">{key.usageCount.toLocaleString()}</div>
                      </div>
                      <div>
                        <span className="text-slate-500">Expires:</span>
                        <div className="font-medium">{key.expiresAt ? formatDate(key.expiresAt) : 'Never'}</div>
                      </div>
                    </div>

                    {key.permissions.length > 0 && (
                      <div className="mt-3">
                        <span className="text-slate-500 text-sm">Permissions:</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {key.permissions.map(permission => (
                            <span key={permission} className="px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded">
                              {permission}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>

                  {hasPermission('api_keys:manage') && (
                    <div className="flex flex-col space-y-2 ml-4">
                      <button
                        onClick={() => toggleAPIKey(key.id, !key.active)}
                        disabled={actionLoading === `toggle-${key.id}`}
                        className={`px-3 py-1 text-xs rounded font-medium transition-colors ${
                          key.active
                            ? 'bg-orange-100 text-orange-700 hover:bg-orange-200'
                            : 'bg-green-100 text-green-700 hover:bg-green-200'
                        }`}
                      >
                        {actionLoading === `toggle-${key.id}` ? '...' : (key.active ? 'Deactivate' : 'Activate')}
                      </button>
                      
                      <button
                        onClick={() => rotateAPIKey(key.id)}
                        disabled={actionLoading === `rotate-${key.id}`}
                        className="px-3 py-1 text-xs bg-blue-100 text-blue-700 hover:bg-blue-200 rounded font-medium transition-colors"
                      >
                        {actionLoading === `rotate-${key.id}` ? '...' : 'Rotate'}
                      </button>
                      
                      <button
                        onClick={() => deleteAPIKey(key.id)}
                        disabled={actionLoading === `delete-${key.id}`}
                        className="px-3 py-1 text-xs bg-red-100 text-red-700 hover:bg-red-200 rounded font-medium transition-colors"
                      >
                        {actionLoading === `delete-${key.id}` ? '...' : 'Delete'}
                      </button>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Statistics */}
      {apiKeys.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <div className="text-2xl font-bold text-slate-900">{apiKeys.length}</div>
            <div className="text-sm text-slate-600">Total Keys</div>
          </div>
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <div className="text-2xl font-bold text-green-600">{apiKeys.filter(k => k.active && !k.isExpired).length}</div>
            <div className="text-sm text-slate-600">Active Keys</div>
          </div>
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <div className="text-2xl font-bold text-red-600">{apiKeys.filter(k => k.isExpired).length}</div>
            <div className="text-sm text-slate-600">Expired Keys</div>
          </div>
          <div className="bg-white rounded-lg border border-slate-200 p-6">
            <div className="text-2xl font-bold text-blue-600">{apiKeys.reduce((sum, k) => sum + k.usageCount, 0).toLocaleString()}</div>
            <div className="text-sm text-slate-600">Total Requests</div>
          </div>
        </div>
      )}
    </div>
  );
};

export default APIKeyManagement;